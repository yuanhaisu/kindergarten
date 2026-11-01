package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
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

// B20：老师端班级相册（openspec class-album）。
// B21：列表 cover_urls 前 3 张 + 单条批量查询守卫。
// OS-ALB-2：发布状态机、统计口径、编辑/软删/审计、照片 file_id 与
// 相册级/照片级幼儿关联、成长记录选图自动带出与显式覆盖。
//
// 打真实 MySQL（本地 configs/app.ini），固定 9960xx 夹具，t.Cleanup 硬删 0 残留。
// 覆盖 TC-NEW-ALB-0001–0017：
//   - 0001–0007 创建/校验/统计/筛选/详情/隔离；
//   - 0008 新建默认草稿；0009 统计只算已发布；0010 发布；0011 下架与非法转换；
//   - 0012 任一老师可操作并写审计；0013 未授权照片计数；0014 编辑重算；
//   - 0015 软删后不可见；0016 成长选图自动带出幼儿；0017 显式覆盖与越权拒绝。
//
// 建表走测试内 AutoMigrate：本切片不允许重启 8080，测试不能依赖后端启动时迁移；
// 与 provider.go 的登记保持同一组模型。

const (
	testTALKG      uint64 = 996001 // 目标园所
	testTALKGOther uint64 = 996002 // 他园

	testTALClassA uint64 = 996010 // 目标班
	testTALClassB uint64 = 996011 // 同园别班
	testTALClassO uint64 = 996012 // 他园班级

	testTALUser    uint64 = 996020 // 相册创建者（夹具统一用它做清理锚点）
	testTALUserTwo uint64 = 996021 // 同班另一位老师
	testTALUserFG  uint64 = 996022 // 他园老师（用于 file_id 越权）
	testTALUserNR  uint64 = 996023 // 无任何角色（用于 file_id 越权）

	testTALStudentA   uint64 = 996030 // 目标班幼儿 A
	testTALStudentB   uint64 = 996031 // 目标班幼儿 B
	testTALStudentOpt uint64 = 996032 // 目标班幼儿 C：家长关闭班内共享
	testTALStudentOut uint64 = 996033 // 别班幼儿（越权参与者）

	testTALFileA       uint64 = 996040 // 老师 1 上传
	testTALFileB       uint64 = 996041 // 同班老师 2 上传
	testTALFileOtherKG uint64 = 996042 // 他园老师上传
	testTALFileNoRole  uint64 = 996043 // 无角色用户上传

	testTALGrowthLog        uint64 = 996050 // 目标班成长记录（关联 A、B）
	testTALGrowthLogClassB  uint64 = 996051 // 别班成长记录
	testTALGrowthMedia      uint64 = 996060 // 目标班成长照片
	testTALGrowthMediaB     uint64 = 996061 // 别班成长照片
	testTALGrowthMediaVideo uint64 = 996062 // 目标班成长视频（相册不收）

	testTALParentA   uint64 = 996070 // A、B 的家长（无设置行 → 默认允许班内共享）
	testTALParentOpt uint64 = 996071 // C 的家长（allow_class_share=false）

	testTALAlbumDeleted uint64 = 996106 // 软删相册
)

func talThisMonthDate(dayOffset int) time.Time {
	now := teacherPortalDay(time.Now())
	return now.AddDate(0, 0, -now.Day()+1+dayOffset)
}

func talLastMonthDate(dayOffset int) time.Time {
	return talThisMonthDate(dayOffset).AddDate(0, -1, 0)
}

func talTodayString() string {
	return talThisMonthDate(0).Format("2006-01-02")
}

func cleanupTALData(t *testing.T, db *gorm.DB) {
	t.Helper()
	del := func(sql string, args ...any) {
		if err := db.Exec(sql, args...).Error; err != nil {
			t.Logf("cleanup: %v", err)
		}
	}
	// 先删关联表：album_photo_students 没有 album_id，只能经照片/相册子查询找回。
	// 服务创建的相册 ID 是自增的，不能只靠固定区间，必须同时按 creator 收敛。
	del(`DELETE FROM album_photo_students WHERE photo_id IN
		(SELECT id FROM album_photos WHERE album_id IN
			(SELECT id FROM albums WHERE creator_id BETWEEN ? AND ? OR id BETWEEN ? AND ?))`,
		testTALUser, testTALUserTwo, uint64(996100), uint64(996199))
	del(`DELETE FROM album_students WHERE album_id IN
		(SELECT id FROM albums WHERE creator_id BETWEEN ? AND ? OR id BETWEEN ? AND ?)`,
		testTALUser, testTALUserTwo, uint64(996100), uint64(996199))
	del(`DELETE FROM album_photos WHERE album_id IN
		(SELECT id FROM albums WHERE creator_id BETWEEN ? AND ? OR id BETWEEN ? AND ?)`,
		testTALUser, testTALUserTwo, uint64(996100), uint64(996199))
	del("DELETE FROM albums WHERE creator_id BETWEEN ? AND ? OR id BETWEEN ? AND ?",
		testTALUser, testTALUserTwo, uint64(996100), uint64(996199))
	del("DELETE FROM operation_logs WHERE user_id BETWEEN ? AND ?", testTALUser, uint64(996029))
	del("DELETE FROM growth_log_students WHERE log_id BETWEEN ? AND ?", testTALGrowthLog, uint64(996059))
	del("DELETE FROM growth_media WHERE id BETWEEN ? AND ?", testTALGrowthMedia, uint64(996069))
	del("DELETE FROM growth_daily_logs WHERE id BETWEEN ? AND ?", testTALGrowthLog, uint64(996059))
	del("DELETE FROM parent_settings WHERE user_id BETWEEN ? AND ?", testTALUser, uint64(996079))
	del("DELETE FROM student_parents WHERE student_id BETWEEN ? AND ?", testTALStudentA, uint64(996039))
	del("DELETE FROM files WHERE id BETWEEN ? AND ?", testTALFileA, uint64(996049))
	del("DELETE FROM user_roles WHERE user_id BETWEEN ? AND ?", testTALUser, uint64(996029))
	del("DELETE FROM students WHERE id BETWEEN ? AND ?", testTALStudentA, uint64(996039))
	del("DELETE FROM classes WHERE id BETWEEN ? AND ?", uint64(996010), uint64(996019))
	del("DELETE FROM users WHERE id BETWEEN ? AND ?", testTALUser, uint64(996079))
	del("DELETE FROM kindergartens WHERE id IN (?, ?)", testTALKG, testTALKGOther)
}

type talFixture struct {
	db      *gorm.DB
	svc     *TeacherPortalService
	tcA     *TeacherPortalContext
	tcB     *TeacherPortalContext
	tcOther *TeacherPortalContext
	tcTwo   *TeacherPortalContext
}

// talMigrateOnce 每个进程只跑一次建表：17 个用例各自 AutoMigrate 13 张表
// 会让整组多花几分钟，且模型结构在测试期间不会变。
var talMigrateOnce sync.Once

