package service

import (
	"context"
	"fmt"
	"regexp"
	"unicode"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

type SystemService struct {
	db           *gorm.DB
	userRepo     *repository.UserRepo
	userRoleRepo *repository.UserRoleRepo
	kgRepo       *repository.KindergartenRepo
}

func NewSystemService(
	db *gorm.DB,
	userRepo *repository.UserRepo,
	userRoleRepo *repository.UserRoleRepo,
	kgRepo *repository.KindergartenRepo,
) *SystemService {
	return &SystemService{
		db:           db,
		userRepo:     userRepo,
		userRoleRepo: userRoleRepo,
		kgRepo:       kgRepo,
	}
}

// IsInitialized 判断系统是否已完成初始化（以是否存在 admin 角色为准）
func (s *SystemService) IsInitialized(ctx context.Context) (bool, error) {
	count, err := s.userRoleRepo.CountByRole(ctx, "admin")
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// InitRequest 初始化请求参数
type InitRequest struct {
	KindergartenName string `json:"kindergarten_name" binding:"required,min=2,max=64"`
	AdminPhone       string `json:"admin_phone" binding:"required"`
	AdminPassword    string `json:"admin_password" binding:"required,min=8"`
	AdminNickname    string `json:"admin_nickname" binding:"required,min=1,max=32"`
}

// Initialize 执行系统初始化，创建幼儿园和超级管理员（幂等保护，已初始化则报错）
func (s *SystemService) Initialize(ctx context.Context, req InitRequest) error {
	initialized, err := s.IsInitialized(ctx)
	if err != nil {
		return err
	}
	if initialized {
		return fmt.Errorf("ALREADY_INITIALIZED")
	}

	if err := validatePhone(req.AdminPhone); err != nil {
		return fmt.Errorf("INVALID_PHONE: %w", err)
	}
	if err := validatePassword(req.AdminPassword); err != nil {
		return fmt.Errorf("WEAK_PASSWORD: %w", err)
	}

	// 检查手机号是否已被使用
	existing, err := s.userRepo.FindByPhone(ctx, req.AdminPhone)
	if err != nil {
		return err
	}
	if existing != nil {
		return fmt.Errorf("PHONE_EXISTS")
	}

	hash, err := HashPassword(req.AdminPassword)
	if err != nil {
		return err
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		kg := &domain.Kindergarten{
			Name: req.KindergartenName,
		}
		if err := tx.Create(kg).Error; err != nil {
			return err
		}

		user := &domain.User{
			Phone:        req.AdminPhone,
			Nickname:     req.AdminNickname,
			PasswordHash: hash,
		}
		if err := tx.Create(user).Error; err != nil {
			return err
		}

		userRole := &domain.UserRole{
			UserID:           user.ID,
			Role:             "admin",
			KindergartenID:   &kg.ID,
			KindergartenName: kg.Name,
			AdminRole:        "super_admin",
		}
		if err := tx.Create(userRole).Error; err != nil {
			return err
		}

		adminRole := &domain.AdminRole{
			KindergartenID: kg.ID,
			Name:           "超级管理员",
			Description:    "系统初始化创建",
			IsSystem:       true,
		}
		if err := tx.Create(adminRole).Error; err != nil {
			return err
		}
		uar := &domain.UserAdminRole{
			UserID:         user.ID,
			AdminRoleID:    adminRole.ID,
			KindergartenID: kg.ID,
		}
		if err := tx.Create(uar).Error; err != nil {
			return err
		}

		return nil
	})
}

var phoneRe = regexp.MustCompile(`^1[3-9]\d{9}$`)

func validatePhone(phone string) error {
	if !phoneRe.MatchString(phone) {
		return fmt.Errorf("手机号格式不正确")
	}
	return nil
}

// validatePassword 要求至少 8 位，包含大写、小写、数字
func validatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("密码长度不能少于 8 位")
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
		return fmt.Errorf("密码须包含大写字母、小写字母和数字")
	}
	return nil
}
