package service

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// 营养分析测试：核心聚合全部走纯函数 buildDietAnalysisResult（不起 DB），
// 只有「快照不变 / 跨园 404 / 空态 422」三条需要真实 MySQL（无 DB 时整组跳过）。
//
// 黄金样本映射：api-samples/report-nutrition-analysis.json 的 187 行明细未落盘
// （样本只保留前 4 行），因此按已发布数值反推等价 fixture：
//   人均 = Σ / dayNum / 0.95，dayNum=5、系数 0.95；
//   每餐能量取「×0.95 前的日值」，使其 display 后与样本逐项一致；
//   12 项营养全部挂到早餐这一条明细，浓度 = 日供给 × 100 / 早餐克数。
// 这样复现 1341.75 / 1484.26 / 90.4、三餐合计 1274.65、85.88% 与分母 1380.7。

const (
	testDietAnalysisKG      uint64 = 990997
	testDietAnalysisOtherKG uint64 = 990998

	dietAnalysisTestDate = "2026-10-26"
)

func dietAnalysisGoldenTargets() map[string]float64 {
	return map[string]float64{
		"energy": 1484.26, "protein": 50.7, "fats": 49.47, "carb": 209.05,
		"vitamin_a": 573.31, "vitamin_b1": 0.67, "vitamin_b2": 0.67,
		"vitamin_c": 67.33, "calcium": 746.61, "zinc": 11.2, "iron": 12,
	}
}

func dietAnalysisTestTargets(overrides map[string]float64) map[string]float64 {
	targets := dietAnalysisGoldenTargets()
	for key, value := range overrides {
		targets[key] = value
	}
	return targets
}

func dietAnalysisConfigJSON(t *testing.T, coefficient float64, targets map[string]float64) string {
	t.Helper()
	meals := []domain.DietMealSnapshot{
		{MealKey: domain.DietMealBreakfast, Name: "早餐", EnergyPercent: 25, Sort: 1, Enabled: true},
		{MealKey: domain.DietMealMorningSnack, Name: "早点", EnergyPercent: 5, Sort: 2, Enabled: true},
		{MealKey: domain.DietMealLunch, Name: "午餐", EnergyPercent: 35, Sort: 3, Enabled: true},
		{MealKey: domain.DietMealAfternoonSnack, Name: "午点", EnergyPercent: 5, Sort: 4, Enabled: true},
		{MealKey: domain.DietMealDinner, Name: "晚餐", EnergyPercent: 25, Sort: 5, Enabled: true},
		{MealKey: domain.DietMealEveningSnack, Name: "晚点", EnergyPercent: 5, Sort: 6, Enabled: false},
	}
	targetsRows := make([]domain.DietDriTargetSnapshot, 0, len(dietAnalysisNutrientSpecs)-1)
	for _, spec := range dietAnalysisNutrientSpecs {
		if spec.key == "heat" {
			continue
		}
		targetsRows = append(targetsRows, domain.DietDriTargetSnapshot{
			NutrientKey: spec.driKey,
			Value:       targets[spec.driKey],
		})
	}
	raw, err := json.Marshal(domain.DietConfigSnapshot{
		CareType:      domain.DietCareTypeDayCare,
		FoodInputType: domain.DietFoodInputTypeMarketWeight,
		RatioMode:     domain.DietRatioModeStandard,
		Coefficient:   coefficient,
		Meals:         meals,
		DriTargets:    targetsRows,
	})
	require.NoError(t, err)
	return string(raw)
}

func dietAnalysisSnapshot(values map[string]float64) domain.DietNutritionSnapshot {
	snapshot := domain.DietNutritionSnapshot{EdiblePercent: 100}
	if edible, ok := values["edible"]; ok {
		snapshot.EdiblePercent = edible
	}
	set := func(dst **float64, key string) {
		if value, ok := values[key]; ok {
			v := value
			*dst = &v
		}
	}
	set(&snapshot.Energy, "energy")
	set(&snapshot.Heat, "heat")
	set(&snapshot.Protein, "protein")
	set(&snapshot.Fats, "fats")
	set(&snapshot.Carb, "carb")
	set(&snapshot.VitaminA, "vitamin_a")
	set(&snapshot.VitaminB1, "vitamin_b1")
	set(&snapshot.VitaminB2, "vitamin_b2")
	set(&snapshot.VitaminC, "vitamin_c")
	set(&snapshot.Calcium, "calcium")
	set(&snapshot.Zinc, "zinc")
	set(&snapshot.Iron, "iron")
	return snapshot
}

func dietAnalysisItem(
	t *testing.T,
	date string,
	meal domain.DietMealKey,
	foodID uint64,
	grams float64,
	values map[string]float64,
) domain.MealPlanItem {
	t.Helper()
	parsed, err := time.Parse(dietPlanDateLayout, date)
	require.NoError(t, err)
	raw, err := json.Marshal(dietAnalysisSnapshot(values))
	require.NoError(t, err)
	var foodPtr *uint64
	if foodID != 0 {
		id := foodID
		foodPtr = &id
	}
	return domain.MealPlanItem{
		Date:              parsed,
		Weekday:           dietWeekdayFromDate(parsed),
		MealKey:           meal,
		FoodID:            foodPtr,
		FoodName:          "S7食物" + strconv.FormatUint(foodID, 10),
		Grams:             grams,
		NutritionSnapshot: string(raw),
	}
}

func dietAnalysisPlan(
	t *testing.T,
	dayNum int,
	coefficient float64,
	targets map[string]float64,
	items []domain.MealPlanItem,
) *domain.MealPlan {
	t.Helper()
	start := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
	return &domain.MealPlan{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: 1017248}},
		Name:            "S7测试第九周",
		StartDate:       start,
		EndDate:         start.AddDate(0, 0, 6),
		DayNum:          dayNum,
		CreatedMethod:   domain.DietPlanCreatedByCustom,
		ConfigSnapshot:  dietAnalysisConfigJSON(t, coefficient, targets),
		Items:           items,
	}
}

