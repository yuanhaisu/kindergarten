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
//
// 托管转移的方向由 Type 表达：1=送入（家庭→幼儿园）2=接走（幼儿园→家庭）。
// 同一学生同日允许「送入—接走」多次交替，不允许连续同向登记。
//
// 送入（DropOff）只作到场标记：不记送托人、不要理由（有意偏离
// design-data/admin-05-接送记录页.json 的「入园/妈妈（李丽）」示例，见
// openspec/changes/pickup-custody-evidence/proposal.md 分歧清单）。
// 接走（PickUp）才做证据留痕，见 PickupPersonID / IsAuthorizedOverride。
type PickupRecord struct {
	BaseModel
	KindergartenID uint64       `json:"kindergarten_id" gorm:"not null;index"`
	ClassID        uint64       `json:"class_id" gorm:"not null;index"`
	StudentID      uint64       `json:"student_id" gorm:"not null;index"`
	Date           time.Time    `json:"date" gorm:"type:date;not null;index"`
	Type           PickupType   `json:"type" gorm:"not null;comment:1送入 2接走"`
	Method         PickupMethod `json:"method" gorm:"not null;comment:1扫码 2手动"`
	// PickupPersonID 本次接走人对应的授权记录。
	// 接走且未走例外通道时必须有值（I1 授权闭合的落库依据）；
	// 例外通道与送入时为 NULL。
	PickupPersonID *uint64      `json:"pickup_person_id,omitempty" gorm:"index"`
	PickupName     string       `json:"pickup_name" gorm:"size:50"`
	PickupRelation string       `json:"pickup_relation" gorm:"size:20"`
	// OperatorID 登记/核验人（园所方教职工）。
	// 契约列名「核验人」，对齐 design-data/admin-05；本次不建双人流程，
	// 故接送人与核验人不分列，只此一人。
	OperatorID     uint64       `json:"operator_id"`
	RecordedAt     time.Time    `json:"recorded_at"`
	// IsAuthorizedOverride 是否走了授权例外通道（来接者不在有效授权集合内）。
	// 与 PickupPersonID 互补：PickupPersonID 为空无法区分「漏选」与「有意例外」，
	// 故显式记录，使例外可观测统计。例外不阻断登记。
	IsAuthorizedOverride bool   `json:"is_authorized_override" gorm:"default:false"`
	OverrideName         string `json:"override_name" gorm:"size:50"`
	// IsAbnormal 由老师手动标记，系统不因重复接走/提前接走/例外而自动置位
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
	// Status 1有效 0已撤销。
	// 不加 gorm:"default:1" —— int8 零值 0 会被 GORM 当「未赋值」跳过并回填 DB 默认值，
	// 导致「已撤销」被写成「有效」。取值由调用方显式给出（见 PickupRepo.CreateAuthorization）。
	Status    int8       `json:"status" gorm:"comment:1有效 0已撤销"`
	CreatorID uint64     `json:"creator_id"`
	AuthType          string     `json:"auth_type" gorm:"size:20;default:permanent;comment:permanent长期 temporary临时"`
	// 下列布尔字段同理：false 是有意义的取值，不得依赖 DB 默认值。
	AllowPickup       bool       `json:"allow_pickup" gorm:"comment:允许接送（入离园）"`
	AllowViewRecord   bool       `json:"allow_view_record" gorm:"comment:允许查看接送记录"`
	AllowViewStatus   bool       `json:"allow_view_status" gorm:"comment:允许查看在校情况"`
	AllowGrowthView   bool       `json:"allow_growth_view" gorm:"comment:允许查看成长记录"`
	NotifyOnPickup    bool       `json:"notify_on_pickup" gorm:"comment:接送消息通知"`
	// ReviewStatus 园所审核：0 待审核 1 通过 2 拒绝
	ReviewStatus int8       `json:"review_status" gorm:"default:0;index"`
	AuditedAt    *time.Time `json:"audited_at,omitempty"`
	AuditRemark  string     `json:"audit_remark" gorm:"size:200"`

	Student *Student `json:"student,omitempty" gorm:"foreignKey:StudentID"`
}

func (PickupAuthorization) TableName() string { return "pickup_authorizations" }

// PickupQRCode 接送二维码
//
// 凭证绑「学生 + 今日接走人」，而非仅绑学生（I4 不可抵赖：票绑人）。
// Code 为纯随机串，不内嵌 student_id 等可枚举标识；学生与接走人由后端按码查库解出。
// 当日过期、一次有效（成功接走后置 IsActive=false）。
type PickupQRCode struct {
	BaseModel
	StudentID       uint64    `json:"student_id" gorm:"not null;index"`
	AuthorizationID uint64    `json:"authorization_id" gorm:"not null;index;comment:今日接走人对应的授权记录"`
	Code            string    `json:"code" gorm:"size:200;uniqueIndex;not null"`
	ExpireAt        time.Time `json:"expire_at"`
	// IsActive 不加 gorm:"default:true" —— bool 零值 false 会被 GORM 当作「未赋值」
	// 跳过该列并回填 DB 默认值，导致「已使用的凭证」被写成有效。
	// 默认值由调用方显式给出（见 PickupRepo.CreateQRCode 的 Select 全字段）。
	IsActive bool `json:"is_active"`

	Authorization *PickupAuthorization `json:"authorization,omitempty" gorm:"foreignKey:AuthorizationID"`
}

func (PickupQRCode) TableName() string { return "pickup_qr_codes" }

// PickupDailyDesignation 今日接走人指定
//
// 家长在授权人列表里指定「今天谁来接走」。与凭证分离的原因：码是一次性易失的
// （用完即焚），若指定只活在码上，焚掉后「原计划谁来接」无处可查，
// 而「计划接走人 vs 实际接走人」正是审计要回答的问题（I3）。
//
// 同一学生同一日只有一条（UNIQUE(student_id, date)），重复指定覆盖旧行，
// 并使该生当日已生成的全部凭证失效。
type PickupDailyDesignation struct {
	BaseModel
	KindergartenID  uint64    `json:"kindergarten_id" gorm:"not null;index"`
	StudentID       uint64    `json:"student_id" gorm:"not null;uniqueIndex:idx_pickup_designation_student_date"`
	AuthorizationID uint64    `json:"authorization_id" gorm:"not null"`
	Date            time.Time `json:"date" gorm:"type:date;not null;uniqueIndex:idx_pickup_designation_student_date"`

	Authorization *PickupAuthorization `json:"authorization,omitempty" gorm:"foreignKey:AuthorizationID"`
	Student       *Student             `json:"student,omitempty" gorm:"foreignKey:StudentID"`
}

func (PickupDailyDesignation) TableName() string { return "pickup_daily_designations" }
