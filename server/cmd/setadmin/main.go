// setadmin — 供给园所租户管理员账号：幂等地创建/更新用户与 admin 角色。
//
// 用法:
//
//	SETADMIN_PASSWORD='...' go run ./cmd/setadmin -phone 13800000000 -kg 7
//	go run ./cmd/setadmin -phone 13800000000 -kg 7 -password '...'
//
// -phone 与 -kg 必填；口令只从 -password 或环境变量 SETADMIN_PASSWORD 读取
// （推荐环境变量，避免进入 shell 历史与进程参数）。没有默认口令，也不会回显。
// 目标园所必须存在，软删除视为不存在。重复执行是幂等的。
//
// CONFIG_PATH 可覆盖配置文件路径（默认 configs/app.ini）；MYSQL_DSN 可覆盖连接串。
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	gormlogger "gorm.io/gorm/logger"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"kindergarten-server/cmd/internal/adminprovision"
	"kindergarten-server/internal/config"
)

func main() {
	phone := flag.String("phone", "", "管理员手机号（必填）")
	kgID := flag.Uint64("kg", 0, "目标园所 ID（必填，必须已存在）")
	password := flag.String("password", "", "管理员口令；建议改用环境变量 SETADMIN_PASSWORD")
	flag.Usage = func() {
		out := flag.CommandLine.Output()
		_, _ = fmt.Fprint(out, "setadmin — 供给园所租户管理员账号（创建/更新用户与 admin 角色）\n\n"+
			"用法: setadmin -phone <手机号> -kg <园所ID> [-password <口令>]\n"+
			"口令也可由环境变量 SETADMIN_PASSWORD 提供（推荐）。\n\n参数:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *phone == "" || *kgID == 0 {
		fmt.Fprintln(os.Stderr, "setadmin: -phone 与 -kg 均为必填")
		flag.Usage()
		os.Exit(2)
	}
	pwd := *password
	if pwd == "" {
		pwd = os.Getenv("SETADMIN_PASSWORD")
	}
	if pwd == "" {
		fmt.Fprintln(os.Stderr, "setadmin: 缺少口令：用 -password 或环境变量 SETADMIN_PASSWORD 提供")
		flag.Usage()
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

	res, err := adminprovision.EnsureTenantAdmin(context.Background(), db, adminprovision.Options{
		Phone:          *phone,
		Password:       pwd,
		KindergartenID: *kgID,
	})
	if err != nil {
		log.Fatal("setadmin: ", err)
	}
	fmt.Println(res.Summary())
}
