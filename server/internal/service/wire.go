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
	NewTeacherPortalService,
	NewParentPortalService,
	NewDictService,
	NewHealthCheckService,
)
