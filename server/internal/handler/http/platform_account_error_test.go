package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/service"
)

// 账号守卫与口令策略必须在 HTTP 层映射成可区分的状态码与文案：
// 管理端据此展示「不能停用自己 / 最后一个账号」提示，而不是退化成 500。
func TestWriteAccountErrorMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name        string
		err         error
		status      int
		code        int
		messagePart string
	}{
		{"not found", service.ErrAccountNotFound, 404, 40400, "不存在"},
		{"username exists", service.ErrUsernameExists, 409, 40900, "用户名已存在"},
		{"invalid status", service.ErrInvalidAccountStatus, 422, 42200, "状态取值不合法"},
		{"disable self", service.ErrCannotDisableSelf, 409, 40900, "不能停用当前登录账号"},
		{"last account", service.ErrLastAvailableAccount, 409, 40900, "最后一个可用账号"},
		{"invalid old password", service.ErrInvalidOldPassword, 422, 42200, "原密码不正确"},
		{"weak password", fmt.Errorf("%w: too short", service.ErrWeakPassword), 422, 42200, "密码强度不足"},
		{"epoch unavailable", fmt.Errorf("%w: redis down", service.ErrPlatformAuthUnavailable), 503, 50300, "暂不可用"},
		{"empty username", service.ErrInvalidUsername, 422, 42200, "用户名不能为空"},
		{"unexpected", errors.New("boom"), 500, 50000, "平台账号操作失败"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			writeAccountError(c, tc.err)
			require.Equal(t, tc.status, w.Code)
			var body struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			assert.Equal(t, tc.code, body.Code)
			assert.Contains(t, body.Message, tc.messagePart)
		})
	}
}
