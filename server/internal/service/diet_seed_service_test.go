package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// 膳食种子测试打真实 MySQL（与 repository/diet_* 测试同源）。
// 清理一律原始 DELETE 硬删：软删除行会占唯一索引，重跑撞键（CLAUDE.md 已知坑）。
// 官方分类与 DRIs 是平台基线数据，测试只断言不清理。
const (
	dietSeedTestKG       uint64 = 990990
	dietSeedTestKGOther  uint64 = 990991
	dietSeedTestPreset          = "s5-test-"
	dietSeedTestCategory        = "S5测试"
)

func openDietSeedTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	cfgPath := filepath.Join("..", "..", "configs", "app.ini")
	if _, err := os.Stat(cfgPath); err != nil {
		t.Skipf("跳过：找不到 %s（本地无 DB 配置）", cfgPath)
	}
	cfg, err := config.Load(cfgPath)
	require.NoError(t, err, "load config")
	if cfg.MySQL.DSN == "" {
		t.Skip("跳过：MySQL DSN 为空")
	}
	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err, "open db")
	require.NoError(t, db.AutoMigrate(
		&domain.MealConfig{}, &domain.MealConfigMeal{},
		&domain.FoodCategory{}, &domain.Food{},
		&domain.Recipe{}, &domain.RecipeItem{},
		&domain.MealPlan{}, &domain.MealPlanItem{},
		&domain.TemplatePlan{}, &domain.TemplatePlanItem{},
		&domain.DriAgeTarget{},
	), "auto migrate diet tables")
	return db
}

func cleanupDietSeedTestData(t *testing.T, db *gorm.DB) {
	t.Helper()
	stmts := []struct {
		sql  string
		args []any
	}{
		{"DELETE FROM diet_template_plan_items WHERE kindergarten_id IN ?", []any{[]uint64{dietSeedTestKG, dietSeedTestKGOther}}},
		{"DELETE FROM diet_template_plans WHERE kindergarten_id IN ?", []any{[]uint64{dietSeedTestKG, dietSeedTestKGOther}}},
		{"DELETE FROM diet_foods WHERE kindergarten_id IN ?", []any{[]uint64{dietSeedTestKG, dietSeedTestKGOther}}},
		{"DELETE FROM diet_foods WHERE kindergarten_id = 0 AND preset_key LIKE ?", []any{dietSeedTestPreset + "%"}},
		{"DELETE FROM diet_food_categories WHERE name LIKE ?", []any{dietSeedTestCategory + "%"}},
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt.sql, stmt.args...).Error; err != nil {
			t.Logf("cleanup %q: %v", stmt.sql, err)
		}
	}
}

func newDietSeedTestService(db *gorm.DB) *DietSeedService {
	return NewDietSeedService(
		db,
		repository.NewDietFoodRepo(db),
		repository.NewDietTemplateRepo(db),
		repository.NewDietDriRepo(db),
	)
}

// ensureDietSeedTestCategory 建一个测试专用分类树，避免依赖官方分类的自增 ID。
func ensureDietSeedTestCategory(t *testing.T, db *gorm.DB) uint64 {
	t.Helper()
	root := domain.FoodCategory{ParentID: 0, Level: 1, Name: dietSeedTestCategory + "一类"}
	require.NoError(t, db.Where("parent_id = ? AND name = ?", 0, root.Name).
		FirstOrCreate(&root).Error)
	leaf := domain.FoodCategory{ParentID: root.ID, Level: 2, Name: dietSeedTestCategory + "谷物"}
	require.NoError(t, db.Where("parent_id = ? AND name = ?", root.ID, leaf.Name).
		FirstOrCreate(&leaf).Error)
	return leaf.ID
}

