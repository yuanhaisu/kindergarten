package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/pkg/pagination"
)

type AnnouncementService struct {
	ann  *repository.AnnouncementRepo
	oplog *repository.OperationLogRepo
}

func NewAnnouncementService(ann *repository.AnnouncementRepo, oplog *repository.OperationLogRepo) *AnnouncementService {
	return &AnnouncementService{ann: ann, oplog: oplog}
}

var errAnnouncementNotFound = errors.New("ANNOUNCEMENT_NOT_FOUND")

func IsAnnouncementNotFound(err error) bool {
	return errors.Is(err, errAnnouncementNotFound)
}

func announcementStatusLabel(st domain.AnnouncementStatus) string {
	switch st {
	case domain.AnnouncementStatusDraft:
		return "草稿"
	case domain.AnnouncementStatusScheduled:
		return "定时发布"
	case domain.AnnouncementStatusPublished:
		return "已发布"
	case domain.AnnouncementStatusRevoked:
		return "已撤回"
	default:
		return fmt.Sprintf("状态%d", st)
	}
}

func announcementContentPreview(s string, maxRunes int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	r := []rune(s)
	return string(r[:maxRunes]) + "..."
}

var allowedAnnouncementTargetRoles = map[string]struct{}{
	"parent": {}, "teacher": {}, "admin": {},
}

func normalizeTargetRoles(in []string) ([]string, error) {
	if len(in) == 0 {
		return []string{"parent", "teacher"}, nil
	}
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, r := range in {
		k := strings.TrimSpace(strings.ToLower(r))
		if k == "" {
			continue
		}
		if _, ok := allowedAnnouncementTargetRoles[k]; !ok {
			return nil, errors.New("BAD_TARGET_ROLES")
		}
		if _, dup := seen[k]; dup {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, k)
	}
	if len(out) == 0 {
		return nil, errors.New("BAD_TARGET_ROLES")
	}
	return out, nil
}

func parseTargetRolesJSON(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}
	}
	var arr []string
	if err := json.Unmarshal([]byte(raw), &arr); err != nil {
		return []string{}
	}
	return arr
}

func (s *AnnouncementService) ListAnnouncements(
	ctx context.Context,
	kindergartenID uint64,
	pq pagination.Query,
	status *int8,
) ([]map[string]interface{}, int64, int, int, error) {
	if kindergartenID == 0 {
		return nil, 0, 0, 0, errors.New("NO_KINDERGARTEN")
	}
	pn := pq
	pn.Normalize()
	offset := pn.Offset()
	var st *domain.AnnouncementStatus
	if status != nil {
		v := domain.AnnouncementStatus(*status)
		st = &v
	}
	rows, total, err := s.ann.ListAnnouncements(ctx, kindergartenID, st, offset, pn.PageSize)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	out := make([]map[string]interface{}, 0, len(rows))
	for _, a := range rows {
		nick := ""
		if a.Creator != nil {
			nick = a.Creator.Nickname
		}
		item := map[string]interface{}{
			"id":              a.ID,
			"title":           a.Title,
			"content_preview": announcementContentPreview(a.Content, 48),
			"cover_url":       a.CoverURL,
			"is_important":    a.IsImportant,
			"is_top":          a.IsTop,
			"target_roles":    parseTargetRolesJSON(a.TargetRoles),
			"status":          int(a.Status),
			"status_label":    announcementStatusLabel(a.Status),
			"view_count":      a.ViewCount,
			"creator":         map[string]interface{}{"nickname": nick},
			"created_at":      a.CreatedAt.UTC().Format(time.RFC3339),
		}
		if a.PublishedAt != nil {
			item["published_at"] = a.PublishedAt.UTC().Format(time.RFC3339)
		}
		out = append(out, item)
	}
	return out, total, pn.Page, pn.PageSize, nil
}

