package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"kindergarten-server/internal/domain"
)

// 覆盖 growthLogVisibilityWhere 的规则构造（design D1 / specs growth-visibility）。
// 该函数是家长端可见性规则的唯一实现点：feed/首页/相册/报告共用同一条件产出。
// 查询执行结果（是否真的过滤到数据）由集成验证承担——纯函数单测断言条件形状。

func TestGrowthVisibilityWhere_NoStudentsIsAlwaysFalse(t *testing.T) {
	where, args := growthLogVisibilityWhere(4, nil, nil)
	assert.Equal(t, "1 = 0", where, "无孩子时不得回退到班级范围，必须恒假")
	assert.Nil(t, args)
}

func TestGrowthVisibilityWhere_SingleStudentScopedToClassAndSelf(t *testing.T) {
	st := domain.Student{ClassID: 7}
	st.ID = 11
	where, args := growthLogVisibilityWhere(4, nil, &st)
	assert.True(t, strings.Contains(where, "growth_daily_logs.class_id = ?"), "必须限制在孩子所在班级")
	assert.True(t, strings.Contains(where, "target_type = 1 OR"), "全班记录必须可见")
	assert.True(t, strings.Contains(where, "student_id = ?"), "个体记录必须按学生命中")
	assert.Equal(t, []any{uint64(4), uint64(7), uint64(11)}, args,
		"参数顺序必须是 kg / class / student")
}

func TestGrowthVisibilityWhere_FamilyIncludesAllChildrenClasses(t *testing.T) {
	a := domain.Student{ClassID: 7}
	a.ID = 11
	b := domain.Student{ClassID: 9}
	b.ID = 12
	c := domain.Student{ClassID: 7} // 与 a 同班，应去重
	c.ID = 13

	where, args := growthLogVisibilityWhere(4, []domain.Student{a, b, c}, nil)
	assert.True(t, strings.Contains(where, "growth_daily_logs.class_id IN ?"), "多孩时按班级集合过滤")
	assert.True(t, strings.Contains(where, "student_id IN ?"), "个体记录按孩子集合命中")
	assert.Equal(t, uint64(4), args[0])
	classIDs, ok := args[1].([]uint64)
	assert.True(t, ok, "第二个参数应为班级 ID 集合")
	assert.ElementsMatch(t, []uint64{7, 9}, classIDs, "同班孩子只计一次")
	ids, ok := args[2].([]uint64)
	assert.True(t, ok, "第三个参数应为孩子 ID 集合")
	assert.ElementsMatch(t, []uint64{11, 12, 13}, ids)
}

func TestGrowthVisibilityWhere_StudentTakesPrecedenceOverFamily(t *testing.T) {
	a := domain.Student{ClassID: 7}
	a.ID = 11
	b := domain.Student{ClassID: 9}
	b.ID = 12
	target := b // 指定 b：只应看到 b 的班级与 b 的个体记录

	where, args := growthLogVisibilityWhere(4, []domain.Student{a, b}, &target)
	assert.True(t, strings.Contains(where, "growth_daily_logs.class_id = ?"), "指定孩子时收缩到单个班级")
	assert.Equal(t, []any{uint64(4), uint64(9), uint64(12)}, args)
}
