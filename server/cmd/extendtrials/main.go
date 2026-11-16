// extendtrials — 存量园所 trial 授权延长（ADR-0018 #1）。
//
// 默认 dry-run（只读预演并产出经运营确认的清单）：
//
//	go run ./cmd/extendtrials -manifest ./extendtrials.csv
//
// 写库需要显式开关、执行人与校验值（默认读 dry-run 写出的 .sha256 侧车文件）：
//
//	EXTENDTRIALS_ALLOW=1 go run ./cmd/extendtrials -manifest ./extendtrials.csv \
//	    -apply -operator "张三"
//
// 回滚按同一份清单恢复原始到期日（只撤销当前值仍等于 apply 目标值的行）：
//
//	EXTENDTRIALS_ALLOW=1 go run ./cmd/extendtrials -manifest ./extendtrials.csv \
//	    -revert -operator "张三"
//
// 口径：未删除、2026-11-15（含）前开通、存在 source='trial' AND order_id IS NULL
// 授权的园所；排除有未取消订单或在期 subscription 授权的园所。
// 默认 dry-run：只做 SELECT，数据库不会发生任何变化。
// manifest 每条 trial 授权一行（可逐条回滚），另存园所汇总；落盘后输出 SHA256。
//
// 时区固定 Asia/Shanghai：DATETIME 列存墙钟时间，连接时区决定参数如何渲染，
// 因此这里强制覆盖 DSN 的 loc，避免在 UTC 容器上整体偏移 8 小时。
//
// 配置读 configs/app.ini（CONFIG_PATH / MYSQL_DSN 可覆盖）。输出不含 DSN 或账号。
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	mysqldrv "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"kindergarten-server/internal/config"
)

const policyDateLabel = "2026-11-15"

func main() {
	manifestPath := flag.String("manifest", "", "manifest 路径（dry-run 写入；apply/revert 读取）")
	apply := flag.Bool("apply", false, "按 manifest 延长 trial 授权至目标日（需 EXTENDTRIALS_ALLOW=1）")
	revert := flag.Bool("revert", false, "按 manifest 恢复原始到期日（需 EXTENDTRIALS_ALLOW=1）")
	operator := flag.String("operator", "", "写操作执行人（apply/revert 必填，写入平台操作日志）")
	sha256Flag := flag.String("sha256", "", "期望的 manifest SHA256（缺省读取 <manifest>.sha256 侧车文件）")
	flag.Parse()

	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		log.Fatal("加载 Asia/Shanghai 失败（容器缺 tzdata？）: ", err)
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
		log.Fatal("MySQL DSN 为空：请检查 configs/app.ini 或设置 MYSQL_DSN")
	}

	dsnCfg, err := mysqldrv.ParseDSN(cfg.MySQL.DSN)
	if err != nil {
		log.Fatal("解析 MySQL DSN 失败（请检查 configs/app.ini / MYSQL_DSN）: ", sanitizeDSN(err, cfg.MySQL.DSN))
	}
	dsnCfg.ParseTime = true
	dsnCfg.Loc = loc

	db, err := gorm.Open(mysql.Open(dsnCfg.FormatDSN()), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		log.Fatal("open db: ", sanitizeDSN(err, dsnCfg.FormatDSN()))
	}

	opts := Options{
		Now:             time.Now().In(loc),
		CutoffExclusive: time.Date(2026, 11, 16, 0, 0, 0, 0, loc),
		TargetExpiry:    time.Date(2027, 1, 31, 23, 59, 59, 0, loc),
		ManifestPath:    *manifestPath,
		Apply:           *apply,
		Revert:          *revert,
		Operator:        *operator,
		ManifestSHA256:  *sha256Flag,
	}

	res, err := run(context.Background(), db, opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "错误:", sanitizeDSN(err, dsnCfg.FormatDSN()))
		os.Exit(1)
	}
	printResult(res, opts)
}

func printResult(res *Result, opts Options) {
	if res.Mode == "apply" || res.Mode == "revert" {
		printWriteResult(res, opts)
		return
	}
	printDryRunResult(res, opts)
}

