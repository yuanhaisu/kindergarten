package domain

import "time"

// 商业化域常量。取值的权威来源：openspec/changes/add-saas-commercial-layer/proposal.md
// 「已确认的产品决策」（2026-11-06 产品方确认）。
const (
	KindergartenStatusActive    = "active"
	KindergartenStatusSuspended = "suspended"

	OrderStatusUnpaid    = "unpaid"
	OrderStatusPartial   = "partial"
	OrderStatusPaid      = "paid"
	OrderStatusExpired   = "expired"
	OrderStatusCancelled = "cancelled"

	EntitlementSourceTrial        = "trial"
	EntitlementSourceSubscription = "subscription"
	EntitlementSourceGift         = "gift"

	CommissionStatusPending    = "pending"
	CommissionStatusSettleable = "settleable"
	CommissionStatusSettled    = "settled"
	CommissionStatusReversed   = "reversed"

	ModuleCategoryBase    = "base"
	ModuleCategoryPremium = "premium"

	PaymentMethodBankTransfer = "bank_transfer"
	PaymentMethodWechat       = "wechat"
	PaymentMethodCash         = "cash"
	PaymentMethodOther        = "other"
)

// PlatformAdmin 平台方运营账号。与园所管理员（users + user_roles）完全独立，
// 凭证 scope=platform 且不含 kindergarten_id（ADR-0014）。
type PlatformAdmin struct {
	BaseModel
	Username     string `gorm:"size:64;not null;uniqueIndex" json:"username"`
	PasswordHash string `gorm:"size:255;not null" json:"-"`
	Name         string `gorm:"size:64;not null;default:''" json:"name"`
	Status       int8   `gorm:"not null;default:1" json:"status"` // 1=启用 0=停用
}

func (PlatformAdmin) TableName() string { return "platform_admins" }

// Distributor 分销商/推荐人。不登录系统，只是平台方维护的名单（ADR-0016）。
type Distributor struct {
	BaseModel
	Name           string `gorm:"size:128;not null" json:"name"`
	Contact        string `gorm:"size:64;not null;default:''" json:"contact"`
	Phone          string `gorm:"size:32;not null;default:''" json:"phone"`
	CommissionRate int    `gorm:"not null;default:0" json:"commission_rate"` // 百分比整数，10 = 10%
	Status         int8   `gorm:"not null;default:1" json:"status"`          // 1=启用 0=停用
}

func (Distributor) TableName() string { return "distributors" }

// Plan 套餐目录。目录可变，但已成交订单以快照为准（ADR-0013）。
type Plan struct {
	BaseModel
	Code       string `gorm:"size:64;not null;uniqueIndex" json:"code"`
	Name       string `gorm:"size:128;not null" json:"name"`
	Tier       int    `gorm:"not null;default:1" json:"tier"`
	PriceCents int64  `gorm:"not null;default:0" json:"price_cents"`
	TermMonths int    `gorm:"not null;default:6" json:"term_months"`
	Status     string `gorm:"size:16;not null;default:'active'" json:"status"` // active / archived
}

func (Plan) TableName() string { return "plans" }

// PlanModule 套餐与功能模块的关联。
type PlanModule struct {
	BaseModel
	PlanID     uint64 `gorm:"not null;index;uniqueIndex:uk_plan_module" json:"plan_id"`
	ModuleCode string `gorm:"size:64;not null;uniqueIndex:uk_plan_module" json:"module_code"`
}

func (PlanModule) TableName() string { return "plan_modules" }

// PlanTier 幼儿数档位：按在园幼儿数决定价格系数。
type PlanTier struct {
	BaseModel
	PlanID          uint64 `gorm:"not null;index" json:"plan_id"`
	Tier            int    `gorm:"not null" json:"tier"`
	MinStudents     int    `gorm:"not null;default:0" json:"min_students"`
	MaxStudents     int    `gorm:"not null;default:0" json:"max_students"` // 0 表示无上限
	PriceMultiplier int    `gorm:"not null;default:100" json:"price_multiplier"`
}

func (PlanTier) TableName() string { return "plan_tiers" }

// Order 园所与平台方之间的成交记录。PlanSnapshot 冻结成交内容，
// 金额与授权均由快照推导（ADR-0013）。
type Order struct {
	BaseModel
	OrderNo        string     `gorm:"size:32;not null;uniqueIndex" json:"order_no"`
	KindergartenID uint64     `gorm:"not null;index" json:"kindergarten_id"`
	PlanID         uint64     `gorm:"not null" json:"plan_id"`
	PlanSnapshot   string     `gorm:"type:json;not null" json:"-"`
	StudentTier    int        `gorm:"not null;default:1" json:"student_tier"`
	AmountDueCents int64      `gorm:"not null;default:0" json:"amount_due_cents"`
	Status         string     `gorm:"size:16;not null;default:'unpaid'" json:"status"`
	DistributorID  *uint64    `gorm:"index" json:"distributor_id"`
	CommissionRate int        `gorm:"not null;default:0" json:"-"`
	InvoiceInfo    string     `gorm:"type:json" json:"-"`
	StartsAt       time.Time  `gorm:"not null" json:"starts_at"`
	ExpiresAt      time.Time  `gorm:"not null" json:"expires_at"`
	ConfirmedAt    *time.Time `json:"confirmed_at"`
	CancelledAt    *time.Time `json:"cancelled_at"`
	CancelReason   string     `gorm:"size:255;not null;default:''" json:"cancel_reason"`
	CreatedBy      string     `gorm:"size:64;not null;default:''" json:"created_by"`
}

