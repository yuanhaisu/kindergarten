package service

import (
	"context"
	"errors"
	"time"

	"kindergarten-server/internal/domain"
)

// CurrentPrivacyPolicyVersion 当前隐私政策版本。官网 content/site.ts 的 policyVersion
// 必须与此值一致；备案通过、文本定稿后两处一起改为生效日期。
const CurrentPrivacyPolicyVersion = "v1"

var (
	ErrConsentVersionMismatch = errors.New("CONSENT_VERSION_MISMATCH")
	ErrConsentSourceInvalid   = errors.New("CONSENT_SOURCE_INVALID")
	ErrConsentStorageMissing  = errors.New("CONSENT_STORAGE_UNAVAILABLE")
)

// ConsentRepository AuthService 依赖的同意记录仓库（由 *repository.ConsentRepo 实现）。
type ConsentRepository interface {
	FindByUserAndVersion(ctx context.Context, userID uint64, version string) (*domain.UserConsent, error)
	CreateIfAbsent(ctx context.Context, c *domain.UserConsent) error
	ListByUser(ctx context.Context, userID uint64) ([]domain.UserConsent, error)
}

type ConsentStatus struct {
	PolicyVersion string     `json:"policy_version"`
	Consented     bool       `json:"consented"`
	ConsentedAt   *time.Time `json:"consented_at,omitempty"`
}

// ConsentChecker 供 consent 中间件使用；由 AuthService 实现。
type ConsentChecker interface {
	ConsentEnforced() bool
	HasConsented(ctx context.Context, userID uint64) (bool, error)
}

func (s *AuthService) ConsentEnforced() bool { return s.consentEnforce }

// consentRequiredFor 登录响应用的同意状态；storage 未装配时（单测）视为无需同意。
func (s *AuthService) consentRequiredFor(ctx context.Context, userID uint64) (bool, error) {
	if s.consentRepo == nil {
		return false, nil
	}
	c, err := s.consentRepo.FindByUserAndVersion(ctx, userID, CurrentPrivacyPolicyVersion)
	if err != nil {
		return false, err
	}
	return c == nil, nil
}

func (s *AuthService) HasConsented(ctx context.Context, userID uint64) (bool, error) {
	if s.consentRepo == nil {
		return true, nil
	}
	c, err := s.consentRepo.FindByUserAndVersion(ctx, userID, CurrentPrivacyPolicyVersion)
	if err != nil {
		return false, err
	}
	return c != nil, nil
}

func (s *AuthService) ConsentStatus(ctx context.Context, userID uint64) (*ConsentStatus, error) {
	if s.consentRepo == nil {
		return &ConsentStatus{PolicyVersion: CurrentPrivacyPolicyVersion}, nil
	}
	c, err := s.consentRepo.FindByUserAndVersion(ctx, userID, CurrentPrivacyPolicyVersion)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return &ConsentStatus{PolicyVersion: CurrentPrivacyPolicyVersion}, nil
	}
	t := c.ConsentedAt
	return &ConsentStatus{
		PolicyVersion: c.PolicyVersion,
		Consented:     true,
		ConsentedAt:   &t,
	}, nil
}

// RecordConsent 幂等记录同意；版本必须等于当前版本，避免旧页面同意旧政策。
func (s *AuthService) RecordConsent(
	ctx context.Context,
	userID uint64,
	version, source, ip, userAgent string,
) (*ConsentStatus, error) {
	if version != CurrentPrivacyPolicyVersion {
		return nil, ErrConsentVersionMismatch
	}
	switch source {
	case domain.ConsentSourceParent, domain.ConsentSourceTeacher, domain.ConsentSourceAdmin:
	default:
		return nil, ErrConsentSourceInvalid
	}
	if s.consentRepo == nil {
		return nil, ErrConsentStorageMissing
	}
	record := &domain.UserConsent{
		UserID:        userID,
		PolicyVersion: version,
		Source:        source,
		IP:            ip,
		UserAgent:     userAgent,
		ConsentedAt:   time.Now(),
	}
	if err := s.consentRepo.CreateIfAbsent(ctx, record); err != nil {
		return nil, err
	}
	return s.ConsentStatus(ctx, userID)
}
