package authharness_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/internal/testutil/authharness"
)

// --- helpers ---

// seedKindergartenEntry 建一个带通道与公开码的园所。#167 需要 public_code 定位目标园，
// 并需要一个可切换成 dedicated 的通道来验证「客户端作用域 ↔ 园所通道」矩阵。
func seedKindergartenEntry(t *testing.T, h *authharness.Harness, name, channel, publicCode string) *domain.Kindergarten {
	t.Helper()
	kg := &domain.Kindergarten{
		Name:       fmt.Sprintf("%s-%d", name, time.Now().UnixNano()),
		AppChannel: channel,
		PublicCode: publicCode,
	}
	require.NoError(t, h.DB.Create(kg).Error)
	t.Cleanup(func() { h.HardDelete(t, &domain.Kindergarten{}, "id = ?", kg.ID) })
	return kg
}

// seedParentAccount 模拟园所预先录入的家长账号：手机号 + 每园 parent 角色 + 家长档案。
// 绑定必须保留这条账号（角色、孩子、档案都在它身上），而不是新建。
func seedParentAccount(t *testing.T, h *authharness.Harness, phone string, kgs ...*domain.Kindergarten) *domain.User {
	t.Helper()
	u := &domain.User{Phone: phone, Nickname: "parent-" + phone}
	require.NoError(t, h.DB.Create(u).Error)
	t.Cleanup(func() {
		h.HardDelete(t, &domain.ParentProfile{}, "user_id = ?", u.ID)
		h.HardDelete(t, &domain.UserRole{}, "user_id = ?", u.ID)
		h.HardDelete(t, &domain.UserIdentity{}, "user_id = ?", u.ID)
		h.HardDelete(t, &domain.User{}, "id = ?", u.ID)
	})
	for _, kg := range kgs {
		kgID := kg.ID
		require.NoError(t, h.DB.Create(&domain.UserRole{
			UserID:           u.ID,
			Role:             domain.ClientRoleParent,
			KindergartenID:   &kgID,
			KindergartenName: kg.Name,
		}).Error)
		require.NoError(t, h.DB.Create(&domain.ParentProfile{
			UserID:         u.ID,
			KindergartenID: kg.ID,
			RealName:       "家长" + phone,
			Status:         1,
		}).Error)
	}
	return u
}

// pendingLogin 走真实的微信登录拿到「待绑定 token」：无角色、无园所。
func pendingLogin(t *testing.T, h *authharness.Harness, appID, openID string) *service.LoginResult {
	t.Helper()
	code := uniqueCode("bind-login-" + openID)
	h.WeChat.StubCode2Session(code, openID, "")
	return loginOK(t, h, appID, code)
}

// cleanupBindUsers 清理一次绑定用例触及的账号：待绑定账号（成功后已被硬删）、
// 预创建账号、以及身份当前指向的账号。父档案必须先删，否则 FK RESTRICT 会挡住
// users 的硬删。
func cleanupBindUsers(t *testing.T, h *authharness.Harness, appID, openID string, extraIDs ...uint64) {
	t.Helper()
	t.Cleanup(func() {
		ids := make(map[uint64]bool, len(extraIDs)+1)
		for _, id := range extraIDs {
			if id != 0 {
				ids[id] = true
			}
		}
		var identity domain.UserIdentity
		if err := h.DB.Where("app_id = ? AND open_id = ?", appID, openID).First(&identity).Error; err == nil {
			ids[identity.UserID] = true
		}
		for id := range ids {
			// 幼儿关联也是 FK RESTRICT，且可能由别处先建（如支持视图用例），
			// 清理时统一先断开，避免清理顺序依赖播种顺序。
			h.HardDelete(t, &domain.StudentParent{}, "user_id = ?", id)
			h.HardDelete(t, &domain.ParentProfile{}, "user_id = ?", id)
			h.HardDelete(t, &domain.UserRole{}, "user_id = ?", id)
			h.HardDelete(t, &domain.UserIdentity{}, "user_id = ?", id)
			h.HardDelete(t, &domain.User{}, "id = ?", id)
		}
	})
}

