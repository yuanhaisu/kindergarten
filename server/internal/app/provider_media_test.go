package app

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/config"
	"kindergarten-server/pkg/response"
)

func TestProvideOSSStorageInstallsMediaSigner(t *testing.T) {
	cfg := &config.Config{}
	cfg.OSS.Endpoint = "https://oss-cn-hangzhou.aliyuncs.com"
	cfg.OSS.AccessKeyID = "test-key"
	cfg.OSS.AccessKeySecret = "test-secret"
	cfg.OSS.BucketName = "test-bucket"

	_, err := ProvideOSSStorage(cfg)
	require.NoError(t, err)
	defer response.SetMediaSigner(nil)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	response.OK(c, map[string]any{"file_url": "growth/1.jpg", "logo_url": "uploads/logo.png"})

	assert.Contains(t, w.Body.String(), "Signature=")
	assert.Contains(t, w.Body.String(), "OSSAccessKeyId=")
	assert.Contains(t, w.Body.String(), "uploads", "园所 logo（#178）必须在媒体白名单内，否则管理端拿到的是裸 key")
	assert.NotContains(t, w.Body.String(), `"logo_url":"uploads/logo.png"`, "logo_url 不能以未签名形式下发")
}
