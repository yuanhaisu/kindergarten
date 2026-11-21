package authharness_test

import (
	"context"
	"encoding/json"
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

// #169：refresh token 必须带完整作用域（uid / app_id / scope / 园所）。
//
// 旧实现只存裸 userID：刷新时无法知道「这是哪个客户端、哪个园」，
// 多客户端账号会被重新推导到错误的园，甚至跨端换出别的角色。
// 这一组锁住存储格式与刷新时的作用域校验。

type storedRefresh struct {
	UID            uint64 `json:"uid"`
	AppID          string `json:"app_id"`
	ScopeKind      string `json:"scope_kind"`
	ScopeID        uint64 `json:"scope_id"`
	KindergartenID uint64 `json:"kindergarten_id"`
}

func refreshRequest(t *testing.T, h *authharness.Harness, refreshToken string) *authharness.Response {
	t.Helper()
	return h.PostJSON(t, "/api/v1/auth/refresh", map[string]string{
		"refresh_token": refreshToken,
	})
}

func refreshOK(t *testing.T, h *authharness.Harness, refreshToken string) *service.LoginResult {
	t.Helper()
	resp := refreshRequest(t, h, refreshToken)
	require.Equal(t, 200, resp.Status, string(resp.Body))
	var result service.LoginResult
	resp.DecodeData(t, &result)
	return &result
}

func TestRefreshTokenStoresScopedJSON(t *testing.T) {
	h := authharness.New(t)
	appID := uniqueAppID("refresh-json")
	seedSharedClient(t, h, appID, domain.ClientRoleParent)
	kg := seedKindergartenWithChannel(t, h, "续期园", domain.AppChannelShared)
	openID := uniqueOpenID("refresh-json")
	user := seedMember(t, h, appID, openID, memberRole{role: domain.ClientRoleParent, kg: kg})

	result := loginWithClient(t, h, appID, openID)

	value, err := auth.ConsumeRefresh(context.Background(), h.RedisClient, result.RefreshToken)
	require.NoError(t, err, "refresh token 必须能在园所作用域内消费")

	var stored storedRefresh
	require.NoError(t, json.Unmarshal([]byte(value), &stored),
		"refresh 存储必须是 JSON，而不是裸 userID；实际值=%q", value)
	assert.Equal(t, user.ID, stored.UID)
	assert.Equal(t, appID, stored.AppID, "refresh 必须绑定签发它的客户端")
	assert.Equal(t, domain.ClientScopeShared, stored.ScopeKind)
	assert.Zero(t, stored.ScopeID)
	assert.Equal(t, kg.ID, stored.KindergartenID, "refresh 必须绑定签发时的园所")
}

// 旧版本把裸 userID 存进 refresh: 键。产品未上线，不做兼容：必须拒绝。
func TestRefreshRejectsLegacyRawValue(t *testing.T) {
	h := authharness.New(t)
	user := seedMember(t, h, "", "")
	legacy := fmt.Sprintf("legacy-refresh-%d", time.Now().UnixNano())
	require.NoError(t, auth.SaveRefresh(context.Background(), h.RedisClient, legacy,
		fmt.Sprintf("%d", user.ID), time.Hour))

	resp := refreshRequest(t, h, legacy)
	require.Equal(t, 401, resp.Status, string(resp.Body))
}

// 刷新必须沿用签发时的客户端与园所：用户同时在多个园、两端都有角色，
// 也只换出同一客户端角色族、同一园所的凭证；换出的 refresh 继续轮换。
func TestRefreshPreservesClientScopeAndRotates(t *testing.T) {
	h := authharness.New(t)
	parentApp := uniqueAppID("refresh-scope-parent")
	teacherApp := uniqueAppID("refresh-scope-teacher")
	seedSharedClient(t, h, parentApp, domain.ClientRoleParent)
	seedSharedClient(t, h, teacherApp, domain.ClientRoleTeacher)

	kgA := seedKindergartenWithChannel(t, h, "甲园", domain.AppChannelShared)
	kgB := seedKindergartenWithChannel(t, h, "乙园", domain.AppChannelShared)
	parentOpenID := uniqueOpenID("refresh-scope-parent")
	teacherOpenID := uniqueOpenID("refresh-scope-teacher")
	user := seedMember(t, h, parentApp, parentOpenID,
		memberRole{role: domain.ClientRoleParent, kg: kgA},
		memberRole{role: domain.ClientRoleParent, kg: kgB},
		memberRole{role: domain.ClientRoleTeacher, kg: kgA},
	)
	require.NoError(t, h.DB.Create(&domain.UserIdentity{
		UserID: user.ID, AppID: teacherApp, OpenID: teacherOpenID,
	}).Error)

	login := loginWithClient(t, h, parentApp, parentOpenID)
	loginClaims, err := auth.ParseAccess(h.Secret, login.Token)
	require.NoError(t, err)
	require.Equal(t, kgA.ID, loginClaims.KindergartenID, "登录取过滤后的第一个园")

	refreshed := refreshOK(t, h, login.RefreshToken)
	require.NotEqual(t, login.RefreshToken, refreshed.RefreshToken, "刷新必须轮换 refresh token")
	claims, err := auth.ParseAccess(h.Secret, refreshed.Token)
	require.NoError(t, err)
	assert.Equal(t, parentApp, claims.AppID, "刷新不得换成另一个客户端的凭证")
	assert.Equal(t, []string{domain.ClientRoleParent}, claims.Roles, "刷新不得带出老师角色")
	assert.Equal(t, kgA.ID, claims.KindergartenID, "刷新必须停留在签发时的园所")

	// 旧 refresh 已被消费，重放必须失败。
	replay := refreshRequest(t, h, login.RefreshToken)
	require.Equal(t, 401, replay.Status, string(replay.Body))

	// 新 refresh 仍可继续轮换，且园所不变。
	next := refreshOK(t, h, refreshed.RefreshToken)
	nextClaims, err := auth.ParseAccess(h.Secret, next.Token)
	require.NoError(t, err)
	assert.Equal(t, kgA.ID, nextClaims.KindergartenID)
	assert.Equal(t, parentApp, nextClaims.AppID)
}

// 客户端被停用后，已签发的 refresh 不能再兑换。
func TestRefreshRejectsDisabledClient(t *testing.T) {
	h := authharness.New(t)
	appID := uniqueAppID("refresh-disabled")
	seedSharedClient(t, h, appID, domain.ClientRoleParent)
	kg := seedKindergartenWithChannel(t, h, "停用客户端园", domain.AppChannelShared)
	openID := uniqueOpenID("refresh-disabled")
	seedMember(t, h, appID, openID, memberRole{role: domain.ClientRoleParent, kg: kg})

	login := loginWithClient(t, h, appID, openID)
	require.NoError(t, h.DB.Model(&domain.Client{}).
		Where("app_id = ?", appID).
		Update("status", domain.ClientStatusDisabled).Error)

	resp := refreshRequest(t, h, login.RefreshToken)
	require.Equal(t, 401, resp.Status, string(resp.Body))
}

// 成员关系被移除后（例如园所删掉了该家长），refresh 不得继续签发该园凭证。
func TestRefreshRejectsRevokedMembership(t *testing.T) {
	h := authharness.New(t)
	appID := uniqueAppID("refresh-revoked")
	seedSharedClient(t, h, appID, domain.ClientRoleParent)
	kg := seedKindergartenWithChannel(t, h, "移除成员园", domain.AppChannelShared)
	openID := uniqueOpenID("refresh-revoked")
	user := seedMember(t, h, appID, openID, memberRole{role: domain.ClientRoleParent, kg: kg})

	login := loginWithClient(t, h, appID, openID)
	h.HardDelete(t, &domain.UserRole{}, "user_id = ?", user.ID)

	resp := refreshRequest(t, h, login.RefreshToken)
	require.Equal(t, 401, resp.Status, string(resp.Body))
}

// 待绑定账号没有角色，但仍需要在 access 过期后静默续期，避免被踢回登录页。
func TestRefreshKeepsUnboundTokenUnbound(t *testing.T) {
	h := authharness.New(t)
	appID := uniqueAppID("refresh-unbound")
	seedSharedClient(t, h, appID, domain.ClientRoleParent)
	openID := uniqueOpenID("refresh-unbound")

	login := loginWithClient(t, h, appID, openID)
	refreshed := refreshOK(t, h, login.RefreshToken)

	claims, err := auth.ParseAccess(h.Secret, refreshed.Token)
	require.NoError(t, err)
	assert.Equal(t, appID, claims.AppID)
	assert.Empty(t, claims.Roles, "待绑定账号刷新后仍不得获得角色")
	assert.Zero(t, claims.KindergartenID)
}
