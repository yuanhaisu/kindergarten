package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/internal/service"
)

// 导入测试打真实 MySQL（与 service/repository 测试同源），只用合成 CSV：
// 数据许可约束禁止提交/分发任何真实食物成分数据。
const (
	dietFoodCmdTestKG   uint64 = 990992
	dietFoodCmdTestKG2  uint64 = 990993
	dietFoodCmdPreset          = "s5-import-test-"
	dietFoodCmdCategory        = "S5导入测试"
)

func openDietFoodCmdTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	cfgPath := filepath.Join("..", "..", "configs", "app.ini")
	if _, err := os.Stat(cfgPath); err != nil {
		t.Skipf("跳过：找不到 %s（本地无 DB 配置）", cfgPath)
	}
	cfg, err := config.Load(cfgPath)
	require.NoError(t, err, "load config")
	if cfg.MySQL.DSN == "" {
		t.Skip("跳过：MySQL DSN 为空")
	}
	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err, "open db")
	require.NoError(t, db.AutoMigrate(
		&domain.Kindergarten{},
		&domain.MealConfig{}, &domain.MealConfigMeal{},
		&domain.FoodCategory{}, &domain.Food{},
		&domain.Recipe{}, &domain.RecipeItem{},
		&domain.MealPlan{}, &domain.MealPlanItem{},
		&domain.TemplatePlan{}, &domain.TemplatePlanItem{},
		&domain.DriAgeTarget{},
	), "auto migrate diet tables")
	return db
}

// ensureDietFoodCmdTestKindergarten 为分发测试准备一个真实园所行（高 ID，清理时删除）。
func ensureDietFoodCmdTestKindergarten(t *testing.T, db *gorm.DB, id uint64, name string) {
	t.Helper()
	var count int64
	require.NoError(t, db.Model(&domain.Kindergarten{}).Where("id = ?", id).Count(&count).Error)
	if count > 0 {
		return
	}
	require.NoError(t, db.Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: id},
		Name:      name,
	}).Error)
}

func cleanupDietFoodCmdTestData(t *testing.T, db *gorm.DB) {
	t.Helper()
	stmts := []struct {
		sql  string
		args []any
	}{
		{"DELETE FROM diet_template_plan_items WHERE kindergarten_id IN ?", []any{[]uint64{dietFoodCmdTestKG, dietFoodCmdTestKG2}}},
		{"DELETE FROM diet_template_plans WHERE kindergarten_id IN ?", []any{[]uint64{dietFoodCmdTestKG, dietFoodCmdTestKG2}}},
		{"DELETE FROM diet_foods WHERE kindergarten_id IN ?", []any{[]uint64{dietFoodCmdTestKG, dietFoodCmdTestKG2}}},
		{"DELETE FROM diet_foods WHERE kindergarten_id = 0 AND preset_key LIKE ?", []any{dietFoodCmdPreset + "%"}},
		{"DELETE FROM diet_food_categories WHERE name LIKE ?", []any{dietFoodCmdCategory + "%"}},
		{"DELETE FROM kindergartens WHERE id IN ?", []any{[]uint64{dietFoodCmdTestKG, dietFoodCmdTestKG2}}},
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt.sql, stmt.args...).Error; err != nil {
			t.Logf("cleanup %q: %v", stmt.sql, err)
		}
	}
}

func newDietFoodCmdTestService(db *gorm.DB) *service.DietSeedService {
	return service.NewDietSeedService(
		db,
		repository.NewDietFoodRepo(db),
		repository.NewDietTemplateRepo(db),
		repository.NewDietDriRepo(db),
	)
}

// buildDietFoodTestCSV 按 service 的表头定义生成合成 CSV；values 只写关心的列，
// 其余列留空（导入为 NULL）。
func buildDietFoodTestCSV(rows []map[string]string) string {
	header := append([]string{}, service.DietFoodImportRequiredColumns()...)
	header = append(header, service.DietFoodImportOptionalColumns()...)

	var b strings.Builder
	b.WriteString(strings.Join(header, ","))
	b.WriteString("\n")
	for _, row := range rows {
		cells := make([]string, len(header))
		for i, column := range header {
			cells[i] = row[column]
		}
		b.WriteString(strings.Join(cells, ","))
		b.WriteString("\n")
	}
	return b.String()
}

func dietFoodTestRows(edibleRice float64) []map[string]string {
	return []map[string]string{
		{
			"preset_key":     dietFoodCmdPreset + "rice",
			"name":           "S5合成大米",
			"spell_code":     "S5HCDM",
			"category_l1":    "谷类及制品",
			"category_l2":    dietFoodCmdCategory + "谷物",
			"edible_portion": fmt.Sprintf("%g", edibleRice),
			"energy":         "123.4",
			"protein":        "6.5",
			// vitamin_c 留空 → NULL
		},
		{
			"preset_key":     dietFoodCmdPreset + "greens",
			"name":           "S5合成青菜",
			"spell_code":     "S5HCQC",
			"category_l1":    "蔬菜类及制品",
			"category_l2":    dietFoodCmdCategory + "叶菜",
			"edible_portion": "90",
			"vitamin_c":      "—",
			"iron":           "-",
			"calcium":        "80",
		},
	}
}

