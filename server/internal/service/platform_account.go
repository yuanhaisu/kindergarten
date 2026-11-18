package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

// 平台账号状态对外的取值口径（design D4）。数据库里仍是 1/0。
const (
	PlatformAccountEnabled  = "enabled"
	PlatformAccountDisabled = "disabled"
)

var (
	// ErrAccountNotFound 目标平台账号不存在。
	ErrAccountNotFound = errors.New("ACCOUNT_NOT_FOUND")
	// ErrUsernameExists 用户名已被占用。
	ErrUsernameExists = errors.New("USERNAME_EXISTS")
	// ErrInvalidAccountStatus 状态取值不是 enabled/disabled。
	ErrInvalidAccountStatus = errors.New("INVALID_ACCOUNT_STATUS")
	// ErrCannotDisableSelf 不允许停用当前登录账号。
	ErrCannotDisableSelf = errors.New("CANNOT_DISABLE_SELF")
	// ErrLastAvailableAccount 停用后会没有可用账号，平台被锁死。
	ErrLastAvailableAccount = errors.New("LAST_AVAILABLE_ACCOUNT")
	// ErrInvalidOldPassword 修改本人密码时旧密码不正确。
	ErrInvalidOldPassword = errors.New("INVALID_OLD_PASSWORD")
	// ErrWeakPassword 新口令不满足强度要求（design D4）。
	ErrWeakPassword = errors.New("WEAK_PASSWORD")
	// ErrInvalidUsername 用户名为空或只有空白字符。
	ErrInvalidUsername = errors.New("INVALID_USERNAME")
	// ErrPlatformAuthUnavailable 凭证失效存储不可用；操作失败关闭（design D5）。
	ErrPlatformAuthUnavailable = errors.New("PLATFORM_AUTH_UNAVAILABLE")
)

// CreateAccountRequest 新建平台账号。角色不开放给调用方：固定 platform。
type CreateAccountRequest struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Name     string `json:"name" binding:"max=64"`
	Password string `json:"password" binding:"required"`
}

