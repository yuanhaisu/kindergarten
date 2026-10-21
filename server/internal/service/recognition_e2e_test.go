package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
)

// 4.6 端到端：照片入队 → worker 认领 → mock 识别 → 建议落库（status=0）。
// mock provider 按图片内容首行指令返回命中：本测试注入 fetch 钩子，令「取图」返回
// MOCK_STUDENTS 指令，从而不依赖真实网络与云服务。
//
// D11 起写建议前按记录范围过滤（全班 → 本班在读），故照片必须绑定记录、
// 命中的学生必须在记录范围内——这也是生产链路的实际形态（只在发布路径入队）。

// seedRQLogMedia 建一条全班记录 + 绑定其中的照片（范围过滤需要记录归属）。
// 先清残留（含任务与关联行）：Enqueue 对 media_id 幂等，残留的 done 任务会让本轮认领不到。
func seedRQLogMedia(t *testing.T, db *gorm.DB, logID, mid uint64) {
	t.Helper()
	db.Unscoped().Where("media_id = ?", mid).Delete(&domain.GrowthMediaStudent{})
	db.Unscoped().Where("media_id = ?", mid).Delete(&domain.RecognitionJob{})
	db.Unscoped().Where("id = ?", logID).Delete(&domain.GrowthDailyLog{})
	db.Unscoped().Where("id = ?", mid).Delete(&domain.GrowthMedia{})
	require.NoError(t, db.Create(&domain.GrowthDailyLog{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: logID}},
		KindergartenID:  4, ClassID: 6, CreatorID: frSeedUploader(t, db), Date: time.Now(),
		Content: "识别队列范围测试", TargetType: 1,
	}).Error)
	require.NoError(t, db.Create(&domain.GrowthMedia{
		BaseModel: domain.BaseModel{ID: mid}, KindergartenID: 4, ClassID: 6, LogID: &logID,
		FileURL: "http://mock.local/rq.png", FileType: "image",
	}).Error)
}

// rqClassStudent 取班级 6 的一名在读学生（mock 命中的学生须在记录范围内才产建议）。
func rqClassStudent(t *testing.T, db *gorm.DB) uint64 {
	t.Helper()
	var st domain.Student
	if err := db.Where("kindergarten_id = ? AND class_id = ? AND status = 1", 4, 6).First(&st).Error; err != nil {
		t.Skip("本机库无班级 6 的在读学生，跳过依赖记录范围的用例")
	}
	return st.ID
}

// seedRQIndividualLogMedia 建一条个体记录（target_type=2）+ 关联学生 + 绑定照片。
func seedRQIndividualLogMedia(t *testing.T, db *gorm.DB, logID, mid, studentID uint64) {
	t.Helper()
	db.Unscoped().Where("media_id = ?", mid).Delete(&domain.GrowthMediaStudent{})
	db.Unscoped().Where("media_id = ?", mid).Delete(&domain.RecognitionJob{})
	db.Unscoped().Where("log_id = ?", logID).Delete(&domain.GrowthLogStudent{})
	db.Unscoped().Where("id = ?", logID).Delete(&domain.GrowthDailyLog{})
	db.Unscoped().Where("id = ?", mid).Delete(&domain.GrowthMedia{})
	require.NoError(t, db.Create(&domain.GrowthDailyLog{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: logID}},
		KindergartenID:  4, ClassID: 6, CreatorID: frSeedUploader(t, db), Date: time.Now(),
		Content: "识别队列个体记录测试", TargetType: 2,
	}).Error)
	require.NoError(t, db.Create(&domain.GrowthLogStudent{LogID: logID, StudentID: studentID}).Error)
	require.NoError(t, db.Create(&domain.GrowthMedia{
		BaseModel: domain.BaseModel{ID: mid}, KindergartenID: 4, ClassID: 6, LogID: &logID,
		FileURL: "http://mock.local/rq-ind.png", FileType: "image",
	}).Error)
}

func rqCleanup(t *testing.T, db *gorm.DB, logID, mid uint64) {
	t.Helper()
	db.Unscoped().Where("media_id = ?", mid).Delete(&domain.GrowthMediaStudent{})
	db.Unscoped().Where("media_id = ?", mid).Delete(&domain.RecognitionJob{})
	db.Unscoped().Where("id = ?", mid).Delete(&domain.GrowthMedia{})
	db.Unscoped().Where("log_id = ?", logID).Delete(&domain.GrowthLogStudent{})
	db.Unscoped().Where("id = ?", logID).Delete(&domain.GrowthDailyLog{})
}

