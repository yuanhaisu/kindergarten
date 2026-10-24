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

// writeTaskError 是管理端任务错误的**唯一**映射出口（CreateTask / UpdateTask /
// DeleteTask / 活动三个端点 / 公开列表都走它）。
//
// C-2 的独立审查正是在这里发现问题：DeleteTask 曾经自带一份映射表，
// 新增的「活动任务不能单删」落到 default → 500，而契约要求 422。
// 这条用例把「新哨兵必须在唯一出口里有位置」钉成可回归的断言。
func TestWriteTaskErrorMapsTaskSentinels(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name   string
		err    error
		status int
		code   int
	}{
		{"title too long", service.ErrTaskTitleTooLong, 422, 42200},
		{"activity task single delete", service.ErrTaskActivityDeleteForbidden, 422, 42200},
		{"activity task single edit", service.ErrTaskActivityEditForbidden, 422, 42200},
		{"activity field forbidden", service.ErrTaskActivityFieldForbidden, 422, 42200},
		{"activity duplicate class", service.ErrTaskActivityDuplicateClass, 422, 42200},
		{"activity classes invalid", service.ErrTaskActivityClassesInvalid, 422, 42200},
		{"activity too many classes", service.ErrTaskActivityTooManyClasses, 422, 42200},
		{"activity id forbidden", service.ErrTaskActivityIDForbidden, 422, 42200},
		{"activity not found", service.ErrTaskActivityNotFound, 404, 40400},
		{"structural lock", service.ErrTaskStructuralLocked, 422, 42200},
		{"task busy", service.ErrTaskBusy, 503, 50300},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			writeTaskError(c, tc.err, "fallback")
			require.Equal(t, tc.status, w.Code)
			var body struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			assert.Equal(t, tc.code, body.Code)
			assert.NotEmpty(t, body.Message)
			assert.NotEqual(t, "fallback", body.Message, "哨兵必须映射到具体文案，不能落到 fallback")
		})
	}
}
