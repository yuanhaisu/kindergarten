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

// AQR-S1b/S3/S4/S6：/admin/diet 当前注册 2 条配置 + 8 条食物 + 7 条菜谱 + 8 条周食谱/模板。
// 路由漏登记权限码会让 NewRouter 启动失败（fail-closed），因此本用例同时覆盖
// 权限登记与精确路由集合；后续 slice 新增 diet 路由时必须同步更新此断言。
func TestRouterPerm_DietConfigRoutesRegistered(t *testing.T) {
	engine, err := NewRouter(&AppHandler{inner: &handler.Handler{}}, nil, JWTSecret("test-secret"), nil)
	require.NoError(t, err, "diet 路由漏登记权限码时 NewRouter 必须启动失败")

	var got []string
	for _, rt := range engine.Routes() {
		if strings.HasPrefix(rt.Path, "/api/v1/admin/diet") {
			got = append(got, rt.Method+" "+rt.Path)
		}
	}
	assert.ElementsMatch(t, []string{
		"GET /api/v1/admin/diet/config",
		"PUT /api/v1/admin/diet/config",

		"GET /api/v1/admin/diet/food-categories",
		"GET /api/v1/admin/diet/foods",
		"POST /api/v1/admin/diet/foods",
		"POST /api/v1/admin/diet/foods/batch-delete",
		"GET /api/v1/admin/diet/foods/:id",
		"PUT /api/v1/admin/diet/foods/:id",
		"DELETE /api/v1/admin/diet/foods/:id",
		"GET /api/v1/admin/diet/foods/:id/replace-candidates",

		"GET /api/v1/admin/diet/recipes",
		"POST /api/v1/admin/diet/recipes",
		"GET /api/v1/admin/diet/recipes/suggest",
		"POST /api/v1/admin/diet/recipes/merge",
		"GET /api/v1/admin/diet/recipes/:id",
		"PUT /api/v1/admin/diet/recipes/:id",
		"DELETE /api/v1/admin/diet/recipes/:id",

		"GET /api/v1/admin/diet/plans",
		"POST /api/v1/admin/diet/plans",
		"GET /api/v1/admin/diet/plans/:id",
		"PUT /api/v1/admin/diet/plans/:id",
		"DELETE /api/v1/admin/diet/plans/:id",
		"POST /api/v1/admin/diet/plans/:id/clear",
		"GET /api/v1/admin/diet/templates",
		"GET /api/v1/admin/diet/templates/:id",
	}, got, "diet 组当前应恰好 25 条路由：2 配置 + 8 食物 + 7 菜谱 + 8 周食谱/模板")
}
