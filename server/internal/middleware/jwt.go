package middleware

import (
	"kindergarten-server/internal/auth"
	"kindergarten-server/pkg/log"
	"kindergarten-server/pkg/response"

	"github.com/gin-gonic/gin"
)

const CtxClaimsKey = "jwt_claims"

// AuthJWT 校验 Bearer access token，并在 jti 非空时检查吊销黑名单。
// Redis 故障按失败关闭处理：返回 503，而不是放行一枚可能已被登出的 token。
func AuthJWT(secret []byte, revoker auth.TokenRevoker) gin.HandlerFunc {
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

		// 加 jti 之前签发的旧 token 没有 jti，无从判断是否被吊销，只能用到过期。
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

		ctx := log.With(c.Request.Context(),
			"user_id", claims.UserID,
		)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func GetClaims(c *gin.Context) *auth.AccessClaims {
	v, exists := c.Get(CtxClaimsKey)
	if !exists {
		return nil
	}
	claims, ok := v.(*auth.AccessClaims)
	if !ok {
		return nil
	}
	return claims
}
