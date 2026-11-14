package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

// 集体对象仓储测试打真实 MySQL（与 diet_* 仓储测试同源）。
// 每个测试用独立高位园所 ID，清理一律原始 DELETE 硬删（CLAUDE.md 已知坑）。
const (
	testDietGroupObjectRepoKG      uint64 = 987011
	testDietGroupObjectRepoOtherKG uint64 = 987012
)

func openDietGroupObjectRepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := openDietRepoTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&domain.Kindergarten{},
		&domain.AcademicYear{},
		&domain.Class{},
		&domain.DietGroupObject{},
		&domain.DietGroupObjectMeal{},
		&domain.DietGroupObjectClass{},
	), "auto migrate group object tables")
	return db
}

func cleanupDietGroupObjectRepoData(t *testing.T, db *gorm.DB) {
	t.Helper()
	kgIDs := []uint64{testDietGroupObjectRepoKG, testDietGroupObjectRepoOtherKG}
	// 先删子表再删父表：关联表对主表、班级对园所都有真实外键。
	for _, sql := range []string{
		"DELETE FROM diet_group_object_classes WHERE kindergarten_id IN ?",
		"DELETE FROM diet_group_object_meals WHERE kindergarten_id IN ?",
		"DELETE FROM diet_group_objects WHERE kindergarten_id IN ?",
		"DELETE FROM classes WHERE kindergarten_id IN ?",
		"DELETE FROM academic_years WHERE kindergarten_id IN ?",
		"DELETE FROM kindergartens WHERE id IN ?",
	} {
		dietHardDelete(t, db, sql, kgIDs)
	}
}

type dietGroupObjectRepoFixture struct {
	db         *gorm.DB
	repo       *DietGroupObjectRepo
	year       *domain.AcademicYear
	otherYear  *domain.AcademicYear
	class1     *domain.Class
	class2     *domain.Class
	otherClass *domain.Class
}

func newDietGroupObjectRepoFixture(t *testing.T) *dietGroupObjectRepoFixture {
	t.Helper()
	db := openDietGroupObjectRepoTestDB(t)
	cleanupDietGroupObjectRepoData(t, db)
	t.Cleanup(func() { cleanupDietGroupObjectRepoData(t, db) })

	require.NoError(t, db.Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: testDietGroupObjectRepoKG},
		Name:      "DGO仓储测试园",
	}).Error)
	require.NoError(t, db.Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: testDietGroupObjectRepoOtherKG},
		Name:      "DGO仓储他园",
	}).Error)

	f := &dietGroupObjectRepoFixture{db: db, repo: NewDietGroupObjectRepo(db)}
	f.year = &domain.AcademicYear{
		KindergartenID: testDietGroupObjectRepoKG, Year: "2026-2027", IsCurrent: true,
	}
	f.otherYear = &domain.AcademicYear{
		KindergartenID: testDietGroupObjectRepoKG, Year: "2025-2026",
	}
	for _, year := range []*domain.AcademicYear{f.year, f.otherYear} {
		require.NoError(t, db.Create(year).Error)
	}

	f.class1 = &domain.Class{
		KindergartenID: testDietGroupObjectRepoKG, Name: "小一班", Grade: "小班",
		AcademicYear: "2026-2027", Status: 1,
	}
	f.class2 = &domain.Class{
		KindergartenID: testDietGroupObjectRepoKG, Name: "中一班", Grade: "中班",
		AcademicYear: "2026-2027", Status: 1,
	}
	f.otherClass = &domain.Class{
		KindergartenID: testDietGroupObjectRepoOtherKG, Name: "他园班", Grade: "小班",
		AcademicYear: "2026-2027", Status: 1,
	}
	for _, class := range []*domain.Class{f.class1, f.class2, f.otherClass} {
		require.NoError(t, db.Create(class).Error)
	}
	return f
}

// newTestDietGroupObject 构造带 6 个餐次与默认「五餐启用」的对象。
func newTestDietGroupObject(
	kindergartenID, yearID uint64,
	name string,
	classIDs ...uint64,
) *domain.DietGroupObject {
	obj := &domain.DietGroupObject{
		KindergartenID: kindergartenID,
		AcademicYearID: yearID,
		Name:           name,
		CareType:       domain.DietCareTypeDayCare,
		FoodInputType:  domain.DietFoodInputTypeMarketWeight,
		RatioMode:      domain.DietRatioModeStandard,
		PolicyName:     "DRIs2000",
	}
	specs := []struct {
		key     domain.DietMealKey
		at      string
		percent float64
		enabled bool
	}{
		{domain.DietMealBreakfast, "07:00", 25, true},
		{domain.DietMealMorningSnack, "09:00", 5, true},
		{domain.DietMealLunch, "11:00", 35, true},
		{domain.DietMealAfternoonSnack, "14:30", 5, true},
		{domain.DietMealDinner, "17:00", 25, true},
		{domain.DietMealEveningSnack, "20:00", 5, false},
	}
	for i, spec := range specs {
		obj.Meals = append(obj.Meals, domain.DietGroupObjectMeal{
			KindergartenID: kindergartenID,
			MealKey:        spec.key,
			Name:           dietRepoTestMealName(spec.key),
			Time:           spec.at,
			EnergyPercent:  spec.percent,
			Sort:           i + 1,
			Enabled:        spec.enabled,
		})
	}
	for _, classID := range classIDs {
		obj.Classes = append(obj.Classes, domain.DietGroupObjectClass{
			KindergartenID: kindergartenID,
			ClassID:        classID,
		})
	}
	return obj
}