func printWriteResult(res *Result, opts Options) {
	action := "延长"
	if res.Mode == "revert" {
		action = "恢复"
	}
	fmt.Printf("%s：按 manifest %s存量园所 trial 授权\n", res.Mode, action)
	fmt.Printf("manifest：%s（sha256: %s）\n", res.ManifestPath, res.ManifestSHA256)
	if res.Mode == "apply" {
		fmt.Printf("目标到期日：%s（Asia/Shanghai）\n", opts.TargetExpiry.Format(manifestTimeLayout))
	} else {
		fmt.Printf("回滚判定：当前到期日须等于 apply 目标日 %s（Asia/Shanghai），否则整体拒绝\n",
			opts.TargetExpiry.Format(manifestTimeLayout))
	}
	if res.AffectedRows == 0 {
		fmt.Println("影响：0 所园 / 0 条授权（幂等重跑，无实际变更）")
	} else {
		fmt.Printf("影响：%d 所园 / %d 条授权\n", res.AffectedKindergartens, res.AffectedRows)
	}
	if res.UnchangedRows > 0 {
		switch res.Mode {
		case "apply":
			fmt.Printf("未变更：%d 条（到期日已等于或晚于目标日，只延长不回退）\n", res.UnchangedRows)
		default:
			fmt.Printf("未变更：%d 条（已恢复过的行，跳过）\n", res.UnchangedRows)
		}
	}
	if res.LogID > 0 {
		fmt.Printf("平台操作日志：platform_operation_logs#%d\n", res.LogID)
	}
}

func printDryRunResult(res *Result, opts Options) {
	plan := res.Plan
	excludedTotal := len(plan.Excluded) + len(plan.NullCreatedAt)

	fmt.Printf("dry-run：存量园所 trial 延长预演（政策日 %s 前开通；目标到期日 %s；时区 Asia/Shanghai）\n",
		policyDateLabel, opts.TargetExpiry.Format(manifestTimeLayout))
	fmt.Printf("扫描 %d 所园所：命中 %d 所 / %d 条 trial 授权；排除 %d 所（其中 created_at 为空 %d 所）\n",
		plan.Scanned, len(plan.Selected), plan.TotalTrialRows(), excludedTotal, len(plan.NullCreatedAt))

	if len(plan.Selected) > 0 {
		earliest, latest := plan.Selected[0].OldEarliest, plan.Selected[0].OldLatest
		fmt.Println("待延长园所：")
		for _, g := range plan.Selected {
			if g.OldEarliest.Before(earliest) {
				earliest = g.OldEarliest
			}
			if g.OldLatest.After(latest) {
				latest = g.OldLatest
			}
			fmt.Printf("  %d\t%s\ttrial %d 条\t原到期日 %s ～ %s\n",
				g.KindergartenID, g.Name, g.TrialCount,
				g.OldEarliest.Format(manifestTimeLayout), g.OldLatest.Format(manifestTimeLayout))
		}
		fmt.Printf("汇总：%d 所园 / %d 条授权；原到期日 %s ～ %s\n",
			len(plan.Selected), plan.TotalTrialRows(),
			earliest.Format(manifestTimeLayout), latest.Format(manifestTimeLayout))
	} else {
		fmt.Println("结果：没有待延长的园所（数据库未做任何修改）")
	}

	if len(plan.Excluded) > 0 {
		fmt.Println("排除：")
		for _, ex := range plan.Excluded {
			fmt.Printf("  %d\t%s\t%s\n", ex.KindergartenID, ex.Name, ex.Reason.Label())
		}
	}
	if len(plan.NullCreatedAt) > 0 {
		fmt.Println("遗留行（created_at 为空，无法判定开通时间，未纳入本次延长，需人工核对）：")
		for _, legacy := range plan.NullCreatedAt {
			fmt.Printf("  %d\t%s\n", legacy.KindergartenID, legacy.Name)
		}
	}

	if res.ManifestWritten {
		fmt.Printf("manifest：%s（sha256: %s；每条 trial 授权一行，可逐条回滚）\n", res.ManifestPath, res.ManifestSHA256)
		fmt.Printf("校验值：%s（apply/revert 默认读取）\n", res.SHAFilePath)
		fmt.Printf("园所汇总：%s\n", res.SummaryPath)
	} else {
		fmt.Println("manifest：没有待延长园所，未生成 manifest（数据库未做任何修改）")
	}
}

// sanitizeDSN 抹掉错误文本中可能出现的 DSN：连接串含口令，不能进日志。
func sanitizeDSN(err error, dsn string) string {
	msg := err.Error()
	if dsn != "" {
		msg = strings.ReplaceAll(msg, dsn, "<dsn>")
	}
	return msg
}
