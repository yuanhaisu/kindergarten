package service

import (
	"context"
	"encoding/json"
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

// B17：家长端班级信息（backend/api/api-parent/08-class.md）与幼儿档案
// （09-student.md）字段补齐。
//
// 班级老师：title_label 映射、脱敏 phone、主班→副班→生活→其余排序、空数组；
// 幼儿档案：birthday / enroll_date 输出 YYYY-MM-DD，零值空串。
//
// 打真实 MySQL（本地 configs/app.ini），9950xx 固定夹具，t.Cleanup 硬删 0 残留。

const (
	testPCPKG      uint64 = 995001 // 园所
	testPCPClassA  uint64 = 995010 // 小一班：5 位老师（含无手机号与未知职务）
	testPCPClassB  uint64 = 995011 // 小二班：1 位老师（隔离用）
	testPCPClassC  uint64 = 995012 // 空班：无老师
	testPCPUser    uint64 = 995090 // 家长（上下文手工构造，不写 user_roles）
	testPCPHead    uint64 = 995001 // 主班老师
	testPCPAssist  uint64 = 995002 // 副班老师
	testPCPNursery uint64 = 995003 // 生活老师
	testPCPUnknown uint64 = 995004 // 未知职务码
	testPCPNoPhone uint64 = 995005 // 副班但无手机号
	testPCPClassBT uint64 = 995006 // 小二班老师
)

var pcpTestDB *gorm.DB

func openPCPTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	if pcpTestDB != nil {
		return pcpTestDB
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
	pcpTestDB = db
	return db
}

func cleanupPCPData(t *testing.T, db *gorm.DB) {
	t.Helper()
	ctx := context.Background()
	del := func(sql string, args ...any) {
		if err := db.WithContext(ctx).Exec(sql, args...).Error; err != nil {
			t.Logf("cleanup: %v", err)
		}
	}
	// 子表先删：student_parents 引用 students，class_teachers 引用 classes/users。
	del("DELETE FROM student_parents WHERE student_id BETWEEN 995100 AND 995199")
	del("DELETE FROM students WHERE id BETWEEN 995100 AND 995199")
	del("DELETE FROM class_teachers WHERE class_id BETWEEN 995010 AND 995019")
	del("DELETE FROM classes WHERE id BETWEEN 995010 AND 995019")
	del("DELETE FROM users WHERE id BETWEEN 995000 AND 995099")
	del("DELETE FROM kindergartens WHERE id = ?", testPCPKG)
}

type pcpFixture struct {
	db     *gorm.DB
	svc    *ParentPortalService
	classA *domain.Class
	classB *domain.Class
	classC *domain.Class
	pcA    *ParentPortalContext
	pcB    *ParentPortalContext
	pcC    *ParentPortalContext
	// birthA 是 A 孩子生日，按「今天减 4 年」生成，age 断言才不会随运行日期漂移。
	birthA time.Time
}

func newPCPFixture(t *testing.T) *pcpFixture {
	t.Helper()
	db := openPCPTestDB(t)
	ctx := context.Background()
	cleanupPCPData(t, db)

	require.NoError(t, db.WithContext(ctx).Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: testPCPKG}, Name: "测试园所-班级档案",
	}).Error)

	users := []struct {
		id          uint64
		phone, nick string
		avatar      string
	}{
		{testPCPHead, "13800001001", "测试主班老师", "https://cdn.example.com/avatar/head.jpg"},
		{testPCPAssist, "13900001002", "测试副班老师", "https://cdn.example.com/avatar/assist.jpg"},
		{testPCPNursery, "13700001003", "测试生活老师", ""},
		{testPCPUnknown, "13600001004", "测试未知职务老师", ""},
		{testPCPNoPhone, "", "测试无手机老师", ""},
		{testPCPClassBT, "13500001006", "小二班老师", ""},
	}
	for _, u := range users {
		require.NoError(t, db.WithContext(ctx).Create(&domain.User{
			BaseModel: domain.BaseModel{ID: u.id}, Phone: u.phone, Nickname: u.nick, Avatar: u.avatar,
		}).Error)
	}

	mkClass := func(id uint64, name string) *domain.Class {
		cl := &domain.Class{
			BaseModel: domain.BaseModel{ID: id}, KindergartenID: testPCPKG,
			Name: name, Grade: "小班", AcademicYear: "2025-2026", Status: 1,
		}
		require.NoError(t, db.WithContext(ctx).Create(cl).Error)
		return cl
	}
	f := &pcpFixture{db: db}
	f.classA = mkClass(testPCPClassA, "测试小一班")
	f.classB = mkClass(testPCPClassB, "测试小二班")
	f.classC = mkClass(testPCPClassC, "测试空班")

	// 故意乱序落库：顺序由 service 归一，而不是靠插入顺序。
	link := func(classID, userID uint64, title domain.TeacherTitle) {
		require.NoError(t, db.WithContext(ctx).Create(&domain.ClassTeacher{
			ClassID: classID, UserID: userID, Title: title, Status: 1,
		}).Error)
	}
	link(testPCPClassA, testPCPNursery, domain.TeacherTitleNursery)
	link(testPCPClassA, testPCPAssist, domain.TeacherTitleAssistant)
	link(testPCPClassA, testPCPHead, domain.TeacherTitleHead)
	link(testPCPClassA, testPCPUnknown, domain.TeacherTitle("volunteer"))
	link(testPCPClassA, testPCPNoPhone, domain.TeacherTitleAssistant)
	link(testPCPClassB, testPCPClassBT, domain.TeacherTitleHead)

	// A 班 3 名（在读 1 + 非在读 1 + 软删 1），B 班在读 1。
	f.birthA = time.Date(time.Now().Year()-4, time.Now().Month(), time.Now().Day(), 12, 0, 0, 0, time.UTC)
	mkStudent := func(id, classID uint64, name string, birthday, enroll time.Time, status int8) {
		require.NoError(t, db.WithContext(ctx).Create(&domain.Student{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: id}},
			KindergartenID:  testPCPKG, ClassID: classID, Name: name,
			Birthday: birthday, EnrollDate: enroll, Status: status,
		}).Error)
	}
	mkStudent(995100, testPCPClassA, "测试甲", f.birthA,
		time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC), 1)
	// 非在读与软删都不计入 student_count（既有口径，回归守卫）。
	mkStudent(995101, testPCPClassA, "测试乙", f.birthA,
		time.Date(2024, 9, 1, 0, 0, 0, 0, time.UTC), 2)
	require.NoError(t, db.WithContext(ctx).Create(&domain.Student{
		SoftDeleteModel: domain.SoftDeleteModel{
			BaseModel: domain.BaseModel{
				ID:        995102,
				DeletedAt: gorm.DeletedAt{Time: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), Valid: true},
			},
		},
		KindergartenID: testPCPKG, ClassID: testPCPClassA, Name: "测试丙",
		Birthday: f.birthA, EnrollDate: time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC), Status: 1,
	}).Error)
	mkStudent(995110, testPCPClassB, "测试丁",
		time.Date(2020, 3, 5, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 9, 1, 0, 0, 0, 0, time.UTC), 1)

	f.svc = NewParentPortalService(db, repository.NewParentRepo(db), repository.NewUserRoleRepo(db),
		repository.NewStudentRepo(db), repository.NewClassRepo(db), repository.NewPickupRepo(db),
		repository.NewCalendarRepo(db), repository.NewNoticeRepo(db))

	// A/B 从库里读回：顺带守住 DATE 列经驱动回来的日期不漂移。
	loadStudent := func(id uint64) *domain.Student {
		var st domain.Student
		require.NoError(t, db.WithContext(ctx).First(&st, id).Error)
		return &st
	}
	mkPC := func(class *domain.Class, st *domain.Student) *ParentPortalContext {
		return &ParentPortalContext{
			UserID: testPCPUser, KindergartenID: testPCPKG,
			CurrentStudent: st, Students: []domain.Student{*st}, Class: class,
		}
	}
	f.pcA = mkPC(f.classA, loadStudent(995100))
	f.pcB = mkPC(f.classB, loadStudent(995110))
	// 空班且生日/入园日期为零值：MySQL 严格模式拒写 0000-00-00，
	// 这里直接在内存里构造零值学生，验证 0001-01-01 不会泄漏到家长端。
	stuC := &domain.Student{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: 995120}},
		KindergartenID:  testPCPKG, ClassID: testPCPClassC, Name: "测试戊", Status: 1,
	}
	f.pcC = mkPC(f.classC, stuC)

	t.Cleanup(func() { cleanupPCPData(t, db) })
	return f
}

