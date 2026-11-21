package service_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/internal/service"
	"kindergarten-server/internal/testutil/authharness"
)

// 园所入口字段（#177）的行为测试。
//
// 测试库由 authharness 派生并执行真实 goose 迁移，因此这里同时验证
// 00014 迁移加的列、默认值与 public_code 唯一索引确实生效。

func newEntryTestKindergarten(t *testing.T, h *authharness.Harness, name string) *domain.Kindergarten {
	t.Helper()
	kg := &domain.Kindergarten{Name: name, Status: domain.KindergartenStatusActive}
	require.NoError(t, h.DB.Create(kg).Error, "创建园所失败")
	t.Cleanup(func() {
		h.HardDelete(t, &domain.Kindergarten{}, "id = ?", kg.ID)
	})
	return kg
}

// 新园所默认走 shared 通道；v1 只有 shared，dedicated/group 只是保留取值。
func TestKindergartenEntryFieldsDefaultToSharedChannel(t *testing.T) {
	h := authharness.New(t)
	repo := repository.NewKindergartenRepo(h.DB)

	kg := newEntryTestKindergarten(t, h, "177-默认通道园")

	var got domain.Kindergarten
	require.NoError(t, h.DB.First(&got, kg.ID).Error)
	assert.Equal(t, domain.AppChannelShared, got.AppChannel, "园所默认通道必须是 shared")
	assert.Equal(t, "", got.PublicCode, "未生成前 public_code 为空")
	assert.Equal(t, "", got.BrandColor, "brand_color 默认空，由 #178 配置")

	dedicated := &domain.Kindergarten{
		Name:       "177-专属通道园",
		Status:     domain.KindergartenStatusActive,
		AppChannel: domain.AppChannelDedicated,
	}
	require.NoError(t, repo.Create(context.Background(), dedicated))
	t.Cleanup(func() { h.HardDelete(t, &domain.Kindergarten{}, "id = ?", dedicated.ID) })

	var dedicatedGot domain.Kindergarten
	require.NoError(t, h.DB.First(&dedicatedGot, dedicated.ID).Error)
	assert.Equal(t, domain.AppChannelDedicated, dedicatedGot.AppChannel)
}

// 公开码按未删除园所查找；软删园所不再返回，且其 public_code 不能被别的园复用。
func TestKindergartenFindByPublicCodeReturnsActiveOnly(t *testing.T) {
	h := authharness.New(t)
	repo := repository.NewKindergartenRepo(h.DB)
	ctx := context.Background()

	kg := newEntryTestKindergarten(t, h, "177-公开码园")
	require.NoError(t, h.DB.Model(&domain.Kindergarten{}).Where("id = ?", kg.ID).
		Update("public_code", "abcde12345").Error)

	found, err := repo.FindByPublicCode(ctx, "abcde12345")
	require.NoError(t, err)
	require.NotNil(t, found, "未删除园所必须能按 public_code 查到")
	assert.Equal(t, kg.ID, found.ID)

	_, err = repo.FindByPublicCode(ctx, "")
	assert.NoError(t, err, "空码不应触发查询错误")

	empty, err := repo.FindByPublicCode(ctx, "")
	require.NoError(t, err)
	assert.Nil(t, empty, "空码不能匹配任何园所")

	require.NoError(t, h.DB.Delete(&domain.Kindergarten{}, kg.ID).Error, "软删园所")
	gone, err := repo.FindByPublicCode(ctx, "abcde12345")
	require.NoError(t, err)
	assert.Nil(t, gone, "软删园所的 public_code 不得返回")

	other := newEntryTestKindergarten(t, h, "177-复用公开码园")
	err = h.DB.Model(&domain.Kindergarten{}).Where("id = ?", other.ID).
		Update("public_code", "abcde12345").Error
	require.Error(t, err, "软删园所的 public_code 仍占用唯一键，不能被复用")
}

