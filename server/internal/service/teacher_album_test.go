package service

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
)

// B20：老师端班级相册（openspec class-album）。
// 打真实 MySQL（本地 configs/app.ini），固定 9960xx 夹具，t.Cleanup 硬删 0 残留。
// 覆盖 TC-NEW-ALB-0001–0007：创建顺序、无照片/超 50/非法分类拒绝、统计跨月、
// 分类筛选、详情顺序、跨班/跨园隔离。
//
// 建表走测试内 AutoMigrate：本切片不允许重启 8080，测试不能依赖后端启动时迁移；
// 与 provider.go 的登记保持同一组模型。

const (
	testTALKG      uint64 = 996001 // 目标园所
	testTALKGOther uint64 = 996002 // 他园

	testTALClassA uint64 = 996010 // 目标班
	testTALClassB uint64 = 996011 // 同园别班
	testTALClassO uint64 = 996012 // 他园班级

	testTALUser uint64 = 996020 // 相册创建者（夹具统一用它做清理锚点）

	testTALAlbumDeleted uint64 = 996106 // 软删相册
)

func talThisMonthDate(dayOffset int) time.Time {
	now := teacherPortalDay(time.Now())
	return now.AddDate(0, 0, -now.Day()+1+dayOffset)
}

func talLastMonthDate(dayOffset int) time.Time {
	return talThisMonthDate(dayOffset).AddDate(0, -1, 0)
}

func cleanupTALData(t *testing.T, db *gorm.DB) {
	t.Helper()
	del := func(sql string, args ...any) {
		if err := db.Exec(sql, args...).Error; err != nil {
			t.Logf("cleanup: %v", err)
		}
	}
	// 先删照片：按创建者找回相册（服务创建的相册 ID 是自增的，不能只靠区间）。
	del(`DELETE FROM album_photos WHERE album_id IN
		(SELECT id FROM albums WHERE creator_id = ? OR id BETWEEN 996100 AND 996199)`, testTALUser)
	del("DELETE FROM albums WHERE creator_id = ? OR id BETWEEN 996100 AND 996199", testTALUser)
	del("DELETE FROM classes WHERE id BETWEEN 996010 AND 996019")
	del("DELETE FROM users WHERE id BETWEEN 996020 AND 996029")
	del("DELETE FROM kindergartens WHERE id IN (?, ?)", testTALKG, testTALKGOther)
}

type talFixture struct {
	db      *gorm.DB
	svc     *TeacherPortalService
	tcA     *TeacherPortalContext
	tcB     *TeacherPortalContext
	tcOther *TeacherPortalContext
}

func newTALFixture(t *testing.T) *talFixture {
	t.Helper()
	db := openCLPTestDB(t)
	require.NoError(t, db.AutoMigrate(&domain.Album{}, &domain.AlbumPhoto{}))
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

	require.NoError(t, db.Create(&domain.User{
		BaseModel: domain.BaseModel{ID: testTALUser}, Nickname: "相册老师", Phone: "13900009961", PasswordHash: "x",
	}).Error)

	return &talFixture{
		db:      db,
		svc:     NewTeacherPortalService(db, nil, nil, nil, nil, nil, config.FaceConfig{}),
		tcA:     &TeacherPortalContext{UserID: testTALUser, KindergartenID: testTALKG, ClassID: testTALClassA},
		tcB:     &TeacherPortalContext{UserID: testTALUser, KindergartenID: testTALKG, ClassID: testTALClassB},
		tcOther: &TeacherPortalContext{UserID: testTALUser, KindergartenID: testTALKGOther, ClassID: testTALClassO},
	}
}

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
	photos, ok := detail["photos"].([]map[string]any)
	require.True(t, ok)
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
}

// TC-NEW-ALB-0002：没有照片整单拒绝，不创建相册。
func TestTeacherAlbumCreateRejectsNoPhotos(t *testing.T) {
	f := newTALFixture(t)

	_, err := createTALAlbum(t, f, f.tcA, "空相册", "daily", talThisMonthDate(0).Format("2006-01-02"), nil)
	require.ErrorIs(t, err, ErrTeacherInvalidParams)
	assert.EqualValues(t, 0, talAlbumCount(t, f.db))
}

// 超过 50 张整单拒绝（上限 1–50）。
func TestTeacherAlbumCreateRejectsTooManyPhotos(t *testing.T) {
	f := newTALFixture(t)

	_, err := createTALAlbum(t, f, f.tcA, "照片太多", "daily", talThisMonthDate(0).Format("2006-01-02"), talPhotoInputs(51))
	require.ErrorIs(t, err, ErrTeacherInvalidParams)
	assert.EqualValues(t, 0, talAlbumCount(t, f.db))
}

