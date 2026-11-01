package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

const testDietConfigKG uint64 = 990941

func cleanupDietConfigData(t *testing.T, db *gorm.DB) {
	t.Helper()
	dietHardDelete(t, db, "DELETE FROM diet_meal_config_meals WHERE kindergarten_id = ?", testDietConfigKG)
	dietHardDelete(t, db, "DELETE FROM diet_meal_configs WHERE kindergarten_id = ?", testDietConfigKG)
}

func newDietConfigFixture() *domain.MealConfig {
	return &domain.MealConfig{
		KindergartenID: testDietConfigKG,
		CareType:       domain.DietCareTypeDayCare,
		FoodInputType:  domain.DietFoodInputTypeMarketWeight,
		Meals: []domain.MealConfigMeal{
			// 故意打乱输入顺序：读回必须按 sort 排序，而不是插入顺序。
			{MealKey: domain.DietMealLunch, Name: "午餐", Time: "11:00", EnergyPercent: 35, Sort: 3, Enabled: true},
			{MealKey: domain.DietMealBreakfast, Name: "早餐", Time: "07:00", EnergyPercent: 25, Sort: 1, Enabled: true},
			{MealKey: domain.DietMealDinner, Name: "晚餐", Time: "17:00", EnergyPercent: 25, Sort: 5, Enabled: true},
			{MealKey: domain.DietMealMorningSnack, Name: "早点", Time: "09:00", EnergyPercent: 5, Sort: 2, Enabled: true},
			{MealKey: domain.DietMealEveningSnack, Name: "晚点", Time: "20:00", EnergyPercent: 5, Sort: 6, Enabled: false},
			{MealKey: domain.DietMealAfternoonSnack, Name: "午点", Time: "14:30", EnergyPercent: 5, Sort: 4, Enabled: true},
		},
	}
}

func TestDietConfigRepo_GetByKindergartenWithMeals(t *testing.T) {
	db := openDietRepoTestDB(t)
	repo := NewDietConfigRepo(db)
	ctx := context.Background()

	cleanupDietConfigData(t, db)
	t.Cleanup(func() { cleanupDietConfigData(t, db) })

	cfg := newDietConfigFixture()
	require.NoError(t, repo.Save(ctx, cfg))
	require.NotZero(t, cfg.ID, "Save 必须回填配置主键")

	got, err := repo.GetByKindergarten(ctx, testDietConfigKG)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, domain.DietCareTypeDayCare, got.CareType)
	assert.Equal(t, domain.DietFoodInputTypeMarketWeight, got.FoodInputType)
	require.Len(t, got.Meals, 6)

	wantOrder := []domain.DietMealKey{
		domain.DietMealBreakfast,
		domain.DietMealMorningSnack,
		domain.DietMealLunch,
		domain.DietMealAfternoonSnack,
		domain.DietMealDinner,
		domain.DietMealEveningSnack,
	}
	for i, key := range wantOrder {
		assert.Equal(t, key, got.Meals[i].MealKey, "第 %d 个餐次必须按 sort 排序", i+1)
		assert.Equal(t, i+1, got.Meals[i].Sort)
		assert.Equal(t, testDietConfigKG, got.Meals[i].KindergartenID)
		assert.Equal(t, cfg.ID, got.Meals[i].ConfigID)
	}
	assert.False(t, got.Meals[5].Enabled, "晚点默认不启用，false 必须原样读回")

	other, err := repo.GetByKindergarten(ctx, testDietConfigKG+1)
	require.NoError(t, err)
	assert.Nil(t, other, "未配置的园所必须返回 nil，不得串园")
}

// 连续保存两次：第二次走「读回 → 改一个值 → 再存」的真实路径。
// 若餐次用软删除，第二次插入会撞 (config_id, meal_key) 唯一索引——本测试锁住这条回归。
func TestDietConfigRepo_SaveTwiceReplacesMeals(t *testing.T) {
	db := openDietRepoTestDB(t)
	repo := NewDietConfigRepo(db)
	ctx := context.Background()

	cleanupDietConfigData(t, db)
	t.Cleanup(func() { cleanupDietConfigData(t, db) })

	require.NoError(t, repo.Save(ctx, newDietConfigFixture()))

	loaded, err := repo.GetByKindergarten(ctx, testDietConfigKG)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	require.Len(t, loaded.Meals, 6)

	// 晚餐在 fixture 里 sort=5，对应切片下标 4。
	require.Equal(t, domain.DietMealDinner, loaded.Meals[4].MealKey)
	loaded.Meals[4].EnergyPercent = 30
	loaded.Meals[4].Enabled = false
	require.NoError(t, repo.Save(ctx, loaded),
		"第二次保存必须成功：软删除行不得占用 (config_id, meal_key) 唯一键")

	got, err := repo.GetByKindergarten(ctx, testDietConfigKG)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Len(t, got.Meals, 6)
	assert.Equal(t, 30.0, got.Meals[4].EnergyPercent)
	assert.False(t, got.Meals[4].Enabled)

	var mealRows int64
	require.NoError(t, db.Unscoped().Model(&domain.MealConfigMeal{}).
		Where("config_id = ?", got.ID).
		Count(&mealRows).Error)
	assert.Equal(t, int64(6), mealRows, "整组保存必须硬删旧餐次，物理行数保持 6")

	var configRows int64
	require.NoError(t, db.Unscoped().Model(&domain.MealConfig{}).
		Where("kindergarten_id = ?", testDietConfigKG).
		Count(&configRows).Error)
	assert.Equal(t, int64(1), configRows, "供餐配置是园所单例，不得新增第二条")
}
