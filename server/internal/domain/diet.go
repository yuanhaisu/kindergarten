package domain

// 膳食（带量食谱）模块的枚举与快照类型（openspec: add-quantified-recipe）。
//
// 餐次用显式字符串而不是参考产品的位掩码：位掩码无法表达可读的餐次顺序，
// 也不便作为唯一键与快照键；参考产品的 1/2/4/8/16/32 只是其历史实现（design D2/D3）。

// DietMealKey 餐次键。
type DietMealKey string

const (
	DietMealBreakfast      DietMealKey = "breakfast"       // 早餐
	DietMealMorningSnack   DietMealKey = "morning_snack"   // 早点
	DietMealLunch          DietMealKey = "lunch"           // 午餐
	DietMealAfternoonSnack DietMealKey = "afternoon_snack" // 午点
	DietMealDinner         DietMealKey = "dinner"          // 晚餐
	DietMealEveningSnack   DietMealKey = "evening_snack"   // 晚点
)

// DietMealKeyOrder 是餐次的固定先后顺序，同时用于展示排序与「时间必须递增」校验。
var DietMealKeyOrder = []DietMealKey{
	DietMealBreakfast,
	DietMealMorningSnack,
	DietMealLunch,
	DietMealAfternoonSnack,
	DietMealDinner,
	DietMealEveningSnack,
}

// DietMealSort 返回餐次的固定序号（从 1 开始）；未知键返回 0。
// 用固定序号而不是请求里的 sort：餐次顺序是业务约束，不能由客户端重排。
func DietMealSort(key DietMealKey) int {
	for i, k := range DietMealKeyOrder {
		if k == key {
			return i + 1
		}
	}
	return 0
}

// DietCareType 托管形式。参考产品用 1/2 表示，这里用字符串避免位掩码式的隐式语义。
type DietCareType string

const (
	DietCareTypeDayCare  DietCareType = "day_care"  // 日托
	DietCareTypeFullCare DietCareType = "full_care" // 全托
)

// DietFoodInputType 食物重量输入口径。本变更只支持市品重量（design D3、TC-NEW-DIE-0007）。
type DietFoodInputType string

const (
	DietFoodInputTypeMarketWeight DietFoodInputType = "market_weight" // 市品重量（唯一合法值）
)

// DietPlanCreatedMethod 周食谱创建方式。
type DietPlanCreatedMethod string

const (
	DietPlanCreatedByCustom   DietPlanCreatedMethod = "custom"   // 自定义
	DietPlanCreatedByTemplate DietPlanCreatedMethod = "template" // 选用模板
	DietPlanCreatedByCopy     DietPlanCreatedMethod = "copy"     // 复制历史
)

// DietFoodSource 食物来源：平台预置或园所自建。
// 「园所是否改过预置行」是另一件事，见 Food.IsCustomized（不对外暴露）。
type DietFoodSource string

const (
	DietFoodSourceSystem DietFoodSource = "system" // 平台预置
	DietFoodSourceCustom DietFoodSource = "custom" // 园所自建
)

// DietPlatformKindergartenID 是平台预置食物行的 kindergarten_id 哨兵值。
// 租户查询必须带 kindergarten_id = <本园>，不得把 kg=0 行回传给园所（design D2）。
const DietPlatformKindergartenID uint64 = 0

// DietFoodGroup12 沿用参考产品 12 个「采购/构成」分类的 ID（37–48）。
// 映射由 22,809 行明细确定性重建（design D2）；37=1谷、38=10豆、39=6+7肉、
// 40=8蛋、41=12乳、42=3+4蔬菌、43=13+14小吃速食、44=5果、45=19调味品、
// 46=9水产、47=18植物油、48=11坚果。薯类/饮料/糖蜜饯无槽位 → NULL。
const (
	DietGroup12Grain     = 37 // 谷类
	DietGroup12Bean      = 38 // 豆类
	DietGroup12Meat      = 39 // 肉类
	DietGroup12Egg       = 40 // 蛋类
	DietGroup12Dairy     = 41 // 奶类
	DietGroup12Vegetable = 42 // 蔬菜类
	DietGroup12Pastry    = 43 // 糕点类（13 小吃甜饼 + 14 速食食品）
	DietGroup12Fruit     = 44 // 水果类
	DietGroup12Condiment = 45 // 调味品
	DietGroup12Seafood   = 46 // 海鲜
	DietGroup12Oil       = 47 // 油
	DietGroup12Nut       = 48 // 坚果
)

// DietFoodGroup16 膳食构成表的 16 行分类（20 个一级类做 4 次合并）。
// 报告页只渲染其中 10 类，接口仍返回 16 类（design D6）。
type DietFoodGroup16 string

