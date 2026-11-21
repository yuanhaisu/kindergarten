package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

// ClientRepo 小程序客户端注册表（#163）。
//
// 不提供删除方法：客户端退役只改 status，历史身份仍引用 app_id。
type ClientRepo struct {
	db *gorm.DB
}

func NewClientRepo(db *gorm.DB) *ClientRepo {
	return &ClientRepo{db: db}
}

func (r *ClientRepo) Create(ctx context.Context, client *domain.Client) error {
	return r.db.WithContext(ctx).Create(client).Error
}

// FindByAppID 未命中返回 (nil, nil)，与其他仓储一致：调用方据此决定是新建还是更新。
func (r *ClientRepo) FindByAppID(ctx context.Context, appID string) (*domain.Client, error) {
	var c domain.Client
	err := r.db.WithContext(ctx).Where("app_id = ?", appID).First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// FindByRole 返回该角色的全部客户端（含 disabled），按 id 升序。
// 同步需要看到已停用的旧行才能判断 AppID 变更；登录侧应改用 FindEnabledByRole。
func (r *ClientRepo) FindByRole(ctx context.Context, role string) ([]domain.Client, error) {
	var rows []domain.Client
	err := r.db.WithContext(ctx).
		Where("role = ?", role).
		Order("id ASC").
		Find(&rows).Error
	return rows, err
}

func (r *ClientRepo) List(ctx context.Context) ([]domain.Client, error) {
	var rows []domain.Client
	err := r.db.WithContext(ctx).Order("id ASC").Find(&rows).Error
	return rows, err
}

func (r *ClientRepo) UpdateFields(ctx context.Context, id uint64, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&domain.Client{}).
		Where("id = ?", id).
		Updates(fields).Error
}
