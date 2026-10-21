// Package face 抽象人脸识别服务（底库注册 + 1:N 搜索）。
//
// 供应商现状（2026-10）：阿里云 facebody 已退市（公告 118619），采用腾讯云 IAI。
// Provider 抽象的价值在本变更规划期已被验证——换供应商只改实现，不动调用方。
//
// 密钥边界（设计硬规则）：所有凭证与第三方调用只存在于后端。
// 客户端（小程序/admin）不得持有 SecretId/SecretKey，也不得直接调用脸服务。
package face

import (
	"context"
	"fmt"

	"kindergarten-server/internal/config"
)

// Match 一条识别命中：某学生与目标照片的相似度。
// Score 为供应商原始分（腾讯云口径 0-100，越高越相似）。
type Match struct {
	StudentID uint64
	Score     float64
}

// SearchResult 一次识别的结果。
// FaceNum 为照片中**检测到的人脸数**（受供应商 MaxFaceNum 上限约束，如腾讯云为 10）——
// 它是免费的「照片分型」信号：1=单人照 / 少量=小组照 / 触顶=大合影（design D11）。
type SearchResult struct {
	Matches []Match
	FaceNum int
}

// Provider 人脸服务抽象。
// 实现方负责：把园所映射为云侧隔离单元（如腾讯云的 Group）、把学生映射为人员条目，
// 并在识别时只在该园所范围内搜索。
type Provider interface {
	// EnsurePerson 确保学生在云端底库有可检索条目（幂等）。
	// images 为底库照片（字节内容）；返回云端人员引用（PersonId）。
	EnsurePerson(ctx context.Context, kgID, studentID uint64, images [][]byte) (personRef string, err error)

	// RemovePerson 从云端底库移除学生（转出/注销时调用；不存在时视为成功）。
	RemovePerson(ctx context.Context, kgID, studentID uint64) error

	// Search 对单张照片做 1:N 搜索，仅在该园所底库范围内，返回候选与检测到的人脸数。
	Search(ctx context.Context, kgID uint64, image []byte) (SearchResult, error)

	// Name 返回供应商标识（"tencent" / "mock"）。
	Name() string
}

// New 按配置构造 Provider。provider 为空或 "mock" 时返回 mock 实现（无凭证环境可用）。
func New(cfg config.FaceConfig) (Provider, error) {
	switch cfg.Provider {
	case "tencent":
		return NewTencentProvider(cfg)
	case "", "mock":
		return NewMockProvider(), nil
	default:
		return nil, fmt.Errorf("unknown face provider: %q", cfg.Provider)
	}
}

// groupID 园所 → 云侧隔离单元标识。
func groupID(kgID uint64) string {
	return fmt.Sprintf("kg_%d", kgID)
}

// personID 学生 → 云侧人员标识（跨园唯一，便于迁移与排查）。
func personID(kgID, studentID uint64) string {
	return fmt.Sprintf("kg%d_stu%d", kgID, studentID)
}
