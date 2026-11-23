package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// 本切片锁定 #203 的确认前置守卫（都在 #200 的事务与订单行锁内、任何写入之前）：
// - 服务期已结束（expires_at <= now）：即使已全额收款也拒绝（409 ORDER_SERVICE_ENDED），
//   不隐式顺延，避免发放「入职即过期」的授权；
// - 派生状态 expired（已过付款到期日的本地日历日）且未收清：一律拒绝
//   （422 ORDER_EXPIRED），零收款理由也不能绕过；
// - 全额晚付的派生状态是 paid，放行；不得用 pay_due_at < now 做原始时间比较；
// - 优先级：服务期结束 > 过期 > 部分收款 > 零收款理由。
// 被拒绝的确认不得留下 confirmed_at、授权或确认审计。
//
// 到期日是日历日口径：到期日当天 23:59:59 仍可确认，次日 00:00 起才算过期。

// updateOrderWindow 直接改写订单的付款到期日与服务期，用于构造 #202 之后
// API 不再允许造出的历史过期/服务期结束订单。
func updateOrderWindow(t *testing.T, db *gorm.DB, orderID uint64, payDueAt, expiresAt time.Time) {
	t.Helper()
	require.NoError(t, db.Model(&domain.Order{}).Where("id = ?", orderID).
		Updates(map[string]any{"pay_due_at": payDueAt, "expires_at": expiresAt}).Error)
}

func assertConfirmRejectedCleanly(t *testing.T, db *gorm.DB, svc *OrderService, orderID uint64) {
	t.Helper()
	fresh, err := repository.NewOrderRepo(db).FindByID(context.Background(), orderID)
	require.NoError(t, err)
	require.NotNil(t, fresh)
	assert.Nil(t, fresh.ConfirmedAt, "被拒绝的确认不得写 confirmed_at")
	assert.False(t, fresh.CreditGranted, "被拒绝的确认不得标记信用开通")
	assert.Zero(t, countEntitlementsOfOrder(t, db, orderID), "被拒绝的确认不得发放授权")
	assert.Zero(t, countConfirmAudits(t, db, orderID), "被拒绝的确认不得写确认审计")
}

// dueDayEnd 返回本地日历「今天」的 23:59:59；派生状态只按日历日比较，当天尚未过期。
func dueDayEnd(now time.Time) time.Time {
	return time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
}

// localMidnight 返回本地时区 now 往前 days 天的零点。
func localMidnight(now time.Time, days int) time.Time {
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, days)
}

// 到期日 23:59:59 仍未过期：零收款 + 理由照常信用开通。
// 这条锁住「必须用派生状态，不能用 pay_due_at < now」的边界。
func TestOrderConfirm_ZeroCreditAllowedOnDueDay(t *testing.T) {
	db := openPlatformTestDB(t)
	ctx := context.Background()
	cleanupConfirmTx(t, db)
	t.Cleanup(func() { cleanupConfirmTx(t, db) })

	f := newConfirmTxFixture(t, db, "-到期日当天")
	order := f.createOrder(t, db)
	now := time.Now()
	updateOrderWindow(t, db, order.ID, dueDayEnd(now), order.ExpiresAt)

	confirmed, ents, err := f.svc.Confirm(ctx, order.ID, "到期日当天先信用开通", confirmTxOperator, "127.0.0.1")
	require.NoError(t, err, "到期日当天 23:59:59 之前仍允许零收款信用开通")
	assert.NotNil(t, confirmed.ConfirmedAt)
	assert.True(t, confirmed.CreditGranted)
	require.Len(t, ents, f.premiumCount)
	assert.EqualValues(t, f.premiumCount, countEntitlementsOfOrder(t, db, order.ID))
	assert.EqualValues(t, 1, countConfirmAudits(t, db, order.ID))
}

// 次日 00:00 起未收清即过期：零收款带理由也必须拒绝，且不留任何写入。
func TestOrderConfirm_ExpiredNextDayRejectsZeroCreditEvenWithReason(t *testing.T) {
	db := openPlatformTestDB(t)
	ctx := context.Background()
	cleanupConfirmTx(t, db)
	t.Cleanup(func() { cleanupConfirmTx(t, db) })

	f := newConfirmTxFixture(t, db, "-次日零收款")
	order := f.createOrder(t, db)
	now := time.Now()
	updateOrderWindow(t, db, order.ID, localMidnight(now, -1), order.ExpiresAt)

	_, _, err := f.svc.Confirm(ctx, order.ID, "过期后仍想信用开通", confirmTxOperator, "127.0.0.1")
	require.ErrorIs(t, err, ErrOrderExpired, "过期订单的零收款确认必须返回 ORDER_EXPIRED")

	assertConfirmRejectedCleanly(t, db, f.svc, order.ID)

	detail, err := f.svc.Detail(ctx, order.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusExpired, detail.Status)
}

