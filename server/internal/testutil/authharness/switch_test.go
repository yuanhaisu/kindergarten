package authharness_test

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/service"
	"kindergarten-server/internal/testutil/authharness"
)

// #169：切换园所只允许切到「当前客户端角色族 ∩ 园所通道」过滤后的成员园，
// 并重新签发绑定目标园的 access + refresh；旧 refresh 由原子消费吊销。
//
// dedicated 客户端行在 v1 运行时不存在（启动同步拒绝），这里直接插库绕过
// 启动校验，只用于验证 scope 锁定这一条契约。

func switchRequest(t *testing.T, h *authharness.Harness, token string, kgID uint64, refresh string) *authharness.Response {
	t.Helper()
	return h.PostJSONWithToken(t, "/api/v1/auth/switch-kindergarten", token, map[string]any{
		"kindergarten_id": kgID,
		"refresh_token":   refresh,
	})
}

func switchOK(t *testing.T, h *authharness.Harness, token string, kgID uint64, refresh string) *service.LoginResult {
	t.Helper()
	resp := switchRequest(t, h, token, kgID, refresh)
	require.Equal(t, 200, resp.Status, string(resp.Body))
	var result service.LoginResult
	resp.DecodeData(t, &result)
	return &result
}

// seedDedicatedClient 直接插一条 kindergarten 作用域的客户端行。
//
// #177 的启动同步会拒绝这种行；本用例需要它在 DB 里存在，才能验证
// 「专属作用域只服务自己的园」。
func seedDedicatedClient(t *testing.T, h *authharness.Harness, appID, role string, kgID uint64) *domain.Client {
	t.Helper()
	client := &domain.Client{
		AppID:     appID,
		Role:      role,
		ScopeKind: domain.ClientScopeKindergarten,
		ScopeID:   kgID,
		Status:    domain.ClientStatusEnabled,
	}
	require.NoError(t, h.DB.Create(client).Error)
	t.Cleanup(func() { h.HardDelete(t, &domain.Client{}, "app_id = ?", appID) })
	return client
}

// 切换成功后：目标园生效、只带当前客户端角色族、旧 refresh 失效、新 refresh
// 继续停留在目标园。
func TestSwitchKindergartenReissuesScopedTokens(t *testing.T) {
	h := authharness.New(t)
	appID := uniqueAppID("switch-scoped")
	seedSharedClient(t, h, appID, domain.ClientRoleParent)

	kgA := seedKindergartenWithChannel(t, h, "切换甲园", domain.AppChannelShared)
	kgB := seedKindergartenWithChannel(t, h, "切换乙园", domain.AppChannelShared)
	openID := uniqueOpenID("switch-scoped")
	seedMember(t, h, appID, openID,
		memberRole{role: domain.ClientRoleParent, kg: kgA},
		memberRole{role: domain.ClientRoleParent, kg: kgB},
		memberRole{role: domain.ClientRoleTeacher, kg: kgA},
	)

	login := loginWithClient(t, h, appID, openID)
	loginClaims, err := auth.ParseAccess(h.Secret, login.Token)
	require.NoError(t, err)
	require.Equal(t, kgA.ID, loginClaims.KindergartenID)

	switched := switchOK(t, h, login.Token, kgB.ID, login.RefreshToken)
	claims, err := auth.ParseAccess(h.Secret, switched.Token)
	require.NoError(t, err)
	assert.Equal(t, appID, claims.AppID, "切换不得改变客户端作用域")
	assert.Equal(t, kgB.ID, claims.KindergartenID, "切换后会话只能有目标园")
	assert.Equal(t, []string{domain.ClientRoleParent}, claims.Roles, "切换不得带出老师角色")
	assert.NotEqual(t, login.RefreshToken, switched.RefreshToken, "切换必须换发新的 refresh")

	// 旧 refresh 已被消费，重放必须失败。
	replay := refreshRequest(t, h, login.RefreshToken)
	require.Equal(t, 401, replay.Status, string(replay.Body))

	// 新 refresh 继续停留在切换后的园所。
	refreshed := refreshOK(t, h, switched.RefreshToken)
	nextClaims, err := auth.ParseAccess(h.Secret, refreshed.Token)
	require.NoError(t, err)
	assert.Equal(t, kgB.ID, nextClaims.KindergartenID)
	assert.Equal(t, appID, nextClaims.AppID)
}

