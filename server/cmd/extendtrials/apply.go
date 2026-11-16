package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

// WriteStats 汇总一次写操作的实际影响，用于终端输出与平台操作日志。
type WriteStats struct {
	Kindergartens   int
	Rows            int
	Unchanged       int
	KindergartenIDs []uint64
	LogID           uint64
}

// operationDetail 是 platform_operation_logs.detail 的固定结构。
type operationDetail struct {
	Mode            string   `json:"mode"`
	Kindergartens   int      `json:"kindergartens"`
	KindergartenIDs []uint64 `json:"kindergarten_ids"`
	Rows            int      `json:"rows"`
	Unchanged       int      `json:"unchanged"`
	TargetExpiry    string   `json:"target_expiry"`
	ManifestSHA256  string   `json:"manifest_sha256"`
}

type lockedEntitlement struct {
	ID             uint64
	KindergartenID uint64
	ExpiresAt      time.Time
	Source         string
	OrderID        *uint64
	DeletedAt      *time.Time
	KgDeletedAt    *time.Time
}

// rowUpdate 是一条待写回的授权到期日变更。
type rowUpdate struct {
	EntitlementID  uint64
	KindergartenID uint64
	ExpiresAt      time.Time
}

// writeManifestTx 是 apply/revert 共用的事务骨架：
// 事务内前置复核（仅 apply）→ 行锁并校验清单 → decide 全量判定并给出写入计划 →
// 逐行 UPDATE → 写平台操作日志。任一步返回错误时整体拒绝，事务内不落任何写入。
func writeManifestTx(
	ctx context.Context,
	db *gorm.DB,
	opts Options,
	m *Manifest,
	mode, action string,
	precheck func(ctx context.Context, tx *gorm.DB) error,
	decide func(current map[uint64]lockedEntitlement) ([]rowUpdate, *WriteStats, error),
) (*WriteStats, error) {
	var stats *WriteStats
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if precheck != nil {
			if err := precheck(ctx, tx); err != nil {
				return err
			}
		}
		current, err := lockManifestRows(ctx, tx, m)
		if err != nil {
			return err
		}
		updates, s, err := decide(current)
		if err != nil {
			return err
		}
		kgSet := make(map[uint64]bool)
		for _, u := range updates {
			if err := tx.Exec(
				"UPDATE entitlements SET expires_at = ?, updated_at = ? WHERE id = ?",
				u.ExpiresAt, opts.Now, u.EntitlementID,
			).Error; err != nil {
				return fmt.Errorf("更新授权 %d 失败: %w", u.EntitlementID, err)
			}
			s.Rows++
			kgSet[u.KindergartenID] = true
		}
		s.KindergartenIDs = sortedIDs(kgSet)
		s.Kindergartens = len(s.KindergartenIDs)
		if err := writeOperationLog(tx, opts, action, mode, s, m); err != nil {
			return err
		}
		stats = s
		return nil
	})
	if err != nil {
		return nil, err
	}
	return stats, nil
}

// ApplyManifest 在单个事务内把清单内仍早于目标日的 trial 授权延到目标日。
//
// 只延长不回退：到期日已等于或晚于目标日的行保持不变（重复执行影响行数为 0）。
// 清单与数据库漂移（授权消失、被删、换了园所、不再是 trial）时整体拒绝，
// 不在运营确认过的清单之外猜意图。dry-run 可能已是数天前的产物，因此事务内
// 还要按最终口径复核园所（未删除、在营、无在期订单/订阅），不再符合即整体拒绝。
func ApplyManifest(ctx context.Context, db *gorm.DB, opts Options, m *Manifest) (*WriteStats, error) {
	if m == nil || len(m.Rows) == 0 {
		return nil, errors.New("manifest 为空，拒绝执行")
	}
	return writeManifestTx(ctx, db, opts, m, "apply", "extend_trials",
		func(ctx context.Context, tx *gorm.DB) error {
			return revalidateKindergartens(ctx, tx, m.Kindergartens(), opts.Now)
		},
		func(current map[uint64]lockedEntitlement) ([]rowUpdate, *WriteStats, error) {
			s := &WriteStats{}
			updates := make([]rowUpdate, 0, len(m.Rows))
			for _, want := range m.Rows {
				if !current[want.EntitlementID].ExpiresAt.Before(opts.TargetExpiry) {
					s.Unchanged++
					continue
				}
				updates = append(updates, rowUpdate{
					EntitlementID:  want.EntitlementID,
					KindergartenID: want.KindergartenID,
					ExpiresAt:      opts.TargetExpiry,
				})
			}
			return updates, s, nil
		})
}

