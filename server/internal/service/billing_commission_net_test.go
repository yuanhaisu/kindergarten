package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// 本文件的测试只清理 S1Bnet- 前缀的数据，避免与其他并行切片共享清理。
func cleanupCommissionNet(t *testing.T, db *gorm.DB) {
	t.Helper()
	del := func(sql string, args ...any) {
		if err := db.Exec(sql, args...).Error; err != nil {
			t.Logf("cleanup %q: %v", sql, err)
		}
	}
	del("DELETE FROM commission_ledger WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE 'S1Bnet-%')")
	del("DELETE FROM payments WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE 'S1Bnet-%')")
	del("DELETE FROM entitlements WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE 'S1Bnet-%')")
	del("DELETE FROM orders WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE 'S1Bnet-%')")
	del("DELETE FROM students WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE 'S1Bnet-%')")
	del("DELETE FROM classes WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE 'S1Bnet-%')")
	del("DELETE FROM kindergartens WHERE name LIKE 'S1Bnet-%'")
	del("DELETE FROM plan_modules WHERE plan_id IN (SELECT id FROM plans WHERE code LIKE 's1bnet_%')")
	del("DELETE FROM plan_tiers WHERE plan_id IN (SELECT id FROM plans WHERE code LIKE 's1bnet_%')")
	del("DELETE FROM plans WHERE code LIKE 's1bnet_%'")
	del("DELETE FROM distributors WHERE name LIKE 'S1Bnet-%'")
}

func beginCommissionNetTest(t *testing.T, db *gorm.DB) {
	t.Helper()
	cleanupCommissionNet(t, db)
	t.Cleanup(func() { cleanupCommissionNet(t, db) })
}

type commissionNetFixture struct {
	kg       *domain.Kindergarten
	plan     *domain.Plan
	dist     *domain.Distributor
	order    *domain.Order
	planSvc  *PlanService
	orderSvc *OrderService
}

// newCommissionNetFixture 建一套独立的园所 / 套餐 / 分销商 / 已确认订单。
func newCommissionNetFixture(t *testing.T, db *gorm.DB, suffix string) *commissionNetFixture {
	t.Helper()
	ctx := context.Background()
	planSvc, orderSvc, _ := newBillingFlowServices(t, db)
	kg := &domain.Kindergarten{Name: "S1Bnet-佣金测试园" + suffix, Status: domain.KindergartenStatusActive}
	require.NoError(t, repository.NewKindergartenRepo(db).Create(ctx, kg))

	plan, err := planSvc.Create(ctx, PlanRequest{
		Code: "s1bnet_plan" + suffix, Name: "S1Bnet套餐" + suffix,
		Tier: 1, PriceCents: 100000, TermMonths: 6, ModuleCodes: []string{"base"},
	}, "s1bnet_ops", "127.0.0.1")
	require.NoError(t, err)

	dist := &domain.Distributor{Name: "S1Bnet-分销" + suffix, CommissionRate: 10, Status: 1}
	require.NoError(t, repository.NewDistributorRepo(db).Create(ctx, dist))

	order, err := orderSvc.Create(ctx, CreateOrderRequest{
		KindergartenID: kg.ID, PlanID: plan.ID, DistributorID: &dist.ID,
	}, "s1bnet_ops", "127.0.0.1")
	require.NoError(t, err)
	_, _, err = orderSvc.Confirm(ctx, order.ID, "s1bnet_ops", "127.0.0.1")
	require.NoError(t, err)

	return &commissionNetFixture{
		kg: kg, plan: plan, dist: dist, order: order,
		planSvc: planSvc, orderSvc: orderSvc,
	}
}

func commissionByPayment(t *testing.T, db *gorm.DB, paymentID uint64) domain.CommissionLedger {
	t.Helper()
	var row domain.CommissionLedger
	require.NoError(t, db.Where("payment_id = ? AND reversal_of IS NULL", paymentID).First(&row).Error)
	return row
}

