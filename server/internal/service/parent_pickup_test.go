package service

// B07 接送多孩隔离 + 关系枚举 9 值回归。
//
// 修复前 ListPickupRecords / ListAuthorizedPersons 用 pc.Students（家庭全部孩子）
// 取数：家长切到孩子 A，仍会看到孩子 B 的接送记录与授权接送人。同理关系枚举
// 只有 5 个英文 code，且创建/编辑接口对 relation 只校验非空，任意字符串都会落库。
// 这里用「两班级两孩子」夹具把两条不变式钉死。

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

// 9904xx 段：避开 9900（接送）、9902（首页 feed）、9903（日历/评论）、9911（晨检）等既有夹具。
const (
	testB07KG       uint64 = 990401
	testB07ClassA   uint64 = 990402
	testB07ClassB   uint64 = 990403
	testB07StudentA uint64 = 990404
	testB07StudentB uint64 = 990405
	testB07Parent   uint64 = 990406
	testB07Teacher  uint64 = 990407
	testB07RecordA  uint64 = 990408
	testB07RecordB  uint64 = 990409
	testB07AuthA    uint64 = 990410
	testB07AuthB    uint64 = 990411
)

func openB07TestDB(t *testing.T) *gorm.DB {
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

// cleanB07Fixture 按 ID 硬删（Unscoped）。先查出行 ID 打日志，作为清理证据；
// 软删行会占住唯一索引，必须 Unscoped，否则重跑会撞键。
func cleanB07Fixture(t *testing.T, db *gorm.DB) {
	t.Helper()
	studentIDs := []uint64{testB07StudentA, testB07StudentB}

	var authIDs []uint64
	require.NoError(t, db.Unscoped().Model(&domain.PickupAuthorization{}).
		Where("student_id IN ?", studentIDs).Pluck("id", &authIDs).Error)
	var recordIDs []uint64
	require.NoError(t, db.Unscoped().Model(&domain.PickupRecord{}).
		Where("student_id IN ?", studentIDs).Pluck("id", &recordIDs).Error)

	del := func(model any, query string, args ...any) {
		if err := db.Unscoped().Where(query, args...).Delete(model).Error; err != nil {
			t.Logf("cleanup: %v", err)
		}
	}
	del(&domain.PickupAuthorization{}, "student_id IN ?", studentIDs)
	del(&domain.PickupRecord{}, "student_id IN ?", studentIDs)
	del(&domain.StudentParent{}, "student_id IN ?", studentIDs)
	del(&domain.UserRole{}, "user_id = ?", testB07Parent)
	del(&domain.Student{}, "id IN ?", studentIDs)
	del(&domain.Class{}, "id IN ?", []uint64{testB07ClassA, testB07ClassB})
	del(&domain.User{}, "id IN ?", []uint64{testB07Parent, testB07Teacher})
	del(&domain.Kindergarten{}, "id = ?", testB07KG)

	t.Logf("B07 cleanup: pickup_authorizations ids=%v pickup_records ids=%v（按 ID Unscoped 硬删）",
		authIDs, recordIDs)
}

func setupB07Fixture(t *testing.T) (*gorm.DB, *ParentPortalService) {
	t.Helper()
	db := openB07TestDB(t)
	cleanB07Fixture(t, db)
	t.Cleanup(func() { cleanB07Fixture(t, db) })

	validDate := time.Date(2021, 6, 1, 0, 0, 0, 0, time.Local)
	require.NoError(t, db.Create(&domain.User{
		BaseModel: domain.BaseModel{ID: testB07Parent}, Nickname: "B07回归家长", PasswordHash: "x",
	}).Error)
	require.NoError(t, db.Create(&domain.User{
		BaseModel: domain.BaseModel{ID: testB07Teacher}, Nickname: "B07回归老师", PasswordHash: "x",
	}).Error)
	require.NoError(t, db.Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: testB07KG}, Name: "测试园所-B07接送",
	}).Error)
	for _, c := range []domain.Class{
		{BaseModel: domain.BaseModel{ID: testB07ClassA}, KindergartenID: testB07KG, Name: "B07A班", Grade: "小班", Status: 1},
		{BaseModel: domain.BaseModel{ID: testB07ClassB}, KindergartenID: testB07KG, Name: "B07B班", Grade: "小班", Status: 1},
	} {
		require.NoError(t, db.Create(&c).Error)
	}

	// MySQL 严格模式拒绝 time.Time 零值，生日/入园日必须给真值。
	for _, st := range []domain.Student{
		{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testB07StudentA}},
			KindergartenID: testB07KG, ClassID: testB07ClassA, Name: "B07学生A", Status: 1,
			Birthday: validDate, EnrollDate: validDate},
		{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testB07StudentB}},
			KindergartenID: testB07KG, ClassID: testB07ClassB, Name: "B07学生B", Status: 1,
			Birthday: validDate, EnrollDate: validDate},
	} {
		require.NoError(t, db.Create(&st).Error)
	}
	for _, sp := range []domain.StudentParent{
		{StudentID: testB07StudentA, UserID: testB07Parent, Relation: domain.RelationMother, IsPrimary: true},
		{StudentID: testB07StudentB, UserID: testB07Parent, Relation: domain.RelationFather},
	} {
		require.NoError(t, db.Create(&sp).Error)
	}
	kgID := testB07KG
	require.NoError(t, db.Create(&domain.UserRole{
		UserID: testB07Parent, Role: "parent", KindergartenID: &kgID,
	}).Error)

	day := time.Date(2026, 10, 28, 0, 0, 0, 0, time.Local)
	for _, r := range []domain.PickupRecord{
		{BaseModel: domain.BaseModel{ID: testB07RecordA}, KindergartenID: testB07KG, ClassID: testB07ClassA,
			StudentID: testB07StudentA, Date: day, Type: domain.PickupTypeDropOff, Method: domain.PickupMethodManual,
			PickupName: "妈妈（家长A）", PickupRelation: "mother", OperatorID: testB07Teacher,
			RecordedAt: day.Add(8 * time.Hour)},
		{BaseModel: domain.BaseModel{ID: testB07RecordB}, KindergartenID: testB07KG, ClassID: testB07ClassB,
			StudentID: testB07StudentB, Date: day, Type: domain.PickupTypeDropOff, Method: domain.PickupMethodManual,
			PickupName: "爸爸（家长B）", PickupRelation: "father", OperatorID: testB07Teacher,
			RecordedAt: day.Add(8*time.Hour + time.Minute)},
	} {
		require.NoError(t, db.Create(&r).Error)
	}
	for _, a := range []domain.PickupAuthorization{
		{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testB07AuthA}},
			StudentID: testB07StudentA, Name: "A的奶奶", Phone: "13800000001", Relation: "grandmother",
			Status: 1, ReviewStatus: 1, AuthType: "permanent", AllowPickup: true, NotifyOnPickup: true},
		{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testB07AuthB}},
			StudentID: testB07StudentB, Name: "B的外公", Phone: "13800000002", Relation: "grandfather",
			Status: 1, ReviewStatus: 1, AuthType: "permanent", AllowPickup: true, NotifyOnPickup: true},
	} {
		require.NoError(t, db.Create(&a).Error)
	}

	svc := NewParentPortalService(db,
		repository.NewParentRepo(db), repository.NewUserRoleRepo(db),
		repository.NewStudentRepo(db), repository.NewClassRepo(db),
		repository.NewPickupRepo(db), repository.NewCalendarRepo(db),
		repository.NewNoticeRepo(db))
	return db, svc
}

