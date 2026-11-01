package service_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"kindergarten-server/internal/app"
	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
	handler "kindergarten-server/internal/handler/http"
	"kindergarten-server/internal/repository"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/pagination"
)

// OS-ALB-5：管理端班级相册（openspec class-album-parent-sharing 6.1–6.3）。
//
// 打真实 MySQL（本地 configs/app.ini），固定 9981xx 夹具，t.Cleanup 硬删 0 残留。
// 覆盖：
//   - TC-NEW-ALB-0018 园长查看并下架：本园隔离、筛选、分页、详情照片/参与者、
//     下架后状态与审计、重复下架幂等、草稿拒绝、跨园/软删 404；
//   - TC-NEW-ALB-0019 园长不能编辑：真实路由表只有 GET/GET/POST unpublish，
//     没有创建/编辑/删除端点。
//
// 本文件用外部测试包（service_test）是为了导入 internal/app 断言真实路由表：
// 在 package service 内导入 app 会形成 import cycle。业务断言仍走已导出的
// AdminService 方法，不复制服务逻辑。

const (
	adminAlbumKG      uint64 = 998101 // 目标园所
	adminAlbumKGOther uint64 = 998102 // 他园

	adminAlbumClassA uint64 = 998110 // 目标班
	adminAlbumClassB uint64 = 998111 // 同园别班
	adminAlbumClassO uint64 = 998112 // 他园班级

	adminAlbumCreator  uint64 = 998120 // 相册创建老师
	adminAlbumOperator uint64 = 998121 // 执行下架的园长

	adminAlbumStudentA uint64 = 998130 // 参与幼儿 A
	adminAlbumStudentB uint64 = 998131 // 参与幼儿 B
	adminAlbumStudentO uint64 = 998132 // 他园幼儿

	adminAlbumPublishedA uint64 = 998150 // 已发布：班 A / 活动 / 3 张照片
	adminAlbumDraftA     uint64 = 998151 // 草稿：班 A / 日常
	adminAlbumUnpubA     uint64 = 998152 // 已下架：班 A / 活动
	adminAlbumPublishedB uint64 = 998153 // 已发布：班 B / 作品
	adminAlbumOtherKg    uint64 = 998154 // 已发布：他园（必须不可见）
	adminAlbumDeleted    uint64 = 998155 // 已发布后软删（必须不可见）
	adminAlbumFestivalA  uint64 = 998156 // 已发布：班 A / 节日
	adminAlbumDraftA2    uint64 = 998157 // 草稿：班 A / 活动

	// 主相册照片 ID = adminAlbumPhotoBase + sort_num。
	adminAlbumPhotoBase  uint64 = 998160
	adminAlbumOtherPhoto uint64 = 998163
)

var adminAlbumTestDB *gorm.DB

func openAdminAlbumTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	if adminAlbumTestDB != nil {
		return adminAlbumTestDB
	}
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
	adminAlbumTestDB = db
	return db
}

func cleanupAdminAlbumData(t *testing.T, db *gorm.DB) {
	t.Helper()
	del := func(sql string, args ...any) {
		if err := db.Exec(sql, args...).Error; err != nil {
			t.Logf("cleanup: %v", err)
		}
	}
	// 关联表先删：album_photo_students 没有 album_id，只能经照片子查询找回。
	del(`DELETE FROM album_photo_students WHERE photo_id IN
		(SELECT id FROM album_photos WHERE album_id BETWEEN ? AND ?)`,
		adminAlbumPublishedA, uint64(998159))
	del("DELETE FROM album_students WHERE album_id BETWEEN ? AND ?",
		adminAlbumPublishedA, uint64(998159))
	del("DELETE FROM album_photos WHERE album_id BETWEEN ? AND ?",
		adminAlbumPublishedA, uint64(998159))
	del("DELETE FROM albums WHERE id BETWEEN ? AND ?", adminAlbumPublishedA, uint64(998159))
	del("DELETE FROM operation_logs WHERE user_id IN (?, ?) OR target_id BETWEEN ? AND ?",
		adminAlbumCreator, adminAlbumOperator, adminAlbumPublishedA, uint64(998159))
	del("DELETE FROM students WHERE id BETWEEN ? AND ?", adminAlbumStudentA, uint64(998139))
	del("DELETE FROM users WHERE id BETWEEN ? AND ?", adminAlbumCreator, uint64(998129))
	del("DELETE FROM classes WHERE id BETWEEN ? AND ?", uint64(998110), uint64(998119))
	del("DELETE FROM kindergartens WHERE id IN (?, ?)", adminAlbumKG, adminAlbumKGOther)
}

