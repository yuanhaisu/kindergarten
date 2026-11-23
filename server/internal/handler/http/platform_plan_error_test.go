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

// #209 标准项：套餐创建/更新的错误映射去重后必须逐条保持原行为。
// 表格同时锁定共享分支（价格/模块/参数前缀）与入口差异：
// 只有创建入口把 PLAN_* 视为参数错误，且 PLAN_CODE_EXISTS 在更新入口
// 仍是通用 500（服务层不会产生该错误，保留原状）。
func TestWritePlanErrorSharedAndEntrySpecificMappings(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name       string
		err        error
		createWant httpExpectation
		updateWant httpExpectation
	}{
		{
			name:       "code exists",
			err:        errors.New("PLAN_CODE_EXISTS"),
			createWant: httpExpectation{409, 40900, "套餐编码已存在"},
			updateWant: httpExpectation{500, 50000, "更新套餐失败"},
		},
		{
			name:       "not found",
			err:        errors.New("PLAN_NOT_FOUND"),
			createWant: httpExpectation{422, 42200, "套餐参数不合法"},
			updateWant: httpExpectation{404, 40400, "套餐不存在"},
		},
		{
			name:       "zero price",
			err:        service.ErrPlanPriceNotPositive,
			createWant: httpExpectation{422, 42200, "套餐价格必须大于 0"},
			updateWant: httpExpectation{422, 42200, "套餐价格必须大于 0"},
		},
		{
			name:       "unknown module",
			err:        errors.New("UNKNOWN_MODULE: growth"),
			createWant: httpExpectation{422, 42200, "套餐参数不合法"},
			updateWant: httpExpectation{422, 42200, "套餐参数不合法"},
		},
		{
			name:       "invalid prefix",
			err:        errors.New("INVALID_TIER"),
			createWant: httpExpectation{422, 42200, "套餐参数不合法"},
			updateWant: httpExpectation{422, 42200, "套餐参数不合法"},
		},
		{
			name: "plan prefix only create",
			// PLAN_NAME_REQUIRED 由 validatePlanRequest 在两个入口都会返回；
			// 原实现里只有创建入口把它收敛成 422，更新入口落到通用 500。
			err:        errors.New("PLAN_NAME_REQUIRED"),
			createWant: httpExpectation{422, 42200, "套餐参数不合法"},
			updateWant: httpExpectation{500, 50000, "更新套餐失败"},
		},
		{
			name:       "unexpected",
			err:        errors.New("BOOM"),
			createWant: httpExpectation{500, 50000, "创建套餐失败"},
			updateWant: httpExpectation{500, 50000, "更新套餐失败"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertMapping(t, writePlanCreateError, tc.err, tc.createWant)
			assertMapping(t, writePlanUpdateError, tc.err, tc.updateWant)
		})
	}
}

type httpExpectation struct {
	status  int
	code    int
	message string
}

func assertMapping(t *testing.T, writer func(*gin.Context, error), err error, want httpExpectation) {
	t.Helper()
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	writer(c, err)
	require.Equal(t, want.status, rec.Code)
	var body struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, want.code, body.Code)
	assert.Contains(t, body.Message, want.message)
}
