package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/pkg/pagination"
)

type TaskService struct {
	tasks *repository.TaskRepo
	oplog *repository.OperationLogRepo
	msg   *repository.MessageRepo
	db    *gorm.DB
}

func NewTaskService(tasks *repository.TaskRepo, oplog *repository.OperationLogRepo, msg *repository.MessageRepo, db *gorm.DB) *TaskService {
	return &TaskService{tasks: tasks, oplog: oplog, msg: msg, db: db}
}

var errTaskNotFound = errors.New("TASK_NOT_FOUND")

func IsTaskNotFound(err error) bool {
	return errors.Is(err, errTaskNotFound)
}

func taskFrequencyLabel(f domain.TaskFrequency) string {
	switch f {
	case domain.TaskFrequencyDaily:
		return "每日任务"
	case domain.TaskFrequencyWeekly:
		return "每周任务"
	case domain.TaskFrequencyOnce:
		return "一次性任务"
	default:
		return string(f)
	}
}

// formatPercent 通用的「已完成/总量 → NN%」格式化（任务与通知模块共用）。
// 任务的新口径走 task_stats.go 的 taskStatRate；这里保留给通知等旧调用方。
func formatPercent(done, total int64) string {
	if total <= 0 {
		return "0%"
	}
	p := int(done * 100 / total)
	if p > 100 {
		p = 100
	}
	return fmt.Sprintf("%d%%", p)
}

func (s *TaskService) ListTasks(
	ctx context.Context,
	kindergartenID uint64,
	pq pagination.Query,
	classID *uint64,
	status *int8,
) ([]map[string]interface{}, int64, int, int, error) {
	if kindergartenID == 0 {
		return nil, 0, 0, 0, errors.New("NO_KINDERGARTEN")
	}
	pn := pq
	pn.Normalize()
	offset := pn.Offset()
	rows, total, err := s.tasks.ListTasks(ctx, kindergartenID, classID, status, offset, pn.PageSize)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	out := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		f := domain.TaskFrequency(row.Frequency)
		out = append(out, map[string]interface{}{
			"id":                 row.ID,
			"title":              row.Title,
			"class_name":         row.ClassName,
			"class_id":           row.ClassID,
			"learning_link":      row.LearningLink,
			"target_student_id":  row.TargetStudentID,
			"frequency":          string(f),
			"frequency_label":    taskFrequencyLabel(f),
			"start_date":         row.StartDate.In(shanghaiLoc()).Format("2006-01-02"),
			"end_date":           row.EndDate.In(shanghaiLoc()).Format("2006-01-02"),
			"status":             int(row.Status),
			"allow_public":       row.AllowPublic,
			"recognition_ritual": row.RecognitionRitual,
			"star_scope":         row.StarScope,
			"star_goal":          row.StarGoal,
			"activity_id":        row.ActivityID,
			// 旧 completion_rate 已下线（7.4）；每任务口径见 GET /admin/tasks/:id
			// 与 GET /admin/tasks/stats，列表不再逐行算比率（省掉两倍 N+1 查询）。
			"creator":    map[string]interface{}{"nickname": row.CreatorNick},
			"created_at": row.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return out, total, pn.Page, pn.PageSize, nil
}

func (s *TaskService) GetTaskDetail(ctx context.Context, kindergartenID, taskID uint64) (map[string]interface{}, error) {
	if kindergartenID == 0 {
		return nil, errors.New("NO_KINDERGARTEN")
	}
	t, err := s.tasks.FindTaskByID(ctx, kindergartenID, taskID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, errTaskNotFound
	}
	// 新口径（7.4）：提交/回应/参与率，替代旧 completion_rate / checkin_count / total_possible。
	metrics, err := s.taskDetailMetrics(ctx, kindergartenID, t)
	if err != nil {
		return nil, err
	}
	className := ""
	if t.Class != nil {
		className = t.Class.Name
	}
	creatorNick, creatorAvatar := "", ""
	var creatorID uint64
	if t.Creator != nil {
		creatorNick = t.Creator.Nickname
		creatorAvatar = t.Creator.Avatar
		creatorID = t.Creator.ID
	}
	out := map[string]interface{}{
		"id":                 t.ID,
		"title":              t.Title,
		"description":        t.Description,
		"class_id":           t.ClassID,
		"class_name":         className,
		"learning_link":      t.LearningLink,
		"target_student_id":  t.TargetStudentID,
		"frequency":          string(t.Frequency),
		"frequency_label":    taskFrequencyLabel(t.Frequency),
		"start_date":         t.StartDate.In(shanghaiLoc()).Format("2006-01-02"),
		"end_date":           t.EndDate.In(shanghaiLoc()).Format("2006-01-02"),
		"status":             int(t.Status),
		"allow_public":       t.AllowPublic,
		"recognition_ritual": t.RecognitionRitual,
		"star_scope":         t.StarScope,
		"star_goal":          t.StarGoal,
		"activity_id":        t.ActivityID,
		// 发布通知的义务集大小；nil = 未知（历史任务或收件人查询失败）。
		"notify_target_count": t.NotifyTargetCount,
		"creator": map[string]interface{}{
			"id":       creatorID,
			"nickname": creatorNick,
			"avatar":   creatorAvatar,
		},
		"created_at": t.CreatedAt.UTC().Format(time.RFC3339),
	}
	for k, v := range metrics {
		out[k] = v
	}
	return out, nil
}

func (s *TaskService) DeleteTask(ctx context.Context, kindergartenID, adminUserID, taskID uint64, ip, ua string) error {
	if kindergartenID == 0 {
		return errors.New("NO_KINDERGARTEN")
	}
	t, err := s.tasks.FindTaskByID(ctx, kindergartenID, taskID)
	if err != nil {
		return err
	}
	if t == nil {
		return errTaskNotFound
	}
	if t.ActivityID != "" {
		// 活动任务只能整组删除：单删会让活动的其它班级任务与统计聚合悬空。
		return ErrTaskActivityDeleteForbidden
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 与老师端 DeleteTask / SupplementResponse / ReactToResponse 同一把序：
		// 先锁 parent_tasks 行，再动 task_responses。否则管理端与老师端并发删除时
		// 会形成「任务锁 ↔ 回应锁」的 A→B / B→A 死锁。
		var locked domain.ParentTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND kindergarten_id = ? AND deleted_at IS NULL", taskID, kindergartenID).
			First(&locked).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errTaskNotFound
			}
			return err
		}
		return deleteTaskCascade(ctx, tx, taskID)
	}); err != nil {
		if errors.Is(err, errTaskNotFound) {
			return errTaskNotFound
		}
		// 死锁/锁等待超时是可重试冲突，不能报成 500。
		if isTaskLockConflict(err) {
			return ErrTaskBusy
		}
		return err
	}
	_ = s.oplog.Create(ctx, &domain.OperationLog{
		KindergartenID: kindergartenID,
		UserID:         adminUserID,
		Module:         "tasks",
		Action:         "delete",
		TargetType:     "parent_task",
		TargetID:       taskID,
		Detail:         fmt.Sprintf("title=%s", t.Title),
		IP:             ip,
		UserAgent:      truncateUA(ua),
	})
	return nil
}

