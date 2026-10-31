package service

// 成长评论（家长端）的嵌套与关系字段回归。
//
// 修复前 ListGrowthComments 返回扁平 list：子回复混在顶层（家长端会当成一级评论重复渲染），
// 且用户对象没有 relation —— detail.vue 的「妈妈/爸爸」标签永远不显示。
// 契约 GET /parent/growth/feed/:id/comments 要求「顶层评论 + replies[] + user.relation」。

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/domain"
)

const (
	testCommentTop1         uint64 = 990330
	testCommentReplyTeacher uint64 = 990331
	testCommentTop2         uint64 = 990332
	testCommentReplyParent  uint64 = 990333
	testCommentOrphan       uint64 = 990334
)

// seedComments 在相册夹具的 A 班记录上写入「两条顶层 + 各一条回复」。
func seedComments(t *testing.T) *ParentPortalService {
	t.Helper()
	db, svc := setupAlbumFixture(t)

	at := func(min int) time.Time { return time.Date(2026, 3, 19, 10, min, 0, 0, time.Local) }
	replyToTop1 := testCommentTop1
	replyToTop2 := testCommentTop2
	rows := []domain.GrowthComment{
		{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testCommentTop1, CreatedAt: at(0)}},
			LogID: testAlbumLogA, UserID: testAlbumParent, Content: "谢谢老师的表扬"},
		{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testCommentReplyTeacher, CreatedAt: at(5)}},
			LogID: testAlbumLogA, UserID: testAlbumTeacher, Content: "小明今天表现确实很棒", ParentID: &replyToTop1},
		{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testCommentTop2, CreatedAt: at(10)}},
			LogID: testAlbumLogA, UserID: testAlbumParent, Content: "回家还一直说美术课"},
		{SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testCommentReplyParent, CreatedAt: at(15)}},
			LogID: testAlbumLogA, UserID: testAlbumParent, Content: "补充一句：他很喜欢画画", ParentID: &replyToTop2},
	}
	for i := range rows {
		require.NoError(t, db.Create(&rows[i]).Error)
	}
	return svc
}

func commentList(t *testing.T, data map[string]any) []map[string]any {
	t.Helper()
	raw, ok := data["list"]
	require.True(t, ok, "评论响应必须含 list")
	list, ok := raw.([]map[string]any)
	require.True(t, ok, "list 必须是对象数组")
	return list
}

func commentUser(t *testing.T, c map[string]any) map[string]any {
	t.Helper()
	u, ok := c["user"].(map[string]any)
	require.True(t, ok, "评论必须带 user 对象")
	return u
}

func commentReplies(t *testing.T, c map[string]any) []map[string]any {
	t.Helper()
	raw, ok := c["replies"]
	require.True(t, ok, "顶层评论必须带 replies 数组")
	replies, ok := raw.([]map[string]any)
	require.True(t, ok, "replies 必须是对象数组")
	return replies
}

