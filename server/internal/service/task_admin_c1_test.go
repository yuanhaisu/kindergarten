package service

import (
	"context"
	"errors"
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

// C-1（7.0 管理端建/改任务新字段与共享规则、3.8/7.5 班级罐、学期 term、8.3 契约）回归。
//
// 断言的是库里的真实状态与真实查询结果：term 只按当前学期匹配、字段不被丢弃、
// 结构锁、关闭公开的撤销——这些用内存桩都测不出来。
//
// 运行约束：
//   - 夹具固定 ID（992731-992736、992974），清理全部用 Unscoped；不要并发跑两个 go test 进程。
//   - 测试直连开发库，跑前确认 :8080 无进程占用同一批表（memory: running-backend-steals-queue-jobs）。
//
// 本文件**不**覆盖：HTTP handler 的错误映射（由 handler 层与真实网关冒烟覆盖）、
// 通知的实际投递失败路径（B3 已覆盖同一条 sendMessageToRecipients）。

const (
	testC1KG      uint64 = 992731
	testC1Class   uint64 = 992732 // 有学年 + academic_years 配置
	testC1ClassB  uint64 = 992733 // 有学年、无配置（自然学期兜底）
	testC1ClassC  uint64 = 992974 // 无学年（自然年兜底）
	testC1Admin   uint64 = 992734
	testC1Parent  uint64 = 992735
	testC1Student uint64 = 992736
)

// ptrTo 便于在测试里表达「显式提供了这个可选字段」。
func ptrTo[T any](v T) *T { return &v }

type c1Fixture struct {
	db    *gorm.DB
	svc   *TaskService
	svcT  *TeacherPortalService
	ctx   context.Context
	today time.Time
	clean func()
}

func setupC1(t *testing.T) *c1Fixture {
	t.Helper()
	db := openTaskB1DB(t)
	loc := shanghaiLoc()
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	clean := func() {
		var taskIDs []uint64
		if err := db.Unscoped().Model(&domain.ParentTask{}).
			Where("kindergarten_id = ?", testC1KG).Pluck("id", &taskIDs).Error; err != nil {
			t.Logf("C1 夹具：查遗留任务失败: %v", err)
		}
		if len(taskIDs) > 0 {
			var respIDs []uint64
			if err := db.Unscoped().Model(&domain.TaskResponse{}).
				Where("task_id IN ?", taskIDs).Pluck("id", &respIDs).Error; err != nil {
				t.Logf("C1 夹具清理失败 responses(pluck): %v", err)
			}
			if len(respIDs) > 0 {
				if err := db.Unscoped().Where("response_id IN ?", respIDs).
					Delete(&domain.TaskResponseReaction{}).Error; err != nil {
					t.Logf("C1 夹具清理失败 reactions: %v", err)
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
					t.Logf("C1 夹具清理失败 %s: %v", step.name, step.err)
				}
			}
		}
		for _, step := range []struct {
			name string
			err  error
		}{
			{"messages", db.Unscoped().Where("user_id IN ?", []uint64{testC1Admin, testC1Parent}).Delete(&domain.Message{}).Error},
			{"operation_logs", db.Unscoped().Where("user_id IN ?", []uint64{testC1Admin, testC1Parent}).Delete(&domain.OperationLog{}).Error},
			{"star_jars", db.Unscoped().Where("kindergarten_id = ?", testC1KG).Delete(&domain.ClassStarJar{}).Error},
			{"student_parents", db.Unscoped().Where("student_id = ?", testC1Student).Delete(&domain.StudentParent{}).Error},
			{"students", db.Unscoped().Where("id = ?", testC1Student).Delete(&domain.Student{}).Error},
			{"classes", db.Unscoped().Where("id IN ?", []uint64{testC1Class, testC1ClassB, testC1ClassC}).Delete(&domain.Class{}).Error},
			{"academic_years", db.Unscoped().Where("kindergarten_id = ?", testC1KG).Delete(&domain.AcademicYear{}).Error},
			{"app_settings", db.Unscoped().Where("kindergarten_id = ?", testC1KG).Delete(&domain.KindergartenAppSetting{}).Error},
			{"users", db.Unscoped().Where("id IN ?", []uint64{testC1Admin, testC1Parent}).Delete(&domain.User{}).Error},
			{"kindergarten", db.Unscoped().Where("id = ?", testC1KG).Delete(&domain.Kindergarten{}).Error},
		} {
			if step.err != nil {
				t.Logf("C1 夹具清理失败 %s: %v", step.name, step.err)
			}
		}
	}
	clean()
	t.Cleanup(clean)

	require.NoError(t, db.Unscoped().Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: testC1KG}, Name: "测试园所-C1",
	}).Error)
	for _, u := range []struct {
		id   uint64
		name string
		ph   string
	}{
		{testC1Admin, "C1管理员", "13900009801"},
		{testC1Parent, "C1家长", "13900009802"},
	} {
		require.NoError(t, db.Unscoped().Create(&domain.User{
			BaseModel: domain.BaseModel{ID: u.id}, Nickname: u.name, Phone: u.ph, PasswordHash: "x",
		}).Error)
	}
	for _, c := range []struct {
		id   uint64
		ay   string
		name string
	}{
		{testC1Class, "2031-2032", "C1配置班"},
		{testC1ClassB, "2020-2021", "C1兜底班"},
		{testC1ClassC, "", "C1无学年班"},
	} {
		require.NoError(t, db.Unscoped().Create(&domain.Class{
			BaseModel: domain.BaseModel{ID: c.id}, KindergartenID: testC1KG,
			Name: c.name, Grade: "小班", AcademicYear: c.ay, Status: 1,
		}).Error)
	}
	require.NoError(t, db.Unscoped().Create(&domain.AcademicYear{
		KindergartenID:      testC1KG,
		Year:                "2031-2032",
		SemesterSpringStart: "2031-02-20",
		SemesterSpringEnd:   "2031-07-10",
		SemesterFallStart:   "2030-09-01",
		SemesterFallEnd:     "2031-01-20",
		IsCurrent:           true,
	}).Error)
	require.NoError(t, db.Unscoped().Create(&domain.KindergartenAppSetting{
		KindergartenID:  testC1KG,
		CurrentSemester: "spring",
		ExtraConfig:     "{}",
	}).Error)
	require.NoError(t, db.Unscoped().Create(&domain.Student{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testC1Student}},
		KindergartenID:  testC1KG, ClassID: testC1Class, Name: "C1孩子", Status: 1,
		Birthday:   time.Date(2021, 5, 1, 0, 0, 0, 0, loc),
		EnrollDate: time.Date(2024, 9, 1, 0, 0, 0, 0, loc),
	}).Error)
	require.NoError(t, db.Unscoped().Create(&domain.StudentParent{
		StudentID: testC1Student, UserID: testC1Parent, Relation: domain.RelationMother, IsPrimary: true,
	}).Error)

	return &c1Fixture{
		db: db,
		svc: NewTaskService(repository.NewTaskRepo(db), repository.NewOperationLogRepo(db),
			repository.NewMessageRepo(db), db),
		// 老师端 service 只用来验证共享下界：其余依赖为 nil，CreateTask 不触碰它们。
		svcT:  NewTeacherPortalService(db, nil, nil, nil, nil, nil, config.FaceConfig{}),
		ctx:   context.Background(),
		today: today,
		clean: clean,
	}
}

