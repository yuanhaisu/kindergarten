package service

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/domain"
)

// 7.1 多班活动端点的真实库用例。

func (f *c2Fixture) loadTaskC2(t *testing.T, id uint64) domain.ParentTask {
	t.Helper()
	var tk domain.ParentTask
	require.NoError(t, f.db.Unscoped().Where("id = ?", id).First(&tk).Error)
	return tk
}

// updateReqOf 按「客户端把表单原样提交」的形状构造 PUT 请求。
func (f *c2Fixture) updateReqOf(tk domain.ParentTask) UpdateTaskReq {
	return UpdateTaskReq{
		Title:        tk.Title,
		Description:  ptrTo(tk.Description),
		LearningLink: tk.LearningLink,
		ClassID:      tk.ClassID,
		Frequency:    string(tk.Frequency),
		StartDate:    tk.StartDate.Format("2006-01-02"),
		EndDate:      tk.EndDate.Format("2006-01-02"),
	}
}

func TestActivityCreateMakesOneTaskPerClass(t *testing.T) {
	f := setupC2(t)
	resp, err := f.svc.CreateActivity(f.ctx, testC2KG, testC2Admin, f.activityReq(testC2ClassB, testC2Class), "", "")
	require.NoError(t, err)

	activityID, _ := resp["activity_id"].(string)
	require.NotEmpty(t, activityID)
	assert.Regexp(t, "^act_[0-9]+_[0-9a-f]{24}$", activityID, "服务端生成、不可猜的活动 id")

	tasks := f.tasksOfActivity(t, activityID)
	require.Len(t, tasks, 2)
	// class_ids 固定按 id 升序处理：建任务顺序稳定，锁序与批量编辑/删除一致。
	assert.Equal(t, testC2Class, tasks[0].ClassID)
	assert.Equal(t, testC2ClassB, tasks[1].ClassID)
	for _, tk := range tasks {
		assert.Equal(t, "C2春季阅读周", tk.Title)
		assert.Equal(t, "配合本周春天主题", tk.LearningLink)
		assert.Equal(t, domain.TaskRitualSticker, tk.RecognitionRitual)
		assert.Equal(t, domain.TaskStarScopeTask, tk.StarScope)
		assert.Equal(t, uint64(0), tk.TargetStudentID, "活动任务没有个体目标")
		assert.Equal(t, int8(1), tk.Status)
	}
	// 两个班各一个家长 → 通知 2 条（每条任务 1 个家长）。
	assert.Equal(t, 2, resp["notified_count"])
	assert.Equal(t, 0, resp["notify_failed_count"])
}

func TestActivityCreateRejectsBadClassIDs(t *testing.T) {
	f := setupC2(t)

	_, err := f.svc.CreateActivity(f.ctx, testC2KG, testC2Admin, f.activityReq(), "", "")
	require.ErrorIs(t, err, ErrTaskActivityClassesInvalid, "空 class_ids 必须失败，不能建出 0 条任务的活动")

	_, err = f.svc.CreateActivity(f.ctx, testC2KG, testC2Admin, f.activityReq(testC2Class, testC2Class), "", "")
	require.ErrorIs(t, err, ErrTaskActivityDuplicateClass, "重复班级按校验失败处理，不静默去重")

	// 跨园所班级：整组拒绝，一条也不建。
	_, err = f.svc.CreateActivity(f.ctx, testC2KG, testC2Admin, f.activityReq(testC2Class, testC2ForeignClass), "", "")
	require.ErrorIs(t, err, ErrTaskActivityClassesInvalid)
	assert.Equal(t, int64(0), f.countTasksInKG(t))
}

func TestActivityCreateRejectsTooManyClasses(t *testing.T) {
	f := setupC2(t)
	ids := make([]uint64, 0, taskActivityMaxClasses+1)
	for i := 0; i <= taskActivityMaxClasses; i++ {
		ids = append(ids, uint64(993000+i))
	}
	_, err := f.svc.CreateActivity(f.ctx, testC2KG, testC2Admin, f.activityReq(ids...), "", "")
	require.ErrorIs(t, err, ErrTaskActivityTooManyClasses, "上限判定必须早于归属查询")
}

