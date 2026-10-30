package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// B18：老师端任务列表 stats/status 筛选/participant_count 与详情两种计数。
// 打真实 MySQL（复用 openCLPTestDB），固定 9943xx 夹具，清理按固定 ID 硬删。
//
// 覆盖：
//   - 列表 participant_count 去重（同一学生多次打卡只算 1）；
//   - stats all/active/ended 三计数且不随分页/筛选变化；
//   - status=active|ended 在 SQL 过滤，未知值回退全部；
//   - 详情 participant_count 与 submission_count 语义区分、total_students 真实值；
//   - 空班/无任务返回 0 与空数组；跨班/跨园隔离；清理 0 残留；
//   - 参与人数列表级一次聚合（防 N+1）。

const (
	testTKSKG          uint64 = 994301
	testTKSKGOther     uint64 = 994302
	testTKSTeacher     uint64 = 994303
	testTKSClassA      uint64 = 994310
	testTKSClassB      uint64 = 994311
	testTKSClassEmpty  uint64 = 994312
	testTKSClassOther  uint64 = 994313
	testTKSStudentA1   uint64 = 994320
	testTKSStudentA2   uint64 = 994321
	testTKSStudentB    uint64 = 994322
	testTKSStudentWait uint64 = 994323 // 待入学：不计入 total_students
	testTKSStudentDel  uint64 = 994324 // 已软删：不计入 total_students
	testTKSStudentBCls uint64 = 994325 // B 班学生
	testTKSStudentOth  uint64 = 994326 // 他园学生

	testTKSParent1 uint64 = 994330
	testTKSParent2 uint64 = 994331
	testTKSParent3 uint64 = 994332
	testTKSParent4 uint64 = 994333

	testTKSTaskActive    uint64 = 994340 // 进行中：status=1，end_date 未过
	testTKSTaskEndedDate uint64 = 994341 // 已结束：status=1 但 end_date 已过
	testTKSTaskEndedStat uint64 = 994342 // 已结束：status=0
	testTKSTaskFuture    uint64 = 994343 // 未开始但未过 end_date：按契约算进行中
	testTKSTaskOtherCls  uint64 = 994344 // 同园别班任务
	testTKSTaskOtherKg   uint64 = 994345 // 他园任务
	testTKSTaskEmpty     uint64 = 994346 // 空班任务（用例内创建）
	testTKSResponse      uint64 = 994350
)

func cleanupTKSData(t *testing.T, db *gorm.DB) {
	t.Helper()
	del := func(sql string, args ...any) {
		if err := db.Exec(sql, args...).Error; err != nil {
			t.Logf("cleanup: %v", err)
		}
	}
	// 子表先删：提交与回应引用任务。用裸 SQL 硬删，避免软删行占住唯一索引。
	del("DELETE FROM task_check_ins WHERE task_id BETWEEN 994340 AND 994349")
	del("DELETE FROM task_responses WHERE task_id BETWEEN 994340 AND 994349")
	del("DELETE FROM parent_tasks WHERE id BETWEEN 994340 AND 994349")
	del("DELETE FROM students WHERE id BETWEEN 994320 AND 994329")
	del("DELETE FROM classes WHERE id BETWEEN 994310 AND 994319")
	del("DELETE FROM users WHERE id BETWEEN 994300 AND 994339")
	del("DELETE FROM kindergartens WHERE id IN (994301, 994302)")
}

type tksFixture struct {
	db      *gorm.DB
	svc     *TeacherPortalService
	tcA     *TeacherPortalContext
	tcB     *TeacherPortalContext
	tcEmpty *TeacherPortalContext

	active        domain.ParentTask
	endedByDate   domain.ParentTask
	endedByStatus domain.ParentTask
	future        domain.ParentTask
	otherClass    domain.ParentTask
	otherTenant   domain.ParentTask
}

