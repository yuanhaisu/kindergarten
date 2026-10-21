package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"kindergarten-server/internal/domain"
)

func TestValidTaskFrequency(t *testing.T) {
	assert.True(t, validTaskFrequency(domain.TaskFrequencyDaily))
	assert.True(t, validTaskFrequency(domain.TaskFrequencyWeekly))
	assert.True(t, validTaskFrequency(domain.TaskFrequencyOnce))
	assert.False(t, validTaskFrequency(domain.TaskFrequency("monthly")))
	assert.False(t, validTaskFrequency(domain.TaskFrequency("")))
}

// 周期汇总回应的覆盖区间：每日=当天、每周=滚动 7 天、一次性=任务起止。
// 汇总只关闭区间内的待回应提交，防止把上周遗留的提交一起关掉。
func TestTaskPeriodRange(t *testing.T) {
	// 与实现同源取时区，避免最小镜像缺 tzdata 时用例本身漂移。
	loc := shanghaiLoc()
	at := func(y, m, d int) time.Time { return time.Date(y, time.Month(m), d, 0, 0, 0, 0, loc) }
	ymd := func(v time.Time) string { return v.In(loc).Format("2006-01-02") }
	now := time.Date(2026, 10, 21, 15, 30, 0, 0, loc)

	start, end := taskPeriodRange(&domain.ParentTask{
		Frequency: domain.TaskFrequencyDaily, StartDate: at(2026, 10, 18), EndDate: at(2026, 10, 25),
	}, now)
	assert.Equal(t, "2026-10-21", ymd(start))
	assert.Equal(t, "2026-10-21", ymd(end))

	start, end = taskPeriodRange(&domain.ParentTask{
		Frequency: domain.TaskFrequencyWeekly, StartDate: at(2026, 10, 1), EndDate: at(2026, 11, 1),
	}, now)
	assert.Equal(t, "2026-10-15", ymd(start))
	assert.Equal(t, "2026-10-21", ymd(end))

	// 任务已结束：锚点取结束日，保证最后一天的待回应仍能关闭
	start, end = taskPeriodRange(&domain.ParentTask{
		Frequency: domain.TaskFrequencyDaily, StartDate: at(2026, 10, 10), EndDate: at(2026, 10, 18),
	}, now)
	assert.Equal(t, "2026-10-18", ymd(start), "已结束的每日任务必须落在最后一天")
	assert.Equal(t, "2026-10-18", ymd(end))

	// 任务尚未开始：锚点取开始日
	start, end = taskPeriodRange(&domain.ParentTask{
		Frequency: domain.TaskFrequencyDaily, StartDate: at(2026, 10, 25), EndDate: at(2026, 10, 30),
	}, now)
	assert.Equal(t, "2026-10-25", ymd(start))

	// 每周任务已结束：锚点=结束日，窗口=结束日往前 7 天
	start, end = taskPeriodRange(&domain.ParentTask{
		Frequency: domain.TaskFrequencyWeekly, StartDate: at(2026, 10, 1), EndDate: at(2026, 10, 18),
	}, now)
	assert.Equal(t, "2026-10-12", ymd(start))
	assert.Equal(t, "2026-10-18", ymd(end))

	// 每周窗口不得早于任务开始日
	start, end = taskPeriodRange(&domain.ParentTask{
		Frequency: domain.TaskFrequencyWeekly, StartDate: at(2026, 10, 18), EndDate: at(2026, 11, 1),
	}, now)
	assert.Equal(t, "2026-10-18", ymd(start))
	assert.Equal(t, "2026-10-21", ymd(end))

	// once 区间=任务起止（仅函数契约；批量入口由 batchRespondAllowed 拦截）
	once := &domain.ParentTask{
		Frequency: domain.TaskFrequencyOnce, StartDate: at(2026, 10, 1), EndDate: at(2026, 10, 30),
	}
	start, end = taskPeriodRange(once, now)
	assert.Equal(t, "2026-10-01", ymd(start))
	assert.Equal(t, "2026-10-30", ymd(end))

	// 脏数据（结束早于开始）退化成单日，避免生成反向区间
	broken := &domain.ParentTask{
		Frequency: domain.TaskFrequencyOnce, StartDate: at(2026, 10, 20), EndDate: at(2026, 10, 10),
	}
	start, end = taskPeriodRange(broken, now)
	assert.Equal(t, "2026-10-20", ymd(start))
	assert.Equal(t, "2026-10-20", ymd(end))
}

// 汇总回应只允许每日/每周的班级任务；个体任务与一次性任务必须逐条回应。
func TestBatchRespondAllowed(t *testing.T) {
	assert.True(t, batchRespondAllowed(&domain.ParentTask{Frequency: domain.TaskFrequencyDaily}))
	assert.True(t, batchRespondAllowed(&domain.ParentTask{Frequency: domain.TaskFrequencyWeekly}))
	assert.False(t, batchRespondAllowed(&domain.ParentTask{Frequency: domain.TaskFrequencyOnce}),
		"一次性活动必须逐条回应")
	assert.False(t, batchRespondAllowed(&domain.ParentTask{Frequency: domain.TaskFrequency("monthly")}),
		"未知频率不得落进 default 分支按全程汇总")

	individual := &domain.ParentTask{Frequency: domain.TaskFrequencyDaily, TargetStudentID: 21}
	assert.False(t, batchRespondAllowed(individual), "个体任务优先级高于频率，必须逐条回应")
}

func TestSameTaskDay(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	a := time.Date(2026, 10, 20, 9, 0, 0, 0, loc)
	b := time.Date(2026, 10, 20, 23, 59, 0, 0, loc)
	c := time.Date(2026, 10, 21, 0, 0, 0, 0, loc)
	assert.True(t, sameTaskDay(a, b))
	assert.False(t, sameTaskDay(a, c))
}
