package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"kindergarten-server/internal/domain"
)

// 家长绑定手机号（#167）。
//
// 绑定 = 把当前待绑定账号的客户端身份（app_id + open_id）并入园所预创建的
// 家长账号。目标园所只用于「校验手机号确实被该园录入」和「绑定后的初始落点」，
// 不是角色边界：该账号在其他 shared 园所已有的成员关系仍然生效。
//
// 无 scene 绑定（不带 public_code）在全部 shared 园所里按手机号找预创建账号，
// 找到就绑定并返回全部成员关系；找不到只提示园所录入，绝不新建账号——
// 家长账号必须由园所先录入，否则绑定等于绕开园所建立关系。

var (
	// ErrBindClientNotParent 绑定只服务家长端客户端。
	ErrBindClientNotParent = errors.New("BIND_CLIENT_NOT_PARENT")
	// ErrBindPhoneIdentityMissing 当前 token 没有客户端身份（旧 token 或刷新路径签发）。
	// 调用方应重新走小程序登录。
	ErrBindPhoneIdentityMissing = errors.New("BIND_IDENTITY_MISSING")
	// ErrBindPhoneCodeInvalid 微信手机号授权 code 无效/已使用。
	ErrBindPhoneCodeInvalid = errors.New("BIND_PHONE_CODE_INVALID")
	// ErrBindKindergartenNotFound public_code / kindergarten_id 指不到有效园所。
	ErrBindKindergartenNotFound = errors.New("BIND_KINDERGARTEN_NOT_FOUND")
	// ErrBindPhoneChannelMismatch 目标园所不在当前客户端服务范围内（通道矩阵，见 #177）。
	ErrBindPhoneChannelMismatch = errors.New("BIND_CHANNEL_MISMATCH")
	// ErrBindPhoneNotEnrolled 目标范围内没有任何园所录入该手机号。
	ErrBindPhoneNotEnrolled = errors.New("BIND_PHONE_NOT_ENROLLED")
	// ErrBindPhoneAlreadyBound 预创建账号在同一客户端已绑定其他微信。
	ErrBindPhoneAlreadyBound = errors.New("BIND_PHONE_ALREADY_BOUND")
	// ErrBindPhoneRateLimited 绑定请求过于频繁（app_id + IP + 手机号）。
	ErrBindPhoneRateLimited = errors.New("BIND_PHONE_RATE_LIMITED")
)

// KindergartenLookup 绑定流程的园所解析（由 *repository.KindergartenRepo 实现）。
//
// 声明在消费方而不是直接依赖仓储，便于单测注入桩；方法集只包含绑定需要的三个读取。
type KindergartenLookup interface {
	FindByID(ctx context.Context, id uint64) (*domain.Kindergarten, error)
	FindByIDs(ctx context.Context, ids []uint64) ([]domain.Kindergarten, error)
	FindByPublicCode(ctx context.Context, code string) (*domain.Kindergarten, error)
}

// IdentityMerger 账号合并（由 *IdentityMergeService 实现）。
type IdentityMerger interface {
	Merge(ctx context.Context, req IdentityMergeRequest) (*IdentityMergeResult, error)
	EnsureParentProfile(ctx context.Context, userID, kindergartenID uint64) error
}

// BindPhoneRateLimiter 手机号维度的绑定限流（#174）。
//
// 请求体里只有一次性 wx_phone_code，中间件解析前拿不到手机号，所以完整的
// app_id + IP + 手机号维度只能在换到手机号之后判定。
type BindPhoneRateLimiter interface {
	Allow(ctx context.Context, ip, appID, phone string) bool
}

// ParentBindPhoneInput 是 POST /common/user/bind-parent-phone 的已校验输入。
// 不携带 user_id：待绑定账号由 (app_id, open_id) 定位，target 由手机号定位。
type ParentBindPhoneInput struct {
	AppID          string
	OpenID         string
	ClientIP       string
	WxPhoneCode    string
	PublicCode     string
	KindergartenID uint64
}

