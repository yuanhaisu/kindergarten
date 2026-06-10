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

	Kindergarten *Kindergarten `json:"kindergarten,omitempty" gorm:"foreignKey:KindergartenID"`
}

func (Class) TableName() string { return "classes" }
