package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

type UserRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) FindByOpenID(ctx context.Context, openID string) (*domain.User, error) {
	var u domain.User
	err := r.db.WithContext(ctx).Where("open_id = ?", openID).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &u, err
}

func (r *UserRepo) FindByID(ctx context.Context, id uint64) (*domain.User, error) {
	var u domain.User
	err := r.db.WithContext(ctx).First(&u, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &u, err
}

func (r *UserRepo) FindByPhone(ctx context.Context, phone string) (*domain.User, error) {
	var u domain.User
	err := r.db.WithContext(ctx).Where("phone = ?", phone).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &u, err
}

// FindByUnionID 按微信 unionid 反查用户，供跨端同人判定使用。
// 同一自然人在两端是两条 User，unionid 相同即表示同一人。
func (r *UserRepo) FindByUnionID(ctx context.Context, unionID string) (*domain.User, error) {
	if unionID == "" {
		return nil, nil
	}
	var u domain.User
	err := r.db.WithContext(ctx).Where("union_id = ?", unionID).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &u, err
}

// FindOtherByUnionID 查同一 unionid 下**另一条**用户记录（跨端同人判定用）。
// 必须排除自己：同一 unionid 对应两条 User，First() 会按 id 命中自己那条，
// 导致查不到另一端账号（同 openid-shared-first-match 一类的误命中）。
func (r *UserRepo) FindOtherByUnionID(ctx context.Context, unionID string, excludeID uint64) (*domain.User, error) {
	if unionID == "" {
		return nil, nil
	}
	var u domain.User
	err := r.db.WithContext(ctx).
		Where("union_id = ? AND id <> ?", unionID, excludeID).
		First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &u, err
}

// FindOtherByPhone 查同一手机号下**另一条**用户记录，用途与排除理由同 FindOtherByUnionID。
func (r *UserRepo) FindOtherByPhone(ctx context.Context, phone string, excludeID uint64) (*domain.User, error) {
	if phone == "" {
		return nil, nil
	}
	var u domain.User
	err := r.db.WithContext(ctx).
		Where("phone = ? AND id <> ?", phone, excludeID).
		First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &u, err
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
