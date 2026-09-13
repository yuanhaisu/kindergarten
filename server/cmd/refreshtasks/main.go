package main
import ("fmt";"time";"gorm.io/driver/mysql";"gorm.io/gorm";gormlogger "gorm.io/gorm/logger";"kindergarten-server/internal/config";"kindergarten-server/internal/domain")
func main() {
	cfg,_:=config.Load("configs/app.ini")
	db,err:=gorm.Open(mysql.Open(cfg.MySQL.DSN),&gorm.Config{Logger:gormlogger.Default.LogMode(gormlogger.Warn)})
	if err!=nil{panic(err)}
	// 删除现所有任务
	db.Where("kindergarten_id=4").Delete(&domain.ParentTask{})
	db.Where("kindergarten_id=4").Delete(&domain.TaskCheckIn{})
	fmt.Println("Deleted old tasks")
	// 重建清晰中文任务
	today:=time.Now()
	tasks:=[]domain.ParentTask{
		{KindergartenID:4,ClassID:6,CreatorID:4,Title:"每日亲子阅读",Description:"每天陪伴孩子阅读15分钟，培养良好的阅读习惯，完成后拍照上传记录",Points:5,Frequency:"daily",StartDate:today,EndDate:today.AddDate(0,0,14),Status:1},
		{KindergartenID:4,ClassID:6,CreatorID:4,Title:"家务小能手",Description:"鼓励孩子参与力所能及的家务劳动（整理玩具、叠衣服等），培养独立能力和责任感",Points:3,Frequency:"daily",StartDate:today,EndDate:today.AddDate(0,0,14),Status:1},
		{KindergartenID:4,ClassID:6,CreatorID:4,Title:"户外运动挑战",Description:"每天户外运动30分钟（跑步、跳绳、球类等），强健体魄，记录运动瞬间",Points:10,Frequency:"daily",StartDate:today,EndDate:today.AddDate(0,0,7),Status:1},
		{KindergartenID:4,ClassID:1,CreatorID:4,Title:"亲子阅读打卡",Description:"每天陪孩子读一本绘本，培养阅读兴趣",Points:5,Frequency:"daily",StartDate:today,EndDate:today.AddDate(0,0,14),Status:1},
	}
	for _,t:=range tasks{db.Create(&t);fmt.Printf("Created: id=%d title=%s\n",t.ID,t.Title)}
}
