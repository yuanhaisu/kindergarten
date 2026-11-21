package auth

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// PlatformRefreshScope 标记平台控制台的续期凭证。园所续期不带作用域，
// 保持 refresh:{jti} 的既有键格式。
const PlatformRefreshScope = "platform"

// RefreshScopeAdminWeb 是管理端密码登录的续期作用域。管理端不进入 clients
// 表，#166 用它把 admin_web 与小程序客户端区分开；app_id 为空。
const RefreshScopeAdminWeb = "admin_web"

// RefreshScope 是园所续期凭证的载荷（#169）。
//
// 存 JSON 而不是裸 userID：刷新时必须知道「哪个客户端、哪个园」，否则多客户端
// 账号会被重新推导到错误的园甚至别的角色族。admin_web 的 app_id 为空，裸拼接
// 会产生歧义，因此不做分隔符拼接。
type RefreshScope struct {
	UID            uint64 `json:"uid"`
	AppID          string `json:"app_id"`
	ScopeKind      string `json:"scope_kind"`
	ScopeID        uint64 `json:"scope_id"`
	KindergartenID uint64 `json:"kindergarten_id"`
	// OpenID 让续期后的 token 仍能定位客户端身份。绑定手机号（#167）依赖
	// claims 里的 open_id；待绑定用户可能在绑定前经历一次续期。
	OpenID string `json:"open_id,omitempty"`
}

func EncodeRefreshScope(scope RefreshScope) (string, error) {
	payload, err := json.Marshal(scope)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

// DecodeRefreshScope 解析续期载荷。旧版本存的裸 userID 不是 JSON，
// 会在这里失败——产品未上线，不保留兼容。
func DecodeRefreshScope(value string) (*RefreshScope, error) {
	var scope RefreshScope
	if err := json.Unmarshal([]byte(value), &scope); err != nil {
		return nil, errors.New("refresh payload is not a scoped JSON object")
	}
	if scope.UID == 0 {
		return nil, errors.New("refresh payload missing uid")
	}
	return &scope, nil
}

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
//
// 必须原子（GETDEL）：切换与续期可能同时拿着同一个旧 refresh，先 GET 再 DEL
// 会让两个请求都拿到值，各自换出一套凭证。删除保证轮换后旧 jti 不能重放。
func ConsumeRefreshScoped(ctx context.Context, rdb *redis.Client, scope, jti string) (string, error) {
	return rdb.GetDel(ctx, refreshKey(scope, jti)).Result()
}

func ConsumeRefresh(ctx context.Context, rdb *redis.Client, jti string) (string, error) {
	return ConsumeRefreshScoped(ctx, rdb, "", jti)
}
