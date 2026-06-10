package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

type ParentRepo struct {
	db *gorm.DB
}

func NewParentRepo(db *gorm.DB) *ParentRepo {
	return &ParentRepo{db: db}
}

func (r *ParentRepo) parentListQuery(
	ctx context.Context,
	kindergartenID uint64,
	keyword string,
	classID *uint64,
) *gorm.DB {
	q := r.db.WithContext(ctx).Model(&domain.User{}).
		Joins("INNER JOIN user_roles ur ON ur.user_id = users.id AND ur.role = ? AND ur.kindergarten_id = ?", "parent", kindergartenID).
		Joins("INNER JOIN parent_profiles pp ON pp.user_id = users.id AND pp.kindergarten_id = ?", kindergartenID)
	if keyword != "" {
		kw := "%" + keyword + "%"
		q = q.Where("users.nickname LIKE ? OR users.phone LIKE ? OR pp.real_name LIKE ?", kw, kw, kw)
	}
	if classID != nil && *classID > 0 {
		q = q.Joins("INNER JOIN student_parents sp ON sp.user_id = users.id").
			Joins("INNER JOIN students s ON s.id = sp.student_id AND s.kindergarten_id = ? AND s.class_id = ?", kindergartenID, *classID)
	}
	return q
}

// ListParentUserIDs 按园所列家长用户 ID（去重）；内存分页。
func (r *ParentRepo) ListParentUserIDs(
	ctx context.Context,
	kindergartenID uint64,
	keyword string,
	classID *uint64,
	offset, limit int,
) ([]uint64, int64, error) {
	q := r.parentListQuery(ctx, kindergartenID, keyword, classID).Distinct("users.id").Order("users.id DESC")
	var all []uint64
	if err := q.Pluck("users.id", &all).Error; err != nil {
		return nil, 0, err
	}
	total := int64(len(all))
	if offset >= len(all) {
		return []uint64{}, total, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], total, nil
}

func (r *ParentRepo) FindProfile(ctx context.Context, kindergartenID, userID uint64) (*domain.ParentProfile, error) {
	var p domain.ParentProfile
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND kindergarten_id = ?", userID, kindergartenID).
		First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

func (r *ParentRepo) CreateProfile(ctx context.Context, p *domain.ParentProfile) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *ParentRepo) SaveProfile(ctx context.Context, p *domain.ParentProfile) error {
	return r.db.WithContext(ctx).Save(p).Error
}

func (r *ParentRepo) UnscopedDeleteProfile(ctx context.Context, kindergartenID, userID uint64) error {
	return r.db.WithContext(ctx).Unscoped().
		Where("user_id = ? AND kindergarten_id = ?", userID, kindergartenID).
		Delete(&domain.ParentProfile{}).Error
}

// ListStudentParentsForUsersInKindergarten 批量加载家长在某园所下的孩子关联（学生限定在该园所）。
func (r *ParentRepo) ListStudentParentsForUsersInKindergarten(
	ctx context.Context,
	kindergartenID uint64,
	userIDs []uint64,
) ([]domain.StudentParent, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	var list []domain.StudentParent
	err := r.db.WithContext(ctx).
		Preload("Student").
		Preload("Student.Class").
		Joins("INNER JOIN students s ON s.id = student_parents.student_id AND s.kindergarten_id = ?", kindergartenID).
		Where("student_parents.user_id IN ?", userIDs).
		Order("student_parents.user_id ASC, student_parents.student_id ASC").
		Find(&list).Error
	return list, err
}

// DeleteStudentParentsForUserInKindergarten 删除该家长与「本园学生」的关联。
func (r *ParentRepo) DeleteStudentParentsForUserInKindergarten(ctx context.Context, userID, kindergartenID uint64) error {
	sub := r.db.WithContext(ctx).Model(&domain.Student{}).
		Select("id").
		Where("kindergarten_id = ?", kindergartenID)
	return r.db.WithContext(ctx).Unscoped().
		Where("user_id = ? AND student_id IN (?)", userID, sub).
		Delete(&domain.StudentParent{}).Error
}
