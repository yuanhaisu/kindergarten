package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/pkg/pagination"
)

// DietRecipeValidationError 表示菜谱入参或业务校验失败。
// handler 用 errors.As 识别并映射为项目统一的 422 包络；Message 可直接展示给管理员。
type DietRecipeValidationError struct {
	Message string
}

func (e *DietRecipeValidationError) Error() string { return e.Message }

func dietRecipeInvalid(format string, args ...any) error {
	return &DietRecipeValidationError{Message: fmt.Sprintf(format, args...)}
}

// 菜谱访问与保存的业务错误。跨园访问在仓储层查不到，统一按 NotFound 处理（契约 404）。
var (
	ErrDietRecipeNotFound     = errors.New("DIET_RECIPE_NOT_FOUND")
	ErrDietRecipeNameConflict = errors.New("DIET_RECIPE_NAME_CONFLICT")
)

const (
	// 契约 §四 suggest：limit 默认 18、上限 100。
	dietRecipeSuggestDefaultLimit = 18
	dietRecipeSuggestMaxLimit     = 100
)

// DietRecipeItemInput 是保存/修改菜谱时提交的一行配料。
type DietRecipeItemInput struct {
	FoodID uint64
	Grams  float64
}

// DietRecipeSaveInput 是 POST /admin/diet/recipes 的入参。
//
// Replace=false 且名称已存在时返回名称冲突，由前端弹「替换 / 仅本次」；
// Replace=true 时覆盖原配方（无同名时等同新增）。「仅本次」由前端不调用本接口实现。
type DietRecipeSaveInput struct {
	Name    string
	Items   []DietRecipeItemInput
	Replace bool
}

// DietRecipeUpdateInput 是 PUT /admin/diet/recipes/:id 的入参。
// Name/Items 为 nil 表示该字段未提交；Items 非 nil 即整体替换。
type DietRecipeUpdateInput struct {
	Name  *string
	Items *[]DietRecipeItemInput
}

// DietRecipeListItem 是列表/联想行：只有名称、拼音码与配料数，不含配料（契约 §四）。
type DietRecipeListItem struct {
	ID        uint64 `json:"id"`
	Name      string `json:"name"`
	SpellCode string `json:"spell_code"`
	ItemCount int64  `json:"item_count"`
}

// DietRecipeItemView 是菜谱详情中的配料行。
// Nutrients 为该食物当前 27 项营养（编辑器载入配方时一次取全）；缺失值为 null。
type DietRecipeItemView struct {
	FoodID    uint64                `json:"food_id"`
	FoodName  string                `json:"food_name"`
	Grams     float64               `json:"grams"`
	Sort      int                   `json:"sort"`
	Nutrients DietFoodNutrientsView `json:"nutrients"`
}

// DietRecipeDetail 是菜品详情（配料 + 每种食物当前营养）。
type DietRecipeDetail struct {
	ID        uint64               `json:"id"`
	Name      string               `json:"name"`
	SpellCode string               `json:"spell_code"`
	Items     []DietRecipeItemView `json:"items"`
}

// DietRecipeSaveResult 是新增/替换菜谱的响应。
// Created 只供 handler 选择 201/200，不对外返回。
type DietRecipeSaveResult struct {
	ID        uint64 `json:"id"`
	Name      string `json:"name"`
	SpellCode string `json:"spell_code"`
	ItemCount int64  `json:"item_count"`
	Created   bool   `json:"-"`
}

// DietRecipeMergeResult 是合并重复菜谱的响应；MergedItemCount 是合并后目标菜谱的配料数。
type DietRecipeMergeResult struct {
	TargetID        uint64 `json:"target_id"`
	MergedItemCount int64  `json:"merged_item_count"`
}

// DietRecipeService 园所菜谱库。
//
// 名称唯一由服务层保证、不建唯一索引：软删除行仍会占唯一键，删除后同名重建会失败
// （design D2）。菜谱不存营养快照，详情按当前食物库实时取营养；修改/删除菜谱
// 不影响历史周食谱（历史明细存自己的菜名与营养快照，ADR-0006）。
type DietRecipeService struct {
	recipes *repository.DietRecipeRepo
	foods   *repository.DietFoodRepo
}

