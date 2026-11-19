package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/pkg/requestid"
)

// failingAuditWriter 注入审计写入失败，用于验证强一致操作整体回滚、
// best-effort 操作不被审计故障阻断。
type failingAuditWriter struct{ err error }

func (f failingAuditWriter) Create(_ context.Context, _ *domain.PlatformOperationLog) error {
	return f.err
}

func (f failingAuditWriter) CreateTx(_ context.Context, _ *gorm.DB, _ *domain.PlatformOperationLog) error {
	return f.err
}

// List 让 fake 同时满足 PlatformAuditStore（PlatformService 的构造参数）。
func (f failingAuditWriter) List(_ context.Context, _ repository.PlatformLogFilter) ([]domain.PlatformOperationLog, int64, error) {
	return nil, 0, f.err
}

var errAuditDown = errors.New("audit store unavailable")

// newPlatformServiceWithAudit 与 newPlatformTestService 同构，但可注入审计写入器，
// 用于验证强一致操作在审计失败时整体回滚。
func newPlatformServiceWithAudit(t *testing.T, db *gorm.DB, audit PlatformAuditStore) *PlatformService {
	t.Helper()
	return NewPlatformService(
		db,
		repository.NewPlatformAdminRepo(db),
		repository.NewKindergartenRepo(db),
		repository.NewEntitlementRepo(db),
		repository.NewOrderRepo(db),
		repository.NewUserRepo(db),
		repository.NewUserRoleRepo(db),
		repository.NewAdminRepo(db),
		repository.NewPaymentRepo(db),
		repository.NewCommissionRepo(db),
		repository.NewDistributorRepo(db),
		audit,
		nil,
		nil,
		nil,
		nil,
		nil,
		[]byte("test-platform-secret"),
		time.Hour,
		7*24*time.Hour,
	)
}

func newOrderServiceWithAudit(db *gorm.DB, audit PlatformAuditWriter) *OrderService {
	return NewOrderService(
		db, repository.NewOrderRepo(db), repository.NewPaymentRepo(db),
		repository.NewPlanRepo(db), repository.NewKindergartenRepo(db),
		repository.NewEntitlementRepo(db), repository.NewDistributorRepo(db),
		repository.NewCommissionRepo(db), audit, nil,
	)
}

// auditedOrderFixture 建一个园所 + 套餐 + 未确认订单，供强一致/回滚断言复用。
func auditedOrderFixture(ctx context.Context, t *testing.T, db *gorm.DB, orderSvc *OrderService, suffix string) (*domain.Kindergarten, *domain.Order) {
	t.Helper()
	kg := &domain.Kindergarten{Name: "T2A审计测试园" + suffix, Status: domain.KindergartenStatusActive}
	require.NoError(t, repository.NewKindergartenRepo(db).Create(ctx, kg))
	plan := &domain.Plan{
		Code: "t2a_strong" + suffix, Name: "T2A审计套餐", PriceCents: 100000,
		TermMonths: 6, Status: "active",
	}
	planRepo := repository.NewPlanRepo(db)
	require.NoError(t, planRepo.Create(ctx, plan))
	// 挂一个 premium 模块，确认订单才有授权可发——回滚断言才不是空集。
	require.NoError(t, planRepo.ReplaceModules(ctx, plan.ID, []string{"growth"}))
	order, err := orderSvc.Create(ctx, CreateOrderRequest{KindergartenID: kg.ID, PlanID: plan.ID}, "t2a_ops", "127.0.0.1")
	require.NoError(t, err)
	return kg, order
}

