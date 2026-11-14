package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// ErrOrderHasUnrefundedPayment 表示订单仍有未退款收款，资金未结清时不得取消。
var ErrOrderHasUnrefundedPayment = errors.New("ORDER_HAS_UNREFUNDED_PAYMENT")

// UnrefundedPaymentError 携带待退金额，供管理端直接展示给运营。
type UnrefundedPaymentError struct {
	AmountCents int64
}

func (e *UnrefundedPaymentError) Error() string { return ErrOrderHasUnrefundedPayment.Error() }

func (e *UnrefundedPaymentError) Unwrap() error { return ErrOrderHasUnrefundedPayment }

// OrderService 订单与收款。订单确认时冻结快照（ADR-0013），
// 收款状态由金额推导（design D7）。
type OrderService struct {
	db        *gorm.DB
	orders    *repository.OrderRepo
	payments  *repository.PaymentRepo
	plans     *repository.PlanRepo
	kgs       *repository.KindergartenRepo
	ents      *repository.EntitlementRepo
	dists     *repository.DistributorRepo
	comm      *repository.CommissionRepo
	logs      *repository.PlatformLogRepo
	entsCache *EntitlementCache
}

func NewOrderService(
	db *gorm.DB,
	orders *repository.OrderRepo,
	payments *repository.PaymentRepo,
	plans *repository.PlanRepo,
	kgs *repository.KindergartenRepo,
	ents *repository.EntitlementRepo,
	dists *repository.DistributorRepo,
	comm *repository.CommissionRepo,
	logs *repository.PlatformLogRepo,
	entsCache *EntitlementCache,
) *OrderService {
	return &OrderService{
		db: db, orders: orders, payments: payments, plans: plans, kgs: kgs,
		ents: ents, dists: dists, comm: comm, logs: logs, entsCache: entsCache,
	}
}

// PlanSnapshot 成交时冻结的套餐内容。
type PlanSnapshot struct {
	PlanCode     string   `json:"plan_code"`
	PlanName     string   `json:"plan_name"`
	ModuleCodes  []string `json:"module_codes"`
	PriceCents   int64    `json:"price_cents"`
	TermMonths   int      `json:"term_months"`
	Tier         int      `json:"tier"`
	StudentCount int64    `json:"student_count"`
	// TierMaxStudents 成交时该档位的幼儿数上限（0 = 无上限）。
	// 用于日后判断园所是否超档——超限只提示，不阻断。
	TierMaxStudents int `json:"tier_max_students"`
}

type CreateOrderRequest struct {
	KindergartenID uint64          `json:"kindergarten_id" binding:"required"`
	PlanID         uint64          `json:"plan_id" binding:"required"`
	DistributorID  *uint64         `json:"distributor_id"`
	InvoiceInfo    json.RawMessage `json:"invoice_info"`
}

