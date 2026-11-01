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
	testDietFoodKG      uint64 = 990942
	testDietFoodOtherKG uint64 = 990943
	testDietPresetKey          = "s1b-test-preset"
	testDietCatParent          = "S1B测试谷类"
	testDietCatChild           = "S1B测试小麦"
	testDietCatOther           = "S1B测试肉类"
)

func cleanupDietFoodData(t *testing.T, db *gorm.DB) {
	t.Helper()
	dietHardDelete(t, db, "DELETE FROM diet_foods WHERE kindergarten_id IN ?", []uint64{testDietFoodKG, testDietFoodOtherKG})
	dietHardDelete(t, db, "DELETE FROM diet_foods WHERE kindergarten_id = 0 AND preset_key = ?", testDietPresetKey)
	dietHardDelete(t, db, "DELETE FROM diet_food_categories WHERE name IN ?",
		[]string{testDietCatParent, testDietCatChild, testDietCatOther})
}

func newTestFood(kindergartenID uint64, name string) *domain.Food {
	return &domain.Food{
		KindergartenID: kindergartenID,
		Name:           name,
		SpellCode:      "S1B",
		EdiblePercent:  100,
		Source:         domain.DietFoodSourceCustom,
	}
}

// 契约要求：营养缺测存 NULL、显示「—」，任何求和都不得把 NULL 当 0。
// 这条测试锁住「NULL 进 NULL 出」，包括整体替换时把旧值改回 NULL。
func TestDietFoodRepo_NullNutrientsStayNull(t *testing.T) {
	db := openDietRepoTestDB(t)
	repo := NewDietFoodRepo(db)
	ctx := context.Background()

	cleanupDietFoodData(t, db)
	t.Cleanup(func() { cleanupDietFoodData(t, db) })

	protein := 8.3
	food := newTestFood(testDietFoodKG, "S1B测试面粉")
	food.Protein = &protein
	require.NoError(t, repo.Create(ctx, food))
	require.NotZero(t, food.ID)

	got, err := repo.FindByID(ctx, testDietFoodKG, food.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Nil(t, got.Energy, "未写入的能量必须读回 nil，不能变 0")
	assert.Nil(t, got.Heat)
	assert.Nil(t, got.Carb)
	require.NotNil(t, got.Protein)
	assert.InDelta(t, 8.3, *got.Protein, 1e-9)

	// 修改时营养整体替换：nil 必须写回 NULL。
	got.Protein = nil
	fats := 2.5
	got.Fats = &fats
	require.NoError(t, repo.Update(ctx, got))

	again, err := repo.FindByID(ctx, testDietFoodKG, food.ID)
	require.NoError(t, err)
	require.NotNil(t, again)
	assert.Nil(t, again.Protein, "整体替换后旧值必须清成 NULL")
	require.NotNil(t, again.Fats)
	assert.Equal(t, 2.5, *again.Fats)
}

// kg=0 是平台预置哨兵：租户查询不得回传预置行，按园所 ID 查详情也不能跨园命中。
func TestDietFoodRepo_TenantIsolationAndPresetFiltering(t *testing.T) {
	db := openDietRepoTestDB(t)
	repo := NewDietFoodRepo(db)
	ctx := context.Background()

	cleanupDietFoodData(t, db)
	t.Cleanup(func() { cleanupDietFoodData(t, db) })

	preset := newTestFood(domain.DietPlatformKindergartenID, "S1B测试预置食物")
	preset.PresetKey = testDietPresetKey
	preset.Source = domain.DietFoodSourceSystem
	require.NoError(t, repo.Create(ctx, preset))

	tenant := newTestFood(testDietFoodKG, "S1B测试园所食物")
	require.NoError(t, repo.Create(ctx, tenant))

	rows, total, err := repo.List(ctx, testDietFoodKG, DietFoodListFilter{}, 0, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, rows, 1)
	assert.Equal(t, tenant.ID, rows[0].ID, "租户列表不得返回 kg=0 预置行")

	presetRows, presetTotal, err := repo.List(ctx, domain.DietPlatformKindergartenID,
		DietFoodListFilter{Keyword: "S1B测试预置"}, 0, 20)
	require.NoError(t, err)
	require.Equal(t, int64(1), presetTotal)
	require.Len(t, presetRows, 1)
	assert.Equal(t, preset.ID, presetRows[0].ID, "kg=0 查询只返回预置行")

	missing, err := repo.FindByID(ctx, testDietFoodKG, preset.ID)
	require.NoError(t, err)
	assert.Nil(t, missing, "按园所查询不得命中 kg=0 预置行")

	other, err := repo.FindByID(ctx, testDietFoodOtherKG, tenant.ID)
	require.NoError(t, err)
	assert.Nil(t, other, "跨园详情必须按不存在处理")

	count, err := repo.CountByKindergarten(ctx, testDietFoodKG)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	presetCount, err := repo.CountByKindergarten(ctx, domain.DietPlatformKindergartenID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, presetCount, int64(1))
}

func TestDietFoodRepo_ListCategoryAndKeywordFilters(t *testing.T) {
	db := openDietRepoTestDB(t)
	repo := NewDietFoodRepo(db)
	ctx := context.Background()

	cleanupDietFoodData(t, db)
	t.Cleanup(func() { cleanupDietFoodData(t, db) })

	parent := &domain.FoodCategory{Name: testDietCatParent, Level: 1, Sort: 1}
	require.NoError(t, db.Create(parent).Error)
	child := &domain.FoodCategory{ParentID: parent.ID, Name: testDietCatChild, Level: 2, Sort: 1}
	require.NoError(t, db.Create(child).Error)
	otherCat := &domain.FoodCategory{Name: testDietCatOther, Level: 1, Sort: 2}
	require.NoError(t, db.Create(otherCat).Error)

	flour := newTestFood(testDietFoodKG, "S1B测试面粉")
	flour.CategoryID = child.ID
	require.NoError(t, repo.Create(ctx, flour))
	pork := newTestFood(testDietFoodKG, "S1B测试猪肉")
	pork.CategoryID = otherCat.ID
	require.NoError(t, repo.Create(ctx, pork))

	cases := []struct {
		name   string
		filter DietFoodListFilter
		wantID uint64
	}{
		{"按一级分类包含二级", DietFoodListFilter{CategoryID: parent.ID}, flour.ID},
		{"按二级分类", DietFoodListFilter{CategoryID: child.ID}, flour.ID},
		{"另一个一级分类", DietFoodListFilter{CategoryID: otherCat.ID}, pork.ID},
		{"按名称关键字", DietFoodListFilter{Keyword: "面粉"}, flour.ID},
		{"关键字无命中", DietFoodListFilter{Keyword: "不存在"}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rows, total, err := repo.List(ctx, testDietFoodKG, tc.filter, 0, 20)
			require.NoError(t, err)
			if tc.wantID == 0 {
				assert.Zero(t, total)
				assert.Empty(t, rows)
				return
			}
			require.Equal(t, int64(1), total)
			require.Len(t, rows, 1)
			assert.Equal(t, tc.wantID, rows[0].ID)
		})
	}

	first, totalFirst, err := repo.List(ctx, testDietFoodKG, DietFoodListFilter{}, 0, 1)
	require.NoError(t, err)
	second, totalSecond, err := repo.List(ctx, testDietFoodKG, DietFoodListFilter{}, 1, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(2), totalFirst, "分页 total 必须是不受 limit 影响的总数")
	assert.Equal(t, int64(2), totalSecond)
	require.Len(t, first, 1)
	require.Len(t, second, 1)
	assert.NotEqual(t, first[0].ID, second[0].ID, "两页不得返回同一行")
}

