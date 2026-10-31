package service

// 老师端成长记录点赞的幂等回归（W8-B）。
//
// 背景：家长端 LikeGrowth 早已对 Create 加 clause.OnConflict{DoNothing}，老师端漏了。
// 并发双击时两个请求都先 Unscoped 查不到行、都去 Create，唯一索引 idx_log_user 会让
// 其中一个返回 1062 duplicate key —— 客户端看到「点赞失败」，尽管点赞已经成功。
//
// 断言的是库里的真实状态（行数），不是接口返回值。

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
)

// 独立 ID 段，避开已有夹具（9902xx-9905xx、991xxxxxx、992000000+）。
const (
	testTGLKG      uint64 = 990601
	testTGLClass   uint64 = 990602
	testTGLTeacher uint64 = 990603
	testTGLLog     uint64 = 990604
)

var (
	testTGLUserIDs  = []uint64{testTGLTeacher}
	testTGLLogIDs   = []uint64{testTGLLog}
	testTGLClassIDs = []uint64{testTGLClass}
)

// cleanTeacherLikeFixture 必须 Unscoped 硬删：软删行会占住 idx_log_user 唯一键，重跑撞键。
// 打印实际删除行数作为清理证据。
func cleanTeacherLikeFixture(t *testing.T, db *gorm.DB) {
	t.Helper()
	del := func(model any, where string, args ...any) int64 {
		res := db.Unscoped().Where(where, args...).Delete(model)
		require.NoError(t, res.Error)
		return res.RowsAffected
	}
	likeRows := del(&domain.GrowthLike{}, "log_id IN ?", testTGLLogIDs) +
		del(&domain.GrowthLike{}, "user_id IN ?", testTGLUserIDs)
	logRows := del(&domain.GrowthDailyLog{}, "id IN ?", testTGLLogIDs)
	classRows := del(&domain.Class{}, "id IN ?", testTGLClassIDs)
	userRows := del(&domain.User{}, "id IN ?", testTGLUserIDs)
	kgRows := del(&domain.Kindergarten{}, "id = ?", testTGLKG)
	t.Logf("teacher-like cleanup: growth_likes=%d logs=%d classes=%d users=%d kindergartens=%d",
		likeRows, logRows, classRows, userRows, kgRows)
}

type teacherLikeFixture struct {
	db  *gorm.DB
	svc *TeacherPortalService
	tc  *TeacherPortalContext
}

func setupTeacherLikeFixture(t *testing.T) *teacherLikeFixture {
	t.Helper()
	db := openHomeFeedTestDB(t)
	cleanTeacherLikeFixture(t, db)
	t.Cleanup(func() { cleanTeacherLikeFixture(t, db) })

	require.NoError(t, db.Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: testTGLKG}, Name: "测试园所-老师点赞",
	}).Error)
	require.NoError(t, db.Create(&domain.Class{
		BaseModel:      domain.BaseModel{ID: testTGLClass},
		KindergartenID: testTGLKG,
		Name:           "老师点赞测试班",
		Grade:          "小班",
		Status:         1,
	}).Error)
	require.NoError(t, db.Create(&domain.User{
		BaseModel:    domain.BaseModel{ID: testTGLTeacher},
		Nickname:     "点赞测试老师",
		PasswordHash: "x",
	}).Error)
	require.NoError(t, db.Create(&domain.GrowthDailyLog{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testTGLLog}},
		KindergartenID:  testTGLKG,
		ClassID:         testTGLClass,
		CreatorID:       testTGLTeacher,
		Date:            time.Date(2026, 10, 31, 0, 0, 0, 0, time.Local),
		TargetType:      1, // 全班
		Content:         "teacher like idempotency fixture",
	}).Error)

	svc := NewTeacherPortalService(db, nil, nil, nil, nil, nil, config.FaceConfig{})
	return &teacherLikeFixture{
		db:  db,
		svc: svc,
		tc:  &TeacherPortalContext{UserID: testTGLTeacher, KindergartenID: testTGLKG, ClassID: testTGLClass},
	}
}

// countTeacherLikeRows 统计该记录上该老师的点赞行；unscoped=true 时把软删行也算进来。
func countTeacherLikeRows(t *testing.T, db *gorm.DB, unscoped bool) int64 {
	t.Helper()
	var n int64
	q := db
	if unscoped {
		q = db.Unscoped()
	}
	require.NoError(t, q.Model(&domain.GrowthLike{}).
		Where("log_id IN ? AND user_id IN ?", testTGLLogIDs, testTGLUserIDs).Count(&n).Error)
	return n
}

