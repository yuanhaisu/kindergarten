package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// 写操作必须显式确认：没有 EXTENDTRIALS_ALLOW=1 时 apply 与 revert 都拒绝。
func TestRunWriteRequiresExplicitOptIn(t *testing.T) {
	db := openScratchDB(t)
	loc := mustShanghai(t)
	insertEligibleKindergarten(t, db, 510, "开关园", loc)

	opts := testOptions(loc)
	opts.ManifestPath = filepath.Join(t.TempDir(), "manifest.csv")
	_, err := run(context.Background(), db, opts)
	require.NoError(t, err)
	before := allEntitlementExpiries(t, db)

	t.Setenv(allowWriteEnvVar, "")
	applyOpts := opts
	applyOpts.Apply = true
	applyOpts.Operator = "ops-guard"
	_, err = run(context.Background(), db, applyOpts)
	require.Error(t, err)
	require.Contains(t, err.Error(), allowWriteEnvVar)

	revertOpts := opts
	revertOpts.Revert = true
	revertOpts.Operator = "ops-guard"
	_, err = run(context.Background(), db, revertOpts)
	require.Error(t, err)
	require.Contains(t, err.Error(), allowWriteEnvVar)

	require.Equal(t, before, allEntitlementExpiries(t, db), "未确认的写操作不得改动数据")
}

// 写操作必须记录执行人。
func TestRunWriteRequiresOperator(t *testing.T) {
	db := openScratchDB(t)
	loc := mustShanghai(t)
	insertEligibleKindergarten(t, db, 511, "执行人园", loc)

	opts := testOptions(loc)
	opts.ManifestPath = filepath.Join(t.TempDir(), "manifest.csv")
	_, err := run(context.Background(), db, opts)
	require.NoError(t, err)
	before := allEntitlementExpiries(t, db)

	t.Setenv(allowWriteEnvVar, "1")
	applyOpts := opts
	applyOpts.Apply = true
	_, err = run(context.Background(), db, applyOpts)
	require.Error(t, err)
	require.Contains(t, err.Error(), "-operator")
	require.Equal(t, before, allEntitlementExpiries(t, db))
}

// manifest 在 dry-run 之后被改动（哪怕只改一行）必须被 SHA256 拦住。
func TestRunApplyRejectsTamperedManifest(t *testing.T) {
	db := openScratchDB(t)
	loc := mustShanghai(t)
	insertEligibleKindergarten(t, db, 512, "篡改园", loc)

	opts := testOptions(loc)
	opts.ManifestPath = filepath.Join(t.TempDir(), "manifest.csv")
	_, err := run(context.Background(), db, opts)
	require.NoError(t, err)
	before := allEntitlementExpiries(t, db)

	raw, err := os.ReadFile(opts.ManifestPath)
	require.NoError(t, err)
	tampered := string(raw) + "9999,9999,2026-12-06 15:04:05\n"
	require.NoError(t, os.WriteFile(opts.ManifestPath, []byte(tampered), 0o644))

	t.Setenv(allowWriteEnvVar, "1")
	applyOpts := opts
	applyOpts.Apply = true
	applyOpts.Operator = "ops-sha"
	_, err = run(context.Background(), db, applyOpts)
	require.Error(t, err)
	require.Contains(t, err.Error(), "SHA256 不匹配")
	require.Equal(t, before, allEntitlementExpiries(t, db), "校验失败不得写库")
}

// 版本行不符（即使 SHA256 自洽）同样拒绝。
func TestRunApplyRejectsUnknownManifestVersion(t *testing.T) {
	db := openScratchDB(t)
	loc := mustShanghai(t)
	insertEligibleKindergarten(t, db, 513, "版本园", loc)

	path := filepath.Join(t.TempDir(), "manifest.csv")
	content := "# extendtrials-manifest v2\n" +
		"kindergarten_id,entitlement_id,old_expires_at\n" +
		"513,5131,2026-12-06 15:04:05\n"
	writeRawManifest(t, path, content)
	before := allEntitlementExpiries(t, db)

	t.Setenv(allowWriteEnvVar, "1")
	opts := testOptions(loc)
	opts.ManifestPath = path
	opts.Apply = true
	opts.Operator = "ops-version"
	_, err := run(context.Background(), db, opts)
	require.Error(t, err)
	require.Contains(t, err.Error(), "版本不匹配")
	require.Equal(t, before, allEntitlementExpiries(t, db))
}

