package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/domain"
)

// 覆盖 PickupOverview / PickupStudents 的状态派生（B1 看最后一条）
// 与超时未接的累计口径。DB 侧真实跑，与 teacher_pickup_create_test.go 同套 fixture。

// 状态以最后一次转移为准（TC-NEW-PKU-042）
func TestPickupStudents_StatusFollowsLastTransfer(t *testing.T) {
	// deadline 00:00 → 不判提前，理由可选
	f := newPickupFixture(t, "00:00", testPickupClass)
	authID := f.makeAuthorization(t, "奶奶（张三）", 1, 1, nil)
	step := func(typ int8) {
		_, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
			StudentID: f.studentID, Type: typ, AuthorizationID: &authID,
		}, domain.PickupMethodManual)
		require.NoError(t, err)
	}

	// 送入 → 判在园
	step(1)
	status := f.pickupStatus(t, "all")
	assert.Equal(t, "not_picked_up", status, "送入后应判「未接走」（在园）")

	// 接走 → 判已离园
	step(2)
	status = f.pickupStatus(t, "all")
	assert.Equal(t, "picked_up", status, "接走后应判「已接走」")

	// 再次送入 → 又判在园（而非「当日接过」）
	step(1)
	status = f.pickupStatus(t, "all")
	assert.Equal(t, "not_picked_up", status,
		"送入→接走→送入 后最后一次是送入，状态必须回到「未接走」")
}

// 超时未接被计数（TC-NEW-PKU-110）
func TestPickupOverview_OverdueCounted(t *testing.T) {
	// deadline 00:00 → 恒「已过放学时刻」；学生仍在园
	f := newPickupFixture(t, "00:00", testPickupClass)
	_, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
		StudentID: f.studentID, Type: 1,
	}, domain.PickupMethodManual)
	require.NoError(t, err)

	ov := f.overview(t)
	assert.Equal(t, int64(1), asI64(ov["overdue_not_picked"]),
		"仍在园且已过放学时刻必须计入超时未接")
	assert.Equal(t, int64(1), asI64(ov["not_picked_up"]), "应在「未接走」口径内")
}

// 随后被接走不重复计数（TC-NEW-PKU-111）
func TestPickupOverview_OverdueNotClearedAfterPickup(t *testing.T) {
	f := newPickupFixture(t, "00:00", testPickupClass)
	authID := f.makeAuthorization(t, "奶奶（张三）", 1, 1, nil)

	// 送入 → 已构成超时未接
	_, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
		StudentID: f.studentID, Type: 1,
	}, domain.PickupMethodManual)
	require.NoError(t, err)

	before := f.overview(t)
	require.Equal(t, int64(1), asI64(before["overdue_not_picked"]), "先确认已计入")

	// 随后接走
	_, err = f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
		StudentID: f.studentID, Type: 2, AuthorizationID: &authID,
	}, domain.PickupMethodManual)
	require.NoError(t, err)

	after := f.overview(t)
	assert.Equal(t, int64(1), asI64(after["overdue_not_picked"]),
		"随后被接走不得清零，当日仍计一次")
	assert.Equal(t, int64(1), asI64(after["picked_up"]), "此时应已计入「已接走」")
}

// 未到校不计入超时未接
func TestPickupOverview_NotArrivedNotOverdue(t *testing.T) {
	f := newPickupFixture(t, "00:00", testPickupClass)
	ov := f.overview(t)
	assert.Equal(t, int64(0), asI64(ov["overdue_not_picked"]), "未到校不属滞留")
	assert.Equal(t, int64(1), asI64(ov["not_arrived"]), "应计入「未入园」")
}

// 输出字段：is_overdue / pickup_deadline / last_remark / is_authorized_override
func TestPickupStudents_OutputFields(t *testing.T) {
	// deadline 23:59 → 恒「提前」，故理由必填；is_overdue 恒 false（未过 23:59）
	f := newPickupFixture(t, "23:59", testPickupClass)
	authID := f.makeAuthorization(t, "奶奶（张三）", 1, 1, nil)

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
	_ = authID

	item := f.pickupItem(t, "all")
	assert.Equal(t, "23:59", item["pickup_deadline"], "pickup_deadline 应为班级配置值")
	assert.Equal(t, "家长临时委托", item["last_remark"], "last_remark 应为当日最后一次登记的理由")
	assert.Equal(t, true, item["is_authorized_override"], "is_authorized_override 应反映例外")
	assert.Equal(t, false, item["is_overdue"], "已接走不应标 is_overdue")
	assert.Equal(t, "picked_up", item["pickup_status"])
}

// is_overdue 实时标志：仍在园且已过放学时刻
func TestPickupStudents_IsOverdueLive(t *testing.T) {
	f := newPickupFixture(t, "00:00", testPickupClass) // 恒已过
	_, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
		StudentID: f.studentID, Type: 1,
	}, domain.PickupMethodManual)
	require.NoError(t, err)

	item := f.pickupItem(t, "all")
	assert.Equal(t, true, item["is_overdue"], "仍在园且已过放学时刻应标 is_overdue")
}

// ---------- 辅助 ----------

func (f *pickupFixture) overview(t *testing.T) map[string]any {
	t.Helper()
	resp, err := f.svc.PickupOverview(ctx(), f.tc)
	require.NoError(t, err)
	return resp
}

func (f *pickupFixture) pickupItem(t *testing.T, status string) map[string]any {
	t.Helper()
	resp, err := f.svc.PickupStudents(ctx(), f.tc, status)
	require.NoError(t, err)
	list, ok := resp["list"].([]map[string]any)
	require.True(t, ok, "list 应为 []map[string]any")
	require.Len(t, list, 1, "fixture 只建 1 名学生")
	return list[0]
}

func (f *pickupFixture) pickupStatus(t *testing.T, status string) string {
	t.Helper()
	item := f.pickupItem(t, status)
	s, _ := item["pickup_status"].(string)
	return s
}

func asI64(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case int32:
		return int64(n)
	case float64:
		return int64(n)
	}
	return -1
}

var _ = context.Background
