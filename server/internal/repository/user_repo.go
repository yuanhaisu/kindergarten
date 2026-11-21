package repository

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

// ErrPhoneRequired 空手机号不能作为账号标识。
var ErrPhoneRequired = errors.New("PHONE_REQUIRED")

// UserProfileDefaults 新建账号时写入的资料。已存在的账号**一个字段都不改**：
// 手机号只负责找到账号，不负责改资料（昵称/头像/性别/密码的修改走个人资料接口）。
type UserProfileDefaults struct {
	Nickname     string
	Avatar       string
	Gender       int
	PasswordHash string
	IDCardNumber string
}

type UserRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

// WithDB 返回绑定到指定事务的仓储副本。多表写入（学生+家长、教师+档案）必须在
// 同一事务里解析账号，否则账号会先落库、后续步骤失败后留下无角色的孤儿账号。
func (r *UserRepo) WithDB(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) FindByID(ctx context.Context, id uint64) (*domain.User, error) {
	var u domain.User
	err := r.db.WithContext(ctx).First(&u, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &u, err
}

// FindByPhone 按「活跃且非空」的手机号查账号。
//
// 走 phone_active 生成列而不是 phone：软删账号与待补手机号的账号不占用身份，
// 否则已注销账号仍能被密码登录命中。未命中返回 (nil, nil)。
func (r *UserRepo) FindByPhone(ctx context.Context, phone string) (*domain.User, error) {
	return r.findByActivePhone(ctx, strings.TrimSpace(phone))
}

func (r *UserRepo) findByActivePhone(ctx context.Context, phone string) (*domain.User, error) {
	if phone == "" {
		return nil, nil
	}
	var u domain.User
	err := r.db.WithContext(ctx).Where("phone_active = ?", phone).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// FindOrCreateByPhone 是「按手机号建/找账号」的唯一入口（#164）。
//
// 不变量：
//   - 空手机号直接拒绝（ErrPhoneRequired）；
//   - 同一活跃手机号永远解析到同一账号，并发插入由 users.phone_active 唯一索引兜底，
//     撞键视为「别人刚建好」并重新读取；
//   - 复用已有账号时绝不覆盖任何资料字段；
//   - 软删账号不参与解析。
//
// 返回值 created 表示本次是否新建，调用方可据此决定是否补发初始化逻辑（如日志）。
func (r *UserRepo) FindOrCreateByPhone(
	ctx context.Context,
	phone string,
	defaults UserProfileDefaults,
) (*domain.User, bool, error) {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return nil, false, ErrPhoneRequired
	}

	existing, err := r.findByActivePhone(ctx, phone)
	if err != nil {
		return nil, false, err
	}
	if existing != nil {
		return existing, false, nil
	}

	u := &domain.User{
		Phone:        phone,
		Nickname:     defaults.Nickname,
		Avatar:       defaults.Avatar,
		Gender:       defaults.Gender,
		PasswordHash: defaults.PasswordHash,
		IDCardNumber: defaults.IDCardNumber,
	}
	if err := r.db.WithContext(ctx).Create(u).Error; err != nil {
		if IsDuplicateKey(err) {
			// 并发解析：另一个调用已插入同一手机号。唯一索引保证只有一行，
			// 重新读取即可；仍读不到说明不是手机号冲突，原样上抛。
			existing, findErr := r.findByActivePhone(ctx, phone)
			if findErr != nil {
				return nil, false, findErr
			}
			if existing != nil {
				return existing, false, nil
			}
		}
		return nil, false, err
	}
	return u, true, nil
}

func (r *UserRepo) Create(ctx context.Context, u *domain.User) error {
	return r.db.WithContext(ctx).Create(u).Error
}

func (r *UserRepo) Update(ctx context.Context, u *domain.User) error {
	return r.db.WithContext(ctx).Save(u).Error
}

func (r *UserRepo) UpdateFields(ctx context.Context, id uint64, fields map[string]any) error {
	return r.db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", id).Updates(fields).Error
}
