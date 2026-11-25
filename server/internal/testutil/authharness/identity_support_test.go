package authharness_test

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/testutil/authharness"
)

// #172：园所管理员 / 平台支持对账号做「换微信」与「手机号录入更正」。
//
// 这两件事都不能悄悄删账号或合并账号：解绑只释放客户端身份，更正只在
// 「先解绑错误身份（如有）→ 改手机号 → 正确家长重新绑定」的顺序下进行。

// --- helpers ---

// seedSupportChild 给家长挂一个孩子（班级 + 学生 + 关系），用来证明
// 解绑/更正不会动到幼儿关联，以及更正后正确家长真的能在业务接口看到孩子。
func seedSupportChild(t *testing.T, h *authharness.Harness, kg *domain.Kindergarten, parentID uint64, name string) *domain.Student {
	t.Helper()
	cls := &domain.Class{KindergartenID: kg.ID, Name: "班级-" + name}
	require.NoError(t, h.DB.Create(cls).Error)
	t.Cleanup(func() { h.HardDelete(t, &domain.Class{}, "id = ?", cls.ID) })

	st := &domain.Student{
		KindergartenID: kg.ID,
		ClassID:        cls.ID,
		Name:           name,
		Status:         1,
		// 非空时间列不能留 Go 零值：MySQL 严格模式拒绝 '0000-00-00'。
		Birthday:   time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		EnrollDate: time.Date(2023, 9, 1, 0, 0, 0, 0, time.UTC),
	}
	require.NoError(t, h.DB.Create(st).Error)
	t.Cleanup(func() {
		h.HardDelete(t, &domain.StudentParent{}, "student_id = ?", st.ID)
		h.HardDelete(t, &domain.Student{}, "id = ?", st.ID)
	})
	require.NoError(t, h.DB.Create(&domain.StudentParent{
		StudentID: st.ID,
		UserID:    parentID,
		Relation:  domain.RelationMother,
		IsPrimary: true,
	}).Error)
	return st
}

// uniqueWxAppID 生成形如真实小程序 app_id 的短标识（wx + 16 位十六进制）。
// 不用「wx-175-场景-19 位纳秒」的朴素写法：审计自由文本会把 ≥18 位数字串按
// 身份证号掩码，那个形状在断言里会失真。
var wxAppIDSeq atomic.Uint64

func uniqueWxAppID() string {
	return fmt.Sprintf("wx%016x", uint64(time.Now().UnixNano())+wxAppIDSeq.Add(1))
}

// adminSupportToken 签发园所管理员 token。审计里的操作人就是这里的 user id，
// 不需要先把管理员账号写进 users——鉴权只认签名与角色。
func adminSupportToken(t *testing.T, h *authharness.Harness, kindergartenID uint64) string {
	t.Helper()
	token, err := auth.SignAccess(h.Secret, "990001", []string{"admin"}, kindergartenID, time.Hour)
	require.NoError(t, err)
	return token
}

// seedIdentityFor 直接写一条客户端身份（模拟「另一个微信已绑定」），清理走硬删。
func seedIdentityFor(t *testing.T, h *authharness.Harness, userID uint64, appID, openID string) *domain.UserIdentity {
	t.Helper()
	identity := &domain.UserIdentity{UserID: userID, AppID: appID, OpenID: openID}
	require.NoError(t, h.DB.Create(identity).Error)
	t.Cleanup(func() { h.HardDelete(t, &domain.UserIdentity{}, "id = ?", identity.ID) })
	return identity
}

// strangerBind 是「错误手机号被别人的微信绑走」的结果快照。
type strangerBind struct {
	UserID       uint64
	IdentityID   uint64
	AccessToken  string
	RefreshToken string
}

