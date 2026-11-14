package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"kindergarten-server/internal/domain"
)

type ConsentRepo struct {
	db *gorm.DB
}

func NewConsentRepo(db *gorm.DB) *ConsentRepo {
	return &ConsentRepo{db: db}
}

func (r *ConsentRepo) FindByUserAndVersion(ctx context.Context, userID uint64, version string) (*domain.UserConsent, error) {
	var c domain.UserConsent
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND policy_version = ?", userID, version).
		First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *ConsentRepo) ListByUser(ctx context.Context, userID uint64) ([]domain.UserConsent, error) {
	var rows []domain.UserConsent
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("consented_at DESC, id DESC").
		Find(&rows).Error
	return rows, err
}

// CreateIfAbsent 幂等写入：同一 user × version 已存在时保持首次记录，不覆盖 consented_at。
func (r *ConsentRepo) CreateIfAbsent(ctx context.Context, c *domain.UserConsent) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "policy_version"}},
		DoNothing: true,
	}).Create(c).Error
}
