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

// 这些测试打真实 MySQL（本地 configs/app.ini），覆盖 spec/pickup-custody 的
// 接走登记端到端行为：扫码归因、例外通道、交替序列、重复/提前理由必填。
// 决策逻辑本身另有纯函数单测（pickup_rules_test.go）；此处验证「决策 + 落库」组合。
//
// 用独立测试园所/班级（高 ID），每例 t.Cleanup 硬删自建行，不污染真实数据。

const (
	testPickupKG    uint64 = 990001
	testPickupClass uint64 = 990001
	testPickupClass2 uint64 = 990002
	// testPickupOperatorID 作 OperatorID（核验人）；pickup_records.operator_id 有外键指向 users
	testPickupOperatorID uint64 = 990001
)

var pickupTestDB *gorm.DB

// openPickupTestDB 打开真实 MySQL。DSN 从 configs/app.ini 读。
func openPickupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	if pickupTestDB != nil {
		return pickupTestDB
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
	pickupTestDB = db
	return db
}

// pickupFixture 一次测试所需的班级/学生/老师上下文。
type pickupFixture struct {
	svc       *TeacherPortalService
	tc        *TeacherPortalContext
	studentID uint64
	classID   uint64
}

// newPickupFixture 建班级 + 学生，deadline 写入 class.pickup_deadline。
// cleanup 硬删本轮建的全部行。
func newPickupFixture(t *testing.T, deadline string, classID uint64) *pickupFixture {
	t.Helper()
	db := openPickupTestDB(t)
	ctx := context.Background()
	now := time.Now()

	// 清理上一轮残留
	cleanupPickupData(t, db, classID)

	// 先建园所：classes.kindergarten_id 有外键指向 kindergartens。
	// 幂等：多个 fixture 共用 testPickupKG（如 DeadlinePerClass 的两班），只建一次。
	var kgCount int64
	db.WithContext(ctx).Model(&domain.Kindergarten{}).Where("id = ?", testPickupKG).Count(&kgCount)
	if kgCount == 0 {
		require.NoError(t, db.WithContext(ctx).Create(&domain.Kindergarten{
			BaseModel: domain.BaseModel{ID: testPickupKG},
			Name:      "测试园所-接送",
		}).Error)
	}

	// 建老师用户：pickup_records.operator_id 有外键指向 users
	var userCount int64
	db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", testPickupOperatorID).Count(&userCount)
	if userCount == 0 {
		require.NoError(t, db.WithContext(ctx).Create(&domain.User{
			BaseModel: domain.BaseModel{ID: testPickupOperatorID},
			Phone:     "13900000001",
			Nickname:  "测试核验老师",
		}).Error)
	}

	require.NoError(t, db.WithContext(ctx).Create(&domain.Class{
		BaseModel:      domain.BaseModel{ID: classID},
		KindergartenID: testPickupKG,
		Name:           fmt.Sprintf("测试%d班", classID),
		Status:         1,
		PickupDeadline: deadline,
	}).Error)

	studentID := classID * 1000
	// time.Time 字段必须给合法值：MySQL 严格模式拒绝零值 '0000-00-00'
	validDate := time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, db.WithContext(ctx).Create(&domain.Student{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: studentID}},
		KindergartenID:  testPickupKG,
		ClassID:         classID,
		Name:            fmt.Sprintf("测试学生%d", studentID),
		Birthday:        validDate,
		EnrollDate:      validDate,
		Status:          1,
	}).Error)

	svc := NewTeacherPortalService(db, nil, repository.NewPickupRepo(db), repository.NewMessageRepo(db), repository.NewNoticeRepo(db), nil, config.FaceConfig{})
	f := &pickupFixture{
		svc: svc,
		tc: &TeacherPortalContext{
			UserID:         testPickupOperatorID,
			KindergartenID: testPickupKG,
			ClassID:        classID,
		},
		studentID: studentID,
		classID:   classID,
	}
	t.Cleanup(func() { cleanupPickupData(t, db, classID) })
	_ = now
	return f
}

