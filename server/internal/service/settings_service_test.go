package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

const (
	settingsTestKG   uint64 = 991201
	settingsTestUser uint64 = 991202
)

// 复用同包的 DB opener；settings 测试只需要 app setting 与 operation log 两张表。
func newSettingsFixture(t *testing.T) (*SettingsService, *gorm.DB) {
	t.Helper()
	db := openHealthTestDB(t)

	cleanup := func() {
		db.Unscoped().Where("kindergarten_id = ?", settingsTestKG).Delete(&domain.OperationLog{})
		db.Unscoped().Where("kindergarten_id = ?", settingsTestKG).Delete(&domain.KindergartenAppSetting{})
		db.Unscoped().Where("id = ?", settingsTestUser).Delete(&domain.User{})
		db.Unscoped().Where("id = ?", settingsTestKG).Delete(&domain.Kindergarten{})
	}
	t.Cleanup(cleanup)
	cleanup()

	require.NoError(t, db.Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: settingsTestKG},
		Name:      "体温策略测试园",
	}).Error)
	require.NoError(t, db.Create(&domain.User{
		BaseModel:    domain.BaseModel{ID: settingsTestUser},
		Nickname:     "测试管理员",
		Phone:        "13900009911",
		PasswordHash: "x",
	}).Error)

	svc := NewSettingsService(
		db,
		repository.NewKindergartenRepo(db),
		repository.NewAcademicYearRepo(db),
		repository.NewKindergartenAppSettingRepo(db),
		nil, nil, nil,
		repository.NewOperationLogRepo(db),
	)
	return svc, db
}

func TestParsePayload_HealthDefaults(t *testing.T) {
	def := defaultSettingsPayload().Health
	cases := []struct {
		name string
		raw  string
		want settingsHealth
	}{
		{"empty", "", def},
		{"empty object", "{}", def},
		{"legacy five blocks", `{"attendance":{"arrive_start_time":"07:30"}}`, def},
		{"explicit", `{"health":{"temperature_required":true,"fever_threshold":37.8}}`,
			settingsHealth{TemperatureRequired: true, FeverThreshold: 37.8}},
		{"invalid threshold keeps flag", `{"health":{"temperature_required":true,"fever_threshold":0}}`,
			settingsHealth{TemperatureRequired: true, FeverThreshold: DefaultFeverThreshold}},
		{"out of range", `{"health":{"temperature_required":true,"fever_threshold":99}}`,
			settingsHealth{TemperatureRequired: true, FeverThreshold: DefaultFeverThreshold}},
		{"malformed", "{", def},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, parsePayload(c.raw).Health)
		})
	}
}

func TestSettingsUpdate_HealthValidation(t *testing.T) {
	svc, db := newSettingsFixture(t)
	ctx := context.Background()

	_, err := svc.UpdateSettings(ctx, settingsTestKG, settingsTestUser, UpdateSettingsReq{
		Health: &settingsHealth{TemperatureRequired: true, FeverThreshold: 37.5},
	}, "", "")
	require.NoError(t, err)

	policy, err := svc.GetHealthPolicy(ctx, settingsTestKG)
	require.NoError(t, err)
	assert.True(t, policy.TemperatureRequired)
	assert.InDelta(t, 37.5, policy.FeverThreshold, 1e-9)

	settings, err := svc.GetSettings(ctx, settingsTestKG)
	require.NoError(t, err)
	assert.True(t, settings.Health.TemperatureRequired)
	assert.InDelta(t, 37.5, settings.Health.FeverThreshold, 1e-9)

	for _, bad := range []float64{34.9, 42.1} {
		_, err := svc.UpdateSettings(ctx, settingsTestKG, settingsTestUser, UpdateSettingsReq{
			Health: &settingsHealth{TemperatureRequired: true, FeverThreshold: bad},
		}, "", "")
		require.ErrorIs(t, err, ErrInvalidFeverThreshold)
		require.True(t, IsSettingsClientErr(err))

		// 越界保存不得覆盖原策略
		policy, err = svc.GetHealthPolicy(ctx, settingsTestKG)
		require.NoError(t, err)
		assert.InDelta(t, 37.5, policy.FeverThreshold, 1e-9)
	}

	var count int64
	require.NoError(t, db.Model(&domain.KindergartenAppSetting{}).
		Where("kindergarten_id = ?", settingsTestKG).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestGetHealthPolicy_NoRowDoesNotCreate(t *testing.T) {
	svc, db := newSettingsFixture(t)
	ctx := context.Background()

	var before int64
	require.NoError(t, db.Model(&domain.KindergartenAppSetting{}).
		Where("kindergarten_id = ?", settingsTestKG).Count(&before).Error)
	assert.Zero(t, before)

	policy, err := svc.GetHealthPolicy(ctx, settingsTestKG)
	require.NoError(t, err)
	assert.False(t, policy.TemperatureRequired)
	assert.InDelta(t, DefaultFeverThreshold, policy.FeverThreshold, 1e-9)

	var after int64
	require.NoError(t, db.Model(&domain.KindergartenAppSetting{}).
		Where("kindergarten_id = ?", settingsTestKG).Count(&after).Error)
	assert.Zero(t, after, "读取策略不得创建配置行")

	_, err = svc.UpdateSettings(ctx, settingsTestKG, settingsTestUser, UpdateSettingsReq{
		Health: &settingsHealth{TemperatureRequired: true, FeverThreshold: 37.4},
	}, "", "")
	require.NoError(t, err)

	policy, err = svc.GetHealthPolicy(ctx, settingsTestKG)
	require.NoError(t, err)
	assert.True(t, policy.TemperatureRequired)
	assert.InDelta(t, 37.4, policy.FeverThreshold, 1e-9)
}