func (s *OrderService) Create(ctx context.Context, req CreateOrderRequest, operator, ip string) (*domain.Order, error) {
	kg, err := s.kgs.FindByID(ctx, req.KindergartenID)
	if err != nil {
		return nil, err
	}
	if kg == nil {
		return nil, fmt.Errorf("KINDERGARTEN_NOT_FOUND")
	}
	plan, err := s.plans.FindByID(ctx, req.PlanID)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, fmt.Errorf("PLAN_NOT_FOUND")
	}
	if plan.Status != "active" {
		return nil, fmt.Errorf("PLAN_ARCHIVED")
	}
	modules, err := s.plans.ListModuleCodes(ctx, plan.ID)
	if err != nil {
		return nil, err
	}
	studentCount, err := s.kgs.CountStudents(ctx, kg.ID)
	if err != nil {
		return nil, err
	}
	tier, multiplier, tierMax, err := s.resolveTier(ctx, plan.ID, studentCount)
	if err != nil {
		return nil, err
	}

	var rate int
	if req.DistributorID != nil {
		d, err := s.dists.FindByID(ctx, *req.DistributorID)
		if err != nil {
			return nil, err
		}
		if d == nil {
			return nil, fmt.Errorf("DISTRIBUTOR_NOT_FOUND")
		}
		rate = d.CommissionRate
	}

	now := time.Now()
	snapshot, err := json.Marshal(PlanSnapshot{
		PlanCode: plan.Code, PlanName: plan.Name, ModuleCodes: modules,
		PriceCents: plan.PriceCents, TermMonths: plan.TermMonths,
		Tier: tier, StudentCount: studentCount, TierMaxStudents: tierMax,
	})
	if err != nil {
		return nil, err
	}
	// invoice_info 是 JSON 列，空串不合法；缺省写 "{}"。
	invoice := "{}"
	if len(req.InvoiceInfo) > 0 && string(req.InvoiceInfo) != "null" {
		invoice = string(req.InvoiceInfo)
	}
	order := &domain.Order{
		OrderNo:        newOrderNo(now),
		KindergartenID: kg.ID,
		PlanID:         plan.ID,
		PlanSnapshot:   string(snapshot),
		StudentTier:    tier,
		AmountDueCents: plan.PriceCents * int64(multiplier) / 100,
		Status:         domain.OrderStatusUnpaid,
		DistributorID:  req.DistributorID,
		CommissionRate: rate,
		InvoiceInfo:    invoice,
		StartsAt:       now,
		ExpiresAt:      now.AddDate(0, plan.TermMonths, 0),
		CreatedBy:      operator,
	}
	if err := s.orders.Create(ctx, order); err != nil {
		return nil, err
	}
	_ = s.logs.Create(ctx, &domain.PlatformOperationLog{
		Operator: operator, Module: "order", Action: "create", TargetType: "order", TargetID: order.ID,
		Detail: fmt.Sprintf("%s / %s", kg.Name, plan.Name), IP: ip,
	})
	return order, nil
}

// resolveTier 按在园幼儿数找档位。没有配置档位时按平价处理（1 / 100 / 无上限）；
// 配置了档位却落不到任何一档，说明是历史遗留的不完整配置，必须报错——
// 静默回落到第一档会把规模更大的园所按最低价成交。
func (s *OrderService) resolveTier(ctx context.Context, planID uint64, studentCount int64) (int, int, int, error) {
	tiers, err := s.plans.ListTiers(ctx, planID)
	if err != nil {
		return 0, 0, 0, err
	}
	if len(tiers) == 0 {
		return 1, 100, 0, nil
	}
	for _, t := range tiers {
		if studentCount >= int64(t.MinStudents) && (t.MaxStudents == 0 || studentCount <= int64(t.MaxStudents)) {
			mult := t.PriceMultiplier
			if mult <= 0 {
				mult = 100
			}
			return t.Tier, mult, t.MaxStudents, nil
		}
	}
	return 0, 0, 0, ErrTierCoverageUnresolved
}

func (s *OrderService) Confirm(ctx context.Context, id uint64, operator, ip string) (*domain.Order, []domain.Entitlement, error) {
	order, err := s.orders.FindByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if order == nil {
		return nil, nil, fmt.Errorf("ORDER_NOT_FOUND")
	}
	if order.ConfirmedAt != nil {
		return nil, nil, fmt.Errorf("ORDER_ALREADY_CONFIRMED")
	}
	if order.CancelledAt != nil {
		return nil, nil, fmt.Errorf("ORDER_CANCELLED")
	}
	now := time.Now()
	paid, err := s.payments.SumByOrder(ctx, order.ID)
	if err != nil {
		return nil, nil, err
	}
	status := deriveOrderStatus(order, paid, now)

	var ents []domain.Entitlement
	var snap PlanSnapshot
	if err := json.Unmarshal([]byte(order.PlanSnapshot), &snap); err != nil {
		return nil, nil, fmt.Errorf("invalid plan snapshot: %w", err)
	}
	for _, code := range snap.ModuleCodes {
		if ModuleCategory(code) != domain.ModuleCategoryPremium {
			continue
		}
		ents = append(ents, domain.Entitlement{
			KindergartenID: order.KindergartenID, ModuleCode: code,
			Source: domain.EntitlementSourceSubscription, OrderID: &order.ID,
			StartsAt: order.StartsAt, ExpiresAt: order.ExpiresAt,
		})
	}

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.orders.UpdateTx(ctx, tx, order.ID, map[string]any{
			"confirmed_at": now, "status": status,
		}); err != nil {
			return err
		}
		return s.ents.CreateBatchTx(ctx, tx, ents)
	}); err != nil {
		return nil, nil, err
	}
	order.ConfirmedAt = &now
	order.Status = status
	s.invalidate(ctx, order.KindergartenID)
	_ = s.logs.Create(ctx, &domain.PlatformOperationLog{
		Operator: operator, Module: "order", Action: "confirm", TargetType: "order", TargetID: order.ID,
		Detail: order.OrderNo, IP: ip,
	})
	return order, ents, nil
}

