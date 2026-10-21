package service

// 这些测试打真实 MySQL，覆盖 align-class-calendar 的 class-timetable spec 场景：
// TC-NEW-TBL-0001~0010（开关、模板+微调、物化+滚动、覆盖保护、模板变更边界）。
// 复用 calendar_semantics_test.go 的 calFixture（9903xx 区间，t.Cleanup 硬删）。

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

func newTimetableSvc(f *calFixture) *TimetableService {
	return NewTimetableService(f.db, repository.NewCalendarRepo(f.db),
		repository.NewTimetableRepo(f.db), repository.NewKindergartenAppSettingRepo(f.db))
}

// countWeekdayInWindow 滚动窗口（[今天, 今天+28) 天）内某星期几的出现次数。
func countWeekdayInWindow(wd int) int {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	if loc == nil {
		loc = time.Local
	}
	n := time.Now().In(loc)
	start := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, loc)
	count := 0
	for d := 0; d < timetableWindowDays; d++ {
		if isoWeekday(start.AddDate(0, 0, d)) == wd {
			count++
		}
	}
	return count
}

// TC-NEW-TBL-0002/0003/0005/0006/0007：模板生效到年级各班、物化完整滚动窗口、幂等、不产生通知。
func TestTimetable_SyncMaterializesAndIdempotent(t *testing.T) {
	f := newCalFixture(t)
	ctx := context.Background()
	ts := newTimetableSvc(f)

	// 维护模板（TBL-0002：开关开启后可维护）——小班年级：每周三第 2 节 美术课
	_, err := ts.SaveTemplate(ctx, testCalKG, "小班", []TimetableEntryInput{
		{Weekday: 3, Period: 2, Title: "美术课", Location: "美术教室", StartTime: "10:00", EndTime: "10:40"},
	})
	require.NoError(t, err)

	// 开启课表（触发同步）
	_, err = ts.UpdateSetting(ctx, testCalKG, true)
	require.NoError(t, err)

	// TBL-0003/0005：两个小班班级都获得完整窗口的周三实例
	wednesdays := countWeekdayInWindow(3)
	require.EqualValues(t, 4, wednesdays, "28 天窗口内应有 4 个周三")
	for _, classID := range []uint64{testCalClassA, testCalClassB} {
		var n int64
		require.NoError(t, f.db.WithContext(ctx).Model(&domain.CalendarEvent{}).
			Where("class_id = ? AND source = ? AND source_slot = ?", classID, domain.CalendarSourceTimetable, "3-2").
			Count(&n).Error)
		require.EqualValues(t, wednesdays, n, "class %d 应有完整窗口的周三实例", classID)
	}

	// 系统物化行：creator 为 NULL、已发布
	var one domain.CalendarEvent
	require.NoError(t, f.db.WithContext(ctx).
		Where("class_id = ? AND source = ?", testCalClassA, domain.CalendarSourceTimetable).
		First(&one).Error)
	require.Equal(t, domain.CalendarStatusPublished, one.Status)
	var nullCreator int64
	require.NoError(t, f.db.WithContext(ctx).Model(&domain.CalendarEvent{}).
		Where("id = ? AND creator_id IS NULL", one.ID).Count(&nullCreator).Error)
	require.EqualValues(t, 1, nullCreator)

	// TBL-0007：同步不产生任何通知
	var notices int64
	require.NoError(t, f.db.WithContext(ctx).Model(&domain.Notice{}).
		Where("kindergarten_id = ?", testCalKG).Count(&notices).Error)
	require.EqualValues(t, 0, notices)

	// TBL-0006 幂等：重跑无新增（5.4 手动/定时同一路径）
	stats, err := ts.SyncKindergarten(ctx, testCalKG)
	require.NoError(t, err)
	require.EqualValues(t, 0, stats["created"])
}

// TC-NEW-TBL-0004：班级微调覆盖模板（只影响本班）。
func TestTimetable_ClassOverrideWins(t *testing.T) {
	f := newCalFixture(t)
	ctx := context.Background()
	ts := newTimetableSvc(f)

	_, err := ts.SaveTemplate(ctx, testCalKG, "小班", []TimetableEntryInput{
		{Weekday: 3, Period: 2, Title: "美术课", Location: "美术教室"},
	})
	require.NoError(t, err)
	_, err = ts.SaveClassOverrides(ctx, testCalKG, testCalClassA, []TimetableOverrideInput{
		{Weekday: 3, Period: 2, Action: "replace", Title: "户外活动", Location: "操场"},
	})
	require.NoError(t, err)
	_, err = ts.UpdateSetting(ctx, testCalKG, true)
	require.NoError(t, err)

	var aRow domain.CalendarEvent
	require.NoError(t, f.db.WithContext(ctx).
		Where("class_id = ? AND source_slot = ?", testCalClassA, "3-2").
		Order("event_date ASC").First(&aRow).Error)
	require.Equal(t, "户外活动", aRow.Title)

	var bRow domain.CalendarEvent
	require.NoError(t, f.db.WithContext(ctx).
		Where("class_id = ? AND source_slot = ?", testCalClassB, "3-2").
		Order("event_date ASC").First(&bRow).Error)
	require.Equal(t, "美术课", bRow.Title)
}

