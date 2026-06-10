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

type NoticeService struct {
	notices *repository.NoticeRepo
	oplog   *repository.OperationLogRepo
}

func NewNoticeService(notices *repository.NoticeRepo, oplog *repository.OperationLogRepo) *NoticeService {
	return &NoticeService{notices: notices, oplog: oplog}
}

var errNoticeNotFound = errors.New("NOTICE_NOT_FOUND")

func IsNoticeNotFound(err error) bool {
	return errors.Is(err, errNoticeNotFound)
}

func noticeCategoryLabel(c domain.NoticeCategory) string {
	switch c {
	case domain.NoticeCategoryHomework:
		return "作业"
	case domain.NoticeCategoryBring:
		return "带物"
	case domain.NoticeCategoryActivity:
		return "活动"
	case domain.NoticeCategoryHealth:
		return "健康"
	case domain.NoticeCategoryOther:
		return "其他"
	default:
		return string(c)
	}
}

func noticeStatusLabel(st domain.NoticeStatus) string {
	switch st {
	case domain.NoticeStatusDraft:
		return "草稿"
	case domain.NoticeStatusScheduled:
		return "定时"
	case domain.NoticeStatusPublished:
		return "已发布"
	case domain.NoticeStatusRevoked:
		return "已撤回"
	default:
		return fmt.Sprintf("状态%d", st)
	}
}

func parseNoticeCategory(s string) (domain.NoticeCategory, error) {
	c := domain.NoticeCategory(strings.TrimSpace(s))
	switch c {
	case domain.NoticeCategoryHomework, domain.NoticeCategoryBring, domain.NoticeCategoryActivity,
		domain.NoticeCategoryHealth, domain.NoticeCategoryOther:
		return c, nil
	default:
		return "", errors.New("BAD_CATEGORY")
	}
}

func (s *NoticeService) ListNotices(
	ctx context.Context,
	kindergartenID uint64,
	pq pagination.Query,
	classID *uint64,
	category string,
	status *int8,
) ([]map[string]interface{}, int64, int, int, error) {
	if kindergartenID == 0 {
		return nil, 0, 0, 0, errors.New("NO_KINDERGARTEN")
	}
	pn := pq
	pn.Normalize()
	offset := pn.Offset()
	rows, total, err := s.notices.ListNotices(ctx, kindergartenID, classID, strings.TrimSpace(category), status, pn.Keyword, offset, pn.PageSize)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	out := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		cat := domain.NoticeCategory(row.Category)
		st := domain.NoticeStatus(row.Status)
		item := map[string]interface{}{
			"id":             row.ID,
			"category":       row.Category,
			"category_label": noticeCategoryLabel(cat),
			"title":          row.Title,
			"class_name":     row.ClassName,
			"class_id":       row.ClassID,
			"creator": map[string]interface{}{
				"id":       row.CreatorID,
				"nickname": row.CreatorNick,
			},
			"status":        int(row.Status),
			"status_label":  noticeStatusLabel(st),
			"read_rate":     formatPercent(row.ReadUsers, row.ParentTotal),
			"created_at":    row.CreatedAt.UTC().Format(time.RFC3339),
		}
		if row.PublishedAt != nil {
			item["published_at"] = row.PublishedAt.UTC().Format(time.RFC3339)
		}
		out = append(out, item)
	}
	return out, total, pn.Page, pn.PageSize, nil
}

