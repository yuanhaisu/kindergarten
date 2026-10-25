package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/domain"
)

// 2.2 / 2.3 / 2.5 / 2.6 / 2.7 / 3.1 / 6.2：家长端任务范围与提交约束。
// 全部跑真实 MySQL：范围谓词、唯一键、状态行锁这些语义在内存桩上测不出来。

func (f *b3Fixture) rawTask(
	t *testing.T, classID, target uint64, title, freq string, startOff, endOff int,
) domain.ParentTask {
	t.Helper()
	loc := shanghaiLoc()
	now := time.Now().In(loc)
	day := func(off int) time.Time {
		d := now.AddDate(0, 0, off)
		return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, loc)
	}
	tk := domain.ParentTask{
		KindergartenID: testB3KG, ClassID: classID, CreatorID: testB3Teacher,
		TargetStudentID: target, Title: title, Frequency: domain.TaskFrequency(freq),
		StartDate: day(startOff), EndDate: day(endOff), Status: 1,
		RecognitionRitual: domain.TaskRitualSticker, StarScope: domain.TaskStarScopeTask,
	}
	require.NoError(t, f.db.Create(&tk).Error)
	return tk
}

func (f *b3Fixture) day(off int) time.Time {
	loc := shanghaiLoc()
	d := time.Now().In(loc).AddDate(0, 0, off)
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, loc)
}

func TestParentTaskContextRequiresOwnedStudent(t *testing.T) {
	f := setupB3(t)
	ctx := context.Background()

	// B3 夹具平时直接构造 ParentPortalContext，没有 user_roles 行；
	// ResolveContext 的真实入口需要它，本用例按生产形状补上。
	kg := testB3KG
	for _, uid := range []uint64{testB3ParentA, testB3ParentB, testB3ParentOther} {
		require.NoError(t, f.db.Unscoped().Create(&domain.UserRole{
			UserID: uid, Role: "parent", KindergartenID: &kg,
		}).Error)
		t.Cleanup(func() {
			f.db.Unscoped().Where("user_id = ? AND role = ? AND kindergarten_id = ?", uid, "parent", kg).
				Delete(&domain.UserRole{})
		})
	}

	_, err := f.svcP.ResolveTaskContext(ctx, testB3ParentA, testB3KG, 0)
	assert.ErrorIs(t, err, ErrParentInvalidParams, "缺少 student_id 必须拒绝")

	_, err = f.svcP.ResolveTaskContext(ctx, testB3ParentA, testB3KG, testB3StudentB)
	assert.ErrorIs(t, err, ErrParentForbidden, "不属于该家长的孩子必须拒绝，不能回退到第一个孩子")

	pc, err := f.svcP.ResolveTaskContext(ctx, testB3ParentA, testB3KG, testB3StudentA2)
	require.NoError(t, err)
	assert.Equal(t, testB3StudentA2, pc.CurrentStudent.ID, "多孩切换按请求的孩子解析")
}

func TestParentCheckinRejectsOutOfScopeTasks(t *testing.T) {
	f := setupB3(t)
	ctx := context.Background()

	otherClass := f.rawTask(t, testB3ClassOther, 0, "B3 别班任务", string(domain.TaskFrequencyDaily), 0, 6)
	otherChild := f.rawTask(t, testB3Class, testB3StudentB, "B3 别人孩子的个体任务", string(domain.TaskFrequencyDaily), 0, 6)

	_, err := f.svcP.CheckinTask(ctx, f.pcA1, otherClass.ID, "跨班", 0)
	assert.ErrorIs(t, err, ErrParentNotFound)

	_, err = f.svcP.CheckinTask(ctx, f.pcA1, otherChild.ID, "别人的个体任务", 0)
	assert.ErrorIs(t, err, ErrParentNotFound)

	var n int64
	require.NoError(t, f.db.Model(&domain.TaskCheckIn{}).
		Where("task_id IN ?", []uint64{otherClass.ID, otherChild.ID}).Count(&n).Error)
	assert.Zero(t, n, "被拒绝时不得落库")
}

func TestParentTaskActiveWindow(t *testing.T) {
	f := setupB3(t)
	ctx := context.Background()

	future := f.rawTask(t, testB3Class, 0, "B3 未开始", string(domain.TaskFrequencyDaily), 2, 5)
	past := f.rawTask(t, testB3Class, 0, "B3 已结束", string(domain.TaskFrequencyDaily), -5, -1)
	active := f.rawTask(t, testB3Class, 0, "B3 进行中", string(domain.TaskFrequencyDaily), 0, 6)

	list, err := f.svcP.ListTasks(ctx, f.pcA1, 1, 20)
	require.NoError(t, err)
	ids := map[uint64]bool{}
	for _, item := range list["list"].([]map[string]any) {
		ids[item["id"].(uint64)] = true
	}
	assert.True(t, ids[active.ID], "进行中的任务必须可见")
	assert.False(t, ids[future.ID], "未开始的任务不进列表")
	assert.False(t, ids[past.ID], "已结束的任务不进列表")

	_, err = f.svcP.CheckinTask(ctx, f.pcA1, future.ID, "太早", 0)
	assert.ErrorIs(t, err, ErrParentInvalidParams)
	_, err = f.svcP.CheckinTask(ctx, f.pcA1, past.ID, "太晚", 0)
	assert.ErrorIs(t, err, ErrParentInvalidParams)

	_, err = f.svcP.CheckinTask(ctx, f.pcA1, active.ID, "正好", 0)
	require.NoError(t, err)
}

