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
	"testing"
	"time"

	mysqldrv "github.com/go-sql-driver/mysql"
	"github.com/pressly/goose/v3"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"kindergarten-server/internal/config"
	"kindergarten-server/internal/testutil/testdb"
	migrations "kindergarten-server/migrations"
)

const (
	// TestDSNEnv 显式指定一个**已存在的独立测试库**。设置后不再按分支派生库名，
	// 但仍然会执行迁移；指向开发库或库名不含 test 会被拒绝（护栏与 testdb 共用）。
	TestDSNEnv = testdb.TestDSNEnv
	// RequireDBEnv 设为真值时，测试库不可用直接失败而不是跳过。CI 必须显式
	// 设置：缺 MySQL 或漏配 DSN 时让门禁变红，而不是把「没跑」当成「通过」。
	RequireDBEnv = testdb.RequireDBEnv
	configEnv    = "CONFIG_PATH"

	migrationLockTimeoutSec = 120
)

// requireMySQL 与 testdb 共用同一判定：值写错时按「要求」处理，
// 宁可失败关闭，也不能因为拼错开关而静默跳过。
func requireMySQL() bool {
	return testdb.RequireMySQL()
}

// failOrSkip 处理「测试库不可用」：默认跳过，让没有 MySQL 的本地环境仍能跑
// 其余测试；RequireDBEnv 生效时一律 Fatal。
func (s *dbSetup) failOrSkip(t testing.TB) {
	t.Helper()
	if s.skipReason == "" {
		return
	}
	if requireMySQL() {
		t.Fatalf("authharness: %s（%s 已设置，缺库必须失败而不是跳过）", s.skipReason, RequireDBEnv)
	}
	t.Skipf("authharness: %s", s.skipReason)
}

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
			// 不展开解析器原文：它可能回显 `dsn = user:pass@...` 这一行。
			s.err = fmt.Errorf("加载 %s 失败（为避免回显敏感行，未展开原始错误）", cfgPath)
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
			s.err = fmt.Errorf("解析开发库 DSN 失败（目标：%s），拒绝运行", testdb.Target(s.cfg.MySQL.DSN))
			return s
		}
		s.devDatabase = devCfg.DBName
	}

	if explicitDSN != "" {
		devDSN := ""
		if s.cfg != nil {
			devDSN = s.cfg.MySQL.DSN
		}
		// 与 testdb.Open 共用同一护栏：tcp DSN、必须有库名、库名须含 test，
		// 且不得与开发库同名。错误只含脱敏目标，不回显账号口令。
		if err := testdb.Guard(explicitDSN, devDSN); err != nil {
			s.err = err
			return s
		}
		testCfg, err := mysqldrv.ParseDSN(explicitDSN)
		if err != nil {
			// Guard 已经解析成功过；这里只是兜底，且不回显原始解析错误。
			s.err = fmt.Errorf("%s 无法解析，拒绝运行（目标：%s）", TestDSNEnv, testdb.Target(explicitDSN))
			return s
		}
		s.testDatabase = testCfg.DBName
		s.testDSN = explicitDSN
		s.dsnFromEnv = true
	} else {
		devDatabase, testDatabase, testDSN, err := deriveTestTarget(s.cfg.MySQL.DSN, repoRoot, testPackageScope())
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

// deriveTestTarget 从开发库 DSN 派生一个按「分支 + 测试包」隔离的测试库名。
//
// 分支隔离让多个 worktree / 分支并行跑测试时互不干扰；测试包隔离是因为
// go test ./... 会并行运行不同包的测试进程，而有些用例会整表清空
// （如 service 包的 client_service_test 硬删 clients），与 authharness 的
// 登录用例共用同一个派生库时会偶发 403 CLIENT_UNAVAILABLE。
func deriveTestTarget(devDSN, repoRoot, testScope string) (devDatabase, testDatabase, testDSN string, err error) {
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
	testDatabase = base + "_test_" + branchScopedSuffix(repoRoot, testScope)

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

// branchScopedSuffix 用「当前 git 分支名 + 测试包作用域」派生 8 位后缀。
// 分支名不可读时退化为固定分支部分，仍然保证名字以 _test_ 标记。
func branchScopedSuffix(repoRoot, testScope string) string {
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
	return shortHash(ref + "|" + testScope)
}

// testPackageScope 返回当前测试二进制对应的包名。go test 每个包编译出独立的
// <pkg>.test[.exe]，因此同一进程内恒定、不同测试包必然不同，可用来把整表清空
// 类用例（client_service_test）与 authharness 的会话用例隔到不同测试库。
func testPackageScope() string {
	name := filepath.Base(os.Args[0])
	name = strings.TrimSuffix(name, ".exe")
	name = strings.TrimSuffix(name, ".test")
	return sanitizeDBName(name)
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
