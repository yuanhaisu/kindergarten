package main
import ("fmt";"log";"gorm.io/driver/mysql";"gorm.io/gorm";gormlogger "gorm.io/gorm/logger";"kindergarten-server/internal/config";"kindergarten-server/internal/domain")
func main() {
	cfg,_:=config.Load("configs/app.ini")
	db,err:=gorm.Open(mysql.Open(cfg.MySQL.DSN),&gorm.Config{Logger:gormlogger.Default.LogMode(gormlogger.Warn)})
	if err!=nil{log.Fatal(err)}
	var count int64
	db.Model(&domain.GrowthDailyLog{}).Where("kindergarten_id=4").Count(&count)
	fmt.Printf("Total growth logs: %d\n",count)
	var rows []domain.GrowthDailyLog
	db.Where("kindergarten_id=4").Order("id DESC").Limit(3).Find(&rows)
	for _,r:=range rows{fmt.Printf("  id=%d class=%d date=%s content=%s\n",r.ID,r.ClassID,r.Date.Format("2006-01-02"),r.Content[:min(30,len(r.Content))])}
}
