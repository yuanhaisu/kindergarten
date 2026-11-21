package authharness_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/service"
	"kindergarten-server/internal/testutil/authharness"
)

// #168：老师邀请绑定。
//
// 这里只通过 HTTP 缝观察行为：登录拿到待绑定 token → bind-invite → 换到园所
// 预创建教师账号的 token。数据库断言只用于验证「预创建账号保留、待绑定账号
// 硬删、身份与邀请记录指向预创建账号」这些无法从响应体完全看出的结果。

type teacherBindFixture struct {
	AppID       string
	OpenID      string
	KG          *domain.Kindergarten
	Class       *domain.Class
	Teacher     *domain.User
	Student     *domain.Student
	Invitation  *domain.TeacherInvitation
	PendingID   uint64
	PendingTok  string
	InviteToken string
}

func TestTeacherBindInviteMergesIntoPreCreatedAccount(t *testing.T) {
	h := authharness.New(t)
	f := setupTeacherBind(t, h, teacherBindOptions{})

	resp := h.PostJSONWithToken(t, "/api/v1/common/user/bind-invite", f.PendingTok, map[string]string{
		"invite_code": f.InviteToken,
	})
	require.Equal(t, 200, resp.Status, string(resp.Body))

	var result service.LoginResult
	resp.DecodeData(t, &result)
	require.NotEmpty(t, result.Token)
	require.NotNil(t, result.User)
	assert.Equal(t, f.Teacher.ID, result.User.ID, "绑定后必须换到园所预创建账号")

	claims, err := auth.ParseAccess(h.Secret, result.Token)
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%d", f.Teacher.ID), claims.UserID)
	assert.Equal(t, f.AppID, claims.AppID)
	assert.Equal(t, []string{domain.ClientRoleTeacher}, claims.Roles)
	assert.Equal(t, f.KG.ID, claims.KindergartenID)

	require.Len(t, result.User.Roles, 1, "绑定后 token 响应只含老师角色")
	assert.Equal(t, domain.ClientRoleTeacher, result.User.Roles[0].Role)
	assert.Equal(t, f.KG.ID, *result.User.Roles[0].KindergartenID)
	require.NotNil(t, result.User.Roles[0].ClassID)
	assert.Equal(t, f.Class.ID, *result.User.Roles[0].ClassID)
	assert.Equal(t, f.Class.Name, result.User.Roles[0].ClassName)

	var invitation domain.TeacherInvitation
	require.NoError(t, h.DB.First(&invitation, f.Invitation.ID).Error)
	assert.Equal(t, int8(domain.TeacherInvitationUsed), invitation.Status)
	assert.Equal(t, f.Teacher.ID, invitation.UserID, "邀请记录必须指向预创建账号")

	var identity domain.UserIdentity
	require.NoError(t, h.DB.Where("app_id = ? AND open_id = ?", f.AppID, f.OpenID).First(&identity).Error)
	assert.Equal(t, f.Teacher.ID, identity.UserID, "客户端身份必须移到预创建账号")

	var pending domain.User
	err = h.DB.Unscoped().First(&pending, f.PendingID).Error
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "待绑定账号必须硬删除")

	assertRowCount(t, h, &domain.UserRole{}, 1, "user_id = ?", f.Teacher.ID)
	assertRowCount(t, h, &domain.TeacherProfile{}, 1, "user_id = ?", f.Teacher.ID)
	assertRowCount(t, h, &domain.ClassTeacher{}, 1, "user_id = ?", f.Teacher.ID)
	assertRowCount(t, h, &domain.StudentParent{}, 1, "user_id = ?", f.Teacher.ID)
}

