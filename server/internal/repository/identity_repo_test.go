package repository_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/internal/testutil/authharness"
)

// uniquePhone 生成本次测试专用的手机号：users.phone 是 varchar(20)，
// 19 位纳秒时间戳正好落在容量内，且足够避开共享测试库里的其他行。
func uniquePhone(i int) string {
	return fmt.Sprintf("%d%01d", time.Now().UnixNano(), i%10)
}

func createTestUser(t *testing.T, db *gorm.DB, phone string) *domain.User {
	t.Helper()
	u := &domain.User{Phone: phone}
	require.NoError(t, db.Create(u).Error, "写入测试用户")
	t.Cleanup(func() {
		// 必须 Unscoped：软删行仍占行位置，也不该留在共享测试库里。
		require.NoError(t, db.Unscoped().Where("id = ?", u.ID).Delete(&domain.User{}).Error)
	})
	return u
}

// 客户端身份唯一性由数据库兜底：同一 app 的同一 openid 只能属于一个账号；
// 同一账号在同一 app 也只能有一条身份。仓储层必须把这些冲突报出来而不是静默覆盖。
func TestUserIdentityRepo_EnforcesUniqueKeys(t *testing.T) {
	db := authharness.NewDB(t)
	repo := repository.NewUserIdentityRepo(db)
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	appID := "wx-app-" + suffix
	openID := "openid-" + suffix

	userA := createTestUser(t, db, uniquePhone(1))
	userB := createTestUser(t, db, uniquePhone(2))
	t.Cleanup(func() {
		require.NoError(t, db.Where("app_id = ?", appID).Delete(&domain.UserIdentity{}).Error)
	})

	first := &domain.UserIdentity{UserID: userA.ID, AppID: appID, OpenID: openID, UnionID: "union-" + suffix}
	require.NoError(t, repo.Create(ctx, first))
	require.NotZero(t, first.ID, "创建后必须回填主键")

	got, err := repo.FindByAppAndOpenID(ctx, appID, openID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, userA.ID, got.UserID)
	assert.Equal(t, "union-"+suffix, got.UnionID)

	// 同一 (app_id, open_id) 不能再挂到另一个账号。
	err = repo.Create(ctx, &domain.UserIdentity{UserID: userB.ID, AppID: appID, OpenID: openID})
	require.ErrorIs(t, err, repository.ErrUserIdentityExists,
		"同一客户端身份重复绑定必须失败，不能静默改挂")

	// 同一 (user_id, app_id) 不能有第二条身份。
	err = repo.Create(ctx, &domain.UserIdentity{UserID: userA.ID, AppID: appID, OpenID: openID + "-other"})
	require.ErrorIs(t, err, repository.ErrUserIdentityExists,
		"同一账号在同一客户端只能有一条身份")

	// 同账号换一个 app、或同 app 换一个账号都是合法的新身份。
	otherApp := &domain.UserIdentity{UserID: userA.ID, AppID: appID + "-2", OpenID: openID}
	require.NoError(t, repo.Create(ctx, otherApp))
	t.Cleanup(func() {
		require.NoError(t, db.Where("app_id = ?", appID+"-2").Delete(&domain.UserIdentity{}).Error)
	})

	byUserApp, err := repo.FindByUserAndApp(ctx, userA.ID, appID)
	require.NoError(t, err)
	require.NotNil(t, byUserApp)
	assert.Equal(t, first.ID, byUserApp.ID)

	missing, err := repo.FindByAppAndOpenID(ctx, appID, "openid-does-not-exist")
	require.NoError(t, err)
	assert.Nil(t, missing, "未命中返回 nil，调用方据此决定是否新建身份")
}

// 身份行没有软删除：解绑/合并要硬删，否则软删行会永久占住唯一索引。
func TestUserIdentityRepo_DeleteIsHardAndFreesUniqueKey(t *testing.T) {
	db := authharness.NewDB(t)
	repo := repository.NewUserIdentityRepo(db)
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	appID := "wx-app-del-" + suffix
	openID := "openid-del-" + suffix

	user := createTestUser(t, db, uniquePhone(3))
	identity := &domain.UserIdentity{UserID: user.ID, AppID: appID, OpenID: openID}
	require.NoError(t, repo.Create(ctx, identity))

	require.NoError(t, repo.Delete(ctx, identity.ID))

	var raw int64
	require.NoError(t, db.Unscoped().Model(&domain.UserIdentity{}).
		Where("id = ?", identity.ID).Count(&raw).Error)
	assert.Zero(t, raw, "Delete 必须硬删；软删行会继续占用唯一索引")

	// 硬删后同一身份可以重新绑定，证明唯一键已被释放。
	require.NoError(t, repo.Create(ctx, &domain.UserIdentity{UserID: user.ID, AppID: appID, OpenID: openID}))
	t.Cleanup(func() {
		require.NoError(t, db.Where("app_id = ?", appID).Delete(&domain.UserIdentity{}).Error)
	})
}