func dietAnalysisFood(id uint64, group domain.DietFoodGroup16, flags ...string) *domain.Food {
	return &domain.Food{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: id}},
		KindergartenID:  testDietAnalysisKG,
		Name:            "S7食物" + strconv.FormatUint(id, 10),
		Group16:         group,
		SourceFlags:     strings.Join(flags, ","),
	}
}

func dietAnalysisFoodMap(foods ...*domain.Food) map[uint64]*domain.Food {
	result := make(map[uint64]*domain.Food, len(foods))
	for _, food := range foods {
		result[food.ID] = food
	}
	return result
}

func dietAnalysisNutritionByKey(result *DietAnalysisResult) map[string]DietAnalysisNutritionItem {
	items := make(map[string]DietAnalysisNutritionItem, len(result.Nutrition.Items))
	for _, item := range result.Nutrition.Items {
		items[item.Key] = item
	}
	return items
}

func dietAnalysisCompositionGrams(result *DietAnalysisResult, key domain.DietFoodGroup16) float64 {
	for _, group := range result.Composition.Groups {
		if group.Key == string(key) {
			return group.Grams
		}
	}
	return -1
}

// goldenPlan 复现参考样本的 7 组数值。5 天 × 5 餐，营养集中在早餐一条明细。
func goldenPlan(t *testing.T) *domain.MealPlan {
	t.Helper()
	dailyEnergy := map[domain.DietMealKey]float64{
		domain.DietMealBreakfast:      174.8836,
		domain.DietMealMorningSnack:   195.8936,
		domain.DietMealLunch:          482.4636,
		domain.DietMealAfternoonSnack: 73.1536,
		domain.DietMealDinner:         348.2660,
	}
	// 样本「平均每人每日」× 0.95 = 未折算日供给。
	// 脂肪单独拆到三条明细，用于复现来源表：植物性 8.48、动物性 22.38、其他 11.88。
	dailyNutrients := map[string]float64{
		"heat":       5335.73998,
		"protein":    49.84004,
		"carb":       186.72003,
		"vitamin_a":  551.69996,
		"vitamin_b1": 0.64999,
		"vitamin_b2": 0.530005,
		"vitamin_c":  61.02002,
		"calcium":    233.38004,
		"zinc":       8.289985,
		"iron":       14.229955,
	}
	fatsByMeal := map[domain.DietMealKey]float64{
		domain.DietMealBreakfast:    8.056,    // 植物性
		domain.DietMealMorningSnack: 11.28296, // 其他脂（残差）
		domain.DietMealDinner:       21.261,   // 动物性
	}
	breakfastGrams := dailyEnergy[domain.DietMealBreakfast]
	breakfastValues := map[string]float64{"energy": 100}
	for key, value := range dailyNutrients {
		breakfastValues[key] = value * 100 / breakfastGrams
	}
	breakfastValues["fats"] = fatsByMeal[domain.DietMealBreakfast] * 100 / breakfastGrams

	dates := []string{"2026-10-26", "2026-10-27", "2026-10-28", "2026-10-29", "2026-10-30"}
	mealFoods := map[domain.DietMealKey]uint64{
		domain.DietMealBreakfast:      101,
		domain.DietMealMorningSnack:   102,
		domain.DietMealLunch:          103,
		domain.DietMealAfternoonSnack: 104,
		domain.DietMealDinner:         105,
	}
	items := make([]domain.MealPlanItem, 0, len(dates)*len(mealFoods))
	for _, date := range dates {
		for _, meal := range domain.DietMealKeyOrder[:5] {
			values := map[string]float64{"energy": 100}
			if meal == domain.DietMealBreakfast {
				for key, value := range breakfastValues {
					values[key] = value
				}
			} else if fats, ok := fatsByMeal[meal]; ok {
				values["fats"] = fats * 100 / dailyEnergy[meal]
			}
			items = append(items, dietAnalysisItem(t, date, meal, mealFoods[meal], dailyEnergy[meal], values))
		}
	}
	return dietAnalysisPlan(t, 5, 0.95, dietAnalysisGoldenTargets(), items)
}

func goldenFoodMap() map[uint64]*domain.Food {
	return dietAnalysisFoodMap(
		dietAnalysisFood(101, domain.DietGroup16Grain, domain.DietSourceFlagPlant),
		dietAnalysisFood(102, domain.DietGroup16Tuber),
		dietAnalysisFood(103, domain.DietGroup16Vegetable, domain.DietSourceFlagPlant),
		dietAnalysisFood(104, domain.DietGroup16Fruit, domain.DietSourceFlagPlant),
		dietAnalysisFood(105, domain.DietGroup16Meat, domain.DietSourceFlagAnimal, domain.DietSourceFlagMeat),
	)
}

func buildGoldenResult(t *testing.T) *DietAnalysisResult {
	t.Helper()
	result, err := buildDietAnalysisResult(goldenPlan(t), goldenFoodMap())
	require.NoError(t, err)
	require.NotNil(t, result)
	return result
}

// TC-NEW-DIE-0044：两条明细按食部折算后求和。
func TestDietAnalysis_TwoItemsNutritionSum(t *testing.T) {
	items := []domain.MealPlanItem{
		dietAnalysisItem(t, dietAnalysisTestDate, domain.DietMealLunch, 201, 30,
			map[string]float64{"energy": 286}),
		dietAnalysisItem(t, dietAnalysisTestDate, domain.DietMealLunch, 202, 10,
			map[string]float64{"energy": 143}),
	}
	plan := dietAnalysisPlan(t, 1, 1, dietAnalysisTestTargets(map[string]float64{"energy": 1000}), items)
	result, err := buildDietAnalysisResult(plan, dietAnalysisFoodMap(
		dietAnalysisFood(201, domain.DietGroup16Grain),
		dietAnalysisFood(202, domain.DietGroup16Meat, domain.DietSourceFlagAnimal, domain.DietSourceFlagMeat),
	))
	require.NoError(t, err)
	energy := dietAnalysisNutritionByKey(result)["energy"]
	// 30 × 286/100 + 10 × 143/100 = 100.1
	assert.InDelta(t, 100.1, energy.Intake, 1e-9)
	assert.InDelta(t, 10.01, energy.ComparePercent, 1e-9)
}

