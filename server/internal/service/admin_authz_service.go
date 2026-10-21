package service

import (
	"context"
	"strings"

	"kindergarten-server/internal/repository"
)

// AdminAuthzService 解析管理端账号的权限码（wire-admin-permissions design D2/D3）。
// 逐请求解析、不做缓存：角色/权限改动即时生效；园所规模下两条索引查询的代价可忽略。
type AdminAuthzService struct {
	admins    *repository.AdminRepo
	userRoles *repository.UserRoleRepo
}

func NewAdminAuthzService(admins *repository.AdminRepo, userRoles *repository.UserRoleRepo) *AdminAuthzService {
	return &AdminAuthzService{admins: admins, userRoles: userRoles}
}

// ResolveAdminCodes 按固定顺序解析 (codes, wildcard)：
//  1. UserAdminRole 有行 → 该角色码集（码集含 "*" 即通配）
//  2. 无行且 admin 角色行标记 super_admin → 通配（uid 4 类账号；
//     未来应把超管也落 UserAdminRole，此分支仅作兜底）
//  3. 否则空集——不获得任何管理端权限
func (s *AdminAuthzService) ResolveAdminCodes(ctx context.Context, kindergartenID, userID uint64) ([]string, bool, error) {
	link, err := s.admins.FindUserAdminRole(ctx, userID, kindergartenID)
	if err != nil {
		return nil, false, err
	}
	if link != nil {
		codes, err := s.admins.ListPermissionCodesForRole(ctx, link.AdminRoleID)
		if err != nil {
			return nil, false, err
		}
		return codes, hasWildcard(codes), nil
	}

	rows, err := s.userRoles.FindByUserID(ctx, userID)
	if err != nil {
		return nil, false, err
	}
	for _, r := range rows {
		if r.Role != "admin" {
			continue
		}
		if r.KindergartenID != nil && *r.KindergartenID != kindergartenID {
			continue
		}
		if strings.TrimSpace(r.AdminRole) == "super_admin" {
			return []string{"*"}, true, nil
		}
	}
	return []string{}, false, nil
}

func hasWildcard(codes []string) bool {
	for _, c := range codes {
		if strings.TrimSpace(c) == "*" {
			return true
		}
	}
	return false
}
