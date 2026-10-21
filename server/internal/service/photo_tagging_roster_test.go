package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

// ── 9.x D11：当日到场名单（present-students）与一键全班（class-roster）──────
//
// 测试用隔离班级 ptRosterClass：students.class_id / growth_daily_logs.class_id
// 有外键约束，不能随便编造 class id，必须落在一张真实 classes 行上；
// 用合成班级避免真实班级的在读学生混入名单（保证断言是精确集合而非包含关系）。

const ptRosterClass = uint64(ptTestBase + 500)

// ptRosterCleanup 逆 FK 依赖顺序清理：attendance/leave → media_students/media/log → students → class。
// 全部硬删（Unscoped）：软删行仍占唯一索引，上次失败运行留下的行会让本轮 Create 直接撞键。
func ptRosterCleanup(t *testing.T, db *gorm.DB) {
	t.Helper()
	db.Unscoped().Where("class_id = ?", ptRosterClass).Delete(&domain.Attendance{})
	db.Unscoped().Where("student_id BETWEEN ? AND ?", ptTestBase+500, ptTestBase+600).Delete(&domain.LeaveRequest{})
	db.Unscoped().Where("media_id BETWEEN ? AND ?", ptTestBase+500, ptTestBase+600).Delete(&domain.GrowthMediaStudent{})
	db.Unscoped().Where("media_id BETWEEN ? AND ?", ptTestBase+500, ptTestBase+600).Delete(&domain.RecognitionJob{})
	db.Unscoped().Where("class_id = ?", ptRosterClass).Delete(&domain.GrowthDailyLog{})
	db.Unscoped().Where("class_id = ?", ptRosterClass).Delete(&domain.GrowthMedia{})
	db.Unscoped().Where("class_id = ?", ptRosterClass).Delete(&domain.Student{})
	db.Unscoped().Where("id = ?", ptRosterClass).Delete(&domain.Class{})
}

// ptSeedRosterClass 建合成班级 + 3 名在读（ids[0..2]）+ 1 名非在读（ids[3]，须被排除）。
func ptSeedRosterClass(t *testing.T, db *gorm.DB) []uint64 {
	t.Helper()
	require.NoError(t, db.Create(&domain.Class{
		BaseModel:      domain.BaseModel{ID: ptRosterClass},
		KindergartenID: 4, Name: "点名测试班", Status: 1,
	}).Error)
	d := time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC)
	ids := make([]uint64, 0, 4)
	for i := range 4 {
		id := uint64(ptTestBase + 501 + i)
		status := int8(1)
		if i == 3 {
			status = 2 // 毕业/离园：不在读，必须被排除
		}
		require.NoError(t, db.Create(&domain.Student{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: id}},
			KindergartenID:  4, ClassID: ptRosterClass, Name: "点到娃" + string(rune('A'+i)),
			Status: status, Birthday: d, EnrollDate: d,
		}).Error)
		ids = append(ids, id)
	}
	return ids
}

func TestPresentStudents_ConservativeAndExclusions(t *testing.T) {
	db := openPTTestDB(t)
	ctx := context.Background()
	ptRosterCleanup(t, db)
	ids := ptSeedRosterClass(t, db)
	defer ptRosterCleanup(t, db)

	tc := &TeacherPortalContext{UserID: 1, KindergartenID: 4, ClassID: ptRosterClass}
	s := ptService(db)
	today := teacherPortalDay(time.Now())

	// ① 无请假/考勤数据 → 保守视为全员到场（成本不对称：漏标孩子的代价远高于误标）
	data, err := s.PresentStudents(ctx, tc, "")
	require.NoError(t, err)
	assert.ElementsMatch(t, []uint64{ids[0], ids[1], ids[2]}, data["list"], "无数据应保守全员（且只含在读）")
	assert.Equal(t, 3, data["total"])

	// ② 已批准请假覆盖当日 → 排除；已批准的昨天假、未批准的今日假都不影响
	applicant := frSeedUploader(t, db)
	require.NoError(t, db.Create(&domain.LeaveRequest{
		KindergartenID: 4, ClassID: ptRosterClass, StudentID: ids[0], ApplicantID: applicant,
		Reason: "病假", StartDate: today, EndDate: today, Days: 1, Status: domain.LeaveStatusApproved,
	}).Error)
	require.NoError(t, db.Create(&domain.LeaveRequest{
		KindergartenID: 4, ClassID: ptRosterClass, StudentID: ids[1], ApplicantID: applicant,
		Reason: "昨天已休完", StartDate: today.AddDate(0, 0, -1), EndDate: today.AddDate(0, 0, -1),
		Days: 1, Status: domain.LeaveStatusApproved,
	}).Error)
	require.NoError(t, db.Create(&domain.LeaveRequest{
		KindergartenID: 4, ClassID: ptRosterClass, StudentID: ids[1], ApplicantID: applicant,
		Reason: "家长刚交还没批", StartDate: today, EndDate: today, Days: 1, Status: domain.LeaveStatusPending,
	}).Error)

	data, err = s.PresentStudents(ctx, tc, "")
	require.NoError(t, err)
	assert.ElementsMatch(t, []uint64{ids[1], ids[2]}, data["list"], "仅「当日已批准」请假排除；过期假与待审批假不影响")

	// ③ 考勤缺席 → 排除
	require.NoError(t, db.Create(&domain.Attendance{
		KindergartenID: 4, ClassID: ptRosterClass, StudentID: ids[2], Date: today, Status: domain.AttendanceAbsent,
	}).Error)
	data, err = s.PresentStudents(ctx, tc, "")
	require.NoError(t, err)
	assert.ElementsMatch(t, []uint64{ids[1]}, data["list"], "缺席的孩子排除")

	// ④ 考勤请假 → 排除（此时名单为空）
	require.NoError(t, db.Create(&domain.Attendance{
		KindergartenID: 4, ClassID: ptRosterClass, StudentID: ids[1], Date: today, Status: domain.AttendanceLeave,
	}).Error)
	data, err = s.PresentStudents(ctx, tc, "")
	require.NoError(t, err)
	assert.Empty(t, data["list"], "全班都明确不在时名单为空（不是保守回退）")
	assert.Equal(t, 0, data["total"])

	// ⑤ date 参数：查昨天 → 昨天的请假生效，今天的考勤不算数
	data, err = s.PresentStudents(ctx, tc, today.AddDate(0, 0, -1).Format("2006-01-02"))
	require.NoError(t, err)
	assert.ElementsMatch(t, []uint64{ids[0], ids[2]}, data["list"], "按指定日期判定：昨日假排除 ids[0]，今日考勤不参与昨日判定")
}

