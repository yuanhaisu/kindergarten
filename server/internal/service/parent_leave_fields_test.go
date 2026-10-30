package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// B14：家长端请假契约字段补齐（backend/api/api-parent/03-leave.md）。
// 覆盖 TC-NEW-LV-0001..0009：类型、附件、审批人/班级、全量统计、撤销守卫、标签、接送人。
// 复用 notice_multitarget_test.go 的夹具（园所 990201、老师 990201、双孩家长 990210）。

// newParentLeaveFixture 在通用夹具上接管 leave_requests / leave_attachments 的迁移与清理。
// 迁移走 Migrator.AddColumn 而非 AutoMigrate：后者会顺带尝试给 belongs-to 关联建外键，
// 开发库里可能已存在孤儿行，建约束会直接失败。
func newParentLeaveFixture(t *testing.T) *nmtFixture {
	t.Helper()
	f := newNMTFixture(t)
	ensureParentLeaveSchema(t, f.db)
	purge := func() {
		if err := f.db.Exec(
			"DELETE FROM leave_attachments WHERE leave_id IN (SELECT id FROM leave_requests WHERE kindergarten_id = ?)",
			testNMTKG).Error; err != nil {
			t.Logf("cleanup leave_attachments: %v", err)
		}
		if err := f.db.Exec("DELETE FROM leave_requests WHERE kindergarten_id = ?", testNMTKG).Error; err != nil {
			t.Logf("cleanup leave_requests: %v", err)
		}
	}
	purge()
	t.Cleanup(purge)
	return f
}

func ensureParentLeaveSchema(t *testing.T, db *gorm.DB) {
	t.Helper()
	m := db.Migrator()
	// GORM v1.31 的 AddColumn 不自己判存在，必须先 HasColumn，否则重复执行报 1060。
	for _, column := range []string{"LeaveType", "EmergencyContact"} {
		if !m.HasColumn(&domain.LeaveRequest{}, column) {
			require.NoError(t, m.AddColumn(&domain.LeaveRequest{}, column))
		}
	}
	if !m.HasTable(&domain.LeaveAttachment{}) {
		require.NoError(t, m.CreateTable(&domain.LeaveAttachment{}))
	}
}

func parentLeaveValidInput() ParentLeaveInput {
	return ParentLeaveInput{
		StudentID: 990220,
		Reason:    "感冒发烧，身体不适",
		StartDate: "2026-03-19",
		EndDate:   "2026-03-21",
	}
}

func createParentLeave(t *testing.T, f *nmtFixture, in ParentLeaveInput) uint64 {
	t.Helper()
	out, err := f.svcP.CreateLeave(context.Background(), f.pcMulti, in)
	require.NoError(t, err)
	id, ok := out["id"].(uint64)
	require.True(t, ok, "创建请假应返回 id")
	return id
}

func auditParentLeave(t *testing.T, f *nmtFixture, id uint64, approve bool, remark string) {
	t.Helper()
	err := f.svcT.AuditLeave(context.Background(), f.tc, id, approve, remark)
	require.NoError(t, err)
}

func parentLeaveListItems(t *testing.T, result map[string]any) map[uint64]map[string]any {
	t.Helper()
	raw, ok := result["list"].([]map[string]any)
	require.True(t, ok)
	out := make(map[uint64]map[string]any, len(raw))
	for _, item := range raw {
		id, ok := item["id"].(uint64)
		require.True(t, ok)
		out[id] = item
	}
	return out
}

func parentLeaveStatsOf(t *testing.T, result map[string]any) map[string]int64 {
	t.Helper()
	stats, ok := result["stats"].(map[string]int64)
	require.True(t, ok, "列表应返回 stats")
	return stats
}

