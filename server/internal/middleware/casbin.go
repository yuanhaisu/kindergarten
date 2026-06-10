package middleware

import (
	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"

	"kindergarten-server/pkg/log"
	"kindergarten-server/pkg/response"
)

func CasbinRBAC(e *casbin.Enforcer) gin.HandlerFunc {
	return func(c *gin.Context) {
		if e == nil {
			c.Next()
			return
		}

		claims := GetClaims(c)
		if claims == nil {
			c.Next()
			return
		}

		obj := c.FullPath()
		act := c.Request.Method

		var allowed bool
		for _, role := range claims.Roles {
			ok, err := e.Enforce(role, obj, act)
			if err != nil {
				log.L(c.Request.Context()).Error("casbin enforce error", "err", err)
				response.ErrInternal(c, "权限校验失败")
				c.Abort()
				return
			}
			if ok {
				allowed = true
				break
			}
		}

		if !allowed {
			response.ErrForbidden(c, "无权限访问该资源")
			c.Abort()
			return
		}
		c.Next()
	}
}
