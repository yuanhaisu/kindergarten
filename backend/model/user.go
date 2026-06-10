package model

// User 统一用户表（三端共用）
type User struct {
	SoftDeleteModel
	Phone    string `json:"phone" gorm:"size:20;uniqueIndex;not null"`
	Password string `json:"-" gorm:"size:200"`
	Nickname string `json:"nickname" gorm:"size:50"`
	Avatar   string `json:"avatar" gorm:"size:500"`
	Gender   int8   `json:"gender" gorm:"default:0;comment:0未知 1男 2女"`
	OpenID   string `json:"-" gorm:"size:100;uniqueIndex"` // 微信 OpenID
	UnionID  string `json:"-" gorm:"size:100;index"`       // 微信 UnionID
	Status   int8   `json:"status" gorm:"default:1;comment:1正常 0禁用"`
}

// RoleType 用户角色
type RoleType string

const (
	RoleAdmin   RoleType = "admin"   // 管理后台：园长/行政
	RoleTeacher RoleType = "teacher" // 老师端
	RoleParent  RoleType = "parent"  // 家长端
)

// UserRole 用户角色关联（一个用户可同时是老师和家长）
type UserRole struct {
	BaseModel
	UserID         uint64   `json:"user_id" gorm:"not null;index"`
	KindergartenID uint64   `json:"kindergarten_id" gorm:"not null;index"`
	Role           RoleType `json:"role" gorm:"size:20;not null"`
	Status         int8     `json:"status" gorm:"default:1;comment:1启用 0禁用"`

	User         *User         `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Kindergarten *Kindergarten `json:"kindergarten,omitempty" gorm:"foreignKey:KindergartenID"`
}

// TeacherTitle 班级老师职务
type TeacherTitle string

const (
	TeacherTitleHead      TeacherTitle = "head"      // 主班老师
	TeacherTitleAssistant TeacherTitle = "assistant" // 配班老师
	TeacherTitleNursery   TeacherTitle = "nursery"   // 保育员
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
