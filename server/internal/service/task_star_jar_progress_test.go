package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// C-4 班级星星罐进度（tasks 3.7 的 class 侧）回归。
//
// 复用 task_response_supplement_test.go 的 setupB1 夹具（固定园所/班级/两个孩子/
// 一个每日任务）。断言落在真实 MySQL 上：口径里「只算 class scope」「只算主回应」
// 「三层软删」「按提交日期落在学期内」四条都只能由真实 SQL 结果证明。
//
// 本文件**不**覆盖：老师端页面的渲染与点击（属真实小程序验证）、
// 学期边界之外的理论分支（term 解析由 task_admin_c1_test.go 覆盖）。

// b1TaskService 用 B1 夹具的库构造 TaskService：班级罐的进度在 TaskService 上，
// 老师端 service 不持有这套查询。
func b1TaskService(db *gorm.DB) *TaskService {
	return NewTaskService(
		repository.NewTaskRepo(db),
		repository.NewOperationLogRepo(db),
		repository.NewMessageRepo(db),
		db,
	)
}

// resetB1Jar 清空 B1 园所的班级罐行。setupB1 的 clean() 不含 class_star_jars，
// 残留行会让后续用例读到上一次的 goal（jar 是 read 路径，不会自动重建）。
func resetB1Jar(t *testing.T, db *gorm.DB) {
	t.Helper()
	del := func() {
		if err := db.Unscoped().Where("kindergarten_id = ?", testB1KG).
			Delete(&domain.ClassStarJar{}).Error; err != nil {
			t.Logf("清理班级罐失败: %v", err)
		}
	}
	del()
	t.Cleanup(del)
}

func readJar(t *testing.T, svc *TaskService) map[string]any {
	t.Helper()
	got, err := svc.GetClassStarJarWithProgress(context.Background(), testB1KG, testB1Class)
	require.NoError(t, err)
	return got
}

// addStarCheckin 直接落一条「主回应是 class_star」的提交。
// 不用 BatchRespondTask：本组用例要构造别的班级/别的园所/上学期的提交，
// 而这些请求走老师端 API 会被班级归属检查挡住——这里测的是计数 SQL 本身。
func addStarCheckin(
	t *testing.T, db *gorm.DB,
	kindergartenID, taskID, studentID, userID uint64,
	date time.Time, ritual string,
) uint64 {
	t.Helper()
	c := domain.TaskCheckIn{TaskID: taskID, StudentID: studentID, UserID: userID, Date: date, Content: "C4 提交"}
	require.NoError(t, db.Create(&c).Error)
	r := domain.TaskResponse{
		KindergartenID: kindergartenID, TaskID: taskID, TeacherID: testB1Teacher,
		Ritual: ritual, ResponseType: domain.TaskResponsePeriod,
	}
	require.NoError(t, db.Create(&r).Error)
	require.NoError(t, db.Model(&domain.TaskCheckIn{}).Where("id = ?", c.ID).
		Update("response_id", r.ID).Error)
	return c.ID
}

func newB1Task(t *testing.T, db *gorm.DB, classID, kindergartenID uint64, scope, ritual string, start, end time.Time) domain.ParentTask {
	t.Helper()
	tk := domain.ParentTask{
		KindergartenID: kindergartenID, ClassID: classID, CreatorID: testB1Teacher,
		Title: "C4 班级罐任务", Frequency: domain.TaskFrequencyDaily,
		StartDate: start, EndDate: end, Status: 1,
		StarScope: scope, RecognitionRitual: ritual,
	}
	require.NoError(t, db.Create(&tk).Error)
	return tk
}

