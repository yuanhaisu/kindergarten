package service

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
	"kindergarten-server/pkg/pagination"
)

// 食物库 service 测试打真实 MySQL（与 repository/其他 service 测试同源）。
// 清理一律原始 DELETE 硬删：软删除行会占唯一索引，重跑撞键（CLAUDE.md 已知坑）。
const (
	testDietFoodSvcKG      uint64 = 990985
	testDietFoodSvcOtherKG uint64 = 990986
	testDietFoodSvcPreset         = "s3-test-preset"
)

func openDietFoodServiceTestDB(t *testing.T) *gorm.DB {
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
	), "auto migrate diet food tables")
	return db
}

func cleanupDietFoodServiceData(t *testing.T, db *gorm.DB) {
	t.Helper()
	stmts := []struct {
		sql  string
		args []any
	}{
		{"DELETE FROM diet_recipe_items WHERE kindergarten_id IN ?", []any{[]uint64{testDietFoodSvcKG, testDietFoodSvcOtherKG}}},
		{"DELETE FROM diet_recipes WHERE kindergarten_id IN ?", []any{[]uint64{testDietFoodSvcKG, testDietFoodSvcOtherKG}}},
		{"DELETE FROM diet_meal_plan_items WHERE kindergarten_id IN ?", []any{[]uint64{testDietFoodSvcKG, testDietFoodSvcOtherKG}}},
		{"DELETE FROM diet_meal_plans WHERE kindergarten_id IN ?", []any{[]uint64{testDietFoodSvcKG, testDietFoodSvcOtherKG}}},
		{"DELETE FROM diet_foods WHERE kindergarten_id IN ?", []any{[]uint64{testDietFoodSvcKG, testDietFoodSvcOtherKG}}},
		{"DELETE FROM diet_foods WHERE kindergarten_id = 0 AND preset_key = ?", []any{testDietFoodSvcPreset}},
		{"DELETE FROM diet_food_categories WHERE name LIKE ?", []any{"S3测试%"}},
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt.sql, stmt.args...).Error; err != nil {
			t.Logf("cleanup %q: %v", stmt.sql, err)
		}
	}
}

type dietFoodFixture struct {
	db         *gorm.DB
	svc        *DietFoodService
	foodRepo   *repository.DietFoodRepo
	recipeRepo *repository.DietRecipeRepo

	root      *domain.FoodCategory
	leaf      *domain.FoodCategory
	leafOther *domain.FoodCategory
	otherRoot *domain.FoodCategory
	leafMeat  *domain.FoodCategory
}

func newDietFoodFixture(t *testing.T) *dietFoodFixture {
	t.Helper()
	db := openDietFoodServiceTestDB(t)
	cleanupDietFoodServiceData(t, db)
	t.Cleanup(func() { cleanupDietFoodServiceData(t, db) })

	f := &dietFoodFixture{
		db:         db,
		foodRepo:   repository.NewDietFoodRepo(db),
		recipeRepo: repository.NewDietRecipeRepo(db),
	}
	f.svc = NewDietFoodService(f.foodRepo, f.recipeRepo)

	f.root = f.createCategory(t, 0, 1, "S3测试谷类", 1)
	f.leaf = f.createCategory(t, f.root.ID, 2, "S3测试小麦", 1)
	f.leafOther = f.createCategory(t, f.root.ID, 2, "S3测试稻米", 2)
	f.otherRoot = f.createCategory(t, 0, 1, "S3测试肉类", 2)
	f.leafMeat = f.createCategory(t, f.otherRoot.ID, 2, "S3测试猪", 1)
	return f
}

func (f *dietFoodFixture) createCategory(
	t *testing.T,
	parentID uint64,
	level int,
	name string,
	sort int,
) *domain.FoodCategory {
	t.Helper()
	row := &domain.FoodCategory{ParentID: parentID, Level: level, Name: name, Sort: sort}
	require.NoError(t, f.db.Create(row).Error)
	return row
}

func floatPtr(v float64) *float64 { return &v }
func uint64Ptr(v uint64) *uint64  { return &v }
func stringPtr(v string) *string  { return &v }

