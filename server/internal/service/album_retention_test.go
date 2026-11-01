package service

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

// OS-ALB-9：软删相册 180 天留存清理（TC-NEW-ALB-0022）与 Phase 1 指标事件。
//
// 打真实 MySQL（本地 configs/app.ini），固定 9962xx 夹具，t.Cleanup 硬删 0 残留。
// 清理只认 deleted_at <= now-180d：179 天与未删除的相册必须原样保留；
// operation_logs 不在清理范围（审计保留）。

const (
	testARKG      uint64 = 996201
	testARClass   uint64 = 996210
	testARUser    uint64 = 996220
	testARStudent uint64 = 996230

	testARAlbumExpired  uint64 = 996240 // 软删 181 天：必须被硬删
	testARAlbumRetained uint64 = 996241 // 软删 179 天：必须保留
	testARAlbumActive   uint64 = 996242 // 未删除：必须保留
	testARAlbumBatch1   uint64 = 996250 // 分批用例：5 本软删 181 天
	testARAlbumBatch5   uint64 = 996254
	testARPhotoBatch    uint64 = 996260 // 分批用例照片 996260–996264
)

func arCleanup(t *testing.T, db *gorm.DB) {
	t.Helper()
	del := func(sql string, args ...any) {
		if err := db.Exec(sql, args...).Error; err != nil {
			t.Logf("cleanup: %v", err)
		}
	}
	inAlbums := `SELECT id FROM albums WHERE creator_id = ? OR id BETWEEN ? AND ?`
	inPhotos := `SELECT id FROM album_photos WHERE album_id IN (` + inAlbums + `)`
	del(`DELETE FROM album_photo_students WHERE photo_id IN (`+inPhotos+`)`,
		testARUser, testARAlbumExpired, uint64(996299))
	del(`DELETE FROM album_students WHERE album_id IN (`+inAlbums+`)`,
		testARUser, testARAlbumExpired, uint64(996299))
	del(`DELETE FROM album_photos WHERE album_id IN (`+inAlbums+`)`,
		testARUser, testARAlbumExpired, uint64(996299))
	del(`DELETE FROM albums WHERE creator_id = ? OR id BETWEEN ? AND ?`,
		testARUser, testARAlbumExpired, uint64(996299))
	del("DELETE FROM operation_logs WHERE user_id BETWEEN ? AND ?", testARUser, uint64(996229))
	del("DELETE FROM students WHERE id BETWEEN ? AND ?", testARStudent, uint64(996239))
	del("DELETE FROM classes WHERE id BETWEEN ? AND ?", testARClass, uint64(996219))
	del("DELETE FROM users WHERE id BETWEEN ? AND ?", testARUser, uint64(996229))
	del("DELETE FROM kindergartens WHERE id = ?", testARKG)
}

type arFixture struct {
	db  *gorm.DB
	svc *AlbumRetentionService
}

var arMigrateOnce sync.Once

func newARFixture(t *testing.T) *arFixture {
	t.Helper()
	db := openCLPTestDB(t)
	arMigrateOnce.Do(func() {
		require.NoError(t, db.AutoMigrate(
			&domain.Album{}, &domain.AlbumPhoto{}, &domain.AlbumStudent{}, &domain.AlbumPhotoStudent{},
			&domain.OperationLog{}, &domain.Kindergarten{}, &domain.Class{}, &domain.User{}, &domain.Student{},
		))
	})
	arCleanup(t, db)
	t.Cleanup(func() { arCleanup(t, db) })

	require.NoError(t, db.Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: testARKG}, Name: "测试园所-相册清理",
	}).Error)
	require.NoError(t, db.Create(&domain.Class{
		BaseModel: domain.BaseModel{ID: testARClass}, KindergartenID: testARKG,
		Name: "清理一班", Grade: "小班", Status: 1,
	}).Error)
	require.NoError(t, db.Create(&domain.User{
		BaseModel: domain.BaseModel{ID: testARUser}, Nickname: "清理老师",
		Phone: "13900009921", PasswordHash: "x",
	}).Error)
	require.NoError(t, db.Create(&domain.Student{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testARStudent}},
		KindergartenID:  testARKG, ClassID: testARClass, Name: "清理幼儿", Status: 1,
		Birthday: time.Now().AddDate(-5, 0, 0), EnrollDate: time.Now().AddDate(-1, 0, 0),
	}).Error)
	return &arFixture{db: db, svc: NewAlbumRetentionService(db)}
}

