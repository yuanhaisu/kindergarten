package adminprovision_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"kindergarten-server/cmd/internal/adminprovision"
	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
)

// issue #56：setadmin 改为显式租户管理员供给。
// 沿用 service 层 MySQL 集成测试的缝隙：读 configs/app.ini，无配置则跳过。
// 测试数据用高位 ID / 专用手机号隔离，每例硬删自建行（软删行会占唯一索引）。

const (
	testKGID        uint64 = 990056
	testKG2ID       uint64 = 990057
	testKGDeletedID uint64 = 990058
	testPhone       string = "19900000056"
	testPasswd      string = "Test-Admin-Pass-56"
)

var testDB *gorm.DB

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	if testDB != nil {
		return testDB
	}
	cfgPath := filepath.Join("..", "..", "..", "configs", "app.ini")
	if _, err := os.Stat(cfgPath); err != nil {
		t.Skipf("跳过：找不到 %s（本地无 DB 配置）", cfgPath)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.MySQL.DSN == "" {
		t.Skip("跳过：MySQL DSN 为空")
	}
	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	testDB = db
	return db
}

// resetData 清掉本文件会写到的行（硬删，含已软删的回魂行）。
func resetData(t *testing.T, db *gorm.DB) {
	t.Helper()
	purgeData(db, func(sql string, err error) { t.Fatalf("cleanup %q: %v", sql, err) })
}

// cleanupAfterTest 在 t.Cleanup 里硬删自建行。失败只记日志：
// Cleanup 阶段 t.Fatalf 会中断其余清理。
func cleanupAfterTest(t *testing.T, db *gorm.DB) {
	t.Helper()
	purgeData(db, func(sql string, err error) { t.Logf("cleanup %q: %v", sql, err) })
}

func purgeData(db *gorm.DB, onErr func(string, error)) {
	del := func(sql string, args ...any) {
		if err := db.Exec(sql, args...).Error; err != nil {
			onErr(sql, err)
		}
	}
	del("DELETE FROM user_roles WHERE user_id IN (SELECT id FROM users WHERE phone = ?)", testPhone)
	del("DELETE FROM users WHERE phone = ?", testPhone)
	del("DELETE FROM kindergartens WHERE id IN (?, ?, ?)", testKGID, testKG2ID, testKGDeletedID)
}

// ensureKindergarten 创建测试园所；调用前 resetData 已硬删同 ID 的行。
func ensureKindergarten(t *testing.T, db *gorm.DB, id uint64, name string) {
	t.Helper()
	require.NoError(t, db.Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: id},
		Name:      name,
	}).Error)
}

func TestEnsureTenantAdminCreatesUserAndRole(t *testing.T) {
	db := openTestDB(t)
	resetData(t, db)
	t.Cleanup(func() { cleanupAfterTest(t, db) })
	ensureKindergarten(t, db, testKGID, "测试园所-供给")

	ctx := context.Background()
	res, err := adminprovision.EnsureTenantAdmin(ctx, db, adminprovision.Options{
		Phone:          testPhone,
		Password:       testPasswd,
		KindergartenID: testKGID,
	})
	require.NoError(t, err)
	assert.True(t, res.UserCreated, "首次供给应创建用户")
	assert.True(t, res.RoleCreated, "首次供给应创建管理员角色")

	var user domain.User
	require.NoError(t, db.Where("phone = ?", testPhone).First(&user).Error)
	assert.Equal(t, res.UserID, user.ID)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(testPasswd)),
		"落库口令应能通过 bcrypt 校验")

	var roles []domain.UserRole
	require.NoError(t, db.Where("user_id = ? AND role = ?", user.ID, "admin").Find(&roles).Error)
	require.Len(t, roles, 1, "只应有一条 admin 角色")
	require.NotNil(t, roles[0].KindergartenID)
	assert.Equal(t, testKGID, *roles[0].KindergartenID)
	assert.Equal(t, "super_admin", roles[0].AdminRole)

	assert.NotContains(t, res.Summary(), testPasswd, "输出不得包含明文口令")
}

