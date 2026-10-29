package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

// 覆盖 openspec/changes/notice-drafts-attachments/specs/notice-authoring/spec.md 的 12 个
// Scenario（TC-NEW-NOT-0001 ~ 0012）：草稿完整性、草稿仅创建者可见、定时时间、
// 状态机、附件落库/上限、删除撤回规则、家长确认状态。
// 复用 notice_multitarget_test.go 的夹具（园所 990201，老师 990201，班级 990202/990203）。

// testDraftPeer 同班第二位老师：用来验证「他人草稿不可见」。
const testDraftPeer uint64 = 990203

type noticeDraftFixture struct {
	*nmtFixture
	peerTC *TeacherPortalContext
}

// newNoticeDraftFixture 在 nmtFixture 之上补一位同班老师，并接管 notice_attachments 的清理。
// 附件表是 BaseModel（无软删），newNMTCleanup 不覆盖它，不硬删会跨轮残留。
func newNoticeDraftFixture(t *testing.T) *noticeDraftFixture {
	t.Helper()
	f := newNMTFixture(t)
	ctx := context.Background()

	purgeAttachments := func() {
		if err := f.db.WithContext(ctx).Exec(
			"DELETE FROM notice_attachments WHERE notice_id IN (SELECT id FROM notices WHERE kindergarten_id = ?)",
			testNMTKG).Error; err != nil {
			t.Logf("cleanup attachments: %v", err)
		}
	}
	purgeAttachments()
	t.Cleanup(func() {
		// 先附件后账号：newNMTFixture 的清理在之后执行，届时 notices 仍在，子查询能命中。
		purgeAttachments()
		f.db.WithContext(ctx).Exec("DELETE FROM class_teachers WHERE user_id = ?", testDraftPeer)
		f.db.WithContext(ctx).Exec("DELETE FROM users WHERE id = ?", testDraftPeer)
	})

	var peerCount int64
	f.db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", testDraftPeer).Count(&peerCount)
	if peerCount == 0 {
		require.NoError(t, f.db.WithContext(ctx).Create(&domain.User{
			BaseModel: domain.BaseModel{ID: testDraftPeer}, Phone: "13900002003", Nickname: "同班另一位老师",
		}).Error)
	}
	require.NoError(t, f.db.WithContext(ctx).Create(&domain.ClassTeacher{
		ClassID: f.classA, UserID: testDraftPeer, Title: "teacher", Status: 1,
	}).Error)

	return &noticeDraftFixture{
		nmtFixture: f,
		peerTC:     &TeacherPortalContext{UserID: testDraftPeer, KindergartenID: testNMTKG, ClassID: f.classA},
	}
}

func noticeStatusPtr(v int8) *int8 { return &v }

func attachmentInputs(n int) []TeacherNoticeAttachmentInput {
	out := make([]TeacherNoticeAttachmentInput, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, TeacherNoticeAttachmentInput{
			FileURL:  fmt.Sprintf("https://cdn.example.com/uploads/notice/draft-%d.jpg", i),
			FileType: "image",
			FileName: fmt.Sprintf("附件%d.jpg", i),
			FileSize: 512000,
		})
	}
	return out
}

func countNoticesIn(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(&domain.Notice{}).Where("kindergarten_id = ?", testNMTKG).Count(&n).Error)
	return n
}