// cleanupPickupData 硬删本测试班级下的全部相关行（含软删行）。
//
// 各表列不同，只能按表分别删：
//   - pickup_records / morning_health_checks / students → 有 student_id 与 class_id
//   - pickup_qr_codes / pickup_daily_designations / pickup_authorizations → 仅 student_id
func cleanupPickupData(t *testing.T, db *gorm.DB, classID uint64) {
	t.Helper()
	ctx := context.Background()
	studentID := classID * 1000

	del := func(sql string, args ...any) {
		if err := db.WithContext(ctx).Exec(sql, args...).Error; err != nil {
			t.Logf("cleanup: %v", err)
		}
	}
	del("DELETE FROM pickup_records WHERE student_id = ?", studentID)
	del("DELETE FROM pickup_qr_codes WHERE student_id = ?", studentID)
	del("DELETE FROM pickup_daily_designations WHERE student_id = ?", studentID)
	del("DELETE FROM pickup_authorizations WHERE student_id = ?", studentID)
	del("DELETE FROM morning_health_checks WHERE student_id = ?", studentID)
	// 消息的 reference_id 在接走通知里是记录 ID、在滞留通知里是学生 ID，
	// 只按 reference_id 删不干净；测试家长的 UserID 是确定值，一并清。
	del("DELETE FROM messages WHERE reference_id = ? OR user_id = ?", studentID, studentID+700000)
	// 子表先删：student_parents 引用 students，students 引用 classes
	del("DELETE FROM student_parents WHERE student_id = ?", studentID)
	del("DELETE FROM students WHERE id = ?", studentID)
	del("DELETE FROM classes WHERE id = ?", classID)
	// 园所与用户是跨 fixture 共享的固定测试 ID，不在每例 cleanup 里删，
	// 否则并存的第二个 fixture（如 DeadlinePerClass 的两班）会被误删外键目标。
}

// makeAuthorization 建一条授权接送人，返回 ID。
func (f *pickupFixture) makeAuthorization(t *testing.T, name string, reviewStatus, status int8, expireAt *time.Time) uint64 {
	t.Helper()
	ctx := context.Background()
	auth := &domain.PickupAuthorization{
		StudentID:    f.studentID,
		Name:         name,
		Phone:        "13800000000",
		Relation:     "grandmother",
		Avatar:       "https://cdn.example.com/pickup/person/3.jpg",
		Status:       status,
		ReviewStatus: reviewStatus,
		AuthType:     "permanent",
		AllowPickup:  true,
		// 显式赋值：domain 不再依赖 DB 默认值（GORM 对零值会跳过字段）
		NotifyOnPickup: true,
		ExpireAt:       expireAt,
	}
	require.NoError(t, repository.NewPickupRepo(f.svc.db).CreateAuthorization(ctx, auth))
	return auth.ID
}

// makeQR 建一张绑到指定授权人的凭证。
// 走 repo 的 CreateQRCode（显式 Select 全字段），否则 IsActive:false 会被 GORM 跳过、
// 由 DB 默认值写成 true。
func (f *pickupFixture) makeQR(t *testing.T, authID uint64, code string, expireAt time.Time, active bool) {
	t.Helper()
	repo := repository.NewPickupRepo(f.svc.db)
	require.NoError(t, repo.CreateQRCode(ctx(), &domain.PickupQRCode{
		StudentID:       f.studentID,
		AuthorizationID: authID,
		Code:            code,
		ExpireAt:        expireAt,
		IsActive:        active,
	}))
}

func ctx() context.Context { return context.Background() }

// endOfDay 返回当日 23:59:59，用作「未过期」凭证的过期时刻。
func endOfDay() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
}

// yesterdayEnd 返回昨日 23:59:59，用作「已过期」凭证。
func yesterdayEnd() time.Time {
	return endOfDay().Add(-24 * time.Hour)
}

// ==================== 扫码路径（3 场景）====================

