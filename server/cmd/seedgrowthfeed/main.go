// seedgrowthfeed 灌入「家长端成长时光 feed」MCP 真实交互验证所需的最小数据集。
//
// 为什么单独一个命令，而不是往 cmd/seedgrowth 上堆：
// seedgrowth 是「造 2 条演示记录」，与「验证矩阵」不是一回事。本命令要覆盖
// TC-GRW-080~092 里的数据依赖（有图/无图、已赞/未赞、有评/零评、同月/跨月、
// 全班/个体、多孩切换），且必须幂等——seedgrowth 不幂等，重跑会翻倍、让月分组与
// 计数验证得出假结论。
//
// 目标账号：uid=11 / 手机号 13900010001 / 学生 4 / 班级 6 / 园所 4
// （摸底结论：uid 12 的孩子在 class 1，可见记录为 0，不能用它验证）
//
// 用法:
//
//	go run ./cmd/seedgrowthfeed           # 幂等灌入
//	go run ./cmd/seedgrowthfeed -reset    # 清除本命令造的数据
//
// 数据标记：记录正文以 R1| … R4| 开头，学生学号 SEED-S2，据此识别与清理。
package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
)

const (
	// 目标账号 = 小程序实际登录的那个用户。2026-10-05 实测：家长端每次
	// wx.login() 用的是当次 openid，与旧账号对不上时会新建 User。
	// 当前是 uid 65（openid oCJZt3fwWq…），其孩子 student 27/32 在 class 4。
	// 早先按 uid 11 / class 6 种的数据对该账号完全不可见（ListGrowthFeed
	// 按孩子的 class_id 过滤），页面会空列表。
	parentUserID    = uint64(65)
	existingStudent = uint64(27) // class 4
	// creator 用「李老师」而不是观看者本人：卡片要显示发布者是老师，
	// 用登录用户自己的话「发布者=观看者」，验不出 creator 渲染是否正确。
	creatorUserID  = uint64(13) // 李老师
	kindergartenID = uint64(4)
	classID        = uint64(4) // 学生 27 所在班级
	newStudentNo   = "SEED-S2"
)

// 8 月两条（同月）、9 月两条（跨月），日期固定而非相对 now，
// 否则换一天跑就变月份，月分组验证不可复现。
var seedDates = map[string]time.Time{
	"R1": time.Date(2026, 8, 10, 9, 30, 0, 0, time.Local),
	"R2": time.Date(2026, 8, 25, 15, 0, 0, 0, time.Local),
	"R3": time.Date(2026, 9, 5, 10, 15, 0, 0, time.Local),
	"R4": time.Date(2026, 9, 20, 16, 45, 0, 0, time.Local),
}

type seedSpec struct {
	mark     string
	target   int8 // 1 全班 2 个体
	content  string
	photos   int
	comments int
	liked    bool
	kids     []uint64 // target=2 时关联的学生
}

var specs = []seedSpec{
	{mark: "R1", target: 1, photos: 2, comments: 2, liked: true,
		content: "R1| 八月·全班·有图有评论已点赞：今天的美术课上，小朋友们画了春天的小树，每个孩子的作品都充满了想象力。"},
	{mark: "R2", target: 1, photos: 0, comments: 0, liked: false,
		content: "R2| 八月·全班·无图零评论未点赞：今天户外活动玩接力赛跑，孩子们跑得特别快，还主动为同伴加油，团队意识很强。"},
	{mark: "R3", target: 2, photos: 1, comments: 0, liked: true, kids: []uint64{existingStudent},
		content: "R3| 九月·个体·有图零评论已点赞：午睡醒来自己穿好了衣服和鞋子，独立性进步特别大，值得表扬。"},
	{mark: "R4", target: 2, photos: 0, comments: 1, liked: false, kids: []uint64{0}, // 0 留待 S2 创建后填
		content: "R4| 九月·个体·无图有评论未点赞：区角活动里主动把玩具分给了新来的小朋友，社交能力让人惊喜。"},
}

func main() {
	reset := flag.Bool("reset", false, "清除本命令造的验证数据")
	flag.Parse()

	cfg, err := config.Load("configs/app.ini")
	if err != nil {
		log.Fatalf("读取配置失败: %v", err)
	}
	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	if *reset {
		resetSeed(db)
		return
	}
	seed(db)
}

