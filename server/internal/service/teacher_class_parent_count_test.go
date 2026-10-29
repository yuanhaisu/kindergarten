package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

// B02：GET /teacher/class/list 的 parent_count（班级潜在触达家长数）口径。
// 打真实 MySQL（本地 configs/app.ini），固定 9940xx 夹具，t.Cleanup 硬删 0 残留。
//
// 覆盖 TC-NEW-CLS-0001–0005：多班批量、同家长去重、空班为 0、
// 非在读/软删排除、班级与租户隔离；另有一条查询次数守卫防 N+1。

const (
	testCLPKG         uint64 = 994001 // 老师所在园所
	testCLPKG2        uint64 = 994002 // 其它租户：同号班级下挂学生，验证不被混入
	testCLPUser       uint64 = 994001 // 老师
	testCLPClassA     uint64 = 994010 // 小一班：2 学生 3 家长
	testCLPClassB     uint64 = 994011 // 小二班：1 学生 1 家长
	testCLPClassEmpty uint64 = 994012 // 空班
	testCLPClassOther uint64 = 994013 // 其它租户的班级（不该出现在本园老师的列表里）
)

var clpTestDB *gorm.DB

func openCLPTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	if clpTestDB != nil {
		return clpTestDB
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
	clpTestDB = db
	return db
}

// clpSQLLogger 记录每次执行的 SQL，用来证明列表级只有一条聚合查询。
type clpSQLLogger struct {
	gormlogger.Interface
	mu   sync.Mutex
	sqls []string
}

func (l *clpSQLLogger) Trace(_ context.Context, _ time.Time, fc func() (string, int64), _ error) {
	sql, _ := fc()
	l.mu.Lock()
	l.sqls = append(l.sqls, sql)
	l.mu.Unlock()
}

func (l *clpSQLLogger) countContaining(sub string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	n := 0
	for _, s := range l.sqls {
		if strings.Contains(s, sub) {
			n++
		}
	}
	return n
}

func cleanupCLPData(t *testing.T, db *gorm.DB) {
	t.Helper()
	ctx := context.Background()
	del := func(sql string, args ...any) {
		if err := db.WithContext(ctx).Exec(sql, args...).Error; err != nil {
			t.Logf("cleanup: %v", err)
		}
	}
	// 子表先删：student_parents 引用 students/users，class_teachers 引用 classes/users。
	del("DELETE FROM student_parents WHERE student_id BETWEEN 994100 AND 994199 OR user_id BETWEEN 994110 AND 994199")
	del("DELETE FROM students WHERE id BETWEEN 994100 AND 994199")
	del("DELETE FROM class_teachers WHERE user_id BETWEEN 994000 AND 994099 OR class_id BETWEEN 994010 AND 994019")
	del("DELETE FROM classes WHERE id BETWEEN 994010 AND 994019")
	del("DELETE FROM users WHERE id BETWEEN 994000 AND 994199")
	del("DELETE FROM kindergartens WHERE id IN (?, ?)", testCLPKG, testCLPKG2)
}

// clpFixture 建两个园所、四个班、七名学生、八位家长与一条软删绑定。
// A 班期望 3（p1 两个孩子同班只计 1），B 班 1，空班 0。
type clpFixture struct {
	db  *gorm.DB
	svc *TeacherPortalService
	tc  *TeacherPortalContext
}

