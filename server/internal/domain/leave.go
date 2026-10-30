package domain

import "time"

type LeaveStatus int8

const (
	LeaveStatusPending  LeaveStatus = 0
	LeaveStatusApproved LeaveStatus = 1
	LeaveStatusRejected LeaveStatus = 2
	LeaveStatusCanceled LeaveStatus = 3
)

// 请假类型代码，与契约 api-parent/03-leave.md 一致。
const (
	LeaveTypeSick     = "sick"
	LeaveTypePersonal = "personal"
	LeaveTypeOther    = "other"
)

// LeaveRequest 请假申请
type LeaveRequest struct {
	SoftDeleteModel
	KindergartenID uint64 `json:"kindergarten_id" gorm:"not null;index"`
	ClassID        uint64 `json:"class_id" gorm:"not null;index"`
	StudentID      uint64 `json:"student_id" gorm:"not null;index"`
	ApplicantID    uint64 `json:"applicant_id" gorm:"not null"`
	LeaveType      string `json:"leave_type" gorm:"size:16;default:other"`
	Reason         string `json:"reason" gorm:"size:500;not null"`
	// EmergencyContact 存接送人的 JSON 文本（name/phone）。契约对外是对象，
	// 但设计把它定为单值列，因此序列化后落在这一列，不在表结构里拆字段。
	EmergencyContact string      `json:"emergency_contact" gorm:"size:100;default:''"`
	StartDate        time.Time   `json:"start_date" gorm:"not null;index"`
	EndDate          time.Time   `json:"end_date" gorm:"not null"`
	Days             int         `json:"days"`
	Status           LeaveStatus `json:"status" gorm:"default:0"`
	ReviewerID       *uint64     `json:"reviewer_id,omitempty"`
	ReviewedAt       *time.Time  `json:"reviewed_at,omitempty"`
	ReviewRemark     string      `json:"review_remark" gorm:"size:500"`

	Student   *Student `json:"student,omitempty" gorm:"foreignKey:StudentID"`
	Applicant *User    `json:"applicant,omitempty" gorm:"foreignKey:ApplicantID"`
	Reviewer  *User    `json:"reviewer,omitempty" gorm:"foreignKey:ReviewerID"`
}

func (LeaveRequest) TableName() string { return "leave_requests" }

// LeaveAttachment 请假附件（凭证图片）。
// 结构对齐 notice_attachments：详情按 sort_num, id 升序返回。
type LeaveAttachment struct {
	BaseModel
	LeaveID  uint64 `json:"leave_id" gorm:"not null;index"`
	FileURL  string `json:"file_url" gorm:"size:500;not null"`
	FileType string `json:"file_type" gorm:"size:20;not null;comment:image"`
	FileName string `json:"file_name" gorm:"size:200"`
	FileSize int64  `json:"file_size"`
	SortNum  int    `json:"sort_num" gorm:"default:0"`
}

func (LeaveAttachment) TableName() string { return "leave_attachments" }
