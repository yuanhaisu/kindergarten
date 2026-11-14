package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// 本文件的测试数据统一使用 S1Btier- 前缀（园所名 / 分销商 / 套餐编码 s6tier%）。
// 套餐编码刻意不以 s1b 开头：其它用例的清理语句 `code LIKE 's1b_%'` 中 `_` 是
// SQL 通配符，会误删 s1b 开头的任何编码，导致本文件的测试对其它用例的清理敏感。
// 清理只删除自己的数据，避免与并行运行的其它用例互相干扰。
func cleanupTierCoverage(t *testing.T, db *gorm.DB) {
	t.Helper()
	del := func(sql string, args ...any) {
		if err := db.Exec(sql, args...).Error; err != nil {
			t.Logf("cleanup %q: %v", sql, err)
		}
	}
	del("DELETE FROM commission_ledger WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE 'S1Btier-%')")
	del("DELETE FROM payments WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE 'S1Btier-%')")
	del("DELETE FROM entitlements WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE 'S1Btier-%')")
	del("DELETE FROM orders WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE 'S1Btier-%')")
	del("DELETE FROM students WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE 'S1Btier-%')")
	del("DELETE FROM classes WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE 'S1Btier-%')")
	del("DELETE FROM kindergartens WHERE name LIKE 'S1Btier-%'")
	del("DELETE FROM platform_operation_logs WHERE target_type = 'plan' AND target_id IN (SELECT id FROM plans WHERE code LIKE 's6tier%')")
	del("DELETE FROM plan_tiers WHERE plan_id IN (SELECT id FROM plans WHERE code LIKE 's6tier%')")
	del("DELETE FROM plan_modules WHERE plan_id IN (SELECT id FROM plans WHERE code LIKE 's6tier%')")
	del("DELETE FROM plans WHERE code LIKE 's6tier%'")
	del("DELETE FROM distributors WHERE name LIKE 'S1Btier-%'")
}

func newTierTestPlan(t *testing.T, planSvc *PlanService, code, name string) *domain.Plan {
	t.Helper()
	plan, err := planSvc.Create(context.Background(), PlanRequest{
		Code: code, Name: name, Tier: 1, PriceCents: 100000, TermMonths: 6,
		ModuleCodes: []string{"base", "growth"},
	}, "s6tier_ops", "127.0.0.1")
	require.NoError(t, err)
	return plan
}

// CAT-0009：档位必须从 0 开始，否则最低人数区间无人可用。
func TestPlanTier_RejectsTiersNotStartingAtZero(t *testing.T) {
	db := openPlatformTestDB(t)
	planSvc, _, _ := newBillingFlowServices(t, db)
	ctx := context.Background()
	cleanupTierCoverage(t, db)
	t.Cleanup(func() { cleanupTierCoverage(t, db) })

	plan := newTierTestPlan(t, planSvc, "s6tier_zero", "S1Btier-不从零版")

	err := planSvc.ReplaceTiers(ctx, plan.ID, []domain.PlanTier{
		{Tier: 1, MinStudents: 10, MaxStudents: 0, PriceMultiplier: 100},
	}, "s6tier_ops", "127.0.0.1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TIER_COVERAGE_NOT_FROM_ZERO")

	tiers, err := planSvc.ListTiers(ctx, plan.ID)
	require.NoError(t, err)
	assert.Empty(t, tiers, "非法档位配置不得落库")

	// 空档位同样没有覆盖任何人数区间。
	err = planSvc.ReplaceTiers(ctx, plan.ID, nil, "s6tier_ops", "127.0.0.1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TIER_COVERAGE_EMPTY")
}

// CAT-0010：相邻档位必须连续——后一档下限 = 前一档上限 + 1。
func TestPlanTier_RejectsCoverageGapOrOverlap(t *testing.T) {
	db := openPlatformTestDB(t)
	planSvc, _, _ := newBillingFlowServices(t, db)
	ctx := context.Background()
	cleanupTierCoverage(t, db)
	t.Cleanup(func() { cleanupTierCoverage(t, db) })

	cases := []struct {
		name     string
		wantCode string
		tiers    []domain.PlanTier
	}{
		{
			name:     "gap",
			wantCode: "TIER_COVERAGE_GAP",
			tiers: []domain.PlanTier{
				{Tier: 1, MinStudents: 0, MaxStudents: 100, PriceMultiplier: 100},
				{Tier: 2, MinStudents: 150, MaxStudents: 0, PriceMultiplier: 120},
			},
		},
		{
			name:     "overlap",
			wantCode: "TIER_COVERAGE_OVERLAP",
			tiers: []domain.PlanTier{
				{Tier: 1, MinStudents: 0, MaxStudents: 100, PriceMultiplier: 100},
				{Tier: 2, MinStudents: 80, MaxStudents: 0, PriceMultiplier: 120},
			},
		},
	}
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan := newTierTestPlan(t, planSvc, "s6tier_contig"+string(rune('a'+i)), "S1Btier-连续性"+tc.name)
			err := planSvc.ReplaceTiers(ctx, plan.ID, tc.tiers, "s6tier_ops", "127.0.0.1")
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantCode)
			tiers, err := planSvc.ListTiers(ctx, plan.ID)
			require.NoError(t, err)
			assert.Empty(t, tiers, "非法档位配置不得落库")
		})
	}
}