// TC-NEW-NOT-0001：仅标题 / 仅正文都能存草稿，且草稿可以没有目标班级。
func TestNoticeDraft_CreateIncompleteDraft(t *testing.T) {
	f := newNoticeDraftFixture(t)
	ctx := context.Background()

	out, err := f.svcT.CreateNotice(ctx, f.tc, TeacherNoticeInput{
		Title:  "只有标题的草稿",
		Status: noticeStatusPtr(0),
	})
	require.NoError(t, err, "仅标题应可存草稿")
	id := out["id"].(uint64)
	assert.Equal(t, domain.NoticeStatusDraft, out["status"])

	ids, err := f.repo.ListNoticeTargetClassIDs(ctx, id)
	require.NoError(t, err)
	assert.Empty(t, ids, "草稿允许没有目标班级")

	detail, err := f.svcT.GetNotice(ctx, f.tc, id)
	require.NoError(t, err, "作者本人必须能读到自己无目标班的草稿")
	assert.Equal(t, "只有标题的草稿", detail["title"])
	assert.Equal(t, "", detail["content"])

	onlyContent, err := f.svcT.CreateNotice(ctx, f.tc, TeacherNoticeInput{
		Content: "只有正文的草稿",
		Status:  noticeStatusPtr(0),
	})
	require.NoError(t, err, "仅正文应可存草稿")
	assert.Equal(t, domain.NoticeStatusDraft, onlyContent["status"])
}

// TC-NEW-NOT-0002：标题正文都为空时拒绝，不落库。
func TestNoticeDraft_CreateEmptyDraftRejected(t *testing.T) {
	f := newNoticeDraftFixture(t)
	_, err := f.svcT.CreateNotice(context.Background(), f.tc, TeacherNoticeInput{
		Title: "   ", Content: "  ", Status: noticeStatusPtr(0),
	})
	assert.ErrorIs(t, err, ErrTeacherInvalidParams)
	assert.Equal(t, int64(0), countNoticesIn(t, f.db))
}

// TC-NEW-NOT-0003：不传 status 的列表不返回他人草稿。
func TestNoticeDraft_HiddenFromOtherTeacherList(t *testing.T) {
	f := newNoticeDraftFixture(t)
	ctx := context.Background()

	_, err := f.svcT.CreateNotice(ctx, f.tc, TeacherNoticeInput{
		Title: "我的草稿", Content: "正文", ClassIDs: []uint64{f.classA}, Status: noticeStatusPtr(0),
	})
	require.NoError(t, err)

	all, err := f.svcT.ListNotices(ctx, f.peerTC, nil, "", 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(0), all["total"], "同班其他老师不应看到他人草稿")

	drafts, err := f.svcT.ListNotices(ctx, f.peerTC, noticeStatusPtr(0), "", 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(0), drafts["total"], "草稿 tab 只返回自己的草稿")

	own, err := f.svcT.ListNotices(ctx, f.tc, nil, "", 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(1), own["total"], "作者本人应在全部列表看到自己的草稿")
}

// TC-NEW-NOT-0004：其他老师直接打开草稿详情得到 404。
func TestNoticeDraft_DetailHiddenFromOtherTeacher(t *testing.T) {
	f := newNoticeDraftFixture(t)
	ctx := context.Background()

	out, err := f.svcT.CreateNotice(ctx, f.tc, TeacherNoticeInput{
		Title: "别人的草稿", Content: "正文", ClassIDs: []uint64{f.classA}, Status: noticeStatusPtr(0),
	})
	require.NoError(t, err)
	id := out["id"].(uint64)

	_, err = f.svcT.GetNotice(ctx, f.peerTC, id)
	assert.ErrorIs(t, err, ErrTeacherNotFound)
}

// TC-NEW-NOT-0005：定时发送的过去时间/无时区时间一律拒绝。
func TestNoticeDraft_ScheduledPastRejected(t *testing.T) {
	f := newNoticeDraftFixture(t)
	ctx := context.Background()

	past := time.Now().Add(-time.Hour).Format(time.RFC3339)
	_, err := f.svcT.CreateNotice(ctx, f.tc, TeacherNoticeInput{
		Title: "过去时间", Content: "正文", ClassIDs: []uint64{f.classA},
		Status: noticeStatusPtr(1), ScheduledAt: past,
	})
	assert.ErrorIs(t, err, ErrTeacherInvalidParams)

	// 裸 "YYYY-MM-DD HH:mm"（修复前前端就是这么发的）没有时区，必须拒绝
	_, err = f.svcT.CreateNotice(ctx, f.tc, TeacherNoticeInput{
		Title: "无时区", Content: "正文", ClassIDs: []uint64{f.classA},
		Status: noticeStatusPtr(1), ScheduledAt: "2026-11-01 09:00",
	})
	assert.ErrorIs(t, err, ErrTeacherInvalidParams)
	assert.Equal(t, int64(0), countNoticesIn(t, f.db))
}

