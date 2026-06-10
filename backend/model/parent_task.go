package model

import "time"

// TaskFrequency 亲子任务频率
type TaskFrequency string

const (
	TaskFrequencyDaily  TaskFrequency = "daily"  // 每日
	TaskFrequencyWeekly TaskFrequency = "weekly" // 每周
	TaskFrequencyOnce   TaskFrequency = "once"   // 一次性
)

// ParentTask 亲子任务（老师布置）
type ParentTask struct {
	SoftDeleteModel
	KindergartenID uint64        `json:"kindergarten_id" gorm:"not null;index"`
	ClassID        uint64        `json:"class_id" gorm:"not null;index"`
	CreatorID      uint64        `json:"creator_id" gorm:"not null"`
	Title          string        `json:"title" gorm:"size:200;not null"`
	Description    string        `json:"description" gorm:"type:text"`
	Frequency      TaskFrequency `json:"frequency" gorm:"size:20;not null"`
	StartDate      time.Time     `json:"start_date" gorm:"type:date;not null"`
	EndDate        time.Time     `json:"end_date" gorm:"type:date;not null"`
	Points         int           `json:"points" gorm:"default:1"`
	Status         int8          `json:"status" gorm:"default:1;comment:1进行中 0已结束"`

	Creator *User  `json:"creator,omitempty" gorm:"foreignKey:CreatorID"`
	Class   *Class `json:"class,omitempty" gorm:"foreignKey:ClassID"`
}

// TaskCheckIn 任务打卡记录（家长提交）
type TaskCheckIn struct {
	BaseModel
	TaskID    uint64    `json:"task_id" gorm:"not null;index"`
	StudentID uint64    `json:"student_id" gorm:"not null;index"`
	UserID    uint64    `json:"user_id" gorm:"not null"`
	Date      time.Time `json:"date" gorm:"type:date;not null;uniqueIndex:idx_task_student_date"`
	Content   string    `json:"content" gorm:"size:500"`
	ImageURL  string    `json:"image_url" gorm:"size:500"`
	Points    int       `json:"points" gorm:"default:0"`

	Task    *ParentTask `json:"task,omitempty" gorm:"foreignKey:TaskID"`
	Student *Student    `json:"student,omitempty" gorm:"foreignKey:StudentID"`
}
