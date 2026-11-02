package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/pkg/pagination"
)

// DietFoodValidationError 表示食物库入参或业务校验失败。
// handler 用 errors.As 识别并映射为项目统一的 422 包络；Message 可直接展示给管理员。
type DietFoodValidationError struct {
	Message string
}

func (e *DietFoodValidationError) Error() string { return e.Message }

func dietFoodInvalid(format string, args ...any) error {
	return &DietFoodValidationError{Message: fmt.Sprintf(format, args...)}
}

// 食物删除与详情访问的业务错误。跨园访问在仓储层查不到，统一按 NotFound 处理（契约 404）。
var (
	ErrDietFoodNotFound           = errors.New("DIET_FOOD_NOT_FOUND")
	ErrDietFoodReferencedByRecipe = errors.New("DIET_FOOD_REFERENCED_BY_RECIPE")
)

// DietFoodNutrientsInput 是新建/修改时提交的 27 项营养。
// 指针区分「未提交」与「提交为 null」——契约要求未提交的键写 NULL，不写 0；
// 修改时整个 nutrients 对象一旦提交即整体替换。
type DietFoodNutrientsInput struct {
	Energy        *float64
	Heat          *float64
	Protein       *float64
	Fats          *float64
	Carb          *float64
	DietaryFiber  *float64
	Ash           *float64
	Water         *float64
	Cholesterol   *float64
	Retinol       *float64
	Carotin       *float64
	VitaminA      *float64
	VitaminB1     *float64
	VitaminB2     *float64
	NicotinicAcid *float64
	VitaminC      *float64
	VitaminE      *float64
	Calcium       *float64
	Phosphorus    *float64
	Potassium     *float64
	Sodium        *float64
	Magnesium     *float64
	Iron          *float64
	Zinc          *float64
	Selenium      *float64
	Copper        *float64
	Manganese     *float64
	// EdiblePercent 是契约禁止的键：食部在顶层，混进 nutrients 必须报错而不是静默忽略。
	EdiblePercent *float64
}

// DietFoodInput 是新建/修改食物的入参。新建时 Name/CategoryID/EdiblePercent 必填；
// 修改时全部可选，未提交的顶层字段保持原值。
type DietFoodInput struct {
	Name          *string
	CategoryID    *uint64
	EdiblePercent *float64
	Group12       *int
	Group16       *domain.DietFoodGroup16
	// SourceFlags 为 nil 表示未提交；空切片表示显式清空。
	SourceFlags []string
	// Nutrients 为 nil 表示未提交（修改时保留旧值）；非 nil 即整体替换。
	Nutrients *DietFoodNutrientsInput
}

// DietFoodCategoryNode 分类树节点（叶子）。
type DietFoodCategoryNode struct {
	ID       uint64 `json:"id"`
	ParentID uint64 `json:"parent_id"`
	Level    int    `json:"level"`
	Name     string `json:"name"`
	Sort     int    `json:"sort"`
}

// DietFoodCategoryView 一级分类及其二级子类。
type DietFoodCategoryView struct {
	DietFoodCategoryNode
	Children []DietFoodCategoryNode `json:"children"`
}

// DietFoodNutrientsView 食物详情的 27 项营养；nil 序列化为 null，前端渲染「—」。
type DietFoodNutrientsView struct {
	Energy        *float64 `json:"energy"`
	Heat          *float64 `json:"heat"`
	Protein       *float64 `json:"protein"`
	Fats          *float64 `json:"fats"`
	Carb          *float64 `json:"carb"`
	DietaryFiber  *float64 `json:"dietary_fiber"`
	Ash           *float64 `json:"ash"`
	Water         *float64 `json:"water"`
	Cholesterol   *float64 `json:"cholesterol"`
	Retinol       *float64 `json:"retinol"`
	Carotin       *float64 `json:"carotin"`
	VitaminA      *float64 `json:"vitamin_a"`
	VitaminB1     *float64 `json:"vitamin_b1"`
	VitaminB2     *float64 `json:"vitamin_b2"`
	NicotinicAcid *float64 `json:"nicotinic_acid"`
	VitaminC      *float64 `json:"vitamin_c"`
	VitaminE      *float64 `json:"vitamin_e"`
	Calcium       *float64 `json:"calcium"`
	Phosphorus    *float64 `json:"phosphorus"`
	Potassium     *float64 `json:"potassium"`
	Sodium        *float64 `json:"sodium"`
	Magnesium     *float64 `json:"magnesium"`
	Iron          *float64 `json:"iron"`
	Zinc          *float64 `json:"zinc"`
	Selenium      *float64 `json:"selenium"`
	Copper        *float64 `json:"copper"`
	Manganese     *float64 `json:"manganese"`
}

