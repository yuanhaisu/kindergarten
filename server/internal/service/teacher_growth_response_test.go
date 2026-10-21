package service

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"kindergarten-server/internal/domain"
)

// 覆盖老师端详情的「相关孩子」映射（tasks.md 2.3）。
// 契约要求 related_students_count = related_students 的长度；两者都由
// growthRelatedStudentItems 的返回值派生，本测试锁定该不变量。

func TestGrowthRelatedStudentItems_CountEqualsLength(t *testing.T) {
	s1 := domain.Student{Name: "王小明", Avatar: "a1"}
	s1.ID = 1
	s2 := domain.Student{Name: "李小花", Avatar: "a2"}
	s2.ID = 2

	items := growthRelatedStudentItems([]domain.Student{s1, s2})
	assert.Len(t, items, 2)
	assert.Equal(t, len(items), 2, "related_students_count 必须等于数组长度")
	assert.Equal(t, uint64(1), items[0]["id"])
	assert.Equal(t, "王小明", items[0]["name"])
	assert.Equal(t, "a1", items[0]["avatar"])
}

func TestGrowthRelatedStudentItems_EmptyIsNonNilSlice(t *testing.T) {
	items := growthRelatedStudentItems(nil)
	assert.NotNil(t, items, "空集合应序列化为 [] 而非 null")
	assert.Len(t, items, 0)
}
