package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// 膳食种子与授权 CSV 导入（openspec: add-quantified-recipe, design D10）。
//
// 数据链路：dietfood 命令把授权 CSV 写成 kg=0 平台预置行 → 启动种子/新园初始化
// 按 preset_key 复制到各园 → 再按食物名称种 2 套模板。
// 本文件只落地「分类 + DRIs + 复制 + 模板 + CSV 导入」；授权数据本身不入库、不入仓库。

// dietRootCategoryNames 是参考产品 foodTypeList 的 20 个一级分类，按 foodTypeID 1–20 排序。
// 分类是平台级只读数据（无 kindergarten_id），食物通过 category_id 挂到二级叶子。
// 文案必须与 diet_food_service.go 的分组映射保持同一套（测试会断言两处一致）。
var dietRootCategoryNames = []string{
	"谷类及制品",
	"薯类、淀粉及制品",
	"蔬菜类及制品",
	"菌藻类",
	"水果类及制品",
	"畜肉类及制品",
	"禽肉类及制品",
	"蛋类及制品",
	"鱼虾蟹贝类",
	"干豆类及制品",
	"坚果、种子类",
	"乳类及制品",
	"小吃、甜饼",
	"速食食品",
	"饮料类",
	"糖、蜜饯类",
	"动物油脂",
	"植物油脂",
	"调味品类",
	"其他",
}

// dietDriAgeBands 是 DRIs 的四个年龄档。
var dietDriAgeBands = [4]int{3, 4, 5, 6}

// DietSeedService 负责膳食平台数据的幂等播种与授权 CSV 导入。
//
// 种子顺序固定：全局分类/DRIs → 按园复制 kg=0 食物 → 种模板（design D10）。
// 所有写入都按业务键判重（分类按名称、DRIs 按年龄档+营养素、食物按 preset_key、
// 模板按 template_key），重复执行不产生重复行。
type DietSeedService struct {
	db        *gorm.DB
	foods     *repository.DietFoodRepo
	templates *repository.DietTemplateRepo
	dri       *repository.DietDriRepo
}

func NewDietSeedService(
	db *gorm.DB,
	foods *repository.DietFoodRepo,
	templates *repository.DietTemplateRepo,
	dri *repository.DietDriRepo,
) *DietSeedService {
	return &DietSeedService{db: db, foods: foods, templates: templates, dri: dri}
}

// DietGlobalSeedResult 是全局种子结果；已存在的行不重复插入。
type DietGlobalSeedResult struct {
	CategoriesAdded int `json:"categories_added"`
	DriRowsAdded    int `json:"dri_rows_added"`
}

// DietTemplateSeedResult 是单园模板种子结果。
type DietTemplateSeedResult struct {
	TemplatesCreated int      `json:"templates_created"`
	ItemsCreated     int      `json:"items_created"`
	SkippedNoFoods   bool     `json:"skipped_no_foods"`
	SkippedTemplates []string `json:"skipped_templates,omitempty"`
	MissingFoods     []string `json:"missing_foods,omitempty"`
}

// DietKindergartenSeedResult 是单园初始化（复制 + 模板）结果。
type DietKindergartenSeedResult struct {
	KindergartenID       uint64   `json:"kindergarten_id"`
	FoodsCopied          int      `json:"foods_copied"`
	TemplatesCreated     int      `json:"templates_created"`
	TemplateItemsCreated int      `json:"template_items_created"`
	SkippedNoFoods       bool     `json:"skipped_no_foods"`
	MissingFoods         []string `json:"missing_foods,omitempty"`
}

// DietSeedForAllResult 是全园补齐结果；单园失败只记告警，不中断其他园所。
type DietSeedForAllResult struct {
	Kindergartens        int      `json:"kindergartens"`
	FoodsCopied          int      `json:"foods_copied"`
	TemplatesCreated     int      `json:"templates_created"`
	TemplateItemsCreated int      `json:"template_items_created"`
	Warnings             []string `json:"warnings,omitempty"`
}

