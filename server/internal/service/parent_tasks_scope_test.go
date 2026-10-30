package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

// B15：家长任务列表三段（active/checked/all）、已结束任务详情、current_checkin、老师昵称。
// 全部跑真实 MySQL：作用域谓词、软删收敛、排序与公开可见性在内存桩上测不出来。

const testB15Teacher2 uint64 = 992813 // 补充回应的第二位老师（setupB3 只建一位老师）

// b15Checkin 建一条提交，日期与创建时间都可指定——checked 排序与 current_checkin
// 依赖创建时间，不能靠 f.checkin 的「当前时刻」。
func b15Checkin(
	t *testing.T, db *gorm.DB, taskID, studentID, userID uint64,
	date, createdAt time.Time, content string,
) domain.TaskCheckIn {
	t.Helper()
	c := domain.TaskCheckIn{
		BaseModel: domain.BaseModel{CreatedAt: createdAt},
		TaskID:    taskID, StudentID: studentID, UserID: userID,
		Date: date, Content: content,
	}
	require.NoError(t, db.Create(&c).Error)
	return c
}

func b15TaskList(
	t *testing.T, svc *ParentPortalService, pc *ParentPortalContext, scope string,
) (order []uint64, byID map[uint64]map[string]any, total int64) {
	t.Helper()
	data, err := svc.ListTasksFiltered(context.Background(), pc, ParentTaskListFilter{Scope: scope}, 1, 50)
	require.NoError(t, err)
	items := data["list"].([]map[string]any)
	order = make([]uint64, 0, len(items))
	byID = make(map[uint64]map[string]any, len(items))
	for _, item := range items {
		id := item["id"].(uint64)
		order = append(order, id)
		byID[id] = item
	}
	return order, byID, data["total"].(int64)
}

func TestParentTaskListScopes(t *testing.T) {
	f := setupB3(t)

	active := f.rawTask(t, testB3Class, 0, "B15 进行中", string(domain.TaskFrequencyDaily), 0, 6)
	future := f.rawTask(t, testB3Class, 0, "B15 未开始", string(domain.TaskFrequencyDaily), 2, 10)
	ended := f.rawTask(t, testB3Class, 0, "B15 已结束", string(domain.TaskFrequencyDaily), -5, -1)
	otherClass := f.rawTask(t, testB3ClassOther, 0, "B15 别班任务", string(domain.TaskFrequencyDaily), 0, 6)

	// 进行中的任务今天有提交（算本周期已提交）；已结束任务的提交更晚创建，用于钉 checked 排序。
	b15Checkin(t, f.db, active.ID, testB3StudentA1, testB3ParentA, f.day(0), time.Now().Add(-2*time.Hour), "进行中提交")
	b15Checkin(t, f.db, ended.ID, testB3StudentA1, testB3ParentA, f.day(-3), time.Now().Add(-30*time.Minute), "已结束提交")

	// 缺省 / 未知 scope 都回退 active：只返回进行中，且 status=1。
	for _, scope := range []string{"", "unknown"} {
		order, byID, total := b15TaskList(t, f.svcP, f.pcA1, scope)
		assert.EqualValues(t, 1, total, "缺省 scope 只统计进行中（%q）", scope)
		assert.Equal(t, []uint64{active.ID}, order)
		assert.EqualValues(t, 1, byID[active.ID]["status"])
		assert.Equal(t, true, byID[active.ID]["submitted_current_period"])
	}

	// all：含已结束与未开始；end_date 倒序、id 倒序；已结束 status=0。
	order, byID, total := b15TaskList(t, f.svcP, f.pcA1, "all")
	assert.EqualValues(t, 3, total)
	assert.Equal(t, []uint64{future.ID, active.ID, ended.ID}, order, "end_date 倒序")
	assert.EqualValues(t, 1, byID[future.ID]["status"], "未开始的任务 status 仍为 1，靠 start_date 区分")
	assert.EqualValues(t, 1, byID[active.ID]["status"])
	assert.EqualValues(t, 0, byID[ended.ID]["status"], "已结束任务 status=0")
	assert.NotContains(t, byID, otherClass.ID, "别班任务不得出现")

	// checked：只含已打卡任务，按最近打卡时间倒序。
	order, byID, total = b15TaskList(t, f.svcP, f.pcA1, "checked")
	assert.EqualValues(t, 2, total)
	assert.Equal(t, []uint64{ended.ID, active.ID}, order, "最近打卡时间倒序")
	assert.EqualValues(t, 0, byID[ended.ID]["status"])
	assert.EqualValues(t, 1, byID[active.ID]["status"])
	assert.NotContains(t, byID, future.ID, "没打卡的任务不进已打卡")
}