// DietFoodListItem 是列表行：只含参考产品列表页展示的 15 项营养，
// 不含 27 项全量（契约 §三 列表裁剪）。
type DietFoodListItem struct {
	ID            uint64                `json:"id"`
	Name          string                `json:"name"`
	SpellCode     string                `json:"spell_code"`
	CategoryID    uint64                `json:"category_id"`
	CategoryName  string                `json:"category_name"`
	Source        domain.DietFoodSource `json:"source"`
	EdiblePercent float64               `json:"edible_percent"`

	Energy        *float64 `json:"energy"`
	Protein       *float64 `json:"protein"`
	Fats          *float64 `json:"fats"`
	Carb          *float64 `json:"carb"`
	Retinol       *float64 `json:"retinol"`
	Carotin       *float64 `json:"carotin"`
	Calcium       *float64 `json:"calcium"`
	Iron          *float64 `json:"iron"`
	Zinc          *float64 `json:"zinc"`
	Phosphorus    *float64 `json:"phosphorus"`
	NicotinicAcid *float64 `json:"nicotinic_acid"`
	VitaminA      *float64 `json:"vitamin_a"`
	VitaminB1     *float64 `json:"vitamin_b1"`
	VitaminB2     *float64 `json:"vitamin_b2"`
	VitaminC      *float64 `json:"vitamin_c"`
}

// DietFoodDetail 是食物详情（27 项全量）。
type DietFoodDetail struct {
	ID               uint64                 `json:"id"`
	Name             string                 `json:"name"`
	SpellCode        string                 `json:"spell_code"`
	CategoryID       uint64                 `json:"category_id"`
	CategoryName     string                 `json:"category_name"`
	CategoryParentID uint64                 `json:"category_parent_id"`
	Source           domain.DietFoodSource  `json:"source"`
	EdiblePercent    float64                `json:"edible_percent"`
	Group12          *int                   `json:"group12"`
	Group16          domain.DietFoodGroup16 `json:"group16"`
	SourceFlags      []string               `json:"source_flags"`
	Nutrients        DietFoodNutrientsView  `json:"nutrients"`
}

// DietFoodCreateResult 是新增食物的响应。
type DietFoodCreateResult struct {
	ID        uint64 `json:"id"`
	Name      string `json:"name"`
	SpellCode string `json:"spell_code"`
}