func newTKSFixture(t *testing.T) *tksFixture {
	t.Helper()
	db := openCLPTestDB(t)
	cleanupTKSData(t, db)
	t.Cleanup(func() { cleanupTKSData(t, db) })

	validDate := time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC)
	softDeletedAt := gorm.DeletedAt{Time: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), Valid: true}
	now := time.Now()
	day := func(off int) time.Time { return teacherPortalDay(now.AddDate(0, 0, off)) }

	require.NoError(t, db.Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: testTKSKG}, Name: "测试园所-任务统计",
	}).Error)
	require.NoError(t, db.Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: testTKSKGOther}, Name: "测试园所-他园",
	}).Error)

	for _, u := range []struct {
		id    uint64
		phone string
		name  string
	}{
		{testTKSTeacher, "13900004301", "任务统计老师"},
		{testTKSParent1, "13900004302", "任务统计家长一"},
		{testTKSParent2, "13900004303", "任务统计家长二"},
		{testTKSParent3, "13900004304", "任务统计家长三"},
		{testTKSParent4, "13900004305", "任务统计家长四"},
	} {
		require.NoError(t, db.Create(&domain.User{
			BaseModel: domain.BaseModel{ID: u.id}, Nickname: u.name, Phone: u.phone, PasswordHash: "x",
		}).Error)
	}

	for _, cl := range []struct {
		id   uint64
		kg   uint64
		name string
	}{
		{testTKSClassA, testTKSKG, "任务统计一班"},
		{testTKSClassB, testTKSKG, "任务统计二班"},
		{testTKSClassEmpty, testTKSKG, "任务统计空班"},
		{testTKSClassOther, testTKSKGOther, "他园班级"},
	} {
		require.NoError(t, db.Create(&domain.Class{
			BaseModel: domain.BaseModel{ID: cl.id}, KindergartenID: cl.kg, Name: cl.name, Grade: "小班", Status: 1,
		}).Error)
	}

	mkStudent := func(id, classID, kg uint64, status int8, deleted gorm.DeletedAt) {
		require.NoError(t, db.Create(&domain.Student{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: id, DeletedAt: deleted}},
			KindergartenID:  kg, ClassID: classID, Name: "任务统计学生",
			Birthday: validDate, EnrollDate: validDate, Status: status,
		}).Error)
		// Student.Status 带 `default:1`，GORM 插入时会跳过零值，必须显式回写。
		if status != 1 {
			require.NoError(t, db.Model(&domain.Student{}).Where("id = ?", id).Update("status", status).Error)
		}
	}
	mkStudent(testTKSStudentA1, testTKSClassA, testTKSKG, 1, gorm.DeletedAt{})
	mkStudent(testTKSStudentA2, testTKSClassA, testTKSKG, 1, gorm.DeletedAt{})
	mkStudent(testTKSStudentB, testTKSClassA, testTKSKG, 1, gorm.DeletedAt{})
	mkStudent(testTKSStudentWait, testTKSClassA, testTKSKG, 0, gorm.DeletedAt{})
	mkStudent(testTKSStudentDel, testTKSClassA, testTKSKG, 1, softDeletedAt)
	mkStudent(testTKSStudentBCls, testTKSClassB, testTKSKG, 1, gorm.DeletedAt{})
	mkStudent(testTKSStudentOth, testTKSClassOther, testTKSKGOther, 1, gorm.DeletedAt{})

	mkTask := func(id, classID, kg uint64, status int8, startOff, endOff int, createdAgo time.Duration, title string) domain.ParentTask {
		task := domain.ParentTask{
			BaseModel:         domain.BaseModel{ID: id, CreatedAt: now.Add(-createdAgo)},
			KindergartenID:    kg,
			ClassID:           classID,
			CreatorID:         testTKSTeacher,
			Title:             title,
			Frequency:         domain.TaskFrequencyDaily,
			StartDate:         day(startOff),
			EndDate:           day(endOff),
			Status:            1,
			RecognitionRitual: domain.TaskRitualSticker,
			StarScope:         domain.TaskStarScopeTask,
		}
		require.NoError(t, db.Create(&task).Error)
		if status != 1 {
			// ParentTask.Status 同样带 `default:1`，零值会被跳过，必须显式回写。
			require.NoError(t, db.Model(&domain.ParentTask{}).Where("id = ?", id).Update("status", status).Error)
			task.Status = status
		}
		return task
	}

	f := &tksFixture{db: db}
	f.active = mkTask(testTKSTaskActive, testTKSClassA, testTKSKG, 1, -3, 3, 5*time.Hour, "进行中任务")
	f.endedByDate = mkTask(testTKSTaskEndedDate, testTKSClassA, testTKSKG, 1, -10, -1, 4*time.Hour, "日期已过任务")
	f.endedByStatus = mkTask(testTKSTaskEndedStat, testTKSClassA, testTKSKG, 0, -3, 3, 3*time.Hour, "已关闭任务")
	f.future = mkTask(testTKSTaskFuture, testTKSClassA, testTKSKG, 1, 2, 9, 2*time.Hour, "未开始任务")
	f.otherClass = mkTask(testTKSTaskOtherCls, testTKSClassB, testTKSKG, 1, -1, 3, time.Hour, "别班任务")
	f.otherTenant = mkTask(testTKSTaskOtherKg, testTKSClassOther, testTKSKGOther, 1, -1, 3, 30*time.Minute, "他园任务")

	mkCheckin := func(taskID, studentID, userID uint64, off int, content string) domain.TaskCheckIn {
		c := domain.TaskCheckIn{
			TaskID: taskID, StudentID: studentID, UserID: userID,
			Date: day(off), Content: content,
		}
		require.NoError(t, db.Create(&c).Error)
		return c
	}

	resp := domain.TaskResponse{
		BaseModel:      domain.BaseModel{ID: testTKSResponse},
		KindergartenID: testTKSKG, TaskID: testTKSTaskActive, TeacherID: testTKSTeacher,
		Ritual: domain.TaskRitualSticker, ResponseType: domain.TaskResponseSingle,
	}
	require.NoError(t, db.Create(&resp).Error)

	// 进行中任务：A1 跨两天各提交一次（行数 2、人数 1），B 提交一次 → 行数 3、人数 2。
	cA1Day1 := mkCheckin(testTKSTaskActive, testTKSStudentA1, testTKSParent1, -1, "A1 第一次")
	mkCheckin(testTKSTaskActive, testTKSStudentA1, testTKSParent1, 0, "A1 第二次")
	mkCheckin(testTKSTaskActive, testTKSStudentB, testTKSParent2, 0, "B 提交")
	// 已结束（日期）任务：A2 一条提交。
	mkCheckin(testTKSTaskEndedDate, testTKSStudentA2, testTKSParent3, -5, "A2 提交")
	// 别班任务的提交不得串进本班统计。
	mkCheckin(testTKSTaskOtherCls, testTKSStudentBCls, testTKSParent4, 0, "别班提交")
	// 已关闭任务与未开始任务没有提交：参与人数必须是 0。

	require.NoError(t, db.Model(&domain.TaskCheckIn{}).Where("id = ?", cA1Day1.ID).
		Update("response_id", resp.ID).Error)

	f.svc = NewTeacherPortalService(db, nil, nil, repository.NewMessageRepo(db), nil, nil, config.FaceConfig{})
	f.tcA = &TeacherPortalContext{UserID: testTKSTeacher, KindergartenID: testTKSKG, ClassID: testTKSClassA}
	f.tcB = &TeacherPortalContext{UserID: testTKSTeacher, KindergartenID: testTKSKG, ClassID: testTKSClassB}
	f.tcEmpty = &TeacherPortalContext{UserID: testTKSTeacher, KindergartenID: testTKSKG, ClassID: testTKSClassEmpty}
	return f
}

