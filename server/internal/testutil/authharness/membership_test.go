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

// #169：我的园所只返回「当前客户端角色族 ∩ 园所通道」过滤后的成员关系。
//
// 这一组只锁「看得到哪些园」：家长端不能看到自己当老师的园，共用客户端
// 不能看到专属通道的园，同一账号在两端各自只看到对应的一侧。

type membershipItem struct {
	KindergartenID uint64 `json:"kindergarten_id"`
	Name           string `json:"name"`
	Role           string `json:"role"`
	PublicCode     string `json:"public_code"`
}

// seedSharedClient 与 login_test.go 的 seedClient 相同，但单独命名避免
// 跨文件改动；app_id 唯一，清理走硬删。
func seedSharedClient(t *testing.T, h *authharness.Harness, appID, role string) *domain.Client {
	t.Helper()
	client := &domain.Client{
		AppID:     appID,
		Role:      role,
		ScopeKind: domain.ClientScopeShared,
		Status:    domain.ClientStatusEnabled,
	}
	require.NoError(t, h.DB.Create(client).Error)
	t.Cleanup(func() { h.HardDelete(t, &domain.Client{}, "app_id = ?", appID) })
	return client
}

func seedKindergartenWithChannel(t *testing.T, h *authharness.Harness, name, channel string) *domain.Kindergarten {
	t.Helper()
	kg := &domain.Kindergarten{
		Name:       fmt.Sprintf("%s-%d", name, time.Now().UnixNano()),
		AppChannel: channel,
	}
	require.NoError(t, h.DB.Create(kg).Error)
	t.Cleanup(func() { h.HardDelete(t, &domain.Kindergarten{}, "id = ?", kg.ID) })
	return kg
}

// seedMember 建立账号 + 客户端身份 + 若干角色。角色用 (role, kindergarten)
// 描述，kindergarten 为 nil 表示不挂园所。
type memberRole struct {
	role string
	kg   *domain.Kindergarten
}

func seedMember(t *testing.T, h *authharness.Harness, appID, openID string, roles ...memberRole) *domain.User {
	t.Helper()
	user := &domain.User{}
	require.NoError(t, h.DB.Create(user).Error)
	t.Cleanup(func() {
		h.HardDelete(t, &domain.UserRole{}, "user_id = ?", user.ID)
		h.HardDelete(t, &domain.UserIdentity{}, "user_id = ?", user.ID)
		h.HardDelete(t, &domain.User{}, "id = ?", user.ID)
	})
	if openID != "" {
		require.NoError(t, h.DB.Create(&domain.UserIdentity{
			UserID: user.ID, AppID: appID, OpenID: openID,
		}).Error)
	}
	for _, r := range roles {
		row := &domain.UserRole{UserID: user.ID, Role: r.role}
		if r.kg != nil {
			kgID := r.kg.ID
			row.KindergartenID = &kgID
			row.KindergartenName = r.kg.Name
		}
		require.NoError(t, h.DB.Create(row).Error)
	}
	return user
}

func loginWithClient(t *testing.T, h *authharness.Harness, appID, openID string) *service.LoginResult {
	t.Helper()
	code := uniqueCode("membership")
	h.WeChat.StubCode2Session(code, openID, "")
	resp := h.PostJSON(t, "/api/v1/auth/login/wechat", map[string]string{
		"app_id": appID,
		"code":   code,
	})
	require.Equal(t, 200, resp.Status, string(resp.Body))
	var result service.LoginResult
	resp.DecodeData(t, &result)
	return &result
}

func listMemberships(t *testing.T, h *authharness.Harness, token string) []membershipItem {
	t.Helper()
	resp := h.Get(t, "/api/v1/common/user/kindergartens", token)
	require.Equal(t, 200, resp.Status, string(resp.Body))
	var items []membershipItem
	resp.DecodeData(t, &items)
	return items
}

