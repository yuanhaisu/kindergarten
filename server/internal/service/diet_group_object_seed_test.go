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

// 集体对象种子测试打真实 MySQL（与 diet_seed_service_test.go 同源）。
// 清理一律原始 DELETE 硬删：软删除行会占唯一索引，重跑撞键（CLAUDE.md 已知坑）。
const dietGroupSeedTestKG uint64 = 990992

func openDietGroupObjectSeedTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := openDietSeedTestDB(t)
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

func cleanupDietGroupObjectSeedTestData(t *testing.T, db *gorm.DB) {
	t.Helper()
	kgIDs := []uint64{dietGroupSeedTestKG}
	// 先删子表再删父表：关联表与主表、班级与园所之间都有真实外键。
	for _, sql := range []string{
		"DELETE FROM diet_group_object_classes WHERE kindergarten_id IN ?",
		"DELETE FROM diet_group_object_meals WHERE kindergarten_id IN ?",
		"DELETE FROM diet_group_objects WHERE kindergarten_id IN ?",
		"DELETE FROM diet_meal_plan_items WHERE kindergarten_id IN ?",
		"DELETE FROM diet_meal_plans WHERE kindergarten_id IN ?",
		"DELETE FROM diet_template_plan_items WHERE kindergarten_id IN ?",
		"DELETE FROM diet_template_plans WHERE kindergarten_id IN ?",
		"DELETE FROM diet_foods WHERE kindergarten_id IN ?",
		"DELETE FROM classes WHERE kindergarten_id IN ?",
		"DELETE FROM academic_years WHERE kindergarten_id IN ?",
		"DELETE FROM kindergartens WHERE id IN ?",
	} {
		if err := db.Exec(sql, kgIDs).Error; err != nil {
			t.Logf("cleanup %q: %v", sql, err)
		}
	}
}

type dietGroupObjectSeedFixture struct {
	db             *gorm.DB
	svc            *DietSeedService
	currentYear    *domain.AcademicYear
	emptyYear      *domain.AcademicYear
	customYear     *domain.AcademicYear
	activeClassIDs []uint64
}

func newDietGroupObjectSeedFixture(t *testing.T) *dietGroupObjectSeedFixture {
	t.Helper()
	db := openDietGroupObjectSeedTestDB(t)
	cleanupDietGroupObjectSeedTestData(t, db)
	t.Cleanup(func() { cleanupDietGroupObjectSeedTestData(t, db) })

	require.NoError(t, db.Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: dietGroupSeedTestKG},
		Name:      "DGO测试园所",
	}).Error)

	f := &dietGroupObjectSeedFixture{db: db, svc: newDietSeedTestService(db)}

	f.currentYear = &domain.AcademicYear{
		KindergartenID: dietGroupSeedTestKG, Year: "2026-2027", IsCurrent: true,
	}
	f.emptyYear = &domain.AcademicYear{KindergartenID: dietGroupSeedTestKG, Year: "2025-2026"}
	f.customYear = &domain.AcademicYear{KindergartenID: dietGroupSeedTestKG, Year: "2024-2025"}
	for _, year := range []*domain.AcademicYear{f.currentYear, f.emptyYear, f.customYear} {
		require.NoError(t, db.Create(year).Error)
	}

	classes := []domain.Class{
		{KindergartenID: dietGroupSeedTestKG, Name: "小一班", Grade: "小班", AcademicYear: "2026-2027", Status: 1},
		{KindergartenID: dietGroupSeedTestKG, Name: "中一班", Grade: "中班", AcademicYear: "2026-2027", Status: 1},
	}
	require.NoError(t, db.Create(&classes).Error)
	f.activeClassIDs = []uint64{classes[0].ID, classes[1].ID}

	// Class.Status 带 default:1，Create 会把零值 0 换成默认值 1，因此停办班级必须显式更新。
	inactive := &domain.Class{
		KindergartenID: dietGroupSeedTestKG, Name: "已停办班", Grade: "大班",
		AcademicYear: "2026-2027", Status: 1,
	}
	require.NoError(t, db.Create(inactive).Error)
	require.NoError(t, db.Model(&domain.Class{}).Where("id = ?", inactive.ID).
		Update("status", 0).Error)

	// 已有对象的学年必须整年跳过：哪怕只有一条手工对象，也不得再种默认对象或回填计划。
	require.NoError(t, db.Create(&domain.DietGroupObject{
		KindergartenID: dietGroupSeedTestKG,
		AcademicYearID: f.customYear.ID,
		Name:           "中班对象",
		CareType:       domain.DietCareTypeDayCare,
		FoodInputType:  domain.DietFoodInputTypeMarketWeight,
		RatioMode:      domain.DietRatioModeStandard,
		PolicyName:     dietGroupObjectPolicyName,
	}).Error)
	return f
}