func (s *OrderService) Cancel(ctx context.Context, id uint64, reason, operator, ip string) (*domain.Order, error) {
	order, err := s.orders.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, fmt.Errorf("ORDER_NOT_FOUND")
	}
	if order.CancelledAt != nil {
		return nil, fmt.Errorf("ORDER_ALREADY_CANCELLED")
	}
	now := time.Now()
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 取消前先看已收净额（正数收款 − 负数退款）。钱未退清就取消，
		// 会留下「款收了、授权没了」的失配，因此必须拒绝并回传待退金额。
		netPaid, err := s.payments.SumByOrderTx(ctx, tx, order.ID)
		if err != nil {
			return err
		}
		if netPaid > 0 {
			return &UnrefundedPaymentError{AmountCents: netPaid}
		}
		if err := s.orders.UpdateTx(ctx, tx, order.ID, map[string]any{
			"cancelled_at": now, "cancel_reason": reason, "status": domain.OrderStatusCancelled,
		}); err != nil {
			return err
		}
		return s.ents.DeleteByOrderID(ctx, tx, order.ID)
	}); err != nil {
		return nil, err
	}
	order.CancelledAt = &now
	order.CancelReason = reason
	order.Status = domain.OrderStatusCancelled
	s.invalidate(ctx, order.KindergartenID)
	_ = s.logs.Create(ctx, &domain.PlatformOperationLog{
		Operator: operator, Module: "order", Action: "cancel", TargetType: "order", TargetID: order.ID,
		Detail: reason, IP: ip,
	})
	return order, nil
}

type PaymentRequest struct {
	AmountCents int64  `json:"amount_cents" binding:"required"`
	Method      string `json:"method"`
	ReceivedAt  string `json:"received_at"`
	VoucherURL  string `json:"voucher_url"`
	Remark      string `json:"remark"`
	// RefundOfPaymentID 负数收款必填，指明被退的原收款；佣金冲回据此精确定位。
	RefundOfPaymentID *uint64 `json:"refund_of_payment_id"`
}

type PaymentResult struct {
	PaymentID       uint64 `json:"payment_id"`
	OrderStatus     string `json:"order_status"`
	AmountPaidCents int64  `json:"amount_paid_cents"`
}

func (s *OrderService) AddPayment(ctx context.Context, orderID uint64, req PaymentRequest, operator, ip string) (*PaymentResult, error) {
	order, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, fmt.Errorf("ORDER_NOT_FOUND")
	}
	if order.CancelledAt != nil {
		return nil, fmt.Errorf("ORDER_CANCELLED")
	}
	if req.AmountCents == 0 {
		return nil, fmt.Errorf("INVALID_AMOUNT")
	}
	if req.AmountCents < 0 {
		if req.RefundOfPaymentID == nil || *req.RefundOfPaymentID == 0 {
			return nil, fmt.Errorf("REFUND_PAYMENT_REQUIRED")
		}
	} else if req.RefundOfPaymentID != nil && *req.RefundOfPaymentID != 0 {
		return nil, fmt.Errorf("REFUND_PAYMENT_NOT_ALLOWED")
	}
	receivedAt := time.Now()
	if req.ReceivedAt != "" {
		parsed, err := time.Parse(time.RFC3339, req.ReceivedAt)
		if err != nil {
			return nil, fmt.Errorf("INVALID_RECEIVED_AT")
		}
		receivedAt = parsed
	}
	method := req.Method
	if method == "" {
		method = domain.PaymentMethodBankTransfer
	}
	payment := &domain.Payment{
		OrderID: order.ID, KindergartenID: order.KindergartenID, AmountCents: req.AmountCents,
		RefundOfPaymentID: req.RefundOfPaymentID,
		Method:            method, ReceivedAt: receivedAt, VoucherURL: req.VoucherURL,
		Remark: req.Remark, RecordedBy: operator,
	}

	var paid int64
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.validateRefundTarget(ctx, tx, order, payment); err != nil {
			return err
		}
		if err := s.payments.CreateTx(ctx, tx, payment); err != nil {
			return err
		}
		sum, err := s.payments.SumByOrderTx(ctx, tx, order.ID)
		if err != nil {
			return err
		}
		paid = sum
		status := deriveOrderStatus(order, sum, time.Now())
		if err := s.orders.UpdateTx(ctx, tx, order.ID, map[string]any{"status": status}); err != nil {
			return err
		}
		order.Status = status
		return s.writeCommission(ctx, tx, order, payment)
	}); err != nil {
		return nil, err
	}
	s.invalidate(ctx, order.KindergartenID)
	_ = s.logs.Create(ctx, &domain.PlatformOperationLog{
		Operator: operator, Module: "payment", Action: "record", TargetType: "order", TargetID: order.ID,
		Detail: fmt.Sprintf("%d 分 / %s", req.AmountCents, method), IP: ip,
	})
	return &PaymentResult{PaymentID: payment.ID, OrderStatus: order.Status, AmountPaidCents: paid}, nil
}