// 一个自然人的多个客户端身份归到同一条 User：这是 v1 身份模型的核心不变量。
func TestUserIdentityRepo_ListByUserIDKeepsMultipleClients(t *testing.T) {
	db := authharness.NewDB(t)
	repo := repository.NewUserIdentityRepo(db)
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	user := createTestUser(t, db, uniquePhone(4))
	t.Cleanup(func() {
		require.NoError(t, db.Where("user_id = ?", user.ID).Delete(&domain.UserIdentity{}).Error)
	})

	require.NoError(t, repo.Create(ctx, &domain.UserIdentity{UserID: user.ID, AppID: "wx-parent-" + suffix, OpenID: "o-p-" + suffix}))
	require.NoError(t, repo.Create(ctx, &domain.UserIdentity{UserID: user.ID, AppID: "wx-teacher-" + suffix, OpenID: "o-t-" + suffix}))

	list, err := repo.ListByUserID(ctx, user.ID)
	require.NoError(t, err)
	require.Len(t, list, 2)
}

// --- 手机号唯一：phone_active 生成列 ---

// 空手机号不进入唯一索引：待补全手机号的账号可以有多条。
// 这里直接验证「两条空手机号能共存」，也就是生成列对空串返回 NULL 的行为。
func TestPhoneActive_EmptyPhonesDoNotCollide(t *testing.T) {
	db := authharness.NewDB(t)

	a := &domain.User{Phone: ""}
	b := &domain.User{Phone: ""}
	require.NoError(t, db.Create(a).Error)
	t.Cleanup(func() { require.NoError(t, db.Unscoped().Where("id = ?", a.ID).Delete(&domain.User{}).Error) })
	require.NoError(t, db.Create(b).Error, "空手机号不得占用唯一索引")
	t.Cleanup(func() { require.NoError(t, db.Unscoped().Where("id = ?", b.ID).Delete(&domain.User{}).Error) })
}

// 用户的手机号在「已删除」「空」两种情况下都不占用唯一键；
// 其余情况（活跃且非空）必须唯一。
func TestPhoneActive_RejectsDuplicateActivePhone(t *testing.T) {
	db := authharness.NewDB(t)
	phone := uniquePhone(5)

	first := createTestUser(t, db, phone)

	dup := &domain.User{Phone: phone}
	err := db.Create(dup).Error
	require.Error(t, err, "两个活跃账号不得共用同一手机号")
	if dup.ID != 0 {
		t.Cleanup(func() { require.NoError(t, db.Unscoped().Where("id = ?", dup.ID).Delete(&domain.User{}).Error) })
	}

	// 软删首条后，同一手机号必须能重新创建——否则账号永远无法重建。
	require.NoError(t, db.Delete(first).Error)
	rebuild := &domain.User{Phone: phone}
	require.NoError(t, db.Create(rebuild).Error, "软删行的手机号不得继续占用唯一键")
	t.Cleanup(func() { require.NoError(t, db.Unscoped().Where("id = ?", rebuild.ID).Delete(&domain.User{}).Error) })

	// 多条软删行的 phone_active 都是 NULL，可以共存；再建一条活跃账号也不受影响。
	require.NoError(t, db.Delete(rebuild).Error)
	third := &domain.User{Phone: phone}
	require.NoError(t, db.Create(third).Error, "软删行不得占用唯一键")
	t.Cleanup(func() { require.NoError(t, db.Unscoped().Where("id = ?", third.ID).Delete(&domain.User{}).Error) })
}

// --- 统一账号解析 ---

