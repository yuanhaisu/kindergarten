package rbac

import (
	"testing"

	"github.com/casbin/casbin/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 核对本次新增接送路由是否落在角色通配前缀内（tasks 3.6）。
// 真加载 configs/rbac_model.conf + configs/rbac_policy.csv，不是肉眼 trace。

// enforce 包一层：casbin v2 的 Enforce 返回 (bool, error)。
func enforce(t *testing.T, e *casbin.Enforcer, role, path, method string) bool {
	t.Helper()
	ok, err := e.Enforce(role, path, method)
	require.NoError(t, err, "Enforce 不应报错")
	return ok
}

func newTestEnforcer(t *testing.T) *casbin.Enforcer {
	t.Helper()
	e, err := casbin.NewEnforcer("../../configs/rbac_model.conf", "../../configs/rbac_policy.csv")
	require.NoError(t, err, "必须能加载真实 Casbin model/policy")
	return e
}

// 新增路由必须放行给对应角色（角色通配覆盖的应放行）
func TestRBAC_NewPickupRoutesAllowed(t *testing.T) {
	e := newTestEnforcer(t)

	cases := []struct {
		role   string
		method string
		path   string
	}{
		// 老师：一键通知滞留家长
		{"teacher", "POST", "/api/v1/teacher/pickup/notify-overdue"},
		// 家长：指定今日接走人
		{"parent", "POST", "/api/v1/parent/pickup/designate"},
		// 管理：滞留名单 / 计数 / 一键通知
		{"admin", "GET", "/api/v1/admin/pickup/overdue"},
		{"admin", "GET", "/api/v1/admin/pickup/overdue/count"},
		{"admin", "POST", "/api/v1/admin/pickup/overdue/notify"},
	}
	for _, c := range cases {
		assert.Truef(t, enforce(t, e, c.role, c.path, c.method),
			"%s 应放行 %s %s（角色通配须覆盖）", c.role, c.method, c.path)
	}
}

// 跨角色必须拒绝（行级隔离靠 service，但路由级 RBAC 也要挡住）
func TestRBAC_NewPickupRoutesCrossRoleDenied(t *testing.T) {
	e := newTestEnforcer(t)

	cases := []struct {
		role   string
		method string
		path   string
	}{
		// 家长不得打老师端的滞留通知
		{"parent", "POST", "/api/v1/teacher/pickup/notify-overdue"},
		// 老师不得打家长端的指定
		{"teacher", "POST", "/api/v1/parent/pickup/designate"},
		// 老师不得打管理端的滞留名单
		{"teacher", "GET", "/api/v1/admin/pickup/overdue"},
		// 家长不得打管理端
		{"parent", "POST", "/api/v1/admin/pickup/overdue/notify"},
	}
	for _, c := range cases {
		assert.Falsef(t, enforce(t, e, c.role, c.path, c.method),
			"%s 必须拒绝 %s %s", c.role, c.method, c.path)
	}
}

// 既有接送路由仍被覆盖（防止本次改动意外破坏通配）
func TestRBAC_ExistingPickupRoutesStillAllowed(t *testing.T) {
	e := newTestEnforcer(t)

	cases := []struct {
		role   string
		method string
		path   string
	}{
		{"teacher", "GET", "/api/v1/teacher/pickup/overview"},
		{"teacher", "POST", "/api/v1/teacher/pickup/scan"},
		{"teacher", "POST", "/api/v1/teacher/pickup/manual"},
		{"parent", "GET", "/api/v1/parent/pickup/qrcode"},
		{"parent", "GET", "/api/v1/parent/pickup/records"},
		{"admin", "GET", "/api/v1/admin/pickup/records"},
		{"admin", "PUT", "/api/v1/admin/pickup/authorized-persons/1/audit"},
	}
	for _, c := range cases {
		assert.Truef(t, enforce(t, e, c.role, c.path, c.method),
			"%s 应放行 %s %s", c.role, c.method, c.path)
	}
}
