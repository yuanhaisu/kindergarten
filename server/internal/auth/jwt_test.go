package auth

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const jwtTestSecret = "jwt-test-secret"

func TestSignAccessAssignsFreshJTI(t *testing.T) {
	a, err := SignAccess([]byte(jwtTestSecret), "1", []string{"parent"}, 4, time.Minute)
	require.NoError(t, err)
	b, err := SignAccess([]byte(jwtTestSecret), "1", []string{"parent"}, 4, time.Minute)
	require.NoError(t, err)

	ca, err := ParseAccess([]byte(jwtTestSecret), a)
	require.NoError(t, err)
	cb, err := ParseAccess([]byte(jwtTestSecret), b)
	require.NoError(t, err)

	assert.NotEmpty(t, ca.ID, "新签发的 token 必须带 jti")
	assert.NotEqual(t, ca.ID, cb.ID, "每次签发都必须换新 jti，否则登出会误伤其他会话")
}

func TestParseAccessAcceptsLegacyTokenWithoutJTI(t *testing.T) {
	now := time.Now()
	claims := AccessClaims{
		UserID: "1",
		Roles:  []string{"parent"},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "kindergarten-api",
			Subject:   "1",
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(jwtTestSecret))
	require.NoError(t, err)

	parsed, err := ParseAccess([]byte(jwtTestSecret), token)
	require.NoError(t, err)
	assert.Empty(t, parsed.ID)
}

func TestRevokerSkipsEmptyJTIAndNonPositiveTTL(t *testing.T) {
	assert.Nil(t, NewTokenRevoker(nil), "没有 Redis 时应显式返回 nil，让调用方走失败关闭")

	// 零值 revoker 的 rdb 是 nil：下面两个调用必须在解引用前返回，
	// 否则空 jti / 已过期 token 会在登出时 panic。
	r := &redisTokenRevoker{}
	ctx := context.Background()
	require.NoError(t, r.Revoke(ctx, "", time.Minute))
	require.NoError(t, r.Revoke(ctx, "some-jti", 0))

	revoked, err := r.Revoked(ctx, "")
	require.NoError(t, err)
	assert.False(t, revoked)
}
