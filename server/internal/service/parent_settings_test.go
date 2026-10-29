package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

// 家长设置持久化。DB 用例复用 openRQTestDB（连本机 MySQL，无配置则 Skip）。
// 数据段 992000000+，与照片点名（991…）、识别队列（990…）错开。
//
// 修复前 GetSettings 硬编码全 true、UpdateSettings 空实现，默认值与写后重读两条
// 用例即失败证据（.claude/runs/B08/test-before-fix.log）。
const parentSettingsTestBase = 992000000

func openParentSettingsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := openRQTestDB(t)
	require.NoError(t, db.AutoMigrate(&domain.ParentSetting{}))
	return db
}

func parentSettingsCtx(userID, kgID uint64) *ParentPortalContext {
	return &ParentPortalContext{UserID: userID, KindergartenID: kgID}
}

// newParentSettingsFixture 返回只带 db 的服务实例（设置读写不依赖其他仓储），
// 并注册硬删清理：软删行仍占唯一索引，会令后续 upsert 撞键。
func newParentSettingsFixture(t *testing.T) (*ParentPortalService, *gorm.DB) {
	t.Helper()
	db := openParentSettingsTestDB(t)
	t.Cleanup(func() { cleanupParentSettings(t, db) })
	// GetSettings / UpdateSettings 只用 s.db，其余仓储留空即可。
	return NewParentPortalService(db, nil, nil, nil, nil, nil, nil, nil), db
}

func cleanupParentSettings(t *testing.T, db *gorm.DB) {
	t.Helper()
	users := []uint64{
		parentSettingsTestBase + 1, parentSettingsTestBase + 2, parentSettingsTestBase + 3,
		parentSettingsTestBase + 4, parentSettingsTestBase + 5, parentSettingsTestBase + 6,
		parentSettingsTestBase + 7,
	}
	res := db.Unscoped().Where("user_id IN ?", users).Delete(&domain.ParentSetting{})
	require.NoError(t, res.Error)
	var left int64
	require.NoError(t, db.Unscoped().Model(&domain.ParentSetting{}).
		Where("user_id IN ?", users).Count(&left).Error)
	require.Zero(t, left, "测试数据必须已清理")
	t.Logf("cleanup parent_settings user_ids=%v deleted=%d remaining=%d", users, res.RowsAffected, left)
}

func parentSettingBool(t *testing.T, res map[string]any, group, key string) bool {
	t.Helper()
	g, ok := res[group].(map[string]any)
	require.True(t, ok, "%s 应为对象", group)
	v, ok := g[key]
	require.True(t, ok, "%s.%s 缺失", group, key)
	b, ok := v.(bool)
	require.True(t, ok, "%s.%s 应为布尔，实际 %T", group, key, v)
	return b
}

func TestParentSettings_DefaultBeforeAnyWrite(t *testing.T) {
	svc, db := newParentSettingsFixture(t)
	ctx := context.Background()
	pc := parentSettingsCtx(parentSettingsTestBase+1, parentSettingsTestBase+1)

	res, err := svc.GetSettings(ctx, pc)
	require.NoError(t, err)

	assert.False(t, parentSettingBool(t, res, "notification", "task_push"), "任务提醒默认关（原型 parent/25-settings）")
	assert.True(t, parentSettingBool(t, res, "notification", "notice_push"))
	assert.True(t, parentSettingBool(t, res, "notification", "leave_push"))
	assert.True(t, parentSettingBool(t, res, "notification", "pickup_push"))
	assert.True(t, parentSettingBool(t, res, "notification", "growth_push"))
	assert.True(t, parentSettingBool(t, res, "notification", "announcement_push"))
	assert.False(t, parentSettingBool(t, res, "privacy", "allow_growth_share"), "允许展示成长记录默认关（原型）")
	assert.True(t, parentSettingBool(t, res, "privacy", "show_phone_to_teacher"))

	var rows int64
	require.NoError(t, db.Model(&domain.ParentSetting{}).Where("user_id = ?", pc.UserID).Count(&rows).Error)
	assert.Zero(t, rows, "读取默认值不得建行")

	// 只回传 null 分组是空操作，同样不得建行
	require.NoError(t, svc.UpdateSettings(ctx, pc, map[string]any{"privacy": nil}))
	require.NoError(t, db.Model(&domain.ParentSetting{}).Where("user_id = ?", pc.UserID).Count(&rows).Error)
	assert.Zero(t, rows, "空操作不得建行")
}

