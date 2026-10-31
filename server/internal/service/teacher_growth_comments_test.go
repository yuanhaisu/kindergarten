package service

// 老师端成长评论（B10）的嵌套、家长称呼、回复层级与评论赞回归。
//
// 修复前 GrowthComments 返回平铺 list：子回复混在顶层，且没有 relation/student_name/replies/likes，
// 老师端详情页无法按设计 06 渲染「关系标签 + 老师回复 + 评论赞」。
// 契约见 backend/api/api-teacher/06-growth.md 的评论段。

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
)

// 独立 ID 段，避开 B05 的 9902xx、相册夹具的 9903xx 与开发库既有数据。
const (
	testTCKG        uint64 = 990401
	testTCClassA    uint64 = 990402
	testTCClassB    uint64 = 990403
	testTCStudentA  uint64 = 990404
	testTCParent    uint64 = 990405
	testTCTeacherA  uint64 = 990406
	testTCTeacherA2 uint64 = 990407
	testTCTeacherB  uint64 = 990408
	testTCLogA      uint64 = 990409
	testTCLogB      uint64 = 990410
	testTCPageLog   uint64 = 990411

	testTCCommentTop1   uint64 = 990420
	testTCCommentReply1 uint64 = 990421
	testTCCommentTop2   uint64 = 990422
	testTCCommentOnLogB uint64 = 990423
	testTCCommentOrphan uint64 = 990424

	// 分页回归的 25 条顶层 + 2 条回复，独立 ID 段。
	testTCPageCommentBase uint64 = 990500
)

var (
	teacherTestLogIDs     = []uint64{testTCLogA, testTCLogB, testTCPageLog}
	teacherTestUserIDs    = []uint64{testTCParent, testTCTeacherA, testTCTeacherA2, testTCTeacherB}
	teacherTestCommentIDs = []uint64{
		testTCCommentTop1, testTCCommentReply1, testTCCommentTop2,
		testTCCommentOnLogB, testTCCommentOrphan,
	}
)

// cleanTeacherCommentsFixture 按 ID 硬删。软删行会占住唯一索引并污染重跑，一律 Unscoped。
func cleanTeacherCommentsFixture(db *gorm.DB) {
	db.Unscoped().Where("comment_id IN ?", teacherTestCommentIDs).Delete(&domain.GrowthCommentLike{})
	db.Unscoped().Where("user_id IN ?", teacherTestUserIDs).Delete(&domain.GrowthCommentLike{})
	db.Unscoped().Where("log_id IN ?", teacherTestLogIDs).Delete(&domain.GrowthComment{})
	db.Unscoped().Where("id IN ?", teacherTestCommentIDs).Delete(&domain.GrowthComment{})
	db.Unscoped().Where("log_id IN ?", teacherTestLogIDs).Delete(&domain.GrowthLogStudent{})
	db.Unscoped().Where("id IN ?", teacherTestLogIDs).Delete(&domain.GrowthDailyLog{})
	db.Unscoped().Where("student_id = ?", testTCStudentA).Delete(&domain.StudentParent{})
	db.Unscoped().Where("user_id IN ? AND kindergarten_id = ?", teacherTestUserIDs, testTCKG).Delete(&domain.UserRole{})
	db.Unscoped().Where("id = ?", testTCStudentA).Delete(&domain.Student{})
	db.Unscoped().Where("id IN ?", []uint64{testTCClassA, testTCClassB}).Delete(&domain.Class{})
	db.Unscoped().Where("id IN ?", teacherTestUserIDs).Delete(&domain.User{})
	db.Unscoped().Where("id = ?", testTCKG).Delete(&domain.Kindergarten{})
}

type teacherCommentsFixture struct {
	db  *gorm.DB
	svc *TeacherPortalService
	tcA *TeacherPortalContext
	tcB *TeacherPortalContext
}

