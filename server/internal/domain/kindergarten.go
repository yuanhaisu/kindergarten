package domain

// 园所租户通道（#177）：决定哪个小程序客户端可以服务该园所。
//
// v1 所有园所都是 shared；dedicated / group 只保留取值，不建 UI/行/流水线。
// group 通道还需要集团实体与成员表，v1 没有，因此永远不可设置。
const (
	AppChannelShared    = "shared"
	AppChannelDedicated = "dedicated"
	AppChannelGroup     = "group"
)

// IsValidAppChannel 报告通道取值是否合法。空串等历史脏值一律非法。
func IsValidAppChannel(channel string) bool {
	switch channel {
	case AppChannelShared, AppChannelDedicated, AppChannelGroup:
		return true
	default:
		return false
	}
}

type Kindergarten struct {
	BaseModel
	// Status 园所状态：active 在营 / suspended 停用（商业化变更引入）。
	// 用 json:"-" 避免影响既有的园所信息接口；平台端显式序列化。
	Status  string `gorm:"size:16;not null;default:'active'" json:"-"`
	Name    string `gorm:"size:128;not null" json:"name"`
	Address string `gorm:"size:256" json:"address"`

	Phone         string `gorm:"size:32;not null;default:''" json:"-"`
	Principal     string `gorm:"size:64;not null;default:''" json:"-"`
	LogoURL       string `gorm:"size:512;not null;default:''" json:"-"`
	LicenseNo     string `gorm:"size:128;not null;default:''" json:"-"`
	Capacity      int    `gorm:"not null;default:0" json:"-"`
	EstablishedAt string `gorm:"size:16;not null;default:''" json:"-"` // YYYY-MM-DD
	Description   string `gorm:"type:text" json:"-"`

	// AppChannel 园所通道，见 AppChannel* 常量。v1 全为 shared。
	AppChannel string `gorm:"size:16;not null;default:'shared';index" json:"-"`
	// PublicCode 园所公开入口码（scene 为 kg_<public_code>）。随机生成、全局唯一，
	// 软删园所的码不复用。空串表示尚未生成，由入口服务回填。
	// 唯一索引建在 DB 生成列 public_code_uniq 上（空码转为 NULL），
	// 因此这里不能加 uniqueIndex tag：AutoMigrate 会把多个空串当成重复键。
	PublicCode string `gorm:"size:16;not null;default:''" json:"-"`
	// BrandColor 园所品牌主题色（#178 配置）。空表示使用默认主题。
	BrandColor string `gorm:"size:32;not null;default:''" json:"-"`
}

func (Kindergarten) TableName() string { return "kindergartens" }

type Class struct {
	BaseModel
	KindergartenID uint64 `gorm:"index;not null" json:"kindergarten_id"`
	Name           string `gorm:"size:64;not null" json:"name"`
	Grade          string `gorm:"size:16;not null;default:''" json:"grade"`
	AcademicYear   string `gorm:"size:32;not null;default:''" json:"academic_year"`
	Capacity       int    `gorm:"not null;default:30" json:"capacity"`
	Status         int8   `gorm:"not null;default:1" json:"status"`
	// PickupDeadline 班级放学时刻，格式 "HH:mm"。空串表示未配置，判定时回退缺省 16:00
	// （产品决策 2026-10-07：班级可配、默认 16:00；design-data 无出处，见 proposal 阻塞清单）。
	// 用于「提前接走须填理由」与「超时未接」两处判定。
	// 用 string 存时刻而非 time.Time：时刻不含日期分量，且与 EstablishedAt 的既有惯例一致。
	PickupDeadline string `gorm:"size:5;not null;default:''" json:"pickup_deadline"`

	Kindergarten *Kindergarten `json:"kindergarten,omitempty" gorm:"foreignKey:KindergartenID"`
}

func (Class) TableName() string { return "classes" }
