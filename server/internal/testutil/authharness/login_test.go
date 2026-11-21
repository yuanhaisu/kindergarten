package authharness_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/service"
	"kindergarten-server/internal/testutil/authharness"
)

// #175：新登录必须先查客户端注册表，再拿该客户端的凭据去换 openid。
//
// 这一组只锁「进门」行为：请求缺 app_id、客户端未注册、客户端已停用，都必须在
// code2session 之前拒绝。否则未注册客户端能拿到真实 openid 并建出账号，
// 停用客户端也能继续换 token——两条都是不可回滚的身份污染。

func TestWeChatLoginRequiresAppID(t *testing.T) {
	h := authharness.New(t)
	code := uniqueCode("no-app-id")
	h.WeChat.StubCode2Session(code, "openid-175-no-app-id", "")

	resp := h.PostJSON(t, "/api/v1/auth/login/wechat", map[string]string{
		"code": code,
		"role": "parent",
	})

	require.Equal(t, 422, resp.Status, string(resp.Body))
	assert.Equal(t, 42200, resp.EnvelopeCode(t), string(resp.Body))
	assert.Empty(t, h.WeChat.Calls(), "缺 app_id 时不得调用微信 code2session")
}

func TestWeChatLoginRejectsUnregisteredClient(t *testing.T) {
	h := authharness.New(t)
	code := uniqueCode("unknown-client")
	h.WeChat.StubCode2Session(code, "openid-175-unknown-client", "")

	resp := h.PostJSON(t, "/api/v1/auth/login/wechat", map[string]string{
		"app_id": uniqueAppID("unregistered"),
		"code":   code,
	})

	require.Equal(t, 403, resp.Status, string(resp.Body))
	assert.Empty(t, h.WeChat.Calls(), "未注册客户端不得调用微信 code2session")
}

func TestWeChatLoginRejectsDisabledClient(t *testing.T) {
	h := authharness.New(t)
	appID := uniqueAppID("disabled")
	seedClient(t, h, appID, domain.ClientRoleParent, domain.ClientStatusDisabled)
	code := uniqueCode("disabled-client")
	h.WeChat.StubCode2Session(code, "openid-175-disabled-client", "")

	resp := h.PostJSON(t, "/api/v1/auth/login/wechat", map[string]string{
		"app_id": appID,
		"code":   code,
	})

	require.Equal(t, 403, resp.Status, string(resp.Body))
	assert.Empty(t, h.WeChat.Calls(), "已停用客户端不得调用微信 code2session")
}

// 请求体里的 role 只能用于一致性校验，不能改变授权角色。客户端取自注册表，
// 因此「用家长端 app_id 冒充 teacher」必须在 code2session 之前被拒。
func TestWeChatLoginRejectsBodyRoleMismatch(t *testing.T) {
	h := authharness.New(t)
	appID := uniqueAppID("parent-role")
	seedClient(t, h, appID, domain.ClientRoleParent, domain.ClientStatusEnabled)
	code := uniqueCode("role-mismatch")
	h.WeChat.StubCode2Session(code, "openid-175-role-mismatch", "")

	resp := h.PostJSON(t, "/api/v1/auth/login/wechat", map[string]string{
		"app_id": appID,
		"code":   code,
		"role":   domain.ClientRoleTeacher,
	})

	require.Equal(t, 422, resp.Status, string(resp.Body))
	assert.Empty(t, h.WeChat.Calls(), "角色冲突时不得调用微信 code2session")
}

// invite_code 只允许老师端携带：家长端带邀请码说明客户端在试探老师绑定通道，
// 必须显式拒绝，而不是静默忽略。
func TestWeChatLoginRejectsInviteCodeOnParentClient(t *testing.T) {
	h := authharness.New(t)
	appID := uniqueAppID("parent-invite")
	seedClient(t, h, appID, domain.ClientRoleParent, domain.ClientStatusEnabled)
	code := uniqueCode("parent-invite")
	h.WeChat.StubCode2Session(code, "openid-175-parent-invite", "")

	resp := h.PostJSON(t, "/api/v1/auth/login/wechat", map[string]string{
		"app_id":      appID,
		"code":        code,
		"invite_code": "invite-175-parent",
	})

	require.Equal(t, 422, resp.Status, string(resp.Body))
	assert.Empty(t, h.WeChat.Calls(), "家长端带邀请码时不得调用微信 code2session")
}

