package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// 管理端滞留名单 / 超时未接计数 / 一键通知留痕（spec TC-NEW-PKU-091 / 101 / 110）。
// 真 DB 跑，复用 teacher_pickup_create_test.go 的 fixture。

func newAdminPickupService(f *pickupFixture) *PickupService {
	return NewPickupService(
		f.svc.db,
		repository.NewPickupRepo(f.svc.db),
		repository.NewStudentRepo(f.svc.db),
		repository.NewClassRepo(f.svc.db),
		repository.NewOperationLogRepo(f.svc.db),
		repository.NewMessageRepo(f.svc.db),
	)
}

// makeParent 给测试学生建一名家长（User + StudentParent），供通知投递。
func (f *pickupFixture) makeParent(t *testing.T) uint64 {
	t.Helper()
	parentUserID := f.studentID + 700000
	// User 可能被其它用例复用，先查
	var cnt int64
	f.svc.db.Model(&domain.User{}).Where("id = ?", parentUserID).Count(&cnt)
	if cnt == 0 {
		require.NoError(t, f.svc.db.Create(&domain.User{
			BaseModel: domain.BaseModel{ID: parentUserID},
			Phone:     "13700000001",
			Nickname:  "测试家长",
		}).Error)
	}
	require.NoError(t, f.svc.db.Create(&domain.StudentParent{
		StudentID: f.studentID,
		UserID:    parentUserID,
		Relation:  domain.RelationMother,
	}).Error)
	return parentUserID
}

// TC-NEW-PKU-091 管理员看到全园滞留名单
func TestAdminOverdue_ListByKindergarten(t *testing.T) {
	// deadline 00:00 → 恒已过；学生仍在园 → 滞留
	f := newPickupFixture(t, "00:00", testPickupClass)
	svc := newAdminPickupService(f)

	_, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
		StudentID: f.studentID, Type: 1,
	}, domain.PickupMethodManual)
	require.NoError(t, err)

	resp, err := svc.ListOverdueStudents(ctx(), testPickupKG, 0, "")
	require.NoError(t, err)

	assert.Equal(t, 1, resp["overdue_total"], "全园应见 1 名滞留")

	byClass, ok := resp["by_class"].([]map[string]interface{})
	require.True(t, ok, "by_class 应为分组列表")
	require.Len(t, byClass, 1, "应归入一个班级分组")

	g := byClass[0]
	assert.Equal(t, testPickupClass, g["class_id"])
	assert.Equal(t, 1, g["overdue_count"])
	assert.Equal(t, "00:00", g["pickup_deadline"], "pickup_deadline 应为班级配置值")

	students, ok := g["students"].([]map[string]interface{})
	require.True(t, ok)
	require.Len(t, students, 1)
	assert.Equal(t, f.studentID, students[0]["student_id"])
}

// 按班级过滤
func TestAdminOverdue_ListByClass(t *testing.T) {
	f := newPickupFixture(t, "00:00", testPickupClass)
	svc := newAdminPickupService(f)

	_, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
		StudentID: f.studentID, Type: 1,
	}, domain.PickupMethodManual)
	require.NoError(t, err)

	// 限本班 → 命中
	resp, err := svc.ListOverdueStudents(ctx(), testPickupKG, testPickupClass, "")
	require.NoError(t, err)
	assert.Equal(t, 1, resp["overdue_total"])

	// 限别的班 → 空
	resp, err = svc.ListOverdueStudents(ctx(), testPickupKG, testPickupClass2, "")
	require.NoError(t, err)
	assert.Equal(t, 0, resp["overdue_total"], "别的班不得越权看到")
}

// TC-NEW-PKU-101 通知留痕可查（重复触发不覆盖历史）
func TestAdminOverdue_NotifyLeavesTrace(t *testing.T) {
	f := newPickupFixture(t, "00:00", testPickupClass)
	svc := newAdminPickupService(f)
	parentUserID := f.makeParent(t)

	_, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
		StudentID: f.studentID, Type: 1,
	}, domain.PickupMethodManual)
	require.NoError(t, err)

	// 第一次通知
	r1, err := svc.NotifyOverdueParents(ctx(), testPickupKG, f.studentID)
	require.NoError(t, err)
	assert.Equal(t, true, r1["notified"], "应发出通知")
	assert.Equal(t, 1, r1["sent_count"])

	// 第二次通知
	r2, err := svc.NotifyOverdueParents(ctx(), testPickupKG, f.studentID)
	require.NoError(t, err)
	assert.Equal(t, true, r2["notified"])

	// 两次都应留痕，不覆盖历史
	var msgs []domain.Message
	require.NoError(t, f.svc.db.Where("user_id = ? AND type = ?",
		parentUserID, string(domain.MessageTypePickup)).Order("id ASC").Find(&msgs).Error)
	assert.Len(t, msgs, 2, "两次触发应留两条记录，不得覆盖")
	assert.Equal(t, msgs[0].Title, msgs[1].Title)
	assert.NotZero(t, msgs[0].ReferenceID)
}

// 非滞留状态不得一键通知
func TestAdminOverdue_NotifyRejectsNonOverdue(t *testing.T) {
	// deadline 23:59 → 恒未过放学时刻 → 不算滞留
	f := newPickupFixture(t, "23:59", testPickupClass)
	svc := newAdminPickupService(f)
	f.makeParent(t)

	_, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
		StudentID: f.studentID, Type: 1,
	}, domain.PickupMethodManual)
	require.NoError(t, err)

	_, err = svc.NotifyOverdueParents(ctx(), testPickupKG, f.studentID)
	assert.Error(t, err, "非滞留状态不得一键通知")
}

// TC-NEW-PKU-110 超时未接计数（累计口径）
func TestAdminOverdue_CountCumulative(t *testing.T) {
	f := newPickupFixture(t, "00:00", testPickupClass)
	svc := newAdminPickupService(f)
	authID := f.makeAuthorization(t, "奶奶（张三）", 1, 1, nil)

	// 送入 → 计 1
	_, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
		StudentID: f.studentID, Type: 1,
	}, domain.PickupMethodManual)
	require.NoError(t, err)

	n, err := svc.CountOverdueNotPicked(ctx(), testPickupKG, f.classID, "")
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)

	// 随后接走 → 仍计 1（不清零）
	_, err = f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
		StudentID: f.studentID, Type: 2, AuthorizationID: &authID,
	}, domain.PickupMethodManual)
	require.NoError(t, err)

	n, err = svc.CountOverdueNotPicked(ctx(), testPickupKG, f.classID, "")
	require.NoError(t, err)
	assert.Equal(t, int64(1), n, "随后被接走不得清零")
}

var _ = context.Background
var _ = time.Now
