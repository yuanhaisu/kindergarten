package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/storage/oss"
	"kindergarten-server/pkg/log"
)

// AuthUserRepository AuthService 依赖的用户仓库接口（由 *repository.UserRepo 隐式实现）
type AuthUserRepository interface {
	FindByPhone(ctx context.Context, phone string) (*domain.User, error)
	FindByUnionID(ctx context.Context, unionID string) (*domain.User, error)
	FindOtherByUnionID(ctx context.Context, unionID string, excludeID uint64) (*domain.User, error)
	FindOtherByPhone(ctx context.Context, phone string, excludeID uint64) (*domain.User, error)
	FindByOpenID(ctx context.Context, openID string) (*domain.User, error)
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
	rdb            *redis.Client
	revoker        auth.TokenRevoker
	secret         []byte
	atTTL          time.Duration
	rtTTL          time.Duration
	consentRepo    ConsentRepository
	consentEnforce bool
}

func NewAuthService(
	userRepo AuthUserRepository,
	userRoleRepo AuthRoleRepository,
	wx WeChatAPI,
	clients ClientResolver,
	identities UserIdentityRepository,
	adminPerms AdminPermissionResolver,
	kgStatus KindergartenStatusChecker,
	rdb *redis.Client,
	secret []byte,
	atTTL time.Duration,
	rtTTL time.Duration,
	consentRepo ConsentRepository,
	consentEnforce bool,
) *AuthService {
	return &AuthService{
		userRepo:       userRepo,
		userRoleRepo:   userRoleRepo,
		wx:             wx,
		clients:        clients,
		identities:     identities,
		adminPerms:     adminPerms,
		kgStatus:       kgStatus,
		rdb:            rdb,
		revoker:        auth.NewTokenRevoker(rdb),
		secret:         secret,
		atTTL:          atTTL,
		rtTTL:          rtTTL,
		consentRepo:    consentRepo,
		consentEnforce: consentEnforce,
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

// RoleScope 限定一次请求可见的角色族（#175）。
//
// 小程序凭证（带 app_id）只允许看到当前客户端角色族的角色，且
// must_change_password 只属于管理端密码登录，对小程序凭证不生效。
// zero value 表示不限制：管理端、平台方与旧凭证行为不变。
type RoleScope struct {
	clientScoped bool
	allowed      []string
}

// RoleScopeFromClaims 由访问凭证推导角色可见范围。
// app_id 存在即视为小程序客户端凭证；允许集合取凭证里的角色，
// 因为签发时已经做过角色族裁剪。
func RoleScopeFromClaims(claims *auth.AccessClaims) RoleScope {
	if claims == nil || claims.AppID == "" {
		return RoleScope{}
	}
	return RoleScope{clientScoped: true, allowed: claims.Roles}
}

// clientRoleScope 是登录路径的作用域：roles 已按客户端角色族裁剪过。
func clientRoleScope(roleNames []string) RoleScope {
	return RoleScope{clientScoped: true, allowed: roleNames}
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
		if s.Allows(r.Role) {
			filtered = append(filtered, r)
		}
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
	return s.userRepo.UpdateFields(ctx, userID, map[string]any{
		"password_hash":        hash,
		"must_change_password": false,
	})
}

// issueTokens 签发 access + refresh；access claims 含 kindergarten_id（由 user_roles 推导）
func (s *AuthService) issueTokens(ctx context.Context, user *domain.User) (*LoginResult, error) {
	roles, err := s.userRoleRepo.FindByUserID(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	return s.issueTokensFromRoles(ctx, user, roles)
}

func (s *AuthService) issueTokensFromRoles(ctx context.Context, user *domain.User, roles []domain.UserRole) (*LoginResult, error) {
	return s.issueTokensFor(ctx, user, roles, nil)
}

// issueClientTokens 签发绑定小程序客户端的凭证：claims 带 app_id 与作用域，
// 角色只来自当前客户端注册表（#175）。
func (s *AuthService) issueClientTokens(ctx context.Context, user *domain.User, roles []domain.UserRole, client *domain.Client) (*LoginResult, error) {
	return s.issueTokensFor(ctx, user, roles, client)
}

// issueTokensFor 是签发的唯一实现。client 为 nil 时是管理端/刷新路径（无 app_id），
// 非 nil 时必须走 SignClientAccess，否则 token 无法回查客户端作用域。
func (s *AuthService) issueTokensFor(ctx context.Context, user *domain.User, roles []domain.UserRole, client *domain.Client) (*LoginResult, error) {
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
	if client != nil {
		accessToken, err = auth.SignClientAccess(s.secret, uid, roleNames, kgID, auth.ClientScope{
			AppID: client.AppID,
			Kind:  client.ScopeKind,
			ID:    client.ScopeID,
		}, s.atTTL)
	} else {
		accessToken, err = auth.SignAccess(s.secret, uid, roleNames, kgID, s.atTTL)
	}
	if err != nil {
		return nil, err
	}

	jti := uuid.NewString()
	if err := auth.SaveRefresh(ctx, s.rdb, jti, uid, s.rtTTL); err != nil {
		return nil, err
	}

	// 响应里的角色必须与 token 一致：小程序路径用裁剪后的角色族，
	// 其余路径保持全量（#175）。
	scope := RoleScope{}
	if client != nil {
		scope = clientRoleScope(roleNames)
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

// SendSMS 发送短信验证码（实际短信发送需对接第三方，此处仅存 Redis）。
//
// v1 未在 HTTP 层暴露（#162）；恢复前必须先补齐真实发送、频控与角色校验。
func (s *AuthService) SendSMS(ctx context.Context, phone, scene string) (int, error) {
	key := fmt.Sprintf("sms:%s:%s", scene, phone)

	ttl, err := s.rdb.TTL(ctx, key).Result()
	if err == nil && ttl > 4*time.Minute {
		return 0, fmt.Errorf("SMS_TOO_FREQUENT")
	}

	code := generateSMSCode()
	expire := 5 * time.Minute
	if err := s.rdb.Set(ctx, key, code, expire).Err(); err != nil {
		return 0, err
	}

	log.L(ctx).Info("sms code generated", "phone", maskPhone(phone), "scene", scene)
	// TODO: 对接实际短信服务商发送验证码
	return int(expire.Seconds()), nil
}

func (s *AuthService) verifySMSCode(ctx context.Context, phone, scene, code string) error {
	key := fmt.Sprintf("sms:%s:%s", scene, phone)
	stored, err := s.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return fmt.Errorf("SMS_CODE_EXPIRED")
	}
	if err != nil {
		return err
	}
	if stored != code {
		return fmt.Errorf("SMS_CODE_INVALID")
	}
	s.rdb.Del(ctx, key)
	return nil
}

// SMSLogin 手机号 + 验证码登录。
//
// v1 关闭（#162）：当前实现忽略 role 且无频控，未在 HTTP 层暴露；
// 恢复前必须补齐角色校验与限流。
func (s *AuthService) SMSLogin(ctx context.Context, phone, code, role string) (*LoginResult, error) {
	if err := s.verifySMSCode(ctx, phone, "login", code); err != nil {
		return nil, err
	}

	user, err := s.userRepo.FindByPhone(ctx, phone)
	if err != nil {
		return nil, err
	}
	if user == nil {
		user = &domain.User{Phone: phone}
		if err := s.userRepo.Create(ctx, user); err != nil {
			return nil, err
		}
		log.L(ctx).Info("new user registered via sms", "user_id", user.ID)
	}

	return s.issueTokens(ctx, user)
}

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

	return s.issueTokensFromRoles(ctx, user, roles)
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
	roles = filterRolesByFamily(roles, client.Role)

	return s.issueClientTokens(ctx, user, roles, client)
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
)

// RefreshToken 刷新令牌
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

	userIDStr, err := auth.ConsumeRefresh(ctx, s.rdb, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token")
	}

	var userID uint64
	if _, err := fmt.Sscanf(userIDStr, "%d", &userID); err != nil {
		return nil, fmt.Errorf("invalid user id in refresh token")
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	return s.issueTokens(ctx, user)
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

func maskPhone(phone string) string {
	if len(phone) < 7 {
		return phone
	}
	return phone[:3] + "****" + phone[len(phone)-4:]
}

func generateSMSCode() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	return fmt.Sprintf("%06d", n.Int64())
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
