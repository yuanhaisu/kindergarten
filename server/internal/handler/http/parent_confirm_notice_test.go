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

// 家长确认通知的唯一错误出口。草稿/定时/已撤回在服务层返回 ErrParentInvalidParams，
// 这里必须映射成 400：落到 default 的 500 会让家长端把「还没发出来」当成服务端故障。
func TestWriteParentConfirmNoticeErrorMapsStatusErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name   string
		err    error
		status int
		code   int
	}{
		{"not found", service.ErrParentNotFound, 404, 40400},
		{"not published", service.ErrParentInvalidParams, 400, 40013},
		{"unknown", assert.AnError, 500, 50000},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			writeParentConfirmNoticeError(c, tc.err)
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
