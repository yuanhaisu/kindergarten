// Package clientdiag 提供小程序客户端注册表的只读诊断（#179）。
//
// 平台支持用它回答「某一端为什么登录不了」：该 app_id 是否注册、是否
// enabled、作用域是否 shared、当前发版版本是多少。命令不写库，也不提供
// 停用/启用开关——status 只能通过数据库/运维操作变更（#163）。
//
// 放在 cmd/internal/ 下：只有 cmd/ 的子命令能导入。
package clientdiag

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"kindergarten-server/internal/domain"
)

// ClientLister 是诊断所需的最小只读接口：只有 List。
//
// 刻意不声明 Create/UpdateFields：诊断命令在类型层面就改不了 status，
// 不依赖「记得别写库」的约定。
type ClientLister interface {
	List(ctx context.Context) ([]domain.Client, error)
}

// ClientDiagnostic 是一条客户端的对外诊断字段，与 #163 注册表一一对应。
// Enabled 由 status 派生，便于脚本直接过滤。
type ClientDiagnostic struct {
	AppID          string `json:"app_id"`
	Role           string `json:"role"`
	ScopeKind      string `json:"scope_kind"`
	ScopeID        uint64 `json:"scope_id"`
	Status         string `json:"status"`
	Enabled        bool   `json:"enabled"`
	ReleaseVersion string `json:"release_version"`
}

// Collect 读取注册表并转成诊断视图。查询失败必须冒泡，不能返回空表。
func Collect(ctx context.Context, lister ClientLister) ([]ClientDiagnostic, error) {
	rows, err := lister.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("读取 clients 注册表失败：%w", err)
	}
	out := make([]ClientDiagnostic, 0, len(rows))
	for _, row := range rows {
		out = append(out, ClientDiagnostic{
			AppID:          row.AppID,
			Role:           row.Role,
			ScopeKind:      row.ScopeKind,
			ScopeID:        row.ScopeID,
			Status:         row.Status,
			Enabled:        row.IsEnabled(),
			ReleaseVersion: row.ReleaseVersion,
		})
	}
	return out, nil
}

// WriteJSON 输出供脚本/工单消费的稳定结构。空注册表输出 []，不是 null。
func WriteJSON(w io.Writer, rows []ClientDiagnostic) error {
	payload := struct {
		Count   int                `json:"count"`
		Clients []ClientDiagnostic `json:"clients"`
	}{Count: len(rows), Clients: rows}
	if payload.Clients == nil {
		payload.Clients = []ClientDiagnostic{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

// WriteTable 输出人类可读的表格：一条客户端一行，列序与表头一致。
func WriteTable(w io.Writer, rows []ClientDiagnostic) error {
	if len(rows) == 0 {
		_, err := fmt.Fprintln(w, "小程序客户端注册表为空（共 0 条）：请确认启动同步是否已执行（配置里两端 AppID 是否齐全）。")
		return err
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintf(tw, "小程序客户端注册表（共 %d 条）：\n", len(rows)); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(tw, "APP_ID\tROLE\tSCOPE_KIND\tSCOPE_ID\tSTATUS\tENABLED\tRELEASE_VERSION"); err != nil {
		return err
	}
	for _, row := range rows {
		enabled := "yes"
		if !row.Enabled {
			enabled = "no"
		}
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%s\t%s\t%s\n",
			row.AppID, row.Role, row.ScopeKind, row.ScopeID, row.Status, enabled, row.ReleaseVersion); err != nil {
			return err
		}
	}
	return tw.Flush()
}
