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
	"kindergarten-server/internal/repository"
)

// 晨检闭环回归：标签服务端派生、越界拒绝、条件更新、家长通知与投递失败可见。
// 断言库里真实状态，不只看接口返回值。

const (
	hcTestKG      uint64 = 991101
	hcTestClass   uint64 = 991102
	hcTestStudent uint64 = 991103
	hcTestDoctor  uint64 = 991104
	hcTestParent  uint64 = 991105
	hcTestPickup  uint64 = 991106
	hcTestCheck   uint64 = 991107
	hcTestFile    uint64 = 991109
)

func openHealthTestDB(t *testing.T) *gorm.DB {
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

type healthPolicyStub struct {
	policy HealthPolicy
	err    error
}

func (s healthPolicyStub) GetHealthPolicy(ctx context.Context, kgID uint64) (HealthPolicy, error) {
	return s.policy, s.err
}

type healthFixture struct {
	db  *gorm.DB
	svc *HealthCheckService
}

// withPolicy 用指定策略构造一个服务，复用同一套夹具。
func (f *healthFixture) withPolicy(policy HealthPolicy) *HealthCheckService {
	return NewHealthCheckService(
		f.db,
		repository.NewHealthCheckRepo(f.db),
		repository.NewMessageRepo(f.db),
		repository.NewFileRepo(f.db),
		healthPolicyStub{policy: policy},
	)
}

func newHealthFixture(t *testing.T) *healthFixture {
	t.Helper()
	db := openHealthTestDB(t)

	cleanup := func() {
		db.Unscoped().Where("user_id = ?", hcTestParent).Delete(&domain.Message{})
		db.Unscoped().Where("check_id = ?", hcTestCheck).Delete(&domain.HealthCheckMedia{})
		db.Unscoped().Where("id = ?", hcTestFile).Delete(&domain.File{})
		db.Unscoped().Where("id = ?", hcTestCheck).Delete(&domain.MorningHealthCheck{})
		db.Unscoped().Where("id = ?", hcTestPickup).Delete(&domain.PickupRecord{})
		db.Unscoped().Where("student_id = ?", hcTestStudent).Delete(&domain.StudentParent{})
		db.Unscoped().Where("id = ?", hcTestStudent).Delete(&domain.Student{})
		db.Unscoped().Where("id = ?", hcTestClass).Delete(&domain.Class{})
		db.Unscoped().Where("id IN ?", []uint64{hcTestParent, hcTestDoctor}).Delete(&domain.User{})
		db.Unscoped().Where("id = ?", hcTestKG).Delete(&domain.Kindergarten{})
	}
	t.Cleanup(cleanup)
	cleanup()

	require.NoError(t, db.Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: hcTestKG},
		Name:      "晨检测试园",
	}).Error)
	require.NoError(t, db.Create(&domain.User{
		BaseModel:    domain.BaseModel{ID: hcTestDoctor},
		Nickname:     "测试保健医",
		Phone:        "13900009901",
		PasswordHash: "x",
	}).Error)
	require.NoError(t, db.Create(&domain.User{
		BaseModel:    domain.BaseModel{ID: hcTestParent},
		Nickname:     "测试家长",
		Phone:        "13900009902",
		PasswordHash: "x",
	}).Error)
	require.NoError(t, db.Create(&domain.Class{
		BaseModel:      domain.BaseModel{ID: hcTestClass},
		KindergartenID: hcTestKG,
		Name:           "晨检测试班",
		Grade:          "小班",
	}).Error)
	require.NoError(t, db.Create(&domain.Student{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: hcTestStudent}},
		KindergartenID:  hcTestKG,
		ClassID:         hcTestClass,
		Name:            "测试幼儿",
		Birthday:        time.Date(2021, 1, 1, 0, 0, 0, 0, time.Local),
		EnrollDate:      time.Date(2024, 9, 1, 0, 0, 0, 0, time.Local),
		Status:          1,
	}).Error)
	require.NoError(t, db.Create(&domain.StudentParent{
		BaseModel: domain.BaseModel{ID: hcTestCheck + 1},
		StudentID: hcTestStudent,
		UserID:    hcTestParent,
		Relation:  domain.RelationMother,
	}).Error)
	require.NoError(t, db.Create(&domain.PickupRecord{
		BaseModel:      domain.BaseModel{ID: hcTestPickup},
		KindergartenID: hcTestKG,
		ClassID:        hcTestClass,
		StudentID:      hcTestStudent,
		Date:           time.Now(),
		Type:           domain.PickupTypeDropOff,
		Method:         domain.PickupMethodManual,
		OperatorID:     hcTestDoctor,
		RecordedAt:     time.Now(),
	}).Error)
	require.NoError(t, createPendingCheck(t, db))

	return &healthFixture{
		db:  db,
		svc: NewHealthCheckService(db, repository.NewHealthCheckRepo(db), repository.NewMessageRepo(db), repository.NewFileRepo(db),
			healthPolicyStub{policy: HealthPolicy{FeverThreshold: DefaultFeverThreshold}}),
	}
}

