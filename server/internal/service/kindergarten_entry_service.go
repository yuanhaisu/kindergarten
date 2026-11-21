package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// 园所入口服务（#177）：负责 public_code 的生成与存量回填。
//
// 公开码会出现在家长端二维码 scene（kg_<public_code>）与公开品牌接口里，
// 因此必须随机、全局唯一、跨租户不可枚举。生成器只保证「候选未被占用」，
// 最终唯一性由 DB 的 uk_kindergartens_public_code 兜底：并发插入撞键时由
// 调用方（Onboard #171 / 回填）换码重试。

const (
	// PublicCodeLength 公开码长度。10 位 base32 = 50 位熵，既够抗枚举，
	// 又让 kg_<public_code> 远低于微信 scene 的 32 字符上限。
	PublicCodeLength = 10
	// publicCodeAlphabet Crockford base32 小写：去掉 i/l/o/u，避免人工抄写歧义。
	publicCodeAlphabet = "0123456789abcdefghjkmnpqrstvwxyz"
	// publicCodeMaxAttempts 单次生成允许的候选数量，超限说明随机源或数据异常。
	publicCodeMaxAttempts = 5
)

type KindergartenEntryService struct {
	kg     *repository.KindergartenRepo
	random io.Reader
}

func NewKindergartenEntryService(kg *repository.KindergartenRepo) *KindergartenEntryService {
	return &KindergartenEntryService{kg: kg, random: rand.Reader}
}

// NewKindergartenEntryServiceWithRandom 供测试注入确定性随机源；生产用 crypto/rand。
func NewKindergartenEntryServiceWithRandom(kg *repository.KindergartenRepo, random io.Reader) *KindergartenEntryService {
	return &KindergartenEntryService{kg: kg, random: random}
}

// GeneratePublicCode 生成一个当前未被任何园所占用（含软删）的公开码。
// 候选撞上已有码时换下一个候选；连续 publicCodeMaxAttempts 次都被占用则报错。
func (s *KindergartenEntryService) GeneratePublicCode(ctx context.Context) (string, error) {
	for attempt := 1; attempt <= publicCodeMaxAttempts; attempt++ {
		code, err := s.randomCode()
		if err != nil {
			return "", err
		}
		existing, err := s.kg.PublicCodeExists(ctx, code)
		if err != nil {
			return "", fmt.Errorf("检查园所公开码占用失败：%w", err)
		}
		if !existing {
			return code, nil
		}
	}
	return "", fmt.Errorf("生成园所公开码失败：连续 %d 次候选均已被占用", publicCodeMaxAttempts)
}

// BackfillPublicCodes 为尚未分配公开码的未删除园所补齐入口码，返回本次分配数量。
// 幂等：已有码的园所不会被改写；软删园所不参与回填。
func (s *KindergartenEntryService) BackfillPublicCodes(ctx context.Context) (int, error) {
	rows, err := s.kg.ListMissingPublicCode(ctx)
	if err != nil {
		return 0, fmt.Errorf("查询缺少公开码的园所失败：%w", err)
	}

	assigned := 0
	for _, kg := range rows {
		wrote, err := s.assignPublicCode(ctx, kg.ID)
		if err != nil {
			return assigned, fmt.Errorf("回填园所 %d 的公开码失败：%w", kg.ID, err)
		}
		if wrote {
			assigned++
		}
	}
	return assigned, nil
}

// FindByPublicCode 按公开码查未删除园所；空码返回 nil。
func (s *KindergartenEntryService) FindByPublicCode(ctx context.Context, code string) (*domain.Kindergarten, error) {
	return s.kg.FindByPublicCode(ctx, code)
}

// assignPublicCode 条件写入一个码；与并发回填/开通撞唯一键时换码重试。
func (s *KindergartenEntryService) assignPublicCode(ctx context.Context, kindergartenID uint64) (bool, error) {
	for attempt := 1; attempt <= publicCodeMaxAttempts; attempt++ {
		code, err := s.GeneratePublicCode(ctx)
		if err != nil {
			return false, err
		}
		wrote, err := s.kg.SetPublicCodeIfEmpty(ctx, kindergartenID, code)
		if errors.Is(err, repository.ErrDuplicatePublicCode) {
			continue
		}
		if err != nil {
			return false, err
		}
		return wrote, nil
	}
	return false, fmt.Errorf("连续 %d 次候选码写入冲突", publicCodeMaxAttempts)
}

func (s *KindergartenEntryService) randomCode() (string, error) {
	buf := make([]byte, PublicCodeLength)
	if _, err := io.ReadFull(s.random, buf); err != nil {
		return "", fmt.Errorf("读取公开码随机源失败：%w", err)
	}
	out := make([]byte, PublicCodeLength)
	for i, b := range buf {
		// 字母表长度 32，256 % 32 == 0，取模不引入偏置。
		out[i] = publicCodeAlphabet[int(b)%len(publicCodeAlphabet)]
	}
	return string(out), nil
}
