package service

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// 开通园所（#171）测试使用独立前缀与手机号，清理只碰本文件造的数据：
// 平台测试共用开发库，不能按表全删。
const (
	onboardTestNamePrefix = "S1E开通测试"
	onboardTestOperator   = "s1e_ops"
)

func cleanupOnboardTestData(t *testing.T, db *gorm.DB, phones ...string) {
	t.Helper()
	del := func(sql string, args ...any) {
		if err := db.Exec(sql, args...).Error; err != nil {
			t.Logf("cleanup %q: %v", sql, err)
		}
	}
	del("DELETE FROM entitlements WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE ?)", onboardTestNamePrefix+"%")
	del("DELETE FROM platform_operation_logs WHERE operator = ?", onboardTestOperator)
	del("DELETE FROM admin_role_permissions WHERE role_id IN (SELECT id FROM admin_roles WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE ?))", onboardTestNamePrefix+"%")
	del("DELETE FROM admin_roles WHERE kindergarten_id IN (SELECT id FROM kindergartens WHERE name LIKE ?)", onboardTestNamePrefix+"%")
	for _, phone := range phones {
		del("DELETE FROM user_admin_roles WHERE user_id IN (SELECT id FROM users WHERE phone = ?)", phone)
		del("DELETE FROM user_roles WHERE user_id IN (SELECT id FROM users WHERE phone = ?)", phone)
		del("DELETE FROM parent_profiles WHERE user_id IN (SELECT id FROM users WHERE phone = ?)", phone)
		del("DELETE FROM teacher_profiles WHERE user_id IN (SELECT id FROM users WHERE phone = ?)", phone)
		del("DELETE FROM users WHERE phone = ?", phone)
	}
	del("DELETE FROM kindergartens WHERE name LIKE ?", onboardTestNamePrefix+"%")
}

func onboardTestKindergarten(t *testing.T, db *gorm.DB, suffix string) *domain.Kindergarten {
	t.Helper()
	kg := &domain.Kindergarten{Name: onboardTestNamePrefix + suffix, Status: domain.KindergartenStatusActive}
	require.NoError(t, db.Create(kg).Error)
	return kg
}

func onboardTestUser(t *testing.T, db *gorm.DB, phone, nickname, avatar string, gender int, password string) *domain.User {
	t.Helper()
	hash, err := HashPassword(password)
	require.NoError(t, err)
	u := &domain.User{Phone: phone, Nickname: nickname, Avatar: avatar, Gender: gender, PasswordHash: hash}
	require.NoError(t, db.Create(u).Error)
	return u
}