// SeedDietGlobals 幂等播种平台级只读数据：20 个一级食物分类 + 3/4/5/6 岁 × 11 项 DRIs。
//
// 只插入缺失行，不覆盖已有值（分类可能被人工调整过 sort，DRIs 是全局基线）。
// 软删除行仍占用唯一索引，因此查到被软删除的同类行时复活它而不是新建。
func (s *DietSeedService) SeedDietGlobals(ctx context.Context) (DietGlobalSeedResult, error) {
	var result DietGlobalSeedResult
	for i, name := range dietRootCategoryNames {
		existing, err := s.foods.FindCategoryByParentName(ctx, 0, name)
		if err != nil {
			return result, fmt.Errorf("query food category %q: %w", name, err)
		}
		if existing != nil {
			if existing.DeletedAt.Valid {
				if err := s.foods.RestoreCategory(ctx, existing.ID); err != nil {
					return result, fmt.Errorf("restore food category %q: %w", name, err)
				}
			}
			continue
		}
		category := &domain.FoodCategory{ParentID: 0, Level: 1, Name: name, Sort: i + 1}
		if err := s.foods.CreateCategory(ctx, category); err != nil {
			// 并发播种（服务启动 + 测试）可能刚插入同名分类；回查确认后再决定是否报错。
			raced, findErr := s.foods.FindCategoryByParentName(ctx, 0, name)
			if findErr != nil {
				return result, fmt.Errorf("query food category %q after create failure: %w", name, findErr)
			}
			if raced == nil {
				return result, fmt.Errorf("create food category %q: %w", name, err)
			}
			if raced.DeletedAt.Valid {
				if err := s.foods.RestoreCategory(ctx, raced.ID); err != nil {
					return result, fmt.Errorf("restore food category %q: %w", name, err)
				}
			}
			continue
		}
		result.CategoriesAdded++
	}

	existing, err := s.dri.ListAllIncludingDeleted(ctx)
	if err != nil {
		return result, fmt.Errorf("query dri targets: %w", err)
	}
	known := make(map[string]domain.DriAgeTarget, len(existing))
	for _, row := range existing {
		known[dietDriKey(row.AgeBand, row.NutrientKey)] = row
	}
	toCreate := make([]domain.DriAgeTarget, 0, 44)
	for _, row := range dietDriSeedRows() {
		prev, ok := known[dietDriKey(row.AgeBand, row.NutrientKey)]
		if ok {
			if prev.DeletedAt.Valid {
				if err := s.dri.Restore(ctx, prev.ID); err != nil {
					return result, fmt.Errorf("restore dri target %d/%s: %w", row.AgeBand, row.NutrientKey, err)
				}
			}
			continue
		}
		toCreate = append(toCreate, row)
	}
	added, err := s.dri.CreateBatch(ctx, toCreate)
	if err != nil {
		return result, fmt.Errorf("create dri targets: %w", err)
	}
	result.DriRowsAdded = int(added)
	return result, nil
}

// SeedDietForAll 遍历全部园所，补齐食物副本与模板。
// 单个园所失败只记录告警：启动路径不应因某个园所的数据问题而中断（design D10）。
func (s *DietSeedService) SeedDietForAll(ctx context.Context) (DietSeedForAllResult, error) {
	var result DietSeedForAllResult
	var kindergartens []domain.Kindergarten
	if err := s.db.WithContext(ctx).
		Select("id", "name").
		Order("id ASC").
		Find(&kindergartens).Error; err != nil {
		return result, fmt.Errorf("list kindergartens: %w", err)
	}
	for _, kg := range kindergartens {
		if kg.ID == 0 {
			continue
		}
		result.Kindergartens++
		seeded, err := s.EnsureKindergartenSeeded(ctx, kg.ID)
		if err != nil {
			warning := fmt.Sprintf("kindergarten %d: %v", kg.ID, err)
			result.Warnings = append(result.Warnings, warning)
			slog.Warn("diet seed: 园所膳食初始化失败，继续下一个园所", "kindergarten_id", kg.ID, "error", err)
			continue
		}
		result.FoodsCopied += seeded.FoodsCopied
		result.TemplatesCreated += seeded.TemplatesCreated
		result.TemplateItemsCreated += seeded.TemplateItemsCreated
	}
	return result, nil
}

// EnsureKindergartenSeeded 幂等补齐单个园所：按 preset_key 复制 kg=0 食物，再种模板。
// 供新园初始化（SystemService.Initialize）与启动种子共用。
//
// 只插入本园缺失的 preset_key：已存在的行（无论园所是否改过）一律不动，
// 平台升级由导入命令按 is_customized 判定冲突（TC-NEW-DIE-0016）。
func (s *DietSeedService) EnsureKindergartenSeeded(
	ctx context.Context,
	kindergartenID uint64,
) (DietKindergartenSeedResult, error) {
	result := DietKindergartenSeedResult{KindergartenID: kindergartenID}
	if kindergartenID == 0 {
		return result, errors.New("diet seed: kindergarten_id 不能为 0")
	}

	copied, err := s.copyPresetFoods(ctx, kindergartenID)
	if err != nil {
		return result, err
	}
	result.FoodsCopied = copied

	templates, err := s.SeedDietTemplates(ctx, kindergartenID)
	if err != nil {
		return result, err
	}
	result.TemplatesCreated = templates.TemplatesCreated
	result.TemplateItemsCreated = templates.ItemsCreated
	result.SkippedNoFoods = templates.SkippedNoFoods
	result.MissingFoods = templates.MissingFoods
	return result, nil
}

