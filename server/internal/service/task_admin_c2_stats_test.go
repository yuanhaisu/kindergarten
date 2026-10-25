package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/domain"
	"kindergarten-server/pkg/pagination"
)

// 7.2 统计新口径 + 7.4 下线旧字段的真实库用例。

// createTaskIn 在指定班级建一条班级任务（个体/活动另有构造）。
func (f *c2Fixture) createTaskIn(t *testing.T, classID uint64, title string) uint64 {
	t.Helper()
	out, err := f.svc.CreateTask(f.ctx, testC2KG, testC2Admin, CreateTaskReq{
		Title:        title,
		LearningLink: "配合本周主题",
		ClassID:      classID,
		Frequency:    string(domain.TaskFrequencyDaily),
		StartDate:    f.date(0),
		EndDate:      f.date(6),
	}, "", "")
	require.NoError(t, err)
	return out["id"].(uint64)
}

// addCheckinOn 建一条提交；day 决定 (task, student, date) 唯一键，createdAt 决定待回应时长。
func (f *c2Fixture) addCheckinOn(
	t *testing.T, taskID, studentID, userID uint64, day time.Time, createdAt time.Time, responseID uint64,
) domain.TaskCheckIn {
	t.Helper()
	c := domain.TaskCheckIn{
		BaseModel:  domain.BaseModel{CreatedAt: createdAt},
		TaskID:     taskID,
		StudentID:  studentID,
		UserID:     userID,
		Date:       day,
		ResponseID: responseID,
	}
	require.NoError(t, f.db.Create(&c).Error)
	return c
}

func (f *c2Fixture) addResponse(
	t *testing.T, taskID uint64, responseType string, createdAt time.Time, targetCount *int,
) uint64 {
	t.Helper()
	r := domain.TaskResponse{
		BaseModel:         domain.BaseModel{CreatedAt: createdAt},
		KindergartenID:    testC2KG,
		TaskID:            taskID,
		TeacherID:         testC2Admin,
		Ritual:            domain.TaskRitualSticker,
		ResponseType:      responseType,
		NotifyTargetCount: targetCount,
	}
	require.NoError(t, f.db.Create(&r).Error)
	return r.ID
}

func (f *c2Fixture) addReaction(t *testing.T, responseID, userID uint64, reaction string) {
	t.Helper()
	require.NoError(t, f.db.Create(&domain.TaskResponseReaction{
		KindergartenID: testC2KG, ResponseID: responseID, UserID: userID, Reaction: reaction,
	}).Error)
}

// statsRange 覆盖夹具任务的统计窗口。
func (f *c2Fixture) statsRange() (string, string) { return f.date(-7), f.date(7) }

func (f *c2Fixture) stats(t *testing.T) map[string]any {
	t.Helper()
	start, end := f.statsRange()
	out, err := f.svc.GetTaskStats(f.ctx, testC2KG, start, end)
	require.NoError(t, err)
	return out
}

func TestTaskStatMedianHours(t *testing.T) {
	assert.Nil(t, taskStatMedianHours(nil), "空集返回 null，不是 0")
	assert.Equal(t, 6.0, taskStatMedianHours([]float64{10, 2, 6}), "奇数取中位")
	assert.Equal(t, 4.0, taskStatMedianHours([]float64{6, 2}), "偶数取两中值平均")
	assert.Equal(t, 2.5, taskStatMedianHours([]float64{4, 1, 3, 2}))
	assert.Equal(t, 1.7, taskStatMedianHours([]float64{1.65, 1.66}), "四舍五入到 1 位小数")
}

func TestStatsNoDataReturnsNullRatios(t *testing.T) {
	f := setupC2(t)
	out := f.stats(t)
	assert.Nil(t, out["response_rate"], "无提交时回应率是 null，不是 0%")
	assert.Nil(t, out["participation_rate"])
	assert.Nil(t, out["feedback_coverage"])
	assert.Nil(t, out["median_response_hours"])
	assert.Equal(t, int64(0), out["total_checkins"])
	assert.Empty(t, out["by_class"])
	assert.Empty(t, out["by_activity"])
}

