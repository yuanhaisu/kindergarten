package main

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

// apply 只处理 manifest 中的授权；dry-run 之后才出现的 trial 行不在清单里，不得被改动。
func TestRunApplyExtendsOnlyManifestRows(t *testing.T) {
	db := openScratchDB(t)
	loc := mustShanghai(t)
	insertEligibleKindergarten(t, db, 501, "待延长园", loc)

	opts := testOptions(loc)
	opts.ManifestPath = filepath.Join(t.TempDir(), "manifest.csv")
	_, err := run(context.Background(), db, opts)
	require.NoError(t, err)

	// 清单外的园所：在 dry-run 之后才出现，apply 不得顺带延长。
	created := time.Date(2026, 10, 1, 10, 0, 0, 0, loc)
	insertKindergarten(t, db, 502, "清单外园", &created)
	outside := time.Date(2026, 12, 6, 15, 4, 5, 0, loc)
	insertEntitlement(t, db, 5021, 502, "growth", "trial", nil,
		time.Date(2026, 11, 6, 15, 4, 5, 0, loc), outside)

	t.Setenv("EXTENDTRIALS_ALLOW", "1")
	applyOpts := opts
	applyOpts.Apply = true
	applyOpts.Operator = "ops-test"
	_, err = run(context.Background(), db, applyOpts)
	require.NoError(t, err)

	expiries := entitlementExpiryMap(t, db)
	for _, id := range []uint64{5011, 5012, 5013, 5014} {
		require.True(t, expiries[id].Equal(opts.TargetExpiry),
			"授权 %d 应延至 %s，实际 %s", id, opts.TargetExpiry, expiries[id])
	}
	require.True(t, expiries[5021].Equal(outside),
		"清单外授权 5021 不得变化，实际 %s", expiries[5021])
}

// 写库必须留痕：一条 platform_operation_logs，含操作人、影响园所数/行数。
func TestRunApplyWritesPlatformOperationLog(t *testing.T) {
	db := openScratchDB(t)
	loc := mustShanghai(t)
	insertEligibleKindergarten(t, db, 503, "审计园", loc)

	opts := testOptions(loc)
	opts.ManifestPath = filepath.Join(t.TempDir(), "manifest.csv")
	_, err := run(context.Background(), db, opts)
	require.NoError(t, err)

	t.Setenv("EXTENDTRIALS_ALLOW", "1")
	applyOpts := opts
	applyOpts.Apply = true
	applyOpts.Operator = "ops-张三"
	res, err := run(context.Background(), db, applyOpts)
	require.NoError(t, err)
	require.Equal(t, 4, res.AffectedRows)
	require.Equal(t, 1, res.AffectedKindergartens)
	require.NotZero(t, res.LogID)

	var logs []domain.PlatformOperationLog
	require.NoError(t, db.Find(&logs).Error)
	require.Len(t, logs, 1)
	require.Equal(t, "ops-张三", logs[0].Operator)
	require.Equal(t, "billing", logs[0].Module)
	require.Equal(t, "extend_trials", logs[0].Action)
	require.Equal(t, res.LogID, logs[0].ID)

	var detail map[string]any
	require.NoError(t, json.Unmarshal([]byte(logs[0].Detail), &detail))
	require.EqualValues(t, 1, detail["kindergartens"])
	require.EqualValues(t, 4, detail["rows"])
	require.Equal(t, []any{float64(503)}, detail["kindergarten_ids"])
	require.Equal(t, opts.TargetExpiry.Format(manifestTimeLayout), detail["target_expiry"])
	require.Equal(t, res.ManifestSHA256, detail["manifest_sha256"])
}

// 重复 apply 必须幂等：影响行数为 0，日期不变，且第二次也留痕（rows=0）。
func TestRunApplyIsIdempotent(t *testing.T) {
	db := openScratchDB(t)
	loc := mustShanghai(t)
	insertEligibleKindergarten(t, db, 504, "幂等园", loc)

	opts := testOptions(loc)
	opts.ManifestPath = filepath.Join(t.TempDir(), "manifest.csv")
	_, err := run(context.Background(), db, opts)
	require.NoError(t, err)

	t.Setenv("EXTENDTRIALS_ALLOW", "1")
	applyOpts := opts
	applyOpts.Apply = true
	applyOpts.Operator = "ops-idem"

	first, err := run(context.Background(), db, applyOpts)
	require.NoError(t, err)
	require.Equal(t, 4, first.AffectedRows)
	afterFirst := allEntitlementExpiries(t, db)

	second, err := run(context.Background(), db, applyOpts)
	require.NoError(t, err)
	require.Equal(t, 0, second.AffectedRows)
	require.Equal(t, 4, second.UnchangedRows)
	require.Equal(t, afterFirst, allEntitlementExpiries(t, db), "重复 apply 不得改动任何日期")

	var logs []domain.PlatformOperationLog
	require.NoError(t, db.Order("id").Find(&logs).Error)
	require.Len(t, logs, 2, "每次写操作都要留痕")
	var detail map[string]any
	require.NoError(t, json.Unmarshal([]byte(logs[1].Detail), &detail))
	require.EqualValues(t, 0, detail["rows"])
}

