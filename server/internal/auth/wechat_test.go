package auth

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClient() *WeChatClient {
	return NewWeChatClient("wx_teacher", "sec_teacher", "wx_parent", "sec_parent", "release", deadRedis())
}

// deadRedis 指向不存在的地址：Get 报错会跳过缓存、Set 的错误被忽略，
// 因此 GetAccessToken 会直接走 HTTP 桩，不需要真实 Redis 或 miniredis。
// MaxRetries=0 关掉重连，否则每个用例要等几秒拨号超时。
func deadRedis() *redis.Client {
	return redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", MaxRetries: 0, DialTimeout: 50 * time.Millisecond})
}

// roundTripFunc 用闭包替换 http.Client 的传输层，免去起真实 HTTP 服务。
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func stubHTTP(w *WeChatClient, body string) {
	w.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	})}
}

// --- 凭据选取 ---

// 2.1 未知 / 空 role 必须显式报错，不猜测、不回退另一端。
func TestCredentialsUnknownRole(t *testing.T) {
	w := newTestClient()

	for _, role := range []string{"", "admin", "doctor", "TEACHER", "Parent"} {
		_, _, err := w.credentials(role)
		assert.Error(t, err, "role=%q", role)
		assert.Contains(t, err.Error(), "unknown role")
	}
}

// 2.1 / 2.4 / 2.5 每端取自己的凭据，不得串。
func TestCredentialsPerEnd(t *testing.T) {
	w := newTestClient()

	id, secret, err := w.credentials("teacher")
	require.NoError(t, err)
	assert.Equal(t, "wx_teacher", id)
	assert.Equal(t, "sec_teacher", secret)

	id, secret, err = w.credentials("parent")
	require.NoError(t, err)
	assert.Equal(t, "wx_parent", id)
	assert.Equal(t, "sec_parent", secret)
}

// 未配置某端时该端报错，绝不回退到另一端凭据。
func TestCredentialsNoFallbackToOtherEnd(t *testing.T) {
	w := NewWeChatClient("wx_teacher", "sec_teacher", "", "", "release", nil)

	_, _, err := w.credentials("parent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `role "parent" not configured`)

	// 老师端仍可用，不受影响
	id, _, err := w.credentials("teacher")
	require.NoError(t, err)
	assert.Equal(t, "wx_teacher", id)
}

// 缺 secret 也算未配置，不能拿 appid 就去换会话。
func TestCredentialsMissingSecret(t *testing.T) {
	w := NewWeChatClient("wx_teacher", "", "wx_parent", "sec_parent", "release", nil)

	_, _, err := w.credentials("teacher")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

// --- Code2Session ---

// 2.1 role 未知时在发起 HTTP 请求前就报错。
func TestCode2SessionUnknownRoleFailsFast(t *testing.T) {
	w := newTestClient()
	// 传输层置为会 panic 的桩：若走到 HTTP 说明没在凭据阶段拦住
	w.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("must not reach HTTP when role is unknown")
		return nil, nil
	})}

	_, err := w.Code2Session(context.Background(), "some-code", "admin")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown role")
}

// 2.2 微信侧 errcode/errmsg 必须进错误链，便于区分「凭据错」与网络抖动。
func TestCode2SessionErrorCarriesErrCode(t *testing.T) {
	w := newTestClient()
	stubHTTP(w, `{"errcode":40029,"errmsg":"invalid code"}`)

	_, err := w.Code2Session(context.Background(), "bad-code", "teacher")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "40029")
	assert.Contains(t, err.Error(), "invalid code")
}

// 2.2 取得成功时返回 openid/unionid，不静默丢字段。
func TestCode2SessionSuccess(t *testing.T) {
	w := newTestClient()
	stubHTTP(w, `{"openid":"o_abc","session_key":"sk","unionid":"u_abc"}`)

	sess, err := w.Code2Session(context.Background(), "good-code", "parent")
	require.NoError(t, err)
	assert.Equal(t, "o_abc", sess.OpenID)
	assert.Equal(t, "u_abc", sess.UnionID)
}

// 换取会话的 URL 必须用该端的 appid —— 串了 appid 就是 openid 串号的根源。
func TestCode2SessionUsesEndAppID(t *testing.T) {
	var gotURL string
	w := newTestClient()
	w.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotURL = r.URL.String()
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"openid":"o"}`)),
		}, nil
	})}

	_, err := w.Code2Session(context.Background(), "c", "teacher")
	require.NoError(t, err)
	assert.Contains(t, gotURL, "appid=wx_teacher")
	assert.NotContains(t, gotURL, "wx_parent")

	gotURL = ""
	_, err = w.Code2Session(context.Background(), "c", "parent")
	require.NoError(t, err)
	assert.Contains(t, gotURL, "appid=wx_parent")
	assert.NotContains(t, gotURL, "wx_teacher")
}

// --- access_token 缓存键 ---

// 2.3 两个 AppID 的缓存键必须不同，否则互相污染 token。
func TestAccessTokenCacheKeyPerAppID(t *testing.T) {
	kTeacher := accessTokenCacheKey("wx_teacher")
	kParent := accessTokenCacheKey("wx_parent")

	assert.NotEqual(t, kTeacher, kParent, "两个 AppID 共用缓存键会互相污染 access_token")
	assert.Equal(t, "wechat:access_token:wx_teacher", kTeacher)
	assert.Equal(t, "wechat:access_token:wx_parent", kParent)
}

// --- 端专属方法的凭据 ---

// 2.5 教师邀请码二维码必须走老师端凭据，否则生成的是家长端小程序码。
func TestGetWXACodeUnlimitUsesTeacherCredentials(t *testing.T) {
	var urls []string
	w := newTestClient()
	w.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		urls = append(urls, r.URL.String())
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"errcode":40001,"errmsg":"invalid credential"}`)),
		}, nil
	})}

	// 老师端凭据未配置时必须报错，不得回退到家长端
	onlyParent := NewWeChatClient("", "", "wx_parent", "sec_parent", "release", deadRedis())
	_, err := onlyParent.GetWXACodeUnlimit(context.Background(), "scene", "pages/index/index")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `role "teacher" not configured`)

	// 配置齐全时，取 token 用的是老师端 appid
	_, _ = w.GetWXACodeUnlimit(context.Background(), "scene", "pages/index/index")
	require.NotEmpty(t, urls)
	assert.Contains(t, urls[0], "appid=wx_teacher", "邀请码二维码必须指向老师端小程序")
	assert.NotContains(t, strings.Join(urls, " "), "wx_parent")
}

