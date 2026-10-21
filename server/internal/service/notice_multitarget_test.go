package service

import (
	"context"
	"fmt"
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

// 这些测试打真实 MySQL，覆盖 notice-multitarget spec 的 6 条 Requirement 主路径：
// 多班发布、非任教班/空目标拒绝、年级快照+混用去重、多孩合并、可见范围、两端已读率一致。
// 对应 spec: openspec/changes/notice-multitarget/specs/notice-targeting/spec.md
// ID 落在 9902xx 区间，t.Cleanup 硬删。

const (
	testNMTKG       uint64 = 990201
	testNMTTeacher  uint64 = 990201
	testNMTAdmin    uint64 = 990202
	testNMTParent   uint64 = 990210
	testNMTParent2  uint64 = 990211 // 无关班家长
)

var nmtDB *gorm.DB

func openNMTTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	if nmtDB != nil {
		return nmtDB
	}
	cfgPath := filepath.Join("..", "..", "configs", "app.ini")
	if _, err := os.Stat(cfgPath); err != nil {
		t.Skipf("跳过：找不到 %s（本地无 DB 配置）", cfgPath)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.MySQL.DSN == "" {
		t.Skip("跳过：MySQL DSN 为空")
	}
	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&domain.NoticeTarget{}); err != nil {
		t.Fatalf("auto migrate notice_targets: %v", err)
	}
	nmtDB = db
	return db
}

// nmtFixture：老师（任教小一/小二，不教无关班）、管理端、双孩家长（两孩分属两班）、无关班家长。
type nmtFixture struct {
	db       *gorm.DB
	repo     *repository.NoticeRepo
	svcAdmin *NoticeService
	svcT     *TeacherPortalService
	svcP     *ParentPortalService
	tc       *TeacherPortalContext
	pcMulti  *ParentPortalContext // 双孩家长
	pcMiss   *ParentPortalContext // 无关班家长
	classA   uint64               // 小一班（目标班1）
	classB   uint64               // 小二班（目标班2）
	classC   uint64               // 无关班
}

func newNMTCleanup(t *testing.T, db *gorm.DB) {
	t.Helper()
	ctx := context.Background()
	del := func(sql string, args ...any) {
		if err := db.WithContext(ctx).Exec(sql, args...).Error; err != nil {
			t.Logf("cleanup: %v", err)
		}
	}
	del("DELETE FROM notice_read_logs WHERE user_id >= 990200 AND user_id < 990300")
	del("DELETE FROM notice_targets WHERE kindergarten_id = ?", testNMTKG)
	del("DELETE FROM notices WHERE kindergarten_id = ?", testNMTKG)
	// 先删 class_teachers（含 990203 非目标老师），否则外键挡住删班级
	del("DELETE FROM class_teachers WHERE user_id >= 990200 AND user_id < 990300")
	del("DELETE FROM student_parents WHERE user_id >= 990200 AND user_id < 990300")
	del("DELETE FROM students WHERE kindergarten_id = ?", testNMTKG)
	del("DELETE FROM classes WHERE id >= 990202 AND id < 990210")
	del("DELETE FROM content_audits WHERE kindergarten_id = ?", testNMTKG)
}

