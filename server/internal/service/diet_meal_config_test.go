package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// 独立测试园所 ID，避开真实数据与其他测试文件；清理一律原始 DELETE 硬删
// （软删除行会占唯一索引，重跑撞键——CLAUDE.md 已知坑）。
const testDietMealConfigKG uint64 = 990981

func openDietMealConfigTestDB(t *testing.T) *gorm.DB {
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
		&domain.MealConfig{}, &domain.MealConfigMeal{},
		&domain.MealPlan{}, &domain.MealPlanItem{},
	), "auto migrate diet config/plan tables")
	return db
}

func cleanupDietMealConfigData(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, sql := range []string{
		"DELETE FROM diet_meal_plan_items WHERE kindergarten_id = ?",
		"DELETE FROM diet_meal_plans WHERE kindergarten_id = ?",
		"DELETE FROM diet_meal_config_meals WHERE kindergarten_id = ?",
		"DELETE FROM diet_meal_configs WHERE kindergarten_id = ?",
	} {
		if err := db.Exec(sql, testDietMealConfigKG).Error; err != nil {
			t.Logf("cleanup %q: %v", sql, err)
		}
	}
}

func newDietMealConfigService(t *testing.T) (*DietConfigService, *gorm.DB) {
	t.Helper()
	db := openDietMealConfigTestDB(t)
	cleanupDietMealConfigData(t, db)
	t.Cleanup(func() { cleanupDietMealConfigData(t, db) })
	return NewDietConfigService(repository.NewDietConfigRepo(db)), db
}

// standardDietConfigInput 返回与默认值一致的标准比例配置（日托、五餐启用、晚点停用）。
func standardDietConfigInput() DietMealConfigInput {
	return DietMealConfigInput{
		CareType:      domain.DietCareTypeDayCare,
		FoodInputType: domain.DietFoodInputTypeMarketWeight,
		RatioMode:     domain.DietRatioModeStandard,
		Meals:         defaultDietMealConfigMeals(),
	}
}

func customDietConfigInput() DietMealConfigInput {
	in := standardDietConfigInput()
	in.RatioMode = domain.DietRatioModeCustom
	return in
}

func dietMealInput(t *testing.T, in *DietMealConfigInput, key domain.DietMealKey) *DietMealConfigMealInput {
	t.Helper()
	for i := range in.Meals {
		if in.Meals[i].MealKey == key {
			return &in.Meals[i]
		}
	}
	t.Fatalf("fixture 缺少餐次 %s", key)
	return nil
}

func requireDietConfigInvalid(t *testing.T, err error, wantSubstring string) {
	t.Helper()
	require.Error(t, err)
	var invalid *DietConfigValidationError
	require.True(t, errors.As(err, &invalid), "必须是可读的业务校验错误，实际：%v", err)
	assert.Contains(t, invalid.Message, wantSubstring)
}