// SeedDietTemplates 幂等种入平台初稿模板。园所没有食物时整体跳过并告警；
// 单个模板里找不到的食物名跳过，最后按模板汇总告警（design D10）。
func (s *DietSeedService) SeedDietTemplates(
	ctx context.Context,
	kindergartenID uint64,
) (DietTemplateSeedResult, error) {
	result := DietTemplateSeedResult{}
	if kindergartenID == 0 {
		return result, errors.New("diet template: kindergarten_id 不能为 0")
	}

	foods, err := s.foods.ListByKindergarten(ctx, kindergartenID)
	if err != nil {
		return result, fmt.Errorf("list kindergarten foods: %w", err)
	}
	if len(foods) == 0 {
		result.SkippedNoFoods = true
		slog.Warn("diet seed: 园所没有食物，跳过模板种子", "kindergarten_id", kindergartenID)
		return result, nil
	}

	byName := make(map[string]domain.Food, len(foods))
	for _, food := range foods {
		if _, ok := byName[food.Name]; !ok {
			byName[food.Name] = food
		}
	}

	missing := make(map[string]bool)
	for _, spec := range dietTemplateSpecs() {
		existing, err := s.templates.FindByKey(ctx, kindergartenID, spec.Key)
		if err != nil {
			return result, fmt.Errorf("query template %q: %w", spec.Key, err)
		}
		if existing != nil {
			continue
		}

		items, templateMissing := buildDietTemplateItems(spec, kindergartenID, byName)
		for name := range templateMissing {
			missing[name] = true
		}
		if len(items) == 0 {
			result.SkippedTemplates = append(result.SkippedTemplates, spec.Key)
			slog.Warn("diet template: 模板食物均不在园所食物库，跳过该模板",
				"kindergarten_id", kindergartenID, "template_key", spec.Key)
			continue
		}
		if len(templateMissing) > 0 {
			slog.Warn("diet template: 部分食物在园所食物库中不存在，已跳过",
				"kindergarten_id", kindergartenID, "template_key", spec.Key,
				"missing_foods", strings.Join(sortedStringKeys(templateMissing), ","))
		}

		tpl := &domain.TemplatePlan{
			KindergartenID: kindergartenID,
			TemplateKey:    spec.Key,
			Name:           spec.Name,
			DayNum:         len(spec.Days),
			MealKeys:       dietTemplateMealKeys,
			Sort:           spec.Sort,
			Items:          items,
		}
		if err := s.templates.CreateWithItems(ctx, tpl); err != nil {
			return result, fmt.Errorf("create template %q: %w", spec.Key, err)
		}
		result.TemplatesCreated++
		result.ItemsCreated += len(items)
	}

	result.MissingFoods = sortedStringKeys(missing)
	return result, nil
}

// copyPresetFoods 把 kg=0 预置行按 preset_key 复制到本园，只补缺失键。
func (s *DietSeedService) copyPresetFoods(ctx context.Context, kindergartenID uint64) (int, error) {
	presets, err := s.foods.ListByKindergarten(ctx, domain.DietPlatformKindergartenID)
	if err != nil {
		return 0, fmt.Errorf("list platform preset foods: %w", err)
	}
	if len(presets) == 0 {
		slog.Warn("diet seed: 平台预置食物库为空（尚未导入授权 CSV），跳过复制",
			"kindergarten_id", kindergartenID)
		return 0, nil
	}

	existing, err := s.foods.ListByKindergarten(ctx, kindergartenID)
	if err != nil {
		return 0, fmt.Errorf("list kindergarten foods: %w", err)
	}
	known := make(map[string]bool, len(existing))
	for _, food := range existing {
		if food.PresetKey != "" {
			known[food.PresetKey] = true
		}
	}

	toCreate := make([]domain.Food, 0, len(presets))
	for i := range presets {
		src := &presets[i]
		if src.PresetKey == "" {
			slog.Warn("diet seed: 预置食物缺少 preset_key，无法幂等复制，已跳过",
				"food_id", src.ID, "name", src.Name)
			continue
		}
		if known[src.PresetKey] {
			continue
		}
		known[src.PresetKey] = true
		toCreate = append(toCreate, clonePresetFood(src, kindergartenID))
	}
	if err := s.foods.CreateBatch(ctx, toCreate); err != nil {
		return 0, fmt.Errorf("copy preset foods: %w", err)
	}
	return len(toCreate), nil
}

// clonePresetFood 复制预置行为园所副本；ID/时间戳必须清零，否则会被当成更新。
func clonePresetFood(src *domain.Food, kindergartenID uint64) domain.Food {
	dst := *src
	dst.ID = 0
	dst.CreatedAt = time.Time{}
	dst.UpdatedAt = time.Time{}
	dst.DeletedAt = gorm.DeletedAt{}
	dst.KindergartenID = kindergartenID
	dst.Source = domain.DietFoodSourceSystem
	dst.IsCustomized = false
	return dst
}

// buildDietTemplateItems 把模板初稿解析成本园 food_id；缺失食物名写入 missing。
func buildDietTemplateItems(
	spec dietTemplateSpec,
	kindergartenID uint64,
	byName map[string]domain.Food,
) ([]domain.TemplatePlanItem, map[string]bool) {
	items := make([]domain.TemplatePlanItem, 0, len(spec.Days)*3)
	missing := make(map[string]bool)
	sort := 0
	for _, day := range spec.Days {
		for _, meal := range day.Meals {
			for _, want := range meal.Foods {
				food, ok := byName[want.Name]
				if !ok {
					missing[want.Name] = true
					continue
				}
				sort++
				items = append(items, domain.TemplatePlanItem{
					KindergartenID: kindergartenID,
					Weekday:        day.Weekday,
					MealKey:        meal.MealKey,
					RecipeName:     meal.RecipeName,
					FoodID:         food.ID,
					FoodName:       food.Name,
					Grams:          want.Grams,
					Sort:           sort,
				})
			}
		}
	}
	return items, missing
}

// dietDriKey 是 DRIs 判重键（唯一索引 age_band + nutrient_key 的字符串形式）。
func dietDriKey(ageBand int, nutrientKey string) string {
	return strconv.Itoa(ageBand) + "/" + nutrientKey
}