// CAT-0011：最高档必须开口（上限 0），否则超出上限的园所会落不到档位。
func TestPlanTier_RejectsBoundedTopOrUnboundedMiddle(t *testing.T) {
	db := openPlatformTestDB(t)
	planSvc, _, _ := newBillingFlowServices(t, db)
	ctx := context.Background()
	cleanupTierCoverage(t, db)
	t.Cleanup(func() { cleanupTierCoverage(t, db) })

	cases := []struct {
		name     string
		wantCode string
		tiers    []domain.PlanTier
	}{
		{
			name:     "bounded top",
			wantCode: "TIER_COVERAGE_BOUNDED_TOP",
			tiers: []domain.PlanTier{
				{Tier: 1, MinStudents: 0, MaxStudents: 100, PriceMultiplier: 100},
				{Tier: 2, MinStudents: 101, MaxStudents: 300, PriceMultiplier: 120},
			},
		},
		{
			name:     "unbounded not last",
			wantCode: "TIER_COVERAGE_UNBOUNDED_NOT_LAST",
			tiers: []domain.PlanTier{
				{Tier: 1, MinStudents: 0, MaxStudents: 0, PriceMultiplier: 100},
				{Tier: 2, MinStudents: 1, MaxStudents: 0, PriceMultiplier: 120},
			},
		},
	}
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan := newTierTestPlan(t, planSvc, "s6tier_top"+string(rune('a'+i)), "S1Btier-开口"+tc.name)
			err := planSvc.ReplaceTiers(ctx, plan.ID, tc.tiers, "s6tier_ops", "127.0.0.1")
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantCode)
			tiers, err := planSvc.ListTiers(ctx, plan.ID)
			require.NoError(t, err)
			assert.Empty(t, tiers, "非法档位配置不得落库")
		})
	}
}

// 合法档位配置可正常保存；输入顺序不影响连续性判定；被拒的保存不得清空已有配置。
func TestPlanTier_AcceptsContiguousConfig(t *testing.T) {
	db := openPlatformTestDB(t)
	planSvc, _, _ := newBillingFlowServices(t, db)
	ctx := context.Background()
	cleanupTierCoverage(t, db)
	t.Cleanup(func() { cleanupTierCoverage(t, db) })

	plan := newTierTestPlan(t, planSvc, "s6tier_valid", "S1Btier-合法版")
	require.NoError(t, planSvc.ReplaceTiers(ctx, plan.ID, []domain.PlanTier{
		{Tier: 2, MinStudents: 101, MaxStudents: 0, PriceMultiplier: 130},
		{Tier: 1, MinStudents: 0, MaxStudents: 100, PriceMultiplier: 0},
	}, "s6tier_ops", "127.0.0.1"), "乱序输入不应影响连续性判定")

	tiers, err := planSvc.ListTiers(ctx, plan.ID)
	require.NoError(t, err)
	require.Len(t, tiers, 2)
	assert.Equal(t, 1, tiers[0].Tier)
	assert.Equal(t, 0, tiers[0].MinStudents)
	assert.Equal(t, 100, tiers[0].MaxStudents)
	assert.Equal(t, 100, tiers[0].PriceMultiplier, "系数 0 按 100 落库")
	assert.Equal(t, 2, tiers[1].Tier)
	assert.Equal(t, 101, tiers[1].MinStudents)
	assert.Equal(t, 0, tiers[1].MaxStudents, "最高档必须开口")
	assert.Equal(t, 130, tiers[1].PriceMultiplier)

	// 被拒的保存不得清空已有配置
	err = planSvc.ReplaceTiers(ctx, plan.ID, []domain.PlanTier{
		{Tier: 1, MinStudents: 0, MaxStudents: 100, PriceMultiplier: 100},
		{Tier: 2, MinStudents: 0, MaxStudents: 0, PriceMultiplier: 130},
	}, "s6tier_ops", "127.0.0.1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TIER_COVERAGE_OVERLAP")
	tiers, err = planSvc.ListTiers(ctx, plan.ID)
	require.NoError(t, err)
	assert.Len(t, tiers, 2, "保存被拒时原有档位必须保留")
}

