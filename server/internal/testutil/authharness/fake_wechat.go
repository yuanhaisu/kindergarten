package authharness

import (
	"context"
	"fmt"
	"sync"

	"kindergarten-server/internal/auth"
)

// WeChatCall 记录一次假微信调用，供测试断言「服务端确实按客户端角色取凭据」。
type WeChatCall struct {
	Method string
	Code   string
	Role   string
	// 小程序码参数（#176）：env_version 必须透传，page 必须指向真实页面。
	// 只对 GetWXACodeUnlimit / GetParentWXACodeUnlimit 填写。
	Page       string
	EnvVersion string
}

// FakeWeChat 是 service.WeChatAPI 的进程内替身。
//
// 行为是白名单式：只有测试显式 Stub 过的 code/scene 才能成功；未登记一律报错。
// 这样「配置缺凭据 → 回退真实网络 → 测试意外通过」不可能发生。
type FakeWeChat struct {
	mu         sync.Mutex
	envVersion string
	sessions   map[string]*auth.Code2SessionResp
	phones     map[string]string
	wxaCodes   map[string][]byte
	// anyWXACode 只在测试显式 StubAnyWXACode 后生效，用于 scene 值由被测代码
	// 生成（如存量园自动补 public_code）而无法预先登记的场景。
	anyWXACode []byte
	calls      []WeChatCall
}

func NewFakeWeChat() *FakeWeChat {
	return &FakeWeChat{
		envVersion: "release",
		sessions:   map[string]*auth.Code2SessionResp{},
		phones:     map[string]string{},
		wxaCodes:   map[string][]byte{},
	}
}

// StubCode2Session 让 code 换取指定的 openid/unionid。
func (f *FakeWeChat) StubCode2Session(code, openID, unionID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sessions[code] = &auth.Code2SessionResp{
		OpenID:     openID,
		UnionID:    unionID,
		SessionKey: "fake-session-key",
	}
}

// StubPhoneNumber 让 code 换取指定手机号。
func (f *FakeWeChat) StubPhoneNumber(code, phone string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.phones[code] = phone
}

// StubWXACode 让指定 scene 返回固定的 PNG 字节。
func (f *FakeWeChat) StubWXACode(scene string, png []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.wxaCodes[scene] = append([]byte(nil), png...)
}

// StubAnyWXACode 让未逐个登记的 scene 也返回同一 PNG。
// 只用于 scene 值由被测代码生成的用例；显式 StubWXACode 仍优先。
func (f *FakeWeChat) StubAnyWXACode(png []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.anyWXACode = append([]byte(nil), png...)
}

// SetCodeEnvVersion 设置 CodeEnvVersion 的返回值（默认 release）。
func (f *FakeWeChat) SetCodeEnvVersion(v string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.envVersion = v
}

// Calls 返回已发生的调用副本。
func (f *FakeWeChat) Calls() []WeChatCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]WeChatCall(nil), f.calls...)
}

func (f *FakeWeChat) record(call WeChatCall) {
	f.calls = append(f.calls, call)
}

func (f *FakeWeChat) Code2Session(_ context.Context, code, role string) (*auth.Code2SessionResp, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record(WeChatCall{Method: "Code2Session", Code: code, Role: role})

	sess, ok := f.sessions[code]
	if !ok {
		return nil, fmt.Errorf("fake wechat: code2session code %q 未登记（测试必须先 StubCode2Session）", code)
	}
	copied := *sess
	return &copied, nil
}

func (f *FakeWeChat) GetPhoneNumber(_ context.Context, code string) (*auth.PhoneNumberResp, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record(WeChatCall{Method: "GetPhoneNumber", Code: code})

	phone, ok := f.phones[code]
	if !ok {
		return nil, fmt.Errorf("fake wechat: getPhoneNumber code %q 未登记（测试必须先 StubPhoneNumber）", code)
	}
	resp := &auth.PhoneNumberResp{ErrCode: 0}
	resp.PhoneInfo.PurePhoneNumber = phone
	resp.PhoneInfo.CountryCode = "86"
	return resp, nil
}

func (f *FakeWeChat) GetWXACodeUnlimit(_ context.Context, scene, page string) ([]byte, error) {
	return f.wxaCode("GetWXACodeUnlimit", scene, page, "release")
}

func (f *FakeWeChat) GetParentWXACodeUnlimit(_ context.Context, scene, page, envVersion string) ([]byte, error) {
	return f.wxaCode("GetParentWXACodeUnlimit", scene, page, envVersion)
}

func (f *FakeWeChat) wxaCode(method, scene, page, envVersion string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record(WeChatCall{Method: method, Code: scene, Page: page, EnvVersion: envVersion})

	png, ok := f.wxaCodes[scene]
	if !ok {
		if f.anyWXACode == nil {
			return nil, fmt.Errorf("fake wechat: wxacode scene %q 未登记（测试必须先 StubWXACode）", scene)
		}
		png = f.anyWXACode
	}
	return append([]byte(nil), png...), nil
}

func (f *FakeWeChat) CodeEnvVersion() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.envVersion
}