// adminAlbumMigrateOnce 每个进程只建一次表，避免每个用例重复 AutoMigrate。
var adminAlbumMigrateOnce sync.Once

func newAdminAlbumFixture(t *testing.T) (*gorm.DB, *service.AdminService) {
	t.Helper()
	db := openAdminAlbumTestDB(t)
	adminAlbumMigrateOnce.Do(func() {
		require.NoError(t, db.AutoMigrate(
			&domain.Kindergarten{}, &domain.Class{}, &domain.User{},
			&domain.Student{}, &domain.Album{}, &domain.AlbumPhoto{},
			&domain.AlbumStudent{}, &domain.AlbumPhotoStudent{},
			&domain.OperationLog{},
		))
	})
	cleanupAdminAlbumData(t, db)
	t.Cleanup(func() { cleanupAdminAlbumData(t, db) })

	require.NoError(t, db.Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: adminAlbumKG}, Name: "测试园所-管理端相册",
	}).Error)
	require.NoError(t, db.Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: adminAlbumKGOther}, Name: "测试园所-管理端相册他园",
	}).Error)
	for _, cl := range []struct {
		id   uint64
		kg   uint64
		name string
	}{
		{adminAlbumClassA, adminAlbumKG, "相册一班"},
		{adminAlbumClassB, adminAlbumKG, "相册二班"},
		{adminAlbumClassO, adminAlbumKGOther, "他园相册班"},
	} {
		require.NoError(t, db.Create(&domain.Class{
			BaseModel: domain.BaseModel{ID: cl.id}, KindergartenID: cl.kg,
			Name: cl.name, Grade: "小班", Status: 1,
		}).Error)
	}
	for _, u := range []domain.User{
		{BaseModel: domain.BaseModel{ID: adminAlbumCreator}, Nickname: "园长-创建人", Phone: "13900009981"},
		{BaseModel: domain.BaseModel{ID: adminAlbumOperator}, Nickname: "园长-操作人", Phone: "13900009982"},
	} {
		require.NoError(t, db.Create(&u).Error)
	}
	for _, st := range []struct {
		id      uint64
		kg      uint64
		classID uint64
		name    string
	}{
		{adminAlbumStudentA, adminAlbumKG, adminAlbumClassA, "幼儿A"},
		{adminAlbumStudentB, adminAlbumKG, adminAlbumClassA, "幼儿B"},
		{adminAlbumStudentO, adminAlbumKGOther, adminAlbumClassO, "他园幼儿"},
	} {
		require.NoError(t, db.Create(&domain.Student{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: st.id}},
			KindergartenID:  st.kg, ClassID: st.classID, Name: st.name, Status: 1,
			Birthday: time.Now().AddDate(-4, 0, 0), EnrollDate: time.Now(),
		}).Error)
	}

	baseDate := time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)
	publishedAt := baseDate.Add(26 * time.Hour)
	for _, a := range []struct {
		id        uint64
		kg        uint64
		classID   uint64
		title     string
		category  domain.AlbumCategory
		status    string
		dayOffset int
		photoNum  int
		deleted   bool
		published bool
	}{
		{adminAlbumPublishedA, adminAlbumKG, adminAlbumClassA, "相册-998150", domain.AlbumCategoryActivity, domain.AlbumStatusPublished, 0, 3, false, true},
		{adminAlbumDraftA, adminAlbumKG, adminAlbumClassA, "相册-998151", domain.AlbumCategoryDaily, domain.AlbumStatusDraft, -1, 1, false, false},
		{adminAlbumUnpubA, adminAlbumKG, adminAlbumClassA, "相册-998152", domain.AlbumCategoryActivity, domain.AlbumStatusUnpublished, -2, 1, false, false},
		{adminAlbumPublishedB, adminAlbumKG, adminAlbumClassB, "相册-998153", domain.AlbumCategoryWork, domain.AlbumStatusPublished, -3, 1, false, true},
		{adminAlbumOtherKg, adminAlbumKGOther, adminAlbumClassO, "相册-998154", domain.AlbumCategoryActivity, domain.AlbumStatusPublished, -4, 1, false, true},
		{adminAlbumDeleted, adminAlbumKG, adminAlbumClassA, "相册-998155", domain.AlbumCategoryFestival, domain.AlbumStatusPublished, -5, 1, true, true},
		{adminAlbumFestivalA, adminAlbumKG, adminAlbumClassA, "相册-998156", domain.AlbumCategoryFestival, domain.AlbumStatusPublished, -6, 1, false, true},
		{adminAlbumDraftA2, adminAlbumKG, adminAlbumClassA, "相册-998157", domain.AlbumCategoryActivity, domain.AlbumStatusDraft, -7, 1, false, false},
	} {
		album := domain.Album{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: a.id}},
			KindergartenID:  a.kg, ClassID: a.classID, CreatorID: adminAlbumCreator,
			Title: a.title, Category: a.category, AlbumDate: baseDate.AddDate(0, 0, a.dayOffset),
			CoverURL:   fmt.Sprintf("https://cdn.example.com/admin-album/%d.jpg", a.id),
			PhotoCount: a.photoNum, Status: a.status,
		}
		if a.published {
			at := publishedAt
			album.PublishedAt = &at
			album.PublishedBy = adminAlbumCreator
		}
		require.NoError(t, db.Create(&album).Error)
		if a.deleted {
			require.NoError(t, db.Delete(&domain.Album{}, a.id).Error)
		}
	}

	// 主相册 3 张照片：sort 0/1/2；照片级关联分别是 A、B、无。
	for i := 0; i < 3; i++ {
		photoID := adminAlbumPhotoBase + uint64(i)
		require.NoError(t, db.Create(&domain.AlbumPhoto{
			BaseModel: domain.BaseModel{ID: photoID},
			AlbumID:   adminAlbumPublishedA,
			FileURL:   fmt.Sprintf("https://cdn.example.com/admin-album/photo-%d.jpg", photoID),
			ThumbURL:  fmt.Sprintf("https://cdn.example.com/admin-album/photo-%d-thumb.jpg", photoID),
			Width:     1200, Height: 900, SortNum: i,
		}).Error)
	}
	require.NoError(t, db.Create(&domain.AlbumPhoto{
		BaseModel: domain.BaseModel{ID: adminAlbumOtherPhoto},
		AlbumID:   adminAlbumFestivalA,
		FileURL:   "https://cdn.example.com/admin-album/festival.jpg",
		SortNum:   0,
	}).Error)
	for _, sid := range []uint64{adminAlbumStudentA, adminAlbumStudentB} {
		require.NoError(t, db.Create(&domain.AlbumStudent{
			KindergartenID: adminAlbumKG, AlbumID: adminAlbumPublishedA, StudentID: sid,
		}).Error)
	}
	for _, link := range []struct {
		photoID   uint64
		studentID uint64
	}{
		{adminAlbumPhotoBase, adminAlbumStudentA},
		{adminAlbumPhotoBase + 1, adminAlbumStudentB},
	} {
		require.NoError(t, db.Create(&domain.AlbumPhotoStudent{
			KindergartenID: adminAlbumKG, PhotoID: link.photoID, StudentID: link.studentID,
		}).Error)
	}

	return db, service.NewAdminService(
		db,
		repository.NewUserRepo(db),
		repository.NewUserRoleRepo(db),
		repository.NewAdminRepo(db),
		repository.NewKindergartenRepo(db),
		repository.NewOperationLogRepo(db),
	)
}