func TestParentTaskDetailAllowsEnded(t *testing.T) {
	f := setupB3(t)
	ctx := context.Background()

	ended := f.rawTask(t, testB3Class, 0, "B15 已结束可读", string(domain.TaskFrequencyDaily), -5, -1)
	future := f.rawTask(t, testB3Class, 0, "B15 未开始可读", string(domain.TaskFrequencyDaily), 2, 6)
	otherClass := f.rawTask(t, testB3ClassOther, 0, "B15 别班不可读", string(domain.TaskFrequencyDaily), 0, 6)
	otherChild := f.rawTask(t, testB3Class, testB3StudentB, "B15 别人孩子不可读", string(domain.TaskFrequencyDaily), 0, 6)

	detail, err := f.svcP.GetTask(ctx, f.pcA1, ended.ID)
	require.NoError(t, err, "已结束任务详情不再 404")
	assert.EqualValues(t, 0, detail["status"])
	assert.Nil(t, detail["current_checkin"], "没有提交时 current_checkin 为 null")

	_, err = f.svcP.GetTask(ctx, f.pcA1, future.ID)
	assert.NoError(t, err, "未开始任务同样可读（与详情口径一致）")

	_, err = f.svcP.GetTask(ctx, f.pcA1, otherClass.ID)
	assert.ErrorIs(t, err, ErrParentNotFound, "跨班仍 404")
	_, err = f.svcP.GetTask(ctx, f.pcA1, otherChild.ID)
	assert.ErrorIs(t, err, ErrParentNotFound, "别人的个体任务仍 404")
	_, err = f.svcP.GetTask(ctx, f.pcA1, 999999999)
	assert.ErrorIs(t, err, ErrParentNotFound, "不存在仍 404")
}