func b07ResolveContext(t *testing.T, svc *ParentPortalService, studentID *uint64) *ParentPortalContext {
	t.Helper()
	pc, err := svc.ResolveContext(context.Background(), testB07Parent, testB07KG, studentID)
	require.NoError(t, err)
	require.NotNil(t, pc.CurrentStudent)
	return pc
}

type b07PickupList struct {
	List []struct {
		ID             uint64 `json:"id"`
		StudentName    string `json:"student_name"`
		PickupRelation string `json:"pickup_relation"`
	} `json:"list"`
	Total int64 `json:"total"`
}

// ListPickupRecords 返回的 list 是函数内局部结构体，测试侧只能经 JSON 解码。
func b07DecodePickupList(t *testing.T, resp map[string]any) b07PickupList {
	t.Helper()
	raw, err := json.Marshal(resp)
	require.NoError(t, err)
	var out b07PickupList
	require.NoError(t, json.Unmarshal(raw, &out))
	return out
}

// TC-B07-001 接送记录按当前孩子隔离，A/B 互不串数据；缺省 student_id 用当前孩子而非全部孩子。
func TestParentPickupRecords_ScopedToCurrentStudent(t *testing.T) {
	_, svc := setupB07Fixture(t)
	studentA, studentB := testB07StudentA, testB07StudentB

	pcA := b07ResolveContext(t, svc, &studentA)
	respA, err := svc.ListPickupRecords(context.Background(), pcA, 1, 20)
	require.NoError(t, err)
	listA := b07DecodePickupList(t, respA)
	require.Len(t, listA.List, 1, "孩子 A 只能看到自己的 1 条记录")
	assert.Equal(t, testB07RecordA, listA.List[0].ID)
	assert.Equal(t, int64(1), listA.Total)

	pcB := b07ResolveContext(t, svc, &studentB)
	respB, err := svc.ListPickupRecords(context.Background(), pcB, 1, 20)
	require.NoError(t, err)
	listB := b07DecodePickupList(t, respB)
	require.Len(t, listB.List, 1, "孩子 B 只能看到自己的 1 条记录")
	assert.Equal(t, testB07RecordB, listB.List[0].ID)
	assert.NotEqual(t, testB07RecordA, listB.List[0].ID, "B 的列表不得混入 A 的记录")

	// 无 student_id：回落到默认孩子（按 student_id 升序的第一个），不得返回全部孩子。
	pcDefault := b07ResolveContext(t, svc, nil)
	assert.Equal(t, testB07StudentA, pcDefault.CurrentStudent.ID)
	respDefault, err := svc.ListPickupRecords(context.Background(), pcDefault, 1, 20)
	require.NoError(t, err)
	listDefault := b07DecodePickupList(t, respDefault)
	require.Len(t, listDefault.List, 1, "未显式指定 student_id 时不得返回全部孩子的记录")
	assert.Equal(t, testB07RecordA, listDefault.List[0].ID)
}

