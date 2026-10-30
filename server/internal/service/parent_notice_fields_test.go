package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// B13：家长端通知契约字段补齐（backend/api/api-parent/02-notice.md）。
// 覆盖列表的 category_label / content_preview / creator / attachments_count / is_read，
// 详情的 category_label / creator / attachments / read_at / confirmed_at，
// 以及「读状态按家长隔离」和「列表无 N+1」。
// 复用 notice_multitarget_test.go 的夹具（园所 990201，老师 990201）。

// parentNoticeParentB 与 pcMulti 同班的第二位家长（990211，夹具里原本是无关班家长）。
// 这里只用上下文表达可见班级，不写 student_parents：服务按 pc.Students 判可见性。
func parentNoticeParentB(f *nmtFixture) *ParentPortalContext {
	pc := &ParentPortalContext{
		UserID:         testNMTParent2,
		KindergartenID: testNMTKG,
		Students: []domain.Student{{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: 990220}},
			ClassID:         f.classA,
		}},
	}
	pc.CurrentStudent = &pc.Students[0]
	return pc
}

// newParentNoticeFieldsFixture 在通用夹具上接管 notice_attachments 的清理。
// 附件表不在 newNMTCleanup 范围内；本清理后注册先执行（LIFO），先删附件再让夹具删通知。
func newParentNoticeFieldsFixture(t *testing.T) *nmtFixture {
	t.Helper()
	f := newNMTFixture(t)
	purgeAttachments := func() {
		if err := f.db.Exec(
			"DELETE FROM notice_attachments WHERE notice_id IN (SELECT id FROM notices WHERE kindergarten_id = ?)",
			testNMTKG).Error; err != nil {
			t.Logf("cleanup attachments: %v", err)
		}
	}
	purgeAttachments()
	t.Cleanup(purgeAttachments)
	return f
}

func createParentNotice(t *testing.T, f *nmtFixture, in TeacherNoticeInput) uint64 {
	t.Helper()
	if len(in.ClassIDs) == 0 {
		in.ClassIDs = []uint64{f.classA}
	}
	out, err := f.svcT.CreateNotice(context.Background(), f.tc, in)
	require.NoError(t, err)
	id, ok := out["id"].(uint64)
	require.True(t, ok, "创建通知应返回 id")
	return id
}

func parentNoticeAttachmentInputs(names ...string) []TeacherNoticeAttachmentInput {
	out := make([]TeacherNoticeAttachmentInput, 0, len(names))
	for i, name := range names {
		out = append(out, TeacherNoticeAttachmentInput{
			FileURL:  fmt.Sprintf("https://cdn.example.com/uploads/notice/b13-%d.jpg", i),
			FileType: "image",
			FileName: name,
			FileSize: int64(512000 + i),
		})
	}
	return out
}

func parentNoticeBoolPtr(v bool) *bool { return &v }

func parentNoticeListItems(t *testing.T, result map[string]any) map[uint64]map[string]any {
	t.Helper()
	raw, ok := result["list"].([]map[string]any)
	require.True(t, ok)
	out := make(map[uint64]map[string]any, len(raw))
	for _, item := range raw {
		id, ok := item["id"].(uint64)
		require.True(t, ok)
		out[id] = item
	}
	return out
}