func TestRecognitionE2E_EnqueueToSuggestion(t *testing.T) {
	db := openRQTestDB(t)
	ctx := context.Background()

	mid := uint64(rqTestBase + 100)
	logID := uint64(rqTestBase + 99)
	studentID := rqClassStudent(t, db)
	seedRQLogMedia(t, db, logID, mid)
	defer rqCleanup(t, db, logID, mid)

	q := testQueue(db, config.FaceConfig{Enabled: true, SuggestLow: 70, SuggestHigh: 80, QPS: 100})
	// 钩子：取图返回 mock 指令（首行）——控制 mock provider 的输出
	q.fetch = func(ctx context.Context, url string) ([]byte, error) {
		return []byte("MOCK_STUDENTS:" + uitoa(studentID) + "\n<binary>"), nil
	}

	require.NoError(t, q.Enqueue(ctx, 4, mid))
	ok := q.processOne(ctx, "e2e")
	require.True(t, ok, "应认领到任务并处理")

	var row domain.GrowthMediaStudent
	require.NoError(t, db.Where("media_id = ? AND student_id = ?", mid, studentID).First(&row).Error)
	require.NotNil(t, row.Status)
	assert.Equal(t, domain.MediaStudentStatusSuggested, *row.Status, "识别结果应为建议态（待老师确认）")
	assert.True(t, row.IsAuto)

	var job domain.RecognitionJob
	require.NoError(t, db.Where("media_id = ?", mid).First(&job).Error)
	assert.Equal(t, domain.JobStatusDone, job.Status)
}

// D11 分型：人脸数 ≥ 阈值（大合影）→ 不产建议，face_count 落任务行（老师端据此呈现「全班」一键）。
func TestRecognitionQueue_GroupPhotoProducesNoSuggestions(t *testing.T) {
	db := openRQTestDB(t)
	ctx := context.Background()

	mid := uint64(rqTestBase + 120)
	logID := uint64(rqTestBase + 119)
	studentID := rqClassStudent(t, db)
	seedRQLogMedia(t, db, logID, mid)
	defer rqCleanup(t, db, logID, mid)

	// 阈值缺省 6：8 张人脸即分型为大合影
	q := testQueue(db, config.FaceConfig{Enabled: true, SuggestLow: 70, SuggestHigh: 80, QPS: 100})
	q.fetch = func(ctx context.Context, url string) ([]byte, error) {
		return []byte("MOCK_FACES:8\nMOCK_STUDENTS:" + uitoa(studentID) + "\n<binary>"), nil
	}

	require.NoError(t, q.Enqueue(ctx, 4, mid))
	require.True(t, q.processOne(ctx, "e2e-gp"))

	var n int64
	db.Model(&domain.GrowthMediaStudent{}).Where("media_id = ?", mid).Count(&n)
	assert.Zero(t, n, "大合影不产逐人建议（逐人名单在人脸上限下注定不完整）")

	var job domain.RecognitionJob
	require.NoError(t, db.Where("media_id = ?", mid).First(&job).Error)
	assert.Equal(t, domain.JobStatusDone, job.Status, "大合影仍是成功任务（只是无需建议）")
	assert.Equal(t, 8, job.FaceCount, "face_count 必须落库（老师端分型呈现的数据面）")
}

