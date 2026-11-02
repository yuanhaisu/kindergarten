package service

import (
	"fmt"
	"math"
	"regexp"

	"kindergarten-server/internal/domain"
)

// 本文件是供餐配置的公共校验与展示文案（openspec add-diet-group-objects design D8）。
//
// 这些函数不持有状态，由集体对象服务（DietGroupObjectService）与营养分析共用：
// 餐次时间、供能占比、比例模式与重量口径的规则只有这一份，错误文案因此不会因
// 旧园所单例配置服务下线而漂移（TC-NEW-DIE-0003..0007）。
// 单例的 DietConfigService 已被按学年的集体对象取代（design D1/D2）。

// DietConfigValidationError 表示供餐配置未通过业务校验。
// handler 用 errors.As 识别并映射为项目统一的 422 包络；Message 可直接展示给管理员。
type DietConfigValidationError struct {
	Message string
}

func (e *DietConfigValidationError) Error() string { return e.Message }

func dietConfigInvalid(format string, args ...any) error {
	return &DietConfigValidationError{Message: fmt.Sprintf(format, args...)}
}

// DietMealConfigMealInput 是单个餐次的保存入参。
type DietMealConfigMealInput struct {
	MealKey       domain.DietMealKey
	Time          string
	EnergyPercent float64
	Enabled       bool
}

// DietMealConfigMealView 是单个餐次的响应项。
type DietMealConfigMealView struct {
	MealKey       domain.DietMealKey `json:"meal_key"`
	Name          string             `json:"name"`
	Time          string             `json:"time"`
	EnergyPercent float64            `json:"energy_percent"`
	Sort          int                `json:"sort"`
	Enabled       bool               `json:"enabled"`
}

// 默认值取自调研 notes §2.3（与契约 23-diet.md 的餐次枚举表一致）：
// 日托、标准比例、早餐/早点/午餐/午点/晚餐启用，晚点默认停用，合计 95%。
func defaultDietMealConfigMeals() []DietMealConfigMealInput {
	return []DietMealConfigMealInput{
		{MealKey: domain.DietMealBreakfast, Time: "07:00", EnergyPercent: 25, Enabled: true},
		{MealKey: domain.DietMealMorningSnack, Time: "09:00", EnergyPercent: 5, Enabled: true},
		{MealKey: domain.DietMealLunch, Time: "11:00", EnergyPercent: 35, Enabled: true},
		{MealKey: domain.DietMealAfternoonSnack, Time: "14:30", EnergyPercent: 5, Enabled: true},
		{MealKey: domain.DietMealDinner, Time: "17:00", EnergyPercent: 25, Enabled: true},
		{MealKey: domain.DietMealEveningSnack, Time: "20:00", EnergyPercent: 5, Enabled: false},
	}
}

// dietStandardEnergyPercent 是 standard 模式下每餐的固定供能占比（契约校验第 5 条）。
// 标准比例是固定预设，不参与相邻上限校验——这正是默认 0.95 能合法保存的前提。
var dietStandardEnergyPercent = map[domain.DietMealKey]float64{
	domain.DietMealBreakfast:      25,
	domain.DietMealMorningSnack:   5,
	domain.DietMealLunch:          35,
	domain.DietMealAfternoonSnack: 5,
	domain.DietMealDinner:         25,
	domain.DietMealEveningSnack:   5,
}

var dietMealNameByKey = map[domain.DietMealKey]string{
	domain.DietMealBreakfast:      "早餐",
	domain.DietMealMorningSnack:   "早点",
	domain.DietMealLunch:          "午餐",
	domain.DietMealAfternoonSnack: "午点",
	domain.DietMealDinner:         "晚餐",
	domain.DietMealEveningSnack:   "晚点",
}

// mealTimePattern 要求两位补零的 24 小时制时间；time.Parse("15:04") 会接受 "7:00"，
// 与契约「HH:mm（两位补零）」不符，所以用正则卡死格式。
var mealTimePattern = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)

