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

// 本文件是商业计费一致性的跨切片验收：把退款净额、取消守卫、收款幂等/上限、
// 在读幼儿数档位放在同一条链路上跑。只清理 S1Bclose- 前缀的数据。
const closeoutPlanCode = "close7_plan"

func cleanupCommercialCloseout(t *testing.T, db *gorm.DB) {
	t.Helper()
	del := func(sql string, args ...any) {
		if err := db.Exec(sql, args...).Error; err != nil {
			t.Logf("cleanup %q: %v", sql, err)
		}
	}
	const kgLike = "S1Bclose-%"
	del("DELETE FROM commission_ledger WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE ?)", kgLike)
	del("DELETE FROM payments WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE ?)", kgLike)
	del("DELETE FROM entitlements WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE ?)", kgLike)
	del("DELETE FROM orders WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE ?)", kgLike)
	del("DELETE FROM students WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE ?)", kgLike)
	del("DELETE FROM classes WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE ?)", kgLike)
	del("DELETE FROM kindergartens WHERE name LIKE ?", kgLike)
	del("DELETE FROM plan_modules WHERE plan_id IN (SELECT id FROM plans WHERE code LIKE 'close7_%')")
	del("DELETE FROM plan_tiers WHERE plan_id IN (SELECT id FROM plans WHERE code LIKE 'close7_%')")
	del("DELETE FROM plans WHERE code LIKE 'close7_%'")
	del("DELETE FROM distributors WHERE name LIKE ?", kgLike)
}

func beginCommercialCloseout(t *testing.T, db *gorm.DB) {
	t.Helper()
	cleanupCommercialCloseout(t, db)
	t.Cleanup(func() { cleanupCommercialCloseout(t, db) })
}

type closeoutFixture struct {
	kg       *domain.Kindergarten
	plan     *domain.Plan
	dist     *domain.Distributor
	order    *domain.Order
	planSvc  *PlanService
	orderSvc *OrderService
}

func newCloseoutFixture(t *testing.T, db *gorm.DB, suffix string) *closeoutFixture {
	t.Helper()
	ctx := context.Background()
	planSvc, orderSvc, _ := newBillingFlowServices(t, db)

	kg := &domain.Kindergarten{Name: "S1Bclose-闭环园" + suffix, Status: domain.KindergartenStatusActive}
	require.NoError(t, repository.NewKindergartenRepo(db).Create(ctx, kg))

	plan, err := planSvc.Create(ctx, PlanRequest{
		Code: closeoutPlanCode + suffix, Name: "S1Bclose-套餐" + suffix,
		Tier: 1, PriceCents: 100000, TermMonths: 6, ModuleCodes: []string{"base", "growth"},
	}, "close7_ops", "127.0.0.1")
	require.NoError(t, err)

	dist := &domain.Distributor{Name: "S1Bclose-分销" + suffix, CommissionRate: 10, Status: 1}
	require.NoError(t, repository.NewDistributorRepo(db).Create(ctx, dist))

	order, err := orderSvc.Create(ctx, CreateOrderRequest{
		KindergartenID: kg.ID, PlanID: plan.ID, DistributorID: &dist.ID,
	}, "close7_ops", "127.0.0.1")
	require.NoError(t, err)
	_, _, err = orderSvc.Confirm(ctx, order.ID, "测试信用开通", "close7_ops", "127.0.0.1")
	require.NoError(t, err)

	return &closeoutFixture{kg: kg, plan: plan, dist: dist, order: order, planSvc: planSvc, orderSvc: orderSvc}
}

func closeoutCommission(t *testing.T, db *gorm.DB, paymentID uint64) domain.CommissionLedger {
	t.Helper()
	var row domain.CommissionLedger
	require.NoError(t, db.Where("payment_id = ? AND reversal_of IS NULL", paymentID).First(&row).Error)
	return row
}

func advanceCloseoutCommission(t *testing.T, db *gorm.DB, commissionID uint64) {
	t.Helper()
	require.NoError(t, db.Model(&domain.CommissionLedger{}).
		Where("id = ?", commissionID).
		Update("eligible_at", time.Now().Add(-24*time.Hour)).Error)
}

