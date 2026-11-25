// setadmin — 供给园所租户管理员账号：幂等地创建/更新用户与 admin 角色。
//
// 用法:
//
//	SETADMIN_PASSWORD='...' go run ./cmd/setadmin -phone 13800000000 -kg 7
//	go run ./cmd/setadmin -phone 13800000000 -kg 7 -password '...'
//
// -phone 与 -kg 必填；口令只从 -password 或环境变量 SETADMIN_PASSWORD 读取
// （推荐环境变量，避免进入 shell 历史与进程参数）。没有默认口令，也不会回显。
// 目标园所必须存在，软删除视为不存在。重复执行是幂等的：复用既有账号会
// 覆盖口令并推进园所账号 epoch，使此前的 access/refresh 立即失效。
//
// 因此本命令需要可用的 Redis（与 API 服务同一实例）。会话 Redis 不可用时
// 直接失败，而不是跳过失效——否则「口令已换、旧会话仍有效」会留下静默缺口。
//
// CONFIG_PATH 可覆盖配置文件路径（默认 configs/app.ini）；MYSQL_DSN / REDIS_ADDR
// 可覆盖连接配置。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	gormlogger "gorm.io/gorm/logger"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"kindergarten-server/cmd/internal/adminprovision"
	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/config"
)

// parseOptions 解析并校验命令行参数，口令依次取 -password 与 SETADMIN_PASSWORD。
// 纯函数：不读配置、不连库，缺参分支无需 DB 即可覆盖。
func parseOptions(args []string, getenv func(string) string) (adminprovision.Options, error) {
	fs := flag.NewFlagSet("setadmin", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // 错误由调用方统一打印，避免 flag 重复输出
	phone := fs.String("phone", "", "管理员手机号（必填）")
	kgID := fs.Uint64("kg", 0, "目标园所 ID（必填，必须已存在）")
	password := fs.String("password", "", "管理员口令；建议改用环境变量 SETADMIN_PASSWORD")
	if err := fs.Parse(args); err != nil {
		return adminprovision.Options{}, err
	}
	if *phone == "" {
		return adminprovision.Options{}, errors.New("-phone 必填")
	}
	if *kgID == 0 {
		return adminprovision.Options{}, errors.New("-kg 必填（目标园所 ID，必须大于 0）")
	}
	pwd := *password
	if pwd == "" {
		pwd = getenv("SETADMIN_PASSWORD")
	}
	if pwd == "" {
		return adminprovision.Options{}, errors.New("缺少口令：用 -password 或环境变量 SETADMIN_PASSWORD 提供")
	}
	return adminprovision.Options{Phone: *phone, Password: pwd, KindergartenID: *kgID}, nil
}

// usage 手写参数说明：parseOptions 用独立 FlagSet，不能再借 flag.PrintDefaults。
func usage() {
	_, _ = fmt.Fprint(os.Stderr, "setadmin — 供给园所租户管理员账号（创建/更新用户与 admin 角色）\n\n"+
		"用法: setadmin -phone <手机号> -kg <园所ID> [-password <口令>]\n"+
		"口令也可由环境变量 SETADMIN_PASSWORD 提供（推荐）。\n\n参数:\n"+
		"  -phone string\n    \t管理员手机号（必填）\n"+
		"  -kg uint\n    \t目标园所 ID（必填，必须已存在）\n"+
		"  -password string\n    \t管理员口令；建议改用环境变量 SETADMIN_PASSWORD\n")
}

func main() {
	opts, err := parseOptions(os.Args[1:], os.Getenv)
	if err != nil {
		fmt.Fprintln(os.Stderr, "setadmin: ", err)
		usage()
		os.Exit(2)
	}

	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "configs/app.ini"
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatal("load config: ", err)
	}
	if cfg.MySQL.DSN == "" {
		log.Fatal("MySQL DSN 为空：请检查配置或设置 MYSQL_DSN")
	}
	if cfg.Redis.Addr == "" {
		log.Fatal("Redis 地址为空：复用既有账号需要写入会话失效记录，请检查配置或设置 REDIS_ADDR")
	}
	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{
		// 首次供给时按手机号/角色查询会命中 record-not-found，属正常分支；
		// 不忽略会把「命令成功」刷成一片红色错误日志，误导运维。
		Logger: gormlogger.New(log.New(os.Stderr, "", log.LstdFlags), gormlogger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  gormlogger.Warn,
			IgnoreRecordNotFoundError: true,
		}),
	})
	if err != nil {
		log.Fatal("open db: ", err)
	}

	res, err := adminprovision.EnsureTenantAdmin(context.Background(), db, opts, newTenantEpochStore(cfg))
	if err != nil {
		log.Fatal("setadmin: ", err)
	}
	fmt.Println(res.Summary())
}

// newTenantEpochStore 按与 API 服务相同的公式构造园所账号失效存储。
// TTL 覆盖 refresh + access + 1 天；默认值（15 分钟 / 7 天）与
// internal/app 的 Provide*TTL 保持一致，配置缺省时不会把 epoch 写短。
func newTenantEpochStore(cfg *config.Config) auth.TenantEpochStore {
	atTTL := time.Duration(cfg.JWT.AccessTTLMin) * time.Minute
	if atTTL <= 0 {
		atTTL = 15 * time.Minute
	}
	rtTTL := time.Duration(cfg.JWT.RefreshTTLDay) * 24 * time.Hour
	if rtTTL <= 0 {
		rtTTL = 7 * 24 * time.Hour
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	return auth.NewTenantEpochStore(rdb, auth.PlatformEpochTTLFor(rtTTL, atTTL))
}