// validateDietMealConfig 是供餐配置的公共校验：托管形式、重量口径、比例模式与全部餐次。
// 返回收敛后的比例模式与按固定餐次顺序排列的入参。
func validateDietMealConfig(
	careType domain.DietCareType,
	foodInputType domain.DietFoodInputType,
	ratioMode domain.DietRatioMode,
	meals []DietMealConfigMealInput,
) (domain.DietRatioMode, []DietMealConfigMealInput, error) {
	switch careType {
	case domain.DietCareTypeDayCare, domain.DietCareTypeFullCare:
	default:
		return "", nil, dietConfigInvalid("托管形式无效，只支持日托或全托")
	}
	// 口径在本变更中不可配置：只接受市品重量（TC-NEW-DIE-0007）。
	if foodInputType != domain.DietFoodInputTypeMarketWeight {
		return "", nil, dietConfigInvalid("食物重量口径只支持市品重量")
	}
	normalizedRatioMode, err := normalizeRatioMode(ratioMode)
	if err != nil {
		return "", nil, err
	}
	if len(meals) != len(domain.DietMealKeyOrder) {
		return "", nil, dietConfigInvalid(
			"餐次必须恰好包含 %d 个已知餐次，每个出现一次", len(domain.DietMealKeyOrder))
	}

	byKey := make(map[domain.DietMealKey]DietMealConfigMealInput, len(meals))
	for _, meal := range meals {
		if domain.DietMealSort(meal.MealKey) == 0 {
			return "", nil, dietConfigInvalid("未知餐次：%s", meal.MealKey)
		}
		if _, dup := byKey[meal.MealKey]; dup {
			return "", nil, dietConfigInvalid("餐次 %s 重复", dietMealName(meal.MealKey))
		}
		byKey[meal.MealKey] = meal
	}

	ordered := make([]DietMealConfigMealInput, 0, len(domain.DietMealKeyOrder))
	enabledCount := 0
	for _, key := range domain.DietMealKeyOrder {
		meal := byKey[key]
		if !mealTimePattern.MatchString(meal.Time) {
			return "", nil, dietConfigInvalid("%s 的时间格式必须为 HH:mm", dietMealName(key))
		}
		if math.IsNaN(meal.EnergyPercent) || math.IsInf(meal.EnergyPercent, 0) ||
			meal.EnergyPercent < 0 {
			return "", nil, dietConfigInvalid("%s 的供能占比无效", dietMealName(key))
		}
		if meal.Enabled {
			if meal.EnergyPercent <= 0 {
				return "", nil, dietConfigInvalid(
					"已启用餐次（%s）的供能占比必须大于 0", dietMealName(key))
			}
			enabledCount++
		}
		ordered = append(ordered, meal)
	}
	if enabledCount == 0 {
		return "", nil, dietConfigInvalid("至少需要启用一个餐次")
	}
	if err := validateEnabledMealTimeOrder(ordered); err != nil {
		return "", nil, err
	}
	switch normalizedRatioMode {
	case domain.DietRatioModeStandard:
		if err := validateStandardEnergyPercent(ordered); err != nil {
			return "", nil, err
		}
	case domain.DietRatioModeCustom:
		if err := validateAdjacentEnergyCaps(careType, byKey); err != nil {
			return "", nil, err
		}
	}
	return normalizedRatioMode, ordered, nil
}

// normalizeRatioMode 把缺省值收敛为 standard（契约：缺省按 standard）。
func normalizeRatioMode(mode domain.DietRatioMode) (domain.DietRatioMode, error) {
	switch mode {
	case "":
		return domain.DietRatioModeStandard, nil
	case domain.DietRatioModeStandard, domain.DietRatioModeCustom:
		return mode, nil
	default:
		return "", dietConfigInvalid("比例模式无效，只支持标准比例或自定比例")
	}
}

