package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/pkg/pagination"
)

// DietPlanValidationError 表示周食谱入参或业务校验失败。
// handler 用 errors.As 识别并映射为项目统一的 422 包络；Message 可直接展示给管理员。
type DietPlanValidationError struct {
	Message string
}

func (e *DietPlanValidationError) Error() string { return e.Message }

func dietPlanInvalid(format string, args ...any) error {
	return &DietPlanValidationError{Message: fmt.Sprintf(format, args...)}
}

// 周食谱访问与创建的业务错误。跨园访问在仓储层查不到，统一按 NotFound 处理（契约 404）。
var (
	ErrDietPlanNotFound     = errors.New("DIET_PLAN_NOT_FOUND")
	ErrDietPlanNameConflict = errors.New("DIET_PLAN_NAME_CONFLICT")
	ErrDietTemplateNotFound = errors.New("DIET_TEMPLATE_NOT_FOUND")
)

const dietPlanDateLayout = "2006-01-02"

// DietPlanListItem 是周食谱列表行：只含名称、周期、天数与创建方式（契约 §五 列表裁剪）。
type DietPlanListItem struct {
	ID                 uint64                       `json:"id"`
	Name               string                       `json:"name"`
	AcademicYearID     uint64                       `json:"academic_year_id"`
	StartDate          string                       `json:"start_date"`
	EndDate            string                       `json:"end_date"`
	DayNum             int                          `json:"day_num"`
	WeekOfYear         int                          `json:"week_of_year"`
	CreatedMethod      domain.DietPlanCreatedMethod `json:"created_method"`
	CreatedMethodLabel string                       `json:"created_method_label"`
}

// DietPlanCreateInput 是 POST /admin/diet/plans 的入参。
// 三种创建方式三选一：custom 不接受附加字段，template 必须带 TemplateID，
// copy 必须带 SourcePlanID（契约校验第 3 条）。
type DietPlanCreateInput struct {
	Name           string
	AcademicYearID uint64
	StartDate      string
	EndDate        string
	WeekOfYear     int
	CreatedMethod  domain.DietPlanCreatedMethod
	TemplateID     *uint64
	SourcePlanID   *uint64
}

// DietPlanCreateResult 是新建周食谱的响应。Warnings 列出复制时被跳过的条目。
type DietPlanCreateResult struct {
	ID         uint64   `json:"id"`
	Name       string   `json:"name"`
	StartDate  string   `json:"start_date"`
	EndDate    string   `json:"end_date"`
	DayNum     int      `json:"day_num"`
	WeekOfYear int      `json:"week_of_year"`
	Warnings   []string `json:"warnings"`
}

// DietPlanItemView 是周食谱详情中的一行明细。
// Weekday 由 Date 推导；NutritionSnapshot 是保存当时的每 100g 浓度快照。
type DietPlanItemView struct {
	ID                uint64                       `json:"id"`
	Date              string                       `json:"date"`
	Weekday           int                          `json:"weekday"`
	MealKey           domain.DietMealKey           `json:"meal_key"`
	RecipeName        string                       `json:"recipe_name"`
	RecipeID          *uint64                      `json:"recipe_id"`
	FoodID            *uint64                      `json:"food_id"`
	FoodName          string                       `json:"food_name"`
	Grams             float64                      `json:"grams"`
	Sort              int                          `json:"sort"`
	NutritionSnapshot domain.DietNutritionSnapshot `json:"nutrition_snapshot"`
}

// DietPlanDetail 是周食谱详情：配置快照 + 全部明细（契约 §五 GET /plans/:id）。
type DietPlanDetail struct {
	ID                 uint64                       `json:"id"`
	Name               string                       `json:"name"`
	AcademicYearID     uint64                       `json:"academic_year_id"`
	StartDate          string                       `json:"start_date"`
	EndDate            string                       `json:"end_date"`
	DayNum             int                          `json:"day_num"`
	WeekOfYear         int                          `json:"week_of_year"`
	CreatedMethod      domain.DietPlanCreatedMethod `json:"created_method"`
	CreatedMethodLabel string                       `json:"created_method_label"`
	SourceTemplateID   *uint64                      `json:"source_template_id"`
	SourcePlanID       *uint64                      `json:"source_plan_id"`
	ConfigSnapshot     domain.DietConfigSnapshot    `json:"config_snapshot"`
	Items              []DietPlanItemView           `json:"items"`
}

// DietPlanItemInput 是整单保存时提交的一行明细。Sort 不接受客户端输入。
type DietPlanItemInput struct {
	Date       string
	MealKey    domain.DietMealKey
	RecipeName string
	RecipeID   *uint64
	FoodID     uint64
	// FoodName 仅作旧数据兜底：校验失败时用于定位单元格，落库一律以服务端食物为准。
	FoodName string
	Grams    float64
}