func tksList(
	t *testing.T, svc *TeacherPortalService, tc *TeacherPortalContext, status string, page, pageSize int,
) ([]map[string]any, map[string]int64, int64) {
	t.Helper()
	data, err := svc.ListTasksFiltered(context.Background(), tc, page, pageSize, status)
	require.NoError(t, err)
	list, ok := data["list"].([]map[string]any)
	require.True(t, ok, "list 必须是数组")
	stats, ok := data["stats"].(map[string]int64)
	require.True(t, ok, "stats 必须是 all/active/ended 计数")
	total, ok := data["total"].(int64)
	require.True(t, ok, "total 必须是 int64")
	return list, stats, total
}

func tksIndex(list []map[string]any) map[uint64]map[string]any {
	out := make(map[uint64]map[string]any, len(list))
	for _, item := range list {
		id, _ := item["id"].(uint64)
		out[id] = item
	}
	return out
}

func (f *tksFixture) day(off int) time.Time {
	return teacherPortalDay(time.Now().AddDate(0, 0, off))
}

// 列表参与人数去重 + 三计数；跨班/跨园任务不得混入。
func TestTeacherTaskListParticipantCount(t *testing.T) {
	f := newTKSFixture(t)

	// 夹具前提：同一学生（A1）在任务上有两条不同日期的打卡，去重前会数成 2。
	var a1Rows int64
	require.NoError(t, f.db.Model(&domain.TaskCheckIn{}).
		Where("task_id = ? AND student_id = ?", f.active.ID, testTKSStudentA1).Count(&a1Rows).Error)
	require.EqualValues(t, 2, a1Rows)

	list, stats, total := tksList(t, f.svc, f.tcA, "", 1, 20)
	require.EqualValues(t, 4, total)
	require.Len(t, list, 4)
	byID := tksIndex(list)

	assert.EqualValues(t, 4, stats["all"])
	assert.EqualValues(t, 2, stats["active"])
	assert.EqualValues(t, 2, stats["ended"])

	assert.EqualValues(t, 2, byID[f.active.ID]["participant_count"], "A1 两次打卡只算 1 人，加上 B 共 2 人")
	assert.EqualValues(t, 1, byID[f.endedByDate.ID]["participant_count"])
	assert.EqualValues(t, 0, byID[f.endedByStatus.ID]["participant_count"])
	assert.EqualValues(t, 0, byID[f.future.ID]["participant_count"])

	assert.NotContains(t, byID, f.otherClass.ID, "别班任务不得混入")
	assert.NotContains(t, byID, f.otherTenant.ID, "他园任务不得混入")
}

