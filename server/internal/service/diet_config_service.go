package service

import (
	"context"
	"fmt"
	"math"
	"regexp"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// DietConfigValidationError 表示供餐配置未通过业务校验。
// handler 用 errors.As 识别并映射为项目统一的 422 包络；Message 可直接展示给管理员。
type DietConfigValidationError struct {
	Message string
}

func (e *DietConfigValidationError) Error() string { return e.Message }

func dietConfigInvalid(format string, args ...any) error {
	return &DietConfigValidationError{Message: fmt.Sprintf(format, args...)}
}

// DietMealConfigInput 是保存供餐配置的入参，只含管理员可改的字段。
// 餐次名称与 sort 由服务端按 meal_key 固定写入（契约 23-diet.md：name/sort 只读）。
type DietMealConfigInput struct {
	CareType      domain.DietCareType
	FoodInputType domain.DietFoodInputType
	RatioMode     domain.DietRatioMode
	Meals         []DietMealConfigMealInput
}

// DietMealConfigMealInput 是单个餐次的保存入参。
type DietMealConfigMealInput struct {
	MealKey       domain.DietMealKey
	Time          string
	EnergyPercent float64
	Enabled       bool
}

// DietConfigView 是 GET/PUT /admin/diet/config 的 data 部分。
// 不直接返回 domain.MealConfig：契约要求 label 与只读 coefficient，
// 且 kindergarten_id 不应出现在响应里。
type DietConfigView struct {
	CareType           domain.DietCareType      `json:"care_type"`
	CareTypeLabel      string                   `json:"care_type_label"`
	FoodInputType      domain.DietFoodInputType `json:"food_input_type"`
	FoodInputTypeLabel string                   `json:"food_input_type_label"`
	RatioMode          domain.DietRatioMode     `json:"ratio_mode"`
	RatioModeLabel     string                   `json:"ratio_mode_label"`
	Coefficient        float64                  `json:"coefficient"`
	Meals              []DietMealConfigMealView `json:"meals"`
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

// DietConfigService 供餐配置服务。
//
// 配置是园所单例：首次读取返回默认值（不落库），保存后读取返回已保存值。
// 已保存周食谱读自己的 config_snapshot，不随本配置回溯（design D3、TC-NEW-DIE-0008）。
type DietConfigService struct {
	cfg *repository.DietConfigRepo
}

func NewDietConfigService(cfg *repository.DietConfigRepo) *DietConfigService {
	return &DietConfigService{cfg: cfg}
}

// GetConfig 返回园所供餐配置；园所未配置时返回默认配置。
// 默认值刻意不落库：读取路径不产生写入，重复读取结果一致。
func (s *DietConfigService) GetConfig(ctx context.Context, kindergartenID uint64) (*DietConfigView, error) {
	if kindergartenID == 0 {
		return nil, dietConfigInvalid("未绑定园所，无法读取供餐配置")
	}
	cfg, err := s.cfg.GetByKindergarten(ctx, kindergartenID)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return buildDietConfigView(
			domain.DietCareTypeDayCare,
			domain.DietFoodInputTypeMarketWeight,
			domain.DietRatioModeStandard,
			defaultDietMealConfigMeals(),
		), nil
	}
	return buildDietConfigView(
		cfg.CareType,
		cfg.FoodInputType,
		normalizeRatioModeForRead(cfg.RatioMode),
		dietMealInputsFromModel(cfg.Meals),
	), nil
}

// SaveConfig 校验并保存园所供餐配置（整组替换餐次），返回保存后的配置。
func (s *DietConfigService) SaveConfig(
	ctx context.Context,
	kindergartenID uint64,
	in DietMealConfigInput,
) (*DietConfigView, error) {
	if kindergartenID == 0 {
		return nil, dietConfigInvalid("未绑定园所，无法保存供餐配置")
	}
	cfg, err := validateDietConfigInput(kindergartenID, in)
	if err != nil {
		return nil, err
	}
	if err := s.cfg.Save(ctx, cfg); err != nil {
		return nil, err
	}
	return buildDietConfigView(cfg.CareType, cfg.FoodInputType, cfg.RatioMode, dietMealInputsFromModel(cfg.Meals)), nil
}

// BuildConfigSnapshot 返回当前供餐配置的固化快照，供 S6 创建周食谱时写入
// MealPlan.config_snapshot。快照一旦写入就不随配置变更回溯（design D3）。
// DRIs 目标值依赖班级人数加权，属于 S6/S7；本方法只填餐次结构、口径、比例模式与折算系数。
func (s *DietConfigService) BuildConfigSnapshot(
	ctx context.Context,
	kindergartenID uint64,
) (*domain.DietConfigSnapshot, error) {
	view, err := s.GetConfig(ctx, kindergartenID)
	if err != nil {
		return nil, err
	}
	snapshot := &domain.DietConfigSnapshot{
		CareType:      view.CareType,
		FoodInputType: view.FoodInputType,
		RatioMode:     view.RatioMode,
		Coefficient:   view.Coefficient,
	}
	for _, meal := range view.Meals {
		snapshot.Meals = append(snapshot.Meals, domain.DietMealSnapshot{
			MealKey:       meal.MealKey,
			Name:          meal.Name,
			Time:          meal.Time,
			EnergyPercent: meal.EnergyPercent,
			Sort:          meal.Sort,
			Enabled:       meal.Enabled,
		})
	}
	return snapshot, nil
}

// validateDietConfigInput 校验并规范化保存入参，返回可直接落库的模型。
// 校验失败时不产生任何写入，原配置保持不变。
func validateDietConfigInput(kindergartenID uint64, in DietMealConfigInput) (*domain.MealConfig, error) {
	switch in.CareType {
	case domain.DietCareTypeDayCare, domain.DietCareTypeFullCare:
	default:
		return nil, dietConfigInvalid("托管形式无效，只支持日托或全托")
	}
	// 口径在本变更中不可配置：只接受市品重量（TC-NEW-DIE-0007）。
	if in.FoodInputType != domain.DietFoodInputTypeMarketWeight {
		return nil, dietConfigInvalid("食物重量口径只支持市品重量")
	}
	ratioMode, err := normalizeRatioMode(in.RatioMode)
	if err != nil {
		return nil, err
	}
	if len(in.Meals) != len(domain.DietMealKeyOrder) {
		return nil, dietConfigInvalid("餐次必须恰好包含 %d 个已知餐次，每个出现一次", len(domain.DietMealKeyOrder))
	}

	byKey := make(map[domain.DietMealKey]DietMealConfigMealInput, len(in.Meals))
	for _, meal := range in.Meals {
		if domain.DietMealSort(meal.MealKey) == 0 {
			return nil, dietConfigInvalid("未知餐次：%s", meal.MealKey)
		}
		if _, dup := byKey[meal.MealKey]; dup {
			return nil, dietConfigInvalid("餐次 %s 重复", dietMealName(meal.MealKey))
		}
		byKey[meal.MealKey] = meal
	}

	ordered := make([]DietMealConfigMealInput, 0, len(domain.DietMealKeyOrder))
	enabledCount := 0
	for _, key := range domain.DietMealKeyOrder {
		meal := byKey[key]
		if !mealTimePattern.MatchString(meal.Time) {
			return nil, dietConfigInvalid("%s 的时间格式必须为 HH:mm", dietMealName(key))
		}
		if math.IsNaN(meal.EnergyPercent) || math.IsInf(meal.EnergyPercent, 0) ||
			meal.EnergyPercent < 0 {
			return nil, dietConfigInvalid("%s 的供能占比无效", dietMealName(key))
		}
		if meal.Enabled {
			if meal.EnergyPercent <= 0 {
				return nil, dietConfigInvalid("已启用餐次（%s）的供能占比必须大于 0", dietMealName(key))
			}
			enabledCount++
		}
		ordered = append(ordered, meal)
	}
	if enabledCount == 0 {
		return nil, dietConfigInvalid("至少需要启用一个餐次")
	}
	if err := validateEnabledMealTimeOrder(ordered); err != nil {
		return nil, err
	}
	switch ratioMode {
	case domain.DietRatioModeStandard:
		if err := validateStandardEnergyPercent(ordered); err != nil {
			return nil, err
		}
	case domain.DietRatioModeCustom:
		if err := validateAdjacentEnergyCaps(in.CareType, byKey); err != nil {
			return nil, err
		}
	}

	meals := make([]domain.MealConfigMeal, 0, len(ordered))
	for _, meal := range ordered {
		meals = append(meals, domain.MealConfigMeal{
			KindergartenID: kindergartenID,
			MealKey:        meal.MealKey,
			Name:           dietMealName(meal.MealKey),
			Time:           meal.Time,
			EnergyPercent:  meal.EnergyPercent,
			Sort:           domain.DietMealSort(meal.MealKey),
			Enabled:        meal.Enabled,
		})
	}
	return &domain.MealConfig{
		KindergartenID: kindergartenID,
		CareType:       in.CareType,
		FoodInputType:  in.FoodInputType,
		RatioMode:      ratioMode,
		Meals:          meals,
	}, nil
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

// buildDietConfigView 按固定餐次顺序输出，并按已启用餐次占比之和计算折算系数。
func buildDietConfigView(
	careType domain.DietCareType,
	foodInputType domain.DietFoodInputType,
	ratioMode domain.DietRatioMode,
	meals []DietMealConfigMealInput,
) *DietConfigView {
	byKey := make(map[domain.DietMealKey]DietMealConfigMealInput, len(meals))
	for _, meal := range meals {
		byKey[meal.MealKey] = meal
	}

	views := make([]DietMealConfigMealView, 0, len(domain.DietMealKeyOrder))
	enabledSum := 0.0
	for _, key := range domain.DietMealKeyOrder {
		meal, ok := byKey[key]
		if !ok {
			continue
		}
		views = append(views, DietMealConfigMealView{
			MealKey:       key,
			Name:          dietMealName(key),
			Time:          meal.Time,
			EnergyPercent: meal.EnergyPercent,
			Sort:          domain.DietMealSort(key),
			Enabled:       meal.Enabled,
		})
		if meal.Enabled {
			enabledSum += meal.EnergyPercent
		}
	}
	return &DietConfigView{
		CareType:           careType,
		CareTypeLabel:      dietCareTypeLabel(careType),
		FoodInputType:      foodInputType,
		FoodInputTypeLabel: dietFoodInputTypeLabel(foodInputType),
		RatioMode:          ratioMode,
		RatioModeLabel:     dietRatioModeLabel(ratioMode),
		Coefficient:        math.Round(enabledSum/100*10000) / 10000,
		Meals:              views,
	}
}

func dietMealInputsFromModel(meals []domain.MealConfigMeal) []DietMealConfigMealInput {
	out := make([]DietMealConfigMealInput, 0, len(meals))
	for _, meal := range meals {
		out = append(out, DietMealConfigMealInput{
			MealKey:       meal.MealKey,
			Time:          meal.Time,
			EnergyPercent: meal.EnergyPercent,
			Enabled:       meal.Enabled,
		})
	}
	return out
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
