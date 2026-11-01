package service

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

// OS-ALB-4：家长端班级相册（openspec class-album-parent-sharing 5.1–5.6）。
//
// 打真实 MySQL（本地 configs/app.ini），固定 9971xx 夹具，t.Cleanup 硬删 0 残留。
// 覆盖 TC-NEW-PCA-0001–0010：
//   - 0001/0002 只返回已发布、未软删、同园同班的相册；
//   - 0003/0004 「我的孩子」照片级过滤与相册级参与者兜底；
//   - 0005/0006 班内共享撤回整张隐藏，自己家长仍可见；
//   - 0007 跨班/跨园 404；0008 过滤后顺序不变；
//   - 0009/0010 毕业 90 天窗口与退学/无时间戳立即拒绝。
//
// 守卫用例（映射到同批 TC 或作为补充证据）：多监护人任一关闭、多孩按当前孩子、
// 列表查询次数不随相册数增长、清理后 0 残留。
//
// 建表走测试内 AutoMigrate：本切片不允许重启 8080，测试不能依赖后端启动时迁移。

const (
	testPCALKg      uint64 = 997101 // 目标园所
	testPCALKgOther uint64 = 997102 // 他园

	testPCALClassA uint64 = 997110 // 目标班
	testPCALClassB uint64 = 997111 // 同园别班
	testPCALClassO uint64 = 997112 // 他园班级

	testPCALParentA  uint64 = 997120 // 我的孩子 A 的家长（主视图）
	testPCALParentB  uint64 = 997121 // B 的家长
	testPCALParentC  uint64 = 997122 // C 的家长：关闭班内共享
	testPCALParentA2 uint64 = 997123 // A 的另一位监护人（未关闭）
	testPCALParentE1 uint64 = 997124 // E 的监护人 1（未关闭）
	testPCALParentE2 uint64 = 997125 // E 的监护人 2（关闭 → 任一关闭即关闭）
	testPCALParentR  uint64 = 997126 // 留存边界学生的家长
	testPCALParentO  uint64 = 997127 // 他园家长

	testPCALStudentA uint64 = 997130 // 我的孩子（班 A）
	testPCALStudentB uint64 = 997131
	testPCALStudentC uint64 = 997132 // 家长关闭班内共享
	testPCALStudentE uint64 = 997134 // 多监护人：E1 开、E2 关
	testPCALStudentF uint64 = 997135 // 同一家长的另一个孩子（班 B）
	testPCALStudentG uint64 = 997136 // 毕业 89 天
	testPCALStudentH uint64 = 997137 // 毕业 90 天（固定 now 单测）
	testPCALStudentI uint64 = 997138 // 毕业 91 天
	testPCALStudentJ uint64 = 997139 // 退学/转园
	testPCALStudentK uint64 = 997140 // 毕业但无 status_changed_at
	testPCALStudentL uint64 = 997141 // 待入学（0）：正常访问
	testPCALStudentY uint64 = 997142 // 他园幼儿

	testPCALAlbumMain    uint64 = 997150 // 已发布主相册：12 张
	testPCALAlbumDraft   uint64 = 997151
	testPCALAlbumUnpub   uint64 = 997152
	testPCALAlbumDeleted uint64 = 997153
	testPCALAlbumClassB  uint64 = 997154
	testPCALAlbumOtherKg uint64 = 997155
	testPCALAlbumEmpty   uint64 = 997156 // 已发布但无照片
	testPCALAlbumOptOut  uint64 = 997157 // 已发布：opt-out 合影 + 对照照片

	// 主相册照片 ID = testPCALMainPhotoBase + sort_num。
	testPCALMainPhotoBase uint64 = 997160
	testPCALPhotoDraft    uint64 = 997172
	testPCALPhotoUnpub    uint64 = 997173
	testPCALPhotoDeleted  uint64 = 997174
	testPCALPhotoClassB   uint64 = 997175
	testPCALPhotoOtherKg  uint64 = 997176
	// opt-out 专辑：pair 关联 {A(查看者), C(关闭共享)}，
	// multiGuardian 关联 {A(查看者), E(多监护人，E2 关闭)}，ownOnly 只关联 A。
	testPCALPhotoOptOutPair uint64 = 997177
	testPCALPhotoAOwnOnly   uint64 = 997178
	testPCALPhotoMultiGuard uint64 = 997179
)

const pcalDay = 24 * time.Hour

func pcalTimePtr(t time.Time) *time.Time { return &t }

func pcalMainPhoto(sortNum int) uint64 { return testPCALMainPhotoBase + uint64(sortNum) }

