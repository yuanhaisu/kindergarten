package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

// manifest 是对外契约：每条待修改授权一行（可精确回滚）、固定 schema、
// UTF-8、Asia/Shanghai 墙钟时间、落盘后给出 SHA256。
func TestWriteManifestHasFixedSchemaAndSHA(t *testing.T) {
	loc := mustShanghai(t)
	plan := &Plan{
		Selected: []TrialGroup{
			{KindergartenID: 101, Name: "小星星幼儿园", TrialCount: 2},
			{KindergartenID: 102, Name: "阳光幼儿园", TrialCount: 1},
		},
		Trials: []TrialRow{
			{KindergartenID: 101, EntitlementID: 9001, ExpiresAt: time.Date(2026, 12, 6, 15, 4, 5, 0, loc)},
			{KindergartenID: 101, EntitlementID: 9002, ExpiresAt: time.Date(2026, 12, 9, 15, 4, 5, 0, loc)},
			{KindergartenID: 102, EntitlementID: 9003, ExpiresAt: time.Date(2026, 12, 7, 9, 0, 0, 0, loc)},
		},
	}

	path := filepath.Join(t.TempDir(), "extendtrials-manifest.csv")
	sha, err := WriteManifest(path, plan)
	require.NoError(t, err)

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.True(t, utf8.Valid(raw), "manifest 必须是 UTF-8")

	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	require.Equal(t, []string{
		"# extendtrials-manifest v1",
		"kindergarten_id,entitlement_id,old_expires_at",
		"101,9001,2026-12-06 15:04:05",
		"101,9002,2026-12-09 15:04:05",
		"102,9003,2026-12-07 09:00:00",
	}, lines)

	sum := sha256.Sum256(raw)
	require.Equal(t, hex.EncodeToString(sum[:]), sha)
}

func TestWriteManifestRefusesEmptyPlan(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.csv")
	_, err := WriteManifest(path, &Plan{})
	require.ErrorIs(t, err, ErrEmptyPlan)

	_, statErr := os.Stat(path)
	require.True(t, os.IsNotExist(statErr), "空结果不应生成一个看起来有效的 manifest")
}

// 园所汇总另存：manifest 只承担逐条回滚，汇总文件承担人读的园所视图。
func TestWriteSummaryListsKindergartenTotals(t *testing.T) {
	plan := &Plan{Selected: []TrialGroup{
		{KindergartenID: 101, Name: "小星星,幼儿园", TrialCount: 4},
		{KindergartenID: 102, Name: "阳光幼儿园", TrialCount: 2},
	}}

	path := filepath.Join(t.TempDir(), "summary.csv")
	require.NoError(t, WriteSummary(path, plan))

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, []string{
		"kindergarten_id,name,trial_count",
		`101,"小星星,幼儿园",4`,
		"102,阳光幼儿园,2",
	}, strings.Split(strings.TrimRight(string(raw), "\n"), "\n"))
}
