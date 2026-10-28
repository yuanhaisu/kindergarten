package service

// 成长相册（家长端）的按孩子隔离与分组回归。
//
// 修复前 ListGrowthMedia 返回 {list,total}，家长端读的是 groups —— 相册页永远走
// 「本月暂无照片」；且可见性用「家庭全部孩子」的班级集合，切到孩子 B 会看到 A 班照片。
// 这里用两个孩子两班级的真实库夹具把契约形状（groups[].date / groups[].media）与
// 「多孩不串」同时钉死。

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

// 独立 ID 段，避开 B05 的 9902xx 与开发库既有数据。
const (
	testAlbumKG        uint64 = 990301
	testAlbumClassA    uint64 = 990302
	testAlbumClassB    uint64 = 990303
	testAlbumStudentA  uint64 = 990304
	testAlbumStudentB  uint64 = 990305
	testAlbumParent    uint64 = 990306
	testAlbumTeacher   uint64 = 990307
	testAlbumLogA      uint64 = 990308
	testAlbumLogB      uint64 = 990309
	testAlbumLogAInd   uint64 = 990310
	testAlbumLogBInd   uint64 = 990311
	testAlbumMediaA1   uint64 = 990312
	testAlbumMediaA2   uint64 = 990313
	testAlbumMediaB1   uint64 = 990314
	testAlbumMediaAInd uint64 = 990315
	testAlbumMediaBInd uint64 = 990316
	testAlbumComment   uint64 = 990317
)

// cleanAlbumFixture 按 ID 硬删。软删行会占住唯一索引并污染重跑，一律 Unscoped。
func cleanAlbumFixture(db *gorm.DB) {
	logIDs := []uint64{testAlbumLogA, testAlbumLogB, testAlbumLogAInd, testAlbumLogBInd}
	db.Unscoped().Where("log_id IN ?", logIDs).Delete(&domain.GrowthComment{})
	db.Unscoped().Where("log_id IN ?", logIDs).Delete(&domain.GrowthMedia{})
	db.Unscoped().Where("log_id IN ?", logIDs).Delete(&domain.GrowthLogStudent{})
	db.Unscoped().Where("id IN ?", logIDs).Delete(&domain.GrowthDailyLog{})
	studentIDs := []uint64{testAlbumStudentA, testAlbumStudentB}
	db.Unscoped().Where("student_id IN ?", studentIDs).Delete(&domain.StudentParent{})
	db.Unscoped().Where("user_id = ? AND kindergarten_id = ?", testAlbumParent, testAlbumKG).Delete(&domain.UserRole{})
	db.Unscoped().Where("id IN ?", studentIDs).Delete(&domain.Student{})
	db.Unscoped().Where("id IN ?", []uint64{testAlbumClassA, testAlbumClassB}).Delete(&domain.Class{})
	db.Unscoped().Where("id IN ?", []uint64{testAlbumParent, testAlbumTeacher}).Delete(&domain.User{})
	db.Unscoped().Where("id = ?", testAlbumKG).Delete(&domain.Kindergarten{})
}