func NewDietRecipeService(
	recipes *repository.DietRecipeRepo,
	foods *repository.DietFoodRepo,
) *DietRecipeService {
	return &DietRecipeService{recipes: recipes, foods: foods}
}

// ListRecipes 返回本园菜谱列表（名称/拼音码/配料数），keyword 按名称或拼音码匹配。
func (s *DietRecipeService) ListRecipes(
	ctx context.Context,
	kindergartenID uint64,
	keyword string,
	pq pagination.Query,
) ([]DietRecipeListItem, int64, int, int, error) {
	if kindergartenID == 0 {
		return nil, 0, 0, 0, dietRecipeInvalid("未绑定园所，无法读取菜谱库")
	}
	pn := pq
	pn.Normalize()

	rows, total, err := s.recipes.List(ctx, kindergartenID, keyword, pn.Offset(), pn.PageSize)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	out := make([]DietRecipeListItem, 0, len(rows))
	for i := range rows {
		out = append(out, dietRecipeListItemView(rows[i]))
	}
	return out, total, pn.Page, pn.PageSize, nil
}

// GetRecipe 返回菜谱详情；跨园或不存在一律 ErrDietRecipeNotFound（契约 404）。
//
// 配料只按本园食物库解析营养：kg=0 平台预置行与跨园食物不会命中，
// 对应行的营养全部为 null，不得回退去读平台源（ADR-0005）。
func (s *DietRecipeService) GetRecipe(
	ctx context.Context,
	kindergartenID, id uint64,
) (*DietRecipeDetail, error) {
	if kindergartenID == 0 {
		return nil, dietRecipeInvalid("未绑定园所，无法读取菜谱库")
	}
	recipe, err := s.recipes.FindByIDWithItems(ctx, kindergartenID, id)
	if err != nil {
		return nil, err
	}
	if recipe == nil {
		return nil, ErrDietRecipeNotFound
	}

	foodIDs := make([]uint64, 0, len(recipe.Items))
	for _, item := range recipe.Items {
		foodIDs = append(foodIDs, item.FoodID)
	}
	foods, err := s.foods.FindByIDs(ctx, kindergartenID, foodIDs)
	if err != nil {
		return nil, err
	}
	foodByID := make(map[uint64]*domain.Food, len(foods))
	for i := range foods {
		foodByID[foods[i].ID] = &foods[i]
	}

	items := make([]DietRecipeItemView, 0, len(recipe.Items))
	for _, item := range recipe.Items {
		view := DietRecipeItemView{
			FoodID: item.FoodID,
			Grams:  item.Grams,
			Sort:   item.Sort,
		}
		if food, ok := foodByID[item.FoodID]; ok {
			view.FoodName = food.Name
			// 复用食物详情的 27 项映射，避免第二份字段清单随模型漂移。
			view.Nutrients = dietFoodDetailView(food, nil).Nutrients
		}
		items = append(items, view)
	}
	return &DietRecipeDetail{
		ID:        recipe.ID,
		Name:      recipe.Name,
		SpellCode: recipe.SpellCode,
		Items:     items,
	}, nil
}

// SaveRecipe 保存菜谱：名称不存在则新增；已存在且 replace=false 返回名称冲突；
// replace=true 覆盖原配方（无同名时等同新增）。
func (s *DietRecipeService) SaveRecipe(
	ctx context.Context,
	kindergartenID uint64,
	in DietRecipeSaveInput,
) (*DietRecipeSaveResult, error) {
	if kindergartenID == 0 {
		return nil, dietRecipeInvalid("未绑定园所，无法保存菜谱")
	}
	name, err := validateDietRecipeName(in.Name)
	if err != nil {
		return nil, err
	}
	items, err := s.normalizeRecipeItems(ctx, kindergartenID, in.Items)
	if err != nil {
		return nil, err
	}
	spellCode := dietSpellCode(name)

	existing, err := s.recipes.FindByName(ctx, kindergartenID, name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if !in.Replace {
			return nil, ErrDietRecipeNameConflict
		}
		existing.Name = name
		existing.SpellCode = spellCode
		existing.Items = items
		if err := s.recipes.Update(ctx, existing); err != nil {
			return nil, err
		}
		return &DietRecipeSaveResult{
			ID:        existing.ID,
			Name:      name,
			SpellCode: spellCode,
			ItemCount: int64(len(items)),
		}, nil
	}

	recipe := &domain.Recipe{
		KindergartenID: kindergartenID,
		Name:           name,
		SpellCode:      spellCode,
		Items:          items,
	}
	if err := s.recipes.Create(ctx, recipe); err != nil {
		return nil, err
	}
	return &DietRecipeSaveResult{
		ID:        recipe.ID,
		Name:      name,
		SpellCode: spellCode,
		ItemCount: int64(len(items)),
		Created:   true,
	}, nil
}

