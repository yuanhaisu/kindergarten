package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/pkg/requestid"
)

// PlatformLogRepo 平台方操作留痕。与园所 operation_logs 分表，避免平台行为
// 混入园所审计视图。
type PlatformLogRepo struct {
	db *gorm.DB
}

func NewPlatformLogRepo(db *gorm.DB) *PlatformLogRepo {
	return &PlatformLogRepo{db: db}
}

// PlatformLogFilter 审计检索条件。From/To 为左闭右开的时间区间；
// KindergartenID 为 0 表示不按园所过滤（账号/套餐类记录的园所列本身就是 0）。
type PlatformLogFilter struct {
	From           *time.Time
	To             *time.Time
	Operator       string
	Module         string
	KindergartenID uint64
	Page           int
	PageSize       int
}

func (f *PlatformLogFilter) normalize() {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 {
		f.PageSize = 20
	}
	if f.PageSize > 100 {
		f.PageSize = 100
	}
}

// Create 写入一条平台审计。request_id 由中间件放进 context，这里统一落库，
// 保证任何调用点都不会漏掉链路标识。
func (r *PlatformLogRepo) Create(ctx context.Context, row *domain.PlatformOperationLog) error {
	return r.create(ctx, r.db.WithContext(ctx), row)
}

// CreateTx 与业务写入共用事务：返回错误即让整个业务操作回滚（design D6）。
func (r *PlatformLogRepo) CreateTx(ctx context.Context, tx *gorm.DB, row *domain.PlatformOperationLog) error {
	return r.create(ctx, tx.WithContext(ctx), row)
}

func (r *PlatformLogRepo) create(ctx context.Context, db *gorm.DB, row *domain.PlatformOperationLog) error {
	if row.RequestID == "" {
		row.RequestID = requestid.From(ctx)
	}
	return db.Create(row).Error
}

func (r *PlatformLogRepo) List(ctx context.Context, f PlatformLogFilter) ([]domain.PlatformOperationLog, int64, error) {
	f.normalize()
	q := r.db.WithContext(ctx).Model(&domain.PlatformOperationLog{})
	if f.From != nil {
		q = q.Where("created_at >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("created_at < ?", *f.To)
	}
	if f.Operator != "" {
		q = q.Where("operator = ?", f.Operator)
	}
	if f.Module != "" {
		q = q.Where("module = ?", f.Module)
	}
	if f.KindergartenID != 0 {
		q = q.Where("kindergarten_id = ?", f.KindergartenID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []domain.PlatformOperationLog
	err := q.Order("id DESC").Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize).Find(&list).Error
	return list, total, err
}
