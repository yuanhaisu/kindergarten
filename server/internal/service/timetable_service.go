package service

// timetable_service.go —— 课表子系统（align-class-calendar D5/D6）：
// 年级模板 + 班级微调（周模式）→ 物化同步为日历事件
// （source=timetable、source_slot 槽位去重、overridden 覆盖保护、滚动窗口、不产生通知）。
//
// 同步算法（每个班级、窗口内每一天）：
//  1. desired = 该星期几的模板条目，叠加班级微调（replace/remove/add）；
//  2. 现存「活体」课表实例：已覆盖（overridden）→ 不触碰；
//     未覆盖且槽位仍在 desired → 内容有差异则更新并移出 desired；
//     未覆盖但槽位不再需要 → **硬删**（系统自有行，释放唯一槽位键位）；
//  3. desired 剩余槽位：任一状态的行（含软删的覆盖行）已占据 → 跳过；否则创建。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/pkg/log"
)

const timetableWindowDays = 28 // 滚动窗口：默认 4 周（D5）

type TimetableService struct {
	db        *gorm.DB
	calendar  *repository.CalendarRepo
	repo      *repository.TimetableRepo
	settings  *repository.KindergartenAppSettingRepo
	startOnce sync.Once
}

func NewTimetableService(
	db *gorm.DB,
	calendar *repository.CalendarRepo,
	repo *repository.TimetableRepo,
	settings *repository.KindergartenAppSettingRepo,
) *TimetableService {
	return &TimetableService{db: db, calendar: calendar, repo: repo, settings: settings}
}

// NewTimetableServiceForApp 应用组装用：额外启动每日滚动任务。
// 测试直接构造裸服务（NewTimetableService），避免后台 goroutine 干扰。
func NewTimetableServiceForApp(
	db *gorm.DB,
	calendar *repository.CalendarRepo,
	repo *repository.TimetableRepo,
	settings *repository.KindergartenAppSettingRepo,
) *TimetableService {
	s := NewTimetableService(db, calendar, repo, settings)
	s.StartDailySync()
	return s
}

// ── 输入类型 ──────────────────────────────────────────────────────────

type TimetableEntryInput struct {
	Weekday   int    `json:"weekday"` // 1-7（周一=1）
	Period    int    `json:"period"`  // 第几节（正整数）
	Title     string `json:"title"`
	Location  string `json:"location"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

type TimetableOverrideInput struct {
	Weekday   int    `json:"weekday"`
	Period    int    `json:"period"`
	Action    string `json:"action"` // replace / remove / add
	Title     string `json:"title"`
	Location  string `json:"location"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

var ErrTimetableInvalidParams = errors.New("TIMETABLE_INVALID_PARAMS")

// ── 年级模板 ──────────────────────────────────────────────────────────

func (s *TimetableService) GetTemplate(ctx context.Context, kindergartenID uint64, grade string) (map[string]any, error) {
	grade = strings.TrimSpace(grade)
	if kindergartenID == 0 || grade == "" {
		return nil, ErrTimetableInvalidParams
	}
	rows, err := s.repo.ListTemplates(ctx, kindergartenID, grade)
	if err != nil {
		return nil, err
	}
	entries := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		entries = append(entries, timetableEntryMap(r.Weekday, r.Period, r.Title, r.Location, r.StartTime, r.EndTime))
	}
	return map[string]any{"grade": grade, "entries": entries}, nil
}

// SaveTemplate 全量替换年级模板；保存后重算未来窗口（未覆盖实例更新，已覆盖/过去不动）。
func (s *TimetableService) SaveTemplate(ctx context.Context, kindergartenID uint64, grade string, entries []TimetableEntryInput) (map[string]any, error) {
	grade = strings.TrimSpace(grade)
	if kindergartenID == 0 || grade == "" {
		return nil, ErrTimetableInvalidParams
	}
	rows := make([]domain.TimetableTemplate, 0, len(entries))
	for _, e := range entries {
		if err := validateTimetableSlot(e.Weekday, e.Period); err != nil {
			return nil, err
		}
		title := strings.TrimSpace(e.Title)
		if title == "" {
			return nil, ErrTimetableInvalidParams
		}
		rows = append(rows, domain.TimetableTemplate{
			Weekday: e.Weekday, Period: e.Period, Title: title,
			Location: e.Location, StartTime: e.StartTime, EndTime: e.EndTime,
		})
	}
	if err := s.repo.ReplaceTemplates(ctx, kindergartenID, grade, rows); err != nil {
		return nil, err
	}
	if err := s.resyncIfEnabled(ctx, kindergartenID); err != nil {
		return nil, err
	}
	return s.GetTemplate(ctx, kindergartenID, grade)
}