func resetSeed(db *gorm.DB) {
	// 只删带标记的数据，不碰真实记录。
	marks := []string{"R1|", "R2|", "R3|", "R4|"}
	var logs []domain.GrowthDailyLog
	if err := db.Where("kindergarten_id = ?", kindergartenID).Find(&logs).Error; err != nil {
		log.Fatalf("查询记录失败: %v", err)
	}
	ids := make([]uint64, 0, len(logs))
	for _, l := range logs {
		for _, m := range marks {
			if strings.HasPrefix(l.Content, m) {
				ids = append(ids, l.ID)
				break
			}
		}
	}
	if len(ids) > 0 {
		// 全部 Unscoped 硬删：这些表都带 BaseModel/SoftDeleteModel，软删会留残留行，
		// 重跑时 Count 把它们算进去，导致附件/评论数量与预期不符。
		for _, tbl := range []any{&domain.GrowthMedia{}, &domain.GrowthLike{}, &domain.GrowthComment{}, &domain.GrowthLogStudent{}} {
			if err := db.Unscoped().Where("log_id IN ?", ids).Delete(tbl).Error; err != nil {
				log.Fatalf("清理关联表失败: %v", err)
			}
		}
		if err := db.Unscoped().Where("id IN ?", ids).Delete(&domain.GrowthDailyLog{}).Error; err != nil {
			log.Fatalf("清理记录失败: %v", err)
		}
	}

	// 兜底：清掉指向已删记录的孤儿关联。只可能由「先软删关联、再硬删记录」的旧清理
	// 路径产生（外键在 growth_media.log_id 上并不存在，所以这类残留不会被 DB 拦住）。
	for _, tbl := range []any{&domain.GrowthMedia{}, &domain.GrowthLike{}, &domain.GrowthComment{}, &domain.GrowthLogStudent{}} {
		if err := db.Unscoped().Where("log_id IS NOT NULL AND log_id NOT IN (SELECT id FROM growth_daily_logs)").
			Delete(tbl).Error; err != nil {
			log.Fatalf("清理孤儿关联失败: %v", err)
		}
	}
	// 只删自己造的那个学生及其家长关联；不要顺手清「所有孤儿关联」——
	// 那会误删别的历史遗留数据，超出本命令的职责。
	var seedStudent domain.Student
	if err := db.Unscoped().Where("student_no = ?", newStudentNo).First(&seedStudent).Error; err == nil {
		// 必须 Unscoped 硬删：StudentParent 走 BaseModel 软删，行留在表里，
		// students 的 ON DELETE RESTRICT 外键仍会拦住删除。
		if err := db.Unscoped().Where("student_id = ?", seedStudent.ID).Delete(&domain.StudentParent{}).Error; err != nil {
			log.Fatalf("清理种子学生关联失败: %v", err)
		}
		if err := db.Unscoped().Where("id = ?", seedStudent.ID).Delete(&domain.Student{}).Error; err != nil {
			log.Fatalf("清理种子学生失败: %v", err)
		}
	}

	fmt.Printf("已清理种子记录 %d 条及关联数据。\n", len(ids))
}

func seed(db *gorm.DB) {
	// 1) 家长昵称：uid 11 的 nickname 在库里是 8 个 U+FFFD（双编码损坏，原文不可恢复）。
	//    不修的话卡片显示乱码，会被误判成渲染 bug。
	var parent domain.User
	if err := db.First(&parent, parentUserID).Error; err != nil {
		log.Fatalf("找不到家长 uid=%d（请确认库是 kindergarten 且已有种子账号）: %v", parentUserID, err)
	}
	if strings.ContainsRune(parent.Nickname, '�') || parent.Nickname == "" {
		if err := db.Model(&domain.User{}).Where("id = ?", parentUserID).
			Update("nickname", "王小明妈妈").Error; err != nil {
			log.Fatalf("修复家长昵称失败: %v", err)
		}
		fmt.Printf("✓ 修复家长 uid=%d 昵称（原为损坏字符/空）→ 王小明妈妈\n", parentUserID)
	} else {
		fmt.Printf("✓ 家长昵称正常：%s\n", parent.Nickname)
	}

	// 2) 多孩：造第二个孩子并绑给 uid 11，用于「切换孩子」验证。
	s2 := ensureStudent(db)
	if err := ensureStudentParent(db, parentUserID, s2.ID, domain.RelationMother); err != nil {
		log.Fatalf("绑定家长-学生失败: %v", err)
	}
	fmt.Printf("✓ 多孩就绪：uid=%d 关联学生 [%d, %d]\n", parentUserID, existingStudent, s2.ID)

	// 3) 四条记录 + 附件/点赞/评论/学生关联
	for i := range specs {
		sp := &specs[i]
		if sp.mark == "R4" {
			sp.kids = []uint64{existingStudent, s2.ID}
		}
		logRow := ensureLog(db, sp)
		ensureMedia(db, logRow, sp)
		ensureLike(db, logRow, sp)
		ensureComments(db, logRow, sp)
		ensureLogStudents(db, logRow, sp)
		fmt.Printf("✓ %s id=%d date=%s target=%d media=%d comments=%d liked=%v kids=%v\n",
			sp.mark, logRow.ID, logRow.Date.Format("2006-01-02"), sp.target, sp.photos, sp.comments, sp.liked, sp.kids)
	}

	printVisibility(db, s2.ID)
}