func TestApplyClassRoster_UnionPreservesRejectedAndManual(t *testing.T) {
	db := openPTTestDB(t)
	ctx := context.Background()
	ptRosterCleanup(t, db)
	ids := ptSeedRosterClass(t, db)
	defer ptRosterCleanup(t, db)

	creator := frSeedUploader(t, db)
	logID := uint64(ptTestBase + 510)
	mid := uint64(ptTestBase + 511)
	require.NoError(t, db.Create(&domain.GrowthDailyLog{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: logID}},
		KindergartenID:  4, ClassID: ptRosterClass, CreatorID: creator, Date: time.Now(),
		Content: "大合影全班测试", TargetType: 1,
	}).Error)
	require.NoError(t, db.Create(&domain.GrowthMedia{
		BaseModel: domain.BaseModel{ID: mid}, KindergartenID: 4, ClassID: ptRosterClass,
		LogID: &logID, FileURL: "http://mock.local/group.png", FileType: "image",
	}).Error)

	confirmed := domain.MediaStudentStatusConfirmed
	rejected := domain.MediaStudentStatusRejected
	suggested := domain.MediaStudentStatusSuggested
	conf := 88.0
	// 既有状态：ids[0] 人工确认；ids[1] 老师已拒绝；ids[2] AI 建议待处理；ids[3] 非在读但历史已点名（并集不得抹除）
	require.NoError(t, db.Create(&domain.GrowthMediaStudent{MediaID: mid, StudentID: ids[0], Status: &confirmed}).Error)
	require.NoError(t, db.Create(&domain.GrowthMediaStudent{MediaID: mid, StudentID: ids[1], Status: &rejected, IsAuto: true, Confidence: &conf}).Error)
	require.NoError(t, db.Create(&domain.GrowthMediaStudent{MediaID: mid, StudentID: ids[2], Status: &suggested, IsAuto: true, Confidence: &conf}).Error)
	require.NoError(t, db.Create(&domain.GrowthMediaStudent{MediaID: mid, StudentID: ids[3], Status: &confirmed}).Error)

	tc := &TeacherPortalContext{UserID: 1, KindergartenID: 4, ClassID: ptRosterClass}
	s := ptService(db)
	data, err := s.ApplyClassRoster(ctx, tc, mid)
	require.NoError(t, err)
	assert.Equal(t, 3, data["present"], "到场=3 名在读（非在读不参与）")
	// 回读的最终名单：确认态 = 到场 3 人 + 既有人工确认的 ids[3]；已拒绝的 ids[1] 不在其中
	assert.ElementsMatch(t, []uint64{ids[0], ids[2], ids[3]}, data["named"], "named 下发服务端最终名单（拒绝不复活）")

	var rows []domain.GrowthMediaStudent
	require.NoError(t, db.Where("media_id = ?", mid).Find(&rows).Error)
	assert.Len(t, rows, 4, "并集语义：不删除任何既有行")
	byStudent := map[uint64]domain.GrowthMediaStudent{}
	for _, r := range rows {
		byStudent[r.StudentID] = r
	}
	assert.Equal(t, domain.MediaStudentStatusConfirmed, *byStudent[ids[0]].Status, "既有确认保持")
	assert.Equal(t, domain.MediaStudentStatusRejected, *byStudent[ids[1]].Status, "已拒绝的行不复活——那是老师此前的明确表态")
	assert.Equal(t, domain.MediaStudentStatusConfirmed, *byStudent[ids[2]].Status, "待处理建议转为确认（全班即最终名单）")
	assert.False(t, byStudent[ids[2]].IsAuto, "一键全班属人工表态（is_auto=0）")
	assert.Equal(t, domain.MediaStudentStatusConfirmed, *byStudent[ids[3]].Status, "名单外学生不受影响（并集）")

	// 幂等：再点一次，行数与状态不变
	_, err = s.ApplyClassRoster(ctx, tc, mid)
	require.NoError(t, err)
	var n int64
	db.Model(&domain.GrowthMediaStudent{}).Where("media_id = ?", mid).Count(&n)
	assert.Equal(t, int64(4), n, "重复一键全班不产生重复行")

	// 软删行（重建点名时被移除）→ 硬复活为确认
	require.NoError(t, db.Where("media_id = ? AND student_id = ?", mid, ids[0]).Delete(&domain.GrowthMediaStudent{}).Error)
	_, err = s.ApplyClassRoster(ctx, tc, mid)
	require.NoError(t, err)
	var revived domain.GrowthMediaStudent
	require.NoError(t, db.Where("media_id = ? AND student_id = ?", mid, ids[0]).First(&revived).Error)
	assert.Equal(t, domain.MediaStudentStatusConfirmed, *revived.Status, "软删行复活")

	// 个体记录：不做全班
	indLogID := uint64(ptTestBase + 515)
	indMid := uint64(ptTestBase + 516)
	require.NoError(t, db.Create(&domain.GrowthDailyLog{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: indLogID}},
		KindergartenID:  4, ClassID: ptRosterClass, CreatorID: creator, Date: time.Now(),
		Content: "个体记录", TargetType: 2,
	}).Error)
	require.NoError(t, db.Create(&domain.GrowthMedia{
		BaseModel: domain.BaseModel{ID: indMid}, KindergartenID: 4, ClassID: ptRosterClass,
		LogID: &indLogID, FileURL: "http://mock.local/single.png", FileType: "image",
	}).Error)
	_, err = s.ApplyClassRoster(ctx, tc, indMid)
	assert.ErrorIs(t, err, ErrTeacherInvalidParams, "个体记录（target_type=2）不做全班")

	// 跨班：媒体不属于本班 → NOT_FOUND
	otherTC := &TeacherPortalContext{UserID: 1, KindergartenID: 4, ClassID: ptRosterClass + 1}
	_, err = s.ApplyClassRoster(ctx, otherTC, mid)
	assert.ErrorIs(t, err, ErrTeacherNotFound, "跨班越权必须 NOT_FOUND")
}

