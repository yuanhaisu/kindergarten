package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"kindergarten-server/internal/repository"
	"kindergarten-server/pkg/pagination"
)

type OperationLogService struct {
	oplog *repository.OperationLogRepo
}

func NewOperationLogService(oplog *repository.OperationLogRepo) *OperationLogService {
	return &OperationLogService{oplog: oplog}
}

type operationLogOperator struct {
	ID       uint64 `json:"id"`
	Nickname string `json:"nickname"`
	IP       string `json:"ip"`
}

// OperationLogListItem GET /admin/logs 单条列表项（对齐 admin 前端类型）
type OperationLogListItem struct {
	ID          uint64                 `json:"id"`
	Module      string                 `json:"module"`
	ModuleLabel string                 `json:"module_label"`
	Action      string                 `json:"action"`
	ActionLabel string                 `json:"action_label"`
	Target      string                 `json:"target"`
	Operator    operationLogOperator   `json:"operator"`
	Detail      map[string]interface{} `json:"detail"`
	CreatedAt   string                 `json:"created_at"`
}

// ListOperationLogs GET /admin/logs
func (s *OperationLogService) ListOperationLogs(
	ctx context.Context,
	kindergartenID uint64,
	pq pagination.Query,
	module string,
	operatorID uint64,
	startDate, endDate string,
) ([]OperationLogListItem, int64, int, int, error) {
	if kindergartenID == 0 {
		return nil, 0, 0, 0, errors.New("NO_KINDERGARTEN")
	}
	pq.Normalize()
	dbMods := apiModuleQueryToDBModules(module)
	start, endExcl, err := parseLogDateRange(startDate, endDate)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	rows, total, err := s.oplog.ListForKindergarten(ctx, kindergartenID, pq, dbMods, operatorID, start, endExcl)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	out := make([]OperationLogListItem, 0, len(rows))
	for _, row := range rows {
		apiMod := dbModuleToAPIModule(row.Module)
		out = append(out, OperationLogListItem{
			ID:          row.ID,
			Module:      apiMod,
			ModuleLabel: moduleLabelForAPI(apiMod),
			Action:      row.Action,
			ActionLabel: actionLabelFor(row.Action),
			Target:      buildTarget(apiMod, row.Action, row.TargetType, row.TargetID, row.Detail),
			Operator: operationLogOperator{
				ID:       row.UserID,
				Nickname: strings.TrimSpace(row.OperatorNickname),
				IP:       row.IP,
			},
			Detail:    detailJSON(row.Detail),
			CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return out, total, pq.Page, pq.PageSize, nil
}

func apiModuleQueryToDBModules(q string) []string {
	q = strings.TrimSpace(strings.ToLower(q))
	if q == "" {
		return nil
	}
	switch q {
	case "student":
		return []string{"students"}
	case "teacher":
		return []string{"teachers"}
	case "parent":
		return []string{"parents"}
	case "notice":
		return []string{"notices"}
	case "growth":
		return []string{"growth"}
	case "attendance":
		return []string{"attendance"}
	case "pickup":
		return []string{"pickup"}
	case "task":
		return []string{"tasks"}
	case "points":
		return []string{"points"}
	case "achievement":
		return []string{"achievements"}
	case "announcement":
		return []string{"announcements"}
	case "admin":
		return []string{"admins", "roles"}
	case "system":
		return []string{"settings", "kindergarten"}
	case "class":
		return []string{"classes"}
	case "leave":
		return []string{"leaves"}
	case "audit":
		return []string{"audit"}
	default:
		return []string{q}
	}
}

func dbModuleToAPIModule(m string) string {
	switch strings.ToLower(strings.TrimSpace(m)) {
	case "students", "student":
		return "student"
	case "teachers", "teacher":
		return "teacher"
	case "parents", "parent":
		return "parent"
	case "notices", "notice":
		return "notice"
	case "classes", "class":
		return "class"
	case "tasks", "task":
		return "task"
	case "achievements", "achievement":
		return "achievement"
	case "announcements", "announcement":
		return "announcement"
	case "admins", "admin":
		return "admin"
	case "roles", "role":
		return "admin"
	case "settings", "kindergarten":
		return "system"
	case "leaves", "leave":
		return "leave"
	default:
		return m
	}
}

func moduleLabelForAPI(apiModule string) string {
	switch apiModule {
	case "student":
		return "学生管理"
	case "teacher":
		return "老师管理"
	case "parent":
		return "家长管理"
	case "notice":
		return "通知管理"
	case "growth":
		return "成长日记"
	case "attendance":
		return "考勤管理"
	case "pickup":
		return "接送管理"
	case "task":
		return "任务管理"
	case "points":
		return "积分管理"
	case "achievement":
		return "成就管理"
	case "announcement":
		return "公告管理"
	case "admin":
		return "管理员与权限"
	case "system":
		return "系统设置"
	case "class":
		return "班级管理"
	case "leave":
		return "请假管理"
	case "audit":
		return "内容审核"
	default:
		return apiModule
	}
}

func actionLabelFor(action string) string {
	switch strings.TrimSpace(action) {
	case "create":
		return "创建"
	case "update", "update_settings", "update_info":
		return "编辑"
	case "delete":
		return "删除"
	case "audit", "decide":
		return "审核"
	case "adjust":
		return "调整"
	case "import":
		return "导入"
	case "update_status":
		return "状态变更"
	case "reset_password":
		return "重置密码"
	case "create_academic_year":
		return "创建学年"
	default:
		return action
	}
}

func detailJSON(detail string) map[string]interface{} {
	s := strings.TrimSpace(detail)
	if s == "" {
		return map[string]interface{}{}
	}
	var m map[string]interface{}
	if json.Unmarshal([]byte(s), &m) == nil && m != nil {
		return m
	}
	return map[string]interface{}{"text": s}
}

func buildTarget(apiModule, action, targetType string, targetID uint64, detail string) string {
	var m map[string]interface{}
	_ = json.Unmarshal([]byte(strings.TrimSpace(detail)), &m)
	if m != nil {
		if s, ok := m["target"].(string); ok && strings.TrimSpace(s) != "" {
			return s
		}
		switch apiModule {
		case "student":
			name, _ := m["name"].(string)
			sn, _ := m["student_no"].(string)
			if name != "" {
				if sn != "" {
					return fmt.Sprintf("学生 %s（%s）", name, sn)
				}
				return "学生 " + name
			}
		case "teacher":
			if n, ok := m["name"].(string); ok && n != "" {
				return "老师 " + n
			}
			if n, ok := m["nickname"].(string); ok && n != "" {
				return "老师 " + n
			}
		case "parent":
			if n, ok := m["nickname"].(string); ok && n != "" {
				return "家长 " + n
			}
		case "notice", "announcement", "growth", "task":
			if n, ok := m["title"].(string); ok && n != "" {
				return n
			}
		}
	}
	if targetType != "" && targetID > 0 {
		return fmt.Sprintf("%s #%d", targetType, targetID)
	}
	if action != "" {
		return action
	}
	return ""
}

func parseLogDateRange(startDate, endDate string) (start *time.Time, endExclusive *time.Time, err error) {
	sd := strings.TrimSpace(startDate)
	ed := strings.TrimSpace(endDate)
	if sd == "" && ed == "" {
		return nil, nil, nil
	}
	parseDay := func(s string) (time.Time, error) {
		return time.ParseInLocation("2006-01-02", s, time.UTC)
	}
	if sd != "" && ed != "" {
		s, e1 := parseDay(sd)
		e, e2 := parseDay(ed)
		if e1 != nil || e2 != nil {
			return nil, nil, errors.New("BAD_DATE_RANGE")
		}
		if e.Before(s) {
			return nil, nil, errors.New("BAD_DATE_RANGE")
		}
		endEx := e.Add(24 * time.Hour)
		return &s, &endEx, nil
	}
	if sd != "" {
		s, e1 := parseDay(sd)
		if e1 != nil {
			return nil, nil, errors.New("BAD_DATE_RANGE")
		}
		return &s, nil, nil
	}
	e, e2 := parseDay(ed)
	if e2 != nil {
		return nil, nil, errors.New("BAD_DATE_RANGE")
	}
	endEx := e.Add(24 * time.Hour)
	return nil, &endEx, nil
}
