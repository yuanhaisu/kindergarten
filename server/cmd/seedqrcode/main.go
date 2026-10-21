// seedqrcode — 补全接送二维码链路所需的种子数据。
//
// 创建：家长用户 → 学生 + 绑定 → 有效授权接送人 → 今日指定接走人。
// 幂等：已存在则跳过。
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

const (
	seedKGID      uint64 = 4
	seedPhone            = "13900001020"
	seedParentID  uint64 = 40
	seedStudentID uint64 = 21
)

func main() {
	cfg, err := config.Load("configs/app.ini")
	if err != nil {
		log.Fatal(err)
	}
	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	now := time.Now()

	// 1. 确保家长用户存在
	var user domain.User
	if err := db.Where("id = ?", seedParentID).First(&user).Error; err != nil {
		user = domain.User{
			BaseModel: domain.BaseModel{ID: seedParentID},
			Phone:     seedPhone,
			Nickname:  "周嘉嘉妈妈",
		}
		if err := db.Create(&user).Error; err != nil {
			log.Fatal("create user:", err)
		}
		fmt.Printf("created parent user id=%d phone=%s\n", seedParentID, seedPhone)
	} else {
		fmt.Printf("parent user exists id=%d phone=%s\n", user.ID, user.Phone)
	}

	// 2. 确保学生存在
	var student domain.Student
	if err := db.Where("id = ?", seedStudentID).First(&student).Error; err != nil {
		student = domain.Student{
			SoftDeleteModel: domain.SoftDeleteModel{
				BaseModel: domain.BaseModel{ID: seedStudentID},
			},
			KindergartenID: seedKGID,
			ClassID:        5,
			Name:           "周嘉嘉",
		}
		if err := db.Create(&student).Error; err != nil {
			log.Fatal("create student:", err)
		}
		fmt.Printf("created student id=%d name=%s\n", seedStudentID, student.Name)
	} else {
		fmt.Printf("student exists id=%d name=%s\n", student.ID, student.Name)
	}

	// 3. 绑定家长-学生
	var bindCount int64
	db.Model(&domain.StudentParent{}).
		Where("student_id = ? AND user_id = ?", seedStudentID, seedParentID).
		Count(&bindCount)
	if bindCount == 0 {
		sp := domain.StudentParent{
			StudentID: seedStudentID,
			UserID:    seedParentID,
			Relation:  domain.RelationMother,
			IsPrimary: true,
		}
		if err := db.Create(&sp).Error; err != nil {
			log.Fatal("create student_parent:", err)
		}
		fmt.Println("created student-parent binding")
	} else {
		fmt.Println("student-parent binding exists")
	}

	// 4. 确保至少一个有效授权接送人
	repo := repository.NewPickupRepo(db)
	var auth domain.PickupAuthorization
	err = db.Where("student_id = ? AND status = 1 AND review_status = 1 AND deleted_at IS NULL", seedStudentID).
		First(&auth).Error
	if err != nil {
		expire := now.Add(30 * 24 * time.Hour)
		auth = domain.PickupAuthorization{
			StudentID:      seedStudentID,
			Name:           "奶奶（张三）",
			Phone:          "13800008888",
			Relation:       "grandmother",
			Status:         1,
			ReviewStatus:   1,
			AuthType:       "permanent",
			AllowPickup:    true,
			NotifyOnPickup: true,
			ExpireAt:       &expire,
		}
		if err := repo.CreateAuthorization(ctx, &auth); err != nil {
			log.Fatal("create authorization:", err)
		}
		fmt.Printf("created authorization id=%d name=%s\n", auth.ID, auth.Name)
	} else {
		fmt.Printf("authorization exists id=%d name=%s\n", auth.ID, auth.Name)
	}

	// 5. 创建今日指定
	if err := repo.UpsertDailyDesignation(ctx, seedKGID, seedStudentID, auth.ID, now); err != nil {
		log.Fatal("create designation:", err)
	}
	fmt.Printf("designation set student=%d auth=%d date=%s\n", seedStudentID, auth.ID, now.Format("2006-01-02"))

	fmt.Println("\n=== 链路数据就绪 ===")
	fmt.Println("家长: id=40 phone=13900001020")
	fmt.Println("学生: id=21 周嘉嘉")
	fmt.Println("授权人: 已指定为今日接走人")
	fmt.Println("下一步: 打开家长端「接送二维码」页即可看到二维码")
}