func (s *AnnouncementService) GetAnnouncementDetail(ctx context.Context, kindergartenID, id uint64) (map[string]interface{}, error) {
	if kindergartenID == 0 {
		return nil, errors.New("NO_KINDERGARTEN")
	}
	a, err := s.ann.FindAnnouncementByID(ctx, kindergartenID, id)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, errAnnouncementNotFound
	}
	atts, err := s.ann.ListAttachmentsByAnnouncementID(ctx, id)
	if err != nil {
		return nil, err
	}
	attachments := make([]map[string]interface{}, 0, len(atts))
	for _, x := range atts {
		attachments = append(attachments, map[string]interface{}{
			"file_url":  x.FileURL,
			"file_name": x.FileName,
			"file_size": x.FileSize,
		})
	}
	nick, avatar := "", ""
	if a.Creator != nil {
		nick = a.Creator.Nickname
		avatar = a.Creator.Avatar
	}
	out := map[string]interface{}{
		"id":            a.ID,
		"title":         a.Title,
		"content":       a.Content,
		"cover_url":     a.CoverURL,
		"is_important":  a.IsImportant,
		"is_top":        a.IsTop,
		"target_roles":  parseTargetRolesJSON(a.TargetRoles),
		"status":        int(a.Status),
		"attachments":   attachments,
		"view_count":    a.ViewCount,
		"creator":       map[string]interface{}{"id": a.CreatorID, "nickname": nick, "avatar": avatar},
		"created_at":    a.CreatedAt.UTC().Format(time.RFC3339),
	}
	if a.PublishedAt != nil {
		out["published_at"] = a.PublishedAt.UTC().Format(time.RFC3339)
	}
	return out, nil
}

// CreateAnnouncementReq POST /admin/announcements
type CreateAnnouncementReq struct {
	Title         string                      `json:"title" binding:"required"`
	Content       string                      `json:"content"`
	CoverURL      string                      `json:"cover_url"`
	IsImportant   bool                        `json:"is_important"`
	IsTop         bool                        `json:"is_top"`
	TargetRoles   []string                    `json:"target_roles"`
	ScheduledAt   *string                     `json:"scheduled_at"`
	Attachments   []AnnouncementAttachmentIn  `json:"attachments"`
}

type AnnouncementAttachmentIn struct {
	FileURL  string `json:"file_url"`
	FileName string `json:"file_name"`
	FileSize int64  `json:"file_size"`
}

func (s *AnnouncementService) CreateAnnouncement(
	ctx context.Context,
	kindergartenID, adminUserID uint64,
	req CreateAnnouncementReq,
	ip, ua string,
) (map[string]interface{}, error) {
	if kindergartenID == 0 {
		return nil, errors.New("NO_KINDERGARTEN")
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, errors.New("BAD_TITLE")
	}
	roles, err := normalizeTargetRoles(req.TargetRoles)
	if err != nil {
		return nil, err
	}
	rolesJSON, err := json.Marshal(roles)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	var st domain.AnnouncementStatus
	var pubAt *time.Time
	var schedAt *time.Time
	if req.ScheduledAt != nil && strings.TrimSpace(*req.ScheduledAt) != "" {
		t, e := time.Parse(time.RFC3339, strings.TrimSpace(*req.ScheduledAt))
		if e != nil {
			return nil, errors.New("BAD_SCHEDULED_AT")
		}
		tu := t.UTC()
		schedAt = &tu
		if tu.After(now) {
			st = domain.AnnouncementStatusScheduled
		} else {
			st = domain.AnnouncementStatusPublished
			pubAt = &tu
		}
	} else {
		st = domain.AnnouncementStatusPublished
		pubAt = &now
	}
	a := &domain.Announcement{
		KindergartenID: kindergartenID,
		CreatorID:      adminUserID,
		Title:          title,
		Content:        req.Content,
		CoverURL:       strings.TrimSpace(req.CoverURL),
		IsImportant:    req.IsImportant,
		IsTop:          req.IsTop,
		TargetRoles:    string(rolesJSON),
		Status:         st,
		ScheduledAt:    schedAt,
		PublishedAt:    pubAt,
	}
	var attRows []domain.AnnouncementAttachment
	for i, x := range req.Attachments {
		u := strings.TrimSpace(x.FileURL)
		if u == "" {
			continue
		}
		attRows = append(attRows, domain.AnnouncementAttachment{
			FileURL:  u,
			FileName: strings.TrimSpace(x.FileName),
			FileSize: x.FileSize,
			SortNum:  i,
		})
	}
	if err := s.ann.CreateWithAttachments(ctx, a, attRows); err != nil {
		return nil, err
	}
	_ = s.oplog.Create(ctx, &domain.OperationLog{
		KindergartenID: kindergartenID,
		UserID:         adminUserID,
		Module:         "announcements",
		Action:         "create",
		TargetType:     "announcement",
		TargetID:       a.ID,
		Detail:         fmt.Sprintf("status=%d", st),
		IP:             ip,
		UserAgent:      truncateUA(ua),
	})
	out := map[string]interface{}{
		"id":     a.ID,
		"status": int(st),
	}
	if pubAt != nil {
		out["published_at"] = pubAt.UTC().Format(time.RFC3339)
	}
	return out, nil
}

