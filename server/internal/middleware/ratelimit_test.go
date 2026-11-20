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