// 首次登录：按 (app_id, open_id) 找不到身份时创建「待绑定账号 + 身份」。
// 待绑定账号没有手机号、没有角色、不属于任何园所，token 角色必须为空——
// 登录本身不得让用户成为任何园所的成员。
func TestWeChatLoginCreatesPendingAccountWithEmptyRoles(t *testing.T) {
	h := authharness.New(t)
	appID := uniqueAppID("pending")
	seedClient(t, h, appID, domain.ClientRoleParent, domain.ClientStatusEnabled)
	openID := uniqueOpenID("pending")
	code := uniqueCode("pending")
	h.WeChat.StubCode2Session(code, openID, "")
	cleanupIdentityArtifacts(t, h, appID, openID)

	resp := h.PostJSON(t, "/api/v1/auth/login/wechat", map[string]string{
		"app_id": appID,
		"code":   code,
		"role":   domain.ClientRoleParent,
	})
	require.Equal(t, 200, resp.Status, string(resp.Body))

	var result service.LoginResult
	resp.DecodeData(t, &result)
	require.NotEmpty(t, result.Token)
	require.NotNil(t, result.User)

	claims, err := auth.ParseAccess(h.Secret, result.Token)
	require.NoError(t, err)
	assert.Equal(t, appID, claims.AppID, "token 必须绑定签发它的客户端")
	assert.Equal(t, domain.ClientScopeShared, claims.ClientScopeKind)
	assert.Empty(t, claims.Roles, "待绑定账号不得携带任何角色")
	assert.Zero(t, claims.KindergartenID, "登录不得自动把用户挂到某个园所")

	assert.Empty(t, result.User.Roles)
	assert.Empty(t, result.User.Phone, "待绑定账号不应有手机号")

	calls := h.WeChat.Calls()
	require.Len(t, calls, 1)
	assert.Equal(t, "Code2Session", calls[0].Method)
	assert.Equal(t, domain.ClientRoleParent, calls[0].Role, "必须用客户端角色对应的微信凭据")

	var identity domain.UserIdentity
	require.NoError(t, h.DB.Where("app_id = ? AND open_id = ?", appID, openID).First(&identity).Error)
	var user domain.User
	require.NoError(t, h.DB.First(&user, identity.UserID).Error)
	assert.Empty(t, user.Phone, "待绑定账号手机号必须为空")

	var roleCount int64
	require.NoError(t, h.DB.Model(&domain.UserRole{}).Where("user_id = ?", user.ID).Count(&roleCount).Error)
	assert.Zero(t, roleCount, "登录不得自动授予任何角色")
}

// 同一 (app_id, open_id) 再次登录必须复用同一账号与身份，不能再建一条。
func TestWeChatLoginReusesExistingIdentity(t *testing.T) {
	h := authharness.New(t)
	appID := uniqueAppID("reuse")
	seedClient(t, h, appID, domain.ClientRoleParent, domain.ClientStatusEnabled)
	openID := uniqueOpenID("reuse")
	code := uniqueCode("reuse")
	h.WeChat.StubCode2Session(code, openID, "")
	cleanupIdentityArtifacts(t, h, appID, openID)

	first := loginOK(t, h, appID, code)
	second := loginOK(t, h, appID, code)

	assert.Equal(t, first.User.ID, second.User.ID, "同一身份必须复用同一账号")

	var identities []domain.UserIdentity
	require.NoError(t, h.DB.Where("app_id = ? AND open_id = ?", appID, openID).Find(&identities).Error)
	require.Len(t, identities, 1, "同一 (app_id, open_id) 只能有一条身份")
	assert.Equal(t, first.User.ID, identities[0].UserID)
}

// invite_code 在老师端是合法输入；绑定动作本身属于 #168，本票不建角色。
func TestWeChatLoginAcceptsInviteCodeOnTeacherClient(t *testing.T) {
	h := authharness.New(t)
	appID := uniqueAppID("teacher-invite")
	seedClient(t, h, appID, domain.ClientRoleTeacher, domain.ClientStatusEnabled)
	openID := uniqueOpenID("teacher-invite")
	code := uniqueCode("teacher-invite")
	h.WeChat.StubCode2Session(code, openID, "")
	cleanupIdentityArtifacts(t, h, appID, openID)

	resp := h.PostJSON(t, "/api/v1/auth/login/wechat", map[string]string{
		"app_id":      appID,
		"code":        code,
		"invite_code": "invite-175-teacher",
	})
	require.Equal(t, 200, resp.Status, string(resp.Body))

	var result service.LoginResult
	resp.DecodeData(t, &result)
	assert.Empty(t, result.User.Roles, "邀请码绑定属于 #168：本票登录不得授予老师角色")

	calls := h.WeChat.Calls()
	require.Len(t, calls, 1)
	assert.Equal(t, domain.ClientRoleTeacher, calls[0].Role)
}