// status=active|ended 在 SQL 过滤；stats 不随筛选与分页变化；未知值回退全部。
func TestTeacherTaskListStatusFilter(t *testing.T) {
	f := newTKSFixture(t)

	list, stats, total := tksList(t, f.svc, f.tcA, "active", 1, 20)
	require.EqualValues(t, 2, total)
	byID := tksIndex(list)
	assert.Contains(t, byID, f.active.ID)
	assert.Contains(t, byID, f.future.ID, "未开始但未过 end_date 的任务仍算进行中")
	assert.NotContains(t, byID, f.endedByDate.ID)
	assert.NotContains(t, byID, f.endedByStatus.ID)
	assert.EqualValues(t, 2, stats["active"], "stats 是全量计数，不受筛选影响")
	assert.EqualValues(t, 4, stats["all"])

	list, _, total = tksList(t, f.svc, f.tcA, "ended", 1, 20)
	require.EqualValues(t, 2, total)
	byID = tksIndex(list)
	assert.Contains(t, byID, f.endedByDate.ID, "过了 end_date 的任务算已结束")
	assert.Contains(t, byID, f.endedByStatus.ID, "status=0 的任务算已结束")
	assert.NotContains(t, byID, f.active.ID)

	// 未知值回退全部，不报错。
	list, stats, total = tksList(t, f.svc, f.tcA, "bogus", 1, 20)
	require.EqualValues(t, 4, total)
	assert.Len(t, list, 4)
	assert.EqualValues(t, 4, stats["all"])

	// 筛选后分页语义不变：total 是筛选结果总数，不是班级全部任务数。
	list, _, total = tksList(t, f.svc, f.tcA, "ended", 2, 1)
	require.EqualValues(t, 2, total)
	require.Len(t, list, 1)
	assert.Equal(t, f.endedByDate.ID, list[0]["id"], "created_at DESC：较早创建的已结束任务在第二页")
}

