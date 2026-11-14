package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/domain"
)

type fakeConsentRepo struct {
	rows      map[string]*domain.UserConsent
	createErr error
	findErr   error
}

func newFakeConsentRepo() *fakeConsentRepo {
	return &fakeConsentRepo{rows: map[string]*domain.UserConsent{}}
}

func consentKey(userID uint64, version string) string {
	return fmt.Sprintf("%d:%s", userID, version)
}

func (f *fakeConsentRepo) FindByUserAndVersion(_ context.Context, userID uint64, version string) (*domain.UserConsent, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	return f.rows[consentKey(userID, version)], nil
}

func (f *fakeConsentRepo) CreateIfAbsent(_ context.Context, c *domain.UserConsent) error {
	if f.createErr != nil {
		return f.createErr
	}
	k := consentKey(c.UserID, c.PolicyVersion)
	if _, ok := f.rows[k]; !ok {
		f.rows[k] = c
	}
	return nil
}

func (f *fakeConsentRepo) ListByUser(_ context.Context, userID uint64) ([]domain.UserConsent, error) {
	var out []domain.UserConsent
	for _, c := range f.rows {
		if c.UserID == userID {
			out = append(out, *c)
		}
	}
	return out, nil
}

func TestConsentService_RecordIsIdempotent(t *testing.T) {
	repo := newFakeConsentRepo()
	svc := &AuthService{consentRepo: repo, consentEnforce: true}
	ctx := context.Background()

	first, err := svc.RecordConsent(ctx, 7, CurrentPrivacyPolicyVersion, domain.ConsentSourceParent, "127.0.0.1", "test")
	require.NoError(t, err)
	require.True(t, first.Consented)
	require.NotNil(t, first.ConsentedAt)

	second, err := svc.RecordConsent(ctx, 7, CurrentPrivacyPolicyVersion, domain.ConsentSourceParent, "10.0.0.1", "again")
	require.NoError(t, err)
	require.True(t, second.Consented)
	require.WithinDuration(t, *first.ConsentedAt, *second.ConsentedAt, time.Second)
	require.Len(t, repo.rows, 1)
}

func TestConsentService_RejectsVersionMismatch(t *testing.T) {
	svc := &AuthService{consentRepo: newFakeConsentRepo(), consentEnforce: true}
	_, err := svc.RecordConsent(context.Background(), 7, "v0", domain.ConsentSourceParent, "", "")
	require.ErrorIs(t, err, ErrConsentVersionMismatch)
}

func TestConsentService_RejectsUnknownSource(t *testing.T) {
	svc := &AuthService{consentRepo: newFakeConsentRepo(), consentEnforce: true}
	_, err := svc.RecordConsent(context.Background(), 7, CurrentPrivacyPolicyVersion, "unknown", "", "")
	require.ErrorIs(t, err, ErrConsentSourceInvalid)
}

func TestConsentService_OldVersionDoesNotCountAsCurrent(t *testing.T) {
	repo := newFakeConsentRepo()
	repo.rows[consentKey(7, "v0")] = &domain.UserConsent{
		UserID:        7,
		PolicyVersion: "v0",
		Source:        domain.ConsentSourceTeacher,
		ConsentedAt:   time.Now().Add(-24 * time.Hour),
	}
	svc := &AuthService{consentRepo: repo, consentEnforce: true}
	ctx := context.Background()

	ok, err := svc.HasConsented(ctx, 7)
	require.NoError(t, err)
	require.False(t, ok)

	status, err := svc.ConsentStatus(ctx, 7)
	require.NoError(t, err)
	require.False(t, status.Consented)
	require.Equal(t, CurrentPrivacyPolicyVersion, status.PolicyVersion)

	_, err = svc.RecordConsent(ctx, 7, CurrentPrivacyPolicyVersion, domain.ConsentSourceTeacher, "", "")
	require.NoError(t, err)

	ok, err = svc.HasConsented(ctx, 7)
	require.NoError(t, err)
	require.True(t, ok)
	require.Len(t, repo.rows, 2, "旧版本记录必须保留")
}

func TestConsentService_ErrorsArePropagated(t *testing.T) {
	repo := newFakeConsentRepo()
	repo.findErr = errors.New("db down")
	svc := &AuthService{consentRepo: repo, consentEnforce: true}
	_, err := svc.HasConsented(context.Background(), 7)
	require.Error(t, err)
}

func TestConsentService_LoginFlag(t *testing.T) {
	ctx := context.Background()

	// 未装配存储（既有单测）时视为无需同意，不改变旧行为。
	svc := &AuthService{}
	required, err := svc.consentRequiredFor(ctx, 7)
	require.NoError(t, err)
	require.False(t, required)

	// 没有当前版本记录 -> 需要同意。
	svc = &AuthService{consentRepo: newFakeConsentRepo()}
	required, err = svc.consentRequiredFor(ctx, 7)
	require.NoError(t, err)
	require.True(t, required)

	_, err = svc.RecordConsent(ctx, 7, CurrentPrivacyPolicyVersion, domain.ConsentSourceParent, "", "")
	require.NoError(t, err)

	// 已同意 -> 登录响应不再要求同意。
	required, err = svc.consentRequiredFor(ctx, 7)
	require.NoError(t, err)
	require.False(t, required)
}
