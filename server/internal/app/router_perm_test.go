package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	handler "kindergarten-server/internal/handler/http"
)

// 登记表必须覆盖全部已注册管理端路由，否则 NewRouter 返回错误（wire-admin-permissions
// 任务 2.3 / TC-NEW-PERM-0007 正例）。用空 Handler 构建：注册只取方法值，不调用服务，
// resolver 传 nil 也不影响注册与自检。
func TestAdminPermissionRegistryCoversAllRoutes(t *testing.T) {
	engine, err := NewRouter(&AppHandler{inner: &handler.Handler{}}, nil, JWTSecret("test-secret"), nil)
	require.NoError(t, err, "登记表存在缺口时 NewRouter 必须报错并列出缺口路由")
	assert.NotNil(t, engine)
}