// 待绑定 token 只能访问绑定相关与资料类接口；业务接口一律 403，且必须给出
// 可判别的业务码，让客户端跳绑定页而不是误报「无权限」。
func TestUnboundTokenIsLimitedToWhitelist(t *testing.T) {
	h := authharness.New(t)
	appID := uniqueAppID("whitelist")
	seedClient(t, h, appID, domain.ClientRoleParent, domain.ClientStatusEnabled)
	openID := uniqueOpenID("whitelist")
	code := uniqueCode("whitelist")
	h.WeChat.StubCode2Session(code, openID, "")
	cleanupIdentityArtifacts(t, h, appID, openID)

	token := loginOK(t, h, appID, code).Token

	blocked := h.Get(t, "/api/v1/parent/home/ping", token)
	require.Equal(t, 403, blocked.Status, string(blocked.Body))
	assert.Equal(t, 40302, blocked.EnvelopeCode(t), "未绑定应返回专属业务码；body=%s", blocked.Body)

	profile := h.Get(t, "/api/v1/common/user/profile", token)
	require.Equal(t, 200, profile.Status, string(profile.Body))
	var profileData service.ProfileInfo
	profile.DecodeData(t, &profileData)
	assert.Empty(t, profileData.Roles)

	logout := h.PostJSONWithToken(t, "/api/v1/auth/logout", token, map[string]string{})
	require.Equal(t, 200, logout.Status, string(logout.Body))
}

// 同一账号同时拥有 parent 与 teacher 角色时，token 与登录/资料响应只能出现
// 当前客户端角色族的角色；园所也由裁剪后的角色推导，不能串到另一端。
//
// must_change_password 只对管理端密码登录生效：小程序客户端登录必须忽略它，
// 否则家长/老师会被卡在一个小程序里根本无法完成的改密流程上。
func TestWeChatLoginFiltersRolesByClientAndIgnoresMustChangePassword(t *testing.T) {
	h := authharness.New(t)
	parentApp := uniqueAppID("filter-parent")
	teacherApp := uniqueAppID("filter-teacher")
	seedClient(t, h, parentApp, domain.ClientRoleParent, domain.ClientStatusEnabled)
	seedClient(t, h, teacherApp, domain.ClientRoleTeacher, domain.ClientStatusEnabled)

	parentOpenID := uniqueOpenID("filter-parent")
	teacherOpenID := uniqueOpenID("filter-teacher")
	parentCode := uniqueCode("filter-parent")
	teacherCode := uniqueCode("filter-teacher")
	h.WeChat.StubCode2Session(parentCode, parentOpenID, "")
	h.WeChat.StubCode2Session(teacherCode, teacherOpenID, "")

	// user_roles.kindergarten_id 有指向 kindergartens 的外键，必须建真实园所。
	kgParent := seedKindergarten(t, h, "家长端园所")
	kgTeacher := seedKindergarten(t, h, "老师端园所")
	kgParentID := kgParent.ID
	kgTeacherID := kgTeacher.ID

	user := &domain.User{MustChangePassword: true}
	require.NoError(t, h.DB.Create(user).Error)
	t.Cleanup(func() {
		h.HardDelete(t, &domain.UserRole{}, "user_id = ?", user.ID)
		h.HardDelete(t, &domain.UserIdentity{}, "user_id = ?", user.ID)
		h.HardDelete(t, &domain.User{}, "id = ?", user.ID)
	})
	require.NoError(t, h.DB.Create(&domain.UserRole{
		UserID: user.ID, Role: domain.ClientRoleParent, KindergartenID: &kgParentID, KindergartenName: kgParent.Name,
	}).Error)
	require.NoError(t, h.DB.Create(&domain.UserRole{
		UserID: user.ID, Role: domain.ClientRoleTeacher, KindergartenID: &kgTeacherID, KindergartenName: kgTeacher.Name,
	}).Error)
	require.NoError(t, h.DB.Create(&domain.UserIdentity{
		UserID: user.ID, AppID: parentApp, OpenID: parentOpenID,
	}).Error)
	require.NoError(t, h.DB.Create(&domain.UserIdentity{
		UserID: user.ID, AppID: teacherApp, OpenID: teacherOpenID,
	}).Error)

	// 家长端：只出 parent 角色，园所取 parent 角色那条。
	parentLogin := h.PostJSON(t, "/api/v1/auth/login/wechat", map[string]string{
		"app_id": parentApp, "code": parentCode,
	})
	require.Equal(t, 200, parentLogin.Status, string(parentLogin.Body))
	var parentResult service.LoginResult
	parentLogin.DecodeData(t, &parentResult)

	parentClaims, err := auth.ParseAccess(h.Secret, parentResult.Token)
	require.NoError(t, err)
	assert.Equal(t, []string{domain.ClientRoleParent}, parentClaims.Roles)
	assert.Equal(t, kgParentID, parentClaims.KindergartenID)
	assert.False(t, parentResult.User.MustChangePassword, "小程序登录不得要求改密")
	require.Len(t, parentResult.User.Roles, 1)
	assert.Equal(t, domain.ClientRoleParent, parentResult.User.Roles[0].Role)

	// 已绑定 token 能通过 Casbin 进入业务接口。
	ping := h.Get(t, "/api/v1/parent/home/ping", parentResult.Token)
	require.Equal(t, 200, ping.Status, string(ping.Body))

	// 资料响应同样只返回当前客户端角色。
	profile := h.Get(t, "/api/v1/common/user/profile", parentResult.Token)
	require.Equal(t, 200, profile.Status, string(profile.Body))
	var profileData service.ProfileInfo
	profile.DecodeData(t, &profileData)
	require.Len(t, profileData.Roles, 1)
	assert.Equal(t, domain.ClientRoleParent, profileData.Roles[0].Role)

	// 老师端：同一账号只出 teacher 角色与老师端园所。
	teacherLogin := h.PostJSON(t, "/api/v1/auth/login/wechat", map[string]string{
		"app_id": teacherApp, "code": teacherCode,
	})
	require.Equal(t, 200, teacherLogin.Status, string(teacherLogin.Body))
	var teacherResult service.LoginResult
	teacherLogin.DecodeData(t, &teacherResult)

	teacherClaims, err := auth.ParseAccess(h.Secret, teacherResult.Token)
	require.NoError(t, err)
	assert.Equal(t, []string{domain.ClientRoleTeacher}, teacherClaims.Roles)
	assert.Equal(t, kgTeacherID, teacherClaims.KindergartenID)
	assert.False(t, teacherResult.User.MustChangePassword)
	require.Len(t, teacherResult.User.Roles, 1)
	assert.Equal(t, domain.ClientRoleTeacher, teacherResult.User.Roles[0].Role)
}