func cleanupPCALData(t *testing.T, db *gorm.DB) {
	t.Helper()
	del := func(sql string, args ...any) {
		if err := db.Exec(sql, args...).Error; err != nil {
			t.Logf("cleanup: %v", err)
		}
	}
	// 关联表先删：album_photo_students 没有 album_id，只能经照片子查询找回。
	del(`DELETE FROM album_photo_students WHERE photo_id BETWEEN ? AND ? OR photo_id IN
		(SELECT id FROM album_photos WHERE album_id BETWEEN ? AND ?)`,
		testPCALMainPhotoBase, uint64(997179), testPCALAlbumMain, uint64(997159))
	del("DELETE FROM album_students WHERE album_id BETWEEN ? AND ?",
		testPCALAlbumMain, uint64(997159))
	del("DELETE FROM album_photos WHERE id BETWEEN ? AND ? OR album_id BETWEEN ? AND ?",
		testPCALMainPhotoBase, uint64(997179), testPCALAlbumMain, uint64(997159))
	del("DELETE FROM albums WHERE id BETWEEN ? AND ?", testPCALAlbumMain, uint64(997159))
	del("DELETE FROM parent_settings WHERE user_id BETWEEN ? AND ?",
		testPCALParentA, uint64(997129))
	del("DELETE FROM student_parents WHERE student_id BETWEEN ? AND ? OR user_id BETWEEN ? AND ?",
		testPCALStudentA, uint64(997149), testPCALParentA, uint64(997129))
	del("DELETE FROM users WHERE id BETWEEN ? AND ?", testPCALParentA, uint64(997129))
	del("DELETE FROM students WHERE id BETWEEN ? AND ?", testPCALStudentA, uint64(997149))
	del("DELETE FROM classes WHERE id BETWEEN ? AND ?", uint64(997110), uint64(997119))
	del("DELETE FROM kindergartens WHERE id IN (?, ?)", testPCALKg, testPCALKgOther)
}

type pcalFixture struct {
	db  *gorm.DB
	svc *ParentPortalService

	pcA  *ParentPortalContext // 当前孩子 A（班 A）
	pcB  *ParentPortalContext // 当前孩子 B
	pcC  *ParentPortalContext // 当前孩子 C（其家长关闭班内共享）
	pcE1 *ParentPortalContext // 当前孩子 E，监护人 1
	pcE2 *ParentPortalContext // 当前孩子 E，监护人 2（关闭）
	pcF  *ParentPortalContext // 同一家长的另一个孩子 F（班 B）
	pcG  *ParentPortalContext // 毕业 89 天
	pcI  *ParentPortalContext // 毕业 91 天
	pcJ  *ParentPortalContext // 退学/转园
	pcK  *ParentPortalContext // 毕业但无时间戳
	pcL  *ParentPortalContext // 待入学
	pcO  *ParentPortalContext // 他园家长
}

// pcalMigrateOnce 每个进程只建一次表，避免每个用例重复 AutoMigrate。
var pcalMigrateOnce sync.Once

