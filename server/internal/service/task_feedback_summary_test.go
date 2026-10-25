package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/domain"
)

// 3.11 老师端反馈汇总：口径必须与管理端 7.2 完全一致
// （分母 = 回应承诺的义务家长数，分子 = 去重后的 (回应, 家长) 对）。
//
// 这些用例跑真实 MySQL：聚合结果、按孩子归属、越权拒绝对内存桩没有意义。

func feedbackTargetCount(n int) *int { return &n }

func (f *c2Fixture) feedbackSummary(t *testing.T, taskID uint64) map[string]any {
	t.Helper()
	out, err := f.svcT.TaskFeedbackSummary(f.ctx, f.teacherCtx(), taskID)
	require.NoError(t, err)
	return out
}

func TestTaskFeedbackSummaryNoCheckins(t *testing.T) {
	f := setupC2(t)
	taskID := f.createTaskIn(t, testC2Class, "反馈汇总-空")

	out := f.feedbackSummary(t, taskID)
	assert.Nil(t, out["feedback_coverage"], "没有提交就没有义务集，给 null 而不是 0%")
	assert.Nil(t, out["child_happy_rate"])
	assert.Nil(t, out["ack_rate"])
	assert.Empty(t, out["children"])
}

func TestTaskFeedbackSummaryPartialAndCoParent(t *testing.T) {
	f := setupC2(t)
	taskID := f.createTaskIn(t, testC2Class, "反馈汇总-部分")
	now := time.Now()
	respID := f.addResponse(t, taskID, domain.TaskResponseSingle, now, feedbackTargetCount(2))
	f.addCheckinOn(t, taskID, testC2Student, testC2Parent, f.today, now, respID)

	// 另一位家长绑定同一个孩子（co-parent）：两条反馈都要保留，不能互相覆盖。
	require.NoError(t, f.db.Unscoped().Create(&domain.StudentParent{
		StudentID: testC2Student, UserID: testC2ParentB,
		Relation: domain.RelationFather, IsPrimary: false,
	}).Error)
	f.addReaction(t, respID, testC2Parent, domain.TaskReactionChildHappy)
	f.addReaction(t, respID, testC2ParentB, domain.TaskReactionThanks)

	out := f.feedbackSummary(t, taskID)
	assert.Equal(t, "100%", out["feedback_coverage"], "2 对反馈 / 义务 2 个家长")
	assert.Equal(t, "50%", out["child_happy_rate"])
	assert.Equal(t, "50%", out["ack_rate"], "thanks 计入家长确认率")
	assert.Equal(t, 0, out["unattributed_feedback"])

	children := out["children"].([]map[string]any)
	require.Len(t, children, 1)
	assert.Equal(t, testC2Student, children[0]["student_id"])
	assert.Equal(t, 1, children[0]["child_happy"])
	assert.Equal(t, 0, children[0]["received"])
	assert.Equal(t, 1, children[0]["thanks"], "co-parent 的反馈归属到同一个孩子")
}

func TestTaskFeedbackSummaryExcludesUnreferencedResponses(t *testing.T) {
	f := setupC2(t)
	taskID := f.createTaskIn(t, testC2Class, "反馈汇总-补充回应")
	now := time.Now()
	mainID := f.addResponse(t, taskID, domain.TaskResponseSingle, now, feedbackTargetCount(1))
	f.addCheckinOn(t, taskID, testC2Student, testC2Parent, f.today, now, mainID)
	f.addReaction(t, mainID, testC2Parent, domain.TaskReactionReceived)

	// 补充回应同样是一行 TaskResponse，但不被任何提交引用 —— 它的反馈不能进主回应指标。
	extraID := f.addResponse(t, taskID, domain.TaskResponseSingle, now, feedbackTargetCount(1))
	f.addReaction(t, extraID, testC2Parent, domain.TaskReactionChildHappy)

	out := f.feedbackSummary(t, taskID)
	assert.Equal(t, "100%", out["feedback_coverage"], "只算被提交引用的主回应")
	assert.Equal(t, "0%", out["child_happy_rate"], "补充回应的 child_happy 不计入")
	assert.Equal(t, "100%", out["ack_rate"])
	counts := out["feedback_counts"].(map[string]any)
	assert.Equal(t, 1, counts["pairs"])
}

func TestTaskFeedbackSummaryAttributesPeriodResponsePerChild(t *testing.T) {
	f := setupC2(t)
	taskID := f.createTaskIn(t, testC2Class, "反馈汇总-汇总回应")
	now := time.Now()
	respID := f.addResponse(t, taskID, domain.TaskResponsePeriod, now, feedbackTargetCount(2))
	f.addCheckinOn(t, taskID, testC2Student, testC2Parent, f.today, now, respID)
	// 同一家长的两个孩子都被这条周期回应覆盖（同班兄弟姐妹）。
	f.addCheckinOn(t, taskID, testC2StudentC, testC2Parent, f.today, now, respID)

	f.addReaction(t, respID, testC2Parent, domain.TaskReactionChildHappy)

	out := f.feedbackSummary(t, taskID)
	assert.Equal(t, "50%", out["feedback_coverage"], "1 对反馈 / 义务 2")
	children := out["children"].([]map[string]any)
	require.Len(t, children, 2)
	for _, c := range children {
		assert.Equal(t, 1, c["child_happy"], "同一家长的反馈对其覆盖的每个孩子各计一次")
	}
}

func TestTaskFeedbackSummaryUnknownBaselineIsExcluded(t *testing.T) {
	f := setupC2(t)
	taskID := f.createTaskIn(t, testC2Class, "反馈汇总-未知义务")
	now := time.Now()
	respID := f.addResponse(t, taskID, domain.TaskResponseSingle, now, nil)
	f.addCheckinOn(t, taskID, testC2Student, testC2Parent, f.today, now, respID)
	f.addReaction(t, respID, testC2Parent, domain.TaskReactionChildHappy)

	out := f.feedbackSummary(t, taskID)
	assert.Nil(t, out["feedback_coverage"], "义务未知时给 null，不能当 0")
	assert.Equal(t, 1, out["notify_baseline_missing"])
}

func TestTaskFeedbackSummaryRejectsOtherClass(t *testing.T) {
	f := setupC2(t)
	taskID := f.createTaskIn(t, testC2Class, "反馈汇总-越权")

	_, err := f.svcT.TaskFeedbackSummary(f.ctx, &TeacherPortalContext{
		UserID: testC2Admin, KindergartenID: testC2KG, ClassID: testC2ClassB,
	}, taskID)
	assert.ErrorIs(t, err, ErrTeacherNotFound, "跨班必须拒绝，不能读别班任务的反馈")

	_, err = f.svcT.TaskFeedbackSummary(f.ctx, &TeacherPortalContext{
		UserID: testC2Admin, KindergartenID: testC2OtherKG, ClassID: testC2ForeignClass,
	}, taskID)
	assert.ErrorIs(t, err, ErrTeacherNotFound, "跨园所同样拒绝")
}
