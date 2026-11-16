package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// 选择口径：未删除、政策日前开通、存在 trial 授权（order_id 为空）。
func TestBuildPlanSelectsLegacyTrialsAndDoesNotWrite(t *testing.T) {
	db := openScratchDB(t)
	loc := mustShanghai(t)
	created := time.Date(2026, 10, 1, 10, 0, 0, 0, loc)

	insertKindergarten(t, db, 101, "小星星幼儿园", &created)
	insertEntitlement(t, db, 1, 101, "base", "gift", nil,
		time.Date(2000, 1, 1, 0, 0, 0, 0, loc), time.Date(2099, 12, 31, 23, 59, 59, 0, loc))
	insertEntitlement(t, db, 2, 101, "growth", "trial", nil,
		time.Date(2026, 11, 6, 15, 4, 5, 0, loc), time.Date(2026, 12, 6, 15, 4, 5, 0, loc))
	insertEntitlement(t, db, 3, 101, "album", "trial", nil,
		time.Date(2026, 11, 6, 15, 4, 5, 0, loc), time.Date(2026, 12, 6, 15, 4, 5, 0, loc))
	insertEntitlement(t, db, 4, 101, "growth_report", "trial", nil,
		time.Date(2026, 11, 6, 15, 4, 5, 0, loc), time.Date(2026, 12, 9, 15, 4, 5, 0, loc))
	insertEntitlement(t, db, 5, 101, "diet", "trial", nil,
		time.Date(2026, 11, 6, 15, 4, 5, 0, loc), time.Date(2026, 12, 9, 15, 4, 5, 0, loc))

	before := allEntitlementExpiries(t, db)

	plan, err := BuildPlan(context.Background(), db, testOptions(loc))
	require.NoError(t, err)

	require.Len(t, plan.Selected, 1)
	got := plan.Selected[0]
	require.Equal(t, uint64(101), got.KindergartenID)
	require.Equal(t, "小星星幼儿园", got.Name)
	require.Equal(t, 4, got.TrialCount)
	require.True(t, got.OldEarliest.Equal(time.Date(2026, 12, 6, 15, 4, 5, 0, loc)), "最早到期日 = %s", got.OldEarliest)
	require.True(t, got.OldLatest.Equal(time.Date(2026, 12, 9, 15, 4, 5, 0, loc)), "最晚到期日 = %s", got.OldLatest)
	require.Empty(t, plan.Excluded)
	require.Empty(t, plan.NullCreatedAt)

	// dry-run 阶段：BuildPlan 只读，数据库必须逐行不变。
	require.Equal(t, before, allEntitlementExpiries(t, db))
}
