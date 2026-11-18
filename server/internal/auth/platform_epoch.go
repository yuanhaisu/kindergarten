package auth

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

const platformEpochKeyPrefix = "platform:auth_epoch:"

// 会话配置缺省值，仅用于 TTL 推导的兜底；实际部署由 jwt.access_ttl_min /
// jwt.refresh_ttl_day 提供。
const (
	defaultPlatformRefreshTTL = 7 * 24 * time.Hour
	defaultPlatformAccessTTL  = 15 * time.Minute
)

// PlatformEpochTTLFor 按会话配置推导 epoch 键的 TTL（design D5：
// refresh TTL + access TTL + 1 天）。必须始终长于续期凭证，否则调大
// refresh_ttl_day 后 epoch 会先过期，旧续期凭证复活。
func PlatformEpochTTLFor(refreshTTL, accessTTL time.Duration) time.Duration {
	if refreshTTL <= 0 {
		refreshTTL = defaultPlatformRefreshTTL
	}
	if accessTTL <= 0 {
		accessTTL = defaultPlatformAccessTTL
	}
	return refreshTTL + accessTTL + 24*time.Hour
}

// PlatformEpochStore 记录平台账号的凭证失效时间。改密、停用与重置密码
// 写入 epoch，平台访问中间件与续期端点据此拒绝此前签发的凭证。
type PlatformEpochStore interface {
	// PlatformEpoch 返回该账号最近一次改密/停用/重置的生效时间；
	// 没有记录时返回零值（表示从未失效）。
	PlatformEpoch(ctx context.Context, adminID string) (time.Time, error)
	// BumpPlatformEpoch 记录一次失效，覆盖在此时间之前签发的凭证。
	BumpPlatformEpoch(ctx context.Context, adminID string, at time.Time) error
}

type redisPlatformEpochStore struct {
	rdb *redis.Client
	ttl time.Duration
}

// NewPlatformEpochStore 返回 Redis 实现；没有 Redis 时返回 nil，
// 调用方须失败关闭（design D5：Redis 不可用时平台接口不可用）。
// ttl 由调用方按会话配置推导（见 PlatformEpochTTLFor），非正数时取默认值。
func NewPlatformEpochStore(rdb *redis.Client, ttl time.Duration) PlatformEpochStore {
	if rdb == nil {
		return nil
	}
	if ttl <= 0 {
		ttl = PlatformEpochTTLFor(0, 0)
	}
	return &redisPlatformEpochStore{rdb: rdb, ttl: ttl}
}

func (s *redisPlatformEpochStore) PlatformEpoch(ctx context.Context, adminID string) (time.Time, error) {
	if adminID == "" {
		return time.Time{}, nil
	}
	sec, err := s.rdb.Get(ctx, platformEpochKeyPrefix+adminID).Int64()
	if errors.Is(err, redis.Nil) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}
	if sec <= 0 {
		return time.Time{}, nil
	}
	return time.Unix(sec, 0), nil
}

func (s *redisPlatformEpochStore) BumpPlatformEpoch(ctx context.Context, adminID string, at time.Time) error {
	if adminID == "" {
		return errors.New("platform epoch: empty admin id")
	}
	// 秒级精度与 JWT iat 对齐：中间件只比较「签发时间是否早于 epoch」。
	return s.rdb.Set(ctx, platformEpochKeyPrefix+adminID, at.Unix(), s.ttl).Err()
}
