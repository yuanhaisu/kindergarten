package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"kindergarten-server/pkg/log"
	"kindergarten-server/pkg/response"
)

// 各入口的默认配额（每分钟）。数值来自 #160 的安全收口：
// 密码登录 10 次/IP+账号；微信登录不设低阈值（code 一次性），故障开放；
// 手机号绑定 20 次/app+IP+手机号。
const (
	PasswordLoginMaxPerMinute = 10
	WeChatLoginMaxPerMinute   = 600
	BindPhoneMaxPerMinute     = 20
)

// RateLimitPart 从请求中提取附加限流维度（手机号、app_id 等）。
// 返回空串表示只按 IP 限流。
type RateLimitPart func(*gin.Context) string

// RateLimit 按来源 IP 做固定窗口限流。
//
// 严格模式：Redis 不可用（含未装配）时失败关闭，返回 503。限流用于保护
// 登录入口，失败开放等于在 Redis 故障期间移除暴力破解防护（design D5）。
func RateLimit(rdb *redis.Client, prefix string, maxRequests int, window time.Duration) gin.HandlerFunc {
	return rateLimitBy(rdb, prefix, maxRequests, window, false, nil)
}

// RateLimitFailOpen 与 RateLimit 的差别只在存储不可用时**放行**。
//
// 用于公开只读数据（园所品牌）：它没有暴力破解面，却是未登录启动页的数据源。
// 失败关闭会把 Redis 抖动放大成「家长端启动 503」，失败开放最多是短时失去
// 滥用保护。配额判定本身仍然严格，Redis 正常时超限照常 429。
func RateLimitFailOpen(rdb *redis.Client, prefix string, maxRequests int, window time.Duration) gin.HandlerFunc {
	return rateLimitBy(rdb, prefix, maxRequests, window, true, nil)
}

// RateLimitBy 按 IP + 附加维度限流，供后续绑定端点等复用。
//
// failOpen 只影响**存储不可用**时的行为：成功判定配额仍然严格。
// part 返回的原文不会进入 Redis key（只存 sha256 前缀），避免手机号落进键空间。
func RateLimitBy(
	rdb *redis.Client,
	prefix string,
	maxRequests int,
	window time.Duration,
	failOpen bool,
	part RateLimitPart,
) gin.HandlerFunc {
	return rateLimitBy(rdb, prefix, maxRequests, window, failOpen, part)
}

// PasswordLoginRateLimit 密码登录：IP + 手机号，失败关闭。
func PasswordLoginRateLimit(rdb *redis.Client, maxRequests int, window time.Duration) gin.HandlerFunc {
	return rateLimitBy(rdb, "password_login", maxRequests, window, false, func(c *gin.Context) string {
		return jsonBodyString(c, "phone", "username")
	})
}

// WeChatLoginRateLimit 微信登录：app_id + IP，高阈值、失败开放。
//
// code 一次性，重放价值低；真正的防线在 code 换取环节。这里只拦明显的
// 脚本洪峰，阈值取「数百/分钟」，共用出口 IP 的整所幼儿园不会被误伤。
func WeChatLoginRateLimit(rdb *redis.Client, maxRequests int, window time.Duration) gin.HandlerFunc {
	return rateLimitBy(rdb, "wechat_login", maxRequests, window, true, func(c *gin.Context) string {
		appID := claimsAppID(c)
		if appID == "" {
			appID = jsonBodyString(c, "app_id")
		}
		return appID
	})
}

// BindPhoneRateLimit 手机号绑定：app_id + IP + 手机号，失败开放并告警。
//
// 绑定端点由 #167/#168 落地，届时直接挂本中间件即可；如果请求体字段不同，
// 用 RateLimitBy 传自定义提取函数。失败开放是因为绑定失败会让家长直接卡在
// 登录流程里，而每次绑定都必须持有微信侧凭据（code），滥用面远小于密码登录。
func BindPhoneRateLimit(rdb *redis.Client, maxRequests int, window time.Duration) gin.HandlerFunc {
	return rateLimitBy(rdb, "bind_phone", maxRequests, window, true, func(c *gin.Context) string {
		appID := claimsAppID(c)
		if appID == "" {
			appID = jsonBodyString(c, "app_id")
		}
		return joinRateLimitParts("app_id", appID, "phone", jsonBodyString(c, "phone"))
	})
}

