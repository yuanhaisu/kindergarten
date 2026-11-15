package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func corsTestRouter(origins []string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS(origins))
	r.Any("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func TestCORSAllowsConfiguredOrigin(t *testing.T) {
	r := corsTestRouter([]string{"https://admin.example.com"})
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://admin.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, "https://admin.example.com", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "Origin", w.Header().Get("Vary"))
}

func TestCORSRejectsUnknownOrigin(t *testing.T) {
	r := corsTestRouter([]string{"https://admin.example.com"})
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	// 即使不放行也要 Vary，避免中间缓存把白名单来源的响应复用到其他来源。
	assert.Equal(t, "Origin", w.Header().Get("Vary"))
}

// 空配置 = 默认拒绝：任何来源都拿不到放行头。
func TestCORSWithNoOriginsRejectsAnyOrigin(t *testing.T) {
	r := corsTestRouter(nil)
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://admin.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "Origin", w.Header().Get("Vary"))
}

// 显式配置 "*" 时才允许任意来源（release 模式下配置层会告警）。
func TestCORSWildcardAllowsAnyOrigin(t *testing.T) {
	r := corsTestRouter([]string{"*"})
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://anything.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	// 通配符分支同样要 Vary：响应与来源无关，但缓存仍须按 Origin 分桶，
	// 否则将来把 "*" 改回白名单时，中间缓存可能把旧响应复用给其他来源。
	assert.Equal(t, "Origin", w.Header().Get("Vary"))
}

// 配置项两侧的空格不能影响精确匹配；回显值必须是去空格后的来源。
func TestCORSTrimsWhitespaceAroundOrigins(t *testing.T) {
	r := corsTestRouter([]string{" https://admin.example.com ", "", "https://ops.example.com\t"})

	for _, origin := range []string{"https://admin.example.com", "https://ops.example.com"} {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, origin, w.Header().Get("Access-Control-Allow-Origin"))
	}
}

// 小程序请求不带 Origin，行为不受 CORS 配置影响。
func TestCORSRequestWithoutOriginIsUntouched(t *testing.T) {
	r := corsTestRouter([]string{"https://admin.example.com"})
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSHandlesPreflight(t *testing.T) {
	r := corsTestRouter([]string{"https://admin.example.com"})
	req := httptest.NewRequest(http.MethodOptions, "/ping", nil)
	req.Header.Set("Origin", "https://admin.example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "https://admin.example.com", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "terminal_from")
}
