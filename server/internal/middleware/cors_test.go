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