// 过期 + 部分收款：拒绝理由回 ORDER_EXPIRED（过期优先于部分收款），
// 而不是 #201 的 ORDER_UNDERPAID。
func TestOrderConfirm_ExpiredUnderpaidRejectedBeforePartialPaymentGuard(t *testing.T) {
	db := openPlatformTestDB(t)
	ctx := context.Background()
	cleanupConfirmTx(t, db)
	t.Cleanup(func() { cleanupConfirmTx(t, db) })

	f := newConfirmTxFixture(t, db, "-过期部分收款")
	order := f.createOrder(t, db)
	now := time.Now()
	updateOrderWindow(t, db, order.ID, localMidnight(now, -1), order.ExpiresAt)

	_, err := f.svc.AddPayment(ctx, order.ID, PaymentRequest{
		AmountCents: 1, IdempotencyKey: "cn4-expired-partial",
	}, confirmTxOperator, "127.0.0.1")
	require.NoError(t, err)

	_, _, err = f.svc.Confirm(ctx, order.ID, "过期仍想确认", confirmTxOperator, "127.0.0.1")
	require.ErrorIs(t, err, ErrOrderExpired, "过期订单必须返回 ORDER_EXPIRED")
	var underpaid *OrderUnderpaidError
	assert.False(t, errors.As(err, &underpaid), "过期守卫优先于部分收款守卫，不应返回 ORDER_UNDERPAID")

	assertConfirmRejectedCleanly(t, db, f.svc, order.ID)
}

// 全额晚付：付款到期日已过，但净收款已收清，派生状态为 paid，确认放行。
// 授权凭据仍取订单的 starts_at / expires_at，而不是 pay_due_at。
func TestOrderConfirm_FullLatePaymentAllowed(t *testing.T) {
	db := openPlatformTestDB(t)
	ctx := context.Background()
	cleanupConfirmTx(t, db)
	t.Cleanup(func() { cleanupConfirmTx(t, db) })

	f := newConfirmTxFixture(t, db, "-全额晚付")
	order := f.createOrder(t, db)
	now := time.Now()
	updateOrderWindow(t, db, order.ID, localMidnight(now, -7), order.ExpiresAt)

	_, err := f.svc.AddPayment(ctx, order.ID, PaymentRequest{
		AmountCents: order.AmountDueCents, IdempotencyKey: "cn4-late-full",
	}, confirmTxOperator, "127.0.0.1")
	require.NoError(t, err)

	confirmed, ents, err := f.svc.Confirm(ctx, order.ID, "", confirmTxOperator, "127.0.0.1")
	require.NoError(t, err, "全额晚付的派生状态是 paid，必须放行")
	assert.NotNil(t, confirmed.ConfirmedAt)
	assert.Equal(t, domain.OrderStatusPaid, confirmed.Status)
	require.Len(t, ents, f.premiumCount)
	for _, e := range ents {
		assert.WithinDuration(t, order.StartsAt, e.StartsAt, time.Second, "授权起始取订单 starts_at")
		assert.WithinDuration(t, order.ExpiresAt, e.ExpiresAt, time.Second, "授权到期取订单 expires_at，而不是 pay_due_at")
	}
	assert.EqualValues(t, f.premiumCount, countEntitlementsOfOrder(t, db, order.ID))
	assert.EqualValues(t, 1, countConfirmAudits(t, db, order.ID))
}

// 服务期结束：即使已全额收款也 409，拒绝发放「发完即过期」的授权；不隐式顺延。
func TestOrderConfirm_ServiceEndedRejectsEvenFullPayment(t *testing.T) {
	db := openPlatformTestDB(t)
	ctx := context.Background()
	cleanupConfirmTx(t, db)
	t.Cleanup(func() { cleanupConfirmTx(t, db) })

	f := newConfirmTxFixture(t, db, "-服务期结束")
	order := f.createOrder(t, db)
	now := time.Now()
	// 付款到期日仍在今天（派生状态 unpaid），只有服务期已结束。
	updateOrderWindow(t, db, order.ID, dueDayEnd(now), now.Add(-time.Minute))

	_, err := f.svc.AddPayment(ctx, order.ID, PaymentRequest{
		AmountCents: order.AmountDueCents, IdempotencyKey: "cn4-service-ended",
	}, confirmTxOperator, "127.0.0.1")
	require.NoError(t, err)

	_, _, err = f.svc.Confirm(ctx, order.ID, "", confirmTxOperator, "127.0.0.1")
	require.ErrorIs(t, err, ErrOrderServiceEnded, "服务期已结束的订单即使全额已付也必须拒绝")

	assertConfirmRejectedCleanly(t, db, f.svc, order.ID)
}

