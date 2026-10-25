package service

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// B3（通知聚合 + 补充回应未读 + 家长端小程序码）回归。
//
// 断言的是库里的真实消息行与真实聚合结果：通知是否按家长去重、action_path 落在哪个孩子、
// 读标记是否按 (家长, 孩子) 各自一行、同秒是否漏判——这些用内存桩都测不出来。
//
// 运行约束与 B1/B2 相同：固定 ID（992801-992812）、清理全部 Unscoped、
// 不要并发跑两个 go test 进程；跨上海零点运行时「今天」会落到下一周期，重跑即可。

const (
	testB3KG          uint64 = 992801
	testB3Class       uint64 = 992802
	testB3ClassOther  uint64 = 992803
	testB3Teacher     uint64 = 992804
	testB3ParentA     uint64 = 992805 // 两个孩子：A1 + A2（同班）
	testB3ParentB     uint64 = 992806 // 孩子 B
	testB3ParentOther uint64 = 992807 // 别的班孩子 C
	testB3StudentA1   uint64 = 992808
	testB3StudentA2   uint64 = 992809
	testB3StudentB    uint64 = 992810
	testB3StudentC    uint64 = 992811
	testB3CoParentA   uint64 = 992812 // 与 ParentA 同为 A1 的家长，未提交
)

type b3Fixture struct {
	db    *gorm.DB
	svcT  *TeacherPortalService
	svcP  *ParentPortalService
	tc    *TeacherPortalContext
	pcA1  *ParentPortalContext
	pcA2  *ParentPortalContext
	pcB   *ParentPortalContext
	clean func()
}

