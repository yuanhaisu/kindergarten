package auth

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

const platformEpochKeyPrefix = "platform:auth_epoch:"

// PlatformEpochTTL 覆盖平台续期凭证的最长生命周期（design D5：refresh TTL
// + access TTL + 1 天）。默认 7 天续期 + 15 分钟访问，9 天留足余量；
// epoch 过期后旧凭证本身也已过期，不会因键消失而复活。
const PlatformEpochTTL = 9 * 24 * time.Hour

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
}

// NewPlatformEpochStore 返回 Redis 实现；没有 Redis 时返回 nil，
// 调用方须失败关闭（design D5：Redis 不可用时平台接口不可用）。
func NewPlatformEpochStore(rdb *redis.Client) PlatformEpochStore {
	if rdb == nil {
		return nil
	}
	return &redisPlatformEpochStore{rdb: rdb}
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
	return s.rdb.Set(ctx, platformEpochKeyPrefix+adminID, at.Unix(), PlatformEpochTTL).Err()
}
