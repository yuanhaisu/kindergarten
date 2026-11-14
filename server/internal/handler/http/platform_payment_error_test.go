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

// 收款写入的新错误码必须可被管理端识别，金额类错误同时给出结构化金额。
func TestWriteAddPaymentErrorMapsAmountCapsAndTime(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name        string
		err         error
		status      int
		code        int
		messagePart string
		data        map[string]int64
	}{
		{
			"exceeds amount due",
			&service.PaymentExceedsAmountDueError{ExcessCents: 10000, RemainingCents: 40000},
			422, 42200, "100.00",
			map[string]int64{"excess_cents": 10000, "remaining_cents": 40000},
		},
		{
			"refund exceeds paid",
			&service.RefundExceedsPaidAmountError{AvailableCents: 50000, RequestedCents: 50001},
			422, 42200, "500.00",
			map[string]int64{"available_cents": 50000, "requested_cents": 50001},
		},
		{"future received at", service.ErrReceivedAtInFuture, 422, 42200, "到账时间不能晚于当前时间", nil},
		{"invalid idempotency key", service.ErrInvalidIdempotencyKey, 422, 42200, "幂等键不合法", nil},
		{"cancelled", errors.New("ORDER_CANCELLED"), 409, 40900, "订单已取消", nil},
		{"unexpected error", errors.New("UNEXPECTED"), 500, 50000, "登记收款失败", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			writeAddPaymentError(c, tc.err)
			require.Equal(t, tc.status, w.Code)
			var body struct {
				Code    int              `json:"code"`
				Message string           `json:"message"`
				Data    map[string]int64 `json:"data"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			assert.Equal(t, tc.code, body.Code)
			assert.Contains(t, body.Message, tc.messagePart)
			assert.Equal(t, tc.data, body.Data)
		})
	}
}