func setupAlbumFixture(t *testing.T) (*gorm.DB, *ParentPortalService) {
	t.Helper()
	db := openHomeFeedTestDB(t)
	cleanAlbumFixture(db)
	t.Cleanup(func() { cleanAlbumFixture(db) })

	require.NoError(t, db.Create(&domain.User{
		BaseModel: domain.BaseModel{ID: testAlbumTeacher}, Nickname: "相册回归老师", PasswordHash: "x",
	}).Error)
	require.NoError(t, db.Create(&domain.User{
		BaseModel: domain.BaseModel{ID: testAlbumParent}, Nickname: "相册回归家长", PasswordHash: "x",
	}).Error)
	require.NoError(t, db.Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: testAlbumKG}, Name: "测试园所-相册回归",
	}).Error)
	for _, cls := range []domain.Class{
		{BaseModel: domain.BaseModel{ID: testAlbumClassA}, KindergartenID: testAlbumKG, Name: "相册回归A班", Grade: "小班", Status: 1},
		{BaseModel: domain.BaseModel{ID: testAlbumClassB}, KindergartenID: testAlbumKG, Name: "相册回归B班", Grade: "大班", Status: 1},
	} {
		require.NoError(t, db.Create(&cls).Error)
	}

	// 开发库是严格模式：time.Time 零值会被 MySQL 拒绝，生日/入园日必须给真值。
	birthday := time.Date(2020, 3, 1, 0, 0, 0, 0, time.Local)
	for _, st := range []domain.Student{
		{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testAlbumStudentA}},
			KindergartenID: testAlbumKG, ClassID: testAlbumClassA, Name: "相册学生A", Status: 1,
			Birthday: birthday, EnrollDate: birthday},
		{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testAlbumStudentB}},
			KindergartenID: testAlbumKG, ClassID: testAlbumClassB, Name: "相册学生B", Status: 1,
			Birthday: birthday, EnrollDate: birthday},
	} {
		require.NoError(t, db.Create(&st).Error)
	}
	for _, sp := range []domain.StudentParent{
		{StudentID: testAlbumStudentA, UserID: testAlbumParent, Relation: domain.RelationMother, IsPrimary: true},
		{StudentID: testAlbumStudentB, UserID: testAlbumParent, Relation: domain.RelationFather},
	} {
		require.NoError(t, db.Create(&sp).Error)
	}
	kgID := testAlbumKG
	require.NoError(t, db.Create(&domain.UserRole{
		UserID: testAlbumParent, Role: "parent", KindergartenID: &kgID,
	}).Error)

	logs := []domain.GrowthDailyLog{
		{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testAlbumLogA}},
			KindergartenID: testAlbumKG, ClassID: testAlbumClassA, CreatorID: testAlbumTeacher,
			Date: time.Date(2026, 3, 19, 0, 0, 0, 0, time.Local), TargetType: 1, Content: "A班全班记录"},
		{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testAlbumLogB}},
			KindergartenID: testAlbumKG, ClassID: testAlbumClassB, CreatorID: testAlbumTeacher,
			Date: time.Date(2026, 3, 20, 0, 0, 0, 0, time.Local), TargetType: 1, Content: "B班全班记录"},
		{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testAlbumLogAInd}},
			KindergartenID: testAlbumKG, ClassID: testAlbumClassA, CreatorID: testAlbumTeacher,
			Date: time.Date(2026, 3, 5, 0, 0, 0, 0, time.Local), TargetType: 2, Content: "只发给A的个体记录"},
		{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testAlbumLogBInd}},
			KindergartenID: testAlbumKG, ClassID: testAlbumClassB, CreatorID: testAlbumTeacher,
			Date: time.Date(2026, 3, 4, 0, 0, 0, 0, time.Local), TargetType: 2, Content: "只发给B的个体记录"},
	}
	for _, l := range logs {
		require.NoError(t, db.Create(&l).Error)
	}
	for _, ls := range []domain.GrowthLogStudent{
		{LogID: testAlbumLogAInd, StudentID: testAlbumStudentA},
		{LogID: testAlbumLogBInd, StudentID: testAlbumStudentB},
	} {
		require.NoError(t, db.Create(&ls).Error)
	}

	// created_at 刻意落在 4 月：相册按「记录所属日期」归月，而不是按上传时刻。
	uploaded := time.Date(2026, 4, 2, 9, 0, 0, 0, time.Local)
	logID := func(v uint64) *uint64 { return &v }
	for _, md := range []domain.GrowthMedia{
		{BaseModel: domain.BaseModel{ID: testAlbumMediaA1, CreatedAt: uploaded},
			KindergartenID: testAlbumKG, ClassID: testAlbumClassA, LogID: logID(testAlbumLogA),
			UploaderID: testAlbumTeacher, FileURL: "https://cdn.test/a1.jpg", ThumbURL: "https://cdn.test/a1_thumb.jpg",
			FileType: "image"},
		{BaseModel: domain.BaseModel{ID: testAlbumMediaA2, CreatedAt: uploaded.Add(time.Minute)},
			KindergartenID: testAlbumKG, ClassID: testAlbumClassA, LogID: logID(testAlbumLogA),
			UploaderID: testAlbumTeacher, FileURL: "https://cdn.test/a2.jpg", ThumbURL: "https://cdn.test/a2_thumb.jpg",
			FileType: "image"},
		{BaseModel: domain.BaseModel{ID: testAlbumMediaB1, CreatedAt: uploaded},
			KindergartenID: testAlbumKG, ClassID: testAlbumClassB, LogID: logID(testAlbumLogB),
			UploaderID: testAlbumTeacher, FileURL: "https://cdn.test/b1.jpg", ThumbURL: "https://cdn.test/b1_thumb.jpg",
			FileType: "image"},
		{BaseModel: domain.BaseModel{ID: testAlbumMediaAInd, CreatedAt: uploaded},
			KindergartenID: testAlbumKG, ClassID: testAlbumClassA, LogID: logID(testAlbumLogAInd),
			UploaderID: testAlbumTeacher, FileURL: "https://cdn.test/aind.jpg", ThumbURL: "https://cdn.test/aind_thumb.jpg",
			FileType: "image"},
		{BaseModel: domain.BaseModel{ID: testAlbumMediaBInd, CreatedAt: uploaded},
			KindergartenID: testAlbumKG, ClassID: testAlbumClassB, LogID: logID(testAlbumLogBInd),
			UploaderID: testAlbumTeacher, FileURL: "https://cdn.test/bind.jpg", ThumbURL: "https://cdn.test/bind_thumb.jpg",
			FileType: "image"},
	} {
		require.NoError(t, db.Create(&md).Error)
	}

	svc := NewParentPortalService(db,
		repository.NewParentRepo(db), repository.NewUserRoleRepo(db),
		repository.NewStudentRepo(db), repository.NewClassRepo(db),
		repository.NewPickupRepo(db), repository.NewCalendarRepo(db),
		repository.NewNoticeRepo(db))
	return db, svc
}

