package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"gorm.io/gorm"
)

// allowWriteEnvVar 是写操作的显式确认开关。dry-run 不需要；
// apply/revert 必须由操作者显式设置，避免误跑命令改动生产数据。
const allowWriteEnvVar = "EXTENDTRIALS_ALLOW"

// Result 是一次命令运行的结果。
type Result struct {
	Plan            *Plan
	Mode            string // "dry-run" / "apply" / "revert"
	ManifestPath    string
	ManifestSHA256  string
	ManifestWritten bool
	SHAFilePath     string
	SummaryPath     string
	SummaryWritten  bool

	AffectedRows          int
	AffectedKindergartens int
	UnchangedRows         int
	LogID                 uint64
}

// run 是命令的可测试入口：不解析 flag、不连库、不打印。
//
// 三种模式：
//   - 默认 dry-run：BuildPlan 只做 SELECT，写 manifest 与校验值侧车文件；
//   - -apply：按 manifest 延长 trial 授权（需 EXTENDTRIALS_ALLOW=1 与 -operator）；
//   - -revert：按 manifest 原始值恢复（同样的写入门槛）。
//
// 写操作始终先校验 manifest 版本与 SHA256，任一不匹配即拒绝。
func run(ctx context.Context, db *gorm.DB, opts Options) (*Result, error) {
	if opts.Apply && opts.Revert {
		return nil, errors.New("-apply 与 -revert 不能同时使用：延长用 -apply，回滚用 -revert，dry-run 不需要开关")
	}
	writeMode := opts.Apply || opts.Revert
	if opts.ManifestPath == "" {
		if writeMode {
			return nil, errors.New("写操作需要 -manifest <path>：只能处理 dry-run 产出并经运营确认的清单")
		}
		return nil, errors.New("dry-run 需要 -manifest <path>：manifest 是本命令的交付物")
	}

	if writeMode {
		if err := requireWriteOptIn(); err != nil {
			return nil, err
		}
		if strings.TrimSpace(opts.Operator) == "" {
			return nil, errors.New("写操作需要 -operator <执行人>：会写入 platform_operation_logs")
		}
		if _, err := os.Stat(opts.ManifestPath); err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("manifest 不存在：%s（请先用 dry-run 生成并经运营确认）", opts.ManifestPath)
			}
			return nil, fmt.Errorf("检查 manifest %s 失败: %w", opts.ManifestPath, err)
		}
		expectedSHA, _, err := ResolveManifestSHA(opts.ManifestPath, opts.ManifestSHA256)
		if err != nil {
			return nil, err
		}
		manifest, err := ReadManifest(opts.ManifestPath, expectedSHA)
		if err != nil {
			return nil, err
		}
		if opts.Apply {
			stats, err := ApplyManifest(ctx, db, opts, manifest)
			if err != nil {
				return nil, err
			}
			return &Result{
				Mode:                  "apply",
				ManifestPath:          opts.ManifestPath,
				ManifestSHA256:        manifest.SHA256,
				AffectedRows:          stats.Rows,
				AffectedKindergartens: stats.Kindergartens,
				UnchangedRows:         stats.Unchanged,
				LogID:                 stats.LogID,
			}, nil
		}
		stats, err := RevertManifest(ctx, db, opts, manifest)
		if err != nil {
			return nil, err
		}
		return &Result{
			Mode:                  "revert",
			ManifestPath:          opts.ManifestPath,
			ManifestSHA256:        manifest.SHA256,
			AffectedRows:          stats.Rows,
			AffectedKindergartens: stats.Kindergartens,
			UnchangedRows:         stats.Unchanged,
			LogID:                 stats.LogID,
		}, nil
	}

	plan, err := BuildPlan(ctx, db, opts)
	if err != nil {
		return nil, err
	}
	res := &Result{Plan: plan, Mode: "dry-run"}

	sha, err := WriteManifest(opts.ManifestPath, plan)
	if errors.Is(err, ErrEmptyPlan) {
		return res, nil
	}
	if err != nil {
		return nil, fmt.Errorf("写 manifest 失败: %w", err)
	}
	res.ManifestPath = opts.ManifestPath
	res.ManifestSHA256 = sha
	res.ManifestWritten = true

	shaPath, err := WriteManifestSHA256(opts.ManifestPath, sha)
	if err != nil {
		return nil, fmt.Errorf("manifest 已写出到 %s，但校验值文件写入失败: %w", opts.ManifestPath, err)
	}
	res.SHAFilePath = shaPath

	summaryPath := opts.ManifestPath + ".summary.csv"
	if err := WriteSummary(summaryPath, plan); err != nil {
		return nil, fmt.Errorf("manifest 已写出到 %s，但汇总文件写入失败: %w", opts.ManifestPath, err)
	}
	res.SummaryPath = summaryPath
	res.SummaryWritten = true
	return res, nil
}

// requireWriteOptIn 检查写操作开关。值必须是精确的 "1"（允许首尾空白）。
func requireWriteOptIn() error {
	if strings.TrimSpace(os.Getenv(allowWriteEnvVar)) != "1" {
		return fmt.Errorf("写操作需显式确认：设置环境变量 %s=1（dry-run 不需要该开关）", allowWriteEnvVar)
	}
	return nil
}