// RevokePublicCheckin 管理端强制撤回一条已公开（或待审核）的提交。
//
// 管理端只能撤回：不允许代替家长申请，也不允许绕过老师审核直接批准——
// 否则「内容中只有提交者孩子」这条人为确认就失去了唯一责任人。
func (s *TaskService) RevokePublicCheckin(ctx context.Context, kindergartenID, adminUserID, checkinID uint64, ip, ua string) (map[string]any, error) {
	if kindergartenID == 0 {
		return nil, errors.New("NO_KINDERGARTEN")
	}
	now := time.Now().In(shanghaiLoc())
	var out map[string]any
	changed := false
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		loc, err := locateCheckin(ctx, tx, checkinID)
		if err != nil {
			return err
		}
		// 第一把锁：parent_tasks 行，与老师端/家长端同一锁序（handoff §12.2）。
		var t domain.ParentTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND kindergarten_id = ? AND deleted_at IS NULL", loc.TaskID, kindergartenID).
			First(&t).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errTaskNotFound
			}
			return err
		}
		var c domain.TaskCheckIn
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND task_id = ?", checkinID, t.ID).First(&c).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errTaskNotFound
			}
			return err
		}
		switch publicCheckinStatus(&c) {
		case TaskPublicPending, TaskPublicApproved:
			if err := tx.Model(&domain.TaskCheckIn{}).Where("id = ?", c.ID).
				Updates(publicRevokeUpdates(now)).Error; err != nil {
				return err
			}
			var fresh domain.TaskCheckIn
			if err := tx.Where("id = ?", c.ID).First(&fresh).Error; err != nil {
				return err
			}
			c = fresh
			changed = true
		}
		out = publicCheckinView(&t, &c, loc.StudentClassID, now)
		return nil
	})
	if err != nil {
		if errors.Is(err, errTaskNotFound) {
			return nil, errTaskNotFound
		}
		// 死锁/锁等待超时是可重试冲突，不能报成 500。
		if isTaskLockConflict(err) {
			return nil, ErrTaskBusy
		}
		return nil, err
	}
	// 空操作不写操作日志：否则「点了但什么都没撤」会被记成一次撤回。
	if changed {
		_ = s.oplog.Create(ctx, &domain.OperationLog{
			KindergartenID: kindergartenID,
			UserID:         adminUserID,
			Module:         "tasks",
			Action:         "revoke_public",
			TargetType:     "task_check_in",
			TargetID:       checkinID,
			Detail:         fmt.Sprintf("task_id=%d", out["task_id"]),
			IP:             ip,
			UserAgent:      truncateUA(ua),
		})
	}
	return out, nil
}

