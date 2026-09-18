package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"kindergarten-server/internal/domain"
	"kindergarten-server/pkg/pagination"
)

// --- mocks ---

type mockClassRepo struct {
	mock.Mock
}

func (m *mockClassRepo) List(ctx context.Context, kgID uint64, grade, keyword string, offset, limit int) ([]domain.Class, int64, error) {
	args := m.Called(ctx, kgID, grade, keyword, offset, limit)
	return args.Get(0).([]domain.Class), args.Get(1).(int64), args.Error(2)
}

func (m *mockClassRepo) FindByID(ctx context.Context, kgID, classID uint64) (*domain.Class, error) {
	args := m.Called(ctx, kgID, classID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Class), args.Error(1)
}

func (m *mockClassRepo) CountStudentsInClass(ctx context.Context, kgID, classID uint64) (int64, error) {
	args := m.Called(ctx, kgID, classID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockClassRepo) CountStudentsByClassIDs(ctx context.Context, kgID uint64, classIDs []uint64) (map[uint64]int64, error) {
	args := m.Called(ctx, kgID, classIDs)
	if args.Get(0) == nil {
		return map[uint64]int64{}, args.Error(1)
	}
	return args.Get(0).(map[uint64]int64), args.Error(1)
}

func (m *mockClassRepo) ListTeachersByClassIDs(ctx context.Context, classIDs []uint64) ([]domain.ClassTeacher, error) {
	args := m.Called(ctx, classIDs)
	return args.Get(0).([]domain.ClassTeacher), args.Error(1)
}

func (m *mockClassRepo) CreateClassWithTeachers(ctx context.Context, class *domain.Class, teachers []domain.ClassTeacher) error {
	args := m.Called(ctx, class, teachers)
	if args.Error(0) == nil && class.ID == 0 {
		class.ID = 100 // simulate auto-increment
	}
	return args.Error(0)
}

func (m *mockClassRepo) Save(ctx context.Context, c *domain.Class) error {
	args := m.Called(ctx, c)
	return args.Error(0)
}

func (m *mockClassRepo) Delete(ctx context.Context, kgID, classID uint64) error {
	args := m.Called(ctx, kgID, classID)
	return args.Error(0)
}

func (m *mockClassRepo) ReplaceClassTeachers(ctx context.Context, classID uint64, teachers []domain.ClassTeacher) error {
	args := m.Called(ctx, classID, teachers)
	return args.Error(0)
}

func (m *mockClassRepo) ListStudentsPreview(ctx context.Context, kgID, classID uint64, limit int) ([]domain.Student, error) {
	args := m.Called(ctx, kgID, classID, limit)
	return args.Get(0).([]domain.Student), args.Error(1)
}

func (m *mockClassRepo) CountStudentsInClassWithIDs(ctx context.Context, kgID, classID uint64, studentIDs []uint64) (int64, error) {
	args := m.Called(ctx, kgID, classID, studentIDs)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockClassRepo) UpdateStudentsClass(ctx context.Context, kgID, fromClassID, toClassID uint64, studentIDs []uint64) (int64, error) {
	args := m.Called(ctx, kgID, fromClassID, toClassID, studentIDs)
	return args.Get(0).(int64), args.Error(1)
}

type mockTeacherLookup struct {
	mock.Mock
}

// FindTeacherInKindergarten 返回 nil 表示该用户在本园不是老师。
// 期望值里带上 kindergartenID，可以直接断言调用方传对了园所。
func (m *mockTeacherLookup) FindTeacherInKindergarten(ctx context.Context, userID, kindergartenID uint64) (*domain.UserRole, error) {
	args := m.Called(ctx, userID, kindergartenID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.UserRole), args.Error(1)
}

type mockOpLogRepo struct {
	mock.Mock
}

func (m *mockOpLogRepo) Create(ctx context.Context, log *domain.OperationLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

// --- helpers ---

func newTestClassService() (*ClassService, *mockClassRepo, *mockTeacherLookup, *mockOpLogRepo) {
	cr := new(mockClassRepo)
	ur := new(mockTeacherLookup)
	ol := new(mockOpLogRepo)
	svc := NewClassService(cr, ur, ol)
	return svc, cr, ur, ol
}

func validCreateReq() CreateClassReq {
	return CreateClassReq{
		Name:     "星星一班",
		Grade:    "小班",
		Capacity: 25,
	}
}

// --- tests ---

func TestCreateClass_Success(t *testing.T) {
	svc, cr, ur, ol := newTestClassService()
	ctx := context.Background()
	req := validCreateReq()

	cr.On("CreateClassWithTeachers", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	ol.On("Create", mock.Anything, mock.Anything).Return(nil)

	resp, err := svc.CreateClass(ctx, 1, 1, req, "127.0.0.1", "test")

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, uint64(100), resp.ID)
	assert.NotEmpty(t, resp.CreatedAt)
	cr.AssertExpectations(t)
	ol.AssertExpectations(t)
	ur.AssertNotCalled(t, "FindTeacherInKindergarten")
}

func TestCreateClass_NoKindergarten(t *testing.T) {
	svc, cr, _, _ := newTestClassService()
	ctx := context.Background()
	req := validCreateReq()

	resp, err := svc.CreateClass(ctx, 0, 1, req, "127.0.0.1", "test")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "NO_KINDERGARTEN")
	assert.Nil(t, resp)
	cr.AssertNotCalled(t, "CreateClassWithTeachers")
}

func TestCreateClass_InvalidGrade(t *testing.T) {
	svc, cr, _, _ := newTestClassService()
	ctx := context.Background()
	req := validCreateReq()
	req.Grade = "托班" // 不支持的年级

	resp, err := svc.CreateClass(ctx, 1, 1, req, "127.0.0.1", "test")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "INVALID_GRADE")
	assert.Nil(t, resp)
	cr.AssertNotCalled(t, "CreateClassWithTeachers")
}

