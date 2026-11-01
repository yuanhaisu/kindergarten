package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

const (
	testDietTemplateKG      uint64 = 990948
	testDietTemplateOtherKG uint64 = 990949
)

func cleanupDietTemplateData(t *testing.T, db *gorm.DB) {
	t.Helper()
	kgIDs := []uint64{testDietTemplateKG, testDietTemplateOtherKG}
	dietHardDelete(t, db, "DELETE FROM diet_template_plan_items WHERE kindergarten_id IN ?", kgIDs)
	dietHardDelete(t, db, "DELETE FROM diet_template_plans WHERE kindergarten_id IN ?", kgIDs)
}

func TestDietTemplateRepo_ListAndDetailWithItems(t *testing.T) {
	db := openDietRepoTestDB(t)
	repo := NewDietTemplateRepo(db)
	ctx := context.Background()

	cleanupDietTemplateData(t, db)
	t.Cleanup(func() { cleanupDietTemplateData(t, db) })

	tpl := &domain.TemplatePlan{
		KindergartenID: testDietTemplateKG,
		TemplateKey:    "s1b-test-template",
		Name:           "S1B测试模板",
		DayNum:         5,
		MealKeys:       "breakfast,lunch,dinner",
		Sort:           1,
	}
	require.NoError(t, db.Omit("Items").Create(tpl).Error)
	require.NotZero(t, tpl.ID)

	items := []domain.TemplatePlanItem{
		{KindergartenID: testDietTemplateKG, TemplateID: tpl.ID, Weekday: 2, MealKey: domain.DietMealLunch, RecipeName: "S1B菜二", FoodID: 2, FoodName: "面条", Grams: 70, Sort: 2},
		{KindergartenID: testDietTemplateKG, TemplateID: tpl.ID, Weekday: 1, MealKey: domain.DietMealBreakfast, RecipeName: "S1B菜一", FoodID: 1, FoodName: "牛奶", Grams: 200, Sort: 1},
		{KindergartenID: testDietTemplateKG, TemplateID: tpl.ID, Weekday: 1, MealKey: domain.DietMealLunch, RecipeName: "S1B菜三", FoodID: 3, FoodName: "米饭", Grams: 80, Sort: 1},
	}
	require.NoError(t, db.Create(&items).Error)

	rows, err := repo.List(ctx, testDietTemplateKG)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, tpl.ID, rows[0].ID)
	assert.Equal(t, "S1B测试模板", rows[0].Name)
	assert.Equal(t, 5, rows[0].DayNum)
	assert.Equal(t, "breakfast,lunch,dinner", rows[0].MealKeys)
	assert.Equal(t, int64(3), rows[0].ItemCount)

	detail, err := repo.FindByIDWithItems(ctx, testDietTemplateKG, tpl.ID)
	require.NoError(t, err)
	require.NotNil(t, detail)
	require.Len(t, detail.Items, 3)
	assert.Equal(t, 1, detail.Items[0].Weekday, "模板明细按 weekday 升序")
	assert.Equal(t, "S1B菜一", detail.Items[0].RecipeName)
	assert.Equal(t, "牛奶", detail.Items[0].FoodName)

	otherRows, err := repo.List(ctx, testDietTemplateOtherKG)
	require.NoError(t, err)
	assert.Empty(t, otherRows, "模板按园隔离")

	otherDetail, err := repo.FindByIDWithItems(ctx, testDietTemplateOtherKG, tpl.ID)
	require.NoError(t, err)
	assert.Nil(t, otherDetail, "跨园模板详情必须按不存在处理")
}