func newTALFixture(t *testing.T) *talFixture {
	t.Helper()
	db := openCLPTestDB(t)
	talMigrateOnce.Do(func() {
		require.NoError(t, db.AutoMigrate(
			&domain.Album{}, &domain.AlbumPhoto{}, &domain.AlbumStudent{}, &domain.AlbumPhotoStudent{},
			&domain.Student{}, &domain.StudentParent{}, &domain.ParentSetting{}, &domain.File{},
			&domain.UserRole{}, &domain.GrowthDailyLog{}, &domain.GrowthLogStudent{}, &domain.GrowthMedia{},
			&domain.OperationLog{},
		))
	})
	cleanupTALData(t, db)
	t.Cleanup(func() { cleanupTALData(t, db) })

	require.NoError(t, db.Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: testTALKG}, Name: "测试园所-班级相册",
	}).Error)
	require.NoError(t, db.Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: testTALKGOther}, Name: "测试园所-班级相册他园",
	}).Error)

	for _, cl := range []struct {
		id   uint64
		kg   uint64
		name string
	}{
		{testTALClassA, testTALKG, "相册一班"},
		{testTALClassB, testTALKG, "相册二班"},
		{testTALClassO, testTALKGOther, "他园相册班"},
	} {
		require.NoError(t, db.Create(&domain.Class{
			BaseModel: domain.BaseModel{ID: cl.id}, KindergartenID: cl.kg, Name: cl.name, Grade: "小班", Status: 1,
		}).Error)
	}

	users := []domain.User{
		{BaseModel: domain.BaseModel{ID: testTALUser}, Nickname: "相册老师", Phone: "13900009961", PasswordHash: "x"},
		{BaseModel: domain.BaseModel{ID: testTALUserTwo}, Nickname: "相册老师二", Phone: "13900009962", PasswordHash: "x"},
		{BaseModel: domain.BaseModel{ID: testTALUserFG}, Nickname: "他园老师", Phone: "13900009963", PasswordHash: "x"},
		{BaseModel: domain.BaseModel{ID: testTALUserNR}, Nickname: "无角色用户", Phone: "13900009964", PasswordHash: "x"},
		{BaseModel: domain.BaseModel{ID: testTALParentA}, Nickname: "家长A", Phone: "13900009970", PasswordHash: "x"},
		{BaseModel: domain.BaseModel{ID: testTALParentOpt}, Nickname: "家长C", Phone: "13900009971", PasswordHash: "x"},
	}
	for i := range users {
		require.NoError(t, db.Create(&users[i]).Error)
	}

	kgID, classA, classO := testTALKG, testTALClassA, testTALClassO
	roles := []domain.UserRole{
		{UserID: testTALUser, Role: "teacher", KindergartenID: &kgID, ClassID: &classA},
		{UserID: testTALUserTwo, Role: "teacher", KindergartenID: &kgID, ClassID: &classA},
		{UserID: testTALUserFG, Role: "teacher", KindergartenID: &kgID, ClassID: &classO},
	}
	// 他园老师必须真的挂在测试用的「他园」上，否则 996002 无老师角色。
	otherKG := testTALKGOther
	roles[2].KindergartenID = &otherKG
	roles[2].ClassID = &classO
	for i := range roles {
		require.NoError(t, db.Create(&roles[i]).Error)
	}

	validDate := talThisMonthDate(0)
	for _, st := range []struct {
		id    uint64
		class uint64
		name  string
	}{
		{testTALStudentA, testTALClassA, "幼儿A"},
		{testTALStudentB, testTALClassA, "幼儿B"},
		{testTALStudentOpt, testTALClassA, "幼儿C"},
		{testTALStudentOut, testTALClassB, "别班幼儿"},
	} {
		require.NoError(t, db.Create(&domain.Student{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: st.id}},
			KindergartenID:  testTALKG, ClassID: st.class, Name: st.name, Status: 1,
			Birthday: validDate, EnrollDate: validDate,
		}).Error)
	}

	for _, sp := range []struct {
		student uint64
		parent  uint64
	}{
		{testTALStudentA, testTALParentA},
		{testTALStudentB, testTALParentA},
		{testTALStudentOpt, testTALParentOpt},
	} {
		require.NoError(t, db.Create(&domain.StudentParent{
			StudentID: sp.student, UserID: sp.parent, Relation: domain.RelationMother,
		}).Error)
	}

	// 只有 C 的家长明确关闭班内共享；A、B 的家长没有设置行（默认开启）。
	require.NoError(t, db.Create(&domain.ParentSetting{
		UserID: testTALParentOpt, KindergartenID: testTALKG,
		AllowGrowthShare: false, AllowClassShare: false, AllowExternalShare: false,
		ShowPhoneToTeacher: true,
	}).Error)

	for _, f := range []struct {
		id       uint64
		uploader uint64
		url      string
	}{
		{testTALFileA, testTALUser, "https://cdn.example.com/files/a.jpg"},
		{testTALFileB, testTALUserTwo, "https://cdn.example.com/files/b.jpg"},
		{testTALFileOtherKG, testTALUserFG, "https://cdn.example.com/files/foreign.jpg"},
		{testTALFileNoRole, testTALUserNR, "https://cdn.example.com/files/norole.jpg"},
	} {
		require.NoError(t, db.Create(&domain.File{
			BaseModel: domain.BaseModel{ID: f.id}, UserID: f.uploader, BizType: "growth",
			FileName: fmt.Sprintf("file-%d.jpg", f.id), FileURL: f.url, FileType: "image",
		}).Error)
	}

	logA, classB, classA2, user1 := testTALGrowthLog, testTALClassB, testTALClassA, testTALUser
	require.NoError(t, db.Create(&domain.GrowthDailyLog{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: logA}},
		KindergartenID:  testTALKG, ClassID: testTALClassA, CreatorID: testTALUser,
		Date: validDate, TargetType: 2, Content: "成长记录A",
	}).Error)
	require.NoError(t, db.Create(&domain.GrowthDailyLog{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testTALGrowthLogClassB}},
		KindergartenID:  testTALKG, ClassID: testTALClassB, CreatorID: testTALUser,
		Date: validDate, TargetType: 2, Content: "成长记录B",
	}).Error)
	for _, sid := range []uint64{testTALStudentA, testTALStudentB} {
		require.NoError(t, db.Create(&domain.GrowthLogStudent{LogID: logA, StudentID: sid}).Error)
	}

	require.NoError(t, db.Create(&domain.GrowthMedia{
		BaseModel:      domain.BaseModel{ID: testTALGrowthMedia},
		KindergartenID: testTALKG, ClassID: classA2, LogID: &logA, UploaderID: user1,
		FileURL: "https://cdn.example.com/files/a.jpg", ThumbURL: "https://cdn.example.com/files/a_thumb.jpg",
		FileType: "image", FileName: "a.jpg", SortNum: 0,
	}).Error)
	require.NoError(t, db.Create(&domain.GrowthMedia{
		BaseModel:      domain.BaseModel{ID: testTALGrowthMediaB},
		KindergartenID: testTALKG, ClassID: classB, LogID: ptrUint64(testTALGrowthLogClassB), UploaderID: user1,
		FileURL: "https://cdn.example.com/files/classb.jpg", FileType: "image", SortNum: 0,
	}).Error)
	require.NoError(t, db.Create(&domain.GrowthMedia{
		BaseModel:      domain.BaseModel{ID: testTALGrowthMediaVideo},
		KindergartenID: testTALKG, ClassID: classA2, LogID: &logA, UploaderID: user1,
		FileURL: "https://cdn.example.com/files/a.mp4", FileType: "video", SortNum: 1,
	}).Error)

	return &talFixture{
		db:      db,
		svc:     NewTeacherPortalService(db, nil, nil, nil, nil, nil, config.FaceConfig{}),
		tcA:     &TeacherPortalContext{UserID: testTALUser, KindergartenID: testTALKG, ClassID: testTALClassA},
		tcB:     &TeacherPortalContext{UserID: testTALUser, KindergartenID: testTALKG, ClassID: testTALClassB},
		tcOther: &TeacherPortalContext{UserID: testTALUser, KindergartenID: testTALKGOther, ClassID: testTALClassO},
		tcTwo:   &TeacherPortalContext{UserID: testTALUserTwo, KindergartenID: testTALKG, ClassID: testTALClassA},
	}
}

func ptrUint64(v uint64) *uint64 { return &v }

func createTALAlbum(
	t *testing.T,
	f *talFixture,
	tc *TeacherPortalContext,
	title, category, albumDate string,
	photos []TeacherAlbumPhotoInput,
) (uint64, error) {
	t.Helper()
	data, err := f.svc.CreateAlbum(context.Background(), tc, TeacherAlbumInput{
		Title: title, Category: category, AlbumDate: albumDate, Photos: photos,
	})
	if err != nil {
		return 0, err
	}
	id, ok := data["id"].(uint64)
	require.True(t, ok, "创建响应必须是 {id: uint64}")
	return id, nil
}

func talPhotoInputs(n int) []TeacherAlbumPhotoInput {
	photos := make([]TeacherAlbumPhotoInput, 0, n)
	for i := 0; i < n; i++ {
		photos = append(photos, TeacherAlbumPhotoInput{
			FileURL: fmt.Sprintf("https://cdn.example.com/album/%d.jpg", i),
			Width:   1200, Height: 900,
		})
	}
	return photos
}

func seedTALAlbum(
	t *testing.T,
	db *gorm.DB,
	id, kg, classID uint64,
	category domain.AlbumCategory,
	date time.Time,
	photoCount int,
	status string,
	deleted bool,
) {
	t.Helper()
	album := domain.Album{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: id}},
		KindergartenID:  kg,
		ClassID:         classID,
		CreatorID:       testTALUser,
		Title:           fmt.Sprintf("种子相册-%d", id),
		Category:        category,
		AlbumDate:       date,
		CoverURL:        fmt.Sprintf("https://cdn.example.com/seed/%d/0.jpg", id),
		PhotoCount:      photoCount,
		Status:          status,
	}
	if status == domain.AlbumStatusPublished {
		at := date
		album.PublishedAt = &at
		album.PublishedBy = testTALUser
	}
	if deleted {
		album.DeletedAt = gorm.DeletedAt{Time: time.Now(), Valid: true}
	}
	require.NoError(t, db.Create(&album).Error)

	if photoCount <= 0 {
		return
	}
	photos := make([]domain.AlbumPhoto, 0, photoCount)
	for i := 0; i < photoCount; i++ {
		photos = append(photos, domain.AlbumPhoto{
			AlbumID: album.ID,
			FileURL: fmt.Sprintf("https://cdn.example.com/seed/%d/%d.jpg", id, i),
			Width:   800, Height: 600, SortNum: i,
		})
	}
	require.NoError(t, db.Create(&photos).Error)
}

func talAlbumCount(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Unscoped().Model(&domain.Album{}).Where("creator_id = ?", testTALUser).Count(&n).Error)
	return n
}

func talCount(t *testing.T, db *gorm.DB, model any, query string, args ...any) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Unscoped().Model(model).Where(query, args...).Count(&n).Error)
	return n
}

func talPhotos(t *testing.T, detail map[string]any) []map[string]any {
	t.Helper()
	photos, ok := detail["photos"].([]map[string]any)
	require.True(t, ok, "photos 必须是数组")
	return photos
}

