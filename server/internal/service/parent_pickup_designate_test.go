package service

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/domain"
)

// 覆盖家长侧「今日接走人指定」与「凭证绑人」（spec 4+5 场景），
// 以及 ListPickupRecords 的 is_abnormal 修漂移。真 DB 跑，复用 teacher_pickup_create_test.go 的 fixture。

type parentPickupFixture struct {
	pf        *pickupFixture
	svc       *ParentPortalService
	pc        *ParentPortalContext
	studentID uint64
}

func newParentPickupFixture(t *testing.T, deadline string, classID uint64) *parentPickupFixture {
	t.Helper()
	pf := newPickupFixture(t, deadline, classID)
	svc := NewParentPortalService(pf.svc.db, nil, nil, nil, nil, pf.svc.pickup, nil, nil)
	st := &domain.Student{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: pf.studentID}},
		KindergartenID:  testPickupKG,
		ClassID:         classID,
		Name:            "测试学生",
		Avatar:          "https://cdn.example.com/student/1.jpg",
		Status:          1,
	}
	pc := &ParentPortalContext{
		UserID:         880001,
		KindergartenID: testPickupKG,
		CurrentStudent: st,
		Students:       []domain.Student{*st},
	}
	return &parentPickupFixture{pf: pf, svc: svc, pc: pc, studentID: pf.studentID}
}

// ==================== 指定今日接走人（4 场景）====================

// TC-NEW-PKU-001 指定成功
func TestDesignate_Success(t *testing.T) {
	f := newParentPickupFixture(t, "00:00", testPickupClass)
	authID := f.pf.makeAuthorization(t, "奶奶（张三）", 1, 1, nil)

	resp, err := f.svc.DesignatePickupPerson(ctx(), f.pc, DesignatePickupInput{AuthorizationID: authID})
	require.NoError(t, err, "指定有效授权人必须成功")

	assert.Equal(t, authID, resp["authorization_id"])
	person, ok := resp["person"].(map[string]any)
	require.True(t, ok, "应回显接走人")
	assert.Equal(t, "奶奶（张三）", person["name"])

	// 落库：指定记录存在且绑对人
	desig, err := f.pf.svc.pickup.FindDailyDesignation(ctx(), f.studentID, time.Now())
	require.NoError(t, err)
	require.NotNil(t, desig, "指定记录必须落库")
	assert.Equal(t, authID, desig.AuthorizationID)
}

// TC-NEW-PKU-002 指定未审核通过者被拒
func TestDesignate_RejectsUnreviewed(t *testing.T) {
	f := newParentPickupFixture(t, "00:00", testPickupClass)
	// review_status=0 待审核
	authID := f.pf.makeAuthorization(t, "奶奶（张三）", 0, 1, nil)

	_, err := f.svc.DesignatePickupPerson(ctx(), f.pc, DesignatePickupInput{AuthorizationID: authID})
	assert.ErrorIs(t, err, ErrParentForbidden, "未审核通过者必须被拒")

	desig, err := f.pf.svc.pickup.FindDailyDesignation(ctx(), f.studentID, time.Now())
	require.NoError(t, err)
	assert.Nil(t, desig, "被拒后不得产生指定记录")
}

// TC-NEW-PKU-003 指定已撤销或已过期者被拒
func TestDesignate_RejectsRevokedOrExpired(t *testing.T) {
	f := newParentPickupFixture(t, "00:00", testPickupClass)

	// 已撤销 status=0
	revoked := f.pf.makeAuthorization(t, "奶奶（张三）", 1, 0, nil)
	_, err := f.svc.DesignatePickupPerson(ctx(), f.pc, DesignatePickupInput{AuthorizationID: revoked})
	assert.ErrorIs(t, err, ErrParentForbidden, "已撤销必须被拒")

	// 已过期
	past := time.Now().Add(-time.Hour)
	expired := f.pf.makeAuthorization(t, "爷爷（李四）", 1, 1, &past)
	_, err = f.svc.DesignatePickupPerson(ctx(), f.pc, DesignatePickupInput{AuthorizationID: expired})
	assert.ErrorIs(t, err, ErrParentForbidden, "已过期必须被拒")
}