func newPCALFixture(t *testing.T) *pcalFixture {
	t.Helper()
	db := openCLPTestDB(t)
	pcalMigrateOnce.Do(func() {
		require.NoError(t, db.AutoMigrate(
			&domain.Kindergarten{}, &domain.Class{}, &domain.User{},
			&domain.Student{}, &domain.StudentParent{}, &domain.ParentSetting{},
			&domain.Album{}, &domain.AlbumPhoto{}, &domain.AlbumStudent{}, &domain.AlbumPhotoStudent{},
		))
	})
	cleanupPCALData(t, db)
	t.Cleanup(func() { cleanupPCALData(t, db) })

	require.NoError(t, db.Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: testPCALKg}, Name: "测试园所-家长班级相册",
	}).Error)
	require.NoError(t, db.Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: testPCALKgOther}, Name: "测试园所-家长班级相册他园",
	}).Error)
	for _, cl := range []struct {
		id   uint64
		kg   uint64
		name string
	}{
		{testPCALClassA, testPCALKg, "相册一班"},
		{testPCALClassB, testPCALKg, "相册二班"},
		{testPCALClassO, testPCALKgOther, "他园相册班"},
	} {
		require.NoError(t, db.Create(&domain.Class{
			BaseModel: domain.BaseModel{ID: cl.id}, KindergartenID: cl.kg,
			Name: cl.name, Grade: "小班", Status: 1,
		}).Error)
	}

	parents := []uint64{
		testPCALParentA, testPCALParentB, testPCALParentC, testPCALParentA2,
		testPCALParentE1, testPCALParentE2, testPCALParentR, testPCALParentO,
	}
	for _, id := range parents {
		require.NoError(t, db.Create(&domain.User{
			BaseModel: domain.BaseModel{ID: id},
			Nickname:  fmt.Sprintf("相册家长-%d", id),
			Phone:     fmt.Sprintf("1390001%04d", id%10000),
		}).Error)
	}

	now := time.Now()
	students := []struct {
		id       uint64
		kg       uint64
		classID  uint64
		name     string
		status   int8
		changed  *time.Time
		guardian []uint64
	}{
		{testPCALStudentA, testPCALKg, testPCALClassA, "幼儿A", 1, nil, []uint64{testPCALParentA, testPCALParentA2}},
		{testPCALStudentB, testPCALKg, testPCALClassA, "幼儿B", 1, nil, []uint64{testPCALParentB}},
		{testPCALStudentC, testPCALKg, testPCALClassA, "幼儿C", 1, nil, []uint64{testPCALParentC}},
		{testPCALStudentE, testPCALKg, testPCALClassA, "幼儿E", 1, nil, []uint64{testPCALParentE1, testPCALParentE2}},
		{testPCALStudentF, testPCALKg, testPCALClassB, "幼儿F", 1, nil, []uint64{testPCALParentA}},
		{testPCALStudentG, testPCALKg, testPCALClassA, "幼儿G", 2, pcalTimePtr(now.Add(-89 * pcalDay)), []uint64{testPCALParentR}},
		{testPCALStudentH, testPCALKg, testPCALClassA, "幼儿H", 2, pcalTimePtr(now.Add(-90 * pcalDay)), []uint64{testPCALParentR}},
		{testPCALStudentI, testPCALKg, testPCALClassA, "幼儿I", 2, pcalTimePtr(now.Add(-91 * pcalDay)), []uint64{testPCALParentR}},
		{testPCALStudentJ, testPCALKg, testPCALClassA, "幼儿J", 3, pcalTimePtr(now.Add(-time.Hour)), []uint64{testPCALParentR}},
		{testPCALStudentK, testPCALKg, testPCALClassA, "幼儿K", 2, nil, []uint64{testPCALParentR}},
		{testPCALStudentL, testPCALKg, testPCALClassA, "幼儿L", 0, nil, []uint64{testPCALParentR}},
		{testPCALStudentY, testPCALKgOther, testPCALClassO, "他园幼儿", 1, nil, []uint64{testPCALParentO}},
	}
	byID := make(map[uint64]*domain.Student, len(students))
	for i := range students {
		st := students[i]
		row := domain.Student{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: st.id}},
			KindergartenID:  st.kg, ClassID: st.classID, Name: st.name,
			Status: st.status, StatusChangedAt: st.changed,
			Birthday: time.Now().AddDate(-4, 0, 0), EnrollDate: now,
		}
		require.NoError(t, db.Create(&row).Error)
		byID[st.id] = &row
		for _, userID := range st.guardian {
			require.NoError(t, db.Create(&domain.StudentParent{
				StudentID: st.id, UserID: userID, Relation: domain.RelationMother,
			}).Error)
		}
	}

	// 只有 C 与 E2 明确关闭班内共享；其余家长没有设置行（design D6 默认开启）。
	for _, userID := range []uint64{testPCALParentC, testPCALParentE2} {
		require.NoError(t, db.Create(&domain.ParentSetting{
			UserID: userID, KindergartenID: testPCALKg,
			AllowGrowthShare: false, AllowClassShare: false, AllowExternalShare: false,
			ShowPhoneToTeacher: true,
		}).Error)
	}

	baseDate := teacherPortalDay(time.Now())
	type albumSeed struct {
		id, kg, classID uint64
		status          string
		dayOffset       int
		photoCount      int
		deleted         bool
	}
	for _, a := range []albumSeed{
		{testPCALAlbumMain, testPCALKg, testPCALClassA, domain.AlbumStatusPublished, 0, 12, false},
		{testPCALAlbumDraft, testPCALKg, testPCALClassA, domain.AlbumStatusDraft, -1, 1, false},
		{testPCALAlbumUnpub, testPCALKg, testPCALClassA, domain.AlbumStatusUnpublished, -2, 1, false},
		{testPCALAlbumDeleted, testPCALKg, testPCALClassA, domain.AlbumStatusPublished, -3, 1, true},
		{testPCALAlbumClassB, testPCALKg, testPCALClassB, domain.AlbumStatusPublished, -4, 1, false},
		{testPCALAlbumOtherKg, testPCALKgOther, testPCALClassO, domain.AlbumStatusPublished, -5, 1, false},
		{testPCALAlbumEmpty, testPCALKg, testPCALClassA, domain.AlbumStatusPublished, -6, 0, false},
		{testPCALAlbumOptOut, testPCALKg, testPCALClassA, domain.AlbumStatusPublished, -7, 3, false},
	} {
		album := domain.Album{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: a.id}},
			KindergartenID:  a.kg, ClassID: a.classID, CreatorID: testPCALParentR,
			Title:      fmt.Sprintf("相册-%d", a.id),
			Category:   domain.AlbumCategoryActivity,
			AlbumDate:  baseDate.AddDate(0, 0, a.dayOffset),
			CoverURL:   fmt.Sprintf("https://cdn.example.com/pcal/%d.jpg", a.id),
			PhotoCount: a.photoCount,
			Status:     a.status,
		}
		if a.status == domain.AlbumStatusPublished {
			at := baseDate
			album.PublishedAt = &at
			album.PublishedBy = testPCALParentR
		}
		if a.deleted {
			album.DeletedAt = gorm.DeletedAt{Time: time.Now(), Valid: true}
		}
		require.NoError(t, db.Create(&album).Error)
	}

	// 主相册 12 张照片：只有 5 张对 A 可见（0/2/5/8/10）。
	mainLinks := map[int][]uint64{
		0:  {testPCALStudentA, testPCALStudentG},
		1:  {testPCALStudentB},
		2:  {testPCALStudentA, testPCALStudentB},
		3:  {testPCALStudentB},
		4:  {testPCALStudentA, testPCALStudentC},
		6:  {testPCALStudentB},
		7:  {testPCALStudentE},
		9:  {testPCALStudentB},
		10: {testPCALStudentA},
		11: {testPCALStudentB},
	}
	for i := 0; i < 12; i++ {
		pcalCreatePhoto(t, db, pcalMainPhoto(i), testPCALAlbumMain, i)
		if ids := mainLinks[i]; len(ids) > 0 {
			pcalLinkStudents(t, db, testPCALKg, pcalMainPhoto(i), ids...)
		}
	}
	// 相册级参与者：无逐张关联的照片（5/6/8）按它兜底。
	pcalAddParticipants(t, db, testPCALKg, testPCALAlbumMain,
		testPCALStudentA, testPCALStudentB, testPCALStudentG)
	pcalAddParticipants(t, db, testPCALKg, testPCALAlbumEmpty, testPCALStudentA)

	// opt-out 专辑的反事实夹具：两张合影都把查看者 A 直接关联进来，
	// 去掉 opt-out 过滤时 A 一定会看到它们；ownOnly 是只关联 A 的对照照片。
	//   pair        {A, C}：C 自己关闭共享（任一关闭）。
	//   multiGuard  {A, E}：E 的两位监护人只关了一位（E2），任何「全部关闭才算关」
	//                        的实现都会让 A 看到它 —— 这正是多监护人口径的反事实。
	pcalCreatePhoto(t, db, testPCALPhotoOptOutPair, testPCALAlbumOptOut, 0)
	pcalLinkStudents(t, db, testPCALKg, testPCALPhotoOptOutPair, testPCALStudentA, testPCALStudentC)
	pcalCreatePhoto(t, db, testPCALPhotoMultiGuard, testPCALAlbumOptOut, 1)
	pcalLinkStudents(t, db, testPCALKg, testPCALPhotoMultiGuard, testPCALStudentA, testPCALStudentE)
	pcalCreatePhoto(t, db, testPCALPhotoAOwnOnly, testPCALAlbumOptOut, 2)
	pcalLinkStudents(t, db, testPCALKg, testPCALPhotoAOwnOnly, testPCALStudentA)
	pcalAddParticipants(t, db, testPCALKg, testPCALAlbumOptOut, testPCALStudentA)

	for _, seed := range []struct {
		albumID uint64
		photoID uint64
		student uint64
	}{
		{testPCALAlbumDraft, testPCALPhotoDraft, testPCALStudentA},
		{testPCALAlbumUnpub, testPCALPhotoUnpub, testPCALStudentA},
		{testPCALAlbumDeleted, testPCALPhotoDeleted, testPCALStudentA},
		{testPCALAlbumClassB, testPCALPhotoClassB, testPCALStudentF},
		{testPCALAlbumOtherKg, testPCALPhotoOtherKg, testPCALStudentY},
	} {
		kg := testPCALKg
		if seed.albumID == testPCALAlbumOtherKg {
			kg = testPCALKgOther
		}
		pcalCreatePhoto(t, db, seed.photoID, seed.albumID, 0)
		pcalLinkStudents(t, db, kg, seed.photoID, seed.student)
	}

	ctxFor := func(userID uint64, st *domain.Student, students []domain.Student) *ParentPortalContext {
		return &ParentPortalContext{
			UserID: userID, KindergartenID: st.KindergartenID,
			CurrentStudent: st, Students: students,
		}
	}
	a := byID[testPCALStudentA]
	f := byID[testPCALStudentF]
	multiChild := []domain.Student{*a, *f}

	return &pcalFixture{
		db:  db,
		svc: newParentPortalServiceWithDB(db),
		pcA: ctxFor(testPCALParentA, a, multiChild),
		pcB: ctxFor(testPCALParentB, byID[testPCALStudentB], []domain.Student{*byID[testPCALStudentB]}),
		pcC: ctxFor(testPCALParentC, byID[testPCALStudentC], []domain.Student{*byID[testPCALStudentC]}),
		pcE1: ctxFor(testPCALParentE1, byID[testPCALStudentE],
			[]domain.Student{*byID[testPCALStudentE]}),
		pcE2: ctxFor(testPCALParentE2, byID[testPCALStudentE],
			[]domain.Student{*byID[testPCALStudentE]}),
		pcF: ctxFor(testPCALParentA, f, multiChild),
		pcG: ctxFor(testPCALParentR, byID[testPCALStudentG],
			[]domain.Student{*byID[testPCALStudentG]}),
		pcI: ctxFor(testPCALParentR, byID[testPCALStudentI],
			[]domain.Student{*byID[testPCALStudentI]}),
		pcJ: ctxFor(testPCALParentR, byID[testPCALStudentJ],
			[]domain.Student{*byID[testPCALStudentJ]}),
		pcK: ctxFor(testPCALParentR, byID[testPCALStudentK],
			[]domain.Student{*byID[testPCALStudentK]}),
		pcL: ctxFor(testPCALParentR, byID[testPCALStudentL],
			[]domain.Student{*byID[testPCALStudentL]}),
		pcO: ctxFor(testPCALParentO, byID[testPCALStudentY],
			[]domain.Student{*byID[testPCALStudentY]}),
	}
}