func TestTeacherBindInviteRejectsExpiredInvitation(t *testing.T) {
	h := authharness.New(t)
	f := setupTeacherBind(t, h, teacherBindOptions{ExpiresAt: time.Now().Add(-time.Hour)})

	resp := h.PostJSONWithToken(t, "/api/v1/common/user/bind-invite", f.PendingTok, map[string]string{
		"invite_code": f.InviteToken,
	})
	require.Equal(t, 400, resp.Status, string(resp.Body))
	assert.Equal(t, 40020, resp.EnvelopeCode(t), "过期邀请码必须按无效处理")

	var invitation domain.TeacherInvitation
	require.NoError(t, h.DB.First(&invitation, f.Invitation.ID).Error)
	assert.Equal(t, int8(domain.TeacherInvitationPending), invitation.Status, "绑定失败不能消耗邀请码")

	var identity domain.UserIdentity
	require.NoError(t, h.DB.Where("app_id = ? AND open_id = ?", f.AppID, f.OpenID).First(&identity).Error)
	assert.Equal(t, f.PendingID, identity.UserID, "绑定失败不能移动身份")

	var pending domain.User
	require.NoError(t, h.DB.First(&pending, f.PendingID).Error, "绑定失败不能删除待绑定账号")
}

func TestTeacherBindInviteRejectsUsedInvitation(t *testing.T) {
	h := authharness.New(t)
	f := setupTeacherBind(t, h, teacherBindOptions{InvitationStatus: int8(domain.TeacherInvitationUsed)})

	resp := h.PostJSONWithToken(t, "/api/v1/common/user/bind-invite", f.PendingTok, map[string]string{
		"invite_code": f.InviteToken,
	})
	require.Equal(t, 400, resp.Status, string(resp.Body))
	assert.Equal(t, 40021, resp.EnvelopeCode(t), "已使用的邀请码必须与无效邀请码区分")

	var identity domain.UserIdentity
	require.NoError(t, h.DB.Where("app_id = ? AND open_id = ?", f.AppID, f.OpenID).First(&identity).Error)
	assert.Equal(t, f.PendingID, identity.UserID, "绑定失败不能移动身份")
}

func TestTeacherBindInviteRejectsNonTeacherClient(t *testing.T) {
	h := authharness.New(t)
	f := setupTeacherBind(t, h, teacherBindOptions{ClientRole: domain.ClientRoleParent})

	resp := h.PostJSONWithToken(t, "/api/v1/common/user/bind-invite", f.PendingTok, map[string]string{
		"invite_code": f.InviteToken,
	})
	require.Equal(t, 403, resp.Status, string(resp.Body))
	assert.Equal(t, 40300, resp.EnvelopeCode(t), "家长端不能使用老师邀请码")

	var invitation domain.TeacherInvitation
	require.NoError(t, h.DB.First(&invitation, f.Invitation.ID).Error)
	assert.Equal(t, int8(domain.TeacherInvitationPending), invitation.Status)
}

func TestTeacherBindInviteRejectsClientChannelMismatch(t *testing.T) {
	h := authharness.New(t)
	// 共用老师端不能绑定专属园所（#177）：园所通道与客户端作用域必须匹配。
	f := setupTeacherBind(t, h, teacherBindOptions{AppChannel: domain.AppChannelDedicated})

	resp := h.PostJSONWithToken(t, "/api/v1/common/user/bind-invite", f.PendingTok, map[string]string{
		"invite_code": f.InviteToken,
	})
	require.Equal(t, 403, resp.Status, string(resp.Body))
	assert.Equal(t, 40300, resp.EnvelopeCode(t))

	var identity domain.UserIdentity
	require.NoError(t, h.DB.Where("app_id = ? AND open_id = ?", f.AppID, f.OpenID).First(&identity).Error)
	assert.Equal(t, f.PendingID, identity.UserID, "绑定失败不能移动身份")
}

