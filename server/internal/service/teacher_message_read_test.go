package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// B22：老师端聚合消息的未读态与已读接口。
//
// 打真实 MySQL（本地 configs/app.ini），固定 9970xx 夹具，t.Cleanup 硬删 0 残留。
// 建表走测试内 AutoMigrate：本切片不允许重启 8080，测试不能依赖后端启动时迁移；
// provider.go 的 AutoMigrate 登记与之保持同一模型。
//
// 覆盖：未读→标已读→未读数变化、重复标记幂等、read-all 归零、
// A/B 老师隔离、不可见/不存在 404、系统消息按条区分、批量查询无 N+1、
// 不污染通知已读率。

const (
	testTMRKG uint64 = 997001

	testTMRClassA uint64 = 997010 // 老师 A 的班
	testTMRClassB uint64 = 997011 // 同园别班（老师 B）

	testTMRTeacherA uint64 = 997020
	testTMRTeacherB uint64 = 997021
	testTMRParent   uint64 = 997022

	testTMRStudentA uint64 = 997030
	testTMRStudentB uint64 = 997031

	testTMRNoticeA1 uint64 = 997040 // A 发布，已发布
	testTMRNoticeA2 uint64 = 997041 // A 发布，用例中改为已撤回
	testTMRNoticeB  uint64 = 997042 // B 发布

	testTMRLeaveA1 uint64 = 997050
	testTMRLeaveA2 uint64 = 997051
	testTMRLeaveB  uint64 = 997052

	testTMRMsgA1 uint64 = 997060 // 系统消息，reference_id 刻意写 0
	testTMRMsgA2 uint64 = 997061 // 同上：证明读状态按 messages.id 区分
	testTMRMsgB  uint64 = 997062
)

func cleanupTMRData(t *testing.T, db *gorm.DB) {
	t.Helper()
	del := func(sql string, args ...any) {
		if err := db.Exec(sql, args...).Error; err != nil {
			t.Logf("cleanup: %v", err)
		}
	}
	// 读记录没有软删，按老师 ID 硬删（自增 ID 不可预测）。
	del("DELETE FROM teacher_message_reads WHERE user_id IN (?, ?)", testTMRTeacherA, testTMRTeacherB)
	del("DELETE FROM notice_read_logs WHERE notice_id BETWEEN 997040 AND 997049")
	del("DELETE FROM notice_targets WHERE notice_id BETWEEN 997040 AND 997049")
	del("DELETE FROM notices WHERE id BETWEEN 997040 AND 997049")
	del("DELETE FROM leave_attachments WHERE leave_id BETWEEN 997050 AND 997059")
	del("DELETE FROM leave_requests WHERE id BETWEEN 997050 AND 997059")
	del("DELETE FROM messages WHERE id BETWEEN 997060 AND 997069")
	del("DELETE FROM student_parents WHERE student_id BETWEEN 997030 AND 997039")
	del("DELETE FROM students WHERE id BETWEEN 997030 AND 997039")
	del("DELETE FROM classes WHERE id BETWEEN 997010 AND 997019")
	del("DELETE FROM users WHERE id BETWEEN 997020 AND 997029")
	del("DELETE FROM kindergartens WHERE id = ?", testTMRKG)
}

type tmrFixture struct {
	db  *gorm.DB
	svc *TeacherPortalService
	tcA *TeacherPortalContext
	tcB *TeacherPortalContext
}