// TestPlatformOnboard_ReusesExistingParentAccount 是 #171 的核心场景：
// 园长手机号已经是另一所园的家长账号时，开通必须复用同一个 user，
// 而不是再建第二个账号（一手机号一账号）。
func TestPlatformOnboard_ReusesExistingParentAccount(t *testing.T) {
	db := openPlatformTestDB(t)
	svc := newPlatformTestService(t, db)
	ctx := context.Background()
	phone := "19900017101"
	cleanupOnboardTestData(t, db, phone)
	t.Cleanup(func() { cleanupOnboardTestData(t, db, phone) })

	otherKG := onboardTestKindergarten(t, db, "既有园")
	parent := onboardTestUser(t, db, phone, "豆豆妈妈", "https://cdn.example.com/avatar.png", 2, "ParentSecret1")
	otherKGID := otherKG.ID
	require.NoError(t, db.Create(&domain.UserRole{
		UserID: parent.ID, Role: "parent", KindergartenID: &otherKGID, KindergartenName: otherKG.Name,
	}).Error)
	require.NoError(t, db.Create(&domain.ParentProfile{
		UserID: parent.ID, KindergartenID: otherKGID, RealName: "刘女士", Status: 1,
	}).Error)

	res, err := svc.Onboard(ctx, OnboardRequest{
		Name: onboardTestNamePrefix + "新园", Principal: "刘园长", Phone: phone, AdminPassword: "InitPass1",
	}, onboardTestOperator, "127.0.0.1")
	require.NoError(t, err, "手机号已有家长账号时必须复用，而不是拒绝")
	require.NotZero(t, res.KindergartenID)

	// 一手机号一账号：不因开通再建第二个 user。
	var users []domain.User
	require.NoError(t, db.Where("phone = ?", phone).Find(&users).Error)
	require.Len(t, users, 1)
	assert.Equal(t, parent.ID, users[0].ID)

	// 原有资料（昵称/头像/性别）必须保留；只更新口令与首登改密标记。
	assert.Equal(t, "豆豆妈妈", users[0].Nickname)
	assert.Equal(t, "https://cdn.example.com/avatar.png", users[0].Avatar)
	assert.Equal(t, 2, users[0].Gender)
	assert.True(t, users[0].MustChangePassword, "新园长首次登录必须改密")
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(users[0].PasswordHash), []byte("InitPass1")),
		"开通时下发的管理端口令必须生效")

	// 既有家长角色与档案不能被复用路径吞掉。
	existingParentRole, err := repository.NewUserRoleRepo(db).FindByUserIDAndRole(ctx, parent.ID, "parent")
	require.NoError(t, err)
	require.NotNil(t, existingParentRole)
	require.NotNil(t, existingParentRole.KindergartenID)
	assert.Equal(t, otherKGID, *existingParentRole.KindergartenID)
	var profiles int64
	require.NoError(t, db.Model(&domain.ParentProfile{}).Where("user_id = ?", parent.ID).Count(&profiles).Error)
	assert.Equal(t, int64(1), profiles)

	// 新园管理员具备通配权限（走 UserAdminRole → 权限码这条真实鉴权路径）。
	codes, wildcard, err := NewAdminAuthzService(repository.NewAdminRepo(db), repository.NewUserRoleRepo(db)).
		ResolveAdminCodes(ctx, res.KindergartenID, parent.ID)
	require.NoError(t, err)
	assert.True(t, wildcard, "新园长应获得通配权限，实际 codes=%v", codes)

	// 开通审计必须落库，且指向新园。
	var logs int64
	require.NoError(t, db.Model(&domain.PlatformOperationLog{}).
		Where("operator = ? AND action = ? AND target_id = ?", onboardTestOperator, "onboard", res.KindergartenID).
		Count(&logs).Error)
	assert.Equal(t, int64(1), logs)
}

// TestPlatformOnboard_ReusesExistingTeacherAccount 覆盖另一种既有身份：
// 手机号是另一所园的老师。老师角色与档案同样不能丢。
func TestPlatformOnboard_ReusesExistingTeacherAccount(t *testing.T) {
	db := openPlatformTestDB(t)
	svc := newPlatformTestService(t, db)
	ctx := context.Background()
	phone := "19900017102"
	cleanupOnboardTestData(t, db, phone)
	t.Cleanup(func() { cleanupOnboardTestData(t, db, phone) })

	otherKG := onboardTestKindergarten(t, db, "老师既有园")
	teacher := onboardTestUser(t, db, phone, "李老师", "", 1, "TeacherSecret1")
	otherKGID := otherKG.ID
	require.NoError(t, db.Create(&domain.UserRole{
		UserID: teacher.ID, Role: "teacher", KindergartenID: &otherKGID, KindergartenName: otherKG.Name,
	}).Error)
	require.NoError(t, db.Create(&domain.TeacherProfile{
		UserID: teacher.ID, KindergartenID: otherKGID, RealName: "李老师",
		JoinDate: time.Now(), Status: 1,
	}).Error)

	res, err := svc.Onboard(ctx, OnboardRequest{
		Name: onboardTestNamePrefix + "老师新园", Principal: "李园长", Phone: phone, AdminPassword: "InitPass1",
	}, onboardTestOperator, "127.0.0.1")
	require.NoError(t, err)
	require.NotZero(t, res.KindergartenID)

	var users []domain.User
	require.NoError(t, db.Where("phone = ?", phone).Find(&users).Error)
	require.Len(t, users, 1)
	assert.Equal(t, teacher.ID, users[0].ID)
	assert.Equal(t, "李老师", users[0].Nickname)
	assert.True(t, users[0].MustChangePassword)

	existingTeacherRole, err := repository.NewUserRoleRepo(db).FindByUserIDAndRole(ctx, teacher.ID, "teacher")
	require.NoError(t, err)
	require.NotNil(t, existingTeacherRole)
	var profiles int64
	require.NoError(t, db.Model(&domain.TeacherProfile{}).Where("user_id = ?", teacher.ID).Count(&profiles).Error)
	assert.Equal(t, int64(1), profiles)

	_, wildcard, err := NewAdminAuthzService(repository.NewAdminRepo(db), repository.NewUserRoleRepo(db)).
		ResolveAdminCodes(ctx, res.KindergartenID, teacher.ID)
	require.NoError(t, err)
	assert.True(t, wildcard)
}