// TC-NEW-PKU-004 重复指定覆盖旧值
func TestDesignate_Overrides(t *testing.T) {
	f := newParentPickupFixture(t, "00:00", testPickupClass)
	a := f.pf.makeAuthorization(t, "奶奶（张三）", 1, 1, nil)
	b := f.pf.makeAuthorization(t, "爷爷（李四）", 1, 1, nil)

	// 先指定 A 并取码
	_, err := f.svc.DesignatePickupPerson(ctx(), f.pc, DesignatePickupInput{AuthorizationID: a})
	require.NoError(t, err)
	qr1, err := f.svc.GetQRCode(ctx(), f.pc)
	require.NoError(t, err)
	require.NotEmpty(t, qr1["qr_code"])

	// 改指定为 B
	_, err = f.svc.DesignatePickupPerson(ctx(), f.pc, DesignatePickupInput{AuthorizationID: b})
	require.NoError(t, err)

	desig, err := f.pf.svc.pickup.FindDailyDesignation(ctx(), f.studentID, time.Now())
	require.NoError(t, err)
	require.NotNil(t, desig)
	assert.Equal(t, b, desig.AuthorizationID, "指定必须被覆盖为后者")

	// 旧凭证必须失效（绑的是 A）
	var oldQR domain.PickupQRCode
	require.NoError(t, f.pf.svc.db.Where("code = ?", qr1["qr_code"]).First(&oldQR).Error)
	assert.False(t, oldQR.IsActive, "换人后旧凭证必须作废")

	// 新码应绑 B
	qr2, err := f.svc.GetQRCode(ctx(), f.pc)
	require.NoError(t, err)
	person := qr2["person"].(map[string]any)
	assert.Equal(t, "爷爷（李四）", person["name"], "新凭证应绑新指定的接走人")
}

// ==================== 凭证（5 场景）====================

// TC-NEW-PKU-010 凭证携带接走人可读信息
func TestQR_CarriesPersonInfo(t *testing.T) {
	f := newParentPickupFixture(t, "00:00", testPickupClass)
	authID := f.pf.makeAuthorization(t, "奶奶（张三）", 1, 1, nil)
	_, err := f.svc.DesignatePickupPerson(ctx(), f.pc, DesignatePickupInput{AuthorizationID: authID})
	require.NoError(t, err)

	qr, err := f.svc.GetQRCode(ctx(), f.pc)
	require.NoError(t, err)

	person, ok := qr["person"].(map[string]any)
	require.True(t, ok, "凭证响应必须含 person")
	assert.Equal(t, "奶奶（张三）", person["name"])
	assert.NotEmpty(t, person["photo"], "person.photo 必须有值，供老师人证对照")

	student, ok := qr["student"].(map[string]any)
	require.True(t, ok)
	assert.NotEmpty(t, student["photo"], "student.photo 必须有值")
}

// TC-NEW-PKU-011 生成时校验接走人资格
func TestQR_GenerationValidatesAuthorization(t *testing.T) {
	f := newParentPickupFixture(t, "00:00", testPickupClass)
	authID := f.pf.makeAuthorization(t, "奶奶（张三）", 1, 1, nil)
	_, err := f.svc.DesignatePickupPerson(ctx(), f.pc, DesignatePickupInput{AuthorizationID: authID})
	require.NoError(t, err)

	// 指定后撤销授权 → 取码必须失败
	require.NoError(t, f.pf.svc.db.Model(&domain.PickupAuthorization{}).
		Where("id = ?", authID).Update("status", 0).Error)

	_, err = f.svc.GetQRCode(ctx(), f.pc)
	assert.ErrorIs(t, err, ErrParentForbidden, "接走人已失效时取码必须被拒")
}

// TC-NEW-PKU-012 凭证不泄露可枚举学生标识
func TestQR_TokenDoesNotLeakStudentID(t *testing.T) {
	f := newParentPickupFixture(t, "00:00", testPickupClass)
	authID := f.pf.makeAuthorization(t, "奶奶（张三）", 1, 1, nil)
	_, err := f.svc.DesignatePickupPerson(ctx(), f.pc, DesignatePickupInput{AuthorizationID: authID})
	require.NoError(t, err)

	qr, err := f.svc.GetQRCode(ctx(), f.pc)
	require.NoError(t, err)
	code, _ := qr["qr_code"].(string)
	require.NotEmpty(t, code)

	assert.False(t, strings.Contains(code, "PKUP"), "不得沿用内嵌 studentID 的旧 token 格式")
	assert.NotContains(t, code, "990001", "凭证串不得含学生 ID")
	assert.NotContains(t, code, "990002", "凭证串不得含班级 ID")
}

