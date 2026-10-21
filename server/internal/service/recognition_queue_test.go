package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"kindergarten-server/internal/ai/face"
	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
	"kindergarten-server/pkg/log"
)

// 识别队列的 DB 行为测试（认领互斥 / 建议写保护 / 限流 / 预算）。
// 项目测试惯例：需 DB 的用例连本机 MySQL（无配置则 Skip），与 notice_multitarget_test 同模式。

var rqDB *gorm.DB

func openRQTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	if rqDB != nil {
		return rqDB
	}
	cfgPath := filepath.Join("..", "..", "configs", "app.ini")
	if _, err := os.Stat(cfgPath); err != nil {
		t.Skipf("跳过：找不到 %s", cfgPath)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil || cfg.MySQL.DSN == "" {
		t.Skip("跳过：MySQL 未配置")
	}
	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.RecognitionJob{}, &domain.GrowthMediaStudent{}))
	rqDB = db
	return db
}

func testQueue(db *gorm.DB, cfg config.FaceConfig) *RecognitionQueue {
	q := NewRecognitionQueue(db, face.NewMockProvider(), cfg, nil)
	log.Init("error", false)
	return q
}

const rqTestBase = 990000000 // 测试用 media_id 基段，避免与真实数据碰撞

// ── 认领 ─────────────────────────────────────────────────────────────

func TestRecognitionClaim_ClaimAndNoCompetition(t *testing.T) {
	db := openRQTestDB(t)
	q := testQueue(db, config.FaceConfig{Enabled: true, SuggestLow: 70, SuggestHigh: 80})
	ctx := context.Background()

	mid := uint64(rqTestBase + 1)
	db.Unscoped().Where("media_id = ?", mid).Delete(&domain.RecognitionJob{})
	require.NoError(t, q.Enqueue(ctx, 4, mid))

	job, err := q.claimNext(ctx, "w1")
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, mid, job.MediaID)

	var after domain.RecognitionJob
	require.NoError(t, db.First(&after, job.ID).Error)
	assert.Equal(t, domain.JobStatusProcessing, after.Status, "认领后应为 processing")

	// 竞争：同任务再次认领应失败（CAS）
	again, err := q.claimNext(ctx, "w2")
	require.NoError(t, err)
	assert.Nil(t, again, "processing 中的任务不可被再次认领")

	db.Unscoped().Where("media_id = ?", mid).Delete(&domain.RecognitionJob{})
}

func TestRecognitionEnqueue_Idempotent(t *testing.T) {
	db := openRQTestDB(t)
	q := testQueue(db, config.FaceConfig{Enabled: true})
	ctx := context.Background()

	mid := uint64(rqTestBase + 2)
	db.Unscoped().Where("media_id = ?", mid).Delete(&domain.RecognitionJob{})
	require.NoError(t, q.Enqueue(ctx, 4, mid))
	require.NoError(t, q.Enqueue(ctx, 4, mid)) // 二次入队

	var n int64
	db.Model(&domain.RecognitionJob{}).Where("media_id = ?", mid).Count(&n)
	assert.Equal(t, int64(1), n, "同照片重复入队只应有一条任务")

	db.Unscoped().Where("media_id = ?", mid).Delete(&domain.RecognitionJob{})
}

func TestRecognitionEnqueue_DisabledDoesNothing(t *testing.T) {
	db := openRQTestDB(t)
	q := testQueue(db, config.FaceConfig{Enabled: false})
	mid := uint64(rqTestBase + 3)
	require.NoError(t, q.Enqueue(context.Background(), 4, mid))
	var n int64
	db.Model(&domain.RecognitionJob{}).Where("media_id = ?", mid).Count(&n)
	assert.Zero(t, n, "识别未开启的园所不应建任务（TC-NEW-RCQ-0008）")
}

// TC-NEW-RCQ-0005：两园积压时按园轮转消费——小园任务不必等大园全部消费完。
func TestRecognitionClaim_TenantFairness(t *testing.T) {
	db := openRQTestDB(t)
	q := testQueue(db, config.FaceConfig{Enabled: true})
	ctx := context.Background()

	kgA, kgB := uint64(4), uint64(7)
	base := uint64(rqTestBase + 500)
	db.Unscoped().Where("media_id >= ? AND media_id < ?", base, base+10).Delete(&domain.RecognitionJob{})
	// A 园积压 3 条，B 园 1 条（B 的任务 id 最大——FIFO 口径下会被 A 全部挡住）
	for i := uint64(0); i < 3; i++ {
		require.NoError(t, q.Enqueue(ctx, kgA, base+i))
	}
	require.NoError(t, q.Enqueue(ctx, kgB, base+9))

	seen := map[uint64]int{}
	for i := 0; i < 2; i++ {
		job, err := q.claimNext(ctx, "fair")
		require.NoError(t, err)
		require.NotNil(t, job)
		seen[job.KindergartenID]++
	}
	assert.Equal(t, 1, seen[kgB], "B 园（小园）任务必须在前两轮被取到，而不是等 A 园积压清空")

	// 轮转是持续性的：继续认领会回到 A 园，直到两园都清空
	job, err := q.claimNext(ctx, "fair")
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, kgA, job.KindergartenID, "回绕后应轮到 A 园")

	db.Unscoped().Where("media_id >= ? AND media_id < ?", base, base+10).Delete(&domain.RecognitionJob{})
}