func TestParentCheckinFrequencyRules(t *testing.T) {
	f := setupB3(t)
	ctx := context.Background()

	// daily：当天第二次被拒（状态行 + 唯一索引兜底）
	daily := f.createTask(t, 0, "B3 每日频率")["id"].(uint64)
	_, err := f.svcP.CheckinTask(ctx, f.pcA1, daily, "第一次", 0)
	require.NoError(t, err)
	_, err = f.svcP.CheckinTask(ctx, f.pcA1, daily, "第二次", 0)
	assert.ErrorIs(t, err, ErrParentInvalidParams)

	// weekly：3 天前已提交（滚动 7 天内）→ 拒绝
	weeklyIn := f.rawTask(t, testB3Class, 0, "B3 每周窗口内", string(domain.TaskFrequencyWeekly), 0, 6)
	threeAgo := f.day(-3)
	require.NoError(t, f.db.Create(&domain.TaskCheckIn{
		TaskID: weeklyIn.ID, StudentID: testB3StudentA1, UserID: testB3ParentA,
		Date: threeAgo, Content: "旧提交",
	}).Error)
	require.NoError(t, f.db.Create(&domain.TaskCheckInState{
		TaskID: weeklyIn.ID, StudentID: testB3StudentA1, LastCheckInDate: &threeAgo,
	}).Error)
	_, err = f.svcP.CheckinTask(ctx, f.pcA1, weeklyIn.ID, "窗口内", 0)
	assert.ErrorIs(t, err, ErrParentInvalidParams)

	// weekly：8 天前已提交（滚动窗口外）→ 允许
	weeklyOut := f.rawTask(t, testB3Class, 0, "B3 每周窗口外", string(domain.TaskFrequencyWeekly), 0, 6)
	eightAgo := f.day(-8)
	require.NoError(t, f.db.Create(&domain.TaskCheckIn{
		TaskID: weeklyOut.ID, StudentID: testB3StudentA1, UserID: testB3ParentA,
		Date: eightAgo, Content: "更早的提交",
	}).Error)
	require.NoError(t, f.db.Create(&domain.TaskCheckInState{
		TaskID: weeklyOut.ID, StudentID: testB3StudentA1, LastCheckInDate: &eightAgo,
	}).Error)
	_, err = f.svcP.CheckinTask(ctx, f.pcA1, weeklyOut.ID, "窗口外", 0)
	require.NoError(t, err)

	// once：周期内第二次被拒
	once := f.rawTask(t, testB3Class, 0, "B3 一次性", string(domain.TaskFrequencyOnce), 0, 6)
	_, err = f.svcP.CheckinTask(ctx, f.pcA1, once.ID, "第一次", 0)
	require.NoError(t, err)
	_, err = f.svcP.CheckinTask(ctx, f.pcA1, once.ID, "第二次", 0)
	assert.ErrorIs(t, err, ErrParentInvalidParams)
}

func TestParentCheckinConcurrentOnlyOneSucceeds(t *testing.T) {
	f := setupB3(t)
	taskID := f.createTask(t, 0, "B3 并发提交")["id"].(uint64)

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = f.svcP.CheckinTask(context.Background(), f.pcA1, taskID, "并发", 0)
		}(i)
	}
	wg.Wait()

	success := 0
	for _, err := range errs {
		if err == nil {
			success++
		}
	}
	assert.Equal(t, 1, success, "并发双击只成功一次")

	var n int64
	require.NoError(t, f.db.Model(&domain.TaskCheckIn{}).
		Where("task_id = ? AND student_id = ?", taskID, testB3StudentA1).Count(&n).Error)
	assert.EqualValues(t, 1, n, "唯一索引兜底：只能落一条")
}

