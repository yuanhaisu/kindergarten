package handler

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/service"
)

// 档位保存的错误映射是管理端看到「为什么存不下」的唯一出口：
// 每种覆盖校验失败都必须有具体文案，不能落到 fallback 500。
func TestWritePlanTierErrorMapsCoverageSentinels(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name        string
		err         error
		status      int
		code        int
		wantMessage string
	}{
		{"empty", service.ErrTierCoverageEmpty, 422, 42200, "至少需要配置一个档位"},
		{"not from zero", service.ErrTierCoverageNotFromZero, 422, 42200, "必须从 0 人开始"},
		{"gap", service.ErrTierCoverageGap, 422, 42200, "区间不连续"},
		{"overlap", service.ErrTierCoverageOverlap, 422, 42200, "区间重叠"},
		{"bounded top", service.ErrTierCoverageBoundedTop, 422, 42200, "最高档必须无上限"},
		{"unbounded not last", service.ErrTierCoverageUnboundedNotLast, 422, 42200, "必须排在最后"},
		{"unexpected error", errors.New("UNEXPECTED"), 500, 50000, "保存档位失败"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			writePlanTierError(c, tc.err)
			require.Equal(t, tc.status, w.Code)
			var body struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			assert.Equal(t, tc.code, body.Code)
			assert.Contains(t, body.Message, tc.wantMessage)
		})
	}
}

// 历史档位配置落不到档位时，建单必须返回可识别的 422 校验错误。
func TestWriteCreateOrderErrorMapsUnresolvedTier(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	writeCreateOrderError(c, service.ErrTierCoverageUnresolved)
	require.Equal(t, 422, w.Code)
	var body struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 42200, body.Code)
	assert.Contains(t, body.Message, "档位")
	assert.NotContains(t, body.Message, "创建订单失败", "不能退化成通用错误")
}
