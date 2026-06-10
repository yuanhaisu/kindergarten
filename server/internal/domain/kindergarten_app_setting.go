package domain

// KindergartenAppSetting 园所级应用配置（出勤规则、通知、成长、接送、积分等非「园所档案」字段）。
// ExtraConfig 存 JSON，与 admin SystemSettings 中除 basic 外的块对应。
type KindergartenAppSetting struct {
	BaseModel
	KindergartenID uint64 `gorm:"not null;uniqueIndex:idx_kg_app_setting_kg" json:"-"`
	CurrentSemester   string `gorm:"size:16;not null;default:'spring'" json:"-"` // spring | fall
	ExtraConfig       string `gorm:"type:json;not null" json:"-"`
}

func (KindergartenAppSetting) TableName() string { return "kindergarten_app_settings" }