func setupTeacherCommentsFixture(t *testing.T) *teacherCommentsFixture {
	t.Helper()
	db := openHomeFeedTestDB(t)
	cleanTeacherCommentsFixture(db)
	t.Cleanup(func() { cleanTeacherCommentsFixture(db) })

	// 开发中的后端进程可能仍在跑旧 schema（加表后没重启）。生产路径是 provider.go 的
	// AutoMigrate；这里补建只为让测试不依赖重启后端。
	require.NoError(t, db.AutoMigrate(&domain.GrowthCommentLike{}))

	require.NoError(t, db.Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: testTCKG}, Name: "测试园所-老师评论",
	}).Error)
	for _, cls := range []domain.Class{
		{BaseModel: domain.BaseModel{ID: testTCClassA}, KindergartenID: testTCKG, Name: "评论回归A班", Grade: "小班", Status: 1},
		{BaseModel: domain.BaseModel{ID: testTCClassB}, KindergartenID: testTCKG, Name: "评论回归B班", Grade: "大班", Status: 1},
	} {
		require.NoError(t, db.Create(&cls).Error)
	}
	for _, u := range []domain.User{
		{BaseModel: domain.BaseModel{ID: testTCParent}, Nickname: "评论回归家长", PasswordHash: "x"},
		{BaseModel: domain.BaseModel{ID: testTCTeacherA}, Nickname: "评论回归老师A", PasswordHash: "x"},
		{BaseModel: domain.BaseModel{ID: testTCTeacherA2}, Nickname: "评论回归老师A2", PasswordHash: "x"},
		{BaseModel: domain.BaseModel{ID: testTCTeacherB}, Nickname: "评论回归老师B", PasswordHash: "x"},
	} {
		require.NoError(t, db.Create(&u).Error)
	}

	// 开发库是严格模式：time.Time 零值会被 MySQL 拒绝，生日/入园日必须给真值。
	birthday := time.Date(2020, 3, 1, 0, 0, 0, 0, time.Local)
	require.NoError(t, db.Create(&domain.Student{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testTCStudentA}},
		KindergartenID:  testTCKG, ClassID: testTCClassA, Name: "评论学生A", Status: 1,
		Birthday: birthday, EnrollDate: birthday,
	}).Error)
	require.NoError(t, db.Create(&domain.StudentParent{
		StudentID: testTCStudentA, UserID: testTCParent, Relation: domain.RelationMother, IsPrimary: true,
	}).Error)

	day := time.Date(2026, 3, 19, 0, 0, 0, 0, time.Local)
	for _, l := range []domain.GrowthDailyLog{
		{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testTCLogA}},
			KindergartenID: testTCKG, ClassID: testTCClassA, CreatorID: testTCTeacherA, Date: day, TargetType: 1, Content: "A班全班记录"},
		{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testTCLogB}},
			KindergartenID: testTCKG, ClassID: testTCClassB, CreatorID: testTCTeacherB, Date: day, TargetType: 1, Content: "B班全班记录"},
	} {
		require.NoError(t, db.Create(&l).Error)
	}

	at := func(min int) time.Time { return time.Date(2026, 3, 19, 10, min, 0, 0, time.Local) }
	replyToTop1 := testTCCommentTop1
	for _, c := range []domain.GrowthComment{
		{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testTCCommentTop1, CreatedAt: at(0)}},
			LogID: testTCLogA, UserID: testTCParent, Content: "谢谢老师的表扬"},
		{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testTCCommentReply1, CreatedAt: at(5)}},
			LogID: testTCLogA, UserID: testTCTeacherA, Content: "小明今天表现确实很棒", ParentID: &replyToTop1},
		{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testTCCommentTop2, CreatedAt: at(10)}},
			LogID: testTCLogA, UserID: testTCParent, Content: "回家还一直说美术课"},
		{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testTCCommentOnLogB, CreatedAt: at(15)}},
			LogID: testTCLogB, UserID: testTCParent, Content: "B班的评论"},
	} {
		require.NoError(t, db.Create(&c).Error)
	}

	svc := NewTeacherPortalService(db, nil, nil, nil, nil, nil, config.FaceConfig{})
	return &teacherCommentsFixture{
		db:  db,
		svc: svc,
		tcA: &TeacherPortalContext{UserID: testTCTeacherA, KindergartenID: testTCKG, ClassID: testTCClassA},
		tcB: &TeacherPortalContext{UserID: testTCTeacherB, KindergartenID: testTCKG, ClassID: testTCClassB},
	}
}

func tcCommentList(t *testing.T, data map[string]any) []map[string]any {
	t.Helper()
	raw, ok := data["list"]
	require.True(t, ok, "评论响应必须含 list")
	list, ok := raw.([]map[string]any)
	require.True(t, ok, "list 必须是对象数组")
	return list
}

func tcCommentReplies(t *testing.T, c map[string]any) []map[string]any {
	t.Helper()
	raw, ok := c["replies"]
	require.True(t, ok, "顶层评论必须带 replies 数组")
	replies, ok := raw.([]map[string]any)
	require.True(t, ok, "replies 必须是对象数组")
	return replies
}

