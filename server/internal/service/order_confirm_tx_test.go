package service

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// 本切片验证订单确认的事务化与幂等（#200）：
// 确认必须在一个事务内锁订单行、重算净收款、派生状态、条件更新并检查影响行数，
// 授权与审计同事务落库；并发或双击确认只能有一个赢家。
// 取消同样锁单，避免「确认赢了、取消看见旧状态」的交叉写入。

const confirmTxPlanCode = "cn4_confirm_tx"
const confirmTxOperator = "cn4_ops"

func cleanupConfirmTx(t *testing.T, db *gorm.DB) {
	t.Helper()
	del := func(sql string, args ...any) {
		if err := db.Exec(sql, args...).Error; err != nil {
			t.Logf("cleanup %q: %v", sql, err)
		}
	}
	const kgLike = "S1Cconf-%"
	del("DELETE FROM platform_operation_logs WHERE operator = ?", confirmTxOperator)
	del("DELETE FROM commission_ledger WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE ?)", kgLike)
	del("DELETE FROM payments WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE ?)", kgLike)
	del("DELETE FROM entitlements WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE ?)", kgLike)
	del("DELETE FROM orders WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE ?)", kgLike)
	del("DELETE FROM kindergartens WHERE name LIKE ?", kgLike)
	del("DELETE FROM plan_modules WHERE plan_id IN (SELECT id FROM plans WHERE code = ?)", confirmTxPlanCode)
	del("DELETE FROM plan_tiers WHERE plan_id IN (SELECT id FROM plans WHERE code = ?)", confirmTxPlanCode)
	del("DELETE FROM plans WHERE code = ?", confirmTxPlanCode)
}

// confirmTxFixture 建一套园所 / 套餐（3 个 premium 模块 + base），premiumCount 即期望授权行数。
type confirmTxFixture struct {
	svc          *OrderService
	kg           *domain.Kindergarten
	premiumCount int
}

func newConfirmTxFixture(t *testing.T, db *gorm.DB, suffix string) *confirmTxFixture {
	t.Helper()
	ctx := context.Background()
	planSvc, orderSvc, _ := newBillingFlowServices(t, db)

	kg := &domain.Kindergarten{Name: "S1Cconf-确认事务园" + suffix, Status: domain.KindergartenStatusActive}
	require.NoError(t, repository.NewKindergartenRepo(db).Create(ctx, kg))

	plan, err := planSvc.Create(ctx, PlanRequest{
		Code: confirmTxPlanCode, Name: "S1Cconf-确认事务套餐", Tier: 1, PriceCents: 100000, TermMonths: 6,
		ModuleCodes: []string{"base", "growth", "album", "diet"},
	}, confirmTxOperator, "127.0.0.1")
	require.NoError(t, err)
	require.NotNil(t, plan)

	// 期望授权行数 = 套餐里的 premium 模块数（base 只随套餐走，不单独发授权）。
	premium := 0
	for _, code := range []string{"base", "growth", "album", "diet"} {
		if ModuleCategory(code) == domain.ModuleCategoryPremium {
			premium++
		}
	}
	require.GreaterOrEqual(t, premium, 2, "用例需要至少两个 premium 模块才能识别重复授权")

	return &confirmTxFixture{svc: orderSvc, kg: kg, premiumCount: premium}
}

func (f *confirmTxFixture) createOrder(t *testing.T, db *gorm.DB) *domain.Order {
	t.Helper()
	ctx := context.Background()
	plan, err := repository.NewPlanRepo(db).FindByCode(ctx, confirmTxPlanCode)
	require.NoError(t, err)
	require.NotNil(t, plan)
	order, err := f.svc.Create(ctx, CreateOrderRequest{KindergartenID: f.kg.ID, PlanID: plan.ID}, confirmTxOperator, "127.0.0.1")
	require.NoError(t, err)
	return order
}

func countEntitlementsOfOrder(t *testing.T, db *gorm.DB, orderID uint64) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(&domain.Entitlement{}).Where("order_id = ?", orderID).Count(&n).Error)
	return n
}

func countConfirmAudits(t *testing.T, db *gorm.DB, orderID uint64) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(&domain.PlatformOperationLog{}).
		Where("operator = ? AND module = ? AND action = ? AND target_id = ?", confirmTxOperator, "order", "confirm", orderID).
		Count(&n).Error)
	return n
}

func countCancelAudits(t *testing.T, db *gorm.DB, orderID uint64) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(&domain.PlatformOperationLog{}).
		Where("operator = ? AND module = ? AND action = ? AND target_id = ?", confirmTxOperator, "order", "cancel", orderID).
		Count(&n).Error)
	return n
}

