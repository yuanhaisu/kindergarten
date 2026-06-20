package main
import ("fmt";"log";"gorm.io/driver/mysql";"gorm.io/gorm";gormlogger "gorm.io/gorm/logger";"kindergarten-server/internal/config")
func main() {
	cfg,_:=config.Load("configs/app.ini")
	db,err:=gorm.Open(mysql.Open(cfg.MySQL.DSN),&gorm.Config{Logger:gormlogger.Default.LogMode(gormlogger.Warn)})
	if err!=nil{log.Fatal(err)}
	var users []struct{ID uint64;Phone string;OpenID string;Nickname string}
	db.Raw("SELECT id,phone,COALESCE(open_id,'') as open_id,nickname FROM users WHERE phone IN ('13900010001','13900010002')").Scan(&users)
	for _,u:=range users{fmt.Printf("User id=%d phone=%s openid=%q nickname=%s\n",u.ID,u.Phone,u.OpenID,u.Nickname)}
}