// 复审裁定（2026-10-16）：大合影分型仅对**全班记录**生效——个体记录里的大合影保留建议
// （建议本就限定在该记录的目标孩子，数量有限，逐人确认不是伪工作）；face_count 仍照常落库。
func TestRecognitionQueue_GroupPhotoOnIndividualRecordKeepsSuggestions(t *testing.T) {
	db := openRQTestDB(t)
	ctx := context.Background()

	mid := uint64(rqTestBase + 140)
	logID := uint64(rqTestBase + 139)
	studentID := rqClassStudent(t, db)
	seedRQIndividualLogMedia(t, db, logID, mid, studentID)
	defer rqCleanup(t, db, logID, mid)

	q := testQueue(db, config.FaceConfig{Enabled: true, SuggestLow: 70, SuggestHigh: 80, QPS: 100})
	q.fetch = func(ctx context.Context, url string) ([]byte, error) {
		return []byte("MOCK_FACES:8\nMOCK_STUDENTS:" + uitoa(studentID) + "\n<binary>"), nil
	}

	require.NoError(t, q.Enqueue(ctx, 4, mid))
	require.True(t, q.processOne(ctx, "e2e-ind"))

	var rows []domain.GrowthMediaStudent
	require.NoError(t, db.Where("media_id = ?", mid).Find(&rows).Error)
	require.Len(t, rows, 1, "个体记录的大合影保留建议（仅目标孩子）")
	assert.Equal(t, studentID, rows[0].StudentID)

	var job domain.RecognitionJob
	require.NoError(t, db.Where("media_id = ?", mid).First(&job).Error)
	assert.Equal(t, 8, job.FaceCount, "人脸数照常落库（老师端据此展示分型信息）")
}

// D11 范围过滤：跨班相似脸的命中不得产生建议（否则老师会看到本班没有的孩子）。
func TestRecognitionQueue_OutOfScopeMatchesFiltered(t *testing.T) {
	db := openRQTestDB(t)
	ctx := context.Background()

	mid := uint64(rqTestBase + 130)
	logID := uint64(rqTestBase + 129)
	inClass := rqClassStudent(t, db)
	outOfScope := uint64(rqTestBase + 999) // 不在本班在读名单
	seedRQLogMedia(t, db, logID, mid)
	defer rqCleanup(t, db, logID, mid)

	q := testQueue(db, config.FaceConfig{Enabled: true, SuggestLow: 70, SuggestHigh: 80, QPS: 100})
	q.fetch = func(ctx context.Context, url string) ([]byte, error) {
		// 两条命中均高于 SuggestLow（90 / 72）：越界的仍须被滤掉
		return []byte("MOCK_STUDENTS:" + uitoa(inClass) + "," + uitoa(outOfScope) + "\n<binary>"), nil
	}

	require.NoError(t, q.Enqueue(ctx, 4, mid))
	require.True(t, q.processOne(ctx, "e2e-scope"))

	var rows []domain.GrowthMediaStudent
	require.NoError(t, db.Where("media_id = ?", mid).Find(&rows).Error)
	require.Len(t, rows, 1, "仅本班在读学生的命中落建议")
	assert.Equal(t, inClass, rows[0].StudentID)
}

// 失败路径：取图失败 → 退避重试（保持 pending 且 next_run_at 后移）
func TestRecognitionE2E_FetchFailureRetries(t *testing.T) {
	db := openRQTestDB(t)
	ctx := context.Background()
	mid := uint64(rqTestBase + 110)
	db.Unscoped().Where("media_id = ?", mid).Delete(&domain.RecognitionJob{})
	db.Unscoped().Where("id = ?", mid).Delete(&domain.GrowthMedia{})
	require.NoError(t, db.Create(&domain.GrowthMedia{
		BaseModel: domain.BaseModel{ID: mid}, KindergartenID: 4, ClassID: 6,
		FileURL: "http://mock.local/fail.png", FileType: "image",
	}).Error)

	cfg := config.FaceConfig{Enabled: true, QPS: 100}
	q := testQueue(db, cfg)
	q.fetch = func(ctx context.Context, url string) ([]byte, error) {
		return nil, assertErr("network down")
	}
	require.NoError(t, q.Enqueue(ctx, 4, mid))
	ok := q.processOne(ctx, "e2e2")
	require.True(t, ok)

	var job domain.RecognitionJob
	require.NoError(t, db.Where("media_id = ?", mid).First(&job).Error)
	assert.Equal(t, domain.JobStatusPending, job.Status, "网络类失败应回退 pending 等待重试")
	assert.Equal(t, 1, job.Retries)
	assert.Greater(t, job.NextRunAt, time.Now().Unix(), "退避：下次执行时间应推后")

	db.Unscoped().Where("media_id = ?", mid).Delete(&domain.RecognitionJob{})
	db.Unscoped().Where("id = ?", mid).Delete(&domain.GrowthMedia{})
}

type assertErr string

func (e assertErr) Error() string { return string(e) }

func uitoa(v uint64) string {
	if v == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	return string(b[i:])
}
