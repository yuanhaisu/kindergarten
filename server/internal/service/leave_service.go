package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/pkg/pagination"
)

type LeaveService struct {
	leaves *repository.LeaveRepo
	oplog  *repository.OperationLogRepo
}

func NewLeaveService(leaves *repository.LeaveRepo, oplog *repository.OperationLogRepo) *LeaveService {
	return &LeaveService{leaves: leaves, oplog: oplog}
}

var (
	errLeaveNotFound   = errors.New("LEAVE_NOT_FOUND")
	errLeaveNotPending = errors.New("LEAVE_NOT_PENDING")
	errBadAuditStatus  = errors.New("BAD_AUDIT_STATUS")
)

func IsLeaveClientErr(err error) bool {
	return errors.Is(err, errLeaveNotPending) || errors.Is(err, errBadAuditStatus)
}

// IsLeaveNotFound 请假记录不存在（园所内）。
func IsLeaveNotFound(err error) bool {
	return errors.Is(err, errLeaveNotFound)
}

func leaveStatusLabel(s domain.LeaveStatus) string {
	switch s {
	case domain.LeaveStatusPending:
		return "待审批"
	case domain.LeaveStatusApproved:
		return "已同意"
	case domain.LeaveStatusRejected:
		return "已驳回"
	case domain.LeaveStatusCanceled:
		return "已取消"
	default:
		return "未知"
	}
}

func formatDateOnly(t time.Time) string {
	return t.In(shanghaiLoc()).Format("2006-01-02")
}

func leaveListItemFromRow(r repository.LeaveListRow) map[string]interface{} {
	student := map[string]interface{}{
		"id":         r.StudentID,
		"name":       r.StudentName,
		"class_name": r.ClassName,
	}
	applicant := map[string]interface{}{
		"nickname": r.ApplicantNick,
		"relation": r.ApplicantRel,
	}
	out := map[string]interface{}{
		"id":           r.ID,
		"student":      student,
		"applicant":    applicant,
		"reason":       r.Reason,
		"start_date":   formatDateOnly(r.StartDate),
		"end_date":     formatDateOnly(r.EndDate),
		"days":         r.Days,
		"status":       int(r.Status),
		"status_label": leaveStatusLabel(r.Status),
		"created_at":   r.CreatedAt.UTC().Format(time.RFC3339),
	}
	if r.ReviewedAt != nil {
		out["reviewed_at"] = r.ReviewedAt.UTC().Format(time.RFC3339)
	}
	if r.ReviewerID != nil && *r.ReviewerID > 0 && r.ReviewerNick != "" {
		out["reviewer"] = map[string]interface{}{"nickname": r.ReviewerNick}
	}
	return out
}

func (s *LeaveService) ListLeaves(
	ctx context.Context,
	kindergartenID uint64,
	pq pagination.Query,
	classID *uint64,
	status *int8,
	startDateStr, endDateStr string,
) ([]map[string]interface{}, int64, int, int, error) {
	if kindergartenID == 0 {
		return nil, 0, 0, 0, errors.New("NO_KINDERGARTEN")
	}
	pn := pq
	pn.Normalize()
	off := pn.Offset()
	var st *domain.LeaveStatus
	if status != nil {
		ls := domain.LeaveStatus(*status)
		st = &ls
	}
	var rs, re *time.Time
	if strings.TrimSpace(startDateStr) != "" && strings.TrimSpace(endDateStr) != "" {
		a, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(startDateStr), shanghaiLoc())
		if err != nil {
			return nil, 0, 0, 0, errors.New("BAD_DATE")
		}
		b, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(endDateStr), shanghaiLoc())
		if err != nil {
			return nil, 0, 0, 0, errors.New("BAD_DATE")
		}
		if b.Before(a) {
			return nil, 0, 0, 0, errors.New("BAD_DATE_RANGE")
		}
		startDay := time.Date(a.Year(), a.Month(), a.Day(), 0, 0, 0, 0, shanghaiLoc())
		endDay := time.Date(b.Year(), b.Month(), b.Day(), 23, 59, 59, 999999999, shanghaiLoc())
		rs = &startDay
		re = &endDay
	}
	rows, total, err := s.leaves.List(ctx, kindergartenID, classID, st, rs, re, off, pn.PageSize)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	out := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		out = append(out, leaveListItemFromRow(r))
	}
	return out, total, pn.Page, pn.PageSize, nil
}

// LeaveStatsReasonRow 统计接口 by_reason 单项。
type LeaveStatsReasonRow struct {
	Reason string `json:"reason"`
	Count  int64  `json:"count"`
}

// LeaveStatsResponse GET /admin/leaves/stats
type LeaveStatsResponse struct {
	Period struct {
		Start string `json:"start"`
		End   string `json:"end"`
	} `json:"period"`
	Total    int64                    `json:"total"`
	Pending  int64                    `json:"pending"`
	Approved int64                    `json:"approved"`
	Rejected int64                    `json:"rejected"`
	ByClass  []map[string]interface{} `json:"by_class"`
	ByReason []LeaveStatsReasonRow    `json:"by_reason"`
}

