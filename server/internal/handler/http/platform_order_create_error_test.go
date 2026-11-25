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

// #202：建单的到期日/零应收校验必须是可识别的 422，不能落成 500
// 或「创建订单失败」这类无法行动的通用错误。
func TestWriteCreateOrderErrorMapsDueAndAmount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name        string
		err         error
		status      int
		code        int
		messagePart string
	}{
		{"past due", service.ErrPayDueAtInPast, 422, 42200, "不能早于今天"},
		{"zero amount", service.ErrOrderAmountNotPositive, 422, 42200, "应收金额必须大于 0"},
		{"invalid format", service.ErrInvalidPayDueAt, 422, 42200, "格式不合法"},
		{"not found", errors.New("PLAN_NOT_FOUND"), 404, 40400, "套餐不存在"},
		{"unexpected", errors.New("UNEXPECTED"), 500, 50000, "创建订单失败"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			writeCreateOrderError(c, tc.err)
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

// #202 决策 6：0 元套餐在创建与更新两个入口都必须返回明确的 422。
func TestWritePlanErrorMapsZeroPrice(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writers := []struct {
		name string
		fn   func(*gin.Context, error)
	}{
		{"create", writePlanCreateError},
		{"update", writePlanUpdateError},
	}
	for _, w := range writers {
		t.Run(w.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			w.fn(c, service.ErrPlanPriceNotPositive)
			require.Equal(t, 422, rec.Code)
			var body struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			assert.Equal(t, 42200, body.Code)
			assert.Contains(t, body.Message, "价格必须大于 0")
			assert.NotContains(t, body.Message, "失败", "不能退化成通用错误")
		})
	}
}