// TestParseDietFoodCSVNulSemantics 确认空值/破折号 → NULL，非法数字与重复键 → 错误行。
func TestParseDietFoodCSVNulSemantics(t *testing.T) {
	rows, rowErrors, err := service.ParseDietFoodCSV(strings.NewReader(
		buildDietFoodTestCSV(dietFoodTestRows(100))))
	require.NoError(t, err)
	require.Empty(t, rowErrors)
	require.Len(t, rows, 2)

	require.NotNil(t, rows[0].Nutrients.Energy)
	assert.Equal(t, 123.4, *rows[0].Nutrients.Energy)
	assert.Nil(t, rows[0].Nutrients.VitaminC, "空单元格必须解析为 NULL")
	assert.Nil(t, rows[1].Nutrients.VitaminC, "— 必须解析为 NULL")
	assert.Nil(t, rows[1].Nutrients.Iron, "- 必须解析为 NULL")
	require.NotNil(t, rows[1].Nutrients.Calcium)
	assert.Equal(t, 80.0, *rows[1].Nutrients.Calcium)
	assert.Equal(t, "S5HCDM", rows[0].SpellCode)
	assert.Equal(t, domain.DietGroup12Vegetable, *rows[1].Group12, "group12 应由一级分类推导")

	badNumber := dietFoodTestRows(100)
	badNumber[0]["energy"] = "abc"
	badRows, badErrors, err := service.ParseDietFoodCSV(strings.NewReader(
		buildDietFoodTestCSV(badNumber)))
	require.NoError(t, err)
	assert.Len(t, badRows, 1)
	require.Len(t, badErrors, 1)
	assert.Contains(t, badErrors[0].Reason, "energy")

	duplicate := dietFoodTestRows(100)
	duplicate[1]["preset_key"] = duplicate[0]["preset_key"]
	dupRows, dupErrors, err := service.ParseDietFoodCSV(strings.NewReader(
		buildDietFoodTestCSV(duplicate)))
	require.NoError(t, err)
	assert.Len(t, dupRows, 1)
	require.Len(t, dupErrors, 1)
	assert.Contains(t, dupErrors[0].Reason, "重复")

	unknown := dietFoodTestRows(100)
	unknown[0]["category_l1"] = "不存在的类"
	_, rowErrors, err = service.ParseDietFoodCSV(strings.NewReader(
		buildDietFoodTestCSV(unknown)))
	require.NoError(t, err)
	require.Len(t, rowErrors, 1)
	assert.Contains(t, rowErrors[0].Reason, "未知一级分类")
}

