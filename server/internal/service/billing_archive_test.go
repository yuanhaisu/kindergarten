package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// seedStudents 创建一个班并插入 n 名在读幼儿，返回班级 ID。
// 学生表对 class_id 有外键，birthday/enroll_date 不接受零值日期。
func seedStudents(t *testing.T, db *gorm.DB, kgID uint64, n int) uint64 {
	t.Helper()
	class := &domain.Class{KindergartenID: kgID, Name: "S1B归档测试班", Capacity: 60, Status: 1}
	require.NoError(t, db.Create(class).Error)
	for range n {
		require.NoError(t, db.Create(&domain.Student{
			KindergartenID: kgID, ClassID: class.ID, Name: "S1B归档测试幼儿", Status: 1,
			Birthday: time.Now(), EnrollDate: time.Now(),
		}).Error)
	}
	return class.ID
}

func newArchiveDistributorService(db *gorm.DB) *DistributorService {
	return NewDistributorService(
		repository.NewDistributorRepo(db),
		repository.NewCommissionRepo(db),
		repository.NewOrderRepo(db),
		repository.NewKindergartenRepo(db),
		repository.NewPlatformLogRepo(db),
	)
}

// CAT-0002：套餐引用未注册模块必须被拒。
func TestPlan_RejectsUnknownModule(t *testing.T) {
	db := openPlatformTestDB(t)
	planSvc, _, _ := newBillingFlowServices(t, db)
	ctx := context.Background()
	cleanupBillingFlow(t, db)
	t.Cleanup(func() { cleanupBillingFlow(t, db) })

	_, err := planSvc.Create(ctx, PlanRequest{
		Code: "s1b_unknown", Name: "S1B非法模块版", ModuleCodes: []string{"base", "not_a_module"},
	}, "s1b_ops", "127.0.0.1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "UNKNOWN_MODULE")
}

// CAT-0005：改价只影响之后创建的订单。
func TestOrder_NewOrdersUseCurrentPrice(t *testing.T) {
	db := openPlatformTestDB(t)
	planSvc, orderSvc, _ := newBillingFlowServices(t, db)
	ctx := context.Background()
	cleanupBillingFlow(t, db)
	t.Cleanup(func() { cleanupBillingFlow(t, db) })

	kg := &domain.Kindergarten{Name: "S1B订单测试园H", Status: domain.KindergartenStatusActive}
	require.NoError(t, repository.NewKindergartenRepo(db).Create(ctx, kg))

	plan, err := planSvc.Create(ctx, PlanRequest{
		Code: "s1b_price", Name: "S1B价格版", PriceCents: 100000, TermMonths: 6,
		ModuleCodes: []string{"base", "growth"},
	}, "s1b_ops", "127.0.0.1")
	require.NoError(t, err)

	first, err := orderSvc.Create(ctx, CreateOrderRequest{KindergartenID: kg.ID, PlanID: plan.ID}, "s1b_ops", "127.0.0.1")
	require.NoError(t, err)
	require.EqualValues(t, 100000, first.AmountDueCents)

	require.NoError(t, planSvc.Update(ctx, plan.ID, PlanRequest{
		Name: "S1B价格版", PriceCents: 200000, ModuleCodes: []string{"base", "growth"},
	}, "s1b_ops", "127.0.0.1"))

	second, err := orderSvc.Create(ctx, CreateOrderRequest{KindergartenID: kg.ID, PlanID: plan.ID}, "s1b_ops", "127.0.0.1")
	require.NoError(t, err)
	assert.EqualValues(t, 200000, second.AmountDueCents, "新订单必须用新价")

	again, err := orderSvc.Detail(ctx, first.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 100000, again.AmountDueCents, "老订单金额不变")
}

// CAT-0006 / CAT-0007：档位由在园幼儿数决定，且成交后不随人数变化。
func TestOrder_TierByStudentCountAndFrozen(t *testing.T) {
	db := openPlatformTestDB(t)
	planSvc, orderSvc, _ := newBillingFlowServices(t, db)
	ctx := context.Background()
	cleanupBillingFlow(t, db)
	t.Cleanup(func() { cleanupBillingFlow(t, db) })

	kg := &domain.Kindergarten{Name: "S1B订单测试园I", Status: domain.KindergartenStatusActive}
	require.NoError(t, repository.NewKindergartenRepo(db).Create(ctx, kg))
	seedStudents(t, db, kg.ID, 2)

	plan, err := planSvc.Create(ctx, PlanRequest{
		Code: "s1b_tier2", Name: "S1B档位版2", PriceCents: 100000, TermMonths: 6,
		ModuleCodes: []string{"base", "growth"},
	}, "s1b_ops", "127.0.0.1")
	require.NoError(t, err)
	require.NoError(t, planSvc.ReplaceTiers(ctx, plan.ID, []domain.PlanTier{
		{Tier: 1, MinStudents: 0, MaxStudents: 1, PriceMultiplier: 100},
		{Tier: 2, MinStudents: 2, MaxStudents: 0, PriceMultiplier: 150},
	}, "s1b_ops", "127.0.0.1"))

	order, err := orderSvc.Create(ctx, CreateOrderRequest{KindergartenID: kg.ID, PlanID: plan.ID}, "s1b_ops", "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, 2, order.StudentTier, "2 名幼儿应落入第二档")
	assert.EqualValues(t, 150000, order.AmountDueCents, "价格按 150% 系数")

	// 学期中人数增加，不改已成交订单
	seedStudents(t, db, kg.ID, 5)
	again, err := orderSvc.Detail(ctx, order.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, again.PlanSnapshot.Tier)
	assert.EqualValues(t, 150000, again.AmountDueCents)
}

// ORD-0008：开票信息可保存并回读；缺失不阻断订单生效。
func TestOrder_InvoiceInfoRoundTrip(t *testing.T) {
	db := openPlatformTestDB(t)
	planSvc, orderSvc, _ := newBillingFlowServices(t, db)
	ctx := context.Background()
	cleanupBillingFlow(t, db)
	t.Cleanup(func() { cleanupBillingFlow(t, db) })

	kg := &domain.Kindergarten{Name: "S1B订单测试园J", Status: domain.KindergartenStatusActive}
	require.NoError(t, repository.NewKindergartenRepo(db).Create(ctx, kg))
	plan, err := planSvc.Create(ctx, PlanRequest{
		Code: "s1b_invoice", Name: "S1B开票版", PriceCents: 100000, TermMonths: 6,
		ModuleCodes: []string{"base"},
	}, "s1b_ops", "127.0.0.1")
	require.NoError(t, err)

	invoice := json.RawMessage(`{"title":"阳光幼儿园","tax_no":"91330100XXXXXXXX"}`)
	withInvoice, err := orderSvc.Create(ctx, CreateOrderRequest{
		KindergartenID: kg.ID, PlanID: plan.ID, InvoiceInfo: invoice,
	}, "s1b_ops", "127.0.0.1")
	require.NoError(t, err)
	_, _, err = orderSvc.Confirm(ctx, withInvoice.ID, "测试信用开通", "s1b_ops", "127.0.0.1")
	require.NoError(t, err)
	detail, err := orderSvc.Detail(ctx, withInvoice.ID)
	require.NoError(t, err)
	var got map[string]string
	require.NoError(t, json.Unmarshal(detail.InvoiceInfo, &got))
	assert.Equal(t, "阳光幼儿园", got["title"])

	// 不填开票信息也能确认
	noInvoice, err := orderSvc.Create(ctx, CreateOrderRequest{KindergartenID: kg.ID, PlanID: plan.ID}, "s1b_ops", "127.0.0.1")
	require.NoError(t, err)
	_, _, err = orderSvc.Confirm(ctx, noInvoice.ID, "测试信用开通", "s1b_ops", "127.0.0.1")
	require.NoError(t, err, "缺少开票信息不得阻断订单生效")
	detail, err = orderSvc.Detail(ctx, noInvoice.ID)
	require.NoError(t, err)
	assert.Nil(t, detail.InvoiceInfo)
}

// DST-0001：归因在成交时冻结——系统没有修改归因的写入路径，
// 后续收款与详情都只能读到成交时的那家分销商。
func TestOrder_AttributionFrozen(t *testing.T) {
	db := openPlatformTestDB(t)
	planSvc, orderSvc, _ := newBillingFlowServices(t, db)
	ctx := context.Background()
	cleanupBillingFlow(t, db)
	t.Cleanup(func() { cleanupBillingFlow(t, db) })

	kg := &domain.Kindergarten{Name: "S1B订单测试园K", Status: domain.KindergartenStatusActive}
	require.NoError(t, repository.NewKindergartenRepo(db).Create(ctx, kg))
	plan, err := planSvc.Create(ctx, PlanRequest{
		Code: "s1b_attr", Name: "S1B归因版", PriceCents: 100000, TermMonths: 6,
		ModuleCodes: []string{"base"},
	}, "s1b_ops", "127.0.0.1")
	require.NoError(t, err)
	dist := &domain.Distributor{Name: "S1B分销商A", CommissionRate: 10, Status: 1}
	require.NoError(t, repository.NewDistributorRepo(db).Create(ctx, dist))

	order, err := orderSvc.Create(ctx, CreateOrderRequest{
		KindergartenID: kg.ID, PlanID: plan.ID, DistributorID: &dist.ID,
	}, "s1b_ops", "127.0.0.1")
	require.NoError(t, err)
	_, _, err = orderSvc.Confirm(ctx, order.ID, "测试信用开通", "s1b_ops", "127.0.0.1")
	require.NoError(t, err)
	_, err = orderSvc.AddPayment(ctx, order.ID, PaymentRequest{AmountCents: 50000}, "s1b_ops", "127.0.0.1")
	require.NoError(t, err)

	detail, err := orderSvc.Detail(ctx, order.ID)
	require.NoError(t, err)
	require.NotNil(t, detail.Distributor)
	assert.Equal(t, dist.ID, detail.Distributor.ID, "收款后归因不得改变")
	assert.Equal(t, 10, detail.Distributor.CommissionRate, "佣金比例取成交时冻结值")
}

// DST-0006：到账满 30 天后，待结算佣金在读取台账时推进为可结算。
func TestCommission_PromotedAfter30Days(t *testing.T) {
	db := openPlatformTestDB(t)
	distSvc := newArchiveDistributorService(db)
	ctx := context.Background()
	cleanupBillingFlow(t, db)
	t.Cleanup(func() { cleanupBillingFlow(t, db) })

	kg := &domain.Kindergarten{Name: "S1B订单测试园L", Status: domain.KindergartenStatusActive}
	require.NoError(t, repository.NewKindergartenRepo(db).Create(ctx, kg))
	dist := &domain.Distributor{Name: "S1B分销商B", CommissionRate: 10, Status: 1}
	require.NoError(t, repository.NewDistributorRepo(db).Create(ctx, dist))

	now := time.Now()
	ledger := []domain.CommissionLedger{
		{OrderID: 9001, PaymentID: 9001, DistributorID: dist.ID, KindergartenID: kg.ID,
			BaseAmountCents: 100000, Rate: 10, AmountCents: 10000,
			Status: domain.CommissionStatusPending, EligibleAt: now.Add(-time.Hour)},
		{OrderID: 9002, PaymentID: 9002, DistributorID: dist.ID, KindergartenID: kg.ID,
			BaseAmountCents: 100000, Rate: 10, AmountCents: 10000,
			Status: domain.CommissionStatusPending, EligibleAt: now.Add(24 * time.Hour)},
	}
	require.NoError(t, db.Create(&ledger).Error)

	rows, _, summary, err := distSvc.ListCommissions(ctx, repository.CommissionFilter{DistributorID: dist.ID})
	require.NoError(t, err)
	require.Len(t, rows, 2)
	for _, row := range rows {
		if row.EligibleAt.Before(now) {
			assert.Equal(t, domain.CommissionStatusSettleable, row.Status, "满 30 天应推进为可结算")
		} else {
			assert.Equal(t, domain.CommissionStatusPending, row.Status, "未满 30 天保持待结算")
		}
	}
	assert.EqualValues(t, 10000, summary[domain.CommissionStatusSettleable])
	assert.EqualValues(t, 10000, summary[domain.CommissionStatusPending])
}

// ENT-0012：试用到期只收紧授权，不删除试用期产生的业务数据。
func TestEntitlement_TrialExpiryKeepsData(t *testing.T) {
	db := openPlatformTestDB(t)
	_, _, entSvc := newBillingFlowServices(t, db)
	ctx := context.Background()
	cleanupBillingFlow(t, db)
	t.Cleanup(func() { cleanupBillingFlow(t, db) })

	kg := &domain.Kindergarten{Name: "S1B订单测试园M", Status: domain.KindergartenStatusActive}
	require.NoError(t, repository.NewKindergartenRepo(db).Create(ctx, kg))
	seedStudents(t, db, kg.ID, 1)

	now := time.Now()
	require.NoError(t, repository.NewEntitlementRepo(db).CreateBatch(ctx, []domain.Entitlement{{
		KindergartenID: kg.ID, ModuleCode: "growth", Source: domain.EntitlementSourceTrial,
		// 过期 30 天，已超出 15 天宽限期
		StartsAt: now.AddDate(0, -3, 0), ExpiresAt: now.AddDate(0, 0, -30),
	}}))

	decision, err := entSvc.Decide(ctx, kg.ID, "growth", true)
	require.NoError(t, err)
	assert.False(t, decision.Allowed, "过期且超出宽限期后不可写")

	count, err := repository.NewKindergartenRepo(db).CountStudents(ctx, kg.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 1, count, "试用期数据必须保留")

	rows, err := repository.NewEntitlementRepo(db).ListByKindergarten(ctx, kg.ID)
	require.NoError(t, err)
	assert.Len(t, rows, 1, "授权行只是过期，不删除，保留追溯")
}
