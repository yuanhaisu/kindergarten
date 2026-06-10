package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

type KindergartenAppSettingRepo struct {
	db *gorm.DB
}

func NewKindergartenAppSettingRepo(db *gorm.DB) *KindergartenAppSettingRepo {
	return &KindergartenAppSettingRepo{db: db}
}

func (r *KindergartenAppSettingRepo) FindByKindergartenID(ctx context.Context, kindergartenID uint64) (*domain.KindergartenAppSetting, error) {
	var row domain.KindergartenAppSetting
	err := r.db.WithContext(ctx).Where("kindergarten_id = ?", kindergartenID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

func (r *KindergartenAppSettingRepo) Create(ctx context.Context, row *domain.KindergartenAppSetting) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *KindergartenAppSettingRepo) Save(ctx context.Context, row *domain.KindergartenAppSetting) error {
	return r.db.WithContext(ctx).Save(row).Error
}
