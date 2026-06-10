package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

type RoleService struct {
	db     *gorm.DB
	admins *repository.AdminRepo
	oplog  *repository.OperationLogRepo
}

func NewRoleService(db *gorm.DB, admins *repository.AdminRepo, oplog *repository.OperationLogRepo) *RoleService {
	return &RoleService{db: db, admins: admins, oplog: oplog}
}

func (s *RoleService) logOp(ctx context.Context, kindergartenID, userID uint64, module, action, targetType string, targetID uint64, detail, ip, ua string) {
	_ = s.oplog.Create(ctx, &domain.OperationLog{
		KindergartenID: kindergartenID,
		UserID:         userID,
		Module:         module,
		Action:         action,
		TargetType:     targetType,
		TargetID:       targetID,
		Detail:         detail,
		IP:             ip,
		UserAgent:      truncateUAAdmin(ua),
	})
}

func (s *RoleService) ensurePermissionCatalog(ctx context.Context) error {
	var n int64
	if err := s.db.WithContext(ctx).Model(&domain.AdminPermission{}).Where("code = ?", "dashboard:view").Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	return seedAdminPermissions(ctx, s.db)
}

type roleListItem struct {
	ID          uint64   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	IsSystem    bool     `json:"is_system"`
	AdminCount  int64    `json:"admin_count"`
	Permissions []string `json:"permissions"`
	CreatedAt   string   `json:"created_at"`
}

type listRolesData struct {
	List []roleListItem `json:"list"`
}

