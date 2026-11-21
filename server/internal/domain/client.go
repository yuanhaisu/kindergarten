package domain

// 小程序客户端注册表（#163）。
//
// 每个客户端对应一个微信小程序 AppID；登录从注册表取角色与凭据，不再把
// 两端凭据写死在服务里。v1 只启用两条 shared 客户端（家长端 / 老师端），
// 由配置在启动时幂等同步。
//
// 生命周期：客户端不删除，只置 status=disabled（身份仍引用 app_id）。
// status 是运行时唯一来源，配置同步不覆盖它，也不覆盖 release_version。
const (
	ClientRoleParent  = "parent"
	ClientRoleTeacher = "teacher"

	// ClientScopeShared 服务所有 app_channel=shared 的园所。
	// kindergarten / group 只在规格中预留取值，v1 不落库。
	ClientScopeShared       = "shared"
	ClientScopeKindergarten = "kindergarten"
	ClientScopeGroup        = "group"

	ClientStatusEnabled  = "enabled"
	ClientStatusDisabled = "disabled"
)

type Client struct {
	BaseModel
	// AppID 微信小程序 AppID。AppID 变更时旧行保留并置 disabled，因此必须唯一，
	// 否则同步与登录都无法确定该用哪一行。
	AppID string `gorm:"size:128;not null;uniqueIndex:uk_clients_app_id" json:"app_id"`
	Role  string `gorm:"size:16;not null;index" json:"role"`
	// ScopeKind 见 ClientScope* 常量；ScopeID 在 shared 作用域固定为 0。
	ScopeKind string `gorm:"size:16;not null;default:'shared'" json:"scope_kind"`
	ScopeID   uint64 `gorm:"not null;default:0" json:"scope_id"`
	Status    string `gorm:"size:16;not null;default:'enabled';index" json:"status"`
	// ReleaseVersion 该客户端当前发版版本（诊断用，不参与登录校验）。
	ReleaseVersion string `gorm:"size:64;not null;default:''" json:"release_version"`
	// CredentialRef 指向凭据所在配置项（如 wechat.teacher），只存引用不存密钥。
	CredentialRef string `gorm:"size:64;not null;default:''" json:"credential_ref"`
}

func (Client) TableName() string { return "clients" }

func (c Client) IsEnabled() bool { return c.Status == ClientStatusEnabled }
