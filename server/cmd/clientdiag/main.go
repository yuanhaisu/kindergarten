// clientdiag — 平台客户端只读诊断：打印小程序客户端注册表的状态/作用域/版本。
//
// 平台支持用它判断「某一端为什么登录不了」：app_id 是否注册、是否 enabled、
// 作用域是否 shared、当前发版版本是多少。命令只读，不提供停用/启用开关；
// status 的变更走数据库/运维操作（#163）。
//
//	go run ./cmd/clientdiag          # 人类可读表格
//	go run ./cmd/clientdiag -json    # JSON（供脚本/工单消费）
//
// CONFIG_PATH 可覆盖配置文件路径（默认 configs/app.ini）；MYSQL_DSN 可覆盖连接串。
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"kindergarten-server/cmd/internal/clientdiag"
	"kindergarten-server/internal/config"
	"kindergarten-server/internal/repository"
)

// 编译期断言：真实仓储满足只读接口。诊断命令在类型层面拿不到写方法。
var _ clientdiag.ClientLister = (*repository.ClientRepo)(nil)

func main() {
	asJSON := flag.Bool("json", false, "以 JSON 输出（供脚本/工单消费）")
	flag.Parse()

	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "configs/app.ini"
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	if cfg.MySQL.DSN == "" {
		fmt.Fprintf(os.Stderr, "MySQL DSN 为空：请检查 %s 或设置 MYSQL_DSN\n", cfgPath)
		os.Exit(1)
	}

	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect db: %v\n", err)
		os.Exit(1)
	}

	rows, err := clientdiag.Collect(context.Background(), repository.NewClientRepo(db))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	if *asJSON {
		err = clientdiag.WriteJSON(os.Stdout, rows)
	} else {
		err = clientdiag.WriteTable(os.Stdout, rows)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "输出失败: %v\n", err)
		os.Exit(1)
	}
}
