package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS 只向显式允许的来源下发 CORS 头。
//
// 生产管理端经 Nginx 与 API 同源，不需要 CORS；小程序请求不带 Origin。
// 因此默认（origins 为空）不下发任何 Access-Control-Allow-Origin，跨域请求
// 由浏览器按同源策略拦截。只有显式配置的来源才能跨域读取响应。
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowAll := false
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, raw := range allowedOrigins {
		origin := strings.TrimSpace(raw)
		if origin == "" {
			continue
		}
		if origin == "*" {
			allowAll = true
			continue
		}
		allowed[origin] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if allowAll {
			c.Header("Access-Control-Allow-Origin", "*")
		} else {
			// 即使本次不放行也要 Vary，避免中间缓存把 A 来源的响应复用到 B 来源。
			c.Header("Vary", "Origin")
			if origin != "" {
				if _, ok := allowed[origin]; ok {
					c.Header("Access-Control-Allow-Origin", origin)
				}
			}
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-Request-ID, terminal_from")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