// lockManifestRows 校验清单每行在库中仍是同一园所的 trial 授权，并加行锁。
// 任一不一致都返回聚合错误；事务因此不会写入任何一行。
func lockManifestRows(ctx context.Context, tx *gorm.DB, m *Manifest) (map[uint64]lockedEntitlement, error) {
	ids := make([]uint64, 0, len(m.Rows))
	for _, r := range m.Rows {
		ids = append(ids, r.EntitlementID)
	}
	var rows []lockedEntitlement
	if err := tx.Raw(`
SELECT e.id, e.kindergarten_id, e.expires_at, e.source, e.order_id,
       e.deleted_at, k.deleted_at AS kg_deleted_at
FROM entitlements e
JOIN kindergartens k ON k.id = e.kindergarten_id
WHERE e.id IN ?
ORDER BY e.id
FOR UPDATE`, ids).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("锁定清单授权失败: %w", err)
	}
	byID := make(map[uint64]lockedEntitlement, len(rows))
	for _, r := range rows {
		byID[r.ID] = r
	}

	var problems []string
	for _, want := range m.Rows {
		got, ok := byID[want.EntitlementID]
		switch {
		case !ok:
			problems = append(problems, fmt.Sprintf("授权 %d 不存在", want.EntitlementID))
		case got.DeletedAt != nil:
			problems = append(problems, fmt.Sprintf("授权 %d 已被删除", want.EntitlementID))
		case got.KindergartenID != want.KindergartenID:
			problems = append(problems, fmt.Sprintf(
				"授权 %d 属于园所 %d，与清单中的园所 %d 不符",
				want.EntitlementID, got.KindergartenID, want.KindergartenID))
		case got.KgDeletedAt != nil:
			problems = append(problems, fmt.Sprintf("园所 %d 已被删除", got.KindergartenID))
		case got.Source != domain.EntitlementSourceTrial || got.OrderID != nil:
			problems = append(problems, fmt.Sprintf(
				"授权 %d 已不是存量 trial（source=%s, order_id=%s）",
				want.EntitlementID, got.Source, formatOrderID(got.OrderID)))
		}
	}
	if len(problems) > 0 {
		return nil, fmt.Errorf("manifest 与数据库不一致，已拒绝执行（未修改任何行）：%s", strings.Join(problems, "；"))
	}
	return byID, nil
}

func formatOrderID(id *uint64) string {
	if id == nil {
		return "NULL"
	}
	return fmt.Sprintf("%d", *id)
}

// writeOperationLog 在写事务内落一条平台操作留痕；失败即整体回滚。
func writeOperationLog(tx *gorm.DB, opts Options, action, mode string, s *WriteStats, m *Manifest) error {
	detail, err := json.Marshal(operationDetail{
		Mode:            mode,
		Kindergartens:   s.Kindergartens,
		KindergartenIDs: s.KindergartenIDs,
		Rows:            s.Rows,
		Unchanged:       s.Unchanged,
		TargetExpiry:    opts.TargetExpiry.Format(manifestTimeLayout),
		ManifestSHA256:  m.SHA256,
	})
	if err != nil {
		return fmt.Errorf("生成平台操作日志详情失败: %w", err)
	}
	entry := &domain.PlatformOperationLog{
		Operator:   opts.Operator,
		Module:     "billing",
		Action:     action,
		TargetType: "entitlement_batch",
		Detail:     string(detail),
	}
	if err := tx.Create(entry).Error; err != nil {
		return fmt.Errorf("写平台操作日志失败: %w", err)
	}
	s.LogID = entry.ID
	return nil
}

func sortedIDs(set map[uint64]bool) []uint64 {
	ids := make([]uint64, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}
