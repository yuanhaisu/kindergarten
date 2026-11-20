package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/domain"
)

// 订单列表的公开行为：一次聚合已收金额（排除软删收款），
// 状态筛选按付款到期日翻译，时间范围按订单创建时间。
func TestOrderRepo_ListAggregatesPaidAndFilters(t *testing.T) {
	db := openBillingRepoTestDB(t)
	ctx := context.Background()
	prefix := "T3订单列表测试园"

	cleanup := func() {
		billingHardDelete(t, db, "DELETE FROM payments WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE ?)", prefix+"%")
		billingHardDelete(t, db, "DELETE FROM orders WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE ?)", prefix+"%")
		billingHardDelete(t, db, "DELETE FROM kindergartens WHERE name LIKE ?", prefix+"%")
	}
	cleanup()
	t.Cleanup(cleanup)

	kg := newTestKindergarten(prefix)
	require.NoError(t, NewKindergartenRepo(db).Create(ctx, kg))
	orderRepo := NewOrderRepo(db)
	payRepo := NewPaymentRepo(db)

	start := time.Date(2026, 10, 1, 10, 0, 0, 0, time.Local)
	duePast := time.Date(2026, 10, 1, 0, 0, 0, 0, time.Local)
	dueFuture := time.Date(2027, 1, 1, 0, 0, 0, 0, time.Local)

	type orderSeed struct {
		no      string
		status  string
		due     time.Time
		created time.Time
		paid    []int64 // 正数为收款，负数通过同方法落库不影响本用例
	}
	seeds := []orderSeed{
		{no: "T3-A", status: domain.OrderStatusUnpaid, due: dueFuture, created: time.Date(2026, 10, 1, 9, 0, 0, 0, time.Local), paid: []int64{30000}},
		{no: "T3-B", status: domain.OrderStatusUnpaid, due: duePast, created: time.Date(2026, 11, 1, 9, 0, 0, 0, time.Local)},
		{no: "T3-C", status: domain.OrderStatusPartial, due: dueFuture, created: time.Date(2026, 11, 10, 9, 0, 0, 0, time.Local), paid: []int64{100000}},
		{no: "T3-D", status: domain.OrderStatusCancelled, due: duePast, created: time.Date(2026, 11, 15, 9, 0, 0, 0, time.Local)},
		{no: "T3-E", status: domain.OrderStatusUnpaid, due: duePast, created: time.Date(2026, 10, 20, 9, 0, 0, 0, time.Local)},
		{no: "T3-F", status: domain.OrderStatusPaid, due: duePast, created: time.Date(2026, 11, 5, 9, 0, 0, 0, time.Local), paid: []int64{100000}},
		// Confirm/AddPayment 会把读时派生的状态写回 status，存储层可能已有 expired。
		{no: "T3-G", status: domain.OrderStatusExpired, due: duePast, created: time.Date(2026, 10, 25, 9, 0, 0, 0, time.Local)},
	}

	orders := make(map[string]*domain.Order, len(seeds))
	for _, s := range seeds {
		o := &domain.Order{
			OrderNo: s.no, KindergartenID: kg.ID, PlanID: 1, PlanSnapshot: "{}",
			InvoiceInfo:    "{}",
			AmountDueCents: 100000, Status: s.status,
			StartsAt: start, ExpiresAt: start.AddDate(0, 6, 0), PayDueAt: s.due,
		}
		require.NoError(t, orderRepo.Create(ctx, o))
		// 固化创建时间，避免用例依赖测试执行时刻。
		require.NoError(t, db.Model(&domain.Order{}).Where("id = ?", o.ID).UpdateColumn("created_at", s.created).Error)
		orders[s.no] = o
		for _, amount := range s.paid {
			require.NoError(t, payRepo.Create(ctx, &domain.Payment{
				OrderID: o.ID, KindergartenID: kg.ID, AmountCents: amount,
				Method: domain.PaymentMethodBankTransfer, ReceivedAt: s.created.Add(time.Hour),
			}))
		}
	}

	// 软删收款不计入已收金额。
	softDeleted := &domain.Payment{
		OrderID: orders["T3-A"].ID, KindergartenID: kg.ID, AmountCents: 50000,
		Method: domain.PaymentMethodBankTransfer, ReceivedAt: start,
	}
	require.NoError(t, payRepo.Create(ctx, softDeleted))
	require.NoError(t, db.Delete(&domain.Payment{}, softDeleted.ID).Error)

	nos := func(rows []OrderWithPaid) []string {
		out := make([]string, 0, len(rows))
		for _, r := range rows {
			out = append(out, r.OrderNo)
		}
		return out
	}

	all, total, err := orderRepo.List(ctx, OrderFilter{KindergartenID: kg.ID, Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.EqualValues(t, len(seeds), total, "JOIN 聚合不得放大总数")
	require.Len(t, all, len(seeds))
	paidByNo := make(map[string]int64, len(all))
	for _, r := range all {
		paidByNo[r.OrderNo] = r.PaidCents
	}
	assert.EqualValues(t, 30000, paidByNo["T3-A"], "软删收款必须排除在已收金额之外")
	assert.EqualValues(t, 0, paidByNo["T3-B"])

	expired, total, err := orderRepo.List(ctx, OrderFilter{KindergartenID: kg.ID, Status: "expired", Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.EqualValues(t, 3, total)
	assert.ElementsMatch(t, []string{"T3-B", "T3-E", "T3-G"}, nos(expired), "cancelled 与未过到期日的不应判为 expired；已落库的 expired 不能被漏掉")

	unpaid, total, err := orderRepo.List(ctx, OrderFilter{KindergartenID: kg.ID, Status: "unpaid", Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	assert.Equal(t, []string{"T3-A"}, nos(unpaid), "已过付款到期日的订单不再算 unpaid")

	partial, total, err := orderRepo.List(ctx, OrderFilter{KindergartenID: kg.ID, Status: "partial", Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	assert.Equal(t, []string{"T3-C"}, nos(partial))

	paid, total, err := orderRepo.List(ctx, OrderFilter{KindergartenID: kg.ID, Status: "paid", Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	assert.Equal(t, []string{"T3-F"}, nos(paid))

	byNo, _, err := orderRepo.List(ctx, OrderFilter{KindergartenID: kg.ID, OrderNo: "T3-B", Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.Equal(t, []string{"T3-B"}, nos(byNo), "订单号支持部分匹配")

	from := time.Date(2026, 11, 1, 0, 0, 0, 0, time.Local)
	to := time.Date(2026, 11, 11, 0, 0, 0, 0, time.Local) // 右开：含 11-10，不含 11-11
	ranged, total, err := orderRepo.List(ctx, OrderFilter{
		KindergartenID: kg.ID, DateFrom: &from, DateTo: &to, Page: 1, PageSize: 20,
	})
	require.NoError(t, err)
	assert.EqualValues(t, 3, total)
	assert.ElementsMatch(t, []string{"T3-B", "T3-C", "T3-F"}, nos(ranged))

	outstanding, total, err := orderRepo.List(ctx, OrderFilter{
		KindergartenID: kg.ID, OutstandingOnly: true, Page: 1, PageSize: 20,
	})
	require.NoError(t, err)
	assert.EqualValues(t, 4, total)
	assert.ElementsMatch(t, []string{"T3-A", "T3-B", "T3-E", "T3-G"}, nos(outstanding), "只返回未取消且未收清的订单")
}
