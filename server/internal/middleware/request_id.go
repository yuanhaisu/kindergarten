package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"kindergarten-server/pkg/requestid"
)

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader("X-Request-ID")
		if rid == "" {
			rid = uuid.NewString()
		}
		c.Set("request_id", rid)
		// 同时放进请求 context：审计等下游从 context 取链路标识，
		// 不依赖每个 service 调用点手工透传。
		c.Request = c.Request.WithContext(requestid.With(c.Request.Context(), rid))
		c.Writer.Header().Set("X-Request-ID", rid)
		c.Next()
	}
}
