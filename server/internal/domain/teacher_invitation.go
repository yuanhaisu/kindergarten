package domain

import "time"

// TeacherInvitation 老师邀请码，用于微信登录时将 OpenID 绑定到预创建的教师账号。
type TeacherInvitation struct {
	BaseModel
	Token          string    `gorm:"uniqueIndex;size:64;not null" json:"token"`
	UserID         uint64    `gorm:"index;not null" json:"user_id"`
	KindergartenID uint64    `gorm:"index;not null" json:"kindergarten_id"`
	Status         int8      `gorm:"default:1;not null" json:"status"` // 1=待使用 2=已使用
	ExpiresAt      time.Time `json:"expires_at"`

	User *User `gorm:"foreignKey:UserID" json:"-"`
}

func (TeacherInvitation) TableName() string { return "teacher_invitations" }

const (
	TeacherInvitationPending = 1
	TeacherInvitationUsed    = 2
)
