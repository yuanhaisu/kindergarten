package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"kindergarten-server/internal/domain"
)

// 管理端任务统计的新口径（design D12 / tasks 7.2、7.4）。
//
// 口径集中在这一个文件里，别处不要再算：
//   - 反馈类分母是「义务集」`task_responses.notify_target_count`，不是 messages 行数
//     （decision §20.10-6）；NULL = 未知，排除并单独计数，不能当 0；
//   - 聚合起点是 `task_check_ins.response_id`：补充回应同样是 TaskResponse，
//     但不被任何提交引用，天然不进任何主回应指标；
//   - 中位数在应用层排序计算（decision §20.10-7），SQL 只负责取数；
//   - 比率一律是 "NN%" 且封顶 100%，无分母时为 nil（JSON null），不是 "0%"。

// taskStatsTask 参与统计的任务（含班级/活动归属与目标口径所需字段）。
type taskStatsTask struct {
	ID              uint64
	ClassID         uint64
	ClassName       string
	Title           string
	ActivityID      string
	TargetStudentID uint64
	Status          int8
}

// taskStatsCheckin 区间内的一条提交。
type taskStatsCheckin struct {
	TaskID     uint64
	StudentID  uint64
	ResponseID uint64
	CreatedAt  time.Time
}

// taskStatsResponse 被提交引用的主回应（或 period 汇总回应）。
type taskStatsResponse struct {
	ResponseType      string
	CreatedAt         time.Time
	NotifyTargetCount *int
}

// taskStatsReaction 家长反馈（已按 (response_id, user_id) 去重）。
// UserID 只用于老师端「按孩子」归属，管理端汇总不需要它。
type taskStatsReaction struct {
	UserID   uint64
	Reaction string
}

// taskStatsInput 一次统计需要的全部原始数据，一次读完再按桶聚合。
type taskStatsInput struct {
	tasks         []*taskStatsTask
	checkins      []*taskStatsCheckin
	responses     map[uint64]taskStatsResponse
	reactions     map[uint64][]taskStatsReaction
	classStudents map[uint64][]uint64
}

// taskStatsBucket 一个聚合桶（全园 / 某班 / 某活动 / 某独立任务）。
type taskStatsBucket struct {
	tasks          []*taskStatsTask
	checkins       []*taskStatsCheckin
	targetStudents map[uint64]struct{}
}

// taskStatsMetrics 一个桶的计算结果（比率字段为 nil 表示无法计算）。
type taskStatsMetrics struct {
	Tasks                 int
	TotalCheckins         int64
	Responded             int64
	Pending               int64
	PendingOver48h        int64
	ResponseRate          any
	ParticipationRate     any
	MedianResponseHours   any
	NegativeExcluded      int
	FeedbackCoverage      any
	ChildHappyRate        any
	AckRate               any
	NotifyBaselineMissing int
}

const (
	taskStatsBucketAll           = "all"
	taskStatsBucketClassPrefix   = "class:"
	taskStatsBucketActPrefix     = "activity:"
	taskStatsBucketStandalonePre = "task:"
)

