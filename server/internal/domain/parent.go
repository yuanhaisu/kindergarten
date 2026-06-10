package domain

import "time"

// ParentProfile 园所内家长档案（与 users + user_roles(role=parent) 配套）
type ParentProfile struct {
	BaseModel
	UserID         uint64     `json:"user_id" gorm:"not null;uniqueIndex:idx_parent_profile_user_kg"`
	KindergartenID uint64     `json:"kindergarten_id" gorm:"not null;uniqueIndex:idx_parent_profile_user_kg"`
	RealName       string     `json:"real_name" gorm:"size:64"`
	Status         int8       `json:"status" gorm:"default:1;comment:1正常 2禁用"`
	LastLoginAt    *time.Time `json:"last_login_at,omitempty"`

	User *User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

func (ParentProfile) TableName() string { return "parent_profiles" }
