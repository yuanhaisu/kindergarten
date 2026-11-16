package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// RevertManifest 在单个事务内按 manifest 恢复原始到期日。
//
// 只撤销「当前值仍等于 apply 目标日」的行，避免覆盖之后合法的续期/调整：
//   - 当前 = 目标日 → 恢复 manifest 原始值；
//   - 当前 = manifest 原始值 → 视为已恢复，跳过（重复 revert 是 no-op）；
//   - 其他任何值 → 整体拒绝，列出具体授权，不落任何修改。
func RevertManifest(ctx context.Context, db *gorm.DB, opts Options, m *Manifest) (*WriteStats, error) {
	if m == nil || len(m.Rows) == 0 {
		return nil, errors.New("manifest 为空，拒绝执行")
	}
	return writeManifestTx(ctx, db, opts, m, "revert", "revert_trials",
		func(current map[uint64]lockedEntitlement) ([]rowUpdate, *WriteStats, error) {
			// 先全量检查再做任何写入：任一漂移都整体拒绝，事务里没有半成品。
			var drifted []string
			for _, want := range m.Rows {
				cur := current[want.EntitlementID]
				if cur.ExpiresAt.Equal(opts.TargetExpiry) || cur.ExpiresAt.Equal(want.ExpiresAt) {
					continue
				}
				drifted = append(drifted, fmt.Sprintf(
					"授权 %d 当前到期日 %s（期望目标日 %s 或 manifest 原始值 %s）",
					want.EntitlementID,
					cur.ExpiresAt.Format(manifestTimeLayout),
					opts.TargetExpiry.Format(manifestTimeLayout),
					want.ExpiresAt.Format(manifestTimeLayout)))
			}
			if len(drifted) > 0 {
				return nil, nil, fmt.Errorf(
					"拒绝回滚：以下授权已有后续变更，manifest 不再是它们的 apply 结果：%s",
					strings.Join(drifted, "；"))
			}

			s := &WriteStats{}
			updates := make([]rowUpdate, 0, len(m.Rows))
			for _, want := range m.Rows {
				if !current[want.EntitlementID].ExpiresAt.Equal(opts.TargetExpiry) {
					s.Unchanged++
					continue
				}
				updates = append(updates, rowUpdate{
					EntitlementID:  want.EntitlementID,
					KindergartenID: want.KindergartenID,
					ExpiresAt:      want.ExpiresAt,
				})
			}
			return updates, s, nil
		})
}
