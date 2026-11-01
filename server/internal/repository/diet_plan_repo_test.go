package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

const (
	testDietPlanKG      uint64 = 990946
	testDietPlanOtherKG uint64 = 990947
)

func cleanupDietPlanData(t *testing.T, db *gorm.DB) {
	t.Helper()
	kgIDs := []uint64{testDietPlanKG, testDietPlanOtherKG}
	dietHardDelete(t, db, "DELETE FROM diet_meal_plan_items WHERE kindergarten_id IN ?", kgIDs)
	dietHardDelete(t, db, "DELETE FROM diet_meal_plans WHERE kindergarten_id IN ?", kgIDs)
}

func newTestPlan(kindergartenID uint64) *domain.MealPlan {
	start := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
	return &domain.MealPlan{
		KindergartenID: kindergartenID,
		AcademicYearID: 31846,
		Name:           "S1B测试第九周",
		StartDate:      start,
		EndDate:        start.AddDate(0, 0, 6),
		DayNum:         2,
		WeekOfYear:     9,
		CreatedMethod:  domain.DietPlanCreatedByCustom,
		ConfigSnapshot: `{"coefficient":0.95,"meals":[]}`,
		Items: []domain.MealPlanItem{
			{
				Date: start, Weekday: 1, MealKey: domain.DietMealLunch,
				RecipeName: "S1B菜A", FoodName: "面粉", Grams: 30, Sort: 1,
				NutritionSnapshot: `{"edible_percent":100,"energy":286}`,
			},
			{
				Date: start, Weekday: 1, MealKey: domain.DietMealLunch,
				RecipeName: "S1B菜A", FoodName: "鸡蛋", Grams: 10, Sort: 2,
				NutritionSnapshot: `{"edible_percent":88,"energy":144}`,
			},
		},
	}
}

