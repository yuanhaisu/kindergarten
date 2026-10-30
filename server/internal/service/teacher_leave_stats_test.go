package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
)

// B19：老师端请假统计（GET /teacher/leaves/stats）。
// 回归背景：LeaveStats 曾把 pending/approved/rejected/monthly 四个计数挂在同一条
// *gorm.DB 链上，GORM 的链式 Where 会原地追加条件，后一个计数继承了前一个 status，
// 导致 approved/rejected/monthly_total 恒为 0。此用例用固定 9944xx 夹具复现并锁定口径。
//
// 覆盖：
//   - pending/approved/rejected 为该老师班级的累计数；
//   - monthly_total 只算 created_at 本月（上月那条不计）；
//   - 软删行不计入；
//   - 同园别班、他园数据不混入；空班返回 0。

const (
	testTLSKG      uint64 = 994401
	testTLSKGOther uint64 = 994402

	testTLSClassA uint64 = 994410 // 目标班
	testTLSClassB uint64 = 994411 // 同园别班
	testTLSClassO uint64 = 994412 // 他园班

	testTLSUser     uint64 = 994420
	testTLSStudentA uint64 = 994430
	testTLSStudentB uint64 = 994431
	testTLSStudentO uint64 = 994432

	testTLSLeavePending   uint64 = 994440 // 目标班 本月 pending
	testTLSLeaveApproved  uint64 = 994441 // 目标班 本月 approved
	testTLSLeaveRejected  uint64 = 994442 // 目标班 本月 rejected
	testTLSLeaveLastMonth uint64 = 994443 // 目标班 上月 pending
	testTLSLeaveDeleted   uint64 = 994444 // 目标班 本月 pending，软删
	testTLSLeaveBPending  uint64 = 994445 // 别班 pending
	testTLSLeaveBApproved uint64 = 994446 // 别班 approved
	testTLSLeaveOPending  uint64 = 994447 // 他园 pending
	testTLSLeaveOApproved uint64 = 994448 // 他园 approved
	testTLSLeaveORejected uint64 = 994449 // 他园 rejected
)

func cleanupTLSData(t *testing.T, db *gorm.DB) {
	t.Helper()
	del := func(sql string, args ...any) {
		if err := db.Exec(sql, args...).Error; err != nil {
			t.Logf("cleanup: %v", err)
		}
	}
	// 裸 SQL 硬删，避免软删行占住唯一索引/污染后续用例计数。子表先删。
	del("DELETE FROM leave_attachments WHERE leave_id BETWEEN 994440 AND 994449")
	del("DELETE FROM leave_requests WHERE id BETWEEN 994440 AND 994449")
	del("DELETE FROM students WHERE id BETWEEN 994430 AND 994439")
	del("DELETE FROM classes WHERE id BETWEEN 994410 AND 994419")
	del("DELETE FROM users WHERE id BETWEEN 994420 AND 994429")
	del("DELETE FROM kindergartens WHERE id IN (994401, 994402)")
}

type tlsFixture struct {
	db        *gorm.DB
	svc       *TeacherPortalService
	tcA       *TeacherPortalContext
	tcB       *TeacherPortalContext
	tcOther   *TeacherPortalContext
	tcNoClass *TeacherPortalContext
}