// TC-NEW-DIE-0045：NULL 营养项不参与求和；全部缺失时按 0 展示而不是报错。
func TestDietAnalysis_NullNutrientSkipped(t *testing.T) {
	items := []domain.MealPlanItem{
		dietAnalysisItem(t, dietAnalysisTestDate, domain.DietMealLunch, 211, 100,
			map[string]float64{"energy": 100}),
		dietAnalysisItem(t, dietAnalysisTestDate, domain.DietMealLunch, 212, 100,
			map[string]float64{"energy": 100, "vitamin_c": 10}),
	}
	plan := dietAnalysisPlan(t, 1, 1, dietAnalysisTestTargets(nil), items)
	result, err := buildDietAnalysisResult(plan, dietAnalysisFoodMap(
		dietAnalysisFood(211, domain.DietGroup16Grain),
		dietAnalysisFood(212, domain.DietGroup16Fruit),
	))
	require.NoError(t, err)
	byKey := dietAnalysisNutritionByKey(result)
	assert.InDelta(t, 10.0, byKey["vitamin_c"].Intake, 1e-9, "NULL 行跳过，只累加有值明细")
	assert.InDelta(t, 0.0, byKey["zinc"].Intake, 1e-9, "全部为 NULL 时结果为 0")
	assert.Equal(t, "低", byKey["zinc"].Comment)
}

// TC-NEW-DIE-0046：显示值 ÷ 0.95、DRIs 行不缩放、比较% 等价于显示值 ÷ 全日 DRIs。
func TestDietAnalysis_ZeroNineFiveDisplay(t *testing.T) {
	items := []domain.MealPlanItem{
		dietAnalysisItem(t, dietAnalysisTestDate, domain.DietMealLunch, 221, 100,
			map[string]float64{"energy": 95}),
	}
	plan := dietAnalysisPlan(t, 1, 0.95, dietAnalysisTestTargets(map[string]float64{"energy": 200}), items)
	result, err := buildDietAnalysisResult(plan, dietAnalysisFoodMap(dietAnalysisFood(221, domain.DietGroup16Grain)))
	require.NoError(t, err)
	energy := dietAnalysisNutritionByKey(result)["energy"]
	assert.InDelta(t, 100.0, energy.Intake, 1e-9, "95 ÷ 0.95 = 100")
	assert.InDelta(t, 200.0, energy.Dri, 1e-9, "DRIs 显示全日原值，不乘系数")
	assert.InDelta(t, 50.0, energy.ComparePercent, 1e-9, "比较% = 100 / 200")
}

// TC-NEW-DIE-0047：系数变为 0.70 后显示值与比较% 按新系数重算。
func TestDietAnalysis_CoefficientChange(t *testing.T) {
	items := []domain.MealPlanItem{
		dietAnalysisItem(t, dietAnalysisTestDate, domain.DietMealLunch, 231, 100,
			map[string]float64{"energy": 95}),
	}
	targets := dietAnalysisTestTargets(map[string]float64{"energy": 200})
	plan := dietAnalysisPlan(t, 1, 0.70, targets, items)
	result, err := buildDietAnalysisResult(plan, dietAnalysisFoodMap(dietAnalysisFood(231, domain.DietGroup16Grain)))
	require.NoError(t, err)
	energy := dietAnalysisNutritionByKey(result)["energy"]
	assert.InDelta(t, 135.71, energy.Intake, 1e-9, "95 ÷ 0.70")
	assert.InDelta(t, 67.86, energy.ComparePercent, 1e-9)
	assert.InDelta(t, 0.70, result.Plan.Coefficient, 1e-9)
}

// TC-NEW-DIE-0048：DRI 目标值来自 config_snapshot，千焦由能量 × 4.186 推导。
func TestDietAnalysis_DriTargetsFromSnapshot(t *testing.T) {
	result := buildGoldenResult(t)
	byKey := dietAnalysisNutritionByKey(result)
	want := map[string]float64{
		"energy": 1484.26, "heat": 6213.11, "protein": 50.7, "fats": 49.47,
		"carb": 209.05, "vitamin_a": 573.31, "vitamin_b1": 0.67, "vitamin_b2": 0.67,
		"vitamin_c": 67.33, "calcium": 746.61, "zinc": 11.2, "iron": 12,
	}
	for key, value := range want {
		assert.InDelta(t, value, byKey[key].Dri, 0.0001, "%s 的 DRIs", key)
	}
	assert.NotContains(t, byKey, "retinol")
	assert.NotContains(t, byKey, "carotin")
	// 快照只存 11 项，千焦不在其中。
	assert.NotZero(t, byKey["heat"].Dri)
}

// TC-NEW-DIE-0049：±20% 闭区间边界。
func TestDietAnalysis_EvaluationBoundaries(t *testing.T) {
	items := []domain.MealPlanItem{
		dietAnalysisItem(t, dietAnalysisTestDate, domain.DietMealLunch, 241, 100, map[string]float64{
			"protein": 79.98, "fats": 80.00, "carb": 120.00, "calcium": 120.07,
		}),
	}
	targets := dietAnalysisTestTargets(map[string]float64{
		"protein": 100, "fats": 100, "carb": 100, "calcium": 100,
	})
	plan := dietAnalysisPlan(t, 1, 1, targets, items)
	result, err := buildDietAnalysisResult(plan, dietAnalysisFoodMap(dietAnalysisFood(241, domain.DietGroup16Grain)))
	require.NoError(t, err)
	byKey := dietAnalysisNutritionByKey(result)
	assert.Equal(t, "低", byKey["protein"].Comment)
	assert.Equal(t, "合格", byKey["fats"].Comment)
	assert.Equal(t, "合格", byKey["carb"].Comment)
	assert.Equal(t, "高", byKey["calcium"].Comment)
}

