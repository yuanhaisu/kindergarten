package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

type AdminRepo struct {
	db *gorm.DB
}

func NewAdminRepo(db *gorm.DB) *AdminRepo {
	return &AdminRepo{db: db}
}

type AdminListRow struct {
	UserID           uint64    `gorm:"column:user_id"`
	Nickname         string    `gorm:"column:nickname"`
	Avatar           string    `gorm:"column:avatar"`
	Phone            string    `gorm:"column:phone"`
	CreatedAt        time.Time `gorm:"column:created_at"`
	UserRoleAdminTag string    `gorm:"column:admin_role_tag"`
	ARoleID          *uint64   `gorm:"column:arole_id"`
	ARoleName        *string   `gorm:"column:arole_name"`
	PermCount        int64     `gorm:"column:perm_count"`
}

func (r *AdminRepo) ListAdmins(
	ctx context.Context,
	kindergartenID uint64,
	keyword string,
	offset, limit int,
) ([]AdminListRow, int64, error) {
	qCount := r.db.WithContext(ctx).Table("user_roles AS ur").
		Joins("INNER JOIN users AS u ON u.id = ur.user_id AND u.deleted_at IS NULL").
		Where("ur.role = ? AND ur.kindergarten_id = ? AND ur.deleted_at IS NULL", "admin", kindergartenID)
	if keyword != "" {
		kw := "%" + keyword + "%"
		qCount = qCount.Where("u.nickname LIKE ? OR u.phone LIKE ?", kw, kw)
	}
	var total int64
	if err := qCount.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []AdminListRow
	q := r.db.WithContext(ctx).Table("user_roles AS ur").
		Select(`u.id AS user_id, u.nickname, u.avatar, u.phone, u.created_at AS created_at,
			ur.admin_role AS admin_role_tag,
			ar.id AS arole_id, ar.name AS arole_name,
			(SELECT COUNT(*) FROM admin_role_permissions arp WHERE arp.role_id = ar.id AND arp.deleted_at IS NULL) AS perm_count`).
		Joins("INNER JOIN users AS u ON u.id = ur.user_id AND u.deleted_at IS NULL").
		Joins("LEFT JOIN user_admin_roles AS uar ON uar.user_id = u.id AND uar.kindergarten_id = ? AND uar.deleted_at IS NULL", kindergartenID).
		Joins("LEFT JOIN admin_roles AS ar ON ar.id = uar.admin_role_id AND ar.deleted_at IS NULL AND ar.kindergarten_id = ?", kindergartenID).
		Where("ur.role = ? AND ur.kindergarten_id = ? AND ur.deleted_at IS NULL", "admin", kindergartenID)
	if keyword != "" {
		kw := "%" + keyword + "%"
		q = q.Where("u.nickname LIKE ? OR u.phone LIKE ?", kw, kw)
	}
	err := q.Order("u.id ASC").Offset(offset).Limit(limit).Scan(&rows).Error
	return rows, total, err
}

func (r *AdminRepo) CountAdminRoles(ctx context.Context, kindergartenID uint64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.AdminRole{}).
		Where("kindergarten_id = ?", kindergartenID).
		Count(&n).Error
	return n, err
}

func (r *AdminRepo) CreateAdminRole(ctx context.Context, row *domain.AdminRole) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *AdminRepo) FindAdminRoleByID(ctx context.Context, kindergartenID, roleID uint64) (*domain.AdminRole, error) {
	var ar domain.AdminRole
	err := r.db.WithContext(ctx).
		Where("id = ? AND kindergarten_id = ? AND deleted_at IS NULL", roleID, kindergartenID).
		First(&ar).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &ar, err
}

func (r *AdminRepo) FirstAdminRoleID(ctx context.Context, kindergartenID uint64) (uint64, error) {
	var ar domain.AdminRole
	err := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND deleted_at IS NULL", kindergartenID).
		Order("id ASC").
		First(&ar).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	return ar.ID, err
}

func (r *AdminRepo) CountRolePermissions(ctx context.Context, roleID uint64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.AdminRolePermission{}).
		Where("role_id = ?", roleID).
		Count(&n).Error
	return n, err
}

func (r *AdminRepo) CreateUserAdminRole(ctx context.Context, row *domain.UserAdminRole) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *AdminRepo) DeleteUserAdminRole(ctx context.Context, userID, kindergartenID uint64) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND kindergarten_id = ?", userID, kindergartenID).
		Delete(&domain.UserAdminRole{}).Error
}

