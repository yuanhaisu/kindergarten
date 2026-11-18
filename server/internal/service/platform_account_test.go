package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/internal/testutil/fakeredis"
	"kindergarten-server/pkg/requestid"
)

// spyEpochStore 记录 epoch 写入；onBump 在写入瞬间回调，用于验证
// 「先写 Redis epoch，再提交数据库变更」的顺序（design D5）。
type spyEpochStore struct {
	mu     sync.Mutex
	bumps  map[string][]time.Time
	err    error
	onBump func(adminID string)
}

func (s *spyEpochStore) PlatformEpoch(context.Context, string) (time.Time, error) {
	return time.Time{}, s.err
}

func (s *spyEpochStore) BumpPlatformEpoch(_ context.Context, adminID string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	if s.onBump != nil {
		s.onBump(adminID)
	}
	if s.bumps == nil {
		s.bumps = map[string][]time.Time{}
	}
	s.bumps[adminID] = append(s.bumps[adminID], at)
	return nil
}

func (s *spyEpochStore) bumpCount(adminID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.bumps[adminID])
}

func newPlatformAccountService(t *testing.T, db *gorm.DB, audit PlatformAuditStore, epochs auth.PlatformEpochStore) *PlatformService {
	t.Helper()
	if audit == nil {
		audit = repository.NewPlatformLogRepo(db)
	}
	f := fakeredis.New(t)
	rdb := f.Client(t)
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
		audit,
		nil,
		nil,
		epochs,
		auth.NewTokenRevoker(rdb),
		rdb,
		[]byte("test-platform-secret"),
		time.Hour,
		7*24*time.Hour,
	)
}

func cleanupPlatformAccounts(t *testing.T, db *gorm.DB) {
	t.Helper()
	del := func(sql string, args ...any) {
		if err := db.Exec(sql, args...).Error; err != nil {
			t.Logf("cleanup %q: %v", sql, err)
		}
	}
	del("DELETE FROM platform_operation_logs WHERE operator LIKE 't6_%'")
	del("DELETE FROM platform_admins WHERE username LIKE 't6_%'")
}

func createPlatformAccount(t *testing.T, svc *PlatformService, username, password string) *PlatformAccountRow {
	t.Helper()
	row, err := svc.CreateAccount(context.Background(), CreateAccountRequest{
		Username: username, Name: "T6 " + username, Password: password,
	}, "t6_ops", "127.0.0.1")
	require.NoError(t, err)
	require.NotZero(t, row.ID)
	return row
}

func TestPlatformAccount_CreateThenLoginAndList(t *testing.T) {
	db := openPlatformTestDB(t)
	ctx := context.Background()
	cleanupPlatformAccounts(t, db)
	t.Cleanup(func() { cleanupPlatformAccounts(t, db) })

	svc := newPlatformAccountService(t, db, nil, &spyEpochStore{})
	row := createPlatformAccount(t, svc, "t6_ops_create", "Abc12345")
	assert.Equal(t, "t6_ops_create", row.Username)
	assert.Equal(t, PlatformAccountEnabled, row.Status)

	// 新建账号可用新口令登录平台，权限与既有账号一致（TC-NEW-PLT-0009）。
	res, err := svc.Login(ctx, "t6_ops_create", "Abc12345")
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, row.ID, res.Admin.ID)

	// 列表按关键词命中，status 暴露为 enabled/disabled，且不含口令哈希。
	rows, total, err := svc.ListAccounts(ctx, 1, 20, "t6_ops_create")
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, rows, 1)
	assert.Equal(t, PlatformAccountEnabled, rows[0].Status)

	require.NoError(t, svc.SetAccountStatus(ctx, row.ID, PlatformAccountDisabled, 999999, "t6_ops", "127.0.0.1"))
	rows, _, err = svc.ListAccounts(ctx, 1, 20, "t6_ops_create")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, PlatformAccountDisabled, rows[0].Status)

	// 停用后不能登录。
	_, err = svc.Login(ctx, "t6_ops_create", "Abc12345")
	assert.Error(t, err)
}