// 契约核心：顶层 + 一层 replies、total 含回复、家长作者带称呼、老师作者不带、每条 id 恰好一次。
func TestTeacherGrowthComments_NestsRepliesAndRelation(t *testing.T) {
	f := setupTeacherCommentsFixture(t)
	ctx := context.Background()

	data, err := f.svc.GrowthComments(ctx, f.tcA, testTCLogA, 1, 20)
	require.NoError(t, err)

	list := tcCommentList(t, data)
	require.Len(t, list, 2, "顶层只应有两条评论，回复不得混入顶层")
	assert.Equal(t, testTCCommentTop1, list[0]["id"], "顶层按 created_at,id 正序")
	assert.Equal(t, testTCCommentTop2, list[1]["id"])
	assert.Equal(t, 3, data["total"], "total 是该记录的评论总数（含回复）")

	replies := tcCommentReplies(t, list[0])
	require.Len(t, replies, 1, "第一条顶层评论下应挂着老师回复")
	assert.Equal(t, testTCCommentReply1, replies[0]["id"])
	parentID, ok := replies[0]["parent_id"].(*uint64)
	require.True(t, ok, "回复必须保留 parent_id")
	assert.Equal(t, testTCCommentTop1, *parentID)

	// 家长作者带 relation / student_name（按记录所在班级取该家长的孩子）。
	parentUser := list[0]["user"].(map[string]any)
	assert.Equal(t, "mother", parentUser["relation"])
	assert.Equal(t, "评论学生A", parentUser["student_name"])

	// 老师没有 StudentParent 行：按契约省略这两字段，不报错。
	teacherUser := replies[0]["user"].(map[string]any)
	_, hasRelation := teacherUser["relation"]
	_, hasStudent := teacherUser["student_name"]
	assert.False(t, hasRelation, "教师作者不应凭空出现 relation")
	assert.False(t, hasStudent)

	// 评论赞字段：初始 0 / false，顶层与回复都有。
	for _, c := range append([]map[string]any{list[0], list[1]}, replies...) {
		assert.Equal(t, int64(0), c["likes_count"])
		assert.Equal(t, false, c["liked"])
	}

	// 不重复：每个 id 在顶层与回复里恰好出现一次。
	seen := map[uint64]int{}
	for _, c := range list {
		seen[c["id"].(uint64)]++
		for _, r := range tcCommentReplies(t, c) {
			seen[r["id"].(uint64)]++
		}
	}
	for _, id := range []uint64{testTCCommentTop1, testTCCommentReply1, testTCCommentTop2} {
		assert.Equal(t, 1, seen[id], "评论 %d 必须恰好出现一次", id)
	}
}

// 父评论已被删除/不存在时，回复不能凭空消失：提升为顶层。
func TestTeacherGrowthComments_OrphanReplyIsNotLost(t *testing.T) {
	f := setupTeacherCommentsFixture(t)
	missingParent := uint64(990499)
	require.NoError(t, f.db.Create(&domain.GrowthComment{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testTCCommentOrphan, CreatedAt: time.Now()}},
		LogID:           testTCLogA, UserID: testTCParent, Content: "父评论已不存在的回复", ParentID: &missingParent,
	}).Error)

	data, err := f.svc.GrowthComments(context.Background(), f.tcA, testTCLogA, 1, 20)
	require.NoError(t, err)

	list := tcCommentList(t, data)
	require.Len(t, list, 3, "孤儿回复提升为顶层，不得被丢弃")
	assert.Equal(t, testTCCommentOrphan, list[2]["id"])
	assert.Empty(t, tcCommentReplies(t, list[2]))
	assert.Equal(t, 4, data["total"], "total 含全部评论（含孤儿回复）")
}

