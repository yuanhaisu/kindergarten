package repository

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
)

func openOperationLogTestDB(t *testing.T) *gorm.DB {
	t.Helper()
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
	require.NoError(t, db.AutoMigrate(&domain.OperationLog{}), "auto migrate operation_logs")
	return db
}

func TestOperationLogRepo_CreateRedactsFreeTextPII(t *testing.T) {
	db := openOperationLogTestDB(t)
	repo := NewOperationLogRepo(db)
	row := &domain.OperationLog{
		KindergartenID: 1,
		UserID:         1,
		Module:         "test",
		Action:         "create",
		Detail:         `{"phone":"13800138000","id_card":"110101199001011234"}`,
	}
	require.NoError(t, repo.Create(context.Background(), row))
	t.Cleanup(func() {
		if err := db.Exec("DELETE FROM operation_logs WHERE id = ?", row.ID).Error; err != nil {
			t.Logf("cleanup operation_logs: %v", err)
		}
	})

	var got domain.OperationLog
	require.NoError(t, db.First(&got, row.ID).Error)
	require.NotContains(t, got.Detail, "13800138000")
	require.Contains(t, got.Detail, "138****8000")
	require.NotContains(t, got.Detail, "110101199001011234")
	require.Contains(t, got.Detail, "110101********1234")
}
