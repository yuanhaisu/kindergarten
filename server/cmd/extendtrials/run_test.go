package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// 命令级行为：默认 dry-run 写 manifest 但不改库；空结果不落盘 manifest。
// （写库路径由 apply_test.go / revert_test.go 覆盖。）
func TestRunDryRunWritesManifestAndLeavesDBUnchanged(t *testing.T) {
	db := openScratchDB(t)
	loc := mustShanghai(t)
	insertEligibleKindergarten(t, db, 301, "预演园", loc)

	before := allEntitlementExpiries(t, db)

	opts := testOptions(loc)
	opts.ManifestPath = filepath.Join(t.TempDir(), "manifest.csv")
	res, err := run(context.Background(), db, opts)
	require.NoError(t, err)
	require.True(t, res.ManifestWritten)
	require.NotEmpty(t, res.ManifestSHA256)
	require.FileExists(t, opts.ManifestPath)
	require.FileExists(t, res.SHAFilePath)
	require.True(t, res.SummaryWritten)
	require.FileExists(t, res.SummaryPath)
	require.Equal(t, opts.ManifestPath+".summary.csv", res.SummaryPath)
	require.Equal(t, before, allEntitlementExpiries(t, db), "dry-run 不得写库")
}

func TestRunEmptyPlanWritesNoManifest(t *testing.T) {
	db := openScratchDB(t)
	loc := mustShanghai(t)

	opts := testOptions(loc)
	opts.ManifestPath = filepath.Join(t.TempDir(), "manifest.csv")
	res, err := run(context.Background(), db, opts)
	require.NoError(t, err)
	require.False(t, res.ManifestWritten)
	require.Empty(t, res.ManifestSHA256)
	require.False(t, res.SummaryWritten)
	_, statErr := os.Stat(opts.ManifestPath)
	require.True(t, os.IsNotExist(statErr), "空结果不得生成 manifest")
	_, statErr = os.Stat(opts.ManifestPath + ".summary.csv")
	require.True(t, os.IsNotExist(statErr), "空结果不得生成汇总文件")
}

func TestRunRequiresManifestPath(t *testing.T) {
	db := openScratchDB(t)
	loc := mustShanghai(t)
	insertEligibleKindergarten(t, db, 302, "缺路径园", loc)

	opts := testOptions(loc)
	_, err := run(context.Background(), db, opts)
	require.Error(t, err)
	require.Contains(t, err.Error(), "-manifest")
}

// insertEligibleKindergarten 构造一所符合口径的园所：4 条 trial 授权，到期 12-06～12-09。
func insertEligibleKindergarten(t *testing.T, db *gorm.DB, id uint64, name string, loc *time.Location) {
	t.Helper()
	created := time.Date(2026, 10, 1, 10, 0, 0, 0, loc)
	insertKindergarten(t, db, id, name, &created)
	start := time.Date(2026, 11, 6, 15, 4, 5, 0, loc)
	modules := []string{"growth", "album", "growth_report", "diet"}
	for i, module := range modules {
		expires := time.Date(2026, 12, 6, 15, 4, 5, 0, loc)
		if i >= 2 {
			expires = time.Date(2026, 12, 9, 15, 4, 5, 0, loc)
		}
		insertEntitlement(t, db, id*10+uint64(i)+1, id, module, "trial", nil, start, expires)
	}
}