func TestClassStarJarProgress_CountsOnlyClassScopeMainClassStar(t *testing.T) {
	f := setupB1(t)
	resetB1Jar(t, f.db)
	svc := b1TaskService(f.db)
	ctx := context.Background()
	loc := shanghaiLoc()
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	_, err := svc.SetClassStarJar(ctx, testB1KG, testB1Class, testB1Teacher, 5, "", "")
	require.NoError(t, err)

	// 1) 按任务范围的任务：period 汇总覆盖 2 条提交（2 星归任务罐），班级罐仍是 0。
	require.NoError(t, f.db.Model(&domain.ParentTask{}).Where("id = ?", f.task.ID).
		Update("star_scope", domain.TaskStarScopeTask).Error)
	_, err = f.svcT.BatchRespondTask(ctx, f.tc, f.task.ID, domain.TaskRitualClassStar, "", "本周汇总", 0)
	require.NoError(t, err)
	jar := readJar(t, svc)
	assert.EqualValues(t, 0, jar["stars"], "按任务范围不能计入班级罐（design D6）")

	// 2) 同一批提交，任务改成按班级范围 → 2 条提交各 1 星（period 覆盖 N 条计 N 星）。
	require.NoError(t, f.db.Model(&domain.ParentTask{}).Where("id = ?", f.task.ID).
		Update("star_scope", domain.TaskStarScopeClass).Error)
	jar = readJar(t, svc)
	assert.EqualValues(t, 2, jar["stars"])

	// 3) 贴纸主回应不计星：另起一条按班级范围的任务。
	task2 := newB1Task(t, f.db, testB1Class, testB1KG,
		domain.TaskStarScopeClass, domain.TaskRitualSticker, today, today.AddDate(0, 0, 6))
	c3 := domain.TaskCheckIn{TaskID: task2.ID, StudentID: testB1Student, UserID: testB1Parent, Date: today, Content: "贴纸"}
	require.NoError(t, f.db.Create(&c3).Error)
	_, err = f.svcT.RespondTask(ctx, f.tc, task2.ID, c3.ID, domain.TaskRitualSticker, "flower", "", 0)
	require.NoError(t, err)
	assert.EqualValues(t, 2, readJar(t, svc)["stars"], "贴纸主回应不计星")

	// 4) 补充回应不计星：直接在库里造一条 class_star 补充回应。
	// 走 API 会被「补充回应只支持贴纸/语音」拦掉（另一条用例覆盖），
	// 这里要证明的是**计数 SQL 认的是 checkin.response_id**，不是「存在 class_star 回应」。
	sp := domain.TaskResponse{
		KindergartenID: testB1KG, TaskID: task2.ID, TeacherID: testB1Teacher,
		Ritual: domain.TaskRitualClassStar, ResponseType: domain.TaskResponseSingle,
	}
	require.NoError(t, f.db.Create(&sp).Error)
	require.NoError(t, f.db.Create(&domain.TaskResponseSupplement{
		KindergartenID: testB1KG, ResponseID: sp.ID, CheckInID: c3.ID,
	}).Error)
	assert.EqualValues(t, 2, readJar(t, svc)["stars"], "补充回应不计星")

	// 5) 软删提交不计星。
	require.NoError(t, f.db.Where("id = ?", f.c2.ID).Delete(&domain.TaskCheckIn{}).Error)
	assert.EqualValues(t, 1, readJar(t, svc)["stars"])

	// 6) 软删回应不计星（c1 的主回应被软删 → 0 星）。
	mainID := f.reloadCheckin(t, f.c1.ID).ResponseID
	require.NotZero(t, mainID)
	require.NoError(t, f.db.Where("id = ?", mainID).Delete(&domain.TaskResponse{}).Error)
	assert.EqualValues(t, 0, readJar(t, svc)["stars"])
}

