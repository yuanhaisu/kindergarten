package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
)

// B1（补充回应 + 家长反馈）回归。
//
// 断言的是**库里的真实状态**：补充回应是否真的没改 ResponseID、关系行是否落在正确的
// checkin 上、反馈是否按家长各自一行。这三件事都只能由真实唯一索引与事务行为证明，
// 用内存桩测不出来。
//
// 运行约束：
//   - 夹具用固定 ID（992601-992610）与固定手机号，清理按这些 ID 做。
//     **不要并发跑两个 `go test ./internal/service` 进程**，后启动的 clean() 会把
//     前一个进程正在用的行删掉。同一进程内没有 t.Parallel()，函数是串行的。
//   - 日期取 `time.Now()`，跨上海零点运行会让「今天提交」落在前一周期，
//     汇总回应取不到行而失败；这不是实现缺陷，重跑即可。
//   - 时区由包内 init() 把 time.Local 对齐到 Asia/Shanghai（见 timezone.go），
//     所以 DSN 的 loc=Local 与夹具的 shanghaiLoc() 是同一个墙钟。
//
// 本文件**不**覆盖（不要把这些当成已验证）：
//   - 3.6 的「补充回应的反馈不进入主回应指标」：指标聚合属 7.2，当前没有可断言的实现。
//     缺口登记在 TEST_CATALOG TC-NEW-TSK-0113 与 handoff §12.4。
//   - 各处的 DB 故障分支，以及 media_file_id 指向非 image/audio 文件的分支。

const (
	testB1KG          uint64 = 992601
	testB1Class       uint64 = 992602
	testB1ClassOther  uint64 = 992603
	testB1Teacher     uint64 = 992604
	testB1Parent      uint64 = 992605
	testB1ParentOther uint64 = 992606
	testB1Student     uint64 = 992607
	testB1StudentOth  uint64 = 992608
	// 同一孩子的第二位家长：有绑定关系、但**没有**提交过。用来区分
	// 「反馈权限按家长账号」与「按当前孩子」两种实现——后者会误判为允许。
	testB1Parent2 uint64 = 992609
	// 别人上传的任务素材：用来钉住 SupplementResponse 的归属校验。
	testB1File uint64 = 992610
	// 老师自己上传的音频/图片/视频：语音仪式必须挂音频，非 image/audio 要被拒。
	testB1Audio uint64 = 992611
	testB1Image uint64 = 992612
	testB1Video uint64 = 992613
	// 真实存在、但与本班任何孩子都没有绑定关系的用户。
	// 用它而不是「未创建的 id」，才能区分「没绑定」与「用户不存在」两种实现。
	testB1Stranger uint64 = 992614
)

func openTaskB1DB(t *testing.T) *gorm.DB {
	t.Helper()
	cfgPath := filepath.Join("..", "..", "configs", "app.ini")
	if _, err := os.Stat(cfgPath); err != nil {
		t.Skipf("跳过：找不到 %s（本地无 DB 配置）", cfgPath)
	}
	cfg, err := config.Load(cfgPath)
	require.NoError(t, err)
	if cfg.MySQL.DSN == "" {
		t.Skip("跳过：MySQL DSN 为空")
	}
	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	return db
}

// b1Fixture 一套「一个班 + 两个孩子 + 两个家庭 + 一个每日班级任务」的最小夹具。
type b1Fixture struct {
	db    *gorm.DB
	task  domain.ParentTask
	svcT  *TeacherPortalService
	svcP  *ParentPortalService
	pc    *ParentPortalContext
	tc    *TeacherPortalContext
	c1    domain.TaskCheckIn // 本家庭孩子的提交
	c2    domain.TaskCheckIn // 别人家孩子的提交
	clean func()
}

