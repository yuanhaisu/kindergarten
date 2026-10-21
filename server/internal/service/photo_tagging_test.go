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

// 照片点名（PTG）相关测试。DB 用例复用 openRQTestDB（连本机 MySQL，无配置则 Skip）。
// 数据段 991000000+，与识别队列测试段（990000000+）错开。

const ptTestBase = 991000000

func openPTTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := openRQTestDB(t)
	require.NoError(t, db.AutoMigrate(&domain.GrowthMedia{}, &domain.GrowthDailyLog{}, &domain.Student{}))
	return db
}

func ptService(db *gorm.DB) *TeacherPortalService {
	return NewTeacherPortalService(db, nil, nil, nil, nil, nil, config.FaceConfig{SuggestHigh: 80, SuggestLow: 70})
}

func ptCleanup(t *testing.T, db *gorm.DB, mediaID uint64) {
	t.Helper()
	// 硬删：软删行仍占唯一索引，会令后续 Create/重建撞唯一键（测试隔离）
	db.Unscoped().Where("media_id = ?", mediaID).Delete(&domain.GrowthMediaStudent{})
	db.Unscoped().Where("id = ?", mediaID).Delete(&domain.GrowthMedia{})
}

// ── 5.1 点名写入：白名单（纯判定）────────────────────────────────────

func TestValidateMediaStudentIDs_Whitelist(t *testing.T) {
	scope := map[uint64]bool{1: true, 2: true}

	// 范围内、重复、0 混排：合法（重复不叠加是幂等语义）
	err := validateMediaStudentIDs([]MediaInput{{StudentIDs: []uint64{1, 1, 0, 2}}}, scope)
	assert.NoError(t, err, "范围内学生（含重复/0）应通过")

	// 越界：拒绝（TC-NEW-PTG-0014）
	err = validateMediaStudentIDs([]MediaInput{{StudentIDs: []uint64{1, 99}}}, scope)
	assert.ErrorIs(t, err, ErrTeacherInvalidParams, "范围外学生必须被拒绝")

	// nil / 空：合法（未提交点名）
	assert.NoError(t, validateMediaStudentIDs([]MediaInput{{StudentIDs: nil}, {StudentIDs: []uint64{}}}, scope))
}

func TestRebuildMediaStudents_RebuildSemantics(t *testing.T) {
	db := openPTTestDB(t)
	ctx := context.Background()
	mid := uint64(ptTestBase + 1)
	ptCleanup(t, db, mid)

	// 初次点名 A/B（含一条 AI 建议 C，验证重建会清掉未提交的关联）
	require.NoError(t, rebuildMediaStudents(ctx, db, mid, []uint64{ptTestBase + 11, ptTestBase + 12}))
	st0 := domain.MediaStudentStatusSuggested
	require.NoError(t, db.Create(&domain.GrowthMediaStudent{MediaID: mid, StudentID: uint64(ptTestBase + 13), IsAuto: true, Status: &st0}).Error)

	// 重建为仅 A + 新 D
	require.NoError(t, rebuildMediaStudents(ctx, db, mid, []uint64{ptTestBase + 11, ptTestBase + 14}))

	var rows []domain.GrowthMediaStudent
	require.NoError(t, db.Where("media_id = ?", mid).Find(&rows).Error)
	byStudent := map[uint64]domain.GrowthMediaStudent{}
	for _, r := range rows {
		byStudent[r.StudentID] = r
	}
	assert.Len(t, rows, 2, "重建后只保留提交名单")
	assert.Contains(t, byStudent, uint64(ptTestBase+11), "提交项保留")
	assert.NotContains(t, byStudent, uint64(ptTestBase+12), "未提交的既有点名被移除")
	assert.NotContains(t, byStudent, uint64(ptTestBase+13), "未提交的 AI 建议被移除（编辑提交即最终名单）")
	for _, r := range byStudent {
		require.NotNil(t, r.Status)
		assert.Equal(t, domain.MediaStudentStatusConfirmed, *r.Status, "人工点名直接落确认态")
		assert.False(t, r.IsAuto)
	}

	// 重复应用同一批点名：不叠加（TC-NEW-PTG-0004）
	require.NoError(t, rebuildMediaStudents(ctx, db, mid, []uint64{ptTestBase + 11, ptTestBase + 14}))
	var n int64
	db.Model(&domain.GrowthMediaStudent{}).Where("media_id = ?", mid).Count(&n)
	assert.Equal(t, int64(2), n, "重复点名不产生重复关联")

	// 软删后再次点名同一学生：复活而非撞唯一键（revive 路径）
	require.NoError(t, rebuildMediaStudents(ctx, db, mid, []uint64{ptTestBase + 12}))
	var revived domain.GrowthMediaStudent
	require.NoError(t, db.Where("media_id = ? AND student_id = ?", mid, uint64(ptTestBase+12)).First(&revived).Error)
	assert.Equal(t, domain.MediaStudentStatusConfirmed, *revived.Status)

	// 清空点名：全部移除（照片变通用照片，TC-NEW-PTG-0002 的数据面）
	require.NoError(t, rebuildMediaStudents(ctx, db, mid, nil))
	db.Model(&domain.GrowthMediaStudent{}).Where("media_id = ?", mid).Count(&n)
	assert.Zero(t, n)

	ptCleanup(t, db, mid)
}