// validateRefundTarget 退款必须指向本订单内一笔可退的正数收款，且累计退款不得超过该笔收款。
func (s *OrderService) validateRefundTarget(ctx context.Context, tx *gorm.DB, order *domain.Order, p *domain.Payment) error {
	if p.RefundOfPaymentID == nil {
		return nil
	}
	target, err := s.payments.FindByIDForUpdateTx(ctx, tx, *p.RefundOfPaymentID)
	if err != nil {
		return err
	}
	if target == nil || target.OrderID != order.ID {
		return fmt.Errorf("REFUND_PAYMENT_NOT_FOUND")
	}
	if target.AmountCents <= 0 {
		return fmt.Errorf("REFUND_PAYMENT_INVALID")
	}
	refunded, err := s.payments.SumRefundsOfTx(ctx, tx, target.ID)
	if err != nil {
		return err
	}
	if -p.AmountCents > target.AmountCents-refunded {
		return fmt.Errorf("REFUND_EXCEEDS_PAYMENT")
	}
	return nil
}

// writeCommission 正数收款生成原始佣金行；负数收款冲回被退收款对应的佣金（ADR-0016）。
// 冲回金额累计进原始行的 ReversedAmountCents，净额归零后状态置为 reversed，
// 不再被 PromoteEligible 推进为可结算。
func (s *OrderService) writeCommission(ctx context.Context, tx *gorm.DB, order *domain.Order, p *domain.Payment) error {
	if order.DistributorID == nil || order.CommissionRate <= 0 {
		return nil
	}
	eligibleAt := p.ReceivedAt.AddDate(0, 0, commissionEligibleDays)
	if p.AmountCents > 0 {
		amount := p.AmountCents * int64(order.CommissionRate) / 100
		status := deriveCommissionStatus(eligibleAt, time.Now())
		return s.comm.CreateBatch(ctx, tx, []domain.CommissionLedger{{
			OrderID: order.ID, PaymentID: p.ID, DistributorID: *order.DistributorID,
			KindergartenID: order.KindergartenID, BaseAmountCents: p.AmountCents,
			Rate: order.CommissionRate, AmountCents: amount, Status: status, EligibleAt: eligibleAt,
		}})
	}
	// 退款：只冲回被退的那笔收款产生的佣金，而不是订单下最近一条。
	if p.RefundOfPaymentID == nil {
		return fmt.Errorf("REFUND_PAYMENT_REQUIRED")
	}
	reversalAmount := p.AmountCents * int64(order.CommissionRate) / 100 // 负数
	original, applied, err := s.comm.ApplyReversalByPaymentTx(
		ctx, tx, order.ID, *p.RefundOfPaymentID, -reversalAmount,
	)
	if err != nil {
		return err
	}
	if original == nil {
		// 该笔收款没有佣金行（无分销商/比例为 0 的历史数据），退款照常入账。
		return nil
	}
	return s.comm.CreateBatch(ctx, tx, []domain.CommissionLedger{{
		OrderID: order.ID, PaymentID: p.ID, DistributorID: *order.DistributorID,
		KindergartenID: order.KindergartenID, BaseAmountCents: -p.AmountCents,
		Rate: order.CommissionRate, AmountCents: -applied,
		Status: domain.CommissionStatusReversed, EligibleAt: eligibleAt, ReversalOf: &original.ID,
	}})
}

