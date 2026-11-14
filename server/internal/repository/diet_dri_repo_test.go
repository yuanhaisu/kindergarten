package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

// 用 993/994 这两个非业务年龄档做测试，避免与未来种子的 3/4/5/6 岁数据冲突。
const (
	testDietDriAgeBand      = 993
	testDietDriOtherAgeBand = 994
)

func cleanupDietDriData(t *testing.T, db *gorm.DB) {
	t.Helper()
	dietHardDelete(t, db, "DELETE FROM diet_dri_age_targets WHERE age_band IN ?",
		[]int{testDietDriAgeBand, testDietDriOtherAgeBand})
}

func TestDietDriRepo_ListByAgeOrdered(t *testing.T) {
	db := openDietRepoTestDB(t)
	repo := NewDietDriRepo(db)
	ctx := context.Background()

	cleanupDietDriData(t, db)
	t.Cleanup(func() { cleanupDietDriData(t, db) })

	rows := []domain.DriAgeTarget{
		{AgeBand: testDietDriAgeBand, NutrientKey: "test_energy", Label: "能量", Unit: "千卡", Value: 1300, Sort: 2},
		{AgeBand: testDietDriAgeBand, NutrientKey: "test_protein", Label: "蛋白质", Unit: "克", Value: 45, Sort: 1},
		{AgeBand: testDietDriOtherAgeBand, NutrientKey: "test_energy", Label: "能量", Unit: "千卡", Value: 1400, Sort: 1},
	}
	require.NoError(t, db.Create(&rows).Error)

	got, err := repo.ListByAge(ctx, testDietDriAgeBand)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "test_protein", got[0].NutrientKey, "按 sort 升序")
	assert.Equal(t, "test_energy", got[1].NutrientKey)
	for _, row := range got {
		assert.Equal(t, testDietDriAgeBand, row.AgeBand, "不得返回其他年龄档")
	}

	other, err := repo.ListByAge(ctx, testDietDriOtherAgeBand)
	require.NoError(t, err)
	require.Len(t, other, 1)
	assert.Equal(t, 1400.0, other[0].Value)

	empty, err := repo.ListByAge(ctx, 995)
	require.NoError(t, err)
	assert.Empty(t, empty)
}
