package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/domain"
)

// B4-b 只读字段回归：老师端要能看见「已补充几条」与「任务罐已集几星」。
// 这两个字段只看，不改状态机；断言落在库里真实行上。
//
// 复用 task_response_supplement_test.go 的 setupB1 夹具（固定园所/班级/任务）。

func teacherTaskItem(t *testing.T, list map[string]any, taskID uint64) map[string]any {
	t.Helper()
	items, _ := list["list"].([]map[string]any)
	for _, item := range items {
		if id, _ := item["id"].(uint64); id == taskID {
			return item
		}
	}
	t.Fatalf("列表里找不到任务 %d", taskID)
	return nil
}

func TestTeacherTaskCheckins_ReturnsSupplements(t *testing.T) {
	f := setupB1(t)
	ctx := context.Background()

	batch, err := f.svcT.BatchRespondTask(ctx, f.tc, f.task.ID, domain.TaskRitualClassStar, "", "本周汇总", 0)
	require.NoError(t, err)
	mainID, _ := batch["id"].(uint64)
	require.NotZero(t, mainID)

	_, err = f.svcT.SupplementResponse(ctx, f.tc, f.task.ID, f.c1.ID, domain.TaskRitualSticker, "flower", "给本家庭孩子", 0)
	require.NoError(t, err)
	_, err = f.svcT.SupplementResponse(ctx, f.tc, f.task.ID, f.c1.ID, domain.TaskRitualVoice, "", "再补一句语音", testB1Audio)
	require.NoError(t, err)

	// c2 还没有补充：契约承诺是 [] 而不是 null。改成 nil slice 时 JSON 会变 null，
	// 客户端 `row.supplements.length` 会直接抛错。
	res, err := f.svcT.TaskCheckins(ctx, f.tc, f.task.ID, "responded")
	require.NoError(t, err)
	items, _ := res["list"].([]map[string]any)
	require.Len(t, items, 2)
	byID := make(map[uint64]map[string]any, 2)
	for _, item := range items {
		id, _ := item["id"].(uint64)
		byID[id] = item
	}
	emptyJSON, err := json.Marshal(byID[f.c2.ID]["supplements"])
	require.NoError(t, err)
	assert.Equal(t, "[]", string(emptyJSON), "无补充时必须是空数组")

	_, err = f.svcT.SupplementResponse(ctx, f.tc, f.task.ID, f.c2.ID, domain.TaskRitualSticker, "heart", "给别人家孩子", 0)
	require.NoError(t, err)

	res, err = f.svcT.TaskCheckins(ctx, f.tc, f.task.ID, "responded")
	require.NoError(t, err)
	items, _ = res["list"].([]map[string]any)
	require.Len(t, items, 2)
	byID = make(map[uint64]map[string]any, 2)
	for _, item := range items {
		id, _ := item["id"].(uint64)
		byID[id] = item
	}

	sp1, _ := byID[f.c1.ID]["supplements"].([]map[string]any)
	require.Len(t, sp1, 2, "本家庭孩子应有 2 条补充，且按时间/ID 升序")
	assert.Equal(t, "flower", sp1[0]["sticker_key"])
	assert.Equal(t, "sticker", sp1[0]["ritual"])
	assert.Equal(t, f.c1.ID, sp1[0]["checkin_id"])
	assert.Equal(t, "voice", sp1[1]["ritual"])
	assert.Equal(t, "https://cdn.example.com/task/voice.mp3", sp1[1]["media_url"])
	assert.NotEmpty(t, sp1[0]["created_at"])

	sp2, _ := byID[f.c2.ID]["supplements"].([]map[string]any)
	require.Len(t, sp2, 1, "另一个孩子的补充不能串到本孩子")
	assert.Equal(t, "heart", sp2[0]["sticker_key"])

	// 补充回应不是主回应：提交仍指向汇总回应。
	assert.EqualValues(t, mainID, byID[f.c1.ID]["response_id"])
}

func TestTeacherTaskRead_StarCount(t *testing.T) {
	f := setupB1(t)
	ctx := context.Background()

	// 任务罐：目标 5。初始没有回应 → 0 星。
	f.task.StarScope = domain.TaskStarScopeTask
	f.task.StarGoal = 5
	require.NoError(t, f.db.Save(&f.task).Error)

	before, err := f.svcT.ListTasks(ctx, f.tc, 1, 20)
	require.NoError(t, err)
	assert.EqualValues(t, 0, teacherTaskItem(t, before, f.task.ID)["star_count"])

	// 汇总回应覆盖 2 条提交 → 2 星（period 覆盖 N 条计 N 星）。
	_, err = f.svcT.BatchRespondTask(ctx, f.tc, f.task.ID, domain.TaskRitualClassStar, "", "本周汇总", 0)
	require.NoError(t, err)

	after, err := f.svcT.ListTasks(ctx, f.tc, 1, 20)
	require.NoError(t, err)
	assert.EqualValues(t, 2, teacherTaskItem(t, after, f.task.ID)["star_count"])

	detail, err := f.svcT.GetTask(ctx, f.tc, f.task.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 2, detail["star_count"])

	// 补充回应不计星：即使它也是 class_star，也不改 checkin.response_id。
	_, err = f.svcT.SupplementResponse(ctx, f.tc, f.task.ID, f.c1.ID, domain.TaskRitualClassStar, "", "补充不计星", 0)
	require.Error(t, err, "补充回应本身不允许 class_star（先钉住前置）")
	_, err = f.svcT.SupplementResponse(ctx, f.tc, f.task.ID, f.c1.ID, domain.TaskRitualSticker, "flower", "补充", 0)
	require.NoError(t, err)
	detail, err = f.svcT.GetTask(ctx, f.tc, f.task.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 2, detail["star_count"], "补充回应不能计星")

	// 软删一条提交后只算剩下的。
	require.NoError(t, f.db.Where("id = ?", f.c2.ID).Delete(&domain.TaskCheckIn{}).Error)
	detail, err = f.svcT.GetTask(ctx, f.tc, f.task.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 1, detail["star_count"])

	// 非 class_star 的主回应不计星。
	stickerTask := domain.ParentTask{
		KindergartenID: testB1KG, ClassID: testB1Class, CreatorID: testB1Teacher,
		Title: "B4-b 贴纸任务", Frequency: domain.TaskFrequencyOnce,
		StartDate: time.Now().In(shanghaiLoc()), EndDate: time.Now().In(shanghaiLoc()),
		Status: 1, StarScope: domain.TaskStarScopeTask, StarGoal: 5,
		RecognitionRitual: domain.TaskRitualSticker,
	}
	require.NoError(t, f.db.Create(&stickerTask).Error)
	stickerCheckin := domain.TaskCheckIn{
		TaskID: stickerTask.ID, StudentID: testB1Student, UserID: testB1Parent,
		Date: time.Now().In(shanghaiLoc()), Content: "贴纸提交",
	}
	require.NoError(t, f.db.Create(&stickerCheckin).Error)
	_, err = f.svcT.RespondTask(ctx, f.tc, stickerTask.ID, stickerCheckin.ID, domain.TaskRitualSticker, "star", "", 0)
	require.NoError(t, err)
	stickerDetail, err := f.svcT.GetTask(ctx, f.tc, stickerTask.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 0, stickerDetail["star_count"])
}

