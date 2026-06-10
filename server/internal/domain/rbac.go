package domain

// AdminRole 后台角色
type AdminRole struct {
	SoftDeleteModel
	KindergartenID uint64 `json:"kindergarten_id" gorm:"not null;index"`
	Name           string `json:"name" gorm:"size:50;not null"`
	Description    string `json:"description" gorm:"size:200"`
	IsSystem       bool   `json:"is_system" gorm:"default:false"`
}

func (AdminRole) TableName() string { return "admin_roles" }

// AdminPermission 权限定义
type AdminPermission struct {
	BaseModel
	Code        string `json:"code" gorm:"size:100;uniqueIndex;not null"`
	Name        string `json:"name" gorm:"size:100;not null"`
	Module      string `json:"module" gorm:"size:50"`
	Description string `json:"description" gorm:"size:200"`
}

func (AdminPermission) TableName() string { return "admin_permissions" }

// AdminRolePermission 角色-权限关联
type AdminRolePermission struct {
	BaseModel
	RoleID       uint64 `json:"role_id" gorm:"not null;uniqueIndex:idx_role_perm"`
	PermissionID uint64 `json:"permission_id" gorm:"not null;uniqueIndex:idx_role_perm"`
}

func (AdminRolePermission) TableName() string { return "admin_role_permissions" }

// UserAdminRole 用户-后台角色关联
type UserAdminRole struct {
	BaseModel
	UserID         uint64 `json:"user_id" gorm:"not null;uniqueIndex:idx_user_role_kg"`
	AdminRoleID    uint64 `json:"admin_role_id" gorm:"not null;uniqueIndex:idx_user_role_kg"`
	KindergartenID uint64 `json:"kindergarten_id" gorm:"not null;uniqueIndex:idx_user_role_kg"`
}

func (UserAdminRole) TableName() string { return "user_admin_roles" }