func pcalCreatePhoto(t *testing.T, db *gorm.DB, id, albumID uint64, sortNum int) {
	t.Helper()
	require.NoError(t, db.Create(&domain.AlbumPhoto{
		BaseModel: domain.BaseModel{ID: id},
		AlbumID:   albumID,
		FileURL:   fmt.Sprintf("https://cdn.example.com/pcal/%d.jpg", id),
		ThumbURL:  fmt.Sprintf("https://cdn.example.com/pcal/%d_thumb.jpg", id),
		Width:     1200, Height: 900, SortNum: sortNum,
	}).Error)
}

func pcalLinkStudents(t *testing.T, db *gorm.DB, kg, photoID uint64, studentIDs ...uint64) {
	t.Helper()
	for _, sid := range studentIDs {
		require.NoError(t, db.Create(&domain.AlbumPhotoStudent{
			KindergartenID: kg, PhotoID: photoID, StudentID: sid,
		}).Error)
	}
}

func pcalAddParticipants(t *testing.T, db *gorm.DB, kg, albumID uint64, studentIDs ...uint64) {
	t.Helper()
	for _, sid := range studentIDs {
		require.NoError(t, db.Create(&domain.AlbumStudent{
			KindergartenID: kg, AlbumID: albumID, StudentID: sid,
		}).Error)
	}
}

func pcalListItems(t *testing.T, data map[string]any) []map[string]any {
	t.Helper()
	items, ok := data["list"].([]map[string]any)
	require.True(t, ok, "list 必须是对象数组")
	return items
}

