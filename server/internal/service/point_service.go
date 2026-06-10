package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/pkg/pagination"
)

// 与 settings 默认 ExtraConfig 对齐；含 growth_featured_points 供积分规则 id=4。
const defaultAppExtraConfigJSON = `{"attendance":{"arrive_start_time":"07:30","arrive_end_time":"09:00","pickup_start_time":"16:30","late_threshold_minutes":15},"notice":{"need_audit":true,"auto_remind_hours":24},"growth":{"need_audit":false,"allow_parent_comment":true,"allow_parent_like":true},"pickup":{"qr_code_expire_hours":24,"need_authorization_audit":true},"points":{"task_checkin_points":5,"weekly_attendance_bonus":10,"monthly_attendance_bonus":50,"growth_featured_points":20}}`

type PointService struct {
	points *repository.PointRepo
	app    *repository.KindergartenAppSettingRepo
	oplog  *repository.OperationLogRepo
}

func NewPointService(
	points *repository.PointRepo,
	app *repository.KindergartenAppSettingRepo,
	oplog *repository.OperationLogRepo,
) *PointService {
	return &PointService{points: points, app: app, oplog: oplog}
}

func pointSourceLabel(src string) string {
	switch src {
	case "task_checkin":
		return "任务打卡"
	case "admin_adjust":
		return "管理员调整"
	case "full_attendance_week":
		return "周全勤"
	case "full_attendance_month":
		return "月全勤"
	case "growth_featured":
		return "成长日志被推荐"
	default:
		return src
	}
}

func monthBoundsShanghai(now time.Time) (from, toExclusive time.Time) {
	loc := shanghaiLoc()
	n := now.In(loc)
	y, m, _ := n.Date()
	from = time.Date(y, m, 1, 0, 0, 0, 0, loc)
	toExclusive = from.AddDate(0, 1, 0)
	return from, toExclusive
}

func (s *PointService) GetOverview(ctx context.Context, kindergartenID uint64) (map[string]interface{}, error) {
	if kindergartenID == 0 {
		return nil, errors.New("NO_KINDERGARTEN")
	}
	circ, err := s.points.SumAvailablePoints(ctx, kindergartenID)
	if err != nil {
		return nil, err
	}
	from, toEx := monthBoundsShanghai(time.Now())
	awarded, err := s.points.SumPositiveTxInRange(ctx, kindergartenID, from, toEx)
	if err != nil {
		return nil, err
	}
	spent, err := s.points.SumNegativeTxAbsInRange(ctx, kindergartenID, from, toEx)
	if err != nil {
		return nil, err
	}
	top, err := s.points.TopEarnersByBalance(ctx, kindergartenID, 10)
	if err != nil {
		return nil, err
	}
	topOut := make([]map[string]interface{}, 0, len(top))
	for _, r := range top {
		if r.Balance <= 0 {
			continue
		}
		topOut = append(topOut, map[string]interface{}{
			"student": map[string]interface{}{
				"id":         r.StudentID,
				"name":       r.StudentName,
				"class_name": r.ClassName,
			},
			"points": r.Balance,
		})
	}
	return map[string]interface{}{
		"total_points_in_circulation":       circ,
		"total_points_awarded_this_month": awarded,
		"total_points_spent_this_month":   spent,
		"top_earners":                       topOut,
	}, nil
}

