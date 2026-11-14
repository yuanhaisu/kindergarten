package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// 停用状态以数据库为准；缓存不参与本用例（nil Redis 走直查）。
func TestKindergartenStatusService_SuspendedReadsDB(t *testing.T) {
	db := openPlatformTestDB(t)
	ctx := context.Background()
	cleanupBillingFlow(t, db)
	t.Cleanup(func() { cleanupBillingFlow(t, db) })

	repo := repository.NewKindergartenRepo(db)
	kg := &domain.Kindergarten{Name: "S1B订单测试园F", Status: domain.KindergartenStatusActive}
	require.NoError(t, repo.Create(ctx, kg))

	svc := NewKindergartenStatusService(repo, nil)
	suspended, err := svc.Suspended(ctx, kg.ID)
	require.NoError(t, err)
	assert.False(t, suspended, "在营园所不应被判定为停用")

	require.NoError(t, repo.UpdateByID(ctx, kg.ID, map[string]any{"status": domain.KindergartenStatusSuspended}))
	suspended, err = svc.Suspended(ctx, kg.ID)
	require.NoError(t, err)
	assert.True(t, suspended, "停用后必须立刻读到新状态")

	// 缓存失效在无 Redis 时是空操作，不报错
	require.NoError(t, svc.Invalidate(ctx, kg.ID))
}
