package app

import (
	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"

	handler "kindergarten-server/internal/handler/http"
	"kindergarten-server/internal/middleware"
)

func NewRouter(h *AppHandler, e *casbin.Enforcer, secret JWTSecret) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestContextLogger())

	r.GET("/health", h.inner.HealthCheck)

	v1 := r.Group("/api/v1")
	{
		v1.GET("/system/status", h.inner.SystemStatus)
		v1.POST("/system/init", h.inner.SystemInit)

		v1.POST("/auth/sms/send", h.inner.SendSMS)
		v1.POST("/auth/login/sms", h.inner.SMSLogin)
		v1.POST("/auth/login/wechat", h.inner.WeChatLogin)
		v1.POST("/auth/login/password", h.inner.PasswordLogin)
		v1.POST("/auth/refresh", h.inner.Refresh)

		// 需要认证的接口
		authed := v1.Group("")
		authed.Use(middleware.AuthJWT([]byte(secret)))
		authed.Use(middleware.CasbinRBAC(e))
		{
			// 退出登�?			authed.POST("/auth/logout", h.inner.Logout)

			// 文件上传
			authed.POST("/common/upload", h.inner.Upload)
			authed.POST("/common/upload/batch", h.inner.BatchUpload)

			// 用户信息
			authed.GET("/common/user/profile", h.inner.GetProfile)
			authed.PUT("/common/user/profile", h.inner.UpdateProfile)
			authed.PUT("/common/user/avatar", h.inner.UpdateAvatar)
			authed.POST("/common/user/bind-invite", h.inner.BindInvite)
			authed.POST("/common/user/bind-parent-phone", h.inner.BindParentPhone)

			// 消息
			authed.GET("/common/messages", h.inner.ListMessages)
			authed.GET("/common/messages/unread-count", h.inner.UnreadCount)
			authed.PUT("/common/messages/:id/read", h.inner.MarkMessageRead)
			authed.PUT("/common/messages/read-all", h.inner.MarkAllRead)

			adminG := authed.Group("/admin")
			registerAdminRoutes(adminG, h.inner)

			teacherG := authed.Group("/teacher")
			registerTeacherRoutes(teacherG, h.inner)

			parentG := authed.Group("/parent")
			registerParentRoutes(parentG, h.inner)

			doctorG := authed.Group("/doctor")
			registerDoctorRoutes(doctorG, h.inner)
		}
	}

	return r
}