func (s *PointService) ListRecords(
	ctx context.Context,
	kindergartenID uint64,
	pq pagination.Query,
	studentID *uint64,
	classID *uint64,
	typeStr string,
) ([]map[string]interface{}, int64, int, int, error) {
	if kindergartenID == 0 {
		return nil, 0, 0, 0, errors.New("NO_KINDERGARTEN")
	}
	var typeEarn *bool
	switch strings.ToLower(strings.TrimSpace(typeStr)) {
	case "earn":
		b := true
		typeEarn = &b
	case "spend":
		b := false
		typeEarn = &b
	}
	pn := pq
	pn.Normalize()
	offset := pn.Offset()
	rows, total, err := s.points.ListTransactions(ctx, kindergartenID, studentID, classID, typeEarn, offset, pn.PageSize)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	out := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		typ := "earn"
		if row.Points < 0 {
			typ = "spend"
		}
		out = append(out, map[string]interface{}{
			"id": row.ID,
			"student": map[string]interface{}{
				"id":         row.StudentID,
				"name":       row.StudentName,
				"class_name": row.ClassName,
			},
			"type":          typ,
			"source":        row.Source,
			"source_label":  pointSourceLabel(row.Source),
			"amount":        row.Points,
			"description":   row.Description,
			"operator":      nil,
			"created_at":    row.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return out, total, pn.Page, pn.PageSize, nil
}

type AdjustPointsReq struct {
	StudentID uint64 `json:"student_id" binding:"required"`
	Type      string `json:"type" binding:"required"` // earn / spend
	Amount    int    `json:"amount" binding:"required"`
	Reason    string `json:"reason" binding:"required"`
}

func (s *PointService) AdjustPoints(
	ctx context.Context,
	kindergartenID, adminUserID uint64,
	req AdjustPointsReq,
	ip, ua string,
) (map[string]interface{}, error) {
	if kindergartenID == 0 {
		return nil, errors.New("NO_KINDERGARTEN")
	}
	ok, err := s.points.StudentExistsInKindergarten(ctx, kindergartenID, req.StudentID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("STUDENT_NOT_FOUND")
	}
	var earn bool
	switch strings.ToLower(strings.TrimSpace(req.Type)) {
	case "earn":
		earn = true
	case "spend":
		earn = false
	default:
		return nil, errors.New("BAD_ADJUST_TYPE")
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return nil, errors.New("BAD_REASON")
	}
	txnID, balance, err := s.points.ApplyAdjustTx(ctx, req.StudentID, earn, req.Amount, reason)
	if err != nil {
		if errors.Is(err, repository.ErrInsufficientPointBalance) {
			return nil, err
		}
		if err.Error() == "BAD_POINT_AMOUNT" {
			return nil, err
		}
		return nil, err
	}
	_ = s.oplog.Create(ctx, &domain.OperationLog{
		KindergartenID: kindergartenID,
		UserID:         adminUserID,
		Module:         "points",
		Action:         "adjust",
		TargetType:     "point_transaction",
		TargetID:       txnID,
		Detail:         fmt.Sprintf("student_id=%d type=%s", req.StudentID, req.Type),
		IP:             ip,
		UserAgent:      truncateUA(ua),
	})
	now := time.Now().UTC()
	return map[string]interface{}{
		"id":         txnID,
		"balance":    balance,
		"created_at": now.Format(time.RFC3339),
	}, nil
}

func readPointsFromExtra(extra string) (taskIn, weekB, monthB, growthF int) {
	var root map[string]interface{}
	raw := strings.TrimSpace(extra)
	if raw == "" {
		raw = defaultAppExtraConfigJSON
	}
	if err := json.Unmarshal([]byte(raw), &root); err != nil {
		_ = json.Unmarshal([]byte(defaultAppExtraConfigJSON), &root)
	}
	pts, _ := root["points"].(map[string]interface{})
	if pts == nil {
		return 5, 10, 50, 20
	}
	taskIn = intFromAny(pts["task_checkin_points"], 5)
	weekB = intFromAny(pts["weekly_attendance_bonus"], 10)
	monthB = intFromAny(pts["monthly_attendance_bonus"], 50)
	growthF = intFromAny(pts["growth_featured_points"], 20)
	if growthF <= 0 {
		growthF = 20
	}
	return taskIn, weekB, monthB, growthF
}

func intFromAny(v interface{}, def int) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	default:
		return def
	}
}