// DietPlanSaveInput 是 PUT /admin/diet/plans/:id 的入参。
// Items 为 nil 表示请求体缺字段（422）；空切片表示清空明细（契约规则 1）。
type DietPlanSaveInput struct {
	Name  *string
	Items []DietPlanItemInput
}

// DietPlanSaveResult 是整单保存的响应。
type DietPlanSaveResult struct {
	ID        uint64 `json:"id"`
	DayNum    int    `json:"day_num"`
	ItemCount int    `json:"item_count"`
}

// DietPlanClearResult 是清空明细的响应。
type DietPlanClearResult struct {
	ID     uint64 `json:"id"`
	DayNum int    `json:"day_num"`
}

// DietTemplateListItem 是模板列表行。
type DietTemplateListItem struct {
	ID        uint64   `json:"id"`
	Name      string   `json:"name"`
	DayNum    int      `json:"day_num"`
	MealKeys  []string `json:"meal_keys"`
	ItemCount int64    `json:"item_count"`
}

// DietTemplateItemView 是模板详情中的条目。
type DietTemplateItemView struct {
	Weekday    int                `json:"weekday"`
	MealKey    domain.DietMealKey `json:"meal_key"`
	RecipeName string             `json:"recipe_name"`
	FoodID     uint64             `json:"food_id"`
	FoodName   string             `json:"food_name"`
	Grams      float64            `json:"grams"`
	Sort       int                `json:"sort"`
}

// DietTemplateDetail 是模板详情（只读）。
type DietTemplateDetail struct {
	ID       uint64                 `json:"id"`
	Name     string                 `json:"name"`
	DayNum   int                    `json:"day_num"`
	MealKeys []string               `json:"meal_keys"`
	Items    []DietTemplateItemView `json:"items"`
}

// DietPlanService 周食谱编排。
//
// 创建时固化供餐配置与按班级人数加权的 DRIs 目标值；此后修改配置、名册或食物库
// 都不回溯已保存的食谱（design D3/D5）。保存采用整单替换，单事务完成。
type DietPlanService struct {
	plans     *repository.DietPlanRepo
	templates *repository.DietTemplateRepo
	foods     *repository.DietFoodRepo
	classes   *repository.ClassRepo
	years     *repository.AcademicYearRepo
	dris      *repository.DietDriRepo
	config    *DietConfigService
}

func NewDietPlanService(
	plans *repository.DietPlanRepo,
	templates *repository.DietTemplateRepo,
	foods *repository.DietFoodRepo,
	classes *repository.ClassRepo,
	years *repository.AcademicYearRepo,
	dris *repository.DietDriRepo,
	config *DietConfigService,
) *DietPlanService {
	return &DietPlanService{
		plans:     plans,
		templates: templates,
		foods:     foods,
		classes:   classes,
		years:     years,
		dris:      dris,
		config:    config,
	}
}

// ListPlans 返回本园周食谱列表。academic_year_id 缺省取当前学年（is_current）；
// 没有当前学年时返回空列表，而不是跨学年混排。
func (s *DietPlanService) ListPlans(
	ctx context.Context,
	kindergartenID, academicYearID uint64,
	keyword string,
	pq pagination.Query,
) ([]DietPlanListItem, int64, int, int, error) {
	if kindergartenID == 0 {
		return nil, 0, 0, 0, dietPlanInvalid("未绑定园所，无法读取周食谱")
	}
	pn := pq
	pn.Normalize()

	yearID := academicYearID
	if yearID == 0 {
		current, err := s.years.FindCurrent(ctx, kindergartenID)
		if err != nil {
			return nil, 0, 0, 0, err
		}
		if current == nil {
			return []DietPlanListItem{}, 0, pn.Page, pn.PageSize, nil
		}
		yearID = current.ID
	}

	rows, total, err := s.plans.List(ctx, kindergartenID, yearID, keyword, pn.Offset(), pn.PageSize)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	out := make([]DietPlanListItem, 0, len(rows))
	for i := range rows {
		out = append(out, dietPlanListItemView(&rows[i]))
	}
	return out, total, pn.Page, pn.PageSize, nil
}