// bindStranger 模拟「错误手机号被别人的微信绑走」：待绑定账号用
// 真实微信登录拿到 token，再用该手机号的授权 code 完成绑定。
func bindStranger(t *testing.T, h *authharness.Harness, appID, openID, phone, publicCode string) *strangerBind {
	t.Helper()
	pending := pendingLogin(t, h, appID, openID)
	cleanupBindUsers(t, h, appID, openID, pending.User.ID)

	phoneCode := uniqueCode("support-bind")
	h.WeChat.StubPhoneNumber(phoneCode, phone)
	resp := h.PostJSONWithToken(t, "/api/v1/common/user/bind-parent-phone", pending.Token, map[string]any{
		"wx_phone_code": phoneCode,
		"public_code":   publicCode,
	})
	require.Equal(t, http.StatusOK, resp.Status, string(resp.Body))

	var login struct {
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}
	resp.DecodeData(t, &login)

	var identity domain.UserIdentity
	require.NoError(t, h.DB.Where("app_id = ? AND open_id = ?", appID, openID).First(&identity).Error)
	return &strangerBind{
		UserID:       identity.UserID,
		IdentityID:   identity.ID,
		AccessToken:  login.Token,
		RefreshToken: login.RefreshToken,
	}
}

// supportIdentityJSON / supportViewJSON 是 #172 支持视图的公开契约，
// 在测试里独立声明，避免直接依赖实现类型。
type supportIdentityJSON struct {
	ID           uint64 `json:"id"`
	AppID        string `json:"app_id"`
	OpenIDMasked string `json:"open_id_masked"`
}

type supportMembershipJSON struct {
	KindergartenID   uint64 `json:"kindergarten_id"`
	KindergartenName string `json:"kindergarten_name"`
	Role             string `json:"role"`
	ClassID          uint64 `json:"class_id"`
	ClassName        string `json:"class_name"`
}

type supportViewJSON struct {
	UserID      uint64                  `json:"user_id"`
	Nickname    string                  `json:"nickname"`
	Phone       string                  `json:"phone"`
	Identities  []supportIdentityJSON   `json:"identities"`
	Memberships []supportMembershipJSON `json:"memberships"`
}

type unbindRespJSON struct {
	UserID            uint64                `json:"user_id"`
	UnboundIdentityID uint64                `json:"unbound_identity_id"`
	Identities        []supportIdentityJSON `json:"identities"`
}

// --- slice 1: 解绑只释放客户端身份 ---

func TestSupportUnbindRemovesOnlyIdentityAndWritesAudit(t *testing.T) {
	h := authharness.New(t)

	appID := uniqueWxAppID()
	seedClient(t, h, appID, domain.ClientRoleParent, domain.ClientStatusEnabled)
	otherAppID := uniqueWxAppID()
	seedClient(t, h, otherAppID, domain.ClientRoleParent, domain.ClientStatusEnabled)

	kg := seedKindergartenEntry(t, h, "support-unbind-kg", domain.AppChannelShared, uniquePublicCode("su"))
	phone := uniquePhone()
	target := seedParentAccount(t, h, phone, kg)
	child := seedSupportChild(t, h, kg, target.ID, "小明")

	openID := uniqueOpenID("support-unbind")
	wrongBind := bindStranger(t, h, appID, openID, phone, kg.PublicCode)
	require.Equal(t, target.ID, wrongBind.UserID, "前置条件：错误微信已绑到预创建账号")

	// 同一账号在另一个（老师端）客户端也有一条身份：解绑必须只作用于指定的那一条。
	otherIdentity := seedIdentityFor(t, h, target.ID, otherAppID, uniqueOpenID("support-unbind-other"))

	adminToken := adminSupportToken(t, h, kg.ID)
	resp := h.Do(t, http.MethodDelete,
		fmt.Sprintf("/api/v1/admin/parents/%d/identities/%d", target.ID, wrongBind.IdentityID),
		adminToken, nil)
	require.Equal(t, http.StatusOK, resp.Status, string(resp.Body))

	var unbindResp unbindRespJSON
	resp.DecodeData(t, &unbindResp)
	assert.Equal(t, target.ID, unbindResp.UserID)
	assert.Equal(t, wrongBind.IdentityID, unbindResp.UnboundIdentityID)
	require.Len(t, unbindResp.Identities, 1)
	assert.Equal(t, otherIdentity.ID, unbindResp.Identities[0].ID)

	// 解绑必须硬删：软删行会继续占用 (app_id, open_id) 与 (user_id, app_id) 唯一键，
	// 同一微信将来就再也绑不回来。
	var gone int64
	require.NoError(t, h.DB.Unscoped().Model(&domain.UserIdentity{}).
		Where("id = ?", wrongBind.IdentityID).Count(&gone).Error)
	assert.Zero(t, gone, "解绑必须物理删除身份行")

	var survivor domain.UserIdentity
	require.NoError(t, h.DB.First(&survivor, otherIdentity.ID).Error, "其他客户端的身份不能被误删")

	// 账号、角色、家长档案、孩子关联都必须保留：解绑换的是微信，不是账号。
	var alive int64
	require.NoError(t, h.DB.Model(&domain.User{}).Where("id = ?", target.ID).Count(&alive).Error)
	assert.EqualValues(t, 1, alive, "解绑不能删除账号")

	var roleCount int64
	require.NoError(t, h.DB.Model(&domain.UserRole{}).
		Where("user_id = ? AND role = ?", target.ID, domain.ClientRoleParent).Count(&roleCount).Error)
	assert.EqualValues(t, 1, roleCount, "解绑不能删除角色")

	var profileCount int64
	require.NoError(t, h.DB.Model(&domain.ParentProfile{}).
		Where("user_id = ? AND kindergarten_id = ?", target.ID, kg.ID).Count(&profileCount).Error)
	assert.EqualValues(t, 1, profileCount, "解绑不能删除家长档案")

	var linkCount int64
	require.NoError(t, h.DB.Model(&domain.StudentParent{}).
		Where("user_id = ? AND student_id = ?", target.ID, child.ID).Count(&linkCount).Error)
	assert.EqualValues(t, 1, linkCount, "解绑不能删除幼儿关联")

	// 审计：谁、在哪个园、解绑了哪个客户端的哪条身份。
	var audit domain.OperationLog
	require.NoError(t, h.DB.Where("action = ? AND target_type = ? AND target_id = ?",
		"unbind_identity", "user_identity", wrongBind.IdentityID).First(&audit).Error,
		"解绑必须写管理端审计")
	assert.Equal(t, kg.ID, audit.KindergartenID)
	assert.EqualValues(t, 990001, audit.UserID)
	assert.Equal(t, "parents", audit.Module)
	assert.Contains(t, audit.Detail, appID)
	assert.NotContains(t, audit.Detail, openID, "审计里不能出现完整 openid")
}

