package repository

import (
	"context"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

// PlatformLogRepo 平台方操作留痕。与园所 operation_logs 分表，避免平台行为
// 混入园所审计视图。
type PlatformLogRepo struct {
	db *gorm.DB
}

func NewPlatformLogRepo(db *gorm.DB) *PlatformLogRepo {
	return &PlatformLogRepo{db: db}
}

func (r *PlatformLogRepo) Create(ctx context.Context, row *domain.PlatformOperationLog) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *PlatformLogRepo) List(ctx context.Context, page, pageSize int, module string) ([]domain.PlatformOperationLog, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	q := r.db.WithContext(ctx).Model(&domain.PlatformOperationLog{})
	if module != "" {
		q = q.Where("module = ?", module)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []domain.PlatformOperationLog
	err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}
