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

const (
	pickupAuthReviewPending  int8 = 0
	pickupAuthReviewApproved int8 = 1
	pickupAuthReviewRejected int8 = 2
)

type PickupService struct {
	pickup   *repository.PickupRepo
	students *repository.StudentRepo
	classes  *repository.ClassRepo
	oplog    *repository.OperationLogRepo
}

func NewPickupService(
	pickup *repository.PickupRepo,
	students *repository.StudentRepo,
	classes *repository.ClassRepo,
	oplog *repository.OperationLogRepo,
) *PickupService {
	return &PickupService{pickup: pickup, students: students, classes: classes, oplog: oplog}
}

var (
	errPickupAuthNotFound   = errors.New("PICKUP_AUTH_NOT_FOUND")
	errPickupAuthNotPending = errors.New("PICKUP_AUTH_NOT_PENDING")
	errBadPickupAction      = errors.New("BAD_ACTION")
)

func IsPickupAuthNotFound(err error) bool {
	return errors.Is(err, errPickupAuthNotFound)
}

func IsPickupAuthNotPendingErr(err error) bool {
	return errors.Is(err, errPickupAuthNotPending)
}

func IsBadPickupAction(err error) bool {
	return errors.Is(err, errBadPickupAction)
}

func pickupTypeLabel(t domain.PickupType) string {
	switch t {
	case domain.PickupTypeDropOff:
		return "送入"
	case domain.PickupTypePickUp:
		return "接走"
	default:
		return "未知"
	}
}

func pickupMethodLabel(m domain.PickupMethod) string {
	switch m {
	case domain.PickupMethodScan:
		return "扫码"
	case domain.PickupMethodManual:
		return "手动"
	default:
		return "未知"
	}
}

func parsePickupDate(dateStr string) (time.Time, error) {
	s := strings.TrimSpace(dateStr)
	loc := shanghaiLoc()
	if s == "" {
		now := time.Now().In(loc)
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc), nil
	}
	t, err := time.ParseInLocation("2006-01-02", s, loc)
	if err != nil {
		return time.Time{}, errors.New("BAD_DATE")
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc), nil
}

func (s *PickupService) GetPickupOverview(ctx context.Context, kindergartenID uint64, dateStr string) (map[string]interface{}, error) {
	if kindergartenID == 0 {
		return nil, errors.New("NO_KINDERGARTEN")
	}
	day, err := parsePickupDate(dateStr)
	if err != nil {
		return nil, err
	}
	totalStudents, err := s.students.CountActiveInKindergarten(ctx, kindergartenID)
	if err != nil {
		return nil, err
	}
	arrived, err := s.pickup.CountDistinctStudentsByType(ctx, kindergartenID, day, domain.PickupTypeDropOff)
	if err != nil {
		return nil, err
	}
	pickedUp, err := s.pickup.CountDistinctStudentsByType(ctx, kindergartenID, day, domain.PickupTypePickUp)
	if err != nil {
		return nil, err
	}
	notPickedUp, err := s.pickup.CountStudentsDroppedButNotPicked(ctx, kindergartenID, day)
	if err != nil {
		return nil, err
	}
	abnormalCount, err := s.pickup.CountAbnormalRecords(ctx, kindergartenID, day)
	if err != nil {
		return nil, err
	}
	aggRows, err := s.pickup.AggregateByClass(ctx, kindergartenID, day)
	if err != nil {
		return nil, err
	}
	aggByClass := make(map[uint64]repository.ClassPickupAgg)
	for _, a := range aggRows {
		aggByClass[a.ClassID] = a
	}
	classList, _, err := s.classes.List(ctx, kindergartenID, "", "", 0, 5000)
	if err != nil {
		return nil, err
	}
	byClass := make([]map[string]interface{}, 0, len(classList))
	for _, c := range classList {
		a, ok := aggByClass[c.ID]
		if !ok {
			byClass = append(byClass, map[string]interface{}{
				"class_id":      c.ID,
				"class_name":    c.Name,
				"arrived":       int64(0),
				"picked_up":     int64(0),
				"not_picked_up": int64(0),
			})
			continue
		}
		byClass = append(byClass, map[string]interface{}{
			"class_id":      c.ID,
			"class_name":    c.Name,
			"arrived":       a.Arrived,
			"picked_up":     a.PickedUp,
			"not_picked_up": a.NotPickedUp,
		})
	}

	return map[string]interface{}{
		"date":           day.Format("2006-01-02"),
		"total_students": totalStudents,
		"arrived":        arrived,
		"picked_up":      pickedUp,
		"not_picked_up":  notPickedUp,
		"abnormal_count": abnormalCount,
		"by_class":       byClass,
	}, nil
}

