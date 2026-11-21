package service_test

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/testutil/authharness"
)

// 公开品牌 HTTP 接口（#170）：未登录可访问、只含公开字段、按 public_code
// 定位而非连续 id、软删 404。走 #161 的单一 HTTP 缝（真实 Gin + 测试库）。

type publicBrandEnvelope struct {
	Code int            `json:"code"`
	Data map[string]any `json:"data"`
}

func publicBrandPath(code string) string {
	return "/api/v1/public/kindergartens/" + code
}

func TestPublicBrandEndpointReturnsOnlyPublicFields(t *testing.T) {
	h := authharness.New(t)
	kg := newPublicBrandKindergarten(t, h, "170-公开品牌园", "httpbrand01", "uploads/logo/http.png", "#1D4ED8")

	resp := h.Get(t, publicBrandPath(kg.PublicCode), "")
	require.Equal(t, http.StatusOK, resp.Status, "未登录必须能访问公开品牌接口；body=%s", resp.Body)

	var env publicBrandEnvelope
	require.NoError(t, json.Unmarshal(resp.Body, &env))
	require.Equal(t, 0, env.Code)

	assert.Equal(t, map[string]struct{}{
		"public_code": {},
		"name":        {},
		"logo_url":    {},
		"brand_color": {},
	}, keysOf(env.Data), "公开品牌只能包含名称/logo/主题色/公开码")

	assert.Equal(t, kg.PublicCode, env.Data["public_code"])
	assert.Equal(t, "170-公开品牌园", env.Data["name"])
	assert.Equal(t, "#1D4ED8", env.Data["brand_color"])

	logo, _ := env.Data["logo_url"].(string)
	assert.Contains(t, logo, "Signature=", "私有 OSS logo 必须以签名 URL 下发")
	assert.NotEqual(t, "uploads/logo/http.png", logo)

	for _, forbidden := range []string{
		"id", "address", "phone", "principal", "license_no", "capacity",
		"description", "established_at", "status", "app_channel",
	} {
		_, present := env.Data[forbidden]
		assert.False(t, present, "公开品牌不得下发私有字段 %q", forbidden)
	}
}

func TestPublicBrandEndpointSoftDeletedIsNotFound(t *testing.T) {
	h := authharness.New(t)
	kg := newPublicBrandKindergarten(t, h, "170-软删接口园", "httpbrand02", "", "")

	// 先确认可访问，再软删，保证 404 来自软删而不是码写错。
	require.Equal(t, http.StatusOK, h.Get(t, publicBrandPath(kg.PublicCode), "").Status)

	require.NoError(t, h.DB.Delete(&kg, kg.ID).Error)
	resp := h.Get(t, publicBrandPath(kg.PublicCode), "")
	assert.Equal(t, http.StatusNotFound, resp.Status, "软删园所必须 404")
}

// 连续 id 不能枚举园所：知道 id 也拿不到品牌。
func TestPublicBrandEndpointDoesNotResolveNumericID(t *testing.T) {
	h := authharness.New(t)
	kg := newPublicBrandKindergarten(t, h, "170-不可枚举园", "httpbrand03", "", "")

	resp := h.Get(t, publicBrandPath(strconv.FormatUint(kg.ID, 10)), "")
	assert.Equal(t, http.StatusNotFound, resp.Status, "数字 id 不得作为公开品牌入口")
}

func TestPublicBrandEndpointUnknownCodeIsNotFound(t *testing.T) {
	h := authharness.New(t)

	resp := h.Get(t, publicBrandPath("nosuchcode"), "")
	assert.Equal(t, http.StatusNotFound, resp.Status)
}

func keysOf(data map[string]any) map[string]struct{} {
	keys := make(map[string]struct{}, len(data))
	for key := range data {
		keys[key] = struct{}{}
	}
	return keys
}