// UpdateAnnouncementReq PUT /admin/announcements/:id
type UpdateAnnouncementReq struct {
	Title        *string                     `json:"title"`
	Content      *string                     `json:"content"`
	CoverURL     *string                     `json:"cover_url"`
	IsImportant  *bool                       `json:"is_important"`
	IsTop        *bool                       `json:"is_top"`
	TargetRoles  *[]string                   `json:"target_roles"`
	ScheduledAt  *string                     `json:"scheduled_at"`
	Attachments  *[]AnnouncementAttachmentIn `json:"attachments"`
}

func (s *AnnouncementService) UpdateAnnouncement(
	ctx context.Context,
	kindergartenID, adminUserID, announcementID uint64,
	req UpdateAnnouncementReq,
	ip, ua string,
) (map[string]interface{}, error) {
	if kindergartenID == 0 {
		return nil, errors.New("NO_KINDERGARTEN")
	}
	a, err := s.ann.FindAnnouncementByID(ctx, kindergartenID, announcementID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, errAnnouncementNotFound
	}
	patch := map[string]interface{}{}
	if req.Title != nil {
		t := strings.TrimSpace(*req.Title)
		if t == "" {
			return nil, errors.New("BAD_TITLE")
		}
		patch["title"] = t
	}
	if req.Content != nil {
		patch["content"] = *req.Content
	}
	if req.CoverURL != nil {
		patch["cover_url"] = strings.TrimSpace(*req.CoverURL)
	}
	if req.IsImportant != nil {
		patch["is_important"] = *req.IsImportant
	}
	if req.IsTop != nil {
		patch["is_top"] = *req.IsTop
	}
	if req.TargetRoles != nil {
		roles, err := normalizeTargetRoles(*req.TargetRoles)
		if err != nil {
			return nil, err
		}
		b, err := json.Marshal(roles)
		if err != nil {
			return nil, err
		}
		patch["target_roles"] = string(b)
	}
	if req.ScheduledAt != nil {
		sa := strings.TrimSpace(*req.ScheduledAt)
		if sa == "" {
			patch["scheduled_at"] = nil
		} else {
			t, e := time.Parse(time.RFC3339, sa)
			if e != nil {
				return nil, errors.New("BAD_SCHEDULED_AT")
			}
			tu := t.UTC()
			patch["scheduled_at"] = &tu
		}
	}
	if len(patch) == 0 && req.Attachments == nil {
		return nil, errors.New("EMPTY_UPDATE")
	}
	if len(patch) > 0 {
		if _, err := s.ann.UpdateColumns(ctx, kindergartenID, announcementID, patch); err != nil {
			return nil, err
		}
	}
	if req.Attachments != nil {
		var attRows []domain.AnnouncementAttachment
		for i, x := range *req.Attachments {
			u := strings.TrimSpace(x.FileURL)
			if u == "" {
				continue
			}
			attRows = append(attRows, domain.AnnouncementAttachment{
				FileURL:  u,
				FileName: strings.TrimSpace(x.FileName),
				FileSize: x.FileSize,
				SortNum:  i,
			})
		}
		if err := s.ann.ReplaceAttachments(ctx, announcementID, attRows); err != nil {
			return nil, err
		}
	}
	a2, err := s.ann.FindAnnouncementByID(ctx, kindergartenID, announcementID)
	if err != nil {
		return nil, err
	}
	if a2 == nil {
		return nil, errAnnouncementNotFound
	}
	_ = s.oplog.Create(ctx, &domain.OperationLog{
		KindergartenID: kindergartenID,
		UserID:         adminUserID,
		Module:         "announcements",
		Action:         "update",
		TargetType:     "announcement",
		TargetID:       announcementID,
		Detail:         "",
		IP:             ip,
		UserAgent:      truncateUA(ua),
	})
	return map[string]interface{}{
		"id":         announcementID,
		"updated_at": a2.UpdatedAt.UTC().Format(time.RFC3339),
	}, nil
}