// CreatePlan 按三种方式之一新建周食谱，创建时固化配置与 DRIs 快照。
func (s *DietPlanService) CreatePlan(
	ctx context.Context,
	kindergartenID uint64,
	in DietPlanCreateInput,
) (*DietPlanCreateResult, error) {
	if kindergartenID == 0 {
		return nil, dietPlanInvalid("未绑定园所，无法新建周食谱")
	}
	name, err := validateDietPlanName(in.Name)
	if err != nil {
		return nil, err
	}
	method, err := validateDietPlanCreatedMethod(in.CreatedMethod)
	if err != nil {
		return nil, err
	}
	if err := validateDietPlanMethodFields(method, in.TemplateID, in.SourcePlanID); err != nil {
		return nil, err
	}
	if in.AcademicYearID == 0 {
		return nil, dietPlanInvalid("请选择学年")
	}
	year, err := s.findAcademicYear(ctx, kindergartenID, in.AcademicYearID)
	if err != nil {
		return nil, err
	}
	if year == nil {
		return nil, dietPlanInvalid("学年不存在或不属于本园")
	}
	start, err := parseDietPlanPeriod(in.StartDate, in.EndDate)
	if err != nil {
		return nil, err
	}
	existing, err := s.plans.FindByName(ctx, kindergartenID, year.ID, name, 0)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrDietPlanNameConflict
	}
	configSnapshot, err := s.buildDietConfigSnapshot(ctx, kindergartenID)
	if err != nil {
		return nil, err
	}

	weekOfYear := in.WeekOfYear
	if weekOfYear < 0 {
		weekOfYear = 0
	}
	plan := &domain.MealPlan{
		KindergartenID: kindergartenID,
		AcademicYearID: year.ID,
		Name:           name,
		StartDate:      start,
		EndDate:        start.AddDate(0, 0, 6),
		WeekOfYear:     weekOfYear,
		CreatedMethod:  method,
		ConfigSnapshot: configSnapshot,
	}
	warnings := make([]string, 0)
	switch method {
	case domain.DietPlanCreatedByTemplate:
		plan.SourceTemplateID = in.TemplateID
		items, warns, err := s.buildItemsFromTemplate(ctx, kindergartenID, start, *in.TemplateID)
		if err != nil {
			return nil, err
		}
		plan.Items = items
		warnings = warns
	case domain.DietPlanCreatedByCopy:
		plan.SourcePlanID = in.SourcePlanID
		items, warns, err := s.buildItemsFromCopy(ctx, kindergartenID, start, *in.SourcePlanID)
		if err != nil {
			return nil, err
		}
		plan.Items = items
		warnings = warns
	}
	plan.DayNum = countDietPlanDays(plan.Items)

	if err := s.plans.Create(ctx, plan); err != nil {
		return nil, err
	}
	return &DietPlanCreateResult{
		ID:         plan.ID,
		Name:       plan.Name,
		StartDate:  plan.StartDate.Format(dietPlanDateLayout),
		EndDate:    plan.EndDate.Format(dietPlanDateLayout),
		DayNum:     plan.DayNum,
		WeekOfYear: plan.WeekOfYear,
		Warnings:   warnings,
	}, nil
}

// GetPlan 返回周食谱详情；跨园或不存在一律 ErrDietPlanNotFound（契约 404）。
func (s *DietPlanService) GetPlan(
	ctx context.Context,
	kindergartenID, id uint64,
) (*DietPlanDetail, error) {
	if kindergartenID == 0 {
		return nil, dietPlanInvalid("未绑定园所，无法读取周食谱")
	}
	plan, err := s.plans.FindByIDWithItems(ctx, kindergartenID, id)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, ErrDietPlanNotFound
	}
	snapshot, err := parseDietConfigSnapshot(plan.ConfigSnapshot)
	if err != nil {
		return nil, err
	}

	items := make([]DietPlanItemView, 0, len(plan.Items))
	for _, item := range plan.Items {
		nutrition, err := parseDietNutritionSnapshot(item.NutritionSnapshot)
		if err != nil {
			return nil, err
		}
		items = append(items, DietPlanItemView{
			ID:                item.ID,
			Date:              item.Date.Format(dietPlanDateLayout),
			Weekday:           dietWeekdayFromDate(item.Date),
			MealKey:           item.MealKey,
			RecipeName:        item.RecipeName,
			RecipeID:          item.RecipeID,
			FoodID:            item.FoodID,
			FoodName:          item.FoodName,
			Grams:             item.Grams,
			Sort:              item.Sort,
			NutritionSnapshot: nutrition,
		})
	}
	// 仓储按 date/sort/id 返回；同一天不同餐次的 sort 各自从 1 开始，
	// 服务层再按「日期 → 餐次固定顺序 → 单元格内顺序」排一次，保证编辑器顺序稳定。
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Date != items[j].Date {
			return items[i].Date < items[j].Date
		}
		mi, mj := domain.DietMealSort(items[i].MealKey), domain.DietMealSort(items[j].MealKey)
		if mi != mj {
			return mi < mj
		}
		if items[i].Sort != items[j].Sort {
			return items[i].Sort < items[j].Sort
		}
		return items[i].ID < items[j].ID
	})

	return &DietPlanDetail{
		ID:                 plan.ID,
		Name:               plan.Name,
		AcademicYearID:     plan.AcademicYearID,
		StartDate:          plan.StartDate.Format(dietPlanDateLayout),
		EndDate:            plan.EndDate.Format(dietPlanDateLayout),
		DayNum:             plan.DayNum,
		WeekOfYear:         plan.WeekOfYear,
		CreatedMethod:      plan.CreatedMethod,
		CreatedMethodLabel: dietPlanCreatedMethodLabel(plan.CreatedMethod),
		SourceTemplateID:   plan.SourceTemplateID,
		SourcePlanID:       plan.SourcePlanID,
		ConfigSnapshot:     snapshot,
		Items:              items,
	}, nil
}

