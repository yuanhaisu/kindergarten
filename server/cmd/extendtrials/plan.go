package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

// Options 是命令的运行参数。政策日期由调用方按 Asia/Shanghai 构造：
// 固定时区是硬要求——DATETIME 列按墙钟时间存取，连接时区决定参数如何渲染。
type Options struct {
	Now             time.Time
	CutoffExclusive time.Time // created_at < CutoffExclusive 视为政策日（含）前开通
	TargetExpiry    time.Time // 统一延长到的到期时刻
	ManifestPath    string
	Apply           bool
	Revert          bool
	Operator        string
	Force           bool // dry-run 允许覆盖已存在的 manifest
}

// TrialGroup 是一所将被延长的园所及其待修改的 trial 授权汇总。
type TrialGroup struct {
	KindergartenID uint64
	Name           string
	TrialCount     int // 本次待延长的 trial 授权条数（不含已是目标日或更晚的行）
	OldEarliest    time.Time
	OldLatest      time.Time
}

// TrialRow 是一条将被修改的 trial 授权。manifest 逐条记录，revert 才能精确恢复
// 每条授权的原到期日；只按园所汇总会在同园到期日不一致时把它们抹平。
type TrialRow struct {
	KindergartenID uint64
	EntitlementID  uint64
	ExpiresAt      time.Time
}

// ExclusionReason 是园所被排除的机器可读原因。
type ExclusionReason string

const (
	ReasonCreatedAfterPolicy ExclusionReason = "created_after_policy_date"
	ReasonNotActive          ExclusionReason = "kindergarten_not_active"
	ReasonNoTrial            ExclusionReason = "no_trial_entitlement"
	ReasonOpenOrder          ExclusionReason = "open_order"
	ReasonActiveSubscription ExclusionReason = "active_subscription"
	ReasonNothingToExtend    ExclusionReason = "nothing_to_extend"
)

func (r ExclusionReason) Label() string {
	switch r {
	case ReasonCreatedAfterPolicy:
		return "政策日后开通"
	case ReasonNotActive:
		return "园所非在营状态（status ≠ active）"
	case ReasonNoTrial:
		return "无 trial 授权"
	case ReasonOpenOrder:
		return "存在在期订单（未取消且未到期，含未来起算）"
	case ReasonActiveSubscription:
		return "存在在期订阅授权"
	case ReasonNothingToExtend:
		return "trial 到期日已不早于目标日（无需修改）"
	default:
		return string(r)
	}
}

// Exclusion 记录一所被排除的园所。
type Exclusion struct {
	KindergartenID uint64
	Name           string
	Reason         ExclusionReason
}

// LegacyKindergarten 是 created_at 为空的遗留行：无法判定开通时间，
// 必须显式报告，不能静默当成「政策日前开通」纳入。
type LegacyKindergarten struct {
	KindergartenID uint64
	Name           string
}

// OrphanTrialGroup 是所属园所不存在或已软删的 trial 授权汇总。命令按园所维度
// 扫描，这些行不会进入 manifest；dry-run 单独报告，供运营确认是否另行处理。
type OrphanTrialGroup struct {
	KindergartenID uint64
	TrialCount     int
}

// Plan 是一次只读预演的结果。
type Plan struct {
	Selected      []TrialGroup
	Trials        []TrialRow // 被选中园所的待修改授权明细，逐条可回滚
	Excluded      []Exclusion
	NullCreatedAt []LegacyKindergarten
	OrphanTrials  []OrphanTrialGroup
	Scanned       int
}

// TotalTrialRows 返回将被修改的 trial 授权总行数。
func (p *Plan) TotalTrialRows() int {
	return len(p.Trials)
}

type kindergartenRow struct {
	ID        uint64
	Name      string
	Status    string
	CreatedAt *time.Time
}

type trialAggRow struct {
	KindergartenID uint64
	TotalRows      int // 该园全部存量 trial 授权行数（含已是目标日或更晚的行）
	ToExtend       int // 其中本次需要延长的行数
	OldEarliest    *time.Time
	OldLatest      *time.Time
}