func setupB3(t *testing.T) *b3Fixture {
	t.Helper()
	db := openTaskB1DB(t)
	loc := shanghaiLoc()

	clean := func() {
		userIDs := []uint64{testB3Teacher, testB3ParentA, testB3ParentB, testB3ParentOther, testB3CoParentA}
		var taskIDs []uint64
		if err := db.Unscoped().Model(&domain.ParentTask{}).
			Where("kindergarten_id = ? OR class_id IN ?", testB3KG, []uint64{testB3Class, testB3ClassOther}).
			Pluck("id", &taskIDs).Error; err != nil {
			t.Logf("B3 夹具：查遗留任务失败: %v", err)
		}
		if len(taskIDs) > 0 {
			var respIDs []uint64
			if err := db.Unscoped().Model(&domain.TaskResponse{}).
				Where("task_id IN ?", taskIDs).Pluck("id", &respIDs).Error; err != nil {
				t.Logf("B3 夹具清理失败 responses(pluck): %v", err)
			}
			if len(respIDs) > 0 {
				if err := db.Unscoped().Where("response_id IN ?", respIDs).
					Delete(&domain.TaskResponseReaction{}).Error; err != nil {
					t.Logf("B3 夹具清理失败 reactions: %v", err)
				}
			}
			for _, step := range []struct {
				name string
				err  error
			}{
				{"supplements", db.Unscoped().Where("check_in_id IN (SELECT id FROM task_check_ins WHERE task_id IN ?)", taskIDs).Delete(&domain.TaskResponseSupplement{}).Error},
				{"checkins", db.Unscoped().Where("task_id IN ?", taskIDs).Delete(&domain.TaskCheckIn{}).Error},
				{"responses", db.Unscoped().Where("task_id IN ?", taskIDs).Delete(&domain.TaskResponse{}).Error},
				{"states", db.Unscoped().Where("task_id IN ?", taskIDs).Delete(&domain.TaskCheckInState{}).Error},
				{"read_marks", db.Unscoped().Where("task_id IN ?", taskIDs).Delete(&domain.TaskReadMark{}).Error},
				{"tasks", db.Unscoped().Where("id IN ?", taskIDs).Delete(&domain.ParentTask{}).Error},
			} {
				if step.err != nil {
					t.Logf("B3 夹具清理失败 %s: %v", step.name, step.err)
				}
			}
		}
		for _, step := range []struct {
			name string
			err  error
		}{
			{"messages", db.Unscoped().Where("user_id IN ?", userIDs).Delete(&domain.Message{}).Error},
			{"student_parents", db.Unscoped().Where("student_id IN ?", []uint64{testB3StudentA1, testB3StudentA2, testB3StudentB, testB3StudentC}).Delete(&domain.StudentParent{}).Error},
			{"students", db.Unscoped().Where("id IN ?", []uint64{testB3StudentA1, testB3StudentA2, testB3StudentB, testB3StudentC}).Delete(&domain.Student{}).Error},
			{"classes", db.Unscoped().Where("id IN ?", []uint64{testB3Class, testB3ClassOther}).Delete(&domain.Class{}).Error},
			{"users", db.Unscoped().Where("id IN ?", userIDs).Delete(&domain.User{}).Error},
			{"kindergarten", db.Unscoped().Where("id = ?", testB3KG).Delete(&domain.Kindergarten{}).Error},
		} {
			if step.err != nil {
				t.Logf("B3 夹具清理失败 %s: %v", step.name, step.err)
			}
		}
	}
	clean()
	t.Cleanup(clean)

	require.NoError(t, db.Unscoped().Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: testB3KG}, Name: "测试园所-B3通知",
	}).Error)
	for _, u := range []struct {
		id    uint64
		name  string
		phone string
	}{
		{testB3Teacher, "B3老师", "13900009701"},
		{testB3ParentA, "B3家长A", "13900009702"},
		{testB3ParentB, "B3家长B", "13900009703"},
		{testB3ParentOther, "B3别班家长", "13900009704"},
		{testB3CoParentA, "B3家长A的配偶", "13900009705"},
	} {
		require.NoError(t, db.Unscoped().Create(&domain.User{
			BaseModel: domain.BaseModel{ID: u.id}, Nickname: u.name,
			Phone: u.phone, PasswordHash: "x",
		}).Error)
	}
	require.NoError(t, db.Unscoped().Create(&domain.Class{
		BaseModel: domain.BaseModel{ID: testB3Class}, KindergartenID: testB3KG,
		Name: "B3测试班", Grade: "小班", Status: 1,
	}).Error)
	require.NoError(t, db.Unscoped().Create(&domain.Class{
		BaseModel: domain.BaseModel{ID: testB3ClassOther}, KindergartenID: testB3KG,
		Name: "B3别班", Grade: "小班", Status: 1,
	}).Error)
	for _, st := range []struct {
		id      uint64
		classID uint64
		name    string
	}{
		{testB3StudentA1, testB3Class, "B3孩子A1"},
		{testB3StudentA2, testB3Class, "B3孩子A2"},
		{testB3StudentB, testB3Class, "B3孩子B"},
		{testB3StudentC, testB3ClassOther, "B3孩子C"},
	} {
		require.NoError(t, db.Unscoped().Create(&domain.Student{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: st.id}},
			KindergartenID:  testB3KG, ClassID: st.classID, Name: st.name, Status: 1,
			Birthday:   time.Date(2021, 5, 1, 0, 0, 0, 0, loc),
			EnrollDate: time.Date(2024, 9, 1, 0, 0, 0, 0, loc),
		}).Error)
	}
	for _, sp := range []struct {
		studentID uint64
		userID    uint64
	}{
		{testB3StudentA1, testB3ParentA},
		{testB3StudentA1, testB3CoParentA},
		{testB3StudentA2, testB3ParentA},
		{testB3StudentB, testB3ParentB},
		{testB3StudentC, testB3ParentOther},
	} {
		require.NoError(t, db.Unscoped().Create(&domain.StudentParent{
			StudentID: sp.studentID, UserID: sp.userID, Relation: domain.RelationMother,
		}).Error)
	}

	f := &b3Fixture{db: db, clean: clean}
	f.svcT = NewTeacherPortalService(db, nil, nil, repository.NewMessageRepo(db), nil, nil, config.FaceConfig{})
	f.svcP = NewParentPortalService(db,
		repository.NewParentRepo(db), repository.NewUserRoleRepo(db),
		repository.NewStudentRepo(db), repository.NewClassRepo(db),
		nil, nil, repository.NewNoticeRepo(db))
	f.tc = &TeacherPortalContext{UserID: testB3Teacher, KindergartenID: testB3KG, ClassID: testB3Class}
	f.pcA1 = b3ParentCtx(testB3ParentA, testB3StudentA1, testB3Class)
	f.pcA2 = b3ParentCtx(testB3ParentA, testB3StudentA2, testB3Class)
	f.pcB = b3ParentCtx(testB3ParentB, testB3StudentB, testB3Class)
	return f
}

func b3ParentCtx(userID, studentID, classID uint64) *ParentPortalContext {
	return &ParentPortalContext{
		UserID:         userID,
		KindergartenID: testB3KG,
		CurrentStudent: &domain.Student{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: studentID}},
			ClassID:         classID,
		},
	}
}

