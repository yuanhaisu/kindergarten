package service

import (
	"context"
	"fmt"
	"time"

	"kindergarten-server/internal/repository"
	"kindergarten-server/pkg/log"
)

// 园所公开品牌服务（#170）。
//
// 未登录的家长端启动页要按 public_code 显示园所名称/logo/主题色。数据是公开的，
// 但必须严格收窄字段：地址、电话、园长、证照、容量、简介一律不下发。
// logo 存在私有 OSS 时由注入的 LogoSigner 签成短期 URL——不复用响应层的全局
// 媒体签名（TTL 固定 1 小时，且 logo_url 不在其白名单里）。

// PublicBrandLogoTTL 公开品牌 logo 的签名有效期。启动页一次拉取即可，
// 短时效避免 URL 被转发后长期可用。
const PublicBrandLogoTTL = 10 * time.Minute

// LogoSigner 把存储里的 logo 引用签成短期可访问 URL。
// 非 OSS 引用（如外链头像）应原样返回。
type LogoSigner interface {
	SignLogo(ref string) (string, error)
}

// LogoSignerFunc 让装配层与测试用普通函数实现 LogoSigner。
type LogoSignerFunc func(ref string) (string, error)

func (f LogoSignerFunc) SignLogo(ref string) (string, error) {
	return f(ref)
}

// PublicBrand 是公开品牌接口的全部字段，新增字段前先确认可公开。
type PublicBrand struct {
	PublicCode string `json:"public_code"`
	Name       string `json:"name"`
	LogoURL    string `json:"logo_url"`
	BrandColor string `json:"brand_color"`
}

type PublicBrandService struct {
	kg    *repository.KindergartenRepo
	logos LogoSigner
}

func NewPublicBrandService(kg *repository.KindergartenRepo, logos LogoSigner) *PublicBrandService {
	return &PublicBrandService{kg: kg, logos: logos}
}

// GetByPublicCode 按公开码取园所品牌；园所不存在或已软删时返回 (nil, nil)，
// 由 handler 映射成 404。
func (s *PublicBrandService) GetByPublicCode(ctx context.Context, code string) (*PublicBrand, error) {
	kg, err := s.kg.FindByPublicCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("按公开码查询园所失败：%w", err)
	}
	if kg == nil {
		return nil, nil
	}
	return &PublicBrand{
		PublicCode: kg.PublicCode,
		Name:       kg.Name,
		LogoURL:    s.signLogo(ctx, kg.LogoURL),
		BrandColor: NormalizeBrandColor(kg.BrandColor),
	}, nil
}

func (s *PublicBrandService) signLogo(ctx context.Context, ref string) string {
	if ref == "" {
		return ""
	}
	if s.logos == nil {
		log.L(ctx).Warn("public brand logo signer not configured, returning raw reference")
		return ref
	}
	signed, err := s.logos.SignLogo(ref)
	if err != nil {
		// 本地未配置 OSS 时对象引用本身就是可读路径；生产签名失败说明配置异常，
		// 记录后回退原始引用，至少不让公开品牌接口整体 500。
		log.L(ctx).Warn("sign public brand logo failed", "err", err)
		return ref
	}
	return signed
}