// DietFoodBatchDeleteSkipped 是批量删除中被跳过的一条。
// reason 只允许契约定义的两个取值：referenced_by_recipe / not_found。
type DietFoodBatchDeleteSkipped struct {
	ID     uint64 `json:"id"`
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

const (
	DietFoodSkipReferencedByRecipe = "referenced_by_recipe"
	DietFoodSkipNotFound           = "not_found"
)

// DietFoodBatchDeleteResult 是批量删除结果：逐条判定，不因单条失败回滚整批。
type DietFoodBatchDeleteResult struct {
	Deleted int                          `json:"deleted"`
	Skipped []DietFoodBatchDeleteSkipped `json:"skipped"`
}

// DietFoodCandidate 是同类替换候选。
type DietFoodCandidate struct {
	ID           uint64 `json:"id"`
	Name         string `json:"name"`
	SpellCode    string `json:"spell_code"`
	CategoryID   uint64 `json:"category_id"`
	CategoryName string `json:"category_name"`
}

// DietFoodService 园所食物库。
//
// 多租户语义（design D2、ADR-0005）：所有读写都带 kindergarten_id，
// kg=0 是平台预置哨兵，绝不回传给园所。删除规则：被菜谱（RecipeItem）引用则拒绝；
// 只被周食谱（MealPlanItem）引用则允许——历史明细自带名称与营养快照（design D4）。
type DietFoodService struct {
	foods   *repository.DietFoodRepo
	recipes *repository.DietRecipeRepo
}

func NewDietFoodService(foods *repository.DietFoodRepo, recipes *repository.DietRecipeRepo) *DietFoodService {
	return &DietFoodService{foods: foods, recipes: recipes}
}

// ListCategories 返回全局两级分类树（一级 20 类，二级按 parent_id 挂载）。
func (s *DietFoodService) ListCategories(ctx context.Context) ([]DietFoodCategoryView, error) {
	rows, err := s.foods.ListCategories(ctx)
	if err != nil {
		return nil, err
	}
	return buildDietFoodCategoryTree(rows), nil
}

// ListFoods 返回本园食物列表。category_id 传一级时包含其全部二级（仓储层处理）。
func (s *DietFoodService) ListFoods(
	ctx context.Context,
	kindergartenID uint64,
	categoryID uint64,
	keyword string,
	pq pagination.Query,
) ([]DietFoodListItem, int64, int, int, error) {
	if kindergartenID == 0 {
		return nil, 0, 0, 0, dietFoodInvalid("未绑定园所，无法读取食物库")
	}
	pn := pq
	pn.Normalize()

	filter := repository.DietFoodListFilter{CategoryID: categoryID, Keyword: keyword}
	rows, total, err := s.foods.List(ctx, kindergartenID, filter, pn.Offset(), pn.PageSize)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	categories, err := s.categoryIndex(ctx)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	out := make([]DietFoodListItem, 0, len(rows))
	for i := range rows {
		out = append(out, dietFoodListItemView(&rows[i], categories))
	}
	return out, total, pn.Page, pn.PageSize, nil
}

// GetFood 返回食物详情；跨园或不存在一律 ErrDietFoodNotFound（契约 404）。
func (s *DietFoodService) GetFood(ctx context.Context, kindergartenID, id uint64) (*DietFoodDetail, error) {
	if kindergartenID == 0 {
		return nil, dietFoodInvalid("未绑定园所，无法读取食物库")
	}
	food, err := s.foods.FindByID(ctx, kindergartenID, id)
	if err != nil {
		return nil, err
	}
	if food == nil {
		return nil, ErrDietFoodNotFound
	}
	categories, err := s.categoryIndex(ctx)
	if err != nil {
		return nil, err
	}
	return dietFoodDetailView(food, categories), nil
}

// CreateFood 新增园所自建食物（source=custom）。
// group12/group16/source_flags 缺省时按所选一级分类推导（契约 §三 POST）。
func (s *DietFoodService) CreateFood(
	ctx context.Context,
	kindergartenID uint64,
	in DietFoodInput,
) (*DietFoodCreateResult, error) {
	if kindergartenID == 0 {
		return nil, dietFoodInvalid("未绑定园所，无法新增食物")
	}
	if in.Name == nil {
		return nil, dietFoodInvalid("食物名称不能为空")
	}
	name, err := validateDietFoodName(*in.Name)
	if err != nil {
		return nil, err
	}
	if in.CategoryID == nil {
		return nil, dietFoodInvalid("请选择食物分类")
	}
	category, rootName, err := s.requireLeafCategory(ctx, *in.CategoryID)
	if err != nil {
		return nil, err
	}
	if in.EdiblePercent == nil {
		return nil, dietFoodInvalid("食部比例不能为空")
	}
	edible, err := validateEdiblePercent(*in.EdiblePercent)
	if err != nil {
		return nil, err
	}

	food := &domain.Food{
		KindergartenID: kindergartenID,
		Name:           name,
		SpellCode:      dietSpellCode(name),
		CategoryID:     category.ID,
		EdiblePercent:  edible,
		Source:         domain.DietFoodSourceCustom,
	}
	if err := applyDietFoodGroups(food, rootName, in); err != nil {
		return nil, err
	}
	if in.Nutrients != nil {
		if err := applyDietFoodNutrients(food, in.Nutrients); err != nil {
			return nil, err
		}
	}
	if err := s.foods.Create(ctx, food); err != nil {
		return nil, err
	}
	return &DietFoodCreateResult{ID: food.ID, Name: food.Name, SpellCode: food.SpellCode}, nil
}

// UpdateFood 修改本园食物。对 source=system 的副本做任何修改都会置 is_customized=true，
// 平台升级据此跳过该行（design D4、TC-NEW-DIE-0016 的 service 侧）。
func (s *DietFoodService) UpdateFood(
	ctx context.Context,
	kindergartenID, id uint64,
	in DietFoodInput,
) error {
	if kindergartenID == 0 {
		return dietFoodInvalid("未绑定园所，无法修改食物")
	}
	food, err := s.foods.FindByID(ctx, kindergartenID, id)
	if err != nil {
		return err
	}
	if food == nil {
		return ErrDietFoodNotFound
	}

	changed := false
	if in.Name != nil {
		name, err := validateDietFoodName(*in.Name)
		if err != nil {
			return err
		}
		if name != food.Name {
			food.Name = name
			food.SpellCode = dietSpellCode(name)
		}
		changed = true
	}
	if in.CategoryID != nil {
		_, _, err := s.requireLeafCategory(ctx, *in.CategoryID)
		if err != nil {
			return err
		}
		food.CategoryID = *in.CategoryID
		changed = true
	}
	if in.EdiblePercent != nil {
		edible, err := validateEdiblePercent(*in.EdiblePercent)
		if err != nil {
			return err
		}
		food.EdiblePercent = edible
		changed = true
	}
	if in.Group12 != nil {
		if err := validateDietGroup12(*in.Group12); err != nil {
			return err
		}
		food.Group12 = in.Group12
		changed = true
	}
	if in.Group16 != nil {
		if err := validateDietGroup16(*in.Group16); err != nil {
			return err
		}
		food.Group16 = *in.Group16
		changed = true
	}
	if in.SourceFlags != nil {
		flags, err := normalizeDietSourceFlags(in.SourceFlags)
		if err != nil {
			return err
		}
		food.SourceFlags = strings.Join(flags, ",")
		changed = true
	}
	if in.Nutrients != nil {
		if err := applyDietFoodNutrients(food, in.Nutrients); err != nil {
			return err
		}
		changed = true
	}
	if !changed {
		return dietFoodInvalid("没有需要修改的字段")
	}
	if food.Source == domain.DietFoodSourceSystem {
		food.IsCustomized = true
	}
	return s.foods.Update(ctx, food)
}

// DeleteFood 软删除本园食物；被菜谱引用时拒绝（历史周食谱引用不阻止删除）。
func (s *DietFoodService) DeleteFood(ctx context.Context, kindergartenID, id uint64) error {
	if kindergartenID == 0 {
		return dietFoodInvalid("未绑定园所，无法删除食物")
	}
	food, err := s.foods.FindByID(ctx, kindergartenID, id)
	if err != nil {
		return err
	}
	if food == nil {
		return ErrDietFoodNotFound
	}
	referenced, err := s.isReferencedByRecipe(ctx, kindergartenID, id)
	if err != nil {
		return err
	}
	if referenced {
		return ErrDietFoodReferencedByRecipe
	}
	affected, err := s.foods.Delete(ctx, kindergartenID, id)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrDietFoodNotFound
	}
	return nil
}

