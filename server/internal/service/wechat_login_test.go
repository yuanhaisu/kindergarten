package service

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/domain"
)

// --- mocks ---

type fakeUserRepo struct {
	byOpenID map[string]*domain.User
	byUnion  map[string]*domain.User
	created  []*domain.User
	updated  []*domain.User
	nextID   uint64
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		byOpenID: map[string]*domain.User{},
		byUnion:  map[string]*domain.User{},
		nextID:   1,
	}
}

func (f *fakeUserRepo) FindByUnionID(ctx context.Context, unionID string) (*domain.User, error) {
	if unionID == "" {
		return nil, nil
	}
	return f.byUnion[unionID], nil
}

// FindOtherByUnionID / FindOtherByPhone 必须排除自己——同一键下有两条 User，
// 不排除就会命中自己那条、查不到另一端。
func (f *fakeUserRepo) FindOtherByUnionID(ctx context.Context, unionID string, excludeID uint64) (*domain.User, error) {
	if unionID == "" {
		return nil, nil
	}
	for _, u := range f.allByID() {
		if u.UnionID == unionID && u.ID != excludeID {
			return u, nil
		}
	}
	return nil, nil
}

func (f *fakeUserRepo) FindOtherByPhone(ctx context.Context, phone string, excludeID uint64) (*domain.User, error) {
	if phone == "" {
		return nil, nil
	}
	for _, u := range f.allByID() {
		if u.Phone == phone && u.ID != excludeID {
			return u, nil
		}
	}
	return nil, nil
}

