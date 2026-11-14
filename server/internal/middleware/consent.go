package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/log"
	"kindergarten-server/pkg/response"
)

// ConsentRequiredCode 客户端据此识别「需要同意」，与普通 403 权限不足区分。
const ConsentRequiredCode = 40301

// RequireConsent 在 enforce=true 时拦截未同意当前隐私政策的业务请求。
// 同意端点、登出、健康检查不在本中间件覆盖的 group 内，天然放行。
func RequireConsent(checker service.ConsentChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		if checker == nil || !checker.ConsentEnforced() {
			c.Next()
			return
		}
		uid := GetUserID(c)
		if uid == 0 {
			response.ErrUnauthorized(c, "缺少认证凭证")
			c.Abort()
			return
		}
		ok, err := checker.HasConsented(c.Request.Context(), uid)
		if err != nil {
			log.L(c.Request.Context()).Error("consent check failed", "err", err)
			response.ErrUnavailable(c, "认证服务暂不可用，请稍后再试")
			c.Abort()
			return
		}
		if !ok {
			response.Err(c, http.StatusForbidden, ConsentRequiredCode, "请先阅读并同意隐私政策")
			c.Abort()
			return
		}
		c.Next()
	}
}
