package handler

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/service"
)

// 家长班级相册详情的唯一错误出口：不存在/草稿/下架/跨班/留窗过期都是 404，
// 不能暴露成 403 或 500；未绑定孩子才是 403。
func TestWriteParentClassAlbumErrorMapsSentinels(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name   string
		err    error
		status int
		code   int
	}{
		{"not found / expired", service.ErrParentNotFound, 404, 40400},
		{"not bound", service.ErrParentForbidden, 403, 40300},
		{"invalid scope", service.ErrParentInvalidParams, 422, 42200},
		{"unknown", assert.AnError, 500, 50000},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			writeParentClassAlbumError(c, tc.err)
			require.Equal(t, tc.status, w.Code)
			var body struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			assert.Equal(t, tc.code, body.Code)
			assert.NotEmpty(t, body.Message)
		})
	}
}