// TC-NEW-LV-0001 / 0002：显式 sick、缺省 other（列表与详情都返回代码与中文标签）。
func TestParentLeaveCreate_LeaveType(t *testing.T) {
	f := newParentLeaveFixture(t)
	ctx := context.Background()

	sickInput := parentLeaveValidInput()
	sickInput.LeaveType = domain.LeaveTypeSick
	sickID := createParentLeave(t, f, sickInput)

	personalInput := parentLeaveValidInput()
	personalInput.LeaveType = domain.LeaveTypePersonal
	personalID := createParentLeave(t, f, personalInput)

	defaultID := createParentLeave(t, f, parentLeaveValidInput())

	want := map[uint64][2]string{
		sickID:     {domain.LeaveTypeSick, "病假"},
		personalID: {domain.LeaveTypePersonal, "事假"},
		defaultID:  {domain.LeaveTypeOther, "其他"},
	}
	for id, pair := range want {
		detail, err := f.svcP.GetLeave(ctx, f.pcMulti, id)
		require.NoError(t, err)
		assert.Equal(t, pair[0], detail["leave_type"], "详情 %d", id)
		assert.Equal(t, pair[1], detail["leave_type_label"], "详情 %d", id)
	}

	result, err := f.svcP.ListLeaves(ctx, f.pcMulti, "", 1, 20)
	require.NoError(t, err)
	items := parentLeaveListItems(t, result)
	require.Len(t, items, 3)
	for id, pair := range want {
		assert.Equal(t, pair[0], items[id]["leave_type"], "列表 %d", id)
		assert.Equal(t, pair[1], items[id]["leave_type_label"], "列表 %d", id)
	}

	bad := parentLeaveValidInput()
	bad.LeaveType = "vacation"
	_, err = f.svcP.CreateLeave(ctx, f.pcMulti, bad)
	require.ErrorIs(t, err, ErrParentInvalidParams, "未知类型整单拒绝")
}

// TC-NEW-LV-0003 / 0004：两张按顺序落库；四张整单拒绝；非图片与 >20MB 同样拒绝。
func TestParentLeaveCreate_Attachments(t *testing.T) {
	f := newParentLeaveFixture(t)
	ctx := context.Background()

	input := parentLeaveValidInput()
	input.Attachments = []ParentLeaveAttachmentInput{
		{FileURL: "https://cdn.example.com/uploads/leave/b14-a.jpg", FileType: "image", FileName: "a.jpg", FileSize: 512000},
		{FileURL: "https://cdn.example.com/uploads/leave/b14-b.jpg", FileType: "image", FileName: "b.jpg", FileSize: 513000},
	}
	id := createParentLeave(t, f, input)

	rows, err := repository.NewLeaveRepo(f.db).ListAttachmentsByLeaveID(ctx, id)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, []string{"a.jpg", "b.jpg"}, []string{rows[0].FileName, rows[1].FileName})
	assert.Equal(t, []int{0, 1}, []int{rows[0].SortNum, rows[1].SortNum})
	assert.Equal(t, int64(512000), rows[0].FileSize)

	detail, err := f.svcP.GetLeave(ctx, f.pcMulti, id)
	require.NoError(t, err)
	attachments, ok := detail["attachments"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, attachments, 2)
	assert.Equal(t, "https://cdn.example.com/uploads/leave/b14-a.jpg", attachments[0]["file_url"])
	assert.Equal(t, "image", attachments[0]["file_type"])
	assert.Equal(t, "a.jpg", attachments[0]["file_name"])
	assert.Equal(t, int64(512000), attachments[0]["file_size"])
	assert.NotZero(t, attachments[0]["id"])
	assert.Equal(t, "b.jpg", attachments[1]["file_name"])

	countLeaves := func() int64 {
		var cnt int64
		require.NoError(t, f.db.Model(&domain.LeaveRequest{}).
			Where("kindergarten_id = ? AND applicant_id = ?", testNMTKG, testNMTParent).Count(&cnt).Error)
		return cnt
	}
	require.Equal(t, int64(1), countLeaves())

	// 4 张：整单拒绝，不产生第二条请假。
	four := parentLeaveValidInput()
	for i := 0; i < 4; i++ {
		four.Attachments = append(four.Attachments, ParentLeaveAttachmentInput{
			FileURL:  fmt.Sprintf("https://cdn.example.com/uploads/leave/b14-%d.jpg", i),
			FileType: "image", FileName: fmt.Sprintf("%d.jpg", i), FileSize: 1024,
		})
	}
	_, err = f.svcP.CreateLeave(ctx, f.pcMulti, four)
	assert.ErrorIs(t, err, ErrParentInvalidParams)
	assert.Equal(t, int64(1), countLeaves(), "4 张附件不应落库")

	// 非图片
	video := parentLeaveValidInput()
	video.Attachments = []ParentLeaveAttachmentInput{{FileURL: "https://cdn.example.com/uploads/leave/b14.mp4", FileType: "video", FileName: "b14.mp4", FileSize: 1024}}
	_, err = f.svcP.CreateLeave(ctx, f.pcMulti, video)
	assert.ErrorIs(t, err, ErrParentInvalidParams)
	assert.Equal(t, int64(1), countLeaves(), "非图片不应落库")

	// 超过 20MB
	tooBig := parentLeaveValidInput()
	tooBig.Attachments = []ParentLeaveAttachmentInput{{FileURL: "https://cdn.example.com/uploads/leave/b14-big.jpg", FileType: "image", FileName: "big.jpg", FileSize: 20<<20 + 1}}
	_, err = f.svcP.CreateLeave(ctx, f.pcMulti, tooBig)
	assert.ErrorIs(t, err, ErrParentInvalidParams)
	assert.Equal(t, int64(1), countLeaves(), "超 20MB 不应落库")

	// 恰好 20MB 允许（边界）
	exact := parentLeaveValidInput()
	exact.Attachments = []ParentLeaveAttachmentInput{{FileURL: "https://cdn.example.com/uploads/leave/b14-exact.jpg", FileType: "image", FileName: "exact.jpg", FileSize: 20 << 20}}
	createParentLeave(t, f, exact)
	assert.Equal(t, int64(2), countLeaves())
}