func setupB1(t *testing.T) *b1Fixture {
	t.Helper()
	db := openTaskB1DB(t)
	loc := shanghaiLoc()
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	// 夹具清理：所有清理都用 Unscoped，软删行会占唯一键，重跑就会撞。
	clean := func() {
		var taskIDs []uint64
		if err := db.Unscoped().Model(&domain.ParentTask{}).
			Where("kindergarten_id = ? OR class_id IN ?", testB1KG, []uint64{testB1Class, testB1ClassOther}).
			Pluck("id", &taskIDs).Error; err != nil {
			t.Logf("B1 夹具：查遗留任务失败: %v", err)
		}
		if len(taskIDs) > 0 {
			// 反馈只挂在 response_id 上，没有 task_id / check_in_id，必须单独按回应删。
			var respIDs []uint64
			if err := db.Unscoped().Model(&domain.TaskResponse{}).
				Where("task_id IN ?", taskIDs).Pluck("id", &respIDs).Error; err != nil {
				// 这里必须报出来：查不到 respIDs 就不会删 reactions，残留行会占住
				// (response_id, user_id) 唯一键，下一次测试插入就失败。
				t.Logf("B1 夹具清理失败 responses(pluck): %v", err)
			}
			if len(respIDs) > 0 {
				if err := db.Unscoped().Where("response_id IN ?", respIDs).
					Delete(&domain.TaskResponseReaction{}).Error; err != nil {
					t.Logf("B1 夹具清理失败 reactions: %v", err)
				}
			}
			for _, step := range []struct {
				name string
				err  error
			}{
				{"supplements", db.Unscoped().Where("check_in_id IN (SELECT id FROM task_check_ins WHERE task_id IN ?)", taskIDs).Delete(&domain.TaskResponseSupplement{}).Error},
				{"checkins", db.Unscoped().Where("task_id IN ?", taskIDs).Delete(&domain.TaskCheckIn{}).Error},
				{"responses", db.Unscoped().Where("task_id IN ?", taskIDs).Delete(&domain.TaskResponse{}).Error},
				{"states", db.Unscoped().Where("task_id IN ?", taskIDs).Delete(&domain.TaskCheckInState{}).Error},
				{"read_marks", db.Unscoped().Where("task_id IN ?", taskIDs).Delete(&domain.TaskReadMark{}).Error},
				{"tasks", db.Unscoped().Where("id IN ?", taskIDs).Delete(&domain.ParentTask{}).Error},
			} {
				if step.err != nil {
					t.Logf("B1 夹具清理失败 %s: %v", step.name, step.err)
				}
			}
		}
		for _, step := range []struct {
			name string
			err  error
		}{
			// 顺序按外键方向：student_parents → students → classes → files → users → kindergarten。
			// 库上确实建了这三条 RESTRICT 外键（fk_student_parents_student / fk_students_class /
			// fk_classes_kindergarten）；先删父表会让本步骤 1451 失败、残留行再让下一轮
			// setup 的 student_parents 插入撞 1452（C-4 审查时实测到过）。
			{"student_parents", db.Unscoped().Where("student_id IN ?", []uint64{testB1Student, testB1StudentOth}).Delete(&domain.StudentParent{}).Error},
			{"students", db.Unscoped().Where("id IN ?", []uint64{testB1Student, testB1StudentOth}).Delete(&domain.Student{}).Error},
			{"classes", db.Unscoped().Where("id IN ?", []uint64{testB1Class, testB1ClassOther}).Delete(&domain.Class{}).Error},
			{"files", db.Unscoped().Where("id IN ?", []uint64{testB1File, testB1Audio, testB1Image, testB1Video}).Delete(&domain.File{}).Error},
			// 回应/发布通知会写 messages（fk_messages_user）。漏删这一张，users 的删除会被 1451 拦住，
			// 残留用户让下一轮 setup 撞 1062——C-4 审查时实测到过（C1 夹具早就删了这一步）。
			{"messages", db.Unscoped().Where("user_id IN ?", []uint64{testB1Teacher, testB1Parent, testB1ParentOther, testB1Parent2, testB1Stranger}).Delete(&domain.Message{}).Error},
			{"users", db.Unscoped().Where("id IN ?", []uint64{testB1Teacher, testB1Parent, testB1ParentOther, testB1Parent2, testB1Stranger}).Delete(&domain.User{}).Error},
			{"app_settings", db.Unscoped().Where("kindergarten_id = ?", testB1KG).Delete(&domain.KindergartenAppSetting{}).Error},
			{"kindergarten", db.Unscoped().Where("id = ?", testB1KG).Delete(&domain.Kindergarten{}).Error},
		} {
			if step.err != nil {
				t.Logf("B1 夹具清理失败 %s: %v", step.name, step.err)
			}
		}
	}
	clean()
	t.Cleanup(clean)

	require.NoError(t, db.Unscoped().Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: testB1KG}, Name: "测试园所-B1补充回应",
	}).Error)
	// 班级罐按「当前学期」圈定日期窗口（spring 1/1–7/31、fall 8/1–12/31）。
	// 夹具的提交都落在今天，用 fall 才能落在窗口内；否则 8–12 月跑测试会算出 0 星。
	require.NoError(t, db.Unscoped().Create(&domain.KindergartenAppSetting{
		KindergartenID: testB1KG, CurrentSemester: "fall", ExtraConfig: "{}",
	}).Error)
	require.NoError(t, db.Unscoped().Create(&domain.User{
		BaseModel: domain.BaseModel{ID: testB1Teacher}, Nickname: "B1老师", Phone: "13900009601", PasswordHash: "x",
	}).Error)
	require.NoError(t, db.Unscoped().Create(&domain.User{
		BaseModel: domain.BaseModel{ID: testB1Parent}, Nickname: "B1家长", Phone: "13900009602", PasswordHash: "x",
	}).Error)
	require.NoError(t, db.Unscoped().Create(&domain.User{
		BaseModel: domain.BaseModel{ID: testB1ParentOther}, Nickname: "B1别人家家长", Phone: "13900009603", PasswordHash: "x",
	}).Error)
	require.NoError(t, db.Unscoped().Create(&domain.User{
		BaseModel: domain.BaseModel{ID: testB1Parent2}, Nickname: "B1同家庭第二位家长", Phone: "13900009604", PasswordHash: "x",
	}).Error)
	require.NoError(t, db.Unscoped().Create(&domain.User{
		BaseModel: domain.BaseModel{ID: testB1Stranger}, Nickname: "B1无绑定用户", Phone: "13900009605", PasswordHash: "x",
	}).Error)
	require.NoError(t, db.Unscoped().Create(&domain.Class{
		BaseModel: domain.BaseModel{ID: testB1Class}, KindergartenID: testB1KG,
		Name: "B1测试班", Grade: "小班", Status: 1,
	}).Error)
	require.NoError(t, db.Unscoped().Create(&domain.Class{
		BaseModel: domain.BaseModel{ID: testB1ClassOther}, KindergartenID: testB1KG,
		Name: "B1别的班", Grade: "小班", Status: 1,
	}).Error)
	require.NoError(t, db.Unscoped().Create(&domain.Student{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testB1Student}},
		KindergartenID:  testB1KG, ClassID: testB1Class, Name: "B1孩子", Status: 1,
		Birthday:   time.Date(2021, 5, 1, 0, 0, 0, 0, loc),
		EnrollDate: time.Date(2024, 9, 1, 0, 0, 0, 0, loc),
	}).Error)
	require.NoError(t, db.Unscoped().Create(&domain.Student{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testB1StudentOth}},
		KindergartenID:  testB1KG, ClassID: testB1Class, Name: "B1别人家孩子", Status: 1,
		Birthday:   time.Date(2021, 6, 1, 0, 0, 0, 0, loc),
		EnrollDate: time.Date(2024, 9, 1, 0, 0, 0, 0, loc),
	}).Error)
	// 归属校验用：素材属于「别人家家长」，老师拿它当补充回应的媒体必须被拒。
	require.NoError(t, db.Unscoped().Create(&domain.File{
		BaseModel: domain.BaseModel{ID: testB1File},
		UserID:    testB1ParentOther, BizType: "task", FileType: "image",
		FileURL: "https://cdn.example.com/task/not-mine.jpg", FileName: "not-mine.jpg",
	}).Error)
	require.NoError(t, db.Unscoped().Create(&domain.File{
		BaseModel: domain.BaseModel{ID: testB1Audio},
		UserID:    testB1Teacher, BizType: "task", FileType: "audio",
		FileURL: "https://cdn.example.com/task/voice.mp3", FileName: "voice.mp3",
	}).Error)
	require.NoError(t, db.Unscoped().Create(&domain.File{
		BaseModel: domain.BaseModel{ID: testB1Image},
		UserID:    testB1Teacher, BizType: "task", FileType: "image",
		FileURL: "https://cdn.example.com/task/pic.jpg", FileName: "pic.jpg",
	}).Error)
	require.NoError(t, db.Unscoped().Create(&domain.File{
		BaseModel: domain.BaseModel{ID: testB1Video},
		UserID:    testB1Teacher, BizType: "task", FileType: "video",
		FileURL: "https://cdn.example.com/task/clip.mp4", FileName: "clip.mp4",
	}).Error)
	// 绑定关系：testB1Parent 与 testB1Parent2 是同一个孩子的两位家长。
	require.NoError(t, db.Unscoped().Create(&domain.StudentParent{
		StudentID: testB1Student, UserID: testB1Parent, Relation: domain.RelationMother, IsPrimary: true,
	}).Error)
	require.NoError(t, db.Unscoped().Create(&domain.StudentParent{
		StudentID: testB1Student, UserID: testB1Parent2, Relation: domain.RelationFather,
	}).Error)
	require.NoError(t, db.Unscoped().Create(&domain.StudentParent{
		StudentID: testB1StudentOth, UserID: testB1ParentOther, Relation: domain.RelationMother, IsPrimary: true,
	}).Error)

	f := &b1Fixture{db: db, clean: clean}
	f.task = domain.ParentTask{
		KindergartenID: testB1KG, ClassID: testB1Class, CreatorID: testB1Teacher,
		Title: "B1每日阅读", Frequency: domain.TaskFrequencyDaily,
		StartDate: today, EndDate: today.AddDate(0, 0, 6), Status: 1,
		RecognitionRitual: domain.TaskRitualClassStar, // 故意设成班级星星：补充回应的默认值不能继承它
	}
	require.NoError(t, db.Create(&f.task).Error)

	f.c1 = domain.TaskCheckIn{
		TaskID: f.task.ID, StudentID: testB1Student, UserID: testB1Parent,
		Date: today, Content: "本家庭提交",
	}
	f.c2 = domain.TaskCheckIn{
		TaskID: f.task.ID, StudentID: testB1StudentOth, UserID: testB1ParentOther,
		Date: today, Content: "别人家提交",
	}
	require.NoError(t, db.Create(&f.c1).Error)
	require.NoError(t, db.Create(&f.c2).Error)

	f.svcT = NewTeacherPortalService(db, nil, nil, nil, nil, nil, config.FaceConfig{})
	f.svcP = NewParentPortalService(db, nil, nil, nil, nil, nil, nil, nil)

	f.tc = &TeacherPortalContext{UserID: testB1Teacher, KindergartenID: testB1KG, ClassID: testB1Class}
	f.pc = &ParentPortalContext{
		UserID:         testB1Parent,
		KindergartenID: testB1KG,
		CurrentStudent: &domain.Student{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testB1Student}},
			ClassID:         testB1Class,
		},
	}
	return f
}

