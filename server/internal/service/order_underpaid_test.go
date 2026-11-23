package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// 本切片锁定 #201 的确认规则：
// - 部分收款（0 < 净收款 < 应收）一律不可确认，理由不能替代收清；
// - 零收款仍可信用开通（ADR-0021 的明确例外），但必须留理由；
// - 收清（净收款 >= 应收）无需理由，确认即发放授权。
// 被拒绝的确认不得留下 confirmed_at、授权或确认审计。

func TestOrderConfirmRejectsPartialPaymentEvenWithReason(t *testing.T) {
	db := openPlatformTestDB(t)
	ctx := context.Background()
	cleanupConfirmTx(t, db)
	t.Cleanup(func() { cleanupConfirmTx(t, db) })

	f := newConfirmTxFixture(t, db, "-部分收款")
	order := f.createOrder(t, db)
	require.Positive(t, order.AmountDueCents, "用例需要正数应收")

	const partial = int64(1)
	_, err := f.svc.AddPayment(ctx, order.ID, PaymentRequest{
		AmountCents: partial, IdempotencyKey: "cn4-underpaid-partial",
	}, confirmTxOperator, "127.0.0.1")
	require.NoError(t, err)

	// 理由不是绕过：即使带理由也必须拒绝。
	_, _, err = f.svc.Confirm(ctx, order.ID, "即使有理由也不能替代收清", confirmTxOperator, "127.0.0.1")
	var underpaid *OrderUnderpaidError
	require.ErrorAs(t, err, &underpaid, "部分收款确认必须返回可识别的欠款错误")
	assert.EqualValues(t, order.AmountDueCents-partial, underpaid.RemainingCents, "剩余金额必须准确")

	fresh, err := repository.NewOrderRepo(db).FindByID(ctx, order.ID)
	require.NoError(t, err)
	require.NotNil(t, fresh)
	assert.Nil(t, fresh.ConfirmedAt, "被拒绝的确认不得落库")
	assert.False(t, fresh.CreditGranted, "部分收款不是信用开通")
	assert.Empty(t, fresh.CreditReason)
	assert.Zero(t, countEntitlementsOfOrder(t, db, order.ID), "被拒绝的确认不得发放授权")
	assert.Zero(t, countConfirmAudits(t, db, order.ID), "被拒绝的确认不得写确认审计")

	detail, err := f.svc.Detail(ctx, order.ID)
	require.NoError(t, err)
	assert.Nil(t, detail.ConfirmedAt, "被拒绝的确认不得在详情里显示已确认")
}

func TestOrderConfirmAllowsFullPaymentWithoutReason(t *testing.T) {
	db := openPlatformTestDB(t)
	ctx := context.Background()
	cleanupConfirmTx(t, db)
	t.Cleanup(func() { cleanupConfirmTx(t, db) })

	f := newConfirmTxFixture(t, db, "-全额收款")
	order := f.createOrder(t, db)

	_, err := f.svc.AddPayment(ctx, order.ID, PaymentRequest{
		AmountCents: order.AmountDueCents, IdempotencyKey: "cn4-underpaid-full",
	}, confirmTxOperator, "127.0.0.1")
	require.NoError(t, err)

	confirmed, ents, err := f.svc.Confirm(ctx, order.ID, "", confirmTxOperator, "127.0.0.1")
	require.NoError(t, err, "收清后无需理由即可确认")
	assert.NotNil(t, confirmed.ConfirmedAt)
	assert.False(t, confirmed.CreditGranted, "收清不是信用开通")
	assert.Equal(t, domain.OrderStatusPaid, confirmed.Status)
	require.Len(t, ents, f.premiumCount, "确认必须发放全套 premium 授权")
	assert.EqualValues(t, f.premiumCount, countEntitlementsOfOrder(t, db, order.ID))
	assert.EqualValues(t, 1, countConfirmAudits(t, db, order.ID))

	detail, err := f.svc.Detail(ctx, order.ID)
	require.NoError(t, err)
	assert.NotNil(t, detail.ConfirmedAt, "确认成功后详情必须显示已确认，管理端据此隐藏确认按钮")
}

// 超收不可通过 AddPayment 制造（会被上限拒绝），因此直接插一行超额收款，
// 锁定实现使用「净收款 >= 应收」而不是「== 应收」。
func TestOrderConfirmAllowsOverpayment(t *testing.T) {
	db := openPlatformTestDB(t)
	ctx := context.Background()
	cleanupConfirmTx(t, db)
	t.Cleanup(func() { cleanupConfirmTx(t, db) })

	f := newConfirmTxFixture(t, db, "-超收")
	order := f.createOrder(t, db)

	_, err := f.svc.AddPayment(ctx, order.ID, PaymentRequest{
		AmountCents: order.AmountDueCents, IdempotencyKey: "cn4-underpaid-base",
	}, confirmTxOperator, "127.0.0.1")
	require.NoError(t, err)
	// 历史数据/人工补录可能留下超额收款，确认不应因此失败。
	require.NoError(t, db.Create(&domain.Payment{
		OrderID: order.ID, KindergartenID: order.KindergartenID,
		AmountCents: 1, Method: domain.PaymentMethodBankTransfer,
		ReceivedAt: order.StartsAt, RecordedBy: confirmTxOperator,
	}).Error)

	_, _, err = f.svc.Confirm(ctx, order.ID, "", confirmTxOperator, "127.0.0.1")
	require.NoError(t, err, "净收款超过应收时确认必须放行")
}

func TestOrderConfirmZeroPaymentStillRequiresReason(t *testing.T) {
	db := openPlatformTestDB(t)
	ctx := context.Background()
	cleanupConfirmTx(t, db)
	t.Cleanup(func() { cleanupConfirmTx(t, db) })

	f := newConfirmTxFixture(t, db, "-零收款")
	order := f.createOrder(t, db)

	_, _, err := f.svc.Confirm(ctx, order.ID, "   ", confirmTxOperator, "127.0.0.1")
	require.ErrorIs(t, err, ErrCreditReasonRequired, "零收款确认缺理由必须被拒绝")

	fresh, err := repository.NewOrderRepo(db).FindByID(ctx, order.ID)
	require.NoError(t, err)
	assert.Nil(t, fresh.ConfirmedAt)
	assert.Zero(t, countEntitlementsOfOrder(t, db, order.ID))
	assert.Zero(t, countConfirmAudits(t, db, order.ID))

	const reason = "合同已签署，对公转账到账后补录"
	confirmed, _, err := f.svc.Confirm(ctx, order.ID, reason, confirmTxOperator, "127.0.0.1")
	require.NoError(t, err, "零收款 + 理由仍可信用开通")
	assert.True(t, confirmed.CreditGranted)
	assert.Equal(t, reason, confirmed.CreditReason)

	detail, err := f.svc.Detail(ctx, order.ID)
	require.NoError(t, err)
	assert.NotNil(t, detail.ConfirmedAt)
	assert.True(t, detail.CreditGranted)
}