// TC-NEW-LV-0005：已审批详情返回 reviewer.nickname 与 class_name；列表返回 reviewer.nickname。
func TestParentLeaveDetail_ReviewerAndClass(t *testing.T) {
	f := newParentLeaveFixture(t)
	ctx := context.Background()

	id := createParentLeave(t, f, parentLeaveValidInput())
	auditParentLeave(t, f, id, true, "注意休息，祝早日康复")

	detail, err := f.svcP.GetLeave(ctx, f.pcMulti, id)
	require.NoError(t, err)
	assert.Equal(t, domain.LeaveStatusApproved, detail["status"])
	assert.Equal(t, "已同意", detail["status_label"])
	student, ok := detail["student"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "测试小一班", student["class_name"], "详情返回学生当前班级")

	reviewer, ok := detail["reviewer"].(map[string]any)
	require.True(t, ok, "已审批应返回 reviewer")
	assert.Equal(t, testNMTTeacher, reviewer["id"])
	assert.Equal(t, "测试多班老师", reviewer["nickname"])
	assert.Equal(t, "注意休息，祝早日康复", detail["review_remark"])

	result, err := f.svcP.ListLeaves(ctx, f.pcMulti, "", 1, 20)
	require.NoError(t, err)
	item, ok := parentLeaveListItems(t, result)[id]
	require.True(t, ok)
	listReviewer, ok := item["reviewer"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "测试多班老师", listReviewer["nickname"])
	assert.NotContains(t, listReviewer, "id", "列表 reviewer 只给署名所需的 nickname")

	// 待审批无审批人：列表与详情均为 null。
	pendingID := createParentLeave(t, f, parentLeaveValidInput())
	result, err = f.svcP.ListLeaves(ctx, f.pcMulti, "", 1, 20)
	require.NoError(t, err)
	assert.Nil(t, parentLeaveListItems(t, result)[pendingID]["reviewer"])
	pendingDetail, err := f.svcP.GetLeave(ctx, f.pcMulti, pendingID)
	require.NoError(t, err)
	assert.Nil(t, pendingDetail["reviewer"])
}

