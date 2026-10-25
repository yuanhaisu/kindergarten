package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// B2（公开申请 + 老师审核 + 班级公开墙 + 管理端撤回）回归。
//
// 断言的是库里的真实状态与真实查询结果：状态机是否正确、公开墙是否真的按
// design D7 的九个条件过滤、软删行会不会漏出来——这些用内存桩都测不出来。
//
// 运行约束与 B1 相同：
//   - 夹具固定 ID（992701-992711），清理全部用 Unscoped；不要并发跑两个 go test 进程。
//   - 日期取 time.Now()：跨上海零点运行时「今天」会落到下一周期，重跑即可。
//
// 本文件**不**覆盖（不要当成已验证）：
//   - 含其他孩子内容的自动识别：规格要求老师人工确认，服务端只承载 scope_confirmed，
//     没有任何模型能自动判断图片/语音里出现了谁。
//   - 各处的 DB 故障分支与锁等待超时（依赖真实并发时序）。

const (
	testB2KG           uint64 = 992701
	testB2Class        uint64 = 992702
	testB2ClassOther   uint64 = 992703
	testB2Teacher      uint64 = 992704
	testB2Parent       uint64 = 992705 // 孩子 A，本班
	testB2ParentPeer   uint64 = 992706 // 孩子 B，同班
	testB2ParentOther  uint64 = 992707 // 孩子 C，别的班
	testB2Student      uint64 = 992708
	testB2StudentPeer  uint64 = 992709
	testB2StudentOther uint64 = 992710
)

type b2Fixture struct {
	db        *gorm.DB
	taskPub   domain.ParentTask // 本班、当天有效、allow_public=true
	taskNoPub domain.ParentTask // 本班、allow_public=false
	taskSelf  domain.ParentTask // 个体任务（target=孩子 A）
	taskOther domain.ParentTask // 别的班、allow_public=true
	svcT      *TeacherPortalService
	svcP      *ParentPortalService
	svcA      *TaskService
	tc        *TeacherPortalContext
	pc        *ParentPortalContext
	pcPeer    *ParentPortalContext
	pcOther   *ParentPortalContext
	clean     func()
}

func setupB2(t *testing.T) *b2Fixture {
	t.Helper()
	// 复用 B1 的连库 helper：只负责打开 configs/app.ini 里的库，无夹具语义。
	db := openTaskB1DB(t)
	loc := shanghaiLoc()
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	clean := func() {
		var taskIDs []uint64
		if err := db.Unscoped().Model(&domain.ParentTask{}).
			Where("kindergarten_id = ? OR class_id IN ?", testB2KG, []uint64{testB2Class, testB2ClassOther}).
			Pluck("id", &taskIDs).Error; err != nil {
			t.Logf("B2 夹具：查遗留任务失败: %v", err)
		}
		if len(taskIDs) > 0 {
			var respIDs []uint64
			if err := db.Unscoped().Model(&domain.TaskResponse{}).
				Where("task_id IN ?", taskIDs).Pluck("id", &respIDs).Error; err != nil {
				t.Logf("B2 夹具清理失败 responses(pluck): %v", err)
			}
			if len(respIDs) > 0 {
				if err := db.Unscoped().Where("response_id IN ?", respIDs).
					Delete(&domain.TaskResponseReaction{}).Error; err != nil {
					t.Logf("B2 夹具清理失败 reactions: %v", err)
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
					t.Logf("B2 夹具清理失败 %s: %v", step.name, step.err)
				}
			}
		}
		for _, step := range []struct {
			name string
			err  error
		}{
			// 按外键方向：student_parents → students → classes → users → kindergartens。
			// students 上有 fk_students_class、student_parents 上有 fk_student_parents_student
			// （RESTRICT），顺序错了会静默留下整棵孤儿树，下次重跑全部撞键。
			// 公开审核留痕也写 operation_logs：不清会让下一次测试读到上一轮的日志。
			{"operation_logs", db.Unscoped().Where("kindergarten_id = ?", testB2KG).Delete(&domain.OperationLog{}).Error},
			{"student_parents", db.Unscoped().Where("student_id IN ?", []uint64{testB2Student, testB2StudentPeer, testB2StudentOther}).Delete(&domain.StudentParent{}).Error},
			{"students", db.Unscoped().Where("id IN ?", []uint64{testB2Student, testB2StudentPeer, testB2StudentOther}).Delete(&domain.Student{}).Error},
			{"classes", db.Unscoped().Where("id IN ?", []uint64{testB2Class, testB2ClassOther}).Delete(&domain.Class{}).Error},
			{"users", db.Unscoped().Where("id IN ?", []uint64{testB2Teacher, testB2Parent, testB2ParentPeer, testB2ParentOther}).Delete(&domain.User{}).Error},
			{"kindergarten", db.Unscoped().Where("id = ?", testB2KG).Delete(&domain.Kindergarten{}).Error},
		} {
			if step.err != nil {
				t.Logf("B2 夹具清理失败 %s: %v", step.name, step.err)
			}
		}
	}
	clean()
	t.Cleanup(clean)

	require.NoError(t, db.Unscoped().Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: testB2KG}, Name: "测试园所-B2公开",
	}).Error)
	for _, u := range []struct {
		id   uint64
		name string
		ph   string
	}{
		{testB2Teacher, "B2老师", "13900009701"},
		{testB2Parent, "B2家长", "13900009702"},
		{testB2ParentPeer, "B2同班家长", "13900009703"},
		{testB2ParentOther, "B2别班家长", "13900009704"},
	} {
		require.NoError(t, db.Unscoped().Create(&domain.User{
			BaseModel: domain.BaseModel{ID: u.id}, Nickname: u.name, Phone: u.ph, PasswordHash: "x",
		}).Error)
	}
	require.NoError(t, db.Unscoped().Create(&domain.Class{
		BaseModel: domain.BaseModel{ID: testB2Class}, KindergartenID: testB2KG, Name: "B2测试班", Grade: "小班", Status: 1,
	}).Error)
	require.NoError(t, db.Unscoped().Create(&domain.Class{
		BaseModel: domain.BaseModel{ID: testB2ClassOther}, KindergartenID: testB2KG, Name: "B2别的班", Grade: "小班", Status: 1,
	}).Error)
	for _, st := range []struct {
		id    uint64
		class uint64
		name  string
	}{
		{testB2Student, testB2Class, "B2孩子A"},
		{testB2StudentPeer, testB2Class, "B2孩子B"},
		{testB2StudentOther, testB2ClassOther, "B2孩子C"},
	} {
		require.NoError(t, db.Unscoped().Create(&domain.Student{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: st.id}},
			KindergartenID:  testB2KG, ClassID: st.class, Name: st.name, Status: 1,
			Birthday:   time.Date(2021, 5, 1, 0, 0, 0, 0, loc),
			EnrollDate: time.Date(2024, 9, 1, 0, 0, 0, 0, loc),
		}).Error)
	}
	for _, sp := range []struct {
		student, user uint64
		rel           domain.ParentRelation
	}{
		{testB2Student, testB2Parent, domain.RelationMother},
		{testB2StudentPeer, testB2ParentPeer, domain.RelationMother},
		{testB2StudentOther, testB2ParentOther, domain.RelationMother},
	} {
		require.NoError(t, db.Unscoped().Create(&domain.StudentParent{
			StudentID: sp.student, UserID: sp.user, Relation: sp.rel, IsPrimary: true,
		}).Error)
	}

	f := &b2Fixture{db: db, clean: clean}
	mkTask := func(classID, target uint64, allowPublic bool, title string) domain.ParentTask {
		tk := domain.ParentTask{
			KindergartenID: testB2KG, ClassID: classID, CreatorID: testB2Teacher,
			TargetStudentID: target, Title: title, Frequency: domain.TaskFrequencyDaily,
			StartDate: today, EndDate: today.AddDate(0, 0, 6), Status: 1,
			AllowPublic: allowPublic, RecognitionRitual: domain.TaskRitualSticker,
		}
		require.NoError(t, db.Create(&tk).Error)
		return tk
	}
	f.taskPub = mkTask(testB2Class, 0, true, "B2公开任务")
	f.taskNoPub = mkTask(testB2Class, 0, false, "B2不公开任务")
	f.taskSelf = mkTask(testB2Class, testB2Student, false, "B2个体任务")
	f.taskOther = mkTask(testB2ClassOther, 0, true, "B2别班公开任务")

	f.svcT = NewTeacherPortalService(db, nil, nil, nil, nil, nil, config.FaceConfig{})
	f.svcP = NewParentPortalService(db, nil, nil, nil, nil, nil, nil, nil)
	f.svcA = NewTaskService(repository.NewTaskRepo(db), repository.NewOperationLogRepo(db),
		repository.NewMessageRepo(db), db)

	f.tc = &TeacherPortalContext{UserID: testB2Teacher, KindergartenID: testB2KG, ClassID: testB2Class}
	f.pc = b2ParentCtx(testB2Parent, testB2Student, testB2Class)
	f.pcPeer = b2ParentCtx(testB2ParentPeer, testB2StudentPeer, testB2Class)
	f.pcOther = b2ParentCtx(testB2ParentOther, testB2StudentOther, testB2ClassOther)
	return f
}

