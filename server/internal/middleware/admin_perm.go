package middleware

import (
	"context"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"kindergarten-server/pkg/log"
	"kindergarten-server/pkg/response"
)

// PermissionResolver 解析管理端账号权限码（由 service.AdminAuthzService 实现）。
// 接口声明在消费方，便于中间件单测注入桩。
type PermissionResolver interface {
	ResolveAdminCodes(ctx context.Context, kindergartenID, userID uint64) ([]string, bool, error)
}

// adminRoutePermission 管理端「路由 → 权限码」登记表。
// 权威定义见 openspec/changes/wire-admin-permissions/design.md D7；
// 未登记的已注册管理端路由在运行期一律拒绝，且启动自检会使其启动失败（design D1）。
//
// 约定：查看类用查看码；写操作用对应 manage/audit/export 码；
// 无独立码的资源组沿用其查看码（leaves→attendance:view、health→dashboard:view，
// 属已登记的近似处理，待权限码补全后收紧）。
var adminRoutePermission = map[string]string{
	// dashboard —— 看板全部只读
	"GET /api/v1/admin/dashboard/overview":         "dashboard:view",
	"GET /api/v1/admin/dashboard/attendance-trend": "dashboard:view",
	"GET /api/v1/admin/dashboard/class-ranking":    "dashboard:view",
	"GET /api/v1/admin/dashboard/growth-stats":     "dashboard:view",

	// classes —— 班级与学籍共码
	"GET /api/v1/admin/classes":        "student:view",
	"POST /api/v1/admin/classes":       "student:manage",
	"GET /api/v1/admin/classes/:id":    "student:view",
	"PUT /api/v1/admin/classes/:id":    "student:manage",
	"DELETE /api/v1/admin/classes/:id": "student:manage",
	"POST /api/v1/admin/classes/:id/adjust": "student:manage",

	// students
	"GET /api/v1/admin/students":             "student:view",
	"POST /api/v1/admin/students":            "student:manage",
	"POST /api/v1/admin/students/import":     "student:import",
	"GET /api/v1/admin/students/export":      "student:export",
	"GET /api/v1/admin/students/:id":         "student:view",
	"PUT /api/v1/admin/students/:id":         "student:manage",
	"DELETE /api/v1/admin/students/:id":      "student:manage",
	"PUT /api/v1/admin/students/:id/status":  "student:manage",
	// 人脸底库（照片点名）：查看用学生查看码，录入/重试/移除用学生管理码
	"GET /api/v1/admin/students/:id/faces":             "student:view",
	"POST /api/v1/admin/students/:id/faces":            "student:manage",
	"POST /api/v1/admin/students/:id/faces/retry":      "student:manage",
	"DELETE /api/v1/admin/students/:id/faces/:faceId":  "student:manage",

	// teachers
	"GET /api/v1/admin/teachers":                        "teacher:view",
	"POST /api/v1/admin/teachers":                       "teacher:manage",
	"GET /api/v1/admin/teachers/:id":                    "teacher:view",
	"PUT /api/v1/admin/teachers/:id":                    "teacher:manage",
	"DELETE /api/v1/admin/teachers/:id":                 "teacher:manage",
	"PUT /api/v1/admin/teachers/:id/status":             "teacher:manage",
	"GET /api/v1/admin/teachers/:id/invitation":         "teacher:view",
	"POST /api/v1/admin/teachers/:id/invitation":        "teacher:manage",
	"GET /api/v1/admin/teachers/:id/invitation/qrcode":  "teacher:view",

	// parents
	"GET /api/v1/admin/parents":             "parent:view",
	"POST /api/v1/admin/parents":            "parent:manage",
	"GET /api/v1/admin/parents/:id":         "parent:view",
	"PUT /api/v1/admin/parents/:id":         "parent:manage",
	"PUT /api/v1/admin/parents/:id/status":  "parent:manage",

	// settings / kindergarten —— 园所配置
	"GET /api/v1/admin/settings":                   "system:kindergarten",
	"PUT /api/v1/admin/settings":                   "system:kindergarten",
	"GET /api/v1/admin/settings/academic-years":    "system:kindergarten",
	"POST /api/v1/admin/settings/academic-years":   "system:kindergarten",
	"GET /api/v1/admin/kindergarten/info":          "system:kindergarten",
	"PUT /api/v1/admin/kindergarten/info":          "system:kindergarten",

	// admins
	"GET /api/v1/admin/admins":                     "admin:view",
	"POST /api/v1/admin/admins":                    "admin:manage",
	"PUT /api/v1/admin/admins/:id":                 "admin:manage",
	"PUT /api/v1/admin/admins/:id/reset-password":  "admin:manage",
	"DELETE /api/v1/admin/admins/:id":              "admin:manage",

	// roles / permissions
	"GET /api/v1/admin/roles":        "admin:role",
	"POST /api/v1/admin/roles":       "admin:role",
	"PUT /api/v1/admin/roles/:id":    "admin:role",
	"DELETE /api/v1/admin/roles/:id": "admin:role",
	"GET /api/v1/admin/permissions":  "admin:role",

	// attendance
	"GET /api/v1/admin/attendance/overview": "attendance:view",
	"GET /api/v1/admin/attendance/records":  "attendance:view",
	"GET /api/v1/admin/attendance/report":   "attendance:export",

	// leaves —— 请假无独立码，与出勤共码（proposal 阻塞项）
	"GET /api/v1/admin/leaves":           "attendance:view",
	"GET /api/v1/admin/leaves/stats":     "attendance:view",
	"PUT /api/v1/admin/leaves/:id/audit": "attendance:view",

	// pickup
	"GET /api/v1/admin/pickup/overview":                      "pickup:view",
	"GET /api/v1/admin/pickup/records":                       "pickup:view",
	"GET /api/v1/admin/pickup/abnormal":                      "pickup:view",
	"GET /api/v1/admin/pickup/authorized-persons":            "pickup:view",
	"PUT /api/v1/admin/pickup/authorized-persons/:id/audit":  "pickup:manage",
	"GET /api/v1/admin/pickup/overdue":                       "pickup:view",
	"GET /api/v1/admin/pickup/overdue/count":                 "pickup:view",
	"POST /api/v1/admin/pickup/overdue/notify":               "pickup:manage",

	// growth
	"GET /api/v1/admin/growth/stats":              "growth:view",
	"GET /api/v1/admin/growth/logs":               "growth:view",
	"GET /api/v1/admin/growth/logs/:id":           "growth:view",
	"DELETE /api/v1/admin/growth/logs/:id":        "growth:manage",
	"PUT /api/v1/admin/growth/logs/:id/audit":     "growth:audit",

	// tasks
	"GET /api/v1/admin/tasks":        "task:view",
	"GET /api/v1/admin/tasks/stats":  "task:view",
	"POST /api/v1/admin/tasks":       "task:manage",
	"GET /api/v1/admin/tasks/:id":    "task:view",
	"PUT /api/v1/admin/tasks/:id":    "task:manage",
	"DELETE /api/v1/admin/tasks/:id": "task:manage",

	// points
	"GET /api/v1/admin/points/overview": "points:view",
	"GET /api/v1/admin/points/records":  "points:view",
	"POST /api/v1/admin/points/adjust":  "points:manage",
	"GET /api/v1/admin/points/rules":    "points:view",
	"PUT /api/v1/admin/points/rules":    "points:manage",

	// achievements
	"GET /api/v1/admin/achievements":        "achievement:view",
	"POST /api/v1/admin/achievements":       "achievement:manage",
	"PUT /api/v1/admin/achievements/:id":    "achievement:manage",
	"DELETE /api/v1/admin/achievements/:id": "achievement:manage",

	// notices
	"GET /api/v1/admin/notices":           "notice:view",
	"POST /api/v1/admin/notices":          "notice:manage",
	"PUT /api/v1/admin/notices/:id/audit": "notice:audit",
	"GET /api/v1/admin/notices/:id":       "notice:view",
	"PUT /api/v1/admin/notices/:id":       "notice:manage",
	"DELETE /api/v1/admin/notices/:id":    "notice:manage",

	// announcements
	"GET /api/v1/admin/announcements":          "announcement:view",
	"POST /api/v1/admin/announcements":         "announcement:manage",
	"PUT /api/v1/admin/announcements/:id/top":  "announcement:manage",
	"GET /api/v1/admin/announcements/:id":      "announcement:view",
	"PUT /api/v1/admin/announcements/:id":      "announcement:manage",
	"DELETE /api/v1/admin/announcements/:id":   "announcement:manage",

	// audit
	"GET /api/v1/admin/audit/list":  "audit:view",
	"GET /api/v1/admin/audit/stats": "audit:view",
	"PUT /api/v1/admin/audit/:id":   "audit:manage",

	// export —— 共享端点按导出功能总码；growth 无专用导出码（design D7 备注）
	"POST /api/v1/admin/export/attendance":         "attendance:export",
	"POST /api/v1/admin/export/students":           "student:export",
	"POST /api/v1/admin/export/growth":             "growth:view",
	"GET /api/v1/admin/export/download/:taskId":    "attendance:export",
	"GET /api/v1/admin/export/:taskId":             "attendance:export",

	// logs / health
	"GET /api/v1/admin/logs":        "system:log",
	"GET /api/v1/admin/health/stats": "dashboard:view",

	// dicts
	"GET /api/v1/admin/dicts/types":             "system:kindergarten",
	"POST /api/v1/admin/dicts/types":            "system:kindergarten",
	"PUT /api/v1/admin/dicts/types/:id":         "system:kindergarten",
	"DELETE /api/v1/admin/dicts/types/:id":      "system:kindergarten",
	"GET /api/v1/admin/dicts/types/:id/items":   "system:kindergarten",
	"POST /api/v1/admin/dicts/types/:id/items":  "system:kindergarten",
	"PUT /api/v1/admin/dicts/items/sort":        "system:kindergarten",
	"PUT /api/v1/admin/dicts/items/:id":         "system:kindergarten",
	"DELETE /api/v1/admin/dicts/items/:id":      "system:kindergarten",

	// calendar —— 园级日历（园级事件；发布者=园长角色；医生走 /doctor 路由不经本表）
	"GET /api/v1/admin/calendar/events":        "calendar:view",
	"POST /api/v1/admin/calendar/events":       "calendar:manage",
	"PUT /api/v1/admin/calendar/events/:id":    "calendar:manage",
	"DELETE /api/v1/admin/calendar/events/:id": "calendar:manage",

	// timetable —— 课表维护与同步
	"GET /api/v1/admin/timetable/templates/:grade":  "timetable:view",
	"PUT /api/v1/admin/timetable/templates/:grade":  "timetable:manage",
	"GET /api/v1/admin/timetable/classes/:class_id": "timetable:view",
	"PUT /api/v1/admin/timetable/classes/:class_id": "timetable:manage",
	"GET /api/v1/admin/timetable/setting":           "timetable:view",
	"PUT /api/v1/admin/timetable/setting":           "timetable:manage",
	"POST /api/v1/admin/timetable/sync":             "timetable:manage",
}