// albumGroups 取出 groups 并断言契约形状：每组含 date(string) 与 media([])。
func albumGroups(t *testing.T, data map[string]any) []map[string]any {
	t.Helper()
	raw, ok := data["groups"]
	require.True(t, ok, "相册响应必须含 groups（契约 GET /parent/growth/album）")
	groups, ok := raw.([]map[string]any)
	require.True(t, ok, "groups 必须是对象数组")
	for _, g := range groups {
		_, ok := g["date"].(string)
		require.True(t, ok, "group.date 必须是 YYYY-MM-DD 字符串")
		_, ok = g["media"].([]map[string]any)
		require.True(t, ok, "group.media 必须是对象数组")
	}
	return groups
}

func albumMediaIDs(t *testing.T, groups []map[string]any) []uint64 {
	t.Helper()
	ids := make([]uint64, 0)
	for _, g := range groups {
		for _, m := range g["media"].([]map[string]any) {
			id, ok := m["id"].(uint64)
			require.True(t, ok, "media.id 必须是 uint64")
			ids = append(ids, id)
		}
	}
	return ids
}

// 核心回归：切到孩子 A 只看到 A 班全班记录与发给 A 的个体记录，B 的照片不得混入。
func TestParentGrowthAlbum_GroupsScopedToCurrentStudent(t *testing.T) {
	_, svc := setupAlbumFixture(t)
	ctx := context.Background()

	pcA, err := svc.ResolveContext(ctx, testAlbumParent, testAlbumKG, ptrU64(testAlbumStudentA))
	require.NoError(t, err)
	dataA, err := svc.ListGrowthMedia(ctx, pcA, 2026, 3)
	require.NoError(t, err)

	groupsA := albumGroups(t, dataA)
	idsA := albumMediaIDs(t, groupsA)
	assert.ElementsMatch(t, []uint64{testAlbumMediaA1, testAlbumMediaA2, testAlbumMediaAInd}, idsA,
		"A 应看到 A 班全班记录媒体与发给 A 的个体记录媒体")
	assert.NotContains(t, idsA, testAlbumMediaB1, "B 班全班记录媒体不得出现在 A 的相册")
	assert.NotContains(t, idsA, testAlbumMediaBInd, "发给 B 的个体记录媒体不得出现在 A 的相册")
	assert.Equal(t, 3, dataA["total"], "total 应为本次返回的媒体条数")

	// 分组键是记录日期（日志 date），不是上传时刻（媒体 created_at 落在 4 月）。
	require.Len(t, groupsA, 2, "A 的两条记录分属两天，应有两个日期分组")
	assert.Equal(t, "2026-03-19", groupsA[0]["date"], "分组按日期倒序，最新在前")
	assert.ElementsMatch(t, []uint64{testAlbumMediaA1, testAlbumMediaA2}, albumMediaIDs(t, groupsA[:1]))
	assert.Equal(t, "2026-03-05", groupsA[1]["date"])
	assert.ElementsMatch(t, []uint64{testAlbumMediaAInd}, albumMediaIDs(t, groupsA[1:]))

	// 媒体项字段符合契约。
	first := groupsA[0]["media"].([]map[string]any)[0]
	for _, key := range []string{"id", "file_url", "thumb_url", "file_type", "log_id"} {
		_, ok := first[key]
		assert.True(t, ok, "媒体项缺少契约字段 %s", key)
	}

	pcB, err := svc.ResolveContext(ctx, testAlbumParent, testAlbumKG, ptrU64(testAlbumStudentB))
	require.NoError(t, err)
	dataB, err := svc.ListGrowthMedia(ctx, pcB, 2026, 3)
	require.NoError(t, err)
	idsB := albumMediaIDs(t, albumGroups(t, dataB))
	assert.ElementsMatch(t, []uint64{testAlbumMediaB1, testAlbumMediaBInd}, idsB,
		"B 只应看到 B 班与发给 B 的个体记录媒体")
	assert.NotContains(t, idsB, testAlbumMediaA1)
	assert.NotContains(t, idsB, testAlbumMediaAInd)
}

