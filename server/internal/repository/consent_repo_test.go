package repository

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
)

var consentRepoTestDB *gorm.DB

func openConsentRepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	if consentRepoTestDB != nil {
		return consentRepoTestDB
	}
	cfgPath := filepath.Join("..", "..", "configs", "app.ini")
	if _, err := os.Stat(cfgPath); err != nil {
		t.Skipf("跳过：找不到 %s（本地无 DB 配置）", cfgPath)
	}
	cfg, err := config.Load(cfgPath)
	require.NoError(t, err, "load config")
	if cfg.MySQL.DSN == "" {
		t.Skip("跳过：MySQL DSN 为空")
	}
	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err, "open db")
	require.NoError(t, db.AutoMigrate(&domain.UserConsent{}), "auto migrate user_consents")
	consentRepoTestDB = db
	return db
}

func TestConsentRepo_CreateIfAbsentIsIdempotent(t *testing.T) {
	db := openConsentRepoTestDB(t)
	repo := NewConsentRepo(db)
	ctx := context.Background()
	userID := uint64(time.Now().UnixNano())
	t.Cleanup(func() {
		if err := db.Exec("DELETE FROM user_consents WHERE user_id = ?", userID).Error; err != nil {
			t.Logf("cleanup user_consents: %v", err)
		}
	})

	first := &domain.UserConsent{
		UserID:        userID,
		PolicyVersion: "v-test",
		Source:        domain.ConsentSourceParent,
		IP:            "127.0.0.1",
		UserAgent:     "consent-repo-test",
		ConsentedAt:   time.Now().Add(-time.Minute),
	}
	require.NoError(t, repo.CreateIfAbsent(ctx, first))

	got, err := repo.FindByUserAndVersion(ctx, userID, "v-test")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, first.ConsentedAt.Unix(), got.ConsentedAt.Unix())

	second := &domain.UserConsent{
		UserID:        userID,
		PolicyVersion: "v-test",
		Source:        domain.ConsentSourceParent,
		IP:            "10.0.0.1",
		UserAgent:     "second-attempt",
		ConsentedAt:   time.Now(),
	}
	require.NoError(t, repo.CreateIfAbsent(ctx, second))

	gotAgain, err := repo.FindByUserAndVersion(ctx, userID, "v-test")
	require.NoError(t, err)
	require.NotNil(t, gotAgain)
	require.Equal(t, got.ID, gotAgain.ID)
	require.WithinDuration(t, got.ConsentedAt, gotAgain.ConsentedAt, time.Second)

	var count int64
	require.NoError(t, db.Model(&domain.UserConsent{}).Where("user_id = ?", userID).Count(&count).Error)
	require.Equal(t, int64(1), count)
}

func TestConsentRepo_VersionsAreIndependent(t *testing.T) {
	db := openConsentRepoTestDB(t)
	repo := NewConsentRepo(db)
	ctx := context.Background()
	userID := uint64(time.Now().UnixNano()) + 1
	t.Cleanup(func() {
		if err := db.Exec("DELETE FROM user_consents WHERE user_id = ?", userID).Error; err != nil {
			t.Logf("cleanup user_consents: %v", err)
		}
	})

	for _, version := range []string{"v1", "v2"} {
		require.NoError(t, repo.CreateIfAbsent(ctx, &domain.UserConsent{
			UserID:        userID,
			PolicyVersion: version,
			Source:        domain.ConsentSourceTeacher,
			ConsentedAt:   time.Now(),
		}))
	}

	rows, err := repo.ListByUser(ctx, userID)
	require.NoError(t, err)
	require.Len(t, rows, 2)
}
