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

// CalendarEvent 班级日历事件
type CalendarEvent struct {
	SoftDeleteModel
	KindergartenID uint64    `json:"kindergarten_id" gorm:"not null;index"`
	ClassID        uint64    `json:"class_id" gorm:"not null;index"`
	CreatorID      uint64    `json:"creator_id" gorm:"not null"`
	Title          string    `json:"title" gorm:"size:200;not null"`
	Description    string    `json:"description" gorm:"type:text"`
	EventType      EventType `json:"event_type" gorm:"size:20;not null"`
	EventDate      time.Time `json:"event_date" gorm:"type:date;not null;index"`
	StartTime      string    `json:"start_time" gorm:"size:10"`
	EndTime        string    `json:"end_time" gorm:"size:10"`
	Location       string    `json:"location" gorm:"size:200"`
	IsAllDay       bool      `json:"is_all_day" gorm:"default:false"`
	Color          string    `json:"color" gorm:"size:20"`
	Importance     string    `json:"importance" gorm:"size:20;default:normal;comment:normal普通 important重要 urgent紧急"`
	Cover          string    `json:"cover" gorm:"size:500"`
	IsPublished    bool      `json:"is_published" gorm:"default:true;comment:是否发布并通知家长"`
	NoticeID       *uint64   `json:"notice_id,omitempty"`

	Creator *User  `json:"creator,omitempty" gorm:"foreignKey:CreatorID"`
	Class   *Class `json:"class,omitempty" gorm:"foreignKey:ClassID"`
}

func (CalendarEvent) TableName() string { return "calendar_events" }
