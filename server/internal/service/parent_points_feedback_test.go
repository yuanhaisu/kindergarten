package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/domain"
)

// B16：家长端积分概览/排行榜/流水与反馈契约字段。
// 契约见 backend/api/api-parent/07-points.md、10-feedback.md。
// 复用 notice_multitarget_test.go 的夹具（园所 990201、班级 990202/990203、双孩家长 990210），
// 本文件的数据全部落在 9902xx 区间，t.Cleanup 硬删，0 残留。

const (
	testPFStudentCurrent uint64 = 990220 // 夹具里的小一班孩子（当前孩子）
	testPFStudentRival   uint64 = 990230 // 本周 20 分，班内第一
	testPFStudentTieA    uint64 = 990231 // 本周 15 分，与当前孩子并列
	testPFStudentZero    uint64 = 990232 // 无流水，仍在榜（0 分）
	testPFStudentTieB    uint64 = 990233 // 本周 15 分，与当前孩子并列
	testPFStudentGrad    uint64 = 990234 // 已毕业（status=2），不计入班级口径
	testPFRangeLow       uint64 = 990200
	testPFRangeHigh      uint64 = 990300
)

type parentPointsFixture struct {
	*nmtFixture
	purge func()
}

// newParentPointsFixture 在通用夹具上补充积分/反馈数据。
// 清理必须先于 nmt 的 cleanup 执行：Cleanup 是 LIFO，后注册的先跑。
func newParentPointsFixture(t *testing.T) *parentPointsFixture {
	t.Helper()
	// 共享夹具用固定 ID 重建班级/学生；开发库里若有别的切片留下的、被外键引用的
	// 测试残留（B14 的 leave_requests 曾在 990201 留下 4 行），students 无法硬删，
	// 重建会撞 classes 主键。这里先清测试区间内的外键依赖，再走通用夹具。
	db := openNMTTestDB(t)
	for _, stmt := range []string{
		"DELETE FROM leave_attachments WHERE leave_id IN (SELECT id FROM leave_requests WHERE kindergarten_id = ?)",
		"DELETE FROM leave_requests WHERE kindergarten_id = ?",
	} {
		if err := db.Exec(stmt, testNMTKG).Error; err != nil {
			t.Logf("pre-clean nmt fixture: %v", err)
		}
	}
	f := newNMTFixture(t)
	ctx := context.Background()

	purge := func() {
		if err := f.db.Unscoped().Where("student_id >= ? AND student_id < ?", testPFRangeLow, testPFRangeHigh).
			Delete(&domain.PointTransaction{}).Error; err != nil {
			t.Logf("cleanup point_transactions: %v", err)
		}
		if err := f.db.Unscoped().Where("student_id >= ? AND student_id < ?", testPFRangeLow, testPFRangeHigh).
			Delete(&domain.PointAccount{}).Error; err != nil {
			t.Logf("cleanup point_accounts: %v", err)
		}
		if err := f.db.Unscoped().Where("kindergarten_id = ?", testNMTKG).
			Delete(&domain.Feedback{}).Error; err != nil {
			t.Logf("cleanup feedbacks: %v", err)
		}
	}
	purge()
	t.Cleanup(purge)

	validDate := time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC)
	for _, e := range []struct {
		id     uint64
		name   string
		status int8
	}{
		{testPFStudentRival, "排行竞争对手", 1},
		{testPFStudentTieA, "并列学生A", 1},
		{testPFStudentZero, "零积分学生", 1},
		{testPFStudentTieB, "并列学生B", 1},
		{testPFStudentGrad, "已毕业学生", 2},
	} {
		require.NoError(t, f.db.WithContext(ctx).Create(&domain.Student{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: e.id}},
			KindergartenID:  testNMTKG, ClassID: f.classA, Name: e.name,
			Avatar:   fmt.Sprintf("https://cdn.example.com/avatars/%d.png", e.id),
			Birthday: validDate, EnrollDate: validDate, Status: e.status,
		}).Error)
	}
	// 夹具里的当前孩子建得更早，补头像供「嵌套 student」断言。
	require.NoError(t, f.db.WithContext(ctx).Model(&domain.Student{}).
		Where("id = ?", testPFStudentCurrent).
		Update("avatar", "https://cdn.example.com/avatars/990220.png").Error)
	// 手写的 pc.Students 只有 ID/班级；生产里 ResolveContext 会带出完整学生行，
	// 概览的 student 展示字段取自上下文，这里补齐成与真实解析一致。
	f.pcMulti.Students[0].Name = "测试学生990220"
	f.pcMulti.Students[0].Avatar = "https://cdn.example.com/avatars/990220.png"

	require.NoError(t, f.db.WithContext(ctx).Create(&domain.PointAccount{
		StudentID: testPFStudentCurrent, TotalPoints: 100, UsedPoints: 20,
	}).Error)

	now := time.Now()
	monthFrom, _ := monthBoundsShanghai(now)
	addTx := func(studentID uint64, points, balance int, source, desc string, at time.Time) {
		t.Helper()
		require.NoError(t, f.db.WithContext(ctx).Create(&domain.PointTransaction{
			BaseModel: domain.BaseModel{CreatedAt: at},
			StudentID: studentID, Points: points, Balance: balance,
			Source: source, Description: desc,
		}).Error)
	}
	// 本周 + 本月：一进一进一消耗，余额链 10 → 15 → 12。
	addTx(testPFStudentCurrent, 10, 10, "task_checkin", "完成任务【每日阅读10分钟】", now)
	addTx(testPFStudentCurrent, 5, 15, "admin_adjust", "管理员调整", now)
	addTx(testPFStudentCurrent, -3, 12, "redeem", "兑换小礼物", now)
	// 上月流水：monthFrom 前 7 天必然早于本周一，故本周/本月都不该计入。
	addTx(testPFStudentCurrent, 100, 112, "task_checkin", "上月奖励", monthFrom.AddDate(0, 0, -7))
	addTx(testPFStudentRival, 20, 20, "task_checkin", "竞争对手本周积分", now)
	addTx(testPFStudentTieA, 15, 15, "growth_featured", "并列学生A本周积分", now)
	addTx(testPFStudentTieB, 15, 15, "full_attendance_week", "并列学生B本周积分", now)
	// 别班学生与已毕业学生：都不该进入本班排名/人数。
	addTx(990221, 1000, 1000, "task_checkin", "别班学生积分", now)
	addTx(testPFStudentGrad, 1000, 1000, "task_checkin", "已毕业学生积分", now)

	// ResolveContext 才填 Class；测试直接构造上下文，需自己补上。
	f.pcMulti.Class = &domain.Class{
		BaseModel:      domain.BaseModel{ID: f.classA},
		KindergartenID: testNMTKG, Name: "测试小一班", Status: 1,
	}
	return &parentPointsFixture{nmtFixture: f, purge: purge}
}

