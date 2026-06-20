package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/pkg/pagination"
)

// --- mocks ---

type mockGrowthRepo struct {
	mock.Mock
}

func (m *mockGrowthRepo) ListLogs(ctx context.Context, kgID uint64, classID, teacherID *uint64, dateStr string, offset, limit int) ([]repository.GrowthLogListRow, int64, error) {
	args := m.Called(ctx, kgID, classID, teacherID, dateStr, offset, limit)
	return args.Get(0).([]repository.GrowthLogListRow), args.Get(1).(int64), args.Error(2)
}

func (m *mockGrowthRepo) FindLogByID(ctx context.Context, kgID, logID uint64) (*domain.GrowthDailyLog, error) {
	args := m.Called(ctx, kgID, logID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.GrowthDailyLog), args.Error(1)
}

func (m *mockGrowthRepo) FindLatestAudit(ctx context.Context, kgID, logID uint64) (*domain.ContentAudit, error) {
	args := m.Called(ctx, kgID, logID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ContentAudit), args.Error(1)
}

func (m *mockGrowthRepo) ListMediaByLogID(ctx context.Context, logID uint64) ([]domain.GrowthMedia, error) {
	args := m.Called(ctx, logID)
	return args.Get(0).([]domain.GrowthMedia), args.Error(1)
}

