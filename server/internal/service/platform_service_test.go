package service

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/internal/testutil/fakeredis"
)

var (
	platformTestDB   *gorm.DB
	platformTestOnce sync.Once
)

func openPlatformTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	platformTestOnce.Do(func() {
		cfgPath := filepath.Join("..", "..", "configs", "app.ini")
		if _, err := os.Stat(cfgPath); err != nil {
			return
		}
		cfg, err := config.Load(cfgPath)
		if err != nil || cfg.MySQL.DSN == "" {
			return
		}
		db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{
			Logger: gormlogger.Default.LogMode(gormlogger.Silent),
		})
		if err != nil {
			return
		}
		if err := db.AutoMigrate(
			&domain.Kindergarten{}, &domain.User{}, &domain.UserRole{},
			&domain.AdminRole{}, &domain.AdminPermission{}, &domain.AdminRolePermission{},
			&domain.UserAdminRole{}, &domain.PlatformAdmin{}, &domain.PlatformOperationLog{},
			&domain.Entitlement{}, &domain.Order{}, &domain.Payment{},
			&domain.Plan{}, &domain.PlanModule{}, &domain.PlanTier{},
			&domain.Distributor{}, &domain.CommissionLedger{},
		); err != nil {
			return
		}
		platformTestDB = db
	})
	if platformTestDB == nil {
		t.Skip("跳过：本地无可用 DB 配置")
	}
	return platformTestDB
}

func newPlatformTestService(t *testing.T, db *gorm.DB) *PlatformService {
	t.Helper()
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
		repository.NewPlatformLogRepo(db),
		nil,
		nil,
		auth.NewPlatformEpochStore(rdb),
		auth.NewTokenRevoker(rdb),
		rdb,
		[]byte("test-platform-secret"),
		time.Hour,
		7*24*time.Hour,
	)
}

func cleanupPlatformTestData(t *testing.T, db *gorm.DB, phone string) {
	t.Helper()
	del := func(sql string, args ...any) {
		if err := db.Exec(sql, args...).Error; err != nil {
			t.Logf("cleanup %q: %v", sql, err)
		}
	}
	del("DELETE FROM entitlements WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE 'S1B平台测试%')")
	del("DELETE FROM platform_operation_logs WHERE operator = 's1b_ops'")
	del("DELETE FROM user_admin_roles WHERE user_id IN (SELECT id FROM users WHERE phone = ?)", phone)
	del("DELETE FROM admin_role_permissions WHERE role_id IN (SELECT id FROM admin_roles WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE 'S1B平台测试%'))")
	del("DELETE FROM admin_roles WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE 'S1B平台测试%')")
	del("DELETE FROM user_roles WHERE user_id IN (SELECT id FROM users WHERE phone = ?)", phone)
	del("DELETE FROM users WHERE phone = ?", phone)
	del("DELETE FROM kindergartens WHERE name LIKE 'S1B平台测试%'")
	del("DELETE FROM platform_admins WHERE username = 's1b_ops'")
}

func TestPlatformService_LoginAndOnboard(t *testing.T) {
	db := openPlatformTestDB(t)
	svc := newPlatformTestService(t, db)
	ctx := context.Background()
	phone := "19900001111"
	cleanupPlatformTestData(t, db, phone)
	t.Cleanup(func() { cleanupPlatformTestData(t, db, phone) })

	hash, err := HashPassword("OpsPassword1")
	require.NoError(t, err)
	require.NoError(t, repository.NewPlatformAdminRepo(db).Create(ctx, &domain.PlatformAdmin{
		Username: "s1b_ops", PasswordHash: hash, Name: "运营", Status: 1,
	}))

	res, err := svc.Login(ctx, "s1b_ops", "OpsPassword1")
	require.NoError(t, err)
	assert.NotEmpty(t, res.Token)
	assert.Equal(t, "s1b_ops", res.Admin.Username)

	_, err = svc.Login(ctx, "s1b_ops", "wrong-password")
	assert.Error(t, err, "错误密码必须被拒绝")

	onboarded, err := svc.Onboard(ctx, OnboardRequest{
		Name: "S1B平台测试园", Principal: "王园长", Phone: phone, AdminPassword: "InitPass1",
	}, "s1b_ops", "127.0.0.1")
	require.NoError(t, err)
	assert.NotZero(t, onboarded.KindergartenID)
	assert.True(t, onboarded.MustChangePassword)

	ents, err := repository.NewEntitlementRepo(db).ListByKindergarten(ctx, onboarded.KindergartenID)
	require.NoError(t, err)
	assert.Len(t, ents, 1+len(PremiumModules()), "开通后应有 base + 每个 premium 模块一条试用授权")

	_, err = svc.Onboard(ctx, OnboardRequest{
		Name: "S1B平台测试园2", Principal: "李园长", Phone: phone, AdminPassword: "InitPass1",
	}, "s1b_ops", "127.0.0.1")
	assert.ErrorIs(t, err, ErrPhoneExists, "同手机号必须被拒绝")

	require.NoError(t, svc.SetKindergartenStatus(ctx, onboarded.KindergartenID, domain.KindergartenStatusSuspended, "s1b_ops", "127.0.0.1"))
	kg, err := repository.NewKindergartenRepo(db).FindByID(ctx, onboarded.KindergartenID)
	require.NoError(t, err)
	assert.Equal(t, domain.KindergartenStatusSuspended, kg.Status)

	logs, total, err := svc.ListLogs(ctx, repository.PlatformLogFilter{Module: "kindergarten"})
	require.NoError(t, err)
	assert.NotZero(t, total)
	assert.NotEmpty(t, logs)
}
