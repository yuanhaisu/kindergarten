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

// 零收款确认缺理由必须是可识别的 422（契约 CREDIT_REASON_REQUIRED），
// 不能退化成 500；其余确认失败沿用既有映射。
func TestWriteConfirmOrderErrorMapsCreditReason(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name        string
		err         error
		status      int
		code        int
		messagePart string
	}{
		{"missing reason", service.ErrCreditReasonRequired, 422, 42200, "理由"},
		{"reason too long", service.ErrCreditReasonTooLong, 422, 42200, "255"},
		{"not found", errors.New("ORDER_NOT_FOUND"), 404, 40400, "订单不存在"},
		{"already confirmed", errors.New("ORDER_ALREADY_CONFIRMED"), 409, 40900, "订单状态不允许确认"},
		{"cancelled", errors.New("ORDER_CANCELLED"), 409, 40900, "订单状态不允许确认"},
		{"unexpected error", errors.New("UNEXPECTED"), 500, 50000, "确认订单失败"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			writeConfirmOrderError(c, tc.err)
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
