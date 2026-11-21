package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/pkg/log"
	"kindergarten-server/pkg/pagination"
	"kindergarten-server/pkg/redact"
)

type AdminService struct {
	db     *gorm.DB
	users  *repository.UserRepo
	roles  *repository.UserRoleRepo
	admins *repository.AdminRepo
	kg     *repository.KindergartenRepo
	oplog  *repository.OperationLogRepo
}

func NewAdminService(
	db *gorm.DB,
	users *repository.UserRepo,
	roles *repository.UserRoleRepo,
	admins *repository.AdminRepo,
	kg *repository.KindergartenRepo,
	oplog *repository.OperationLogRepo,
) *AdminService {
	return &AdminService{db: db, users: users, roles: roles, admins: admins, kg: kg, oplog: oplog}
}

func (s *AdminService) logOp(ctx context.Context, kindergartenID, userID uint64, module, action, targetType string, targetID uint64, detail, ip, ua string) {
	if err := s.oplog.Create(ctx, &domain.OperationLog{
		KindergartenID: kindergartenID,
		UserID:         userID,
		Module:         module,
		Action:         action,
		TargetType:     targetType,
		TargetID:       targetID,
		Detail:         detail,
		IP:             ip,
		UserAgent:      truncateUAAdmin(ua),
	}); err != nil {
		// 审计失败不回滚业务（与既有 admin 写操作一致），但必须留痕，不能静默。
		log.L(ctx).Error("admin operation log write failed",
			"module", module, "action", action, "target", targetType, "target_id", targetID, "err", err)
	}
}

func truncateUAAdmin(ua string) string {
	if utf8.RuneCountInString(ua) <= 300 {
		return ua
	}
	return string([]rune(ua)[:300])
}

var adminPhoneRe = regexp.MustCompile(`^1[3-9]\d{9}$`)

func validateAdminPassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("WEAK_PASSWORD")
	}
	var hasUpper, hasLower, hasDigit bool
	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasDigit = true
		}
	}
	if !hasUpper || !hasLower || !hasDigit {
		return fmt.Errorf("WEAK_PASSWORD")
	}
	return nil
}

func maskAdminPhone(phone string) string {
	if len(phone) < 7 {
		return phone
	}
	return phone[:3] + "****" + phone[len(phone)-4:]
}

func adminTagLabel(tag string) string {
	switch strings.TrimSpace(tag) {
	case "super_admin":
		return "超级管理员"
	case "admin":
		return "管理员"
	default:
		if tag == "" {
			return "管理员"
		}
		return tag
	}
}

type adminRoleBrief struct {
	ID               uint64 `json:"id"`
	Name             string `json:"name"`
	PermissionsCount int64  `json:"permissions_count"`
}

type adminListItem struct {
	ID          uint64         `json:"id"`
	Nickname    string         `json:"nickname"`
	Avatar      string         `json:"avatar"`
	Phone       string         `json:"phone"`
	AdminRole   adminRoleBrief `json:"admin_role"`
	Status      int8           `json:"status"`
	StatusLabel string         `json:"status_label"`
	LastLoginAt *string        `json:"last_login_at,omitempty"`
	CreatedAt   string         `json:"created_at"`
}