// countSupplements 统计某条提交上的补充关系行数。
func (f *b1Fixture) countSupplements(t *testing.T, checkinID uint64) int64 {
	t.Helper()
	var n int64
	require.NoError(t, f.db.Model(&domain.TaskResponseSupplement{}).
		Where("check_in_id = ?", checkinID).Count(&n).Error)
	return n
}

// reloadCheckin 读回提交行，用来断言 ResponseID 没有被补充回应改动。
func (f *b1Fixture) reloadCheckin(t *testing.T, id uint64) domain.TaskCheckIn {
	t.Helper()
	var c domain.TaskCheckIn
	require.NoError(t, f.db.Where("id = ?", id).First(&c).Error)
	return c
}

// mainResponseFor 按提交的 response_id 取回主回应。
// 不能用 `Order("id DESC").First(&resp)`：库里还有开发数据与其它测试行，
// 那条查询会把别的会话刚写入的回应当成自己的。
func (f *b1Fixture) mainResponseFor(t *testing.T, checkinID uint64) domain.TaskResponse {
	t.Helper()
	c := f.reloadCheckin(t, checkinID)
	require.NotZero(t, c.ResponseID, "该提交还没有主回应")
	var resp domain.TaskResponse
	require.NoError(t, f.db.Where("id = ?", c.ResponseID).First(&resp).Error)
	return resp
}

// countTaskResponses 统计该任务下的回应总数（含 period/single/supplement）。
func (f *b1Fixture) countTaskResponses(t *testing.T) int64 {
	t.Helper()
	var n int64
	require.NoError(t, f.db.Model(&domain.TaskResponse{}).
		Where("task_id = ?", f.task.ID).Count(&n).Error)
	return n
}

func TestSupplement_AfterPeriodResponseKeepsMainResponseID(t *testing.T) {
	f := setupB1(t)
	ctx := context.Background()

	batch, err := f.svcT.BatchRespondTask(ctx, f.tc, f.task.ID, domain.TaskRitualClassStar, "star", "本周大家都很棒", 0)
	require.NoError(t, err)
	mainID, _ := batch["id"].(uint64)
	require.NotZero(t, mainID, "汇总回应必须真的创建")
	assert.EqualValues(t, 2, batch["responded_count"])
	// responded_count 是服务内部累加出来的数字，必须和库里的真实归属对上，
	// 否则「计数恒等于候选数」这类实现错误测不出来。
	var attached int64
	require.NoError(t, f.db.Model(&domain.TaskCheckIn{}).
		Where("task_id = ? AND response_id = ?", f.task.ID, mainID).Count(&attached).Error)
	assert.EqualValues(t, 2, attached, "库里必须恰有 2 条提交指向该汇总回应")
	assert.EqualValues(t, attached, batch["responded_count"])

	mainResp := f.reloadCheckin(t, f.c1.ID)
	require.Equal(t, mainID, mainResp.ResponseID)

	// 汇总之后仍可追加个别补充；补充不改变 ResponseID。
	got, err := f.svcT.SupplementResponse(ctx, f.tc, f.task.ID, f.c1.ID, domain.TaskRitualSticker, "flower", "单独夸一句", 0)
	require.NoError(t, err)
	supID, _ := got["id"].(uint64)
	require.NotZero(t, supID)
	assert.NotEqual(t, mainID, supID, "补充回应必须是一条独立的 TaskResponse")
	assert.NotEmpty(t, got["created_at"])

	after := f.reloadCheckin(t, f.c1.ID)
	assert.Equal(t, mainID, after.ResponseID, "补充回应 MUST NOT 改动提交指向的主回应")
	assert.EqualValues(t, 1, f.countSupplements(t, f.c1.ID))

	// 语音同样允许，且允许对同一条提交追加多条补充。
	_, err = f.svcT.SupplementResponse(ctx, f.tc, f.task.ID, f.c1.ID, domain.TaskRitualVoice, "", "再说一句", testB1Audio)
	require.NoError(t, err)
	assert.EqualValues(t, 2, f.countSupplements(t, f.c1.ID))

	var supRows []domain.TaskResponseSupplement
	require.NoError(t, f.db.Where("check_in_id = ?", f.c1.ID).Order("id ASC").Find(&supRows).Error)
	require.Len(t, supRows, 2)
	assert.Equal(t, supID, supRows[0].ResponseID, "关系行必须指向补充回应本身")
	var supResp domain.TaskResponse
	require.NoError(t, f.db.Where("id = ?", supRows[0].ResponseID).First(&supResp).Error)
	assert.Equal(t, domain.TaskResponseSingle, supResp.ResponseType, "补充回应固定 single")
	assert.Equal(t, domain.TaskRitualSticker, supResp.Ritual)
	// 写入的归属字段必须正确：错了会串租户/串老师，而上面所有断言都发现不了。
	assert.Equal(t, f.task.ID, supResp.TaskID)
	assert.Equal(t, testB1KG, supResp.KindergartenID)
	assert.Equal(t, testB1Teacher, supResp.TeacherID)
	assert.Equal(t, testB1KG, supRows[0].KindergartenID, "关系行也要带园所")
	assert.Equal(t, 3, int(f.countTaskResponses(t)), "1 条汇总 + 2 条补充")
}

