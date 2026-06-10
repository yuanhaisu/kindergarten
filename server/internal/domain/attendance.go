package domain

import "time"

type AttendanceStatus int8

const (
	AttendancePresent AttendanceStatus = 1
	AttendanceAbsent  AttendanceStatus = 2
	AttendanceLeave   AttendanceStatus = 3
	AttendanceLate    AttendanceStatus = 4
)

// Attendance 每日出勤记录
type Attendance struct {
	BaseModel
	KindergartenID uint64           `json:"kindergarten_id" gorm:"not null;index"`
	ClassID        uint64           `json:"class_id" gorm:"not null;index"`
	StudentID      uint64           `json:"student_id" gorm:"not null;uniqueIndex:idx_student_date"`
	Date           time.Time        `json:"date" gorm:"type:date;not null;uniqueIndex:idx_student_date"`
	Status         AttendanceStatus `json:"status" gorm:"not null"`
	CheckInTime    *time.Time       `json:"check_in_time,omitempty"`
	Remark         string           `json:"remark" gorm:"size:200"`
	RecorderID     uint64           `json:"recorder_id"`

	Student *Student `json:"student,omitempty" gorm:"foreignKey:StudentID"`
}

func (Attendance) TableName() string { return "attendances" }
