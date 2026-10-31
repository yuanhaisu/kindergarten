package service

// 家长成长报告四格统计（OpenSpec growth-report-stats）回归。
//
// growth_reports 目前没有写入方，报告页此前永远落在空态。这里用真实库夹具钉住四格口径：
// 出勤率分母 = 班级实际产生考勤记录的天数（迟到算出席、请假进分母不进分子）；积分只算
// 正向流水；任务分子限定在分母任务集合内；成长记录必须走可见性谓词；孩子之间互不混入。

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

// 独立 ID 段，避开其它切片已用的 990xxx / 991xxx / 992xxx。
const (
	testReportKG       uint64 = 990501
	testReportClassA   uint64 = 990502
	testReportClassB   uint64 = 990503
	testReportStudentA uint64 = 990504
	testReportStudentB uint64 = 990505
	testReportStudentC uint64 = 990506
	testReportParent   uint64 = 990507
	testReportTeacher  uint64 = 990508
	testReportYear     uint64 = 990509
	testReportRow      uint64 = 990510
	testReportDim1     uint64 = 990511
	testReportDim2     uint64 = 990512
	testReportRowPrev  uint64 = 990513 // 上一周期的同类型正文
	testReportDim3     uint64 = 990514
	testReportDim4     uint64 = 990515
	testReportLogBase  uint64 = 990520 // 22 条 log：17 可见 + 5 不可见/周期外
	testReportTaskBase uint64 = 990560 // 12 条周期内任务，后接 2 条周期外、2 条他人任务
)

func reportDay(y int, m time.Month, day int) time.Time {
	return time.Date(y, m, day, 0, 0, 0, 0, time.Local)
}

func reportMoment(y int, m time.Month, day, hour int) time.Time {
	return time.Date(y, m, day, hour, 0, 0, 0, time.Local)
}

// cleanGrowthReportFixture 按 ID/园所硬删。软删行会占住唯一索引并污染重跑，一律 Unscoped。
func cleanGrowthReportFixture(db *gorm.DB) {
	studentIDs := []uint64{testReportStudentA, testReportStudentB, testReportStudentC}
	reportIDs := []uint64{testReportRow, testReportRowPrev}
	db.Unscoped().Where("report_id IN ?", reportIDs).Delete(&domain.GrowthReportDimension{})
	db.Unscoped().Where("id IN ?", reportIDs).Delete(&domain.GrowthReport{})
	db.Unscoped().Where("student_id IN ?", studentIDs).Delete(&domain.GrowthReport{})
	db.Unscoped().Where("kindergarten_id = ?", testReportKG).Delete(&domain.GrowthDailyLog{})
	db.Unscoped().Where("kindergarten_id = ?", testReportKG).Delete(&domain.Attendance{})
	db.Unscoped().Where("student_id IN ?", studentIDs).Delete(&domain.PointTransaction{})
	db.Unscoped().Where("student_id IN ?", studentIDs).Delete(&domain.TaskCheckIn{})
	db.Unscoped().Where("kindergarten_id = ?", testReportKG).Delete(&domain.ParentTask{})
	db.Unscoped().Where("kindergarten_id = ?", testReportKG).Delete(&domain.AcademicYear{})
	db.Unscoped().Where("student_id IN ?", studentIDs).Delete(&domain.StudentParent{})
	db.Unscoped().Where("user_id = ? AND kindergarten_id = ?", testReportParent, testReportKG).Delete(&domain.UserRole{})
	db.Unscoped().Where("id IN ?", studentIDs).Delete(&domain.Student{})
	db.Unscoped().Where("id IN ?", []uint64{testReportClassA, testReportClassB}).Delete(&domain.Class{})
	db.Unscoped().Where("id IN ?", []uint64{testReportParent, testReportTeacher}).Delete(&domain.User{})
	db.Unscoped().Where("id = ?", testReportKG).Delete(&domain.Kindergarten{})
}