// 强一致清单（design D6）：订单确认/取消/收款、园所开通/启停都必须在业务事务内写审计；
// 审计失败 → 业务整体回滚。
func TestPlatformAudit_StrongWritesRollBackOnAuditFailure(t *testing.T) {
	db := openPlatformTestDB(t)
	ctx := context.Background()
	cleanupPlatformAudit(t, db)
	t.Cleanup(func() { cleanupPlatformAudit(t, db) })

	orderSvc := newOrderServiceWithAudit(db, failingAuditWriter{err: errAuditDown})
	kg, order := auditedOrderFixture(context.Background(), t, db, orderSvc, "A")

	_, _, err := orderSvc.Confirm(ctx, order.ID, "测试信用开通", "t2a_ops", "127.0.0.1")
	require.ErrorIs(t, err, errAuditDown)
	reloaded, err := repository.NewOrderRepo(db).FindByID(ctx, order.ID)
	require.NoError(t, err)
	assert.Nil(t, reloaded.ConfirmedAt, "审计失败时订单确认必须回滚")
	ents, err := repository.NewEntitlementRepo(db).ListByKindergarten(ctx, kg.ID)
	require.NoError(t, err)
	assert.Empty(t, ents, "审计失败时授权发放必须回滚")

	_, err = orderSvc.AddPayment(ctx, order.ID, PaymentRequest{AmountCents: 1000}, "t2a_ops", "127.0.0.1")
	require.ErrorIs(t, err, errAuditDown)
	payments, err := repository.NewPaymentRepo(db).ListByOrder(ctx, order.ID)
	require.NoError(t, err)
	assert.Empty(t, payments, "审计失败时收款必须回滚")

	_, err = orderSvc.Cancel(ctx, order.ID, "审计故障", "t2a_ops", "127.0.0.1")
	require.ErrorIs(t, err, errAuditDown)
	reloaded, err = repository.NewOrderRepo(db).FindByID(ctx, order.ID)
	require.NoError(t, err)
	assert.Nil(t, reloaded.CancelledAt, "审计失败时订单取消必须回滚")

	// 园所开通：审计失败时园所与园长账号都不能留下。
	platformSvc := newPlatformServiceWithAudit(t, db, failingAuditWriter{err: errAuditDown})
	phone := "19900002222"
	_, err = platformSvc.Onboard(ctx, OnboardRequest{
		Name: "T2A审计测试园开通", Principal: "王园长", Phone: phone, AdminPassword: "InitPass1",
	}, "t2a_ops", "127.0.0.1")
	require.ErrorIs(t, err, errAuditDown)
	created, err := repository.NewUserRepo(db).FindByPhone(ctx, phone)
	require.NoError(t, err)
	assert.Nil(t, created, "审计失败时园长账号必须回滚")

	// 园所启停：审计失败时状态不变。
	assert.ErrorIs(t, platformSvc.SetKindergartenStatus(ctx, kg.ID, domain.KindergartenStatusSuspended, "t2a_ops", "127.0.0.1"), errAuditDown)
	unchanged, err := repository.NewKindergartenRepo(db).FindByID(ctx, kg.ID)
	require.NoError(t, err)
	assert.NotEqual(t, domain.KindergartenStatusSuspended, unchanged.Status, "审计失败时状态变更必须回滚")
}

// best-effort 清单（design D6）：套餐 / 分销商目录变更与订单创建的审计失败
// 只记错误日志，不阻断业务。
func TestPlatformAudit_BestEffortDoesNotBlockBusiness(t *testing.T) {
	db := openPlatformTestDB(t)
	ctx := context.Background()
	cleanupPlatformAudit(t, db)
	t.Cleanup(func() { cleanupPlatformAudit(t, db) })

	failing := failingAuditWriter{err: errAuditDown}
	planSvc := NewPlanService(repository.NewPlanRepo(db), failing)
	plan, err := planSvc.Create(ctx, PlanRequest{
		Code: "t2a_best", Name: "T2A best-effort 套餐", PriceCents: 100000, TermMonths: 6,
		ModuleCodes: []string{"base"},
	}, "t2a_ops", "127.0.0.1")
	require.NoError(t, err, "套餐创建不能被审计故障阻断")
	require.NotZero(t, plan.ID)

	require.NoError(t, planSvc.Update(ctx, plan.ID, PlanRequest{Name: "T2A best-effort 套餐改名", PriceCents: 120000}, "t2a_ops", "127.0.0.1"),
		"套餐更新不能被审计故障阻断")
	require.NoError(t, planSvc.ReplaceTiers(ctx, plan.ID, []domain.PlanTier{
		{Tier: 1, MinStudents: 0, MaxStudents: 0, PriceMultiplier: 100},
	}, "t2a_ops", "127.0.0.1"), "档位保存不能被审计故障阻断")

	distSvc := NewDistributorService(
		repository.NewDistributorRepo(db), repository.NewCommissionRepo(db),
		repository.NewOrderRepo(db), repository.NewKindergartenRepo(db), failing,
	)
	dist, err := distSvc.CreateDistributor(ctx, DistributorRequest{Name: "T2A分销商best"}, "t2a_ops", "127.0.0.1")
	require.NoError(t, err, "分销商创建不能被审计故障阻断")
	require.NoError(t, distSvc.UpdateDistributor(ctx, dist.ID, DistributorRequest{Name: "T2A分销商best改"}, "t2a_ops", "127.0.0.1"),
		"分销商更新不能被审计故障阻断")

	kg := &domain.Kindergarten{Name: "T2A审计测试园C", Status: domain.KindergartenStatusActive}
	require.NoError(t, repository.NewKindergartenRepo(db).Create(ctx, kg))
	orderSvc := newOrderServiceWithAudit(db, failing)
	_, err = orderSvc.Create(ctx, CreateOrderRequest{KindergartenID: kg.ID, PlanID: plan.ID}, "t2a_ops", "127.0.0.1")
	require.NoError(t, err, "订单创建不能被审计故障阻断")
}

