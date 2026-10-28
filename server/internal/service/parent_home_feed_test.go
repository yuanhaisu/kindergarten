package service

// 首页动态流的多孩隔离回归（TC-PRN-HOME-002 学生切换 / TC-PRN-HOME-004 成长时光流）。
//
// 修复前 ListHomeFeeds 用「家庭全部孩子」的班级集合取成长记录，切到孩子 B 仍能看到
// A 班的记录（多孩串数据）。这里用两孩子两班级的真实库夹具把「不串数据」钉死，
// 同时钉住两条不变式：通知仍是家庭级（不按孩子过滤）、students[] 必须带 relation。

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// 独立 ID 段避免与开发库既有数据/其他测试夹具相撞。
const (
	testHomeKG       uint64 = 990241
	testHomeClassA   uint64 = 990242
	testHomeClassB   uint64 = 990243
	testHomeStudentA uint64 = 990244
	testHomeStudentB uint64 = 990245
	testHomeParent   uint64 = 990246
	testHomeTeacher  uint64 = 990247
	testHomeLogA     uint64 = 990248
	testHomeLogB     uint64 = 990249
	testHomeLogAInd  uint64 = 990250
	testHomeStudentC uint64 = 990251
	testHomeNotice   uint64 = 990252
)

func openHomeFeedTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	cfgPath := filepath.Join("..", "..", "configs", "app.ini")
	if _, err := os.Stat(cfgPath); err != nil {
		t.Skipf("跳过：找不到 %s（本地无 DB 配置）", cfgPath)
	}
	cfg, err := config.Load(cfgPath)
	require.NoError(t, err)
	if cfg.MySQL.DSN == "" {
		t.Skip("跳过：MySQL DSN 为空")
	}
	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	return db
}

// cleanHomeFeedFixture 按 ID 硬删。SoftDeleteModel 的软删行会留在表里污染重跑，
// 关联表（growth_log_students / notice_targets / student_parents）没有显式 ID，按外键列删。
func cleanHomeFeedFixture(db *gorm.DB) {
	db.Unscoped().Where("notice_id = ?", testHomeNotice).Delete(&domain.NoticeTarget{})
	db.Unscoped().Where("id = ?", testHomeNotice).Delete(&domain.Notice{})
	db.Unscoped().Where("log_id IN ?", []uint64{testHomeLogA, testHomeLogB, testHomeLogAInd}).Delete(&domain.GrowthLogStudent{})
	db.Unscoped().Where("id IN ?", []uint64{testHomeLogA, testHomeLogB, testHomeLogAInd}).Delete(&domain.GrowthDailyLog{})
	db.Unscoped().Where("student_id IN ?", []uint64{testHomeStudentA, testHomeStudentB, testHomeStudentC}).Delete(&domain.StudentParent{})
	db.Unscoped().Where("user_id = ? AND kindergarten_id = ?", testHomeParent, testHomeKG).Delete(&domain.UserRole{})
	db.Unscoped().Where("id IN ?", []uint64{testHomeStudentA, testHomeStudentB, testHomeStudentC}).Delete(&domain.Student{})
	db.Unscoped().Where("id IN ?", []uint64{testHomeClassA, testHomeClassB}).Delete(&domain.Class{})
	db.Unscoped().Where("id IN ?", []uint64{testHomeParent, testHomeTeacher}).Delete(&domain.User{})
	db.Unscoped().Where("id = ?", testHomeKG).Delete(&domain.Kindergarten{})
}