// 多园所清单：影响园所数按实际被改动的园所去重统计，并写入日志详情。
func TestRunApplyCountsDistinctKindergartens(t *testing.T) {
	db := openScratchDB(t)
	loc := mustShanghai(t)
	insertEligibleKindergarten(t, db, 515, "多园甲", loc)
	insertEligibleKindergarten(t, db, 516, "多园乙", loc)

	opts := testOptions(loc)
	opts.ManifestPath = filepath.Join(t.TempDir(), "manifest.csv")
	_, err := run(context.Background(), db, opts)
	require.NoError(t, err)

	t.Setenv("EXTENDTRIALS_ALLOW", "1")
	applyOpts := opts
	applyOpts.Apply = true
	applyOpts.Operator = "ops-multi"
	res, err := run(context.Background(), db, applyOpts)
	require.NoError(t, err)
	require.Equal(t, 8, res.AffectedRows)
	require.Equal(t, 2, res.AffectedKindergartens)

	var log domain.PlatformOperationLog
	require.NoError(t, db.First(&log, res.LogID).Error)
	var detail map[string]any
	require.NoError(t, json.Unmarshal([]byte(log.Detail), &detail))
	require.EqualValues(t, 2, detail["kindergartens"])
	require.Equal(t, []any{float64(515), float64(516)}, detail["kindergarten_ids"])
}

// apply 必须在写事务内按最终口径复核清单园所：dry-run 与 apply 之间可能相隔数天，
// 期间出现新的在期订单/订阅、园所被停用或删除时，整体拒绝并点名园所，
// 绝不把试用叠加到已付费授权上。
func TestRunApplyRevalidatesEligibility(t *testing.T) {
	loc := mustShanghai(t)
	cases := []struct {
		name    string
		mutate  func(t *testing.T, db *gorm.DB, loc *time.Location)
		wantMsg string
	}{
		{
			name: "新增在期订单",
			mutate: func(t *testing.T, db *gorm.DB, loc *time.Location) {
				insertOrder(t, db, 5202, 520, "paid",
					time.Date(2026, 11, 1, 0, 0, 0, 0, loc), time.Date(2027, 6, 1, 0, 0, 0, 0, loc))
			},
			wantMsg: "存在在期订单",
		},
		{
			name: "新增未来起算订单",
			mutate: func(t *testing.T, db *gorm.DB, loc *time.Location) {
				insertOrder(t, db, 5203, 520, "paid",
					time.Date(2026, 12, 1, 0, 0, 0, 0, loc), time.Date(2027, 6, 1, 0, 0, 0, 0, loc))
			},
			wantMsg: "存在在期订单",
		},
		{
			name: "新增在期订阅授权",
			mutate: func(t *testing.T, db *gorm.DB, loc *time.Location) {
				insertEntitlement(t, db, 5299, 520, "growth", "subscription", nil,
					time.Date(2026, 11, 1, 0, 0, 0, 0, loc), time.Date(2027, 6, 1, 0, 0, 0, 0, loc))
			},
			wantMsg: "存在在期订阅授权",
		},
		{
			name: "园所停用",
			mutate: func(t *testing.T, db *gorm.DB, _ *time.Location) {
				require.NoError(t, db.Exec("UPDATE kindergartens SET status = ? WHERE id = ?",
					domain.KindergartenStatusSuspended, 520).Error)
			},
			wantMsg: "非在营",
		},
		{
			name: "园所删除",
			mutate: func(t *testing.T, db *gorm.DB, _ *time.Location) {
				require.NoError(t, db.Exec("UPDATE kindergartens SET deleted_at = NOW(3) WHERE id = ?", 520).Error)
			},
			wantMsg: "已删除",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := openScratchDB(t)
			insertEligibleKindergarten(t, db, 520, "复核园", loc)

			opts := testOptions(loc)
			opts.ManifestPath = filepath.Join(t.TempDir(), "manifest.csv")
			_, err := run(context.Background(), db, opts)
			require.NoError(t, err)

			tc.mutate(t, db, loc)
			before := allEntitlementExpiries(t, db)

			t.Setenv(allowWriteEnvVar, "1")
			applyOpts := opts
			applyOpts.Apply = true
			applyOpts.Operator = "ops-recheck"
			_, err = run(context.Background(), db, applyOpts)
			require.Error(t, err)
			require.Contains(t, err.Error(), "前置复核失败")
			require.Contains(t, err.Error(), tc.wantMsg)
			require.Contains(t, err.Error(), "520", "错误必须点名园所")
			require.Equal(t, before, allEntitlementExpiries(t, db), "复核失败不得留下部分更新")

			var logs []domain.PlatformOperationLog
			require.NoError(t, db.Find(&logs).Error)
			require.Empty(t, logs, "被拒绝的 apply 不写操作日志")
		})
	}
}

// 事务内任一步失败都必须整体回滚：这里让最后的操作日志写入失败，
// 此时授权已 UPDATE 过，必须随事务一起撤销。
func TestRunApplyRollsBackWhenLogWriteFails(t *testing.T) {
	db := openScratchDB(t)
	loc := mustShanghai(t)
	insertEligibleKindergarten(t, db, 508, "回滚事务园", loc)

	opts := testOptions(loc)
	opts.ManifestPath = filepath.Join(t.TempDir(), "manifest.csv")
	_, err := run(context.Background(), db, opts)
	require.NoError(t, err)
	before := allEntitlementExpiries(t, db)

	require.NoError(t, db.Exec("RENAME TABLE platform_operation_logs TO platform_operation_logs_broken").Error)
	defer func() {
		require.NoError(t, db.Exec("RENAME TABLE platform_operation_logs_broken TO platform_operation_logs").Error)
	}()

	t.Setenv("EXTENDTRIALS_ALLOW", "1")
	applyOpts := opts
	applyOpts.Apply = true
	applyOpts.Operator = "ops-rollback"
	_, err = run(context.Background(), db, applyOpts)
	require.Error(t, err)
	require.Contains(t, err.Error(), "平台操作日志")
	require.Equal(t, before, allEntitlementExpiries(t, db), "失败后不得留下部分更新")
}
