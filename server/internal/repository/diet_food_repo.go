package repository

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

// DietFoodListFilter 食物列表筛选。
//
// CategoryID 传一级分类时自动包含其全部二级（参考产品按一级筛选也能命中叶子食物）；
// 传二级分类时只命中该叶子。0 表示不按分类过滤。
type DietFoodListFilter struct {
	CategoryID uint64
	Keyword    string
}

// DietFoodRepo 食物库。所有查询都以 kindergarten_id 为强制条件：
// kg=0 是平台预置哨兵行，绝不能被当作园所数据回传（ADR-0005）。
type DietFoodRepo struct {
	db *gorm.DB
}

func NewDietFoodRepo(db *gorm.DB) *DietFoodRepo {
	return &DietFoodRepo{db: db}
}

func (r *DietFoodRepo) List(
	ctx context.Context,
	kindergartenID uint64,
	filter DietFoodListFilter,
	offset, limit int,
) ([]domain.Food, int64, error) {
	q := r.db.WithContext(ctx).Model(&domain.Food{}).
		Where("kindergarten_id = ?", kindergartenID)

	if filter.CategoryID > 0 {
		q = q.Where(
			"category_id = ? OR category_id IN (?)",
			filter.CategoryID,
			r.db.WithContext(ctx).Model(&domain.FoodCategory{}).
				Select("id").
				Where("parent_id = ?", filter.CategoryID),
		)
	}
	if kw := strings.TrimSpace(filter.Keyword); kw != "" {
		// 名称模糊 + 拼音码前缀：管理员既可能按中文名，也可能按拼音首字母检索。
		q = q.Where(
			"(name LIKE ? OR spell_code LIKE ?)",
			"%"+kw+"%",
			strings.ToUpper(kw)+"%",
		)
	}

	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []domain.Food
	err := q.Session(&gorm.Session{}).
		Order("name ASC, id ASC").
		Offset(offset).Limit(limit).
		Find(&rows).Error
	return rows, total, err
}

func (r *DietFoodRepo) FindByID(ctx context.Context, kindergartenID, id uint64) (*domain.Food, error) {
	var row domain.Food
	err := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND id = ?", kindergartenID, id).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *DietFoodRepo) Create(ctx context.Context, food *domain.Food) error {
	if food == nil {
		return errors.New("diet food: nil food")
	}
	return r.db.WithContext(ctx).Create(food).Error
}

// Update 整体写回园所食物副本。用 Select("*") 让营养列中的 nil 落成 NULL；
// 契约要求修改时"未提交的键写 NULL"，而不是保留旧值或写 0。
func (r *DietFoodRepo) Update(ctx context.Context, food *domain.Food) error {
	if food == nil {
		return errors.New("diet food: nil food")
	}
	return r.db.WithContext(ctx).
		Model(&domain.Food{}).
		Where("id = ? AND kindergarten_id = ?", food.ID, food.KindergartenID).
		Select("*").
		Omit("id", "created_at", "deleted_at").
		Updates(food).Error
}

// Delete 软删除园所食物；跨园 ID 不会命中。
func (r *DietFoodRepo) Delete(ctx context.Context, kindergartenID, id uint64) (int64, error) {
	res := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND id = ?", kindergartenID, id).
		Delete(&domain.Food{})
	return res.RowsAffected, res.Error
}

// BatchDelete 软删除一批园所食物，返回实际删除行数。引用判定在服务层完成。
func (r *DietFoodRepo) BatchDelete(ctx context.Context, kindergartenID uint64, ids []uint64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	res := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND id IN ?", kindergartenID, ids).
		Delete(&domain.Food{})
	return res.RowsAffected, res.Error
}

// ListByKindergarten 返回本园全部食物（不分页），供种子复制与模板按名称解析使用。
// 复制必须看到全部预置行，不能用分页列表逐页拼接。
func (r *DietFoodRepo) ListByKindergarten(ctx context.Context, kindergartenID uint64) ([]domain.Food, error) {
	var rows []domain.Food
	err := r.db.WithContext(ctx).
		Where("kindergarten_id = ?", kindergartenID).
		Order("id ASC").
		Find(&rows).Error
	return rows, err
}

