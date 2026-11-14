package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/log"
	"kindergarten-server/pkg/response"
)

// premiumRouteRules 路由 → 模块归属。按 contains 匹配，顺序敏感
// （growth_report 必须排在 growth 之前）。未命中的路由一律归 base。
//
// 用命名空间匹配而不是逐条登记：现有 161 条路由里新增一条成长/膳食接口时，
// 只要路径落在同一命名空间就会被自动纳入授权，不会静默漏掉。
var premiumRouteRules = []struct {
	Contains string
	Module   string
}{
	{"/growth/report", "growth_report"},
	{"/growth", "growth"},
	{"/album", "album"},
	{"/diet", "diet"},
}

// PremiumModuleCodes 全部 premium 模块（供启动校验）。
func PremiumModuleCodes() []string { return []string{"growth", "growth_report", "album", "diet"} }

// ModuleForRoute 返回路由所属模块；无法归类时为 base。
func ModuleForRoute(method, path string) string {
	p := strings.ToLower(path)
	for _, r := range premiumRouteRules {
		if strings.Contains(p, r.Contains) {
			return r.Module
		}
	}
	return "base"
}

// ModuleRegistryGaps 启动校验：每个 premium 模块必须至少匹配一条已注册路由。
// 匹配不到说明规则写错或对应功能已被删除，带着这种缺口启动没有意义。
func ModuleRegistryGaps(routes []gin.RouteInfo) []string {
	hit := make(map[string]bool, len(premiumRouteRules))
	for _, r := range routes {
		for _, rule := range premiumRouteRules {
			if strings.Contains(strings.ToLower(r.Path), rule.Contains) {
				hit[rule.Module] = true
			}
		}
	}
	var gaps []string
	for _, code := range PremiumModuleCodes() {
		if !hit[code] {
			gaps = append(gaps, code)
		}
	}
	return gaps
}

func isWriteMethod(method string) bool {
	switch method {
	case "GET", "HEAD", "OPTIONS":
		return false
	default:
		return true
	}
}

// ModuleGate 按授权裁剪 premium 模块。挂在认证之后，对全部园所路由生效。
// 开关关闭时只记录「本应拒绝」的请求并放行（design D11）。
func ModuleGate(authz service.ModuleAuthorizer) gin.HandlerFunc {
	return func(c *gin.Context) {
		if authz == nil {
			c.Next()
			return
		}
		module := ModuleForRoute(c.Request.Method, c.FullPath())
		if module == "base" {
			c.Next()
			return
		}
		kgID := GetKindergartenID(c)
		decision, err := authz.Decide(c.Request.Context(), kgID, module, isWriteMethod(c.Request.Method))
		if err != nil {
			log.L(c.Request.Context()).Error("module gate failed", "module", module, "err", err)
			response.ErrInternal(c, "功能授权校验失败")
			c.Abort()
			return
		}
		if decision.Allowed {
			if strings.HasPrefix(decision.Reason, "would_deny:") {
				log.L(c.Request.Context()).Warn("module gate would deny",
					"kindergarten_id", kgID, "module", module, "path", c.FullPath(), "reason", decision.Reason)
			}
			c.Next()
			return
		}
		response.ErrForbidden(c, "该功能未开通或已到期")
		c.Abort()
	}
}