// createCustomFood 走 service 新建园所自建食物，返回落库后的行。
func (f *dietFoodFixture) createCustomFood(
	t *testing.T,
	kindergartenID uint64,
	name string,
	categoryID uint64,
	nutrients *DietFoodNutrientsInput,
) *domain.Food {
	t.Helper()
	result, err := f.svc.CreateFood(context.Background(), kindergartenID, DietFoodInput{
		Name:          stringPtr(name),
		CategoryID:    uint64Ptr(categoryID),
		EdiblePercent: floatPtr(100),
		Nutrients:     nutrients,
	})
	require.NoError(t, err)
	food, err := f.foodRepo.FindByID(context.Background(), kindergartenID, result.ID)
	require.NoError(t, err)
	require.NotNil(t, food)
	return food
}

// 契约 §三 POST：group12/group16/source_flags 缺省时按所选一级分类推导。
func TestDietFood_GroupDefaultsByRootCategory(t *testing.T) {
	food := &domain.Food{}
	require.NoError(t, applyDietFoodGroups(food, "谷类及制品", DietFoodInput{}))
	assert.Equal(t, domain.DietGroup16Grain, food.Group16)
	require.NotNil(t, food.Group12)
	assert.Equal(t, domain.DietGroup12Grain, *food.Group12)
	assert.Equal(t, domain.DietSourceFlagPlant, food.SourceFlags)

	group12 := domain.DietGroup12Nut
	group16 := domain.DietGroup16Nut
	explicit := &domain.Food{}
	require.NoError(t, applyDietFoodGroups(explicit, "谷类及制品", DietFoodInput{
		Group12:     &group12,
		Group16:     &group16,
		SourceFlags: []string{domain.DietSourceFlagAnimal},
	}))
	assert.Equal(t, domain.DietGroup16Nut, explicit.Group16)
	assert.Equal(t, domain.DietGroup12Nut, *explicit.Group12)
	assert.Equal(t, domain.DietSourceFlagAnimal, explicit.SourceFlags, "显式值优先于推导")

	noSlot := &domain.Food{}
	require.NoError(t, applyDietFoodGroups(noSlot, "薯类、淀粉及制品", DietFoodInput{}))
	assert.Nil(t, noSlot.Group12, "薯类没有 group12 槽位")
	assert.Equal(t, domain.DietGroup16Tuber, noSlot.Group16)
}

// 分类树：一级 20 类为根，二级只挂在 children 下（契约 §二）。
func TestDietFood_CategoryTreeTwoLevels(t *testing.T) {
	f := newDietFoodFixture(t)
	list, err := f.svc.ListCategories(context.Background())
	require.NoError(t, err)

	byID := make(map[uint64]DietFoodCategoryView, len(list))
	for _, node := range list {
		byID[node.ID] = node
	}
	root, ok := byID[f.root.ID]
	require.True(t, ok, "一级分类必须出现在根列表")
	assert.Equal(t, 1, root.Level)
	childIDs := make([]uint64, 0, len(root.Children))
	for _, child := range root.Children {
		childIDs = append(childIDs, child.ID)
		assert.Equal(t, f.root.ID, child.ParentID)
		assert.Equal(t, 2, child.Level)
	}
	assert.ElementsMatch(t, []uint64{f.leaf.ID, f.leafOther.ID}, childIDs)
	_, leafIsRoot := byID[f.leaf.ID]
	assert.False(t, leafIsRoot, "二级分类不得作为根节点返回")
}

