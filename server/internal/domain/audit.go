package domain

import "time"

type AuditTargetType string

const (
	AuditTargetNotice  AuditTargetType = "notice"
	AuditTargetGrowth  AuditTargetType = "growth"
	AuditTargetComment AuditTargetType = "comment"
)

type AuditStatus int8

const (
	AuditStatusPending  AuditStatus = 0
	AuditStatusApproved AuditStatus = 1
	AuditStatusRejected AuditStatus = 2
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

func (ContentAudit) TableName() string { return "content_audits" }
