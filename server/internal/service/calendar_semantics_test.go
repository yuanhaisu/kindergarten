package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// 这些测试打真实 MySQL，覆盖 align-class-calendar 的 class-calendar spec 场景：
// TC-NEW-CAL-0004~0011、0013~0015（园级可见与只读、草稿可见性、推送真链路、变更要旨与窗口）。
// ID 落在 9903xx 区间，t.Cleanup 硬删。
// 对应 spec: openspec/changes/align-class-calendar/specs/class-calendar/spec.md

const (
	testCalKG       uint64 = 990301
	testCalTeacherA uint64 = 990311
	testCalTeacherB uint64 = 990312
	testCalParentA  uint64 = 990321
	testCalParentB  uint64 = 990322
	testCalClassA   uint64 = 990302
	testCalClassB   uint64 = 990303
	testCalStudentA uint64 = 990331
	testCalStudentB uint64 = 990332
)

var calDB *gorm.DB

func openCalTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	if calDB != nil {
		return calDB
	}
	cfgPath := filepath.Join("..", "..", "configs", "app.ini")
	if _, err := os.Stat(cfgPath); err != nil {
		t.Skipf("跳过：找不到 %s（本地无 DB 配置）", cfgPath)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.MySQL.DSN == "" {
		t.Skip("跳过：MySQL DSN 为空")
	}
	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	calDB = db
	return db
}

func newCalCleanup(t *testing.T, db *gorm.DB) {
	t.Helper()
	ctx := context.Background()
	del := func(sql string, args ...any) {
		if err := db.WithContext(ctx).Exec(sql, args...).Error; err != nil {
			t.Logf("cleanup: %v", err)
		}
	}
	del("DELETE FROM notice_read_logs WHERE notice_id IN (SELECT id FROM notices WHERE kindergarten_id = ?)", testCalKG)
	del("DELETE FROM notice_targets WHERE kindergarten_id = ?", testCalKG)
	del("DELETE FROM notices WHERE kindergarten_id = ?", testCalKG)
	del("DELETE FROM calendar_events WHERE kindergarten_id = ?", testCalKG)
	del("DELETE FROM admin_role_permissions WHERE role_id IN (SELECT id FROM admin_roles WHERE kindergarten_id = ?)", testCalKG)
	del("DELETE FROM admin_roles WHERE kindergarten_id = ?", testCalKG)
	del("DELETE FROM timetable_templates WHERE kindergarten_id = ?", testCalKG)
	del("DELETE FROM timetable_overrides WHERE kindergarten_id = ?", testCalKG)
	del("DELETE FROM kindergarten_app_settings WHERE kindergarten_id = ?", testCalKG)
	del("DELETE FROM student_parents WHERE user_id >= 990300 AND user_id < 990400")
	del("DELETE FROM students WHERE kindergarten_id = ?", testCalKG)
	del("DELETE FROM class_teachers WHERE user_id >= 990300 AND user_id < 990400")
	del("DELETE FROM classes WHERE id >= 990300 AND id < 990400")
	del("DELETE FROM sys_dict_items WHERE type_id IN (SELECT id FROM sys_dict_types WHERE kindergarten_id = ?)", testCalKG)
	del("DELETE FROM sys_dict_types WHERE kindergarten_id = ?", testCalKG)
}

func calDay(offsetDays int) time.Time {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	if loc == nil {
		loc = time.Local
	}
	n := time.Now().In(loc)
	return time.Date(n.Year(), n.Month(), n.Day()+offsetDays, 0, 0, 0, 0, loc)
}

// seedCalDict 为测试园所播种日历类型字典（校验「渲染取自字典」的断言基础）。
func seedCalDict(t *testing.T, db *gorm.DB) {
	t.Helper()
	ctx := context.Background()
	typ := &domain.SysDictType{
		KindergartenID: testCalKG, TypeKey: "calendar_event_type",
		TypeName: "日历事件类型", IsSystem: true,
	}
	require.NoError(t, db.WithContext(ctx).Create(typ).Error)
	items := []domain.SysDictItem{
		{TypeID: typ.ID, ItemKey: "notice", ItemLabel: "通知", Extra: `{"color":"#3B82F6"}`, SortOrder: 1, IsEnabled: true},
		{TypeID: typ.ID, ItemKey: "activity", ItemLabel: "活动", Extra: `{"color":"#22C55E"}`, SortOrder: 2, IsEnabled: true},
		{TypeID: typ.ID, ItemKey: "holiday", ItemLabel: "假期", Extra: `{"color":"#F97316"}`, SortOrder: 3, IsEnabled: true},
		{TypeID: typ.ID, ItemKey: "meeting", ItemLabel: "家长会", Extra: `{"color":"#A855F7"}`, SortOrder: 4, IsEnabled: true},
		{TypeID: typ.ID, ItemKey: "checkup", ItemLabel: "体检", Extra: `{"color":"#EF4444"}`, SortOrder: 5, IsEnabled: true},
	}
	for i := range items {
		require.NoError(t, db.WithContext(ctx).Create(&items[i]).Error)
	}
}