// TC-NEW-DIE-0001：新园所首次读取返回日托、标准比例、5 个默认餐次与市品重量口径。
func TestDietMealConfig_TC0001_FirstReadReturnsDefault(t *testing.T) {
	svc, db := newDietMealConfigService(t)
	ctx := context.Background()

	first, err := svc.GetConfig(ctx, testDietMealConfigKG)
	require.NoError(t, err)
	assert.Equal(t, domain.DietCareTypeDayCare, first.CareType)
	assert.Equal(t, "日托", first.CareTypeLabel)
	assert.Equal(t, domain.DietFoodInputTypeMarketWeight, first.FoodInputType)
	assert.Equal(t, "市品重量", first.FoodInputTypeLabel)
	assert.Equal(t, domain.DietRatioModeStandard, first.RatioMode)
	assert.Equal(t, "标准比例", first.RatioModeLabel)
	assert.Equal(t, 0.95, first.Coefficient)
	require.Len(t, first.Meals, 6)

	want := []struct {
		key     domain.DietMealKey
		name    string
		time    string
		percent float64
		enabled bool
	}{
		{domain.DietMealBreakfast, "早餐", "07:00", 25, true},
		{domain.DietMealMorningSnack, "早点", "09:00", 5, true},
		{domain.DietMealLunch, "午餐", "11:00", 35, true},
		{domain.DietMealAfternoonSnack, "午点", "14:30", 5, true},
		{domain.DietMealDinner, "晚餐", "17:00", 25, true},
		{domain.DietMealEveningSnack, "晚点", "20:00", 5, false},
	}
	for i, w := range want {
		assert.Equal(t, w.key, first.Meals[i].MealKey)
		assert.Equal(t, w.name, first.Meals[i].Name)
		assert.Equal(t, w.time, first.Meals[i].Time)
		assert.Equal(t, w.percent, first.Meals[i].EnergyPercent)
		assert.Equal(t, w.enabled, first.Meals[i].Enabled)
		assert.Equal(t, i+1, first.Meals[i].Sort)
	}

	// 读取路径不产生写入：默认值不落库，重复读取结果一致。
	var configRows int64
	require.NoError(t, db.Model(&domain.MealConfig{}).
		Where("kindergarten_id = ?", testDietMealConfigKG).Count(&configRows).Error)
	assert.Zero(t, configRows, "首次读取不得落库")

	second, err := svc.GetConfig(ctx, testDietMealConfigKG)
	require.NoError(t, err)
	assert.Equal(t, first, second, "重复读取必须返回相同默认值")
}

// TC-NEW-DIE-0002：标准模式保存默认比例（含改时间）后读取一致，系数 0.95。
func TestDietMealConfig_TC0002_SaveStandardAndReadBack(t *testing.T) {
	svc, db := newDietMealConfigService(t)
	ctx := context.Background()

	in := standardDietConfigInput()
	dietMealInput(t, &in, domain.DietMealBreakfast).Time = "07:15"
	dietMealInput(t, &in, domain.DietMealLunch).Time = "11:20"

	saved, err := svc.SaveConfig(ctx, testDietMealConfigKG, in)
	require.NoError(t, err, "标准比例保存默认值必须成功（跳过相邻上限）")
	assert.Equal(t, 0.95, saved.Coefficient)
	assert.Equal(t, domain.DietRatioModeStandard, saved.RatioMode)

	got, err := svc.GetConfig(ctx, testDietMealConfigKG)
	require.NoError(t, err)
	assert.Equal(t, saved, got, "保存后读取必须与保存结果一致")
	assert.Equal(t, "07:15", got.Meals[0].Time)
	assert.Equal(t, "11:20", got.Meals[2].Time)

	var config domain.MealConfig
	require.NoError(t, db.Where("kindergarten_id = ?", testDietMealConfigKG).First(&config).Error)
	assert.Equal(t, domain.DietRatioModeStandard, config.RatioMode, "ratio_mode 必须持久化")
}

// TC-NEW-DIE-0003：两种模式下，已启用餐次时间倒序保存被拒绝，且原配置不变。
func TestDietMealConfig_TC0003_RejectsTimeOutOfOrder(t *testing.T) {
	svc, _ := newDietMealConfigService(t)
	ctx := context.Background()

	saved, err := svc.SaveConfig(ctx, testDietMealConfigKG, standardDietConfigInput())
	require.NoError(t, err)

	for _, mode := range []domain.DietRatioMode{domain.DietRatioModeStandard, domain.DietRatioModeCustom} {
		t.Run(string(mode), func(t *testing.T) {
			bad := standardDietConfigInput()
			bad.RatioMode = mode
			dietMealInput(t, &bad, domain.DietMealLunch).Time = "08:30" // 早于早点 09:00
			if mode == domain.DietRatioModeCustom {
				// 自定模式基线本身超上限，先改成合规值，确保失败来自时间校验。
				dietMealInput(t, &bad, domain.DietMealBreakfast).EnergyPercent = 24
				dietMealInput(t, &bad, domain.DietMealLunch).EnergyPercent = 34
			}

			_, err := svc.SaveConfig(ctx, testDietMealConfigKG, bad)
			requireDietConfigInvalid(t, err, "递增")

			after, err := svc.GetConfig(ctx, testDietMealConfigKG)
			require.NoError(t, err)
			assert.Equal(t, saved, after, "校验失败不得落库")
		})
	}
}