func TestSupplement_PendingCheckinRejected(t *testing.T) {
	f := setupB1(t)
	ctx := context.Background()

	// 不做主回应：该提交仍是待回应。
	_, err := f.svcT.SupplementResponse(ctx, f.tc, f.task.ID, f.c1.ID, domain.TaskRitualSticker, "star", "", 0)
	require.ErrorIs(t, err, ErrTaskSupplementForbidden)
	assert.EqualValues(t, 0, f.countSupplements(t, f.c1.ID), "被拒时不能留下补充关系")
	// 也要数 TaskResponse：只查关系行的话，「先建回应再校验」的实现会留下孤儿回应却测不出来。
	assert.Zero(t, f.countTaskResponses(t), "被拒时不能留下任何 TaskResponse")
	assert.Zero(t, f.reloadCheckin(t, f.c1.ID).ResponseID)
}

func TestSupplement_RitualRules(t *testing.T) {
	f := setupB1(t)
	ctx := context.Background()

	_, err := f.svcT.RespondTask(ctx, f.tc, f.task.ID, f.c1.ID, domain.TaskRitualSticker, "star", "主回应", 0)
	require.NoError(t, err)

	// 班级星星明确拒绝：一条提交只有一个主回应能计入星星罐。
	mainOnly := f.countTaskResponses(t)
	_, err = f.svcT.SupplementResponse(ctx, f.tc, f.task.ID, f.c1.ID, domain.TaskRitualClassStar, "", "", 0)
	require.ErrorIs(t, err, ErrTaskSupplementRitual)
	assert.EqualValues(t, 0, f.countSupplements(t, f.c1.ID))
	assert.Equal(t, mainOnly, f.countTaskResponses(t), "被拒时不能留下 TaskResponse")

	// 未知取值同样拒绝，不能悄悄落成贴纸。
	_, err = f.svcT.SupplementResponse(ctx, f.tc, f.task.ID, f.c1.ID, "banana", "", "", 0)
	require.ErrorIs(t, err, ErrTaskSupplementRitual)
	assert.EqualValues(t, 0, f.countSupplements(t, f.c1.ID))
	assert.Equal(t, mainOnly, f.countTaskResponses(t), "被拒时不能留下 TaskResponse")

	// 未指定仪式 → 默认贴纸，**不继承**任务默认仪式（本任务默认是 class_star）。
	_, err = f.svcT.SupplementResponse(ctx, f.tc, f.task.ID, f.c1.ID, "", "heart", "默认仪式", 0)
	require.NoError(t, err)
	assert.EqualValues(t, 1, f.countSupplements(t, f.c1.ID), "只应写一条补充关系")
	assert.Equal(t, mainOnly+1, f.countTaskResponses(t), "只应新增一条 TaskResponse")
	var sup domain.TaskResponseSupplement
	require.NoError(t, f.db.Where("check_in_id = ?", f.c1.ID).First(&sup).Error)
	var resp domain.TaskResponse
	require.NoError(t, f.db.Where("id = ?", sup.ResponseID).First(&resp).Error)
	assert.Equal(t, domain.TaskRitualSticker, resp.Ritual, "未指定仪式时默认贴纸，不得继承任务默认仪式")
}

func TestSupplement_CrossClassRejected(t *testing.T) {
	f := setupB1(t)
	ctx := context.Background()

	_, err := f.svcT.RespondTask(ctx, f.tc, f.task.ID, f.c1.ID, domain.TaskRitualSticker, "star", "", 0)
	require.NoError(t, err)

	otherTC := &TeacherPortalContext{UserID: testB1Teacher, KindergartenID: testB1KG, ClassID: testB1ClassOther}
	_, err = f.svcT.SupplementResponse(ctx, otherTC, f.task.ID, f.c1.ID, domain.TaskRitualSticker, "star", "", 0)
	require.ErrorIs(t, err, ErrTeacherNotFound, "别的班老师不能给该任务补回应")
	assert.EqualValues(t, 0, f.countSupplements(t, f.c1.ID))

	// 跨园所：班级 id 对得上也不能放行。这条钉住 WHERE 里的 kindergarten_id = ?，
	// 只改 ClassID 的用例发现不了「园所条件被删」。
	otherKGTC := &TeacherPortalContext{UserID: testB1Teacher, KindergartenID: testB1KG + 1, ClassID: testB1Class}
	_, err = f.svcT.SupplementResponse(ctx, otherKGTC, f.task.ID, f.c1.ID, domain.TaskRitualSticker, "star", "", 0)
	require.ErrorIs(t, err, ErrTeacherNotFound, "跨园所老师不能给该任务补回应")
	assert.EqualValues(t, 0, f.countSupplements(t, f.c1.ID))

	// 提交不属于该任务：拒绝，且不能借别的任务的提交写关系。
	other := domain.ParentTask{
		KindergartenID: testB1KG, ClassID: testB1Class, CreatorID: testB1Teacher,
		Title: "B1另一个任务", Frequency: domain.TaskFrequencyDaily,
		StartDate: f.task.StartDate, EndDate: f.task.EndDate, Status: 1,
	}
	require.NoError(t, f.db.Create(&other).Error)
	_, err = f.svcT.SupplementResponse(ctx, f.tc, other.ID, f.c1.ID, domain.TaskRitualSticker, "star", "", 0)
	require.ErrorIs(t, err, ErrTeacherInvalidParams)
	assert.EqualValues(t, 0, f.countSupplements(t, f.c1.ID))
}