// TC-NEW-TBL-0008/0009：老师调整/取消的实例不被后续同步冲掉/复活。
func TestTimetable_TeacherOverrideProtected(t *testing.T) {
	f := newCalFixture(t)
	ctx := context.Background()
	ts := newTimetableSvc(f)

	_, err := ts.SaveTemplate(ctx, testCalKG, "小班", []TimetableEntryInput{
		{Weekday: 3, Period: 2, Title: "美术课"},
	})
	require.NoError(t, err)
	_, err = ts.UpdateSetting(ctx, testCalKG, true)
	require.NoError(t, err)

	var instances []domain.CalendarEvent
	require.NoError(t, f.db.WithContext(ctx).
		Where("class_id = ? AND source = ?", testCalClassA, domain.CalendarSourceTimetable).
		Order("event_date ASC").Find(&instances).Error)
	require.GreaterOrEqual(t, len(instances), 2)
	edited := instances[0]
	cancelled := instances[1]

	// 老师调整 → overridden=true；同步后保留（TBL-0008）
	require.NoError(t, f.svcT.UpdateCalendarEvent(ctx, f.tcA, edited.ID, TeacherCalendarInput{
		Title: "户外游戏（老师调）", EventType: "activity",
		EventDate: edited.EventDate.Format("2006-01-02"),
		StartTime: "10:00", EndTime: "10:40",
		Status: domain.CalendarStatusPublished, Notify: false,
	}))
	var afterEdit domain.CalendarEvent
	require.NoError(t, f.db.WithContext(ctx).First(&afterEdit, edited.ID).Error)
	require.True(t, afterEdit.Overridden, "老师调整后应标记覆盖")

	_, err = ts.SyncKindergarten(ctx, testCalKG)
	require.NoError(t, err)
	require.NoError(t, f.db.WithContext(ctx).First(&afterEdit, edited.ID).Error)
	require.Equal(t, "户外游戏（老师调）", afterEdit.Title, "同步不应冲掉老师的调整")

	// 老师取消 → 软删 + 覆盖标记；同步不复活（TBL-0009）
	require.NoError(t, f.svcT.DeleteCalendarEvent(ctx, f.tcA, cancelled.ID))
	var gone domain.CalendarEvent
	err = f.db.WithContext(ctx).First(&gone, cancelled.ID).Error
	require.Error(t, err, "取消后活体查询应未命中")
	var deadRow domain.CalendarEvent
	require.NoError(t, f.db.WithContext(ctx).Unscoped().First(&deadRow, cancelled.ID).Error)
	require.True(t, deadRow.Overridden, "取消后软删行应保留覆盖标记（占位）")

	_, err = ts.SyncKindergarten(ctx, testCalKG)
	require.NoError(t, err)
	var liveAfter int64
	require.NoError(t, f.db.WithContext(ctx).Model(&domain.CalendarEvent{}).
		Where("class_id = ? AND event_date = ? AND source_slot = ?", testCalClassA, cancelled.EventDate, "3-2").
		Count(&liveAfter).Error)
	require.EqualValues(t, 0, liveAfter, "取消的实例不应被同步复活")
}

// TC-NEW-TBL-0010：模板变更只影响未来未覆盖实例；已覆盖与过去实例不变。
func TestTimetable_TemplateChangeBoundary(t *testing.T) {
	f := newCalFixture(t)
	ctx := context.Background()
	ts := newTimetableSvc(f)

	_, err := ts.SaveTemplate(ctx, testCalKG, "小班", []TimetableEntryInput{
		{Weekday: 3, Period: 2, Title: "美术课"},
	})
	require.NoError(t, err)
	_, err = ts.UpdateSetting(ctx, testCalKG, true)
	require.NoError(t, err)

	// 过去实例（直接造行）：同步只处理 [今天, +窗口)，不应触碰
	past := &domain.CalendarEvent{
		KindergartenID: testCalKG, ClassID: testCalClassA, CreatorID: 0,
		Title: "美术课", EventType: domain.EventTypeActivity,
		Scope: domain.CalendarScopeClass, Status: domain.CalendarStatusPublished,
		Source: domain.CalendarSourceTimetable, SourceSlot: "3-2",
		EventDate: calDay(-1),
	}
	require.NoError(t, f.db.WithContext(ctx).Create(past).Error)

	// 挑一个未来实例标记为已覆盖（模拟老师改过）
	var future domain.CalendarEvent
	require.NoError(t, f.db.WithContext(ctx).
		Where("class_id = ? AND source = ?", testCalClassA, domain.CalendarSourceTimetable).
		Order("event_date DESC").First(&future).Error)
	require.NoError(t, f.db.WithContext(ctx).Model(&domain.CalendarEvent{}).
		Where("id = ?", future.ID).Update("overridden", true).Error)

	// 模板改标题 → 自动重算
	_, err = ts.SaveTemplate(ctx, testCalKG, "小班", []TimetableEntryInput{
		{Weekday: 3, Period: 2, Title: "美术创作"},
	})
	require.NoError(t, err)

	// 未覆盖的未来实例已更新
	var updatedCount int64
	require.NoError(t, f.db.WithContext(ctx).Model(&domain.CalendarEvent{}).
		Where("class_id = ? AND source_slot = ? AND overridden = 0 AND title = ?", testCalClassA, "3-2", "美术创作").
		Count(&updatedCount).Error)
	require.EqualValues(t, countWeekdayInWindow(3)-1, updatedCount, "除已覆盖的一条外，未来实例应全部更新")

	// 已覆盖实例保持旧标题
	var kept domain.CalendarEvent
	require.NoError(t, f.db.WithContext(ctx).First(&kept, future.ID).Error)
	require.Equal(t, "美术课", kept.Title)

	// 过去实例不变
	var pastAfter domain.CalendarEvent
	require.NoError(t, f.db.WithContext(ctx).First(&pastAfter, past.ID).Error)
	require.Equal(t, "美术课", pastAfter.Title)
}