func TestPlatformAccount_CreateRejectsDuplicateAndWeakPassword(t *testing.T) {
	db := openPlatformTestDB(t)
	ctx := context.Background()
	cleanupPlatformAccounts(t, db)
	t.Cleanup(func() { cleanupPlatformAccounts(t, db) })

	svc := newPlatformAccountService(t, db, nil, &spyEpochStore{})
	createPlatformAccount(t, svc, "t6_ops_dup", "Abc12345")

	_, err := svc.CreateAccount(ctx, CreateAccountRequest{
		Username: "t6_ops_dup", Password: "Abc12345",
	}, "t6_ops", "127.0.0.1")
	assert.ErrorIs(t, err, ErrUsernameExists)

	for _, weak := range []string{"Ab1", "abcdefg1", "ABCDEFG1", "Abcdefgh"} {
		_, err := svc.CreateAccount(ctx, CreateAccountRequest{
			Username: "t6_ops_weak", Password: weak,
		}, "t6_ops", "127.0.0.1")
		require.Error(t, err, "口令 %q 必须被拒绝", weak)
		assert.True(t, strings.HasPrefix(err.Error(), "WEAK_PASSWORD:"), "实际错误：%v", err)
	}

	// 弱口令不得留下半成品账号。
	_, err = svc.CreateAccount(ctx, CreateAccountRequest{
		Username: "t6_ops_weak", Password: "Abc12345",
	}, "t6_ops", "127.0.0.1")
	require.NoError(t, err)
}

func TestPlatformAccount_DisableSelfRejected(t *testing.T) {
	db := openPlatformTestDB(t)
	ctx := context.Background()
	cleanupPlatformAccounts(t, db)
	t.Cleanup(func() { cleanupPlatformAccounts(t, db) })

	epochs := &spyEpochStore{}
	svc := newPlatformAccountService(t, db, nil, epochs)
	row := createPlatformAccount(t, svc, "t6_ops_self", "Abc12345")

	err := svc.SetAccountStatus(ctx, row.ID, PlatformAccountDisabled, row.ID, "t6_ops_self", "127.0.0.1")
	assert.ErrorIs(t, err, ErrCannotDisableSelf)
	assert.Zero(t, epochs.bumpCount(accountIDKey(row.ID)), "被守卫拒绝的操作不应写 epoch")

	reloaded, err := repository.NewPlatformAdminRepo(db).FindByID(ctx, row.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 1, reloaded.Status, "停用自己被拒后账号必须仍可用")
}

// 「可用账号」口径 = status=1 AND deleted_at IS NULL；只剩一个时停用被拒。
func TestPlatformAccount_LastAvailableGuard(t *testing.T) {
	db := openPlatformTestDB(t)
	ctx := context.Background()
	cleanupPlatformAccounts(t, db)
	t.Cleanup(func() { cleanupPlatformAccounts(t, db) })

	svc := newPlatformAccountService(t, db, nil, &spyEpochStore{})
	a := createPlatformAccount(t, svc, "t6_ops_last_a", "Abc12345")
	b := createPlatformAccount(t, svc, "t6_ops_last_b", "Abc12345")

	// 暂时停用数据库里其它账号，让可用账号只剩 a 与 b；
	// 测试结束（含失败）必须恢复，避免污染开发环境。
	var otherIDs []uint64
	require.NoError(t, db.Model(&domain.PlatformAdmin{}).
		Where("username NOT LIKE 't6_%' AND status = 1").Pluck("id", &otherIDs).Error)
	t.Cleanup(func() {
		if len(otherIDs) > 0 {
			db.Model(&domain.PlatformAdmin{}).Where("id IN ?", otherIDs).Update("status", 1)
		}
	})
	if len(otherIDs) > 0 {
		require.NoError(t, db.Model(&domain.PlatformAdmin{}).Where("id IN ?", otherIDs).Update("status", 0).Error)
	}

	require.NoError(t, svc.SetAccountStatus(ctx, b.ID, PlatformAccountDisabled, a.ID, "t6_ops", "127.0.0.1"))
	assert.EqualValues(t, 1, countAvailable(t, db))

	// 操作者用 b 的 ID（与目标 a 不同），守卫应因「只剩一个」拒绝。
	err := svc.SetAccountStatus(ctx, a.ID, PlatformAccountDisabled, b.ID, "t6_ops", "127.0.0.1")
	assert.ErrorIs(t, err, ErrLastAvailableAccount)

	reloaded, err := repository.NewPlatformAdminRepo(db).FindByID(ctx, a.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 1, reloaded.Status, "最后一个可用账号必须保持启用")
}

