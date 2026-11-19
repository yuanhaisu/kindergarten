package service

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// ModuleAuthorizer 供中间件调用的模块授权判定。放在 service 包里，
// middleware 通过接口依赖它，避免反向 import（design D2）。
type ModuleAuthorizer interface {
	Decide(ctx context.Context, kindergartenID uint64, module string, isWrite bool) (ModuleDecision, error)
}

type ModuleDecision struct {
	Allowed bool
	Reason  string
}

// EntitlementService 园所级功能授权：逐模块到期判定、宽限期与降级。
type EntitlementService struct {
	ents    *repository.EntitlementRepo
	orders  *repository.OrderRepo
	kgs     *repository.KindergartenRepo
	cache   *EntitlementCache
	enforce bool
	now     func() time.Time
}

func NewEntitlementService(
	ents *repository.EntitlementRepo,
	orders *repository.OrderRepo,
	kgs *repository.KindergartenRepo,
	cache *EntitlementCache,
	enforce bool,
) *EntitlementService {
	return &EntitlementService{ents: ents, orders: orders, kgs: kgs, cache: cache, enforce: enforce, now: time.Now}
}

// Resolve 返回园所每个模块的可用性。base 始终可用，不查库。
func (s *EntitlementService) Resolve(ctx context.Context, kindergartenID uint64) (map[string]ModuleAccess, error) {
	if cached, ok := s.cache.Get(ctx, kindergartenID); ok {
		return cached, nil
	}
	now := s.now()
	rows, err := s.ents.ListByKindergarten(ctx, kindergartenID)
	if err != nil {
		return nil, err
	}
	out := make(map[string]ModuleAccess)
	for _, row := range rows {
		state := entitlementState(row.StartsAt, row.ExpiresAt, now, graceDuration)
		// 同一模块叠加时取最好的状态：active 优于 grace，grace 优于 expired。
		if prev, ok := out[row.ModuleCode]; ok && stateRank(prev.State) <= stateRank(state) {
			continue
		}
		out[row.ModuleCode] = ModuleAccess{State: state, ExpiresAt: row.ExpiresAt}
	}
	s.cache.Set(ctx, kindergartenID, out)
	return out, nil
}

// Decide 判定一次请求是否可以访问某个模块。
// base 永远允许；premium 在期或宽限期允许；降级后只允许读。
func (s *EntitlementService) Decide(ctx context.Context, kindergartenID uint64, module string, isWrite bool) (ModuleDecision, error) {
	if module == "" || ModuleCategory(module) != domain.ModuleCategoryPremium {
		return ModuleDecision{Allowed: true, Reason: "base"}, nil
	}
	if kindergartenID == 0 {
		return ModuleDecision{Allowed: false, Reason: "no_kindergarten"}, nil
	}
	access, err := s.Resolve(ctx, kindergartenID)
	if err != nil {
		return ModuleDecision{}, err
	}
	m, ok := access[module]
	if !ok {
		return s.deny("not_purchased"), nil
	}
	switch m.State {
	case "active", "grace":
		return ModuleDecision{Allowed: true, Reason: m.State}, nil
	default:
		if !isWrite {
			return ModuleDecision{Allowed: true, Reason: "read_only"}, nil
		}
		return s.deny("expired_read_only"), nil
	}
}

func (s *EntitlementService) deny(reason string) ModuleDecision {
	if !s.enforce {
		return ModuleDecision{Allowed: true, Reason: "would_deny:" + reason}
	}
	return ModuleDecision{Allowed: false, Reason: reason}
}

// IsEnforced 是否已开启强制拦截（开关关闭时只记录）。
func (s *EntitlementService) IsEnforced() bool { return s.enforce }

// Grant 人工发放/延长授权。来源为 gift，用于试点或补偿。
func (s *EntitlementService) Grant(ctx context.Context, kindergartenID uint64, module string, expiresAt time.Time) error {
	if !IsRegisteredModule(module) {
		return errUnknownModule(module)
	}
	if err := s.ents.CreateBatch(ctx, []domain.Entitlement{{
		KindergartenID: kindergartenID, ModuleCode: module,
		Source: domain.EntitlementSourceGift, StartsAt: s.now(), ExpiresAt: expiresAt,
	}}); err != nil {
		return err
	}
	return s.cache.Invalidate(ctx, kindergartenID)
}

// Invalidate 供订单/收款写入后清理缓存。
func (s *EntitlementService) Invalidate(ctx context.Context, kindergartenID uint64) error {
	return s.cache.Invalidate(ctx, kindergartenID)
}

