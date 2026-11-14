package service

import "kindergarten-server/internal/domain"

// 平台初稿周食谱模板。模板条目按食物「名称」引用本园食物库：授权 CSV 尚未导入、
// 或食物名与初稿不一致时，复制流程会跳过找不到的条目并汇总告警（design D10）。
//
// 内容为初稿，上线前必须由园所营养师审核。刻意只做 5 个工作日 × 早餐/午餐/晚餐、
// 每餐 1 道菜、每道菜 1–2 种食物，与 design D10 的模板规则一致。
type dietTemplateSpec struct {
	Key  string
	Name string
	Sort int
	Days []dietTemplateDay
}

type dietTemplateDay struct {
	Weekday int // 1=周一 … 5=周五
	Meals   []dietTemplateMeal
}

type dietTemplateMeal struct {
	MealKey    domain.DietMealKey
	RecipeName string
	Foods      []dietTemplateFood
}

type dietTemplateFood struct {
	Name  string
	Grams float64
}

// dietTemplateMealKeys 是初稿模板覆盖的餐次集合（TemplatePlan.meal_keys）。
const dietTemplateMealKeys = "breakfast,lunch,dinner"

func dietTemplateSpecs() []dietTemplateSpec {
	return []dietTemplateSpec{
		{
			Key:  "balanced_week",
			Name: "均衡周",
			Sort: 1,
			Days: []dietTemplateDay{
				{Weekday: 1, Meals: []dietTemplateMeal{
					{MealKey: domain.DietMealBreakfast, RecipeName: "牛奶燕麦粥", Foods: []dietTemplateFood{{Name: "牛奶", Grams: 200}, {Name: "燕麦片", Grams: 30}}},
					{MealKey: domain.DietMealLunch, RecipeName: "番茄鸡蛋面", Foods: []dietTemplateFood{{Name: "面条", Grams: 60}, {Name: "鸡蛋", Grams: 50}}},
					{MealKey: domain.DietMealDinner, RecipeName: "米饭蒸鱼", Foods: []dietTemplateFood{{Name: "稻米", Grams: 55}, {Name: "鲈鱼", Grams: 40}}},
				}},
				{Weekday: 2, Meals: []dietTemplateMeal{
					{MealKey: domain.DietMealBreakfast, RecipeName: "小米粥", Foods: []dietTemplateFood{{Name: "小米", Grams: 30}}},
					{MealKey: domain.DietMealLunch, RecipeName: "猪肉白菜饺子", Foods: []dietTemplateFood{{Name: "面粉", Grams: 55}, {Name: "猪肉", Grams: 35}}},
					{MealKey: domain.DietMealDinner, RecipeName: "豆腐米饭", Foods: []dietTemplateFood{{Name: "稻米", Grams: 55}, {Name: "豆腐", Grams: 50}}},
				}},
				{Weekday: 3, Meals: []dietTemplateMeal{
					{MealKey: domain.DietMealBreakfast, RecipeName: "鸡蛋面", Foods: []dietTemplateFood{{Name: "面条", Grams: 50}, {Name: "鸡蛋", Grams: 50}}},
					{MealKey: domain.DietMealLunch, RecipeName: "米饭鸡丁", Foods: []dietTemplateFood{{Name: "稻米", Grams: 55}, {Name: "鸡胸肉", Grams: 40}}},
					{MealKey: domain.DietMealDinner, RecipeName: "南瓜小米粥", Foods: []dietTemplateFood{{Name: "小米", Grams: 30}, {Name: "南瓜", Grams: 50}}},
				}},
				{Weekday: 4, Meals: []dietTemplateMeal{
					{MealKey: domain.DietMealBreakfast, RecipeName: "豆浆馒头", Foods: []dietTemplateFood{{Name: "馒头", Grams: 50}, {Name: "豆浆", Grams: 200}}},
					{MealKey: domain.DietMealLunch, RecipeName: "牛肉面", Foods: []dietTemplateFood{{Name: "面条", Grams: 60}, {Name: "牛肉", Grams: 35}}},
					{MealKey: domain.DietMealDinner, RecipeName: "米饭青菜", Foods: []dietTemplateFood{{Name: "稻米", Grams: 55}, {Name: "小白菜", Grams: 80}}},
				}},
				{Weekday: 5, Meals: []dietTemplateMeal{
					{MealKey: domain.DietMealBreakfast, RecipeName: "牛奶面包", Foods: []dietTemplateFood{{Name: "面包", Grams: 50}, {Name: "牛奶", Grams: 200}}},
					{MealKey: domain.DietMealLunch, RecipeName: "米饭虾仁", Foods: []dietTemplateFood{{Name: "稻米", Grams: 55}, {Name: "虾仁", Grams: 40}}},
					{MealKey: domain.DietMealDinner, RecipeName: "菠菜粥", Foods: []dietTemplateFood{{Name: "稻米", Grams: 40}, {Name: "菠菜", Grams: 60}}},
				}},
			},
		},
		{
			Key:  "light_week",
			Name: "清淡周",
			Sort: 2,
			Days: []dietTemplateDay{
				{Weekday: 1, Meals: []dietTemplateMeal{
					{MealKey: domain.DietMealBreakfast, RecipeName: "白粥", Foods: []dietTemplateFood{{Name: "稻米", Grams: 30}}},
					{MealKey: domain.DietMealLunch, RecipeName: "青菜鸡蛋面", Foods: []dietTemplateFood{{Name: "面条", Grams: 55}, {Name: "鸡蛋", Grams: 40}}},
					{MealKey: domain.DietMealDinner, RecipeName: "清蒸鱼米饭", Foods: []dietTemplateFood{{Name: "稻米", Grams: 45}, {Name: "鲈鱼", Grams: 45}}},
				}},
				{Weekday: 2, Meals: []dietTemplateMeal{
					{MealKey: domain.DietMealBreakfast, RecipeName: "牛奶馒头", Foods: []dietTemplateFood{{Name: "馒头", Grams: 50}, {Name: "牛奶", Grams: 200}}},
					{MealKey: domain.DietMealLunch, RecipeName: "番茄豆腐面", Foods: []dietTemplateFood{{Name: "面条", Grams: 55}, {Name: "豆腐", Grams: 60}}},
					{MealKey: domain.DietMealDinner, RecipeName: "小米粥时蔬", Foods: []dietTemplateFood{{Name: "小米", Grams: 30}, {Name: "小白菜", Grams: 80}}},
				}},
				{Weekday: 3, Meals: []dietTemplateMeal{
					{MealKey: domain.DietMealBreakfast, RecipeName: "燕麦粥", Foods: []dietTemplateFood{{Name: "燕麦片", Grams: 30}}},
					{MealKey: domain.DietMealLunch, RecipeName: "香菇鸡肉面", Foods: []dietTemplateFood{{Name: "面条", Grams: 55}, {Name: "鸡胸肉", Grams: 40}}},
					{MealKey: domain.DietMealDinner, RecipeName: "米饭蒸蛋", Foods: []dietTemplateFood{{Name: "稻米", Grams: 50}, {Name: "鸡蛋", Grams: 50}}},
				}},
				{Weekday: 4, Meals: []dietTemplateMeal{
					{MealKey: domain.DietMealBreakfast, RecipeName: "豆浆花卷", Foods: []dietTemplateFood{{Name: "花卷", Grams: 50}, {Name: "豆浆", Grams: 200}}},
					{MealKey: domain.DietMealLunch, RecipeName: "冬瓜瘦肉粥", Foods: []dietTemplateFood{{Name: "稻米", Grams: 45}, {Name: "猪肉", Grams: 30}}},
					{MealKey: domain.DietMealDinner, RecipeName: "紫菜蛋花面", Foods: []dietTemplateFood{{Name: "面条", Grams: 55}, {Name: "鸡蛋", Grams: 40}}},
				}},
				{Weekday: 5, Meals: []dietTemplateMeal{
					{MealKey: domain.DietMealBreakfast, RecipeName: "牛奶小米粥", Foods: []dietTemplateFood{{Name: "小米", Grams: 25}, {Name: "牛奶", Grams: 180}}},
					{MealKey: domain.DietMealLunch, RecipeName: "清蒸鲈鱼米饭", Foods: []dietTemplateFood{{Name: "稻米", Grams: 50}, {Name: "鲈鱼", Grams: 45}}},
					{MealKey: domain.DietMealDinner, RecipeName: "蔬菜豆腐羹", Foods: []dietTemplateFood{{Name: "豆腐", Grams: 60}, {Name: "胡萝卜", Grams: 50}}},
				}},
			},
		},
	}
}