func b2ParentCtx(userID, studentID, classID uint64) *ParentPortalContext {
	return &ParentPortalContext{
		UserID:         userID,
		KindergartenID: testB2KG,
		CurrentStudent: &domain.Student{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: studentID}},
			ClassID:         classID,
		},
	}
}

// checkin 为指定孩子在该任务下建一条提交。
func (f *b2Fixture) checkin(t *testing.T, task *domain.ParentTask, studentID, userID uint64) domain.TaskCheckIn {
	t.Helper()
	loc := shanghaiLoc()
	now := time.Now().In(loc)
	c := domain.TaskCheckIn{
		TaskID: task.ID, StudentID: studentID, UserID: userID,
		Date:    time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc),
		Content: "B2 提交",
	}
	require.NoError(t, f.db.Create(&c).Error)
	return c
}

func (f *b2Fixture) reload(t *testing.T, id uint64) domain.TaskCheckIn {
	t.Helper()
	var c domain.TaskCheckIn
	require.NoError(t, f.db.First(&c, id).Error)
	return c
}

// wallIDs 返回公开墙上的 checkin id 集合。
func (f *b2Fixture) wallIDs(t *testing.T, pc *ParentPortalContext) []uint64 {
	t.Helper()
	out, err := f.svcP.PublicWall(context.Background(), pc, 1, 50)
	require.NoError(t, err)
	list, ok := out["list"].([]map[string]any)
	require.True(t, ok, "list 应为 []map[string]any")
	ids := make([]uint64, 0, len(list))
	for _, item := range list {
		ids = append(ids, item["checkin_id"].(uint64))
	}
	return ids
}

// auditLogs 读该园所的公开审核日志（按 id 升序）。
func (f *b2Fixture) auditLogs(t *testing.T, action string) []domain.OperationLog {
	t.Helper()
	var out []domain.OperationLog
	require.NoError(t, f.db.Where("kindergarten_id = ? AND action = ?", testB2KG, action).
		Order("id ASC").Find(&out).Error)
	return out
}

// requestAndApprove 走完「家长申请 → 老师确认并批准」，返回提交的当前状态。
func (f *b2Fixture) requestAndApprove(t *testing.T, task *domain.ParentTask, checkinID uint64) {
	t.Helper()
	_, err := f.svcP.RequestPublicCheckin(context.Background(), f.pc, checkinID)
	require.NoError(t, err)
	_, err = f.svcT.ReviewPublicCheckin(context.Background(), f.tc, task.ID, checkinID, true, true)
	require.NoError(t, err)
}

// ---------- 4.1 家长申请 / 取消 ----------

func TestPublicRequestAllowed(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)

	out, err := f.svcP.RequestPublicCheckin(context.Background(), f.pc, c.ID)
	require.NoError(t, err)
	assert.Equal(t, TaskPublicPending, out["public_status"])
	assert.Equal(t, false, out["public_visible"], "申请不等于公开，此时墙上不应可见")

	row := f.reload(t, c.ID)
	assert.True(t, row.IsPublicRequested)
	assert.False(t, row.PublicScopeConfirmed, "申请时必须清空旧确认")
	assert.Nil(t, row.PublicApprovedAt, "申请时必须清空旧批准")
	assert.Zero(t, row.PublicApprovedBy)
	assert.Nil(t, row.PublicRevokedAt, "申请必须清掉旧撤回时间，否则状态机停在 revoked")

	// 家长端读路径要能看到自己提交的公开状态（UI 的「申请公开」开关依赖它）。
	hist, err := f.svcP.CheckinHistory(context.Background(), f.pc, f.taskPub.ID, 1, 20)
	require.NoError(t, err)
	rows := hist["list"].([]map[string]any)
	require.Len(t, rows, 1)
	assert.Equal(t, true, rows[0]["is_public_requested"])
	assert.Equal(t, false, rows[0]["public_visible"], "待审核不得显示为已公开")
}

// 4.5：一键完成（无文字、无媒体）没有可公开的内容，不能进审核队列。
func TestPublicRequestRejectsEmptyOneTapCheckin(t *testing.T) {
	f := setupB2(t)
	loc := shanghaiLoc()
	now := time.Now().In(loc)
	c := domain.TaskCheckIn{
		TaskID: f.taskPub.ID, StudentID: testB2Student, UserID: testB2Parent,
		Date: time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc),
	}
	require.NoError(t, f.db.Create(&c).Error)

	_, err := f.svcP.RequestPublicCheckin(context.Background(), f.pc, c.ID)
	assert.ErrorIs(t, err, ErrTaskPublicEmptyContent)
	assert.False(t, f.reload(t, c.ID).IsPublicRequested, "被拒绝时不得写入申请状态")

	// 纯空白文字同样算「没有内容」。
	c.Content = "  \n "
	require.NoError(t, f.db.Save(&c).Error)
	_, err = f.svcP.RequestPublicCheckin(context.Background(), f.pc, c.ID)
	assert.ErrorIs(t, err, ErrTaskPublicEmptyContent)

	// 只有 media_file_id（URL 由读取时生成）也算有内容：不能误拦照片/语音提交。
	c.Content = ""
	c.MediaFileID = 9927999
	require.NoError(t, f.db.Save(&c).Error)
	_, err = f.svcP.RequestPublicCheckin(context.Background(), f.pc, c.ID)
	require.NoError(t, err)
	assert.True(t, f.reload(t, c.ID).IsPublicRequested)
}