// TC-NEW-NOT-0006：带 +08:00 的 RFC3339 按同一时刻落库。
func TestNoticeDraft_ScheduledRFC3339Persisted(t *testing.T) {
	f := newNoticeDraftFixture(t)
	ctx := context.Background()

	raw := "2026-11-01T09:00:00+08:00"
	out, err := f.svcT.CreateNotice(ctx, f.tc, TeacherNoticeInput{
		Title: "定时通知", Content: "正文", ClassIDs: []uint64{f.classA},
		Status: noticeStatusPtr(1), ScheduledAt: raw,
	})
	require.NoError(t, err)
	id := out["id"].(uint64)

	var n domain.Notice
	require.NoError(t, f.db.WithContext(ctx).First(&n, id).Error)
	want, err := time.Parse(time.RFC3339, raw)
	require.NoError(t, err)
	assert.Equal(t, domain.NoticeStatusScheduled, n.Status)
	require.NotNil(t, n.ScheduledAt)
	assert.True(t, want.Equal(*n.ScheduledAt), "定时时间应按同一时区解释：want %s got %s", want, n.ScheduledAt)
	assert.Nil(t, n.PublishedAt, "定时通知未发布前不该有发布时间")
}

// TC-NEW-NOT-0007：已发布通知不可编辑。
func TestNoticeDraft_EditPublishedRejected(t *testing.T) {
	f := newNoticeDraftFixture(t)
	ctx := context.Background()

	out, err := f.svcT.CreateNotice(ctx, f.tc, TeacherNoticeInput{
		Title: "已发布", Content: "原文", ClassIDs: []uint64{f.classA},
	})
	require.NoError(t, err)
	id := out["id"].(uint64)

	_, err = f.svcT.UpdateNotice(ctx, f.tc, id, TeacherNoticeInput{
		Title: "被篡改", Content: "新文", ClassIDs: []uint64{f.classA},
	})
	assert.ErrorIs(t, err, ErrTeacherNoticeState)

	var n domain.Notice
	require.NoError(t, f.db.WithContext(ctx).First(&n, id).Error)
	assert.Equal(t, "已发布", n.Title, "编辑被拒时内容不变")
}

// TC-NEW-NOT-0008：不完整草稿发布被拒且保持草稿；完整草稿发布成功并写 published_at。
func TestNoticeDraft_PublishRequiresComplete(t *testing.T) {
	f := newNoticeDraftFixture(t)
	ctx := context.Background()

	out, err := f.svcT.CreateNotice(ctx, f.tc, TeacherNoticeInput{
		Title: "半个草稿", ClassIDs: []uint64{f.classA}, Status: noticeStatusPtr(0),
	})
	require.NoError(t, err)
	id := out["id"].(uint64)

	_, err = f.svcT.UpdateNotice(ctx, f.tc, id, TeacherNoticeInput{
		Title: "半个草稿", ClassIDs: []uint64{f.classA}, Status: noticeStatusPtr(2),
	})
	assert.ErrorIs(t, err, ErrTeacherInvalidParams, "正文为空不能发布")

	var draft domain.Notice
	require.NoError(t, f.db.WithContext(ctx).First(&draft, id).Error)
	assert.Equal(t, domain.NoticeStatusDraft, draft.Status, "发布失败应保留草稿")
	assert.Nil(t, draft.PublishedAt)

	updated, err := f.svcT.UpdateNotice(ctx, f.tc, id, TeacherNoticeInput{
		Title: "补全的草稿", Content: "正文补上了", ClassIDs: []uint64{f.classA},
		Status: noticeStatusPtr(2),
	})
	require.NoError(t, err)
	// PUT 响应契约：{id, status, updated_at}
	assert.Equal(t, domain.NoticeStatusPublished, updated["status"])
	assert.NotEmpty(t, updated["updated_at"])

	var published domain.Notice
	require.NoError(t, f.db.WithContext(ctx).First(&published, id).Error)
	assert.Equal(t, domain.NoticeStatusPublished, published.Status)
	require.NotNil(t, published.PublishedAt, "发布必须写发布时间")
	assert.Nil(t, published.ScheduledAt)
}