func (f *c1Fixture) date(offset int) string {
	return f.today.AddDate(0, 0, offset).Format("2006-01-02")
}

func (f *c1Fixture) createReq(classID uint64) CreateTaskReq {
	return CreateTaskReq{
		Title:        "C1任务",
		LearningLink: "延续本周班级主题",
		ClassID:      classID,
		Frequency:    string(domain.TaskFrequencyDaily),
		StartDate:    f.date(0),
		EndDate:      f.date(6),
	}
}

func (f *c1Fixture) loadTask(t *testing.T, id any) domain.ParentTask {
	t.Helper()
	var tk domain.ParentTask
	require.NoError(t, f.db.Unscoped().Where("id = ?", id).First(&tk).Error)
	return tk
}

// --- 学期 term ---

func TestClassStarJarTermUsesAcademicYearConfig(t *testing.T) {
	f := setupC1(t)
	term, err := classStarJarTermOf(f.ctx, f.db, testC1KG, testC1Class, time.Now())
	require.NoError(t, err)
	assert.Equal(t, "2031-2032-spring", term.Term)
	// 配置里 fall 在 spring 之前（2030-09-01..2031-01-20）：两学期必须无缝，
	// spring 从 fall 结束次日开始（寒假 1/21–2/19 的提交归 spring）。
	assert.Equal(t, "2031-01-21", term.Start.Format("2006-01-02"))
	assert.Equal(t, "2031-07-10", term.End.Format("2006-01-02"))
}

// 两学期 MUST 无缝：配置里 spring 与 fall 之间留了暑假空档时，fall 起点前移到
// spring 结束次日。否则暑假的提交不属于任何学期，切学期后永远不计入。
func TestClassStarJarTermSealsGapBetweenSemesters(t *testing.T) {
	f := setupC1(t)
	require.NoError(t, f.db.Unscoped().Create(&domain.AcademicYear{
		KindergartenID:      testC1KG,
		Year:                "2020-2021",
		SemesterSpringStart: "2021-02-16",
		SemesterSpringEnd:   "2021-07-10",
		SemesterFallStart:   "2021-09-01",
		SemesterFallEnd:     "2022-01-18",
	}).Error)

	// spring 是前一个学期，窗口保持配置值。
	spring, err := classStarJarTermOf(f.ctx, f.db, testC1KG, testC1ClassB, time.Now())
	require.NoError(t, err)
	assert.Equal(t, "2020-2021-spring", spring.Term)
	assert.Equal(t, "2021-02-16", spring.Start.Format("2006-01-02"))
	assert.Equal(t, "2021-07-10", spring.End.Format("2006-01-02"))

	require.NoError(t, f.db.Model(&domain.KindergartenAppSetting{}).
		Where("kindergarten_id = ?", testC1KG).Update("current_semester", "fall").Error)
	fall, err := classStarJarTermOf(f.ctx, f.db, testC1KG, testC1ClassB, time.Now())
	require.NoError(t, err)
	assert.Equal(t, "2020-2021-fall", fall.Term)
	assert.Equal(t, "2021-07-11", fall.Start.Format("2006-01-02"), "fall 必须紧接 spring 结束次日")
	assert.Equal(t, "2022-01-18", fall.End.Format("2006-01-02"))
}

func TestClassStarJarTermFallsBackToNaturalSemester(t *testing.T) {
	f := setupC1(t)
	term, err := classStarJarTermOf(f.ctx, f.db, testC1KG, testC1ClassB, time.Now())
	require.NoError(t, err)
	assert.Equal(t, "2020-2021-spring", term.Term)
	assert.Equal(t, "2021-02-01", term.Start.Format("2006-01-02"))
	assert.Equal(t, "2021-07-31", term.End.Format("2006-01-02"))
}

