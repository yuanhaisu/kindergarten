package middleware

import (
	"kindergarten-server/internal/auth"
	"kindergarten-server/pkg/log"
	"kindergarten-server/pkg/response"

	"github.com/gin-gonic/gin"
)

// PlatformAuth 校验平台方凭证：必须是 scope=platform 的 token。
// 与 AuthJWT 是两套独立的信任域（ADR-0014）。
func PlatformAuth(secret []byte, revoker auth.TokenRevoker) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if len(h) < len(prefix) || h[:len(prefix)] != prefix {
			response.ErrUnauthorized(c, "缺少认证凭证")
			c.Abort()
			return
		}

		claims, err := auth.ParseAccess(secret, h[len(prefix):])
		if err != nil {
			response.ErrUnauthorized(c, "认证凭证无效或已过期")
			c.Abort()
			return
		}
		if claims.Scope != auth.ScopePlatform {
			response.ErrForbidden(c, "园所凭证不能访问平台接口")
			c.Abort()
			return
		}

		if claims.ID != "" {
			if revoker == nil {
				log.L(c.Request.Context()).Error("token revoker not configured")
				response.ErrUnavailable(c, "认证服务暂不可用，请稍后再试")
				c.Abort()
				return
			}
			revoked, err := revoker.Revoked(c.Request.Context(), claims.ID)
			if err != nil {
				log.L(c.Request.Context()).Error("token blacklist check failed", "err", err)
				response.ErrUnavailable(c, "认证服务暂不可用，请稍后再试")
				c.Abort()
				return
			}
			if revoked {
				response.ErrUnauthorized(c, "登录状态已失效，请重新登录")
				c.Abort()
				return
			}
		}

		c.Set(CtxClaimsKey, claims)
		ctx := log.With(c.Request.Context(), "platform_admin_id", claims.UserID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// GetPlatformAdminID 平台账号 ID（字符串形式，与园所用户 ID 空间无关）。
func GetPlatformAdminID(c *gin.Context) string {
	claims := GetClaims(c)
	if claims == nil || claims.Scope != auth.ScopePlatform {
		return ""
	}
	return claims.UserID
}
