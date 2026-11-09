package repository

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

	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
)

var (
	billingTestDB   *gorm.DB
	billingTestOnce sync.Once
)

func openBillingRepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	billingTestOnce.Do(func() {
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
			&domain.Kindergarten{},
			&domain.PlatformAdmin{},
			&domain.Distributor{},
			&domain.Plan{}, &domain.PlanModule{}, &domain.PlanTier{},
			&domain.Order{}, &domain.Payment{},
			&domain.Entitlement{}, &domain.CommissionLedger{},
		); err != nil {
			return
		}
		billingTestDB = db
	})
	if billingTestDB == nil {
		t.Skip("跳过：本地无可用 DB 配置")
	}
	return billingTestDB
}

func billingHardDelete(t *testing.T, db *gorm.DB, sql string, args ...any) {
	t.Helper()
	if err := db.Exec(sql, args...).Error; err != nil {
		t.Logf("cleanup %q: %v", sql, err)
	}
}

func newTestKindergarten(name string) *domain.Kindergarten {
	return &domain.Kindergarten{
		Name:   name,
		Status: domain.KindergartenStatusActive,
	}
}

// 平台查询返回两个园，园所查询只返回本园——这是 design D9 的边界。
func TestBillingRepo_PlatformListVsTenantScope(t *testing.T) {
	db := openBillingRepoTestDB(t)
	kgRepo := NewKindergartenRepo(db)
	entRepo := NewEntitlementRepo(db)
	ctx := context.Background()

	prefix := "S1B商业化测试园"
	cleanup := func() {
		billingHardDelete(t, db, "DELETE FROM entitlements WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE ?)", prefix+"%")
		billingHardDelete(t, db, "DELETE FROM kindergartens WHERE name LIKE ?", prefix+"%")
	}
	cleanup()
	t.Cleanup(cleanup)

	kgA := newTestKindergarten(prefix + "A")
	kgB := newTestKindergarten(prefix + "B")
	require.NoError(t, kgRepo.Create(ctx, kgA))
	require.NoError(t, kgRepo.Create(ctx, kgB))

	list, total, err := kgRepo.List(ctx, 1, 100, prefix, "")
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	assert.Len(t, list, 2, "平台列表必须跨园返回")

	now := time.Now()
	require.NoError(t, entRepo.CreateBatch(ctx, []domain.Entitlement{
		{KindergartenID: kgA.ID, ModuleCode: "base", Source: domain.EntitlementSourceGift, StartsAt: now, ExpiresAt: now.AddDate(10, 0, 0)},
		{KindergartenID: kgA.ID, ModuleCode: "growth", Source: domain.EntitlementSourceTrial, StartsAt: now, ExpiresAt: now.AddDate(0, 1, 0)},
		{KindergartenID: kgB.ID, ModuleCode: "base", Source: domain.EntitlementSourceGift, StartsAt: now, ExpiresAt: now.AddDate(10, 0, 0)},
	}))

	rowsA, err := entRepo.ListByKindergarten(ctx, kgA.ID)
	require.NoError(t, err)
	assert.Len(t, rowsA, 2)
	for _, row := range rowsA {
		assert.Equal(t, kgA.ID, row.KindergartenID, "园所查询不得返回别园授权")
	}

	active, err := entRepo.ListActiveByKindergarten(ctx, kgA.ID, now.AddDate(0, 0, 1))
	require.NoError(t, err)
	assert.Len(t, active, 2)

	expired, err := entRepo.ListActiveByKindergarten(ctx, kgA.ID, now.AddDate(0, 2, 0))
	require.NoError(t, err)
	assert.Len(t, expired, 1, "超出到期日的授权不应再被视为在期")
}