func setupGrowthReportFixture(t *testing.T) (*gorm.DB, *ParentPortalService) {
	t.Helper()
	db := openHomeFeedTestDB(t)
	cleanGrowthReportFixture(db)
	t.Cleanup(func() { cleanGrowthReportFixture(db) })

	require.NoError(t, db.Create(&domain.User{
		BaseModel: domain.BaseModel{ID: testReportParent}, Nickname: "报告回归家长", PasswordHash: "x",
	}).Error)
	require.NoError(t, db.Create(&domain.User{
		BaseModel: domain.BaseModel{ID: testReportTeacher}, Nickname: "报告回归老师", PasswordHash: "x",
	}).Error)
	require.NoError(t, db.Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: testReportKG}, Name: "测试园所-报告回归",
	}).Error)
	for _, cls := range []domain.Class{
		{BaseModel: domain.BaseModel{ID: testReportClassA}, KindergartenID: testReportKG, Name: "报告回归A班", Grade: "小班", Status: 1},
		{BaseModel: domain.BaseModel{ID: testReportClassB}, KindergartenID: testReportKG, Name: "报告回归B班", Grade: "大班", Status: 1},
	} {
		require.NoError(t, db.Create(&cls).Error)
	}

	// 开发库是严格模式：生日/入园日必须给真值，否则零值时间会被 MySQL 拒绝。
	birthday := reportDay(2020, 3, 1)
	for _, st := range []domain.Student{
		{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testReportStudentA}},
			KindergartenID: testReportKG, ClassID: testReportClassA, Name: "报告学生A", Birthday: birthday, EnrollDate: birthday, Status: 1},
		{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testReportStudentB}},
			KindergartenID: testReportKG, ClassID: testReportClassB, Name: "报告学生B", Birthday: birthday, EnrollDate: birthday, Status: 1},
		// C 只用来制造「班级当天有考勤、A 没有」的日子，验证分母按班级天数算。
		{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testReportStudentC}},
			KindergartenID: testReportKG, ClassID: testReportClassA, Name: "报告学生C", Birthday: birthday, EnrollDate: birthday, Status: 1},
	} {
		require.NoError(t, db.Create(&st).Error)
	}
	for _, sp := range []domain.StudentParent{
		{StudentID: testReportStudentA, UserID: testReportParent, Relation: domain.RelationMother, IsPrimary: true},
		{StudentID: testReportStudentB, UserID: testReportParent, Relation: domain.RelationFather},
	} {
		require.NoError(t, db.Create(&sp).Error)
	}
	kgID := testReportKG
	require.NoError(t, db.Create(&domain.UserRole{
		UserID: testReportParent, Role: "parent", KindergartenID: &kgID,
	}).Error)

	attendance := []domain.Attendance{
		// 3-16 那一周：A 出勤 + 迟到 + 请假，班级共 3 天有考勤 → 2/3 = 67%。
		{KindergartenID: testReportKG, ClassID: testReportClassA, StudentID: testReportStudentA,
			Date: reportDay(2026, 3, 16), Status: domain.AttendancePresent},
		{KindergartenID: testReportKG, ClassID: testReportClassA, StudentID: testReportStudentA,
			Date: reportDay(2026, 3, 17), Status: domain.AttendanceLate},
		{KindergartenID: testReportKG, ClassID: testReportClassA, StudentID: testReportStudentA,
			Date: reportDay(2026, 3, 18), Status: domain.AttendanceLeave},
		// 3-09 那一周：A 只出勤 1 天，班级另有一天只有 C 的考勤 → 1/2 = 50%。
		{KindergartenID: testReportKG, ClassID: testReportClassA, StudentID: testReportStudentA,
			Date: reportDay(2026, 3, 9), Status: domain.AttendancePresent},
		{KindergartenID: testReportKG, ClassID: testReportClassA, StudentID: testReportStudentC,
			Date: reportDay(2026, 3, 10), Status: domain.AttendancePresent},
		// 学期内、但不在上面任何一周的考勤，供 term 口径断言。
		{KindergartenID: testReportKG, ClassID: testReportClassA, StudentID: testReportStudentA,
			Date: reportDay(2026, 3, 2), Status: domain.AttendancePresent},
		// 学期外的缺勤日：缺勤进分母、不进分子，也不污染 term 统计。
		{KindergartenID: testReportKG, ClassID: testReportClassA, StudentID: testReportStudentA,
			Date: reportDay(2026, 2, 23), Status: domain.AttendanceAbsent},
		// B 班的考勤与 A 的统计无关。
		{KindergartenID: testReportKG, ClassID: testReportClassB, StudentID: testReportStudentB,
			Date: reportDay(2026, 3, 16), Status: domain.AttendancePresent},
	}
	for _, row := range attendance {
		require.NoError(t, db.Create(&row).Error)
	}

	for _, tx := range []domain.PointTransaction{
		{BaseModel: domain.BaseModel{CreatedAt: reportMoment(2026, 3, 16, 10)}, StudentID: testReportStudentA, Points: 30, Source: "task", Balance: 30},
		{BaseModel: domain.BaseModel{CreatedAt: reportMoment(2026, 3, 17, 10)}, StudentID: testReportStudentA, Points: 15, Source: "task", Balance: 45},
		{BaseModel: domain.BaseModel{CreatedAt: reportMoment(2026, 3, 18, 10)}, StudentID: testReportStudentA, Points: -10, Source: "redeem", Balance: 35},
		// 周期外（但学期内）的流水：只验证周报不计入。
		{BaseModel: domain.BaseModel{CreatedAt: reportMoment(2026, 3, 2, 10)}, StudentID: testReportStudentA, Points: 100, Source: "task", Balance: 135},
		{BaseModel: domain.BaseModel{CreatedAt: reportMoment(2026, 3, 16, 10)}, StudentID: testReportStudentB, Points: 7, Source: "task", Balance: 7},
	} {
		require.NoError(t, db.Create(&tx).Error)
	}

	// 12 条周期内班级任务；周期外 2 条；B 班 1 条；只发给 B 的个体任务 1 条（挂在 A 班下）。
	for i := 0; i < 12; i++ {
		require.NoError(t, db.Create(&domain.ParentTask{
			BaseModel:      domain.BaseModel{ID: testReportTaskBase + uint64(i)},
			KindergartenID: testReportKG, ClassID: testReportClassA, CreatorID: testReportTeacher,
			Title: "报告回归任务", Frequency: domain.TaskFrequencyOnce,
			StartDate: reportDay(2026, 3, 16), EndDate: reportDay(2026, 3, 22), Status: 1,
		}).Error)
	}
	for i := 0; i < 2; i++ {
		require.NoError(t, db.Create(&domain.ParentTask{
			BaseModel:      domain.BaseModel{ID: testReportTaskBase + 12 + uint64(i)},
			KindergartenID: testReportKG, ClassID: testReportClassA, CreatorID: testReportTeacher,
			Title: "报告回归周期外任务", Frequency: domain.TaskFrequencyOnce,
			StartDate: reportDay(2026, 2, 2), EndDate: reportDay(2026, 2, 8), Status: 1,
		}).Error)
	}
	require.NoError(t, db.Create(&domain.ParentTask{
		BaseModel:      domain.BaseModel{ID: testReportTaskBase + 14},
		KindergartenID: testReportKG, ClassID: testReportClassB, CreatorID: testReportTeacher,
		Title: "报告回归B班任务", Frequency: domain.TaskFrequencyOnce,
		StartDate: reportDay(2026, 3, 16), EndDate: reportDay(2026, 3, 22), Status: 1,
	}).Error)
	require.NoError(t, db.Create(&domain.ParentTask{
		BaseModel:       domain.BaseModel{ID: testReportTaskBase + 15},
		TargetStudentID: testReportStudentB,
		KindergartenID:  testReportKG, ClassID: testReportClassA, CreatorID: testReportTeacher,
		Title: "报告回归B个体任务", Frequency: domain.TaskFrequencyOnce,
		StartDate: reportDay(2026, 3, 16), EndDate: reportDay(2026, 3, 22), Status: 1,
	}).Error)

	// A 打卡 8 条周期内任务；再给周期外任务打一次卡，验证分子不越出分母集合。
	for i := 0; i < 8; i++ {
		require.NoError(t, db.Create(&domain.TaskCheckIn{
			TaskID: testReportTaskBase + uint64(i), StudentID: testReportStudentA, UserID: testReportParent,
			Date: reportDay(2026, 3, 18), Content: "已打卡",
		}).Error)
	}
	require.NoError(t, db.Create(&domain.TaskCheckIn{
		TaskID: testReportTaskBase + 12, StudentID: testReportStudentA, UserID: testReportParent,
		Date: reportDay(2026, 3, 18), Content: "周期外任务打卡",
	}).Error)
	require.NoError(t, db.Create(&domain.TaskCheckIn{
		TaskID: testReportTaskBase + 14, StudentID: testReportStudentB, UserID: testReportParent,
		Date: reportDay(2026, 3, 18), Content: "B 班打卡",
	}).Error)

	// 15 条 A 可见（全班）记录在周期内；2 条周期外；3 条个体记录未点名 A（不可见）；
	// 2 条 B 班全班记录（对 A 不可见）。
	logs := make([]domain.GrowthDailyLog, 0, 22)
	for i := 0; i < 15; i++ {
		logs = append(logs, domain.GrowthDailyLog{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testReportLogBase + uint64(i)}},
			KindergartenID:  testReportKG, ClassID: testReportClassA, CreatorID: testReportTeacher,
			Date: reportDay(2026, 3, 16).AddDate(0, 0, i%5), TargetType: 1, Content: "A 班可见记录",
		})
	}
	for i := 0; i < 2; i++ {
		logs = append(logs, domain.GrowthDailyLog{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testReportLogBase + 15 + uint64(i)}},
			KindergartenID:  testReportKG, ClassID: testReportClassA, CreatorID: testReportTeacher,
			Date: reportDay(2026, 2, 10), TargetType: 1, Content: "周期外记录",
		})
	}
	for i := 0; i < 3; i++ {
		logs = append(logs, domain.GrowthDailyLog{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testReportLogBase + 17 + uint64(i)}},
			KindergartenID:  testReportKG, ClassID: testReportClassA, CreatorID: testReportTeacher,
			Date: reportDay(2026, 3, 17), TargetType: 2, Content: "未点名 A 的个体记录",
		})
	}
	for i := 0; i < 2; i++ {
		logs = append(logs, domain.GrowthDailyLog{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testReportLogBase + 20 + uint64(i)}},
			KindergartenID:  testReportKG, ClassID: testReportClassB, CreatorID: testReportTeacher,
			Date: reportDay(2026, 3, 17), TargetType: 1, Content: "B 班记录",
		})
	}
	for _, row := range logs {
		require.NoError(t, db.Create(&row).Error)
	}

	require.NoError(t, db.Create(&domain.AcademicYear{
		BaseModel: domain.BaseModel{ID: testReportYear}, KindergartenID: testReportKG, Year: "2025-2026",
		SemesterSpringStart: "2026-03-01", SemesterSpringEnd: "2026-06-30",
		SemesterFallStart: "2025-09-01", SemesterFallEnd: "2026-01-31",
		IsCurrent: true,
	}).Error)

	svc := NewParentPortalService(db,
		repository.NewParentRepo(db), repository.NewUserRoleRepo(db),
		repository.NewStudentRepo(db), repository.NewClassRepo(db),
		repository.NewPickupRepo(db), repository.NewCalendarRepo(db),
		repository.NewNoticeRepo(db))
	return db, svc
}