// ParentBoundKindergarten 绑定后可见的一个成员园所。
type ParentBoundKindergarten struct {
	KindergartenID   uint64 `json:"kindergarten_id"`
	KindergartenName string `json:"kindergarten_name"`
}

// ParentBindPhoneResult 绑定响应：内嵌登录结果（新 token 必须立即能进家长业务），
// 外加全部成员关系。成员列表不依赖 #169 的成员接口——绑定当下就要能展示。
type ParentBindPhoneResult struct {
	*LoginResult
	Kindergartens []ParentBoundKindergarten `json:"kindergartens"`
}

// BindParentPhone 把当前客户端身份绑定到「该手机号在园所预创建的家长账号」。
func (s *AuthService) BindParentPhone(ctx context.Context, in ParentBindPhoneInput) (*ParentBindPhoneResult, error) {
	if s.clients == nil || s.wx == nil || s.identities == nil || s.kgLookup == nil || s.merger == nil {
		return nil, fmt.Errorf("bind parent phone not configured")
	}

	client, err := s.clients.ResolveEnabledByAppID(ctx, in.AppID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrClientUnavailable, err)
	}
	if client.Role != domain.ClientRoleParent {
		return nil, fmt.Errorf("%w: app_id=%q role=%q", ErrBindClientNotParent, client.AppID, client.Role)
	}
	// openid 只来自 token：从请求体采信等于允许把任意微信绑进账号。
	if strings.TrimSpace(in.OpenID) == "" {
		return nil, ErrBindPhoneIdentityMissing
	}
	if strings.TrimSpace(in.WxPhoneCode) == "" {
		return nil, fmt.Errorf("%w: empty wx_phone_code", ErrBindPhoneCodeInvalid)
	}

	phoneResp, err := s.wx.GetPhoneNumber(ctx, in.WxPhoneCode)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBindPhoneCodeInvalid, err)
	}
	if phoneResp == nil {
		return nil, fmt.Errorf("%w: empty response", ErrBindPhoneCodeInvalid)
	}
	phone := strings.TrimSpace(phoneResp.PhoneInfo.PurePhoneNumber)
	if phone == "" {
		return nil, fmt.Errorf("%w: empty phone", ErrBindPhoneCodeInvalid)
	}

	if s.bindLimiter != nil && !s.bindLimiter.Allow(ctx, in.ClientIP, client.AppID, phone) {
		return nil, fmt.Errorf("%w: phone=%s", ErrBindPhoneRateLimited, maskPhone(phone))
	}

	targetUser, roles, kgs, err := s.resolveBindTarget(ctx, client, phone, in)
	if err != nil {
		return nil, err
	}
	landing := landingKindergartenID(roles)

	// 预创建账号是权威数据：方向固定 pending → target，保留 target 的角色、
	// 幼儿与档案，只补空缺；pending 被硬删。
	_, err = s.merger.Merge(ctx, IdentityMergeRequest{
		AppID:                       client.AppID,
		OpenID:                      in.OpenID,
		TargetUserID:                targetUser.ID,
		ParentProfileKindergartenID: landing,
	})
	switch {
	case err == nil:
	case errors.Is(err, ErrIdentityMergePhoneAlreadyBound):
		return nil, fmt.Errorf("%w: user_id=%d app_id=%s", ErrBindPhoneAlreadyBound, targetUser.ID, client.AppID)
	case errors.Is(err, ErrIdentityMergeIdentityNotFound), errors.Is(err, ErrIdentityMergePendingNotFound):
		return nil, fmt.Errorf("%w: %v", ErrBindPhoneIdentityMissing, err)
	default:
		return nil, fmt.Errorf("merge identity: %w", err)
	}
	// Merge 的幂等分支（身份已属于 target）会跳过档案创建，这里无条件兜底。
	if err := s.merger.EnsureParentProfile(ctx, targetUser.ID, landing); err != nil {
		return nil, fmt.Errorf("ensure parent profile: %w", err)
	}

	targetUser, err = s.userRepo.FindByID(ctx, targetUser.ID)
	if err != nil {
		return nil, err
	}
	if targetUser == nil {
		return nil, fmt.Errorf("%w: target user %d", ErrIdentityMergeTargetNotFound, landing)
	}

	login, err := s.issueClientTokens(ctx, targetUser, roles, client, in.OpenID)
	if err != nil {
		return nil, err
	}

	return &ParentBindPhoneResult{
		LoginResult:   login,
		Kindergartens: membershipsFromRoles(roles, kgs),
	}, nil
}

