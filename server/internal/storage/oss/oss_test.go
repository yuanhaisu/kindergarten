package oss

import (
	"net/url"
	"strings"
	"testing"
	"time"

	alioss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testEndpoint = "https://oss-cn-hangzhou.aliyuncs.com"

func testStorage(t *testing.T) *Storage {
	t.Helper()
	client, err := alioss.New(testEndpoint, "test-key", "test-secret")
	require.NoError(t, err)
	bucket, err := client.Bucket("test-bucket")
	require.NoError(t, err)
	return &Storage{bucket: bucket, bucketName: "test-bucket", endpoint: testEndpoint}
}

func TestNormalizeRef(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"object key", "growth/2026/1.jpg", "growth/2026/1.jpg"},
		{"key with process", "growth/2026/1.jpg?x-oss-process=image%2Fresize%2Cw_200", "growth/2026/1.jpg?x-oss-process=image%2Fresize%2Cw_200"},
		{
			"legacy signed url",
			"https://test-bucket.oss-cn-hangzhou.aliyuncs.com/growth/1.jpg?x-oss-process=image%2Fresize%2Cw_200&Signature=abc&Expires=123",
			"growth/1.jpg?x-oss-process=image%2Fresize%2Cw_200",
		},
		{"external url", "https://thirdwx.qlogo.cn/avatar.jpg", "https://thirdwx.qlogo.cn/avatar.jpg"},
		{"local asset path", "/static/avatar.png", "/static/avatar.png"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, NormalizeRef(c.in))
		})
	}
}

func TestParseRef(t *testing.T) {
	s := testStorage(t)

	key := s.ParseRef("growth/1.jpg?x-oss-process=image%2Fresize%2Cw_200")
	assert.True(t, key.OSS)
	assert.False(t, key.Signed)
	assert.Equal(t, "growth/1.jpg", key.Key)
	assert.Equal(t, "image/resize,w_200", key.Process)

	legacy := s.ParseRef("https://test-bucket.oss-cn-hangzhou.aliyuncs.com/growth/1.jpg?Signature=abc&Expires=123")
	assert.True(t, legacy.OSS)
	assert.True(t, legacy.Signed)
	assert.Equal(t, "growth/1.jpg", legacy.Key)

	external := s.ParseRef("https://thirdwx.qlogo.cn/avatar.jpg")
	assert.False(t, external.OSS)

	local := s.ParseRef("/static/avatar.png")
	assert.False(t, local.OSS, "本地静态资源不应被当成 OSS object key")
}

func TestSignRef(t *testing.T) {
	s := testStorage(t)

	signed, err := s.SignRef("growth/1.jpg", time.Hour, false)
	require.NoError(t, err)
	assert.Contains(t, signed, "Signature=")
	assert.Contains(t, signed, "OSSAccessKeyId=")

	thumb, err := s.SignRef("growth/1.jpg", time.Hour, true)
	require.NoError(t, err)
	u, err := url.Parse(thumb)
	require.NoError(t, err)
	assert.Equal(t, "image/resize,w_200", u.Query().Get("x-oss-process"))

	legacyThumb, err := s.SignRef(
		"https://test-bucket.oss-cn-hangzhou.aliyuncs.com/growth/1.mp4?x-oss-process=video%2Fsnapshot%2Ct_1000%2Cw_200%2Cm_fast",
		time.Hour, true,
	)
	require.NoError(t, err)
	u, err = url.Parse(legacyThumb)
	require.NoError(t, err)
	assert.Equal(t, "video/snapshot,t_1000,w_200,m_fast", u.Query().Get("x-oss-process"))

	external := "https://thirdwx.qlogo.cn/avatar.jpg"
	unchanged, err := s.SignRef(external, time.Hour, false)
	require.NoError(t, err)
	assert.Equal(t, external, unchanged)

	again, err := s.SignRef(signed, time.Hour, false)
	require.NoError(t, err)
	assert.Equal(t, signed, again, "已签名 URL 不应重复签名")
}

func TestThumbRef(t *testing.T) {
	assert.Equal(t, "a.jpg?x-oss-process=image%2Fresize%2Cw_200", ThumbRef("a.jpg", "image"))
	assert.True(t, strings.HasPrefix(ThumbRef("a.mp4", "video"), "a.mp4?x-oss-process="))
	assert.Empty(t, ThumbRef("a.pdf", "document"))
}

func TestConfigured(t *testing.T) {
	assert.False(t, (&Storage{}).Configured())
	assert.True(t, testStorage(t).Configured())
}
