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
		key := rateLimitKey(prefix, ip)

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

// RateLimitFailOpen 与 RateLimit 的差别只在存储不可用时**放行**。
//
// 用于公开只读数据（园所品牌）：它没有暴力破解面，却是未登录启动页的数据源。
// 失败关闭会把 Redis 抖动放大成「家长端启动 503」，失败开放最多是短时失去
// 滥用保护。配额判定本身仍然严格，Redis 正常时超限照常 429。
func RateLimitFailOpen(rdb *redis.Client, prefix string, maxRequests int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if rdb == nil {
			log.L(c.Request.Context()).Warn("rate limit store not configured; allowing public request", "prefix", prefix)
			c.Next()
			return
		}

		ip := c.ClientIP()
		key := rateLimitKey(prefix, ip)

		ctx := context.Background()
		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			log.L(c.Request.Context()).Warn("rate limit increment failed; allowing public request",
				"prefix", prefix, "err", err)
			c.Next()
			return
		}
		if count == 1 {
			if err := rdb.Expire(ctx, key, window).Err(); err != nil {
				// 没有窗口的计数会永久累积；放行的同时清掉脏键，避免下一次正常请求被误伤。
				log.L(c.Request.Context()).Warn("rate limit expire failed; allowing public request",
					"prefix", prefix, "err", err)
				_ = rdb.Del(ctx, key).Err()
				c.Next()
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

func rateLimitKey(prefix, ip string) string {
	return fmt.Sprintf("ratelimit:%s:%s", prefix, ip)
}
