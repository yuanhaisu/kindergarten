package repository

// timetable_repo.go —— 课表两层模型（年级模板 + 班级微调）的数据访问（align-class-calendar D5）。
// 全量替换采用「软删旧 + 插新」；本表刻意无唯一索引（软删行会占唯一键位，见迁移文件注释）。

import (
	"context"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

type TimetableRepo struct {
	db *gorm.DB
}

func NewTimetableRepo(db *gorm.DB) *TimetableRepo {
	return &TimetableRepo{db: db}
}

// ListTemplates 年级模板条目（按星期、节次排序）。
func (r *TimetableRepo) ListTemplates(ctx context.Context, kindergartenID uint64, grade string) ([]domain.TimetableTemplate, error) {
	var rows []domain.TimetableTemplate
	err := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND grade = ?", kindergartenID, grade).
		Order("weekday ASC, period ASC").
		Find(&rows).Error
	return rows, err
}

// ReplaceTemplates 全量替换某年级模板（事务：软删旧 + 插新）。
func (r *TimetableRepo) ReplaceTemplates(ctx context.Context, kindergartenID uint64, grade string, entries []domain.TimetableTemplate) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("kindergarten_id = ? AND grade = ?", kindergartenID, grade).
			Delete(&domain.TimetableTemplate{}).Error; err != nil {
			return err
		}
		for i := range entries {
			entries[i].ID = 0
			entries[i].KindergartenID = kindergartenID
			entries[i].Grade = grade
			if err := tx.Create(&entries[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ListOverrides 班级微调条目（按星期、节次排序）。
func (r *TimetableRepo) ListOverrides(ctx context.Context, kindergartenID, classID uint64) ([]domain.TimetableOverride, error) {
	var rows []domain.TimetableOverride
	err := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND class_id = ?", kindergartenID, classID).
		Order("weekday ASC, period ASC").
		Find(&rows).Error
	return rows, err
}

// ReplaceOverrides 全量替换某班微调（事务：软删旧 + 插新）。
func (r *TimetableRepo) ReplaceOverrides(ctx context.Context, kindergartenID, classID uint64, entries []domain.TimetableOverride) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("kindergarten_id = ? AND class_id = ?", kindergartenID, classID).
			Delete(&domain.TimetableOverride{}).Error; err != nil {
			return err
		}
		for i := range entries {
			entries[i].ID = 0
			entries[i].KindergartenID = kindergartenID
			entries[i].ClassID = classID
			if err := tx.Create(&entries[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
