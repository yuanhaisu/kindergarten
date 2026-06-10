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

type AuditService struct {
	audits *repository.AuditRepo
	app    *repository.KindergartenAppSettingRepo
	oplog  *repository.OperationLogRepo
}

func NewAuditService(
	audits *repository.AuditRepo,
	app *repository.KindergartenAppSettingRepo,
	oplog *repository.OperationLogRepo,
) *AuditService {
	return &AuditService{audits: audits, app: app, oplog: oplog}
}

var errAuditNotFound = errors.New("AUDIT_NOT_FOUND")

func IsAuditNotFound(err error) bool {
	return errors.Is(err, errAuditNotFound)
}

type auditExtraJSON struct {
	Notice *struct {
		NeedAudit bool `json:"need_audit"`
	} `json:"notice"`
	Growth *struct {
		NeedAudit bool `json:"need_audit"`
	} `json:"growth"`
}

func (s *AuditService) loadNeedAuditFlags(ctx context.Context, kindergartenID uint64) (noticeNeed, growthNeed bool, err error) {
	noticeNeed, growthNeed = true, false
	row, err := s.app.FindByKindergartenID(ctx, kindergartenID)
	if err != nil {
		return false, false, err
	}
	if row == nil || strings.TrimSpace(row.ExtraConfig) == "" {
		return noticeNeed, growthNeed, nil
	}
	var p auditExtraJSON
	if e := json.Unmarshal([]byte(row.ExtraConfig), &p); e != nil {
		return noticeNeed, growthNeed, nil
	}
	if p.Notice != nil {
		noticeNeed = p.Notice.NeedAudit
	}
	if p.Growth != nil {
		growthNeed = p.Growth.NeedAudit
	}
	return noticeNeed, growthNeed, nil
}

func (s *AuditService) syncPendingIfNeeded(ctx context.Context, kindergartenID uint64, typeFilter string) error {
	noticeNeed, growthNeed, err := s.loadNeedAuditFlags(ctx, kindergartenID)
	if err != nil {
		return err
	}
	tf := strings.TrimSpace(strings.ToLower(typeFilter))
	doNotice := noticeNeed && (tf == "" || tf == "notice")
	doGrowth := growthNeed && (tf == "" || tf == "growth")
	if !doNotice && !doGrowth {
		return nil
	}
	return s.audits.EnsurePendingAudits(ctx, kindergartenID, doNotice, doGrowth)
}

func auditTypeLabel(t string) string {
	switch domain.AuditTargetType(strings.TrimSpace(t)) {
	case domain.AuditTargetNotice:
		return "通知"
	case domain.AuditTargetGrowth:
		return "成长"
	case domain.AuditTargetComment:
		return "评论"
	default:
		return t
	}
}

func auditStatusLabelFromCode(st int8) (code, label string) {
	switch domain.AuditStatus(st) {
	case domain.AuditStatusApproved:
		return "passed", "已通过"
	case domain.AuditStatusRejected:
		return "rejected", "已驳回"
	default:
		return "pending", "待审核"
	}
}

func (s *AuditService) pickListTitle(row *repository.AuditListScan) string {
	switch domain.AuditTargetType(row.TargetType) {
	case domain.AuditTargetNotice:
		if strings.TrimSpace(row.NoticeTitle) != "" {
			return row.NoticeTitle
		}
	case domain.AuditTargetGrowth:
		if row.GrowthDate != nil {
			return "成长记录 " + row.GrowthDate.In(shanghaiLoc()).Format("2006-01-02")
		}
	case domain.AuditTargetComment:
		return "评论"
	}
	return "（内容不可用）"
}

func (s *AuditService) pickContentBody(row *repository.AuditListScan) string {
	switch domain.AuditTargetType(row.TargetType) {
	case domain.AuditTargetNotice:
		return row.NoticeContent
	case domain.AuditTargetGrowth:
		return row.GrowthContent
	default:
		return ""
	}
}

func (s *AuditService) ListAudits(
	ctx context.Context,
	kindergartenID uint64,
	pq pagination.Query,
	typeFilter, statusFilter string,
) ([]map[string]interface{}, int64, int, int, error) {
	if kindergartenID == 0 {
		return nil, 0, 0, 0, errors.New("NO_KINDERGARTEN")
	}
	pn := pq
	pn.Normalize()
	if err := s.syncPendingIfNeeded(ctx, kindergartenID, typeFilter); err != nil {
		return nil, 0, 0, 0, err
	}
	rows, total, err := s.audits.ListAudits(ctx, kindergartenID, typeFilter, statusFilter, pn.Offset(), pn.PageSize)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	out := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		stStr, stLabel := auditStatusLabelFromCode(row.Status)
		item := map[string]interface{}{
			"id":              row.CAID,
			"type":            row.TargetType,
			"type_label":      auditTypeLabel(row.TargetType),
			"reference_id":    row.TargetID,
			"title":           s.pickListTitle(&row),
			"content_preview": growthContentPreview(s.pickContentBody(&row), 80),
			"submitter": map[string]interface{}{
				"id":         row.SubmitterID,
				"nickname":   row.SubmitterNick,
				"class_name": row.ClassName,
			},
			"status":        stStr,
			"status_label":  stLabel,
			"submitted_at":  row.CreatedAt.UTC().Format(time.RFC3339),
		}
		out = append(out, item)
	}
	return out, total, pn.Page, pn.PageSize, nil
}