// TC-NEW-DIE-0004：自定模式下相邻餐次供能占比超限被拒绝；标准模式跳过上限。
func TestDietMealConfig_TC0004_RejectsCustomAdjacentCap(t *testing.T) {
	svc, _ := newDietMealConfigService(t)
	ctx := context.Background()

	t.Run("full_care_breakfast_pair_30", func(t *testing.T) {
		in := customDietConfigInput()
		in.CareType = domain.DietCareTypeFullCare
		dietMealInput(t, &in, domain.DietMealMorningSnack).EnergyPercent = 6 // 25+6=31
		_, err := svc.SaveConfig(ctx, testDietMealConfigKG, in)
		requireDietConfigInvalid(t, err, "30")
	})

	t.Run("day_care_breakfast_pair_29", func(t *testing.T) {
		in := customDietConfigInput()
		dietMealInput(t, &in, domain.DietMealBreakfast).EnergyPercent = 24
		dietMealInput(t, &in, domain.DietMealMorningSnack).EnergyPercent = 6 // 24+6=30 > 29
		_, err := svc.SaveConfig(ctx, testDietMealConfigKG, in)
		requireDietConfigInvalid(t, err, "29")
	})

	t.Run("lunch_pair_39", func(t *testing.T) {
		// 全托下早餐对 25+5=30 合规，让失败点落在午餐+午点 35+5=40。
		// 默认占比在自定模式下超 39——这正是标准/自定双模式存在的原因。
		in := customDietConfigInput()
		in.CareType = domain.DietCareTypeFullCare
		_, err := svc.SaveConfig(ctx, testDietMealConfigKG, in)
		requireDietConfigInvalid(t, err, "39")
	})

	t.Run("valid_custom_saves", func(t *testing.T) {
		in := customDietConfigInput()
		dietMealInput(t, &in, domain.DietMealBreakfast).EnergyPercent = 24
		dietMealInput(t, &in, domain.DietMealLunch).EnergyPercent = 34 // 24+5=29、34+5=39
		saved, err := svc.SaveConfig(ctx, testDietMealConfigKG, in)
		require.NoError(t, err)
		assert.Equal(t, 0.93, saved.Coefficient) // 24+5+34+5+25
	})

	t.Run("standard_skips_cap", func(t *testing.T) {
		saved, err := svc.SaveConfig(ctx, testDietMealConfigKG, standardDietConfigInput())
		require.NoError(t, err, "标准模式 35+5=40 必须允许")
		assert.Equal(t, 0.95, saved.Coefficient)
	})
}

// TC-NEW-DIE-0005：标准五餐配置的折算系数为 0.95。
func TestDietMealConfig_TC0005_StandardCoefficient95(t *testing.T) {
	svc, _ := newDietMealConfigService(t)
	ctx := context.Background()

	saved, err := svc.SaveConfig(ctx, testDietMealConfigKG, standardDietConfigInput())
	require.NoError(t, err)
	assert.Equal(t, 0.95, saved.Coefficient, "25+5+35+5+25 = 95")

	snapshot, err := svc.BuildConfigSnapshot(ctx, testDietMealConfigKG)
	require.NoError(t, err)
	assert.Equal(t, 0.95, snapshot.Coefficient)
	assert.Equal(t, domain.DietRatioModeStandard, snapshot.RatioMode)
}