const (
	DietGroup16Grain       DietFoodGroup16 = "grain"        // 谷类 ← 1
	DietGroup16Tuber       DietFoodGroup16 = "tuber"        // 薯类 ← 2
	DietGroup16Vegetable   DietFoodGroup16 = "vegetable"    // 蔬菜 ← 3 + 4 菌藻
	DietGroup16Fruit       DietFoodGroup16 = "fruit"        // 水果 ← 5
	DietGroup16Meat        DietFoodGroup16 = "meat"         // 畜禽肉类 ← 6 + 7
	DietGroup16Egg         DietFoodGroup16 = "egg"          // 蛋类 ← 8
	DietGroup16Aquatic     DietFoodGroup16 = "aquatic"      // 水产类 ← 9
	DietGroup16Soybean     DietFoodGroup16 = "soybean"      // 大豆 ← 10
	DietGroup16Nut         DietFoodGroup16 = "nut"          // 坚果 ← 11
	DietGroup16Dairy       DietFoodGroup16 = "dairy"        // 乳制品 ← 12
	DietGroup16InstantFood DietFoodGroup16 = "instant_food" // 速食小吃 ← 13 + 14
	DietGroup16Beverage    DietFoodGroup16 = "beverage"     // 饮料 ← 15
	DietGroup16Confection  DietFoodGroup16 = "confection"   // 糖果蜜饯 ← 16
	DietGroup16Oil         DietFoodGroup16 = "oil"          // 油脂类 ← 17 + 18
	DietGroup16Condiment   DietFoodGroup16 = "condiment"    // 调味品 ← 19
	DietGroup16Other       DietFoodGroup16 = "other"        // 其他 ← 20
)

// 食物属性标记（Food.SourceFlags，逗号分隔存储）。
// 用于蛋白质/脂肪/钙铁来源报表的集合运算，避免把 20 个一级类的合并规则散落在报表代码里。
const (
	DietSourceFlagAnimal = "animal" // 动物性
	DietSourceFlagPlant  = "plant"  // 植物性
	DietSourceFlagMilk   = "milk"   // 乳类（钙来源单列，且仍计入动物性）
	DietSourceFlagMeat   = "meat"   // 肉类（铁来源只认畜肉+禽肉）
)

// DietMealSnapshot 是供餐配置快照中的单个餐次。
// 与 MealConfigMeal 的差别：不含 ID/租户字段，只保留报表与折算需要的内容。
type DietMealSnapshot struct {
	MealKey       DietMealKey `json:"meal_key"`
	Name          string      `json:"name"`
	Time          string      `json:"time"` // HH:mm
	EnergyPercent float64     `json:"energy_percent"`
	Sort          int         `json:"sort"`
	Enabled       bool        `json:"enabled"`
}

// DietDriTargetSnapshot 是 11 项 DRIs 目标值中的一项（按班级人数加权后的结果）。
type DietDriTargetSnapshot struct {
	NutrientKey string  `json:"nutrient_key"`
	Label       string  `json:"label"`
	Unit        string  `json:"unit"`
	Value       float64 `json:"value"`
}

// DietDriWeight 是 3/4/5/6 岁档在加权中占的权重，随快照留档以便复核目标值。
type DietDriWeight struct {
	AgeBand int     `json:"age_band"` // 3/4/5/6
	Weight  float64 `json:"weight"`   // 0–1
}

// DietConfigSnapshot 是周食谱创建时固化的供餐配置快照（MealPlan.config_snapshot）。
//
// 快照一旦写入，后续修改供餐配置或学生名册都不得回溯历史食谱（design D3、TC-NEW-DIE-0008）。
// Coefficient = 启用餐次占比之和（默认 0.95），报表在分子侧除以它、在基准侧乘以它。
type DietConfigSnapshot struct {
	CareType      DietCareType            `json:"care_type"`
	FoodInputType DietFoodInputType       `json:"food_input_type"`
	Coefficient   float64                 `json:"coefficient"`
	Meals         []DietMealSnapshot      `json:"meals"`
	DriTargets    []DietDriTargetSnapshot `json:"dri_targets"`
	DriWeights    []DietDriWeight         `json:"dri_weights"`
}

// DietNutritionSnapshot 是周食谱明细在录入时固化的营养浓度快照（ADR-0006）。
// 全部用指针：NULL = 未测定/不适用，报表求和必须跳过，不能当 0。
type DietNutritionSnapshot struct {
	EdiblePercent float64  `json:"edible_percent"`
	Energy        *float64 `json:"energy"`         // 千卡
	Heat          *float64 `json:"heat"`           // 千焦 = 千卡 × 4.186
	Protein       *float64 `json:"protein"`        // 克
	Fats          *float64 `json:"fats"`           // 克
	Carb          *float64 `json:"carb"`           // 克
	DietaryFiber  *float64 `json:"dietary_fiber"`  // 克
	Ash           *float64 `json:"ash"`            // 克
	Water         *float64 `json:"water"`          // 克
	Cholesterol   *float64 `json:"cholesterol"`    // 毫克
	Retinol       *float64 `json:"retinol"`        // 微克
	Carotin       *float64 `json:"carotin"`        // 微克
	VitaminA      *float64 `json:"vitamin_a"`      // 微克 RE
	VitaminB1     *float64 `json:"vitamin_b1"`     // 毫克
	VitaminB2     *float64 `json:"vitamin_b2"`     // 毫克
	NicotinicAcid *float64 `json:"nicotinic_acid"` // 毫克
	VitaminC      *float64 `json:"vitamin_c"`      // 毫克
	VitaminE      *float64 `json:"vitamin_e"`      // 毫克 α-TE
	Calcium       *float64 `json:"calcium"`        // 毫克
	Phosphorus    *float64 `json:"phosphorus"`     // 毫克
	Potassium     *float64 `json:"potassium"`      // 毫克
	Sodium        *float64 `json:"sodium"`         // 毫克
	Magnesium     *float64 `json:"magnesium"`      // 毫克
	Iron          *float64 `json:"iron"`           // 毫克
	Zinc          *float64 `json:"zinc"`           // 毫克
	Selenium      *float64 `json:"selenium"`       // 微克
	Copper        *float64 `json:"copper"`         // 毫克
	Manganese     *float64 `json:"manganese"`      // 毫克
}
