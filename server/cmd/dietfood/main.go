package main

// dietfood 把授权食物成分 CSV 导入平台预置食物库（kg=0），可选分发到指定园所。
//
// 数据许可约束（design Context / Migration Plan）：仓库不提交、不分发任何真实食物
// 成分数据；本命令只读取使用者提供的授权文件。测试只用合成 CSV。
//
// 用法：
//
//	go run ./cmd/dietfood --csv <path> [--kindergarten <id>] [--dry-run]
//
// 幂等：按 preset_key upsert kg=0 行；--kindergarten 按 (kindergarten_id, preset_key)
// 复制/更新，园所已修改的行（is_customized=true）不覆盖，冲突写入报告。
//
// 退出码：0 成功；1 有行级错误；2 参数/读文件/写库失败。

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"kindergarten-server/cmd/internal/seedutil"
	"kindergarten-server/internal/repository"
	"kindergarten-server/internal/service"
)

const dietFoodMaxReportedErrors = 20

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dietfood", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	csvPath := fs.String("csv", "", "授权食物成分 CSV 路径")
	kindergartenID := fs.Uint64("kindergarten", 0, "导入后复制/更新到指定园所 ID；0 表示只写平台预置")
	dryRun := fs.Bool("dry-run", false, "只解析与校验，不连库、不写库")
	fs.Usage = func() { fmt.Fprint(stdout, dietFoodUsage()) }

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0 // flag 内部已调用 Usage，帮助信息已输出
		}
		fmt.Fprintln(stderr, "参数错误:", err)
		fmt.Fprint(stderr, dietFoodUsage())
		return 2
	}
	if strings.TrimSpace(*csvPath) == "" {
		fmt.Fprintln(stderr, "缺少 --csv 参数")
		fmt.Fprint(stderr, dietFoodUsage())
		return 2
	}

	file, err := os.Open(*csvPath)
	if err != nil {
		fmt.Fprintln(stderr, "打开 CSV 失败:", err)
		return 2
	}
	defer file.Close()

	rows, parseErrors, err := service.ParseDietFoodCSV(file)
	if err != nil {
		fmt.Fprintln(stderr, "解析 CSV 失败:", err)
		return 2
	}

	if *dryRun {
		fmt.Fprintf(stdout, "dry-run：解析 %d 行，校验通过 %d 行，错误 %d 行（未连库、未写库）\n",
			len(rows)+len(parseErrors), len(rows), len(parseErrors))
		printImportErrors(stderr, parseErrors)
		if len(parseErrors) > 0 {
			return 1
		}
		return 0
	}
	if len(rows) == 0 {
		fmt.Fprintf(stdout, "预置库 (kg=0)：新增 0 / 更新 0 / 跳过 0 / 错误 %d\n", len(parseErrors))
		printImportErrors(stderr, parseErrors)
		if len(parseErrors) == 0 {
			fmt.Fprintln(stderr, "CSV 没有可导入的数据行")
		}
		return 1
	}

	db := seedutil.Open()
	seeder := service.NewDietSeedService(
		db,
		repository.NewDietFoodRepo(db),
		repository.NewDietTemplateRepo(db),
		repository.NewDietDriRepo(db),
	)
	result, err := seeder.ImportFoods(context.Background(), rows, *kindergartenID)
	if err != nil {
		fmt.Fprintln(stderr, "导入失败:", err)
		return 2
	}

	allErrors := append(parseErrors, result.Errors...)
	fmt.Fprintf(stdout, "预置库 (kg=0)：新增 %d / 更新 %d / 跳过 %d / 错误 %d\n",
		result.Added, result.Updated, result.Skipped, len(allErrors))
	if result.Globals.CategoriesAdded > 0 || result.Globals.DriRowsAdded > 0 {
		fmt.Fprintf(stdout, "全局种子：新增分类 %d / 新增 DRIs %d\n",
			result.Globals.CategoriesAdded, result.Globals.DriRowsAdded)
	}
	if *kindergartenID > 0 {
		fmt.Fprintf(stdout, "分发到园所 %d：新增 %d / 更新 %d / 跳过 %d / 冲突 %d\n",
			*kindergartenID,
			result.Distributed.Added, result.Distributed.Updated,
			result.Distributed.Skipped, len(result.Distributed.Conflicts))
		printImportConflicts(stdout, result.Distributed.Conflicts)
	}
	printImportErrors(stderr, allErrors)

	if len(allErrors) > 0 {
		return 1
	}
	return 0
}

