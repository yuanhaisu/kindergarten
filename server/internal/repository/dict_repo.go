package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

type DictRepo struct {
	db *gorm.DB
}

func NewDictRepo(db *gorm.DB) *DictRepo {
	return &DictRepo{db: db}
}

// ==================== Dict Types ====================

func (r *DictRepo) ListTypes(ctx context.Context, kgID uint64) ([]domain.SysDictType, error) {
	var rows []domain.SysDictType
	err := r.db.WithContext(ctx).
		Where("kindergarten_id = ?", kgID).
		Order("sort_order ASC, id ASC").
		Find(&rows).Error
	return rows, err
}

func (r *DictRepo) FindTypeByID(ctx context.Context, kgID, typeID uint64) (*domain.SysDictType, error) {
	var row domain.SysDictType
	err := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND id = ?", kgID, typeID).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

func (r *DictRepo) FindTypeByKey(ctx context.Context, kgID uint64, typeKey string) (*domain.SysDictType, error) {
	var row domain.SysDictType
	err := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND type_key = ?", kgID, typeKey).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

func (r *DictRepo) CreateType(ctx context.Context, t *domain.SysDictType) error {
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *DictRepo) UpdateType(ctx context.Context, t *domain.SysDictType) error {
	return r.db.WithContext(ctx).Save(t).Error
}

// DeleteType 软删除字典类型。返回被影响的行数。
func (r *DictRepo) DeleteType(ctx context.Context, kgID, typeID uint64) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND id = ?", kgID, typeID).
		Delete(&domain.SysDictType{})
	return result.RowsAffected, result.Error
}

// ==================== Dict Items ====================

func (r *DictRepo) ListItemsByTypeID(ctx context.Context, typeID uint64) ([]domain.SysDictItem, error) {
	var rows []domain.SysDictItem
	err := r.db.WithContext(ctx).
		Where("type_id = ?", typeID).
		Order("sort_order ASC, id ASC").
		Find(&rows).Error
	return rows, err
}

func (r *DictRepo) ListEnabledItemsByTypeKey(ctx context.Context, kgID uint64, typeKey string) ([]domain.SysDictItem, error) {
	var rows []domain.SysDictItem
	err := r.db.WithContext(ctx).
		Joins("JOIN sys_dict_types ON sys_dict_types.id = sys_dict_items.type_id").
		Where("sys_dict_types.kindergarten_id = ? AND sys_dict_types.type_key = ? AND sys_dict_items.is_enabled = ?", kgID, typeKey, true).
		Order("sys_dict_items.sort_order ASC, sys_dict_items.id ASC").
		Find(&rows).Error
	return rows, err
}

func (r *DictRepo) CountItemsByTypeID(ctx context.Context, typeID uint64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.SysDictItem{}).
		Where("type_id = ?", typeID).
		Count(&count).Error
	return count, err
}

func (r *DictRepo) FindItemByID(ctx context.Context, itemID uint64) (*domain.SysDictItem, error) {
	var row domain.SysDictItem
	err := r.db.WithContext(ctx).Where("id = ?", itemID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

func (r *DictRepo) CreateItem(ctx context.Context, item *domain.SysDictItem) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *DictRepo) UpdateItem(ctx context.Context, item *domain.SysDictItem) error {
	return r.db.WithContext(ctx).Save(item).Error
}

// DeleteItem 软删除字典项。返回被影响的行数。
func (r *DictRepo) DeleteItem(ctx context.Context, itemID uint64) (int64, error) {
	result := r.db.WithContext(ctx).Where("id = ?", itemID).Delete(&domain.SysDictItem{})
	return result.RowsAffected, result.Error
}

// BatchUpdateSortOrder 在事务中批量更新字典项的排序值。
func (r *DictRepo) BatchUpdateSortOrder(ctx context.Context, items []domain.SysDictItem) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			if err := tx.Model(&item).Update("sort_order", item.SortOrder).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
