package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/auth"
	"kindergarten-server/pkg/log"
	"kindergarten-server/pkg/response"
)

// PasswordChangeRequiredCode 首登强制改密的业务码。
//
// 与 40300「无权限」、401「会话失效」区分：客户端必须跳转改密页，
// 不能当作登出清掉会话。见 backend/api/api-common.md §1。
const PasswordChangeRequiredCode = 40303

// PasswordChangeChecker 每请求读取账号是否仍在使用初始口令。
// 由 *service.AuthService 实现；读取失败必须返回 error（门禁失败关闭）。
type PasswordChangeChecker interface {
	MustChangePassword(ctx context.Context, userID uint64) (bool, error)
}

// passwordChangeAllowedRoutes 未改密管理端唯一可达的路由（method + gin 路由模式）。
//
// 精确白名单：同意（GET/POST）、个人资料读取、自助改密、登出。刻意不放
// `PUT /common/user/profile` 与 upload/messages——未改密账号还不需要这些写入口，
// 多放一条就多一条绕过改密流程的路径。
var passwordChangeAllowedRoutes = map[string]struct{}{
	"GET /api/v1/common/consent":       {},
	"POST /api/v1/common/consent":      {},
	"GET /api/v1/common/user/profile":  {},
	"PUT /api/v1/common/user/password": {},
	"POST /api/v1/auth/logout":         {},
}

// RequirePasswordChange 是首登强制改密的服务端门禁（#198）。
//
// 只对管理端（admin_web）凭证生效：小程序凭证没有可完成的改密流程，
// 平台凭证走独立信任域、不经过本中间件。判定真相源是 users.must_change_password，
// 不信任 token claim——改密后同一 access 的下一个请求必须立即放行。
//
// 每请求一次主键读取是刻意的：缓存会把「刚改完密」的用户继续挡住，
// 也会让 flag 的变更延迟一个 TTL。读取失败按 fail closed 返回 503。
func RequirePasswordChange(checker PasswordChangeChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := GetClaims(c)
		// 没有 claims 说明前面的 AuthJWT 未执行或未通过；交给后续中间件处理，
		// 本中间件不承担认证职责。
		if claims == nil || claims.ClientScopeKind != auth.ClientScopeAdminWeb {
			c.Next()
			return
		}

		if _, ok := passwordChangeAllowedRoutes[c.Request.Method+" "+c.FullPath()]; ok {
			c.Next()
			return
		}

		uid := GetUserID(c)
		if uid == 0 {
			response.ErrUnauthorized(c, "认证凭证无效")
			c.Abort()
			return
		}
		if checker == nil {
			log.L(c.Request.Context()).Error("password change checker not configured")
			response.ErrUnavailable(c, "认证服务暂不可用，请稍后再试")
			c.Abort()
			return
		}

		mustChange, err := checker.MustChangePassword(c.Request.Context(), uid)
		if err != nil {
			log.L(c.Request.Context()).Error("password change check failed", "err", err)
			response.ErrUnavailable(c, "认证服务暂不可用，请稍后再试")
			c.Abort()
			return
		}
		if mustChange {
			response.Err(c, http.StatusForbidden, PasswordChangeRequiredCode, "请先修改初始密码")
			c.Abort()
			return
		}

		c.Next()
	}
}