// registerTeacherRoutes
func registerTeacherRoutes(teacherG *gin.RouterGroup, h *handler.Handler) {
	home := teacherG.Group("/home")
	{
		home.GET("/overview", h.TeacherHomeOverview)
		home.GET("/todos", h.TeacherHomeTodos)
		home.PUT("/todos/:id/action", h.TeacherTodoAction)
	}

	notices := teacherG.Group("/notices")
	{
		notices.GET("", h.TeacherListNotices)
		notices.POST("", h.TeacherCreateNotice)
		notices.GET("/templates", h.TeacherListNoticeTemplates)
		notices.POST("/templates", h.TeacherCreateNoticeTemplate)
		notices.PUT("/templates/:id", h.TeacherUpdateNoticeTemplate)
		notices.DELETE("/templates/:id", h.TeacherDeleteNoticeTemplate)
		notices.GET("/:id", h.TeacherGetNotice)
		notices.PUT("/:id", h.TeacherUpdateNotice)
		notices.DELETE("/:id", h.TeacherDeleteNotice)
		notices.PUT("/:id/revoke", h.TeacherRevokeNotice)
		notices.GET("/:id/read-status", h.TeacherNoticeReadStatus)
	}

	leaves := teacherG.Group("/leaves")
	{
		leaves.GET("", h.TeacherListLeaves)
		leaves.GET("/stats", h.TeacherLeaveStats)
		leaves.GET("/:id", h.TeacherGetLeave)
		leaves.PUT("/:id/approve", h.TeacherApproveLeave)
		leaves.PUT("/:id/reject", h.TeacherRejectLeave)
	}

	growth := teacherG.Group("/growth")
	{
		growth.GET("/logs", h.TeacherListGrowthLogs)
		growth.POST("/logs", h.TeacherCreateGrowthLog)
		growth.GET("/logs/:id", h.TeacherGetGrowthLog)
		growth.PUT("/logs/:id", h.TeacherUpdateGrowthLog)
		growth.DELETE("/logs/:id", h.TeacherDeleteGrowthLog)
		growth.GET("/logs/:id/comments", h.TeacherGrowthComments)
		growth.POST("/logs/:id/comments", h.TeacherCreateGrowthComment)
		growth.GET("/logs/:id/likes", h.TeacherGrowthLikes)
		growth.POST("/logs/:id/like", h.TeacherGrowthLike)
		growth.DELETE("/logs/:id/like", h.TeacherGrowthUnlike)
		growth.GET("/logs/:id/related-students", h.TeacherGrowthRelatedStudents)
		growth.POST("/media/upload", h.TeacherGrowthMediaUpload)
	}

	pickup := teacherG.Group("/pickup")
	{
		pickup.GET("/overview", h.TeacherPickupOverview)
		pickup.GET("/students", h.TeacherPickupStudents)
		pickup.POST("/scan", h.TeacherPickupScan)
		pickup.POST("/manual", h.TeacherPickupManual)
		pickup.GET("/records", h.TeacherPickupRecords)
		pickup.GET("/records/:id", h.TeacherPickupRecord)
		pickup.PUT("/records/:id/abnormal", h.TeacherPickupAbnormal)
		pickup.PUT("/authorized-persons/:id/approve", h.TeacherPickupAuthApprove)
		pickup.PUT("/authorized-persons/:id/reject", h.TeacherPickupAuthReject)
	}

	calendar := teacherG.Group("/calendar")
	{
		calendar.GET("/events", h.TeacherCalendarEvents)
		calendar.POST("/events", h.TeacherCreateCalendarEvent)
		calendar.GET("/events/detail/:id", h.TeacherGetCalendarEvent)
		calendar.GET("/events/:date", h.TeacherCalendarEventsByDate)
		calendar.PUT("/events/:id", h.TeacherUpdateCalendarEvent)
		calendar.DELETE("/events/:id", h.TeacherDeleteCalendarEvent)
		calendar.GET("/dots", h.TeacherCalendarDots)
	}

	class := teacherG.Group("/class")
	{
		class.GET("/info", h.TeacherClassInfo)
			class.GET("/list", h.TeacherClassList)
		class.GET("/students", h.TeacherClassStudents)
		class.GET("/students/:id", h.TeacherClassStudent)
	}

	messages := teacherG.Group("/messages")
	{
		messages.GET("", h.TeacherListMessages)
	}

	tasks := teacherG.Group("/tasks")
	{
		tasks.GET("", h.TeacherListTasks)
		tasks.POST("", h.TeacherCreateTask)
		tasks.PUT("/:id", h.TeacherUpdateTask)
		tasks.DELETE("/:id", h.TeacherDeleteTask)
		tasks.GET("/:id/stats", h.TeacherTaskStats)
		tasks.GET("/:id/checkins", h.TeacherTaskCheckins)
	}

		dict := teacherG.Group("/dicts")
		{
			dict.GET("/:type_key", h.GetDictItems)
			dict.GET("", h.GetDictItemsBatch)
		}
}


