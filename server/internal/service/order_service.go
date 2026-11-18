package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// ErrOrderHasUnrefundedPayment 表示订单仍有未退款收款，资金未结清时不得取消。
var ErrOrderHasUnrefundedPayment = errors.New("ORDER_HAS_UNREFUNDED_PAYMENT")

// 收款写入的可识别错误。金额类错误携带结构化数据，管理端可直接展示。
var (
	ErrPaymentExceedsAmountDue = errors.New("PAYMENT_EXCEEDS_AMOUNT_DUE")
	ErrRefundExceedsPaidAmount = errors.New("REFUND_EXCEEDS_PAID_AMOUNT")
	ErrReceivedAtInFuture      = errors.New("RECEIVED_AT_IN_FUTURE")
	ErrInvalidIdempotencyKey   = errors.New("INVALID_IDEMPOTENCY_KEY")
	// ErrInvalidPayDueAt 表示创建订单时显式传入的付款到期日不是 YYYY-MM-DD。
	ErrInvalidPayDueAt = errors.New("INVALID_PAY_DUE_AT")
	// ErrCreditReasonRequired 表示净收款为零的确认缺少信用开通理由。
	ErrCreditReasonRequired = errors.New("CREDIT_REASON_REQUIRED")
	// ErrCreditReasonTooLong 表示信用开通理由超出 credit_reason 列宽。
	ErrCreditReasonTooLong = errors.New("CREDIT_REASON_TOO_LONG")
)

// PaymentExceedsAmountDueError 携带超出金额与剩余应收，供管理端与园所核对。
type PaymentExceedsAmountDueError struct {
	ExcessCents    int64
	RemainingCents int64
}

func (e *PaymentExceedsAmountDueError) Error() string { return ErrPaymentExceedsAmountDue.Error() }

func (e *PaymentExceedsAmountDueError) Unwrap() error { return ErrPaymentExceedsAmountDue }

// RefundExceedsPaidAmountError 携带当前可退上限，避免运营反复试错。
type RefundExceedsPaidAmountError struct {
	AvailableCents int64
	RequestedCents int64
}

func (e *RefundExceedsPaidAmountError) Error() string { return ErrRefundExceedsPaidAmount.Error() }

func (e *RefundExceedsPaidAmountError) Unwrap() error { return ErrRefundExceedsPaidAmount }

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
	logs      PlatformAuditWriter
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
	logs PlatformAuditWriter,
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
	// PayDueAt 可选，YYYY-MM-DD；缺省时由 Order.BeforeCreate 取服务起始日。
	PayDueAt string `json:"pay_due_at"`
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
	payDueAt, err := parsePayDueAt(req.PayDueAt)
	if err != nil {
		return nil, err
	}
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
		PayDueAt:       payDueAt,
		CreatedBy:      operator,
	}
	if err := s.orders.Create(ctx, order); err != nil {
		return nil, err
	}
	// 订单创建不在强一致清单内：审计失败只记日志，不阻断成交录入。
	writeAuditBestEffort(ctx, s.logs, &domain.PlatformOperationLog{
		Operator: operator, Module: "order", Action: "create", TargetType: "order", TargetID: order.ID,
		KindergartenID: kg.ID, Detail: fmt.Sprintf("%s / %s", kg.Name, plan.Name), IP: ip,
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

func (s *OrderService) Confirm(ctx context.Context, id uint64, reason, operator, ip string) (*domain.Order, []domain.Entitlement, error) {
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

	// 零收款确认 = 信用开通：必须留理由，否则拒绝（design D2）。
	// 校验在任何写入之前完成，被拒绝的请求不留确认痕迹。
	reason = strings.TrimSpace(reason)
	creditGranted := paid == 0
	if creditGranted {
		if reason == "" {
			return nil, nil, ErrCreditReasonRequired
		}
		if utf8.RuneCountInString(reason) > 255 {
			return nil, nil, ErrCreditReasonTooLong
		}
	}

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

	fields := map[string]any{"confirmed_at": now, "status": status}
	auditDetail := order.OrderNo
	if creditGranted {
		fields["credit_granted"] = true
		fields["credit_reason"] = reason
		auditDetail = fmt.Sprintf("%s / 信用开通：%s", order.OrderNo, reason)
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.orders.UpdateTx(ctx, tx, order.ID, fields); err != nil {
			return err
		}
		if err := s.ents.CreateBatchTx(ctx, tx, ents); err != nil {
			return err
		}
		// 确认订单（含信用开通）属于强一致清单（design D6）：审计与授权同事务，
		// 审计失败即整体回滚，不允许「授权发了、痕迹没留」。
		return s.logs.CreateTx(ctx, tx, &domain.PlatformOperationLog{
			Operator: operator, Module: "order", Action: "confirm", TargetType: "order", TargetID: order.ID,
			KindergartenID: order.KindergartenID, Detail: auditDetail, IP: ip,
		})
	}); err != nil {
		return nil, nil, err
	}
	order.ConfirmedAt = &now
	order.Status = status
	if creditGranted {
		order.CreditGranted = true
		order.CreditReason = reason
	}
	s.invalidate(ctx, order.KindergartenID)
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
		if err := s.ents.DeleteByOrderID(ctx, tx, order.ID); err != nil {
			return err
		}
		// 取消订单属于强一致清单：审计失败时状态与授权回收一并回滚。
		return s.logs.CreateTx(ctx, tx, &domain.PlatformOperationLog{
			Operator: operator, Module: "order", Action: "cancel", TargetType: "order", TargetID: order.ID,
			KindergartenID: order.KindergartenID, Detail: reason, IP: ip,
		})
	}); err != nil {
		return nil, err
	}
	order.CancelledAt = &now
	order.CancelReason = reason
	order.Status = domain.OrderStatusCancelled
	s.invalidate(ctx, order.KindergartenID)
	return order, nil
}

