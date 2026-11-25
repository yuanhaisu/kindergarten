package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/repository"
)

// #202 决策 6：0 元套餐会生成永远收不清的订单，创建与更新都必须拒绝。
func TestPlan_ZeroPriceCreateRejected(t *testing.T) {
	db := openPlatformTestDB(t)
	planSvc, _, _ := newBillingFlowServices(t, db)
	ctx := context.Background()
	cleanupBillingFlow(t, db)
	t.Cleanup(func() { cleanupBillingFlow(t, db) })

	_, err := planSvc.Create(ctx, PlanRequest{
		Code: "s1b_zero_create", Name: "T3零元套餐", Tier: 1, PriceCents: 0, TermMonths: 6,
		ModuleCodes: []string{"base"},
	}, "t3_ops", "127.0.0.1")
	require.ErrorIs(t, err, ErrPlanPriceNotPositive)

	found, err := repository.NewPlanRepo(db).FindByCode(ctx, "s1b_zero_create")
	require.NoError(t, err)
	assert.Nil(t, found, "被拒绝的套餐不得落库")
}

func TestPlan_ZeroPriceUpdateRejected(t *testing.T) {
	db := openPlatformTestDB(t)
	planSvc, _, _ := newBillingFlowServices(t, db)
	ctx := context.Background()
	cleanupBillingFlow(t, db)
	t.Cleanup(func() { cleanupBillingFlow(t, db) })

	plan := newDueDateTestPlan(t, planSvc, "s1b_zero_update")

	err := planSvc.Update(ctx, plan.ID, PlanRequest{
		Name: "T3到期测试套餐", PriceCents: 0, ModuleCodes: []string{"base", "growth"},
	}, "t3_ops", "127.0.0.1")
	require.ErrorIs(t, err, ErrPlanPriceNotPositive)

	stored, err := repository.NewPlanRepo(db).FindByID(ctx, plan.ID)
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.EqualValues(t, 100000, stored.PriceCents, "被拒绝的改价不得覆盖原价格")
}

// 负数价格与 0 元同样非法。
func TestPlan_NegativePriceRejected(t *testing.T) {
	db := openPlatformTestDB(t)
	planSvc, _, _ := newBillingFlowServices(t, db)
	ctx := context.Background()
	cleanupBillingFlow(t, db)
	t.Cleanup(func() { cleanupBillingFlow(t, db) })

	_, err := planSvc.Create(ctx, PlanRequest{
		Code: "s1b_negative_price", Name: "T3负价套餐", PriceCents: -1, ModuleCodes: []string{"base"},
	}, "t3_ops", "127.0.0.1")
	require.ErrorIs(t, err, ErrPlanPriceNotPositive)
}
