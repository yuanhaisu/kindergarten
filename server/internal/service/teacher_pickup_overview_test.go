package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
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

// authorized_person_list：只含有效授权人，name/relation 正确且与字符串同序
func TestPickupStudents_AuthorizedPersonList(t *testing.T) {
	f := newPickupFixture(t, "00:00", testPickupClass)

	// 有效：两个不同关系（按 id 升序即创建顺序）
	f.makeAuthorizationWithRelation(t, "奶奶（张三）", "grandmother", 1, 1, nil)
	f.makeAuthorizationWithRelation(t, "爸爸（王明）", "father", 1, 1, nil)
	// 无效：未审核 / 已撤销 / 软删，三种都不得出现在列表里
	f.makeAuthorizationWithRelation(t, "叔叔（王强）", "uncle", 0, 1, nil)
	f.makeAuthorizationWithRelation(t, "阿姨（李梅）", "aunt", 1, 0, nil)
	softDeletedID := f.makeAuthorizationWithRelation(t, "爷爷（王老）", "grandfather", 1, 1, nil)
	require.NoError(t, repository.NewPickupRepo(f.svc.db).
		SoftDeleteAuthorization(ctx(), f.studentID, softDeletedID))

	item := f.pickupItem(t, "all")

	persons, ok := item["authorized_person_list"].([]pickupAuthorizedPerson)
	require.True(t, ok, "authorized_person_list 必须是结构化数组")
	require.Len(t, persons, 2, "未审核/已撤销/软删的授权人都必须被过滤")
	assert.Equal(t, pickupAuthorizedPerson{Name: "奶奶（张三）", Relation: "grandmother"}, persons[0])
	assert.Equal(t, pickupAuthorizedPerson{Name: "爸爸（王明）", Relation: "father"}, persons[1])

	// 线上形状：json tag 决定端上拿到的字段名，逐字节断言，避免改名漏测
	raw, err := json.Marshal(persons)
	require.NoError(t, err)
	t.Logf("authorized_person_list JSON: %s", raw)
	assert.JSONEq(t,
		`[{"name":"奶奶（张三）","relation":"grandmother"},{"name":"爸爸（王明）","relation":"father"}]`,
		string(raw))

	// 兼容字段仍在，且与数组同源同序
	assert.Equal(t, "奶奶（张三）、爸爸（王明）", item["authorized_persons"],
		"authorized_persons 字符串必须与结构化列表同序")
}

// 无有效授权人时：兼容串回落「暂无」，数组为空数组而非 null
func TestPickupStudents_AuthorizedPersonListEmpty(t *testing.T) {
	f := newPickupFixture(t, "00:00", testPickupClass)

	item := f.pickupItem(t, "all")
	persons, ok := item["authorized_person_list"].([]pickupAuthorizedPerson)
	require.True(t, ok, "无授权人也必须返回数组，不能是 null")
	assert.Empty(t, persons)
	assert.Equal(t, "暂无", item["authorized_persons"], "兼容串保持既有回落文案")
}

// 例外接走后 overview.abnormal 计 1；普通登记不计
func TestPickupOverview_OverrideCountedAsAbnormal(t *testing.T) {
	f := newPickupFixture(t, "00:00", testPickupClass)

	_, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
		StudentID: f.studentID, Type: 1,
	}, domain.PickupMethodManual)
	require.NoError(t, err)
	require.Equal(t, int64(0), asI64(f.overview(t)["abnormal"]), "普通送入不置异常")

	_, err = f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
		StudentID: f.studentID, Type: 2,
		IsAuthorizedOverride: true, OverrideName: "邻居王五",
		Remark: "家长临时委托",
	}, domain.PickupMethodManual)
	require.NoError(t, err)

	assert.Equal(t, int64(1), asI64(f.overview(t)["abnormal"]),
		"例外接走必须计入当日异常数")
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