// parent_id 规则：合法顶层父评论可回复；跨记录被拒；对回复再回复被拒；拒绝时不落库。
func TestTeacherCreateGrowthComment_ParentIDRules(t *testing.T) {
	f := setupTeacherCommentsFixture(t)
	ctx := context.Background()

	countOnLogA := func() int64 {
		var n int64
		require.NoError(t, f.db.Model(&domain.GrowthComment{}).Where("log_id = ?", testTCLogA).Count(&n).Error)
		return n
	}
	parentRef := func(v uint64) *uint64 { return &v }
	before := countOnLogA()

	data, err := f.svc.CreateGrowthComment(ctx, f.tcA, testTCLogA, " 好的，继续加油  ", parentRef(testTCCommentTop1))
	require.NoError(t, err, "回复本记录的顶层评论合法")
	assert.NotZero(t, data["id"])
	createdID := data["id"].(uint64)
	var created domain.GrowthComment
	require.NoError(t, f.db.First(&created, createdID).Error)
	assert.Equal(t, "好的，继续加油", created.Content, "content 去空白后落库")
	require.NotNil(t, created.ParentID)
	assert.Equal(t, testTCCommentTop1, *created.ParentID)
	after := countOnLogA()
	require.Equal(t, before+1, after)

	// 跨记录父评论：comment 990423 属于 B 班记录。
	_, err = f.svc.CreateGrowthComment(ctx, f.tcA, testTCLogA, "跨记录的回复", parentRef(testTCCommentOnLogB))
	assert.ErrorIs(t, err, ErrTeacherInvalidParams, "父评论不属于本条记录必须拒绝")
	assert.Equal(t, after, countOnLogA(), "被拒请求不得落库")

	// 对回复再回复：只允许一层。
	_, err = f.svc.CreateGrowthComment(ctx, f.tcA, testTCLogA, "多级回复", parentRef(testTCCommentReply1))
	assert.ErrorIs(t, err, ErrTeacherInvalidParams, "父评论本身已是回复，必须拒绝")
	assert.Equal(t, after, countOnLogA(), "被拒请求不得落库")
}

// 评论赞：首次 1、重复仍 1、另一账号 +1、取消回落，且不影响同记录其他评论。
func TestTeacherGrowthCommentLikes_IdempotentAndScoped(t *testing.T) {
	f := setupTeacherCommentsFixture(t)
	ctx := context.Background()
	tcA2 := &TeacherPortalContext{UserID: testTCTeacherA2, KindergartenID: testTCKG, ClassID: testTCClassA}

	data, err := f.svc.LikeGrowthComment(ctx, f.tcA, testTCCommentTop1)
	require.NoError(t, err)
	assert.Equal(t, true, data["liked"])
	assert.Equal(t, int64(1), data["likes_count"])

	data, err = f.svc.LikeGrowthComment(ctx, f.tcA, testTCCommentTop1)
	require.NoError(t, err, "重复点赞不报错")
	assert.Equal(t, int64(1), data["likes_count"], "重复点赞不重复计数")

	data, err = f.svc.LikeGrowthComment(ctx, tcA2, testTCCommentTop1)
	require.NoError(t, err)
	assert.Equal(t, int64(2), data["likes_count"], "另一账号点赞累加")

	data, err = f.svc.UnlikeGrowthComment(ctx, f.tcA, testTCCommentTop1)
	require.NoError(t, err)
	assert.Equal(t, false, data["liked"])
	assert.Equal(t, int64(1), data["likes_count"])

	data, err = f.svc.UnlikeGrowthComment(ctx, f.tcA, testTCCommentTop1)
	require.NoError(t, err, "未点赞时取消也成功")
	assert.Equal(t, int64(1), data["likes_count"])

	// GET 必须反映真实计数与「当前老师是否赞过」。
	data, err = f.svc.GrowthComments(ctx, f.tcA, testTCLogA, 1, 20)
	require.NoError(t, err)
	list := tcCommentList(t, data)
	require.Len(t, list, 2)
	assert.Equal(t, int64(1), list[0]["likes_count"])
	assert.Equal(t, false, list[0]["liked"], "老师A已取消，不应再显示已赞")
	assert.Equal(t, int64(0), list[1]["likes_count"], "不得影响同记录其他评论")
	assert.Equal(t, false, list[1]["liked"])

	data, err = f.svc.GrowthComments(ctx, tcA2, testTCLogA, 1, 20)
	require.NoError(t, err)
	list = tcCommentList(t, data)
	assert.Equal(t, true, list[0]["liked"], "老师A2赞过，按当前登录用户返回 liked")

	// 取消赞是硬删：表里不留软删行，否则软删行会占住 uk_comment_like_user。
	var softDeleted int64
	require.NoError(t, f.db.Unscoped().Model(&domain.GrowthCommentLike{}).
		Where("comment_id = ? AND deleted_at IS NOT NULL", testTCCommentTop1).Count(&softDeleted).Error)
	assert.Zero(t, softDeleted)
}

