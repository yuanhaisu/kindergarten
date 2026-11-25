package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/storage/oss"
)

// AuthUserRepository AuthService 依赖的用户仓库接口（由 *repository.UserRepo 隐式实现）
type AuthUserRepository interface {
	FindByPhone(ctx context.Context, phone string) (*domain.User, error)
	FindByID(ctx context.Context, id uint64) (*domain.User, error)
	Create(ctx context.Context, user *domain.User) error
	Update(ctx context.Context, user *domain.User) error
	UpdateFields(ctx context.Context, id uint64, fields map[string]any) error
}

// AuthRoleRepository AuthService 依赖的角色仓库接口（由 *repository.UserRoleRepo 隐式实现）
type AuthRoleRepository interface {
	FindByUserID(ctx context.Context, userID uint64) ([]domain.UserRole, error)
}

// AdminPermissionResolver 解析管理端账号权限码（由 *AdminAuthzService 实现）。
// 声明在消费方，便于 AuthService 单测注入桩。
type AdminPermissionResolver interface {
	ResolveAdminCodes(ctx context.Context, kindergartenID, userID uint64) ([]string, bool, error)
}

// ClientResolver 按 app_id 解析启用中的小程序客户端（由 *ClientService 实现）。
// 登录的角色与微信凭据都来自这里，不信任请求 body（#175）。
type ClientResolver interface {
	ResolveEnabledByAppID(ctx context.Context, appID string) (*domain.Client, error)
}

// UserIdentityRepository 客户端身份仓储接口（由 *repository.UserIdentityRepo 实现）。
type UserIdentityRepository interface {
	FindOrCreatePendingUser(ctx context.Context, appID, openID, unionID string) (*domain.User, error)
	// FindByUserAndApp 用于续期/切换时回填 claims 里的 open_id：绑定手机号
	// 依赖它定位客户端身份，不能因为刷新过 token 就丢掉。
	FindByUserAndApp(ctx context.Context, userID uint64, appID string) (*domain.UserIdentity, error)
}

// KindergartenMembershipRepository 读取成员园所的名称与通道（#169）。
// 「我的园所」与切换都必须按真实园所行过滤，user_roles 里冗余的名称不可信。
type KindergartenMembershipRepository interface {
	FindByID(ctx context.Context, id uint64) (*domain.Kindergarten, error)
	FindByIDs(ctx context.Context, ids []uint64) ([]domain.Kindergarten, error)
}

// AuthService v1 认证服务。
//
// 小程序登录（#175）持有微信 API 与客户端注册表：app_id → 客户端角色/作用域，
// 再用该角色对应的微信凭据换 openid。邀请码绑定仍由 #168 接管。
type AuthService struct {
	userRepo       AuthUserRepository
	userRoleRepo   AuthRoleRepository
	wx             WeChatAPI
	clients        ClientResolver
	identities     UserIdentityRepository
	adminPerms     AdminPermissionResolver
	kgStatus       KindergartenStatusChecker
	kindergartens  KindergartenMembershipRepository
	rdb            *redis.Client
	revoker        auth.TokenRevoker
	tenantEpochs   auth.TenantEpochStore
	secret         []byte
	atTTL          time.Duration
	rtTTL          time.Duration
	consentRepo    ConsentRepository
	consentEnforce bool

	// 家长绑定手机号（#167）的依赖。三者任一缺失时端点报未装配，
	// 而不是让绑定流程跑一半。
	bindKindergartens KindergartenLookup
	merger            IdentityMerger
	bindLimiter       BindPhoneRateLimiter
}

func NewAuthService(
	userRepo AuthUserRepository,
	userRoleRepo AuthRoleRepository,
	wx WeChatAPI,
	clients ClientResolver,
	identities UserIdentityRepository,
	adminPerms AdminPermissionResolver,
	kgStatus KindergartenStatusChecker,
	kindergartens KindergartenMembershipRepository,
	rdb *redis.Client,
	secret []byte,
	atTTL time.Duration,
	rtTTL time.Duration,
	consentRepo ConsentRepository,
	consentEnforce bool,
	bindKindergartens KindergartenLookup,
	merger IdentityMerger,
	bindLimiter BindPhoneRateLimiter,
) *AuthService {
	return &AuthService{
		userRepo:          userRepo,
		userRoleRepo:      userRoleRepo,
		wx:                wx,
		clients:           clients,
		identities:        identities,
		adminPerms:        adminPerms,
		kgStatus:          kgStatus,
		kindergartens:     kindergartens,
		rdb:               rdb,
		revoker:           auth.NewTokenRevoker(rdb),
		tenantEpochs:      auth.NewTenantEpochStore(rdb, auth.PlatformEpochTTLFor(rtTTL, atTTL)),
		secret:            secret,
		atTTL:             atTTL,
		rtTTL:             rtTTL,
		consentRepo:       consentRepo,
		consentEnforce:    consentEnforce,
		bindKindergartens: bindKindergartens,
		merger:            merger,
		bindLimiter:       bindLimiter,
	}
}

type LoginResult struct {
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	User         *UserInfo `json:"user"`
	// 增量字段：旧客户端忽略即可；新版客户端据此进入隐私政策阻断页。
	ConsentRequired      bool   `json:"consent_required"`
	CurrentPolicyVersion string `json:"current_policy_version"`
}

type UserInfo struct {
	ID                 uint64     `json:"id"`
	Phone              string     `json:"phone"`
	Nickname           string     `json:"nickname"`
	Avatar             string     `json:"avatar"`
	Gender             int        `json:"gender"`
	MustChangePassword bool       `json:"must_change_password"`
	Roles              []RoleInfo `json:"roles"`
}