// RequirePermission 按登记表强制管理端接口权限，挂在 adminG 组上（AuthJWT/CasbinRBAC 之后）。
// 缺码 403；登记表缺项一律拒绝并留痕（fail-closed）。
func RequirePermission(resolver PermissionResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		if resolver == nil {
			// 未接线（如单测直接构造 router）时不拦截，与 CasbinRBAC 的 nil 语义一致
			c.Next()
			return
		}
		claims := GetClaims(c)
		if claims == nil {
			// 认证由 AuthJWT 负责；挂载位置正确时不会走到这里，拒绝兜底
			response.ErrUnauthorized(c, "未认证")
			c.Abort()
			return
		}
		key := c.Request.Method + " " + c.FullPath()
		code, registered := adminRoutePermission[key]
		if !registered {
			log.L(c.Request.Context()).Error("admin permission registry gap", "route", key)
			response.ErrForbidden(c, "该接口未登记权限，已拒绝")
			c.Abort()
			return
		}
		uid, err := strconv.ParseUint(claims.UserID, 10, 64)
		if err != nil || uid == 0 {
			response.ErrUnauthorized(c, "未认证")
			c.Abort()
			return
		}
		codes, wildcard, err := resolver.ResolveAdminCodes(c.Request.Context(), claims.KindergartenID, uid)
		if err != nil {
			// 不了静默：解析失败按 500 处理，绝不按“无权限”吞掉
			log.L(c.Request.Context()).Error("admin permission resolve failed", "err", err, "route", key)
			response.ErrInternal(c, "权限校验失败")
			c.Abort()
			return
		}
		if wildcard || codeIn(codes, code) {
			c.Next()
			return
		}
		response.ErrForbidden(c, "无权限访问该资源")
		c.Abort()
	}
}

func codeIn(codes []string, code string) bool {
	for _, c := range codes {
		if strings.TrimSpace(c) == code {
			return true
		}
	}
	return false
}

// AdminPermissionGaps 返回已注册管理端路由中未登记权限码的 "METHOD path" 列表，
// 供启动自检使用（design D1）：返回非空即启动失败。
func AdminPermissionGaps(routes []gin.RouteInfo) []string {
	return adminPermissionGaps(routes, adminRoutePermission)
}

func adminPermissionGaps(routes []gin.RouteInfo, table map[string]string) []string {
	var gaps []string
	for _, rt := range routes {
		if rt.Path != "/api/v1/admin" && !strings.HasPrefix(rt.Path, "/api/v1/admin/") {
			continue
		}
		key := rt.Method + " " + rt.Path
		if _, ok := table[key]; !ok {
			gaps = append(gaps, key)
		}
	}
	return gaps
}