// dietDriSeedRows 生成 3/4/5/6 岁 × 11 项 = 44 行 DRIs 目标值。
//
// 取值来源 docs/research/2026-11-01-nutrition-data/notes.md：
//   - 能量用男值列 1350/1450/1600/1700，蛋白质 45/50/55/55（§5.4 反推口径已确认）；
//   - 脂肪 = 能量 × 30% ÷ 9，碳水 = (能量 − 蛋白质×4 − 脂肪×9) ÷ 4（design D6）；
//   - 维生素A 500/600/600/600、VC 60/70/70/70、钙 600/800/800/800、
//     锌 9/12/12/12（§5.4 表中明确列出）；
//   - B1/B2：4–6 岁 0.7（§5.1），3 岁未单列，按 §5.4 的加权观测 0.67 与 3 岁权重
//     26.7% 反解为 0.6（0.7 − (0.7−0.6)×0.267 = 0.673 ≈ 0.67），不是凭空取值；
//   - 铁：观测值 12 对四档都成立，四档均取 12（§5.4）。
//
// 视黄醇/胡萝卜素不进目标表：报表页不展示它们（design D2/D6）。
// 千焦不单独成行：报表侧按千卡 × 4.186 推导。
func dietDriSeedRows() []domain.DriAgeTarget {
	energy := [4]float64{1350, 1450, 1600, 1700}
	protein := [4]float64{45, 50, 55, 55}
	fats := [4]float64{}
	carb := [4]float64{}
	for i := range energy {
		fats[i] = roundDietValue(energy[i] * 0.30 / 9)
		carb[i] = roundDietValue((energy[i] - protein[i]*4 - energy[i]*0.30) / 4)
	}

	specs := []struct {
		key    string
		label  string
		unit   string
		values [4]float64
	}{
		{"energy", "能量", "千卡", energy},
		{"protein", "蛋白质", "克", protein},
		{"fats", "脂肪", "克", fats},
		{"carb", "碳水化合物", "克", carb},
		{"vitamin_a", "维生素A", "微克RE", [4]float64{500, 600, 600, 600}},
		{"vitamin_b1", "维生素B1", "毫克", [4]float64{0.6, 0.7, 0.7, 0.7}},
		{"vitamin_b2", "维生素B2", "毫克", [4]float64{0.6, 0.7, 0.7, 0.7}},
		{"vitamin_c", "维生素C", "毫克", [4]float64{60, 70, 70, 70}},
		{"calcium", "钙", "毫克", [4]float64{600, 800, 800, 800}},
		{"zinc", "锌", "毫克", [4]float64{9, 12, 12, 12}},
		{"iron", "铁", "毫克", [4]float64{12, 12, 12, 12}},
	}

	rows := make([]domain.DriAgeTarget, 0, len(specs)*len(dietDriAgeBands))
	for i, spec := range specs {
		for j, ageBand := range dietDriAgeBands {
			rows = append(rows, domain.DriAgeTarget{
				AgeBand:     ageBand,
				NutrientKey: spec.key,
				Label:       spec.label,
				Unit:        spec.unit,
				Value:       spec.values[j],
				Sort:        i + 1,
			})
		}
	}
	return rows
}

func roundDietValue(v float64) float64 {
	return math.Round(v*10000) / 10000
}

// ---------------------------------------------------------------------------
// 授权 CSV 导入
// ---------------------------------------------------------------------------

// DietFoodImportRow 是解析并校验通过的一行导入数据。
// 分类只写名称：ID 在写库时按 (parent_id, name) upsert 得到。
type DietFoodImportRow struct {
	Line          int
	PresetKey     string
	Name          string
	SpellCode     string
	CategoryL1    string
	CategoryL2    string
	EdiblePercent float64
	Group12       *int
	Group16       domain.DietFoodGroup16
	SourceFlags   []string
	Nutrients     *DietFoodNutrientsInput
}

// DietFoodImportError 是一行导入失败（解析/类别/数值）。
type DietFoodImportError struct {
	Line      int    `json:"line"`
	PresetKey string `json:"preset_key,omitempty"`
	Reason    string `json:"reason"`
}

// DietFoodImportConflict 是分发到园所时因「园所已修改」而未覆盖的行（TC-NEW-DIE-0016）。
type DietFoodImportConflict struct {
	PresetKey string `json:"preset_key"`
	Name      string `json:"name"`
}

// DietFoodDistributeResult 是 --kindergarten 分发计数。
type DietFoodDistributeResult struct {
	Added     int                      `json:"added"`
	Updated   int                      `json:"updated"`
	Skipped   int                      `json:"skipped"`
	Conflicts []DietFoodImportConflict `json:"conflicts,omitempty"`
}

// DietFoodImportResult 是导入结果：kg=0 预置库四类计数 + 可选分发计数。
type DietFoodImportResult struct {
	Globals     DietGlobalSeedResult     `json:"globals"`
	Added       int                      `json:"added"`
	Updated     int                      `json:"updated"`
	Skipped     int                      `json:"skipped"`
	Errors      []DietFoodImportError    `json:"errors,omitempty"`
	Distributed DietFoodDistributeResult `json:"distributed"`
}

// CSV 表头映射。营养列名与 Food 的字段名/json 标签一致，方便授权数据直接对齐。
type dietFoodNutrientColumn struct {
	Name   string
	Assign func(*DietFoodNutrientsInput, *float64)
}

var dietFoodImportFixedColumns = []string{
	"preset_key", "name", "spell_code", "category_l1", "category_l2", "edible_portion",
}

var dietFoodImportOptionalColumns = []string{"group12", "group16", "source_flags"}