// S1Bnet-1：退款必须指明被退收款，冲回行精确链接到该笔收款的佣金，
// 而不是订单下最近一条佣金。
func TestCommissionNet_RefundLinksExactPaymentCommission(t *testing.T) {
	db := openPlatformTestDB(t)
	beginCommissionNetTest(t, db)
	f := newCommissionNetFixture(t, db, "A")
	ctx := context.Background()

	first, err := f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{AmountCents: 60000}, "s1bnet_ops", "127.0.0.1")
	require.NoError(t, err)
	second, err := f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{AmountCents: 40000}, "s1bnet_ops", "127.0.0.1")
	require.NoError(t, err)

	firstCommission := commissionByPayment(t, db, first.PaymentID)
	secondCommission := commissionByPayment(t, db, second.PaymentID)
	require.EqualValues(t, 6000, firstCommission.AmountCents)
	require.EqualValues(t, 4000, secondCommission.AmountCents)

	// 退第一笔（不是最近一笔）收款。
	refundTarget := first.PaymentID
	refund, err := f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{
		AmountCents: -60000, RefundOfPaymentID: &refundTarget,
	}, "s1bnet_ops", "127.0.0.1")
	require.NoError(t, err)

	var reversal domain.CommissionLedger
	require.NoError(t, db.Where("order_id = ? AND reversal_of IS NOT NULL", f.order.ID).First(&reversal).Error)
	require.NotNil(t, reversal.ReversalOf)
	assert.Equal(t, firstCommission.ID, *reversal.ReversalOf, "冲回行必须指向被退收款对应的佣金")
	assert.Equal(t, refund.PaymentID, reversal.PaymentID, "冲回行记录退款收款")
	assert.EqualValues(t, -6000, reversal.AmountCents)

	reloadedFirst := commissionByPayment(t, db, first.PaymentID)
	reloadedSecond := commissionByPayment(t, db, second.PaymentID)
	assert.EqualValues(t, 6000, reloadedFirst.ReversedAmountCents, "原始佣金记录已冲回金额")
	assert.EqualValues(t, 0, reloadedSecond.ReversedAmountCents, "未被退的佣金不受影响")

	var refundPayment domain.Payment
	require.NoError(t, db.First(&refundPayment, refund.PaymentID).Error)
	require.NotNil(t, refundPayment.RefundOfPaymentID)
	assert.Equal(t, first.PaymentID, *refundPayment.RefundOfPaymentID, "退款收款必须留痕被退收款")

	// 缺少被退收款标识的退款必须被拒绝。
	_, err = f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{AmountCents: -100}, "s1bnet_ops", "127.0.0.1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "REFUND_PAYMENT_REQUIRED")

	// 指向其他订单的收款必须被拒绝。
	other := newCommissionNetFixture(t, db, "B")
	otherPay, err := other.orderSvc.AddPayment(ctx, other.order.ID, PaymentRequest{AmountCents: 10000}, "s1bnet_ops", "127.0.0.1")
	require.NoError(t, err)
	foreignTarget := otherPay.PaymentID
	_, err = f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{
		AmountCents: -1000, RefundOfPaymentID: &foreignTarget,
	}, "s1bnet_ops", "127.0.0.1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "REFUND_PAYMENT_NOT_FOUND")
}

// advanceCommissionEligible 把佣金行推进到「已满 30 天」，避开对 received_at 回填的依赖。
func advanceCommissionEligible(t *testing.T, db *gorm.DB, commissionID uint64) {
	t.Helper()
	require.NoError(t, db.Model(&domain.CommissionLedger{}).
		Where("id = ?", commissionID).
		Update("eligible_at", time.Now().Add(-24*time.Hour)).Error)
}