func TestClassStarJarTermFallsBackToNaturalYear(t *testing.T) {
	f := setupC1(t)
	now := time.Now().In(shanghaiLoc())
	// 无学年：自然年按学期切成两段（1/1–7/31 / 8/1–12/31），两段不重叠、无空档。
	term, err := classStarJarTermOf(f.ctx, f.db, testC1KG, testC1ClassC, now)
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%d-spring", now.Year()), term.Term)
	assert.Equal(t, fmt.Sprintf("%d-01-01", now.Year()), term.Start.Format("2006-01-02"))
	assert.Equal(t, fmt.Sprintf("%d-07-31", now.Year()), term.End.Format("2006-01-02"))

	require.NoError(t, f.db.Model(&domain.KindergartenAppSetting{}).
		Where("kindergarten_id = ?", testC1KG).Update("current_semester", "fall").Error)
	fall, err := classStarJarTermOf(f.ctx, f.db, testC1KG, testC1ClassC, now)
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%d-fall", now.Year()), fall.Term)
	assert.Equal(t, fmt.Sprintf("%d-08-01", now.Year()), fall.Start.Format("2006-01-02"), "fall 紧接 spring 结束次日")
	assert.Equal(t, fmt.Sprintf("%d-12-31", now.Year()), fall.End.Format("2006-01-02"))
}

func TestClassStarJarTermRejectsForeignClass(t *testing.T) {
	f := setupC1(t)
	_, err := classStarJarTermOf(f.ctx, f.db, testC1KG, 999999, time.Now())
	require.ErrorIs(t, err, ErrClassNotFound)
}

// --- 校验与配置端点 ---

func TestValidateClassStarJarOnlyAcceptsCurrentTerm(t *testing.T) {
	f := setupC1(t)
	require.NoError(t, f.db.Create(&domain.ClassStarJar{
		KindergartenID: testC1KG, ClassID: testC1Class, Term: "2020-2021-spring", Goal: 30,
	}).Error)
	require.ErrorIs(t, validateClassStarJar(f.ctx, f.db, testC1KG, testC1Class), errTaskStarJarMissing,
		"上学期目标不得放行本学期任务")

	_, err := upsertClassStarJar(f.ctx, f.db, testC1KG, testC1Class, testC1Admin, 30, time.Now())
	require.NoError(t, err)
	require.NoError(t, validateClassStarJar(f.ctx, f.db, testC1KG, testC1Class))
}

func TestClassStarJarGetPutUpsertsSingleRow(t *testing.T) {
	f := setupC1(t)
	got, err := f.svc.GetClassStarJar(f.ctx, testC1KG, testC1Class)
	require.NoError(t, err)
	assert.Equal(t, 0, got["goal"])
	assert.Equal(t, false, got["configured"])
	assert.Equal(t, "2031-2032-spring", got["term"])

	_, err = f.svc.SetClassStarJar(f.ctx, testC1KG, testC1Class, testC1Admin, 0, "", "")
	require.ErrorIs(t, err, ErrStarJarGoalInvalid, "目标 0 必须被拒，0 是未配置不是关闭")

	got, err = f.svc.SetClassStarJar(f.ctx, testC1KG, testC1Class, testC1Admin, 30, "", "")
	require.NoError(t, err)
	assert.Equal(t, 30, got["goal"])
	assert.Equal(t, true, got["configured"])

	got, err = f.svc.SetClassStarJar(f.ctx, testC1KG, testC1Class, testC1Admin, 45, "", "")
	require.NoError(t, err)
	assert.Equal(t, 45, got["goal"])

	var rows int64
	require.NoError(t, f.db.Model(&domain.ClassStarJar{}).
		Where("kindergarten_id = ? AND class_id = ?", testC1KG, testC1Class).Count(&rows).Error)
	assert.Equal(t, int64(1), rows, "同一学期重复设置必须是覆盖，不新增行")

	// 软删后重设必须复活同一行（唯一索引包含软删行）。
	require.NoError(t, f.db.Where("kindergarten_id = ? AND class_id = ?", testC1KG, testC1Class).
		Delete(&domain.ClassStarJar{}).Error)
	got, err = f.svc.SetClassStarJar(f.ctx, testC1KG, testC1Class, testC1Admin, 60, "", "")
	require.NoError(t, err)
	assert.Equal(t, 60, got["goal"])
	require.NoError(t, f.db.Model(&domain.ClassStarJar{}).
		Where("kindergarten_id = ? AND class_id = ?", testC1KG, testC1Class).Count(&rows).Error)
	assert.Equal(t, int64(1), rows)
}

func TestClassStarJarGetRejectsForeignClass(t *testing.T) {
	f := setupC1(t)
	_, err := f.svc.GetClassStarJar(f.ctx, testC1KG, 999999)
	require.ErrorIs(t, err, ErrClassNotFound)
	_, err = f.svc.SetClassStarJar(f.ctx, testC1KG, 999999, testC1Admin, 10, "", "")
	require.ErrorIs(t, err, ErrClassNotFound)
}

// --- 管理端创建任务 ---

func TestAdminCreateTaskPersistsStructureFields(t *testing.T) {
	f := setupC1(t)
	req := f.createReq(testC1Class)
	req.AllowPublic = true
	req.RecognitionRitual = "voice"
	req.StarScope = "task"
	req.StarGoal = 5
	resp, err := f.svc.CreateTask(f.ctx, testC1KG, testC1Admin, req, "", "")
	require.NoError(t, err)

	tk := f.loadTask(t, resp["id"])
	assert.Equal(t, "延续本周班级主题", tk.LearningLink)
	assert.True(t, tk.AllowPublic)
	assert.Equal(t, domain.TaskRitualVoice, tk.RecognitionRitual)
	assert.Equal(t, domain.TaskStarScopeTask, tk.StarScope)
	assert.Equal(t, 5, tk.StarGoal)
	assert.Equal(t, uint64(0), tk.TargetStudentID)
	assert.Equal(t, 1, resp["notified_count"], "发布通知按家长去重，本夹具 1 个家长")
	assert.Equal(t, 0, resp["notify_failed_count"])
}