type RoleInfo struct {
	Role             string   `json:"role"`
	KindergartenID   *uint64  `json:"kindergarten_id,omitempty"`
	KindergartenName string   `json:"kindergarten_name,omitempty"`
	ClassID          *uint64  `json:"class_id,omitempty"`
	ClassName        string   `json:"class_name,omitempty"`
	Title            string   `json:"title,omitempty"`
	TitleLabel       string   `json:"title_label,omitempty"`
	AdminRole        string   `json:"admin_role,omitempty"`
	Permissions      []string `json:"permissions,omitempty"`
}

// RoleScope 限定一次请求可见的角色族（#175 / #160 会话单园）。
//
// 小程序凭证（带 app_id）只允许看到「当前客户端角色族 ∩ 当前会话园所」的角色，
// 且 must_change_password 只属于管理端密码登录，对小程序凭证不生效。
// 完整成员关系走「我的园所」，不混在 roles 里。
// zero value 表示不限制：管理端、平台方与旧凭证行为不变。
type RoleScope struct {
	clientScoped   bool
	allowed        []string
	kindergartenID uint64
}

// RoleScopeFromClaims 由访问凭证推导角色可见范围。
// app_id 存在即视为小程序客户端凭证；允许集合取凭证里的角色（签发时已按
// 角色族裁剪），园所取凭证里的会话园所——token 始终只代表一个园。
func RoleScopeFromClaims(claims *auth.AccessClaims) RoleScope {
	if claims == nil || claims.AppID == "" {
		return RoleScope{}
	}
	return RoleScope{clientScoped: true, allowed: claims.Roles, kindergartenID: claims.KindergartenID}
}

// clientRoleScope 是登录路径的作用域：roles 已按客户端角色族裁剪过，
// kindergartenID 是本次签发 token 的会话园所。
func clientRoleScope(roleNames []string, kindergartenID uint64) RoleScope {
	return RoleScope{clientScoped: true, allowed: roleNames, kindergartenID: kindergartenID}
}

func (s RoleScope) Allows(role string) bool {
	if !s.clientScoped {
		return true
	}
	for _, allowed := range s.allowed {
		if allowed == role {
			return true
		}
	}
	return false
}

func (s RoleScope) filter(roles []domain.UserRole) []domain.UserRole {
	if !s.clientScoped {
		return roles
	}
	filtered := make([]domain.UserRole, 0, len(roles))
	for _, r := range roles {
		if !s.Allows(r.Role) {
			continue
		}
		// 会话单园：响应里的 roles 必须与 token 的 kindergarten_id 一致，
		// 否则两端多园账号会拿到 token 之外的园所角色（#160）。
		if r.KindergartenID == nil || *r.KindergartenID != s.kindergartenID {
			continue
		}
		filtered = append(filtered, r)
	}
	return filtered
}

func (s *AuthService) buildUserInfo(ctx context.Context, user *domain.User, scope RoleScope) (*UserInfo, error) {
	roles, err := s.userRoleRepo.FindByUserID(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	roles = scope.filter(roles)

	roleInfos := make([]RoleInfo, 0, len(roles))
	for _, r := range roles {
		roleInfos = append(roleInfos, RoleInfo{
			Role:             r.Role,
			KindergartenID:   r.KindergartenID,
			KindergartenName: r.KindergartenName,
			ClassID:          r.ClassID,
			ClassName:        r.ClassName,
			Title:            r.Title,
			TitleLabel:       r.TitleLabel,
			AdminRole:        r.AdminRole,
		})
	}

	return &UserInfo{
		ID:       user.ID,
		Phone:    maskPhone(user.Phone),
		Nickname: user.Nickname,
		Avatar:   user.Avatar,
		Gender:   user.Gender,
		// 改密流程只存在于管理端（#166）；小程序客户端登录必须忽略该标记，
		// 否则家长/老师会被卡在一个小程序里无法完成的流程上。
		MustChangePassword: user.MustChangePassword && !scope.clientScoped,
		Roles:              roleInfos,
	}, nil
}

// ChangePassword 自助改密。平台开通园所时创建的园长账号首次登录后必须走这里，
// 改密成功后清除 must_change_password。
func (s *AuthService) ChangePassword(ctx context.Context, userID uint64, oldPassword, newPassword string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return fmt.Errorf("USER_NOT_FOUND")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return fmt.Errorf("PASSWORD_INCORRECT")
	}
	if err := validatePassword(newPassword); err != nil {
		return fmt.Errorf("WEAK_PASSWORD: %w", err)
	}
	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	// 账号级失效（#182）：先推进 epoch 再改库，Redis 不可用时失败关闭——
	// 不能出现「口令已换、旧会话仍能用旧 access/refresh」。
	if err := s.bumpTenantEpoch(ctx, userID); err != nil {
		return fmt.Errorf("%w: %v", ErrAuthStorageUnavailable, err)
	}
	return s.userRepo.UpdateFields(ctx, userID, map[string]any{
		"password_hash":        hash,
		"must_change_password": false,
	})
}

// issueAdminTokens 签发管理端会话（#166）：先收敛出唯一园所的管理员角色，
// 再走统一签发。access/refresh 都落在 admin_web 作用域。
func (s *AuthService) issueAdminTokens(ctx context.Context, user *domain.User, roles []domain.UserRole) (*LoginResult, error) {
	adminRoles, _, err := adminSessionRoles(roles)
	if err != nil {
		return nil, err
	}
	return s.issueTokensFor(ctx, user, adminRoles, nil, "")
}