// UpdateRecipe 改名与/或改配方（菜谱库管理页）。items 提交即整体替换；
// 改名撞上本园其他菜谱名称返回名称冲突（契约 409）。
func (s *DietRecipeService) UpdateRecipe(
	ctx context.Context,
	kindergartenID, id uint64,
	in DietRecipeUpdateInput,
) error {
	if kindergartenID == 0 {
		return dietRecipeInvalid("未绑定园所，无法修改菜谱")
	}
	recipe, err := s.recipes.FindByIDWithItems(ctx, kindergartenID, id)
	if err != nil {
		return err
	}
	if recipe == nil {
		return ErrDietRecipeNotFound
	}

	changed := false
	if in.Name != nil {
		name, err := validateDietRecipeName(*in.Name)
		if err != nil {
			return err
		}
		if name != recipe.Name {
			other, err := s.recipes.FindByName(ctx, kindergartenID, name)
			if err != nil {
				return err
			}
			if other != nil && other.ID != recipe.ID {
				return ErrDietRecipeNameConflict
			}
			recipe.Name = name
			recipe.SpellCode = dietSpellCode(name)
		}
		changed = true
	}
	if in.Items != nil {
		items, err := s.normalizeRecipeItems(ctx, kindergartenID, *in.Items)
		if err != nil {
			return err
		}
		recipe.Items = items
		changed = true
	}
	if !changed {
		return dietRecipeInvalid("没有需要修改的字段")
	}
	return s.recipes.Update(ctx, recipe)
}

// DeleteRecipe 软删除菜谱并硬删其配料；历史周食谱保持原菜名、克数与营养快照。
func (s *DietRecipeService) DeleteRecipe(ctx context.Context, kindergartenID, id uint64) error {
	if kindergartenID == 0 {
		return dietRecipeInvalid("未绑定园所，无法删除菜谱")
	}
	affected, err := s.recipes.Delete(ctx, kindergartenID, id)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrDietRecipeNotFound
	}
	return nil
}

// MergeRecipes 合并重复菜谱：源菜谱的同种食物克数累加到目标，然后删除源菜谱。
// 源与目标都必须属于本园，否则 404；source_id == target_id 是参数错误（422）。
func (s *DietRecipeService) MergeRecipes(
	ctx context.Context,
	kindergartenID, sourceID, targetID uint64,
) (*DietRecipeMergeResult, error) {
	if kindergartenID == 0 {
		return nil, dietRecipeInvalid("未绑定园所，无法合并菜谱")
	}
	if sourceID == 0 || targetID == 0 {
		return nil, dietRecipeInvalid("菜谱 ID 无效")
	}
	if sourceID == targetID {
		return nil, dietRecipeInvalid("不能把菜谱合并到自身")
	}
	source, err := s.recipes.FindByIDWithItems(ctx, kindergartenID, sourceID)
	if err != nil {
		return nil, err
	}
	if source == nil {
		return nil, ErrDietRecipeNotFound
	}
	target, err := s.recipes.FindByIDWithItems(ctx, kindergartenID, targetID)
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, ErrDietRecipeNotFound
	}

	merged := mergeDietRecipeItems(target.Items, source.Items)
	count, err := s.recipes.MergeInto(ctx, kindergartenID, sourceID, targetID, merged)
	if err != nil {
		return nil, err
	}
	return &DietRecipeMergeResult{TargetID: targetID, MergedItemCount: count}, nil
}