// 7.6：参与率分母是「在读学生」，不排除未绑定家长的学生。
// 排除会把「家长没绑小程序」这类触达问题藏进分母之外，指标反而失真。
func TestStatsParticipationCountsUnboundStudents(t *testing.T) {
	f := setupC2(t)
	const unbound uint64 = 992752
	loc := shanghaiLoc()
	require.NoError(t, f.db.Unscoped().Create(&domain.Student{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: unbound}},
		KindergartenID:  testC2KG, ClassID: testC2Class, Name: "C2无家长", Status: 1,
		Birthday:   time.Date(2021, 5, 1, 0, 0, 0, 0, loc),
		EnrollDate: time.Date(2024, 9, 1, 0, 0, 0, 0, loc),
	}).Error)
	// LIFO：本清理先于 setupC2 的夹具清理执行，避免学生挡住班级删除。
	t.Cleanup(func() { f.db.Unscoped().Delete(&domain.Student{}, unbound) })

	taskID := f.createTaskIn(t, testC2Class, "参与率-未绑定家长")
	now := time.Now()
	respID := f.addResponse(t, taskID, domain.TaskResponseSingle, now, nil)
	f.addCheckinOn(t, taskID, testC2Student, testC2Parent, f.today, now, respID)

	out := f.stats(t)
	// 班级 A 在读 3 人（孩子 A、孩子 C、无家长的孩子），只有 A 提交。
	assert.Equal(t, "33%", out["participation_rate"], "分母含未绑定家长的在读学生")
}

func TestStatsResponseAndParticipationAndPending(t *testing.T) {
	f := setupC2(t)
	// 班级 A 有 2 个在读学生（A、C）；只有 A 提交 → 参与率 50%。
	taskID := f.createTaskIn(t, testC2Class, "C2统计任务")
	now := time.Now()
	respID := f.addResponse(t, taskID, domain.TaskResponseSingle, now.Add(-time.Hour), nil)
	// 已回应：3 天前提交、1 小时前回应。
	f.addCheckinOn(t, taskID, testC2Student, testC2Parent, f.today.AddDate(0, 0, -3), now.Add(-72*time.Hour), respID)
	// 待回应且超过 48h：同一孩子换个日期（唯一键含 date）。
	f.addCheckinOn(t, taskID, testC2Student, testC2Parent, f.today.AddDate(0, 0, -2), now.Add(-70*time.Hour), 0)

	out := f.stats(t)
	assert.Equal(t, int64(2), out["total_checkins"])
	assert.Equal(t, "50%", out["response_rate"], "1/2 有主回应")
	assert.Equal(t, "50%", out["participation_rate"], "1 个孩子提交 / 班级 2 个在读")
	assert.Equal(t, int64(1), out["pending_count"])
	assert.Equal(t, int64(1), out["pending_over_48h"])

	byClass := out["by_class"].([]map[string]any)
	require.Len(t, byClass, 1)
	assert.Equal(t, "50%", byClass[0]["response_rate"])
}

func TestStatsParticipationIsUnionNotPerTaskSum(t *testing.T) {
	f := setupC2(t)
	// 同一孩子参与两个任务：分母仍是「班级目标孩子去重数」，不得按任务累加。
	t1 := f.createTaskIn(t, testC2Class, "C2任务一")
	t2 := f.createTaskIn(t, testC2Class, "C2任务二")
	f.addCheckinOn(t, t1, testC2Student, testC2Parent, f.today, time.Now(), 0)
	f.addCheckinOn(t, t2, testC2Student, testC2Parent, f.today, time.Now(), 0)

	out := f.stats(t)
	assert.Equal(t, "50%", out["participation_rate"], "1 个孩子 / 2 个在读（不是 2/4 或 100%）")
}

func TestStatsMedianUsesSingleResponsesOnly(t *testing.T) {
	f := setupC2(t)
	taskID := f.createTaskIn(t, testC2Class, "C2中位数任务")
	now := time.Now()

	// two single responses: +2h and +6h → 中位 4.0
	r1 := f.addResponse(t, taskID, domain.TaskResponseSingle, now.Add(-2*time.Hour), nil)
	r2 := f.addResponse(t, taskID, domain.TaskResponseSingle, now.Add(-6*time.Hour), nil)
	f.addCheckinOn(t, taskID, testC2Student, testC2Parent, f.today.AddDate(0, 0, -3), now.Add(-4*time.Hour), r1)
	f.addCheckinOn(t, taskID, testC2StudentC, testC2Parent, f.today.AddDate(0, 0, -3), now.Add(-12*time.Hour), r2)

	// period 汇总回应：覆盖的提交计入「已回应」，但**不进**中位数。
	rp := f.addResponse(t, taskID, domain.TaskResponsePeriod, now.Add(-100*time.Hour), nil)
	f.addCheckinOn(t, taskID, testC2Student, testC2Parent, f.today.AddDate(0, 0, -4), now.Add(-200*time.Hour), rp)

	// 负时长：回应时间早于提交（时钟漂移/手工改库）→ 剔除并计数。
	rn := f.addResponse(t, taskID, domain.TaskResponseSingle, now.Add(-10*time.Hour), nil)
	f.addCheckinOn(t, taskID, testC2StudentC, testC2Parent, f.today.AddDate(0, 0, -2), now.Add(-1*time.Hour), rn)

	out := f.stats(t)
	assert.Equal(t, 4.0, out["median_response_hours"], "period 与负时长都不能进中位数")
	assert.Equal(t, 1, out["median_response_negative_excluded"])
	assert.Equal(t, int64(4), out["total_checkins"])
	assert.Equal(t, "100%", out["response_rate"], "period 覆盖也算已回应")
}