// 软删的回应不能计星：任务被删除提交与回应是级联软删，但这里单独验证回应软删。
func TestTeacherTaskRead_SoftDeletedResponseNotCounted(t *testing.T) {
	f := setupB1(t)
	ctx := context.Background()

	f.task.StarScope = domain.TaskStarScopeTask
	f.task.StarGoal = 5
	require.NoError(t, f.db.Save(&f.task).Error)
	_, err := f.svcT.RespondTask(ctx, f.tc, f.task.ID, f.c1.ID, domain.TaskRitualClassStar, "", "", 0)
	require.NoError(t, err)

	detail, err := f.svcT.GetTask(ctx, f.tc, f.task.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 1, detail["star_count"])

	respID := f.reloadCheckin(t, f.c1.ID).ResponseID
	require.NotZero(t, respID)
	require.NoError(t, f.db.Where("id = ?", respID).Delete(&domain.TaskResponse{}).Error)
	detail, err = f.svcT.GetTask(ctx, f.tc, f.task.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 0, detail["star_count"], "软删的回应不能计星")
}

// 星数必须按「提交的主回应」计数，不能按任务里存在 class_star 回应就乘上所有提交。
// 同一任务：一条 class_star 主回应 + 一条 sticker 主回应 → 只有 1 星。
func TestTeacherTaskRead_StarCountOnlyMainResponseCheckins(t *testing.T) {
	f := setupB1(t)
	ctx := context.Background()

	f.task.StarScope = domain.TaskStarScopeTask
	f.task.StarGoal = 5
	require.NoError(t, f.db.Save(&f.task).Error)

	_, err := f.svcT.RespondTask(ctx, f.tc, f.task.ID, f.c1.ID, domain.TaskRitualClassStar, "", "", 0)
	require.NoError(t, err)
	_, err = f.svcT.RespondTask(ctx, f.tc, f.task.ID, f.c2.ID, domain.TaskRitualSticker, "heart", "", 0)
	require.NoError(t, err)

	detail, err := f.svcT.GetTask(ctx, f.tc, f.task.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 1, detail["star_count"], "只有 c1 的主回应是 class_star")

	list, err := f.svcT.ListTasks(ctx, f.tc, 1, 20)
	require.NoError(t, err)
	assert.EqualValues(t, 1, teacherTaskItem(t, list, f.task.ID)["star_count"])
}

// 结构锁必须是独立错误：handler 要据此给出「有提交后只能改…」，
// 不能落进笼统的 ErrTeacherInvalidParams（老师看不出原因）。
func TestTeacherUpdateTask_StructuralLockReturnsSpecificError(t *testing.T) {
	f := setupB1(t)
	ctx := context.Background()

	// 夹具默认 class_star + 空 scope/goal，这种组合在 CreateTask 里会被拒绝；
	// 这里先改成合法的「贴纸 + 任务罐 + 目标 0」，否则更新会先撞仪式校验而不是结构锁。
	f.task.RecognitionRitual = domain.TaskRitualSticker
	f.task.StarScope = domain.TaskStarScopeTask
	f.task.StarGoal = 0
	require.NoError(t, f.db.Save(&f.task).Error)

	base := TeacherTaskInput{
		Title:             f.task.Title,
		LearningLink:      "新的学习关联",
		TargetStudentID:   f.task.TargetStudentID,
		Frequency:         string(f.task.Frequency),
		StartDate:         f.task.StartDate.Format("2006-01-02"),
		EndDate:           f.task.EndDate.Format("2006-01-02"),
		RecognitionRitual: f.task.RecognitionRitual,
		StarScope:         f.task.StarScope,
		StarGoal:          ptrTo(f.task.StarGoal),
		Status:            f.task.Status,
	}

	// 夹具里已经有两条提交；改频率就是结构变更。
	locked := base
	locked.Frequency = string(domain.TaskFrequencyWeekly)
	err := f.svcT.UpdateTask(ctx, f.tc, f.task.ID, locked)
	assert.ErrorIs(t, err, ErrTaskStructuralLocked)

	// 只改安全字段应成功。
	require.NoError(t, f.svcT.UpdateTask(ctx, f.tc, f.task.ID, base))
}