func pcalAlbumIDs(t *testing.T, data map[string]any) []uint64 {
	t.Helper()
	ids := make([]uint64, 0)
	for _, item := range pcalListItems(t, data) {
		id, ok := item["id"].(uint64)
		require.True(t, ok, "相册 id 必须是 uint64")
		ids = append(ids, id)
	}
	return ids
}

func pcalListItem(t *testing.T, data map[string]any, albumID uint64) map[string]any {
	t.Helper()
	for _, item := range pcalListItems(t, data) {
		if id, _ := item["id"].(uint64); id == albumID {
			return item
		}
	}
	require.FailNowf(t, "列表中缺少相册", "album=%d", albumID)
	return nil
}

func pcalDetailPhotoIDs(t *testing.T, detail map[string]any) []uint64 {
	t.Helper()
	photos, ok := detail["photos"].([]map[string]any)
	require.True(t, ok, "photos 必须是对象数组")
	ids := make([]uint64, 0, len(photos))
	for _, p := range photos {
		id, ok := p["id"].(uint64)
		require.True(t, ok, "照片 id 必须是 uint64")
		ids = append(ids, id)
	}
	return ids
}

func pcalCount(t *testing.T, db *gorm.DB, model any, query string, args ...any) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Unscoped().Model(model).Where(query, args...).Count(&n).Error)
	return n
}

// TC-NEW-PCA-0001：本班已发布相册列表与详情都可见，列表只给元信息与可见张数。
func TestParentClassAlbumPublishedVisible(t *testing.T) {
	f := newPCALFixture(t)
	ctx := context.Background()

	data, err := f.svc.ListClassAlbums(ctx, f.pcA, 1, 20)
	require.NoError(t, err)
	require.EqualValues(t, 3, data["total"], "本班已发布相册：主相册 + 空相册 + opt-out 专辑")

	item := pcalListItem(t, data, testPCALAlbumMain)
	assert.Equal(t, 12, item["photo_count"], "photo_count 是相册总张数")
	assert.Equal(t, 5, item["visible_photo_count"])
	assert.Equal(t, []string{
		"https://cdn.example.com/pcal/997160.jpg",
		"https://cdn.example.com/pcal/997162.jpg",
		"https://cdn.example.com/pcal/997165.jpg",
	}, item["cover_urls"], "封面只取可见照片的前 3 张")
	assert.Equal(t, "活动", item["category_label"])
	assert.NotContains(t, item, "status", "家长端不暴露运营状态")

	detail, err := f.svc.GetClassAlbum(ctx, f.pcA, testPCALAlbumMain)
	require.NoError(t, err)
	assert.EqualValues(t, testPCALAlbumMain, detail["id"])
	assert.Equal(t, 5, detail["visible_photo_count"])
	assert.Len(t, pcalDetailPhotoIDs(t, detail), 5)
}

// TC-NEW-PCA-0002：草稿、已下架、软删相册不在列表里，详情 404；空相册只给元信息。
func TestParentClassAlbumOnlyPublishedVisible(t *testing.T) {
	f := newPCALFixture(t)
	ctx := context.Background()

	data, err := f.svc.ListClassAlbums(ctx, f.pcA, 1, 20)
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]uint64{testPCALAlbumMain, testPCALAlbumEmpty, testPCALAlbumOptOut},
		pcalAlbumIDs(t, data))

	for _, hidden := range []uint64{testPCALAlbumDraft, testPCALAlbumUnpub, testPCALAlbumDeleted} {
		_, err := f.svc.GetClassAlbum(ctx, f.pcA, hidden)
		require.ErrorIs(t, err, ErrParentNotFound, "相册 %d 不可见", hidden)
	}

	empty := pcalListItem(t, data, testPCALAlbumEmpty)
	assert.Equal(t, 0, empty["visible_photo_count"])
	assert.Equal(t, []string{}, empty["cover_urls"], "无可见照片时封面是空数组而不是 null")
}

// TC-NEW-PCA-0003：不含自己孩子的照片被过滤；有逐张关联时以逐张为准。
func TestParentClassAlbumFiltersByMyChild(t *testing.T) {
	f := newPCALFixture(t)
	ctx := context.Background()

	detail, err := f.svc.GetClassAlbum(ctx, f.pcA, testPCALAlbumMain)
	require.NoError(t, err)
	ids := pcalDetailPhotoIDs(t, detail)
	assert.Equal(t, []uint64{
		pcalMainPhoto(0), pcalMainPhoto(2), pcalMainPhoto(5), pcalMainPhoto(8), pcalMainPhoto(10),
	}, ids)
	assert.NotContains(t, ids, pcalMainPhoto(1), "只包含 B 的照片对 A 不可见")
	// 第 7 张只关联 E：A 看不到它是「我的孩子」过滤，不是 opt-out 证明。
	// 多监护人口径的反事实证据见 TestParentClassAlbumMultiGuardianAnyOptOut。
	assert.NotContains(t, ids, pcalMainPhoto(7), "只包含 E 的照片对 A 不可见")
}

// TC-NEW-PCA-0004：照片没有逐张关联时，按相册级参与者兜底。
func TestParentClassAlbumAlbumLevelFallback(t *testing.T) {
	f := newPCALFixture(t)
	ctx := context.Background()

	detail, err := f.svc.GetClassAlbum(ctx, f.pcA, testPCALAlbumMain)
	require.NoError(t, err)
	ids := pcalDetailPhotoIDs(t, detail)
	for _, idx := range []int{5, 8} {
		assert.Contains(t, ids, pcalMainPhoto(idx),
			"第 %d 张没有逐张关联，A 在相册级参与者中", idx)
	}
	// 逐张关联优先于相册级参与者：B 在参与者里，但第 1 张只关联 B。
	assert.NotContains(t, ids, pcalMainPhoto(1))
}

