package domain

type User struct {
	BaseModel
	OpenID       string `gorm:"index;size:128" json:"-"`
	UnionID      string `gorm:"index;size:128" json:"-"`
	Nickname     string `gorm:"size:64" json:"nickname"`
	Avatar       string `gorm:"size:512" json:"avatar"`
	Phone        string `gorm:"size:20;index" json:"phone"`
	Gender       int    `gorm:"default:0" json:"gender"`
	PasswordHash string `gorm:"size:128" json:"-"`
	IDCardNumber string `gorm:"size:32" json:"id_card_number,omitempty"`
}

func (User) TableName() string { return "users" }

// UserRole 用户角色关联（一个用户可有多个角色，如同时是老师和家长）
type UserRole struct {
	BaseModel
	UserID           uint64  `gorm:"index;not null" json:"user_id"`
	Role             string  `gorm:"size:32;not null;index" json:"role"`
	KindergartenID   *uint64 `gorm:"index" json:"kindergarten_id,omitempty"`
	KindergartenName string  `gorm:"size:128" json:"kindergarten_name"`
	ClassID          *uint64 `gorm:"index" json:"class_id,omitempty"`
	ClassName        string  `gorm:"size:64" json:"class_name,omitempty"`
	Title            string  `gorm:"size:32" json:"title,omitempty"`
	TitleLabel       string  `gorm:"size:32" json:"title_label,omitempty"`
	AdminRole        string  `gorm:"size:64" json:"admin_role,omitempty"`

	User         *User         `json:"-" gorm:"foreignKey:UserID"`
	Kindergarten *Kindergarten `json:"-" gorm:"foreignKey:KindergartenID"`
	Class        *Class        `json:"-" gorm:"foreignKey:ClassID"`
}

func (UserRole) TableName() string { return "user_roles" }