func TestStatsFeedbackDenominatorIsObligationSet(t *testing.T) {
	f := setupC2(t)
	taskID := f.createTaskIn(t, testC2Class, "C2反馈任务")
	now := time.Now()
	// 义务集 2 个家长，实际 2 个人反馈。
	targets := 2
	respID := f.addResponse(t, taskID, domain.TaskResponseSingle, now.Add(-time.Hour), &targets)
	f.addCheckinOn(t, taskID, testC2Student, testC2Parent, f.today, now.Add(-3*time.Hour), respID)
	f.addReaction(t, respID, testC2Parent, domain.TaskReactionChildHappy)

	out := f.stats(t)
	assert.Equal(t, "50%", out["feedback_coverage"], "1 个家长反馈 / 义务 2 个家长")
	assert.Equal(t, "50%", out["child_happy_rate"])
	assert.Equal(t, "0%", out["ack_rate"], "child_happy 不能算进家长确认率")
	assert.Equal(t, 0, out["notify_baseline_missing"])

	// 第二位家长确认收到：覆盖率 100%，确认率 50%。
	f.addReaction(t, respID, testC2ParentB, domain.TaskReactionReceived)
	out = f.stats(t)
	assert.Equal(t, "100%", out["feedback_coverage"])
	assert.Equal(t, "50%", out["child_happy_rate"])
	assert.Equal(t, "50%", out["ack_rate"])
}

func TestStatsExcludesNullBaselineInsteadOfZero(t *testing.T) {
	f := setupC2(t)
	taskID := f.createTaskIn(t, testC2Class, "C2未知义务任务")
	now := time.Now()

	// 一个可用的义务集：2 个家长，其中 1 个反馈。
	two := 2
	known := f.addResponse(t, taskID, domain.TaskResponseSingle, now.Add(-time.Hour), &two)
	f.addCheckinOn(t, taskID, testC2Student, testC2Parent, f.today.AddDate(0, 0, -1), now.Add(-3*time.Hour), known)
	f.addReaction(t, known, testC2Parent, domain.TaskReactionChildHappy)

	// 另一个回应义务未知（历史数据/收件人查询失败），也有一条反馈。
	unknown := f.addResponse(t, taskID, domain.TaskResponseSingle, now.Add(-2*time.Hour), nil)
	f.addCheckinOn(t, taskID, testC2StudentC, testC2Parent, f.today.AddDate(0, 0, -2), now.Add(-4*time.Hour), unknown)
	f.addReaction(t, unknown, testC2Parent, domain.TaskReactionChildHappy)

	out := f.stats(t)
	// 未知那条既不进分母（不能被当 0 摊薄）也不进分子。
	// 若把 NULL 当 0，覆盖率会变成 2/2 = 100%。
	assert.Equal(t, "50%", out["feedback_coverage"], "NULL 义务必须排除，不能当 0")
	assert.Equal(t, 1, out["notify_baseline_missing"])
}

func TestStatsClampsRatesAt100(t *testing.T) {
	f := setupC2(t)
	taskID := f.createTaskIn(t, testC2Class, "C2超额反馈任务")
	now := time.Now()
	// 义务集只记了 1 个家长，但两位家长都反馈了（历史数据/手工改库）。
	one := 1
	respID := f.addResponse(t, taskID, domain.TaskResponseSingle, now.Add(-time.Hour), &one)
	f.addCheckinOn(t, taskID, testC2Student, testC2Parent, f.today, now.Add(-3*time.Hour), respID)
	f.addReaction(t, respID, testC2Parent, domain.TaskReactionChildHappy)
	f.addReaction(t, respID, testC2ParentB, domain.TaskReactionThanks)

	out := f.stats(t)
	assert.Equal(t, "100%", out["feedback_coverage"], "比率封顶 100%，不允许 200%")
}