// 并发确认（双击 / 多标签页）：恰好一个成功，其余 409，授权只发一套、确认审计只留一条。
func TestOrderConfirm_ConcurrentConfirmSingleWinner(t *testing.T) {
	db := openPlatformTestDB(t)
	ctx := context.Background()
	cleanupConfirmTx(t, db)
	t.Cleanup(func() { cleanupConfirmTx(t, db) })

	f := newConfirmTxFixture(t, db, "")
	order := f.createOrder(t, db)

	const workers = 8
	start := make(chan struct{})
	var wg sync.WaitGroup
	errs := make([]error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			_, _, errs[i] = f.svc.Confirm(ctx, order.ID, "并发确认信用开通", confirmTxOperator, "127.0.0.1")
		}(i)
	}
	close(start)
	wg.Wait()

	successes := 0
	for i, err := range errs {
		if err == nil {
			successes++
			continue
		}
		// handler 的 writeConfirmOrderError 把这两个错误映射为 409。
		assert.Contains(t, []string{"ORDER_ALREADY_CONFIRMED", "ORDER_CANCELLED"}, err.Error(),
			"并发失败请求必须是可映射 409 的状态错误（worker %d）", i)
	}
	assert.Equal(t, 1, successes, "并发确认必须恰好一个成功")
	assert.EqualValues(t, f.premiumCount, countEntitlementsOfOrder(t, db, order.ID), "授权行数必须等于 premium 模块数")
	assert.EqualValues(t, 1, countConfirmAudits(t, db, order.ID), "确认审计必须只有一条")

	fresh, err := repository.NewOrderRepo(db).FindByID(ctx, order.ID)
	require.NoError(t, err)
	require.NotNil(t, fresh.ConfirmedAt)
}

// 顺序重复确认：第二次直接 409，不得再发授权、再写审计。
func TestOrderConfirm_RepeatConfirmKeepsSingleEntitlementSet(t *testing.T) {
	db := openPlatformTestDB(t)
	ctx := context.Background()
	cleanupConfirmTx(t, db)
	t.Cleanup(func() { cleanupConfirmTx(t, db) })

	f := newConfirmTxFixture(t, db, "-重复")
	order := f.createOrder(t, db)

	_, _, err := f.svc.Confirm(ctx, order.ID, "顺序确认信用开通", confirmTxOperator, "127.0.0.1")
	require.NoError(t, err)

	_, _, err = f.svc.Confirm(ctx, order.ID, "顺序确认信用开通", confirmTxOperator, "127.0.0.1")
	require.Error(t, err)
	assert.Equal(t, "ORDER_ALREADY_CONFIRMED", err.Error())

	assert.EqualValues(t, f.premiumCount, countEntitlementsOfOrder(t, db, order.ID))
	assert.EqualValues(t, 1, countConfirmAudits(t, db, order.ID))
}

// 确认与全额收款并发：两者锁同一订单行串行化，确认不得用事务外的旧净额
// 把状态写回 unpaid，也不得重复授权。
func TestOrderConfirm_ConcurrentWithPaymentKeepsPaidState(t *testing.T) {
	db := openPlatformTestDB(t)
	ctx := context.Background()
	cleanupConfirmTx(t, db)
	t.Cleanup(func() { cleanupConfirmTx(t, db) })

	f := newConfirmTxFixture(t, db, "-收款")
	const rounds = 12
	for round := 0; round < rounds; round++ {
		order := f.createOrder(t, db)

		var wg sync.WaitGroup
		start := make(chan struct{})
		var confirmErr, payErr error
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			_, _, confirmErr = f.svc.Confirm(ctx, order.ID, "并发收款确认信用开通", confirmTxOperator, "127.0.0.1")
		}()
		go func() {
			defer wg.Done()
			<-start
			_, payErr = f.svc.AddPayment(ctx, order.ID, PaymentRequest{
				AmountCents: order.AmountDueCents, IdempotencyKey: fmt.Sprintf("cn4-round-%d", round),
			}, confirmTxOperator, "127.0.0.1")
		}()
		close(start)
		wg.Wait()

		require.NoError(t, confirmErr, "round %d: 确认与收款并发时确认必须成功", round)
		require.NoError(t, payErr, "round %d: 全额收款必须成功", round)

		fresh, err := repository.NewOrderRepo(db).FindByID(ctx, order.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.OrderStatusPaid, fresh.Status,
			"round %d: 收款已全额落库，确认不得把状态覆盖回未收清", round)
		assert.EqualValues(t, f.premiumCount, countEntitlementsOfOrder(t, db, order.ID),
			"round %d: 授权只发一套", round)
		assert.EqualValues(t, 1, countConfirmAudits(t, db, order.ID), "round %d: 确认审计只留一条", round)

		var payCount int64
		require.NoError(t, db.Model(&domain.Payment{}).Where("order_id = ?", order.ID).Count(&payCount).Error)
		assert.EqualValues(t, 1, payCount, "round %d: 只应有一笔收款", round)
	}
}