func setupHomeFeedFixture(t *testing.T) (*gorm.DB, *ParentPortalService) {
	t.Helper()
	db := openHomeFeedTestDB(t)
	cleanHomeFeedFixture(db)
	t.Cleanup(func() { cleanHomeFeedFixture(db) })

	require.NoError(t, db.Create(&domain.User{
		BaseModel: domain.BaseModel{ID: testHomeTeacher}, Nickname: "首页feed回归老师", PasswordHash: "x",
	}).Error)
	require.NoError(t, db.Create(&domain.User{
		BaseModel: domain.BaseModel{ID: testHomeParent}, Nickname: "首页feed回归家长", PasswordHash: "x",
	}).Error)
	require.NoError(t, db.Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: testHomeKG}, Name: "测试园所-首页feed回归",
	}).Error)
	require.NoError(t, db.Create(&domain.Class{
		BaseModel:      domain.BaseModel{ID: testHomeClassA},
		KindergartenID: testHomeKG, Name: "首页回归A班", Grade: "小班", Status: 1,
	}).Error)
	require.NoError(t, db.Create(&domain.Class{
		BaseModel:      domain.BaseModel{ID: testHomeClassB},
		KindergartenID: testHomeKG, Name: "首页回归B班", Grade: "大班", Status: 1,
	}).Error)

	// 开发库是严格模式：time.Time 零值会被 MySQL 拒绝（'0000-00-00'），生日/入园日必须给真值。
	birthday := time.Date(2020, 3, 1, 0, 0, 0, 0, time.Local)
	for _, st := range []domain.Student{
		{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testHomeStudentA}},
			KindergartenID: testHomeKG, ClassID: testHomeClassA, Name: "学生A", Status: 1,
			Birthday: birthday, EnrollDate: birthday},
		{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testHomeStudentB}},
			KindergartenID: testHomeKG, ClassID: testHomeClassB, Name: "学生B", Status: 1,
			Birthday: birthday, EnrollDate: birthday},
		// C 有家长关联但 relation 为空：首页必须回退「家长」而不是报错/输出空称呼。
		{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testHomeStudentC}},
			KindergartenID: testHomeKG, ClassID: testHomeClassA, Name: "学生C", Status: 1,
			Birthday: birthday, EnrollDate: birthday},
	} {
		require.NoError(t, db.Create(&st).Error)
	}

	for _, sp := range []domain.StudentParent{
		{StudentID: testHomeStudentA, UserID: testHomeParent, Relation: domain.RelationMother, IsPrimary: true},
		{StudentID: testHomeStudentB, UserID: testHomeParent, Relation: domain.RelationFather},
		{StudentID: testHomeStudentC, UserID: testHomeParent, Relation: ""},
	} {
		require.NoError(t, db.Create(&sp).Error)
	}

	kgID := testHomeKG
	require.NoError(t, db.Create(&domain.UserRole{
		UserID: testHomeParent, Role: "parent", KindergartenID: &kgID,
	}).Error)

	day := time.Date(2026, 10, 27, 0, 0, 0, 0, time.Local)
	for _, l := range []domain.GrowthDailyLog{{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testHomeLogA}},
		KindergartenID: testHomeKG, ClassID: testHomeClassA, CreatorID: testHomeTeacher,
		Date: day, TargetType: 1, Content: "A班全班记录"},
		{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testHomeLogB}},
			KindergartenID: testHomeKG, ClassID: testHomeClassB, CreatorID: testHomeTeacher,
			Date: day, TargetType: 1, Content: "B班全班记录"},
		{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testHomeLogAInd}},
			KindergartenID: testHomeKG, ClassID: testHomeClassA, CreatorID: testHomeTeacher,
			Date: day, TargetType: 2, Content: "只发给A的个体记录"},
	} {
		require.NoError(t, db.Create(&l).Error)
	}
	require.NoError(t, db.Create(&domain.GrowthLogStudent{
		LogID: testHomeLogAInd, StudentID: testHomeStudentA,
	}).Error)

	published := time.Now().Add(-time.Hour)
	require.NoError(t, db.Create(&domain.Notice{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testHomeNotice}},
		KindergartenID:  testHomeKG, ClassID: testHomeClassA, CreatorID: testHomeTeacher,
		Category: domain.NoticeCategoryActivity, Title: "首页回归通知", Content: "通知正文",
		Status: domain.NoticeStatusPublished, PublishedAt: &published,
	}).Error)
	require.NoError(t, db.Create(&domain.NoticeTarget{
		NoticeID: testHomeNotice, ClassID: testHomeClassA, KindergartenID: testHomeKG,
	}).Error)

	svc := NewParentPortalService(db,
		repository.NewParentRepo(db), repository.NewUserRoleRepo(db),
		repository.NewStudentRepo(db), repository.NewClassRepo(db),
		repository.NewPickupRepo(db), repository.NewCalendarRepo(db),
		repository.NewNoticeRepo(db))
	return db, svc
}

// feedIDsByType 从首页动态流里取某类型的条目 id（list 的元素是 map，键按 type 区分）。
func feedIDsByType(t *testing.T, feed map[string]any, typ string) []uint64 {
	t.Helper()
	list, ok := feed["list"].([]map[string]any)
	require.True(t, ok, "feed.list 类型应为 []map[string]any")
	ids := make([]uint64, 0, len(list))
	for _, item := range list {
		if item["type"] != typ {
			continue
		}
		id, ok := item["id"].(uint64)
		require.True(t, ok, "动态条目 id 应为 uint64")
		ids = append(ids, id)
	}
	return ids
}