func adminAlbumIDs(items []service.AdminAlbumListItem) []uint64 {
	out := make([]uint64, 0, len(items))
	for _, item := range items {
		out = append(out, item.ID)
	}
	return out
}

func adminAlbumPhotoIDs(photos []service.AdminAlbumPhoto) []uint64 {
	out := make([]uint64, 0, len(photos))
	for _, p := range photos {
		out = append(out, p.ID)
	}
	return out
}

func adminAlbumStudentIDs(students []service.AdminAlbumStudent) []uint64 {
	out := make([]uint64, 0, len(students))
	for _, st := range students {
		out = append(out, st.ID)
	}
	return out
}

// TC-NEW-ALB-0018（列表/隔离）：
// 只返回本园未软删相册；他园与软删相册不出现；class/status/category 筛选生效；
// 非法筛选值 422 而不是静默返回全部。
func TestAdminAlbumListIsolationAndFilters(t *testing.T) {
	db, svc := newAdminAlbumFixture(t)
	ctx := context.Background()

	list, total, page, pageSize, err := svc.ListAdminAlbums(ctx, adminAlbumKG, 0, "", "", pagination.Query{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(6), total)
	assert.Equal(t, 1, page)
	assert.Equal(t, 20, pageSize)
	assert.NotNil(t, list)
	assert.Equal(t,
		[]uint64{adminAlbumPublishedA, adminAlbumDraftA, adminAlbumUnpubA, adminAlbumPublishedB, adminAlbumFestivalA, adminAlbumDraftA2},
		adminAlbumIDs(list))
	assert.NotContains(t, adminAlbumIDs(list), adminAlbumOtherKg, "他园相册必须不可见")
	assert.NotContains(t, adminAlbumIDs(list), adminAlbumDeleted, "软删相册必须不可见")

	var main service.AdminAlbumListItem
	for _, item := range list {
		if item.ID == adminAlbumPublishedA {
			main = item
		}
	}
	assert.Equal(t, "相册一班", main.ClassName)
	assert.Equal(t, "活动", main.CategoryLabel)
	assert.Equal(t, "published", main.Status)
	assert.Equal(t, 3, main.PhotoCount)
	assert.Equal(t, adminAlbumCreator, main.Creator.ID)
	assert.Equal(t, "园长-创建人", main.Creator.Nickname)
	require.NotNil(t, main.PublishedAt)

	cases := []struct {
		name      string
		classID   uint64
		status    string
		category  string
		wantIDs   []uint64
		wantTotal int64
	}{
		{"班级筛选", adminAlbumClassA, "", "", []uint64{adminAlbumPublishedA, adminAlbumDraftA, adminAlbumUnpubA, adminAlbumFestivalA, adminAlbumDraftA2}, 5},
		{"状态筛选-已发布", 0, domain.AlbumStatusPublished, "", []uint64{adminAlbumPublishedA, adminAlbumPublishedB, adminAlbumFestivalA}, 3},
		{"状态筛选-草稿", 0, domain.AlbumStatusDraft, "", []uint64{adminAlbumDraftA, adminAlbumDraftA2}, 2},
		{"分类筛选-活动", 0, "", string(domain.AlbumCategoryActivity), []uint64{adminAlbumPublishedA, adminAlbumUnpubA, adminAlbumDraftA2}, 3},
		{"分类筛选-节日", 0, "", string(domain.AlbumCategoryFestival), []uint64{adminAlbumFestivalA}, 1},
		{"组合筛选", adminAlbumClassA, domain.AlbumStatusPublished, "", []uint64{adminAlbumPublishedA, adminAlbumFestivalA}, 2},
		{"他园班级筛选", adminAlbumClassO, "", "", []uint64{}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, gotTotal, _, _, err := svc.ListAdminAlbums(ctx, adminAlbumKG, tc.classID, tc.status, tc.category, pagination.Query{Page: 1, PageSize: 20})
			require.NoError(t, err)
			assert.Equal(t, tc.wantTotal, gotTotal)
			assert.Equal(t, tc.wantIDs, adminAlbumIDs(got))
		})
	}

	_, _, _, _, err = svc.ListAdminAlbums(ctx, adminAlbumKG, 0, "archived", "", pagination.Query{})
	assert.ErrorIs(t, err, service.ErrAdminAlbumInvalidParam, "非法状态必须 422，不能退化成全部")
	_, _, _, _, err = svc.ListAdminAlbums(ctx, adminAlbumKG, 0, "", "party", pagination.Query{})
	assert.ErrorIs(t, err, service.ErrAdminAlbumInvalidParam)
	_, _, _, _, err = svc.ListAdminAlbums(ctx, 0, 0, "", "", pagination.Query{})
	assert.Error(t, err)

	// 他园管理员视角：看不到目标园的任何相册。
	other, otherTotal, _, _, err := svc.ListAdminAlbums(ctx, adminAlbumKGOther, 0, "", "", pagination.Query{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(1), otherTotal)
	assert.Equal(t, []uint64{adminAlbumOtherKg}, adminAlbumIDs(other))

	// 使用 db 仅为确认夹具未含意外残留（后续用例各自清理）；含软删行共 8 条。
	var albumCount int64
	require.NoError(t, db.Unscoped().Model(&domain.Album{}).
		Where("id BETWEEN ? AND ?", adminAlbumPublishedA, uint64(998159)).Count(&albumCount).Error)
	assert.Equal(t, int64(8), albumCount, "本租户相册夹具含软删共 8 条")
}

// TC-NEW-ALB-0018（分页/排序）：
// album_date DESC, id DESC；page/page_size 回显且不重叠。
func TestAdminAlbumListPaginationAndOrder(t *testing.T) {
	_, svc := newAdminAlbumFixture(t)
	ctx := context.Background()

	for _, tc := range []struct {
		page    int
		wantIDs []uint64
	}{
		{1, []uint64{adminAlbumPublishedA, adminAlbumDraftA}},
		{2, []uint64{adminAlbumUnpubA, adminAlbumPublishedB}},
		{3, []uint64{adminAlbumFestivalA, adminAlbumDraftA2}},
		{4, []uint64{}},
	} {
		list, total, page, pageSize, err := svc.ListAdminAlbums(ctx, adminAlbumKG, 0, "", "", pagination.Query{Page: tc.page, PageSize: 2})
		require.NoError(t, err)
		assert.Equal(t, int64(6), total)
		assert.Equal(t, tc.page, page)
		assert.Equal(t, 2, pageSize)
		assert.Equal(t, tc.wantIDs, adminAlbumIDs(list))
	}
}

// TC-NEW-ALB-0018（详情）：
// 详情返回元信息 + 照片（sort 顺序）+ 照片级幼儿 + 相册级参与者；
// 跨园/软删/不存在一律 404，不泄露存在性。
func TestAdminAlbumDetailPhotosAndParticipants(t *testing.T) {
	_, svc := newAdminAlbumFixture(t)
	ctx := context.Background()

	detail, err := svc.GetAdminAlbum(ctx, adminAlbumKG, adminAlbumPublishedA)
	require.NoError(t, err)
	require.NotNil(t, detail)
	assert.Equal(t, adminAlbumPublishedA, detail.ID)
	assert.Equal(t, adminAlbumClassA, detail.ClassID)
	assert.Equal(t, "相册一班", detail.ClassName)
	assert.Equal(t, "相册-998150", detail.Title)
	assert.Equal(t, "活动", detail.CategoryLabel)
	assert.Equal(t, "2026-10-25", detail.AlbumDate)
	assert.Equal(t, 3, detail.PhotoCount)
	assert.Equal(t, "published", detail.Status)
	require.NotNil(t, detail.PublishedAt)
	assert.Equal(t, "园长-创建人", detail.Creator.Nickname)

	require.Len(t, detail.Photos, 3)
	assert.Equal(t, []uint64{adminAlbumPhotoBase, adminAlbumPhotoBase + 1, adminAlbumPhotoBase + 2}, adminAlbumPhotoIDs(detail.Photos))
	assert.Equal(t, []int{0, 1, 2}, []int{detail.Photos[0].SortNum, detail.Photos[1].SortNum, detail.Photos[2].SortNum})
	assert.Equal(t, adminAlbumPhotoBase, detail.Photos[0].ID)
	assert.Equal(t, "https://cdn.example.com/admin-album/photo-998160.jpg", detail.Photos[0].FileURL)
	assert.Equal(t, "https://cdn.example.com/admin-album/photo-998160-thumb.jpg", detail.Photos[0].ThumbURL)
	assert.Equal(t, 1200, detail.Photos[0].Width)
	assert.Equal(t, 900, detail.Photos[0].Height)
	assert.Nil(t, detail.Photos[0].FileID)

	assert.Equal(t, []uint64{adminAlbumStudentA}, adminAlbumStudentIDs(detail.Photos[0].Students))
	assert.Equal(t, []uint64{adminAlbumStudentB}, adminAlbumStudentIDs(detail.Photos[1].Students))
	assert.NotNil(t, detail.Photos[2].Students)
	assert.Empty(t, detail.Photos[2].Students)

	require.Len(t, detail.Participants, 2)
	assert.Equal(t, []uint64{adminAlbumStudentA, adminAlbumStudentB}, adminAlbumStudentIDs(detail.Participants))
	assert.Equal(t, []string{"幼儿A", "幼儿B"}, []string{detail.Participants[0].Name, detail.Participants[1].Name})

	// 草稿与已下架仍可读（园长只读）。
	for _, id := range []uint64{adminAlbumDraftA, adminAlbumUnpubA} {
		got, err := svc.GetAdminAlbum(ctx, adminAlbumKG, id)
		require.NoError(t, err, "相册 %d 园长应可读", id)
		assert.Equal(t, id, got.ID)
	}

	for _, id := range []uint64{adminAlbumOtherKg, adminAlbumDeleted, 999999} {
		_, err := svc.GetAdminAlbum(ctx, adminAlbumKG, id)
		assert.ErrorIs(t, err, service.ErrAdminAlbumNotFound, "相册 %d 必须 404", id)
	}
	_, err = svc.GetAdminAlbum(ctx, 0, adminAlbumPublishedA)
	assert.Error(t, err)
}

// TC-NEW-ALB-0018（下架）：状态迁移 + 家长侧查询立即不可见 + 审计字段完整。
func TestAdminAlbumUnpublishWritesAuditAndHidesFromParent(t *testing.T) {
	db, svc := newAdminAlbumFixture(t)
	ctx := context.Background()

	res, err := svc.UnpublishAdminAlbum(ctx, adminAlbumKG, adminAlbumOperator, adminAlbumPublishedA,
		"10.1.2.3", "admin-album-test-ua")
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, adminAlbumPublishedA, res.ID)
	assert.Equal(t, domain.AlbumStatusUnpublished, res.Status)
	assert.True(t, res.Changed)

	var row domain.Album
	require.NoError(t, db.First(&row, adminAlbumPublishedA).Error)
	assert.Equal(t, domain.AlbumStatusUnpublished, row.Status)
	assert.Nil(t, row.PublishedAt)
	assert.Equal(t, uint64(0), row.PublishedBy)

	// 家长端列表只查 status=published：下架后同班只剩节日相册。
	published, total, err := repository.NewAlbumRepo(db).List(
		ctx, adminAlbumKG, adminAlbumClassA, "", domain.AlbumStatusPublished, 0, 100)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	ids := make([]uint64, 0, len(published))
	for _, a := range published {
		ids = append(ids, a.ID)
	}
	assert.Equal(t, []uint64{adminAlbumFestivalA}, ids)
	assert.NotContains(t, ids, adminAlbumPublishedA)

	var logs []domain.OperationLog
	require.NoError(t, db.Where("target_id = ? AND module = ?", adminAlbumPublishedA, "albums").Find(&logs).Error)
	require.Len(t, logs, 1, "真实下架必须且只能写一条审计")
	l := logs[0]
	assert.Equal(t, adminAlbumKG, l.KindergartenID)
	assert.Equal(t, adminAlbumOperator, l.UserID)
	assert.Equal(t, "unpublish", l.Action)
	assert.Equal(t, "album", l.TargetType)
	assert.Equal(t, adminAlbumPublishedA, l.TargetID)
	assert.Contains(t, l.Detail, "相册-998150")
	assert.Contains(t, l.Detail, "相册一班")
	assert.Contains(t, l.Detail, "published->unpublished")
	assert.Equal(t, "10.1.2.3", l.IP)
	assert.Equal(t, "admin-album-test-ua", l.UserAgent)
	assert.False(t, l.CreatedAt.IsZero(), "审计必须带操作时间")
}

