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

// 零收款确认 = 信用开通（design D2）：
// - 净收款为零时必须填写理由，否则拒绝且不留痕迹；
// - 有理由时标记 credit_granted + credit_reason，理由只进详情；
// - 信用开通计入未收余额（订单应收未收，仍出现在未收清筛选里）；
// - 审计与确认同事务，记录操作账号、时间与理由。

func newCreditGrantFixture(t *testing.T, db *gorm.DB) (*OrderService, *domain.Order) {
	t.Helper()
	ctx := context.Background()
	planSvc := NewPlanService(repository.NewPlanRepo(db), repository.NewPlatformLogRepo(db))
	orderSvc := NewOrderService(
		db, repository.NewOrderRepo(db), repository.NewPaymentRepo(db),
		repository.NewPlanRepo(db), repository.NewKindergartenRepo(db),
		repository.NewEntitlementRepo(db), repository.NewDistributorRepo(db),
		repository.NewCommissionRepo(db), repository.NewPlatformLogRepo(db), nil,
	)
	kg := &domain.Kindergarten{Name: "S4信用开通测试园", Principal: "王园长", Status: domain.KindergartenStatusActive}
	require.NoError(t, repository.NewKindergartenRepo(db).Create(ctx, kg))
	plan, err := planSvc.Create(ctx, PlanRequest{
		Code: "s4_credit", Name: "S4信用套餐", Tier: 1, PriceCents: 200000, TermMonths: 6,
		ModuleCodes: []string{"base", "growth"},
	}, "s4_ops", "127.0.0.1")
	require.NoError(t, err)
	order, err := orderSvc.Create(ctx, CreateOrderRequest{KindergartenID: kg.ID, PlanID: plan.ID}, "s4_ops", "127.0.0.1")
	require.NoError(t, err)
	return orderSvc, order
}

func cleanupCreditGrant(t *testing.T, db *gorm.DB) {
	t.Helper()
	del := func(sql string, args ...any) {
		if err := db.Exec(sql, args...).Error; err != nil {
			t.Logf("cleanup %q: %v", sql, err)
		}
	}
	del("DELETE FROM platform_operation_logs WHERE operator LIKE 's4_%'")
	del("DELETE FROM commission_ledger WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE 'S4信用开通测试%')")
	del("DELETE FROM payments WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE 'S4信用开通测试%')")
	del("DELETE FROM entitlements WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE 'S4信用开通测试%')")
	del("DELETE FROM orders WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE 'S4信用开通测试%')")
	del("DELETE FROM kindergartens WHERE name LIKE 'S4信用开通测试%'")
	del("DELETE FROM plans WHERE code LIKE 's4_credit%'")
}

func TestBillingCreditGrant_ZeroPaidConfirmRequiresReason(t *testing.T) {
	db := openPlatformTestDB(t)
	cleanupCreditGrant(t, db)
	t.Cleanup(func() { cleanupCreditGrant(t, db) })
	orderSvc, order := newCreditGrantFixture(t, db)
	ctx := context.Background()

	for _, reason := range []string{"", "   "} {
		_, _, err := orderSvc.Confirm(ctx, order.ID, reason, "s4_ops", "127.0.0.1")
		require.ErrorIs(t, err, ErrCreditReasonRequired, "零收款确认缺理由必须被拒绝（reason=%q）", reason)
	}

	reloaded, err := repository.NewOrderRepo(db).FindByID(ctx, order.ID)
	require.NoError(t, err)
	assert.Nil(t, reloaded.ConfirmedAt, "被拒绝的确认不得落库")
	assert.False(t, reloaded.CreditGranted)
	assert.Empty(t, reloaded.CreditReason)
	ents, err := repository.NewEntitlementRepo(db).ListByKindergarten(ctx, order.KindergartenID)
	require.NoError(t, err)
	assert.Empty(t, ents, "被拒绝的确认不得发放授权")
	var auditCount int64
	require.NoError(t, db.Model(&domain.PlatformOperationLog{}).
		Where("operator = ? AND module = ? AND action = ?", "s4_ops", "order", "confirm").
		Count(&auditCount).Error)
	assert.Zero(t, auditCount, "被拒绝的确认不得写确认审计")
}