func TestParentGetTask_ReturnsMainAndSupplementsScopedToChild(t *testing.T) {
	f := setupB1(t)
	ctx := context.Background()

	batch, err := f.svcT.BatchRespondTask(ctx, f.tc, f.task.ID, domain.TaskRitualSticker, "star", "本周汇总", 0)
	require.NoError(t, err)
	mainID, _ := batch["id"].(uint64)

	_, err = f.svcT.SupplementResponse(ctx, f.tc, f.task.ID, f.c1.ID, domain.TaskRitualSticker, "flower", "给本家庭孩子的补充", 0)
	require.NoError(t, err)
	_, err = f.svcT.SupplementResponse(ctx, f.tc, f.task.ID, f.c2.ID, domain.TaskRitualSticker, "moon", "给别人家孩子的补充", 0)
	require.NoError(t, err)

	detail, err := f.svcP.GetTask(ctx, f.pc, f.task.ID)
	require.NoError(t, err)
	respList, _ := detail["responses"].([]map[string]any)
	require.Len(t, respList, 1, "汇总回应是一条主回应")
	assert.Equal(t, mainID, respList[0]["id"])
	spList, _ := respList[0]["supplements"].([]map[string]any)
	require.Len(t, spList, 1, "只能看到本家庭孩子的补充回应")
	assert.Equal(t, "flower", spList[0]["sticker_key"])
	assert.Equal(t, f.c1.ID, spList[0]["checkin_id"])
}

func TestReactToResponse_PerParentUpsert(t *testing.T) {
	f := setupB1(t)
	ctx := context.Background()

	batch, err := f.svcT.BatchRespondTask(ctx, f.tc, f.task.ID, domain.TaskRitualSticker, "star", "本周汇总", 0)
	require.NoError(t, err)
	mainID, _ := batch["id"].(uint64)

	got, err := f.svcP.ReactToResponse(ctx, f.pc, mainID, domain.TaskReactionChildHappy)
	require.NoError(t, err)
	assert.EqualValues(t, mainID, got["response_id"])
	assert.Equal(t, domain.TaskReactionChildHappy, got["reaction"])

	// 同一家长更新自己的反馈：只有一行，值被改写。
	_, err = f.svcP.ReactToResponse(ctx, f.pc, mainID, domain.TaskReactionThanks)
	require.NoError(t, err)
	var rows []domain.TaskResponseReaction
	require.NoError(t, f.db.Where("response_id = ?", mainID).Find(&rows).Error)
	require.Len(t, rows, 1, "同一家长重复反馈必须 upsert，不能产生第二行")
	assert.Equal(t, domain.TaskReactionThanks, rows[0].Reaction)
	assert.Equal(t, testB1Parent, rows[0].UserID)

	// 另一个家庭对同一条汇总回应反馈：各自一行，互不覆盖。
	pcOther := &ParentPortalContext{
		UserID:         testB1ParentOther,
		KindergartenID: testB1KG,
		CurrentStudent: &domain.Student{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testB1StudentOth}},
			ClassID:         testB1Class,
		},
	}
	_, err = f.svcP.ReactToResponse(ctx, pcOther, mainID, domain.TaskReactionReceived)
	require.NoError(t, err)
	require.NoError(t, f.db.Where("response_id = ?", mainID).Find(&rows).Error)
	require.Len(t, rows, 2, "多个家庭分别反馈必须各占一行")
	byUser := map[uint64]string{}
	for _, r := range rows {
		byUser[r.UserID] = r.Reaction
	}
	assert.Equal(t, domain.TaskReactionThanks, byUser[testB1Parent])
	assert.Equal(t, domain.TaskReactionReceived, byUser[testB1ParentOther])
}

func TestReactToResponse_CrossFamilyRejected(t *testing.T) {
	f := setupB1(t)
	ctx := context.Background()

	_, err := f.svcT.RespondTask(ctx, f.tc, f.task.ID, f.c2.ID, domain.TaskRitualSticker, "star", "只给别人家孩子的回应", 0)
	require.NoError(t, err)
	otherResp := f.mainResponseFor(t, f.c2.ID)

	// 本家长在库里完全没有该回应的提交。
	_, err = f.svcP.ReactToResponse(ctx, f.pc, otherResp.ID, domain.TaskReactionChildHappy)
	require.ErrorIs(t, err, ErrParentForbidden)
	var n int64
	require.NoError(t, f.db.Model(&domain.TaskResponseReaction{}).
		Where("response_id = ?", otherResp.ID).Count(&n).Error)
	assert.Zero(t, n, "被拒时不能写反馈行")
}

func TestReactToResponse_SupplementAllowedAndVisible(t *testing.T) {
	f := setupB1(t)
	ctx := context.Background()

	_, err := f.svcT.RespondTask(ctx, f.tc, f.task.ID, f.c1.ID, domain.TaskRitualSticker, "star", "主回应", 0)
	require.NoError(t, err)
	got, err := f.svcT.SupplementResponse(ctx, f.tc, f.task.ID, f.c1.ID, domain.TaskRitualSticker, "flower", "补充", 0)
	require.NoError(t, err)
	supID, _ := got["id"].(uint64)

	// 补充回应也可以反馈：权限走补充关系表，而不是 checkin.response_id。
	_, err = f.svcP.ReactToResponse(ctx, f.pc, supID, domain.TaskReactionReceived)
	require.NoError(t, err)

	detail, err := f.svcP.GetTask(ctx, f.pc, f.task.ID)
	require.NoError(t, err)
	respList, _ := detail["responses"].([]map[string]any)
	require.Len(t, respList, 1)
	spList, _ := respList[0]["supplements"].([]map[string]any)
	require.Len(t, spList, 1)
	assert.Equal(t, domain.TaskReactionReceived, spList[0]["my_reaction"], "补充回应的反馈必须回显")
	assert.Equal(t, "", respList[0]["my_reaction"], "未反馈的主回应必须是空串")
}