// pcForStudent 构造一个指向指定孩子的家长上下文（Class 与孩子班级一致）。
func pcForStudent(f *nmtFixture, studentID, classID uint64) *ParentPortalContext {
	pc := &ParentPortalContext{
		UserID: testNMTParent, KindergartenID: testNMTKG,
		Students: []domain.Student{{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: studentID}},
			ClassID:         classID,
		}},
	}
	pc.CurrentStudent = &pc.Students[0]
	pc.Class = &domain.Class{
		BaseModel:      domain.BaseModel{ID: classID},
		KindergartenID: testNMTKG, Name: "测试班级", Status: 1,
	}
	return pc
}

// 概览：排名（含并列）、class_total、this_month 真实、next_level_points=0。
func TestParentPointsOverview_RankAndMonthly(t *testing.T) {
	f := newParentPointsFixture(t)
	ctx := context.Background()

	out, err := f.svcP.PointsOverview(ctx, f.pcMulti)
	require.NoError(t, err)

	assert.Equal(t, 80, out["total_points"], "100 累计 - 20 已用")
	assert.Equal(t, 1, out["level"])
	assert.Equal(t, "新手上路", out["level_name"])
	assert.Equal(t, "", out["level_icon"], "无等级模型：图标暂未启用")
	assert.Equal(t, int64(0), out["next_level_points"], "无等级模型：进度恒 0，前端隐藏进度条")

	// 990230 20 分第一；990220/990231/990233 同 15 分并列第 2（取最小名次）；990232 0 分第 5。
	assert.Equal(t, 2, out["rank_in_class"])
	assert.Equal(t, 5, out["class_total"], "在读 5 人；已毕业的 990234 不计入")

	// 本月/本周只算正向积分：10 + 5；上月 +100 与 -3 都不计入。
	assert.Equal(t, int64(15), out["this_week"])
	assert.Equal(t, int64(15), out["this_month"])

	student, ok := out["student"].(map[string]any)
	require.True(t, ok, "概览需返回 student 嵌套对象")
	assert.Equal(t, testPFStudentCurrent, student["id"])
	assert.Equal(t, "测试学生990220", student["name"])
	assert.Equal(t, "https://cdn.example.com/avatars/990220.png", student["avatar"])
}

