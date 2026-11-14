package main

// platformadmin 创建或重置首个平台方账号。
//
//	PLATFORM_ADMIN_USERNAME=ops PLATFORM_ADMIN_PASSWORD='xxxx' go run ./cmd/platformadmin
//
// 平台方账号与园所账号完全独立（ADR-0014），不走 /system/init。
// 幂等：账号已存在时只重置密码。

import (
	"context"
	"log"
	"os"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

func main() {
	username := os.Getenv("PLATFORM_ADMIN_USERNAME")
	password := os.Getenv("PLATFORM_ADMIN_PASSWORD")
	name := os.Getenv("PLATFORM_ADMIN_NAME")
	if username == "" || password == "" {
		log.Fatal("必须设置 PLATFORM_ADMIN_USERNAME 与 PLATFORM_ADMIN_PASSWORD")
	}
	if len(password) < 8 {
		log.Fatal("PLATFORM_ADMIN_PASSWORD 至少 8 位")
	}
	if name == "" {
		name = username
	}

	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "configs/app.ini"
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatal("load config: ", err)
	}
	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{})
	if err != nil {
		log.Fatal("open db: ", err)
	}
	repo := repository.NewPlatformAdminRepo(db)
	ctx := context.Background()

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("hash: ", err)
	}

	existing, err := repo.FindByUsername(ctx, username)
	if err != nil {
		log.Fatal("find: ", err)
	}
	if existing != nil {
		if err := db.WithContext(ctx).Model(&domain.PlatformAdmin{}).
			Where("id = ?", existing.ID).
			Updates(map[string]any{"password_hash": string(hash), "status": 1}).Error; err != nil {
			log.Fatal("reset password: ", err)
		}
		log.Printf("已重置平台账号 %s（id=%d）", username, existing.ID)
		return
	}
	admin := &domain.PlatformAdmin{Username: username, PasswordHash: string(hash), Name: name, Status: 1}
	if err := repo.Create(ctx, admin); err != nil {
		log.Fatal("create: ", err)
	}
	log.Printf("已创建平台账号 %s（id=%d）", username, admin.ID)
}