func talStudentIDs(t *testing.T, raw any) []uint64 {
	t.Helper()
	rows, ok := raw.([]map[string]any)
	require.True(t, ok, "students 必须是对象数组")
	ids := make([]uint64, 0, len(rows))
	for _, row := range rows {
		id, ok := row["id"].(uint64)
		require.True(t, ok, "学生 id 必须是 uint64")
		ids = append(ids, id)
	}
	return ids
}

func talAlbumAuditRows(t *testing.T, db *gorm.DB, albumID uint64) []domain.OperationLog {
	t.Helper()
	var logs []domain.OperationLog
	require.NoError(t, db.Where("module = ? AND target_id = ?", "albums", albumID).
		Order("id ASC").Find(&logs).Error)
	return logs
}

// TC-NEW-ALB-0001：创建含 3 张照片的相册，照片按提交顺序保存，封面与计数冗余写入。
func TestTeacherAlbumCreateThreePhotos(t *testing.T) {
	f := newTALFixture(t)
	date := talThisMonthDate(3).Format("2006-01-02")

	id, err := createTALAlbum(t, f, f.tcA, "  秋日郊游  ", "activity", date, []TeacherAlbumPhotoInput{
		{FileURL: "https://cdn.example.com/album/1.jpg", ThumbURL: "https://cdn.example.com/album/1t.jpg", Width: 1200, Height: 900},
		{FileURL: "https://cdn.example.com/album/2.jpg", Width: 800, Height: 600},
		{FileURL: "https://cdn.example.com/album/3.jpg", Width: 1000, Height: 750},
	})
	require.NoError(t, err)
	require.NotZero(t, id)

	var album domain.Album
	require.NoError(t, f.db.First(&album, id).Error)
	assert.Equal(t, "秋日郊游", album.Title, "标题必须去首尾空格")
	assert.Equal(t, domain.AlbumCategoryActivity, album.Category)
	assert.EqualValues(t, testTALKG, album.KindergartenID)
	assert.EqualValues(t, testTALClassA, album.ClassID)
	assert.EqualValues(t, testTALUser, album.CreatorID)
	assert.Equal(t, 3, album.PhotoCount)
	assert.Equal(t, "https://cdn.example.com/album/1.jpg", album.CoverURL, "封面取第一张照片")
	assert.Equal(t, date, teacherAlbumDateString(album.AlbumDate))

	detail, err := f.svc.GetAlbum(context.Background(), f.tcA, id)
	require.NoError(t, err)
	photos := talPhotos(t, detail)
	require.Len(t, photos, 3)
	for i, want := range []string{
		"https://cdn.example.com/album/1.jpg",
		"https://cdn.example.com/album/2.jpg",
		"https://cdn.example.com/album/3.jpg",
	} {
		assert.Equal(t, want, photos[i]["file_url"], "第 %d 张顺序不符", i+1)
	}
	assert.Equal(t, "https://cdn.example.com/album/1t.jpg", photos[0]["thumb_url"])
	assert.Equal(t, 1200, photos[0]["width"])
	assert.Nil(t, photos[0]["file_id"], "旧 URL 路径不产生 file_id")
}

// TC-NEW-ALB-0002：没有照片整单拒绝，不创建相册。
func TestTeacherAlbumCreateRejectsNoPhotos(t *testing.T) {
	f := newTALFixture(t)

	_, err := createTALAlbum(t, f, f.tcA, "空相册", "daily", talTodayString(), nil)
	require.ErrorIs(t, err, ErrTeacherInvalidParams)
	assert.EqualValues(t, 0, talAlbumCount(t, f.db))
}

// 超过 50 张整单拒绝（上限 1–50）。
func TestTeacherAlbumCreateRejectsTooManyPhotos(t *testing.T) {
	f := newTALFixture(t)

	_, err := createTALAlbum(t, f, f.tcA, "照片太多", "daily", talTodayString(), talPhotoInputs(51))
	require.ErrorIs(t, err, ErrTeacherInvalidParams)
	assert.EqualValues(t, 0, talAlbumCount(t, f.db))
}

// TC-NEW-ALB-0007：分类只接受四值。
func TestTeacherAlbumCreateRejectsInvalidCategory(t *testing.T) {
	f := newTALFixture(t)

	for _, category := range []string{"other", "", "Activity", "全部"} {
		_, err := createTALAlbum(t, f, f.tcA, "非法分类", category, talTodayString(), talPhotoInputs(1))
		require.ErrorIs(t, err, ErrTeacherInvalidParams, "category=%q 必须拒绝", category)
	}
	assert.EqualValues(t, 0, talAlbumCount(t, f.db))
}

// 标题 1–30、日期 YYYY-MM-DD、URL http(s)、尺寸非负：逐条拒绝且不落库。
func TestTeacherAlbumCreateValidation(t *testing.T) {
	f := newTALFixture(t)
	date := talTodayString()

	cases := []struct {
		name   string
		mutate func(*TeacherAlbumInput)
	}{
		{"标题为空", func(in *TeacherAlbumInput) { in.Title = "   " }},
		{"标题超 30 字", func(in *TeacherAlbumInput) { in.Title = strings.Repeat("相", 31) }},
		{"日期用斜杠", func(in *TeacherAlbumInput) { in.AlbumDate = "2026/10/24" }},
		{"日期不存在", func(in *TeacherAlbumInput) { in.AlbumDate = "2026-02-30" }},
		{"原图非 http(s)", func(in *TeacherAlbumInput) { in.Photos[0].FileURL = "ftp://cdn.example.com/a.jpg" }},
		{"原图是相对路径", func(in *TeacherAlbumInput) { in.Photos[0].FileURL = "/tmp/upload/a.jpg" }},
		{"原图缺 host", func(in *TeacherAlbumInput) { in.Photos[0].FileURL = "https://" }},
		{"缩略图非法", func(in *TeacherAlbumInput) { in.Photos[0].ThumbURL = "not-a-url" }},
		{"宽度为负", func(in *TeacherAlbumInput) { in.Photos[0].Width = -1 }},
		{"高度为负", func(in *TeacherAlbumInput) { in.Photos[0].Height = -1 }},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// 每个子用例独立构造照片切片：共享底层数组会让上一个子用例的改动漏进来。
			in := TeacherAlbumInput{Title: "合法标题", Category: "activity", AlbumDate: date, Photos: talPhotoInputs(1)}
			c.mutate(&in)
			_, err := f.svc.CreateAlbum(context.Background(), f.tcA, in)
			require.ErrorIs(t, err, ErrTeacherInvalidParams)
		})
	}
	assert.EqualValues(t, 0, talAlbumCount(t, f.db), "非法入参不得落任何相册")
}

// TC-NEW-ALB-0003：统计跨月——本月新增只算 album_date 在本月的照片；
// 软删相册、别班与他园数据都不混入。
func TestTeacherAlbumStatsAcrossMonths(t *testing.T) {
	f := newTALFixture(t)

	seedTALAlbum(t, f.db, 996101, testTALKG, testTALClassA, domain.AlbumCategoryActivity, talThisMonthDate(0), 3, domain.AlbumStatusPublished, false)
	seedTALAlbum(t, f.db, 996102, testTALKG, testTALClassA, domain.AlbumCategoryDaily, talThisMonthDate(1), 2, domain.AlbumStatusPublished, false)
	seedTALAlbum(t, f.db, 996103, testTALKG, testTALClassA, domain.AlbumCategoryActivity, talLastMonthDate(0), 1, domain.AlbumStatusPublished, false)
	seedTALAlbum(t, f.db, 996104, testTALKG, testTALClassB, domain.AlbumCategoryWork, talThisMonthDate(0), 4, domain.AlbumStatusPublished, false)
	seedTALAlbum(t, f.db, 996105, testTALKGOther, testTALClassO, domain.AlbumCategoryFestival, talThisMonthDate(0), 5, domain.AlbumStatusPublished, false)
	seedTALAlbum(t, f.db, testTALAlbumDeleted, testTALKG, testTALClassA, domain.AlbumCategoryActivity, talThisMonthDate(0), 6, domain.AlbumStatusPublished, true)

	statsA, err := f.svc.AlbumStats(context.Background(), f.tcA)
	require.NoError(t, err)
	assert.EqualValues(t, 6, statsA["total_photos"], "本班 3+2+1；软删与别班/他园不计")
	assert.EqualValues(t, 5, statsA["monthly_new"], "只算本月 3+2；上月 1 张不计")
	assert.EqualValues(t, 3, statsA["album_count"])

	statsB, err := f.svc.AlbumStats(context.Background(), f.tcB)
	require.NoError(t, err)
	assert.EqualValues(t, 4, statsB["total_photos"])
	assert.EqualValues(t, 4, statsB["monthly_new"])
	assert.EqualValues(t, 1, statsB["album_count"])

	statsO, err := f.svc.AlbumStats(context.Background(), f.tcOther)
	require.NoError(t, err)
	assert.EqualValues(t, 5, statsO["total_photos"])
	assert.EqualValues(t, 1, statsO["album_count"])
}