func TestParentTaskDetailCurrentCheckin(t *testing.T) {
	f := setupB3(t)
	ctx := context.Background()

	task := f.rawTask(t, testB3Class, 0, "B15 当前提交", string(domain.TaskFrequencyDaily), 0, 6)
	require.NoError(t, f.db.Model(&domain.ParentTask{}).Where("id = ?", task.ID).
		Update("allow_public", true).Error)

	// 无打卡 → null
	detail, err := f.svcP.GetTask(ctx, f.pcA1, task.ID)
	require.NoError(t, err)
	assert.Nil(t, detail["current_checkin"])

	// 两条提交：按创建时间取最近一条（不是最早一条）。
	b15Checkin(t, f.db, task.ID, testB3StudentA1, testB3ParentA, f.day(-1), time.Now().Add(-2*time.Hour), "旧提交")
	newer := b15Checkin(t, f.db, task.ID, testB3StudentA1, testB3ParentA, f.day(0), time.Now().Add(-time.Hour), "新提交")
	require.NoError(t, f.db.Model(&domain.TaskCheckIn{}).Where("id = ?", newer.ID).Updates(map[string]any{
		"media_url":              "https://cdn.example.com/b15/latest.jpg",
		"is_public_requested":    true,
		"public_scope_confirmed": true,
		"public_approved_at":     time.Now(),
		"public_approved_by":     testB3Teacher,
	}).Error)

	detail, err = f.svcP.GetTask(ctx, f.pcA1, task.ID)
	require.NoError(t, err)
	cur, ok := detail["current_checkin"].(map[string]any)
	require.True(t, ok, "有提交时 current_checkin 是对象")
	assert.EqualValues(t, newer.ID, cur["id"], "取最近一条提交")
	assert.Equal(t, "新提交", cur["content"])
	assert.Equal(t, "https://cdn.example.com/b15/latest.jpg", cur["media_url"])
	assert.Equal(t, f.day(0).Format("2006-01-02"), cur["date"])
	assert.Equal(t, "approved", cur["public_status"])
	assert.Equal(t, true, cur["public_visible"])
	assert.NotEmpty(t, cur["created_at"], "「我的提交」需要提交时间")

	// 取消公开后：状态 revoked、不可见（与 CancelPublicCheckin 写入口径一致）。
	require.NoError(t, f.db.Model(&domain.TaskCheckIn{}).Where("id = ?", newer.ID).Updates(map[string]any{
		"is_public_requested":    false,
		"public_scope_confirmed": false,
		"public_approved_at":     nil,
		"public_revoked_at":      time.Now(),
	}).Error)
	detail, err = f.svcP.GetTask(ctx, f.pcA1, task.ID)
	require.NoError(t, err)
	cur = detail["current_checkin"].(map[string]any)
	assert.Equal(t, "revoked", cur["public_status"])
	assert.Equal(t, false, cur["public_visible"])

	// 任务已结束：即使曾批准也不可见（公开墙同口径），状态仍是 approved。
	ended := f.rawTask(t, testB3Class, 0, "B15 结束任务提交", string(domain.TaskFrequencyDaily), -5, -1)
	require.NoError(t, f.db.Model(&domain.ParentTask{}).Where("id = ?", ended.ID).
		Update("allow_public", true).Error)
	ck := b15Checkin(t, f.db, ended.ID, testB3StudentA1, testB3ParentA, f.day(-3), time.Now().Add(-time.Hour), "已结束提交")
	require.NoError(t, f.db.Model(&domain.TaskCheckIn{}).Where("id = ?", ck.ID).Updates(map[string]any{
		"is_public_requested":    true,
		"public_scope_confirmed": true,
		"public_approved_at":     time.Now(),
		"public_approved_by":     testB3Teacher,
	}).Error)
	detail, err = f.svcP.GetTask(ctx, f.pcA1, ended.ID)
	require.NoError(t, err)
	cur = detail["current_checkin"].(map[string]any)
	assert.Equal(t, "approved", cur["public_status"])
	assert.Equal(t, false, cur["public_visible"], "任务结束后公开墙不再展示")
}

