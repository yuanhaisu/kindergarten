package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"
)

// ErrEmptyPlan 表示没有待修改的园所。此时不落盘：一份只有表头的 manifest
// 看起来像合法产物，容易被后续 apply 误用。
var ErrEmptyPlan = errors.New("没有待延长的园所，未生成 manifest")

const (
	manifestMagic  = "# extendtrials-manifest v1"
	manifestHeader = "kindergarten_id,name,trial_count,old_expires_at"
	// manifestTimeLayout 是 manifest 的固定时间格式（Asia/Shanghai 墙钟时间）。
	manifestTimeLayout = "2006-01-02 15:04:05"
)

// WriteManifest 按固定 schema 写出 manifest 并返回文件字节的 SHA256。
//
// 列旧到期日取该园所待修改 trial 行中最早的一条：manifest 每园只有一列，
// 无法表达行间差异；ADDR-0018 回填时同一园所的 trial 到期日一致，取最早值
// 是保守的恢复基准（第一个功能失效时刻）。若同园所行间不一致，dry-run 输出会告警。
func WriteManifest(path string, plan *Plan) (string, error) {
	if plan == nil || len(plan.Selected) == 0 {
		return "", ErrEmptyPlan
	}
	if path == "" {
		return "", errors.New("manifest 路径为空")
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write([]string{manifestMagic}); err != nil {
		return "", fmt.Errorf("写 manifest 首行失败: %w", err)
	}
	if err := w.Write([]string{"kindergarten_id", "name", "trial_count", "old_expires_at"}); err != nil {
		return "", fmt.Errorf("写 manifest 表头失败: %w", err)
	}
	for _, g := range plan.Selected {
		if err := w.Write([]string{
			strconv.FormatUint(g.KindergartenID, 10),
			g.Name,
			strconv.Itoa(g.TrialCount),
			g.OldEarliest.Format(manifestTimeLayout),
		}); err != nil {
			return "", fmt.Errorf("写园所 %d 失败: %w", g.KindergartenID, err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", fmt.Errorf("生成 manifest 失败: %w", err)
	}

	raw := buf.Bytes()
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return "", fmt.Errorf("写出 manifest 失败: %w", err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

// hasHeterogeneousExpiry 报告该园待修改行的原到期日是否不一致。
// 不一致时单列 manifest 无法精确回滚，需要人工确认（ADR-0018 回填数据本应一致）。
func hasHeterogeneousExpiry(g TrialGroup) bool {
	return !g.OldEarliest.Equal(g.OldLatest)
}