func newTMRFixture(t *testing.T) *tmrFixture {
	t.Helper()
	db := openCLPTestDB(t)
	require.NoError(t, db.AutoMigrate(&domain.TeacherMessageRead{}))
	cleanupTMRData(t, db)
	t.Cleanup(func() { cleanupTMRData(t, db) })

	require.NoError(t, db.Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: testTMRKG}, Name: "测试园所-消息已读",
	}).Error)
	for _, cl := range []struct {
		id   uint64
		name string
	}{
		{testTMRClassA, "消息一班"},
		{testTMRClassB, "消息二班"},
	} {
		require.NoError(t, db.Create(&domain.Class{
			BaseModel: domain.BaseModel{ID: cl.id}, KindergartenID: testTMRKG, Name: cl.name, Grade: "小班", Status: 1,
		}).Error)
	}
	for _, u := range []struct {
		id    uint64
		nick  string
		phone string
	}{
		{testTMRTeacherA, "消息老师A", "13900009951"},
		{testTMRTeacherB, "消息老师B", "13900009952"},
		{testTMRParent, "消息家长", "13900009953"},
	} {
		require.NoError(t, db.Create(&domain.User{
			BaseModel: domain.BaseModel{ID: u.id}, Nickname: u.nick, Phone: u.phone, PasswordHash: "x",
		}).Error)
	}

	validDate := time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC)
	for _, st := range []struct {
		id      uint64
		classID uint64
	}{
		{testTMRStudentA, testTMRClassA},
		{testTMRStudentB, testTMRClassB},
	} {
		require.NoError(t, db.Create(&domain.Student{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: st.id}},
			KindergartenID:  testTMRKG, ClassID: st.classID, Name: "消息学生",
			Birthday: validDate, EnrollDate: validDate, Status: 1,
		}).Error)
	}

	now := time.Now()
	mkNotice := func(id, classID, creator uint64, title string) {
		require.NoError(t, db.Create(&domain.Notice{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: id}},
			KindergartenID:  testTMRKG, ClassID: classID, CreatorID: creator,
			Category: domain.NoticeCategoryActivity, Title: title, Content: title + "内容",
			Status: domain.NoticeStatusPublished,
		}).Error)
	}
	mkNotice(testTMRNoticeA1, testTMRClassA, testTMRTeacherA, "班级活动通知")
	mkNotice(testTMRNoticeA2, testTMRClassA, testTMRTeacherA, "作业通知")
	mkNotice(testTMRNoticeB, testTMRClassB, testTMRTeacherB, "二班活动通知")

	mkLeave := func(id, classID, studentID uint64, minutesAgo int) {
		require.NoError(t, db.Create(&domain.LeaveRequest{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{
				ID: id, CreatedAt: now.Add(-time.Duration(minutesAgo) * time.Minute),
			}},
			KindergartenID: testTMRKG, ClassID: classID, StudentID: studentID, ApplicantID: testTMRParent,
			Reason: "消息已读用例", StartDate: validDate, EndDate: validDate, Days: 1,
			Status: domain.LeaveStatusPending,
		}).Error)
	}
	mkLeave(testTMRLeaveA1, testTMRClassA, testTMRStudentA, 3)
	mkLeave(testTMRLeaveA2, testTMRClassA, testTMRStudentA, 2)
	mkLeave(testTMRLeaveB, testTMRClassB, testTMRStudentB, 1)

	mkMsg := func(id, userID uint64, title string) {
		require.NoError(t, db.Create(&domain.Message{
			BaseModel: domain.BaseModel{ID: id},
			UserID:    userID, Type: string(domain.MessageTypeSystem), Title: title,
			Content: title + "内容", ReferenceID: 0,
		}).Error)
	}
	mkMsg(testTMRMsgA1, testTMRTeacherA, "系统维护通知")
	mkMsg(testTMRMsgA2, testTMRTeacherA, "版本更新通知")
	mkMsg(testTMRMsgB, testTMRTeacherB, "二班系统通知")

	// 通知已读率样本：目标班 1 位家长已读 → readUsers=1 / parentTotal=1。
	require.NoError(t, db.Create(&domain.NoticeTarget{
		NoticeID: testTMRNoticeA1, ClassID: testTMRClassA, KindergartenID: testTMRKG,
	}).Error)
	require.NoError(t, db.Create(&domain.StudentParent{
		StudentID: testTMRStudentA, UserID: testTMRParent, Relation: domain.RelationMother, IsPrimary: true,
	}).Error)
	require.NoError(t, db.Create(&domain.NoticeReadLog{
		NoticeID: testTMRNoticeA1, UserID: testTMRParent, ReadAt: now,
	}).Error)

	return &tmrFixture{
		db:  db,
		svc: NewTeacherPortalService(db, nil, nil, nil, nil, nil, config.FaceConfig{}),
		tcA: &TeacherPortalContext{UserID: testTMRTeacherA, KindergartenID: testTMRKG, ClassID: testTMRClassA},
		tcB: &TeacherPortalContext{UserID: testTMRTeacherB, KindergartenID: testTMRKG, ClassID: testTMRClassB},
	}
}

func tmrList(t *testing.T, svc *TeacherPortalService, tc *TeacherPortalContext) map[string]any {
	t.Helper()
	data, err := svc.ListTeacherMessages(context.Background(), tc, 1, 50)
	require.NoError(t, err)
	return data
}

