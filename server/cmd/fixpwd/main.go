package main
import (
	"log"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"kindergarten-server/internal/service"
	"kindergarten-server/internal/config"
)
func main() {
	cfg, _ := config.Load("configs/app.ini")
	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Warn)})
	if err != nil { log.Fatal(err) }
	hash, _ := service.HashPassword("Parent@123")
	db.Exec("UPDATE users SET password_hash=? WHERE phone IN ('13900010001','13900010002')", hash)
	log.Println("Done - parent passwords set to Parent@123")
}