// SubscriptionView 园所端「我的套餐」。
type SubscriptionView struct {
	PlanName   string                   `json:"plan_name"`
	Modules    []PlatformEntitlementRow `json:"modules"`
	ExpiresAt  *time.Time               `json:"expires_at"`
	GraceUntil *time.Time               `json:"grace_until"`
	State      string                   `json:"state"`
	// 超档提示：只展示，不阻断（ENT-0013）。
	StudentCount    int64 `json:"student_count"`
	TierMaxStudents int   `json:"tier_max_students"`
	OverLimit       bool  `json:"over_limit"`
}

func (s *EntitlementService) Subscription(ctx context.Context, kindergartenID uint64) (*SubscriptionView, error) {
	now := s.now()
	rows, err := s.ents.ListByKindergarten(ctx, kindergartenID)
	if err != nil {
		return nil, err
	}
	view := &SubscriptionView{Modules: make([]PlatformEntitlementRow, 0, len(rows))}
	if s.kgs != nil {
		if count, err := s.kgs.CountStudents(ctx, kindergartenID); err == nil {
			view.StudentCount = count
		}
	}
	if s.orders != nil {
		if o, err := s.orders.FindLatestActiveByKindergarten(ctx, kindergartenID); err == nil && o != nil {
			view.PlanName = snapshotPlanName(o.PlanSnapshot)
			var snap PlanSnapshot
			if err := json.Unmarshal([]byte(o.PlanSnapshot), &snap); err == nil && snap.TierMaxStudents > 0 {
				view.TierMaxStudents = snap.TierMaxStudents
				view.OverLimit = view.StudentCount > int64(snap.TierMaxStudents)
			}
		}
	}
	// 同一模块可能同时有试用与订阅两条授权；页面按模块展示一行，
	// 取状态最好的那条（同等状态下取更晚到期）。
	byModule := make(map[string]PlatformEntitlementRow, len(rows))
	for _, row := range rows {
		state := entitlementState(row.StartsAt, row.ExpiresAt, now, graceDuration)
		candidate := PlatformEntitlementRow{
			ModuleCode: row.ModuleCode, ModuleName: ModuleName(row.ModuleCode),
			Source: row.Source, StartsAt: row.StartsAt, ExpiresAt: row.ExpiresAt, State: state,
		}
		prev, ok := byModule[row.ModuleCode]
		if !ok || stateRank(state) < stateRank(prev.State) ||
			(stateRank(state) == stateRank(prev.State) && row.ExpiresAt.After(prev.ExpiresAt)) {
			byModule[row.ModuleCode] = candidate
		}
	}
	rank := 0
	for _, m := range byModule {
		view.Modules = append(view.Modules, m)
		if view.ExpiresAt == nil || m.ExpiresAt.Before(*view.ExpiresAt) {
			t := m.ExpiresAt
			view.ExpiresAt = &t
		}
		if r := stateRank(m.State); r > rank {
			rank = r
			view.State = m.State
		}
		if m.State == "grace" || m.State == "expired" {
			g := m.ExpiresAt.Add(graceDuration)
			if view.GraceUntil == nil || g.After(*view.GraceUntil) {
				view.GraceUntil = &g
			}
		}
	}
	sort.Slice(view.Modules, func(i, j int) bool { return view.Modules[i].ModuleCode < view.Modules[j].ModuleCode })
	if view.State == "" {
		view.State = "active"
	}
	return view, nil
}

// EntitlementModules 供两端小程序裁剪入口：只返回当前可用模块。
func (s *EntitlementService) EntitlementModules(ctx context.Context, kindergartenID uint64) ([]string, *time.Time, error) {
	access, err := s.Resolve(ctx, kindergartenID)
	if err != nil {
		return nil, nil, err
	}
	// base 始终可用；授权表里也可能有 base 行，用集合去重。
	seen := map[string]struct{}{"base": {}}
	modules := []string{"base"}
	var soonest *time.Time
	// 全部越过宽限期时没有可用 premium 模块，用最早的过期日兜底，
	// 保证两端仍能按过去时间显示「已到期」（#125）。base 不参与到期提示。
	var expiredSoonest *time.Time
	for code, a := range access {
		if code == "base" {
			continue
		}
		if a.State == "expired" {
			if expiredSoonest == nil || a.ExpiresAt.Before(*expiredSoonest) {
				t := a.ExpiresAt
				expiredSoonest = &t
			}
			continue
		}
		if _, ok := seen[code]; !ok {
			seen[code] = struct{}{}
			modules = append(modules, code)
		}
		if soonest == nil || a.ExpiresAt.Before(*soonest) {
			t := a.ExpiresAt
			soonest = &t
		}
	}
	if soonest == nil {
		soonest = expiredSoonest
	}
	return modules, soonest, nil
}

func stateRank(state string) int {
	switch state {
	case "active":
		return 0
	case "grace":
		return 1
	default:
		return 2
	}
}
