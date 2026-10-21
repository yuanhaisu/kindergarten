package service

// 这些测试打真实 MySQL，覆盖 align-class-calendar 的园级发布路径：
// TC-NEW-CAL-0016（园长发布成功）、TC-NEW-CAL-0012（园级通知面向全园家长）、
// TC-NEW-CAL-0017 支撑（老师路径只能产生班级事件）、以及 4.3 园长角色播种的幂等性。
// 复用 calendar_semantics_test.go 的 calFixture（9903xx 区间，t.Cleanup 硬删）。
// 说明：医生端与服务端共用 GardenCalendarService；路由层角色鉴权（doctor/admin）为既有
// 基础设施（Casbin + 权限登记表），其实调在端页面验收（组 7）完成。

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

func newGardenSvc(f *calFixture) *GardenCalendarService {
	return NewGardenCalendarService(f.db, repository.NewCalendarRepo(f.db),
		NewNoticeService(repository.NewNoticeRepo(f.db), repository.NewOperationLogRepo(f.db)))
}

// TC-NEW-CAL-0016 + 0012：园长发布园级事件（scope=kindergarten、class_id 为 NULL）；
// 确认通知时面向全园家长（目标=本园全部在读班级），两班家长均可见。
func TestGardenCalendar_Publish_NotifiesAllClasses(t *testing.T) {
	f := newCalFixture(t)
	ctx := context.Background()
	svc := newGardenSvc(f)

	out, err := svc.CreateGardenEvent(ctx, testCalKG, testCalTeacherA, GardenCalendarInput{
		Title: "园级春季家长会", EventType: "meeting", EventDate: calDay(3).Format("2006-01-02"),
		StartTime: "16:30", Location: "多功能厅",
		Status: domain.CalendarStatusPublished, Notify: true,
	})
	require.NoError(t, err)
	id := out["id"].(uint64)

	// 落库为园级 + class_id 为 NULL（表上有外键，哨兵 0 会撞 FK；NULL 不受外键检查）
	var nullCount int64
	require.NoError(t, f.db.WithContext(ctx).Model(&domain.CalendarEvent{}).
		Where("id = ? AND scope = ? AND class_id IS NULL", id, domain.CalendarScopeKindergarten).
		Count(&nullCount).Error)
	require.EqualValues(t, 1, nullCount)

	// 全园通知：目标集合覆盖两班
	var notice domain.Notice
	require.NoError(t, f.db.WithContext(ctx).
		Where("kindergarten_id = ? AND title = ?", testCalKG, "园级春季家长会").First(&notice).Error)
	var targetCount int64
	require.NoError(t, f.db.WithContext(ctx).Model(&domain.NoticeTarget{}).
		Where("notice_id = ?", notice.ID).Count(&targetCount).Error)
	require.EqualValues(t, 2, targetCount, "园级通知应覆盖本园全部在读班级")

	// 两班家长均可见（全园受众）
	for name, pc := range map[string]*ParentPortalContext{"parentA": f.pcA, "parentB": f.pcB} {
		pOut, err := f.svcP.ListClassCalendar(ctx, pc, "", "")
		require.NoError(t, err, name)
		require.NotNil(t, findCalEvent(calList(t, pOut), id), "%s 应可见园级事件", name)
	}
}

// TC-NEW-CAL-0017 支撑：老师路径创建只能产生班级级事件（园级入口在管理端/医生端，老师无处可入）。
func TestCalendar_TeacherCreateIsClassScopeOnly(t *testing.T) {
	f := newCalFixture(t)
	ctx := context.Background()

	out, err := f.svcT.CreateCalendarEvent(ctx, f.tcA, TeacherCalendarInput{
		Title: "老师的班级事件", EventType: "activity", EventDate: calDay(1).Format("2006-01-02"),
		Status: domain.CalendarStatusPublished, Notify: false,
	})
	require.NoError(t, err)
	id := out["id"].(uint64)

	var row domain.CalendarEvent
	require.NoError(t, f.db.WithContext(ctx).First(&row, id).Error)
	require.Equal(t, domain.CalendarScopeClass, row.Scope)
	require.Equal(t, testCalClassA, row.ClassID)
}

// 4.3：园长角色播种——4 个角色就位且各挂日历权限码（calendar:view + calendar:manage）；重复执行幂等。
// 注意：播种是全园所扫描（与字典播种同语义），本测试同时会把真实园所补齐（预期行为）。
func TestSeedPrincipalRoles_Idempotent(t *testing.T) {
	f := newCalFixture(t)
	ctx := context.Background()

	require.NoError(t, SeedPrincipalRolesForAll(ctx, f.db))
	require.NoError(t, SeedPrincipalRolesForAll(ctx, f.db)) // 幂等：重跑不新增

	var roles []domain.AdminRole
	require.NoError(t, f.db.WithContext(ctx).
		Where("kindergarten_id = ? AND name IN ?", testCalKG, principalRoleNames).
		Find(&roles).Error)
	require.Len(t, roles, 4, "四个园长角色应各恰好一条")

	for _, r := range roles {
		var bindingCount int64
		require.NoError(t, f.db.WithContext(ctx).Model(&domain.AdminRolePermission{}).
			Where("role_id = ?", r.ID).Count(&bindingCount).Error)
		require.EqualValues(t, 2, bindingCount, "角色 %s 应挂 calendar:view + calendar:manage", r.Name)
	}
}