func TestClassStarJarProgress_OverGoalIsFlaggedAndCapped(t *testing.T) {
	f := setupB1(t)
	resetB1Jar(t, f.db)
	svc := b1TaskService(f.db)
	ctx := context.Background()

	require.NoError(t, f.db.Model(&domain.ParentTask{}).Where("id = ?", f.task.ID).
		Update("star_scope", domain.TaskStarScopeClass).Error)
	_, err := f.svcT.BatchRespondTask(ctx, f.tc, f.task.ID, domain.TaskRitualClassStar, "", "本周汇总", 0)
	require.NoError(t, err)

	// 未满罐：2 星 / 目标 3 → 66%，不算超额。
	_, err = svc.SetClassStarJar(ctx, testB1KG, testB1Class, testB1Teacher, 3, "", "")
	require.NoError(t, err)
	jar := readJar(t, svc)
	assert.EqualValues(t, 2, jar["stars"])
	assert.EqualValues(t, 66, jar["progress"], "2/3 向下取整")
	assert.Equal(t, false, jar["over_goal"])

	// 目标下调到低于已集星 → 封顶 100 且标「已超额」，绝不出现 >100。
	_, err = svc.SetClassStarJar(ctx, testB1KG, testB1Class, testB1Teacher, 1, "", "")
	require.NoError(t, err)
	jar = readJar(t, svc)
	assert.EqualValues(t, 100, jar["progress"])
	assert.Equal(t, true, jar["over_goal"])

	// 恰好达到目标 → 100%，不标超额。
	_, err = svc.SetClassStarJar(ctx, testB1KG, testB1Class, testB1Teacher, 2, "", "")
	require.NoError(t, err)
	jar = readJar(t, svc)
	assert.EqualValues(t, 100, jar["progress"])
	assert.Equal(t, false, jar["over_goal"])

	// 未配置：progress 为 null（不是 0%），星数仍然如实返回。
	require.NoError(t, f.db.Where("kindergarten_id = ? AND class_id = ?", testB1KG, testB1Class).
		Delete(&domain.ClassStarJar{}).Error)
	jar = readJar(t, svc)
	assert.Equal(t, false, jar["configured"])
	assert.EqualValues(t, 0, jar["goal"])
	assert.Nil(t, jar["progress"], "未配置时进度没有分母，不能给 0")
	assert.Equal(t, false, jar["over_goal"])
	assert.EqualValues(t, 2, jar["stars"])
}

func TestClassStarJarProgress_TermClassAndKindergartenScoped(t *testing.T) {
	f := setupB1(t)
	resetB1Jar(t, f.db)
	svc := b1TaskService(f.db)
	ctx := context.Background()
	loc := shanghaiLoc()
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	_, err := svc.SetClassStarJar(ctx, testB1KG, testB1Class, testB1Teacher, 10, "", "")
	require.NoError(t, err)

	// 本班、本学期、class scope、class_star → 计 1 星（基线）。
	base := newB1Task(t, f.db, testB1Class, testB1KG,
		domain.TaskStarScopeClass, domain.TaskRitualClassStar, today, today)
	addStarCheckin(t, f.db, testB1KG, base.ID, testB1Student, testB1Parent, today, domain.TaskRitualClassStar)
	assert.EqualValues(t, 1, readJar(t, svc)["stars"])

	// 别的班级：同园所另一班的任务不计入本班罐。
	other := newB1Task(t, f.db, testB1ClassOther, testB1KG,
		domain.TaskStarScopeClass, domain.TaskRitualClassStar, today, today)
	addStarCheckin(t, f.db, testB1KG, other.ID, testB1StudentOth, testB1ParentOther, today, domain.TaskRitualClassStar)
	assert.EqualValues(t, 1, readJar(t, svc)["stars"], "跨班不计入")

	// 别的园所：班级 id 相同但园所不同 → 不计入（多租户隔离）。
	foreign := newB1Task(t, f.db, testB1Class, 999999,
		domain.TaskStarScopeClass, domain.TaskRitualClassStar, today, today)
	addStarCheckin(t, f.db, 999999, foreign.ID, testB1Student, testB1Parent, today.AddDate(0, 0, 1), domain.TaskRitualClassStar)
	assert.EqualValues(t, 1, readJar(t, svc)["stars"], "跨园所不计入")

	// 上一学期的提交：罐按学期重置，上学期的星不能带到本学期。
	prev := newB1Task(t, f.db, testB1Class, testB1KG,
		domain.TaskStarScopeClass, domain.TaskRitualClassStar, today.AddDate(-1, 0, 0), today)
	addStarCheckin(t, f.db, testB1KG, prev.ID, testB1Student, testB1Parent,
		today.AddDate(-1, 0, 0), domain.TaskRitualClassStar)
	jar := readJar(t, svc)
	assert.EqualValues(t, 1, jar["stars"], "学期外提交不计入")

	// B1 班级没有学年、园所学期是 fall → 自然年切成 8/1–12/31（两学期无缝不重叠）。
	assert.Equal(t, time.Date(now.Year(), 8, 1, 0, 0, 0, 0, loc).Format("2006-01-02"), jar["term_start"])
	assert.Equal(t, time.Date(now.Year(), 12, 31, 0, 0, 0, 0, loc).Format("2006-01-02"), jar["term_end"])
}

