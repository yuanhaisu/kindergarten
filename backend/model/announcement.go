package model

import "time"

// Announcement 园所公告（管理后台发布，全园推送）
type Announcement struct {
	SoftDeleteModel
	KindergartenID uint64     `json:"kindergarten_id" gorm:"not null;index"`
	CreatorID      uint64     `json:"creator_id" gorm:"not null"`
	Title          string     `json:"title" gorm:"size:200;not null"`
	Content        string     `json:"content" gorm:"type:text"`
	IsTop          bool       `json:"is_top" gorm:"default:false"`
	PublishedAt    *time.Time `json:"published_at,omitempty"`
	Status         int8       `json:"status" gorm:"default:0;comment:0草稿 1已发布 2已撤回"`

	Creator *User `json:"creator,omitempty" gorm:"foreignKey:CreatorID"`
}