func TestTeacherBindInviteRejectsSecondWeChatForSameApp(t *testing.T) {
	h := authharness.New(t)
	f := setupTeacherBind(t, h, teacherBindOptions{})

	// 预创建教师账号已经在这个老师端绑了另一个微信：后到的 openid 不能顶掉前者。
	boundOpenID := uniqueOpenID("bind-invite-bound")
	require.NoError(t, h.DB.Create(&domain.UserIdentity{
		UserID: f.Teacher.ID,
		AppID:  f.AppID,
		OpenID: boundOpenID,
	}).Error)
	t.Cleanup(func() {
		h.HardDelete(t, &domain.UserIdentity{}, "app_id = ? AND open_id = ?", f.AppID, boundOpenID)
	})

	resp := h.PostJSONWithToken(t, "/api/v1/common/user/bind-invite", f.PendingTok, map[string]string{
		"invite_code": f.InviteToken,
	})
	require.Equal(t, 409, resp.Status, string(resp.Body))
	assert.Equal(t, 40900, resp.EnvelopeCode(t))

	var newIdentity domain.UserIdentity
	require.NoError(t, h.DB.Where("app_id = ? AND open_id = ?", f.AppID, f.OpenID).First(&newIdentity).Error)
	assert.Equal(t, f.PendingID, newIdentity.UserID, "第二个微信必须保持在待绑定账号上")

	var invitation domain.TeacherInvitation
	require.NoError(t, h.DB.First(&invitation, f.Invitation.ID).Error)
	assert.Equal(t, int8(domain.TeacherInvitationPending), invitation.Status, "冲突时不能消耗邀请码")
}

// 同一自然人既是家长又是老师时必须落在同一个账号：手机号唯一索引保证不会
// 出现两个 User，绑定只是在既有账号上补一条老师客户端身份与老师角色。
func TestTeacherBindInviteSharesAccountWithSamePhoneParent(t *testing.T) {
	h := authharness.New(t)

	phone := uniquePhone()
	account := &domain.User{Phone: phone, Nickname: "既是家长又是老师"}
	require.NoError(t, h.DB.Create(account).Error)

	f := setupTeacherBind(t, h, teacherBindOptions{Teacher: account})

	// 家长侧既有数据：家长角色、家长档案、家长端客户端身份。
	parentApp := uniqueAppID("bind-invite-parent")
	parentOpenID := uniqueOpenID("bind-invite-parent")
	kgID := f.KG.ID
	require.NoError(t, h.DB.Create(&domain.UserRole{
		UserID:         account.ID,
		Role:           domain.ClientRoleParent,
		KindergartenID: &kgID,
	}).Error)
	require.NoError(t, h.DB.Create(&domain.ParentProfile{
		UserID:         account.ID,
		KindergartenID: kgID,
		RealName:       "家长实名",
		Status:         1,
	}).Error)
	require.NoError(t, h.DB.Create(&domain.UserIdentity{
		UserID: account.ID,
		AppID:  parentApp,
		OpenID: parentOpenID,
	}).Error)
	t.Cleanup(func() {
		h.HardDelete(t, &domain.ParentProfile{}, "user_id = ?", account.ID)
		h.HardDelete(t, &domain.UserIdentity{}, "app_id = ? AND open_id = ?", parentApp, parentOpenID)
	})

	resp := h.PostJSONWithToken(t, "/api/v1/common/user/bind-invite", f.PendingTok, map[string]string{
		"invite_code": f.InviteToken,
	})
	require.Equal(t, 200, resp.Status, string(resp.Body))

	var result service.LoginResult
	resp.DecodeData(t, &result)
	require.NotNil(t, result.User)
	assert.Equal(t, account.ID, result.User.ID, "绑定后必须是同一个自然人账号")
	require.Len(t, result.User.Roles, 1, "老师端 token 只携带老师角色")
	assert.Equal(t, domain.ClientRoleTeacher, result.User.Roles[0].Role)

	claims, err := auth.ParseAccess(h.Secret, result.Token)
	require.NoError(t, err)
	assert.Equal(t, []string{domain.ClientRoleTeacher}, claims.Roles, "老师端 token 不能混入家长角色")

	var userCount int64
	require.NoError(t, h.DB.Model(&domain.User{}).Where("phone = ?", phone).Count(&userCount).Error)
	assert.Equal(t, int64(1), userCount, "同一手机号只能有一个账号")
	assertRowCount(t, h, &domain.UserRole{}, 2, "user_id = ?", account.ID)

	var parentIdentity domain.UserIdentity
	require.NoError(t, h.DB.Where("app_id = ? AND open_id = ?", parentApp, parentOpenID).First(&parentIdentity).Error)
	assert.Equal(t, account.ID, parentIdentity.UserID, "家长端身份必须保留在同一账号")

	var pending domain.User
	err = h.DB.Unscoped().First(&pending, f.PendingID).Error
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "待绑定账号必须硬删除")
}