func uniquePublicCode(scene string) string {
	return fmt.Sprintf("kg%s%d", scene, time.Now().UnixNano()%1_000_000_000)
}

// --- tests ---

// 目标园绑定：手机号在 A、B 两园都有预创建家长账号时，用 A 的 public_code 绑定，
// 初始落点必须是 A，同时响应要列出全部 shared 成员关系（不依赖 #169）。
// 绑定成功后待绑定账号必须已被硬删，身份挂到预创建账号，新 token 能进家长业务。
func TestBindParentPhoneTargetsKindergartenAndReturnsAllSharedMemberships(t *testing.T) {
	h := authharness.New(t)

	appID := uniqueAppID("bind-target")
	seedClient(t, h, appID, domain.ClientRoleParent, domain.ClientStatusEnabled)

	kgA := seedKindergartenEntry(t, h, "bind-target-a", domain.AppChannelShared, uniquePublicCode("a"))
	kgB := seedKindergartenEntry(t, h, "bind-target-b", domain.AppChannelShared, uniquePublicCode("b"))
	phone := uniquePhone()
	targetUser := seedParentAccount(t, h, phone, kgA, kgB)

	openID := uniqueOpenID("bind-target")
	pending := pendingLogin(t, h, appID, openID)
	cleanupBindUsers(t, h, appID, openID, pending.User.ID, targetUser.ID)

	phoneCode := uniqueCode("bind-phone-target")
	h.WeChat.StubPhoneNumber(phoneCode, phone)

	resp := h.PostJSONWithToken(t, "/api/v1/common/user/bind-parent-phone", pending.Token, map[string]any{
		"wx_phone_code": phoneCode,
		"public_code":   kgA.PublicCode,
	})
	require.Equal(t, http.StatusOK, resp.Status, string(resp.Body))

	var result service.ParentBindPhoneResult
	resp.DecodeData(t, &result)
	require.NotNil(t, result.LoginResult, "绑定响应必须带新的会话凭证")
	require.NotEmpty(t, result.Token)

	claims, err := auth.ParseAccess(h.Secret, result.Token)
	require.NoError(t, err)
	assert.Equal(t, targetUser.ID, result.User.ID, "绑定后身份应落到预创建账号")
	assert.Equal(t, []string{domain.ClientRoleParent}, claims.Roles)
	assert.Equal(t, kgA.ID, claims.KindergartenID, "初始落点必须是目标园")

	gotKG := make(map[uint64]bool, len(result.Kindergartens))
	for _, m := range result.Kindergartens {
		gotKG[m.KindergartenID] = true
	}
	assert.ElementsMatch(t, []uint64{kgA.ID, kgB.ID}, keysOf(gotKG),
		"响应必须列出该账号全部 shared 成员关系")

	// 待绑定账号必须被硬删（不是软删），否则 phone_active/身份唯一键会留下脏行。
	var pendingRows int64
	require.NoError(t, h.DB.Unscoped().Model(&domain.User{}).Where("id = ?", pending.User.ID).Count(&pendingRows).Error)
	assert.Zero(t, pendingRows, "待绑定账号必须被硬删")

	var identity domain.UserIdentity
	require.NoError(t, h.DB.Where("app_id = ? AND open_id = ?", appID, openID).First(&identity).Error)
	assert.Equal(t, targetUser.ID, identity.UserID, "客户端身份必须挂到预创建账号")

	var profileCount int64
	require.NoError(t, h.DB.Model(&domain.ParentProfile{}).
		Where("user_id = ? AND kindergarten_id = ?", targetUser.ID, kgA.ID).Count(&profileCount).Error)
	assert.EqualValues(t, 1, profileCount, "已有家长档案必须保留")

	// 新 token 必须真的能进家长业务（待绑定 token 在此路由会被 40302 拦住）。
	ping := h.Get(t, "/api/v1/parent/home/ping", result.Token)
	require.Equal(t, http.StatusOK, ping.Status, string(ping.Body))
}

