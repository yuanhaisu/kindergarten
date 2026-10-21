package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/ai/face"
	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
)

// 底库注册（FRG）测试：用 mock provider + 注入取图，验证注册状态落行与重建语义。

const frTestBase = 992000000

func openFRTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := openRQTestDB(t)
	require.NoError(t, db.AutoMigrate(&domain.StudentFace{}, &domain.Student{}, &domain.File{}))
	return db
}

func frCleanup(t *testing.T, db *gorm.DB, studentID uint64) {
	t.Helper()
	db.Unscoped().Where("student_id = ?", studentID).Delete(&domain.StudentFace{})
	db.Unscoped().Where("id = ?", studentID).Delete(&domain.Student{})
	db.Unscoped().Where("id >= ? AND id < ?", uint64(frTestBase+100), uint64(frTestBase+200)).Delete(&domain.File{})
}

// frSeedUploader 建一个上传者用户（files.user_id / growth_daily_logs.creator_id 有外键约束）。
// 复用已存在的行：被其他表外键引用时删除会被数据库拒绝，硬删会静默失败。
func frSeedUploader(t *testing.T, db *gorm.DB) uint64 {
	t.Helper()
	uid := uint64(frTestBase + 900)
	var u domain.User
	if err := db.Unscoped().Where("id = ?", uid).First(&u).Error; err == nil {
		return uid
	}
	require.NoError(t, db.Create(&domain.User{BaseModel: domain.BaseModel{ID: uid}, Phone: "13900000900", Nickname: "底库上传者"}).Error)
	return uid
}

func TestFaceRegistry_AddListDeleteLifecycle(t *testing.T) {
	db := openFRTestDB(t)
	ctx := context.Background()
	studentID := uint64(frTestBase + 1)
	frCleanup(t, db, studentID)

	validDate := time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, db.Create(&domain.Student{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: studentID}},
		KindergartenID: 4, ClassID: 6, Name: "底库娃", Status: 1, Birthday: validDate, EnrollDate: validDate}).Error)

	// 两张已上传的图片素材（统一上传通道的产物）
	uid := frSeedUploader(t, db)
	fileIDs := []uint64{frTestBase + 101, frTestBase + 102}
	for i, fid := range fileIDs {
		require.NoError(t, db.Create(&domain.File{BaseModel: domain.BaseModel{ID: fid}, UserID: uid, BizType: "face",
			FileURL: "http://mock.local/face_" + uitoa(uint64(i)) + ".png", FileType: "image"}).Error)
	}

	svc := NewFaceRegistryService(db, face.NewMockProvider(), config.FaceConfig{Enabled: true})
	svc.fetch = func(ctx context.Context, url string) ([]byte, error) { return []byte("FACE"), nil }

	// 录入 → 注册成功（TC-NEW-FRG-0001）
	data, err := svc.AddStudentFaces(ctx, 4, studentID, fileIDs)
	require.NoError(t, err)
	assert.Equal(t, int8(1), data["status"], "全部照片注册成功后聚合状态为已注册")
	assert.Equal(t, "已注册", data["status_label"])

	var rows []domain.StudentFace
	require.NoError(t, db.Where("student_id = ?", studentID).Order("id ASC").Find(&rows).Error)
	require.Len(t, rows, 2)
	for _, r := range rows {
		assert.Equal(t, domain.FaceStatusIndexed, r.Status)
		assert.NotEmpty(t, r.PersonRef, "注册成功应记录云端人员引用")
	}

	// 非图片素材：拒绝
	require.NoError(t, db.Create(&domain.File{BaseModel: domain.BaseModel{ID: frTestBase + 103}, UserID: uid, BizType: "face",
		FileURL: "http://mock.local/doc.pdf", FileType: "file"}).Error)
	_, err = svc.AddStudentFaces(ctx, 4, studentID, []uint64{frTestBase + 103})
	assert.Error(t, err, "非图片素材不得进底库")

	// 上限 5 张：已有 2 张，再加 4 张应拒绝
	_, err = svc.AddStudentFaces(ctx, 4, studentID, []uint64{frTestBase + 104, frTestBase + 105, frTestBase + 106, frTestBase + 107})
	assert.EqualError(t, err, "FACE_LIMIT_EXCEEDED")

	// 删除一张 → 重建（剩 1 张仍为已注册）
	require.NoError(t, svc.DeleteStudentFace(ctx, 4, studentID, rows[0].ID))
	data, err = svc.ListStudentFaces(ctx, 4, studentID)
	require.NoError(t, err)
	assert.Equal(t, int8(1), data["status"])
	assert.Equal(t, 1, data["total"])

	// 删除最后一张 → 未录入（云端条目被移除）
	require.NoError(t, svc.DeleteStudentFace(ctx, 4, studentID, rows[1].ID))

	// 底库变更不影响既有记录与照片点名（TC-NEW-FRG-0006）：播种一条该学生的点名关联，删除后仍在
	namedMedia := uint64(frTestBase + 50)
	stConfirmed := domain.MediaStudentStatusConfirmed
	db.Unscoped().Where("media_id = ?", namedMedia).Delete(&domain.GrowthMediaStudent{})
	require.NoError(t, db.Create(&domain.GrowthMediaStudent{MediaID: namedMedia, StudentID: studentID, Status: &stConfirmed}).Error)
	var stillTagged int64
	db.Model(&domain.GrowthMediaStudent{}).Where("media_id = ? AND student_id = ?", namedMedia, studentID).Count(&stillTagged)
	assert.Equal(t, int64(1), stillTagged, "底库照片删除不影响既有点名关联")
	db.Unscoped().Where("media_id = ?", namedMedia).Delete(&domain.GrowthMediaStudent{})
	data, err = svc.ListStudentFaces(ctx, 4, studentID)
	require.NoError(t, err)
	assert.Equal(t, int8(0), data["status"], "清空后回到未录入")
	assert.Equal(t, 0, data["total"])

	// 删除不存在的照片：NOT_FOUND
	err = svc.DeleteStudentFace(ctx, 4, studentID, rows[0].ID)
	assert.Error(t, err)

	// 跨园访问：NOT_FOUND（TC-NEW-FRG-0005 的数据面：底库按园隔离）
	_, err = svc.ListStudentFaces(ctx, 999, studentID)
	assert.Error(t, err)

	frCleanup(t, db, studentID)
}

