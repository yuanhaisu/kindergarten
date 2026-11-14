package main

// redactlogs 一次性回填：把既有 operation_logs.detail 中可识别的手机号与身份证号掩码。
//
//	go run ./cmd/redactlogs                 # dry-run，只统计
//	go run ./cmd/redactlogs -dry-run=false  # 正式执行
//
// 只处理手机号与身份证号；自由文本里的姓名与学号无法可靠识别，
// 需要人工复核（新写入已由结构化写入点与 OperationLogRepo.Create 覆盖）。
import (
	"flag"
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"kindergarten-server/internal/config"
	"kindergarten-server/pkg/redact"
)

type logRow struct {
	ID     uint64 `gorm:"column:id"`
	Detail string `gorm:"column:detail"`
}

type options struct {
	DryRun bool
	Batch  int
	Limit  int
}

func main() {
	dryRun := flag.Bool("dry-run", true, "只统计将修改的行数，不写库")
	batch := flag.Int("batch", 500, "每批处理行数")
	limit := flag.Int("limit", 0, "最多扫描行数，0 表示不限制")
	flag.Parse()

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

	scanned, changed, err := run(db, "operation_logs", options{
		DryRun: *dryRun,
		Batch:  *batch,
		Limit:  *limit,
	})
	if err != nil {
		log.Fatal(err)
	}

	mode := "dry-run"
	if !*dryRun {
		mode = "applied"
	}
	fmt.Printf("redactlogs %s: scanned=%d changed=%d\n", mode, scanned, changed)
	fmt.Println("提示：历史自由文本中的姓名/学号无法自动识别，需人工复核；新写入已脱敏。")
}

// run 扫描指定表并回填；table 只允许由调用方传入固定值，不接受用户输入。
func run(db *gorm.DB, table string, opts options) (int64, int64, error) {
	if opts.Batch <= 0 {
		opts.Batch = 500
	}
	const piiFilter = `detail REGEXP '1[3-9][0-9]{9}|[0-9]{17}[0-9Xx]'`

	var scanned, changed int64
	var lastID uint64
	for {
		var rows []logRow
		q := db.Table(table).
			Select("id, COALESCE(detail, '') AS detail").
			Where("id > ?", lastID).
			Where(piiFilter).
			Order("id ASC").
			Limit(opts.Batch)
		if err := q.Find(&rows).Error; err != nil {
			return scanned, changed, fmt.Errorf("scan: %w", err)
		}
		if len(rows) == 0 {
			break
		}
		for _, row := range rows {
			lastID = row.ID
			scanned++
			next := redact.Text(row.Detail)
			if next == row.Detail {
				continue
			}
			changed++
			if opts.DryRun {
				continue
			}
			if err := db.Table(table).
				Where("id = ?", row.ID).
				Update("detail", next).Error; err != nil {
				return scanned, changed, fmt.Errorf("update id=%d: %w", row.ID, err)
			}
		}
		if opts.Limit > 0 && scanned >= int64(opts.Limit) {
			break
		}
	}
	return scanned, changed, nil
}