// --- slice 2: 录错号 → 陌生人绑走 → 解绑 → 改号 → 真实家长绑定并看到孩子 ---

func TestSupportCorrectPhoneAfterUnbindLetsRealParentSeeChild(t *testing.T) {
	h := authharness.New(t)

	appID := uniqueWxAppID()
	seedClient(t, h, appID, domain.ClientRoleParent, domain.ClientStatusEnabled)
	kg := seedKindergartenEntry(t, h, "support-correct", domain.AppChannelShared, uniquePublicCode("co"))
	wrongPhone := uniquePhone()
	target := seedParentAccount(t, h, wrongPhone, kg)
	child := seedSupportChild(t, h, kg, target.ID, "小红")

	// 园所录错号：号码属于陌生人，陌生人绑定后落到预创建账号上。
	strangerOpenID := uniqueOpenID("support-correct-stranger")
	stranger := bindStranger(t, h, appID, strangerOpenID, wrongPhone, kg.PublicCode)
	require.Equal(t, target.ID, stranger.UserID)

	adminToken := adminSupportToken(t, h, kg.ID)
	viewPath := fmt.Sprintf("/api/v1/admin/parents/%d/support", target.ID)

	// 支持视图：手机号、客户端身份（掩码）、园所成员关系。
	viewResp := h.Do(t, http.MethodGet, viewPath, adminToken, nil)
	require.Equal(t, http.StatusOK, viewResp.Status, string(viewResp.Body))
	var view supportViewJSON
	viewResp.DecodeData(t, &view)
	assert.Equal(t, target.ID, view.UserID)
	assert.Equal(t, wrongPhone, view.Phone)
	require.Len(t, view.Identities, 1)
	assert.Equal(t, appID, view.Identities[0].AppID)
	assert.NotEqual(t, strangerOpenID, view.Identities[0].OpenIDMasked, "支持视图必须掩码 openid")
	require.Len(t, view.Memberships, 1)
	assert.Equal(t, kg.ID, view.Memberships[0].KindergartenID)
	assert.Equal(t, kg.Name, view.Memberships[0].KindergartenName)
	assert.Equal(t, domain.ClientRoleParent, view.Memberships[0].Role)

	correctPhone := uniquePhone()
	phonePath := fmt.Sprintf("/api/v1/admin/parents/%d/phone", target.ID)

	// 错误微信还挂在账号上时改号必须被拒绝：否则陌生人会随着新号继续拥有访问权。
	blocked := h.PutJSONWithToken(t, phonePath, adminToken, map[string]any{"phone": correctPhone})
	require.Equal(t, http.StatusConflict, blocked.Status, string(blocked.Body))
	var unchanged domain.User
	require.NoError(t, h.DB.First(&unchanged, target.ID).Error)
	assert.Equal(t, wrongPhone, unchanged.Phone, "被拒绝的改号不能留下半成品")

	// 先解绑错误身份。
	unbindResp := h.Do(t, http.MethodDelete,
		fmt.Sprintf("/api/v1/admin/parents/%d/identities/%d", target.ID, stranger.IdentityID),
		adminToken, nil)
	require.Equal(t, http.StatusOK, unbindResp.Status, string(unbindResp.Body))

	// 再改号：返回更新后的支持视图，身份列表为空。
	putResp := h.PutJSONWithToken(t, phonePath, adminToken, map[string]any{"phone": correctPhone})
	require.Equal(t, http.StatusOK, putResp.Status, string(putResp.Body))
	var corrected supportViewJSON
	putResp.DecodeData(t, &corrected)
	assert.Equal(t, correctPhone, corrected.Phone)
	assert.Empty(t, corrected.Identities)

	var updated domain.User
	require.NoError(t, h.DB.First(&updated, target.ID).Error)
	assert.Equal(t, correctPhone, updated.Phone)

	// 真实家长用自己的微信 + 新号重新绑定：仍落到同一账号。
	realOpenID := uniqueOpenID("support-correct-real")
	real := bindStranger(t, h, appID, realOpenID, correctPhone, kg.PublicCode)
	require.Equal(t, target.ID, real.UserID, "正确家长必须绑定到同一条预创建账号")

	// 绑定后能看到自己的孩子。
	homeResp := h.Get(t, "/api/v1/parent/home/overview", real.AccessToken)
	require.Equal(t, http.StatusOK, homeResp.Status, string(homeResp.Body))
	var home struct {
		Students []struct {
			ID   uint64 `json:"id"`
			Name string `json:"name"`
		} `json:"students"`
	}
	homeResp.DecodeData(t, &home)
	require.Len(t, home.Students, 1, "真实家长重新绑定后必须能看到孩子")
	assert.Equal(t, child.ID, home.Students[0].ID)
	assert.Equal(t, "小红", home.Students[0].Name)

	// 改号写审计：操作人、园所、目标账号。
	var audit domain.OperationLog
	require.NoError(t, h.DB.Where("action = ? AND target_type = ? AND target_id = ?",
		"correct_phone", "user", target.ID).First(&audit).Error, "改号必须写管理端审计")
	assert.Equal(t, kg.ID, audit.KindergartenID)
	assert.EqualValues(t, 990001, audit.UserID)
	assert.NotContains(t, audit.Detail, correctPhone, "审计里不能出现完整新手机号")
}