// TC-NEW-PKU-013 凭证当日过期
func TestQR_ExpiresEndOfDay(t *testing.T) {
	f := newParentPickupFixture(t, "00:00", testPickupClass)
	authID := f.pf.makeAuthorization(t, "奶奶（张三）", 1, 1, nil)
	_, err := f.svc.DesignatePickupPerson(ctx(), f.pc, DesignatePickupInput{AuthorizationID: authID})
	require.NoError(t, err)

	qr, err := f.svc.GetQRCode(ctx(), f.pc)
	require.NoError(t, err)
	expireStr, _ := qr["expire_at"].(string)
	require.NotEmpty(t, expireStr)

	expire, err := time.Parse(time.RFC3339, expireStr)
	require.NoError(t, err)
	now := time.Now()
	assert.Equal(t, now.Year(), expire.In(now.Location()).Year(), "须当日过期")
	assert.Equal(t, now.YearDay(), expire.In(now.Location()).YearDay(), "须当日过期")
}

// TC-NEW-PKU-014 凭证一次有效
func TestQR_OneTimeUse(t *testing.T) {
	f := newParentPickupFixture(t, "00:00", testPickupClass)
	authID := f.pf.makeAuthorization(t, "奶奶（张三）", 1, 1, nil)
	_, err := f.svc.DesignatePickupPerson(ctx(), f.pc, DesignatePickupInput{AuthorizationID: authID})
	require.NoError(t, err)

	// 先送入，否则接走被「须先送入」拒
	_, err = f.pf.svc.CreatePickupRecord(ctx(), f.pf.tc, TeacherPickupInput{
		StudentID: f.studentID, Type: 1,
	}, domain.PickupMethodManual)
	require.NoError(t, err)

	qr, err := f.svc.GetQRCode(ctx(), f.pc)
	require.NoError(t, err)
	code, _ := qr["qr_code"].(string)

	// 第一次扫码成功
	_, err = f.pf.svc.CreatePickupRecord(ctx(), f.pf.tc, TeacherPickupInput{QRCode: code}, domain.PickupMethodScan)
	require.NoError(t, err)

	// 再次返园 + 第二次扫码同一凭证 → 必须被拒
	_, err = f.pf.svc.CreatePickupRecord(ctx(), f.pf.tc, TeacherPickupInput{
		StudentID: f.studentID, Type: 1,
	}, domain.PickupMethodManual)
	require.NoError(t, err)

	_, err = f.pf.svc.CreatePickupRecord(ctx(), f.pf.tc, TeacherPickupInput{QRCode: code}, domain.PickupMethodScan)
	assert.ErrorIs(t, err, ErrTeacherInvalidQRCode, "凭证一次有效，二次使用必须被拒")
}

// ==================== is_abnormal 修漂移 ====================

// 契约已列 is_abnormal 但实现此前未返回（既存漂移）。
// 覆盖有/无异常两种情形，确认列表响应真的带上该字段。
func TestListPickupRecords_IsAbnormalReturned(t *testing.T) {
	f := newParentPickupFixture(t, "00:00", testPickupClass)

	_, err := f.pf.svc.CreatePickupRecord(ctx(), f.pf.tc, TeacherPickupInput{
		StudentID: f.studentID, Type: 1,
	}, domain.PickupMethodManual)
	require.NoError(t, err)

	// 无异常时：字段存在且为 false
	resp, err := f.svc.ListPickupRecords(ctx(), f.pc, 1, 20)
	require.NoError(t, err)
	s, err := json.Marshal(resp["list"])
	require.NoError(t, err)
	assert.Contains(t, string(s), `"is_abnormal":false`, "无异常时 is_abnormal 必须在响应中且为 false")

	// 标异常后：为 true
	require.NoError(t, f.pf.svc.db.Model(&domain.PickupRecord{}).
		Where("student_id = ?", f.studentID).Update("is_abnormal", true).Error)

	resp, err = f.svc.ListPickupRecords(ctx(), f.pc, 1, 20)
	require.NoError(t, err)
	s, err = json.Marshal(resp["list"])
	require.NoError(t, err)
	assert.Contains(t, string(s), `"is_abnormal":true`, "有异常时 is_abnormal 必须为 true")
}