// 公开码是 10 位 Crockford base32（去掉 i/l/o/u），且生成结果不能重复。
func TestGeneratePublicCodeFormat(t *testing.T) {
	h := authharness.New(t)
	svc := service.NewKindergartenEntryService(repository.NewKindergartenRepo(h.DB))

	const alphabet = "0123456789abcdefghjkmnpqrstvwxyz"
	seen := map[string]bool{}
	for i := 0; i < 8; i++ {
		code, err := svc.GeneratePublicCode(context.Background())
		require.NoError(t, err)
		require.Len(t, code, service.PublicCodeLength, "公开码必须是 10 位")
		for _, ch := range code {
			assert.Contains(t, alphabet, string(ch), "公开码字符必须来自 base32 字母表")
		}
		assert.False(t, seen[code], "生成器不能在未碰撞时重复输出")
		seen[code] = true
	}
}

// 候选码已被占用时必须换一个：碰撞重试是 public_code 全局唯一的前提。
// 注入确定性随机源，让第一次输出命中已占用码、第二次输出空闲码。
func TestGeneratePublicCodeRetriesOnCollision(t *testing.T) {
	h := authharness.New(t)
	repo := repository.NewKindergartenRepo(h.DB)
	ctx := context.Background()

	occupied := newEntryTestKindergarten(t, h, "177-碰撞占用园")
	require.NoError(t, h.DB.Model(&domain.Kindergarten{}).Where("id = ?", occupied.ID).
		Update("public_code", "0123456789").Error)

	// 索引 0-9 = "0123456789"（已占用），索引 10-19 = "abcdefghjk"（空闲）。
	random := bytes.NewReader([]byte{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
		10, 11, 12, 13, 14, 15, 16, 17, 18, 19,
	})
	svc := service.NewKindergartenEntryServiceWithRandom(repo, random)

	code, err := svc.GeneratePublicCode(ctx)
	require.NoError(t, err)
	assert.Equal(t, "abcdefghjk", code, "碰撞后必须采用下一个候选码")
}

// 软删园所的公开码仍占唯一键：生成候选时必须把它当作已占用，否则会拿到一个
// 「查不到但插不进去」的码。
func TestGeneratePublicCodeSkipsSoftDeletedCode(t *testing.T) {
	h := authharness.New(t)
	repo := repository.NewKindergartenRepo(h.DB)
	ctx := context.Background()

	deleted := newEntryTestKindergarten(t, h, "177-软删占用码园")
	require.NoError(t, h.DB.Model(&domain.Kindergarten{}).Where("id = ?", deleted.ID).
		Update("public_code", "0123456789").Error)
	require.NoError(t, h.DB.Delete(&domain.Kindergarten{}, deleted.ID).Error)

	random := bytes.NewReader([]byte{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
		10, 11, 12, 13, 14, 15, 16, 17, 18, 19,
	})
	svc := service.NewKindergartenEntryServiceWithRandom(repo, random)

	code, err := svc.GeneratePublicCode(ctx)
	require.NoError(t, err)
	assert.Equal(t, "abcdefghjk", code, "软删园所的码仍被占用，不能重新分配")
}

// 连续碰撞到上限必须显式报错，而不是返回一个已占用的码。
func TestGeneratePublicCodeFailsAfterMaxAttempts(t *testing.T) {
	h := authharness.New(t)
	repo := repository.NewKindergartenRepo(h.DB)
	ctx := context.Background()

	occupied := newEntryTestKindergarten(t, h, "177-碰撞耗尽园")
	require.NoError(t, h.DB.Model(&domain.Kindergarten{}).Where("id = ?", occupied.ID).
		Update("public_code", "0000000000").Error)

	svc := service.NewKindergartenEntryServiceWithRandom(repo, bytes.NewReader(make([]byte, 10*5)))

	_, err := svc.GeneratePublicCode(ctx)
	require.Error(t, err, "候选码持续冲突时必须报错")
	assert.Contains(t, err.Error(), "5", "错误信息要说明已经尝试的次数")
}

