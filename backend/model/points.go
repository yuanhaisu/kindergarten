package model

import "time"

// PointAccount 积分账户
type PointAccount struct {
	BaseModel
	StudentID   uint64 `json:"student_id" gorm:"not null;uniqueIndex"`
	TotalPoints int    `json:"total_points" gorm:"default:0"`
	UsedPoints  int    `json:"used_points" gorm:"default:0"`

	Student *Student `json:"student,omitempty" gorm:"foreignKey:StudentID"`
}

// PointTransaction 积分流水
type PointTransaction struct {
	BaseModel
	StudentID   uint64 `json:"student_id" gorm:"not null;index"`
	Points      int    `json:"points" gorm:"not null"`
	Balance     int    `json:"balance"`
	Source      string `json:"source" gorm:"size:50;not null"`
	ReferenceID uint64 `json:"reference_id"`
	Description string `json:"description" gorm:"size:200"`
}

// Achievement 成就徽章定义
type Achievement struct {
	SoftDeleteModel
	Name        string `json:"name" gorm:"size:100;not null"`
	Description string `json:"description" gorm:"size:500"`
	Icon        string `json:"icon" gorm:"size:500"`
	Condition   string `json:"condition" gorm:"type:text"`
	Points      int    `json:"points" gorm:"default:0"`
	SortNum     int    `json:"sort_num" gorm:"default:0"`
}

// StudentAchievement 学生已获成就
type StudentAchievement struct {
	BaseModel
	StudentID     uint64    `json:"student_id" gorm:"not null;uniqueIndex:idx_student_achievement"`
	AchievementID uint64    `json:"achievement_id" gorm:"not null;uniqueIndex:idx_student_achievement"`
	AchievedAt    time.Time `json:"achieved_at"`

	Student     *Student     `json:"student,omitempty" gorm:"foreignKey:StudentID"`
	Achievement *Achievement `json:"achievement,omitempty" gorm:"foreignKey:AchievementID"`
}
