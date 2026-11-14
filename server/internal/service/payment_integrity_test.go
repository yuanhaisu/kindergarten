package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// 本切片只清理 S1Bpay- 前缀的数据；套餐编码用 pay4_ 前缀，
// 避免命中其他切片按 LIKE 's1b_%' 执行的清理。
const paymentIntegrityPlanCode = "pay4_integrity"

func cleanupPaymentIntegrity(t *testing.T, db *gorm.DB) {
	t.Helper()
	del := func(sql string, args ...any) {
		if err := db.Exec(sql, args...).Error; err != nil {
			t.Logf("cleanup %q: %v", sql, err)
		}
	}
	const kgLike = "S1Bpay-%"
	del("DELETE FROM commission_ledger WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE ?)", kgLike)
	del("DELETE FROM payments WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE ?)", kgLike)
	del("DELETE FROM entitlements WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE ?)", kgLike)
	del("DELETE FROM orders WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE ?)", kgLike)
	del("DELETE FROM kindergartens WHERE name LIKE ?", kgLike)
	// 套餐编码显式写全，不用 LIKE：下划线是通配符。
	del("DELETE FROM plan_modules WHERE plan_id IN (SELECT id FROM plans WHERE code = ?)", paymentIntegrityPlanCode)
	del("DELETE FROM plan_tiers WHERE plan_id IN (SELECT id FROM plans WHERE code = ?)", paymentIntegrityPlanCode)
	del("DELETE FROM plans WHERE code = ?", paymentIntegrityPlanCode)
	del("DELETE FROM distributors WHERE name LIKE 'S1Bpay-分销%'")
}

type paymentIntegrityFixture struct {
	kg       *domain.Kindergarten
	dist     *domain.Distributor
	order    *domain.Order
	orderSvc *OrderService
}

// newPaymentIntegrityFixture 建一套独立的园所 / 套餐 / 分销商 / 已确认订单。
func newPaymentIntegrityFixture(t *testing.T, db *gorm.DB, suffix string) *paymentIntegrityFixture {
	t.Helper()
	ctx := context.Background()
	planSvc, orderSvc, _ := newBillingFlowServices(t, db)

	kg := &domain.Kindergarten{Name: "S1Bpay-收款测试园" + suffix, Status: domain.KindergartenStatusActive}
	require.NoError(t, repository.NewKindergartenRepo(db).Create(ctx, kg))

	plan, err := planSvc.Create(ctx, PlanRequest{
		Code: paymentIntegrityPlanCode, Name: "S1Bpay-收款套餐", Tier: 1, PriceCents: 100000, TermMonths: 6,
		ModuleCodes: []string{"base"},
	}, "pay4_ops", "127.0.0.1")
	require.NoError(t, err)

	dist := &domain.Distributor{Name: "S1Bpay-分销" + suffix, CommissionRate: 10, Status: 1}
	require.NoError(t, repository.NewDistributorRepo(db).Create(ctx, dist))

	order, err := orderSvc.Create(ctx, CreateOrderRequest{
		KindergartenID: kg.ID, PlanID: plan.ID, DistributorID: &dist.ID,
	}, "pay4_ops", "127.0.0.1")
	require.NoError(t, err)
	_, _, err = orderSvc.Confirm(ctx, order.ID, "pay4_ops", "127.0.0.1")
	require.NoError(t, err)

	return &paymentIntegrityFixture{kg: kg, dist: dist, order: order, orderSvc: orderSvc}
}

func beginPaymentIntegrityTest(t *testing.T, db *gorm.DB) {
	t.Helper()
	cleanupPaymentIntegrity(t, db)
	t.Cleanup(func() { cleanupPaymentIntegrity(t, db) })
}

func countPayments(t *testing.T, db *gorm.DB, orderID uint64) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(&domain.Payment{}).Where("order_id = ?", orderID).Count(&n).Error)
	return n
}

func countCommissions(t *testing.T, db *gorm.DB, orderID uint64) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(&domain.CommissionLedger{}).Where("order_id = ?", orderID).Count(&n).Error)
	return n
}

