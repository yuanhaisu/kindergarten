package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// 回归背景（2026-10-17）：growth_likes 是软删除模型，但唯一索引 idx_log_user(log_id, user_id)
// 不含 deleted_at —— 取消点赞留下的软删行会占住唯一键，下一次 Create 撞键失败。当时
// LikeGrowth 又忽略了 Create 的错误，于是接口返回 is_liked:true、库里却查不到，
// 用户看到「点了又没了」。
//
// 断言的是**库里的真实状态**，不是接口返回值 —— 返回值在 bug 存在时同样是 true。

const (
	testLikeKG      uint64 = 990231
	testLikeClass   uint64 = 990232
	testLikeLog     uint64 = 990233
	testLikeStudent uint64 = 990234
	testLikeParent  uint64 = 990235
	testLikeTeacher uint64 = 990236
)

func openLikeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	cfgPath := filepath.Join("..", "..", "configs", "app.ini")
	if _, err := os.Stat(cfgPath); err != nil {
		t.Skipf("跳过：找不到 %s（本地无 DB 配置）", cfgPath)
	}
	cfg, err := config.Load(cfgPath)
	require.NoError(t, err)
	if cfg.MySQL.DSN == "" {
		t.Skip("跳过：MySQL DSN 为空")
	}
	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	return db
}

// countLikeRows 统计测试家长在该记录上的点赞行；unscoped=true 时把软删行也算进来。
func countLikeRows(t *testing.T, db *gorm.DB, unscoped bool) int64 {
	t.Helper()
	var n int64
	q := db
	if unscoped {
		q = db.Unscoped()
	}
	require.NoError(t, q.Model(&domain.GrowthLike{}).
		Where("log_id = ? AND user_id = ?", testLikeLog, testLikeParent).Count(&n).Error)
	return n
}

func TestParentGrowthLike_RelikeAfterUnlikePersists(t *testing.T) {
	db := openLikeTestDB(t)

	// 夹具清理必须 Unscoped 硬删：软删行会占唯一索引，重跑就会撞键。
	t.Cleanup(func() {
		db.Unscoped().Where("log_id = ?", testLikeLog).Delete(&domain.GrowthLike{})
		db.Unscoped().Where("id = ?", testLikeLog).Delete(&domain.GrowthDailyLog{})
		db.Unscoped().Where("id = ?", testLikeClass).Delete(&domain.Class{})
		db.Unscoped().Where("id = ?", testLikeTeacher).Delete(&domain.User{})
	})
	db.Unscoped().Where("log_id = ?", testLikeLog).Delete(&domain.GrowthLike{})
	db.Unscoped().Where("id = ?", testLikeLog).Delete(&domain.GrowthDailyLog{})
	db.Unscoped().Where("id = ?", testLikeClass).Delete(&domain.Class{})
	db.Unscoped().Where("id = ?", testLikeTeacher).Delete(&domain.User{})

	require.NoError(t, db.Create(&domain.User{
		BaseModel:    domain.BaseModel{ID: testLikeTeacher},
		Nickname:     "点赞回归测试老师",
		Phone:        "13900002999",
		PasswordHash: "x",
	}).Error)

	var kgCount int64
	require.NoError(t, db.Model(&domain.Kindergarten{}).Where("id = ?", testLikeKG).Count(&kgCount).Error)
	if kgCount == 0 {
		require.NoError(t, db.Create(&domain.Kindergarten{
			BaseModel: domain.BaseModel{ID: testLikeKG}, Name: "测试园所-点赞回归",
		}).Error)
	}

	require.NoError(t, db.Create(&domain.Class{
		BaseModel:      domain.BaseModel{ID: testLikeClass},
		KindergartenID: testLikeKG,
		Name:           "点赞回归测试班",
		Grade:          "小班",
		Status:         1,
	}).Error)

	require.NoError(t, db.Create(&domain.GrowthDailyLog{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testLikeLog}},
		KindergartenID:  testLikeKG,
		ClassID:         testLikeClass,
		CreatorID:       testLikeTeacher,
		Date:            time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC),
		TargetType:      1, // 全班
		Content:         "like regression fixture",
	}).Error)

	svc := NewParentPortalService(db,
		repository.NewParentRepo(db), repository.NewUserRoleRepo(db),
		repository.NewStudentRepo(db), repository.NewClassRepo(db),
		repository.NewPickupRepo(db), repository.NewCalendarRepo(db),
		repository.NewNoticeRepo(db))

	pc := &ParentPortalContext{
		UserID:         testLikeParent,
		KindergartenID: testLikeKG,
		Students: []domain.Student{{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testLikeStudent}},
			ClassID:         testLikeClass,
		}},
	}
	ctx := context.Background()

	// 1. 首次点赞
	got, err := svc.LikeGrowth(ctx, pc, testLikeLog)
	require.NoError(t, err)
	assert.Equal(t, true, got["is_liked"])
	assert.EqualValues(t, 1, countLikeRows(t, db, false), "首次点赞后应有未删除的点赞行")

	// 2. 取消点赞
	got, err = svc.UnlikeGrowth(ctx, pc, testLikeLog)
	require.NoError(t, err)
	assert.Equal(t, false, got["is_liked"])
	assert.EqualValues(t, 0, countLikeRows(t, db, false), "取消点赞后不应有未删除的点赞行")

	// 3. 再次点赞 —— 旧实现会撞上软删行占用的唯一索引并静默失败
	got, err = svc.LikeGrowth(ctx, pc, testLikeLog)
	require.NoError(t, err)
	assert.Equal(t, true, got["is_liked"])
	assert.EqualValues(t, 1, countLikeRows(t, db, false),
		"再次点赞必须真正落库（旧实现撞软删行占用的唯一索引，且 Create 错误被忽略）")
}