func TestParentGrowthComments_NestsRepliesAndRelation(t *testing.T) {
	svc := seedComments(t)
	ctx := context.Background()

	pc, err := svc.ResolveContext(ctx, testAlbumParent, testAlbumKG, ptrU64(testAlbumStudentA))
	require.NoError(t, err)
	data, err := svc.ListGrowthComments(ctx, pc, testAlbumLogA, 1, 20)
	require.NoError(t, err)

	list := commentList(t, data)
	require.Len(t, list, 2, "顶层只应有两条评论，回复不得混入顶层")
	assert.Equal(t, testCommentTop1, list[0]["id"], "顶层按创建时间正序")
	assert.Equal(t, testCommentTop2, list[1]["id"])
	assert.Equal(t, 4, data["total"], "total 仍是该记录下的评论总数（含回复）")

	replies1 := commentReplies(t, list[0])
	require.Len(t, replies1, 1, "第一条顶层评论下应挂着老师回复")
	assert.Equal(t, testCommentReplyTeacher, replies1[0]["id"])
	replyParentID, ok := replies1[0]["parent_id"].(*uint64)
	require.True(t, ok, "回复必须保留 parent_id（JSON 序列化为数字）")
	assert.Equal(t, testCommentTop1, *replyParentID)
	assert.Equal(t, "小明今天表现确实很棒", replies1[0]["content"])

	replies2 := commentReplies(t, list[1])
	require.Len(t, replies2, 1)
	assert.Equal(t, testCommentReplyParent, replies2[0]["id"])

	// 家长评论作者带 relation 与孩子姓名（按记录所在班级取当前家长的孩子）。
	parentUser := commentUser(t, list[0])
	assert.Equal(t, "mother", parentUser["relation"])
	assert.Equal(t, "相册学生A", parentUser["student_name"])

	// 老师没有 StudentParent 关系：按契约省略 relation / student_name，不报错。
	teacherUser := commentUser(t, replies1[0])
	_, hasRelation := teacherUser["relation"]
	_, hasStudent := teacherUser["student_name"]
	assert.False(t, hasRelation, "教师作者不应凭空出现 relation")
	assert.False(t, hasStudent)

	// 不重复：每个 id 在顶层与回复里恰好出现一次。
	seen := map[uint64]int{}
	for _, c := range list {
		seen[c["id"].(uint64)]++
		for _, r := range commentReplies(t, c) {
			seen[r["id"].(uint64)]++
		}
	}
	for _, id := range []uint64{testCommentTop1, testCommentReplyTeacher, testCommentTop2, testCommentReplyParent} {
		assert.Equal(t, 1, seen[id], "评论 %d 必须恰好出现一次", id)
	}
}

func TestParentGrowthComments_EmptyListIsArray(t *testing.T) {
	_, svc := setupAlbumFixture(t)
	ctx := context.Background()

	pc, err := svc.ResolveContext(ctx, testAlbumParent, testAlbumKG, ptrU64(testAlbumStudentB))
	require.NoError(t, err)
	data, err := svc.ListGrowthComments(ctx, pc, testAlbumLogB, 1, 20)
	require.NoError(t, err)

	list := commentList(t, data)
	assert.NotNil(t, list, "无评论时必须是 []，不能是 null")
	assert.Empty(t, list)
	assert.Equal(t, 0, data["total"])
}

// 父评论被删除/不存在时，回复不能凭空消失：提升为顶层，至少可见。
func TestParentGrowthComments_OrphanReplyIsNotLost(t *testing.T) {
	db, svc := setupAlbumFixture(t)
	missingParent := uint64(990399)
	require.NoError(t, db.Create(&domain.GrowthComment{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testCommentOrphan, CreatedAt: time.Now()}},
		LogID:           testAlbumLogA, UserID: testAlbumParent, Content: "父评论已不存在的回复", ParentID: &missingParent,
	}).Error)

	ctx := context.Background()
	pc, err := svc.ResolveContext(ctx, testAlbumParent, testAlbumKG, ptrU64(testAlbumStudentA))
	require.NoError(t, err)
	data, err := svc.ListGrowthComments(ctx, pc, testAlbumLogA, 1, 20)
	require.NoError(t, err)

	list := commentList(t, data)
	require.Len(t, list, 1)
	assert.Equal(t, testCommentOrphan, list[0]["id"])
}

