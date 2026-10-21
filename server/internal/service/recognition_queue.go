package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
	"gorm.io/gorm"

	"kindergarten-server/internal/ai/face"
	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
	"kindergarten-server/pkg/log"
)

// RecognitionQueue 照片人脸识别异步队列（设计 D2-D5）。
//
// 形态：DB 表驱动轮询（recognition_jobs），worker 池并发消费；
// 限流保护云配额、按园所轮转认领保证租户公平、日预算护栏、故障退避重试。
// 识别的产物是「建议」（status=0），老师确认后才对家长生效（status=1）。
type RecognitionQueue struct {
	db       *gorm.DB
	provider face.Provider
	cfg      config.FaceConfig
	limiter  *rate.Limiter // 全局令牌桶：多 worker 共享，保护云端配额

	budget *budgetCounter // 日预算计数（Redis 不可用时退化为进程内计数）

	stopOnce sync.Once
	stopCh   chan struct{}
	wg       sync.WaitGroup

	// 租户公平轮转游标（D4）：记录上次认领的园所，下一次从它之后找。
	// 单实例内多 worker 共享（加锁）；多实例时各实例独立轮转，仍近似公平。
	rrMu   sync.Mutex
	rrLast uint64

	// 测试钩子：下载图片（默认 HTTP GET，测试注入假实现）
	fetch func(ctx context.Context, url string) ([]byte, error)
	// 测试钩子：KG 是否开启识别（默认查 kindergarten_app_settings，未接入前全开）
	kgEnabled func(ctx context.Context, kgID uint64) bool
}

func NewRecognitionQueue(db *gorm.DB, provider face.Provider, cfg config.FaceConfig, rdb redisLike) *RecognitionQueue {
	qps := cfg.QPS
	if qps <= 0 {
		qps = 10
	}
	q := &RecognitionQueue{
		db:       db,
		provider: provider,
		cfg:      cfg,
		limiter:  rate.NewLimiter(rate.Limit(qps), qps), // burst=1 秒的量
		budget:   newBudgetCounter(rdb, cfg.DailyBudget),
		stopCh:   make(chan struct{}),
	}
	q.fetch = httpFetch
	q.kgEnabled = func(ctx context.Context, kgID uint64) bool { return true } // 灰度白名单接入点
	return q
}

// redisLike 最小 Redis 接口（便于测试注入；生产传入 redisAdapter）。
type redisLike interface {
	Incr(ctx context.Context, key string) (int64, error)
	Expire(ctx context.Context, key string, ttl time.Duration) error
}

// redisAdapter 把 *redis.Client 适配为 redisLike（go-redis 的方法带 *IntCmd 返回值，不直接匹配接口）。
type redisAdapter struct{ c *redis.Client }

func (a redisAdapter) Incr(ctx context.Context, key string) (int64, error) {
	return a.c.Incr(ctx, key).Result()
}

func (a redisAdapter) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return a.c.Expire(ctx, key, ttl).Err()
}

// NewRecognitionQueueForApp 生产构造：Redis 适配 + provider 工厂。
// 作为应用级后台服务在构造时自启动（队列为空时 worker 空转，无副作用）——
// 避免改动 Handler/main 的装配链。
func NewRecognitionQueueForApp(db *gorm.DB, rdb *redis.Client, cfg *config.Config, provider face.Provider) *RecognitionQueue {
	q := NewRecognitionQueue(db, provider, cfg.Face, redisAdapter{c: rdb})
	q.Start(3)
	return q
}

// Start 启动 worker 池。幂等（重复调用直接返回）。
func (q *RecognitionQueue) Start(workers int) {
	if !q.cfg.Enabled {
		log.L(context.Background()).Info("recognition queue disabled by config, not starting")
		return
	}
	if workers <= 0 {
		workers = 3
	}
	for i := 0; i < workers; i++ {
		q.wg.Add(1)
		go q.workerLoop(fmt.Sprintf("w%d", i))
	}
	log.L(context.Background()).Info("recognition queue started", "workers", workers, "provider", q.provider.Name())
}

// Stop 优雅停止（等在途任务结束）。
func (q *RecognitionQueue) Stop() {
	q.stopOnce.Do(func() { close(q.stopCh) })
	q.wg.Wait()
}