func TestParentSettings_UpdatePersists(t *testing.T) {
	svc, db := newParentSettingsFixture(t)
	ctx := context.Background()
	pc := parentSettingsCtx(parentSettingsTestBase+2, parentSettingsTestBase+1)

	err := svc.UpdateSettings(ctx, pc, map[string]any{
		"notification": map[string]any{
			"notice_push":       false,
			"leave_push":        true,
			"pickup_push":       true,
			"growth_push":       false,
			"task_push":         true,
			"announcement_push": false,
		},
		"privacy": map[string]any{
			"show_phone_to_teacher": false,
			"allow_growth_share":    true,
		},
	})
	require.NoError(t, err)

	// 换一个服务实例读取，排除内存态
	reloaded := NewParentPortalService(db, nil, nil, nil, nil, nil, nil, nil)
	res, err := reloaded.GetSettings(ctx, pc)
	require.NoError(t, err)
	assert.False(t, parentSettingBool(t, res, "notification", "notice_push"))
	assert.True(t, parentSettingBool(t, res, "notification", "leave_push"))
	assert.False(t, parentSettingBool(t, res, "notification", "growth_push"))
	assert.False(t, parentSettingBool(t, res, "notification", "announcement_push"))
	assert.True(t, parentSettingBool(t, res, "notification", "task_push"))
	assert.False(t, parentSettingBool(t, res, "privacy", "show_phone_to_teacher"))
	assert.True(t, parentSettingBool(t, res, "privacy", "allow_growth_share"))

	var rows int64
	require.NoError(t, db.Model(&domain.ParentSetting{}).Where("user_id = ?", pc.UserID).Count(&rows).Error)
	assert.EqualValues(t, 1, rows)
}

func TestParentSettings_PartialUpdateKeepsOtherKeys(t *testing.T) {
	svc, _ := newParentSettingsFixture(t)
	ctx := context.Background()
	pc := parentSettingsCtx(parentSettingsTestBase+3, parentSettingsTestBase+1)

	require.NoError(t, svc.UpdateSettings(ctx, pc, map[string]any{
		"notification": map[string]any{"task_push": true},
		"privacy":      map[string]any{"allow_growth_share": true},
	}))

	// 只改两项，其余必须保持上一次写入的值
	require.NoError(t, svc.UpdateSettings(ctx, pc, map[string]any{
		"notification": map[string]any{"growth_push": false},
		"privacy":      map[string]any{"show_phone_to_teacher": false},
	}))

	res, err := svc.GetSettings(ctx, pc)
	require.NoError(t, err)
	assert.True(t, parentSettingBool(t, res, "notification", "task_push"), "部分更新不得把 task_push 重置为默认")
	assert.False(t, parentSettingBool(t, res, "notification", "growth_push"))
	assert.True(t, parentSettingBool(t, res, "notification", "notice_push"))
	assert.True(t, parentSettingBool(t, res, "privacy", "allow_growth_share"), "部分更新不得重置 privacy")
	assert.False(t, parentSettingBool(t, res, "privacy", "show_phone_to_teacher"))
}

