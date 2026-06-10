package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

type KindergartenRepo struct {
	db *gorm.DB
}

func NewKindergartenRepo(db *gorm.DB) *KindergartenRepo {
	return &KindergartenRepo{db: db}
}

func (r *KindergartenRepo) Create(ctx context.Context, kg *domain.Kindergarten) error {
	return r.db.WithContext(ctx).Create(kg).Error
}

func (r *KindergartenRepo) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.Kindergarten{}).Count(&count).Error
	return count, err
}

func (r *KindergartenRepo) FindByID(ctx context.Context, id uint64) (*domain.Kindergarten, error) {
	var k domain.Kindergarten
	err := r.db.WithContext(ctx).First(&k, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &k, err
}

func (r *KindergartenRepo) UpdateByID(ctx context.Context, id uint64, cols map[string]any) error {
	if len(cols) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Model(&domain.Kindergarten{}).Where("id = ?", id).Updates(cols).Error
}