// ListPublicCheckins 管理端查看某任务的公开内容（7.3 的「查看」入口）。
//
// 只读：不改任何公开状态。撤回仍走 RevokePublicCheckin——
// 这条端点存在的意义是让管理端拿到 checkin_id 与 public_status/public_visible，
// 而不是让管理员复制一套状态机判定。
//
// status: public（默认，is_public_requested=true）/ revoked（已撤回）/ all。
func (s *TaskService) ListPublicCheckins(
	ctx context.Context,
	kindergartenID, taskID uint64,
	status string,
	pq pagination.Query,
) (map[string]any, error) {
	if kindergartenID == 0 {
		return nil, errors.New("NO_KINDERGARTEN")
	}
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		status = "public"
	}
	switch status {
	case "public", "revoked", "all":
	default:
		return nil, errors.New("BAD_PUBLIC_STATUS")
	}
	t, err := s.tasks.FindTaskByID(ctx, kindergartenID, taskID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, errTaskNotFound
	}

	var rows []struct {
		ID                   uint64     `gorm:"column:id"`
		StudentID            uint64     `gorm:"column:student_id"`
		Content              string     `gorm:"column:content"`
		MediaURL             string     `gorm:"column:media_url"`
		CreatedAt            time.Time  `gorm:"column:created_at"`
		IsPublicRequested    bool       `gorm:"column:is_public_requested"`
		PublicScopeConfirmed bool       `gorm:"column:public_scope_confirmed"`
		PublicApprovedAt     *time.Time `gorm:"column:public_approved_at"`
		PublicApprovedBy     uint64     `gorm:"column:public_approved_by"`
		PublicRevokedAt      *time.Time `gorm:"column:public_revoked_at"`
		StudentName          string     `gorm:"column:student_name"`
		StudentClassID       uint64     `gorm:"column:student_class_id"`
	}
	q := s.db.WithContext(ctx).Table("task_check_ins AS c").
		Select(`c.id, c.student_id, c.content, c.media_url, c.created_at,
			c.is_public_requested, c.public_scope_confirmed, c.public_approved_at,
			c.public_approved_by, c.public_revoked_at,
			COALESCE(s.name, '') AS student_name, COALESCE(s.class_id, 0) AS student_class_id`).
		Joins("LEFT JOIN students AS s ON s.id = c.student_id AND s.deleted_at IS NULL").
		Where("c.task_id = ? AND c.deleted_at IS NULL", taskID)
	switch status {
	case "public":
		// pending + approved：这两类才是「能撤回」的公开内容。
		q = q.Where("c.is_public_requested = ?", true)
	case "revoked":
		q = q.Where("c.is_public_requested = ? AND c.public_revoked_at IS NOT NULL", false)
	}
	// 分页是必须的：status=all 对学期长的每日任务可能一次返回几千行带正文与媒体。
	pn := pq
	pn.Normalize()
	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, err
	}
	if err := q.Order("c.created_at DESC, c.id DESC").
		Offset(pn.Offset()).Limit(pn.PageSize).Scan(&rows).Error; err != nil {
		return nil, err
	}

	now := time.Now().In(shanghaiLoc())
	list := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		c := domain.TaskCheckIn{
			BaseModel:            domain.BaseModel{ID: r.ID, CreatedAt: r.CreatedAt},
			TaskID:               t.ID,
			StudentID:            r.StudentID,
			Content:              r.Content,
			MediaURL:             r.MediaURL,
			IsPublicRequested:    r.IsPublicRequested,
			PublicScopeConfirmed: r.PublicScopeConfirmed,
			PublicApprovedAt:     r.PublicApprovedAt,
			PublicApprovedBy:     r.PublicApprovedBy,
			PublicRevokedAt:      r.PublicRevokedAt,
		}
		// public_status / public_visible 与公开墙、撤回端点共用同一套判定：
		// 客户端自己拼条件会导致「界面显示已公开、墙上却没有」。
		item := publicCheckinView(t, &c, r.StudentClassID, now)
		item["student_id"] = r.StudentID
		item["student_name"] = r.StudentName
		item["content"] = r.Content
		item["media_url"] = r.MediaURL
		item["created_at"] = r.CreatedAt.UTC().Format(time.RFC3339)
		list = append(list, item)
	}
	return map[string]any{
		"list":      list,
		"total":     total,
		"page":      pn.Page,
		"page_size": pn.PageSize,
	}, nil
}