func TestAdminCreateTaskDefaults(t *testing.T) {
	f := setupC1(t)
	resp, err := f.svc.CreateTask(f.ctx, testC1KG, testC1Admin, f.createReq(testC1Class), "", "")
	require.NoError(t, err)
	tk := f.loadTask(t, resp["id"])
	assert.False(t, tk.AllowPublic, "allow_public 默认 false")
	assert.Equal(t, domain.TaskRitualSticker, tk.RecognitionRitual)
	assert.Equal(t, domain.TaskStarScopeTask, tk.StarScope)
	assert.Equal(t, 0, tk.StarGoal)
}

func TestAdminCreateTaskRequiresLearningLink(t *testing.T) {
	f := setupC1(t)
	req := f.createReq(testC1Class)
	req.LearningLink = ""
	_, err := f.svc.CreateTask(f.ctx, testC1KG, testC1Admin, req, "", "")
	require.EqualError(t, err, "BAD_LEARNING_LINK")
}

func TestAdminCreateTaskClassScopeRequiresJar(t *testing.T) {
	f := setupC1(t)
	req := f.createReq(testC1Class)
	req.StarScope = "class"
	_, err := f.svc.CreateTask(f.ctx, testC1KG, testC1Admin, req, "", "")
	require.ErrorIs(t, err, errTaskStarJarMissing)

	_, err = f.svc.SetClassStarJar(f.ctx, testC1KG, testC1Class, testC1Admin, 20, "", "")
	require.NoError(t, err)
	_, err = f.svc.CreateTask(f.ctx, testC1KG, testC1Admin, req, "", "")
	require.NoError(t, err)
}

func TestAdminCreateTaskIndividualForcesPublicOff(t *testing.T) {
	f := setupC1(t)
	req := f.createReq(testC1Class)
	req.TargetStudentID = testC1Student
	req.AllowPublic = true
	resp, err := f.svc.CreateTask(f.ctx, testC1KG, testC1Admin, req, "", "")
	require.NoError(t, err)
	tk := f.loadTask(t, resp["id"])
	assert.Equal(t, testC1Student, tk.TargetStudentID)
	assert.False(t, tk.AllowPublic, "个体任务强制不可公开")
}

func TestAdminCreateTaskRejectsForeignTargetStudent(t *testing.T) {
	f := setupC1(t)
	req := f.createReq(testC1ClassB)
	req.TargetStudentID = testC1Student // 学生属于 testC1Class
	_, err := f.svc.CreateTask(f.ctx, testC1KG, testC1Admin, req, "", "")
	require.ErrorIs(t, err, errTaskTargetInvalid)
}

func TestAdminCreateTaskRejectsInvalidEnumsAndActivityID(t *testing.T) {
	f := setupC1(t)

	req := f.createReq(testC1Class)
	req.RecognitionRitual = "firework"
	_, err := f.svc.CreateTask(f.ctx, testC1KG, testC1Admin, req, "", "")
	require.ErrorIs(t, err, ErrTaskRitualInvalid)

	req = f.createReq(testC1Class)
	req.StarScope = "galaxy"
	_, err = f.svc.CreateTask(f.ctx, testC1KG, testC1Admin, req, "", "")
	require.ErrorIs(t, err, ErrTaskStarScopeInvalid)

	req = f.createReq(testC1Class)
	req.RecognitionRitual = "class_star"
	req.StarScope = "task"
	req.StarGoal = 0
	_, err = f.svc.CreateTask(f.ctx, testC1KG, testC1Admin, req, "", "")
	require.ErrorIs(t, err, ErrTaskStarGoalInvalid)

	req = f.createReq(testC1Class)
	req.ActivityID = "act_injected"
	_, err = f.svc.CreateTask(f.ctx, testC1KG, testC1Admin, req, "", "")
	require.ErrorIs(t, err, ErrTaskActivityIDForbidden)
}

// --- 管理端编辑任务 ---

func (f *c1Fixture) addCheckin(t *testing.T, taskID, userID uint64, public bool) domain.TaskCheckIn {
	t.Helper()
	c := domain.TaskCheckIn{
		TaskID: taskID, StudentID: testC1Student, UserID: userID, Date: f.today,
	}
	if public {
		now := time.Now()
		c.IsPublicRequested = true
		c.PublicScopeConfirmed = true
		c.PublicApprovedAt = &now
		c.PublicApprovedBy = testC1Admin
	}
	require.NoError(t, f.db.Create(&c).Error)
	return c
}

func TestAdminUpdateTaskStructuralLock(t *testing.T) {
	f := setupC1(t)
	resp, err := f.svc.CreateTask(f.ctx, testC1KG, testC1Admin, f.createReq(testC1Class), "", "")
	require.NoError(t, err)
	taskID := resp["id"].(uint64)
	f.addCheckin(t, taskID, testC1Parent, false)

	req := f.createReq(testC1Class)
	req.Title = "改标题"
	req.Frequency = string(domain.TaskFrequencyWeekly)
	err = f.svc.UpdateTask(f.ctx, testC1KG, testC1Admin, taskID, updateFromCreate(req), "", "")
	require.ErrorIs(t, err, ErrTaskStructuralLocked)

	req = f.createReq(testC1Class)
	req.Title = "只改标题"
	req.LearningLink = "新的学习关联"
	require.NoError(t, f.svc.UpdateTask(f.ctx, testC1KG, testC1Admin, taskID, updateFromCreate(req), "", ""))
	tk := f.loadTask(t, taskID)
	assert.Equal(t, "只改标题", tk.Title)
	assert.Equal(t, "新的学习关联", tk.LearningLink)

	// status 不是结构字段：有提交后仍必须能结束任务。
	end := int8(0)
	u := updateFromCreate(f.createReq(testC1Class))
	u.Title = "只改标题"
	u.LearningLink = "新的学习关联"
	u.Status = &end
	require.NoError(t, f.svc.UpdateTask(f.ctx, testC1KG, testC1Admin, taskID, u, "", ""))
	assert.Equal(t, int8(0), f.loadTask(t, taskID).Status)
}