// TC-NEW-DIE-0050：低与高都触发评价（样本值 32.9% / 124.82%）。
func TestDietAnalysis_EvaluationLowHigh(t *testing.T) {
	items := []domain.MealPlanItem{
		dietAnalysisItem(t, dietAnalysisTestDate, domain.DietMealLunch, 251, 100, map[string]float64{
			"calcium": 32.9, "iron": 124.82,
		}),
	}
	targets := dietAnalysisTestTargets(map[string]float64{"calcium": 100, "iron": 100})
	plan := dietAnalysisPlan(t, 1, 1, targets, items)
	result, err := buildDietAnalysisResult(plan, dietAnalysisFoodMap(dietAnalysisFood(251, domain.DietGroup16Grain)))
	require.NoError(t, err)
	byKey := dietAnalysisNutritionByKey(result)
	assert.Equal(t, "低", byKey["calcium"].Comment)
	assert.Equal(t, "高", byKey["iron"].Comment)
	assert.InDelta(t, 32.9, byKey["calcium"].ComparePercent, 1e-9)
	assert.InDelta(t, 124.82, byKey["iron"].ComparePercent, 1e-9)
}

// TC-NEW-DIE-0051 / TC-NEW-DIE-0058：营养分析表 12 项数值与样本精确匹配。
func TestDietAnalysis_NutritionGoldenSample(t *testing.T) {
	result := buildGoldenResult(t)
	require.Len(t, result.Nutrition.Items, 12)
	byKey := dietAnalysisNutritionByKey(result)

	wantIntake := map[string]float64{
		"energy": 1341.75, "heat": 5616.57, "protein": 52.46, "fats": 42.74,
		"carb": 196.55, "vitamin_a": 580.74, "vitamin_b1": 0.68, "vitamin_b2": 0.56,
		"vitamin_c": 64.23, "calcium": 245.66, "zinc": 8.73, "iron": 14.98,
	}
	wantCompare := map[string]float64{
		"energy": 90.4, "heat": 90.4, "protein": 103.48, "fats": 86.39,
		"carb": 94.02, "vitamin_a": 101.3, "vitamin_b1": 102.12, "vitamin_b2": 83.27,
		"vitamin_c": 95.4, "calcium": 32.9, "zinc": 77.91, "iron": 124.82,
	}
	for key, value := range wantIntake {
		assert.InDelta(t, value, byKey[key].Intake, 0.011, "%s intake", key)
	}
	for key, value := range wantCompare {
		assert.InDelta(t, value, byKey[key].ComparePercent, 0.011, "%s compare", key)
	}
	assert.Equal(t, "合格", byKey["energy"].Comment)
	assert.Equal(t, "低", byKey["calcium"].Comment)
	assert.Equal(t, "低", byKey["zinc"].Comment)
	assert.Equal(t, "高", byKey["iron"].Comment)

	// 契约顺序：热量(千卡/千焦)/蛋白质/脂肪/碳水/维生素A/B1/B2/C/钙/锌/铁。
	wantKeys := []string{
		"energy", "heat", "protein", "fats", "carb", "vitamin_a",
		"vitamin_b1", "vitamin_b2", "vitamin_c", "calcium", "zinc", "iron",
	}
	gotKeys := make([]string, 0, len(result.Nutrition.Items))
	for _, item := range result.Nutrition.Items {
		gotKeys = append(gotKeys, item.Key)
	}
	assert.Equal(t, wantKeys, gotKeys)
}

// TC-NEW-DIE-0052：三大营养素供能四行口径与分母。
func TestDietAnalysis_MacroEnergy(t *testing.T) {
	result := buildGoldenResult(t)
	require.Len(t, result.MacroEnergy.Rows, 4)
	assert.InDelta(t, 1380.7, result.MacroEnergy.Denominator, 1e-9,
		"分母 = 蛋白×4 + 脂肪×9 + 碳水×4")

	rows := make(map[string]DietAnalysisMacroRow, len(result.MacroEnergy.Rows))
	for _, row := range result.MacroEnergy.Rows {
		rows[row.Key] = row
	}
	// 前两行是 DRIs 派生的「要求」值（样本 202.8 = 50.7 × 4）。
	assert.InDelta(t, 202.8, rows["kcal"].Protein, 1e-9)
	assert.InDelta(t, 445.23, rows["kcal"].Fats, 1e-9)
	assert.InDelta(t, 836.2, rows["kcal"].Carb, 1e-9)
	assert.InDelta(t, 848.92, rows["kj"].Protein, 1e-9)
	assert.InDelta(t, 1863.73, rows["kj"].Fats, 1e-9)
	assert.InDelta(t, 3500.33, rows["kj"].Carb, 1e-9)

	assert.InDelta(t, 15.2, rows["percent"].Protein, 1e-9)
	assert.InDelta(t, 27.86, rows["percent"].Fats, 1e-9)
	assert.InDelta(t, 56.94, rows["percent"].Carb, 1e-9)
	assert.InDelta(t, 100, rows["percent"].Protein+rows["percent"].Fats+rows["percent"].Carb, 1e-9)
	require.NotNil(t, rows["percent"].Recommended)
	assert.Equal(t, "12-15%", rows["percent"].Recommended.Protein)
	assert.Equal(t, "30-35%", rows["percent"].Recommended.Fats)
	assert.Equal(t, "50-60%", rows["percent"].Recommended.Carb)

	assert.InDelta(t, 1.0, rows["grams_ratio"].Protein, 1e-9)
	assert.InDelta(t, 0.81, rows["grams_ratio"].Fats, 1e-9)
	assert.InDelta(t, 3.75, rows["grams_ratio"].Carb, 1e-9)
	require.NotNil(t, rows["grams_ratio"].Recommended)
	assert.Equal(t, "4-5", rows["grams_ratio"].Recommended.Carb)
}

