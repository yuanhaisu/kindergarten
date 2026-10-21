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

// 管理端权限码解析三态（wire-admin-permissions，TC-NEW-PERM-0001/0002/0003）。
// 真 DB 跑，风格对齐 teacher_pickup_create_test.go；测试数据用高位 ID 隔离。

const (
	testAuthzKG      uint64 = 990002
	testAuthzUser    uint64 = 991000001 // 有角色关联
	testAuthzSuperID uint64 = 991000002 // 无关联 + super_admin 标记
	testAuthzNoneID  uint64 = 991000003 // 无关联且非超管
	testAuthzRoleID  uint64 = 991000010
	testAuthzPermID  uint64 = 991000020
)

func openAuthzTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// 复用接送 fixture 的连接（同样读 configs/app.ini；无配置则跳过）
	return openPickupTestDB(t)
}

func cleanupAuthzData(t *testing.T, db *gorm.DB) {
	t.Helper()
	ctx := context.Background()
	del := func(sql string, args ...any) {
		if err := db.WithContext(ctx).Exec(sql, args...).Error; err != nil {
			t.Logf("cleanup: %v", err)
		}
	}
	del("DELETE FROM user_admin_roles WHERE user_id IN (?,?,?)", testAuthzUser, testAuthzSuperID, testAuthzNoneID)
	del("DELETE FROM admin_role_permissions WHERE role_id = ?", testAuthzRoleID)
	del("DELETE FROM admin_roles WHERE id = ?", testAuthzRoleID)
	del("DELETE FROM admin_permissions WHERE id = ?", testAuthzPermID)
	del("DELETE FROM user_roles WHERE user_id IN (?,?,?)", testAuthzUser, testAuthzSuperID, testAuthzNoneID)
	del("DELETE FROM users WHERE id IN (?,?,?)", testAuthzUser, testAuthzSuperID, testAuthzNoneID)
}

func newAuthzTestService(t *testing.T) (*AdminAuthzService, *gorm.DB) {
	t.Helper()
	db := openAuthzTestDB(t)
	cleanupAuthzData(t, db)
	ctx := context.Background()

	// 园所是 admin_roles / user_admin_roles 的外键目标，幂等创建
	var kgCount int64
	db.WithContext(ctx).Model(&domain.Kindergarten{}).Where("id = ?", testAuthzKG).Count(&kgCount)
	if kgCount == 0 {
		require.NoError(t, db.WithContext(ctx).Create(&domain.Kindergarten{
			BaseModel: domain.BaseModel{ID: testAuthzKG},
			Name:      "测试园所-权限解析",
		}).Error)
	}

	// 三个用户
	for _, u := range []struct {
		id    uint64
		phone string
	}{
		{testAuthzUser, "13910000001"},
		{testAuthzSuperID, "13910000002"},
		{testAuthzNoneID, "13910000003"},
	} {
		require.NoError(t, db.WithContext(ctx).Create(&domain.User{
			BaseModel: domain.BaseModel{ID: u.id},
			Phone:     u.phone,
			Nickname:  "权限解析测试",
		}).Error)
	}

	// 用户 1：有 UserAdminRole 关联 → 角色挂一个专用权限码
	require.NoError(t, db.WithContext(ctx).Create(&domain.AdminRole{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: testAuthzRoleID}},
		KindergartenID:  testAuthzKG,
		Name:            "权限解析测试角色",
	}).Error)
	require.NoError(t, db.WithContext(ctx).Create(&domain.AdminPermission{
		BaseModel: domain.BaseModel{ID: testAuthzPermID},
		Code:      "test-authz:view",
		Name:      "测试权限码",
		Module:    "test",
	}).Error)
	require.NoError(t, db.WithContext(ctx).Create(&domain.AdminRolePermission{
		RoleID:       testAuthzRoleID,
		PermissionID: testAuthzPermID,
	}).Error)
	require.NoError(t, db.WithContext(ctx).Create(&domain.UserAdminRole{
		UserID:         testAuthzUser,
		AdminRoleID:    testAuthzRoleID,
		KindergartenID: testAuthzKG,
	}).Error)

	// 用户 2：无关联 + admin 角色行标记 super_admin
	kgID := testAuthzKG
	require.NoError(t, db.WithContext(ctx).Create(&domain.UserRole{
		UserID:         testAuthzSuperID,
		Role:           "admin",
		KindergartenID: &kgID,
		AdminRole:      "super_admin",
	}).Error)

	// 用户 3：无关联且非超管
	require.NoError(t, db.WithContext(ctx).Create(&domain.UserRole{
		UserID:         testAuthzNoneID,
		Role:           "admin",
		KindergartenID: &kgID,
		AdminRole:      "admin",
	}).Error)

	svc := NewAdminAuthzService(repository.NewAdminRepo(db), repository.NewUserRoleRepo(db))
	t.Cleanup(func() { cleanupAuthzData(t, db) })
	return svc, db
}

// TC-NEW-PERM-0001 角色关联优先：码集取自角色所配权限。
func TestResolveAdminCodes_RoleLink(t *testing.T) {
	svc, _ := newAuthzTestService(t)
	codes, wildcard, err := svc.ResolveAdminCodes(context.Background(), testAuthzKG, testAuthzUser)
	require.NoError(t, err)
	assert.False(t, wildcard, "有角色关联时不应通配")
	assert.Equal(t, []string{"test-authz:view"}, codes)
}

// TC-NEW-PERM-0002 无关联 + super_admin 标记 → 通配。
func TestResolveAdminCodes_SuperAdminFallback(t *testing.T) {
	svc, _ := newAuthzTestService(t)
	codes, wildcard, err := svc.ResolveAdminCodes(context.Background(), testAuthzKG, testAuthzSuperID)
	require.NoError(t, err)
	assert.True(t, wildcard, "super_admin 标记应解析为通配")
	assert.Equal(t, []string{"*"}, codes)
}

// TC-NEW-PERM-0003 无关联且非超管 → 空集。
func TestResolveAdminCodes_EmptyForOutsider(t *testing.T) {
	svc, _ := newAuthzTestService(t)
	codes, wildcard, err := svc.ResolveAdminCodes(context.Background(), testAuthzKG, testAuthzNoneID)
	require.NoError(t, err)
	assert.False(t, wildcard)
	assert.Empty(t, codes)
}
