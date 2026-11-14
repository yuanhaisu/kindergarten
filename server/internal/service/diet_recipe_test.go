package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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
	"kindergarten-server/pkg/pagination"
)

// 菜谱库 service 测试打真实 MySQL（与 repository/其他 service 测试同源）。
// 清理一律原始 DELETE 硬删：软删除行会占唯一索引，重跑撞键（CLAUDE.md 已知坑）。
const (
	testDietRecipeSvcKG      uint64 = 990993
	testDietRecipeSvcOtherKG uint64 = 990994
)

func openDietRecipeServiceTestDB(t *testing.T) *gorm.DB {
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
		&domain.FoodCategory{}, &domain.Food{},
		&domain.Recipe{}, &domain.RecipeItem{},
		&domain.MealPlan{}, &domain.MealPlanItem{},
	), "auto migrate diet recipe tables")
	return db
}

func cleanupDietRecipeServiceData(t *testing.T, db *gorm.DB) {
	t.Helper()
	kgIDs := []uint64{testDietRecipeSvcKG, testDietRecipeSvcOtherKG}
	// 先删明细再删主表：plan 与 item 之间有真实外键。
	for _, sql := range []string{
		"DELETE FROM diet_meal_plan_items WHERE kindergarten_id IN ?",
		"DELETE FROM diet_meal_plans WHERE kindergarten_id IN ?",
		"DELETE FROM diet_recipe_items WHERE kindergarten_id IN ?",
		"DELETE FROM diet_recipes WHERE kindergarten_id IN ?",
		"DELETE FROM diet_foods WHERE kindergarten_id IN ?",
	} {
		if err := db.Exec(sql, kgIDs).Error; err != nil {
			t.Logf("cleanup %q: %v", sql, err)
		}
	}
	// 详情用例会插入 kg=0 的预置食物（模拟脏数据），按测试名前缀清理。
	if err := db.Exec(
		"DELETE FROM diet_foods WHERE kindergarten_id = 0 AND name LIKE ?",
		"S4测试平台预置%",
	).Error; err != nil {
		t.Logf("cleanup preset foods: %v", err)
	}
}

type dietRecipeFixture struct {
	db         *gorm.DB
	svc        *DietRecipeService
	foodRepo   *repository.DietFoodRepo
	recipeRepo *repository.DietRecipeRepo
	planRepo   *repository.DietPlanRepo
}

func newDietRecipeFixture(t *testing.T) *dietRecipeFixture {
	t.Helper()
	db := openDietRecipeServiceTestDB(t)
	cleanupDietRecipeServiceData(t, db)
	t.Cleanup(func() { cleanupDietRecipeServiceData(t, db) })

	foodRepo := repository.NewDietFoodRepo(db)
	recipeRepo := repository.NewDietRecipeRepo(db)
	return &dietRecipeFixture{
		db:         db,
		svc:        NewDietRecipeService(recipeRepo, foodRepo),
		foodRepo:   foodRepo,
		recipeRepo: recipeRepo,
		planRepo:   repository.NewDietPlanRepo(db),
	}
}

func (f *dietRecipeFixture) createFood(
	t *testing.T,
	kindergartenID uint64,
	name string,
	energy, protein float64,
) *domain.Food {
	t.Helper()
	food := &domain.Food{
		KindergartenID: kindergartenID,
		Name:           name,
		CategoryID:     1,
		EdiblePercent:  100,
		Source:         domain.DietFoodSourceCustom,
		Energy:         &energy,
		Protein:        &protein,
	}
	require.NoError(t, f.foodRepo.Create(context.Background(), food))
	return food
}

func (f *dietRecipeFixture) saveRecipe(
	t *testing.T,
	kindergartenID uint64,
	name string,
	replace bool,
	items ...DietRecipeItemInput,
) *DietRecipeSaveResult {
	t.Helper()
	result, err := f.svc.SaveRecipe(context.Background(), kindergartenID, DietRecipeSaveInput{
		Name:    name,
		Items:   items,
		Replace: replace,
	})
	require.NoError(t, err)
	return result
}

