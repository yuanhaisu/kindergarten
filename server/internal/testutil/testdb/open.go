package testdb

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	mysqldrv "github.com/go-sql-driver/mysql"
	"github.com/pressly/goose/v3"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"kindergarten-server/internal/config"
	migrations "kindergarten-server/migrations"
)

const (
	// configEnv 与 authharness / cmd 保持一致：显式指定配置文件路径。
	configEnv = "CONFIG_PATH"
	// migrationLockTimeoutSec 是跨进程迁移锁的等待上限。
	migrationLockTimeoutSec = 120
)

// Info 描述 Open 实际连接的库。Target 是脱敏后的 host:port/db，可直接打印。
type Info struct {
	DB          *gorm.DB
	Database    string
	Target      string
	DevDatabase string
	// FromEnv 为 true 表示库名由 KDG_TEST_MYSQL_DSN 派生；false 表示回退到配置里的开发库。
	FromEnv bool
}

// Open 返回当前测试进程可用的数据库句柄。
//
//   - 设置了 KDG_TEST_MYSQL_DSN：校验后派生「分支 + 测试包」隔离的测试库并执行
//     goose 迁移；不会连接开发库。
//   - 未设置：保持现有本地行为，直接连接配置里的开发库；配置缺失时跳过测试。
//   - 设置 KDG_TEST_REQUIRE_MYSQL 后，跳过改为失败（CI 用，缺库不得静默变绿）。
//
// 本函数不会删除任何数据库；调用方必须在 Cleanup 里硬删自己写入的行。
func Open(t testing.TB) *gorm.DB {
	t.Helper()
	return OpenInfo(t).DB
}

// OpenInfo 与 Open 相同，但额外返回库名与脱敏目标，供断言与诊断使用。
func OpenInfo(t testing.TB) Info {
	t.Helper()
	s := sharedSetup()
	if s.skipReason != "" {
		if RequireMySQL() {
			t.Fatalf("testdb: %s（%s 已设置，缺库必须失败而不是跳过）", s.skipReason, RequireDBEnv)
		}
		t.Skipf("testdb: %s", s.skipReason)
	}
	if s.err != nil {
		t.Fatalf("testdb: %v", s.err)
	}
	return Info{
		DB:          s.db,
		Database:    s.database,
		Target:      s.target,
		DevDatabase: s.devDatabase,
		FromEnv:     s.fromEnv,
	}
}

// setup 是包级共享状态：同一测试进程里的多个用例复用同一个库，迁移只跑一次。
type setup struct {
	db          *gorm.DB
	database    string
	target      string
	devDatabase string
	fromEnv     bool

	skipReason string
	err        error
}

var (
	setupOnce sync.Once
	setupVal  *setup
)

func sharedSetup() *setup {
	setupOnce.Do(func() { setupVal = initSetup() })
	return setupVal
}

func initSetup() *setup {
	s := &setup{}

	repoRoot, err := findRepoRoot()
	if err != nil {
		s.err = fmt.Errorf("未找到 Go 模块根目录：%w", err)
		return s
	}

	explicit := os.Getenv(TestDSNEnv)
	devDSN, configFound, err := loadDevDSN(repoRoot)
	if err != nil {
		s.err = err
		return s
	}

	if explicit == "" {
		if !configFound || devDSN == "" {
			reason := fmt.Sprintf("未设置 %s，且未找到可用的 configs/app.ini；DB 测试需要其中之一", TestDSNEnv)
			if RequireMySQL() {
				// CI 要求必须真连库：缺库时 Fatal，避免整个包静默跳过。
				s.err = fmt.Errorf("%s（%s 已设置，缺库必须失败而不是跳过）", reason, RequireDBEnv)
				return s
			}
			s.skipReason = reason
			return s
		}
		cfg, err := mysqldrv.ParseDSN(devDSN)
		if err != nil {
			s.err = fmt.Errorf("解析开发库 DSN 失败（目标：%s）", Target(devDSN))
			return s
		}
		db, err := openGORM(devDSN)
		if err != nil {
			s.err = fmt.Errorf("连接开发库失败（目标：%s）：%w", Target(devDSN), err)
			return s
		}
		s.db = db
		s.database = cfg.DBName
		s.devDatabase = cfg.DBName
		s.target = Target(devDSN)
		return s
	}

	if err := Guard(explicit, devDSN); err != nil {
		s.err = err
		return s
	}
	cfg, err := mysqldrv.ParseDSN(explicit)
	if err != nil {
		// Guard 已经解析成功过；这里只是兜底，且不回显原始解析错误。
		s.err = fmt.Errorf("%s 无法解析，拒绝运行（目标：%s）", TestDSNEnv, Target(explicit))
		return s
	}
	if devDSN != "" {
		if devCfg, err := mysqldrv.ParseDSN(devDSN); err == nil {
			s.devDatabase = devCfg.DBName
		}
	}

	database := deriveDatabaseName(cfg.DBName, currentBranch(repoRoot), packageScope())
	derivedCfg := *cfg
	derivedCfg.DBName = database
	dsn := derivedCfg.FormatDSN()

	db, err := openAndMigrate(dsn, database)
	if err != nil {
		s.err = err
		return s
	}
	s.db = db
	s.database = database
	s.target = Target(dsn)
	s.fromEnv = true
	return s
}

