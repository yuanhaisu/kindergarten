package model

import "time"

// AuditTargetType 审核对象类型
type AuditTargetType string

const (
	AuditTargetNotice AuditTargetType = "notice"
	AuditTargetGrowth AuditTargetType = "growth"
)

// AuditStatus 审核状态
type AuditStatus int8

const (
	AuditStatusPending  AuditStatus = 0 // 待审核
	AuditStatusApproved AuditStatus = 1 // 已通过
	AuditStatusRejected AuditStatus = 2 // 已驳回
)

// ContentAudit 内容审核记录
type ContentAudit struct {
	BaseModel
	KindergartenID uint64          `json:"kindergarten_id" gorm:"not null;index"`
	TargetType     AuditTargetType `json:"target_type" gorm:"size:20;not null"`
	TargetID       uint64          `json:"target_id" gorm:"not null;index"`
	SubmitterID    uint64          `json:"submitter_id" gorm:"not null"`
	ReviewerID     *uint64         `json:"reviewer_id,omitempty"`
	Status         AuditStatus     `json:"status" gorm:"default:0"`
	ReviewedAt     *time.Time      `json:"reviewed_at,omitempty"`
	RejectReason   string          `json:"reject_reason" gorm:"size:500"`
}