// 4.7：PublicApprovedBy 被清空后，日志是唯一能回答「谁批准过」的地方。
func TestPublicAuditTrailSurvivesApproverReset(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)
	f.requestAndApprove(t, &f.taskPub, c.ID)

	assert.Equal(t, testB2Teacher, f.reload(t, c.ID).PublicApprovedBy)
	approveLogs := f.auditLogs(t, PublicAuditApprove)
	require.Len(t, approveLogs, 1)
	assert.Equal(t, testB2Teacher, approveLogs[0].UserID)
	assert.Equal(t, c.ID, approveLogs[0].TargetID)
	assert.Contains(t, approveLogs[0].Detail, "prior_status=pending")

	// 家长取消：审核人被清空，但批准记录不能被抹掉。
	_, err := f.svcP.CancelPublicCheckin(context.Background(), f.pc, c.ID)
	require.NoError(t, err)
	assert.Zero(t, f.reload(t, c.ID).PublicApprovedBy, "取消后审核人必须清空")
	assert.Len(t, f.auditLogs(t, PublicAuditApprove), 1, "批准日志必须保留")

	cancelLogs := f.auditLogs(t, PublicAuditCancel)
	require.Len(t, cancelLogs, 1)
	assert.Equal(t, testB2Parent, cancelLogs[0].UserID)
	assert.Contains(t, cancelLogs[0].Detail, fmt.Sprintf("approved_by=%d", testB2Teacher),
		"撤销日志要记下变更前的审核人")

	// 重新申请后老师拒绝：同样留痕，且 prior_status 是 pending。
	_, err = f.svcP.RequestPublicCheckin(context.Background(), f.pc, c.ID)
	require.NoError(t, err)
	_, err = f.svcT.ReviewPublicCheckin(context.Background(), f.tc, f.taskPub.ID, c.ID, false, false)
	require.NoError(t, err)
	revokeLogs := f.auditLogs(t, PublicAuditRevoke)
	require.Len(t, revokeLogs, 1)
	assert.Equal(t, testB2Teacher, revokeLogs[0].UserID)
	assert.Contains(t, revokeLogs[0].Detail, "prior_status=pending")
}

// 审计页按 action 渲染中文标签；新增动作不能掉进 default 显示成裸英文。
func TestPublicAuditActionLabels(t *testing.T) {
	for action, want := range map[string]string{
		PublicAuditApprove:    "批准公开",
		PublicAuditRevoke:     "撤回公开",
		PublicAuditCancel:     "家长取消公开",
		PublicAuditRevokeTask: "关闭任务公开",
	} {
		assert.Equal(t, want, actionLabelFor(action), "action=%s", action)
	}
}

// 4.7：任务级撤销（关公开 / 改个体）也要留痕，且空操作不写假记录。
func TestRevokePublicForTaskWritesAuditLog(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)
	f.requestAndApprove(t, &f.taskPub, c.ID)

	require.NoError(t, f.db.Transaction(func(tx *gorm.DB) error {
		return revokePublicForTask(context.Background(), tx, testB2KG, testB2Teacher, f.taskPub.ID, "allow_public_off")
	}))
	logs := f.auditLogs(t, PublicAuditRevokeTask)
	require.Len(t, logs, 1)
	assert.Equal(t, f.taskPub.ID, logs[0].TargetID)
	assert.Equal(t, testB2Teacher, logs[0].UserID)
	assert.Contains(t, logs[0].Detail, "reason=allow_public_off")
	assert.Contains(t, logs[0].Detail, "rows=1")

	// 已经没有可撤销的公开状态：不得再写一条「撤回了一次」。
	require.NoError(t, f.db.Transaction(func(tx *gorm.DB) error {
		return revokePublicForTask(context.Background(), tx, testB2KG, testB2Teacher, f.taskPub.ID, "allow_public_off")
	}))
	assert.Len(t, f.auditLogs(t, PublicAuditRevokeTask), 1, "空操作不写日志")
}

func TestPublicRequestRejectedWhenTaskNotAllowPublic(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskNoPub, testB2Student, testB2Parent)

	_, err := f.svcP.RequestPublicCheckin(context.Background(), f.pc, c.ID)
	assert.ErrorIs(t, err, ErrTaskPublicNotAllowed)

	row := f.reload(t, c.ID)
	assert.False(t, row.IsPublicRequested, "被拒绝时不得写入申请状态")
}

func TestPublicRequestRejectedForIndividualTask(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskSelf, testB2Student, testB2Parent)

	_, err := f.svcP.RequestPublicCheckin(context.Background(), f.pc, c.ID)
	assert.ErrorIs(t, err, ErrTaskPublicNotAllowed)
	assert.False(t, f.reload(t, c.ID).IsPublicRequested)
}

func TestPublicRequestRejectedCrossFamily(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)

	// 同班另一位家长申请别人孩子的提交。
	_, err := f.svcP.RequestPublicCheckin(context.Background(), f.pcPeer, c.ID)
	assert.ErrorIs(t, err, ErrParentForbidden)
	assert.False(t, f.reload(t, c.ID).IsPublicRequested)
}

func TestPublicRequestIdempotentWhenApproved(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)
	f.requestAndApprove(t, &f.taskPub, c.ID)

	before := f.reload(t, c.ID).PublicApprovedAt
	require.NotNil(t, before)

	// 家长再点一次「申请公开」：不能让已公开内容回到待审核。
	out, err := f.svcP.RequestPublicCheckin(context.Background(), f.pc, c.ID)
	require.NoError(t, err)
	assert.Equal(t, TaskPublicApproved, out["public_status"])

	after := f.reload(t, c.ID)
	require.NotNil(t, after.PublicApprovedAt)
	assert.True(t, before.Equal(*after.PublicApprovedAt), "重复申请不得刷新或清空批准时间")
	assert.True(t, after.PublicScopeConfirmed)
	assert.Contains(t, f.wallIDs(t, f.pc), c.ID)
}

func TestPublicCancelClearsApproval(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)
	f.requestAndApprove(t, &f.taskPub, c.ID)
	require.Contains(t, f.wallIDs(t, f.pc), c.ID)

	out, err := f.svcP.CancelPublicCheckin(context.Background(), f.pc, c.ID)
	require.NoError(t, err)
	assert.Equal(t, TaskPublicRevoked, out["public_status"])
	assert.Equal(t, false, out["public_visible"])

	row := f.reload(t, c.ID)
	assert.False(t, row.IsPublicRequested)
	assert.False(t, row.PublicScopeConfirmed, "取消必须清空确认")
	assert.Nil(t, row.PublicApprovedAt, "取消必须清空批准")
	assert.Zero(t, row.PublicApprovedBy, "取消必须清空审核人")
	require.NotNil(t, row.PublicRevokedAt)

	assert.NotContains(t, f.wallIDs(t, f.pc), c.ID, "取消后必须立即从墙上消失")
}