// BatchDeleteFoods 逐条判定并删除；被菜谱引用或不存在/跨园的条目进 skipped，不中断整批。
func (s *DietFoodService) BatchDeleteFoods(
	ctx context.Context,
	kindergartenID uint64,
	ids []uint64,
) (*DietFoodBatchDeleteResult, error) {
	if kindergartenID == 0 {
		return nil, dietFoodInvalid("未绑定园所，无法删除食物")
	}
	unique, err := normalizeDietFoodIDs(ids)
	if err != nil {
		return nil, err
	}
	rows, err := s.foods.FindByIDs(ctx, kindergartenID, unique)
	if err != nil {
		return nil, err
	}
	byID := make(map[uint64]*domain.Food, len(rows))
	for i := range rows {
		byID[rows[i].ID] = &rows[i]
	}

	result := &DietFoodBatchDeleteResult{Skipped: make([]DietFoodBatchDeleteSkipped, 0)}
	for _, id := range unique {
		food, ok := byID[id]
		if !ok {
			result.Skipped = append(result.Skipped, DietFoodBatchDeleteSkipped{
				ID: id, Name: "", Reason: DietFoodSkipNotFound,
			})
			continue
		}
		referenced, err := s.isReferencedByRecipe(ctx, kindergartenID, id)
		if err != nil {
			return nil, err
		}
		if referenced {
			result.Skipped = append(result.Skipped, DietFoodBatchDeleteSkipped{
				ID: id, Name: food.Name, Reason: DietFoodSkipReferencedByRecipe,
			})
			continue
		}
		affected, err := s.foods.Delete(ctx, kindergartenID, id)
		if err != nil {
			return nil, err
		}
		if affected == 0 {
			result.Skipped = append(result.Skipped, DietFoodBatchDeleteSkipped{
				ID: id, Name: food.Name, Reason: DietFoodSkipNotFound,
			})
			continue
		}
		result.Deleted++
	}
	return result, nil
}