func TestEnsureTenantAdminIsIdempotent(t *testing.T) {
	db := openTestDB(t)
	resetData(t, db)
	t.Cleanup(func() { cleanupAfterTest(t, db) })
	ensureKindergarten(t, db, testKGID, "测试园所-供给")

	ctx := context.Background()
	opts := adminprovision.Options{Phone: testPhone, Password: testPasswd, KindergartenID: testKGID}
	first, err := adminprovision.EnsureTenantAdmin(ctx, db, opts)
	require.NoError(t, err)

	second, err := adminprovision.EnsureTenantAdmin(ctx, db, opts)
	require.NoError(t, err)
	assert.False(t, second.UserCreated, "重复执行不应新建用户")
	assert.False(t, second.RoleCreated, "重复执行不应新建角色")
	assert.Equal(t, first.UserID, second.UserID)

	var userCount, roleCount int64
	require.NoError(t, db.Model(&domain.User{}).Where("phone = ?", testPhone).Count(&userCount).Error)
	assert.Equal(t, int64(1), userCount)
	require.NoError(t, db.Model(&domain.UserRole{}).
		Where("user_id = ? AND role = ?", first.UserID, "admin").Count(&roleCount).Error)
	assert.Equal(t, int64(1), roleCount)
}

func TestEnsureTenantAdminRebindsExistingRole(t *testing.T) {
	db := openTestDB(t)
	resetData(t, db)
	t.Cleanup(func() { cleanupAfterTest(t, db) })
	ensureKindergarten(t, db, testKGID, "测试园所-供给")
	ensureKindergarten(t, db, testKG2ID, "测试园所-供给二")

	ctx := context.Background()
	first, err := adminprovision.EnsureTenantAdmin(ctx, db, adminprovision.Options{
		Phone: testPhone, Password: testPasswd, KindergartenID: testKGID,
	})
	require.NoError(t, err)

	var roleBefore domain.UserRole
	require.NoError(t, db.Where("user_id = ? AND role = ?", first.UserID, "admin").First(&roleBefore).Error)

	second, err := adminprovision.EnsureTenantAdmin(ctx, db, adminprovision.Options{
		Phone: testPhone, Password: testPasswd, KindergartenID: testKG2ID,
	})
	require.NoError(t, err)
	assert.False(t, second.UserCreated)
	assert.False(t, second.RoleCreated, "改绑不应新建角色")
	assert.Equal(t, testKG2ID, second.KindergartenID)

	var roles []domain.UserRole
	require.NoError(t, db.Where("user_id = ? AND role = ?", first.UserID, "admin").Find(&roles).Error)
	require.Len(t, roles, 1, "改绑不得新增第二条角色")
	assert.Equal(t, roleBefore.ID, roles[0].ID, "应更新同一条角色记录")
	require.NotNil(t, roles[0].KindergartenID)
	assert.Equal(t, testKG2ID, *roles[0].KindergartenID)
	assert.Equal(t, "测试园所-供给二", roles[0].KindergartenName)
	assert.Equal(t, "super_admin", roles[0].AdminRole)
}

func TestEnsureTenantAdminNormalizesAdminRole(t *testing.T) {
	db := openTestDB(t)
	resetData(t, db)
	t.Cleanup(func() { cleanupAfterTest(t, db) })
	ensureKindergarten(t, db, testKGID, "测试园所-供给")

	ctx := context.Background()
	first, err := adminprovision.EnsureTenantAdmin(ctx, db, adminprovision.Options{
		Phone: testPhone, Password: testPasswd, KindergartenID: testKGID,
	})
	require.NoError(t, err)

	// 模拟历史数据：admin 角色的 admin_role 缺失或被写成别的值。
	require.NoError(t, db.Model(&domain.UserRole{}).
		Where("user_id = ? AND role = ?", first.UserID, "admin").
		Update("admin_role", "teacher").Error)

	second, err := adminprovision.EnsureTenantAdmin(ctx, db, adminprovision.Options{
		Phone: testPhone, Password: testPasswd, KindergartenID: testKGID,
	})
	require.NoError(t, err)
	assert.False(t, second.RoleCreated, "已存在的角色不应新建")

	var role domain.UserRole
	require.NoError(t, db.Where("user_id = ? AND role = ?", first.UserID, "admin").First(&role).Error)
	assert.Equal(t, "super_admin", role.AdminRole, "供给应把 admin_role 规范为 super_admin")
}