func (s *NoticeService) GetNoticeDetail(ctx context.Context, kindergartenID, noticeID uint64) (map[string]interface{}, error) {
	if kindergartenID == 0 {
		return nil, errors.New("NO_KINDERGARTEN")
	}
	n, err := s.notices.FindNoticeByID(ctx, kindergartenID, noticeID)
	if err != nil {
		return nil, err
	}
	if n == nil {
		return nil, errNoticeNotFound
	}
	atts, err := s.notices.ListAttachmentsByNoticeID(ctx, noticeID)
	if err != nil {
		return nil, err
	}
	attachments := make([]map[string]interface{}, 0, len(atts))
	for _, a := range atts {
		attachments = append(attachments, map[string]interface{}{
			"id":        a.ID,
			"file_url":  a.FileURL,
			"file_type": a.FileType,
			"file_name": a.FileName,
		})
	}
	readUsers, confirmed, parentTotal, err := s.notices.NoticeReadStats(ctx, kindergartenID, n.ClassID, noticeID)
	if err != nil {
		return nil, err
	}
	unread := parentTotal - readUsers
	if unread < 0 {
		unread = 0
	}
	readRate := formatPercent(readUsers, parentTotal)
	ca, _ := s.notices.FindLatestAudit(ctx, kindergartenID, noticeID)
	var auditRaw *int8
	if ca != nil {
		st := int8(ca.Status)
		auditRaw = &st
	}
	className := ""
	if n.Class != nil {
		className = n.Class.Name
	}
	creatorNick := ""
	creatorAvatar := ""
	if n.Creator != nil {
		creatorNick = n.Creator.Nickname
		creatorAvatar = n.Creator.Avatar
	}
	out := map[string]interface{}{
		"id":             n.ID,
		"category":       string(n.Category),
		"category_label": noticeCategoryLabel(n.Category),
		"title":          n.Title,
		"content":        n.Content,
		"class_id":       n.ClassID,
		"class_name":     className,
		"need_confirm":   n.NeedConfirm,
		"status":         int(n.Status),
		"attachments":    attachments,
		"read_stats": map[string]interface{}{
			"read_count":      readUsers,
			"unread_count":    unread,
			"confirmed_count": confirmed,
			"total_count":     parentTotal,
			"read_rate":       readRate,
		},
		"creator": map[string]interface{}{
			"id":       n.CreatorID,
			"nickname": creatorNick,
			"avatar":   creatorAvatar,
		},
		"audit_status": auditStatusString(auditRaw),
		"created_at":   n.CreatedAt.UTC().Format(time.RFC3339),
	}
	if n.PublishedAt != nil {
		out["published_at"] = n.PublishedAt.UTC().Format(time.RFC3339)
	}
	if ca != nil && ca.ReviewerID != nil && *ca.ReviewerID > 0 {
		nick, av, ok := s.notices.FindUserDisplay(ctx, *ca.ReviewerID)
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

func (s *NoticeService) DeleteNotice(
	ctx context.Context,
	kindergartenID, adminUserID, noticeID uint64,
	ip, ua string,
) error {
	if kindergartenID == 0 {
		return errors.New("NO_KINDERGARTEN")
	}
	n, err := s.notices.FindNoticeByID(ctx, kindergartenID, noticeID)
	if err != nil {
		return err
	}
	if n == nil {
		return errNoticeNotFound
	}
	affected, err := s.notices.SoftDeleteNotice(ctx, kindergartenID, noticeID)
	if err != nil {
		return err
	}
	if affected == 0 {
		return errNoticeNotFound
	}
	_ = s.oplog.Create(ctx, &domain.OperationLog{
		KindergartenID: kindergartenID,
		UserID:         adminUserID,
		Module:         "notices",
		Action:         "delete",
		TargetType:     "notice",
		TargetID:       noticeID,
		Detail:         fmt.Sprintf("class_id=%d", n.ClassID),
		IP:             ip,
		UserAgent:      truncateUA(ua),
	})
	return nil
}

// AuditNoticeReq PUT /admin/notices/:id/audit
type AuditNoticeReq struct {
	Action string `json:"action" binding:"required"`
	Remark string `json:"remark"`
}

func (s *NoticeService) AuditNotice(
	ctx context.Context,
	kindergartenID, adminUserID, noticeID uint64,
	req AuditNoticeReq,
	ip, ua string,
) (map[string]interface{}, error) {
	if kindergartenID == 0 {
		return nil, errors.New("NO_KINDERGARTEN")
	}
	n, err := s.notices.FindNoticeByID(ctx, kindergartenID, noticeID)
	if err != nil {
		return nil, err
	}
	if n == nil {
		return nil, errNoticeNotFound
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
		TargetType:     domain.AuditTargetNotice,
		TargetID:       noticeID,
		SubmitterID:    n.CreatorID,
		ReviewerID:     &reviewerID,
		Status:         st,
		ReviewedAt:     &now,
		RejectReason:   strings.TrimSpace(req.Remark),
	}
	if err := s.notices.SaveContentAudit(ctx, ca); err != nil {
		return nil, err
	}
	raw := int8(st)
	_ = s.oplog.Create(ctx, &domain.OperationLog{
		KindergartenID: kindergartenID,
		UserID:         adminUserID,
		Module:         "notices",
		Action:         "audit",
		TargetType:     "notice",
		TargetID:       noticeID,
		Detail:         fmt.Sprintf("action=%s", req.Action),
		IP:             ip,
		UserAgent:      truncateUA(ua),
	})
	return map[string]interface{}{
		"id":           noticeID,
		"audit_status": auditStatusString(&raw),
		"audited_at":   now.Format(time.RFC3339),
	}, nil
}

// NoticeAttachmentInput 创建/更新通知时的附件项。
type NoticeAttachmentInput struct {
	FileURL  string `json:"file_url"`
	FileType string `json:"file_type"`
	FileName string `json:"file_name"`
}

// CreateNoticeReq POST /admin/notices
type CreateNoticeReq struct {
	ClassID      uint64                  `json:"class_id" binding:"required"`
	Category     string                  `json:"category" binding:"required"`
	Title        string                  `json:"title" binding:"required"`
	Content      string                  `json:"content"`
	NeedConfirm  bool                    `json:"need_confirm"`
	ScheduledAt  *string                 `json:"scheduled_at"`
	Attachments  []NoticeAttachmentInput `json:"attachments"`
}

func (s *NoticeService) CreateNotice(
	ctx context.Context,
	kindergartenID, adminUserID uint64,
	req CreateNoticeReq,
	ip, ua string,
) (map[string]interface{}, error) {
	if kindergartenID == 0 {
		return nil, errors.New("NO_KINDERGARTEN")
	}
	ok, err := s.notices.IsClassInKindergarten(ctx, kindergartenID, req.ClassID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("CLASS_NOT_FOUND")
	}
	cat, err := parseNoticeCategory(req.Category)
	if err != nil {
		return nil, err
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, errors.New("BAD_TITLE")
	}
	var status domain.NoticeStatus
	var publishedAt *time.Time
	var scheduledAt *time.Time
	now := time.Now().UTC()
	if req.ScheduledAt != nil && strings.TrimSpace(*req.ScheduledAt) != "" {
		t, e := time.Parse(time.RFC3339, strings.TrimSpace(*req.ScheduledAt))
		if e != nil {
			return nil, errors.New("BAD_SCHEDULED_AT")
		}
		t = t.UTC()
		scheduledAt = &t
		if t.After(now) {
			status = domain.NoticeStatusScheduled
		} else {
			status = domain.NoticeStatusPublished
			publishedAt = &t
		}
	} else {
		status = domain.NoticeStatusPublished
		publishedAt = &now
	}
	n := &domain.Notice{
		KindergartenID: kindergartenID,
		ClassID:          req.ClassID,
		CreatorID:        adminUserID,
		Category:         cat,
		Title:            title,
		Content:          req.Content,
		NeedConfirm:      req.NeedConfirm,
		ScheduledAt:      scheduledAt,
		PublishedAt:      publishedAt,
		Status:           status,
	}
	if err := s.notices.CreateNotice(ctx, n); err != nil {
		return nil, err
	}
	var attRows []domain.NoticeAttachment
	for i, a := range req.Attachments {
		url := strings.TrimSpace(a.FileURL)
		if url == "" {
			continue
		}
		ft := strings.TrimSpace(a.FileType)
		if ft == "" {
			ft = "image"
		}
		attRows = append(attRows, domain.NoticeAttachment{
			NoticeID: n.ID,
			FileURL:  url,
			FileType: ft,
			FileName: strings.TrimSpace(a.FileName),
			SortNum:  i,
		})
	}
	if len(attRows) > 0 {
		if err := s.notices.CreateAttachments(ctx, attRows); err != nil {
			return nil, err
		}
	}
	_ = s.oplog.Create(ctx, &domain.OperationLog{
		KindergartenID: kindergartenID,
		UserID:         adminUserID,
		Module:         "notices",
		Action:         "create",
		TargetType:     "notice",
		TargetID:       n.ID,
		Detail:         fmt.Sprintf("class_id=%d", req.ClassID),
		IP:             ip,
		UserAgent:      truncateUA(ua),
	})
	return map[string]interface{}{
		"id":         n.ID,
		"status":     int(n.Status),
		"created_at": n.CreatedAt.UTC().Format(time.RFC3339),
	}, nil
}

// UpdateNoticeReq PUT /admin/notices/:id
type UpdateNoticeReq struct {
	ClassID      *uint64                  `json:"class_id"`
	Category     *string                  `json:"category"`
	Title        *string                  `json:"title"`
	Content      *string                  `json:"content"`
	NeedConfirm  *bool                    `json:"need_confirm"`
	ScheduledAt  *string                  `json:"scheduled_at"`
	Status       *int8                    `json:"status"`
	Attachments  *[]NoticeAttachmentInput `json:"attachments"`
}

func (s *NoticeService) UpdateNotice(
	ctx context.Context,
	kindergartenID, adminUserID, noticeID uint64,
	req UpdateNoticeReq,
	ip, ua string,
) error {
	if kindergartenID == 0 {
		return errors.New("NO_KINDERGARTEN")
	}
	n, err := s.notices.FindNoticeByID(ctx, kindergartenID, noticeID)
	if err != nil {
		return err
	}
	if n == nil {
		return errNoticeNotFound
	}
	patch := map[string]interface{}{}
	if req.ClassID != nil {
		if *req.ClassID == 0 {
			return errors.New("BAD_CLASS_ID")
		}
		ok, err := s.notices.IsClassInKindergarten(ctx, kindergartenID, *req.ClassID)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("CLASS_NOT_FOUND")
		}
		patch["class_id"] = *req.ClassID
	}
	if req.Category != nil {
		cat, err := parseNoticeCategory(*req.Category)
		if err != nil {
			return err
		}
		patch["category"] = cat
	}
	if req.Title != nil {
		t := strings.TrimSpace(*req.Title)
		if t == "" {
			return errors.New("BAD_TITLE")
		}
		patch["title"] = t
	}
	if req.Content != nil {
		patch["content"] = *req.Content
	}
	if req.NeedConfirm != nil {
		patch["need_confirm"] = *req.NeedConfirm
	}
	if req.ScheduledAt != nil {
		sa := strings.TrimSpace(*req.ScheduledAt)
		if sa == "" {
			patch["scheduled_at"] = nil
		} else {
			t, e := time.Parse(time.RFC3339, sa)
			if e != nil {
				return errors.New("BAD_SCHEDULED_AT")
			}
			tu := t.UTC()
			patch["scheduled_at"] = &tu
		}
	}
	if req.Status != nil {
		st := domain.NoticeStatus(*req.Status)
		switch st {
		case domain.NoticeStatusDraft, domain.NoticeStatusScheduled, domain.NoticeStatusPublished, domain.NoticeStatusRevoked:
			patch["status"] = st
			if st == domain.NoticeStatusPublished {
				now := time.Now().UTC()
				patch["published_at"] = &now
			}
		default:
			return errors.New("BAD_STATUS")
		}
	}
	if len(patch) == 0 && req.Attachments == nil {
		return errors.New("EMPTY_UPDATE")
	}
	if len(patch) > 0 {
		if _, err := s.notices.UpdateNoticeFields(ctx, kindergartenID, noticeID, patch); err != nil {
			return err
		}
	}
	if req.Attachments != nil {
		if err := s.notices.DeleteAttachmentsByNoticeID(ctx, noticeID); err != nil {
			return err
		}
		var attRows []domain.NoticeAttachment
		for i, a := range *req.Attachments {
			url := strings.TrimSpace(a.FileURL)
			if url == "" {
				continue
			}
			ft := strings.TrimSpace(a.FileType)
			if ft == "" {
				ft = "image"
			}
			attRows = append(attRows, domain.NoticeAttachment{
				NoticeID: noticeID,
				FileURL:  url,
				FileType: ft,
				FileName: strings.TrimSpace(a.FileName),
				SortNum:  i,
			})
		}
		if len(attRows) > 0 {
			if err := s.notices.CreateAttachments(ctx, attRows); err != nil {
				return err
			}
		}
	}
	_ = s.oplog.Create(ctx, &domain.OperationLog{
		KindergartenID: kindergartenID,
		UserID:         adminUserID,
		Module:         "notices",
		Action:         "update",
		TargetType:     "notice",
		TargetID:       noticeID,
		Detail:         "",
		IP:             ip,
		UserAgent:      truncateUA(ua),
	})
	return nil
}