// CreateTaskReq POST /admin/tasks
//
// 管理端创建的是**独立任务**：activity_id 只能由 /admin/task-activities 生成，
// 这里收到非空值直接拒绝（decision §20.7-4）——静默忽略会让调用方以为分组成功。
type CreateTaskReq struct {
	Title             string `json:"title" binding:"required,max=200"`
	Description       string `json:"description"`
	LearningLink      string `json:"learning_link"`
	ClassID           uint64 `json:"class_id" binding:"required"`
	TargetStudentID   uint64 `json:"target_student_id"`
	Frequency         string `json:"frequency" binding:"required"`
	StartDate         string `json:"start_date" binding:"required"`
	EndDate           string `json:"end_date" binding:"required"`
	AllowPublic       bool   `json:"allow_public"`
	RecognitionRitual string `json:"recognition_ritual"`
	StarScope         string `json:"star_scope"`
	StarGoal          int    `json:"star_goal"`
	ActivityID        string `json:"activity_id"`
}

// resolveTaskRitual 空值取任务默认仪式；创建默认贴纸，更新保留原值（空仍为空）。
// 空字符串在历史数据里表示「未设置」，不能强行归一化——9.7-#3 已踩过一次。
func resolveTaskRitual(reqRitual, fallback, whenEmpty string) string {
	r := strings.TrimSpace(reqRitual)
	if r != "" {
		return r
	}
	if fallback != "" {
		return fallback
	}
	return whenEmpty
}

// resolveTaskStarScope 空值取 fallback；仍为空时按设计默认 task。
func resolveTaskStarScope(reqScope, fallback string) string {
	s := strings.TrimSpace(reqScope)
	if s != "" {
		return s
	}
	if fallback != "" {
		return fallback
	}
	return domain.TaskStarScopeTask
}

func parseTaskFrequency(s string) (domain.TaskFrequency, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case string(domain.TaskFrequencyDaily):
		return domain.TaskFrequencyDaily, nil
	case string(domain.TaskFrequencyWeekly):
		return domain.TaskFrequencyWeekly, nil
	case string(domain.TaskFrequencyOnce):
		return domain.TaskFrequencyOnce, nil
	default:
		return "", errors.New("BAD_FREQUENCY")
	}
}

// parseTaskDateRange 解析任务起止日期（上海时区，YYYY-MM-DD）。
// 单任务与多班活动共用：两份实现迟早会在「结束日早于开始日」上给出不同错误码。
func parseTaskDateRange(start, end string) (time.Time, time.Time, error) {
	loc := shanghaiLoc()
	sd, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(start), loc)
	if err != nil {
		return time.Time{}, time.Time{}, errors.New("BAD_DATE")
	}
	ed, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(end), loc)
	if err != nil {
		return time.Time{}, time.Time{}, errors.New("BAD_DATE")
	}
	if ed.Before(sd) {
		return time.Time{}, time.Time{}, errors.New("BAD_DATE_RANGE")
	}
	return sd, ed, nil
}

