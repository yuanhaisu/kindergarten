package model

import "time"

// MessageType 站内消息类型
type MessageType string

const (
	MessageTypeNotice MessageType = "notice" // 通知消息
	MessageTypeLeave  MessageType = "leave"  // 请假消息
	MessageTypePickup MessageType = "pickup" // 接送消息
	MessageTypeGrowth MessageType = "growth" // 成长记录
	MessageTypeTask   MessageType = "task"   // 任务消息
	MessageTypeSystem MessageType = "system" // 系统消息
)

// Message 站内消息
type Message struct {
	BaseModel
	UserID      uint64      `json:"user_id" gorm:"not null;index"`
	Type        MessageType `json:"type" gorm:"size:20;not null"`
	Title       string      `json:"title" gorm:"size:200;not null"`
	Content     string      `json:"content" gorm:"size:500"`
	ReferenceID uint64      `json:"reference_id"`
	IsRead      bool        `json:"is_read" gorm:"default:false;index"`
	ReadAt      *time.Time  `json:"read_at,omitempty"`
}

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