// TC-NEW-ORD-0009 / DST-0008：收款→部分退款→满 30 天按剩余净额可结算；
// 再退掉剩余部分后净额归零，即使已过结算日也不再可结算。
func TestCommercialCloseout_RefundNetZeroAfter30Days(t *testing.T) {
	db := openPlatformTestDB(t)
	beginCommercialCloseout(t, db)
	f := newCloseoutFixture(t, db, "A")
	ctx := context.Background()
	distSvc := newArchiveDistributorService(db)

	pay, err := f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{
		AmountCents: 100000, IdempotencyKey: "close7-refund-1",
	}, "close7_ops", "127.0.0.1")
	require.NoError(t, err)

	original := closeoutCommission(t, db, pay.PaymentID)
	require.EqualValues(t, 10000, original.AmountCents)

	target := pay.PaymentID
	_, err = f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{
		AmountCents: -40000, RefundOfPaymentID: &target,
	}, "close7_ops", "127.0.0.1")
	require.NoError(t, err)

	advanceCloseoutCommission(t, db, original.ID)
	rows, _, summary, err := distSvc.ListCommissions(ctx, repository.CommissionFilter{DistributorID: f.dist.ID})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, domain.CommissionStatusSettleable, rows[0].Status)
	assert.EqualValues(t, 6000, rows[0].NetAmountCents)
	assert.EqualValues(t, 6000, summary[domain.CommissionStatusSettleable])

	_, err = f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{
		AmountCents: -60000, RefundOfPaymentID: &target,
	}, "close7_ops", "127.0.0.1")
	require.NoError(t, err)

	reloaded := closeoutCommission(t, db, pay.PaymentID)
	assert.EqualValues(t, 10000, reloaded.ReversedAmountCents)
	assert.EqualValues(t, 0, reloaded.AmountCents-reloaded.ReversedAmountCents)
	assert.Equal(t, domain.CommissionStatusReversed, reloaded.Status, "净额归零后不得再可结算")

	rows, _, summary, err = distSvc.ListCommissions(ctx, repository.CommissionFilter{DistributorID: f.dist.ID})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.EqualValues(t, 0, rows[0].NetAmountCents)
	assert.EqualValues(t, 0, summary[domain.CommissionStatusSettleable], "汇总不得包含已退款佣金")
	assert.EqualValues(t, 10000, summary[domain.CommissionStatusReversed])
}

// TC-NEW-ORD-0010：有未退款收款时取消被拒并给出待退金额；退清后可取消并收回授权。
func TestCommercialCloseout_CancelGuardThenRefundThenCancel(t *testing.T) {
	db := openPlatformTestDB(t)
	beginCommercialCloseout(t, db)
	f := newCloseoutFixture(t, db, "B")
	ctx := context.Background()

	pay, err := f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{AmountCents: 100000}, "close7_ops", "127.0.0.1")
	require.NoError(t, err)

	_, err = f.orderSvc.Cancel(ctx, f.order.ID, "园所暂缓采购", "close7_ops", "127.0.0.1")
	require.Error(t, err)
	var unrefunded *UnrefundedPaymentError
	require.ErrorAs(t, err, &unrefunded)
	assert.EqualValues(t, 100000, unrefunded.AmountCents)
	assert.ErrorIs(t, err, ErrOrderHasUnrefundedPayment)

	before, err := repository.NewOrderRepo(db).FindByID(ctx, f.order.ID)
	require.NoError(t, err)
	assert.Nil(t, before.CancelledAt)
	ents, err := repository.NewEntitlementRepo(db).ListByOrderID(ctx, f.order.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, ents, "拒绝取消时授权不清退")

	target := pay.PaymentID
	_, err = f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{
		AmountCents: -100000, RefundOfPaymentID: &target,
	}, "close7_ops", "127.0.0.1")
	require.NoError(t, err)

	_, err = f.orderSvc.Cancel(ctx, f.order.ID, "园所暂缓采购", "close7_ops", "127.0.0.1")
	require.NoError(t, err)
	after, err := repository.NewOrderRepo(db).FindByID(ctx, f.order.ID)
	require.NoError(t, err)
	require.NotNil(t, after.CancelledAt)
	ents, err = repository.NewEntitlementRepo(db).ListByOrderID(ctx, f.order.ID)
	require.NoError(t, err)
	assert.Empty(t, ents, "取消后收回该订单授权")
}

