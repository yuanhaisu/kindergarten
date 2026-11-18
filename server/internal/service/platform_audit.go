package service

import (
	"context"
	"log/slog"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// PlatformAuditWriter 平台审计写入接缝。生产实现是 repository.PlatformLogRepo；
// 测试通过注入失败实现，验证「审计写入失败时强一致操作回滚」与
// 「best-effort 操作不被审计故障阻断」两条规则（design D6）。
type PlatformAuditWriter interface {
	Create(ctx context.Context, row *domain.PlatformOperationLog) error
	CreateTx(ctx context.Context, tx *gorm.DB, row *domain.PlatformOperationLog) error
}

// PlatformAuditStore 审计写入 + 查询。PlatformService 同时承载审计查询端点。
type PlatformAuditStore interface {
	PlatformAuditWriter
	List(ctx context.Context, f repository.PlatformLogFilter) ([]domain.PlatformOperationLog, int64, error)
}

// writeAuditBestEffort 写平台审计但不阻断业务。仅用于 best-effort 清单
// （套餐 / 分销商目录变更、园所开通后的膳食初始化提示）：失败只记错误日志。
func writeAuditBestEffort(ctx context.Context, w PlatformAuditWriter, row *domain.PlatformOperationLog) {
	if err := w.Create(ctx, row); err != nil {
		slog.Error("platform audit write failed",
			"module", row.Module, "action", row.Action,
			"target_type", row.TargetType, "target_id", row.TargetID, "err", err)
	}
}