// createPlanItem 用 MealPlanRepo 落一条含菜名与营养快照的历史食谱行（TC-NEW-DIE-0025/0027）。
func (f *dietRecipeFixture) createPlanItem(
	t *testing.T,
	kindergartenID uint64,
	recipeName string,
	recipeID uint64,
	foodID uint64,
	foodName string,
	grams float64,
	snapshot string,
) uint64 {
	t.Helper()
	plan := &domain.MealPlan{
		KindergartenID: kindergartenID,
		AcademicYearID: 1,
		Name:           "S4测试历史食谱",
		StartDate:      time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
		DayNum:         1,
		CreatedMethod:  domain.DietPlanCreatedByCustom,
		ConfigSnapshot: "{}",
		Items: []domain.MealPlanItem{{
			Date:              time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC),
			Weekday:           1,
			MealKey:           domain.DietMealLunch,
			RecipeName:        recipeName,
			RecipeID:          &recipeID,
			FoodID:            &foodID,
			FoodName:          foodName,
			Grams:             grams,
			NutritionSnapshot: snapshot,
			Sort:              1,
		}},
	}
	require.NoError(t, f.planRepo.Create(context.Background(), plan))
	return plan.ID
}

func (f *dietRecipeFixture) findPlanItem(t *testing.T, planID uint64) domain.MealPlanItem {
	t.Helper()
	plan, err := f.planRepo.FindByIDWithItems(context.Background(), testDietRecipeSvcKG, planID)
	require.NoError(t, err)
	require.NotNil(t, plan)
	require.Len(t, plan.Items, 1)
	return plan.Items[0]
}

// TC-NEW-DIE-0018：库中不存在的新名称保存为新菜谱，拼音码由服务端生成。
func TestDietRecipe_SaveNewNameCreatesRecipe(t *testing.T) {
	f := newDietRecipeFixture(t)
	ctx := context.Background()

	flour := f.createFood(t, testDietRecipeSvcKG, "S4测试面粉", 344, 11.2)
	pork := f.createFood(t, testDietRecipeSvcKG, "S4测试猪肉", 395, 13.2)

	name := "S4测试鲜菇云吞"
	result := f.saveRecipe(t, testDietRecipeSvcKG, name, false,
		DietRecipeItemInput{FoodID: flour.ID, Grams: 10},
		DietRecipeItemInput{FoodID: pork.ID, Grams: 5},
	)
	assert.True(t, result.Created, "新名称必须走新增分支")
	require.NotZero(t, result.ID)
	assert.Equal(t, name, result.Name)
	assert.Equal(t, dietSpellCode(name), result.SpellCode)
	assert.Equal(t, int64(2), result.ItemCount)

	list, total, page, pageSize, err := f.svc.ListRecipes(ctx, testDietRecipeSvcKG, "云吞", pagination.Query{})
	require.NoError(t, err)
	assert.Equal(t, 1, page)
	assert.Equal(t, 20, pageSize)
	assert.Equal(t, int64(1), total)
	require.Len(t, list, 1)
	assert.Equal(t, int64(2), list[0].ItemCount)

	detail, err := f.svc.GetRecipe(ctx, testDietRecipeSvcKG, result.ID)
	require.NoError(t, err)
	require.Len(t, detail.Items, 2)
	assert.Equal(t, flour.ID, detail.Items[0].FoodID)
	assert.Equal(t, "S4测试面粉", detail.Items[0].FoodName)
	assert.InDelta(t, 10.0, detail.Items[0].Grams, 1e-9)
	assert.Equal(t, 1, detail.Items[0].Sort)
	assert.Equal(t, pork.ID, detail.Items[1].FoodID)
}