// resolveBindTarget 找到手机号对应的预创建家长账号，以及该账号在当前客户端
// 服务范围内的全部家长成员关系（landing 在前）。
func (s *AuthService) resolveBindTarget(
	ctx context.Context,
	client *domain.Client,
	phone string,
	in ParentBindPhoneInput,
) (*domain.User, []domain.UserRole, map[uint64]*domain.Kindergarten, error) {
	// 先解析目标园所与通道：无效入口码/不支持的通道与手机号是否录入无关，
	// 早于手机号判定返回，避免把「这个手机号有没有账号」泄露给伪造入口码。
	targetKG, hasTarget, err := s.explicitTarget(ctx, in)
	if err != nil {
		return nil, nil, nil, err
	}
	if hasTarget && !client.ServesKindergarten(targetKG) {
		return nil, nil, nil, fmt.Errorf("%w: app_id=%s scope=%s/%d kg=%d channel=%s",
			ErrBindPhoneChannelMismatch, client.AppID, client.ScopeKind, client.ScopeID, targetKG.ID, targetKG.AppChannel)
	}

	user, err := s.userRepo.FindByPhone(ctx, phone)
	if err != nil {
		return nil, nil, nil, err
	}
	if user == nil {
		return nil, nil, nil, fmt.Errorf("%w: phone not pre-created", ErrBindPhoneNotEnrolled)
	}

	allRoles, err := s.userRoleRepo.FindByUserID(ctx, user.ID)
	if err != nil {
		return nil, nil, nil, err
	}
	parentRoles := filterRolesByFamily(allRoles, domain.ClientRoleParent)
	if len(parentRoles) == 0 {
		return nil, nil, nil, fmt.Errorf("%w: user %d has no parent role", ErrBindPhoneNotEnrolled, user.ID)
	}

	if hasTarget {
		if roleInKindergarten(parentRoles, targetKG.ID) == nil {
			return nil, nil, nil, fmt.Errorf("%w: phone not in kindergarten %d", ErrBindPhoneNotEnrolled, targetKG.ID)
		}
		served, kgs, err := s.servedParentRoles(ctx, client, parentRoles)
		if err != nil {
			return nil, nil, nil, err
		}
		return user, orderRolesLandingFirst(served, targetKG.ID), kgs, nil
	}

	served, kgs, err := s.servedParentRoles(ctx, client, parentRoles)
	if err != nil {
		return nil, nil, nil, err
	}
	if len(served) == 0 {
		return nil, nil, nil, fmt.Errorf("%w: no shared kindergarten served by app_id=%s", ErrBindPhoneNotEnrolled, client.AppID)
	}
	// 无 scene：确定性落点（最小园所 ID），便于测试与产品口径一致。
	sort.SliceStable(served, func(i, j int) bool {
		return kindergartenIDOf(served[i]) < kindergartenIDOf(served[j])
	})
	return user, served, kgs, nil
}