func pcpTeachersOf(t *testing.T, result map[string]any) []map[string]any {
	t.Helper()
	raw, ok := result["teachers"].([]map[string]any)
	require.True(t, ok, "teachers 应为数组")
	return raw
}

// TestParentClassInfo_TeacherLabelsOrderAndMasking 覆盖三位以上老师的标签映射、
// 排序（主班→副班→生活→其余，同职务按 id）与手机号脱敏。
func TestParentClassInfo_TeacherLabelsOrderAndMasking(t *testing.T) {
	f := newPCPFixture(t)
	ctx := context.Background()

	result, err := f.svc.ClassInfo(ctx, f.pcA)
	require.NoError(t, err)

	assert.Equal(t, "测试小一班", result["name"])
	assert.Equal(t, "小班", result["grade"])
	assert.Equal(t, "2025-2026", result["academic_year"])
	assert.Equal(t, int64(1), result["student_count"], "非在读与软删学生不计入")

	teachers := pcpTeachersOf(t, result)
	require.Len(t, teachers, 5)

	want := []struct {
		id       uint64
		nickname string
		title    string
		label    string
		phone    string
		avatar   string
	}{
		{testPCPHead, "测试主班老师", "head", "主班老师", "138****1001", "https://cdn.example.com/avatar/head.jpg"},
		{testPCPAssist, "测试副班老师", "assistant", "副班老师", "139****1002", "https://cdn.example.com/avatar/assist.jpg"},
		{testPCPNoPhone, "测试无手机老师", "assistant", "副班老师", "", ""},
		{testPCPNursery, "测试生活老师", "nursery", "生活老师", "137****1003", ""},
		{testPCPUnknown, "测试未知职务老师", "volunteer", "volunteer", "136****1004", ""},
	}
	for i, w := range want {
		got := teachers[i]
		assert.Equal(t, w.id, got["id"], "第 %d 位排序", i)
		assert.Equal(t, w.nickname, got["nickname"], "id=%d", w.id)
		assert.Equal(t, w.title, got["title"], "id=%d", w.id)
		assert.Equal(t, w.label, got["title_label"], "id=%d", w.id)
		assert.Equal(t, w.phone, got["phone"], "id=%d", w.id)
		assert.Equal(t, w.avatar, got["avatar"], "id=%d", w.id)
	}

	// 线上形状（08-class.md）：字段齐全、脱敏串直接下发。
	raw, err := json.Marshal(result)
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"title_label":"主班老师"`)
	assert.Contains(t, string(raw), `"phone":"138****1001"`)
	assert.NotContains(t, string(raw), "13800001001", "明文手机号不得下发")
}

// TestParentClassInfo_EmptyTeachersIsArray 空班返回 []，不是 null，也不报错。
func TestParentClassInfo_EmptyTeachersIsArray(t *testing.T) {
	f := newPCPFixture(t)
	ctx := context.Background()

	result, err := f.svc.ClassInfo(ctx, f.pcC)
	require.NoError(t, err)

	teachers := pcpTeachersOf(t, result)
	assert.Empty(t, teachers)

	raw, err := json.Marshal(result)
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"teachers":[]`, "空班应为空数组而非 null")
}