// TC-NEW-ALB-0009：统计只算已发布——草稿、已下架与软删都不计入。
func TestTeacherAlbumStatsOnlyPublished(t *testing.T) {
	f := newTALFixture(t)

	seedTALAlbum(t, f.db, 996171, testTALKG, testTALClassA, domain.AlbumCategoryActivity, talThisMonthDate(1), 2, domain.AlbumStatusPublished, false)
	seedTALAlbum(t, f.db, 996172, testTALKG, testTALClassA, domain.AlbumCategoryDaily, talLastMonthDate(1), 3, domain.AlbumStatusPublished, false)
	seedTALAlbum(t, f.db, 996173, testTALKG, testTALClassA, domain.AlbumCategoryWork, talThisMonthDate(2), 4, domain.AlbumStatusDraft, false)
	seedTALAlbum(t, f.db, 996174, testTALKG, testTALClassA, domain.AlbumCategoryFestival, talThisMonthDate(3), 5, domain.AlbumStatusUnpublished, false)
	seedTALAlbum(t, f.db, 996175, testTALKG, testTALClassA, domain.AlbumCategoryActivity, talThisMonthDate(4), 6, domain.AlbumStatusPublished, true)

	stats, err := f.svc.AlbumStats(context.Background(), f.tcA)
	require.NoError(t, err)
	assert.EqualValues(t, 5, stats["total_photos"], "2+3；草稿 4、下架 5、软删 6 都不计")
	assert.EqualValues(t, 2, stats["monthly_new"], "本月只有 2；上月 3 不计")
	assert.EqualValues(t, 2, stats["album_count"])
}

// TC-NEW-ALB-0004：分类筛选只返回该分类，日期倒序；不传返回全部；非法筛选值 422。
func TestTeacherAlbumListFilterByCategory(t *testing.T) {
	f := newTALFixture(t)

	seedTALAlbum(t, f.db, 996111, testTALKG, testTALClassA, domain.AlbumCategoryActivity, talThisMonthDate(3), 1, domain.AlbumStatusDraft, false)
	seedTALAlbum(t, f.db, 996112, testTALKG, testTALClassA, domain.AlbumCategoryActivity, talThisMonthDate(1), 1, domain.AlbumStatusDraft, false)
	seedTALAlbum(t, f.db, 996113, testTALKG, testTALClassA, domain.AlbumCategoryDaily, talThisMonthDate(2), 1, domain.AlbumStatusDraft, false)
	seedTALAlbum(t, f.db, 996114, testTALKG, testTALClassA, domain.AlbumCategoryFestival, talThisMonthDate(0), 1, domain.AlbumStatusDraft, false)
	seedTALAlbum(t, f.db, 996115, testTALKG, testTALClassA, domain.AlbumCategoryWork, talThisMonthDate(4), 1, domain.AlbumStatusDraft, false)
	seedTALAlbum(t, f.db, 996116, testTALKG, testTALClassB, domain.AlbumCategoryActivity, talThisMonthDate(5), 1, domain.AlbumStatusDraft, false)

	data, err := f.svc.ListAlbums(context.Background(), f.tcA, "activity", "", 1, 20)
	require.NoError(t, err)
	assert.EqualValues(t, 2, data["total"])
	list := data["list"].([]map[string]any)
	require.Len(t, list, 2)
	assert.EqualValues(t, 996111, list[0]["id"], "活动分类按日期倒序")
	assert.EqualValues(t, 996112, list[1]["id"])
	assert.Equal(t, "活动", list[0]["category_label"])
	assert.Equal(t, domain.AlbumStatusDraft, list[0]["status"])
	assert.NotContains(t, fmt.Sprint(list), "996116", "别班活动相册不得出现")

	all, err := f.svc.ListAlbums(context.Background(), f.tcA, "", "", 1, 20)
	require.NoError(t, err)
	assert.EqualValues(t, 5, all["total"])
	allList := all["list"].([]map[string]any)
	require.Len(t, allList, 5)
	assert.EqualValues(t, 996115, allList[0]["id"], "全部按日期倒序")
	assert.EqualValues(t, 996114, allList[4]["id"])

	paged, err := f.svc.ListAlbums(context.Background(), f.tcA, "", "", 3, 2)
	require.NoError(t, err)
	assert.EqualValues(t, 5, paged["total"])
	assert.Len(t, paged["list"].([]map[string]any), 1)

	_, err = f.svc.ListAlbums(context.Background(), f.tcA, "other", "", 1, 20)
	require.ErrorIs(t, err, ErrTeacherInvalidParams)
}

// B21：列表每项带 cover_urls（最多 3 张，sort_num 升序），cover_url 保留兼容；
// 无照片（生产不应出现）必须是空数组而不是 null。
func TestTeacherAlbumListCoverURLs(t *testing.T) {
	f := newTALFixture(t)

	seedTALAlbum(t, f.db, 996151, testTALKG, testTALClassA, domain.AlbumCategoryActivity, talThisMonthDate(2), 4, domain.AlbumStatusDraft, false)
	seedTALAlbum(t, f.db, 996152, testTALKG, testTALClassA, domain.AlbumCategoryDaily, talThisMonthDate(1), 1, domain.AlbumStatusDraft, false)
	seedTALAlbum(t, f.db, 996153, testTALKG, testTALClassA, domain.AlbumCategoryWork, talThisMonthDate(0), 0, domain.AlbumStatusDraft, false)

	data, err := f.svc.ListAlbums(context.Background(), f.tcA, "", "", 1, 20)
	require.NoError(t, err)
	list := data["list"].([]map[string]any)
	require.Len(t, list, 3)

	assert.Equal(t, []string{
		"https://cdn.example.com/seed/996151/0.jpg",
		"https://cdn.example.com/seed/996151/1.jpg",
		"https://cdn.example.com/seed/996151/2.jpg",
	}, list[0]["cover_urls"], "4 张只取前 3 张且按 sort_num 升序")
	assert.Equal(t, "https://cdn.example.com/seed/996151/0.jpg", list[0]["cover_url"], "cover_url 保留兼容")
	assert.Equal(t, []string{"https://cdn.example.com/seed/996152/0.jpg"}, list[1]["cover_urls"], "1 张就返回 1 张")
	assert.Equal(t, []string{}, list[2]["cover_urls"], "无照片是空数组，不是 nil")

	// HTTP 形状：null 与 [] 在端上不等价，空相册必须序列化成 []。
	raw, err := json.Marshal(data)
	require.NoError(t, err)
	var shape struct {
		List []map[string]json.RawMessage `json:"list"`
	}
	require.NoError(t, json.Unmarshal(raw, &shape))
	require.Len(t, shape.List, 3)
	assert.JSONEq(t, `["https://cdn.example.com/seed/996151/0.jpg","https://cdn.example.com/seed/996151/1.jpg","https://cdn.example.com/seed/996151/2.jpg"]`,
		string(shape.List[0]["cover_urls"]))
	assert.Equal(t, "[]", string(shape.List[2]["cover_urls"]), "空相册的 cover_urls 必须是 []，不能是 null")
}

// 性能守卫：整页相册只发一条 album_photos 批量查询，禁止逐相册取封面（N+1）。
func TestTeacherAlbumListCoverURLsSingleQuery(t *testing.T) {
	f := newTALFixture(t)
	seedTALAlbum(t, f.db, 996161, testTALKG, testTALClassA, domain.AlbumCategoryActivity, talThisMonthDate(2), 4, domain.AlbumStatusDraft, false)
	seedTALAlbum(t, f.db, 996162, testTALKG, testTALClassA, domain.AlbumCategoryDaily, talThisMonthDate(1), 4, domain.AlbumStatusDraft, false)
	seedTALAlbum(t, f.db, 996163, testTALKG, testTALClassA, domain.AlbumCategoryWork, talThisMonthDate(0), 4, domain.AlbumStatusDraft, false)

	counter := &clpSQLLogger{Interface: gormlogger.Default.LogMode(gormlogger.Silent)}
	svc := NewTeacherPortalService(f.db.Session(&gorm.Session{Logger: counter}), nil, nil, nil, nil, nil, config.FaceConfig{})
	_, err := svc.ListAlbums(context.Background(), f.tcA, "", "", 1, 20)
	require.NoError(t, err)

	assert.Equal(t, 1, counter.countContaining("album_photos"), "3 个相册只能有一条批量查询")
}

// TC-NEW-ALB-0005：详情返回全部照片，顺序与创建时一致。
func TestTeacherAlbumDetailPhotoOrder(t *testing.T) {
	f := newTALFixture(t)
	seedTALAlbum(t, f.db, 996121, testTALKG, testTALClassA, domain.AlbumCategoryWork, talThisMonthDate(2), 12, domain.AlbumStatusDraft, false)

	detail, err := f.svc.GetAlbum(context.Background(), f.tcA, 996121)
	require.NoError(t, err)
	photos := talPhotos(t, detail)
	require.Len(t, photos, 12)
	for i, p := range photos {
		assert.Equal(t, fmt.Sprintf("https://cdn.example.com/seed/996121/%d.jpg", i), p["file_url"], "第 %d 张顺序不符", i+1)
	}
	assert.EqualValues(t, 12, detail["photo_count"])
	assert.Equal(t, "作品", detail["category_label"])
	assert.Equal(t, []uint64{}, talStudentIDs(t, detail["album_students"]), "无参与者是空数组")

	_, err = f.svc.GetAlbum(context.Background(), f.tcA, 996129)
	require.ErrorIs(t, err, ErrTeacherNotFound, "不存在的相册返回 404 语义")
}

