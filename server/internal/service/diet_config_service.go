package service

import (
	"context"
	"errors"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// ErrDietNotImplemented 标记膳食模块尚未实现。
// AQR-S1b 只落路由与权限骨架，真实行为在 AQR-S2（tasks 2.1/2.2/2.4）。
var ErrDietNotImplemented = errors.New("DIET_NOT_IMPLEMENTED")

// DietConfigService 供餐配置服务骨架。
//
// 本 slice 刻意不实现任何业务规则：默认值、餐次时间递增、相邻餐次占比上限、
// 市品重量固定、配置快照都属于 S2 的 2.1。handler 收到 ErrDietNotImplemented
// 时统一返回 501，避免前端把空配置误当成已实现。
type DietConfigService struct {
	// S1b 尚未调用 cfg；保留依赖是为了让 DietConfigRepo 进入 Wire 依赖图，
	// S2 的读写直接复用，不必再改构造签名。
	cfg *repository.DietConfigRepo
}

func NewDietConfigService(cfg *repository.DietConfigRepo) *DietConfigService {
	return &DietConfigService{cfg: cfg}
}

// GetConfig 返回园所供餐配置。
// TODO(AQR-S2): 缺省时返回默认值（日托 / 5 餐 / 市品重量 / 折算系数 0.95）。
func (s *DietConfigService) GetConfig(ctx context.Context, kindergartenID uint64) (*domain.MealConfig, error) {
	return nil, ErrDietNotImplemented
}

// SaveConfig 保存园所供餐配置。
// TODO(AQR-S2): 校验 6 个 meal_key 齐全、时间递增、相邻餐次占比上限后落库。
func (s *DietConfigService) SaveConfig(ctx context.Context, kindergartenID uint64) error {
	return ErrDietNotImplemented
}