// TC-NEW-DIE-0053 / TC-NEW-DIE-0060：三餐供能比 6 行与 85.88% 合计。
func TestDietAnalysis_MealEnergy(t *testing.T) {
	result := buildGoldenResult(t)
	require.Len(t, result.MealEnergy.Meals, 6)

	wantEnergy := []float64{174.88, 195.89, 482.46, 73.15, 348.27, 0}
	wantPercent := []float64{11.78, 13.2, 32.51, 4.93, 23.46, 0}
	wantKeys := []string{"breakfast", "morning_snack", "lunch", "afternoon_snack", "dinner", "evening_snack"}
	wantLabels := []string{"早餐", "早点", "午餐", "午点", "晚餐", "晚点"}
	for i, meal := range result.MealEnergy.Meals {
		assert.Equal(t, wantKeys[i], meal.MealKey)
		assert.Equal(t, wantLabels[i], meal.Label)
		assert.InDelta(t, wantEnergy[i], meal.Energy, 0.011, "第 %d 餐能量", i)
		assert.InDelta(t, wantPercent[i], meal.Percent, 0.011, "第 %d 餐比例", i)
	}
	assert.InDelta(t, 85.88, result.MealEnergy.TotalPercent, 0.011,
		"三餐比例合计 85.88%，不归一化为 100%")
}

// TC-NEW-DIE-0054：膳食构成返回 16 类、只标记 10 类展示。
func TestDietAnalysis_Composition16Groups(t *testing.T) {
	groups := []domain.DietFoodGroup16{
		domain.DietGroup16Grain, domain.DietGroup16Tuber, domain.DietGroup16Vegetable,
		domain.DietGroup16Fruit, domain.DietGroup16Meat, domain.DietGroup16Egg,
		domain.DietGroup16Aquatic, domain.DietGroup16Soybean, domain.DietGroup16Dairy,
		domain.DietGroup16Oil, domain.DietGroup16Nut, domain.DietGroup16InstantFood,
		domain.DietGroup16Beverage, domain.DietGroup16Confection, domain.DietGroup16Condiment,
		domain.DietGroup16Other,
	}
	items := make([]domain.MealPlanItem, 0, len(groups))
	foods := make([]*domain.Food, 0, len(groups))
	for i, group := range groups {
		id := uint64(300 + i)
		items = append(items, dietAnalysisItem(t, dietAnalysisTestDate, domain.DietMealLunch, id, 10,
			map[string]float64{"energy": 10}))
		foods = append(foods, dietAnalysisFood(id, group))
	}
	plan := dietAnalysisPlan(t, 1, 1, dietAnalysisTestTargets(nil), items)
	result, err := buildDietAnalysisResult(plan, dietAnalysisFoodMap(foods...))
	require.NoError(t, err)
	require.Len(t, result.Composition.Groups, 16)
	displayed := 0
	for _, group := range result.Composition.Groups {
		if group.Displayed {
			displayed++
		}
		assert.InDelta(t, 10.0, group.Grams, 1e-9, group.Key)
	}
	assert.Equal(t, 10, displayed)
	assert.Equal(t, "谷类", result.Composition.Groups[0].Label)
	assert.Equal(t, "其他", result.Composition.Groups[15].Label)
	assert.False(t, result.Composition.Groups[15].Displayed)
	// 未识别的食物（无 FoodID）落「其他」。
	assert.InDelta(t, 10.0, result.Composition.Groups[15].Grams, 1e-9)
}

// TC-NEW-DIE-0055：钙三来源不互斥合计 126%，铁两来源互斥合计 100%。
func TestDietAnalysis_CalciumIron(t *testing.T) {
	items := []domain.MealPlanItem{
		dietAnalysisItem(t, dietAnalysisTestDate, domain.DietMealLunch, 401, 100,
			map[string]float64{"calcium": 36.44}),
		dietAnalysisItem(t, dietAnalysisTestDate, domain.DietMealLunch, 402, 100,
			map[string]float64{"calcium": 69.47}),
		dietAnalysisItem(t, dietAnalysisTestDate, domain.DietMealLunch, 403, 100,
			map[string]float64{"calcium": 134.12}),
		dietAnalysisItem(t, dietAnalysisTestDate, domain.DietMealLunch, 404, 100,
			map[string]float64{"calcium": 5.63}),
		dietAnalysisItem(t, dietAnalysisTestDate, domain.DietMealLunch, 405, 100,
			map[string]float64{"iron": 2.35}),
		dietAnalysisItem(t, dietAnalysisTestDate, domain.DietMealLunch, 406, 100,
			map[string]float64{"iron": 12.6289}),
	}
	foods := dietAnalysisFoodMap(
		dietAnalysisFood(401, domain.DietGroup16Meat, domain.DietSourceFlagAnimal, domain.DietSourceFlagMeat),
		dietAnalysisFood(402, domain.DietGroup16Dairy, domain.DietSourceFlagAnimal, domain.DietSourceFlagMilk),
		dietAnalysisFood(403, domain.DietGroup16Vegetable, domain.DietSourceFlagPlant),
		dietAnalysisFood(404, domain.DietGroup16Other),
		dietAnalysisFood(405, domain.DietGroup16Meat, domain.DietSourceFlagAnimal, domain.DietSourceFlagMeat),
		dietAnalysisFood(406, domain.DietGroup16Grain, domain.DietSourceFlagPlant),
	)
	plan := dietAnalysisPlan(t, 1, 1, dietAnalysisTestTargets(nil), items)
	result, err := buildDietAnalysisResult(plan, foods)
	require.NoError(t, err)

	calcium := result.CalciumIron.Calcium
	assert.InDelta(t, 245.66, calcium.Total, 0.011)
	require.Len(t, calcium.Sources, 3)
	assert.Equal(t, "animal", calcium.Sources[0].Key)
	assert.InDelta(t, 105.91, calcium.Sources[0].Value, 0.011, "动物性含乳类")
	assert.InDelta(t, 43.11, calcium.Sources[0].Percent, 0.011)
	assert.InDelta(t, 134.12, calcium.Sources[1].Value, 0.011)
	assert.InDelta(t, 54.6, calcium.Sources[1].Percent, 0.011)
	assert.InDelta(t, 69.47, calcium.Sources[2].Value, 0.011, "乳类重复计入动物性")
	assert.InDelta(t, 28.28, calcium.Sources[2].Percent, 0.011)
	sourceSum := 0.0
	for _, source := range calcium.Sources {
		sourceSum += source.Percent
	}
	assert.InDelta(t, 126.0, sourceSum, 0.02, "钙三来源合计 126%，不互斥")

	iron := result.CalciumIron.Iron
	assert.InDelta(t, 14.98, iron.Total, 0.011)
	require.Len(t, iron.Sources, 2)
	assert.Equal(t, "meat", iron.Sources[0].Key)
	assert.InDelta(t, 2.35, iron.Sources[0].Value, 0.011, "铁肉类只认畜肉+禽肉")
	assert.InDelta(t, 15.69, iron.Sources[0].Percent, 0.011)
	assert.InDelta(t, 12.63, iron.Sources[1].Value, 0.011)
	assert.InDelta(t, 100.0, iron.Sources[0].Percent+iron.Sources[1].Percent, 0.02)
}

