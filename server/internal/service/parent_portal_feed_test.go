package service

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/domain"
)

// 这些测试只覆盖 assembleFeedItems / splitPerformanceTags 的纯逻辑：
// 响应形状（含「不含冗余字段」）是 spec 的验收点，而批量查询本身依赖真实库，由集成验证覆盖。
// 对应 TC 见 openspec/changes/align-parent-growth-feed/specs/growth-feed/spec.md。

func feedRow(id uint64, target int8, content string) domain.GrowthDailyLog {
	return domain.GrowthDailyLog{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{
			ID:        id,
			CreatedAt: time.Date(2026, 3, 19, 11, 30, 0, 0, time.UTC),
		}},
		CreatorID:  7,
		Date:       time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC),
		Content:    content,
		TargetType: target,
		Creator: &domain.User{
			BaseModel: domain.BaseModel{ID: 7},
			Nickname:  "李老师",
			Avatar:    "https://cdn.example.com/avatar/1.jpg",
		},
	}
}

// TC-GRW-099/100/101：发送范围映射为展示层 scope；未知值防御性归为 class。
func TestFeedScope(t *testing.T) {
	assert.Equal(t, "class", feedScope(1))
	assert.Equal(t, "individual", feedScope(2))
	assert.Equal(t, "class", feedScope(0))
}

// TC-GRW-099/100：组装条目按 TargetType 输出 scope。
func TestAssembleFeedItems_ScopeUsesTargetType(t *testing.T) {
	rows := []domain.GrowthDailyLog{
		feedRow(1, 1, "全班记录"),
		feedRow(2, 2, "个体记录"),
	}

	list := assembleFeedItems(rows, nil, nil, nil, nil, nil)

	assert.Equal(t, "class", list[0]["scope"])
	assert.Equal(t, "individual", list[1]["scope"])
}

// TC-GRW-080/031/032：卡片显示发布者昵称、发布时刻、正文
func TestAssembleFeedItems_CardShowsCreatorTimeAndContent(t *testing.T) {
	rows := []domain.GrowthDailyLog{feedRow(1, 1, "今天的美术课主题是《我心中的春天》")}

	list := assembleFeedItems(rows, nil, nil, nil, nil, nil)

	assert.Len(t, list, 1)
	assert.Equal(t, uint64(1), list[0]["id"])
	assert.Equal(t, "今天的美术课主题是《我心中的春天》", list[0]["content"])
	assert.Equal(t, "2026-03-19", list[0]["date"])
	assert.Equal(t, "2026-03-19T11:30:00Z", list[0]["created_at"])

	creator, ok := list[0]["creator"].(map[string]any)
	assert.True(t, ok, "creator 必须是对象")
	assert.Equal(t, uint64(7), creator["id"])
	assert.Equal(t, "李老师", creator["nickname"])
	assert.Equal(t, "https://cdn.example.com/avatar/1.jpg", creator["avatar"])
}

// TC-GRW-083：有照片的记录把 media 挂上
func TestAssembleFeedItems_MediaAttachedWhenPresent(t *testing.T) {
	rows := []domain.GrowthDailyLog{feedRow(1, 1, "x")}
	mediaByLog := map[uint64][]domain.GrowthMedia{
		1: {
			{LogID: ptrU64(1), FileType: "image", FileURL: "https://cdn.example.com/a.jpg"},
			{LogID: ptrU64(1), FileType: "image", FileURL: "https://cdn.example.com/b.jpg"},
		},
	}

	list := assembleFeedItems(rows, mediaByLog, nil, nil, nil, nil)

	media, ok := list[0]["media"].([]feedMediaItem)
	assert.True(t, ok)
	assert.Len(t, media, 2)
}

// TC-GRW-084：无照片的记录 media 是空数组而非 null——前端 v-if 按 length 判断
func TestAssembleFeedItems_MediaIsEmptySliceNotNil(t *testing.T) {
	rows := []domain.GrowthDailyLog{feedRow(1, 1, "x")}

	list := assembleFeedItems(rows, map[uint64][]domain.GrowthMedia{}, nil, nil, nil, nil)

	media, ok := list[0]["media"].([]feedMediaItem)
	assert.True(t, ok, "media 必须是切片，不能是 nil，否则 JSON 变成 null")
	assert.Len(t, media, 0)
	assert.NotNil(t, media)
}