// TC-B07-002 授权接送人按当前孩子隔离；缺省 student_id 用当前孩子。
func TestParentAuthorizedPersons_ScopedToCurrentStudent(t *testing.T) {
	_, svc := setupB07Fixture(t)
	studentA, studentB := testB07StudentA, testB07StudentB

	pcA := b07ResolveContext(t, svc, &studentA)
	respA, err := svc.ListAuthorizedPersons(context.Background(), pcA, 1, 20)
	require.NoError(t, err)
	listA, ok := respA["list"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, listA, 1, "孩子 A 只能看到自己的授权人")
	assert.Equal(t, testB07AuthA, listA[0]["id"])
	assert.Equal(t, testB07StudentA, listA[0]["student_id"])

	pcB := b07ResolveContext(t, svc, &studentB)
	respB, err := svc.ListAuthorizedPersons(context.Background(), pcB, 1, 20)
	require.NoError(t, err)
	listB, ok := respB["list"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, listB, 1, "孩子 B 只能看到自己的授权人")
	assert.Equal(t, testB07AuthB, listB[0]["id"])
	assert.Equal(t, testB07StudentB, listB[0]["student_id"])

	pcDefault := b07ResolveContext(t, svc, nil)
	respDefault, err := svc.ListAuthorizedPersons(context.Background(), pcDefault, 1, 20)
	require.NoError(t, err)
	listDefault, ok := respDefault["list"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, listDefault, 1, "未显式指定 student_id 时不得返回全部孩子的授权人")
	assert.Equal(t, testB07AuthA, listDefault[0]["id"])
}

// TC-B07-003 9 个关系 code 均可创建，落库为稳定代码，服务端标签映射覆盖 9 值。
func TestCreateAuthorizedPerson_AcceptsNineRelationCodes(t *testing.T) {
	db, svc := setupB07Fixture(t)
	studentA := testB07StudentA
	pcA := b07ResolveContext(t, svc, &studentA)

	relations := []struct {
		code  string
		label string
	}{
		{"father", "爸爸"},
		{"mother", "妈妈"},
		{"grandfather", "爷爷"},
		{"grandmother", "奶奶"},
		{"maternal_grandfather", "外公"},
		{"maternal_grandmother", "外婆"},
		{"uncle", "叔叔"},
		{"aunt", "阿姨"},
		{"other", "其他"},
	}
	seen := make(map[string]bool, len(relations))
	for _, rel := range relations {
		require.False(t, seen[rel.code], "关系 code 不得重复: %s", rel.code)
		seen[rel.code] = true

		resp, err := svc.CreateAuthorizedPerson(context.Background(), pcA, CreateAuthorizedPersonInput{
			Name: "关系-" + rel.code, Phone: "13800000000", Relation: rel.code,
		})
		require.NoError(t, err, "关系 code %q 必须通过校验", rel.code)
		id, ok := resp["id"].(uint64)
		require.True(t, ok)
		var saved domain.PickupAuthorization
		require.NoError(t, db.First(&saved, id).Error)
		assert.Equal(t, rel.code, saved.Relation, "落库必须是稳定代码")
		assert.Equal(t, rel.label, RelationLabel(domain.ParentRelation(rel.code)),
			"服务端关系标签映射必须覆盖 %q", rel.code)
	}
}