// TC-NEW-DIE-0011：缺失营养存 NULL 读 NULL，修改时整体替换（未提交键写 NULL）。
func TestDietFood_DetailPreservesNullNutrients(t *testing.T) {
	f := newDietFoodFixture(t)
	ctx := context.Background()

	protein := 8.3
	food := f.createCustomFood(t, testDietFoodSvcKG, "S3测试面粉", f.leaf.ID,
		&DietFoodNutrientsInput{Protein: &protein})

	detail, err := f.svc.GetFood(ctx, testDietFoodSvcKG, food.ID)
	require.NoError(t, err)
	require.NotNil(t, detail.Nutrients.Protein)
	assert.InDelta(t, 8.3, *detail.Nutrients.Protein, 1e-9)
	assert.Nil(t, detail.Nutrients.Energy, "未测项必须是 null，不能转 0")
	assert.Nil(t, detail.Nutrients.VitaminC)
	assert.Nil(t, detail.Nutrients.Selenium)

	// 修改：nutrients 提交即整体替换，旧 protein 必须清成 NULL。
	energy := 286.0
	require.NoError(t, f.svc.UpdateFood(ctx, testDietFoodSvcKG, food.ID, DietFoodInput{
		Nutrients: &DietFoodNutrientsInput{Energy: &energy},
	}))
	updated, err := f.svc.GetFood(ctx, testDietFoodSvcKG, food.ID)
	require.NoError(t, err)
	assert.Nil(t, updated.Nutrients.Protein, "整体替换后旧值必须写回 NULL")
	require.NotNil(t, updated.Nutrients.Energy)
	require.NotNil(t, updated.Nutrients.Heat, "heat 缺省且 energy 有值时必须派生")
	assert.InDelta(t, 1197.2, *updated.Nutrients.Heat, 0.01)

	// 营养可空：不传 nutrients 的新建食物 27 项全为 null。
	empty := f.createCustomFood(t, testDietFoodSvcKG, "S3测试空营养", f.leaf.ID, nil)
	emptyDetail, err := f.svc.GetFood(ctx, testDietFoodSvcKG, empty.ID)
	require.NoError(t, err)
	assert.Nil(t, emptyDetail.Nutrients.Energy)
	assert.Nil(t, emptyDetail.Nutrients.Protein)
}

// TC-NEW-DIE-0013：一级分类包含其全部二级，再叠加名称片段过滤。
func TestDietFood_ListFiltersByCategoryAndKeyword(t *testing.T) {
	f := newDietFoodFixture(t)
	ctx := context.Background()

	flour := f.createCustomFood(t, testDietFoodSvcKG, "S3测试面粉", f.leaf.ID, nil)
	rice := f.createCustomFood(t, testDietFoodSvcKG, "S3测试米粉", f.leafOther.ID, nil)
	pork := f.createCustomFood(t, testDietFoodSvcKG, "S3测试猪肉", f.leafMeat.ID, nil)

	cases := []struct {
		name       string
		categoryID uint64
		keyword    string
		wantIDs    []uint64
	}{
		{"一级分类含全部二级", f.root.ID, "", []uint64{flour.ID, rice.ID}},
		{"二级分类精确", f.leafOther.ID, "", []uint64{rice.ID}},
		{"一级分类+名称片段", f.root.ID, "面粉", []uint64{flour.ID}},
		{"关键字无命中", f.root.ID, "不存在", nil},
		{"跨一级分类不串", f.otherRoot.ID, "", []uint64{pork.ID}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rows, total, page, pageSize, err := f.svc.ListFoods(
				ctx, testDietFoodSvcKG, tc.categoryID, tc.keyword, pagination.Query{})
			require.NoError(t, err)
			assert.Equal(t, 1, page)
			assert.Equal(t, 20, pageSize)
			assert.Equal(t, int64(len(tc.wantIDs)), total)
			got := make([]uint64, 0, len(rows))
			for _, row := range rows {
				got = append(got, row.ID)
				assert.NotEmpty(t, row.CategoryName)
			}
			assert.ElementsMatch(t, tc.wantIDs, got)
		})
	}
}

