package repository

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"kindergarten-server/internal/domain"
)

// DietDriRepo 分年龄档 DRIs 目标值（全局只读，无租户字段）。
type DietDriRepo struct {
	db *gorm.DB
}

func NewDietDriRepo(db *gorm.DB) *DietDriRepo {
	return &DietDriRepo{db: db}
}

// ListByAge 返回指定年龄档的全部目标值，按 sort 升序。
func (r *DietDriRepo) ListByAge(ctx context.Context, ageBand int) ([]domain.DriAgeTarget, error) {
	var rows []domain.DriAgeTarget
	err := r.db.WithContext(ctx).
		Where("age_band = ?", ageBand).
		Order("sort ASC, id ASC").
		Find(&rows).Error
	return rows, err
}

// ListAllIncludingDeleted 返回全部目标值（含软删除行），供幂等种子判重。
// 唯一索引 (age_band, nutrient_key) 不区分软删除，漏掉被软删除的行会让
// 后续插入撞键，所以这里必须 Unscoped。
func (r *DietDriRepo) ListAllIncludingDeleted(ctx context.Context) ([]domain.DriAgeTarget, error) {
	var rows []domain.DriAgeTarget
	err := r.db.WithContext(ctx).Unscoped().
		Order("age_band ASC, sort ASC, id ASC").
		Find(&rows).Error
	return rows, err
}

// CreateBatch 插入一批目标值；返回真正写入的行数。
//
// 用 ON DUPLICATE KEY DO NOTHING：多个进程（服务启动 + 测试）可能同时播种，
// 并发插入同一 (age_band, nutrient_key) 时不应让整批失败。
func (r *DietDriRepo) CreateBatch(ctx context.Context, rows []domain.DriAgeTarget) (int64, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	res := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		CreateInBatches(&rows, 100)
	return res.RowsAffected, res.Error
}

// Restore 复活被软删除的目标值，语义同 RestoreCategory。
func (r *DietDriRepo) Restore(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Unscoped().
		Model(&domain.DriAgeTarget{}).
		Where("id = ?", id).
		Update("deleted_at", nil).Error
}