// ── 班级微调 ──────────────────────────────────────────────────────────

func (s *TimetableService) GetClassOverrides(ctx context.Context, kindergartenID, classID uint64) (map[string]any, error) {
	if kindergartenID == 0 || classID == 0 {
		return nil, ErrTimetableInvalidParams
	}
	rows, err := s.repo.ListOverrides(ctx, kindergartenID, classID)
	if err != nil {
		return nil, err
	}
	overrides := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		m := timetableEntryMap(r.Weekday, r.Period, r.Title, r.Location, r.StartTime, r.EndTime)
		m["action"] = r.Action
		overrides = append(overrides, m)
	}
	return map[string]any{"class_id": classID, "overrides": overrides}, nil
}

// SaveClassOverrides 全量替换班级微调；保存后重算未来窗口。
func (s *TimetableService) SaveClassOverrides(ctx context.Context, kindergartenID, classID uint64, entries []TimetableOverrideInput) (map[string]any, error) {
	if kindergartenID == 0 || classID == 0 {
		return nil, ErrTimetableInvalidParams
	}
	rows := make([]domain.TimetableOverride, 0, len(entries))
	for _, e := range entries {
		if err := validateTimetableSlot(e.Weekday, e.Period); err != nil {
			return nil, err
		}
		action := strings.TrimSpace(e.Action)
		switch action {
		case "replace", "remove", "add":
		default:
			return nil, ErrTimetableInvalidParams
		}
		title := strings.TrimSpace(e.Title)
		if action != "remove" && title == "" {
			return nil, ErrTimetableInvalidParams
		}
		rows = append(rows, domain.TimetableOverride{
			Weekday: e.Weekday, Period: e.Period, Action: action, Title: title,
			Location: e.Location, StartTime: e.StartTime, EndTime: e.EndTime,
		})
	}
	if err := s.repo.ReplaceOverrides(ctx, kindergartenID, classID, rows); err != nil {
		return nil, err
	}
	if err := s.resyncIfEnabled(ctx, kindergartenID); err != nil {
		return nil, err
	}
	return s.GetClassOverrides(ctx, kindergartenID, classID)
}

// ── 园所开关 ──────────────────────────────────────────────────────────

func (s *TimetableService) GetSetting(ctx context.Context, kindergartenID uint64) (map[string]any, error) {
	enabled, err := loadTimetableEnabled(ctx, s.db, kindergartenID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"timetable_enabled": enabled}, nil
}

// UpdateSetting 更新开关（读-改-写 ExtraConfig，保留其他配置块）；开启时立即同步一次。
func (s *TimetableService) UpdateSetting(ctx context.Context, kindergartenID uint64, enabled bool) (map[string]any, error) {
	if kindergartenID == 0 {
		return nil, ErrTimetableInvalidParams
	}
	row, err := s.settings.FindByKindergartenID(ctx, kindergartenID)
	if err != nil {
		return nil, err
	}
	raw := "{}"
	if row != nil && strings.TrimSpace(row.ExtraConfig) != "" {
		raw = row.ExtraConfig
	}
	root := map[string]json.RawMessage{}
	if err := json.Unmarshal([]byte(raw), &root); err != nil {
		root = map[string]json.RawMessage{}
	}
	cal := map[string]json.RawMessage{}
	if b, ok := root["calendar"]; ok {
		_ = json.Unmarshal(b, &cal)
	}
	cal["timetable_enabled"] = json.RawMessage([]byte(fmt.Sprintf("%t", enabled)))
	cb, err := json.Marshal(cal)
	if err != nil {
		return nil, err
	}
	root["calendar"] = cb
	out, err := json.Marshal(root)
	if err != nil {
		return nil, err
	}
	if row == nil {
		if err := s.settings.Create(ctx, &domain.KindergartenAppSetting{
			KindergartenID: kindergartenID, ExtraConfig: string(out),
		}); err != nil {
			return nil, err
		}
	} else {
		row.ExtraConfig = string(out)
		if err := s.settings.Save(ctx, row); err != nil {
			return nil, err
		}
	}
	if enabled {
		if _, err := s.SyncKindergarten(ctx, kindergartenID); err != nil {
			return nil, err
		}
	}
	return map[string]any{"timetable_enabled": enabled}, nil
}