// 强一致记录必须携带目标园所与请求链路标识。
func TestPlatformAudit_StrongRowsCarryKindergartenAndRequestID(t *testing.T) {
	db := openPlatformTestDB(t)
	cleanupPlatformAudit(t, db)
	t.Cleanup(func() { cleanupPlatformAudit(t, db) })

	ctx := requestid.With(context.Background(), "req-t2a-strong")
	audit := repository.NewPlatformLogRepo(db)
	orderSvc := newOrderServiceWithAudit(db, audit)
	kg, order := auditedOrderFixture(ctx, t, db, orderSvc, "B")

	_, _, err := orderSvc.Confirm(ctx, order.ID, "测试信用开通", "t2a_ops", "127.0.0.1")
	require.NoError(t, err)
	_, err = orderSvc.AddPayment(ctx, order.ID, PaymentRequest{AmountCents: 1000}, "t2a_ops", "127.0.0.1")
	require.NoError(t, err)

	logs, total, err := audit.List(ctx, repository.PlatformLogFilter{KindergartenID: kg.ID, Module: "order"})
	require.NoError(t, err)
	assert.EqualValues(t, 2, total, "create（best-effort）与 confirm（同事务）都记 order 模块")
	require.Len(t, logs, 2)
	actions := make([]string, 0, len(logs))
	for _, l := range logs {
		assert.Equal(t, "req-t2a-strong", l.RequestID)
		assert.Equal(t, kg.ID, l.KindergartenID)
		actions = append(actions, l.Action)
	}
	assert.ElementsMatch(t, []string{"create", "confirm"}, actions)

	payLogs, total, err := audit.List(ctx, repository.PlatformLogFilter{KindergartenID: kg.ID, Module: "payment"})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, payLogs, 1)
	assert.Equal(t, "req-t2a-strong", payLogs[0].RequestID)
	assert.Equal(t, kg.ID, payLogs[0].KindergartenID)

	// 园所启停同样带目标园所与链路标识。
	platformSvc := newPlatformServiceWithAudit(t, db, audit)
	require.NoError(t, platformSvc.SetKindergartenStatus(ctx, kg.ID, domain.KindergartenStatusSuspended, "t2a_ops", "127.0.0.1"))
	kgLogs, total, err := audit.List(ctx, repository.PlatformLogFilter{KindergartenID: kg.ID, Module: "kindergarten"})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, kgLogs, 1)
	assert.Equal(t, "req-t2a-strong", kgLogs[0].RequestID)
	assert.Equal(t, "update_status", kgLogs[0].Action)
}