// S1Bpay-1：相同幂等键重复提交只产生一条收款与一份佣金，第二次返回首次收款标识。
func TestPaymentIntegrity_DuplicateIdempotencyKeyReturnsFirstPayment(t *testing.T) {
	db := openPlatformTestDB(t)
	beginPaymentIntegrityTest(t, db)
	f := newPaymentIntegrityFixture(t, db, "A")
	ctx := context.Background()

	first, err := f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{
		AmountCents: 50000, IdempotencyKey: "pay4-key-1",
	}, "pay4_ops", "127.0.0.1")
	require.NoError(t, err)

	replay, err := f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{
		AmountCents: 50000, IdempotencyKey: "pay4-key-1",
	}, "pay4_ops", "127.0.0.1")
	require.NoError(t, err, "重复提交必须返回首次结果而不是报错")

	assert.Equal(t, first.PaymentID, replay.PaymentID, "重复提交必须返回同一收款标识")
	assert.EqualValues(t, 1, countPayments(t, db, f.order.ID), "重复提交不得产生第二条收款")
	assert.EqualValues(t, 1, countCommissions(t, db, f.order.ID), "重复提交不得产生第二份佣金")
	assert.EqualValues(t, 50000, replay.AmountPaidCents)
}

// S1Bpay-2：缺少幂等键的旧请求仍可写入，但不享受幂等保护。
func TestPaymentIntegrity_LegacyRequestsWithoutKeyAreNotDeduplicated(t *testing.T) {
	db := openPlatformTestDB(t)
	beginPaymentIntegrityTest(t, db)
	f := newPaymentIntegrityFixture(t, db, "B")
	ctx := context.Background()

	first, err := f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{AmountCents: 1000}, "pay4_ops", "127.0.0.1")
	require.NoError(t, err)
	second, err := f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{AmountCents: 1000}, "pay4_ops", "127.0.0.1")
	require.NoError(t, err, "缺少幂等键的旧请求必须继续可写")

	assert.NotEqual(t, first.PaymentID, second.PaymentID)
	assert.EqualValues(t, 2, countPayments(t, db, f.order.ID))
}

// S1Bpay-3：幂等键超长必须被拒绝，不能把截断风险留给数据库。
func TestPaymentIntegrity_RejectsOversizedIdempotencyKey(t *testing.T) {
	db := openPlatformTestDB(t)
	beginPaymentIntegrityTest(t, db)
	f := newPaymentIntegrityFixture(t, db, "C")
	ctx := context.Background()

	_, err := f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{
		AmountCents: 1000, IdempotencyKey: strings.Repeat("k", 65),
	}, "pay4_ops", "127.0.0.1")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidIdempotencyKey)
	assert.EqualValues(t, 0, countPayments(t, db, f.order.ID))
}

// S1Bpay-4：正数收款不得超过订单应收，超出的部分与剩余额度可被识别。
func TestPaymentIntegrity_PositivePaymentCannotExceedAmountDue(t *testing.T) {
	db := openPlatformTestDB(t)
	beginPaymentIntegrityTest(t, db)
	f := newPaymentIntegrityFixture(t, db, "D")
	ctx := context.Background()

	_, err := f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{AmountCents: 60000}, "pay4_ops", "127.0.0.1")
	require.NoError(t, err)

	_, err = f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{AmountCents: 50000}, "pay4_ops", "127.0.0.1")
	require.Error(t, err, "超过应收的收款必须被拒绝")
	var exceeds *PaymentExceedsAmountDueError
	require.ErrorAs(t, err, &exceeds)
	assert.EqualValues(t, 10000, exceeds.ExcessCents, "必须给出超出金额")
	assert.EqualValues(t, 40000, exceeds.RemainingCents, "必须给出剩余应收")
	assert.ErrorIs(t, err, ErrPaymentExceedsAmountDue)

	paid, err := repository.NewPaymentRepo(db).SumByOrder(ctx, f.order.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 60000, paid, "被拒绝时已收金额不变")
	assert.EqualValues(t, 1, countPayments(t, db, f.order.ID), "被拒绝时不得留下收款行")
	assert.EqualValues(t, 1, countCommissions(t, db, f.order.ID), "被拒绝时不得留下佣金行")

	// 恰好补齐剩余应收仍然允许。
	res, err := f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{AmountCents: 40000}, "pay4_ops", "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusPaid, res.OrderStatus)
	assert.EqualValues(t, 100000, res.AmountPaidCents)
}

