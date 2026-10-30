package domain

import "time"

// TeacherMessageRead 老师端消息聚合项的已读记录。
//
// 与通知/消息表内的读状态是两套口径：通知已读率来自 notice_read_logs（家长读），
// 系统消息的 messages.is_read 也不参与；本表只表达「当前老师是否点开过这条聚合消息」
// （design teacher/18-messages-index：点开即读、已读降灰）。
//
// 软删不参与：记录写入后长期有效，没有 deleted_at；唯一键
// (user_id, message_type, reference_id) 保证重复标记幂等。
type TeacherMessageRead struct {
	ID          uint64    `gorm:"primarykey" json:"id"`
	UserID      uint64    `gorm:"not null;uniqueIndex:uk_teacher_message_read" json:"-"`
	MessageType string    `gorm:"size:32;not null;uniqueIndex:uk_teacher_message_read" json:"message_type"`
	ReferenceID uint64    `gorm:"not null;uniqueIndex:uk_teacher_message_read" json:"reference_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (TeacherMessageRead) TableName() string { return "teacher_message_reads" }