// 2.4 家长绑定手机号必须走家长端凭据（getPhoneNumber 仅家长端用）。
func TestGetPhoneNumberUsesParentCredentials(t *testing.T) {
	var urls []string
	w := newTestClient()
	w.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		urls = append(urls, r.URL.String())
		// 第一次是取 access_token，第二次才是换取手机号
		if len(urls) == 1 {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"access_token":"at","expires_in":7200}`)),
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"errcode":0,"phone_info":{"purePhoneNumber":"13800000000"}}`)),
		}, nil
	})}

	_, err := w.GetPhoneNumber(context.Background(), "phone-code")
	require.NoError(t, err)
	require.NotEmpty(t, urls)
	assert.Contains(t, urls[0], "appid=wx_parent", "getPhoneNumber 的 code 由家长端签发，必须用家长端凭据换取")
	assert.NotContains(t, strings.Join(urls, " "), "wx_teacher")

	// 家长端凭据未配置时必须报错，不得回退到老师端
	onlyTeacher := NewWeChatClient("wx_teacher", "sec_teacher", "", "", "release", deadRedis())
	_, err = onlyTeacher.GetPhoneNumber(context.Background(), "phone-code")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `role "parent" not configured`)
}

// --- 家长端任务分享码（B3 5.4）---

// 家长端小程序码必须走家长端凭据，并带上调用方给的 envVersion。
// 用老师端凭据会生成老师端小程序码（扫码进错端）；漏传 env_version 会生成正式的码，
// 开发/测试环境扫不进去。
func TestGetParentWXACodeUnlimitUsesParentCredentialsAndEnv(t *testing.T) {
	var urls []string
	var posts []string
	w := NewWeChatClient("wx_teacher", "sec_teacher", "wx_parent", "sec_parent", "trial", deadRedis())
	w.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		urls = append(urls, r.URL.String())
		body := ""
		if r.Body != nil {
			b, _ := io.ReadAll(r.Body)
			body = string(b)
			posts = append(posts, body)
		}
		respBody := `{"access_token":"at","expires_in":7200}`
		if r.Method == http.MethodPost {
			// 微信的 getwxacodeunlimit 成功时返回 PNG 二进制，JSON 表示错误
			respBody = "\x89PNG\r\n\x1a\nfake"
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(respBody)),
		}, nil
	})}

	png, err := w.GetParentWXACodeUnlimit(context.Background(), "t=42", "pages/tasks/detail", "")
	require.NoError(t, err)
	assert.NotEmpty(t, png)

	require.NotEmpty(t, urls)
	assert.Contains(t, urls[0], "appid=wx_parent", "任务分享码必须指向家长端小程序")
	assert.NotContains(t, strings.Join(urls, " "), "wx_teacher")

	require.Len(t, posts, 1)
	assert.Contains(t, posts[0], `"scene":"t=42"`)
	assert.Contains(t, posts[0], `"page":"pages/tasks/detail"`)
	assert.Contains(t, posts[0], `"check_path":false`)
	assert.Contains(t, posts[0], `"env_version":"trial"`, "缺省必须用配置里的 code_env_version，不能写死 release")

	// 显式传值覆盖配置
	_, err = w.GetParentWXACodeUnlimit(context.Background(), "t=42", "pages/tasks/detail", "develop")
	require.NoError(t, err)
	assert.Contains(t, posts[1], `"env_version":"develop"`)
}

// 非法的 envVersion 必须在发起 HTTP 前报错：静默回退会产出「另一个版本的码」。
func TestGetParentWXACodeUnlimitRejectsInvalidEnv(t *testing.T) {
	w := NewWeChatClient("wx_teacher", "sec_teacher", "wx_parent", "sec_parent", "release", deadRedis())
	w.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("invalid env_version must not reach HTTP")
		return nil, nil
	})}

	_, err := w.GetParentWXACodeUnlimit(context.Background(), "t=1", "pages/tasks/detail", "production")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidCodeEnvVersion, "配置非法要能被调用方识别，而不是只能匹配字符串")
	assert.Contains(t, err.Error(), "invalid env_version")
}
