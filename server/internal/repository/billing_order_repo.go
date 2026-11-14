package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"kindergarten-server/internal/domain"
)

// OrderFilter 平台侧订单查询。平台查询必须显式给出过滤条件，
// 不允许复用按 kindergarten_id 自动过滤的园所仓储（design D9）。
type OrderFilter struct {
	KindergartenID uint64
	Status         string
	DistributorID  uint64
	Page           int
	PageSize       int
}

func (f *OrderFilter) normalize() {
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

type OrderRepo struct {
	db *gorm.DB
}

func NewOrderRepo(db *gorm.DB) *OrderRepo {
	return &OrderRepo{db: db}
}

func (r *OrderRepo) Create(ctx context.Context, o *domain.Order) error {
	return r.db.WithContext(ctx).Create(o).Error
}

func (r *OrderRepo) FindByID(ctx context.Context, id uint64) (*domain.Order, error) {
	var o domain.Order
	err := r.db.WithContext(ctx).First(&o, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &o, err
}

func (r *OrderRepo) FindByIDForUpdate(ctx context.Context, tx *gorm.DB, id uint64) (*domain.Order, error) {
	var o domain.Order
	err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&o, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &o, err
}

func (r *OrderRepo) List(ctx context.Context, f OrderFilter) ([]domain.Order, int64, error) {
	f.normalize()
	q := r.db.WithContext(ctx).Model(&domain.Order{})
	if f.KindergartenID != 0 {
		q = q.Where("kindergarten_id = ?", f.KindergartenID)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.DistributorID != 0 {
		q = q.Where("distributor_id = ?", f.DistributorID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []domain.Order
	err := q.Order("id DESC").Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize).Find(&list).Error
	return list, total, err
}

func (r *OrderRepo) Update(ctx context.Context, id uint64, cols map[string]any) error {
	if len(cols) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Model(&domain.Order{}).Where("id = ?", id).Updates(cols).Error
}

func (r *OrderRepo) UpdateTx(ctx context.Context, tx *gorm.DB, id uint64, cols map[string]any) error {
	if len(cols) == 0 {
		return nil
	}
	return tx.WithContext(ctx).Model(&domain.Order{}).Where("id = ?", id).Updates(cols).Error
}

// FindLatestActiveByKindergarten 返回园所最近一张已确认且未取消的订单，
// 用于园所列表展示当前套餐名。
func (r *OrderRepo) FindLatestActiveByKindergarten(ctx context.Context, kindergartenID uint64) (*domain.Order, error) {
	var o domain.Order
	err := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND status <> ? AND confirmed_at IS NOT NULL", kindergartenID, domain.OrderStatusCancelled).
		Order("id DESC").
		First(&o).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &o, err
}

// ListDueForExpiry 返回已过到期日但仍未收清、未取消的订单，供状态推进使用。
func (r *OrderRepo) ListDueForExpiry(ctx context.Context, before any) ([]domain.Order, error) {
	var list []domain.Order
	err := r.db.WithContext(ctx).
		Where("expires_at < ?", before).
		Where("status IN ?", []string{domain.OrderStatusUnpaid, domain.OrderStatusPartial}).
		Find(&list).Error
	return list, err
}

// PaymentRepo 收款登记。一张订单可有多笔，退款为负数。
type PaymentRepo struct {
	db *gorm.DB
}

func NewPaymentRepo(db *gorm.DB) *PaymentRepo {
	return &PaymentRepo{db: db}
}

func (r *PaymentRepo) Create(ctx context.Context, p *domain.Payment) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *PaymentRepo) CreateTx(ctx context.Context, tx *gorm.DB, p *domain.Payment) error {
	return tx.WithContext(ctx).Create(p).Error
}

// FindByIDForUpdateTx 在事务内锁定并读取单笔收款。退款校验先锁被退收款行，
// 并发退款会在此串行化，避免两笔退款同时通过可退金额校验。
func (r *PaymentRepo) FindByIDForUpdateTx(ctx context.Context, tx *gorm.DB, id uint64) (*domain.Payment, error) {
	var p domain.Payment
	err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&p, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

// SumRefundsOfTx 返回某笔原收款已被退款的金额（正数合计）。
func (r *PaymentRepo) SumRefundsOfTx(ctx context.Context, tx *gorm.DB, paymentID uint64) (int64, error) {
	var sum int64
	err := tx.WithContext(ctx).Model(&domain.Payment{}).
		Where("refund_of_payment_id = ?", paymentID).
		Select("COALESCE(SUM(-amount_cents), 0)").
		Scan(&sum).Error
	return sum, err
}

func (r *PaymentRepo) ListByOrder(ctx context.Context, orderID uint64) ([]domain.Payment, error) {
	var list []domain.Payment
	err := r.db.WithContext(ctx).Where("order_id = ?", orderID).Order("id ASC").Find(&list).Error
	return list, err
}

// SumByOrder 已收金额 = 正数收款之和 + 负数退款之和。
func (r *PaymentRepo) SumByOrder(ctx context.Context, orderID uint64) (int64, error) {
	return r.sumByOrderTx(ctx, r.db, orderID)
}

func (r *PaymentRepo) SumByOrderTx(ctx context.Context, tx *gorm.DB, orderID uint64) (int64, error) {
	return r.sumByOrderTx(ctx, tx, orderID)
}

func (r *PaymentRepo) sumByOrderTx(ctx context.Context, db *gorm.DB, orderID uint64) (int64, error) {
	var sum int64
	err := db.WithContext(ctx).Model(&domain.Payment{}).
		Where("order_id = ?", orderID).
		Select("COALESCE(SUM(amount_cents), 0)").
		Scan(&sum).Error
	return sum, err
}
