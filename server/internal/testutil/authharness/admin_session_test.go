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
	"kindergarten-server/internal/repository"
	"kindergarten-server/internal/service"
	"kindergarten-server/internal/testutil/authharness"
)

// #166：管理端密码登录使用内建 admin_web 作用域（不是 clients 表的一行）。
// access 必须带 app_id=""、client_scope_kind=admin_web、scope_id=0 与唯一园所，
// refresh 存储沿用 #169 的 JSON 规则并带同一个园所。

func createAdminUser(t *testing.T, h *authharness.Harness, phone, password string, kindergartens ...*domain.Kindergarten) *domain.User {
	t.Helper()
	user := createUser(t, h, phone, password, "")
	for _, kg := range kindergartens {
		kgID := kg.ID
		require.NoError(t, h.DB.Create(&domain.UserRole{
			UserID:           user.ID,
			Role:             "admin",
			KindergartenID:   &kgID,
			KindergartenName: kg.Name,
			AdminRole:        "admin",
		}).Error)
	}
	return user
}

func TestAdminPasswordLoginIssuesAdminWebSession(t *testing.T) {
	h := authharness.New(t)
	kg := seedKindergarten(t, h, "管理端会话园")
	phone := uniquePhone()
	user := createAdminUser(t, h, phone, "Password@123", kg)

	res := loginWithPassword(t, h, phone, "Password@123")
	require.Equal(t, 200, res.Status, string(res.Body))

	var result service.LoginResult
	res.DecodeData(t, &result)
	require.NotEmpty(t, result.Token)
	require.NotEmpty(t, result.RefreshToken)

	claims, err := auth.ParseAccess(h.Secret, result.Token)
	require.NoError(t, err)
	assert.Empty(t, claims.AppID, "管理端不是 clients 表里的小程序客户端")
	assert.Equal(t, "admin_web", claims.ClientScopeKind)
	assert.Zero(t, claims.ClientScopeID)
	assert.Equal(t, []string{"admin"}, claims.Roles)
	assert.Equal(t, kg.ID, claims.KindergartenID, "管理端凭证必须恰好绑定一个园所")

	value, err := auth.ConsumeRefresh(context.Background(), h.RedisClient, result.RefreshToken)
	require.NoError(t, err)
	var stored storedRefresh
	require.NoError(t, json.Unmarshal([]byte(value), &stored), "refresh 必须沿用小程序的 JSON 作用域；实际值=%q", value)
	assert.Equal(t, user.ID, stored.UID)
	assert.Empty(t, stored.AppID)
	assert.Equal(t, "admin_web", stored.ScopeKind)
	assert.Zero(t, stored.ScopeID)
	assert.Equal(t, kg.ID, stored.KindergartenID)
}

// 管理端凭证不属于「待绑定」：即使角色为空也不能被 40302 拦住；
// 同时它没有 app_id，切换园所必须被拒（管理端只有一个园所）。
func TestAdminTokenIsNotUnboundAndCannotSwitch(t *testing.T) {
	h := authharness.New(t)
	kg := seedKindergarten(t, h, "管理端切换园")
	phone := uniquePhone()
	createAdminUser(t, h, phone, "Password@123", kg)

	res := loginWithPassword(t, h, phone, "Password@123")
	require.Equal(t, 200, res.Status, string(res.Body))
	var result service.LoginResult
	res.DecodeData(t, &result)

	claims, err := auth.ParseAccess(h.Secret, result.Token)
	require.NoError(t, err)
	require.Equal(t, "admin_web", claims.ClientScopeKind)

	info := h.Get(t, "/api/v1/admin/kindergarten/info", result.Token)
	assert.Equal(t, 200, info.Status, "管理端凭证必须能进入 admin 业务路由：%s", info.Body)

	ping := h.Get(t, "/api/v1/parent/home/ping", result.Token)
	assert.Equal(t, 200, ping.Status, "管理端不是未绑定账号，不得被 40302 白名单拦截：%s", ping.Body)

	switched := switchRequest(t, h, result.Token, kg.ID, result.RefreshToken)
	assert.Equal(t, 403, switched.Status, "管理端凭证不能切换园所：%s", switched.Body)
}

