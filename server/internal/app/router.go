package app

import (
	"fmt"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/auth"
	handler "kindergarten-server/internal/handler/http"
	"kindergarten-server/internal/middleware"
)

func NewRouter(h *AppHandler, e *casbin.Enforcer, secret JWTSecret, resolver middleware.PermissionResolver) (*gin.Engine, error) {
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
		authed.Use(middleware.AuthJWT([]byte(secret), h.tokenRevoker))
		authed.Use(middleware.CasbinRBAC(e))
		{
			// 退出登录
			authed.POST("/auth/logout", h.inner.Logout)

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

			// 管理端细闸：按路由→权限码登记表强制（AuthJWT/CasbinRBAC 之后）
			adminG := authed.Group("/admin")
			adminG.Use(middleware.RequirePermission(resolver))
			registerAdminRoutes(adminG, h.inner)

			teacherG := authed.Group("/teacher")
			registerTeacherRoutes(teacherG, h.inner)

			parentG := authed.Group("/parent")
			registerParentRoutes(parentG, h.inner)

			doctorG := authed.Group("/doctor")
			registerDoctorRoutes(doctorG, h.inner)
		}
	}

	// 启动自检：已注册管理端路由必须全部登记权限码（wire-admin-permissions design D1）。
	// 缺口存在即启动失败，不允许带着缺口对外服务。
	if gaps := middleware.AdminPermissionGaps(r.Routes()); len(gaps) > 0 {
		return nil, fmt.Errorf("admin permission registry has %d gap(s): %s", len(gaps), strings.Join(gaps, "; "))
	}

	return r, nil
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
		growth.POST("/comments/:id/like", h.TeacherLikeGrowthComment)
		growth.DELETE("/comments/:id/like", h.TeacherUnlikeGrowthComment)
		growth.GET("/logs/:id/likes", h.TeacherGrowthLikes)
		growth.POST("/logs/:id/like", h.TeacherGrowthLike)
		growth.DELETE("/logs/:id/like", h.TeacherGrowthUnlike)
		growth.GET("/logs/:id/related-students", h.TeacherGrowthRelatedStudents)
		growth.GET("/logs/:id/tagging-status", h.TeacherGrowthTaggingStatus)               // 照片识别状态（轮询）
		growth.POST("/logs/:id/suggestions/confirm-high", h.TeacherConfirmHighSuggestions) // 一键确认高置信建议
		growth.PUT("/media/:id/students/:sid", h.TeacherReviewMediaSuggestion)             // 识别建议确认/拒绝
		growth.POST("/media/:id/class-roster", h.TeacherApplyClassRoster)                  // 大合影「一键全班」（D11）
		growth.POST("/media/upload", h.TeacherGrowthMediaUpload)
	}

	// 成就徽章：老师端只列本园已启用的徽章 + 授予（授予可选绑定成长记录为获奖见证）
	achT := teacherG.Group("/achievements")
	{
		achT.GET("", h.TeacherListAchievements)
		achT.POST("/:id/grant", h.TeacherGrantAchievement)
	}

	pickup := teacherG.Group("/pickup")
	{
		pickup.GET("/overview", h.TeacherPickupOverview)
		pickup.GET("/students", h.TeacherPickupStudents)
		pickup.GET("/students/:id/authorized-persons", h.TeacherPickupStudentAuthPersons) // teacher-pickup-auth-picker
		pickup.POST("/scan", h.TeacherPickupScan)
		pickup.POST("/manual", h.TeacherPickupManual)
		pickup.GET("/records", h.TeacherPickupRecords)
		pickup.GET("/records/:id", h.TeacherPickupRecord)
		pickup.PUT("/records/:id/abnormal", h.TeacherPickupAbnormal)
		pickup.POST("/notify-overdue", h.TeacherPickupNotifyOverdue) // 本次新增：滞留一键通知
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
		class.GET("/present-students", h.TeacherPresentStudents) // 当日到场（D11）
		class.GET("/students/:id", h.TeacherClassStudent)
		// 班级星星罐目标（C 切片 3.8）：按学期独立配置，任务 StarScope=class 时使用。
		class.GET("/star-jar", h.TeacherGetClassStarJar)
		class.PUT("/star-jar", h.TeacherSetClassStarJar)
	}

	// 班级相册（openspec class-album）。/stats 是静态路径，与 /:id 并存时静态优先，
	// 这里先声明以保持「统计、列表、创建、详情」的阅读顺序。
	// 生命周期：创建即草稿，publish/unpublish 切状态，edit 不改变状态，delete 为软删。
	albums := teacherG.Group("/albums")
	{
		albums.GET("/stats", h.TeacherAlbumStats)
		albums.GET("", h.TeacherListAlbums)
		albums.POST("", h.TeacherCreateAlbum)
		albums.GET("/:id", h.TeacherGetAlbum)
		albums.PUT("/:id", h.TeacherUpdateAlbum)
		albums.DELETE("/:id", h.TeacherDeleteAlbum)
		albums.POST("/:id/publish", h.TeacherPublishAlbum)
		albums.POST("/:id/unpublish", h.TeacherUnpublishAlbum)
	}

	messages := teacherG.Group("/messages")
	{
		messages.GET("", h.TeacherListMessages)
		// read-all 是全局动作，先声明；与 /:id/read 段数不同，不冲突。
		messages.PUT("/read-all", h.TeacherMarkAllMessagesRead)
		messages.PUT("/:id/read", h.TeacherMarkMessageRead)
	}

	tasks := teacherG.Group("/tasks")
	{
		tasks.GET("", h.TeacherListTasks)
		tasks.POST("", h.TeacherCreateTask)
		tasks.GET("/:id", h.TeacherGetTask)
		tasks.PUT("/:id", h.TeacherUpdateTask)
		tasks.DELETE("/:id", h.TeacherDeleteTask)
		tasks.GET("/:id/share-card", h.TeacherGetTaskShareCard)
		tasks.GET("/:id/stats", h.TeacherTaskStats)
		tasks.GET("/:id/feedback-summary", h.TeacherTaskFeedbackSummary)
		tasks.GET("/:id/checkins", h.TeacherTaskCheckins)
		tasks.POST("/:id/checkins/:checkin_id/response", h.TeacherRespondTask)
		tasks.POST("/:id/checkins/:checkin_id/supplement", h.TeacherSupplementResponse)
		tasks.PUT("/:id/checkins/:checkin_id/public", h.TeacherReviewTaskPublic)
		tasks.POST("/:id/batch-response", h.TeacherBatchRespondTask)
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
		// 班级星星罐目标（C 切片 7.5）：与老师端同一 service，权限走 task:*。
		cg.GET("/:id/star-jar", h.GetClassStarJar)
		cg.PUT("/:id/star-jar", h.SetClassStarJar)
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
		sg.GET("/:id/faces", h.ListStudentFaces)             // 人脸底库照片与注册状态
		sg.POST("/:id/faces", h.AddStudentFaces)             // 录入底库照片（file_id 来自统一上传通道）
		sg.POST("/:id/faces/retry", h.RetryStudentFaces)     // 重试底库注册
		sg.DELETE("/:id/faces/:faceId", h.DeleteStudentFace) // 移除底库照片
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
		pk.GET("/overdue", h.AdminPickupOverdue)               // 本次新增：滞留名单 + 各班计数
		pk.GET("/overdue/count", h.AdminPickupOverdueCount)    // 本次新增：超时未接计数（累计口径）
		pk.POST("/overdue/notify", h.AdminPickupNotifyOverdue) // 本次新增：一键通知家长
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
		// 7.3 查看公开内容（只读；撤回走 PUT /tasks/checkins/:checkin_id/public）
		tk.GET("/:id/checkins", h.ListTaskPublicCheckins)
		tk.PUT("/:id", h.UpdateTask)
		tk.DELETE("/:id", h.DeleteTask)
		tk.PUT("/checkins/:checkin_id/public", h.RevokeTaskPublicCheckin)
	}

	// 多班活动：activity_id 由服务端生成，只有本组能管理分组（design D12）。
	ta := adminG.Group("/task-activities")
	{
		ta.POST("", h.CreateTaskActivity)
		ta.PUT("/:activity_id", h.UpdateTaskActivity)
		ta.DELETE("/:activity_id", h.DeleteTaskActivity)
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

	gcal := adminG.Group("/calendar")
	{
		gcal.GET("/events", h.AdminListCalendarEvents)
		gcal.POST("/events", h.AdminCreateCalendarEvent)
		gcal.PUT("/events/:id", h.AdminUpdateCalendarEvent)
		gcal.DELETE("/events/:id", h.AdminDeleteCalendarEvent)
	}

	tt := adminG.Group("/timetable")
	{
		tt.GET("/templates/:grade", h.AdminGetTimetableTemplate)
		tt.PUT("/templates/:grade", h.AdminSaveTimetableTemplate)
		tt.GET("/classes/:class_id", h.AdminGetTimetableClassOverrides)
		tt.PUT("/classes/:class_id", h.AdminSaveTimetableClassOverrides)
		tt.GET("/setting", h.AdminGetTimetableSetting)
		tt.PUT("/setting", h.AdminUpdateTimetableSetting)
		tt.POST("/sync", h.AdminSyncTimetable)
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
		pickup.POST("/designate", h.ParentPickupDesignate)    // 本次新增：指定今日接走人
		pickup.GET("/designation", h.ParentPickupDesignation) // 本次新增：只读查当前指定
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
		tasks.POST("/:id/read", h.ParentMarkTaskRead)
		tasks.POST("/:id/checkin", h.ParentCheckinTask)
		tasks.GET("/:id/checkin-history", h.ParentCheckinHistory)
		tasks.POST("/responses/:response_id/reaction", h.ParentReactResponse)
		tasks.POST("/checkins/:checkin_id/public-request", h.ParentRequestTaskPublic)
		tasks.DELETE("/checkins/:checkin_id/public-request", h.ParentCancelTaskPublic)
		tasks.GET("/public-wall", h.ParentTaskPublicWall)
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

	// 班级相册（openspec class-album-parent-sharing 5.4）：只读、已发布、按当前孩子过滤。
	classAlbums := pg.Group("/class-albums")
	{
		classAlbums.GET("", h.ParentListClassAlbums)
		classAlbums.GET("/:id", h.ParentGetClassAlbum)
	}

	student := pg.Group("/student")
	{
		student.GET("/profile", h.ParentStudentProfile)
		student.PUT("/profile", h.ParentUpdateStudentProfile)
	}

	health := pg.Group("/health")
	{
		health.GET("/records", h.ParentHealthRecords)
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
	// 园级日历（保健医发布体检等园级事件；对外路径 /api/v1/doctor/calendar/*）
	cal := dg.Group("/calendar")
	{
		cal.GET("/events", h.DoctorListCalendarEvents)
		cal.POST("/events", h.DoctorCreateCalendarEvent)
		cal.PUT("/events/:id", h.DoctorUpdateCalendarEvent)
		cal.DELETE("/events/:id", h.DoctorDeleteCalendarEvent)
	}

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
	// tokenRevoker 由认证服务提供，AuthJWT 用它校验登出黑名单；两处必须是同一份存储。
	tokenRevoker auth.TokenRevoker
}

func NewAppHandler(h *handler.Handler) *AppHandler {
	return &AppHandler{inner: h, tokenRevoker: h.TokenRevoker()}
}
