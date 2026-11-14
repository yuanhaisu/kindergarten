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

// 取消订单被未退款收款阻断时，契约里必须同时给出可识别状态码与待退金额，
// 管理端据此展示「先退款」提示，而不是退化成 500 或只给泛化文案。
func TestWriteCancelOrderErrorMapsUnrefundedAmount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name        string
		err         error
		status      int
		code        int
		messagePart string
		amount      int64
	}{
		{"unrefunded", &service.UnrefundedPaymentError{AmountCents: 60000}, 409, 40900, "600.00", 60000},
		{"not found", errors.New("ORDER_NOT_FOUND"), 404, 40400, "订单不存在", 0},
		{"already cancelled", errors.New("ORDER_ALREADY_CANCELLED"), 409, 40900, "订单已取消", 0},
		{"unexpected error", errors.New("UNEXPECTED"), 500, 50000, "取消订单失败", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			writeCancelOrderError(c, tc.err)
			require.Equal(t, tc.status, w.Code)
			var body struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
				Data    struct {
					UnrefundedAmountCents int64 `json:"unrefunded_amount_cents"`
				} `json:"data"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			assert.Equal(t, tc.code, body.Code)
			assert.Contains(t, body.Message, tc.messagePart)
			assert.Equal(t, tc.amount, body.Data.UnrefundedAmountCents)
		})
	}
}