// BuildPlan 按 ADR-0018 的存量园所口径做只读选择。它绝不写库。
//
// 口径：未删除、status='active'、created_at 早于政策日次日零点、存在
// source='trial' AND order_id IS NULL 的授权；排除存在在期订单（未取消且
// expires_at > now，含未来起算）或在期 subscription 授权（starts_at <= now <
// expires_at）的园所。只把 expires_at < TargetExpiry 的 trial 行计入待修改行：
// 已经延到目标日之后的行不动，重复执行影响行数为 0。
func BuildPlan(ctx context.Context, db *gorm.DB, opts Options) (*Plan, error) {
	var kgs []kindergartenRow
	if err := db.WithContext(ctx).Raw(
		"SELECT id, name, status, created_at FROM kindergartens WHERE deleted_at IS NULL ORDER BY id",
	).Scan(&kgs).Error; err != nil {
		return nil, fmt.Errorf("查询园所失败: %w", err)
	}

	plan := &Plan{Scanned: len(kgs)}
	orphans, err := loadOrphanTrialGroups(ctx, db, opts.TargetExpiry)
	if err != nil {
		return nil, err
	}
	plan.OrphanTrials = orphans

	remaining := make([]uint64, 0, len(kgs))
	for _, kg := range kgs {
		switch {
		case kg.Status != domain.KindergartenStatusActive:
			plan.Excluded = append(plan.Excluded, Exclusion{
				KindergartenID: kg.ID,
				Name:           kg.Name,
				Reason:         ReasonNotActive,
			})
		case kg.CreatedAt == nil:
			plan.NullCreatedAt = append(plan.NullCreatedAt, LegacyKindergarten{
				KindergartenID: kg.ID,
				Name:           kg.Name,
			})
		case !kg.CreatedAt.Before(opts.CutoffExclusive):
			plan.Excluded = append(plan.Excluded, Exclusion{
				KindergartenID: kg.ID,
				Name:           kg.Name,
				Reason:         ReasonCreatedAfterPolicy,
			})
		default:
			remaining = append(remaining, kg.ID)
		}
	}
	if len(remaining) == 0 {
		return plan, nil
	}

	trials, err := loadTrialAggregates(ctx, db, remaining, opts.TargetExpiry)
	if err != nil {
		return nil, err
	}
	blockedOrders, err := loadIDsWithOpenOrders(ctx, db, remaining, opts.Now)
	if err != nil {
		return nil, err
	}
	activeSubs, err := loadIDsWithActiveSubscription(ctx, db, remaining, opts.Now)
	if err != nil {
		return nil, err
	}

	nameByID := make(map[uint64]string, len(kgs))
	for _, kg := range kgs {
		nameByID[kg.ID] = kg.Name
	}

	for _, id := range remaining {
		agg, ok := trials[id]
		switch {
		case !ok || agg.TotalRows == 0:
			plan.Excluded = append(plan.Excluded, Exclusion{id, nameByID[id], ReasonNoTrial})
		case blockedOrders[id]:
			plan.Excluded = append(plan.Excluded, Exclusion{id, nameByID[id], ReasonOpenOrder})
		case activeSubs[id]:
			plan.Excluded = append(plan.Excluded, Exclusion{id, nameByID[id], ReasonActiveSubscription})
		case agg.ToExtend == 0:
			plan.Excluded = append(plan.Excluded, Exclusion{id, nameByID[id], ReasonNothingToExtend})
		default:
			plan.Selected = append(plan.Selected, TrialGroup{
				KindergartenID: id,
				Name:           nameByID[id],
				TrialCount:     agg.ToExtend,
				OldEarliest:    *agg.OldEarliest,
				OldLatest:      *agg.OldLatest,
			})
		}
	}
	sort.Slice(plan.Selected, func(i, j int) bool {
		return plan.Selected[i].KindergartenID < plan.Selected[j].KindergartenID
	})

	if len(plan.Selected) > 0 {
		selectedIDs := make([]uint64, 0, len(plan.Selected))
		for _, g := range plan.Selected {
			selectedIDs = append(selectedIDs, g.KindergartenID)
		}
		rows, err := loadTrialRows(ctx, db, selectedIDs, opts.TargetExpiry)
		if err != nil {
			return nil, err
		}
		plan.Trials = rows
		if err := verifyRowCounts(plan); err != nil {
			return nil, err
		}
	}
	return plan, nil
}