// loadDevDSN 读取开发库 DSN：CONFIG_PATH 优先，否则按仓库根目录找
// configs/app.ini、app.local.ini。找不到配置文件不算错误（found=false）。
//
// 加载错误刻意不包装 ini 解析器的原文：解析错误可能回显 `dsn = user:pass@...`
// 这一行，把凭据带进测试日志。
func loadDevDSN(repoRoot string) (dsn string, found bool, err error) {
	if p := os.Getenv(configEnv); p != "" {
		cfg, err := config.Load(p)
		if err != nil {
			return "", true, fmt.Errorf("加载配置文件 %s 失败（为避免回显敏感行，未展开原始错误）", p)
		}
		return cfg.MySQL.DSN, true, nil
	}
	for _, rel := range []string{"configs/app.ini", "configs/app.local.ini"} {
		p := filepath.Join(repoRoot, rel)
		info, statErr := os.Stat(p)
		if statErr != nil || info.IsDir() {
			continue
		}
		cfg, err := config.Load(p)
		if err != nil {
			return "", true, fmt.Errorf("加载配置文件 %s 失败（为避免回显敏感行，未展开原始错误）", p)
		}
		return cfg.MySQL.DSN, true, nil
	}
	return "", false, nil
}

func openGORM(dsn string) (*gorm.DB, error) {
	return gorm.Open(gormmysql.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
}

func openAndMigrate(testDSN, database string) (*gorm.DB, error) {
	target := Target(testDSN)
	if database == "" || strings.Contains(database, "`") {
		return nil, fmt.Errorf("派生测试库名 %q 非法，无法安全拼接 CREATE DATABASE", database)
	}

	cfg, err := mysqldrv.ParseDSN(testDSN)
	if err != nil {
		return nil, fmt.Errorf("解析派生测试库 DSN 失败（目标：%s）", target)
	}
	adminCfg := *cfg
	adminCfg.DBName = ""
	adminDB, err := sql.Open("mysql", adminCfg.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("打开 MySQL 管理连接失败（目标：%s）：%w", target, err)
	}
	defer func() { _ = adminDB.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := adminDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("MySQL 不可达（目标：%s）：%w；请启动测试 MySQL，或检查 %s",
			target, err, TestDSNEnv)
	}

	stmt := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", database)
	if _, err := adminDB.ExecContext(ctx, stmt); err != nil {
		return nil, fmt.Errorf("创建测试库失败（目标：%s）：%w；请授予当前账号 CREATE 权限", target, err)
	}

	db, err := openGORM(testDSN)
	if err != nil {
		return nil, fmt.Errorf("连接测试库失败（目标：%s）：%w", target, err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取测试库连接池失败（目标：%s）：%w", target, err)
	}
	if err := migrateDB(ctx, sqlDB, database); err != nil {
		return nil, err
	}
	if !db.Migrator().HasTable("users") {
		return nil, fmt.Errorf("测试库 %s 迁移后仍无 users 表，迁移未生效（目标：%s）", database, target)
	}
	return db, nil
}

// goose 的 SetBaseFS / SetDialect 是包级全局状态，进程内并发调用会互相覆盖。
var gooseMu sync.Mutex

func migrateDB(ctx context.Context, sqlDB *sql.DB, database string) error {
	// GET_LOCK 是跨进程互斥：同一分支的同一包被并行跑两次时，
	// 没有它就会并发执行 baseline 迁移并撞表/撞索引。
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return fmt.Errorf("获取迁移锁连接失败：%w", err)
	}
	defer func() { _ = conn.Close() }()

	lockName := "testdb_" + shortHash(database)
	var acquired sql.NullInt64
	if err := conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, ?)", lockName, migrationLockTimeoutSec).Scan(&acquired); err != nil {
		return fmt.Errorf("申请迁移锁 %s 失败：%w", lockName, err)
	}
	if !acquired.Valid || acquired.Int64 != 1 {
		return fmt.Errorf("等待迁移锁 %s 超过 %d 秒", lockName, migrationLockTimeoutSec)
	}
	defer func() {
		_, _ = conn.ExecContext(context.Background(), "SELECT RELEASE_LOCK(?)", lockName)
	}()

	gooseMu.Lock()
	defer gooseMu.Unlock()

	goose.SetLogger(goose.NopLogger())
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("mysql"); err != nil {
		return fmt.Errorf("设置 goose dialect 失败：%w", err)
	}
	if err := goose.UpContext(ctx, sqlDB, "."); err != nil {
		return fmt.Errorf("执行 goose 迁移失败（测试库 %s）：%w", database, err)
	}
	return nil
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("从当前目录向上未找到 go.mod")
		}
		dir = parent
	}
}