func TestMyKindergartensFilterByClientRoleAndChannel(t *testing.T) {
	h := authharness.New(t)
	parentApp := uniqueAppID("kg-parent")
	teacherApp := uniqueAppID("kg-teacher")
	seedSharedClient(t, h, parentApp, domain.ClientRoleParent)
	seedSharedClient(t, h, teacherApp, domain.ClientRoleTeacher)

	kgA := seedKindergartenWithChannel(t, h, "一号园", domain.AppChannelShared)
	kgB := seedKindergartenWithChannel(t, h, "二号园", domain.AppChannelShared)
	kgC := seedKindergartenWithChannel(t, h, "三号园", domain.AppChannelShared)
	kgDedicated := seedKindergartenWithChannel(t, h, "专属园", domain.AppChannelDedicated)

	parentOpenID := uniqueOpenID("kg-parent")
	teacherOpenID := uniqueOpenID("kg-teacher")
	user := seedMember(t, h, parentApp, parentOpenID,
		memberRole{role: domain.ClientRoleParent, kg: kgA},
		memberRole{role: domain.ClientRoleParent, kg: kgB},
		memberRole{role: domain.ClientRoleTeacher, kg: kgC},
		// 共用家长端不得看到专属通道的园，即便本人确实有成员关系。
		memberRole{role: domain.ClientRoleParent, kg: kgDedicated},
	)
	require.NoError(t, h.DB.Create(&domain.UserIdentity{
		UserID: user.ID, AppID: teacherApp, OpenID: teacherOpenID,
	}).Error)

	parentResult := loginWithClient(t, h, parentApp, parentOpenID)
	parentItems := listMemberships(t, h, parentResult.Token)
	require.Len(t, parentItems, 2, "家长端只应看到 shared 的家长园")
	assert.Equal(t, kgA.ID, parentItems[0].KindergartenID)
	assert.Equal(t, kgA.Name, parentItems[0].Name)
	assert.Equal(t, domain.ClientRoleParent, parentItems[0].Role)
	assert.Equal(t, kgB.ID, parentItems[1].KindergartenID)

	teacherResult := loginWithClient(t, h, teacherApp, teacherOpenID)
	teacherItems := listMemberships(t, h, teacherResult.Token)
	require.Len(t, teacherItems, 1, "老师端只应看到老师角色族的园")
	assert.Equal(t, kgC.ID, teacherItems[0].KindergartenID)
	assert.Equal(t, domain.ClientRoleTeacher, teacherItems[0].Role)
}

// #176：扫码落地要把 scene 里的 kg_<public_code> 匹配到成员园，
// 所以「我的园所」必须带出 public_code（公开码本身不是敏感信息）。
func TestMyKindergartensExposePublicCode(t *testing.T) {
	h := authharness.New(t)
	appID := uniqueAppID("kg-public-code")
	seedSharedClient(t, h, appID, domain.ClientRoleParent)
	kg := seedKindergartenWithChannel(t, h, "扫码园", domain.AppChannelShared)
	require.NoError(t, h.DB.Model(&domain.Kindergarten{}).
		Where("id = ?", kg.ID).Update("public_code", "scantest01").Error)
	openID := uniqueOpenID("kg-public-code")
	seedMember(t, h, appID, openID, memberRole{role: domain.ClientRoleParent, kg: kg})

	result := loginWithClient(t, h, appID, openID)
	items := listMemberships(t, h, result.Token)
	require.Len(t, items, 1)
	assert.Equal(t, "scantest01", items[0].PublicCode, "成员园必须带出公开码，否则扫码无法匹配到园")
}