// 排行：嵌套形状、is_self、class_total、同分按 id 升序且并列取最小名次、空班。
func TestParentLeaderboard_NestedShapeAndTies(t *testing.T) {
	f := newParentPointsFixture(t)
	ctx := context.Background()

	out, err := f.svcP.Leaderboard(ctx, f.pcMulti)
	require.NoError(t, err)

	assert.Equal(t, "weekly", out["period"])
	assert.Equal(t, 5, out["class_total"])
	assert.Equal(t, 2, out["my_rank"])
	assert.Equal(t, int64(15), out["my_points"])

	list, ok := out["list"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, list, 5, "零积分学生也在榜内")

	first := list[0]
	assert.Equal(t, 1, first["rank"])
	assert.Equal(t, int64(20), first["points"])
	assert.Equal(t, false, first["is_self"])
	firstStudent, ok := first["student"].(map[string]any)
	require.True(t, ok, "榜单行必须是嵌套 student")
	assert.Equal(t, testPFStudentRival, firstStudent["id"])
	assert.Equal(t, "排行竞争对手", firstStudent["name"])
	assert.Equal(t, "https://cdn.example.com/avatars/990230.png", firstStudent["avatar"])

	// 同 15 分：排名并列第 2，行序按学生 id 升序。
	for i, wantID := range []uint64{testPFStudentCurrent, testPFStudentTieA, testPFStudentTieB} {
		row := list[1+i]
		assert.Equal(t, 2, row["rank"], "并列取最小名次")
		assert.Equal(t, int64(15), row["points"])
		stu := row["student"].(map[string]any)
		assert.Equal(t, wantID, stu["id"], "同分按 id 升序")
	}
	assert.Equal(t, true, list[1]["is_self"], "当前孩子高亮")
	assert.Equal(t, false, list[2]["is_self"])

	last := list[4]
	assert.Equal(t, 5, last["rank"], "0 分学生排在并列组之后")
	assert.Equal(t, int64(0), last["points"])
	assert.Equal(t, testPFStudentZero, last["student"].(map[string]any)["id"])

	for _, row := range list {
		assert.NotEqual(t, testPFStudentGrad, row["student"].(map[string]any)["id"], "已毕业学生不进排名")
	}

	// 空班：有班级但没有任何在读学生 → 空数组，不报错。
	out, err = f.svcP.Leaderboard(ctx, pcForStudent(f.nmtFixture, testPFStudentCurrent, 999999))
	require.NoError(t, err)
	assert.Equal(t, "weekly", out["period"])
	assert.Equal(t, 0, out["class_total"])
	assert.Equal(t, 0, out["my_rank"])
	assert.Equal(t, int64(0), out["my_points"])
	emptyList, ok := out["list"].([]map[string]any)
	require.True(t, ok)
	assert.Len(t, emptyList, 0)
}

// 排行只返回前 20 名，但 class_total 与名次按全班计算。
func TestParentLeaderboard_Top20Limit(t *testing.T) {
	f := newParentPointsFixture(t)
	ctx := context.Background()
	now := time.Now()

	for i := 0; i < 24; i++ {
		id := uint64(990240 + i)
		require.NoError(t, f.db.Create(&domain.Student{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: id}},
			KindergartenID:  testNMTKG, ClassID: f.classB, Name: fmt.Sprintf("榜单学生%d", id), Status: 1,
			Birthday:   time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC),
			EnrollDate: time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC),
		}).Error)
		require.NoError(t, f.db.Create(&domain.PointTransaction{
			BaseModel: domain.BaseModel{CreatedAt: now},
			StudentID: id, Points: 100 - i, Balance: 100 - i, Source: "task_checkin",
		}).Error)
	}

	pcB := pcForStudent(f.nmtFixture, 990221, f.classB)
	out, err := f.svcP.Leaderboard(ctx, pcB)
	require.NoError(t, err)

	assert.Equal(t, 25, out["class_total"], "990221 + 24 名新增学生")
	assert.Equal(t, 1, out["my_rank"], "990221 本周 1000 分")
	list := out["list"].([]map[string]any)
	require.Len(t, list, 20, "只返回前 20 名")
	assert.Equal(t, uint64(990221), list[0]["student"].(map[string]any)["id"])
	assert.Equal(t, uint64(990258), list[19]["student"].(map[string]any)["id"], "第 20 名是 990240+18")
}

