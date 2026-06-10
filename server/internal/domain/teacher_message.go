package domain

import "time"

// TeacherMessageItem 老师端消息聚合项（非持久化，仅用于聚合查询结果）
type TeacherMessageItem struct {
	Type        string                 `json:"type"`
	Title       string                 `json:"title"`
	Content     string                 `json:"content"`
	ReferenceID uint64                 `json:"reference_id"`
	Extra       map[string]any `json:"extra"`
	CreatedAt   time.Time              `json:"created_at"`
}

// ---- 消息类型常量（用于聚合接口） ----
const (
	TeacherMsgTypeNotice = "notice"
	TeacherMsgTypeLeave  = "leave"
	TeacherMsgTypeSystem = "system"
)