func TestEnsureTenantAdminCollapsesDuplicateAdminRoles(t *testing.T) {
	db := openTestDB(t)
	resetData(t, db)
	t.Cleanup(func() { cleanupAfterTest(t, db) })
	ensureKindergarten(t, db, testKGID, "测试园所-供给")
	ensureKindergarten(t, db, testKG2ID, "测试园所-供给二")

	ctx := context.Background()
	first, err := adminprovision.EnsureTenantAdmin(ctx, db, adminprovision.Options{
		Phone: testPhone, Password: testPasswd, KindergartenID: testKGID,
	})
	require.NoError(t, err)

	var kept domain.UserRole
	require.NoError(t, db.Where("user_id = ? AND role = ?", first.UserID, "admin").First(&kept).Error)

	// 模拟历史重复：同一 user 已有第二条 admin 角色，且值都不正确。
	kg2 := testKG2ID
	require.NoError(t, db.Create(&domain.UserRole{
		UserID:           first.UserID,
		Role:             "admin",
		KindergartenID:   &kg2,
		KindergartenName: "测试园所-供给二",
		AdminRole:        "teacher",
	}).Error)

	_, err = adminprovision.EnsureTenantAdmin(ctx, db, adminprovision.Options{
		Phone: testPhone, Password: testPasswd, KindergartenID: testKG2ID,
	})
	require.NoError(t, err)

	var active []domain.UserRole
	require.NoError(t, db.Where("user_id = ? AND role = ?", first.UserID, "admin").Find(&active).Error)
	require.Len(t, active, 1, "重复的 admin 角色应被收敛为一条")
	assert.Equal(t, kept.ID, active[0].ID, "应保留 id 最小的那条")
	require.NotNil(t, active[0].KindergartenID)
	assert.Equal(t, testKG2ID, *active[0].KindergartenID)
	assert.Equal(t, "测试园所-供给二", active[0].KindergartenName)
	assert.Equal(t, "super_admin", active[0].AdminRole)

	// 历史行按软删语义保留，不做物理删除。
	var all []domain.UserRole
	require.NoError(t, db.Unscoped().
		Where("user_id = ? AND role = ?", first.UserID, "admin").Order("id").Find(&all).Error)
	require.Len(t, all, 2, "软删行应仍在表中")
	assert.False(t, all[0].DeletedAt.Valid, "保留行不应被软删")
	assert.True(t, all[1].DeletedAt.Valid, "多余行应被软删")
}

func TestEnsureTenantAdminRejectsMissingKindergarten(t *testing.T) {
	db := openTestDB(t)

	// 已软删除的园所：创建后软删，必须与不存在同等待遇。
	resetData(t, db)
	t.Cleanup(func() { cleanupAfterTest(t, db) })
	ensureKindergarten(t, db, testKGDeletedID, "测试园所-已删除")
	require.NoError(t, db.Delete(&domain.Kindergarten{BaseModel: domain.BaseModel{ID: testKGDeletedID}}).Error)

	cases := []struct {
		name string
		kgID uint64
	}{
		{"不存在", testKGID},
		{"已软删除", testKGDeletedID},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := adminprovision.EnsureTenantAdmin(context.Background(), db, adminprovision.Options{
				Phone: testPhone, Password: testPasswd, KindergartenID: tc.kgID,
			})
			require.Error(t, err)
			assert.ErrorIs(t, err, adminprovision.ErrKindergartenNotFound)

			var userCount, roleCount int64
			require.NoError(t, db.Model(&domain.User{}).Where("phone = ?", testPhone).Count(&userCount).Error)
			assert.Zero(t, userCount, "园所校验失败不得写用户")
			require.NoError(t, db.Model(&domain.UserRole{}).Where("role = ? AND kindergarten_id = ?", "admin", tc.kgID).
				Count(&roleCount).Error)
			assert.Zero(t, roleCount, "园所校验失败不得写角色")
		})
	}
}

// 参数校验必须先于任何数据库访问：传 nil DB 也能拿到 ErrInvalidOptions。
// 若未来把校验挪到查询之后，这里会以 nil pointer panic 暴露，而不是静默通过。
func TestEnsureTenantAdminRejectsIncompleteOptions(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name string
		opts adminprovision.Options
	}{
		{"缺手机号", adminprovision.Options{Password: testPasswd, KindergartenID: testKGID}},
		{"缺口令", adminprovision.Options{Phone: testPhone, KindergartenID: testKGID}},
		{"缺园所", adminprovision.Options{Phone: testPhone, Password: testPasswd}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := adminprovision.EnsureTenantAdmin(ctx, nil, tc.opts)
			require.Error(t, err)
			assert.ErrorIs(t, err, adminprovision.ErrInvalidOptions)
		})
	}
}