// --- slice 3: 改号冲突与非法输入 ---

func TestSupportCorrectPhoneRejectsConflictAndInvalidInput(t *testing.T) {
	h := authharness.New(t)

	kg := seedKindergartenEntry(t, h, "support-conflict", domain.AppChannelShared, uniquePublicCode("cf"))
	wrongPhone := uniquePhone()
	target := seedParentAccount(t, h, wrongPhone, kg)
	occupiedPhone := uniquePhone()
	other := seedParentAccount(t, h, occupiedPhone, kg)

	adminToken := adminSupportToken(t, h, kg.ID)
	phonePath := fmt.Sprintf("/api/v1/admin/parents/%d/phone", target.ID)

	invalid := h.PutJSONWithToken(t, phonePath, adminToken, map[string]any{"phone": "12345"})
	// 项目统一用 422 表达参数校验失败（response.ErrValidation）。
	require.Equal(t, http.StatusUnprocessableEntity, invalid.Status, string(invalid.Body))

	conflict := h.PutJSONWithToken(t, phonePath, adminToken, map[string]any{"phone": occupiedPhone})
	require.Equal(t, http.StatusConflict, conflict.Status, string(conflict.Body))

	var targetAfter, otherAfter domain.User
	require.NoError(t, h.DB.First(&targetAfter, target.ID).Error)
	require.NoError(t, h.DB.First(&otherAfter, other.ID).Error)
	assert.Equal(t, wrongPhone, targetAfter.Phone, "冲突时目标账号不能改号")
	assert.Equal(t, occupiedPhone, otherAfter.Phone, "冲突时占用方不能被改动")

	// 软删账号不占用手机号：phone_active 生成列为 NULL，改号应当成功。
	require.NoError(t, h.DB.Delete(&domain.User{}, other.ID).Error)
	ok := h.PutJSONWithToken(t, phonePath, adminToken, map[string]any{"phone": occupiedPhone})
	require.Equal(t, http.StatusOK, ok.Status, string(ok.Body))
}

