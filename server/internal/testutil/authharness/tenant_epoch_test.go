package authharness_test

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/service"
	"kindergarten-server/internal/testutil/authharness"
)

// #182：改密 / 管理员重置密码 / 删除管理员后，此前签发的 access 与 refresh
// 必须立即失效，重新登录不受影响；Redis 故障一律失败关闭。
//
// epoch 是秒级精度（与 JWT iat、平台会话一致）：失效动作必须落在签发 token
// 之后的时钟秒，否则「同一秒内签发的旧凭证仍有效」是设计行为，测试会偶发。
func waitNextClockSecond(t *testing.T) {
	t.Helper()
	next := time.Unix(time.Now().Unix()+1, 0)
	time.Sleep(time.Until(next) + 20*time.Millisecond)
}

func adminLoginOK(t *testing.T, h *authharness.Harness, phone, password string) *service.LoginResult {
	t.Helper()
	res := loginWithPassword(t, h, phone, password)
	require.Equal(t, http.StatusOK, res.Status, string(res.Body))
	var result service.LoginResult
	res.DecodeData(t, &result)
	require.NotEmpty(t, result.Token)
	require.NotEmpty(t, result.RefreshToken)
	return &result
}

func assertProtectedStatus(t *testing.T, h *authharness.Harness, token string, want int, msg string) {
	t.Helper()
	res := h.Get(t, "/api/v1/admin/kindergarten/info", token)
	assert.Equal(t, want, res.Status, "%s：body=%s", msg, res.Body)
}

func findAdminAudit(t *testing.T, h *authharness.Harness, targetID uint64, action string) *domain.OperationLog {
	t.Helper()
	var audit domain.OperationLog
	require.NoError(t, h.DB.
		Where("target_id = ? AND module = ? AND action = ?", targetID, "admins", action).
		First(&audit).Error, "缺少 %s 审计", action)
	return &audit
}

// 无 jti 的旧 token 不触发吊销黑名单查询，专门用来验证 epoch 读取失败
// 本身也会失败关闭（否则会被 revoker 的同类错误掩盖）。
func signLegacyAccess(t *testing.T, secret []byte, userID, kindergartenID uint64) string {
	t.Helper()
	return signLegacyAccessAt(t, secret, userID, kindergartenID, time.Now())
}

