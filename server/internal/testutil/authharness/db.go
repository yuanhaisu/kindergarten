package authharness

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	// TestDSNEnv 显式指定一个**已存在的独立测试库**。设置后不再按分支派生库名，
	// 但仍然会执行迁移；指向开发库会被拒绝。
	TestDSNEnv = "KDG_TEST_MYSQL_DSN"
	configEnv  = "CONFIG_PATH"

	migrationLockTimeoutSec = 120
)

// dbSetup 是包级共享的测试库句柄：同一 go test 进程里的多个用例复用同一个
// 派生库，迁移只跑一次（GET_LOCK 保证跨包、跨进程也只跑一次）。
type dbSetup struct {
	cfg      *config.Config
	db       *gorm.DB
	repoRoot string

	devDatabase  string
	testDatabase string
	testDSN      string
	dsnFromEnv   bool

	skipReason string
	err        error
}

var (
	dbSetupOnce sync.Once
	dbSetupVal  *dbSetup
)

func sharedTestDB() *dbSetup {
	dbSetupOnce.Do(func() { dbSetupVal = initTestDB() })
	return dbSetupVal
}

func initTestDB() *dbSetup {
	s := &dbSetup{}

	repoRoot, err := findRepoRoot()
	if err != nil {
		s.skipReason = fmt.Sprintf("未找到 Go 模块根目录：%v", err)
		return s
	}
	s.repoRoot = repoRoot

	cfgPath := findConfigPath(repoRoot)
	if cfgPath != "" {
		cfg, err := config.Load(cfgPath)
		if err != nil {
			// 配置文件存在却读不了，说明环境本身坏了，必须显式失败而不是跳过。
			s.err = fmt.Errorf("加载 %s 失败：%w", cfgPath, err)
			return s
		}
		s.cfg = cfg
	}

	explicitDSN := os.Getenv(TestDSNEnv)
	if explicitDSN == "" && (s.cfg == nil || s.cfg.MySQL.DSN == "") {
		s.skipReason = fmt.Sprintf("未找到 %s/configs/app.ini 且未设置 %s；认证 HTTP 测试需要独立测试库",
			repoRoot, TestDSNEnv)
		return s
	}

	if s.cfg != nil && s.cfg.MySQL.DSN != "" {
		devCfg, err := mysqldrv.ParseDSN(s.cfg.MySQL.DSN)
		if err != nil {
			s.err = fmt.Errorf("解析开发库 DSN 失败：%w", err)
			return s
		}
		s.devDatabase = devCfg.DBName
	}

	if explicitDSN != "" {
		testCfg, err := mysqldrv.ParseDSN(explicitDSN)
		if err != nil {
			s.err = fmt.Errorf("解析 %s 失败：%w", TestDSNEnv, err)
			return s
		}
		if testCfg.DBName == "" {
			s.err = fmt.Errorf("%s 必须包含数据库名", TestDSNEnv)
			return s
		}
		if testCfg.DBName == s.devDatabase && s.devDatabase != "" {
			s.err = fmt.Errorf("%s 指向开发库 %q；测试会写入并硬删真实数据，必须换成独立测试库",
				TestDSNEnv, testCfg.DBName)
			return s
		}
		s.testDatabase = testCfg.DBName
		s.testDSN = explicitDSN
		s.dsnFromEnv = true
	} else {
		devDatabase, testDatabase, testDSN, err := deriveTestTarget(s.cfg.MySQL.DSN, repoRoot)
		if err != nil {
			s.err = err
			return s
		}
		s.devDatabase = devDatabase
		s.testDatabase = testDatabase
		s.testDSN = testDSN
	}

	if strings.Contains(s.testDatabase, "`") {
		s.err = fmt.Errorf("测试库名 %q 含反引号，无法安全拼接 CREATE DATABASE", s.testDatabase)
		return s
	}

	db, skipReason, err := openAndMigrateTestDB(s.testDSN, s.testDatabase)
	if skipReason != "" {
		s.skipReason = skipReason
		return s
	}
	if err != nil {
		s.err = err
		return s
	}
	s.db = db
	return s
}

// deriveTestTarget 从开发库 DSN 派生一个按分支隔离的测试库名。
// 按分支隔离是为了让多个 worktree / 分支并行跑测试时互不干扰。
func deriveTestTarget(devDSN, repoRoot string) (devDatabase, testDatabase, testDSN string, err error) {
	dsnCfg, err := mysqldrv.ParseDSN(devDSN)
	if err != nil {
		return "", "", "", fmt.Errorf("解析 MySQL DSN 失败：%w", err)
	}
	if dsnCfg.DBName == "" {
		return "", "", "", errors.New("MySQL DSN 未指定数据库名，无法派生测试库")
	}
	devDatabase = dsnCfg.DBName

	base := sanitizeDBName(devDatabase)
	const maxBaseLen = 40
	if len(base) > maxBaseLen {
		base = base[:maxBaseLen]
	}
	testDatabase = base + "_test_" + branchScopedSuffix(repoRoot)

	dsnCfg.DBName = testDatabase
	return devDatabase, testDatabase, dsnCfg.FormatDSN(), nil
}