func TestActivityCreateRollsBackWholeGroupWhenClassJarMissing(t *testing.T) {
	f := setupC2(t)
	// 只给一班配班级罐，二班没配：二班在事务中途失败，一班已建的任务必须一起回滚。
	_, err := f.svc.SetClassStarJar(f.ctx, testC2KG, testC2Class, testC2Admin, 20, "", "")
	require.NoError(t, err)

	req := f.activityReq(testC2Class, testC2ClassB)
	req.StarScope = "class"
	_, err = f.svc.CreateActivity(f.ctx, testC2KG, testC2Admin, req, "", "")
	require.ErrorIs(t, err, errTaskStarJarMissing)
	assert.Equal(t, int64(0), f.countTasksInKG(t), "任一班级失败必须整组回滚，不留半组任务")
}

func TestActivityCreateRejectsClientActivityID(t *testing.T) {
	f := setupC2(t)
	req := f.activityReq(testC2Class)
	req.ActivityID = "act_injected"
	_, err := f.svc.CreateActivity(f.ctx, testC2KG, testC2Admin, req, "", "")
	require.ErrorIs(t, err, ErrTaskActivityIDForbidden)
	assert.Equal(t, int64(0), f.countTasksInKG(t))
}

func TestActivityUpdateOnlySafeFields(t *testing.T) {
	f := setupC2(t)
	resp, err := f.svc.CreateActivity(f.ctx, testC2KG, testC2Admin, f.activityReq(testC2Class, testC2ClassB), "", "")
	require.NoError(t, err)
	activityID := resp["activity_id"].(string)

	out, err := f.svc.UpdateActivity(f.ctx, testC2KG, testC2Admin, activityID,
		UpdateActivityReq{Title: ptrTo("C2春季阅读周-改"), Description: ptrTo("新的说明")}, "", "")
	require.NoError(t, err)
	assert.Equal(t, 2, out["updated_tasks"])

	for _, tk := range f.tasksOfActivity(t, activityID) {
		assert.Equal(t, "C2春季阅读周-改", tk.Title)
		assert.Equal(t, "新的说明", tk.Description)
	}

	// 结构字段：显式传入即拒绝（即使取值与当前相同），不能静默忽略。
	cases := map[string]UpdateActivityReq{
		"frequency":   {Frequency: "weekly"},
		"start_date":  {StartDate: f.date(0)},
		"end_date":    {EndDate: f.date(6)},
		"class_ids":   {ClassIDs: []uint64{testC2Class}},
		"target":      {TargetStudentID: ptrTo(uint64(1))},
		"ritual":      {RecognitionRitual: "sticker"},
		"star_scope":  {StarScope: "task"},
		"star_goal":   {StarGoal: ptrTo(5)},
		"activity_id": {ActivityID: activityID},
	}
	for name, req := range cases {
		_, err := f.svc.UpdateActivity(f.ctx, testC2KG, testC2Admin, activityID, req, "", "")
		require.ErrorIs(t, err, ErrTaskActivityFieldForbidden, "批量编辑不得改结构字段: %s", name)
	}
}

func TestActivityUpdateClosingPublicRevokesAllTasks(t *testing.T) {
	f := setupC2(t)
	req := f.activityReq(testC2Class, testC2ClassB)
	req.AllowPublic = true
	resp, err := f.svc.CreateActivity(f.ctx, testC2KG, testC2Admin, req, "", "")
	require.NoError(t, err)
	activityID := resp["activity_id"].(string)
	tasks := f.tasksOfActivity(t, activityID)
	require.Len(t, tasks, 2)

	// 两条任务各有一条已批准的公开提交。
	now := time.Now()
	students := []uint64{testC2Student, testC2StudentB}
	for i, tk := range tasks {
		require.NoError(t, f.db.Create(&domain.TaskCheckIn{
			TaskID: tk.ID, StudentID: students[i], UserID: testC2Parent,
			Date: f.today, IsPublicRequested: true, PublicScopeConfirmed: true,
			PublicApprovedAt: &now, PublicApprovedBy: testC2Admin,
		}).Error)
	}

	_, err = f.svc.UpdateActivity(f.ctx, testC2KG, testC2Admin, activityID,
		UpdateActivityReq{AllowPublic: ptrTo(false)}, "", "")
	require.NoError(t, err)

	var checkins []domain.TaskCheckIn
	require.NoError(t, f.db.Where("task_id IN ?", []uint64{tasks[0].ID, tasks[1].ID}).Find(&checkins).Error)
	require.Len(t, checkins, 2)
	for _, c := range checkins {
		assert.False(t, c.IsPublicRequested)
		assert.False(t, c.PublicScopeConfirmed)
		assert.Nil(t, c.PublicApprovedAt)
		assert.Equal(t, uint64(0), c.PublicApprovedBy)
		assert.NotNil(t, c.PublicRevokedAt, "关闭公开必须在同一事务里撤销全部任务")
	}
	for _, tk := range f.tasksOfActivity(t, activityID) {
		assert.False(t, tk.AllowPublic)
	}
}

