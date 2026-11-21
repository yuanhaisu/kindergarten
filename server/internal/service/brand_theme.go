package service

import (
	"math"
	"strconv"
	"strings"
)

// 园所公开品牌的主题色校验（#170，管理端配置由 #178 复用）。
//
// 校验分两层：预设色板直接放行（设计 token 里选定的品牌色）；任意 #RRGGBB
// 必须通过对比度校验（相对白色 >= 3:1，WCAG AA 大字号/非文本组件）。非法值
// 不回显，统一回落默认主题——公开品牌接口是未登录启动页的数据源，脏值不能
// 变成前端不可读的文字/按钮。
const DefaultBrandColor = "#F97316"

// minBrandContrast 自定义 hex 与白色背景的最低对比度。
// 3:1 而非 4.5:1：品牌色主要作大字号/色块背景使用，预设色板已有的亮橙
// （#F97316 ≈ 2.8:1）也说明 4.5 会把整条暖色系挡在门外。
const minBrandContrast = 3.0

// presetBrandColors 设计 token 中可安全用作园所主题的品牌色。
// 预设绕过对比度校验：它们是设计侧选定并全局使用的颜色。
var presetBrandColors = []string{
	"#F97316", // 家长端品牌橙（默认）
	"#FF8A3D", // 老师端品牌橙
	"#E8A23A", // identity 琥珀
	"#C36540", // DNA 陶土
	"#5C7359", // DNA 鼠尾草绿
	"#173220", // DNA 松绿
	"#3B82F6", // 管理端品牌蓝
}

var presetBrandColorSet = func() map[string]struct{} {
	set := make(map[string]struct{}, len(presetBrandColors))
	for _, color := range presetBrandColors {
		set[color] = struct{}{}
	}
	return set
}()

// NormalizeBrandColor 把存储值归一化成一个可下发的主题色（大写 #RRGGBB）。
// 空值、非法格式或对比度不足时回落 DefaultBrandColor。
func NormalizeBrandColor(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return DefaultBrandColor
	}
	normalized := strings.ToUpper(trimmed)
	if _, ok := presetBrandColorSet[normalized]; ok {
		return normalized
	}
	rgb, ok := parseHexColor(normalized)
	if !ok {
		return DefaultBrandColor
	}
	if contrastRatio(rgb, rgbWhite) < minBrandContrast {
		return DefaultBrandColor
	}
	return normalized
}

type rgb struct {
	r, g, b uint8
}

var rgbWhite = rgb{255, 255, 255}

// parseHexColor 只接受规范的 #RRGGBB（调用方已转大写）。
func parseHexColor(value string) (rgb, bool) {
	if len(value) != 7 || value[0] != '#' {
		return rgb{}, false
	}
	channel := func(offset int) (uint8, bool) {
		n, err := strconv.ParseUint(value[offset:offset+2], 16, 8)
		if err != nil {
			return 0, false
		}
		return uint8(n), true
	}
	r, ok := channel(1)
	if !ok {
		return rgb{}, false
	}
	g, ok := channel(3)
	if !ok {
		return rgb{}, false
	}
	b, ok := channel(5)
	if !ok {
		return rgb{}, false
	}
	return rgb{r: r, g: g, b: b}, true
}

// contrastRatio 按 WCAG 2.x 相对亮度公式计算两种颜色的对比度。
func contrastRatio(a, b rgb) float64 {
	la := relativeLuminance(a)
	lb := relativeLuminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

func relativeLuminance(c rgb) float64 {
	channel := func(v uint8) float64 {
		s := float64(v) / 255
		if s <= 0.04045 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*channel(c.r) + 0.7152*channel(c.g) + 0.0722*channel(c.b)
}