// TC-NEW-LV-0006：stats 为全量计数，切换 status 筛选后不变。
func TestParentLeaveList_StatsIndependentOfFilter(t *testing.T) {
	f := newParentLeaveFixture(t)
	ctx := context.Background()

	pendingID := createParentLeave(t, f, parentLeaveValidInput())
	approvedID := createParentLeave(t, f, parentLeaveValidInput())
	auditParentLeave(t, f, approvedID, true, "同意")
	rejectedID := createParentLeave(t, f, parentLeaveValidInput())
	auditParentLeave(t, f, rejectedID, false, "病假凭证不足")
	canceledID := createParentLeave(t, f, parentLeaveValidInput())
	require.NoError(t, f.svcP.CancelLeave(ctx, f.pcMulti, canceledID))

	all, err := f.svcP.ListLeaves(ctx, f.pcMulti, "", 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(4), all["total"])
	assert.Equal(t, map[string]int64{"all": 4, "pending": 1, "approved": 1}, parentLeaveStatsOf(t, all))

	pendingList, err := f.svcP.ListLeaves(ctx, f.pcMulti, "pending", 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(1), pendingList["total"])
	assert.Equal(t, map[string]int64{"all": 4, "pending": 1, "approved": 1}, parentLeaveStatsOf(t, pendingList),
		"筛选待审批后 stats 不变")
	assert.Contains(t, parentLeaveListItems(t, pendingList), pendingID)

	approvedList, err := f.svcP.ListLeaves(ctx, f.pcMulti, "approved", 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(1), approvedList["total"])
	assert.Equal(t, map[string]int64{"all": 4, "pending": 1, "approved": 1}, parentLeaveStatsOf(t, approvedList),
		"筛选已同意后 stats 不变")
}

// TC-NEW-LV-0007：撤销仅 pending；其余三态返回业务错误且状态不变。
func TestParentLeaveCancel_OnlyPending(t *testing.T) {
	f := newParentLeaveFixture(t)
	ctx := context.Background()

	pendingID := createParentLeave(t, f, parentLeaveValidInput())
	require.NoError(t, f.svcP.CancelLeave(ctx, f.pcMulti, pendingID))
	detail, err := f.svcP.GetLeave(ctx, f.pcMulti, pendingID)
	require.NoError(t, err)
	assert.Equal(t, domain.LeaveStatusCanceled, detail["status"])
	assert.Equal(t, "已取消", detail["status_label"])

	canceledAgain := f.svcP.CancelLeave(ctx, f.pcMulti, pendingID)
	assert.ErrorIs(t, canceledAgain, ErrParentLeaveNotCancelable, "已取消不可再次撤销")

	cases := []struct {
		name    string
		prepare func(id uint64)
	}{
		{"approved", func(id uint64) { auditParentLeave(t, f, id, true, "同意") }},
		{"rejected", func(id uint64) { auditParentLeave(t, f, id, false, "事由不充分") }},
		{"canceled", func(id uint64) { require.NoError(t, f.svcP.CancelLeave(ctx, f.pcMulti, id)) }},
	}
	for _, tc := range cases {
		id := createParentLeave(t, f, parentLeaveValidInput())
		tc.prepare(id)
		before, err := f.svcP.GetLeave(ctx, f.pcMulti, id)
		require.NoError(t, err)
		err = f.svcP.CancelLeave(ctx, f.pcMulti, id)
		assert.ErrorIs(t, err, ErrParentLeaveNotCancelable, "%s 状态不可撤销", tc.name)
		after, err := f.svcP.GetLeave(ctx, f.pcMulti, id)
		require.NoError(t, err)
		assert.Equal(t, before["status"], after["status"], "%s 状态不应变化", tc.name)
		assert.Equal(t, before["status_label"], after["status_label"], "%s 标签不应变化", tc.name)
	}

	// 不存在 / 非本人：仍按 404 语义处理。
	assert.ErrorIs(t, f.svcP.CancelLeave(ctx, f.pcMulti, 999999), ErrParentNotFound)
}