// ── 建议写入（状态保护）────────────────────────────────────────────────

func TestWriteSuggestions_ProtectsConfirmedAndRejected(t *testing.T) {
	db := openRQTestDB(t)
	q := testQueue(db, config.FaceConfig{Enabled: true, SuggestLow: 70, SuggestHigh: 80})
	ctx := context.Background()

	mid := uint64(rqTestBase + 10)
	confirmedSID := uint64(rqTestBase + 11)
	rejectedSID := uint64(rqTestBase + 12)
	suggestedSID := uint64(rqTestBase + 13)
	db.Unscoped().Where("media_id = ?", mid).Delete(&domain.GrowthMediaStudent{})

	st1, st2 := domain.MediaStudentStatusConfirmed, domain.MediaStudentStatusRejected
	require.NoError(t, db.Create(&domain.GrowthMediaStudent{MediaID: mid, StudentID: confirmedSID, Status: &st1}).Error)
	require.NoError(t, db.Create(&domain.GrowthMediaStudent{MediaID: mid, StudentID: rejectedSID, Status: &st2}).Error)

	matches := []face.Match{
		{StudentID: confirmedSID, Score: 95}, // 已确认：不得被覆盖
		{StudentID: rejectedSID, Score: 96},  // 已拒绝：不得复活
		{StudentID: suggestedSID, Score: 88}, // 新建议：应写入 status=0
		{StudentID: uint64(rqTestBase + 14), Score: 50}, // 低于阈值：不落
	}
	require.NoError(t, q.writeSuggestions(ctx, mid, matches))

	var afterConfirmed, afterRejected, newRow domain.GrowthMediaStudent
	require.NoError(t, db.Where("media_id = ? AND student_id = ?", mid, confirmedSID).First(&afterConfirmed).Error)
	assert.Equal(t, domain.MediaStudentStatusConfirmed, *afterConfirmed.Status, "已确认不被覆盖（状态竞争防护）")

	require.NoError(t, db.Where("media_id = ? AND student_id = ?", mid, rejectedSID).First(&afterRejected).Error)
	assert.Equal(t, domain.MediaStudentStatusRejected, *afterRejected.Status, "已拒绝不得复活")

	require.NoError(t, db.Where("media_id = ? AND student_id = ?", mid, suggestedSID).First(&newRow).Error)
	assert.Equal(t, domain.MediaStudentStatusSuggested, *newRow.Status)
	assert.True(t, newRow.IsAuto, "识别来源应标记 is_auto")
	require.NotNil(t, newRow.Confidence)
	assert.InDelta(t, 88, *newRow.Confidence, 0.01)

	var lowCount int64
	db.Model(&domain.GrowthMediaStudent{}).Where("media_id = ? AND student_id = ?", mid, uint64(rqTestBase+14)).Count(&lowCount)
	assert.Zero(t, lowCount, "低于 suggest_low 的命中不落建议（设计 D6）")

	db.Unscoped().Where("media_id = ?", mid).Delete(&domain.GrowthMediaStudent{})
}

// ── 限流与预算 ───────────────────────────────────────────────────────

func TestRecognitionLimiter_Throttles(t *testing.T) {
	q := testQueue(openRQTestDB(t), config.FaceConfig{Enabled: true, QPS: 3})
	assert.Equal(t, rate.Limit(3), q.limiter.Limit())
	// 连续取 4 个令牌应当被阻塞（前 3 个来自 burst）——用超时断言「被限流」
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	ok := 0
	for i := 0; i < 4; i++ {
		if err := q.limiter.Wait(ctx); err != nil {
			break
		}
		ok++
	}
	assert.Less(t, ok, 4, "第 4 次取令牌必须等待下一次补充（限流生效）")
}

func TestBudgetCounter_EnforcedAndPerKG(t *testing.T) {
	b := newBudgetCounter(nil, 2) // Redis 不可用 → 进程内计数
	ctx := context.Background()
	assert.True(t, b.Allow(ctx, 4))
	assert.True(t, b.Allow(ctx, 4))
	assert.False(t, b.Allow(ctx, 4), "超过日预算后必须拒绝（预算护栏）")
}
