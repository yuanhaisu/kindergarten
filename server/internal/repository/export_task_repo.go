package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

type ExportTaskRepo struct {
	db *gorm.DB
}

func NewExportTaskRepo(db *gorm.DB) *ExportTaskRepo {
	return &ExportTaskRepo{db: db}
}

func (r *ExportTaskRepo) Create(ctx context.Context, t *domain.ExportTask) error {
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *ExportTaskRepo) FindByPublicID(ctx context.Context, kindergartenID uint64, publicID string) (*domain.ExportTask, error) {
	var row domain.ExportTask
	err := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND public_id = ?", kindergartenID, publicID).
		First(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func (r *ExportTaskRepo) FindByID(ctx context.Context, id uint64) (*domain.ExportTask, error) {
	var row domain.ExportTask
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func (r *ExportTaskRepo) UpdateStatus(ctx context.Context, id uint64, status string, progress int) error {
	return r.db.WithContext(ctx).Model(&domain.ExportTask{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":   status,
			"progress": progress,
		}).Error
}

func (r *ExportTaskRepo) MarkCompleted(ctx context.Context, id uint64, localPath, fileName string, fileSize int64, expireAt, completedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&domain.ExportTask{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       "completed",
			"progress":     100,
			"local_path":   localPath,
			"file_name":    fileName,
			"file_size":    fileSize,
			"expire_at":    expireAt,
			"completed_at": completedAt,
		}).Error
}

func (r *ExportTaskRepo) MarkFailed(ctx context.Context, id uint64, msg string) error {
	return r.db.WithContext(ctx).Model(&domain.ExportTask{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":        "failed",
			"progress":      0,
			"error_message": msg,
		}).Error
}