type calFixture struct {
	db   *gorm.DB
	svcT *TeacherPortalService
	svcP *ParentPortalService
	tcA  *TeacherPortalContext
	tcB  *TeacherPortalContext
	pcA  *ParentPortalContext
	pcB  *ParentPortalContext
}

func newCalFixture(t *testing.T) *calFixture {
	t.Helper()
	db := openCalTestDB(t)
	ctx := context.Background()
	validDate := time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC)

	newCalCleanup(t, db)

	var kgCount int64
	db.WithContext(ctx).Model(&domain.Kindergarten{}).Where("id = ?", testCalKG).Count(&kgCount)
	if kgCount == 0 {
		require.NoError(t, db.WithContext(ctx).Create(&domain.Kindergarten{
			BaseModel: domain.BaseModel{ID: testCalKG}, Name: "测试园所-日历语义",
		}).Error)
	}
	for _, u := range []struct {
		id          uint64
		phone, nick string
	}{
		{testCalTeacherA, "13900003101", "测试老师A"},
		{testCalTeacherB, "13900003102", "测试老师B"},
		{testCalParentA, "13900003121", "测试家长A"},
		{testCalParentB, "13900003122", "测试家长B"},
	} {
		var c int64
		db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", u.id).Count(&c)
		if c == 0 {
			require.NoError(t, db.WithContext(ctx).Create(&domain.User{
				BaseModel: domain.BaseModel{ID: u.id}, Phone: u.phone, Nickname: u.nick,
			}).Error)
		}
	}
	for _, cl := range []struct {
		id    uint64
		name  string
		grade string
	}{
		{testCalClassA, "测试小一班", "小班"},
		{testCalClassB, "测试小二班", "小班"},
	} {
		require.NoError(t, db.WithContext(ctx).Create(&domain.Class{
			BaseModel: domain.BaseModel{ID: cl.id}, KindergartenID: testCalKG,
			Name: cl.name, Grade: cl.grade, Status: 1,
		}).Error)
	}
	// 老师各自的任教行（推送目标校验用）
	require.NoError(t, db.WithContext(ctx).Create(&domain.ClassTeacher{ClassID: testCalClassA, UserID: testCalTeacherA, Title: "teacher", Status: 1}).Error)
	require.NoError(t, db.WithContext(ctx).Create(&domain.ClassTeacher{ClassID: testCalClassB, UserID: testCalTeacherB, Title: "teacher", Status: 1}).Error)
	// 学生与家长绑定
	for _, st := range []struct {
		id      uint64
		classID uint64
		parent  uint64
		name    string
	}{
		{testCalStudentA, testCalClassA, testCalParentA, "测试学生A"},
		{testCalStudentB, testCalClassB, testCalParentB, "测试学生B"},
	} {
		require.NoError(t, db.WithContext(ctx).Create(&domain.Student{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: st.id}},
			KindergartenID:  testCalKG, ClassID: st.classID,
			Name: st.name, Birthday: validDate, EnrollDate: validDate, Status: 1,
		}).Error)
		require.NoError(t, db.WithContext(ctx).Create(&domain.StudentParent{
			StudentID: st.id, UserID: st.parent, Relation: "mother",
		}).Error)
	}
	seedCalDict(t, db)

	notices := repository.NewNoticeRepo(db)
	f := &calFixture{db: db}
	f.svcT = NewTeacherPortalService(db, repository.NewCalendarRepo(db), repository.NewPickupRepo(db), repository.NewMessageRepo(db), notices, nil, config.FaceConfig{})
	f.svcP = NewParentPortalService(db, repository.NewParentRepo(db), repository.NewUserRoleRepo(db),
		repository.NewStudentRepo(db), repository.NewClassRepo(db), repository.NewPickupRepo(db),
		repository.NewCalendarRepo(db), notices)
	f.tcA = &TeacherPortalContext{UserID: testCalTeacherA, KindergartenID: testCalKG, ClassID: testCalClassA}
	f.tcB = &TeacherPortalContext{UserID: testCalTeacherB, KindergartenID: testCalKG, ClassID: testCalClassB}
	stuA := domain.Student{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testCalStudentA}}, ClassID: testCalClassA}
	stuB := domain.Student{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testCalStudentB}}, ClassID: testCalClassB}
	f.pcA = &ParentPortalContext{UserID: testCalParentA, KindergartenID: testCalKG, CurrentStudent: &stuA, Students: []domain.Student{stuA}}
	f.pcB = &ParentPortalContext{UserID: testCalParentB, KindergartenID: testCalKG, CurrentStudent: &stuB, Students: []domain.Student{stuB}}

	t.Cleanup(func() { newCalCleanup(t, db) })
	return f
}