// --- slice 4: 租户与信任域边界 ---

func TestSupportEndpointsRespectTenantAndTokenBoundaries(t *testing.T) {
	h := authharness.New(t)

	appID := uniqueWxAppID()
	seedClient(t, h, appID, domain.ClientRoleParent, domain.ClientStatusEnabled)
	kgA := seedKindergartenEntry(t, h, "support-boundary-a", domain.AppChannelShared, uniquePublicCode("ba"))
	kgB := seedKindergartenEntry(t, h, "support-boundary-b", domain.AppChannelShared, uniquePublicCode("bb"))
	phone := uniquePhone()
	target := seedParentAccount(t, h, phone, kgA)

	stranger := bindStranger(t, h, appID, uniqueOpenID("support-boundary"), phone, kgA.PublicCode)
	require.Equal(t, target.ID, stranger.UserID)

	viewPath := fmt.Sprintf("/api/v1/admin/parents/%d/support", target.ID)
	unbindPath := fmt.Sprintf("/api/v1/admin/parents/%d/identities/%d", target.ID, stranger.IdentityID)
	phonePath := fmt.Sprintf("/api/v1/admin/parents/%d/phone", target.ID)

	// 未登录 401；家长 token 403（管理端路由不属于家长策略）。
	require.Equal(t, http.StatusUnauthorized, h.Get(t, viewPath, "").Status)
	require.Equal(t, http.StatusForbidden, h.Get(t, viewPath, stranger.AccessToken).Status)

	// 跨园管理员：读取、解绑、改号都必须按「账号不存在」处理。
	adminB := adminSupportToken(t, h, kgB.ID)
	require.Equal(t, http.StatusNotFound, h.Get(t, viewPath, adminB).Status)
	require.Equal(t, http.StatusNotFound, h.Do(t, http.MethodDelete, unbindPath, adminB, nil).Status)
	require.Equal(t, http.StatusNotFound,
		h.PutJSONWithToken(t, phonePath, adminB, map[string]any{"phone": uniquePhone()}).Status)

	var identityCount int64
	require.NoError(t, h.DB.Model(&domain.UserIdentity{}).
		Where("id = ?", stranger.IdentityID).Count(&identityCount).Error)
	assert.EqualValues(t, 1, identityCount, "跨园管理员不能解绑其他园的身份")

	var unchanged domain.User
	require.NoError(t, h.DB.First(&unchanged, target.ID).Error)
	assert.Equal(t, phone, unchanged.Phone, "跨园管理员不能改其他园的号")

	// 本园管理员可读，且只看得到本园成员关系。
	adminA := adminSupportToken(t, h, kgA.ID)
	own := h.Get(t, viewPath, adminA)
	require.Equal(t, http.StatusOK, own.Status, string(own.Body))
	var view supportViewJSON
	own.DecodeData(t, &view)
	require.Len(t, view.Memberships, 1)
	assert.Equal(t, kgA.ID, view.Memberships[0].KindergartenID)
}

// --- slice 5: 不能借道解绑别人的身份 ---

