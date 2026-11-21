package service_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/service"
	"kindergarten-server/internal/testutil/authharness"
)

// uniqueMergePhone 生成 20 位以内的测试手机号：users.phone 是 varchar(20)，
// 纳秒时间戳 + 序号足够避开共享测试库里的其他行，也不会撞 phone_active 唯一键。
func uniqueMergePhone(seq int) string {
	return fmt.Sprintf("%d%d", time.Now().UnixNano(), seq%10)
}

// mergeFixture 是一次合并测试的最小租户：园所 + 班级 + 幼儿 + 预创建账号 + 待绑定账号。
//
// 待绑定账号只持有客户端身份，模拟 #175 登录后、尚未绑定手机号的账号。
type mergeFixture struct {
	DB      *gorm.DB
	Suffix  string
	KG      *domain.Kindergarten
	Class   *domain.Class
	Student *domain.Student
	Target  *domain.User
	Pending *domain.User
	AppID   string
	OpenID  string
}

func newMergeFixture(t *testing.T) *mergeFixture {
	t.Helper()
	db := authharness.NewDB(t)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	kg := &domain.Kindergarten{Name: "合并测试园-" + suffix, AppChannel: domain.AppChannelShared}
	require.NoError(t, db.Create(kg).Error, "创建测试园所")
	class := &domain.Class{KindergartenID: kg.ID, Name: "合并测试班-" + suffix}
	require.NoError(t, db.Create(class).Error, "创建测试班级")
	student := &domain.Student{
		KindergartenID: kg.ID,
		ClassID:        class.ID,
		Name:           "合并测试幼儿-" + suffix,
		Birthday:       time.Now(),
		EnrollDate:     time.Now(),
	}
	require.NoError(t, db.Create(student).Error, "创建测试幼儿")

	target := &domain.User{Phone: uniqueMergePhone(1), Nickname: "园所录入昵称"}
	require.NoError(t, db.Create(target).Error, "创建预创建账号")
	pending := &domain.User{Nickname: "微信昵称-" + suffix, Avatar: "oss://avatar/" + suffix}
	require.NoError(t, db.Create(pending).Error, "创建待绑定账号")

	f := &mergeFixture{
		DB: db, Suffix: suffix,
		KG: kg, Class: class, Student: student,
		Target: target, Pending: pending,
		AppID:  "wx-merge-" + suffix,
		OpenID: "openid-merge-" + suffix,
	}
	require.NoError(t, db.Create(&domain.UserIdentity{
		UserID: pending.ID, AppID: f.AppID, OpenID: f.OpenID,
	}).Error, "创建待绑定账号的客户端身份")

	t.Cleanup(func() { f.cleanup(t) })
	return f
}

