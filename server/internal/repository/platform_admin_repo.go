package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

// PlatformAdminRepo 平台方账号仓储。与园所管理员完全独立（ADR-0014）。
type PlatformAdminRepo struct {
	db *gorm.DB
}

func NewPlatformAdminRepo(db *gorm.DB) *PlatformAdminRepo {
	return &PlatformAdminRepo{db: db}
}

func (r *PlatformAdminRepo) Create(ctx context.Context, a *domain.PlatformAdmin) error {
	return r.db.WithContext(ctx).Create(a).Error
}

func (r *PlatformAdminRepo) FindByUsername(ctx context.Context, username string) (*domain.PlatformAdmin, error) {
	var a domain.PlatformAdmin
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&a).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &a, err
}

func (r *PlatformAdminRepo) FindByID(ctx context.Context, id uint64) (*domain.PlatformAdmin, error) {
	var a domain.PlatformAdmin
	err := r.db.WithContext(ctx).First(&a, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &a, err
}

func (r *PlatformAdminRepo) Count(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.PlatformAdmin{}).Count(&n).Error
	return n, err
}