func TestSupportUnbindRejectsIdentityOfAnotherUser(t *testing.T) {
	h := authharness.New(t)

	appID := uniqueWxAppID()
	seedClient(t, h, appID, domain.ClientRoleParent, domain.ClientStatusEnabled)
	kg := seedKindergartenEntry(t, h, "support-owner", domain.AppChannelShared, uniquePublicCode("ow"))
	phone := uniquePhone()
	target := seedParentAccount(t, h, phone, kg)
	other := seedParentAccount(t, h, uniquePhone(), kg)
	otherIdentity := seedIdentityFor(t, h, other.ID, appID, uniqueOpenID("support-owner-other"))

	adminToken := adminSupportToken(t, h, kg.ID)
	resp := h.Do(t, http.MethodDelete,
		fmt.Sprintf("/api/v1/admin/parents/%d/identities/%d", target.ID, otherIdentity.ID),
		adminToken, nil)
	require.Equal(t, http.StatusNotFound, resp.Status, string(resp.Body))

	var alive int64
	require.NoError(t, h.DB.Model(&domain.UserIdentity{}).
		Where("id = ?", otherIdentity.ID).Count(&alive).Error)
	assert.EqualValues(t, 1, alive, "身份属于别的账号时不能解绑")
}

// --- slice 6: 解绑后旧会话不能再续期 ---

func TestSupportUnbindBlocksStrangerRefreshSession(t *testing.T) {
	h := authharness.New(t)

	appID := uniqueWxAppID()
	seedClient(t, h, appID, domain.ClientRoleParent, domain.ClientStatusEnabled)
	kg := seedKindergartenEntry(t, h, "support-refresh", domain.AppChannelShared, uniquePublicCode("rf"))
	phone := uniquePhone()
	target := seedParentAccount(t, h, phone, kg)
	stranger := bindStranger(t, h, appID, uniqueOpenID("support-refresh"), phone, kg.PublicCode)
	require.Equal(t, target.ID, stranger.UserID)

	// 先正常续期一次，拿一个未被消费的 refresh token。
	first := h.PostJSON(t, "/api/v1/auth/refresh", map[string]string{"refresh_token": stranger.RefreshToken})
	require.Equal(t, http.StatusOK, first.Status, string(first.Body))
	var refreshed struct {
		RefreshToken string `json:"refresh_token"`
	}
	first.DecodeData(t, &refreshed)
	require.NotEmpty(t, refreshed.RefreshToken)

	adminToken := adminSupportToken(t, h, kg.ID)
	unbind := h.Do(t, http.MethodDelete,
		fmt.Sprintf("/api/v1/admin/parents/%d/identities/%d", target.ID, stranger.IdentityID),
		adminToken, nil)
	require.Equal(t, http.StatusOK, unbind.Status, string(unbind.Body))

	// 身份已解绑，旧 refresh token 不能再换出该账号的会话。
	after := h.PostJSON(t, "/api/v1/auth/refresh", map[string]string{"refresh_token": refreshed.RefreshToken})
	require.Equal(t, http.StatusUnauthorized, after.Status, string(after.Body))
}

// --- slice 7: 平台支持入口（独立信任域，跨园成员关系） ---