// 回填只补未分配码的未删除园所；已有码不动，软删园所不补，重复执行幂等。
func TestBackfillPublicCodesFillsMissingOnly(t *testing.T) {
	h := authharness.New(t)
	repo := repository.NewKindergartenRepo(h.DB)
	svc := service.NewKindergartenEntryService(repo)
	ctx := context.Background()

	preCoded := newEntryTestKindergarten(t, h, "177-已有码不回填园")
	require.NoError(t, h.DB.Model(&domain.Kindergarten{}).Where("id = ?", preCoded.ID).
		Update("public_code", "0123456789").Error)

	freshA := newEntryTestKindergarten(t, h, "177-待回填园A")
	freshB := newEntryTestKindergarten(t, h, "177-待回填园B")

	deleted := newEntryTestKindergarten(t, h, "177-软删不回填园")
	require.NoError(t, h.DB.Delete(&domain.Kindergarten{}, deleted.ID).Error)

	assigned, err := svc.BackfillPublicCodes(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, assigned, 2, "两个未分配码的园所都必须回填")

	readCode := func(id uint64, unscoped bool) string {
		q := h.DB
		if unscoped {
			q = q.Unscoped()
		}
		var kg domain.Kindergarten
		require.NoError(t, q.First(&kg, id).Error)
		return kg.PublicCode
	}

	assert.Equal(t, "0123456789", readCode(preCoded.ID, false), "已有码不得被覆盖")
	codeA := readCode(freshA.ID, false)
	codeB := readCode(freshB.ID, false)
	require.Len(t, codeA, service.PublicCodeLength)
	require.Len(t, codeB, service.PublicCodeLength)
	assert.NotEqual(t, codeA, codeB, "不同园所的公开码必须不同")
	assert.Equal(t, "", readCode(deleted.ID, true), "软删园所不参与回填")

	_, err = svc.BackfillPublicCodes(ctx)
	require.NoError(t, err)
	assert.Equal(t, codeA, readCode(freshA.ID, false), "重复回填不得改码")
	assert.Equal(t, codeB, readCode(freshB.ID, false), "重复回填不得改码")
}

// 回填写入必须条件化：已被并发填过的行不能被覆盖。
func TestSetPublicCodeIfEmptyDoesNotOverwrite(t *testing.T) {
	h := authharness.New(t)
	repo := repository.NewKindergartenRepo(h.DB)
	ctx := context.Background()

	kg := newEntryTestKindergarten(t, h, "177-条件写入园")
	ok, err := repo.SetPublicCodeIfEmpty(ctx, kg.ID, "0123456789")
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = repo.SetPublicCodeIfEmpty(ctx, kg.ID, "9876543210")
	require.NoError(t, err)
	assert.False(t, ok, "已有码时不得再写入")

	var got domain.Kindergarten
	require.NoError(t, h.DB.First(&got, kg.ID).Error)
	assert.Equal(t, "0123456789", got.PublicCode)
}

// 撞唯一键要能被上层识别（回填换码重试依赖它），不能让 SQL 错误裸奔。
func TestSetPublicCodeIfEmptyReportsDuplicate(t *testing.T) {
	h := authharness.New(t)
	repo := repository.NewKindergartenRepo(h.DB)
	ctx := context.Background()

	first := newEntryTestKindergarten(t, h, "177-占码园")
	second := newEntryTestKindergarten(t, h, "177-撞码园")
	_, err := repo.SetPublicCodeIfEmpty(ctx, first.ID, "abcdefghjk")
	require.NoError(t, err)

	_, err = repo.SetPublicCodeIfEmpty(ctx, second.ID, "abcdefghjk")
	require.ErrorIs(t, err, repository.ErrDuplicatePublicCode)
}

// #170/#176 通过服务查园所：软删不返回，空码直接返回 nil 而不查库。
func TestEntryServiceFindByPublicCode(t *testing.T) {
	h := authharness.New(t)
	svc := service.NewKindergartenEntryService(repository.NewKindergartenRepo(h.DB))
	ctx := context.Background()

	kg := newEntryTestKindergarten(t, h, "177-服务查码园")
	require.NoError(t, h.DB.Model(&domain.Kindergarten{}).Where("id = ?", kg.ID).
		Update("public_code", strings.Repeat("a", 10)).Error)

	found, err := svc.FindByPublicCode(ctx, strings.Repeat("a", 10))
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, kg.ID, found.ID)

	empty, err := svc.FindByPublicCode(ctx, "   ")
	require.NoError(t, err)
	assert.Nil(t, empty)

	require.NoError(t, h.DB.Delete(&domain.Kindergarten{}, kg.ID).Error)
	gone, err := svc.FindByPublicCode(ctx, strings.Repeat("a", 10))
	require.NoError(t, err)
	assert.Nil(t, gone)
}
