package service

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"kindergarten-server/internal/domain"
)

// 覆盖成长记录媒体入参的纯判定（TC-NEW-GMP-0010/0011 的规则前置、specs/growth-media-pipeline）。
// DB 侧的存在性/占用校验由 API 级集成验证承担——纯函数单测断言规则形状。

func TestValidateGrowthMediaPart_EmptyIsValid(t *testing.T) {
	ids, err := validateGrowthMediaPart(nil, nil)
	assert.NoError(t, err, "空数组合法（无附件记录）")
	assert.Nil(t, ids)
	ids, err = validateGrowthMediaPart([]MediaInput{}, map[uint64]bool{})
	assert.NoError(t, err)
	assert.Nil(t, ids)
}

func TestValidateGrowthMediaPart_OverLimitRejected(t *testing.T) {
	items := make([]MediaInput, 21)
	for i := range items {
		items[i] = MediaInput{FileID: uint64(i + 1)}
	}
	_, err := validateGrowthMediaPart(items, nil)
	assert.ErrorIs(t, err, ErrTeacherInvalidParams, "超过上限 20 必须拒绝")
}

func TestValidateGrowthMediaPart_AtLimitAccepted(t *testing.T) {
	items := make([]MediaInput, 20)
	for i := range items {
		items[i] = MediaInput{FileID: uint64(i + 1)}
	}
	ids, err := validateGrowthMediaPart(items, nil)
	assert.NoError(t, err, "恰好 20 合法")
	assert.Len(t, ids, 20)
}

func TestValidateGrowthMediaPart_ForeignMediaIDRejected(t *testing.T) {
	mine := map[uint64]bool{11: true, 12: true}
	items := []MediaInput{{ID: 99, FileID: 7}} // 99 不属于本记录
	_, err := validateGrowthMediaPart(items, mine)
	assert.ErrorIs(t, err, ErrTeacherInvalidParams, "编辑态引用非本记录 media.id 必须拒绝")
}

func TestValidateGrowthMediaPart_OwnMediaIDAccepted(t *testing.T) {
	mine := map[uint64]bool{11: true}
	items := []MediaInput{{ID: 11, FileID: 7, RefDesc: "保留项"}}
	ids, err := validateGrowthMediaPart(items, mine)
	assert.NoError(t, err)
	assert.True(t, ids[7], "本记录媒体可保留")
}

func TestValidateGrowthMediaPart_FileIDsDeduped(t *testing.T) {
	items := []MediaInput{{FileID: 5}, {FileID: 5}, {FileID: 6}}
	ids, err := validateGrowthMediaPart(items, nil)
	assert.NoError(t, err)
	assert.Len(t, ids, 2, "重复 file_id 去重后送校验")
}

// 关联到 domain 的编译期锚点：MediaInput 的字段语义与 domain.GrowthMedia.RefDesc 对应
// （引用层描述 L2），防止后续重构时字段拆散而测试静默失效。
var _ = domain.GrowthMedia{RefDesc: ""}
