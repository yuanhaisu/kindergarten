package auth

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/testutil/fakeredis"
)

// 平台续期凭证与园所续期凭证必须分键存储：平台端点只能消费
// refresh:platform:{jti}，园所端点只能消费 refresh:{jti}。
// 否则平台 refresh 可以在园所 /auth/refresh 换出园所访问凭证
// （design D5，TC-NEW-PLT-0011 的安全前提）。
func TestScopedRefreshSeparatesPlatformAndTenantKeys(t *testing.T) {
	f := fakeredis.New(t)
	rdb := f.Client(t)
	ctx := context.Background()

	require.NoError(t, SaveRefreshScoped(ctx, rdb, PlatformRefreshScope, "p1", "7:1700000000", 7*24*time.Hour))

	ok, ttl := f.Exists("refresh:platform:p1")
	require.True(t, ok, "平台续期必须写到 refresh:platform:{jti}")
	assert.Greater(t, ttl, 6*24*time.Hour, "平台续期 TTL 必须来自 refresh_ttl_day")

	// 园所端点只消费 refresh:{jti}：平台键原样保留。
	_, err := ConsumeRefresh(ctx, rdb, "p1")
	assert.ErrorIs(t, err, redis.Nil, "园所端点不能读到平台续期")
	ok, _ = f.Exists("refresh:platform:p1")
	assert.True(t, ok, "园所端点的消费不能删除平台键")

	// 平台端点也不消费园所键。
	require.NoError(t, SaveRefresh(ctx, rdb, "t1", "42", time.Hour))
	_, err = ConsumeRefreshScoped(ctx, rdb, PlatformRefreshScope, "t1")
	assert.ErrorIs(t, err, redis.Nil, "平台端点不能读到园所续期")
	ok, _ = f.Exists("refresh:t1")
	assert.True(t, ok, "平台端点的消费不能删除园所键")

	// 平台端点消费自己的键，并返回 adminID:issuedAtUnix。
	v, err := ConsumeRefreshScoped(ctx, rdb, PlatformRefreshScope, "p1")
	require.NoError(t, err)
	assert.Equal(t, "7:1700000000", v)
	ok, _ = f.Exists("refresh:platform:p1")
	assert.False(t, ok, "消费后续期键必须删除，旧 jti 不能重放")
}

// 平台登出按作用域读取剩余 TTL 时不能落到园所前缀：读错前缀会让
// 续期凭证的黑名单 TTL 变成 0，登出也就吊销不了它（design D5）。
func TestRefreshRemainingTTLScopedReadsScopedKey(t *testing.T) {
	f := fakeredis.New(t)
	rdb := f.Client(t)
	ctx := context.Background()

	require.NoError(t, SaveRefreshScoped(ctx, rdb, PlatformRefreshScope, "p2", "7:1700000000", 7*24*time.Hour))

	ttl, err := RefreshRemainingTTLScoped(ctx, rdb, PlatformRefreshScope, "p2")
	require.NoError(t, err)
	assert.Greater(t, ttl, 6*24*time.Hour)

	// 园所前缀下没有这个键，按作用域读取不能张冠李戴。
	tenantTTL, err := RefreshRemainingTTL(ctx, rdb, "p2")
	require.NoError(t, err)
	assert.Zero(t, tenantTTL)
}

// 空作用域必须保持园所的既有键格式，避免改动波及老师端/家长端。
func TestEmptyScopeKeepsTenantRefreshKey(t *testing.T) {
	f := fakeredis.New(t)
	rdb := f.Client(t)
	ctx := context.Background()

	require.NoError(t, SaveRefreshScoped(ctx, rdb, "", "t2", "42", time.Hour))
	ok, _ := f.Exists("refresh:t2")
	assert.True(t, ok)
}