// loadTaskStatsInput 读一次原始数据：任务、区间内提交、被引用的回应、反馈、班级在读学生。
func (s *TaskService) loadTaskStatsInput(
	ctx context.Context,
	kindergartenID uint64,
	rangeStart, rangeEnd time.Time,
) (*taskStatsInput, error) {
	rows, err := s.tasks.ListTasksOverlappingRange(ctx, kindergartenID, rangeStart, rangeEnd)
	if err != nil {
		return nil, err
	}
	in := &taskStatsInput{
		tasks:         make([]*taskStatsTask, 0, len(rows)),
		responses:     map[uint64]taskStatsResponse{},
		reactions:     map[uint64][]taskStatsReaction{},
		classStudents: map[uint64][]uint64{},
	}
	taskIDs := make([]uint64, 0, len(rows))
	classSet := map[uint64]struct{}{}
	for _, r := range rows {
		in.tasks = append(in.tasks, &taskStatsTask{
			ID: r.ID, ClassID: r.ClassID, ClassName: r.ClassName, Title: r.Title,
			ActivityID: r.ActivityID, TargetStudentID: r.TargetStudentID, Status: r.Status,
		})
		taskIDs = append(taskIDs, r.ID)
		if r.TargetStudentID == 0 {
			classSet[r.ClassID] = struct{}{}
		}
	}
	if len(taskIDs) == 0 {
		return in, nil
	}

	// 提交按 `date` 落区间：只统计区间内发生的提交，与 total_checkins 同口径。
	var checkinRows []struct {
		TaskID     uint64    `gorm:"column:task_id"`
		StudentID  uint64    `gorm:"column:student_id"`
		ResponseID uint64    `gorm:"column:response_id"`
		CreatedAt  time.Time `gorm:"column:created_at"`
	}
	if err := s.db.WithContext(ctx).Table("task_check_ins").
		Select("task_id, student_id, response_id, created_at").
		Where("task_id IN ? AND deleted_at IS NULL", taskIDs).
		Where("date >= ? AND date <= ?", rangeStart, rangeEnd).
		Scan(&checkinRows).Error; err != nil {
		return nil, err
	}
	in.checkins = make([]*taskStatsCheckin, 0, len(checkinRows))
	responseSet := map[uint64]struct{}{}
	for _, c := range checkinRows {
		in.checkins = append(in.checkins, &taskStatsCheckin{
			TaskID: c.TaskID, StudentID: c.StudentID, ResponseID: c.ResponseID, CreatedAt: c.CreatedAt,
		})
		if c.ResponseID != 0 {
			responseSet[c.ResponseID] = struct{}{}
		}
	}

	if len(responseSet) > 0 {
		responseIDs := make([]uint64, 0, len(responseSet))
		for id := range responseSet {
			responseIDs = append(responseIDs, id)
		}
		sort.Slice(responseIDs, func(i, j int) bool { return responseIDs[i] < responseIDs[j] })

		var respRows []struct {
			ID                uint64    `gorm:"column:id"`
			ResponseType      string    `gorm:"column:response_type"`
			CreatedAt         time.Time `gorm:"column:created_at"`
			NotifyTargetCount *int      `gorm:"column:notify_target_count"`
		}
		if err := s.db.WithContext(ctx).Table("task_responses").
			Select("id, response_type, created_at, notify_target_count").
			Where("id IN ? AND deleted_at IS NULL", responseIDs).
			Scan(&respRows).Error; err != nil {
			return nil, err
		}
		for _, r := range respRows {
			in.responses[r.ID] = taskStatsResponse{
				ResponseType: r.ResponseType, CreatedAt: r.CreatedAt, NotifyTargetCount: r.NotifyTargetCount,
			}
		}

		var reactRows []struct {
			ResponseID uint64 `gorm:"column:response_id"`
			UserID     uint64 `gorm:"column:user_id"`
			Reaction   string `gorm:"column:reaction"`
		}
		if err := s.db.WithContext(ctx).Table("task_response_reactions").
			Select("response_id, user_id, reaction").
			Where("response_id IN ? AND deleted_at IS NULL", responseIDs).
			Scan(&reactRows).Error; err != nil {
			return nil, err
		}
		// 唯一索引保证 (response_id, user_id) 只有一条；这里仍按对去重，
		// 免得将来软删复活出现两行时把「同一家长的反馈」算两次。
		seen := make(map[[2]uint64]struct{}, len(reactRows))
		for _, r := range reactRows {
			key := [2]uint64{r.ResponseID, r.UserID}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			in.reactions[r.ResponseID] = append(in.reactions[r.ResponseID], taskStatsReaction{UserID: r.UserID, Reaction: r.Reaction})
		}
	}

	// 班级任务的目标孩子 = 当前在读学生。个体任务的目标是 TargetStudentID，单独算。
	if len(classSet) > 0 {
		classIDs := make([]uint64, 0, len(classSet))
		for id := range classSet {
			classIDs = append(classIDs, id)
		}
		var studentRows []struct {
			ID      uint64 `gorm:"column:id"`
			ClassID uint64 `gorm:"column:class_id"`
		}
		if err := s.db.WithContext(ctx).Table("students").
			Select("id, class_id").
			Where("kindergarten_id = ? AND deleted_at IS NULL AND status = 1 AND class_id IN ?", kindergartenID, classIDs).
			Scan(&studentRows).Error; err != nil {
			return nil, err
		}
		for _, st := range studentRows {
			in.classStudents[st.ClassID] = append(in.classStudents[st.ClassID], st.ID)
		}
	}
	return in, nil
}

