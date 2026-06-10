package middleware

import (
	"github.com/gin-gonic/gin"

	"kindergarten-server/pkg/log"
)

func RequestContextLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := log.With(c.Request.Context(),
			"request_id", c.GetString("request_id"),
		)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