// ── 同步引擎 ──────────────────────────────────────────────────────────

// TimetableSyncStats 一次同步的计数（供手动同步响应与日志）。
type TimetableSyncStats struct {
	Created, Updated, Deleted, Skipped int
}

// SyncKindergarten 重算一个园所全部在读班级的未来滚动窗口（幂等）。
func (s *TimetableService) SyncKindergarten(ctx context.Context, kindergartenID uint64) (map[string]any, error) {
	var classes []domain.Class
	if err := s.db.WithContext(ctx).
		Where("kindergarten_id = ? AND status = 1", kindergartenID).
		Find(&classes).Error; err != nil {
		return nil, err
	}
	now := time.Now()
	total := TimetableSyncStats{}
	for _, cl := range classes {
		st, err := s.syncClass(ctx, kindergartenID, cl.ID, cl.Grade, now)
		if err != nil {
			return nil, err
		}
		total.Created += st.Created
		total.Updated += st.Updated
		total.Deleted += st.Deleted
		total.Skipped += st.Skipped
	}
	return map[string]any{
		"synced": true, "created": total.Created, "updated": total.Updated,
		"deleted": total.Deleted, "skipped_overridden": total.Skipped,
	}, nil
}

type timetableDesired struct {
	Title     string
	Location  string
	StartTime string
	EndTime   string
}

func (s *TimetableService) syncClass(ctx context.Context, kindergartenID, classID uint64, grade string, now time.Time) (TimetableSyncStats, error) {
	var stats TimetableSyncStats

	templateRows, err := s.repo.ListTemplates(ctx, kindergartenID, grade)
	if err != nil {
		return stats, err
	}
	tmpl := map[int]map[int]domain.TimetableTemplate{}
	for _, t := range templateRows {
		if tmpl[t.Weekday] == nil {
			tmpl[t.Weekday] = map[int]domain.TimetableTemplate{}
		}
		tmpl[t.Weekday][t.Period] = t
	}
	overrideRows, err := s.repo.ListOverrides(ctx, kindergartenID, classID)
	if err != nil {
		return stats, err
	}
	ovr := map[int]map[int]domain.TimetableOverride{}
	for _, o := range overrideRows {
		if ovr[o.Weekday] == nil {
			ovr[o.Weekday] = map[int]domain.TimetableOverride{}
		}
		ovr[o.Weekday][o.Period] = o
	}

	start := teacherPortalDay(now)
	for d := 0; d < timetableWindowDays; d++ {
		date := start.AddDate(0, 0, d)
		wd := isoWeekday(date)

		// 1) desired：模板叠加微调
		desired := map[string]timetableDesired{}
		for period, t := range tmpl[wd] {
			desired[timetableSlotKey(wd, period)] = timetableDesired{
				Title: t.Title, Location: t.Location, StartTime: t.StartTime, EndTime: t.EndTime,
			}
		}
		for period, o := range ovr[wd] {
			key := timetableSlotKey(wd, period)
			switch o.Action {
			case "remove":
				delete(desired, key)
			case "replace":
				desired[key] = timetableDesired{Title: o.Title, Location: o.Location, StartTime: o.StartTime, EndTime: o.EndTime}
			case "add":
				if _, exists := desired[key]; !exists {
					desired[key] = timetableDesired{Title: o.Title, Location: o.Location, StartTime: o.StartTime, EndTime: o.EndTime}
				}
			}
		}

		// 2) 现存活体课表实例（软删行由 gorm 默认作用域排除）
		var live []domain.CalendarEvent
		if err := s.db.WithContext(ctx).
			Where("class_id = ? AND event_date = ? AND source = ?", classID, date, domain.CalendarSourceTimetable).
			Find(&live).Error; err != nil {
			return stats, err
		}
		for i := range live {
			row := &live[i]
			if row.Overridden {
				stats.Skipped++ // 老师覆盖过：永久不触碰
				continue
			}
			if want, ok := desired[row.SourceSlot]; ok {
				updates := map[string]any{}
				if row.Title != want.Title {
					updates["title"] = want.Title
				}
				if row.Location != want.Location {
					updates["location"] = want.Location
				}
				if row.StartTime != want.StartTime {
					updates["start_time"] = want.StartTime
				}
				if row.EndTime != want.EndTime {
					updates["end_time"] = want.EndTime
				}
				if len(updates) > 0 {
					if _, err := s.calendar.UpdateEvent(ctx, kindergartenID, classID, row.ID, updates); err != nil {
						return stats, err
					}
					stats.Updated++
				}
				delete(desired, row.SourceSlot)
			} else {
				// 槽位不再需要：系统自有行（未被覆盖）硬删，释放唯一槽位键位
				if err := s.calendar.HardDeleteEvent(ctx, kindergartenID, row.ID); err != nil {
					return stats, err
				}
				stats.Deleted++
			}
		}

		// 3) 新增：槽位未被任何行占据（含软删的覆盖行）才创建
		for key, want := range desired {
			taken, err := s.calendar.SlotTaken(ctx, classID, date, key)
			if err != nil {
				return stats, err
			}
			if taken {
				stats.Skipped++
				continue
			}
			row := &domain.CalendarEvent{
				KindergartenID: kindergartenID, ClassID: classID, CreatorID: 0,
				Title: want.Title, EventType: domain.EventTypeActivity,
				Scope: domain.CalendarScopeClass, Status: domain.CalendarStatusPublished,
				Source: domain.CalendarSourceTimetable, SourceSlot: key,
				EventDate: date, StartTime: want.StartTime, EndTime: want.EndTime, Location: want.Location,
			}
			if err := s.calendar.CreateEvent(ctx, row); err != nil {
				return stats, err
			}
			stats.Created++
		}
	}
	return stats, nil
}