func TestParentSettings_UserAndKindergartenIsolation(t *testing.T) {
	svc, _ := newParentSettingsFixture(t)
	ctx := context.Background()
	a := parentSettingsCtx(parentSettingsTestBase+4, parentSettingsTestBase+1)
	b := parentSettingsCtx(parentSettingsTestBase+5, parentSettingsTestBase+1)
	aOtherKg := parentSettingsCtx(parentSettingsTestBase+4, parentSettingsTestBase+2)

	require.NoError(t, svc.UpdateSettings(ctx, a, map[string]any{
		"notification": map[string]any{"notice_push": false, "task_push": true},
	}))

	bRes, err := svc.GetSettings(ctx, b)
	require.NoError(t, err)
	assert.True(t, parentSettingBool(t, bRes, "notification", "notice_push"), "A 的设置不得影响 B")
	assert.False(t, parentSettingBool(t, bRes, "notification", "task_push"))

	kgRes, err := svc.GetSettings(ctx, aOtherKg)
	require.NoError(t, err)
	assert.True(t, parentSettingBool(t, kgRes, "notification", "notice_push"), "同一用户在其他园所应独立")
}

func TestParentSettings_IdempotentRepeatWrite(t *testing.T) {
	svc, db := newParentSettingsFixture(t)
	ctx := context.Background()
	pc := parentSettingsCtx(parentSettingsTestBase+6, parentSettingsTestBase+1)
	patch := map[string]any{"notification": map[string]any{"pickup_push": false}}

	require.NoError(t, svc.UpdateSettings(ctx, pc, patch))
	require.NoError(t, svc.UpdateSettings(ctx, pc, patch), "重复写入同值不得报错")

	var rows int64
	require.NoError(t, db.Model(&domain.ParentSetting{}).Where("user_id = ?", pc.UserID).Count(&rows).Error)
	assert.EqualValues(t, 1, rows, "幂等写入不得产生多行")

	res, err := svc.GetSettings(ctx, pc)
	require.NoError(t, err)
	assert.False(t, parentSettingBool(t, res, "notification", "pickup_push"))
}

func TestParentSettings_RejectsInvalidTypesAndUnknownKeys(t *testing.T) {
	svc, db := newParentSettingsFixture(t)
	ctx := context.Background()
	pc := parentSettingsCtx(parentSettingsTestBase+7, parentSettingsTestBase+1)

	cases := []struct {
		name string
		body map[string]any
	}{
		{"字符串当布尔", map[string]any{"notification": map[string]any{"task_push": "yes"}}},
		{"数字当布尔", map[string]any{"privacy": map[string]any{"allow_growth_share": 1}}},
		{"空值当布尔", map[string]any{"notification": map[string]any{"task_push": nil}}},
		{"未知设置项", map[string]any{"notification": map[string]any{"task_puhs": true}}},
		{"未知分组", map[string]any{"other": map[string]any{"x": true}}},
		{"分组不是对象", map[string]any{"notification": []any{true}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := svc.UpdateSettings(ctx, pc, c.body)
			require.Error(t, err)
			assert.True(t, errors.Is(err, ErrParentInvalidParams), "应返回参数错误，实际 %v", err)
		})
	}

	// 非法写入不得落库，读回仍是默认值
	res, err := svc.GetSettings(ctx, pc)
	require.NoError(t, err)
	assert.False(t, parentSettingBool(t, res, "notification", "task_push"))
	var rows int64
	require.NoError(t, db.Model(&domain.ParentSetting{}).Where("user_id = ?", pc.UserID).Count(&rows).Error)
	assert.Zero(t, rows)
}

func TestParentSettings_RejectsMissingContext(t *testing.T) {
	svc, _ := newParentSettingsFixture(t)
	ctx := context.Background()

	_, err := svc.GetSettings(ctx, nil)
	assert.True(t, errors.Is(err, ErrParentInvalidParams))

	err = svc.UpdateSettings(ctx, &ParentPortalContext{UserID: 0, KindergartenID: 0},
		map[string]any{"notification": map[string]any{"task_push": true}})
	assert.True(t, errors.Is(err, ErrParentInvalidParams))
}
