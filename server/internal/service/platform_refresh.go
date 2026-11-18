package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/domain"
)

// 平台会话相关错误。handler 把它们映射成 401 / 503，不能混进「内部错误」。
var (
	// ErrPlatformRefreshInvalid 续期凭证无效：不存在、已消费、已吊销、
	// 账号已停用，或签发时间早于账号最近一次改密/停用（design D5）。
	ErrPlatformRefreshInvalid = errors.New("PLATFORM_REFRESH_INVALID")
	// ErrPlatformAuthStorageUnavailable 平台会话存储（Redis）不可用。
	// 平台会话一律失败关闭，调用方映射为 503。
	ErrPlatformAuthStorageUnavailable = errors.New("PLATFORM_AUTH_STORAGE_UNAVAILABLE")
)

// issuePlatformSession 签发平台访问凭证与续期凭证（design D5）：
// access TTL 来自 jwt.access_ttl_min，refresh 存 refresh:platform:{jti}，
// 值格式 adminID:issuedAtUnix，TTL 来自 jwt.refresh_ttl_day。
func (s *PlatformService) issuePlatformSession(ctx context.Context, admin *domain.PlatformAdmin) (*PlatformLoginResult, error) {
	if s.rdb == nil {
		return nil, fmt.Errorf("%w: platform refresh store is not configured", ErrPlatformAuthStorageUnavailable)
	}
	adminID := strconv.FormatUint(admin.ID, 10)
	token, err := auth.SignPlatformAccess(s.secret, adminID, s.atTTL)
	if err != nil {
		return nil, err
	}

	jti := uuid.NewString()
	value := fmt.Sprintf("%s:%d", adminID, time.Now().Unix())
	if err := auth.SaveRefreshScoped(ctx, s.rdb, auth.PlatformRefreshScope, jti, value, s.rtTTL); err != nil {
		return nil, fmt.Errorf("%w: save platform refresh: %v", ErrPlatformAuthStorageUnavailable, err)
	}

	return &PlatformLoginResult{
		Token:        token,
		RefreshToken: jti,
		ExpiresIn:    int64(s.atTTL.Seconds()),
		Admin:        PlatformAdminInfo{ID: admin.ID, Username: admin.Username, Name: admin.Name},
	}, nil
}

// Refresh 用续期凭证换取新的访问凭证，并轮换续期凭证。
// 续期必须重新校验账号状态与签发时间，否则停用/改密后的旧凭
// 仍能换出 epoch 之后的新 access，账号级失效形同虚设（design D5）。
func (s *PlatformService) Refresh(ctx context.Context, refreshToken string) (*PlatformLoginResult, error) {
	if refreshToken == "" {
		return nil, ErrPlatformRefreshInvalid
	}
	if s.rdb == nil || s.revoker == nil || s.epochs == nil {
		return nil, fmt.Errorf("%w: platform auth dependencies are not configured", ErrPlatformAuthStorageUnavailable)
	}

	revoked, err := s.revoker.Revoked(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("%w: check platform refresh revocation: %v", ErrPlatformAuthStorageUnavailable, err)
	}
	if revoked {
		return nil, ErrPlatformRefreshInvalid
	}

	value, err := auth.ConsumeRefreshScoped(ctx, s.rdb, auth.PlatformRefreshScope, refreshToken)
	if errors.Is(err, redis.Nil) {
		return nil, ErrPlatformRefreshInvalid
	}
	if err != nil {
		return nil, fmt.Errorf("%w: consume platform refresh: %v", ErrPlatformAuthStorageUnavailable, err)
	}

	adminIDStr, issuedAtUnix, err := parsePlatformRefreshValue(value)
	if err != nil {
		return nil, ErrPlatformRefreshInvalid
	}
	adminID, err := strconv.ParseUint(adminIDStr, 10, 64)
	if err != nil {
		return nil, ErrPlatformRefreshInvalid
	}

	admin, err := s.adminRepo.FindByID(ctx, adminID)
	if err != nil {
		return nil, err
	}
	if admin == nil || admin.Status != 1 {
		return nil, ErrPlatformRefreshInvalid
	}

	epoch, err := s.epochs.PlatformEpoch(ctx, adminIDStr)
	if err != nil {
		return nil, fmt.Errorf("%w: read platform epoch: %v", ErrPlatformAuthStorageUnavailable, err)
	}
	// 秒级精度：与 JWT iat、epoch 存储保持一致；等于 epoch 的凭证仍有效。
	if time.Unix(issuedAtUnix, 0).Before(epoch) {
		return nil, ErrPlatformRefreshInvalid
	}

	return s.issuePlatformSession(ctx, admin)
}

// Logout 吊销平台访问凭证与续期凭证。续期键是主凭据（删除即失效），
// 黑名单额外覆盖剩余 TTL，防止已读出的 jti 被重放（design D5）。
func (s *PlatformService) Logout(ctx context.Context, tokenStr, refreshToken string) error {
	if s.revoker == nil {
		return fmt.Errorf("%w: token revoker is not configured", ErrPlatformAuthStorageUnavailable)
	}

	if claims, err := auth.ParseAccess(s.secret, tokenStr); err == nil && claims.ID != "" && claims.ExpiresAt != nil {
		ttl := time.Until(claims.ExpiresAt.Time) + time.Second
		if err := s.revoker.Revoke(ctx, claims.ID, ttl); err != nil {
			return fmt.Errorf("%w: revoke platform access: %v", ErrPlatformAuthStorageUnavailable, err)
		}
	}

	if refreshToken == "" {
		return nil
	}

	ttl, err := auth.RefreshRemainingTTLScoped(ctx, s.rdb, auth.PlatformRefreshScope, refreshToken)
	if err != nil {
		return fmt.Errorf("%w: read platform refresh ttl: %v", ErrPlatformAuthStorageUnavailable, err)
	}
	// 先删键：键删除本身就是吊销，即使黑名单写入失败，会话也已结束。
	if _, err := auth.ConsumeRefreshScoped(ctx, s.rdb, auth.PlatformRefreshScope, refreshToken); err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("%w: delete platform refresh: %v", ErrPlatformAuthStorageUnavailable, err)
	}
	// 已经消费/过期的续期没有可吊销的会话，不必写黑名单。
	if ttl > 0 {
		if err := s.revoker.Revoke(ctx, refreshToken, ttl+time.Second); err != nil {
			return fmt.Errorf("%w: revoke platform refresh: %v", ErrPlatformAuthStorageUnavailable, err)
		}
	}
	return nil
}

// parsePlatformRefreshValue 解析 adminID:issuedAtUnix。
func parsePlatformRefreshValue(value string) (string, int64, error) {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 || parts[0] == "" {
		return "", 0, fmt.Errorf("invalid platform refresh value")
	}
	issuedAt, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || issuedAt <= 0 {
		return "", 0, fmt.Errorf("invalid platform refresh issued-at")
	}
	return parts[0], issuedAt, nil
}