// TestParentNoticeList_ContractFields 列表补齐契约字段：中文类型、单行预览、作者、附件数。
func TestParentNoticeList_ContractFields(t *testing.T) {
	f := newParentNoticeFieldsFixture(t)
	ctx := context.Background()

	withAttachment := createParentNotice(t, f, TeacherNoticeInput{
		Category:    "homework",
		Title:       "带附件的作业通知",
		Content:     "各位家长您好，\n\n本周作业如下：  1. 复习儿歌；\n2. 完成手工。",
		Attachments: parentNoticeAttachmentInputs("附件一.jpg", "附件二.jpg"),
	})
	longContent := strings.Repeat("家长您好，本周请配合完成亲子阅读与手工准备。", 5)
	noAttachment := createParentNotice(t, f, TeacherNoticeInput{
		Category: "task", Title: "零附件的任务通知", Content: longContent,
	})
	// 草稿不是 status=2，列表不应出现
	createParentNotice(t, f, TeacherNoticeInput{
		Category: "other", Title: "草稿不应出现", Content: "草稿正文", Status: noticeStatusPtr(0),
	})

	result, err := f.svcP.ListNotices(ctx, f.pcMulti, "", 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(2), result["total"], "只返回已发布通知")
	items := parentNoticeListItems(t, result)

	att, ok := items[withAttachment]
	require.True(t, ok)
	assert.Equal(t, "作业", att["category_label"])
	assert.Equal(t, 2, att["attachments_count"])
	assert.Equal(t, map[string]any{"nickname": "测试多班老师", "avatar": ""}, att["creator"])
	assert.False(t, att["is_read"].(bool))
	assert.False(t, att["is_confirmed"].(bool))
	assert.Equal(t, "各位家长您好， 本周作业如下： 1. 复习儿歌； 2. 完成手工。", att["content_preview"])
	assert.NotContains(t, att["content_preview"].(string), "\n", "预览必须压成一行")

	noAtt, ok := items[noAttachment]
	require.True(t, ok)
	assert.Equal(t, "任务", noAtt["category_label"])
	assert.Equal(t, 0, noAtt["attachments_count"], "无附件返回 0")
	preview := noAtt["content_preview"].(string)
	assert.True(t, strings.HasSuffix(preview, "…"), "超长正文应截断并加省略号")
	assert.Equal(t, parentNoticePreviewRunes+1, len([]rune(preview)), "截断长度为 60 字符 + 省略号")
}

// TestParentNoticeCategoryLabel_ContractValues 六个契约分类的中文标签，列表与详情同源。
func TestParentNoticeCategoryLabel_ContractValues(t *testing.T) {
	f := newParentNoticeFieldsFixture(t)
	ctx := context.Background()

	want := map[string]string{
		"task": "任务", "homework": "作业", "bring": "带物",
		"activity": "活动", "health": "健康", "other": "其他",
	}
	ids := make(map[uint64]string, len(want))
	for category := range want {
		ids[createParentNotice(t, f, TeacherNoticeInput{
			Category: category, Title: "分类-" + category, Content: "正文",
		})] = category
	}

	result, err := f.svcP.ListNotices(ctx, f.pcMulti, "", 1, 20)
	require.NoError(t, err)
	items := parentNoticeListItems(t, result)
	require.Len(t, items, len(want))
	for id, category := range ids {
		assert.Equal(t, want[category], items[id]["category_label"], "category=%s", category)
		assert.Equal(t, category, string(items[id]["category"].(domain.NoticeCategory)))
	}
	for id, category := range ids {
		detail, err := f.svcP.GetNotice(ctx, f.pcMulti, id)
		require.NoError(t, err)
		assert.Equal(t, want[category], detail["category_label"], "详情 category=%s", category)
	}
}