func recordToMap(row repository.PickupRecordListRow) map[string]interface{} {
	out := map[string]interface{}{
		"id":           row.ID,
		"student":      map[string]interface{}{"id": row.StudentID, "name": row.StudentName, "class_name": row.ClassName},
		"type":         int(row.Type),
		"type_label":   pickupTypeLabel(row.Type),
		"method":       int(row.Method),
		"method_label": pickupMethodLabel(row.Method),
		"pickup_name":  row.PickupName,
		"is_abnormal":  row.IsAbnormal,
		"recorded_at":  row.RecordedAt.UTC().Format(time.RFC3339),
	}
	if strings.TrimSpace(row.OperatorNick) != "" {
		out["operator"] = map[string]interface{}{"nickname": row.OperatorNick}
	}
	return out
}

func (s *PickupService) ListPickupRecords(
	ctx context.Context,
	kindergartenID uint64,
	pq pagination.Query,
	classID *uint64,
	dateStr string,
	abnormal *bool,
) ([]map[string]interface{}, int64, int, int, error) {
	if kindergartenID == 0 {
		return nil, 0, 0, 0, errors.New("NO_KINDERGARTEN")
	}
	day, err := parsePickupDate(dateStr)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	pn := pq
	pn.Normalize()
	off := pn.Offset()
	rows, total, err := s.pickup.ListRecords(ctx, kindergartenID, classID, day, abnormal, off, pn.PageSize)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	out := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		out = append(out, recordToMap(r))
	}
	return out, total, pn.Page, pn.PageSize, nil
}

func abnormalRowToMap(row repository.PickupRecordListRow) map[string]interface{} {
	m := map[string]interface{}{
		"id": row.ID,
		"student": map[string]interface{}{
			"id": row.StudentID, "name": row.StudentName, "class_name": row.ClassName,
		},
		"type":            int(row.Type),
		"type_label":      pickupTypeLabel(row.Type),
		"pickup_name":     row.PickupName,
		"is_authorized":   row.PickupPersonID != nil,
		"abnormal_reason": row.AbnormalReason,
		"recorded_at":     row.RecordedAt.UTC().Format(time.RFC3339),
	}
	if strings.TrimSpace(row.OperatorNick) != "" {
		m["operator"] = map[string]interface{}{"nickname": row.OperatorNick}
	}
	return m
}

func (s *PickupService) ListPickupAbnormal(
	ctx context.Context,
	kindergartenID uint64,
	pq pagination.Query,
	dateStr string,
) ([]map[string]interface{}, int64, int, int, error) {
	if kindergartenID == 0 {
		return nil, 0, 0, 0, errors.New("NO_KINDERGARTEN")
	}
	var dayPtr *time.Time
	if strings.TrimSpace(dateStr) != "" {
		d, err := parsePickupDate(dateStr)
		if err != nil {
			return nil, 0, 0, 0, err
		}
		dayPtr = &d
	}
	pn := pq
	pn.Normalize()
	off := pn.Offset()
	rows, total, err := s.pickup.ListAbnormalRecords(ctx, kindergartenID, dayPtr, off, pn.PageSize)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	out := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		out = append(out, abnormalRowToMap(r))
	}
	return out, total, pn.Page, pn.PageSize, nil
}