// TC-NEW-PCA-0005：幼儿 A 关闭共享后，其他家长看不到含 C 的合影；同相册其他照片仍可见。
func TestParentClassAlbumOptOutHidesWholePhoto(t *testing.T) {
	f := newPCALFixture(t)
	ctx := context.Background()

	idsA := pcalDetailPhotoIDs(t, requireDetail(t, f, f.pcA))
	assert.NotContains(t, idsA, pcalMainPhoto(4), "含 C 的合影对 A 整张隐藏")
	assert.Contains(t, idsA, pcalMainPhoto(0), "同相册其他照片仍可见——不能整本隐藏")

	detailB, err := f.svc.GetClassAlbum(ctx, f.pcB, testPCALAlbumMain)
	require.NoError(t, err)
	idsB := pcalDetailPhotoIDs(t, detailB)
	assert.Len(t, idsB, 8)
	assert.NotContains(t, idsB, pcalMainPhoto(4))

	itemB := pcalListItem(t, mustList(t, f, f.pcB), testPCALAlbumMain)
	assert.Equal(t, 8, itemB["visible_photo_count"])
	assert.NotEqual(t, 0, itemB["visible_photo_count"], "相册本身仍出现在列表里")

	// PPS-0004：重新开启后立即恢复可见——读路径每次现查设置，不做缓存。
	require.NoError(t, f.db.Model(&domain.ParentSetting{}).
		Where("user_id = ? AND kindergarten_id = ?", testPCALParentC, testPCALKg).
		Updates(map[string]any{"allow_class_share": true, "allow_growth_share": true}).Error)
	assert.Contains(t, pcalDetailPhotoIDs(t, requireDetail(t, f, f.pcA)), pcalMainPhoto(4))
}

// TC-NEW-PCA-0006：关闭共享的幼儿自己的家长仍能看到这些照片。
func TestParentClassAlbumOptOutOwnParentStillSees(t *testing.T) {
	f := newPCALFixture(t)

	detailC, err := f.svc.GetClassAlbum(context.Background(), f.pcC, testPCALAlbumMain)
	require.NoError(t, err)
	assert.Equal(t, []uint64{pcalMainPhoto(4)}, pcalDetailPhotoIDs(t, detailC),
		"C 只出现在第 4 张；即使关闭共享，自己的孩子照片仍可见")

	itemC := pcalListItem(t, mustList(t, f, f.pcC), testPCALAlbumMain)
	assert.Equal(t, 1, itemC["visible_photo_count"])
}

// TC-NEW-PCA-0005 补充：照片里既有查看者自己的孩子 A、也有关闭共享的 C。
// 这是反事实用例——A 与这张照片有直接关联，去掉 opt-out 过滤后 A 必然看到它，
// 因此「看不到合影、只看得到对照照片」只有在整张隐藏生效时才成立；不是
// 「没关联所以看不到」的平凡断言。C 的家长仍能看到同一张（自己的孩子始终可见）。
//
// 未用 E1/E2 做查看者：E 因监护人 E2 关闭共享而自身处于 opt-out 集合，
// {C, E} 会让 C 的家长也看不到，三条断言不可能同时成立。
func TestParentClassAlbumOptOutHidesPhotoForOtherFamily(t *testing.T) {
	f := newPCALFixture(t)
	ctx := context.Background()

	detailA, err := f.svc.GetClassAlbum(ctx, f.pcA, testPCALAlbumOptOut)
	require.NoError(t, err)
	assert.Equal(t, []uint64{testPCALPhotoAOwnOnly}, pcalDetailPhotoIDs(t, detailA),
		"{A,C} 合影因 C 关闭共享对 A 的家长整张隐藏；只关联 A 的对照照片仍可见")

	detailC, err := f.svc.GetClassAlbum(ctx, f.pcC, testPCALAlbumOptOut)
	require.NoError(t, err)
	assert.Equal(t, []uint64{testPCALPhotoOptOutPair}, pcalDetailPhotoIDs(t, detailC),
		"C 自己的孩子始终可见，即使关闭了共享")
}

// TC-NEW-PCA-0007：跨班/跨园相册 404，列表按孩子当前班级收敛。
func TestParentClassAlbumIsolation(t *testing.T) {
	f := newPCALFixture(t)
	ctx := context.Background()

	for _, tc := range []struct {
		name  string
		pc    *ParentPortalContext
		album uint64
	}{
		{"同园别班相册", f.pcA, testPCALAlbumClassB},
		{"他班家长读本班", f.pcF, testPCALAlbumMain},
		{"他园家长读本园", f.pcO, testPCALAlbumMain},
		{"本园家长读他园", f.pcA, testPCALAlbumOtherKg},
	} {
		_, err := f.svc.GetClassAlbum(ctx, tc.pc, tc.album)
		require.ErrorIs(t, err, ErrParentNotFound, tc.name)
	}

	assert.ElementsMatch(t,
		[]uint64{testPCALAlbumMain, testPCALAlbumEmpty, testPCALAlbumOptOut},
		pcalAlbumIDs(t, mustList(t, f, f.pcA)))
	assert.Equal(t, []uint64{testPCALAlbumOtherKg}, pcalAlbumIDs(t, mustList(t, f, f.pcO)))
}