// TC-NEW-DIE-0010：园所 A 修改副本不影响园所 B，也不影响 kg=0 预置源。
func TestDietFood_UpdateCopyIsolation(t *testing.T) {
	f := newDietFoodFixture(t)
	ctx := context.Background()

	protein := 10.0
	preset := &domain.Food{
		KindergartenID: domain.DietPlatformKindergartenID,
		Name:           "S3测试预置谷物",
		CategoryID:     f.leaf.ID,
		EdiblePercent:  100,
		Source:         domain.DietFoodSourceSystem,
		PresetKey:      testDietFoodSvcPreset,
		Protein:        &protein,
	}
	require.NoError(t, f.foodRepo.Create(ctx, preset))

	copyA := &domain.Food{
		KindergartenID: testDietFoodSvcKG,
		Name:           preset.Name,
		CategoryID:     f.leaf.ID,
		EdiblePercent:  100,
		Source:         domain.DietFoodSourceSystem,
		PresetKey:      testDietFoodSvcPreset,
		Protein:        &protein,
	}
	require.NoError(t, f.foodRepo.Create(ctx, copyA))
	copyB := &domain.Food{
		KindergartenID: testDietFoodSvcOtherKG,
		Name:           preset.Name,
		CategoryID:     f.leaf.ID,
		EdiblePercent:  100,
		Source:         domain.DietFoodSourceSystem,
		PresetKey:      testDietFoodSvcPreset,
		Protein:        &protein,
	}
	require.NoError(t, f.foodRepo.Create(ctx, copyB))

	require.NoError(t, f.svc.UpdateFood(ctx, testDietFoodSvcKG, copyA.ID, DietFoodInput{
		EdiblePercent: floatPtr(80),
	}))

	gotA, err := f.foodRepo.FindByID(ctx, testDietFoodSvcKG, copyA.ID)
	require.NoError(t, err)
	require.NotNil(t, gotA)
	assert.Equal(t, 80.0, gotA.EdiblePercent)

	gotB, err := f.foodRepo.FindByID(ctx, testDietFoodSvcOtherKG, copyB.ID)
	require.NoError(t, err)
	require.NotNil(t, gotB)
	assert.Equal(t, 100.0, gotB.EdiblePercent, "他园副本必须保持原值")

	gotPreset, err := f.foodRepo.FindByID(ctx, domain.DietPlatformKindergartenID, preset.ID)
	require.NoError(t, err)
	require.NotNil(t, gotPreset)
	assert.Equal(t, 100.0, gotPreset.EdiblePercent, "kg=0 预置源不得被园所修改")

	// 跨园详情按不存在处理（契约 404）；kg=0 也不能被园所读取或修改。
	_, err = f.svc.GetFood(ctx, testDietFoodSvcOtherKG, copyA.ID)
	assert.ErrorIs(t, err, ErrDietFoodNotFound)
	_, err = f.svc.GetFood(ctx, testDietFoodSvcKG, preset.ID)
	assert.ErrorIs(t, err, ErrDietFoodNotFound)
	assert.ErrorIs(t,
		f.svc.UpdateFood(ctx, testDietFoodSvcKG, preset.ID, DietFoodInput{EdiblePercent: floatPtr(1)}),
		ErrDietFoodNotFound)
}

