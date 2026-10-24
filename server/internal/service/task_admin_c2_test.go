package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// C-2（7.1 多班活动 / 7.2 统计新口径 / 7.3 公开内容查看 / 7.4 下线旧字段）回归。
//
// 断言的是库里的真实状态与真实聚合结果；内存桩测不出「整组回滚」「按活动分组」
// 「分母是义务集」这些语义。
//
// 运行约束：
//   - 夹具固定 ID（992741-992750），清理全部用 Unscoped；不要并发跑两个 go test 进程。
//   - 测试直连开发库，跑前确认 :8080 无进程占用同一批表（memory: running-backend-steals-queue-jobs）。

const (
	testC2KG           uint64 = 992741
	testC2OtherKG      uint64 = 992742 // 用于「跨园所班级」校验
	testC2Class        uint64 = 992743
	testC2ClassB       uint64 = 992744
	testC2ForeignClass uint64 = 992745 // 属于 testC2OtherKG
	testC2Admin        uint64 = 992746
	testC2Parent       uint64 = 992747
	testC2Student      uint64 = 992748 // 班级 A
	testC2StudentB     uint64 = 992749 // 班级 B
	testC2ParentB      uint64 = 992750
	testC2StudentC     uint64 = 992751 // 班级 A，无家长（只用来做参与率分母）
)

type c2Fixture struct {
	db    *gorm.DB
	svc   *TaskService
	svcT  *TeacherPortalService // 通知写入点走老师端 service（主回应/补充回应的真实入口）
	ctx   context.Context
	today time.Time
}