// TC-NEW-PCA-0008：12 张里 5 张可见，顺序与老师创建（sort_num）一致。
func TestParentClassAlbumDetailKeepsOrder(t *testing.T) {
	f := newPCALFixture(t)

	ids := pcalDetailPhotoIDs(t, requireDetail(t, f, f.pcA))
	require.Len(t, ids, 5)
	for i := 1; i < len(ids); i++ {
		assert.Less(t, ids[i-1], ids[i], "过滤只删减，不改变相对顺序")
	}
}

// TC-NEW-PCA-0009：毕业未满 90 天仍可查看；90 天整是窗口内、超过立即拒绝。
func TestParentClassAlbumGraduationWindow(t *testing.T) {
	f := newPCALFixture(t)
	ctx := context.Background()

	item := pcalListItem(t, mustList(t, f, f.pcG), testPCALAlbumMain)
	assert.Equal(t, 3, item["visible_photo_count"], "G 在参与者与第 0 张关联里：0/5/8")

	detailG, err := f.svc.GetClassAlbum(ctx, f.pcG, testPCALAlbumMain)
	require.NoError(t, err)
	assert.Contains(t, pcalDetailPhotoIDs(t, detailG), pcalMainPhoto(0))

	now := time.Now()
	graduated := func(ago time.Duration) *domain.Student {
		return &domain.Student{Status: parentStudentStatusGraduated, StatusChangedAt: pcalTimePtr(now.Add(-ago))}
	}
	assert.True(t, parentAlbumAccessAllowed(graduated(89*pcalDay), now))
	assert.True(t, parentAlbumAccessAllowed(graduated(90*pcalDay), now), "90 天整仍在窗口内")
	assert.False(t, parentAlbumAccessAllowed(graduated(90*pcalDay+time.Nanosecond), now))
	assert.False(t, parentAlbumAccessAllowed(graduated(91*pcalDay), now))
}

// TC-NEW-PCA-0010：毕业超过 90 天、退学/转园、无 status_changed_at 一律列表为空、详情 404。
func TestParentClassAlbumRetentionDenied(t *testing.T) {
	f := newPCALFixture(t)
	ctx := context.Background()

	for _, tc := range []struct {
		name string
		pc   *ParentPortalContext
	}{
		{"毕业 91 天", f.pcI},
		{"退学/转园", f.pcJ},
		{"毕业但无时间戳", f.pcK},
	} {
		data, err := f.svc.ListClassAlbums(ctx, tc.pc, 1, 20)
		require.NoError(t, err, tc.name)
		assert.Zero(t, data["total"], tc.name)
		assert.Empty(t, pcalListItems(t, data), tc.name)
		_, err = f.svc.GetClassAlbum(ctx, tc.pc, testPCALAlbumMain)
		require.ErrorIs(t, err, ErrParentNotFound, tc.name)
	}

	// 待入学（0）与在读（1）不受留存窗口限制。
	assert.Contains(t, pcalAlbumIDs(t, mustList(t, f, f.pcL)), testPCALAlbumMain)

	// 状态 3 没有任何时间戳豁免；状态 2 缺时间戳不放行。
	assert.False(t, parentAlbumAccessAllowed(&domain.Student{Status: parentStudentStatusLeft}, time.Now()))
	assert.False(t, parentAlbumAccessAllowed(&domain.Student{Status: parentStudentStatusGraduated}, time.Now()))
}

// 守卫：同一幼儿多监护人时任一关闭即关闭；自己家长（开或关）仍能看到自己孩子。
//
// 反事实：{A,E} 照片与查看者 A 直接关联，且 E 的监护人只有 E2 关闭（E1 仍开）。
// 若忽略 opt-out、或把「任一关闭」误实现成「全部关闭」，A 都会看到这张照片；
// 因此下面的整张隐藏断言只有在任一关闭生效时才成立。
func TestParentClassAlbumMultiGuardianAnyOptOut(t *testing.T) {
	f := newPCALFixture(t)
	ctx := context.Background()

	detailA, err := f.svc.GetClassAlbum(ctx, f.pcA, testPCALAlbumOptOut)
	require.NoError(t, err)
	idsA := pcalDetailPhotoIDs(t, detailA)
	assert.NotContains(t, idsA, testPCALPhotoMultiGuard, "{A,E} 因 E 的一位监护人关闭而整张隐藏")
	assert.NotContains(t, idsA, testPCALPhotoOptOutPair, "{A,C} 因 C 关闭而整张隐藏")
	assert.Equal(t, []uint64{testPCALPhotoAOwnOnly}, idsA, "对照照片仍可见，相册本身未被整本隐藏")

	for _, pc := range []*ParentPortalContext{f.pcE1, f.pcE2} {
		detailE, err := f.svc.GetClassAlbum(ctx, pc, testPCALAlbumOptOut)
		require.NoError(t, err)
		assert.Equal(t, []uint64{testPCALPhotoMultiGuard}, pcalDetailPhotoIDs(t, detailE),
			"E 自己的孩子始终可见，即使一位监护人关闭共享")
	}
}

// 守卫：本家长的孩子始终可见；「整张隐藏」只针对别的家长的孩子。
func TestParentClassAlbumOwnChildAlwaysVisible(t *testing.T) {
	photos := []domain.AlbumPhoto{{BaseModel: domain.BaseModel{ID: 1}}}
	links := map[uint64][]uint64{1: {10, 20}}
	optOut := map[uint64]bool{10: true}

	// 当前孩子 20，10 是同胞且关闭了共享：本家长仍应看到这张合影。
	ownFamily := parentVisibleAlbumPhotos(photos, links, nil, 20,
		map[uint64]bool{10: true, 20: true}, optOut)
	require.Len(t, ownFamily, 1)

	// 别的家长（自己的孩子只有 20）：10 关闭共享 → 整张隐藏。
	otherFamily := parentVisibleAlbumPhotos(photos, links, nil, 20,
		map[uint64]bool{20: true}, optOut)
	require.Len(t, otherFamily, 0)
}