// S1Bpay-5：退款绝对值不得超过当前已收净额。
func TestPaymentIntegrity_RefundCannotExceedPaidAmount(t *testing.T) {
	db := openPlatformTestDB(t)
	beginPaymentIntegrityTest(t, db)
	f := newPaymentIntegrityFixture(t, db, "E")
	ctx := context.Background()

	pay, err := f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{AmountCents: 50000}, "pay4_ops", "127.0.0.1")
	require.NoError(t, err)
	target := pay.PaymentID

	_, err = f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{
		AmountCents: -50001, RefundOfPaymentID: &target,
	}, "pay4_ops", "127.0.0.1")
	require.Error(t, err, "超过已收的退款必须被拒绝")

	paid, err := repository.NewPaymentRepo(db).SumByOrder(ctx, f.order.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 50000, paid, "被拒绝时已收金额不变")
	assert.EqualValues(t, 1, countPayments(t, db, f.order.ID))

	// 历史遗留的无链接负数行会把净已收压低，此时即使被退收款仍显示可退，
	// 也不得让订单已收净额变成负数。
	require.NoError(t, db.Create(&domain.Payment{
		OrderID: f.order.ID, KindergartenID: f.kg.ID, AmountCents: -50000,
		Method: domain.PaymentMethodBankTransfer, ReceivedAt: time.Now(), RecordedBy: "legacy",
	}).Error)

	_, err = f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{
		AmountCents: -1, RefundOfPaymentID: &target,
	}, "pay4_ops", "127.0.0.1")
	require.Error(t, err, "净已收为 0 时不得再退款")
	var exceeds *RefundExceedsPaidAmountError
	require.ErrorAs(t, err, &exceeds)
	assert.EqualValues(t, 0, exceeds.AvailableCents)
	assert.ErrorIs(t, err, ErrRefundExceedsPaidAmount)
}

// S1Bpay-6：到账时间不得晚于当前时间。
func TestPaymentIntegrity_RejectsFutureReceivedAt(t *testing.T) {
	db := openPlatformTestDB(t)
	beginPaymentIntegrityTest(t, db)
	f := newPaymentIntegrityFixture(t, db, "F")
	ctx := context.Background()

	future := time.Now().Add(2 * time.Hour).Format(time.RFC3339)
	_, err := f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{
		AmountCents: 10000, ReceivedAt: future,
	}, "pay4_ops", "127.0.0.1")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrReceivedAtInFuture)
	assert.EqualValues(t, 0, countPayments(t, db, f.order.ID), "被拒绝时不得留下收款行")
	assert.EqualValues(t, 0, countCommissions(t, db, f.order.ID), "被拒绝时不得留下佣金行")
}

// S1Bpay-7：回填历史 received_at 能落库，但佣金可结算日按服务端登记时间计算，
// 不能用回填日期把佣金「催熟」。
func TestPaymentIntegrity_BackdatedReceivedAtDoesNotAccelerateCommission(t *testing.T) {
	db := openPlatformTestDB(t)
	beginPaymentIntegrityTest(t, db)
	f := newPaymentIntegrityFixture(t, db, "G")
	ctx := context.Background()

	backdated := time.Now().AddDate(0, 0, -60)
	res, err := f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{
		AmountCents: 50000, ReceivedAt: backdated.Format(time.RFC3339),
	}, "pay4_ops", "127.0.0.1")
	require.NoError(t, err, "历史到账时间允许回填")

	var payment domain.Payment
	require.NoError(t, db.First(&payment, res.PaymentID).Error)
	assert.WithinDuration(t, backdated, payment.ReceivedAt, time.Minute, "received_at 按回填值落库，供账务展示")

	var commission domain.CommissionLedger
	require.NoError(t, db.Where("payment_id = ? AND reversal_of IS NULL", res.PaymentID).First(&commission).Error)
	assert.True(t, commission.EligibleAt.After(time.Now().AddDate(0, 0, 29)),
		"可结算日必须基于服务端登记时间，而不是回填的 received_at")
	assert.True(t, commission.EligibleAt.Before(time.Now().AddDate(0, 0, 31)))
	assert.Equal(t, domain.CommissionStatusPending, commission.Status)

	// 惰性推进也不能把回填日期的佣金提前置为可结算。
	require.NoError(t, repository.NewCommissionRepo(db).PromoteEligible(ctx, time.Now()))
	require.NoError(t, db.First(&commission, commission.ID).Error)
	assert.Equal(t, domain.CommissionStatusPending, commission.Status)
	assert.EqualValues(t, 5000, commission.AmountCents)
}
