package service

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// OS-ALB-9：学生状态时间 status_changed_at。
//
// 家长端相册的 90 天留存窗口（TC-NEW-ALB-0020/0021）靠这列判定，
// 所以这里验证写入端：真实变化才写 now、回到 0/1 清空、同状态不刷新。
//
// 打真实 MySQL（本地 configs/app.ini），固定 9963xx 夹具，t.Cleanup 硬删 0 残留。

const (
	testSSTKG      uint64 = 996301
	testSSTClass   uint64 = 996310
	testSSTUser    uint64 = 996320
	testSSTStudent uint64 = 996330
)

func sstCleanup(t *testing.T, db *gorm.DB) {
	t.Helper()
	del := func(sql string, args ...any) {
		if err := db.Exec(sql, args...).Error; err != nil {
			t.Logf("cleanup: %v", err)
		}
	}
	del("DELETE FROM operation_logs WHERE user_id BETWEEN ? AND ?", testSSTUser, uint64(996329))
	del("DELETE FROM students WHERE id BETWEEN ? AND ?", testSSTStudent, uint64(996339))
	del("DELETE FROM classes WHERE id BETWEEN ? AND ?", testSSTClass, uint64(996319))
	del("DELETE FROM users WHERE id BETWEEN ? AND ?", testSSTUser, uint64(996329))
	del("DELETE FROM kindergartens WHERE id = ?", testSSTKG)
}

type sstFixture struct {
	db  *gorm.DB
	svc *StudentService
}

var sstMigrateOnce sync.Once

func newSSTFixture(t *testing.T) *sstFixture {
	t.Helper()
	db := openCLPTestDB(t)
	sstMigrateOnce.Do(func() {
		require.NoError(t, db.AutoMigrate(
			&domain.Student{}, &domain.OperationLog{}, &domain.Kindergarten{}, &domain.Class{}, &domain.User{},
		))
	})
	sstCleanup(t, db)
	t.Cleanup(func() { sstCleanup(t, db) })

	require.NoError(t, db.Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: testSSTKG}, Name: "测试园所-状态时间",
	}).Error)
	require.NoError(t, db.Create(&domain.Class{
		BaseModel: domain.BaseModel{ID: testSSTClass}, KindergartenID: testSSTKG,
		Name: "状态一班", Grade: "大班", Status: 1,
	}).Error)
	require.NoError(t, db.Create(&domain.User{
		BaseModel: domain.BaseModel{ID: testSSTUser}, Nickname: "状态老师",
		Phone: "13900009931", PasswordHash: "x",
	}).Error)

	svc := NewStudentService(db,
		repository.NewStudentRepo(db), repository.NewClassRepo(db), repository.NewUserRepo(db),
		repository.NewUserRoleRepo(db), repository.NewKindergartenRepo(db), repository.NewOperationLogRepo(db))
	return &sstFixture{db: db, svc: svc}
}

// createStudent 造一个学生；changedAt 非空用于模拟「已经毕业/退学过」的历史行。
func (f *sstFixture) createStudent(t *testing.T, id uint64, status int8, changedAt *time.Time) {
	t.Helper()
	require.NoError(t, f.db.Create(&domain.Student{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: id}},
		KindergartenID:  testSSTKG,
		ClassID:         testSSTClass,
		Name:            fmt.Sprintf("状态幼儿-%d", id),
		Status:          status,
		StatusChangedAt: changedAt,
		Birthday:        time.Now().AddDate(-5, 0, 0),
		EnrollDate:      time.Now().AddDate(-1, 0, 0),
	}).Error)
}

func (f *sstFixture) reload(t *testing.T, id uint64) domain.Student {
	t.Helper()
	var st domain.Student
	require.NoError(t, f.db.First(&st, id).Error)
	return st
}

func (f *sstFixture) updateStatus(t *testing.T, id uint64, status int8) {
	t.Helper()
	_, err := f.svc.UpdateStudentStatus(context.Background(), testSSTKG, testSSTUser, id,
		UpdateStudentStatusReq{Status: status, Reason: "测试"}, "127.0.0.1", "os-alb-9")
	require.NoError(t, err)
}

