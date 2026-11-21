package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/testutil/fakeredis"
)

// newRateLimitedEngine 挂上与平台登录相同的严格限流（10 次/分钟/IP）。
func newRateLimitedEngine(rdb *redis.Client) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.POST("/api/v1/platform/auth/login", RateLimit(rdb, "platform_login", 10, time.Minute),
		func(c *gin.Context) { c.Status(http.StatusNoContent) })
	return engine
}

func callRateLimited(engine *gin.Engine) int {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/auth/login", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w.Code
}

// 平台登录限流 10 次/分钟/IP：第 11 次必须被拒绝（TC-NEW-PLT-0012）。
func TestPlatformAuthLoginRateLimitRejectsBurst(t *testing.T) {
	f := fakeredis.New(t)
	engine := newRateLimitedEngine(f.Client(t))

	for i := 1; i <= 10; i++ {
		require.Equal(t, http.StatusNoContent, callRateLimited(engine), "第 %d 次应在配额内", i)
	}
	assert.Equal(t, http.StatusTooManyRequests, callRateLimited(engine), "第 11 次必须 429")
}

// Redis 故障时限流必须失败关闭：失败开放等于暴力破解无限制（design D5）。
func TestPlatformAuthLoginRateLimitFailsClosedOnRedisError(t *testing.T) {
	f := fakeredis.New(t)
	f.SetFailWrites(true)
	engine := newRateLimitedEngine(f.Client(t))

	assert.Equal(t, http.StatusServiceUnavailable, callRateLimited(engine))
}

// 未装配 Redis 时同样失败关闭，不能把「读不到计数」当成「未限流」。
func TestPlatformAuthLoginRateLimitFailsClosedWithoutRedis(t *testing.T) {
	engine := newRateLimitedEngine(nil)

	assert.Equal(t, http.StatusServiceUnavailable, callRateLimited(engine))
}

// ---------- 公开只读端点：失败开放（#170） ----------

// newPublicBrandRateLimitedEngine 挂上公开品牌接口的失败开放限流（测试用 3 次/分钟）。
func newPublicBrandRateLimitedEngine(rdb *redis.Client) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/api/v1/public/kindergartens/:public_code",
		RateLimitFailOpen(rdb, "public_brand", 3, time.Minute),
		func(c *gin.Context) { c.Status(http.StatusNoContent) })
	return engine
}

func callPublicBrandRateLimited(engine *gin.Engine) int {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/kindergartens/abc", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w.Code
}

// 配额内正常放行，超出后 429——限流本身仍然生效。
func TestPublicBrandRateLimitRejectsBurst(t *testing.T) {
	f := fakeredis.New(t)
	engine := newPublicBrandRateLimitedEngine(f.Client(t))

	for i := 1; i <= 3; i++ {
		require.Equal(t, http.StatusNoContent, callPublicBrandRateLimited(engine), "第 %d 次应在配额内", i)
	}
	assert.Equal(t, http.StatusTooManyRequests, callPublicBrandRateLimited(engine), "第 4 次必须 429")
}

// 公开品牌是未登录启动页的数据源：Redis 故障必须放行（失败开放），
// 不能因为限流存储抖动把家长端启动打成 503。
func TestPublicBrandRateLimitFailsOpenOnRedisError(t *testing.T) {
	f := fakeredis.New(t)
	f.SetFailWrites(true)
	engine := newPublicBrandRateLimitedEngine(f.Client(t))

	assert.Equal(t, http.StatusNoContent, callPublicBrandRateLimited(engine))
}

// 未装配 Redis 时同样失败开放：公开数据没有暴力破解面，可用性优先。
func TestPublicBrandRateLimitFailsOpenWithoutRedis(t *testing.T) {
	engine := newPublicBrandRateLimitedEngine(nil)

	assert.Equal(t, http.StatusNoContent, callPublicBrandRateLimited(engine))
}
