// Package seedutil 收纳 cmd/seed* 系列命令共用的启动样板：
// 读配置、连库、解析目标园所、加载业务时区。
//
// 放在 cmd/internal/ 下，因此只有 cmd/ 的子命令能导入。
package seedutil

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"time"

	gormlogger "gorm.io/gorm/logger"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"kindergarten-server/internal/config"
)

// DefaultKindergartenID 是这批一次性脚本惯用的目标园所。
const DefaultKindergartenID uint64 = 4

// Open 读取配置并建立数据库连接。失败即退出。
// 配置路径优先取 CONFIG_PATH，未设置时回落 configs/app.ini。
func Open() *gorm.DB {
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "configs/app.ini"
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatal("load config:", err)
	}
	if cfg.MySQL.DSN == "" {
		log.Fatal("MySQL DSN 为空：请检查 configs/app.ini 或设置 MYSQL_DSN")
	}
	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		log.Fatal("open:", err)
	}
	return db
}

// KindergartenID 返回目标园所：取环境变量 SEED_KG_ID，未设置时用 DefaultKindergartenID。
func KindergartenID() uint64 {
	v := os.Getenv("SEED_KG_ID")
	if v == "" {
		return DefaultKindergartenID
	}
	n, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		log.Fatalf("SEED_KG_ID 不是合法数字: %q", v)
	}
	return n
}

// Shanghai 加载业务时区。
//
// 刻意不静默退回 time.Local：驱动按 cfg.Loc 渲染时间，时区不符会让写入的
// date 列整体偏移一天（见 CLAUDE.md「Time / timezone」一节）。
func Shanghai() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		log.Fatal("加载 Asia/Shanghai 失败（容器缺 tzdata？）:", err)
	}
	return loc
}

const (
	envAllowDemo        = "SEED_ALLOW_DEMO"
	envAllowDestructive = "SEED_ALLOW_DESTRUCTIVE"
)

// RequireDemoOptIn 是演示数据命令的护栏：没有显式 opt-in 直接退出。
//
// 这些命令会往目标库写入编造数据；开发时习惯性地 `go run ./cmd/seed*`，
// 到了生产环境就是误写的源头。报错里带脱敏后的 host:port/db，便于核对目标。
func RequireDemoOptIn(command string) {
	if err := demoOptInError(command, targetSummary()); err != nil {
		log.Fatal(err)
	}
}

// RequireDestructiveOptIn 在演示开关之上再要求破坏性开关。
// 适用于会先删除再重建数据的命令（initcheck / refreshtasks）。
func RequireDestructiveOptIn(command string) {
	if err := destructiveOptInError(command, targetSummary()); err != nil {
		log.Fatal(err)
	}
}

func demoOptInError(command, target string) error {
	if os.Getenv(envAllowDemo) == "1" {
		return nil
	}
	return fmt.Errorf("%s 会写入演示数据；确认目标库后设置 %s=1 再运行（目标：%s）",
		command, envAllowDemo, target)
}

func destructiveOptInError(command, target string) error {
	if os.Getenv(envAllowDestructive) != "1" {
		return fmt.Errorf("%s 会删除并重建数据；确认目标库后设置 %s=1 再运行（目标：%s）",
			command, envAllowDestructive, target)
	}
	if os.Getenv(envAllowDemo) != "1" {
		return fmt.Errorf("%s 同时会写入演示数据；再设置 %s=1 后重试（目标：%s）",
			command, envAllowDemo, target)
	}
	return nil
}

var dsnTargetPattern = regexp.MustCompile(`@tcp\(([^)]+)\)/([^?]+)`)

// targetSummary 返回脱敏后的目标库 host:port/db。
// 配置读不到时只返回占位描述，不影响护栏本身的判断。
func targetSummary() string {
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "configs/app.ini"
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return "未知（读取 " + cfgPath + " 失败）"
	}
	if cfg.MySQL.DSN == "" {
		return "未知（DSN 为空）"
	}
	return dsnTarget(cfg.MySQL.DSN)
}

// dsnTarget 只保留 host:port/db，丢掉账号与口令。
func dsnTarget(dsn string) string {
	if m := dsnTargetPattern.FindStringSubmatch(dsn); m != nil {
		return m[1] + "/" + m[2]
	}
	return "无法解析 DSN（仅支持 tcp 形式）"
}
