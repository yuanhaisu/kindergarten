package service

import "github.com/google/wire"

var Set = wire.NewSet(
	NewFileService,
	NewMessageService,
	NewSystemService,
	NewDashboardService,
	NewClassService,
	NewStudentService,
	NewTeacherService,
	NewParentService,
	NewSettingsService,
	NewAdminService,
	NewRoleService,
	NewAttendanceService,
	NewLeaveService,
	NewPickupService,
	NewGrowthService,
	NewTaskService,
	NewPointService,
	NewAchievementService,
	NewNoticeService,
	NewAnnouncementService,
	NewAuditService,
	NewOperationLogService,
	NewExportService,
	NewRecognitionQueueForApp,
	NewTeacherPortalService,
	NewParentPortalService,
	NewDictService,
	NewHealthCheckService,
	NewGardenCalendarService,
	NewTimetableServiceForApp,
	NewAdminAuthzService,
	NewFaceRegistryService,

	// 晨检服务依赖策略读取接口，而不是整个 SettingsService。
	wire.Bind(new(HealthPolicyProvider), new(*SettingsService)),
)
