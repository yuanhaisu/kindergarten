package middleware

import (
	"kindergarten-server/internal/auth"
	"kindergarten-server/pkg/log"
	"kindergarten-server/pkg/response"

	"github.com/gin-gonic/gin"
)

const CtxClaimsKey = "jwt_claims"

func AuthJWT(secret []byte) gin.HandlerFunc {
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