func (q *RecognitionQueue) workerLoop(workerID string) {
	defer q.wg.Done()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-q.stopCh:
			return
		case <-ticker.C:
			// 每个 tick 尽量排空可领任务（直到无可领或预算触顶）
			for q.processOne(context.Background(), workerID) {
			}
		}
	}
}

// processOne 认领并处理一个任务；返回是否处理了任务（false = 无可领，worker 回睡）。
func (q *RecognitionQueue) processOne(ctx context.Context, workerID string) bool {
	job, err := q.claimNext(ctx, workerID)
	if err != nil {
		log.L(ctx).Error("recognition claim failed", "err", err)
		return false
	}
	if job == nil {
		return false // 队空
	}

	if !q.budget.Allow(ctx, job.KindergartenID) {
		// 预算触顶：任务回退为 pending（不消费配额），本轮停止认领
		q.release(ctx, job.ID, "daily budget reached")
		return false
	}

	if err := q.handle(ctx, job); err != nil {
		q.retryOrFail(ctx, job, err)
		return true
	}
	q.db.WithContext(ctx).Model(&domain.RecognitionJob{}).Where("id = ?", job.ID).
		Updates(map[string]any{"status": domain.JobStatusDone, "last_error": "", "face_count": job.FaceCount})
	return true
}

// claimNext 原子认领：按园所轮转取该园最早任务，再 CAS 置为 processing（防多 worker 重复认领）。
func (q *RecognitionQueue) claimNext(ctx context.Context, workerID string) (*domain.RecognitionJob, error) {
	now := time.Now().Unix()
	kgID, err := q.nextKindergarten(ctx, now)
	if err != nil || kgID == 0 {
		return nil, err
	}
	var job domain.RecognitionJob
	if err := q.db.WithContext(ctx).
		Where("kindergarten_id = ? AND status = ? AND next_run_at <= ?", kgID, domain.JobStatusPending, now).
		Order("id ASC").First(&job).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	// CAS：只有仍是 pending 才归本 worker
	res := q.db.WithContext(ctx).Model(&domain.RecognitionJob{}).
		Where("id = ? AND status = ?", job.ID, domain.JobStatusPending).
		Updates(map[string]any{"status": domain.JobStatusProcessing, "worker_id": workerID})
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, nil // 被其他 worker 抢走
	}
	q.rrMu.Lock()
	q.rrLast = kgID
	q.rrMu.Unlock()
	return &job, nil
}

// nextKindergarten 轮转选取下一个有待领任务的园所（D4 租户公平）：
// 起于上次认领的园所之后（id 序），到末尾则回绕——保证各园任务交替被消费，
// 而不是「谁的任务最老谁先做」（那会让大园积压把小园饿死）。
func (q *RecognitionQueue) nextKindergarten(ctx context.Context, now int64) (uint64, error) {
	q.rrMu.Lock()
	last := q.rrLast
	q.rrMu.Unlock()

	pick := func(after uint64) (uint64, error) {
		var kgID uint64
		err := q.db.WithContext(ctx).Model(&domain.RecognitionJob{}).
			Where("status = ? AND next_run_at <= ? AND kindergarten_id > ?", domain.JobStatusPending, now, after).
			Order("kindergarten_id ASC").Limit(1).Pluck("kindergarten_id", &kgID).Error
		return kgID, err
	}
	kgID, err := pick(last)
	if err != nil || kgID != 0 {
		return kgID, err
	}
	if last == 0 {
		return 0, nil // 无待领任务
	}
	return pick(0) // 回绕
}