// access 过期后，管理端用 refresh 换回同一园所的 admin_web 会话；
// 旧 refresh 立即失效（一次性消费），新 refresh 继续轮换。
func TestAdminRefreshRestoresExpiredAccessSession(t *testing.T) {
	h := authharness.New(t)
	kg := seedKindergarten(t, h, "管理端续期园")
	phone := uniquePhone()
	user := createAdminUser(t, h, phone, "Password@123", kg)

	login := loginWithPassword(t, h, phone, "Password@123")
	require.Equal(t, 200, login.Status, string(login.Body))
	var first service.LoginResult
	login.DecodeData(t, &first)

	// 直接签一枚已过期的管理端 access，模拟 15 分钟后客户端发起请求。
	expired, err := auth.SignAdminAccess(h.Secret, fmt.Sprintf("%d", user.ID), []string{"admin"}, kg.ID, -time.Minute)
	require.NoError(t, err)
	unauthorized := h.Get(t, "/api/v1/admin/kindergarten/info", expired)
	require.Equal(t, 401, unauthorized.Status, string(unauthorized.Body))

	refresh := refreshRequest(t, h, first.RefreshToken)
	require.Equal(t, 200, refresh.Status, string(refresh.Body))
	var renewed service.LoginResult
	refresh.DecodeData(t, &renewed)
	require.NotEqual(t, first.RefreshToken, renewed.RefreshToken, "刷新必须轮换 refresh token")

	claims, err := auth.ParseAccess(h.Secret, renewed.Token)
	require.NoError(t, err)
	assert.Equal(t, "admin_web", claims.ClientScopeKind)
	assert.Empty(t, claims.AppID)
	assert.Equal(t, kg.ID, claims.KindergartenID)
	assert.Equal(t, []string{"admin"}, claims.Roles)

	info := h.Get(t, "/api/v1/admin/kindergarten/info", renewed.Token)
	assert.Equal(t, 200, info.Status, "续期后的 access 必须可用：%s", info.Body)

	// 旧 refresh 已被消费，重放必须失败。
	replay := refreshRequest(t, h, first.RefreshToken)
	assert.Equal(t, 401, replay.Status, string(replay.Body))
}

// 改密后旧 refresh 必须作废：密码是管理端唯一的登录凭据，
// 改密通常意味着「怀疑账号泄露」。
func TestAdminRefreshRejectsChangedPassword(t *testing.T) {
	h := authharness.New(t)
	kg := seedKindergarten(t, h, "管理端改密园")
	phone := uniquePhone()
	user := createAdminUser(t, h, phone, "Password@123", kg)

	login := loginWithPassword(t, h, phone, "Password@123")
	require.Equal(t, 200, login.Status, string(login.Body))
	var first service.LoginResult
	login.DecodeData(t, &first)

	newHash, err := service.HashPassword("Changed@123")
	require.NoError(t, err)
	require.NoError(t, h.DB.Model(&domain.User{}).Where("id = ?", user.ID).
		Update("password_hash", newHash).Error)

	resp := refreshRequest(t, h, first.RefreshToken)
	assert.Equal(t, 401, resp.Status, "改密后旧 refresh 必须被拒：%s", resp.Body)
}

// 账号被停用（软删）后，已签发的 refresh 不能再换出新凭证。
func TestAdminRefreshRejectsDisabledAccount(t *testing.T) {
	h := authharness.New(t)
	kg := seedKindergarten(t, h, "管理端停用园")
	phone := uniquePhone()
	user := createAdminUser(t, h, phone, "Password@123", kg)

	login := loginWithPassword(t, h, phone, "Password@123")
	require.Equal(t, 200, login.Status, string(login.Body))
	var first service.LoginResult
	login.DecodeData(t, &first)

	require.NoError(t, h.DB.Delete(&domain.User{}, user.ID).Error)

	resp := refreshRequest(t, h, first.RefreshToken)
	assert.Equal(t, 401, resp.Status, "停用账号的 refresh 必须被拒：%s", resp.Body)
}