// 详情两种计数必须区分：submission_count 是行数，participant_count 是人数。
func TestTeacherTaskStatsParticipantVsSubmission(t *testing.T) {
	f := newTKSFixture(t)

	data, err := f.svc.TaskStats(context.Background(), f.tcA, f.active.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 3, data["total_students"], "本班在读 3 人：待入学与软删学生不计")
	assert.EqualValues(t, 3, data["submission_count"], "打卡行数：A1 两条 + B 一条")
	assert.EqualValues(t, 2, data["participant_count"], "去重学生数：A1 与 B")
	assert.EqualValues(t, 1, data["responded_count"])
	assert.EqualValues(t, 2, data["pending_count"])
}

// 空班/无任务：0 与空数组，不报错；空班任务的 total_students 是 0。
func TestTeacherTaskListAndStatsEmptyClass(t *testing.T) {
	f := newTKSFixture(t)

	list, stats, total := tksList(t, f.svc, f.tcEmpty, "", 1, 20)
	require.EqualValues(t, 0, total)
	assert.Empty(t, list)
	assert.EqualValues(t, 0, stats["all"])
	assert.EqualValues(t, 0, stats["active"])
	assert.EqualValues(t, 0, stats["ended"])

	task := domain.ParentTask{
		BaseModel:         domain.BaseModel{ID: testTKSTaskEmpty},
		KindergartenID:    testTKSKG,
		ClassID:           testTKSClassEmpty,
		CreatorID:         testTKSTeacher,
		Title:             "空班任务",
		Frequency:         domain.TaskFrequencyDaily,
		StartDate:         f.day(-1),
		EndDate:           f.day(3),
		Status:            1,
		RecognitionRitual: domain.TaskRitualSticker,
		StarScope:         domain.TaskStarScopeTask,
	}
	require.NoError(t, f.db.Create(&task).Error)

	data, err := f.svc.TaskStats(context.Background(), f.tcEmpty, task.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 0, data["total_students"])
	assert.EqualValues(t, 0, data["submission_count"])
	assert.EqualValues(t, 0, data["participant_count"])
	assert.EqualValues(t, 0, data["responded_count"])
	assert.EqualValues(t, 0, data["pending_count"])
}

// 跨班/跨园隔离：别班任务 404，B 班统计只含自己的任务。
func TestTeacherTaskStatsIsolation(t *testing.T) {
	f := newTKSFixture(t)
	ctx := context.Background()

	_, err := f.svc.TaskStats(ctx, f.tcA, f.otherClass.ID)
	assert.ErrorIs(t, err, ErrTeacherNotFound, "别班任务不可读")
	_, err = f.svc.TaskStats(ctx, f.tcA, f.otherTenant.ID)
	assert.ErrorIs(t, err, ErrTeacherNotFound, "他园任务不可读")

	list, stats, total := tksList(t, f.svc, f.tcB, "", 1, 20)
	require.EqualValues(t, 1, total)
	require.Len(t, list, 1)
	assert.Equal(t, f.otherClass.ID, list[0]["id"])
	assert.EqualValues(t, 1, stats["all"])
	assert.EqualValues(t, 1, stats["active"])
	assert.EqualValues(t, 0, stats["ended"])
}

// total_students 查询失败必须返回错误，不能静默 0。
func TestTeacherTaskStatsTotalStudentsErrorNotSilent(t *testing.T) {
	f := newTKSFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := f.svc.classStudentCountStrict(ctx, f.tcA)
	require.Error(t, err, "查询失败必须返回错误，不能静默 0")
}