// TC-NEW-ALB-0006：班级与园所隔离——别班/他园读不到，创建只能落在当前班。
func TestTeacherAlbumIsolation(t *testing.T) {
	f := newTALFixture(t)
	seedTALAlbum(t, f.db, 996131, testTALKG, testTALClassA, domain.AlbumCategoryActivity, talThisMonthDate(0), 2, domain.AlbumStatusDraft, false)

	_, err := f.svc.GetAlbum(context.Background(), f.tcB, 996131)
	require.ErrorIs(t, err, ErrTeacherNotFound, "同园别班读取必须 404")

	_, err = f.svc.GetAlbum(context.Background(), f.tcOther, 996131)
	require.ErrorIs(t, err, ErrTeacherNotFound, "他园读取必须 404")

	listB, err := f.svc.ListAlbums(context.Background(), f.tcB, "", "", 1, 20)
	require.NoError(t, err)
	assert.EqualValues(t, 0, listB["total"])

	statsB, err := f.svc.AlbumStats(context.Background(), f.tcB)
	require.NoError(t, err)
	assert.EqualValues(t, 0, statsB["total_photos"])
	assert.EqualValues(t, 0, statsB["album_count"])

	// 创建不接受班级参数，只能落在当前老师班级。
	idB, err := createTALAlbum(t, f, f.tcB, "二班相册", "daily", talTodayString(), talPhotoInputs(1))
	require.NoError(t, err)
	var albumB domain.Album
	require.NoError(t, f.db.First(&albumB, idB).Error)
	assert.EqualValues(t, testTALKG, albumB.KindergartenID)
	assert.EqualValues(t, testTALClassB, albumB.ClassID)

	// 别班创建后，目标班仍看不到它。
	listA, err := f.svc.ListAlbums(context.Background(), f.tcA, "", "", 1, 20)
	require.NoError(t, err)
	assert.EqualValues(t, 1, listA["total"], "一班只看到自己的相册")

	// 生命周期端点同样按班级隔离：别班/他园操作按 404 处理。
	_, err = f.svc.PublishAlbum(context.Background(), f.tcB, 996131)
	require.ErrorIs(t, err, ErrTeacherNotFound, "别班老师发布必须 404")
	_, err = f.svc.UnpublishAlbum(context.Background(), f.tcOther, 996131)
	require.ErrorIs(t, err, ErrTeacherNotFound, "他园老师下架必须 404")
	_, err = f.svc.DeleteAlbum(context.Background(), f.tcB, 996131)
	require.ErrorIs(t, err, ErrTeacherNotFound, "别班老师删除必须 404")
}

// TC-NEW-ALB-0008：新建相册默认草稿，家长视图（只查 published）不可见。
func TestTeacherAlbumCreateDefaultsToDraft(t *testing.T) {
	f := newTALFixture(t)

	id, err := createTALAlbum(t, f, f.tcA, "草稿相册", "daily", talTodayString(), talPhotoInputs(2))
	require.NoError(t, err)

	var album domain.Album
	require.NoError(t, f.db.First(&album, id).Error)
	assert.Equal(t, domain.AlbumStatusDraft, album.Status)
	assert.Nil(t, album.PublishedAt)
	assert.Zero(t, album.PublishedBy)

	detail, err := f.svc.GetAlbum(context.Background(), f.tcA, id)
	require.NoError(t, err)
	assert.Equal(t, domain.AlbumStatusDraft, detail["status"])
	assert.Nil(t, detail["published_at"])
	assert.EqualValues(t, 0, detail["published_by"])

	pub, err := f.svc.ListAlbums(context.Background(), f.tcA, "", domain.AlbumStatusPublished, 1, 20)
	require.NoError(t, err)
	assert.EqualValues(t, 0, pub["total"], "草稿不得出现在已发布视图")

	stats, err := f.svc.AlbumStats(context.Background(), f.tcA)
	require.NoError(t, err)
	assert.EqualValues(t, 0, stats["album_count"])

	drafts, err := f.svc.ListAlbums(context.Background(), f.tcA, "", domain.AlbumStatusDraft, 1, 20)
	require.NoError(t, err)
	assert.EqualValues(t, 1, drafts["total"])

	_, err = f.svc.ListAlbums(context.Background(), f.tcA, "", "pending", 1, 20)
	require.ErrorIs(t, err, ErrTeacherInvalidParams, "非法状态筛选 422")
}

// TC-NEW-ALB-0010：发布草稿后状态/发布人/时间落库且对家长视图可见；
// 已发布不可重复发布；无照片不能发布。
func TestTeacherAlbumPublish(t *testing.T) {
	f := newTALFixture(t)
	id, err := createTALAlbum(t, f, f.tcA, "待发布相册", "activity", talTodayString(), talPhotoInputs(3))
	require.NoError(t, err)

	data, err := f.svc.PublishAlbum(context.Background(), f.tcA, id)
	require.NoError(t, err)
	assert.Equal(t, domain.AlbumStatusPublished, data["status"])

	var album domain.Album
	require.NoError(t, f.db.First(&album, id).Error)
	assert.Equal(t, domain.AlbumStatusPublished, album.Status)
	require.NotNil(t, album.PublishedAt)
	assert.EqualValues(t, testTALUser, album.PublishedBy)

	detail, err := f.svc.GetAlbum(context.Background(), f.tcA, id)
	require.NoError(t, err)
	assert.Equal(t, domain.AlbumStatusPublished, detail["status"])
	assert.NotNil(t, detail["published_at"])

	pub, err := f.svc.ListAlbums(context.Background(), f.tcA, "", domain.AlbumStatusPublished, 1, 20)
	require.NoError(t, err)
	assert.EqualValues(t, 1, pub["total"])
	assert.EqualValues(t, id, pub["list"].([]map[string]any)[0]["id"])

	_, err = f.svc.PublishAlbum(context.Background(), f.tcA, id)
	require.ErrorIs(t, err, ErrTeacherInvalidParams, "已发布再次发布必须 422")

	// 0 张照片的相册不能发布（正常创建路径不允许，这里直接造数据守门）。
	seedTALAlbum(t, f.db, 996176, testTALKG, testTALClassA, domain.AlbumCategoryDaily, talThisMonthDate(0), 0, domain.AlbumStatusDraft, false)
	_, err = f.svc.PublishAlbum(context.Background(), f.tcA, 996176)
	require.ErrorIs(t, err, ErrTeacherInvalidParams)
}

// TC-NEW-ALB-0011：下架后家长视图立即不可见；非法状态转换 422；下架后可重新上架。
func TestTeacherAlbumUnpublishAndInvalidTransitions(t *testing.T) {
	f := newTALFixture(t)
	id, err := createTALAlbum(t, f, f.tcA, "下架测试", "daily", talTodayString(), talPhotoInputs(1))
	require.NoError(t, err)

	_, err = f.svc.UnpublishAlbum(context.Background(), f.tcA, id)
	require.ErrorIs(t, err, ErrTeacherInvalidParams, "草稿不能下架")

	_, err = f.svc.PublishAlbum(context.Background(), f.tcA, id)
	require.NoError(t, err)

	data, err := f.svc.UnpublishAlbum(context.Background(), f.tcA, id)
	require.NoError(t, err)
	assert.Equal(t, domain.AlbumStatusUnpublished, data["status"])

	var album domain.Album
	require.NoError(t, f.db.First(&album, id).Error)
	assert.Equal(t, domain.AlbumStatusUnpublished, album.Status)
	assert.Nil(t, album.PublishedAt, "下架后清空发布元数据")
	assert.Zero(t, album.PublishedBy)

	pub, err := f.svc.ListAlbums(context.Background(), f.tcA, "", domain.AlbumStatusPublished, 1, 20)
	require.NoError(t, err)
	assert.EqualValues(t, 0, pub["total"], "下架后家长视图不可见")

	unpub, err := f.svc.ListAlbums(context.Background(), f.tcA, "", domain.AlbumStatusUnpublished, 1, 20)
	require.NoError(t, err)
	assert.EqualValues(t, 1, unpub["total"])

	_, err = f.svc.UnpublishAlbum(context.Background(), f.tcA, id)
	require.ErrorIs(t, err, ErrTeacherInvalidParams, "已下架不能再次下架")

	// 下架后重新上架是合法转换（发布流程的可逆路径）。
	_, err = f.svc.PublishAlbum(context.Background(), f.tcA, id)
	require.NoError(t, err)
	require.NoError(t, f.db.First(&album, id).Error)
	assert.Equal(t, domain.AlbumStatusPublished, album.Status)
}