// ── 5.2 出参：家长只见确认项；老师可见建议与分级 ───────────────────

func TestConfirmedMediaStudents_OnlyConfirmed(t *testing.T) {
	db := openPTTestDB(t)
	ctx := context.Background()
	mid := uint64(ptTestBase + 20)
	ptCleanup(t, db, mid)

	confirmed := domain.MediaStudentStatusConfirmed
	rejected := domain.MediaStudentStatusRejected
	suggested := domain.MediaStudentStatusSuggested
	require.NoError(t, db.Create(&domain.GrowthMediaStudent{MediaID: mid, StudentID: ptTestBase + 21, Status: &confirmed}).Error)
	require.NoError(t, db.Create(&domain.GrowthMediaStudent{MediaID: mid, StudentID: ptTestBase + 22, Status: nil}).Error) // 存量行：NULL=确认
	require.NoError(t, db.Create(&domain.GrowthMediaStudent{MediaID: mid, StudentID: ptTestBase + 23, Status: &suggested, IsAuto: true}).Error)
	require.NoError(t, db.Create(&domain.GrowthMediaStudent{MediaID: mid, StudentID: ptTestBase + 24, Status: &rejected}).Error)

	p := &ParentPortalService{db: db}
	got, err := p.confirmedMediaStudents(ctx, []uint64{mid})
	require.NoError(t, err)
	assert.ElementsMatch(t, []uint64{ptTestBase + 21, ptTestBase + 22}, got[mid],
		"家长出参只含确认项（1 与存量 NULL）；建议（0）与拒绝（2）不得出现")

	ptCleanup(t, db, mid)
}