// 没有园所绑定的 admin 角色无法确定租户，管理端不得签发会话。
func TestAdminLoginRejectsAdminWithoutKindergarten(t *testing.T) {
	h := authharness.New(t)
	phone := uniquePhone()
	createUser(t, h, phone, "Password@123", "admin")

	res := loginWithPassword(t, h, phone, "Password@123")
	assert.Equal(t, 403, res.Status, "无园所管理员不得登录管理端：%s", res.Body)
}

// 一个手机号只能是一个园所的管理员：多园 admin 会让 token 的
// kindergarten_id 变成任意选择，必须在登录时拒绝。
func TestAdminLoginRejectsMultipleKindergartenAdmins(t *testing.T) {
	h := authharness.New(t)
	kgA := seedKindergarten(t, h, "管理端甲园")
	kgB := seedKindergarten(t, h, "管理端乙园")
	phone := uniquePhone()
	createAdminUser(t, h, phone, "Password@123", kgA, kgB)

	res := loginWithPassword(t, h, phone, "Password@123")
	assert.Equal(t, 403, res.Status, "多园管理员不得登录管理端：%s", res.Body)
}

// 签发后又被加上另一个园所的管理员身份：旧 refresh 必须停止续签，
// 不能悄悄把会话迁到另一个园。
func TestAdminRefreshRejectsAddedKindergartenMembership(t *testing.T) {
	h := authharness.New(t)
	kgA := seedKindergarten(t, h, "管理端续期甲园")
	kgB := seedKindergarten(t, h, "管理端续期乙园")
	phone := uniquePhone()
	user := createAdminUser(t, h, phone, "Password@123", kgA)

	login := loginWithPassword(t, h, phone, "Password@123")
	require.Equal(t, 200, login.Status, string(login.Body))
	var first service.LoginResult
	login.DecodeData(t, &first)

	kgBID := kgB.ID
	require.NoError(t, h.DB.Create(&domain.UserRole{
		UserID:           user.ID,
		Role:             "admin",
		KindergartenID:   &kgBID,
		KindergartenName: kgB.Name,
		AdminRole:        "admin",
	}).Error)

	resp := refreshRequest(t, h, first.RefreshToken)
	assert.Equal(t, 401, resp.Status, "多园管理员身份出现后不得继续续期：%s", resp.Body)
}

// 建管理员时就要挡住「一个手机号管理两个园」：否则会建出一个永远无法登录
// 管理端的账号（登录会被 admin_web 的唯一园所约束拒绝）。
func TestCreateAdminRejectsPhoneAdminInAnotherKindergarten(t *testing.T) {
	h := authharness.New(t)
	kgA := seedKindergarten(t, h, "创建管理员甲园")
	kgB := seedKindergarten(t, h, "创建管理员乙园")
	t.Cleanup(func() { h.HardDelete(t, &domain.AdminRole{}, "kindergarten_id = ?", kgB.ID) })

	phone := uniquePhone()
	user := createAdminUser(t, h, phone, "Password@123", kgA)

	svc := service.NewAdminService(
		h.DB,
		repository.NewUserRepo(h.DB),
		repository.NewUserRoleRepo(h.DB),
		repository.NewAdminRepo(h.DB),
		repository.NewKindergartenRepo(h.DB),
		repository.NewOperationLogRepo(h.DB),
	)
	_, err := svc.CreateAdmin(context.Background(), kgB.ID, 1, service.CreateAdminReq{
		Phone:    phone,
		Nickname: "跨园管理员",
		Password: "Password@123",
	}, "", "")
	require.Error(t, err)
	assert.Equal(t, "DUPLICATE_ADMIN_PHONE", err.Error())

	var count int64
	require.NoError(t, h.DB.Model(&domain.UserRole{}).
		Where("user_id = ? AND role = ?", user.ID, "admin").Count(&count).Error)
	assert.Equal(t, int64(1), count, "不得在第二个园所再建 admin 角色")
}
