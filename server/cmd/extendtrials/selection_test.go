package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// 排除口径与遗留行报告：任何一所园所的去向都必须可解释，不能静默丢弃。
func TestBuildPlanExcludesPaidAndReportsLegacyRows(t *testing.T) {
	db := openScratchDB(t)
	loc := mustShanghai(t)
	early := time.Date(2025, 9, 1, 10, 0, 0, 0, loc)
	policyDayEnd := time.Date(2026, 11, 15, 23, 59, 59, 0, loc)
	afterPolicy := time.Date(2026, 11, 16, 0, 0, 1, 0, loc)
	trialStart := time.Date(2026, 11, 6, 15, 4, 5, 0, loc)
	trialEnd := time.Date(2026, 12, 6, 15, 4, 5, 0, loc)
	lateTrialEnd := time.Date(2027, 2, 1, 0, 0, 0, 0, loc)

	// 201 政策日当天最后一秒开通：应命中。
	insertKindergarten(t, db, 201, "政策日当天园", &policyDayEnd)
	insertEntitlement(t, db, 2011, 201, "growth", "trial", nil, trialStart, trialEnd)

	// 202 政策日次日开通：排除。
	insertKindergarten(t, db, 202, "新政园", &afterPolicy)
	insertEntitlement(t, db, 2021, 202, "growth", "trial", nil, afterPolicy, lateTrialEnd)

	// 203 有在期未付款订单（未取消）：排除。
	insertKindergarten(t, db, 203, "已下单园", &early)
	insertEntitlement(t, db, 2031, 203, "growth", "trial", nil, trialStart, trialEnd)
	insertOrder(t, db, 2032, 203, "unpaid", early, lateTrialEnd)

	// 204 只有已取消订单：不阻断，应命中。
	insertKindergarten(t, db, 204, "取消订单园", &early)
	insertEntitlement(t, db, 2041, 204, "growth", "trial", nil, trialStart, trialEnd)
	insertOrder(t, db, 2042, 204, "cancelled", early, lateTrialEnd)

	// 213 有未取消但已过生效窗口的历史订单：不阻断，应命中。
	insertKindergarten(t, db, 213, "历史订单园", &early)
	insertEntitlement(t, db, 2131, 213, "growth", "trial", nil, trialStart, trialEnd)
	insertOrder(t, db, 2132, 213, "expired", early, time.Date(2025, 12, 31, 23, 59, 59, 0, loc))

	// 205 有在期订阅授权：排除。
	insertKindergarten(t, db, 205, "已付费园", &early)
	insertEntitlement(t, db, 2051, 205, "growth", "trial", nil, trialStart, trialEnd)
	insertEntitlement(t, db, 2052, 205, "growth", "subscription", nil, early, lateTrialEnd)

	// 206 订阅已过期：不阻断，应命中。
	insertKindergarten(t, db, 206, "订阅已过期园", &early)
	insertEntitlement(t, db, 2061, 206, "growth", "trial", nil, trialStart, trialEnd)
	insertEntitlement(t, db, 2062, 206, "growth", "subscription", nil,
		time.Date(2025, 1, 1, 0, 0, 0, 0, loc), time.Date(2025, 12, 31, 0, 0, 0, 0, loc))

	// 207 created_at 为空：单独报告。
	insertKindergarten(t, db, 207, "遗留空时间园", nil)
	insertEntitlement(t, db, 2071, 207, "growth", "trial", nil, trialStart, trialEnd)

	// 208 无 trial 授权：排除。
	insertKindergarten(t, db, 208, "无试用园", &early)
	insertEntitlement(t, db, 2081, 208, "base", "gift", nil, early, lateTrialEnd)

	// 209 已软删：完全不扫描。
	deletedAt := time.Date(2026, 10, 1, 0, 0, 0, 0, loc)
	insertSoftDeletedKindergarten(t, db, 209, "已删除园", deletedAt)
	insertEntitlement(t, db, 2091, 209, "growth", "trial", nil, trialStart, trialEnd)

	// 210 trial 已在目标日之后：无需修改。
	insertKindergarten(t, db, 210, "已延长园", &early)
	insertEntitlement(t, db, 2101, 210, "growth", "trial", nil, trialStart, lateTrialEnd)

	// 211 只有软删的 trial 授权：视为无 trial。
	insertKindergarten(t, db, 211, "试用已软删园", &early)
	err := db.Exec(
		"INSERT INTO entitlements (id, created_at, updated_at, deleted_at, kindergarten_id, module_code, source, order_id, starts_at, expires_at) VALUES (?, NOW(3), NOW(3), NOW(3), 211, 'growth', 'trial', NULL, ?, ?)",
		2111, trialStart, trialEnd,
	).Error
	require.NoError(t, err)

	// 212 trial 挂在订单上（order_id 非空）：不算存量试用。
	insertKindergarten(t, db, 212, "订单试用园", &early)
	orderID := uint64(2121)
	insertEntitlement(t, db, 2122, 212, "growth", "trial", &orderID, trialStart, trialEnd)

	plan, err := BuildPlan(context.Background(), db, testOptions(loc))
	require.NoError(t, err)

	require.Equal(t, 12, plan.Scanned, "软删园所不扫描")
	require.Equal(t, []uint64{201, 204, 206, 213}, selectedIDs(plan))
	require.Equal(t, []LegacyKindergarten{{KindergartenID: 207, Name: "遗留空时间园"}}, plan.NullCreatedAt)

	reasons := map[uint64]ExclusionReason{}
	for _, ex := range plan.Excluded {
		reasons[ex.KindergartenID] = ex.Reason
	}
	require.Equal(t, map[uint64]ExclusionReason{
		202: ReasonCreatedAfterPolicy,
		203: ReasonUncancelledOrder,
		205: ReasonActiveSubscription,
		208: ReasonNoTrial,
		210: ReasonNothingToExtend,
		211: ReasonNoTrial,
		212: ReasonNoTrial,
	}, reasons)
}

func selectedIDs(plan *Plan) []uint64 {
	ids := make([]uint64, 0, len(plan.Selected))
	for _, g := range plan.Selected {
		ids = append(ids, g.KindergartenID)
	}
	return ids
}
