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
	"gorm.io/gorm"

	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/domain"
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

// InvitationRepository AuthService 依赖的邀请仓库接口（由 *repository.TeacherInvitationRepo 隐式实现）
type InvitationRepository interface {
	FindByToken(ctx context.Context, token string) (*domain.TeacherInvitation, error)
	MarkUsed(ctx context.Context, id, userID uint64, openID, unionID string) error
}

// AdminPermissionResolver 解析管理端账号权限码（由 *AdminAuthzService 实现）。
// 声明在消费方，便于 AuthService 单测注入桩。
type AdminPermissionResolver interface {
	ResolveAdminCodes(ctx context.Context, kindergartenID, userID uint64) ([]string, bool, error)
}

type AuthService struct {
	db             *gorm.DB
	userRepo       AuthUserRepository
	userRoleRepo   AuthRoleRepository
	invitationRepo InvitationRepository
	adminPerms     AdminPermissionResolver
	rdb            *redis.Client
	revoker        auth.TokenRevoker
	wx             *auth.WeChatClient
	secret         []byte
	atTTL          time.Duration
	rtTTL          time.Duration
}

func NewAuthService(
	db *gorm.DB,
	userRepo AuthUserRepository,
	userRoleRepo AuthRoleRepository,
	invitationRepo InvitationRepository,
	adminPerms AdminPermissionResolver,
	rdb *redis.Client,
	wx *auth.WeChatClient,
	secret []byte,
	atTTL time.Duration,
	rtTTL time.Duration,
) *AuthService {
	return &AuthService{
		db:             db,
		userRepo:       userRepo,
		userRoleRepo:   userRoleRepo,
		invitationRepo: invitationRepo,
		adminPerms:     adminPerms,
		rdb:            rdb,
		revoker:        auth.NewTokenRevoker(rdb),
		wx:             wx,
		secret:         secret,
		atTTL:          atTTL,
		rtTTL:          rtTTL,
	}
}

type LoginResult struct {
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	User         *UserInfo `json:"user"`
}

type UserInfo struct {
	ID       uint64     `json:"id"`
	Phone    string     `json:"phone"`
	Nickname string     `json:"nickname"`
	Avatar   string     `json:"avatar"`
	Gender   int        `json:"gender"`
	Roles    []RoleInfo `json:"roles"`
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

func (s *AuthService) buildUserInfo(ctx context.Context, user *domain.User) (*UserInfo, error) {
	roles, err := s.userRoleRepo.FindByUserID(ctx, user.ID)
	if err != nil {
		return nil, err
	}

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
		Roles:    roleInfos,
	}, nil
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
	roleNames := roleNamesFromUserRoles(roles)
	kgID := kindergartenIDFromUserRoles(roles)

	uid := fmt.Sprintf("%d", user.ID)
	accessToken, err := auth.SignAccess(s.secret, uid, roleNames, kgID, s.atTTL)
	if err != nil {
		return nil, err
	}

	jti := uuid.NewString()
	if err := auth.SaveRefresh(ctx, s.rdb, jti, uid, s.rtTTL); err != nil {
		return nil, err
	}

	userInfo, err := s.buildUserInfo(ctx, user)
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		Token:        accessToken,
		RefreshToken: jti,
		ExpiresAt:    time.Now().Add(s.atTTL),
		User:         userInfo,
	}, nil
}