func setupC2(t *testing.T) *c2Fixture {
	t.Helper()
	db := openTaskB1DB(t)
	loc := shanghaiLoc()
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	clean := func() {
		var taskIDs []uint64
		if err := db.Unscoped().Model(&domain.ParentTask{}).
			Where("kindergarten_id IN ?", []uint64{testC2KG, testC2OtherKG}).Pluck("id", &taskIDs).Error; err != nil {
			t.Logf("C2 夹具：查遗留任务失败: %v", err)
		}
		if len(taskIDs) > 0 {
			var respIDs []uint64
			if err := db.Unscoped().Model(&domain.TaskResponse{}).
				Where("task_id IN ?", taskIDs).Pluck("id", &respIDs).Error; err != nil {
				t.Logf("C2 夹具清理失败 responses(pluck): %v", err)
			}
			if len(respIDs) > 0 {
				for _, step := range []struct {
					name string
					err  error
				}{
					{"reactions", db.Unscoped().Where("response_id IN ?", respIDs).Delete(&domain.TaskResponseReaction{}).Error},
					{"supplements", db.Unscoped().Where("response_id IN ?", respIDs).Delete(&domain.TaskResponseSupplement{}).Error},
				} {
					if step.err != nil {
						t.Logf("C2 夹具清理失败 %s: %v", step.name, step.err)
					}
				}
			}
			for _, step := range []struct {
				name string
				err  error
			}{
				{"checkins", db.Unscoped().Where("task_id IN ?", taskIDs).Delete(&domain.TaskCheckIn{}).Error},
				{"responses", db.Unscoped().Where("task_id IN ?", taskIDs).Delete(&domain.TaskResponse{}).Error},
				{"states", db.Unscoped().Where("task_id IN ?", taskIDs).Delete(&domain.TaskCheckInState{}).Error},
				{"read_marks", db.Unscoped().Where("task_id IN ?", taskIDs).Delete(&domain.TaskReadMark{}).Error},
				{"tasks", db.Unscoped().Where("id IN ?", taskIDs).Delete(&domain.ParentTask{}).Error},
			} {
				if step.err != nil {
					t.Logf("C2 夹具清理失败 %s: %v", step.name, step.err)
				}
			}
		}
		users := []uint64{testC2Admin, testC2Parent, testC2ParentB}
		for _, step := range []struct {
			name string
			err  error
		}{
			{"messages", db.Unscoped().Where("user_id IN ?", users).Delete(&domain.Message{}).Error},
			{"operation_logs", db.Unscoped().Where("user_id IN ?", users).Delete(&domain.OperationLog{}).Error},
			{"star_jars", db.Unscoped().Where("kindergarten_id IN ?", []uint64{testC2KG, testC2OtherKG}).Delete(&domain.ClassStarJar{}).Error},
			{"student_parents", db.Unscoped().Where("student_id IN ?", []uint64{testC2Student, testC2StudentB, testC2StudentC}).Delete(&domain.StudentParent{}).Error},
			{"students", db.Unscoped().Where("id IN ?", []uint64{testC2Student, testC2StudentB, testC2StudentC}).Delete(&domain.Student{}).Error},
			{"classes", db.Unscoped().Where("id IN ?", []uint64{testC2Class, testC2ClassB, testC2ForeignClass}).Delete(&domain.Class{}).Error},
			{"app_settings", db.Unscoped().Where("kindergarten_id IN ?", []uint64{testC2KG, testC2OtherKG}).Delete(&domain.KindergartenAppSetting{}).Error},
			{"users", db.Unscoped().Where("id IN ?", users).Delete(&domain.User{}).Error},
			{"kindergartens", db.Unscoped().Where("id IN ?", []uint64{testC2KG, testC2OtherKG}).Delete(&domain.Kindergarten{}).Error},
		} {
			if step.err != nil {
				t.Logf("C2 夹具清理失败 %s: %v", step.name, step.err)
			}
		}
	}
	clean()
	t.Cleanup(clean)

	require.NoError(t, db.Unscoped().Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: testC2KG}, Name: "测试园所-C2",
	}).Error)
	require.NoError(t, db.Unscoped().Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: testC2OtherKG}, Name: "测试园所-C2-其他",
	}).Error)

	for _, u := range []struct {
		id   uint64
		name string
		ph   string
	}{
		{testC2Admin, "C2管理员", "13900009701"},
		{testC2Parent, "C2家长A", "13900009702"},
		{testC2ParentB, "C2家长B", "13900009703"},
	} {
		require.NoError(t, db.Unscoped().Create(&domain.User{
			BaseModel: domain.BaseModel{ID: u.id}, Nickname: u.name, Phone: u.ph, PasswordHash: "x",
		}).Error)
	}

	for _, c := range []struct {
		id   uint64
		kg   uint64
		name string
	}{
		{testC2Class, testC2KG, "C2一班"},
		{testC2ClassB, testC2KG, "C2二班"},
		{testC2ForeignClass, testC2OtherKG, "C2外园班"},
	} {
		require.NoError(t, db.Unscoped().Create(&domain.Class{
			BaseModel: domain.BaseModel{ID: c.id}, KindergartenID: c.kg,
			Name: c.name, Grade: "小班", AcademicYear: "2031-2032", Status: 1,
		}).Error)
	}
	require.NoError(t, db.Unscoped().Create(&domain.KindergartenAppSetting{
		KindergartenID: testC2KG, CurrentSemester: "spring", ExtraConfig: "{}",
	}).Error)

	for _, s := range []struct {
		id      uint64
		classID uint64
		name    string
	}{
		{testC2Student, testC2Class, "C2孩子A"},
		{testC2StudentB, testC2ClassB, "C2孩子B"},
		{testC2StudentC, testC2Class, "C2孩子C"},
	} {
		require.NoError(t, db.Unscoped().Create(&domain.Student{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: s.id}},
			KindergartenID:  testC2KG, ClassID: s.classID, Name: s.name, Status: 1,
			Birthday:   time.Date(2021, 5, 1, 0, 0, 0, 0, loc),
			EnrollDate: time.Date(2024, 9, 1, 0, 0, 0, 0, loc),
		}).Error)
	}
	require.NoError(t, db.Unscoped().Create(&domain.StudentParent{
		StudentID: testC2Student, UserID: testC2Parent, Relation: domain.RelationMother, IsPrimary: true,
	}).Error)
	require.NoError(t, db.Unscoped().Create(&domain.StudentParent{
		StudentID: testC2StudentB, UserID: testC2ParentB, Relation: domain.RelationMother, IsPrimary: true,
	}).Error)
	// studentC 也挂在家长 A 名下：班级 A 有 2 个目标孩子，但只有 1 个家长
	// ——参与率分母是学生数，通知计数按家长去重，两者不能被混为一谈。
	require.NoError(t, db.Unscoped().Create(&domain.StudentParent{
		StudentID: testC2StudentC, UserID: testC2Parent, Relation: domain.RelationFather, IsPrimary: false,
	}).Error)

	return &c2Fixture{
		db: db,
		svc: NewTaskService(repository.NewTaskRepo(db), repository.NewOperationLogRepo(db),
			repository.NewMessageRepo(db), db),
		svcT: NewTeacherPortalService(db, nil, nil, repository.NewMessageRepo(db), nil, nil,
			config.FaceConfig{}),
		ctx:   context.Background(),
		today: today,
	}
}

func (f *c2Fixture) date(offset int) string {
	return f.today.AddDate(0, 0, offset).Format("2006-01-02")
}

func (f *c2Fixture) activityReq(classIDs ...uint64) CreateActivityReq {
	return CreateActivityReq{
		Title:        "C2春季阅读周",
		LearningLink: "配合本周春天主题",
		Frequency:    string(domain.TaskFrequencyDaily),
		StartDate:    f.date(0),
		EndDate:      f.date(6),
		ClassIDs:     classIDs,
	}
}

// tasksOfActivity 读该活动下未删除的任务（按 id 升序）。
func (f *c2Fixture) tasksOfActivity(t *testing.T, activityID string) []domain.ParentTask {
	t.Helper()
	var out []domain.ParentTask
	require.NoError(t, f.db.Where("kindergarten_id = ? AND activity_id = ? AND deleted_at IS NULL",
		testC2KG, activityID).Order("id ASC").Find(&out).Error)
	return out
}

func (f *c2Fixture) countTasksInKG(t *testing.T) int64 {
	t.Helper()
	var n int64
	require.NoError(t, f.db.Unscoped().Model(&domain.ParentTask{}).
		Where("kindergarten_id = ?", testC2KG).Count(&n).Error)
	return n
}