func (s *AdminService) ListAdmins(
	ctx context.Context,
	kindergartenID uint64,
	q pagination.Query,
) ([]adminListItem, int64, int, int, error) {
	if kindergartenID == 0 {
		return nil, 0, 0, 0, fmt.Errorf("NO_KINDERGARTEN")
	}
	q.Normalize()
	rows, total, err := s.admins.ListAdmins(ctx, kindergartenID, q.Keyword, q.Offset(), q.PageSize)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	out := make([]adminListItem, 0, len(rows))
	for _, row := range rows {
		var rid uint64
		var rname string
		var pcount int64
		if row.ARoleID != nil && *row.ARoleID > 0 {
			rid = *row.ARoleID
			if row.ARoleName != nil {
				rname = *row.ARoleName
			}
			pcount = row.PermCount
		} else {
			rid = 0
			rname = adminTagLabel(row.UserRoleAdminTag)
			pcount = 0
		}
		out = append(out, adminListItem{
			ID:          row.UserID,
			Nickname:    row.Nickname,
			Avatar:      row.Avatar,
			Phone:       maskAdminPhone(row.Phone),
			AdminRole:   adminRoleBrief{ID: rid, Name: rname, PermissionsCount: pcount},
			Status:      1,
			StatusLabel: "正常",
			LastLoginAt: nil,
			CreatedAt:   row.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	if out == nil {
		out = []adminListItem{}
	}
	return out, total, q.Page, q.PageSize, nil
}

type CreateAdminReq struct {
	Phone    string `json:"phone"`
	Nickname string `json:"nickname"`
	RoleID   uint64 `json:"role_id"`
	Password string `json:"password"`
}

type createAdminResp struct {
	ID        uint64 `json:"id"`
	CreatedAt string `json:"created_at"`
}

func (s *AdminService) CreateAdmin(ctx context.Context, kindergartenID, operatorID uint64, req CreateAdminReq, ip, ua string) (*createAdminResp, error) {
	if kindergartenID == 0 {
		return nil, fmt.Errorf("NO_KINDERGARTEN")
	}
	if !adminPhoneRe.MatchString(strings.TrimSpace(req.Phone)) {
		return nil, fmt.Errorf("INVALID_PHONE")
	}
	if err := validateAdminPassword(req.Password); err != nil {
		return nil, err
	}
	kg, err := s.kg.FindByID(ctx, kindergartenID)
	if err != nil {
		return nil, err
	}
	if kg == nil {
		return nil, fmt.Errorf("KINDERGARTEN_NOT_FOUND")
	}
	roleID, err := s.admins.ResolveAdminRoleID(ctx, kindergartenID, req.RoleID)
	if err != nil {
		return nil, err
	}
	ar, err := s.admins.FindAdminRoleByID(ctx, kindergartenID, roleID)
	if err != nil {
		return nil, err
	}
	if ar == nil {
		return nil, fmt.Errorf("ROLE_NOT_FOUND")
	}
	hash, err := HashPassword(req.Password)
	if err != nil {
		return nil, err
	}
	existing, err := s.users.FindByPhone(ctx, req.Phone)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		ur, err := s.roles.FindAdminInKindergarten(ctx, existing.ID, kindergartenID)
		if err != nil {
			return nil, err
		}
		if ur != nil {
			return nil, fmt.Errorf("DUPLICATE_ADMIN_PHONE")
		}
		// 同一手机号在别的园所已有管理员身份时也要拒绝（#166）：管理端会话
		// 必须恰好绑定一个园所，跨园重复会建出一个永远登录不进来的账号。
		other, err := s.roles.FindAdminRoleOutsideKindergarten(ctx, existing.ID, kindergartenID)
		if err != nil {
			return nil, err
		}
		if other != nil {
			return nil, fmt.Errorf("DUPLICATE_ADMIN_PHONE")
		}
	}
	var newUserID uint64
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 建管理员也必须走统一的手机号账号解析（#164/#173）：直接在事务里 Create
		// 一条 User 会把未规范化的手机号原样写进 phone_active，和密码登录的
		// FindByPhone 口径不一致，也绕开了「一人一账号」的唯一入口。
		u, created, err := s.users.WithDB(tx).FindOrCreateByPhone(ctx, req.Phone, repository.UserProfileDefaults{
			Nickname:     strings.TrimSpace(req.Nickname),
			PasswordHash: hash,
		})
		if err != nil {
			return err
		}
		if !created {
			// 解析器复用已有账号时不改资料；管理员创建语义仍以本次提交的昵称/口令为准。
			if err := tx.Model(u).Updates(map[string]any{
				"nickname":      strings.TrimSpace(req.Nickname),
				"password_hash": hash,
			}).Error; err != nil {
				return err
			}
		}
		newUserID = u.ID
		kid := kindergartenID
		nur := &domain.UserRole{
			UserID:           newUserID,
			Role:             "admin",
			KindergartenID:   &kid,
			KindergartenName: kg.Name,
			AdminRole:        ar.Name,
		}
		if err := tx.Create(nur).Error; err != nil {
			return err
		}
		uar := &domain.UserAdminRole{
			UserID:         newUserID,
			AdminRoleID:    roleID,
			KindergartenID: kindergartenID,
		}
		if err := tx.Create(uar).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.logOp(ctx, kindergartenID, operatorID, "admins", "create", "admin", newUserID,
		fmt.Sprintf(`{"phone":%q,"nickname":%q,"role_id":%d}`, redact.Phone(req.Phone), redact.Name(req.Nickname), roleID), ip, ua)
	u2, err := s.users.FindByID(ctx, newUserID)
	if err != nil || u2 == nil {
		return &createAdminResp{ID: newUserID, CreatedAt: time.Now().UTC().Format(time.RFC3339)}, nil
	}
	return &createAdminResp{ID: newUserID, CreatedAt: u2.CreatedAt.UTC().Format(time.RFC3339)}, nil
}

type UpdateAdminReq struct {
	Nickname *string `json:"nickname"`
	RoleID   *uint64 `json:"role_id"`
}

type updateAdminResp struct {
	ID        uint64 `json:"id"`
	UpdatedAt string `json:"updated_at"`
}

func (s *AdminService) UpdateAdmin(ctx context.Context, kindergartenID, operatorID, targetUserID uint64, req UpdateAdminReq, ip, ua string) (*updateAdminResp, error) {
	if kindergartenID == 0 {
		return nil, fmt.Errorf("NO_KINDERGARTEN")
	}
	ur, err := s.roles.FindAdminInKindergarten(ctx, targetUserID, kindergartenID)
	if err != nil {
		return nil, err
	}
	if ur == nil {
		return nil, fmt.Errorf("NOT_FOUND")
	}
	var roleID uint64
	var ar *domain.AdminRole
	if req.RoleID != nil {
		rid, err := s.admins.ResolveAdminRoleID(ctx, kindergartenID, *req.RoleID)
		if err != nil {
			return nil, err
		}
		ar, err = s.admins.FindAdminRoleByID(ctx, kindergartenID, rid)
		if err != nil {
			return nil, err
		}
		if ar == nil {
			return nil, fmt.Errorf("ROLE_NOT_FOUND")
		}
		roleID = ar.ID
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if req.Nickname != nil {
			if err := tx.Model(&domain.User{}).Where("id = ?", targetUserID).
				Update("nickname", strings.TrimSpace(*req.Nickname)).Error; err != nil {
				return err
			}
		}
		if req.RoleID != nil {
			if err := s.admins.UpsertUserAdminRoleTx(tx, targetUserID, kindergartenID, roleID); err != nil {
				return err
			}
			if ar != nil {
				if err := tx.Model(&domain.UserRole{}).Where("id = ?", ur.ID).Update("admin_role", ar.Name).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	u2, err := s.users.FindByID(ctx, targetUserID)
	if err != nil || u2 == nil {
		return nil, fmt.Errorf("NOT_FOUND")
	}
	detail := map[string]any{}
	if req.Nickname != nil {
		detail["nickname"] = redact.Name(*req.Nickname)
	}
	if req.RoleID != nil {
		detail["role_id"] = *req.RoleID
	}
	detailBytes, _ := json.Marshal(detail)
	s.logOp(ctx, kindergartenID, operatorID, "admins", "update", "admin", targetUserID, string(detailBytes), ip, ua)
	return &updateAdminResp{ID: targetUserID, UpdatedAt: u2.UpdatedAt.UTC().Format(time.RFC3339)}, nil
}

func (s *AdminService) DeleteAdmin(ctx context.Context, kindergartenID, operatorID, targetUserID uint64, ip, ua string) error {
	if kindergartenID == 0 {
		return fmt.Errorf("NO_KINDERGARTEN")
	}
	if targetUserID == operatorID {
		return fmt.Errorf("CANNOT_DELETE_SELF")
	}
	ur, err := s.roles.FindAdminInKindergarten(ctx, targetUserID, kindergartenID)
	if err != nil {
		return err
	}
	if ur == nil {
		return fmt.Errorf("NOT_FOUND")
	}
	n, err := s.roles.CountAdminsInKindergarten(ctx, kindergartenID)
	if err != nil {
		return err
	}
	if n <= 1 {
		return fmt.Errorf("LAST_ADMIN")
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ? AND kindergarten_id = ?", targetUserID, kindergartenID).
			Delete(&domain.UserAdminRole{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ? AND role = ? AND kindergarten_id = ?", targetUserID, "admin", kindergartenID).
			Delete(&domain.UserRole{}).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	s.logOp(ctx, kindergartenID, operatorID, "admins", "delete", "admin", targetUserID, "{}", ip, ua)
	return nil
}

type ResetAdminPasswordReq struct {
	NewPassword string `json:"new_password"`
}

type resetAdminPasswordResp struct {
	Message string `json:"message"`
}

func (s *AdminService) ResetAdminPassword(ctx context.Context, kindergartenID, operatorID, targetUserID uint64, req ResetAdminPasswordReq, ip, ua string) (*resetAdminPasswordResp, error) {
	if kindergartenID == 0 {
		return nil, fmt.Errorf("NO_KINDERGARTEN")
	}
	ur, err := s.roles.FindAdminInKindergarten(ctx, targetUserID, kindergartenID)
	if err != nil {
		return nil, err
	}
	if ur == nil {
		return nil, fmt.Errorf("NOT_FOUND")
	}
	if err := validateAdminPassword(req.NewPassword); err != nil {
		return nil, err
	}
	hash, err := HashPassword(req.NewPassword)
	if err != nil {
		return nil, err
	}
	if err := s.users.UpdateFields(ctx, targetUserID, map[string]any{"password_hash": hash}); err != nil {
		return nil, err
	}
	s.logOp(ctx, kindergartenID, operatorID, "admins", "reset_password", "admin", targetUserID, `{}`, ip, ua)
	return &resetAdminPasswordResp{Message: "密码已重置"}, nil
}

func IsAdminClientErr(err error) bool {
	if err == nil {
		return false
	}
	switch err.Error() {
	case "INVALID_PHONE", "WEAK_PASSWORD", "DUPLICATE_ADMIN_PHONE", "ROLE_NOT_FOUND",
		"CANNOT_DELETE_SELF", "LAST_ADMIN", "KINDERGARTEN_NOT_FOUND":
		return true
	default:
		return false
	}
}