// S1Bnet-3：部分退款后，剩余净额仍可在满 30 天后进入可结算；
// 明细附带冲回记录，汇总与明细口径一致。
func TestCommissionNet_PartialRefundKeepsRemainingNetSettleable(t *testing.T) {
	db := openPlatformTestDB(t)
	beginCommissionNetTest(t, db)
	f := newCommissionNetFixture(t, db, "D")
	ctx := context.Background()
	distSvc := newArchiveDistributorService(db)

	pay, err := f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{AmountCents: 100000}, "s1bnet_ops", "127.0.0.1")
	require.NoError(t, err)
	target := pay.PaymentID
	refund, err := f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{
		AmountCents: -40000, RefundOfPaymentID: &target,
	}, "s1bnet_ops", "127.0.0.1")
	require.NoError(t, err)

	original := commissionByPayment(t, db, pay.PaymentID)
	assert.EqualValues(t, 4000, original.ReversedAmountCents)
	assert.Equal(t, domain.CommissionStatusPending, original.Status, "部分冲回不改变未满 30 天的待结算状态")

	// 未满 30 天：净额 6000 仍待结算。
	rows, _, summary, err := distSvc.ListCommissions(ctx, repository.CommissionFilter{DistributorID: f.dist.ID})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.EqualValues(t, 6000, rows[0].NetAmountCents)
	assert.EqualValues(t, 6000, summary[domain.CommissionStatusPending])
	assert.EqualValues(t, 4000, summary[domain.CommissionStatusReversed])

	advanceCommissionEligible(t, db, original.ID)
	rows, _, summary, err = distSvc.ListCommissions(ctx, repository.CommissionFilter{DistributorID: f.dist.ID})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, domain.CommissionStatusSettleable, rows[0].Status, "剩余净额满 30 天后可结算")
	assert.EqualValues(t, 6000, rows[0].NetAmountCents)
	assert.EqualValues(t, 6000, summary[domain.CommissionStatusSettleable], "汇总取净额")
	assert.EqualValues(t, 4000, summary[domain.CommissionStatusReversed], "已冲回卡片为累计冲回金额")
	assert.EqualValues(t, 0, summary[domain.CommissionStatusPending])

	// 明细可追溯：冲回记录指向被退的那笔收款。
	detail, err := distSvc.GetCommission(ctx, original.ID)
	require.NoError(t, err)
	require.Len(t, detail.Reversals, 1)
	assert.Equal(t, refund.PaymentID, detail.Reversals[0].PaymentID)
	assert.EqualValues(t, -4000, detail.Reversals[0].AmountCents)

	// 退掉剩余 60000：净额归零后不再可结算。
	_, err = f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{
		AmountCents: -60000, RefundOfPaymentID: &target,
	}, "s1bnet_ops", "127.0.0.1")
	require.NoError(t, err)
	reloaded := commissionByPayment(t, db, pay.PaymentID)
	assert.EqualValues(t, 10000, reloaded.ReversedAmountCents)
	assert.EqualValues(t, 0, reloaded.AmountCents-reloaded.ReversedAmountCents)
	assert.Equal(t, domain.CommissionStatusReversed, reloaded.Status, "净额归零后即使此前可结算也转为已冲回")

	_, _, summary, err = distSvc.ListCommissions(ctx, repository.CommissionFilter{DistributorID: f.dist.ID})
	require.NoError(t, err)
	assert.EqualValues(t, 0, summary[domain.CommissionStatusSettleable])
	assert.EqualValues(t, 10000, summary[domain.CommissionStatusReversed])
}

// S1Bnet-4：退款不能超过被退收款的可退金额，也不能指向退款收款自身。
func TestCommissionNet_RefundTargetValidation(t *testing.T) {
	db := openPlatformTestDB(t)
	beginCommissionNetTest(t, db)
	f := newCommissionNetFixture(t, db, "E")
	ctx := context.Background()

	pay, err := f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{AmountCents: 50000}, "s1bnet_ops", "127.0.0.1")
	require.NoError(t, err)
	target := pay.PaymentID
	refund, err := f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{
		AmountCents: -20000, RefundOfPaymentID: &target,
	}, "s1bnet_ops", "127.0.0.1")
	require.NoError(t, err)

	// 超过该笔收款剩余可退金额。
	_, err = f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{
		AmountCents: -30001, RefundOfPaymentID: &target,
	}, "s1bnet_ops", "127.0.0.1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "REFUND_EXCEEDS_PAYMENT")

	// 把退款收款当作被退对象。
	refundTarget := refund.PaymentID
	_, err = f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{
		AmountCents: -1000, RefundOfPaymentID: &refundTarget,
	}, "s1bnet_ops", "127.0.0.1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "REFUND_PAYMENT_INVALID")

	// 剩余可退金额内允许继续退款。
	_, err = f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{
		AmountCents: -30000, RefundOfPaymentID: &target,
	}, "s1bnet_ops", "127.0.0.1")
	require.NoError(t, err)

	// 两笔退款各生成一条冲回行，都精确指向同一笔原始佣金。
	original := commissionByPayment(t, db, target)
	var reversals []domain.CommissionLedger
	require.NoError(t, db.Where("reversal_of = ?", original.ID).Order("id ASC").Find(&reversals).Error)
	require.Len(t, reversals, 2, "每笔退款生成且仅生成一条冲回行")
	assert.EqualValues(t, -2000, reversals[0].AmountCents)
	assert.EqualValues(t, -3000, reversals[1].AmountCents)
	assert.EqualValues(t, 5000, original.ReversedAmountCents, "冲回金额累计")
	assert.EqualValues(t, 0, original.AmountCents-original.ReversedAmountCents)
	assert.Equal(t, domain.CommissionStatusReversed, original.Status)

	// 台账明细仍只有一条原始佣金行。
	distSvc := newArchiveDistributorService(db)
	rows, total, _, err := distSvc.ListCommissions(ctx, repository.CommissionFilter{DistributorID: f.dist.ID})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, rows, 1)
}

