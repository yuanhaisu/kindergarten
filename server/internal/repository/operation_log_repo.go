package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/pkg/pagination"
)

type OperationLogRepo struct {
	db *gorm.DB
}

func NewOperationLogRepo(db *gorm.DB) *OperationLogRepo {
	return &OperationLogRepo{db: db}
}

func (r *OperationLogRepo) Create(ctx context.Context, log *domain.OperationLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// OperationLogListRow 列表查询（LEFT JOIN users 取操作人昵称）
type OperationLogListRow struct {
	domain.OperationLog `gorm:"embedded"`
	OperatorNickname    string `gorm:"column:operator_nickname" json:"-"`
}

// ListForKindergarten 管理端操作日志分页列表，按 created_at DESC。
func (r *OperationLogRepo) ListForKindergarten(
	ctx context.Context,
	kindergartenID uint64,
	pq pagination.Query,
	dbModules []string,
	operatorID uint64,
	start, endExclusive *time.Time,
) ([]OperationLogListRow, int64, error) {
	q := r.db.WithContext(ctx).Model(&domain.OperationLog{}).
		Where("kindergarten_id = ?", kindergartenID)
	if len(dbModules) > 0 {
		q = q.Where("module IN ?", dbModules)
	}
	if operatorID > 0 {
		q = q.Where("user_id = ?", operatorID)
	}
	if start != nil {
		q = q.Where("created_at >= ?", *start)
	}
	if endExclusive != nil {
		q = q.Where("created_at < ?", *endExclusive)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return nil, 0, nil
	}

	var rows []OperationLogListRow
	listQ := r.db.WithContext(ctx).
		Table("operation_logs AS ol").
		Select("ol.*, COALESCE(u.nickname, '') AS operator_nickname").
		Joins("LEFT JOIN users u ON u.id = ol.user_id").
		Where("ol.kindergarten_id = ?", kindergartenID)
	if len(dbModules) > 0 {
		listQ = listQ.Where("ol.module IN ?", dbModules)
	}
	if operatorID > 0 {
		listQ = listQ.Where("ol.user_id = ?", operatorID)
	}
	if start != nil {
		listQ = listQ.Where("ol.created_at >= ?", *start)
	}
	if endExclusive != nil {
		listQ = listQ.Where("ol.created_at < ?", *endExclusive)
	}
	if err := listQ.Order("ol.created_at DESC").Offset(pq.Offset()).Limit(pq.PageSize).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}