// 场景：扫码成功并归因到接走人（TC-NEW-PKU-020）
func TestCreatePickupRecord_ScanAttributesToPerson(t *testing.T) {
	f := newPickupFixture(t, "00:00", testPickupClass) // 00:00 → 不判提前
	authID := f.makeAuthorization(t, "奶奶（张三）", 1, 1, nil)
	f.makeQR(t, authID, "QSCANOK001", endOfDay(), true)

	// 先送入，否则接走被「须先送入」拒
	_, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{StudentID: f.studentID, Type: 1}, domain.PickupMethodManual)
	require.NoError(t, err)

	resp, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{QRCode: "QSCANOK001"}, domain.PickupMethodScan)
	require.NoError(t, err, "有效凭证扫码必须成功")

	assert.Equal(t, true, resp["is_authorized"], "扫码路径必为授权接走")
	assert.Equal(t, false, resp["is_authorized_override"], "扫码路径不是例外")

	person, ok := resp["person"].(map[string]any)
	require.True(t, ok, "响应必须含 person 供人证对照")
	assert.Equal(t, "奶奶（张三）", person["name"], "person.name 必须是凭证绑定的接走人")
	assert.Equal(t, "https://cdn.example.com/pickup/person/3.jpg", person["photo"],
		"person.photo 必须回显授权人头像，供老师人证对照")

	// PickupPersonID 必须落库为凭证绑定的授权人（I1 闭合的落库依据）
	var rec domain.PickupRecord
	require.NoError(t, f.svc.db.Where("student_id = ?", f.studentID).Order("id DESC").First(&rec).Error)
	require.NotNil(t, rec.PickupPersonID, "PickupPersonID 不得为空")
	assert.Equal(t, authID, *rec.PickupPersonID, "PickupPersonID 必须为凭证绑定的授权人")

	// 凭证即焚
	var qr domain.PickupQRCode
	require.NoError(t, f.svc.db.Where("code = ?", "QSCANOK001").First(&qr).Error)
	assert.False(t, qr.IsActive, "用过的凭证必须失效（一次有效）")
}

// 场景：接走人授权在凭证生成后被撤销（TC-NEW-PKU-021）
func TestCreatePickupRecord_ScanRejectedWhenAuthorizationRevoked(t *testing.T) {
	f := newPickupFixture(t, "00:00", testPickupClass)
	authID := f.makeAuthorization(t, "奶奶（张三）", 1, 1, nil)
	f.makeQR(t, authID, "QSCANREV01", endOfDay(), true)

	_, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{StudentID: f.studentID, Type: 1}, domain.PickupMethodManual)
	require.NoError(t, err)

	// 凭证建好后撤销授权
	require.NoError(t, f.svc.db.Model(&domain.PickupAuthorization{}).
		Where("id = ?", authID).Update("status", 0).Error)

	_, err = f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{QRCode: "QSCANREV01"}, domain.PickupMethodScan)
	assert.ErrorIs(t, err, ErrTeacherAuthorizationInvalid,
		"授权撤销后扫码必须被拒，不得产生接走记录")
}

// 场景：凭证已失效 / 已过期被拒（TC-NEW-PKU-022）
func TestCreatePickupRecord_ScanRejectedWhenCodeInvalid(t *testing.T) {
	f := newPickupFixture(t, "00:00", testPickupClass)
	authID := f.makeAuthorization(t, "奶奶（张三）", 1, 1, nil)

	// 已用过（is_active=false）
	f.makeQR(t, authID, "QSCANUSED01", endOfDay(), false)
	// 已过期
	f.makeQR(t, authID, "QSCANEXP001", yesterdayEnd(), true)

	_, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{QRCode: "QSCANUSED01"}, domain.PickupMethodScan)
	assert.ErrorIs(t, err, ErrTeacherInvalidQRCode, "已使用的凭证必须被拒")

	_, err = f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{QRCode: "QSCANEXP001"}, domain.PickupMethodScan)
	assert.ErrorIs(t, err, ErrTeacherInvalidQRCode, "已过期的凭证必须被拒")
}

// ==================== 例外通道（3 场景）====================

// 场景：手动登记从授权人列表选人（TC-NEW-PKU-030）
func TestCreatePickupRecord_ManualPickFromList(t *testing.T) {
	f := newPickupFixture(t, "00:00", testPickupClass)
	authID := f.makeAuthorization(t, "爸爸（王明）", 1, 1, nil)

	_, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{StudentID: f.studentID, Type: 1}, domain.PickupMethodManual)
	require.NoError(t, err)

	resp, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
		StudentID: f.studentID, Type: 2, AuthorizationID: &authID,
	}, domain.PickupMethodManual)
	require.NoError(t, err, "从授权人列表选人登记必须成功")

	assert.Equal(t, false, resp["is_authorized_override"], "列表选人不计例外")
	assert.Equal(t, true, resp["is_authorized"], "列表选人属授权接走")

	var rec domain.PickupRecord
	require.NoError(t, f.svc.db.Where("student_id = ?", f.studentID).Order("id DESC").First(&rec).Error)
	require.NotNil(t, rec.PickupPersonID, "PickupPersonID 不得为空")
	assert.Equal(t, authID, *rec.PickupPersonID, "PickupPersonID 必须落为所选授权人")
	assert.False(t, rec.IsAuthorizedOverride)
}