// TestPlatformOnboard_AssignsUniquePublicCode 新园必须拿到公开入口码，
// 返回给平台控制台，并可直接用于家长端入口解析；不泄露内部字段。
func TestPlatformOnboard_AssignsUniquePublicCode(t *testing.T) {
	db := openPlatformTestDB(t)
	svc := newPlatformTestService(t, db)
	ctx := context.Background()
	phones := []string{"19900017105", "19900017106"}
	cleanupOnboardTestData(t, db, phones...)
	t.Cleanup(func() { cleanupOnboardTestData(t, db, phones...) })

	first, err := svc.Onboard(ctx, OnboardRequest{
		Name: onboardTestNamePrefix + "公开码一号园", Principal: "周园长", Phone: phones[0], AdminPassword: "InitPass1",
	}, onboardTestOperator, "127.0.0.1")
	require.NoError(t, err)
	second, err := svc.Onboard(ctx, OnboardRequest{
		Name: onboardTestNamePrefix + "公开码二号园", Principal: "吴园长", Phone: phones[1], AdminPassword: "InitPass1",
	}, onboardTestOperator, "127.0.0.1")
	require.NoError(t, err)

	assert.Len(t, first.PublicCode, PublicCodeLength, "开通结果必须回传公开码")
	assert.Len(t, second.PublicCode, PublicCodeLength)
	assert.NotEqual(t, first.PublicCode, second.PublicCode, "公开码必须唯一")
	for _, code := range []string{first.PublicCode, second.PublicCode} {
		assert.Regexp(t, "^[0-9abcdefghjkmnpqrstvwxyz]{10}$", code, "公开码字符集必须是 Crockford base32 小写")
	}

	kgRepo := repository.NewKindergartenRepo(db)
	stored, err := kgRepo.FindByID(ctx, first.KindergartenID)
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.Equal(t, first.PublicCode, stored.PublicCode, "返回的公开码必须真的落库")

	// 平台控制台可见：详情与列表都要带出来。
	detail, err := svc.GetKindergarten(ctx, first.KindergartenID)
	require.NoError(t, err)
	assert.Equal(t, first.PublicCode, detail.PublicCode)

	rows, _, err := svc.ListKindergartens(ctx, 1, 20, onboardTestNamePrefix+"公开码一号园", "")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, first.PublicCode, rows[0].PublicCode)

	// 家长端入口按公开码能解析到同一所园。
	entry := NewKindergartenEntryService(kgRepo)
	found, err := entry.FindByPublicCode(ctx, first.PublicCode)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, first.KindergartenID, found.ID)
}

