package service

import (
	"context"
	"errors"
	"math"
	"sort"
	"strings"
	"unicode/utf8"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/pkg/pagination"
)

// DietGroupObjectService 按学年管理的集体对象（基本参数）读写编排。
//
// 对象是供餐配置的唯一边界（design D2）：餐次、占比、口径与系数都挂在对象上，
// 周食谱创建时据此固化快照。名称唯一、班级必选、餐次规则都在写入前校验，
// 校验失败不产生任何写入（design D7/D8）。
type DietGroupObjectService struct {
	objects *repository.DietGroupObjectRepo
	years   *repository.AcademicYearRepo
	classes *repository.ClassRepo
}

func NewDietGroupObjectService(
	objects *repository.DietGroupObjectRepo,
	years *repository.AcademicYearRepo,
	classes *repository.ClassRepo,
) *DietGroupObjectService {
	return &DietGroupObjectService{objects: objects, years: years, classes: classes}
}

// ErrDietGroupObjectNotFound 表示对象不存在或不属于当前园所；handler 映射为 404。
var ErrDietGroupObjectNotFound = errors.New("DIET_GROUP_OBJECT_NOT_FOUND")

// DietGroupObjectInput 是新增/修改集体对象的入参，只含管理员可改的字段。
// 餐次名称、排序与 DRIs 标准由服务端固定写入；学年在修改时只读（design D5）。
type DietGroupObjectInput struct {
	AcademicYearID uint64
	Name           string
	CareType       domain.DietCareType
	FoodInputType  domain.DietFoodInputType
	RatioMode      domain.DietRatioMode
	ClassIDs       []uint64
	Meals          []DietMealConfigMealInput
}

// DietGroupObjectClassView 是就餐班级的展示项。班级是动态数据，
// 只存 class_id，读取时 join 当前班级表；已删除的班级不再返回（design Risks）。
type DietGroupObjectClassView struct {
	ID    uint64 `json:"id"`
	Name  string `json:"name"`
	Grade string `json:"grade"`
}

// DietGroupObjectListItem 是列表行渲染所需字段（契约 23-diet.md 列表裁剪）。
// MealNames 只含已启用餐次：列表「餐次」列展示实际供餐的餐次。
type DietGroupObjectListItem struct {
	ID                 uint64                     `json:"id"`
	Name               string                     `json:"name"`
	AcademicYearID     uint64                     `json:"academic_year_id"`
	CareType           domain.DietCareType        `json:"care_type"`
	CareTypeLabel      string                     `json:"care_type_label"`
	FoodInputType      domain.DietFoodInputType   `json:"food_input_type"`
	FoodInputTypeLabel string                     `json:"food_input_type_label"`
	PolicyName         string                     `json:"policy_name"`
	MealNames          []string                   `json:"meal_names"`
	Classes            []DietGroupObjectClassView `json:"classes"`
}

// DietGroupObjectDetail 是详情：列表字段 + 比例模式、系数与完整餐次配置。
// 不重复返回 class_ids——classes[].id 已可派生的字段只在输入侧出现（design D7）。
type DietGroupObjectDetail struct {
	DietGroupObjectListItem
	RatioMode      domain.DietRatioMode     `json:"ratio_mode"`
	RatioModeLabel string                   `json:"ratio_mode_label"`
	Coefficient    float64                  `json:"coefficient"`
	Meals          []DietMealConfigMealView `json:"meals"`
}

// List 返回本园某学年的集体对象。academic_year_id 缺省取当前学年（is_current）；
// 没有当前学年时返回空列表，不在读取路径创建对象（TC-NEW-DIE-0065）。
func (s *DietGroupObjectService) List(
	ctx context.Context,
	kindergartenID, academicYearID uint64,
	keyword string,
	pq pagination.Query,
) ([]DietGroupObjectListItem, int64, int, int, error) {
	if kindergartenID == 0 {
		return nil, 0, 0, 0, dietConfigInvalid("未绑定园所，无法读取集体对象")
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
			return []DietGroupObjectListItem{}, 0, pn.Page, pn.PageSize, nil
		}
		yearID = current.ID
	}

	rows, total, err := s.objects.List(
		ctx, kindergartenID, yearID, keyword, pn.Offset(), pn.PageSize)
	if err != nil {
		return nil, 0, 0, 0, err
	}

	// 同一页内班级通常重复出现，按 ID 缓存避免逐行回查。
	cache := make(map[uint64]*domain.Class)
	out := make([]DietGroupObjectListItem, 0, len(rows))
	for i := range rows {
		item, err := s.listItem(ctx, kindergartenID, &rows[i], cache)
		if err != nil {
			return nil, 0, 0, 0, err
		}
		out = append(out, item)
	}
	return out, total, pn.Page, pn.PageSize, nil
}