// 绑定入口限流（#174）：app_id + IP + 邀请码，20 次/分钟，失败开放。
func TestTeacherBindInviteRateLimited(t *testing.T) {
	h := authharness.New(t)
	f := setupTeacherBind(t, h, teacherBindOptions{})

	const path = "/api/v1/common/user/bind-invite"
	payload := map[string]string{"invite_code": "not-a-real-invite"}
	for i := 0; i < 20; i++ {
		resp := h.PostJSONWithToken(t, path, f.PendingTok, payload)
		require.Equal(t, 400, resp.Status, "第 %d 次请求应通过限流；body=%s", i+1, resp.Body)
	}

	resp := h.PostJSONWithToken(t, path, f.PendingTok, payload)
	require.Equal(t, 429, resp.Status, string(resp.Body))
	assert.Equal(t, 42900, resp.EnvelopeCode(t))
}

// --- fixture ---

type teacherBindOptions struct {
	// 空值表示使用 shared 作用域的 shared 园所。
	ScopeKind  string
	ScopeID    uint64
	AppChannel string
	// ClientRole 客户端注册角色，空值默认 teacher。
	ClientRole string
	// Teacher 复用已存在的自然人账号作为园所预创建教师（同手机号家长/老师场景）。
	// 空值则新建一个带唯一手机号的教师账号。
	Teacher          *domain.User
	InvitationStatus int8
	ExpiresAt        time.Time
}

func setupTeacherBind(t *testing.T, h *authharness.Harness, opts teacherBindOptions) *teacherBindFixture {
	t.Helper()

	appID := uniqueAppID("bind-invite")
	openID := uniqueOpenID("bind-invite")
	code := uniqueCode("bind-invite")
	h.WeChat.StubCode2Session(code, openID, "")

	kg := &domain.Kindergarten{Name: fmt.Sprintf("邀请绑定园所-%d", time.Now().UnixNano())}
	if opts.AppChannel != "" {
		kg.AppChannel = opts.AppChannel
	}
	require.NoError(t, h.DB.Create(kg).Error)
	class := &domain.Class{KindergartenID: kg.ID, Name: "小一班"}
	require.NoError(t, h.DB.Create(class).Error)
	teacher := opts.Teacher
	if teacher == nil {
		teacher = &domain.User{Phone: uniquePhone(), Nickname: "预创建老师"}
		require.NoError(t, h.DB.Create(teacher).Error)
	}
	seedTeacherRelations(t, h, kg, class, teacher)

	inviteToken := fmt.Sprintf("invite-168-%d", time.Now().UnixNano())
	status := opts.InvitationStatus
	if status == 0 {
		status = domain.TeacherInvitationPending
	}
	expiresAt := opts.ExpiresAt
	if expiresAt.IsZero() {
		expiresAt = time.Now().Add(7 * 24 * time.Hour)
	}
	invitation := &domain.TeacherInvitation{
		Token:          inviteToken,
		UserID:         teacher.ID,
		KindergartenID: kg.ID,
		Status:         status,
		ExpiresAt:      expiresAt,
	}
	require.NoError(t, h.DB.Create(invitation).Error)

	scopeKind := opts.ScopeKind
	if scopeKind == "" {
		scopeKind = domain.ClientScopeShared
	}
	clientRole := opts.ClientRole
	if clientRole == "" {
		clientRole = domain.ClientRoleTeacher
	}
	seedClientScoped(t, h, appID, clientRole, scopeKind, opts.ScopeID, domain.ClientStatusEnabled)

	login := loginOK(t, h, appID, code)

	t.Cleanup(func() {
		h.HardDelete(t, &domain.TeacherInvitation{}, "id = ?", invitation.ID)
		h.HardDelete(t, &domain.StudentParent{}, "user_id = ?", teacher.ID)
		h.HardDelete(t, &domain.StudentParent{}, "student_id IN (SELECT id FROM students WHERE class_id = ?)", class.ID)
		h.HardDelete(t, &domain.Student{}, "class_id = ?", class.ID)
		h.HardDelete(t, &domain.ClassTeacher{}, "user_id = ?", teacher.ID)
		h.HardDelete(t, &domain.TeacherProfile{}, "user_id = ?", teacher.ID)
		h.HardDelete(t, &domain.UserRole{}, "user_id = ?", teacher.ID)
		h.HardDelete(t, &domain.UserIdentity{}, "app_id = ? AND open_id = ?", appID, openID)
		h.HardDelete(t, &domain.User{}, "id IN ?", []uint64{teacher.ID, login.User.ID})
		h.HardDelete(t, &domain.Class{}, "id = ?", class.ID)
		h.HardDelete(t, &domain.Kindergarten{}, "id = ?", kg.ID)
	})

	return &teacherBindFixture{
		AppID:       appID,
		OpenID:      openID,
		KG:          kg,
		Class:       class,
		Teacher:     teacher,
		Invitation:  invitation,
		PendingID:   login.User.ID,
		PendingTok:  login.Token,
		InviteToken: inviteToken,
	}
}

