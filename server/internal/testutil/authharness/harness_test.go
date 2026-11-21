package authharness_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/testutil/authharness"
)

// 这是认证/身份单缝的入口测试：后续身份票复用同一个 harness，不再另建
// service/中间件双缝。四个用例分别锁定独立测试库、HTTP 认证链、假微信
// 注入与硬删清理。

func TestHarnessUsesDedicatedTestDatabase(t *testing.T) {
	h := authharness.New(t)

	assert.NotEqual(t, h.DevDatabase, h.TestDatabase,
		"harness 必须连独立测试库：复用开发库会被 phone_active 唯一索引与存量行污染")
	if !h.TestDSNFromEnv {
		// 派生库名必须带 _test 标记，避免误连生产/开发库；显式指定 DSN 时
		// 库名由使用者决定，只要与开发库不同即可。
		assert.Contains(t, h.TestDatabase, "_test",
			"派生库名必须带 _test 标记，避免误连生产/开发库")
	}
}

// 缺 token 必须在 AuthJWT 被拦下，返回统一包络的 401，而不是落到 handler 或 500。
func TestAuthedRouteWithoutTokenReturnsUnauthorized(t *testing.T) {
	h := authharness.New(t)

	resp := h.Get(t, "/api/v1/common/user/profile", "")

	require.Equal(t, http.StatusUnauthorized, resp.Status, string(resp.Body))
	assert.Equal(t, 40100, resp.EnvelopeCode(t), string(resp.Body))
}

func TestUnknownRouteReturnsNotFound(t *testing.T) {
	h := authharness.New(t)

	resp := h.Get(t, "/api/v1/definitely-not-a-route-161", "")

	require.Equal(t, http.StatusNotFound, resp.Status, string(resp.Body))
}

// 假微信必须真的替换掉真实微信客户端：code 在替身里登记过才可能成功，
// 成功后又必须能在真实测试库里查到落库用户。
func TestFakeWeChatLoginCreatesSessionOverHTTP(t *testing.T) {
	h := authharness.New(t)
	openID := fmt.Sprintf("openid-161-login-%d", time.Now().UnixNano())
	code := "wx-code-161-" + openID
	h.WeChat.StubCode2Session(code, openID, "")
	t.Cleanup(func() { h.HardDelete(t, &domain.User{}, "open_id = ?", openID) })

	resp := h.PostJSON(t, "/api/v1/auth/login/wechat", map[string]string{
		"code": code,
		"role": "parent",
	})
	require.Equal(t, http.StatusOK, resp.Status, string(resp.Body))

	var login struct {
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}
	resp.DecodeData(t, &login)
	require.NotEmpty(t, login.Token)
	require.NotEmpty(t, login.RefreshToken)

	// 登录签发的 token 能穿过同一套 AuthJWT + 园所状态 + Casbin 中间件。
	profile := h.Get(t, "/api/v1/common/user/profile", login.Token)
	require.Equal(t, http.StatusOK, profile.Status, string(profile.Body))

	var count int64
	require.NoError(t, h.DB.Unscoped().Model(&domain.User{}).
		Where("open_id = ?", openID).Count(&count).Error)
	require.EqualValues(t, 1, count, "登录应落一条真实用户行；数量为 0 说明打到了真实微信或没连测试库")

	calls := h.WeChat.Calls()
	require.Len(t, calls, 1)
	assert.Equal(t, "Code2Session", calls[0].Method)
	assert.Equal(t, "parent", calls[0].Role, "假微信应收到客户端角色，而不是被写死的端")

	// 未登记的 code 必须失败——替身不会回退到真实网络请求。
	bad := h.PostJSON(t, "/api/v1/auth/login/wechat", map[string]string{
		"code": "code-not-stubbed",
		"role": "parent",
	})
	assert.Equal(t, http.StatusBadRequest, bad.Status, string(bad.Body))
}

// 软删行仍占用行位置（后续 phone_active 唯一索引的坑），清理必须硬删。
func TestHardDeleteRemovesSoftDeletedRows(t *testing.T) {
	h := authharness.New(t)
	openID := fmt.Sprintf("openid-161-cleanup-%d", time.Now().UnixNano())
	user := &domain.User{OpenID: openID}
	require.NoError(t, h.DB.Create(user).Error)
	require.NoError(t, h.DB.Delete(user).Error, "gorm Delete 只写 deleted_at")

	var softCount, rawCount int64
	require.NoError(t, h.DB.Model(&domain.User{}).Where("id = ?", user.ID).Count(&softCount).Error)
	require.NoError(t, h.DB.Unscoped().Model(&domain.User{}).Where("id = ?", user.ID).Count(&rawCount).Error)
	require.Zero(t, softCount)
	require.EqualValues(t, 1, rawCount, "前提：软删后行仍在表里")

	h.HardDelete(t, &domain.User{}, "id = ?", user.ID)

	require.NoError(t, h.DB.Unscoped().Model(&domain.User{}).Where("id = ?", user.ID).Count(&rawCount).Error)
	assert.Zero(t, rawCount, "HardDelete 必须绕过软删，真正删掉行")
}

// 后续登录/绑定限流依赖 INCR + EXPIRE；假 Redis 必须支持这些命令，
// 让限流测试不依赖开发环境 Redis。
func TestFakeRedisSupportsRateLimitCommands(t *testing.T) {
	h := authharness.New(t)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(middleware.RateLimit(h.RedisClient, "harness_test", 2, time.Minute))
	engine.GET("/limited", func(c *gin.Context) { c.Status(http.StatusOK) })
	server := httptest.NewServer(engine)
	defer server.Close()

	statuses := make([]int, 0, 3)
	for i := 0; i < 3; i++ {
		resp, err := http.Get(server.URL + "/limited")
		require.NoError(t, err)
		_ = resp.Body.Close()
		statuses = append(statuses, resp.StatusCode)
	}

	assert.Equal(t, []int{http.StatusOK, http.StatusOK, http.StatusTooManyRequests}, statuses,
		"同一 IP 第 3 次请求应被限流；失败说明假 Redis 不支持 INCR/EXPIRE")
}