func TestBillingCreditGrant_ConfirmRecordsCreditAndAudit(t *testing.T) {
	db := openPlatformTestDB(t)
	cleanupCreditGrant(t, db)
	t.Cleanup(func() { cleanupCreditGrant(t, db) })
	orderSvc, order := newCreditGrantFixture(t, db)
	ctx := context.Background()

	const reason = "合同已签署，对公转账到账后补录"
	confirmed, ents, err := orderSvc.Confirm(ctx, order.ID, reason, "s4_ops", "127.0.0.1")
	require.NoError(t, err)
	assert.NotNil(t, confirmed.ConfirmedAt)
	assert.True(t, confirmed.CreditGranted, "零收款确认必须标记为信用开通")
	assert.Equal(t, reason, confirmed.CreditReason)
	assert.NotEmpty(t, ents, "确认仍然发放授权")

	detail, err := orderSvc.Detail(ctx, order.ID)
	require.NoError(t, err)
	assert.True(t, detail.CreditGranted)
	assert.Equal(t, reason, detail.CreditReason, "理由只在详情返回")
	assert.Zero(t, detail.AmountPaidCents, "信用开通报文没有收款")

	// 信用开通计入未收余额：订单仍在「未收清」筛选里，应收全额未收。
	rows, total, err := orderSvc.List(ctx, repository.OrderFilter{
		KindergartenID: order.KindergartenID, OutstandingOnly: true, Page: 1, PageSize: 20,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, rows, 1)
	assert.True(t, rows[0].CreditGranted, "列表必须展示信用开通标记")
	assert.EqualValues(t, order.AmountDueCents, rows[0].AmountDueCents)
	assert.Zero(t, rows[0].AmountPaidCents)

	// 审计与确认同事务：操作账号、时间与理由都可追溯。
	var log domain.PlatformOperationLog
	require.NoError(t, db.Where("operator = ? AND module = ? AND action = ?", "s4_ops", "order", "confirm").
		First(&log).Error)
	assert.EqualValues(t, order.ID, log.TargetID)
	assert.EqualValues(t, order.KindergartenID, log.KindergartenID)
	assert.Contains(t, log.Detail, reason, "审计必须包含信用开通理由")
	assert.False(t, log.CreatedAt.IsZero(), "审计必须记录时间")
}

func TestBillingCreditGrant_PaidConfirmNeedsNoReason(t *testing.T) {
	db := openPlatformTestDB(t)
	cleanupCreditGrant(t, db)
	t.Cleanup(func() { cleanupCreditGrant(t, db) })
	orderSvc, order := newCreditGrantFixture(t, db)
	ctx := context.Background()

	_, err := orderSvc.AddPayment(ctx, order.ID, PaymentRequest{AmountCents: order.AmountDueCents}, "s4_ops", "127.0.0.1")
	require.NoError(t, err)

	confirmed, _, err := orderSvc.Confirm(ctx, order.ID, "", "s4_ops", "127.0.0.1")
	require.NoError(t, err, "有实收的确认不需要理由")
	assert.False(t, confirmed.CreditGranted, "有实收的订单不是信用开通")
	assert.Empty(t, confirmed.CreditReason)

	detail, err := orderSvc.Detail(ctx, order.ID)
	require.NoError(t, err)
	assert.False(t, detail.CreditGranted)
	assert.Empty(t, detail.CreditReason)
}

func TestBillingCreditGrant_RejectsOverlongReason(t *testing.T) {
	db := openPlatformTestDB(t)
	cleanupCreditGrant(t, db)
	t.Cleanup(func() { cleanupCreditGrant(t, db) })
	orderSvc, order := newCreditGrantFixture(t, db)
	ctx := context.Background()

	reason := ""
	for i := 0; i < 256; i++ {
		reason += "理"
	}
	_, _, err := orderSvc.Confirm(ctx, order.ID, reason, "s4_ops", "127.0.0.1")
	require.ErrorIs(t, err, ErrCreditReasonTooLong)

	reloaded, err := repository.NewOrderRepo(db).FindByID(ctx, order.ID)
	require.NoError(t, err)
	assert.Nil(t, reloaded.ConfirmedAt)
}