func TestAdminUpdateTaskClosingPublicRevokes(t *testing.T) {
	f := setupC1(t)
	req := f.createReq(testC1Class)
	req.AllowPublic = true
	resp, err := f.svc.CreateTask(f.ctx, testC1KG, testC1Admin, req, "", "")
	require.NoError(t, err)
	taskID := resp["id"].(uint64)
	c := f.addCheckin(t, taskID, testC1Parent, true)

	off := false
	u := updateFromCreate(f.createReq(testC1Class))
	u.AllowPublic = &off
	require.NoError(t, f.svc.UpdateTask(f.ctx, testC1KG, testC1Admin, taskID, u, "", ""))

	var fresh domain.TaskCheckIn
	require.NoError(t, f.db.Unscoped().Where("id = ?", c.ID).First(&fresh).Error)
	assert.False(t, fresh.IsPublicRequested)
	assert.False(t, fresh.PublicScopeConfirmed)
	assert.Nil(t, fresh.PublicApprovedAt)
	assert.NotNil(t, fresh.PublicRevokedAt)
	assert.Equal(t, TaskPublicRevoked, publicCheckinStatus(&fresh))
}

// 改个体必须强制关闭公开。注意：有提交后该路径本身被结构锁拦住
// （D12「有提交后 target_student_id 一律拒绝」），所以「撤销已公开内容」
// 在管理端只可能在并发窗口里发生——断言落到 task 落库值，不伪造一条不可达的链路。
func TestAdminUpdateTaskIndividualForcesPublicOff(t *testing.T) {
	f := setupC1(t)
	req := f.createReq(testC1Class)
	req.AllowPublic = true
	resp, err := f.svc.CreateTask(f.ctx, testC1KG, testC1Admin, req, "", "")
	require.NoError(t, err)
	taskID := resp["id"].(uint64)

	keepPublic := true
	studentID := testC1Student
	u := updateFromCreate(f.createReq(testC1Class))
	u.TargetStudentID = &studentID
	u.AllowPublic = &keepPublic
	require.NoError(t, f.svc.UpdateTask(f.ctx, testC1KG, testC1Admin, taskID, u, "", ""))

	tk := f.loadTask(t, taskID)
	assert.Equal(t, testC1Student, tk.TargetStudentID)
	assert.False(t, tk.AllowPublic, "改个体必须强制关闭公开")
}

// 有提交后连「改个体」也必须被结构锁挡住（D12）：否则已公开内容会留在墙上。
func TestAdminUpdateTaskRejectsTargetChangeAfterCheckin(t *testing.T) {
	f := setupC1(t)
	req := f.createReq(testC1Class)
	req.AllowPublic = true
	resp, err := f.svc.CreateTask(f.ctx, testC1KG, testC1Admin, req, "", "")
	require.NoError(t, err)
	taskID := resp["id"].(uint64)
	f.addCheckin(t, taskID, testC1Parent, true)

	studentID := testC1Student
	u := updateFromCreate(f.createReq(testC1Class))
	u.TargetStudentID = &studentID
	require.ErrorIs(t, f.svc.UpdateTask(f.ctx, testC1KG, testC1Admin, taskID, u, "", ""),
		ErrTaskStructuralLocked)
	// 断言**本次要改的那个字段**没落库：只看 AllowPublic 的话，
	// 把实现改成「先写 target 再撞结构锁」也照样通过。
	assert.Equal(t, uint64(0), f.loadTask(t, taskID).TargetStudentID, "被拒的更新不得留下半个改动")
}

func TestAdminUpdateTaskOmittedPointersKeepValues(t *testing.T) {
	f := setupC1(t)
	req := f.createReq(testC1Class)
	req.AllowPublic = true
	req.StarGoal = 7
	req.Description = "原来的说明"
	resp, err := f.svc.CreateTask(f.ctx, testC1KG, testC1Admin, req, "", "")
	require.NoError(t, err)
	taskID := resp["id"].(uint64)

	u := updateFromCreate(f.createReq(testC1Class))
	u.Title = "只改标题"
	u.LearningLink = "延续本周班级主题"
	u.AllowPublic = nil
	u.StarGoal = nil
	u.Description = nil
	require.NoError(t, f.svc.UpdateTask(f.ctx, testC1KG, testC1Admin, taskID, u, "", ""))

	tk := f.loadTask(t, taskID)
	assert.True(t, tk.AllowPublic, "缺省 allow_public 不得把公开悄悄关掉")
	assert.Equal(t, 7, tk.StarGoal, "缺省 star_goal 不得把目标清零")
	assert.Equal(t, "原来的说明", tk.Description, "缺省 description 不得把说明清空")
}