func createPendingCheck(t *testing.T, db *gorm.DB) error {
	t.Helper()
	return db.Create(&domain.MorningHealthCheck{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: hcTestCheck}},
		KindergartenID:  hcTestKG,
		ClassID:         hcTestClass,
		StudentID:       hcTestStudent,
		PickupRecordID:  hcTestPickup,
		Date:            time.Now().Format("2006-01-02"),
		CheckStatus:     domain.HealthCheckPending,
	}).Error
}

func int8p(v int8) *int8 { return &v }

func TestDeriveSymptomTags(t *testing.T) {
	temp := 37.5
	check := &domain.MorningHealthCheck{
		Temperature:  &temp,
		ThroatStatus: 2,
		HandStatus:   2,
		MouthStatus:  3, // 其他：合法但不产标签
		EyeStatus:    1,
		SkinStatus:   0,
	}
	assert.Equal(t, "fever,swollen_throat,hand_blister,red_eye", deriveSymptomTags(check, DefaultFeverThreshold))

	normalTemp := 36.5
	normal := &domain.MorningHealthCheck{Temperature: &normalTemp}
	assert.Equal(t, "", deriveSymptomTags(normal, DefaultFeverThreshold))

	// 阈值边界：37.2 在 37.2 阈值下算发烧，在 37.5 阈值下不算
	edgeTemp := 37.2
	edge := &domain.MorningHealthCheck{Temperature: &edgeTemp}
	assert.Equal(t, "fever", deriveSymptomTags(edge, 37.2))
	assert.Equal(t, "", deriveSymptomTags(edge, 37.5))
}

func TestValidateHealthItems(t *testing.T) {
	require.NoError(t, validateHealthItems(SubmitHealthCheckReq{
		ThroatStatus: int8p(2), HandStatus: int8p(3), MouthStatus: int8p(3),
		EyeStatus: int8p(2), SkinStatus: int8p(3), OverallStatus: 2,
	}))

	err := validateHealthItems(SubmitHealthCheckReq{ThroatStatus: int8p(3), OverallStatus: 0})
	require.ErrorIs(t, err, ErrHealthCheckInvalidItem)

	err = validateHealthItems(SubmitHealthCheckReq{HandStatus: int8p(-1), OverallStatus: 0})
	require.ErrorIs(t, err, ErrHealthCheckInvalidItem)

	err = validateHealthItems(SubmitHealthCheckReq{OverallStatus: 3})
	require.ErrorIs(t, err, ErrHealthCheckInvalidItem)
}

