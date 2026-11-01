package domain

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"kindergarten-server/internal/config"
)

// MealConfigMeal.Enabled 曾带 gorm:"default:true"。GORM 在 Create 路径会把零值 false
// 替换成该默认值（callbacks/create.go），于是「停用晚餐」写库后仍是 true，
// 默认 0.95 系数被破坏（TC-NEW-DIE-0005/0006）。这条测试锁住「不得再声明默认值」。
func TestMealConfigMealEnabledHasNoGormDefault(t *testing.T) {
	parsed, err := schema.Parse(&MealConfigMeal{}, &sync.Map{}, schema.NamingStrategy{})
	require.NoError(t, err)

	field := parsed.LookUpField("Enabled")
	require.NotNil(t, field, "MealConfigMeal.Enabled 字段必须存在")

	assert.False(t, field.HasDefaultValue,
		"Enabled 不得声明数据库默认值：GORM 会把显式 false 当零值替换为默认 true")
	assert.Nil(t, field.DefaultValueInterface,
		"Enabled 不得声明数据库默认值：GORM 会把显式 false 当零值替换为默认 true")
}

// 独立测试园所 ID，避开真实数据；cleanup 用硬删，软删除行会占唯一索引。
const testDietConfigKG uint64 = 990901

func openDietConfigTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	cfgPath := filepath.Join("..", "..", "configs", "app.ini")
	if _, err := os.Stat(cfgPath); err != nil {
		t.Skipf("跳过：找不到 %s（本地无 DB 配置）", cfgPath)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.MySQL.DSN == "" {
		t.Skip("跳过：MySQL DSN 为空")
	}
	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Skipf("跳过：本地 MySQL 不可用（%v）", err)
	}
	// 等价于部署时 AutoMigrate 建表；两张表一起迁移，外键才有所指向。
	if err := db.AutoMigrate(&MealConfig{}, &MealConfigMeal{}); err != nil {
		t.Fatalf("auto migrate diet config tables: %v", err)
	}
	return db
}

func cleanupDietConfigData(t *testing.T, db *gorm.DB) {
	t.Helper()
	ctx := context.Background()
	del := func(sql string) {
		if err := db.WithContext(ctx).Exec(sql, testDietConfigKG).Error; err != nil {
			t.Logf("cleanup: %v", err)
		}
	}
	// 先删子表（外键），且用原始 DELETE 硬删，避免软删除行占唯一索引。
	del("DELETE FROM diet_meal_config_meals WHERE kindergarten_id = ?")
	del("DELETE FROM diet_meal_configs WHERE kindergarten_id = ?")
}

// 在没有数据库默认值的情况下，enabled=false 必须原样落库并读回 false。
func TestMealConfigMealEnabledFalseRoundTrip(t *testing.T) {
	db := openDietConfigTestDB(t)
	ctx := context.Background()

	cleanupDietConfigData(t, db)
	t.Cleanup(func() { cleanupDietConfigData(t, db) })

	cfg := MealConfig{
		KindergartenID: testDietConfigKG,
		CareType:       DietCareTypeDayCare,
		FoodInputType:  DietFoodInputTypeMarketWeight,
	}
	require.NoError(t, db.WithContext(ctx).Create(&cfg).Error)

	meal := MealConfigMeal{
		KindergartenID: testDietConfigKG,
		ConfigID:       cfg.ID,
		MealKey:        DietMealDinner,
		Name:           "晚餐",
		Time:           "17:00",
		EnergyPercent:  0,
		Sort:           5,
		Enabled:        false,
	}
	require.NoError(t, db.WithContext(ctx).Create(&meal).Error)
	require.NotZero(t, meal.ID)

	var got MealConfigMeal
	require.NoError(t, db.WithContext(ctx).First(&got, meal.ID).Error)
	assert.False(t, got.Enabled, "enabled=false 必须原样落库，不能被 GORM 默认值改写为 true")
}
