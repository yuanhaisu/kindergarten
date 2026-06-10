package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/pkg/pagination"
)

type GrowthService struct {
	growth *repository.GrowthRepo
	oplog  *repository.OperationLogRepo
}

func NewGrowthService(growth *repository.GrowthRepo, oplog *repository.OperationLogRepo) *GrowthService {
	return &GrowthService{growth: growth, oplog: oplog}
}

var errGrowthNotFound = errors.New("GROWTH_LOG_NOT_FOUND")

func IsGrowthNotFound(err error) bool {
	return errors.Is(err, errGrowthNotFound)
}

func auditStatusString(raw *int8) string {
	if raw == nil {
		return "pending"
	}
	switch domain.AuditStatus(*raw) {
	case domain.AuditStatusApproved:
		return "passed"
	case domain.AuditStatusRejected:
		return "rejected"
	default:
		return "pending"
	}
}

func growthContentPreview(s string, maxRunes int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	return string(r[:maxRunes]) + "..."
}

func (s *GrowthService) ListGrowthLogs(
	ctx context.Context,
	kindergartenID uint64,
	pq pagination.Query,
	classID *uint64,
	teacherID *uint64,
	dateStr string,
) ([]map[string]interface{}, int64, int, int, error) {
	if kindergartenID == 0 {
		return nil, 0, 0, 0, errors.New("NO_KINDERGARTEN")
	}
	pn := pq
	pn.Normalize()
	offset := pn.Offset()
	rows, total, err := s.growth.ListLogs(ctx, kindergartenID, classID, teacherID, strings.TrimSpace(dateStr), offset, pn.PageSize)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	out := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		out = append(out, map[string]interface{}{
			"id":              row.ID,
			"date":            row.LogDate.In(shanghaiLoc()).Format("2006-01-02"),
			"class_name":      row.ClassName,
			"content_preview": growthContentPreview(row.Content, 48),
			"media_count":     row.MediaCount,
			"likes_count":     row.LikesCount,
			"comments_count":  row.CommentsCount,
			"creator": map[string]interface{}{
				"id":       row.CreatorID,
				"nickname": row.CreatorNick,
			},
			"audit_status": auditStatusString(row.AuditStatusRaw),
			"created_at":   row.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return out, total, pn.Page, pn.PageSize, nil
}

func (s *GrowthService) GetGrowthLogDetail(ctx context.Context, kindergartenID, logID uint64) (map[string]interface{}, error) {
	if kindergartenID == 0 {
		return nil, errors.New("NO_KINDERGARTEN")
	}
	log, err := s.growth.FindLogByID(ctx, kindergartenID, logID)
	if err != nil {
		return nil, err
	}
	if log == nil {
		return nil, errGrowthNotFound
	}
	mediaRows, err := s.growth.ListMediaByLogID(ctx, logID)
	if err != nil {
		return nil, err
	}
	media := make([]map[string]interface{}, 0, len(mediaRows))
	for _, m := range mediaRows {
		media = append(media, map[string]interface{}{
			"id":        m.ID,
			"file_url":  m.FileURL,
			"thumb_url": m.ThumbURL,
			"file_type": m.FileType,
		})
	}
	ca, _ := s.growth.FindLatestAudit(ctx, kindergartenID, logID)
	var auditRaw *int8
	if ca != nil {
		st := int8(ca.Status)
		auditRaw = &st
	}
	creatorNick := ""
	creatorAvatar := ""
	if log.Creator != nil {
		creatorNick = log.Creator.Nickname
		creatorAvatar = log.Creator.Avatar
	}
	className := ""
	if log.Class != nil {
		className = log.Class.Name
	}
	likes, err := s.growth.CountLikesByLogID(ctx, logID)
	if err != nil {
		return nil, err
	}
	comments, err := s.growth.CountCommentsByLogID(ctx, logID)
	if err != nil {
		return nil, err
	}

	out := map[string]interface{}{
		"id":             log.ID,
		"date":           log.Date.In(shanghaiLoc()).Format("2006-01-02"),
		"class_id":       log.ClassID,
		"class_name":     className,
		"content":        log.Content,
		"media":          media,
		"media_count":    int64(len(media)),
		"likes_count":    likes,
		"comments_count": comments,
		"creator": map[string]interface{}{
			"id":       log.CreatorID,
			"nickname": creatorNick,
			"avatar":   creatorAvatar,
		},
		"audit_status": auditStatusString(auditRaw),
		"created_at":   log.CreatedAt.UTC().Format(time.RFC3339),
	}
	if ca != nil && ca.ReviewerID != nil && *ca.ReviewerID > 0 {
		nick, av, ok := s.growth.FindUserDisplay(ctx, *ca.ReviewerID)
		if ok {
			out["auditor"] = map[string]interface{}{
				"id":       *ca.ReviewerID,
				"nickname": nick,
				"avatar":   av,
			}
		}
	}
	if ca != nil && ca.ReviewedAt != nil {
		out["audited_at"] = ca.ReviewedAt.UTC().Format(time.RFC3339)
	}
	return out, nil
}

func (s *GrowthService) DeleteGrowthLog(
	ctx context.Context,
	kindergartenID, adminUserID, logID uint64,
	ip, ua string,
) error {
	if kindergartenID == 0 {
		return errors.New("NO_KINDERGARTEN")
	}
	log, err := s.growth.FindLogByID(ctx, kindergartenID, logID)
	if err != nil {
		return err
	}
	if log == nil {
		return errGrowthNotFound
	}
	n, err := s.growth.SoftDeleteLog(ctx, kindergartenID, logID)
	if err != nil {
		return err
	}
	if n == 0 {
		return errGrowthNotFound
	}
	_ = s.oplog.Create(ctx, &domain.OperationLog{
		KindergartenID: kindergartenID,
		UserID:         adminUserID,
		Module:         "growth",
		Action:         "delete",
		TargetType:     "growth_daily_log",
		TargetID:       logID,
		Detail:         fmt.Sprintf("class_id=%d", log.ClassID),
		IP:             ip,
		UserAgent:      truncateUA(ua),
	})
	return nil
}

// AuditGrowthLogReq PUT /admin/growth/logs/:id/audit
type AuditGrowthLogReq struct {
	Action string `json:"action" binding:"required"` // pass / reject
	Remark string `json:"remark"`
}

func (s *GrowthService) AuditGrowthLog(
	ctx context.Context,
	kindergartenID, adminUserID, logID uint64,
	req AuditGrowthLogReq,
	ip, ua string,
) (map[string]interface{}, error) {
	if kindergartenID == 0 {
		return nil, errors.New("NO_KINDERGARTEN")
	}
	log, err := s.growth.FindLogByID(ctx, kindergartenID, logID)
	if err != nil {
		return nil, err
	}
	if log == nil {
		return nil, errGrowthNotFound
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
	reviewerID := adminUserID
	ca := &domain.ContentAudit{
		KindergartenID: kindergartenID,
		TargetType:     domain.AuditTargetGrowth,
		TargetID:       logID,
		SubmitterID:    log.CreatorID,
		ReviewerID:     &reviewerID,
		Status:         st,
		ReviewedAt:     &now,
		RejectReason:   strings.TrimSpace(req.Remark),
	}
	if err := s.growth.SaveContentAudit(ctx, ca); err != nil {
		return nil, err
	}
	raw := int8(st)
	_ = s.oplog.Create(ctx, &domain.OperationLog{
		KindergartenID: kindergartenID,
		UserID:         adminUserID,
		Module:         "growth",
		Action:         "audit",
		TargetType:     "growth_daily_log",
		TargetID:       logID,
		Detail:         fmt.Sprintf("action=%s", req.Action),
		IP:             ip,
		UserAgent:      truncateUA(ua),
	})
	return map[string]interface{}{
		"id":           logID,
		"audit_status": auditStatusString(&raw),
		"audited_at":   now.Format(time.RFC3339),
	}, nil
}

// GetGrowthStats 管理端统计（与 dashboard 分接口，字段按 09-growth.md）。
func (s *GrowthService) GetGrowthStats(ctx context.Context, kindergartenID uint64, startDateStr, endDateStr string) (map[string]interface{}, error) {
	if kindergartenID == 0 {
		return nil, errors.New("NO_KINDERGARTEN")
	}
	loc := shanghaiLoc()
	now := time.Now().In(loc)
	sTrim := strings.TrimSpace(startDateStr)
	eTrim := strings.TrimSpace(endDateStr)
	if (sTrim == "") != (eTrim == "") {
		return nil, errors.New("BAD_DATE")
	}
	var startDay, endDay time.Time
	var err error
	if sTrim == "" && eTrim == "" {
		y, m, _ := now.Date()
		startDay = time.Date(y, m, 1, 0, 0, 0, 0, loc)
		endDay = startDay.AddDate(0, 1, 0).Add(-time.Nanosecond)
	} else {
		startDay, err = time.ParseInLocation("2006-01-02", sTrim, loc)
		if err != nil {
			return nil, errors.New("BAD_DATE")
		}
		endDay, err = time.ParseInLocation("2006-01-02", eTrim, loc)
		if err != nil {
			return nil, errors.New("BAD_DATE")
		}
		if endDay.Before(startDay) {
			return nil, errors.New("BAD_DATE_RANGE")
		}
		endDay = time.Date(endDay.Year(), endDay.Month(), endDay.Day(), 23, 59, 59, 999999999, loc)
	}
	// DB `date` 列为 DATE，用当地日历日的 0 点作为上下界传入 GORM
	startDB := time.Date(startDay.Year(), startDay.Month(), startDay.Day(), 0, 0, 0, 0, loc)
	endDB := time.Date(endDay.Year(), endDay.Month(), endDay.Day(), 0, 0, 0, 0, loc)

	totalLogs, err := s.growth.CountLogsInDateRange(ctx, kindergartenID, startDB, endDB)
	if err != nil {
		return nil, err
	}
	totalMedia, err := s.growth.CountMediaInLogDateRange(ctx, kindergartenID, startDB, endDB)
	if err != nil {
		return nil, err
	}
	totalLikes, err := s.growth.CountLikesInLogDateRange(ctx, kindergartenID, startDB, endDB)
	if err != nil {
		return nil, err
	}
	totalComments, err := s.growth.CountCommentsInLogDateRange(ctx, kindergartenID, startDB, endDB)
	if err != nil {
		return nil, err
	}
	byClass, err := s.growth.StatsByClass(ctx, kindergartenID, startDB, endDB)
	if err != nil {
		return nil, err
	}
	trend, err := s.growth.StatsDailyTrend(ctx, kindergartenID, startDB, endDB)
	if err != nil {
		return nil, err
	}
	byClassOut := make([]map[string]interface{}, 0, len(byClass))
	for _, row := range byClass {
		byClassOut = append(byClassOut, map[string]interface{}{
			"class_name": row.ClassName,
			"teacher":    row.Teacher,
			"logs":       row.Logs,
			"media":      row.Media,
		})
	}
	dailyOut := make([]map[string]interface{}, 0, len(trend))
	for _, row := range trend {
		dailyOut = append(dailyOut, map[string]interface{}{
			"date":  row.Day.In(loc).Format("2006-01-02"),
			"logs":  row.Logs,
			"media": row.Media,
		})
	}
	return map[string]interface{}{
		"period": map[string]interface{}{
			"start": startDB.In(loc).Format("2006-01-02"),
			"end":   endDB.In(loc).Format("2006-01-02"),
		},
		"total_logs":     totalLogs,
		"total_media":    totalMedia,
		"total_likes":    totalLikes,
		"total_comments": totalComments,
		"by_class":       byClassOut,
		"daily_trend":    dailyOut,
	}, nil
}

func growthExportDateRange(loc *time.Location, startDay, endDay time.Time) (from, toExclusive time.Time) {
	sy, sm, sd := startDay.In(loc).Date()
	ey, em, ed := endDay.In(loc).Date()
	from = time.Date(sy, sm, sd, 0, 0, 0, 0, loc)
	endOnly := time.Date(ey, em, ed, 0, 0, 0, 0, loc)
	toExclusive = endOnly.AddDate(0, 0, 1)
	return from, toExclusive
}

// ExportGrowthLogsXLSX 成长记录导出 xlsx（异步任务与同步导出可复用）。
func (s *GrowthService) ExportGrowthLogsXLSX(
	ctx context.Context,
	kindergartenID uint64,
	classID *uint64,
	startStr, endStr, format string,
) ([]byte, string, error) {
	if kindergartenID == 0 {
		return nil, "", fmt.Errorf("NO_KINDERGARTEN")
	}
	if format != "" && format != "xlsx" {
		return nil, "", fmt.Errorf("BAD_FORMAT")
	}
	loc := shanghaiLoc()
	startDay, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(startStr), loc)
	if err != nil {
		return nil, "", fmt.Errorf("BAD_DATE")
	}
	endDay, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(endStr), loc)
	if err != nil {
		return nil, "", fmt.Errorf("BAD_DATE")
	}
	if endDay.Before(startDay) {
		return nil, "", fmt.Errorf("BAD_DATE_RANGE")
	}
	from, toExc := growthExportDateRange(loc, startDay, endDay)

	rows, err := s.growth.ListLogsForExportDateRange(ctx, kindergartenID, classID, from, toExc)
	if err != nil {
		return nil, "", err
	}
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	headers := []string{"日期", "班级", "记录内容", "老师", "创建时间"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}
	for i, row := range rows {
		r := i + 2
		d := row.LogDate.In(loc).Format("2006-01-02")
		content := row.Content
		rs := []rune(content)
		if len(rs) > 5000 {
			content = string(rs[:5000])
		}
		ca := row.CreatedAt.In(loc).Format("2006-01-02 15:04")
		vals := []interface{}{d, row.ClassName, content, row.Creator, ca}
		for col, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(col+1, r)
			_ = f.SetCellValue(sheet, cell, v)
		}
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, "", err
	}
	fn := fmt.Sprintf("growth_export_%s_%s.xlsx", startDay.Format("20060102"), endDay.Format("20060102"))
	return buf.Bytes(), fn, nil
}
