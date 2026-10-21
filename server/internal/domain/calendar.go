package domain

import "time"

type EventType string

const (
	EventTypeNotice   EventType = "notice"
	EventTypeActivity EventType = "activity"
	EventTypeHoliday  EventType = "holiday"
	EventTypeMeeting  EventType = "meeting"
	EventTypeCheckup  EventType = "checkup"
)

// 日历事件受众范围 / 发布状态 / 来源（align-class-calendar 变更引入）。
const (
	CalendarScopeClass        = "class"        // 班级级：class_id 指向班级
	CalendarScopeKindergarten = "kindergarten" // 园级：class_id 为 NULL，全园可见

	CalendarStatusDraft     = "draft"     // 草稿：家长不可见
	CalendarStatusPublished = "published" // 已发布：家长可见

	CalendarSourceManual    = "manual"    // 手动创建
	CalendarSourceTimetable = "timetable" // 课表物化（source_slot 去重，overridden 后同步不触碰）
)

// CalendarEvent 班级日历事件。
//
// 语义（align-class-calendar）：
//   - 受众范围由 scope 决定：class（本班）或 kindergarten（园级，class_id 为 NULL）；
//   - 家长可见性由 status 决定（仅 published）；is_published 收窄为「是否已通知过」的记录位；
//   - source=timetable 的行是课表物化实例：source_slot 为唯一槽位键（“星期-节次”），
//     被老师调整/取消后 overridden=true，同步引擎不再触碰该槽位。
type CalendarEvent struct {
	SoftDeleteModel
	KindergartenID uint64 `json:"kindergarten_id" gorm:"not null;index"`
	// ClassID 班级事件为真实班级 id；园级事件为 NULL（default:null 使 0 在插入时省略该列）。
	// 表上有外键 fk_calendar_events_class（→ classes.id, RESTRICT），MySQL 外键对 NULL 不检查，
	// 因此园级事件既不受班级约束、真实班级行仍受 RESTRICT 保护。
	ClassID     uint64    `json:"class_id" gorm:"index;uniqueIndex:uk_calendar_slot,priority:1;default:null"`
	// CreatorID 系统物化（课表同步）行可为 NULL（default:null 使 0 在插入时省略该列）；
	// 表上有外键 fk_calendar_events_creator（→ users.id, RESTRICT），外键对 NULL 不检查。
	CreatorID uint64 `json:"creator_id" gorm:"default:null"`
	Title       string    `json:"title" gorm:"size:200;not null"`
	Description string    `json:"description" gorm:"type:text"`
	EventType   EventType `json:"event_type" gorm:"size:20;not null"`
	Scope       string    `json:"scope" gorm:"size:20;not null;default:class;comment:class 班级 / kindergarten 园级"`
	Status      string    `json:"status" gorm:"size:20;not null;default:published;comment:draft 草稿 / published 已发布（家长仅见 published）"`
	Source      string    `json:"source" gorm:"size:20;not null;default:manual;comment:manual 手动 / timetable 课表物化"`
	// SourceSlot 课表槽位键（如 "3-2" = 周三第 2 节）。default:null 让手动事件在插入时省略该列 → NULL，
	// 唯一索引 uk_calendar_slot 不约束 NULL，手动事件因此不受槽位唯一性影响。
	SourceSlot string `json:"source_slot" gorm:"size:40;uniqueIndex:uk_calendar_slot,priority:3;default:null"`
	Overridden bool   `json:"overridden" gorm:"not null;default:false;comment:课表实例被老师覆盖（调整/取消）后为 true，同步不触碰"`
	// RevisedAt / ChangeSummary：最近一次实质变更（家长端变更提示窗口用）；由服务端生成，客户端不回传。
	RevisedAt     *time.Time `json:"revised_at,omitempty"`
	ChangeSummary string     `json:"change_summary" gorm:"size:255"`
	EventDate     time.Time  `json:"event_date" gorm:"type:date;not null;index;uniqueIndex:uk_calendar_slot,priority:2"`
	StartTime     string     `json:"start_time" gorm:"size:10"`
	EndTime       string     `json:"end_time" gorm:"size:10"`
	Location      string     `json:"location" gorm:"size:200"`
	IsAllDay      bool       `json:"is_all_day" gorm:"default:false"`
	// Color 历史字段：客户端渲染色已改由字典 calendar_event_type 解析，本字段不再用于渲染。
	Color string `json:"color" gorm:"size:20"`
	// Importance / Cover 无设计出处且产品未裁决（见 align-class-calendar proposal 阻塞清单），暂不外露。
	Importance  string  `json:"importance" gorm:"size:20;default:normal;comment:normal普通 important重要 urgent紧急"`
	Cover       string  `json:"cover" gorm:"size:500"`
	IsPublished bool    `json:"is_published" gorm:"default:true;comment:收窄为推送记录位；家长可见性由 status 决定"`
	NoticeID    *uint64 `json:"notice_id,omitempty"`

	Creator *User  `json:"creator,omitempty" gorm:"foreignKey:CreatorID"`
	Class   *Class `json:"class,omitempty" gorm:"foreignKey:ClassID"`
}

func (CalendarEvent) TableName() string { return "calendar_events" }
