package service

import (
	"context"
	"errors"
	"fmt"

	"kindergarten-server/internal/domain"
)

// ClientSpec 是一条来自配置的客户端声明。
//
// 配置只负责 seed：AppID/角色/作用域用于幂等补齐，ReleaseVersion 只在首次
// 插入时写入；之后 status 与 release_version 都由数据库维护。
type ClientSpec struct {
	AppID          string
	Role           string
	ScopeKind      string
	ScopeID        uint64
	ReleaseVersion string
	CredentialRef  string
}

// ClientSyncResult 汇总一次启动同步：调用方据此打日志，测试据此断言外部行为。
type ClientSyncResult struct {
	Created   []string
	Disabled  []string
	Unchanged []string
	// Warnings 是需要人看一眼但不阻断启动的配置/DB 差异（如某一端未配置）。
	Warnings []string
}

// ClientRepository 客户端仓储接口（由 *repository.ClientRepo 实现）。
// 声明在消费方，便于同步逻辑单测注入桩。
type ClientRepository interface {
	Create(ctx context.Context, client *domain.Client) error
	FindByAppID(ctx context.Context, appID string) (*domain.Client, error)
	FindByRole(ctx context.Context, role string) ([]domain.Client, error)
	List(ctx context.Context) ([]domain.Client, error)
	UpdateFields(ctx context.Context, id uint64, fields map[string]any) error
}

type ClientService struct {
	repo ClientRepository
}

func NewClientService(repo ClientRepository) *ClientService {
	return &ClientService{repo: repo}
}

// SyncFromConfig 幂等同步配置中的客户端。
//
// 对每条声明：
//   - 同角色下 app_id 不同的 enabled 行置 disabled（AppID 变更），行保留；
//   - 声明里的 app_id 已存在：不覆盖 status / release_version（DB 是运行时来源）；
//   - 不存在才插入，初始 status=enabled。
func (s *ClientService) SyncFromConfig(ctx context.Context, specs []ClientSpec) (*ClientSyncResult, error) {
	if err := validateClientSpecs(specs); err != nil {
		return nil, err
	}
	if err := s.validateExistingRows(ctx); err != nil {
		return nil, err
	}

	result := &ClientSyncResult{}
	configuredRoles := make(map[string]bool, len(specs))
	for _, spec := range specs {
		if spec.AppID == "" {
			continue
		}
		configuredRoles[spec.Role] = true
		if err := s.syncOne(ctx, spec, result); err != nil {
			return nil, err
		}
	}
	for _, role := range []string{domain.ClientRoleTeacher, domain.ClientRoleParent} {
		if !configuredRoles[role] {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("角色 %s 未配置 AppID，本次跳过客户端同步（该端登录不可用）", role))
		}
	}
	if err := s.validateEnabledConsistency(ctx, configuredRoles); err != nil {
		return nil, err
	}
	return result, nil
}

// validateEnabledConsistency 在同步后校验「配置与启用中的客户端一致」：
// 每条 enabled 行都必须有配置凭据支撑，否则运行时会出现「可登录但无凭据」的入口。
func (s *ClientService) validateEnabledConsistency(ctx context.Context, configuredRoles map[string]bool) error {
	rows, err := s.repo.List(ctx)
	if err != nil {
		return fmt.Errorf("读取客户端注册表失败：%w", err)
	}
	for _, row := range rows {
		if !row.IsEnabled() {
			continue
		}
		if !configuredRoles[row.Role] {
			return fmt.Errorf("客户端 %s（role=%s）在数据库中为 enabled，但配置中没有该角色的 AppID："+
				"请补全配置或先停用该客户端", row.AppID, row.Role)
		}
	}
	return nil
}

// validateExistingRows 拒绝库里已在生效的 v1 不支持的客户端。
// disabled 的历史行是允许的：它们只是保留给身份引用，不参与登录。
func (s *ClientService) validateExistingRows(ctx context.Context) error {
	rows, err := s.repo.List(ctx)
	if err != nil {
		return fmt.Errorf("读取客户端注册表失败：%w", err)
	}
	for _, row := range rows {
		if !row.IsEnabled() {
			continue
		}
		if row.ScopeKind != domain.ClientScopeShared {
			return fmt.Errorf("客户端 %s（role=%s）处于 enabled 但 scope_kind=%q：v1 只支持 shared 作用域，请先停用该行",
				row.AppID, row.Role, row.ScopeKind)
		}
	}
	return nil
}

// validateClientSpecs 在写库前拒绝 v1 不支持的客户端声明，避免半截同步。
func validateClientSpecs(specs []ClientSpec) error {
	seenAppIDs := make(map[string]string, len(specs))
	seenRoles := make(map[string]string, len(specs))
	for _, spec := range specs {
		if spec.AppID == "" {
			continue
		}
		if spec.Role != domain.ClientRoleParent && spec.Role != domain.ClientRoleTeacher {
			return fmt.Errorf("客户端 %s 的 role=%q 不合法：小程序客户端只能是 %s / %s",
				spec.AppID, spec.Role, domain.ClientRoleParent, domain.ClientRoleTeacher)
		}
		if prevRole, ok := seenAppIDs[spec.AppID]; ok {
			return fmt.Errorf("AppID %s 同时被 %s 与 %s 角色声明：两端必须使用不同的小程序，否则 openid 无法区分身份",
				spec.AppID, prevRole, spec.Role)
		}
		seenAppIDs[spec.AppID] = spec.Role
		if prevAppID, ok := seenRoles[spec.Role]; ok {
			return fmt.Errorf("角色 %s 同时声明了客户端 %s 与 %s：一个角色只能有一个客户端",
				spec.Role, prevAppID, spec.AppID)
		}
		seenRoles[spec.Role] = spec.AppID

		scopeKind := spec.ScopeKind
		if scopeKind == "" {
			scopeKind = domain.ClientScopeShared
		}
		if scopeKind != domain.ClientScopeShared {
			return fmt.Errorf("客户端 %s 声明了 scope_kind=%q：v1 只支持 shared 作用域，dedicated/group 客户端尚未开放",
				spec.AppID, scopeKind)
		}
		if spec.ScopeID != 0 {
			return fmt.Errorf("客户端 %s 作用域为 shared，scope_id 必须为 0（实际 %d）", spec.AppID, spec.ScopeID)
		}
	}
	return nil
}

