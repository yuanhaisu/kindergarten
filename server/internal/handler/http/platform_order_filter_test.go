package handler

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 订单检索的时间范围使用 YYYY-MM-DD；end_date 含当日，转换为右开区间。
func TestParseOrderDateRange(t *testing.T) {
	from, to, err := parseOrderDateRange("2026-11-01", "2026-11-18")
	require.NoError(t, err)
	require.NotNil(t, from)
	require.NotNil(t, to)
	assert.Equal(t, time.Date(2026, 11, 1, 0, 0, 0, 0, time.Local), *from)
	assert.Equal(t, time.Date(2026, 11, 19, 0, 0, 0, 0, time.Local), *to, "end_date 必须含当日")

	from, to, err = parseOrderDateRange("", "")
	require.NoError(t, err)
	assert.Nil(t, from)
	assert.Nil(t, to)

	// 同一天是合法区间。
	from, to, err = parseOrderDateRange("2026-11-18", "2026-11-18")
	require.NoError(t, err)
	assert.True(t, to.After(*from))

	_, _, err = parseOrderDateRange("2026/11/01", "")
	assert.Error(t, err)
	_, _, err = parseOrderDateRange("", "2026-11-32")
	assert.Error(t, err)
	_, _, err = parseOrderDateRange("2026-11-10", "2026-11-01")
	assert.Error(t, err)
}

// 非法筛选参数必须在触达 service 之前以 422 拒绝。
func TestPlatformListOrders_RejectsInvalidFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name  string
		query string
	}{
		{"invalid start_date", "?start_date=2026%2F11%2F01"},
		{"invalid end_date", "?end_date=18-11-2026"},
		{"end before start", "?start_date=2026-11-10&end_date=2026-11-01"},
		{"invalid outstanding_only", "?outstanding_only=yes"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/api/v1/platform/orders"+tc.query, nil)
			h := &Handler{}
			h.PlatformListOrders(c)
			assert.Equal(t, 422, w.Code)
		})
	}
}