// 确认与取消并发：两者锁同一订单行串行化，不得出现「已取消但授权还在」
// 或重复授权；确认被取消挡住时不留授权与审计。
func TestOrderConfirm_ConcurrentWithCancelKeepsInvariants(t *testing.T) {
	db := openPlatformTestDB(t)
	ctx := context.Background()
	cleanupConfirmTx(t, db)
	t.Cleanup(func() { cleanupConfirmTx(t, db) })

	f := newConfirmTxFixture(t, db, "-取消")
	const rounds = 12
	for round := 0; round < rounds; round++ {
		order := f.createOrder(t, db)

		var wg sync.WaitGroup
		start := make(chan struct{})
		var confirmErr, cancelErr error
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			_, _, confirmErr = f.svc.Confirm(ctx, order.ID, "并发取消确认信用开通", confirmTxOperator, "127.0.0.1")
		}()
		go func() {
			defer wg.Done()
			<-start
			_, cancelErr = f.svc.Cancel(ctx, order.ID, "并发取消", confirmTxOperator, "127.0.0.1")
		}()
		close(start)
		wg.Wait()

		fresh, err := repository.NewOrderRepo(db).FindByID(ctx, order.ID)
		require.NoError(t, err)
		ents := countEntitlementsOfOrder(t, db, order.ID)
		audits := countConfirmAudits(t, db, order.ID)

		assert.LessOrEqual(t, ents, int64(f.premiumCount), "round %d: 授权不得超过 premium 模块数", round)
		assert.LessOrEqual(t, audits, int64(1), "round %d: 确认审计至多一条", round)
		assert.LessOrEqual(t, countCancelAudits(t, db, order.ID), int64(1), "round %d: 取消审计至多一条", round)

		if confirmErr != nil {
			assert.Equal(t, "ORDER_CANCELLED", confirmErr.Error(), "round %d: 取消先行时确认必须报已取消", round)
			assert.Nil(t, fresh.ConfirmedAt, "round %d: 被拒确认不得落库", round)
			assert.Zero(t, ents, "round %d: 被拒确认不得留下授权", round)
			assert.Zero(t, audits, "round %d: 被拒确认不得写审计", round)
		} else {
			assert.NotNil(t, fresh.ConfirmedAt, "round %d: 确认成功必须落库", round)
			assert.EqualValues(t, 1, audits, "round %d: 确认成功后审计一条", round)
			if fresh.CancelledAt == nil {
				assert.EqualValues(t, f.premiumCount, ents, "round %d: 未取消的已确认订单授权必须完整", round)
			}
		}
		if fresh.CancelledAt != nil {
			assert.Zero(t, ents, "round %d: 已取消订单不得残留授权", round)
		}
		if cancelErr != nil {
			assert.Equal(t, "ORDER_ALREADY_CANCELLED", cancelErr.Error(), "round %d: 取消失败只能是重复取消", round)
		}
	}
}

// 并发取消：锁单后恰好一个成功，另一个 409，审计只留一条。
func TestOrderCancel_ConcurrentCancelSingleWinner(t *testing.T) {
	db := openPlatformTestDB(t)
	ctx := context.Background()
	cleanupConfirmTx(t, db)
	t.Cleanup(func() { cleanupConfirmTx(t, db) })

	f := newConfirmTxFixture(t, db, "-并发取消")
	order := f.createOrder(t, db)
	_, _, err := f.svc.Confirm(ctx, order.ID, "并发取消前确认", confirmTxOperator, "127.0.0.1")
	require.NoError(t, err)

	const workers = 6
	start := make(chan struct{})
	var wg sync.WaitGroup
	errs := make([]error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			_, errs[i] = f.svc.Cancel(ctx, order.ID, "并发取消", confirmTxOperator, "127.0.0.1")
		}(i)
	}
	close(start)
	wg.Wait()

	successes := 0
	for i, err := range errs {
		if err == nil {
			successes++
			continue
		}
		assert.Equal(t, "ORDER_ALREADY_CANCELLED", err.Error(), "worker %d 必须得到重复取消", i)
	}
	assert.Equal(t, 1, successes, "并发取消必须恰好一个成功")
	assert.Zero(t, countEntitlementsOfOrder(t, db, order.ID), "取消后授权必须回收")
	assert.EqualValues(t, 1, countCancelAudits(t, db, order.ID), "取消审计只留一条")

	var fresh domain.Order
	require.NoError(t, db.First(&fresh, order.ID).Error)
	require.NotNil(t, fresh.CancelledAt)
}
