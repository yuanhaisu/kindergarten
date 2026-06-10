package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

type TeacherRepo struct {
	db *gorm.DB
}

func NewTeacherRepo(db *gorm.DB) *TeacherRepo {
	return &TeacherRepo{db: db}
}

func (r *TeacherRepo) teacherListQuery(
	ctx context.Context,
	kindergartenID uint64,
	keyword string,
	classID *uint64,
	status *int8,
) *gorm.DB {
	q := r.db.WithContext(ctx).Model(&domain.User{}).
		Joins("INNER JOIN user_roles ur ON ur.user_id = users.id AND ur.role = ? AND ur.kindergarten_id = ?", "teacher", kindergartenID).
		Joins("INNER JOIN teacher_profiles tp ON tp.user_id = users.id AND tp.kindergarten_id = ?", kindergartenID)
	if keyword != "" {
		kw := "%" + keyword + "%"
		q = q.Where("users.nickname LIKE ? OR users.phone LIKE ?", kw, kw)
	}
	if status != nil {
		q = q.Where("tp.status = ?", *status)
	}
	if classID != nil && *classID > 0 {
		q = q.Joins("INNER JOIN class_teachers ct ON ct.user_id = users.id AND ct.class_id = ?", *classID).
			Joins("INNER JOIN classes c ON c.id = ct.class_id AND c.kindergarten_id = ?", kindergartenID)
	}
	return q
}

// ListTeacherUserIDs 按园所列老师用户 ID（去重）；数据量按园所规模在内存分页。
func (r *TeacherRepo) ListTeacherUserIDs(
	ctx context.Context,
	kindergartenID uint64,
	keyword string,
	classID *uint64,
	status *int8,
	offset, limit int,
) ([]uint64, int64, error) {
	q := r.teacherListQuery(ctx, kindergartenID, keyword, classID, status).Distinct("users.id").Order("users.id DESC")
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

func (r *TeacherRepo) FindProfile(ctx context.Context, kindergartenID, userID uint64) (*domain.TeacherProfile, error) {
	var p domain.TeacherProfile
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND kindergarten_id = ?", userID, kindergartenID).
		First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

func (r *TeacherRepo) CreateProfile(ctx context.Context, p *domain.TeacherProfile) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *TeacherRepo) SaveProfile(ctx context.Context, p *domain.TeacherProfile) error {
	return r.db.WithContext(ctx).Save(p).Error
}

func (r *TeacherRepo) UnscopedDeleteProfile(ctx context.Context, kindergartenID, userID uint64) error {
	return r.db.WithContext(ctx).Unscoped().
		Where("user_id = ? AND kindergarten_id = ?", userID, kindergartenID).
		Delete(&domain.TeacherProfile{}).Error
}

func (r *TeacherRepo) ListClassTeachersForUsersInKindergarten(
	ctx context.Context,
	kindergartenID uint64,
	userIDs []uint64,
) ([]domain.ClassTeacher, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	var list []domain.ClassTeacher
	err := r.db.WithContext(ctx).
		Preload("Class").
		Joins("INNER JOIN classes c ON c.id = class_teachers.class_id AND c.kindergarten_id = ?", kindergartenID).
		Where("class_teachers.user_id IN ?", userIDs).
		Order("class_teachers.user_id ASC, class_teachers.class_id ASC").
		Find(&list).Error
	return list, err
}

func (r *TeacherRepo) DeleteClassTeachersForUserInKindergarten(ctx context.Context, userID, kindergartenID uint64) error {
	sub := r.db.WithContext(ctx).Model(&domain.Class{}).
		Select("id").
		Where("kindergarten_id = ?", kindergartenID)
	return r.db.WithContext(ctx).Unscoped().
		Where("user_id = ? AND class_id IN (?)", userID, sub).
		Delete(&domain.ClassTeacher{}).Error
}

func (r *TeacherRepo) CreateClassTeacher(ctx context.Context, ct *domain.ClassTeacher) error {
	return r.db.WithContext(ctx).Create(ct).Error
}

func (r *TeacherRepo) CountGrowthLogsByCreator(ctx context.Context, kindergartenID, creatorID uint64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.GrowthDailyLog{}).
		Where("kindergarten_id = ? AND creator_id = ?", kindergartenID, creatorID).
		Count(&n).Error
	return n, err
}

func (r *TeacherRepo) CountNoticesByCreator(ctx context.Context, kindergartenID, creatorID uint64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.Notice{}).
		Where("kindergarten_id = ? AND creator_id = ?", kindergartenID, creatorID).
		Count(&n).Error
	return n, err
}
