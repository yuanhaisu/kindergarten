package domain

import "time"

type TeacherTitle string

const (
	TeacherTitleHead      TeacherTitle = "head"
	TeacherTitleAssistant TeacherTitle = "assistant"
	TeacherTitleNursery   TeacherTitle = "nursery"
)

// ClassTeacher 班级-老师关联
type ClassTeacher struct {
	BaseModel
	ClassID uint64       `json:"class_id" gorm:"not null;uniqueIndex:idx_class_teacher"`
	UserID  uint64       `json:"user_id" gorm:"not null;uniqueIndex:idx_class_teacher"`
	Title   TeacherTitle `json:"title" gorm:"size:20;not null"`
	Status  int8         `json:"status" gorm:"default:1"`

	Class *Class `json:"class,omitempty" gorm:"foreignKey:ClassID"`
	User  *User  `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

func (ClassTeacher) TableName() string { return "class_teachers" }

// TeacherProfile 园所内老师档案（与 users + user_roles(role=teacher) 配套）
type TeacherProfile struct {
	BaseModel
	UserID         uint64    `json:"user_id" gorm:"not null;uniqueIndex:idx_teacher_profile_user_kg"`
	KindergartenID uint64    `json:"kindergarten_id" gorm:"not null;uniqueIndex:idx_teacher_profile_user_kg"`
	RealName       string    `json:"real_name" gorm:"size:64"`
	Education      string    `json:"education" gorm:"size:50"`
	Major          string    `json:"major" gorm:"size:100"`
	CertificateNo  string    `json:"certificate_no" gorm:"size:100"`
	JoinDate       time.Time `json:"join_date" gorm:"type:date"`
	Status         int8      `json:"status" gorm:"default:1;comment:1在职 2离职"`

	User *User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

func (TeacherProfile) TableName() string { return "teacher_profiles" }
