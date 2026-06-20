package main
import ("fmt";"log";"gorm.io/driver/mysql";"gorm.io/gorm";gormlogger "gorm.io/gorm/logger";"kindergarten-server/internal/config")
func main() {
	cfg,_:=config.Load("configs/app.ini")
	db,err:=gorm.Open(mysql.Open(cfg.MySQL.DSN),&gorm.Config{Logger:gormlogger.Default.LogMode(gormlogger.Warn)})
	if err!=nil{log.Fatal(err)}
	var users []struct{ID uint64;Phone string;OpenID string;Nickname string}
	db.Raw("SELECT id,phone,COALESCE(open_id,'') as open_id,nickname FROM users ORDER BY id DESC LIMIT 5").Scan(&users)
	for _,u:=range users{fmt.Printf("id=%d phone=%s openid=%q nickname=%s\n",u.ID,u.Phone,u.OpenID,u.Nickname)}
	// Bind latest OpenID to parent users
	var latestOpenID string
	db.Raw("SELECT COALESCE(open_id,'') FROM users WHERE open_id != '' ORDER BY id DESC LIMIT 1").Scan(&latestOpenID)
	if latestOpenID!=""{
		db.Exec("UPDATE users SET open_id=? WHERE phone IN ('13900010001','13900010002') AND open_id=''",latestOpenID)
		fmt.Printf("\nBound OpenID %q to parent users\n",latestOpenID)
	}
}