// S1Bnet-5：多笔部分退款的整数取整尾差收敛到最后一笔冲回，
// 冲回行金额合计与原始行累计冲回严格相等，审计可对账。
func TestCommissionNet_ReversalRowsSumToAccumulated(t *testing.T) {
	db := openPlatformTestDB(t)
	beginCommissionNetTest(t, db)
	f := newCommissionNetFixture(t, db, "F")
	ctx := context.Background()

	pay, err := f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{AmountCents: 100000}, "s1bnet_ops", "127.0.0.1")
	require.NoError(t, err)
	target := pay.PaymentID
	// 10% 下每笔 33333 元折算 3333 分，三笔合计会比原始佣金少 1 分。
	for _, amount := range []int64{-33333, -33333, -33334} {
		_, err := f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{
			AmountCents: amount, RefundOfPaymentID: &target,
		}, "s1bnet_ops", "127.0.0.1")
		require.NoError(t, err)
	}

	original := commissionByPayment(t, db, target)
	assert.EqualValues(t, 10000, original.ReversedAmountCents)
	assert.EqualValues(t, 0, original.AmountCents-original.ReversedAmountCents)
	assert.Equal(t, domain.CommissionStatusReversed, original.Status)

	var reversals []domain.CommissionLedger
	require.NoError(t, db.Where("reversal_of = ?", original.ID).Order("id ASC").Find(&reversals).Error)
	require.Len(t, reversals, 3)
	var sum int64
	for _, r := range reversals {
		assert.Greater(t, r.BaseAmountCents, int64(0), "冲回行计佣基数记为正数退款基数")
		assert.Less(t, r.AmountCents, int64(0), "冲回行金额为负")
		sum += -r.AmountCents
	}
	assert.EqualValues(t, original.ReversedAmountCents, sum, "冲回行金额合计必须等于累计冲回")
	assert.EqualValues(t, -3334, reversals[2].AmountCents, "尾差收敛到最后一笔退款")
}

func TestCommissionNet_FullRefundNeverSettleable(t *testing.T) {
	db := openPlatformTestDB(t)
	beginCommissionNetTest(t, db)
	f := newCommissionNetFixture(t, db, "C")
	ctx := context.Background()
	distSvc := newArchiveDistributorService(db)

	pay, err := f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{AmountCents: 100000}, "s1bnet_ops", "127.0.0.1")
	require.NoError(t, err)
	target := pay.PaymentID
	_, err = f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{
		AmountCents: -100000, RefundOfPaymentID: &target,
	}, "s1bnet_ops", "127.0.0.1")
	require.NoError(t, err)

	original := commissionByPayment(t, db, pay.PaymentID)
	assert.EqualValues(t, 10000, original.ReversedAmountCents)
	assert.Equal(t, domain.CommissionStatusReversed, original.Status, "净额归零即视为已冲回")

	advanceCommissionEligible(t, db, original.ID)

	rows, total, summary, err := distSvc.ListCommissions(ctx, repository.CommissionFilter{DistributorID: f.dist.ID})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total, "台账只列原始佣金行，冲回行仅作审计")
	require.Len(t, rows, 1)
	assert.EqualValues(t, 10000, rows[0].AmountCents, "明细保留原始佣金金额")
	assert.EqualValues(t, 10000, rows[0].ReversedAmountCents)
	assert.EqualValues(t, 0, rows[0].NetAmountCents, "明细输出净额")
	assert.Equal(t, domain.CommissionStatusReversed, rows[0].Status)
	assert.EqualValues(t, 0, summary[domain.CommissionStatusSettleable], "净额归零的佣金不得可结算")
	assert.EqualValues(t, 10000, summary[domain.CommissionStatusReversed], "已冲回汇总为累计冲回金额")

	// 再读一次也不能被惰性推进「复活」。
	_, _, summary, err = distSvc.ListCommissions(ctx, repository.CommissionFilter{DistributorID: f.dist.ID})
	require.NoError(t, err)
	assert.EqualValues(t, 0, summary[domain.CommissionStatusSettleable])
	reloaded := commissionByPayment(t, db, pay.PaymentID)
	assert.Equal(t, domain.CommissionStatusReversed, reloaded.Status)
}
