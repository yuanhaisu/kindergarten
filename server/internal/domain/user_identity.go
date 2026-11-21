package domain

import "time"

// UserIdentity 一个账号在某个客户端（微信小程序 AppID）上的身份（#164）。
//
// 身份的唯一键是 (app_id, open_id)：openid 是「用户 × 小程序」二元的，同一自然人
// 在家长端与老师端是两条身份，但都指向同一条 User。一个账号在同一 app 只能有
// 一条身份（(user_id, app_id) 唯一）。
//
// 刻意不嵌 BaseModel：解绑/账号合并时必须硬删。软删行会继续占用
// (app_id, open_id) 与 (user_id, app_id) 两个唯一索引，导致身份永远无法重新绑定。
// 需要保留历史时另建审计表，不要给本表加 deleted_at。
type UserIdentity struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	UserID    uint64    `gorm:"not null;uniqueIndex:uk_user_identities_user_app,priority:1" json:"user_id"`
	AppID     string    `gorm:"size:128;not null;uniqueIndex:uk_user_identities_app_open,priority:1;uniqueIndex:uk_user_identities_user_app,priority:2" json:"app_id"`
	OpenID    string    `gorm:"size:128;not null;uniqueIndex:uk_user_identities_app_open,priority:2" json:"open_id"`
	UnionID   string    `gorm:"size:128;not null;default:''" json:"union_id,omitempty"`
}

func (UserIdentity) TableName() string { return "user_identities" }