// TC-NEW-NOT-0009：附件落库，详情与列表一致；编辑时整体替换。
func TestNoticeDraft_AttachmentsPersistedAndReplaced(t *testing.T) {
	f := newNoticeDraftFixture(t)
	ctx := context.Background()

	out, err := f.svcT.CreateNotice(ctx, f.tc, TeacherNoticeInput{
		Title: "带图草稿", Content: "正文", ClassIDs: []uint64{f.classA},
		Status: noticeStatusPtr(0), Attachments: attachmentInputs(3),
	})
	require.NoError(t, err)
	id := out["id"].(uint64)

	detail, err := f.svcT.GetNotice(ctx, f.tc, id)
	require.NoError(t, err)
	atts := detail["attachments"].([]map[string]any)
	require.Len(t, atts, 3, "详情应返回 3 条附件")
	assert.Equal(t, "https://cdn.example.com/uploads/notice/draft-0.jpg", atts[0]["file_url"])

	list, err := f.svcT.ListNotices(ctx, f.tc, noticeStatusPtr(0), "", 1, 20)
	require.NoError(t, err)
	rows := list["list"].([]map[string]any)
	require.Len(t, rows, 1)
	assert.Equal(t, 3, rows[0]["attachments_count"], "列表附件数应为真实计数")

	// 编辑时删掉一张再保存：以提交列表为准（整体替换）
	_, err = f.svcT.UpdateNotice(ctx, f.tc, id, TeacherNoticeInput{
		Title: "带图草稿", Content: "正文", ClassIDs: []uint64{f.classA},
		Status: noticeStatusPtr(0), Attachments: attachmentInputs(3)[1:],
	})
	require.NoError(t, err)
	detail, err = f.svcT.GetNotice(ctx, f.tc, id)
	require.NoError(t, err)
	atts = detail["attachments"].([]map[string]any)
	require.Len(t, atts, 2, "删除一张后应只剩 2 条")
	assert.Equal(t, "https://cdn.example.com/uploads/notice/draft-1.jpg", atts[0]["file_url"], "保留项按 sort_num 重排")

	var rowsInDB int64
	require.NoError(t, f.db.Model(&domain.NoticeAttachment{}).Where("notice_id = ?", id).Count(&rowsInDB).Error)
	assert.Equal(t, int64(2), rowsInDB, "被删附件必须硬删，不能残留")
}

// TC-NEW-NOT-0010：附件非图片、超过 9 张、单张超 20MB 全部整单拒绝；9 张边界通过。
func TestNoticeDraft_AttachmentLimits(t *testing.T) {
	f := newNoticeDraftFixture(t)
	ctx := context.Background()

	_, err := f.svcT.CreateNotice(ctx, f.tc, TeacherNoticeInput{
		Title: "十张图", Content: "正文", ClassIDs: []uint64{f.classA}, Attachments: attachmentInputs(10),
	})
	assert.ErrorIs(t, err, ErrTeacherInvalidParams, "超过 9 张应拒绝")

	notImage := attachmentInputs(1)
	notImage[0].FileType = "video"
	_, err = f.svcT.CreateNotice(ctx, f.tc, TeacherNoticeInput{
		Title: "视频附件", Content: "正文", ClassIDs: []uint64{f.classA}, Attachments: notImage,
	})
	assert.ErrorIs(t, err, ErrTeacherInvalidParams, "非图片应拒绝")

	tooLarge := attachmentInputs(1)
	tooLarge[0].FileSize = 20<<20 + 1
	_, err = f.svcT.CreateNotice(ctx, f.tc, TeacherNoticeInput{
		Title: "超大附件", Content: "正文", ClassIDs: []uint64{f.classA}, Attachments: tooLarge,
	})
	assert.ErrorIs(t, err, ErrTeacherInvalidParams, "单张超过 20MB 应拒绝")

	assert.Equal(t, int64(0), countNoticesIn(t, f.db), "非法附件整单不保存")

	out, err := f.svcT.CreateNotice(ctx, f.tc, TeacherNoticeInput{
		Title: "九张图", Content: "正文", ClassIDs: []uint64{f.classA}, Attachments: attachmentInputs(9),
	})
	require.NoError(t, err, "9 张是上限边界，应通过")
	assert.NotZero(t, out["id"])
}

