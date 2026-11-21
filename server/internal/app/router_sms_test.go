package app

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// #162：短信登录与验证码发送在 v1 关闭。两个端点的一致行为是 403（路由保留、
// 明确拒绝），而不是 404——客户端能收到「功能未开放」的信号，而不是误判为路径写错。
//
// 这是规格 #160「安全收口」的验收点：SMSLogin 不看 role、没有频控，验证码只在
// Redis 里生成从不真正下发，任何 role 都不能经由短信拿到 token。
func TestSMSEndpointsAreForbiddenInV1(t *testing.T) {
	rdb, _ := newFakeRedisClient(t)
	engine := newTestEngine(t, rdb)

	// 先塞一个「正确」的验证码：即使 Redis 里已有可用 code，端点也不能把它换成 token。
	require.NoError(t, rdb.Set(context.Background(), "sms:login:13800000001", "123456", 5*time.Minute).Err())

	cases := []struct {
		name string
		path string
		body string
	}{
		{"send code", "/api/v1/auth/sms/send", `{"phone":"13800000001","scene":"login"}`},
		{"login as parent", "/api/v1/auth/login/sms", `{"phone":"13800000001","code":"123456","role":"parent"}`},
		{"login as teacher", "/api/v1/auth/login/sms", `{"phone":"13800000001","code":"123456","role":"teacher"}`},
		{"login as admin", "/api/v1/auth/login/sms", `{"phone":"13800000001","code":"123456","role":"admin"}`},
		{"login with malformed body", "/api/v1/auth/login/sms", `{}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := doRequest(t, engine, http.MethodPost, tc.path, "", tc.body)
			require.Equal(t, http.StatusForbidden, w.Code, "body: %s", w.Body.String())

			var env struct {
				Code    int             `json:"code"`
				Message string          `json:"message"`
				Data    json.RawMessage `json:"data"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
			assert.Equal(t, 40300, env.Code)
			assert.NotEmpty(t, env.Message)
			assert.JSONEq(t, "null", string(env.Data), "被拒绝的短信端点不能返回任何数据（含 token）")

			// 兜底：不论响应形状如何，都不允许出现 token 字段。
			var raw map[string]any
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &raw))
			for _, key := range []string{"token", "refresh_token", "access_token"} {
				_, found := raw[key]
				assert.Falsef(t, found, "响应中不应出现 %s", key)
			}
		})
	}
}

// 关闭必须是端点级行为，而不是「验证码不对所以登不上」：
// 不带任何手机号/验证码的请求同样被拒绝。
func TestSMSEndpointsRejectWithoutReadingCredentials(t *testing.T) {
	rdb, _ := newFakeRedisClient(t)
	engine := newTestEngine(t, rdb)

	for _, path := range []string{"/api/v1/auth/sms/send", "/api/v1/auth/login/sms"} {
		w := doRequest(t, engine, http.MethodPost, path, "", "")
		assert.Equalf(t, http.StatusForbidden, w.Code, "%s 应在读取请求体前就拒绝：%s", path, w.Body.String())
	}
}