// 1→2 写时间、2→1 清空、1→1 不写；2→2 重复提交不刷新窗口起点。
func TestStudentStatusChangedAtOnStatusUpdate(t *testing.T) {
	f := newSSTFixture(t)

	// 1 → 2 毕业：写 now。
	f.createStudent(t, testSSTStudent, 1, nil)
	f.updateStatus(t, testSSTStudent, 2)
	got := f.reload(t, testSSTStudent)
	assert.Equal(t, int8(2), got.Status)
	require.NotNil(t, got.StatusChangedAt, "毕业必须写 status_changed_at")
	assert.WithinDuration(t, time.Now(), *got.StatusChangedAt, 5*time.Second)

	// 2 → 2：重复提交同一状态不刷新时间（否则会无限延长 90 天窗口）。
	first := *got.StatusChangedAt
	f.updateStatus(t, testSSTStudent, 2)
	got = f.reload(t, testSSTStudent)
	require.NotNil(t, got.StatusChangedAt)
	assert.True(t, first.Equal(*got.StatusChangedAt), "状态未变化时时间戳必须原样保留")

	// 2 → 1 恢复在读：清空。
	f.updateStatus(t, testSSTStudent, 1)
	got = f.reload(t, testSSTStudent)
	assert.Equal(t, int8(1), got.Status)
	assert.Nil(t, got.StatusChangedAt, "回到在读必须清空 status_changed_at")

	// 1 → 1：原本为空，保持为空。
	f.updateStatus(t, testSSTStudent, 1)
	got = f.reload(t, testSSTStudent)
	assert.Nil(t, got.StatusChangedAt)
}

// 1→3 退学也写时间；历史 2/3 行为空时，重复提交同一状态不回填。
func TestStudentStatusChangedAtOnLeaveAndHistoricalRows(t *testing.T) {
	f := newSSTFixture(t)

	f.createStudent(t, 996331, 1, nil)
	f.updateStatus(t, 996331, 3)
	got := f.reload(t, 996331)
	assert.Equal(t, int8(3), got.Status)
	require.NotNil(t, got.StatusChangedAt, "退学必须写 status_changed_at")
	assert.WithinDuration(t, time.Now(), *got.StatusChangedAt, 5*time.Second)

	// 迁移前的毕业行没有时间戳：读取端按已过期处理，写入端也不能猜一个出来。
	f.createStudent(t, 996332, 2, nil)
	f.updateStatus(t, 996332, 2)
	got = f.reload(t, 996332)
	assert.Equal(t, int8(2), got.Status)
	assert.Nil(t, got.StatusChangedAt, "历史行不因重复提交被回填时间")
}

// 编辑路径（UpdateStudent）不改变状态，也不得把已有 status_changed_at 刷新成编辑时间。
func TestStudentStatusChangedAtPreservedOnEdit(t *testing.T) {
	f := newSSTFixture(t)

	graduatedAt := time.Now().Add(-30 * 24 * time.Hour).Truncate(time.Second)
	f.createStudent(t, 996334, 2, &graduatedAt)

	_, err := f.svc.UpdateStudent(context.Background(), testSSTKG, testSSTUser, 996334,
		UpdateStudentReq{Name: "改名幼儿", HealthNote: "备注"}, "127.0.0.1", "os-alb-9")
	require.NoError(t, err)

	got := f.reload(t, 996334)
	assert.Equal(t, "改名幼儿", got.Name)
	assert.Equal(t, int8(2), got.Status)
	require.NotNil(t, got.StatusChangedAt)
	assert.True(t, graduatedAt.Equal(*got.StatusChangedAt), "编辑不能把留存窗口起点改成编辑时间")

	// 在读学生编辑后仍为空：编辑不是状态迁移。
	f.createStudent(t, 996335, 1, nil)
	_, err = f.svc.UpdateStudent(context.Background(), testSSTKG, testSSTUser, 996335,
		UpdateStudentReq{Name: "在读幼儿"}, "127.0.0.1", "os-alb-9")
	require.NoError(t, err)
	got = f.reload(t, 996335)
	assert.Nil(t, got.StatusChangedAt)
}