// 可见范围不因评论接口放宽：别班老师既列不出也赞不了。
func TestTeacherGrowthComments_ClassScope(t *testing.T) {
	f := setupTeacherCommentsFixture(t)
	ctx := context.Background()

	_, err := f.svc.GrowthComments(ctx, f.tcB, testTCLogA, 1, 20)
	assert.ErrorIs(t, err, ErrTeacherNotFound, "别班老师不得列出该记录评论")

	_, err = f.svc.LikeGrowthComment(ctx, f.tcB, testTCCommentTop1)
	assert.ErrorIs(t, err, ErrTeacherNotFound, "别班老师不得点赞该记录评论")

	_, err = f.svc.CreateGrowthComment(ctx, f.tcB, testTCLogA, "越权评论", nil)
	assert.ErrorIs(t, err, ErrTeacherNotFound, "别班老师不得评论该记录")

	var n int64
	require.NoError(t, f.db.Model(&domain.GrowthComment{}).Where("log_id = ?", testTCLogA).Count(&n).Error)
	assert.Equal(t, int64(3), n, "越权请求不得落库")

	// 软删的评论不可再被点赞：growthCommentRow 的 JOIN 必须带上 growth_comments.deleted_at IS NULL。
	require.NoError(t, f.db.Delete(&domain.GrowthComment{}, testTCCommentTop2).Error)
	_, err = f.svc.LikeGrowthComment(ctx, f.tcA, testTCCommentTop2)
	assert.ErrorIs(t, err, ErrTeacherNotFound, "已删除的评论不得再点赞")
}

// 清理证据：夹具硬删后库内 0 残留。
func TestTeacherGrowthComments_FixtureCleanupLeavesNoResidue(t *testing.T) {
	f := setupTeacherCommentsFixture(t)
	cleanTeacherCommentsFixture(f.db)

	count := func(model any, where string, args ...any) int64 {
		var n int64
		require.NoError(t, f.db.Unscoped().Model(model).Where(where, args...).Count(&n).Error)
		return n
	}
	assert.Zero(t, count(&domain.GrowthComment{}, "log_id IN ?", teacherTestLogIDs))
	assert.Zero(t, count(&domain.GrowthCommentLike{}, "comment_id IN ? OR user_id IN ?", teacherTestCommentIDs, teacherTestUserIDs))
	assert.Zero(t, count(&domain.GrowthDailyLog{}, "id IN ?", teacherTestLogIDs))
	assert.Zero(t, count(&domain.StudentParent{}, "student_id = ?", testTCStudentA))
	assert.Zero(t, count(&domain.Student{}, "id = ?", testTCStudentA))
	assert.Zero(t, count(&domain.Class{}, "id IN ?", []uint64{testTCClassA, testTCClassB}))
	assert.Zero(t, count(&domain.User{}, "id IN ?", teacherTestUserIDs))
	assert.Zero(t, count(&domain.Kindergarten{}, "id = ?", testTCKG))
}