func cleanupPlatformAudit(t *testing.T, db *gorm.DB) {
	t.Helper()
	del := func(sql string, args ...any) {
		if err := db.Exec(sql, args...).Error; err != nil {
			t.Logf("cleanup %q: %v", sql, err)
		}
	}
	del("DELETE FROM platform_operation_logs WHERE operator LIKE 't2a_%'")
	del("DELETE FROM commission_ledger WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE 'T2A审计测试%')")
	del("DELETE FROM payments WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE 'T2A审计测试%')")
	del("DELETE FROM entitlements WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE 'T2A审计测试%')")
	del("DELETE FROM orders WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE 'T2A审计测试%')")
	del("DELETE FROM students WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE 'T2A审计测试%')")
	del("DELETE FROM classes WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE 'T2A审计测试%')")
	del("DELETE FROM user_admin_roles WHERE user_id IN (SELECT id FROM users WHERE phone = '19900002222')")
	del("DELETE FROM admin_role_permissions WHERE role_id IN (SELECT id FROM admin_roles WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE 'T2A审计测试%'))")
	del("DELETE FROM admin_roles WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE 'T2A审计测试%')")
	del("DELETE FROM user_roles WHERE user_id IN (SELECT id FROM users WHERE phone = '19900002222')")
	del("DELETE FROM users WHERE phone = '19900002222'")
	del("DELETE FROM kindergartens WHERE name LIKE 'T2A审计测试%'")
	del("DELETE FROM plans WHERE code LIKE 't2a_%'")
	del("DELETE FROM distributors WHERE name LIKE 'T2A分销%'")
}

func TestPlatformAudit_QueryFiltersAndRequestID(t *testing.T) {
	db := openPlatformTestDB(t)
	cleanupPlatformAudit(t, db)
	t.Cleanup(func() { cleanupPlatformAudit(t, db) })

	repo := repository.NewPlatformLogRepo(db)
	svc := newPlatformTestService(t, db)
	ctx := requestid.With(context.Background(), "req-t2a-query")

	// 固定在一个远离历史数据的未来窗口，避免 dev 库既有审计行污染时间筛选；
	// GORM 会自动把 created_at 设成当前时间，所以插入后必须逐行改写。
	now := time.Date(2035, 1, 1, 12, 0, 0, 0, time.Local)
	rows := []domain.PlatformOperationLog{
		{Operator: "t2a_a", Module: "order", Action: "confirm", TargetType: "order", TargetID: 1, KindergartenID: 910001, Detail: "首单", IP: "127.0.0.1"},
		{Operator: "t2a_b", Module: "t2a_plan", Action: "create", TargetType: "plan", TargetID: 2, KindergartenID: 0, Detail: "套餐", IP: "127.0.0.1"},
		{Operator: "t2a_a", Module: "t2a_plan", Action: "update", TargetType: "plan", TargetID: 2, KindergartenID: 0, Detail: "调价", IP: "127.0.0.1"},
	}
	for i := range rows {
		require.NoError(t, repo.Create(ctx, &rows[i]))
		assert.Equal(t, "req-t2a-query", rows[i].RequestID, "写入时必须从 context 落 request_id")
	}
	// 把首行时间推前一天，验证时间范围筛选；其余两行固定在 now。
	for i := range rows {
		created := now
		if i == 0 {
			created = now.AddDate(0, 0, -1)
		}
		require.NoError(t, db.Model(&domain.PlatformOperationLog{}).Where("id = ?", rows[i].ID).
			UpdateColumn("created_at", created).Error)
	}

	from := now.AddDate(0, 0, -2)
	to := now.AddDate(0, 0, 1)
	cases := []struct {
		name   string
		filter repository.PlatformLogFilter
		want   int64
	}{
		{"按园所", repository.PlatformLogFilter{KindergartenID: 910001}, 1},
		{"按操作人", repository.PlatformLogFilter{Operator: "t2a_a"}, 2},
		{"按模块", repository.PlatformLogFilter{Module: "t2a_plan"}, 2},
		{"按时间范围", repository.PlatformLogFilter{From: &from, To: &to}, 3},
		{"组合筛选", repository.PlatformLogFilter{Operator: "t2a_a", Module: "t2a_plan"}, 1},
		{"排除时间范围", repository.PlatformLogFilter{From: &now}, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			list, total, err := svc.ListLogs(ctx, tc.filter)
			require.NoError(t, err)
			assert.EqualValues(t, tc.want, total)
			assert.Len(t, list, int(tc.want))
		})
	}

	// 分页：page_size=1 第二页只能是另一条。
	page, total, err := svc.ListLogs(ctx, repository.PlatformLogFilter{Operator: "t2a_a", Page: 2, PageSize: 1})
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	require.Len(t, page, 1)
}