// adminSessionRoles 返回管理端会话可用的角色行：必须恰好属于一个园所。
//
// 「一个手机号只能是一个园所的管理员」是 #160 的身份约束。0 个园所无法确定
// 租户；多个园所会让 token 的 kindergarten_id 变成一个任意选择，必须拒绝。
func adminSessionRoles(roles []domain.UserRole) ([]domain.UserRole, uint64, error) {
	var (
		adminRoles []domain.UserRole
		kgID       uint64
	)
	for _, r := range roles {
		if r.Role != "admin" || r.KindergartenID == nil {
			continue
		}
		if kgID == 0 {
			kgID = *r.KindergartenID
		} else if kgID != *r.KindergartenID {
			return nil, 0, ErrAdminKindergartenConflict
		}
		adminRoles = append(adminRoles, r)
	}
	if kgID == 0 {
		return nil, 0, ErrAdminKindergartenRequired
	}
	return adminRoles, kgID, nil
}

// issueClientTokens 签发绑定小程序客户端的凭证：claims 带 app_id 与作用域，
// 角色只来自当前客户端注册表（#175）。openID 一并写进 claims，绑定手机号（#167）
// 需要用它在 pending 已被硬删的重试里定位身份。
func (s *AuthService) issueClientTokens(ctx context.Context, user *domain.User, roles []domain.UserRole, client *domain.Client, openID string) (*LoginResult, error) {
	return s.issueTokensFor(ctx, user, roles, client, openID)
}

// IssueClientLogin 为已完成绑定的账号按当前客户端签发凭证（#168）。
//
// 绑定把待绑定身份并入园所预创建账号后，必须立刻换发一枚属于预创建账号、
// 只含该客户端角色族的 token；否则客户端仍会拿着空角色的待绑定凭证。
func (s *AuthService) IssueClientLogin(ctx context.Context, userID uint64, client *domain.Client) (*LoginResult, error) {
	if client == nil {
		return nil, fmt.Errorf("issue client login: client is nil")
	}
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("issue client login: user %d not found", userID)
	}
	roles, err := s.userRoleRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	// 绑定把客户端身份挂到了这个账号，token 要带上 open_id（#167 续签/再绑定依赖）。
	openID, err := s.clientOpenID(ctx, userID, client)
	if err != nil {
		return nil, err
	}
	return s.issueClientTokens(ctx, user, filterRolesByFamily(roles, client.Role), client, openID)
}

// clientOpenID 取该账号在当前客户端的身份 open_id。续期与切换都要把它
// 写回 claims；身份缺失不是刷新失败的理由（待绑定用户也有身份，缺失只
// 会让后续绑定报 IDENTITY_MISSING），但仓储错误必须显式失败。
func (s *AuthService) clientOpenID(ctx context.Context, userID uint64, client *domain.Client) (string, error) {
	if s.identities == nil || client == nil || client.AppID == "" {
		return "", nil
	}
	identity, err := s.identities.FindByUserAndApp(ctx, userID, client.AppID)
	if err != nil {
		return "", fmt.Errorf("load client identity: %w", err)
	}
	if identity == nil {
		return "", nil
	}
	return identity.OpenID, nil
}

// issueTokensFor 是签发的唯一实现。client 为 nil 时是管理端/刷新路径（无 app_id），
// 非 nil 时必须走 SignClientAccess，否则 token 无法回查客户端作用域。
func (s *AuthService) issueTokensFor(ctx context.Context, user *domain.User, roles []domain.UserRole, client *domain.Client, openID string) (*LoginResult, error) {
	roleNames := roleNamesFromUserRoles(roles)
	kgID := kindergartenIDFromUserRoles(roles)

	// 停用园所不签发新凭证：登录与刷新都会走到这里。
	if kgID > 0 && s.kgStatus != nil {
		suspended, err := s.kgStatus.Suspended(ctx, kgID)
		if err != nil {
			return nil, err
		}
		if suspended {
			return nil, ErrKindergartenSuspended
		}
	}

	uid := fmt.Sprintf("%d", user.ID)
	var (
		accessToken string
		err         error
	)
	switch {
	case client != nil:
		accessToken, err = auth.SignClientAccess(s.secret, uid, roleNames, kgID, auth.ClientScope{
			AppID:  client.AppID,
			Kind:   client.ScopeKind,
			ID:     client.ScopeID,
			OpenID: openID,
		}, s.atTTL)
	case hasAdminRole(roles):
		// 管理端内建作用域：无 app_id，必须恰好一个园所（#166）。
		accessToken, err = auth.SignAdminAccess(s.secret, uid, roleNames, kgID, s.atTTL)
	default:
		accessToken, err = auth.SignAccess(s.secret, uid, roleNames, kgID, s.atTTL)
	}
	if err != nil {
		return nil, err
	}

	// refresh 存储带上完整作用域（#169）：刷新/切换时按同一客户端与园所续签。
	refreshScope := auth.RefreshScope{UID: user.ID, KindergartenID: kgID, IssuedAt: time.Now().Unix()}
	if client != nil {
		refreshScope.AppID = client.AppID
		refreshScope.ScopeKind = client.ScopeKind
		refreshScope.ScopeID = client.ScopeID
	} else {
		refreshScope.ScopeKind = auth.RefreshScopeAdminWeb
		if hasAdminRole(roles) {
			// 改密后旧 refresh 失效；指纹随每次签发刷新（#166）。
			refreshScope.PasswordVersion = auth.PasswordFingerprint(user.PasswordHash)
		}
	}
	payload, err := auth.EncodeRefreshScope(refreshScope)
	if err != nil {
		return nil, err
	}
	jti := uuid.NewString()
	if err := auth.SaveRefresh(ctx, s.rdb, jti, payload, s.rtTTL); err != nil {
		return nil, err
	}

	// 响应里的角色必须与 token 一致：小程序路径用裁剪后的角色族，
	// 其余路径保持全量（#175）。
	scope := RoleScope{}
	if client != nil {
		scope = clientRoleScope(roleNames, kgID)
	}
	userInfo, err := s.buildUserInfo(ctx, user, scope)
	if err != nil {
		return nil, err
	}

	consentRequired, err := s.consentRequiredFor(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		Token:                accessToken,
		RefreshToken:         jti,
		ExpiresAt:            time.Now().Add(s.atTTL),
		User:                 userInfo,
		ConsentRequired:      consentRequired,
		CurrentPolicyVersion: CurrentPrivacyPolicyVersion,
	}, nil
}