// TestParentNoticeList_IsReadFilter is_read 三态：不传不过滤、true 只看已读、false 只看未读。
func TestParentNoticeList_IsReadFilter(t *testing.T) {
	f := newParentNoticeFieldsFixture(t)
	ctx := context.Background()

	readA := createParentNotice(t, f, TeacherNoticeInput{Category: "other", Title: "已读A", Content: "正文"})
	readB := createParentNotice(t, f, TeacherNoticeInput{Category: "other", Title: "已读B", Content: "正文"})
	unread := createParentNotice(t, f, TeacherNoticeInput{Category: "other", Title: "未读", Content: "正文"})
	require.NoError(t, f.svcP.MarkNoticeRead(ctx, f.pcMulti, readA))
	require.NoError(t, f.svcP.MarkNoticeRead(ctx, f.pcMulti, readB))

	all, err := f.svcP.ListNotices(ctx, f.pcMulti, "", 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(3), all["total"], "不传 is_read 不过滤")

	onlyRead, err := f.svcP.ListNoticesFiltered(ctx, f.pcMulti, ParentNoticeListFilter{IsRead: parentNoticeBoolPtr(true)}, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(2), onlyRead["total"], "is_read=true 只返回有读记录的")
	readItems := parentNoticeListItems(t, onlyRead)
	assert.Contains(t, readItems, readA)
	assert.Contains(t, readItems, readB)
	for _, item := range readItems {
		assert.True(t, item["is_read"].(bool))
	}

	onlyUnread, err := f.svcP.ListNoticesFiltered(ctx, f.pcMulti, ParentNoticeListFilter{IsRead: parentNoticeBoolPtr(false)}, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(1), onlyUnread["total"], "is_read=false 只返回无读记录的")
	unreadItems := parentNoticeListItems(t, onlyUnread)
	item, ok := unreadItems[unread]
	require.True(t, ok)
	assert.False(t, item["is_read"].(bool))
}

// TestParentNoticeDetail_ContractFieldsAndReadStates 详情字段、附件排序与读/确认三态。
func TestParentNoticeDetail_ContractFieldsAndReadStates(t *testing.T) {
	f := newParentNoticeFieldsFixture(t)
	ctx := context.Background()

	id := createParentNotice(t, f, TeacherNoticeInput{
		Category: "task", Title: "详情契约字段", Content: "正文内容", NeedConfirm: true,
		Attachments: parentNoticeAttachmentInputs("附件一.jpg", "附件二.jpg", "附件三.jpg"),
	})
	// 打乱 sort_num：atts[0] 与 atts[1] 同为 1，id 更小的 atts[0] 应在前；atts[2] 置 0 排最前。
	atts, err := f.repo.ListAttachmentsByNoticeID(ctx, id)
	require.NoError(t, err)
	require.Len(t, atts, 3)
	require.NoError(t, f.db.WithContext(ctx).Model(&domain.NoticeAttachment{}).Where("id = ?", atts[0].ID).Update("sort_num", 1).Error)
	require.NoError(t, f.db.WithContext(ctx).Model(&domain.NoticeAttachment{}).Where("id = ?", atts[2].ID).Update("sort_num", 0).Error)

	// 未读：read_at / confirmed_at 都是 null
	detail, err := f.svcP.GetNotice(ctx, f.pcMulti, id)
	require.NoError(t, err)
	assert.Equal(t, "任务", detail["category_label"])
	creator, ok := detail["creator"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, testNMTTeacher, creator["id"])
	assert.Equal(t, "测试多班老师", creator["nickname"])
	assert.Equal(t, "", creator["avatar"])
	assert.False(t, detail["is_read"].(bool))
	assert.False(t, detail["is_confirmed"].(bool))
	assert.Nil(t, detail["read_at"])
	assert.Nil(t, detail["confirmed_at"])

	attachments, ok := detail["attachments"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, attachments, 3)
	names := make([]string, 0, len(attachments))
	for _, a := range attachments {
		names = append(names, a["file_name"].(string))
		assert.NotZero(t, a["id"])
		assert.NotEmpty(t, a["file_url"])
		assert.Equal(t, "image", a["file_type"])
		assert.NotZero(t, a["file_size"])
	}
	assert.Equal(t, []string{"附件三.jpg", "附件一.jpg", "附件二.jpg"}, names, "按 sort_num ASC, id ASC")

	// 已读未确认
	require.NoError(t, f.svcP.MarkNoticeRead(ctx, f.pcMulti, id))
	detail, err = f.svcP.GetNotice(ctx, f.pcMulti, id)
	require.NoError(t, err)
	assert.True(t, detail["is_read"].(bool))
	assert.False(t, detail["is_confirmed"].(bool))
	readAt, ok := detail["read_at"].(string)
	require.True(t, ok, "已读后 read_at 应为 RFC3339 字符串")
	_, err = time.Parse(time.RFC3339, readAt)
	require.NoError(t, err)
	assert.Nil(t, detail["confirmed_at"], "未确认时 confirmed_at 为 null")

	// 确认后
	_, err = f.svcP.ConfirmNotice(ctx, f.pcMulti, id)
	require.NoError(t, err)
	detail, err = f.svcP.GetNotice(ctx, f.pcMulti, id)
	require.NoError(t, err)
	assert.True(t, detail["is_confirmed"].(bool))
	confirmedAt, ok := detail["confirmed_at"].(string)
	require.True(t, ok, "确认后 confirmed_at 应为 RFC3339 字符串")
	_, err = time.Parse(time.RFC3339, confirmedAt)
	require.NoError(t, err)

	// 仅 status=2 可读
	draftID := createParentNotice(t, f, TeacherNoticeInput{
		Category: "other", Title: "草稿", Content: "草稿正文", Status: noticeStatusPtr(0),
	})
	_, err = f.svcP.GetNotice(ctx, f.pcMulti, draftID)
	assert.ErrorIs(t, err, ErrParentNotFound)
}