// 性能守卫：参与人数列表级只发一条 GROUP BY，stats 只发一条聚合，禁止逐任务查询。
func TestTeacherTaskListUsesBatchAggregates(t *testing.T) {
	f := newTKSFixture(t)
	counter := &clpSQLLogger{Interface: gormlogger.Default.LogMode(gormlogger.Silent)}
	svc := NewTeacherPortalService(
		f.db.Session(&gorm.Session{Logger: counter}),
		nil, nil, repository.NewMessageRepo(f.db), nil, nil, config.FaceConfig{},
	)
	_, _, total := tksList(t, svc, f.tcA, "", 1, 20)
	require.EqualValues(t, 4, total)

	assert.Equal(t, 1, counter.countContaining("COUNT(DISTINCT student_id)"),
		"4 个任务的参与人数只能有一条聚合查询")
	assert.Equal(t, 1, counter.countContaining("THEN 'active' ELSE 'ended'"),
		"stats 三计数必须一次聚合")
}

// 响应形状：list 每项带 participant_count（0 是数字不是 null），顶层带 stats。
func TestTeacherTaskListJSONShape(t *testing.T) {
	f := newTKSFixture(t)
	data, err := f.svc.ListTasksFiltered(context.Background(), f.tcA, 1, 20, "")
	require.NoError(t, err)

	raw, err := json.Marshal(data)
	require.NoError(t, err)
	var decoded struct {
		List  []map[string]json.RawMessage `json:"list"`
		Stats struct {
			All    int64 `json:"all"`
			Active int64 `json:"active"`
			Ended  int64 `json:"ended"`
		} `json:"stats"`
		Total int64 `json:"total"`
	}
	require.NoError(t, json.Unmarshal(raw, &decoded))
	require.Len(t, decoded.List, 4)
	assert.EqualValues(t, 4, decoded.Stats.All)
	assert.EqualValues(t, 2, decoded.Stats.Active)
	assert.EqualValues(t, 2, decoded.Stats.Ended)
	assert.EqualValues(t, 4, decoded.Total)

	for _, item := range decoded.List {
		count, present := item["participant_count"]
		assert.True(t, present, "每项必须带 participant_count")
		// 0 人的任务必须是数字 0：null 反序列化到 int64 会失败。
		var n int64
		require.NoError(t, json.Unmarshal(count, &n), "participant_count 必须是数字，不能是 null")
		assert.GreaterOrEqual(t, n, int64(0))
	}

	detail, err := f.svc.TaskStats(context.Background(), f.tcA, f.active.ID)
	require.NoError(t, err)
	detailRaw, err := json.Marshal(detail)
	require.NoError(t, err)
	var detailDecoded map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(detailRaw, &detailDecoded))
	assert.Contains(t, detailDecoded, "participant_count")
	assert.Contains(t, detailDecoded, "submission_count")
	assert.Contains(t, detailDecoded, "total_students")
}

// 清理硬删：固定 ID 范围内 0 残留（含软删行）。
func TestTeacherTaskStatsCleanupLeavesNoResidue(t *testing.T) {
	f := newTKSFixture(t)
	cleanupTKSData(t, f.db)

	count := func(model any, query string, args ...any) int64 {
		var n int64
		require.NoError(t, f.db.Unscoped().Model(model).Where(query, args...).Count(&n).Error)
		return n
	}
	assert.EqualValues(t, 0, count(&domain.TaskCheckIn{}, "task_id BETWEEN ? AND ?", 994340, 994349))
	assert.EqualValues(t, 0, count(&domain.TaskResponse{}, "task_id BETWEEN ? AND ?", 994340, 994349))
	assert.EqualValues(t, 0, count(&domain.ParentTask{}, "id BETWEEN ? AND ?", 994340, 994349))
	assert.EqualValues(t, 0, count(&domain.Student{}, "id BETWEEN ? AND ?", 994320, 994329))
	assert.EqualValues(t, 0, count(&domain.Class{}, "id BETWEEN ? AND ?", 994310, 994319))
	assert.EqualValues(t, 0, count(&domain.User{}, "id BETWEEN ? AND ?", 994300, 994339))
	assert.EqualValues(t, 0, count(&domain.Kindergarten{}, "id IN (?)", []uint64{testTKSKG, testTKSKGOther}))
}
