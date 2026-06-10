package domain

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
	StudentNo      string    `json:"student_no" gorm:"size:50;index"`
	IDCardNumber   string    `json:"id_card_number" gorm:"size:32;index"`
	Address        string    `json:"address" gorm:"size:256"`
	HealthNote     string    `json:"health_note" gorm:"size:500"`
	EnrollDate     time.Time `json:"enroll_date"`
	Status         int8      `json:"status" gorm:"default:1;comment:0待入学 1在读 2毕业 3退学"`

	Kindergarten *Kindergarten `json:"kindergarten,omitempty" gorm:"foreignKey:KindergartenID"`
	Class        *Class        `json:"class,omitempty" gorm:"foreignKey:ClassID"`
}

func (Student) TableName() string { return "students" }

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
	IsPrimary bool           `json:"is_primary" gorm:"default:false"`

	Student *Student `json:"student,omitempty" gorm:"foreignKey:StudentID"`
	User    *User    `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

func (StudentParent) TableName() string { return "student_parents" }