// arSeedAlbum 造一本相册 + 若干照片 + 相册级/照片级幼儿关联。
// deletedAt 非空时直接以该时间软删落库（模拟「软删已满 N 天」的历史行）。
func arSeedAlbum(t *testing.T, db *gorm.DB, albumID uint64, photoIDs []uint64, deletedAt *time.Time) {
	t.Helper()
	album := domain.Album{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: albumID}},
		KindergartenID:  testARKG,
		ClassID:         testARClass,
		CreatorID:       testARUser,
		Title:           fmt.Sprintf("清理相册-%d", albumID),
		Category:        domain.AlbumCategoryDaily,
		AlbumDate:       time.Now(),
		PhotoCount:      len(photoIDs),
		Status:          domain.AlbumStatusPublished,
	}
	if deletedAt != nil {
		album.DeletedAt = gorm.DeletedAt{Time: *deletedAt, Valid: true}
	}
	require.NoError(t, db.Create(&album).Error)

	for i, pid := range photoIDs {
		require.NoError(t, db.Create(&domain.AlbumPhoto{
			BaseModel: domain.BaseModel{ID: pid}, AlbumID: albumID,
			FileURL: fmt.Sprintf("https://cdn.example.com/retention/%d.jpg", pid), SortNum: i,
		}).Error)
		require.NoError(t, db.Create(&domain.AlbumPhotoStudent{
			KindergartenID: testARKG, PhotoID: pid, StudentID: testARStudent,
		}).Error)
	}
	require.NoError(t, db.Create(&domain.AlbumStudent{
		KindergartenID: testARKG, AlbumID: albumID, StudentID: testARStudent,
	}).Error)
}

// arCount 含软删行的计数：清理结果的「残留 0」必须在 Unscoped 口径下成立。
func arCount(t *testing.T, db *gorm.DB, model any, query string, args ...any) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Unscoped().Model(model).Where(query, args...).Count(&n).Error)
	return n
}

// TC-NEW-ALB-0022：软删满 180 天才清理；179 天与未删除的相册原样保留；
// 照片/照片级关联/相册级参与者全部硬删，operation_logs 审计保留。
func TestAlbumRetentionCleanupExpiresAfter180Days(t *testing.T) {
	f := newARFixture(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)
	expiredAt := now.Add(-181 * 24 * time.Hour)
	retainedAt := now.Add(-179 * 24 * time.Hour)

	arSeedAlbum(t, f.db, testARAlbumExpired, []uint64{996270, 996271}, &expiredAt)
	arSeedAlbum(t, f.db, testARAlbumRetained, []uint64{996272}, &retainedAt)
	arSeedAlbum(t, f.db, testARAlbumActive, []uint64{996273}, nil)
	require.NoError(t, f.db.Create(&domain.OperationLog{
		KindergartenID: testARKG, UserID: testARUser, Module: "albums",
		Action: "publish", TargetType: "album", TargetID: testARAlbumExpired,
		Detail: `{"title":"清理相册"}`,
	}).Error)

	stats, err := f.svc.CountExpiredAlbums(ctx, now)
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.Albums, "只有 181 天的相册到期")
	assert.Equal(t, int64(2), stats.Photos)
	assert.Equal(t, int64(2), stats.PhotoStudents)
	assert.Equal(t, int64(1), stats.AlbumStudents)

	deleted, err := f.svc.CleanupExpiredAlbums(ctx, now, 100)
	require.NoError(t, err)
	assert.Equal(t, 1, deleted)

	// 到期相册与其全部内容硬删，残留 0。
	assert.Equal(t, int64(0), arCount(t, f.db, &domain.Album{}, "id = ?", testARAlbumExpired))
	assert.Equal(t, int64(0), arCount(t, f.db, &domain.AlbumPhoto{}, "album_id = ?", testARAlbumExpired))
	assert.Equal(t, int64(0), arCount(t, f.db, &domain.AlbumStudent{}, "album_id = ?", testARAlbumExpired))
	assert.Equal(t, int64(0), arCount(t, f.db, &domain.AlbumPhotoStudent{}, "photo_id IN ?", []uint64{996270, 996271}))

	// 179 天的相册保留，且仍是软删状态（清理没有提前删它）。
	assert.Equal(t, int64(1), arCount(t, f.db, &domain.Album{}, "id = ? AND deleted_at IS NOT NULL", testARAlbumRetained))
	assert.Equal(t, int64(1), arCount(t, f.db, &domain.AlbumPhoto{}, "album_id = ?", testARAlbumRetained))
	assert.Equal(t, int64(1), arCount(t, f.db, &domain.AlbumStudent{}, "album_id = ?", testARAlbumRetained))
	assert.Equal(t, int64(1), arCount(t, f.db, &domain.AlbumPhotoStudent{}, "photo_id = ?", 996272))

	// 未删除的相册完全不受影响。
	assert.Equal(t, int64(1), arCount(t, f.db, &domain.Album{}, "id = ? AND deleted_at IS NULL", testARAlbumActive))
	assert.Equal(t, int64(1), arCount(t, f.db, &domain.AlbumPhoto{}, "album_id = ?", testARAlbumActive))
	assert.Equal(t, int64(1), arCount(t, f.db, &domain.AlbumPhotoStudent{}, "photo_id = ?", 996273))

	// 审计保留：相册行被删掉后，发布日志仍可按 target_id 追查。
	assert.Equal(t, int64(1), arCount(t, f.db, &domain.OperationLog{},
		"target_type = ? AND target_id = ?", "album", testARAlbumExpired))

	// 幂等：再跑一次不再删任何行。
	deleted, err = f.svc.CleanupExpiredAlbums(ctx, now, 100)
	require.NoError(t, err)
	assert.Equal(t, 0, deleted)
}