func categorizeLeaveReason(raw string) string {
	r := strings.TrimSpace(raw)
	if r == "" {
		return "其他"
	}
	if strings.Contains(r, "家事") || strings.Contains(r, "事假") {
		return "家事"
	}
	if strings.Contains(r, "病") || strings.Contains(r, "感冒") || strings.Contains(r, "发烧") || strings.Contains(r, "医") {
		return "生病"
	}
	return "其他"
}

func (s *LeaveService) GetLeaveStats(
	ctx context.Context,
	kindergartenID uint64,
	startDateStr, endDateStr string,
) (*LeaveStatsResponse, error) {
	if kindergartenID == 0 {
		return nil, errors.New("NO_KINDERGARTEN")
	}
	loc := shanghaiLoc()
	var start, end time.Time
	if strings.TrimSpace(startDateStr) == "" || strings.TrimSpace(endDateStr) == "" {
		now := time.Now().In(loc)
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		end = start.AddDate(0, 1, 0).Add(-time.Nanosecond)
	} else {
		a, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(startDateStr), loc)
		if err != nil {
			return nil, errors.New("BAD_DATE")
		}
		b, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(endDateStr), loc)
		if err != nil {
			return nil, errors.New("BAD_DATE")
		}
		if b.Before(a) {
			return nil, errors.New("BAD_DATE_RANGE")
		}
		start = time.Date(a.Year(), a.Month(), a.Day(), 0, 0, 0, 0, loc)
		end = time.Date(b.Year(), b.Month(), b.Day(), 23, 59, 59, 999999999, loc)
	}

	pending, approved, rejected, _, total, err := s.leaves.CountByStatusInRange(ctx, kindergartenID, start, end)
	if err != nil {
		return nil, err
	}

	byClassRows, err := s.leaves.CountByClassInRange(ctx, kindergartenID, start, end)
	if err != nil {
		return nil, err
	}
	byClass := make([]map[string]interface{}, 0, len(byClassRows))
	for _, r := range byClassRows {
		byClass = append(byClass, map[string]interface{}{
			"class_name": r.ClassName,
			"count":      r.Cnt,
		})
	}

	reasonRows, err := s.leaves.CountReasonsInRange(ctx, kindergartenID, start, end)
	if err != nil {
		return nil, err
	}
	buckets := map[string]int64{"生病": 0, "家事": 0, "其他": 0}
	for _, rc := range reasonRows {
		cat := categorizeLeaveReason(rc.Reason)
		buckets[cat] += rc.Cnt
	}
	byReason := []LeaveStatsReasonRow{
		{Reason: "生病", Count: buckets["生病"]},
		{Reason: "家事", Count: buckets["家事"]},
		{Reason: "其他", Count: buckets["其他"]},
	}

	out := &LeaveStatsResponse{
		Total:    total,
		Pending:  pending,
		Approved: approved,
		Rejected: rejected,
		ByClass:  byClass,
		ByReason: byReason,
	}
	out.Period.Start = start.Format("2006-01-02")
	out.Period.End = end.Format("2006-01-02")
	return out, nil
}

// AuditLeaveReq PUT /admin/leaves/:id/audit
type AuditLeaveReq struct {
	Status       int8   `json:"status" binding:"required"` // 1 同意 2 驳回
	ReviewRemark string `json:"review_remark"`
}

func (s *LeaveService) AuditLeave(
	ctx context.Context,
	kindergartenID, adminUserID, leaveID uint64,
	req AuditLeaveReq,
	ip, ua string,
) error {
	if kindergartenID == 0 {
		return errors.New("NO_KINDERGARTEN")
	}
	var target domain.LeaveStatus
	switch req.Status {
	case int8(domain.LeaveStatusApproved):
		target = domain.LeaveStatusApproved
	case int8(domain.LeaveStatusRejected):
		target = domain.LeaveStatusRejected
	default:
		return errBadAuditStatus
	}
	lr, err := s.leaves.FindByID(ctx, kindergartenID, leaveID)
	if err != nil {
		return err
	}
	if lr == nil {
		return errLeaveNotFound
	}
	if lr.Status != domain.LeaveStatusPending {
		return errLeaveNotPending
	}
	now := time.Now().UTC()
	if err := s.leaves.UpdateAudit(ctx, kindergartenID, leaveID, target, adminUserID, strings.TrimSpace(req.ReviewRemark), now); err != nil {
		return err
	}
	detail := fmt.Sprintf("status=%d", req.Status)
	_ = s.oplog.Create(ctx, &domain.OperationLog{
		KindergartenID: kindergartenID,
		UserID:         adminUserID,
		Module:         "leaves",
		Action:         "audit",
		TargetType:     "leave_request",
		TargetID:       leaveID,
		Detail:         detail,
		IP:             ip,
		UserAgent:      truncateUA(ua),
	})
	return nil
}