// explicitTarget 解析 public_code / kindergarten_id 指定的目标园所。
// 两者都没带时返回 ok=false，走无 scene 绑定。
func (s *AuthService) explicitTarget(ctx context.Context, in ParentBindPhoneInput) (*domain.Kindergarten, bool, error) {
	if code := strings.TrimSpace(in.PublicCode); code != "" {
		kg, err := s.kgLookup.FindByPublicCode(ctx, code)
		if err != nil {
			return nil, false, err
		}
		if kg == nil {
			return nil, false, fmt.Errorf("%w: public_code=%q", ErrBindKindergartenNotFound, code)
		}
		return kg, true, nil
	}
	if in.KindergartenID != 0 {
		kg, err := s.kgLookup.FindByID(ctx, in.KindergartenID)
		if err != nil {
			return nil, false, err
		}
		if kg == nil {
			return nil, false, fmt.Errorf("%w: kindergarten_id=%d", ErrBindKindergartenNotFound, in.KindergartenID)
		}
		return kg, true, nil
	}
	return nil, false, nil
}

// servedParentRoles 过滤出「园所存在且当前客户端可服务」的家长角色。
func (s *AuthService) servedParentRoles(
	ctx context.Context,
	client *domain.Client,
	roles []domain.UserRole,
) ([]domain.UserRole, map[uint64]*domain.Kindergarten, error) {
	ids := make([]uint64, 0, len(roles))
	seen := make(map[uint64]bool, len(roles))
	for _, r := range roles {
		if r.KindergartenID == nil || seen[*r.KindergartenID] {
			continue
		}
		seen[*r.KindergartenID] = true
		ids = append(ids, *r.KindergartenID)
	}
	kgs, err := s.kgLookup.FindByIDs(ctx, ids)
	if err != nil {
		return nil, nil, err
	}
	byID := make(map[uint64]*domain.Kindergarten, len(kgs))
	for i := range kgs {
		byID[kgs[i].ID] = &kgs[i]
	}

	served := make([]domain.UserRole, 0, len(roles))
	for _, r := range roles {
		if r.KindergartenID == nil {
			continue
		}
		kg, ok := byID[*r.KindergartenID]
		if !ok || !client.ServesKindergarten(kg) {
			continue
		}
		served = append(served, r)
	}
	return served, byID, nil
}

func roleInKindergarten(roles []domain.UserRole, kindergartenID uint64) *domain.UserRole {
	for i := range roles {
		if roles[i].KindergartenID != nil && *roles[i].KindergartenID == kindergartenID {
			return &roles[i]
		}
	}
	return nil
}

// orderRolesLandingFirst 把目标园所的角色放在最前：kindergartenIDFromUserRoles
// 取第一条带园所的角色，token 的初始落点因此是绑定目标。
func orderRolesLandingFirst(roles []domain.UserRole, landingID uint64) []domain.UserRole {
	ordered := make([]domain.UserRole, 0, len(roles))
	for _, r := range roles {
		if r.KindergartenID != nil && *r.KindergartenID == landingID {
			ordered = append(ordered, r)
		}
	}
	for _, r := range roles {
		if r.KindergartenID != nil && *r.KindergartenID == landingID {
			continue
		}
		ordered = append(ordered, r)
	}
	return ordered
}

func landingKindergartenID(roles []domain.UserRole) uint64 {
	for _, r := range roles {
		if r.KindergartenID != nil {
			return *r.KindergartenID
		}
	}
	return 0
}

func kindergartenIDOf(role domain.UserRole) uint64 {
	if role.KindergartenID == nil {
		return 0
	}
	return *role.KindergartenID
}

// membershipsFromRoles 去重后给出成员园所；角色行里没有园所名时回落到园所表。
func membershipsFromRoles(roles []domain.UserRole, kgs map[uint64]*domain.Kindergarten) []ParentBoundKindergarten {
	out := make([]ParentBoundKindergarten, 0, len(roles))
	seen := make(map[uint64]bool, len(roles))
	for _, r := range roles {
		if r.KindergartenID == nil || seen[*r.KindergartenID] {
			continue
		}
		seen[*r.KindergartenID] = true
		name := r.KindergartenName
		if name == "" {
			if kg, ok := kgs[*r.KindergartenID]; ok {
				name = kg.Name
			}
		}
		out = append(out, ParentBoundKindergarten{
			KindergartenID:   *r.KindergartenID,
			KindergartenName: name,
		})
	}
	return out
}