func ensureStudent(db *gorm.DB) domain.Student {
	var s domain.Student
	err := db.Where("student_no = ?", newStudentNo).First(&s).Error
	if err == nil {
		return s
	}
	if err != gorm.ErrRecordNotFound {
		log.Fatalf("查询种子学生失败: %v", err)
	}
	s = domain.Student{
		KindergartenID: kindergartenID,
		ClassID:        classID,
		Name:           "王小花",
		Gender:         2,
		// birthday 不能留 Go 零值：MySQL 严格模式会拒绝 '0000-00-00'
		Birthday:   time.Date(2021, 6, 1, 0, 0, 0, 0, time.Local),
		StudentNo:  newStudentNo,
		Status:     1,
		EnrollDate: time.Date(2026, 3, 1, 0, 0, 0, 0, time.Local),
	}
	if err := db.Create(&s).Error; err != nil {
		log.Fatalf("创建种子学生失败: %v", err)
	}
	return s
}

func ensureStudentParent(db *gorm.DB, userID, studentID uint64, rel domain.ParentRelation) error {
	var n int64
	db.Model(&domain.StudentParent{}).
		Where("user_id = ? AND student_id = ?", userID, studentID).Count(&n)
	if n > 0 {
		return nil
	}
	return db.Create(&domain.StudentParent{
		UserID: userID, StudentID: studentID, Relation: rel, IsPrimary: studentID == existingStudent,
	}).Error
}

func ensureLog(db *gorm.DB, sp *seedSpec) domain.GrowthDailyLog {
	var row domain.GrowthDailyLog
	err := db.Where("kindergarten_id = ? AND content LIKE ?", kindergartenID, sp.mark+"|%").
		First(&row).Error
	if err == nil {
		return row
	}
	if err != gorm.ErrRecordNotFound {
		log.Fatalf("查询 %s 失败: %v", sp.mark, err)
	}
	row = domain.GrowthDailyLog{
		KindergartenID: kindergartenID,
		ClassID:        classID,
		CreatorID:      creatorUserID,
		Date:           seedDates[sp.mark],
		Content:        sp.content,
		SyncToFeed:     true,
		TargetType:     sp.target,
	}
	if err := db.Create(&row).Error; err != nil {
		log.Fatalf("创建 %s 失败: %v", sp.mark, err)
	}
	// created_at 被 GORM 置为当前时间，验证「发布时刻」时分要看它；
	// 显式对齐到 seedDates，避免「日期是 8 月、时间却是 10 月」的矛盾显示。
	if err := db.Model(&row).Update("created_at", seedDates[sp.mark]).Error; err != nil {
		log.Fatalf("对齐 %s 的 created_at 失败: %v", sp.mark, err)
	}
	db.First(&row, row.ID)
	return row
}

func ensureMedia(db *gorm.DB, logRow domain.GrowthDailyLog, sp *seedSpec) {
	var n int64
	db.Model(&domain.GrowthMedia{}).Where("log_id = ?", logRow.ID).Count(&n)
	if int(n) >= sp.photos {
		return
	}
	logID := logRow.ID
	for i := int(n); i < sp.photos; i++ {
		m := domain.GrowthMedia{
			KindergartenID: kindergartenID,
			ClassID:        classID,
			LogID:          &logID,
			UploaderID:     creatorUserID,
			// 占位图即可：验证的是「是否渲染缩略图」，不需要真 OSS 对象
			FileURL:  fmt.Sprintf("https://placehold.co/400x400/png?text=%s-%d", sp.mark, i+1),
			ThumbURL: fmt.Sprintf("https://placehold.co/200x200/png?text=%s-%d", sp.mark, i+1),
			FileType: "image",
			FileName: fmt.Sprintf("%s-%d.jpg", sp.mark, i+1),
			FileSize: 1024,
			SortNum:  i + 1,
		}
		if err := db.Create(&m).Error; err != nil {
			log.Fatalf("创建 %s 的媒体失败: %v", sp.mark, err)
		}
	}
}