// TC-NEW-DIE-0006：停用晚餐后折算系数变为 0.70（标准模式允许保存）。
func TestDietMealConfig_TC0006_DisableDinnerCoefficient70(t *testing.T) {
	svc, _ := newDietMealConfigService(t)
	ctx := context.Background()

	in := standardDietConfigInput()
	dietMealInput(t, &in, domain.DietMealDinner).Enabled = false

	saved, err := svc.SaveConfig(ctx, testDietMealConfigKG, in)
	require.NoError(t, err)
	assert.Equal(t, 0.7, saved.Coefficient, "25+5+35+5 = 70")

	got, err := svc.GetConfig(ctx, testDietMealConfigKG)
	require.NoError(t, err)
	assert.Equal(t, 0.7, got.Coefficient)
	assert.False(t, got.Meals[4].Enabled)

	snapshot, err := svc.BuildConfigSnapshot(ctx, testDietMealConfigKG)
	require.NoError(t, err)
	assert.Equal(t, 0.7, snapshot.Coefficient)
}

// TC-NEW-DIE-0007：重量口径只接受市品重量，可食重量被拒绝且配置不变。
func TestDietMealConfig_TC0007_RejectsEdibleWeight(t *testing.T) {
	svc, _ := newDietMealConfigService(t)
	ctx := context.Background()

	saved, err := svc.SaveConfig(ctx, testDietMealConfigKG, standardDietConfigInput())
	require.NoError(t, err)

	bad := standardDietConfigInput()
	bad.FoodInputType = domain.DietFoodInputType("edible_weight")
	_, err = svc.SaveConfig(ctx, testDietMealConfigKG, bad)
	requireDietConfigInvalid(t, err, "市品重量")

	got, err := svc.GetConfig(ctx, testDietMealConfigKG)
	require.NoError(t, err)
	assert.Equal(t, domain.DietFoodInputTypeMarketWeight, got.FoodInputType)
	assert.Equal(t, saved, got, "被拒绝的口径不得落库")
}

// TC-NEW-DIE-0008：修改配置不回溯已保存食谱的 config_snapshot。
func TestDietMealConfig_TC0008_SnapshotNotRetroactive(t *testing.T) {
	svc, db := newDietMealConfigService(t)
	ctx := context.Background()

	original, err := svc.BuildConfigSnapshot(ctx, testDietMealConfigKG)
	require.NoError(t, err)
	require.Equal(t, 0.95, original.Coefficient)
	require.Equal(t, domain.DietRatioModeStandard, original.RatioMode)
	originalJSON, err := json.Marshal(original)
	require.NoError(t, err)

	planRepo := repository.NewDietPlanRepo(db)
	plan := &domain.MealPlan{
		KindergartenID: testDietMealConfigKG,
		AcademicYearID: 1,
		Name:           "TC0008 快照不回溯",
		StartDate:      time.Date(2026, 10, 26, 0, 0, 0, 0, time.Local),
		EndDate:        time.Date(2026, 10, 30, 0, 0, 0, 0, time.Local),
		ConfigSnapshot: string(originalJSON),
	}
	require.NoError(t, planRepo.Create(ctx, plan))
	require.NotZero(t, plan.ID)

	// 改成自定比例、不同占比与系数。
	changed := customDietConfigInput()
	dietMealInput(t, &changed, domain.DietMealBreakfast).EnergyPercent = 24
	dietMealInput(t, &changed, domain.DietMealLunch).EnergyPercent = 34
	dietMealInput(t, &changed, domain.DietMealDinner).EnergyPercent = 20 // 系数 0.88
	_, err = svc.SaveConfig(ctx, testDietMealConfigKG, changed)
	require.NoError(t, err)

	loaded, err := planRepo.FindByIDWithItems(ctx, testDietMealConfigKG, plan.ID)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	// MySQL 的 JSON 列会重排键并压缩空白，比较语义而不是原始字节。
	assert.JSONEq(t, string(originalJSON), loaded.ConfigSnapshot, "历史食谱快照不得被配置变更改写")

	var loadedSnapshot domain.DietConfigSnapshot
	require.NoError(t, json.Unmarshal([]byte(loaded.ConfigSnapshot), &loadedSnapshot))
	assert.Equal(t, 0.95, loadedSnapshot.Coefficient)
	assert.Equal(t, domain.DietRatioModeStandard, loadedSnapshot.RatioMode)
	assert.Equal(t, original, &loadedSnapshot)

	fresh, err := svc.BuildConfigSnapshot(ctx, testDietMealConfigKG)
	require.NoError(t, err)
	assert.Equal(t, 0.88, fresh.Coefficient)
	assert.Equal(t, domain.DietRatioModeCustom, fresh.RatioMode)
}