func TestDietFoodRepo_DeleteAndBatchDelete(t *testing.T) {
	db := openDietRepoTestDB(t)
	repo := NewDietFoodRepo(db)
	ctx := context.Background()

	cleanupDietFoodData(t, db)
	t.Cleanup(func() { cleanupDietFoodData(t, db) })

	foods := []*domain.Food{
		newTestFood(testDietFoodKG, "S1B测试食物一"),
		newTestFood(testDietFoodKG, "S1B测试食物二"),
		newTestFood(testDietFoodKG, "S1B测试食物三"),
	}
	for _, f := range foods {
		require.NoError(t, repo.Create(ctx, f))
	}

	affected, err := repo.Delete(ctx, testDietFoodKG, foods[0].ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), affected)

	got, err := repo.FindByID(ctx, testDietFoodKG, foods[0].ID)
	require.NoError(t, err)
	assert.Nil(t, got, "软删除后详情必须不可见")

	// 跨园删除不得命中本园行。
	affected, err = repo.Delete(ctx, testDietFoodOtherKG, foods[1].ID)
	require.NoError(t, err)
	assert.Zero(t, affected)
	still, err := repo.FindByID(ctx, testDietFoodKG, foods[1].ID)
	require.NoError(t, err)
	assert.NotNil(t, still, "跨园删除不得影响本园食物")

	affected, err = repo.BatchDelete(ctx, testDietFoodKG, []uint64{foods[1].ID, foods[2].ID})
	require.NoError(t, err)
	assert.Equal(t, int64(2), affected)

	count, err := repo.CountByKindergarten(ctx, testDietFoodKG)
	require.NoError(t, err)
	assert.Zero(t, count)

	affected, err = repo.BatchDelete(ctx, testDietFoodKG, nil)
	require.NoError(t, err)
	assert.Zero(t, affected, "空 ID 列表必须是无操作")
}
