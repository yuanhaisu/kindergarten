package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
)

// OS-ALB-3：老师端班级学生详情的家长联系方式（TC-NEW-PPS-0006..0008）。
// 打真实 MySQL（本地 configs/app.ini），固定 9930xx 夹具，t.Cleanup 硬删 0 残留。
//
// 手机号按每个家长各自在当前园所的 show_phone_to_teacher 过滤；
// 关闭时键缺省而不是空串，无设置行按默认开启，无 User 行的绑定直接跳过。

const (
	ttcKG      uint64 = 993001 // 目标园所
	ttcKGOther uint64 = 993002 // 他园：设置行与幼儿都不应串到目标园
	ttcClass   uint64 = 993010 // 目标班
	ttcClassB  uint64 = 993011 // 同园别班
	ttcClassO  uint64 = 993012 // 他园班级

	ttcTeacher uint64 = 993020 // 老师

	ttcStudent        uint64 = 993030 // 目标幼儿：3 位可见家长 + 1 条无 User 绑定
	ttcStudentB       uint64 = 993031 // 同班幼儿：家长不应出现在目标幼儿详情
	ttcStudentNoBind  uint64 = 993033 // 目标班幼儿：没有任何绑定
	ttcStudentOtherKG uint64 = 993034 // 他园幼儿

	ttcParentShow    uint64 = 993040 // show_phone_to_teacher=true
	ttcParentHide    uint64 = 993041 // show_phone_to_teacher=false（他园为 true，验证按园所隔离）
	ttcParentDefault uint64 = 993042 // 目标园无设置行（他园 false，仍应按默认展示）
	ttcParentOther   uint64 = 993043 // 只绑定到 ttcStudentB
	ttcParentGhost   uint64 = 993044 // StudentParent 有绑定但 users 表无此行
	ttcParentNoBind  uint64 = 993045 // 仅用于他园设置行，不绑定任何幼儿
)

var ttcMigrateOnce sync.Once

func cleanupTTC(t *testing.T, db *gorm.DB) {
	t.Helper()
	del := func(sql string, args ...any) {
		if err := db.Exec(sql, args...).Error; err != nil {
			t.Logf("cleanup: %v", err)
		}
	}
	del("DELETE FROM parent_settings WHERE user_id BETWEEN ? AND ?", ttcParentShow, ttcParentNoBind)
	del("DELETE FROM student_parents WHERE student_id BETWEEN ? AND ?", ttcStudent, ttcStudentOtherKG)
	del("DELETE FROM students WHERE id BETWEEN ? AND ?", ttcStudent, ttcStudentOtherKG)
	del("DELETE FROM user_roles WHERE user_id = ?", ttcTeacher)
	del("DELETE FROM users WHERE id BETWEEN ? AND ?", ttcTeacher, ttcParentNoBind)
	del("DELETE FROM classes WHERE id BETWEEN ? AND ?", ttcClass, ttcClassO)
	del("DELETE FROM kindergartens WHERE id IN (?, ?)", ttcKG, ttcKGOther)

	count := func(model any, query string, args ...any) int64 {
		var n int64
		require.NoError(t, db.Unscoped().Model(model).Where(query, args...).Count(&n).Error)
		return n
	}
	require.Zero(t, count(&domain.ParentSetting{}, "user_id BETWEEN ? AND ?", ttcParentShow, ttcParentNoBind))
	require.Zero(t, count(&domain.StudentParent{}, "student_id BETWEEN ? AND ?", ttcStudent, ttcStudentOtherKG))
	require.Zero(t, count(&domain.Student{}, "id BETWEEN ? AND ?", ttcStudent, ttcStudentOtherKG))
	require.Zero(t, count(&domain.User{}, "id BETWEEN ? AND ?", ttcTeacher, ttcParentNoBind))
	require.Zero(t, count(&domain.Class{}, "id BETWEEN ? AND ?", ttcClass, ttcClassO))
	require.Zero(t, count(&domain.Kindergarten{}, "id IN (?, ?)", ttcKG, ttcKGOther))
}

