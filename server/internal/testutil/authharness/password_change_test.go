package authharness_test

import (
	"fmt"
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

func setMustChangePassword(t *testing.T, h *authharness.Harness, userID uint64) {
	t.Helper()
	require.NoError(t, h.DB.Model(&domain.User{}).Where("id = ?", userID).
		Update("must_change_password", true).Error)
}

// 平台开通 → 园长登录（flag=true）→ 业务接口 403/40303；精确豁免 200；
// 未改密 refresh 仍 200 但新 access 仍被拦；改密后旧 access / refresh 全部 401，
// 必须用新口令重新登录。这是 #198 + #182 的完整验收链路。
func TestForcedPasswordChangeGateEndToEnd(t *testing.T) {
	h := authharness.New(t)
	kg := seedKindergarten(t, h, "首登改密园")
	phone := uniquePhone()
	user := createAdminUser(t, h, phone, "Password@123", kg)
	setMustChangePassword(t, h, user.ID)

	login := loginWithPassword(t, h, phone, "Password@123")
	require.Equal(t, 200, login.Status, string(login.Body))
	var first service.LoginResult
	login.DecodeData(t, &first)
	require.NotEmpty(t, first.Token)
	require.True(t, first.User.MustChangePassword, "登录响应必须带出首登标记")

	blocked := h.Get(t, "/api/v1/admin/kindergarten/info", first.Token)
	require.Equal(t, 403, blocked.Status, string(blocked.Body))
	assert.Equal(t, middleware.PasswordChangeRequiredCode, blocked.EnvelopeCode(t))

	// 豁免端点：个人资料读取、同意 GET/POST。
	assert.Equal(t, 200, h.Get(t, "/api/v1/common/user/profile", first.Token).Status)
	assert.Equal(t, 200, h.Get(t, "/api/v1/common/consent", first.Token).Status)
	consented := h.PostJSONWithToken(t, "/api/v1/common/consent", first.Token,
		map[string]string{"policy_version": "v1"})
	assert.Equal(t, 200, consented.Status, string(consented.Body))

	// 未改密时 refresh 仍可续期，但新 access 继续被 40303 拦截。
	renewed := refreshOK(t, h, first.RefreshToken)
	renewedBlocked := h.Get(t, "/api/v1/admin/kindergarten/info", renewed.Token)
	require.Equal(t, 403, renewedBlocked.Status, string(renewedBlocked.Body))
	assert.Equal(t, middleware.PasswordChangeRequiredCode, renewedBlocked.EnvelopeCode(t))

	// 改密成功：标记落库为 false，但 #182 会推进账号级失效时间，
	// 当前 access / refresh 都必须失效，客户端需要重新登录。
	// epoch 是秒级精度：等过签发秒，避免「同一秒内旧 token 仍有效」的设计行为。
	waitNextClockSecond(t)
	changed := h.PutJSONWithToken(t, "/api/v1/common/user/password", renewed.Token,
		map[string]string{"old_password": "Password@123", "new_password": "NewPassword@456"})
	require.Equal(t, 200, changed.Status, string(changed.Body))

	oldAccess := h.Get(t, "/api/v1/admin/kindergarten/info", renewed.Token)
	assert.Equal(t, 401, oldAccess.Status, "改密后旧 access 必须 401：%s", oldAccess.Body)

	oldRefresh := refreshRequest(t, h, renewed.RefreshToken)
	assert.Equal(t, 401, oldRefresh.Status, "改密后旧 refresh 必须 401：%s", oldRefresh.Body)

	// 用新口令重新登录后，会话才恢复；登出仍能正常吊销新 access。
	relogin := loginWithPassword(t, h, phone, "NewPassword@456")
	require.Equal(t, 200, relogin.Status, string(relogin.Body))
	var second service.LoginResult
	relogin.DecodeData(t, &second)
	assert.Equal(t, 200, h.Get(t, "/api/v1/admin/kindergarten/info", second.Token).Status)
	logout := h.PostJSONWithToken(t, "/api/v1/auth/logout", second.Token, map[string]any{})
	assert.Equal(t, 200, logout.Status, string(logout.Body))
	afterLogout := h.Get(t, "/api/v1/admin/kindergarten/info", second.Token)
	assert.Equal(t, 401, afterLogout.Status, "登出后 access 必须失效：%s", afterLogout.Body)
}

// checker 读取失败必须失败关闭：模拟账号行消失（读取即 error），
// 业务接口返回 503，不得放行。
func TestForcedPasswordChangeGateFailsClosedWhenUserRowMissing(t *testing.T) {
	h := authharness.New(t)
	kg := seedKindergarten(t, h, "首登改密失败关闭园")
	phone := uniquePhone()
	user := createAdminUser(t, h, phone, "Password@123", kg)
	setMustChangePassword(t, h, user.ID)

	login := loginWithPassword(t, h, phone, "Password@123")
	require.Equal(t, 200, login.Status, string(login.Body))
	var first service.LoginResult
	login.DecodeData(t, &first)

	// FK 是 RESTRICT：先清角色引用，再硬删账号。登出黑名单不依赖 users 行。
	h.HardDelete(t, &domain.UserRole{}, "user_id = ?", user.ID)
	h.HardDelete(t, &domain.User{}, "id = ?", user.ID)

	resp := h.Get(t, "/api/v1/admin/kindergarten/info", first.Token)
	require.Equal(t, 503, resp.Status, "checker 读取失败必须 503：%s", resp.Body)
	assert.Equal(t, 50300, resp.EnvelopeCode(t))
}

// 门禁只作用于 admin_web：同一账号即使 must_change_password=true，
// 小程序凭证也照常访问业务接口（小程序没有可完成的改密流程）。
func TestForcedPasswordChangeGateIgnoresMiniProgramTokens(t *testing.T) {
	h := authharness.New(t)
	kg := seedKindergarten(t, h, "首登改密小程序园")
	phone := uniquePhone()
	user := createUser(t, h, phone, "Password@123", "")
	setMustChangePassword(t, h, user.ID)

	token, err := auth.SignClientAccess(h.Secret, fmt.Sprintf("%d", user.ID), []string{"parent"}, kg.ID,
		auth.ClientScope{AppID: "wx-parent-198-test", Kind: "shared"}, time.Minute)
	require.NoError(t, err)

	resp := h.Get(t, "/api/v1/parent/home/ping", token)
	assert.Equal(t, 200, resp.Status, "小程序凭证不得受首登改密门禁影响：%s", resp.Body)
}