func (s *TaskService) CreateTask(
	ctx context.Context,
	kindergartenID, adminUserID uint64,
	req CreateTaskReq,
	ip, ua string,
) (map[string]interface{}, error) {
	if kindergartenID == 0 {
		return nil, errors.New("NO_KINDERGARTEN")
	}
	if strings.TrimSpace(req.ActivityID) != "" {
		return nil, ErrTaskActivityIDForbidden
	}
	if strings.TrimSpace(req.Title) == "" {
		// gin 的 required 只判非空字符串，纯空白标题会以空串落库（更新路径已拒，创建也必须拒）。
		return nil, errors.New("BAD_TITLE")
	}
	if err := validateTaskTitleLength(req.Title); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.LearningLink) == "" {
		// D15：学习关联必填。管理端与老师端同一口径，否则家长看到的任务没有学习关联。
		return nil, errors.New("BAD_LEARNING_LINK")
	}
	ok, err := s.tasks.ClassBelongsToKindergarten(ctx, kindergartenID, req.ClassID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrClassNotFound
	}
	freq, err := parseTaskFrequency(req.Frequency)
	if err != nil {
		return nil, err
	}
	sd, ed, err := parseTaskDateRange(req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}
	// 结构规则全部走 task_rules.go，不在 service 里重写一遍（D12）。
	// 老师端 / 管理端独立任务 / 多班活动三个入口共用同一个校验函数。
	structure := taskStructure{
		Ritual:    resolveTaskRitual(req.RecognitionRitual, "", domain.TaskRitualSticker),
		StarScope: resolveTaskStarScope(req.StarScope, ""),
		StarGoal:  req.StarGoal,
	}
	if err := validateTaskStructureForCreate(ctx, s.db, kindergartenID, req.ClassID, req.TargetStudentID, structure); err != nil {
		return nil, err
	}
	// 个体任务强制不可公开：公开墙只服务班级活动（D7）。
	allowPublic := req.AllowPublic && req.TargetStudentID == 0
	t := &domain.ParentTask{
		KindergartenID:    kindergartenID,
		ClassID:           req.ClassID,
		CreatorID:         adminUserID,
		TargetStudentID:   req.TargetStudentID,
		Title:             strings.TrimSpace(req.Title),
		Description:       strings.TrimSpace(req.Description),
		LearningLink:      strings.TrimSpace(req.LearningLink),
		Frequency:         freq,
		StartDate:         sd,
		EndDate:           ed,
		Status:            1,
		AllowPublic:       allowPublic,
		RecognitionRitual: structure.Ritual,
		StarScope:         structure.StarScope,
		StarGoal:          structure.StarGoal,
	}
	if err := s.tasks.CreateTask(ctx, t); err != nil {
		return nil, err
	}
	_ = s.oplog.Create(ctx, &domain.OperationLog{
		KindergartenID: kindergartenID,
		UserID:         adminUserID,
		Module:         "tasks",
		Action:         "create",
		TargetType:     "parent_task",
		TargetID:       t.ID,
		Detail:         fmt.Sprintf("class_id=%d", req.ClassID),
		IP:             ip,
		UserAgent:      truncateUA(ua),
	})
	// 通知在落库之后：失败只记日志与计数，不回滚任务（D9）。
	notified, notifyFailed := notifyTaskPublishedFor(ctx, s.db, s.msg, t)
	return map[string]interface{}{
		"id":                  t.ID,
		"created_at":          t.CreatedAt.UTC().Format(time.RFC3339),
		"notified_count":      notified,
		"notify_failed_count": notifyFailed,
	}, nil
}

// UpdateTaskReq PUT /admin/tasks/:id
//
// 新增字段一律用指针表示「未提供」，否则老客户端不发某个字段会被读成零值：
// `allow_public` 缺省=false 会把公开悄悄关掉并批量撤回，`star_goal` 缺省=0 会把目标清零，
// `description` 缺省="" 会静默清空已有说明。
type UpdateTaskReq struct {
	Title             string  `json:"title" binding:"required,max=200"`
	Description       *string `json:"description"`
	LearningLink      string  `json:"learning_link"`
	ClassID           uint64  `json:"class_id" binding:"required"`
	TargetStudentID   *uint64 `json:"target_student_id"`
	Frequency         string  `json:"frequency" binding:"required"`
	StartDate         string  `json:"start_date" binding:"required"`
	EndDate           string  `json:"end_date" binding:"required"`
	Status            *int8   `json:"status"` // 0 已结束 1 进行中；缺省表示不改
	AllowPublic       *bool   `json:"allow_public"`
	RecognitionRitual string  `json:"recognition_ritual"`
	StarScope         string  `json:"star_scope"`
	StarGoal          *int    `json:"star_goal"`
}