func openAndMigrateTestDB(testDSN, testDatabase string) (db *gorm.DB, skipReason string, err error) {
	dsnCfg, err := mysqldrv.ParseDSN(testDSN)
	if err != nil {
		return nil, "", fmt.Errorf("解析测试库 DSN 失败：%w", err)
	}
	addr := dsnCfg.Addr

	adminCfg := *dsnCfg
	adminCfg.DBName = ""
	adminDB, err := sql.Open("mysql", adminCfg.FormatDSN())
	if err != nil {
		return nil, "", fmt.Errorf("打开 MySQL 管理连接失败：%w", err)
	}
	defer func() { _ = adminDB.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := adminDB.PingContext(ctx); err != nil {
		return nil, fmt.Sprintf("MySQL %s 不可达（%v）；认证 HTTP 测试需要独立测试库，请先启动本地 MySQL", addr, err), nil
	}

	stmt := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", testDatabase)
	if _, err := adminDB.ExecContext(ctx, stmt); err != nil {
		return nil, "", fmt.Errorf(
			"创建测试库 %s 失败（MySQL %s）：%w；请授予当前账号 CREATE 权限，或设置 %s 指向一个已存在的独立测试库",
			testDatabase, addr, err, TestDSNEnv)
	}

	db, err = gorm.Open(gormmysql.Open(testDSN), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		return nil, "", fmt.Errorf("连接测试库 %s 失败：%w", testDatabase, err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, "", fmt.Errorf("获取测试库连接池失败：%w", err)
	}

	if err := migrateTestDB(ctx, sqlDB, testDatabase); err != nil {
		return nil, "", err
	}
	if !db.Migrator().HasTable("users") {
		return nil, "", fmt.Errorf("测试库 %s 迁移后仍无 users 表，迁移未生效", testDatabase)
	}
	return db, "", nil
}

// goose 的 SetBaseFS / SetDialect 是包级全局状态，进程内并发调用会互相覆盖。
var gooseMu sync.Mutex

func migrateTestDB(ctx context.Context, sqlDB *sql.DB, testDatabase string) error {
	// GET_LOCK 是跨进程互斥：不同包的测试进程会同时启动，没有它就会并发跑
	// baseline 迁移，撞表/撞索引后互相报错。
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return fmt.Errorf("获取迁移锁连接失败：%w", err)
	}
	defer func() { _ = conn.Close() }()

	lockName := "authharness_" + shortHash(testDatabase)
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
		return fmt.Errorf("执行 goose 迁移失败（测试库 %s）：%w", testDatabase, err)
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
			return "", errors.New("从当前目录向上未找到 go.mod")
		}
		dir = parent
	}
}

func findConfigPath(repoRoot string) string {
	if p := os.Getenv(configEnv); p != "" {
		return p
	}
	for _, rel := range []string{"configs/app.ini", "configs/app.local.ini"} {
		p := filepath.Join(repoRoot, rel)
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p
		}
	}
	return ""
}

// branchScopedSuffix 用当前 git 分支名派生 8 位后缀。
// 分支名不可读时退化为固定后缀，仍然保证名字以 _test_ 标记。
func branchScopedSuffix(repoRoot string) string {
	ref := "unknown"
	if headPath, err := findGitHEAD(repoRoot); err == nil {
		if b, err := os.ReadFile(headPath); err == nil {
			v := strings.TrimSpace(string(b))
			v = strings.TrimPrefix(v, "ref: refs/heads/")
			if v != "" {
				ref = v
			}
		}
	}
	return shortHash(ref)
}

// findGitHEAD 兼容普通仓库（.git 目录）与 worktree（.git 文件指向 gitdir）。
func findGitHEAD(start string) (string, error) {
	dir := start
	for {
		gitPath := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitPath); err == nil {
			if info.IsDir() {
				return filepath.Join(gitPath, "HEAD"), nil
			}
			b, err := os.ReadFile(gitPath)
			if err != nil {
				return "", err
			}
			line := strings.TrimSpace(string(b))
			if constPrefix := "gitdir:"; strings.HasPrefix(line, constPrefix) {
				gitdir := strings.TrimSpace(strings.TrimPrefix(line, constPrefix))
				if !filepath.IsAbs(gitdir) {
					gitdir = filepath.Join(dir, gitdir)
				}
				return filepath.Join(gitdir, "HEAD"), nil
			}
			return "", fmt.Errorf("无法解析 %s", gitPath)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("未找到 .git")
		}
		dir = parent
	}
}

func shortHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])[:8]
}

func sanitizeDBName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "kindergarten"
	}
	return b.String()
}