func TestPublicCancelWhenNeverRequestedIsNoop(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)

	out, err := f.svcP.CancelPublicCheckin(context.Background(), f.pc, c.ID)
	require.NoError(t, err)
	assert.Equal(t, TaskPublicNone, out["public_status"])

	row := f.reload(t, c.ID)
	assert.Nil(t, row.PublicRevokedAt, "从未申请过时取消不得凭空写一条撤回记录")
}

func TestPublicWallDefaultPrivate(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)

	// 其他家庭与本家庭都看不到：默认私密。
	assert.NotContains(t, f.wallIDs(t, f.pcPeer), c.ID)
	assert.NotContains(t, f.wallIDs(t, f.pc), c.ID)
}

// 撤回不能被任务状态锁死：家长随时要能把自己的内容撤下来。
func TestPublicCancelWorksAfterPublicDisabled(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)
	f.requestAndApprove(t, &f.taskPub, c.ID)

	require.NoError(t, f.db.Model(&domain.ParentTask{}).Where("id = ?", f.taskPub.ID).
		Update("allow_public", false).Error)

	out, err := f.svcP.CancelPublicCheckin(context.Background(), f.pc, c.ID)
	require.NoError(t, err, "关掉公开开关后家长仍必须能撤回")
	assert.Equal(t, TaskPublicRevoked, out["public_status"])
	assert.Nil(t, f.reload(t, c.ID).PublicApprovedAt)
}

func TestPublicCancelWorksAfterTaskEnded(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)
	f.requestAndApprove(t, &f.taskPub, c.ID)

	loc := shanghaiLoc()
	now := time.Now().In(loc)
	yesterday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -1)
	require.NoError(t, f.db.Model(&domain.ParentTask{}).Where("id = ?", f.taskPub.ID).
		Update("end_date", yesterday).Error)

	out, err := f.svcP.CancelPublicCheckin(context.Background(), f.pc, c.ID)
	require.NoError(t, err, "任务结束后家长仍必须能撤回")
	assert.Equal(t, TaskPublicRevoked, out["public_status"])
}

// ---------- 4.2 老师审核 ----------

func TestTeacherApproveRequiresScopeConfirmed(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)
	_, err := f.svcP.RequestPublicCheckin(context.Background(), f.pc, c.ID)
	require.NoError(t, err)

	_, err = f.svcT.ReviewPublicCheckin(context.Background(), f.tc, f.taskPub.ID, c.ID, true, false)
	assert.ErrorIs(t, err, ErrTaskPublicScopeUnconfirmed)

	row := f.reload(t, c.ID)
	assert.Nil(t, row.PublicApprovedAt)
	assert.False(t, row.PublicScopeConfirmed)
}

func TestTeacherApproveRequiresPendingState(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)

	// 家长没申请过，老师直接批准：必须拒绝，不能凭老师记忆放行。
	_, err := f.svcT.ReviewPublicCheckin(context.Background(), f.tc, f.taskPub.ID, c.ID, true, true)
	assert.ErrorIs(t, err, ErrTaskPublicStateConflict)
	assert.Nil(t, f.reload(t, c.ID).PublicApprovedAt)
}

func TestTeacherApproveRecordsAuditAndMakesVisible(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)
	_, err := f.svcP.RequestPublicCheckin(context.Background(), f.pc, c.ID)
	require.NoError(t, err)

	out, err := f.svcT.ReviewPublicCheckin(context.Background(), f.tc, f.taskPub.ID, c.ID, true, true)
	require.NoError(t, err)
	assert.Equal(t, TaskPublicApproved, out["public_status"])
	assert.Equal(t, true, out["public_visible"])

	row := f.reload(t, c.ID)
	require.NotNil(t, row.PublicApprovedAt)
	assert.Equal(t, testB2Teacher, row.PublicApprovedBy, "必须留痕审核人")
	assert.True(t, row.PublicScopeConfirmed)
	assert.Nil(t, row.PublicRevokedAt)

	// 本班其他家长能看到；别的班看不到。
	assert.Contains(t, f.wallIDs(t, f.pcPeer), c.ID)
	assert.NotContains(t, f.wallIDs(t, f.pcOther), c.ID)

	hist, err := f.svcP.CheckinHistory(context.Background(), f.pc, f.taskPub.ID, 1, 20)
	require.NoError(t, err)
	rows := hist["list"].([]map[string]any)
	require.Len(t, rows, 1)
	assert.Equal(t, true, rows[0]["public_visible"], "批准后家长端读路径应与公开墙同口径")
}

func TestTeacherRejectClearsApproval(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)
	f.requestAndApprove(t, &f.taskPub, c.ID)

	out, err := f.svcT.ReviewPublicCheckin(context.Background(), f.tc, f.taskPub.ID, c.ID, false, false)
	require.NoError(t, err)
	assert.Equal(t, TaskPublicRevoked, out["public_status"])

	row := f.reload(t, c.ID)
	assert.False(t, row.IsPublicRequested)
	assert.False(t, row.PublicScopeConfirmed)
	assert.Nil(t, row.PublicApprovedAt)
	assert.Zero(t, row.PublicApprovedBy)
	require.NotNil(t, row.PublicRevokedAt)
	assert.NotContains(t, f.wallIDs(t, f.pcPeer), c.ID)
}

func TestTeacherReviewRejectsForeignClassTask(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskOther, testB2StudentOther, testB2ParentOther)

	// 别的班的任务：即使 checkinID 正确也必须 404（老师端权限按班级收窄）。
	_, err := f.svcT.ReviewPublicCheckin(context.Background(), f.tc, f.taskOther.ID, c.ID, true, true)
	assert.ErrorIs(t, err, ErrTeacherNotFound)
}

func TestTeacherReviewRejectsWrongTaskCheckinPair(t *testing.T) {
	f := setupB2(t)
	loc := shanghaiLoc()
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	// 同班、允许公开的第二个任务：用它证明「提交不属于该任务」与
	// 「任务不允许公开」是两条独立的校验，前者不能被后者掩盖成 404。
	sibling := domain.ParentTask{
		KindergartenID: testB2KG, ClassID: testB2Class, CreatorID: testB2Teacher,
		Title: "B2同班另一个公开任务", Frequency: domain.TaskFrequencyDaily,
		StartDate: today, EndDate: today.AddDate(0, 0, 6), Status: 1, AllowPublic: true,
	}
	require.NoError(t, f.db.Create(&sibling).Error)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)

	_, err := f.svcT.ReviewPublicCheckin(context.Background(), f.tc, sibling.ID, c.ID, true, true)
	assert.ErrorIs(t, err, ErrTeacherInvalidParams)
}

func TestPublicRequestRejectedWhenTaskEnded(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)

	loc := shanghaiLoc()
	now := time.Now().In(loc)
	lastWeek := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -7)
	require.NoError(t, f.db.Model(&domain.ParentTask{}).Where("id = ?", f.taskPub.ID).
		Updates(map[string]any{
			"start_date": lastWeek, "end_date": lastWeek.AddDate(0, 0, 1),
		}).Error)

	// 已结束任务的申请永远不可能出现在墙上，直接拒绝比静默接受更诚实。
	_, err := f.svcP.RequestPublicCheckin(context.Background(), f.pc, c.ID)
	assert.ErrorIs(t, err, ErrTaskPublicNotAllowed)
	assert.False(t, f.reload(t, c.ID).IsPublicRequested)
}

