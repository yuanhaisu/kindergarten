package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"kindergarten-server/pkg/requestid"
)

// 审计等下游从 request context 取链路标识，不依赖每个 service 调用点手工透传：
// 中间件必须把 X-Request-ID 同时写进 gin context 与 request context。
func TestRequestIDPropagatesIntoRequestContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/probe", func(c *gin.Context) {
		c.String(http.StatusOK, requestid.From(c.Request.Context()))
	})

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("X-Request-ID", "req-mw-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Body.String() != "req-mw-1" {
		t.Fatalf("request context request_id = %q, want req-mw-1", w.Body.String())
	}
	if got := w.Header().Get("X-Request-ID"); got != "req-mw-1" {
		t.Fatalf("response X-Request-ID = %q, want req-mw-1", got)
	}
}

// 未携带请求头时必须生成一个非空标识，否则审计记录无法关联链路日志。
func TestRequestIDGeneratesWhenHeaderMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/probe", func(c *gin.Context) {
		c.String(http.StatusOK, requestid.From(c.Request.Context()))
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/probe", nil))

	if w.Body.String() == "" {
		t.Fatal("request context request_id must not be empty when header is absent")
	}
	if w.Header().Get("X-Request-ID") != w.Body.String() {
		t.Fatalf("response header and context id disagree: %q vs %q", w.Header().Get("X-Request-ID"), w.Body.String())
	}
}