func registerAdminRoutes(adminG *gin.RouterGroup, h *handler.Handler) {
	dash := adminG.Group("/dashboard")
	{
		dash.GET("/overview", h.GetDashboardOverview)
		dash.GET("/attendance-trend", h.GetDashboardAttendanceTrend)
		dash.GET("/class-ranking", h.GetDashboardClassRanking)
		dash.GET("/growth-stats", h.GetDashboardGrowthStats)
	}

	cg := adminG.Group("/classes")
	{
		cg.GET("", h.ListClasses)
		cg.POST("", h.CreateClass)
		cg.GET("/:id", h.GetClassDetail)
		cg.PUT("/:id", h.UpdateClass)
		cg.DELETE("/:id", h.DeleteClass)
		cg.POST("/:id/adjust", h.AdjustClassStudents)
	}

	sg := adminG.Group("/students")
	{
		sg.GET("", h.ListStudents)
		sg.POST("", h.CreateStudent)
		sg.POST("/import", h.ImportStudents)
		sg.GET("/export", h.ExportStudents)
		sg.GET("/:id", h.GetStudent)
		sg.PUT("/:id", h.UpdateStudent)
		sg.DELETE("/:id", h.DeleteStudent)
		sg.PUT("/:id/status", h.UpdateStudentStatus)
	}

	tg := adminG.Group("/teachers")
	{
		tg.GET("", h.ListTeachers)
		tg.POST("", h.CreateTeacher)
		tg.GET("/:id", h.GetTeacher)
		tg.PUT("/:id", h.UpdateTeacher)
		tg.DELETE("/:id", h.DeleteTeacher)
		tg.PUT("/:id/status", h.UpdateTeacherStatus)
		tg.GET("/:id/invitation", h.GetTeacherInvitation)
		tg.POST("/:id/invitation", h.RegenerateTeacherInvitation)
		tg.GET("/:id/invitation/qrcode", h.GetTeacherInviteQRCode)
	}

	pg := adminG.Group("/parents")
	{
		pg.GET("", h.ListParents)
		pg.POST("", h.CreateParent)
		pg.GET("/:id", h.GetParent)
		pg.PUT("/:id", h.UpdateParent)
		pg.PUT("/:id/status", h.UpdateParentStatus)
	}

	set := adminG.Group("/settings")
	{
		set.GET("", h.GetSettings)
		set.PUT("", h.UpdateSettings)
		set.GET("/academic-years", h.ListAcademicYears)
		set.POST("/academic-years", h.CreateAcademicYear)
	}

	kg := adminG.Group("/kindergarten")
	{
		kg.GET("/info", h.GetKindergartenInfo)
		kg.PUT("/info", h.UpdateKindergartenInfo)
	}

	ag := adminG.Group("/admins")
	{
		ag.GET("", h.ListAdmins)
		ag.POST("", h.CreateAdmin)
		ag.PUT("/:id/reset-password", h.ResetAdminPassword)
		ag.PUT("/:id", h.UpdateAdmin)
		ag.DELETE("/:id", h.DeleteAdmin)
	}

	rg := adminG.Group("/roles")
	{
		rg.GET("", h.ListRoles)
		rg.POST("", h.CreateRole)
		rg.PUT("/:id", h.UpdateRole)
		rg.DELETE("/:id", h.DeleteRole)
	}
	adminG.GET("/permissions", h.GetPermissionTree)

	att := adminG.Group("/attendance")
	{
		att.GET("/overview", h.GetAttendanceOverview)
		att.GET("/records", h.GetAttendanceRecords)
		att.GET("/report", h.ExportAttendanceReport)
	}

	lg := adminG.Group("/leaves")
	{
		lg.GET("", h.ListLeaves)
		lg.GET("/stats", h.GetLeaveStats)
		lg.PUT("/:id/audit", h.AuditLeave)
	}

	pk := adminG.Group("/pickup")
	{
		pk.GET("/overview", h.GetPickupOverview)
		pk.GET("/records", h.GetPickupRecords)
		pk.GET("/abnormal", h.GetPickupAbnormal)
		pk.GET("/authorized-persons", h.GetAuthorizedPersons)
		pk.PUT("/authorized-persons/:id/audit", h.AuditPickupAuthorizedPerson)
	}

	gr := adminG.Group("/growth")
	{
		gr.GET("/stats", h.GetGrowthStats)
		gr.GET("/logs", h.ListGrowthLogs)
		gr.GET("/logs/:id", h.GetGrowthLog)
		gr.DELETE("/logs/:id", h.DeleteGrowthLog)
		gr.PUT("/logs/:id/audit", h.AuditGrowthLog)
	}

	tk := adminG.Group("/tasks")
	{
		tk.GET("/stats", h.GetTaskStats)
		tk.GET("", h.ListTasks)
		tk.POST("", h.CreateTask)
		tk.GET("/:id", h.GetTask)
		tk.PUT("/:id", h.UpdateTask)
		tk.DELETE("/:id", h.DeleteTask)
	}

	po := adminG.Group("/points")
	{
		po.GET("/overview", h.GetPointsOverview)
		po.GET("/records", h.ListPointRecords)
		po.POST("/adjust", h.AdjustPoints)
		po.GET("/rules", h.GetPointRules)
		po.PUT("/rules", h.UpdatePointRules)
	}

	ac := adminG.Group("/achievements")
	{
		ac.GET("", h.ListAchievements)
		ac.POST("", h.CreateAchievement)
		ac.PUT("/:id", h.UpdateAchievement)
		ac.DELETE("/:id", h.DeleteAchievement)
	}

	ng := adminG.Group("/notices")
	{
		ng.GET("", h.ListNotices)
		ng.POST("", h.CreateNotice)
		ng.PUT("/:id/audit", h.AuditNotice)
		ng.GET("/:id", h.GetNotice)
		ng.PUT("/:id", h.UpdateNotice)
		ng.DELETE("/:id", h.DeleteNotice)
	}

	an := adminG.Group("/announcements")
	{
		an.GET("", h.ListAnnouncements)
		an.POST("", h.CreateAnnouncement)
		an.PUT("/:id/top", h.SetAnnouncementTop)
		an.GET("/:id", h.GetAnnouncement)
		an.PUT("/:id", h.UpdateAnnouncement)
		an.DELETE("/:id", h.DeleteAnnouncement)
	}

	ad := adminG.Group("/audit")
	{
		ad.GET("/list", h.GetAuditList)
		ad.GET("/stats", h.GetAuditStats)
		ad.PUT("/:id", h.AuditContent)
	}

	ex := adminG.Group("/export")
	{
		ex.POST("/attendance", h.PostExportAttendance)
		ex.POST("/students", h.PostExportStudents)
		ex.POST("/growth", h.PostExportGrowth)
		ex.GET("/download/:taskId", h.DownloadExportFile)
		ex.GET("/:taskId", h.GetExportTask)
	}

	adminG.GET("/logs", h.ListOperationLogs)

		health := adminG.Group("/health")
		{
			health.GET("/stats", h.AdminHealthStats)
		}

		dict := adminG.Group("/dicts")
		{
			dict.GET("/types", h.ListDictTypes)
			dict.POST("/types", h.CreateDictType)
			dict.PUT("/types/:id", h.UpdateDictType)
			dict.DELETE("/types/:id", h.DeleteDictType)
			dict.GET("/types/:id/items", h.ListDictItems)
			dict.POST("/types/:id/items", h.CreateDictItem)
			dict.PUT("/items/:id", h.UpdateDictItem)
			dict.DELETE("/items/:id", h.DeleteDictItem)
			dict.PUT("/items/sort", h.BatchSortDictItems)
		}
	}


