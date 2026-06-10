package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

type ClassRepo struct {
	db *gorm.DB
}

func NewClassRepo(db *gorm.DB) *ClassRepo {
	return &ClassRepo{db: db}
}

func (r *ClassRepo) CountInKindergarten(ctx context.Context, kindergartenID uint64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.Class{}).
		Where("kindergarten_id = ?", kindergartenID).
		Count(&n).Error
	return n, err
}

func (r *ClassRepo) List(
	ctx context.Context,
	kindergartenID uint64,
	grade, keyword string,
	offset, limit int,
) ([]domain.Class, int64, error) {
	q := r.db.WithContext(ctx).Model(&domain.Class{}).Where("kindergarten_id = ?", kindergartenID)
	if grade != "" {
		q = q.Where("grade = ?", grade)
	}
	if keyword != "" {
		q = q.Where("name LIKE ?", "%"+keyword+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []domain.Class
	if err := q.Order("id DESC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *ClassRepo) FindByID(ctx context.Context, kindergartenID, classID uint64) (*domain.Class, error) {
	var c domain.Class
	err := r.db.WithContext(ctx).
		Where("id = ? AND kindergarten_id = ?", classID, kindergartenID).
		First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &c, err
}

func (r *ClassRepo) Create(ctx context.Context, c *domain.Class) error {
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *ClassRepo) Save(ctx context.Context, c *domain.Class) error {
	return r.db.WithContext(ctx).Save(c).Error
}

func (r *ClassRepo) Delete(ctx context.Context, kindergartenID, classID uint64) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND kindergarten_id = ?", classID, kindergartenID).
		Delete(&domain.Class{})
	return res.Error
}

func (r *ClassRepo) CountStudentsInClass(ctx context.Context, kindergartenID, classID uint64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.Student{}).
		Where("kindergarten_id = ? AND class_id = ? AND status = ?", kindergartenID, classID, 1).
		Count(&n).Error
	return n, err
}

func (r *ClassRepo) CountStudentsByClassIDs(ctx context.Context, kindergartenID uint64, classIDs []uint64) (map[uint64]int64, error) {
	out := make(map[uint64]int64)
	if len(classIDs) == 0 {
		return out, nil
	}
	type row struct {
		ClassID uint64 `gorm:"column:class_id"`
		Cnt     int64  `gorm:"column:cnt"`
	}
	var rows []row
	err := r.db.WithContext(ctx).Model(&domain.Student{}).
		Select("class_id, COUNT(*) AS cnt").
		Where("kindergarten_id = ? AND status = ? AND class_id IN ?", kindergartenID, 1, classIDs).
		Group("class_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, x := range rows {
		out[x.ClassID] = x.Cnt
	}
	return out, nil
}

func (r *ClassRepo) ListTeachersByClassIDs(ctx context.Context, classIDs []uint64) ([]domain.ClassTeacher, error) {
	if len(classIDs) == 0 {
		return nil, nil
	}
	var list []domain.ClassTeacher
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("class_id IN ?", classIDs).
		Order("class_id ASC, id ASC").
		Find(&list).Error
	return list, err
}

func (r *ClassRepo) replaceClassTeachersTx(tx *gorm.DB, classID uint64, teachers []domain.ClassTeacher) error {
	if err := tx.Unscoped().Where("class_id = ?", classID).Delete(&domain.ClassTeacher{}).Error; err != nil {
		return err
	}
	for i := range teachers {
		teachers[i].ClassID = classID
		teachers[i].ID = 0
		if err := tx.Create(&teachers[i]).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *ClassRepo) ReplaceClassTeachers(ctx context.Context, classID uint64, teachers []domain.ClassTeacher) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return r.replaceClassTeachersTx(tx, classID, teachers)
	})
}

func (r *ClassRepo) CountStudentsInClassWithIDs(
	ctx context.Context,
	kindergartenID, classID uint64,
	studentIDs []uint64,
) (int64, error) {
	if len(studentIDs) == 0 {
		return 0, nil
	}
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.Student{}).
		Where("kindergarten_id = ? AND class_id = ? AND status = ? AND id IN ?", kindergartenID, classID, 1, studentIDs).
		Count(&n).Error
	return n, err
}

func (r *ClassRepo) ListStudentsPreview(ctx context.Context, kindergartenID, classID uint64, limit int) ([]domain.Student, error) {
	var list []domain.Student
	err := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND class_id = ? AND status = ?", kindergartenID, classID, 1).
		Order("id ASC").
		Limit(limit).
		Find(&list).Error
	return list, err
}

func (r *ClassRepo) UpdateStudentsClass(
	ctx context.Context,
	kindergartenID, fromClassID, toClassID uint64,
	studentIDs []uint64,
) (int64, error) {
	if len(studentIDs) == 0 {
		return 0, nil
	}
	res := r.db.WithContext(ctx).Model(&domain.Student{}).
		Where("kindergarten_id = ? AND class_id = ? AND id IN ?", kindergartenID, fromClassID, studentIDs).
		Update("class_id", toClassID)
	return res.RowsAffected, res.Error
}

func (r *ClassRepo) CreateClassWithTeachers(ctx context.Context, class *domain.Class, teachers []domain.ClassTeacher) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(class).Error; err != nil {
			return err
		}
		return r.replaceClassTeachersTx(tx, class.ID, teachers)
	})
}