// TC-NEW-DIE-0017 / 0020：同名且 replace=false 返回冲突且不写入（仅本次由前端不调接口实现）。
func TestDietRecipe_DuplicateNameWithoutReplaceConflict(t *testing.T) {
	f := newDietRecipeFixture(t)
	ctx := context.Background()

	flour := f.createFood(t, testDietRecipeSvcKG, "S4测试面粉", 344, 11.2)
	pork := f.createFood(t, testDietRecipeSvcKG, "S4测试猪肉", 395, 13.2)
	name := "S4测试鲜菇云吞"
	created := f.saveRecipe(t, testDietRecipeSvcKG, name, false,
		DietRecipeItemInput{FoodID: flour.ID, Grams: 10})

	_, err := f.svc.SaveRecipe(ctx, testDietRecipeSvcKG, DietRecipeSaveInput{
		Name:  name,
		Items: []DietRecipeItemInput{{FoodID: pork.ID, Grams: 99}},
	})
	assert.ErrorIs(t, err, ErrDietRecipeNameConflict)

	_, total, _, _, err := f.svc.ListRecipes(ctx, testDietRecipeSvcKG, "", pagination.Query{})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total, "冲突分支不得创建第二条同名菜谱")

	detail, err := f.svc.GetRecipe(ctx, testDietRecipeSvcKG, created.ID)
	require.NoError(t, err)
	require.Len(t, detail.Items, 1)
	assert.Equal(t, flour.ID, detail.Items[0].FoodID, "冲突分支不得改写原配方")
	assert.InDelta(t, 10.0, detail.Items[0].Grams, 1e-9)
}

// TC-NEW-DIE-0019：replace=true 覆盖原菜谱配方并返回同一个 ID（不再新增）。
func TestDietRecipe_ReplaceOverwritesFormula(t *testing.T) {
	f := newDietRecipeFixture(t)
	ctx := context.Background()

	flour := f.createFood(t, testDietRecipeSvcKG, "S4测试面粉", 344, 11.2)
	pork := f.createFood(t, testDietRecipeSvcKG, "S4测试猪肉", 395, 13.2)
	name := "S4测试鲜菇云吞"
	created := f.saveRecipe(t, testDietRecipeSvcKG, name, false,
		DietRecipeItemInput{FoodID: flour.ID, Grams: 10})

	replaced := f.saveRecipe(t, testDietRecipeSvcKG, name, true,
		DietRecipeItemInput{FoodID: flour.ID, Grams: 12},
		DietRecipeItemInput{FoodID: pork.ID, Grams: 4},
	)
	assert.False(t, replaced.Created)
	assert.Equal(t, created.ID, replaced.ID, "替换必须复用原菜谱")
	assert.Equal(t, int64(2), replaced.ItemCount)

	detail, err := f.svc.GetRecipe(ctx, testDietRecipeSvcKG, created.ID)
	require.NoError(t, err)
	require.Len(t, detail.Items, 2)
	assert.InDelta(t, 12.0, detail.Items[0].Grams, 1e-9)
	assert.InDelta(t, 4.0, detail.Items[1].Grams, 1e-9)

	_, total, _, _, err := f.svc.ListRecipes(ctx, testDietRecipeSvcKG, "", pagination.Query{})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
}

// TC-NEW-DIE-0021：同一道菜内同一种食物的多行合并为一行、克数相加（新增与修改都适用）。
func TestDietRecipe_DuplicateFoodRowsMergeGrams(t *testing.T) {
	f := newDietRecipeFixture(t)
	ctx := context.Background()

	flour := f.createFood(t, testDietRecipeSvcKG, "S4测试面粉", 344, 11.2)
	pork := f.createFood(t, testDietRecipeSvcKG, "S4测试猪肉", 395, 13.2)
	result := f.saveRecipe(t, testDietRecipeSvcKG, "S4测试合并云吞", false,
		DietRecipeItemInput{FoodID: flour.ID, Grams: 5},
		DietRecipeItemInput{FoodID: pork.ID, Grams: 3},
		DietRecipeItemInput{FoodID: flour.ID, Grams: 5},
	)
	assert.Equal(t, int64(2), result.ItemCount)

	detail, err := f.svc.GetRecipe(ctx, testDietRecipeSvcKG, result.ID)
	require.NoError(t, err)
	require.Len(t, detail.Items, 2)
	assert.Equal(t, flour.ID, detail.Items[0].FoodID)
	assert.InDelta(t, 10.0, detail.Items[0].Grams, 1e-9, "核桃(鲜) 5g + 5g = 10g")
	assert.Equal(t, pork.ID, detail.Items[1].FoodID)

	items := []DietRecipeItemInput{
		{FoodID: pork.ID, Grams: 2},
		{FoodID: flour.ID, Grams: 1},
		{FoodID: pork.ID, Grams: 3},
	}
	require.NoError(t, f.svc.UpdateRecipe(ctx, testDietRecipeSvcKG, result.ID,
		DietRecipeUpdateInput{Items: &items}))

	updated, err := f.svc.GetRecipe(ctx, testDietRecipeSvcKG, result.ID)
	require.NoError(t, err)
	require.Len(t, updated.Items, 2)
	assert.Equal(t, pork.ID, updated.Items[0].FoodID)
	assert.InDelta(t, 5.0, updated.Items[0].Grams, 1e-9)
	assert.Equal(t, flour.ID, updated.Items[1].FoodID)
	assert.InDelta(t, 1.0, updated.Items[1].Grams, 1e-9)
}