func TestPendingTaskCountOnlyCurrentPeriod(t *testing.T) {
	f := setupB3(t)
	ctx := context.Background()

	daily := f.createTask(t, 0, "B3 待办每日")["id"].(uint64)
	_ = f.rawTask(t, testB3Class, 0, "B3 待办每周", string(domain.TaskFrequencyWeekly), 0, 6)
	_ = f.rawTask(t, testB3Class, 0, "B3 待办一次性", string(domain.TaskFrequencyOnce), 0, 6)
	ended := f.rawTask(t, testB3Class, 0, "B3 待办已结束", string(domain.TaskFrequencyDaily), 0, 6)
	require.NoError(t, f.db.Model(&domain.ParentTask{}).Where("id = ?", ended.ID).Update("status", 0).Error)
	_ = f.rawTask(t, testB3Class, 0, "B3 待办已过期", string(domain.TaskFrequencyDaily), -5, -1)

	pending, err := pendingTaskCount(ctx, f.db, f.pcA1, time.Now().In(shanghaiLoc()))
	require.NoError(t, err)
	assert.EqualValues(t, 3, pending, "daily/weekly/once 各 1；已结束与已过期不计")

	_, err = f.svcP.CheckinTask(ctx, f.pcA1, daily, "完成", 0)
	require.NoError(t, err)
	pending, err = pendingTaskCount(ctx, f.db, f.pcA1, time.Now().In(shanghaiLoc()))
	require.NoError(t, err)
	assert.EqualValues(t, 2, pending, "已提交的每日任务不再计入")
}

func TestParentCheckinValidatesMediaOwnership(t *testing.T) {
	f := setupB3(t)
	ctx := context.Background()

	mkFile := func(userID uint64, bizType, fileType string) uint64 {
		file := domain.File{
			UserID: userID, BizType: bizType, FileType: fileType,
			FileName: "f", FileURL: "https://cdn.example.com/f", FileSize: 1,
		}
		require.NoError(t, f.db.Create(&file).Error)
		t.Cleanup(func() { f.db.Unscoped().Delete(&domain.File{}, file.ID) })
		return file.ID
	}

	okTask := f.createTask(t, 0, "B3 媒体合法")["id"].(uint64)
	_, err := f.svcP.CheckinTask(ctx, f.pcA1, okTask, "照片", mkFile(testB3ParentA, "task", "image"))
	require.NoError(t, err)

	foreignTask := f.createTask(t, 0, "B3 媒体别人的")["id"].(uint64)
	_, err = f.svcP.CheckinTask(ctx, f.pcA1, foreignTask, "照片", mkFile(testB3ParentB, "task", "image"))
	assert.ErrorIs(t, err, ErrParentInvalidParams, "不能用别人的文件")

	wrongBizTask := f.createTask(t, 0, "B3 媒体错业务")["id"].(uint64)
	_, err = f.svcP.CheckinTask(ctx, f.pcA1, wrongBizTask, "照片", mkFile(testB3ParentA, "growth", "image"))
	assert.ErrorIs(t, err, ErrParentInvalidParams, "业务类型必须是 task")

	wrongTypeTask := f.createTask(t, 0, "B3 媒体错类型")["id"].(uint64)
	_, err = f.svcP.CheckinTask(ctx, f.pcA1, wrongTypeTask, "文件", mkFile(testB3ParentA, "task", "pdf"))
	assert.ErrorIs(t, err, ErrParentInvalidParams, "只接受 image/audio")

	audioTask := f.createTask(t, 0, "B3 媒体语音")["id"].(uint64)
	_, err = f.svcP.CheckinTask(ctx, f.pcA1, audioTask, "语音", mkFile(testB3ParentA, "task", "audio"))
	require.NoError(t, err)
}

func TestParentCheckinDoesNotChangePoints(t *testing.T) {
	f := setupB3(t)
	ctx := context.Background()

	require.NoError(t, f.db.Create(&domain.PointAccount{
		StudentID: testB3StudentA1, TotalPoints: 10, UsedPoints: 2,
	}).Error)
	require.NoError(t, f.db.Create(&domain.PointTransaction{
		StudentID: testB3StudentA1, Points: 10, Balance: 10, Source: "growth",
	}).Error)
	t.Cleanup(func() {
		f.db.Unscoped().Where("student_id = ?", testB3StudentA1).Delete(&domain.PointAccount{})
		f.db.Unscoped().Where("student_id = ?", testB3StudentA1).Delete(&domain.PointTransaction{})
	})

	taskID := f.createTask(t, 0, "B3 积分不变")["id"].(uint64)
	_, err := f.svcP.CheckinTask(ctx, f.pcA1, taskID, "完成", 0)
	require.NoError(t, err)

	var account domain.PointAccount
	require.NoError(t, f.db.Where("student_id = ?", testB3StudentA1).First(&account).Error)
	assert.Equal(t, 10, account.TotalPoints, "提交不得改积分账户")
	assert.Equal(t, 2, account.UsedPoints)

	var n int64
	require.NoError(t, f.db.Model(&domain.PointTransaction{}).
		Where("student_id = ?", testB3StudentA1).Count(&n).Error)
	assert.EqualValues(t, 1, n, "提交不得新增积分流水")
}