func TestActivityDeleteCascadesWholeGroup(t *testing.T) {
	f := setupC2(t)
	resp, err := f.svc.CreateActivity(f.ctx, testC2KG, testC2Admin, f.activityReq(testC2Class, testC2ClassB), "", "")
	require.NoError(t, err)
	activityID := resp["activity_id"].(string)
	tasks := f.tasksOfActivity(t, activityID)
	require.Len(t, tasks, 2)
	require.NoError(t, f.db.Create(&domain.TaskCheckIn{
		TaskID: tasks[0].ID, StudentID: testC2Student, UserID: testC2Parent, Date: f.today,
	}).Error)
	// 另一个活动不受影响。
	other, err := f.svc.CreateActivity(f.ctx, testC2KG, testC2Admin, f.activityReq(testC2Class), "", "")
	require.NoError(t, err)
	otherID := other["activity_id"].(string)

	out, err := f.svc.DeleteActivity(f.ctx, testC2KG, testC2Admin, activityID, "", "")
	require.NoError(t, err)
	assert.Equal(t, 2, out["deleted_tasks"])
	assert.Empty(t, f.tasksOfActivity(t, activityID))

	var n int64
	require.NoError(t, f.db.Model(&domain.TaskCheckIn{}).Where("task_id = ?", tasks[0].ID).Count(&n).Error)
	assert.Zero(t, n, "删除活动必须级联软删提交")
	assert.Len(t, f.tasksOfActivity(t, otherID), 1, "其它活动不受影响")

	_, err = f.svc.DeleteActivity(f.ctx, testC2KG, testC2Admin, activityID, "", "")
	require.ErrorIs(t, err, ErrTaskActivityNotFound, "重复删除按 404，不是 200 空操作")
}

func TestActivityTaskRejectsSingleDelete(t *testing.T) {
	f := setupC2(t)
	resp, err := f.svc.CreateActivity(f.ctx, testC2KG, testC2Admin, f.activityReq(testC2Class, testC2ClassB), "", "")
	require.NoError(t, err)
	activityID := resp["activity_id"].(string)
	tasks := f.tasksOfActivity(t, activityID)
	require.Len(t, tasks, 2)

	err = f.svc.DeleteTask(f.ctx, testC2KG, testC2Admin, tasks[0].ID, "", "")
	require.ErrorIs(t, err, ErrTaskActivityDeleteForbidden)
	assert.Len(t, f.tasksOfActivity(t, activityID), 2, "被拒的单删不得留下半个删除")
}

func TestActivityTaskRejectsSingleStructuralUpdate(t *testing.T) {
	f := setupC2(t)
	resp, err := f.svc.CreateActivity(f.ctx, testC2KG, testC2Admin, f.activityReq(testC2Class), "", "")
	require.NoError(t, err)
	task := f.tasksOfActivity(t, resp["activity_id"].(string))[0]

	// 没有提交，结构锁不生效；活动任务仍不允许单条改结构。
	req := f.updateReqOf(task)
	req.StarGoal = ptrTo(9)
	err = f.svc.UpdateTask(f.ctx, testC2KG, testC2Admin, task.ID, req, "", "")
	require.ErrorIs(t, err, ErrTaskActivityEditForbidden)
	assert.Equal(t, 0, f.loadTaskC2(t, task.ID).StarGoal, "被拒的更新不得落库")

	// 安全字段仍然允许。
	req = f.updateReqOf(task)
	req.Title = "单条改标题"
	require.NoError(t, f.svc.UpdateTask(f.ctx, testC2KG, testC2Admin, task.ID, req, "", ""))
	assert.Equal(t, "单条改标题", f.loadTaskC2(t, task.ID).Title)
}