// TestParseDietFoodCSVRequiresHeader 确认缺列是整批错误。
func TestParseDietFoodCSVRequiresHeader(t *testing.T) {
	_, _, err := service.ParseDietFoodCSV(strings.NewReader("preset_key,name\nx,y\n"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "缺少必需列")
}

// TestDietFoodImport 覆盖四类计数、二次导入幂等、NULL 不转 0、--kindergarten 分发
// 与 TC-NEW-DIE-0016（园所已修改行不被覆盖，冲突进报告）。
func TestDietFoodImport(t *testing.T) {
	db := openDietFoodCmdTestDB(t)
	cleanupDietFoodCmdTestData(t, db)
	t.Cleanup(func() { cleanupDietFoodCmdTestData(t, db) })
	ensureDietFoodCmdTestKindergarten(t, db, dietFoodCmdTestKG, "S5导入测试园")

	svc := newDietFoodCmdTestService(db)
	ctx := context.Background()

	parse := func(edibleRice float64) []service.DietFoodImportRow {
		rows, rowErrors, err := service.ParseDietFoodCSV(strings.NewReader(
			buildDietFoodTestCSV(dietFoodTestRows(edibleRice))))
		require.NoError(t, err)
		require.Empty(t, rowErrors)
		return rows
	}

	first, err := svc.ImportFoods(ctx, parse(100), dietFoodCmdTestKG)
	require.NoError(t, err)
	assert.Equal(t, 2, first.Added)
	assert.Zero(t, first.Updated)
	assert.Zero(t, first.Skipped)
	assert.Empty(t, first.Errors)
	assert.Equal(t, 2, first.Distributed.Added)

	var presetRice domain.Food
	require.NoError(t, db.Where("kindergarten_id = ? AND preset_key = ?",
		domain.DietPlatformKindergartenID, dietFoodCmdPreset+"rice").First(&presetRice).Error)
	require.NotNil(t, presetRice.Energy)
	assert.Equal(t, 123.4, *presetRice.Energy)
	require.NotNil(t, presetRice.Heat, "heat 缺省时由 energy × 4.186 派生")
	assert.InDelta(t, 516.55, *presetRice.Heat, 0.001)
	assert.Nil(t, presetRice.VitaminC, "缺失营养项写 NULL，不得写 0")

	var nullCount, zeroCount int64
	require.NoError(t, db.Raw(
		"SELECT COUNT(*) FROM diet_foods WHERE kindergarten_id = 0 AND preset_key = ? AND vitamin_c IS NULL",
		dietFoodCmdPreset+"rice").Scan(&nullCount).Error)
	require.NoError(t, db.Raw(
		"SELECT COUNT(*) FROM diet_foods WHERE kindergarten_id = 0 AND preset_key = ? AND vitamin_c = 0",
		dietFoodCmdPreset+"rice").Scan(&zeroCount).Error)
	assert.EqualValues(t, 1, nullCount)
	assert.Zero(t, zeroCount)

	var parkRice domain.Food
	require.NoError(t, db.Where("kindergarten_id = ? AND preset_key = ?",
		dietFoodCmdTestKG, dietFoodCmdPreset+"rice").First(&parkRice).Error)
	assert.Equal(t, domain.DietFoodSourceSystem, parkRice.Source)
	assert.False(t, parkRice.IsCustomized)
	assert.Equal(t, 100.0, parkRice.EdiblePercent)

	second, err := svc.ImportFoods(ctx, parse(100), dietFoodCmdTestKG)
	require.NoError(t, err)
	assert.Zero(t, second.Added)
	assert.Zero(t, second.Updated)
	assert.Equal(t, 2, second.Skipped, "同一 CSV 二次导入必须全部跳过")
	assert.Zero(t, second.Distributed.Added)
	assert.Equal(t, 2, second.Distributed.Skipped)

	// 模拟园所修改了预置副本：改名 + is_customized。
	require.NoError(t, db.Model(&domain.Food{}).Where("id = ?", parkRice.ID).
		Updates(map[string]any{
			"is_customized": true,
			"name":          "S5导入测试园所改名",
		}).Error)
	var customized domain.Food
	require.NoError(t, db.Where("id = ?", parkRice.ID).First(&customized).Error)
	require.True(t, customized.IsCustomized)

	third, err := svc.ImportFoods(ctx, parse(95), dietFoodCmdTestKG)
	require.NoError(t, err)
	assert.Equal(t, 1, third.Updated, "kg=0 预置行应被更新")
	assert.Equal(t, 1, third.Skipped)
	require.Len(t, third.Distributed.Conflicts, 1, "TC-NEW-DIE-0016：升级必须报告冲突")
	assert.Equal(t, dietFoodCmdPreset+"rice", third.Distributed.Conflicts[0].PresetKey)

	var parkAfter domain.Food
	require.NoError(t, db.Where("id = ?", customized.ID).First(&parkAfter).Error)
	assert.Equal(t, "S5导入测试园所改名", parkAfter.Name, "园所已修改行不得被覆盖")
	assert.Equal(t, 100.0, parkAfter.EdiblePercent)

	var presetAfter domain.Food
	require.NoError(t, db.Where("kindergarten_id = ? AND preset_key = ?",
		domain.DietPlatformKindergartenID, dietFoodCmdPreset+"rice").First(&presetAfter).Error)
	assert.Equal(t, 95.0, presetAfter.EdiblePercent)

	var presetCount int64
	require.NoError(t, db.Model(&domain.Food{}).
		Where("kindergarten_id = ? AND preset_key LIKE ?", domain.DietPlatformKindergartenID,
			dietFoodCmdPreset+"%").
		Count(&presetCount).Error)
	assert.EqualValues(t, 2, presetCount, "换内容重导不得产生重复预置行")
}

// TestRunHelpExitZero 覆盖验收：--help 退出码 0 且能看到表头。
func TestRunHelpExitZero(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--help"}, &stdout, &stderr)
	assert.Equal(t, 0, code)
	assert.Contains(t, stdout.String(), "--csv")
	assert.Contains(t, stdout.String(), "preset_key")
	assert.Contains(t, stdout.String(), "27 个营养列")
	assert.Contains(t, stdout.String(), "edible_portion")
}

// TestRunDryRun 覆盖 dry-run：只解析校验，不连库。
func TestRunDryRun(t *testing.T) {
	dir := t.TempDir()
	valid := filepath.Join(dir, "valid.csv")
	require.NoError(t, os.WriteFile(valid,
		[]byte(buildDietFoodTestCSV(dietFoodTestRows(100))), 0o600))

	var stdout, stderr bytes.Buffer
	code := run([]string{"--dry-run", "--csv", valid}, &stdout, &stderr)
	assert.Equal(t, 0, code)
	assert.Contains(t, stdout.String(), "dry-run")
	assert.Contains(t, stdout.String(), "校验通过 2 行")

	invalid := filepath.Join(dir, "invalid.csv")
	rows := dietFoodTestRows(100)
	rows[0]["energy"] = "not-a-number"
	require.NoError(t, os.WriteFile(invalid,
		[]byte(buildDietFoodTestCSV(rows)), 0o600))
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--dry-run", "--csv", invalid}, &stdout, &stderr)
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr.String(), "energy")

	code = run(nil, &stdout, &stderr)
	assert.Equal(t, 2, code, "缺少 --csv 应返回参数错误")
}