// seedTeacherRelations 模拟园所后台「新增教师」：老师角色（带班）+ 教师档案 +
// 带班关系 + 一个孩子。绑定后这些都必须在预创建账号上保留。
func seedTeacherRelations(t *testing.T, h *authharness.Harness, kg *domain.Kindergarten, class *domain.Class, teacher *domain.User) {
	t.Helper()

	kgID := kg.ID
	classID := class.ID
	require.NoError(t, h.DB.Create(&domain.UserRole{
		UserID:           teacher.ID,
		Role:             domain.ClientRoleTeacher,
		KindergartenID:   &kgID,
		KindergartenName: kg.Name,
		ClassID:          &classID,
		ClassName:        class.Name,
	}).Error)
	require.NoError(t, h.DB.Create(&domain.TeacherProfile{
		UserID:         teacher.ID,
		KindergartenID: kg.ID,
		RealName:       "预创建老师",
		JoinDate:       time.Now(),
		Status:         1,
	}).Error)
	require.NoError(t, h.DB.Create(&domain.ClassTeacher{
		ClassID: class.ID,
		UserID:  teacher.ID,
		Title:   "主班",
		Status:  1,
	}).Error)

	student := &domain.Student{
		KindergartenID: kg.ID,
		ClassID:        class.ID,
		Name:           "小明",
		Birthday:       time.Now().AddDate(-4, 0, 0),
		EnrollDate:     time.Now(),
		Status:         1,
	}
	require.NoError(t, h.DB.Create(student).Error)
	require.NoError(t, h.DB.Create(&domain.StudentParent{
		StudentID: student.ID,
		UserID:    teacher.ID,
		Relation:  domain.RelationFather,
		IsPrimary: true,
	}).Error)
	t.Cleanup(func() {
		h.HardDelete(t, &domain.StudentParent{}, "student_id = ?", student.ID)
		h.HardDelete(t, &domain.Student{}, "id = ?", student.ID)
	})
}

func seedClientScoped(t *testing.T, h *authharness.Harness, appID, role, scopeKind string, scopeID uint64, status string) *domain.Client {
	t.Helper()
	client := &domain.Client{
		AppID:     appID,
		Role:      role,
		ScopeKind: scopeKind,
		ScopeID:   scopeID,
		Status:    status,
	}
	require.NoError(t, h.DB.Create(client).Error)
	t.Cleanup(func() { h.HardDelete(t, &domain.Client{}, "app_id = ?", appID) })
	return client
}

func assertRowCount(t *testing.T, h *authharness.Harness, model any, want int64, where string, args ...any) {
	t.Helper()
	var count int64
	require.NoError(t, h.DB.Model(model).Where(where, args...).Count(&count).Error)
	assert.Equal(t, want, count, "行数不符：%T where %s", model, where)
}