// TestParentNoticeReadState_IsolatedPerParent A 的读/确认不影响同班 B 家长。
func TestParentNoticeReadState_IsolatedPerParent(t *testing.T) {
	f := newParentNoticeFieldsFixture(t)
	ctx := context.Background()
	id := createParentNotice(t, f, TeacherNoticeInput{
		Category: "health", Title: "隔离验证", Content: "正文", NeedConfirm: true,
	})
	pcB := parentNoticeParentB(f)

	require.NoError(t, f.svcP.MarkNoticeRead(ctx, f.pcMulti, id))
	_, err := f.svcP.ConfirmNotice(ctx, f.pcMulti, id)
	require.NoError(t, err)

	detailA, err := f.svcP.GetNotice(ctx, f.pcMulti, id)
	require.NoError(t, err)
	assert.True(t, detailA["is_read"].(bool))
	assert.True(t, detailA["is_confirmed"].(bool))

	detailB, err := f.svcP.GetNotice(ctx, pcB, id)
	require.NoError(t, err)
	assert.False(t, detailB["is_read"].(bool))
	assert.False(t, detailB["is_confirmed"].(bool))
	assert.Nil(t, detailB["read_at"])
	assert.Nil(t, detailB["confirmed_at"])

	listB, err := f.svcP.ListNotices(ctx, pcB, "", 1, 20)
	require.NoError(t, err)
	itemB, ok := parentNoticeListItems(t, listB)[id]
	require.True(t, ok)
	assert.False(t, itemB["is_read"].(bool))
	assert.False(t, itemB["is_confirmed"].(bool))

	// 反向：B 读后不改变 A 的确认状态
	require.NoError(t, f.svcP.MarkNoticeRead(ctx, pcB, id))
	detailA, err = f.svcP.GetNotice(ctx, f.pcMulti, id)
	require.NoError(t, err)
	assert.True(t, detailA["is_confirmed"].(bool))
}