func calList(t *testing.T, out map[string]any) []map[string]any {
	t.Helper()
	list, ok := out["list"].([]map[string]any)
	require.True(t, ok, "list 形状不符: %T", out["list"])
	return list
}

func findCalEvent(list []map[string]any, id uint64) map[string]any {
	for _, m := range list {
		if v, ok := m["id"].(uint64); ok && v == id {
			return m
		}
	}
	return nil
}

// TC-NEW-CAL-0004 + 0006：园级事件全园（两班老师与家长）可见；老师对园级事件只读。
func TestCalendar_GardenEvent_FullVisibility_TeacherReadOnly(t *testing.T) {
	f := newCalFixture(t)
	ctx := context.Background()

	garden := &domain.CalendarEvent{
		KindergartenID: testCalKG, ClassID: 0, CreatorID: testCalTeacherA,
		Title: "园级清明节假期", EventType: domain.EventTypeHoliday,
		Scope: domain.CalendarScopeKindergarten, Status: domain.CalendarStatusPublished,
		Source:    domain.CalendarSourceManual,
		EventDate: calDay(2), IsAllDay: true,
	}
	require.NoError(t, f.db.WithContext(ctx).Create(garden).Error)

	for name, tc := range map[string]*TeacherPortalContext{"teacherA": f.tcA, "teacherB": f.tcB} {
		out, err := f.svcT.CalendarEvents(ctx, tc, "", "")
		require.NoError(t, err, name)
		got := findCalEvent(calList(t, out), garden.ID)
		require.NotNil(t, got, "%s 应能看到园级事件", name)
		require.Equal(t, domain.CalendarScopeKindergarten, got["scope"])
	}
	for name, pc := range map[string]*ParentPortalContext{"parentA": f.pcA, "parentB": f.pcB} {
		out, err := f.svcP.ListClassCalendar(ctx, pc, "", "")
		require.NoError(t, err, name)
		require.NotNil(t, findCalEvent(calList(t, out), garden.ID), "%s 应能看到园级事件", name)
	}

	// 老师对园级事件只读（TC-NEW-CAL-0006）
	err := f.svcT.UpdateCalendarEvent(ctx, f.tcA, garden.ID, TeacherCalendarInput{Title: "改标题", Status: domain.CalendarStatusPublished})
	require.ErrorIs(t, err, ErrTeacherForbidden)
	require.ErrorIs(t, f.svcT.DeleteCalendarEvent(ctx, f.tcA, garden.ID), ErrTeacherForbidden)
}

// TC-NEW-CAL-0007/0008/0009：草稿对家长不可见、老师可见并带标记；发布后可见；回退草稿后不可见。
func TestCalendar_DraftVisibility_Toggles(t *testing.T) {
	f := newCalFixture(t)
	ctx := context.Background()
	date := calDay(1).Format("2006-01-02")

	out, err := f.svcT.CreateCalendarEvent(ctx, f.tcA, TeacherCalendarInput{
		Title: "美术课：春天的小树", EventType: "activity", EventDate: date,
		StartTime: "10:00", EndTime: "11:00", Location: "教室",
		Status: domain.CalendarStatusDraft, Notify: false,
	})
	require.NoError(t, err)
	id := out["id"].(uint64)

	// 草稿：家长不可见，老师可见且有草稿标记
	pOut, err := f.svcP.ListClassCalendar(ctx, f.pcA, "", "")
	require.NoError(t, err)
	require.Nil(t, findCalEvent(calList(t, pOut), id), "草稿不应出现在家长端")
	tOut, err := f.svcT.CalendarEvents(ctx, f.tcA, "", "")
	require.NoError(t, err)
	got := findCalEvent(calList(t, tOut), id)
	require.NotNil(t, got)
	require.Equal(t, domain.CalendarStatusDraft, got["status"])

	// 发布后家长可见，且标签/颜色取自字典（TC-NEW-CAL-0008 + 字典驱动）
	require.NoError(t, f.svcT.UpdateCalendarEvent(ctx, f.tcA, id, TeacherCalendarInput{
		Title: "美术课：春天的小树", EventType: "activity", EventDate: date,
		StartTime: "10:00", EndTime: "11:00", Location: "教室",
		Status: domain.CalendarStatusPublished, Notify: false,
	}))
	pOut, err = f.svcP.ListClassCalendar(ctx, f.pcA, "", "")
	require.NoError(t, err)
	got = findCalEvent(calList(t, pOut), id)
	require.NotNil(t, got, "发布后家长应可见")
	require.Equal(t, "活动", got["event_type_label"])
	require.Equal(t, "#22C55E", got["color"])

	// 回退草稿：家长端不再可见（TC-NEW-CAL-0009）
	require.NoError(t, f.svcT.UpdateCalendarEvent(ctx, f.tcA, id, TeacherCalendarInput{
		Title: "美术课：春天的小树", EventType: "activity", EventDate: date,
		Status: domain.CalendarStatusDraft, Notify: false,
	}))
	pOut, err = f.svcP.ListClassCalendar(ctx, f.pcA, "", "")
	require.NoError(t, err)
	require.Nil(t, findCalEvent(calList(t, pOut), id), "回退草稿后家长端不应可见")
}

