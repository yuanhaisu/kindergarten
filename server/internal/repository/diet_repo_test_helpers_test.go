package repository

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
)

// 膳食仓储测试打真实 MySQL（本地 configs/app.ini），与 domain / notice 仓储测试同源。
// 每个测试用独立的高位园所 ID；清理一律原始 DELETE 硬删——软删除行会占住唯一索引，
// 重跑时撞键（CLAUDE.md「软删除+唯一索引」已知坑）。
//
// 表结构用 AutoMigrate 保证与模型一致（部署时跑的是同一份 AutoMigrate）。

var dietRepoTestDB *gorm.DB

func openDietRepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	if dietRepoTestDB != nil {
		return dietRepoTestDB
	}
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
		&domain.FoodCategory{}, &domain.Food{},
		&domain.Recipe{}, &domain.RecipeItem{},
		&domain.MealPlan{}, &domain.MealPlanItem{},
		&domain.TemplatePlan{}, &domain.TemplatePlanItem{},
		&domain.DriAgeTarget{},
	), "auto migrate diet tables")
	dietRepoTestDB = db
	return db
}

// dietHardDelete 执行原始 DELETE：绕过 GORM 软删除，确保测试可重复执行。
func dietHardDelete(t *testing.T, db *gorm.DB, sql string, args ...any) {
	t.Helper()
	if err := db.Exec(sql, args...).Error; err != nil {
		t.Logf("cleanup %q: %v", sql, err)
	}
}