func (m *mockGrowthRepo) SoftDeleteLog(ctx context.Context, kgID, logID uint64) (int64, error) {
	args := m.Called(ctx, kgID, logID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockGrowthRepo) SaveContentAudit(ctx context.Context, ca *domain.ContentAudit) error {
	args := m.Called(ctx, ca)
	return args.Error(0)
}

func (m *mockGrowthRepo) CountLogsInDateRange(ctx context.Context, kgID uint64, start, end time.Time) (int64, error) {
	args := m.Called(ctx, kgID, start, end)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockGrowthRepo) CountMediaInLogDateRange(ctx context.Context, kgID uint64, start, end time.Time) (int64, error) {
	args := m.Called(ctx, kgID, start, end)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockGrowthRepo) CountLikesInLogDateRange(ctx context.Context, kgID uint64, start, end time.Time) (int64, error) {
	args := m.Called(ctx, kgID, start, end)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockGrowthRepo) CountCommentsInLogDateRange(ctx context.Context, kgID uint64, start, end time.Time) (int64, error) {
	args := m.Called(ctx, kgID, start, end)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockGrowthRepo) StatsByClass(ctx context.Context, kgID uint64, start, end time.Time) ([]repository.GrowthClassStatRow, error) {
	args := m.Called(ctx, kgID, start, end)
	return args.Get(0).([]repository.GrowthClassStatRow), args.Error(1)
}

func (m *mockGrowthRepo) StatsDailyTrend(ctx context.Context, kgID uint64, start, end time.Time) ([]repository.GrowthDailyTrendRow, error) {
	args := m.Called(ctx, kgID, start, end)
	return args.Get(0).([]repository.GrowthDailyTrendRow), args.Error(1)
}

func (m *mockGrowthRepo) CountLikesByLogID(ctx context.Context, logID uint64) (int64, error) {
	args := m.Called(ctx, logID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockGrowthRepo) CountCommentsByLogID(ctx context.Context, logID uint64) (int64, error) {
	args := m.Called(ctx, logID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockGrowthRepo) FindUserDisplay(ctx context.Context, userID uint64) (nickname, avatar string, ok bool) {
	args := m.Called(ctx, userID)
	return args.String(0), args.String(1), args.Bool(2)
}

func (m *mockGrowthRepo) ListLogsForExportDateRange(ctx context.Context, kgID uint64, classID *uint64, start, end time.Time) ([]repository.GrowthLogExportRow, error) {
	args := m.Called(ctx, kgID, classID, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repository.GrowthLogExportRow), args.Error(1)
}

// --- helpers ---

func newTestGrowthService() (*GrowthService, *mockGrowthRepo, *mockOpLogRepo) {
	gr := new(mockGrowthRepo)
	ol := new(mockOpLogRepo)
	svc := NewGrowthService(gr, ol)
	return svc, gr, ol
}

func mockGrowthLog(id uint64) *domain.GrowthDailyLog {
	now := time.Now()
	return &domain.GrowthDailyLog{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: id, CreatedAt: now, UpdatedAt: now}},
		KindergartenID:  1,
		ClassID:         1,
		CreatorID:       5,
		Date:            now,
		BreakfastRating: 3,
		LunchRating:     2,
		DinnerRating:    3,
		NapDuration:     120,
		SleepQuality:    3,
		MorningMood:     3,
		AfternoonMood:   2,
		PerformanceTags: "乐于助人,积极发言",
		Content:         "今天小朋友们表现很好",
		SyncToFeed:      true,
		TargetType:      1,
	}
}

// ============================================================
// 纯函数测试
// ============================================================

func TestAuditStatusString(t *testing.T) {
	approved := int8(domain.AuditStatusApproved)
	rejected := int8(domain.AuditStatusRejected)
	pending := int8(0)

	assert.Equal(t, "passed", auditStatusString(&approved))
	assert.Equal(t, "rejected", auditStatusString(&rejected))
	assert.Equal(t, "pending", auditStatusString(&pending))
	assert.Equal(t, "pending", auditStatusString(nil))
}

func TestGrowthContentPreview(t *testing.T) {
	assert.Equal(t, "", growthContentPreview("", 50))
	assert.Equal(t, "", growthContentPreview("   ", 50)) // whitespace trimmed to empty
	assert.Equal(t, "Hello", growthContentPreview("Hello", 50))
	assert.Equal(t, "Hello...", growthContentPreview("Hello World This Is A Very Long Text", 5))
	assert.Equal(t, "你好世界", growthContentPreview("你好世界", 10))
	assert.Equal(t, "你好...", growthContentPreview("你好世界测试文本", 2))
}

// ============================================================
// 评分与枚举常量测试
// ============================================================

func TestMealRatingConstants(t *testing.T) {
	assert.Equal(t, domain.MealRating(3), domain.MealRatingGood)
	assert.Equal(t, domain.MealRating(2), domain.MealRatingNormal)
	assert.Equal(t, domain.MealRating(1), domain.MealRatingPoor)
}

func TestSleepQualityConstants(t *testing.T) {
	assert.Equal(t, domain.SleepQuality(3), domain.SleepQualityGood)
	assert.Equal(t, domain.SleepQuality(2), domain.SleepQualityNormal)
	assert.Equal(t, domain.SleepQuality(1), domain.SleepQualityPoor)
}

func TestMoodRatingConstants(t *testing.T) {
	assert.Equal(t, domain.MoodRating(3), domain.MoodHappy)
	assert.Equal(t, domain.MoodRating(2), domain.MoodNormal)
	assert.Equal(t, domain.MoodRating(1), domain.MoodSad)
}

func TestAuditStatusConstants(t *testing.T) {
	assert.Equal(t, domain.AuditStatus(1), domain.AuditStatusApproved)
	assert.Equal(t, domain.AuditStatus(2), domain.AuditStatusRejected)
}

// ============================================================
// 业务逻辑测试 — 删除日志
// ============================================================

func TestDeleteGrowthLog_Success(t *testing.T) {
	svc, gr, ol := newTestGrowthService()
	ctx := context.Background()
	log := mockGrowthLog(100)

	gr.On("FindLogByID", mock.Anything, uint64(1), uint64(100)).Return(log, nil)
	gr.On("SoftDeleteLog", mock.Anything, uint64(1), uint64(100)).Return(int64(1), nil)
	ol.On("Create", mock.Anything, mock.Anything).Return(nil)

	err := svc.DeleteGrowthLog(ctx, 1, 1, 100, "127.0.0.1", "test")

	assert.NoError(t, err)
	gr.AssertExpectations(t)
}

func TestDeleteGrowthLog_NotFound(t *testing.T) {
	svc, gr, _ := newTestGrowthService()
	ctx := context.Background()

	gr.On("FindLogByID", mock.Anything, uint64(1), uint64(999)).Return(nil, nil)

	err := svc.DeleteGrowthLog(ctx, 1, 1, 999, "127.0.0.1", "test")

	assert.Error(t, err)
	assert.True(t, IsGrowthNotFound(err))
	gr.AssertNotCalled(t, "SoftDeleteLog")
}

func TestDeleteGrowthLog_NoKindergarten(t *testing.T) {
	svc, gr, _ := newTestGrowthService()
	ctx := context.Background()

	err := svc.DeleteGrowthLog(ctx, 0, 1, 100, "127.0.0.1", "test")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "NO_KINDERGARTEN")
	gr.AssertNotCalled(t, "FindLogByID")
}

