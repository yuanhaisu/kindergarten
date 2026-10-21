package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/domain"
)

func user(id uint64, openID, unionID, phone string) *domain.User {
	return &domain.User{
		BaseModel: domain.BaseModel{ID: id},
		OpenID:    openID,
		UnionID:   unionID,
		Phone:     phone,
	}
}

// --- TC-NEW-ATH-0009 / 0010：unionid 优先 ---

// TC-NEW-ATH-0009 两端都有 unionid 且相同 → 同一人。
func TestSamePerson_BothUnionIDEqual(t *testing.T) {
	a := user(1, "o_teacher_1", "u_same", "13800001111")
	b := user(2, "o_parent_1", "u_same", "13800002222")

	assert.True(t, SamePerson(a, b), "两端 unionid 相同必须判为同一人")
}

// TC-NEW-ATH-0010 两端 unionid 都非空但不同 → 不同人，即使手机号相同。
func TestSamePerson_BothUnionIDDifferent(t *testing.T) {
	a := user(1, "o_teacher_1", "u_a", "13800001111")
	b := user(2, "o_parent_1", "u_b", "13800001111")

	assert.False(t, SamePerson(a, b),
		"unionid 都非空时以它为准，不得因手机号一致而判为同人")
}

// unionid 是首选键：两侧都有时不能被手机号推翻。
func TestSamePerson_UnionIDTakesPrecedenceOverPhone(t *testing.T) {
	// unionid 相同、手机号不同 → 同人
	assert.True(t, SamePerson(
		user(1, "o_t", "u_same", "13800001111"),
		user(2, "o_p", "u_same", "13800002222"),
	))
	// unionid 不同、手机号相同 → 不同人
	assert.False(t, SamePerson(
		user(1, "o_t", "u_a", "13800001111"),
		user(2, "o_p", "u_b", "13800001111"),
	))
}

// --- TC-NEW-ATH-0011 / 0012：手机号兜底 ---

// TC-NEW-ATH-0011 一端无 unionid（个人主体小程序拿不到）但手机号一致 → 同一人。
func TestSamePerson_PhoneFallbackWhenUnionIDMissing(t *testing.T) {
	teacher := user(1, "o_teacher_1", "", "13800001111") // 个人主体，无 unionid
	parent := user(2, "o_parent_1", "u_parent", "13800001111")

	assert.True(t, SamePerson(teacher, parent),
		"至少一侧无 unionid 时按手机号兜底，一致即同人")
}

// TC-NEW-ATH-0012 无 unionid 且手机号不一致或缺失 → 不同人 / 无法判定，不猜测。
func TestSamePerson_NoUnionIDPhoneMismatch(t *testing.T) {
	// 手机号不一致
	assert.False(t, SamePerson(
		user(1, "o_t", "", "13800001111"),
		user(2, "o_p", "", "13800002222"),
	))

	// 一侧手机号为空
	assert.False(t, SamePerson(
		user(1, "o_t", "", "13800001111"),
		user(2, "o_p", "", ""),
	))

	// 两侧都只有 openid，什么键都没有 → 无法判定
	assert.False(t, SamePerson(
		user(1, "o_t", "", ""),
		user(2, "o_p", "", ""),
	))
}

// nil 入参必须安全返回 false。
func TestSamePerson_NilSafe(t *testing.T) {
	assert.False(t, SamePerson(nil, user(1, "o", "u", "p")))
	assert.False(t, SamePerson(user(1, "o", "u", "p"), nil))
	assert.False(t, SamePerson(nil, nil))
}

// --- TC-NEW-ATH-0014：关联不修改账号归属 ---

// 关联判定是纯函数，不得有任何副作用。
func TestSamePerson_DoesNotMutateAccounts(t *testing.T) {
	a := user(1, "o_teacher_1", "u_same", "13800001111")
	b := user(2, "o_parent_1", "u_same", "13800002222")
	aCopy := *a
	bCopy := *b

	assert.True(t, SamePerson(a, b))

	assert.Equal(t, aCopy, *a, "判定不得修改任一侧记录")
	assert.Equal(t, bCopy, *b, "判定不得修改任一侧记录")
}