// --- 老师端也必须守住活动不变量（spec：活动任务 MUST NOT 走单条删除或 unsafe 单条编辑）---

func (f *c2Fixture) teacherCtx() *TeacherPortalContext {
	return &TeacherPortalContext{UserID: testC2Admin, KindergartenID: testC2KG, ClassID: testC2Class}
}

func (f *c2Fixture) teacherInput(tk domain.ParentTask) TeacherTaskInput {
	return TeacherTaskInput{
		Title:        tk.Title,
		LearningLink: tk.LearningLink,
		Frequency:    string(tk.Frequency),
		StartDate:    tk.StartDate.Format("2006-01-02"),
		EndDate:      tk.EndDate.Format("2006-01-02"),
	}
}

func TestActivityTaskRejectsTeacherDelete(t *testing.T) {
	f := setupC2(t)
	resp, err := f.svc.CreateActivity(f.ctx, testC2KG, testC2Admin, f.activityReq(testC2Class, testC2ClassB), "", "")
	require.NoError(t, err)
	activityID := resp["activity_id"].(string)
	tasks := f.tasksOfActivity(t, activityID)
	require.Len(t, tasks, 2)

	err = f.svcT.DeleteTask(f.ctx, f.teacherCtx(), tasks[0].ID)
	require.ErrorIs(t, err, ErrTaskActivityDeleteForbidden,
		"老师单删活动任务会让活动分组与统计聚合悬空")
	assert.Len(t, f.tasksOfActivity(t, activityID), 2)
}

func TestActivityTaskRejectsTeacherStructuralUpdate(t *testing.T) {
	f := setupC2(t)
	resp, err := f.svc.CreateActivity(f.ctx, testC2KG, testC2Admin, f.activityReq(testC2Class), "", "")
	require.NoError(t, err)
	task := f.tasksOfActivity(t, resp["activity_id"].(string))[0]

	// 改成个体任务：没有提交，结构锁不生效——活动任务的禁令必须独立存在。
	in := f.teacherInput(task)
	in.TargetStudentID = testC2Student
	err = f.svcT.UpdateTask(f.ctx, f.teacherCtx(), task.ID, in)
	require.ErrorIs(t, err, ErrTaskActivityEditForbidden)
	fresh := f.loadTaskC2(t, task.ID)
	assert.Equal(t, uint64(0), fresh.TargetStudentID, "被拒的更新不得落库")
	assert.NotEmpty(t, fresh.ActivityID, "活动归属不得被清掉")

	// 改频率同样拒绝。
	in = f.teacherInput(task)
	in.Frequency = string(domain.TaskFrequencyWeekly)
	require.ErrorIs(t, f.svcT.UpdateTask(f.ctx, f.teacherCtx(), task.ID, in), ErrTaskActivityEditForbidden)

	// 安全字段仍然允许。
	in = f.teacherInput(task)
	in.Title = "老师改活动任务标题"
	require.NoError(t, f.svcT.UpdateTask(f.ctx, f.teacherCtx(), task.ID, in))
	assert.Equal(t, "老师改活动任务标题", f.loadTaskC2(t, task.ID).Title)
}

func TestActivityTitleLengthBoundary(t *testing.T) {
	f := setupC2(t)
	// 列宽按**字符**算（varchar(200)）：200 个汉字合法，201 个必须 422 而不是落库报错。
	ok := f.activityReq(testC2Class)
	ok.Title = strings.Repeat("读", taskTitleMaxLen)
	resp, err := f.svc.CreateActivity(f.ctx, testC2KG, testC2Admin, ok, "", "")
	require.NoError(t, err, "恰好 200 个字符应当允许")

	tooLong := f.activityReq(testC2Class)
	tooLong.Title = strings.Repeat("读", taskTitleMaxLen+1)
	_, err = f.svc.CreateActivity(f.ctx, testC2KG, testC2Admin, tooLong, "", "")
	require.ErrorIs(t, err, ErrTaskTitleTooLong, "超长标题必须是校验错误，不能变成 MySQL 1406 / 500")

	activityID := resp["activity_id"].(string)
	upd := UpdateActivityReq{Title: ptrTo(strings.Repeat("读", taskTitleMaxLen+1))}
	_, err = f.svc.UpdateActivity(f.ctx, testC2KG, testC2Admin, activityID, upd, "", "")
	require.ErrorIs(t, err, ErrTaskTitleTooLong)
	assert.Equal(t, strings.Repeat("读", taskTitleMaxLen), f.tasksOfActivity(t, activityID)[0].Title,
		"被拒的批量编辑不得改掉标题")
}