// ListReplaceCandidates 返回同一一级分类下的其他本园食物（契约 §三）。
// 按名称排序；limit 默认 20、上限 100。
func (s *DietFoodService) ListReplaceCandidates(
	ctx context.Context,
	kindergartenID, id uint64,
	keyword string,
	limit int,
) ([]DietFoodCandidate, error) {
	if kindergartenID == 0 {
		return nil, dietFoodInvalid("未绑定园所，无法读取食物库")
	}
	food, err := s.foods.FindByID(ctx, kindergartenID, id)
	if err != nil {
		return nil, err
	}
	if food == nil {
		return nil, ErrDietFoodNotFound
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	categories, err := s.categoryIndex(ctx)
	if err != nil {
		return nil, err
	}
	rootID := food.CategoryID
	if leaf, ok := categories[food.CategoryID]; ok {
		if leaf.Level == 2 {
			rootID = leaf.ParentID
		} else {
			rootID = leaf.ID
		}
	}
	rows, err := s.foods.ListReplaceCandidates(ctx, kindergartenID, rootID, id, keyword, limit)
	if err != nil {
		return nil, err
	}
	out := make([]DietFoodCandidate, 0, len(rows))
	for i := range rows {
		out = append(out, DietFoodCandidate{
			ID:           rows[i].ID,
			Name:         rows[i].Name,
			SpellCode:    rows[i].SpellCode,
			CategoryID:   rows[i].CategoryID,
			CategoryName: categories[rows[i].CategoryID].Name,
		})
	}
	return out, nil
}

func (s *DietFoodService) isReferencedByRecipe(ctx context.Context, kindergartenID, foodID uint64) (bool, error) {
	count, err := s.recipes.CountByFoodID(ctx, kindergartenID, foodID)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *DietFoodService) requireLeafCategory(
	ctx context.Context,
	id uint64,
) (*domain.FoodCategory, string, error) {
	if id == 0 {
		return nil, "", dietFoodInvalid("请选择食物分类")
	}
	category, err := s.foods.FindCategoryByID(ctx, id)
	if err != nil {
		return nil, "", err
	}
	if category == nil {
		return nil, "", dietFoodInvalid("食物分类不存在")
	}
	if category.Level != 2 {
		return nil, "", dietFoodInvalid("请选择二级（叶子）分类")
	}
	// 分组兜底按一级分类推导，因此这里顺带取父类名称。
	rootName := ""
	if category.ParentID != 0 {
		parent, err := s.foods.FindCategoryByID(ctx, category.ParentID)
		if err != nil {
			return nil, "", err
		}
		if parent != nil {
			rootName = parent.Name
		}
	}
	return category, rootName, nil
}

func (s *DietFoodService) categoryIndex(ctx context.Context) (map[uint64]domain.FoodCategory, error) {
	rows, err := s.foods.ListCategories(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[uint64]domain.FoodCategory, len(rows))
	for _, row := range rows {
		out[row.ID] = row
	}
	return out, nil
}

// DietFoodGroupDefaults 按一级分类名推导分组的兜底值。
//
// 分类是平台固定的 20 个一级类（调研 notes §2.10），种子在 S5 落库；
// 这里按名称而不是 ID 映射，避免依赖自增 ID 的插入顺序。
// 未列出的分类不推导（group12 无槽位、group16/source_flags 留空），管理员可显式覆盖。
var dietGroup16ByRootCategory = map[string]domain.DietFoodGroup16{
	"谷类及制品":    domain.DietGroup16Grain,
	"薯类、淀粉及制品": domain.DietGroup16Tuber,
	"蔬菜类及制品":   domain.DietGroup16Vegetable,
	"菌藻类":      domain.DietGroup16Vegetable,
	"水果类及制品":   domain.DietGroup16Fruit,
	"畜肉类及制品":   domain.DietGroup16Meat,
	"禽肉类及制品":   domain.DietGroup16Meat,
	"蛋类及制品":    domain.DietGroup16Egg,
	"鱼虾蟹贝类":    domain.DietGroup16Aquatic,
	"干豆类及制品":   domain.DietGroup16Soybean,
	"坚果、种子类":   domain.DietGroup16Nut,
	"乳类及制品":    domain.DietGroup16Dairy,
	"小吃、甜饼":    domain.DietGroup16InstantFood,
	"速食食品":     domain.DietGroup16InstantFood,
	"饮料类":      domain.DietGroup16Beverage,
	"糖、蜜饯类":    domain.DietGroup16Confection,
	"动物油脂":     domain.DietGroup16Oil,
	"植物油脂":     domain.DietGroup16Oil,
	"调味品类":     domain.DietGroup16Condiment,
	"其他":       domain.DietGroup16Other,
}

// dietGroup12ByRootCategory 对应 design D2 的 37–48 槽位；无槽位的分类不出现。
var dietGroup12ByRootCategory = map[string]int{
	"谷类及制品":  domain.DietGroup12Grain,
	"干豆类及制品": domain.DietGroup12Bean,
	"畜肉类及制品": domain.DietGroup12Meat,
	"禽肉类及制品": domain.DietGroup12Meat,
	"蛋类及制品":  domain.DietGroup12Egg,
	"乳类及制品":  domain.DietGroup12Dairy,
	"蔬菜类及制品": domain.DietGroup12Vegetable,
	"菌藻类":    domain.DietGroup12Vegetable,
	"小吃、甜饼":  domain.DietGroup12Pastry,
	"速食食品":   domain.DietGroup12Pastry,
	"水果类及制品": domain.DietGroup12Fruit,
	"调味品类":   domain.DietGroup12Condiment,
	"鱼虾蟹贝类":  domain.DietGroup12Seafood,
	"植物油脂":   domain.DietGroup12Oil,
	"坚果、种子类": domain.DietGroup12Nut,
}

// dietSourceFlagsByRootCategory 是报表集合运算的粗粒度兜底；具体食物可在编辑页覆盖。
var dietSourceFlagsByRootCategory = map[string][]string{
	"谷类及制品":    {domain.DietSourceFlagPlant},
	"薯类、淀粉及制品": {domain.DietSourceFlagPlant},
	"蔬菜类及制品":   {domain.DietSourceFlagPlant},
	"菌藻类":      {domain.DietSourceFlagPlant},
	"水果类及制品":   {domain.DietSourceFlagPlant},
	"畜肉类及制品":   {domain.DietSourceFlagAnimal, domain.DietSourceFlagMeat},
	"禽肉类及制品":   {domain.DietSourceFlagAnimal, domain.DietSourceFlagMeat},
	"蛋类及制品":    {domain.DietSourceFlagAnimal},
	"鱼虾蟹贝类":    {domain.DietSourceFlagAnimal},
	"干豆类及制品":   {domain.DietSourceFlagPlant},
	"坚果、种子类":   {domain.DietSourceFlagPlant},
	"乳类及制品":    {domain.DietSourceFlagAnimal, domain.DietSourceFlagMilk},
	"糖、蜜饯类":    {domain.DietSourceFlagPlant},
	"动物油脂":     {domain.DietSourceFlagAnimal},
	"植物油脂":     {domain.DietSourceFlagPlant},
}

// 分类固定顺序，保证 source_flags 输出稳定。
var dietSourceFlagOrder = []string{
	domain.DietSourceFlagAnimal,
	domain.DietSourceFlagPlant,
	domain.DietSourceFlagMilk,
	domain.DietSourceFlagMeat,
}

func applyDietFoodGroups(food *domain.Food, rootCategoryName string, in DietFoodInput) error {
	if in.Group12 != nil {
		if err := validateDietGroup12(*in.Group12); err != nil {
			return err
		}
		food.Group12 = in.Group12
	} else if v, ok := dietGroup12ByRootCategory[rootCategoryName]; ok {
		value := v
		food.Group12 = &value
	}
	if in.Group16 != nil {
		if err := validateDietGroup16(*in.Group16); err != nil {
			return err
		}
		food.Group16 = *in.Group16
	} else if v, ok := dietGroup16ByRootCategory[rootCategoryName]; ok {
		food.Group16 = v
	}
	if in.SourceFlags != nil {
		flags, err := normalizeDietSourceFlags(in.SourceFlags)
		if err != nil {
			return err
		}
		food.SourceFlags = strings.Join(flags, ",")
	} else if flags, ok := dietSourceFlagsByRootCategory[rootCategoryName]; ok {
		food.SourceFlags = strings.Join(flags, ",")
	}
	return nil
}

func applyDietFoodNutrients(food *domain.Food, in *DietFoodNutrientsInput) error {
	if in.EdiblePercent != nil {
		return dietFoodInvalid("nutrients 中不允许出现 edible_percent，请使用顶层字段")
	}
	food.Energy = in.Energy
	food.Heat = in.Heat
	if food.Heat == nil && in.Energy != nil {
		derived := math.Round(*in.Energy*4.186*100) / 100
		food.Heat = &derived
	}
	food.Protein = in.Protein
	food.Fats = in.Fats
	food.Carb = in.Carb
	food.DietaryFiber = in.DietaryFiber
	food.Ash = in.Ash
	food.Water = in.Water
	food.Cholesterol = in.Cholesterol
	food.Retinol = in.Retinol
	food.Carotin = in.Carotin
	food.VitaminA = in.VitaminA
	food.VitaminB1 = in.VitaminB1
	food.VitaminB2 = in.VitaminB2
	food.NicotinicAcid = in.NicotinicAcid
	food.VitaminC = in.VitaminC
	food.VitaminE = in.VitaminE
	food.Calcium = in.Calcium
	food.Phosphorus = in.Phosphorus
	food.Potassium = in.Potassium
	food.Sodium = in.Sodium
	food.Magnesium = in.Magnesium
	food.Iron = in.Iron
	food.Zinc = in.Zinc
	food.Selenium = in.Selenium
	food.Copper = in.Copper
	food.Manganese = in.Manganese
	return nil
}

func validateDietFoodName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", dietFoodInvalid("食物名称不能为空")
	}
	if utf8.RuneCountInString(name) > 128 {
		return "", dietFoodInvalid("食物名称不能超过 128 个字符")
	}
	return name, nil
}