type PaymentRequest struct {
	AmountCents int64  `json:"amount_cents" binding:"required"`
	Method      string `json:"method"`
	// ReceivedAt 只作账务展示与对账，不得晚于当前时间；佣金结算时钟用服务端登记时间。
	ReceivedAt string `json:"received_at"`
	VoucherURL string `json:"voucher_url"`
	Remark     string `json:"remark"`
	// RefundOfPaymentID 负数收款必填，指明被退的原收款；佣金冲回据此精确定位。
	RefundOfPaymentID *uint64 `json:"refund_of_payment_id"`
	// IdempotencyKey 由管理端收款弹窗生成，同一订单下重复提交返回首次结果。
	// 请求头 Idempotency-Key 优先于本字段；缺省时旧调用仍可写入，但没有幂等保护。
	IdempotencyKey string `json:"idempotency_key"`
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
	// 与数据库状态无关的输入校验先做，避免为必然失败的请求开启事务。
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
	idempotencyKey, err := normalizeIdempotencyKey(req.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	// recordedAt 是本次登记的服务端时间：既用于拒绝未来 received_at，
	// 也是佣金 30 天结算时钟的起点，回填历史到账日不会让佣金提前可结算。
	recordedAt := time.Now()
	receivedAt := recordedAt
	if req.ReceivedAt != "" {
		parsed, err := time.Parse(time.RFC3339, req.ReceivedAt)
		if err != nil {
			return nil, fmt.Errorf("INVALID_RECEIVED_AT")
		}
		if parsed.After(recordedAt) {
			return nil, ErrReceivedAtInFuture
		}
		receivedAt = parsed
	}
	method := req.Method
	if method == "" {
		method = domain.PaymentMethodBankTransfer
	}
	payment := &domain.Payment{
		OrderID: order.ID, KindergartenID: order.KindergartenID, AmountCents: req.AmountCents,
		IdempotencyKey: idempotencyKey, RefundOfPaymentID: req.RefundOfPaymentID,
		Method: method, ReceivedAt: receivedAt, VoucherURL: req.VoucherURL,
		Remark: req.Remark, RecordedBy: operator,
	}

	var result *PaymentResult
	created := false
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 锁订单行，使同一订单的收款/退款串行化：金额上限与幂等判定
		// 都建立在这道锁上，避免并发请求同时通过校验。
		locked, err := s.orders.FindByIDForUpdate(ctx, tx, order.ID)
		if err != nil {
			return err
		}
		if locked == nil {
			return fmt.Errorf("ORDER_NOT_FOUND")
		}
		// 幂等重放优先于状态检查：首次请求已落库时，重复请求原样返回首次结果。
		if idempotencyKey != nil {
			existing, err := s.payments.FindByOrderAndIdempotencyKeyTx(ctx, tx, order.ID, *idempotencyKey)
			if err != nil {
				return err
			}
			if existing != nil {
				paid, err := s.payments.SumByOrderTx(ctx, tx, order.ID)
				if err != nil {
					return err
				}
				result = &PaymentResult{
					PaymentID: existing.ID, OrderStatus: locked.Status, AmountPaidCents: paid,
				}
				return nil
			}
		}
		if locked.CancelledAt != nil {
			return fmt.Errorf("ORDER_CANCELLED")
		}
		if err := s.validateRefundTarget(ctx, tx, locked, payment); err != nil {
			return err
		}
		netPaid, err := s.payments.SumByOrderTx(ctx, tx, order.ID)
		if err != nil {
			return err
		}
		if err := validatePaymentAmount(locked, netPaid, payment.AmountCents); err != nil {
			return err
		}
		if err := s.payments.CreateTx(ctx, tx, payment); err != nil {
			return err
		}
		created = true
		sum, err := s.payments.SumByOrderTx(ctx, tx, order.ID)
		if err != nil {
			return err
		}
		status := deriveOrderStatus(locked, sum, time.Now())
		if err := s.orders.UpdateTx(ctx, tx, order.ID, map[string]any{"status": status}); err != nil {
			return err
		}
		locked.Status = status
		if err := s.writeCommission(ctx, tx, locked, payment, recordedAt); err != nil {
			return err
		}
		// 收款/退款属于强一致清单：审计失败时收款、状态与佣金一并回滚。
		// 幂等重放不写新审计（上面已提前返回）。
		if err := s.logs.CreateTx(ctx, tx, &domain.PlatformOperationLog{
			Operator: operator, Module: "payment", Action: "record", TargetType: "order", TargetID: order.ID,
			KindergartenID: order.KindergartenID,
			Detail:         fmt.Sprintf("%d 分 / %s", req.AmountCents, method), IP: ip,
		}); err != nil {
			return err
		}
		result = &PaymentResult{PaymentID: payment.ID, OrderStatus: status, AmountPaidCents: sum}
		return nil
	}); err != nil {
		// 唯一约束兜底：并发重复请求中较晚提交的一笔在插入时撞键，
		// 回读首次落库的收款并返回同一标识。
		if idempotencyKey != nil && repository.IsDuplicateKey(err) {
			existing, findErr := s.payments.FindByOrderAndIdempotencyKey(ctx, order.ID, *idempotencyKey)
			if findErr != nil {
				return nil, findErr
			}
			if existing != nil {
				paid, sumErr := s.payments.SumByOrder(ctx, order.ID)
				if sumErr != nil {
					return nil, sumErr
				}
				status := order.Status
				if fresh, err := s.orders.FindByID(ctx, order.ID); err == nil && fresh != nil {
					status = fresh.Status
				}
				return &PaymentResult{PaymentID: existing.ID, OrderStatus: status, AmountPaidCents: paid}, nil
			}
		}
		return nil, err
	}
	if created {
		s.invalidate(ctx, order.KindergartenID)
	}
	return result, nil
}