// TC-NEW-DIE-0022：联想按名称包含或拼音码前缀匹配；limit 默认 18、上限 100。
func TestDietRecipe_SuggestMatchesNameAndSpellPrefix(t *testing.T) {
	f := newDietRecipeFixture(t)
	ctx := context.Background()

	flour := f.createFood(t, testDietRecipeSvcKG, "S4测试面粉", 344, 11.2)
	wonton := f.saveRecipe(t, testDietRecipeSvcKG, "S4测试鲜菇云吞", false,
		DietRecipeItemInput{FoodID: flour.ID, Grams: 10})
	noodle := f.saveRecipe(t, testDietRecipeSvcKG, "S4测试面条", false,
		DietRecipeItemInput{FoodID: flour.ID, Grams: 20})

	byName, err := f.svc.SuggestRecipes(ctx, testDietRecipeSvcKG, "云吞", 0)
	require.NoError(t, err)
	require.Len(t, byName, 1)
	assert.Equal(t, wonton.ID, byName[0].ID)
	assert.Equal(t, int64(1), byName[0].ItemCount)

	prefix := wonton.SpellCode
	if len(prefix) > 5 {
		prefix = prefix[:5]
	}
	bySpell, err := f.svc.SuggestRecipes(ctx, testDietRecipeSvcKG, strings.ToLower(prefix), 0)
	require.NoError(t, err)
	require.Len(t, bySpell, 1, "拼音码前缀匹配必须忽略大小写")
	assert.Equal(t, wonton.ID, bySpell[0].ID)

	// 列表关键字同样支持拼音码前缀（验收 1：keyword 匹配名称/拼音码）。
	listBySpell, listTotal, _, _, err := f.svc.ListRecipes(
		ctx, testDietRecipeSvcKG, strings.ToLower(prefix), pagination.Query{})
	require.NoError(t, err)
	assert.Equal(t, int64(1), listTotal)
	require.Len(t, listBySpell, 1)
	assert.Equal(t, wonton.ID, listBySpell[0].ID)

	all, err := f.svc.SuggestRecipes(ctx, testDietRecipeSvcKG, "", 100)
	require.NoError(t, err)
	assert.Len(t, all, 2)
	assert.ElementsMatch(t, []uint64{wonton.ID, noodle.ID}, []uint64{all[0].ID, all[1].ID})

	limited, err := f.svc.SuggestRecipes(ctx, testDietRecipeSvcKG, "", 1)
	require.NoError(t, err)
	assert.Len(t, limited, 1)

	overCap, err := f.svc.SuggestRecipes(ctx, testDietRecipeSvcKG, "", 1000)
	require.NoError(t, err)
	assert.Len(t, overCap, 2, "limit 超过 100 时按上限处理，不得报错")
}