func validateEdiblePercent(v float64) (float64, error) {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > 100 {
		return 0, dietFoodInvalid("食部比例必须在 0 到 100 之间")
	}
	return v, nil
}

func validateDietGroup12(v int) error {
	if v < domain.DietGroup12Grain || v > domain.DietGroup12Nut {
		return dietFoodInvalid("采购分类 group12 必须在 37 到 48 之间")
	}
	return nil
}

func validateDietGroup16(v domain.DietFoodGroup16) error {
	switch v {
	case domain.DietGroup16Grain, domain.DietGroup16Tuber, domain.DietGroup16Vegetable,
		domain.DietGroup16Fruit, domain.DietGroup16Meat, domain.DietGroup16Egg,
		domain.DietGroup16Aquatic, domain.DietGroup16Soybean, domain.DietGroup16Nut,
		domain.DietGroup16Dairy, domain.DietGroup16InstantFood, domain.DietGroup16Beverage,
		domain.DietGroup16Confection, domain.DietGroup16Oil, domain.DietGroup16Condiment,
		domain.DietGroup16Other:
		return nil
	default:
		return dietFoodInvalid("膳食构成分类 group16 无效")
	}
}

func normalizeDietSourceFlags(flags []string) ([]string, error) {
	seen := make(map[string]bool, len(flags))
	for _, raw := range flags {
		flag := strings.TrimSpace(raw)
		valid := false
		for _, allowed := range dietSourceFlagOrder {
			if flag == allowed {
				valid = true
				break
			}
		}
		if !valid {
			return nil, dietFoodInvalid("食物属性无效：%s", raw)
		}
		seen[flag] = true
	}
	out := make([]string, 0, len(seen))
	for _, flag := range dietSourceFlagOrder {
		if seen[flag] {
			out = append(out, flag)
		}
	}
	return out, nil
}