func (r *AdminRepo) FindUserAdminRole(ctx context.Context, userID, kindergartenID uint64) (*domain.UserAdminRole, error) {
	var row domain.UserAdminRole
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND kindergarten_id = ? AND deleted_at IS NULL", userID, kindergartenID).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

func (r *AdminRepo) UpdateUserAdminRole(ctx context.Context, userID, kindergartenID, newRoleID uint64) error {
	return r.db.WithContext(ctx).Model(&domain.UserAdminRole{}).
		Where("user_id = ? AND kindergarten_id = ?", userID, kindergartenID).
		Update("admin_role_id", newRoleID).Error
}

func (r *AdminRepo) UpsertUserAdminRoleTx(tx *gorm.DB, userID, kindergartenID, roleID uint64) error {
	var row domain.UserAdminRole
	err := tx.Where("user_id = ? AND kindergarten_id = ?", userID, kindergartenID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return tx.Create(&domain.UserAdminRole{
			UserID:         userID,
			AdminRoleID:    roleID,
			KindergartenID: kindergartenID,
		}).Error
	}
	if err != nil {
		return err
	}
	return tx.Model(&domain.UserAdminRole{}).Where("id = ?", row.ID).Update("admin_role_id", roleID).Error
}

// GetOrCreatePrimaryAdminRole 返回园所内第一条后台角色 ID；若不存在则创建「园所管理员」。
func (r *AdminRepo) GetOrCreatePrimaryAdminRole(ctx context.Context, kindergartenID uint64) (uint64, error) {
	id, err := r.FirstAdminRoleID(ctx, kindergartenID)
	if err != nil {
		return 0, err
	}
	if id > 0 {
		return id, nil
	}
	row := &domain.AdminRole{
		KindergartenID: kindergartenID,
		Name:           "园所管理员",
		Description:    "系统自动创建，可在角色管理中调整",
		IsSystem:       true,
	}
	if err := r.CreateAdminRole(ctx, row); err != nil {
		return 0, err
	}
	return row.ID, nil
}

// ResolveAdminRoleID 解析后台角色 ID：非 0 时须匹配园所内角色；园所尚无角色时允许任意非 0 请求并自动创建默认角色。
func (r *AdminRepo) ListAdminRolesByKindergarten(ctx context.Context, kindergartenID uint64) ([]domain.AdminRole, error) {
	var rows []domain.AdminRole
	err := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND deleted_at IS NULL", kindergartenID).
		Order("id ASC").
		Find(&rows).Error
	return rows, err
}

func (r *AdminRepo) CountUserAdminRolesForRole(ctx context.Context, kindergartenID, roleID uint64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.UserAdminRole{}).
		Where("kindergarten_id = ? AND admin_role_id = ? AND deleted_at IS NULL", kindergartenID, roleID).
		Count(&n).Error
	return n, err
}

func (r *AdminRepo) ListPermissionCodesForRole(ctx context.Context, roleID uint64) ([]string, error) {
	var rows []struct {
		Code string `gorm:"column:code"`
	}
	err := r.db.WithContext(ctx).Table("admin_role_permissions AS arp").
		Select("ap.code").
		Joins("INNER JOIN admin_permissions AS ap ON ap.id = arp.permission_id AND ap.deleted_at IS NULL").
		Where("arp.role_id = ? AND arp.deleted_at IS NULL", roleID).
		Order("ap.code ASC").
		Scan(&rows).Error
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.Code)
	}
	return out, err
}

func (r *AdminRepo) FindPermissionIDsByCodes(ctx context.Context, codes []string) (map[string]uint64, error) {
	if len(codes) == 0 {
		return map[string]uint64{}, nil
	}
	var rows []domain.AdminPermission
	if err := r.db.WithContext(ctx).Where("code IN ?", codes).Find(&rows).Error; err != nil {
		return nil, err
	}
	m := make(map[string]uint64, len(rows))
	for _, row := range rows {
		m[row.Code] = row.ID
	}
	return m, nil
}

func (r *AdminRepo) ReplaceRolePermissionLinks(ctx context.Context, roleID uint64, permissionIDs []uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("role_id = ?", roleID).Delete(&domain.AdminRolePermission{}).Error; err != nil {
			return err
		}
		for _, pid := range permissionIDs {
			row := domain.AdminRolePermission{
				RoleID:       roleID,
				PermissionID: pid,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *AdminRepo) UpdateAdminRoleFields(ctx context.Context, kindergartenID, roleID uint64, name, description string) error {
	return r.db.WithContext(ctx).Model(&domain.AdminRole{}).
		Where("id = ? AND kindergarten_id = ? AND deleted_at IS NULL", roleID, kindergartenID).
		Updates(map[string]any{"name": name, "description": description}).Error
}

func (r *AdminRepo) SoftDeleteAdminRole(ctx context.Context, kindergartenID, roleID uint64) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND kindergarten_id = ?", roleID, kindergartenID).
		Delete(&domain.AdminRole{}).Error
}

func (r *AdminRepo) FindAdminRoleByName(ctx context.Context, kindergartenID uint64, name string, excludeID uint64) (*domain.AdminRole, error) {
	q := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND name = ? AND deleted_at IS NULL", kindergartenID, name)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	var ar domain.AdminRole
	err := q.First(&ar).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &ar, err
}

func (r *AdminRepo) ResolveAdminRoleID(ctx context.Context, kindergartenID uint64, requested uint64) (uint64, error) {
	if requested > 0 {
		ar, err := r.FindAdminRoleByID(ctx, kindergartenID, requested)
		if err != nil {
			return 0, err
		}
		if ar != nil {
			return ar.ID, nil
		}
		n, err := r.CountAdminRoles(ctx, kindergartenID)
		if err != nil {
			return 0, err
		}
		if n == 0 {
			return r.GetOrCreatePrimaryAdminRole(ctx, kindergartenID)
		}
		return 0, fmt.Errorf("ROLE_NOT_FOUND")
	}
	return r.GetOrCreatePrimaryAdminRole(ctx, kindergartenID)
}
