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
//
// epochs 是园所账号级凭证失效时间（#182）：自助改密、管理员重置密码与
// 删除管理员后，此前签发的访问凭证立即失效。读不到 epoch 同样失败关闭，
// 不能把「存储不可用」当成「从未失效」。
func AuthJWT(secret []byte, revoker auth.TokenRevoker, epochs auth.TenantEpochStore) gin.HandlerFunc {
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
		// 平台凭证不属于园所信任域，不能用于园所接口（ADR-0014）。
		if claims.Scope == auth.ScopePlatform {
			response.ErrForbidden(c, "平台凭证不能访问园所接口")
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

		// 账号级失效：改密/重置/删除后，epoch 之前签发的凭证一律拒绝。
		// 没有 jti 的旧 token 也必须过这一关，否则停用后旧凭证仍可用。
		if epochs == nil {
			log.L(c.Request.Context()).Error("tenant epoch store not configured")
			response.ErrUnavailable(c, "认证服务暂不可用，请稍后再试")
			c.Abort()
			return
		}
		epoch, err := epochs.TenantEpoch(c.Request.Context(), claims.UserID)
		if err != nil {
			log.L(c.Request.Context()).Error("tenant epoch lookup failed", "err", err)
			response.ErrUnavailable(c, "认证服务暂不可用，请稍后再试")
			c.Abort()
			return
		}
		if claims.IssuedAt == nil || claims.IssuedAt.Before(epoch) {
			response.ErrUnauthorized(c, "登录状态已失效，请重新登录")
			c.Abort()
			return
		}

		c.Set(CtxClaimsKey, claims)

		ctx := log.With(c.Request.Context(),
			"user_id", claims.UserID,
			"app_id", claims.AppID,
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
