package model

import "time"

// NoticeCategory 通知分类
type NoticeCategory string

const (
	NoticeCategoryHomework NoticeCategory = "homework" // 作业
	NoticeCategoryBring    NoticeCategory = "bring"    // 带物品
	NoticeCategoryActivity NoticeCategory = "activity" // 活动
	NoticeCategoryHealth   NoticeCategory = "health"   // 健康
	NoticeCategoryOther    NoticeCategory = "other"    // 其他
)

// NoticeStatus 通知状态
type NoticeStatus int8

const (
	NoticeStatusDraft     NoticeStatus = 0 // 草稿
	NoticeStatusScheduled NoticeStatus = 1 // 定时待发
	NoticeStatusPublished NoticeStatus = 2 // 已发布
	NoticeStatusRevoked   NoticeStatus = 3 // 已撤回
)

// Notice 通知
type Notice struct {
	SoftDeleteModel
	KindergartenID uint64         `json:"kindergarten_id" gorm:"not null;index"`
	ClassID        uint64         `json:"class_id" gorm:"not null;index"`
	CreatorID      uint64         `json:"creator_id" gorm:"not null;index"` // 发布老师
	Category       NoticeCategory `json:"category" gorm:"size:20;not null"`
	Title          string         `json:"title" gorm:"size:200;not null"`
	Content        string         `json:"content" gorm:"type:text"`
	NeedConfirm    bool           `json:"need_confirm" gorm:"default:false"` // 是否需要家长确认
	ScheduledAt    *time.Time     `json:"scheduled_at,omitempty"`            // 定时发送时间
	PublishedAt    *time.Time     `json:"published_at,omitempty"`
	Status         NoticeStatus   `json:"status" gorm:"default:0"`
	TemplateID     *uint64        `json:"template_id,omitempty"` // 使用的模板

	Creator *User  `json:"creator,omitempty" gorm:"foreignKey:CreatorID"`
	Class   *Class `json:"class,omitempty" gorm:"foreignKey:ClassID"`
}

// NoticeAttachment 通知附件（图片/视频）
type NoticeAttachment struct {
	BaseModel
	NoticeID uint64 `json:"notice_id" gorm:"not null;index"`
	FileURL  string `json:"file_url" gorm:"size:500;not null"`
	FileType string `json:"file_type" gorm:"size:20;not null;comment:image/video"`
	FileName string `json:"file_name" gorm:"size:200"`
	FileSize int64  `json:"file_size"`
	SortNum  int    `json:"sort_num" gorm:"default:0"`
}

// NoticeReadLog 通知已读记录
type NoticeReadLog struct {
	BaseModel
	NoticeID    uint64     `json:"notice_id" gorm:"not null;uniqueIndex:idx_notice_user"`
	UserID      uint64     `json:"user_id" gorm:"not null;uniqueIndex:idx_notice_user"`
	ReadAt      time.Time  `json:"read_at"`
	ConfirmedAt *time.Time `json:"confirmed_at,omitempty"` // 确认收到时间
}

// NoticeTemplate 通知模板
type NoticeTemplate struct {
	SoftDeleteModel
	KindergartenID uint64         `json:"kindergarten_id" gorm:"not null;index"`
	CreatorID      uint64         `json:"creator_id" gorm:"not null"`
	Category       NoticeCategory `json:"category" gorm:"size:20"`
	Title          string         `json:"title" gorm:"size:200;not null"`
	Content        string         `json:"content" gorm:"type:text"`
	UsageCount     int            `json:"usage_count" gorm:"default:0"`
}
