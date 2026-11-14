package service

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// 营养分析：只读周食谱的两份快照（config_snapshot / nutrition_snapshot），
// 聚合参考产品的 7 组报表与自动结论。口径刻意保留旧产品的非标准算法（ADR-0004）：
// 三个能量总量不统一、膳食构成返回 16 类但页面只展示 10 类、钙来源不互斥等。
//
// 营养值、食物分类（group16 / source_flags）与供餐配置都只认快照，
// 改食物库不会回溯历史报表（TC-NEW-DIE-0063）；旧行快照缺分类时回退食物库当前值。

var (
	// ErrDietAnalysisEmpty 表示食谱没有明细，无法生成营养分析（契约 422）。
	ErrDietAnalysisEmpty = errors.New("DIET_ANALYSIS_EMPTY")
)

const (
	// dietAnalysisDefaultCoefficient 用于快照缺系数时兜底，避免除零。
	dietAnalysisDefaultCoefficient = 1.0
	// dietAnalysisHeatFactor 是参考产品的千卡→千焦换算系数（不是精确物理值）。
	dietAnalysisHeatFactor = 4.186

	dietAnalysisProteinKcal = 4.0
	dietAnalysisFatsKcal    = 9.0
	dietAnalysisCarbKcal    = 4.0
)

// DietAnalysisPlan 是报表顶部标识的食谱信息。
type DietAnalysisPlan struct {
	ID          uint64  `json:"id"`
	Name        string  `json:"name"`
	StartDate   string  `json:"start_date"`
	EndDate     string  `json:"end_date"`
	DayNum      int     `json:"day_num"`
	Coefficient float64 `json:"coefficient"`
}

// DietAnalysisNutritionItem 是营养分析表的一行（12 项数值）。
type DietAnalysisNutritionItem struct {
	Key            string  `json:"key"`
	Label          string  `json:"label"`
	Unit           string  `json:"unit"`
	Intake         float64 `json:"intake"`
	Dri            float64 `json:"dri"`
	ComparePercent float64 `json:"compare_percent"`
	Comment        string  `json:"comment"`
}

// DietAnalysisNutrition 是营养分析表。
type DietAnalysisNutrition struct {
	Items []DietAnalysisNutritionItem `json:"items"`
}

// DietAnalysisMacroRecommended 是「占总热量%」「营养素比例」两行的推荐区间。
type DietAnalysisMacroRecommended struct {
	Protein string `json:"protein"`
	Fats    string `json:"fats"`
	Carb    string `json:"carb"`
}

// DietAnalysisMacroRow 是三大营养素供能的一行。
type DietAnalysisMacroRow struct {
	Key         string                        `json:"key"`
	Label       string                        `json:"label"`
	Protein     float64                       `json:"protein"`
	Fats        float64                       `json:"fats"`
	Carb        float64                       `json:"carb"`
	Recommended *DietAnalysisMacroRecommended `json:"recommended,omitempty"`
}

// DietAnalysisMacroEnergy 是三大营养素供能表。
type DietAnalysisMacroEnergy struct {
	Denominator float64                `json:"denominator"`
	Rows        []DietAnalysisMacroRow `json:"rows"`
}

// DietAnalysisMeal 是三餐供能比的一行。
type DietAnalysisMeal struct {
	MealKey string  `json:"meal_key"`
	Label   string  `json:"label"`
	Energy  float64 `json:"energy"`
	Percent float64 `json:"percent"`
}

// DietAnalysisMealEnergy 是三餐供能比表。
type DietAnalysisMealEnergy struct {
	Meals        []DietAnalysisMeal `json:"meals"`
	TotalPercent float64            `json:"total_percent"`
}

// DietAnalysisCompositionGroup 是膳食构成的一类。
type DietAnalysisCompositionGroup struct {
	Key       string  `json:"key"`
	Label     string  `json:"label"`
	Grams     float64 `json:"grams"`
	Displayed bool    `json:"displayed"`
}

// DietAnalysisComposition 是膳食构成表（16 类，页面只渲染 displayed=true 的 10 类）。
type DietAnalysisComposition struct {
	Groups []DietAnalysisCompositionGroup `json:"groups"`
}