// SavePlan 整单保存：校验 → 同菜同食物合并 → 用当前食物库重算营养快照 →
// 单事务替换表头与全部明细（契约 §五 PUT 规则 1–9）。
func (s *DietPlanService) SavePlan(
	ctx context.Context,
	kindergartenID, id uint64,
	in DietPlanSaveInput,
) (*DietPlanSaveResult, error) {
	if kindergartenID == 0 {
		return nil, dietPlanInvalid("未绑定园所，无法保存周食谱")
	}
	plan, err := s.plans.FindByIDWithItems(ctx, kindergartenID, id)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, ErrDietPlanNotFound
	}
	// 契约规则 1：items 必填；空数组等同清空（前端负责二次确认）。
	if in.Items == nil {
		return nil, dietPlanInvalid("items 必填；清空明细请提交空数组")
	}

	snapshot, err := parseDietConfigSnapshot(plan.ConfigSnapshot)
	if err != nil {
		return nil, err
	}
	enabledMeals := dietEnabledMeals(snapshot)
	startKey := plan.StartDate.Format(dietPlanDateLayout)
	endKey := plan.EndDate.Format(dietPlanDateLayout)

	rows := make([]dietPlanSaveRow, 0, len(in.Items))
	foodIDSet := make(map[uint64]bool, len(in.Items))
	foodIDs := make([]uint64, 0, len(in.Items))
	for _, item := range in.Items {
		date, err := parseDietPlanDate(item.Date)
		if err != nil {
			return nil, dietPlanInvalid("明细日期格式必须为 YYYY-MM-DD：%s", item.Date)
		}
		recipeName := strings.TrimSpace(item.RecipeName)
		foodLabel := dietPlanFoodLabel(item.FoodID, item.FoodName)
		if dateKey := date.Format(dietPlanDateLayout); dateKey < startKey || dateKey > endKey {
			return nil, dietPlanInvalid(
				"「%s」的「%s」日期 %s 不在食谱周期内", recipeName, foodLabel, item.Date)
		}
		if domain.DietMealSort(item.MealKey) == 0 {
			return nil, dietPlanInvalid("「%s」的「%s」餐次无效：%s", recipeName, foodLabel, item.MealKey)
		}
		if recipeName == "" {
			return nil, dietPlanInvalid("菜名不能为空")
		}
		if utf8.RuneCountInString(recipeName) > 128 {
			return nil, dietPlanInvalid("菜名不能超过 128 个字符")
		}
		if math.IsNaN(item.Grams) || math.IsInf(item.Grams, 0) || item.Grams <= 0 {
			return nil, dietPlanInvalid("「%s」的「%s」一人均量必须大于 0", recipeName, foodLabel)
		}
		if item.FoodID == 0 {
			return nil, dietPlanInvalid("「%s」的「%s」未选择食物", recipeName, foodLabel)
		}
		if !enabledMeals[item.MealKey] {
			return nil, dietPlanInvalid(
				"「%s」的「%s」所在餐次（%s）未在食谱配置中启用",
				recipeName, foodLabel, dietMealName(item.MealKey))
		}
		rows = append(rows, dietPlanSaveRow{
			date:       date,
			weekday:    dietWeekdayFromDate(date),
			mealKey:    item.MealKey,
			recipeName: recipeName,
			recipeID:   item.RecipeID,
			foodID:     item.FoodID,
			foodLabel:  foodLabel,
			grams:      item.Grams,
		})
		if !foodIDSet[item.FoodID] {
			foodIDSet[item.FoodID] = true
			foodIDs = append(foodIDs, item.FoodID)
		}
	}

	foods, err := s.foods.FindByIDs(ctx, kindergartenID, foodIDs)
	if err != nil {
		return nil, err
	}
	foodByID := make(map[uint64]*domain.Food, len(foods))
	for i := range foods {
		foodByID[foods[i].ID] = &foods[i]
	}
	for _, row := range rows {
		if _, ok := foodByID[row.foodID]; !ok {
			return nil, dietPlanInvalid(
				"「%s」的「%s」不在本园食物库中，请重新选择", row.recipeName, row.foodLabel)
		}
	}

	items, dayNum, err := buildDietPlanItems(ctx, rows, foodByID)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		name, err := validateDietPlanName(*in.Name)
		if err != nil {
			return nil, err
		}
		if name != plan.Name {
			other, err := s.plans.FindByName(ctx, kindergartenID, plan.AcademicYearID, name, plan.ID)
			if err != nil {
				return nil, err
			}
			if other != nil {
				return nil, ErrDietPlanNameConflict
			}
		}
		plan.Name = name
	}
	plan.DayNum = dayNum
	affected, err := s.plans.UpdateWithItems(ctx, plan, items)
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, ErrDietPlanNotFound
	}
	return &DietPlanSaveResult{ID: plan.ID, DayNum: dayNum, ItemCount: len(items)}, nil
}