// 不在过滤后成员列表里的园一律拒绝：角色族不匹配、完全无成员关系都要拒绝。
func TestSwitchRejectsKindergartenOutsideMembershipFilter(t *testing.T) {
	h := authharness.New(t)
	appID := uniqueAppID("switch-nonmember")
	seedSharedClient(t, h, appID, domain.ClientRoleParent)

	kgA := seedKindergartenWithChannel(t, h, "本人家长园", domain.AppChannelShared)
	kgTeacher := seedKindergartenWithChannel(t, h, "本人老师园", domain.AppChannelShared)
	kgOther := seedKindergartenWithChannel(t, h, "无关园", domain.AppChannelShared)
	openID := uniqueOpenID("switch-nonmember")
	seedMember(t, h, appID, openID,
		memberRole{role: domain.ClientRoleParent, kg: kgA},
		memberRole{role: domain.ClientRoleTeacher, kg: kgTeacher},
	)

	for name, target := range map[string]uint64{
		"老师角色园": kgTeacher.ID,
		"无关园":   kgOther.ID,
	} {
		t.Run(name, func(t *testing.T) {
			login := loginWithClient(t, h, appID, openID)
			resp := switchRequest(t, h, login.Token, target, login.RefreshToken)
			require.Equal(t, 403, resp.Status, string(resp.Body))
		})
	}
}

// 共用客户端看不到专属通道的园，也不能通过切换越界进入。
func TestSwitchSharedClientCannotEnterDedicatedKindergarten(t *testing.T) {
	h := authharness.New(t)
	appID := uniqueAppID("switch-shared-dedicated")
	seedSharedClient(t, h, appID, domain.ClientRoleParent)

	kgShared := seedKindergartenWithChannel(t, h, "共用园", domain.AppChannelShared)
	kgDedicated := seedKindergartenWithChannel(t, h, "专属通道园", domain.AppChannelDedicated)
	openID := uniqueOpenID("switch-shared-dedicated")
	seedMember(t, h, appID, openID,
		memberRole{role: domain.ClientRoleParent, kg: kgShared},
		memberRole{role: domain.ClientRoleParent, kg: kgDedicated},
	)

	login := loginWithClient(t, h, appID, openID)
	resp := switchRequest(t, h, login.Token, kgDedicated.ID, login.RefreshToken)
	require.Equal(t, 403, resp.Status, string(resp.Body))
}

// 专属客户端（test-only）锁定自己的园：列表只有它，切换到其它成员园被拒。
func TestSwitchDedicatedClientLockedToItsKindergarten(t *testing.T) {
	h := authharness.New(t)
	appID := uniqueAppID("switch-dedicated")
	kgDedicated := seedKindergartenWithChannel(t, h, "专属客户端园", domain.AppChannelDedicated)
	kgShared := seedKindergartenWithChannel(t, h, "专属客户端无关园", domain.AppChannelShared)
	seedDedicatedClient(t, h, appID, domain.ClientRoleParent, kgDedicated.ID)

	openID := uniqueOpenID("switch-dedicated")
	seedMember(t, h, appID, openID,
		memberRole{role: domain.ClientRoleParent, kg: kgDedicated},
		memberRole{role: domain.ClientRoleParent, kg: kgShared},
	)

	login := loginWithClient(t, h, appID, openID)
	claims, err := auth.ParseAccess(h.Secret, login.Token)
	require.NoError(t, err)
	require.Equal(t, kgDedicated.ID, claims.KindergartenID)

	items := listMemberships(t, h, login.Token)
	require.Len(t, items, 1, "专属客户端只能看到 scope 内的园")
	assert.Equal(t, kgDedicated.ID, items[0].KindergartenID)

	resp := switchRequest(t, h, login.Token, kgShared.ID, login.RefreshToken)
	require.Equal(t, 403, resp.Status, string(resp.Body))
}

// 管理端凭证没有 app_id，不能切换园所。
func TestSwitchRejectsAdminToken(t *testing.T) {
	h := authharness.New(t)
	kg := seedKindergartenWithChannel(t, h, "管理端目标园", domain.AppChannelShared)

	token := signScopedAccess(t, h, "admin", kg.ID, "")
	resp := switchRequest(t, h, token, kg.ID, "any-refresh")
	require.Equal(t, 403, resp.Status, string(resp.Body))
}

// 待绑定 token 不属于可切换会话，必须在白名单阶段被拒。
func TestSwitchRejectsUnboundToken(t *testing.T) {
	h := authharness.New(t)
	appID := uniqueAppID("switch-unbound")
	seedSharedClient(t, h, appID, domain.ClientRoleParent)
	kg := seedKindergartenWithChannel(t, h, "未绑定目标园", domain.AppChannelShared)
	openID := uniqueOpenID("switch-unbound")

	login := loginWithClient(t, h, appID, openID)
	resp := switchRequest(t, h, login.Token, kg.ID, login.RefreshToken)
	require.Equal(t, 403, resp.Status, string(resp.Body))
	assert.Equal(t, 40302, resp.EnvelopeCode(t), "未绑定必须返回绑定专用业务码")
}

