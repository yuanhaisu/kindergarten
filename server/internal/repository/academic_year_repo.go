package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

type AcademicYearRepo struct {
	db *gorm.DB
}

func NewAcademicYearRepo(db *gorm.DB) *AcademicYearRepo {
	return &AcademicYearRepo{db: db}
}

func (r *AcademicYearRepo) ListByKindergarten(ctx context.Context, kindergartenID uint64) ([]domain.AcademicYear, error) {
	var list []domain.AcademicYear
	err := r.db.WithContext(ctx).
		Where("kindergarten_id = ?", kindergartenID).
		Order("is_current DESC, id DESC").
		Find(&list).Error
	return list, err
}

func (r *AcademicYearRepo) CountByKindergarten(ctx context.Context, kindergartenID uint64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.AcademicYear{}).
		Where("kindergarten_id = ?", kindergartenID).
		Count(&n).Error
	return n, err
}

func (r *AcademicYearRepo) FindByYear(ctx context.Context, kindergartenID uint64, year string) (*domain.AcademicYear, error) {
	var row domain.AcademicYear
	err := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND year = ?", kindergartenID, year).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

func (r *AcademicYearRepo) FindCurrent(ctx context.Context, kindergartenID uint64) (*domain.AcademicYear, error) {
	var row domain.AcademicYear
	err := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND is_current = ?", kindergartenID, true).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

func (r *AcademicYearRepo) FindLatest(ctx context.Context, kindergartenID uint64) (*domain.AcademicYear, error) {
	var row domain.AcademicYear
	err := r.db.WithContext(ctx).
		Where("kindergarten_id = ?", kindergartenID).
		Order("id DESC").
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

func (r *AcademicYearRepo) Create(ctx context.Context, row *domain.AcademicYear) error {
	return r.db.WithContext(ctx).Create(row).Error
}
