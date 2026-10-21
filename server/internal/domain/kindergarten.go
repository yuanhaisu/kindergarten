package domain

type Kindergarten struct {
	BaseModel
	Name    string `gorm:"size:128;not null" json:"name"`
	Address string `gorm:"size:256" json:"address"`

	Phone         string `gorm:"size:32;not null;default:''" json:"-"`
	Principal     string `gorm:"size:64;not null;default:''" json:"-"`
	LogoURL       string `gorm:"size:512;not null;default:''" json:"-"`
	LicenseNo     string `gorm:"size:128;not null;default:''" json:"-"`
	Capacity      int    `gorm:"not null;default:0" json:"-"`
	EstablishedAt string `gorm:"size:16;not null;default:''" json:"-"` // YYYY-MM-DD
	Description   string `gorm:"type:text" json:"-"`
}

func (Kindergarten) TableName() string { return "kindergartens" }

type Class struct {
	BaseModel
	KindergartenID uint64 `gorm:"index;not null" json:"kindergarten_id"`
	Name             string `gorm:"size:64;not null" json:"name"`
	Grade            string `gorm:"size:16;not null;default:''" json:"grade"`
	AcademicYear     string `gorm:"size:32;not null;default:''" json:"academic_year"`
	Capacity         int    `gorm:"not null;default:30" json:"capacity"`
	Status           int8   `gorm:"not null;default:1" json:"status"`
	// PickupDeadline 班级放学时刻，格式 "HH:mm"。空串表示未配置，判定时回退缺省 16:00
	// （产品决策 2026-10-07：班级可配、默认 16:00；design-data 无出处，见 proposal 阻塞清单）。
	// 用于「提前接走须填理由」与「超时未接」两处判定。
	// 用 string 存时刻而非 time.Time：时刻不含日期分量，且与 EstablishedAt 的既有惯例一致。
	PickupDeadline string `gorm:"size:5;not null;default:''" json:"pickup_deadline"`

	Kindergarten *Kindergarten `json:"kindergarten,omitempty" gorm:"foreignKey:KindergartenID"`
}

func (Class) TableName() string { return "classes" }