// DietAnalysisSource 是钙/铁来源的一项。
type DietAnalysisSource struct {
	Key     string  `json:"key"`
	Label   string  `json:"label"`
	Value   float64 `json:"value"`
	Percent float64 `json:"percent"`
}

// DietAnalysisCalciumIronSide 是钙或铁的一组来源。
type DietAnalysisCalciumIronSide struct {
	Total   float64              `json:"total"`
	Sources []DietAnalysisSource `json:"sources"`
}

// DietAnalysisCalciumIron 是钙铁来源表。
type DietAnalysisCalciumIron struct {
	Calcium DietAnalysisCalciumIronSide `json:"calcium"`
	Iron    DietAnalysisCalciumIronSide `json:"iron"`
}

// DietAnalysisProteinItem 是蛋白质来源的一项。
type DietAnalysisProteinItem struct {
	Key     string  `json:"key"`
	Label   string  `json:"label"`
	Value   float64 `json:"value"`
	Percent float64 `json:"percent"`
}

// DietAnalysisProteinSource 是蛋白质来源表：只统计动物 + 豆类，不是穷尽分解。
type DietAnalysisProteinSource struct {
	Total             float64                   `json:"total"`
	Items             []DietAnalysisProteinItem `json:"items"`
	AnimalBeanPercent float64                   `json:"animal_bean_percent"`
}

// DietAnalysisFatItem 是脂肪来源的一项；其他脂是残差，只进分析图。
type DietAnalysisFatItem struct {
	Key       string  `json:"key"`
	Label     string  `json:"label"`
	Value     float64 `json:"value"`
	Percent   float64 `json:"percent"`
	Displayed bool    `json:"displayed"`
}

// DietAnalysisFatSource 是脂肪来源表（接口 3 项，页面/导出 2 列）。
type DietAnalysisFatSource struct {
	Total float64               `json:"total"`
	Items []DietAnalysisFatItem `json:"items"`
}

// DietAnalysisDriPercent 是结论里的 DRIs 占比（11 项，不含千焦）。
type DietAnalysisDriPercent struct {
	Energy    float64 `json:"energy"`
	Protein   float64 `json:"protein"`
	Fats      float64 `json:"fats"`
	Carb      float64 `json:"carb"`
	VitaminA  float64 `json:"vitamin_a"`
	VitaminB1 float64 `json:"vitamin_b1"`
	VitaminB2 float64 `json:"vitamin_b2"`
	VitaminC  float64 `json:"vitamin_c"`
	Calcium   float64 `json:"calcium"`
	Iron      float64 `json:"iron"`
	Zinc      float64 `json:"zinc"`
}

// DietAnalysisMacroPercent 是结论里的三大营养素供能占比。
type DietAnalysisMacroPercent struct {
	Protein float64 `json:"protein"`
	Fats    float64 `json:"fats"`
	Carb    float64 `json:"carb"`
}

// DietAnalysisSuggestion 是结论里按营养素触发的固定话术。
type DietAnalysisSuggestion struct {
	NutrientKey string `json:"nutrient_key"`
	Level       string `json:"level"` // low / high
	Text        string `json:"text"`
}

// DietAnalysisConclusion 是自动结论的结构化结果。
type DietAnalysisConclusion struct {
	FoodKindCount            int                      `json:"food_kind_count"`
	CondimentKindCount       int                      `json:"condiment_kind_count"`
	NonCondimentKindCount    int                      `json:"non_condiment_kind_count"`
	DriPercent               DietAnalysisDriPercent   `json:"dri_percent"`
	MacroPercent             DietAnalysisMacroPercent `json:"macro_percent"`
	MacroRatio               string                   `json:"macro_ratio"`
	AnimalBeanProteinPercent float64                  `json:"animal_bean_protein_percent"`
	Suggestions              []DietAnalysisSuggestion `json:"suggestions"`
}

