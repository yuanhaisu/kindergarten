package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/domain"
)

// 补齐 specs/pickup-custody/spec.md 中主路径此前未被覆盖的 Requirement：
//   Req 8  送入仅作到场标记（TC-NEW-PKU-070/071）
//   Req 9  接送异常由老师手动标记（TC-NEW-PKU-080/081/082）
//   Req 13 接走事件按配置通知家长（TC-NEW-PKU-120/121/122）
// 与 Req 11 老师侧一键通知。

// ==================== Req 8 送入仅作到场标记 ====================

// TC-NEW-PKU-070 送入不记录送托人
func TestDropOff_DoesNotRequirePerson(t *testing.T) {
	f := newPickupFixture(t, "00:00", testPickupClass)

	// 不给 authorization_id、不标例外 —— 送入必须能登记
	resp, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
		StudentID: f.studentID, Type: 1,
	}, domain.PickupMethodManual)
	require.NoError(t, err, "送入不记交接人，不得要求 authorization_id")

	assert.Equal(t, false, resp["is_authorized"], "送入不归因到授权人")

	var rec domain.PickupRecord
	require.NoError(t, f.svc.db.Where("student_id = ?", f.studentID).First(&rec).Error)
	assert.Nil(t, rec.PickupPersonID, "送入不得落 PickupPersonID")
	assert.False(t, rec.IsAuthorizedOverride, "送入不得误标授权例外")
	assert.Empty(t, rec.OverrideName, "送入不得记来接者姓名")
	assert.False(t, rec.IsAbnormal, "送入不得置异常")
	assert.Empty(t, rec.AbnormalReason, "送入不得有异常原因")
}

// TC-NEW-PKU-071 送入不要求理由
func TestDropOff_DoesNotRequireRemark(t *testing.T) {
	f := newPickupFixture(t, "23:59", testPickupClass) // 恒「提前」阈值下仍不要求

	_, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
		StudentID: f.studentID, Type: 1,
	}, domain.PickupMethodManual)
	assert.NoError(t, err, "送入一律不要理由，即使在「提前接走」阈值内")
}

// ==================== Req 9 接送异常由老师手动标记 ====================

// TC-NEW-PKU-080 老师手动标记异常
func TestAbnormal_ManualMark(t *testing.T) {
	f := newPickupFixture(t, "00:00", testPickupClass)
	_, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
		StudentID: f.studentID, Type: 1,
	}, domain.PickupMethodManual)
	require.NoError(t, err)

	var rec domain.PickupRecord
	require.NoError(t, f.svc.db.Where("student_id = ?", f.studentID).First(&rec).Error)

	resp, err := f.svc.MarkPickupAbnormal(ctx(), f.tc, rec.ID, true, "非授权人员接送，待复核")
	require.NoError(t, err)
	assert.Equal(t, true, resp["is_abnormal"])
	assert.Equal(t, "非授权人员接送，待复核", resp["abnormal_reason"])
}

// TC-NEW-PKU-081 老师手动取消异常
func TestAbnormal_ManualUnmark(t *testing.T) {
	f := newPickupFixture(t, "00:00", testPickupClass)
	_, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
		StudentID: f.studentID, Type: 1,
	}, domain.PickupMethodManual)
	require.NoError(t, err)

	var rec domain.PickupRecord
	require.NoError(t, f.svc.db.Where("student_id = ?", f.studentID).First(&rec).Error)

	_, err = f.svc.MarkPickupAbnormal(ctx(), f.tc, rec.ID, true, "先标后撤")
	require.NoError(t, err)
	_, err = f.svc.MarkPickupAbnormal(ctx(), f.tc, rec.ID, false, "")
	require.NoError(t, err)

	require.NoError(t, f.svc.db.Where("id = ?", rec.ID).First(&rec).Error)
	assert.False(t, rec.IsAbnormal, "取消后不得再计为异常")
}

// TC-NEW-PKU-082 例外自动置异常，其余触发不置异常
//
// 2026-10-29 语义修订（用户确认）：授权例外 = 非授权接送人放行，属风险信号，
// 登记时即置 is_abnormal。重复接走、提前接走仍只留理由、不自动置异常。
func TestAbnormal_OverrideAutoSet(t *testing.T) {
	deadline := "23:59" // 恒「提前」，故接走必须带理由
	f := newPickupFixture(t, deadline, testPickupClass)
	authID := f.makeAuthorization(t, "奶奶（张三）", 1, 1, nil)

	step := func(typ int8, remark string, auth *uint64, override bool, overrideName string) {
		_, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
			StudentID: f.studentID, Type: typ,
			AuthorizationID: auth, Remark: remark,
			IsAuthorizedOverride: override, OverrideName: overrideName,
		}, domain.PickupMethodManual)
		require.NoError(t, err)
	}

	step(1, "", nil, false, "")                // 送入
	step(2, "提前接走", &authID, false, "")        // 提前接走 → 不置异常
	step(1, "", nil, false, "")                // 返园
	step(2, "重复接走", &authID, false, "")        // 重复接走 → 不置异常
	step(1, "", nil, false, "")                // 再返园
	step(2, "家长临时委托，已电话确认", nil, true, "邻居王五") // 授权例外 → 置异常

	var recs []domain.PickupRecord
	require.NoError(t, f.svc.db.Where("student_id = ?", f.studentID).Order("id ASC").Find(&recs).Error)
	require.Len(t, recs, 6)

	for i, r := range recs[:5] {
		assert.Falsef(t, r.IsAbnormal, "第 %d 条（非例外）不得被系统自动置异常", i+1)
		assert.Emptyf(t, r.AbnormalReason, "第 %d 条（非例外）不得有异常原因", i+1)
	}
	assert.True(t, recs[5].IsAuthorizedOverride, "第 6 条应为例外接走")
	assert.True(t, recs[5].IsAbnormal, "例外接走必须自动置异常")
	assert.Equal(t, "家长临时委托，已电话确认", recs[5].AbnormalReason,
		"异常原因取登记理由")
}