// handle 执行识别：读图 → 限流 → 调 provider → 写建议。
func (q *RecognitionQueue) handle(ctx context.Context, job *domain.RecognitionJob) error {
	var media domain.GrowthMedia
	if err := q.db.WithContext(ctx).First(&media, job.MediaID).Error; err != nil {
		return fmt.Errorf("media %d 不存在: %w", job.MediaID, err) // 记录被删：按业务失败处理
	}
	img, err := q.fetch(ctx, media.FileURL)
	if err != nil {
		return fmt.Errorf("取图失败: %w", err)
	}
	if err := q.limiter.Wait(ctx); err != nil { // 限流（保护云配额，D3）
		return err
	}
	res, err := q.provider.Search(ctx, job.KindergartenID, img)
	if err != nil {
		return fmt.Errorf("识别失败: %w", err)
	}
	job.FaceCount = res.FaceNum
	// 建议只覆盖本记录涉及的学生（跨班命中不产建议、不可确认）
	allowed, targetType, err := q.allowedStudents(ctx, media)
	if err != nil {
		return fmt.Errorf("解析记录范围失败: %w", err)
	}
	// 大合影（D11）：逐人名单在人脸上限下注定不完整，产建议只会让老师做伪工作——
	// 不写任何建议，只把人脸数留给 UI 呈现（老师端呈现「大合影 · 一键全班」）。
	// 仅对**全班记录**分型：个体记录的建议本就限定在该记录的目标孩子（数量有限），
	// 逐人确认不是伪工作，保留建议（复审裁定 2026-10-16）。
	if targetType == 1 && res.FaceNum >= q.groupPhotoMinFaces() {
		return nil
	}
	return q.writeSuggestions(ctx, media.ID, filterMatches(res.Matches, allowed))
}

// groupPhotoMinFaces 大合影阈值（配置驱动，默认 6）。
func (q *RecognitionQueue) groupPhotoMinFaces() int {
	if q.cfg.GroupPhotoMinFaces > 0 {
		return q.cfg.GroupPhotoMinFaces
	}
	return 6
}

// allowedStudents 本记录允许被建议的学生集合（口径与点名白名单 D9 一致）：
// 全班记录 → 本班在读；个体记录 → 该记录关联学生；无归属 → 空集。
// 同时返回记录类型（1 全班 / 2 个体；无归属为 0），供大合影分型判定（个体记录不做抑制）。
func (q *RecognitionQueue) allowedStudents(ctx context.Context, media domain.GrowthMedia) (map[uint64]bool, int8, error) {
	out := make(map[uint64]bool)
	if media.LogID == nil {
		return out, 0, nil
	}
	var log domain.GrowthDailyLog
	if err := q.db.WithContext(ctx).First(&log, *media.LogID).Error; err != nil {
		return nil, 0, err
	}
	var ids []uint64
	if log.TargetType == 2 {
		if err := q.db.WithContext(ctx).Model(&domain.GrowthLogStudent{}).
			Where("log_id = ?", log.ID).Pluck("student_id", &ids).Error; err != nil {
			return nil, 0, err
		}
	} else {
		if err := q.db.WithContext(ctx).Model(&domain.Student{}).
			Where("kindergarten_id = ? AND class_id = ? AND status = 1", log.KindergartenID, log.ClassID).
			Pluck("id", &ids).Error; err != nil {
			return nil, 0, err
		}
	}
	for _, id := range ids {
		out[id] = true
	}
	return out, log.TargetType, nil
}

// filterMatches 过滤出范围外的命中（跨班相似脸不得产生建议）。
func filterMatches(matches []face.Match, allowed map[uint64]bool) []face.Match {
	out := make([]face.Match, 0, len(matches))
	for _, m := range matches {
		if allowed[m.StudentID] {
			out = append(out, m)
		}
	}
	return out
}

// writeSuggestions 将识别命中写为建议行（status=0）。
// 已存在且已确认/拒绝的行不被覆盖（状态机竞争防护，设计 Risks 最后一条）。
func (q *RecognitionQueue) writeSuggestions(ctx context.Context, mediaID uint64, matches []face.Match) error {
	low := float64(q.cfg.SuggestLow)
	for _, m := range matches {
		if m.Score < low {
			continue // 低于低阈值不落建议（设计 D6）
		}
		conf := m.Score
		status := domain.MediaStudentStatusSuggested
		err := q.db.WithContext(ctx).Exec(`
			INSERT INTO growth_media_students (media_id, student_id, is_auto, status, confidence, created_at, updated_at)
			VALUES (?, ?, 1, ?, ?, NOW(3), NOW(3))
			ON DUPLICATE KEY UPDATE
			  confidence = IF(status IS NULL OR status = 0, VALUES(confidence), confidence)`,
			mediaID, m.StudentID, status, conf).Error
		if err != nil {
			return err
		}
	}
	return nil
}