// signLegacyAccessAt 显式指定 iat，用来精确验证 epoch 的秒级比较边界。
func signLegacyAccessAt(t *testing.T, secret []byte, userID, kindergartenID uint64, issuedAt time.Time) string {
	t.Helper()
	claims := auth.AccessClaims{
		UserID:         fmt.Sprintf("%d", userID),
		Roles:          []string{"admin"},
		KindergartenID: kindergartenID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(issuedAt.Add(2 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			NotBefore: jwt.NewNumericDate(issuedAt),
			Issuer:    "kindergarten-api",
			Subject:   fmt.Sprintf("%d", userID),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
	require.NoError(t, err)
	return token
}

// epoch 与 JWT iat 都是秒级精度，比较用「严格早于」：iat == epoch 的凭证
// 按设计放行，iat 早一秒即拒绝。用构造的 iat 固定边界，不依赖 sleep 抢秒。
func TestAuthJWTTenantEpochIsSecondPrecise(t *testing.T) {
	h := authharness.New(t)
	kg := seedKindergarten(t, h, "epoch 秒级边界园")
	phone := uniquePhone()
	user := createAdminUser(t, h, phone, "Password@123", kg)
	store := auth.NewTenantEpochStore(h.RedisClient, auth.PlatformEpochTTLFor(0, 0))
	bump := func(at time.Time) {
		t.Helper()
		require.NoError(t, store.BumpTenantEpoch(context.Background(), strconv.FormatUint(user.ID, 10), at))
	}

	issuedAt := time.Now().Add(-time.Minute).Truncate(time.Second)
	bump(issuedAt)
	token := signLegacyAccessAt(t, h.Secret, user.ID, kg.ID, issuedAt)
	assertProtectedStatus(t, h, token, http.StatusOK, "iat == epoch 的凭证按设计放行")

	bump(issuedAt.Add(time.Second))
	assertProtectedStatus(t, h, token, http.StatusUnauthorized, "iat 早于 epoch 一秒即必须拒绝")
}

func TestChangePasswordInvalidatesExistingSessions(t *testing.T) {
	h := authharness.New(t)
	kg := seedKindergarten(t, h, "改密失效园")
	phone := uniquePhone()
	user := createAdminUser(t, h, phone, "Password@123", kg)

	first := adminLoginOK(t, h, phone, "Password@123")
	assertProtectedStatus(t, h, first.Token, http.StatusOK, "改密前旧 access 应可用")

	waitNextClockSecond(t)
	change := h.PutJSONWithToken(t, "/api/v1/common/user/password", first.Token, map[string]string{
		"old_password": "Password@123",
		"new_password": "Changed@123",
	})
	require.Equal(t, http.StatusOK, change.Status, string(change.Body))

	assertProtectedStatus(t, h, first.Token, http.StatusUnauthorized, "改密后旧 access 必须 401")

	replay := refreshRequest(t, h, first.RefreshToken)
	assert.Equal(t, http.StatusUnauthorized, replay.Status, "改密后旧 refresh 必须 401：%s", replay.Body)

	second := adminLoginOK(t, h, phone, "Changed@123")
	assertProtectedStatus(t, h, second.Token, http.StatusOK, "改密后重新登录必须可用")

	stale := loginWithPassword(t, h, phone, "Password@123")
	assert.Equal(t, http.StatusBadRequest, stale.Status, "旧口令不得再登录：%s", stale.Body)

	audit := findAuthAudit(t, h, user.ID, "change_password")
	assert.Equal(t, kg.ID, audit.KindergartenID, "改密审计必须记录当前园所")
	assert.Equal(t, "user", audit.TargetType)
	assert.Equal(t, user.ID, audit.TargetID, "改密审计必须记录账号")
	assert.NotContains(t, audit.Detail, "Password@123", "审计不得记录旧口令")
	assert.NotContains(t, audit.Detail, "Changed@123", "审计不得记录新口令")
}

// epoch 写入失败时必须失败关闭：既返回 503，也绝不能把新口令落库。
func TestChangePasswordFailsClosedWhenEpochWriteFails(t *testing.T) {
	h := authharness.New(t)
	kg := seedKindergarten(t, h, "改密失败关闭园")
	phone := uniquePhone()
	createAdminUser(t, h, phone, "Password@123", kg)
	first := adminLoginOK(t, h, phone, "Password@123")

	h.Redis.SetFailWrites(true)
	change := h.PutJSONWithToken(t, "/api/v1/common/user/password", first.Token, map[string]string{
		"old_password": "Password@123",
		"new_password": "Changed@123",
	})
	h.Redis.SetFailWrites(false)

	require.Equal(t, http.StatusServiceUnavailable, change.Status, string(change.Body))

	// 口令未变：旧口令仍可登录，新口令不可用。账号级失效不可用时宁可不改密。
	stillOld := loginWithPassword(t, h, phone, "Password@123")
	assert.Equal(t, http.StatusOK, stillOld.Status, "失败关闭时口令不得被修改：%s", stillOld.Body)
	notNew := loginWithPassword(t, h, phone, "Changed@123")
	assert.Equal(t, http.StatusBadRequest, notNew.Status, "失败关闭时新口令不得生效：%s", notNew.Body)
}

// epoch 读故障时受保护请求一律 503；用无 jti 的旧 token 排除吊销黑名单干扰。
func TestAuthJWTFailsClosedWhenTenantEpochReadFails(t *testing.T) {
	h := authharness.New(t)
	kg := seedKindergarten(t, h, "epoch 读取失败园")
	phone := uniquePhone()
	user := createAdminUser(t, h, phone, "Password@123", kg)
	require.NotZero(t, user.ID)

	token := signLegacyAccess(t, h.Secret, user.ID, kg.ID)
	assertProtectedStatus(t, h, token, http.StatusOK, "epoch 可读时旧格式 access 仍可用")

	h.Redis.SetFailReads(true)
	res := h.Get(t, "/api/v1/admin/kindergarten/info", token)
	h.Redis.SetFailReads(false)

	assert.Equal(t, http.StatusServiceUnavailable, res.Status,
		"epoch 读取失败必须失败关闭，不能放行：%s", res.Body)
}

func TestAdminResetPasswordInvalidatesExistingSessions(t *testing.T) {
	h := authharness.New(t)
	kg := seedKindergarten(t, h, "重置失效园")
	operatorPhone := uniquePhone()
	createAdminUser(t, h, operatorPhone, "Operator@123", kg)
	targetPhone := uniquePhone()
	target := createAdminUser(t, h, targetPhone, "Password@123", kg)

	operator := adminLoginOK(t, h, operatorPhone, "Operator@123")
	victim := adminLoginOK(t, h, targetPhone, "Password@123")
	assertProtectedStatus(t, h, victim.Token, http.StatusOK, "重置前 victim access 应可用")

	waitNextClockSecond(t)
	reset := h.PutJSONWithToken(t, fmt.Sprintf("/api/v1/admin/admins/%d/reset-password", target.ID),
		operator.Token, map[string]string{"new_password": "Reset@1234"})
	require.Equal(t, http.StatusOK, reset.Status, string(reset.Body))

	assertProtectedStatus(t, h, victim.Token, http.StatusUnauthorized, "重置后旧 access 必须 401")

	replay := refreshRequest(t, h, victim.RefreshToken)
	assert.Equal(t, http.StatusUnauthorized, replay.Status, "重置后旧 refresh 必须 401：%s", replay.Body)

	relogin := adminLoginOK(t, h, targetPhone, "Reset@1234")
	assertProtectedStatus(t, h, relogin.Token, http.StatusOK, "重置后新口令登录必须可用")
	assertProtectedStatus(t, h, operator.Token, http.StatusOK, "操作者的会话不受影响")

	audit := findAdminAudit(t, h, target.ID, "reset_password")
	assert.Equal(t, kg.ID, audit.KindergartenID, "重置审计必须记录当前园所")
	assert.Equal(t, "admin", audit.TargetType)
	assert.NotContains(t, audit.Detail, "Reset@1234", "审计不得记录新口令")
}

// 重置时的 epoch 写入同样失败关闭：不能出现「口令已换、旧会话仍在」。
func TestAdminResetPasswordFailsClosedWhenEpochWriteFails(t *testing.T) {
	h := authharness.New(t)
	kg := seedKindergarten(t, h, "重置失败关闭园")
	operatorPhone := uniquePhone()
	createAdminUser(t, h, operatorPhone, "Operator@123", kg)
	targetPhone := uniquePhone()
	target := createAdminUser(t, h, targetPhone, "Password@123", kg)
	operator := adminLoginOK(t, h, operatorPhone, "Operator@123")

	h.Redis.SetFailWrites(true)
	reset := h.PutJSONWithToken(t, fmt.Sprintf("/api/v1/admin/admins/%d/reset-password", target.ID),
		operator.Token, map[string]string{"new_password": "Reset@1234"})
	h.Redis.SetFailWrites(false)

	require.Equal(t, http.StatusServiceUnavailable, reset.Status, string(reset.Body))

	stillOld := loginWithPassword(t, h, targetPhone, "Password@123")
	assert.Equal(t, http.StatusOK, stillOld.Status, "失败关闭时口令不得被重置：%s", stillOld.Body)
}

// seedExistingTeacherWithSession 准备一个「既有非管理员账号」：带手机号、
// 带老师角色，并已登记老师端微信身份。添加管理员会按手机号复用该账号。
func seedExistingTeacherWithSession(t *testing.T, h *authharness.Harness, kg *domain.Kindergarten) (*domain.User, string, string, string) {
	t.Helper()
	appID := uniqueAppID("create-admin-existing")
	seedSharedClient(t, h, appID, domain.ClientRoleTeacher)
	openID := uniqueOpenID("create-admin-existing")
	user := seedMember(t, h, appID, openID, memberRole{role: domain.ClientRoleTeacher, kg: kg})
	// 成功创建管理员会追加 user_admin_roles；必须在 seedMember 的清理之前删掉，
	// 否则外键 RESTRICT 会挡住 users 删除。
	t.Cleanup(func() { h.HardDelete(t, &domain.UserAdminRole{}, "user_id = ?", user.ID) })

	phone := uniquePhone()
	hash, err := service.HashPassword("Teacher@123")
	require.NoError(t, err)
	require.NoError(t, h.DB.Model(&domain.User{}).Where("id = ?", user.ID).
		Updates(map[string]any{"phone": phone, "password_hash": hash}).Error)
	return user, appID, openID, phone
}

// #182 范围补漏：CreateAdmin 对既有非管理员账号是「凭据重置」——它会用本次
// 提交的口令覆盖该账号的 password_hash。旧 access / refresh 必须立即失效，
// 且必须留下独立的重置审计，不能被「创建管理员」字样掩盖。
func TestCreateAdminResetsExistingNonAdminCredentials(t *testing.T) {
	h := authharness.New(t)
	kg := seedKindergarten(t, h, "创建管理员重置园")
	t.Cleanup(func() { h.HardDelete(t, &domain.AdminRole{}, "kindergarten_id = ?", kg.ID) })
	operatorPhone := uniquePhone()
	createAdminUser(t, h, operatorPhone, "Operator@123", kg)
	operator := adminLoginOK(t, h, operatorPhone, "Operator@123")

	teacher, appID, openID, phone := seedExistingTeacherWithSession(t, h, kg)
	session := loginWithClient(t, h, appID, openID)
	require.NotEmpty(t, session.RefreshToken)
	before := h.Get(t, "/api/v1/common/user/profile", session.Token)
	require.Equal(t, http.StatusOK, before.Status, "创建前既有账号的 access 应可用：%s", before.Body)

	waitNextClockSecond(t)
	res := h.PostJSONWithToken(t, "/api/v1/admin/admins", operator.Token, map[string]string{
		"phone": phone, "nickname": "老师转管理员", "password": "Admin@1234",
	})
	require.Equal(t, http.StatusCreated, res.Status, string(res.Body))
	var created struct {
		ID uint64 `json:"id"`
	}
	res.DecodeData(t, &created)
	require.Equal(t, teacher.ID, created.ID, "必须复用同一手机号账号，不得另建")

	oldAccess := h.Get(t, "/api/v1/common/user/profile", session.Token)
	assert.Equal(t, http.StatusUnauthorized, oldAccess.Status,
		"口令被创建流程覆盖后旧 access 必须 401：%s", oldAccess.Body)

	oldRefresh := refreshRequest(t, h, session.RefreshToken)
	assert.Equal(t, http.StatusUnauthorized, oldRefresh.Status,
		"口令被创建流程覆盖后旧 refresh 必须 401：%s", oldRefresh.Body)

	adminRelogin := adminLoginOK(t, h, phone, "Admin@1234")
	assertProtectedStatus(t, h, adminRelogin.Token, http.StatusOK, "新口令登录管理端必须可用")

	fresh := loginWithClient(t, h, appID, openID)
	freshProfile := h.Get(t, "/api/v1/common/user/profile", fresh.Token)
	assert.Equal(t, http.StatusOK, freshProfile.Status, "重新登录后新会话必须可用：%s", freshProfile.Body)

	audit := findAdminAudit(t, h, teacher.ID, "reset_password")
	assert.Equal(t, kg.ID, audit.KindergartenID, "重置审计必须记录当前园所")
	assert.Equal(t, "admin", audit.TargetType)
	assert.Contains(t, audit.Detail, "create_admin", "重置审计必须标明来自创建管理员流程")
	assert.NotContains(t, audit.Detail, "Admin@1234", "审计不得记录新口令")
	findAdminAudit(t, h, teacher.ID, "create")
}

// 创建流程里的 epoch 写入同样失败关闭：不能出现「口令已被覆盖、旧会话仍在」，
// 也不能留下管理员身份。
func TestCreateAdminResetFailsClosedWhenEpochWriteFails(t *testing.T) {
	h := authharness.New(t)
	kg := seedKindergarten(t, h, "创建管理员失败关闭园")
	t.Cleanup(func() { h.HardDelete(t, &domain.AdminRole{}, "kindergarten_id = ?", kg.ID) })
	operatorPhone := uniquePhone()
	createAdminUser(t, h, operatorPhone, "Operator@123", kg)
	operator := adminLoginOK(t, h, operatorPhone, "Operator@123")

	teacher, appID, openID, phone := seedExistingTeacherWithSession(t, h, kg)
	session := loginWithClient(t, h, appID, openID)
	var before domain.User
	require.NoError(t, h.DB.First(&before, teacher.ID).Error)

	h.Redis.SetFailWrites(true)
	res := h.PostJSONWithToken(t, "/api/v1/admin/admins", operator.Token, map[string]string{
		"phone": phone, "nickname": "老师转管理员", "password": "Admin@1234",
	})
	h.Redis.SetFailWrites(false)
	require.Equal(t, http.StatusServiceUnavailable, res.Status, string(res.Body))

	var after domain.User
	require.NoError(t, h.DB.First(&after, teacher.ID).Error)
	assert.Equal(t, before.PasswordHash, after.PasswordHash, "epoch 写失败时不得覆盖口令")

	var adminRoles int64
	require.NoError(t, h.DB.Model(&domain.UserRole{}).
		Where("user_id = ? AND role = ?", teacher.ID, "admin").Count(&adminRoles).Error)
	assert.Zero(t, adminRoles, "epoch 写失败时不得授予管理员身份")

	still := h.Get(t, "/api/v1/common/user/profile", session.Token)
	assert.Equal(t, http.StatusOK, still.Status, "失败关闭时旧会话不得被误伤：%s", still.Body)

	var audits int64
	require.NoError(t, h.DB.Model(&domain.OperationLog{}).
		Where("target_id = ? AND module = ?", teacher.ID, "admins").Count(&audits).Error)
	assert.Zero(t, audits, "失败的操作不得留下成功审计")
}

func TestDeleteAdminInvalidatesExistingSessions(t *testing.T) {
	h := authharness.New(t)
	kg := seedKindergarten(t, h, "删除失效园")
	operatorPhone := uniquePhone()
	createAdminUser(t, h, operatorPhone, "Operator@123", kg)
	targetPhone := uniquePhone()
	target := createAdminUser(t, h, targetPhone, "Password@123", kg)

	operator := adminLoginOK(t, h, operatorPhone, "Operator@123")
	victim := adminLoginOK(t, h, targetPhone, "Password@123")
	assertProtectedStatus(t, h, victim.Token, http.StatusOK, "删除前 victim access 应可用")

	waitNextClockSecond(t)
	del := h.Do(t, http.MethodDelete, fmt.Sprintf("/api/v1/admin/admins/%d", target.ID), operator.Token, nil)
	require.Equal(t, http.StatusOK, del.Status, string(del.Body))

	assertProtectedStatus(t, h, victim.Token, http.StatusUnauthorized, "删除后旧 access 必须 401")

	replay := refreshRequest(t, h, victim.RefreshToken)
	assert.Equal(t, http.StatusUnauthorized, replay.Status, "删除后旧 refresh 必须 401：%s", replay.Body)

	// 管理员身份已被移除：不能再以管理端身份登录。
	relogin := loginWithPassword(t, h, targetPhone, "Password@123")
	assert.Equal(t, http.StatusForbidden, relogin.Status,
		"被删除的管理员不得重新登录管理端：%s", relogin.Body)

	audit := findAdminAudit(t, h, target.ID, "delete")
	assert.Equal(t, kg.ID, audit.KindergartenID, "删除审计必须记录当前园所")
	assert.Equal(t, "admin", audit.TargetType)
}
