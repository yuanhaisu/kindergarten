package domain

import "time"

type NoticeCategory string

const (
	NoticeCategoryTask     NoticeCategory = "task"
	NoticeCategoryHomework NoticeCategory = "homework"
	NoticeCategoryBring    NoticeCategory = "bring"
	NoticeCategoryActivity NoticeCategory = "activity"
	NoticeCategoryHealth   NoticeCategory = "health"
	NoticeCategoryOther    NoticeCategory = "other"
)

type NoticeStatus int8

const (
	NoticeStatusDraft     NoticeStatus = 0
	NoticeStatusScheduled NoticeStatus = 1
	NoticeStatusPublished NoticeStatus = 2
	NoticeStatusRevoked   NoticeStatus = 3
)

// Notice 通知
type Notice struct {
	SoftDeleteModel
	KindergartenID uint64         `json:"kindergarten_id" gorm:"not null;index"`
	// ClassID 已弃用：单班归属改由 NoticeTarget 目标集合表达。本列冗余写入首个目标班
	// （回滚降级用：旧代码按 class_id 查仍能看到多班通知的首个班），新代码禁止读取；
	// 手工迁移删除前先跑 docs/migrations 对账 SQL。
	ClassID   uint64         `json:"class_id" gorm:"not null;index"`
	CreatorID uint64         `json:"creator_id" gorm:"not null;index"`
	Category  NoticeCategory `json:"category" gorm:"size:20;not null"`
	Title     string         `json:"title" gorm:"size:200;not null"`
	Content   string         `json:"content" gorm:"type:text"`
	NeedConfirm    bool       `json:"need_confirm" gorm:"default:false"`
	ScheduledAt    *time.Time `json:"scheduled_at,omitempty"`
	PublishedAt    *time.Time `json:"published_at,omitempty"`
	Status         NoticeStatus `json:"status" gorm:"default:0"`
	TemplateID     *uint64      `json:"template_id,omitempty"`

	Creator *User  `json:"creator,omitempty" gorm:"foreignKey:CreatorID"`
	Class   *Class `json:"class,omitempty" gorm:"foreignKey:ClassID"`
}

func (Notice) TableName() string { return "notices" }

// NoticeTarget 通知投递目标：一条通知对应 N 个班级（notice-multitarget）。
// KindergartenID 同列冗余存储，避免查询目标集时必须回连 notices 做租户过滤。
type NoticeTarget struct {
	BaseModel
	NoticeID       uint64 `json:"notice_id" gorm:"not null;uniqueIndex:idx_notice_target,priority:1"`
	ClassID        uint64 `json:"class_id" gorm:"not null;uniqueIndex:idx_notice_target,priority:2;index"`
	KindergartenID uint64 `json:"kindergarten_id" gorm:"not null;index"`
}

func (NoticeTarget) TableName() string { return "notice_targets" }

// NoticeAttachment 通知附件
type NoticeAttachment struct {
	BaseModel
	NoticeID uint64 `json:"notice_id" gorm:"not null;index"`
	FileURL  string `json:"file_url" gorm:"size:500;not null"`
	FileType string `json:"file_type" gorm:"size:20;not null;comment:image/video"`
	FileName string `json:"file_name" gorm:"size:200"`
	FileSize int64  `json:"file_size"`
	SortNum  int    `json:"sort_num" gorm:"default:0"`
}

func (NoticeAttachment) TableName() string { return "notice_attachments" }

// NoticeReadLog 通知已读记录
type NoticeReadLog struct {
	BaseModel
	NoticeID    uint64     `json:"notice_id" gorm:"not null;uniqueIndex:idx_notice_user"`
	UserID      uint64     `json:"user_id" gorm:"not null;uniqueIndex:idx_notice_user"`
	ReadAt      time.Time  `json:"read_at"`
	ConfirmedAt *time.Time `json:"confirmed_at,omitempty"`
}

func (NoticeReadLog) TableName() string { return "notice_read_logs" }

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

func (NoticeTemplate) TableName() string { return "notice_templates" }