// TC-NEW-ALB-0018（幂等与非法状态）：
// 重复下架 200/changed=false 且不重复写审计；草稿 422；跨园/软删 404。
func TestAdminAlbumUnpublishIdempotentAndRejectsDraft(t *testing.T) {
	db, svc := newAdminAlbumFixture(t)
	ctx := context.Background()

	first, err := svc.UnpublishAdminAlbum(ctx, adminAlbumKG, adminAlbumOperator, adminAlbumPublishedA, "10.0.0.1", "ua-1")
	require.NoError(t, err)
	assert.True(t, first.Changed)

	second, err := svc.UnpublishAdminAlbum(ctx, adminAlbumKG, adminAlbumOperator, adminAlbumPublishedA, "10.0.0.2", "ua-2")
	require.NoError(t, err, "重复下架必须幂等成功")
	assert.False(t, second.Changed)
	assert.Equal(t, domain.AlbumStatusUnpublished, second.Status)

	var logCount int64
	require.NoError(t, db.Model(&domain.OperationLog{}).
		Where("target_id = ? AND action = ?", adminAlbumPublishedA, "unpublish").Count(&logCount).Error)
	assert.Equal(t, int64(1), logCount, "幂等调用不得新增审计")

	// 夹具中已是下架的相册：幂等成功且审计为 0。
	idem, err := svc.UnpublishAdminAlbum(ctx, adminAlbumKG, adminAlbumOperator, adminAlbumUnpubA, "10.0.0.3", "ua-3")
	require.NoError(t, err)
	assert.False(t, idem.Changed)
	require.NoError(t, db.Model(&domain.OperationLog{}).
		Where("target_id = ?", adminAlbumUnpubA).Count(&logCount).Error)
	assert.Equal(t, int64(0), logCount)

	// 草稿下架：422，状态不变，不写审计。
	_, err = svc.UnpublishAdminAlbum(ctx, adminAlbumKG, adminAlbumOperator, adminAlbumDraftA, "10.0.0.4", "ua-4")
	assert.ErrorIs(t, err, service.ErrAdminAlbumNotPublished)
	var draft domain.Album
	require.NoError(t, db.First(&draft, adminAlbumDraftA).Error)
	assert.Equal(t, domain.AlbumStatusDraft, draft.Status)
	require.NoError(t, db.Model(&domain.OperationLog{}).
		Where("target_id = ?", adminAlbumDraftA).Count(&logCount).Error)
	assert.Equal(t, int64(0), logCount)

	// 跨园与软删按不存在处理。
	_, err = svc.UnpublishAdminAlbum(ctx, adminAlbumKG, adminAlbumOperator, adminAlbumOtherKg, "10.0.0.5", "ua-5")
	assert.ErrorIs(t, err, service.ErrAdminAlbumNotFound)
	_, err = svc.UnpublishAdminAlbum(ctx, adminAlbumKG, adminAlbumOperator, adminAlbumDeleted, "10.0.0.6", "ua-6")
	assert.ErrorIs(t, err, service.ErrAdminAlbumNotFound)
	_, err = svc.UnpublishAdminAlbum(ctx, adminAlbumKG, adminAlbumOperator, 999999, "10.0.0.7", "ua-7")
	assert.ErrorIs(t, err, service.ErrAdminAlbumNotFound)
}

