package main

import (
	"context"
	"fmt"
	"log"

	gormlogger "gorm.io/gorm/logger"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/internal/service"
)

func main() {
	dsn := "test24yangshi:test24yangshi123!@tcp(rm-bp1w012d612w344y7po.mysql.rds.aliyuncs.com:3306)/yuan-test?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Info),
	})
	if err != nil {
		log.Fatal("open:", err)
	}

	userRepo := repository.NewUserRepo(db)
	userRoleRepo := repository.NewUserRoleRepo(db)
	kgRepo := repository.NewKindergartenRepo(db)

	svc := service.NewSystemService(db, userRepo, userRoleRepo, kgRepo)

	err = svc.Initialize(context.Background(), service.InitRequest{
		KindergartenName: "TestKG",
		AdminPhone:       "13800138000",
		AdminPassword:    "Admin@123",
		AdminNickname:    "admin",
	})
	if err != nil {
		fmt.Println("ERROR:", err)
		return
	}
	fmt.Println("OK - init success")

	var userCount, roleCount, kgCount int64
	db.Model(&domain.User{}).Count(&userCount)
	db.Model(&domain.UserRole{}).Count(&roleCount)
	db.Model(&domain.Kindergarten{}).Count(&kgCount)
	fmt.Printf("users=%d roles=%d kgs=%d\n", userCount, roleCount, kgCount)

	// 清理测试数据
	db.Unscoped().Where("1=1").Delete(&domain.UserRole{})
	db.Unscoped().Where("1=1").Delete(&domain.User{})
	db.Unscoped().Where("1=1").Delete(&domain.Kindergarten{})
	fmt.Println("Cleaned up")
}
