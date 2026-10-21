package main

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"kindergarten-server/internal/config"
	"kindergarten-server/internal/service"
)

func main() {
	cfg, err := config.Load("configs/app.ini")
	if err != nil { log.Fatal(err) }
	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil { log.Fatal(err) }
	hash, err := service.HashPassword("Parent@123")
	if err != nil { log.Fatal(err) }
	r := db.Exec("UPDATE users SET password_hash=? WHERE id IN (61,65)", hash)
	fmt.Println("updated rows:", r.RowsAffected)
}