// TC-NEW-ALB-0012：同班任一老师可发布/编辑，审计记录操作人与动作。
func TestTeacherAlbumAnyTeacherCanOperateAndAudit(t *testing.T) {
	f := newTALFixture(t)
	id, err := createTALAlbum(t, f, f.tcA, "审计相册", "activity", talTodayString(), talPhotoInputs(1))
	require.NoError(t, err)

	_, err = f.svc.PublishAlbum(context.Background(), f.tcTwo, id)
	require.NoError(t, err, "同班另一位老师可发布")

	detail, err := f.svc.GetAlbum(context.Background(), f.tcA, id)
	require.NoError(t, err)
	photoID := talPhotos(t, detail)[0]["id"].(uint64)
	photoURL := talPhotos(t, detail)[0]["file_url"].(string)

	_, err = f.svc.UnpublishAlbum(context.Background(), f.tcA, id)
	require.NoError(t, err)

	_, err = f.svc.UpdateAlbum(context.Background(), f.tcTwo, id, TeacherAlbumInput{
		Title: "审计改名", Category: "activity", AlbumDate: talTodayString(),
		Photos: []TeacherAlbumPhotoInput{{ID: photoID, FileURL: photoURL}},
	})
	require.NoError(t, err)

	_, err = f.svc.DeleteAlbum(context.Background(), f.tcA, id)
	require.NoError(t, err)

	logs := talAlbumAuditRows(t, f.db, id)
	require.Len(t, logs, 4)
	wantActions := []string{"publish", "unpublish", "edit", "delete"}
	wantUsers := []uint64{testTALUserTwo, testTALUser, testTALUserTwo, testTALUser}
	for i, log := range logs {
		assert.Equal(t, "albums", log.Module)
		assert.Equal(t, wantActions[i], log.Action)
		assert.Equal(t, "album", log.TargetType)
		assert.EqualValues(t, id, log.TargetID)
		assert.EqualValues(t, wantUsers[i], log.UserID)
		assert.EqualValues(t, testTALKG, log.KindergartenID)
		assert.Contains(t, log.Detail, "title=")
		assert.Contains(t, log.Detail, "status=")
	}
	assert.Contains(t, logs[0].Detail, "title=审计相册")
	assert.Contains(t, logs[2].Detail, "title=审计改名")
}

// TC-NEW-ALB-0013：发布预览的未授权照片计数——opt-out 幼儿计入，无 opt-out 为 0；
// 照片没有逐张关联时按相册级参与者兜底。
func TestTeacherAlbumUnauthorizedPhotoCount(t *testing.T) {
	f := newTALFixture(t)
	ctx := context.Background()

	// 有 opt-out：一张只含 C 的照片应计入。
	id1, err := f.svc.CreateAlbum(ctx, f.tcA, TeacherAlbumInput{
		Title: "含未授权", Category: "activity", AlbumDate: talTodayString(),
		Participants: []uint64{testTALStudentA},
		Photos: []TeacherAlbumPhotoInput{
			{FileURL: "https://cdn.example.com/u/1.jpg", StudentIDs: []uint64{testTALStudentA}},
			{FileURL: "https://cdn.example.com/u/2.jpg", StudentIDs: []uint64{testTALStudentOpt}},
		},
	})
	require.NoError(t, err)
	detail, err := f.svc.GetAlbum(ctx, f.tcA, id1["id"].(uint64))
	require.NoError(t, err)
	assert.EqualValues(t, 1, detail["unauthorized_photo_count"])

	// 无 opt-out：全部 0。
	id2, err := f.svc.CreateAlbum(ctx, f.tcA, TeacherAlbumInput{
		Title: "全部授权", Category: "daily", AlbumDate: talTodayString(),
		Participants: []uint64{testTALStudentA, testTALStudentB},
		Photos:       talPhotoInputs(2),
	})
	require.NoError(t, err)
	detail, err = f.svc.GetAlbum(ctx, f.tcA, id2["id"].(uint64))
	require.NoError(t, err)
	assert.EqualValues(t, 0, detail["unauthorized_photo_count"])

	// 相册级兜底：照片没有逐张关联，但参与幼儿里有 opt-out 的 C。
	id3, err := f.svc.CreateAlbum(ctx, f.tcA, TeacherAlbumInput{
		Title: "兜底口径", Category: "work", AlbumDate: talTodayString(),
		Participants: []uint64{testTALStudentOpt},
		Photos:       talPhotoInputs(1),
	})
	require.NoError(t, err)
	detail, err = f.svc.GetAlbum(ctx, f.tcA, id3["id"].(uint64))
	require.NoError(t, err)
	assert.EqualValues(t, 1, detail["unauthorized_photo_count"])
}

// 多监护人口径：任一监护人关闭班内共享即视为该幼儿 opt-out（隐私优先），
// 另一位监护人没有设置行（默认开启）不能把开关翻回来。
func TestTeacherAlbumUnauthorizedCountAnyGuardianOptOut(t *testing.T) {
	f := newTALFixture(t)
	ctx := context.Background()

	// A 的第二位监护人：显式关闭班内共享。A 的第一位监护人（fixture 996070）无设置行。
	const secondGuardian uint64 = 996072
	require.NoError(t, f.db.Create(&domain.User{
		BaseModel: domain.BaseModel{ID: secondGuardian}, Nickname: "家长A2", Phone: "13900009972", PasswordHash: "x",
	}).Error)
	require.NoError(t, f.db.Create(&domain.ParentSetting{
		UserID: secondGuardian, KindergartenID: testTALKG,
		AllowGrowthShare: false, AllowClassShare: false, AllowExternalShare: false,
		ShowPhoneToTeacher: true,
	}).Error)
	require.NoError(t, f.db.Create(&domain.StudentParent{
		StudentID: testTALStudentA, UserID: secondGuardian, Relation: domain.RelationFather,
	}).Error)

	data, err := f.svc.CreateAlbum(ctx, f.tcA, TeacherAlbumInput{
		Title: "多监护人", Category: "activity", AlbumDate: talTodayString(),
		Photos: []TeacherAlbumPhotoInput{
			{FileURL: "https://cdn.example.com/mg/1.jpg", StudentIDs: []uint64{testTALStudentA}},
			{FileURL: "https://cdn.example.com/mg/2.jpg", StudentIDs: []uint64{testTALStudentB}},
		},
	})
	require.NoError(t, err)

	detail, err := f.svc.GetAlbum(ctx, f.tcA, data["id"].(uint64))
	require.NoError(t, err)
	assert.EqualValues(t, 1, detail["unauthorized_photo_count"],
		"A 的两位监护人一位关闭即 opt-out；B 的监护人未关闭不计")
}

// TC-NEW-ALB-0014：编辑已发布相册立即生效；照片 ID 可保留、未提交的照片硬删；
// photo_count/cover_url 重算；重复提交幂等；引用别班照片 ID 被拒。
func TestTeacherAlbumEditRecalculatesAndKeepsPhotoIDs(t *testing.T) {
	f := newTALFixture(t)
	ctx := context.Background()

	id, err := f.svc.CreateAlbum(ctx, f.tcA, TeacherAlbumInput{
		Title: "原始标题", Category: "activity", AlbumDate: talTodayString(),
		Participants: []uint64{testTALStudentA, testTALStudentB},
		Photos: []TeacherAlbumPhotoInput{
			{FileURL: "https://cdn.example.com/edit/1.jpg", Width: 100, Height: 80},
			{FileURL: "https://cdn.example.com/edit/2.jpg", Width: 200, Height: 160},
		},
	})
	require.NoError(t, err)
	albumID := id["id"].(uint64)
	_, err = f.svc.PublishAlbum(ctx, f.tcA, albumID)
	require.NoError(t, err)

	detail, err := f.svc.GetAlbum(ctx, f.tcA, albumID)
	require.NoError(t, err)
	photos := talPhotos(t, detail)
	require.Len(t, photos, 2)
	keepID := photos[0]["id"].(uint64)
	removedID := photos[1]["id"].(uint64)

	edit := TeacherAlbumInput{
		Title: "编辑后的标题", Category: "work", AlbumDate: talTodayString(),
		Participants: []uint64{testTALStudentA},
		Photos: []TeacherAlbumPhotoInput{
			{ID: keepID, FileURL: "https://cdn.example.com/edit/1.jpg", StudentIDs: []uint64{testTALStudentB}},
			{FileURL: "https://cdn.example.com/edit/3.jpg", StudentIDs: []uint64{testTALStudentA}},
		},
	}
	_, err = f.svc.UpdateAlbum(ctx, f.tcA, albumID, edit)
	require.NoError(t, err)

	var album domain.Album
	require.NoError(t, f.db.First(&album, albumID).Error)
	assert.Equal(t, "编辑后的标题", album.Title)
	assert.Equal(t, domain.AlbumCategoryWork, album.Category)
	assert.Equal(t, 2, album.PhotoCount, "photo_count 必须重算")
	assert.Equal(t, "https://cdn.example.com/edit/1.jpg", album.CoverURL, "cover_url 取第一张")
	assert.Equal(t, domain.AlbumStatusPublished, album.Status, "编辑不改变状态")

	detail, err = f.svc.GetAlbum(ctx, f.tcA, albumID)
	require.NoError(t, err)
	photos = talPhotos(t, detail)
	require.Len(t, photos, 2)
	assert.EqualValues(t, keepID, photos[0]["id"], "提交了 id 的照片保留原 ID")
	assert.Equal(t, []uint64{testTALStudentB}, talStudentIDs(t, photos[0]["students"]))
	assert.Equal(t, "https://cdn.example.com/edit/3.jpg", photos[1]["file_url"])
	assert.Equal(t, []uint64{testTALStudentA}, talStudentIDs(t, detail["album_students"]))

	assert.EqualValues(t, 2, talCount(t, f.db, &domain.AlbumPhoto{}, "album_id = ?", albumID))
	assert.EqualValues(t, 0, talCount(t, f.db, &domain.AlbumPhoto{}, "id = ?", removedID), "未提交的照片硬删")
	assert.EqualValues(t, 0, talCount(t, f.db, &domain.AlbumPhotoStudent{}, "photo_id = ?", removedID), "被删照片的关联一并清理")

	// 重复提交同一编辑（幂等）：唯一键上不做软删占位，不应报错或翻倍。
	detail, err = f.svc.GetAlbum(ctx, f.tcA, albumID)
	require.NoError(t, err)
	photos = talPhotos(t, detail)
	require.Len(t, photos, 2)
	editWithIDs := edit
	editWithIDs.Photos = []TeacherAlbumPhotoInput{
		{ID: photos[0]["id"].(uint64), FileURL: "https://cdn.example.com/edit/1.jpg", StudentIDs: []uint64{testTALStudentB}},
		{ID: photos[1]["id"].(uint64), FileURL: "https://cdn.example.com/edit/3.jpg", StudentIDs: []uint64{testTALStudentA}},
	}
	_, err = f.svc.UpdateAlbum(ctx, f.tcA, albumID, editWithIDs)
	require.NoError(t, err)
	_, err = f.svc.UpdateAlbum(ctx, f.tcA, albumID, editWithIDs)
	require.NoError(t, err)
	assert.EqualValues(t, 2, talCount(t, f.db, &domain.AlbumPhoto{}, "album_id = ?", albumID))
	assert.EqualValues(t, 1, talCount(t, f.db, &domain.AlbumStudent{}, "album_id = ?", albumID))
	assert.EqualValues(t, 2, talCount(t, f.db, &domain.AlbumPhotoStudent{}, "photo_id IN ?",
		[]uint64{photos[0]["id"].(uint64), photos[1]["id"].(uint64)}), "重复提交不得让关联翻倍")

	// 引用其他相册的照片 ID：整单拒绝，不误删本相册照片。
	other, err := f.svc.CreateAlbum(ctx, f.tcA, TeacherAlbumInput{
		Title: "别的相册", Category: "daily", AlbumDate: talTodayString(), Photos: talPhotoInputs(1),
	})
	require.NoError(t, err)
	otherDetail, err := f.svc.GetAlbum(ctx, f.tcA, other["id"].(uint64))
	require.NoError(t, err)
	foreignPhotoID := talPhotos(t, otherDetail)[0]["id"].(uint64)
	_, err = f.svc.UpdateAlbum(ctx, f.tcA, albumID, TeacherAlbumInput{
		Title: "越权编辑", Category: "work", AlbumDate: talTodayString(),
		Photos: []TeacherAlbumPhotoInput{{ID: foreignPhotoID, FileURL: "https://cdn.example.com/edit/1.jpg"}},
	})
	require.ErrorIs(t, err, ErrTeacherInvalidParams)
	require.NoError(t, f.db.First(&album, albumID).Error)
	assert.Equal(t, "编辑后的标题", album.Title, "拒绝后相册不变")
}