func TestLoadMediaTagView_TeacherShape(t *testing.T) {
	db := openPTTestDB(t)
	ctx := context.Background()
	mid := uint64(ptTestBase + 30)
	sid := uint64(ptTestBase + 31)
	ptCleanup(t, db, mid)
	validDate := time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC)
	for i := uint64(0); i < 3; i++ {
		db.Unscoped().Where("id = ?", sid+i).Delete(&domain.Student{})
		require.NoError(t, db.Create(&domain.Student{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: sid + i}},
			KindergartenID: 4, ClassID: 6, Name: "测试娃", Status: 1,
			Birthday: validDate, EnrollDate: validDate}).Error)
	}

	confirmed := domain.MediaStudentStatusConfirmed
	suggested := domain.MediaStudentStatusSuggested
	confHigh, confLow := 91.5, 72.0
	require.NoError(t, db.Create(&domain.GrowthMediaStudent{MediaID: mid, StudentID: sid, Status: &confirmed}).Error)
	require.NoError(t, db.Create(&domain.GrowthMediaStudent{MediaID: mid, StudentID: sid + 1, IsAuto: true, Status: &suggested, Confidence: &confHigh}).Error)
	require.NoError(t, db.Create(&domain.GrowthMediaStudent{MediaID: mid, StudentID: sid + 2, IsAuto: true, Status: &suggested, Confidence: &confLow}).Error)

	s := ptService(db)
	view, err := s.loadMediaTagView(ctx, []uint64{mid})
	require.NoError(t, err)
	assert.Equal(t, []uint64{sid}, view.confirmed[mid])

	require.Len(t, view.suggestions[mid], 2)
	byTier := map[string]map[string]any{}
	for _, sug := range view.suggestions[mid] {
		byTier[sug["tier"].(string)] = sug
	}
	assert.Equal(t, "high", byTier["high"]["tier"], "91.5 ≥ suggest_high(80) 应为高置信（预勾）")
	assert.Equal(t, "low", byTier["low"]["tier"], "72 < 80 应为低置信（提示不预勾）")
	// 建议项带名字，老师端无需二次查询
	for _, sug := range view.suggestions[mid] {
		assert.NotEmpty(t, sug["name"])
	}

	db.Unscoped().Where("media_id = ?", mid).Delete(&domain.GrowthMediaStudent{})
	for i := uint64(0); i < 3; i++ {
		db.Unscoped().Where("id = ?", sid+i).Delete(&domain.Student{})
	}
}

// ── 8.1 一键确认高置信（D10 省力路径）──────────────────────────────

func TestConfirmHighConfidenceSuggestions_OnlyHigh(t *testing.T) {
	db := openPTTestDB(t)
	ctx := context.Background()
	logID := uint64(ptTestBase + 60)
	mid1, mid2 := uint64(ptTestBase+61), uint64(ptTestBase+62)
	ptCleanup(t, db, mid1)
	ptCleanup(t, db, mid2)
	db.Unscoped().Where("id = ?", logID).Delete(&domain.GrowthDailyLog{})
	creator := frSeedUploader(t, db) // 日志 creator 有外键约束，需真实用户
	require.NoError(t, db.Create(&domain.GrowthDailyLog{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: logID}},
		KindergartenID:  4, ClassID: 6, CreatorID: creator, Date: time.Now(), Content: "批量确认测试", TargetType: 1,
	}).Error)
	for _, mid := range []uint64{mid1, mid2} {
		require.NoError(t, db.Create(&domain.GrowthMedia{
			BaseModel: domain.BaseModel{ID: mid}, KindergartenID: 4, ClassID: 6, LogID: &logID,
			FileURL: "http://mock.local/bulk.png", FileType: "image",
		}).Error)
	}

	suggested := domain.MediaStudentStatusSuggested
	confirmed := domain.MediaStudentStatusConfirmed
	high, low := 91.0, 72.0
	require.NoError(t, db.Create(&domain.GrowthMediaStudent{MediaID: mid1, StudentID: 1, IsAuto: true, Status: &suggested, Confidence: &high}).Error)
	require.NoError(t, db.Create(&domain.GrowthMediaStudent{MediaID: mid1, StudentID: 2, IsAuto: true, Status: &suggested, Confidence: &low}).Error)
	require.NoError(t, db.Create(&domain.GrowthMediaStudent{MediaID: mid2, StudentID: 3, IsAuto: true, Status: &suggested, Confidence: &high}).Error)
	require.NoError(t, db.Create(&domain.GrowthMediaStudent{MediaID: mid2, StudentID: 4, Status: &confirmed, Confidence: &high}).Error)

	tc := &TeacherPortalContext{UserID: 1, KindergartenID: 4, ClassID: 6}
	s := ptService(db) // SuggestHigh=80
	data, err := s.ConfirmHighConfidenceSuggestions(ctx, tc, logID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), data["confirmed"], "两张照片各一条高置信 → 确认 2 条")

	// 注意：不能复用同一个 struct 变量连续 First——GORM 会把上次读出的主键带入下一次查询条件
	var rowHigh, rowLow, rowExisting domain.GrowthMediaStudent
	require.NoError(t, db.Where("media_id = ? AND student_id = ?", mid1, 1).First(&rowHigh).Error)
	assert.Equal(t, domain.MediaStudentStatusConfirmed, *rowHigh.Status, "高置信转生效")
	require.NoError(t, db.Where("media_id = ? AND student_id = ?", mid1, 2).First(&rowLow).Error)
	assert.Equal(t, domain.MediaStudentStatusSuggested, *rowLow.Status, "低置信保持待确认")
	require.NoError(t, db.Where("media_id = ? AND student_id = ?", mid2, 4).First(&rowExisting).Error)
	assert.Equal(t, domain.MediaStudentStatusConfirmed, *rowExisting.Status, "已确认行不变")

	// 归属：跨班不可批量确认
	_, err = s.ConfirmHighConfidenceSuggestions(ctx, &TeacherPortalContext{KindergartenID: 4, ClassID: 999}, logID)
	assert.ErrorIs(t, err, ErrTeacherNotFound)

	ptCleanup(t, db, mid1)
	ptCleanup(t, db, mid2)
	db.Unscoped().Where("id = ?", logID).Delete(&domain.GrowthDailyLog{})
}