func TestTeacherApproveRejectedWhenPublicTurnedOff(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)
	_, err := f.svcP.RequestPublicCheckin(context.Background(), f.pc, c.ID)
	require.NoError(t, err)

	// 老师打开审核页之后、点确认之前，任务被关掉了公开开关。
	require.NoError(t, f.db.Model(&domain.ParentTask{}).Where("id = ?", f.taskPub.ID).
		Update("allow_public", false).Error)

	_, err = f.svcT.ReviewPublicCheckin(context.Background(), f.tc, f.taskPub.ID, c.ID, true, true)
	assert.ErrorIs(t, err, ErrTaskPublicNotAllowed)
	assert.Nil(t, f.reload(t, c.ID).PublicApprovedAt)

	// 但「撤回」不受任务开关限制：关掉公开后老师仍必须能把残留状态清干净。
	out, err := f.svcT.ReviewPublicCheckin(context.Background(), f.tc, f.taskPub.ID, c.ID, false, false)
	require.NoError(t, err)
	assert.Equal(t, TaskPublicRevoked, out["public_status"])
}

// ---------- 4.3/4.4 公开墙条件 ----------

func TestPublicWallHidesWhenAllowPublicTurnedOff(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)
	f.requestAndApprove(t, &f.taskPub, c.ID)
	require.Contains(t, f.wallIDs(t, f.pcPeer), c.ID)

	// 老师关掉 allow_public（2.9 会同时 revokePublicForTask，两件事分开验证：
	// 这里只改开关，证明公开墙自己不依赖撤销副作用）。
	require.NoError(t, f.db.Model(&domain.ParentTask{}).Where("id = ?", f.taskPub.ID).
		Update("allow_public", false).Error)
	assert.NotContains(t, f.wallIDs(t, f.pcPeer), c.ID)

	// 恢复开关也不能让旧批准复活——但这是 revoke 的职责；仅把开关改回来时
	// 批准字段仍在，属于「未走撤销路径」的异常数据。这里断言 revoke 的真正效果。
	require.NoError(t, f.db.Transaction(func(tx *gorm.DB) error {
		return revokePublicForTask(context.Background(), tx, testB2KG, testB2Teacher, f.taskPub.ID, "test")
	}))
	require.NoError(t, f.db.Model(&domain.ParentTask{}).Where("id = ?", f.taskPub.ID).
		Update("allow_public", true).Error)
	assert.NotContains(t, f.wallIDs(t, f.pcPeer), c.ID, "撤销后重新打开公开，旧批准不得恢复")
}

func TestPublicWallHidesEndedTask(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)
	f.requestAndApprove(t, &f.taskPub, c.ID)
	require.Contains(t, f.wallIDs(t, f.pcPeer), c.ID)

	loc := shanghaiLoc()
	now := time.Now().In(loc)
	yesterday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -1)
	require.NoError(t, f.db.Model(&domain.ParentTask{}).Where("id = ?", f.taskPub.ID).
		Update("end_date", yesterday).Error)

	assert.NotContains(t, f.wallIDs(t, f.pcPeer), c.ID, "任务结束后公开内容不再展示")
}

func TestPublicWallHidesClosedTask(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)
	f.requestAndApprove(t, &f.taskPub, c.ID)

	require.NoError(t, f.db.Model(&domain.ParentTask{}).Where("id = ?", f.taskPub.ID).
		Update("status", 0).Error)
	assert.NotContains(t, f.wallIDs(t, f.pcPeer), c.ID, "已关闭任务的内容不得上墙")
}

func TestPublicWallHidesSoftDeletedTask(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)
	f.requestAndApprove(t, &f.taskPub, c.ID)
	require.Contains(t, f.wallIDs(t, f.pcPeer), c.ID)

	// 只软删任务行，提交保持存活。DeleteTask 的级联会连提交一起软删，
	// 那样只能证明 c.deleted_at 过滤，证明不了 t.deleted_at 过滤。
	require.NoError(t, f.db.Where("id = ?", f.taskPub.ID).Delete(&domain.ParentTask{}).Error)
	assert.NotContains(t, f.wallIDs(t, f.pcPeer), c.ID, "任务软删后不得再返回（不能只靠提交软删兜底）")
}

func TestPublicWallHidesAfterDeleteTaskCascade(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)
	f.requestAndApprove(t, &f.taskPub, c.ID)
	require.Contains(t, f.wallIDs(t, f.pcPeer), c.ID)

	require.NoError(t, f.svcT.DeleteTask(context.Background(), f.tc, f.taskPub.ID))
	assert.NotContains(t, f.wallIDs(t, f.pcPeer), c.ID, "删除任务后公开墙不得再返回")
}

func TestPublicWallHidesRevokedEvenIfApprovalLingers(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)

	// 「已批准但已撤回」的残留形态：approved_at 与 revoked_at 同时存在。
	// 只要撤回时间为空这一条没被守住，取消/撤回就会漏到墙上。
	now := time.Now()
	require.NoError(t, f.db.Model(&domain.TaskCheckIn{}).Where("id = ?", c.ID).Updates(map[string]any{
		"is_public_requested":    true,
		"public_scope_confirmed": true,
		"public_approved_at":     now,
		"public_approved_by":     testB2Teacher,
		"public_revoked_at":      now,
	}).Error)
	assert.NotContains(t, f.wallIDs(t, f.pcPeer), c.ID, "public_revoked_at 非空时不得上墙")
}

func TestPublicWallRequiresPublicRequest(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)

	// 「有确认、有批准、但没有申请」的残留形态：例如取消路径哪天漏写 is_public_requested。
	now := time.Now()
	require.NoError(t, f.db.Model(&domain.TaskCheckIn{}).Where("id = ?", c.ID).Updates(map[string]any{
		"is_public_requested":    false,
		"public_scope_confirmed": true,
		"public_approved_at":     now,
		"public_approved_by":     testB2Teacher,
		"public_revoked_at":      nil,
	}).Error)
	assert.NotContains(t, f.wallIDs(t, f.pcPeer), c.ID, "未处于申请状态时不得上墙")
}

func TestPublicWallRequiresApprovedAt(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)

	// 「有申请、有确认、没有批准时间」：这不该算已公开。
	require.NoError(t, f.db.Model(&domain.TaskCheckIn{}).Where("id = ?", c.ID).Updates(map[string]any{
		"is_public_requested":    true,
		"public_scope_confirmed": true,
		"public_approved_at":     nil,
		"public_revoked_at":      nil,
	}).Error)
	assert.NotContains(t, f.wallIDs(t, f.pcPeer), c.ID, "没有批准时间的记录不得上墙")
	assert.Equal(t, TaskPublicPending, publicCheckinStatus(ptrCheckin(f.reload(t, c.ID))))

	require.NoError(t, f.db.Model(&domain.TaskCheckIn{}).Where("id = ?", c.ID).
		Updates(map[string]any{"public_approved_at": time.Now(), "public_approved_by": testB2Teacher}).Error)
	assert.Contains(t, f.wallIDs(t, f.pcPeer), c.ID)
}