// 三个创建入口必须返回同一句引导（spec「星星罐支持按任务与按班级两种范围」）：
// 老师端曾把「班级罐缺失」折叠成笼统的「请求参数无效」，老师看不出该去哪配。
func TestTeacherTaskClassScopeMissingJarIsActionable(t *testing.T) {
	f := setupB1(t)
	resetB1Jar(t, f.db)
	ctx := context.Background()
	loc := shanghaiLoc()
	today := time.Now().In(loc).Format("2006-01-02")
	end := time.Now().In(loc).AddDate(0, 0, 6).Format("2006-01-02")

	_, err := f.svcT.CreateTask(ctx, f.tc, TeacherTaskInput{
		Title: "C4 未配置班级罐", LearningLink: "学习关联",
		Frequency: string(domain.TaskFrequencyOnce), StartDate: today, EndDate: end,
		StarScope: domain.TaskStarScopeClass, RecognitionRitual: domain.TaskRitualClassStar,
	})
	require.ErrorIs(t, err, ErrTaskStarJarMissing, "创建路径必须让 handler 拿到可读引导")

	// 更新为 class 也是同一句引导。
	task := newB1Task(t, f.db, testB1Class, testB1KG,
		domain.TaskStarScopeTask, domain.TaskRitualSticker, time.Now().In(loc), time.Now().In(loc).AddDate(0, 0, 6))
	goal := 0
	err = f.svcT.UpdateTask(ctx, f.tc, task.ID, TeacherTaskInput{
		Title: task.Title, LearningLink: "学习关联",
		Frequency: string(domain.TaskFrequencyDaily), StartDate: today, EndDate: end,
		StarScope: domain.TaskStarScopeClass, StarGoal: &goal,
	})
	require.ErrorIs(t, err, ErrTaskStarJarMissing, "更新为 class 也必须给引导")
}

// 管理端契约（C-3 已验证）不含进度：老师端的新字段不能漏到管理端响应里。
func TestClassStarJarProgress_AdminViewUnchanged(t *testing.T) {
	f := setupB1(t)
	resetB1Jar(t, f.db)
	svc := b1TaskService(f.db)
	ctx := context.Background()

	_, err := svc.SetClassStarJar(ctx, testB1KG, testB1Class, testB1Teacher, 10, "", "")
	require.NoError(t, err)

	admin, err := svc.GetClassStarJar(ctx, testB1KG, testB1Class)
	require.NoError(t, err)
	for _, key := range []string{"stars", "progress", "over_goal"} {
		_, ok := admin[key]
		assert.Falsef(t, ok, "管理端响应不应出现 %s", key)
	}

	teacher, err := svc.GetClassStarJarWithProgress(ctx, testB1KG, testB1Class)
	require.NoError(t, err)
	for _, key := range []string{"stars", "progress", "over_goal"} {
		_, ok := teacher[key]
		assert.Truef(t, ok, "老师端响应缺少 %s", key)
	}
}
