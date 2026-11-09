package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

// EntitlementRepo 园所级功能授权。所有查询都必须显式给出 kindergarten_id，
// 不存在跨园授权查询（design D9）。
type EntitlementRepo struct {
	db *gorm.DB
}

func NewEntitlementRepo(db *gorm.DB) *EntitlementRepo {
	return &EntitlementRepo{db: db}
}

func (r *EntitlementRepo) CreateBatch(ctx context.Context, rows []domain.Entitlement) error {
	if len(rows) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&rows).Error
}

func (r *EntitlementRepo) CreateBatchTx(ctx context.Context, tx *gorm.DB, rows []domain.Entitlement) error {
	if len(rows) == 0 {
		return nil
	}
	return tx.WithContext(ctx).Create(&rows).Error
}

func (r *EntitlementRepo) ListByKindergarten(ctx context.Context, kindergartenID uint64) ([]domain.Entitlement, error) {
	var list []domain.Entitlement
	err := r.db.WithContext(ctx).
		Where("kindergarten_id = ?", kindergartenID).
		Order("module_code ASC, expires_at DESC").
		Find(&list).Error
	return list, err
}

// ListActiveByKindergarten 返回在指定时刻仍在有效期内的授权行。
func (r *EntitlementRepo) ListActiveByKindergarten(ctx context.Context, kindergartenID uint64, at time.Time) ([]domain.Entitlement, error) {
	var list []domain.Entitlement
	err := r.db.WithContext(ctx).
		Where("kindergarten_id = ?", kindergartenID).
		Where("starts_at <= ? AND expires_at > ?", at, at).
		Find(&list).Error
	return list, err
}

func (r *EntitlementRepo) ListByOrderID(ctx context.Context, orderID uint64) ([]domain.Entitlement, error) {
	var list []domain.Entitlement
	err := r.db.WithContext(ctx).Where("order_id = ?", orderID).Find(&list).Error
	return list, err
}

// DeleteByOrderID 收回订单产生的授权（订单取消时调用）。
func (r *EntitlementRepo) DeleteByOrderID(ctx context.Context, tx *gorm.DB, orderID uint64) error {
	return tx.WithContext(ctx).Where("order_id = ?", orderID).Delete(&domain.Entitlement{}).Error
}

// CommissionFilter 佣金台账查询。
type CommissionFilter struct {
	Month         string // YYYY-MM，按台账生成时间过滤
	DistributorID uint64
	Status        string
	Page          int
	PageSize      int
}

func (f *CommissionFilter) normalize() {
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

type CommissionRepo struct {
	db *gorm.DB
}

func NewCommissionRepo(db *gorm.DB) *CommissionRepo {
	return &CommissionRepo{db: db}
}

func (r *CommissionRepo) CreateBatch(ctx context.Context, tx *gorm.DB, rows []domain.CommissionLedger) error {
	if len(rows) == 0 {
		return nil
	}
	return tx.WithContext(ctx).Create(&rows).Error
}

func (r *CommissionRepo) FindByID(ctx context.Context, id uint64) (*domain.CommissionLedger, error) {
	var c domain.CommissionLedger
	err := r.db.WithContext(ctx).First(&c, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &c, err
}

func (r *CommissionRepo) List(ctx context.Context, f CommissionFilter) ([]domain.CommissionLedger, int64, error) {
	f.normalize()
	q := r.db.WithContext(ctx).Model(&domain.CommissionLedger{})
	q = applyCommissionFilter(q, f)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []domain.CommissionLedger
	err := q.Order("id DESC").Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize).Find(&list).Error
	return list, total, err
}

// Summary 按状态汇总金额，供台账页顶部卡片使用。
func (r *CommissionRepo) Summary(ctx context.Context, f CommissionFilter) (map[string]int64, error) {
	q := r.db.WithContext(ctx).Model(&domain.CommissionLedger{})
	q = applyCommissionFilter(q, f)
	type row struct {
		Status string
		Total  int64
	}
	var rows []row
	if err := q.Select("status, COALESCE(SUM(amount_cents),0) AS total").Group("status").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := map[string]int64{
		domain.CommissionStatusPending:    0,
		domain.CommissionStatusSettleable: 0,
		domain.CommissionStatusSettled:    0,
		domain.CommissionStatusReversed:   0,
	}
	for _, r := range rows {
		out[r.Status] = r.Total
	}
	return out, nil
}

func applyCommissionFilter(q *gorm.DB, f CommissionFilter) *gorm.DB {
	if f.Month != "" {
		start, err := time.Parse("2006-01", f.Month)
		if err == nil {
			q = q.Where("created_at >= ? AND created_at < ?", start, start.AddDate(0, 1, 0))
		}
	}
	if f.DistributorID != 0 {
		q = q.Where("distributor_id = ?", f.DistributorID)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	return q
}
