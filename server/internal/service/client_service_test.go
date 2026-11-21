package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/internal/service"
	"kindergarten-server/internal/testutil/authharness"
)

// 小程序客户端注册表（#163）的行为测试。
//
// 断言只看外部行为：clients 表里的行与 SyncFromConfig 的返回。测试库由
// authharness 派生并执行真实 goose 迁移，因此这里同时验证 00012 迁移与
// app_id 唯一索引确实存在。
//
// 用外部测试包（service_test）导入 authharness：后者依赖 service，放在
// package service 内会形成 import cycle。

const (
	teacherAppID163 = "wx_teacher_163_test"
	parentAppID163  = "wx_parent_163_test"
)

func newClientService(t *testing.T) (*authharness.Harness, *service.ClientService) {
	t.Helper()
	h := authharness.New(t)
	h.HardDelete(t, &domain.Client{}, "1 = 1")
	return h, service.NewClientService(repository.NewClientRepo(h.DB))
}

func sharedSpecs(teacherAppID, parentAppID string) []service.ClientSpec {
	return []service.ClientSpec{
		{
			AppID:          teacherAppID,
			Role:           domain.ClientRoleTeacher,
			ScopeKind:      domain.ClientScopeShared,
			ReleaseVersion: "0.1.0",
			CredentialRef:  "wechat.teacher",
		},
		{
			AppID:          parentAppID,
			Role:           domain.ClientRoleParent,
			ScopeKind:      domain.ClientScopeShared,
			ReleaseVersion: "0.1.0",
			CredentialRef:  "wechat.parent",
		},
	}
}

func listClients(t *testing.T, h *authharness.Harness) []domain.Client {
	t.Helper()
	var rows []domain.Client
	require.NoError(t, h.DB.Order("id").Find(&rows).Error, "读取 clients 失败")
	return rows
}

// 配置同步首次运行：两条 shared 客户端落库，状态 enabled，版本与凭据引用来自配置。
func TestClientSyncSeedsTwoSharedClients(t *testing.T) {
	h, svc := newClientService(t)

	result, err := svc.SyncFromConfig(context.Background(), sharedSpecs(teacherAppID163, parentAppID163))
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.ElementsMatch(t, []string{teacherAppID163, parentAppID163}, result.Created)

	rows := listClients(t, h)
	require.Len(t, rows, 2)

	byRole := map[string]domain.Client{}
	for _, row := range rows {
		byRole[row.Role] = row
	}

	teacher, ok := byRole[domain.ClientRoleTeacher]
	require.True(t, ok, "缺少老师端客户端")
	assert.Equal(t, teacherAppID163, teacher.AppID)
	assert.Equal(t, domain.ClientScopeShared, teacher.ScopeKind)
	assert.Equal(t, uint64(0), teacher.ScopeID, "shared 作用域的 scope_id 必须是 0")
	assert.Equal(t, domain.ClientStatusEnabled, teacher.Status)
	assert.Equal(t, "0.1.0", teacher.ReleaseVersion)
	assert.Equal(t, "wechat.teacher", teacher.CredentialRef)

	parent, ok := byRole[domain.ClientRoleParent]
	require.True(t, ok, "缺少家长端客户端")
	assert.Equal(t, parentAppID163, parent.AppID)
	assert.Equal(t, domain.ClientStatusEnabled, parent.Status)
}

// 重复启动不能产生重复行，也不应把已有行当成新建。
func TestClientSyncIsIdempotent(t *testing.T) {
	h, svc := newClientService(t)

	first, err := svc.SyncFromConfig(context.Background(), sharedSpecs(teacherAppID163, parentAppID163))
	require.NoError(t, err)
	require.Len(t, first.Created, 2)

	second, err := svc.SyncFromConfig(context.Background(), sharedSpecs(teacherAppID163, parentAppID163))
	require.NoError(t, err)
	assert.Empty(t, second.Created, "重复同步不得重复插入")
	assert.ElementsMatch(t, []string{teacherAppID163, parentAppID163}, second.Unchanged)
	assert.Len(t, listClients(t, h), 2)
}