func TestCreateClass_WithTeachers(t *testing.T) {
	svc, cr, ur, ol := newTestClassService()
	ctx := context.Background()
	req := validCreateReq()
	req.TeacherIDs = []classTeacherAssign{
		{UserID: 5, Title: "head"},
		{UserID: 6, Title: "assistant"},
	}

	// 期望值里带上 kindergartenID=1，顺带断言调用方确实把园所传了进来。
	ur.On("FindTeacherInKindergarten", mock.Anything, uint64(5), uint64(1)).
		Return(&domain.UserRole{UserID: 5, Role: "teacher"}, nil)
	ur.On("FindTeacherInKindergarten", mock.Anything, uint64(6), uint64(1)).
		Return(&domain.UserRole{UserID: 6, Role: "teacher"}, nil)
	cr.On("CreateClassWithTeachers", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	ol.On("Create", mock.Anything, mock.Anything).Return(nil)

	resp, err := svc.CreateClass(ctx, 1, 1, req, "127.0.0.1", "test")

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	ur.AssertNumberOfCalls(t, "FindTeacherInKindergarten", 2)
}

func TestCreateClass_DuplicateTeacher(t *testing.T) {
	svc, _, ur, _ := newTestClassService()
	ctx := context.Background()
	req := validCreateReq()
	req.TeacherIDs = []classTeacherAssign{
		{UserID: 5, Title: "head"},
		{UserID: 5, Title: "assistant"},
	}

	ur.On("FindTeacherInKindergarten", mock.Anything, uint64(5), uint64(1)).
		Return(&domain.UserRole{UserID: 5, Role: "teacher"}, nil)

	resp, err := svc.CreateClass(ctx, 1, 1, req, "127.0.0.1", "test")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "DUPLICATE_TEACHER")
	assert.Nil(t, resp)
}

// 被指派的用户可能真实存在、甚至确实是老师，但不属于本园。这正是要拦的跨租户
// 指派：class_teachers 没有 kindergarten_id 列，数据库层没有兜底，
// buildClassTeachers 的归属校验是唯一防线。
func TestCreateClass_TeacherNotInKindergarten(t *testing.T) {
	svc, cr, ur, _ := newTestClassService()
	ctx := context.Background()
	req := validCreateReq()
	req.TeacherIDs = []classTeacherAssign{
		{UserID: 99, Title: "head"},
	}

	// 用户不存在、或只属于别的园所 —— 对调用方都表现为 nil。
	ur.On("FindTeacherInKindergarten", mock.Anything, uint64(99), uint64(1)).Return(nil, nil)

	_, err := svc.CreateClass(ctx, 1, 1, req, "127.0.0.1", "test")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TEACHER_NOT_IN_KINDERGARTEN")
	// 关键：校验失败时不能已经落库。
	cr.AssertNotCalled(t, "CreateClassWithTeachers")
}

func TestDeleteClass_Success(t *testing.T) {
	svc, cr, _, ol := newTestClassService()
	ctx := context.Background()

	cr.On("FindByID", mock.Anything, uint64(1), uint64(1)).
		Return(&domain.Class{BaseModel: domain.BaseModel{ID: 1}, KindergartenID: 1, Name: "x", Grade: "小班"}, nil)
	cr.On("CountStudentsInClass", mock.Anything, uint64(1), uint64(1)).Return(int64(0), nil)
	cr.On("ReplaceClassTeachers", mock.Anything, uint64(1), []domain.ClassTeacher(nil)).Return(nil)
	cr.On("Delete", mock.Anything, uint64(1), uint64(1)).Return(nil)
	ol.On("Create", mock.Anything, mock.Anything).Return(nil)

	err := svc.DeleteClass(ctx, 1, 1, 1, "127.0.0.1", "test")

	assert.NoError(t, err)
}

func TestDeleteClass_NotEmpty(t *testing.T) {
	svc, cr, _, _ := newTestClassService()
	ctx := context.Background()

	cr.On("FindByID", mock.Anything, uint64(1), uint64(1)).
		Return(&domain.Class{BaseModel: domain.BaseModel{ID: 1}, KindergartenID: 1}, nil)
	cr.On("CountStudentsInClass", mock.Anything, uint64(1), uint64(1)).Return(int64(5), nil)

	err := svc.DeleteClass(ctx, 1, 1, 1, "127.0.0.1", "test")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CLASS_NOT_EMPTY")
	cr.AssertNotCalled(t, "Delete")
}

func TestListClasses_Success(t *testing.T) {
	svc, cr, _, _ := newTestClassService()
	ctx := context.Background()

	classes := []domain.Class{{BaseModel: domain.BaseModel{ID: 1}, KindergartenID: 1, Name: "小一班", Grade: "小班", Status: 1}}
	cr.On("List", mock.Anything, uint64(1), "", "", 0, 20).Return(classes, int64(1), nil)
	cr.On("CountStudentsByClassIDs", mock.Anything, uint64(1), mock.Anything).Return(map[uint64]int64{1: 10}, nil)
	cr.On("ListTeachersByClassIDs", mock.Anything, mock.Anything).Return([]domain.ClassTeacher{}, nil)

	list, total, page, pageSize, err := svc.ListClasses(ctx, 1, pagination.Query{Page: 1, PageSize: 20}, "")

	assert.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, 1, page)
	assert.Equal(t, 20, pageSize)
	assert.Equal(t, "小一班", list[0].Name)
}

