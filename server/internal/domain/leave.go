package domain

import "time"

type LeaveStatus int8

const (
	LeaveStatusPending  LeaveStatus = 0
	LeaveStatusApproved LeaveStatus = 1
	LeaveStatusRejected LeaveStatus = 2
	LeaveStatusCanceled LeaveStatus = 3
)

// LeaveRequest 请假申请
type LeaveRequest struct {
	SoftDeleteModel
	KindergartenID uint64      `json:"kindergarten_id" gorm:"not null;index"`
	ClassID        uint64      `json:"class_id" gorm:"not null;index"`
	StudentID      uint64      `json:"student_id" gorm:"not null;index"`
	ApplicantID    uint64      `json:"applicant_id" gorm:"not null"`
	Reason         string      `json:"reason" gorm:"size:500;not null"`
	StartDate      time.Time   `json:"start_date" gorm:"not null;index"`
	EndDate        time.Time   `json:"end_date" gorm:"not null"`
	Days           int         `json:"days"`
	Status         LeaveStatus `json:"status" gorm:"default:0"`
	ReviewerID     *uint64     `json:"reviewer_id,omitempty"`
	ReviewedAt     *time.Time  `json:"reviewed_at,omitempty"`
	ReviewRemark   string      `json:"review_remark" gorm:"size:500"`

	Student   *Student `json:"student,omitempty" gorm:"foreignKey:StudentID"`
	Applicant *User    `json:"applicant,omitempty" gorm:"foreignKey:ApplicantID"`
	Reviewer  *User    `json:"reviewer,omitempty" gorm:"foreignKey:ReviewerID"`
}

func (LeaveRequest) TableName() string { return "leave_requests" }
