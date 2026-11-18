package auth

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// PlatformRefreshScope 标记平台控制台的续期凭证。园所续期不带作用域，
// 保持 refresh:{jti} 的既有键格式。
const PlatformRefreshScope = "platform"

// refreshKey 按作用域拼存储键：空作用域是园所（refresh:{jti}），
// 平台是 refresh:platform:{jti}。两类键永不重叠，端点只能消费自己的前缀。
func refreshKey(scope, jti string) string {
	if scope == "" {
		return "refresh:" + jti
	}
	return "refresh:" + scope + ":" + jti
}

// SaveRefreshScoped 保存续期凭证。value 由调用方定义：园所存 userID，
// 平台存 adminID:issuedAtUnix（design D5）。
func SaveRefreshScoped(ctx context.Context, rdb *redis.Client, scope, jti, value string, ttl time.Duration) error {
	return rdb.Set(ctx, refreshKey(scope, jti), value, ttl).Err()
}

func SaveRefresh(ctx context.Context, rdb *redis.Client, jti, userID string, ttl time.Duration) error {
	return SaveRefreshScoped(ctx, rdb, "", jti, userID, ttl)
}

// RefreshRemainingTTLScoped 返回指定作用域下 refresh token 的剩余有效期。
// 不存在或没有过期时间时返回 0，表示没有可吊销的会话。
func RefreshRemainingTTLScoped(ctx context.Context, rdb *redis.Client, scope, jti string) (time.Duration, error) {
	if rdb == nil || jti == "" {
		return 0, nil
	}
	ttl, err := rdb.TTL(ctx, refreshKey(scope, jti)).Result()
	if err != nil {
		return 0, err
	}
	if ttl < 0 {
		return 0, nil
	}
	return ttl, nil
}

func RefreshRemainingTTL(ctx context.Context, rdb *redis.Client, jti string) (time.Duration, error) {
	return RefreshRemainingTTLScoped(ctx, rdb, "", jti)
}

// ConsumeRefreshScoped 读取并删除指定作用域的 refresh token。
// 删除保证续期轮换后旧 jti 不能重放。
func ConsumeRefreshScoped(ctx context.Context, rdb *redis.Client, scope, jti string) (string, error) {
	key := refreshKey(scope, jti)
	value, err := rdb.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}
	_ = rdb.Del(ctx, key).Err()
	return value, nil
}

func ConsumeRefresh(ctx context.Context, rdb *redis.Client, jti string) (string, error) {
	return ConsumeRefreshScoped(ctx, rdb, "", jti)
}
