package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

type UserRoleRepo struct {
	db *gorm.DB
}

func NewUserRoleRepo(db *gorm.DB) *UserRoleRepo {
	return &UserRoleRepo{db: db}
}

func (r *UserRoleRepo) FindByUserID(ctx context.Context, userID uint64) ([]domain.UserRole, error) {
	var roles []domain.UserRole
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&roles).Error
	return roles, err
}

func (r *UserRoleRepo) FindByUserIDAndRole(ctx context.Context, userID uint64, role string) (*domain.UserRole, error) {
	var ur domain.UserRole
	err := r.db.WithContext(ctx).Where("user_id = ? AND role = ?", userID, role).First(&ur).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &ur, err
}

func (r *UserRoleRepo) CountByRole(ctx context.Context, role string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.UserRole{}).Where("role = ?", role).Count(&count).Error
	return count, err
}

func (r *UserRoleRepo) CountByRoleInKindergarten(ctx context.Context, role string, kindergartenID uint64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.UserRole{}).
		Where("role = ? AND kindergarten_id = ?", role, kindergartenID).
		Count(&count).Error
	return count, err
}

func (r *UserRoleRepo) Create(ctx context.Context, ur *domain.UserRole) error {
	return r.db.WithContext(ctx).Create(ur).Error
}

func (r *UserRoleRepo) FindParentInKindergarten(ctx context.Context, userID, kindergartenID uint64) (*domain.UserRole, error) {
	var ur domain.UserRole
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND role = ? AND kindergarten_id = ?", userID, "parent", kindergartenID).
		First(&ur).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &ur, err
}

func (r *UserRoleRepo) FindTeacherInKindergarten(ctx context.Context, userID, kindergartenID uint64) (*domain.UserRole, error) {
	var ur domain.UserRole
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND role = ? AND kindergarten_id = ?", userID, "teacher", kindergartenID).
		First(&ur).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &ur, err
}

func (r *UserRoleRepo) DeleteTeacherRole(ctx context.Context, userID, kindergartenID uint64) error {
	return r.db.WithContext(ctx).
		Unscoped().
		Where("user_id = ? AND role = ? AND kindergarten_id = ?", userID, "teacher", kindergartenID).
		Delete(&domain.UserRole{}).Error
}

func (r *UserRoleRepo) DeleteParentRole(ctx context.Context, userID, kindergartenID uint64) error {
	return r.db.WithContext(ctx).
		Unscoped().
		Where("user_id = ? AND role = ? AND kindergarten_id = ?", userID, "parent", kindergartenID).
		Delete(&domain.UserRole{}).Error
}

func (r *UserRoleRepo) FindAdminInKindergarten(ctx context.Context, userID, kindergartenID uint64) (*domain.UserRole, error) {
	var ur domain.UserRole
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND role = ? AND kindergarten_id = ?", userID, "admin", kindergartenID).
		First(&ur).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &ur, err
}

func (r *UserRoleRepo) CountAdminsInKindergarten(ctx context.Context, kindergartenID uint64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.UserRole{}).
		Where("role = ? AND kindergarten_id = ?", "admin", kindergartenID).
		Count(&n).Error
	return n, err
}

func (r *UserRoleRepo) DeleteAdminRole(ctx context.Context, userID, kindergartenID uint64) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND role = ? AND kindergarten_id = ?", userID, "admin", kindergartenID).
		Delete(&domain.UserRole{}).Error
}