// TC-NEW-TBL-0001：开关关闭隐藏课表来源事件（数据保留），手动事件不受影响；开启恢复。
func TestTimetable_SwitchHidesAndShows(t *testing.T) {
	f := newCalFixture(t)
	ctx := context.Background()
	ts := newTimetableSvc(f)

	_, err := ts.SaveTemplate(ctx, testCalKG, "小班", []TimetableEntryInput{{Weekday: 3, Period: 2, Title: "美术课"}})
	require.NoError(t, err)
	_, err = ts.UpdateSetting(ctx, testCalKG, true)
	require.NoError(t, err)

	var ttInst domain.CalendarEvent
	require.NoError(t, f.db.WithContext(ctx).
		Where("class_id = ? AND source = ?", testCalClassA, domain.CalendarSourceTimetable).
		Order("event_date ASC").First(&ttInst).Error)

	// 手动事件（不受开关影响）。若 calDay(1) 恰好与课表实例同一天，dots 会因为
	// 手动事件仍存在而包含该日期，无法验证「关闭后课表日期消失」；换到次日。
	manualDate := calDay(1)
	if manualDate.Format("2006-01-02") == ttInst.EventDate.Format("2006-01-02") {
		manualDate = calDay(2)
	}
	mOut, err := f.svcT.CreateCalendarEvent(ctx, f.tcA, TeacherCalendarInput{
		Title: "手动活动", EventType: "activity", EventDate: manualDate.Format("2006-01-02"),
		Status: domain.CalendarStatusPublished, Notify: false,
	})
	require.NoError(t, err)
	manualID := mOut["id"].(uint64)

	// 开启状态：老师与家长都能看到课表实例
	tOut, err := f.svcT.CalendarEvents(ctx, f.tcA, "", "")
	require.NoError(t, err)
	require.NotNil(t, findCalEvent(calList(t, tOut), ttInst.ID), "开启时老师端应见课表实例")
	pOut, err := f.svcP.ListClassCalendar(ctx, f.pcA, "", "")
	require.NoError(t, err)
	require.NotNil(t, findCalEvent(calList(t, pOut), ttInst.ID), "开启时家长端应见课表实例")

	// 关闭：课表实例两端不呈现；手动事件照常
	_, err = ts.UpdateSetting(ctx, testCalKG, false)
	require.NoError(t, err)
	tOut, err = f.svcT.CalendarEvents(ctx, f.tcA, "", "")
	require.NoError(t, err)
	require.Nil(t, findCalEvent(calList(t, tOut), ttInst.ID), "关闭时老师端不应见课表实例")
	require.NotNil(t, findCalEvent(calList(t, tOut), manualID), "关闭时手动事件应照常")
	pOut, err = f.svcP.ListClassCalendar(ctx, f.pcA, "", "")
	require.NoError(t, err)
	require.Nil(t, findCalEvent(calList(t, pOut), ttInst.ID), "关闭时家长端不应见课表实例")
	require.NotNil(t, findCalEvent(calList(t, pOut), manualID))

	// dots 同口径：关闭后课表实例日期不再出现在标记点
	dotsOut, err := f.svcT.CalendarDots(ctx, f.tcA, "", "")
	require.NoError(t, err)
	dots, ok := dotsOut["dots"].(map[string][]string)
	require.True(t, ok)
	require.NotContains(t, dots, ttInst.EventDate.Format("2006-01-02"), "关闭时该日期不应有标记点")

	// 数据保留：实例仍在库里（仅不呈现）
	var stillThere int64
	require.NoError(t, f.db.WithContext(ctx).Model(&domain.CalendarEvent{}).
		Where("id = ?", ttInst.ID).Count(&stillThere).Error)
	require.EqualValues(t, 1, stillThere)

	// 重新开启恢复可见
	_, err = ts.UpdateSetting(ctx, testCalKG, true)
	require.NoError(t, err)
	tOut, err = f.svcT.CalendarEvents(ctx, f.tcA, "", "")
	require.NoError(t, err)
	require.NotNil(t, findCalEvent(calList(t, tOut), ttInst.ID), "重新开启后应恢复可见")
}