// TC-NEW-CAL-0010/0011：确认通知 → 生成面向本班家长的通知并回写记录位；不通知 → 事件照常生效但无通知。
func TestCalendar_Notify_CreatesNotice_OrNot(t *testing.T) {
	f := newCalFixture(t)
	ctx := context.Background()
	date := calDay(1).Format("2006-01-02")

	out, err := f.svcT.CreateCalendarEvent(ctx, f.tcA, TeacherCalendarInput{
		Title: "春季出游活动", EventType: "activity", EventDate: date,
		StartTime: "09:00", Location: "城市公园",
		Status: domain.CalendarStatusPublished, Notify: true,
	})
	require.NoError(t, err)
	id := out["id"].(uint64)

	var notice domain.Notice
	require.NoError(t, f.db.WithContext(ctx).
		Where("kindergarten_id = ? AND title = ?", testCalKG, "春季出游活动").First(&notice).Error)
	var targetCount int64
	require.NoError(t, f.db.WithContext(ctx).Model(&domain.NoticeTarget{}).
		Where("notice_id = ? AND class_id = ?", notice.ID, testCalClassA).Count(&targetCount).Error)
	require.EqualValues(t, 1, targetCount, "通知目标应命中本班")

	var row domain.CalendarEvent
	require.NoError(t, f.db.WithContext(ctx).First(&row, id).Error)
	require.True(t, row.IsPublished, "推送成功后应回写 is_published")
	require.NotNil(t, row.NoticeID, "推送成功后应回写 notice_id")

	// 不通知：事件发布生效（家长端可见），无通知生成（TC-NEW-CAL-0011）
	out2, err := f.svcT.CreateCalendarEvent(ctx, f.tcA, TeacherCalendarInput{
		Title: "不通知的活动", EventType: "activity", EventDate: date,
		Status: domain.CalendarStatusPublished, Notify: false,
	})
	require.NoError(t, err)
	id2 := out2["id"].(uint64)
	var c2 int64
	require.NoError(t, f.db.WithContext(ctx).Model(&domain.Notice{}).
		Where("kindergarten_id = ? AND title = ?", testCalKG, "不通知的活动").Count(&c2).Error)
	require.EqualValues(t, 0, c2, "未确认通知不应生成通知")
	pOut, err := f.svcP.ListClassCalendar(ctx, f.pcA, "", "")
	require.NoError(t, err)
	require.NotNil(t, findCalEvent(calList(t, pOut), id2), "不通知但已发布的事件家长端应可见")
}