// TC-NEW-ALB-0007：分类只接受四值。
func TestTeacherAlbumCreateRejectsInvalidCategory(t *testing.T) {
	f := newTALFixture(t)

	for _, category := range []string{"other", "", "Activity", "全部"} {
		_, err := createTALAlbum(t, f, f.tcA, "非法分类", category, talThisMonthDate(0).Format("2006-01-02"), talPhotoInputs(1))
		require.ErrorIs(t, err, ErrTeacherInvalidParams, "category=%q 必须拒绝", category)
	}
	assert.EqualValues(t, 0, talAlbumCount(t, f.db))
}

// 标题 1–30、日期 YYYY-MM-DD、URL http(s)、尺寸非负：逐条拒绝且不落库。
func TestTeacherAlbumCreateValidation(t *testing.T) {
	f := newTALFixture(t)
	date := talThisMonthDate(0).Format("2006-01-02")

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

	seedTALAlbum(t, f.db, 996101, testTALKG, testTALClassA, domain.AlbumCategoryActivity, talThisMonthDate(0), 3, false)
	seedTALAlbum(t, f.db, 996102, testTALKG, testTALClassA, domain.AlbumCategoryDaily, talThisMonthDate(1), 2, false)
	seedTALAlbum(t, f.db, 996103, testTALKG, testTALClassA, domain.AlbumCategoryActivity, talLastMonthDate(0), 1, false)
	seedTALAlbum(t, f.db, 996104, testTALKG, testTALClassB, domain.AlbumCategoryWork, talThisMonthDate(0), 4, false)
	seedTALAlbum(t, f.db, 996105, testTALKGOther, testTALClassO, domain.AlbumCategoryFestival, talThisMonthDate(0), 5, false)
	seedTALAlbum(t, f.db, testTALAlbumDeleted, testTALKG, testTALClassA, domain.AlbumCategoryActivity, talThisMonthDate(0), 6, true)

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

// TC-NEW-ALB-0004：分类筛选只返回该分类，日期倒序；不传返回全部；非法筛选值 422。
func TestTeacherAlbumListFilterByCategory(t *testing.T) {
	f := newTALFixture(t)

	seedTALAlbum(t, f.db, 996111, testTALKG, testTALClassA, domain.AlbumCategoryActivity, talThisMonthDate(3), 1, false)
	seedTALAlbum(t, f.db, 996112, testTALKG, testTALClassA, domain.AlbumCategoryActivity, talThisMonthDate(1), 1, false)
	seedTALAlbum(t, f.db, 996113, testTALKG, testTALClassA, domain.AlbumCategoryDaily, talThisMonthDate(2), 1, false)
	seedTALAlbum(t, f.db, 996114, testTALKG, testTALClassA, domain.AlbumCategoryFestival, talThisMonthDate(0), 1, false)
	seedTALAlbum(t, f.db, 996115, testTALKG, testTALClassA, domain.AlbumCategoryWork, talThisMonthDate(4), 1, false)
	seedTALAlbum(t, f.db, 996116, testTALKG, testTALClassB, domain.AlbumCategoryActivity, talThisMonthDate(5), 1, false)

	data, err := f.svc.ListAlbums(context.Background(), f.tcA, "activity", 1, 20)
	require.NoError(t, err)
	assert.EqualValues(t, 2, data["total"])
	list := data["list"].([]map[string]any)
	require.Len(t, list, 2)
	assert.EqualValues(t, 996111, list[0]["id"], "活动分类按日期倒序")
	assert.EqualValues(t, 996112, list[1]["id"])
	assert.Equal(t, "活动", list[0]["category_label"])
	assert.NotContains(t, fmt.Sprint(list), "996116", "别班活动相册不得出现")

	all, err := f.svc.ListAlbums(context.Background(), f.tcA, "", 1, 20)
	require.NoError(t, err)
	assert.EqualValues(t, 5, all["total"])
	allList := all["list"].([]map[string]any)
	require.Len(t, allList, 5)
	assert.EqualValues(t, 996115, allList[0]["id"], "全部按日期倒序")
	assert.EqualValues(t, 996114, allList[4]["id"])

	paged, err := f.svc.ListAlbums(context.Background(), f.tcA, "", 3, 2)
	require.NoError(t, err)
	assert.EqualValues(t, 5, paged["total"])
	assert.Len(t, paged["list"].([]map[string]any), 1)

	_, err = f.svc.ListAlbums(context.Background(), f.tcA, "other", 1, 20)
	require.ErrorIs(t, err, ErrTeacherInvalidParams)
}

// TC-NEW-ALB-0005：详情返回全部照片，顺序与创建时一致。
func TestTeacherAlbumDetailPhotoOrder(t *testing.T) {
	f := newTALFixture(t)
	seedTALAlbum(t, f.db, 996121, testTALKG, testTALClassA, domain.AlbumCategoryWork, talThisMonthDate(2), 12, false)

	detail, err := f.svc.GetAlbum(context.Background(), f.tcA, 996121)
	require.NoError(t, err)
	photos := detail["photos"].([]map[string]any)
	require.Len(t, photos, 12)
	for i, p := range photos {
		assert.Equal(t, fmt.Sprintf("https://cdn.example.com/seed/996121/%d.jpg", i), p["file_url"], "第 %d 张顺序不符", i+1)
	}
	assert.EqualValues(t, 12, detail["photo_count"])
	assert.Equal(t, "作品", detail["category_label"])

	_, err = f.svc.GetAlbum(context.Background(), f.tcA, 996129)
	require.ErrorIs(t, err, ErrTeacherNotFound, "不存在的相册返回 404 语义")
}

// TC-NEW-ALB-0006：班级与园所隔离——别班/他园读不到，创建只能落在当前班。
func TestTeacherAlbumIsolation(t *testing.T) {
	f := newTALFixture(t)
	seedTALAlbum(t, f.db, 996131, testTALKG, testTALClassA, domain.AlbumCategoryActivity, talThisMonthDate(0), 2, false)

	_, err := f.svc.GetAlbum(context.Background(), f.tcB, 996131)
	require.ErrorIs(t, err, ErrTeacherNotFound, "同园别班读取必须 404")

	_, err = f.svc.GetAlbum(context.Background(), f.tcOther, 996131)
	require.ErrorIs(t, err, ErrTeacherNotFound, "他园读取必须 404")

	listB, err := f.svc.ListAlbums(context.Background(), f.tcB, "", 1, 20)
	require.NoError(t, err)
	assert.EqualValues(t, 0, listB["total"])

	statsB, err := f.svc.AlbumStats(context.Background(), f.tcB)
	require.NoError(t, err)
	assert.EqualValues(t, 0, statsB["total_photos"])
	assert.EqualValues(t, 0, statsB["album_count"])

	// 创建不接受班级参数，只能落在当前老师班级。
	idB, err := createTALAlbum(t, f, f.tcB, "二班相册", "daily", talThisMonthDate(0).Format("2006-01-02"), talPhotoInputs(1))
	require.NoError(t, err)
	var albumB domain.Album
	require.NoError(t, f.db.First(&albumB, idB).Error)
	assert.EqualValues(t, testTALKG, albumB.KindergartenID)
	assert.EqualValues(t, testTALClassB, albumB.ClassID)

	// 别班创建后，目标班仍看不到它。
	listA, err := f.svc.ListAlbums(context.Background(), f.tcA, "", 1, 20)
	require.NoError(t, err)
	assert.EqualValues(t, 1, listA["total"], "一班只看到自己的相册")
}

// 清理证据：硬删后 9960xx 夹具 0 残留。
func TestTeacherAlbumCleanupLeavesNoResidue(t *testing.T) {
	f := newTALFixture(t)
	seedTALAlbum(t, f.db, 996141, testTALKG, testTALClassA, domain.AlbumCategoryActivity, talThisMonthDate(0), 2, false)
	_, err := createTALAlbum(t, f, f.tcA, "清理用相册", "daily", talThisMonthDate(0).Format("2006-01-02"), talPhotoInputs(2))
	require.NoError(t, err)

	var albumIDs []uint64
	require.NoError(t, f.db.Unscoped().Model(&domain.Album{}).
		Where("creator_id = ?", testTALUser).Pluck("id", &albumIDs).Error)
	require.NotEmpty(t, albumIDs)

	cleanupTALData(t, f.db)

	count := func(model any, query string, args ...any) int64 {
		var n int64
		require.NoError(t, f.db.Unscoped().Model(model).Where(query, args...).Count(&n).Error)
		return n
	}
	assert.EqualValues(t, 0, count(&domain.Album{}, "id IN ?", albumIDs))
	assert.EqualValues(t, 0, count(&domain.AlbumPhoto{}, "album_id IN ?", albumIDs))
	assert.EqualValues(t, 0, count(&domain.Album{}, "creator_id = ?", testTALUser))
	assert.EqualValues(t, 0, count(&domain.Class{}, "id BETWEEN ? AND ?", uint64(996010), uint64(996019)))
	assert.EqualValues(t, 0, count(&domain.User{}, "id BETWEEN ? AND ?", uint64(996020), uint64(996029)))
	assert.EqualValues(t, 0, count(&domain.Kindergarten{}, "id IN ?", []uint64{testTALKG, testTALKGOther}))
}