// verifyRowCounts 防御两次查询之间数据被并发修改：聚合数与明细数不一致时，
// 生成的 manifest 会与执行时看到的清单不符，必须报错而不是照写。
func verifyRowCounts(plan *Plan) error {
	byKg := make(map[uint64]int, len(plan.Selected))
	for _, row := range plan.Trials {
		byKg[row.KindergartenID]++
	}
	for _, g := range plan.Selected {
		if byKg[g.KindergartenID] != g.TrialCount {
			return fmt.Errorf(
				"园所 %d 的 trial 授权在读取过程中发生变化（聚合 %d 条，明细 %d 条），请重试",
				g.KindergartenID, g.TrialCount, byKg[g.KindergartenID])
		}
	}
	return nil
}

func loadTrialRows(ctx context.Context, db *gorm.DB, ids []uint64, target time.Time) ([]TrialRow, error) {
	type row struct {
		EntitlementID  uint64
		KindergartenID uint64
		ExpiresAt      time.Time
	}
	var dbRows []row
	if err := db.WithContext(ctx).Raw(`
SELECT id AS entitlement_id, kindergarten_id, expires_at
FROM entitlements
WHERE deleted_at IS NULL
  AND source = ?
  AND order_id IS NULL
  AND expires_at < ?
  AND kindergarten_id IN ?
ORDER BY kindergarten_id, id`,
		domain.EntitlementSourceTrial, target, ids,
	).Scan(&dbRows).Error; err != nil {
		return nil, fmt.Errorf("查询待修改 trial 明细失败: %w", err)
	}
	rows := make([]TrialRow, 0, len(dbRows))
	for _, r := range dbRows {
		rows = append(rows, TrialRow{
			KindergartenID: r.KindergartenID,
			EntitlementID:  r.EntitlementID,
			ExpiresAt:      r.ExpiresAt,
		})
	}
	return rows, nil
}

func loadTrialAggregates(ctx context.Context, db *gorm.DB, ids []uint64, target time.Time) (map[uint64]trialAggRow, error) {
	var rows []trialAggRow
	if err := db.WithContext(ctx).Raw(`
SELECT kindergarten_id,
       COUNT(*) AS total_rows,
       COUNT(CASE WHEN expires_at < ? THEN 1 END) AS to_extend,
       MIN(CASE WHEN expires_at < ? THEN expires_at END) AS old_earliest,
       MAX(CASE WHEN expires_at < ? THEN expires_at END) AS old_latest
FROM entitlements
WHERE deleted_at IS NULL
  AND source = ?
  AND order_id IS NULL
  AND kindergarten_id IN ?
GROUP BY kindergarten_id
ORDER BY kindergarten_id`,
		target, target, target, domain.EntitlementSourceTrial, ids,
	).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("查询 trial 授权失败: %w", err)
	}
	byID := make(map[uint64]trialAggRow, len(rows))
	for _, r := range rows {
		byID[r.KindergartenID] = r
	}
	return byID, nil
}

// loadIDsWithOpenOrders 找出存在「在期订单」的园所：未取消且未到期
// （expires_at > now）。不限制 starts_at —— 已付款但从未来起算的订单同样阻断，
// 否则 trial 延到目标日会与付费周期重叠。已取消订单不阻断；已过期的历史订单
// 也不阻断，本轮延长的目的正是覆盖 trial 到期后的空档。
func loadIDsWithOpenOrders(ctx context.Context, db *gorm.DB, ids []uint64, now time.Time) (map[uint64]bool, error) {
	var found []uint64
	if err := db.WithContext(ctx).Raw(`
SELECT DISTINCT kindergarten_id FROM orders
WHERE deleted_at IS NULL
  AND status <> ?
  AND expires_at > ?
  AND kindergarten_id IN ?`,
		domain.OrderStatusCancelled, now, ids,
	).Scan(&found).Error; err != nil {
		return nil, fmt.Errorf("查询订单失败: %w", err)
	}
	return toSet(found), nil
}

