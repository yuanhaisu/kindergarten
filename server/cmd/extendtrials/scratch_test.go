package main

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	mysqldrv "github.com/go-sql-driver/mysql"
	"github.com/pressly/goose/v3"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"kindergarten-server/internal/config"
	migrations "kindergarten-server/migrations"
)

// scratchDBName 是测试专用库。测试只操作这一份 schema，绝不碰共享开发库：
// 迁移在库内真跑一遍（goose），断言的是命令在真实 schema 上的行为。
const scratchDBName = "kindergarten_extendtrials_test"

// openScratchDB 连接本机 MySQL，重建 scratch 库、跑完全部 goose 迁移并清空业务表。
//
// DSN 来源：
//  1. EXTENDTRIALS_TEST_DSN（显式指定，便于换机器/换端口）
//  2. configs/app.ini + MYSQL_DSN 环境变量覆盖（与仓库其它 DB 测试一致）
//
// 找不到配置时跳过；建库或迁移失败则测试失败——那属于测试环境坏了，不该静默放过。
func openScratchDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := testDSN(t)

	dsnCfg, err := mysqldrv.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("解析测试 DSN 失败: %v", err)
	}
	loc := mustShanghai(t)
	dsnCfg.Loc = loc
	dsnCfg.ParseTime = true
	dsnCfg.DBName = ""

	admin, err := sql.Open("mysql", dsnCfg.FormatDSN())
	if err != nil {
		t.Fatalf("连接 MySQL 失败: %v", err)
	}
	defer func() { _ = admin.Close() }()
	if _, err := admin.Exec("CREATE DATABASE IF NOT EXISTS `" + scratchDBName + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci"); err != nil {
		t.Fatalf("创建 scratch 库 %s 失败（账号需要 CREATE 权限）: %v", scratchDBName, err)
	}

	dsnCfg.DBName = scratchDBName
	scratchDSN := dsnCfg.FormatDSN()

	runMigrations(t, scratchDSN)

	db, err := gorm.Open(mysql.Open(scratchDSN), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("连接 scratch 库失败: %v", err)
	}
	resetScratchTables(t, db)
	return db
}

func testDSN(t *testing.T) string {
	t.Helper()
	if v := os.Getenv("EXTENDTRIALS_TEST_DSN"); v != "" {
		return v
	}
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "configs/app.ini"
	}
	if _, err := os.Stat(cfgPath); err != nil {
		t.Skipf("跳过：找不到 %s（可用 EXTENDTRIALS_TEST_DSN 指定测试库）", cfgPath)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("读取 %s 失败: %v", cfgPath, err)
	}
	if cfg.MySQL.DSN == "" {
		t.Skip("跳过：MySQL DSN 为空")
	}
	return cfg.MySQL.DSN
}

func runMigrations(t *testing.T, dsn string) {
	t.Helper()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("打开迁移连接失败: %v", err)
	}
	defer func() { _ = db.Close() }()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("mysql"); err != nil {
		t.Fatalf("设置 goose dialect 失败: %v", err)
	}
	if err := goose.UpContext(context.Background(), db, "."); err != nil {
		t.Fatalf("scratch 库迁移失败: %v", err)
	}
}

func resetScratchTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	stmts := []string{
		"SET FOREIGN_KEY_CHECKS=0",
		"TRUNCATE TABLE `entitlements`",
		"TRUNCATE TABLE `orders`",
		"TRUNCATE TABLE `kindergartens`",
		"SET FOREIGN_KEY_CHECKS=1",
	}
	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			t.Fatalf("清理 scratch 库失败 %q: %v", s, err)
		}
	}
}

func mustShanghai(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("加载 Asia/Shanghai 失败: %v", err)
	}
	return loc
}

// ---- 夹具：显式 ID，断言可读；所有时间都按 Asia/Shanghai 理解 ----

func insertKindergarten(t *testing.T, db *gorm.DB, id uint64, name string, createdAt *time.Time) {
	t.Helper()
	var err error
	if createdAt == nil {
		err = db.Exec(
			"INSERT INTO kindergartens (id, created_at, updated_at, deleted_at, name) VALUES (?, NULL, NOW(3), NULL, ?)",
			id, name,
		).Error
	} else {
		err = db.Exec(
			"INSERT INTO kindergartens (id, created_at, updated_at, deleted_at, name) VALUES (?, ?, ?, NULL, ?)",
			id, *createdAt, *createdAt, name,
		).Error
	}
	if err != nil {
		t.Fatalf("插入园所 %d 失败: %v", id, err)
	}
}

func insertSoftDeletedKindergarten(t *testing.T, db *gorm.DB, id uint64, name string, createdAt time.Time) {
	t.Helper()
	err := db.Exec(
		"INSERT INTO kindergartens (id, created_at, updated_at, deleted_at, name) VALUES (?, ?, ?, ?, ?)",
		id, createdAt, createdAt, createdAt, name,
	).Error
	if err != nil {
		t.Fatalf("插入软删园所 %d 失败: %v", id, err)
	}
}

func insertEntitlement(t *testing.T, db *gorm.DB, id, kgID uint64, module, source string, orderID *uint64, startsAt, expiresAt time.Time) {
	t.Helper()
	err := db.Exec(
		"INSERT INTO entitlements (id, created_at, updated_at, deleted_at, kindergarten_id, module_code, source, order_id, starts_at, expires_at) VALUES (?, NOW(3), NOW(3), NULL, ?, ?, ?, ?, ?, ?)",
		id, kgID, module, source, orderID, startsAt, expiresAt,
	).Error
	if err != nil {
		t.Fatalf("插入授权 %d 失败: %v", id, err)
	}
}

func insertOrder(t *testing.T, db *gorm.DB, id, kgID uint64, status string, expiresAt time.Time) {
	t.Helper()
	err := db.Exec(
		"INSERT INTO orders (id, created_at, updated_at, deleted_at, order_no, kindergarten_id, plan_id, plan_snapshot, status, starts_at, expires_at) VALUES (?, NOW(3), NOW(3), NULL, ?, ?, 1, '{}', ?, NOW(3), ?)",
		id, "TEST-"+status+"-"+time.Now().Format("150405.000000000"), kgID, status, expiresAt,
	).Error
	if err != nil {
		t.Fatalf("插入订单 %d 失败: %v", id, err)
	}
}

type expiryRow struct {
	ID        uint64
	ExpiresAt time.Time
}

func allEntitlementExpiries(t *testing.T, db *gorm.DB) []expiryRow {
	t.Helper()
	var rows []expiryRow
	if err := db.Raw("SELECT id, expires_at FROM entitlements ORDER BY id").Scan(&rows).Error; err != nil {
		t.Fatalf("读取授权到期日失败: %v", err)
	}
	return rows
}

func testOptions(loc *time.Location) Options {
	return Options{
		Now:             time.Date(2026, 11, 16, 12, 0, 0, 0, loc),
		CutoffExclusive: time.Date(2026, 11, 16, 0, 0, 0, 0, loc),
		TargetExpiry:    time.Date(2027, 1, 31, 23, 59, 59, 0, loc),
	}
}
