package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

const entitlementCacheTTL = 5 * time.Minute

// ModuleAccess 单个模块在某一时刻的可用性快照。
type ModuleAccess struct {
	State     string    `json:"state"` // active / grace / expired
	ExpiresAt time.Time `json:"expires_at"`
}

// EntitlementCache 授权判定的 Redis 缓存。写路径（订单/收款/授权变更）必须失效，
// 保证续费即时恢复（design D5）。
type EntitlementCache struct {
	rdb *redis.Client
}

func NewEntitlementCache(rdb *redis.Client) *EntitlementCache {
	return &EntitlementCache{rdb: rdb}
}

func entitlementCacheKey(kgID uint64) string {
	return "ent:" + itoa(kgID)
}

func itoa(v uint64) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}

func (c *EntitlementCache) Get(ctx context.Context, kgID uint64) (map[string]ModuleAccess, bool) {
	if c == nil || c.rdb == nil {
		return nil, false
	}
	raw, err := c.rdb.Get(ctx, entitlementCacheKey(kgID)).Result()
	if err != nil {
		return nil, false
	}
	var out map[string]ModuleAccess
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, false
	}
	return out, true
}

func (c *EntitlementCache) Set(ctx context.Context, kgID uint64, value map[string]ModuleAccess) {
	if c == nil || c.rdb == nil {
		return
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return
	}
	_ = c.rdb.Set(ctx, entitlementCacheKey(kgID), raw, entitlementCacheTTL).Err()
}

func (c *EntitlementCache) Invalidate(ctx context.Context, kgID uint64) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.Del(ctx, entitlementCacheKey(kgID)).Err()
}