func TestReactToResponse_InvalidValueAndForeignTenant(t *testing.T) {
	f := setupB1(t)
	ctx := context.Background()

	_, err := f.svcT.RespondTask(ctx, f.tc, f.task.ID, f.c1.ID, domain.TaskRitualSticker, "star", "", 0)
	require.NoError(t, err)
	resp := f.mainResponseFor(t, f.c1.ID)

	_, err = f.svcP.ReactToResponse(ctx, f.pc, resp.ID, "five_stars")
	require.ErrorIs(t, err, ErrParentInvalidParams)

	// 跨园所：即使 id 存在也必须当不存在。
	otherKG := *f.pc
	otherKG.KindergartenID = testB1KG + 1
	_, err = f.svcP.ReactToResponse(ctx, &otherKG, resp.ID, domain.TaskReactionChildHappy)
	require.ErrorIs(t, err, ErrParentNotFound)
}

// 唯一键 (response_id, user_id) 不含 deleted_at：软删行会占住唯一键。
// 这里钉住 upsert 必须把 deleted_at 清回来，否则「取消反馈后再反馈」会静默失效。
func TestReactToResponse_ResurrectsSoftDeletedRow(t *testing.T) {
	f := setupB1(t)
	ctx := context.Background()

	_, err := f.svcT.RespondTask(ctx, f.tc, f.task.ID, f.c1.ID, domain.TaskRitualSticker, "star", "", 0)
	require.NoError(t, err)
	resp := f.mainResponseFor(t, f.c1.ID)

	_, err = f.svcP.ReactToResponse(ctx, f.pc, resp.ID, domain.TaskReactionReceived)
	require.NoError(t, err)
	require.NoError(t, f.db.Where("response_id = ? AND user_id = ?", resp.ID, testB1Parent).
		Delete(&domain.TaskResponseReaction{}).Error)

	_, err = f.svcP.ReactToResponse(ctx, f.pc, resp.ID, domain.TaskReactionChildHappy)
	require.NoError(t, err)

	var visible int64
	require.NoError(t, f.db.Model(&domain.TaskResponseReaction{}).
		Where("response_id = ? AND user_id = ?", resp.ID, testB1Parent).Count(&visible).Error)
	assert.EqualValues(t, 1, visible, "再次反馈必须让软删行重新可见")

	var rows []domain.TaskResponseReaction
	require.NoError(t, f.db.Unscoped().Where("response_id = ? AND user_id = ?", resp.ID, testB1Parent).Find(&rows).Error)
	require.Len(t, rows, 1, "不能因为软删行占键而插入第二行")
	assert.Equal(t, domain.TaskReactionChildHappy, rows[0].Reaction)
	assert.False(t, rows[0].DeletedAt.Valid, "deleted_at 必须被清空")
}

// 媒体的归属校验与主回应一致：必须是**当前老师**上传的、biz_type=task 的 image/audio。
// 这条钉住 `user_id = ?` 条件——删掉它，老师就能挂别人上传的素材。
func TestSupplement_MediaMustBelongToTeacher(t *testing.T) {
	f := setupB1(t)
	ctx := context.Background()

	_, err := f.svcT.RespondTask(ctx, f.tc, f.task.ID, f.c1.ID, domain.TaskRitualSticker, "star", "主回应", 0)
	require.NoError(t, err)
	before := f.countTaskResponses(t)

	_, err = f.svcT.SupplementResponse(ctx, f.tc, f.task.ID, f.c1.ID, domain.TaskRitualVoice, "", "挂别人的素材", testB1File)
	require.ErrorIs(t, err, ErrTeacherInvalidParams, "不能引用别人上传的文件")
	assert.EqualValues(t, 0, f.countSupplements(t, f.c1.ID))
	assert.Equal(t, before, f.countTaskResponses(t), "被拒时不能留下 TaskResponse")

	// 自己的视频文件：类型不在 image/audio 白名单里，同样拒绝。
	_, err = f.svcT.SupplementResponse(ctx, f.tc, f.task.ID, f.c1.ID, domain.TaskRitualVoice, "", "挂视频", testB1Video)
	require.ErrorIs(t, err, ErrTeacherInvalidParams, "视频不能当补充回应的媒体")
	assert.EqualValues(t, 0, f.countSupplements(t, f.c1.ID))
	assert.Equal(t, before, f.countTaskResponses(t), "被拒时不能留下 TaskResponse")
}

// 反馈按**家长账号**隔离：同一回应下两个家庭各自反馈，彼此看不到对方的 my_reaction。
// 删掉 GetTask 里反馈查询的 `AND user_id = ?`，本测试会看到别人家的反馈而失败。
func TestParentGetTask_ReactionIsPerParentAccount(t *testing.T) {
	f := setupB1(t)
	ctx := context.Background()

	batch, err := f.svcT.BatchRespondTask(ctx, f.tc, f.task.ID, domain.TaskRitualSticker, "star", "本周汇总", 0)
	require.NoError(t, err)
	mainID, _ := batch["id"].(uint64)
	require.NotZero(t, mainID)

	_, err = f.svcP.ReactToResponse(ctx, f.pc, mainID, domain.TaskReactionChildHappy)
	require.NoError(t, err)

	pcOther := &ParentPortalContext{
		UserID:         testB1ParentOther,
		KindergartenID: testB1KG,
		CurrentStudent: &domain.Student{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testB1StudentOth}},
			ClassID:         testB1Class,
		},
	}
	detail, err := f.svcP.GetTask(ctx, pcOther, f.task.ID)
	require.NoError(t, err)
	respList, _ := detail["responses"].([]map[string]any)
	require.Len(t, respList, 1, "别人家的家长看到的也是同一条汇总回应")
	assert.Equal(t, "", respList[0]["my_reaction"], "别人家的反馈不能出现在本家长的 my_reaction 里")
}