// TC-NEW-DIE-0023：详情返回完整配料与每种食物当前 27 项营养；缺失项为 null。
// 同时验证 kg=0 平台预置食物不会把营养泄漏进园所菜谱详情（验收 9）。
func TestDietRecipe_DetailReturnsNutrientsAndHidesPlatformFoods(t *testing.T) {
	f := newDietRecipeFixture(t)
	ctx := context.Background()

	food := f.createFood(t, testDietRecipeSvcKG, "S4测试面粉", 344, 11.2)
	result := f.saveRecipe(t, testDietRecipeSvcKG, "S4测试详情云吞", false,
		DietRecipeItemInput{FoodID: food.ID, Grams: 10})

	detail, err := f.svc.GetRecipe(ctx, testDietRecipeSvcKG, result.ID)
	require.NoError(t, err)
	require.Len(t, detail.Items, 1)
	item := detail.Items[0]
	assert.Equal(t, food.ID, item.FoodID)
	assert.Equal(t, "S4测试面粉", item.FoodName)
	require.NotNil(t, item.Nutrients.Energy)
	assert.InDelta(t, 344.0, *item.Nutrients.Energy, 1e-9)
	require.NotNil(t, item.Nutrients.Protein)
	assert.InDelta(t, 11.2, *item.Nutrients.Protein, 1e-9)
	assert.Nil(t, item.Nutrients.VitaminC, "未测营养必须是 null，不能转 0")

	// 直接插入一条引用 kg=0 预置食物的配料（绕过服务校验，模拟历史脏数据）。
	preset := &domain.Food{
		KindergartenID: domain.DietPlatformKindergartenID,
		Name:           "S4测试平台预置面粉",
		CategoryID:     1,
		EdiblePercent:  100,
		Source:         domain.DietFoodSourceSystem,
		Energy:         floatPtr(999),
	}
	require.NoError(t, f.foodRepo.Create(ctx, preset))
	require.NoError(t, f.db.Create(&domain.RecipeItem{
		KindergartenID: testDietRecipeSvcKG,
		RecipeID:       result.ID,
		FoodID:         preset.ID,
		Grams:          5,
		Sort:           2,
	}).Error)

	leaked, err := f.svc.GetRecipe(ctx, testDietRecipeSvcKG, result.ID)
	require.NoError(t, err)
	require.Len(t, leaked.Items, 2)
	platformItem := leaked.Items[1]
	assert.Empty(t, platformItem.FoodName, "kg=0 食物不属于本园，不得回传名称")
	assert.Nil(t, platformItem.Nutrients.Energy, "kg=0 食物的营养不得泄漏进园所详情")
	assert.Nil(t, platformItem.Nutrients.Protein)
}

// TC-NEW-DIE-0024 / 0027：改名与改配方后，联想使用新名称，历史周食谱的菜名与快照不变。
func TestDietRecipe_UpdateRenameAndFormulaKeepsHistory(t *testing.T) {
	f := newDietRecipeFixture(t)
	ctx := context.Background()

	flour := f.createFood(t, testDietRecipeSvcKG, "S4测试面粉", 344, 11.2)
	oldName := "S4测试鲜菇云吞"
	created := f.saveRecipe(t, testDietRecipeSvcKG, oldName, false,
		DietRecipeItemInput{FoodID: flour.ID, Grams: 10})

	snapshot := `{"edible_percent":100,"energy":286,"protein":8.3}`
	planID := f.createPlanItem(t, testDietRecipeSvcKG, oldName, created.ID, flour.ID, "S4测试面粉", 30, snapshot)

	newName := "S4测试鲜菇鲜肉云吞"
	items := []DietRecipeItemInput{{FoodID: flour.ID, Grams: 12}}
	require.NoError(t, f.svc.UpdateRecipe(ctx, testDietRecipeSvcKG, created.ID,
		DietRecipeUpdateInput{Name: &newName, Items: &items}))

	detail, err := f.svc.GetRecipe(ctx, testDietRecipeSvcKG, created.ID)
	require.NoError(t, err)
	assert.Equal(t, newName, detail.Name)
	assert.Equal(t, dietSpellCode(newName), detail.SpellCode)
	require.Len(t, detail.Items, 1)
	assert.InDelta(t, 12.0, detail.Items[0].Grams, 1e-9)

	byNewName, err := f.svc.SuggestRecipes(ctx, testDietRecipeSvcKG, "鲜肉云吞", 0)
	require.NoError(t, err)
	require.Len(t, byNewName, 1)
	assert.Equal(t, created.ID, byNewName[0].ID)

	old, err := f.recipeRepo.FindByName(ctx, testDietRecipeSvcKG, oldName)
	require.NoError(t, err)
	assert.Nil(t, old, "改名后旧名称不得再命中")

	history := f.findPlanItem(t, planID)
	assert.Equal(t, oldName, history.RecipeName, "历史周食谱的菜名不得随菜谱改名")
	assert.InDelta(t, 30.0, history.Grams, 1e-9)
	assert.JSONEq(t, snapshot, history.NutritionSnapshot)
}