// 关联建立后两端登录仍各自命中本端 User，角色/孩子绑定不变。
func TestFindLinkedUser_DoesNotChangeLoginHit(t *testing.T) {
	repo := newFakeUserRepo()
	s := newAuthServiceForLogin(repo, &fakeInviteRepo{})
	ctx := context.Background()

	teacher := user(1, "o_teacher_1", "u_same", "13800001111")
	parent := user(2, "o_parent_1", "u_same", "13800002222")
	repo.index(teacher)
	repo.index(parent)

	linked, err := s.FindLinkedUser(ctx, teacher)
	require.NoError(t, err)
	require.NotNil(t, linked, "unionid 相同应查到另一端账号")
	assert.Equal(t, parent.ID, linked.ID)

	// 关联只读：两条记录都没被改动
	assert.Equal(t, "o_teacher_1", repo.byOpenID["o_teacher_1"].OpenID)
	assert.Equal(t, "o_parent_1", repo.byOpenID["o_parent_1"].OpenID)

	// 各自再登录仍命中本端
	got, err := s.resolveWeChatUser(ctx, &auth.Code2SessionResp{OpenID: "o_teacher_1"})
	require.NoError(t, err)
	assert.Equal(t, teacher.ID, got.ID, "老师端登录仍命中老师端那条")

	got, err = s.resolveWeChatUser(ctx, &auth.Code2SessionResp{OpenID: "o_parent_1"})
	require.NoError(t, err)
	assert.Equal(t, parent.ID, got.ID, "家长端登录仍命中家长端那条")
}

// --- FindLinkedUser 查找 ---

func TestFindLinkedUser_ByUnionID(t *testing.T) {
	repo := newFakeUserRepo()
	s := newAuthServiceForLogin(repo, &fakeInviteRepo{})
	ctx := context.Background()

	teacher := user(1, "o_teacher_1", "u_same", "")
	parent := user(2, "o_parent_1", "u_same", "")
	repo.index(teacher)
	repo.index(parent)

	got, err := s.FindLinkedUser(ctx, teacher)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, parent.ID, got.ID)

	// 反向也能查到
	got, err = s.FindLinkedUser(ctx, parent)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, teacher.ID, got.ID)
}

func TestFindLinkedUser_ByPhoneWhenNoUnionID(t *testing.T) {
	repo := newFakeUserRepo()
	s := newAuthServiceForLogin(repo, &fakeInviteRepo{})
	ctx := context.Background()

	teacher := user(1, "o_teacher_1", "", "13800001111") // 个人主体，无 unionid
	parent := user(2, "o_parent_1", "u_p", "13800001111")
	repo.index(teacher)
	repo.index(parent)

	got, err := s.FindLinkedUser(ctx, teacher)
	require.NoError(t, err)
	require.NotNil(t, got, "unionid 缺失时应按手机号兜底")
	assert.Equal(t, parent.ID, got.ID)
}

func TestFindLinkedUser_NoMatch(t *testing.T) {
	repo := newFakeUserRepo()
	s := newAuthServiceForLogin(repo, &fakeInviteRepo{})
	ctx := context.Background()

	alone := user(1, "o_alone", "", "")
	repo.index(alone)

	got, err := s.FindLinkedUser(ctx, alone)
	require.NoError(t, err)
	assert.Nil(t, got, "没有任何关联键时不得猜测关联")
}

// 查到自己不算关联——那是同一条记录，不是「另一端的账号」。
func TestFindLinkedUser_SelfIsNotLinked(t *testing.T) {
	repo := newFakeUserRepo()
	s := newAuthServiceForLogin(repo, &fakeInviteRepo{})
	ctx := context.Background()

	only := user(1, "o_only", "u_only", "13800001111")
	repo.index(only)

	got, err := s.FindLinkedUser(ctx, only)
	require.NoError(t, err)
	assert.Nil(t, got, "只有自己时不得返回自己")
}

// unionid 优先于手机号：能用 unionid 查到就不再看手机号。
func TestFindLinkedUser_UnionIDPreferred(t *testing.T) {
	repo := newFakeUserRepo()
	s := newAuthServiceForLogin(repo, &fakeInviteRepo{})
	ctx := context.Background()

	// 同手机号但不同 unionid：unionid 不同就是不同人，不得按手机号误关联
	teacher := user(1, "o_t", "u_a", "13800001111")
	samePhoneOther := user(2, "o_p", "u_b", "13800001111")
	repo.index(teacher)
	repo.index(samePhoneOther)

	got, err := s.FindLinkedUser(ctx, teacher)
	require.NoError(t, err)
	// unionid=u_a 查不到别人 → 返回 nil，不得退化成按手机号把 u_b 那条捞出来
	assert.Nil(t, got, "unionid 都非空但不同，不得按手机号误关联")
}