func newCLPFixture(t *testing.T) *clpFixture {
	t.Helper()
	db := openCLPTestDB(t)
	ctx := context.Background()
	cleanupCLPData(t, db)

	validDate := time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC)
	softDeletedAt := gorm.DeletedAt{Time: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), Valid: true}

	require.NoError(t, db.WithContext(ctx).Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: testCLPKG}, Name: "测试园所-班级家长数",
	}).Error)
	require.NoError(t, db.WithContext(ctx).Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: testCLPKG2}, Name: "测试园所-其它租户",
	}).Error)

	require.NoError(t, db.WithContext(ctx).Create(&domain.User{
		BaseModel: domain.BaseModel{ID: testCLPUser}, Phone: "13900004001", Nickname: "测试班级老师",
	}).Error)
	// 家长用户：994110–994118（994118 只属于其它租户）
	for i, nick := range []string{"家长一", "家长二", "家长三", "待入学家长", "毕业家长", "软删学生家长", "软删绑定家长", "乙班家长", "他园家长"} {
		require.NoError(t, db.WithContext(ctx).Create(&domain.User{
			BaseModel: domain.BaseModel{ID: uint64(994110 + i)}, Phone: phoneForCLP(i), Nickname: nick,
		}).Error)
	}

	for _, cl := range []struct {
		id   uint64
		name string
		kg   uint64
	}{
		{testCLPClassA, "测试小一班", testCLPKG},
		{testCLPClassB, "测试小二班", testCLPKG},
		{testCLPClassEmpty, "测试空班", testCLPKG},
		{testCLPClassOther, "他园班级", testCLPKG2},
	} {
		require.NoError(t, db.WithContext(ctx).Create(&domain.Class{
			BaseModel: domain.BaseModel{ID: cl.id}, KindergartenID: cl.kg, Name: cl.name, Status: 1,
		}).Error)
	}

	// 老师任教 A/B/空班；另在他园挂一个班，验证按园所隔离。
	for _, classID := range []uint64{testCLPClassA, testCLPClassB, testCLPClassEmpty, testCLPClassOther} {
		require.NoError(t, db.WithContext(ctx).Create(&domain.ClassTeacher{
			ClassID: classID, UserID: testCLPUser, Title: domain.TeacherTitleHead, Status: 1,
		}).Error)
	}

	mkStudent := func(id, classID, kg uint64, status int8, deleted gorm.DeletedAt) {
		require.NoError(t, db.WithContext(ctx).Create(&domain.Student{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: id, DeletedAt: deleted}},
			KindergartenID:  kg, ClassID: classID, Name: "测试学生",
			Birthday: validDate, EnrollDate: validDate, Status: status,
		}).Error)
		// Student.Status 带 `default:1`，GORM 插入时会跳过零值，必须显式回写。
		if status != 1 {
			require.NoError(t, db.WithContext(ctx).Model(&domain.Student{}).
				Where("id = ?", id).Update("status", status).Error)
		}
	}
	mkStudent(994100, testCLPClassA, testCLPKG, 1, gorm.DeletedAt{})
	mkStudent(994101, testCLPClassA, testCLPKG, 1, gorm.DeletedAt{})
	mkStudent(994102, testCLPClassA, testCLPKG, 0, gorm.DeletedAt{})  // 待入学：不计
	mkStudent(994103, testCLPClassA, testCLPKG, 2, gorm.DeletedAt{})  // 毕业：不计
	mkStudent(994104, testCLPClassA, testCLPKG, 1, softDeletedAt)     // 学生已软删：不计
	mkStudent(994105, testCLPClassB, testCLPKG, 1, gorm.DeletedAt{})  // B 班唯一在读学生
	mkStudent(994106, testCLPClassA, testCLPKG2, 1, gorm.DeletedAt{}) // 他租户学生，class_id 撞 A 班

	mkParent := func(studentID, userID uint64, relation domain.ParentRelation, deleted bool) {
		sp := &domain.StudentParent{StudentID: studentID, UserID: userID, Relation: relation}
		if deleted {
			sp.DeletedAt = softDeletedAt
		}
		require.NoError(t, db.WithContext(ctx).Create(sp).Error)
	}
	mkParent(994100, 994110, domain.RelationFather, false) // 家长一：两个孩子同班（994100/994101）
	mkParent(994100, 994111, domain.RelationMother, false)
	mkParent(994101, 994110, domain.RelationFather, false) // 重复绑定同一家长
	mkParent(994101, 994112, domain.RelationMother, false)
	mkParent(994102, 994113, domain.RelationFather, false) // 待入学学生家长：不计
	mkParent(994103, 994114, domain.RelationFather, false) // 毕业学生家长：不计
	mkParent(994104, 994115, domain.RelationFather, false) // 软删学生家长：不计
	mkParent(994101, 994116, domain.RelationOther, true)   // 绑定已软删：不计
	mkParent(994105, 994117, domain.RelationFather, false) // B 班家长
	mkParent(994106, 994118, domain.RelationFather, false) // 他租户家长：不计

	svc := NewTeacherPortalService(db, nil, repository.NewPickupRepo(db), repository.NewMessageRepo(db), repository.NewNoticeRepo(db), nil, config.FaceConfig{})
	f := &clpFixture{
		db:  db,
		svc: svc,
		tc: &TeacherPortalContext{
			UserID:         testCLPUser,
			KindergartenID: testCLPKG,
			ClassID:        testCLPClassA,
		},
	}
	t.Cleanup(func() { cleanupCLPData(t, db) })
	return f
}

func phoneForCLP(i int) string {
	return "139000040" + []string{"10", "11", "12", "13", "14", "15", "16", "17", "18"}[i]
}

func (f *clpFixture) byClassID(t *testing.T) map[uint64]map[string]any {
	t.Helper()
	rows, err := f.svc.ListTeacherClasses(ctx(), f.tc)
	require.NoError(t, err)
	out := make(map[uint64]map[string]any, len(rows))
	for _, r := range rows {
		id, ok := r["class_id"].(uint64)
		require.True(t, ok, "class_id 应为 uint64")
		out[id] = r
	}
	return out
}

// TC-NEW-CLS-0001：多班一次读取，每班人数各自正确。
func TestListTeacherClasses_ParentCountByClass(t *testing.T) {
	f := newCLPFixture(t)
	got := f.byClassID(t)

	require.Len(t, got, 3, "只返回本园任教的三个班，他园班级不混入")
	assert.EqualValues(t, 3, got[testCLPClassA]["parent_count"])
	assert.EqualValues(t, 1, got[testCLPClassB]["parent_count"])
	assert.EqualValues(t, 0, got[testCLPClassEmpty]["parent_count"])

	// 既有字段不受影响
	assert.Equal(t, "测试小一班", got[testCLPClassA]["class_name"])
	assert.Equal(t, string(domain.TeacherTitleHead), got[testCLPClassA]["title"])
	assert.Equal(t, "主班老师", got[testCLPClassA]["title_label"])
}

