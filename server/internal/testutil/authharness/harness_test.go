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

// #173：旧跨端身份列必须由迁移删除。留着它们就会与 user_identities 形成
// 「一个身份两处存」的双源；测试库跑的是真实 goose 迁移，因此这里能验证到最终 schema。
func TestTestDatabaseHasNoLegacyUserIdentityColumns(t *testing.T) {
	h := authharness.New(t)

	for _, column := range []string{"open_id", "union_id"} {
		assert.Falsef(t, h.DB.Migrator().HasColumn(&domain.User{}, column),
			"users.%s 应已由 #173 迁移删除，身份只存 user_identities", column)
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

// 旧微信登录的「已下线」断言由 #175 取代：端点重新开放为新式 app_id 登录，
// 旧请求体（只有 code）由 login_test.go 的 TestWeChatLoginRequiresAppID 覆盖。

// 旧绑定端点（#164）由 #167、#168 重写：
// bind-invite 已重新上线（#168，需要待绑定 token，未认证应为 401）；
// 家长手机号绑定仍归 #167，本分支未注册，未认证请求应为 404。
func TestLegacyBindEndpointStatus(t *testing.T) {
	h := authharness.New(t)

	resp := h.PostJSON(t, "/api/v1/common/user/bind-invite", map[string]string{
		"invite_code": "legacy-invite",
	})
	assert.Equal(t, http.StatusUnauthorized, resp.Status, "bind-invite 已由 #168 上线；body=%s", resp.Body)

	resp = h.PostJSON(t, "/api/v1/common/user/bind-parent-phone", map[string]string{
		"wx_phone_code": "legacy-phone-code",
	})
	assert.Equal(t, http.StatusNotFound, resp.Status, "家长手机号绑定仍归 #167；body=%s", resp.Body)
}

// 软删行仍占用行位置（后续 phone_active 唯一索引的坑），清理必须硬删。
func TestHardDeleteRemovesSoftDeletedRows(t *testing.T) {
	h := authharness.New(t)
	user := &domain.User{Nickname: fmt.Sprintf("cleanup-161-%d", time.Now().UnixNano())}
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