// Get 返回对象详情；跨园或不存在一律 ErrDietGroupObjectNotFound（契约 404）。
func (s *DietGroupObjectService) Get(
	ctx context.Context,
	kindergartenID, id uint64,
) (*DietGroupObjectDetail, error) {
	if kindergartenID == 0 {
		return nil, dietConfigInvalid("未绑定园所，无法读取集体对象")
	}
	obj, err := s.objects.Get(ctx, kindergartenID, id)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, ErrDietGroupObjectNotFound
	}
	return s.detailView(ctx, kindergartenID, obj)
}

// Create 校验并新增集体对象，返回落库后的详情。
func (s *DietGroupObjectService) Create(
	ctx context.Context,
	kindergartenID uint64,
	in DietGroupObjectInput,
) (*DietGroupObjectDetail, error) {
	if kindergartenID == 0 {
		return nil, dietConfigInvalid("未绑定园所，无法新增集体对象")
	}
	obj, err := s.buildObject(ctx, kindergartenID, in, nil)
	if err != nil {
		return nil, err
	}
	if err := s.objects.Create(ctx, obj); err != nil {
		return nil, err
	}
	return s.Get(ctx, kindergartenID, obj.ID)
}

// Update 校验并修改集体对象。学年在修改时只读：仍校验提交的学年属于本园，
// 但落库沿用对象原学年，避免被周食谱引用的对象被挪到别的学年（design D3/D5）。
func (s *DietGroupObjectService) Update(
	ctx context.Context,
	kindergartenID, id uint64,
	in DietGroupObjectInput,
) (*DietGroupObjectDetail, error) {
	if kindergartenID == 0 {
		return nil, dietConfigInvalid("未绑定园所，无法修改集体对象")
	}
	existing, err := s.objects.Get(ctx, kindergartenID, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrDietGroupObjectNotFound
	}
	obj, err := s.buildObject(ctx, kindergartenID, in, existing)
	if err != nil {
		return nil, err
	}
	if err := s.objects.Update(ctx, obj); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDietGroupObjectNotFound
		}
		return nil, err
	}
	return s.Get(ctx, kindergartenID, obj.ID)
}

// BuildConfigSnapshot 返回该对象的配置快照，供创建周食谱时写入
// MealPlan.config_snapshot。对象改名/改班级后旧食谱仍按快照显示与计算（design D2）。
// DRIs 目标值依赖就餐班级人数加权，由周食谱服务补（本方法留空）。
func (s *DietGroupObjectService) BuildConfigSnapshot(
	ctx context.Context,
	kindergartenID, id uint64,
) (*domain.DietConfigSnapshot, error) {
	if kindergartenID == 0 {
		return nil, dietConfigInvalid("未绑定园所，无法读取集体对象")
	}
	obj, err := s.objects.Get(ctx, kindergartenID, id)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, ErrDietGroupObjectNotFound
	}

	snapshot := &domain.DietConfigSnapshot{
		GroupObjectID:       obj.ID,
		GroupObjectName:     obj.Name,
		GroupObjectClassIDs: make([]uint64, 0, len(obj.Classes)),
		CareType:            obj.CareType,
		FoodInputType:       obj.FoodInputType,
		RatioMode:           normalizeRatioModeForRead(obj.RatioMode),
	}
	for _, link := range obj.Classes {
		snapshot.GroupObjectClassIDs = append(snapshot.GroupObjectClassIDs, link.ClassID)
	}
	sort.Slice(snapshot.GroupObjectClassIDs, func(i, j int) bool {
		return snapshot.GroupObjectClassIDs[i] < snapshot.GroupObjectClassIDs[j]
	})

	byKey := make(map[domain.DietMealKey]domain.DietGroupObjectMeal, len(obj.Meals))
	for _, meal := range obj.Meals {
		byKey[meal.MealKey] = meal
	}
	enabledSum := 0.0
	for _, key := range domain.DietMealKeyOrder {
		meal, ok := byKey[key]
		if !ok {
			continue
		}
		snapshot.Meals = append(snapshot.Meals, domain.DietMealSnapshot{
			MealKey:       key,
			Name:          dietGroupObjectMealName(meal),
			Time:          meal.Time,
			EnergyPercent: meal.EnergyPercent,
			Sort:          domain.DietMealSort(key),
			Enabled:       meal.Enabled,
		})
		if meal.Enabled {
			enabledSum += meal.EnergyPercent
		}
	}
	snapshot.Coefficient = math.Round(enabledSum/100*10000) / 10000
	return snapshot, nil
}