var dietFoodNutrientColumns = []dietFoodNutrientColumn{
	{"energy", func(n *DietFoodNutrientsInput, v *float64) { n.Energy = v }},
	{"heat", func(n *DietFoodNutrientsInput, v *float64) { n.Heat = v }},
	{"protein", func(n *DietFoodNutrientsInput, v *float64) { n.Protein = v }},
	{"fats", func(n *DietFoodNutrientsInput, v *float64) { n.Fats = v }},
	{"carb", func(n *DietFoodNutrientsInput, v *float64) { n.Carb = v }},
	{"dietary_fiber", func(n *DietFoodNutrientsInput, v *float64) { n.DietaryFiber = v }},
	{"ash", func(n *DietFoodNutrientsInput, v *float64) { n.Ash = v }},
	{"water", func(n *DietFoodNutrientsInput, v *float64) { n.Water = v }},
	{"cholesterol", func(n *DietFoodNutrientsInput, v *float64) { n.Cholesterol = v }},
	{"retinol", func(n *DietFoodNutrientsInput, v *float64) { n.Retinol = v }},
	{"carotin", func(n *DietFoodNutrientsInput, v *float64) { n.Carotin = v }},
	{"vitamin_a", func(n *DietFoodNutrientsInput, v *float64) { n.VitaminA = v }},
	{"vitamin_b1", func(n *DietFoodNutrientsInput, v *float64) { n.VitaminB1 = v }},
	{"vitamin_b2", func(n *DietFoodNutrientsInput, v *float64) { n.VitaminB2 = v }},
	{"nicotinic_acid", func(n *DietFoodNutrientsInput, v *float64) { n.NicotinicAcid = v }},
	{"vitamin_c", func(n *DietFoodNutrientsInput, v *float64) { n.VitaminC = v }},
	{"vitamin_e", func(n *DietFoodNutrientsInput, v *float64) { n.VitaminE = v }},
	{"calcium", func(n *DietFoodNutrientsInput, v *float64) { n.Calcium = v }},
	{"phosphorus", func(n *DietFoodNutrientsInput, v *float64) { n.Phosphorus = v }},
	{"potassium", func(n *DietFoodNutrientsInput, v *float64) { n.Potassium = v }},
	{"sodium", func(n *DietFoodNutrientsInput, v *float64) { n.Sodium = v }},
	{"magnesium", func(n *DietFoodNutrientsInput, v *float64) { n.Magnesium = v }},
	{"iron", func(n *DietFoodNutrientsInput, v *float64) { n.Iron = v }},
	{"zinc", func(n *DietFoodNutrientsInput, v *float64) { n.Zinc = v }},
	{"selenium", func(n *DietFoodNutrientsInput, v *float64) { n.Selenium = v }},
	{"copper", func(n *DietFoodNutrientsInput, v *float64) { n.Copper = v }},
	{"manganese", func(n *DietFoodNutrientsInput, v *float64) { n.Manganese = v }},
}

// DietFoodImportRequiredColumns 返回必须出现在表头里的列（含 spell_code，但值可空）。
func DietFoodImportRequiredColumns() []string {
	out := make([]string, 0, len(dietFoodImportFixedColumns)+len(dietFoodNutrientColumns))
	out = append(out, dietFoodImportFixedColumns...)
	for _, col := range dietFoodNutrientColumns {
		out = append(out, col.Name)
	}
	return out
}

// DietFoodImportOptionalColumns 返回可选列。
func DietFoodImportOptionalColumns() []string {
	return append([]string(nil), dietFoodImportOptionalColumns...)
}

// DietFoodImportNutrientColumns 返回 27 个营养列名（供 --help 与文档使用）。
func DietFoodImportNutrientColumns() []string {
	out := make([]string, 0, len(dietFoodNutrientColumns))
	for _, col := range dietFoodNutrientColumns {
		out = append(out, col.Name)
	}
	return out
}

// ParseDietFoodCSV 解析授权 CSV。返回可直接写库的行与被拒绝的行；
// 只有表头缺列这类整批性问题才返回 error。行级错误不中断整批。
//
// 缺失值语义：空字符串 / "-" / "—" 一律解析为 nil（写 NULL），不得写成 0。
func ParseDietFoodCSV(r io.Reader) ([]DietFoodImportRow, []DietFoodImportError, error) {
	if r == nil {
		return nil, nil, errors.New("CSV reader 为空")
	}
	br := bufio.NewReader(r)
	if bom, err := br.Peek(3); err == nil && bytes.Equal(bom, []byte{0xEF, 0xBB, 0xBF}) {
		_, _ = br.Discard(3)
	}
	reader := csv.NewReader(br)
	reader.TrimLeadingSpace = true

	header, err := reader.Read()
	if errors.Is(err, io.EOF) {
		return nil, nil, errors.New("CSV 为空：缺少表头")
	}
	if err != nil {
		return nil, nil, fmt.Errorf("读取表头失败: %w", err)
	}
	index := make(map[string]int, len(header))
	for i, name := range header {
		index[normalizeDietFoodHeader(name)] = i
	}
	for _, column := range DietFoodImportRequiredColumns() {
		if _, ok := index[column]; !ok {
			return nil, nil, fmt.Errorf("CSV 缺少必需列 %q（当前表头：%s）",
				column, strings.Join(header, ","))
		}
	}

	rows := make([]DietFoodImportRow, 0, 64)
	rowErrors := make([]DietFoodImportError, 0)
	seen := make(map[string]int)
	line := 1
	for {
		record, err := reader.Read()
		line++
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			rowErrors = append(rowErrors, DietFoodImportError{Line: line, Reason: "CSV 记录读取失败: " + err.Error()})
			continue
		}
		row, parseErr := parseDietFoodImportRecord(index, record, line)
		if parseErr != nil {
			rowErrors = append(rowErrors, DietFoodImportError{
				Line: line, PresetKey: row.PresetKey, Reason: parseErr.Error(),
			})
			continue
		}
		if prev, ok := seen[row.PresetKey]; ok {
			rowErrors = append(rowErrors, DietFoodImportError{
				Line:      line,
				PresetKey: row.PresetKey,
				Reason:    fmt.Sprintf("preset_key 与第 %d 行重复", prev),
			})
			continue
		}
		seen[row.PresetKey] = line
		rows = append(rows, row)
	}
	return rows, rowErrors, nil
}

