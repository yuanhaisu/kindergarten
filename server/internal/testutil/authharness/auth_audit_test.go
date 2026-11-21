package authharness_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/testutil/authharness"
)

// #160 L205：会话建立必须留审计（operation_logs）——登录、家长绑定、老师扫码
// 绑定、切换园所都要能回答「谁、用哪个 app_id、在哪个园」。审计不落 openid：
// 那属于客户端身份细节，且已有 #172 的脱敏查看路径。

func findAuthAudit(t *testing.T, h *authharness.Harness, userID uint64, action string) *domain.OperationLog {
	t.Helper()
	var audit domain.OperationLog
	require.NoError(t, h.DB.
		Where("user_id = ? AND module = ? AND action = ?", userID, "auth", action).
		First(&audit).Error, "缺少 %s 审计", action)
	return &audit
}

func TestWeChatLoginWritesAuthAudit(t *testing.T) {
	h := authharness.New(t)
	appID := uniqueWxAppID()
	seedSharedClient(t, h, appID, domain.ClientRoleParent)
	kg := seedKindergartenWithChannel(t, h, "审计登录园", domain.AppChannelShared)
	openID := uniqueOpenID("audit-login")
	user := seedMember(t, h, appID, openID, memberRole{role: domain.ClientRoleParent, kg: kg})

	loginWithClient(t, h, appID, openID)

	audit := findAuthAudit(t, h, user.ID, "login")
	assert.Equal(t, kg.ID, audit.KindergartenID, "登录审计必须落在会话园")
	assert.Equal(t, "user", audit.TargetType)
	assert.Equal(t, user.ID, audit.TargetID)
	assert.Contains(t, audit.Detail, appID, "审计必须记录 app_id")
	assert.NotContains(t, audit.Detail, openID, "审计不得记录 openid")
}

// 待绑定会话没有园所：此时不写审计（kg=0 的行按园所检索不可见），
// 绑定成功那一步自会写一条有园所的记录。
func TestUnboundLoginSkipsAuthAudit(t *testing.T) {
	h := authharness.New(t)
	appID := uniqueWxAppID()
	seedSharedClient(t, h, appID, domain.ClientRoleParent)
	openID := uniqueOpenID("audit-unbound")

	pending := pendingLogin(t, h, appID, openID)
	require.NotNil(t, pending.User)
	require.Empty(t, pending.User.Roles)

	var count int64
	require.NoError(t, h.DB.Model(&domain.OperationLog{}).
		Where("user_id = ? AND module = ?", pending.User.ID, "auth").
		Count(&count).Error)
	assert.Zero(t, count, "待绑定登录不应写园所级审计")
}

func TestParentBindPhoneWritesAuthAudit(t *testing.T) {
	h := authharness.New(t)
	appID := uniqueWxAppID()
	seedSharedClient(t, h, appID, domain.ClientRoleParent)
	kg := seedKindergartenEntry(t, h, "审计家长绑定园", domain.AppChannelShared, uniquePublicCode("ab"))
	phone := uniquePhone()
	target := seedParentAccount(t, h, phone, kg)
	openID := uniqueOpenID("audit-bind-parent")

	bindStranger(t, h, appID, openID, phone, kg.PublicCode)

	audit := findAuthAudit(t, h, target.ID, "bind_parent_phone")
	assert.Equal(t, kg.ID, audit.KindergartenID)
	assert.Equal(t, target.ID, audit.TargetID)
	assert.Contains(t, audit.Detail, appID)
	assert.NotContains(t, audit.Detail, openID)
}

func TestSwitchKindergartenWritesAuthAudit(t *testing.T) {
	h := authharness.New(t)
	appID := uniqueWxAppID()
	seedSharedClient(t, h, appID, domain.ClientRoleParent)
	kgA := seedKindergartenWithChannel(t, h, "审计切换甲园", domain.AppChannelShared)
	kgB := seedKindergartenWithChannel(t, h, "审计切换乙园", domain.AppChannelShared)
	openID := uniqueOpenID("audit-switch")
	user := seedMember(t, h, appID, openID,
		memberRole{role: domain.ClientRoleParent, kg: kgA},
		memberRole{role: domain.ClientRoleParent, kg: kgB},
	)

	login := loginWithClient(t, h, appID, openID)
	switched := switchOK(t, h, login.Token, kgB.ID, login.RefreshToken)
	require.Equal(t, kgB.ID, *switched.User.Roles[0].KindergartenID)

	audit := findAuthAudit(t, h, user.ID, "switch_kindergarten")
	assert.Equal(t, kgB.ID, audit.KindergartenID, "切换审计必须落在目标园")
	assert.Contains(t, audit.Detail, appID)
}

func TestTeacherBindInviteWritesAuthAudit(t *testing.T) {
	h := authharness.New(t)
	f := setupTeacherBind(t, h, teacherBindOptions{})

	resp := h.PostJSONWithToken(t, "/api/v1/common/user/bind-invite", f.PendingTok, map[string]string{
		"invite_code": f.InviteToken,
	})
	require.Equal(t, 200, resp.Status, string(resp.Body))

	audit := findAuthAudit(t, h, f.Teacher.ID, "bind_invite")
	assert.Equal(t, f.KG.ID, audit.KindergartenID)
	assert.Equal(t, f.Teacher.ID, audit.TargetID)
	assert.Contains(t, audit.Detail, f.AppID)
	assert.NotContains(t, audit.Detail, f.OpenID, "审计不得记录 openid")
}