func dietRepoTestMealName(key domain.DietMealKey) string {
	switch key {
	case domain.DietMealBreakfast:
		return "早餐"
	case domain.DietMealMorningSnack:
		return "早点"
	case domain.DietMealLunch:
		return "午餐"
	case domain.DietMealAfternoonSnack:
		return "午点"
	case domain.DietMealDinner:
		return "晚餐"
	case domain.DietMealEveningSnack:
		return "晚点"
	default:
		return ""
	}
}

func TestDietGroupObjectRepo_CreateGetListAndCountByName(t *testing.T) {
	f := newDietGroupObjectRepoFixture(t)
	ctx := context.Background()

	first := newTestDietGroupObject(testDietGroupObjectRepoKG, f.year.ID, "全体幼儿", f.class1.ID, f.class2.ID)
	require.NoError(t, f.repo.Create(ctx, first))
	require.NotZero(t, first.ID)
	require.Len(t, first.Meals, 6)
	for _, meal := range first.Meals {
		assert.NotZero(t, meal.ID, "餐次必须同事务落库")
		assert.Equal(t, first.ID, meal.GroupObjectID)
		assert.Equal(t, testDietGroupObjectRepoKG, meal.KindergartenID)
	}
	for _, link := range first.Classes {
		assert.NotZero(t, link.ID, "班级关联必须同事务落库")
		assert.Equal(t, first.ID, link.GroupObjectID)
	}

	second := newTestDietGroupObject(testDietGroupObjectRepoKG, f.year.ID, "日托班", f.class1.ID)
	require.NoError(t, f.repo.Create(ctx, second))
	otherYear := newTestDietGroupObject(testDietGroupObjectRepoKG, f.otherYear.ID, "全体幼儿", f.class1.ID)
	require.NoError(t, f.repo.Create(ctx, otherYear))
	otherKG := newTestDietGroupObject(testDietGroupObjectRepoOtherKG, f.year.ID, "全体幼儿", f.otherClass.ID)
	require.NoError(t, f.repo.Create(ctx, otherKG))

	got, err := f.repo.Get(ctx, testDietGroupObjectRepoKG, first.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "全体幼儿", got.Name)
	assert.Equal(t, f.year.ID, got.AcademicYearID)
	assert.Equal(t, domain.DietCareTypeDayCare, got.CareType)
	assert.Equal(t, "DRIs2000", got.PolicyName)
	require.Len(t, got.Meals, 6)
	assert.Equal(t, domain.DietMealBreakfast, got.Meals[0].MealKey, "餐次按 sort 升序返回")
	assert.False(t, got.Meals[5].Enabled)
	require.Len(t, got.Classes, 2)

	crossKG, err := f.repo.Get(ctx, testDietGroupObjectRepoOtherKG, first.ID)
	require.NoError(t, err)
	assert.Nil(t, crossKG, "跨园读取必须返回 nil")

	missing, err := f.repo.Get(ctx, testDietGroupObjectRepoKG, 1<<40)
	require.NoError(t, err)
	assert.Nil(t, missing)

	rows, total, err := f.repo.List(ctx, testDietGroupObjectRepoKG, f.year.ID, "", 0, 10)
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	require.Len(t, rows, 2)
	assert.Equal(t, second.ID, rows[0].ID, "按创建时间倒序：后建的在前")
	assert.Equal(t, first.ID, rows[1].ID)
	require.Len(t, rows[1].Meals, 6, "列表预加载餐次")

	filtered, filteredTotal, err := f.repo.List(ctx, testDietGroupObjectRepoKG, f.year.ID, " 日托 ", 0, 10)
	require.NoError(t, err)
	assert.EqualValues(t, 1, filteredTotal)
	require.Len(t, filtered, 1)
	assert.Equal(t, second.ID, filtered[0].ID)

	otherYearRows, otherYearTotal, err := f.repo.List(
		ctx, testDietGroupObjectRepoKG, f.otherYear.ID, "", 0, 10)
	require.NoError(t, err)
	assert.EqualValues(t, 1, otherYearTotal)
	require.Len(t, otherYearRows, 1)
	assert.Equal(t, otherYear.ID, otherYearRows[0].ID)

	page, pageTotal, err := f.repo.List(ctx, testDietGroupObjectRepoKG, f.year.ID, "", 1, 1)
	require.NoError(t, err)
	assert.EqualValues(t, 2, pageTotal)
	require.Len(t, page, 1)
	assert.Equal(t, first.ID, page[0].ID)

	count, err := f.repo.CountByName(ctx, testDietGroupObjectRepoKG, f.year.ID, "全体幼儿", 0)
	require.NoError(t, err)
	assert.EqualValues(t, 1, count)
	count, err = f.repo.CountByName(ctx, testDietGroupObjectRepoKG, f.year.ID, "全体幼儿", first.ID)
	require.NoError(t, err)
	assert.Zero(t, count, "排除自身后不得再命中")
	count, err = f.repo.CountByName(ctx, testDietGroupObjectRepoOtherKG, f.year.ID, "全体幼儿", 0)
	require.NoError(t, err)
	assert.EqualValues(t, 1, count, "名称唯一按园所隔离")
}

