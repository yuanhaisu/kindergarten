package model

// Kindergarten 幼儿园（顶层租户）
type Kindergarten struct {
	SoftDeleteModel
	Name      string `json:"name" gorm:"size:100;not null"`
	Logo      string `json:"logo" gorm:"size:500"`
	Address   string `json:"address" gorm:"size:300"`
	Phone     string `json:"phone" gorm:"size:20"`
	Principal string `json:"principal" gorm:"size:50"`   // 园长姓名
	LicenseNo string `json:"license_no" gorm:"size:100"` // 办学许可证号
	Province  string `json:"province" gorm:"size:50"`
	City      string `json:"city" gorm:"size:50"`
	District  string `json:"district" gorm:"size:50"`
	Status    int8   `json:"status" gorm:"default:1;comment:1启用 0禁用"`
}

// Class 班级
type Class struct {
	SoftDeleteModel
	KindergartenID uint64 `json:"kindergarten_id" gorm:"not null;index"`
	Name           string `json:"name" gorm:"size:50;not null"` // 如"小一班"
	Grade          string `json:"grade" gorm:"size:20"`         // 小班/中班/大班
	AcademicYear   string `json:"academic_year" gorm:"size:20"` // 学年 如"2025-2026"
	Capacity       int    `json:"capacity"`                     // 班级容量
	Status         int8   `json:"status" gorm:"default:1;comment:1正常 0已毕业 -1已解散"`

	Kindergarten *Kindergarten `json:"kindergarten,omitempty" gorm:"foreignKey:KindergartenID"`
}