// 同一孩子的**第二位家长**（有绑定关系、自己没有提交）也必须能反馈。
// 规格：家长只能对「与自己孩子提交相关的回应」反馈——同一位孩子的另一位家长属于这一类。
//
// 能证伪什么：退回「只认提交人本人」会 403。
// 不能证伪什么：本夹具里 period 回应同时覆盖了两个孩子，所以「按当前孩子判定」这种
// 实现在这里也会放行。那属于另一种可接受实现（handler 已保证 CurrentStudent 属于调用者），
// 故不在此区分；真正要拦的「完全忽略 userID」由 CrossFamilyRejected 覆盖。
func TestReactToResponse_CoParentOfSameChildAllowed(t *testing.T) {
	f := setupB1(t)
	ctx := context.Background()

	batch, err := f.svcT.BatchRespondTask(ctx, f.tc, f.task.ID, domain.TaskRitualSticker, "star", "本周汇总", 0)
	require.NoError(t, err)
	mainID, _ := batch["id"].(uint64)
	require.NotZero(t, mainID)

	pcCoParent := &ParentPortalContext{
		UserID:         testB1Parent2, // 与 testB1Parent 是同一个孩子
		KindergartenID: testB1KG,
		CurrentStudent: &domain.Student{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testB1Student}},
			ClassID:         testB1Class,
		},
	}
	_, err = f.svcP.ReactToResponse(ctx, pcCoParent, mainID, domain.TaskReactionThanks)
	require.NoError(t, err, "同一孩子的另一位家长应能反馈")

	var rows []domain.TaskResponseReaction
	require.NoError(t, f.db.Where("response_id = ?", mainID).Find(&rows).Error)
	require.Len(t, rows, 1)
	assert.Equal(t, testB1Parent2, rows[0].UserID, "反馈必须记在第二位家长自己名下")
	assert.Equal(t, domain.TaskReactionThanks, rows[0].Reaction)
	assert.Equal(t, testB1KG, rows[0].KindergartenID)
}

// 完全没有绑定关系、也没有提交的家长仍然被拒（跨家庭边界不能因为放开 co-parent 而丢失）。
func TestReactToResponse_UnboundParentForbidden(t *testing.T) {
	f := setupB1(t)
	ctx := context.Background()

	batch, err := f.svcT.BatchRespondTask(ctx, f.tc, f.task.ID, domain.TaskRitualSticker, "star", "本周汇总", 0)
	require.NoError(t, err)
	mainID, _ := batch["id"].(uint64)

	// 真实存在、但没有绑定、也没有提交的账号。
	pcUnbound := &ParentPortalContext{
		UserID:         testB1Stranger,
		KindergartenID: testB1KG,
		CurrentStudent: &domain.Student{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testB1Student}},
			ClassID:         testB1Class,
		},
	}
	_, err = f.svcP.ReactToResponse(ctx, pcUnbound, mainID, domain.TaskReactionReceived)
	require.ErrorIs(t, err, ErrParentForbidden)

	var n int64
	require.NoError(t, f.db.Model(&domain.TaskResponseReaction{}).Where("response_id = ?", mainID).Count(&n).Error)
	assert.Zero(t, n)
}

// 贴纸必须带 sticker_key、语音必须带音频媒体——否则家长端渲染/播放不出来
// （规格：贴纸 MUST 能被渲染；语音 MUST 能被播放）。
func TestRitualPayload_Validation(t *testing.T) {
	f := setupB1(t)
	ctx := context.Background()

	_, err := f.svcT.RespondTask(ctx, f.tc, f.task.ID, f.c1.ID, domain.TaskRitualSticker, "star", "主回应", 0)
	require.NoError(t, err)
	before := f.countTaskResponses(t)

	// 贴纸没有 key
	_, err = f.svcT.SupplementResponse(ctx, f.tc, f.task.ID, f.c1.ID, domain.TaskRitualSticker, "", "无 key", 0)
	require.ErrorIs(t, err, ErrTaskRitualPayload)

	// 语音没有媒体
	_, err = f.svcT.SupplementResponse(ctx, f.tc, f.task.ID, f.c1.ID, domain.TaskRitualVoice, "", "无媒体", 0)
	require.ErrorIs(t, err, ErrTaskRitualPayload)

	// 语音挂图片：能上传但不能播放
	_, err = f.svcT.SupplementResponse(ctx, f.tc, f.task.ID, f.c1.ID, domain.TaskRitualVoice, "", "挂图片", testB1Image)
	require.ErrorIs(t, err, ErrTeacherRitualMediaType)

	// 主回应与批量回应的语音也必须是音频——这两条路径此前会以 201 写入播不出来的回应。
	_, err = f.svcT.RespondTask(ctx, f.tc, f.task.ID, f.c2.ID, domain.TaskRitualVoice, "", "主回应挂图片", testB1Image)
	require.ErrorIs(t, err, ErrTeacherRitualMediaType, "主回应的语音不能挂图片")

	assert.Equal(t, before, f.countTaskResponses(t), "被拒时不能留下 TaskResponse")

	// 换成音频就能过
	_, err = f.svcT.SupplementResponse(ctx, f.tc, f.task.ID, f.c1.ID, domain.TaskRitualVoice, "", "正常语音", testB1Audio)
	require.NoError(t, err)

	// class_star 本来就不需要载体：空 sticker_key + 无媒体也必须放行。
	// 少了这条，把 validateRitualPayload 改成「class_star 也要 key」不会被任何用例发现。
	_, err = f.svcT.RespondTask(ctx, f.tc, f.task.ID, f.c2.ID, domain.TaskRitualClassStar, "", "班级星星", 0)
	require.NoError(t, err, "class_star 不需要 sticker_key 或媒体")
}

// 一条 period 回应覆盖同一孩子的多条提交时，家长端 responses[] 只能出现一次。
// 裸 join 会让它重复 N 次，本次新增的 supplements/my_reaction 也跟着重复。
func TestParentGetTask_PeriodResponseNotDuplicated(t *testing.T) {
	f := setupB1(t)
	ctx := context.Background()

	yesterday := f.c1.Date.AddDate(0, 0, -1)
	second := domain.TaskCheckIn{
		TaskID: f.task.ID, StudentID: testB1Student, UserID: testB1Parent,
		Date: yesterday, Content: "前一天的提交",
	}
	require.NoError(t, f.db.Create(&second).Error)

	resp := domain.TaskResponse{
		KindergartenID: testB1KG, TaskID: f.task.ID, TeacherID: testB1Teacher,
		Ritual: domain.TaskRitualSticker, StickerKey: "star",
		ResponseType: domain.TaskResponsePeriod, PeriodStart: &yesterday, PeriodEnd: &f.c1.Date,
	}
	require.NoError(t, f.db.Create(&resp).Error)
	require.NoError(t, f.db.Model(&domain.TaskCheckIn{}).
		Where("id IN ?", []uint64{f.c1.ID, second.ID}).Update("response_id", resp.ID).Error)

	// 给其中一条提交挂补充回应：若 join 去重失效，supplements 会跟着重复。
	_, err := f.svcT.SupplementResponse(ctx, f.tc, f.task.ID, second.ID, domain.TaskRitualSticker, "star", "补充一句", 0)
	require.NoError(t, err)

	detail, err := f.svcP.GetTask(ctx, f.pc, f.task.ID)
	require.NoError(t, err)
	respList, _ := detail["responses"].([]map[string]any)
	require.Len(t, respList, 1, "一条 period 回应覆盖 2 条提交也只能返回 1 个元素")
	assert.Equal(t, resp.ID, respList[0]["id"])
	spList, _ := respList[0]["supplements"].([]map[string]any)
	assert.Len(t, spList, 1, "补充回应也不能随 join 重复")
}