// FindByPresetKey 按 (kindergarten_id, preset_key) 查一行；不存在时返回 (nil, nil)。
// 只匹配未软删除行：本表没有 preset_key 唯一索引，软删除行不占键，
// 删除后重新导入应当插入新行而不是复活旧行。
func (r *DietFoodRepo) FindByPresetKey(
	ctx context.Context,
	kindergartenID uint64,
	presetKey string,
) (*domain.Food, error) {
	if presetKey == "" {
		return nil, nil
	}
	var row domain.Food
	err := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND preset_key = ?", kindergartenID, presetKey).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// CreateBatch 分块插入食物。一次导入可能上千行，单条 INSERT 会超过
// max_allowed_packet，因此交给 GORM 拆包。
func (r *DietFoodRepo) CreateBatch(ctx context.Context, foods []domain.Food) error {
	if len(foods) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(&foods, 100).Error
}

// FindCategoryByParentName 按 (parent_id, name) 查分类。
// 用 Unscoped：软删除行仍占用唯一索引，重建前必须先看到它。
func (r *DietFoodRepo) FindCategoryByParentName(
	ctx context.Context,
	parentID uint64,
	name string,
) (*domain.FoodCategory, error) {
	var row domain.FoodCategory
	err := r.db.WithContext(ctx).Unscoped().
		Where("parent_id = ? AND name = ?", parentID, name).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// CreateCategory 新建食物分类。
func (r *DietFoodRepo) CreateCategory(ctx context.Context, category *domain.FoodCategory) error {
	if category == nil {
		return errors.New("diet food: nil category")
	}
	return r.db.WithContext(ctx).Create(category).Error
}

// RestoreCategory 复活被软删除的分类。唯一索引仍占着键，重建只能复用原行。
func (r *DietFoodRepo) RestoreCategory(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Unscoped().
		Model(&domain.FoodCategory{}).
		Where("id = ?", id).
		Update("deleted_at", nil).Error
}

// CountByKindergarten 统计园所食物副本数；kindergartenID=0 时统计平台预置行，
// 供导入/种子流程判断是否已就绪。
func (r *DietFoodRepo) CountByKindergarten(ctx context.Context, kindergartenID uint64) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.Food{}).
		Where("kindergarten_id = ?", kindergartenID).
		Count(&total).Error
	return total, err
}

// FindByIDs 批量取本园食物；批量删除前用它区分「不存在/跨园」与「被引用」。
func (r *DietFoodRepo) FindByIDs(ctx context.Context, kindergartenID uint64, ids []uint64) ([]domain.Food, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var rows []domain.Food
	err := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND id IN ?", kindergartenID, ids).
		Find(&rows).Error
	return rows, err
}

// ListCategories 返回全局两级分类，一级在前；树形组装留给服务层。
// 分类是平台级只读数据，不带 kindergarten_id。
func (r *DietFoodRepo) ListCategories(ctx context.Context) ([]domain.FoodCategory, error) {
	var rows []domain.FoodCategory
	err := r.db.WithContext(ctx).
		Order("sort ASC, id ASC").
		Find(&rows).Error
	return rows, err
}

// FindCategoryByID 按 ID 取分类；不存在时返回 (nil, nil)。
func (r *DietFoodRepo) FindCategoryByID(ctx context.Context, id uint64) (*domain.FoodCategory, error) {
	var row domain.FoodCategory
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// ListReplaceCandidates 返回同一一级分类下的其他本园食物（契约 §三「同类替换候选」）。
// 契约刻意按一级分类而不是叶子分类：这样「猪瘦肉 → 牛腩」这类跨子类替换才能命中。
// rootCategoryID=0 时退化为不限分类（分类数据缺失时的兜底）。
func (r *DietFoodRepo) ListReplaceCandidates(
	ctx context.Context,
	kindergartenID, rootCategoryID, excludeID uint64,
	keyword string,
	limit int,
) ([]domain.Food, error) {
	q := r.db.WithContext(ctx).Model(&domain.Food{}).
		Where("kindergarten_id = ?", kindergartenID)
	if rootCategoryID > 0 {
		q = q.Where(
			"(category_id = ? OR category_id IN (?))",
			rootCategoryID,
			r.db.WithContext(ctx).Model(&domain.FoodCategory{}).
				Select("id").
				Where("parent_id = ?", rootCategoryID),
		)
	}
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	if kw := strings.TrimSpace(keyword); kw != "" {
		q = q.Where(
			"(name LIKE ? OR spell_code LIKE ?)",
			"%"+kw+"%",
			strings.ToUpper(kw)+"%",
		)
	}
	var rows []domain.Food
	err := q.Order("name ASC, id ASC").Limit(limit).Find(&rows).Error
	return rows, err
}