func tmrFind(t *testing.T, data map[string]any, msgType string, ref uint64) map[string]any {
	t.Helper()
	list, ok := data["list"].([]map[string]any)
	require.True(t, ok, "list 必须是 []map[string]any")
	for _, item := range list {
		if item["type"] == msgType && item["reference_id"] == ref {
			return item
		}
	}
	t.Fatalf("未找到消息 type=%s reference_id=%d", msgType, ref)
	return nil
}

func tmrReadRows(t *testing.T, db *gorm.DB, userID uint64) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(&domain.TeacherMessageRead{}).Where("user_id = ?", userID).Count(&n).Error)
	return n
}

// 未读 → 标已读 → 未读数下降；重复标记幂等。
func TestTeacherMessageReadUnreadCountFlow(t *testing.T) {
	f := newTMRFixture(t)
	ctx := context.Background()

	list := tmrList(t, f.svc, f.tcA)
	assert.EqualValues(t, 6, list["total"], "2 通知 + 2 请假 + 2 系统")
	assert.EqualValues(t, 6, list["unread_count"])
	assert.Equal(t, false, tmrFind(t, list, domain.TeacherMsgTypeNotice, testTMRNoticeA1)["is_read"])

	resp, err := f.svc.MarkTeacherMessageRead(ctx, f.tcA, domain.TeacherMsgTypeNotice, testTMRNoticeA1)
	require.NoError(t, err)
	assert.Equal(t, true, resp["is_read"])

	list = tmrList(t, f.svc, f.tcA)
	assert.EqualValues(t, 5, list["unread_count"])
	assert.Equal(t, true, tmrFind(t, list, domain.TeacherMsgTypeNotice, testTMRNoticeA1)["is_read"])
	assert.Equal(t, false, tmrFind(t, list, domain.TeacherMsgTypeNotice, testTMRNoticeA2)["is_read"])

	// 幂等：重复标记不新增记录、不改变未读数。
	_, err = f.svc.MarkTeacherMessageRead(ctx, f.tcA, domain.TeacherMsgTypeNotice, testTMRNoticeA1)
	require.NoError(t, err)
	assert.EqualValues(t, 5, tmrList(t, f.svc, f.tcA)["unread_count"])
	assert.EqualValues(t, 1, tmrReadRows(t, f.db, testTMRTeacherA))
}

// 契约形状：顶层 unread_count + 每条 is_read（端上靠这两个字段渲染未读点与计数）。
func TestTeacherMessageListJSONShape(t *testing.T) {
	f := newTMRFixture(t)

	raw, err := json.Marshal(tmrList(t, f.svc, f.tcA))
	require.NoError(t, err)
	var shape struct {
		List        []map[string]json.RawMessage `json:"list"`
		UnreadCount json.RawMessage              `json:"unread_count"`
		Total       int64                        `json:"total"`
	}
	require.NoError(t, json.Unmarshal(raw, &shape))
	assert.EqualValues(t, 6, shape.Total)
	assert.JSONEq(t, `6`, string(shape.UnreadCount))
	require.Len(t, shape.List, 6)
	for _, item := range shape.List {
		require.Contains(t, item, "is_read")
		assert.JSONEq(t, `false`, string(item["is_read"]))
	}
}

// 系统消息的 reference_id 常为 0：读状态必须按 messages.id 逐条区分，
// 且不得改写 messages.is_read（那是另一套口径）。
func TestTeacherMessageReadSystemIsPerMessage(t *testing.T) {
	f := newTMRFixture(t)

	_, err := f.svc.MarkTeacherMessageRead(context.Background(), f.tcA, domain.TeacherMsgTypeSystem, testTMRMsgA1)
	require.NoError(t, err)

	list := tmrList(t, f.svc, f.tcA)
	assert.Equal(t, true, tmrFind(t, list, domain.TeacherMsgTypeSystem, testTMRMsgA1)["is_read"])
	assert.Equal(t, false, tmrFind(t, list, domain.TeacherMsgTypeSystem, testTMRMsgA2)["is_read"], "另一条系统消息不得被连带已读")
	assert.EqualValues(t, 5, list["unread_count"])

	var msg domain.Message
	require.NoError(t, f.db.First(&msg, testTMRMsgA1).Error)
	assert.False(t, msg.IsRead, "老师消息读记录不改 messages.is_read")
}