// CAT-0012 / ORD-0009：历史不完整档位配置落不到任何档位时，
// 建单必须返回可识别错误，不能静默回落到第一档 100% 生成低价订单。
func TestOrder_UnresolvedTierReturnsError(t *testing.T) {
	db := openPlatformTestDB(t)
	planSvc, orderSvc, _ := newBillingFlowServices(t, db)
	ctx := context.Background()
	cleanupTierCoverage(t, db)
	t.Cleanup(func() { cleanupTierCoverage(t, db) })

	kg := newTierTestKindergarten(t, db, "S1Btier-落档园", 5)
	plan := newTierTestPlan(t, planSvc, "s6tier_legacy", "S1Btier-历史缺口版")
	// 绕过保存校验直接写入历史遗留配置：只覆盖 0-2 名幼儿。
	require.NoError(t, db.Create(&domain.PlanTier{
		PlanID: plan.ID, Tier: 1, MinStudents: 0, MaxStudents: 2, PriceMultiplier: 100,
	}).Error)

	_, err := orderSvc.Create(ctx, CreateOrderRequest{KindergartenID: kg.ID, PlanID: plan.ID}, "s6tier_ops", "127.0.0.1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TIER_COVERAGE_UNRESOLVED")

	var orderCount int64
	require.NoError(t, db.Model(&domain.Order{}).Where("kindergarten_id = ?", kg.ID).Count(&orderCount).Error)
	assert.Zero(t, orderCount, "落不到档位时不得生成订单")
}

// 未配置档位的套餐仍按平价成交，不能因为「落不到档位」被误伤。
func TestOrder_NoTierConfigUsesFlatPrice(t *testing.T) {
	db := openPlatformTestDB(t)
	planSvc, orderSvc, _ := newBillingFlowServices(t, db)
	ctx := context.Background()
	cleanupTierCoverage(t, db)
	t.Cleanup(func() { cleanupTierCoverage(t, db) })

	kg := newTierTestKindergarten(t, db, "S1Btier-平价园", 5)
	plan := newTierTestPlan(t, planSvc, "s6tier_flat", "S1Btier-平价版")

	order, err := orderSvc.Create(ctx, CreateOrderRequest{KindergartenID: kg.ID, PlanID: plan.ID}, "s6tier_ops", "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, 1, order.StudentTier)
	assert.EqualValues(t, 100000, order.AmountDueCents, "无档位配置时按 100% 平价")

	var snap PlanSnapshot
	require.NoError(t, json.Unmarshal([]byte(order.PlanSnapshot), &snap))
	assert.Equal(t, 0, snap.TierMaxStudents, "无档位时快照上限为 0（无上限）")
}

// 档位变更只影响之后创建的订单，已成交订单继续使用成交快照（ADR-0013）。
func TestOrder_TierChangeOnlyAffectsNewOrders(t *testing.T) {
	db := openPlatformTestDB(t)
	planSvc, orderSvc, _ := newBillingFlowServices(t, db)
	ctx := context.Background()
	cleanupTierCoverage(t, db)
	t.Cleanup(func() { cleanupTierCoverage(t, db) })

	kg := newTierTestKindergarten(t, db, "S1Btier-改档园", 5)
	plan := newTierTestPlan(t, planSvc, "s6tier_change", "S1Btier-改档版")

	require.NoError(t, planSvc.ReplaceTiers(ctx, plan.ID, []domain.PlanTier{
		{Tier: 1, MinStudents: 0, MaxStudents: 10, PriceMultiplier: 100},
		{Tier: 2, MinStudents: 11, MaxStudents: 0, PriceMultiplier: 150},
	}, "s6tier_ops", "127.0.0.1"))
	first, err := orderSvc.Create(ctx, CreateOrderRequest{KindergartenID: kg.ID, PlanID: plan.ID}, "s6tier_ops", "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, 1, first.StudentTier)
	assert.EqualValues(t, 100000, first.AmountDueCents)

	// 改档：5 名幼儿从第一档进入第二档
	require.NoError(t, planSvc.ReplaceTiers(ctx, plan.ID, []domain.PlanTier{
		{Tier: 1, MinStudents: 0, MaxStudents: 2, PriceMultiplier: 100},
		{Tier: 2, MinStudents: 3, MaxStudents: 0, PriceMultiplier: 200},
	}, "s6tier_ops", "127.0.0.1"))
	second, err := orderSvc.Create(ctx, CreateOrderRequest{KindergartenID: kg.ID, PlanID: plan.ID}, "s6tier_ops", "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, 2, second.StudentTier, "新订单使用新档位")
	assert.EqualValues(t, 200000, second.AmountDueCents)

	again, err := orderSvc.Detail(ctx, first.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, again.PlanSnapshot.Tier, "老订单档位不变")
	assert.Equal(t, 10, again.PlanSnapshot.TierMaxStudents, "老订单沿用原档位上限")
	assert.EqualValues(t, 100000, again.AmountDueCents, "老订单金额不变")
}

func newTierTestKindergarten(t *testing.T, db *gorm.DB, name string, students int) *domain.Kindergarten {
	t.Helper()
	kg := &domain.Kindergarten{Name: name, Status: domain.KindergartenStatusActive}
	require.NoError(t, repository.NewKindergartenRepo(db).Create(context.Background(), kg))
	if students > 0 {
		seedStudents(t, db, kg.ID, students)
	}
	return kg
}