func TestAdminUpdateTaskClassScopeRequiresJar(t *testing.T) {
	f := setupC1(t)
	resp, err := f.svc.CreateTask(f.ctx, testC1KG, testC1Admin, f.createReq(testC1Class), "", "")
	require.NoError(t, err)
	taskID := resp["id"].(uint64)

	u := updateFromCreate(f.createReq(testC1Class))
	u.StarScope = "class"
	require.ErrorIs(t, f.svc.UpdateTask(f.ctx, testC1KG, testC1Admin, taskID, u, "", ""), errTaskStarJarMissing)

	_, err = f.svc.SetClassStarJar(f.ctx, testC1KG, testC1Class, testC1Admin, 15, "", "")
	require.NoError(t, err)
	require.NoError(t, f.svc.UpdateTask(f.ctx, testC1KG, testC1Admin, taskID, u, "", ""))
	assert.Equal(t, domain.TaskStarScopeClass, f.loadTask(t, taskID).StarScope)
}

func TestAdminUpdateTaskNotFound(t *testing.T) {
	f := setupC1(t)
	u := updateFromCreate(f.createReq(testC1Class))
	err := f.svc.UpdateTask(f.ctx, testC1KG, testC1Admin, 999999, u, "", "")
	require.True(t, errors.Is(err, errTaskNotFound))
}

// --- 审查后补的边界用例 ---

// 星星目标不能为负：0 表示不启用，负数会让进度/封顶逻辑全部失效。
func TestAdminCreateTaskRejectsNegativeStarGoal(t *testing.T) {
	f := setupC1(t)
	req := f.createReq(testC1Class)
	req.StarGoal = -1
	_, err := f.svc.CreateTask(f.ctx, testC1KG, testC1Admin, req, "", "")
	require.ErrorIs(t, err, ErrTaskStarGoalNegative)
}

func TestAdminUpdateTaskRejectsNegativeStarGoal(t *testing.T) {
	f := setupC1(t)
	req := f.createReq(testC1Class)
	req.StarGoal = 3 // 先给一个非零目标：若实现把负数 clamp 成 0，断言才抓得住
	resp, err := f.svc.CreateTask(f.ctx, testC1KG, testC1Admin, req, "", "")
	require.NoError(t, err)
	taskID := resp["id"].(uint64)

	bad := -5
	u := updateFromCreate(f.createReq(testC1Class))
	u.StarGoal = &bad
	require.ErrorIs(t, f.svc.UpdateTask(f.ctx, testC1KG, testC1Admin, taskID, u, "", ""),
		ErrTaskStarGoalNegative)
	assert.Equal(t, 3, f.loadTask(t, taskID).StarGoal, "被拒的目标不得落库，也不得被 clamp 成 0")
}

// 老师端与 admin 共用同一个下界：两个入口各写一套校验迟早会漂移。
func TestTeacherCreateTaskRejectsNegativeStarGoal(t *testing.T) {
	f := setupC1(t)
	tc := &TeacherPortalContext{UserID: testC1Admin, KindergartenID: testC1KG, ClassID: testC1Class}
	_, err := f.svcT.CreateTask(f.ctx, tc, TeacherTaskInput{
		Title: "C1老师任务", LearningLink: "学习关联",
		Frequency: string(domain.TaskFrequencyOnce),
		StartDate: f.date(0), EndDate: f.date(6),
		StarGoal: ptrTo(-1),
	})
	require.ErrorIs(t, err, ErrTeacherInvalidParams)
}

// 同学年（2031-2031）必须整段退化成自然年：fall 兜底若用 endYear 会倒置。
func TestClassStarJarTermRejectsSameYearAcademicYear(t *testing.T) {
	f := setupC1(t)
	require.NoError(t, f.db.Model(&domain.KindergartenAppSetting{}).
		Where("kindergarten_id = ?", testC1KG).Update("current_semester", "fall").Error)
	require.NoError(t, f.db.Model(&domain.Class{}).
		Where("id = ?", testC1ClassC).Update("academic_year", "2031-2031").Error)

	now := time.Now().In(shanghaiLoc())
	term, err := classStarJarTermOf(f.ctx, f.db, testC1KG, testC1ClassC, now)
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%d-fall", now.Year()), term.Term)
	assert.False(t, term.End.Before(term.Start), "区间不得倒置")
}

// 非相邻学年（2031-2035）的 fall 兜底必须只跨一年，不能拉到 2035。
func TestClassStarJarTermFallFallbackUsesStartYear(t *testing.T) {
	f := setupC1(t)
	require.NoError(t, f.db.Model(&domain.KindergartenAppSetting{}).
		Where("kindergarten_id = ?", testC1KG).Update("current_semester", "fall").Error)
	require.NoError(t, f.db.Model(&domain.Class{}).
		Where("id = ?", testC1ClassB).Update("academic_year", "2031-2035").Error)

	term, err := classStarJarTermOf(f.ctx, f.db, testC1KG, testC1ClassB, time.Now())
	require.NoError(t, err)
	assert.Equal(t, "2031-2035-fall", term.Term)
	assert.Equal(t, "2031-09-01", term.Start.Format("2006-01-02"))
	assert.Equal(t, "2032-01-31", term.End.Format("2006-01-02"))
}

// 学年里带空格也要能命中 academic_years：term 与查询都用规范形式。
func TestClassStarJarTermNormalizesWhitespaceAcademicYear(t *testing.T) {
	f := setupC1(t)
	require.NoError(t, f.db.Model(&domain.Class{}).
		Where("id = ?", testC1Class).Update("academic_year", "2031 - 2032").Error)

	term, err := classStarJarTermOf(f.ctx, f.db, testC1KG, testC1Class, time.Now())
	require.NoError(t, err)
	assert.Equal(t, "2031-2032-spring", term.Term)
	// 命中 academic_years 的配置，并按「两学期无缝」把 spring 起点收到 fall 结束次日。
	assert.Equal(t, "2031-01-21", term.Start.Format("2006-01-02"), "必须命中 academic_years 的配置")
}

