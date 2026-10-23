package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

type PhoneNumberResp struct {
	ErrCode   int    `json:"errcode"`
	ErrMsg    string `json:"errmsg"`
	PhoneInfo struct {
		PurePhoneNumber string `json:"purePhoneNumber"`
		CountryCode     string `json:"countryCode"`
	} `json:"phone_info"`
}

type Code2SessionResp struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionID    string `json:"unionid"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

type accessTokenResp struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

type wxacodeReq struct {
	Scene      string `json:"scene"`
	Page       string `json:"page"`
	CheckPath  bool   `json:"check_path"`
	EnvVersion string `json:"env_version"`
}

// WeChatClient 微信小程序开放能力客户端，持有两端各自独立的凭据。
//
// 两端 AppID 必须分开：openid 是「用户×小程序」二元的，共用 AppID 会让同一个人
// 在两端拿到同一个 openid，只命中一条 User，另一端角色丢失、接口全 403。
type WeChatClient struct {
	teacherAppID  string
	teacherSecret string
	parentAppID   string
	parentSecret  string
	// codeEnvVersion 小程序码指向的版本；空值按 release 处理。
	codeEnvVersion string
	rdb            *redis.Client
	client         *http.Client
}

func NewWeChatClient(teacherAppID, teacherSecret, parentAppID, parentSecret, codeEnvVersion string, rdb *redis.Client) *WeChatClient {
	if codeEnvVersion == "" {
		codeEnvVersion = "release"
	}
	return &WeChatClient{
		teacherAppID:   teacherAppID,
		teacherSecret:  teacherSecret,
		parentAppID:    parentAppID,
		parentSecret:   parentSecret,
		codeEnvVersion: codeEnvVersion,
		rdb:            rdb,
		client:         &http.Client{Timeout: 10 * time.Second},
	}
}

// CodeEnvVersion 返回配置的小程序码版本（默认 release）。
func (w *WeChatClient) CodeEnvVersion() string {
	if w == nil || w.codeEnvVersion == "" {
		return "release"
	}
	return w.codeEnvVersion
}

// credentials 按端取凭据。role 只接受 "teacher" / "parent"；
// 未知 role 或该端未配置一律报错，绝不回退到另一端——回退即重现 openid 串号。
func (w *WeChatClient) credentials(role string) (appID, secret string, err error) {
	switch role {
	case "teacher":
		appID, secret = w.teacherAppID, w.teacherSecret
	case "parent":
		appID, secret = w.parentAppID, w.parentSecret
	default:
		return "", "", fmt.Errorf("wechat credentials: unknown role %q", role)
	}
	if appID == "" || secret == "" {
		return "", "", fmt.Errorf("wechat credentials: role %q not configured (need both app_id and secret)", role)
	}
	return appID, secret, nil
}

// Code2Session 用指定端的凭据换取微信登录会话。
// role 决定用哪套凭据；错误里必须带微信侧 errcode/errmsg，便于区分「凭据错」与网络抖动。
func (w *WeChatClient) Code2Session(ctx context.Context, code, role string) (*Code2SessionResp, error) {
	appID, secret, err := w.credentials(role)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		appID, secret, code,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result Code2SessionResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.ErrCode != 0 {
		return nil, fmt.Errorf("wechat code2session failed: %d %s", result.ErrCode, result.ErrMsg)
	}
	return &result, nil
}

// accessTokenCacheKey 缓存键必须带 appid：两个 AppID 共用一个键会互相污染 token，
// 后续 getPhoneNumber / 二维码全部报微信 40001，症状指向「凭据错」而非「缓存串了」。
func accessTokenCacheKey(appID string) string {
	return "wechat:access_token:" + appID
}

// GetAccessToken 获取微信 access_token，优先用 Redis 缓存。
func (w *WeChatClient) GetAccessToken(ctx context.Context, appID, secret string) (string, error) {
	cacheKey := accessTokenCacheKey(appID)

	if cached, err := w.rdb.Get(ctx, cacheKey).Result(); err == nil && cached != "" {
		return cached, nil
	}

	url := fmt.Sprintf(
		"https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s",
		appID, secret,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	var result accessTokenResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.ErrCode != 0 {
		return "", fmt.Errorf("get access_token failed: %d %s", result.ErrCode, result.ErrMsg)
	}

	// TTL 比 expires_in 略短，留缓冲
	ttl := time.Duration(result.ExpiresIn-120) * time.Second
	if ttl <= 0 {
		ttl = time.Duration(result.ExpiresIn) * time.Second
	}
	_ = w.rdb.Set(ctx, cacheKey, result.AccessToken, ttl).Err()

	return result.AccessToken, nil
}

// GetWXACodeUnlimit 生成不限数量的小程序码，返回 PNG 二进制。
// 固定走老师端凭据：调用方是教师邀请码二维码（teacher_service），码必须指向老师端小程序，
// 用家长端凭据会生成家长端小程序码，扫码进错端、重绑主路径断掉。
func (w *WeChatClient) GetWXACodeUnlimit(ctx context.Context, scene, page string) ([]byte, error) {
	return w.getWXACodeUnlimit(ctx, "teacher", scene, page, "release")
}

// ErrInvalidCodeEnvVersion 小程序码版本配置非法（wechat.code_env_version）。
// 单独成哨兵错误：调用方要能把它和「微信接口失败」区分开，给出可操作的提示。
var ErrInvalidCodeEnvVersion = errors.New("wxacode: invalid env_version")

// GetParentWXACodeUnlimit 生成指向**家长端**的小程序码（任务分享码）。
//
// 必须走家长端凭据：用老师端凭据会生成老师端小程序码，家长扫码进错端，任务触达不到。
// envVersion 取 release / trial / develop；空值回退配置值，非法值显式报错
// （静默回退会把「配置写错」变成「生成了另一个版本的码」）。
func (w *WeChatClient) GetParentWXACodeUnlimit(ctx context.Context, scene, page, envVersion string) ([]byte, error) {
	if envVersion == "" {
		envVersion = w.CodeEnvVersion()
	}
	switch envVersion {
	case "release", "trial", "develop":
	default:
		return nil, fmt.Errorf("%w: %q (want release|trial|develop)", ErrInvalidCodeEnvVersion, envVersion)
	}
	return w.getWXACodeUnlimit(ctx, "parent", scene, page, envVersion)
}

func (w *WeChatClient) getWXACodeUnlimit(ctx context.Context, role, scene, page, envVersion string) ([]byte, error) {
	appID, secret, err := w.credentials(role)
	if err != nil {
		return nil, err
	}
	token, err := w.GetAccessToken(ctx, appID, secret)
	if err != nil {
		return nil, fmt.Errorf("wxacode access_token: %w", err)
	}

	url := fmt.Sprintf("https://api.weixin.qq.com/wxa/getwxacodeunlimit?access_token=%s", token)

	body := wxacodeReq{
		Scene:      scene,
		Page:       page,
		CheckPath:  false,
		EnvVersion: envVersion,
	}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 微信返回 JSON 表示错误，返回二进制表示成功
	if len(data) > 0 && data[0] == '{' {
		var errResp struct {
			ErrCode int    `json:"errcode"`
			ErrMsg  string `json:"errmsg"`
		}
		if json.Unmarshal(data, &errResp) == nil && errResp.ErrCode != 0 {
			return nil, fmt.Errorf("wxacode getwxacodeunlimit failed: %d %s", errResp.ErrCode, errResp.ErrMsg)
		}
	}

	return data, nil
}

// GetPhoneNumber 通过微信小程序 getPhoneNumber 返回的 code 换取手机号。
// 固定走家长端凭据：getPhoneNumber 组件仅家长端使用（绑定手机号），
// 该 code 由家长端小程序签发，必须用家长端凭据换取。
func (w *WeChatClient) GetPhoneNumber(ctx context.Context, code string) (*PhoneNumberResp, error) {
	appID, secret, err := w.credentials("parent")
	if err != nil {
		return nil, err
	}
	token, err := w.GetAccessToken(ctx, appID, secret)
	if err != nil {
		return nil, fmt.Errorf("getPhoneNumber access_token: %w", err)
	}

	url := fmt.Sprintf("https://api.weixin.qq.com/wxa/business/getuserphonenumber?access_token=%s", token)

	body := map[string]string{"code": code}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result PhoneNumberResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.ErrCode != 0 {
		return nil, fmt.Errorf("wechat getPhoneNumber failed: %d %s", result.ErrCode, result.ErrMsg)
	}
	return &result, nil
}