// TC-NEW-NOT-0011：仅草稿可删除、仅已发布可撤回；合法路径状态正确。
func TestNoticeDraft_DeleteAndRevokeRules(t *testing.T) {
	f := newNoticeDraftFixture(t)
	ctx := context.Background()

	publishedOut, err := f.svcT.CreateNotice(ctx, f.tc, TeacherNoticeInput{
		Title: "已发布", Content: "正文", ClassIDs: []uint64{f.classA},
	})
	require.NoError(t, err)
	publishedID := publishedOut["id"].(uint64)

	assert.ErrorIs(t, f.svcT.DeleteNotice(ctx, f.tc, publishedID), ErrTeacherNoticeState, "已发布不可删除")
	assert.Equal(t, int64(1), countNoticesIn(t, f.db), "删除被拒后通知仍存在")

	scheduledOut, err := f.svcT.CreateNotice(ctx, f.tc, TeacherNoticeInput{
		Title: "定时", Content: "正文", ClassIDs: []uint64{f.classA},
		Status: noticeStatusPtr(1), ScheduledAt: time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	})
	require.NoError(t, err)
	assert.ErrorIs(t, f.svcT.RevokeNotice(ctx, f.tc, scheduledOut["id"].(uint64)), ErrTeacherNoticeState, "定时未发不可撤回")

	draftOut, err := f.svcT.CreateNotice(ctx, f.tc, TeacherNoticeInput{
		Title: "草稿", Status: noticeStatusPtr(0), Attachments: attachmentInputs(2),
	})
	require.NoError(t, err)
	draftID := draftOut["id"].(uint64)
	require.NoError(t, f.svcT.DeleteNotice(ctx, f.tc, draftID), "草稿可删除")

	// 软删通知后附件与目标集都不能留成孤儿
	var orphanAttachments, orphanTargets int64
	require.NoError(t, f.db.Model(&domain.NoticeAttachment{}).
		Where("notice_id = ?", draftID).Count(&orphanAttachments).Error)
	assert.Zero(t, orphanAttachments, "删除草稿应一并清理附件")
	require.NoError(t, f.db.Model(&domain.NoticeTarget{}).
		Where("notice_id = ?", draftID).Count(&orphanTargets).Error)
	assert.Zero(t, orphanTargets, "删除草稿应一并清理目标集")

	require.NoError(t, f.svcT.RevokeNotice(ctx, f.tc, publishedID), "已发布可撤回")
	var revoked domain.Notice
	require.NoError(t, f.db.WithContext(ctx).First(&revoked, publishedID).Error)
	assert.Equal(t, domain.NoticeStatusRevoked, revoked.Status)
}