// TC-NEW-PTG-0018 的出参面：tagging-status 按照片下发 face_count（D11 大合影分型）。
func TestGrowthTaggingStatus_FaceCount(t *testing.T) {
	db := openPTTestDB(t)
	ctx := context.Background()
	ptRosterCleanup(t, db)
	ptSeedRosterClass(t, db) // 仅为满足 log.class_id 外键（本用例不需要学生）
	creator := frSeedUploader(t, db)
	logID := uint64(ptTestBase + 520)
	mid := uint64(ptTestBase + 521)
	require.NoError(t, db.Create(&domain.GrowthDailyLog{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: logID}},
		KindergartenID:  4, ClassID: ptRosterClass, CreatorID: creator, Date: time.Now(),
		Content: "分型测试", TargetType: 1,
	}).Error)
	require.NoError(t, db.Create(&domain.GrowthMedia{
		BaseModel: domain.BaseModel{ID: mid}, KindergartenID: 4, ClassID: ptRosterClass,
		LogID: &logID, FileURL: "http://mock.local/fc.png", FileType: "image",
	}).Error)
	defer ptRosterCleanup(t, db)

	require.NoError(t, db.Create(&domain.RecognitionJob{
		MediaID: mid, KindergartenID: 4, Status: domain.JobStatusDone, FaceCount: 8,
	}).Error)

	tc := &TeacherPortalContext{UserID: 1, KindergartenID: 4, ClassID: ptRosterClass}
	s := ptService(db)
	data, err := s.GrowthTaggingStatus(ctx, tc, logID)
	require.NoError(t, err)
	media := data["media"].([]map[string]any)
	require.Len(t, media, 1)
	assert.Equal(t, 8, media[0]["face_count"], "大合影分型：人脸数必须下发给老师端")
	assert.Equal(t, true, media[0]["is_group_photo"], "阈值（默认 6）由后端判定，前端不硬编码")
	assert.Equal(t, domain.JobStatusDone, media[0]["recognition_status"])
	assert.False(t, data["pending"].(bool))
}