func normalizeDietFoodHeader(name string) string {
	name = strings.TrimSpace(name)
	// 用 rune 构造 BOM，避免源码里出现会被 Go 编译器拒绝的字面 BOM 字符。
	name = strings.TrimPrefix(name, string(rune(0xFEFF)))
	return strings.ToLower(name)
}

func parseDietFoodImportRecord(
	index map[string]int,
	record []string,
	line int,
) (DietFoodImportRow, error) {
	row := DietFoodImportRow{Line: line}
	get := func(column string) string {
		at, ok := index[column]
		if !ok || at >= len(record) {
			return ""
		}
		return strings.TrimSpace(record[at])
	}

	row.PresetKey = get("preset_key")
	if row.PresetKey == "" {
		return row, errors.New("preset_key 不能为空")
	}
	if len(row.PresetKey) > 128 {
		return row, errors.New("preset_key 不能超过 128 个字符")
	}

	row.Name = get("name")
	if row.Name == "" {
		return row, errors.New("name 不能为空")
	}
	if utf8.RuneCountInString(row.Name) > 128 {
		return row, errors.New("name 不能超过 128 个字符")
	}

	row.SpellCode = strings.ToUpper(get("spell_code"))
	if row.SpellCode == "" {
		row.SpellCode = dietSpellCode(row.Name)
	}
	if utf8.RuneCountInString(row.SpellCode) > 64 {
		return row, errors.New("spell_code 不能超过 64 个字符")
	}

	row.CategoryL1 = get("category_l1")
	if !dietRootCategoryKnown(row.CategoryL1) {
		return row, fmt.Errorf("未知一级分类 %q", row.CategoryL1)
	}
	row.CategoryL2 = get("category_l2")
	if row.CategoryL2 == "" {
		return row, errors.New("category_l2 不能为空")
	}
	if utf8.RuneCountInString(row.CategoryL2) > 64 {
		return row, errors.New("category_l2 不能超过 64 个字符")
	}

	edibleRaw := get("edible_portion")
	if edibleRaw == "" {
		return row, errors.New("edible_portion 不能为空")
	}
	edible, err := strconv.ParseFloat(edibleRaw, 64)
	if err != nil || math.IsNaN(edible) || math.IsInf(edible, 0) || edible < 0 || edible > 100 {
		return row, fmt.Errorf("edible_portion 必须是 0–100 的数字: %q", edibleRaw)
	}
	row.EdiblePercent = edible

	group12, err := parseDietGroup12Column(get("group12"))
	if err != nil {
		return row, err
	}
	if group12 == nil {
		if value, ok := dietGroup12ByRootCategory[row.CategoryL1]; ok {
			v := value
			group12 = &v
		}
	}
	row.Group12 = group12

	group16Raw := strings.ToLower(get("group16"))
	if group16Raw == "" {
		row.Group16 = dietGroup16ByRootCategory[row.CategoryL1]
	} else {
		group16 := domain.DietFoodGroup16(group16Raw)
		if err := validateDietGroup16(group16); err != nil {
			return row, fmt.Errorf("group16 无效: %q", group16Raw)
		}
		row.Group16 = group16
	}

	flagsRaw := get("source_flags")
	if flagsRaw == "" {
		if flags, ok := dietSourceFlagsByRootCategory[row.CategoryL1]; ok {
			row.SourceFlags = append([]string(nil), flags...)
		}
	} else {
		rawFlags := strings.FieldsFunc(flagsRaw, func(r rune) bool { return r == '|' || r == ';' })
		flags, err := normalizeDietSourceFlags(rawFlags)
		if err != nil {
			return row, fmt.Errorf("source_flags 无效: %q", flagsRaw)
		}
		row.SourceFlags = flags
	}

	nutrients := &DietFoodNutrientsInput{}
	for _, column := range dietFoodNutrientColumns {
		value, err := parseDietNullFloat(get(column.Name))
		if err != nil {
			return row, fmt.Errorf("%s: %w", column.Name, err)
		}
		column.Assign(nutrients, value)
	}
	row.Nutrients = nutrients
	return row, nil
}

func parseDietGroup12Column(raw string) (*int, error) {
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return nil, fmt.Errorf("group12 必须是 37–48 的整数: %q", raw)
	}
	if err := validateDietGroup12(value); err != nil {
		return nil, fmt.Errorf("group12 无效: %q", raw)
	}
	return &value, nil
}

// parseDietNullFloat 把空值/破折号解析为 NULL，非法数字返回错误。
// 负数、NaN、Inf 一律拒绝：营养浓度不存在这些取值。
func parseDietNullFloat(raw string) (*float64, error) {
	switch raw {
	case "", "-", "—":
		return nil, nil
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return nil, fmt.Errorf("不是合法数字: %q", raw)
	}
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return nil, fmt.Errorf("不是有限数字: %q", raw)
	}
	if value < 0 {
		return nil, fmt.Errorf("不能为负数: %q", raw)
	}
	return &value, nil
}