func registerParentRoutes(pg *gin.RouterGroup, h *handler.Handler) {
	home := pg.Group("/home")
	{
		home.GET("/overview", h.ParentHomeOverview)
		home.PUT("/switch-student", h.ParentSwitchStudent)
		home.GET("/announcements", h.ParentAnnouncements)
		home.GET("/feeds", h.ParentHomeFeeds)
	}

	notices := pg.Group("/notices")
	{
		notices.GET("", h.ParentListNotices)
		notices.GET("/:id", h.ParentGetNotice)
		notices.PUT("/:id/read", h.ParentMarkNoticeRead)
		notices.PUT("/:id/confirm", h.ParentConfirmNotice)
	}

	leaves := pg.Group("/leaves")
	{
		leaves.GET("", h.ParentListLeaves)
		leaves.GET("/:id", h.ParentGetLeave)
		leaves.POST("", h.ParentCreateLeave)
		leaves.PUT("/:id", h.ParentUpdateLeave)
		leaves.PUT("/:id/cancel", h.ParentCancelLeave)
	}

	growth := pg.Group("/growth")
	{
		growth.GET("/feed", h.ParentGrowthFeed)
		growth.GET("/feed/:id", h.ParentGrowthDetail)
		growth.POST("/feed/:id/like", h.ParentGrowthLike)
		growth.DELETE("/feed/:id/like", h.ParentGrowthUnlike)
		growth.GET("/feed/:id/comments", h.ParentGrowthComments)
		growth.POST("/feed/:id/comments", h.ParentCreateGrowthComment)
		growth.GET("/album", h.ParentGrowthAlbum)
		growth.GET("/report", h.ParentGrowthReport)
	}

	pickup := pg.Group("/pickup")
	{
		pickup.GET("/qrcode", h.ParentPickupQRCode)
		pickup.GET("/records", h.ParentPickupRecords)
		pickup.GET("/authorized-persons", h.ParentAuthorizedPersons)
		pickup.POST("/authorized-persons", h.ParentCreateAuthorizedPerson)
		pickup.PUT("/authorized-persons/:id", h.ParentUpdateAuthorizedPerson)
		pickup.DELETE("/authorized-persons/:id", h.ParentDeleteAuthorizedPerson)
	}

	tasks := pg.Group("/tasks")
	{
		tasks.GET("", h.ParentListTasks)
		tasks.GET("/:id", h.ParentGetTask)
		tasks.POST("/:id/checkin", h.ParentCheckinTask)
		tasks.GET("/:id/checkin-history", h.ParentCheckinHistory)
	}

	points := pg.Group("/points")
	{
		points.GET("/overview", h.ParentPointsOverview)
		points.GET("/history", h.ParentPointsHistory)
		points.GET("/leaderboard", h.ParentLeaderboard)
	}

	pg.GET("/achievements", h.ParentAchievements)

	class := pg.Group("/class")
	{
		class.GET("/info", h.ParentClassInfo)
		class.GET("/calendar", h.ParentClassCalendar)
	}

	student := pg.Group("/student")
	{
		student.GET("/profile", h.ParentStudentProfile)
		student.PUT("/profile", h.ParentUpdateStudentProfile)
	}

	pg.GET("/students", h.ParentListStudents)
	pg.POST("/students/bind", h.ParentBindStudent)
	pg.DELETE("/students/:id", h.ParentUnbindStudent)

	pg.GET("/feedback", h.ParentListFeedback)
	pg.POST("/feedback", h.ParentCreateFeedback)

	pg.GET("/settings", h.ParentGetSettings)
	pg.PUT("/settings", h.ParentUpdateSettings)
}

func registerDoctorRoutes(dg *gin.RouterGroup, h *handler.Handler) {
	health := dg.Group("/health")
	{
		health.GET("/queue", h.DoctorGetQueue)
		health.GET("/check/:id", h.DoctorGetCheckDetail)
		health.PUT("/check/:id", h.DoctorSubmitCheck)
		health.POST("/check/:id/media", h.DoctorAddMedia)
		health.GET("/today/stats", h.DoctorTodayStats)
	}
}

type AppHandler struct {
	inner *handler.Handler
}

func NewAppHandler(h *handler.Handler) *AppHandler {
	return &AppHandler{inner: h}
}