func newNMTFixture(t *testing.T) *nmtFixture {
	t.Helper()
	db := openNMTTestDB(t)
	ctx := context.Background()
	validDate := time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC)

	newNMTCleanup(t, db)

	var kgCount int64
	db.WithContext(ctx).Model(&domain.Kindergarten{}).Where("id = ?", testNMTKG).Count(&kgCount)
	if kgCount == 0 {
		require.NoError(t, db.WithContext(ctx).Create(&domain.Kindergarten{
			BaseModel: domain.BaseModel{ID: testNMTKG}, Name: "测试园所-通知多目标",
		}).Error)
	}
	for _, u := range []struct {
		id   uint64
		phone, nick string
	}{
		{testNMTTeacher, "13900002001", "测试多班老师"},
		{testNMTAdmin, "13900002002", "测试管理端"},
		{testNMTParent, "13900002010", "双孩家长"},
		{testNMTParent2, "13900002011", "无关班家长"},
	} {
		var c int64
		db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", u.id).Count(&c)
		if c == 0 {
			require.NoError(t, db.WithContext(ctx).Create(&domain.User{
				BaseModel: domain.BaseModel{ID: u.id}, Phone: u.phone, Nickname: u.nick,
			}).Error)
		}
	}

	f := &nmtFixture{
		db: db, repo: repository.NewNoticeRepo(db),
		classA: 990202, classB: 990203, classC: 990204,
	}
	classes := []struct {
		id    uint64
		name  string
		grade string
	}{
		{f.classA, "测试小一班", "小班"},
		{f.classB, "测试小二班", "小班"},
		{f.classC, "测试中一班", "中班"},
	}
	for _, cl := range classes {
		require.NoError(t, db.WithContext(ctx).Create(&domain.Class{
			BaseModel: domain.BaseModel{ID: cl.id}, KindergartenID: testNMTKG,
			Name: cl.name, Grade: cl.grade, Status: 1,
		}).Error)
	}

	// 老师任教小一/小二（不教无关班）
	for _, classID := range []uint64{f.classA, f.classB} {
		require.NoError(t, db.WithContext(ctx).Create(&domain.ClassTeacher{
			ClassID: classID, UserID: testNMTTeacher, Title: "teacher", Status: 1,
		}).Error)
	}

	// 双孩家长：两孩分属小一/小二
	for i, classID := range []uint64{f.classA, f.classB} {
		studentID := uint64(990220 + i)
		require.NoError(t, db.WithContext(ctx).Create(&domain.Student{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: studentID}},
			KindergartenID:  testNMTKG, ClassID: classID,
			Name: fmt.Sprintf("测试学生%d", studentID), Birthday: validDate, EnrollDate: validDate, Status: 1,
		}).Error)
		require.NoError(t, db.WithContext(ctx).Create(&domain.StudentParent{
			StudentID: studentID, UserID: testNMTParent, Relation: "mother",
		}).Error)
	}
	// 无关班家长
	require.NoError(t, db.WithContext(ctx).Create(&domain.Student{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: 990223}},
		KindergartenID:  testNMTKG, ClassID: f.classC,
		Name: "测试学生990223", Birthday: validDate, EnrollDate: validDate, Status: 1,
	}).Error)
	require.NoError(t, db.WithContext(ctx).Create(&domain.StudentParent{
		StudentID: 990223, UserID: testNMTParent2, Relation: "father",
	}).Error)

	f.repo = repository.NewNoticeRepo(db)
	f.svcAdmin = NewNoticeService(f.repo, repository.NewOperationLogRepo(db))
	f.svcT = NewTeacherPortalService(db, nil, repository.NewPickupRepo(db), repository.NewMessageRepo(db), f.repo, nil, config.FaceConfig{})
	f.svcP = NewParentPortalService(db, repository.NewParentRepo(db), repository.NewUserRoleRepo(db),
		repository.NewStudentRepo(db), repository.NewClassRepo(db), repository.NewPickupRepo(db),
		repository.NewCalendarRepo(db), f.repo)
	f.tc = &TeacherPortalContext{UserID: testNMTTeacher, KindergartenID: testNMTKG, ClassID: f.classA}
	f.pcMulti = &ParentPortalContext{
		UserID: testNMTParent, KindergartenID: testNMTKG,
		Students: []domain.Student{
			{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: 990220}}, ClassID: f.classA},
			{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: 990221}}, ClassID: f.classB},
		},
	}
	f.pcMulti.CurrentStudent = &f.pcMulti.Students[0]
	f.pcMiss = &ParentPortalContext{
		UserID: testNMTParent2, KindergartenID: testNMTKG,
		Students: []domain.Student{
			{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: 990223}}, ClassID: f.classC},
		},
	}
	f.pcMiss.CurrentStudent = &f.pcMiss.Students[0]

	t.Cleanup(func() { newNMTCleanup(t, db) })
	return f
}