// TC-NEW-ALB-0019：真实路由表里只有列表/详情/下架；没有创建、编辑、删除端点。
// 该断言同时经过 NewRouter 的权限登记自检（新增 admin 路由必须登记权限码）。
func TestAdminAlbumRoutesHaveNoEditEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := handler.NewHandler(
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, // auth..settings
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, // admin..notice
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, // announcement..gardenCalendar
		nil, // timetable
		nil, // dietConfig
	)
	engine, err := app.NewRouter(app.NewAppHandler(h), nil, app.JWTSecret("test-secret"), nil)
	require.NoError(t, err, "admin 路由权限登记缺口会让 NewRouter 启动失败")

	var got []string
	for _, rt := range engine.Routes() {
		if strings.HasPrefix(rt.Path, "/api/v1/admin/albums") {
			got = append(got, rt.Method+" "+rt.Path)
		}
	}
	assert.ElementsMatch(t, []string{
		"GET /api/v1/admin/albums",
		"GET /api/v1/admin/albums/:id",
		"POST /api/v1/admin/albums/:id/unpublish",
	}, got)

	for _, rt := range engine.Routes() {
		if !strings.Contains(rt.Path, "/admin/albums") {
			continue
		}
		assert.NotContains(t, []string{"PUT", "PATCH", "DELETE"}, rt.Method,
			"管理端相册不得有编辑/删除端点：%s %s", rt.Method, rt.Path)
	}
}