// 改名撞上其他菜谱名称返回冲突（契约 409）；改成自己的原名不算冲突。
func TestDietRecipe_UpdateRenameConflict(t *testing.T) {
	f := newDietRecipeFixture(t)
	ctx := context.Background()

	flour := f.createFood(t, testDietRecipeSvcKG, "S4测试面粉", 344, 11.2)
	a := f.saveRecipe(t, testDietRecipeSvcKG, "S4测试云吞A", false,
		DietRecipeItemInput{FoodID: flour.ID, Grams: 10})
	b := f.saveRecipe(t, testDietRecipeSvcKG, "S4测试云吞B", false,
		DietRecipeItemInput{FoodID: flour.ID, Grams: 20})

	conflictName := "S4测试云吞B"
	err := f.svc.UpdateRecipe(ctx, testDietRecipeSvcKG, a.ID,
		DietRecipeUpdateInput{Name: &conflictName})
	assert.ErrorIs(t, err, ErrDietRecipeNameConflict)

	unchanged, err := f.svc.GetRecipe(ctx, testDietRecipeSvcKG, a.ID)
	require.NoError(t, err)
	assert.Equal(t, "S4测试云吞A", unchanged.Name, "冲突时不得改名")

	sameName := "S4测试云吞A"
	require.NoError(t, f.svc.UpdateRecipe(ctx, testDietRecipeSvcKG, a.ID,
		DietRecipeUpdateInput{Name: &sameName}), "改回原名不算撞名")

	bDetail, err := f.svc.GetRecipe(ctx, testDietRecipeSvcKG, b.ID)
	require.NoError(t, err)
	assert.Equal(t, "S4测试云吞B", bDetail.Name)
}

// TC-NEW-DIE-0025：删除后列表不再返回；历史周食谱保持原菜名、克数与营养快照；
// 软删除行不占名称键，同名可以重建。
func TestDietRecipe_DeleteSoftDeletesAndAllowsRecreate(t *testing.T) {
	f := newDietRecipeFixture(t)
	ctx := context.Background()

	flour := f.createFood(t, testDietRecipeSvcKG, "S4测试面粉", 344, 11.2)
	name := "S4测试删除云吞"
	created := f.saveRecipe(t, testDietRecipeSvcKG, name, false,
		DietRecipeItemInput{FoodID: flour.ID, Grams: 10})

	snapshot := `{"edible_percent":100,"energy":344}`
	planID := f.createPlanItem(t, testDietRecipeSvcKG, name, created.ID, flour.ID, "S4测试面粉", 25, snapshot)

	require.NoError(t, f.svc.DeleteRecipe(ctx, testDietRecipeSvcKG, created.ID))

	_, total, _, _, err := f.svc.ListRecipes(ctx, testDietRecipeSvcKG, "", pagination.Query{})
	require.NoError(t, err)
	assert.Zero(t, total, "删除后列表不得返回该条目")
	_, err = f.svc.GetRecipe(ctx, testDietRecipeSvcKG, created.ID)
	assert.ErrorIs(t, err, ErrDietRecipeNotFound)

	history := f.findPlanItem(t, planID)
	assert.Equal(t, name, history.RecipeName)
	assert.InDelta(t, 25.0, history.Grams, 1e-9)
	assert.JSONEq(t, snapshot, history.NutritionSnapshot)

	recreated := f.saveRecipe(t, testDietRecipeSvcKG, name, false,
		DietRecipeItemInput{FoodID: flour.ID, Grams: 8})
	assert.True(t, recreated.Created)
	assert.NotEqual(t, created.ID, recreated.ID, "软删除后同名必须能重建为新行")
}