// DietAnalysisResult 是 GET /admin/diet/plans/:id/analysis 的完整响应。
type DietAnalysisResult struct {
	Plan          DietAnalysisPlan          `json:"plan"`
	Nutrition     DietAnalysisNutrition     `json:"nutrition"`
	MacroEnergy   DietAnalysisMacroEnergy   `json:"macro_energy"`
	MealEnergy    DietAnalysisMealEnergy    `json:"meal_energy"`
	Composition   DietAnalysisComposition   `json:"composition"`
	CalciumIron   DietAnalysisCalciumIron   `json:"calcium_iron"`
	ProteinSource DietAnalysisProteinSource `json:"protein_source"`
	FatSource     DietAnalysisFatSource     `json:"fat_source"`
	Conclusion    DietAnalysisConclusion    `json:"conclusion"`
}

// DietAnalysisService 营养分析只读服务。
type DietAnalysisService struct {
	plans *repository.DietPlanRepo
	foods *repository.DietFoodRepo
}

func NewDietAnalysisService(
	plans *repository.DietPlanRepo,
	foods *repository.DietFoodRepo,
) *DietAnalysisService {
	return &DietAnalysisService{plans: plans, foods: foods}
}

// Analyze 生成某周食谱的 7 组报表与结论；跨园食谱按 404 处理。
func (s *DietAnalysisService) Analyze(
	ctx context.Context,
	kindergartenID, planID uint64,
) (*DietAnalysisResult, error) {
	if kindergartenID == 0 {
		return nil, dietPlanInvalid("未绑定园所，无法生成营养分析")
	}
	if planID == 0 {
		return nil, ErrDietPlanNotFound
	}
	plan, err := s.plans.FindByIDWithItems(ctx, kindergartenID, planID)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, ErrDietPlanNotFound
	}
	if len(plan.Items) == 0 {
		return nil, ErrDietAnalysisEmpty
	}

	ids := make([]uint64, 0, len(plan.Items))
	seen := make(map[uint64]bool, len(plan.Items))
	for _, item := range plan.Items {
		if item.FoodID == nil || seen[*item.FoodID] {
			continue
		}
		seen[*item.FoodID] = true
		ids = append(ids, *item.FoodID)
	}
	foodRows, err := s.foods.FindByIDs(ctx, kindergartenID, ids)
	if err != nil {
		return nil, err
	}
	foodByID := make(map[uint64]*domain.Food, len(foodRows))
	for i := range foodRows {
		foodByID[foodRows[i].ID] = &foodRows[i]
	}
	return buildDietAnalysisResult(plan, foodByID)
}

// dietAnalysisNutrientSpecs 是营养分析表 12 项的固定顺序与展示文案。
// 顺序与标签必须与契约一致。千焦的 DRI 由能量 × 4.186 推导，快照只存 11 项。
var dietAnalysisNutrientSpecs = []struct {
	key    string
	label  string
	unit   string
	driKey string
}{
	{"energy", "热量", "千卡", "energy"},
	{"heat", "热量", "千焦", "energy"},
	{"protein", "蛋白质", "克", "protein"},
	{"fats", "脂肪", "克", "fats"},
	{"carb", "碳水化合物", "克", "carb"},
	{"vitamin_a", "维生素A", "微克", "vitamin_a"},
	{"vitamin_b1", "维生素B1", "毫克", "vitamin_b1"},
	{"vitamin_b2", "维生素B2", "毫克", "vitamin_b2"},
	{"vitamin_c", "维生素C", "毫克", "vitamin_c"},
	{"calcium", "钙", "毫克", "calcium"},
	{"zinc", "锌", "毫克", "zinc"},
	{"iron", "铁", "毫克", "iron"},
}

// dietAnalysisNutrientKeys 是送进聚合器的营养快照字段（与 specs 对齐）。
var dietAnalysisNutrientKeys = []string{
	"energy", "heat", "protein", "fats", "carb",
	"vitamin_a", "vitamin_b1", "vitamin_b2", "vitamin_c",
	"calcium", "zinc", "iron",
}

