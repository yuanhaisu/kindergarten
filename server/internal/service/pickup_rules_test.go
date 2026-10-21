package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/domain"
)

// 这些测试覆盖 pickup_rules.go 的纯判定逻辑（design D5）。
// DB 侧的 upsert 覆盖语义与联查命中/未命中不由单测覆盖——项目测试惯例是
// 纯函数单测 + DB 行为走集成验证（见 cross_end_identity_test.go），
// 对应仓库层事实由 `go run ./cmd/dbcheck` 与 spec 逐条验收承担。

func dayTime(hhmm string) time.Time {
	t, err := time.Parse("2006-01-02 15:04", "2026-03-19 "+hhmm)
	if err != nil {
		panic(err)
	}
	return t
}

func rec(t domain.PickupType) *domain.PickupRecord {
	return &domain.PickupRecord{Type: t}
}

// --- ResolvePickupDeadline ---

func TestResolvePickupDeadline_Fallbacks(t *testing.T) {
	assert.Equal(t, "16:00", ResolvePickupDeadline(""), "空值必须回退缺省 16:00")
	assert.Equal(t, "16:00", ResolvePickupDeadline("   "), "空白必须回退缺省")
	assert.Equal(t, "16:00", ResolvePickupDeadline("bad"), "非法值必须回退缺省，不得 panic")
	assert.Equal(t, "17:30", ResolvePickupDeadline("17:30"), "合法值必须原样保留")
	assert.Equal(t, "17:30", ResolvePickupDeadline(" 17:30 "), "前后空白应被裁剪")
}

// --- IsConsecutiveSameDirection（连续同向）---

func TestIsConsecutiveSameDirection_FirstOfDayAllowsAny(t *testing.T) {
	assert.False(t, IsConsecutiveSameDirection(nil, domain.PickupTypeDropOff),
		"当日无记录时，送入可作为首次登记")
	assert.False(t, IsConsecutiveSameDirection(nil, domain.PickupTypePickUp),
		"当日无记录时，接走也可作为首次登记（但 service 层另有「须先送入」约束）")
}

func TestIsConsecutiveSameDirection_AlternatingAllowed(t *testing.T) {
	assert.False(t, IsConsecutiveSameDirection(rec(domain.PickupTypeDropOff), domain.PickupTypePickUp),
		"送入之后接走是合法交替")
	assert.False(t, IsConsecutiveSameDirection(rec(domain.PickupTypePickUp), domain.PickupTypeDropOff),
		"接走之后再次送入（返园）是合法交替")
}

func TestIsConsecutiveSameDirection_SameDirectionRejected(t *testing.T) {
	assert.True(t, IsConsecutiveSameDirection(rec(domain.PickupTypeDropOff), domain.PickupTypeDropOff),
		"连续送入必须被拒")
	assert.True(t, IsConsecutiveSameDirection(rec(domain.PickupTypePickUp), domain.PickupTypePickUp),
		"连续接走必须被拒")
}

// --- NeedsRemarkForRepeatPickup（重复接走）---

func TestNeedsRemarkForRepeatPickup(t *testing.T) {
	assert.False(t, NeedsRemarkForRepeatPickup(0), "首次接走理由可选")
	assert.True(t, NeedsRemarkForRepeatPickup(1), "当日第 2 次接走理由必填")
	assert.True(t, NeedsRemarkForRepeatPickup(3), "第 4 次同理必填")
}

// --- IsEarlyPickup（提前接走）---

func TestIsEarlyPickup_BeforeDeadlineIsEarly(t *testing.T) {
	assert.True(t, IsEarlyPickup(dayTime("15:59"), "16:00"), "放学前 1 分钟算提前")
	assert.True(t, IsEarlyPickup(dayTime("14:00"), "16:00"), "明显早于阈值算提前")
}

func TestIsEarlyPickup_AtOrAfterDeadlineNotEarly(t *testing.T) {
	assert.False(t, IsEarlyPickup(dayTime("16:00"), "16:00"), "恰好到达阈值不算提前")
	assert.False(t, IsEarlyPickup(dayTime("18:25"), "16:00"), "迟接不算提前（归滞留处置）")
}

func TestIsEarlyPickup_ClassDeadlineOverridesDefault(t *testing.T) {
	// 班级配置 17:00：16:30 尚未到该班放学时刻 → 算提前接走
	assert.True(t, IsEarlyPickup(dayTime("16:30"), "17:00"),
		"班级配置 17:00 时，16:30 早于该班放学时刻，应判提前")
	// 未配置走缺省 16:00：16:30 已过 → 不算提前
	assert.False(t, IsEarlyPickup(dayTime("16:30"), ""),
		"未配置时按缺省 16:00 判，16:30 已过该时刻，不判提前")
}

// --- NeedsRemarkForPickup（汇总判定）---

func TestNeedsRemarkForPickup_DropOffNeverRequires(t *testing.T) {
	required, reason := NeedsRemarkForPickup(domain.PickupTypeDropOff, 5, dayTime("07:00"), "16:00")
	assert.False(t, required, "送入一律不要理由")
	assert.Empty(t, reason, "送入不得给出必填原因")
}

func TestNeedsRemarkForPickup_FirstPickupInWindowOptional(t *testing.T) {
	required, _ := NeedsRemarkForPickup(domain.PickupTypePickUp, 0, dayTime("17:30"), "16:00")
	assert.False(t, required, "首次接走且在放学后，理由可选")
}