func rateLimitBy(
	rdb *redis.Client,
	prefix string,
	maxRequests int,
	window time.Duration,
	failOpen bool,
	part RateLimitPart,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		if rdb == nil {
			if failOpen {
				log.L(c.Request.Context()).Warn("rate limit store not configured; allowing request", "prefix", prefix)
				c.Next()
				return
			}
			log.L(c.Request.Context()).Error("rate limit store not configured", "prefix", prefix)
			response.ErrUnavailable(c, "限流服务暂不可用，请稍后再试")
			c.Abort()
			return
		}

		key := rateLimitKey(prefix, c.ClientIP(), rateLimitPart(c, part))

		ctx := context.Background()
		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			if failOpen {
				log.L(c.Request.Context()).Warn("rate limit increment failed; allowing request",
					"prefix", prefix, "err", err)
				c.Next()
				return
			}
			log.L(c.Request.Context()).Error("rate limit increment failed", "prefix", prefix, "err", err)
			response.ErrUnavailable(c, "限流服务暂不可用，请稍后再试")
			c.Abort()
			return
		}
		if count == 1 {
			if err := rdb.Expire(ctx, key, window).Err(); err != nil {
				// 计数没有窗口会永久累积：严格模式宁可本次拒绝，也不留下永不释放的配额；
				// 开放模式放行并清掉脏键，避免下一次正常请求被误伤。
				if failOpen {
					log.L(c.Request.Context()).Warn("rate limit expire failed; allowing request",
						"prefix", prefix, "err", err)
					_ = rdb.Del(ctx, key).Err()
					c.Next()
					return
				}
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

func rateLimitPart(c *gin.Context, part RateLimitPart) string {
	if part == nil {
		return ""
	}
	return part(c)
}

// rateLimitKey 保持无附加维度时的键格式不变（平台登录、公开品牌已在用）。
func rateLimitKey(prefix, ip, part string) string {
	if part == "" {
		return fmt.Sprintf("ratelimit:%s:%s", prefix, ip)
	}
	sum := sha256.Sum256([]byte(part))
	return fmt.Sprintf("ratelimit:%s:%s:%s", prefix, ip, hex.EncodeToString(sum[:8]))
}

func joinRateLimitParts(pairs ...string) string {
	var b strings.Builder
	for i := 0; i+1 < len(pairs); i += 2 {
		if pairs[i+1] == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte('|')
		}
		b.WriteString(pairs[i])
		b.WriteByte('=')
		b.WriteString(pairs[i+1])
	}
	return b.String()
}

func claimsAppID(c *gin.Context) string {
	if claims := GetClaims(c); claims != nil {
		return strings.TrimSpace(claims.AppID)
	}
	return ""
}

// BindPhoneLimiter 在拿到手机号之后按 app_id + IP + 手机号做固定窗口限流（#167）。
//
// BindPhoneRateLimit 中间件只能看到一次性 wx_phone_code，拿不到手机号；
// 完整维度必须等 service 用微信凭据换出手机号后再判定。Redis 故障失败开放
// 并告警：绑定失败会把家长卡在登录流程里，而每次绑定都要持有微信侧凭据，
// 滥用面远小于密码登录。
type BindPhoneLimiter struct {
	rdb    *redis.Client
	max    int
	window time.Duration
}

// NewBindPhoneLimiter 构造手机号维度限流器；rdb 为 nil 时永远放行并告警。
func NewBindPhoneLimiter(rdb *redis.Client, maxRequests int, window time.Duration) *BindPhoneLimiter {
	return &BindPhoneLimiter{rdb: rdb, max: maxRequests, window: window}
}

// Allow 报告本次绑定是否仍在配额内。手机号只以 sha256 前缀进入键空间。
func (l *BindPhoneLimiter) Allow(ctx context.Context, ip, appID, phone string) bool {
	if l == nil || l.rdb == nil {
		log.L(ctx).Warn("bind phone limiter not configured; allowing request")
		return true
	}
	key := rateLimitKey("bind_phone", ip, joinRateLimitParts("app_id", appID, "phone", phone))
	count, err := l.rdb.Incr(ctx, key).Result()
	if err != nil {
		log.L(ctx).Warn("bind phone rate limit increment failed; allowing request", "err", err)
		return true
	}
	if count == 1 {
		if err := l.rdb.Expire(ctx, key, l.window).Err(); err != nil {
			log.L(ctx).Warn("bind phone rate limit expire failed; allowing request", "err", err)
			_ = l.rdb.Del(ctx, key).Err()
			return true
		}
	}
	return count <= int64(l.max)
}

const rateLimitBodyKey = "middleware.ratelimit.json_body"
// jsonBodyString 读取请求体里的字符串字段。读取后还原请求体，handler 仍能绑定。
// 结果按请求缓存，多个限流中间件不会重复读取。
func jsonBodyString(c *gin.Context, keys ...string) string {
	body := rateLimitJSONBody(c)
	for _, k := range keys {
		if v, ok := body[k].(string); ok {
			if v = strings.TrimSpace(v); v != "" {
				return v
			}
		}
	}
	return ""
}

func rateLimitJSONBody(c *gin.Context) map[string]any {
	if v, ok := c.Get(rateLimitBodyKey); ok {
		body, _ := v.(map[string]any)
		return body
	}

	var body map[string]any
	if c.Request != nil && c.Request.Body != nil {
		// 1 MiB 上限：登录/绑定请求体都很小，超限的请求交给 handler 报校验错误。
		raw, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
		if err == nil && len(raw) > 0 {
			_ = json.Unmarshal(raw, &body)
		}
		if err != nil {
			log.L(c.Request.Context()).Warn("rate limit read body failed", "err", err)
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(raw))
	}
	c.Set(rateLimitBodyKey, body)
	return body
}
