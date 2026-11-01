package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// openspec class-album-classwide 1.2：管理端 settings 契约的 album.classwide_enabled。
// 打真实 MySQL（同 settings_service_test.go 的开发库），9912xx 夹具，t.Cleanup 硬删。

const (
	settingsCAWKg   uint64 = 991231
	settingsCAWUser uint64 = 991232
)

func newClasswideSettingsFixture(t *testing.T) (*SettingsService, *gorm.DB) {
	t.Helper()
	db := openHealthTestDB(t)
	// 本切片不重启 8080，测试自己补新列；已存在时 AutoMigrate 是 no-op。
	require.NoError(t, db.AutoMigrate(&domain.KindergartenAppSetting{}))

	cleanup := func() {
		db.Unscoped().Where("kindergarten_id = ?", settingsCAWKg).Delete(&domain.OperationLog{})
		db.Unscoped().Where("kindergarten_id = ?", settingsCAWKg).Delete(&domain.KindergartenAppSetting{})
		db.Unscoped().Where("id = ?", settingsCAWUser).Delete(&domain.User{})
		db.Unscoped().Where("id = ?", settingsCAWKg).Delete(&domain.Kindergarten{})
	}
	t.Cleanup(cleanup)
	cleanup()

	require.NoError(t, db.Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: settingsCAWKg},
		Name:      "全班视图设置测试园",
	}).Error)
	require.NoError(t, db.Create(&domain.User{
		BaseModel:    domain.BaseModel{ID: settingsCAWUser},
		Nickname:     "全班视图测试管理员",
		Phone:        "13900009912",
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

// TC-NEW-CAW-0001：从未配置过的园所读到 album.classwide_enabled=false，
// 且首次读取建出的默认行新列也是 false。
func TestSettingsClasswideDefaultsFalse(t *testing.T) {
	svc, db := newClasswideSettingsFixture(t)
	ctx := context.Background()

	var before int64
	require.NoError(t, db.Unscoped().Model(&domain.KindergartenAppSetting{}).
		Where("kindergarten_id = ?", settingsCAWKg).Count(&before).Error)
	require.Zero(t, before, "夹具必须从「没有设置行」开始")

	got, err := svc.GetSettings(ctx, settingsCAWKg)
	require.NoError(t, err)
	assert.False(t, got.Album.ClasswideEnabled, "未配置的园所默认关闭")

	var row domain.KindergartenAppSetting
	require.NoError(t, db.Where("kindergarten_id = ?", settingsCAWKg).First(&row).Error)
	assert.False(t, row.ClassAlbumClasswideEnabled, "默认行列值必须是 false")
}

// TC-NEW-CAW-0002：管理员开启后再关闭，两次都落库并写审计；
// 只改其它设置块时开关保持不变（部分更新）。
func TestSettingsClasswideUpdateAndAudit(t *testing.T) {
	svc, db := newClasswideSettingsFixture(t)
	ctx := context.Background()

	update := func(req UpdateSettingsReq) {
		t.Helper()
		_, err := svc.UpdateSettings(ctx, settingsCAWKg, settingsCAWUser, req, "127.0.0.1", "test-agent")
		require.NoError(t, err)
	}
	read := func() bool {
		t.Helper()
		got, err := svc.GetSettings(ctx, settingsCAWKg)
		require.NoError(t, err)
		return got.Album.ClasswideEnabled
	}

	update(UpdateSettingsReq{Album: &settingsAlbum{ClasswideEnabled: true}})
	assert.True(t, read())

	update(UpdateSettingsReq{Notice: &settingsNotice{NeedAudit: false, AutoRemindHours: 12}})
	assert.True(t, read(), "未传 album 必须保持原值")
	got, err := svc.GetSettings(ctx, settingsCAWKg)
	require.NoError(t, err)
	assert.False(t, got.Notice.NeedAudit, "同一请求里的其它块仍正常更新")

	update(UpdateSettingsReq{Album: &settingsAlbum{ClasswideEnabled: false}})
	assert.False(t, read())

	// 审计：三次 update_settings 都落库，detail 中的开关值与当时状态一致。
	var logs []domain.OperationLog
	require.NoError(t, db.Where("kindergarten_id = ? AND module = ? AND action = ?",
		settingsCAWKg, "settings", "update_settings").Order("id ASC").Find(&logs).Error)
	require.Len(t, logs, 3)

	flags := make([]bool, 0, len(logs))
	for _, l := range logs {
		var detail struct {
			Album struct {
				ClasswideEnabled bool `json:"classwide_enabled"`
			} `json:"album"`
		}
		require.NoError(t, json.Unmarshal([]byte(l.Detail), &detail), "审计 detail 必须是 JSON")
		flags = append(flags, detail.Album.ClasswideEnabled)
	}
	assert.Equal(t, []bool{true, true, false}, flags)
}