// 分页只作用于顶层：每页返回 page_size 条顶层，回复随所属顶层一起返回；
// root_total 是顶层条数（含被提升的孤立回复），total 仍是含回复的总数。
func TestTeacherGrowthComments_PaginatesRoots(t *testing.T) {
	f := setupTeacherCommentsFixture(t)
	ctx := context.Background()

	require.NoError(t, f.db.Create(&domain.GrowthDailyLog{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testTCPageLog}},
		KindergartenID:  testTCKG, ClassID: testTCClassA, CreatorID: testTCTeacherA,
		Date: time.Date(2026, 3, 20, 0, 0, 0, 0, time.Local), TargetType: 1, Content: "分页回归记录",
	}).Error)

	const roots = 25
	base := time.Date(2026, 3, 20, 10, 0, 0, 0, time.Local)
	rootIDs := make([]uint64, 0, roots)
	for i := 0; i < roots; i++ {
		id := testTCPageCommentBase + uint64(i)
		rootIDs = append(rootIDs, id)
		require.NoError(t, f.db.Create(&domain.GrowthComment{
			SoftDeleteModel: domain.SoftDeleteModel{
				BaseModel: domain.BaseModel{ID: id, CreatedAt: base.Add(time.Duration(i) * time.Minute)},
			},
			LogID: testTCPageLog, UserID: testTCParent, Content: fmt.Sprintf("顶层评论 %d", i+1),
		}).Error)
	}
	// 一条回复挂第 1 页的顶层（第 1 条），一条挂第 2 页的顶层（第 11 条）。
	replyOnPage1 := testTCPageCommentBase + 25
	replyOnPage2 := testTCPageCommentBase + 26
	for i, row := range []struct {
		id     uint64
		parent uint64
		at     time.Time
	}{
		{replyOnPage1, rootIDs[0], base.Add(60 * time.Minute)},
		{replyOnPage2, rootIDs[10], base.Add(61 * time.Minute)},
	} {
		p := row.parent
		require.NoError(t, f.db.Create(&domain.GrowthComment{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: row.id, CreatedAt: row.at}},
			LogID:           testTCPageLog, UserID: testTCTeacherA, Content: fmt.Sprintf("回复 %d", i+1), ParentID: &p,
		}).Error)
	}

	page1, err := f.svc.GrowthComments(ctx, f.tcA, testTCPageLog, 1, 10)
	require.NoError(t, err)
	list1 := tcCommentList(t, page1)
	require.Len(t, list1, 10, "page1 取前 10 条顶层")
	assert.Equal(t, 25, page1["root_total"], "root_total 是顶层条数")
	assert.Equal(t, true, page1["has_more"])
	assert.Equal(t, 27, page1["total"], "total 含回复，不随分页变化")
	assert.Equal(t, rootIDs[0], list1[0]["id"], "顶层按 created_at,id 正序")
	replies1 := tcCommentReplies(t, list1[0])
	require.Len(t, replies1, 1, "第 1 页顶层的回复随顶层返回")
	assert.Equal(t, replyOnPage1, replies1[0]["id"])
	for _, c := range list1 {
		assert.NotEqual(t, rootIDs[10], c["id"], "第 2 页的顶层不得出现在第 1 页")
		assert.NotEqual(t, replyOnPage2, c["id"], "第 2 页顶层的回复不得出现在第 1 页")
	}

	page2, err := f.svc.GrowthComments(ctx, f.tcA, testTCPageLog, 2, 10)
	require.NoError(t, err)
	list2 := tcCommentList(t, page2)
	require.Len(t, list2, 10)
	assert.Equal(t, true, page2["has_more"])
	assert.Equal(t, 27, page2["total"])
	assert.Equal(t, rootIDs[10], list2[0]["id"], "第 2 页从第 11 条顶层开始")
	replies2 := tcCommentReplies(t, list2[0])
	require.Len(t, replies2, 1, "第 2 页顶层的回复必须随所属顶层返回")
	assert.Equal(t, replyOnPage2, replies2[0]["id"])

	page3, err := f.svc.GrowthComments(ctx, f.tcA, testTCPageLog, 3, 10)
	require.NoError(t, err)
	list3 := tcCommentList(t, page3)
	require.Len(t, list3, 5, "最后一页只剩 5 条顶层")
	assert.Equal(t, false, page3["has_more"])
	assert.Equal(t, 25, page3["root_total"])
	assert.Equal(t, rootIDs[24], list3[4]["id"])

	// 越界页：空 list、has_more=false，不报错。
	page99, err := f.svc.GrowthComments(ctx, f.tcA, testTCPageLog, 99, 10)
	require.NoError(t, err)
	list99 := tcCommentList(t, page99)
	assert.NotNil(t, list99, "越界也必须是 []，不能是 null")
	assert.Empty(t, list99)
	assert.Equal(t, 25, page99["root_total"])
	assert.Equal(t, false, page99["has_more"])
	assert.Equal(t, 27, page99["total"])

	// 极大 page：先判越界再相乘，不得溢出成负索引而 panic。
	pageMax, err := f.svc.GrowthComments(ctx, f.tcA, testTCPageLog, math.MaxInt, 10)
	require.NoError(t, err, "极大 page 不得报错")
	listMax := tcCommentList(t, pageMax)
	assert.NotNil(t, listMax, "极大 page 也必须是 []")
	assert.Empty(t, listMax)
	assert.Equal(t, 25, pageMax["root_total"])
	assert.Equal(t, false, pageMax["has_more"])
	assert.Equal(t, 27, pageMax["total"])

	// 清理并断言无残留。
	cleanTeacherCommentsFixture(f.db)
	var residue int64
	require.NoError(t, f.db.Unscoped().Model(&domain.GrowthComment{}).
		Where("log_id = ?", testTCPageLog).Count(&residue).Error)
	assert.Zero(t, residue, "硬删后分页夹具无残留")
}
