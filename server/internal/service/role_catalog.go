package service

import (
	"context"
	"errors"
	"sync"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

type permissionModuleResp struct {
	Module      string               `json:"module"`
	ModuleLabel string               `json:"module_label"`
	Items       []permissionItemResp `json:"items"`
}

type permissionItemResp struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// permissionTree modules match backend/api/api-admin/17-roles.md GET /admin/permissions.
func adminPermissionModules() []permissionModuleResp {
	return []permissionModuleResp{
		{
			Module: "dashboard", ModuleLabel: "数据看板",
			Items: []permissionItemResp{{Key: "dashboard:view", Label: "查看看板"}},
		},
		{
			Module: "student", ModuleLabel: "学生管理",
			Items: []permissionItemResp{
				{Key: "student:view", Label: "查看学生"},
				{Key: "student:manage", Label: "管理学生"},
				{Key: "student:import", Label: "批量导入"},
				{Key: "student:export", Label: "导出名册"},
			},
		},
		{
			Module: "teacher", ModuleLabel: "老师管理",
			Items: []permissionItemResp{
				{Key: "teacher:view", Label: "查看老师"},
				{Key: "teacher:manage", Label: "管理老师"},
			},
		},
		{
			Module: "parent", ModuleLabel: "家长管理",
			Items: []permissionItemResp{
				{Key: "parent:view", Label: "查看家长"},
				{Key: "parent:manage", Label: "管理家长"},
			},
		},
		{
			Module: "notice", ModuleLabel: "通知管理",
			Items: []permissionItemResp{
				{Key: "notice:view", Label: "查看通知"},
				{Key: "notice:manage", Label: "管理通知"},
				{Key: "notice:audit", Label: "审核通知"},
			},
		},
		{
			Module: "growth", ModuleLabel: "成长记录",
			Items: []permissionItemResp{
				{Key: "growth:view", Label: "查看记录"},
				{Key: "growth:manage", Label: "管理记录"},
				{Key: "growth:audit", Label: "审核记录"},
			},
		},
		{
			Module: "attendance", ModuleLabel: "出勤管理",
			Items: []permissionItemResp{
				{Key: "attendance:view", Label: "查看出勤"},
				{Key: "attendance:export", Label: "导出报表"},
			},
		},
		{
			Module: "pickup", ModuleLabel: "接送管理",
			Items: []permissionItemResp{
				{Key: "pickup:view", Label: "查看接送"},
				{Key: "pickup:manage", Label: "管理接送"},
			},
		},
		{
			Module: "task", ModuleLabel: "亲子任务",
			Items: []permissionItemResp{
				{Key: "task:view", Label: "查看任务"},
				{Key: "task:manage", Label: "管理任务"},
			},
		},
		{
			Module: "points", ModuleLabel: "积分管理",
			Items: []permissionItemResp{
				{Key: "points:view", Label: "查看积分"},
				{Key: "points:manage", Label: "管理积分"},
			},
		},
		{
			Module: "achievement", ModuleLabel: "成就管理",
			Items: []permissionItemResp{
				{Key: "achievement:view", Label: "查看成就"},
				{Key: "achievement:manage", Label: "管理成就"},
			},
		},
		{
			Module: "announcement", ModuleLabel: "公告管理",
			Items: []permissionItemResp{
				{Key: "announcement:view", Label: "查看公告"},
				{Key: "announcement:manage", Label: "管理公告"},
			},
		},
		{
			Module: "audit", ModuleLabel: "内容审核",
			Items: []permissionItemResp{
				{Key: "audit:view", Label: "查看审核"},
				{Key: "audit:manage", Label: "执行审核"},
			},
		},
		{
			Module: "calendar", ModuleLabel: "园级日历",
			Items: []permissionItemResp{
				{Key: "calendar:view", Label: "查看园级日历"},
				{Key: "calendar:manage", Label: "管理园级事件"},
			},
		},
		{
			Module: "timetable", ModuleLabel: "课表管理",
			Items: []permissionItemResp{
				{Key: "timetable:view", Label: "查看课表"},
				{Key: "timetable:manage", Label: "管理课表"},
			},
		},
		{
			Module: "admin", ModuleLabel: "管理员",
			Items: []permissionItemResp{
				{Key: "admin:view", Label: "查看管理员"},
				{Key: "admin:manage", Label: "管理管理员"},
				{Key: "admin:role", Label: "角色权限配置"},
			},
		},
		{
			Module: "system", ModuleLabel: "系统设置",
			Items: []permissionItemResp{
				{Key: "system:kindergarten", Label: "园所设置"},
				{Key: "system:log", Label: "操作日志"},
			},
		},
	}
}

func flattenCatalogPermissions(mods []permissionModuleResp) []domain.AdminPermission {
	out := make([]domain.AdminPermission, 0, 64)
	for _, m := range mods {
		for _, it := range m.Items {
			out = append(out, domain.AdminPermission{
				Code:        it.Key,
				Name:        it.Label,
				Module:      m.Module,
				Description: "",
			})
		}
	}
	out = append(out, domain.AdminPermission{
		Code: "*", Name: "全部权限", Module: "system", Description: "超级管理员",
	})
	return out
}

var (
	validPermOnce sync.Once
	validPermSet  map[string]struct{}
)

func isValidPermissionCode(code string) bool {
	validPermOnce.Do(func() {
		validPermSet = make(map[string]struct{})
		for _, mod := range adminPermissionModules() {
			for _, it := range mod.Items {
				validPermSet[it.Key] = struct{}{}
			}
		}
		validPermSet["*"] = struct{}{}
	})
	_, ok := validPermSet[code]
	return ok
}

func seedAdminPermissions(ctx context.Context, db *gorm.DB) error {
	for _, row := range flattenCatalogPermissions(adminPermissionModules()) {
		var existing domain.AdminPermission
		err := db.WithContext(ctx).Where("code = ?", row.Code).Take(&existing).Error
		if err == nil {
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		cp := row
		if err := db.WithContext(ctx).Create(&cp).Error; err != nil {
			return err
		}
	}
	return nil
}

// principalRoleNames 园级事件发布角色（align-class-calendar 产品裁决，2026-10-16）。
var principalRoleNames = []string{"行政园长", "教学园长", "后勤园长", "招生园长"}

// SeedPrincipalRolesForAll 为每个园所幂等补齐 4 个园长角色，并各挂日历管理权限码
// （calendar:view / calendar:manage）。角色为非系统角色，园所可在角色页调整权限。
// 先确保「园所管理员」存在：保持既有「首个后台角色 = 新管理员默认绑定角色」语义不被园长角色抢占。
func SeedPrincipalRolesForAll(ctx context.Context, db *gorm.DB) error {
	if err := seedAdminPermissions(ctx, db); err != nil {
		return err
	}
	var perms []domain.AdminPermission
	if err := db.WithContext(ctx).
		Where("code IN ?", []string{"calendar:view", "calendar:manage"}).
		Find(&perms).Error; err != nil {
		return err
	}
	if len(perms) == 0 {
		return nil // 不应发生：seedAdminPermissions 已先行
	}
	var kgs []domain.Kindergarten
	if err := db.WithContext(ctx).Find(&kgs).Error; err != nil {
		return err
	}
	for _, kg := range kgs {
		if _, err := repository.NewAdminRepo(db).GetOrCreatePrimaryAdminRole(ctx, kg.ID); err != nil {
			return err
		}
		for _, name := range principalRoleNames {
			var existing domain.AdminRole
			err := db.WithContext(ctx).
				Where("kindergarten_id = ? AND name = ?", kg.ID, name).
				Take(&existing).Error
			if err == nil {
				continue
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			role := &domain.AdminRole{
				KindergartenID: kg.ID, Name: name,
				Description: "园级事件发布角色（系统初始化）", IsSystem: false,
			}
			if err := db.WithContext(ctx).Create(role).Error; err != nil {
				return err
			}
			for _, p := range perms {
				if err := db.WithContext(ctx).Create(&domain.AdminRolePermission{
					RoleID: role.ID, PermissionID: p.ID,
				}).Error; err != nil {
					return err
				}
			}
		}
	}
	return nil
}