// loadIDsWithActiveSubscription 找出存在在期订阅授权的园所。到期边界与领域惯例
// 一致（EntitlementRepo.ListActiveByKindergarten）：starts_at <= now < expires_at。
func loadIDsWithActiveSubscription(ctx context.Context, db *gorm.DB, ids []uint64, now time.Time) (map[uint64]bool, error) {
	var found []uint64
	if err := db.WithContext(ctx).Raw(`
SELECT DISTINCT kindergarten_id FROM entitlements
WHERE deleted_at IS NULL
  AND source = ?
  AND starts_at <= ?
  AND expires_at > ?
  AND kindergarten_id IN ?`,
		domain.EntitlementSourceSubscription, now, now, ids,
	).Scan(&found).Error; err != nil {
		return nil, fmt.Errorf("查询订阅授权失败: %w", err)
	}
	return toSet(found), nil
}

// loadOrphanTrialGroups 找出所属园所不存在或已软删的 trial 授权。它们不在
// 园所扫描集合内，永远不会进入 manifest；必须显式报告，不能静默忽略。
func loadOrphanTrialGroups(ctx context.Context, db *gorm.DB, target time.Time) ([]OrphanTrialGroup, error) {
	var groups []OrphanTrialGroup
	if err := db.WithContext(ctx).Raw(`
SELECT e.kindergarten_id, COUNT(*) AS trial_count
FROM entitlements e
LEFT JOIN kindergartens k ON k.id = e.kindergarten_id AND k.deleted_at IS NULL
WHERE e.deleted_at IS NULL
  AND e.source = ?
  AND e.order_id IS NULL
  AND e.expires_at < ?
  AND k.id IS NULL
GROUP BY e.kindergarten_id
ORDER BY e.kindergarten_id`,
		domain.EntitlementSourceTrial, target,
	).Scan(&groups).Error; err != nil {
		return nil, fmt.Errorf("查询孤儿 trial 授权失败: %w", err)
	}
	return groups, nil
}

// revalidateKindergartens 在 apply 事务内按最终口径复核清单涉及的园所。
// dry-run 与 apply 之间可能相隔数小时到数天：期间园所可能被停用/删除，或出现
// 新的在期订单/订阅授权。任一园所不再符合口径即整体拒绝并点名，避免把试用
// 叠加到已付费授权上。revert 不做此复核——回滚必须能撤销已经发生的事实。
func revalidateKindergartens(ctx context.Context, tx *gorm.DB, kgIDs []uint64, now time.Time) error {
	if len(kgIDs) == 0 {
		return nil
	}
	type kgRow struct {
		ID        uint64
		Status    string
		DeletedAt *time.Time
	}
	var rows []kgRow
	if err := tx.WithContext(ctx).Raw(`
SELECT id, status, deleted_at
FROM kindergartens
WHERE id IN ?
ORDER BY id
FOR UPDATE`, kgIDs).Scan(&rows).Error; err != nil {
		return fmt.Errorf("复核园所状态失败: %w", err)
	}
	byID := make(map[uint64]kgRow, len(rows))
	for _, r := range rows {
		byID[r.ID] = r
	}

	var problems []string
	for _, id := range kgIDs {
		got, ok := byID[id]
		switch {
		case !ok:
			problems = append(problems, fmt.Sprintf("园所 %d 不存在", id))
		case got.DeletedAt != nil:
			problems = append(problems, fmt.Sprintf("园所 %d 已删除", id))
		case got.Status != domain.KindergartenStatusActive:
			problems = append(problems, fmt.Sprintf("园所 %d 状态为 %q（非在营）", id, got.Status))
		}
	}
	openOrders, err := loadIDsWithOpenOrders(ctx, tx, kgIDs, now)
	if err != nil {
		return err
	}
	activeSubs, err := loadIDsWithActiveSubscription(ctx, tx, kgIDs, now)
	if err != nil {
		return err
	}
	for _, id := range kgIDs {
		if openOrders[id] {
			problems = append(problems, fmt.Sprintf("园所 %d 存在在期订单（未取消且未到期）", id))
		}
		if activeSubs[id] {
			problems = append(problems, fmt.Sprintf("园所 %d 存在在期订阅授权", id))
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf(
			"apply 前置复核失败：清单内园所已不符合延长口径，拒绝执行（未修改任何行）：%s",
			strings.Join(problems, "；"))
	}
	return nil
}

func toSet(ids []uint64) map[uint64]bool {
	set := make(map[uint64]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set
}
