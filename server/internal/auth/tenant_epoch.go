package auth

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// tenantEpochKeyPrefix 是园所账号的凭证失效记录键前缀（#182）。
// 与平台前缀 platform:auth_epoch: 分属两个信任域，键空间不得重叠。
const tenantEpochKeyPrefix = "tenant:auth_epoch:"

// TenantEpochTTLFor 按会话配置推导园所 epoch 键的 TTL。
// 与平台共用同一会话生命周期公式（refresh TTL + access TTL + 1 天）：
// epoch 必须先于任何可续期凭证消失，否则调大 refresh_ttl_day 后旧续期
// 凭证会复活（design D5）。
func TenantEpochTTLFor(refreshTTL, accessTTL time.Duration) time.Duration {
	return PlatformEpochTTLFor(refreshTTL, accessTTL)
}

// TenantEpochStore 记录园所账号的凭证失效时间。自助改密、管理员重置密码
// 与管理员删除都会写入 epoch；AuthJWT 与续期端点据此拒绝此前签发的凭证。
//
// Redis 淘汰/清空的影响：epoch 与会话数据（refresh:、吊销黑名单）存于同一
// Redis。键被淘汰或 FLUSH 后失效记录消失，已签发但未过期的旧 access 会在
// 剩余 TTL 内重新可用；refresh 续期仍受口令指纹、角色复核等既有校验约束，
// 但 epoch 这道兜底已不存在。TTL（refresh + access + 1 天）覆盖完整会话
// 生命周期；部署时须为会话 Redis 预留内存，避免 allkeys-lru 一类淘汰策略。
type TenantEpochStore interface {
	// TenantEpoch 返回该账号最近一次改密/重置/删除的生效时间；
	// 没有记录时返回零值（表示从未失效）。
	TenantEpoch(ctx context.Context, userID string) (time.Time, error)
	// BumpTenantEpoch 记录一次失效，覆盖在此时间之前签发的凭证。
	BumpTenantEpoch(ctx context.Context, userID string, at time.Time) error
}

type redisTenantEpochStore struct {
	rdb *redis.Client
	ttl time.Duration
}

// NewTenantEpochStore 返回 Redis 实现；没有 Redis 时返回 nil，
// 调用方须失败关闭（认证基础设施不可用时园所接口不可用）。
// ttl 由调用方按会话配置推导（见 TenantEpochTTLFor），非正数时取默认值。
func NewTenantEpochStore(rdb *redis.Client, ttl time.Duration) TenantEpochStore {
	if rdb == nil {
		return nil
	}
	if ttl <= 0 {
		ttl = TenantEpochTTLFor(0, 0)
	}
	return &redisTenantEpochStore{rdb: rdb, ttl: ttl}
}

func (s *redisTenantEpochStore) TenantEpoch(ctx context.Context, userID string) (time.Time, error) {
	if userID == "" {
		return time.Time{}, nil
	}
	sec, err := s.rdb.Get(ctx, tenantEpochKeyPrefix+userID).Int64()
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

func (s *redisTenantEpochStore) BumpTenantEpoch(ctx context.Context, userID string, at time.Time) error {
	if userID == "" {
		return errors.New("tenant epoch: empty user id")
	}
	// 秒级精度与 JWT iat 对齐：中间件只比较「签发时间是否早于 epoch」。
	return s.rdb.Set(ctx, tenantEpochKeyPrefix+userID, at.Unix(), s.ttl).Err()
}
