package auth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/testutil/fakeredis"
)

// epoch 键必须活到续期凭证过期之后：只写死 9 天时，把 refresh_ttl_day
// 调到 30 天会让 epoch 先消失，改密/停用后的旧续期凭证随即复活（design D5）。
func TestPlatformEpochTTLFor_CoversRefreshAndAccess(t *testing.T) {
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
			ttl := PlatformEpochTTLFor(tc.refreshTTL, tc.accessTTL)
			assert.Greater(t, ttl, tc.refreshTTL+tc.accessTTL,
				"epoch TTL 必须覆盖完整会话生命周期并留余量")
		})
	}
}

func TestPlatformEpochStore_UsesConfiguredTTL(t *testing.T) {
	f := fakeredis.New(t)
	ttl := PlatformEpochTTLFor(30*24*time.Hour, 15*time.Minute)
	store := NewPlatformEpochStore(f.Client(t), ttl)
	require.NotNil(t, store)

	require.NoError(t, store.BumpPlatformEpoch(context.Background(), "42", time.Now()))

	ok, remaining := f.Exists(platformEpochKeyPrefix + "42")
	require.True(t, ok, "epoch 键必须写入")
	assert.InDelta(t, ttl.Seconds(), remaining.Seconds(), 2)
}

func TestPlatformEpochStore_FallsBackToDefaultTTL(t *testing.T) {
	f := fakeredis.New(t)
	store := NewPlatformEpochStore(f.Client(t), 0)
	require.NotNil(t, store)

	require.NoError(t, store.BumpPlatformEpoch(context.Background(), "7", time.Now()))

	ok, remaining := f.Exists(platformEpochKeyPrefix + "7")
	require.True(t, ok)
	assert.GreaterOrEqual(t, remaining, 7*24*time.Hour, "缺省 TTL 至少覆盖 7 天续期")
}