func (f *dietGroupObjectSeedFixture) createPlan(
	t *testing.T,
	yearID uint64,
	name string,
) *domain.MealPlan {
	t.Helper()
	plan := &domain.MealPlan{
		KindergartenID: dietGroupSeedTestKG,
		AcademicYearID: yearID,
		Name:           name,
		StartDate:      time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC),
		ConfigSnapshot: "{}",
	}
	require.NoError(t, f.db.Create(plan).Error)
	return plan
}

func (f *dietGroupObjectSeedFixture) objectForYear(t *testing.T, yearID uint64) domain.DietGroupObject {
	t.Helper()
	var object domain.DietGroupObject
	require.NoError(t, f.db.
		Where("kindergarten_id = ? AND academic_year_id = ?", dietGroupSeedTestKG, yearID).
		First(&object).Error)
	return object
}

// TestDietGroupObjectSeedCreatesDefaultsAndIsIdempotent 覆盖 design D6 的迁移规则：
// 每个学年一条「全体幼儿」、6 个餐次、学年班级（无班级回退全园）、旧计划回填，
// 连续执行两次不产生重复。
func TestDietGroupObjectSeedCreatesDefaultsAndIsIdempotent(t *testing.T) {
	f := newDietGroupObjectSeedFixture(t)
	ctx := context.Background()

	currentPlan := f.createPlan(t, f.currentYear.ID, "2026 第 1 周")
	emptyYearPlan := f.createPlan(t, f.emptyYear.ID, "2025 第 1 周")
	customYearPlan := f.createPlan(t, f.customYear.ID, "2024 第 1 周")

	first, err := f.svc.EnsureDietGroupObjects(ctx, dietGroupSeedTestKG)
	require.NoError(t, err)
	assert.Equal(t, 2, first.ObjectsCreated, "只有没有对象的两个学年需要种默认对象")
	assert.Equal(t, 12, first.MealsCreated, "每条对象固定 6 个餐次")
	assert.Equal(t, len(f.activeClassIDs)*2, first.ClassesLinked, "有班级按学年取；无班级回退全园在读班级")
	assert.EqualValues(t, 2, first.PlansBackfilled, "只回填被种了对象的学年")

	again, err := f.svc.EnsureDietGroupObjects(ctx, dietGroupSeedTestKG)
	require.NoError(t, err)
	assert.Zero(t, again.ObjectsCreated, "已有对象的学年必须整体跳过")
	assert.Zero(t, again.MealsCreated)
	assert.Zero(t, again.ClassesLinked)
	assert.Zero(t, again.PlansBackfilled)

	var objects []domain.DietGroupObject
	require.NoError(t, f.db.Where("kindergarten_id = ?", dietGroupSeedTestKG).
		Order("academic_year_id ASC").Find(&objects).Error)
	require.Len(t, objects, 3, "两个学年各一条默认对象，已有对象不被替换")

	current := f.objectForYear(t, f.currentYear.ID)
	assert.Equal(t, "全体幼儿", current.Name)
	assert.Equal(t, domain.DietCareTypeDayCare, current.CareType)
	assert.Equal(t, domain.DietFoodInputTypeMarketWeight, current.FoodInputType)
	assert.Equal(t, domain.DietRatioModeStandard, current.RatioMode)
	assert.Equal(t, "DRIs2000", current.PolicyName)

	var meals []domain.DietGroupObjectMeal
	require.NoError(t, f.db.Where("group_object_id = ?", current.ID).
		Order("sort ASC").Find(&meals).Error)
	require.Len(t, meals, 6, "默认对象固定 6 个餐次")
	expectedMeals := []struct {
		key           domain.DietMealKey
		name          string
		at            string
		energyPercent float64
		enabled       bool
	}{
		{domain.DietMealBreakfast, "早餐", "07:00", 25, true},
		{domain.DietMealMorningSnack, "早点", "09:00", 5, true},
		{domain.DietMealLunch, "午餐", "11:00", 35, true},
		{domain.DietMealAfternoonSnack, "午点", "14:30", 5, true},
		{domain.DietMealDinner, "晚餐", "17:00", 25, true},
		{domain.DietMealEveningSnack, "晚点", "20:00", 5, false},
	}
	for i, want := range expectedMeals {
		assert.Equal(t, want.key, meals[i].MealKey)
		assert.Equal(t, want.name, meals[i].Name)
		assert.Equal(t, want.at, meals[i].Time)
		assert.Equal(t, want.energyPercent, meals[i].EnergyPercent)
		assert.Equal(t, want.enabled, meals[i].Enabled)
		assert.Equal(t, i+1, meals[i].Sort)
		assert.Equal(t, dietGroupSeedTestKG, meals[i].KindergartenID)
	}

	var links []domain.DietGroupObjectClass
	require.NoError(t, f.db.Where("group_object_id = ?", current.ID).
		Order("class_id ASC").Find(&links).Error)
	require.Len(t, links, len(f.activeClassIDs), "状态非在读的班级不得进入就餐对象")
	for i, classID := range f.activeClassIDs {
		assert.Equal(t, classID, links[i].ClassID)
		assert.Equal(t, dietGroupSeedTestKG, links[i].KindergartenID)
	}

	// 该学年没有班级 → 回退到本园全部在读班级。
	emptyYearObject := f.objectForYear(t, f.emptyYear.ID)
	var fallbackLinks []domain.DietGroupObjectClass
	require.NoError(t, f.db.Where("group_object_id = ?", emptyYearObject.ID).
		Order("class_id ASC").Find(&fallbackLinks).Error)
	require.Len(t, fallbackLinks, len(f.activeClassIDs), "无匹配班级的学年回退本园在读班级")

	customObject := f.objectForYear(t, f.customYear.ID)
	assert.Equal(t, "中班对象", customObject.Name, "已有对象的学年不得被默认对象替换")

	reloadedCurrent := f.createPlanReload(t, currentPlan.ID)
	assert.Equal(t, current.ID, reloadedCurrent.GroupObjectID, "同学年旧计划回填到默认对象")
	reloadedEmpty := f.createPlanReload(t, emptyYearPlan.ID)
	assert.Equal(t, emptyYearObject.ID, reloadedEmpty.GroupObjectID)
	reloadedCustom := f.createPlanReload(t, customYearPlan.ID)
	assert.Zero(t, reloadedCustom.GroupObjectID, "已有对象的学年不种新对象，也不回填")
}