// TestDietTemplateSpecsShape 守住 design D10 的模板规则：2 套、5 个工作日、
// 每餐 1 道菜、每道菜 1–2 种食物。
func TestDietTemplateSpecsShape(t *testing.T) {
	specs := dietTemplateSpecs()
	require.Len(t, specs, 2)
	keys := make(map[string]bool, len(specs))
	for _, spec := range specs {
		assert.NotEmpty(t, spec.Key)
		assert.NotEmpty(t, spec.Name)
		assert.False(t, keys[spec.Key], "template_key 必须唯一")
		keys[spec.Key] = true
		require.Len(t, spec.Days, 5)
		for i, day := range spec.Days {
			assert.Equal(t, i+1, day.Weekday)
			require.Len(t, day.Meals, 3)
			for _, meal := range day.Meals {
				assert.NotEmpty(t, meal.RecipeName)
				assert.NotEmpty(t, meal.Foods, "每餐必须有 1 道菜")
				assert.LessOrEqual(t, len(meal.Foods), 2, "每道菜最多 2 种食物")
				for _, food := range meal.Foods {
					assert.NotEmpty(t, food.Name)
					assert.Greater(t, food.Grams, 0.0)
				}
			}
		}
	}
}

// TestDietSeedGlobals 覆盖全局种子：20 个一级分类、44 行 DRIs，重复执行不新增。
func TestDietSeedGlobals(t *testing.T) {
	db := openDietSeedTestDB(t)
	svc := newDietSeedTestService(db)
	ctx := context.Background()

	_, err := svc.SeedDietGlobals(ctx)
	require.NoError(t, err, "first SeedDietGlobals")

	second, err := svc.SeedDietGlobals(ctx)
	require.NoError(t, err, "second SeedDietGlobals")
	assert.Zero(t, second.CategoriesAdded, "已存在的一级分类不得重复插入")
	assert.Zero(t, second.DriRowsAdded, "已存在的 DRIs 不得重复插入")

	var categoryCount int64
	require.NoError(t, db.Model(&domain.FoodCategory{}).
		Where("parent_id = 0 AND name IN ?", dietRootCategoryNames).
		Count(&categoryCount).Error)
	assert.EqualValues(t, len(dietRootCategoryNames), categoryCount, "20 个官方一级分类必须齐全")

	var driCount int64
	require.NoError(t, db.Model(&domain.DriAgeTarget{}).
		Where("age_band IN ?", []int{3, 4, 5, 6}).
		Count(&driCount).Error)
	assert.EqualValues(t, 44, driCount, "3/4/5/6 岁 × 11 项 = 44 行")

	var duplicates int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM (
		SELECT age_band, nutrient_key FROM diet_dri_age_targets
		WHERE age_band IN (3,4,5,6)
		GROUP BY age_band, nutrient_key HAVING COUNT(*) > 1
	) dup`).Scan(&duplicates).Error)
	assert.Zero(t, duplicates, "DRIs 不得有重复 (age_band, nutrient_key)")

	var energy domain.DriAgeTarget
	require.NoError(t, db.Where("age_band = ? AND nutrient_key = ?", 3, "energy").
		First(&energy).Error)
	assert.Equal(t, 1350.0, energy.Value, "3 岁能量取 DRIs2000 男值")

	assert.Len(t, dietRootCategoryNames, 20)
	for _, name := range dietRootCategoryNames {
		_, ok := dietGroup16ByRootCategory[name]
		assert.True(t, ok, "一级分类 %q 缺少 group16 映射，种子与食物服务的文案已漂移", name)
	}
}

// TestDietSeedCopiesPresetFoods 覆盖 TC-NEW-DIE-0009：复制 kg=0 预置行到园所，幂等且保留 NULL。
func TestDietSeedCopiesPresetFoods(t *testing.T) {
	db := openDietSeedTestDB(t)
	cleanupDietSeedTestData(t, db)
	t.Cleanup(func() { cleanupDietSeedTestData(t, db) })

	categoryID := ensureDietSeedTestCategory(t, db)
	presets := []domain.Food{
		{
			KindergartenID: domain.DietPlatformKindergartenID,
			Name:           "S5测试大米",
			SpellCode:      "S5CSDM",
			CategoryID:     categoryID,
			EdiblePercent:  100,
			Source:         domain.DietFoodSourceSystem,
			PresetKey:      dietSeedTestPreset + "rice",
			Energy:         floatPtr(123.4),
		},
		{
			KindergartenID: domain.DietPlatformKindergartenID,
			Name:           "S5测试青菜",
			SpellCode:      "S5CSQC",
			CategoryID:     categoryID,
			EdiblePercent:  90,
			Source:         domain.DietFoodSourceSystem,
			PresetKey:      dietSeedTestPreset + "greens",
			VitaminC:       floatPtr(45),
		},
	}
	require.NoError(t, db.Create(&presets).Error)

	svc := newDietSeedTestService(db)
	ctx := context.Background()
	result, err := svc.EnsureKindergartenSeeded(ctx, dietSeedTestKG)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, result.FoodsCopied, 2, "至少复制两条测试预置行")

	var copies []domain.Food
	require.NoError(t, db.Where("kindergarten_id = ? AND preset_key IN ?",
		dietSeedTestKG, []string{dietSeedTestPreset + "rice", dietSeedTestPreset + "greens"}).
		Order("preset_key ASC").Find(&copies).Error)
	require.Len(t, copies, 2, "TC-NEW-DIE-0009：新园应看到预置副本")

	for _, copy := range copies {
		assert.Equal(t, domain.DietFoodSourceSystem, copy.Source)
		assert.False(t, copy.IsCustomized, "复制出的副本不是园所修改行")
		assert.NotZero(t, copy.ID)
		assert.NotEqual(t, presets[0].ID, copy.ID)
	}
	// 按 preset_key 升序：greens 在前、rice 在后。
	assert.Equal(t, 45.0, *copies[0].VitaminC)
	assert.Nil(t, copies[1].VitaminC, "预置行缺失的营养项必须保持 NULL，不得写成 0")

	again, err := svc.EnsureKindergartenSeeded(ctx, dietSeedTestKG)
	require.NoError(t, err)
	assert.Zero(t, again.FoodsCopied, "第二次初始化不得重复复制")
	var count int64
	require.NoError(t, db.Model(&domain.Food{}).
		Where("kindergarten_id = ? AND preset_key IN ?", dietSeedTestKG,
			[]string{dietSeedTestPreset + "rice", dietSeedTestPreset + "greens"}).
		Count(&count).Error)
	assert.EqualValues(t, 2, count)
}

// dietSeedTemplateFoods 是覆盖两套模板各至少一道菜的食物集合。
var dietSeedTemplateFoods = []string{
	"稻米", "鸡蛋", "牛奶", "小米", "面条", "小白菜", "豆腐",
}

func createDietSeedTemplateFoods(t *testing.T, db *gorm.DB, kindergartenID uint64) {
	t.Helper()
	categoryID := ensureDietSeedTestCategory(t, db)
	for i, name := range dietSeedTemplateFoods {
		food := domain.Food{
			KindergartenID: kindergartenID,
			Name:           name,
			CategoryID:     categoryID,
			EdiblePercent:  100,
			Source:         domain.DietFoodSourceCustom,
			PresetKey:      dietSeedTestPreset + "tpl-" + string(rune('a'+i)),
		}
		require.NoError(t, db.Create(&food).Error)
	}
}

// TestDietTemplateSeedsTwoPlans 覆盖 design D10：有食物的园所种 2 套模板，重复播种幂等，
// 缺失食物跳过并汇总。
func TestDietTemplateSeedsTwoPlans(t *testing.T) {
	db := openDietSeedTestDB(t)
	cleanupDietSeedTestData(t, db)
	t.Cleanup(func() { cleanupDietSeedTestData(t, db) })

	createDietSeedTemplateFoods(t, db, dietSeedTestKG)
	svc := newDietSeedTestService(db)
	ctx := context.Background()

	result, err := svc.SeedDietTemplates(ctx, dietSeedTestKG)
	require.NoError(t, err)
	assert.Equal(t, 2, result.TemplatesCreated, "应种入 2 套模板")
	assert.False(t, result.SkippedNoFoods)
	assert.NotEmpty(t, result.MissingFoods, "初稿里未建库的食物应被跳过并汇总")
	assert.NotContains(t, result.MissingFoods, "稻米")

	expectedItems := 0
	for _, spec := range dietTemplateSpecs() {
		for _, day := range spec.Days {
			for _, meal := range day.Meals {
				for _, food := range meal.Foods {
					for _, name := range dietSeedTemplateFoods {
						if food.Name == name {
							expectedItems++
						}
					}
				}
			}
		}
	}
	assert.Equal(t, expectedItems, result.ItemsCreated, "只有能按名称解析的食物才写成模板条目")

	var templates []domain.TemplatePlan
	require.NoError(t, db.Where("kindergarten_id = ?", dietSeedTestKG).
		Order("sort ASC").Find(&templates).Error)
	require.Len(t, templates, 2)
	assert.Equal(t, "balanced_week", templates[0].TemplateKey)
	assert.Equal(t, "light_week", templates[1].TemplateKey)
	assert.Equal(t, 5, templates[0].DayNum)

	for _, tpl := range templates {
		var items []domain.TemplatePlanItem
		require.NoError(t, db.Where("template_id = ?", tpl.ID).Find(&items).Error)
		assert.NotEmpty(t, items, "模板 %s 应有条目", tpl.TemplateKey)
		for _, item := range items {
			assert.GreaterOrEqual(t, item.Weekday, 1)
			assert.LessOrEqual(t, item.Weekday, 5)
			assert.Contains(t, []domain.DietMealKey{
				domain.DietMealBreakfast, domain.DietMealLunch, domain.DietMealDinner,
			}, item.MealKey)
			assert.NotZero(t, item.FoodID)
			assert.Greater(t, item.Grams, 0.0)
		}
	}

	again, err := svc.SeedDietTemplates(ctx, dietSeedTestKG)
	require.NoError(t, err)
	assert.Zero(t, again.TemplatesCreated, "重复播种不得新增模板")
	var count int64
	require.NoError(t, db.Model(&domain.TemplatePlan{}).
		Where("kindergarten_id = ?", dietSeedTestKG).Count(&count).Error)
	assert.EqualValues(t, 2, count)
}

// TestDietTemplateSkipsNoFoods 覆盖「无食物跳过模板并告警」。
func TestDietTemplateSkipsNoFoods(t *testing.T) {
	db := openDietSeedTestDB(t)
	cleanupDietSeedTestData(t, db)
	t.Cleanup(func() { cleanupDietSeedTestData(t, db) })

	svc := newDietSeedTestService(db)
	result, err := svc.SeedDietTemplates(context.Background(), dietSeedTestKGOther)
	require.NoError(t, err)
	assert.True(t, result.SkippedNoFoods)
	assert.Zero(t, result.TemplatesCreated)
	assert.Zero(t, result.ItemsCreated)

	var count int64
	require.NoError(t, db.Model(&domain.TemplatePlan{}).
		Where("kindergarten_id = ?", dietSeedTestKGOther).Count(&count).Error)
	assert.Zero(t, count)
}

// TestDietTemplateSkipsMissingFoods 确认只写入能解析的食物，缺失项进 MissingFoods。
func TestDietTemplateSkipsMissingFoods(t *testing.T) {
	db := openDietSeedTestDB(t)
	cleanupDietSeedTestData(t, db)
	t.Cleanup(func() { cleanupDietSeedTestData(t, db) })

	categoryID := ensureDietSeedTestCategory(t, db)
	require.NoError(t, db.Create(&domain.Food{
		KindergartenID: dietSeedTestKGOther,
		Name:           "稻米",
		CategoryID:     categoryID,
		EdiblePercent:  100,
		Source:         domain.DietFoodSourceCustom,
		PresetKey:      dietSeedTestPreset + "only-rice",
	}).Error)

	svc := newDietSeedTestService(db)
	result, err := svc.SeedDietTemplates(context.Background(), dietSeedTestKGOther)
	require.NoError(t, err)
	assert.Equal(t, 2, result.TemplatesCreated, "稻米在两套模板里都有，仍应种出 2 套")
	assert.NotEmpty(t, result.MissingFoods)
	assert.NotContains(t, result.MissingFoods, "稻米")

	var items []domain.TemplatePlanItem
	require.NoError(t, db.Where("kindergarten_id = ?", dietSeedTestKGOther).Find(&items).Error)
	require.NotEmpty(t, items)
	for _, item := range items {
		assert.Equal(t, "稻米", item.FoodName)
	}
}