// UpdateTask 管理端编辑任务。
//
// 规则与老师端 UpdateTask 共用 task_rules.go（D12）：有提交后只允许
// title/description/learning_link/allow_public；改个体或关公开必须在同一事务里撤销公开。
func (s *TaskService) UpdateTask(
	ctx context.Context,
	kindergartenID, adminUserID, taskID uint64,
	req UpdateTaskReq,
	ip, ua string,
) error {
	if kindergartenID == 0 {
		return errors.New("NO_KINDERGARTEN")
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return errors.New("BAD_TITLE")
	}
	if err := validateTaskTitleLength(title); err != nil {
		return err
	}
	freq, err := parseTaskFrequency(req.Frequency)
	if err != nil {
		return err
	}
	sd, ed, err := parseTaskDateRange(req.StartDate, req.EndDate)
	if err != nil {
		return err
	}
	if req.Status != nil && *req.Status != 0 && *req.Status != 1 {
		return errors.New("BAD_STATUS")
	}
	// 只校验本次请求给的值：库里已有的历史负值不该把「改标题」也一起挡住。
	if req.StarGoal != nil && !validTaskStarGoal(*req.StarGoal) {
		return ErrTaskStarGoalNegative
	}
	// 与创建路径同口径：只有真正给了非空值才校验，纯空白视为「未提供」。
	if r := strings.TrimSpace(req.RecognitionRitual); r != "" && !validTaskRitual(r) {
		return ErrTaskRitualInvalid
	}
	if sc := strings.TrimSpace(req.StarScope); sc != "" && sc != domain.TaskStarScopeTask && sc != domain.TaskStarScopeClass {
		return ErrTaskStarScopeInvalid
	}
	ok, err := s.tasks.ClassBelongsToKindergarten(ctx, kindergartenID, req.ClassID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrClassNotFound
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 先锁任务行：后续的撤销公开会改 task_check_ins，锁序必须与删除/回应一致。
		var cur domain.ParentTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND kindergarten_id = ? AND deleted_at IS NULL", taskID, kindergartenID).
			First(&cur).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errTaskNotFound
			}
			return err
		}

		learningLink := strings.TrimSpace(req.LearningLink)
		if learningLink == "" {
			// 未提供（或空）时保留原值：D15 要求必填，但老客户端不会带这个字段。
			learningLink = cur.LearningLink
		}
		if learningLink == "" {
			return errors.New("BAD_LEARNING_LINK")
		}
		targetStudentID := cur.TargetStudentID
		if req.TargetStudentID != nil {
			targetStudentID = *req.TargetStudentID
		}
		allowPublic := cur.AllowPublic
		if req.AllowPublic != nil {
			allowPublic = *req.AllowPublic
		}
		starGoal := cur.StarGoal
		if req.StarGoal != nil {
			starGoal = *req.StarGoal
		}
		ritual := resolveTaskRitual(req.RecognitionRitual, cur.RecognitionRitual, "")
		if !validTaskRitual(ritual) {
			return ErrTaskRitualInvalid
		}
		starScope := resolveTaskStarScope(req.StarScope, cur.StarScope)

		if err := validateTargetStudent(ctx, tx, kindergartenID, req.ClassID, targetStudentID); err != nil {
			return err
		}
		// 只在**星星范围或班级真的变了**时校验：任务本来就是 class、而当前学期的罐后来被删时，
		// 不该连改标题都 422（契约只承诺「更新为 class 时」校验）。
		if starScope == domain.TaskStarScopeClass &&
			(starScope != cur.StarScope || req.ClassID != cur.ClassID) {
			if err := validateClassStarJar(ctx, tx, kindergartenID, req.ClassID); err != nil {
				return err
			}
		}
		// 个体任务强制不可公开：落库值与 D7 一致，也避免 revoke 判定看原始请求值。
		effectiveAllowPublic := allowPublic && targetStudentID == 0
		// 只在本次动到目标/仪式/范围时校验：历史脏数据（class_star + goal=0）
		// 不该把「只改标题」一起挡住，与 star_goal 负值那条同口径。
		goalTouched := req.StarGoal != nil || ritual != cur.RecognitionRitual || starScope != cur.StarScope
		if ritual == domain.TaskRitualClassStar && starScope == domain.TaskStarScopeTask && starGoal <= 0 && goalTouched {
			return ErrTaskStarGoalInvalid
		}

		updates := map[string]any{}
		if title != cur.Title {
			updates["title"] = title
		}
		desc := cur.Description
		if req.Description != nil {
			desc = strings.TrimSpace(*req.Description)
		}
		if desc != cur.Description {
			updates["description"] = desc
		}
		if learningLink != cur.LearningLink {
			updates["learning_link"] = learningLink
		}
		if req.ClassID != cur.ClassID {
			updates["class_id"] = req.ClassID
		}
		if targetStudentID != cur.TargetStudentID {
			updates["target_student_id"] = targetStudentID
		}
		if freq != cur.Frequency {
			updates["frequency"] = string(freq)
		}
		if !sameTaskDay(sd, cur.StartDate) {
			updates["start_date"] = sd
		}
		if !sameTaskDay(ed, cur.EndDate) {
			updates["end_date"] = ed
		}
		if effectiveAllowPublic != cur.AllowPublic {
			updates["allow_public"] = effectiveAllowPublic
		}
		if ritual != cur.RecognitionRitual {
			updates["recognition_ritual"] = ritual
		}
		if starScope != cur.StarScope {
			updates["star_scope"] = starScope
		}
		if starGoal != cur.StarGoal {
			updates["star_goal"] = starGoal
		}
		if req.Status != nil && *req.Status != cur.Status {
			updates["status"] = *req.Status
		}
		if len(updates) == 0 {
			return nil
		}
		// 活动任务不能单条改结构（design D12）：标题/说明/学习关联/公开开关可以，
		// 其余（含 status）一律指向活动端点整组改。与「有提交后的结构锁」独立——
		// 活动任务即使还没有提交，也不允许单条改频率/日期/班级/星星。
		if cur.ActivityID != "" {
			for k := range updates {
				if !taskSafeEditableField(k) {
					return ErrTaskActivityEditForbidden
				}
			}
		}
		// status 不是结构字段（D12 的禁止清单里没有它）：单独排除，
		// 否则有提交后连「结束任务」都会被结构锁挡掉。
		structural := make(map[string]any, len(updates))
		for k, v := range updates {
			if k != "status" {
				structural[k] = v
			}
		}
		if err := assertTaskEditable(ctx, tx, taskID, structural); err != nil {
			return err
		}
		if err := tx.Model(&domain.ParentTask{}).
			Where("id = ? AND kindergarten_id = ?", taskID, kindergartenID).
			Updates(updates).Error; err != nil {
			return err
		}
		becameIndividual := targetStudentID > 0 && cur.TargetStudentID == 0
		if becameIndividual || (cur.AllowPublic && !effectiveAllowPublic) {
			reason := "allow_public_off"
			if becameIndividual {
				reason = "became_individual"
			}
			return revokePublicForTask(ctx, tx, kindergartenID, adminUserID, taskID, reason)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, errTaskNotFound) || errors.Is(err, ErrClassNotFound) {
			return err
		}
		// 死锁/锁等待超时是可重试冲突，不能报成 500。
		if isTaskLockConflict(err) {
			return ErrTaskBusy
		}
		return err
	}
	_ = s.oplog.Create(ctx, &domain.OperationLog{
		KindergartenID: kindergartenID,
		UserID:         adminUserID,
		Module:         "tasks",
		Action:         "update",
		TargetType:     "parent_task",
		TargetID:       taskID,
		Detail:         "",
		IP:             ip,
		UserAgent:      truncateUA(ua),
	})
	return nil
}