// TC-NEW-NTC-0001：老师一次发给两个班
func TestTeacherCreateNotice_TwoClasses(t *testing.T) {
	f := newNMTFixture(t)
	ctx := context.Background()

	out, err := f.svcT.CreateNotice(ctx, f.tc, TeacherNoticeInput{
		Title: "双班作业通知", Content: "正文", ClassIDs: []uint64{f.classA, f.classB},
	})
	require.NoError(t, err)
	noticeID := out["id"].(uint64)

	ids, err := f.repo.ListNoticeTargetClassIDs(ctx, noticeID)
	require.NoError(t, err)
	assert.Equal(t, []uint64{f.classA, f.classB}, ids, "目标应为小一班与小二班")

	// 两班家长均可见（双孩家长两孩分属两班）
	list, err := f.svcP.ListNotices(ctx, f.pcMulti, "", 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(1), list["total"], "双孩家长列表只出一条（合并）")
}

// TC-NEW-NTC-0002：非任教班被拒
func TestTeacherCreateNotice_NonTeachedClassRejected(t *testing.T) {
	f := newNMTFixture(t)
	_, err := f.svcT.CreateNotice(context.Background(), f.tc, TeacherNoticeInput{
		Title: "越权通知", Content: "正文", ClassIDs: []uint64{f.classC},
	})
	assert.ErrorIs(t, err, ErrTeacherForbidden, "选到非任教班应拒绝")

	var n int64
	f.db.Model(&domain.Notice{}).Where("kindergarten_id = ?", testNMTKG).Count(&n)
	assert.Equal(t, int64(0), n, "不生成通知")
}

// TC-NEW-NTC-0003：目标为空被拒
func TestTeacherCreateNotice_EmptyTargetsRejected(t *testing.T) {
	f := newNMTFixture(t)
	_, err := f.svcT.CreateNotice(context.Background(), f.tc, TeacherNoticeInput{
		Title: "空目标通知", Content: "正文",
	})
	assert.ErrorIs(t, err, ErrTeacherInvalidParams, "目标集合为空应拒绝")
}

// TC-NEW-NTC-0004/0005：年级快照展开 + 混用去重
func TestAdminCreateNotice_GradeSnapshotAndDedupe(t *testing.T) {
	f := newNMTFixture(t)
	ctx := context.Background()

	// 圈「小班」（含小一、小二）又手工勾小一班 → 小一只出现一次
	out, err := f.svcAdmin.CreateNotice(ctx, testNMTKG, testNMTAdmin, CreateNoticeReq{
		ClassIDs: []uint64{f.classA}, Grades: []string{"小班"},
		Category: "other", Title: "年级圈选通知", Content: "正文",
	}, "", "")
	require.NoError(t, err)
	noticeID := out["id"].(uint64)

	ids, err := f.repo.ListNoticeTargetClassIDs(ctx, noticeID)
	require.NoError(t, err)
	assert.Equal(t, []uint64{f.classA, f.classB}, ids, "小班展开为小一+小二，与手工勾选去重")

	// 快照语义：发布后新建的小三班不被覆盖
	require.NoError(t, f.db.WithContext(ctx).Create(&domain.Class{
		BaseModel: domain.BaseModel{ID: 990205}, KindergartenID: testNMTKG,
		Name: "测试小三班", Grade: "小班", Status: 1,
	}).Error)
	t.Cleanup(func() {
		f.db.WithContext(ctx).Exec("DELETE FROM classes WHERE id = ?", 990205)
	})
	ids, err = f.repo.ListNoticeTargetClassIDs(ctx, noticeID)
	require.NoError(t, err)
	assert.Equal(t, []uint64{f.classA, f.classB}, ids, "新建班级不卷入已发布通知（快照）")
}

// TC-NEW-NTC-0006/0007/0008：多孩合并、单孩命中可见、无关家长不可见
func TestParentVisibility_HitMissAndMerge(t *testing.T) {
	f := newNMTFixture(t)
	ctx := context.Background()

	_, err := f.svcT.CreateNotice(ctx, f.tc, TeacherNoticeInput{
		Title: "目标小一+小二", Content: "正文", ClassIDs: []uint64{f.classA, f.classB},
	})
	require.NoError(t, err)

	// 双孩家长：只出一条
	list, err := f.svcP.ListNotices(ctx, f.pcMulti, "", 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(1), list["total"])

	// 无关班家长：列表为空，详情 404
	listMiss, err := f.svcP.ListNotices(ctx, f.pcMiss, "", 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(0), listMiss["total"], "家长看不到无关班级的通知")

	var noticeID uint64
	f.db.WithContext(ctx).Model(&domain.Notice{}).Where("kindergarten_id = ?", testNMTKG).
		Order("id DESC").Limit(1).Select("id").Scan(&noticeID)
	_, err = f.svcP.GetNotice(ctx, f.pcMiss, noticeID)
	assert.ErrorIs(t, err, ErrParentNotFound, "无关家长不可见详情")

	// 确认一次对双孩生效
	_, err = f.svcP.ConfirmNotice(ctx, f.pcMulti, noticeID)
	require.NoError(t, err)
	var confirmed int64
	f.db.WithContext(ctx).Model(&domain.NoticeReadLog{}).
		Where("notice_id = ? AND user_id = ? AND confirmed_at IS NOT NULL", noticeID, testNMTParent).
		Count(&confirmed)
	assert.Equal(t, int64(1), confirmed, "确认一次（按用户），对全部孩子生效")
}

