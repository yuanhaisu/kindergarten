package domain

// KindergartenAppSetting 园所级应用配置（出勤规则、通知、成长、接送、积分等非「园所档案」字段）。
// ExtraConfig 存 JSON，与 admin SystemSettings 中除 basic 外的块对应。
type KindergartenAppSetting struct {
	BaseModel
	KindergartenID uint64 `gorm:"not null;uniqueIndex:idx_kg_app_setting_kg" json:"-"`
	CurrentSemester   string `gorm:"size:16;not null;default:'spring'" json:"-"` // spring | fall
	// ClassAlbumClasswideEnabled 园所级「全班视图」开关，默认关闭。
	// 独立成列而不是塞进 ExtraConfig：家长相册服务每次读列表都要判断它，
	// 走列可以直接查，不必复制一份 settingsPayload 的 JSON 解析（design D1）。
	ClassAlbumClasswideEnabled bool   `gorm:"not null;default:false" json:"-"`
	ExtraConfig                string `gorm:"type:json;not null" json:"-"`
}

func (KindergartenAppSetting) TableName() string { return "kindergarten_app_settings" }