// createTask 走真实的老师创建路径（含发布通知）。
func (f *b3Fixture) createTask(t *testing.T, targetStudentID uint64, title string) map[string]any {
	t.Helper()
	out, err := f.svcT.CreateTask(context.Background(), f.tc, TeacherTaskInput{
		Title:             title,
		LearningLink:      "B3 学习关联",
		TargetStudentID:   targetStudentID,
		Frequency:         string(domain.TaskFrequencyDaily),
		RecognitionRitual: domain.TaskRitualSticker,
	})
	require.NoError(t, err)
	return out
}

// taskFor 按 id 读回任务（测试内部用）。
func (f *b3Fixture) taskFor(t *testing.T, id uint64) domain.ParentTask {
	t.Helper()
	var task domain.ParentTask
	require.NoError(t, f.db.First(&task, id).Error)
	return task
}

// checkin 为指定孩子建一条当天提交。
func (f *b3Fixture) checkin(t *testing.T, taskID, studentID, userID uint64) domain.TaskCheckIn {
	t.Helper()
	loc := shanghaiLoc()
	now := time.Now().In(loc)
	c := domain.TaskCheckIn{
		TaskID: taskID, StudentID: studentID, UserID: userID,
		Date:    time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc),
		Content: "B3 提交",
	}
	require.NoError(t, f.db.Create(&c).Error)
	return c
}

func (f *b3Fixture) messagesTo(t *testing.T, userID uint64) []domain.Message {
	t.Helper()
	var rows []domain.Message
	require.NoError(t, f.db.Where("user_id = ?", userID).Order("id ASC").Find(&rows).Error)
	return rows
}

// messagesByRef 取某业务对象（任务 id / 回应 id）产生的消息。
// 注意：回应测试里同一家长一定还有一条发布通知，不能直接数总条数。
func (f *b3Fixture) messagesByRef(t *testing.T, userID, referenceID uint64) []domain.Message {
	t.Helper()
	var rows []domain.Message
	require.NoError(t, f.db.
		Where("user_id = ? AND reference_id = ?", userID, referenceID).
		Order("id ASC").Find(&rows).Error)
	return rows
}

// ---------- 5.1 任务发布通知 ----------

func TestTaskPublishNotifiesClassFamiliesOncePerParent(t *testing.T) {
	f := setupB3(t)

	out := f.createTask(t, 0, "B3 班级共读")
	task := f.taskFor(t, out["id"].(uint64))

	// 本班家庭：ParentA（两个孩子）+ 配偶 + ParentB = 3 个家长；别班家长不通知。
	assert.Equal(t, 3, out["notified_count"], "按家长去重：同一家长两个孩子只计一次")
	assert.Equal(t, 0, out["notify_failed_count"])

	msgsA := f.messagesTo(t, testB3ParentA)
	require.Len(t, msgsA, 1, "同一任务对同一家长至多一条")
	assert.Equal(t, string(domain.MessageTypeTask), msgsA[0].Type)
	assert.EqualValues(t, task.ID, msgsA[0].ReferenceID)
	assert.Contains(t, msgsA[0].Title, "B3 班级共读")
	assert.Contains(t, msgsA[0].Content, "B3 班级共读")

	// action_path 固定取该家长在本班 id 最小的孩子（A1），不能随查询顺序漂移。
	assert.Equal(t, taskActionPath(task.ID, testB3StudentA1), msgsA[0].ActionPath)
	assert.Contains(t, msgsA[0].ActionPath, "student_id=992808")

	assert.Len(t, f.messagesTo(t, testB3CoParentA), 1)
	assert.Len(t, f.messagesTo(t, testB3ParentB), 1)
	assert.Empty(t, f.messagesTo(t, testB3ParentOther), "别的班家长不应收到本班任务通知")
}

func TestTaskPublishNotifiesTargetFamilyOnly(t *testing.T) {
	f := setupB3(t)

	out := f.createTask(t, testB3StudentA1, "B3 个体任务")
	task := f.taskFor(t, out["id"].(uint64))

	// 目标孩子 A1 的两位家长，各一条；班级其他家庭与别班都不通知。
	assert.Equal(t, 2, out["notified_count"])
	msgsA := f.messagesTo(t, testB3ParentA)
	require.Len(t, msgsA, 1)
	assert.Equal(t, taskActionPath(task.ID, testB3StudentA1), msgsA[0].ActionPath)
	assert.Len(t, f.messagesTo(t, testB3CoParentA), 1)
	assert.Empty(t, f.messagesTo(t, testB3ParentB), "个体任务不通知其他家庭")
	assert.Empty(t, f.messagesTo(t, testB3ParentOther))
}