// read-all 全量置读、只计新增；重复调用返回 0；A 的读记录不影响 B。
func TestTeacherMessageReadAll(t *testing.T) {
	f := newTMRFixture(t)
	ctx := context.Background()

	// 先读一条，验证 marked_count 只计新增。
	_, err := f.svc.MarkTeacherMessageRead(ctx, f.tcA, domain.TeacherMsgTypeLeave, testTMRLeaveA1)
	require.NoError(t, err)

	marked, err := f.svc.MarkAllTeacherMessagesRead(ctx, f.tcA)
	require.NoError(t, err)
	assert.EqualValues(t, 5, marked, "6 条中已读 1 条，只有 5 条新增")

	list := tmrList(t, f.svc, f.tcA)
	assert.EqualValues(t, 0, list["unread_count"])
	items := list["list"].([]map[string]any)
	require.Len(t, items, 6)
	for _, item := range items {
		assert.Equal(t, true, item["is_read"], "%s/%v 应已读", item["type"], item["reference_id"])
	}

	marked, err = f.svc.MarkAllTeacherMessagesRead(ctx, f.tcA)
	require.NoError(t, err)
	assert.EqualValues(t, 0, marked, "重复全部已读必须幂等返回 0")
	assert.EqualValues(t, 6, tmrReadRows(t, f.db, testTMRTeacherA))

	listB := tmrList(t, f.svc, f.tcB)
	assert.EqualValues(t, 3, listB["unread_count"], "B 的消息不受 A 的已读影响")
}

// A/B 老师隔离：别班/别园消息一律 404，互不污染未读数。
func TestTeacherMessageReadIsolation(t *testing.T) {
	f := newTMRFixture(t)
	ctx := context.Background()

	for _, c := range []struct {
		tc  *TeacherPortalContext
		typ string
		id  uint64
	}{
		{f.tcB, domain.TeacherMsgTypeNotice, testTMRNoticeA1},
		{f.tcA, domain.TeacherMsgTypeNotice, testTMRNoticeB},
		{f.tcA, domain.TeacherMsgTypeLeave, testTMRLeaveB},
		{f.tcA, domain.TeacherMsgTypeSystem, testTMRMsgB},
	} {
		_, err := f.svc.MarkTeacherMessageRead(ctx, c.tc, c.typ, c.id)
		require.ErrorIs(t, err, ErrTeacherNotFound, "%s/%d 不属于当前老师，必须 404", c.typ, c.id)
	}

	_, err := f.svc.MarkTeacherMessageRead(ctx, f.tcB, domain.TeacherMsgTypeNotice, testTMRNoticeB)
	require.NoError(t, err)
	assert.EqualValues(t, 6, tmrList(t, f.svc, f.tcA)["unread_count"], "B 置读不影响 A")
	assert.EqualValues(t, 2, tmrList(t, f.svc, f.tcB)["unread_count"])
}

// 不存在 / 不可见 / 非法 type / 非法 id 一律拒绝，不写读记录。
func TestTeacherMessageReadRejectsUnknown(t *testing.T) {
	f := newTMRFixture(t)
	ctx := context.Background()

	for _, c := range []struct {
		typ string
		id  uint64
	}{
		{domain.TeacherMsgTypeNotice, 997049},
		{domain.TeacherMsgTypeLeave, 997059},
		{domain.TeacherMsgTypeSystem, 997069},
	} {
		_, err := f.svc.MarkTeacherMessageRead(ctx, f.tcA, c.typ, c.id)
		require.ErrorIs(t, err, ErrTeacherNotFound, "%s/%d 不存在，必须 404", c.typ, c.id)
	}

	_, err := f.svc.MarkTeacherMessageRead(ctx, f.tcA, "task", testTMRNoticeA1)
	require.ErrorIs(t, err, ErrTeacherInvalidParams)
	_, err = f.svc.MarkTeacherMessageRead(ctx, f.tcA, domain.TeacherMsgTypeNotice, 0)
	require.ErrorIs(t, err, ErrTeacherInvalidParams)

	// 已撤回通知不在聚合列表内，也不能标记。
	require.NoError(t, f.db.Model(&domain.Notice{}).
		Where("id = ?", testTMRNoticeA2).Update("status", domain.NoticeStatusRevoked).Error)
	_, err = f.svc.MarkTeacherMessageRead(ctx, f.tcA, domain.TeacherMsgTypeNotice, testTMRNoticeA2)
	require.ErrorIs(t, err, ErrTeacherNotFound)

	assert.EqualValues(t, 0, tmrReadRows(t, f.db, testTMRTeacherA), "被拒绝的标记不得留下读记录")
}