// 家长端分页同样只切顶层：回复随所属顶层返回，root_total/has_more 与 total 语义分工明确。
func TestParentGrowthComments_PaginatesRoots(t *testing.T) {
	db, svc := setupAlbumFixture(t)
	ctx := context.Background()

	const baseID uint64 = 990340
	const roots = 25
	base := time.Date(2026, 3, 19, 10, 0, 0, 0, time.Local)
	rootIDs := make([]uint64, 0, roots)
	for i := 0; i < roots; i++ {
		id := baseID + uint64(i)
		rootIDs = append(rootIDs, id)
		require.NoError(t, db.Create(&domain.GrowthComment{
			SoftDeleteModel: domain.SoftDeleteModel{
				BaseModel: domain.BaseModel{ID: id, CreatedAt: base.Add(time.Duration(i) * time.Minute)},
			},
			LogID: testAlbumLogA, UserID: testAlbumParent, Content: fmt.Sprintf("顶层评论 %d", i+1),
		}).Error)
	}
	replyOnPage1 := baseID + 25
	replyOnPage2 := baseID + 26
	for i, row := range []struct {
		id     uint64
		parent uint64
		at     time.Time
	}{
		{replyOnPage1, rootIDs[0], base.Add(60 * time.Minute)},
		{replyOnPage2, rootIDs[10], base.Add(61 * time.Minute)},
	} {
		p := row.parent
		require.NoError(t, db.Create(&domain.GrowthComment{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: row.id, CreatedAt: row.at}},
			LogID:           testAlbumLogA, UserID: testAlbumParent, Content: fmt.Sprintf("回复 %d", i+1), ParentID: &p,
		}).Error)
	}

	pc, err := svc.ResolveContext(ctx, testAlbumParent, testAlbumKG, ptrU64(testAlbumStudentA))
	require.NoError(t, err)

	page1, err := svc.ListGrowthComments(ctx, pc, testAlbumLogA, 1, 10)
	require.NoError(t, err)
	list1 := commentList(t, page1)
	require.Len(t, list1, 10, "page1 取前 10 条顶层")
	assert.Equal(t, 25, page1["root_total"])
	assert.Equal(t, true, page1["has_more"])
	assert.Equal(t, 27, page1["total"], "total 含回复，不随分页变化")
	assert.Equal(t, rootIDs[0], list1[0]["id"])
	replies1 := commentReplies(t, list1[0])
	require.Len(t, replies1, 1)
	assert.Equal(t, replyOnPage1, replies1[0]["id"])
	for _, c := range list1 {
		assert.NotEqual(t, rootIDs[10], c["id"], "第 2 页的顶层不得出现在第 1 页")
		assert.NotEqual(t, replyOnPage2, c["id"], "第 2 页顶层的回复不得出现在第 1 页")
	}

	page2, err := svc.ListGrowthComments(ctx, pc, testAlbumLogA, 2, 10)
	require.NoError(t, err)
	list2 := commentList(t, page2)
	require.Len(t, list2, 10)
	assert.Equal(t, true, page2["has_more"])
	assert.Equal(t, rootIDs[10], list2[0]["id"])
	replies2 := commentReplies(t, list2[0])
	require.Len(t, replies2, 1, "第 2 页顶层的回复必须随所属顶层返回")
	assert.Equal(t, replyOnPage2, replies2[0]["id"])

	page3, err := svc.ListGrowthComments(ctx, pc, testAlbumLogA, 3, 10)
	require.NoError(t, err)
	list3 := commentList(t, page3)
	require.Len(t, list3, 5)
	assert.Equal(t, false, page3["has_more"])
	assert.Equal(t, rootIDs[24], list3[4]["id"])

	page99, err := svc.ListGrowthComments(ctx, pc, testAlbumLogA, 99, 10)
	require.NoError(t, err)
	list99 := commentList(t, page99)
	assert.NotNil(t, list99, "越界也必须是 []，不能是 null")
	assert.Empty(t, list99)
	assert.Equal(t, 25, page99["root_total"])
	assert.Equal(t, false, page99["has_more"])
	assert.Equal(t, 27, page99["total"])

	cleanAlbumFixture(db)
	var residue int64
	require.NoError(t, db.Unscoped().Model(&domain.GrowthComment{}).
		Where("log_id IN ?", []uint64{testAlbumLogA, testAlbumLogB, testAlbumLogAInd, testAlbumLogBInd}).
		Count(&residue).Error)
	assert.Zero(t, residue, "硬删后分页夹具无残留")
}