// TC-NEW-DIE-0016（service 侧）：修改预置副本置 is_customized，供 S5 升级跳过。
func TestDietFood_UpdatePresetCopyMarksCustomized(t *testing.T) {
	f := newDietFoodFixture(t)
	ctx := context.Background()

	systemFood := &domain.Food{
		KindergartenID: testDietFoodSvcKG,
		Name:           "S3测试系统食物",
		CategoryID:     f.leaf.ID,
		EdiblePercent:  100,
		Source:         domain.DietFoodSourceSystem,
		PresetKey:      testDietFoodSvcPreset,
	}
	require.NoError(t, f.foodRepo.Create(ctx, systemFood))
	untouched := &domain.Food{
		KindergartenID: testDietFoodSvcKG,
		Name:           "S3测试系统食物二",
		CategoryID:     f.leaf.ID,
		EdiblePercent:  100,
		Source:         domain.DietFoodSourceSystem,
		PresetKey:      testDietFoodSvcPreset + "-2",
	}
	require.NoError(t, f.foodRepo.Create(ctx, untouched))

	require.NoError(t, f.svc.UpdateFood(ctx, testDietFoodSvcKG, systemFood.ID, DietFoodInput{
		EdiblePercent: floatPtr(95),
	}))
	got, err := f.foodRepo.FindByID(ctx, testDietFoodSvcKG, systemFood.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.True(t, got.IsCustomized, "改过的预置副本必须标记 is_customized")

	still, err := f.foodRepo.FindByID(ctx, testDietFoodSvcKG, untouched.ID)
	require.NoError(t, err)
	require.NotNil(t, still)
	assert.False(t, still.IsCustomized, "未修改的预置副本保持 false")

	// 园所自建食物不需要该标记。
	custom := f.createCustomFood(t, testDietFoodSvcKG, "S3测试自建", f.leaf.ID, nil)
	require.NoError(t, f.svc.UpdateFood(ctx, testDietFoodSvcKG, custom.ID, DietFoodInput{
		Name: stringPtr("S3测试自建改名"),
	}))
	gotCustom, err := f.foodRepo.FindByID(ctx, testDietFoodSvcKG, custom.ID)
	require.NoError(t, err)
	require.NotNil(t, gotCustom)
	assert.False(t, gotCustom.IsCustomized)
}

// TC-NEW-DIE-0014：被菜谱引用时拒绝删除；解除引用后可删。
func TestDietFood_DeleteReferencedByRecipeRejected(t *testing.T) {
	f := newDietFoodFixture(t)
	ctx := context.Background()

	food := f.createCustomFood(t, testDietFoodSvcKG, "S3测试鸡蛋", f.leaf.ID, nil)
	recipe := &domain.Recipe{
		KindergartenID: testDietFoodSvcKG,
		Name:           "S3测试番茄炒蛋",
		SpellCode:      "S3CS",
		Items:          []domain.RecipeItem{{FoodID: food.ID, Grams: 50}},
	}
	require.NoError(t, f.recipeRepo.Create(ctx, recipe))

	err := f.svc.DeleteFood(ctx, testDietFoodSvcKG, food.ID)
	assert.ErrorIs(t, err, ErrDietFoodReferencedByRecipe)

	still, err := f.foodRepo.FindByID(ctx, testDietFoodSvcKG, food.ID)
	require.NoError(t, err)
	assert.NotNil(t, still, "被引用时食物必须仍然存在")

	_, err = f.recipeRepo.Delete(ctx, testDietFoodSvcKG, recipe.ID)
	require.NoError(t, err)
	require.NoError(t, f.svc.DeleteFood(ctx, testDietFoodSvcKG, food.ID), "解除菜谱引用后应可删除")
	deleted, err := f.foodRepo.FindByID(ctx, testDietFoodSvcKG, food.ID)
	require.NoError(t, err)
	assert.Nil(t, deleted)
}

// TC-NEW-DIE-0015：只被历史周食谱引用时允许删除，历史明细的名称与快照不变。
func TestDietFood_DeleteReferencedOnlyByMealPlanAllowed(t *testing.T) {
	f := newDietFoodFixture(t)
	ctx := context.Background()

	food := f.createCustomFood(t, testDietFoodSvcKG, "S3测试历史食物", f.leaf.ID, nil)
	// diet_meal_plan_items.plan_id 有真实外键，先建一条周食谱。
	plan := &domain.MealPlan{
		KindergartenID: testDietFoodSvcKG,
		AcademicYearID: 1,
		Name:           "S3测试历史食谱",
		StartDate:      time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
		DayNum:         1,
		CreatedMethod:  domain.DietPlanCreatedByCustom,
		ConfigSnapshot: "{}",
	}
	require.NoError(t, f.db.Create(plan).Error)

	snapshot := `{"edible_percent":100,"energy":286}`
	item := &domain.MealPlanItem{
		KindergartenID:    testDietFoodSvcKG,
		PlanID:            plan.ID,
		Date:              time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC),
		Weekday:           1,
		MealKey:           domain.DietMealLunch,
		RecipeName:        "S3测试历史菜",
		FoodID:            &food.ID,
		FoodName:          food.Name,
		Grams:             30,
		NutritionSnapshot: snapshot,
		Sort:              1,
	}
	require.NoError(t, f.db.Create(item).Error)

	require.NoError(t, f.svc.DeleteFood(ctx, testDietFoodSvcKG, food.ID), "只被周食谱引用时允许删除")

	var reloaded domain.MealPlanItem
	require.NoError(t, f.db.Where("id = ?", item.ID).First(&reloaded).Error)
	assert.Equal(t, "S3测试历史食物", reloaded.FoodName, "历史明细保留名称快照")
	assert.Equal(t, 30.0, reloaded.Grams)
	assert.JSONEq(t, snapshot, reloaded.NutritionSnapshot, "营养快照必须原样保留")
}