// GetTaskStats 管理端任务统计（新口径，7.2）。
//
// 全部指标都从 task_stats.go 取：一次读原始数据、按「全园 / 班级 / 活动或独立任务」
// 三个维度各自算一遍，避免同一套公式在多个入口各写一份。
func (s *TaskService) GetTaskStats(ctx context.Context, kindergartenID uint64, startDateStr, endDateStr string) (map[string]interface{}, error) {
	if kindergartenID == 0 {
		return nil, errors.New("NO_KINDERGARTEN")
	}
	rangeStart, rangeEndDay, err := taskStatsRange(startDateStr, endDateStr)
	if err != nil {
		return nil, err
	}

	in, err := s.loadTaskStatsInput(ctx, kindergartenID, rangeStart, rangeEndDay)
	if err != nil {
		return nil, err
	}
	now := time.Now()

	var active, completed int64
	for _, t := range in.tasks {
		if t.Status == 1 {
			active++
		} else {
			completed++
		}
	}

	buckets := in.buildBuckets()
	all := buckets[taskStatsBucketAll]
	if all == nil {
		all = &taskStatsBucket{targetStudents: map[uint64]struct{}{}}
	}
	out := in.metrics(all, now).toMap()
	out["period"] = map[string]interface{}{
		"start": rangeStart.Format("2006-01-02"),
		"end":   rangeEndDay.Format("2006-01-02"),
	}
	out["total_tasks"] = len(in.tasks)
	out["active_tasks"] = active
	out["completed_tasks"] = completed

	// 按班级：同一任务只属于一个班级，直接一个桶一个条目。
	classNameByID := map[uint64]string{}
	for _, t := range in.tasks {
		classNameByID[t.ClassID] = t.ClassName
	}
	classIDs := make([]uint64, 0, len(classNameByID))
	for id := range classNameByID {
		classIDs = append(classIDs, id)
	}
	sort.Slice(classIDs, func(i, j int) bool { return classIDs[i] < classIDs[j] })
	byClass := make([]map[string]interface{}, 0, len(classIDs))
	for _, cid := range classIDs {
		b := buckets[fmt.Sprintf("%s%d", taskStatsBucketClassPrefix, cid)]
		if b == nil {
			continue
		}
		m := in.metrics(b, now)
		row := m.toMap()
		row["class_id"] = cid
		row["class_name"] = classNameByID[cid]
		row["tasks"] = m.Tasks
		byClass = append(byClass, row)
	}
	out["by_class"] = byClass

	// 按活动：活动共享一个桶；独立任务各占一条（design D12：不能塞进「空活动」）。
	activityIDs := make([]string, 0)
	standaloneIDs := make([]uint64, 0)
	titleByActivity := map[string]string{}
	titleByTask := map[uint64]string{}
	for _, t := range in.tasks {
		if t.ActivityID != "" {
			if _, ok := titleByActivity[t.ActivityID]; !ok {
				titleByActivity[t.ActivityID] = t.Title
				activityIDs = append(activityIDs, t.ActivityID)
			}
			continue
		}
		titleByTask[t.ID] = t.Title
		standaloneIDs = append(standaloneIDs, t.ID)
	}
	sort.Strings(activityIDs)
	sort.Slice(standaloneIDs, func(i, j int) bool { return standaloneIDs[i] < standaloneIDs[j] })

	byActivity := make([]map[string]interface{}, 0, len(activityIDs)+len(standaloneIDs))
	for _, aid := range activityIDs {
		b := buckets[taskStatsBucketActPrefix+aid]
		if b == nil {
			continue
		}
		m := in.metrics(b, now)
		row := m.toMap()
		row["activity_id"] = aid
		row["task_id"] = uint64(0)
		row["title"] = titleByActivity[aid]
		row["tasks"] = m.Tasks
		byActivity = append(byActivity, row)
	}
	for _, tid := range standaloneIDs {
		b := buckets[fmt.Sprintf("%s%d", taskStatsBucketStandalonePre, tid)]
		if b == nil {
			continue
		}
		m := in.metrics(b, now)
		row := m.toMap()
		// 独立任务用任务 id 分组，activity_id 保持空串。
		row["activity_id"] = ""
		row["task_id"] = tid
		row["title"] = titleByTask[tid]
		row["tasks"] = m.Tasks
		byActivity = append(byActivity, row)
	}
	out["by_activity"] = byActivity

	return out, nil
}