func (s *AnnouncementService) DeleteAnnouncement(
	ctx context.Context,
	kindergartenID, adminUserID, announcementID uint64,
	ip, ua string,
) error {
	if kindergartenID == 0 {
		return errors.New("NO_KINDERGARTEN")
	}
	a, err := s.ann.FindAnnouncementByID(ctx, kindergartenID, announcementID)
	if err != nil {
		return err
	}
	if a == nil {
		return errAnnouncementNotFound
	}
	n, err := s.ann.SoftDeleteAnnouncement(ctx, kindergartenID, announcementID)
	if err != nil {
		return err
	}
	if n == 0 {
		return errAnnouncementNotFound
	}
	_ = s.oplog.Create(ctx, &domain.OperationLog{
		KindergartenID: kindergartenID,
		UserID:         adminUserID,
		Module:         "announcements",
		Action:         "delete",
		TargetType:     "announcement",
		TargetID:       announcementID,
		Detail:         "",
		IP:             ip,
		UserAgent:      truncateUA(ua),
	})
	return nil
}

type SetAnnouncementTopReq struct {
	IsTop bool `json:"is_top"`
}

func (s *AnnouncementService) SetAnnouncementTop(
	ctx context.Context,
	kindergartenID, adminUserID, announcementID uint64,
	req SetAnnouncementTopReq,
	ip, ua string,
) (map[string]interface{}, error) {
	if kindergartenID == 0 {
		return nil, errors.New("NO_KINDERGARTEN")
	}
	a, err := s.ann.FindAnnouncementByID(ctx, kindergartenID, announcementID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, errAnnouncementNotFound
	}
	if _, err := s.ann.UpdateColumns(ctx, kindergartenID, announcementID, map[string]interface{}{"is_top": req.IsTop}); err != nil {
		return nil, err
	}
	a2, err := s.ann.FindAnnouncementByID(ctx, kindergartenID, announcementID)
	if err != nil {
		return nil, err
	}
	if a2 == nil {
		return nil, errAnnouncementNotFound
	}
	_ = s.oplog.Create(ctx, &domain.OperationLog{
		KindergartenID: kindergartenID,
		UserID:         adminUserID,
		Module:         "announcements",
		Action:         "set_top",
		TargetType:     "announcement",
		TargetID:       announcementID,
		Detail:         fmt.Sprintf("is_top=%v", req.IsTop),
		IP:             ip,
		UserAgent:      truncateUA(ua),
	})
	return map[string]interface{}{
		"id":         announcementID,
		"is_top":     req.IsTop,
		"updated_at": a2.UpdatedAt.UTC().Format(time.RFC3339),
	}, nil
}