// dietAnalysisCompositionOrder 是膳食构成的固定输出顺序；前 10 类页面渲染。
var dietAnalysisCompositionOrder = []struct {
	key       string
	label     string
	displayed bool
}{
	{string(domain.DietGroup16Grain), "谷类", true},
	{string(domain.DietGroup16Tuber), "薯类", true},
	{string(domain.DietGroup16Vegetable), "蔬菜", true},
	{string(domain.DietGroup16Fruit), "水果", true},
	{string(domain.DietGroup16Meat), "畜禽肉类", true},
	{string(domain.DietGroup16Egg), "蛋类", true},
	{string(domain.DietGroup16Aquatic), "水产类", true},
	{string(domain.DietGroup16Soybean), "大豆", true},
	{string(domain.DietGroup16Dairy), "乳制品", true},
	{string(domain.DietGroup16Oil), "油脂类", true},
	{string(domain.DietGroup16Nut), "坚果", false},
	{string(domain.DietGroup16InstantFood), "速食小吃", false},
	{string(domain.DietGroup16Beverage), "饮料", false},
	{string(domain.DietGroup16Confection), "糖果蜜饯", false},
	{string(domain.DietGroup16Condiment), "调味品", false},
	{string(domain.DietGroup16Other), "其他", false},
}

type dietAnalysisAggregate struct {
	nutrients     map[string]float64
	mealEnergy    map[domain.DietMealKey]float64
	composition   map[domain.DietFoodGroup16]float64
	proteinAnimal float64
	proteinBean   float64
	fatAnimal     float64
	fatPlant      float64
	calciumAnimal float64
	calciumPlant  float64
	calciumMilk   float64
	ironMeat      float64
	foodKeys      map[string]struct{}
	condimentKeys map[string]struct{}
}

// buildDietAnalysisResult 是纯函数：不碰仓储，便于用固定 fixture 做黄金回归。
func buildDietAnalysisResult(
	plan *domain.MealPlan,
	foods map[uint64]*domain.Food,
) (*DietAnalysisResult, error) {
	if plan == nil {
		return nil, ErrDietPlanNotFound
	}
	if len(plan.Items) == 0 {
		return nil, ErrDietAnalysisEmpty
	}

	configSnapshot, err := parseDietConfigSnapshot(plan.ConfigSnapshot)
	if err != nil {
		return nil, err
	}
	dayNum := plan.DayNum
	if dayNum <= 0 {
		dayNum = countDietPlanDays(plan.Items)
	}
	if dayNum <= 0 {
		return nil, ErrDietAnalysisEmpty
	}
	coefficient := configSnapshot.Coefficient
	if coefficient <= 0 {
		coefficient = dietAnalysisDefaultCoefficient
	}

	aggregate, err := aggregateDietAnalysisItems(plan.Items, foods)
	if err != nil {
		return nil, err
	}
	targets := make(map[string]float64, len(configSnapshot.DriTargets))
	for _, target := range configSnapshot.DriTargets {
		targets[target.NutrientKey] = target.Value
	}

	normalize := func(sum float64) float64 {
		return sum / float64(dayNum) / coefficient
	}

	result := &DietAnalysisResult{
		Plan: DietAnalysisPlan{
			ID:          plan.ID,
			Name:        plan.Name,
			StartDate:   plan.StartDate.Format(dietPlanDateLayout),
			EndDate:     plan.EndDate.Format(dietPlanDateLayout),
			DayNum:      dayNum,
			Coefficient: coefficient,
		},
	}

	compareByKey := make(map[string]float64, len(dietAnalysisNutrientSpecs))
	for _, spec := range dietAnalysisNutrientSpecs {
		intakeRaw := normalize(aggregate.nutrients[spec.key])
		dri := targets[spec.driKey]
		if spec.key == "heat" {
			dri = dietAnalysisRound2(targets["energy"] * dietAnalysisHeatFactor)
		}
		percent := 0.0
		if dri > 0 {
			percent = intakeRaw / dri * 100
		}
		percent = dietAnalysisRound2(percent)
		compareByKey[spec.key] = percent
		result.Nutrition.Items = append(result.Nutrition.Items, DietAnalysisNutritionItem{
			Key:            spec.key,
			Label:          spec.label,
			Unit:           spec.unit,
			Intake:         dietAnalysisRound2(intakeRaw),
			Dri:            dietAnalysisRound2(dri),
			ComparePercent: percent,
			Comment:        dietAnalysisComment(percent),
		})
	}

	result.MacroEnergy = buildDietAnalysisMacroEnergy(aggregate, targets, normalize)
	result.MealEnergy = buildDietAnalysisMealEnergy(aggregate, configSnapshot, targets, dayNum)
	result.Composition = buildDietAnalysisComposition(aggregate, normalize)
	result.CalciumIron = buildDietAnalysisCalciumIron(aggregate, normalize)
	result.ProteinSource = buildDietAnalysisProteinSource(aggregate, normalize)
	result.FatSource = buildDietAnalysisFatSource(aggregate, normalize)
	result.Conclusion = buildDietAnalysisConclusion(dietAnalysisConclusionInput{
		foodKindCount:      len(aggregate.foodKeys),
		condimentKindCount: len(aggregate.condimentKeys),
		compareByKey:       compareByKey,
		macroEnergy:        result.MacroEnergy,
		proteinSource:      result.ProteinSource,
		comments:           nutritionComments(result.Nutrition.Items),
	})
	return result, nil
}