// 清理证据：执行同一套清理后，本园所不留通知 / 目标 / 附件（附件表无软删，必须单独硬删）。
func TestNoticeDraft_CleanupLeavesNoResidue(t *testing.T) {
	f := newNoticeDraftFixture(t)
	ctx := context.Background()

	out, err := f.svcT.CreateNotice(ctx, f.tc, TeacherNoticeInput{
		Title: "待清理", Content: "正文", ClassIDs: []uint64{f.classA}, Attachments: attachmentInputs(2),
	})
	require.NoError(t, err)
	noticeID := out["id"].(uint64)

	require.NoError(t, f.db.WithContext(ctx).Exec(
		"DELETE FROM notice_attachments WHERE notice_id IN (SELECT id FROM notices WHERE kindergarten_id = ?)",
		testNMTKG).Error)
	newNMTCleanup(t, f.db)

	var notices, targets, attachments int64
	require.NoError(t, f.db.WithContext(ctx).Model(&domain.Notice{}).
		Where("kindergarten_id = ?", testNMTKG).Count(&notices).Error)
	require.NoError(t, f.db.WithContext(ctx).Model(&domain.NoticeTarget{}).
		Where("kindergarten_id = ?", testNMTKG).Count(&targets).Error)
	require.NoError(t, f.db.WithContext(ctx).Model(&domain.NoticeAttachment{}).
		Where("notice_id = ?", noticeID).Count(&attachments).Error)
	assert.Zero(t, notices, "通知应清零")
	assert.Zero(t, targets, "目标集应清零")
	assert.Zero(t, attachments, "附件应清零")
}

// TC-NEW-NOT-0012：家长只能确认已发布通知；草稿/定时/已撤回不产生确认记录。
func TestNoticeDraft_ParentConfirmOnlyPublished(t *testing.T) {
	f := newNoticeDraftFixture(t)
	ctx := context.Background()

	require.NoError(t, f.db.WithContext(ctx).Exec(
		"DELETE FROM notice_read_logs WHERE user_id = ?", f.pcMulti.UserID).Error)

	scheduledOut, err := f.svcT.CreateNotice(ctx, f.tc, TeacherNoticeInput{
		Title: "定时", Content: "正文", ClassIDs: []uint64{f.classA},
		Status: noticeStatusPtr(1), ScheduledAt: time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	})
	require.NoError(t, err)
	_, err = f.svcP.ConfirmNotice(ctx, f.pcMulti, scheduledOut["id"].(uint64))
	assert.ErrorIs(t, err, ErrParentInvalidParams, "定时通知不可确认")

	draftOut, err := f.svcT.CreateNotice(ctx, f.tc, TeacherNoticeInput{
		Title: "草稿", Content: "正文", ClassIDs: []uint64{f.classA}, Status: noticeStatusPtr(0),
	})
	require.NoError(t, err)
	_, err = f.svcP.ConfirmNotice(ctx, f.pcMulti, draftOut["id"].(uint64))
	assert.ErrorIs(t, err, ErrParentInvalidParams, "草稿不可确认")

	publishedOut, err := f.svcT.CreateNotice(ctx, f.tc, TeacherNoticeInput{
		Title: "已发布", Content: "正文", ClassIDs: []uint64{f.classA},
	})
	require.NoError(t, err)
	publishedID := publishedOut["id"].(uint64)
	require.NoError(t, f.svcT.RevokeNotice(ctx, f.tc, publishedID))
	_, err = f.svcP.ConfirmNotice(ctx, f.pcMulti, publishedID)
	assert.ErrorIs(t, err, ErrParentInvalidParams, "已撤回不可确认")

	var confirmed int64
	require.NoError(t, f.db.Model(&domain.NoticeReadLog{}).
		Where("notice_id IN ? AND user_id = ? AND confirmed_at IS NOT NULL",
			[]uint64{scheduledOut["id"].(uint64), draftOut["id"].(uint64), publishedID}, f.pcMulti.UserID).
		Count(&confirmed).Error)
	assert.Equal(t, int64(0), confirmed, "被拒的确认不得写入确认记录")

	publishedOut2, err := f.svcT.CreateNotice(ctx, f.tc, TeacherNoticeInput{
		Title: "已发布2", Content: "正文", ClassIDs: []uint64{f.classA},
	})
	require.NoError(t, err)
	res, err := f.svcP.ConfirmNotice(ctx, f.pcMulti, publishedOut2["id"].(uint64))
	require.NoError(t, err)
	assert.Equal(t, true, res["is_confirmed"])
}