// 旧 refresh 不能跨客户端兑换：老师端 access + 家长端 refresh 必须失败。
func TestSwitchRejectsRefreshFromAnotherClient(t *testing.T) {
	h := authharness.New(t)
	parentApp := uniqueAppID("switch-cross-parent")
	teacherApp := uniqueAppID("switch-cross-teacher")
	seedSharedClient(t, h, parentApp, domain.ClientRoleParent)
	seedSharedClient(t, h, teacherApp, domain.ClientRoleTeacher)

	kgParent := seedKindergartenWithChannel(t, h, "家长园", domain.AppChannelShared)
	kgTeacher := seedKindergartenWithChannel(t, h, "老师园", domain.AppChannelShared)
	parentOpenID := uniqueOpenID("switch-cross-parent")
	teacherOpenID := uniqueOpenID("switch-cross-teacher")
	user := seedMember(t, h, parentApp, parentOpenID,
		memberRole{role: domain.ClientRoleParent, kg: kgParent},
		memberRole{role: domain.ClientRoleTeacher, kg: kgTeacher},
	)
	require.NoError(t, h.DB.Create(&domain.UserIdentity{
		UserID: user.ID, AppID: teacherApp, OpenID: teacherOpenID,
	}).Error)

	parentLogin := loginWithClient(t, h, parentApp, parentOpenID)
	teacherLogin := loginWithClient(t, h, teacherApp, teacherOpenID)

	resp := switchRequest(t, h, teacherLogin.Token, kgTeacher.ID, parentLogin.RefreshToken)
	require.Equal(t, 401, resp.Status, string(resp.Body))
}

// 切换与续期拿着同一个旧 refresh 并发请求：原子消费只能有一个成功。
func TestSwitchAndRefreshConsumeRefreshOnce(t *testing.T) {
	h := authharness.New(t)
	appID := uniqueAppID("switch-race")
	seedSharedClient(t, h, appID, domain.ClientRoleParent)

	kgA := seedKindergartenWithChannel(t, h, "竞态甲园", domain.AppChannelShared)
	kgB := seedKindergartenWithChannel(t, h, "竞态乙园", domain.AppChannelShared)
	openID := uniqueOpenID("switch-race")
	seedMember(t, h, appID, openID,
		memberRole{role: domain.ClientRoleParent, kg: kgA},
		memberRole{role: domain.ClientRoleParent, kg: kgB},
	)

	login := loginWithClient(t, h, appID, openID)

	var wg sync.WaitGroup
	statuses := make(chan int, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		statuses <- switchRequest(t, h, login.Token, kgB.ID, login.RefreshToken).Status
	}()
	go func() {
		defer wg.Done()
		statuses <- refreshRequest(t, h, login.RefreshToken).Status
	}()
	wg.Wait()
	close(statuses)

	var ok, unauthorized int
	for status := range statuses {
		switch status {
		case 200:
			ok++
		case 401:
			unauthorized++
		}
	}
	assert.Equal(t, 1, ok, "同一旧 refresh 只能有一个请求成功")
	assert.Equal(t, 1, unauthorized, "输掉竞态的请求必须明确失败")

	// 竞态结束后旧 refresh 一定已被消费。
	require.Equal(t, 401, refreshRequest(t, h, login.RefreshToken).Status)
}

// 缺少目标园或 refresh 的请求在绑定层就应拒绝，不能落到业务逻辑。
func TestSwitchRequiresKindergartenAndRefresh(t *testing.T) {
	h := authharness.New(t)
	appID := uniqueAppID("switch-invalid")
	seedSharedClient(t, h, appID, domain.ClientRoleParent)
	kg := seedKindergartenWithChannel(t, h, "校验园", domain.AppChannelShared)
	openID := uniqueOpenID("switch-invalid")
	seedMember(t, h, appID, openID, memberRole{role: domain.ClientRoleParent, kg: kg})
	login := loginWithClient(t, h, appID, openID)

	resp := h.PostJSONWithToken(t, "/api/v1/auth/switch-kindergarten", login.Token, map[string]any{
		"kindergarten_id": 0,
		"refresh_token":   login.RefreshToken,
	})
	require.Equal(t, 422, resp.Status, string(resp.Body))
}
