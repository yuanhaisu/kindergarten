package main

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/domain"
)

// revert 按 manifest 逐条恢复原始到期日（同园到期日不同也必须各回各的）。
func TestRunRevertRestoresManifestOriginalExpiries(t *testing.T) {
	db := openScratchDB(t)
	loc := mustShanghai(t)
	insertEligibleKindergarten(t, db, 505, "回滚园", loc)

	opts := testOptions(loc)
	opts.ManifestPath = filepath.Join(t.TempDir(), "manifest.csv")
	_, err := run(context.Background(), db, opts)
	require.NoError(t, err)

	t.Setenv("EXTENDTRIALS_ALLOW", "1")
	applyOpts := opts
	applyOpts.Apply = true
	applyOpts.Operator = "ops-apply"
	_, err = run(context.Background(), db, applyOpts)
	require.NoError(t, err)

	revertOpts := opts
	revertOpts.Revert = true
	revertOpts.Operator = "ops-revert"
	res, err := run(context.Background(), db, revertOpts)
	require.NoError(t, err)
	require.Equal(t, "revert", res.Mode)
	require.Equal(t, 4, res.AffectedRows)
	require.Equal(t, 1, res.AffectedKindergartens)
	require.NotZero(t, res.LogID)

	want := map[uint64]time.Time{
		5051: time.Date(2026, 12, 6, 15, 4, 5, 0, loc),
		5052: time.Date(2026, 12, 6, 15, 4, 5, 0, loc),
		5053: time.Date(2026, 12, 9, 15, 4, 5, 0, loc),
		5054: time.Date(2026, 12, 9, 15, 4, 5, 0, loc),
	}
	expiries := entitlementExpiryMap(t, db)
	for id, expected := range want {
		require.True(t, expiries[id].Equal(expected), "授权 %d 应恢复到 %s，实际 %s", id, expected, expiries[id])
	}

	var logs []domain.PlatformOperationLog
	require.NoError(t, db.Order("id").Find(&logs).Error)
	require.Len(t, logs, 2)
	require.Equal(t, "revert_trials", logs[1].Action)
	var detail map[string]any
	require.NoError(t, json.Unmarshal([]byte(logs[1].Detail), &detail))
	require.Equal(t, "revert", detail["mode"])
	require.EqualValues(t, 4, detail["rows"])
}

// 第二次 revert 是 no-op：0 行受影响、日期不变。
func TestRunRevertIsIdempotent(t *testing.T) {
	db := openScratchDB(t)
	loc := mustShanghai(t)
	insertEligibleKindergarten(t, db, 506, "回滚幂等园", loc)

	opts := testOptions(loc)
	opts.ManifestPath = filepath.Join(t.TempDir(), "manifest.csv")
	_, err := run(context.Background(), db, opts)
	require.NoError(t, err)

	t.Setenv("EXTENDTRIALS_ALLOW", "1")
	applyOpts := opts
	applyOpts.Apply = true
	applyOpts.Operator = "ops-apply"
	_, err = run(context.Background(), db, applyOpts)
	require.NoError(t, err)

	revertOpts := opts
	revertOpts.Revert = true
	revertOpts.Operator = "ops-revert"
	_, err = run(context.Background(), db, revertOpts)
	require.NoError(t, err)
	afterFirst := allEntitlementExpiries(t, db)

	second, err := run(context.Background(), db, revertOpts)
	require.NoError(t, err)
	require.Equal(t, 0, second.AffectedRows)
	require.Equal(t, 4, second.UnchangedRows)
	require.Equal(t, afterFirst, allEntitlementExpiries(t, db), "重复 revert 不得改动任何日期")
}

// apply 之后有授权被合法调整（当前值既不是目标日也不是原始值）时，
// revert 必须整体拒绝并在错误里点名，不得把其他行先恢复了再报错。
func TestRunRevertRefusesWhenRowChangedAfterApply(t *testing.T) {
	db := openScratchDB(t)
	loc := mustShanghai(t)
	insertEligibleKindergarten(t, db, 507, "已有后续变更园", loc)

	opts := testOptions(loc)
	opts.ManifestPath = filepath.Join(t.TempDir(), "manifest.csv")
	_, err := run(context.Background(), db, opts)
	require.NoError(t, err)

	t.Setenv("EXTENDTRIALS_ALLOW", "1")
	applyOpts := opts
	applyOpts.Apply = true
	applyOpts.Operator = "ops-apply"
	_, err = run(context.Background(), db, applyOpts)
	require.NoError(t, err)

	later := time.Date(2027, 2, 15, 12, 0, 0, 0, loc)
	require.NoError(t, db.Exec("UPDATE entitlements SET expires_at = ? WHERE id = ?", later, 5072).Error)
	before := allEntitlementExpiries(t, db)

	revertOpts := opts
	revertOpts.Revert = true
	revertOpts.Operator = "ops-revert"
	_, err = run(context.Background(), db, revertOpts)
	require.Error(t, err)
	require.Contains(t, err.Error(), "拒绝回滚")
	require.Contains(t, err.Error(), "5072")

	require.Equal(t, before, allEntitlementExpiries(t, db), "任一漂移都必须整体回滚：其他行也不得被恢复")

	var logs []domain.PlatformOperationLog
	require.NoError(t, db.Find(&logs).Error)
	require.Len(t, logs, 1, "被拒绝的回滚不写操作日志")
}