func TestParentTaskDetailTeacherNickname(t *testing.T) {
	f := setupB3(t)
	ctx := context.Background()

	require.NoError(t, f.db.Unscoped().Create(&domain.User{
		BaseModel: domain.BaseModel{ID: testB15Teacher2}, Nickname: "B15第二老师",
		Phone: "13900009706", PasswordHash: "x",
	}).Error)
	t.Cleanup(func() {
		f.db.Unscoped().Where("id = ?", testB15Teacher2).Delete(&domain.User{})
	})

	task := f.rawTask(t, testB3Class, 0, "B15 老师昵称", string(domain.TaskFrequencyDaily), 0, 6)
	c := b15Checkin(t, f.db, task.ID, testB3StudentA1, testB3ParentA, f.day(0), time.Now().Add(-time.Hour), "提交")

	main := domain.TaskResponse{
		KindergartenID: testB3KG, TaskID: task.ID, TeacherID: testB3Teacher,
		Ritual: domain.TaskRitualSticker, StickerKey: "star", Content: "主回应",
		ResponseType: domain.TaskResponseSingle,
	}
	require.NoError(t, f.db.Create(&main).Error)
	require.NoError(t, f.db.Model(&domain.TaskCheckIn{}).Where("id = ?", c.ID).
		Update("response_id", main.ID).Error)

	sup := domain.TaskResponse{
		KindergartenID: testB3KG, TaskID: task.ID, TeacherID: testB15Teacher2,
		Ritual: domain.TaskRitualSticker, StickerKey: "flower", Content: "补充",
		ResponseType: domain.TaskResponseSingle,
	}
	require.NoError(t, f.db.Create(&sup).Error)
	require.NoError(t, f.db.Create(&domain.TaskResponseSupplement{
		KindergartenID: testB3KG, ResponseID: sup.ID, CheckInID: c.ID,
	}).Error)

	detail, err := f.svcP.GetTask(ctx, f.pcA1, task.ID)
	require.NoError(t, err)
	responses := detail["responses"].([]map[string]any)
	require.Len(t, responses, 1)
	assert.Equal(t, "B3老师", responses[0]["teacher"].(map[string]any)["nickname"])

	supplements := responses[0]["supplements"].([]map[string]any)
	require.Len(t, supplements, 1)
	assert.Equal(t, "B15第二老师", supplements[0]["teacher"].(map[string]any)["nickname"])
}

// 老师昵称一次 IN 查询：回应对数从 1 组涨到 4 组，查询次数不变。
func TestParentTaskDetailTeacherNicknameNoNPlusOne(t *testing.T) {
	f := setupB3(t)
	ctx := context.Background()

	require.NoError(t, f.db.Unscoped().Create(&domain.User{
		BaseModel: domain.BaseModel{ID: testB15Teacher2}, Nickname: "B15第二老师",
		Phone: "13900009706", PasswordHash: "x",
	}).Error)
	t.Cleanup(func() {
		f.db.Unscoped().Where("id = ?", testB15Teacher2).Delete(&domain.User{})
	})

	task := f.rawTask(t, testB3Class, 0, "B15 昵称批量", string(domain.TaskFrequencyDaily), 0, 6)

	// 一组：提交 + 主回应 + 补充回应。
	addResponsePair := func(dayOffset int, teacherID uint64) {
		c := b15Checkin(t, f.db, task.ID, testB3StudentA1, testB3ParentA,
			f.day(dayOffset), time.Now().Add(time.Duration(dayOffset-10)*time.Hour), "提交")
		main := domain.TaskResponse{
			KindergartenID: testB3KG, TaskID: task.ID, TeacherID: teacherID,
			Ritual: domain.TaskRitualSticker, StickerKey: "star", Content: "主回应",
			ResponseType: domain.TaskResponseSingle,
		}
		require.NoError(t, f.db.Create(&main).Error)
		require.NoError(t, f.db.Model(&domain.TaskCheckIn{}).Where("id = ?", c.ID).
			Update("response_id", main.ID).Error)
		sup := domain.TaskResponse{
			KindergartenID: testB3KG, TaskID: task.ID, TeacherID: testB15Teacher2,
			Ritual: domain.TaskRitualSticker, StickerKey: "flower", Content: "补充",
			ResponseType: domain.TaskResponseSingle,
		}
		require.NoError(t, f.db.Create(&sup).Error)
		require.NoError(t, f.db.Create(&domain.TaskResponseSupplement{
			KindergartenID: testB3KG, ResponseID: sup.ID, CheckInID: c.ID,
		}).Error)
	}
	addResponsePair(0, testB3Teacher)

	counter := &noticeQueryCounter{}
	svc := newParentPortalServiceWithDB(f.db.Session(&gorm.Session{Logger: counter}))
	_, err := svc.GetTask(ctx, f.pcA1, task.ID)
	require.NoError(t, err)
	single := counter.value()

	for i := 1; i <= 3; i++ {
		addResponsePair(-i, testB15Teacher2)
	}
	counter.reset()
	detail, err := svc.GetTask(ctx, f.pcA1, task.ID)
	require.NoError(t, err)
	full := counter.value()

	assert.Len(t, detail["responses"].([]map[string]any), 4)
	t.Logf("1 组回应查询数：%d；4 组回应查询数：%d", single, full)
	assert.Equal(t, single, full, "回应条数增长不得增加查询次数（老师昵称必须批量取）")
	assert.LessOrEqual(t, full, int64(10), "详情查询次数应为常数")
}