// TC-NEW-DIE-0056：蛋白质来源只算动物+豆类，三项合计不等于总蛋白。
func TestDietAnalysis_ProteinSource(t *testing.T) {
	items := []domain.MealPlanItem{
		dietAnalysisItem(t, dietAnalysisTestDate, domain.DietMealLunch, 501, 100,
			map[string]float64{"protein": 54.05}),
		dietAnalysisItem(t, dietAnalysisTestDate, domain.DietMealLunch, 502, 100,
			map[string]float64{"protein": 8.44}),
		dietAnalysisItem(t, dietAnalysisTestDate, domain.DietMealLunch, 503, 100,
			map[string]float64{"protein": 37.51}),
	}
	foods := dietAnalysisFoodMap(
		dietAnalysisFood(501, domain.DietGroup16Meat, domain.DietSourceFlagAnimal),
		dietAnalysisFood(502, domain.DietGroup16Soybean, domain.DietSourceFlagPlant),
		dietAnalysisFood(503, domain.DietGroup16Grain, domain.DietSourceFlagPlant),
	)
	plan := dietAnalysisPlan(t, 1, 1, dietAnalysisTestTargets(nil), items)
	result, err := buildDietAnalysisResult(plan, foods)
	require.NoError(t, err)

	source := result.ProteinSource
	assert.InDelta(t, 100.0, source.Total, 0.011)
	require.Len(t, source.Items, 3)
	assert.Equal(t, "animal", source.Items[0].Key)
	assert.InDelta(t, 54.05, source.Items[0].Value, 0.011)
	assert.InDelta(t, 54.05, source.Items[0].Percent, 0.011)
	assert.Equal(t, "bean", source.Items[1].Key)
	assert.InDelta(t, 8.44, source.Items[1].Value, 0.011)
	assert.Equal(t, "required", source.Items[2].Key)
	assert.InDelta(t, 50.0, source.Items[2].Value, 0.011, "要求 = 总蛋白 × 50%")
	assert.InDelta(t, 62.49, source.AnimalBeanPercent, 0.011)
	assert.InDelta(t, 62.49, source.Items[0].Value+source.Items[1].Value, 0.011)
	assert.NotEqual(t, source.Total, source.Items[0].Value+source.Items[1].Value+source.Items[2].Value,
		"只统计动物+豆类，不是穷尽分解")
}

// TC-NEW-DIE-0057：脂肪来源 3 项、页面 2 列，其他脂是残差。
func TestDietAnalysis_FatSource(t *testing.T) {
	items := []domain.MealPlanItem{
		dietAnalysisItem(t, dietAnalysisTestDate, domain.DietMealLunch, 601, 100,
			map[string]float64{"fats": 22.38}),
		dietAnalysisItem(t, dietAnalysisTestDate, domain.DietMealLunch, 602, 100,
			map[string]float64{"fats": 8.48}),
		dietAnalysisItem(t, dietAnalysisTestDate, domain.DietMealLunch, 603, 100,
			map[string]float64{"fats": 11.88}),
	}
	foods := dietAnalysisFoodMap(
		dietAnalysisFood(601, domain.DietGroup16Meat, domain.DietSourceFlagAnimal),
		dietAnalysisFood(602, domain.DietGroup16Vegetable, domain.DietSourceFlagPlant),
		dietAnalysisFood(603, domain.DietGroup16Condiment),
	)
	plan := dietAnalysisPlan(t, 1, 1, dietAnalysisTestTargets(nil), items)
	result, err := buildDietAnalysisResult(plan, foods)
	require.NoError(t, err)

	source := result.FatSource
	assert.InDelta(t, 42.74, source.Total, 0.011)
	require.Len(t, source.Items, 3)
	assert.InDelta(t, 22.38, source.Items[0].Value, 0.011)
	assert.InDelta(t, 52.36, source.Items[0].Percent, 0.011)
	assert.InDelta(t, 8.48, source.Items[1].Value, 0.011)
	assert.InDelta(t, 19.84, source.Items[1].Percent, 0.011)
	assert.InDelta(t, 11.88, source.Items[2].Value, 0.011, "其他脂是残差")
	assert.True(t, source.Items[0].Displayed)
	assert.True(t, source.Items[1].Displayed)
	assert.False(t, source.Items[2].Displayed, "其他脂只进分析图，不进页面/导出表格")
	assert.InDelta(t, 100.0,
		source.Items[0].Percent+source.Items[1].Percent+source.Items[2].Percent, 0.02)
}