// normalizeRatioModeForRead 兼容加列前写入的历史行：空值按 standard 展示。
func normalizeRatioModeForRead(mode domain.DietRatioMode) domain.DietRatioMode {
	if mode == "" {
		return domain.DietRatioModeStandard
	}
	return mode
}

// validateStandardEnergyPercent 只校验已启用餐次：停用餐次保留任意占比但不算入系数。
func validateStandardEnergyPercent(ordered []DietMealConfigMealInput) error {
	for _, meal := range ordered {
		if !meal.Enabled {
			continue
		}
		standard := dietStandardEnergyPercent[meal.MealKey]
		if math.Abs(meal.EnergyPercent-standard) > 1e-9 {
			return dietConfigInvalid(
				"标准比例模式下%s的供能占比必须为 %g%%", dietMealName(meal.MealKey), standard)
		}
	}
	return nil
}

// validateEnabledMealTimeOrder 只比较已启用餐次：停用餐次保留时间但参与不了供餐，
// 不应拦住保存。ordered 已按固定餐次顺序排列。
func validateEnabledMealTimeOrder(ordered []DietMealConfigMealInput) error {
	var prevKey domain.DietMealKey
	prevMinutes := 0
	for _, meal := range ordered {
		if !meal.Enabled {
			continue
		}
		minutes := mealTimeMinutes(meal.Time)
		if prevKey != "" && minutes <= prevMinutes {
			return dietConfigInvalid("餐次时间必须按早餐、早点、午餐、午点、晚餐、晚点的顺序严格递增")
		}
		prevKey = meal.MealKey
		prevMinutes = minutes
	}
	return nil
}

// validateAdjacentEnergyCaps 校验自定比例模式下相邻餐次共享的供能占比上限。
// 日托的早餐+早点上限比全托少 1 个百分点（调研 notes §2.3、契约校验第 6 条）。
func validateAdjacentEnergyCaps(
	careType domain.DietCareType,
	byKey map[domain.DietMealKey]DietMealConfigMealInput,
) error {
	firstPairLimit := 30.0
	if careType == domain.DietCareTypeDayCare {
		firstPairLimit = 29
	}
	pairs := []struct {
		first, second domain.DietMealKey
		limit         float64
		label         string
	}{
		{domain.DietMealBreakfast, domain.DietMealMorningSnack, firstPairLimit, "早餐与早点"},
		{domain.DietMealLunch, domain.DietMealAfternoonSnack, 39, "午餐与午点"},
		{domain.DietMealDinner, domain.DietMealEveningSnack, 30, "晚餐与晚点"},
	}
	for _, pair := range pairs {
		sum := byKey[pair.first].EnergyPercent + byKey[pair.second].EnergyPercent
		if sum > pair.limit+1e-9 {
			return dietConfigInvalid("%s的供能占比合计不能超过 %g%%", pair.label, pair.limit)
		}
	}
	return nil
}

func dietMealName(key domain.DietMealKey) string { return dietMealNameByKey[key] }

func dietCareTypeLabel(careType domain.DietCareType) string {
	switch careType {
	case domain.DietCareTypeDayCare:
		return "日托"
	case domain.DietCareTypeFullCare:
		return "全托"
	default:
		return ""
	}
}

func dietFoodInputTypeLabel(foodInputType domain.DietFoodInputType) string {
	if foodInputType == domain.DietFoodInputTypeMarketWeight {
		return "市品重量"
	}
	return ""
}

func dietRatioModeLabel(ratioMode domain.DietRatioMode) string {
	switch ratioMode {
	case domain.DietRatioModeStandard:
		return "标准比例"
	case domain.DietRatioModeCustom:
		return "自定比例"
	default:
		return ""
	}
}

// mealTimeMinutes 把已通过正则校验的 HH:mm 转成分钟数，用于时间递增比较。
func mealTimeMinutes(hhmm string) int {
	return (int(hhmm[0]-'0')*10+int(hhmm[1]-'0'))*60 +
		int(hhmm[3]-'0')*10 + int(hhmm[4]-'0')
}