// PlatformAccountRow 账号列表行。不含口令哈希与 role（design D4）。
type PlatformAccountRow struct {
	ID        uint64    `json:"id"`
	Username  string    `json:"username"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func platformAccountStatusLabel(status int8) string {
	if status == 1 {
		return PlatformAccountEnabled
	}
	return PlatformAccountDisabled
}

func platformAccountRow(a *domain.PlatformAdmin) PlatformAccountRow {
	return PlatformAccountRow{
		ID: a.ID, Username: a.Username, Name: a.Name,
		Status: platformAccountStatusLabel(a.Status), CreatedAt: a.CreatedAt,
	}
}

func (s *PlatformService) ListAccounts(ctx context.Context, page, pageSize int, keyword string) ([]PlatformAccountRow, int64, error) {
	admins, total, err := s.adminRepo.List(ctx, page, pageSize, keyword)
	if err != nil {
		return nil, 0, err
	}
	rows := make([]PlatformAccountRow, 0, len(admins))
	for i := range admins {
		rows = append(rows, platformAccountRow(&admins[i]))
	}
	return rows, total, nil
}

// CreateAccount 新建平台账号；新建账号与既有账号权限一致（TC-NEW-PLT-0009）。
// 账号写入与审计同事务：审计失败时账号不落库（design D6）。
func (s *PlatformService) CreateAccount(ctx context.Context, req CreateAccountRequest, operator, ip string) (*PlatformAccountRow, error) {
	username := strings.TrimSpace(req.Username)
	if username == "" {
		return nil, ErrInvalidUsername
	}
	if err := validatePassword(req.Password); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrWeakPassword, err)
	}
	existing, err := s.adminRepo.FindByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrUsernameExists
	}
	hash, err := HashPassword(req.Password)
	if err != nil {
		return nil, err
	}
	admin := &domain.PlatformAdmin{
		Username: username, PasswordHash: hash, Name: strings.TrimSpace(req.Name),
		Role: "platform", Status: 1,
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.adminRepo.CreateTx(ctx, tx, admin); err != nil {
			return err
		}
		return s.logRepo.CreateTx(ctx, tx, &domain.PlatformOperationLog{
			Operator: operator, Module: "account", Action: "create", TargetType: "platform_admin",
			TargetID: admin.ID, Detail: fmt.Sprintf("创建平台账号：%s", username), IP: ip,
		})
	}); err != nil {
		return nil, err
	}
	row := platformAccountRow(admin)
	return &row, nil
}

// SetAccountStatus 启用或停用平台账号。停用自己与停用最后一个可用账号都会被
// 拒绝（design D4）；守卫在事务内复核通过后才写凭证失效 epoch，再提交数据库
// （design D5），被拒绝的操作不写 epoch。
func (s *PlatformService) SetAccountStatus(ctx context.Context, id uint64, status string, operatorID uint64, operator, ip string) error {
	var newStatus int8
	switch status {
	case PlatformAccountEnabled:
		newStatus = 1
	case PlatformAccountDisabled:
		newStatus = 0
	default:
		return ErrInvalidAccountStatus
	}
	admin, err := s.adminRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if admin == nil {
		return ErrAccountNotFound
	}
	if admin.Status == newStatus {
		// 幂等：状态未变，不写 epoch、不重复留痕。
		return nil
	}
	if newStatus == 0 {
		if operatorID != 0 && operatorID == id {
			return ErrCannotDisableSelf
		}
	}

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if newStatus == 0 {
			// 事务内加锁复核，防止并发停用绕过最后一个可用账号守卫。
			// 守卫必须先于 epoch 写入：被拒绝的操作不能把目标账号踢下线。
			available, err := s.adminRepo.CountAvailableTx(ctx, tx)
			if err != nil {
				return err
			}
			if available <= 1 {
				return ErrLastAvailableAccount
			}
			// 复核通过后才失效凭证：epoch 先于数据库提交（design D5）。
			if err := s.bumpPlatformEpoch(ctx, id); err != nil {
				return err
			}
		}
		if err := s.adminRepo.UpdateByIDTx(ctx, tx, id, map[string]any{"status": newStatus}); err != nil {
			return err
		}
		return s.logRepo.CreateTx(ctx, tx, &domain.PlatformOperationLog{
			Operator: operator, Module: "account", Action: platformAccountStatusAction(newStatus),
			TargetType: "platform_admin", TargetID: id,
			Detail: fmt.Sprintf("%s: %s -> %s", admin.Username, platformAccountStatusLabel(admin.Status), status), IP: ip,
		})
	}); err != nil {
		return err
	}
	return nil
}

func platformAccountStatusAction(status int8) string {
	if status == 1 {
		return "enable"
	}
	return "disable"
}

// ResetAccountPassword 由运营重置指定账号口令。口令由操作者提供，响应不回显。
func (s *PlatformService) ResetAccountPassword(ctx context.Context, id uint64, newPassword, operator, ip string) error {
	if err := validatePassword(newPassword); err != nil {
		return fmt.Errorf("%w: %v", ErrWeakPassword, err)
	}
	admin, err := s.adminRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if admin == nil {
		return ErrAccountNotFound
	}
	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	// 先失效既有凭证，再提交新口令。
	if err := s.bumpPlatformEpoch(ctx, id); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.adminRepo.UpdateByIDTx(ctx, tx, id, map[string]any{"password_hash": hash}); err != nil {
			return err
		}
		return s.logRepo.CreateTx(ctx, tx, &domain.PlatformOperationLog{
			Operator: operator, Module: "account", Action: "reset_password", TargetType: "platform_admin",
			TargetID: id, Detail: fmt.Sprintf("重置平台账号密码：%s", admin.Username), IP: ip,
		})
	})
}

// ChangeOwnPassword 平台账号修改本人密码。不属于强一致审计清单（design D6），
// 审计失败只记日志，不阻断改密。
func (s *PlatformService) ChangeOwnPassword(ctx context.Context, id uint64, oldPassword, newPassword string) error {
	if err := validatePassword(newPassword); err != nil {
		return fmt.Errorf("%w: %v", ErrWeakPassword, err)
	}
	admin, err := s.adminRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if admin == nil {
		return ErrAccountNotFound
	}
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(oldPassword)); err != nil {
		return ErrInvalidOldPassword
	}
	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := s.bumpPlatformEpoch(ctx, id); err != nil {
		return err
	}
	if err := s.adminRepo.UpdateByID(ctx, id, map[string]any{"password_hash": hash}); err != nil {
		return err
	}
	writeAuditBestEffort(ctx, s.logRepo, &domain.PlatformOperationLog{
		Operator: admin.Username, Module: "account", Action: "change_password",
		TargetType: "platform_admin", TargetID: id, Detail: "修改本人密码",
	})
	return nil
}

// bumpPlatformEpoch 让该账号此前的平台凭证全部失效。存储不可用时失败关闭，
// 调用方不得继续提交数据库变更（design D5）。
func (s *PlatformService) bumpPlatformEpoch(ctx context.Context, id uint64) error {
	if s.epochs == nil {
		return fmt.Errorf("%w: platform epoch store is not configured", ErrPlatformAuthUnavailable)
	}
	if err := s.epochs.BumpPlatformEpoch(ctx, strconv.FormatUint(id, 10), time.Now()); err != nil {
		return fmt.Errorf("%w: invalidate platform sessions: %w", ErrPlatformAuthUnavailable, err)
	}
	return nil
}
