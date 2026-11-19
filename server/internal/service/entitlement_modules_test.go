package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// TC-NEW-ENT-0023：全部 premium 模块越过宽限期后，仍要返回到期日供两端显示「已到期」提示（#125）。
func TestEntitlementModules_ExpiredFallback(t *testing.T) {
	db := openPlatformTestDB(t)
	_, _, entSvc := newBillingFlowServices(t, db)
	ctx := context.Background()
	cleanupBillingFlow(t, db)
	t.Cleanup(func() { cleanupBillingFlow(t, db) })

	newKindergarten := func(name string) *domain.Kindergarten {
		kg := &domain.Kindergarten{Name: name, Status: domain.KindergartenStatusActive}
		require.NoError(t, repository.NewKindergartenRepo(db).Create(ctx, kg))
		return kg
	}

	t.Run("全部越过宽限期时取最早过期日兜底", func(t *testing.T) {
		kg := newKindergarten("S1B订单测试园T125A")
		now := time.Now()
		earliest := now.Add(-20 * 24 * time.Hour)
		require.NoError(t, repository.NewEntitlementRepo(db).CreateBatch(ctx, []domain.Entitlement{
			// base 行即使更早到期也不参与到期提示。
			{KindergartenID: kg.ID, ModuleCode: "base", Source: domain.EntitlementSourceGift, StartsAt: now.AddDate(-1, 0, 0), ExpiresAt: now.Add(-100 * 24 * time.Hour)},
			{KindergartenID: kg.ID, ModuleCode: "growth", Source: domain.EntitlementSourceTrial, StartsAt: now.AddDate(0, -2, 0), ExpiresAt: now.Add(-16 * 24 * time.Hour)},
			{KindergartenID: kg.ID, ModuleCode: "diet", Source: domain.EntitlementSourceTrial, StartsAt: now.AddDate(0, -2, 0), ExpiresAt: earliest},
			{KindergartenID: kg.ID, ModuleCode: "album", Source: domain.EntitlementSourceTrial, StartsAt: now.AddDate(0, -2, 0), ExpiresAt: now.Add(-18 * 24 * time.Hour)},
		}))

		modules, expiresAt, err := entSvc.EntitlementModules(ctx, kg.ID)
		require.NoError(t, err)
		assert.Equal(t, []string{"base"}, modules, "过期模块不再出现在可用列表")
		require.NotNil(t, expiresAt, "全部越过宽限期仍需返回到期日，否则前端不显示已到期提示")
		assert.True(t, expiresAt.Before(time.Now()), "兜底到期日必须是过去时间")
		assert.WithinDuration(t, earliest, *expiresAt, time.Second, "兜底取最早的 expired 到期日，且跳过 base")
	})

	t.Run("宽限期内行为不变", func(t *testing.T) {
		kg := newKindergarten("S1B订单测试园T125B")
		now := time.Now()
		graceExpiry := now.Add(-24 * time.Hour)
		require.NoError(t, repository.NewEntitlementRepo(db).CreateBatch(ctx, []domain.Entitlement{
			{KindergartenID: kg.ID, ModuleCode: "growth", Source: domain.EntitlementSourceTrial, StartsAt: now.AddDate(0, -2, 0), ExpiresAt: graceExpiry},
		}))

		modules, expiresAt, err := entSvc.EntitlementModules(ctx, kg.ID)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"base", "growth"}, modules, "宽限期内的模块仍可用")
		require.NotNil(t, expiresAt)
		assert.WithinDuration(t, graceExpiry, *expiresAt, time.Second)
	})

	t.Run("混合态取在期模块到期日", func(t *testing.T) {
		kg := newKindergarten("S1B订单测试园T125C")
		now := time.Now()
		activeExpiry := now.AddDate(0, 1, 0)
		require.NoError(t, repository.NewEntitlementRepo(db).CreateBatch(ctx, []domain.Entitlement{
			{KindergartenID: kg.ID, ModuleCode: "growth", Source: domain.EntitlementSourceSubscription, StartsAt: now.AddDate(0, -1, 0), ExpiresAt: activeExpiry},
			{KindergartenID: kg.ID, ModuleCode: "diet", Source: domain.EntitlementSourceTrial, StartsAt: now.AddDate(0, -2, 0), ExpiresAt: now.Add(-20 * 24 * time.Hour)},
		}))

		modules, expiresAt, err := entSvc.EntitlementModules(ctx, kg.ID)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"base", "growth"}, modules)
		require.NotNil(t, expiresAt)
		assert.WithinDuration(t, activeExpiry, *expiresAt, time.Second, "有在期模块时不使用过期模块兜底")
	})

	t.Run("无任何授权行仍返回空到期日", func(t *testing.T) {
		kg := newKindergarten("S1B订单测试园T125D")

		modules, expiresAt, err := entSvc.EntitlementModules(ctx, kg.ID)
		require.NoError(t, err)
		assert.Equal(t, []string{"base"}, modules)
		assert.Nil(t, expiresAt, "没有授权行不能凭空提示已到期")
	})
}