// TC-NEW-ALB-0015：软删后老师详情 404、家长视图不可见、统计清零；重复删除 404。
func TestTeacherAlbumSoftDeleteHidesEverywhere(t *testing.T) {
	f := newTALFixture(t)
	ctx := context.Background()

	id, err := createTALAlbum(t, f, f.tcA, "待删除相册", "daily", talTodayString(), talPhotoInputs(2))
	require.NoError(t, err)
	_, err = f.svc.PublishAlbum(ctx, f.tcA, id)
	require.NoError(t, err)

	data, err := f.svc.DeleteAlbum(ctx, f.tcA, id)
	require.NoError(t, err)
	assert.EqualValues(t, id, data["id"])

	_, err = f.svc.GetAlbum(ctx, f.tcA, id)
	require.ErrorIs(t, err, ErrTeacherNotFound)

	pub, err := f.svc.ListAlbums(ctx, f.tcA, "", domain.AlbumStatusPublished, 1, 20)
	require.NoError(t, err)
	assert.EqualValues(t, 0, pub["total"])

	stats, err := f.svc.AlbumStats(ctx, f.tcA)
	require.NoError(t, err)
	assert.EqualValues(t, 0, stats["album_count"])
	assert.EqualValues(t, 0, stats["total_photos"])

	var album domain.Album
	require.NoError(t, f.db.Unscoped().First(&album, id).Error)
	assert.True(t, album.DeletedAt.Valid, "软删保留行供 180 天审计")

	_, err = f.svc.DeleteAlbum(ctx, f.tcA, id)
	require.ErrorIs(t, err, ErrTeacherNotFound)
	photosBefore := talCount(t, f.db, &domain.AlbumPhoto{}, "album_id = ?", id)
	require.EqualValues(t, 2, photosBefore)
	_, err = f.svc.UpdateAlbum(ctx, f.tcA, id, TeacherAlbumInput{Title: "x", Category: "daily", AlbumDate: talTodayString(), Photos: talPhotoInputs(1)})
	require.ErrorIs(t, err, ErrTeacherNotFound)
	assert.EqualValues(t, photosBefore, talCount(t, f.db, &domain.AlbumPhoto{}, "album_id = ?", id), "404 后不得再写照片")
	_, err = f.svc.PublishAlbum(ctx, f.tcA, id)
	require.ErrorIs(t, err, ErrTeacherNotFound)
}

// 仓库层守卫：相册在读到之后被并发软删/移除时，Update 必须整单失败，
// 不能把照片与关联写到孤儿相册上。
func TestTeacherAlbumRepoUpdateRejectsDeletedAlbum(t *testing.T) {
	f := newTALFixture(t)
	ctx := context.Background()
	repo := repository.NewAlbumRepo(f.db)

	id, err := createTALAlbum(t, f, f.tcA, "并发删除", "daily", talTodayString(), talPhotoInputs(2))
	require.NoError(t, err)
	_, err = f.svc.DeleteAlbum(ctx, f.tcA, id)
	require.NoError(t, err)

	deleted := &domain.Album{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: id}},
		KindergartenID:  testTALKG, ClassID: testTALClassA,
		Title: "迟到编辑", Category: domain.AlbumCategoryDaily, AlbumDate: talThisMonthDate(0),
		CoverURL: "https://cdn.example.com/late/1.jpg", PhotoCount: 1,
	}
	err = repo.Update(ctx, deleted,
		[]domain.AlbumPhoto{{FileURL: "https://cdn.example.com/late/1.jpg", Width: 1, Height: 1}},
		[]uint64{testTALStudentA},
		[][]uint64{{testTALStudentA}})
	require.ErrorIs(t, err, repository.ErrAlbumNotFound, "已软删相册必须拒绝")
	assert.EqualValues(t, 2, talCount(t, f.db, &domain.AlbumPhoto{}, "album_id = ?", id), "事务回滚，不得新增照片")
	assert.EqualValues(t, 0, talCount(t, f.db, &domain.AlbumStudent{}, "album_id = ?", id))

	// 完全不存在的相册同样走哨兵错误。
	missing := &domain.Album{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: 996199}},
		KindergartenID:  testTALKG, ClassID: testTALClassA,
		Title: "不存在", Category: domain.AlbumCategoryDaily, AlbumDate: talThisMonthDate(0),
		CoverURL: "https://cdn.example.com/missing/1.jpg", PhotoCount: 1,
	}
	err = repo.Update(ctx, missing,
		[]domain.AlbumPhoto{{FileURL: "https://cdn.example.com/missing/1.jpg"}}, nil, nil)
	require.ErrorIs(t, err, repository.ErrAlbumNotFound)
	assert.EqualValues(t, 0, talCount(t, f.db, &domain.AlbumPhoto{}, "album_id = ?", uint64(996199)))
}

// TC-NEW-ALB-0016：从成长记录选图自动带出该记录关联的幼儿，并回填 file_id。
func TestTeacherAlbumGrowthMediaAutoLinksStudents(t *testing.T) {
	f := newTALFixture(t)
	ctx := context.Background()

	data, err := f.svc.CreateAlbum(ctx, f.tcA, TeacherAlbumInput{
		Title: "成长选图", Category: "activity", AlbumDate: talTodayString(),
		Photos: []TeacherAlbumPhotoInput{{SourceMediaID: testTALGrowthMedia}},
	})
	require.NoError(t, err)

	detail, err := f.svc.GetAlbum(ctx, f.tcA, data["id"].(uint64))
	require.NoError(t, err)
	photos := talPhotos(t, detail)
	require.Len(t, photos, 1)
	assert.Equal(t, "https://cdn.example.com/files/a.jpg", photos[0]["file_url"], "URL 以 growth_media 为准")
	assert.Equal(t, "https://cdn.example.com/files/a_thumb.jpg", photos[0]["thumb_url"])
	assert.EqualValues(t, testTALFileA, photos[0]["file_id"], "按 URL 反查 files 回填 file_id")
	assert.Equal(t, []uint64{testTALStudentA, testTALStudentB}, talStudentIDs(t, photos[0]["students"]),
		"成长记录关联的 A/B 自动带出")

	assert.EqualValues(t, 2, talCount(t, f.db, &domain.AlbumPhotoStudent{}, "photo_id = ?", photos[0]["id"]))
}