// cleanup 按外键依赖倒序硬删测试行。合并会删除待绑定账号，
// 所以必须 Unscoped；否则软删行会继续占用 phone_active 等唯一索引。
func (f *mergeFixture) cleanup(t *testing.T) {
	t.Helper()
	ids := []any{f.Target.ID, f.Pending.ID}
	// 同一语句可能出现 2 个或 4 个占位符（user_id / creator_id 成对），
	// 按占位符个数轮流填两个用户 id。
	two := func(q string) {
		n := strings.Count(q, "?")
		args := make([]any, n)
		for i := 0; i < n; i++ {
			args[i] = ids[i%len(ids)]
		}
		require.NoError(t, f.DB.Exec(q, args...).Error, "清理测试数据：%s", q)
	}
	one := func(q string, args ...any) {
		require.NoError(t, f.DB.Exec(q, args...).Error, "清理测试数据：%s", q)
	}
	two("DELETE FROM user_identities WHERE user_id IN (?, ?)")
	two("DELETE FROM user_consents WHERE user_id IN (?, ?)")
	two("DELETE FROM messages WHERE user_id IN (?, ?)")
	two("DELETE FROM wechat_push_logs WHERE user_id IN (?, ?)")
	two("DELETE FROM notice_read_logs WHERE user_id IN (?, ?)")
	two("DELETE FROM teacher_message_reads WHERE user_id IN (?, ?)")
	two("DELETE FROM parent_settings WHERE user_id IN (?, ?)")
	two("DELETE FROM growth_likes WHERE user_id IN (?, ?)")
	two("DELETE FROM growth_comment_likes WHERE user_id IN (?, ?)")
	two("DELETE FROM files WHERE user_id IN (?, ?)")
	two("DELETE FROM file_records WHERE uploader_id IN (?, ?)")
	two("DELETE FROM feedbacks WHERE user_id IN (?, ?)")
	two("DELETE FROM growth_comments WHERE user_id IN (?, ?)")
	two("DELETE FROM growth_media WHERE uploader_id IN (?, ?)")
	two("DELETE FROM attendances WHERE recorder_id IN (?, ?)")
	two("DELETE FROM morning_health_checks WHERE operator_id IN (?, ?)")
	two("DELETE FROM pickup_records WHERE operator_id IN (?, ?)")
	two("DELETE FROM pickup_authorizations WHERE user_id IN (?, ?) OR creator_id IN (?, ?)")
	two("DELETE FROM leave_requests WHERE applicant_id IN (?, ?) OR reviewer_id IN (?, ?)")
	two("DELETE FROM announcements WHERE creator_id IN (?, ?)")
	two("DELETE FROM calendar_events WHERE creator_id IN (?, ?)")
	two("DELETE FROM growth_daily_logs WHERE creator_id IN (?, ?)")
	two("DELETE FROM notices WHERE creator_id IN (?, ?)")
	two("DELETE FROM notice_templates WHERE creator_id IN (?, ?)")
	// task_check_ins.task_id 外键指向 parent_tasks，必须先删 check-in。
	two("DELETE FROM task_check_ins WHERE user_id IN (?, ?)")
	two("DELETE FROM parent_tasks WHERE creator_id IN (?, ?)")
	two("DELETE FROM task_read_marks WHERE user_id IN (?, ?)")
	two("DELETE FROM task_response_reactions WHERE user_id IN (?, ?)")
	two("DELETE FROM task_responses WHERE teacher_id IN (?, ?)")
	two("DELETE FROM albums WHERE creator_id IN (?, ?) OR published_by IN (?, ?)")
	two("DELETE FROM class_star_jars WHERE created_by IN (?, ?) OR updated_by IN (?, ?)")
	two("DELETE FROM content_audits WHERE reviewer_id IN (?, ?)")
	two("DELETE FROM teacher_invitations WHERE user_id IN (?, ?)")
	two("DELETE FROM user_admin_roles WHERE user_id IN (?, ?)")
	two("DELETE FROM user_roles WHERE user_id IN (?, ?)")
	two("DELETE FROM parent_profiles WHERE user_id IN (?, ?)")
	two("DELETE FROM teacher_profiles WHERE user_id IN (?, ?)")
	two("DELETE FROM student_parents WHERE user_id IN (?, ?)")
	two("DELETE FROM class_teachers WHERE user_id IN (?, ?)")
	one("DELETE FROM point_accounts WHERE student_id = ?", f.Student.ID)
	one("DELETE FROM students WHERE id = ?", f.Student.ID)
	one("DELETE FROM classes WHERE id = ?", f.Class.ID)
	two("DELETE FROM users WHERE id IN (?, ?)")
	one("DELETE FROM kindergartens WHERE id = ?", f.KG.ID)
}

func (f *mergeFixture) merge(t *testing.T) (*service.IdentityMergeResult, error) {
	t.Helper()
	return f.mergeWith(t, service.IdentityMergeRequest{})
}

func (f *mergeFixture) mergeWith(t *testing.T, req service.IdentityMergeRequest) (*service.IdentityMergeResult, error) {
	t.Helper()
	req.AppID = f.AppID
	req.OpenID = f.OpenID
	req.TargetUserID = f.Target.ID
	return service.NewIdentityMergeService(f.DB).Merge(context.Background(), req)
}

func (f *mergeFixture) identity(t *testing.T) *domain.UserIdentity {
	t.Helper()
	var row domain.UserIdentity
	require.NoError(t, f.DB.Where("app_id = ? AND open_id = ?", f.AppID, f.OpenID).First(&row).Error,
		"读取客户端身份")
	return &row
}

func (f *mergeFixture) userExists(t *testing.T, id uint64) bool {
	t.Helper()
	var count int64
	require.NoError(t, f.DB.Unscoped().Model(&domain.User{}).Where("id = ?", id).Count(&count).Error)
	return count > 0
}