// roleNamesFromUserRoles 去重后返回角色名。
// 空角色就是空：待绑定账号没有角色，兜底 "parent" 会让未绑定用户直接拿到
// 家长端权限，绕过绑定流程（#175）。
func roleNamesFromUserRoles(roles []domain.UserRole) []string {
	var names []string
	seen := make(map[string]bool)
	for _, r := range roles {
		if !seen[r.Role] {
			seen[r.Role] = true
			names = append(names, r.Role)
		}
	}
	return names
}

// kindergartenIDFromUserRoles 优先取 admin 角色绑定的园所，否则取任意一条带园所的角色
func kindergartenIDFromUserRoles(roles []domain.UserRole) uint64 {
	for _, r := range roles {
		if r.Role == "admin" && r.KindergartenID != nil {
			return *r.KindergartenID
		}
	}
	for _, r := range roles {
		if r.KindergartenID != nil {
			return *r.KindergartenID
		}
	}
	return 0
}

// 短信登录（SendSMS / verifySMSCode / SMSLogin）已在 #162 关闭并删除：
// handler 固定返回 403，保留一套没人调用、却忽略 role 且无频控的实现只会
// 让「恢复短信」看起来比实际更近。恢复时必须重写发送、频控与角色校验。

// PasswordLogin 密码登录（仅管理后台）。
//
// v1 密码登录只允许 admin 角色：家长/老师走各自小程序的微信登录，预建账号
// 不持有可用密码（#174）。先校验密码再判角色，避免把「这个手机号是不是
// 管理后台账号」变成无需密码即可探测的信息。
func (s *AuthService) PasswordLogin(ctx context.Context, phone, password string) (*LoginResult, error) {
	user, err := s.userRepo.FindByPhone(ctx, phone)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("USER_NOT_FOUND")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("PASSWORD_INCORRECT")
	}

	roles, err := s.userRoleRepo.FindByUserID(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	if !hasAdminRole(roles) {
		return nil, fmt.Errorf("NOT_ADMIN")
	}

	return s.issueAdminTokens(ctx, user, roles)
}

// WeChatLoginRequest 是 POST /auth/login/wechat 的输入。
//
// Role 仅用于客户端一致性校验：角色与凭据一律取自 app_id 对应的客户端注册表，
// 请求体里的 role 不能改变授权范围。
type WeChatLoginRequest struct {
	AppID      string
	Code       string
	InviteCode string
	Role       string
}

// WeChatLogin 小程序登录（#175）：app_id → 客户端 → 用该角色凭据换 openid。
func (s *AuthService) WeChatLogin(ctx context.Context, req WeChatLoginRequest) (*LoginResult, error) {
	if s.clients == nil || s.wx == nil || s.identities == nil {
		return nil, fmt.Errorf("wechat login not configured")
	}

	client, err := s.clients.ResolveEnabledByAppID(ctx, req.AppID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrClientUnavailable, err)
	}

	// body 里的 role 只做一致性校验：授权角色永远来自客户端注册表。
	if req.Role != "" && req.Role != client.Role {
		return nil, fmt.Errorf("%w: body role=%q client role=%q", ErrClientRoleMismatch, req.Role, client.Role)
	}
	// 邀请码是老师端绑定通道；家长端携带视为客户端行为异常。
	if req.InviteCode != "" && client.Role != domain.ClientRoleTeacher {
		return nil, fmt.Errorf("%w: role=%q", ErrInviteCodeNotAllowed, client.Role)
	}

	sess, err := s.wx.Code2Session(ctx, req.Code, client.Role)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrWeChatLoginFailed, err)
	}
	if sess == nil || sess.OpenID == "" {
		return nil, fmt.Errorf("%w: empty openid", ErrWeChatLoginFailed)
	}

	// (app_id, open_id) 是身份的唯一来源；没有身份就建「待绑定账号 + 身份」，
	// 但不自动授予任何角色，也不把账号挂进任何园所。
	user, err := s.identities.FindOrCreatePendingUser(ctx, client.AppID, sess.OpenID, sess.UnionID)
	if err != nil {
		return nil, fmt.Errorf("resolve client identity: %w", err)
	}

	roles, err := s.userRoleRepo.FindByUserID(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	// 同一账号可同时是家长和老师：token 只能携带当前客户端角色族的角色，
	// 否则家长端会拿到老师权限，园所也会串到另一端（#175）。
	// 通道同样参与过滤：共用客户端不把用户挂进 dedicated/group 的园（#169）。
	roles, _, err = s.resolveClientMemberRoles(ctx, roles, client)
	if err != nil {
		return nil, err
	}

	return s.issueClientTokens(ctx, user, roles, client, sess.OpenID)
}