func resolveReportPC(t *testing.T, svc *ParentPortalService, studentID uint64) *ParentPortalContext {
	t.Helper()
	pc, err := svc.ResolveContext(context.Background(), testReportParent, testReportKG, &studentID)
	require.NoError(t, err)
	return pc
}

func reportIntField(t *testing.T, data map[string]any, key string) int64 {
	t.Helper()
	v, ok := data[key].(int64)
	require.True(t, ok, "%s 应为 int64，实际 %T", key, data[key])
	return v
}

func reportAttendanceField(t *testing.T, data map[string]any, key string) int64 {
	t.Helper()
	att, ok := data["attendance"].(map[string]any)
	require.True(t, ok, "响应必须含 attendance 对象")
	v, ok := att[key].(int64)
	require.True(t, ok, "attendance.%s 应为 int64，实际 %T", key, att[key])
	return v
}

func reportPeriodMap(t *testing.T, data map[string]any) map[string]any {
	t.Helper()
	period, ok := data["period"].(map[string]any)
	require.True(t, ok, "响应必须含 period 对象")
	return period
}

// TC-NEW-GRP-0001：周期内没有报告正文行时，四格统计仍要返回。
func TestParentGrowthReport_StatsWithoutReportRow(t *testing.T) {
	_, svc := setupGrowthReportFixture(t)
	pc := resolveReportPC(t, svc, testReportStudentA)

	data, err := svc.GetGrowthReport(context.Background(), pc, 0, "weekly", "2026-03-18")
	require.NoError(t, err)

	_, hasReport := data["report"]
	assert.False(t, hasReport, "周期内无报告正文时不应返回 report 键")
	dims, ok := data["dimensions"].([]domain.GrowthReportDimension)
	require.True(t, ok, "dimensions 应为数组而不是 null")
	assert.Empty(t, dims)

	student, ok := data["student"].(map[string]any)
	require.True(t, ok, "响应必须含 student")
	assert.Equal(t, testReportStudentA, student["id"])
	assert.Equal(t, "报告学生A", student["name"])

	period := reportPeriodMap(t, data)
	assert.Equal(t, "weekly", period["type"])
	assert.Equal(t, "2026-03-16", period["start_date"])
	assert.Equal(t, "2026-03-22", period["end_date"])
}

