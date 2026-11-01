package app

import (
	"strings"
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

// AQR-S1b：/admin/diet 本 slice 只注册供餐配置两条路由。
// 路由漏登记权限码会让 NewRouter 启动失败（fail-closed），因此本用例同时覆盖 1.6 与 2.3。
func TestRouterPerm_DietConfigRoutesRegistered(t *testing.T) {
	engine, err := NewRouter(&AppHandler{inner: &handler.Handler{}}, nil, JWTSecret("test-secret"), nil)
	require.NoError(t, err, "diet 配置路由漏登记权限码时 NewRouter 必须启动失败")

	var got []string
	for _, rt := range engine.Routes() {
		if strings.HasPrefix(rt.Path, "/api/v1/admin/diet") {
			got = append(got, rt.Method+" "+rt.Path)
		}
	}
	assert.ElementsMatch(t, []string{
		"GET /api/v1/admin/diet/config",
		"PUT /api/v1/admin/diet/config",
	}, got, "本 slice 只注册配置两条路由，其余 diet 路由由后续 slice 各自登记")
}