// TC-NEW-DIE-0026：合并把源菜谱同种食物克数累加到目标，随后删除源菜谱。
func TestDietRecipe_MergeSumsGramsAndDeletesSource(t *testing.T) {
	f := newDietRecipeFixture(t)
	ctx := context.Background()

	a := f.createFood(t, testDietRecipeSvcKG, "S4测试面粉", 344, 11.2)
	b := f.createFood(t, testDietRecipeSvcKG, "S4测试猪肉", 395, 13.2)
	c := f.createFood(t, testDietRecipeSvcKG, "S4测试核桃", 646, 14.9)

	target := f.saveRecipe(t, testDietRecipeSvcKG, "S4测试合并目标", false,
		DietRecipeItemInput{FoodID: a.ID, Grams: 10},
		DietRecipeItemInput{FoodID: b.ID, Grams: 5},
	)
	source := f.saveRecipe(t, testDietRecipeSvcKG, "S4测试合并源", false,
		DietRecipeItemInput{FoodID: a.ID, Grams: 7},
		DietRecipeItemInput{FoodID: c.ID, Grams: 3},
	)

	result, err := f.svc.MergeRecipes(ctx, testDietRecipeSvcKG, source.ID, target.ID)
	require.NoError(t, err)
	assert.Equal(t, target.ID, result.TargetID)
	assert.Equal(t, int64(3), result.MergedItemCount)

	detail, err := f.svc.GetRecipe(ctx, testDietRecipeSvcKG, target.ID)
	require.NoError(t, err)
	require.Len(t, detail.Items, 3)
	assert.Equal(t, a.ID, detail.Items[0].FoodID)
	assert.InDelta(t, 17.0, detail.Items[0].Grams, 1e-9, "10g + 7g = 17g")
	assert.Equal(t, b.ID, detail.Items[1].FoodID)
	assert.InDelta(t, 5.0, detail.Items[1].Grams, 1e-9)
	assert.Equal(t, c.ID, detail.Items[2].FoodID)
	assert.InDelta(t, 3.0, detail.Items[2].Grams, 1e-9)

	_, err = f.svc.GetRecipe(ctx, testDietRecipeSvcKG, source.ID)
	assert.ErrorIs(t, err, ErrDietRecipeNotFound, "合并后源菜谱必须删除")

	_, total, _, _, err := f.svc.ListRecipes(ctx, testDietRecipeSvcKG, "", pagination.Query{})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)

	var sourceItems int64
	require.NoError(t, f.db.Unscoped().Model(&domain.RecipeItem{}).
		Where("recipe_id = ? AND kindergarten_id = ?", source.ID, testDietRecipeSvcKG).
		Count(&sourceItems).Error)
	assert.Zero(t, sourceItems, "源配料必须硬删，不留软删行")
}

// 合并参数边界：source==target → 422；跨园/不存在 → 404。
func TestDietRecipe_MergeRejectsSameAndCrossGarden(t *testing.T) {
	f := newDietRecipeFixture(t)
	ctx := context.Background()

	flour := f.createFood(t, testDietRecipeSvcKG, "S4测试面粉", 344, 11.2)
	recipe := f.saveRecipe(t, testDietRecipeSvcKG, "S4测试合并边界", false,
		DietRecipeItemInput{FoodID: flour.ID, Grams: 10})
	otherFood := f.createFood(t, testDietRecipeSvcOtherKG, "S4测试他园面粉", 300, 10)
	other := f.saveRecipe(t, testDietRecipeSvcOtherKG, "S4测试他园菜谱", false,
		DietRecipeItemInput{FoodID: otherFood.ID, Grams: 10})

	_, err := f.svc.MergeRecipes(ctx, testDietRecipeSvcKG, recipe.ID, recipe.ID)
	assert.ErrorAs(t, err, new(*DietRecipeValidationError))

	_, err = f.svc.MergeRecipes(ctx, testDietRecipeSvcKG, other.ID, recipe.ID)
	assert.ErrorIs(t, err, ErrDietRecipeNotFound, "跨园源菜谱按 404 处理")

	_, err = f.svc.MergeRecipes(ctx, testDietRecipeSvcKG, recipe.ID, other.ID)
	assert.ErrorIs(t, err, ErrDietRecipeNotFound, "跨园目标菜谱按 404 处理")

	_, err = f.svc.MergeRecipes(ctx, testDietRecipeSvcKG, 999999999, recipe.ID)
	assert.ErrorIs(t, err, ErrDietRecipeNotFound)
}

