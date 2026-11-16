package main

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// ErrApplyNotImplemented 是 #65 的边界：只读预演先行，写库由 #66 接手。
var ErrApplyNotImplemented = errors.New("-apply not implemented in this ticket (#65); see #66")

// Result 是一次命令运行的结果。
type Result struct {
	Plan            *Plan
	ManifestPath    string
	ManifestSHA256  string
	ManifestWritten bool
	SummaryPath     string
	SummaryWritten  bool
}

// run 是命令的可测试入口：不解析 flag、不连库、不打印。
//
// 默认只读：BuildPlan 只做 SELECT；manifest/汇总写盘不属于数据库写入。
// 空结果不落盘任何文件，但仍是成功运行（明确提示由调用方负责）。
func run(ctx context.Context, db *gorm.DB, opts Options) (*Result, error) {
	if opts.Apply {
		return nil, ErrApplyNotImplemented
	}
	if opts.ManifestPath == "" {
		return nil, errors.New("dry-run 需要 -manifest <path>：manifest 是本命令的交付物")
	}

	plan, err := BuildPlan(ctx, db, opts)
	if err != nil {
		return nil, err
	}
	res := &Result{Plan: plan}

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

	summaryPath := opts.ManifestPath + ".summary.csv"
	if err := WriteSummary(summaryPath, plan); err != nil {
		return nil, fmt.Errorf("manifest 已写出到 %s，但汇总文件写入失败: %w", opts.ManifestPath, err)
	}
	res.SummaryPath = summaryPath
	res.SummaryWritten = true
	return res, nil
}