// buildObject 校验入参并组装可落库的对象；existing 非空表示修改。
// 所有校验都在写库前完成，失败时原对象保持不变（TC-NEW-DIE-0067/0068）。
func (s *DietGroupObjectService) buildObject(
	ctx context.Context,
	kindergartenID uint64,
	in DietGroupObjectInput,
	existing *domain.DietGroupObject,
) (*domain.DietGroupObject, error) {
	name, err := validateDietGroupObjectName(in.Name)
	if err != nil {
		return nil, err
	}
	yearID, err := s.resolveGroupObjectYear(ctx, kindergartenID, in.AcademicYearID, existing)
	if err != nil {
		return nil, err
	}
	excludeID := uint64(0)
	if existing != nil {
		excludeID = existing.ID
	}
	sameName, err := s.objects.CountByName(ctx, kindergartenID, yearID, name, excludeID)
	if err != nil {
		return nil, err
	}
	if sameName > 0 {
		return nil, dietConfigInvalid("该学年已存在同名集体对象")
	}
	classIDs, err := s.validateGroupObjectClassIDs(ctx, kindergartenID, in.ClassIDs)
	if err != nil {
		return nil, err
	}
	ratioMode, ordered, err := validateDietMealConfig(
		in.CareType, in.FoodInputType, in.RatioMode, in.Meals)
	if err != nil {
		return nil, err
	}

	obj := &domain.DietGroupObject{
		KindergartenID: kindergartenID,
		AcademicYearID: yearID,
		Name:           name,
		CareType:       in.CareType,
		FoodInputType:  in.FoodInputType,
		RatioMode:      ratioMode,
		// DRIs 标准固定 DRIs2000，不接受客户端输入（design D5/D6）。
		PolicyName: dietGroupObjectPolicyName,
	}
	if existing != nil {
		obj.ID = existing.ID
	}
	obj.Meals = make([]domain.DietGroupObjectMeal, 0, len(ordered))
	for _, meal := range ordered {
		obj.Meals = append(obj.Meals, domain.DietGroupObjectMeal{
			KindergartenID: kindergartenID,
			MealKey:        meal.MealKey,
			Name:           dietMealName(meal.MealKey),
			Time:           meal.Time,
			EnergyPercent:  meal.EnergyPercent,
			Sort:           domain.DietMealSort(meal.MealKey),
			Enabled:        meal.Enabled,
		})
	}
	obj.Classes = make([]domain.DietGroupObjectClass, 0, len(classIDs))
	for _, classID := range classIDs {
		obj.Classes = append(obj.Classes, domain.DietGroupObjectClass{
			KindergartenID: kindergartenID,
			ClassID:        classID,
		})
	}
	return obj, nil
}

// resolveGroupObjectYear 校验学年归属并返回落库学年。
//
// 新增时学年必填；修改时沿用对象原学年（design D5：修改学年只读），
// 但仍校验提交值属于本园，挡住跨租户写入。
func (s *DietGroupObjectService) resolveGroupObjectYear(
	ctx context.Context,
	kindergartenID, requestedYearID uint64,
	existing *domain.DietGroupObject,
) (uint64, error) {
	yearID := requestedYearID
	if yearID == 0 && existing != nil {
		yearID = existing.AcademicYearID
	}
	if yearID == 0 {
		return 0, dietConfigInvalid("请选择学年")
	}
	year, err := s.findAcademicYear(ctx, kindergartenID, yearID)
	if err != nil {
		return 0, err
	}
	if year == nil {
		return 0, dietConfigInvalid("学年不存在或不属于本园")
	}
	if existing != nil {
		return existing.AcademicYearID, nil
	}
	return year.ID, nil
}

