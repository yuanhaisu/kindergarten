package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

// 覆盖 openspec/changes/notice-scheduled-publish/specs/notice-scheduling/spec.md
// 的 6 条 Scenario（TC-NEW-SCH-0001 ~ 0006）：到点发布、未到点不动、连续两轮、
// 并发只成功一次、失败隔离、非定时状态跳过。
// 复用 notice_multitarget_test.go 的园所 990201 夹具；结束由 newNMTCleanup 硬删，0 残留。

type schedulerFixture struct {
	*nmtFixture
	scheduler *NoticeScheduler
}

func newSchedulerFixture(t *testing.T) *schedulerFixture {
	t.Helper()
	f := newNMTFixture(t)
	return &schedulerFixture{nmtFixture: f, scheduler: NewNoticeScheduler(f.repo, 0)}
}

// createNotice 直接落库以精确控制 scheduled_at / status（不走 service 的发布前校验）。
func (f *schedulerFixture) createNotice(t *testing.T, title string, status domain.NoticeStatus, scheduledAt, publishedAt *time.Time) uint64 {
	t.Helper()
	n := &domain.Notice{
		KindergartenID: testNMTKG,
		CreatorID:      testNMTTeacher,
		ClassID:        f.classA,
		Category:       domain.NoticeCategoryHomework,
		Title:          title,
		Content:        "正文",
		Status:         status,
		ScheduledAt:    scheduledAt,
		PublishedAt:    publishedAt,
	}
	require.NoError(t, f.db.Create(n).Error)
	return n.ID
}

func reloadNotice(t *testing.T, db *gorm.DB, id uint64) domain.Notice {
	t.Helper()
	var n domain.Notice
	require.NoError(t, db.Unscoped().First(&n, id).Error)
	return n
}

// TC-NEW-SCH-0001：到点通知被下一轮扫描发布，写入 published_at。
func TestNoticeSchedulerPublishesDueNotice(t *testing.T) {
	f := newSchedulerFixture(t)
	ctx := context.Background()
	due := time.Now().Add(-time.Minute)
	id := f.createNotice(t, "到点通知", domain.NoticeStatusScheduled, &due, nil)

	published, err := f.scheduler.RunOnce(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, published, 1)

	got := reloadNotice(t, f.db, id)
	assert.Equal(t, domain.NoticeStatusPublished, got.Status)
	require.NotNil(t, got.PublishedAt, "到点发布必须写 published_at")
	assert.WithinDuration(t, time.Now(), *got.PublishedAt, time.Minute)
}

// TC-NEW-SCH-0002：预定时间仍在未来时扫描不改变状态。
func TestNoticeSchedulerLeavesFutureNoticeAlone(t *testing.T) {
	f := newSchedulerFixture(t)
	ctx := context.Background()
	future := time.Now().Add(time.Hour)
	id := f.createNotice(t, "未来通知", domain.NoticeStatusScheduled, &future, nil)

	_, err := f.scheduler.RunOnce(ctx)
	require.NoError(t, err)

	got := reloadNotice(t, f.db, id)
	assert.Equal(t, domain.NoticeStatusScheduled, got.Status)
	assert.Nil(t, got.PublishedAt, "未到点不得写发布时间")
}

// TC-NEW-SCH-0003：连续两轮扫描，第二轮不再改 published_at。
func TestNoticeSchedulerSecondRoundIsNoop(t *testing.T) {
	f := newSchedulerFixture(t)
	ctx := context.Background()
	due := time.Now().Add(-time.Minute)
	id := f.createNotice(t, "幂等通知", domain.NoticeStatusScheduled, &due, nil)

	_, err := f.scheduler.RunOnce(ctx)
	require.NoError(t, err)
	first := reloadNotice(t, f.db, id)
	require.NotNil(t, first.PublishedAt)

	_, err = f.scheduler.RunOnce(ctx)
	require.NoError(t, err)
	second := reloadNotice(t, f.db, id)

	assert.Equal(t, domain.NoticeStatusPublished, second.Status)
	require.NotNil(t, second.PublishedAt)
	assert.True(t, first.PublishedAt.Equal(*second.PublishedAt),
		"第二轮不得重写 published_at：first=%s second=%s", first.PublishedAt, second.PublishedAt)
}

// TC-NEW-SCH-0004：两个 goroutine 同时发布同一条，只有一个拿到 RowsAffected==1。
func TestNoticeSchedulerConcurrentPublishOnlyOnce(t *testing.T) {
	f := newSchedulerFixture(t)
	due := time.Now().Add(-time.Minute)
	id := f.createNotice(t, "并发通知", domain.NoticeStatusScheduled, &due, nil)
	now := time.Now()

	results := make([]bool, 2)
	errs := make([]error, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			results[i], errs[i] = f.repo.PublishScheduledNotice(context.Background(), id, now)
		}(i)
	}
	close(start)
	wg.Wait()

	successes := 0
	for i := range results {
		require.NoError(t, errs[i])
		if results[i] {
			successes++
		}
	}
	assert.Equal(t, 1, successes, "同一条通知只能被条件更新成功一次")
	assert.Equal(t, domain.NoticeStatusPublished, reloadNotice(t, f.db, id).Status)
}

// failOnePublishRepo 只让指定通知的发布返回错误，用于验证失败隔离。
type failOnePublishRepo struct {
	NoticeSchedulerRepo
	failID uint64
}

