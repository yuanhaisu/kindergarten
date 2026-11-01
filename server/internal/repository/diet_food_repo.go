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
		q = q.Where("name LIKE ?", "%"+kw+"%")
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

// CountByKindergarten 统计园所食物副本数；kindergartenID=0 时统计平台预置行，
// 供导入/种子流程判断是否已就绪。
func (r *DietFoodRepo) CountByKindergarten(ctx context.Context, kindergartenID uint64) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.Food{}).
		Where("kindergarten_id = ?", kindergartenID).
		Count(&total).Error
	return total, err
}
