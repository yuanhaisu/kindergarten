package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/domain"
)

// 3.2 / 3.4 / 5.6：老师端待回应队列、周期汇总边界、读标记时区。

func TestTeacherTaskCheckinsStatusFilterAndOrder(t *testing.T) {
	f := setupB3(t)
	ctx := context.Background()
	taskID := f.createTask(t, 0, "B3 待回应排序")["id"].(uint64)

	c1 := f.checkin(t, taskID, testB3StudentA1, testB3ParentA)
	c2 := f.checkin(t, taskID, testB3StudentA2, testB3ParentA)
	c3 := f.checkin(t, taskID, testB3StudentB, testB3ParentB)
	now := time.Now()
	for _, row := range []struct {
		id uint64
		at time.Time
	}{
		{c1.ID, now.Add(-2 * time.Hour)},
		{c2.ID, now.Add(-1 * time.Hour)},
		{c3.ID, now.Add(-3 * time.Hour)},
	} {
		require.NoError(t, f.db.Model(&domain.TaskCheckIn{}).
			Where("id = ?", row.id).Update("created_at", row.at).Error)
	}

	out, err := f.svcT.TaskCheckins(ctx, f.tc, taskID, "pending")
	require.NoError(t, err)
	rows := out["list"].([]map[string]any)
	require.Len(t, rows, 3)
	got := []uint64{rows[0]["id"].(uint64), rows[1]["id"].(uint64), rows[2]["id"].(uint64)}
	assert.Equal(t, []uint64{c3.ID, c1.ID, c2.ID}, got, "pending 必须按 created_at ASC")

	_, err = f.svcT.RespondTask(ctx, f.tc, taskID, c2.ID, domain.TaskRitualSticker, "star", "回应", 0)
	require.NoError(t, err)

	out, err = f.svcT.TaskCheckins(ctx, f.tc, taskID, "pending")
	require.NoError(t, err)
	assert.Len(t, out["list"].([]map[string]any), 2, "已回应的不再出现在 pending")

	out, err = f.svcT.TaskCheckins(ctx, f.tc, taskID, "responded")
	require.NoError(t, err)
	responded := out["list"].([]map[string]any)
	require.Len(t, responded, 1)
	assert.Equal(t, c2.ID, responded[0]["id"])
}

func TestBatchRespondEmptyAndPartiallyResponded(t *testing.T) {
	f := setupB3(t)
	ctx := context.Background()
	taskID := f.createTask(t, 0, "B3 汇总边界")["id"].(uint64)

	// 空列表：没有任何待回应，返回 0 而不是造一条空回应。
	res, err := f.svcT.BatchRespondTask(ctx, f.tc, taskID, domain.TaskRitualSticker, "star", "汇总", 0)
	require.NoError(t, err)
	assert.EqualValues(t, 0, res["responded_count"])
	assert.EqualValues(t, 0, res["id"])
	assert.Empty(t, res["failed_ids"])

	// 另一条任务的待回应不能被这次汇总关掉（跨任务隔离）。
	otherTaskID := f.createTask(t, 0, "B3 别的任务")["id"].(uint64)
	other := f.checkin(t, otherTaskID, testB3StudentB, testB3ParentB)

	// 已回应的那条保持原主回应，汇总只关闭仍未回应的。
	c1 := f.checkin(t, taskID, testB3StudentA1, testB3ParentA)
	c2 := f.checkin(t, taskID, testB3StudentA2, testB3ParentA)
	_, err = f.svcT.RespondTask(ctx, f.tc, taskID, c1.ID, domain.TaskRitualSticker, "star", "单条", 0)
	require.NoError(t, err)
	var before domain.TaskCheckIn
	require.NoError(t, f.db.First(&before, c1.ID).Error)

	res, err = f.svcT.BatchRespondTask(ctx, f.tc, taskID, domain.TaskRitualSticker, "star", "汇总", 0)
	require.NoError(t, err)
	assert.EqualValues(t, 1, res["responded_count"], "只计入仍未回应的那条")
	assert.Empty(t, res["failed_ids"])

	var afterFirst, afterSecond, untouched domain.TaskCheckIn
	require.NoError(t, f.db.First(&afterFirst, c1.ID).Error)
	require.NoError(t, f.db.First(&afterSecond, c2.ID).Error)
	require.NoError(t, f.db.First(&untouched, other.ID).Error)
	assert.Equal(t, before.ResponseID, afterFirst.ResponseID, "已回应的提交不得被汇总改写")
	assert.NotZero(t, afterSecond.ResponseID, "待回应的提交被汇总关闭")
	assert.Zero(t, untouched.ResponseID, "别的任务的提交不受影响")
}

// 5.6：读标记与补充回应都落 datetime(3)；同一瞬间用 UTC 表示写入也不能误判。
func TestSupplementUnreadAcrossTimezones(t *testing.T) {
	f := setupB3(t)
	out := f.createTask(t, 0, "B3 时区读标记")
	task := f.taskFor(t, out["id"].(uint64))
	_, supplementAt := f.supplementOnce(t, task.ID, testB3StudentA1, testB3ParentA)

	// 读发生在补充之后 1ms（用 UTC 表示同一瞬间）：必须判定为已读。
	readAt := supplementAt.Add(time.Millisecond).UTC()
	require.NoError(t, f.db.Create(&domain.TaskReadMark{
		KindergartenID: testB3KG, TaskID: task.ID, UserID: testB3ParentA,
		StudentID: testB3StudentA1, LastReadAt: &readAt,
	}).Error)

	detail, err := f.svcP.GetTask(context.Background(), f.pcA1, task.ID)
	require.NoError(t, err)
	assert.Equal(t, false, detail["has_unread_supplement"], "UTC 同一瞬间不得误判为未读")
}