// TC-NEW-DIE-0059：响应不出现 retinol / carotin；视黄醇列由 vitamin_a 复用。
func TestDietAnalysis_NoRetinolCarotinInResponse(t *testing.T) {
	result := buildGoldenResult(t)
	raw, err := json.Marshal(result)
	require.NoError(t, err)
	body := string(raw)
	assert.NotContains(t, body, "retinol")
	assert.NotContains(t, body, "carotin")
	assert.Contains(t, body, `"vitamin_a"`)
}

// TC-NEW-DIE-0061：钙偏低触发固定建议话术。
func TestDietAnalysis_ConclusionCalciumLowSuggestion(t *testing.T) {
	result := buildGoldenResult(t)
	conclusion := result.Conclusion
	var calcium, zinc *DietAnalysisSuggestion
	for i := range conclusion.Suggestions {
		switch conclusion.Suggestions[i].NutrientKey {
		case "calcium":
			calcium = &conclusion.Suggestions[i]
		case "zinc":
			zinc = &conclusion.Suggestions[i]
		}
	}
	require.NotNil(t, calcium, "钙评价为低必须触发建议")
	assert.Equal(t, "low", calcium.Level)
	assert.Equal(t,
		"钙摄入量偏低。建议补充一些牛奶与奶制品鱼类、大豆与豆制品、杏仁、酵母。",
		calcium.Text)
	require.NotNil(t, zinc)
	assert.Equal(t, "low", zinc.Level)
	assert.Equal(t,
		"锌摄入量偏低。建议补充一些牛肉、乳酪、蛋鱼、豆类、果仁、麦牙、粗粮。",
		zinc.Text)
	assert.Len(t, conclusion.Suggestions, 2)
}

// TC-NEW-DIE-0062：合格营养素不触发建议；未登记话术的高值也不编造文案。
func TestDietAnalysis_ConclusionNoSuggestionForOK(t *testing.T) {
	result := buildGoldenResult(t)
	okKeys := make([]string, 0, len(result.Nutrition.Items))
	for _, item := range result.Nutrition.Items {
		if item.Comment == "合格" {
			okKeys = append(okKeys, item.Key)
		}
	}
	require.NotEmpty(t, okKeys)
	for _, suggestion := range result.Conclusion.Suggestions {
		assert.NotContains(t, okKeys, suggestion.NutrientKey, "合格项不得出现在建议里")
	}
	// 样本中铁为「高」但没有建议文案，复刻同样不输出（证据范围内只有钙/锌话术）。
	for _, suggestion := range result.Conclusion.Suggestions {
		assert.NotEqual(t, "iron", suggestion.NutrientKey)
	}
	assert.Len(t, result.Conclusion.Suggestions, 2)
	assert.Equal(t, "1 : 0.81 : 3.75", result.Conclusion.MacroRatio)
	assert.InDelta(t, 15.2, result.Conclusion.MacroPercent.Protein, 1e-9)
	assert.InDelta(t, 90.4, result.Conclusion.DriPercent.Energy, 1e-9)
}

// 食品种类数：按食物去重，调味品单独计数。
func TestDietAnalysis_ConclusionFoodKindCount(t *testing.T) {
	items := []domain.MealPlanItem{
		dietAnalysisItem(t, "2026-10-26", domain.DietMealLunch, 701, 10, map[string]float64{"energy": 10}),
		dietAnalysisItem(t, "2026-10-27", domain.DietMealLunch, 701, 10, map[string]float64{"energy": 10}),
		dietAnalysisItem(t, "2026-10-26", domain.DietMealLunch, 702, 10, map[string]float64{"energy": 10}),
		dietAnalysisItem(t, "2026-10-26", domain.DietMealLunch, 703, 10, map[string]float64{"energy": 10}),
	}
	foods := dietAnalysisFoodMap(
		dietAnalysisFood(701, domain.DietGroup16Grain),
		dietAnalysisFood(702, domain.DietGroup16Condiment),
		dietAnalysisFood(703, domain.DietGroup16Condiment),
	)
	plan := dietAnalysisPlan(t, 2, 1, dietAnalysisTestTargets(nil), items)
	result, err := buildDietAnalysisResult(plan, foods)
	require.NoError(t, err)
	assert.Equal(t, 3, result.Conclusion.FoodKindCount)
	assert.Equal(t, 2, result.Conclusion.CondimentKindCount)
	assert.Equal(t, 1, result.Conclusion.NonCondimentKindCount)
}

// 空明细直接返回 422 语义错误。
func TestDietAnalysis_EmptyPlanReturnsError(t *testing.T) {
	plan := dietAnalysisPlan(t, 0, 0.95, dietAnalysisTestTargets(nil), nil)
	_, err := buildDietAnalysisResult(plan, nil)
	assert.True(t, errors.Is(err, ErrDietAnalysisEmpty))
}

// --- 需要真实 MySQL 的三条：快照不变、跨园 404、空态 422 ---

// 真实 MySQL 测试沿用 S6 的连接夹具；无 DB 时整组跳过。
func newDietAnalysisDBFixture(t *testing.T) (context.Context, *repository.DietPlanRepo, *repository.DietFoodRepo, func()) {
	t.Helper()
	db := openDietPlanServiceTestDB(t)
	cleanup := func() {
		for _, sql := range []string{
			"DELETE FROM diet_meal_plan_items WHERE kindergarten_id IN (?, ?)",
			"DELETE FROM diet_meal_plans WHERE kindergarten_id IN (?, ?)",
			"DELETE FROM diet_foods WHERE kindergarten_id IN (?, ?)",
		} {
			if err := db.Exec(sql, testDietAnalysisKG, testDietAnalysisOtherKG).Error; err != nil {
				t.Logf("cleanup %q: %v", sql, err)
			}
		}
	}
	cleanup()
	t.Cleanup(cleanup)
	return context.Background(),
		repository.NewDietPlanRepo(db),
		repository.NewDietFoodRepo(db),
		cleanup
}

