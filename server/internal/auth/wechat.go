package auth

import (
	"bytes"
	"context"
	"encoding/json"
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

type WeChatClient struct {
	appID  string
	secret string
	rdb    *redis.Client
	client *http.Client
}

func NewWeChatClient(appID, secret string, rdb *redis.Client) *WeChatClient {
	return &WeChatClient{
		appID:  appID,
		secret: secret,
		rdb:    rdb,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (w *WeChatClient) Code2Session(ctx context.Context, code string) (*Code2SessionResp, error) {
	url := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		w.appID, w.secret, code,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result Code2SessionResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.ErrCode != 0 {
		return nil, fmt.Errorf("wechat code2session failed: %d %s", result.ErrCode, result.ErrMsg)
	}
	return &result, nil
}

// GetAccessToken 获取微信 access_token，优先用 Redis 缓存。
func (w *WeChatClient) GetAccessToken(ctx context.Context) (string, error) {
	const cacheKey = "wechat:access_token"

	if cached, err := w.rdb.Get(ctx, cacheKey).Result(); err == nil && cached != "" {
		return cached, nil
	}

	url := fmt.Sprintf(
		"https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s",
		w.appID, w.secret,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

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
func (w *WeChatClient) GetWXACodeUnlimit(ctx context.Context, scene, page string) ([]byte, error) {
	token, err := w.GetAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("wxacode access_token: %w", err)
	}

	url := fmt.Sprintf("https://api.weixin.qq.com/wxa/getwxacodeunlimit?access_token=%s", token)

	body := wxacodeReq{
		Scene:      scene,
		Page:       page,
		CheckPath:  false,
		EnvVersion: "release",
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
	defer resp.Body.Close()

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
func (w *WeChatClient) GetPhoneNumber(ctx context.Context, code string) (*PhoneNumberResp, error) {
	token, err := w.GetAccessToken(ctx)
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
	defer resp.Body.Close()

	var result PhoneNumberResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.ErrCode != 0 {
		return nil, fmt.Errorf("wechat getPhoneNumber failed: %d %s", result.ErrCode, result.ErrMsg)
	}
	return &result, nil
}
