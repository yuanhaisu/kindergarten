package model

import "time"

// PickupType 接送类型
type PickupType int8

const (
	PickupTypeDropOff PickupType = 1 // 送入园
	PickupTypePickUp  PickupType = 2 // 接走
)

// PickupMethod 登记方式
type PickupMethod int8

const (
	PickupMethodScan   PickupMethod = 1 // 扫码
	PickupMethodManual PickupMethod = 2 // 手动登记
)

// PickupRecord 接送记录
type PickupRecord struct {
	BaseModel
	KindergartenID uint64       `json:"kindergarten_id" gorm:"not null;index"`
	ClassID        uint64       `json:"class_id" gorm:"not null;index"`
	StudentID      uint64       `json:"student_id" gorm:"not null;index"`
	Date           time.Time    `json:"date" gorm:"type:date;not null;index"`
	Type           PickupType   `json:"type" gorm:"not null;comment:1送入 2接走"`
	Method         PickupMethod `json:"method" gorm:"not null;comment:1扫码 2手动"`
	PickupPersonID *uint64      `json:"pickup_person_id,omitempty"`     // 接送人（已注册用户）
	PickupName     string       `json:"pickup_name" gorm:"size:50"`     // 接送人姓名（兜底）
	PickupRelation string       `json:"pickup_relation" gorm:"size:20"` // 与孩子关系
	OperatorID     uint64       `json:"operator_id"`                    // 操作老师
	RecordedAt     time.Time    `json:"recorded_at"`
	IsAbnormal     bool         `json:"is_abnormal" gorm:"default:false"` // 是否异常
	AbnormalReason string       `json:"abnormal_reason" gorm:"size:200"`
	Remark         string       `json:"remark" gorm:"size:200"`

	Student  *Student `json:"student,omitempty" gorm:"foreignKey:StudentID"`
	Operator *User    `json:"operator,omitempty" gorm:"foreignKey:OperatorID"`
}

// PickupAuthorization 授权接送人
type PickupAuthorization struct {
	SoftDeleteModel
	StudentID uint64     `json:"student_id" gorm:"not null;index"`
	Name      string     `json:"name" gorm:"size:50;not null"`
	Phone     string     `json:"phone" gorm:"size:20;not null"`
	Relation  string     `json:"relation" gorm:"size:20;not null"` // 与孩子关系
	IDCard    string     `json:"id_card" gorm:"size:50"`           // 身份证号
	Avatar    string     `json:"avatar" gorm:"size:500"`
	UserID    *uint64    `json:"user_id,omitempty"`   // 关联已注册用户
	ExpireAt  *time.Time `json:"expire_at,omitempty"` // 授权有效期
	Status    int8       `json:"status" gorm:"default:1;comment:1有效 0已撤销"`
	CreatorID uint64     `json:"creator_id"` // 创建人（家长）

	Student *Student `json:"student,omitempty" gorm:"foreignKey:StudentID"`
}

// PickupQRCode 接送二维码
type PickupQRCode struct {
	BaseModel
	StudentID uint64    `json:"student_id" gorm:"not null;index"`
	Code      string    `json:"code" gorm:"size:200;uniqueIndex;not null"` // 二维码内容
	ExpireAt  time.Time `json:"expire_at"`
	IsActive  bool      `json:"is_active" gorm:"default:true"`
}
