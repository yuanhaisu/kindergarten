package service

// calendar_support.go —— 日历模块的服务层共享辅助（align-class-calendar）：
// 字典元数据解析（label/color 的唯一来源是园所字典 calendar_event_type）、
// 事件响应映射（内部字段不外泄）、变更要旨、通知内容编排。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

// calendarTypeMeta 日历事件类型的展示元数据（来源：字典项 Extra）。
type calendarTypeMeta struct {
	Label string
	Color string
}

// loadCalendarTypeMeta 读取园所日历类型字典：item_key → {label, color}。
// Table() 裸查询不带软删作用域，须显式过滤 deleted_at。
func loadCalendarTypeMeta(ctx context.Context, db *gorm.DB, kindergartenID uint64) (map[string]calendarTypeMeta, error) {
	var rows []struct {
		ItemKey   string
		ItemLabel string
		Extra     string
	}
	err := db.WithContext(ctx).
		Table("sys_dict_items AS i").
		Select("i.item_key, i.item_label, i.extra").
		Joins("JOIN sys_dict_types AS t ON t.id = i.type_id").
		Where("t.kindergarten_id = ? AND t.type_key = ?", kindergartenID, "calendar_event_type").
		Where("t.deleted_at IS NULL AND i.deleted_at IS NULL AND i.is_enabled = 1").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	meta := make(map[string]calendarTypeMeta, len(rows))
	for _, r := range rows {
		m := calendarTypeMeta{Label: r.ItemLabel}
		if r.Extra != "" {
			var extra struct {
				Color string `json:"color"`
			}
			if json.Unmarshal([]byte(r.Extra), &extra) == nil {
				m.Color = extra.Color
			}
		}
		meta[r.ItemKey] = m
	}
	return meta, nil
}

// calendarLabel 按字典解析类型标签与颜色；字典缺项时降级为类型键 + 空色。
func calendarLabel(meta map[string]calendarTypeMeta, t domain.EventType) (string, string) {
	if m, ok := meta[string(t)]; ok {
		label := m.Label
		if label == "" {
			label = string(t)
		}
		return label, m.Color
	}
	return string(t), ""
}

// calendarNormalizeStatus 缺省 published；非法值返回 ok=false（调用方按无效参数处理）。
func calendarNormalizeStatus(s string) (string, bool) {
	switch strings.TrimSpace(s) {
	case "":
		return domain.CalendarStatusPublished, true
	case domain.CalendarStatusDraft:
		return domain.CalendarStatusDraft, true
	case domain.CalendarStatusPublished:
		return domain.CalendarStatusPublished, true
	default:
		return "", false
	}
}

func calendarNormalizeEventType(s string) domain.EventType {
	t := domain.EventType(strings.TrimSpace(s))
	if t == "" {
		return domain.EventTypeActivity
	}
	return t
}

func calendarNullString(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}

// teacherCalendarEventMap 老师端事件响应：契约字段集（kindergarten_id/creator_id/is_published/notice_id 等内部字段不外泄）。
func teacherCalendarEventMap(row *domain.CalendarEvent, meta map[string]calendarTypeMeta) map[string]any {
	label, color := calendarLabel(meta, row.EventType)
	return map[string]any{
		"id": row.ID, "title": row.Title, "description": row.Description,
		"event_type": row.EventType, "event_type_label": label,
		"scope": row.Scope, "status": row.Status,
		"event_date": row.EventDate.Format("2006-01-02"),
		"start_time": calendarNullString(row.StartTime), "end_time": calendarNullString(row.EndTime),
		"location": row.Location, "is_all_day": row.IsAllDay, "color": color,
	}
}

// calendarChangeWindow 变更提示窗口：近 7 天内修订、且尚未发生的事件才下发变更要旨。
const calendarChangeWindow = 7 * 24 * time.Hour

// parentCalendarEventMap 家长端事件响应：仅展示所需字段 + 窗口内变更要旨。
func parentCalendarEventMap(row *domain.CalendarEvent, meta map[string]calendarTypeMeta, now time.Time) map[string]any {
	label, color := calendarLabel(meta, row.EventType)
	m := map[string]any{
		"id": row.ID, "title": row.Title, "description": row.Description,
		"event_type": row.EventType, "event_type_label": label,
		"event_date": row.EventDate.Format("2006-01-02"),
		"start_time": calendarNullString(row.StartTime), "end_time": calendarNullString(row.EndTime),
		"location": row.Location, "is_all_day": row.IsAllDay, "color": color,
		"change_summary": nil,
	}
	// 「尚未发生」按日期字符串比较（event_date 为 DATE 列，跨时区解析安全）：今天及以后算未发生。
	if row.RevisedAt != nil && strings.TrimSpace(row.ChangeSummary) != "" &&
		row.RevisedAt.After(now.Add(-calendarChangeWindow)) &&
		row.EventDate.Format("2006-01-02") >= now.Format("2006-01-02") {
		m["change_summary"] = row.ChangeSummary
	}
	return m
}

