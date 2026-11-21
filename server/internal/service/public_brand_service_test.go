package service_test

import (
	"context"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/internal/service"
	"kindergarten-server/internal/storage/oss"
	"kindergarten-server/internal/testutil/authharness"
)

// 公开品牌服务（#170）：只返回名称/logo/主题色/公开码，logo 走短期签名。
//
// 测试库由 authharness 派生并跑真实迁移；签名用真实 *oss.Storage（伪造凭据，
// SignURL 是本地 HMAC 计算，不发网络请求），因此验证的是实际签名逻辑而不是替身。

func newPublicBrandKindergarten(t *testing.T, h *authharness.Harness, name, code, logo, color string) *domain.Kindergarten {
	t.Helper()
	kg := &domain.Kindergarten{
		Name:       name,
		Status:     domain.KindergartenStatusActive,
		PublicCode: code,
		LogoURL:    logo,
		BrandColor: color,
	}
	require.NoError(t, h.DB.Create(kg).Error, "创建园所失败")
	t.Cleanup(func() {
		h.HardDelete(t, &domain.Kindergarten{}, "id = ?", kg.ID)
	})
	return kg
}

func newTestLogoSigner(t *testing.T) service.LogoSigner {
	t.Helper()
	storage, err := oss.NewStorage(oss.Config{
		Endpoint:        "oss-cn-hangzhou.aliyuncs.com",
		AccessKeyID:     "test-access-key",
		AccessKeySecret: "test-secret",
		BucketName:      "test-bucket",
	})
	require.NoError(t, err)
	require.True(t, storage.Configured(), "伪造凭据应能构造可签名的 Storage")
	return service.LogoSignerFunc(func(ref string) (string, error) {
		return storage.SignRef(ref, service.PublicBrandLogoTTL, false)
	})
}

func TestPublicBrandReturnsSignedLogoWithShortTTL(t *testing.T) {
	h := authharness.New(t)
	svc := service.NewPublicBrandService(repository.NewKindergartenRepo(h.DB), newTestLogoSigner(t))

	kg := newPublicBrandKindergarten(t, h, "170-品牌园", "brandabc12", "uploads/logo/kindergarten.png", "#C2410C")

	brand, err := svc.GetByPublicCode(context.Background(), kg.PublicCode)
	require.NoError(t, err)
	require.NotNil(t, brand, "未删除园所必须能按公开码取到品牌")

	assert.Equal(t, kg.PublicCode, brand.PublicCode)
	assert.Equal(t, "170-品牌园", brand.Name)
	assert.Equal(t, "#C2410C", brand.BrandColor, "通过对比度的自定义主题色应原样返回")

	require.NotEqual(t, "uploads/logo/kindergarten.png", brand.LogoURL,
		"私有 OSS 对象不能把 object key 直接下发")
	signed, err := url.Parse(brand.LogoURL)
	require.NoError(t, err, "签名后的 logo 必须是合法 URL")
	query := signed.Query()
	assert.NotEmpty(t, query.Get("Signature"), "logo URL 必须带 OSS 签名")
	assert.Equal(t, "test-access-key", query.Get("OSSAccessKeyId"))

	expires, err := strconv.ParseInt(query.Get("Expires"), 10, 64)
	require.NoError(t, err, "签名 URL 必须带过期时间")
	remaining := time.Until(time.Unix(expires, 0))
	assert.LessOrEqual(t, remaining, service.PublicBrandLogoTTL+2*time.Second,
		"签名 TTL 必须是短时效（%s），不能沿用响应层 1 小时全局签名", service.PublicBrandLogoTTL)
	assert.Greater(t, remaining, time.Duration(0), "签名不能一生成就过期")
}

func TestPublicBrandFallsBackForInvalidBrandColor(t *testing.T) {
	h := authharness.New(t)
	svc := service.NewPublicBrandService(repository.NewKindergartenRepo(h.DB), newTestLogoSigner(t))

	kg := newPublicBrandKindergarten(t, h, "170-非法主题色园", "brandcolor2", "", "#FFEB3B")

	brand, err := svc.GetByPublicCode(context.Background(), kg.PublicCode)
	require.NoError(t, err)
	require.NotNil(t, brand)
	assert.Equal(t, service.DefaultBrandColor, brand.BrandColor, "对比度不足的主题色必须回落默认")
	assert.Equal(t, "", brand.LogoURL, "无 logo 时返回空串")
}

func TestPublicBrandDoesNotReturnSoftDeletedKindergarten(t *testing.T) {
	h := authharness.New(t)
	svc := service.NewPublicBrandService(repository.NewKindergartenRepo(h.DB), newTestLogoSigner(t))

	kg := newPublicBrandKindergarten(t, h, "170-软删园", "brandsoftde", "", "")
	require.NoError(t, h.DB.Delete(&domain.Kindergarten{}, kg.ID).Error, "软删园所")

	brand, err := svc.GetByPublicCode(context.Background(), kg.PublicCode)
	require.NoError(t, err)
	assert.Nil(t, brand, "软删园所的品牌不得返回")
}

func TestPublicBrandReturnsNilForUnknownOrEmptyCode(t *testing.T) {
	h := authharness.New(t)
	svc := service.NewPublicBrandService(repository.NewKindergartenRepo(h.DB), newTestLogoSigner(t))

	for _, code := range []string{"", "   ", "notexists"} {
		brand, err := svc.GetByPublicCode(context.Background(), code)
		require.NoError(t, err, "code=%q", code)
		assert.Nil(t, brand, "code=%q 不应命中任何园所", code)
	}
}