// buildBuckets 把任务与提交分到「全园 / 班级 / 活动或独立任务」三类桶里。
// 独立任务按任务 id 各自成桶（design D12：不能全塞进一个「空活动」）。
func (in *taskStatsInput) buildBuckets() map[string]*taskStatsBucket {
	buckets := make(map[string]*taskStatsBucket)
	get := func(key string) *taskStatsBucket {
		b := buckets[key]
		if b == nil {
			b = &taskStatsBucket{targetStudents: map[uint64]struct{}{}}
			buckets[key] = b
		}
		return b
	}
	taskKeys := make(map[uint64][]string, len(in.tasks))
	for _, t := range in.tasks {
		keys := []string{
			taskStatsBucketAll,
			fmt.Sprintf("%s%d", taskStatsBucketClassPrefix, t.ClassID),
		}
		if t.ActivityID != "" {
			keys = append(keys, taskStatsBucketActPrefix+t.ActivityID)
		} else {
			keys = append(keys, fmt.Sprintf("%s%d", taskStatsBucketStandalonePre, t.ID))
		}
		taskKeys[t.ID] = keys
		for _, k := range keys {
			b := get(k)
			b.tasks = append(b.tasks, t)
			if t.TargetStudentID > 0 {
				b.targetStudents[t.TargetStudentID] = struct{}{}
			} else {
				for _, sid := range in.classStudents[t.ClassID] {
					b.targetStudents[sid] = struct{}{}
				}
			}
		}
	}
	for _, c := range in.checkins {
		for _, k := range taskKeys[c.TaskID] {
			b := get(k)
			b.checkins = append(b.checkins, c)
		}
	}
	return buckets
}

// metrics 计算一个桶的全部指标。
func (in *taskStatsInput) metrics(b *taskStatsBucket, now time.Time) taskStatsMetrics {
	m := taskStatsMetrics{Tasks: len(b.tasks), TotalCheckins: int64(len(b.checkins))}
	cutoff := now.Add(-48 * time.Hour)

	participating := make(map[uint64]struct{}, len(b.checkins))
	referenced := make(map[uint64]struct{})
	var durations []float64
	for _, c := range b.checkins {
		participating[c.StudentID] = struct{}{}
		if c.ResponseID == 0 {
			m.Pending++
			if c.CreatedAt.Before(cutoff) {
				m.PendingOver48h++
			}
			continue
		}
		m.Responded++
		referenced[c.ResponseID] = struct{}{}
		r, ok := in.responses[c.ResponseID]
		// 中位数只算逐条 single 主回应：period 汇总覆盖 N 条提交，
		// 拿它当 N 个样本会让「平均回应时长」被汇总节奏带偏（design D12）。
		if !ok || r.ResponseType != domain.TaskResponseSingle {
			continue
		}
		h := r.CreatedAt.Sub(c.CreatedAt).Hours()
		if h < 0 {
			// 时钟漂移或手工改库：负时长没有意义，剔除但要能被观察到。
			m.NegativeExcluded++
			continue
		}
		durations = append(durations, h)
	}

	m.ResponseRate = taskStatRate64(m.Responded, m.TotalCheckins)
	// 参与率：目标范围里去重后至少提交过一次的孩子 / 目标孩子去重数。
	// 用集合而不是「每任务比例求平均」，避免同一孩子被多个任务重复计入分母。
	participants := 0
	for sid := range participating {
		if _, ok := b.targetStudents[sid]; ok {
			participants++
		}
	}
	m.ParticipationRate = taskStatRate64(int64(participants), int64(len(b.targetStudents)))
	m.MedianResponseHours = taskStatMedianHours(durations)

	// 反馈类：分母 = 被本桶提交引用的回应所承诺的义务集。
	responseIDs := make([]uint64, 0, len(referenced))
	for id := range referenced {
		responseIDs = append(responseIDs, id)
	}
	sort.Slice(responseIDs, func(i, j int) bool { return responseIDs[i] < responseIDs[j] })
	totals := taskFeedbackTotalsOf(responseIDs, in.responses, in.reactions)
	m.NotifyBaselineMissing = totals.BaselineMissing
	m.FeedbackCoverage = taskStatRate(totals.Pairs, totals.Denominator)
	m.ChildHappyRate = taskStatRate(totals.Happy, totals.Denominator)
	m.AckRate = taskStatRate(totals.Ack(), totals.Denominator)
	return m
}

// taskFeedbackTotals 是按 (回应, 家长) 对聚合出来的反馈总量。
//
// 这是反馈口径的**唯一实现**：管理端 7.2 统计与老师端 3.11 任务汇总共用，
// 避免「管理端算义务集、老师端算反馈行」这类漂移。
type taskFeedbackTotals struct {
	Denominator     int // 被引用的主回应所承诺的义务家长数（notify_target_count 之和）
	Pairs           int // 去重后的 (回应, 家长) 反馈对数
	Happy           int // child_happy
	Received        int // received
	Thanks          int // thanks
	BaselineMissing int // notify_target_count 为 NULL 的回应数（未知义务，排除并计数）
}

