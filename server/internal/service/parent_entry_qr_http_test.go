package service_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/service"
	"kindergarten-server/internal/testutil/authharness"
)

// 家长端园所二维码（#176）。
//
// 断言落在 HTTP 行为上：管理端按当前园所生成码、scene 是 kg_<public_code>、
// page 指向家长端真实页面、env_version 透传给**家长端**凭据。
// 走 #161 的单一 HTTP 缝（真实 Gin + 测试库 + 假微信）。

const parentEntryQRPath = "/api/v1/admin/kindergarten/parent-qrcode"

type parentEntryQRData struct {
	PublicCode  string `json:"public_code"`
	Scene       string `json:"scene"`
	Page        string `json:"page"`
	EnvVersion  string `json:"env_version"`
	ImageBase64 string `json:"image_base64"`
}

func seedEntryQRClient(t *testing.T, h *authharness.Harness, role string) *domain.Client {
	t.Helper()
	// FindEnabledForRole 要求该角色「恰有一个」enabled 客户端；同包的 client
	// 测试会在共享测试库里留下它们同步出的行，先清空再播种（与 newClientService 同约定）。
	h.HardDelete(t, &domain.Client{}, "1 = 1")
	client := &domain.Client{
		AppID:     fmt.Sprintf("wx-176-%s-%d", role, time.Now().UnixNano()),
		Role:      role,
		ScopeKind: domain.ClientScopeShared,
		Status:    domain.ClientStatusEnabled,
	}
	require.NoError(t, h.DB.Create(client).Error, "创建客户端失败")
	t.Cleanup(func() { h.HardDelete(t, &domain.Client{}, "app_id = ?", client.AppID) })
	return client
}

func seedEntryQRKindergarten(t *testing.T, h *authharness.Harness, channel, code string) *domain.Kindergarten {
	t.Helper()
	kg := &domain.Kindergarten{
		Name:       fmt.Sprintf("176-园所-%d", time.Now().UnixNano()),
		Status:     domain.KindergartenStatusActive,
		AppChannel: channel,
		PublicCode: code,
	}
	require.NoError(t, h.DB.Create(kg).Error, "创建园所失败")
	t.Cleanup(func() { h.HardDelete(t, &domain.Kindergarten{}, "id = ?", kg.ID) })
	return kg
}

func decodeParentEntryQR(t *testing.T, resp *authharness.Response) parentEntryQRData {
	t.Helper()
	var data parentEntryQRData
	resp.DecodeData(t, &data)
	return data
}

// scene 必须是 kg_<public_code>、page 必须是家长端真实页面、env_version 透传，
// 且必须走 GetParentWXACodeUnlimit —— 用老师端凭据会生成另一个小程序的码。
func TestParentEntryQRCodeUsesParentCredentialsAndEnvVersion(t *testing.T) {
	h := authharness.New(t)
	seedEntryQRClient(t, h, domain.ClientRoleParent)
	kg := seedEntryQRKindergarten(t, h, domain.AppChannelShared, "qrtest0001")
	png := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a}
	h.WeChat.StubWXACode("kg_"+kg.PublicCode, png)
	token := adminToken(t, h, 991176, kg.ID)

	resp := h.Get(t, parentEntryQRPath+"?env_version=trial", token)
	require.Equal(t, http.StatusOK, resp.Status, "生成园所二维码必须成功；body=%s", resp.Body)

	data := decodeParentEntryQR(t, resp)
	assert.Equal(t, kg.PublicCode, data.PublicCode)
	assert.Equal(t, "kg_"+kg.PublicCode, data.Scene, "scene 语法必须是 kg_<public_code>")
	assert.LessOrEqual(t, len(data.Scene), 32, "scene 不得超过微信 32 字符上限")
	assert.Equal(t, "pages/index/index", data.Page)
	assert.Equal(t, "trial", data.EnvVersion, "env_version 必须透传，不能写死 release")

	decoded, err := base64.StdEncoding.DecodeString(data.ImageBase64)
	require.NoError(t, err, "image_base64 必须是合法 base64")
	assert.Equal(t, png, decoded)

	calls := h.WeChat.Calls()
	require.Len(t, calls, 1, "一次生成只应调用一次微信")
	assert.Equal(t, "GetParentWXACodeUnlimit", calls[0].Method,
		"必须走家长端凭据；老师端 GetWXACodeUnlimit 会生成另一个小程序的码")
	assert.Equal(t, "kg_"+kg.PublicCode, calls[0].Code)
	assert.Equal(t, "pages/index/index", calls[0].Page)
	assert.Equal(t, "trial", calls[0].EnvVersion)
}