func deriveCommissionStatus(eligibleAt, now time.Time) string {
	if !now.Before(eligibleAt) {
		return domain.CommissionStatusSettleable
	}
	return domain.CommissionStatusPending
}

func deriveOrderStatus(order *domain.Order, paid int64, now time.Time) string {
	if order.CancelledAt != nil {
		return domain.OrderStatusCancelled
	}
	if order.AmountDueCents > 0 && paid >= order.AmountDueCents {
		return domain.OrderStatusPaid
	}
	if now.After(order.ExpiresAt) {
		return domain.OrderStatusExpired
	}
	if paid > 0 {
		return domain.OrderStatusPartial
	}
	return domain.OrderStatusUnpaid
}

func (s *OrderService) invalidate(ctx context.Context, kgID uint64) {
	if s.entsCache != nil {
		_ = s.entsCache.Invalidate(ctx, kgID)
	}
}

func newOrderNo(now time.Time) string {
	return fmt.Sprintf("ORD%s%03d", now.Format("20060102150405"), rand.Intn(1000))
}

// OrderRow 平台订单列表行。
type OrderRow struct {
	ID               uint64    `json:"id"`
	OrderNo          string    `json:"order_no"`
	KindergartenID   uint64    `json:"kindergarten_id"`
	KindergartenName string    `json:"kindergarten_name"`
	PlanName         string    `json:"plan_name"`
	StudentTier      int       `json:"student_tier"`
	AmountDueCents   int64     `json:"amount_due_cents"`
	AmountPaidCents  int64     `json:"amount_paid_cents"`
	Status           string    `json:"status"`
	StartsAt         time.Time `json:"starts_at"`
	ExpiresAt        time.Time `json:"expires_at"`
	DistributorName  string    `json:"distributor_name"`
	CreatedAt        time.Time `json:"created_at"`
}