// 场景：标记为非授权/临时（TC-NEW-PKU-031）
func TestCreatePickupRecord_ManualOverride(t *testing.T) {
	f := newPickupFixture(t, "00:00", testPickupClass)

	_, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{StudentID: f.studentID, Type: 1}, domain.PickupMethodManual)
	require.NoError(t, err)

	resp, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
		StudentID: f.studentID, Type: 2,
		IsAuthorizedOverride: true, OverrideName: "邻居王五",
	}, domain.PickupMethodManual)
	require.NoError(t, err, "例外通道不得阻断登记")

	assert.Equal(t, true, resp["is_authorized_override"], "必须记为授权例外")

	// 例外时 person 回显来接者姓名（供老师人证对照），但不得伪造成授权人
	person, ok := resp["person"].(map[string]any)
	require.True(t, ok, "例外也应回显来接者，供老师比对")
	assert.Equal(t, "邻居王五", person["name"], "person.name 应是例外来接者姓名")
	assert.Empty(t, person["relation"], "例外无授权关系，relation 不得伪填")

	// 例外必须留姓名，落库可查
	var rec domain.PickupRecord
	require.NoError(t, f.svc.db.Where("student_id = ?", f.studentID).Order("id DESC").First(&rec).Error)
	assert.Equal(t, "邻居王五", rec.OverrideName, "来接者姓名必须落库")
	assert.Nil(t, rec.PickupPersonID, "例外时 PickupPersonID 必须为空")
	assert.True(t, rec.IsAuthorizedOverride)
}

// 场景：例外缺姓名被拒（例外不阻断，但必须留痕）
func TestCreatePickupRecord_ManualOverrideRequiresName(t *testing.T) {
	f := newPickupFixture(t, "00:00", testPickupClass)

	_, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{StudentID: f.studentID, Type: 1}, domain.PickupMethodManual)
	require.NoError(t, err)

	_, err = f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
		StudentID: f.studentID, Type: 2, IsAuthorizedOverride: true,
	}, domain.PickupMethodManual)
	assert.ErrorIs(t, err, ErrTeacherOverrideNameRequired, "例外必须填来接者姓名")
}

// ==================== 交替序列（3 场景）====================

// 场景：同日多次往返可登记（TC-NEW-PKU-040）
func TestCreatePickupRecord_MultiRoundTrip(t *testing.T) {
	f := newPickupFixture(t, "00:00", testPickupClass)
	authID := f.makeAuthorization(t, "奶奶（张三）", 1, 1, nil)

	step := func(typ int8, remark string) error {
		_, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
			StudentID: f.studentID, Type: typ,
			AuthorizationID: &authID, Remark: remark,
		}, domain.PickupMethodManual)
		return err
	}
	require.NoError(t, step(1, ""), "上午送入")
	require.NoError(t, step(2, "中午接走就医"), "中午接走就医（重复否=首次，但 00:00 不提前，理由可选）")
	require.NoError(t, step(1, ""), "下午返园送入")
	require.NoError(t, step(2, "放学接走"), "放学接走（第 2 次接走，理由必填）")
}

// 场景：连续同向被拒（TC-NEW-PKU-041）
func TestCreatePickupRecord_ConsecutiveSameDirectionRejected(t *testing.T) {
	f := newPickupFixture(t, "00:00", testPickupClass)

	_, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{StudentID: f.studentID, Type: 1}, domain.PickupMethodManual)
	require.NoError(t, err)

	_, err = f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{StudentID: f.studentID, Type: 1}, domain.PickupMethodManual)
	assert.ErrorIs(t, err, ErrTeacherConsecutiveDirection, "连续送入必须被拒")

	// 接走后再接走同样被拒
	authID := f.makeAuthorization(t, "奶奶（张三）", 1, 1, nil)
	_, err = f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
		StudentID: f.studentID, Type: 2, AuthorizationID: &authID,
	}, domain.PickupMethodManual)
	require.NoError(t, err)

	_, err = f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
		StudentID: f.studentID, Type: 2, AuthorizationID: &authID,
	}, domain.PickupMethodManual)
	assert.ErrorIs(t, err, ErrTeacherConsecutiveDirection, "连续接走必须被拒")
}