// -dry-run 语义：CountExpiredAlbums 只统计，不删除任何行。
func TestAlbumRetentionDryRunDeletesNothing(t *testing.T) {
	f := newARFixture(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)
	expiredAt := now.Add(-181 * 24 * time.Hour)
	arSeedAlbum(t, f.db, testARAlbumExpired, []uint64{996274, 996275}, &expiredAt)

	stats, err := f.svc.CountExpiredAlbums(ctx, now)
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.Albums)
	assert.Equal(t, int64(2), stats.Photos)
	assert.Equal(t, int64(2), stats.PhotoStudents)
	assert.Equal(t, int64(1), stats.AlbumStudents)

	assert.Equal(t, int64(1), arCount(t, f.db, &domain.Album{}, "id = ?", testARAlbumExpired))
	assert.Equal(t, int64(2), arCount(t, f.db, &domain.AlbumPhoto{}, "album_id = ?", testARAlbumExpired))
	assert.Equal(t, int64(2), arCount(t, f.db, &domain.AlbumPhotoStudent{}, "photo_id IN ?", []uint64{996274, 996275}))
	assert.Equal(t, int64(1), arCount(t, f.db, &domain.AlbumStudent{}, "album_id = ?", testARAlbumExpired))

	// true 清理仍然删得掉：dry-run 没有留下任何副作用（如误标状态）。
	deleted, err := f.svc.CleanupExpiredAlbums(ctx, now, 100)
	require.NoError(t, err)
	assert.Equal(t, 1, deleted)
	assert.Equal(t, int64(0), arCount(t, f.db, &domain.Album{}, "id = ?", testARAlbumExpired))
}