func TestNeedsRemarkForPickup_RepeatRequires(t *testing.T) {
	required, reason := NeedsRemarkForPickup(domain.PickupTypePickUp, 1, dayTime("17:30"), "16:00")
	assert.True(t, required, "重复接走理由必填")
	assert.Contains(t, reason, "重复接走")
}

func TestNeedsRemarkForPickup_EarlyRequires(t *testing.T) {
	required, reason := NeedsRemarkForPickup(domain.PickupTypePickUp, 0, dayTime("14:00"), "16:00")
	assert.True(t, required, "提前接走理由必填")
	assert.Contains(t, reason, "提前接走")
}

func TestNeedsRemarkForPickup_BothTriggersListed(t *testing.T) {
	required, reason := NeedsRemarkForPickup(domain.PickupTypePickUp, 1, dayTime("14:00"), "16:00")
	assert.True(t, required, "重复 + 提前同现时理由必填")
	assert.Contains(t, reason, "重复接走")
	assert.Contains(t, reason, "提前接走")
}

// --- IsOverdueNotPicked（超时未接）---

func TestIsOverdueNotPicked_NoRecordNotOverdue(t *testing.T) {
	assert.False(t, IsOverdueNotPicked(nil, dayTime("18:00"), "16:00"),
		"当日无记录 = 未到园，不属滞留")
}

func TestIsOverdueNotPicked_LastIsPickupNotOverdue(t *testing.T) {
	assert.False(t, IsOverdueNotPicked(rec(domain.PickupTypePickUp), dayTime("18:00"), "16:00"),
		"最后一次是接走 = 已离园，不属滞留")
}

func TestIsOverdueNotPicked_InSchoolAfterDeadlineOverdue(t *testing.T) {
	assert.True(t, IsOverdueNotPicked(rec(domain.PickupTypeDropOff), dayTime("16:30"), "16:00"),
		"仍在园且已过放学时刻 = 超时未接")
	assert.True(t, IsOverdueNotPicked(rec(domain.PickupTypeDropOff), dayTime("16:00"), "16:00"),
		"恰好到达放学时刻即算超时")
}

func TestIsOverdueNotPicked_InSchoolBeforeDeadlineNotOverdue(t *testing.T) {
	assert.False(t, IsOverdueNotPicked(rec(domain.PickupTypeDropOff), dayTime("15:00"), "16:00"),
		"仍在园但未到放学时刻，不属滞留")
}

// --- authorizationValid（I1 授权闭合）---

func validAuth() *domain.PickupAuthorization {
	return &domain.PickupAuthorization{
		Status:       1,
		ReviewStatus: 1,
	}
}

func TestAuthorizationValid_ValidPasses(t *testing.T) {
	assert.True(t, authorizationValid(validAuth(), dayTime("17:30")),
		"有效 + 已审核 + 未过期 必须判为可接")
}

func TestAuthorizationValid_NilRejected(t *testing.T) {
	assert.False(t, authorizationValid(nil, dayTime("17:30")), "nil 授权必须拒绝")
}

func TestAuthorizationValid_ReviewStatusGates(t *testing.T) {
	for _, rs := range []int8{0, 2} {
		a := validAuth()
		a.ReviewStatus = rs
		assert.False(t, authorizationValid(a, dayTime("17:30")),
			"review_status=%d（待审核/已拒绝）必须拒绝", rs)
	}
}

func TestAuthorizationValid_RevokedRejected(t *testing.T) {
	a := validAuth()
	a.Status = 0
	assert.False(t, authorizationValid(a, dayTime("17:30")), "已撤销必须拒绝")
}

func TestAuthorizationValid_ExpiredRejected(t *testing.T) {
	a := validAuth()
	expired := dayTime("17:00")
	a.ExpireAt = &expired
	assert.False(t, authorizationValid(a, dayTime("17:30")), "已过期必须拒绝")

	// 未过期则通过
	future := dayTime("23:59")
	a.ExpireAt = &future
	assert.True(t, authorizationValid(a, dayTime("17:30")), "未过期应通过")
}

// 场景：中午接走就医、下午返园、放学再接走 —— 交替序列下应全程允许登记，
// 且只有第 2 次接走要求理由。
func TestSequence_MultiRoundTrip(t *testing.T) {
	last := (*domain.PickupRecord)(nil)

	// 送入
	require.False(t, IsConsecutiveSameDirection(last, domain.PickupTypeDropOff))
	last = rec(domain.PickupTypeDropOff)

	// 中午接走就医（首次接走，17:00 窗口内但重复否）
	req, _ := NeedsRemarkForPickup(domain.PickupTypePickUp, 0, dayTime("17:30"), "16:00")
	require.False(t, req, "第 1 次接走理由可选")
	require.False(t, IsConsecutiveSameDirection(last, domain.PickupTypePickUp))
	last = rec(domain.PickupTypePickUp)

	// 下午返园
	require.False(t, IsConsecutiveSameDirection(last, domain.PickupTypeDropOff))
	last = rec(domain.PickupTypeDropOff)

	// 放学再接走（第 2 次接走，理由必填）
	req, reason := NeedsRemarkForPickup(domain.PickupTypePickUp, 1, dayTime("17:30"), "16:00")
	require.True(t, req, "第 2 次接走理由必填")
	assert.Contains(t, reason, "重复接走")
	require.False(t, IsConsecutiveSameDirection(last, domain.PickupTypePickUp))
}