// 附件只暴露契约声明的 6 个字段；GORM 模型里的记账字段不得泄漏出去
// （与本次砍掉的 target_type/class_name 属同一类问题：实现泄漏内部字段）
func TestToFeedMedia_ExactFieldSet(t *testing.T) {
	out := toFeedMedia([]domain.GrowthMedia{{
		BaseModel:      domain.BaseModel{ID: 9},
		KindergartenID: 4,
		ClassID:        6,
		LogID:          ptrU64(1),
		UploaderID:     61,
		FileURL:        "https://cdn.example.com/a.jpg",
		ThumbURL:       "https://cdn.example.com/a_thumb.jpg",
		FileType:       "video",
		FileName:       "a.mp4",
		FileSize:       2048,
		Duration:       30,
		SortNum:        1,
	}}, map[uint64][]uint64{9: {11, 22}})

	require.Len(t, out, 1)
	assert.Equal(t, uint64(9), out[0].ID)
	assert.Equal(t, "https://cdn.example.com/a.jpg", out[0].FileURL)
	assert.Equal(t, "https://cdn.example.com/a_thumb.jpg", out[0].ThumbURL)
	assert.Equal(t, "video", out[0].FileType)
	assert.Equal(t, 30, out[0].Duration)
	assert.Equal(t, []uint64{11, 22}, out[0].StudentIDs)

	// 用 JSON 往返验证键集，避免只测 Go 结构体字段而漏掉序列化层面的泄漏
	b, err := json.Marshal(out[0])
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	assert.Len(t, m, 6, "附件 JSON 只能有 6 个字段，实际: %v", m)
	for _, banned := range []string{"class_id", "kindergarten_id", "log_id", "uploader_id", "file_name", "file_size", "sort_num", "created_at", "updated_at"} {
		_, exists := m[banned]
		assert.False(t, exists, "附件 JSON 不得含 %q", banned)
	}
}

// 无点名的照片 student_ids 是空数组而非 null——前端按 includes 判断无需判空
func TestToFeedMedia_StudentIDsEmptySliceNotNil(t *testing.T) {
	out := toFeedMedia([]domain.GrowthMedia{{BaseModel: domain.BaseModel{ID: 9}, FileType: "image"}}, nil)
	require.Len(t, out, 1)
	assert.NotNil(t, out[0].StudentIDs)
	assert.Len(t, out[0].StudentIDs, 0)
	b, err := json.Marshal(out[0])
	require.NoError(t, err)
	assert.Contains(t, string(b), `"student_ids":[]`)
}

// TC-GRW-085：点赞态反映本人操作
func TestAssembleFeedItems_IsLikedReflectsCurrentUser(t *testing.T) {
	rows := []domain.GrowthDailyLog{feedRow(1, 1, "a"), feedRow(2, 1, "b")}
	likedSet := map[uint64]bool{2: true}

	list := assembleFeedItems(rows, nil, nil, likedSet, nil, nil)

	assert.Equal(t, false, list[0]["is_liked"])
	assert.Equal(t, true, list[1]["is_liked"])
}

// TC-GRW-086：评论数真实
func TestAssembleFeedItems_CommentsCount(t *testing.T) {
	rows := []domain.GrowthDailyLog{feedRow(1, 1, "a"), feedRow(2, 1, "b")}
	commentCounts := map[uint64]int64{1: 2}

	list := assembleFeedItems(rows, nil, nil, nil, commentCounts, nil)

	assert.Equal(t, int64(2), list[0]["comments_count"])
	assert.Equal(t, int64(0), list[1]["comments_count"], "无评论时为 0，前端渲染为「评论」")
}

// TC-GRW-089 / TC-GRW-092：
// 列表不承载详情页才需要的信息（related_students），也不输出设计稿从不渲染的冗余字段。
// 即便是个体记录（target_type=2）也不输出；scope 是两模式必需字段，单独保留。
func TestAssembleFeedItems_OmitsSuperfluousAndDetailOnlyFields(t *testing.T) {
	rows := []domain.GrowthDailyLog{feedRow(1, 2, "个体记录")}

	list := assembleFeedItems(rows, nil, nil, nil, nil, nil)
	assert.Equal(t, "individual", list[0]["scope"])

	for _, key := range []string{
		"target_type", "class_name", "performance_tags",
		"likes_count", "media_count",
		"related_students", "related_students_count",
	} {
		_, exists := list[0][key]
		assert.False(t, exists, "列表响应不得出现 %q", key)
	}
}

// 条目必须恰好包含卡片渲染所需的键——多一个少一个都是漂移
func TestAssembleFeedItems_ExactKeySet(t *testing.T) {
	rows := []domain.GrowthDailyLog{feedRow(1, 1, "x")}

	list := assembleFeedItems(rows, nil, nil, nil, nil, nil)

	expected := []string{"id", "scope", "date", "content", "created_at", "media", "is_liked", "comments_count", "creator"}
	assert.Len(t, list[0], len(expected))
	for _, k := range expected {
		_, ok := list[0][k]
		assert.True(t, ok, "缺少必需字段 %q", k)
	}
}

