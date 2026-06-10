package handler

import (
	"kindergarten-server/internal/service"
)

// Handler 聚合各模块的 HTTP 方法。新增业务模块时：
// 1) 在此结构体增加 *service.XxxService 字段；
// 2) 调整 NewHandler 构造函数；
// 3) 在 internal/app/router.go 的 registerAdminRoutes 中注册路径。
type Handler struct {
	authSvc          *service.AuthService
	fileSvc          *service.FileService
	msgSvc           *service.MessageService
	systemSvc        *service.SystemService
	dashSvc          *service.DashboardService
	classSvc         *service.ClassService
	studentSvc       *service.StudentService
	teacherSvc       *service.TeacherService
	parentSvc        *service.ParentService
	settingsSvc      *service.SettingsService
	adminSvc         *service.AdminService
	roleSvc          *service.RoleService
	attendanceSvc    *service.AttendanceService
	leaveSvc         *service.LeaveService
	pickupSvc        *service.PickupService
	growthSvc        *service.GrowthService
	taskSvc          *service.TaskService
	pointSvc         *service.PointService
	achievementSvc   *service.AchievementService
	noticeSvc        *service.NoticeService
	announcementSvc  *service.AnnouncementService
	auditSvc         *service.AuditService
	oplogSvc         *service.OperationLogService
	exportSvc        *service.ExportService
	teacherPortalSvc *service.TeacherPortalService
	parentPortalSvc  *service.ParentPortalService
	dictSvc          *service.DictService
	healthCheckSvc   *service.HealthCheckService
}

func NewHandler(
	authSvc *service.AuthService,
	fileSvc *service.FileService,
	msgSvc *service.MessageService,
	systemSvc *service.SystemService,
	dashSvc *service.DashboardService,
	classSvc *service.ClassService,
	studentSvc *service.StudentService,
	teacherSvc *service.TeacherService,
	parentSvc *service.ParentService,
	settingsSvc *service.SettingsService,
	adminSvc *service.AdminService,
	roleSvc *service.RoleService,
	attendanceSvc *service.AttendanceService,
	leaveSvc *service.LeaveService,
	pickupSvc *service.PickupService,
	growthSvc *service.GrowthService,
	taskSvc *service.TaskService,
	pointSvc *service.PointService,
	achievementSvc *service.AchievementService,
	noticeSvc *service.NoticeService,
	announcementSvc *service.AnnouncementService,
	auditSvc *service.AuditService,
	oplogSvc *service.OperationLogService,
	exportSvc *service.ExportService,
	teacherPortalSvc *service.TeacherPortalService,
	parentPortalSvc *service.ParentPortalService,
	dictSvc *service.DictService,
	healthCheckSvc *service.HealthCheckService,
) *Handler {
	return &Handler{
		authSvc:          authSvc,
		fileSvc:          fileSvc,
		msgSvc:           msgSvc,
		systemSvc:        systemSvc,
		dashSvc:          dashSvc,
		classSvc:         classSvc,
		studentSvc:       studentSvc,
		teacherSvc:       teacherSvc,
		parentSvc:        parentSvc,
		settingsSvc:      settingsSvc,
		adminSvc:         adminSvc,
		roleSvc:          roleSvc,
		attendanceSvc:    attendanceSvc,
		leaveSvc:         leaveSvc,
		pickupSvc:        pickupSvc,
		growthSvc:        growthSvc,
		taskSvc:          taskSvc,
		pointSvc:         pointSvc,
		achievementSvc:   achievementSvc,
		noticeSvc:        noticeSvc,
		announcementSvc:  announcementSvc,
		auditSvc:         auditSvc,
		oplogSvc:         oplogSvc,
		exportSvc:        exportSvc,
		teacherPortalSvc: teacherPortalSvc,
		parentPortalSvc:  parentPortalSvc,
		dictSvc:          dictSvc,
		healthCheckSvc:   healthCheckSvc,
	}
}