// 清理证据：夹具的硬删清理后本园所不留任务/提交/回应/补充/反馈。
func TestParentTaskScopesCleanupLeavesNoResidue(t *testing.T) {
	f := setupB3(t)

	task := f.rawTask(t, testB3Class, 0, "B15 待清理", string(domain.TaskFrequencyDaily), 0, 6)
	c := b15Checkin(t, f.db, task.ID, testB3StudentA1, testB3ParentA, f.day(0), time.Now(), "提交")
	main := domain.TaskResponse{
		KindergartenID: testB3KG, TaskID: task.ID, TeacherID: testB3Teacher,
		Ritual: domain.TaskRitualSticker, StickerKey: "star", Content: "主回应",
		ResponseType: domain.TaskResponseSingle,
	}
	require.NoError(t, f.db.Create(&main).Error)
	require.NoError(t, f.db.Model(&domain.TaskCheckIn{}).Where("id = ?", c.ID).
		Update("response_id", main.ID).Error)
	sup := domain.TaskResponse{
		KindergartenID: testB3KG, TaskID: task.ID, TeacherID: testB3Teacher,
		Ritual: domain.TaskRitualSticker, StickerKey: "flower", Content: "补充",
		ResponseType: domain.TaskResponseSingle,
	}
	require.NoError(t, f.db.Create(&sup).Error)
	require.NoError(t, f.db.Create(&domain.TaskResponseSupplement{
		KindergartenID: testB3KG, ResponseID: sup.ID, CheckInID: c.ID,
	}).Error)
	require.NoError(t, f.db.Create(&domain.TaskResponseReaction{
		KindergartenID: testB3KG, ResponseID: main.ID, UserID: testB3ParentA,
		Reaction: domain.TaskReactionReceived,
	}).Error)

	// 与夹具同一套清理，然后直接数残留。
	f.clean()

	var tasks, checkins, responses, supplements, reactions int64
	require.NoError(t, f.db.Unscoped().Model(&domain.ParentTask{}).
		Where("kindergarten_id = ? OR class_id IN ?", testB3KG, []uint64{testB3Class, testB3ClassOther}).
		Count(&tasks).Error)
	require.NoError(t, f.db.Unscoped().Model(&domain.TaskCheckIn{}).
		Where("student_id IN ?", []uint64{testB3StudentA1, testB3StudentA2, testB3StudentB, testB3StudentC}).
		Count(&checkins).Error)
	require.NoError(t, f.db.Unscoped().Model(&domain.TaskResponse{}).
		Where("kindergarten_id = ?", testB3KG).Count(&responses).Error)
	require.NoError(t, f.db.Unscoped().Model(&domain.TaskResponseSupplement{}).
		Where("kindergarten_id = ?", testB3KG).Count(&supplements).Error)
	require.NoError(t, f.db.Unscoped().Model(&domain.TaskResponseReaction{}).
		Where("user_id IN ?", []uint64{testB3ParentA, testB3ParentB, testB3ParentOther, testB3CoParentA}).
		Count(&reactions).Error)

	assert.Zero(t, tasks, "任务应硬删清零")
	assert.Zero(t, checkins, "提交应硬删清零")
	assert.Zero(t, responses, "回应应硬删清零")
	assert.Zero(t, supplements, "补充回应应硬删清零")
	assert.Zero(t, reactions, "反馈应硬删清零")
}