func TestPublicWallHidesSoftDeletedCheckin(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)
	f.requestAndApprove(t, &f.taskPub, c.ID)
	require.Contains(t, f.wallIDs(t, f.pcPeer), c.ID)

	// 只软删提交，任务保持存活：证明 c.deleted_at 这一条单独生效
	// （DeleteTask 级联会连任务一起软删，那样测不出来）。
	require.NoError(t, f.db.Where("id = ?", c.ID).Delete(&domain.TaskCheckIn{}).Error)
	assert.NotContains(t, f.wallIDs(t, f.pcPeer), c.ID, "软删的提交不得上墙")
}

func TestPublicWallOrdersNewestFirst(t *testing.T) {
	f := setupB2(t)
	older := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)
	newer := f.checkin(t, &f.taskPub, testB2StudentPeer, testB2ParentPeer)
	f.requestAndApprove(t, &f.taskPub, older.ID)
	_, err := f.svcP.RequestPublicCheckin(context.Background(), f.pcPeer, newer.ID)
	require.NoError(t, err)
	_, err = f.svcT.ReviewPublicCheckin(context.Background(), f.tc, f.taskPub.ID, newer.ID, true, true)
	require.NoError(t, err)

	now := time.Now()
	require.NoError(t, f.db.Model(&domain.TaskCheckIn{}).Where("id = ?", older.ID).
		Update("created_at", now.Add(-time.Hour)).Error)
	require.NoError(t, f.db.Model(&domain.TaskCheckIn{}).Where("id = ?", newer.ID).
		Update("created_at", now).Error)

	assert.Equal(t, []uint64{newer.ID, older.ID}, f.wallIDs(t, f.pc), "公开墙按新→旧排序")
}

// 转班：内容不能继续留在原班级墙上，但撤回入口也不能因此锁死。
func TestPublicContentFollowsStudentClassMove(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)
	f.requestAndApprove(t, &f.taskPub, c.ID)
	require.Contains(t, f.wallIDs(t, f.pcPeer), c.ID)

	require.NoError(t, f.db.Model(&domain.Student{}).Where("id = ?", testB2Student).
		Update("class_id", testB2ClassOther).Error)
	f.pc.CurrentStudent.ClassID = testB2ClassOther

	assert.NotContains(t, f.wallIDs(t, f.pcPeer), c.ID, "转班后原班级墙不得继续展示该孩子的内容")

	out, err := f.svcP.CancelPublicCheckin(context.Background(), f.pc, c.ID)
	require.NoError(t, err, "转班不能锁死撤回入口")
	assert.Equal(t, TaskPublicRevoked, out["public_status"])

	// 但也不能再对原班级任务申请公开：那会把内容放到一个他已经不属于的班上。
	// 契约把「任务仍属于孩子当前班级」列为允许条件——不满足是 422，不是 404。
	_, err = f.svcP.RequestPublicCheckin(context.Background(), f.pc, c.ID)
	assert.ErrorIs(t, err, ErrTaskPublicNotAllowed)
}

// 管理端把任务改派到别的班（7.1 结构字段锁定落地前仍可达）：
// 原班孩子的照片/文字不得出现在新班家长面前。
func TestPublicWallHidesWhenTaskReassignedToOtherClass(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)
	f.requestAndApprove(t, &f.taskPub, c.ID)
	require.Contains(t, f.wallIDs(t, f.pcPeer), c.ID)

	require.NoError(t, f.db.Model(&domain.ParentTask{}).Where("id = ?", f.taskPub.ID).
		Update("class_id", testB2ClassOther).Error)

	assert.NotContains(t, f.wallIDs(t, f.pcOther), c.ID, "改派后原班孩子的内容不得漏到新班")
}

// 编辑一个本来就不公开的任务，不得给提交盖「已撤回」时间戳。
// 否则「从未申请」会被洗成「已撤回」，且撤回时间随每次无关编辑漂移。
func TestEditingNonPublicTaskDoesNotStampRevokedAt(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskNoPub, testB2Student, testB2Parent)
	require.Nil(t, f.reload(t, c.ID).PublicRevokedAt)

	require.NoError(t, f.svcT.UpdateTask(context.Background(), f.tc, f.taskNoPub.ID, TeacherTaskInput{
		Title: "B2改名任务", LearningLink: "B2学习关联",
	}))

	var updated domain.ParentTask
	require.NoError(t, f.db.First(&updated, f.taskNoPub.ID).Error)
	assert.Equal(t, "B2改名任务", updated.Title, "编辑必须真的落库，否则本测试证明不了什么")

	row := f.reload(t, c.ID)
	assert.Nil(t, row.PublicRevokedAt, "从未申请的提交不该被盖上撤回时间")
	assert.False(t, row.IsPublicRequested)
}

// 与公开无关的编辑不得刷新已有撤回时间：否则审计时间每次改标题都会漂移。
// （「从未申请不被盖章」已由 revokePublicForTask 的 WHERE 收窄兜住，
//
//	这里守的是「已经撤回过的行」不被反复覆盖。）
func TestEditingTaskDoesNotRefreshRevokedAt(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)
	f.requestAndApprove(t, &f.taskPub, c.ID)

	require.NoError(t, f.svcT.UpdateTask(context.Background(), f.tc, f.taskPub.ID, TeacherTaskInput{
		Title: f.taskPub.Title, LearningLink: "B2学习关联", AllowPublic: ptrTo(false),
	}))
	require.NotNil(t, f.reload(t, c.ID).PublicRevokedAt)

	pinned := time.Date(2026, 1, 2, 3, 4, 5, 0, shanghaiLoc())
	require.NoError(t, f.db.Model(&domain.TaskCheckIn{}).Where("id = ?", c.ID).
		Update("public_revoked_at", pinned).Error)

	// 任务此时已经不公开，再改标题不该再走一遍撤销。
	require.NoError(t, f.svcT.UpdateTask(context.Background(), f.tc, f.taskPub.ID, TeacherTaskInput{
		Title: "B2再次改名", LearningLink: "B2学习关联",
	}))

	got := f.reload(t, c.ID).PublicRevokedAt
	require.NotNil(t, got)
	assert.True(t, pinned.Equal(*got), "无关编辑不得刷新撤回时间：want %v, got %v", pinned, *got)
}

func TestPublicWallCrossClassIsolation(t *testing.T) {
	f := setupB2(t)
	mine := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)
	other := f.checkin(t, &f.taskOther, testB2StudentOther, testB2ParentOther)

	f.requestAndApprove(t, &f.taskPub, mine.ID)
	_, err := f.svcP.RequestPublicCheckin(context.Background(), f.pcOther, other.ID)
	require.NoError(t, err)
	_, err = f.svcT.ReviewPublicCheckin(context.Background(),
		&TeacherPortalContext{UserID: testB2Teacher, KindergartenID: testB2KG, ClassID: testB2ClassOther},
		f.taskOther.ID, other.ID, true, true)
	require.NoError(t, err)

	classA := f.wallIDs(t, f.pc)
	classB := f.wallIDs(t, f.pcOther)
	assert.Contains(t, classA, mine.ID)
	assert.NotContains(t, classA, other.ID, "别的班的公开内容不得出现在本班墙上")
	assert.Contains(t, classB, other.ID)
	assert.NotContains(t, classB, mine.ID)
}