// 缺失/空/只有表头的 manifest 都必须拒绝，不能当成「无事可做」静默通过。
func TestRunWriteRejectsMissingOrEmptyManifest(t *testing.T) {
	db := openScratchDB(t)
	loc := mustShanghai(t)
	t.Setenv(allowWriteEnvVar, "1")

	opts := testOptions(loc)
	opts.Apply = true
	opts.Operator = "ops-file"

	opts.ManifestPath = filepath.Join(t.TempDir(), "missing.csv")
	_, err := run(context.Background(), db, opts)
	require.Error(t, err)
	require.Contains(t, err.Error(), "manifest 不存在")

	emptyPath := filepath.Join(t.TempDir(), "empty.csv")
	writeRawManifest(t, emptyPath, "")
	opts.ManifestPath = emptyPath
	_, err = run(context.Background(), db, opts)
	require.Error(t, err)
	require.Contains(t, err.Error(), "manifest 为空")

	headerOnly := filepath.Join(t.TempDir(), "header-only.csv")
	writeRawManifest(t, headerOnly, manifestMagic+"\n"+manifestHeader+"\n")
	opts.ManifestPath = headerOnly
	_, err = run(context.Background(), db, opts)
	require.Error(t, err)
	require.Contains(t, err.Error(), "没有任何授权行")
}

// 只延长不回退：manifest 内的行若已被改到目标日之后，apply 跳过它而不是改回去。
func TestRunApplyNeverShortensRowsBeyondTarget(t *testing.T) {
	db := openScratchDB(t)
	loc := mustShanghai(t)
	insertEligibleKindergarten(t, db, 514, "已更晚园", loc)

	opts := testOptions(loc)
	opts.ManifestPath = filepath.Join(t.TempDir(), "manifest.csv")
	_, err := run(context.Background(), db, opts)
	require.NoError(t, err)

	later := time.Date(2027, 6, 1, 0, 0, 0, 0, loc)
	require.NoError(t, db.Exec("UPDATE entitlements SET expires_at = ? WHERE id = ?", later, 5142).Error)

	t.Setenv(allowWriteEnvVar, "1")
	applyOpts := opts
	applyOpts.Apply = true
	applyOpts.Operator = "ops-noshorten"
	res, err := run(context.Background(), db, applyOpts)
	require.NoError(t, err)
	require.Equal(t, 3, res.AffectedRows)
	require.Equal(t, 1, res.UnchangedRows)

	expiries := entitlementExpiryMap(t, db)
	require.True(t, expiries[5142].Equal(later), "已更晚的到期日不得被改回目标日，实际 %s", expiries[5142])
	for _, id := range []uint64{5141, 5143, 5144} {
		require.True(t, expiries[id].Equal(opts.TargetExpiry), "授权 %d 应延至目标日", id)
	}
}

// -apply 与 -revert 是互斥模式。
func TestRunRejectsApplyAndRevertTogether(t *testing.T) {
	db := openScratchDB(t)
	loc := mustShanghai(t)
	t.Setenv(allowWriteEnvVar, "1")

	opts := testOptions(loc)
	opts.ManifestPath = filepath.Join(t.TempDir(), "manifest.csv")
	opts.Apply = true
	opts.Revert = true
	opts.Operator = "ops-both"
	_, err := run(context.Background(), db, opts)
	require.Error(t, err)
	require.Contains(t, err.Error(), "不能同时使用")
}

// writeRawManifest 写入故意不合规的清单，并同步生成 .sha256 侧车，
// 让测试能穿过哈希校验、命中后续的内容校验。
func writeRawManifest(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	sum := sha256.Sum256([]byte(content))
	_, err := WriteManifestSHA256(path, hex.EncodeToString(sum[:]))
	require.NoError(t, err)
}