// TC-NEW-GRP-0002：有报告正文行时，正文/维度原样返回，四格照常聚合。
func TestParentGrowthReport_WithReportRowKeepsReportAndDimensions(t *testing.T) {
	db, svc := setupGrowthReportFixture(t)
	require.NoError(t, db.Create(&domain.GrowthReport{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testReportRow}},
		KindergartenID:  testReportKG, StudentID: testReportStudentA, ReportType: "weekly",
		PeriodStart: reportDay(2026, 3, 16), PeriodEnd: reportDay(2026, 3, 22),
		Content: "周报正文", GeneratedAt: reportMoment(2026, 3, 22, 18), Status: 1,
	}).Error)
	for _, dim := range []domain.GrowthReportDimension{
		{BaseModel: domain.BaseModel{ID: testReportDim1}, ReportID: testReportRow, DimName: "语言", Score: 85, MaxScore: 100},
		{BaseModel: domain.BaseModel{ID: testReportDim2}, ReportID: testReportRow, DimName: "运动", Score: 72, MaxScore: 100},
	} {
		require.NoError(t, db.Create(&dim).Error)
	}

	pc := resolveReportPC(t, svc, testReportStudentA)
	data, err := svc.GetGrowthReport(context.Background(), pc, 0, "weekly", "2026-03-18")
	require.NoError(t, err)

	report, ok := data["report"].(domain.GrowthReport)
	require.True(t, ok, "有正文行时必须返回 report")
	assert.Equal(t, testReportRow, report.ID)
	dims, ok := data["dimensions"].([]domain.GrowthReportDimension)
	require.True(t, ok)
	assert.Len(t, dims, 2)

	// 正文行不影响四格口径。
	assert.Equal(t, int64(45), reportIntField(t, data, "points_earned"))
	assert.Equal(t, int64(15), reportIntField(t, data, "growth_record_count"))
}