func TestTeacherTaskTitleLengthBoundary(t *testing.T) {
	f := setupC2(t)
	// TeacherTaskInput 没有 binding：校验必须在 service，否则 201 个字符是 MySQL 1406 → 500。
	in := f.teacherInput(domain.ParentTask{Title: "占位", LearningLink: "学习关联"})
	in.Title = strings.Repeat("读", taskTitleMaxLen+1)
	_, err := f.svcT.CreateTask(f.ctx, f.teacherCtx(), in)
	require.ErrorIs(t, err, ErrTaskTitleTooLong)

	var n int64
	require.NoError(t, f.db.Model(&domain.ParentTask{}).Where("kindergarten_id = ?", testC2KG).Count(&n).Error)
	assert.Zero(t, n, "超长标题不得落库")

	out, err := f.svcT.CreateTask(f.ctx, f.teacherCtx(), f.teacherInput(domain.ParentTask{
		Title: strings.Repeat("读", taskTitleMaxLen), LearningLink: "学习关联",
	}))
	require.NoError(t, err)
	taskID := out["id"].(uint64)

	upd := f.teacherInput(f.loadTaskC2(t, taskID))
	upd.Title = strings.Repeat("读", taskTitleMaxLen+1)
	require.ErrorIs(t, f.svcT.UpdateTask(f.ctx, f.teacherCtx(), taskID, upd), ErrTaskTitleTooLong)
	assert.Equal(t, strings.Repeat("读", taskTitleMaxLen), f.loadTaskC2(t, taskID).Title)
}

func TestActivityUpdateEndsWholeGroup(t *testing.T) {
	f := setupC2(t)
	resp, err := f.svc.CreateActivity(f.ctx, testC2KG, testC2Admin, f.activityReq(testC2Class, testC2ClassB), "", "")
	require.NoError(t, err)
	activityID := resp["activity_id"].(string)
	tasks := f.tasksOfActivity(t, activityID)
	require.Len(t, tasks, 2)

	// 有提交也不影响结束：status 不是结构字段（C-1 口径），不能被结构锁挡住。
	students := []uint64{testC2Student, testC2StudentB}
	for i, tk := range tasks {
		require.NoError(t, f.db.Create(&domain.TaskCheckIn{
			TaskID: tk.ID, StudentID: students[i], UserID: testC2Parent, Date: f.today,
		}).Error)
	}

	out, err := f.svc.UpdateActivity(f.ctx, testC2KG, testC2Admin, activityID,
		UpdateActivityReq{Status: ptrTo(int8(0))}, "", "")
	require.NoError(t, err)
	assert.Equal(t, 2, out["updated_tasks"])
	for _, tk := range f.tasksOfActivity(t, activityID) {
		assert.Equal(t, int8(0), f.loadTaskC2(t, tk.ID).Status, "整组结束必须一起生效")
	}

	// 非法状态值拒绝，且不改任何一条。
	_, err = f.svc.UpdateActivity(f.ctx, testC2KG, testC2Admin, activityID,
		UpdateActivityReq{Status: ptrTo(int8(2))}, "", "")
	require.EqualError(t, err, "BAD_STATUS")
	for _, tk := range f.tasksOfActivity(t, activityID) {
		assert.Equal(t, int8(0), f.loadTaskC2(t, tk.ID).Status)
	}
}

func TestActivityTaskSingleUpdateStillRejectsStatus(t *testing.T) {
	f := setupC2(t)
	resp, err := f.svc.CreateActivity(f.ctx, testC2KG, testC2Admin, f.activityReq(testC2Class), "", "")
	require.NoError(t, err)
	task := f.tasksOfActivity(t, resp["activity_id"].(string))[0]

	// 单条改状态会让活动的班级间状态不一致：只能整组结束。
	req := f.updateReqOf(task)
	req.Status = ptrTo(int8(0))
	err = f.svc.UpdateTask(f.ctx, testC2KG, testC2Admin, task.ID, req, "", "")
	require.ErrorIs(t, err, ErrTaskActivityEditForbidden)
	assert.Equal(t, int8(1), f.loadTaskC2(t, task.ID).Status)
}