// filterRolesByFamily 保留与当前客户端角色一致的授权行。
func filterRolesByFamily(roles []domain.UserRole, role string) []domain.UserRole {
	filtered := make([]domain.UserRole, 0, len(roles))
	for _, r := range roles {
		if r.Role == role {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// loadKindergartenMap 批量取角色关联的园所（名称与通道）并按 id 建表。
// 没有任何园所关联时返回空表，不触发仓储调用。
func (s *AuthService) loadKindergartenMap(ctx context.Context, roles []domain.UserRole) (map[uint64]*domain.Kindergarten, error) {
	ids := make([]uint64, 0, len(roles))
	seen := make(map[uint64]bool, len(roles))
	for _, r := range roles {
		if r.KindergartenID == nil || seen[*r.KindergartenID] {
			continue
		}
		seen[*r.KindergartenID] = true
		ids = append(ids, *r.KindergartenID)
	}
	if len(ids) == 0 {
		return map[uint64]*domain.Kindergarten{}, nil
	}
	if s.kindergartens == nil {
		return nil, errors.New("kindergarten membership repository not configured")
	}
	rows, err := s.kindergartens.FindByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("查询成员园所失败：%w", err)
	}
	byID := make(map[uint64]*domain.Kindergarten, len(rows))
	for i := range rows {
		kg := rows[i]
		byID[kg.ID] = &kg
	}
	return byID, nil
}

// filterClientRoles 是「客户端角色族 + 园所通道」过滤的唯一实现（#169）：
// 没有园所关联、园所不存在/已删除、通道不匹配的角色一律不进入小程序会话。
func filterClientRoles(roles []domain.UserRole, client *domain.Client, byID map[uint64]*domain.Kindergarten) []domain.UserRole {
	filtered := make([]domain.UserRole, 0, len(roles))
	for _, r := range roles {
		if r.KindergartenID == nil {
			continue
		}
		if !client.ServesKindergarten(byID[*r.KindergartenID]) {
			continue
		}
		filtered = append(filtered, r)
	}
	return filtered
}

// resolveClientMemberRoles 先按角色族裁剪，再按客户端作用域过滤通道。
func (s *AuthService) resolveClientMemberRoles(ctx context.Context, roles []domain.UserRole, client *domain.Client) ([]domain.UserRole, map[uint64]*domain.Kindergarten, error) {
	roles = filterRolesByFamily(roles, client.Role)
	byID, err := s.loadKindergartenMap(ctx, roles)
	if err != nil {
		return nil, nil, err
	}
	return filterClientRoles(roles, client, byID), byID, nil
}

// KindergartenMembership 是「我的园所」的一项：暴露 id/名称/当前客户端角色/公开入口码。
// public_code 让家长端能把扫到的 kg_<code> 匹配到成员园（#176）；公开码本就印在对外物料上。
type KindergartenMembership struct {
	KindergartenID uint64 `json:"kindergarten_id"`
	Name           string `json:"name"`
	Role           string `json:"role"`
	PublicCode     string `json:"public_code"`
}

// ErrClientScopeRequired 表示端点只接受小程序客户端凭证（带 app_id）。
// 管理端密码登录不进入 clients 表，不能调用园所列表或切换。
var ErrClientScopeRequired = errors.New("CLIENT_SCOPE_REQUIRED")

// ErrKindergartenNotMember 目标园不在「当前客户端角色族 ∩ 园所通道」过滤后的
// 成员列表里（含角色族不匹配、通道不匹配、无成员关系三种情况）。
var ErrKindergartenNotMember = errors.New("KINDERGARTEN_NOT_MEMBER")

// ListMyKindergartens 返回当前客户端可见的成员园所（#169）。
// 过滤依据是真实园所行：角色族取当前客户端，通道走 ServesKindergarten。
func (s *AuthService) ListMyKindergartens(ctx context.Context, userID uint64, claims *auth.AccessClaims) ([]KindergartenMembership, error) {
	if claims == nil || claims.AppID == "" {
		return nil, ErrClientScopeRequired
	}
	client, err := s.clients.ResolveEnabledByAppID(ctx, claims.AppID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrClientUnavailable, err)
	}
	roles, err := s.userRoleRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	roles, byID, err := s.resolveClientMemberRoles(ctx, roles, client)
	if err != nil {
		return nil, err
	}

	items := make([]KindergartenMembership, 0, len(roles))
	seen := make(map[uint64]bool, len(roles))
	for _, r := range roles {
		kgID := *r.KindergartenID
		if seen[kgID] {
			continue
		}
		seen[kgID] = true
		items = append(items, KindergartenMembership{
			KindergartenID: kgID,
			Name:           byID[kgID].Name,
			Role:           r.Role,
			PublicCode:     byID[kgID].PublicCode,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].KindergartenID < items[j].KindergartenID
	})
	return items, nil
}

// SwitchKindergarten 切换当前客户端的单园会话（#169）。
//
// 目标园必须仍在过滤后的成员列表里；旧 refresh 通过 GETDEL 原子消费，
// 成功后才换发绑定目标园的 access + refresh，避免切换与续期并发时各换一套凭证。
func (s *AuthService) SwitchKindergarten(ctx context.Context, userID uint64, claims *auth.AccessClaims, targetKindergartenID uint64, refreshToken string) (*LoginResult, error) {
	if claims == nil || claims.AppID == "" {
		return nil, ErrClientScopeRequired
	}
	if s.revoker == nil || s.clients == nil {
		return nil, fmt.Errorf("%w: auth dependencies not configured", ErrAuthStorageUnavailable)
	}
	revoked, err := s.revoker.Revoked(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAuthStorageUnavailable, err)
	}
	if revoked {
		return nil, ErrTokenRevoked
	}

	// 先消费再校验：旧 refresh 一次性使用，校验失败也不允许重放。
	value, err := auth.ConsumeRefresh(ctx, s.rdb, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token")
	}
	scope, err := auth.DecodeRefreshScope(value)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}
	// 切换同样消耗 refresh：账号级失效后旧凭证不得再换出目标园会话（#182）。
	if err := s.ensureTenantEpochCurrent(ctx, scope.UID, scope.IssuedAt); err != nil {
		return nil, err
	}
	// 凭证必须属于同一个账号、同一个客户端，不能拿别人的 refresh 切自己的会话。
	if scope.UID != userID || scope.AppID != claims.AppID {
		return nil, fmt.Errorf("invalid refresh token: scope mismatch")
	}

	client, err := s.clients.ResolveEnabledByAppID(ctx, scope.AppID)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}
	if client.ScopeKind != scope.ScopeKind || client.ScopeID != scope.ScopeID ||
		client.ScopeKind != claims.ClientScopeKind || client.ScopeID != claims.ClientScopeID {
		return nil, fmt.Errorf("invalid refresh token: client scope changed")
	}

	user, err := s.userRepo.FindByID(ctx, scope.UID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}
	// 解绑（#172）后旧 refresh 不能再换出新凭证。
	if err := s.ensureSessionIdentityActive(ctx, user.ID, client.AppID); err != nil {
		return nil, err
	}

	roles, err := s.userRoleRepo.FindByUserID(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	roles, _, err = s.resolveClientMemberRoles(ctx, roles, client)
	if err != nil {
		return nil, err
	}
	scopedRoles := filterRolesByKindergarten(roles, targetKindergartenID)
	if len(scopedRoles) == 0 {
		return nil, ErrKindergartenNotMember
	}
	openID, err := s.clientOpenID(ctx, user.ID, client)
	if err != nil {
		return nil, err
	}
	return s.issueTokensFor(ctx, user, scopedRoles, client, openID)
}