// 场景：状态以最后一次转移为准（TC-NEW-PKU-042）
func TestCreatePickupRecord_StateFollowsLastTransfer(t *testing.T) {
	f := newPickupFixture(t, "00:00", testPickupClass)
	authID := f.makeAuthorization(t, "奶奶（张三）", 1, 1, nil)

	_, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{StudentID: f.studentID, Type: 1}, domain.PickupMethodManual)
	require.NoError(t, err)
	_, err = f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
		StudentID: f.studentID, Type: 2, AuthorizationID: &authID,
	}, domain.PickupMethodManual)
	require.NoError(t, err)
	_, err = f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{StudentID: f.studentID, Type: 1}, domain.PickupMethodManual)
	require.NoError(t, err)

	last, err := f.svc.pickup.FindLastRecordOfDay(ctx(), f.studentID, time.Now())
	require.NoError(t, err)
	require.NotNil(t, last)
	assert.Equal(t, domain.PickupTypeDropOff, last.Type,
		"送入→接走→送入 后，最后一次是送入，状态应为在园")
}

// ==================== 重复接走理由必填（4 场景）====================

// 场景：首次接走理由可选（TC-NEW-PKU-050）
func TestCreatePickupRecord_FirstPickupRemarkOptional(t *testing.T) {
	f := newPickupFixture(t, "00:00", testPickupClass) // 不提前
	authID := f.makeAuthorization(t, "奶奶（张三）", 1, 1, nil)

	_, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{StudentID: f.studentID, Type: 1}, domain.PickupMethodManual)
	require.NoError(t, err)

	_, err = f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
		StudentID: f.studentID, Type: 2, AuthorizationID: &authID,
	}, domain.PickupMethodManual)
	assert.NoError(t, err, "首次接走不填理由必须成功")
}

// 场景：重复接走缺理由被拒（TC-NEW-PKU-051）
func TestCreatePickupRecord_RepeatPickupRequiresRemark(t *testing.T) {
	f := newPickupFixture(t, "00:00", testPickupClass)
	authID := f.makeAuthorization(t, "奶奶（张三）", 1, 1, nil)

	step := func(typ int8, remark string) error {
		_, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
			StudentID: f.studentID, Type: typ, AuthorizationID: &authID, Remark: remark,
		}, domain.PickupMethodManual)
		return err
	}
	require.NoError(t, step(1, ""))
	require.NoError(t, step(2, ""), "首次接走")
	require.NoError(t, step(1, ""), "返园")

	err := step(2, "") // 第 2 次接走，无理由
	assert.ErrorIs(t, err, ErrTeacherRemarkRequired, "重复接走缺理由必须被拒")
}

// 场景：重复接走带理由成功（TC-NEW-PKU-052）
func TestCreatePickupRecord_RepeatPickupWithRemarkSucceeds(t *testing.T) {
	f := newPickupFixture(t, "00:00", testPickupClass)
	authID := f.makeAuthorization(t, "奶奶（张三）", 1, 1, nil)

	step := func(typ int8, remark string) error {
		_, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
			StudentID: f.studentID, Type: typ, AuthorizationID: &authID, Remark: remark,
		}, domain.PickupMethodManual)
		return err
	}
	require.NoError(t, step(1, ""))
	require.NoError(t, step(2, ""))
	require.NoError(t, step(1, ""))
	assert.NoError(t, step(2, "放学接走"), "重复接走带理由必须成功")
}

// 场景：重复送入不强制理由（TC-NEW-PKU-053）
func TestCreatePickupRecord_RepeatDropOffNoRemark(t *testing.T) {
	f := newPickupFixture(t, "00:00", testPickupClass)
	authID := f.makeAuthorization(t, "奶奶（张三）", 1, 1, nil)

	step := func(typ int8, remark string) error {
		_, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
			StudentID: f.studentID, Type: typ, AuthorizationID: &authID, Remark: remark,
		}, domain.PickupMethodManual)
		return err
	}
	require.NoError(t, step(1, ""))
	require.NoError(t, step(2, ""))
	assert.NoError(t, step(1, ""), "第 2 次送入不填理由必须成功")
}