// #160 L131：登录与资料响应的 roles 只含「当前客户端角色族 ∩ 会话园所」。
// 会话始终单园（token 的 kindergarten_id）；完整成员关系走「我的园所」。
func TestLoginAndProfileRolesNarrowedToSessionKindergarten(t *testing.T) {
	h := authharness.New(t)
	appID := uniqueAppID("kg-session-roles")
	seedSharedClient(t, h, appID, domain.ClientRoleParent)

	kgA := seedKindergartenWithChannel(t, h, "会话甲园", domain.AppChannelShared)
	kgB := seedKindergartenWithChannel(t, h, "会话乙园", domain.AppChannelShared)
	openID := uniqueOpenID("kg-session-roles")
	seedMember(t, h, appID, openID,
		memberRole{role: domain.ClientRoleParent, kg: kgA},
		memberRole{role: domain.ClientRoleParent, kg: kgB},
	)

	result := loginWithClient(t, h, appID, openID)
	claims, err := auth.ParseAccess(h.Secret, result.Token)
	require.NoError(t, err)
	require.NotZero(t, claims.KindergartenID, "登录必须落到一个会话园")

	require.Len(t, result.User.Roles, 1, "两园家长登录响应只能带会话园的角色")
	assert.Equal(t, domain.ClientRoleParent, result.User.Roles[0].Role)
	require.NotNil(t, result.User.Roles[0].KindergartenID)
	assert.Equal(t, claims.KindergartenID, *result.User.Roles[0].KindergartenID,
		"roles 必须与 token 的 kindergarten_id 一致")

	profileResp := h.Get(t, "/api/v1/common/user/profile", result.Token)
	require.Equal(t, 200, profileResp.Status, string(profileResp.Body))
	var profile service.ProfileInfo
	profileResp.DecodeData(t, &profile)
	require.Len(t, profile.Roles, 1, "资料响应同样只带会话园角色")
	require.NotNil(t, profile.Roles[0].KindergartenID)
	assert.Equal(t, claims.KindergartenID, *profile.Roles[0].KindergartenID)

	items := listMemberships(t, h, result.Token)
	require.Len(t, items, 2, "完整成员关系仍应出现在「我的园所」")

	// 切换后 roles 必须跟着会话一起换到目标园。
	otherKg := kgA.ID
	if claims.KindergartenID == kgA.ID {
		otherKg = kgB.ID
	}
	switched := switchOK(t, h, result.Token, otherKg, result.RefreshToken)
	require.Len(t, switched.User.Roles, 1)
	require.NotNil(t, switched.User.Roles[0].KindergartenID)
	assert.Equal(t, otherKg, *switched.User.Roles[0].KindergartenID)
}

// 老师端同一账号带两个园的老师角色时，登录响应同样只出一个园。
func TestTeacherLoginRolesNarrowedToSessionKindergarten(t *testing.T) {
	h := authharness.New(t)
	appID := uniqueAppID("kg-session-teacher")
	seedSharedClient(t, h, appID, domain.ClientRoleTeacher)

	kgA := seedKindergartenWithChannel(t, h, "老师甲园", domain.AppChannelShared)
	kgB := seedKindergartenWithChannel(t, h, "老师乙园", domain.AppChannelShared)
	openID := uniqueOpenID("kg-session-teacher")
	seedMember(t, h, appID, openID,
		memberRole{role: domain.ClientRoleTeacher, kg: kgA},
		memberRole{role: domain.ClientRoleTeacher, kg: kgB},
	)

	result := loginWithClient(t, h, appID, openID)
	claims, err := auth.ParseAccess(h.Secret, result.Token)
	require.NoError(t, err)
	require.Len(t, result.User.Roles, 1)
	require.NotNil(t, result.User.Roles[0].KindergartenID)
	assert.Equal(t, claims.KindergartenID, *result.User.Roles[0].KindergartenID)
}

// 待绑定 token 在未绑定白名单内：可以访问，但没有任何成员关系。
func TestMyKindergartensEmptyForUnboundToken(t *testing.T) {
	h := authharness.New(t)
	appID := uniqueAppID("kg-unbound")
	seedSharedClient(t, h, appID, domain.ClientRoleParent)
	openID := uniqueOpenID("kg-unbound")

	result := loginWithClient(t, h, appID, openID)
	require.Empty(t, result.User.Roles, "待绑定账号不应有角色")

	items := listMemberships(t, h, result.Token)
	assert.Empty(t, items)
}

// 管理端凭证没有小程序客户端作用域，不能调用「我的园所」。
func TestMyKindergartensRejectsTokenWithoutClientScope(t *testing.T) {
	h := authharness.New(t)

	token := signScopedAccess(t, h, domain.ClientRoleParent, 0, "")
	resp := h.Get(t, "/api/v1/common/user/kindergartens", token)
	require.Equal(t, 403, resp.Status, string(resp.Body))
}

// signScopedAccess 伪造一条测试用 access token（不经登录），用于覆盖
// 中间件/端点对「无客户端凭证」的拒绝。
func signScopedAccess(t *testing.T, h *authharness.Harness, role string, kgID uint64, appID string) string {
	t.Helper()
	uid := fmt.Sprintf("%d", time.Now().UnixNano())
	var (
		token string
		err   error
	)
	if appID == "" {
		token, err = auth.SignAccess(h.Secret, uid, []string{role}, kgID, time.Hour)
	} else {
		token, err = auth.SignClientAccess(h.Secret, uid, []string{role}, kgID, auth.ClientScope{
			AppID: appID, Kind: domain.ClientScopeShared,
		}, time.Hour)
	}
	require.NoError(t, err)
	return token
}