func (s *PointService) GetPointRules(ctx context.Context, kindergartenID uint64) (map[string]interface{}, error) {
	if kindergartenID == 0 {
		return nil, errors.New("NO_KINDERGARTEN")
	}
	row, err := s.app.FindByKindergartenID(ctx, kindergartenID)
	if err != nil {
		return nil, err
	}
	extra := ""
	if row != nil {
		extra = row.ExtraConfig
	}
	ti, wb, mb, gf := readPointsFromExtra(extra)
	rules := []map[string]interface{}{
		{"id": 1, "source": "task_checkin", "label": "任务打卡", "points": ti, "editable": true},
		{"id": 2, "source": "full_attendance_week", "label": "周全勤", "points": wb, "editable": true},
		{"id": 3, "source": "full_attendance_month", "label": "月全勤", "points": mb, "editable": true},
		{"id": 4, "source": "growth_featured", "label": "成长日志被推荐", "points": gf, "editable": true},
	}
	return map[string]interface{}{"rules": rules}, nil
}

type UpdatePointRulesReq struct {
	Rules []struct {
		ID     int `json:"id" binding:"required"`
		Points int `json:"points" binding:"required"`
	} `json:"rules" binding:"required"`
}

func (s *PointService) UpdatePointRules(
	ctx context.Context,
	kindergartenID, adminUserID uint64,
	req UpdatePointRulesReq,
	ip, ua string,
) (map[string]interface{}, error) {
	if kindergartenID == 0 {
		return nil, errors.New("NO_KINDERGARTEN")
	}
	patch := make(map[string]int)
	for _, r := range req.Rules {
		if r.Points < 0 {
			return nil, errors.New("BAD_POINTS")
		}
		switch r.ID {
		case 1:
			patch["task_checkin_points"] = r.Points
		case 2:
			patch["weekly_attendance_bonus"] = r.Points
		case 3:
			patch["monthly_attendance_bonus"] = r.Points
		case 4:
			patch["growth_featured_points"] = r.Points
		default:
			return nil, errors.New("BAD_RULE_ID")
		}
	}
	row, err := s.app.FindByKindergartenID(ctx, kindergartenID)
	if err != nil {
		return nil, err
	}
	extra := defaultAppExtraConfigJSON
	if row != nil && strings.TrimSpace(row.ExtraConfig) != "" {
		extra = row.ExtraConfig
	}
	newExtra, err := mergePointsPatchIntoExtra(extra, patch)
	if err != nil {
		return nil, err
	}
	if row == nil {
		row = &domain.KindergartenAppSetting{
			KindergartenID: kindergartenID,
			CurrentSemester: "spring",
			ExtraConfig:     newExtra,
		}
		if err := s.app.Create(ctx, row); err != nil {
			return nil, err
		}
	} else {
		row.ExtraConfig = newExtra
		if err := s.app.Save(ctx, row); err != nil {
			return nil, err
		}
	}
	_ = s.oplog.Create(ctx, &domain.OperationLog{
		KindergartenID: kindergartenID,
		UserID:         adminUserID,
		Module:         "points",
		Action:         "update_rules",
		TargetType:     "kindergarten_app_setting",
		TargetID:       row.ID,
		Detail:         fmt.Sprintf("count=%d", len(req.Rules)),
		IP:             ip,
		UserAgent:      truncateUA(ua),
	})
	now := time.Now().UTC()
	return map[string]interface{}{
		"updated_count": len(req.Rules),
		"updated_at":    now.Format(time.RFC3339),
	}, nil
}

func mergePointsPatchIntoExtra(extra string, patch map[string]int) (string, error) {
	var root map[string]interface{}
	src := strings.TrimSpace(extra)
	if src == "" {
		src = defaultAppExtraConfigJSON
	}
	if err := json.Unmarshal([]byte(src), &root); err != nil {
		root = nil
		if err2 := json.Unmarshal([]byte(defaultAppExtraConfigJSON), &root); err2 != nil {
			return "", err
		}
	}
	pts, _ := root["points"].(map[string]interface{})
	if pts == nil {
		pts = make(map[string]interface{})
	}
	for k, v := range patch {
		pts[k] = v
	}
	root["points"] = pts
	b, err := json.Marshal(root)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
