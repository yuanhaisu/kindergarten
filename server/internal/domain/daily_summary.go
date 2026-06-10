package domain

import "time"

// StudentDailySummary 学生每日摘要（按学生+日期冗余快照，支撑首页高频查询）
type StudentDailySummary struct {
	BaseModel
	StudentID      uint64     `json:"student_id" gorm:"not null;uniqueIndex:idx_student_date"`
	Date           time.Time  `json:"date" gorm:"type:date;not null;uniqueIndex:idx_student_date"`
	AttendanceStatus int8     `json:"attendance_status" gorm:"comment:沿用 AttendanceStatus 枚举"`
	CheckInTime    *time.Time `json:"check_in_time,omitempty"`
	MealRating     int8       `json:"meal_rating" gorm:"comment:综合餐食评分 1poor 2normal 3good"`
	NapDuration    int        `json:"nap_duration" gorm:"comment:午睡时长(分钟)"`
	SleepQuality   int8       `json:"sleep_quality" gorm:"comment:1poor 2normal 3good"`
	MoodRating     int8       `json:"mood_rating" gorm:"comment:综合情绪 1sad 2normal 3happy"`
	PerformanceTag string     `json:"performance_tag" gorm:"size:200"`
}

func (StudentDailySummary) TableName() string { return "student_daily_summaries" }