func normalizeDietFoodIDs(ids []uint64) ([]uint64, error) {
	if len(ids) == 0 {
		return nil, dietFoodInvalid("请选择要删除的食物")
	}
	seen := make(map[uint64]bool, len(ids))
	out := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			return nil, dietFoodInvalid("食物 ID 无效")
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out, nil
}

func buildDietFoodCategoryTree(rows []domain.FoodCategory) []DietFoodCategoryView {
	roots := make([]DietFoodCategoryView, 0)
	index := make(map[uint64]int)
	for _, row := range rows {
		if row.Level != 1 {
			continue
		}
		index[row.ID] = len(roots)
		roots = append(roots, DietFoodCategoryView{
			DietFoodCategoryNode: dietFoodCategoryNode(row),
			Children:             make([]DietFoodCategoryNode, 0),
		})
	}
	for _, row := range rows {
		if row.Level != 2 {
			continue
		}
		at, ok := index[row.ParentID]
		if !ok {
			continue
		}
		roots[at].Children = append(roots[at].Children, dietFoodCategoryNode(row))
	}
	return roots
}

func dietFoodCategoryNode(row domain.FoodCategory) DietFoodCategoryNode {
	return DietFoodCategoryNode{
		ID:       row.ID,
		ParentID: row.ParentID,
		Level:    row.Level,
		Name:     row.Name,
		Sort:     row.Sort,
	}
}

func dietFoodListItemView(food *domain.Food, categories map[uint64]domain.FoodCategory) DietFoodListItem {
	return DietFoodListItem{
		ID:            food.ID,
		Name:          food.Name,
		SpellCode:     food.SpellCode,
		CategoryID:    food.CategoryID,
		CategoryName:  categories[food.CategoryID].Name,
		Source:        food.Source,
		EdiblePercent: food.EdiblePercent,
		Energy:        food.Energy,
		Protein:       food.Protein,
		Fats:          food.Fats,
		Carb:          food.Carb,
		Retinol:       food.Retinol,
		Carotin:       food.Carotin,
		Calcium:       food.Calcium,
		Iron:          food.Iron,
		Zinc:          food.Zinc,
		Phosphorus:    food.Phosphorus,
		NicotinicAcid: food.NicotinicAcid,
		VitaminA:      food.VitaminA,
		VitaminB1:     food.VitaminB1,
		VitaminB2:     food.VitaminB2,
		VitaminC:      food.VitaminC,
	}
}