// status 是运行时唯一来源：运维在 DB 停用客户端后，重启同步不得把它重新启用。
// release_version 同理——配置只在首次插入时 seed，之后由 DB 维护。
func TestClientSyncDoesNotOverwriteStatusOrReleaseVersion(t *testing.T) {
	h, svc := newClientService(t)
	_, err := svc.SyncFromConfig(context.Background(), sharedSpecs(teacherAppID163, parentAppID163))
	require.NoError(t, err)

	var teacher domain.Client
	require.NoError(t, h.DB.Where("app_id = ?", teacherAppID163).First(&teacher).Error)
	require.NoError(t, h.DB.Model(&domain.Client{}).Where("id = ?", teacher.ID).Updates(map[string]any{
		"status":          domain.ClientStatusDisabled,
		"release_version": "9.9.9",
	}).Error)

	specs := sharedSpecs(teacherAppID163, parentAppID163)
	specs[0].ReleaseVersion = "0.2.0"
	result, err := svc.SyncFromConfig(context.Background(), specs)
	require.NoError(t, err, "DB 已停用的客户端不应让同步失败——它是运维的紧急止损开关")
	require.Len(t, result.Warnings, 1, "DB 已停用的客户端要产生一条警告")
	assert.Contains(t, result.Warnings[0], teacherAppID163)

	var after domain.Client
	require.NoError(t, h.DB.Where("app_id = ?", teacherAppID163).First(&after).Error)
	assert.Equal(t, domain.ClientStatusDisabled, after.Status, "同步不得覆盖 DB 的 status")
	assert.Equal(t, "9.9.9", after.ReleaseVersion, "同步不得覆盖 DB 的 release_version")
}

// 换 AppID 时旧客户端置 disabled 后保留：身份仍引用旧 app_id，删除会断掉历史。
func TestClientSyncDisablesOldClientWhenAppIDChanges(t *testing.T) {
	ctx := context.Background()
	h, svc := newClientService(t)
	oldTeacherAppID := "wx_teacher_163_old"
	_, err := svc.SyncFromConfig(ctx, sharedSpecs(oldTeacherAppID, parentAppID163))
	require.NoError(t, err)

	result, err := svc.SyncFromConfig(ctx, sharedSpecs(teacherAppID163, parentAppID163))
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{teacherAppID163}, result.Created)
	assert.ElementsMatch(t, []string{oldTeacherAppID}, result.Disabled)
	assert.ElementsMatch(t, []string{parentAppID163}, result.Unchanged,
		"AppID 没变的家长端不应被停用")

	rows := listClients(t, h)
	require.Len(t, rows, 3, "旧客户端必须保留而不是删除")

	byAppID := map[string]domain.Client{}
	for _, row := range rows {
		byAppID[row.AppID] = row
	}
	old, ok := byAppID[oldTeacherAppID]
	require.True(t, ok, "旧 app_id 行必须保留")
	assert.Equal(t, domain.ClientStatusDisabled, old.Status)

	now, ok := byAppID[teacherAppID163]
	require.True(t, ok)
	assert.Equal(t, domain.ClientStatusEnabled, now.Status)

	parent, ok := byAppID[parentAppID163]
	require.True(t, ok)
	assert.Equal(t, domain.ClientStatusEnabled, parent.Status)

	// 再跑一次不能把旧行重新启用（status 由 DB 决定）。
	_, err = svc.SyncFromConfig(ctx, sharedSpecs(teacherAppID163, parentAppID163))
	require.NoError(t, err)
	var oldAfter domain.Client
	require.NoError(t, h.DB.Where("app_id = ?", oldTeacherAppID).First(&oldAfter).Error)
	assert.Equal(t, domain.ClientStatusDisabled, oldAfter.Status)
}

// v1 只允许 shared 客户端：配置里出现 dedicated/group 声明必须拒绝，且不落库。
func TestClientSyncRejectsNonSharedScopeSpec(t *testing.T) {
	h, svc := newClientService(t)
	specs := sharedSpecs(teacherAppID163, parentAppID163)
	specs[0].ScopeKind = domain.ClientScopeKindergarten
	specs[0].ScopeID = 7

	_, err := svc.SyncFromConfig(context.Background(), specs)
	require.Error(t, err, "v1 不允许 dedicated/group 客户端")
	assert.Empty(t, listClients(t, h), "校验失败时不得写入任何行")
}

// 配置里没声明、但库里已有一条 enabled 的 dedicated 客户端：同样拒绝启动。
// 这种行会让「谁是运行时客户端」出现两个答案，必须在同步前就停下来。
func TestClientSyncRejectsEnabledNonSharedClientRow(t *testing.T) {
	h, svc := newClientService(t)
	require.NoError(t, h.DB.Create(&domain.Client{
		AppID:     "wx_dedicated_163",
		Role:      domain.ClientRoleTeacher,
		ScopeKind: domain.ClientScopeKindergarten,
		ScopeID:   9,
		Status:    domain.ClientStatusEnabled,
	}).Error)

	_, err := svc.SyncFromConfig(context.Background(), sharedSpecs(teacherAppID163, parentAppID163))
	require.Error(t, err, "库里存在 enabled 的非 shared 客户端时启动必须失败")
	assert.Contains(t, err.Error(), "wx_dedicated_163", "错误信息要指出是哪一条客户端")
	assert.Len(t, listClients(t, h), 1, "校验失败时不得写入新行")
}

