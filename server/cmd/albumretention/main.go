// albumretention — 软删相册 180 天留存清理（运维辅助命令，design D8）。
//
//	go run ./cmd/albumretention -dry-run         只统计，不删除
//	go run ./cmd/albumretention -batch-size 100  每批硬删 100 本相册
//
// 只硬删软删除已满 180 天的相册及其照片、相册级/照片级幼儿关联；
// operation_logs 审计记录保留。不新增常驻 worker，调度交给运维定时执行。
//
// 配置读 configs/app.ini（可用 CONFIG_PATH 覆盖）。输出不含 DSN/账号，
// 出错时也只打印抹掉 DSN 的错误文本。
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	gormlogger "gorm.io/gorm/logger"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"kindergarten-server/internal/config"
	"kindergarten-server/internal/service"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "只统计到期待清理的行数，不删除")
	batchSize := flag.Int("batch-size", 100, "每批硬删的相册数")
	flag.Parse()

	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "configs/app.ini"
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatal("load config: ", err)
	}
	if cfg.MySQL.DSN == "" {
		log.Fatal("MySQL DSN 为空：请检查 configs/app.ini 或设置 MYSQL_DSN")
	}

	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		// 连接错误里可能带整条 DSN（含口令），必须抹掉后再打。
		log.Fatal("open db: ", sanitizeDSN(err, cfg.MySQL.DSN))
	}

	svc := service.NewAlbumRetentionService(db)
	ctx := context.Background()
	now := time.Now()
	cutoff := now.Add(-service.AlbumSoftDeleteRetention).Format("2006-01-02 15:04:05")

	if *dryRun {
		start := time.Now()
		stats, err := svc.CountExpiredAlbums(ctx, now)
		if err != nil {
			log.Fatal("统计到期数据失败: ", sanitizeDSN(err, cfg.MySQL.DSN))
		}
		fmt.Printf("dry-run：截止 %s 前的软删相册 %d 本、照片 %d 张、照片级幼儿关联 %d 条、相册级参与者 %d 条；未删除任何数据（耗时 %s）\n",
			cutoff, stats.Albums, stats.Photos, stats.PhotoStudents, stats.AlbumStudents,
			time.Since(start).Round(time.Millisecond))
		return
	}

	start := time.Now()
	deleted, err := svc.CleanupExpiredAlbums(ctx, now, *batchSize)
	elapsed := time.Since(start).Round(time.Millisecond)
	if err != nil {
		log.Fatalf("清理中断（已删除 %d 本相册，耗时 %s）: %s", deleted, elapsed, sanitizeDSN(err, cfg.MySQL.DSN))
	}
	fmt.Printf("清理完成：截止 %s 前的软删相册已清理 %d 本（batch-size=%d，耗时 %s）\n",
		cutoff, deleted, *batchSize, elapsed)
}

// sanitizeDSN 把错误文本里可能出现的 DSN 替换掉：连接串含口令，不能进日志。
func sanitizeDSN(err error, dsn string) string {
	msg := err.Error()
	if dsn != "" {
		msg = strings.ReplaceAll(msg, dsn, "<dsn>")
	}
	return msg
}