// Ack 是家长确认（received + thanks），与孩子感知信号分开。
func (t taskFeedbackTotals) Ack() int { return t.Received + t.Thanks }

func taskFeedbackTotalsOf(
	referenced []uint64,
	responses map[uint64]taskStatsResponse,
	reactions map[uint64][]taskStatsReaction,
) taskFeedbackTotals {
	var out taskFeedbackTotals
	for _, id := range referenced {
		r, ok := responses[id]
		if !ok {
			// 回应已删：既没有分母也没有「已反馈」，整条跳过。
			continue
		}
		if r.NotifyTargetCount == nil {
			// 未知义务：排除并计数，不能当 0（否则这个回应会拖低整体覆盖率）。
			out.BaselineMissing++
			continue
		}
		out.Denominator += *r.NotifyTargetCount
		for _, rc := range reactions[id] {
			out.Pairs++
			switch rc.Reaction {
			case domain.TaskReactionChildHappy:
				out.Happy++
			case domain.TaskReactionReceived:
				out.Received++
			case domain.TaskReactionThanks:
				out.Thanks++
			}
		}
	}
	return out
}

// taskStatRate 整数分子/分母 → "NN%"，封顶 100%；分母 <= 0 返回 nil。
// 封顶是必须的：义务集小于实际反馈家长数时（历史数据、手工改库），
// 未封顶会算出 120% 这种一眼假的数字（spec：MUST NOT 超过 100%）。
func taskStatRate(num, den int) any {
	if den <= 0 {
		return nil
	}
	p := num * 100 / den
	if p > 100 {
		p = 100
	}
	if p < 0 {
		p = 0
	}
	return fmt.Sprintf("%d%%", p)
}

func taskStatRate64(num, den int64) any {
	if den <= 0 {
		return nil
	}
	p := num * 100 / den
	if p > 100 {
		p = 100
	}
	if p < 0 {
		p = 0
	}
	return fmt.Sprintf("%d%%", p)
}

// taskStatMedianHours 应用层中位数（decision §20.10-7）：
// 偶数取两中值平均，四舍五入到 1 位小数；空集返回 nil。
func taskStatMedianHours(values []float64) any {
	if len(values) == 0 {
		return nil
	}
	sort.Float64s(values)
	var med float64
	mid := len(values) / 2
	if len(values)%2 == 1 {
		med = values[mid]
	} else {
		med = (values[mid-1] + values[mid]) / 2
	}
	return math.Round(med*10) / 10
}

// toMap 转成契约字段；桶内与全园共用同一套键名。
func (m taskStatsMetrics) toMap() map[string]any {
	return map[string]any{
		"total_checkins":                    m.TotalCheckins,
		"response_rate":                     m.ResponseRate,
		"participation_rate":                m.ParticipationRate,
		"pending_count":                     m.Pending,
		"pending_over_48h":                  m.PendingOver48h,
		"median_response_hours":             m.MedianResponseHours,
		"median_response_negative_excluded": m.NegativeExcluded,
		"feedback_coverage":                 m.FeedbackCoverage,
		"child_happy_rate":                  m.ChildHappyRate,
		"ack_rate":                          m.AckRate,
		"notify_baseline_missing":           m.NotifyBaselineMissing,
	}
}

// taskDetailMetrics 单任务详情用的新口径（7.4 替代旧 completion_rate/checkin_count）。
// 不按时间窗过滤：任务详情是累计视角。
func (s *TaskService) taskDetailMetrics(
	ctx context.Context,
	kindergartenID uint64,
	t *domain.ParentTask,
) (map[string]any, error) {
	submissions, err := s.tasks.CountCheckinsByTaskID(ctx, t.ID)
	if err != nil {
		return nil, err
	}
	var responded int64
	if err := s.db.WithContext(ctx).Model(&domain.TaskCheckIn{}).
		Where("task_id = ? AND response_id <> 0", t.ID).Count(&responded).Error; err != nil {
		return nil, err
	}
	participants, err := s.tasks.CountParticipatingStudents(ctx, kindergartenID, t)
	if err != nil {
		return nil, err
	}
	var targets int64
	if t.TargetStudentID > 0 {
		targets = 1
	} else {
		targets, err = s.tasks.CountActiveStudentsInClass(ctx, kindergartenID, t.ClassID)
		if err != nil {
			return nil, err
		}
	}
	return map[string]any{
		"submission_count":   submissions,
		"responded_count":    responded,
		"pending_count":      submissions - responded,
		"response_rate":      taskStatRate64(responded, submissions),
		"participation_rate": taskStatRate64(participants, targets),
	}, nil
}