// --- helpers ---

func uniqueOpenID(scene string) string {
	return fmt.Sprintf("openid-175-%s-%d", scene, time.Now().UnixNano())
}

func loginOK(t *testing.T, h *authharness.Harness, appID, code string) *service.LoginResult {
	t.Helper()
	resp := h.PostJSON(t, "/api/v1/auth/login/wechat", map[string]string{
		"app_id": appID,
		"code":   code,
	})
	require.Equal(t, 200, resp.Status, string(resp.Body))
	var result service.LoginResult
	resp.DecodeData(t, &result)
	return &result
}

// cleanupIdentityArtifacts 硬删本用例写入的 user_roles / user_identities / users。
// 软删行仍占 phone_active 等唯一索引，必须 Unscoped 清理。
func cleanupIdentityArtifacts(t *testing.T, h *authharness.Harness, appID, openID string) {
	t.Helper()
	t.Cleanup(func() {
		var identity domain.UserIdentity
		if err := h.DB.Where("app_id = ? AND open_id = ?", appID, openID).First(&identity).Error; err != nil {
			return
		}
		h.HardDelete(t, &domain.UserRole{}, "user_id = ?", identity.UserID)
		h.HardDelete(t, &domain.UserIdentity{}, "id = ?", identity.ID)
		h.HardDelete(t, &domain.User{}, "id = ?", identity.UserID)
	})
}

func seedKindergarten(t *testing.T, h *authharness.Harness, name string) *domain.Kindergarten {
	t.Helper()
	kg := &domain.Kindergarten{Name: fmt.Sprintf("%s-%d", name, time.Now().UnixNano())}
	require.NoError(t, h.DB.Create(kg).Error)
	t.Cleanup(func() { h.HardDelete(t, &domain.Kindergarten{}, "id = ?", kg.ID) })
	return kg
}

func uniqueCode(scene string) string {
	return fmt.Sprintf("code-175-%s-%d", scene, time.Now().UnixNano())
}

func uniqueAppID(scene string) string {
	return fmt.Sprintf("wx-175-%s-%d", scene, time.Now().UnixNano())
}

// seedClient 直接写注册表：每个用例用独立 app_id，避免共享测试库里
// 别的用例改 status 影响本用例。clients 没有删除接口，清理走硬删。
func seedClient(t *testing.T, h *authharness.Harness, appID, role, status string) *domain.Client {
	t.Helper()
	client := &domain.Client{
		AppID:     appID,
		Role:      role,
		ScopeKind: domain.ClientScopeShared,
		Status:    status,
	}
	require.NoError(t, h.DB.Create(client).Error)
	t.Cleanup(func() { h.HardDelete(t, &domain.Client{}, "app_id = ?", appID) })
	return client
}
