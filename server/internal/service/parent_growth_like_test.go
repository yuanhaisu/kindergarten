package service

import (
	"context"
	"os"
	"path/filepath"
	"sync"
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

// cleanLikeFixture 必须 Unscoped 硬删：软删行会占唯一索引，重跑就会撞键。
func cleanLikeFixture(db *gorm.DB) {
	db.Unscoped().Where("log_id = ?", testLikeLog).Delete(&domain.GrowthLike{})
	db.Unscoped().Where("id = ?", testLikeLog).Delete(&domain.GrowthDailyLog{})
	db.Unscoped().Where("id = ?", testLikeClass).Delete(&domain.Class{})
	db.Unscoped().Where("id = ?", testLikeTeacher).Delete(&domain.User{})
	db.Unscoped().Where("id = ?", testLikeKG).Delete(&domain.Kindergarten{})
}

// setupLikeFixture 建一套独立 ID 的成长记录夹具，并注册清理。
func setupLikeFixture(t *testing.T) (*gorm.DB, *ParentPortalService, *ParentPortalContext) {
	t.Helper()
	db := openLikeTestDB(t)
	cleanLikeFixture(db)
	t.Cleanup(func() { cleanLikeFixture(db) })

	require.NoError(t, db.Create(&domain.User{
		BaseModel:    domain.BaseModel{ID: testLikeTeacher},
		Nickname:     "点赞回归测试老师",
		PasswordHash: "x",
	}).Error)

	require.NoError(t, db.Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: testLikeKG}, Name: "测试园所-点赞回归",
	}).Error)

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
	return db, svc, pc
}

func TestParentGrowthLike_RelikeAfterUnlikePersists(t *testing.T) {
	// TC-GRW-041 / TC-GRW-042 / TC-GRW-043 / TC-GRW-074
	db, svc, pc := setupLikeFixture(t)
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

// feedLiked 在 feed 响应里找指定记录，返回 is_liked 与是否命中。
func feedLiked(t *testing.T, feed map[string]any, logID uint64) (bool, bool) {
	t.Helper()
	list, ok := feed["list"].([]map[string]any)
	require.True(t, ok, "feed.list 类型应为 []map[string]any")
	for _, item := range list {
		if item["id"] == logID {
			v, _ := item["is_liked"].(bool)
			return v, true
		}
	}
	return false, false
}

// 回归：POST/DELETE 之后立刻读列表与详情，必须与库里的真实状态一致。
// 旧的 LikeGrowth 忽略 Create 错误时，POST 返回 true 而后续 GET 是 false。
func TestParentGrowthLike_FeedAndDetailReflectLikeState(t *testing.T) {
	// TC-GRW-040 / TC-GRW-042 / TC-GRW-085
	db, svc, pc := setupLikeFixture(t)
	ctx := context.Background()

	got, err := svc.LikeGrowth(ctx, pc, testLikeLog)
	require.NoError(t, err)
	assert.Equal(t, true, got["is_liked"])
	assert.EqualValues(t, 1, countLikeRows(t, db, false))

	feed, err := svc.ListGrowthFeed(ctx, pc, nil, 0, 0, 1, 20)
	require.NoError(t, err)
	liked, found := feedLiked(t, feed, testLikeLog)
	require.True(t, found, "点赞后记录应仍在 feed 中")
	assert.True(t, liked, "POST 后 GET feed 应为 is_liked=true")

	detail, err := svc.GrowthFeedDetail(ctx, pc, testLikeLog)
	require.NoError(t, err)
	assert.Equal(t, true, detail["is_liked"], "POST 后 GET detail 应为 is_liked=true")

	got, err = svc.UnlikeGrowth(ctx, pc, testLikeLog)
	require.NoError(t, err)
	assert.Equal(t, false, got["is_liked"])
	assert.EqualValues(t, 0, countLikeRows(t, db, false))

	// 重新进入页面 = 重新拉 feed/detail
	feed, err = svc.ListGrowthFeed(ctx, pc, nil, 0, 0, 1, 20)
	require.NoError(t, err)
	liked, found = feedLiked(t, feed, testLikeLog)
	require.True(t, found)
	assert.False(t, liked, "取消后 GET feed 应为 is_liked=false")

	detail, err = svc.GrowthFeedDetail(ctx, pc, testLikeLog)
	require.NoError(t, err)
	assert.Equal(t, false, detail["is_liked"], "取消后 GET detail 应为 is_liked=false")
}

// 回归：家长快速双击会并发发两个 POST。旧实现两个请求都查不到行、都去 Create，
// 唯一索引让其中一个返回 duplicate key —— 客户端看到「点赞失败」，尽管点赞已经成功。
func TestParentGrowthLike_ConcurrentDoubleClickIsIdempotent(t *testing.T) {
	// TC-GRW-041 / TC-GRW-074
	db, svc, pc := setupLikeFixture(t)
	ctx := context.Background()

	const clicks = 8
	start := make(chan struct{})
	errs := make(chan error, clicks)
	var wg sync.WaitGroup
	for i := 0; i < clicks; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start // 让所有请求尽量同时撞在同一窗口
			_, err := svc.LikeGrowth(ctx, pc, testLikeLog)
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		assert.NoError(t, err, "重复点赞必须幂等，不能返回错误")
	}
	assert.EqualValues(t, 1, countLikeRows(t, db, false), "并发点赞只能落一行")

	detail, err := svc.GrowthFeedDetail(ctx, pc, testLikeLog)
	require.NoError(t, err)
	assert.Equal(t, true, detail["is_liked"])
}