// hasAdminRole 密码登录只服务管理后台（#174）：家长/老师账号即使有密码
// 也不能走密码通道换取管理端 token。
func hasAdminRole(roles []domain.UserRole) bool {
	for _, r := range roles {
		if r.Role == "admin" {
			return true
		}
	}
	return false
}

// ErrTokenRevoked 表示 token 已被登出吊销；ErrAuthStorageUnavailable 表示吊销存储
// （Redis）不可用。两者由 handler 映射成 401 / 503，不能混进「token 无效」。
var (
	ErrTokenRevoked           = errors.New("TOKEN_REVOKED")
	ErrAuthStorageUnavailable = errors.New("AUTH_STORAGE_UNAVAILABLE")
	// ErrKindergartenSuspended 园所被平台方停用，不签发凭证。
	ErrKindergartenSuspended = errors.New("KINDERGARTEN_SUSPENDED")
	// ErrClientUnavailable 客户端未注册或已停用：不得换取 token，也不得建号。
	ErrClientUnavailable = errors.New("CLIENT_UNAVAILABLE")
	// ErrClientRoleMismatch 请求体自报角色与客户端注册的角色不一致。
	ErrClientRoleMismatch = errors.New("CLIENT_ROLE_MISMATCH")
	// ErrInviteCodeNotAllowed 非老师端客户端携带 invite_code。
	ErrInviteCodeNotAllowed = errors.New("INVITE_CODE_NOT_ALLOWED")
	// ErrWeChatLoginFailed 微信 code2session 失败或未返回 openid。
	ErrWeChatLoginFailed = errors.New("WECHAT_LOGIN_FAILED")
	// ErrAdminKindergartenRequired 管理端账号必须绑定一个园所。
	ErrAdminKindergartenRequired = errors.New("ADMIN_KINDERGARTEN_REQUIRED")
	// ErrAdminKindergartenConflict 同一手机号在多个园所拥有管理员身份。
	ErrAdminKindergartenConflict = errors.New("ADMIN_KINDERGARTEN_CONFLICT")
	// ErrIdentityUnbound 续期凭证对应的客户端身份已被解绑（#172）。
	// 解绑后旧会话不能再续期，否则「换微信」形同虚设。
	ErrIdentityUnbound = errors.New("IDENTITY_UNBOUND")
)

// RefreshToken 刷新令牌。
//
// 刷新不是「重新登录」：必须沿用签发时的客户端与园所，只换新 access + refresh。
// 旧版本存的裸 userID 会在这里被拒绝（#169）。
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*LoginResult, error) {
	if s.revoker == nil {
		return nil, fmt.Errorf("%w: token revoker not configured", ErrAuthStorageUnavailable)
	}
	revoked, err := s.revoker.Revoked(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAuthStorageUnavailable, err)
	}
	if revoked {
		return nil, ErrTokenRevoked
	}

	value, err := auth.ConsumeRefresh(ctx, s.rdb, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token")
	}
	scope, err := auth.DecodeRefreshScope(value)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}
	// 账号级失效（#182）：改密/重置/删除后，此前签发的 refresh 一律作废。
	// 与 AuthJWT 同源比较；存储故障失败关闭。
	if err := s.ensureTenantEpochCurrent(ctx, scope.UID, scope.IssuedAt); err != nil {
		return nil, err
	}

	user, err := s.userRepo.FindByID(ctx, scope.UID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	roles, err := s.userRoleRepo.FindByUserID(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	// 管理端（admin_web）：沿用 #169 的 JSON 作用域，但续期时必须复核
	// 「恰好一个园所」与「口令未变」——改密即吊销旧 refresh（#166）。
	if scope.AppID == "" {
		if scope.ScopeKind != auth.RefreshScopeAdminWeb {
			return nil, fmt.Errorf("invalid refresh scope")
		}
		if scope.PasswordVersion == "" || scope.PasswordVersion != auth.PasswordFingerprint(user.PasswordHash) {
			return nil, fmt.Errorf("invalid refresh token: password changed")
		}
		adminRoles, kgID, err := adminSessionRoles(roles)
		if err != nil {
			return nil, err
		}
		if scope.KindergartenID != kgID {
			return nil, fmt.Errorf("invalid refresh token: kindergarten changed")
		}
		return s.issueTokensFor(ctx, user, adminRoles, nil, "")
	}

	if s.clients == nil {
		return nil, fmt.Errorf("invalid refresh token: client resolver not configured")
	}
	client, err := s.clients.ResolveEnabledByAppID(ctx, scope.AppID)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}
	// 客户端作用域变了（如 AppID 行被另一条替换），旧 refresh 不能滑过去。
	if client.ScopeKind != scope.ScopeKind || client.ScopeID != scope.ScopeID {
		return nil, fmt.Errorf("invalid refresh token: client scope changed")
	}
	// 身份只有在仍然存在且属于该账号时才能续期（#172）。解绑删的是
	// user_identities 行，若这里不复核，被解绑的微信还能拿旧 refresh 续出新 access。
	if err := s.ensureSessionIdentityActive(ctx, user.ID, client.AppID); err != nil {
		return nil, err
	}

	roles, _, err = s.resolveClientMemberRoles(ctx, roles, client)
	if err != nil {
		return nil, err
	}
	scopedRoles := filterRolesByKindergarten(roles, scope.KindergartenID)
	// 已绑定用户必须有且只续签签发时的那个园；待绑定用户必须仍然没有成员关系。
	if (scope.KindergartenID == 0 && len(roles) > 0) || (scope.KindergartenID != 0 && len(scopedRoles) == 0) {
		return nil, fmt.Errorf("invalid refresh token: membership changed")
	}

	openID, err := s.clientOpenID(ctx, user.ID, client)
	if err != nil {
		return nil, err
	}
	return s.issueTokensFor(ctx, user, scopedRoles, client, openID)
}

