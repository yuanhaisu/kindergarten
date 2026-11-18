package service

import (
	"context"
	"encoding/json"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

func newBillingFlowServices(t *testing.T, db *gorm.DB) (*PlanService, *OrderService, *EntitlementService) {
	t.Helper()
	planSvc := NewPlanService(repository.NewPlanRepo(db), repository.NewPlatformLogRepo(db))
	orderSvc := NewOrderService(
		db, repository.NewOrderRepo(db), repository.NewPaymentRepo(db),
		repository.NewPlanRepo(db), repository.NewKindergartenRepo(db),
		repository.NewEntitlementRepo(db), repository.NewDistributorRepo(db),
		repository.NewCommissionRepo(db), repository.NewPlatformLogRepo(db), nil,
	)
	entSvc := NewEntitlementService(repository.NewEntitlementRepo(db), repository.NewOrderRepo(db), repository.NewKindergartenRepo(db), nil, true)
	return planSvc, orderSvc, entSvc
}

func cleanupBillingFlow(t *testing.T, db *gorm.DB) {
	t.Helper()
	del := func(sql string, args ...any) {
		if err := db.Exec(sql, args...).Error; err != nil {
			t.Logf("cleanup %q: %v", sql, err)
		}
	}
	del("DELETE FROM commission_ledger WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE 'S1B订单测试%')")
	del("DELETE FROM payments WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE 'S1B订单测试%')")
	del("DELETE FROM entitlements WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE 'S1B订单测试%')")
	del("DELETE FROM orders WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE 'S1B订单测试%')")
	del("DELETE FROM students WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE 'S1B订单测试%')")
	del("DELETE FROM classes WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE 'S1B订单测试%')")
	del("DELETE FROM kindergartens WHERE name LIKE 'S1B订单测试%'")
	del("DELETE FROM plans WHERE code LIKE 's1b_%'")
	del("DELETE FROM distributors WHERE name LIKE 'S1B分销%'")
}

func TestBillingFlow_SnapshotPaymentsAndCancel(t *testing.T) {
	db := openPlatformTestDB(t)
	planSvc, orderSvc, _ := newBillingFlowServices(t, db)
	ctx := context.Background()
	cleanupBillingFlow(t, db)
	t.Cleanup(func() { cleanupBillingFlow(t, db) })

	kg := &domain.Kindergarten{Name: "S1B订单测试园", Principal: "王园长", Status: domain.KindergartenStatusActive}
	require.NoError(t, repository.NewKindergartenRepo(db).Create(ctx, kg))

	plan, err := planSvc.Create(ctx, PlanRequest{
		Code: "s1b_flagship", Name: "S1B旗舰", Tier: 2, PriceCents: 100000, TermMonths: 6,
		ModuleCodes: []string{"base", "growth", "diet"},
	}, "s1b_ops", "127.0.0.1")
	require.NoError(t, err)

	dist := &domain.Distributor{Name: "S1B分销商", CommissionRate: 10, Status: 1}
	require.NoError(t, repository.NewDistributorRepo(db).Create(ctx, dist))

	order, err := orderSvc.Create(ctx, CreateOrderRequest{
		KindergartenID: kg.ID, PlanID: plan.ID, DistributorID: &dist.ID,
	}, "s1b_ops", "127.0.0.1")
	require.NoError(t, err)
	assert.EqualValues(t, 100000, order.AmountDueCents)
	assert.Equal(t, domain.OrderStatusUnpaid, order.Status)
	assert.Equal(t, 10, order.CommissionRate, "佣金比例在成交时冻结")

	// 目录调价不影响已创建订单
	require.NoError(t, planSvc.Update(ctx, plan.ID, PlanRequest{
		Name: "S1B旗舰改名", PriceCents: 200000, ModuleCodes: []string{"base"},
	}, "s1b_ops", "127.0.0.1"))
	again, err := orderSvc.Detail(ctx, order.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 100000, again.AmountDueCents, "订单金额必须来自快照")
	assert.Contains(t, again.PlanSnapshot.ModuleCodes, "growth", "模块清单必须来自快照")

	confirmed, ents, err := orderSvc.Confirm(ctx, order.ID, "测试信用开通", "s1b_ops", "127.0.0.1")
	require.NoError(t, err)
	assert.Len(t, ents, 2, "只给 premium 模块发授权，base 不重复发放")
	assert.NotNil(t, confirmed.ConfirmedAt)

	// 首付 → 部分收款，并按实收生成佣金
	res, err := orderSvc.AddPayment(ctx, order.ID, PaymentRequest{
		AmountCents: 50000, Method: domain.PaymentMethodBankTransfer,
	}, "s1b_ops", "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusPartial, res.OrderStatus)
	assert.EqualValues(t, 50000, res.AmountPaidCents)
	firstPaymentID := res.PaymentID

	var commissions []domain.CommissionLedger
	require.NoError(t, db.Where("order_id = ?", order.ID).Find(&commissions).Error)
	require.Len(t, commissions, 1)
	assert.EqualValues(t, 5000, commissions[0].AmountCents, "佣金按实收 10% 计算")
	assert.Equal(t, domain.CommissionStatusPending, commissions[0].Status)
	assert.True(t, commissions[0].EligibleAt.After(time.Now().AddDate(0, 0, 25)), "30 天后才可结算")

	// 收清
	res, err = orderSvc.AddPayment(ctx, order.ID, PaymentRequest{AmountCents: 50000}, "s1b_ops", "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusPaid, res.OrderStatus)
	secondPaymentID := res.PaymentID

	// 退款以负数登记，冲回被退收款对应的佣金
	res, err = orderSvc.AddPayment(ctx, order.ID, PaymentRequest{
		AmountCents: -50000, RefundOfPaymentID: &secondPaymentID,
	}, "s1b_ops", "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusPartial, res.OrderStatus)
	var reversal domain.CommissionLedger
	require.NoError(t, db.Where("order_id = ? AND reversal_of IS NOT NULL", order.ID).First(&reversal).Error)
	assert.Equal(t, domain.CommissionStatusReversed, reversal.Status)
	assert.EqualValues(t, -5000, reversal.AmountCents)

	// 退清剩余收款后净已收归零，才允许取消（未退款收款会阻断取消）。
	res, err = orderSvc.AddPayment(ctx, order.ID, PaymentRequest{
		AmountCents: -50000, RefundOfPaymentID: &firstPaymentID,
	}, "s1b_ops", "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusUnpaid, res.OrderStatus)
	assert.EqualValues(t, 0, res.AmountPaidCents)

	// 取消订单收回未到期授权
	_, err = orderSvc.Cancel(ctx, order.ID, "暂停采购", "s1b_ops", "127.0.0.1")
	require.NoError(t, err)
	remaining, err := repository.NewEntitlementRepo(db).ListByOrderID(ctx, order.ID)
	require.NoError(t, err)
	assert.Empty(t, remaining, "取消订单必须收回其产生的授权")
}

func TestEntitlement_DecideGraceAndReadOnly(t *testing.T) {
	db := openPlatformTestDB(t)
	_, _, entSvc := newBillingFlowServices(t, db)
	ctx := context.Background()
	cleanupBillingFlow(t, db)
	t.Cleanup(func() { cleanupBillingFlow(t, db) })

	kg := &domain.Kindergarten{Name: "S1B订单测试园B", Status: domain.KindergartenStatusActive}
	require.NoError(t, repository.NewKindergartenRepo(db).Create(ctx, kg))

	now := time.Now()
	rows := []domain.Entitlement{
		{KindergartenID: kg.ID, ModuleCode: "growth", Source: domain.EntitlementSourceSubscription, StartsAt: now.AddDate(0, -1, 0), ExpiresAt: now.AddDate(0, 1, 0)},
		{KindergartenID: kg.ID, ModuleCode: "diet", Source: domain.EntitlementSourceSubscription, StartsAt: now.AddDate(0, -2, 0), ExpiresAt: now.AddDate(0, 0, -5)},
		{KindergartenID: kg.ID, ModuleCode: "album", Source: domain.EntitlementSourceSubscription, StartsAt: now.AddDate(0, -6, 0), ExpiresAt: now.AddDate(0, 0, -30)},
	}
	require.NoError(t, repository.NewEntitlementRepo(db).CreateBatch(ctx, rows))

	decision, err := entSvc.Decide(ctx, kg.ID, "growth", true)
	require.NoError(t, err)
	assert.True(t, decision.Allowed, "在期模块可写")

	decision, err = entSvc.Decide(ctx, kg.ID, "diet", true)
	require.NoError(t, err)
	assert.True(t, decision.Allowed, "宽限期内仍可写")

	decision, err = entSvc.Decide(ctx, kg.ID, "album", true)
	require.NoError(t, err)
	assert.False(t, decision.Allowed, "超出宽限期的写操作必须拒绝")

	decision, err = entSvc.Decide(ctx, kg.ID, "album", false)
	require.NoError(t, err)
	assert.True(t, decision.Allowed, "降级后仍可读")

	decision, err = entSvc.Decide(ctx, kg.ID, "base", true)
	require.NoError(t, err)
	assert.True(t, decision.Allowed, "base 模块始终可用")

	decision, err = entSvc.Decide(ctx, kg.ID, "growth_report", true)
	require.NoError(t, err)
	assert.False(t, decision.Allowed, "未购买模块不可用")
}

func TestEntitlement_UnionOfOverlappingGrants(t *testing.T) {
	db := openPlatformTestDB(t)
	_, _, entSvc := newBillingFlowServices(t, db)
	ctx := context.Background()
	cleanupBillingFlow(t, db)
	t.Cleanup(func() { cleanupBillingFlow(t, db) })

	kg := &domain.Kindergarten{Name: "S1B订单测试园C", Status: domain.KindergartenStatusActive}
	require.NoError(t, repository.NewKindergartenRepo(db).Create(ctx, kg))
	now := time.Now()
	require.NoError(t, repository.NewEntitlementRepo(db).CreateBatch(ctx, []domain.Entitlement{
		{KindergartenID: kg.ID, ModuleCode: "diet", Source: domain.EntitlementSourceTrial, StartsAt: now.AddDate(0, -1, 0), ExpiresAt: now.AddDate(0, 0, -20)},
		{KindergartenID: kg.ID, ModuleCode: "diet", Source: domain.EntitlementSourceSubscription, StartsAt: now.AddDate(0, -1, 0), ExpiresAt: now.AddDate(0, 2, 0)},
	}))
	decision, err := entSvc.Decide(ctx, kg.ID, "diet", true)
	require.NoError(t, err)
	assert.True(t, decision.Allowed, "同一模块叠加时只要一条在期即可用")
}

// 续费后立即恢复写能力：新授权生效即覆盖旧的降级状态。
func TestEntitlement_RenewalRestoresWrite(t *testing.T) {
	db := openPlatformTestDB(t)
	_, _, entSvc := newBillingFlowServices(t, db)
	ctx := context.Background()
	cleanupBillingFlow(t, db)
	t.Cleanup(func() { cleanupBillingFlow(t, db) })

	kg := &domain.Kindergarten{Name: "S1B订单测试园D", Status: domain.KindergartenStatusActive}
	require.NoError(t, repository.NewKindergartenRepo(db).Create(ctx, kg))
	now := time.Now()
	require.NoError(t, repository.NewEntitlementRepo(db).CreateBatch(ctx, []domain.Entitlement{{
		KindergartenID: kg.ID, ModuleCode: "diet", Source: domain.EntitlementSourceSubscription,
		StartsAt: now.AddDate(0, -8, 0), ExpiresAt: now.AddDate(0, 0, -30),
	}}))

	decision, err := entSvc.Decide(ctx, kg.ID, "diet", true)
	require.NoError(t, err)
	require.False(t, decision.Allowed)

	require.NoError(t, entSvc.Grant(ctx, kg.ID, "diet", now.AddDate(0, 1, 0)))

	decision, err = entSvc.Decide(ctx, kg.ID, "diet", true)
	require.NoError(t, err)
	assert.True(t, decision.Allowed, "续费后必须立即恢复写能力")
}

// 契约字段集：园所端两个授权端点只返回 design.md 列出的字段。
func TestBillingContract_SubscriptionFieldSet(t *testing.T) {
	db := openPlatformTestDB(t)
	_, orderSvc, entSvc := newBillingFlowServices(t, db)
	ctx := context.Background()
	cleanupBillingFlow(t, db)
	t.Cleanup(func() { cleanupBillingFlow(t, db) })

	kg := &domain.Kindergarten{Name: "S1B订单测试园E", Status: domain.KindergartenStatusActive}
	require.NoError(t, repository.NewKindergartenRepo(db).Create(ctx, kg))
	now := time.Now()
	require.NoError(t, repository.NewEntitlementRepo(db).CreateBatch(ctx, []domain.Entitlement{
		{KindergartenID: kg.ID, ModuleCode: "base", Source: domain.EntitlementSourceGift, StartsAt: now.AddDate(-1, 0, 0), ExpiresAt: now.AddDate(10, 0, 0)},
		{KindergartenID: kg.ID, ModuleCode: "growth", Source: domain.EntitlementSourceTrial, StartsAt: now, ExpiresAt: now.AddDate(0, 1, 0)},
	}))

	view, err := entSvc.Subscription(ctx, kg.ID)
	require.NoError(t, err)
	raw, err := json.Marshal(view)
	require.NoError(t, err)
	var top map[string]any
	require.NoError(t, json.Unmarshal(raw, &top))
	assert.Equal(t,
		[]string{"expires_at", "grace_until", "modules", "over_limit", "plan_name", "state", "student_count", "tier_max_students"},
		sortedKeys(top))

	modules := top["modules"].([]any)
	require.Len(t, modules, 2, "同一模块多来源应合并为一行")
	assert.Equal(t,
		[]string{"expires_at", "module_code", "module_name", "source", "starts_at", "state"},
		sortedKeys(modules[0].(map[string]any)))

	// 小程序端点只返回 modules 与 expires_at
	mods, expiresAt, err := entSvc.EntitlementModules(ctx, kg.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{"base", "growth"}, mods)
	require.NotNil(t, expiresAt)

	_ = orderSvc
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// 超档只提示、不阻断：快照记录档位上限，Subscription 据此给出升级线索。
func TestSubscription_OverLimit(t *testing.T) {
	db := openPlatformTestDB(t)
	planSvc, orderSvc, entSvc := newBillingFlowServices(t, db)
	ctx := context.Background()
	cleanupBillingFlow(t, db)
	t.Cleanup(func() { cleanupBillingFlow(t, db) })

	kg := &domain.Kindergarten{Name: "S1B订单测试园G", Status: domain.KindergartenStatusActive}
	require.NoError(t, repository.NewKindergartenRepo(db).Create(ctx, kg))

	plan, err := planSvc.Create(ctx, PlanRequest{
		Code: "s1b_tier", Name: "S1B档位版", Tier: 1, PriceCents: 100000, TermMonths: 6,
		ModuleCodes: []string{"base", "growth"},
	}, "s1b_ops", "127.0.0.1")
	require.NoError(t, err)
	require.NoError(t, planSvc.ReplaceTiers(ctx, plan.ID, []domain.PlanTier{
		{Tier: 1, MinStudents: 0, MaxStudents: 2, PriceMultiplier: 100},
		{Tier: 2, MinStudents: 3, MaxStudents: 0, PriceMultiplier: 150},
	}, "s1b_ops", "127.0.0.1"))

	order, err := orderSvc.Create(ctx, CreateOrderRequest{KindergartenID: kg.ID, PlanID: plan.ID}, "s1b_ops", "127.0.0.1")
	require.NoError(t, err)
	var snap PlanSnapshot
	require.NoError(t, json.Unmarshal([]byte(order.PlanSnapshot), &snap))
	assert.Equal(t, 2, snap.TierMaxStudents, "快照必须记录档位上限")
	_, _, err = orderSvc.Confirm(ctx, order.ID, "测试信用开通", "s1b_ops", "127.0.0.1")
	require.NoError(t, err)

	// 3 名在园幼儿，超过上限 2
	class := &domain.Class{KindergartenID: kg.ID, Name: "S1B超档测试班", Capacity: 30, Status: 1}
	require.NoError(t, db.Create(class).Error)
	for range 3 {
		require.NoError(t, db.Create(&domain.Student{
			KindergartenID: kg.ID, ClassID: class.ID, Name: "S1B超档测试幼儿", Status: 1,
			Birthday: time.Now(), EnrollDate: time.Now(),
		}).Error)
	}

	view, err := entSvc.Subscription(ctx, kg.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 3, view.StudentCount)
	assert.Equal(t, 2, view.TierMaxStudents)
	assert.True(t, view.OverLimit, "超过档位上限必须标记 over_limit")
	assert.Equal(t, "S1B档位版", view.PlanName)
}
