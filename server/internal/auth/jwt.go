package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type AccessClaims struct {
	UserID         string   `json:"uid"`
	Roles          []string `json:"roles"`
	KindergartenID uint64   `json:"kindergarten_id,omitempty"`
	jwt.RegisteredClaims
}

func SignAccess(secret []byte, userID string, roles []string, kindergartenID uint64, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := AccessClaims{
		UserID:         userID,
		Roles:          roles,
		KindergartenID: kindergartenID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "kindergarten-api",
			Subject:   userID,
			// 每次签发都换新 jti：登出只能吊销某一枚 token，不能按用户或前缀误伤。
			ID: uuid.NewString(),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(secret)
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
