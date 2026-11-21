package domain

// User 平台账号。v1 起「一人一账号」：同一自然人在各客户端只有这一条 User，
// 客户端身份（app_id + open_id）存在 user_identities 表（#164）。
//
// 手机号唯一性由数据库保证：users.phone_active 是 STORED 生成列
// IF(deleted_at IS NULL AND phone <> "", phone, NULL)，其上建唯一索引。
// 空手机号与软删账号的 phone_active 为 NULL，不参与唯一比较——因此它们可以
// 有多条，而一个活跃非空手机号只能对应一条 User。这里刻意不声明 PhoneActive
// 字段：生成列不可写，且未迁移的库上 SELECT 它会直接报错。
//
// OpenID / UnionID 是共用 AppID 时代的遗留列，**不再是身份来源**，仅为 cmd 与
// 旧视图兼容保留，由 #173 删除。新代码一律走 user_identities / FindOrCreateByPhone。
type User struct {
	BaseModel
	// Deprecated: 使用 user_identities(app_id, open_id) 定位客户端身份；#173 删除本列。
	OpenID string `gorm:"index;size:128" json:"-"`
	// Deprecated: 仅供旧跨端关联判定（cross_end_identity）；#173 删除本列。
	UnionID      string `gorm:"index;size:128" json:"-"`
	Nickname     string `gorm:"size:64" json:"nickname"`
	Avatar       string `gorm:"size:512" json:"avatar"`
	Phone        string `gorm:"size:20;index" json:"phone"`
	Gender       int    `gorm:"default:0" json:"gender"`
	PasswordHash string `gorm:"size:128" json:"-"`
	IDCardNumber string `gorm:"size:32" json:"id_card_number,omitempty"`
	// MustChangePassword 平台方开通园所时创建的园长账号为 true，改密后清零。
	MustChangePassword bool `gorm:"not null;default:false" json:"must_change_password"`
}

func (User) TableName() string { return "users" }

// UserRole 用户角色关联。
//
// 一条 User 可挂多个角色，且可跨园所、跨端：老师兼家长不再是两条账号，
// 而是同一条 User 同时持有 teacher 与 parent 角色（客户端身份见 UserIdentity）。
// 角色本身不承载登录身份——登录认的是 (app_id, open_id)，角色只决定授权范围。
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