// allByID 按 id 升序返回，避免 map 迭代顺序导致用例不稳定。
func (f *fakeUserRepo) allByID() []*domain.User {
	seen := map[uint64]bool{}
	var out []*domain.User
	for _, u := range f.byOpenID {
		if !seen[u.ID] {
			seen[u.ID] = true
			out = append(out, u)
		}
	}
	for _, u := range f.byUnion {
		if !seen[u.ID] {
			seen[u.ID] = true
			out = append(out, u)
		}
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].ID < out[i].ID {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

func (f *fakeUserRepo) FindByPhone(ctx context.Context, phone string) (*domain.User, error) {
	for _, u := range f.byOpenID {
		if u.Phone == phone {
			return u, nil
		}
	}
	return nil, nil
}

func (f *fakeUserRepo) FindByOpenID(ctx context.Context, openID string) (*domain.User, error) {
	return f.byOpenID[openID], nil
}

func (f *fakeUserRepo) FindByID(ctx context.Context, id uint64) (*domain.User, error) {
	for _, u := range f.byOpenID {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, nil
}

func (f *fakeUserRepo) index(u *domain.User) {
	if u.OpenID != "" {
		f.byOpenID[u.OpenID] = u
	}
	if u.UnionID != "" {
		f.byUnion[u.UnionID] = u
	}
}

func (f *fakeUserRepo) Create(ctx context.Context, user *domain.User) error {
	user.ID = f.nextID
	f.nextID++
	f.index(user)
	f.created = append(f.created, user)
	return nil
}

func (f *fakeUserRepo) Update(ctx context.Context, user *domain.User) error {
	f.index(user)
	f.updated = append(f.updated, user)
	return nil
}

func (f *fakeUserRepo) UpdateFields(ctx context.Context, id uint64, fields map[string]any) error {
	return nil
}

type fakeInviteRepo struct {
	byToken map[string]*domain.TeacherInvitation
	marked  []uint64
}

func (f *fakeInviteRepo) FindByToken(ctx context.Context, token string) (*domain.TeacherInvitation, error) {
	return f.byToken[token], nil
}

func (f *fakeInviteRepo) MarkUsed(ctx context.Context, id, userID uint64, openID, unionID string) error {
	f.marked = append(f.marked, id)
	return nil
}

type fakeRoleRepo struct{}

func (f *fakeRoleRepo) FindByUserID(ctx context.Context, userID uint64) ([]domain.UserRole, error) {
	return nil, nil
}

// deadRedis 指向不存在的地址：SaveRefresh 会返回连接错误而不是 panic。
// 本组用例只验身份落库 / 邀请码重绑，签发令牌（需 Redis）不在范围内。
func deadRedis() *redis.Client {
	return redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", MaxRetries: 0, DialTimeout: 50 * time.Millisecond})
}

func newAuthServiceForLogin(users *fakeUserRepo, invites *fakeInviteRepo) *AuthService {
	return &AuthService{
		userRepo:       users,
		userRoleRepo:   &fakeRoleRepo{},
		invitationRepo: invites,
		rdb:            deadRedis(),
		secret:         []byte("test-secret"),
		atTTL:          time.Minute,
		rtTTL:          time.Hour,
	}
}

// --- TC-NEW-ATH-0001 ~ 0004：按端落独立 User ---

// TC-NEW-ATH-0001 老师端首次微信登录：本端 openid 未命中 → 新建一条 User。
func TestWeChatUser_TeacherFirstLoginCreatesUser(t *testing.T) {
	repo := newFakeUserRepo()
	s := newAuthServiceForLogin(repo, &fakeInviteRepo{})

	user, err := s.resolveWeChatUser(context.Background(), &auth.Code2SessionResp{OpenID: "o_teacher_1", UnionID: "u_same"})
	require.NoError(t, err)
	require.NotNil(t, user)

	assert.Equal(t, "o_teacher_1", user.OpenID)
	assert.Len(t, repo.created, 1, "首次登录应新建一条 User")
}

// TC-NEW-ATH-0002 老师端再次登录：命中本端已有 User，不新建。
func TestWeChatUser_TeacherSecondLoginHitsSameUser(t *testing.T) {
	repo := newFakeUserRepo()
	s := newAuthServiceForLogin(repo, &fakeInviteRepo{})

	first, err := s.resolveWeChatUser(context.Background(), &auth.Code2SessionResp{OpenID: "o_teacher_1"})
	require.NoError(t, err)

	second, err := s.resolveWeChatUser(context.Background(), &auth.Code2SessionResp{OpenID: "o_teacher_1"})
	require.NoError(t, err)

	assert.Equal(t, first.ID, second.ID, "再次登录必须命中同一条 User")
	assert.Len(t, repo.created, 1, "第二次登录不得再新建")
}

// TC-NEW-ATH-0003 家长端首次登录：本端 openid 未命中 → 新建一条 User。
func TestWeChatUser_ParentFirstLoginCreatesUser(t *testing.T) {
	repo := newFakeUserRepo()
	s := newAuthServiceForLogin(repo, &fakeInviteRepo{})

	user, err := s.resolveWeChatUser(context.Background(), &auth.Code2SessionResp{OpenID: "o_parent_1", UnionID: "u_same"})
	require.NoError(t, err)

	assert.Equal(t, "o_parent_1", user.OpenID)
	assert.Len(t, repo.created, 1)
}

// TC-NEW-ATH-0004 同一自然人换端登录：两端各落一条，互不命中。
//
// 关键：即使 unionid 相同（同一个人），也不得合并——openid 不同就必须是两条 User。
func TestWeChatUser_SameHumanBothEndsGetsTwoUsers(t *testing.T) {
	repo := newFakeUserRepo()
	s := newAuthServiceForLogin(repo, &fakeInviteRepo{})
	ctx := context.Background()

	teacherUser, err := s.resolveWeChatUser(ctx, &auth.Code2SessionResp{OpenID: "o_teacher_1", UnionID: "u_same_human"})
	require.NoError(t, err)

	parentUser, err := s.resolveWeChatUser(ctx, &auth.Code2SessionResp{OpenID: "o_parent_1", UnionID: "u_same_human"})
	require.NoError(t, err)

	assert.NotEqual(t, teacherUser.ID, parentUser.ID,
		"同一个人两端必须是两条 User，不得因 unionid 相同而合并")
	assert.Len(t, repo.created, 2, "两端各新建一条")
	assert.Equal(t, "o_teacher_1", teacherUser.OpenID)
	assert.Equal(t, "o_parent_1", parentUser.OpenID)

	// 各自再次登录仍命中本端那条
	againTeacher, err := s.resolveWeChatUser(ctx, &auth.Code2SessionResp{OpenID: "o_teacher_1", UnionID: "u_same_human"})
	require.NoError(t, err)
	assert.Equal(t, teacherUser.ID, againTeacher.ID, "老师端再登录必须命中老师端那条")

	againParent, err := s.resolveWeChatUser(ctx, &auth.Code2SessionResp{OpenID: "o_parent_1", UnionID: "u_same_human"})
	require.NoError(t, err)
	assert.Equal(t, parentUser.ID, againParent.ID, "家长端再登录必须命中家长端那条")
}

// openid 变化时不得沿用旧账号 —— 切换 AppID 后旧 openid 全失效，
// 会走「未命中 → 新建」路径，这正是需要邀请码重绑的原因。
func TestWeChatUser_OpenIDChangeCreatesNewUser(t *testing.T) {
	repo := newFakeUserRepo()
	s := newAuthServiceForLogin(repo, &fakeInviteRepo{})
	ctx := context.Background()

	_, err := s.resolveWeChatUser(ctx, &auth.Code2SessionResp{OpenID: "o_old_appid"})
	require.NoError(t, err)

	_, err = s.resolveWeChatUser(ctx, &auth.Code2SessionResp{OpenID: "o_new_appid"})
	require.NoError(t, err)

	assert.Len(t, repo.created, 2, "openid 变了就是新身份，不得命中旧账号")
}

// --- TC-NEW-ATH-0008：邀请码无效或已使用 ---

func TestWeChatLoginWithInvite_InvalidCode(t *testing.T) {
	users := newFakeUserRepo()
	invites := &fakeInviteRepo{byToken: map[string]*domain.TeacherInvitation{}}
	s := newAuthServiceForLogin(users, invites)
	ctx := context.Background()

	_, err := s.weChatLoginWithInvite(ctx, &auth.Code2SessionResp{OpenID: "o_teacher_1"}, "no-such-token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "INVITE_CODE_INVALID")
	assert.Empty(t, users.created, "邀请码无效不得新建 User")
}

func TestWeChatLoginWithInvite_MissingInvite(t *testing.T) {
	users := newFakeUserRepo()
	invites := &fakeInviteRepo{byToken: map[string]*domain.TeacherInvitation{
		// 已使用的邀请码不返回
	}}
	s := newAuthServiceForLogin(users, invites)

	_, err := s.weChatLoginWithInvite(context.Background(), &auth.Code2SessionResp{OpenID: "o_teacher_1"}, "used-token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "INVITE_CODE_INVALID")
}

// TC-NEW-ATH-0007 的绑定写入部分：有效邀请码把新 openid 写进预创建的教师账号。
// （签发令牌需要 Redis，属 E2E，不在本用例范围。）
func TestWeChatLoginWithInvite_RebindsOpenIDToPreCreatedTeacher(t *testing.T) {
	preCreated := &domain.User{BaseModel: domain.BaseModel{ID: 42}, Phone: "13800001111"}
	users := newFakeUserRepo()
	users.byOpenID["o_teacher_old"] = preCreated
	users.nextID = 100

	invites := &fakeInviteRepo{byToken: map[string]*domain.TeacherInvitation{
		"tok": {BaseModel: domain.BaseModel{ID: 7}, UserID: 42, Status: domain.TeacherInvitationPending},
	}}
	s := newAuthServiceForLogin(users, invites)

	_, err := s.weChatLoginWithInvite(context.Background(), &auth.Code2SessionResp{OpenID: "o_teacher_new"}, "tok")

	// 令牌签发会因无 Redis 失败，但重绑必须已经完成
	if err != nil {
		assert.NotContains(t, err.Error(), "INVITE", "重绑本身不应报邀请错误")
	}
	require.Len(t, users.updated, 1, "必须把新 openid 写回预创建的教师账号")
	assert.Equal(t, "o_teacher_new", users.updated[0].OpenID)
	assert.Equal(t, uint64(42), users.updated[0].ID, "不得新建 User，只能更新预创建那条")
	assert.Equal(t, "13800001111", users.updated[0].Phone, "原有手机号保持不变")
	assert.Empty(t, users.created, "邀请码路径不得新建 User")
	assert.Equal(t, []uint64{7}, invites.marked, "邀请码应被标记为已使用")
}
