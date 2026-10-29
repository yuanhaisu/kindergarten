package auth

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const tokenBlacklistKeyPrefix = "token_blacklist:"

// TokenRevoker 是 access/refresh token 的吊销存储。key 只用 token 的 jti，
// 不再用 token 字符串前缀——JWT 头部对所有 HS256 token 都相同，按前缀会误伤全员。
type TokenRevoker interface {
	// Revoked 报告 jti 是否已被吊销。空 jti（加 jti 之前签发的旧 token）视为未吊销。
	Revoked(ctx context.Context, jti string) (bool, error)
	// Revoke 写入黑名单，ttl 应覆盖 token 的剩余有效期。
	Revoke(ctx context.Context, jti string, ttl time.Duration) error
}

type redisTokenRevoker struct {
	rdb *redis.Client
}

func NewTokenRevoker(rdb *redis.Client) TokenRevoker {
	if rdb == nil {
		return nil
	}
	return &redisTokenRevoker{rdb: rdb}
}

func (r *redisTokenRevoker) Revoked(ctx context.Context, jti string) (bool, error) {
	if jti == "" {
		return false, nil
	}
	n, err := r.rdb.Exists(ctx, tokenBlacklistKeyPrefix+jti).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *redisTokenRevoker) Revoke(ctx context.Context, jti string, ttl time.Duration) error {
	// 已过期或没有 jti 的 token 本来就不需要吊销。
	if jti == "" || ttl <= 0 {
		return nil
	}
	return r.rdb.Set(ctx, tokenBlacklistKeyPrefix+jti, "1", ttl).Err()
}