func newTTCFixture(t *testing.T) (*TeacherPortalService, *TeacherPortalContext, *gorm.DB) {
	t.Helper()
	db := openCLPTestDB(t)
	ttcMigrateOnce.Do(func() {
		require.NoError(t, db.AutoMigrate(
			&domain.Kindergarten{}, &domain.Class{}, &domain.User{}, &domain.UserRole{},
			&domain.Student{}, &domain.StudentParent{}, &domain.ParentSetting{},
		))
	})
	cleanupTTC(t, db)
	t.Cleanup(func() { cleanupTTC(t, db) })

	for _, kg := range []struct {
		id   uint64
		name string
	}{{ttcKG, "测试园所-家长联系方式"}, {ttcKGOther, "测试园所-他园"}} {
		require.NoError(t, db.Create(&domain.Kindergarten{
			BaseModel: domain.BaseModel{ID: kg.id}, Name: kg.name,
		}).Error)
	}
	for _, cl := range []struct {
		id   uint64
		kg   uint64
		name string
	}{
		{ttcClass, ttcKG, "联系方式一班"},
		{ttcClassB, ttcKG, "联系方式二班"},
		{ttcClassO, ttcKGOther, "他园联系方式班"},
	} {
		require.NoError(t, db.Create(&domain.Class{
			BaseModel: domain.BaseModel{ID: cl.id}, KindergartenID: cl.kg, Name: cl.name, Grade: "小班", Status: 1,
		}).Error)
	}

	users := []domain.User{
		{BaseModel: domain.BaseModel{ID: ttcTeacher}, Nickname: "班主任", Phone: "13800000001", PasswordHash: "x"},
		{BaseModel: domain.BaseModel{ID: ttcParentShow}, Nickname: "家长-可见", Phone: "13800001001", PasswordHash: "x"},
		{BaseModel: domain.BaseModel{ID: ttcParentHide}, Nickname: "家长-隐藏", Phone: "13800001002", PasswordHash: "x"},
		{BaseModel: domain.BaseModel{ID: ttcParentDefault}, Nickname: "家长-默认", Phone: "13800001003", PasswordHash: "x"},
		{BaseModel: domain.BaseModel{ID: ttcParentOther}, Nickname: "家长-别班", Phone: "13800001004", PasswordHash: "x"},
		{BaseModel: domain.BaseModel{ID: ttcParentGhost}, Nickname: "家长-幽灵", Phone: "13800001005", PasswordHash: "x"},
		{BaseModel: domain.BaseModel{ID: ttcParentNoBind}, Nickname: "家长-未绑定", Phone: "13800001006", PasswordHash: "x"},
	}
	for i := range users {
		require.NoError(t, db.Create(&users[i]).Error)
	}

	kgID, classID := ttcKG, ttcClass
	require.NoError(t, db.Create(&domain.UserRole{
		UserID: ttcTeacher, Role: "teacher", KindergartenID: &kgID, ClassID: &classID,
	}).Error)

	for _, st := range []struct {
		id    uint64
		kg    uint64
		class uint64
		name  string
	}{
		{ttcStudent, ttcKG, ttcClass, "幼儿-多家长"},
		{ttcStudentB, ttcKG, ttcClass, "幼儿-同班"},
		{ttcStudentNoBind, ttcKG, ttcClass, "幼儿-无绑定"},
		{ttcStudentOtherKG, ttcKGOther, ttcClassO, "幼儿-他园"},
	} {
		require.NoError(t, db.Create(&domain.Student{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: st.id}},
			KindergartenID:  st.kg, ClassID: st.class, Name: st.name, Status: 1,
			Birthday:   time.Date(2021, 9, 15, 0, 0, 0, 0, time.Local),
			EnrollDate: time.Date(2025, 9, 1, 0, 0, 0, 0, time.Local),
		}).Error)
	}

	for _, sp := range []struct {
		student  uint64
		parent   uint64
		relation domain.ParentRelation
		primary  bool
	}{
		{ttcStudent, ttcParentShow, domain.RelationFather, true},
		{ttcStudent, ttcParentHide, domain.RelationMother, false},
		{ttcStudent, ttcParentDefault, domain.RelationGrandmother, false},
		{ttcStudent, ttcParentGhost, domain.RelationUncle, false},
		{ttcStudentB, ttcParentOther, domain.RelationMother, true},
	} {
		require.NoError(t, db.Create(&domain.StudentParent{
			StudentID: sp.student, UserID: sp.parent, Relation: sp.relation, IsPrimary: sp.primary,
		}).Error)
	}

	// FK 要求 users 行存在，不能直接插入「悬空绑定」；软删用户模拟账号已注销：
	// 普通查询看不到该 User，但 student_parents 行仍在（真实数据缺失/注销后的形态）。
	require.NoError(t, db.Delete(&domain.User{}, ttcParentGhost).Error)

	for _, ps := range []struct {
		user uint64
		kg   uint64
		show bool
	}{
		{ttcParentShow, ttcKG, true},
		{ttcParentHide, ttcKG, false},
		// 他园设置不作用于本园：hide 在他园为 true，default 在他园为 false。
		{ttcParentHide, ttcKGOther, true},
		{ttcParentDefault, ttcKGOther, false},
		{ttcParentOther, ttcKG, true},
		{ttcParentNoBind, ttcKGOther, false},
	} {
		require.NoError(t, db.Create(&domain.ParentSetting{
			UserID: ps.user, KindergartenID: ps.kg,
			NoticePush: true, LeavePush: true, PickupPush: true, GrowthPush: true,
			AnnouncementPush: true, ShowPhoneToTeacher: ps.show,
			AllowClassShare: true, AllowExternalShare: false,
		}).Error)
	}

	svc := NewTeacherPortalService(db, nil, nil, nil, nil, nil, config.FaceConfig{})
	tc := &TeacherPortalContext{UserID: ttcTeacher, KindergartenID: ttcKG, ClassID: ttcClass}
	return svc, tc, db
}