// TC-NEW-LV-0008：四种状态的代码与标签在列表与详情一致。
func TestParentLeaveStatusLabels_Unified(t *testing.T) {
	f := newParentLeaveFixture(t)
	ctx := context.Background()

	pendingID := createParentLeave(t, f, parentLeaveValidInput())
	approvedID := createParentLeave(t, f, parentLeaveValidInput())
	auditParentLeave(t, f, approvedID, true, "同意")
	rejectedID := createParentLeave(t, f, parentLeaveValidInput())
	auditParentLeave(t, f, rejectedID, false, "事由不充分")
	canceledID := createParentLeave(t, f, parentLeaveValidInput())
	require.NoError(t, f.svcP.CancelLeave(ctx, f.pcMulti, canceledID))

	want := map[uint64]struct {
		status domain.LeaveStatus
		label  string
	}{
		pendingID:  {domain.LeaveStatusPending, "待审批"},
		approvedID: {domain.LeaveStatusApproved, "已同意"},
		rejectedID: {domain.LeaveStatusRejected, "已拒绝"},
		canceledID: {domain.LeaveStatusCanceled, "已取消"},
	}
	result, err := f.svcP.ListLeaves(ctx, f.pcMulti, "", 1, 20)
	require.NoError(t, err)
	items := parentLeaveListItems(t, result)
	for id, exp := range want {
		item, ok := items[id]
		require.True(t, ok)
		assert.Equal(t, exp.status, item["status"])
		assert.Equal(t, exp.label, item["status_label"])

		detail, err := f.svcP.GetLeave(ctx, f.pcMulti, id)
		require.NoError(t, err)
		assert.Equal(t, exp.status, detail["status"], "详情状态码与列表一致")
		assert.Equal(t, exp.label, detail["status_label"], "详情标签与列表一致")
	}
}

// TC-NEW-LV-0009：接送人选填，创建后可回显；未填写返回 null。
func TestParentLeaveEmergencyContact(t *testing.T) {
	f := newParentLeaveFixture(t)
	ctx := context.Background()

	withContact := parentLeaveValidInput()
	withContact.EmergencyContact = &ParentLeaveContactInput{Name: "王芳", Phone: "13812345678"}
	id := createParentLeave(t, f, withContact)
	detail, err := f.svcP.GetLeave(ctx, f.pcMulti, id)
	require.NoError(t, err)
	contact, ok := detail["emergency_contact"].(map[string]any)
	require.True(t, ok, "填写接送人后详情应返回")
	assert.Equal(t, map[string]any{"name": "王芳", "phone": "13812345678"}, contact)

	withoutID := createParentLeave(t, f, parentLeaveValidInput())
	detail, err = f.svcP.GetLeave(ctx, f.pcMulti, withoutID)
	require.NoError(t, err)
	assert.Nil(t, detail["emergency_contact"], "未填写时返回 null")

	// 只有空白的接送人按未填写处理。
	blank := parentLeaveValidInput()
	blank.EmergencyContact = &ParentLeaveContactInput{Name: "  ", Phone: " "}
	blankID := createParentLeave(t, f, blank)
	detail, err = f.svcP.GetLeave(ctx, f.pcMulti, blankID)
	require.NoError(t, err)
	assert.Nil(t, detail["emergency_contact"])
}