func TestPublicWallOnlyReturnsOwnStudentContent(t *testing.T) {
	f := setupB2(t)
	mine := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)
	peers := f.checkin(t, &f.taskPub, testB2StudentPeer, testB2ParentPeer)

	f.requestAndApprove(t, &f.taskPub, mine.ID)
	// 同班另一家的提交保持私密。
	ids := f.wallIDs(t, f.pcPeer)
	assert.Contains(t, ids, mine.ID)
	assert.NotContains(t, ids, peers.ID, "未申请的提交不得出现在墙上")

	out, err := f.svcP.PublicWall(context.Background(), f.pcPeer, 1, 50)
	require.NoError(t, err)
	list := out["list"].([]map[string]any)
	require.Len(t, list, 1)
	student := list[0]["student"].(map[string]any)
	assert.Equal(t, testB2Student, student["id"], "每条只带提交者孩子，不带别人")
	assert.NotContains(t, student, "phone")
}

func TestPublicWallRequiresScopeConfirmed(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)

	// 直接写一条「有批准时间但没有确认」的行：这是「含其他孩子内容却被批准」的
	// 唯一可能残留形态，公开墙必须自己挡住，不能只依赖老师端校验。
	now := time.Now()
	require.NoError(t, f.db.Model(&domain.TaskCheckIn{}).Where("id = ?", c.ID).Updates(map[string]any{
		"is_public_requested":    true,
		"public_approved_at":     now,
		"public_approved_by":     testB2Teacher,
		"public_scope_confirmed": false,
	}).Error)
	assert.NotContains(t, f.wallIDs(t, f.pcPeer), c.ID, "未确认内容边界的记录不得上墙")
	assert.Equal(t, TaskPublicPending, publicCheckinStatus(ptrCheckin(f.reload(t, c.ID))))

	require.NoError(t, f.db.Model(&domain.TaskCheckIn{}).Where("id = ?", c.ID).
		Update("public_scope_confirmed", true).Error)
	assert.Contains(t, f.wallIDs(t, f.pcPeer), c.ID)
}

func TestPublicWallNeverShowsIndividualTask(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)
	f.requestAndApprove(t, &f.taskPub, c.ID)
	require.Contains(t, f.wallIDs(t, f.pcPeer), c.ID)

	// 模拟 2.9 回归：任务被改成个体任务却残留 allow_public=true 与旧批准。
	// 公开墙必须自己排除个体任务，否则「个体任务不可公开」只剩前端约束。
	require.NoError(t, f.db.Model(&domain.ParentTask{}).Where("id = ?", f.taskPub.ID).
		Updates(map[string]any{"target_student_id": testB2Student, "allow_public": true}).Error)
	assert.NotContains(t, f.wallIDs(t, f.pcPeer), c.ID)
}

func ptrCheckin(c domain.TaskCheckIn) *domain.TaskCheckIn { return &c }

// ---------- 重新申请必须重新审核 ----------

func TestReRequestRequiresReapproval(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)
	f.requestAndApprove(t, &f.taskPub, c.ID)
	require.Contains(t, f.wallIDs(t, f.pcPeer), c.ID)

	_, err := f.svcP.CancelPublicCheckin(context.Background(), f.pc, c.ID)
	require.NoError(t, err)

	// 再次申请：回到待审核，不能沿用旧批准或旧确认。
	out, err := f.svcP.RequestPublicCheckin(context.Background(), f.pc, c.ID)
	require.NoError(t, err)
	assert.Equal(t, TaskPublicPending, out["public_status"])
	assert.Equal(t, false, out["public_visible"])
	assert.NotContains(t, f.wallIDs(t, f.pcPeer), c.ID)

	row := f.reload(t, c.ID)
	assert.Nil(t, row.PublicApprovedAt)
	assert.False(t, row.PublicScopeConfirmed)
	assert.Nil(t, row.PublicRevokedAt)

	// 未重新确认就不能批准。
	_, err = f.svcT.ReviewPublicCheckin(context.Background(), f.tc, f.taskPub.ID, c.ID, true, false)
	assert.ErrorIs(t, err, ErrTaskPublicScopeUnconfirmed)

	// 重新确认并批准后才可见。
	_, err = f.svcT.ReviewPublicCheckin(context.Background(), f.tc, f.taskPub.ID, c.ID, true, true)
	require.NoError(t, err)
	assert.Contains(t, f.wallIDs(t, f.pcPeer), c.ID)
}

func TestTeacherApproveAfterParentCancelConflicts(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)
	_, err := f.svcP.RequestPublicCheckin(context.Background(), f.pc, c.ID)
	require.NoError(t, err)
	// 家长在老师点确认前取消：批准必须失败，不能把已经撤回的记录重新放上墙。
	_, err = f.svcP.CancelPublicCheckin(context.Background(), f.pc, c.ID)
	require.NoError(t, err)

	_, err = f.svcT.ReviewPublicCheckin(context.Background(), f.tc, f.taskPub.ID, c.ID, true, true)
	assert.ErrorIs(t, err, ErrTaskPublicStateConflict)
	assert.NotContains(t, f.wallIDs(t, f.pcPeer), c.ID)
}

// 关闭公开开关必须走真实编辑路径把内容撤下（spec 0143）。
func TestTeacherUpdateTaskClosingPublicRevokes(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)
	// 同任务、另一家的提交：从未申请过公开，关闭开关时不该被盖上撤回时间。
	peer := f.checkin(t, &f.taskPub, testB2StudentPeer, testB2ParentPeer)
	f.requestAndApprove(t, &f.taskPub, c.ID)
	require.Contains(t, f.wallIDs(t, f.pcPeer), c.ID)

	require.NoError(t, f.svcT.UpdateTask(context.Background(), f.tc, f.taskPub.ID, TeacherTaskInput{
		Title: f.taskPub.Title, LearningLink: "B2学习关联", AllowPublic: ptrTo(false),
	}))

	var updated domain.ParentTask
	require.NoError(t, f.db.First(&updated, f.taskPub.ID).Error)
	assert.False(t, updated.AllowPublic, "编辑必须真的关闭公开开关")

	row := f.reload(t, c.ID)
	assert.False(t, row.IsPublicRequested)
	assert.False(t, row.PublicScopeConfirmed)
	assert.Nil(t, row.PublicApprovedAt)
	assert.Zero(t, row.PublicApprovedBy)
	require.NotNil(t, row.PublicRevokedAt, "关闭公开必须批量写撤回时间")
	assert.NotContains(t, f.wallIDs(t, f.pcPeer), c.ID)

	peerRow := f.reload(t, peer.ID)
	assert.Nil(t, peerRow.PublicRevokedAt, "从未申请的提交不该被洗成已撤回")
	assert.Equal(t, TaskPublicNone, publicCheckinStatus(&peerRow))
}

