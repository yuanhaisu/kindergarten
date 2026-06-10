package auth

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

func SaveRefresh(ctx context.Context, rdb *redis.Client, jti, userID string, ttl time.Duration) error {
	return rdb.Set(ctx, "refresh:"+jti, userID, ttl).Err()
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
