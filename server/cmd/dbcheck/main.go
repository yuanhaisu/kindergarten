package main

// dbcheck 只读诊断：打印指定表的索引定义与行数。
// 用于核对迁移是否生效（例如 task_check_ins 的唯一索引）。
//
//	go run ./cmd/dbcheck task_check_ins
//	go run ./cmd/dbcheck --query "SELECT id, phone FROM users LIMIT 10"
//
// --query 模式只执行只读 SQL（SELECT/SHOW/DESCRIBE），用于验收时定位测试数据。

import (
	"fmt"
	"log"
	"os"
	"strings"

	gormlogger "gorm.io/gorm/logger"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"kindergarten-server/internal/config"
)

type indexRow struct {
	KeyName    string `gorm:"column:Key_name"`
	SeqInIndex int    `gorm:"column:Seq_in_index"`
	ColumnName string `gorm:"column:Column_name"`
	NonUnique  int    `gorm:"column:Non_unique"`
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("用法: go run ./cmd/dbcheck <表名>")
	}
	table := os.Args[1]

	cfg, err := config.Load("configs/app.ini")
	if err != nil {
		log.Fatal("load config:", err)
	}
	if cfg.MySQL.DSN == "" {
		log.Fatal("MySQL DSN 为空")
	}

	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		log.Fatal("open:", err)
	}

	if table == "--query" {
		if len(os.Args) < 3 {
			log.Fatal("用法: go run ./cmd/dbcheck --query \"<SELECT ...>\"")
		}
		sql := os.Args[2]
		upper := strings.ToUpper(strings.TrimSpace(sql))
		if !strings.HasPrefix(upper, "SELECT") && !strings.HasPrefix(upper, "SHOW") && !strings.HasPrefix(upper, "DESCRIBE") {
			log.Fatal("--query 只允许 SELECT / SHOW / DESCRIBE")
		}
		rows, err := db.Raw(sql).Rows()
		if err != nil {
			log.Fatal("query:", err)
		}
		defer func() { _ = rows.Close() }()
		cols, _ := rows.Columns()
		fmt.Println(strings.Join(cols, " | "))
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		for rows.Next() {
			if err := rows.Scan(ptrs...); err != nil {
				log.Fatal("scan:", err)
			}
			line := make([]string, len(cols))
			for i, v := range vals {
				if v == nil {
					line[i] = "NULL"
				} else if b, ok := v.([]byte); ok {
					line[i] = string(b)
				} else {
					line[i] = fmt.Sprint(v)
				}
			}
			fmt.Println(strings.Join(line, " | "))
		}
		return
	}

	var rows []indexRow
	if err := db.Raw("SHOW INDEX FROM " + table).Scan(&rows).Error; err != nil {
		log.Fatalf("SHOW INDEX FROM %s 失败: %v", table, err)
	}
	if len(rows) == 0 {
		log.Fatalf("表 %s 不存在或没有任何索引", table)
	}

	fmt.Printf("表 %s 的索引：\n", table)
	current := ""
	for _, r := range rows {
		if r.KeyName != current {
			current = r.KeyName
			unique := "普通"
			if r.NonUnique == 0 {
				unique = "唯一"
			}
			fmt.Printf("\n  [%s] %s\n", unique, r.KeyName)
		}
		fmt.Printf("    %d. %s\n", r.SeqInIndex, r.ColumnName)
	}

	var count int64
	if err := db.Table(table).Count(&count).Error; err != nil {
		log.Printf("统计行数失败: %v", err)
	} else {
		fmt.Printf("\n总行数: %d\n", count)
	}
}