// 顺序重复点赞必须幂等：两次都成功、计数一致、库里只有一行。
func TestTeacherGrowthLike_RepeatLikeIsIdempotent(t *testing.T) {
	f := setupTeacherLikeFixture(t)
	ctx := context.Background()

	first, err := f.svc.LikeGrowth(ctx, f.tc, testTGLLog)
	require.NoError(t, err)
	assert.Equal(t, true, first["is_liked"])
	assert.EqualValues(t, 1, first["likes_count"])

	second, err := f.svc.LikeGrowth(ctx, f.tc, testTGLLog)
	require.NoError(t, err)
	assert.Equal(t, true, second["is_liked"])
	assert.EqualValues(t, 1, second["likes_count"], "重复点赞不得重复计数")
	assert.EqualValues(t, 1, countTeacherLikeRows(t, f.db, false), "重复点赞只允许一行")
}

// 回归 W8-B：老师快速双击并发发两个 POST。旧实现两个请求都查不到行、都 Create，
// 唯一索引让其中一个返回 duplicate key —— 客户端看到「点赞失败」。
func TestTeacherGrowthLike_ConcurrentDoubleClickIsIdempotent(t *testing.T) {
	f := setupTeacherLikeFixture(t)
	ctx := context.Background()

	const clicks = 8
	start := make(chan struct{})
	errs := make(chan error, clicks)
	counts := make(chan int64, clicks)
	var wg sync.WaitGroup
	for i := 0; i < clicks; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			data, err := f.svc.LikeGrowth(ctx, f.tc, testTGLLog)
			if err == nil {
				if n, ok := data["likes_count"].(int64); ok {
					counts <- n
				}
			}
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	close(counts)

	for err := range errs {
		assert.NoError(t, err, "并发重复点赞必须幂等，不能返回错误")
	}
	for n := range counts {
		assert.EqualValues(t, 1, n, "每次响应都应看到同一条点赞")
	}
	assert.EqualValues(t, 1, countTeacherLikeRows(t, f.db, false), "并发点赞只能落一行")
	assert.EqualValues(t, 1, countTeacherLikeRows(t, f.db, true), "不得留下软删行")
}

// 软删复位路径仍要保留：取消（硬删）后再点赞必须真正落库。
func TestTeacherGrowthLike_RelikeAfterUnlikePersists(t *testing.T) {
	f := setupTeacherLikeFixture(t)
	ctx := context.Background()

	_, err := f.svc.LikeGrowth(ctx, f.tc, testTGLLog)
	require.NoError(t, err)
	_, err = f.svc.UnlikeGrowth(ctx, f.tc, testTGLLog)
	require.NoError(t, err)
	assert.EqualValues(t, 0, countTeacherLikeRows(t, f.db, false))

	data, err := f.svc.LikeGrowth(ctx, f.tc, testTGLLog)
	require.NoError(t, err)
	assert.Equal(t, true, data["is_liked"])
	assert.EqualValues(t, 1, data["likes_count"])
	assert.EqualValues(t, 1, countTeacherLikeRows(t, f.db, false), "再次点赞必须真正落库")
}

// 清理证据：夹具硬删后（含软删行）库内 0 残留。
func TestTeacherGrowthLike_FixtureCleanupLeavesNoResidue(t *testing.T) {
	f := setupTeacherLikeFixture(t)
	cleanTeacherLikeFixture(t, f.db)

	count := func(model any, where string, args ...any) int64 {
		var n int64
		require.NoError(t, f.db.Unscoped().Model(model).Where(where, args...).Count(&n).Error)
		return n
	}
	assert.Zero(t, count(&domain.GrowthLike{}, "log_id IN ?", testTGLLogIDs))
	assert.Zero(t, count(&domain.GrowthLike{}, "user_id IN ?", testTGLUserIDs))
	assert.Zero(t, count(&domain.GrowthDailyLog{}, "id IN ?", testTGLLogIDs))
	assert.Zero(t, count(&domain.Class{}, "id IN ?", testTGLClassIDs))
	assert.Zero(t, count(&domain.User{}, "id IN ?", testTGLUserIDs))
	assert.Zero(t, count(&domain.Kindergarten{}, "id = ?", testTGLKG))
}