func TestPlatformSupportSeesAllMembershipsAndAuditsSeparately(t *testing.T) {
	h := authharness.New(t)

	appID := uniqueWxAppID()
	seedClient(t, h, appID, domain.ClientRoleParent, domain.ClientStatusEnabled)
	kgA := seedKindergartenEntry(t, h, "platform-support-a", domain.AppChannelShared, uniquePublicCode("pa"))
	kgB := seedKindergartenEntry(t, h, "platform-support-b", domain.AppChannelShared, uniquePublicCode("pb"))
	phone := uniquePhone()
	target := seedParentAccount(t, h, phone, kgA, kgB)
	identity := seedIdentityFor(t, h, target.ID, appID, uniqueOpenID("platform-support"))

	platformToken, err := auth.SignPlatformAccess(h.PlatformSecret, "plat-001", time.Hour)
	require.NoError(t, err)

	viewPath := fmt.Sprintf("/api/v1/platform/users/%d/support", target.ID)
	viewResp := h.Get(t, viewPath, platformToken)
	require.Equal(t, http.StatusOK, viewResp.Status, string(viewResp.Body))
	var view supportViewJSON
	viewResp.DecodeData(t, &view)
	assert.Equal(t, phone, view.Phone)
	require.Len(t, view.Identities, 1)
	assert.NotEqual(t, identity.OpenID, view.Identities[0].OpenIDMasked)
	gotKG := make(map[uint64]bool, len(view.Memberships))
	for _, m := range view.Memberships {
		gotKG[m.KindergartenID] = true
	}
	assert.ElementsMatch(t, []uint64{kgA.ID, kgB.ID}, keysOf(gotKG),
		"平台支持必须看到目标用户的全部园所成员关系")

	// 信任域隔离（#183）：两个域使用不同的签名密钥，跨域凭证在验签阶段就被
	// 拒绝（401），而不再是验签通过后靠 scope claim 判 403。scope 检查仍保留
	// 作为纵深防御，见 middleware 的单元测试。
	adminToken := adminSupportToken(t, h, kgA.ID)
	require.Equal(t, http.StatusUnauthorized, h.Get(t, viewPath, adminToken).Status,
		"园所 JWT 用平台密钥验签必须失败")
	require.Equal(t, http.StatusUnauthorized, h.Do(t, http.MethodDelete,
		fmt.Sprintf("/api/v1/admin/parents/%d/identities/%d", target.ID, identity.ID),
		platformToken, nil).Status, "平台凭证用园所密钥验签必须失败")

	// 验收要求的两个伪造方向：用另一域的密钥签发本域凭证，HTTP 边界必须拒绝。
	forgedTenant, err := auth.SignAccess(h.PlatformSecret, fmt.Sprint(target.ID), []string{"parent"}, kgA.ID, time.Hour)
	require.NoError(t, err)
	require.Equal(t, http.StatusUnauthorized,
		h.Get(t, "/api/v1/common/user/profile", forgedTenant).Status,
		"用平台密钥签发的园所凭证必须被园所接口拒绝")

	forgedPlatform, err := auth.SignPlatformAccess(h.Secret, "plat-001", time.Hour)
	require.NoError(t, err)
	require.Equal(t, http.StatusUnauthorized, h.Get(t, viewPath, forgedPlatform).Status,
		"用园所密钥签发的平台凭证必须被平台接口拒绝")

	// 平台解绑身份，审计必须落 platform_operation_logs，而不是园所 operation_logs。
	unbindResp := h.Do(t, http.MethodDelete,
		fmt.Sprintf("/api/v1/platform/users/%d/identities/%d", target.ID, identity.ID),
		platformToken, nil)
	require.Equal(t, http.StatusOK, unbindResp.Status, string(unbindResp.Body))

	var plog domain.PlatformOperationLog
	require.NoError(t, h.DB.Where("action = ? AND target_type = ? AND target_id = ?",
		"unbind_identity", "user_identity", identity.ID).First(&plog).Error,
		"平台解绑必须写平台审计")
	assert.Equal(t, "plat-001", plog.Operator)
	assert.Equal(t, "support", plog.Module)

	var adminAudit int64
	require.NoError(t, h.DB.Model(&domain.OperationLog{}).
		Where("action = ? AND target_id = ?", "unbind_identity", identity.ID).Count(&adminAudit).Error)
	assert.Zero(t, adminAudit, "平台操作不能混进园所审计表")

	// 平台更正手机号，同样落平台审计。
	correctPhone := uniquePhone()
	corrResp := h.PutJSONWithToken(t,
		fmt.Sprintf("/api/v1/platform/users/%d/phone", target.ID), platformToken,
		map[string]any{"phone": correctPhone})
	require.Equal(t, http.StatusOK, corrResp.Status, string(corrResp.Body))
	var corrected supportViewJSON
	corrResp.DecodeData(t, &corrected)
	assert.Equal(t, correctPhone, corrected.Phone)

	var phoneLog domain.PlatformOperationLog
	require.NoError(t, h.DB.Where("action = ? AND target_type = ? AND target_id = ?",
		"correct_phone", "user", target.ID).First(&phoneLog).Error,
		"平台改号必须写平台审计")
	assert.Equal(t, "plat-001", phoneLog.Operator)
}