// TestParentNoticeFields_CleanupLeavesNoResidue 清理证据：同一套清理后本园所不留通知/目标/附件/读记录。
func TestParentNoticeFields_CleanupLeavesNoResidue(t *testing.T) {
	f := newParentNoticeFieldsFixture(t)
	ctx := context.Background()

	id := createParentNotice(t, f, TeacherNoticeInput{
		Category: "other", Title: "待清理", Content: "正文",
		Attachments: parentNoticeAttachmentInputs("附件一.jpg"),
	})
	require.NoError(t, f.svcP.MarkNoticeRead(ctx, f.pcMulti, id))
	_, err := f.svcP.ConfirmNotice(ctx, f.pcMulti, id)
	require.NoError(t, err)

	// 与夹具同一套清理：附件先硬删，其余由 newNMTCleanup 处理。
	require.NoError(t, f.db.WithContext(ctx).Exec(
		"DELETE FROM notice_attachments WHERE notice_id IN (SELECT id FROM notices WHERE kindergarten_id = ?)",
		testNMTKG).Error)
	newNMTCleanup(t, f.db)

	var notices, targets, attachments, readLogs int64
	require.NoError(t, f.db.WithContext(ctx).Model(&domain.Notice{}).
		Where("kindergarten_id = ?", testNMTKG).Count(&notices).Error)
	require.NoError(t, f.db.WithContext(ctx).Model(&domain.NoticeTarget{}).
		Where("kindergarten_id = ?", testNMTKG).Count(&targets).Error)
	require.NoError(t, f.db.WithContext(ctx).Model(&domain.NoticeAttachment{}).
		Where("notice_id = ?", id).Count(&attachments).Error)
	require.NoError(t, f.db.WithContext(ctx).Model(&domain.NoticeReadLog{}).
		Where("notice_id = ?", id).Count(&readLogs).Error)
	assert.Zero(t, notices, "通知应清零")
	assert.Zero(t, targets, "目标集应清零")
	assert.Zero(t, attachments, "附件应清零")
	assert.Zero(t, readLogs, "读记录应清零")
}

// noticeQueryCounter 统计 GORM 实际执行的语句数（每个语句 Trace 一次）。
type noticeQueryCounter struct {
	mu    sync.Mutex
	count int64
}

func (c *noticeQueryCounter) LogMode(gormlogger.LogLevel) gormlogger.Interface { return c }
func (c *noticeQueryCounter) Info(context.Context, string, ...interface{})     {}
func (c *noticeQueryCounter) Warn(context.Context, string, ...interface{})     {}
func (c *noticeQueryCounter) Error(context.Context, string, ...interface{})    {}
func (c *noticeQueryCounter) Trace(context.Context, time.Time, func() (string, int64), error) {
	c.mu.Lock()
	c.count++
	c.mu.Unlock()
}

func (c *noticeQueryCounter) value() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.count
}

func (c *noticeQueryCounter) reset() {
	c.mu.Lock()
	c.count = 0
	c.mu.Unlock()
}

func newParentPortalServiceWithDB(db *gorm.DB) *ParentPortalService {
	return NewParentPortalService(db, repository.NewParentRepo(db), repository.NewUserRoleRepo(db),
		repository.NewStudentRepo(db), repository.NewClassRepo(db), repository.NewPickupRepo(db),
		repository.NewCalendarRepo(db), repository.NewNoticeRepo(db))
}

// TestParentNoticeList_NoNPlusOne 查询次数不随行数增长：整页 20 条与单条相同。
func TestParentNoticeList_NoNPlusOne(t *testing.T) {
	f := newParentNoticeFieldsFixture(t)
	ctx := context.Background()
	for i := 0; i < 20; i++ {
		createParentNotice(t, f, TeacherNoticeInput{
			Category: "other", Title: fmt.Sprintf("批量通知 %d", i), Content: "正文",
			Attachments: parentNoticeAttachmentInputs("附件.jpg"),
		})
	}

	counter := &noticeQueryCounter{}
	svc := newParentPortalServiceWithDB(f.db.Session(&gorm.Session{Logger: counter}))

	_, err := svc.ListNotices(ctx, f.pcMulti, "", 1, 20)
	require.NoError(t, err)
	fullPage := counter.value()
	// 5 = Count + Find + 作者批量 + 附件数批量 + 读记录批量
	t.Logf("整页 20 条的查询数：%d", fullPage)
	assert.LessOrEqual(t, fullPage, int64(6), "一页 20 条应为常数次查询（Count/Find/作者/附件数/读记录）")

	counter.reset()
	_, err = svc.ListNotices(ctx, f.pcMulti, "", 1, 1)
	require.NoError(t, err)
	singleRow := counter.value()
	t.Logf("单条页的查询数：%d", singleRow)
	assert.Equal(t, fullPage, singleRow, "整页与单条查询数相同 —— 没有逐条查询")
}
