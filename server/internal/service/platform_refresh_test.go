package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/internal/testutil/fakeredis"
)

const t7aTestAdmin = "t7a_ops"

// newPlatformRefreshService 装配真实的 PlatformService（含平台会话存储），
// Redis 用进程内替身，绝不连开发环境 Redis（ticket #119）。
func newPlatformRefreshService(t *testing.T, db *gorm.DB, rdb *redis.Client) *PlatformService {
	t.Helper()
	return NewPlatformService(
		db,
		repository.NewPlatformAdminRepo(db),
		repository.NewKindergartenRepo(db),
		repository.NewEntitlementRepo(db),
		repository.NewOrderRepo(db),
		repository.NewUserRepo(db),
		repository.NewUserRoleRepo(db),
		repository.NewAdminRepo(db),
		repository.NewPaymentRepo(db),
		repository.NewCommissionRepo(db),
		repository.NewDistributorRepo(db),
		repository.NewPlatformLogRepo(db),
		nil,
		nil,
		auth.NewPlatformEpochStore(rdb, auth.PlatformEpochTTLFor(7*24*time.Hour, 15*time.Minute)),
		auth.NewTokenRevoker(rdb),
		rdb,
		[]byte("test-platform-secret"),
		15*time.Minute,
		7*24*time.Hour,
	)
}

func seedT7AAdmin(t *testing.T, db *gorm.DB) *domain.PlatformAdmin {
	t.Helper()
	hash, err := HashPassword("OpsPassword1")
	require.NoError(t, err)
	admin := &domain.PlatformAdmin{Username: t7aTestAdmin, PasswordHash: hash, Name: "运营", Status: 1, Role: "platform"}
	require.NoError(t, repository.NewPlatformAdminRepo(db).Create(context.Background(), admin))
	t.Cleanup(func() {
		// 硬删：软删行会占用用户唯一索引，重跑会撞键。
		db.Unscoped().Where("username = ?", t7aTestAdmin).Delete(&domain.PlatformAdmin{})
	})
	return admin
}

// 平台登录必须签发续期凭证：15 分钟访问 + 7 天续期，续期键在
// refresh:platform:{jti}，值包含 adminID 与签发时间（design D5）。
func TestPlatformLoginIssuesPlatformRefreshToken(t *testing.T) {
	db := openPlatformTestDB(t)
	f := fakeredis.New(t)
	rdb := f.Client(t)
	svc := newPlatformRefreshService(t, db, rdb)
	admin := seedT7AAdmin(t, db)

	res, err := svc.Login(context.Background(), t7aTestAdmin, "OpsPassword1")
	require.NoError(t, err)
	assert.Equal(t, int64(900), res.ExpiresIn, "平台访问凭证 TTL 必须来自 jwt.access_ttl_min（15 分钟），不能再硬编码 24 小时")
	require.NotEmpty(t, res.RefreshToken, "登录响应必须带续期凭证")

	value, ok := f.Get("refresh:platform:" + res.RefreshToken)
	require.True(t, ok, "续期必须写到 refresh:platform:{jti}")
	parts := strings.SplitN(value, ":", 2)
	require.Len(t, parts, 2, "续期值必须是 adminID:issuedAtUnix，实际 %q", value)
	assert.Equal(t, fmt.Sprintf("%d", admin.ID), parts[0])
	issuedAt, err := time.ParseDuration(parts[1] + "s")
	require.NoError(t, err)
	assert.InDelta(t, time.Now().Unix(), int64(issuedAt.Seconds()), 5, "续期值必须记录签发时间")

	_, platformTTL := f.Exists("refresh:platform:" + res.RefreshToken)
	assert.Greater(t, platformTTL, 6*24*time.Hour, "续期 TTL 必须是 7 天")

	ok, _ = f.Exists("refresh:" + res.RefreshToken)
	assert.False(t, ok, "平台续期不能写到园所前缀")
}

// 续期必须轮换：旧 jti 消费后即失效，不能重放（TC-NEW-PLT-0011）。
func TestPlatformRefreshRotatesAndConsumesOldToken(t *testing.T) {
	db := openPlatformTestDB(t)
	f := fakeredis.New(t)
	rdb := f.Client(t)
	svc := newPlatformRefreshService(t, db, rdb)
	seedT7AAdmin(t, db)
	ctx := context.Background()

	login, err := svc.Login(ctx, t7aTestAdmin, "OpsPassword1")
	require.NoError(t, err)

	refreshed, err := svc.Refresh(ctx, login.RefreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, refreshed.Token)
	require.NotEmpty(t, refreshed.RefreshToken)
	assert.NotEqual(t, login.RefreshToken, refreshed.RefreshToken, "续期必须轮换续期凭证")
	assert.NotEqual(t, login.Token, refreshed.Token, "续期必须签发新的访问凭证")
	assert.Equal(t, int64(900), refreshed.ExpiresIn)

	// 旧续期已被消费，重放必须失败。
	_, err = svc.Refresh(ctx, login.RefreshToken)
	assert.Error(t, err, "旧 jti 消费后不能重放")

	// 新续期只能再用一次。
	_, err = svc.Refresh(ctx, refreshed.RefreshToken)
	assert.NoError(t, err)
	_, err = svc.Refresh(ctx, refreshed.RefreshToken)
	assert.Error(t, err)
}

