package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// 取消守卫用例使用独立前缀，避免与其他计费用例的清理语句互相误删。
const cancelGuardPlanCode = "cn3_cancel_guard"

func cleanupCancelGuard(t *testing.T, db *gorm.DB) {
	t.Helper()
	del := func(sql string, args ...any) {
		if err := db.Exec(sql, args...).Error; err != nil {
			t.Logf("cleanup %q: %v", sql, err)
		}
	}
	const kgLike = "S1Bcancel-%"
	del("DELETE FROM commission_ledger WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE ?)", kgLike)
	del("DELETE FROM payments WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE ?)", kgLike)
	del("DELETE FROM entitlements WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE ?)", kgLike)
	del("DELETE FROM orders WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE ?)", kgLike)
	del("DELETE FROM students WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE ?)", kgLike)
	del("DELETE FROM classes WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE ?)", kgLike)
	del("DELETE FROM kindergartens WHERE name LIKE ?", kgLike)
	// 显式列出自己的套餐编码，不用 LIKE：下划线是通配符。
	del("DELETE FROM plan_modules WHERE plan_id IN (SELECT id FROM plans WHERE code = ?)", cancelGuardPlanCode)
	del("DELETE FROM plan_tiers WHERE plan_id IN (SELECT id FROM plans WHERE code = ?)", cancelGuardPlanCode)
	del("DELETE FROM plans WHERE code = ?", cancelGuardPlanCode)
	del("DELETE FROM distributors WHERE name LIKE 'S1Bcancel-分销%'")
}

func newCancelGuardOrder(t *testing.T, db *gorm.DB, kgName string) (*OrderService, *domain.Order, *domain.Plan) {
	t.Helper()
	ctx := context.Background()
	planSvc := NewPlanService(repository.NewPlanRepo(db), repository.NewPlatformLogRepo(db))
	orderSvc := NewOrderService(
		db, repository.NewOrderRepo(db), repository.NewPaymentRepo(db),
		repository.NewPlanRepo(db), repository.NewKindergartenRepo(db),
		repository.NewEntitlementRepo(db), repository.NewDistributorRepo(db),
		repository.NewCommissionRepo(db), repository.NewPlatformLogRepo(db), nil,
	)
	kg := &domain.Kindergarten{Name: kgName, Status: domain.KindergartenStatusActive}
	require.NoError(t, repository.NewKindergartenRepo(db).Create(ctx, kg))
	plan, err := planSvc.Create(ctx, PlanRequest{
		Code: cancelGuardPlanCode, Name: "S1Bcancel-旗舰", Tier: 1, PriceCents: 100000, TermMonths: 6,
		ModuleCodes: []string{"base", "growth"},
	}, "s1b_ops", "127.0.0.1")
	require.NoError(t, err)
	order, err := orderSvc.Create(ctx, CreateOrderRequest{KindergartenID: kg.ID, PlanID: plan.ID}, "s1b_ops", "127.0.0.1")
	require.NoError(t, err)
	_, _, err = orderSvc.Confirm(ctx, order.ID, "测试信用开通", "s1b_ops", "127.0.0.1")
	require.NoError(t, err)
	return orderSvc, order, plan
}

// 已收净额大于 0 时取消必须被拒绝并给出未退款金额；退款净额归零后才可取消并收回授权。
func TestCancelGuard_RejectsUnrefundedCollectedOrder(t *testing.T) {
	db := openPlatformTestDB(t)
	ctx := context.Background()
	cleanupCancelGuard(t, db)
	t.Cleanup(func() { cleanupCancelGuard(t, db) })

	orderSvc, order, _ := newCancelGuardOrder(t, db, "S1Bcancel-已收款园")

	pay, err := orderSvc.AddPayment(ctx, order.ID, PaymentRequest{AmountCents: 100000}, "s1b_ops", "127.0.0.1")
	require.NoError(t, err)

	_, err = orderSvc.Cancel(ctx, order.ID, "园所暂缓采购", "s1b_ops", "127.0.0.1")
	require.Error(t, err, "有未退款收款时不得取消")
	var unrefunded *UnrefundedPaymentError
	require.ErrorAs(t, err, &unrefunded)
	assert.EqualValues(t, 100000, unrefunded.AmountCents, "必须返回未退款金额")
	assert.ErrorIs(t, err, ErrOrderHasUnrefundedPayment)

	after, err := repository.NewOrderRepo(db).FindByID(ctx, order.ID)
	require.NoError(t, err)
	assert.Nil(t, after.CancelledAt, "拒绝取消时订单保持未取消")
	assert.NotEqual(t, domain.OrderStatusCancelled, after.Status)
	ents, err := repository.NewEntitlementRepo(db).ListByOrderID(ctx, order.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, ents, "拒绝取消时授权不清退")

	// 部分退款后仍有未退净额，继续拒绝并给出剩余金额。
	_, err = orderSvc.AddPayment(ctx, order.ID, PaymentRequest{
		AmountCents: -40000, RefundOfPaymentID: &pay.PaymentID,
	}, "s1b_ops", "127.0.0.1")
	require.NoError(t, err)
	_, err = orderSvc.Cancel(ctx, order.ID, "园所暂缓采购", "s1b_ops", "127.0.0.1")
	require.Error(t, err)
	var partial *UnrefundedPaymentError
	require.ErrorAs(t, err, &partial)
	assert.EqualValues(t, 60000, partial.AmountCents, "未退款金额按已收净额计算")

	// 退清后净额为 0，允许取消并收回授权。
	_, err = orderSvc.AddPayment(ctx, order.ID, PaymentRequest{
		AmountCents: -60000, RefundOfPaymentID: &pay.PaymentID,
	}, "s1b_ops", "127.0.0.1")
	require.NoError(t, err)
	cancelled, err := orderSvc.Cancel(ctx, order.ID, "全额退款后取消", "s1b_ops", "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusCancelled, cancelled.Status)
	assert.Equal(t, "全额退款后取消", cancelled.CancelReason)
	assert.NotNil(t, cancelled.CancelledAt)
	remaining, err := repository.NewEntitlementRepo(db).ListByOrderID(ctx, order.ID)
	require.NoError(t, err)
	assert.Empty(t, remaining, "取消订单必须收回其产生的授权")

	// 已取消订单不能再登记收款。
	_, err = orderSvc.AddPayment(ctx, order.ID, PaymentRequest{AmountCents: 1000}, "s1b_ops", "127.0.0.1")
	require.Error(t, err)
	assert.Equal(t, "ORDER_CANCELLED", err.Error())
}

// 未收款订单仍可直接取消，行为不变。
func TestCancelGuard_UnpaidOrderStillCancels(t *testing.T) {
	db := openPlatformTestDB(t)
	ctx := context.Background()
	cleanupCancelGuard(t, db)
	t.Cleanup(func() { cleanupCancelGuard(t, db) })

	orderSvc, order, _ := newCancelGuardOrder(t, db, "S1Bcancel-未收款园")

	cancelled, err := orderSvc.Cancel(ctx, order.ID, "未成交", "s1b_ops", "127.0.0.1")
	require.NoError(t, err, "未收款订单可直接取消")
	assert.Equal(t, domain.OrderStatusCancelled, cancelled.Status)
	assert.NotNil(t, cancelled.CancelledAt)
}