func TestDietMealPlanRepo_CreateDetailReplaceAndDelete(t *testing.T) {
	db := openDietRepoTestDB(t)
	repo := NewDietPlanRepo(db)
	ctx := context.Background()

	cleanupDietPlanData(t, db)
	t.Cleanup(func() { cleanupDietPlanData(t, db) })

	plan := newTestPlan(testDietPlanKG)
	require.NoError(t, repo.Create(ctx, plan))
	require.NotZero(t, plan.ID)
	require.Len(t, plan.Items, 2)
	for _, it := range plan.Items {
		assert.NotZero(t, it.ID, "明细必须同事务落库")
		assert.Equal(t, plan.ID, it.PlanID)
		assert.Equal(t, testDietPlanKG, it.KindergartenID)
	}

	detail, err := repo.FindByIDWithItems(ctx, testDietPlanKG, plan.ID)
	require.NoError(t, err)
	require.NotNil(t, detail)
	assert.Equal(t, "S1B测试第九周", detail.Name)
	assert.JSONEq(t, `{"coefficient":0.95,"meals":[]}`, detail.ConfigSnapshot)
	assert.Equal(t, "2026-10-26", detail.StartDate.Format("2006-01-02"))
	assert.Equal(t, "2026-11-01", detail.EndDate.Format("2006-01-02"))
	require.Len(t, detail.Items, 2)
	assert.Equal(t, "2026-10-26", detail.Items[0].Date.Format("2006-01-02"))
	assert.Equal(t, 1, detail.Items[0].Sort)
	assert.Equal(t, domain.DietMealLunch, detail.Items[0].MealKey)

	rows, total, err := repo.List(ctx, testDietPlanKG, 31846, "", 0, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, rows, 1)
	assert.Equal(t, plan.ID, rows[0].ID)
	assert.Empty(t, rows[0].Items, "列表不返回明细")

	_, otherYearTotal, err := repo.List(ctx, testDietPlanKG, 99999, "", 0, 20)
	require.NoError(t, err)
	assert.Zero(t, otherYearTotal, "按学年筛选不得命中其他学年")

	_, otherKgTotal, err := repo.List(ctx, testDietPlanOtherKG, 31846, "", 0, 20)
	require.NoError(t, err)
	assert.Zero(t, otherKgTotal, "跨园列表必须为空")

	crossKg, err := repo.FindByIDWithItems(ctx, testDietPlanOtherKG, plan.ID)
	require.NoError(t, err)
	assert.Nil(t, crossKg, "跨园详情必须按不存在处理")

	// 整单替换两次：第二次之后物理行数必须只剩新行，软删行不得累积。
	start := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
	require.NoError(t, repo.ReplaceItems(ctx, plan.ID, testDietPlanKG, []domain.MealPlanItem{
		{Date: start, Weekday: 1, MealKey: domain.DietMealBreakfast, RecipeName: "S1B菜B", FoodName: "牛奶", Grams: 200, Sort: 1, NutritionSnapshot: `{"edible_percent":100}`},
		{Date: start, Weekday: 1, MealKey: domain.DietMealLunch, RecipeName: "S1B菜C", FoodName: "米饭", Grams: 80, Sort: 1, NutritionSnapshot: `{"edible_percent":100}`},
		{Date: start.AddDate(0, 0, 1), Weekday: 2, MealKey: domain.DietMealLunch, RecipeName: "S1B菜D", FoodName: "面条", Grams: 70, Sort: 1, NutritionSnapshot: `{"edible_percent":100}`},
	}))
	replaced, err := repo.FindByIDWithItems(ctx, testDietPlanKG, plan.ID)
	require.NoError(t, err)
	require.NotNil(t, replaced)
	require.Len(t, replaced.Items, 3)

	require.NoError(t, repo.ReplaceItems(ctx, plan.ID, testDietPlanKG, []domain.MealPlanItem{
		{Date: start, Weekday: 1, MealKey: domain.DietMealDinner, RecipeName: "S1B菜E", FoodName: "鱼", Grams: 50, Sort: 1, NutritionSnapshot: `{"edible_percent":60}`},
	}))
	replacedAgain, err := repo.FindByIDWithItems(ctx, testDietPlanKG, plan.ID)
	require.NoError(t, err)
	require.NotNil(t, replacedAgain)
	require.Len(t, replacedAgain.Items, 1)

	var itemRows int64
	require.NoError(t, db.Unscoped().Model(&domain.MealPlanItem{}).
		Where("plan_id = ?", plan.ID).
		Count(&itemRows).Error)
	assert.Equal(t, int64(1), itemRows, "整单替换必须硬删旧明细")

	// 表头更新不得改写 config_snapshot。
	replacedAgain.Name = "S1B测试第九周（修订）"
	replacedAgain.DayNum = 3
	require.NoError(t, repo.Update(ctx, replacedAgain))

	updated, err := repo.FindByIDWithItems(ctx, testDietPlanKG, plan.ID)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "S1B测试第九周（修订）", updated.Name)
	assert.Equal(t, 3, updated.DayNum)
	assert.JSONEq(t, `{"coefficient":0.95,"meals":[]}`, updated.ConfigSnapshot,
		"配置快照在创建时固化，后续更新不得改写")

	affected, err := repo.Delete(ctx, testDietPlanKG, plan.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), affected)

	gone, err := repo.FindByIDWithItems(ctx, testDietPlanKG, plan.ID)
	require.NoError(t, err)
	assert.Nil(t, gone)

	require.NoError(t, db.Unscoped().Model(&domain.MealPlanItem{}).
		Where("plan_id = ?", plan.ID).
		Count(&itemRows).Error)
	assert.Zero(t, itemRows, "删除周食谱必须硬删明细")
}

// 跨园删除不得命中他园食谱。
func TestDietMealPlanRepo_DeleteIsTenantScoped(t *testing.T) {
	db := openDietRepoTestDB(t)
	repo := NewDietPlanRepo(db)
	ctx := context.Background()

	cleanupDietPlanData(t, db)
	t.Cleanup(func() { cleanupDietPlanData(t, db) })

	plan := newTestPlan(testDietPlanKG)
	require.NoError(t, repo.Create(ctx, plan))

	affected, err := repo.Delete(ctx, testDietPlanOtherKG, plan.ID)
	require.NoError(t, err)
	assert.Zero(t, affected)

	still, err := repo.FindByIDWithItems(ctx, testDietPlanKG, plan.ID)
	require.NoError(t, err)
	assert.NotNil(t, still, "跨园删除不得影响本园食谱")
}
