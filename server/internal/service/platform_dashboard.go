package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// PlatformDashboard 平台运营概览（design D7）：6 个指标 + 2 个待办。
// 一次聚合完成，金额口径与订单、收款、佣金明细一致，前端不拼多个列表接口。
type PlatformDashboard struct {
	KindergartenTotal         int64 `json:"kindergarten_total"`
	KindergartenActive        int64 `json:"kindergarten_active"`
	KindergartenExpiring30d   int64 `json:"kindergarten_expiring_30d"`
	ReceivedThisMonthCents    int64 `json:"received_this_month_cents"`
	OutstandingCents          int64 `json:"outstanding_cents"`
	CommissionSettleableCents int64 `json:"commission_settleable_cents"`

	// 7 天内到期园所待办。
	ExpiringKindergartens []PlatformExpiringKindergarten `json:"expiring_kindergartens"`
	// 按分销商聚合的可结算佣金待办。
	SettleableCommissions []PlatformSettleableCommission `json:"settleable_commissions"`
}

// PlatformExpiringKindergarten 7 天内最早到期的一所在营园所。
type PlatformExpiringKindergarten struct {
	ID        uint64    `json:"id"`
	Name      string    `json:"name"`
	ExpiresAt time.Time `json:"expires_at"`
}

// PlatformSettleableCommission 某个分销商当前可结算的佣金净额。
type PlatformSettleableCommission struct {
	DistributorID   uint64 `json:"distributor_id"`
	DistributorName string `json:"distributor_name"`
	AmountCents     int64  `json:"amount_cents"`
	CommissionCount int64  `json:"commission_count"`
}

// 看板两个到期窗口：指标看 30 天，待办只看 7 天。
const (
	dashboardExpiringWindow = 30 * 24 * time.Hour
	dashboardTodoWindow     = 7 * 24 * time.Hour
)

// Dashboard 聚合平台运营概览。
func (s *PlatformService) Dashboard(ctx context.Context) (*PlatformDashboard, error) {
	now := time.Now()

	total, err := s.kgRepo.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("count kindergartens: %w", err)
	}
	activeKgs, err := s.kgRepo.ListByStatus(ctx, domain.KindergartenStatusActive)
	if err != nil {
		return nil, fmt.Errorf("list active kindergartens: %w", err)
	}
	activeIDs := make([]uint64, 0, len(activeKgs))
	names := make(map[uint64]string, len(activeKgs))
	for _, kg := range activeKgs {
		activeIDs = append(activeIDs, kg.ID)
		names[kg.ID] = kg.Name
	}

	// 无在营园所时不能把空 ID 列表交给仓储（空表示不限园所）。
	var expiring30, expiring7 []domain.Entitlement
	if len(activeIDs) > 0 {
		until := now.Add(dashboardExpiringWindow)
		expiring30, err = s.entRepo.ListActiveExpiring(ctx, repository.ActiveExpiryFilter{
			KindergartenIDs: activeIDs, Now: now, Until: &until,
		})
		if err != nil {
			return nil, fmt.Errorf("list kindergartens expiring in 30 days: %w", err)
		}
		until = now.Add(dashboardTodoWindow)
		expiring7, err = s.entRepo.ListActiveExpiring(ctx, repository.ActiveExpiryFilter{
			KindergartenIDs: activeIDs, Now: now, Until: &until,
		})
		if err != nil {
			return nil, fmt.Errorf("list kindergartens expiring in 7 days: %w", err)
		}
	}

	expiring30IDs := make(map[uint64]struct{}, len(activeIDs))
	for _, e := range expiring30 {
		expiring30IDs[e.KindergartenID] = struct{}{}
	}
	// 每个园所只取最早到期的一条：7 天待办给出的是「最早到期授权落在 7 天内」。
	soonest := make(map[uint64]time.Time, len(expiring7))
	for _, e := range expiring7 {
		if at, ok := soonest[e.KindergartenID]; !ok || e.ExpiresAt.Before(at) {
			soonest[e.KindergartenID] = e.ExpiresAt
		}
	}
	todo := make([]PlatformExpiringKindergarten, 0, len(soonest))
	for id, at := range soonest {
		todo = append(todo, PlatformExpiringKindergarten{ID: id, Name: names[id], ExpiresAt: at})
	}
	sort.Slice(todo, func(i, j int) bool {
		if todo[i].ExpiresAt.Equal(todo[j].ExpiresAt) {
			return todo[i].ID < todo[j].ID
		}
		return todo[i].ExpiresAt.Before(todo[j].ExpiresAt)
	})

	// 「本月」按服务器本地时区取自然月，不做 UTC 换算（design D7）。
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	received, err := s.paymentRepo.SumReceivedBetween(ctx, monthStart, monthStart.AddDate(0, 1, 0))
	if err != nil {
		return nil, fmt.Errorf("sum received this month: %w", err)
	}
	outstanding, err := s.orderRepo.SumOutstanding(ctx)
	if err != nil {
		return nil, fmt.Errorf("sum outstanding: %w", err)
	}

	// 待结算佣金：读时按派生条件聚合（settleable + 已满 30 天的 pending），
	// 不在读路径推进台账状态（design D7）。合计由分组结果相加，保证指标与待办同口径。
	settleable, err := s.commRepo.SettleableByDistributor(ctx, now)
	if err != nil {
		return nil, fmt.Errorf("sum settleable commissions: %w", err)
	}
	distIDs := make([]uint64, 0, len(settleable))
	var settleableTotal int64
	for _, row := range settleable {
		distIDs = append(distIDs, row.DistributorID)
		settleableTotal += row.AmountCents
	}
	distNames := make(map[uint64]string, len(distIDs))
	if len(distIDs) > 0 {
		dists, err := s.distRepo.FindByIDs(ctx, distIDs)
		if err != nil {
			return nil, fmt.Errorf("load distributor names: %w", err)
		}
		for _, d := range dists {
			distNames[d.ID] = d.Name
		}
	}
	commissions := make([]PlatformSettleableCommission, 0, len(settleable))
	for _, row := range settleable {
		commissions = append(commissions, PlatformSettleableCommission{
			DistributorID: row.DistributorID, DistributorName: distNames[row.DistributorID],
			AmountCents: row.AmountCents, CommissionCount: row.Count,
		})
	}
	sort.Slice(commissions, func(i, j int) bool {
		if commissions[i].AmountCents == commissions[j].AmountCents {
			return commissions[i].DistributorID < commissions[j].DistributorID
		}
		return commissions[i].AmountCents > commissions[j].AmountCents
	})

	return &PlatformDashboard{
		KindergartenTotal:         total,
		KindergartenActive:        int64(len(activeKgs)),
		KindergartenExpiring30d:   int64(len(expiring30IDs)),
		ReceivedThisMonthCents:    received,
		OutstandingCents:          outstanding,
		CommissionSettleableCents: settleableTotal,
		ExpiringKindergartens:     todo,
		SettleableCommissions:     commissions,
	}, nil
}