// 入参校验：名称、配料、克数、食物归属都必须合法。
func TestDietRecipe_ValidationRejectsInvalidInput(t *testing.T) {
	f := newDietRecipeFixture(t)
	ctx := context.Background()

	flour := f.createFood(t, testDietRecipeSvcKG, "S4测试面粉", 344, 11.2)
	otherFood := f.createFood(t, testDietRecipeSvcOtherKG, "S4测试他园面粉", 300, 10)
	valid := []DietRecipeItemInput{{FoodID: flour.ID, Grams: 10}}

	cases := []struct {
		name string
		in   DietRecipeSaveInput
	}{
		{"缺少名称", DietRecipeSaveInput{Items: valid}},
		{"名称空白", DietRecipeSaveInput{Name: "   ", Items: valid}},
		{"配料为空", DietRecipeSaveInput{Name: "S4测试空配料"}},
		{"克数为 0", DietRecipeSaveInput{Name: "S4测试零克数",
			Items: []DietRecipeItemInput{{FoodID: flour.ID, Grams: 0}}}},
		{"克数为负", DietRecipeSaveInput{Name: "S4测试负克数",
			Items: []DietRecipeItemInput{{FoodID: flour.ID, Grams: -1}}}},
		{"食物 ID 为 0", DietRecipeSaveInput{Name: "S4测试零食物",
			Items: []DietRecipeItemInput{{FoodID: 0, Grams: 10}}}},
		{"食物不存在", DietRecipeSaveInput{Name: "S4测试无食物",
			Items: []DietRecipeItemInput{{FoodID: 999999999, Grams: 10}}}},
		{"跨园食物", DietRecipeSaveInput{Name: "S4测试跨园食物",
			Items: []DietRecipeItemInput{{FoodID: otherFood.ID, Grams: 10}}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := f.svc.SaveRecipe(ctx, testDietRecipeSvcKG, tc.in)
			assert.ErrorAs(t, err, new(*DietRecipeValidationError))
		})
	}

	recipe := f.saveRecipe(t, testDietRecipeSvcKG, "S4测试校验菜谱", false, valid...)
	assert.ErrorAs(t,
		f.svc.UpdateRecipe(ctx, testDietRecipeSvcKG, recipe.ID, DietRecipeUpdateInput{}),
		new(*DietRecipeValidationError), "空修改返回 422")
	assert.ErrorIs(t,
		f.svc.UpdateRecipe(ctx, testDietRecipeSvcKG, 999999999, DietRecipeUpdateInput{Name: stringPtr("S4不存在")}),
		ErrDietRecipeNotFound)
	assert.ErrorIs(t,
		f.svc.DeleteRecipe(ctx, testDietRecipeSvcKG, 999999999), ErrDietRecipeNotFound)
}

// 名称唯一按园隔离：他园同名菜谱互不可见，详情/删除按 404 处理。
func TestDietRecipe_NameUniquePerKindergarten(t *testing.T) {
	f := newDietRecipeFixture(t)
	ctx := context.Background()

	flour := f.createFood(t, testDietRecipeSvcKG, "S4测试面粉", 344, 11.2)
	otherFood := f.createFood(t, testDietRecipeSvcOtherKG, "S4测试他园面粉", 300, 10)
	name := "S4测试同名云吞"

	mine := f.saveRecipe(t, testDietRecipeSvcKG, name, false,
		DietRecipeItemInput{FoodID: flour.ID, Grams: 10})
	theirs := f.saveRecipe(t, testDietRecipeSvcOtherKG, name, false,
		DietRecipeItemInput{FoodID: otherFood.ID, Grams: 20})
	assert.NotEqual(t, mine.ID, theirs.ID)

	myList, myTotal, _, _, err := f.svc.ListRecipes(ctx, testDietRecipeSvcKG, name, pagination.Query{})
	require.NoError(t, err)
	assert.Equal(t, int64(1), myTotal)
	require.Len(t, myList, 1)
	assert.Equal(t, mine.ID, myList[0].ID)

	_, err = f.svc.GetRecipe(ctx, testDietRecipeSvcKG, theirs.ID)
	assert.ErrorIs(t, err, ErrDietRecipeNotFound, "跨园菜谱按 404 处理")
	assert.ErrorIs(t, f.svc.DeleteRecipe(ctx, testDietRecipeSvcKG, theirs.ID), ErrDietRecipeNotFound)

	theirDetail, err := f.svc.GetRecipe(ctx, testDietRecipeSvcOtherKG, theirs.ID)
	require.NoError(t, err)
	require.Len(t, theirDetail.Items, 1)
	assert.Equal(t, otherFood.ID, theirDetail.Items[0].FoodID)
}