// TC-B07-004 未知关系值被拒绝且不落库。
func TestCreateAuthorizedPerson_RejectsUnknownRelation(t *testing.T) {
	db, svc := setupB07Fixture(t)
	studentA := testB07StudentA
	pcA := b07ResolveContext(t, svc, &studentA)

	var before int64
	require.NoError(t, db.Model(&domain.PickupAuthorization{}).
		Where("student_id = ?", testB07StudentA).Count(&before).Error)

	for _, bad := range []string{"cousin", "舅舅", "fathers", "FATHER"} {
		_, err := svc.CreateAuthorizedPerson(context.Background(), pcA, CreateAuthorizedPersonInput{
			Name: "非法关系", Phone: "13800000000", Relation: bad,
		})
		assert.ErrorIs(t, err, ErrParentInvalidParams, "未知关系值 %q 必须被拒", bad)
	}

	var after int64
	require.NoError(t, db.Model(&domain.PickupAuthorization{}).
		Where("student_id = ?", testB07StudentA).Count(&after).Error)
	assert.Equal(t, before, after, "被拒的关系值不得落库")
}

// TC-B07-005 编辑授权人同样校验关系：新 code 可改，未知值被拒且不改库。
func TestUpdateAuthorizedPerson_RelationValidation(t *testing.T) {
	db, svc := setupB07Fixture(t)
	studentA := testB07StudentA
	pcA := b07ResolveContext(t, svc, &studentA)

	_, err := svc.UpdateAuthorizedPerson(context.Background(), pcA, testB07AuthA,
		UpdateAuthorizedPersonInput{Relation: "maternal_grandfather"})
	require.NoError(t, err, "新增关系 code 必须可编辑")

	var saved domain.PickupAuthorization
	require.NoError(t, db.First(&saved, testB07AuthA).Error)
	assert.Equal(t, "maternal_grandfather", saved.Relation)

	_, err = svc.UpdateAuthorizedPerson(context.Background(), pcA, testB07AuthA,
		UpdateAuthorizedPersonInput{Relation: "cousin"})
	assert.ErrorIs(t, err, ErrParentInvalidParams, "未知关系值必须被拒")

	require.NoError(t, db.First(&saved, testB07AuthA).Error)
	assert.Equal(t, "maternal_grandfather", saved.Relation, "被拒的编辑不得改库")
}

// TC-B07-006 过渡兼容：旧版家长端提交中文标签，写入时归一化为代码；
// 存量中文行不迁移、不改写（历史数据保持原样）。
func TestCreateAuthorizedPerson_LegacyChineseRelationNormalized(t *testing.T) {
	db, svc := setupB07Fixture(t)
	studentA := testB07StudentA
	pcA := b07ResolveContext(t, svc, &studentA)

	labels := map[string]string{
		"爸爸": "father",
		"妈妈": "mother",
		"爷爷": "grandfather",
		"奶奶": "grandmother",
		"外公": "maternal_grandfather",
		"外婆": "maternal_grandmother",
		"叔叔": "uncle",
		"阿姨": "aunt",
		"其他": "other",
	}
	for label, code := range labels {
		resp, err := svc.CreateAuthorizedPerson(context.Background(), pcA, CreateAuthorizedPersonInput{
			Name: "旧页面-" + label, Phone: "13800000000", Relation: label,
		})
		require.NoError(t, err, "旧版中文标签 %q 在过渡期必须被接受", label)
		id, ok := resp["id"].(uint64)
		require.True(t, ok)
		var saved domain.PickupAuthorization
		require.NoError(t, db.First(&saved, id).Error)
		assert.Equal(t, code, saved.Relation, "中文标签 %q 必须归一化为代码落库", label)
	}
}