// TC-NEW-NTC-0009/0010：老师命中可查、非目标老师不可见
func TestTeacherVisibility_HitAndMiss(t *testing.T) {
	f := newNMTFixture(t)
	ctx := context.Background()

	_, err := f.svcT.CreateNotice(ctx, f.tc, TeacherNoticeInput{
		Title: "目标小一", Content: "正文", ClassIDs: []uint64{f.classA},
	})
	require.NoError(t, err)
	var noticeID uint64
	f.db.WithContext(ctx).Model(&domain.Notice{}).Where("kindergarten_id = ?", testNMTKG).
		Order("id DESC").Limit(1).Select("id").Scan(&noticeID)

	// 任教小一的老师可查（命中）
	_, err = f.svcT.GetNotice(ctx, f.tc, noticeID)
	require.NoError(t, err)

	// 不教小一的老师不可见
	otherTc := &TeacherPortalContext{UserID: 990203, KindergartenID: testNMTKG, ClassID: f.classC}
	var otherUserCount int64
	f.db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", 990203).Count(&otherUserCount)
	if otherUserCount == 0 {
		require.NoError(t, f.db.WithContext(ctx).Create(&domain.User{
			BaseModel: domain.BaseModel{ID: 990203}, Phone: "13900002003", Nickname: "非目标老师",
		}).Error)
	}
	require.NoError(t, f.db.WithContext(ctx).Create(&domain.ClassTeacher{
		ClassID: f.classC, UserID: 990203, Title: "teacher", Status: 1,
	}).Error)
	_, err = f.svcT.GetNotice(ctx, otherTc, noticeID)
	assert.ErrorIs(t, err, ErrTeacherNotFound, "非目标班老师不可见")
}

// TC-NEW-NTC-0012：两端已读率一致（双班通知+双孩家长）
func TestReadRate_ConsistentAcrossEnds(t *testing.T) {
	f := newNMTFixture(t)
	ctx := context.Background()

	out, err := f.svcT.CreateNotice(ctx, f.tc, TeacherNoticeInput{
		Title: "两端一致性", Content: "正文", ClassIDs: []uint64{f.classA, f.classB},
	})
	require.NoError(t, err)
	noticeID := out["id"].(uint64)

	// 双孩家长读一次
	require.NoError(t, f.svcP.MarkNoticeRead(ctx, f.pcMulti, noticeID))

	tOut, err := f.svcT.GetNotice(ctx, f.tc, noticeID)
	require.NoError(t, err)
	aOut, err := f.svcAdmin.GetNoticeDetail(ctx, testNMTKG, noticeID)
	require.NoError(t, err)

	tStats := tOut["read_stats"].(map[string]any)
	aStats := aOut["read_stats"].(map[string]interface{})
	assert.Equal(t, aStats["total_count"], tStats["total_count"], "分母两端一致")
	assert.Equal(t, aStats["read_count"], tStats["read_count"], "分子两端一致")
	assert.Equal(t, aStats["read_rate"], tStats["read_rate"], "已读率两端一致")
	// 双孩家长在分母中只计一人：目标范围家长数 = 2（双孩家长 + 无关班家长不在目标内→只有双孩家长？）
	// 小一、小二的学生家长只有双孩家长一位（990223 在无关班），故分母=1
	assert.Equal(t, int64(1), tStats["total_count"])
	assert.Equal(t, int64(1), tStats["read_count"])
}