// ClearPlan 清空全部明细但保留食谱本身，day_num 归零（契约 §五 clear）。
func (s *DietPlanService) ClearPlan(
	ctx context.Context,
	kindergartenID, id uint64,
) (*DietPlanClearResult, error) {
	if kindergartenID == 0 {
		return nil, dietPlanInvalid("未绑定园所，无法清空周食谱")
	}
	plan, err := s.plans.FindByIDWithItems(ctx, kindergartenID, id)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, ErrDietPlanNotFound
	}
	plan.DayNum = 0
	affected, err := s.plans.UpdateWithItems(ctx, plan, nil)
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, ErrDietPlanNotFound
	}
	return &DietPlanClearResult{ID: plan.ID, DayNum: 0}, nil
}

// DeletePlan 软删除周食谱并硬删明细；列表与详情不再返回。
func (s *DietPlanService) DeletePlan(ctx context.Context, kindergartenID, id uint64) error {
	if kindergartenID == 0 {
		return dietPlanInvalid("未绑定园所，无法删除周食谱")
	}
	affected, err := s.plans.Delete(ctx, kindergartenID, id)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrDietPlanNotFound
	}
	return nil
}

// ListTemplates 返回本园可用模板（只读，不分页）。
func (s *DietPlanService) ListTemplates(
	ctx context.Context,
	kindergartenID uint64,
) ([]DietTemplateListItem, error) {
	if kindergartenID == 0 {
		return nil, dietPlanInvalid("未绑定园所，无法读取模板")
	}
	rows, err := s.templates.List(ctx, kindergartenID)
	if err != nil {
		return nil, err
	}
	out := make([]DietTemplateListItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, DietTemplateListItem{
			ID:        row.ID,
			Name:      row.Name,
			DayNum:    row.DayNum,
			MealKeys:  splitDietTemplateMealKeys(row.MealKeys),
			ItemCount: row.ItemCount,
		})
	}
	return out, nil
}

// GetTemplate 返回模板详情；跨园或不存在一律 ErrDietTemplateNotFound（契约 404）。
func (s *DietPlanService) GetTemplate(
	ctx context.Context,
	kindergartenID, id uint64,
) (*DietTemplateDetail, error) {
	if kindergartenID == 0 {
		return nil, dietPlanInvalid("未绑定园所，无法读取模板")
	}
	tpl, err := s.templates.FindByIDWithItems(ctx, kindergartenID, id)
	if err != nil {
		return nil, err
	}
	if tpl == nil {
		return nil, ErrDietTemplateNotFound
	}
	items := make([]DietTemplateItemView, 0, len(tpl.Items))
	for _, item := range tpl.Items {
		items = append(items, DietTemplateItemView{
			Weekday:    item.Weekday,
			MealKey:    item.MealKey,
			RecipeName: item.RecipeName,
			FoodID:     item.FoodID,
			FoodName:   item.FoodName,
			Grams:      item.Grams,
			Sort:       item.Sort,
		})
	}
	return &DietTemplateDetail{
		ID:       tpl.ID,
		Name:     tpl.Name,
		DayNum:   tpl.DayNum,
		MealKeys: splitDietTemplateMealKeys(tpl.MealKeys),
		Items:    items,
	}, nil
}

// dietPlanSaveRow 是通过校验、等待合并与落库的一行明细。
type dietPlanSaveRow struct {
	date       time.Time
	weekday    int
	mealKey    domain.DietMealKey
	recipeName string
	recipeID   *uint64
	foodID     uint64
	foodLabel  string
	grams      float64
}

// buildDietPlanItems 在同一道菜（date + meal_key + recipe_name）内合并同种食物，
// 以服务端食物名为准生成营养快照，并按数组顺序在 (date, meal_key) 内重写 sort。
func buildDietPlanItems(
	ctx context.Context,
	rows []dietPlanSaveRow,
	foodByID map[uint64]*domain.Food,
) ([]domain.MealPlanItem, int, error) {
	items := make([]domain.MealPlanItem, 0, len(rows))
	index := make(map[string]int, len(rows))
	sortCounters := make(map[string]int, len(rows))
	days := make(map[string]bool, 7)

	for _, row := range rows {
		food := foodByID[row.foodID]
		dateKey := row.date.Format(dietPlanDateLayout)
		mergeKey := dateKey + "|" + string(row.mealKey) + "|" + row.recipeName + "|" +
			strconv.FormatUint(row.foodID, 10)
		if at, ok := index[mergeKey]; ok {
			items[at].Grams += row.grams
			days[dateKey] = true
			continue
		}
		nutrition, err := dietNutritionSnapshotJSON(food)
		if err != nil {
			return nil, 0, err
		}
		foodID := row.foodID
		groupKey := dateKey + "|" + string(row.mealKey)
		sortCounters[groupKey]++
		index[mergeKey] = len(items)
		items = append(items, domain.MealPlanItem{
			Date:              row.date,
			Weekday:           row.weekday,
			MealKey:           row.mealKey,
			RecipeName:        row.recipeName,
			RecipeID:          row.recipeID,
			FoodID:            &foodID,
			FoodName:          food.Name,
			Grams:             row.grams,
			NutritionSnapshot: nutrition,
			Sort:              sortCounters[groupKey],
		})
		days[dateKey] = true
	}
	return items, len(days), nil
}