func TestStatsReactionPairsAreDeduped(t *testing.T) {
	f := setupC2(t)
	taskID := f.createTaskIn(t, testC2Class, "C2反馈去重任务")
	now := time.Now()
	two := 2
	respID := f.addResponse(t, taskID, domain.TaskResponseSingle, now.Add(-time.Hour), &two)
	f.addCheckinOn(t, taskID, testC2Student, testC2Parent, f.today, now.Add(-3*time.Hour), respID)
	// 同一家长改反馈：库里只有一行（唯一索引），换了取值也只算一对。
	f.addReaction(t, respID, testC2Parent, domain.TaskReactionReceived)
	require.NoError(t, f.db.Model(&domain.TaskResponseReaction{}).
		Where("response_id = ? AND user_id = ?", respID, testC2Parent).
		Update("reaction", domain.TaskReactionChildHappy).Error)

	out := f.stats(t)
	assert.Equal(t, "50%", out["feedback_coverage"], "1 对反馈 / 义务 2")
	assert.Equal(t, "50%", out["child_happy_rate"], "按去重后的最新取值计")
	assert.Equal(t, "0%", out["ack_rate"])
}

func TestStatsParticipationIgnoresTransferredStudents(t *testing.T) {
	f := setupC2(t)
	// studentB 属于班级 B；这里直接给它写一条班级 A 任务的提交，模拟「提交后转班」。
	// 参与率的分母是班级 A 的 2 个在读孩子，分子只能算仍在目标范围里的提交，
	// 否则会算出 2/2=100%（而实际上班级里只有 studentA 参与）。
	taskID := f.createTaskIn(t, testC2Class, "C2转班任务")
	f.addCheckinOn(t, taskID, testC2Student, testC2Parent, f.today, time.Now(), 0)
	f.addCheckinOn(t, taskID, testC2StudentB, testC2ParentB, f.today, time.Now(), 0)

	out := f.stats(t)
	assert.Equal(t, "50%", out["participation_rate"], "转班孩子的旧提交不得让参与率超过目标范围")
}

func TestStatsByActivityGroupsStandaloneByTask(t *testing.T) {
	f := setupC2(t)
	resp, err := f.svc.CreateActivity(f.ctx, testC2KG, testC2Admin, f.activityReq(testC2Class, testC2ClassB), "", "")
	require.NoError(t, err)
	activityID := resp["activity_id"].(string)
	standalone := f.createTaskIn(t, testC2Class, "C2独立任务")

	out := f.stats(t)
	byActivity := out["by_activity"].([]map[string]any)
	require.Len(t, byActivity, 2, "一个活动 + 一个独立任务各占一条")

	var actRow, soloRow map[string]any
	for _, row := range byActivity {
		if row["activity_id"] == activityID {
			actRow = row
		}
		if row["activity_id"] == "" {
			soloRow = row
		}
	}
	require.NotNil(t, actRow, "活动必须按 activity_id 聚合成一条")
	assert.Equal(t, 2, actRow["tasks"])
	assert.Equal(t, uint64(0), actRow["task_id"])
	require.NotNil(t, soloRow, "独立任务不得被塞进一个「空活动」")
	assert.Equal(t, standalone, soloRow["task_id"])
	assert.Equal(t, 1, soloRow["tasks"])
}

func TestStatsByClassAggregatesPerClass(t *testing.T) {
	f := setupC2(t)
	a := f.createTaskIn(t, testC2Class, "C2一班任务")
	f.createTaskIn(t, testC2ClassB, "C2二班任务")
	f.addCheckinOn(t, a, testC2Student, testC2Parent, f.today, time.Now(), 0)

	out := f.stats(t)
	byClass := out["by_class"].([]map[string]any)
	require.Len(t, byClass, 2)
	for _, row := range byClass {
		switch row["class_id"] {
		case testC2Class:
			assert.Equal(t, "50%", row["participation_rate"], "1/2 在读孩子提交")
			assert.Equal(t, "0%", row["response_rate"])
		case testC2ClassB:
			// 有 1 个在读孩子、0 条提交 → 0%（分母存在，不是 null）。
			assert.Equal(t, "0%", row["participation_rate"])
			assert.Nil(t, row["response_rate"], "没有提交时回应率是 null")
		default:
			t.Fatalf("意外的班级行: %v", row["class_id"])
		}
	}
}

