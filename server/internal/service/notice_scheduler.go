package service

import (
	"context"
	"sync"
	"time"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/pkg/log"
)

const (
	// noticePublishScanLimit 每轮扫描上限：单条独立更新，限制批量避免长事务（设计 D1）。
	noticePublishScanLimit = 100
	// noticePublishInterval 默认扫描间隔：到点最多延迟一个间隔，可接受（设计 D1）。
	noticePublishInterval = 60 * time.Second
)

// NoticeSchedulerRepo 定时发布所需的最小仓储操作。
// 抽成接口是为了测试能注入单条失败（失败隔离），生产实现是 *repository.NoticeRepo。
type NoticeSchedulerRepo interface {
	ListDueScheduledNotices(ctx context.Context, now time.Time, limit int) ([]domain.Notice, error)
	PublishScheduledNotice(ctx context.Context, noticeID uint64, now time.Time) (bool, error)
}

// NoticeScheduler 定时通知到点发布 worker（openspec/changes/notice-scheduled-publish）。
//
// 形态：固定间隔轮询 status=1 且 scheduled_at<=now 的通知，逐条条件更新为已发布。
// 幂等与多实例安全不靠锁：PublishScheduledNotice 的 WHERE 条件保证同一条通知
// 无论被扫到几次、几个实例同时扫，只有一次能拿到 RowsAffected==1，其余跳过。
type NoticeScheduler struct {
	repo     NoticeSchedulerRepo
	interval time.Duration
	limit    int

	startOnce sync.Once
	stopOnce  sync.Once
	stopCh    chan struct{}
	wg        sync.WaitGroup
}

func NewNoticeScheduler(repo NoticeSchedulerRepo, interval time.Duration) *NoticeScheduler {
	if interval <= 0 {
		interval = noticePublishInterval
	}
	return &NoticeScheduler{
		repo:     repo,
		interval: interval,
		limit:    noticePublishScanLimit,
		stopCh:   make(chan struct{}),
	}
}

// NewNoticeSchedulerForApp 生产构造（不启动；启动点在应用装配处，便于 wire 重新生成后仍生效）。
func NewNoticeSchedulerForApp(db *gorm.DB) *NoticeScheduler {
	return NewNoticeScheduler(repository.NewNoticeRepo(db), 0)
}

// RunOnce 执行一轮扫描并逐条发布，返回本轮成功发布条数。
// 单条发布失败只记结构化日志并继续（失败条目保持定时状态，下一轮自然重试）；
// 只有扫描本身失败才返回错误，由调用方记录后等下一轮。
func (s *NoticeScheduler) RunOnce(ctx context.Context) (int, error) {
	now := time.Now()
	due, err := s.repo.ListDueScheduledNotices(ctx, now, s.limit)
	if err != nil {
		return 0, err
	}
	published := 0
	for i := range due {
		id := due[i].ID
		ok, err := s.repo.PublishScheduledNotice(ctx, id, now)
		if err != nil {
			log.L(ctx).Error("notice scheduled publish failed", "notice_id", id, "err", err)
			continue
		}
		if !ok {
			// 0 行受影响：已被其它实例/上一轮发布，或状态在扫描后变化——跳过即可。
			continue
		}
		published++
		log.L(ctx).Info("notice scheduled published", "notice_id", id)
	}
	return published, nil
}

// Start 启动后台轮询。幂等：重复调用不会叠加 goroutine。
func (s *NoticeScheduler) Start() {
	s.startOnce.Do(func() {
		s.wg.Add(1)
		go s.loop()
		log.L(context.Background()).Info("notice scheduler started", "interval", s.interval.String())
	})
}

func (s *NoticeScheduler) loop() {
	defer s.wg.Done()
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			if _, err := s.RunOnce(context.Background()); err != nil {
				log.L(context.Background()).Error("notice scheduler scan failed", "err", err)
			}
		}
	}
}

// Stop 停止轮询并等待在途一轮结束。幂等。
func (s *NoticeScheduler) Stop() {
	s.stopOnce.Do(func() { close(s.stopCh) })
	s.wg.Wait()
}