// 删除任务：提交/回应软删除（读路径看不到），补充关系与反馈硬删除（不留行）。
func TestDeleteTask_CascadeCleansSupplementsAndReactions(t *testing.T) {
	f := setupB1(t)
	ctx := context.Background()

	batch, err := f.svcT.BatchRespondTask(ctx, f.tc, f.task.ID, domain.TaskRitualSticker, "star", "本周汇总", 0)
	require.NoError(t, err)
	mainID, _ := batch["id"].(uint64)
	_, err = f.svcT.SupplementResponse(ctx, f.tc, f.task.ID, f.c1.ID, domain.TaskRitualSticker, "flower", "补充", 0)
	require.NoError(t, err)
	_, err = f.svcP.ReactToResponse(ctx, f.pc, mainID, domain.TaskReactionChildHappy)
	require.NoError(t, err)

	// 频率状态行与读标记也必须被硬删（deleteTaskCascade 里各自的删除语句）。
	require.NoError(t, f.db.Create(&domain.TaskCheckInState{TaskID: f.task.ID, StudentID: testB1Student}).Error)
	require.NoError(t, f.db.Create(&domain.TaskReadMark{
		KindergartenID: testB1KG, TaskID: f.task.ID, UserID: testB1Parent, StudentID: testB1Student,
	}).Error)

	// 别的班老师删不掉这个任务，且任务必须仍然存在。
	otherTC := &TeacherPortalContext{UserID: testB1Teacher, KindergartenID: testB1KG, ClassID: testB1ClassOther}
	require.ErrorIs(t, f.svcT.DeleteTask(ctx, otherTC, f.task.ID), ErrTeacherNotFound)
	var stillThere int64
	require.NoError(t, f.db.Model(&domain.ParentTask{}).Where("id = ?", f.task.ID).Count(&stillThere).Error)
	require.EqualValues(t, 1, stillThere, "越权删除失败后任务必须还在")

	checkinIDs := []uint64{f.c1.ID, f.c2.ID}
	require.NoError(t, f.svcT.DeleteTask(ctx, f.tc, f.task.ID))

	// 提交/回应是软删除：活着的行必须为 0。
	var liveResponses, liveCheckins int64
	require.NoError(t, f.db.Model(&domain.TaskResponse{}).Where("task_id = ?", f.task.ID).Count(&liveResponses).Error)
	require.NoError(t, f.db.Model(&domain.TaskCheckIn{}).Where("task_id = ?", f.task.ID).Count(&liveCheckins).Error)
	assert.Zero(t, liveResponses, "删除后列表/统计不能再看到回应")
	assert.Zero(t, liveCheckins, "删除后列表/统计不能再看到提交")

	// 补充关系与反馈是硬删除：连软删行都不该留。
	var orphanSupp, orphanReaction int64
	require.NoError(t, f.db.Unscoped().Model(&domain.TaskResponseSupplement{}).
		Where("check_in_id IN ?", checkinIDs).Count(&orphanSupp).Error)
	require.NoError(t, f.db.Unscoped().Model(&domain.TaskResponseReaction{}).
		Where("response_id = ?", mainID).Count(&orphanReaction).Error)
	assert.Zero(t, orphanSupp, "补充关系不能留孤儿")
	assert.Zero(t, orphanReaction, "反馈不能留孤儿")

	var orphanState, orphanMark int64
	require.NoError(t, f.db.Unscoped().Model(&domain.TaskCheckInState{}).
		Where("task_id = ?", f.task.ID).Count(&orphanState).Error)
	require.NoError(t, f.db.Unscoped().Model(&domain.TaskReadMark{}).
		Where("task_id = ?", f.task.ID).Count(&orphanMark).Error)
	assert.Zero(t, orphanState, "频率状态行不能留孤儿")
	assert.Zero(t, orphanMark, "读标记不能留孤儿")
}

// 补充回应的权限与主回应同口径：co-parent 也要能反馈。
// 这条专门打到 reactionAllowed 第三段查询里的 `OR binding` 分支——
// 在此之前，该分支没有被任何用例执行过（TestReactToResponse_SupplementAllowedAndVisible 只测了提交人本人）。
func TestReactToResponse_SupplementCoParentAllowed(t *testing.T) {
	f := setupB1(t)
	ctx := context.Background()

	_, err := f.svcT.RespondTask(ctx, f.tc, f.task.ID, f.c1.ID, domain.TaskRitualSticker, "star", "主回应", 0)
	require.NoError(t, err)
	got, err := f.svcT.SupplementResponse(ctx, f.tc, f.task.ID, f.c1.ID, domain.TaskRitualSticker, "flower", "补充", 0)
	require.NoError(t, err)
	supID, _ := got["id"].(uint64)

	pcCoParent := &ParentPortalContext{
		UserID:         testB1Parent2,
		KindergartenID: testB1KG,
		CurrentStudent: &domain.Student{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testB1Student}},
			ClassID:         testB1Class,
		},
	}
	_, err = f.svcP.ReactToResponse(ctx, pcCoParent, supID, domain.TaskReactionChildHappy)
	require.NoError(t, err, "co-parent 也要能对补充回应反馈")

	var rows []domain.TaskResponseReaction
	require.NoError(t, f.db.Where("response_id = ?", supID).Find(&rows).Error)
	require.Len(t, rows, 1)
	assert.Equal(t, testB1Parent2, rows[0].UserID)
}
