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

// 付款状态机是读时派生的纯函数：逾期只看 pay_due_at，服务期结束日不再参与；
// pay_due_at 为零时回退到 starts_at（design D1）。
func TestDeriveOrderStatus_PayDueAt(t *testing.T) {
	now := time.Date(2026, 11, 18, 12, 0, 0, 0, time.Local)
	cancelledAt := now.Add(-time.Hour)
	base := domain.Order{
		AmountDueCents: 100000,
		StartsAt:       now.AddDate(0, -1, 0),
		ExpiresAt:      now.AddDate(0, 5, 0), // 服务期仍在进行中
	}

	cases := []struct {
		name   string
		mutate func(o *domain.Order)
		paid   int64
		want   string
	}{
		{
			name:   "未过付款到期日且未收款",
			mutate: func(o *domain.Order) { o.PayDueAt = now.AddDate(0, 0, 7) },
			want:   domain.OrderStatusUnpaid,
		},
		{
			name:   "未过付款到期日且部分收款",
			mutate: func(o *domain.Order) { o.PayDueAt = now.AddDate(0, 0, 7) },
			paid:   30000,
			want:   domain.OrderStatusPartial,
		},
		{
			name:   "服务期内但已过付款到期日且未收款（TC-NEW-ORD-0014）",
			mutate: func(o *domain.Order) { o.PayDueAt = now.AddDate(0, 0, -7) },
			want:   domain.OrderStatusExpired,
		},
		{
			name:   "服务期内、部分收款但已过付款到期日（TC-NEW-ORD-0014）",
			mutate: func(o *domain.Order) { o.PayDueAt = now.AddDate(0, 0, -7) },
			paid:   30000,
			want:   domain.OrderStatusExpired,
		},
		{
			name:   "已收清优先于逾期",
			mutate: func(o *domain.Order) { o.PayDueAt = now.AddDate(0, 0, -7) },
			paid:   100000,
			want:   domain.OrderStatusPaid,
		},
		{
			name: "零值到期日回退服务起始日：尚未到期",
			mutate: func(o *domain.Order) {
				o.PayDueAt = time.Time{}
				o.StartsAt = now.AddDate(0, 0, 3)
			},
			want: domain.OrderStatusUnpaid,
		},
		{
			name: "零值到期日回退服务起始日：已过起始日",
			mutate: func(o *domain.Order) {
				o.PayDueAt = time.Time{}
				o.StartsAt = now.AddDate(0, 0, -3)
			},
			want: domain.OrderStatusExpired,
		},
		{
			name: "已取消",
			mutate: func(o *domain.Order) {
				o.PayDueAt = now.AddDate(0, 0, 7)
				o.CancelledAt = &cancelledAt
			},
			paid: 100000,
			want: domain.OrderStatusCancelled,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			order := base
			tc.mutate(&order)
			assert.Equal(t, tc.want, deriveOrderStatus(&order, tc.paid, now))
		})
	}
}

// TC-NEW-ORD-0018：未指定到期日时取服务起始日，且落库非空。
func TestOrderCreate_DefaultsPayDueAtToStartsAt(t *testing.T) {
	db := openPlatformTestDB(t)
	planSvc, orderSvc, _ := newBillingFlowServices(t, db)
	ctx := context.Background()
	cleanupBillingFlow(t, db)
	t.Cleanup(func() { cleanupBillingFlow(t, db) })

	kg := newDueDateTestKindergarten(t, db, "S1B订单测试到期A")
	plan := newDueDateTestPlan(t, planSvc, "s1b_due_default")

	order, err := orderSvc.Create(ctx, CreateOrderRequest{
		KindergartenID: kg.ID, PlanID: plan.ID,
	}, "t3_ops", "127.0.0.1")
	require.NoError(t, err)
	require.False(t, order.PayDueAt.IsZero(), "创建订单不得产生零值付款到期日")
	assert.WithinDuration(t, order.StartsAt, order.PayDueAt, time.Second)

	stored, err := repository.NewOrderRepo(db).FindByID(ctx, order.ID)
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.WithinDuration(t, stored.StartsAt, stored.PayDueAt, time.Second)
}