// 流水：type 过滤、amount 符号、balance、分页 total、source_label。
func TestParentPointsHistory_ContractShapeAndFilter(t *testing.T) {
	f := newParentPointsFixture(t)
	ctx := context.Background()

	all, err := f.svcP.PointsHistory(ctx, f.pcMulti, "", 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(4), all["total"], "当前孩子 4 笔；别班学生的 +20 不混入")

	list := all["list"].([]map[string]any)
	require.Len(t, list, 4)
	// 同一时刻写入按 id 倒序：先 -3，再 +5，最后 +10；上月那笔排最后。
	spend := list[0]
	assert.Equal(t, "spend", spend["type"])
	assert.Equal(t, "消耗", spend["type_label"])
	assert.Equal(t, -3, spend["amount"], "amount 带符号")
	assert.Equal(t, 12, spend["balance"], "balance 为该笔后余额")
	assert.Equal(t, "redeem", spend["source"])
	assert.Equal(t, "redeem", spend["source_label"], "未知 source 原样返回")
	assert.Equal(t, "兑换小礼物", spend["description"])
	assert.Equal(t, uint64(0), spend["reference_id"])
	_, err = time.Parse(time.RFC3339, spend["created_at"].(string))
	assert.NoError(t, err, "created_at 为 RFC3339")

	earn := list[1]
	assert.Equal(t, "earn", earn["type"])
	assert.Equal(t, "获得", earn["type_label"])
	assert.Equal(t, 5, earn["amount"])
	assert.Equal(t, 15, earn["balance"])
	assert.Equal(t, "admin_adjust", earn["source"])
	assert.Equal(t, "管理员调整", earn["source_label"])

	oldest := list[3]
	assert.Equal(t, 100, oldest["amount"])
	assert.Equal(t, "task_checkin", oldest["source"])
	assert.Equal(t, "任务打卡", oldest["source_label"])

	earnOnly, err := f.svcP.PointsHistory(ctx, f.pcMulti, "earn", 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(3), earnOnly["total"], "+10 / +5 / 上月 +100")
	earnList := earnOnly["list"].([]map[string]any)
	require.Len(t, earnList, 3)
	for _, row := range earnList {
		assert.Equal(t, "earn", row["type"])
		assert.Positive(t, row["amount"].(int))
	}

	spendOnly, err := f.svcP.PointsHistory(ctx, f.pcMulti, "spend", 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(1), spendOnly["total"])
	assert.Equal(t, "spend", spendOnly["list"].([]map[string]any)[0]["type"])

	allKeyword, err := f.svcP.PointsHistory(ctx, f.pcMulti, "all", 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(4), allKeyword["total"])

	page1, err := f.svcP.PointsHistory(ctx, f.pcMulti, "", 1, 2)
	require.NoError(t, err)
	page2, err := f.svcP.PointsHistory(ctx, f.pcMulti, "", 2, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(4), page1["total"], "分页 total 是全量")
	assert.Equal(t, int64(4), page2["total"])
	rows1 := page1["list"].([]map[string]any)
	rows2 := page2["list"].([]map[string]any)
	require.Len(t, rows1, 2)
	require.Len(t, rows2, 2)
	assert.NotEqual(t, rows1[0]["id"], rows2[0]["id"], "两页不重复")

	// 无流水的学生：list 是 [] 而不是 null。
	zero, err := f.svcP.PointsHistory(ctx, pcForStudent(f.nmtFixture, testPFStudentZero, f.classA), "", 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(0), zero["total"])
	zeroList, ok := zero["list"].([]map[string]any)
	require.True(t, ok)
	assert.Len(t, zeroList, 0)
}

