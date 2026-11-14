package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"kindergarten-server/internal/auth"
)

type stubKgStatus struct {
	suspended bool
	err       error
}

func (s stubKgStatus) Suspended(context.Context, uint64) (bool, error) {
	return s.suspended, s.err
}

func newStatusGuardedEngine(checker interface {
	Suspended(context.Context, uint64) (bool, error)
}, kgID uint64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("")
	g.Use(func(c *gin.Context) {
		c.Set(CtxClaimsKey, &auth.AccessClaims{UserID: "1", KindergartenID: kgID})
		c.Next()
	})
	g.Use(RequireActiveKindergarten(checker))
	g.GET("/protected", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	return r
}

func TestRequireActiveKindergarten(t *testing.T) {
	cases := []struct {
		name     string
		checker  stubKgStatus
		kgID     uint64
		wantCode int
	}{
		{"在营园所放行", stubKgStatus{}, 42, http.StatusNoContent},
		{"停用园所被拒", stubKgStatus{suspended: true}, 42, http.StatusForbidden},
		{"无园所凭证不检查", stubKgStatus{suspended: true}, 0, http.StatusNoContent},
		{"查询失败返回 503", stubKgStatus{err: context.DeadlineExceeded}, 42, http.StatusServiceUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			engine := newStatusGuardedEngine(tc.checker, tc.kgID)
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			assert.Equal(t, tc.wantCode, w.Code)
		})
	}
}