// 续期必须校验账号仍可用：停用后旧续期被拒（TC-NEW-PLT-0023）。
func TestPlatformRefreshRejectsDisabledAccount(t *testing.T) {
	db := openPlatformTestDB(t)
	f := fakeredis.New(t)
	rdb := f.Client(t)
	svc := newPlatformRefreshService(t, db, rdb)
	admin := seedT7AAdmin(t, db)
	ctx := context.Background()

	login, err := svc.Login(ctx, t7aTestAdmin, "OpsPassword1")
	require.NoError(t, err)

	require.NoError(t, repository.NewPlatformAdminRepo(db).UpdateByID(ctx, admin.ID, map[string]any{"status": 0}))

	_, err = svc.Refresh(ctx, login.RefreshToken)
	assert.Error(t, err, "停用后的续期凭证必须被拒绝")
}

// 续期必须校验签发时间不早于账号 epoch：改密/停用后旧续期不能绕过失效（design D5）。
func TestPlatformRefreshRejectsIssuedBeforeEpoch(t *testing.T) {
	db := openPlatformTestDB(t)
	f := fakeredis.New(t)
	rdb := f.Client(t)
	svc := newPlatformRefreshService(t, db, rdb)
	admin := seedT7AAdmin(t, db)
	ctx := context.Background()

	login, err := svc.Login(ctx, t7aTestAdmin, "OpsPassword1")
	require.NoError(t, err)

	// epoch 晚于续期签发时间（秒级精度，留 2 秒余量）。
	require.NoError(t, auth.NewPlatformEpochStore(rdb, auth.PlatformEpochTTLFor(7*24*time.Hour, time.Hour)).BumpPlatformEpoch(ctx, fmt.Sprintf("%d", admin.ID), time.Now().Add(2*time.Second)))

	_, err = svc.Refresh(ctx, login.RefreshToken)
	assert.Error(t, err, "签发时间早于 epoch 的续期必须被拒绝")
}

// 平台登出必须吊销续期：删除平台键并写入黑名单（TC-NEW-PLT-0024）。
func TestPlatformLogoutRevokesPlatformRefresh(t *testing.T) {
	db := openPlatformTestDB(t)
	f := fakeredis.New(t)
	rdb := f.Client(t)
	svc := newPlatformRefreshService(t, db, rdb)
	seedT7AAdmin(t, db)
	ctx := context.Background()

	login, err := svc.Login(ctx, t7aTestAdmin, "OpsPassword1")
	require.NoError(t, err)
	require.True(t, mustExists(f, "refresh:platform:"+login.RefreshToken))

	require.NoError(t, svc.Logout(ctx, login.Token, login.RefreshToken))

	ok, _ := f.Exists("refresh:platform:" + login.RefreshToken)
	assert.False(t, ok, "登出必须删除平台续期键")
	ok, ttl := f.Exists("token_blacklist:" + login.RefreshToken)
	assert.True(t, ok, "登出必须把续期 jti 写入黑名单")
	assert.Greater(t, ttl, time.Duration(0), "黑名单必须覆盖续期剩余有效期")

	_, err = svc.Refresh(ctx, login.RefreshToken)
	assert.Error(t, err, "登出后的续期必须被拒绝")

	// 访问凭证也被吊销。
	revoker := auth.NewTokenRevoker(rdb)
	claims, err := auth.ParseAccess([]byte("test-platform-secret"), login.Token)
	require.NoError(t, err)
	revoked, err := revoker.Revoked(ctx, claims.ID)
	require.NoError(t, err)
	assert.True(t, revoked, "登出必须吊销当前访问凭证")
}

// 平台续期端点只消费平台前缀：园所 refresh 不能被平台端点兑换（design D5）。
func TestPlatformRefreshRejectsTenantRefreshToken(t *testing.T) {
	db := openPlatformTestDB(t)
	f := fakeredis.New(t)
	rdb := f.Client(t)
	svc := newPlatformRefreshService(t, db, rdb)
	seedT7AAdmin(t, db)
	ctx := context.Background()

	require.NoError(t, auth.SaveRefresh(ctx, rdb, "tenant-jti", "1", time.Hour))

	_, err := svc.Refresh(ctx, "tenant-jti")
	assert.Error(t, err, "平台端点不能消费园所续期凭证")
	ok, _ := f.Exists("refresh:tenant-jti")
	assert.True(t, ok, "平台端点不能删除园所键")
}

// Redis 不可用时登录与续期都必须失败关闭（design D5）。
func TestPlatformSessionFailsClosedWhenStorageUnavailable(t *testing.T) {
	f := fakeredis.New(t)
	rdb := f.Client(t)
	f.SetFailReads(true)
	f.SetFailWrites(true)
	svc := NewPlatformService(
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		auth.NewTokenRevoker(rdb), rdb, []byte("test-platform-secret"),
		15*time.Minute, 7*24*time.Hour,
	)

	_, err := svc.Refresh(context.Background(), "any-jti")
	assert.True(t, errors.Is(err, ErrPlatformAuthStorageUnavailable), "续期存储故障必须返回可映射为 503 的错误，实际 %v", err)
}

func mustExists(f *fakeredis.Server, key string) bool {
	ok, _ := f.Exists(key)
	return ok
}
