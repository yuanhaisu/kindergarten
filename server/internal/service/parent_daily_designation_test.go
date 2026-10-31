package service

// 家长接走人指定接口的前置校验回归：未选中孩子时服务层必须返回参数错误，
// handler 据此映射 422；落到 500 会让家长端把客户端参数问题当成服务端故障。

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDailyDesignation_RejectsMissingCurrentStudent(t *testing.T) {
	// 该分支在触库前返回，零值 service 即可覆盖，无需真实数据库。
	svc := &ParentPortalService{}
	data, err := svc.GetDailyDesignation(context.Background(), &ParentPortalContext{})
	require.ErrorIs(t, err, ErrParentInvalidParams)
	assert.Nil(t, data)
}