// buildItemsFromTemplate 复制模板条目到新周期；缺失食物跳过并写入 warnings（契约规则 5）。
func (s *DietPlanService) buildItemsFromTemplate(
	ctx context.Context,
	kindergartenID uint64,
	start time.Time,
	templateID uint64,
) ([]domain.MealPlanItem, []string, error) {
	tpl, err := s.templates.FindByIDWithItems(ctx, kindergartenID, templateID)
	if err != nil {
		return nil, nil, err
	}
	if tpl == nil {
		return nil, nil, ErrDietTemplateNotFound
	}

	foodIDs := make([]uint64, 0, len(tpl.Items))
	for _, item := range tpl.Items {
		if item.FoodID != 0 {
			foodIDs = append(foodIDs, item.FoodID)
		}
	}
	foods, err := s.foods.FindByIDs(ctx, kindergartenID, foodIDs)
	if err != nil {
		return nil, nil, err
	}
	foodByID := make(map[uint64]*domain.Food, len(foods))
	for i := range foods {
		foodByID[foods[i].ID] = &foods[i]
	}

	items := make([]domain.MealPlanItem, 0, len(tpl.Items))
	warnings := make([]string, 0)
	for _, tplItem := range tpl.Items {
		if tplItem.Weekday < 1 || tplItem.Weekday > 7 {
			warnings = append(warnings, fmt.Sprintf(
				"模板「%s」中「%s」的星期无效，已跳过", tplItem.RecipeName, tplItem.FoodName))
			continue
		}
		if domain.DietMealSort(tplItem.MealKey) == 0 {
			warnings = append(warnings, fmt.Sprintf(
				"模板「%s」中「%s」的餐次无效，已跳过", tplItem.RecipeName, tplItem.FoodName))
			continue
		}
		food, ok := foodByID[tplItem.FoodID]
		if !ok {
			warnings = append(warnings, fmt.Sprintf(
				"模板第 %d 天「%s」的食物「%s」已不存在，已跳过",
				tplItem.Weekday, tplItem.RecipeName, tplItem.FoodName))
			continue
		}
		if math.IsNaN(tplItem.Grams) || math.IsInf(tplItem.Grams, 0) || tplItem.Grams <= 0 {
			warnings = append(warnings, fmt.Sprintf(
				"模板第 %d 天「%s」的食物「%s」克数无效，已跳过",
				tplItem.Weekday, tplItem.RecipeName, tplItem.FoodName))
			continue
		}
		nutrition, err := dietNutritionSnapshotJSON(food)
		if err != nil {
			return nil, nil, err
		}
		// 模板条目自带食物名快照：生成结果忠实复制模板内容；
		// 仅当模板快照为空时才回退到当前食物名。
		foodName := strings.TrimSpace(tplItem.FoodName)
		if foodName == "" {
			foodName = food.Name
		}
		foodID := food.ID
		items = append(items, domain.MealPlanItem{
			Date:              start.AddDate(0, 0, tplItem.Weekday-1),
			Weekday:           tplItem.Weekday,
			MealKey:           tplItem.MealKey,
			RecipeName:        tplItem.RecipeName,
			FoodID:            &foodID,
			FoodName:          foodName,
			Grams:             tplItem.Grams,
			NutritionSnapshot: nutrition,
			Sort:              tplItem.Sort,
		})
	}
	return items, warnings, nil
}

