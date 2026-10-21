package domain

import "time"

type MessageType string

const (
	MessageTypeNotice MessageType = "notice"
	MessageTypeLeave  MessageType = "leave"
	MessageTypePickup MessageType = "pickup"
	MessageTypeGrowth MessageType = "growth"
	MessageTypeTask   MessageType = "task"
	MessageTypeSystem MessageType = "system"
)

// Message 站内消息
type Message struct {
	BaseModel
	UserID      uint64     `gorm:"index;not null" json:"-"`
	Type        string     `gorm:"size:32;index;not null" json:"type"`
	Title       string     `gorm:"size:256;not null" json:"title"`
	Content     string     `gorm:"type:text" json:"content"`
	ReferenceID uint64     `gorm:"default:0" json:"reference_id"`
	ActionPath  string     `gorm:"size:255" json:"action_path"`
	IsRead      bool       `gorm:"default:false;index" json:"is_read"`
	ReadAt      *time.Time `json:"read_at"`

	User *User `json:"-" gorm:"foreignKey:UserID"`
}

func (Message) TableName() string { return "messages" }

// WechatPushLog 微信推送记录
type WechatPushLog struct {
	BaseModel
	UserID     uint64     `json:"user_id" gorm:"not null;index"`
	TemplateID string     `json:"template_id" gorm:"size:100"`
	Content    string     `json:"content" gorm:"type:text"`
	Status     int8       `json:"status" gorm:"default:0;comment:0待发 1成功 -1失败"`
	ErrorMsg   string     `json:"error_msg" gorm:"size:500"`
	SentAt     *time.Time `json:"sent_at,omitempty"`
}

func (WechatPushLog) TableName() string { return "wechat_push_logs" }
