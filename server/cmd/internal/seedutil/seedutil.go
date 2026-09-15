// Package seedutil 收纳 cmd/seed* 系列命令共用的启动样板：
// 读配置、连库、解析目标园所、加载业务时区。
//
// 放在 cmd/internal/ 下，因此只有 cmd/ 的子命令能导入。
package seedutil

import (
	"log"
	"os"
	"strconv"
	"time"

	gormlogger "gorm.io/gorm/logger"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"kindergarten-server/internal/config"
)

// DefaultKindergartenID 是这批一次性脚本惯用的目标园所。
const DefaultKindergartenID uint64 = 4

// Open 读取 configs/app.ini 并建立数据库连接。失败即退出。
func Open() *gorm.DB {
	cfg, err := config.Load("configs/app.ini")
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