// 2.1：范围谓词三类覆盖——班级任务全班可见、个体任务只对目标孩子、别班不可见。
func TestParentTaskScopeThreeCases(t *testing.T) {
	f := setupB3(t)
	classID := f.createTask(t, 0, "B3 班级范围")["id"].(uint64)
	individualID := f.createTask(t, testB3StudentA1, "B3 个体范围")["id"].(uint64)

	ids := func(pc *ParentPortalContext) map[uint64]bool {
		list, err := f.svcP.ListTasks(context.Background(), pc, 1, 20)
		require.NoError(t, err)
		out := map[uint64]bool{}
		for _, item := range list["list"].([]map[string]any) {
			out[item["id"].(uint64)] = true
		}
		return out
	}

	a1 := ids(f.pcA1)
	assert.True(t, a1[classID], "班级任务对目标孩子可见")
	assert.True(t, a1[individualID], "个体任务对目标孩子可见")

	b := ids(f.pcB)
	assert.True(t, b[classID], "班级任务对本班其他家庭可见")
	assert.False(t, b[individualID], "个体任务不得对其他家庭可见")

	other := b3ParentCtx(testB3ParentOther, testB3StudentC, testB3ClassOther)
	assert.Empty(t, ids(other), "别班看不到本班任何任务")
}

func TestTaskActionPathOnlyAllowsInternalPages(t *testing.T) {
	assert.True(t, isInternalActionPath(taskActionPath(1, 2)))

	for _, bad := range []string{
		"https://example.com/pages/tasks/detail",
		"pages/tasks/detail?id=1",
		"/pagesx/tasks/detail",
		"/teacher/pages/tasks/detail",
		"",
	} {
		assert.False(t, isInternalActionPath(bad), "非内部 /pages/ 路径必须被拒: %q", bad)
	}
}

// 5.3 的部分失败：一个收件人成功、另一个失败，计数必须分别递增，且不能互相影响。
// 用超长 action_path 定向制造单收件人失败（varchar(255) + STRICT_TRANS_TABLES），
// 不依赖任何 mock。
func TestSendMessagePartialFailureCountsPerRecipient(t *testing.T) {
	f := setupB3(t)
	repo := repository.NewMessageRepo(f.db)
	longPath := "/pages/tasks/detail?pad=" + strings.Repeat("x", 300)

	sent, failed := sendMessageToRecipients(context.Background(), repo,
		[]messageRecipient{
			{UserID: testB3ParentA, ActionPath: taskActionPath(1, testB3StudentA1)},
			{UserID: testB3ParentB, ActionPath: longPath},
		},
		domain.MessageTypeTask, "部分失败标题", "内容", 900004, false)
	assert.Equal(t, 1, sent, "成功的收件人必须计入 sent")
	assert.Equal(t, 1, failed, "失败的收件人必须计入 failed")
	assert.Len(t, f.messagesTo(t, testB3ParentA), 1)
	assert.Empty(t, f.messagesTo(t, testB3ParentB), "失败收件人不能留下半条消息")
}

// 校验必须真的接在写路径上：外部链接不能落库、也不能算作已送达。
func TestSendMessageRejectsExternalActionPath(t *testing.T) {
	f := setupB3(t)
	repo := repository.NewMessageRepo(f.db)

	sent, failed := sendMessageToRecipients(context.Background(), repo,
		[]messageRecipient{{UserID: testB3ParentA, ActionPath: "https://evil.example.com/pages/x"}},
		domain.MessageTypeTask, "标题", "内容", 900002, false)
	assert.Zero(t, sent)
	assert.Equal(t, 1, failed)
	assert.Empty(t, f.messagesTo(t, testB3ParentA), "外部路径不能落库")
}

// ---------- 5.2 主回应按家长聚合 ----------

func TestBatchResponseNotifiesOncePerParent(t *testing.T) {
	f := setupB3(t)
	out := f.createTask(t, 0, "B3 汇总通知")
	task := f.taskFor(t, out["id"].(uint64))

	f.checkin(t, task.ID, testB3StudentA1, testB3ParentA)
	f.checkin(t, task.ID, testB3StudentA2, testB3ParentA)
	f.checkin(t, task.ID, testB3StudentB, testB3ParentB)

	res, err := f.svcT.BatchRespondTask(context.Background(), f.tc, task.ID,
		domain.TaskRitualSticker, "star", "本周都很棒", 0)
	require.NoError(t, err)
	responseID, _ := res["id"].(uint64)
	require.NotZero(t, responseID)

	// 覆盖三个孩子 → 三位家长（ParentA、配偶、ParentB）；ParentA 两条提交只计一次。
	assert.Equal(t, 3, res["notified_count"])
	assert.Equal(t, 0, res["notify_failed_count"])

	msgsA := f.messagesByRef(t, testB3ParentA, responseID)
	require.Len(t, msgsA, 1, "一次主回应覆盖同一家长多条提交只发一条")
	assert.Contains(t, msgsA[0].ActionPath, "student_id=992808")
	assert.Len(t, f.messagesByRef(t, testB3CoParentA, responseID), 1)
	assert.Len(t, f.messagesByRef(t, testB3ParentB, responseID), 1)

	// 补充回应不单独发通知。
	checkins, err := f.respondedCheckinIDs(task.ID)
	require.NoError(t, err)
	require.NotEmpty(t, checkins)
	_, err = f.svcT.SupplementResponse(context.Background(), f.tc, task.ID, checkins[0],
		domain.TaskRitualSticker, "flower", "单独夸一句", 0)
	require.NoError(t, err)
	assert.Len(t, f.messagesTo(t, testB3ParentA), 2, "补充回应不发通知（发布 + 主回应各一条）")
}