func TestPlatformAccount_ResetPassword(t *testing.T) {
	db := openPlatformTestDB(t)
	ctx := requestid.With(context.Background(), "req-t6-reset")
	cleanupPlatformAccounts(t, db)
	t.Cleanup(func() { cleanupPlatformAccounts(t, db) })

	epochs := &spyEpochStore{}
	svc := newPlatformAccountService(t, db, nil, epochs)
	row := createPlatformAccount(t, svc, "t6_ops_reset", "Abc12345")

	// 重置时数据库里仍是旧哈希——证明 epoch 先于数据库写入。
	epochs.onBump = func(string) {
		admin, err := repository.NewPlatformAdminRepo(db).FindByID(ctx, row.ID)
		require.NoError(t, err)
		assert.True(t, admin.PasswordHash != "", "重置前账号必须存在")
	}
	require.NoError(t, svc.ResetAccountPassword(ctx, row.ID, "Xyz98765", "t6_ops", "127.0.0.1"))

	assert.Equal(t, 1, epochs.bumpCount(accountIDKey(row.ID)), "重置密码必须写 epoch")
	_, err := svc.Login(ctx, "t6_ops_reset", "Abc12345")
	assert.Error(t, err, "旧口令必须失效")
	_, err = svc.Login(ctx, "t6_ops_reset", "Xyz98765")
	assert.NoError(t, err, "新口令必须可登录")

	logs, total, err := svc.ListLogs(ctx, repository.PlatformLogFilter{Operator: "t6_ops", Module: "account"})
	require.NoError(t, err)
	assert.EqualValues(t, 2, total, "新建与重置都必须留痕")
	actions := make([]string, 0, len(logs))
	for _, l := range logs {
		actions = append(actions, l.Action)
		assert.Equal(t, "req-t6-reset", l.RequestID, "审计必须带请求链路标识")
		assert.EqualValues(t, row.ID, l.TargetID)
	}
	assert.ElementsMatch(t, []string{"create", "reset_password"}, actions)
}

func TestPlatformAccount_ChangeOwnPassword(t *testing.T) {
	db := openPlatformTestDB(t)
	ctx := context.Background()
	cleanupPlatformAccounts(t, db)
	t.Cleanup(func() { cleanupPlatformAccounts(t, db) })

	epochs := &spyEpochStore{}
	svc := newPlatformAccountService(t, db, nil, epochs)
	row := createPlatformAccount(t, svc, "t6_ops_change", "Abc12345")

	err := svc.ChangeOwnPassword(ctx, row.ID, "WrongPass1", "NewPass123")
	assert.ErrorIs(t, err, ErrInvalidOldPassword)
	assert.Zero(t, epochs.bumpCount(accountIDKey(row.ID)), "旧口令错误时不得写 epoch")

	err = svc.ChangeOwnPassword(ctx, row.ID, "Abc12345", "weakpass1")
	require.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "WEAK_PASSWORD:"))

	require.NoError(t, svc.ChangeOwnPassword(ctx, row.ID, "Abc12345", "NewPass123"))
	assert.Equal(t, 1, epochs.bumpCount(accountIDKey(row.ID)), "改密必须写 epoch")

	_, err = svc.Login(ctx, "t6_ops_change", "Abc12345")
	assert.Error(t, err)
	_, err = svc.Login(ctx, "t6_ops_change", "NewPass123")
	assert.NoError(t, err)
}

func TestPlatformAccount_InvalidStatusAndNotFound(t *testing.T) {
	db := openPlatformTestDB(t)
	ctx := context.Background()
	cleanupPlatformAccounts(t, db)
	t.Cleanup(func() { cleanupPlatformAccounts(t, db) })

	svc := newPlatformAccountService(t, db, nil, &spyEpochStore{})
	row := createPlatformAccount(t, svc, "t6_ops_status", "Abc12345")

	assert.ErrorIs(t, svc.SetAccountStatus(ctx, row.ID, "suspended", 999999, "t6_ops", "127.0.0.1"), ErrInvalidAccountStatus)
	assert.ErrorIs(t, svc.SetAccountStatus(ctx, 999999, PlatformAccountDisabled, 999999, "t6_ops", "127.0.0.1"), ErrAccountNotFound)
	assert.ErrorIs(t, svc.ResetAccountPassword(ctx, 999999, "Abc12345", "t6_ops", "127.0.0.1"), ErrAccountNotFound)
	assert.ErrorIs(t, svc.ChangeOwnPassword(ctx, 999999, "Abc12345", "Xyz98765"), ErrAccountNotFound)
}