type AuditContentReq struct {
	Action string `json:"action" binding:"required"`
	Remark string `json:"remark"`
}

func (s *AuditService) AuditContent(
	ctx context.Context,
	kindergartenID, adminUserID, auditID uint64,
	req AuditContentReq,
	ip, ua string,
) (map[string]interface{}, error) {
	if kindergartenID == 0 {
		return nil, errors.New("NO_KINDERGARTEN")
	}
	ca, err := s.audits.FindByID(ctx, kindergartenID, auditID)
	if err != nil {
		return nil, err
	}
	if ca == nil {
		return nil, errAuditNotFound
	}
	var st domain.AuditStatus
	switch strings.TrimSpace(strings.ToLower(req.Action)) {
	case "pass":
		st = domain.AuditStatusApproved
	case "reject":
		st = domain.AuditStatusRejected
	default:
		return nil, errors.New("BAD_AUDIT_ACTION")
	}
	now := time.Now().UTC()
	reason := strings.TrimSpace(req.Remark)
	if err := s.audits.UpdateAuditDecision(ctx, kindergartenID, auditID, st, adminUserID, now, reason); err != nil {
		return nil, err
	}
	stStr, _ := auditStatusLabelFromCode(int8(st))
	_ = s.oplog.Create(ctx, &domain.OperationLog{
		KindergartenID: kindergartenID,
		UserID:         adminUserID,
		Module:         "audit",
		Action:         "decide",
		TargetType:     string(ca.TargetType),
		TargetID:       ca.TargetID,
		Detail:         fmt.Sprintf("audit_id=%d action=%s", auditID, req.Action),
		IP:             ip,
		UserAgent:      truncateUA(ua),
	})
	return map[string]interface{}{
		"id":         auditID,
		"status":     stStr,
		"audited_at": now.Format(time.RFC3339),
	}, nil
}

type auditTypeBucket struct {
	Pending  int64
	Passed   int64
	Rejected int64
}

func (s *AuditService) GetAuditStats(ctx context.Context, kindergartenID uint64) (map[string]interface{}, error) {
	if kindergartenID == 0 {
		return nil, errors.New("NO_KINDERGARTEN")
	}
	if err := s.syncPendingIfNeeded(ctx, kindergartenID, ""); err != nil {
		return nil, err
	}
	pending, err := s.audits.CountPending(ctx, kindergartenID)
	if err != nil {
		return nil, err
	}
	loc := shanghaiLoc()
	now := time.Now().In(loc)
	startDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	endDay := startDay.Add(24 * time.Hour)
	startUTC := startDay.UTC()
	endUTC := endDay.UTC()
	passedToday, err := s.audits.CountReviewedBetween(ctx, kindergartenID, domain.AuditStatusApproved, startUTC, endUTC)
	if err != nil {
		return nil, err
	}
	rejectedToday, err := s.audits.CountReviewedBetween(ctx, kindergartenID, domain.AuditStatusRejected, startUTC, endUTC)
	if err != nil {
		return nil, err
	}
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
	monthEnd := monthStart.AddDate(0, 1, 0)
	totalMonth, err := s.audits.CountReviewedInMonth(ctx, kindergartenID, monthStart.UTC(), monthEnd.UTC())
	if err != nil {
		return nil, err
	}
	agg, err := s.audits.CountByTypeAndStatus(ctx, kindergartenID)
	if err != nil {
		return nil, err
	}
	buckets := map[string]*auditTypeBucket{
		"notice":  {},
		"growth":  {},
		"comment": {},
	}
	for _, row := range agg {
		key := row.TargetType
		b, ok := buckets[key]
		if !ok {
			continue
		}
		switch domain.AuditStatus(row.Status) {
		case domain.AuditStatusPending:
			b.Pending += row.Cnt
		case domain.AuditStatusApproved:
			b.Passed += row.Cnt
		case domain.AuditStatusRejected:
			b.Rejected += row.Cnt
		}
	}
	byType := map[string]interface{}{}
	for k, b := range buckets {
		byType[k] = map[string]interface{}{
			"pending":  b.Pending,
			"passed":   b.Passed,
			"rejected": b.Rejected,
		}
	}
	return map[string]interface{}{
		"pending":            pending,
		"passed_today":       passedToday,
		"rejected_today":     rejectedToday,
		"total_this_month":   totalMonth,
		"by_type":            byType,
	}, nil
}