func (Order) TableName() string { return "orders" }

// Payment 收款登记。退款以负数金额表达，不设独立退款单（ADR-0016 / design D7）。
// 负数收款必须用 RefundOfPaymentID 指明被退的原收款，佣金冲回据此精确定位。
type Payment struct {
	BaseModel
	OrderID           uint64    `gorm:"not null;index" json:"order_id"`
	RefundOfPaymentID *uint64   `gorm:"index" json:"refund_of_payment_id"`
	KindergartenID    uint64    `gorm:"not null;index" json:"kindergarten_id"`
	AmountCents       int64     `gorm:"not null" json:"amount_cents"`
	Method            string    `gorm:"size:32;not null;default:'bank_transfer'" json:"method"`
	ReceivedAt        time.Time `gorm:"not null" json:"received_at"`
	VoucherURL        string    `gorm:"size:512;not null;default:''" json:"voucher_url"`
	Remark            string    `gorm:"size:255;not null;default:''" json:"remark"`
	RecordedBy        string    `gorm:"size:64;not null;default:''" json:"recorded_by"`
}

func (Payment) TableName() string { return "payments" }

// Entitlement 园所对某个功能模块的使用权。多个授权可叠加，各自独立到期（ADR-0012/0013）。
type Entitlement struct {
	BaseModel
	KindergartenID uint64    `gorm:"not null;index:idx_ent_kg_module" json:"kindergarten_id"`
	ModuleCode     string    `gorm:"size:64;not null;index:idx_ent_kg_module" json:"module_code"`
	Source         string    `gorm:"size:16;not null;default:'subscription'" json:"source"`
	OrderID        *uint64   `gorm:"index" json:"order_id"`
	StartsAt       time.Time `gorm:"not null" json:"starts_at"`
	ExpiresAt      time.Time `gorm:"not null;index" json:"expires_at"`
}

func (Entitlement) TableName() string { return "entitlements" }

// CommissionLedger 佣金台账。金额由收款与订单冻结比例推导，不提供手工改写（design D8）。
// 原始行的净额 = AmountCents - ReversedAmountCents；净额归零即视为已冲回。
// 冲回行（ReversalOf 非空）是审计记录，其金额已累计进原始行的 ReversedAmountCents，
// 不参与台账明细与汇总，避免重复扣减。
type CommissionLedger struct {
	BaseModel
	OrderID             uint64    `gorm:"not null;index" json:"order_id"`
	PaymentID           uint64    `gorm:"not null;index" json:"payment_id"`
	DistributorID       uint64    `gorm:"not null;index" json:"distributor_id"`
	KindergartenID      uint64    `gorm:"not null;index" json:"kindergarten_id"`
	BaseAmountCents     int64     `gorm:"not null" json:"base_amount_cents"`
	Rate                int       `gorm:"not null" json:"rate"`
	AmountCents         int64     `gorm:"not null" json:"amount_cents"`
	ReversedAmountCents int64     `gorm:"not null;default:0" json:"reversed_amount_cents"`
	Status              string    `gorm:"size:16;not null;default:'pending'" json:"status"`
	EligibleAt          time.Time `gorm:"not null" json:"eligible_at"`
	ReversalOf          *uint64   `gorm:"index" json:"reversal_of"`
}

func (CommissionLedger) TableName() string { return "commission_ledger" }

// PlatformOperationLog 平台方敏感操作留痕。与园所的 operation_logs 分开，
// 避免平台行为混入园所审计视图。
type PlatformOperationLog struct {
	BaseModel
	Operator   string `gorm:"size:64;not null;index" json:"operator"`
	Module     string `gorm:"size:50;not null" json:"module"`
	Action     string `gorm:"size:50;not null" json:"action"`
	TargetType string `gorm:"size:50;not null;default:''" json:"target_type"`
	TargetID   uint64 `gorm:"not null;default:0" json:"target_id"`
	Detail     string `gorm:"type:text" json:"detail"`
	IP         string `gorm:"size:50;not null;default:''" json:"ip"`
}

func (PlatformOperationLog) TableName() string { return "platform_operation_logs" }