// 合并方向固定：待绑定登录账号 → 预创建账号。身份挂到预创建账号上，
// 待绑定账号被硬删，昵称/头像只在预创建账号为空时补齐，不覆盖园所录入的数据。
func TestIdentityMerge_MovesIdentityAndKeepsTargetProfile(t *testing.T) {
	f := newMergeFixture(t)

	// 预创建账号有园所录入的昵称，没有头像；待绑定账号有微信昵称与头像。
	require.NoError(t, f.DB.Model(&domain.User{}).Where("id = ?", f.Target.ID).
		Update("avatar", "").Error)

	result, err := f.merge(t)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.AlreadyMerged)
	assert.Equal(t, f.Target.ID, result.TargetUserID)

	identity := f.identity(t)
	assert.Equal(t, f.Target.ID, identity.UserID, "客户端身份必须挂到预创建账号")

	assert.False(t, f.userExists(t, f.Pending.ID), "待绑定账号必须被硬删（Unscoped 也查不到）")

	var target domain.User
	require.NoError(t, f.DB.First(&target, f.Target.ID).Error)
	assert.Equal(t, "园所录入昵称", target.Nickname, "预创建账号已有昵称不得被覆盖")
	assert.Equal(t, "oss://avatar/"+f.Suffix, target.Avatar, "预创建账号缺失头像时用待绑定账号补齐")
	assert.Equal(t, f.Target.Phone, target.Phone, "手机号仍属于预创建账号")
}

// 预创建账号已在同一客户端绑定另一个微信时，合并必须拒绝：
// 不能把先绑定的身份挤掉。拒绝时不得留下任何半套写入。
func TestIdentityMerge_RejectsTargetBoundInSameApp(t *testing.T) {
	f := newMergeFixture(t)
	otherOpenID := "openid-other-" + f.Suffix
	require.NoError(t, f.DB.Create(&domain.UserIdentity{
		UserID: f.Target.ID, AppID: f.AppID, OpenID: otherOpenID,
	}).Error, "给预创建账号绑定另一个微信")

	_, err := f.merge(t)
	require.ErrorIs(t, err, service.ErrIdentityMergePhoneAlreadyBound)

	identity := f.identity(t)
	assert.Equal(t, f.Pending.ID, identity.UserID, "冲突时待绑定账号的身份不得被挪走")
	assert.True(t, f.userExists(t, f.Pending.ID), "冲突时待绑定账号必须保留")

	var other domain.UserIdentity
	require.NoError(t, f.DB.Where("app_id = ? AND open_id = ?", f.AppID, otherOpenID).
		First(&other).Error)
	assert.Equal(t, f.Target.ID, other.UserID, "预创建账号原有身份不得被挤掉")
}

// 重复调用同一次合并是重试，不是错误：身份已在预创建账号上时直接成功返回。
func TestIdentityMerge_IsIdempotentOnRetry(t *testing.T) {
	f := newMergeFixture(t)

	first, err := f.merge(t)
	require.NoError(t, err)
	assert.False(t, first.AlreadyMerged)

	second, err := f.merge(t)
	require.NoError(t, err)
	require.NotNil(t, second)
	assert.True(t, second.AlreadyMerged, "身份已属于预创建账号时重复调用应返回已合并")
	assert.Equal(t, f.Target.ID, second.TargetUserID)

	identity := f.identity(t)
	assert.Equal(t, f.Target.ID, identity.UserID)
}

// 预创建账号已有头像时同样不得覆盖。
func TestIdentityMerge_DoesNotOverwriteExistingAvatar(t *testing.T) {
	f := newMergeFixture(t)
	require.NoError(t, f.DB.Model(&domain.User{}).Where("id = ?", f.Target.ID).
		Update("avatar", "oss://target/avatar.png").Error)

	_, err := f.merge(t)
	require.NoError(t, err)

	var target domain.User
	require.NoError(t, f.DB.First(&target, f.Target.ID).Error)
	assert.Equal(t, "oss://target/avatar.png", target.Avatar, "已有头像不得被待绑定账号覆盖")
}

// 身份不存在说明调用方没先登录，必须显式失败，不能凭空造出关联。
func TestIdentityMerge_RejectsUnknownIdentity(t *testing.T) {
	f := newMergeFixture(t)

	_, err := service.NewIdentityMergeService(f.DB).Merge(context.Background(), service.IdentityMergeRequest{
		AppID: f.AppID, OpenID: "openid-missing-" + f.Suffix, TargetUserID: f.Target.ID,
	})
	require.ErrorIs(t, err, service.ErrIdentityMergeIdentityNotFound)
}

// 幂等重试也必须确认预创建账号仍然可用：target 被软删后不能假报成功，
// 否则调用方会拿到一个登录不进去的账号。
func TestIdentityMerge_RejectsSoftDeletedTargetOnRetry(t *testing.T) {
	f := newMergeFixture(t)
	_, err := f.merge(t)
	require.NoError(t, err)

	require.NoError(t, f.DB.Delete(&domain.User{}, f.Target.ID).Error)

	_, err = f.merge(t)
	require.ErrorIs(t, err, service.ErrIdentityMergeTargetNotFound)

	identity := f.identity(t)
	assert.Equal(t, f.Target.ID, identity.UserID, "失败时不得改动身份归属")
}
