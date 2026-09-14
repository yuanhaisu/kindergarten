package domain

import "time"

type TaskFrequency string

const (
	TaskFrequencyDaily  TaskFrequency = "daily"
	TaskFrequencyWeekly TaskFrequency = "weekly"
	TaskFrequencyOnce   TaskFrequency = "once"
)

// ParentTask 亲子任务（老师布置）
type ParentTask struct {
	SoftDeleteModel
	KindergartenID uint64        `json:"kindergarten_id" gorm:"not null;index"`
	ClassID        uint64        `json:"class_id" gorm:"not null;index"`
	CreatorID      uint64        `json:"creator_id" gorm:"not null"`
	Title          string        `json:"title" gorm:"size:200;not null"`
	Description    string        `json:"description" gorm:"type:text"`
	Intro          string        `json:"intro" gorm:"type:text;comment:任务详细介绍"`
	Requirements   string        `json:"requirements" gorm:"type:text;comment:JSON数组-任务要求"`
	Rewards        string        `json:"rewards" gorm:"type:text;comment:JSON数组-奖励"`
	Frequency      TaskFrequency `json:"frequency" gorm:"size:20;not null"`
	StartDate      time.Time     `json:"start_date" gorm:"type:date;not null"`
	EndDate        time.Time     `json:"end_date" gorm:"type:date;not null"`
	Points         int           `json:"points" gorm:"default:1"`
	Status         int8          `json:"status" gorm:"default:1;comment:1进行中 0已结束"`

	Creator *User  `json:"creator,omitempty" gorm:"foreignKey:CreatorID"`
	Class   *Class `json:"class,omitempty" gorm:"foreignKey:ClassID"`
}

func (ParentTask) TableName() string { return "parent_tasks" }

// TaskCheckIn 任务打卡记录（家长提交）
//
// 唯一约束是 (task_id, student_id, date) 三元组——每个孩子每天每项任务打卡一次。
// 三个字段必须都带 uniqueIndex 且同名，GORM 才会生成复合索引；只标 Date 会退化成
// 「全平台每天只允许一条打卡」。参照 Attendance 的 idx_student_date。
//
// ⚠️ AutoMigrate 只增不改：已存在的错误索引 idx_task_student_date(date) 不会被
// 自动修正，需先手工 DROP（见 docs/migrations/）。
type TaskCheckIn struct {
	BaseModel
	TaskID    uint64    `json:"task_id" gorm:"not null;index;uniqueIndex:idx_task_student_date"`
	StudentID uint64    `json:"student_id" gorm:"not null;index;uniqueIndex:idx_task_student_date"`
	UserID    uint64    `json:"user_id" gorm:"not null"`
	Date      time.Time `json:"date" gorm:"type:date;not null;uniqueIndex:idx_task_student_date"`
	Content   string    `json:"content" gorm:"size:500"`
	ImageURL  string    `json:"image_url" gorm:"size:500"`
	Points    int       `json:"points" gorm:"default:0"`

	Task    *ParentTask `json:"task,omitempty" gorm:"foreignKey:TaskID"`
	Student *Student    `json:"student,omitempty" gorm:"foreignKey:StudentID"`
}

func (TaskCheckIn) TableName() string { return "task_check_ins" }
