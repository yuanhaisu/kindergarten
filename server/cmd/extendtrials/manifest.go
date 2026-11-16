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
	"strings"
	"time"
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

// 表头只保留字符串常量一处定义，写文件时按逗号拆成 CSV 字段。
var (
	manifestHeaderFields = strings.Split(manifestHeader, ",")
	summaryHeaderFields  = strings.Split(summaryHeader, ",")
)

// ensureManifestTargetWritable 拒绝覆盖已存在的 manifest 或其 .sha256 侧车：
// 覆盖会毁掉运营已确认、用于回滚的那份清单。重新生成须显式传 -force。
func ensureManifestTargetWritable(path string, force bool) error {
	if force {
		return nil
	}
	for _, p := range []string{path, path + manifestSHASuffix} {
		_, err := os.Stat(p)
		if err == nil {
			return fmt.Errorf("%s 已存在，拒绝覆盖（确认要重新生成时加 -force）", p)
		}
		if !os.IsNotExist(err) {
			return fmt.Errorf("检查 %s 失败: %w", p, err)
		}
	}
	return nil
}

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
	if err := w.Write(manifestHeaderFields); err != nil {
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
// manifestSHASuffix 是 dry-run 写出的校验值侧车文件后缀。
// apply/revert 只读它，不接受手工转录的哈希：侧车必须与 manifest 成对出现。
const manifestSHASuffix = ".sha256"

// Manifest 是 dry-run 落盘、apply/revert 读取的运营确认清单。
type Manifest struct {
	Path   string
	SHA256 string
	Rows   []TrialRow
}

// Kindergartens 返回清单涉及的园所 ID（升序去重）。
func (m *Manifest) Kindergartens() []uint64 {
	set := make(map[uint64]bool, len(m.Rows))
	for _, r := range m.Rows {
		set[r.KindergartenID] = true
	}
	return sortedIDs(set)
}

// WriteManifestSHA256 写出 manifest 校验值侧车文件，返回侧车路径。
func WriteManifestSHA256(path, sha string) (string, error) {
	if path == "" {
		return "", errors.New("manifest 路径为空")
	}
	if sha == "" {
		return "", errors.New("manifest SHA256 为空")
	}
	side := path + manifestSHASuffix
	if err := os.WriteFile(side, []byte(sha+"\n"), 0o644); err != nil {
		return "", fmt.Errorf("写出 manifest 校验值失败: %w", err)
	}
	return side, nil
}

// ResolveManifestSHA 读取 <manifest>.sha256 侧车文件里的期望哈希。
func ResolveManifestSHA(path string) (string, error) {
	side := path + manifestSHASuffix
	raw, err := os.ReadFile(side)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf(
				"缺少 manifest 校验值：%s 不存在（请确认它与 dry-run 生成的 manifest 在同一目录）", side)
		}
		return "", fmt.Errorf("读取 manifest 校验值 %s 失败: %w", side, err)
	}
	v := strings.ToLower(strings.TrimSpace(string(raw)))
	if _, err := hex.DecodeString(v); err != nil || len(v) != sha256.Size*2 {
		return "", fmt.Errorf("manifest 校验值文件 %s 内容不是合法的 SHA256: %q", side, strings.TrimSpace(string(raw)))
	}
	return v, nil
}

// ReadManifest 读取并校验 manifest：版本行、表头、逐行格式与 SHA256。
// 空文件、只有表头的空清单、重复授权一律拒绝——残缺文件看起来像合法产物，
// 放行会让 apply/revert 在错误范围内执行。
func ReadManifest(path, expectedSHA string) (*Manifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("manifest 不存在：%s（请先用 dry-run 生成并经运营确认）", path)
		}
		return nil, fmt.Errorf("读取 manifest %s 失败: %w", path, err)
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, fmt.Errorf("manifest 为空：%s", path)
	}

	sum := sha256.Sum256(raw)
	actual := hex.EncodeToString(sum[:])
	if expectedSHA != "" && !strings.EqualFold(actual, expectedSHA) {
		return nil, fmt.Errorf(
			"manifest SHA256 不匹配：期望 %s，实际 %s（文件在 dry-run 之后被修改？）", expectedSHA, actual)
	}

	r := csv.NewReader(bytes.NewReader(raw))
	r.FieldsPerRecord = -1
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("解析 manifest %s 失败: %w", path, err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("manifest 结构不完整：%s（缺少版本行或表头）", path)
	}
	if len(records[0]) != 1 || records[0][0] != manifestMagic {
		return nil, fmt.Errorf("manifest 版本不匹配：期望 %q，实际 %q", manifestMagic, strings.Join(records[0], ","))
	}
	if got := strings.Join(records[1], ","); got != manifestHeader {
		return nil, fmt.Errorf("manifest 表头不匹配：期望 %q，实际 %q", manifestHeader, got)
	}
	if len(records) == 2 {
		return nil, fmt.Errorf("manifest 没有任何授权行（空清单）：%s", path)
	}

	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return nil, fmt.Errorf("加载 Asia/Shanghai 失败（容器缺 tzdata？）: %w", err)
	}
	rows := make([]TrialRow, 0, len(records)-2)
	seen := make(map[uint64]bool, len(records)-2)
	for i, rec := range records[2:] {
		line := i + 3
		if len(rec) != 3 {
			return nil, fmt.Errorf("manifest 第 %d 行列数不为 3: %q", line, strings.Join(rec, ","))
		}
		kgID, err := strconv.ParseUint(strings.TrimSpace(rec[0]), 10, 64)
		if err != nil || kgID == 0 {
			return nil, fmt.Errorf("manifest 第 %d 行 kindergarten_id 非法: %q", line, rec[0])
		}
		entID, err := strconv.ParseUint(strings.TrimSpace(rec[1]), 10, 64)
		if err != nil || entID == 0 {
			return nil, fmt.Errorf("manifest 第 %d 行 entitlement_id 非法: %q", line, rec[1])
		}
		expiresAt, err := time.ParseInLocation(manifestTimeLayout, strings.TrimSpace(rec[2]), loc)
		if err != nil {
			return nil, fmt.Errorf("manifest 第 %d 行 old_expires_at 非法: %q", line, rec[2])
		}
		if seen[entID] {
			return nil, fmt.Errorf("manifest 授权 %d 重复出现", entID)
		}
		seen[entID] = true
		rows = append(rows, TrialRow{
			KindergartenID: kgID,
			EntitlementID:  entID,
			ExpiresAt:      expiresAt,
		})
	}
	return &Manifest{Path: path, SHA256: actual, Rows: rows}, nil
}

func WriteSummary(path string, plan *Plan) error {
	if path == "" {
		return errors.New("汇总文件路径为空")
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write(summaryHeaderFields); err != nil {
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
