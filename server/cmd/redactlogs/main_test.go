package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"kindergarten-server/internal/config"
)

func openRedactLogsTestDB(t *testing.T) *gorm.DB {
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
	return db
}

func TestRun_DryRunThenApplyIsIdempotent(t *testing.T) {
	db := openRedactLogsTestDB(t)
	const table = "redactlogs_test"

	require.NoError(t, db.Exec("DROP TABLE IF EXISTS "+table).Error)
	require.NoError(t, db.Exec("CREATE TABLE "+table+" LIKE operation_logs").Error)
	t.Cleanup(func() {
		if err := db.Exec("DROP TABLE IF EXISTS " + table).Error; err != nil {
			t.Logf("cleanup %s: %v", table, err)
		}
	})

	require.NoError(t, db.Exec(
		"INSERT INTO "+table+" (kindergarten_id, user_id, detail) VALUES (1, 1, ?)",
		`{"phone":"13800138000"}`,
	).Error)
	require.NoError(t, db.Exec(
		"INSERT INTO "+table+" (kindergarten_id, user_id, detail) VALUES (1, 2, ?)",
		`{"note":"no pii"}`,
	).Error)

	scanned, changed, err := run(db, table, options{DryRun: true, Batch: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), scanned)
	require.Equal(t, int64(1), changed)
	require.Equal(t, `{"phone":"13800138000"}`, detailOf(t, db, table, 1), "dry-run 不得写库")

	scanned, changed, err = run(db, table, options{DryRun: false, Batch: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), scanned)
	require.Equal(t, int64(1), changed)
	require.Equal(t, `{"phone":"138****8000"}`, detailOf(t, db, table, 1))

	scanned, changed, err = run(db, table, options{DryRun: false, Batch: 10})
	require.NoError(t, err)
	require.Equal(t, int64(0), scanned)
	require.Equal(t, int64(0), changed)
}

func detailOf(t *testing.T, db *gorm.DB, table string, userID uint64) string {
	t.Helper()
	var details []string
	require.NoError(t, db.Table(table).Where("user_id = ?", userID).Pluck("detail", &details).Error)
	require.Len(t, details, 1)
	return details[0]
}