// taskStatsRange 解析统计区间：缺省为当月；起止必须成对出现且 end >= start。
func taskStatsRange(startDateStr, endDateStr string) (time.Time, time.Time, error) {
	sTrim := strings.TrimSpace(startDateStr)
	eTrim := strings.TrimSpace(endDateStr)
	if (sTrim == "") != (eTrim == "") {
		return time.Time{}, time.Time{}, errors.New("BAD_DATE")
	}
	loc := shanghaiLoc()
	var rs, re time.Time
	var err error
	if sTrim == "" && eTrim == "" {
		now := time.Now().In(loc)
		y, m, _ := now.Date()
		rs = time.Date(y, m, 1, 0, 0, 0, 0, loc)
		re = rs.AddDate(0, 1, 0).Add(-time.Nanosecond)
	} else {
		rs, err = time.ParseInLocation("2006-01-02", sTrim, loc)
		if err != nil {
			return time.Time{}, time.Time{}, errors.New("BAD_DATE")
		}
		re, err = time.ParseInLocation("2006-01-02", eTrim, loc)
		if err != nil {
			return time.Time{}, time.Time{}, errors.New("BAD_DATE")
		}
		if re.Before(rs) {
			return time.Time{}, time.Time{}, errors.New("BAD_DATE_RANGE")
		}
	}
	return time.Date(rs.Year(), rs.Month(), rs.Day(), 0, 0, 0, 0, loc),
		time.Date(re.Year(), re.Month(), re.Day(), 0, 0, 0, 0, loc), nil
}
