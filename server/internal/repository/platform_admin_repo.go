package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"kindergarten-server/internal/domain"
)

// PlatformAdminRepo 平台方账号仓储。与园所管理员完全独立（ADR-0014）。
type PlatformAdminRepo struct {
	db *gorm.DB
}

func NewPlatformAdminRepo(db *gorm.DB) *PlatformAdminRepo {
	return &PlatformAdminRepo{db: db}
}

func (r *PlatformAdminRepo) Create(ctx context.Context, a *domain.PlatformAdmin) error {
	return r.db.WithContext(ctx).Create(a).Error
}

// CreateTx 在调用方事务内创建账号，供「账号 + 审计」同事务写入。
func (r *PlatformAdminRepo) CreateTx(ctx context.Context, tx *gorm.DB, a *domain.PlatformAdmin) error {
	return tx.WithContext(ctx).Create(a).Error
}

// List 分页返回平台账号；keyword 匹配用户名或姓名。
func (r *PlatformAdminRepo) List(ctx context.Context, page, pageSize int, keyword string) ([]domain.PlatformAdmin, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	q := r.db.WithContext(ctx).Model(&domain.PlatformAdmin{})
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("username LIKE ? OR name LIKE ?", like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []domain.PlatformAdmin
	err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

// CountAvailable 统计可用账号：status=1 且未软删。
// 平台「不得锁死」守卫以这个口径为准（design D4）。
func (r *PlatformAdminRepo) CountAvailable(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.PlatformAdmin{}).
		Where("status = 1 AND deleted_at IS NULL").Count(&n).Error
	return n, err
}

// CountAvailableTx 在事务内加锁统计可用账号，防止并发停用绕过「最后一个」守卫。
func (r *PlatformAdminRepo) CountAvailableTx(ctx context.Context, tx *gorm.DB) (int64, error) {
	var n int64
	err := tx.WithContext(ctx).Model(&domain.PlatformAdmin{}).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("status = 1 AND deleted_at IS NULL").Count(&n).Error
	return n, err
}

func (r *PlatformAdminRepo) UpdateByID(ctx context.Context, id uint64, updates map[string]any) error {
	return r.db.WithContext(ctx).Model(&domain.PlatformAdmin{}).Where("id = ?", id).Updates(updates).Error
}

func (r *PlatformAdminRepo) UpdateByIDTx(ctx context.Context, tx *gorm.DB, id uint64, updates map[string]any) error {
	return tx.WithContext(ctx).Model(&domain.PlatformAdmin{}).Where("id = ?", id).Updates(updates).Error
}

func (r *PlatformAdminRepo) FindByUsername(ctx context.Context, username string) (*domain.PlatformAdmin, error) {
	var a domain.PlatformAdmin
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&a).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &a, err
}

func (r *PlatformAdminRepo) FindByID(ctx context.Context, id uint64) (*domain.PlatformAdmin, error) {
	var a domain.PlatformAdmin
	err := r.db.WithContext(ctx).First(&a, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &a, err
}

func (r *PlatformAdminRepo) Count(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.PlatformAdmin{}).Count(&n).Error
	return n, err
}
