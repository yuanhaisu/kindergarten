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
	testDietRecipeKG      uint64 = 990944
	testDietRecipeOtherKG uint64 = 990945
)

func cleanupDietRecipeData(t *testing.T, db *gorm.DB) {
	t.Helper()
	kgIDs := []uint64{testDietRecipeKG, testDietRecipeOtherKG}
	dietHardDelete(t, db, "DELETE FROM diet_recipe_items WHERE kindergarten_id IN ?", kgIDs)
	dietHardDelete(t, db, "DELETE FROM diet_recipes WHERE kindergarten_id IN ?", kgIDs)
}

func newTestRecipe(kindergartenID uint64, name string, items ...domain.RecipeItem) *domain.Recipe {
	return &domain.Recipe{
		KindergartenID: kindergartenID,
		Name:           name,
		SpellCode:      "S1BCSYT",
		Items:          items,
	}
}

func TestDietRecipeRepo_CreateListFindUpdateDelete(t *testing.T) {
	db := openDietRepoTestDB(t)
	repo := NewDietRecipeRepo(db)
	ctx := context.Background()

	cleanupDietRecipeData(t, db)
	t.Cleanup(func() { cleanupDietRecipeData(t, db) })

	recipe := newTestRecipe(testDietRecipeKG, "S1B测试云吞",
		domain.RecipeItem{FoodID: 10001, Grams: 10, Sort: 1},
		domain.RecipeItem{FoodID: 10005, Grams: 5, Sort: 2},
		domain.RecipeItem{FoodID: 10011, Grams: 5, Sort: 3},
	)
	require.NoError(t, repo.Create(ctx, recipe))
	require.NotZero(t, recipe.ID)
	require.Len(t, recipe.Items, 3)
	for _, it := range recipe.Items {
		assert.NotZero(t, it.ID, "配料必须同事务落库")
		assert.Equal(t, recipe.ID, it.RecipeID)
		assert.Equal(t, testDietRecipeKG, it.KindergartenID)
	}

	rows, total, err := repo.List(ctx, testDietRecipeKG, "", 0, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, rows, 1)
	assert.Equal(t, recipe.ID, rows[0].ID)
	assert.Equal(t, "S1B测试云吞", rows[0].Name)
	assert.Equal(t, int64(3), rows[0].ItemCount)

	_, totalMiss, err := repo.List(ctx, testDietRecipeKG, "不存在", 0, 20)
	require.NoError(t, err)
	assert.Zero(t, totalMiss)

	detail, err := repo.FindByIDWithItems(ctx, testDietRecipeKG, recipe.ID)
	require.NoError(t, err)
	require.NotNil(t, detail)
	require.Len(t, detail.Items, 3)
	assert.Equal(t, uint64(10001), detail.Items[0].FoodID)
	assert.Equal(t, 1, detail.Items[0].Sort)
	assert.Equal(t, uint64(10011), detail.Items[2].FoodID)

	byName, err := repo.FindByName(ctx, testDietRecipeKG, "S1B测试云吞")
	require.NoError(t, err)
	require.NotNil(t, byName)
	assert.Equal(t, recipe.ID, byName.ID)

	otherName, err := repo.FindByName(ctx, testDietRecipeOtherKG, "S1B测试云吞")
	require.NoError(t, err)
	assert.Nil(t, otherName, "同名菜谱在不同园所互不可见")

	// 另一个园所可以有同名菜谱，两个园所的列表互不串数据。
	otherRecipe := newTestRecipe(testDietRecipeOtherKG, "S1B测试云吞",
		domain.RecipeItem{FoodID: 20001, Grams: 7, Sort: 1})
	require.NoError(t, repo.Create(ctx, otherRecipe))
	otherRows, otherTotal, err := repo.List(ctx, testDietRecipeOtherKG, "", 0, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(1), otherTotal)
	require.Len(t, otherRows, 1)
	assert.Equal(t, otherRecipe.ID, otherRows[0].ID)

	// 改名 + 整体替换配料。
	detail.Name = "S1B测试云吞（改）"
	detail.SpellCode = "S1BCSYTG"
	detail.Items = []domain.RecipeItem{
		{FoodID: 10001, Grams: 12, Sort: 1},
		{FoodID: 10005, Grams: 5, Sort: 2},
	}
	require.NoError(t, repo.Update(ctx, detail))

	updated, err := repo.FindByIDWithItems(ctx, testDietRecipeKG, recipe.ID)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "S1B测试云吞（改）", updated.Name)
	require.Len(t, updated.Items, 2)
	assert.InDelta(t, 12.0, updated.Items[0].Grams, 1e-9)

	updatedRows, _, err := repo.List(ctx, testDietRecipeKG, "（改）", 0, 20)
	require.NoError(t, err)
	require.Len(t, updatedRows, 1)
	assert.Equal(t, int64(2), updatedRows[0].ItemCount, "整组替换后配料数必须重算")

	affected, err := repo.Delete(ctx, testDietRecipeKG, recipe.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), affected)

	gone, err := repo.FindByIDWithItems(ctx, testDietRecipeKG, recipe.ID)
	require.NoError(t, err)
	assert.Nil(t, gone)

	var itemRows int64
	require.NoError(t, db.Unscoped().Model(&domain.RecipeItem{}).
		Where("recipe_id = ?", recipe.ID).
		Count(&itemRows).Error)
	assert.Zero(t, itemRows, "删除菜谱必须硬删配料，不留软删行")
}

func TestDietRecipeRepo_SuggestMatchesNameAndSpellPrefix(t *testing.T) {
	db := openDietRepoTestDB(t)
	repo := NewDietRecipeRepo(db)
	ctx := context.Background()

	cleanupDietRecipeData(t, db)
	t.Cleanup(func() { cleanupDietRecipeData(t, db) })

	wonton := newTestRecipe(testDietRecipeKG, "S1B测试云吞", domain.RecipeItem{FoodID: 1, Grams: 10, Sort: 1})
	wonton.SpellCode = "S1BCSYT"
	noodle := newTestRecipe(testDietRecipeKG, "S1B测试面条", domain.RecipeItem{FoodID: 2, Grams: 20, Sort: 1})
	noodle.SpellCode = "S1BCSMT"
	require.NoError(t, repo.Create(ctx, wonton))
	require.NoError(t, repo.Create(ctx, noodle))

	byName, err := repo.Suggest(ctx, testDietRecipeKG, "云吞", 20)
	require.NoError(t, err)
	require.Len(t, byName, 1)
	assert.Equal(t, wonton.ID, byName[0].ID)
	assert.Equal(t, int64(1), byName[0].ItemCount)

	bySpellPrefix, err := repo.Suggest(ctx, testDietRecipeKG, "S1BCSY", 20)
	require.NoError(t, err)
	require.Len(t, bySpellPrefix, 1)
	assert.Equal(t, wonton.ID, bySpellPrefix[0].ID)

	lowerSpell, err := repo.Suggest(ctx, testDietRecipeKG, "s1bcsy", 20)
	require.NoError(t, err)
	require.Len(t, lowerSpell, 1, "拼音码前缀匹配必须忽略大小写")

	all, err := repo.Suggest(ctx, testDietRecipeKG, "", 20)
	require.NoError(t, err)
	assert.Len(t, all, 2)

	limited, err := repo.Suggest(ctx, testDietRecipeKG, "", 1)
	require.NoError(t, err)
	assert.Len(t, limited, 1)
}