// 老师端请假读写回归：新列不影响老师列表/详情/审批。
func TestParentLeaveTeacherRegression(t *testing.T) {
	f := newParentLeaveFixture(t)
	ctx := context.Background()

	approveID := createParentLeave(t, f, parentLeaveValidInput())
	rejectID := createParentLeave(t, f, parentLeaveValidInput())

	list, err := f.svcT.ListLeaves(ctx, f.tc, "", 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(2), list["total"])
	teacherItems := make(map[uint64]map[string]any)
	for _, raw := range list["list"].([]map[string]any) {
		teacherItems[raw["id"].(uint64)] = raw
	}
	require.Contains(t, teacherItems, approveID)
	assert.Equal(t, "待审批", teacherItems[approveID]["status_label"])

	teacherDetail, err := f.svcT.GetLeave(ctx, f.tc, approveID)
	require.NoError(t, err)
	assert.Equal(t, parentLeaveValidInput().Reason, teacherDetail["reason"])

	auditParentLeave(t, f, approveID, true, "同意")
	auditParentLeave(t, f, rejectID, false, "事由不充分")

	// 老师端按状态筛选仍按审批结果读取（LeaveStats 未纳入：其内部复用
	// 同一 *gorm.DB 连续 Where，GORM v1.31 在 clone=0 时会累积条件，
	// 属基线已有的独立缺陷，不在本切片允许改动范围内）。
	for status, want := range map[string]int64{"pending": 0, "approved": 1, "rejected": 1} {
		filtered, err := f.svcT.ListLeaves(ctx, f.tc, status, 1, 20)
		require.NoError(t, err)
		assert.Equal(t, want, filtered["total"], "老师端 status=%s", status)
	}

	// 老师审批结果在家长详情可读。
	detail, err := f.svcP.GetLeave(ctx, f.pcMulti, approveID)
	require.NoError(t, err)
	assert.Equal(t, "已同意", detail["status_label"])
	detail, err = f.svcP.GetLeave(ctx, f.pcMulti, rejectID)
	require.NoError(t, err)
	assert.Equal(t, "已拒绝", detail["status_label"])
}

// 列表审批人批量查询：查询次数不随行数增长（Count/Find/2 个 Preload/stats/审批人 = 6）。
func TestParentLeaveList_NoNPlusOne(t *testing.T) {
	f := newParentLeaveFixture(t)
	ctx := context.Background()
	for i := 0; i < 20; i++ {
		id := createParentLeave(t, f, parentLeaveValidInput())
		auditParentLeave(t, f, id, true, "同意")
	}

	counter := &noticeQueryCounter{}
	svc := newParentPortalServiceWithDB(f.db.Session(&gorm.Session{Logger: counter}))

	_, err := svc.ListLeaves(ctx, f.pcMulti, "", 1, 20)
	require.NoError(t, err)
	fullPage := counter.value()
	t.Logf("整页 20 条的查询数：%d", fullPage)
	assert.LessOrEqual(t, fullPage, int64(7), "一页 20 条应为常数次查询")

	counter.reset()
	_, err = svc.ListLeaves(ctx, f.pcMulti, "", 1, 1)
	require.NoError(t, err)
	singleRow := counter.value()
	t.Logf("单条页的查询数：%d", singleRow)
	assert.Equal(t, fullPage, singleRow, "整页与单条查询数相同 —— 审批人没有逐条查询")
}

// 清理证据：硬删后 leave_requests / leave_attachments 0 残留（含软删行）。
func TestParentLeaveFields_CleanupLeavesNoResidue(t *testing.T) {
	f := newParentLeaveFixture(t)
	ctx := context.Background()

	input := parentLeaveValidInput()
	input.Attachments = []ParentLeaveAttachmentInput{
		{FileURL: "https://cdn.example.com/uploads/leave/b14-cleanup-a.jpg", FileType: "image", FileName: "a.jpg", FileSize: 1024},
	}
	id := createParentLeave(t, f, input)
	require.NoError(t, f.svcP.CancelLeave(ctx, f.pcMulti, id))

	// 与夹具同一套清理：附件先硬删，再硬删请假。
	require.NoError(t, f.db.Exec(
		"DELETE FROM leave_attachments WHERE leave_id IN (SELECT id FROM leave_requests WHERE kindergarten_id = ?)",
		testNMTKG).Error)
	require.NoError(t, f.db.Exec("DELETE FROM leave_requests WHERE kindergarten_id = ?", testNMTKG).Error)

	var leaves, attachments int64
	require.NoError(t, f.db.Unscoped().Model(&domain.LeaveRequest{}).
		Where("kindergarten_id = ?", testNMTKG).Count(&leaves).Error)
	require.NoError(t, f.db.Unscoped().Model(&domain.LeaveAttachment{}).
		Where("leave_id = ?", id).Count(&attachments).Error)
	assert.Zero(t, leaves, "请假应清零")
	assert.Zero(t, attachments, "附件应清零")
}