// TC-NEW-GRP-0010：同类型正文分属两个周期时，正文必须与请求周期严格对应。
// 上周报告的 created_at 刻意更新：旧实现只按 report_type + created_at DESC 取最新，
// 会把上周正文返回给本周请求。
func TestParentGrowthReport_BodyMatchesRequestedPeriod(t *testing.T) {
	db, svc := setupGrowthReportFixture(t)

	require.NoError(t, db.Create(&domain.GrowthReport{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{
			ID: testReportRowPrev, CreatedAt: reportMoment(2026, 3, 23, 9),
		}},
		KindergartenID: testReportKG, StudentID: testReportStudentA, ReportType: "weekly",
		PeriodStart: reportDay(2026, 3, 9), PeriodEnd: reportDay(2026, 3, 15),
		Content: "上周正文", GeneratedAt: reportMoment(2026, 3, 23, 9), Status: 1,
	}).Error)
	require.NoError(t, db.Create(&domain.GrowthReport{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{
			ID: testReportRow, CreatedAt: reportMoment(2026, 3, 22, 9),
		}},
		KindergartenID: testReportKG, StudentID: testReportStudentA, ReportType: "weekly",
		PeriodStart: reportDay(2026, 3, 16), PeriodEnd: reportDay(2026, 3, 22),
		Content: "本周正文", GeneratedAt: reportMoment(2026, 3, 22, 9), Status: 1,
	}).Error)
	require.NoError(t, db.Create(&domain.GrowthReportDimension{
		BaseModel: domain.BaseModel{ID: testReportDim3}, ReportID: testReportRow, DimName: "语言", Score: 85, MaxScore: 100,
	}).Error)
	require.NoError(t, db.Create(&domain.GrowthReportDimension{
		BaseModel: domain.BaseModel{ID: testReportDim4}, ReportID: testReportRowPrev, DimName: "运动", Score: 70, MaxScore: 100,
	}).Error)

	pc := resolveReportPC(t, svc, testReportStudentA)
	ctx := context.Background()

	cur, err := svc.GetGrowthReport(ctx, pc, 0, "weekly", "2026-03-18")
	require.NoError(t, err)
	curReport, ok := cur["report"].(domain.GrowthReport)
	require.True(t, ok, "本周请求必须返回正文")
	assert.Equal(t, testReportRow, curReport.ID, "即使上周正文更新，也只能返回本周正文")
	assert.Equal(t, "本周正文", curReport.Content)
	assert.True(t, curReport.PeriodStart.Equal(reportDay(2026, 3, 16)))
	curPeriod := reportPeriodMap(t, cur)
	assert.Equal(t, "2026-03-16", curPeriod["start_date"])
	assert.Equal(t, "2026-03-22", curPeriod["end_date"])
	assert.Equal(t, curPeriod["start_date"], curReport.PeriodStart.Format("2006-01-02"), "正文周期必须与四格 period 一致")
	curDims, ok := cur["dimensions"].([]domain.GrowthReportDimension)
	require.True(t, ok)
	require.Len(t, curDims, 1)
	assert.Equal(t, "语言", curDims[0].DimName, "维度必须跟随正文所属报告")

	prev, err := svc.GetGrowthReport(ctx, pc, 0, "weekly", "2026-03-11")
	require.NoError(t, err)
	prevReport, ok := prev["report"].(domain.GrowthReport)
	require.True(t, ok, "上周请求必须返回正文")
	assert.Equal(t, testReportRowPrev, prevReport.ID)
	assert.Equal(t, "上周正文", prevReport.Content)
	assert.True(t, prevReport.PeriodStart.Equal(reportDay(2026, 3, 9)))
	prevPeriod := reportPeriodMap(t, prev)
	assert.Equal(t, "2026-03-09", prevPeriod["start_date"])
	assert.Equal(t, "2026-03-15", prevPeriod["end_date"])
	assert.Equal(t, prevPeriod["start_date"], prevReport.PeriodStart.Format("2006-01-02"))
	prevDims, ok := prev["dimensions"].([]domain.GrowthReportDimension)
	require.True(t, ok)
	require.Len(t, prevDims, 1)
	assert.Equal(t, "运动", prevDims[0].DimName)
}