// 同一活跃手机号只解析出一个账号；复用时不得覆盖已有资料。
func TestFindOrCreateByPhone_ReusesAccountWithoutOverwritingProfile(t *testing.T) {
	db := authharness.NewDB(t)
	repo := repository.NewUserRepo(db)
	ctx := context.Background()
	phone := uniquePhone(6)
	t.Cleanup(func() {
		require.NoError(t, db.Unscoped().Where("phone = ?", phone).Delete(&domain.User{}).Error)
	})

	created, isNew, err := repo.FindOrCreateByPhone(ctx, phone, repository.UserProfileDefaults{
		Nickname:     "原始昵称",
		Avatar:       "oss://avatar-a.png",
		Gender:       2,
		PasswordHash: "hash-a",
		IDCardNumber: "id-card-a",
	})
	require.NoError(t, err)
	require.True(t, isNew, "首次解析应创建账号")
	require.NotZero(t, created.ID)
	assert.Equal(t, phone, created.Phone)
	assert.Equal(t, "原始昵称", created.Nickname)

	reused, isNew, err := repo.FindOrCreateByPhone(ctx, phone, repository.UserProfileDefaults{
		Nickname:     "管理员填的昵称",
		Avatar:       "oss://avatar-b.png",
		Gender:       1,
		PasswordHash: "hash-b",
		IDCardNumber: "id-card-b",
	})
	require.NoError(t, err)
	assert.False(t, isNew, "手机号已有账号时必须复用")
	assert.Equal(t, created.ID, reused.ID, "同一手机号必须始终解析到同一账号")

	assert.Equal(t, "原始昵称", reused.Nickname, "复用不得覆盖已有昵称")
	assert.Equal(t, "oss://avatar-a.png", reused.Avatar, "复用不得覆盖已有头像")
	assert.Equal(t, 2, reused.Gender, "复用不得覆盖已有性别")
	assert.Equal(t, "hash-a", reused.PasswordHash, "复用不得覆盖已有密码")
	assert.Equal(t, "id-card-a", reused.IDCardNumber, "复用不得覆盖已有身份证号")
}

// 空白手机号没有账号语义，必须显式拒绝，而不是写入一条空账号。
func TestFindOrCreateByPhone_RejectsBlankPhone(t *testing.T) {
	db := authharness.NewDB(t)
	repo := repository.NewUserRepo(db)
	ctx := context.Background()

	var before int64
	require.NoError(t, db.Unscoped().Model(&domain.User{}).Where("phone = ''").Count(&before).Error)

	_, _, err := repo.FindOrCreateByPhone(ctx, "   ", repository.UserProfileDefaults{Nickname: "空白"})
	require.ErrorIs(t, err, repository.ErrPhoneRequired)

	var after int64
	require.NoError(t, db.Unscoped().Model(&domain.User{}).Where("phone = ''").Count(&after).Error)
	assert.Equal(t, before, after, "拒绝时不得写入任何行")
}

// 软删账号不可被解析出来；同一手机号应重建一个新账号（新 id）。
func TestFindOrCreateByPhone_SoftDeletedAccountIsNotReused(t *testing.T) {
	db := authharness.NewDB(t)
	repo := repository.NewUserRepo(db)
	ctx := context.Background()
	phone := uniquePhone(7)

	old := createTestUser(t, db, phone)
	oldID := old.ID
	require.NoError(t, db.Delete(old).Error)

	rebuilt, isNew, err := repo.FindOrCreateByPhone(ctx, phone, repository.UserProfileDefaults{Nickname: "重建"})
	require.NoError(t, err)
	require.True(t, isNew, "软删账号不得被复用")
	assert.NotEqual(t, oldID, rebuilt.ID, "重建必须是新账号")
	t.Cleanup(func() {
		require.NoError(t, db.Unscoped().Where("phone = ?", phone).Delete(&domain.User{}).Error)
	})

	// 软删账号也不能被按手机号查到——否则密码登录会登上已注销账号。
	got, err := repo.FindByPhone(ctx, phone)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, rebuilt.ID, got.ID, "FindByPhone 只能命中活跃账号")
}

// 并发解析同一手机号必须收敛到一个账号：唯一索引拒绝第二个插入，
// 解析器要把它当作「别人已经建好」重新读取，而不是把错误抛给调用方。
func TestFindOrCreateByPhone_ConcurrentCallsConvergeToOneAccount(t *testing.T) {
	db := authharness.NewDB(t)
	repo := repository.NewUserRepo(db)
	ctx := context.Background()
	phone := uniquePhone(8)
	t.Cleanup(func() {
		require.NoError(t, db.Unscoped().Where("phone = ?", phone).Delete(&domain.User{}).Error)
	})

	const goroutines = 8
	ids := make([]uint64, goroutines)
	errs := make([]error, goroutines)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			u, _, err := repo.FindOrCreateByPhone(ctx, phone, repository.UserProfileDefaults{Nickname: fmt.Sprintf("并发-%d", i)})
			if err != nil {
				errs[i] = err
				return
			}
			ids[i] = u.ID
		}(i)
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		require.NoError(t, err, "并发解析第 %d 个调用不应失败", i)
	}
	for i := 1; i < goroutines; i++ {
		assert.Equal(t, ids[0], ids[i], "并发解析必须全部收敛到同一账号")
	}

	var rows int64
	require.NoError(t, db.Unscoped().Model(&domain.User{}).Where("phone = ?", phone).Count(&rows).Error)
	assert.EqualValues(t, 1, rows, "同一手机号只能落一个账号")
}
