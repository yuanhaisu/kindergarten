package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"kindergarten-server/pkg/response"
)

// UnboundCode 未绑定账号访问业务接口的业务码。
//
// 与普通 40300 区分开：客户端据此进入绑定页，而不是提示「无权限」。
const UnboundCode = 40302

// CtxUnboundWhitelistedKey 由 RequireBoundAccount 写入，CasbinRBAC 读取。
//
// 待绑定 token 的角色为空，任何 Casbin 策略都匹配不上；为了白名单在策略里
// 造一个默认角色会重新打开「未绑定即有权限」的口子。这里改为显式标记：
// 只有白名单内的路由会被标记并跳过 Casbin，其余请求在 RequireBoundAccount
// 就已 403。
const CtxUnboundWhitelistedKey = "unbound_whitelisted"

// unboundAllowedRoutes 待绑定 token 唯一可达的路由（method + gin 路由模式）。
//
// 绑定接口（#167/#168）尚未注册，先登记路径：路由上线后无需再改这里，
// 也不会因为漏加白名单让绑定流程自己 403。公开品牌路由不在 authed 组内，
// /auth/refresh 也不在（它们不经过本中间件）。
var unboundAllowedRoutes = map[string]struct{}{
	"POST /api/v1/auth/logout":                   {},
	"GET /api/v1/common/user/profile":            {},
	"PUT /api/v1/common/user/profile":            {},
	"PUT /api/v1/common/user/avatar":             {},
	"GET /api/v1/common/consent":                 {},
	"POST /api/v1/common/consent":                {},
	"GET /api/v1/common/user/kindergartens":      {},
	"POST /api/v1/common/user/bind-invite":       {},
	"POST /api/v1/common/user/bind-parent-phone": {},
}

// RequireBoundAccount 把「未绑定 token」限制在白名单内（#175）。
//
// 判定依据是 token 自身：带 app_id（来自小程序客户端）且角色为空 = 待绑定。
// 管理端/平台凭证没有 app_id，行为完全不变；已绑定用户至少有一个当前客户端
// 角色，直接放行给 Casbin 做细粒度校验。
func RequireBoundAccount() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := GetClaims(c)
		if claims == nil || claims.AppID == "" || len(claims.Roles) > 0 {
			c.Next()
			return
		}

		if _, ok := unboundAllowedRoutes[c.Request.Method+" "+c.FullPath()]; !ok {
			response.Err(c, http.StatusForbidden, UnboundCode, "账号未绑定，请先完成绑定")
			c.Abort()
			return
		}

		c.Set(CtxUnboundWhitelistedKey, true)
		c.Next()
	}
}