func TestDietFood_BatchDeleteSkipsReferencedAndMissing(t *testing.T) {
	f := newDietFoodFixture(t)
	ctx := context.Background()

	a := f.createCustomFood(t, testDietFoodSvcKG, "S3测试批量一", f.leaf.ID, nil)
	b := f.createCustomFood(t, testDietFoodSvcKG, "S3测试批量二", f.leaf.ID, nil)
	c := f.createCustomFood(t, testDietFoodSvcKG, "S3测试批量三", f.leaf.ID, nil)

	recipe := &domain.Recipe{
		KindergartenID: testDietFoodSvcKG,
		Name:           "S3测试批量菜谱",
		Items:          []domain.RecipeItem{{FoodID: b.ID, Grams: 10}},
	}
	require.NoError(t, f.recipeRepo.Create(ctx, recipe))

	const missingID uint64 = 999999998
	result, err := f.svc.BatchDeleteFoods(ctx, testDietFoodSvcKG,
		[]uint64{a.ID, b.ID, missingID, c.ID, a.ID})
	require.NoError(t, err)
	assert.Equal(t, 2, result.Deleted, "重复 ID 只处理一次；a/c 可删")
	require.Len(t, result.Skipped, 2)
	assert.Equal(t, DietFoodBatchDeleteSkipped{ID: b.ID, Name: "S3测试批量二", Reason: DietFoodSkipReferencedByRecipe},
		result.Skipped[0])
	assert.Equal(t, DietFoodBatchDeleteSkipped{ID: missingID, Name: "", Reason: DietFoodSkipNotFound},
		result.Skipped[1])

	for _, id := range []uint64{a.ID, c.ID} {
		row, err := f.foodRepo.FindByID(ctx, testDietFoodSvcKG, id)
		require.NoError(t, err)
		assert.Nil(t, row)
	}
	kept, err := f.foodRepo.FindByID(ctx, testDietFoodSvcKG, b.ID)
	require.NoError(t, err)
	assert.NotNil(t, kept)

	_, err = f.svc.BatchDeleteFoods(ctx, testDietFoodSvcKG, nil)
	assert.ErrorAs(t, err, new(*DietFoodValidationError), "空 ID 列表返回 422")
}

// 契约 §三：替换候选是「同一一级分类」而不是同叶子分类（猪瘦肉 → 牛腩）。
func TestDietFood_ReplaceCandidatesUseRootCategory(t *testing.T) {
	f := newDietFoodFixture(t)
	ctx := context.Background()

	seed := f.createCustomFood(t, testDietFoodSvcKG, "S3测试全麦粉", f.leaf.ID, nil)
	sibling := f.createCustomFood(t, testDietFoodSvcKG, "S3测试糙米", f.leafOther.ID, nil)
	f.createCustomFood(t, testDietFoodSvcKG, "S3测试牛肉", f.leafMeat.ID, nil)
	f.createCustomFood(t, testDietFoodSvcOtherKG, "S3测试他园米", f.leafOther.ID, nil)
	preset := &domain.Food{
		KindergartenID: domain.DietPlatformKindergartenID,
		Name:           "S3测试预置米",
		CategoryID:     f.leafOther.ID,
		EdiblePercent:  100,
		Source:         domain.DietFoodSourceSystem,
		PresetKey:      testDietFoodSvcPreset,
	}
	require.NoError(t, f.foodRepo.Create(ctx, preset))

	list, err := f.svc.ListReplaceCandidates(ctx, testDietFoodSvcKG, seed.ID, "", 20)
	require.NoError(t, err)
	got := make([]uint64, 0, len(list))
	for _, item := range list {
		got = append(got, item.ID)
	}
	assert.ElementsMatch(t, []uint64{sibling.ID}, got,
		"只返回同一一级分类下的本园食物，排除自身/跨园/kg=0/其他一级分类")

	filtered, err := f.svc.ListReplaceCandidates(ctx, testDietFoodSvcKG, seed.ID, "糙米", 20)
	require.NoError(t, err)
	require.Len(t, filtered, 1)
	assert.Equal(t, sibling.ID, filtered[0].ID)
	assert.Equal(t, f.leafOther.Name, filtered[0].CategoryName)

	limited, err := f.svc.ListReplaceCandidates(ctx, testDietFoodSvcKG, seed.ID, "", 1)
	require.NoError(t, err)
	assert.Len(t, limited, 1)

}

