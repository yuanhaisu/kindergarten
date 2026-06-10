package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/pkg/pagination"
)

type TaskService struct {
	tasks *repository.TaskRepo
	oplog *repository.OperationLogRepo
}

func NewTaskService(tasks *repository.TaskRepo, oplog *repository.OperationLogRepo) *TaskService {
	return &TaskService{tasks: tasks, oplog: oplog}
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
			"id":               row.ID,
			"title":            row.Title,
			"class_name":       row.ClassName,
			"class_id":         row.ClassID,
			"frequency":        string(f),
			"frequency_label":  taskFrequencyLabel(f),
			"start_date":       row.StartDate.In(shanghaiLoc()).Format("2006-01-02"),
			"end_date":         row.EndDate.In(shanghaiLoc()).Format("2006-01-02"),
			"points":           row.Points,
			"status":           int(row.Status),
			"completion_rate":  rate,
			"creator":          map[string]interface{}{"nickname": row.CreatorNick},
			"created_at":       row.CreatedAt.UTC().Format(time.RFC3339),
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
		"id":               t.ID,
		"title":            t.Title,
		"description":      t.Description,
		"class_id":         t.ClassID,
		"class_name":       className,
		"frequency":        string(t.Frequency),
		"frequency_label":  taskFrequencyLabel(t.Frequency),
		"start_date":       t.StartDate.In(shanghaiLoc()).Format("2006-01-02"),
		"end_date":         t.EndDate.In(shanghaiLoc()).Format("2006-01-02"),
		"points":           t.Points,
		"status":           int(t.Status),
		"completion_rate":  rate,
		"checkin_count":    ci,
		"total_possible":   tp,
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
	n, err := s.tasks.SoftDeleteTask(ctx, kindergartenID, taskID)
	if err != nil {
		return err
	}
	if n == 0 {
		return errTaskNotFound
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

// CreateTaskReq POST /admin/tasks
type CreateTaskReq struct {
	Title       string `json:"title" binding:"required,max=200"`
	Description string `json:"description"`
	ClassID     uint64 `json:"class_id" binding:"required"`
	Frequency   string `json:"frequency" binding:"required"`
	StartDate   string `json:"start_date" binding:"required"`
	EndDate     string `json:"end_date" binding:"required"`
	Points      int    `json:"points"`
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
	ok, err := s.tasks.ClassBelongsToKindergarten(ctx, kindergartenID, req.ClassID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("CLASS_NOT_FOUND")
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
	points := req.Points
	if points <= 0 {
		points = 1
	}
	t := &domain.ParentTask{
		KindergartenID: kindergartenID,
		ClassID:        req.ClassID,
		CreatorID:      adminUserID,
		Title:          strings.TrimSpace(req.Title),
		Description:    strings.TrimSpace(req.Description),
		Frequency:      freq,
		StartDate:      sd,
		EndDate:        ed,
		Points:         points,
		Status:         1,
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
	return map[string]interface{}{
		"id":         t.ID,
		"created_at": t.CreatedAt.UTC().Format(time.RFC3339),
	}, nil
}

// UpdateTaskReq PUT /admin/tasks/:id（全量字段）
type UpdateTaskReq struct {
	Title       string `json:"title" binding:"required,max=200"`
	Description string `json:"description"`
	ClassID     uint64 `json:"class_id" binding:"required"`
	Frequency   string `json:"frequency" binding:"required"`
	StartDate   string `json:"start_date" binding:"required"`
	EndDate     string `json:"end_date" binding:"required"`
	Points      int    `json:"points"`
	Status      int8   `json:"status" binding:"required"` // 0 已结束 1 进行中
}

func (s *TaskService) UpdateTask(
	ctx context.Context,
	kindergartenID, adminUserID, taskID uint64,
	req UpdateTaskReq,
	ip, ua string,
) error {
	if kindergartenID == 0 {
		return errors.New("NO_KINDERGARTEN")
	}
	cur, err := s.tasks.FindTaskByID(ctx, kindergartenID, taskID)
	if err != nil {
		return err
	}
	if cur == nil {
		return errTaskNotFound
	}
	ok, err := s.tasks.ClassBelongsToKindergarten(ctx, kindergartenID, req.ClassID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("CLASS_NOT_FOUND")
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
	if req.Status != 0 && req.Status != 1 {
		return errors.New("BAD_STATUS")
	}
	points := req.Points
	if points <= 0 {
		points = 1
	}
	updates := map[string]interface{}{
		"title":        strings.TrimSpace(req.Title),
		"description":  strings.TrimSpace(req.Description),
		"class_id":     req.ClassID,
		"frequency":    string(freq),
		"start_date":   sd,
		"end_date":     ed,
		"points":       points,
		"status":       req.Status,
	}
	n, err := s.tasks.UpdateTask(ctx, kindergartenID, taskID, updates)
	if err != nil {
		return err
	}
	if n == 0 {
		return errTaskNotFound
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
	totalPoints, err := s.tasks.SumCheckinPointsInDateRange(ctx, kindergartenID, rangeStart, rangeEndDay)
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
		"total_tasks":          len(tasks),
		"active_tasks":         active,
		"completed_tasks":      completed,
		"total_checkins":       totalCheckins,
		"total_points_awarded": totalPoints,
		"avg_completion_rate":  avgRate,
		"by_class":             byClassOut,
	}, nil
}
