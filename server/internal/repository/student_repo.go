package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

type StudentRepo struct {
	db *gorm.DB
}

func NewStudentRepo(db *gorm.DB) *StudentRepo {
	return &StudentRepo{db: db}
}

func (r *StudentRepo) List(
	ctx context.Context,
	kindergartenID uint64,
	classID *uint64,
	status *int8,
	keyword string,
	offset, limit int,
) ([]domain.Student, int64, error) {
	q := r.db.WithContext(ctx).Model(&domain.Student{}).Where("kindergarten_id = ?", kindergartenID)
	if classID != nil && *classID > 0 {
		q = q.Where("class_id = ?", *classID)
	}
	if status != nil {
		q = q.Where("status = ?", *status)
	}
	if keyword != "" {
		kw := "%" + keyword + "%"
		q = q.Where("name LIKE ? OR student_no LIKE ?", kw, kw)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []domain.Student
	if err := q.Preload("Class").Order("id DESC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *StudentRepo) FindByID(ctx context.Context, kindergartenID, studentID uint64) (*domain.Student, error) {
	var s domain.Student
	err := r.db.WithContext(ctx).
		Preload("Class").
		Where("id = ? AND kindergarten_id = ?", studentID, kindergartenID).
		First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *StudentRepo) Create(ctx context.Context, s *domain.Student) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *StudentRepo) Save(ctx context.Context, s *domain.Student) error {
	return r.db.WithContext(ctx).Save(s).Error
}

func (r *StudentRepo) Delete(ctx context.Context, kindergartenID, studentID uint64) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND kindergarten_id = ?", studentID, kindergartenID).
		Delete(&domain.Student{}).Error
}

func (r *StudentRepo) CountParentsByStudentIDs(ctx context.Context, studentIDs []uint64) (map[uint64]int64, error) {
	out := make(map[uint64]int64)
	if len(studentIDs) == 0 {
		return out, nil
	}
	type row struct {
		StudentID uint64 `gorm:"column:student_id"`
		Cnt       int64  `gorm:"column:cnt"`
	}
	var rows []row
	err := r.db.WithContext(ctx).Model(&domain.StudentParent{}).
		Select("student_id, COUNT(*) AS cnt").
		Where("student_id IN ?", studentIDs).
		Group("student_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, x := range rows {
		out[x.StudentID] = x.Cnt
	}
	return out, nil
}

func (r *StudentRepo) ListParentsWithUser(ctx context.Context, studentID uint64) ([]domain.StudentParent, error) {
	var list []domain.StudentParent
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("student_id = ?", studentID).
		Order("is_primary DESC, id ASC").
		Find(&list).Error
	return list, err
}

func (r *StudentRepo) ListPickupAuthorizations(ctx context.Context, studentID uint64) ([]domain.PickupAuthorization, error) {
	var list []domain.PickupAuthorization
	err := r.db.WithContext(ctx).
		Where("student_id = ?", studentID).
		Order("id ASC").
		Find(&list).Error
	return list, err
}

func (r *StudentRepo) UnscopedDeleteParentsByStudent(ctx context.Context, studentID uint64) error {
	return r.db.WithContext(ctx).Unscoped().Where("student_id = ?", studentID).Delete(&domain.StudentParent{}).Error
}

func (r *StudentRepo) UnscopedDeletePickupAuthsByStudent(ctx context.Context, studentID uint64) error {
	return r.db.WithContext(ctx).Unscoped().Where("student_id = ?", studentID).Delete(&domain.PickupAuthorization{}).Error
}

func (r *StudentRepo) UnscopedDeletePickupQRCodesByStudent(ctx context.Context, studentID uint64) error {
	return r.db.WithContext(ctx).Unscoped().Where("student_id = ?", studentID).Delete(&domain.PickupQRCode{}).Error
}

func (r *StudentRepo) CreateParentLink(ctx context.Context, sp *domain.StudentParent) error {
	return r.db.WithContext(ctx).Create(sp).Error
}

func (r *StudentRepo) CountInKindergarten(ctx context.Context, kindergartenID uint64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.Student{}).
		Where("kindergarten_id = ?", kindergartenID).
		Count(&n).Error
	return n, err
}

func (r *StudentRepo) CountActiveInKindergarten(ctx context.Context, kindergartenID uint64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.Student{}).
		Where("kindergarten_id = ? AND status = ?", kindergartenID, 1).
		Count(&n).Error
	return n, err
}

func (r *StudentRepo) FindClassByName(ctx context.Context, kindergartenID uint64, name string) (*domain.Class, error) {
	var c domain.Class
	err := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND name = ?", kindergartenID, name).
		First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &c, err
}

func (r *StudentRepo) ListForExport(
	ctx context.Context,
	kindergartenID uint64,
	classID *uint64,
	status *int8,
) ([]domain.Student, error) {
	q := r.db.WithContext(ctx).Model(&domain.Student{}).Where("kindergarten_id = ?", kindergartenID)
	if classID != nil && *classID > 0 {
		q = q.Where("class_id = ?", *classID)
	}
	if status != nil {
		q = q.Where("status = ?", *status)
	}
	var list []domain.Student
	err := q.Preload("Class").Order("class_id ASC, id ASC").Find(&list).Error
	return list, err
}
