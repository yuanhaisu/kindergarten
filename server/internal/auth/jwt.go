package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ScopePlatform 标记平台方凭证。园所凭证不带 scope（空串），
// 两类凭证互不通用（ADR-0014）。
const ScopePlatform = "platform"

// ClientScope 描述一枚 access token 来自哪个小程序客户端（#163）。
// Kind/ID 对应 clients.scope_kind / clients.scope_id：shared 的 ID 恒为 0。
//
// OpenID 是 (app_id, open_id) 这枚客户端身份的另一半：绑定手机号（#167）需要
// 在「待绑定账号已被硬删」的重试里仍能定位身份，只有 JWT 能携带它。
type ClientScope struct {
	AppID  string
	Kind   string
	ID     uint64
	OpenID string
}

type AccessClaims struct {
	UserID         string   `json:"uid"`
	Roles          []string `json:"roles"`
	KindergartenID uint64   `json:"kindergarten_id,omitempty"`
	Scope          string   `json:"scope,omitempty"`
	// AppID + ClientScopeKind/ID 标记签发该 token 的小程序客户端。
	// 管理端与平台凭证不带这些字段（#166 另行设计管理端会话）。
	AppID           string `json:"app_id,omitempty"`
	ClientScopeKind string `json:"client_scope_kind,omitempty"`
	ClientScopeID   uint64 `json:"client_scope_id,omitempty"`
	// OpenID 是当前客户端身份（app_id + open_id）。绑定/换绑时据此定位
	// user_identities，服务端不从请求体采信 openid（#167）。
	OpenID string `json:"open_id,omitempty"`
	jwt.RegisteredClaims
}

func newRegisteredClaims(userID string, ttl time.Duration) jwt.RegisteredClaims {
	now := time.Now()
	return jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		Issuer:    "kindergarten-api",
		Subject:   userID,
		// 每次签发都换新 jti：登出只能吊销某一枚 token，不能按用户或前缀误伤。
		ID: uuid.NewString(),
	}
}

func signClaims(secret []byte, claims AccessClaims) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

func SignAccess(secret []byte, userID string, roles []string, kindergartenID uint64, ttl time.Duration) (string, error) {
	claims := AccessClaims{
		UserID:           userID,
		Roles:            roles,
		KindergartenID:   kindergartenID,
		RegisteredClaims: newRegisteredClaims(userID, ttl),
	}
	return signClaims(secret, claims)
}

// SignClientAccess 签发绑定小程序客户端的 access token（#163）。
// 未来的登录路径（#175）必须走这里，而不是 SignAccess——没有 app_id 的
// 园所凭证无法回查客户端作用域与角色裁剪。
func SignClientAccess(secret []byte, userID string, roles []string, kindergartenID uint64, scope ClientScope, ttl time.Duration) (string, error) {
	if scope.AppID == "" {
		return "", errors.New("app_id 不能为空：客户端作用域 token 必须绑定小程序客户端")
	}
	if scope.Kind == "" {
		return "", errors.New("client_scope_kind 不能为空")
	}
	claims := AccessClaims{
		UserID:           userID,
		Roles:            roles,
		KindergartenID:   kindergartenID,
		AppID:            scope.AppID,
		ClientScopeKind:  scope.Kind,
		ClientScopeID:    scope.ID,
		OpenID:           scope.OpenID,
		RegisteredClaims: newRegisteredClaims(userID, ttl),
	}
	return signClaims(secret, claims)
}

// SignPlatformAccess 签发平台方凭证：带 scope=platform，不含 kindergarten_id。
func SignPlatformAccess(secret []byte, adminID string, ttl time.Duration) (string, error) {
	claims := AccessClaims{
		UserID:           adminID,
		Roles:            []string{"platform"},
		Scope:            ScopePlatform,
		RegisteredClaims: newRegisteredClaims(adminID, ttl),
	}
	return signClaims(secret, claims)
}

func ParseAccess(secret []byte, tokenStr string) (*AccessClaims, error) {
	t, err := jwt.ParseWithClaims(tokenStr, &AccessClaims{}, func(t *jwt.Token) (any, error) {
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	c, ok := t.Claims.(*AccessClaims)
	if !ok || !t.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return c, nil
}
