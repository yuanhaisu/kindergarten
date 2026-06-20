package domain

import "time"

type PickupType int8

const (
	PickupTypeDropOff PickupType = 1
	PickupTypePickUp  PickupType = 2
)

type PickupMethod int8

const (
	PickupMethodScan   PickupMethod = 1
	PickupMethodManual PickupMethod = 2
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
	PickupPersonID *uint64      `json:"pickup_person_id,omitempty"`
	PickupName     string       `json:"pickup_name" gorm:"size:50"`
	PickupRelation string       `json:"pickup_relation" gorm:"size:20"`
	OperatorID     uint64       `json:"operator_id"`
	RecordedAt     time.Time    `json:"recorded_at"`
	IsAbnormal     bool         `json:"is_abnormal" gorm:"default:false"`
	AbnormalReason string       `json:"abnormal_reason" gorm:"size:200"`
	Remark         string       `json:"remark" gorm:"size:200"`

	Student  *Student `json:"student,omitempty" gorm:"foreignKey:StudentID"`
	Operator *User    `json:"operator,omitempty" gorm:"foreignKey:OperatorID"`
}

func (PickupRecord) TableName() string { return "pickup_records" }

// PickupAuthorization 授权接送人
type PickupAuthorization struct {
	SoftDeleteModel
	StudentID uint64     `json:"student_id" gorm:"not null;index"`
	Name      string     `json:"name" gorm:"size:50;not null"`
	Phone     string     `json:"phone" gorm:"size:20;not null"`
	Relation  string     `json:"relation" gorm:"size:20;not null"`
	IDCard    string     `json:"id_card" gorm:"size:50"`
	Avatar    string     `json:"avatar" gorm:"size:500"`
	UserID    *uint64    `json:"user_id,omitempty"`
	ExpireAt  *time.Time `json:"expire_at,omitempty"`
	Status    int8       `json:"status" gorm:"default:1;comment:1有效 0已撤销"`
	CreatorID uint64     `json:"creator_id"`
	AuthType          string     `json:"auth_type" gorm:"size:20;default:permanent;comment:permanent长期 temporary临时"`
	AllowPickup       bool       `json:"allow_pickup" gorm:"default:true;comment:允许接送（入离园）"`
	AllowViewRecord   bool       `json:"allow_view_record" gorm:"default:false;comment:允许查看接送记录"`
	AllowViewStatus   bool       `json:"allow_view_status" gorm:"default:false;comment:允许查看在校情况"`
	AllowGrowthView   bool       `json:"allow_growth_view" gorm:"default:false;comment:允许查看成长记录"`
	NotifyOnPickup    bool       `json:"notify_on_pickup" gorm:"default:true;comment:接送消息通知"`
	// ReviewStatus 园所审核：0 待审核 1 通过 2 拒绝
	ReviewStatus int8       `json:"review_status" gorm:"default:0;index"`
	AuditedAt    *time.Time `json:"audited_at,omitempty"`
	AuditRemark  string     `json:"audit_remark" gorm:"size:200"`

	Student *Student `json:"student,omitempty" gorm:"foreignKey:StudentID"`
}

func (PickupAuthorization) TableName() string { return "pickup_authorizations" }

// PickupQRCode 接送二维码
type PickupQRCode struct {
	BaseModel
	StudentID uint64    `json:"student_id" gorm:"not null;index"`
	Code      string    `json:"code" gorm:"size:200;uniqueIndex;not null"`
	ExpireAt  time.Time `json:"expire_at"`
	IsActive  bool      `json:"is_active" gorm:"default:true"`
}

func (PickupQRCode) TableName() string { return "pickup_qr_codes" }
