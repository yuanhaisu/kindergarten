package domain

// ParentSetting 家长端个人设置：6 个通知开关 + 2 个隐私开关。
//
// 作用域是 (user_id, kindergarten_id)：同一家长在不同园所可以有不同口径，
// 家长之间也互不可见。与 KindergartenAppSetting（一园一条的园所级配置）不是一回事。
//
// 列不写 DB default：GORM 会把「带 default 标签的零值字段」从 INSERT 中省略，
// 显式保存的 false 因此会被库默认值翻回 true。默认值只在 service 层定义。
// 本表不提供删除接口；测试清理用 Unscoped 硬删，避免软删行占住唯一索引。
type ParentSetting struct {
	BaseModel
	UserID             uint64 `gorm:"not null;uniqueIndex:uk_parent_setting_user_kg" json:"user_id"`
	KindergartenID     uint64 `gorm:"not null;uniqueIndex:uk_parent_setting_user_kg" json:"kindergarten_id"`
	NoticePush         bool   `gorm:"not null" json:"notice_push"`
	LeavePush          bool   `gorm:"not null" json:"leave_push"`
	PickupPush         bool   `gorm:"not null" json:"pickup_push"`
	GrowthPush         bool   `gorm:"not null" json:"growth_push"`
	TaskPush           bool   `gorm:"not null" json:"task_push"`
	AnnouncementPush   bool   `gorm:"not null" json:"announcement_push"`
	ShowPhoneToTeacher bool   `gorm:"not null" json:"show_phone_to_teacher"`
	AllowGrowthShare   bool   `gorm:"not null" json:"allow_growth_share"`
}

func (ParentSetting) TableName() string { return "parent_settings" }