// 跨园所班级 + goal=0：404 优先于 422（契约：班级不属于当前园所 → 404）。
func TestSetClassStarJarForeignClassTakesPrecedence(t *testing.T) {
	f := setupC1(t)
	_, err := f.svc.SetClassStarJar(f.ctx, testC1KG, 999999, testC1Admin, 0, "", "")
	require.ErrorIs(t, err, ErrClassNotFound)
}

// --- 老师端：可选字段缺省必须保留原值 ---

func (f *c1Fixture) teacherCtx() *TeacherPortalContext {
	return &TeacherPortalContext{UserID: testC1Admin, KindergartenID: testC1KG, ClassID: testC1Class}
}

func (f *c1Fixture) teacherCreate(t *testing.T, mut func(*TeacherTaskInput)) uint64 {
	t.Helper()
	in := TeacherTaskInput{
		Title:             "老师任务",
		LearningLink:      "学习关联",
		Frequency:         string(domain.TaskFrequencyDaily),
		StartDate:         f.date(0),
		EndDate:           f.date(6),
		RecognitionRitual: string(domain.TaskRitualSticker),
		StarScope:         string(domain.TaskStarScopeTask),
	}
	if mut != nil {
		mut(&in)
	}
	out, err := f.svcT.CreateTask(f.ctx, f.teacherCtx(), in)
	require.NoError(t, err)
	return out["id"].(uint64)
}

func TestTeacherUpdateTaskOmittedStarGoalKeepsValue(t *testing.T) {
	f := setupC1(t)
	goal := 20
	taskID := f.teacherCreate(t, func(in *TeacherTaskInput) { in.StarGoal = &goal })

	// 契约示例里的最小编辑：不带 star_goal。
	require.NoError(t, f.svcT.UpdateTask(f.ctx, f.teacherCtx(), taskID, TeacherTaskInput{
		Title: "老师改名", LearningLink: "学习关联",
	}))
	assert.Equal(t, 20, f.loadTask(t, taskID).StarGoal, "省略 star_goal 不得把目标清零")
}

func TestTeacherUpdateTaskOmittedAllowPublicKeepsValue(t *testing.T) {
	f := setupC1(t)
	yes := true
	taskID := f.teacherCreate(t, func(in *TeacherTaskInput) { in.AllowPublic = &yes })
	c := f.addCheckin(t, taskID, testC1Parent, true)

	require.NoError(t, f.svcT.UpdateTask(f.ctx, f.teacherCtx(), taskID, TeacherTaskInput{
		Title: "老师改名", LearningLink: "学习关联",
	}))
	assert.True(t, f.loadTask(t, taskID).AllowPublic, "省略 allow_public 不得把公开悄悄关掉")

	var fresh domain.TaskCheckIn
	require.NoError(t, f.db.Unscoped().Where("id = ?", c.ID).First(&fresh).Error)
	assert.Nil(t, fresh.PublicRevokedAt, "省略 allow_public 不得撤销已公开内容")
}

// 老师端与管理端同一套枚举校验：非规范值不得落库。
func TestTeacherCreateTaskRejectsInvalidEnums(t *testing.T) {
	f := setupC1(t)
	_, err := f.svcT.CreateTask(f.ctx, f.teacherCtx(), TeacherTaskInput{
		Title: "老师任务", LearningLink: "学习关联",
		Frequency: string(domain.TaskFrequencyOnce), StartDate: f.date(0), EndDate: f.date(6),
		RecognitionRitual: "firework",
	})
	require.ErrorIs(t, err, ErrTeacherInvalidParams)

	_, err = f.svcT.CreateTask(f.ctx, f.teacherCtx(), TeacherTaskInput{
		Title: "老师任务", LearningLink: "学习关联",
		Frequency: string(domain.TaskFrequencyOnce), StartDate: f.date(0), EndDate: f.date(6),
		StarScope: "galaxy",
	})
	require.ErrorIs(t, err, ErrTeacherInvalidParams)

	// 带空格的范围会被 trim 成 class，从而触发班级罐校验（而不是静默落一个脏值）。
	// 未配置班级罐时老师端返回 ErrTaskStarJarMissing（C-4 修正）：spec 要求
	// teacher/admin/activity 三个入口给同一句引导，不能折叠成笼统的「参数无效」。
	_, err = f.svcT.CreateTask(f.ctx, f.teacherCtx(), TeacherTaskInput{
		Title: "老师任务", LearningLink: "学习关联",
		Frequency: string(domain.TaskFrequencyOnce), StartDate: f.date(0), EndDate: f.date(6),
		StarScope: " class ",
	})
	require.ErrorIs(t, err, ErrTaskStarJarMissing)
}

// 老师端与管理端同口径：罐被删后，改标题不该连带 422。
func TestTeacherUpdateTaskClassScopeTitleEditNotBlockedByMissingJar(t *testing.T) {
	f := setupC1(t)
	_, err := f.svc.SetClassStarJar(f.ctx, testC1KG, testC1Class, testC1Admin, 20, "", "")
	require.NoError(t, err)
	taskID := f.teacherCreate(t, func(in *TeacherTaskInput) {
		in.StarScope = string(domain.TaskStarScopeClass)
	})

	require.NoError(t, f.db.Unscoped().
		Where("kindergarten_id = ? AND class_id = ?", testC1KG, testC1Class).
		Delete(&domain.ClassStarJar{}).Error)

	require.NoError(t, f.svcT.UpdateTask(f.ctx, f.teacherCtx(), taskID, TeacherTaskInput{
		Title: "老师改名", LearningLink: "学习关联",
	}))
	assert.Equal(t, "老师改名", f.loadTask(t, taskID).Title)
}

