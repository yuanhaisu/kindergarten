package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

// DistributorRepo 分销商名单仓储。分销商不登录系统（ADR-0016）。
type DistributorRepo struct {
	db *gorm.DB
}

func NewDistributorRepo(db *gorm.DB) *DistributorRepo {
	return &DistributorRepo{db: db}
}

func (r *DistributorRepo) Create(ctx context.Context, d *domain.Distributor) error {
	return r.db.WithContext(ctx).Create(d).Error
}

func (r *DistributorRepo) FindByID(ctx context.Context, id uint64) (*domain.Distributor, error) {
	var d domain.Distributor
	err := r.db.WithContext(ctx).First(&d, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &d, err
}

func (r *DistributorRepo) List(ctx context.Context, status int8) ([]domain.Distributor, int64, error) {
	q := r.db.WithContext(ctx).Model(&domain.Distributor{})
	if status != 0 {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []domain.Distributor
	if err := q.Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *DistributorRepo) Update(ctx context.Context, id uint64, cols map[string]any) error {
	if len(cols) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Model(&domain.Distributor{}).Where("id = ?", id).Updates(cols).Error
}

func (r *DistributorRepo) FindByIDs(ctx context.Context, ids []uint64) ([]domain.Distributor, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var list []domain.Distributor
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&list).Error
	return list, err
}
