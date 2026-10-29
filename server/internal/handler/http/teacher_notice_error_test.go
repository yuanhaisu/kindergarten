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

// 契约 api-teacher/04-notices.md 要求通知的参数/状态错误返回 400
// （定时时间非法、发布不完整、附件超限、删除非草稿、撤回非已发布），
// 而成长、任务等其它老师端契约仍写 422——所以必须走通知专属出口，
// 共用的 teacherPortalErr 不能跟着改。
func TestTeacherNoticeErrorMapsInvalidParamsTo400(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name   string
		err    error
		status int
		code   int
	}{
		{"invalid params", service.ErrTeacherInvalidParams, 400, 40014},
		{"notice state", service.ErrTeacherNoticeState, 400, 40012},
		{"not found", service.ErrTeacherNotFound, 404, 40400},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			teacherNoticeErr(c, tc.err, "fallback")
			require.Equal(t, tc.status, w.Code)
			var body struct {
				Code int `json:"code"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			assert.Equal(t, tc.code, body.Code)
		})
	}
}

// 共用出口保持 422：任务、成长等模块的契约没有改，不能被通知的 400 带崩。
func TestTeacherPortalErrKeeps422ForOtherModules(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	teacherPortalErr(c, service.ErrTeacherInvalidParams, "fallback")
	require.Equal(t, 422, w.Code)
	var body struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 42200, body.Code)
}