func TestHealthCheckSubmit_ObserveNotifiesParent(t *testing.T) {
	f := newHealthFixture(t)
	temp := 37.6

	result, err := f.svc.SubmitCheck(context.Background(), hcTestKG, hcTestDoctor, hcTestCheck, SubmitHealthCheckReq{
		Temperature:   &temp,
		ThroatStatus:  int8p(2),
		OverallStatus: int8(domain.HealthOverallObserve),
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.NotifiedCount)
	assert.Equal(t, 0, result.NotifyFailedCount)
	assert.Equal(t, "fever,swollen_throat", result.SymptomTags)

	var msg domain.Message
	require.NoError(t, f.db.Where("user_id = ? AND type = ?", hcTestParent, string(domain.MessageTypeSystem)).
		First(&msg).Error)
	assert.Contains(t, msg.Title, "需观察")
	assert.Equal(t, hcTestCheck, msg.ReferenceID)
}

func TestHealthCheckSubmit_HealthyDoesNotNotify(t *testing.T) {
	f := newHealthFixture(t)
	_, err := f.svc.SubmitCheck(context.Background(), hcTestKG, hcTestDoctor, hcTestCheck, SubmitHealthCheckReq{
		OverallStatus: int8(domain.HealthOverallHealthy),
	})
	require.NoError(t, err)

	var count int64
	require.NoError(t, f.db.Model(&domain.Message{}).Where("user_id = ?", hcTestParent).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

func TestHealthCheckSubmit_RejectIsolatedUrgentTitle(t *testing.T) {
	f := newHealthFixture(t)
	isolated := true
	_, err := f.svc.SubmitCheck(context.Background(), hcTestKG, hcTestDoctor, hcTestCheck, SubmitHealthCheckReq{
		OverallStatus: int8(domain.HealthOverallReject),
		IsIsolated:    &isolated,
	})
	require.NoError(t, err)

	var msg domain.Message
	require.NoError(t, f.db.Where("user_id = ? AND type = ?", hcTestParent, string(domain.MessageTypeSystem)).
		First(&msg).Error)
	assert.Contains(t, msg.Title, "【紧急】")
	assert.Contains(t, msg.Title, "劝返就医")
}

func TestHealthCheckSubmit_DuplicateRejectedAndNoDuplicateMessage(t *testing.T) {
	f := newHealthFixture(t)
	req := SubmitHealthCheckReq{OverallStatus: int8(domain.HealthOverallObserve)}
	_, err := f.svc.SubmitCheck(context.Background(), hcTestKG, hcTestDoctor, hcTestCheck, req)
	require.NoError(t, err)

	_, err = f.svc.SubmitCheck(context.Background(), hcTestKG, hcTestDoctor, hcTestCheck, req)
	require.ErrorIs(t, err, ErrHealthCheckAlreadyDone)

	var count int64
	require.NoError(t, f.db.Model(&domain.Message{}).Where("user_id = ?", hcTestParent).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestHealthCheckSubmit_NotifyFailureKeepsConclusion(t *testing.T) {
	f := newHealthFixture(t)

	// 用一条已关闭的连接构造消息仓储，模拟投递失败；晨检主库仍然可用。
	cfg, err := config.Load(filepath.Join("..", "..", "configs", "app.ini"))
	require.NoError(t, err)
	brokenDB, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	sqlDB, err := brokenDB.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	svc := NewHealthCheckService(f.db, repository.NewHealthCheckRepo(f.db), repository.NewMessageRepo(brokenDB), repository.NewFileRepo(f.db),
		healthPolicyStub{policy: HealthPolicy{FeverThreshold: DefaultFeverThreshold}})
	result, err := svc.SubmitCheck(context.Background(), hcTestKG, hcTestDoctor, hcTestCheck, SubmitHealthCheckReq{
		OverallStatus: int8(domain.HealthOverallObserve),
	})
	require.NoError(t, err)
	assert.Equal(t, 0, result.NotifiedCount)
	assert.Equal(t, 1, result.NotifyFailedCount)

	var check domain.MorningHealthCheck
	require.NoError(t, f.db.Where("id = ?", hcTestCheck).First(&check).Error)
	assert.Equal(t, domain.HealthCheckCompleted, check.CheckStatus)
}

func TestHealthCheckSubmit_RejectDoesNotMarkPickupAbnormal(t *testing.T) {
	f := newHealthFixture(t)
	_, err := f.svc.SubmitCheck(context.Background(), hcTestKG, hcTestDoctor, hcTestCheck, SubmitHealthCheckReq{
		OverallStatus: int8(domain.HealthOverallReject),
	})
	require.NoError(t, err)

	var pr domain.PickupRecord
	require.NoError(t, f.db.Where("id = ?", hcTestPickup).First(&pr).Error)
	assert.False(t, pr.IsAbnormal)
	assert.Empty(t, pr.AbnormalReason)
}

func TestHealthCheckSubmit_InvalidItemRejected(t *testing.T) {
	f := newHealthFixture(t)
	_, err := f.svc.SubmitCheck(context.Background(), hcTestKG, hcTestDoctor, hcTestCheck, SubmitHealthCheckReq{
		ThroatStatus:  int8p(3),
		OverallStatus: int8(domain.HealthOverallHealthy),
	})
	require.ErrorIs(t, err, ErrHealthCheckInvalidItem)

	var check domain.MorningHealthCheck
	require.NoError(t, f.db.Where("id = ?", hcTestCheck).First(&check).Error)
	assert.Equal(t, domain.HealthCheckPending, check.CheckStatus)
}

func createHealthTestFile(t *testing.T, db *gorm.DB, userID uint64, bizType string) {
	t.Helper()
	require.NoError(t, db.Create(&domain.File{
		BaseModel: domain.BaseModel{ID: hcTestFile},
		UserID:    userID,
		BizType:   bizType,
		FileName:  "throat.jpg",
		FileURL:   "https://oss.test/throat.jpg",
		ThumbURL:  "https://oss.test/throat.jpg?x-oss-process=image/resize,w_200",
		FileSize:  1024,
		FileType:  "image",
	}).Error)
}

func TestHealthCheckMedia_AttachWithFileID(t *testing.T) {
	f := newHealthFixture(t)
	createHealthTestFile(t, f.db, hcTestDoctor, "health")

	media, err := f.svc.AddMedia(context.Background(), hcTestKG, hcTestCheck, hcTestFile, hcTestDoctor, "throat", "咽部红肿")
	require.NoError(t, err)
	assert.Equal(t, "https://oss.test/throat.jpg", media.FileURL)

	var row domain.HealthCheckMedia
	require.NoError(t, f.db.Where("check_id = ?", hcTestCheck).First(&row).Error)
	require.NotNil(t, row.FileID)
	assert.Equal(t, hcTestFile, *row.FileID)
	assert.Equal(t, "咽部红肿", row.RefDesc)

	detail, err := f.svc.GetDetail(context.Background(), hcTestKG, hcTestCheck)
	require.NoError(t, err)
	require.Len(t, detail.Media, 1)
	assert.Equal(t, "throat", detail.Media[0].Category)
}

func TestHealthCheckMedia_RejectsCrossTenant(t *testing.T) {
	f := newHealthFixture(t)
	createHealthTestFile(t, f.db, hcTestDoctor, "health")

	_, err := f.svc.AddMedia(context.Background(), 991999, hcTestCheck, hcTestFile, hcTestDoctor, "throat", "")
	require.ErrorIs(t, err, ErrHealthCheckNotFound)

	var count int64
	require.NoError(t, f.db.Model(&domain.HealthCheckMedia{}).Where("check_id = ?", hcTestCheck).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

func TestHealthCheckMedia_RejectsForeignFile(t *testing.T) {
	f := newHealthFixture(t)
	createHealthTestFile(t, f.db, hcTestParent, "health")

	// 他人上传的文件
	_, err := f.svc.AddMedia(context.Background(), hcTestKG, hcTestCheck, hcTestFile, hcTestDoctor, "throat", "")
	require.ErrorIs(t, err, ErrHealthCheckMediaInvalid)

	// 非晨检业务类型
	require.NoError(t, f.db.Model(&domain.File{}).Where("id = ?", hcTestFile).
		Updates(map[string]any{"user_id": hcTestDoctor, "biz_type": "growth"}).Error)
	_, err = f.svc.AddMedia(context.Background(), hcTestKG, hcTestCheck, hcTestFile, hcTestDoctor, "throat", "")
	require.ErrorIs(t, err, ErrHealthCheckMediaInvalid)

	// 未知拍摄部位
	require.NoError(t, f.db.Model(&domain.File{}).Where("id = ?", hcTestFile).
		Update("biz_type", "health").Error)
	_, err = f.svc.AddMedia(context.Background(), hcTestKG, hcTestCheck, hcTestFile, hcTestDoctor, "head", "")
	require.ErrorIs(t, err, ErrHealthCheckMediaInvalid)
}

func TestHealthCheckTenantIsolation(t *testing.T) {
	f := newHealthFixture(t)

	_, err := f.svc.GetDetail(context.Background(), 991999, hcTestCheck)
	require.ErrorIs(t, err, ErrHealthCheckNotFound)

	_, err = f.svc.SubmitCheck(context.Background(), 991999, hcTestDoctor, hcTestCheck, SubmitHealthCheckReq{
		OverallStatus: int8(domain.HealthOverallHealthy),
	})
	require.ErrorIs(t, err, ErrHealthCheckNotFound)

	var check domain.MorningHealthCheck
	require.NoError(t, f.db.Where("id = ?", hcTestCheck).First(&check).Error)
	assert.Equal(t, domain.HealthCheckPending, check.CheckStatus)
}

func newParentHealthService(db *gorm.DB) *ParentPortalService {
	return NewParentPortalService(db, nil, nil, nil, nil, nil, nil, nil)
}

func parentHealthContext() *ParentPortalContext {
	return &ParentPortalContext{
		UserID:         hcTestParent,
		KindergartenID: hcTestKG,
		Students: []domain.Student{{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: hcTestStudent}},
		}},
	}
}

func TestParentHealthRecords_OnlyCompletedWithinWindow(t *testing.T) {
	f := newHealthFixture(t)
	svc := newParentHealthService(f.db)
	pc := parentHealthContext()

	// 待检不返回
	records, err := svc.ListHealthRecords(context.Background(), pc, hcTestStudent)
	require.NoError(t, err)
	assert.Len(t, records, 0)

	today := time.Now().Format("2006-01-02")
	require.NoError(t, f.db.Model(&domain.MorningHealthCheck{}).Where("id = ?", hcTestCheck).
		Updates(map[string]any{"check_status": domain.HealthCheckCompleted, "date": today}).Error)
	records, err = svc.ListHealthRecords(context.Background(), pc, hcTestStudent)
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, today, records[0].CheckDate)

	// 第 6 天仍在窗口内
	inside := time.Now().AddDate(0, 0, -6).Format("2006-01-02")
	require.NoError(t, f.db.Model(&domain.MorningHealthCheck{}).Where("id = ?", hcTestCheck).
		Update("date", inside).Error)
	records, err = svc.ListHealthRecords(context.Background(), pc, hcTestStudent)
	require.NoError(t, err)
	assert.Len(t, records, 1)

	// 第 7 天在窗口外
	outside := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	require.NoError(t, f.db.Model(&domain.MorningHealthCheck{}).Where("id = ?", hcTestCheck).
		Update("date", outside).Error)
	records, err = svc.ListHealthRecords(context.Background(), pc, hcTestStudent)
	require.NoError(t, err)
	assert.Len(t, records, 0)
}

func TestParentHealthRecords_ReadableItems(t *testing.T) {
	f := newHealthFixture(t)
	temp := 38.2
	require.NoError(t, f.db.Model(&domain.MorningHealthCheck{}).Where("id = ?", hcTestCheck).
		Updates(map[string]any{
			"check_status":   domain.HealthCheckCompleted,
			"date":           time.Now().Format("2006-01-02"),
			"temperature":    temp,
			"throat_status":  2,
			"hand_status":    2,
			"mouth_status":   3,
			"eye_status":     0,
			"skin_status":    0,
			"overall_status": domain.HealthOverallObserve,
			"remark":         "建议多喝水",
			"is_isolated":    true,
		}).Error)

	svc := newParentHealthService(f.db)
	records, err := svc.ListHealthRecords(context.Background(), parentHealthContext(), hcTestStudent)
	require.NoError(t, err)
	require.Len(t, records, 1)

	record := records[0]
	assert.Equal(t, "需观察", record.OverallStatusLabel)
	assert.Equal(t, "建议多喝水", record.Remark)
	assert.True(t, record.IsIsolated)
	require.Len(t, record.AbnormalItems, 3)
	assert.Equal(t, "喉咙", record.AbnormalItems[0].PartLabel)
	assert.Equal(t, "红肿", record.AbnormalItems[0].StatusLabel)
	assert.Equal(t, "手部", record.AbnormalItems[1].PartLabel)
	assert.Equal(t, "疱疹", record.AbnormalItems[1].StatusLabel)
	assert.Equal(t, "口腔", record.AbnormalItems[2].PartLabel)
	assert.Equal(t, "其他异常", record.AbnormalItems[2].StatusLabel)
}

func TestParentHealthRecords_RejectsUnboundStudent(t *testing.T) {
	f := newHealthFixture(t)
	svc := newParentHealthService(f.db)

	_, err := svc.ListHealthRecords(context.Background(), parentHealthContext(), 123456)
	require.ErrorIs(t, err, ErrParentForbidden)
}

func TestHealthCheckSubmit_RequiredTemperatureRejected(t *testing.T) {
	f := newHealthFixture(t)
	svc := f.withPolicy(HealthPolicy{TemperatureRequired: true, FeverThreshold: DefaultFeverThreshold})

	_, err := svc.SubmitCheck(context.Background(), hcTestKG, hcTestDoctor, hcTestCheck, SubmitHealthCheckReq{
		OverallStatus: int8(domain.HealthOverallHealthy),
	})
	require.ErrorIs(t, err, ErrHealthCheckTemperatureRequired)

	var check domain.MorningHealthCheck
	require.NoError(t, f.db.Where("id = ?", hcTestCheck).First(&check).Error)
	assert.Equal(t, domain.HealthCheckPending, check.CheckStatus)
}

func TestHealthCheckSubmit_RequiredTemperatureAccepted(t *testing.T) {
	f := newHealthFixture(t)
	svc := f.withPolicy(HealthPolicy{TemperatureRequired: true, FeverThreshold: DefaultFeverThreshold})
	temp := 36.6

	_, err := svc.SubmitCheck(context.Background(), hcTestKG, hcTestDoctor, hcTestCheck, SubmitHealthCheckReq{
		Temperature:   &temp,
		OverallStatus: int8(domain.HealthOverallHealthy),
	})
	require.NoError(t, err)

	var check domain.MorningHealthCheck
	require.NoError(t, f.db.Where("id = ?", hcTestCheck).First(&check).Error)
	require.NotNil(t, check.Temperature)
	assert.InDelta(t, 36.6, *check.Temperature, 1e-9)
}

func TestHealthCheckSubmit_RequiredSkipExempt(t *testing.T) {
	f := newHealthFixture(t)
	svc := f.withPolicy(HealthPolicy{TemperatureRequired: true, FeverThreshold: DefaultFeverThreshold})
	skip := true

	_, err := svc.SubmitCheck(context.Background(), hcTestKG, hcTestDoctor, hcTestCheck, SubmitHealthCheckReq{Skip: &skip})
	require.NoError(t, err)

	var check domain.MorningHealthCheck
	require.NoError(t, f.db.Where("id = ?", hcTestCheck).First(&check).Error)
	assert.Equal(t, domain.HealthCheckSkipped, check.CheckStatus)
}

func TestHealthCheckSubmit_OptionalAllowsNullTemperature(t *testing.T) {
	f := newHealthFixture(t)

	_, err := f.svc.SubmitCheck(context.Background(), hcTestKG, hcTestDoctor, hcTestCheck, SubmitHealthCheckReq{
		OverallStatus: int8(domain.HealthOverallHealthy),
	})
	require.NoError(t, err)

	var check domain.MorningHealthCheck
	require.NoError(t, f.db.Where("id = ?", hcTestCheck).First(&check).Error)
	assert.Nil(t, check.Temperature, "按需测温不得把空体温写成默认正常值")
}

func TestGetQueueWithPolicy_IncludesPolicy(t *testing.T) {
	f := newHealthFixture(t)

	res, err := f.svc.GetQueueWithPolicy(context.Background(), hcTestKG, nil, time.Now().Format("2006-01-02"))
	require.NoError(t, err)
	require.Len(t, res.List, 1)
	assert.False(t, res.Policy.TemperatureRequired)
	assert.InDelta(t, DefaultFeverThreshold, res.Policy.FeverThreshold, 1e-9)
}