// --- 7.4 下线旧字段 ---

func TestTaskListAndDetailDropLegacyFields(t *testing.T) {
	f := setupC2(t)
	taskID := f.createTaskIn(t, testC2Class, "C2旧字段任务")

	list, _, _, _, err := f.svc.ListTasks(f.ctx, testC2KG, pagination.Query{Page: 1, PageSize: 10}, nil, nil)
	require.NoError(t, err)
	require.Len(t, list, 1)
	for _, k := range []string{"completion_rate", "checkin_count", "total_possible", "not_checkin_count", "points"} {
		assert.NotContains(t, list[0], k, "列表不得再返回旧口径字段: %s", k)
	}

	detail, err := f.svc.GetTaskDetail(f.ctx, testC2KG, taskID)
	require.NoError(t, err)
	for _, k := range []string{"completion_rate", "checkin_count", "total_possible", "not_checkin_count", "points"} {
		assert.NotContains(t, detail, k, "详情不得再返回旧口径字段: %s", k)
	}
	// 新口径字段必须在。
	for _, k := range []string{"submission_count", "responded_count", "pending_count", "response_rate", "participation_rate", "notify_target_count"} {
		assert.Contains(t, detail, k, "详情缺少新口径字段: %s", k)
	}
}

func TestTaskDetailMetricsAfterResponses(t *testing.T) {
	f := setupC2(t)
	taskID := f.createTaskIn(t, testC2Class, "C2详情指标任务")
	now := time.Now()
	respID := f.addResponse(t, taskID, domain.TaskResponseSingle, now, nil)
	f.addCheckinOn(t, taskID, testC2Student, testC2Parent, f.today, now, respID)
	f.addCheckinOn(t, taskID, testC2Student, testC2Parent, f.today.AddDate(0, 0, -1), now, 0)

	detail, err := f.svc.GetTaskDetail(f.ctx, testC2KG, taskID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), detail["submission_count"])
	assert.Equal(t, int64(1), detail["responded_count"])
	assert.Equal(t, int64(1), detail["pending_count"])
	assert.Equal(t, "50%", detail["response_rate"])
	assert.Equal(t, "50%", detail["participation_rate"], "1 个孩子 / 班级 2 个在读")
	// CreateTask 已经写过发布义务集（1 个家长）。
	require.NotNil(t, detail["notify_target_count"])
}

func TestStatsDropLegacyFields(t *testing.T) {
	f := setupC2(t)
	out := f.stats(t)
	for _, k := range []string{"avg_completion_rate", "total_points_awarded"} {
		assert.NotContains(t, out, k)
	}
	for _, k := range []string{
		"response_rate", "participation_rate", "pending_count", "pending_over_48h",
		"median_response_hours", "median_response_negative_excluded",
		"feedback_coverage", "child_happy_rate", "ack_rate", "notify_baseline_missing",
		"by_class", "by_activity",
	} {
		assert.Contains(t, out, k, "统计缺少新口径字段: %s", k)
	}
}

func TestIndividualTaskParticipationCountsStudentOnce(t *testing.T) {
	f := setupC2(t)
	out, err := f.svc.CreateTask(f.ctx, testC2KG, testC2Admin, CreateTaskReq{
		Title:           "C2个体任务",
		LearningLink:    "配合本周主题",
		ClassID:         testC2Class,
		TargetStudentID: testC2Student,
		Frequency:       string(domain.TaskFrequencyDaily),
		StartDate:       f.date(0),
		EndDate:         f.date(6),
	}, "", "")
	require.NoError(t, err)
	taskID := out["id"].(uint64)
	now := time.Now()
	// 每日任务：同一孩子两天各提交一次。
	f.addCheckinOn(t, taskID, testC2Student, testC2Parent, f.today, now, 0)
	f.addCheckinOn(t, taskID, testC2Student, testC2Parent, f.today.AddDate(0, 0, -1), now, 0)

	tk := f.loadTaskC2(t, taskID)
	participants, err := f.svc.tasks.CountParticipatingStudents(f.ctx, testC2KG, &tk)
	require.NoError(t, err)
	assert.Equal(t, int64(1), participants,
		"个体任务的分子按学生去重：两次提交仍是 1 个孩子（数行会给 2/1）")

	detail, err := f.svc.GetTaskDetail(f.ctx, testC2KG, taskID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), detail["submission_count"])
	assert.Equal(t, "100%", detail["participation_rate"])
}