func printImportErrors(w io.Writer, failures []service.DietFoodImportError) {
	if len(failures) == 0 {
		return
	}
	fmt.Fprintf(w, "错误明细（最多 %d 条）：\n", dietFoodMaxReportedErrors)
	for i, failure := range failures {
		if i >= dietFoodMaxReportedErrors {
			fmt.Fprintf(w, "  …另有 %d 条未显示\n", len(failures)-dietFoodMaxReportedErrors)
			break
		}
		fmt.Fprintf(w, "  第 %d 行 %s: %s\n", failure.Line, failure.PresetKey, failure.Reason)
	}
}

func printImportConflicts(w io.Writer, conflicts []service.DietFoodImportConflict) {
	if len(conflicts) == 0 {
		return
	}
	fmt.Fprintf(w, "冲突明细（园所已修改，未覆盖；最多 %d 条）：\n", dietFoodMaxReportedErrors)
	for i, conflict := range conflicts {
		if i >= dietFoodMaxReportedErrors {
			fmt.Fprintf(w, "  …另有 %d 条未显示\n", len(conflicts)-dietFoodMaxReportedErrors)
			break
		}
		fmt.Fprintf(w, "  preset_key=%s name=%s\n", conflict.PresetKey, conflict.Name)
	}
}

// dietFoodUsage 是 --help 输出。列名直接取自 service 的表头定义，避免文档与解析器漂移。
func dietFoodUsage() string {
	var b strings.Builder
	b.WriteString("用法:\n")
	b.WriteString("  go run ./cmd/dietfood --csv <path> [--kindergarten <id>] [--dry-run]\n\n")
	b.WriteString("必需列（UTF-8 CSV，表头名固定）：\n  ")
	b.WriteString(strings.Join(service.DietFoodImportRequiredColumns(), ","))
	b.WriteString("\n\n27 个营养列（每 100g 可食部）：\n  ")
	b.WriteString(strings.Join(service.DietFoodImportNutrientColumns(), ","))
	b.WriteString("\n\n可选列：\n  ")
	b.WriteString(strings.Join(service.DietFoodImportOptionalColumns(), ","))
	b.WriteString("\n  group12: 37–48；source_flags: animal|plant|milk|meat\n")
	b.WriteString("  group16: grain|tuber|vegetable|fruit|meat|egg|aquatic|soybean|nut|dairy|" +
		"instant_food|beverage|confection|oil|condiment|other\n")
	b.WriteString("  可选列留空时按一级分类的默认值推导。\n\n")
	b.WriteString("缺失值语义：空字符串、\"-\"、\"—\" 写入 NULL，绝不写成 0。\n")
	b.WriteString("非法数字、未知一级分类、重复 preset_key 会记为错误行，不中断整批。\n\n")
	b.WriteString("行为：\n")
	b.WriteString("  按 preset_key 幂等 upsert kg=0 平台预置库。\n")
	b.WriteString("  --kindergarten 把同样内容复制/更新到指定园；园所已修改的行不覆盖，列入冲突。\n")
	b.WriteString("  --dry-run 只解析与校验，不连库、不写库。\n\n")
	b.WriteString("前置：数据库已跑过 AutoMigrate（启动一次服务即可），且 configs/app.ini 可读。\n")
	b.WriteString("退出码：0 成功；1 有错误行；2 参数/读文件/写库失败。\n")
	return b.String()
}