// AuditPickupAuthReq PUT 审核授权接送人
type AuditPickupAuthReq struct {
	Action string `json:"action" binding:"required"` // approve / reject
	Remark string `json:"remark"`
}

func (s *PickupService) AuditPickupAuthorization(
	ctx context.Context,
	kindergartenID, adminUserID, authID uint64,
	req AuditPickupAuthReq,
	ip, ua string,
) (map[string]interface{}, error) {
	if kindergartenID == 0 {
		return nil, errors.New("NO_KINDERGARTEN")
	}
	pa, err := s.pickup.FindAuthorizationForKindergarten(ctx, kindergartenID, authID)
	if err != nil {
		return nil, err
	}
	if pa == nil {
		return nil, errPickupAuthNotFound
	}
	if pa.ReviewStatus != pickupAuthReviewPending {
		return nil, errPickupAuthNotPending
	}
	now := time.Now().UTC()
	var reviewStatus int8
	var respStatus int
	var label string
	switch strings.ToLower(strings.TrimSpace(req.Action)) {
	case "approve":
		reviewStatus = pickupAuthReviewApproved
		respStatus = 1
		label = "已审核"
	case "reject":
		reviewStatus = pickupAuthReviewRejected
		respStatus = 2
		label = "已驳回"
	default:
		return nil, errBadPickupAction
	}
	if err := s.pickup.UpdateAuthorizationReview(ctx, authID, reviewStatus, now, strings.TrimSpace(req.Remark)); err != nil {
		return nil, err
	}
	_ = s.oplog.Create(ctx, &domain.OperationLog{
		KindergartenID: kindergartenID,
		UserID:         adminUserID,
		Module:         "pickup",
		Action:         "audit_authorized_person",
		TargetType:     "pickup_authorization",
		TargetID:       authID,
		Detail:         fmt.Sprintf("action=%s", req.Action),
		IP:             ip,
		UserAgent:      truncateUA(ua),
	})
	return map[string]interface{}{
		"id":           authID,
		"status":       respStatus,
		"status_label": label,
		"audited_at":   now.Format(time.RFC3339),
	}, nil
}

func (s *PickupService) ListAuthorizations(
	ctx context.Context,
	kindergartenID uint64,
	pq pagination.Query,
	statusFilter string,
) ([]map[string]interface{}, int64, int, int, error) {
	if kindergartenID == 0 {
		return nil, 0, 0, 0, errors.New("NO_KINDERGARTEN")
	}
	pn := pq
	pn.Normalize()
	var reviewStatus *int8
	if statusFilter == "pending" {
		rs := pickupAuthReviewPending
		reviewStatus = &rs
	}
	rows, total, err := s.pickup.ListAuthorizationsByKindergarten(ctx, kindergartenID, reviewStatus, pn.Offset(), pn.PageSize)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	out := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		reviewLabel := "已审核"
		switch row.ReviewStatus {
		case pickupAuthReviewPending:
			reviewLabel = "待审核"
		case pickupAuthReviewRejected:
			reviewLabel = "已拒绝"
		}
		item := map[string]interface{}{
			"id":               row.ID,
			"student_id":       row.StudentID,
			"student_name":     row.StudentName,
			"class_name":       row.ClassName,
			"name":             row.Name,
			"phone":            row.Phone,
			"relation":         row.Relation,
			"id_card":          row.IDCard,
			"avatar":           row.Avatar,
			"auth_type":        row.AuthType,
			"review_status":    row.ReviewStatus,
			"review_label":     reviewLabel,
			"audited_at":       nil,
			"audit_remark":     row.AuditRemark,
			"allow_pickup":     row.AllowPickup,
			"allow_view_record": row.AllowViewRecord,
			"allow_growth_view": row.AllowGrowthView,
			"created_at":       row.CreatedAt.UTC().Format(time.RFC3339),
		}
		if row.AuditedAt != nil {
			item["audited_at"] = row.AuditedAt.UTC().Format(time.RFC3339)
		}
		out = append(out, item)
	}
	return out, total, pn.Page, pn.PageSize, nil
}