// 入参校验：名称/叶子分类/食部必填，nutrients 里禁止 edible_percent。
func TestDietFood_ValidationRejectsInvalidInput(t *testing.T) {
	f := newDietFoodFixture(t)
	ctx := context.Background()

	cases := []struct {
		name string
		in   DietFoodInput
	}{
		{"缺少名称", DietFoodInput{
			CategoryID: uint64Ptr(f.leaf.ID), EdiblePercent: floatPtr(100),
		}},
		{"名称空白", DietFoodInput{
			Name: stringPtr("   "), CategoryID: uint64Ptr(f.leaf.ID), EdiblePercent: floatPtr(100),
		}},
		{"一级分类不可选", DietFoodInput{
			Name: stringPtr("S3测试错误分类"), CategoryID: uint64Ptr(f.root.ID), EdiblePercent: floatPtr(100),
		}},
		{"分类不存在", DietFoodInput{
			Name: stringPtr("S3测试错误分类二"), CategoryID: uint64Ptr(999999997), EdiblePercent: floatPtr(100),
		}},
		{"缺少食部", DietFoodInput{
			Name: stringPtr("S3测试无食部"), CategoryID: uint64Ptr(f.leaf.ID),
		}},
		{"食部超范围", DietFoodInput{
			Name: stringPtr("S3测试食部超限"), CategoryID: uint64Ptr(f.leaf.ID), EdiblePercent: floatPtr(101),
		}},
		{"nutrients 混入食部", DietFoodInput{
			Name: stringPtr("S3测试混入食部"), CategoryID: uint64Ptr(f.leaf.ID), EdiblePercent: floatPtr(100),
			Nutrients: &DietFoodNutrientsInput{EdiblePercent: floatPtr(90)},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := f.svc.CreateFood(ctx, testDietFoodSvcKG, tc.in)
			assert.ErrorAs(t, err, new(*DietFoodValidationError))
		})
	}

	food := f.createCustomFood(t, testDietFoodSvcKG, "S3测试空修改", f.leaf.ID, nil)
	assert.ErrorAs(t,
		f.svc.UpdateFood(ctx, testDietFoodSvcKG, food.ID, DietFoodInput{}),
		new(*DietFoodValidationError), "空修改返回 422")
}

// 契约要求 spell_code 由服务端按名称生成，不接受客户端输入。
func TestDietFood_SpellCodeFromPinyinInitials(t *testing.T) {
	assert.Equal(t, "MT", dietSpellCode("面条"), "契约示例：面条 → MT")
	assert.Equal(t, "ZZNGB", dietSpellCode("自制南瓜饼"), "契约示例：自制南瓜饼 → ZZNGB")
	assert.Equal(t, "ABC123", dietSpellCode("abc 123"))
	assert.Equal(t, "", dietSpellCode(""))
}

// 列表/候选/详情都不得泄漏 kg=0 预置行；关键字可走拼音码前缀。
func TestDietFood_ListExcludesPresetRowsAndMatchesSpellCode(t *testing.T) {
	f := newDietFoodFixture(t)
	ctx := context.Background()

	preset := &domain.Food{
		KindergartenID: domain.DietPlatformKindergartenID,
		Name:           "S3测试预置面粉",
		SpellCode:      "S3CSYMF",
		CategoryID:     f.leaf.ID,
		EdiblePercent:  100,
		Source:         domain.DietFoodSourceSystem,
		PresetKey:      testDietFoodSvcPreset,
	}
	require.NoError(t, f.foodRepo.Create(ctx, preset))
	tenant := &domain.Food{
		KindergartenID: testDietFoodSvcKG,
		Name:           "S3测试园所面粉",
		SpellCode:      "S3CSYYMF",
		CategoryID:     f.leaf.ID,
		EdiblePercent:  100,
		Source:         domain.DietFoodSourceCustom,
	}
	require.NoError(t, f.foodRepo.Create(ctx, tenant))

	rows, total, _, _, err := f.svc.ListFoods(ctx, testDietFoodSvcKG, 0, "", pagination.Query{})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, rows, 1)
	assert.Equal(t, tenant.ID, rows[0].ID, "kg=0 预置行不得出现在园所列表")

	bySpell, totalSpell, _, _, err := f.svc.ListFoods(ctx, testDietFoodSvcKG, 0, "S3CSYY", pagination.Query{})
	require.NoError(t, err)
	require.Equal(t, int64(1), totalSpell)
	require.Len(t, bySpell, 1)
	assert.Equal(t, tenant.ID, bySpell[0].ID)

	_, err = f.svc.GetFood(ctx, testDietFoodSvcKG, preset.ID)
	assert.ErrorIs(t, err, ErrDietFoodNotFound)
}