// TestParentStudentProfile_DateOnlyFields 生日/入园日期只下发 YYYY-MM-DD；零值空串。
func TestParentStudentProfile_DateOnlyFields(t *testing.T) {
	f := newPCPFixture(t)
	ctx := context.Background()

	profile, err := f.svc.GetStudentProfile(ctx, f.pcA)
	require.NoError(t, err)
	assert.Equal(t, f.birthA.Format("2006-01-02"), profile["birthday"])
	assert.Equal(t, "2025-09-01", profile["enroll_date"])
	assert.Equal(t, "4岁", profile["age"], "age 口径保持不变")
	assert.Equal(t, "测试小一班", profile["class_name"])

	raw, err := json.Marshal(profile)
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"birthday":"`+f.birthA.Format("2006-01-02")+`"`)
	assert.Contains(t, string(raw), `"enroll_date":"2025-09-01"`)
	assert.NotContains(t, string(raw), "T00:00:00", "不得再下发 RFC3339")

	// 零值（未填生日/入园日期）返回空串，而不是 0001-01-01。
	zero, err := f.svc.GetStudentProfile(ctx, f.pcC)
	require.NoError(t, err)
	assert.Equal(t, "", zero["birthday"])
	assert.Equal(t, "", zero["enroll_date"])
	assert.Equal(t, "", zero["age"])
	rawZero, err := json.Marshal(zero)
	require.NoError(t, err)
	assert.Contains(t, string(rawZero), `"birthday":""`)
	assert.NotContains(t, string(rawZero), "0001-01-01")
}

// TestParentClassProfile_ClassIsolation 换班不串数据：老师只属于本班，档案跟随当前孩子。
func TestParentClassProfile_ClassIsolation(t *testing.T) {
	f := newPCPFixture(t)
	ctx := context.Background()

	resultB, err := f.svc.ClassInfo(ctx, f.pcB)
	require.NoError(t, err)
	teachersB := pcpTeachersOf(t, resultB)
	require.Len(t, teachersB, 1)
	assert.Equal(t, testPCPClassBT, teachersB[0]["id"], "只返回本班老师")
	assert.Equal(t, "测试小二班", resultB["name"])

	resultA, err := f.svc.ClassInfo(ctx, f.pcA)
	require.NoError(t, err)
	for _, tea := range pcpTeachersOf(t, resultA) {
		assert.NotEqual(t, testPCPClassBT, tea["id"], "A 班不得混入 B 班老师")
	}

	profileB, err := f.svc.GetStudentProfile(ctx, f.pcB)
	require.NoError(t, err)
	assert.Equal(t, uint64(995110), profileB["id"])
	assert.Equal(t, "测试丁", profileB["name"])
	assert.Equal(t, "测试小二班", profileB["class_name"])
	assert.Equal(t, "2020-03-05", profileB["birthday"])
	assert.Equal(t, "2024-09-01", profileB["enroll_date"])
}

// TestParentClassProfile_CleanupLeavesNoResidue 夹具清理后 9950xx 区间 0 残留。
// 用 Table 而非 Model 计数，软删行也算——这里要证明的是硬删。
func TestParentClassProfile_CleanupLeavesNoResidue(t *testing.T) {
	f := newPCPFixture(t)
	cleanupPCPData(t, f.db)

	checks := []struct {
		table string
		where string
		args  []any
	}{
		{"students", "id BETWEEN 995100 AND 995199", nil},
		{"student_parents", "student_id BETWEEN 995100 AND 995199", nil},
		{"class_teachers", "class_id BETWEEN 995010 AND 995019", nil},
		{"classes", "id BETWEEN 995010 AND 995019", nil},
		{"users", "id BETWEEN 995000 AND 995099", nil},
		{"kindergartens", "id = ?", []any{testPCPKG}},
	}
	for _, c := range checks {
		var n int64
		require.NoError(t, f.db.Table(c.table).Where(c.where, c.args...).Count(&n).Error, c.table)
		assert.Zero(t, n, "%s 残留", c.table)
	}
}