// 一次回应覆盖同一家长多个孩子时，深链取「第一个被覆盖的提交」所属的孩子，
// 而不是 id 最小的孩子（design D9）。
func TestBatchResponseActionPathUsesFirstCoveredCheckin(t *testing.T) {
	f := setupB3(t)
	out := f.createTask(t, 0, "B3 深链顺序")
	task := f.taskFor(t, out["id"].(uint64))

	// 先提交 A2，再提交 A1：student_id 是 A1 更小（992808 < 992809），
	// 但「第一个被覆盖的提交」是 A2 的，深链必须按提交顺序而不是最小 student_id 选。
	f.checkin(t, task.ID, testB3StudentA2, testB3ParentA)
	f.checkin(t, task.ID, testB3StudentA1, testB3ParentA)

	res, err := f.svcT.BatchRespondTask(context.Background(), f.tc, task.ID,
		domain.TaskRitualSticker, "star", "本周都很棒", 0)
	require.NoError(t, err)
	responseID, _ := res["id"].(uint64)

	msgs := f.messagesByRef(t, testB3ParentA, responseID)
	require.Len(t, msgs, 1)
	assert.Equal(t, taskActionPath(task.ID, testB3StudentA2), msgs[0].ActionPath,
		"深链必须落在第一个被覆盖的提交上")
}

func (f *b3Fixture) respondedCheckinIDs(taskID uint64) ([]uint64, error) {
	var ids []uint64
	err := f.db.Model(&domain.TaskCheckIn{}).
		Where("task_id = ? AND response_id <> 0", taskID).
		Order("id ASC").Pluck("id", &ids).Error
	return ids, err
}

func TestSingleResponseNotifiesCoveredStudentParents(t *testing.T) {
	f := setupB3(t)
	out := f.createTask(t, 0, "B3 单条通知")
	task := f.taskFor(t, out["id"].(uint64))
	c := f.checkin(t, task.ID, testB3StudentB, testB3ParentB)

	res, err := f.svcT.RespondTask(context.Background(), f.tc, task.ID, c.ID,
		domain.TaskRitualSticker, "heart", "真棒", 0)
	require.NoError(t, err)
	responseID, _ := res["id"].(uint64)

	assert.EqualValues(t, 1, res["notified_count"])
	assert.EqualValues(t, 0, res["notify_failed_count"])
	msgs := f.messagesByRef(t, testB3ParentB, responseID)
	require.Len(t, msgs, 1)
	assert.Equal(t, taskActionPath(task.ID, testB3StudentB), msgs[0].ActionPath)
	assert.Empty(t, f.messagesByRef(t, testB3ParentA, responseID), "只通知被回应提交的家庭")
}

// ---------- 5.3 通知失败不回滚 ----------

func TestTaskAndResponseSurviveNotificationFailure(t *testing.T) {
	f := setupB3(t)

	// 另开一条连接并关掉它：消息写入必然失败，但主业务用的是 f.db。
	badDB := openTaskB1DB(t)
	sqlDB, err := badDB.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	f.svcT = NewTeacherPortalService(f.db, nil, nil, repository.NewMessageRepo(badDB), nil, nil, config.FaceConfig{})

	out := f.createTask(t, testB3StudentA1, "B3 通知失败")
	task := f.taskFor(t, out["id"].(uint64))
	assert.NotZero(t, task.ID, "通知失败不能回滚任务创建")
	assert.EqualValues(t, 0, out["notified_count"])
	assert.EqualValues(t, 2, out["notify_failed_count"], "A1 的两位家长都计入失败")

	c := f.checkin(t, task.ID, testB3StudentA1, testB3ParentA)
	res, err := f.svcT.RespondTask(context.Background(), f.tc, task.ID, c.ID,
		domain.TaskRitualSticker, "star", "回应", 0)
	require.NoError(t, err, "通知失败不能让回应接口报错")
	responseID, _ := res["id"].(uint64)
	assert.EqualValues(t, 0, res["notified_count"])
	assert.EqualValues(t, 2, res["notify_failed_count"])

	var response domain.TaskResponse
	require.NoError(t, f.db.First(&response, responseID).Error, "回应必须已经落库")
	assert.EqualValues(t, responseID, f.reloadCheckinRow(t, c.ID).ResponseID)
}