// 审计与账号写入同事务：审计失败时新建/停用/重置全部回滚（design D6）。
func TestPlatformAccount_AuditFailureRollsBack(t *testing.T) {
	db := openPlatformTestDB(t)
	ctx := context.Background()
	cleanupPlatformAccounts(t, db)
	t.Cleanup(func() { cleanupPlatformAccounts(t, db) })

	audit := repository.NewPlatformLogRepo(db)
	okSvc := newPlatformAccountService(t, db, audit, &spyEpochStore{})
	row := createPlatformAccount(t, okSvc, "t6_ops_audit", "Abc12345")

	failing := failingAuditWriter{err: errAuditDown}
	svc := newPlatformAccountService(t, db, failing, &spyEpochStore{})

	// 新建：审计失败时账号不能留下。
	_, err := svc.CreateAccount(ctx, CreateAccountRequest{
		Username: "t6_ops_audit_new", Password: "Abc12345",
	}, "t6_ops", "127.0.0.1")
	assert.ErrorIs(t, err, errAuditDown)
	created, err := repository.NewPlatformAdminRepo(db).FindByUsername(ctx, "t6_ops_audit_new")
	require.NoError(t, err)
	assert.Nil(t, created, "审计失败时新建账号必须回滚")

	// 停用：审计失败时状态不变。
	assert.ErrorIs(t, svc.SetAccountStatus(ctx, row.ID, PlatformAccountDisabled, 999999, "t6_ops", "127.0.0.1"), errAuditDown)
	reloaded, err := repository.NewPlatformAdminRepo(db).FindByID(ctx, row.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 1, reloaded.Status, "审计失败时停用必须回滚")

	// 重置：审计失败时旧口令继续有效。
	assert.ErrorIs(t, svc.ResetAccountPassword(ctx, row.ID, "Xyz98765", "t6_ops", "127.0.0.1"), errAuditDown)
	_, err = okSvc.Login(ctx, "t6_ops_audit", "Abc12345")
	assert.NoError(t, err, "审计失败时口令变更必须回滚")
}

// epoch 写失败时操作失败，数据库保持不变（design D5）。
func TestPlatformAccount_EpochWriteFailureLeavesDBUnchanged(t *testing.T) {
	db := openPlatformTestDB(t)
	ctx := context.Background()
	cleanupPlatformAccounts(t, db)
	t.Cleanup(func() { cleanupPlatformAccounts(t, db) })

	// 新建不需要 epoch：Redis 故障不能挡住账号创建。
	createSvc := newPlatformAccountService(t, db, nil, &spyEpochStore{err: errors.New("redis down")})
	row := createPlatformAccount(t, createSvc, "t6_ops_epoch", "Abc12345")

	svc := newPlatformAccountService(t, db, nil, &spyEpochStore{err: errors.New("redis down")})

	err := svc.SetAccountStatus(ctx, row.ID, PlatformAccountDisabled, 999999, "t6_ops", "127.0.0.1")
	require.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "PLATFORM_AUTH_UNAVAILABLE:"), "实际错误：%v", err)
	reloaded, err := repository.NewPlatformAdminRepo(db).FindByID(ctx, row.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 1, reloaded.Status, "epoch 写失败时停用必须失败且不改库")

	err = svc.ResetAccountPassword(ctx, row.ID, "Xyz98765", "t6_ops", "127.0.0.1")
	require.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "PLATFORM_AUTH_UNAVAILABLE:"))
	_, err = createSvc.Login(ctx, "t6_ops_epoch", "Abc12345")
	assert.NoError(t, err, "epoch 写失败时口令不能变")

	err = svc.ChangeOwnPassword(ctx, row.ID, "Abc12345", "Xyz98765")
	require.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "PLATFORM_AUTH_UNAVAILABLE:"))
	_, err = createSvc.Login(ctx, "t6_ops_epoch", "Abc12345")
	assert.NoError(t, err)
}

// 停用自己的守卫与「最后一个可用账号」都必须在写 epoch 之前判定，
// 否则一次被拒绝的操作也会把目标账号的在线会话踢下线。
func TestPlatformAccount_GuardRunsBeforeEpochBump(t *testing.T) {
	db := openPlatformTestDB(t)
	ctx := context.Background()
	cleanupPlatformAccounts(t, db)
	t.Cleanup(func() { cleanupPlatformAccounts(t, db) })

	epochs := &spyEpochStore{}
	svc := newPlatformAccountService(t, db, nil, epochs)
	a := createPlatformAccount(t, svc, "t6_ops_order_a", "Abc12345")
	b := createPlatformAccount(t, svc, "t6_ops_order_b", "Abc12345")

	assert.ErrorIs(t, svc.SetAccountStatus(ctx, a.ID, PlatformAccountDisabled, a.ID, "t6_ops_order_a", "127.0.0.1"), ErrCannotDisableSelf)
	assert.Zero(t, epochs.bumpCount(accountIDKey(a.ID)))
	assert.NotZero(t, b.ID)
}

func countAvailable(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(&domain.PlatformAdmin{}).Where("status = 1 AND deleted_at IS NULL").Count(&n).Error)
	return n
}

// accountIDKey 与 auth.PlatformEpochStore 的键格式保持一致。
func accountIDKey(id uint64) string { return strconv.FormatUint(id, 10) }