// SharedClientSpecs 把配置里的两端 AppID / 发版版本转成启动同步声明。
// v1 只有 shared 作用域；空 AppID 原样保留，由 SyncFromConfig 跳过并告警。
func SharedClientSpecs(teacherAppID, parentAppID, teacherVersion, parentVersion string) []ClientSpec {
	return []ClientSpec{
		{
			AppID:          teacherAppID,
			Role:           domain.ClientRoleTeacher,
			ScopeKind:      domain.ClientScopeShared,
			ReleaseVersion: teacherVersion,
			CredentialRef:  "wechat.teacher",
		},
		{
			AppID:          parentAppID,
			Role:           domain.ClientRoleParent,
			ScopeKind:      domain.ClientScopeShared,
			ReleaseVersion: parentVersion,
			CredentialRef:  "wechat.parent",
		},
	}
}

// ResolveEnabledByAppID 按 app_id 取一条启用中的客户端（登录与签发工具用）。
// 未注册与已停用都显式报错：停用的客户端不能再换取 token。
func (s *ClientService) ResolveEnabledByAppID(ctx context.Context, appID string) (*domain.Client, error) {
	if appID == "" {
		return nil, errors.New("app_id 为空，无法解析客户端")
	}
	client, err := s.repo.FindByAppID(ctx, appID)
	if err != nil {
		return nil, fmt.Errorf("查询客户端 %s 失败：%w", appID, err)
	}
	if client == nil {
		return nil, fmt.Errorf("客户端 %s 未注册", appID)
	}
	if !client.IsEnabled() {
		return nil, fmt.Errorf("客户端 %s 已停用", appID)
	}
	return client, nil
}

// FindEnabledForRole 返回该角色唯一启用中的客户端。
// 0 条或多条都无法确定该用哪个小程序，必须报错而不是猜。
func (s *ClientService) FindEnabledForRole(ctx context.Context, role string) (*domain.Client, error) {
	if role != domain.ClientRoleParent && role != domain.ClientRoleTeacher {
		return nil, fmt.Errorf("角色 %s 不是小程序客户端角色", role)
	}
	rows, err := s.repo.FindByRole(ctx, role)
	if err != nil {
		return nil, fmt.Errorf("查询角色 %s 的客户端失败：%w", role, err)
	}
	enabled := make([]domain.Client, 0, 1)
	for _, row := range rows {
		if row.IsEnabled() {
			enabled = append(enabled, row)
		}
	}
	switch len(enabled) {
	case 0:
		return nil, fmt.Errorf("角色 %s 没有启用中的客户端", role)
	case 1:
		return &enabled[0], nil
	default:
		return nil, fmt.Errorf("角色 %s 有 %d 条启用中的客户端，无法确定使用哪一条", role, len(enabled))
	}
}

func (s *ClientService) syncOne(ctx context.Context, spec ClientSpec, result *ClientSyncResult) error {
	rows, err := s.repo.FindByRole(ctx, spec.Role)
	if err != nil {
		return fmt.Errorf("查询角色 %s 的客户端失败：%w", spec.Role, err)
	}

	var existing *domain.Client
	for i := range rows {
		row := rows[i]
		if row.AppID == spec.AppID {
			existing = &row
			continue
		}
		// 先停用旧行再插入新行：中途崩溃时该角色没有 enabled 客户端（登录直接
		// 拒绝），好过留下两条 enabled 造成「该用哪条」的歧义。
		if !row.IsEnabled() {
			continue
		}
		if err := s.repo.UpdateFields(ctx, row.ID, map[string]any{"status": domain.ClientStatusDisabled}); err != nil {
			return fmt.Errorf("停用旧客户端 %s 失败：%w", row.AppID, err)
		}
		result.Disabled = append(result.Disabled, row.AppID)
	}

	if existing != nil {
		if existing.IsEnabled() {
			result.Unchanged = append(result.Unchanged, spec.AppID)
		} else {
			result.Warnings = append(result.Warnings, fmt.Sprintf(
				"客户端 %s（角色 %s）在数据库中为 disabled，本次保持停用：status 以数据库为准", spec.AppID, spec.Role))
		}
		return nil
	}

	scopeKind := spec.ScopeKind
	if scopeKind == "" {
		scopeKind = domain.ClientScopeShared
	}
	client := &domain.Client{
		AppID:          spec.AppID,
		Role:           spec.Role,
		ScopeKind:      scopeKind,
		ScopeID:        spec.ScopeID,
		Status:         domain.ClientStatusEnabled,
		ReleaseVersion: spec.ReleaseVersion,
		CredentialRef:  spec.CredentialRef,
	}
	if err := s.repo.Create(ctx, client); err != nil {
		return fmt.Errorf("创建客户端 %s 失败：%w", spec.AppID, err)
	}
	result.Created = append(result.Created, spec.AppID)
	return nil
}
