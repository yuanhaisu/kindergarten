// Package adminprovision 供给园所租户管理员：幂等地创建/更新用户与 admin 角色。
//
// 只被 cmd/setadmin 使用，因此放在 cmd/internal/ 下。命令层只负责解析参数、
// 读配置、连库与打印；可测的行为都在 EnsureTenantAdmin 里。
package adminprovision

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

// ErrInvalidOptions 参数不全（手机号/口令/园所 ID 缺失）。
var ErrInvalidOptions = errors.New("参数不完整")

// ErrKindergartenNotFound 目标园所不存在或已软删除。
var ErrKindergartenNotFound = errors.New("园所不存在")

// Options 一次供给请求。Password 只用于计算哈希，不会出现在 Result 里。
type Options struct {
	Phone          string
	Password       string
	KindergartenID uint64
}

// Result 供给结果，仅含可安全打印的字段。
type Result struct {
	UserID         uint64
	Phone          string
	KindergartenID uint64
	UserCreated    bool
	RoleCreated    bool
}

// Summary 供命令层原样打印。刻意不含口令：运维日志会留档。
func (r Result) Summary() string {
	userAction := "已更新"
	if r.UserCreated {
		userAction = "已创建"
	}
	roleAction := "已更新"
	if r.RoleCreated {
		roleAction = "已创建"
	}
	return fmt.Sprintf("租户管理员就绪：user_id=%d phone=%s kindergarten_id=%d 用户%s 角色%s",
		r.UserID, r.Phone, r.KindergartenID, userAction, roleAction)
}

// EnsureTenantAdmin 确保 phone 对应的管理员账号存在，并绑定到 kindergartenID。
//
// 幂等语义：
//   - 用户按手机号 upsert（与 PasswordLogin 的 FindByPhone 口径一致，取最小 id）；
//   - admin 角色按 user_id + role 唯一；已存在时更新园所绑定并把 admin_role
//     规范为 super_admin，不新增第二条；
//   - 同一 user 的历史重复 admin 角色在供给后收敛：保留 id 最小的一条，
//     其余按模型软删除语义置 deleted_at，不物理删历史数据。
//
// 园所查询走 GORM 默认作用域，已软删除的园所视为不存在。
func EnsureTenantAdmin(ctx context.Context, db *gorm.DB, opts Options) (Result, error) {
	if strings.TrimSpace(opts.Phone) == "" {
		return Result{}, fmt.Errorf("%w: 手机号不能为空", ErrInvalidOptions)
	}
	if opts.Password == "" {
		return Result{}, fmt.Errorf("%w: 口令不能为空", ErrInvalidOptions)
	}
	if opts.KindergartenID == 0 {
		return Result{}, fmt.Errorf("%w: 园所 ID 不能为空", ErrInvalidOptions)
	}

	kg, err := findKindergarten(ctx, db, opts.KindergartenID)
	if err != nil {
		return Result{}, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(opts.Password), bcrypt.DefaultCost)
	if err != nil {
		return Result{}, fmt.Errorf("计算口令哈希失败: %w", err)
	}

	res := Result{Phone: opts.Phone, KindergartenID: opts.KindergartenID}
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		user, created, err := upsertUser(tx, opts.Phone, string(hash))
		if err != nil {
			return err
		}
		res.UserID = user.ID
		res.UserCreated = created

		role, roleCreated, err := upsertAdminRole(tx, user.ID, kg)
		if err != nil {
			return err
		}
		res.RoleCreated = roleCreated

		if err := pruneDuplicateAdminRoles(tx, user.ID, role.ID); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	return res, nil
}

func findKindergarten(ctx context.Context, db *gorm.DB, id uint64) (*domain.Kindergarten, error) {
	var kg domain.Kindergarten
	err := db.WithContext(ctx).Select("id", "name").First(&kg, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: id=%d（不存在或已删除）", ErrKindergartenNotFound, id)
	}
	if err != nil {
		return nil, fmt.Errorf("查询园所失败: %w", err)
	}
	return &kg, nil
}

// upsertUser 按手机号找用户：不存在则建，存在则只重置口令。
// 不改昵称等资料，避免覆盖已有账号的展示信息。
func upsertUser(tx *gorm.DB, phone, passwordHash string) (*domain.User, bool, error) {
	var user domain.User
	err := tx.Where("phone = ?", phone).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		user = domain.User{Nickname: "admin", Phone: phone, PasswordHash: passwordHash}
		if err := tx.Create(&user).Error; err != nil {
			return nil, false, fmt.Errorf("创建用户失败: %w", err)
		}
		return &user, true, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("查询用户失败: %w", err)
	}
	if err := tx.Model(&user).Update("password_hash", passwordHash).Error; err != nil {
		return nil, false, fmt.Errorf("更新口令失败: %w", err)
	}
	return &user, false, nil
}

// upsertAdminRole 按 user_id + role 找管理员角色，绑定到 kg。
// 已存在时同时改绑 kindergarten_id / kindergarten_name，并把 admin_role
// 规范为 super_admin（历史数据可能缺失或残留其他值）。
// 返回落库后的角色与「本次是否新建」。
func upsertAdminRole(tx *gorm.DB, userID uint64, kg *domain.Kindergarten) (*domain.UserRole, bool, error) {
	kgID := kg.ID
	var role domain.UserRole
	err := tx.Where("user_id = ? AND role = ?", userID, "admin").First(&role).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		role = domain.UserRole{
			UserID:           userID,
			Role:             "admin",
			KindergartenID:   &kgID,
			KindergartenName: kg.Name,
			AdminRole:        "super_admin",
		}
		if err := tx.Create(&role).Error; err != nil {
			return nil, false, fmt.Errorf("创建管理员角色失败: %w", err)
		}
		return &role, true, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("查询管理员角色失败: %w", err)
	}
	if err := tx.Model(&role).Updates(map[string]any{
		"kindergarten_id":   kgID,
		"kindergarten_name": kg.Name,
		"admin_role":        "super_admin",
	}).Error; err != nil {
		return nil, false, fmt.Errorf("更新管理员角色失败: %w", err)
	}
	return &role, false, nil
}

// pruneDuplicateAdminRoles 软删同一用户除 keepID 之外的 admin 角色。
// 旧的供给逻辑按 user+role 取 First，历史库可能留下多条；这里幂等收敛。
// 用 GORM 默认作用域删除：UserRole 带 DeletedAt，历史行保留、可回溯。
func pruneDuplicateAdminRoles(tx *gorm.DB, userID, keepID uint64) error {
	if err := tx.Where("user_id = ? AND role = ? AND id <> ?", userID, "admin", keepID).
		Delete(&domain.UserRole{}).Error; err != nil {
		return fmt.Errorf("清理重复管理员角色失败: %w", err)
	}
	return nil
}
