package auth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/testutil/fakeredis"
)

// #182：园所会话失效记录复用平台 epoch 模式（design D5）。
// 键前缀是票面约定，测试用字面量钉死，防止实现漂移。

func TestTenantEpochTTLFor_CoversRefreshAndAccess(t *testing.T) {
	cases := []struct {
		name       string
		refreshTTL time.Duration
		accessTTL  time.Duration
	}{
		{"默认 7 天续期", 7 * 24 * time.Hour, 15 * time.Minute},
		{"调大到 30 天续期", 30 * 24 * time.Hour, 15 * time.Minute},
		{"调小到 1 小时续期", time.Hour, 5 * time.Minute},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ttl := TenantEpochTTLFor(tc.refreshTTL, tc.accessTTL)
			assert.Greater(t, ttl, tc.refreshTTL+tc.accessTTL,
				"epoch TTL 必须覆盖完整会话生命周期并留余量")
		})
	}
}

// 未写过 epoch 时返回零值且不报错：中间件据此放行（从未失效）。
func TestTenantEpochStore_AbsentEpochIsZero(t *testing.T) {
	f := fakeredis.New(t)
	store := NewTenantEpochStore(f.Client(t), TenantEpochTTLFor(7*24*time.Hour, 15*time.Minute))
	require.NotNil(t, store)

	epoch, err := store.TenantEpoch(context.Background(), "42")
	require.NoError(t, err)
	assert.True(t, epoch.IsZero())
}

func TestTenantEpochStore_RoundTripUsesTicketKeyPrefix(t *testing.T) {
	f := fakeredis.New(t)
	ttl := TenantEpochTTLFor(30*24*time.Hour, 15*time.Minute)
	store := NewTenantEpochStore(f.Client(t), ttl)
	require.NotNil(t, store)

	at := time.Unix(time.Now().Unix(), 0)
	require.NoError(t, store.BumpTenantEpoch(context.Background(), "42", at))

	ok, remaining := f.Exists("tenant:auth_epoch:42")
	require.True(t, ok, "键必须使用票面约定的 tenant:auth_epoch:<user_id> 前缀")
	assert.InDelta(t, ttl.Seconds(), remaining.Seconds(), 2, "TTL 必须覆盖完整会话生命周期")

	epoch, err := store.TenantEpoch(context.Background(), "42")
	require.NoError(t, err)
	assert.Equal(t, at.Unix(), epoch.Unix())
}

func TestTenantEpochStore_FallsBackToDefaultTTL(t *testing.T) {
	f := fakeredis.New(t)
	store := NewTenantEpochStore(f.Client(t), 0)
	require.NotNil(t, store)

	require.NoError(t, store.BumpTenantEpoch(context.Background(), "7", time.Now()))

	_, remaining := f.Exists("tenant:auth_epoch:7")
	assert.GreaterOrEqual(t, remaining, 7*24*time.Hour, "缺省 TTL 至少覆盖 7 天续期")
}

// 没有 Redis 时返回 nil：调用方（中间件/改密/重置/删除）必须失败关闭。
func TestNewTenantEpochStore_WithoutRedisReturnsNil(t *testing.T) {
	assert.Nil(t, NewTenantEpochStore(nil, time.Hour))
}

func TestTenantEpochStore_RejectsEmptyUserID(t *testing.T) {
	f := fakeredis.New(t)
	store := NewTenantEpochStore(f.Client(t), time.Hour)
	require.NotNil(t, store)

	assert.Error(t, store.BumpTenantEpoch(context.Background(), "", time.Now()))
}