func TestDietGroupObjectRepo_UpdateReplacesLinksAndRestoresSoftDeleted(t *testing.T) {
	f := newDietGroupObjectRepoFixture(t)
	ctx := context.Background()

	obj := newTestDietGroupObject(testDietGroupObjectRepoKG, f.year.ID, "全体幼儿", f.class1.ID, f.class2.ID)
	require.NoError(t, f.repo.Create(ctx, obj))

	// 修改：改名、只保留 class2、改早餐时间、停用晚餐。
	obj.Name = "全体幼儿（修改）"
	obj.Meals[0].Time = "07:15"
	obj.Meals[4].Enabled = false
	obj.Classes = []domain.DietGroupObjectClass{{
		KindergartenID: testDietGroupObjectRepoKG,
		ClassID:        f.class2.ID,
	}}
	require.NoError(t, f.repo.Update(ctx, obj))

	got, err := f.repo.Get(ctx, testDietGroupObjectRepoKG, obj.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "全体幼儿（修改）", got.Name)
	assert.Equal(t, "07:15", got.Meals[0].Time)
	assert.False(t, got.Meals[4].Enabled)
	require.Len(t, got.Classes, 1)
	assert.Equal(t, f.class2.ID, got.Classes[0].ClassID)

	// 被移除的 class1 关联必须硬删，不能留下软删除行占键。
	var rawClassLinks int64
	require.NoError(t, f.db.Unscoped().Model(&domain.DietGroupObjectClass{}).
		Where("group_object_id = ?", obj.ID).Count(&rawClassLinks).Error)
	assert.EqualValues(t, 1, rawClassLinks)
	var rawMeals int64
	require.NoError(t, f.db.Unscoped().Model(&domain.DietGroupObjectMeal{}).
		Where("group_object_id = ?", obj.ID).Count(&rawMeals).Error)
	assert.EqualValues(t, 6, rawMeals, "餐次按业务键 upsert，不得重复插入")

	// 软删除现有班级关联与一个餐次行，再用同一业务键保存：必须复活而不是撞唯一索引。
	require.NoError(t, f.db.Where("group_object_id = ? AND class_id = ?", obj.ID, f.class2.ID).
		Delete(&domain.DietGroupObjectClass{}).Error)
	require.NoError(t, f.db.Where("group_object_id = ? AND meal_key = ?", obj.ID, domain.DietMealBreakfast).
		Delete(&domain.DietGroupObjectMeal{}).Error)

	obj.Classes = []domain.DietGroupObjectClass{
		{KindergartenID: testDietGroupObjectRepoKG, ClassID: f.class1.ID},
		{KindergartenID: testDietGroupObjectRepoKG, ClassID: f.class2.ID},
	}
	require.NoError(t, f.repo.Update(ctx, obj), "软删除行占键时必须靠 upsert 复活而不是撞键")

	restored, err := f.repo.Get(ctx, testDietGroupObjectRepoKG, obj.ID)
	require.NoError(t, err)
	require.NotNil(t, restored)
	require.Len(t, restored.Classes, 2)
	assert.Equal(t, "07:15", restored.Meals[0].Time, "复活的餐次保留本次保存值")
	require.Len(t, restored.Meals, 6)

	var activeClassLinks, allClassLinks int64
	require.NoError(t, f.db.Model(&domain.DietGroupObjectClass{}).
		Where("group_object_id = ?", obj.ID).Count(&activeClassLinks).Error)
	require.NoError(t, f.db.Unscoped().Model(&domain.DietGroupObjectClass{}).
		Where("group_object_id = ?", obj.ID).Count(&allClassLinks).Error)
	assert.EqualValues(t, 2, activeClassLinks)
	assert.EqualValues(t, 2, allClassLinks, "复用软删除行后不得残留额外行")

	missing := newTestDietGroupObject(testDietGroupObjectRepoKG, f.year.ID, "不存在", f.class1.ID)
	missing.ID = 1 << 40
	err = f.repo.Update(ctx, missing)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
