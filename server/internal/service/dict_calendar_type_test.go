package service

// TC-NEW-CAL-0002/0003：日历类型字典 —— 园所自定义类型全端可读；系统内置项不可删除。
// 复用 calendar_semantics_test.go 的 calFixture（9903xx 区间，t.Cleanup 硬删）。

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

func TestDict_CalendarTypeItems(t *testing.T) {
	f := newCalFixture(t)
	ctx := context.Background()
	svc := NewDictService(f.db, repository.NewDictRepo(f.db))

	var typ domain.SysDictType
	require.NoError(t, f.db.WithContext(ctx).
		Where("kindergarten_id = ? AND type_key = ?", testCalKG, "calendar_event_type").
		First(&typ).Error)

	// 0003：内置项删除被拒且保持可用
	var holiday domain.SysDictItem
	require.NoError(t, f.db.WithContext(ctx).
		Where("type_id = ? AND item_key = ?", typ.ID, "holiday").First(&holiday).Error)
	err := svc.DeleteItem(ctx, testCalKG, holiday.ID)
	require.ErrorIs(t, err, ErrDictSystemType)
	var still int64
	require.NoError(t, f.db.WithContext(ctx).Model(&domain.SysDictItem{}).
		Where("id = ?", holiday.ID).Count(&still).Error)
	require.EqualValues(t, 1, still, "删除被拒后内置项仍在")

	// 跨园删除也不成立（字典项归属校验）
	require.ErrorIs(t, svc.DeleteItem(ctx, 990999, holiday.ID), ErrDictItemNotFound)

	// 0002：园所自定义项（老师端选择器来源）与家长端渲染元数据同源
	custom := domain.SysDictItem{
		TypeID: typ.ID, ItemKey: "interest", ItemLabel: "兴趣班",
		Extra: `{"color":"#0EA5E9"}`, SortOrder: 6, IsEnabled: true,
	}
	require.NoError(t, f.db.WithContext(ctx).Create(&custom).Error)

	resp, err := svc.GetDictItems(ctx, testCalKG, "calendar_event_type")
	require.NoError(t, err)
	var found *DictItemPublic
	for i := range resp.Items {
		if resp.Items[i].ItemKey == "interest" {
			found = &resp.Items[i]
		}
	}
	require.NotNil(t, found, "老师端选择器应包含自定义类型")
	require.Equal(t, "兴趣班", found.ItemLabel)
	var extra struct {
		Color string `json:"color"`
	}
	require.NoError(t, json.Unmarshal(found.Extra, &extra))
	require.Equal(t, "#0EA5E9", extra.Color)

	// 服务端映射（家长端事件响应里的 label/color）同样取自该字典
	meta, err := loadCalendarTypeMeta(ctx, f.db, testCalKG)
	require.NoError(t, err)
	require.Equal(t, "兴趣班", meta["interest"].Label)
	require.Equal(t, "#0EA5E9", meta["interest"].Color)
}
