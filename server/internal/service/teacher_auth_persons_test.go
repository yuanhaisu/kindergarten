package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== StudentAuthorizedPersons ====================

// 场景：返回有效授权人（TC-NEW-PKU-AUTH-001）
func TestStudentAuthorizedPersons_ReturnsValid(t *testing.T) {
	f := newPickupFixture(t, "00:00", testPickupClass)
	f.makeAuthorization(t, "奶奶（张三）", 1, 1, nil)            // 有效
	authID := f.makeAuthorization(t, "爸爸（王明）", 1, 1, nil) // 有效

	result, err := f.svc.StudentAuthorizedPersons(ctx(), f.tc, f.studentID)
	require.NoError(t, err)
	list, ok := result["list"].([]map[string]any)
	require.True(t, ok)
	assert.Equal(t, 2, len(list))
	assert.Equal(t, 2, result["total"])
	_ = authID

	// 验证字段存在
	for _, item := range list {
		assert.NotEmpty(t, item["id"])
		assert.NotEmpty(t, item["name"])
		assert.NotEmpty(t, item["relation"])
		assert.NotEmpty(t, item["auth_type"])
	}
}

// 场景：过滤未审核/已撤销/已过期（TC-NEW-PKU-AUTH-002）
func TestStudentAuthorizedPersons_FiltersInvalid(t *testing.T) {
	f := newPickupFixture(t, "00:00", testPickupClass)

	// 有效
	f.makeAuthorization(t, "奶奶（张三）", 1, 1, nil)
	// 未审核（review_status=0）
	f.makeAuthorization(t, "待审核阿姨", 0, 1, nil)
	// 已撤销（status=0）
	f.makeAuthorization(t, "已撤销爷爷", 1, 0, nil)
	// 已过期
	expired := yesterdayEnd()
	f.makeAuthorization(t, "过期叔叔", 1, 1, &expired)

	result, err := f.svc.StudentAuthorizedPersons(ctx(), f.tc, f.studentID)
	require.NoError(t, err)
	list := result["list"].([]map[string]any)

	// 只有 1 条有效
	require.Equal(t, 1, len(list), "只应返回有效授权人")
	assert.Equal(t, "奶奶（张三）", list[0]["name"])
	assert.Equal(t, 1, result["total"])
}

// 场景：空列表（TC-NEW-PKU-AUTH-003）
func TestStudentAuthorizedPersons_EmptyList(t *testing.T) {
	f := newPickupFixture(t, "00:00", testPickupClass)

	result, err := f.svc.StudentAuthorizedPersons(ctx(), f.tc, f.studentID)
	require.NoError(t, err)
	list := result["list"].([]map[string]any)
	assert.Equal(t, 0, len(list))
	assert.Equal(t, 0, result["total"])
}

// 场景：学生不属于本班被拒
func TestStudentAuthorizedPersons_DeniesOtherClass(t *testing.T) {
	f := newPickupFixture(t, "00:00", testPickupClass)

	// 用另一个班的 tc 查本班学生
	otherTC := &TeacherPortalContext{
		UserID:         testPickupOperatorID,
		KindergartenID: testPickupKG,
		ClassID:        testPickupClass2, // 不同班
	}
	_, err := f.svc.StudentAuthorizedPersons(ctx(), otherTC, f.studentID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不属于本班")
}
