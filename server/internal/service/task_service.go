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

func inclusiveCalendarDays(a, b time.Time, loc *time.Location) int64 {
	sa := time.Date(a.Year(), a.Month(), a.Day(), 0, 0, 0, 0, loc)
	sb := time.Date(b.Year(), b.Month(), b.Day(), 0, 0, 0, 0, loc)
	if sb.Before(sa) {
		return 0
	}
	return int64(sb.Sub(sa).Hours()/24) + 1
}

func totalPossibleCheckins(freq domain.TaskFrequency, students int64, startDate, endDate time.Time) int64 {
	if students <= 0 {
		return 0
	}
	loc := shanghaiLoc()
	days := inclusiveCalendarDays(startDate, endDate, loc)
	if days < 1 {
		days = 1
	}
	switch freq {
	case domain.TaskFrequencyDaily:
		return students * days
	case domain.TaskFrequencyWeekly:
		weeks := (days + 6) / 7
		if weeks < 1 {
			weeks = 1
		}
		return students * weeks
	case domain.TaskFrequencyOnce:
		return students
	default:
		return students * days
	}
}

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

func (s *TaskService) completionForTask(ctx context.Context, kgID uint64, t *domain.ParentTask, checkins int64) (rate string, totalPossible int64, err error) {
	students, err := s.tasks.CountActiveStudentsInClass(ctx, kgID, t.ClassID)
	if err != nil {
		return "", 0, err
	}
	tp := totalPossibleCheckins(t.Frequency, students, t.StartDate, t.EndDate)
	return formatPercent(checkins, tp), tp, nil
}

