package domain

// AcademicYear 园所学年配置（管理端设置模块）。
type AcademicYear struct {
	BaseModel
	KindergartenID uint64 `gorm:"not null;uniqueIndex:idx_academic_year_kg_year,priority:1" json:"-"`
	Year string `gorm:"size:32;not null;uniqueIndex:idx_academic_year_kg_year,priority:2" json:"-"`
	SemesterSpringStart string `gorm:"size:16;not null;default:''" json:"-"` // YYYY-MM-DD
	SemesterSpringEnd   string `gorm:"size:16;not null;default:''" json:"-"`
	SemesterFallStart   string `gorm:"size:16;not null;default:''" json:"-"`
	SemesterFallEnd     string `gorm:"size:16;not null;default:''" json:"-"`
	IsCurrent           bool   `gorm:"not null;default:0" json:"-"`
}

func (AcademicYear) TableName() string { return "academic_years" }
