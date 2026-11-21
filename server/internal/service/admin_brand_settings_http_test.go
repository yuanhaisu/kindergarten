package service_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/service"
	"kindergarten-server/internal/storage/oss"
	"kindergarten-server/internal/testutil/authharness"
	"kindergarten-server/pkg/response"
)

// 管理端园所品牌设置（#178）。
//
// 品牌是租户对外可见的属性，因此断言必须落在两处真实行为上：管理端保存后
// 「公开品牌接口」立即反映新值（同一份 DB，无中间缓存），且写操作留审计。
// 走 #161 的单一 HTTP 缝（真实 Gin + 测试库）。

const adminKgInfoPath = "/api/v1/admin/kindergarten/info"

// adminToken 签一个管理端凭证（role=admin），Casbin 策略 p, admin, /api/v1/*, *
// 放行全部园所接口。审计只需要 user_id，不依赖用户行是否存在。
func adminToken(t *testing.T, h *authharness.Harness, userID, kgID uint64) string {
	t.Helper()
	token, err := auth.SignAccess(h.Secret, strconv.FormatUint(userID, 10), []string{"admin"}, kgID, 15*time.Minute)
	require.NoError(t, err, "签发管理端测试凭证失败")
	return token
}

func adminKgInfoData(t *testing.T, h *authharness.Harness, token string) map[string]any {
	t.Helper()
	resp := h.Get(t, adminKgInfoPath, token)
	require.Equal(t, http.StatusOK, resp.Status, "读取园所信息必须成功；body=%s", resp.Body)
	var data map[string]any
	resp.DecodeData(t, &data)
	return data
}

// 保存品牌色后：管理端回显归一化后的值、公开品牌接口立即反映、审计留痕。
func TestAdminBrandColorRoundTripReflectsInPublicBrand(t *testing.T) {
	h := authharness.New(t)
	kg := newPublicBrandKindergarten(t, h, "178-原园名", "admin178a1", "", "")
	token := adminToken(t, h, 991178, kg.ID)

	resp := h.PutJSONWithToken(t, adminKgInfoPath, token, map[string]any{
		"name":        "178-新园名",
		"brand_color": "#1d4ed8",
	})
	require.Equal(t, http.StatusOK, resp.Status, "保存园所品牌必须成功；body=%s", resp.Body)
	require.Equal(t, 0, resp.EnvelopeCode(t))

	data := adminKgInfoData(t, h, token)
	assert.Equal(t, "178-新园名", data["name"])
	assert.Equal(t, "#1D4ED8", data["brand_color"], "主题色必须归一化（大写 #RRGGBB）后回显")

	brandResp := h.Get(t, publicBrandPath(kg.PublicCode), "")
	require.Equal(t, http.StatusOK, brandResp.Status)
	var brand publicBrandEnvelope
	require.NoError(t, json.Unmarshal(brandResp.Body, &brand))
	assert.Equal(t, "178-新园名", brand.Data["name"], "管理端改名必须立即出现在公开品牌接口")
	assert.Equal(t, "#1D4ED8", brand.Data["brand_color"], "管理端改主题色必须立即出现在公开品牌接口")

	var logs []domain.OperationLog
	require.NoError(t, h.DB.Where("kindergarten_id = ? AND action = ?", kg.ID, "update_info").
		Order("id DESC").Find(&logs).Error)
	require.NotEmpty(t, logs, "保存园所信息必须写审计")
	var detail map[string]any
	require.NoError(t, json.Unmarshal([]byte(logs[0].Detail), &detail), "审计 detail 必须是保存字段的 JSON；detail=%s", logs[0].Detail)
	assert.Equal(t, "#1D4ED8", detail["brand_color"], "审计必须记录品牌色变更")
	assert.Equal(t, "178-新园名", detail["name"])
}

// 非法主题色在服务端拒绝（前端选择器可绕过）：对比度不足、颜色名、三位简写、
// 非法字符都不能写进库，也不能悄悄换成默认色。
func TestAdminBrandSettingsRejectsInvalidColor(t *testing.T) {
	h := authharness.New(t)
	kg := newPublicBrandKindergarten(t, h, "178-非法色园", "admin178b1", "", "#C2410C")
	token := adminToken(t, h, 991179, kg.ID)

	for _, invalid := range []string{"#FFEB3B", "red", "#C24", "#GGGGGG"} {
		resp := h.PutJSONWithToken(t, adminKgInfoPath, token, map[string]any{"brand_color": invalid})
		assert.Equal(t, http.StatusUnprocessableEntity, resp.Status,
			"非法主题色 %q 必须被拒绝；body=%s", invalid, resp.Body)
		assert.Equal(t, 42200, resp.EnvelopeCode(t), "非法主题色走统一校验错误码")
	}

	var stored domain.Kindergarten
	require.NoError(t, h.DB.First(&stored, kg.ID).Error)
	assert.Equal(t, "#C2410C", stored.BrandColor, "被拒绝的写入不能改动已保存的主题色")

	brandResp := h.Get(t, publicBrandPath(kg.PublicCode), "")
	require.Equal(t, http.StatusOK, brandResp.Status)
	var brand publicBrandEnvelope
	require.NoError(t, json.Unmarshal(brandResp.Body, &brand))
	assert.Equal(t, "#C2410C", brand.Data["brand_color"])
}

