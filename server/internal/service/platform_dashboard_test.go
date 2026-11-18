package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// 平台看板测试夹具统一用 T8 前缀，清理一律硬删：软删行会占用唯一索引，
// 重跑时撞键（见 MEMORY: soft-delete-unique-index-test-trap）。
const dashboardTestPrefix = "T8看板测试"

func createDashboardKindergarten(t *testing.T, db *gorm.DB, suffix string) *domain.Kindergarten {
	t.Helper()
	kg := &domain.Kindergarten{
		Name: dashboardTestPrefix + suffix, Principal: "测试园长",
		Status: domain.KindergartenStatusActive,
	}
	require.NoError(t, db.Create(kg).Error)
	return kg
}

func createDashboardEntitlement(t *testing.T, db *gorm.DB, kgID uint64, module string, expiresIn time.Duration) *domain.Entitlement {
	t.Helper()
	now := time.Now()
	e := &domain.Entitlement{
		KindergartenID: kgID, ModuleCode: module, Source: domain.EntitlementSourceTrial,
		StartsAt: now.Add(-24 * time.Hour), ExpiresAt: now.Add(expiresIn),
	}
	require.NoError(t, db.Create(e).Error)
	return e
}

func createDashboardOrder(t *testing.T, db *gorm.DB, kgID uint64, tag string, amountDue int64, status string) *domain.Order {
	t.Helper()
	now := time.Now()
	o := &domain.Order{
		// T8DASH 前缀供清理定位；后缀取纳秒保证重跑不撞唯一键。
		OrderNo:        fmt.Sprintf("T8DASH%s%d", tag, now.UnixNano()),
		KindergartenID: kgID, PlanID: 0, PlanSnapshot: "{}", StudentTier: 1,
		AmountDueCents: amountDue, Status: status,
		StartsAt: now, ExpiresAt: now.AddDate(0, 1, 0),
	}
	require.NoError(t, db.Create(o).Error)
	return o
}

func createDashboardPayment(t *testing.T, db *gorm.DB, order *domain.Order, amount int64, receivedAt time.Time) *domain.Payment {
	t.Helper()
	p := &domain.Payment{
		OrderID: order.ID, KindergartenID: order.KindergartenID,
		AmountCents: amount, Method: "bank_transfer", ReceivedAt: receivedAt,
	}
	require.NoError(t, db.Create(p).Error)
	return p
}

func createDashboardDistributor(t *testing.T, db *gorm.DB) *domain.Distributor {
	t.Helper()
	d := &domain.Distributor{
		Name:           fmt.Sprintf("%s分销%d", dashboardTestPrefix, time.Now().UnixNano()),
		CommissionRate: 10, Status: 1,
	}
	require.NoError(t, db.Create(d).Error)
	return d
}

func createDashboardCommission(
	t *testing.T, db *gorm.DB, order *domain.Order, payment *domain.Payment,
	dist *domain.Distributor, amount int64, eligibleAt time.Time,
) *domain.CommissionLedger {
	t.Helper()
	c := &domain.CommissionLedger{
		OrderID: order.ID, PaymentID: payment.ID, DistributorID: dist.ID,
		KindergartenID: order.KindergartenID, BaseAmountCents: amount * 10, Rate: 10,
		AmountCents: amount, Status: domain.CommissionStatusPending, EligibleAt: eligibleAt,
	}
	require.NoError(t, db.Create(c).Error)
	return c
}

func cleanupDashboardTestData(t *testing.T, db *gorm.DB) {
	t.Helper()
	stmts := []string{
		"DELETE FROM commission_ledger WHERE distributor_id IN (SELECT id FROM distributors WHERE name LIKE '" + dashboardTestPrefix + "%')",
		"DELETE FROM payments WHERE order_id IN (SELECT id FROM orders WHERE order_no LIKE 'T8DASH%')",
		"DELETE FROM orders WHERE order_no LIKE 'T8DASH%'",
		"DELETE FROM entitlements WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE '" + dashboardTestPrefix + "%')",
		"DELETE FROM kindergartens WHERE name LIKE '" + dashboardTestPrefix + "%'",
		"DELETE FROM distributors WHERE name LIKE '" + dashboardTestPrefix + "%'",
	}
	for _, sql := range stmts {
		if err := db.Exec(sql).Error; err != nil {
			t.Logf("cleanup %q: %v", sql, err)
		}
	}
}

