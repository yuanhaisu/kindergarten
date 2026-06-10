package model

import "time"

// Student 学生
type Student struct {
	SoftDeleteModel
	KindergartenID uint64    `json:"kindergarten_id" gorm:"not null;index"`
	ClassID        uint64    `json:"class_id" gorm:"not null;index"`
	Name           string    `json:"name" gorm:"size:50;not null"`
	Gender         int8      `json:"gender" gorm:"default:0;comment:0未知 1男 2女"`
	Birthday       time.Time `json:"birthday"`
	Avatar         string    `json:"avatar" gorm:"size:500"`
	StudentNo      string    `json:"student_no" gorm:"size:50;index"` // 学号
	HealthNote     string    `json:"health_note" gorm:"size:500"`     // 健康备注（过敏等）
	EnrollDate     time.Time `json:"enroll_date"`                     // 入园日期
	Status         int8      `json:"status" gorm:"default:1;comment:1在园 0离园"`

	Kindergarten *Kindergarten `json:"kindergarten,omitempty" gorm:"foreignKey:KindergartenID"`
	Class        *Class        `json:"class,omitempty" gorm:"foreignKey:ClassID"`
}

// ParentRelation 家长与幼儿关系
type ParentRelation string

const (
	RelationFather      ParentRelation = "father"
	RelationMother      ParentRelation = "mother"
	RelationGrandfather ParentRelation = "grandfather"
	RelationGrandmother ParentRelation = "grandmother"
	RelationOther       ParentRelation = "other"
)

// StudentParent 学生-家长关联
type StudentParent struct {
	BaseModel
	StudentID uint64         `json:"student_id" gorm:"not null;uniqueIndex:idx_student_parent"`
	UserID    uint64         `json:"user_id" gorm:"not null;uniqueIndex:idx_student_parent"`
	Relation  ParentRelation `json:"relation" gorm:"size:20;not null"`
	IsPrimary bool           `json:"is_primary" gorm:"default:false"` // 是否主要联系人

	Student *Student `json:"student,omitempty" gorm:"foreignKey:StudentID"`
	User    *User    `json:"user,omitempty" gorm:"foreignKey:UserID"`
}
