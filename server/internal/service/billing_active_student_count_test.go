package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// 本文件的数据统一用 S1Bact- 前缀，清理只按该前缀，避免与其他并发用例互删。
const activeCountPrefix = "S1Bact-"

// 套餐 code 刻意不以 s1b 开头：MySQL 的 LIKE 里 `_` 是单字符通配，
// 其他用例的 `code LIKE 's1b_%'` 会误伤 S1Bact- 前缀且不区分大小写。
const activeCountPlanCode = "act5_s1bact_plan"

func cleanupActiveCountTest(t *testing.T, db *gorm.DB) {
	t.Helper()
	del := func(sql string, args ...any) {
		if err := db.Exec(sql, args...).Error; err != nil {
			t.Logf("cleanup %q: %v", sql, err)
		}
	}
	sub := "SELECT id FROM kindergartens WHERE name LIKE 'S1Bact-%'"
	del("DELETE FROM commission_ledger WHERE kindergarten_id IN (" + sub + ")")
	del("DELETE FROM payments WHERE kindergarten_id IN (" + sub + ")")
	del("DELETE FROM entitlements WHERE kindergarten_id IN (" + sub + ")")
	del("DELETE FROM orders WHERE kindergarten_id IN (" + sub + ")")
	del("DELETE FROM students WHERE kindergarten_id IN (" + sub + ")")
	del("DELETE FROM classes WHERE kindergarten_id IN (" + sub + ")")
	del("DELETE FROM kindergartens WHERE name LIKE 'S1Bact-%'")
	del("DELETE FROM plan_tiers WHERE plan_id IN (SELECT id FROM plans WHERE code = ?)", activeCountPlanCode)
	del("DELETE FROM plan_modules WHERE plan_id IN (SELECT id FROM plans WHERE code = ?)", activeCountPlanCode)
	del("DELETE FROM plans WHERE code = ?", activeCountPlanCode)
}

// createActiveCountStudent 插入一名指定状态的幼儿。
// students.status 带 default:1，GORM 会省略零值，待入学(0) 必须插入后显式更新。
func createActiveCountStudent(t *testing.T, db *gorm.DB, kgID, classID uint64, status int8) *domain.Student {
	t.Helper()
	st := &domain.Student{
		KindergartenID: kgID, ClassID: classID, Name: activeCountPrefix + "幼儿", Status: 1,
		Birthday: time.Now(), EnrollDate: time.Now(),
	}
	require.NoError(t, db.Create(st).Error)
	if status != 1 {
		require.NoError(t, db.Model(st).Update("status", status).Error)
		st.Status = status
	}
	return st
}

// 在园幼儿数只算「在读且未删除」：2 名在读 + 5 名待入学/毕业/退学
// 必须落第一档，不能推高档位，也不能误报超档（issue #5）。
func TestActiveStudentCount_DrivesTierAndOverLimit(t *testing.T) {
	db := openPlatformTestDB(t)
	planSvc, orderSvc, entSvc := newBillingFlowServices(t, db)
	platformSvc := newPlatformTestService(t, db)
	ctx := context.Background()
	cleanupActiveCountTest(t, db)
	t.Cleanup(func() { cleanupActiveCountTest(t, db) })

	kg := &domain.Kindergarten{Name: activeCountPrefix + "园", Status: domain.KindergartenStatusActive}
	require.NoError(t, repository.NewKindergartenRepo(db).Create(ctx, kg))
	class := &domain.Class{KindergartenID: kg.ID, Name: activeCountPrefix + "班", Capacity: 60, Status: 1}
	require.NoError(t, db.Create(class).Error)

	for range 2 {
		createActiveCountStudent(t, db, kg.ID, class.ID, 1)
	}
	// 待入学 / 毕业 / 退学各命中一次，合计 5 名不计数。
	createActiveCountStudent(t, db, kg.ID, class.ID, 0)
	createActiveCountStudent(t, db, kg.ID, class.ID, 2)
	createActiveCountStudent(t, db, kg.ID, class.ID, 2)
	createActiveCountStudent(t, db, kg.ID, class.ID, 3)
	createActiveCountStudent(t, db, kg.ID, class.ID, 3)
	// 已软删的在读也不计数。
	removed := createActiveCountStudent(t, db, kg.ID, class.ID, 1)
	require.NoError(t, db.Delete(removed).Error)

	plan, err := planSvc.Create(ctx, PlanRequest{
		Code: activeCountPlanCode, Name: activeCountPrefix + "档位版", PriceCents: 100000, TermMonths: 6,
		ModuleCodes: []string{"base", "growth"},
	}, "s1bact_ops", "127.0.0.1")
	require.NoError(t, err)
	require.NoError(t, planSvc.ReplaceTiers(ctx, plan.ID, []domain.PlanTier{
		{Tier: 1, MinStudents: 0, MaxStudents: 2, PriceMultiplier: 100},
		{Tier: 2, MinStudents: 3, MaxStudents: 0, PriceMultiplier: 150},
	}, "s1bact_ops", "127.0.0.1"))

	order, err := orderSvc.Create(ctx, CreateOrderRequest{KindergartenID: kg.ID, PlanID: plan.ID}, "s1bact_ops", "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, 1, order.StudentTier, "2 名在读应落第一档，毕业/退学/待入学不推高档位")
	assert.EqualValues(t, 100000, order.AmountDueCents, "第一档按 100% 计价")

	var snap PlanSnapshot
	require.NoError(t, json.Unmarshal([]byte(order.PlanSnapshot), &snap))
	assert.EqualValues(t, 2, snap.StudentCount, "快照里的在园幼儿数只含在读")

	_, _, err = orderSvc.Confirm(ctx, order.ID, "测试信用开通", "s1bact_ops", "127.0.0.1")
	require.NoError(t, err)

	view, err := entSvc.Subscription(ctx, kg.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 2, view.StudentCount)
	assert.Equal(t, 2, view.TierMaxStudents)
	assert.False(t, view.OverLimit, "在读 2 人未超过档位上限 2，不得误报超档")

	rows, _, err := platformSvc.ListKindergartens(ctx, 1, 20, activeCountPrefix, "")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.EqualValues(t, 2, rows[0].StudentCount, "平台列表与档位提示必须同一口径")
	assert.Equal(t, 2, rows[0].TierMaxStudents)
	assert.False(t, rows[0].OverLimit)
}
