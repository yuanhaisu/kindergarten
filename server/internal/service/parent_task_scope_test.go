package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"kindergarten-server/internal/domain"
)

// 覆盖 taskScopeWhere：家长端任务可见性（班级任务按班级、个体任务按目标孩子）的唯一实现点。
// 条件是否真的过滤到数据由集成验证承担，这里只钉住条件形状与参数顺序，
// 防止「个体任务按班级可见」「漏掉园所隔离」两类回归。

func taskScopeCtx(kg, classID, studentID uint64) *ParentPortalContext {
	st := domain.Student{ClassID: classID}
	st.ID = studentID
	return &ParentPortalContext{KindergartenID: kg, CurrentStudent: &st}
}

func TestTaskScopeWhere_HasTenantAndClassAndStudentRules(t *testing.T) {
	where, args := taskScopeWhere(taskScopeCtx(4, 6, 4))

	assert.True(t, strings.Contains(where, "kindergarten_id = ?"), "必须带园所隔离")
	assert.True(t, strings.Contains(where, "target_student_id = 0 AND class_id = ?"),
		"班级任务必须同时满足 target_student_id=0 与 class_id")
	assert.True(t, strings.Contains(where, "OR target_student_id = ?"),
		"个体任务必须按目标孩子命中")
	assert.Equal(t, []any{uint64(4), uint64(6), uint64(4)}, args,
		"参数顺序必须是 kg / class / student")
}

func TestTaskScopeWhere_ClassPredicateNeverEscapesIndividualGuard(t *testing.T) {
	where, _ := taskScopeWhere(taskScopeCtx(4, 6, 4))

	// class_id 只能出现在 (target_student_id = 0 AND class_id = ?) 里；
	// 一旦出现 OR class_id = ?，个体任务就会对整个班级可见。
	assert.NotContains(t, where, "OR class_id = ?", "禁止把班级条件放宽成 OR，否则个体任务全班可见")
	assert.Equal(t, 1, strings.Count(where, "class_id = ?"), "class_id 条件只应出现一次")
	assert.Equal(t, 1, strings.Count(where, "target_student_id = ?"), "个体任务条件只应出现一次")
}

func TestTaskScopeWhere_UsesCurrentStudentNotFirstSibling(t *testing.T) {
	// 多孩家庭：条件必须取 CurrentStudent，而不是 Students[0]。
	st := domain.Student{ClassID: 9}
	st.ID = 32
	sibling := domain.Student{ClassID: 6}
	sibling.ID = 27
	pc := &ParentPortalContext{
		KindergartenID: 4,
		CurrentStudent: &st,
		Students:       []domain.Student{sibling},
	}
	where, args := taskScopeWhere(pc)

	assert.Equal(t, uint64(9), args[1], "班级条件必须取当前孩子的班级")
	assert.Equal(t, uint64(32), args[2], "个体任务条件必须取当前孩子的 id")
	assert.NotContains(t, where, "IN ?", "单孩上下文不应退化成班级集合")
}