func keysOf(m map[uint64]bool) []uint64 {
	keys := make([]uint64, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// 无 scene（不带 public_code）：按手机号在 shared 园所里找到预创建账号即绑定，
// 落点确定性取最小园所 ID，成员列表包含全部 shared 园所。
func TestBindParentPhoneWithoutSceneFindsSharedMemberships(t *testing.T) {
	h := authharness.New(t)

	appID := uniqueAppID("bind-noscene")
	seedClient(t, h, appID, domain.ClientRoleParent, domain.ClientStatusEnabled)

	kgA := seedKindergartenEntry(t, h, "bind-noscene-a", domain.AppChannelShared, uniquePublicCode("na"))
	kgB := seedKindergartenEntry(t, h, "bind-noscene-b", domain.AppChannelShared, uniquePublicCode("nb"))
	phone := uniquePhone()
	targetUser := seedParentAccount(t, h, phone, kgA, kgB)

	openID := uniqueOpenID("bind-noscene")
	pending := pendingLogin(t, h, appID, openID)
	cleanupBindUsers(t, h, appID, openID, pending.User.ID, targetUser.ID)

	phoneCode := uniqueCode("bind-noscene-phone")
	h.WeChat.StubPhoneNumber(phoneCode, phone)

	resp := h.PostJSONWithToken(t, "/api/v1/common/user/bind-parent-phone", pending.Token, map[string]any{
		"wx_phone_code": phoneCode,
	})
	require.Equal(t, http.StatusOK, resp.Status, string(resp.Body))

	var result service.ParentBindPhoneResult
	resp.DecodeData(t, &result)
	claims, err := auth.ParseAccess(h.Secret, result.Token)
	require.NoError(t, err)
	assert.Equal(t, targetUser.ID, result.User.ID)
	assert.Equal(t, kgA.ID, claims.KindergartenID, "无 scene 时落点取最小园所 ID")

	got := make(map[uint64]bool, len(result.Kindergartens))
	for _, m := range result.Kindergartens {
		got[m.KindergartenID] = true
	}
	assert.ElementsMatch(t, []uint64{kgA.ID, kgB.ID}, keysOf(got))
}

// 手机号没有任何园所录入时：明确报「园所尚未录入」，且绝不新建账号，
// 待绑定账号与身份保持原样（家长可以等园所录入后重试）。
func TestBindParentPhoneUnknownPhoneCreatesNoAccount(t *testing.T) {
	h := authharness.New(t)

	appID := uniqueAppID("bind-unknown")
	seedClient(t, h, appID, domain.ClientRoleParent, domain.ClientStatusEnabled)
	kgB := seedKindergartenEntry(t, h, "bind-unknown-b", domain.AppChannelShared, uniquePublicCode("ub"))
	phone := uniquePhone()

	openID := uniqueOpenID("bind-unknown")
	pending := pendingLogin(t, h, appID, openID)
	cleanupBindUsers(t, h, appID, openID, pending.User.ID)

	phoneCode := uniqueCode("bind-unknown-phone")
	h.WeChat.StubPhoneNumber(phoneCode, phone)

	// 无 scene
	resp := h.PostJSONWithToken(t, "/api/v1/common/user/bind-parent-phone", pending.Token, map[string]any{
		"wx_phone_code": phoneCode,
	})
	require.Equal(t, http.StatusBadRequest, resp.Status, string(resp.Body))
	assert.Equal(t, 40031, resp.EnvelopeCode(t), string(resp.Body))

	// 指定目标园，但该园没有这个手机号的家长角色
	resp = h.PostJSONWithToken(t, "/api/v1/common/user/bind-parent-phone", pending.Token, map[string]any{
		"wx_phone_code": phoneCode,
		"public_code":   kgB.PublicCode,
	})
	require.Equal(t, http.StatusBadRequest, resp.Status, string(resp.Body))
	assert.Equal(t, 40031, resp.EnvelopeCode(t), string(resp.Body))

	var userCount int64
	require.NoError(t, h.DB.Model(&domain.User{}).Where("phone_active = ?", phone).Count(&userCount).Error)
	assert.Zero(t, userCount, "绑定失败绝不能新建家长账号")

	var identity domain.UserIdentity
	require.NoError(t, h.DB.Where("app_id = ? AND open_id = ?", appID, openID).First(&identity).Error)
	assert.Equal(t, pending.User.ID, identity.UserID, "失败后身份必须仍指向待绑定账号")

	var alive int64
	require.NoError(t, h.DB.Model(&domain.User{}).Where("id = ?", pending.User.ID).Count(&alive).Error)
	assert.EqualValues(t, 1, alive, "失败后待绑定账号不能被删除")
}

// 手机号只在 B 园录入时，用 A 园的 public_code 绑定必须失败：
// 目标园校验的是「该园是否录入这个手机号」，而不是全局存在即可。
func TestBindParentPhoneTargetKindergartenMustHaveEnrolledPhone(t *testing.T) {
	h := authharness.New(t)

	appID := uniqueAppID("bind-wrong-target")
	seedClient(t, h, appID, domain.ClientRoleParent, domain.ClientStatusEnabled)
	kgA := seedKindergartenEntry(t, h, "bind-wrong-a", domain.AppChannelShared, uniquePublicCode("wa"))
	kgB := seedKindergartenEntry(t, h, "bind-wrong-b", domain.AppChannelShared, uniquePublicCode("wb"))
	phone := uniquePhone()
	targetUser := seedParentAccount(t, h, phone, kgB)

	openID := uniqueOpenID("bind-wrong-target")
	pending := pendingLogin(t, h, appID, openID)
	cleanupBindUsers(t, h, appID, openID, pending.User.ID, targetUser.ID)

	phoneCode := uniqueCode("bind-wrong-phone")
	h.WeChat.StubPhoneNumber(phoneCode, phone)

	resp := h.PostJSONWithToken(t, "/api/v1/common/user/bind-parent-phone", pending.Token, map[string]any{
		"wx_phone_code": phoneCode,
		"public_code":   kgA.PublicCode,
	})
	require.Equal(t, http.StatusBadRequest, resp.Status, string(resp.Body))
	assert.Equal(t, 40031, resp.EnvelopeCode(t), string(resp.Body))

	var identity domain.UserIdentity
	require.NoError(t, h.DB.Where("app_id = ? AND open_id = ?", appID, openID).First(&identity).Error)
	assert.Equal(t, pending.User.ID, identity.UserID, "失败后身份仍指向待绑定账号")
}

// 找不到 public_code 时返回 404，且不触碰任何账号。
func TestBindParentPhoneUnknownPublicCode(t *testing.T) {
	h := authharness.New(t)

	appID := uniqueAppID("bind-bad-code")
	seedClient(t, h, appID, domain.ClientRoleParent, domain.ClientStatusEnabled)
	openID := uniqueOpenID("bind-bad-code")
	pending := pendingLogin(t, h, appID, openID)
	cleanupBindUsers(t, h, appID, openID, pending.User.ID)

	phoneCode := uniqueCode("bind-bad-code-phone")
	h.WeChat.StubPhoneNumber(phoneCode, uniquePhone())

	resp := h.PostJSONWithToken(t, "/api/v1/common/user/bind-parent-phone", pending.Token, map[string]any{
		"wx_phone_code": phoneCode,
		"public_code":   "no-such-code-167",
	})
	require.Equal(t, http.StatusNotFound, resp.Status, string(resp.Body))

	var alive int64
	require.NoError(t, h.DB.Model(&domain.User{}).Where("id = ?", pending.User.ID).Count(&alive).Error)
	assert.EqualValues(t, 1, alive)
}

// 共用（shared）家长端不能绑定专属（dedicated）园所：通道矩阵不匹配时
// 必须在查询手机号之前就拒绝。
func TestBindParentPhoneRejectsChannelMismatch(t *testing.T) {
	h := authharness.New(t)

	appID := uniqueAppID("bind-channel")
	seedClient(t, h, appID, domain.ClientRoleParent, domain.ClientStatusEnabled)
	kgDedicated := seedKindergartenEntry(t, h, "bind-channel-dedicated", domain.AppChannelDedicated, uniquePublicCode("dc"))
	phone := uniquePhone()
	targetUser := seedParentAccount(t, h, phone, kgDedicated)

	openID := uniqueOpenID("bind-channel")
	pending := pendingLogin(t, h, appID, openID)
	cleanupBindUsers(t, h, appID, openID, pending.User.ID, targetUser.ID)

	phoneCode := uniqueCode("bind-channel-phone")
	h.WeChat.StubPhoneNumber(phoneCode, phone)

	resp := h.PostJSONWithToken(t, "/api/v1/common/user/bind-parent-phone", pending.Token, map[string]any{
		"wx_phone_code": phoneCode,
		"public_code":   kgDedicated.PublicCode,
	})
	require.Equal(t, http.StatusBadRequest, resp.Status, string(resp.Body))
	assert.Equal(t, 40033, resp.EnvelopeCode(t), string(resp.Body))
}

// 绑定只服务家长端客户端；老师端即便手机号在园所已录入也不能走这条通道。
func TestBindParentPhoneRejectsTeacherClient(t *testing.T) {
	h := authharness.New(t)

	appID := uniqueAppID("bind-teacher")
	seedClient(t, h, appID, domain.ClientRoleTeacher, domain.ClientStatusEnabled)
	kg := seedKindergartenEntry(t, h, "bind-teacher-kg", domain.AppChannelShared, uniquePublicCode("tc"))
	phone := uniquePhone()
	targetUser := seedParentAccount(t, h, phone, kg)

	openID := uniqueOpenID("bind-teacher")
	pending := pendingLogin(t, h, appID, openID)
	cleanupBindUsers(t, h, appID, openID, pending.User.ID, targetUser.ID)

	phoneCode := uniqueCode("bind-teacher-phone")
	h.WeChat.StubPhoneNumber(phoneCode, phone)

	resp := h.PostJSONWithToken(t, "/api/v1/common/user/bind-parent-phone", pending.Token, map[string]any{
		"wx_phone_code": phoneCode,
		"public_code":   kg.PublicCode,
	})
	require.Equal(t, http.StatusForbidden, resp.Status, string(resp.Body))

	var identity domain.UserIdentity
	require.NoError(t, h.DB.Where("app_id = ? AND open_id = ?", appID, openID).First(&identity).Error)
	assert.Equal(t, pending.User.ID, identity.UserID, "老师端请求不能触碰任何家长账号")
}

// 预创建账号在同一客户端已绑定另一个微信时，第二个微信必须被拒绝：
// 不能悄悄顶掉先绑的人，也不能删掉第二个待绑定账号（本人还可以改用正确手机号）。
func TestBindParentPhoneRejectsSecondWeChat(t *testing.T) {
	h := authharness.New(t)

	appID := uniqueAppID("bind-second-wx")
	seedClient(t, h, appID, domain.ClientRoleParent, domain.ClientStatusEnabled)
	kg := seedKindergartenEntry(t, h, "bind-second-wx-kg", domain.AppChannelShared, uniquePublicCode("sw"))
	phone := uniquePhone()
	targetUser := seedParentAccount(t, h, phone, kg)

	openA := uniqueOpenID("bind-second-a")
	pendingA := pendingLogin(t, h, appID, openA)
	openB := uniqueOpenID("bind-second-b")
	pendingB := pendingLogin(t, h, appID, openB)
	cleanupBindUsers(t, h, appID, openA, pendingA.User.ID, pendingB.User.ID, targetUser.ID)

	codeA := uniqueCode("bind-second-phone-a")
	h.WeChat.StubPhoneNumber(codeA, phone)
	resp := h.PostJSONWithToken(t, "/api/v1/common/user/bind-parent-phone", pendingA.Token, map[string]any{
		"wx_phone_code": codeA,
		"public_code":   kg.PublicCode,
	})
	require.Equal(t, http.StatusOK, resp.Status, string(resp.Body))

	codeB := uniqueCode("bind-second-phone-b")
	h.WeChat.StubPhoneNumber(codeB, phone)
	resp = h.PostJSONWithToken(t, "/api/v1/common/user/bind-parent-phone", pendingB.Token, map[string]any{
		"wx_phone_code": codeB,
		"public_code":   kg.PublicCode,
	})
	require.Equal(t, http.StatusBadRequest, resp.Status, string(resp.Body))
	assert.Equal(t, 40032, resp.EnvelopeCode(t), string(resp.Body))

	var identityA domain.UserIdentity
	require.NoError(t, h.DB.Where("app_id = ? AND open_id = ?", appID, openA).First(&identityA).Error)
	assert.Equal(t, targetUser.ID, identityA.UserID, "先绑定的微信必须保持绑定")

	var identityB domain.UserIdentity
	require.NoError(t, h.DB.Where("app_id = ? AND open_id = ?", appID, openB).First(&identityB).Error)
	assert.Equal(t, pendingB.User.ID, identityB.UserID, "被拒绝的微信身份必须保持原样")

	var pendingAlive int64
	require.NoError(t, h.DB.Model(&domain.User{}).Where("id = ?", pendingB.User.ID).Count(&pendingAlive).Error)
	assert.EqualValues(t, 1, pendingAlive, "被拒绝时不能删除第二个待绑定账号")
}

// 同一微信重复绑定必须幂等：返回成功、账号不变、档案与身份都不重复。
// 应用重启或网络重试走的就是这条路径。
func TestBindParentPhoneRetryIsIdempotent(t *testing.T) {
	h := authharness.New(t)

	appID := uniqueAppID("bind-retry")
	seedClient(t, h, appID, domain.ClientRoleParent, domain.ClientStatusEnabled)
	kg := seedKindergartenEntry(t, h, "bind-retry-kg", domain.AppChannelShared, uniquePublicCode("rt"))
	phone := uniquePhone()
	targetUser := seedParentAccount(t, h, phone, kg)

	openID := uniqueOpenID("bind-retry")
	pending := pendingLogin(t, h, appID, openID)
	cleanupBindUsers(t, h, appID, openID, pending.User.ID, targetUser.ID)

	code1 := uniqueCode("bind-retry-phone-1")
	h.WeChat.StubPhoneNumber(code1, phone)
	resp := h.PostJSONWithToken(t, "/api/v1/common/user/bind-parent-phone", pending.Token, map[string]any{
		"wx_phone_code": code1,
		"public_code":   kg.PublicCode,
	})
	require.Equal(t, http.StatusOK, resp.Status, string(resp.Body))
	var first service.ParentBindPhoneResult
	resp.DecodeData(t, &first)

	code2 := uniqueCode("bind-retry-phone-2")
	h.WeChat.StubPhoneNumber(code2, phone)
	resp = h.PostJSONWithToken(t, "/api/v1/common/user/bind-parent-phone", first.Token, map[string]any{
		"wx_phone_code": code2,
		"public_code":   kg.PublicCode,
	})
	require.Equal(t, http.StatusOK, resp.Status, string(resp.Body))
	var second service.ParentBindPhoneResult
	resp.DecodeData(t, &second)
	assert.Equal(t, first.User.ID, second.User.ID, "重试必须落到同一个账号")

	var profileCount int64
	require.NoError(t, h.DB.Model(&domain.ParentProfile{}).
		Where("user_id = ? AND kindergarten_id = ?", targetUser.ID, kg.ID).Count(&profileCount).Error)
	assert.EqualValues(t, 1, profileCount, "重试不能重复创建家长档案")

	var identityCount int64
	require.NoError(t, h.DB.Model(&domain.UserIdentity{}).
		Where("user_id = ? AND app_id = ?", targetUser.ID, appID).Count(&identityCount).Error)
	assert.EqualValues(t, 1, identityCount, "重试不能产生第二条身份")
}

// 预创建账号缺家长档案时，绑定必须补一条；否则落点园所里看不到孩子。
func TestBindParentPhoneCreatesMissingParentProfile(t *testing.T) {
	h := authharness.New(t)

	appID := uniqueAppID("bind-no-profile")
	seedClient(t, h, appID, domain.ClientRoleParent, domain.ClientStatusEnabled)
	kg := seedKindergartenEntry(t, h, "bind-no-profile-kg", domain.AppChannelShared, uniquePublicCode("np"))
	phone := uniquePhone()

	// 只有账号和角色，没有 ParentProfile。
	targetUser := &domain.User{Phone: phone, Nickname: "parent-" + phone}
	require.NoError(t, h.DB.Create(targetUser).Error)
	kgID := kg.ID
	require.NoError(t, h.DB.Create(&domain.UserRole{
		UserID:           targetUser.ID,
		Role:             domain.ClientRoleParent,
		KindergartenID:   &kgID,
		KindergartenName: kg.Name,
	}).Error)

	openID := uniqueOpenID("bind-no-profile")
	pending := pendingLogin(t, h, appID, openID)
	cleanupBindUsers(t, h, appID, openID, pending.User.ID, targetUser.ID)

	var before int64
	require.NoError(t, h.DB.Model(&domain.ParentProfile{}).Where("user_id = ?", targetUser.ID).Count(&before).Error)
	require.Zero(t, before, "前置条件：账号没有家长档案")

	phoneCode := uniqueCode("bind-no-profile-phone")
	h.WeChat.StubPhoneNumber(phoneCode, phone)
	resp := h.PostJSONWithToken(t, "/api/v1/common/user/bind-parent-phone", pending.Token, map[string]any{
		"wx_phone_code": phoneCode,
		"public_code":   kg.PublicCode,
	})
	require.Equal(t, http.StatusOK, resp.Status, string(resp.Body))

	var after int64
	require.NoError(t, h.DB.Model(&domain.ParentProfile{}).
		Where("user_id = ? AND kindergarten_id = ?", targetUser.ID, kg.ID).Count(&after).Error)
	assert.EqualValues(t, 1, after, "绑定必须补齐家长档案")
}

// 绑定接口按 app_id + IP（+ 手机号）限流：一分钟内第 21 次请求必须 429，
// 不能继续用一次性手机号 code 试探手机号是否被园所录入。
func TestBindParentPhoneIsRateLimited(t *testing.T) {
	h := authharness.New(t)

	appID := uniqueAppID("bind-ratelimit")
	seedClient(t, h, appID, domain.ClientRoleParent, domain.ClientStatusEnabled)

	openID := uniqueOpenID("bind-ratelimit")
	pending := pendingLogin(t, h, appID, openID)
	cleanupBindUsers(t, h, appID, openID, pending.User.ID)

	phoneCode := uniqueCode("bind-ratelimit-phone")
	h.WeChat.StubPhoneNumber(phoneCode, uniquePhone()) // 未录入，前 20 次都是 40031

	payload := map[string]any{"wx_phone_code": phoneCode}
	for i := 1; i <= middleware.BindPhoneMaxPerMinute; i++ {
		resp := h.PostJSONWithToken(t, "/api/v1/common/user/bind-parent-phone", pending.Token, payload)
		require.Equal(t, http.StatusBadRequest, resp.Status, "第 %d 次应在限流之前：%s", i, resp.Body)
		assert.Equal(t, 40031, resp.EnvelopeCode(t), "第 %d 次：%s", i, resp.Body)
	}
	resp := h.PostJSONWithToken(t, "/api/v1/common/user/bind-parent-phone", pending.Token, payload)
	assert.Equal(t, http.StatusTooManyRequests, resp.Status, string(resp.Body))
}