// 反馈：category_label 三类与未知回退、replied_at null/有值、状态标签、多家长隔离。
func TestParentFeedback_LabelsAndRepliedAt(t *testing.T) {
	f := newParentPointsFixture(t)
	ctx := context.Background()

	create := func(pc *ParentPortalContext, category, content string) uint64 {
		t.Helper()
		out, err := f.svcP.CreateFeedback(ctx, pc, ParentFeedbackInput{Category: category, Content: content})
		require.NoError(t, err)
		id, ok := out["id"].(uint64)
		require.True(t, ok)
		return id
	}

	ids := map[string]uint64{
		"suggestion": create(f.pcMulti, "suggestion", "希望成长日志里能看到孩子的饮食情况评分"),
		"complaint":  create(f.pcMulti, "complaint", "饭菜温度不够"),
		"bug":        create(f.pcMulti, "bug", "通知详情打不开"),
		"empty":      create(f.pcMulti, "", "没选类型"),
		"praise":     create(f.pcMulti, "praise", "老师很负责"),
	}
	create(f.pcMiss, "suggestion", "别家孩子的反馈")

	// 仓库暂无管理端回复入口，回复态直接落库（与设计 notes 遗留 2 一致）。
	const repliedAt = "2026-10-20T10:00:00Z"
	require.NoError(t, f.db.Model(&domain.Feedback{}).Where("id = ?", ids["complaint"]).
		Updates(map[string]any{"status": 1, "reply": "感谢您的反馈", "replied_at": repliedAt}).Error)
	require.NoError(t, f.db.Model(&domain.Feedback{}).Where("id = ?", ids["bug"]).
		Update("status", 2).Error)

	out, err := f.svcP.ListFeedback(ctx, f.pcMulti, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(5), out["total"], "只含当前家长，别家家长的反馈不混入")

	items := make(map[uint64]map[string]any)
	for _, item := range out["list"].([]map[string]any) {
		items[item["id"].(uint64)] = item
	}

	assert.Equal(t, "建议", items[ids["suggestion"]]["category_label"])
	assert.Equal(t, "投诉", items[ids["complaint"]]["category_label"])
	assert.Equal(t, "问题", items[ids["bug"]]["category_label"])
	assert.Equal(t, "", items[ids["empty"]]["category_label"], "空分类回退原值")
	assert.Equal(t, "praise", items[ids["praise"]]["category_label"], "未知分类回退原值")

	assert.Nil(t, items[ids["suggestion"]]["replied_at"], "无回复为 null")
	assert.Nil(t, items[ids["bug"]]["replied_at"])
	assert.Equal(t, repliedAt, items[ids["complaint"]]["replied_at"])
	assert.Equal(t, "感谢您的反馈", items[ids["complaint"]]["reply"])

	assert.Equal(t, "已提交", items[ids["suggestion"]]["status_label"])
	assert.Equal(t, "已回复", items[ids["complaint"]]["status_label"])
	assert.Equal(t, "已关闭", items[ids["bug"]]["status_label"])

	// 内容必填；未知分类原样落库（契约允许 category 可空）。
	_, err = f.svcP.CreateFeedback(ctx, f.pcMulti, ParentFeedbackInput{Category: "suggestion", Content: "   "})
	assert.ErrorIs(t, err, ErrParentInvalidParams)

	var row domain.Feedback
	require.NoError(t, f.db.First(&row, ids["praise"]).Error)
	assert.Equal(t, "praise", row.Category)
}

// 清理证据：硬删后积分流水/账户/反馈 0 残留（含软删行）。
func TestParentPointsFeedback_CleanupLeavesNoResidue(t *testing.T) {
	f := newParentPointsFixture(t)
	ctx := context.Background()

	_, err := f.svcP.CreateFeedback(ctx, f.pcMulti, ParentFeedbackInput{Category: "suggestion", Content: "清理测试"})
	require.NoError(t, err)

	f.purge()

	count := func(model any, where string, args ...any) int64 {
		var n int64
		require.NoError(t, f.db.Unscoped().Model(model).Where(where, args...).Count(&n).Error)
		return n
	}
	assert.Zero(t, count(&domain.PointTransaction{}, "student_id >= ? AND student_id < ?", testPFRangeLow, testPFRangeHigh))
	assert.Zero(t, count(&domain.PointAccount{}, "student_id >= ? AND student_id < ?", testPFRangeLow, testPFRangeHigh))
	assert.Zero(t, count(&domain.Feedback{}, "kindergarten_id = ?", testNMTKG))
}