// 空 env_version 回落到配置值（部署环境决定 release/develop）。
func TestParentEntryQRCodeFallsBackToConfiguredEnvVersion(t *testing.T) {
	h := authharness.New(t)
	seedEntryQRClient(t, h, domain.ClientRoleParent)
	kg := seedEntryQRKindergarten(t, h, domain.AppChannelShared, "qrtest0002")
	h.WeChat.SetCodeEnvVersion("develop")
	png := []byte{1, 2, 3}
	h.WeChat.StubWXACode("kg_"+kg.PublicCode, png)
	token := adminToken(t, h, 991176, kg.ID)

	resp := h.Get(t, parentEntryQRPath, token)
	require.Equal(t, http.StatusOK, resp.Status, "body=%s", resp.Body)
	data := decodeParentEntryQR(t, resp)
	assert.Equal(t, "develop", data.EnvVersion)
	assert.Equal(t, "develop", h.WeChat.Calls()[0].EnvVersion)
}

// 非法 env_version 必须显式拒绝：静默回落会把配置错误变成「另一个版本的码」。
func TestParentEntryQRCodeRejectsInvalidEnvVersion(t *testing.T) {
	h := authharness.New(t)
	seedEntryQRClient(t, h, domain.ClientRoleParent)
	kg := seedEntryQRKindergarten(t, h, domain.AppChannelShared, "qrtest0003")
	token := adminToken(t, h, 991176, kg.ID)

	resp := h.Get(t, parentEntryQRPath+"?env_version=beta", token)
	assert.Equal(t, http.StatusUnprocessableEntity, resp.Status, "非法 env_version 必须 422；body=%s", resp.Body)
	assert.Empty(t, h.WeChat.Calls(), "参数校验失败不得调用微信")
}

// shared client 只服务 shared 园：专属通道园必须拒绝，且不能先调微信再生效。
func TestParentEntryQRCodeRejectsKindergartenOutsideClientScope(t *testing.T) {
	h := authharness.New(t)
	seedEntryQRClient(t, h, domain.ClientRoleParent)
	kg := seedEntryQRKindergarten(t, h, domain.AppChannelDedicated, "qrtest0004")
	h.WeChat.StubAnyWXACode([]byte{1})
	token := adminToken(t, h, 991176, kg.ID)

	resp := h.Get(t, parentEntryQRPath, token)
	assert.Equal(t, http.StatusConflict, resp.Status, "通道不匹配必须拒绝；body=%s", resp.Body)
	assert.Empty(t, h.WeChat.Calls(), "通道不匹配必须在调用微信前拒绝")
}

// 没有 public_code 的存量园在首次生成时补齐；再生成复用同一码，可重复。
func TestParentEntryQRCodeAssignsMissingPublicCodeAndIsRepeatable(t *testing.T) {
	h := authharness.New(t)
	seedEntryQRClient(t, h, domain.ClientRoleParent)
	kg := seedEntryQRKindergarten(t, h, domain.AppChannelShared, "")
	h.WeChat.StubAnyWXACode([]byte{0x89, 'P', 'N', 'G'})
	token := adminToken(t, h, 991176, kg.ID)

	first := h.Get(t, parentEntryQRPath, token)
	require.Equal(t, http.StatusOK, first.Status, "body=%s", first.Body)
	data := decodeParentEntryQR(t, first)
	require.Len(t, data.PublicCode, service.PublicCodeLength, "缺少公开码的园所必须自动补齐")
	assert.Equal(t, "kg_"+data.PublicCode, data.Scene)

	var stored domain.Kindergarten
	require.NoError(t, h.DB.First(&stored, kg.ID).Error)
	assert.Equal(t, data.PublicCode, stored.PublicCode, "补齐的公开码必须落库")

	second := h.Get(t, parentEntryQRPath, token)
	require.Equal(t, http.StatusOK, second.Status)
	assert.Equal(t, data.PublicCode, decodeParentEntryQR(t, second).PublicCode, "重复生成必须复用同一公开码")
}

func TestParentEntryQRCodeUnknownKindergartenIsNotFound(t *testing.T) {
	h := authharness.New(t)
	seedEntryQRClient(t, h, domain.ClientRoleParent)
	token := adminToken(t, h, 991176, 987654321)

	resp := h.Get(t, parentEntryQRPath, token)
	assert.Equal(t, http.StatusNotFound, resp.Status, "body=%s", resp.Body)
}

// page 必须真实存在于家长端 pages.json。CheckPath:false 不会校验路径，
// 写错只会生成一张扫码即死的码。
func TestParentEntryQRCodePageExistsInParentPagesJSON(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok, "无法定位测试文件路径")
	repoRoot := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(file))))
	pagesPath := filepath.Join(repoRoot, "parent", "src", "pages.json")

	raw, err := os.ReadFile(pagesPath)
	require.NoError(t, err, "读取家长端 pages.json 失败（%s）", pagesPath)

	var parsed struct {
		Pages []struct {
			Path string `json:"path"`
		} `json:"pages"`
	}
	require.NoError(t, json.Unmarshal(raw, &parsed), "解析家长端 pages.json 失败")

	found := false
	for _, p := range parsed.Pages {
		if p.Path == service.ParentEntryQRPage {
			found = true
			break
		}
	}
	assert.True(t, found, "二维码 page %q 不在 parent/src/pages.json", service.ParentEntryQRPage)
}
