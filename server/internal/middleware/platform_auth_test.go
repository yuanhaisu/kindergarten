package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/auth"
)

// stubEpochs 是平台账号 epoch 的替身。平台凭证失效只取决于「token 签发时间
// 与账号最近一次改密/停用时间」的比较，用桩即可覆盖，不依赖真实 Redis。
type stubEpochs struct {
	epoch time.Time
	err   error
}

func (s *stubEpochs) PlatformEpoch(_ context.Context, _ string) (time.Time, error) {
	return s.epoch, s.err
}

func (s *stubEpochs) BumpPlatformEpoch(context.Context, string, time.Time) error { return s.err }

func newPlatformEngine(revoker auth.TokenRevoker, epochs auth.PlatformEpochStore) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("")
	g.Use(PlatformAuth([]byte(middlewarePlatformTestSecret), revoker, epochs))
	g.GET("/platform/protected", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	return r
}

func signedPlatformToken(t *testing.T) string {
	t.Helper()
	return signedPlatformTokenWith(t, middlewarePlatformTestSecret)
}

// signedPlatformTokenWith 用指定密钥签发平台 scope 的 token，用于跨信任域验签测试。
func signedPlatformTokenWith(t *testing.T, secret string) string {
	t.Helper()
	token, err := auth.SignPlatformAccess([]byte(secret), "7", time.Minute)
	require.NoError(t, err)
	return token
}

func doPlatformRequest(engine *gin.Engine, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/platform/protected", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

func TestPlatformAuthAcceptsPlatformToken(t *testing.T) {
	engine := newPlatformEngine(&stubRevoker{}, &stubEpochs{})
	w := doPlatformRequest(engine, signedPlatformToken(t))
	assert.Equal(t, http.StatusNoContent, w.Code)
}

// 两类凭证互不可代换：园所密钥签发的凭证进不了平台接口。
// #183 起两个域密钥不同，验签阶段即失败（401）。
func TestPlatformAuthRejectsTenantToken(t *testing.T) {
	engine := newPlatformEngine(&stubRevoker{}, &stubEpochs{})
	w := doPlatformRequest(engine, signedToken(t))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// 反方向：平台密钥签发的凭证进不了园所接口（验签失败，401）。
func TestAuthJWTRejectsPlatformToken(t *testing.T) {
	engine := newGuardedEngine(&stubRevoker{}, &stubTenantEpochs{})
	code := callProtected(engine, signedPlatformToken(t))
	assert.Equal(t, http.StatusUnauthorized, code)
}

// #183：平台 scope 的 token 若用园所密钥签发，PlatformAuth 必须拒绝（401）。
// 这是「共用一把密钥」退化的直接回归：一旦平台验收园所密钥，园所侧泄漏的
// 密钥就能伪造平台凭证。
func TestPlatformAuthRejectsPlatformScopeTokenSignedWithTenantKey(t *testing.T) {
	engine := newPlatformEngine(&stubRevoker{}, &stubEpochs{})
	w := doPlatformRequest(engine, signedPlatformTokenWith(t, middlewareTestSecret))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPlatformAuthRejectsRevokedToken(t *testing.T) {
	engine := newPlatformEngine(&stubRevoker{revoked: true}, &stubEpochs{})
	w := doPlatformRequest(engine, signedPlatformToken(t))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// 改密/停用/重置后，此前签发的访问凭证必须立即失效（design D5）：
// token 的 iat 早于账号 epoch 时返回 401，而不是继续放行。
func TestPlatformAuthRejectsTokenIssuedBeforeEpoch(t *testing.T) {
	engine := newPlatformEngine(&stubRevoker{}, &stubEpochs{epoch: time.Now().Add(time.Hour)})
	w := doPlatformRequest(engine, signedPlatformToken(t))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// 反向：epoch 早于 token 签发时间时凭证仍然有效。
func TestPlatformAuthAcceptsTokenIssuedAfterEpoch(t *testing.T) {
	engine := newPlatformEngine(&stubRevoker{}, &stubEpochs{epoch: time.Now().Add(-time.Hour)})
	w := doPlatformRequest(engine, signedPlatformToken(t))
	assert.Equal(t, http.StatusNoContent, w.Code)
}

// Redis 故障时必须失败关闭（503），不能把「读不到 epoch」当成「账号未失效」。
func TestPlatformAuthFailsClosedWhenEpochLookupFails(t *testing.T) {
	engine := newPlatformEngine(&stubRevoker{}, &stubEpochs{err: errors.New("redis down")})
	w := doPlatformRequest(engine, signedPlatformToken(t))
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// 未配置 epoch 存储同样失败关闭，避免装配遗漏时静默跳过失效检查。
func TestPlatformAuthFailsClosedWhenEpochStoreMissing(t *testing.T) {
	engine := newPlatformEngine(&stubRevoker{}, nil)
	w := doPlatformRequest(engine, signedPlatformToken(t))
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}