// TC-NEW-CLS-0002：同一家长两个孩子同班只计 1。
func TestListTeacherClasses_DedupeSameParent(t *testing.T) {
	f := newCLPFixture(t)
	got := f.byClassID(t)

	// 家长一在 A 班有两条有效绑定（994100/994101），应只计 1；
	// 若去掉 DISTINCT，A 班会变成 4。
	var rawBindings int64
	require.NoError(t, f.db.Model(&domain.StudentParent{}).
		Where("user_id = ? AND deleted_at IS NULL", 994110).Count(&rawBindings).Error)
	require.EqualValues(t, 2, rawBindings, "夹具前提：同一位家长在本班挂了两个孩子")
	assert.EqualValues(t, 3, got[testCLPClassA]["parent_count"])
}

// TC-NEW-CLS-0003：没有在读学生/没有家长的班级返回 0，而不是 null 或报错。
func TestListTeacherClasses_EmptyClassReturnsZero(t *testing.T) {
	f := newCLPFixture(t)
	got := f.byClassID(t)

	empty, ok := got[testCLPClassEmpty]
	require.True(t, ok)
	count, present := empty["parent_count"]
	assert.True(t, present, "字段必须存在")
	assert.NotNil(t, count, "空班级必须是 0，不是 null")
	assert.EqualValues(t, 0, count)
}

// TC-NEW-CLS-0004：非在读学生（待入学/毕业）、软删学生、软删绑定都不计入。
func TestListTeacherClasses_ExcludesNotEnrolledAndDeleted(t *testing.T) {
	f := newCLPFixture(t)
	got := f.byClassID(t)

	// 只有 994110/994111/994112 属于有效范围；994113（待入学）、994114（毕业）、
	// 994115（学生软删）、994116（绑定软删）都不得出现。
	assert.EqualValues(t, 3, got[testCLPClassA]["parent_count"])
}

// TC-NEW-CLS-0005：班级之间与园所之间不串数。
func TestListTeacherClasses_TenantAndClassIsolation(t *testing.T) {
	f := newCLPFixture(t)
	got := f.byClassID(t)

	// 他租户学生在 class_id 上撞 A 班；按 kindergarten_id 过滤后不得计进 A。
	assert.EqualValues(t, 3, got[testCLPClassA]["parent_count"], "他租户学生/家长不得混入本园班级人数")
	// A 班家长不串到 B 班。
	assert.EqualValues(t, 1, got[testCLPClassB]["parent_count"])
	// 他租户的班级不因同一老师挂职而出现在本园列表里。
	_, leaked := got[testCLPClassOther]
	assert.False(t, leaked, "跨园所的任教班级不得出现在本园列表")
}

// 响应形状：每项只多这一个字段，0 序列化成数字 0（不是 null）。
func TestListTeacherClasses_JSONShape(t *testing.T) {
	f := newCLPFixture(t)
	rows, err := f.svc.ListTeacherClasses(ctx(), f.tc)
	require.NoError(t, err)

	raw, err := json.Marshal(rows)
	require.NoError(t, err)
	var decoded []map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &decoded))
	require.Len(t, decoded, 3)

	for _, item := range decoded {
		assert.Len(t, item, 5, "只新增 parent_count，不得夹带其它字段")
		for _, key := range []string{"class_id", "class_name", "title", "title_label", "parent_count"} {
			assert.Contains(t, item, key)
		}
		var classID uint64
		require.NoError(t, json.Unmarshal(item["class_id"], &classID))
		if classID == testCLPClassEmpty {
			assert.Equal(t, "0", string(item["parent_count"]), "空班级必须是数字 0，不是 null")
		}
	}
}

// 性能守卫：列表级只发一条 GROUP BY，不能逐班 COUNT（N+1）。
func TestListTeacherClasses_SingleAggregateQuery(t *testing.T) {
	f := newCLPFixture(t)
	counter := &clpSQLLogger{Interface: gormlogger.Default.LogMode(gormlogger.Silent)}
	svc := NewTeacherPortalService(
		f.db.Session(&gorm.Session{Logger: counter}),
		nil, repository.NewPickupRepo(f.db), repository.NewMessageRepo(f.db), repository.NewNoticeRepo(f.db), nil, config.FaceConfig{},
	)
	rows, err := svc.ListTeacherClasses(ctx(), f.tc)
	require.NoError(t, err)
	require.Len(t, rows, 3)

	assert.Equal(t, 1, counter.countContaining("GROUP BY"), "3 个班级也只能有一条聚合查询")
	assert.Equal(t, 1, counter.countContaining("student_parents"), "禁止逐班查询家长绑定")
}