// ==================== 提前接走理由必填（4 场景）====================

// 场景：提前接走缺理由被拒（TC-NEW-PKU-060）
func TestCreatePickupRecord_EarlyPickupRequiresRemark(t *testing.T) {
	// deadline 23:59 → 当前时刻恒早于它 → 恒判「提前」
	f := newPickupFixture(t, "23:59", testPickupClass)
	authID := f.makeAuthorization(t, "奶奶（张三）", 1, 1, nil)

	_, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{StudentID: f.studentID, Type: 1}, domain.PickupMethodManual)
	require.NoError(t, err)

	_, err = f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
		StudentID: f.studentID, Type: 2, AuthorizationID: &authID,
	}, domain.PickupMethodManual)
	assert.ErrorIs(t, err, ErrTeacherRemarkRequired, "提前接走缺理由必须被拒")
}

// 场景：提前接走带理由成功（TC-NEW-PKU-061）
func TestCreatePickupRecord_EarlyPickupWithRemarkSucceeds(t *testing.T) {
	f := newPickupFixture(t, "23:59", testPickupClass)
	authID := f.makeAuthorization(t, "奶奶（张三）", 1, 1, nil)

	_, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{StudentID: f.studentID, Type: 1}, domain.PickupMethodManual)
	require.NoError(t, err)

	_, err = f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
		StudentID: f.studentID, Type: 2, AuthorizationID: &authID,
		Remark: "医院复诊，已与老师说明",
	}, domain.PickupMethodManual)
	assert.NoError(t, err, "提前接走带理由必须成功")
}

// 场景：放学后接走不要求理由（TC-NEW-PKU-062）
func TestCreatePickupRecord_OnTimePickupNoRemark(t *testing.T) {
	// deadline 00:00 → 当前时刻恒晚于它 → 恒不判「提前」
	f := newPickupFixture(t, "00:00", testPickupClass)
	authID := f.makeAuthorization(t, "奶奶（张三）", 1, 1, nil)

	_, err := f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{StudentID: f.studentID, Type: 1}, domain.PickupMethodManual)
	require.NoError(t, err)

	_, err = f.svc.CreatePickupRecord(ctx(), f.tc, TeacherPickupInput{
		StudentID: f.studentID, Type: 2, AuthorizationID: &authID,
	}, domain.PickupMethodManual)
	assert.NoError(t, err, "放学后接走不填理由必须成功")
}

// 场景：放学时刻按班级配置（TC-NEW-PKU-063）
func TestCreatePickupRecord_DeadlinePerClass(t *testing.T) {
	// 甲班 23:59（恒提前）→ 必填理由
	fast := newPickupFixture(t, "23:59", testPickupClass)
	authA := fast.makeAuthorization(t, "奶奶（甲）", 1, 1, nil)
	_, err := fast.svc.CreatePickupRecord(ctx(), fast.tc, TeacherPickupInput{StudentID: fast.studentID, Type: 1}, domain.PickupMethodManual)
	require.NoError(t, err)
	_, err = fast.svc.CreatePickupRecord(ctx(), fast.tc, TeacherPickupInput{
		StudentID: fast.studentID, Type: 2, AuthorizationID: &authA,
	}, domain.PickupMethodManual)
	assert.ErrorIs(t, err, ErrTeacherRemarkRequired, "23:59 班级应判提前")

	// 乙班 00:00（恒不提前）→ 无需理由
	slow := newPickupFixture(t, "00:00", testPickupClass2)
	authB := slow.makeAuthorization(t, "奶奶（乙）", 1, 1, nil)
	_, err = slow.svc.CreatePickupRecord(ctx(), slow.tc, TeacherPickupInput{StudentID: slow.studentID, Type: 1}, domain.PickupMethodManual)
	require.NoError(t, err)
	_, err = slow.svc.CreatePickupRecord(ctx(), slow.tc, TeacherPickupInput{
		StudentID: slow.studentID, Type: 2, AuthorizationID: &authB,
	}, domain.PickupMethodManual)
	assert.NoError(t, err, "00:00 班级不应判提前，理由可选")
}