// 历史脏数据（class_star + task + goal=0）不该把管理端的无关编辑一起挡住。
func TestAdminUpdateTaskLegacyZeroGoalTitleEdit(t *testing.T) {
	f := setupC1(t)
	resp, err := f.svc.CreateTask(f.ctx, testC1KG, testC1Admin, f.createReq(testC1Class), "", "")
	require.NoError(t, err)
	taskID := resp["id"].(uint64)
	require.NoError(t, f.db.Model(&domain.ParentTask{}).Where("id = ?", taskID).Updates(map[string]any{
		"recognition_ritual": string(domain.TaskRitualClassStar),
		"star_scope":         string(domain.TaskStarScopeTask),
		"star_goal":          0,
	}).Error)

	u := updateFromCreate(f.createReq(testC1Class))
	u.Title = "只改标题"
	u.RecognitionRitual = ""
	u.StarScope = ""
	require.NoError(t, f.svc.UpdateTask(f.ctx, testC1KG, testC1Admin, taskID, u, "", ""))
	assert.Equal(t, "只改标题", f.loadTask(t, taskID).Title)
}

// 秋季学期的 term 与区间来自 academic_years，不能被 spring 硬编码遮住。
func TestClassStarJarTermFallSemester(t *testing.T) {
	f := setupC1(t)
	require.NoError(t, f.db.Model(&domain.KindergartenAppSetting{}).
		Where("kindergarten_id = ?", testC1KG).Update("current_semester", "fall").Error)

	term, err := classStarJarTermOf(f.ctx, f.db, testC1KG, testC1Class, time.Now())
	require.NoError(t, err)
	assert.Equal(t, "2031-2032-fall", term.Term)
	assert.Equal(t, "2030-09-01", term.Start.Format("2006-01-02"))
	assert.Equal(t, "2031-01-20", term.End.Format("2006-01-02"))
}

// 倒序学年必须整段退化成自然年，而不是产出 term_start > term_end 的区间。
func TestClassStarJarTermRejectsReversedAcademicYear(t *testing.T) {
	f := setupC1(t)
	require.NoError(t, f.db.Model(&domain.Class{}).
		Where("id = ?", testC1ClassC).Update("academic_year", "2032-2031").Error)
	now := time.Now().In(shanghaiLoc())
	term, err := classStarJarTermOf(f.ctx, f.db, testC1KG, testC1ClassC, now)
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%d-spring", now.Year()), term.Term)
	assert.False(t, term.End.Before(term.Start), "区间不得倒置")
}

// 创建时的纯空白标题必须拒绝：required 只判非空字符串，落库会变成空串。
func TestAdminCreateTaskRejectsBlankTitle(t *testing.T) {
	f := setupC1(t)
	req := f.createReq(testC1Class)
	req.Title = "   "
	_, err := f.svc.CreateTask(f.ctx, testC1KG, testC1Admin, req, "", "")
	require.EqualError(t, err, "BAD_TITLE")
}

// 更新时省略 learning_link 必须保留原值（不能因为「空串」被当成清空）。
func TestAdminUpdateTaskOmittedLearningLinkKeepsValue(t *testing.T) {
	f := setupC1(t)
	req := f.createReq(testC1Class)
	req.LearningLink = "原有学习关联"
	resp, err := f.svc.CreateTask(f.ctx, testC1KG, testC1Admin, req, "", "")
	require.NoError(t, err)
	taskID := resp["id"].(uint64)

	u := updateFromCreate(f.createReq(testC1Class))
	u.Title = "只改标题"
	u.LearningLink = ""
	require.NoError(t, f.svc.UpdateTask(f.ctx, testC1KG, testC1Admin, taskID, u, "", ""))
	assert.Equal(t, "原有学习关联", f.loadTask(t, taskID).LearningLink)
}

// 任务本来就是 class 范围、罐后来被删：改标题不能连带被 422。
func TestAdminUpdateTaskClassScopeTitleEditNotBlockedByMissingJar(t *testing.T) {
	f := setupC1(t)
	_, err := f.svc.SetClassStarJar(f.ctx, testC1KG, testC1Class, testC1Admin, 20, "", "")
	require.NoError(t, err)
	req := f.createReq(testC1Class)
	req.StarScope = "class"
	resp, err := f.svc.CreateTask(f.ctx, testC1KG, testC1Admin, req, "", "")
	require.NoError(t, err)
	taskID := resp["id"].(uint64)

	require.NoError(t, f.db.Unscoped().
		Where("kindergarten_id = ? AND class_id = ?", testC1KG, testC1Class).
		Delete(&domain.ClassStarJar{}).Error)

	u := updateFromCreate(f.createReq(testC1Class))
	u.Title = "只改标题"
	u.StarScope = "class"
	require.NoError(t, f.svc.UpdateTask(f.ctx, testC1KG, testC1Admin, taskID, u, "", ""))
	assert.Equal(t, "只改标题", f.loadTask(t, taskID).Title)
}

// updateFromCreate 把创建请求展开成更新请求；未显式设置的指针字段保持 nil，
// 用来守住「缺省 = 保留原值」的语义。
func updateFromCreate(req CreateTaskReq) UpdateTaskReq {
	desc := req.Description
	return UpdateTaskReq{
		Title:             req.Title,
		Description:       &desc,
		LearningLink:      req.LearningLink,
		ClassID:           req.ClassID,
		Frequency:         req.Frequency,
		StartDate:         req.StartDate,
		EndDate:           req.EndDate,
		RecognitionRitual: req.RecognitionRitual,
		StarScope:         req.StarScope,
	}
}
