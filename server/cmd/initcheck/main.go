package main

import (
	"context"
	"fmt"
	"log"
	"os"

	gormlogger "gorm.io/gorm/logger"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"kindergarten-server/cmd/internal/seedutil"
	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/internal/service"
)

// 注意：本命令会 Unscoped 清空 user_identities / user_roles / users /
// kindergartens，只能在确认过目标的测试库上运行。运行前必须设置
// SEED_ALLOW_DESTRUCTIVE=1 与 SEED_ALLOW_DEMO=1。
func main() {
	seedutil.RequireDestructiveOptIn("initcheck")

	cfg, err := config.Load(seedutil.ConfigPath())
	if err != nil {
		log.Fatal("load config:", err)
	}
	// DSN 取自 configs/app.ini，可用 MYSQL_DSN 环境变量覆盖。
	// 不要在源码里硬编码连接串——会随 git 历史泄漏。
	if cfg.MySQL.DSN == "" {
		log.Fatal("MySQL DSN 为空：请检查 configs/app.ini 或设置 MYSQL_DSN")
	}

	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Info),
	})
	if err != nil {
		log.Fatal("open:", err)
	}

	adminPassword := os.Getenv("INIT_ADMIN_PASSWORD")
	if adminPassword == "" {
		log.Fatal("环境变量 INIT_ADMIN_PASSWORD 未设置：拒绝使用默认口令")
	}

	userRepo := repository.NewUserRepo(db)
	userRoleRepo := repository.NewUserRoleRepo(db)
	kgRepo := repository.NewKindergartenRepo(db)
	dietSeedSvc := service.NewDietSeedService(
		db,
		repository.NewDietFoodRepo(db),
		repository.NewDietTemplateRepo(db),
		repository.NewDietDriRepo(db),
	)

	svc := service.NewSystemService(db, userRepo, userRoleRepo, kgRepo, dietSeedSvc)

	err = svc.Initialize(context.Background(), service.InitRequest{
		KindergartenName: "TestKG",
		AdminPhone:       "13800138000",
		AdminPassword:    adminPassword,
		AdminNickname:    "admin",
	})
	if err != nil {
		fmt.Println("ERROR:", err)
		return
	}
	fmt.Println("OK - init success")

	var userCount, roleCount, identityCount, kgCount int64
	db.Model(&domain.User{}).Count(&userCount)
	db.Model(&domain.UserRole{}).Count(&roleCount)
	db.Model(&domain.UserIdentity{}).Count(&identityCount)
	db.Model(&domain.Kindergarten{}).Count(&kgCount)
	fmt.Printf("users=%d roles=%d identities=%d kgs=%d\n", userCount, roleCount, identityCount, kgCount)

	// 清理测试数据：身份表先删，否则清空 users 后会留下指向不存在账号的身份。
	db.Unscoped().Where("1=1").Delete(&domain.UserIdentity{})
	db.Unscoped().Where("1=1").Delete(&domain.UserRole{})
	db.Unscoped().Where("1=1").Delete(&domain.User{})
	db.Unscoped().Where("1=1").Delete(&domain.Kindergarten{})
	fmt.Println("Cleaned up")
}