func (s *OrderService) List(ctx context.Context, f repository.OrderFilter) ([]OrderRow, int64, error) {
	orders, total, err := s.orders.List(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	kgIDs := make([]uint64, 0, len(orders))
	distIDs := make([]uint64, 0, len(orders))
	for _, o := range orders {
		kgIDs = append(kgIDs, o.KindergartenID)
		if o.DistributorID != nil {
			distIDs = append(distIDs, *o.DistributorID)
		}
	}
	kgMap := make(map[uint64]string)
	if kgs, err := s.kgs.FindByIDs(ctx, uniqueIDs(kgIDs)); err == nil {
		for _, k := range kgs {
			kgMap[k.ID] = k.Name
		}
	}
	distMap := make(map[uint64]string)
	if dists, err := s.dists.FindByIDs(ctx, uniqueIDs(distIDs)); err == nil {
		for _, d := range dists {
			distMap[d.ID] = d.Name
		}
	}
	rows := make([]OrderRow, 0, len(orders))
	for _, o := range orders {
		paid, _ := s.payments.SumByOrder(ctx, o.ID)
		var snap PlanSnapshot
		_ = json.Unmarshal([]byte(o.PlanSnapshot), &snap)
		var distName string
		if o.DistributorID != nil {
			distName = distMap[*o.DistributorID]
		}
		rows = append(rows, OrderRow{
			ID: o.ID, OrderNo: o.OrderNo, KindergartenID: o.KindergartenID,
			KindergartenName: kgMap[o.KindergartenID], PlanName: snap.PlanName,
			StudentTier: o.StudentTier, AmountDueCents: o.AmountDueCents, AmountPaidCents: paid,
			Status: o.Status, StartsAt: o.StartsAt, ExpiresAt: o.ExpiresAt,
			DistributorName: distName, CreatedAt: o.CreatedAt,
		})
	}
	return rows, total, nil
}

type PaymentRow struct {
	ID                uint64    `json:"id"`
	AmountCents       int64     `json:"amount_cents"`
	Method            string    `json:"method"`
	ReceivedAt        time.Time `json:"received_at"`
	VoucherURL        string    `json:"voucher_url"`
	Remark            string    `json:"remark"`
	RecordedBy        string    `json:"recorded_by"`
	RefundOfPaymentID *uint64   `json:"refund_of_payment_id"`
}

type OrderDetail struct {
	ID               uint64          `json:"id"`
	OrderNo          string          `json:"order_no"`
	KindergartenID   uint64          `json:"kindergarten_id"`
	KindergartenName string          `json:"kindergarten_name"`
	Status           string          `json:"status"`
	AmountDueCents   int64           `json:"amount_due_cents"`
	AmountPaidCents  int64           `json:"amount_paid_cents"`
	StartsAt         time.Time       `json:"starts_at"`
	ExpiresAt        time.Time       `json:"expires_at"`
	PlanSnapshot     PlanSnapshot    `json:"plan_snapshot"`
	Distributor      *DistributorRef `json:"distributor"`
	InvoiceInfo      json.RawMessage `json:"invoice_info"`
	Payments         []PaymentRow    `json:"payments"`
}

type DistributorRef struct {
	ID             uint64 `json:"id"`
	Name           string `json:"name"`
	CommissionRate int    `json:"commission_rate"`
}

func (s *OrderService) Detail(ctx context.Context, id uint64) (*OrderDetail, error) {
	o, err := s.orders.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, fmt.Errorf("ORDER_NOT_FOUND")
	}
	paid, err := s.payments.SumByOrder(ctx, o.ID)
	if err != nil {
		return nil, err
	}
	var snap PlanSnapshot
	_ = json.Unmarshal([]byte(o.PlanSnapshot), &snap)
	var kgName string
	if kg, err := s.kgs.FindByID(ctx, o.KindergartenID); err == nil && kg != nil {
		kgName = kg.Name
	}
	var distRef *DistributorRef
	if o.DistributorID != nil {
		if d, err := s.dists.FindByID(ctx, *o.DistributorID); err == nil && d != nil {
			distRef = &DistributorRef{ID: d.ID, Name: d.Name, CommissionRate: o.CommissionRate}
		}
	}
	payments, err := s.payments.ListByOrder(ctx, o.ID)
	if err != nil {
		return nil, err
	}
	paymentRows := make([]PaymentRow, 0, len(payments))
	for _, p := range payments {
		paymentRows = append(paymentRows, PaymentRow{
			ID: p.ID, AmountCents: p.AmountCents, Method: p.Method,
			ReceivedAt: p.ReceivedAt, VoucherURL: p.VoucherURL, Remark: p.Remark, RecordedBy: p.RecordedBy,
			RefundOfPaymentID: p.RefundOfPaymentID,
		})
	}
	detail := &OrderDetail{
		ID: o.ID, OrderNo: o.OrderNo, KindergartenID: o.KindergartenID, KindergartenName: kgName,
		Status: o.Status, AmountDueCents: o.AmountDueCents, AmountPaidCents: paid,
		StartsAt: o.StartsAt, ExpiresAt: o.ExpiresAt, PlanSnapshot: snap,
		Distributor: distRef, Payments: paymentRows,
	}
	if o.InvoiceInfo != "" && o.InvoiceInfo != "{}" {
		detail.InvoiceInfo = json.RawMessage(o.InvoiceInfo)
	}
	return detail, nil
}

func (s *OrderService) ListPayments(ctx context.Context, orderID uint64) ([]PaymentRow, error) {
	payments, err := s.payments.ListByOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	out := make([]PaymentRow, 0, len(payments))
	for _, p := range payments {
		out = append(out, PaymentRow{
			ID: p.ID, AmountCents: p.AmountCents, Method: p.Method,
			ReceivedAt: p.ReceivedAt, VoucherURL: p.VoucherURL, Remark: p.Remark, RecordedBy: p.RecordedBy,
			RefundOfPaymentID: p.RefundOfPaymentID,
		})
	}
	return out, nil
}