// TC-NEW-ORD-0011 / ORD-0012 / ORD-0013：重复幂等键只落一条收款，
// 超额收款与超额退款被拒，未来 received_at 被拒。
func TestCommercialCloseout_PaymentIdempotencyAndCaps(t *testing.T) {
	db := openPlatformTestDB(t)
	beginCommercialCloseout(t, db)
	f := newCloseoutFixture(t, db, "C")
	ctx := context.Background()

	first, err := f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{
		AmountCents: 60000, IdempotencyKey: "close7-key-1",
	}, "close7_ops", "127.0.0.1")
	require.NoError(t, err)
	replay, err := f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{
		AmountCents: 60000, IdempotencyKey: "close7-key-1",
	}, "close7_ops", "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, first.PaymentID, replay.PaymentID)
	assert.EqualValues(t, 1, countCloseoutPayments(t, db, f.order.ID))
	assert.EqualValues(t, 1, countCloseoutCommissions(t, db, f.order.ID))

	_, err = f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{AmountCents: 50000}, "close7_ops", "127.0.0.1")
	require.Error(t, err, "超过应收的收款必须被拒绝")
	assert.EqualValues(t, 60000, paidCloseout(t, db, f.order.ID))

	target := first.PaymentID
	_, err = f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{
		AmountCents: -70000, RefundOfPaymentID: &target,
	}, "close7_ops", "127.0.0.1")
	require.Error(t, err, "超过已收的退款必须被拒绝")
	assert.Contains(t, err.Error(), "REFUND_EXCEEDS_PAYMENT", "拒绝原因必须是超额，而不是缺少退款关联")
	assert.EqualValues(t, 60000, paidCloseout(t, db, f.order.ID))

	_, err = f.orderSvc.AddPayment(ctx, f.order.ID, PaymentRequest{
		AmountCents: 10000, ReceivedAt: time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	}, "close7_ops", "127.0.0.1")
	require.Error(t, err, "未来 received_at 必须被拒绝")
	assert.EqualValues(t, 60000, paidCloseout(t, db, f.order.ID))
}