// 无 creator 时不崩，且不输出空 creator 键
func TestAssembleFeedItems_NilCreator(t *testing.T) {
	row := feedRow(1, 1, "x")
	row.Creator = nil

	list := assembleFeedItems([]domain.GrowthDailyLog{row}, nil, nil, nil, nil, nil)

	_, exists := list[0]["creator"]
	assert.False(t, exists)
}

// 空页：返回空列表，不报错
func TestAssembleFeedItems_EmptyRows(t *testing.T) {
	list := assembleFeedItems(nil, nil, nil, nil, nil, nil)
	assert.NotNil(t, list)
	assert.Len(t, list, 0)
}

// splitPerformanceTags：契约与 detail.vue 按数组消费
func TestSplitPerformanceTags(t *testing.T) {
	assert.Equal(t, []string{}, splitPerformanceTags(""), "空串返回空数组，不能是 [\"\"]")
	assert.Equal(t, []string{}, splitPerformanceTags("  , ,"), "纯分隔符返回空数组")
	assert.Equal(t, []string{"乐于助人", "积极发言"}, splitPerformanceTags("乐于助人,积极发言"))
	assert.Equal(t, []string{"乐于助人", "积极发言"}, splitPerformanceTags(" 乐于助人 , 积极发言 "), "容忍前后空格")
}

func ptrU64(v uint64) *uint64 { return &v }

// TC-GRW-096：见证记录显示徽章（名称/图标/色类齐备）
func TestAssembleFeedItems_BadgeShownOnWitnessRecord(t *testing.T) {
	rows := []domain.GrowthDailyLog{feedRow(1, 1, "恭喜小明获得全勤小明星称号")}
	badgeByLog := map[uint64]map[string]any{
		1: toBadge(domain.Achievement{Name: "全勤小明星", Icon: "https://cdn/badge.png", Category: "attendance"}),
	}

	list := assembleFeedItems(rows, nil, nil, nil, nil, badgeByLog)

	b, ok := list[0]["badge"].(map[string]any)
	require.True(t, ok, "见证记录必须带 badge 对象")
	assert.Equal(t, "全勤小明星", b["name"])
	assert.Equal(t, "https://cdn/badge.png", b["icon"])
	assert.Equal(t, "info", b["color_kind"], "attendance 类应映射到 info 色系")
}

// TC-GRW-097：非见证记录不出现 badge 键
func TestAssembleFeedItems_NoBadgeOnNormalRecord(t *testing.T) {
	rows := []domain.GrowthDailyLog{feedRow(1, 1, "普通记录")}

	list := assembleFeedItems(rows, nil, nil, nil, nil, map[uint64]map[string]any{})

	_, exists := list[0]["badge"]
	assert.False(t, exists, "非见证记录不得出现 badge 键，否则前端要处理 null")
}

// TC-GRW-098：无见证的徽章不落到任何卡片（badgeByLog 为空即无徽章）
func TestAssembleFeedItems_BadgeWithoutWitnessNotShown(t *testing.T) {
	rows := []domain.GrowthDailyLog{feedRow(1, 1, "普通记录")}
	list := assembleFeedItems(rows, nil, nil, nil, nil, nil)
	_, exists := list[0]["badge"]
	assert.False(t, exists)
}

// badgeColorKind：色类只表达「哪套语义色」，不带具体色值
func TestBadgeColorKind_MapsCategoryToSemanticFamily(t *testing.T) {
	assert.Equal(t, "info", badgeColorKind("attendance"))
	assert.Equal(t, "info", badgeColorKind("points"))
	assert.Equal(t, "warning", badgeColorKind("growth"))
	assert.Equal(t, "warning", badgeColorKind("ART"))
	assert.Equal(t, "info", badgeColorKind("unknown-cat"))
	assert.Equal(t, "info", badgeColorKind(""))
}

// toBadge JSON 键集必须恰好 3 个——不得泄漏 Condition/TargetValue/Points 等后台配置
func TestToBadge_ExactFieldSet(t *testing.T) {
	b, err := json.Marshal(toBadge(domain.Achievement{
		Name: "x", Icon: "i", Category: "growth",
		Condition: "secret", TargetValue: 10, Points: 5, Status: 1,
	}))
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	assert.Len(t, m, 3, "badge JSON 只能有 name/icon/color_kind，实际: %v", m)
	for _, banned := range []string{"condition", "target_value", "points", "status", "id", "kindergarten_id", "description"} {
		_, exists := m[banned]
		assert.False(t, exists, "badge 不得含 %q", banned)
	}
}