// TC-NEW-GRP-0011：请求的周期没有正文（其它周期有）时不得跨周期取正文；
// dimensions 为空数组，四格统计照常返回。
func TestParentGrowthReport_PeriodWithoutBodyKeepsStats(t *testing.T) {
	db, svc := setupGrowthReportFixture(t)
	require.NoError(t, db.Create(&domain.GrowthReport{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testReportRow}},
		KindergartenID:  testReportKG, StudentID: testReportStudentA, ReportType: "weekly",
		PeriodStart: reportDay(2026, 3, 16), PeriodEnd: reportDay(2026, 3, 22),
		Content: "本周正文", GeneratedAt: reportMoment(2026, 3, 22, 18), Status: 1,
	}).Error)

	pc := resolveReportPC(t, svc, testReportStudentA)
	// 2-23 周：夹具里只有一条 A 的缺勤考勤；正文只存在于 3-16 周。
	data, err := svc.GetGrowthReport(context.Background(), pc, 0, "weekly", "2026-02-23")
	require.NoError(t, err)

	_, hasReport := data["report"]
	assert.False(t, hasReport, "无正文的周期不得返回其它周期的正文")
	dims, ok := data["dimensions"].([]domain.GrowthReportDimension)
	require.True(t, ok, "dimensions 应为数组而不是 null")
	assert.Empty(t, dims)

	period := reportPeriodMap(t, data)
	assert.Equal(t, "2026-02-23", period["start_date"])
	assert.Equal(t, "2026-03-01", period["end_date"])
	assert.Equal(t, int64(0), reportAttendanceField(t, data, "present_days"))
	assert.Equal(t, int64(1), reportAttendanceField(t, data, "total_days"), "四格不依赖正文，仍要返回")
}

// TC-NEW-GRP-0003：出勤率分母 = 班级有考勤记录的天数；迟到算出勤；请假进分母不进分子。
func TestParentGrowthReport_AttendanceUsesClassDays(t *testing.T) {
	_, svc := setupGrowthReportFixture(t)
	pc := resolveReportPC(t, svc, testReportStudentA)
	ctx := context.Background()

	data, err := svc.GetGrowthReport(ctx, pc, 0, "weekly", "2026-03-18")
	require.NoError(t, err)
	assert.Equal(t, int64(2), reportAttendanceField(t, data, "present_days"), "迟到应计入出勤")
	assert.Equal(t, int64(1), reportAttendanceField(t, data, "leave_days"))
	assert.Equal(t, int64(3), reportAttendanceField(t, data, "total_days"))
	assert.Equal(t, "67%", data["attendance"].(map[string]any)["rate"])

	// 上一周 A 只出勤一天；班里另有一天只有 C 的考勤，分母仍按班级天数算 2。
	prev, err := svc.GetGrowthReport(ctx, pc, 0, "weekly", "2026-03-09")
	require.NoError(t, err)
	assert.Equal(t, int64(1), reportAttendanceField(t, prev, "present_days"))
	assert.Equal(t, int64(2), reportAttendanceField(t, prev, "total_days"), "分母应按班级考勤天数，不按孩子自己的记录数")
	assert.Equal(t, "50%", prev["attendance"].(map[string]any)["rate"])

	// 缺勤日计入分母、不计入出勤（该周只有这一天有考勤）。
	absent, err := svc.GetGrowthReport(ctx, pc, 0, "weekly", "2026-02-23")
	require.NoError(t, err)
	assert.Equal(t, int64(0), reportAttendanceField(t, absent, "present_days"))
	assert.Equal(t, int64(1), reportAttendanceField(t, absent, "total_days"))
	assert.Equal(t, "0%", absent["attendance"].(map[string]any)["rate"])
}