// TC-NEW-ALB-0017：显式 student_ids 覆盖自动带出；参与者/照片级关联回读；
// 越权 file_id、越权媒体、视频媒体与跨班幼儿全部拒绝。
func TestTeacherAlbumExplicitStudentsOverrideAndRejectForeignInputs(t *testing.T) {
	f := newTALFixture(t)
	ctx := context.Background()

	// 显式覆盖：成长记录带 A/B，显式只保留 A。
	data, err := f.svc.CreateAlbum(ctx, f.tcA, TeacherAlbumInput{
		Title: "显式覆盖", Category: "activity", AlbumDate: talTodayString(),
		Photos: []TeacherAlbumPhotoInput{{SourceMediaID: testTALGrowthMedia, StudentIDs: []uint64{testTALStudentA}}},
	})
	require.NoError(t, err)
	detail, err := f.svc.GetAlbum(ctx, f.tcA, data["id"].(uint64))
	require.NoError(t, err)
	photos := talPhotos(t, detail)
	assert.Equal(t, []uint64{testTALStudentA}, talStudentIDs(t, photos[0]["students"]), "显式值覆盖自动值")
	assert.EqualValues(t, 1, talCount(t, f.db, &domain.AlbumPhotoStudent{}, "photo_id = ?", photos[0]["id"]))

	// 相册级 + 照片级关联写入与回读（含显式空数组 = 不关联任何幼儿）。
	data, err = f.svc.CreateAlbum(ctx, f.tcA, TeacherAlbumInput{
		Title: "关联回读", Category: "daily", AlbumDate: talTodayString(),
		Participants: []uint64{testTALStudentA, testTALStudentB},
		Photos: []TeacherAlbumPhotoInput{
			{FileURL: "https://cdn.example.com/rel/1.jpg", StudentIDs: []uint64{testTALStudentA}},
			{FileURL: "https://cdn.example.com/rel/2.jpg", StudentIDs: []uint64{}},
		},
	})
	require.NoError(t, err)
	detail, err = f.svc.GetAlbum(ctx, f.tcA, data["id"].(uint64))
	require.NoError(t, err)
	photos = talPhotos(t, detail)
	assert.Equal(t, []uint64{testTALStudentA, testTALStudentB}, talStudentIDs(t, detail["album_students"]))
	assert.Equal(t, []uint64{testTALStudentA}, talStudentIDs(t, photos[0]["students"]))
	assert.Equal(t, []uint64{}, talStudentIDs(t, photos[1]["students"]))

	// file_id 同园其他老师的文件可用；URL 以 files 为准，不接受端上改写。
	okData, err := f.svc.CreateAlbum(ctx, f.tcA, TeacherAlbumInput{
		Title: "文件行", Category: "work", AlbumDate: talTodayString(),
		Photos: []TeacherAlbumPhotoInput{{FileID: testTALFileB, FileURL: "https://evil.example.com/x.jpg"}},
	})
	require.NoError(t, err)
	okDetail, err := f.svc.GetAlbum(ctx, f.tcA, okData["id"].(uint64))
	require.NoError(t, err)
	okPhoto := talPhotos(t, okDetail)[0]
	assert.EqualValues(t, testTALFileB, okPhoto["file_id"])
	assert.Equal(t, "https://cdn.example.com/files/b.jpg", okPhoto["file_url"])

	rejectCases := []struct {
		name   string
		mutate func(*TeacherAlbumInput)
	}{
		{"file_id 不存在", func(in *TeacherAlbumInput) { in.Photos[0] = TeacherAlbumPhotoInput{FileID: 999999} }},
		{"file_id 属他园老师", func(in *TeacherAlbumInput) { in.Photos[0] = TeacherAlbumPhotoInput{FileID: testTALFileOtherKG} }},
		{"file_id 上传者无角色", func(in *TeacherAlbumInput) { in.Photos[0] = TeacherAlbumPhotoInput{FileID: testTALFileNoRole} }},
		{"媒体属别班", func(in *TeacherAlbumInput) { in.Photos[0] = TeacherAlbumPhotoInput{SourceMediaID: testTALGrowthMediaB} }},
		{"媒体是视频", func(in *TeacherAlbumInput) {
			in.Photos[0] = TeacherAlbumPhotoInput{SourceMediaID: testTALGrowthMediaVideo}
		}},
		{"媒体不存在", func(in *TeacherAlbumInput) { in.Photos[0] = TeacherAlbumPhotoInput{SourceMediaID: 999999} }},
		{"参与者属别班", func(in *TeacherAlbumInput) { in.Participants = []uint64{testTALStudentOut} }},
		{"照片幼儿属别班", func(in *TeacherAlbumInput) {
			in.Photos[0] = TeacherAlbumPhotoInput{FileURL: "https://cdn.example.com/x.jpg", StudentIDs: []uint64{testTALStudentOut}}
		}},
	}
	for _, c := range rejectCases {
		t.Run(c.name, func(t *testing.T) {
			in := TeacherAlbumInput{Title: "越权输入", Category: "activity", AlbumDate: talTodayString(), Photos: talPhotoInputs(1)}
			c.mutate(&in)
			_, err := f.svc.CreateAlbum(ctx, f.tcA, in)
			require.ErrorIs(t, err, ErrTeacherInvalidParams)
		})
	}
}

// 清理证据：硬删后 9960xx 夹具 0 残留（含新关联表与审计日志）。
func TestTeacherAlbumCleanupLeavesNoResidue(t *testing.T) {
	f := newTALFixture(t)
	ctx := context.Background()
	seedTALAlbum(t, f.db, 996141, testTALKG, testTALClassA, domain.AlbumCategoryActivity, talThisMonthDate(0), 2, domain.AlbumStatusDraft, false)

	data, err := f.svc.CreateAlbum(ctx, f.tcA, TeacherAlbumInput{
		Title: "清理用相册", Category: "daily", AlbumDate: talTodayString(),
		Participants: []uint64{testTALStudentA},
		Photos:       []TeacherAlbumPhotoInput{{SourceMediaID: testTALGrowthMedia}},
	})
	require.NoError(t, err)
	albumID := data["id"].(uint64)
	_, err = f.svc.PublishAlbum(ctx, f.tcA, albumID)
	require.NoError(t, err)
	_, err = f.svc.DeleteAlbum(ctx, f.tcA, albumID)
	require.NoError(t, err)

	var albumIDs []uint64
	require.NoError(t, f.db.Unscoped().Model(&domain.Album{}).
		Where("creator_id BETWEEN ? AND ?", testTALUser, testTALUserTwo).Pluck("id", &albumIDs).Error)
	require.NotEmpty(t, albumIDs)

	cleanupTALData(t, f.db)

	count := func(model any, query string, args ...any) int64 { return talCount(t, f.db, model, query, args...) }
	assert.EqualValues(t, 0, count(&domain.Album{}, "id IN ?", albumIDs))
	assert.EqualValues(t, 0, count(&domain.AlbumPhoto{}, "album_id IN ?", albumIDs))
	assert.EqualValues(t, 0, count(&domain.AlbumStudent{}, "album_id IN ?", albumIDs))
	assert.EqualValues(t, 0, count(&domain.AlbumPhotoStudent{}, "kindergarten_id IN ?", []uint64{testTALKG, testTALKGOther}))
	assert.EqualValues(t, 0, count(&domain.OperationLog{}, "user_id BETWEEN ? AND ?", testTALUser, uint64(996029)))
	assert.EqualValues(t, 0, count(&domain.GrowthMedia{}, "id BETWEEN ? AND ?", testTALGrowthMedia, uint64(996069)))
	assert.EqualValues(t, 0, count(&domain.GrowthLogStudent{}, "log_id BETWEEN ? AND ?", testTALGrowthLog, uint64(996059)))
	assert.EqualValues(t, 0, count(&domain.GrowthDailyLog{}, "id BETWEEN ? AND ?", testTALGrowthLog, uint64(996059)))
	assert.EqualValues(t, 0, count(&domain.ParentSetting{}, "user_id BETWEEN ? AND ?", testTALUser, uint64(996079)))
	assert.EqualValues(t, 0, count(&domain.StudentParent{}, "student_id BETWEEN ? AND ?", testTALStudentA, uint64(996039)))
	assert.EqualValues(t, 0, count(&domain.File{}, "id BETWEEN ? AND ?", testTALFileA, uint64(996049)))
	assert.EqualValues(t, 0, count(&domain.UserRole{}, "user_id BETWEEN ? AND ?", testTALUser, uint64(996029)))
	assert.EqualValues(t, 0, count(&domain.Student{}, "id BETWEEN ? AND ?", testTALStudentA, uint64(996039)))
	assert.EqualValues(t, 0, count(&domain.Album{}, "creator_id BETWEEN ? AND ?", testTALUser, testTALUserTwo))
	assert.EqualValues(t, 0, count(&domain.Class{}, "id BETWEEN ? AND ?", uint64(996010), uint64(996019)))
	assert.EqualValues(t, 0, count(&domain.User{}, "id BETWEEN ? AND ?", testTALUser, uint64(996079)))
	assert.EqualValues(t, 0, count(&domain.Kindergarten{}, "id IN ?", []uint64{testTALKG, testTALKGOther}))
}
