package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/domain"
)

// stubAuthUserRepo 只实现 MustChangePassword 用到的 FindByID；
// 其余方法保持空实现以满足 AuthUserRepository 接口。
type stubAuthUserRepo struct {
	user *domain.User
	err  error
}

func (s *stubAuthUserRepo) FindByPhone(context.Context, string) (*domain.User, error) {
	return nil, nil
}

func (s *stubAuthUserRepo) FindByID(context.Context, uint64) (*domain.User, error) {
	return s.user, s.err
}

func (s *stubAuthUserRepo) Create(context.Context, *domain.User) error { return nil }
func (s *stubAuthUserRepo) Update(context.Context, *domain.User) error { return nil }
func (s *stubAuthUserRepo) UpdateFields(context.Context, uint64, map[string]any) error {
	return nil
}

// 首登改密门禁的真相源是 users.must_change_password，每请求读一次；
// 改密后由账号级失效（#182）让旧 access 失效，客户端重新登录。
func TestAuthServiceMustChangePasswordReadsUserRow(t *testing.T) {
	svc := &AuthService{userRepo: &stubAuthUserRepo{
		user: &domain.User{BaseModel: domain.BaseModel{ID: 42}, MustChangePassword: true},
	}}

	mustChange, err := svc.MustChangePassword(context.Background(), 42)
	require.NoError(t, err)
	assert.True(t, mustChange)
}

func TestAuthServiceMustChangePasswordFalseAfterChange(t *testing.T) {
	svc := &AuthService{userRepo: &stubAuthUserRepo{
		user: &domain.User{BaseModel: domain.BaseModel{ID: 42}},
	}}

	mustChange, err := svc.MustChangePassword(context.Background(), 42)
	require.NoError(t, err)
	assert.False(t, mustChange)
}

// 读取失败与账号不存在都必须返回 error：中间件据此 503 失败关闭，
// 不能把「查不到」解释成「不需要改密」。
func TestAuthServiceMustChangePasswordFailsClosed(t *testing.T) {
	t.Run("repository error", func(t *testing.T) {
		boom := errors.New("db down")
		svc := &AuthService{userRepo: &stubAuthUserRepo{err: boom}}

		_, err := svc.MustChangePassword(context.Background(), 42)
		assert.ErrorIs(t, err, boom)
	})

	t.Run("user not found", func(t *testing.T) {
		svc := &AuthService{userRepo: &stubAuthUserRepo{}}

		_, err := svc.MustChangePassword(context.Background(), 42)
		assert.Error(t, err)
	})
}

type stubAuthRoleRepo struct {
	roles []domain.UserRole
}

func (s *stubAuthRoleRepo) FindByUserID(context.Context, uint64) ([]domain.UserRole, error) {
	return s.roles, nil
}

// 硬刷新后管理端 store 为空，只能靠 profile 判断是否仍需强制改密（#199）。
// 该接口在门禁期间被豁免，正是为了让客户端读到这个标记。
func TestAuthServiceGetProfileCarriesMustChangePassword(t *testing.T) {
	svc := &AuthService{
		userRepo: &stubAuthUserRepo{
			user: &domain.User{BaseModel: domain.BaseModel{ID: 42}, MustChangePassword: true},
		},
		userRoleRepo: &stubAuthRoleRepo{},
	}

	profile, err := svc.GetProfile(context.Background(), 42, RoleScope{})
	require.NoError(t, err)
	assert.True(t, profile.MustChangePassword)
}

// 与登录响应同一口径：小程序凭证永远看不到管理端的改密标记。
func TestAuthServiceGetProfileHidesMustChangePasswordForClientScope(t *testing.T) {
	svc := &AuthService{
		userRepo: &stubAuthUserRepo{
			user: &domain.User{BaseModel: domain.BaseModel{ID: 42}, MustChangePassword: true},
		},
		userRoleRepo: &stubAuthRoleRepo{},
	}

	profile, err := svc.GetProfile(context.Background(), 42, RoleScope{clientScoped: true})
	require.NoError(t, err)
	assert.False(t, profile.MustChangePassword)
}