func TestFaceRegistry_DisabledRejectsWrites(t *testing.T) {
	db := openFRTestDB(t)
	ctx := context.Background()
	studentID := uint64(frTestBase + 2)
	frCleanup(t, db, studentID)
	validDate := time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, db.Create(&domain.Student{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: studentID}},
		KindergartenID: 4, ClassID: 6, Name: "未开园娃", Status: 1, Birthday: validDate, EnrollDate: validDate}).Error)

	svc := NewFaceRegistryService(db, face.NewMockProvider(), config.FaceConfig{Enabled: false})
	_, err := svc.AddStudentFaces(ctx, 4, studentID, []uint64{frTestBase + 101})
	assert.ErrorIs(t, err, ErrFaceNotEnabled, "未开启人脸能力的园所不得录入底库（合规前置门）")
	_, err = svc.RetryStudentFaces(ctx, 4, studentID)
	assert.ErrorIs(t, err, ErrFaceNotEnabled)

	frCleanup(t, db, studentID)
}

// 注册失败落到行状态（接口不失败），失败原因可诊断并可重试成功。
func TestFaceRegistry_FailureVisibleAndRetryable(t *testing.T) {
	db := openFRTestDB(t)
	ctx := context.Background()
	studentID := uint64(frTestBase + 3)
	frCleanup(t, db, studentID)
	validDate := time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, db.Create(&domain.Student{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: studentID}},
		KindergartenID: 4, ClassID: 6, Name: "重试娃", Status: 1, Birthday: validDate, EnrollDate: validDate}).Error)
	fid := uint64(frTestBase + 101)
	uid := frSeedUploader(t, db)
	require.NoError(t, db.Create(&domain.File{BaseModel: domain.BaseModel{ID: fid}, UserID: uid, BizType: "face",
		FileURL: "http://mock.local/f.png", FileType: "image"}).Error)

	svc := NewFaceRegistryService(db, face.NewMockProvider(), config.FaceConfig{Enabled: true})
	failFetch := true
	svc.fetch = func(ctx context.Context, url string) ([]byte, error) {
		if failFetch {
			return nil, assertErr("取图超时")
		}
		return []byte("FACE"), nil
	}

	data, err := svc.AddStudentFaces(ctx, 4, studentID, []uint64{fid})
	require.NoError(t, err, "注册失败不使接口失败——原因落在行状态")
	assert.Equal(t, int8(2), data["status"], "取图失败 → 聚合状态为注册失败（TC-NEW-FRG-0003）")

	var row domain.StudentFace
	require.NoError(t, db.Where("student_id = ?", studentID).First(&row).Error)
	assert.Contains(t, row.LastError, "取图超时", "失败原因可诊断")

	failFetch = false
	data, err = svc.RetryStudentFaces(ctx, 4, studentID)
	require.NoError(t, err)
	assert.Equal(t, int8(1), data["status"], "条件恢复后重试成功（TC-NEW-FRG-0004）")

	frCleanup(t, db, studentID)
}