// 月份过滤按记录日期生效；空月份返回空数组而不是 null。
func TestParentGrowthAlbum_MonthFilterAndEmptyGroups(t *testing.T) {
	_, svc := setupAlbumFixture(t)
	ctx := context.Background()

	pc, err := svc.ResolveContext(ctx, testAlbumParent, testAlbumKG, ptrU64(testAlbumStudentA))
	require.NoError(t, err)

	// 媒体是 4 月上传的，但记录属于 3 月，因此 4 月查询不应命中。
	april, err := svc.ListGrowthMedia(ctx, pc, 2026, 4)
	require.NoError(t, err)
	groups := albumGroups(t, april)
	assert.Empty(t, groups)
	assert.Equal(t, 0, april["total"])
	assert.Equal(t, 2026, april["year"])
	assert.Equal(t, 4, april["month"])

	march, err := svc.ListGrowthMedia(ctx, pc, 2026, 3)
	require.NoError(t, err)
	assert.NotEmpty(t, albumMediaIDs(t, albumGroups(t, march)))
}

// 夹具清理证据：按固定 ID 硬删后，各表不得残留（软删行会占唯一索引并污染重跑）。
func TestParentGrowthFixture_CleanupByIDLeavesNoResidue(t *testing.T) {
	db, _ := setupAlbumFixture(t)
	require.NoError(t, db.Create(&domain.GrowthComment{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testAlbumComment}},
		LogID:           testAlbumLogA, UserID: testAlbumParent, Content: "清理证据用评论",
	}).Error)

	count := func(model any, q string, args ...any) int64 {
		var n int64
		require.NoError(t, db.Unscoped().Model(model).Where(q, args...).Count(&n).Error)
		return n
	}
	mediaIDs := []uint64{testAlbumMediaA1, testAlbumMediaA2, testAlbumMediaB1, testAlbumMediaAInd, testAlbumMediaBInd}
	logIDs := []uint64{testAlbumLogA, testAlbumLogB, testAlbumLogAInd, testAlbumLogBInd}
	studentIDs := []uint64{testAlbumStudentA, testAlbumStudentB}

	before := map[string]int64{
		"growth_media":        count(&domain.GrowthMedia{}, "id IN ?", mediaIDs),
		"growth_daily_logs":   count(&domain.GrowthDailyLog{}, "id IN ?", logIDs),
		"growth_log_students": count(&domain.GrowthLogStudent{}, "log_id IN ?", logIDs),
		"growth_comments":     count(&domain.GrowthComment{}, "log_id IN ?", logIDs),
		"student_parents":     count(&domain.StudentParent{}, "student_id IN ?", studentIDs),
		"students":            count(&domain.Student{}, "id IN ?", studentIDs),
		"users":               count(&domain.User{}, "id IN ?", []uint64{testAlbumParent, testAlbumTeacher}),
		"kindergartens":       count(&domain.Kindergarten{}, "id = ?", testAlbumKG),
	}
	cleanAlbumFixture(db)
	after := map[string]int64{
		"growth_media":        count(&domain.GrowthMedia{}, "id IN ?", mediaIDs),
		"growth_daily_logs":   count(&domain.GrowthDailyLog{}, "id IN ?", logIDs),
		"growth_log_students": count(&domain.GrowthLogStudent{}, "log_id IN ?", logIDs),
		"growth_comments":     count(&domain.GrowthComment{}, "log_id IN ?", logIDs),
		"student_parents":     count(&domain.StudentParent{}, "student_id IN ?", studentIDs),
		"students":            count(&domain.Student{}, "id IN ?", studentIDs),
		"users":               count(&domain.User{}, "id IN ?", []uint64{testAlbumParent, testAlbumTeacher}),
		"kindergartens":       count(&domain.Kindergarten{}, "id = ?", testAlbumKG),
	}
	t.Logf("cleanup before=%v after=%v", before, after)
	for table, n := range after {
		assert.Zero(t, n, "%s 清理后不应残留", table)
	}
}