// 核心回归：A/B 两个孩子的成长记录互不串数据。
func TestParentHomeFeeds_GrowthScopedToCurrentStudent(t *testing.T) {
	// TC-PRN-HOME-002 / TC-PRN-HOME-004
	_, svc := setupHomeFeedFixture(t)
	ctx := context.Background()

	pcA, err := svc.ResolveContext(ctx, testHomeParent, testHomeKG, ptrU64(testHomeStudentA))
	require.NoError(t, err)
	feedA, err := svc.ListHomeFeeds(ctx, pcA, 1, 20)
	require.NoError(t, err)
	growthA := feedIDsByType(t, feedA, "growth")
	assert.Contains(t, growthA, testHomeLogA, "当前孩子所在班级的全班记录必须可见")
	assert.Contains(t, growthA, testHomeLogAInd, "明确发给当前孩子的个体记录必须可见")
	assert.NotContains(t, growthA, testHomeLogB, "切到 A 时不得出现 B 班的成长记录")

	pcB, err := svc.ResolveContext(ctx, testHomeParent, testHomeKG, ptrU64(testHomeStudentB))
	require.NoError(t, err)
	feedB, err := svc.ListHomeFeeds(ctx, pcB, 1, 20)
	require.NoError(t, err)
	growthB := feedIDsByType(t, feedB, "growth")
	assert.Contains(t, growthB, testHomeLogB)
	assert.NotContains(t, growthB, testHomeLogA, "切到 B 时不得出现 A 班的成长记录")
	assert.NotContains(t, growthB, testHomeLogAInd, "发给另一个孩子的个体记录不得出现在首页")
}

// 通知保持家庭级：不随当前孩子收缩，避免切换孩子后通知区闪空。
func TestParentHomeFeeds_NoticesStayFamilyWide(t *testing.T) {
	// TC-PRN-HOME-003
	_, svc := setupHomeFeedFixture(t)
	ctx := context.Background()

	for _, studentID := range []uint64{testHomeStudentA, testHomeStudentB} {
		pc, err := svc.ResolveContext(ctx, testHomeParent, testHomeKG, ptrU64(studentID))
		require.NoError(t, err)
		feed, err := svc.ListHomeFeeds(ctx, pc, 1, 20)
		require.NoError(t, err)
		assert.Contains(t, feedIDsByType(t, feed, "notice"), testHomeNotice,
			"通知按家庭孩子的班级集下发，不按当前孩子过滤（student_id=%d）", studentID)
	}
}

// students[] 必须带 relation（问候语「妈妈 您好~」的数据来源），缺失时回退「家长」。
func TestParentHomeOverview_StudentsCarryRelation(t *testing.T) {
	// TC-PRN-HOME-001
	_, svc := setupHomeFeedFixture(t)
	ctx := context.Background()

	pc, err := svc.ResolveContext(ctx, testHomeParent, testHomeKG, ptrU64(testHomeStudentA))
	require.NoError(t, err)
	overview, err := svc.HomeOverview(ctx, pc)
	require.NoError(t, err)
	students, ok := overview["students"].([]map[string]any)
	require.True(t, ok, "students 必须是对象数组")

	byID := make(map[uint64]map[string]any, len(students))
	for _, st := range students {
		byID[st["id"].(uint64)] = st
	}
	require.Contains(t, byID, testHomeStudentA)
	require.Contains(t, byID, testHomeStudentB)
	assert.Equal(t, "mother", byID[testHomeStudentA]["relation"], "relation 输出 StudentParent 的原始 code，供前端映射")
	assert.Equal(t, "妈妈", byID[testHomeStudentA]["relation_label"])
	assert.Equal(t, "father", byID[testHomeStudentB]["relation"])
	assert.Equal(t, "爸爸", byID[testHomeStudentB]["relation_label"])

	// 无关系（relation 为空）的孩子：不报错，称呼回退「家长」。
	pcC, err := svc.ResolveContext(ctx, testHomeParent, testHomeKG, ptrU64(testHomeStudentC))
	require.NoError(t, err)
	overviewC, err := svc.HomeOverview(ctx, pcC)
	require.NoError(t, err)
	for _, st := range overviewC["students"].([]map[string]any) {
		if st["id"] == testHomeStudentC {
			assert.Equal(t, "", st["relation"])
			assert.Equal(t, "家长", st["relation_label"], "关系缺失必须回退「家长」")
			return
		}
	}
	t.Fatal("students[] 缺少学生 C")
}