// 创建请求可显式指定付款到期日，与服务起始日不同。
func TestOrderCreate_ExplicitPayDueAt(t *testing.T) {
	db := openPlatformTestDB(t)
	planSvc, orderSvc, _ := newBillingFlowServices(t, db)
	ctx := context.Background()
	cleanupBillingFlow(t, db)
	t.Cleanup(func() { cleanupBillingFlow(t, db) })

	kg := newDueDateTestKindergarten(t, db, "S1B订单测试到期B")
	plan := newDueDateTestPlan(t, planSvc, "s1b_due_explicit")

	order, err := orderSvc.Create(ctx, CreateOrderRequest{
		KindergartenID: kg.ID, PlanID: plan.ID, PayDueAt: "2026-12-31",
	}, "t3_ops", "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, 2026, order.PayDueAt.Year())
	assert.Equal(t, time.December, order.PayDueAt.Month())
	assert.Equal(t, 31, order.PayDueAt.Day())
	assert.False(t, order.PayDueAt.Equal(order.StartsAt), "显式到期日不应被服务起始日覆盖")

	_, err = orderSvc.Create(ctx, CreateOrderRequest{
		KindergartenID: kg.ID, PlanID: plan.ID, PayDueAt: "2026/12/31",
	}, "t3_ops", "127.0.0.1")
	require.ErrorIs(t, err, ErrInvalidPayDueAt)
}

// TC-NEW-ORD-0014：服务期未结束但已过付款到期日，列表与详情都派生出 expired。
func TestOrderListAndDetail_ExpiredByPayDueAt(t *testing.T) {
	db := openPlatformTestDB(t)
	planSvc, orderSvc, _ := newBillingFlowServices(t, db)
	ctx := context.Background()
	cleanupBillingFlow(t, db)
	t.Cleanup(func() { cleanupBillingFlow(t, db) })

	kg := newDueDateTestKindergarten(t, db, "S1B订单测试到期C")
	plan := newDueDateTestPlan(t, planSvc, "s1b_due_expired")

	due := time.Now().AddDate(0, 0, -3)
	order, err := orderSvc.Create(ctx, CreateOrderRequest{
		KindergartenID: kg.ID, PlanID: plan.ID, PayDueAt: due.Format("2006-01-02"),
	}, "t3_ops", "127.0.0.1")
	require.NoError(t, err)
	require.True(t, order.ExpiresAt.After(time.Now()), "用例前提：服务期尚未结束")

	rows, total, err := orderSvc.List(ctx, repository.OrderFilter{KindergartenID: kg.ID, Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, rows, 1)
	assert.Equal(t, domain.OrderStatusExpired, rows[0].Status, "存储状态仍是 unpaid，展示必须走派生状态")
	assert.Equal(t, order.PayDueAt.Unix(), rows[0].PayDueAt.Unix())

	detail, err := orderSvc.Detail(ctx, order.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusExpired, detail.Status)
	assert.Equal(t, order.PayDueAt.Unix(), detail.PayDueAt.Unix())
}

// TC-NEW-ORD-0016：调整服务期不改变付款到期日，也不改变逾期判定。
func TestOrder_PayDueAtIndependentOfServicePeriod(t *testing.T) {
	db := openPlatformTestDB(t)
	planSvc, orderSvc, _ := newBillingFlowServices(t, db)
	ctx := context.Background()
	cleanupBillingFlow(t, db)
	t.Cleanup(func() { cleanupBillingFlow(t, db) })

	kg := newDueDateTestKindergarten(t, db, "S1B订单测试到期D")
	plan := newDueDateTestPlan(t, planSvc, "s1b_due_period")

	due := time.Now().AddDate(0, 0, -1)
	order, err := orderSvc.Create(ctx, CreateOrderRequest{
		KindergartenID: kg.ID, PlanID: plan.ID, PayDueAt: due.Format("2006-01-02"),
	}, "t3_ops", "127.0.0.1")
	require.NoError(t, err)

	newExpires := order.ExpiresAt.AddDate(0, 3, 0)
	require.NoError(t, db.Model(&domain.Order{}).Where("id = ?", order.ID).
		Update("expires_at", newExpires).Error)

	fresh, err := repository.NewOrderRepo(db).FindByID(ctx, order.ID)
	require.NoError(t, err)
	require.NotNil(t, fresh)
	assert.Equal(t, order.PayDueAt.Unix(), fresh.PayDueAt.Unix(), "服务期变更不得改写付款到期日")
	assert.Equal(t, newExpires.Unix(), fresh.ExpiresAt.Unix())

	rows, _, err := orderSvc.List(ctx, repository.OrderFilter{KindergartenID: kg.ID, Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, domain.OrderStatusExpired, rows[0].Status)
}

func newDueDateTestKindergarten(t *testing.T, db *gorm.DB, name string) *domain.Kindergarten {
	t.Helper()
	kg := &domain.Kindergarten{Name: name, Principal: "测试园长", Status: domain.KindergartenStatusActive}
	require.NoError(t, repository.NewKindergartenRepo(db).Create(context.Background(), kg))
	return kg
}

func newDueDateTestPlan(t *testing.T, planSvc *PlanService, code string) *domain.Plan {
	t.Helper()
	plan, err := planSvc.Create(context.Background(), PlanRequest{
		Code: code, Name: "T3到期测试套餐", Tier: 1, PriceCents: 100000, TermMonths: 6,
		ModuleCodes: []string{"base", "growth"},
	}, "t3_ops", "127.0.0.1")
	require.NoError(t, err)
	return plan
}