func (s *RoleService) ListRoles(ctx context.Context, kindergartenID uint64) (*listRolesData, error) {
	if kindergartenID == 0 {
		return nil, fmt.Errorf("NO_KINDERGARTEN")
	}
	if err := s.ensurePermissionCatalog(ctx); err != nil {
		return nil, err
	}
	// 新园所或历史数据可能尚无后台角色行；列表前先保证存在系统默认「园所管理员」，避免空列表与管理员绑定逻辑脱节。
	if _, err := s.admins.GetOrCreatePrimaryAdminRole(ctx, kindergartenID); err != nil {
		return nil, err
	}
	rows, err := s.admins.ListAdminRolesByKindergarten(ctx, kindergartenID)
	if err != nil {
		return nil, err
	}
	out := make([]roleListItem, 0, len(rows))
	for _, ar := range rows {
		cnt, err := s.admins.CountUserAdminRolesForRole(ctx, kindergartenID, ar.ID)
		if err != nil {
			return nil, err
		}
		codes, err := s.admins.ListPermissionCodesForRole(ctx, ar.ID)
		if err != nil {
			return nil, err
		}
		perms := normalizeRolePermissionResponse(&ar, codes)
		out = append(out, roleListItem{
			ID:          ar.ID,
			Name:        ar.Name,
			Description: ar.Description,
			IsSystem:    ar.IsSystem,
			AdminCount:  cnt,
			Permissions: perms,
			CreatedAt:   ar.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return &listRolesData{List: out}, nil
}

func normalizeRolePermissionResponse(ar *domain.AdminRole, codes []string) []string {
	if len(codes) == 0 && ar.Name == "超级管理员" {
		return []string{"*"}
	}
	for _, c := range codes {
		if c == "*" {
			return []string{"*"}
		}
	}
	sort.Strings(codes)
	return codes
}

func validatePermissionPayload(perms []string, required bool) error {
	if required && len(perms) == 0 {
		return fmt.Errorf("EMPTY_PERMISSIONS")
	}
	hasStar := false
	for _, p := range perms {
		p = strings.TrimSpace(p)
		if p == "*" {
			hasStar = true
			break
		}
	}
	if hasStar && len(perms) != 1 {
		return fmt.Errorf("INVALID_WILDCARD_MIX")
	}
	for _, p := range perms {
		p = strings.TrimSpace(p)
		if !isValidPermissionCode(p) {
			return fmt.Errorf("INVALID_PERMISSION")
		}
	}
	return nil
}

func normalizeNameDesc(name, description string) (string, string, error) {
	n := strings.TrimSpace(name)
	if n == "" {
		return "", "", fmt.Errorf("EMPTY_NAME")
	}
	if utf8.RuneCountInString(n) > 50 {
		return "", "", fmt.Errorf("NAME_TOO_LONG")
	}
	d := strings.TrimSpace(description)
	if utf8.RuneCountInString(d) > 200 {
		return "", "", fmt.Errorf("DESCRIPTION_TOO_LONG")
	}
	return n, d, nil
}

type CreateRoleReq struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

type createRoleResp struct {
	ID        uint64 `json:"id"`
	CreatedAt string `json:"created_at"`
}

func (s *RoleService) CreateRole(ctx context.Context, kindergartenID, operatorID uint64, req CreateRoleReq, ip, ua string) (*createRoleResp, error) {
	if kindergartenID == 0 {
		return nil, fmt.Errorf("NO_KINDERGARTEN")
	}
	if err := s.ensurePermissionCatalog(ctx); err != nil {
		return nil, err
	}
	name, desc, err := normalizeNameDesc(req.Name, req.Description)
	if err != nil {
		return nil, err
	}
	if err := validatePermissionPayload(req.Permissions, true); err != nil {
		return nil, err
	}
	dup, err := s.admins.FindAdminRoleByName(ctx, kindergartenID, name, 0)
	if err != nil {
		return nil, err
	}
	if dup != nil {
		return nil, fmt.Errorf("DUPLICATE_ROLE_NAME")
	}
	row := &domain.AdminRole{
		KindergartenID: kindergartenID,
		Name:           name,
		Description:    desc,
		IsSystem:       false,
	}
	if err := s.admins.CreateAdminRole(ctx, row); err != nil {
		return nil, err
	}
	ids, err := s.permissionIDsForPayload(ctx, req.Permissions)
	if err != nil {
		return nil, err
	}
	if err := s.admins.ReplaceRolePermissionLinks(ctx, row.ID, ids); err != nil {
		return nil, err
	}
	detail, _ := json.Marshal(req)
	s.logOp(ctx, kindergartenID, operatorID, "roles", "create", "role", row.ID, string(detail), ip, ua)
	return &createRoleResp{ID: row.ID, CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339)}, nil
}

func (s *RoleService) permissionIDsForPayload(ctx context.Context, perms []string) ([]uint64, error) {
	if len(perms) == 1 && strings.TrimSpace(perms[0]) == "*" {
		m, err := s.admins.FindPermissionIDsByCodes(ctx, []string{"*"})
		if err != nil {
			return nil, err
		}
		id, ok := m["*"]
		if !ok {
			return nil, fmt.Errorf("CATALOG_NOT_READY")
		}
		return []uint64{id}, nil
	}
	codes := make([]string, 0, len(perms))
	for _, p := range perms {
		codes = append(codes, strings.TrimSpace(p))
	}
	m, err := s.admins.FindPermissionIDsByCodes(ctx, codes)
	if err != nil {
		return nil, err
	}
	if len(m) != len(codes) {
		return nil, fmt.Errorf("INVALID_PERMISSION")
	}
	ids := make([]uint64, 0, len(codes))
	for _, c := range codes {
		ids = append(ids, m[c])
	}
	return ids, nil
}

type UpdateRoleReq struct {
	Name        *string  `json:"name"`
	Description *string  `json:"description"`
	Permissions []string `json:"permissions"`
}

type updateRoleResp struct {
	ID        uint64 `json:"id"`
	UpdatedAt string `json:"updated_at"`
}

func (s *RoleService) UpdateRole(ctx context.Context, kindergartenID, operatorID, roleID uint64, req UpdateRoleReq, ip, ua string) (*updateRoleResp, error) {
	if kindergartenID == 0 {
		return nil, fmt.Errorf("NO_KINDERGARTEN")
	}
	if err := s.ensurePermissionCatalog(ctx); err != nil {
		return nil, err
	}
	ar, err := s.admins.FindAdminRoleByID(ctx, kindergartenID, roleID)
	if err != nil {
		return nil, err
	}
	if ar == nil {
		return nil, fmt.Errorf("NOT_FOUND")
	}
	name := ar.Name
	desc := ar.Description
	if req.Name != nil {
		n := strings.TrimSpace(*req.Name)
		if ar.IsSystem && n != ar.Name {
			return nil, fmt.Errorf("SYSTEM_ROLE_NAME_LOCKED")
		}
		if n == "" {
			return nil, fmt.Errorf("EMPTY_NAME")
		}
		if utf8.RuneCountInString(n) > 50 {
			return nil, fmt.Errorf("NAME_TOO_LONG")
		}
		dup, err := s.admins.FindAdminRoleByName(ctx, kindergartenID, n, roleID)
		if err != nil {
			return nil, err
		}
		if dup != nil {
			return nil, fmt.Errorf("DUPLICATE_ROLE_NAME")
		}
		name = n
	}
	if req.Description != nil {
		d := strings.TrimSpace(*req.Description)
		if utf8.RuneCountInString(d) > 200 {
			return nil, fmt.Errorf("DESCRIPTION_TOO_LONG")
		}
		desc = d
	}
	if req.Name != nil || req.Description != nil {
		if err := s.admins.UpdateAdminRoleFields(ctx, kindergartenID, roleID, name, desc); err != nil {
			return nil, err
		}
	}
	if req.Permissions != nil {
		if err := validatePermissionPayload(req.Permissions, true); err != nil {
			return nil, err
		}
		ids, err := s.permissionIDsForPayload(ctx, req.Permissions)
		if err != nil {
			return nil, err
		}
		if err := s.admins.ReplaceRolePermissionLinks(ctx, roleID, ids); err != nil {
			return nil, err
		}
		if err := s.db.WithContext(ctx).Model(&domain.AdminRole{}).
			Where("id = ? AND kindergarten_id = ? AND deleted_at IS NULL", roleID, kindergartenID).
			Update("updated_at", time.Now()).Error; err != nil {
			return nil, err
		}
	}
	u2, err := s.fetchRoleUpdatedAt(ctx, roleID)
	if err != nil {
		return nil, err
	}
	detail, _ := json.Marshal(req)
	s.logOp(ctx, kindergartenID, operatorID, "roles", "update", "role", roleID, string(detail), ip, ua)
	return &updateRoleResp{ID: roleID, UpdatedAt: u2.UTC().Format(time.RFC3339)}, nil
}

func (s *RoleService) fetchRoleUpdatedAt(ctx context.Context, roleID uint64) (time.Time, error) {
	var row domain.AdminRole
	err := s.db.WithContext(ctx).Select("updated_at").Where("id = ?", roleID).First(&row).Error
	return row.UpdatedAt, err
}

func (s *RoleService) DeleteRole(ctx context.Context, kindergartenID, operatorID, roleID uint64, ip, ua string) error {
	if kindergartenID == 0 {
		return fmt.Errorf("NO_KINDERGARTEN")
	}
	ar, err := s.admins.FindAdminRoleByID(ctx, kindergartenID, roleID)
	if err != nil {
		return err
	}
	if ar == nil {
		return fmt.Errorf("NOT_FOUND")
	}
	if ar.IsSystem {
		return fmt.Errorf("SYSTEM_ROLE_PROTECTED")
	}
	n, err := s.admins.CountUserAdminRolesForRole(ctx, kindergartenID, roleID)
	if err != nil {
		return err
	}
	if n > 0 {
		return fmt.Errorf("ROLE_IN_USE")
	}
	if err := s.admins.SoftDeleteAdminRole(ctx, kindergartenID, roleID); err != nil {
		return err
	}
	s.logOp(ctx, kindergartenID, operatorID, "roles", "delete", "role", roleID, `{}`, ip, ua)
	return nil
}

func (s *RoleService) PermissionTree(ctx context.Context) ([]permissionModuleResp, error) {
	if err := s.ensurePermissionCatalog(ctx); err != nil {
		return nil, err
	}
	return adminPermissionModules(), nil
}

func IsRoleClientErr(err error) bool {
	if err == nil {
		return false
	}
	switch err.Error() {
	case "NO_KINDERGARTEN", "EMPTY_NAME", "EMPTY_PERMISSIONS", "NAME_TOO_LONG", "DESCRIPTION_TOO_LONG",
		"DUPLICATE_ROLE_NAME", "INVALID_PERMISSION", "INVALID_WILDCARD_MIX", "SYSTEM_ROLE_NAME_LOCKED",
		"SYSTEM_ROLE_PROTECTED", "ROLE_IN_USE", "NOT_FOUND":
		return true
	default:
		return false
	}
}
