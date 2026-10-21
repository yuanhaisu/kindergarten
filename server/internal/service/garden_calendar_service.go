package service

// garden_calendar_service.go —— 园级日历事件（发布者：管理端园长角色 / 医生端，见 align-class-calendar D8）。
// 与老师端班级事件的差异：受众为全园（scope=kindergarten、class_id 为 NULL）；
// 发布/变更经确认时生成面向全园家长的通知（复用通知模块的定向与已读口径）。

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

type GardenCalendarService struct {
	db       *gorm.DB
	calendar *repository.CalendarRepo
	notices  *NoticeService
}

func NewGardenCalendarService(db *gorm.DB, calendar *repository.CalendarRepo, notices *NoticeService) *GardenCalendarService {
	return &GardenCalendarService{db: db, calendar: calendar, notices: notices}
}

var (
	ErrGardenNotFound      = errors.New("GARDEN_CALENDAR_NOT_FOUND")
	ErrGardenInvalidParams = errors.New("GARDEN_CALENDAR_INVALID_PARAMS")
	ErrGardenNoClasses     = errors.New("GARDEN_CALENDAR_NO_CLASSES")
)

type GardenCalendarInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	EventType   string `json:"event_type"`
	EventDate   string `json:"event_date"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	Location    string `json:"location"`
	IsAllDay    bool   `json:"is_all_day"`
	Status      string `json:"status"` // draft / published（缺省 published）
	Notify      bool   `json:"notify"` // 保存后是否通知全园家长（草稿忽略）
}

// ListGardenEvents 园级事件列表（status 可选过滤）。
func (s *GardenCalendarService) ListGardenEvents(ctx context.Context, kindergartenID uint64, start, end, status string) (map[string]any, error) {
	if kindergartenID == 0 {
		return nil, ErrGardenInvalidParams
	}
	if status != "" {
		if _, ok := calendarNormalizeStatus(status); !ok {
			return nil, ErrGardenInvalidParams
		}
	}
	rows, err := s.calendar.FindGardenEvents(ctx, kindergartenID, start, end, status)
	if err != nil {
		return nil, err
	}
	meta, err := loadCalendarTypeMeta(ctx, s.db, kindergartenID)
	if err != nil {
		return nil, err
	}
	list := make([]map[string]any, 0, len(rows))
	for i := range rows {
		list = append(list, teacherCalendarEventMap(&rows[i], meta))
	}
	return map[string]any{"list": list, "total": len(list)}, nil
}

// CreateGardenEvent 创建园级事件；发布且确认通知时生成全园通知。
func (s *GardenCalendarService) CreateGardenEvent(ctx context.Context, kindergartenID, userID uint64, req GardenCalendarInput) (map[string]any, error) {
	if kindergartenID == 0 || userID == 0 {
		return nil, ErrGardenInvalidParams
	}
	title := strings.TrimSpace(req.Title)
	if title == "" || strings.TrimSpace(req.EventDate) == "" {
		return nil, ErrGardenInvalidParams
	}
	day, err := time.ParseInLocation("2006-01-02", req.EventDate, time.Local)
	if err != nil {
		return nil, ErrGardenInvalidParams
	}
	status, ok := calendarNormalizeStatus(req.Status)
	if !ok {
		return nil, ErrGardenInvalidParams
	}
	row := &domain.CalendarEvent{
		KindergartenID: kindergartenID, ClassID: 0, CreatorID: userID,
		Title: title, Description: req.Description,
		EventType: calendarNormalizeEventType(req.EventType),
		Scope:     domain.CalendarScopeKindergarten, Status: status, Source: domain.CalendarSourceManual,
		EventDate: teacherPortalDay(day), StartTime: req.StartTime, EndTime: req.EndTime,
		Location: req.Location, IsAllDay: req.IsAllDay,
	}
	if err := s.calendar.CreateEvent(ctx, row); err != nil {
		return nil, err
	}
	if status == domain.CalendarStatusPublished && req.Notify {
		if err := s.pushGardenNotice(ctx, kindergartenID, userID, row, ""); err != nil {
			return nil, err
		}
	}
	return map[string]any{"id": row.ID, "created_at": row.CreatedAt.UTC().Format(time.RFC3339)}, nil
}

// UpdateGardenEvent 更新园级事件；已发布实质变更记录要旨；确认通知时生成全园变更通知。
func (s *GardenCalendarService) UpdateGardenEvent(ctx context.Context, kindergartenID, userID, id uint64, req GardenCalendarInput) (map[string]any, error) {
	existing, err := s.calendar.FindGardenEventByID(ctx, kindergartenID, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrGardenNotFound
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, ErrGardenInvalidParams
	}
	status, ok := calendarNormalizeStatus(req.Status)
	if !ok {
		return nil, ErrGardenInvalidParams
	}
	next := *existing
	next.Title = title
	next.Description = req.Description
	next.EventType = calendarNormalizeEventType(req.EventType)
	next.StartTime = req.StartTime
	next.EndTime = req.EndTime
	next.Location = req.Location
	next.IsAllDay = req.IsAllDay
	next.Status = status
	if req.EventDate != "" {
		day, err := time.ParseInLocation("2006-01-02", req.EventDate, time.Local)
		if err != nil {
			return nil, ErrGardenInvalidParams
		}
		next.EventDate = teacherPortalDay(day)
	}

	updates := map[string]any{
		"title": next.Title, "description": next.Description, "event_type": next.EventType,
		"start_time": next.StartTime, "end_time": next.EndTime, "location": next.Location,
		"is_all_day": next.IsAllDay, "status": next.Status, "event_date": next.EventDate,
	}
	// 变更要旨：仅针对已发布事件（草稿无人知晓）；有实质差异才落库。
	summary := ""
	if existing.Status == domain.CalendarStatusPublished {
		summary = calendarChangeSummary(existing, &next)
		if summary != "" {
			updates["revised_at"] = time.Now()
			updates["change_summary"] = summary
		}
	}
	affected, err := s.calendar.UpdateGardenEvent(ctx, kindergartenID, id, updates)
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, ErrGardenNotFound
	}
	if next.Status == domain.CalendarStatusPublished && req.Notify {
		if err := s.pushGardenNotice(ctx, kindergartenID, userID, &next, summary); err != nil {
			return nil, err
		}
	}
	return map[string]any{"id": id}, nil
}

func (s *GardenCalendarService) DeleteGardenEvent(ctx context.Context, kindergartenID, id uint64) error {
	affected, err := s.calendar.DeleteGardenEvent(ctx, kindergartenID, id)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrGardenNotFound
	}
	return nil
}

// pushGardenNotice 生成面向全园家长的通知：目标 = 本园全部在读班级快照
// （复用通知模块的展开与已读口径，不另建定向逻辑），并回写事件的推送记录位。
func (s *GardenCalendarService) pushGardenNotice(ctx context.Context, kindergartenID, userID uint64, event *domain.CalendarEvent, changeSummary string) error {
	var classIDs []uint64
	if err := s.db.WithContext(ctx).Model(&domain.Class{}).
		Where("kindergarten_id = ? AND status = 1", kindergartenID).
		Pluck("id", &classIDs).Error; err != nil {
		return err
	}
	if len(classIDs) == 0 {
		return ErrGardenNoClasses
	}
	res, err := s.notices.CreateNotice(ctx, kindergartenID, userID, CreateNoticeReq{
		ClassIDs: classIDs,
		Category: calendarNoticeCategory(event.EventType),
		Title:    calendarNoticeTitle(event, changeSummary),
		Content:  calendarNoticeContent(event, changeSummary),
	}, "", "")
	if err != nil {
		return err
	}
	updates := map[string]any{"is_published": true}
	if id, ok := res["id"].(uint64); ok && id > 0 {
		updates["notice_id"] = id
	}
	_, err = s.calendar.UpdateGardenEvent(ctx, kindergartenID, event.ID, updates)
	return err
}
