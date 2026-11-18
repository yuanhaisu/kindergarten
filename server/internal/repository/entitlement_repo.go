package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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

// ActiveExpiryFilter 授权到期查询的共用条件（design D7）：
// 未删除、已生效（starts_at <= Now）且尚未到期（expires_at > Now）。
// 园所列表的 soonest_expire_at 与平台看板的 30/7 天窗口共用这一查询，避免两套口径。
type ActiveExpiryFilter struct {
	// KindergartenIDs 为空表示不限园所。
	KindergartenIDs []uint64
	Now             time.Time
	// Until 非空时只返回到期时间不晚于它的授权（看板的到期窗口）。
	Until *time.Time
}

// ListActiveExpiring 按 ActiveExpiryFilter 查询在期授权。
func (r *EntitlementRepo) ListActiveExpiring(ctx context.Context, f ActiveExpiryFilter) ([]domain.Entitlement, error) {
	q := r.db.WithContext(ctx).
		Where("starts_at <= ? AND expires_at > ?", f.Now, f.Now)
	if len(f.KindergartenIDs) > 0 {
		q = q.Where("kindergarten_id IN ?", f.KindergartenIDs)
	}
	if f.Until != nil {
		q = q.Where("expires_at <= ?", *f.Until)
	}
	var list []domain.Entitlement
	err := q.Find(&list).Error
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

// ApplyReversalByPaymentTx 把退款金额累计到被退收款产生的原始佣金行，
// 返回被冲回的原始行与本次实际冲回金额（正数）；该收款没有佣金行时返回 nil。
// 行锁保证并发退款不会丢失冲回金额。
func (r *CommissionRepo) ApplyReversalByPaymentTx(
	ctx context.Context, tx *gorm.DB, orderID, paymentID uint64, amount int64,
) (*domain.CommissionLedger, int64, error) {
	var c domain.CommissionLedger
	err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("order_id = ? AND payment_id = ? AND reversal_of IS NULL", orderID, paymentID).
		Order("id ASC").First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, err
	}
	// 本次退款覆盖了原收款的剩余全部金额时直接冲满剩余净额，
	// 避免多笔部分退款各自整数取整后留下尾差。
	var refundedTotal int64
	if err := tx.WithContext(ctx).Model(&domain.Payment{}).
		Where("refund_of_payment_id = ?", paymentID).
		Select("COALESCE(SUM(-amount_cents), 0)").Scan(&refundedTotal).Error; err != nil {
		return nil, 0, err
	}
	if refundedTotal >= c.BaseAmountCents {
		amount = c.AmountCents - c.ReversedAmountCents
	}
	reversed := c.ReversedAmountCents + amount
	cols := map[string]any{"reversed_amount_cents": reversed}
	if c.AmountCents-reversed <= 0 {
		cols["status"] = domain.CommissionStatusReversed
	}
	if err := tx.WithContext(ctx).Model(&domain.CommissionLedger{}).
		Where("id = ?", c.ID).Updates(cols).Error; err != nil {
		return nil, 0, err
	}
	c.ReversedAmountCents = reversed
	if c.AmountCents-reversed <= 0 {
		c.Status = domain.CommissionStatusReversed
	}
	return &c, amount, nil
}

// ListReversals 返回某条原始佣金行对应的冲回记录，按发生顺序排列。
func (r *CommissionRepo) ListReversals(ctx context.Context, commissionID uint64) ([]domain.CommissionLedger, error) {
	var list []domain.CommissionLedger
	err := r.db.WithContext(ctx).
		Where("reversal_of = ?", commissionID).
		Order("id ASC").Find(&list).Error
	return list, err
}

// PromoteEligible 把「到账已满 30 天且净额为正」的待结算佣金推进为可结算。
// 没有定时任务，所以在读取台账前惰性推进；幂等。净额归零的佣金状态为
// reversed，不会被此条件命中。
func (r *CommissionRepo) PromoteEligible(ctx context.Context, now time.Time) error {
	return r.db.WithContext(ctx).Model(&domain.CommissionLedger{}).
		Where("status = ? AND eligible_at <= ? AND reversal_of IS NULL AND amount_cents - reversed_amount_cents > 0",
			domain.CommissionStatusPending, now).
		Update("status", domain.CommissionStatusSettleable).Error
}

func (r *CommissionRepo) FindByID(ctx context.Context, id uint64) (*domain.CommissionLedger, error) {
	var c domain.CommissionLedger
	err := r.db.WithContext(ctx).First(&c, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &c, err
}

// DistributorSettleable 看板待办：某分销商当前可结算的佣金净额与笔数。
type DistributorSettleable struct {
	DistributorID uint64
	AmountCents   int64
	Count         int64
}

// SettleableByDistributor 按分销商聚合可结算佣金净额，口径与台账 summary 一致
// （只列原始佣金行，净额 = amount - reversed）。调用前必须先 PromoteEligible，
// 否则满 30 天但未被台账页触发的记录会漏算（design D7）。
func (r *CommissionRepo) SettleableByDistributor(ctx context.Context) ([]DistributorSettleable, error) {
	var rows []DistributorSettleable
	err := r.db.WithContext(ctx).Model(&domain.CommissionLedger{}).
		Select("distributor_id, COALESCE(SUM(amount_cents - reversed_amount_cents), 0) AS amount_cents, COUNT(*) AS count").
		Where("status = ? AND reversal_of IS NULL", domain.CommissionStatusSettleable).
		Group("distributor_id").
		Scan(&rows).Error
	return rows, err
}

// List 返回台账明细。只列原始佣金行：冲回行的金额已经累计进原始行的
// reversed_amount_cents，再列一次会重复扣减。
func (r *CommissionRepo) List(ctx context.Context, f CommissionFilter) ([]domain.CommissionLedger, int64, error) {
	f.normalize()
	q := r.db.WithContext(ctx).Model(&domain.CommissionLedger{}).Where("reversal_of IS NULL")
	q = applyCommissionFilter(q, f)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []domain.CommissionLedger
	err := q.Order("id DESC").Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize).Find(&list).Error
	return list, total, err
}

// Summary 按状态汇总净额，供台账页顶部卡片使用。
// pending / settleable / settled 为对应状态原始佣金行的净额合计；
// reversed 为期间累计冲回金额（正数），与明细的 reversed_amount_cents 一致。
func (r *CommissionRepo) Summary(ctx context.Context, f CommissionFilter) (map[string]int64, error) {
	q := r.db.WithContext(ctx).Model(&domain.CommissionLedger{}).Where("reversal_of IS NULL")
	q = applyCommissionFilter(q, f)
	type row struct {
		Status        string
		NetTotal      int64
		ReversedTotal int64
	}
	var rows []row
	if err := q.Select(
		"status, COALESCE(SUM(amount_cents - reversed_amount_cents),0) AS net_total, " +
			"COALESCE(SUM(reversed_amount_cents),0) AS reversed_total",
	).Group("status").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := map[string]int64{
		domain.CommissionStatusPending:    0,
		domain.CommissionStatusSettleable: 0,
		domain.CommissionStatusSettled:    0,
		domain.CommissionStatusReversed:   0,
	}
	var reversedTotal int64
	for _, r := range rows {
		out[r.Status] = r.NetTotal
		reversedTotal += r.ReversedTotal
	}
	// 已冲回卡片展示累计冲回金额，而不是 reversed 行的净额（后者恒为 0）。
	out[domain.CommissionStatusReversed] = reversedTotal
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