func dietFoodDetailView(food *domain.Food, categories map[uint64]domain.FoodCategory) *DietFoodDetail {
	category := categories[food.CategoryID]
	return &DietFoodDetail{
		ID:               food.ID,
		Name:             food.Name,
		SpellCode:        food.SpellCode,
		CategoryID:       food.CategoryID,
		CategoryName:     category.Name,
		CategoryParentID: category.ParentID,
		Source:           food.Source,
		EdiblePercent:    food.EdiblePercent,
		Group12:          food.Group12,
		Group16:          food.Group16,
		SourceFlags:      splitDietSourceFlags(food.SourceFlags),
		Nutrients: DietFoodNutrientsView{
			Energy:        food.Energy,
			Heat:          food.Heat,
			Protein:       food.Protein,
			Fats:          food.Fats,
			Carb:          food.Carb,
			DietaryFiber:  food.DietaryFiber,
			Ash:           food.Ash,
			Water:         food.Water,
			Cholesterol:   food.Cholesterol,
			Retinol:       food.Retinol,
			Carotin:       food.Carotin,
			VitaminA:      food.VitaminA,
			VitaminB1:     food.VitaminB1,
			VitaminB2:     food.VitaminB2,
			NicotinicAcid: food.NicotinicAcid,
			VitaminC:      food.VitaminC,
			VitaminE:      food.VitaminE,
			Calcium:       food.Calcium,
			Phosphorus:    food.Phosphorus,
			Potassium:     food.Potassium,
			Sodium:        food.Sodium,
			Magnesium:     food.Magnesium,
			Iron:          food.Iron,
			Zinc:          food.Zinc,
			Selenium:      food.Selenium,
			Copper:        food.Copper,
			Manganese:     food.Manganese,
		},
	}
}

func splitDietSourceFlags(raw string) []string {
	out := make([]string, 0, 4)
	for _, part := range strings.Split(raw, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

// dietSpellCode 生成拼音码。
//
// 汉字按 GB2312 一级字库的拼音排序区间取首字母（与参考产品「面条→MT」一致）；
// 非汉字只保留 ASCII 字母/数字并大写。GBK 编码失败（生僻字/表情）时退回纯 ASCII。
// 多音字取 GB2312 排序所用读音，属于可接受的近似。
func dietSpellCode(name string) string {
	encoded, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte(name))
	if err != nil {
		return dietASCIISpellCode(name)
	}
	var b strings.Builder
	for i := 0; i < len(encoded); {
		if encoded[i] < 0x80 {
			writeDietSpellRune(&b, rune(encoded[i]))
			i++
			continue
		}
		if i+1 >= len(encoded) {
			break
		}
		code := int(encoded[i])<<8 | int(encoded[i+1])
		if initial := dietGBKPinyinInitial(code); initial != 0 {
			b.WriteByte(initial)
		}
		i += 2
	}
	return b.String()
}

func dietASCIISpellCode(name string) string {
	var b strings.Builder
	for _, r := range name {
		writeDietSpellRune(&b, r)
	}
	return b.String()
}

func writeDietSpellRune(b *strings.Builder, r rune) {
	switch {
	case r >= 'a' && r <= 'z':
		b.WriteRune(r - ('a' - 'A'))
	case (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'):
		b.WriteRune(r)
	}
}

// dietGBKPinyinInitial 按 GB2312 一级汉字（B0A1–D7F9）的拼音顺序区间返回首字母。
// 区间边界用实际编码核对过：挖 CDDA=W、昔 CEF4=X、压 D1B9=Y、匝 D4D1=Z、中 D6D0=Z。
func dietGBKPinyinInitial(code int) byte {
	switch {
	case code >= 0xB0A1 && code <= 0xB0C4:
		return 'A'
	case code >= 0xB0C5 && code <= 0xB2C0:
		return 'B'
	case code >= 0xB2C1 && code <= 0xB4ED:
		return 'C'
	case code >= 0xB4EE && code <= 0xB6E9:
		return 'D'
	case code >= 0xB6EA && code <= 0xB7A1:
		return 'E'
	case code >= 0xB7A2 && code <= 0xB8C0:
		return 'F'
	case code >= 0xB8C1 && code <= 0xB9FD:
		return 'G'
	case code >= 0xB9FE && code <= 0xBBF6:
		return 'H'
	case code >= 0xBBF7 && code <= 0xBFA5:
		return 'J'
	case code >= 0xBFA6 && code <= 0xC0AB:
		return 'K'
	case code >= 0xC0AC && code <= 0xC2E7:
		return 'L'
	case code >= 0xC2E8 && code <= 0xC4C2:
		return 'M'
	case code >= 0xC4C3 && code <= 0xC5B5:
		return 'N'
	case code >= 0xC5B6 && code <= 0xC5BD:
		return 'O'
	case code >= 0xC5BE && code <= 0xC6D9:
		return 'P'
	case code >= 0xC6DA && code <= 0xC8BA:
		return 'Q'
	case code >= 0xC8BB && code <= 0xC8F5:
		return 'R'
	case code >= 0xC8F6 && code <= 0xCBF9:
		return 'S'
	case code >= 0xCBFA && code <= 0xCDD9:
		return 'T'
	case code >= 0xCDDA && code <= 0xCEF3:
		return 'W'
	case code >= 0xCEF4 && code <= 0xD1B8:
		return 'X'
	case code >= 0xD1B9 && code <= 0xD4D0:
		return 'Y'
	case code >= 0xD4D1 && code <= 0xD7F9:
		return 'Z'
	}
	return 0
}
