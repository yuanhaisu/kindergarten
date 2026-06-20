package main
import ("fmt";"log";"time";"gorm.io/driver/mysql";"gorm.io/gorm";gormlogger "gorm.io/gorm/logger";"kindergarten-server/internal/config";"kindergarten-server/internal/domain")
func main() {
	cfg,_:=config.Load("configs/app.ini")
	db,err:=gorm.Open(mysql.Open(cfg.MySQL.DSN),&gorm.Config{Logger:gormlogger.Default.LogMode(gormlogger.Warn)})
	if err!=nil{log.Fatal(err)}
	now:=time.Now()
	logs:=[]domain.GrowthDailyLog{
		{KindergartenID:4,ClassID:6,CreatorID:61,Date:now,Content:"今天大二班的户外运动课，小朋友们在操场上进行了接力赛跑比赛，刘晨晨跑得特别快，获得了全班第一名！大家都很开心~",SyncToFeed:true,TargetType:1},
		{KindergartenID:4,ClassID:6,CreatorID:61,Date:now.AddDate(0,0,-1),Content:"今天的美术课上，大二班的小朋友们学习了画春天的小树，每个孩子的作品都充满了想象力",SyncToFeed:true,TargetType:1},
	}
	for _,l:=range logs{
		db.Create(&l)
		fmt.Printf("Created growth log id=%d date=%s\n",l.ID,l.Date.Format("2006-01-02"))
	}
}
