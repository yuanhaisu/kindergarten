// mktoken — 验证辅助工具：为指定手机号+角色签发一枚本地 JWT。
//
// 用途：在无法完成微信登录的自动化环境里，驱动家长端/老师端 API 做
// 本变更的集成验收（TC-NEW-VIS-*、TC-NEW-TGR-*）。签名密钥取自
// configs/app.ini（可用 CONFIG_PATH 覆盖），与本地后端同源。
//
// 仅限本机验证使用；不进入 HTTP 服务，且产物是短期 token。
// 用法：go run ./cmd/mktoken <phone> <role>
package main

import (
	"fmt"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: go run ./cmd/mktoken <phone> <role>")
		os.Exit(1)
	}
	phone, role := os.Args[1], os.Args[2]

	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "configs/app.ini"
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	if cfg.JWT.Secret == "" {
		fmt.Fprintln(os.Stderr, "jwt.secret 为空，无法签发")
		os.Exit(1)
	}

	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect db: %v\n", err)
		os.Exit(1)
	}

	var user domain.User
	if err := db.Where("phone = ?", phone).First(&user).Error; err != nil {
		fmt.Fprintf(os.Stderr, "user %s not found: %v\n", phone, err)
		os.Exit(1)
	}

	var roles []domain.UserRole
	if err := db.Where("user_id = ?", user.ID).Find(&roles).Error; err != nil {
		fmt.Fprintf(os.Stderr, "load roles: %v\n", err)
		os.Exit(1)
	}

	roleNames := make([]string, 0, len(roles))
	var kgID uint64
	for _, r := range roles {
		if r.KindergartenID != nil {
			kgID = *r.KindergartenID
		}
		roleNames = append(roleNames, r.Role)
	}

	target := role
	found := false
	for _, r := range roleNames {
		if r == role {
			found = true
			break
		}
	}
	if !found {
		fmt.Fprintf(os.Stderr, "user %s has roles %v, not %s\n", phone, roleNames, role)
		os.Exit(1)
	}

	ttl := time.Duration(cfg.JWT.AccessTTLMin) * time.Minute
	if ttl <= 0 {
		ttl = time.Hour
	}
	token, err := auth.SignAccess([]byte(cfg.JWT.Secret), fmt.Sprint(user.ID), []string{target}, kgID, ttl)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sign: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(token)
}