func (f *b3Fixture) reloadCheckinRow(t *testing.T, id uint64) domain.TaskCheckIn {
	t.Helper()
	var c domain.TaskCheckIn
	require.NoError(t, f.db.First(&c, id).Error)
	return c
}

// ---------- 5.5 补充回应未读与读标记 ----------

// supplementOnce 建一条主回应 + 一条补充回应，返回补充回应的创建时间。
func (f *b3Fixture) supplementOnce(t *testing.T, taskID, studentID, userID uint64) (domain.TaskCheckIn, time.Time) {
	t.Helper()
	c := f.checkin(t, taskID, studentID, userID)
	_, err := f.svcT.RespondTask(context.Background(), f.tc, taskID, c.ID,
		domain.TaskRitualSticker, "star", "主回应", 0)
	require.NoError(t, err)
	sup, err := f.svcT.SupplementResponse(context.Background(), f.tc, taskID, c.ID,
		domain.TaskRitualSticker, "flower", "补充", 0)
	require.NoError(t, err)
	var row domain.TaskResponse
	require.NoError(t, f.db.First(&row, sup["id"].(uint64)).Error)
	return c, row.CreatedAt
}

func TestSupplementUnreadFirstTimeThenClearedByRead(t *testing.T) {
	f := setupB3(t)
	out := f.createTask(t, 0, "B3 未读")
	task := f.taskFor(t, out["id"].(uint64))
	_, _ = f.supplementOnce(t, task.ID, testB3StudentA1, testB3ParentA)

	detail, err := f.svcP.GetTask(context.Background(), f.pcA1, task.ID)
	require.NoError(t, err)
	assert.Equal(t, true, detail["has_unread_supplement"], "首次没有读标记也算未读")

	list, err := f.svcP.ListTasks(context.Background(), f.pcA1, 1, 20)
	require.NoError(t, err)
	items := list["list"].([]map[string]any)
	require.Len(t, items, 1)
	assert.Equal(t, true, items[0]["has_unread_supplement"])

	mark, err := f.svcP.MarkTaskRead(context.Background(), f.pcA1, task.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, mark["last_read_at"])

	detail, err = f.svcP.GetTask(context.Background(), f.pcA1, task.ID)
	require.NoError(t, err)
	assert.Equal(t, false, detail["has_unread_supplement"], "读后必须清除")

	list, err = f.svcP.ListTasks(context.Background(), f.pcA1, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, false, list["list"].([]map[string]any)[0]["has_unread_supplement"])
}

// 10.2：列表要能区分「待完成 / 已提交 / 已回应」，否则前端只能显示两态。
func TestParentTaskListRespondedState(t *testing.T) {
	f := setupB3(t)
	out := f.createTask(t, 0, "B3 列表已回应态")
	taskID := out["id"].(uint64)

	list, err := f.svcP.ListTasks(context.Background(), f.pcA1, 1, 20)
	require.NoError(t, err)
	items := list["list"].([]map[string]any)
	require.Len(t, items, 1)
	assert.Equal(t, false, items[0]["submitted_current_period"], "无提交 = 待完成")
	assert.Equal(t, false, items[0]["responded"])

	c := f.checkin(t, taskID, testB3StudentA1, testB3ParentA)
	list, err = f.svcP.ListTasks(context.Background(), f.pcA1, 1, 20)
	require.NoError(t, err)
	items = list["list"].([]map[string]any)
	assert.Equal(t, true, items[0]["submitted_current_period"], "有提交 = 已提交")
	assert.Equal(t, false, items[0]["responded"], "老师还没回应")

	_, err = f.svcT.RespondTask(context.Background(), f.tc, taskID, c.ID,
		domain.TaskRitualSticker, "star", "回应", 0)
	require.NoError(t, err)
	list, err = f.svcP.ListTasks(context.Background(), f.pcA1, 1, 20)
	require.NoError(t, err)
	items = list["list"].([]map[string]any)
	assert.Equal(t, true, items[0]["responded"], "主回应后 = 已回应")
}