func ensureLike(db *gorm.DB, logRow domain.GrowthDailyLog, sp *seedSpec) {
	if !sp.liked {
		return
	}
	var n int64
	db.Model(&domain.GrowthLike{}).
		Where("log_id = ? AND user_id = ?", logRow.ID, parentUserID).Count(&n)
	if n > 0 {
		return
	}
	if err := db.Create(&domain.GrowthLike{LogID: logRow.ID, UserID: parentUserID}).Error; err != nil {
		log.Fatalf("创建 %s 的点赞失败: %v", sp.mark, err)
	}
}

func ensureComments(db *gorm.DB, logRow domain.GrowthDailyLog, sp *seedSpec) {
	var n int64
	db.Model(&domain.GrowthComment{}).Where("log_id = ?", logRow.ID).Count(&n)
	for i := int(n); i < sp.comments; i++ {
		c := domain.GrowthComment{
			LogID:   logRow.ID,
			UserID:  parentUserID,
			Content: fmt.Sprintf("谢谢老师的记录，孩子回家特别开心！（%s #%d）", sp.mark, i+1),
		}
		if err := db.Create(&c).Error; err != nil {
			log.Fatalf("创建 %s 的评论失败: %v", sp.mark, err)
		}
	}
}

func ensureLogStudents(db *gorm.DB, logRow domain.GrowthDailyLog, sp *seedSpec) {
	if sp.target != 2 {
		return // 全班记录不挂个体学生，否则 TC-091 测不了
	}
	for _, sid := range sp.kids {
		var n int64
		db.Model(&domain.GrowthLogStudent{}).
			Where("log_id = ? AND student_id = ?", logRow.ID, sid).Count(&n)
		if n > 0 {
			continue
		}
		if err := db.Create(&domain.GrowthLogStudent{LogID: logRow.ID, StudentID: sid}).Error; err != nil {
			log.Fatalf("创建 %s 的学生关联失败: %v", sp.mark, err)
		}
	}
}

// printVisibility 用 ListGrowthFeed 的可见性规则算一遍，把「家长登录后会看到什么」
// 直接打出来——避免数据没准备好就开始验证，把空列表误判成代码问题。
func printVisibility(db *gorm.DB, s2ID uint64) {
	fmt.Printf("\n=== 家长 uid=%d 登录后可见的成长时光（按 ListGrowthFeed 规则） ===\n", parentUserID)
	for label, sid := range map[string]uint64{
		fmt.Sprintf("已有孩子 %d", existingStudent): existingStudent,
		"王小花（新造）": s2ID,
	} {
		var rows []domain.GrowthDailyLog
		db.Where("growth_daily_logs.kindergarten_id = ? AND growth_daily_logs.class_id = ? AND growth_daily_logs.deleted_at IS NULL",
			kindergartenID, classID).
			Where("growth_daily_logs.target_type = 1 OR growth_daily_logs.id IN (SELECT log_id FROM growth_log_students WHERE student_id = ?)", sid).
			Order("growth_daily_logs.date DESC, growth_daily_logs.id DESC").
			Find(&rows)
		fmt.Printf("\n【%s】共 %d 条\n", label, len(rows))
		for _, r := range rows {
			var media, likes, comments int64
			db.Model(&domain.GrowthMedia{}).Where("log_id = ?", r.ID).Count(&media)
			db.Model(&domain.GrowthLike{}).Where("log_id = ? AND user_id = ?", r.ID, parentUserID).Count(&likes)
			db.Model(&domain.GrowthComment{}).Where("log_id = ?", r.ID).Count(&comments)
			mark := r.Content
			if i := strings.Index(mark, "|"); i > 0 {
				mark = mark[:i]
			}
			fmt.Printf("  %s date=%s target=%d media=%d comments=%d is_liked=%v\n",
				mark, r.Date.Format("2006-01-02"), r.TargetType, media, comments, likes > 0)
		}
	}
	fmt.Println("\n覆盖矩阵对照（TC-GRW-080~092）:")
	fmt.Println("  同月分组 ← R1+R2（都在 2026-08）    跨月分组 ← 8 月 vs 9 月")
	fmt.Println("  有图/无图 ← R1,R3 有 / R2,R4 无     点赞两态 ← R1,R3 已赞 / R2,R4 未赞")
	fmt.Println("  评论两态 ← R1=2,R4=1 / R2,R3=0     全班/个体 ← R1,R2 / R3,R4")
	fmt.Println("  详情相关孩子 ← R3→[4] R4→[4,王小花]  切换孩子 ← uid11 有两个孩子")
	fmt.Println("\n⚠️  登录前置仍未解决：uid=11 目前没有 openid。")
	fmt.Println("    需先用家长微信登录一次产生 openid，再跑 go run ./cmd/openid 绑定，")
	fmt.Println("    并用 go run ./cmd/linkwx 核对（一个 openid 只能属一个 user，否则 403）。")
}