// TC-NEW-PTG-0016/0017：列表出参携带待确认建议数（无建议为 0）
func TestListGrowthLogs_PendingSuggestionsCount(t *testing.T) {
	db := openPTTestDB(t)
	ctx := context.Background()
	logID := uint64(ptTestBase + 70)
	mid := uint64(ptTestBase + 71)
	ptCleanup(t, db, mid)
	db.Unscoped().Where("id = ?", logID).Delete(&domain.GrowthDailyLog{})
	creator := frSeedUploader(t, db)
	require.NoError(t, db.Create(&domain.GrowthDailyLog{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: logID}},
		KindergartenID:  4, ClassID: 6, CreatorID: creator, Date: time.Now(), Content: "待确认计数测试", TargetType: 1,
	}).Error)
	require.NoError(t, db.Create(&domain.GrowthMedia{
		BaseModel: domain.BaseModel{ID: mid}, KindergartenID: 4, ClassID: 6, LogID: &logID,
		FileURL: "http://mock.local/cnt.png", FileType: "image",
	}).Error)
	suggested := domain.MediaStudentStatusSuggested
	conf, low := 90.0, 71.0
	require.NoError(t, db.Create(&domain.GrowthMediaStudent{MediaID: mid, StudentID: 1, IsAuto: true, Status: &suggested, Confidence: &conf}).Error)
	require.NoError(t, db.Create(&domain.GrowthMediaStudent{MediaID: mid, StudentID: 2, IsAuto: true, Status: &suggested, Confidence: &low}).Error)

	s := ptService(db)
	tc := &TeacherPortalContext{UserID: 1, KindergartenID: 4, ClassID: 6}
	res, err := s.ListGrowthLogs(ctx, tc, 1, 20)
	require.NoError(t, err)
	list, _ := res["list"].([]map[string]any)
	require.NotEmpty(t, list)
	found := false
	for _, item := range list {
		if item["id"] == logID {
			assert.Equal(t, int64(2), item["pending_suggestions_count"], "两条待确认建议应计数为 2")
			found = true
		}
	}
	require.True(t, found, "本记录应出现在列表首页")

	// 处理掉一条后计数降为 1
	require.NoError(t, s.db.WithContext(ctx).Model(&domain.GrowthMediaStudent{}).
		Where("media_id = ? AND student_id = ?", mid, 1).Update("status", domain.MediaStudentStatusConfirmed).Error)
	res, err = s.ListGrowthLogs(ctx, tc, 1, 20)
	require.NoError(t, err)
	list, _ = res["list"].([]map[string]any)
	for _, item := range list {
		if item["id"] == logID {
			assert.Equal(t, int64(1), item["pending_suggestions_count"])
		}
	}

	ptCleanup(t, db, mid)
	db.Unscoped().Where("id = ?", logID).Delete(&domain.GrowthDailyLog{})
}