// 两端共用一个 AppID 会拿到同一个 openid，无法判断角色——必须在写库前拒绝。
func TestClientSyncRejectsSameAppIDForBothRoles(t *testing.T) {
	h, svc := newClientService(t)
	specs := sharedSpecs(teacherAppID163, parentAppID163)
	specs[1].AppID = specs[0].AppID

	_, err := svc.SyncFromConfig(context.Background(), specs)
	require.Error(t, err, "同一 AppID 不能同时用于老师端与家长端")
	assert.Contains(t, err.Error(), teacherAppID163, "错误信息要指出冲突的 AppID")
	assert.Empty(t, listClients(t, h), "校验失败时不得写入任何行")
}

// 一个角色只能有一条客户端：配置里同角色出现两次是配置写错，必须报错而不是后者覆盖前者。
func TestClientSyncRejectsDuplicateRoleSpecs(t *testing.T) {
	h, svc := newClientService(t)
	specs := append(sharedSpecs(teacherAppID163, parentAppID163), service.ClientSpec{
		AppID:     "wx_teacher_163_alt",
		Role:      domain.ClientRoleTeacher,
		ScopeKind: domain.ClientScopeShared,
	})

	_, err := svc.SyncFromConfig(context.Background(), specs)
	require.Error(t, err, "同一角色不能声明两个客户端")
	assert.Contains(t, err.Error(), domain.ClientRoleTeacher)
	assert.Empty(t, listClients(t, h), "校验失败时不得写入任何行")
}

// 角色决定登录路由与 JWT 角色裁剪，只能是 parent / teacher。
func TestClientSyncRejectsUnknownRoleSpec(t *testing.T) {
	h, svc := newClientService(t)

	_, err := svc.SyncFromConfig(context.Background(), []service.ClientSpec{{
		AppID:     "wx_admin_163",
		Role:      "admin",
		ScopeKind: domain.ClientScopeShared,
	}})
	require.Error(t, err, "小程序客户端只能是 parent / teacher")
	assert.Contains(t, err.Error(), "admin")
	assert.Empty(t, listClients(t, h))
}

// 本地开发常只配一端：另一端 AppID 留空应跳过同步并给出警告，而不是报错。
func TestClientSyncSkipsUnconfiguredClient(t *testing.T) {
	h, svc := newClientService(t)

	result, err := svc.SyncFromConfig(context.Background(), sharedSpecs("", parentAppID163))
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{parentAppID163}, result.Created)
	assert.Len(t, listClients(t, h), 1)
	require.Len(t, result.Warnings, 1, "未配置的角色要有一条警告")
	assert.Contains(t, result.Warnings[0], domain.ClientRoleTeacher)
}

// 反向不一致：库里有一条 enabled 的老师端客户端，但配置没有它的 AppID。
// 运行时会把这条客户端当作可登录入口，实际却没有凭据——必须拒绝启动。
func TestClientSyncRejectsEnabledClientForUnconfiguredRole(t *testing.T) {
	ctx := context.Background()
	_, svc := newClientService(t)
	_, err := svc.SyncFromConfig(ctx, sharedSpecs(teacherAppID163, parentAppID163))
	require.NoError(t, err)

	_, err = svc.SyncFromConfig(ctx, sharedSpecs("", parentAppID163))
	require.Error(t, err, "已启用但配置缺失的客户端必须让启动失败")
	assert.Contains(t, err.Error(), teacherAppID163)
}

// 配置到启动声明的映射：AppID 必须来自各自端，串了就会让两端共用一个 openid 空间。
func TestSharedClientSpecsMapsConfiguredAppIDs(t *testing.T) {
	specs := service.SharedClientSpecs("wx_teacher_cfg", "wx_parent_cfg", "1.2.3", "1.2.4")
	require.Len(t, specs, 2)

	byRole := map[string]service.ClientSpec{}
	for _, spec := range specs {
		byRole[spec.Role] = spec
	}

	teacher := byRole[domain.ClientRoleTeacher]
	assert.Equal(t, "wx_teacher_cfg", teacher.AppID)
	assert.Equal(t, domain.ClientScopeShared, teacher.ScopeKind)
	assert.Equal(t, "1.2.3", teacher.ReleaseVersion)
	assert.Equal(t, "wechat.teacher", teacher.CredentialRef)

	parent := byRole[domain.ClientRoleParent]
	assert.Equal(t, "wx_parent_cfg", parent.AppID)
	assert.Equal(t, domain.ClientScopeShared, parent.ScopeKind)
	assert.Equal(t, "1.2.4", parent.ReleaseVersion)
	assert.Equal(t, "wechat.parent", parent.CredentialRef)
}