// 老师端 public_visible 必须和公开墙同口径：孩子转班后不能再显示「已公开」。
func TestTeacherCheckinsPublicVisibleFollowsStudentClass(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)
	f.requestAndApprove(t, &f.taskPub, c.ID)

	first := func() map[string]any {
		out, err := f.svcT.TaskCheckins(context.Background(), f.tc, f.taskPub.ID, "")
		require.NoError(t, err)
		rows := out["list"].([]map[string]any)
		require.Len(t, rows, 1)
		return rows[0]
	}
	assert.Equal(t, true, first()["public_visible"])

	require.NoError(t, f.db.Model(&domain.Student{}).Where("id = ?", testB2Student).
		Update("class_id", testB2ClassOther).Error)
	assert.Equal(t, false, first()["public_visible"], "转班后墙上已隐藏，老师端不能仍显示已公开")
}

// 孩子已经不在本班时，批准公开只会造出一个永远不上墙的「已公开」状态。
func TestTeacherApproveRejectedAfterStudentLeavesClass(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)
	_, err := f.svcP.RequestPublicCheckin(context.Background(), f.pc, c.ID)
	require.NoError(t, err)

	require.NoError(t, f.db.Model(&domain.Student{}).Where("id = ?", testB2Student).
		Update("class_id", testB2ClassOther).Error)

	_, err = f.svcT.ReviewPublicCheckin(context.Background(), f.tc, f.taskPub.ID, c.ID, true, true)
	assert.ErrorIs(t, err, ErrTaskPublicNotAllowed)
	assert.Nil(t, f.reload(t, c.ID).PublicApprovedAt)

	// 拒绝/撤回仍可用，能把待审核状态清掉。
	out, err := f.svcT.ReviewPublicCheckin(context.Background(), f.tc, f.taskPub.ID, c.ID, false, false)
	require.NoError(t, err)
	assert.Equal(t, TaskPublicRevoked, out["public_status"])
}

// ---------- 4.4 管理端撤回 ----------

func TestAdminRevokeHidesContent(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)
	f.requestAndApprove(t, &f.taskPub, c.ID)
	require.Contains(t, f.wallIDs(t, f.pcPeer), c.ID)

	out, err := f.svcA.RevokePublicCheckin(context.Background(), testB2KG, testB2Teacher, c.ID, "127.0.0.1", "test")
	require.NoError(t, err)
	assert.Equal(t, TaskPublicRevoked, out["public_status"])
	assert.NotContains(t, f.wallIDs(t, f.pcPeer), c.ID)

	row := f.reload(t, c.ID)
	assert.False(t, row.IsPublicRequested)
	assert.False(t, row.PublicScopeConfirmed)
	assert.Nil(t, row.PublicApprovedAt)
	assert.Zero(t, row.PublicApprovedBy)
	require.NotNil(t, row.PublicRevokedAt)

	// 撤回要留操作日志（4.7 的可追溯要求）。
	var n int64
	require.NoError(t, f.db.Model(&domain.OperationLog{}).
		Where("target_id = ? AND action = ? AND module = ?", c.ID, "revoke_public", "tasks").Count(&n).Error)
	assert.Equal(t, int64(1), n)
}

func TestAdminRevokeForeignKindergartenIsNotFound(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)
	f.requestAndApprove(t, &f.taskPub, c.ID)

	_, err := f.svcA.RevokePublicCheckin(context.Background(), testB2KG+999, testB2Teacher, c.ID, "", "")
	assert.Error(t, err)
	assert.True(t, IsTaskNotFound(err))
	// 未撤回：仍然是公开可见。
	assert.Equal(t, TaskPublicApproved, publicCheckinStatus(func() *domain.TaskCheckIn {
		row := f.reload(t, c.ID)
		return &row
	}()))
}

func TestAdminRevokeNoopDoesNotWriteLog(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)

	out, err := f.svcA.RevokePublicCheckin(context.Background(), testB2KG, testB2Teacher, c.ID, "", "")
	require.NoError(t, err)
	assert.Equal(t, TaskPublicNone, out["public_status"])

	var n int64
	require.NoError(t, f.db.Model(&domain.OperationLog{}).
		Where("target_id = ? AND action = ?", c.ID, "revoke_public").Count(&n).Error)
	assert.Zero(t, n, "空操作不得记成一次撤回")
	assert.Nil(t, f.reload(t, c.ID).PublicRevokedAt)
}

// ---------- 提交者学生记录消失后的边界 ----------

// 学生记录被（软）删除后，公开墙的 `st.class_id = t.class_id` 必然不成立，
// 内容已经不可能出现在墙上；老师端不能仍报 public_visible=true。
func TestTeacherCheckinsPublicVisibleHidesDeletedStudent(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)
	f.requestAndApprove(t, &f.taskPub, c.ID)
	require.Contains(t, f.wallIDs(t, f.pcPeer), c.ID)

	require.NoError(t, f.db.Delete(&domain.Student{}, testB2Student).Error)
	assert.NotContains(t, f.wallIDs(t, f.pcPeer), c.ID, "学生记录删除后墙上不再有这条")

	out, err := f.svcT.TaskCheckins(context.Background(), f.tc, f.taskPub.ID, "")
	require.NoError(t, err)
	rows := out["list"].([]map[string]any)
	require.Len(t, rows, 1)
	assert.Equal(t, false, rows[0]["public_visible"],
		"拿不到提交者班级时，public_visible 必须与公开墙一致（false）")
}

// 提交者学生记录已删除时批准，只会造出一个永远不上墙的「已公开」。
func TestTeacherApproveRejectedWhenStudentDeleted(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)
	_, err := f.svcP.RequestPublicCheckin(context.Background(), f.pc, c.ID)
	require.NoError(t, err)

	require.NoError(t, f.db.Delete(&domain.Student{}, testB2Student).Error)

	_, err = f.svcT.ReviewPublicCheckin(context.Background(), f.tc, f.taskPub.ID, c.ID, true, true)
	assert.ErrorIs(t, err, ErrTaskPublicNotAllowed)
	assert.Nil(t, f.reload(t, c.ID).PublicApprovedAt)
}

// 契约要求「撤回永远可用」：学生记录已删除不是拒绝撤回的理由。
func TestTeacherRevokeWorksAfterStudentDeleted(t *testing.T) {
	f := setupB2(t)
	c := f.checkin(t, &f.taskPub, testB2Student, testB2Parent)
	f.requestAndApprove(t, &f.taskPub, c.ID)

	require.NoError(t, f.db.Delete(&domain.Student{}, testB2Student).Error)

	out, err := f.svcT.ReviewPublicCheckin(context.Background(), f.tc, f.taskPub.ID, c.ID, false, false)
	require.NoError(t, err, "撤回必须永远可用，不能因为学生记录被删除而 422")
	assert.Equal(t, TaskPublicRevoked, out["public_status"])
	assert.Equal(t, false, out["public_visible"])

	row := f.reload(t, c.ID)
	assert.False(t, row.IsPublicRequested)
	assert.Nil(t, row.PublicApprovedAt)
	require.NotNil(t, row.PublicRevokedAt)
}