// 性能守卫：整页 is_read 只发一条 teacher_message_reads 查询，禁止逐条查。
func TestTeacherMessageListReadSetSingleQuery(t *testing.T) {
	f := newTMRFixture(t)
	_, err := f.svc.MarkTeacherMessageRead(context.Background(), f.tcA, domain.TeacherMsgTypeNotice, testTMRNoticeA1)
	require.NoError(t, err)

	counter := &clpSQLLogger{Interface: gormlogger.Default.LogMode(gormlogger.Silent)}
	svc := NewTeacherPortalService(f.db.Session(&gorm.Session{Logger: counter}), nil, nil, nil, nil, nil, config.FaceConfig{})
	_, err = svc.ListTeacherMessages(context.Background(), f.tcA, 1, 50)
	require.NoError(t, err)

	assert.Equal(t, 1, counter.countContaining("teacher_message_reads"), "6 条消息的 is_read 只能有一条批量查询")
}

// 已读接口只影响老师消息列表，不得写 notice_read_logs / 改变家长已读率。
func TestTeacherMessageReadDoesNotAffectNoticeReadRate(t *testing.T) {
	f := newTMRFixture(t)
	ctx := context.Background()
	notices := repository.NewNoticeRepo(f.db)

	readBefore, _, totalBefore, err := notices.NoticeReadStats(ctx, testTMRKG, testTMRNoticeA1)
	require.NoError(t, err)
	require.EqualValues(t, 1, readBefore)
	require.EqualValues(t, 1, totalBefore)

	_, err = f.svc.MarkTeacherMessageRead(ctx, f.tcA, domain.TeacherMsgTypeNotice, testTMRNoticeA1)
	require.NoError(t, err)

	readAfter, _, totalAfter, err := notices.NoticeReadStats(ctx, testTMRKG, testTMRNoticeA1)
	require.NoError(t, err)
	assert.EqualValues(t, readBefore, readAfter, "老师置读不得改变家长已读率分子")
	assert.EqualValues(t, totalBefore, totalAfter)

	var logs int64
	require.NoError(t, f.db.Model(&domain.NoticeReadLog{}).Where("notice_id = ?", testTMRNoticeA1).Count(&logs).Error)
	assert.EqualValues(t, 1, logs, "不得写入 notice_read_logs")

	// 同一条消息在老师列表里已读，但通知已读率仍是家长口径的 100%。
	item := tmrFind(t, tmrList(t, f.svc, f.tcA), domain.TeacherMsgTypeNotice, testTMRNoticeA1)
	assert.Equal(t, true, item["is_read"])
	assert.Equal(t, "100%", item["extra"].(map[string]any)["read_rate"])
}

// 清理证据：硬删后 9970xx 夹具 0 残留。
func TestTeacherMessageReadCleanupLeavesNoResidue(t *testing.T) {
	f := newTMRFixture(t)
	_, err := f.svc.MarkAllTeacherMessagesRead(context.Background(), f.tcA)
	require.NoError(t, err)

	cleanupTMRData(t, f.db)

	count := func(model any, query string, args ...any) int64 {
		var n int64
		require.NoError(t, f.db.Unscoped().Model(model).Where(query, args...).Count(&n).Error)
		return n
	}
	assert.EqualValues(t, 0, count(&domain.TeacherMessageRead{}, "user_id IN ?", []uint64{testTMRTeacherA, testTMRTeacherB}))
	assert.EqualValues(t, 0, count(&domain.Notice{}, "id BETWEEN ? AND ?", uint64(997040), uint64(997049)))
	assert.EqualValues(t, 0, count(&domain.LeaveRequest{}, "id BETWEEN ? AND ?", uint64(997050), uint64(997059)))
	assert.EqualValues(t, 0, count(&domain.Message{}, "id BETWEEN ? AND ?", uint64(997060), uint64(997069)))
	assert.EqualValues(t, 0, count(&domain.Kindergarten{}, "id = ?", testTMRKG))
}