// TC-NEW-CAL-0013/0014/0015：变更保存时确认通知 → 变更通知 + 变更要旨；家长端提示窗口内非空、无变更为空。
func TestCalendar_ChangeSummary_And_Window(t *testing.T) {
	f := newCalFixture(t)
	ctx := context.Background()
	date := calDay(3).Format("2006-01-02")

	out, err := f.svcT.CreateCalendarEvent(ctx, f.tcA, TeacherCalendarInput{
		Title: "春游活动", EventType: "activity", EventDate: date,
		StartTime: "10:00", Location: "城市公园",
		Status: domain.CalendarStatusPublished, Notify: false,
	})
	require.NoError(t, err)
	id := out["id"].(uint64)

	// 无变更不产生要旨（TC-NEW-CAL-0015）
	require.NoError(t, f.svcT.UpdateCalendarEvent(ctx, f.tcA, id, TeacherCalendarInput{
		Title: "春游活动", EventType: "activity", EventDate: date,
		StartTime: "10:00", Location: "城市公园",
		Status: domain.CalendarStatusPublished, Notify: false,
	}))
	var row domain.CalendarEvent
	require.NoError(t, f.db.WithContext(ctx).First(&row, id).Error)
	require.Empty(t, row.ChangeSummary)
	require.Nil(t, row.RevisedAt)

	// 时间变更 + 确认通知（TC-NEW-CAL-0013/0014）
	require.NoError(t, f.svcT.UpdateCalendarEvent(ctx, f.tcA, id, TeacherCalendarInput{
		Title: "春游活动", EventType: "activity", EventDate: date,
		StartTime: "10:30", Location: "城市公园",
		Status: domain.CalendarStatusPublished, Notify: true,
	}))
	require.NoError(t, f.db.WithContext(ctx).First(&row, id).Error)
	require.Equal(t, "时间 10:00 → 10:30", row.ChangeSummary)
	require.NotNil(t, row.RevisedAt)

	var changeNotice domain.Notice
	require.NoError(t, f.db.WithContext(ctx).
		Where("kindergarten_id = ? AND title = ?", testCalKG, "【变更】春游活动").First(&changeNotice).Error)

	// 家长端：未来事件 + 近期修订 → 变更要旨非空（banner 数据源）
	pOut, err := f.svcP.ListClassCalendar(ctx, f.pcA, "", "")
	require.NoError(t, err)
	got := findCalEvent(calList(t, pOut), id)
	require.NotNil(t, got)
	require.Equal(t, "时间 10:00 → 10:30", got["change_summary"])

	// 窗口边界：事件日期已过 → 不再下发变更要旨（提示只对「尚未发生」有意义）
	require.NoError(t, f.db.WithContext(ctx).Model(&domain.CalendarEvent{}).
		Where("id = ?", id).Update("event_date", calDay(-1)).Error)
	pOut, err = f.svcP.ListClassCalendar(ctx, f.pcA, "", "")
	require.NoError(t, err)
	got = findCalEvent(calList(t, pOut), id)
	require.NotNil(t, got)
	require.Nil(t, got["change_summary"], "已发生的事件不再下发变更要旨")
}

// TC-NEW-CAL-0005：班级事件跨班不可见（老师与家长两侧）。
func TestCalendar_CrossClassInvisible(t *testing.T) {
	f := newCalFixture(t)
	ctx := context.Background()

	out, err := f.svcT.CreateCalendarEvent(ctx, f.tcA, TeacherCalendarInput{
		Title: "小一班的班级活动", EventType: "activity", EventDate: calDay(1).Format("2006-01-02"),
		Status: domain.CalendarStatusPublished, Notify: false,
	})
	require.NoError(t, err)
	id := out["id"].(uint64)

	tOut, err := f.svcT.CalendarEvents(ctx, f.tcB, "", "")
	require.NoError(t, err)
	require.Nil(t, findCalEvent(calList(t, tOut), id), "他班老师不应看到班级事件")

	pOut, err := f.svcP.ListClassCalendar(ctx, f.pcB, "", "")
	require.NoError(t, err)
	require.Nil(t, findCalEvent(calList(t, pOut), id), "他班家长不应看到班级事件")
}

// TC-NEW-CAL-0001 支撑：类型字典经既有 teacher dicts 端点路径可读（含 Extra 颜色元数据）。
// 端点 `GET /teacher/dicts/:type_key` 的 handler → DictService.GetDictItems 即本调用。
func TestCalendar_DictExposure_ViaTeacherDictPath(t *testing.T) {
	f := newCalFixture(t)
	ctx := context.Background()

	svc := NewDictService(f.db, repository.NewDictRepo(f.db))
	resp, err := svc.GetDictItems(ctx, testCalKG, "calendar_event_type")
	require.NoError(t, err)
	require.Len(t, resp.Items, 5)
	byKey := make(map[string]DictItemPublic, len(resp.Items))
	for _, it := range resp.Items {
		byKey[it.ItemKey] = it
	}
	require.Equal(t, "活动", byKey["activity"].ItemLabel)
	require.Contains(t, string(byKey["activity"].Extra), "#22C55E")
	require.Equal(t, "体检", byKey["checkup"].ItemLabel)
	require.Contains(t, string(byKey["checkup"].Extra), "#EF4444")
}
