package service_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"kindergarten-server/internal/service"
)

// 公开品牌主题色（#170）：预设值直接放行，任意 hex 必须过对比度校验，
// 其余一律回落到默认主题。期望值取自设计 token 与 WCAG AA（非文本 3:1）计算，
// 不依赖实现内部常量，避免同义反复。
func TestNormalizeBrandColor(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"空值回落默认", "", service.DefaultBrandColor},
		{"空白回落默认", "   ", service.DefaultBrandColor},
		{"预设父端橙", "#F97316", "#F97316"},
		{"预设小写归一化", "#ff8a3d", "#FF8A3D"},
		{"预设琥珀", "#E8A23A", "#E8A23A"},
		{"自定义深橙通过对比度", "#C2410C", "#C2410C"},
		{"自定义深蓝通过对比度", "#1D4ED8", "#1D4ED8"},
		{"前后空白被裁掉", "  #c2410c  ", "#C2410C"},
		{"浅黄对比度不足", "#FFEB3B", service.DefaultBrandColor},
		{"颜色名非法", "red", service.DefaultBrandColor},
		{"缺井号非法", "C2410C", service.DefaultBrandColor},
		{"三位简写不接受", "#C24", service.DefaultBrandColor},
		{"非法字符", "#GGGGGG", service.DefaultBrandColor},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, service.NormalizeBrandColor(tc.in))
		})
	}
}