func TestSupplementUnreadSameSecondIsNotMissed(t *testing.T) {
	f := setupB3(t)
	out := f.createTask(t, 0, "B3 同秒")
	task := f.taskFor(t, out["id"].(uint64))
	_, supplementAt := f.supplementOnce(t, task.ID, testB3StudentA1, testB3ParentA)

	// 读标记恰好等于补充回应的时间：用 >= 判定，必须仍算未读。
	require.NoError(t, f.db.Create(&domain.TaskReadMark{
		KindergartenID: testB3KG, TaskID: task.ID, UserID: testB3ParentA,
		StudentID: testB3StudentA1, LastReadAt: &supplementAt,
	}).Error)

	detail, err := f.svcP.GetTask(context.Background(), f.pcA1, task.ID)
	require.NoError(t, err)
	assert.Equal(t, true, detail["has_unread_supplement"], "同秒补充不能被漏判为已读")
}

func TestReadMarkConcurrentDoesNotFail(t *testing.T) {
	f := setupB3(t)
	out := f.createTask(t, 0, "B3 并发读")
	task := f.taskFor(t, out["id"].(uint64))
	f.checkin(t, task.ID, testB3StudentA1, testB3ParentA)

	var wg sync.WaitGroup
	errs := make([]error, 0, 4)
	var mu sync.Mutex
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := f.svcP.MarkTaskRead(context.Background(), f.pcA1, task.ID)
			mu.Lock()
			errs = append(errs, err)
			mu.Unlock()
		}()
	}
	wg.Wait()
	for _, err := range errs {
		require.NoError(t, err, "并发读写标记不能 500")
	}

	var n int64
	require.NoError(t, f.db.Model(&domain.TaskReadMark{}).
		Where("task_id = ? AND user_id = ? AND student_id = ?", task.ID, testB3ParentA, testB3StudentA1).
		Count(&n).Error)
	assert.EqualValues(t, 1, n, "唯一键兜底：只能有一行")
}

func TestReadMarkIsolatedPerParentAndPerChild(t *testing.T) {
	f := setupB3(t)
	out := f.createTask(t, 0, "B3 隔离")
	task := f.taskFor(t, out["id"].(uint64))
	// A1 有补充；A2 也有补充（同一家长的两个孩子）。
	_, _ = f.supplementOnce(t, task.ID, testB3StudentA1, testB3ParentA)
	_, _ = f.supplementOnce(t, task.ID, testB3StudentA2, testB3ParentA)

	_, err := f.svcP.MarkTaskRead(context.Background(), f.pcA1, task.ID)
	require.NoError(t, err)

	// 同一家长的另一个孩子仍算未读。
	detailA2, err := f.svcP.GetTask(context.Background(), f.pcA2, task.ID)
	require.NoError(t, err)
	assert.Equal(t, true, detailA2["has_unread_supplement"], "孩子之间的读标记互不影响")

	// 同孩子的另一位家长仍算未读。
	pcCo := b3ParentCtx(testB3CoParentA, testB3StudentA1, testB3Class)
	detailCo, err := f.svcP.GetTask(context.Background(), pcCo, task.ID)
	require.NoError(t, err)
	assert.Equal(t, true, detailCo["has_unread_supplement"], "家长之间的读标记互不影响")
}

// 任务发布与老师回应共用 type=task，但 reference_id 分别来自 task_id / response_id，
// 两张表的自增 id 会撞数。判重必须带上标题，否则一条正常通知会被当成「已送达」吞掉。
func TestMessageDedupeDistinguishesTaskAndResponseIDs(t *testing.T) {
	f := setupB3(t)
	repo := repository.NewMessageRepo(f.db)
	const ref = uint64(900001)

	require.NoError(t, f.db.Create(&domain.Message{
		UserID: testB3ParentA, Type: string(domain.MessageTypeTask),
		Title: "老师回应了：某任务", ReferenceID: ref,
	}).Error)

	exists, err := repo.Exists(context.Background(), testB3ParentA, string(domain.MessageTypeTask), ref, "老师回应了：某任务")
	require.NoError(t, err)
	assert.True(t, exists, "同一业务文案的重放必须判为已送达")

	exists, err = repo.Exists(context.Background(), testB3ParentA, string(domain.MessageTypeTask), ref, "新亲子任务：某任务")
	require.NoError(t, err)
	assert.False(t, exists, "id 撞数但业务类型不同，不能误判为已送达")
}

