// setadmin — 创建/更新管理后台默认账号。
//
// 用法: go run ./cmd/setadmin [phone] [password]
// 默认: phone=admin password=123456
package main

import (
	"fmt"
	"log"
	"os"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
)

func main() {
	phone := "admin"
	password := "123456"
	if len(os.Args) >= 3 {
		phone = os.Args[1]
		password = os.Args[2]
	} else if len(os.Args) == 2 {
		password = os.Args[1]
	}

	cfg, err := config.Load("configs/app.ini")
	if err != nil {
		log.Fatal(err)
	}
	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}

	// 查找是否已有该 phone 的用户
	var user domain.User
	err = db.Where("phone = ?", phone).First(&user).Error
	if err != nil {
		// 不存在则创建
		user = domain.User{
			Nickname:     "admin",
			Phone:        phone,
			PasswordHash: string(hash),
		}
		if err := db.Create(&user).Error; err != nil {
			log.Fatal("create user:", err)
		}
		fmt.Printf("created admin user id=%d phone=%s\n", user.ID, phone)
	} else {
		// 已存在则更新密码
		user.PasswordHash = string(hash)
		user.Nickname = "admin"
		if err := db.Save(&user).Error; err != nil {
			log.Fatal("update user:", err)
		}
		fmt.Printf("updated admin user id=%d phone=%s\n", user.ID, phone)
	}

	// 确保 admin 角色（绑定园所 ID=4）
	kgID := uint64(4)
	var roleCount int64
	db.Model(&domain.UserRole{}).
		Where("user_id = ? AND role = ?", user.ID, "admin").
		Count(&roleCount)
	if roleCount == 0 {
		ur := domain.UserRole{
			UserID:          user.ID,
			Role:            "admin",
			AdminRole:       "super_admin",
			KindergartenID:  &kgID,
		}
		if err := db.Create(&ur).Error; err != nil {
			log.Fatal("create role:", err)
		}
		fmt.Println("created admin role (kg=4)")
	} else {
		// 更新 KindergartenID
		db.Model(&domain.UserRole{}).
			Where("user_id = ? AND role = ?", user.ID, "admin").
			Update("kindergarten_id", kgID)
		fmt.Println("updated admin role kindergarten_id=4")
	}

	fmt.Printf("\n=== 管理后台账号就绪 ===\n账号: %s\n密码: %s\n", phone, password)
}