// 登录与 mktoken 按 app_id 取客户端：未知 app_id 与 disabled 都必须显式报错。
func TestClientServiceResolveEnabledByAppID(t *testing.T) {
	ctx := context.Background()
	h, svc := newClientService(t)
	_, err := svc.SyncFromConfig(ctx, sharedSpecs(teacherAppID163, parentAppID163))
	require.NoError(t, err)

	got, err := svc.ResolveEnabledByAppID(ctx, teacherAppID163)
	require.NoError(t, err)
	assert.Equal(t, teacherAppID163, got.AppID)
	assert.Equal(t, domain.ClientRoleTeacher, got.Role)

	_, err = svc.ResolveEnabledByAppID(ctx, "wx_unknown_163")
	require.Error(t, err, "未注册的 app_id 不能签发 token")

	require.NoError(t, h.DB.Model(&domain.Client{}).
		Where("app_id = ?", parentAppID163).
		Update("status", domain.ClientStatusDisabled).Error)
	_, err = svc.ResolveEnabledByAppID(ctx, parentAppID163)
	require.Error(t, err, "停用的客户端不能签发 token")
}

// mktoken 不传 app_id 时按角色查表：同角色多条 enabled 无法确定用哪条，必须报错。
func TestClientServiceFindEnabledForRole(t *testing.T) {
	ctx := context.Background()
	h, svc := newClientService(t)
	_, err := svc.SyncFromConfig(ctx, sharedSpecs(teacherAppID163, parentAppID163))
	require.NoError(t, err)

	got, err := svc.FindEnabledForRole(ctx, domain.ClientRoleParent)
	require.NoError(t, err)
	assert.Equal(t, parentAppID163, got.AppID)

	require.NoError(t, h.DB.Create(&domain.Client{
		AppID:     "wx_parent_163_dup",
		Role:      domain.ClientRoleParent,
		ScopeKind: domain.ClientScopeShared,
		Status:    domain.ClientStatusEnabled,
	}).Error)
	_, err = svc.FindEnabledForRole(ctx, domain.ClientRoleParent)
	require.Error(t, err, "同角色多条 enabled 必须报错")

	_, err = svc.FindEnabledForRole(ctx, "admin")
	require.Error(t, err, "admin 不是小程序客户端角色")
}

// v1 的 dedicated/group 拒绝只发生在启动同步/注册入口，不在通用仓储读路径：
// #169 需要一个测试专用 dedicated 行来验证 scope 锁定，仓储必须能把它读出来。
func TestClientRepositoryReadsDirectlyInsertedDedicatedRow(t *testing.T) {
	h := authharness.New(t)
	h.HardDelete(t, &domain.Client{}, "1 = 1")
	repo := repository.NewClientRepo(h.DB)

	require.NoError(t, h.DB.Create(&domain.Client{
		AppID:     "wx_dedicated_177_test",
		Role:      domain.ClientRoleParent,
		ScopeKind: domain.ClientScopeKindergarten,
		ScopeID:   9,
		Status:    domain.ClientStatusEnabled,
	}).Error)

	got, err := repo.FindByAppID(context.Background(), "wx_dedicated_177_test")
	require.NoError(t, err, "通用仓储读取不得因 scope_kind 报错")
	require.NotNil(t, got, "直接插入的 dedicated 行必须能读出来")
	assert.Equal(t, domain.ClientScopeKindergarten, got.ScopeKind)
	assert.Equal(t, uint64(9), got.ScopeID)
}

// 历史 disabled 的非 shared 行保留但不阻断启动：它们只给旧身份引用，
// 不参与登录；v1 拒绝的是 enabled 行（见 TestClientSyncRejectsEnabledNonSharedClientRow）。
func TestClientSyncAllowsDisabledNonSharedClientRow(t *testing.T) {
	h, svc := newClientService(t)
	require.NoError(t, h.DB.Create(&domain.Client{
		AppID:     "wx_dedicated_177_disabled",
		Role:      domain.ClientRoleParent,
		ScopeKind: domain.ClientScopeKindergarten,
		ScopeID:   9,
		Status:    domain.ClientStatusDisabled,
	}).Error)

	_, err := svc.SyncFromConfig(context.Background(), sharedSpecs(teacherAppID163, parentAppID163))
	require.NoError(t, err, "disabled 的历史 dedicated 行不应阻断启动")
	assert.Len(t, listClients(t, h), 3)
}