func dietRootCategoryKnown(name string) bool {
	for _, official := range dietRootCategoryNames {
		if official == name {
			return true
		}
	}
	return false
}

// ImportFoods 把已解析的行幂等写入 kg=0 平台预置库；targetKindergartenID > 0 时
// 再把同样的内容分发到该园。行级类别问题写入 Errors，不中断整批。
func (s *DietSeedService) ImportFoods(
	ctx context.Context,
	rows []DietFoodImportRow,
	targetKindergartenID uint64,
) (DietFoodImportResult, error) {
	result := DietFoodImportResult{}

	if targetKindergartenID > 0 {
		var count int64
		if err := s.db.WithContext(ctx).Model(&domain.Kindergarten{}).
			Where("id = ?", targetKindergartenID).Count(&count).Error; err != nil {
			return result, fmt.Errorf("query kindergarten %d: %w", targetKindergartenID, err)
		}
		if count == 0 {
			return result, fmt.Errorf("kindergarten %d 不存在，拒绝分发", targetKindergartenID)
		}
	}

	globals, err := s.SeedDietGlobals(ctx)
	if err != nil {
		return result, err
	}
	result.Globals = globals

	categories, err := s.foods.ListCategories(ctx)
	if err != nil {
		return result, fmt.Errorf("list food categories: %w", err)
	}
	roots := make(map[string]domain.FoodCategory, len(categories))
	for _, category := range categories {
		if category.Level == 1 {
			roots[category.Name] = category
		}
	}
	leafCache := make(map[string]uint64)

	for _, row := range rows {
		root, ok := roots[row.CategoryL1]
		if !ok {
			result.Errors = append(result.Errors, DietFoodImportError{
				Line: row.Line, PresetKey: row.PresetKey,
				Reason: fmt.Sprintf("一级分类 %q 未在平台分类表中找到", row.CategoryL1),
			})
			continue
		}
		leafID, err := s.ensureLeafCategory(ctx, root, row.CategoryL2, leafCache)
		if err != nil {
			return result, err
		}

		desired, err := dietFoodFromImportRow(row, leafID, domain.DietPlatformKindergartenID)
		if err != nil {
			return result, err
		}
		existing, err := s.foods.FindByPresetKey(ctx, domain.DietPlatformKindergartenID, row.PresetKey)
		if err != nil {
			return result, fmt.Errorf("query preset food %q: %w", row.PresetKey, err)
		}
		switch {
		case existing == nil:
			if err := s.foods.Create(ctx, desired); err != nil {
				return result, fmt.Errorf("create preset food %q: %w", row.PresetKey, err)
			}
			result.Added++
		case dietFoodContentEqual(existing, desired):
			result.Skipped++
		default:
			dietFoodApplyImportContent(existing, desired)
			if err := s.foods.Update(ctx, existing); err != nil {
				return result, fmt.Errorf("update preset food %q: %w", row.PresetKey, err)
			}
			result.Updated++
		}

		if targetKindergartenID > 0 {
			if err := s.distributeFood(ctx, targetKindergartenID, row, leafID, &result.Distributed); err != nil {
				return result, err
			}
		}
	}
	return result, nil
}

// ensureLeafCategory 按 (parent_id, name) upsert 二级分类；软删除行先复活再用。
func (s *DietSeedService) ensureLeafCategory(
	ctx context.Context,
	root domain.FoodCategory,
	name string,
	cache map[string]uint64,
) (uint64, error) {
	key := strconv.FormatUint(root.ID, 10) + "/" + name
	if id, ok := cache[key]; ok {
		return id, nil
	}
	existing, err := s.foods.FindCategoryByParentName(ctx, root.ID, name)
	if err != nil {
		return 0, fmt.Errorf("query food category %q/%q: %w", root.Name, name, err)
	}
	if existing != nil {
		if existing.DeletedAt.Valid {
			if err := s.foods.RestoreCategory(ctx, existing.ID); err != nil {
				return 0, fmt.Errorf("restore food category %q/%q: %w", root.Name, name, err)
			}
		}
		cache[key] = existing.ID
		return existing.ID, nil
	}
	category := &domain.FoodCategory{ParentID: root.ID, Level: 2, Name: name}
	if err := s.foods.CreateCategory(ctx, category); err != nil {
		// 并发导入时另一进程可能刚建好同名叶子；回查沿用而不是让整批失败。
		raced, findErr := s.foods.FindCategoryByParentName(ctx, root.ID, name)
		if findErr != nil {
			return 0, fmt.Errorf("query food category %q/%q after create failure: %w", root.Name, name, findErr)
		}
		if raced == nil {
			return 0, fmt.Errorf("create food category %q/%q: %w", root.Name, name, err)
		}
		if raced.DeletedAt.Valid {
			if err := s.foods.RestoreCategory(ctx, raced.ID); err != nil {
				return 0, fmt.Errorf("restore food category %q/%q: %w", root.Name, name, err)
			}
		}
		cache[key] = raced.ID
		return raced.ID, nil
	}
	cache[key] = category.ID
	return category.ID, nil
}

