package main

// migrate 执行 goose 版本化迁移。
//
//	go run ./cmd/migrate up
//	go run ./cmd/migrate status
//	go run ./cmd/migrate -database kindergarten_migrate_test up
//
// 生产部署时由一次性迁移容器执行；应用进程启动不再建表。

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"os"

	mysqldrv "github.com/go-sql-driver/mysql"
	"github.com/pressly/goose/v3"

	"kindergarten-server/internal/config"
	migrations "kindergarten-server/migrations"
)

func main() {
	database := flag.String("database", "", "覆盖 DSN 中的数据库名（用于本地验证）")
	flag.Parse()

	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "configs/app.ini"
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatal("load config:", err)
	}
	if cfg.MySQL.DSN == "" {
		log.Fatal("MySQL DSN 为空")
	}

	dsn := cfg.MySQL.DSN
	if *database != "" {
		dsnCfg, parseErr := mysqldrv.ParseDSN(dsn)
		if parseErr != nil {
			log.Fatalf("parse DSN: %v", parseErr)
		}
		dsnCfg.DBName = *database
		dsn = dsnCfg.FormatDSN()
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("open:", err)
	}
	defer func() { _ = db.Close() }()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("mysql"); err != nil {
		log.Fatal("set dialect:", err)
	}

	args := flag.Args()
	command := "up"
	if len(args) > 0 {
		command = args[0]
		args = args[1:]
	}
	if err := goose.RunContext(context.Background(), command, db, ".", args...); err != nil {
		log.Fatalf("goose %s: %v", command, err)
	}
	log.Printf("goose %s completed", command)
}