func newTLSFixture(t *testing.T) *tlsFixture {
	t.Helper()
	db := openCLPTestDB(t)
	cleanupTLSData(t, db)
	t.Cleanup(func() { cleanupTLSData(t, db) })

	validDate := time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC)
	now := time.Now()
	monthStart := teacherPortalDay(now).AddDate(0, 0, -now.Day()+1)
	lastMonth := monthStart.Add(-time.Hour)

	require.NoError(t, db.Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: testTLSKG}, Name: "测试园所-请假统计",
	}).Error)
	require.NoError(t, db.Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: testTLSKGOther}, Name: "测试园所-请假统计他园",
	}).Error)

	for _, cl := range []struct {
		id   uint64
		kg   uint64
		name string
	}{
		{testTLSClassA, testTLSKG, "请假统计一班"},
		{testTLSClassB, testTLSKG, "请假统计二班"},
		{testTLSClassO, testTLSKGOther, "他园班级"},
	} {
		require.NoError(t, db.Create(&domain.Class{
			BaseModel: domain.BaseModel{ID: cl.id}, KindergartenID: cl.kg, Name: cl.name, Grade: "小班", Status: 1,
		}).Error)
	}

	require.NoError(t, db.Create(&domain.User{
		BaseModel: domain.BaseModel{ID: testTLSUser}, Nickname: "请假统计家长", Phone: "13900004401", PasswordHash: "x",
	}).Error)
	for _, st := range []struct {
		id      uint64
		classID uint64
		kg      uint64
	}{
		{testTLSStudentA, testTLSClassA, testTLSKG},
		{testTLSStudentB, testTLSClassB, testTLSKG},
		{testTLSStudentO, testTLSClassO, testTLSKGOther},
	} {
		require.NoError(t, db.Create(&domain.Student{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: st.id}},
			KindergartenID:  st.kg, ClassID: st.classID, Name: "请假统计学生",
			Birthday: validDate, EnrollDate: validDate, Status: 1,
		}).Error)
	}

	mkLeave := func(id, classID, kg, studentID uint64, status domain.LeaveStatus, createdAt time.Time, deleted bool) {
		lr := domain.LeaveRequest{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: id, CreatedAt: createdAt}},
			KindergartenID:  kg,
			ClassID:         classID,
			StudentID:       studentID,
			ApplicantID:     testTLSUser,
			Reason:          "请假统计用例",
			StartDate:       validDate,
			EndDate:         validDate,
			Days:            1,
			Status:          status,
		}
		if deleted {
			lr.DeletedAt = gorm.DeletedAt{Time: now, Valid: true}
		}
		require.NoError(t, db.Create(&lr).Error)
	}

	// 目标班：本月 pending/approved/rejected 各 1，上月 pending 1，软删 pending 1。
	mkLeave(testTLSLeavePending, testTLSClassA, testTLSKG, testTLSStudentA, domain.LeaveStatusPending, now, false)
	mkLeave(testTLSLeaveApproved, testTLSClassA, testTLSKG, testTLSStudentA, domain.LeaveStatusApproved, now, false)
	mkLeave(testTLSLeaveRejected, testTLSClassA, testTLSKG, testTLSStudentA, domain.LeaveStatusRejected, now, false)
	mkLeave(testTLSLeaveLastMonth, testTLSClassA, testTLSKG, testTLSStudentA, domain.LeaveStatusPending, lastMonth, false)
	mkLeave(testTLSLeaveDeleted, testTLSClassA, testTLSKG, testTLSStudentA, domain.LeaveStatusPending, now, true)
	// 同园别班。
	mkLeave(testTLSLeaveBPending, testTLSClassB, testTLSKG, testTLSStudentB, domain.LeaveStatusPending, now, false)
	mkLeave(testTLSLeaveBApproved, testTLSClassB, testTLSKG, testTLSStudentB, domain.LeaveStatusApproved, now, false)
	// 他园。
	mkLeave(testTLSLeaveOPending, testTLSClassO, testTLSKGOther, testTLSStudentO, domain.LeaveStatusPending, now, false)
	mkLeave(testTLSLeaveOApproved, testTLSClassO, testTLSKGOther, testTLSStudentO, domain.LeaveStatusApproved, now, false)
	mkLeave(testTLSLeaveORejected, testTLSClassO, testTLSKGOther, testTLSStudentO, domain.LeaveStatusRejected, now, false)

	f := &tlsFixture{db: db}
	f.svc = NewTeacherPortalService(db, nil, nil, nil, nil, nil, config.FaceConfig{})
	f.tcA = &TeacherPortalContext{UserID: testTLSUser, KindergartenID: testTLSKG, ClassID: testTLSClassA}
	f.tcB = &TeacherPortalContext{UserID: testTLSUser, KindergartenID: testTLSKG, ClassID: testTLSClassB}
	f.tcOther = &TeacherPortalContext{UserID: testTLSUser, KindergartenID: testTLSKGOther, ClassID: testTLSClassO}
	f.tcNoClass = &TeacherPortalContext{UserID: testTLSUser, KindergartenID: testTLSKG, ClassID: 994419}
	return f
}

func tlsStats(t *testing.T, svc *TeacherPortalService, tc *TeacherPortalContext) map[string]any {
	t.Helper()
	stats, err := svc.LeaveStats(context.Background(), tc)
	require.NoError(t, err)
	return stats
}

// 四计数口径：状态累计 + 本月 created_at；软删与上月行不计。
func TestTeacherLeaveStatsCounts(t *testing.T) {
	f := newTLSFixture(t)
	stats := tlsStats(t, f.svc, f.tcA)

	assert.EqualValues(t, 2, stats["pending"], "本月 + 上月 pending 累计；软删那条不计")
	assert.EqualValues(t, 1, stats["approved"])
	assert.EqualValues(t, 1, stats["rejected"])
	assert.EqualValues(t, 3, stats["monthly_total"], "只算本月：pending/approved/rejected 各 1，上月 pending 不计")
}

// 班级与园所隔离：只统计当前老师所带班级。
func TestTeacherLeaveStatsClassIsolation(t *testing.T) {
	f := newTLSFixture(t)

	statsB := tlsStats(t, f.svc, f.tcB)
	assert.EqualValues(t, 1, statsB["pending"], "别班数据不得串入目标班")
	assert.EqualValues(t, 1, statsB["approved"])
	assert.EqualValues(t, 0, statsB["rejected"])
	assert.EqualValues(t, 2, statsB["monthly_total"])

	statsO := tlsStats(t, f.svc, f.tcOther)
	assert.EqualValues(t, 1, statsO["pending"])
	assert.EqualValues(t, 1, statsO["approved"])
	assert.EqualValues(t, 1, statsO["rejected"])
	assert.EqualValues(t, 3, statsO["monthly_total"])

	empty := tlsStats(t, f.svc, f.tcNoClass)
	assert.EqualValues(t, 0, empty["pending"])
	assert.EqualValues(t, 0, empty["approved"])
	assert.EqualValues(t, 0, empty["rejected"])
	assert.EqualValues(t, 0, empty["monthly_total"])
}