func (s *TaskService) completionForRow(ctx context.Context, kgID uint64, row repository.TaskListRow, checkins int64) (string, int64, error) {
	f := domain.TaskFrequency(row.Frequency)
	pt := &domain.ParentTask{
		ClassID:   row.ClassID,
		Frequency: f,
		StartDate: row.StartDate,
		EndDate:   row.EndDate,
	}
	return s.completionForTask(ctx, kgID, pt, checkins)
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
		ci, _ := s.tasks.CountCheckinsByTaskID(ctx, row.ID)
		rate, _, _ := s.completionForRow(ctx, kindergartenID, row, ci)
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
			// 旧口径，7.4 统一改新指标后下线。
			"completion_rate": rate,
			"creator":         map[string]interface{}{"nickname": row.CreatorNick},
			"created_at":      row.CreatedAt.UTC().Format(time.RFC3339),
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
	ci, err := s.tasks.CountCheckinsByTaskID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	rate, tp, err := s.completionForTask(ctx, kindergartenID, t, ci)
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
	return map[string]interface{}{
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
		// 旧口径，7.4 统一改新指标后下线。
		"completion_rate": rate,
		"checkin_count":   ci,
		"total_possible":  tp,
		"creator": map[string]interface{}{
			"id":       creatorID,
			"nickname": creatorNick,
			"avatar":   creatorAvatar,
		},
		"created_at": t.CreatedAt.UTC().Format(time.RFC3339),
	}, nil
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
	loc := shanghaiLoc()
	sd, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(req.StartDate), loc)
	if err != nil {
		return nil, errors.New("BAD_DATE")
	}
	ed, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(req.EndDate), loc)
	if err != nil {
		return nil, errors.New("BAD_DATE")
	}
	if ed.Before(sd) {
		return nil, errors.New("BAD_DATE_RANGE")
	}
	// 结构规则全部走 task_rules.go，不在 service 里重写一遍（D12）。
	if err := validateTargetStudent(ctx, s.db, kindergartenID, req.ClassID, req.TargetStudentID); err != nil {
		return nil, err
	}
	ritual := resolveTaskRitual(req.RecognitionRitual, "", domain.TaskRitualSticker)
	if !validTaskRitual(ritual) {
		return nil, ErrTaskRitualInvalid
	}
	starScope := resolveTaskStarScope(req.StarScope, "")
	if starScope != domain.TaskStarScopeTask && starScope != domain.TaskStarScopeClass {
		return nil, ErrTaskStarScopeInvalid
	}
	if !validTaskStarGoal(req.StarGoal) {
		return nil, ErrTaskStarGoalNegative
	}
	if starScope == domain.TaskStarScopeClass {
		if err := validateClassStarJar(ctx, s.db, kindergartenID, req.ClassID); err != nil {
			return nil, err
		}
	}
	if ritual == domain.TaskRitualClassStar && starScope == domain.TaskStarScopeTask && req.StarGoal <= 0 {
		return nil, ErrTaskStarGoalInvalid
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
		RecognitionRitual: ritual,
		StarScope:         starScope,
		StarGoal:          req.StarGoal,
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
	freq, err := parseTaskFrequency(req.Frequency)
	if err != nil {
		return err
	}
	loc := shanghaiLoc()
	sd, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(req.StartDate), loc)
	if err != nil {
		return errors.New("BAD_DATE")
	}
	ed, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(req.EndDate), loc)
	if err != nil {
		return errors.New("BAD_DATE")
	}
	if ed.Before(sd) {
		return errors.New("BAD_DATE_RANGE")
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
			return revokePublicForTask(ctx, tx, taskID)
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

func (s *TaskService) GetTaskStats(ctx context.Context, kindergartenID uint64, startDateStr, endDateStr string) (map[string]interface{}, error) {
	if kindergartenID == 0 {
		return nil, errors.New("NO_KINDERGARTEN")
	}
	sTrim := strings.TrimSpace(startDateStr)
	eTrim := strings.TrimSpace(endDateStr)
	if (sTrim == "") != (eTrim == "") {
		return nil, errors.New("BAD_DATE")
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
			return nil, errors.New("BAD_DATE")
		}
		re, err = time.ParseInLocation("2006-01-02", eTrim, loc)
		if err != nil {
			return nil, errors.New("BAD_DATE")
		}
		if re.Before(rs) {
			return nil, errors.New("BAD_DATE_RANGE")
		}
		re = time.Date(re.Year(), re.Month(), re.Day(), 23, 59, 59, 999999999, loc)
	}
	rangeStart := time.Date(rs.Year(), rs.Month(), rs.Day(), 0, 0, 0, 0, loc)
	rangeEndDay := time.Date(re.Year(), re.Month(), re.Day(), 0, 0, 0, 0, loc)

	tasks, err := s.tasks.ListTasksOverlappingRange(ctx, kindergartenID, rangeStart, rangeEndDay)
	if err != nil {
		return nil, err
	}
	var active, completed int64
	ids := make([]uint64, 0, len(tasks))
	for _, t := range tasks {
		ids = append(ids, t.ID)
		if t.Status == 1 {
			active++
		} else {
			completed++
		}
	}
	checkinMap, err := s.tasks.CountCheckinsByTaskIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	totalCheckins, err := s.tasks.CountCheckinsInDateRange(ctx, kindergartenID, rangeStart, rangeEndDay)
	if err != nil {
		return nil, err
	}

	var sumPct int64
	var nPct int64
	type classAgg struct {
		className string
		tasks     int64
		sumRate   int64
		n         int64
	}
	byClassMap := make(map[uint64]*classAgg)
	for _, row := range tasks {
		pt := &domain.ParentTask{
			ClassID:   row.ClassID,
			Frequency: domain.TaskFrequency(row.Frequency),
			StartDate: row.StartDate,
			EndDate:   row.EndDate,
		}
		ci := checkinMap[row.ID]
		_, tp, err := s.completionForTask(ctx, kindergartenID, pt, ci)
		if err != nil {
			return nil, err
		}
		var pct int64
		if tp > 0 {
			pct = ci * 100 / tp
			if pct > 100 {
				pct = 100
			}
		}
		sumPct += pct
		nPct++
		a := byClassMap[row.ClassID]
		if a == nil {
			a = &classAgg{className: row.ClassName}
			byClassMap[row.ClassID] = a
		}
		a.tasks++
		a.sumRate += pct
		a.n++
	}
	avgRate := "0%"
	if nPct > 0 {
		avgRate = fmt.Sprintf("%d%%", sumPct/nPct)
	}

	type classRow struct {
		name string
		avg  string
		t    int64
	}
	classRows := make([]classRow, 0, len(byClassMap))
	for _, a := range byClassMap {
		if a == nil || a.n == 0 {
			continue
		}
		classRows = append(classRows, classRow{
			name: a.className,
			avg:  fmt.Sprintf("%d%%", a.sumRate/a.n),
			t:    a.tasks,
		})
	}
	sort.Slice(classRows, func(i, j int) bool { return classRows[i].name < classRows[j].name })
	byClassOut := make([]map[string]interface{}, 0, len(classRows))
	for _, r := range classRows {
		byClassOut = append(byClassOut, map[string]interface{}{
			"class_name": r.name,
			"tasks":      r.t,
			"avg_rate":   r.avg,
		})
	}

	return map[string]interface{}{
		"period": map[string]interface{}{
			"start": rangeStart.Format("2006-01-02"),
			"end":   rangeEndDay.Format("2006-01-02"),
		},
		"total_tasks":         len(tasks),
		"active_tasks":        active,
		"completed_tasks":     completed,
		"total_checkins":      totalCheckins,
		"avg_completion_rate": avgRate,
		"by_class":            byClassOut,
	}, nil
}