// TC-NEW-CAT-0011：毕业/退学/待入学不计入档位，只有在读幼儿决定档位。
func TestCommercialCloseout_OnlyActiveStudentsDriveTier(t *testing.T) {
	db := openPlatformTestDB(t)
	beginCommercialCloseout(t, db)
	ctx := context.Background()
	planSvc, orderSvc, _ := newBillingFlowServices(t, db)

	kg := &domain.Kindergarten{Name: "S1Bclose-口径园D", Status: domain.KindergartenStatusActive}
	require.NoError(t, repository.NewKindergartenRepo(db).Create(ctx, kg))
	class := &domain.Class{KindergartenID: kg.ID, Name: "S1Bclose-班", Capacity: 60, Status: 1}
	require.NoError(t, db.Create(class).Error)

	mk := func(status int8, n int) {
		for i := 0; i < n; i++ {
			// Status 带 default:1，GORM 会把 0 当零值省略，先按在读写入再显式改状态。
			st := &domain.Student{
				KindergartenID: kg.ID, ClassID: class.ID, Name: "S1Bclose-幼儿", Status: 1,
				Birthday: time.Now(), EnrollDate: time.Now(),
			}
			require.NoError(t, db.Create(st).Error)
			if status != 1 {
				require.NoError(t, db.Model(st).Update("status", status).Error)
			}
		}
	}
	mk(1, 2) // 在读
	mk(0, 2) // 待入学
	mk(2, 2) // 毕业
	mk(3, 1) // 退学

	plan, err := planSvc.Create(ctx, PlanRequest{
		Code: "close7_tier", Name: "S1Bclose-档位套餐",
		PriceCents: 100000, TermMonths: 6, ModuleCodes: []string{"base"},
	}, "close7_ops", "127.0.0.1")
	require.NoError(t, err)
	require.NoError(t, planSvc.ReplaceTiers(ctx, plan.ID, []domain.PlanTier{
		{Tier: 1, MinStudents: 0, MaxStudents: 2, PriceMultiplier: 100},
		{Tier: 2, MinStudents: 3, MaxStudents: 0, PriceMultiplier: 150},
	}, "close7_ops", "127.0.0.1"))

	order, err := orderSvc.Create(ctx, CreateOrderRequest{KindergartenID: kg.ID, PlanID: plan.ID}, "close7_ops", "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, 1, order.StudentTier, "只有 2 名在读，应落第一档")
	assert.EqualValues(t, 100000, order.AmountDueCents)
}

// TC-NEW-ORD-0001 / CAT-0003：确认后再改套餐（含新增模块），订单快照与老园所授权都不变。
func TestCommercialCloseout_PlanChangeAfterConfirmKeepsSnapshot(t *testing.T) {
	db := openPlatformTestDB(t)
	beginCommercialCloseout(t, db)
	f := newCloseoutFixture(t, db, "E")
	ctx := context.Background()

	before, err := f.orderSvc.Detail(ctx, f.order.ID)
	require.NoError(t, err)
	require.EqualValues(t, 100000, before.AmountDueCents)
	require.ElementsMatch(t, []string{"base", "growth"}, before.PlanSnapshot.ModuleCodes)

	require.NoError(t, f.planSvc.Update(ctx, f.plan.ID, PlanRequest{
		Name: "S1Bclose-套餐E改名", PriceCents: 200000,
		ModuleCodes: []string{"base", "growth", "diet"},
	}, "close7_ops", "127.0.0.1"))

	after, err := f.orderSvc.Detail(ctx, f.order.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 100000, after.AmountDueCents, "确认后改价不影响已成交订单")
	assert.Equal(t, "S1Bclose-套餐E", after.PlanSnapshot.PlanName, "确认后改名不影响快照")
	assert.ElementsMatch(t, []string{"base", "growth"}, after.PlanSnapshot.ModuleCodes, "确认后新增模块不进入快照")

	ents, err := repository.NewEntitlementRepo(db).ListByOrderID(ctx, f.order.ID)
	require.NoError(t, err)
	for _, e := range ents {
		assert.NotEqual(t, "diet", e.ModuleCode, "老园所不得获得套餐新增模块")
	}
	all, err := repository.NewEntitlementRepo(db).ListByKindergarten(ctx, f.kg.ID)
	require.NoError(t, err)
	for _, e := range all {
		assert.NotEqual(t, "diet", e.ModuleCode, "园所授权清单不得包含新增模块")
	}
}

// TC-NEW-ENT-0010：已降级的园所登记收款并确认订单生效后，立即恢复写能力。
func TestCommercialCloseout_PaymentThenConfirmRestoresWrite(t *testing.T) {
	db := openPlatformTestDB(t)
	beginCommercialCloseout(t, db)
	ctx := context.Background()
	planSvc, orderSvc, entSvc := newBillingFlowServices(t, db)

	kg := &domain.Kindergarten{Name: "S1Bclose-续费园F", Status: domain.KindergartenStatusActive}
	require.NoError(t, repository.NewKindergartenRepo(db).Create(ctx, kg))
	now := time.Now()
	require.NoError(t, repository.NewEntitlementRepo(db).CreateBatch(ctx, []domain.Entitlement{{
		KindergartenID: kg.ID, ModuleCode: "growth", Source: domain.EntitlementSourceTrial,
		StartsAt: now.AddDate(0, -2, 0), ExpiresAt: now.AddDate(0, 0, -30),
	}}))

	dec, err := entSvc.Decide(ctx, kg.ID, "growth", true)
	require.NoError(t, err)
	require.False(t, dec.Allowed, "过期超过宽限期后不可写")

	plan, err := planSvc.Create(ctx, PlanRequest{
		Code: "close7_restore", Name: "S1Bclose-续费套餐", PriceCents: 100000, TermMonths: 6,
		ModuleCodes: []string{"base", "growth"},
	}, "close7_ops", "127.0.0.1")
	require.NoError(t, err)
	order, err := orderSvc.Create(ctx, CreateOrderRequest{KindergartenID: kg.ID, PlanID: plan.ID}, "close7_ops", "127.0.0.1")
	require.NoError(t, err)
	_, err = orderSvc.AddPayment(ctx, order.ID, PaymentRequest{AmountCents: 100000}, "close7_ops", "127.0.0.1")
	require.NoError(t, err)
	_, _, err = orderSvc.Confirm(ctx, order.ID, "测试信用开通", "close7_ops", "127.0.0.1")
	require.NoError(t, err)

	dec, err = entSvc.Decide(ctx, kg.ID, "growth", true)
	require.NoError(t, err)
	assert.True(t, dec.Allowed, "收款并确认订单后立即恢复写能力")
	assert.Equal(t, "active", dec.Reason)
}

// TC-NEW-ENT-0013：超档只提示，不阻断新增幼儿，也不阻断 premium 写操作。
func TestCommercialCloseout_OverLimitDoesNotBlockExistingBusiness(t *testing.T) {
	db := openPlatformTestDB(t)
	beginCommercialCloseout(t, db)
	ctx := context.Background()
	planSvc, orderSvc, entSvc := newBillingFlowServices(t, db)

	kg := &domain.Kindergarten{Name: "S1Bclose-超档园G", Status: domain.KindergartenStatusActive}
	require.NoError(t, repository.NewKindergartenRepo(db).Create(ctx, kg))
	class := &domain.Class{KindergartenID: kg.ID, Name: "S1Bclose-超档班", Capacity: 60, Status: 1}
	require.NoError(t, db.Create(class).Error)
	for i := 0; i < 2; i++ {
		require.NoError(t, db.Create(&domain.Student{
			KindergartenID: kg.ID, ClassID: class.ID, Name: "S1Bclose-在读幼儿", Status: 1,
			Birthday: time.Now(), EnrollDate: time.Now(),
		}).Error)
	}

	plan, err := planSvc.Create(ctx, PlanRequest{
		Code: "close7_over", Name: "S1Bclose-超档套餐", PriceCents: 100000, TermMonths: 6,
		ModuleCodes: []string{"base", "growth"},
	}, "close7_ops", "127.0.0.1")
	require.NoError(t, err)
	require.NoError(t, planSvc.ReplaceTiers(ctx, plan.ID, []domain.PlanTier{
		{Tier: 1, MinStudents: 0, MaxStudents: 2, PriceMultiplier: 100},
		{Tier: 2, MinStudents: 3, MaxStudents: 0, PriceMultiplier: 150},
	}, "close7_ops", "127.0.0.1"))
	order, err := orderSvc.Create(ctx, CreateOrderRequest{KindergartenID: kg.ID, PlanID: plan.ID}, "close7_ops", "127.0.0.1")
	require.NoError(t, err)
	_, _, err = orderSvc.Confirm(ctx, order.ID, "测试信用开通", "close7_ops", "127.0.0.1")
	require.NoError(t, err)

	// 成交时 2 人落第一档（上限 2），之后新增到 3 人即超档。
	require.NoError(t, db.Create(&domain.Student{
		KindergartenID: kg.ID, ClassID: class.ID, Name: "S1Bclose-超档幼儿", Status: 1,
		Birthday: time.Now(), EnrollDate: time.Now(),
	}).Error)

	view, err := entSvc.Subscription(ctx, kg.ID)
	require.NoError(t, err)
	require.True(t, view.OverLimit, "3 名在读超过档位上限 2")
	assert.Equal(t, 3, int(view.StudentCount))
	assert.Equal(t, 2, view.TierMaxStudents)

	// 超限后新增幼儿不被阻断。
	require.NoError(t, db.Create(&domain.Student{
		KindergartenID: kg.ID, ClassID: class.ID, Name: "S1Bclose-超档后新增", Status: 1,
		Birthday: time.Now(), EnrollDate: time.Now(),
	}).Error)
	count, err := repository.NewKindergartenRepo(db).CountStudents(ctx, kg.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 4, count)

	dec, err := entSvc.Decide(ctx, kg.ID, "growth", true)
	require.NoError(t, err)
	assert.True(t, dec.Allowed, "超档不得阻断 premium 写操作")
}

func countCloseoutPayments(t *testing.T, db *gorm.DB, orderID uint64) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(&domain.Payment{}).Where("order_id = ?", orderID).Count(&n).Error)
	return n
}

func countCloseoutCommissions(t *testing.T, db *gorm.DB, orderID uint64) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(&domain.CommissionLedger{}).Where("order_id = ?", orderID).Count(&n).Error)
	return n
}

func paidCloseout(t *testing.T, db *gorm.DB, orderID uint64) int64 {
	t.Helper()
	var sum int64
	require.NoError(t, db.Model(&domain.Payment{}).Where("order_id = ?", orderID).
		Select("COALESCE(SUM(amount_cents),0)").Scan(&sum).Error)
	return sum
}