// distributeFood 把一行导入内容复制/更新到园所，语义见 design D4/TC-NEW-DIE-0016：
// 园所已修改的行（is_customized=true）绝不覆盖，内容不同时记为冲突。
func (s *DietSeedService) distributeFood(
	ctx context.Context,
	kindergartenID uint64,
	row DietFoodImportRow,
	leafID uint64,
	result *DietFoodDistributeResult,
) error {
	desired, err := dietFoodFromImportRow(row, leafID, kindergartenID)
	if err != nil {
		return err
	}
	existing, err := s.foods.FindByPresetKey(ctx, kindergartenID, row.PresetKey)
	if err != nil {
		return fmt.Errorf("query kindergarten food %q: %w", row.PresetKey, err)
	}
	switch {
	case existing == nil:
		if err := s.foods.Create(ctx, desired); err != nil {
			return fmt.Errorf("create kindergarten food %q: %w", row.PresetKey, err)
		}
		result.Added++
	case dietFoodContentEqual(existing, desired):
		result.Skipped++
	case existing.IsCustomized:
		result.Conflicts = append(result.Conflicts, DietFoodImportConflict{
			PresetKey: row.PresetKey, Name: existing.Name,
		})
	default:
		dietFoodApplyImportContent(existing, desired)
		if err := s.foods.Update(ctx, existing); err != nil {
			return fmt.Errorf("update kindergarten food %q: %w", row.PresetKey, err)
		}
		result.Updated++
	}
	return nil
}

func dietFoodFromImportRow(
	row DietFoodImportRow,
	categoryID uint64,
	kindergartenID uint64,
) (*domain.Food, error) {
	food := &domain.Food{
		KindergartenID: kindergartenID,
		Name:           row.Name,
		SpellCode:      row.SpellCode,
		CategoryID:     categoryID,
		EdiblePercent:  row.EdiblePercent,
		Source:         domain.DietFoodSourceSystem,
		Group12:        row.Group12,
		Group16:        row.Group16,
		SourceFlags:    strings.Join(row.SourceFlags, ","),
		PresetKey:      row.PresetKey,
	}
	if row.Nutrients != nil {
		if err := applyDietFoodNutrients(food, row.Nutrients); err != nil {
			return nil, fmt.Errorf("apply nutrients for %q: %w", row.PresetKey, err)
		}
	}
	return food, nil
}

// dietFoodImportContent 是可比较的导入内容快照。指针字段交给 reflect.DeepEqual，
// 因此 nil 与 0 会被区分开（NULL 语义不能被"值相同"误判）。
type dietFoodImportContent struct {
	Name          string
	SpellCode     string
	CategoryID    uint64
	EdiblePercent float64
	Source        domain.DietFoodSource
	Group12       *int
	Group16       domain.DietFoodGroup16
	SourceFlags   string
	Nutrients     [27]*float64
}

func dietFoodImportContentOf(food *domain.Food) dietFoodImportContent {
	return dietFoodImportContent{
		Name:          food.Name,
		SpellCode:     food.SpellCode,
		CategoryID:    food.CategoryID,
		EdiblePercent: food.EdiblePercent,
		Source:        food.Source,
		Group12:       food.Group12,
		Group16:       food.Group16,
		SourceFlags:   food.SourceFlags,
		Nutrients: [27]*float64{
			food.Energy, food.Heat, food.Protein, food.Fats, food.Carb,
			food.DietaryFiber, food.Ash, food.Water, food.Cholesterol,
			food.Retinol, food.Carotin, food.VitaminA, food.VitaminB1,
			food.VitaminB2, food.NicotinicAcid, food.VitaminC, food.VitaminE,
			food.Calcium, food.Phosphorus, food.Potassium, food.Sodium,
			food.Magnesium, food.Iron, food.Zinc, food.Selenium,
			food.Copper, food.Manganese,
		},
	}
}

func dietFoodContentEqual(a, b *domain.Food) bool {
	return reflect.DeepEqual(dietFoodImportContentOf(a), dietFoodImportContentOf(b))
}

// dietFoodApplyImportContent 只覆盖导入字段，保留 ID/租户/preset_key/is_customized。
func dietFoodApplyImportContent(dst, src *domain.Food) {
	dst.Name = src.Name
	dst.SpellCode = src.SpellCode
	dst.CategoryID = src.CategoryID
	dst.EdiblePercent = src.EdiblePercent
	dst.Source = src.Source
	dst.Group12 = src.Group12
	dst.Group16 = src.Group16
	dst.SourceFlags = src.SourceFlags

	dst.Energy = src.Energy
	dst.Heat = src.Heat
	dst.Protein = src.Protein
	dst.Fats = src.Fats
	dst.Carb = src.Carb
	dst.DietaryFiber = src.DietaryFiber
	dst.Ash = src.Ash
	dst.Water = src.Water
	dst.Cholesterol = src.Cholesterol
	dst.Retinol = src.Retinol
	dst.Carotin = src.Carotin
	dst.VitaminA = src.VitaminA
	dst.VitaminB1 = src.VitaminB1
	dst.VitaminB2 = src.VitaminB2
	dst.NicotinicAcid = src.NicotinicAcid
	dst.VitaminC = src.VitaminC
	dst.VitaminE = src.VitaminE
	dst.Calcium = src.Calcium
	dst.Phosphorus = src.Phosphorus
	dst.Potassium = src.Potassium
	dst.Sodium = src.Sodium
	dst.Magnesium = src.Magnesium
	dst.Iron = src.Iron
	dst.Zinc = src.Zinc
	dst.Selenium = src.Selenium
	dst.Copper = src.Copper
	dst.Manganese = src.Manganese
}

func sortedStringKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}