func dashboardKindergartensIDs(rows []PlatformExpiringKindergarten) []uint64 {
	ids := make([]uint64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}

// TC-NEW-PLT-0013：概览覆盖全部园所，而不是当前选中的某一所；
// 7 天内到期待办只列最早到期落在 7 天内的在营园所。
func TestPlatformService_Dashboard_CoversAllKindergartens(t *testing.T) {
	db := openPlatformTestDB(t)
	svc := newPlatformTestService(t, db)
	ctx := context.Background()
	cleanupDashboardTestData(t, db)
	t.Cleanup(func() { cleanupDashboardTestData(t, db) })

	// 先取基线：开发库里已有其他园所，指标必须覆盖它们，断言增量而非绝对值。
	before, err := svc.Dashboard(ctx)
	require.NoError(t, err)

	a := createDashboardKindergarten(t, db, "甲园")
	b := createDashboardKindergarten(t, db, "乙园")
	createDashboardEntitlement(t, db, a.ID, "growth", 5*24*time.Hour)
	createDashboardEntitlement(t, db, b.ID, "diet", 20*24*time.Hour)

	after, err := svc.Dashboard(ctx)
	require.NoError(t, err)

	assert.EqualValues(t, before.KindergartenTotal+2, after.KindergartenTotal, "园所总数覆盖全部园所")
	assert.EqualValues(t, before.KindergartenActive+2, after.KindergartenActive, "两所新园均为在营")
	assert.EqualValues(t, before.KindergartenExpiring30d+2, after.KindergartenExpiring30d,
		"30 天内到期按在营且未删除授权计数")

	ids := dashboardKindergartensIDs(after.ExpiringKindergartens)
	assert.Contains(t, ids, a.ID, "5 天后到期的园所进入 7 天待办")
	assert.NotContains(t, ids, b.ID, "20 天后到期的园所不进入 7 天待办")
}

// TC-NEW-PLT-0021：概览中的本月实收、未收余额与收款/订单明细逐笔核对一致。
// 本月实收含负数退款；未收余额只算未取消订单的 max(应收-已收, 0)。
func TestPlatformService_Dashboard_AmountsMatchDetails(t *testing.T) {
	db := openPlatformTestDB(t)
	svc := newPlatformTestService(t, db)
	ctx := context.Background()
	cleanupDashboardTestData(t, db)
	t.Cleanup(func() { cleanupDashboardTestData(t, db) })

	before, err := svc.Dashboard(ctx)
	require.NoError(t, err)

	kg := createDashboardKindergarten(t, db, "金额园")
	now := time.Now()

	// 订单 A：应收 1000.00，本月收 300.00 → 未收 700.00。
	orderA := createDashboardOrder(t, db, kg.ID, "A", 100000, domain.OrderStatusPartial)
	createDashboardPayment(t, db, orderA, 30000, now)
	// 订单 B：应收 500.00，本月收 500.00 后退款 100.00 → 未收 100.00。
	orderB := createDashboardOrder(t, db, kg.ID, "B", 50000, domain.OrderStatusPaid)
	createDashboardPayment(t, db, orderB, 50000, now)
	createDashboardPayment(t, db, orderB, -10000, now)
	// 已取消订单：应收 200.00，不计入未收余额。
	createDashboardOrder(t, db, kg.ID, "C", 20000, domain.OrderStatusCancelled)

	after, err := svc.Dashboard(ctx)
	require.NoError(t, err)

	assert.EqualValues(t, before.ReceivedThisMonthCents+70000, after.ReceivedThisMonthCents,
		"本月实收 = 300.00 + 500.00 - 100.00（含负数退款）")
	assert.EqualValues(t, before.OutstandingCents+80000, after.OutstandingCents,
		"未收余额 = 700.00 + 100.00，已取消订单不计")
}

func findSettleableCommission(rows []PlatformSettleableCommission, distributorID uint64) *PlatformSettleableCommission {
	for i := range rows {
		if rows[i].DistributorID == distributorID {
			return &rows[i]
		}
	}
	return nil
}

// TC-NEW-PLT-0025：佣金到账满 30 天且期间无退款，即使平台方从未打开过佣金台账页，
// 看板也必须先 PromoteEligible 再聚合，把它计入待结算佣金。
func TestPlatformService_Dashboard_PromotesEligibleCommissions(t *testing.T) {
	db := openPlatformTestDB(t)
	svc := newPlatformTestService(t, db)
	ctx := context.Background()
	cleanupDashboardTestData(t, db)
	t.Cleanup(func() { cleanupDashboardTestData(t, db) })

	// 先取基线（顺带把既有满 30 天的佣金推进掉），保证增量只来自本用例。
	before, err := svc.Dashboard(ctx)
	require.NoError(t, err)

	kg := createDashboardKindergarten(t, db, "佣金园")
	dist := createDashboardDistributor(t, db)
	now := time.Now()
	order := createDashboardOrder(t, db, kg.ID, "K", 100000, domain.OrderStatusPaid)
	payment := createDashboardPayment(t, db, order, 100000, now.Add(-31*24*time.Hour))
	commission := createDashboardCommission(t, db, order, payment, dist, 10000, now.Add(-31*24*time.Hour))

	// 这里刻意不调用 DistributorService.ListCommissions：台账页未被打开过。
	after, err := svc.Dashboard(ctx)
	require.NoError(t, err)

	assert.EqualValues(t, before.CommissionSettleableCents+10000, after.CommissionSettleableCents,
		"满 30 天的佣金在聚合前被推进为可结算")

	row := findSettleableCommission(after.SettleableCommissions, dist.ID)
	require.NotNil(t, row, "可结算佣金待办按分销商聚合")
	assert.EqualValues(t, 10000, row.AmountCents)
	assert.EqualValues(t, 1, row.CommissionCount)
	assert.Equal(t, dist.Name, row.DistributorName)

	stored, err := repository.NewCommissionRepo(db).FindByID(ctx, commission.ID)
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.Equal(t, domain.CommissionStatusSettleable, stored.Status)
}