func roleNamesFromUserRoles(roles []domain.UserRole) []string {
	var names []string
	seen := make(map[string]bool)
	for _, r := range roles {
		if !seen[r.Role] {
			seen[r.Role] = true
			names = append(names, r.Role)
		}
	}
	if len(names) == 0 {
		names = append(names, "parent")
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

// SendSMS 发送短信验证码（实际短信发送需对接第三方，此处仅存 Redis）
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

// SMSLogin 手机号 + 验证码登录
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

// WeChatLogin 微信小程序登录。inviteCode 非空时通过邀请码绑定预创建的教师账号。
//
// role 决定用哪端的微信凭据换取会话——两端 AppID 不同，openid 也不同，
// 同一个人在两端会落两条 User（见 specs/wechat-login）。
// role 未配置凭据或未知时返回明确错误，绝不回退到另一端凭据。
func (s *AuthService) WeChatLogin(ctx context.Context, wxCode, role, inviteCode string) (*LoginResult, error) {
	sess, err := s.wx.Code2Session(ctx, wxCode, role)
	if err != nil {
		return nil, fmt.Errorf("wechat code2session: %w", err)
	}

	if inviteCode != "" {
		return s.weChatLoginWithInvite(ctx, sess, inviteCode)
	}

	user, err := s.resolveWeChatUser(ctx, sess)
	if err != nil {
		return nil, err
	}
	return s.issueTokens(ctx, user)
}

// resolveWeChatUser 按本端 openid 解析出 User，没有就新建一条。
//
// 两端 AppID 不同 ⇒ 同一自然人在两端的 openid 也不同 ⇒ 必然落两条 User，
// 彼此独立（见 specs/wechat-login）。跨端「是不是同一个人」由 unionid / 手机号
// 关联判定负责，这里绝不合并账号。
func (s *AuthService) resolveWeChatUser(ctx context.Context, sess *auth.Code2SessionResp) (*domain.User, error) {
	user, err := s.userRepo.FindByOpenID(ctx, sess.OpenID)
	if err != nil {
		return nil, err
	}
	if user != nil {
		return user, nil
	}

	user = &domain.User{
		OpenID:  sess.OpenID,
		UnionID: sess.UnionID,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}
	log.L(ctx).Info("new user registered via wechat", "user_id", user.ID)
	return user, nil
}

func (s *AuthService) weChatLoginWithInvite(ctx context.Context, sess *auth.Code2SessionResp, inviteCode string) (*LoginResult, error) {
	inv, err := s.invitationRepo.FindByToken(ctx, inviteCode)
	if err != nil {
		return nil, err
	}
	if inv == nil {
		return nil, fmt.Errorf("INVITE_CODE_INVALID")
	}

	user, err := s.userRepo.FindByID(ctx, inv.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("INVITE_USER_NOT_FOUND")
	}

	user.OpenID = sess.OpenID
	if sess.UnionID != "" {
		user.UnionID = sess.UnionID
	}
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	if err := s.invitationRepo.MarkUsed(ctx, inv.ID, user.ID, sess.OpenID, sess.UnionID); err != nil {
		return nil, err
	}

	log.L(ctx).Info("teacher bound via invite", "user_id", user.ID, "kindergarten_id", inv.KindergartenID)
	return s.issueTokens(ctx, user)
}

// BindInvite 已登录用户通过邀请码绑定预创建的教师账号。
// currentUserID 是微信登录自动创建的空白用户，inviteCode 对应的 User B 是管理员预创建的教师。
// 事务：将 User B 的所有角色/档案/带班关系迁移到 currentUser，删除 User B，签发新 JWT。
func (s *AuthService) BindInvite(ctx context.Context, currentUserID uint64, inviteCode string) (*LoginResult, error) {
	inv, err := s.invitationRepo.FindByToken(ctx, inviteCode)
	if err != nil {
		return nil, err
	}
	if inv == nil {
		return nil, fmt.Errorf("INVITE_CODE_INVALID")
	}

	targetUser, err := s.userRepo.FindByID(ctx, inv.UserID)
	if err != nil {
		return nil, err
	}
	if targetUser == nil {
		return nil, fmt.Errorf("INVITE_USER_NOT_FOUND")
	}

	if targetUser.ID == currentUserID {
		return nil, fmt.Errorf("INVITE_SELF_BIND")
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 迁移 user_roles
		if err := tx.Model(&domain.UserRole{}).Where("user_id = ?", targetUser.ID).
			Update("user_id", currentUserID).Error; err != nil {
			return err
		}
		// 迁移 teacher_profiles
		if err := tx.Model(&domain.TeacherProfile{}).Where("user_id = ?", targetUser.ID).
			Update("user_id", currentUserID).Error; err != nil {
			return err
		}
		// 迁移 class_teachers
		if err := tx.Model(&domain.ClassTeacher{}).Where("user_id = ?", targetUser.ID).
			Update("user_id", currentUserID).Error; err != nil {
			return err
		}
		// 更新邀请记录
		if err := tx.Model(&domain.TeacherInvitation{}).Where("id = ?", inv.ID).
			Updates(map[string]any{"user_id": currentUserID, "status": domain.TeacherInvitationUsed}).Error; err != nil {
			return err
		}
		// 软删除原目标用户
		if err := tx.Where("id = ?", targetUser.ID).Delete(&domain.User{}).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	log.L(ctx).Info("user bound via invite", "user_id", currentUserID, "target_user_id", targetUser.ID)
	return s.issueTokens(ctx, &domain.User{BaseModel: domain.BaseModel{ID: currentUserID}})
}

// PasswordLogin 密码登录（管理后台 + 家长端共用）
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

	return s.issueTokensFromRoles(ctx, user, roles)
}

// ErrTokenRevoked 表示 token 已被登出吊销；ErrAuthStorageUnavailable 表示吊销存储
// （Redis）不可用。两者由 handler 映射成 401 / 503，不能混进「token 无效」。
var (
	ErrTokenRevoked           = errors.New("TOKEN_REVOKED")
	ErrAuthStorageUnavailable = errors.New("AUTH_STORAGE_UNAVAILABLE")
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

func (s *AuthService) GetUserInfo(ctx context.Context, userID uint64) (*UserInfo, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}
	info, err := s.buildUserInfo(ctx, user)
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

func (s *AuthService) GetProfile(ctx context.Context, userID uint64) (*ProfileInfo, error) {
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

func (s *AuthService) UpdateProfile(ctx context.Context, userID uint64, nickname string, gender int) (*UserInfo, error) {
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
	return s.GetUserInfo(ctx, userID)
}

// BindParentPhone 家长微信登录后绑定手机号，匹配管理员预创建的家长账号。
// currentUserID: 微信登录自动创建的空白 User（仅有 OpenID）。
// 通过微信手机号 code 获取手机号 → 查找预创建的家长 User → 合并两个账号。
func (s *AuthService) BindParentPhone(ctx context.Context, currentUserID uint64, wxPhoneCode string) (*LoginResult, error) {
	phoneResp, err := s.wx.GetPhoneNumber(ctx, wxPhoneCode)
	if err != nil {
		return nil, fmt.Errorf("wechat getPhoneNumber: %w", err)
	}

	phone := phoneResp.PhoneInfo.PurePhoneNumber
	if phone == "" {
		return nil, fmt.Errorf("PHONE_NOT_FOUND")
	}

	existing, err := s.userRepo.FindByPhone(ctx, phone)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("PARENT_RECORD_NOT_FOUND")
	}

	// 如果预创建的 User 没有 OpenID，将当前微信用户的 OpenID 回填
	currentUser, err := s.userRepo.FindByID(ctx, currentUserID)
	if err != nil {
		return nil, err
	}
	if currentUser == nil {
		return nil, fmt.Errorf("USER_NOT_FOUND")
	}

	if existing.ID == currentUserID {
		// 同一个用户，直接签发 token
		log.L(ctx).Info("parent phone already bound", "user_id", currentUserID)
		return s.issueTokens(ctx, currentUser)
	}

	// 事务：将预创建 User 的 Role/Profile/Student 迁移到微信 User，更新预创建 User 的 OpenID
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 合并策略：将预创建 User(手机号) 的关联迁移到微信 User(OpenID)
		if err := tx.Model(&domain.UserRole{}).Where("user_id = ?", existing.ID).
			Update("user_id", currentUserID).Error; err != nil {
			return err
		}
		if err := tx.Model(&domain.ParentProfile{}).Where("user_id = ?", existing.ID).
			Update("user_id", currentUserID).Error; err != nil {
			return err
		}
		if err := tx.Model(&domain.StudentParent{}).Where("user_id = ?", existing.ID).
			Update("user_id", currentUserID).Error; err != nil {
			return err
		}
		// 将预创建 User 的 Phone 和密码迁移到微信 User
		fields := map[string]any{}
		if existing.Phone != "" {
			fields["phone"] = existing.Phone
		}
		if existing.PasswordHash != "" {
			fields["password_hash"] = existing.PasswordHash
		}
		if existing.Nickname != "" && currentUser.Nickname == "" {
			fields["nickname"] = existing.Nickname
		}
		if len(fields) > 0 {
			if err := tx.Model(&domain.User{}).Where("id = ?", currentUserID).Updates(fields).Error; err != nil {
				return err
			}
		}
		// 软删除预创建 User
		if err := tx.Where("id = ?", existing.ID).Delete(&domain.User{}).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	log.L(ctx).Info("parent bound via phone",
		"current_user_id", currentUserID,
		"merged_user_id", existing.ID,
		"phone", maskPhone(phone),
	)

	return s.issueTokens(ctx, &domain.User{BaseModel: domain.BaseModel{ID: currentUserID}})
}

func (s *AuthService) UpdateAvatar(ctx context.Context, userID uint64, avatar string) (string, error) {
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