// ── 5.3 建议确认端点：状态机迁移 ─────────────────────────────────────

func TestReviewMediaSuggestion_Transitions(t *testing.T) {
	db := openPTTestDB(t)
	ctx := context.Background()
	mid := uint64(ptTestBase + 40)
	ptCleanup(t, db, mid)
	logID := mid
	require.NoError(t, db.Create(&domain.GrowthMedia{
		BaseModel: domain.BaseModel{ID: mid}, KindergartenID: 4, ClassID: 6,
		LogID: &logID, FileURL: "http://mock.local/ptg.png", FileType: "image",
	}).Error)

	tc := &TeacherPortalContext{UserID: 1, KindergartenID: 4, ClassID: 6}
	s := ptService(db)

	suggested := domain.MediaStudentStatusSuggested

	// 0→1 确认
	sid1 := uint64(ptTestBase + 41)
	require.NoError(t, db.Create(&domain.GrowthMediaStudent{MediaID: mid, StudentID: sid1, IsAuto: true, Status: &suggested}).Error)
	data, err := s.ReviewMediaSuggestion(ctx, tc, mid, sid1, true)
	require.NoError(t, err)
	assert.Equal(t, domain.MediaStudentStatusConfirmed, data["status"])
	var r1 domain.GrowthMediaStudent
	require.NoError(t, db.Where("media_id = ? AND student_id = ?", mid, sid1).First(&r1).Error)
	assert.Equal(t, domain.MediaStudentStatusConfirmed, *r1.Status, "确认后家长端立即可见（TC-NEW-PTG-0008）")

	// 再拒绝已确认项：拒绝（唯一合法迁移是 0→1/0→2）
	_, err = s.ReviewMediaSuggestion(ctx, tc, mid, sid1, false)
	assert.ErrorIs(t, err, ErrTeacherInvalidParams, "已确认的行不得经本端点转为拒绝")

	// 重复确认：幂等
	_, err = s.ReviewMediaSuggestion(ctx, tc, mid, sid1, true)
	assert.NoError(t, err, "重复确认应幂等成功")

	// 0→2 拒绝
	sid2 := uint64(ptTestBase + 42)
	require.NoError(t, db.Create(&domain.GrowthMediaStudent{MediaID: mid, StudentID: sid2, IsAuto: true, Status: &suggested}).Error)
	_, err = s.ReviewMediaSuggestion(ctx, tc, mid, sid2, false)
	require.NoError(t, err)
	var r2 domain.GrowthMediaStudent
	require.NoError(t, db.Where("media_id = ? AND student_id = ?", mid, sid2).First(&r2).Error)
	assert.Equal(t, domain.MediaStudentStatusRejected, *r2.Status, "拒绝后不再提示（TC-NEW-PTG-0009）")

	// NULL（存量确认行）：确认幂等通过，拒绝非法
	sid3 := uint64(ptTestBase + 43)
	require.NoError(t, db.Create(&domain.GrowthMediaStudent{MediaID: mid, StudentID: sid3, Status: nil}).Error)
	_, err = s.ReviewMediaSuggestion(ctx, tc, mid, sid3, true)
	assert.NoError(t, err)
	_, err = s.ReviewMediaSuggestion(ctx, tc, mid, sid3, false)
	assert.ErrorIs(t, err, ErrTeacherInvalidParams)

	// 媒体不属于本班：NOT_FOUND（跨班越权）
	otherTC := &TeacherPortalContext{UserID: 1, KindergartenID: 4, ClassID: 999}
	_, err = s.ReviewMediaSuggestion(ctx, otherTC, mid, sid1, true)
	assert.ErrorIs(t, err, ErrTeacherNotFound)

	// 不存在的关联：NOT_FOUND
	_, err = s.ReviewMediaSuggestion(ctx, tc, mid, uint64(ptTestBase+99), true)
	assert.ErrorIs(t, err, ErrTeacherNotFound)

	ptCleanup(t, db, mid)
}