// TC-NEW-GRP-0004：获得积分只算正向流水。
func TestParentGrowthReport_PointsOnlyPositive(t *testing.T) {
	_, svc := setupGrowthReportFixture(t)
	pc := resolveReportPC(t, svc, testReportStudentA)

	data, err := svc.GetGrowthReport(context.Background(), pc, 0, "weekly", "2026-03-18")
	require.NoError(t, err)
	assert.Equal(t, int64(45), reportIntField(t, data, "points_earned"), "+30 +15 -10 应为 45")
}

// TC-NEW-GRP-0005：完成任务 = 分母任务集合内的打卡条数 / 周期内发布任务条数。
func TestParentGrowthReport_TasksCompletedOverTotal(t *testing.T) {
	_, svc := setupGrowthReportFixture(t)
	pc := resolveReportPC(t, svc, testReportStudentA)

	data, err := svc.GetGrowthReport(context.Background(), pc, 0, "weekly", "2026-03-18")
	require.NoError(t, err)
	assert.Equal(t, int64(8), reportIntField(t, data, "tasks_completed"))
	assert.Equal(t, int64(12), reportIntField(t, data, "tasks_total"))
}

// TC-NEW-GRP-0006：成长记录只数可见记录，周期外的与未点名的都不计入。
func TestParentGrowthReport_GrowthRecordsVisibleOnly(t *testing.T) {
	_, svc := setupGrowthReportFixture(t)
	pc := resolveReportPC(t, svc, testReportStudentA)

	data, err := svc.GetGrowthReport(context.Background(), pc, 0, "weekly", "2026-03-18")
	require.NoError(t, err)
	assert.Equal(t, int64(15), reportIntField(t, data, "growth_record_count"))
}

// TC-NEW-GRP-0007：非法 type/date 与 term 无学年配置都返回参数错误；term 命中配置时用学期区间。
func TestParentGrowthReport_RejectsInvalidPeriod(t *testing.T) {
	_, svc := setupGrowthReportFixture(t)
	pc := resolveReportPC(t, svc, testReportStudentA)
	ctx := context.Background()

	for _, tc := range []struct {
		name, typ, date string
	}{
		{"不支持的周期", "daily", "2026-03-18"},
		{"空周期", "", "2026-03-18"},
		{"非法日期", "weekly", "2026-13-40"},
		{"term 无学年配置", "term", "2020-05-01"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.GetGrowthReport(ctx, pc, 0, tc.typ, tc.date)
			assert.ErrorIs(t, err, ErrParentReportPeriod)
		})
	}
}

// term 命中 academic_years 时按春季区间统计（3-01 ~ 6-30）。
func TestParentGrowthReport_TermUsesAcademicYearConfig(t *testing.T) {
	_, svc := setupGrowthReportFixture(t)
	pc := resolveReportPC(t, svc, testReportStudentA)

	data, err := svc.GetGrowthReport(context.Background(), pc, 0, "term", "2026-04-15")
	require.NoError(t, err)

	period := reportPeriodMap(t, data)
	assert.Equal(t, "term", period["type"])
	assert.Equal(t, "2026-03-01", period["start_date"])
	assert.Equal(t, "2026-06-30", period["end_date"])

	// 学期口径下 3-02 的 +100 积分与 3-02 的考勤都进入统计。
	assert.Equal(t, int64(145), reportIntField(t, data, "points_earned"))
	assert.Equal(t, int64(4), reportAttendanceField(t, data, "present_days"))
	assert.Equal(t, int64(6), reportAttendanceField(t, data, "total_days"))
}

// TC-NEW-GRP-0008：同一家长的两个孩子统计互不混入。
func TestParentGrowthReport_IsolatesStudents(t *testing.T) {
	_, svc := setupGrowthReportFixture(t)
	ctx := context.Background()

	dataA, err := svc.GetGrowthReport(ctx, resolveReportPC(t, svc, testReportStudentA), 0, "weekly", "2026-03-18")
	require.NoError(t, err)
	dataB, err := svc.GetGrowthReport(ctx, resolveReportPC(t, svc, testReportStudentB), 0, "weekly", "2026-03-18")
	require.NoError(t, err)

	assert.Equal(t, int64(45), reportIntField(t, dataA, "points_earned"))
	assert.Equal(t, int64(7), reportIntField(t, dataB, "points_earned"), "A 的正向积分不得混入 B")
	assert.Equal(t, int64(15), reportIntField(t, dataA, "growth_record_count"))
	assert.Equal(t, int64(2), reportIntField(t, dataB, "growth_record_count"), "B 班记录不得混入 A，A 班记录也不得混入 B")
	assert.Equal(t, int64(8), reportIntField(t, dataA, "tasks_completed"))
	assert.Equal(t, int64(1), reportIntField(t, dataB, "tasks_completed"))
	assert.Equal(t, int64(12), reportIntField(t, dataA, "tasks_total"), "只发给 B 的个体任务不得进入 A 的分母")
	assert.Equal(t, int64(2), reportIntField(t, dataB, "tasks_total"))
	assert.Equal(t, int64(3), reportAttendanceField(t, dataA, "total_days"))
	assert.Equal(t, int64(1), reportAttendanceField(t, dataB, "total_days"))
}

