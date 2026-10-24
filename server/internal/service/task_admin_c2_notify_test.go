package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// decision §20.10-6 的写入点：notify_target_count 先写义务、再投递；NULL 表示未知。

func (f *c2Fixture) createClassTask(t *testing.T) uint64 {
	t.Helper()
	out, err := f.svc.CreateTask(f.ctx, testC2KG, testC2Admin, CreateTaskReq{
		Title:        "C2通知任务",
		LearningLink: "配合本周主题",
		ClassID:      testC2Class,
		Frequency:    string(domain.TaskFrequencyDaily),
		StartDate:    f.date(0),
		EndDate:      f.date(6),
	}, "", "")
	require.NoError(t, err)
	return out["id"].(uint64)
}

func (f *c2Fixture) countMessagesTo(t *testing.T, userID uint64) int64 {
	t.Helper()
	var n int64
	require.NoError(t, f.db.Model(&domain.Message{}).Where("user_id = ?", userID).Count(&n).Error)
	return n
}

func TestRecordNotifyTargetCountWritesNullForUnknown(t *testing.T) {
	f := setupC2(t)
	taskID := f.createClassTask(t)
	require.NotNil(t, f.loadTaskC2(t, taskID).NotifyTargetCount)

	// 义务未知时必须把旧值清成 NULL：写 0 会让「未知」与「确实没有家长」不可区分。
	recordTaskNotifyTargetCount(f.ctx, f.db, taskID, nil)
	assert.Nil(t, f.loadTaskC2(t, taskID).NotifyTargetCount)

	response := domain.TaskResponse{
		KindergartenID: testC2KG, TaskID: taskID, TeacherID: testC2Admin,
		Ritual: domain.TaskRitualSticker, ResponseType: domain.TaskResponseSingle,
	}
	require.NoError(t, f.db.Create(&response).Error)
	one := 1
	recordResponseNotifyTargetCount(f.ctx, f.db, response.ID, &one)
	var fresh domain.TaskResponse
	require.NoError(t, f.db.Where("id = ?", response.ID).First(&fresh).Error)
	require.NotNil(t, fresh.NotifyTargetCount)

	recordResponseNotifyTargetCount(f.ctx, f.db, response.ID, nil)
	require.NoError(t, f.db.Where("id = ?", response.ID).First(&fresh).Error)
	assert.Nil(t, fresh.NotifyTargetCount, "回应侧同样：未知写 NULL")
}

func TestPublishNotifyTargetCountWritten(t *testing.T) {
	f := setupC2(t)
	taskID := f.createClassTask(t)

	tk := f.loadTaskC2(t, taskID)
	require.NotNil(t, tk.NotifyTargetCount, "发布时必须先写义务集，不能留 NULL")
	assert.Equal(t, 1, *tk.NotifyTargetCount, "本夹具班级 A 只有 1 个家长")
	assert.Equal(t, int64(1), f.countMessagesTo(t, testC2Parent))
}

func TestPublishNotifyTargetCountStableAndNotInflatedByDedupe(t *testing.T) {
	f := setupC2(t)
	taskID := f.createClassTask(t)
	tk := f.loadTaskC2(t, taskID)
	require.NotNil(t, tk.NotifyTargetCount)
	first := *tk.NotifyTargetCount

	// 再发一次（同一任务）：判重命中，不写新消息；义务集大小不变。
	notifyTaskPublishedFor(f.ctx, f.db, repository.NewMessageRepo(f.db), &tk)

	again := f.loadTaskC2(t, taskID)
	require.NotNil(t, again.NotifyTargetCount)
	assert.Equal(t, first, *again.NotifyTargetCount, "判重命中不得改变义务集（分母不是 messages 行数）")
	assert.Equal(t, int64(1), f.countMessagesTo(t, testC2Parent), "同一任务对同一家长至多一条")
}

func TestResponseNotifyTargetCountWritten(t *testing.T) {
	f := setupC2(t)
	taskID := f.createClassTask(t)
	tk := f.loadTaskC2(t, taskID)

	response := domain.TaskResponse{
		KindergartenID: testC2KG, TaskID: taskID, TeacherID: testC2Admin,
		Ritual: domain.TaskRitualSticker, ResponseType: domain.TaskResponseSingle,
	}
	require.NoError(t, f.db.Create(&response).Error)
	checkin := domain.TaskCheckIn{
		TaskID: taskID, StudentID: testC2Student, UserID: testC2Parent,
		Date: f.today, ResponseID: response.ID,
	}
	require.NoError(t, f.db.Create(&checkin).Error)

	sent, failed := f.svcT.notifyTaskResponse(f.ctx, &tk, response.ID, []domain.TaskCheckIn{checkin})
	assert.Equal(t, 1, sent)
	assert.Equal(t, 0, failed)

	var fresh domain.TaskResponse
	require.NoError(t, f.db.Where("id = ?", response.ID).First(&fresh).Error)
	require.NotNil(t, fresh.NotifyTargetCount, "主回应必须写入义务集")
	assert.Equal(t, 1, *fresh.NotifyTargetCount)
}

func TestSupplementResponseLeavesNotifyTargetCountNull(t *testing.T) {
	f := setupC2(t)
	taskID := f.createClassTask(t)

	main := domain.TaskResponse{
		KindergartenID: testC2KG, TaskID: taskID, TeacherID: testC2Admin,
		Ritual: domain.TaskRitualSticker, ResponseType: domain.TaskResponseSingle,
	}
	require.NoError(t, f.db.Create(&main).Error)
	checkin := domain.TaskCheckIn{
		TaskID: taskID, StudentID: testC2Student, UserID: testC2Parent,
		Date: f.today, ResponseID: main.ID,
	}
	require.NoError(t, f.db.Create(&checkin).Error)

	tc := &TeacherPortalContext{UserID: testC2Admin, KindergartenID: testC2KG, ClassID: testC2Class}
	_, err := f.svcT.SupplementResponse(f.ctx, tc, taskID, checkin.ID, "sticker", "smile", "补一句", 0)
	require.NoError(t, err)

	var supplement domain.TaskResponse
	require.NoError(t, f.db.Where("task_id = ? AND id <> ?", taskID, main.ID).First(&supplement).Error)
	assert.Nil(t, supplement.NotifyTargetCount,
		"补充回应不投递，义务集必须保持 NULL——它不进反馈率分母")
}