func ttcParentsByName(t *testing.T, data map[string]any) map[string]map[string]any {
	t.Helper()
	raw, ok := data["parents"].([]map[string]any)
	require.True(t, ok, "parents 应为数组，实际 %T", data["parents"])
	out := make(map[string]map[string]any, len(raw))
	for _, p := range raw {
		name, _ := p["name"].(string)
		out[name] = p
	}
	return out
}

func TestTeacherClassStudentParentsPhoneFiltering(t *testing.T) {
	svc, tc, db := newTTCFixture(t)
	ctx := context.Background()

	data, err := svc.ClassStudent(ctx, tc, ttcStudent)
	require.NoError(t, err)
	st, ok := data["student"].(domain.Student)
	require.True(t, ok, "既有 student 字段必须保留")
	assert.Equal(t, ttcStudent, st.ID)

	byName := ttcParentsByName(t, data)
	require.Len(t, byName, 3, "无 User 行的绑定跳过；同班别家幼儿的家长不出现")
	assert.NotContains(t, byName, "家长-别班")
	assert.NotContains(t, byName, "家长-未绑定")

	// TC-NEW-PPS-0007：开启时可见。
	show := byName["家长-可见"]
	assert.Equal(t, "father", show["relation"])
	assert.Equal(t, "爸爸", show["relation_label"])
	assert.Equal(t, maskPhone("13800001001"), show["phone"])

	// TC-NEW-PPS-0006：关闭时字段缺省，不是空串。
	hide := byName["家长-隐藏"]
	assert.Equal(t, "mother", hide["relation"])
	assert.Equal(t, "妈妈", hide["relation_label"])
	_, hasPhone := hide["phone"]
	assert.False(t, hasPhone, "关闭 show_phone_to_teacher 时不得返回 phone 键")

	// TC-NEW-PPS-0008：多家长逐个过滤；无设置行默认展示。
	def := byName["家长-默认"]
	assert.Equal(t, maskPhone("13800001003"), def["phone"], "无设置行按默认允许展示")
	assert.Equal(t, "奶奶", def["relation_label"])

	var rows int64
	require.NoError(t, db.Model(&domain.ParentSetting{}).
		Where("user_id = ? AND kindergarten_id = ?", ttcParentHide, ttcKG).Count(&rows).Error)
	require.EqualValues(t, 1, rows, "夹具应有关闭行")
}

func TestTeacherClassStudentParentsEmptyWhenNoBindings(t *testing.T) {
	svc, tc, _ := newTTCFixture(t)
	ctx := context.Background()

	data, err := svc.ClassStudent(ctx, tc, ttcStudentNoBind)
	require.NoError(t, err)
	raw, ok := data["parents"].([]map[string]any)
	require.True(t, ok, "parents 应为数组")
	assert.NotNil(t, raw, "无绑定应返回空数组而不是 null")
	assert.Empty(t, raw)
}

func TestTeacherClassStudentParentsIsolation(t *testing.T) {
	svc, tc, _ := newTTCFixture(t)
	ctx := context.Background()

	_, err := svc.ClassStudent(ctx, tc, ttcStudentB)
	require.NoError(t, err, "同班学生可读")

	// 跨班 / 跨园读取一律 404，不泄露其他幼儿的家长。
	otherClass := &TeacherPortalContext{UserID: ttcTeacher, KindergartenID: ttcKG, ClassID: ttcClassB}
	_, err = svc.ClassStudent(ctx, otherClass, ttcStudent)
	assert.ErrorIs(t, err, ErrTeacherNotFound, "跨班读取应 404")

	_, err = svc.ClassStudent(ctx, tc, ttcStudentOtherKG)
	assert.ErrorIs(t, err, ErrTeacherNotFound, "跨园读取应 404")
}