// normalizeIdempotencyKey 空串视为未提供（写 NULL，多个旧请求互不冲突）。
func normalizeIdempotencyKey(raw string) (*string, error) {
	key := strings.TrimSpace(raw)
	if key == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(key) > 64 {
		return nil, ErrInvalidIdempotencyKey
	}
	return &key, nil
}

// validatePaymentAmount 校验金额上限：正数收款不得超过应收，负数退款不得超过已收净额。
func validatePaymentAmount(order *domain.Order, netPaid, amount int64) error {
	if amount > 0 {
		remaining := order.AmountDueCents - netPaid
		if remaining < 0 {
			remaining = 0
		}
		if amount > remaining {
			return &PaymentExceedsAmountDueError{ExcessCents: amount - remaining, RemainingCents: remaining}
		}
		return nil
	}
	available := netPaid
	if available < 0 {
		available = 0
	}
	if -amount > available {
		return &RefundExceedsPaidAmountError{AvailableCents: available, RequestedCents: -amount}
	}
	return nil
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
// recordedAt 是服务端登记时间：结算时钟以它为准，received_at 只作展示与对账。
func (s *OrderService) writeCommission(ctx context.Context, tx *gorm.DB, order *domain.Order, p *domain.Payment, recordedAt time.Time) error {
	if order.DistributorID == nil || order.CommissionRate <= 0 {
		return nil
	}
	eligibleAt := recordedAt.AddDate(0, 0, commissionEligibleDays)
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

// deriveOrderStatus 是付款状态的唯一出口：逾期只看付款到期日，
// 服务期结束日不再参与。PayDueAt 为零值时回退到服务起始日，
// 让绕过 BeforeCreate 的历史数据也不会被判成过期（design D1）。
func deriveOrderStatus(order *domain.Order, paid int64, now time.Time) string {
	if order.CancelledAt != nil {
		return domain.OrderStatusCancelled
	}
	if order.AmountDueCents > 0 && paid >= order.AmountDueCents {
		return domain.OrderStatusPaid
	}
	dueAt := order.PayDueAt
	if dueAt.IsZero() {
		dueAt = order.StartsAt
	}
	if now.After(dueAt) {
		return domain.OrderStatusExpired
	}
	if paid > 0 {
		return domain.OrderStatusPartial
	}
	return domain.OrderStatusUnpaid
}

// parsePayDueAt 解析创建订单时显式指定的付款到期日（YYYY-MM-DD）。
// 空串表示未指定，返回零值由 BeforeCreate 兜底为服务起始日。
func parsePayDueAt(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, nil
	}
	t, err := time.ParseInLocation("2006-01-02", raw, time.Local)
	if err != nil {
		return time.Time{}, ErrInvalidPayDueAt
	}
	return t, nil
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
	PayDueAt         time.Time `json:"pay_due_at"`
	// CreditGranted 标记零收款确认的订单；理由只在详情返回。
	CreditGranted   bool      `json:"credit_granted"`
	DistributorName string    `json:"distributor_name"`
	CreatedAt       time.Time `json:"created_at"`
}

func (s *OrderService) List(ctx context.Context, f repository.OrderFilter) ([]OrderRow, int64, error) {
	orders, total, err := s.orders.List(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	now := time.Now()
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
		var snap PlanSnapshot
		_ = json.Unmarshal([]byte(o.PlanSnapshot), &snap)
		var distName string
		if o.DistributorID != nil {
			distName = distMap[*o.DistributorID]
		}
		// 已收金额来自列表的单次聚合；展示状态统一读时派生，
		// 不直接返回存储的 status（design D3）。
		rows = append(rows, OrderRow{
			ID: o.ID, OrderNo: o.OrderNo, KindergartenID: o.KindergartenID,
			KindergartenName: kgMap[o.KindergartenID], PlanName: snap.PlanName,
			StudentTier: o.StudentTier, AmountDueCents: o.AmountDueCents, AmountPaidCents: o.PaidCents,
			Status: deriveOrderStatus(&o.Order, o.PaidCents, now), StartsAt: o.StartsAt,
			ExpiresAt: o.ExpiresAt, PayDueAt: o.PayDueAt, CreditGranted: o.CreditGranted,
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
	PayDueAt         time.Time       `json:"pay_due_at"`
	CreditGranted    bool            `json:"credit_granted"`
	CreditReason     string          `json:"credit_reason"`
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
		// 详情与列表一致：状态按付款到期日读时派生。
		Status: deriveOrderStatus(o, paid, time.Now()), AmountDueCents: o.AmountDueCents, AmountPaidCents: paid,
		StartsAt: o.StartsAt, ExpiresAt: o.ExpiresAt, PayDueAt: o.PayDueAt,
		CreditGranted: o.CreditGranted, CreditReason: o.CreditReason, PlanSnapshot: snap,
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