// 服务期未结束（expires_at > now）：同样的已收清订单必须放行，
// 锁定守卫只看 expires_at 是否已过，不误伤在期订单。
func TestOrderConfirm_ServiceEndInFutureStillAllowed(t *testing.T) {
	db := openPlatformTestDB(t)
	ctx := context.Background()
	cleanupConfirmTx(t, db)
	t.Cleanup(func() { cleanupConfirmTx(t, db) })

	f := newConfirmTxFixture(t, db, "-服务期在期")
	order := f.createOrder(t, db)

	_, err := f.svc.AddPayment(ctx, order.ID, PaymentRequest{
		AmountCents: order.AmountDueCents, IdempotencyKey: "cn4-service-live",
	}, confirmTxOperator, "127.0.0.1")
	require.NoError(t, err)

	_, _, err = f.svc.Confirm(ctx, order.ID, "", confirmTxOperator, "127.0.0.1")
	require.NoError(t, err, "服务期未结束的已收清订单必须可确认")
}

// 优先级：服务期结束 > 过期 > 部分收款 > 零收款理由。
// 每一行都同时满足该行及更低优先级的触发条件，断言命中的是高优先级错误。
func TestOrderConfirm_GuardPrecedence(t *testing.T) {
	db := openPlatformTestDB(t)
	ctx := context.Background()
	cleanupConfirmTx(t, db)
	t.Cleanup(func() { cleanupConfirmTx(t, db) })

	f := newConfirmTxFixture(t, db, "-守卫优先级")
	now := time.Now()

	t.Run("服务期结束优先于过期与部分收款", func(t *testing.T) {
		order := f.createOrder(t, db)
		// 同时过期 + 服务期结束 + 部分收款。
		updateOrderWindow(t, db, order.ID, localMidnight(now, -7), now.Add(-time.Hour))
		_, err := f.svc.AddPayment(ctx, order.ID, PaymentRequest{
			AmountCents: 1, IdempotencyKey: "cn4-prio-ended",
		}, confirmTxOperator, "127.0.0.1")
		require.NoError(t, err)

		_, _, err = f.svc.Confirm(ctx, order.ID, "带理由", confirmTxOperator, "127.0.0.1")
		require.ErrorIs(t, err, ErrOrderServiceEnded)
		assertConfirmRejectedCleanly(t, db, f.svc, order.ID)
	})

	t.Run("过期优先于部分收款与零收款理由", func(t *testing.T) {
		order := f.createOrder(t, db)
		updateOrderWindow(t, db, order.ID, localMidnight(now, -1), order.ExpiresAt)
		_, err := f.svc.AddPayment(ctx, order.ID, PaymentRequest{
			AmountCents: 1, IdempotencyKey: "cn4-prio-expired",
		}, confirmTxOperator, "127.0.0.1")
		require.NoError(t, err)

		_, _, err = f.svc.Confirm(ctx, order.ID, "带理由", confirmTxOperator, "127.0.0.1")
		require.ErrorIs(t, err, ErrOrderExpired)
		assertConfirmRejectedCleanly(t, db, f.svc, order.ID)
	})

	t.Run("部分收款优先于零收款理由", func(t *testing.T) {
		order := f.createOrder(t, db)
		updateOrderWindow(t, db, order.ID, dueDayEnd(now), order.ExpiresAt)
		_, err := f.svc.AddPayment(ctx, order.ID, PaymentRequest{
			AmountCents: 1, IdempotencyKey: "cn4-prio-partial",
		}, confirmTxOperator, "127.0.0.1")
		require.NoError(t, err)

		_, _, err = f.svc.Confirm(ctx, order.ID, "", confirmTxOperator, "127.0.0.1")
		var underpaid *OrderUnderpaidError
		require.ErrorAs(t, err, &underpaid, "未过期部分收款命中 #201 欠款守卫")
		assert.EqualValues(t, order.AmountDueCents-1, underpaid.RemainingCents)
	})
}

// 并发确认过期订单：所有请求都必须被 ORDER_EXPIRED 拒绝，
// 不得因并发竞争漏过守卫、留下授权或审计。
func TestOrderConfirm_ExpiredConcurrentConfirmAllRejected(t *testing.T) {
	db := openPlatformTestDB(t)
	ctx := context.Background()
	cleanupConfirmTx(t, db)
	t.Cleanup(func() { cleanupConfirmTx(t, db) })

	f := newConfirmTxFixture(t, db, "-并发过期")
	order := f.createOrder(t, db)
	now := time.Now()
	updateOrderWindow(t, db, order.ID, localMidnight(now, -1), order.ExpiresAt)

	const workers = 6
	start := make(chan struct{})
	var wg sync.WaitGroup
	errs := make([]error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			_, _, errs[i] = f.svc.Confirm(ctx, order.ID, "并发过期确认", confirmTxOperator, "127.0.0.1")
		}(i)
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		require.ErrorIs(t, err, ErrOrderExpired, "worker %d 必须命中过期守卫", i)
	}
	assertConfirmRejectedCleanly(t, db, f.svc, order.ID)
}