// TC-NEW-DIE-0063：保存后修改食物库的分类与营养值，历史报表 7 组全部不变。
func TestDietAnalysis_SnapshotUnaffectedByFoodChanges(t *testing.T) {
	f := newDietPlanFixture(t)
	ctx := context.Background()

	food := &domain.Food{
		KindergartenID: testDietPlanSvcKG,
		Name:           "S7快照食物",
		CategoryID:     1,
		EdiblePercent:  100,
		Source:         domain.DietFoodSourceCustom,
		Group16:        domain.DietGroup16Grain,
		SourceFlags:    domain.DietSourceFlagPlant,
		Energy:         dietFloat(200),
		Protein:        dietFloat(10),
		Fats:           dietFloat(5),
		Calcium:        dietFloat(40),
		Iron:           dietFloat(2),
	}
	require.NoError(t, f.foodRepo.Create(ctx, food))

	plan := f.createCustomPlan(t, "S7快照食谱")
	_, err := f.svc.SavePlan(ctx, testDietPlanSvcKG, plan.ID, DietPlanSaveInput{
		Items: []DietPlanItemInput{
			dietPlanItemInput("2026-10-26", domain.DietMealLunch, "米饭", food, 100),
		},
	})
	require.NoError(t, err)

	// 保存时必须把分类固化进明细快照，否则下面的断言走的是旧行回退路径。
	detail, err := f.svc.GetPlan(ctx, testDietPlanSvcKG, plan.ID)
	require.NoError(t, err)
	require.Len(t, detail.Items, 1)
	assert.Equal(t, domain.DietGroup16Grain, detail.Items[0].NutritionSnapshot.Group16)
	assert.Equal(t, domain.DietSourceFlagPlant, detail.Items[0].NutritionSnapshot.SourceFlags)

	analysis := NewDietAnalysisService(f.planRepo, f.foodRepo)
	before, err := analysis.Analyze(ctx, testDietPlanSvcKG, plan.ID)
	require.NoError(t, err)

	// 非零断言，确保比较的不是一组全零报表。
	assert.InDelta(t, 105.26, dietAnalysisCompositionGrams(before, domain.DietGroup16Grain), 0.011)
	assert.InDelta(t, 10.53, before.ProteinSource.Total, 0.011)
	assert.InDelta(t, 42.11, before.CalciumIron.Calcium.Total, 0.011)
	assert.InDelta(t, 5.26, before.FatSource.Total, 0.011)

	// 改分类、来源标记与全部营养值；历史报表必须原样返回。
	food.Group16 = domain.DietGroup16Meat
	food.SourceFlags = domain.DietSourceFlagAnimal + "," + domain.DietSourceFlagMilk + "," + domain.DietSourceFlagMeat
	food.Energy = dietFloat(500)
	food.Protein = dietFloat(99)
	food.Fats = dietFloat(88)
	food.Calcium = dietFloat(777)
	food.Iron = dietFloat(66)
	require.NoError(t, f.foodRepo.Update(ctx, food))

	after, err := analysis.Analyze(ctx, testDietPlanSvcKG, plan.ID)
	require.NoError(t, err)
	beforeJSON, err := json.Marshal(before)
	require.NoError(t, err)
	afterJSON, err := json.Marshal(after)
	require.NoError(t, err)
	assert.JSONEq(t, string(beforeJSON), string(afterJSON),
		"改食物库分类与营养值不得改变历史报表任一字段（TC-0063）")
}

// 旧行快照没有 group16 / source_flags：分析回退到食物库当前分类。
func TestDietAnalysis_LegacySnapshotFallsBackToFood(t *testing.T) {
	items := []domain.MealPlanItem{
		dietAnalysisItem(t, dietAnalysisTestDate, domain.DietMealLunch, 301, 100,
			map[string]float64{"energy": 100, "protein": 10, "calcium": 40}),
	}
	plan := dietAnalysisPlan(t, 1, 1, dietAnalysisTestTargets(nil), items)
	result, err := buildDietAnalysisResult(plan, dietAnalysisFoodMap(
		dietAnalysisFood(301, domain.DietGroup16Meat,
			domain.DietSourceFlagAnimal, domain.DietSourceFlagMilk),
	))
	require.NoError(t, err)
	assert.InDelta(t, 100, dietAnalysisCompositionGrams(result, domain.DietGroup16Meat), 1e-9)
	assert.InDelta(t, 10, result.ProteinSource.Items[0].Value, 1e-9, "动物蛋白")
	assert.InDelta(t, 40, result.CalciumIron.Calcium.Sources[0].Value, 1e-9, "动物性钙")
}

// 跨园访问必须 404；空明细必须 422。
func TestDietAnalysis_CrossTenantAndEmptyPlan(t *testing.T) {
	ctx, planRepo, foodRepo, _ := newDietAnalysisDBFixture(t)
	svc := NewDietAnalysisService(planRepo, foodRepo)

	start := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
	empty := &domain.MealPlan{
		KindergartenID: testDietAnalysisKG,
		Name:           "S7空食谱",
		StartDate:      start,
		EndDate:        start.AddDate(0, 0, 6),
		DayNum:         0,
		CreatedMethod:  domain.DietPlanCreatedByCustom,
		ConfigSnapshot: dietAnalysisConfigJSON(t, 0.95, dietAnalysisTestTargets(nil)),
	}
	require.NoError(t, planRepo.Create(ctx, empty))

	_, err := svc.Analyze(ctx, testDietAnalysisKG, empty.ID)
	assert.True(t, errors.Is(err, ErrDietAnalysisEmpty), "无明细应返回 422 语义错误：%v", err)

	_, err = svc.Analyze(ctx, testDietAnalysisOtherKG, empty.ID)
	assert.True(t, errors.Is(err, ErrDietPlanNotFound), "跨园必须 404：%v", err)

	_, err = svc.Analyze(ctx, 0, empty.ID)
	require.Error(t, err)
	var invalid *DietPlanValidationError
	assert.True(t, errors.As(err, &invalid), "未绑定园所应为业务校验错误")
}