// 空串表示清空主题色 → 公开接口回到默认主题；nil（未下发）不改动。
func TestAdminBrandSettingsClearsBrandColorToDefault(t *testing.T) {
	h := authharness.New(t)
	kg := newPublicBrandKindergarten(t, h, "178-清空园", "admin178c1", "", "#C2410C")
	token := adminToken(t, h, 991180, kg.ID)

	// 只改名称时不下发 brand_color，已保存的主题色不能被误清。
	resp := h.PutJSONWithToken(t, adminKgInfoPath, token, map[string]any{"name": "178-只改名字"})
	require.Equal(t, http.StatusOK, resp.Status, "body=%s", resp.Body)
	assert.Equal(t, "#C2410C", adminKgInfoData(t, h, token)["brand_color"], "nil 字段不得清空主题色")

	resp = h.PutJSONWithToken(t, adminKgInfoPath, token, map[string]any{"brand_color": "  "})
	require.Equal(t, http.StatusOK, resp.Status, "body=%s", resp.Body)
	assert.Equal(t, "", adminKgInfoData(t, h, token)["brand_color"], "清空后管理端回显空串（使用默认主题）")

	brandResp := h.Get(t, publicBrandPath(kg.PublicCode), "")
	require.Equal(t, http.StatusOK, brandResp.Status)
	var brand publicBrandEnvelope
	require.NoError(t, json.Unmarshal(brandResp.Body, &brand))
	assert.Equal(t, service.DefaultBrandColor, brand.Data["brand_color"], "清空主题色后公开接口回落默认主题")
}

// 上传后的 logo 是签名 URL（签名会过期）：保存时必须归一化回 object key，
// 管理端/公开接口再各自签新的 URL。否则签名过期后品牌 logo 全挂。
func TestAdminBrandSettingsStoresLogoRefAndResignsForPreview(t *testing.T) {
	h := authharness.New(t)
	kg := newPublicBrandKindergarten(t, h, "178-logo园", "admin178d1", "", "")
	token := adminToken(t, h, 991181, kg.ID)
	installTestMediaSigner(t)

	const (
		logoKey = "uploads/logo/2026/11/uuid.png"
		// /common/upload 的实测返回形状：完整 OSS URL + 签名参数。
		uploaded = "https://test-bucket.oss-cn-hangzhou.aliyuncs.com/" + logoKey +
			"?OSSAccessKeyId=test-access-key&Expires=9999999999&Signature=deadbeef"
	)

	resp := h.PutJSONWithToken(t, adminKgInfoPath, token, map[string]any{"logo_url": uploaded})
	require.Equal(t, http.StatusOK, resp.Status, "保存 logo 必须成功；body=%s", resp.Body)

	var stored domain.Kindergarten
	require.NoError(t, h.DB.First(&stored, kg.ID).Error)
	assert.Equal(t, logoKey, stored.LogoURL, "库里只能存 object key，不能存会过期的签名 URL")

	preview, _ := adminKgInfoData(t, h, token)["logo_url"].(string)
	assert.Contains(t, preview, "Signature=", "管理端回显的 logo 必须是可预览的签名 URL")
	assert.NotContains(t, preview, "Expires=9999999999", "回显必须重新签名，不能复用上传时的签名")

	brandResp := h.Get(t, publicBrandPath(kg.PublicCode), "")
	require.Equal(t, http.StatusOK, brandResp.Status)
	var brand publicBrandEnvelope
	require.NoError(t, json.Unmarshal(brandResp.Body, &brand))
	logo, _ := brand.Data["logo_url"].(string)
	assert.Contains(t, logo, "Signature=", "公开品牌 logo 必须签名")
	signedPath, err := url.PathUnescape(logo)
	require.NoError(t, err)
	assert.Contains(t, signedPath, logoKey, "公开品牌签的必须是库里保存的 object key")

	resp = h.PutJSONWithToken(t, adminKgInfoPath, token, map[string]any{"logo_url": ""})
	require.Equal(t, http.StatusOK, resp.Status, "body=%s", resp.Body)
	assert.Equal(t, "", adminKgInfoData(t, h, token)["logo_url"], "空串清空 logo")
}

func installTestMediaSigner(t *testing.T) {
	t.Helper()
	storage, err := oss.NewStorage(oss.Config{
		Endpoint:        "oss-cn-hangzhou.aliyuncs.com",
		AccessKeyID:     "test-access-key",
		AccessKeySecret: "test-secret",
		BucketName:      "test-bucket",
	})
	require.NoError(t, err)
	response.SetMediaSigner(func(ref string, thumb bool) string {
		signed, signErr := storage.SignRef(ref, time.Hour, thumb)
		if signErr != nil {
			return ref
		}
		return signed
	})
	t.Cleanup(func() { response.SetMediaSigner(nil) })
}
