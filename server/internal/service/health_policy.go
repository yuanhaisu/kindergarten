package service

import "context"

const (
	// DefaultFeverThreshold 未配置园所策略时使用的发烧阈值。
	DefaultFeverThreshold = 37.3
	MinFeverThreshold     = 35.0
	MaxFeverThreshold     = 42.0
)

// HealthPolicy 园所级晨检体温策略。
type HealthPolicy struct {
	TemperatureRequired bool    `json:"temperature_required"`
	FeverThreshold      float64 `json:"fever_threshold"`
}

// HealthPolicyProvider 让晨检服务依赖策略读取能力，而不是整个 SettingsService。
type HealthPolicyProvider interface {
	GetHealthPolicy(ctx context.Context, kindergartenID uint64) (HealthPolicy, error)
}