// ensureSessionIdentityActive 复核会话对应的客户端身份仍存在（#172）。
// 解绑后身份行已被物理删除，旧 refresh / 切换不能再换出新凭证；
// 否则「换微信」只对新登录生效，被解绑的微信仍可无限续期。
func (s *AuthService) ensureSessionIdentityActive(ctx context.Context, userID uint64, appID string) error {
	if s.identities == nil {
		return fmt.Errorf("identity store not configured")
	}
	identity, err := s.identities.FindByUserAndApp(ctx, userID, appID)
	if err != nil {
		return err
	}
	if identity == nil {
		return ErrIdentityUnbound
	}
	return nil
}

// filterRolesByKindergarten 只保留目标园所的角色行；kgID=0 返回空
// （待绑定会话没有园所，也不允许凭空获得角色）。
func filterRolesByKindergarten(roles []domain.UserRole, kgID uint64) []domain.UserRole {
	if kgID == 0 {
		return nil
	}
	filtered := make([]domain.UserRole, 0, len(roles))
	for _, r := range roles {
		if r.KindergartenID != nil && *r.KindergartenID == kgID {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// Logout 退出登录：吊销当前 access token；请求带 refresh token 时一并吊销。
func (s *AuthService) Logout(ctx context.Context, tokenStr, refreshToken string) error {
	if s.revoker == nil {
		return fmt.Errorf("token revoker not configured")
	}
	claims, err := auth.ParseAccess(s.secret, tokenStr)
	if err != nil {
		// 中间件已校验过签名，走到这里说明调用路径异常；没有可吊销的声明，不谎报失败。
		return nil
	}

	if claims.ID != "" && claims.ExpiresAt != nil {
		ttl := time.Until(claims.ExpiresAt.Time) + time.Second
		if err := s.revoker.Revoke(ctx, claims.ID, ttl); err != nil {
			return fmt.Errorf("revoke access token: %w", err)
		}
	}

	if refreshToken != "" {
		ttl, err := auth.RefreshRemainingTTL(ctx, s.rdb, refreshToken)
		if err != nil {
			return fmt.Errorf("read refresh token ttl: %w", err)
		}
		// 已经消费/过期的 refresh token 没有可吊销的会话，不必写黑名单。
		if ttl > 0 {
			if err := s.revoker.Revoke(ctx, refreshToken, ttl+time.Second); err != nil {
				return fmt.Errorf("revoke refresh token: %w", err)
			}
		}
	}
	return nil
}

// TokenRevoker 暴露吊销存储给路由装配：AuthJWT 必须与签发方共用同一份黑名单。
func (s *AuthService) TokenRevoker() auth.TokenRevoker {
	if s == nil {
		return nil
	}
	return s.revoker
}

// TenantEpochStore 暴露园所账号级失效存储给路由与中间件装配（#182）。
// AuthJWT、续期与新会话的 epoch 比较必须使用同一份存储。
func (s *AuthService) TenantEpochStore() auth.TenantEpochStore {
	if s == nil {
		return nil
	}
	return s.tenantEpochs
}

// bumpTenantEpoch 推进账号级失效水位（#182）。改密 / 重置 / 删除管理员都必须
// 先成功推进它再改库；写失败由调用方上抛，最终映射成 503（失败关闭）。
//
// epoch 写 Redis、不参与调用方的事务：若其后改库失败，账号会被多登出一次，
// 而口令/角色未变。这个方向只会让用户重新登录，不会留下「凭据已换、旧会话
// 仍有效」的窗口，因此可接受，也无需补偿写。
func (s *AuthService) bumpTenantEpoch(ctx context.Context, userID uint64) error {
	if s.tenantEpochs == nil {
		return fmt.Errorf("tenant epoch store not configured")
	}
	return s.tenantEpochs.BumpTenantEpoch(ctx, fmt.Sprintf("%d", userID), time.Now())
}

// ensureTenantEpochCurrent 校验续期凭证签发于账号级失效水位之后（#182）。
// 旧载荷 IssuedAt=0 在没有任何 epoch 时仍按原行为可用；一旦账号被改密/重置/
// 删除，epoch 必然晚于 1970，旧载荷随之失效。存储故障失败关闭。
//
// 比较是「严格早于」：epoch 为秒级精度（与 JWT iat 对齐），iat == epoch 的凭证
// 仍放行，失效动作与签发落在同一秒时旧凭证最多还能再用不到 1 秒。
//
// 调用点（Refresh / SwitchKindergarten）都先 GETDEL 消费 refresh 再到这里校验，
// 所以一旦 epoch 读失败或判定失效，这次 refresh 已被烧掉：重试同一个 refresh
// 只会得到 401，客户端必须重新登录。这是有意的失败关闭——不允许校验不过的
// refresh 重放。
func (s *AuthService) ensureTenantEpochCurrent(ctx context.Context, userID uint64, issuedAt int64) error {
	if s.tenantEpochs == nil {
		return fmt.Errorf("%w: tenant epoch store not configured", ErrAuthStorageUnavailable)
	}
	epoch, err := s.tenantEpochs.TenantEpoch(ctx, fmt.Sprintf("%d", userID))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrAuthStorageUnavailable, err)
	}
	if time.Unix(issuedAt, 0).Before(epoch) {
		return ErrTokenRevoked
	}
	return nil
}

func maskPhone(phone string) string {
	if len(phone) < 7 {
		return phone
	}
	return phone[:3] + "****" + phone[len(phone)-4:]
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func (s *AuthService) GetUserInfo(ctx context.Context, userID uint64, scope RoleScope) (*UserInfo, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}
	info, err := s.buildUserInfo(ctx, user, scope)
	if err != nil {
		return nil, err
	}
	info.Phone = maskPhone(user.Phone)
	return info, nil
}

type ProfileInfo struct {
	ID        uint64     `json:"id"`
	Phone     string     `json:"phone"`
	Nickname  string     `json:"nickname"`
	Avatar    string     `json:"avatar"`
	Gender    int        `json:"gender"`
	Roles     []RoleInfo `json:"roles"`
	CreatedAt time.Time  `json:"created_at"`
}

func (s *AuthService) GetProfile(ctx context.Context, userID uint64, scope RoleScope) (*ProfileInfo, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	roles, err := s.userRoleRepo.FindByUserID(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	// 资料与登录响应同源：小程序凭证只能看到当前客户端角色族（#175）。
	roles = scope.filter(roles)

	roleInfos := make([]RoleInfo, 0, len(roles))
	for _, r := range roles {
		roleInfos = append(roleInfos, RoleInfo{
			Role:             r.Role,
			KindergartenID:   r.KindergartenID,
			KindergartenName: r.KindergartenName,
			ClassID:          r.ClassID,
			ClassName:        r.ClassName,
			Title:            r.Title,
			TitleLabel:       r.TitleLabel,
			AdminRole:        r.AdminRole,
		})
	}
	if err := s.fillAdminPermissions(ctx, user.ID, roleInfos); err != nil {
		return nil, err
	}

	return &ProfileInfo{
		ID:        user.ID,
		Phone:     maskPhone(user.Phone),
		Nickname:  user.Nickname,
		Avatar:    user.Avatar,
		Gender:    user.Gender,
		Roles:     roleInfos,
		CreatedAt: user.CreatedAt,
	}, nil
}

// fillAdminPermissions 为 admin 角色条目注入权限码（wire-admin-permissions design D6）。
// 解析失败返回错误——profile 是管理端菜单与路由守卫的数据来源，
// 宁可请求失败，也不静默降级为「无权限」造成界面撒谎。
func (s *AuthService) fillAdminPermissions(ctx context.Context, userID uint64, infos []RoleInfo) error {
	if s.adminPerms == nil {
		return nil
	}
	for i := range infos {
		if infos[i].Role != "admin" || infos[i].KindergartenID == nil {
			continue
		}
		codes, _, err := s.adminPerms.ResolveAdminCodes(ctx, *infos[i].KindergartenID, userID)
		if err != nil {
			return fmt.Errorf("resolve admin permissions: %w", err)
		}
		infos[i].Permissions = codes
	}
	return nil
}

func (s *AuthService) UpdateProfile(ctx context.Context, userID uint64, nickname string, gender int, scope RoleScope) (*UserInfo, error) {
	fields := map[string]any{}
	if nickname != "" {
		fields["nickname"] = nickname
	}
	if gender >= 0 {
		fields["gender"] = gender
	}
	if len(fields) > 0 {
		if err := s.userRepo.UpdateFields(ctx, userID, fields); err != nil {
			return nil, err
		}
	}
	return s.GetUserInfo(ctx, userID, scope)
}

func (s *AuthService) UpdateAvatar(ctx context.Context, userID uint64, avatar string) (string, error) {
	avatar = oss.NormalizeRef(avatar)
	if err := s.userRepo.UpdateFields(ctx, userID, map[string]any{"avatar": avatar}); err != nil {
		return "", err
	}
	return avatar, nil
}

// ExtractRoleNames 从 user_roles 表中提取角色名列表，供 JWT claims 使用
func ExtractRoleNames(roles []domain.UserRole) []string {
	seen := make(map[string]bool)
	var names []string
	for _, r := range roles {
		if !seen[r.Role] {
			seen[r.Role] = true
			names = append(names, r.Role)
		}
	}
	return names
}

// MaskPhone 公开的手机号脱敏方法
func MaskPhone(phone string) string {
	return maskPhone(phone)
}

// SplitRoles 兼容旧的 user.Roles 字段（逗号分隔）
func SplitRoles(roles string) []string {
	if roles == "" {
		return []string{"parent"}
	}
	return strings.Split(roles, ",")
}