func aggregateDietAnalysisItems(
	items []domain.MealPlanItem,
	foods map[uint64]*domain.Food,
) (*dietAnalysisAggregate, error) {
	aggregate := &dietAnalysisAggregate{
		nutrients:     make(map[string]float64, len(dietAnalysisNutrientKeys)),
		mealEnergy:    make(map[domain.DietMealKey]float64, len(domain.DietMealKeyOrder)),
		composition:   make(map[domain.DietFoodGroup16]float64, len(dietAnalysisCompositionOrder)),
		foodKeys:      make(map[string]struct{}, len(items)),
		condimentKeys: make(map[string]struct{}),
	}
	for i := range items {
		item := &items[i]
		snapshot, err := parseDietNutritionSnapshot(item.NutritionSnapshot)
		if err != nil {
			return nil, err
		}
		edibleGrams := item.Grams * snapshot.EdiblePercent / 100

		amounts := map[string]*float64{
			"energy":     dietNutritionAmount(item.Grams, snapshot.EdiblePercent, snapshot.Energy),
			"heat":       dietNutritionAmount(item.Grams, snapshot.EdiblePercent, snapshot.Heat),
			"protein":    dietNutritionAmount(item.Grams, snapshot.EdiblePercent, snapshot.Protein),
			"fats":       dietNutritionAmount(item.Grams, snapshot.EdiblePercent, snapshot.Fats),
			"carb":       dietNutritionAmount(item.Grams, snapshot.EdiblePercent, snapshot.Carb),
			"vitamin_a":  dietNutritionAmount(item.Grams, snapshot.EdiblePercent, snapshot.VitaminA),
			"vitamin_b1": dietNutritionAmount(item.Grams, snapshot.EdiblePercent, snapshot.VitaminB1),
			"vitamin_b2": dietNutritionAmount(item.Grams, snapshot.EdiblePercent, snapshot.VitaminB2),
			"vitamin_c":  dietNutritionAmount(item.Grams, snapshot.EdiblePercent, snapshot.VitaminC),
			"calcium":    dietNutritionAmount(item.Grams, snapshot.EdiblePercent, snapshot.Calcium),
			"zinc":       dietNutritionAmount(item.Grams, snapshot.EdiblePercent, snapshot.Zinc),
			"iron":       dietNutritionAmount(item.Grams, snapshot.EdiblePercent, snapshot.Iron),
		}
		for _, key := range dietAnalysisNutrientKeys {
			if amount := amounts[key]; amount != nil {
				aggregate.nutrients[key] += *amount
			}
		}
		if energy := amounts["energy"]; energy != nil {
			aggregate.mealEnergy[item.MealKey] += *energy
		}

		// 分类优先读明细快照：事后改食物库分类不得改写历史报表。
		// 旧行快照没有这两个字段，空值才回退当前食物库。
		group := snapshot.Group16
		flags := snapshot.SourceFlags
		if group == "" || flags == "" {
			var food *domain.Food
			if item.FoodID != nil {
				food = foods[*item.FoodID]
			}
			if food != nil {
				if group == "" {
					group = food.Group16
				}
				if flags == "" {
					flags = food.SourceFlags
				}
			}
		}
		if group == "" {
			group = domain.DietGroup16Other
		}
		aggregate.composition[group] += edibleGrams
		foodKey := dietAnalysisFoodKey(item)
		aggregate.foodKeys[foodKey] = struct{}{}
		if group == domain.DietGroup16Condiment {
			aggregate.condimentKeys[foodKey] = struct{}{}
		}

		protein := dietAnalysisAmountValue(amounts["protein"])
		fats := dietAnalysisAmountValue(amounts["fats"])
		calcium := dietAnalysisAmountValue(amounts["calcium"])
		iron := dietAnalysisAmountValue(amounts["iron"])
		if dietFoodHasFlag(flags, domain.DietSourceFlagAnimal) {
			aggregate.proteinAnimal += protein
			aggregate.fatAnimal += fats
			aggregate.calciumAnimal += calcium
		}
		if dietFoodHasFlag(flags, domain.DietSourceFlagPlant) {
			aggregate.fatPlant += fats
			aggregate.calciumPlant += calcium
		}
		if dietFoodHasFlag(flags, domain.DietSourceFlagMilk) {
			aggregate.calciumMilk += calcium
		}
		if dietFoodHasFlag(flags, domain.DietSourceFlagMeat) {
			aggregate.ironMeat += iron
		}
		if group == domain.DietGroup16Soybean {
			aggregate.proteinBean += protein
		}
	}
	return aggregate, nil
}

