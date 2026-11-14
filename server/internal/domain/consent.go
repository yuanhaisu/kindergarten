package domain

import "time"

// UserConsent 记录一个用户对某个隐私政策版本的同意。
//
// 只追加、不软删：同一用户同一版本只有一条（唯一键），政策升级后新增一行。
// 不使用 BaseModel，避免 DeletedAt 占住唯一索引后无法重新写入。
type UserConsent struct {
	ID            uint64    `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	UserID        uint64    `gorm:"not null;uniqueIndex:uk_user_consent_version" json:"user_id"`
	PolicyVersion string    `gorm:"size:32;not null;uniqueIndex:uk_user_consent_version" json:"policy_version"`
	Source        string    `gorm:"size:16;not null" json:"source"`
	IP            string    `gorm:"size:64;not null;default:''" json:"ip"`
	UserAgent     string    `gorm:"size:512;not null;default:''" json:"user_agent"`
	ConsentedAt   time.Time `gorm:"not null" json:"consented_at"`
}

func (UserConsent) TableName() string { return "user_consents" }

const (
	ConsentSourceParent  = "parent"
	ConsentSourceTeacher = "teacher"
	ConsentSourceAdmin   = "admin"
)