// TC-NEW-GRP-0009：周期内没有任何数据时返回零值成功，不报错。
func TestParentGrowthReport_ZeroValues(t *testing.T) {
	_, svc := setupGrowthReportFixture(t)
	pc := resolveReportPC(t, svc, testReportStudentA)

	data, err := svc.GetGrowthReport(context.Background(), pc, 0, "weekly", "2025-12-03")
	require.NoError(t, err)

	assert.Equal(t, int64(0), reportAttendanceField(t, data, "present_days"))
	assert.Equal(t, int64(0), reportAttendanceField(t, data, "leave_days"))
	assert.Equal(t, int64(0), reportAttendanceField(t, data, "total_days"))
	assert.Equal(t, "0%", data["attendance"].(map[string]any)["rate"])
	assert.Equal(t, int64(0), reportIntField(t, data, "points_earned"))
	assert.Equal(t, int64(0), reportIntField(t, data, "tasks_completed"))
	assert.Equal(t, int64(0), reportIntField(t, data, "tasks_total"))
	assert.Equal(t, int64(0), reportIntField(t, data, "growth_record_count"))

	// date 缺省时按今天所在周成功返回（不静默报错）。
	emptyDate, err := svc.GetGrowthReport(context.Background(), pc, 0, "weekly", "")
	require.NoError(t, err)
	start, err := time.ParseInLocation("2006-01-02", reportPeriodMap(t, emptyDate)["start_date"].(string), time.Local)
	require.NoError(t, err)
	assert.Equal(t, time.Monday, start.Weekday(), "weekly 周期应从周一开始")
}

// 夹具清理必须 0 残留：软删行会占住唯一索引并让下一次重跑撞键。
func TestParentGrowthReportFixture_CleanupLeavesNoResidue(t *testing.T) {
	db, _ := setupGrowthReportFixture(t)
	cleanGrowthReportFixture(db)

	studentIDs := []uint64{testReportStudentA, testReportStudentB, testReportStudentC}
	for name, q := range map[string]*gorm.DB{
		"kindergartens":            db.Unscoped().Model(&domain.Kindergarten{}).Where("id = ?", testReportKG),
		"classes":                  db.Unscoped().Model(&domain.Class{}).Where("id IN ?", []uint64{testReportClassA, testReportClassB}),
		"students":                 db.Unscoped().Model(&domain.Student{}).Where("id IN ?", studentIDs),
		"users":                    db.Unscoped().Model(&domain.User{}).Where("id IN ?", []uint64{testReportParent, testReportTeacher}),
		"student_parents":          db.Unscoped().Model(&domain.StudentParent{}).Where("student_id IN ?", studentIDs),
		"user_roles":               db.Unscoped().Model(&domain.UserRole{}).Where("user_id = ? AND kindergarten_id = ?", testReportParent, testReportKG),
		"attendances":              db.Unscoped().Model(&domain.Attendance{}).Where("kindergarten_id = ?", testReportKG),
		"point_transactions":       db.Unscoped().Model(&domain.PointTransaction{}).Where("student_id IN ?", studentIDs),
		"parent_tasks":             db.Unscoped().Model(&domain.ParentTask{}).Where("kindergarten_id = ?", testReportKG),
		"task_check_ins":           db.Unscoped().Model(&domain.TaskCheckIn{}).Where("student_id IN ?", studentIDs),
		"growth_daily_logs":        db.Unscoped().Model(&domain.GrowthDailyLog{}).Where("kindergarten_id = ?", testReportKG),
		"growth_reports":           db.Unscoped().Model(&domain.GrowthReport{}).Where("student_id IN ?", studentIDs),
		"growth_report_dimensions": db.Unscoped().Model(&domain.GrowthReportDimension{}).Where("report_id IN ?", []uint64{testReportRow, testReportRowPrev}),
		"academic_years":           db.Unscoped().Model(&domain.AcademicYear{}).Where("kindergarten_id = ?", testReportKG),
	} {
		var n int64
		require.NoError(t, q.Count(&n).Error, name)
		assert.Zero(t, n, "%s 清理后应为 0 行", name)
	}
}