func (f *dietGroupObjectSeedFixture) createPlanReload(t *testing.T, planID uint64) domain.MealPlan {
	t.Helper()
	var plan domain.MealPlan
	require.NoError(t, f.db.First(&plan, planID).Error)
	return plan
}

// TestDietGroupObjectSeedRunsFromKindergartenSeed 守住种子入口：新园初始化路径经
// EnsureKindergartenSeeded 也要得到默认对象，且重复初始化幂等。
func TestDietGroupObjectSeedRunsFromKindergartenSeed(t *testing.T) {
	f := newDietGroupObjectSeedFixture(t)
	ctx := context.Background()

	_, err := f.svc.EnsureKindergartenSeeded(ctx, dietGroupSeedTestKG)
	require.NoError(t, err)

	var count int64
	require.NoError(t, f.db.Model(&domain.DietGroupObject{}).
		Where("kindergarten_id = ?", dietGroupSeedTestKG).Count(&count).Error)
	assert.EqualValues(t, 3, count, "EnsureKindergartenSeeded 必须触发默认对象种子（含已有对象学年）")

	_, err = f.svc.EnsureKindergartenSeeded(ctx, dietGroupSeedTestKG)
	require.NoError(t, err)
	require.NoError(t, f.db.Model(&domain.DietGroupObject{}).
		Where("kindergarten_id = ?", dietGroupSeedTestKG).Count(&count).Error)
	assert.EqualValues(t, 3, count, "重复初始化不得重复种对象")
}