// TestPlatformOnboard_PublicCodeCollisionRetries 候选码已被占用时必须换码，
// 不能把撞码的园所留在「无公开码」状态。注入确定性随机源来复现撞码。
func TestPlatformOnboard_PublicCodeCollisionRetries(t *testing.T) {
	db := openPlatformTestDB(t)
	ctx := context.Background()
	phone := "19900017107"
	cleanupOnboardTestData(t, db, phone)
	t.Cleanup(func() { cleanupOnboardTestData(t, db, phone) })

	// 预占字节 0..9 对应的码，开通时第一个候选必然撞上。
	taken := &domain.Kindergarten{
		Name: onboardTestNamePrefix + "占位园", Status: domain.KindergartenStatusActive, PublicCode: "0123456789",
	}
	require.NoError(t, db.Create(taken).Error)

	// 每次生成消耗 10 字节：第一组 0..9 → "0123456789"（已占用），
	// 第二组全 10 → "aaaaaaaaaa"（可用）。
	random := bytes.NewReader([]byte{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
		10, 10, 10, 10, 10, 10, 10, 10, 10, 10,
	})
	entry := NewKindergartenEntryServiceWithRandom(repository.NewKindergartenRepo(db), random)
	svc := newPlatformTestServiceWithEntry(t, db, entry)

	res, err := svc.Onboard(ctx, OnboardRequest{
		Name: onboardTestNamePrefix + "撞码园", Principal: "郑园长", Phone: phone, AdminPassword: "InitPass1",
	}, onboardTestOperator, "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, "aaaaaaaaaa", res.PublicCode, "撞码后必须换下一个候选")

	stored, err := repository.NewKindergartenRepo(db).FindByID(ctx, res.KindergartenID)
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.Equal(t, "aaaaaaaaaa", stored.PublicCode)
}

// TestPlatformOnboard_RejectsPhoneAlreadyAdmin 一手机号最多一个园所管理员。
// 拒绝必须发生在建园之前：失败的请求不能留下半开的园所。
func TestPlatformOnboard_RejectsPhoneAlreadyAdmin(t *testing.T) {
	db := openPlatformTestDB(t)
	svc := newPlatformTestService(t, db)
	ctx := context.Background()
	phones := []string{"19900017103", "19900017104"}
	cleanupOnboardTestData(t, db, phones...)
	t.Cleanup(func() { cleanupOnboardTestData(t, db, phones...) })

	t.Run("开通出来的管理员不能再开第二所园", func(t *testing.T) {
		phone := phones[0]
		first, err := svc.Onboard(ctx, OnboardRequest{
			Name: onboardTestNamePrefix + "一号园", Principal: "王园长", Phone: phone, AdminPassword: "InitPass1",
		}, onboardTestOperator, "127.0.0.1")
		require.NoError(t, err)

		_, err = svc.Onboard(ctx, OnboardRequest{
			Name: onboardTestNamePrefix + "二号园", Principal: "王园长", Phone: phone, AdminPassword: "InitPass1",
		}, onboardTestOperator, "127.0.0.1")
		require.ErrorIs(t, err, ErrPhoneAlreadyAdmin)

		var n int64
		require.NoError(t, db.Model(&domain.Kindergarten{}).
			Where("name = ?", onboardTestNamePrefix+"二号园").Count(&n).Error)
		assert.Zero(t, n, "被拒绝的开通不能创建园所")

		var user domain.User
		require.NoError(t, db.Where("phone = ?", phone).First(&user).Error)
		var adminRoles []domain.UserRole
		require.NoError(t, db.Where("user_id = ? AND role = ?", user.ID, "admin").Find(&adminRoles).Error)
		require.Len(t, adminRoles, 1, "同一手机号只能有一条 admin 角色")
		require.NotNil(t, adminRoles[0].KindergartenID)
		assert.Equal(t, first.KindergartenID, *adminRoles[0].KindergartenID, "管理员仍属于第一所园")
	})

	t.Run("setadmin 建出来的管理员同样被拒绝", func(t *testing.T) {
		phone := phones[1]
		otherKG := onboardTestKindergarten(t, db, "setadmin既有园")
		u := onboardTestUser(t, db, phone, "赵老师", "", 1, "LegacyPass1")
		otherKGID := otherKG.ID
		require.NoError(t, db.Create(&domain.UserRole{
			UserID: u.ID, Role: "admin", KindergartenID: &otherKGID,
			KindergartenName: otherKG.Name, AdminRole: "super_admin",
		}).Error)

		_, err := svc.Onboard(ctx, OnboardRequest{
			Name: onboardTestNamePrefix + "抢开园", Principal: "赵园长", Phone: phone, AdminPassword: "InitPass1",
		}, onboardTestOperator, "127.0.0.1")
		require.ErrorIs(t, err, ErrPhoneAlreadyAdmin)
	})
}