// 标准模式只允许固定占比；已启用餐次传非标准值必须被拒绝。
func TestDietMealConfig_StandardRejectsNonStandardPercent(t *testing.T) {
	svc, _ := newDietMealConfigService(t)
	ctx := context.Background()

	in := standardDietConfigInput()
	dietMealInput(t, &in, domain.DietMealBreakfast).EnergyPercent = 24
	_, err := svc.SaveConfig(ctx, testDietMealConfigKG, in)
	requireDietConfigInvalid(t, err, "标准比例")
}

// 缺省 ratio_mode 按 standard 保存（契约校验第 4 条）。
func TestDietMealConfig_DefaultRatioModeIsStandard(t *testing.T) {
	svc, db := newDietMealConfigService(t)
	ctx := context.Background()

	in := standardDietConfigInput()
	in.RatioMode = ""
	saved, err := svc.SaveConfig(ctx, testDietMealConfigKG, in)
	require.NoError(t, err)
	assert.Equal(t, domain.DietRatioModeStandard, saved.RatioMode)

	var config domain.MealConfig
	require.NoError(t, db.Where("kindergarten_id = ?", testDietMealConfigKG).First(&config).Error)
	assert.Equal(t, domain.DietRatioModeStandard, config.RatioMode)
}

func TestDietMealConfig_RejectsUnknownRatioMode(t *testing.T) {
	svc, _ := newDietMealConfigService(t)
	in := standardDietConfigInput()
	in.RatioMode = domain.DietRatioMode("free")
	_, err := svc.SaveConfig(context.Background(), testDietMealConfigKG, in)
	requireDietConfigInvalid(t, err, "比例模式")
}

func TestDietMealConfig_RejectsNoEnabledMeal(t *testing.T) {
	svc, _ := newDietMealConfigService(t)
	in := standardDietConfigInput()
	for i := range in.Meals {
		in.Meals[i].Enabled = false
	}
	_, err := svc.SaveConfig(context.Background(), testDietMealConfigKG, in)
	requireDietConfigInvalid(t, err, "至少需要启用一个餐次")
}

func TestDietMealConfig_RejectsNonPositiveEnabledPercent(t *testing.T) {
	svc, _ := newDietMealConfigService(t)
	in := customDietConfigInput()
	dietMealInput(t, &in, domain.DietMealBreakfast).EnergyPercent = 0
	_, err := svc.SaveConfig(context.Background(), testDietMealConfigKG, in)
	requireDietConfigInvalid(t, err, "大于 0")
}

func TestDietMealConfig_RejectsMissingOrDuplicateMealKeys(t *testing.T) {
	svc, _ := newDietMealConfigService(t)
	ctx := context.Background()

	missing := standardDietConfigInput()
	missing.Meals = missing.Meals[:5]
	_, err := svc.SaveConfig(ctx, testDietMealConfigKG, missing)
	requireDietConfigInvalid(t, err, "6 个已知餐次")

	duplicate := standardDietConfigInput()
	duplicate.Meals[5].MealKey = domain.DietMealDinner
	_, err = svc.SaveConfig(ctx, testDietMealConfigKG, duplicate)
	requireDietConfigInvalid(t, err, "重复")
}

func TestDietMealConfig_RejectsInvalidTimeFormat(t *testing.T) {
	svc, _ := newDietMealConfigService(t)
	in := standardDietConfigInput()
	dietMealInput(t, &in, domain.DietMealBreakfast).Time = "7:00"
	_, err := svc.SaveConfig(context.Background(), testDietMealConfigKG, in)
	requireDietConfigInvalid(t, err, "HH:mm")
}
