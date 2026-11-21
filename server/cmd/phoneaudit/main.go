package main

// phoneaudit 只读排查「同一活跃手机号挂多个账号」。
//
// migration 00013 在建 users.phone_active 唯一索引前会用存储过程做同样的检查，
// 发现重复就 SIGNAL 中止（迁移失败，不会半途改表）。本命令用于迁移前先看清楚
// 要人工处理哪些账号：
//
//	go run ./cmd/phoneaudit             # 只读报告；有重复时退出码 1
//	go run ./cmd/phoneaudit -limit 50   # 每个重复组最多列出 50 个账号
//
// 绝不删除或修改任何数据：合并/软删哪些账号是业务决定，必须由人确认后手动执行。
// 输出只给脱敏后的手机号与账号 id，避免把手机号写进终端日志或工单。
//
// 配置读 configs/app.ini（CONFIG_PATH 覆盖路径，MYSQL_DSN 覆盖连接串）。

import (
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
	"kindergarten-server/pkg/redact"
)

type duplicateGroup struct {
	Phone string `gorm:"column:phone"`
	Total int64  `gorm:"column:cnt"`
}

type accountRow struct {
	ID        uint64    `gorm:"column:id"`
	Nickname  string    `gorm:"column:nickname"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func main() {
	limit := flag.Int("limit", 20, "每个重复组最多列出的账号数")
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
		// 连接错误可能带整条 DSN（含口令），只报错不回显 DSN。
		log.Fatal("open db 失败（DSN 已隐藏）")
	}

	var groups []duplicateGroup
	if err := db.Raw(`
		SELECT phone, COUNT(*) AS cnt
		FROM users
		WHERE deleted_at IS NULL AND phone IS NOT NULL AND phone <> ''
		GROUP BY phone
		HAVING cnt > 1
		ORDER BY cnt DESC, phone
	`).Scan(&groups).Error; err != nil {
		log.Fatal("查询重复手机号失败: ", err)
	}

	if len(groups) == 0 {
		fmt.Println("未发现活跃账号重复手机号：migration 00013 的唯一索引可以建立。")
		printIndexStatus(db)
		return
	}

	var affected int64
	for _, g := range groups {
		affected += g.Total
	}
	fmt.Printf("发现 %d 个手机号被多个活跃账号占用，涉及 %d 个账号：\n\n", len(groups), affected)

	for _, g := range groups {
		fmt.Printf("手机号 %s（%d 个账号）：\n", redact.Phone(g.Phone), g.Total)
		var rows []accountRow
		if err := db.Raw(`
			SELECT id, nickname, created_at
			FROM users
			WHERE deleted_at IS NULL AND phone = ?
			ORDER BY id
			LIMIT ?
		`, g.Phone, *limit).Scan(&rows).Error; err != nil {
			log.Fatalf("查询账号失败（手机号 %s）: %v", redact.Phone(g.Phone), err)
		}
		for _, r := range rows {
			fmt.Printf("  - id=%d 昵称=%s 创建于=%s\n",
				r.ID, maskName(r.Nickname), r.CreatedAt.Format("2006-01-02 15:04:05"))
		}
		if g.Total > int64(len(rows)) {
			fmt.Printf("  ... 另有 %d 个账号未列出（-limit 调大）\n", g.Total-int64(len(rows)))
		}
		fmt.Println()
	}

	fmt.Println("处理建议：确认哪个账号是要保留的主账号，把其余账号的业务关系迁移过去后软删；")
	fmt.Println("清理完成前 migration 00013 会拒绝执行，本命令不会自动删除任何数据。")
	printIndexStatus(db)
	os.Exit(1)
}

// printIndexStatus 说明唯一索引是否已生效——已生效时上面的重复列表理论上不可能非空，
// 出现即说明索引被绕过（例如手工 DROP），需要人工核查。
func printIndexStatus(db *gorm.DB) {
	var exists int64
	if err := db.Raw(`
		SELECT COUNT(*)
		FROM information_schema.statistics
		WHERE table_schema = DATABASE()
		  AND table_name = 'users'
		  AND index_name = 'uk_users_phone_active'
	`).Scan(&exists).Error; err != nil {
		fmt.Println("（无法确认 uk_users_phone_active 索引状态：", err, "）")
		return
	}
	if exists > 0 {
		fmt.Println("索引状态：uk_users_phone_active 已存在。")
	} else {
		fmt.Println("索引状态：uk_users_phone_active 尚未建立（migration 00013 未执行）。")
	}
}

// maskName 昵称也属于个人信息，报告里同样脱敏：保留首字，其余打码。
func maskName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "(空)"
	}
	runes := []rune(s)
	if len(runes) == 1 {
		return string(runes)
	}
	return string(runes[0]) + strings.Repeat("*", len(runes)-1)
}