func TestDeleteGrowthLog_SoftDeleteFails(t *testing.T) {
	svc, gr, _ := newTestGrowthService()
	ctx := context.Background()
	log := mockGrowthLog(100)

	gr.On("FindLogByID", mock.Anything, uint64(1), uint64(100)).Return(log, nil)
	gr.On("SoftDeleteLog", mock.Anything, uint64(1), uint64(100)).Return(int64(0), nil)

	err := svc.DeleteGrowthLog(ctx, 1, 1, 100, "127.0.0.1", "test")

	assert.Error(t, err)
	assert.True(t, IsGrowthNotFound(err))
}

// ============================================================
// 审核日志测试
// ============================================================

func TestAuditGrowthLog_Pass(t *testing.T) {
	svc, gr, ol := newTestGrowthService()
	ctx := context.Background()
	log := mockGrowthLog(100)

	gr.On("FindLogByID", mock.Anything, uint64(1), uint64(100)).Return(log, nil)
	gr.On("SaveContentAudit", mock.Anything, mock.Anything).Return(nil)
	ol.On("Create", mock.Anything, mock.Anything).Return(nil)

	resp, err := svc.AuditGrowthLog(ctx, 1, 1, 100, AuditGrowthLogReq{Action: "pass", Remark: "合规"}, "127.0.0.1", "test")

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "passed", resp["audit_status"])
	gr.AssertExpectations(t)
}

func TestAuditGrowthLog_Reject(t *testing.T) {
	svc, gr, ol := newTestGrowthService()
	ctx := context.Background()
	log := mockGrowthLog(100)

	gr.On("FindLogByID", mock.Anything, uint64(1), uint64(100)).Return(log, nil)
	gr.On("SaveContentAudit", mock.Anything, mock.Anything).Return(nil)
	ol.On("Create", mock.Anything, mock.Anything).Return(nil)

	resp, err := svc.AuditGrowthLog(ctx, 1, 1, 100, AuditGrowthLogReq{Action: "reject", Remark: "内容不当"}, "127.0.0.1", "test")

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "rejected", resp["audit_status"])
}

func TestAuditGrowthLog_InvalidAction(t *testing.T) {
	svc, gr, _ := newTestGrowthService()
	ctx := context.Background()
	log := mockGrowthLog(100)

	gr.On("FindLogByID", mock.Anything, uint64(1), uint64(100)).Return(log, nil)

	_, err := svc.AuditGrowthLog(ctx, 1, 1, 100, AuditGrowthLogReq{Action: "invalid"}, "127.0.0.1", "test")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ACTION")
	gr.AssertNotCalled(t, "SaveContentAudit")
}

func TestAuditGrowthLog_LogNotFound(t *testing.T) {
	svc, gr, _ := newTestGrowthService()
	ctx := context.Background()

	gr.On("FindLogByID", mock.Anything, uint64(1), uint64(999)).Return(nil, nil)

	_, err := svc.AuditGrowthLog(ctx, 1, 1, 999, AuditGrowthLogReq{Action: "pass"}, "127.0.0.1", "test")

	assert.Error(t, err)
	assert.True(t, IsGrowthNotFound(err))
}

// ============================================================
// 边界测试
// ============================================================

func TestGrowthService_NoKindergarten(t *testing.T) {
	svc, gr, _ := newTestGrowthService()
	ctx := context.Background()

	// ListGrowthLogs
	_, _, _, _, err := svc.ListGrowthLogs(ctx, 0, pagination.Query{}, nil, nil, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "NO_KINDERGARTEN")
	gr.AssertNotCalled(t, "ListLogs")

	// GetGrowthLogDetail
	_, err = svc.GetGrowthLogDetail(ctx, 0, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "NO_KINDERGARTEN")

	// GetGrowthStats
	_, err = svc.GetGrowthStats(ctx, 0, "", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "NO_KINDERGARTEN")
}

func TestIsGrowthNotFound(t *testing.T) {
	assert.True(t, IsGrowthNotFound(errGrowthNotFound))
	assert.False(t, IsGrowthNotFound(context.Canceled))
	assert.False(t, IsGrowthNotFound(nil))
}