// ==================== Req 13 接走事件按配置通知家长 ====================

// TC-NEW-PKU-120 开启通知时推送家长
func TestPickupNotify_WhenEnabled(t *testing.T) {
	f := newPickupFixture(t, "00:00", testPickupClass)
	parentUserID := f.makeParent(t)
	authID := f.makeAuthorization(t, "奶奶（张三）", 1, 1, nil) // NotifyOnPickup 默认 true

	_, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
		StudentID: f.studentID, Type: 1,
	}, domain.PickupMethodManual)
	require.NoError(t, err)
	_, err = f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
		StudentID: f.studentID, Type: 2, AuthorizationID: &authID,
	}, domain.PickupMethodManual)
	require.NoError(t, err)

	var msgs []domain.Message
	require.NoError(t, f.svc.db.Where("user_id = ? AND type = ?",
		parentUserID, string(domain.MessageTypePickup)).Find(&msgs).Error)
	require.Len(t, msgs, 1, "接走应向家长推一条接送消息")
	assert.Contains(t, msgs[0].Content, "接走", "消息须告知接走发生")
}

// TC-NEW-PKU-121 关闭通知时不推送
func TestPickupNotify_WhenDisabled(t *testing.T) {
	f := newPickupFixture(t, "00:00", testPickupClass)
	parentUserID := f.makeParent(t)
	authID := f.makeAuthorization(t, "奶奶（张三）", 1, 1, nil)

	// 关掉该授权人的接送通知
	require.NoError(t, f.svc.db.Model(&domain.PickupAuthorization{}).
		Where("id = ?", authID).Update("notify_on_pickup", false).Error)

	_, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
		StudentID: f.studentID, Type: 1,
	}, domain.PickupMethodManual)
	require.NoError(t, err)
	_, err = f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
		StudentID: f.studentID, Type: 2, AuthorizationID: &authID,
	}, domain.PickupMethodManual)
	require.NoError(t, err)

	var cnt int64
	f.svc.db.Model(&domain.Message{}).Where("user_id = ? AND type = ?",
		parentUserID, string(domain.MessageTypePickup)).Count(&cnt)
	assert.Equal(t, int64(0), cnt, "配置关闭时不得推送")
}

// TC-NEW-PKU-122 授权例外的接走也按配置通知
func TestPickupNotify_OverrideAlwaysNotifies(t *testing.T) {
	f := newPickupFixture(t, "00:00", testPickupClass)
	parentUserID := f.makeParent(t)

	_, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
		StudentID: f.studentID, Type: 1,
	}, domain.PickupMethodManual)
	require.NoError(t, err)
	_, err = f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
		StudentID: f.studentID, Type: 2,
		IsAuthorizedOverride: true, OverrideName: "邻居王五",
		Remark: "家长临时委托",
	}, domain.PickupMethodManual)
	require.NoError(t, err)

	var msgs []domain.Message
	require.NoError(t, f.svc.db.Where("user_id = ? AND type = ?",
		parentUserID, string(domain.MessageTypePickup)).Find(&msgs).Error)
	require.Len(t, msgs, 1, "例外接走必须通知（风险信号，不看 NotifyOnPickup）")
	assert.Contains(t, msgs[0].Content, "授权例外", "消息须体现本次为例外接走")
}

// ==================== Req 11 老师侧一键通知（限本班） ====================

// 老师一键通知滞留家长（TC-NEW-PKU-100 的老师端路径）
func TestTeacherNotifyOverdue_WorksForOwnClass(t *testing.T) {
	f := newPickupFixture(t, "00:00", testPickupClass) // 恒已过
	parentUserID := f.makeParent(t)

	_, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
		StudentID: f.studentID, Type: 1,
	}, domain.PickupMethodManual)
	require.NoError(t, err)

	resp, err := f.svc.NotifyOverdueParents(ctx(), f.tc, f.studentID)
	require.NoError(t, err, "本班滞留学生应可一键通知")
	assert.Equal(t, true, resp["notified"])

	var cnt int64
	f.svc.db.Model(&domain.Message{}).Where("user_id = ? AND type = ?",
		parentUserID, string(domain.MessageTypePickup)).Count(&cnt)
	assert.Equal(t, int64(1), cnt, "家长应收到接送类站内消息")
}

// 老师不得越权通知别班学生
func TestTeacherNotifyOverdue_DeniesOtherClass(t *testing.T) {
	f := newPickupFixture(t, "00:00", testPickupClass)
	f.makeParent(t)
	_, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
		StudentID: f.studentID, Type: 1,
	}, domain.PickupMethodManual)
	require.NoError(t, err)

	// 构造一个「别班」的老师上下文
	otherTC := &TeacherPortalContext{
		UserID:         testPickupOperatorID,
		KindergartenID: testPickupKG,
		ClassID:        testPickupClass2, // 不是学生的班级
	}
	_, err = f.svc.NotifyOverdueParents(ctx(), otherTC, f.studentID)
	assert.Error(t, err, "老师不得通知别班学生")
}
