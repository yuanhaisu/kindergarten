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

// ErrEmptyPlan 表示没有待修改的授权。此时不落盘：一份只有表头的 manifest
// 看起来像合法产物，容易被后续 apply 误用。
var ErrEmptyPlan = errors.New("没有待延长的 trial 授权，未生成 manifest")

const (
	manifestMagic  = "# extendtrials-manifest v1"
	manifestHeader = "kindergarten_id,entitlement_id,old_expires_at"
	summaryHeader  = "kindergarten_id,name,trial_count"
	// manifestTimeLayout 是 manifest 的固定时间格式（Asia/Shanghai 墙钟时间）。
	manifestTimeLayout = "2006-01-02 15:04:05"
)

// WriteManifest 按固定 schema 逐条写出待修改授权，并返回文件字节的 SHA256。
//
// 每条 trial 授权一行：同园多条授权原到期日不一致时，revert 仍能逐条精确恢复。
// 园所汇总（name / trial_count）写入 WriteSummary 生成的旁路文件。
func WriteManifest(path string, plan *Plan) (string, error) {
	if plan == nil || len(plan.Trials) == 0 {
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
	if err := w.Write([]string{"kindergarten_id", "entitlement_id", "old_expires_at"}); err != nil {
		return "", fmt.Errorf("写 manifest 表头失败: %w", err)
	}
	for _, row := range plan.Trials {
		if err := w.Write([]string{
			strconv.FormatUint(row.KindergartenID, 10),
			strconv.FormatUint(row.EntitlementID, 10),
			row.ExpiresAt.Format(manifestTimeLayout),
		}); err != nil {
			return "", fmt.Errorf("写授权 %d 失败: %w", row.EntitlementID, err)
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

// WriteSummary 另存园所汇总（人读视图），不参与逐条回滚。
func WriteSummary(path string, plan *Plan) error {
	if path == "" {
		return errors.New("汇总文件路径为空")
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write([]string{"kindergarten_id", "name", "trial_count"}); err != nil {
		return fmt.Errorf("写汇总表头失败: %w", err)
	}
	for _, g := range plan.Selected {
		if err := w.Write([]string{
			strconv.FormatUint(g.KindergartenID, 10),
			g.Name,
			strconv.Itoa(g.TrialCount),
		}); err != nil {
			return fmt.Errorf("写园所 %d 汇总失败: %w", g.KindergartenID, err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return fmt.Errorf("生成汇总失败: %w", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("写出汇总失败: %w", err)
	}
	return nil
}