// buildDietAnalysisMacroEnergy 复刻参考产品的四行结构。
// 前两行是 DRIs 派生的「要求」值（旧产品如此：示例 protein 202.8 = 50.7 × 4），
// 占总热量% 与营养素比例才是「现状」派生值；分母 = 现状蛋白×4 + 脂肪×9 + 碳水×4。
func buildDietAnalysisMacroEnergy(
	aggregate *dietAnalysisAggregate,
	targets map[string]float64,
	normalize func(float64) float64,
) DietAnalysisMacroEnergy {
	protein := dietAnalysisRound2(normalize(aggregate.nutrients["protein"]))
	fats := dietAnalysisRound2(normalize(aggregate.nutrients["fats"]))
	carb := dietAnalysisRound2(normalize(aggregate.nutrients["carb"]))

	proteinKcal := dietAnalysisRound2(protein * dietAnalysisProteinKcal)
	fatsKcal := dietAnalysisRound2(fats * dietAnalysisFatsKcal)
	carbKcal := dietAnalysisRound2(carb * dietAnalysisCarbKcal)
	denominator := dietAnalysisRound2(proteinKcal + fatsKcal + carbKcal)

	recommendedProtein := dietAnalysisRound2(targets["protein"] * dietAnalysisProteinKcal)
	recommendedFats := dietAnalysisRound2(targets["fats"] * dietAnalysisFatsKcal)
	recommendedCarb := dietAnalysisRound2(targets["carb"] * dietAnalysisCarbKcal)

	percent := func(value float64) float64 {
		if denominator <= 0 {
			return 0
		}
		return dietAnalysisRound2(value / denominator * 100)
	}
	ratio := func(value float64) float64 {
		if protein <= 0 {
			return 0
		}
		return dietAnalysisRound2(value / protein)
	}

	return DietAnalysisMacroEnergy{
		Denominator: denominator,
		Rows: []DietAnalysisMacroRow{
			{
				Key: "kcal", Label: "千卡",
				Protein: recommendedProtein, Fats: recommendedFats, Carb: recommendedCarb,
			},
			{
				Key: "kj", Label: "千焦",
				Protein: dietAnalysisRound2(recommendedProtein * dietAnalysisHeatFactor),
				Fats:    dietAnalysisRound2(recommendedFats * dietAnalysisHeatFactor),
				Carb:    dietAnalysisRound2(recommendedCarb * dietAnalysisHeatFactor),
			},
			{
				Key: "percent", Label: "占总热量%",
				Protein: percent(proteinKcal), Fats: percent(fatsKcal), Carb: percent(carbKcal),
				Recommended: &DietAnalysisMacroRecommended{
					Protein: "12-15%", Fats: "30-35%", Carb: "50-60%",
				},
			},
			{
				Key: "grams_ratio", Label: "营养素比例",
				Protein: 1, Fats: ratio(fats), Carb: ratio(carb),
				Recommended: &DietAnalysisMacroRecommended{
					Protein: "1", Fats: "1", Carb: "4-5",
				},
			},
		},
	}
}

