package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/auth"
)

const middlewareTestSecret = "middleware-test-secret"

type stubRevoker struct {
	revoked bool
	err     error
	calls   []string
}

func (s *stubRevoker) Revoked(_ context.Context, jti string) (bool, error) {
	s.calls = append(s.calls, jti)
	return s.revoked, s.err
}

func (s *stubRevoker) Revoke(context.Context, string, time.Duration) error { return nil }

// stubTenantEpochs 是园所账号级失效时间的替身：零值表示从未失效。
type stubTenantEpochs struct {
	epoch time.Time
	err   error
	calls []string
}

func (s *stubTenantEpochs) TenantEpoch(_ context.Context, userID string) (time.Time, error) {
	s.calls = append(s.calls, userID)
	return s.epoch, s.err
}

func (s *stubTenantEpochs) BumpTenantEpoch(context.Context, string, time.Time) error { return nil }

func newGuardedEngine(revoker auth.TokenRevoker, epochs auth.TenantEpochStore) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("")
	g.Use(AuthJWT([]byte(middlewareTestSecret), revoker, epochs))
	g.GET("/protected", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	return r
}

func signedToken(t *testing.T) string {
	t.Helper()
	token, err := auth.SignAccess([]byte(middlewareTestSecret), "1", []string{"parent"}, 0, time.Minute)
	require.NoError(t, err)
	return token
}

func signedLegacyToken(t *testing.T) string {
	t.Helper()
	now := time.Now()
	claims := auth.AccessClaims{
		UserID: "1",
		Roles:  []string{"parent"},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "kindergarten-api",
			Subject:   "1",
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(middlewareTestSecret))
	require.NoError(t, err)
	return token
}

// signedAt 在指定秒级时间签发一枚带 jti 的 access token，
// 用于锁定「iat 等于 epoch」与「早于 epoch」的边界。
func signedAt(t *testing.T, at time.Time) string {
	t.Helper()
	claims := auth.AccessClaims{
		UserID: "1",
		Roles:  []string{"parent"},
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        "jti-at-" + at.UTC().Format("20060102150405"),
			ExpiresAt: jwt.NewNumericDate(at.Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(at),
			NotBefore: jwt.NewNumericDate(at),
			Issuer:    "kindergarten-api",
			Subject:   "1",
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(middlewareTestSecret))
	require.NoError(t, err)
	return token
}

func callProtected(engine *gin.Engine, token string) int {
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w.Code
}

func TestAuthJWTRejectsMissingOrInvalidToken(t *testing.T) {
	engine := newGuardedEngine(&stubRevoker{}, &stubTenantEpochs{})
	assert.Equal(t, http.StatusUnauthorized, callProtected(engine, ""))
	assert.Equal(t, http.StatusUnauthorized, callProtected(engine, "not-a-token"))
}

func TestAuthJWTAllowsValidTokenAndChecksItsJTI(t *testing.T) {
	revoker := &stubRevoker{}
	engine := newGuardedEngine(revoker, &stubTenantEpochs{})

	require.Equal(t, http.StatusNoContent, callProtected(engine, signedToken(t)))
	require.Len(t, revoker.calls, 1, "带 jti 的 token 必须查一次黑名单")
	assert.NotEmpty(t, revoker.calls[0])
}

func TestAuthJWTRejectsRevokedToken(t *testing.T) {
	engine := newGuardedEngine(&stubRevoker{revoked: true}, &stubTenantEpochs{})
	assert.Equal(t, http.StatusUnauthorized, callProtected(engine, signedToken(t)))
}

// Redis 故障失败关闭：返回 503，而不是放行。
func TestAuthJWTFailsClosedWhenRevokerErrors(t *testing.T) {
	engine := newGuardedEngine(&stubRevoker{err: errors.New("redis down")}, &stubTenantEpochs{})
	assert.Equal(t, http.StatusServiceUnavailable, callProtected(engine, signedToken(t)))
}

func TestAuthJWTRejectsWhenRevokerNotConfigured(t *testing.T) {
	engine := newGuardedEngine(nil, &stubTenantEpochs{})
	assert.Equal(t, http.StatusServiceUnavailable, callProtected(engine, signedToken(t)))
}

// 加 jti 之前签发的旧 token 无法判断吊销状态，仍按原逻辑放行到过期。
func TestAuthJWTAllowsLegacyTokenWithoutJTI(t *testing.T) {
	revoker := &stubRevoker{err: errors.New("redis down")}
	engine := newGuardedEngine(revoker, &stubTenantEpochs{})

	assert.Equal(t, http.StatusNoContent, callProtected(engine, signedLegacyToken(t)))
	assert.Empty(t, revoker.calls, "无 jti 的 token 不应触发黑名单查询")

	assert.Equal(t, http.StatusNoContent, callProtected(newGuardedEngine(nil, &stubTenantEpochs{}), signedLegacyToken(t)))
}

// #182：改密/重置/删除后写入 epoch，此前签发的 access 必须 401。
func TestAuthJWTRejectsTokenIssuedBeforeTenantEpoch(t *testing.T) {
	epochs := &stubTenantEpochs{epoch: time.Now().Add(time.Second)}
	engine := newGuardedEngine(&stubRevoker{}, epochs)

	assert.Equal(t, http.StatusUnauthorized, callProtected(engine, signedToken(t)))
	assert.NotEmpty(t, epochs.calls, "中间件必须按 token 里的 user_id 查 epoch")
}

// 等于 epoch 的凭证仍有效：epoch 使用秒级精度，同一秒内签发的新会话不能误伤。
func TestAuthJWTAllowsTokenIssuedAtTenantEpoch(t *testing.T) {
	now := time.Unix(time.Now().Unix(), 0)
	epochs := &stubTenantEpochs{epoch: now}
	engine := newGuardedEngine(&stubRevoker{}, epochs)

	assert.Equal(t, http.StatusNoContent, callProtected(engine, signedAt(t, now)))
}

// 没有 jti 的旧 token 也必须受账号级失效约束，否则停用后旧凭证仍可用。
func TestAuthJWTRejectsLegacyTokenIssuedBeforeTenantEpoch(t *testing.T) {
	engine := newGuardedEngine(&stubRevoker{}, &stubTenantEpochs{epoch: time.Now().Add(time.Second)})
	assert.Equal(t, http.StatusUnauthorized, callProtected(engine, signedLegacyToken(t)))
}

// Redis 读故障失败关闭：不能把「读不到 epoch」当成「未失效」而放行。
func TestAuthJWTFailsClosedWhenTenantEpochStoreErrors(t *testing.T) {
	engine := newGuardedEngine(&stubRevoker{}, &stubTenantEpochs{err: errors.New("redis down")})
	assert.Equal(t, http.StatusServiceUnavailable, callProtected(engine, signedToken(t)))
}

// 未装配 epoch 存储时同样失败关闭。
func TestAuthJWTRejectsWhenTenantEpochStoreNotConfigured(t *testing.T) {
	engine := newGuardedEngine(&stubRevoker{}, nil)
	assert.Equal(t, http.StatusServiceUnavailable, callProtected(engine, signedToken(t)))
}
