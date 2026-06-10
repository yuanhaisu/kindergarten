package domain

import "time"

// 公告状态（与 api-admin/14-announcements.md 对齐）：0 草稿 / 1 定时 / 2 已发布 / 3 已撤回
type AnnouncementStatus int8

const (
	AnnouncementStatusDraft     AnnouncementStatus = 0
	AnnouncementStatusScheduled AnnouncementStatus = 1
	AnnouncementStatusPublished AnnouncementStatus = 2
	AnnouncementStatusRevoked   AnnouncementStatus = 3
)

// Announcement 园所公告（管理后台发布，全园推送）
type Announcement struct {
	SoftDeleteModel
	KindergartenID uint64             `json:"kindergarten_id" gorm:"not null;index"`
	CreatorID      uint64             `json:"creator_id" gorm:"not null;index"`
	Title          string             `json:"title" gorm:"size:200;not null"`
	Content        string             `json:"content" gorm:"type:text"`
	CoverURL       string             `json:"cover_url" gorm:"size:512"`
	IsImportant    bool               `json:"is_important" gorm:"default:false"`
	IsTop          bool               `json:"is_top" gorm:"default:false;index"`
	TargetRoles    string             `json:"target_roles" gorm:"type:text;comment:JSON 字符串数组"`
	ViewCount      int64              `json:"view_count" gorm:"default:0"`
	ScheduledAt    *time.Time         `json:"scheduled_at,omitempty"`
	PublishedAt    *time.Time         `json:"published_at,omitempty"`
	Status         AnnouncementStatus `json:"status" gorm:"default:0"`

	Creator *User `json:"creator,omitempty" gorm:"foreignKey:CreatorID"`
}

func (Announcement) TableName() string { return "announcements" }

// AnnouncementAttachment 公告附件
type AnnouncementAttachment struct {
	BaseModel
	AnnouncementID uint64 `json:"announcement_id" gorm:"not null;index"`
	FileURL          string `json:"file_url" gorm:"size:500;not null"`
	FileName         string `json:"file_name" gorm:"size:200"`
	FileSize         int64  `json:"file_size" gorm:"default:0"`
	SortNum          int    `json:"sort_num" gorm:"default:0"`
}

func (AnnouncementAttachment) TableName() string { return "announcement_attachments" }