// SuggestRecipes 编辑器菜名联想：名称包含或拼音码前缀匹配，返回候选（不含配料）。
// limit 默认 18、上限 100（契约 §四）。
func (s *DietRecipeService) SuggestRecipes(
	ctx context.Context,
	kindergartenID uint64,
	keyword string,
	limit int,
) ([]DietRecipeListItem, error) {
	if kindergartenID == 0 {
		return nil, dietRecipeInvalid("未绑定园所，无法读取菜谱库")
	}
	if limit <= 0 {
		limit = dietRecipeSuggestDefaultLimit
	}
	if limit > dietRecipeSuggestMaxLimit {
		limit = dietRecipeSuggestMaxLimit
	}
	rows, err := s.recipes.Suggest(ctx, kindergartenID, strings.TrimSpace(keyword), limit)
	if err != nil {
		return nil, err
	}
	out := make([]DietRecipeListItem, 0, len(rows))
	for i := range rows {
		out = append(out, dietRecipeListItemView(rows[i]))
	}
	return out, nil
}

// normalizeRecipeItems 校验并归一化配料：克数 > 0、食物属于本园且未删除、
// 同一种食物多行合并为一行（克数相加，TC-NEW-DIE-0021）。
// 合并后保留首次出现的顺序，sort 由服务端按顺序重写（契约不接受客户端 sort）。
func (s *DietRecipeService) normalizeRecipeItems(
	ctx context.Context,
	kindergartenID uint64,
	in []DietRecipeItemInput,
) ([]domain.RecipeItem, error) {
	if len(in) == 0 {
		return nil, dietRecipeInvalid("菜谱至少需要一种食物")
	}

	merged := make([]domain.RecipeItem, 0, len(in))
	index := make(map[uint64]int, len(in))
	for _, row := range in {
		if row.FoodID == 0 {
			return nil, dietRecipeInvalid("食物 ID 无效")
		}
		if math.IsNaN(row.Grams) || math.IsInf(row.Grams, 0) || row.Grams <= 0 {
			return nil, dietRecipeInvalid("食物克数必须大于 0")
		}
		if at, ok := index[row.FoodID]; ok {
			merged[at].Grams += row.Grams
			continue
		}
		index[row.FoodID] = len(merged)
		merged = append(merged, domain.RecipeItem{
			FoodID: row.FoodID,
			Grams:  row.Grams,
			Sort:   len(merged) + 1,
		})
	}

	foodIDs := make([]uint64, 0, len(merged))
	for _, item := range merged {
		foodIDs = append(foodIDs, item.FoodID)
	}
	foods, err := s.foods.FindByIDs(ctx, kindergartenID, foodIDs)
	if err != nil {
		return nil, err
	}
	found := make(map[uint64]bool, len(foods))
	for i := range foods {
		found[foods[i].ID] = true
	}
	for _, id := range foodIDs {
		if !found[id] {
			return nil, dietRecipeInvalid("食物不存在或已删除：%d", id)
		}
	}
	return merged, nil
}

// mergeDietRecipeItems 把源配方并入目标：目标顺序优先，源中独有的食物按源顺序追加，
// 最后统一重写 sort。合并结果整体替换目标配料（调用方负责事务）。
func mergeDietRecipeItems(target, source []domain.RecipeItem) []domain.RecipeItem {
	out := make([]domain.RecipeItem, 0, len(target)+len(source))
	index := make(map[uint64]int, len(target)+len(source))

	appendItem := func(item domain.RecipeItem) {
		if at, ok := index[item.FoodID]; ok {
			out[at].Grams += item.Grams
			return
		}
		index[item.FoodID] = len(out)
		out = append(out, domain.RecipeItem{
			FoodID: item.FoodID,
			Grams:  item.Grams,
		})
	}
	for _, item := range target {
		appendItem(item)
	}
	for _, item := range source {
		appendItem(item)
	}
	for i := range out {
		out[i].Sort = i + 1
	}
	return out
}

func validateDietRecipeName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", dietRecipeInvalid("菜谱名称不能为空")
	}
	if utf8.RuneCountInString(name) > 128 {
		return "", dietRecipeInvalid("菜谱名称不能超过 128 个字符")
	}
	return name, nil
}

func dietRecipeListItemView(row repository.DietRecipeListRow) DietRecipeListItem {
	return DietRecipeListItem{
		ID:        row.ID,
		Name:      row.Name,
		SpellCode: row.SpellCode,
		ItemCount: row.ItemCount,
	}
}