// retryOrFail 网络类错误退避重试（≤3 次）；超限落 failed 保留可诊断原因（设计 D5）。
func (q *RecognitionQueue) retryOrFail(ctx context.Context, job *domain.RecognitionJob, cause error) {
	const maxRetries = 3
	msg := cause.Error()
	if len(msg) > 250 {
		msg = msg[:250]
	}
	if job.Retries+1 >= maxRetries {
		q.db.WithContext(ctx).Model(&domain.RecognitionJob{}).Where("id = ?", job.ID).
			Updates(map[string]any{"status": domain.JobStatusFailed, "last_error": msg, "retries": job.Retries + 1})
		log.L(ctx).Error("recognition job failed permanently", "job", job.ID, "err", msg)
		return
	}
	backoff := int64(30 * (job.Retries + 1)) // 30s/60s/90s
	q.db.WithContext(ctx).Model(&domain.RecognitionJob{}).Where("id = ?", job.ID).
		Updates(map[string]any{
			"status": domain.JobStatusPending, "retries": job.Retries + 1,
			"last_error": msg, "next_run_at": time.Now().Unix() + backoff,
		})
}

// release 将 processing 任务放回 pending（预算触顶等可恢复情形）。
func (q *RecognitionQueue) release(ctx context.Context, jobID uint64, reason string) {
	q.db.WithContext(ctx).Model(&domain.RecognitionJob{}).Where("id = ?", jobID).
		Updates(map[string]any{
			"status": domain.JobStatusPending, "worker_id": "",
			"next_run_at": time.Now().Unix() + 300, "last_error": reason,
		})
}

// Enqueue 为一张照片建识别任务（幂等：media_id 唯一；该园未开启识别时不建）。
func (q *RecognitionQueue) Enqueue(ctx context.Context, kgID, mediaID uint64) error {
	if !q.cfg.Enabled || !q.kgEnabled(ctx, kgID) {
		return nil
	}
	return q.db.WithContext(ctx).Exec(`
		INSERT INTO recognition_jobs (media_id, kindergarten_id, status, retries, next_run_at, created_at, updated_at)
		VALUES (?, ?, 'pending', 0, 0, NOW(3), NOW(3))
		ON DUPLICATE KEY UPDATE updated_at = updated_at`, mediaID, kgID).Error
}

// ── 辅助 ─────────────────────────────────────────────────────────────

// httpFetch 默认取图实现（OSS 直链）。
func httpFetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("取图 %s: HTTP %d", url, resp.StatusCode)
	}
	// 5MB 上限（腾讯云 Image 参数限制，提前拦截避免浪费配额）
	return io.ReadAll(io.LimitReader(resp.Body, 5<<20))
}

// ── 预算护栏（设计 D5）────────────────────────────────────────────────

// budgetCounter 日调用预算计数：Redis 全局 + 按园双维；Redis 不可用退化为进程内计数（宁可少认不失控）。
type budgetCounter struct {
	rdb   redisLike
	limit int
	mu    sync.Mutex
	local map[string]int64
}

func newBudgetCounter(rdb redisLike, limit int) *budgetCounter {
	if limit <= 0 {
		limit = 10000
	}
	return &budgetCounter{rdb: rdb, limit: limit, local: map[string]int64{}}
}

// Allow 记账并判断是否仍可发起调用。key 维度：全局 + kg:{id}。
func (b *budgetCounter) Allow(ctx context.Context, kgID uint64) bool {
	day := time.Now().Format("20060102")
	globalKey := "face:budget:global:" + day
	kgKey := fmt.Sprintf("face:budget:kg%d:%s", kgID, day)

	if b.rdb != nil {
		g, err1 := b.rdb.Incr(ctx, globalKey)
		if err1 == nil {
			_ = b.rdb.Expire(ctx, globalKey, 48*time.Hour)
			k, err2 := b.rdb.Incr(ctx, kgKey)
			if err2 == nil {
				_ = b.rdb.Expire(ctx, kgKey, 48*time.Hour)
			}
			// 单园上限 = 全局上限（可后续细分）；超限即拒绝并告警
			if g > int64(b.limit) || k > int64(b.limit) {
				log.L(ctx).Warn("recognition daily budget reached", "global", g, "kg", kgID, "kg_count", k, "limit", b.limit)
				return false
			}
			return true
		}
	}
	// Redis 不可用：进程内计数（重启清零——宁可多认几次也不因计数故障完全停摆）
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.local[globalKey] >= int64(b.limit) {
		return false
	}
	b.local[globalKey]++
	b.local[kgKey]++
	return true
}