// 分享码：任务必须属于当前老师的班级；已软删的任务也不能继续生成。
// （正向生成需要真实微信接口，不在单测范围内——凭据与 envVersion 由 auth 包测试覆盖。）
func TestTaskShareCardRejectsForeignOrDeletedTask(t *testing.T) {
	f := setupB3(t)
	svc := NewTeacherService(f.db, nil, nil, nil, nil, nil, nil, nil, nil)
	out := f.createTask(t, 0, "B3 分享码归属")
	task := f.taskFor(t, out["id"].(uint64))

	_, err := svc.GenerateTaskShareCard(context.Background(), testB3KG, testB3ClassOther, task.ID)
	require.Error(t, err)
	assert.True(t, IsNotFound(err), "别的班不能为这个任务生成码")

	require.NoError(t, f.db.Model(&domain.ParentTask{}).Where("id = ?", task.ID).Delete(&domain.ParentTask{}).Error)
	_, err = svc.GenerateTaskShareCard(context.Background(), testB3KG, testB3Class, task.ID)
	assert.True(t, IsNotFound(err), "已删除的任务不能再生成码")
}

// 补充未读也要覆盖「周期汇总回应之后再补」的路径（此前只测了单条主回应）。
func TestUnreadSupplementUnderPeriodResponse(t *testing.T) {
	f := setupB3(t)
	out := f.createTask(t, 0, "B3 周期补充未读")
	task := f.taskFor(t, out["id"].(uint64))
	c1 := f.checkin(t, task.ID, testB3StudentA1, testB3ParentA)
	f.checkin(t, task.ID, testB3StudentA2, testB3ParentA)

	_, err := f.svcT.BatchRespondTask(context.Background(), f.tc, task.ID,
		domain.TaskRitualSticker, "star", "本周汇总", 0)
	require.NoError(t, err)
	_, err = f.svcT.SupplementResponse(context.Background(), f.tc, task.ID, c1.ID,
		domain.TaskRitualSticker, "flower", "单独夸一句", 0)
	require.NoError(t, err)

	detailA1, err := f.svcP.GetTask(context.Background(), f.pcA1, task.ID)
	require.NoError(t, err)
	assert.Equal(t, true, detailA1["has_unread_supplement"], "周期回应后追加的补充同样要产生未读")

	detailA2, err := f.svcP.GetTask(context.Background(), f.pcA2, task.ID)
	require.NoError(t, err)
	assert.Equal(t, false, detailA2["has_unread_supplement"], "补充只挂在 A1 的提交上")
}

func TestReadMarkRevivesSoftDeletedRow(t *testing.T) {
	f := setupB3(t)
	out := f.createTask(t, 0, "B3 软删读标记")
	task := f.taskFor(t, out["id"].(uint64))
	_, _ = f.supplementOnce(t, task.ID, testB3StudentA1, testB3ParentA)

	// 软删行仍占着 (task_id,user_id,student_id) 唯一键：upsert 命中它时必须清 deleted_at，
	// 否则读标记永远“写不进去”，任务红点清了又回来。
	require.NoError(t, f.db.Create(&domain.TaskReadMark{
		KindergartenID: testB3KG, TaskID: task.ID, UserID: testB3ParentA, StudentID: testB3StudentA1,
	}).Error)
	require.NoError(t, f.db.Where("task_id = ? AND user_id = ? AND student_id = ?",
		task.ID, testB3ParentA, testB3StudentA1).Delete(&domain.TaskReadMark{}).Error)

	_, err := f.svcP.MarkTaskRead(context.Background(), f.pcA1, task.ID)
	require.NoError(t, err)

	detail, err := f.svcP.GetTask(context.Background(), f.pcA1, task.ID)
	require.NoError(t, err)
	assert.Equal(t, false, detail["has_unread_supplement"], "软删行必须被复活为有效读标记")
}

func TestReadMarkRejectsInvisibleTask(t *testing.T) {
	f := setupB3(t)
	out := f.createTask(t, 0, "B3 不可见")
	task := f.taskFor(t, out["id"].(uint64))

	// 别班家长（任务不匹配其孩子班级）→ 404，不泄漏任务存在性。
	pcOther := b3ParentCtx(testB3ParentOther, testB3StudentC, testB3ClassOther)
	_, err := f.svcP.MarkTaskRead(context.Background(), pcOther, task.ID)
	assert.ErrorIs(t, err, ErrParentNotFound)

	// 已结束的任务与详情同口径：不可见。
	require.NoError(t, f.db.Model(&domain.ParentTask{}).Where("id = ?", task.ID).
		Update("end_date", time.Now().AddDate(0, 0, -1)).Error)
	_, err = f.svcP.MarkTaskRead(context.Background(), f.pcA1, task.ID)
	assert.ErrorIs(t, err, ErrParentNotFound)
}