// resyncIfEnabled 课表变更后重算（仅当园所开启课表）。
func (s *TimetableService) resyncIfEnabled(ctx context.Context, kindergartenID uint64) error {
	enabled, err := loadTimetableEnabled(ctx, s.db, kindergartenID)
	if err != nil {
		return err
	}
	if !enabled {
		return nil
	}
	_, err = s.SyncKindergarten(ctx, kindergartenID)
	return err
}

// ── 每日滚动任务（D6）────────────────────────────────────────────────

// StartDailySync 启动每日滚动补齐（幂等，应用组装用；测试不调用）。
// 启动后先跑一次（服务重启后的补齐），此后每 24h 一次。
func (s *TimetableService) StartDailySync() {
	s.startOnce.Do(func() {
		go func() {
			for {
				s.RunDailySyncOnce(context.Background())
				time.Sleep(24 * time.Hour)
			}
		}()
	})
}

// RunDailySyncOnce 对全部开启课表的园所执行一次滚动补齐。
// 失败显式告警（不静默）：单个园所失败不影响其他园所。
func (s *TimetableService) RunDailySyncOnce(ctx context.Context) {
	var kgs []domain.Kindergarten
	if err := s.db.WithContext(ctx).Find(&kgs).Error; err != nil {
		log.L(ctx).Error("timetable daily sync: list kindergartens failed", "err", err)
		return
	}
	for _, kg := range kgs {
		enabled, err := loadTimetableEnabled(ctx, s.db, kg.ID)
		if err != nil {
			log.L(ctx).Error("timetable daily sync: load switch failed", "kindergarten", kg.ID, "err", err)
			continue
		}
		if !enabled {
			continue
		}
		if _, err := s.SyncKindergarten(ctx, kg.ID); err != nil {
			log.L(ctx).Error("timetable daily sync failed", "kindergarten", kg.ID, "err", err)
		}
	}
}

// ── 辅助 ──────────────────────────────────────────────────────────────

func validateTimetableSlot(weekday, period int) error {
	if weekday < 1 || weekday > 7 || period <= 0 {
		return ErrTimetableInvalidParams
	}
	return nil
}

func timetableSlotKey(weekday, period int) string {
	return fmt.Sprintf("%d-%d", weekday, period)
}

func timetableEntryMap(weekday, period int, title, location, startTime, endTime string) map[string]any {
	return map[string]any{
		"weekday": weekday, "period": period, "title": title,
		"location": location, "start_time": startTime, "end_time": endTime,
	}
}

// isoWeekday Go 的 Sunday=0 转 ISO（1=周一 … 7=周日），与契约字段一致。
func isoWeekday(t time.Time) int {
	wd := int(t.Weekday())
	if wd == 0 {
		return 7
	}
	return wd
}