// findAcademicYear 按 ID 在园所学年列表中查找；跨园与不存在返回 nil, nil。
func (s *DietGroupObjectService) findAcademicYear(
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

// validateGroupObjectClassIDs 去重并校验每个班级都属于本园，至少保留一个。
// 班级按学年过滤、按年级分组属于前端交互（design D5），后端只保证租户边界。
func (s *DietGroupObjectService) validateGroupObjectClassIDs(
	ctx context.Context,
	kindergartenID uint64,
	classIDs []uint64,
) ([]uint64, error) {
	seen := make(map[uint64]bool, len(classIDs))
	out := make([]uint64, 0, len(classIDs))
	for _, classID := range classIDs {
		if seen[classID] {
			continue
		}
		seen[classID] = true
		class, err := s.classes.FindByID(ctx, kindergartenID, classID)
		if err != nil {
			return nil, err
		}
		if class == nil {
			return nil, dietConfigInvalid("就餐班级不存在或不属于本园")
		}
		out = append(out, classID)
	}
	if len(out) == 0 {
		return nil, dietConfigInvalid("请至少选择一个就餐班级")
	}
	return out, nil
}

func (s *DietGroupObjectService) listItem(
	ctx context.Context,
	kindergartenID uint64,
	obj *domain.DietGroupObject,
	cache map[uint64]*domain.Class,
) (DietGroupObjectListItem, error) {
	classes, err := s.classViews(ctx, kindergartenID, obj.Classes, cache)
	if err != nil {
		return DietGroupObjectListItem{}, err
	}
	return DietGroupObjectListItem{
		ID:                 obj.ID,
		Name:               obj.Name,
		AcademicYearID:     obj.AcademicYearID,
		CareType:           obj.CareType,
		CareTypeLabel:      dietCareTypeLabel(obj.CareType),
		FoodInputType:      obj.FoodInputType,
		FoodInputTypeLabel: dietFoodInputTypeLabel(obj.FoodInputType),
		PolicyName:         obj.PolicyName,
		MealNames:          dietGroupObjectMealNames(obj.Meals),
		Classes:            classes,
	}, nil
}

func (s *DietGroupObjectService) detailView(
	ctx context.Context,
	kindergartenID uint64,
	obj *domain.DietGroupObject,
) (*DietGroupObjectDetail, error) {
	item, err := s.listItem(ctx, kindergartenID, obj, make(map[uint64]*domain.Class))
	if err != nil {
		return nil, err
	}

	byKey := make(map[domain.DietMealKey]domain.DietGroupObjectMeal, len(obj.Meals))
	for _, meal := range obj.Meals {
		byKey[meal.MealKey] = meal
	}
	meals := make([]DietMealConfigMealView, 0, len(domain.DietMealKeyOrder))
	enabledSum := 0.0
	for _, key := range domain.DietMealKeyOrder {
		meal, ok := byKey[key]
		if !ok {
			continue
		}
		meals = append(meals, DietMealConfigMealView{
			MealKey:       key,
			Name:          dietGroupObjectMealName(meal),
			Time:          meal.Time,
			EnergyPercent: meal.EnergyPercent,
			Sort:          domain.DietMealSort(key),
			Enabled:       meal.Enabled,
		})
		if meal.Enabled {
			enabledSum += meal.EnergyPercent
		}
	}
	ratioMode := normalizeRatioModeForRead(obj.RatioMode)
	return &DietGroupObjectDetail{
		DietGroupObjectListItem: item,
		RatioMode:               ratioMode,
		RatioModeLabel:          dietRatioModeLabel(ratioMode),
		Coefficient:             math.Round(enabledSum/100*10000) / 10000,
		Meals:                   meals,
	}, nil
}

// classViews 把 class_id 关联解析为班级展示项；已删除的班级直接跳过。
func (s *DietGroupObjectService) classViews(
	ctx context.Context,
	kindergartenID uint64,
	links []domain.DietGroupObjectClass,
	cache map[uint64]*domain.Class,
) ([]DietGroupObjectClassView, error) {
	out := make([]DietGroupObjectClassView, 0, len(links))
	for _, link := range links {
		class, ok := cache[link.ClassID]
		if !ok {
			found, err := s.classes.FindByID(ctx, kindergartenID, link.ClassID)
			if err != nil {
				return nil, err
			}
			class = found
			cache[link.ClassID] = class
		}
		if class == nil {
			continue
		}
		out = append(out, DietGroupObjectClassView{
			ID:    class.ID,
			Name:  class.Name,
			Grade: class.Grade,
		})
	}
	return out, nil
}

// validateDietGroupObjectName 名称去首尾空白后 1..20 个字符（design D7、TC-NEW-DIE-0067）。
func validateDietGroupObjectName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", dietConfigInvalid("请设置集体对象名称")
	}
	if utf8.RuneCountInString(name) > 20 {
		return "", dietConfigInvalid("集体对象名称不能超过 20 个字符")
	}
	return name, nil
}

// dietGroupObjectMealNames 按固定餐次顺序返回已启用餐次的名称。
func dietGroupObjectMealNames(meals []domain.DietGroupObjectMeal) []string {
	byKey := make(map[domain.DietMealKey]domain.DietGroupObjectMeal, len(meals))
	for _, meal := range meals {
		byKey[meal.MealKey] = meal
	}
	out := make([]string, 0, len(meals))
	for _, key := range domain.DietMealKeyOrder {
		meal, ok := byKey[key]
		if !ok || !meal.Enabled {
			continue
		}
		out = append(out, dietGroupObjectMealName(meal))
	}
	return out
}

// dietGroupObjectMealName 以落库名称优先；历史行名称为空时回退到固定餐次名。
func dietGroupObjectMealName(meal domain.DietGroupObjectMeal) string {
	if name := strings.TrimSpace(meal.Name); name != "" {
		return name
	}
	return dietMealName(meal.MealKey)
}
