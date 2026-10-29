package auth

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

func SaveRefresh(ctx context.Context, rdb *redis.Client, jti, userID string, ttl time.Duration) error {
	return rdb.Set(ctx, "refresh:"+jti, userID, ttl).Err()
}

// RefreshRemainingTTL 返回 refresh token 在 Redis 中的剩余有效期。
// 不存在或没有过期时间时返回 0，表示没有可吊销的会话。
func RefreshRemainingTTL(ctx context.Context, rdb *redis.Client, jti string) (time.Duration, error) {
	if rdb == nil || jti == "" {
		return 0, nil
	}
	ttl, err := rdb.TTL(ctx, "refresh:"+jti).Result()
	if err != nil {
		return 0, err
	}
	if ttl < 0 {
		return 0, nil
	}
	return ttl, nil
}

func ConsumeRefresh(ctx context.Context, rdb *redis.Client, jti string) (string, error) {
	key := "refresh:" + jti
	userID, err := rdb.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}
	_ = rdb.Del(ctx, key).Err()
	return userID, nil
}
