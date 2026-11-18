package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"kindergarten-server/pkg/log"
	"kindergarten-server/pkg/response"
)

// RateLimit 按来源 IP 做固定窗口限流。
//
// 严格模式：Redis 不可用（含未装配）时失败关闭，返回 503。限流用于保护
// 登录入口，失败开放等于在 Redis 故障期间移除暴力破解防护（design D5）。
func RateLimit(rdb *redis.Client, prefix string, maxRequests int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if rdb == nil {
			log.L(c.Request.Context()).Error("rate limit store not configured", "prefix", prefix)
			response.ErrUnavailable(c, "限流服务暂不可用，请稍后再试")
			c.Abort()
			return
		}

		ip := c.ClientIP()
		key := fmt.Sprintf("ratelimit:%s:%s", prefix, ip)

		ctx := context.Background()
		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			log.L(c.Request.Context()).Error("rate limit increment failed", "prefix", prefix, "err", err)
			response.ErrUnavailable(c, "限流服务暂不可用，请稍后再试")
			c.Abort()
			return
		}
		if count == 1 {
			if err := rdb.Expire(ctx, key, window).Err(); err != nil {
				// 计数没有窗口会永久累积：宁可本次拒绝，也不留下永不释放的配额。
				log.L(c.Request.Context()).Error("rate limit expire failed", "prefix", prefix, "err", err)
				_ = rdb.Del(ctx, key).Err()
				response.ErrUnavailable(c, "限流服务暂不可用，请稍后再试")
				c.Abort()
				return
			}
		}

		if count > int64(maxRequests) {
			response.ErrRateLimited(c)
			c.Abort()
			return
		}
		c.Next()
	}
}