func (r failOnePublishRepo) PublishScheduledNotice(ctx context.Context, noticeID uint64, now time.Time) (bool, error) {
	if noticeID == r.failID {
		return false, errors.New("injected publish failure")
	}
	return r.NoticeSchedulerRepo.PublishScheduledNotice(ctx, noticeID, now)
}

// TC-NEW-SCH-0005：同批一条失败不影响其它，失败条目保持定时状态并记日志。
func TestNoticeSchedulerFailureIsolation(t *testing.T) {
	f := newSchedulerFixture(t)
	ctx := context.Background()
	earlier := time.Now().Add(-2 * time.Minute)
	later := time.Now().Add(-time.Minute)
	failID := f.createNotice(t, "注入失败", domain.NoticeStatusScheduled, &earlier, nil)
	okID := f.createNotice(t, "正常发布", domain.NoticeStatusScheduled, &later, nil)

	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	defer slog.SetDefault(prev)

	s := NewNoticeScheduler(failOnePublishRepo{NoticeSchedulerRepo: f.repo, failID: failID}, 0)
	published, err := s.RunOnce(ctx)
	require.NoError(t, err, "单条失败不应升级为整轮错误")
	assert.GreaterOrEqual(t, published, 1)

	failed := reloadNotice(t, f.db, failID)
	assert.Equal(t, domain.NoticeStatusScheduled, failed.Status, "失败条目保持定时状态，下一轮重试")
	assert.Nil(t, failed.PublishedAt)
	assert.Equal(t, domain.NoticeStatusPublished, reloadNotice(t, f.db, okID).Status, "同批其它通知不受影响")

	logs := buf.String()
	assert.Contains(t, logs, "notice scheduled publish failed")
	assert.Contains(t, logs, fmt.Sprintf("notice_id=%d", failID), "失败日志必须带 notice_id")
}

// TC-NEW-SCH-0006：草稿 / 已发布 / 已撤回 / 软删通知即使到点也不被发布。
func TestNoticeSchedulerSkipsNonScheduledStatuses(t *testing.T) {
	f := newSchedulerFixture(t)
	ctx := context.Background()
	due := time.Now().Add(-time.Minute)
	alreadyPublished := time.Now().Add(-2 * time.Minute).Truncate(time.Millisecond)

	draftID := f.createNotice(t, "草稿", domain.NoticeStatusDraft, &due, nil)
	publishedID := f.createNotice(t, "已发布", domain.NoticeStatusPublished, &due, &alreadyPublished)
	revokedID := f.createNotice(t, "已撤回", domain.NoticeStatusRevoked, &due, nil)
	deletedID := f.createNotice(t, "软删定时", domain.NoticeStatusScheduled, &due, nil)
	require.NoError(t, f.db.Delete(&domain.Notice{}, deletedID).Error)

	_, err := f.scheduler.RunOnce(ctx)
	require.NoError(t, err)

	assert.Equal(t, domain.NoticeStatusDraft, reloadNotice(t, f.db, draftID).Status)
	published := reloadNotice(t, f.db, publishedID)
	assert.Equal(t, domain.NoticeStatusPublished, published.Status)
	require.NotNil(t, published.PublishedAt)
	assert.WithinDuration(t, alreadyPublished, *published.PublishedAt, time.Millisecond, "已发布通知的发布时间不得被重写")
	assert.Equal(t, domain.NoticeStatusRevoked, reloadNotice(t, f.db, revokedID).Status)

	deleted := reloadNotice(t, f.db, deletedID)
	assert.Equal(t, domain.NoticeStatusScheduled, deleted.Status, "软删通知不发布")
	assert.Nil(t, deleted.PublishedAt)
	assert.True(t, deleted.DeletedAt.Valid)

	dueList, err := f.repo.ListDueScheduledNotices(ctx, time.Now(), 100)
	require.NoError(t, err)
	skipped := []uint64{draftID, publishedID, revokedID, deletedID}
	for _, n := range dueList {
		assert.NotContains(t, skipped, n.ID, "非定时/软删通知不得进入到期查询")
	}
}

// 清理证据：执行同一套清理后本园所通知 0 残留（含软删行）。
func TestNoticeSchedulerCleanupLeavesNoResidue(t *testing.T) {
	f := newSchedulerFixture(t)
	ctx := context.Background()
	due := time.Now().Add(-time.Minute)
	f.createNotice(t, "待清理", domain.NoticeStatusScheduled, &due, nil)
	_, err := f.scheduler.RunOnce(ctx)
	require.NoError(t, err)

	newNMTCleanup(t, f.db)

	var notices int64
	require.NoError(t, f.db.WithContext(ctx).Unscoped().Model(&domain.Notice{}).
		Where("kindergarten_id = ?", testNMTKG).Count(&notices).Error)
	assert.Zero(t, notices, "清理后本园所通知应为 0（含软删）")
}

// Start 幂等由 sync.Once 保证；这里只钉住重复 Start 后 Stop 仍能及时返回（不挂死）。
func TestNoticeSchedulerStartStopIdempotent(t *testing.T) {
	f := newSchedulerFixture(t)
	s := NewNoticeScheduler(f.repo, 10*time.Millisecond)
	s.Start()
	s.Start()

	done := make(chan struct{})
	go func() {
		s.Stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("重复 Start 后 Stop 未及时返回")
	}
}