// 分批：batch-size=2 时 5 本到期相册要全部删掉，不能只删第一批。
func TestAlbumRetentionCleanupInBatches(t *testing.T) {
	f := newARFixture(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)
	expiredAt := now.Add(-181 * 24 * time.Hour)

	albumIDs := []uint64{testARAlbumBatch1, 996251, 996252, 996253, testARAlbumBatch5}
	for i, id := range albumIDs {
		arSeedAlbum(t, f.db, id, []uint64{testARPhotoBatch + uint64(i)}, &expiredAt)
	}

	deleted, err := f.svc.CleanupExpiredAlbums(ctx, now, 2)
	require.NoError(t, err)
	assert.Equal(t, 5, deleted)
	for i, id := range albumIDs {
		assert.Equal(t, int64(0), arCount(t, f.db, &domain.Album{}, "id = ?", id), "相册 %d 应已硬删", id)
		assert.Equal(t, int64(0), arCount(t, f.db, &domain.AlbumPhoto{}, "album_id = ?", id))
		assert.Equal(t, int64(0), arCount(t, f.db, &domain.AlbumStudent{}, "album_id = ?", id))
		assert.Equal(t, int64(0), arCount(t, f.db, &domain.AlbumPhotoStudent{}, "photo_id = ?", testARPhotoBatch+uint64(i)))
	}
}

// ==================== 指标事件（design D10） ====================

type arCapturedRecord struct {
	Msg   string
	Attrs map[string]string
}

type arCaptureHandler struct {
	records []arCapturedRecord
}

func (h *arCaptureHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *arCaptureHandler) Handle(_ context.Context, r slog.Record) error {
	rec := arCapturedRecord{Msg: r.Message, Attrs: map[string]string{}}
	r.Attrs(func(a slog.Attr) bool {
		rec.Attrs[a.Key] = fmt.Sprint(a.Value.Any())
		return true
	})
	h.records = append(h.records, rec)
	return nil
}

func (h *arCaptureHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *arCaptureHandler) WithGroup(string) slog.Handler      { return h }

func arCaptureLogs(t *testing.T) *arCaptureHandler {
	t.Helper()
	h := &arCaptureHandler{}
	prev := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return h
}

func arRecordsWithMsg(h *arCaptureHandler, msg string) []arCapturedRecord {
	out := make([]arCapturedRecord, 0, len(h.records))
	for _, rec := range h.records {
		if rec.Msg == msg {
			out = append(out, rec)
		}
	}
	return out
}

func arFindRecord(t *testing.T, h *arCaptureHandler, msg string) arCapturedRecord {
	t.Helper()
	found := arRecordsWithMsg(h, msg)
	require.NotEmptyf(t, found, "未找到事件 %s", msg)
	return found[len(found)-1]
}

// 指标事件只带 ID 与计数：不出现手机号/openid 等敏感字段，也没有多余的键。
func TestAlbumMetricEventsCarryOnlyIDsAndCounts(t *testing.T) {
	h := arCaptureLogs(t)
	ctx := context.Background()

	LogAlbumPublish(ctx, AlbumPublishEvent{
		KindergartenID: 7, ClassID: 8, AlbumID: 9, UserID: 10, PhotoCount: 3,
	})
	LogAlbumParentView(ctx, AlbumParentViewEvent{
		KindergartenID: 7, ClassID: 8, AlbumID: 9, StudentID: 11, VisiblePhotoCount: 2,
	})
	LogAlbumOptOut(ctx, AlbumOptOutEvent{KindergartenID: 7, UserID: 10, AllowClassShare: false})

	require.Len(t, h.records, 3)
	assert.Equal(t, AlbumEventPublish, h.records[0].Msg)
	assert.Equal(t, map[string]string{
		"kindergarten_id": "7", "class_id": "8", "album_id": "9",
		"user_id": "10", "photo_count": "3", "role": "teacher",
	}, h.records[0].Attrs)

	assert.Equal(t, AlbumEventParentView, h.records[1].Msg)
	assert.Equal(t, map[string]string{
		"kindergarten_id": "7", "class_id": "8", "album_id": "9",
		"student_id": "11", "visible_photo_count": "2", "role": "parent",
	}, h.records[1].Attrs)

	assert.Equal(t, AlbumEventOptOut, h.records[2].Msg)
	assert.Equal(t, map[string]string{
		"kindergarten_id": "7", "user_id": "10", "allow_class_share": "false", "role": "parent",
	}, h.records[2].Attrs)

	phoneRe := regexp.MustCompile(`\b1[3-9]\d{9}\b`)
	for _, rec := range h.records {
		for _, key := range []string{"phone", "openid", "unionid", "nickname", "avatar"} {
			_, ok := rec.Attrs[key]
			assert.Falsef(t, ok, "%s 不应记录 %s", rec.Msg, key)
		}
		for k, v := range rec.Attrs {
			assert.NotRegexpf(t, phoneRe, v, "%s 的 %s 不应是手机号", rec.Msg, k)
		}
	}
}

