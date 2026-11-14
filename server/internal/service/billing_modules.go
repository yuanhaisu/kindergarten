package service

import (
	"fmt"
	"sort"
)

// ModuleDef 可售功能模块定义。模块清单是服务端常量，不建表（design D3）。
// 路由默认归 base；只有 premium 模块参与授权裁剪。
type ModuleDef struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Sellable bool   `json:"sellable"`
}

// moduleCatalog 与 migrations/00003_backfill_commercial_entitlements.sql 中的
// premium 模块清单必须保持一致。
var moduleCatalog = []ModuleDef{
	{Code: "base", Name: "园务基础", Category: "base", Sellable: false},
	{Code: "growth", Name: "成长记录", Category: "premium", Sellable: true},
	{Code: "album", Name: "班级相册", Category: "premium", Sellable: true},
	{Code: "growth_report", Name: "成长报告", Category: "premium", Sellable: true},
	{Code: "diet", Name: "膳食营养", Category: "premium", Sellable: true},
}

func AllModules() []ModuleDef {
	out := make([]ModuleDef, len(moduleCatalog))
	copy(out, moduleCatalog)
	return out
}

func PremiumModules() []ModuleDef {
	out := make([]ModuleDef, 0, len(moduleCatalog))
	for _, m := range moduleCatalog {
		if m.Category == "premium" {
			out = append(out, m)
		}
	}
	return out
}

func IsRegisteredModule(code string) bool {
	for _, m := range moduleCatalog {
		if m.Code == code {
			return true
		}
	}
	return false
}

func ModuleCategory(code string) string {
	for _, m := range moduleCatalog {
		if m.Code == code {
			return m.Category
		}
	}
	return ""
}

func ModuleName(code string) string {
	for _, m := range moduleCatalog {
		if m.Code == code {
			return m.Name
		}
	}
	return code
}

// normalizeModuleCodes 去重并保持稳定顺序，供套餐保存与快照使用。
func normalizeModuleCodes(codes []string) []string {
	seen := make(map[string]struct{}, len(codes))
	out := make([]string, 0, len(codes))
	for _, c := range codes {
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

func errUnknownModule(code string) error {
	return fmt.Errorf("UNKNOWN_MODULE: %s", code)
}