// buildItemsFromCopy 按星期几把源食谱映射到新周期，并用当前食物库重新固化营养快照。
// 源行引用的食物已被删除时跳过并写入 warnings（契约规则 6）。
func (s *DietPlanService) buildItemsFromCopy(
	ctx context.Context,
	kindergartenID uint64,
	start time.Time,
	sourcePlanID uint64,
) ([]domain.MealPlanItem, []string, error) {
	source, err := s.plans.FindByIDWithItems(ctx, kindergartenID, sourcePlanID)
	if err != nil {
		return nil, nil, err
	}
	if source == nil {
		return nil, nil, ErrDietPlanNotFound
	}

	foodIDs := make([]uint64, 0, len(source.Items))
	for _, item := range source.Items {
		if item.FoodID != nil && *item.FoodID != 0 {
			foodIDs = append(foodIDs, *item.FoodID)
		}
	}
	foods, err := s.foods.FindByIDs(ctx, kindergartenID, foodIDs)
	if err != nil {
		return nil, nil, err
	}
	foodByID := make(map[uint64]*domain.Food, len(foods))
	for i := range foods {
		foodByID[foods[i].ID] = &foods[i]
	}

	items := make([]domain.MealPlanItem, 0, len(source.Items))
	warnings := make([]string, 0)
	for _, sourceItem := range source.Items {
		weekday := sourceItem.Weekday
		if weekday < 1 || weekday > 7 {
			weekday = dietWeekdayFromDate(sourceItem.Date)
		}
		if sourceItem.FoodID == nil || *sourceItem.FoodID == 0 {
			warnings = append(warnings, fmt.Sprintf(
				"原食谱「%s」的食物「%s」未关联食物库，已跳过", sourceItem.RecipeName, sourceItem.FoodName))
			continue
		}
		food, ok := foodByID[*sourceItem.FoodID]
		if !ok {
			warnings = append(warnings, fmt.Sprintf(
				"原食谱「%s」的食物「%s」已删除，已跳过", sourceItem.RecipeName, sourceItem.FoodName))
			continue
		}
		if domain.DietMealSort(sourceItem.MealKey) == 0 {
			warnings = append(warnings, fmt.Sprintf(
				"原食谱「%s」中「%s」的餐次无效，已跳过", sourceItem.RecipeName, sourceItem.FoodName))
			continue
		}
		nutrition, err := dietNutritionSnapshotJSON(food)
		if err != nil {
			return nil, nil, err
		}
		foodID := food.ID
		items = append(items, domain.MealPlanItem{
			Date:              start.AddDate(0, 0, weekday-1),
			Weekday:           weekday,
			MealKey:           sourceItem.MealKey,
			RecipeName:        sourceItem.RecipeName,
			RecipeID:          sourceItem.RecipeID,
			FoodID:            &foodID,
			FoodName:          food.Name,
			Grams:             sourceItem.Grams,
			NutritionSnapshot: nutrition,
			Sort:              sourceItem.Sort,
		})
	}
	return items, warnings, nil
}

// buildDietConfigSnapshot 固化当前配置 + 按班级人数加权的 11 项 DRIs 目标值。
func (s *DietPlanService) buildDietConfigSnapshot(
	ctx context.Context,
	kindergartenID uint64,
) (string, error) {
	snapshot, err := s.config.BuildConfigSnapshot(ctx, kindergartenID)
	if err != nil {
		return "", err
	}
	targets, weights, err := s.buildDietDriSnapshot(ctx, kindergartenID)
	if err != nil {
		return "", err
	}
	snapshot.DriTargets = targets
	snapshot.DriWeights = weights
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func (s *DietPlanService) findAcademicYear(
	ctx context.Context,
	kindergartenID, academicYearID uint64,
) (*domain.AcademicYear, error) {
	years, err := s.years.ListByKindergarten(ctx, kindergartenID)
	if err != nil {
		return nil, err
	}
	for i := range years {
		if years[i].ID == academicYearID {
			return &years[i], nil
		}
	}
	return nil, nil
}

func validateDietPlanName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", dietPlanInvalid("食谱名称不能为空")
	}
	if utf8.RuneCountInString(name) > 128 {
		return "", dietPlanInvalid("食谱名称不能超过 128 个字符")
	}
	return name, nil
}

func validateDietPlanCreatedMethod(
	method domain.DietPlanCreatedMethod,
) (domain.DietPlanCreatedMethod, error) {
	switch method {
	case domain.DietPlanCreatedByCustom, domain.DietPlanCreatedByTemplate, domain.DietPlanCreatedByCopy:
		return method, nil
	default:
		return "", dietPlanInvalid("创建方式无效，只支持自定义、选用模板或复制历史")
	}
}

// validateDietPlanMethodFields 校验三种创建方式的附加字段互斥（契约校验第 3 条）。
func validateDietPlanMethodFields(
	method domain.DietPlanCreatedMethod,
	templateID, sourcePlanID *uint64,
) error {
	switch method {
	case domain.DietPlanCreatedByCustom:
		if templateID != nil || sourcePlanID != nil {
			return dietPlanInvalid("自定义创建不接受模板或来源食谱")
		}
	case domain.DietPlanCreatedByTemplate:
		if templateID == nil || *templateID == 0 {
			return dietPlanInvalid("选用模板创建必须指定 template_id")
		}
		if sourcePlanID != nil {
			return dietPlanInvalid("选用模板创建不接受 source_plan_id")
		}
	case domain.DietPlanCreatedByCopy:
		if sourcePlanID == nil || *sourcePlanID == 0 {
			return dietPlanInvalid("复制历史创建必须指定 source_plan_id")
		}
		if templateID != nil {
			return dietPlanInvalid("复制历史创建不接受 template_id")
		}
	}
	return nil
}