// buildDietAnalysisMealEnergy 各餐能量只除 dayNum（不除系数），比例再除全日 DRIs。
// 因此比例合计不等于 100%（参考产品 85.88%），这不是 bug。
func buildDietAnalysisMealEnergy(
	aggregate *dietAnalysisAggregate,
	configSnapshot domain.DietConfigSnapshot,
	targets map[string]float64,
	dayNum int,
) DietAnalysisMealEnergy {
	labels := make(map[domain.DietMealKey]string, len(configSnapshot.Meals))
	for _, meal := range configSnapshot.Meals {
		if strings.TrimSpace(meal.Name) != "" {
			labels[meal.MealKey] = meal.Name
		}
	}
	energyDri := targets["energy"]
	totalPercent := 0.0
	meals := make([]DietAnalysisMeal, 0, len(domain.DietMealKeyOrder))
	for _, mealKey := range domain.DietMealKeyOrder {
		energy := aggregate.mealEnergy[mealKey] / float64(dayNum)
		percent := 0.0
		if energyDri > 0 {
			percent = energy / energyDri * 100
		}
		totalPercent += percent
		label := labels[mealKey]
		if label == "" {
			label = dietMealName(mealKey)
		}
		meals = append(meals, DietAnalysisMeal{
			MealKey: string(mealKey),
			Label:   label,
			Energy:  dietAnalysisRound2(energy),
			Percent: dietAnalysisRound2(percent),
		})
	}
	return DietAnalysisMealEnergy{
		Meals:        meals,
		TotalPercent: dietAnalysisRound2(totalPercent),
	}
}

// buildDietAnalysisComposition 按 group16 汇总可食部克数；16 类全部返回，
// displayed 只标记页面实际渲染的 10 类。
func buildDietAnalysisComposition(
	aggregate *dietAnalysisAggregate,
	normalize func(float64) float64,
) DietAnalysisComposition {
	groups := make([]DietAnalysisCompositionGroup, 0, len(dietAnalysisCompositionOrder))
	for _, spec := range dietAnalysisCompositionOrder {
		groups = append(groups, DietAnalysisCompositionGroup{
			Key:       spec.key,
			Label:     spec.label,
			Grams:     dietAnalysisRound2(normalize(aggregate.composition[domain.DietFoodGroup16(spec.key)])),
			Displayed: spec.displayed,
		})
	}
	return DietAnalysisComposition{Groups: groups}
}

// buildDietAnalysisCalciumIron 复刻旧产品口径：
// 钙的动物性含乳类、乳类又单列一次（三来源合计 126%）；
// 铁只认畜肉+禽肉为肉类，其余全部归非肉类（两来源互斥合计 100%）。
func buildDietAnalysisCalciumIron(
	aggregate *dietAnalysisAggregate,
	normalize func(float64) float64,
) DietAnalysisCalciumIron {
	calciumTotal := normalize(aggregate.nutrients["calcium"])
	ironTotal := normalize(aggregate.nutrients["iron"])

	source := func(key, label string, raw, total float64) DietAnalysisSource {
		return DietAnalysisSource{
			Key:     key,
			Label:   label,
			Value:   dietAnalysisRound2(raw),
			Percent: dietAnalysisPercent(raw, total),
		}
	}

	calciumAnimal := normalize(aggregate.calciumAnimal)
	calciumPlant := normalize(aggregate.calciumPlant)
	calciumMilk := normalize(aggregate.calciumMilk)
	ironMeat := normalize(aggregate.ironMeat)
	ironNonMeat := ironTotal - ironMeat

	return DietAnalysisCalciumIron{
		Calcium: DietAnalysisCalciumIronSide{
			Total: dietAnalysisRound2(calciumTotal),
			Sources: []DietAnalysisSource{
				source("animal", "动物性", calciumAnimal, calciumTotal),
				source("plant", "植物性", calciumPlant, calciumTotal),
				source("milk", "乳类", calciumMilk, calciumTotal),
			},
		},
		Iron: DietAnalysisCalciumIronSide{
			Total: dietAnalysisRound2(ironTotal),
			Sources: []DietAnalysisSource{
				source("meat", "肉类", ironMeat, ironTotal),
				source("non_meat", "非肉类", ironNonMeat, ironTotal),
			},
		},
	}
}