// 夹具硬删 0 残留（与既有切片一致的可重复执行守卫）。
func TestAdminAlbumCleanupLeavesNoResidue(t *testing.T) {
	db, _ := newAdminAlbumFixture(t)
	cleanupAdminAlbumData(t, db)

	count := func(model any, query string, args ...any) int64 {
		var n int64
		require.NoError(t, db.Model(model).Where(query, args...).Count(&n).Error)
		return n
	}
	assert.Zero(t, count(&domain.Album{}, "id BETWEEN ? AND ?", adminAlbumPublishedA, uint64(998159)))
	assert.Zero(t, count(&domain.AlbumPhoto{}, "album_id BETWEEN ? AND ?", adminAlbumPublishedA, uint64(998159)))
	assert.Zero(t, count(&domain.AlbumStudent{}, "album_id BETWEEN ? AND ?", adminAlbumPublishedA, uint64(998159)))
	assert.Zero(t, count(&domain.AlbumPhotoStudent{}, "photo_id BETWEEN ? AND ?", adminAlbumPhotoBase, uint64(998169)))
	assert.Zero(t, count(&domain.OperationLog{}, "user_id IN (?, ?)", adminAlbumCreator, adminAlbumOperator))
	assert.Zero(t, count(&domain.Student{}, "id BETWEEN ? AND ?", adminAlbumStudentA, uint64(998139)))
	assert.Zero(t, count(&domain.User{}, "id BETWEEN ? AND ?", adminAlbumCreator, uint64(998129)))
	assert.Zero(t, count(&domain.Class{}, "id BETWEEN ? AND ?", uint64(998110), uint64(998119)))
	assert.Zero(t, count(&domain.Kindergarten{}, "id IN (?, ?)", adminAlbumKG, adminAlbumKGOther))
}