// ==================== 指标调用点（teacher / parent / settings） ====================

// 发布成功必须产出 album_publish；创建与下架都不产出。
func TestTeacherAlbumPublishEmitsMetricEvent(t *testing.T) {
	h := arCaptureLogs(t)
	f := newTALFixture(t)
	ctx := context.Background()

	id, err := createTALAlbum(t, f, f.tcA, "发布埋点", "activity",
		talThisMonthDate(0).Format("2006-01-02"), talPhotoInputs(2))
	require.NoError(t, err)
	assert.Empty(t, arRecordsWithMsg(h, AlbumEventPublish), "建草稿不是发布")

	_, err = f.svc.PublishAlbum(ctx, f.tcA, id)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"kindergarten_id": fmt.Sprint(testTALKG),
		"class_id":        fmt.Sprint(testTALClassA),
		"album_id":        fmt.Sprint(id),
		"user_id":         fmt.Sprint(testTALUser),
		"photo_count":     "2",
		"role":            "teacher",
	}, arFindRecord(t, h, AlbumEventPublish).Attrs)

	_, err = f.svc.UnpublishAlbum(ctx, f.tcA, id)
	require.NoError(t, err)
	assert.Len(t, arRecordsWithMsg(h, AlbumEventPublish), 1, "下架不是发布")
}

// 家长打开已发布详情产出 album_parent_view，张数与响应一致；404 路径不产出。
func TestParentAlbumViewEmitsMetricEvent(t *testing.T) {
	h := arCaptureLogs(t)
	f := newPCALFixture(t)
	ctx := context.Background()

	detail, err := f.svc.GetClassAlbum(ctx, f.pcA, testPCALAlbumMain)
	require.NoError(t, err)
	visible, ok := detail["visible_photo_count"].(int)
	require.True(t, ok, "visible_photo_count 必须是 int")

	assert.Equal(t, map[string]string{
		"kindergarten_id":     fmt.Sprint(testPCALKg),
		"class_id":            fmt.Sprint(testPCALClassA),
		"album_id":            fmt.Sprint(testPCALAlbumMain),
		"student_id":          fmt.Sprint(testPCALStudentA),
		"visible_photo_count": fmt.Sprint(visible),
		"role":                "parent",
	}, arFindRecord(t, h, AlbumEventParentView).Attrs)

	// 草稿相册 404：不算一次查看。
	_, err = f.svc.GetClassAlbum(ctx, f.pcA, testPCALAlbumDraft)
	require.ErrorIs(t, err, ErrParentNotFound)
	assert.Len(t, arRecordsWithMsg(h, AlbumEventParentView), 1)
}

// 班内共享真实「开 → 关」才记 album_opt_out：重复关闭与重新开启都不记。
func TestParentSettingsOptOutEmitsMetricEvent(t *testing.T) {
	h := arCaptureLogs(t)
	f := newPCALFixture(t)
	ctx := context.Background()
	setShare := func(v bool) error {
		return f.svc.UpdateSettings(ctx, f.pcA, map[string]any{
			"privacy": map[string]any{"allow_class_share": v},
		})
	}

	// 家长 A 没有设置行 → 默认开启，关闭是一次真实迁移。
	require.NoError(t, setShare(false))
	assert.Equal(t, map[string]string{
		"kindergarten_id":   fmt.Sprint(testPCALKg),
		"user_id":           fmt.Sprint(testPCALParentA),
		"allow_class_share": "false",
		"role":              "parent",
	}, arFindRecord(t, h, AlbumEventOptOut).Attrs)

	// 已经关闭后重复保存：没有真实变化，不再记事件。
	require.NoError(t, setShare(false))
	assert.Len(t, arRecordsWithMsg(h, AlbumEventOptOut), 1)

	// 重新开启不是 opt-out。
	require.NoError(t, setShare(true))
	assert.Len(t, arRecordsWithMsg(h, AlbumEventOptOut), 1)
}