// calendarChangeSummary 对比新旧内容生成变更要旨（家长端可读）；无实质差异返回空串。
// 日期比较用格式化字符串：写入经 teacherPortalDay（Asia/Shanghai），读回可能按 DSN 时区解析。
func calendarChangeSummary(old, next *domain.CalendarEvent) string {
	var parts []string
	if old.EventDate.Format("2006-01-02") != next.EventDate.Format("2006-01-02") {
		parts = append(parts, fmt.Sprintf("日期 %s → %s",
			old.EventDate.Format("2006-01-02"), next.EventDate.Format("2006-01-02")))
	}
	if old.StartTime != next.StartTime || old.EndTime != next.EndTime {
		parts = append(parts, fmt.Sprintf("时间 %s → %s", calendarTimeRange(old), calendarTimeRange(next)))
	}
	if old.Location != next.Location {
		parts = append(parts, fmt.Sprintf("地点 %s → %s", calendarDash(old.Location), calendarDash(next.Location)))
	}
	if old.Title != next.Title {
		parts = append(parts, "标题已更新")
	}
	if old.Description != next.Description {
		parts = append(parts, "内容已更新")
	}
	if old.EventType != next.EventType {
		parts = append(parts, "类型已更新")
	}
	if len(parts) == 0 {
		return ""
	}
	out := strings.Join(parts, "；")
	if runes := []rune(out); len(runes) > 120 {
		out = string(runes[:120]) + "…"
	}
	return out
}

func calendarTimeRange(e *domain.CalendarEvent) string {
	if strings.TrimSpace(e.StartTime) == "" {
		return "未设"
	}
	if strings.TrimSpace(e.EndTime) == "" {
		return e.StartTime
	}
	return e.StartTime + "-" + e.EndTime
}

func calendarDash(v string) string {
	if strings.TrimSpace(v) == "" {
		return "无"
	}
	return v
}

func calendarNoticeTitle(event *domain.CalendarEvent, changeSummary string) string {
	if changeSummary != "" {
		return "【变更】" + event.Title
	}
	return event.Title
}

func calendarNoticeContent(event *domain.CalendarEvent, changeSummary string) string {
	var b strings.Builder
	if changeSummary != "" {
		b.WriteString(changeSummary)
		b.WriteString("\n")
	}
	if d := strings.TrimSpace(event.Description); d != "" {
		b.WriteString(d)
		b.WriteString("\n")
	}
	b.WriteString("时间：" + event.EventDate.Format("2006-01-02"))
	if r := calendarTimeRange(event); r != "未设" {
		b.WriteString(" " + r)
	}
	if loc := strings.TrimSpace(event.Location); loc != "" {
		b.WriteString("\n地点：" + loc)
	}
	return b.String()
}

// calendarNoticeCategory 事件类型 → 通知分类（复用既有 NoticeCategory 枚举）。
func calendarNoticeCategory(t domain.EventType) string {
	switch t {
	case domain.EventTypeActivity:
		return string(domain.NoticeCategoryActivity)
	case domain.EventTypeCheckup:
		return string(domain.NoticeCategoryHealth)
	default:
		return string(domain.NoticeCategoryOther)
	}
}

// loadTimetableEnabled 读取园所课表开关（ExtraConfig.calendar.timetable_enabled）。
// 无设置行/无该块 = 未开启（课表是园所级 opt-in，缺省关闭）。
func loadTimetableEnabled(ctx context.Context, db *gorm.DB, kindergartenID uint64) (bool, error) {
	var row domain.KindergartenAppSetting
	err := db.WithContext(ctx).Where("kindergarten_id = ?", kindergartenID).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return parseTimetableEnabled(row.ExtraConfig), nil
}

func parseTimetableEnabled(raw string) bool {
	if strings.TrimSpace(raw) == "" {
		return false
	}
	var root struct {
		Calendar struct {
			TimetableEnabled bool `json:"timetable_enabled"`
		} `json:"calendar"`
	}
	if json.Unmarshal([]byte(raw), &root) != nil {
		return false
	}
	return root.Calendar.TimetableEnabled
}