// parseDietPlanPeriod 校验周期：开始日期必须是周一，结束日期必须等于开始 + 6 天。
func parseDietPlanPeriod(startRaw, endRaw string) (time.Time, error) {
	start, err := parseDietPlanDate(startRaw)
	if err != nil {
		return time.Time{}, dietPlanInvalid("开始日期格式必须为 YYYY-MM-DD")
	}
	if start.Weekday() != time.Monday {
		return time.Time{}, dietPlanInvalid("开始日期必须是周一")
	}
	end, err := parseDietPlanDate(endRaw)
	if err != nil {
		return time.Time{}, dietPlanInvalid("结束日期格式必须为 YYYY-MM-DD")
	}
	if !end.Equal(start.AddDate(0, 0, 6)) {
		return time.Time{}, dietPlanInvalid("结束日期必须等于开始日期 + 6 天")
	}
	return start, nil
}

func parseDietPlanDate(raw string) (time.Time, error) {
	return time.Parse(dietPlanDateLayout, strings.TrimSpace(raw))
}

// dietWeekdayFromDate 把日期转成 1=周一 … 7=周日。
func dietWeekdayFromDate(date time.Time) int {
	weekday := int(date.Weekday())
	if weekday == 0 {
		return 7
	}
	return weekday
}

func countDietPlanDays(items []domain.MealPlanItem) int {
	days := make(map[string]bool, 7)
	for _, item := range items {
		days[item.Date.Format(dietPlanDateLayout)] = true
	}
	return len(days)
}

func dietPlanListItemView(plan *domain.MealPlan) DietPlanListItem {
	return DietPlanListItem{
		ID:                 plan.ID,
		Name:               plan.Name,
		AcademicYearID:     plan.AcademicYearID,
		StartDate:          plan.StartDate.Format(dietPlanDateLayout),
		EndDate:            plan.EndDate.Format(dietPlanDateLayout),
		DayNum:             plan.DayNum,
		WeekOfYear:         plan.WeekOfYear,
		CreatedMethod:      plan.CreatedMethod,
		CreatedMethodLabel: dietPlanCreatedMethodLabel(plan.CreatedMethod),
	}
}

func dietPlanCreatedMethodLabel(method domain.DietPlanCreatedMethod) string {
	switch method {
	case domain.DietPlanCreatedByCustom:
		return "自定义"
	case domain.DietPlanCreatedByTemplate:
		return "选用模板"
	case domain.DietPlanCreatedByCopy:
		return "复制历史"
	default:
		return ""
	}
}

func parseDietConfigSnapshot(raw string) (domain.DietConfigSnapshot, error) {
	var snapshot domain.DietConfigSnapshot
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "null" {
		return snapshot, nil
	}
	if err := json.Unmarshal([]byte(trimmed), &snapshot); err != nil {
		return snapshot, fmt.Errorf("diet plan: parse config snapshot: %w", err)
	}
	return snapshot, nil
}

func parseDietNutritionSnapshot(raw string) (domain.DietNutritionSnapshot, error) {
	var snapshot domain.DietNutritionSnapshot
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "null" {
		return snapshot, nil
	}
	if err := json.Unmarshal([]byte(trimmed), &snapshot); err != nil {
		return snapshot, fmt.Errorf("diet plan: parse nutrition snapshot: %w", err)
	}
	return snapshot, nil
}

func dietNutritionSnapshotJSON(food *domain.Food) (string, error) {
	raw, err := json.Marshal(food.NutritionSnapshot())
	if err != nil {
		return "", fmt.Errorf("diet plan: marshal nutrition snapshot: %w", err)
	}
	return string(raw), nil
}

// dietEnabledMeals 从配置快照取已启用餐次集合。快照缺餐次时为空集：
// 保存会被拒绝，而不是放行任意餐次（保存规则 2 要求餐次必须在快照启用集合内）。
func dietEnabledMeals(snapshot domain.DietConfigSnapshot) map[domain.DietMealKey]bool {
	enabled := make(map[domain.DietMealKey]bool, len(snapshot.Meals))
	for _, meal := range snapshot.Meals {
		if meal.Enabled {
			enabled[meal.MealKey] = true
		}
	}
	return enabled
}

func dietPlanFoodLabel(foodID uint64, fallbackName string) string {
	if name := strings.TrimSpace(fallbackName); name != "" {
		return name
	}
	if foodID == 0 {
		return "未选择"
	}
	return "食物#" + strconv.FormatUint(foodID, 10)
}

func splitDietTemplateMealKeys(raw string) []string {
	out := make([]string, 0, 6)
	for _, part := range strings.Split(raw, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}