// buildDietAnalysisProteinSource 只统计动物蛋白与豆类蛋白，三项合计不等于总蛋白。
func buildDietAnalysisProteinSource(
	aggregate *dietAnalysisAggregate,
	normalize func(float64) float64,
) DietAnalysisProteinSource {
	total := normalize(aggregate.nutrients["protein"])
	animal := normalize(aggregate.proteinAnimal)
	bean := normalize(aggregate.proteinBean)
	required := total * 0.5

	item := func(key, label string, value float64) DietAnalysisProteinItem {
		return DietAnalysisProteinItem{
			Key:     key,
			Label:   label,
			Value:   dietAnalysisRound2(value),
			Percent: dietAnalysisPercent(value, total),
		}
	}
	return DietAnalysisProteinSource{
		Total: dietAnalysisRound2(total),
		Items: []DietAnalysisProteinItem{
			item("animal", "动物蛋白", animal),
			item("bean", "豆类蛋白", bean),
			item("required", "要求", required),
		},
		AnimalBeanPercent: dietAnalysisPercent(animal+bean, total),
	}
}

// buildDietAnalysisFatSource 动物性/植物性按标记求和，其他脂是残差（含未归类食物）。
func buildDietAnalysisFatSource(
	aggregate *dietAnalysisAggregate,
	normalize func(float64) float64,
) DietAnalysisFatSource {
	total := normalize(aggregate.nutrients["fats"])
	animal := normalize(aggregate.fatAnimal)
	plant := normalize(aggregate.fatPlant)
	other := total - animal - plant

	item := func(key, label string, value float64, displayed bool) DietAnalysisFatItem {
		return DietAnalysisFatItem{
			Key:       key,
			Label:     label,
			Value:     dietAnalysisRound2(value),
			Percent:   dietAnalysisPercent(value, total),
			Displayed: displayed,
		}
	}
	return DietAnalysisFatSource{
		Total: dietAnalysisRound2(total),
		Items: []DietAnalysisFatItem{
			item("animal", "动物性", animal, true),
			item("plant", "植物性", plant, true),
			item("other", "其他", other, false),
		},
	}
}

func dietAnalysisAmountValue(amount *float64) float64 {
	if amount == nil {
		return 0
	}
	return *amount
}

// dietAnalysisFoodKey 用食物 ID 判重；没有 ID 的历史行退回菜名。
func dietAnalysisFoodKey(item *domain.MealPlanItem) string {
	if item.FoodID != nil {
		return "id:" + strconv.FormatUint(*item.FoodID, 10)
	}
	return "name:" + strings.TrimSpace(item.FoodName)
}

// dietFoodHasFlag 判断逗号分隔的来源标记（animal/plant/milk/meat）。
func dietFoodHasFlag(flags, want string) bool {
	if flags == "" {
		return false
	}
	for _, flag := range strings.Split(flags, ",") {
		if strings.TrimSpace(flag) == want {
			return true
		}
	}
	return false
}

func nutritionComments(items []DietAnalysisNutritionItem) map[string]string {
	comments := make(map[string]string, len(items))
	for _, item := range items {
		comments[item.Key] = item.Comment
	}
	return comments
}

// dietAnalysisComment 使用 ±20% 闭区间评价；入参是已按 2 位小数四舍五入的比较百分比。
func dietAnalysisComment(percent float64) string {
	switch {
	case percent < 80:
		return "低"
	case percent > 120:
		return "高"
	default:
		return "合格"
	}
}

func dietAnalysisPercent(value, total float64) float64 {
	if total <= 0 {
		return 0
	}
	return dietAnalysisRound2(value / total * 100)
}

func dietAnalysisRound2(v float64) float64 {
	return math.Round(v*100) / 100
}
