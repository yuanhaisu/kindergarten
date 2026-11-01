package repository

import (
	"context"

	"gorm.io/gorm"

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