// 守卫：多孩按当前孩子过滤，另一个孩子的班级相册不串。
func TestParentClassAlbumMultiChildUsesCurrentChild(t *testing.T) {
	f := newPCALFixture(t)
	ctx := context.Background()

	assert.ElementsMatch(t,
		[]uint64{testPCALAlbumMain, testPCALAlbumEmpty, testPCALAlbumOptOut},
		pcalAlbumIDs(t, mustList(t, f, f.pcA)))
	assert.Equal(t, []uint64{testPCALAlbumClassB}, pcalAlbumIDs(t, mustList(t, f, f.pcF)))

	detailF, err := f.svc.GetClassAlbum(ctx, f.pcF, testPCALAlbumClassB)
	require.NoError(t, err)
	assert.Equal(t, []uint64{testPCALPhotoClassB}, pcalDetailPhotoIDs(t, detailF))

	_, err = f.svc.GetClassAlbum(ctx, f.pcF, testPCALAlbumMain)
	require.ErrorIs(t, err, ErrParentNotFound)
}

// 守卫：列表查询次数固定 6，不随相册/照片数量增长（无 N+1）。
func TestParentClassAlbumList_NoNPlusOne(t *testing.T) {
	f := newPCALFixture(t)
	ctx := context.Background()

	counter := &noticeQueryCounter{} // SQL 计数器在 parent_notice_fields_test.go 中定义，同包复用
	svc := newParentPortalServiceWithDB(f.db.Session(&gorm.Session{Logger: counter}))

	_, err := svc.ListClassAlbums(ctx, f.pcA, 1, 20)
	require.NoError(t, err)
	fullPage := counter.value()
	t.Logf("整页查询数：%d", fullPage)
	assert.LessOrEqual(t, fullPage, int64(6),
		"Count/相册/照片/照片级关联/相册级参与者/opt-out 各一次")

	counter.reset()
	_, err = svc.ListClassAlbums(ctx, f.pcA, 1, 1)
	require.NoError(t, err)
	assert.Equal(t, fullPage, counter.value(), "整页与单条查询数相同 —— 没有逐相册查询")
}

// 守卫：清理后 9971xx 夹具 0 残留。
func TestParentClassAlbum_CleanupLeavesNoResidue(t *testing.T) {
	f := newPCALFixture(t)
	ctx := context.Background()

	_, err := f.svc.ListClassAlbums(ctx, f.pcA, 1, 20)
	require.NoError(t, err)
	_, err = f.svc.GetClassAlbum(ctx, f.pcA, testPCALAlbumMain)
	require.NoError(t, err)

	cleanupPCALData(t, f.db)

	assert.Zero(t, pcalCount(t, f.db, &domain.Album{}, "id BETWEEN ? AND ?",
		testPCALAlbumMain, uint64(997159)), "相册残留")
	assert.Zero(t, pcalCount(t, f.db, &domain.AlbumPhoto{}, "id BETWEEN ? AND ?",
		testPCALMainPhotoBase, uint64(997179)), "照片残留")
	assert.Zero(t, pcalCount(t, f.db, &domain.AlbumStudent{}, "album_id BETWEEN ? AND ?",
		testPCALAlbumMain, uint64(997159)), "相册级参与者残留")
	assert.Zero(t, pcalCount(t, f.db, &domain.AlbumPhotoStudent{}, "photo_id BETWEEN ? AND ?",
		testPCALMainPhotoBase, uint64(997179)), "照片级关联残留")
	assert.Zero(t, pcalCount(t, f.db, &domain.ParentSetting{}, "user_id BETWEEN ? AND ?",
		testPCALParentA, uint64(997129)), "设置残留")
	assert.Zero(t, pcalCount(t, f.db, &domain.StudentParent{}, "student_id BETWEEN ? AND ?",
		testPCALStudentA, uint64(997149)), "绑定残留")
	assert.Zero(t, pcalCount(t, f.db, &domain.Student{}, "id BETWEEN ? AND ?",
		testPCALStudentA, uint64(997149)), "学生残留")
	assert.Zero(t, pcalCount(t, f.db, &domain.User{}, "id BETWEEN ? AND ?",
		testPCALParentA, uint64(997129)), "用户残留")
	assert.Zero(t, pcalCount(t, f.db, &domain.Class{}, "id BETWEEN ? AND ?",
		uint64(997110), uint64(997119)), "班级残留")
	assert.Zero(t, pcalCount(t, f.db, &domain.Kindergarten{}, "id IN (?, ?)",
		testPCALKg, testPCALKgOther), "园所残留")
}

func mustList(t *testing.T, f *pcalFixture, pc *ParentPortalContext) map[string]any {
	t.Helper()
	data, err := f.svc.ListClassAlbums(context.Background(), pc, 1, 20)
	require.NoError(t, err)
	return data
}

func requireDetail(t *testing.T, f *pcalFixture, pc *ParentPortalContext) map[string]any {
	t.Helper()
	detail, err := f.svc.GetClassAlbum(context.Background(), pc, testPCALAlbumMain)
	require.NoError(t, err)
	return detail
}
