package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

type KindergartenRepo struct {
	db *gorm.DB
}

func NewKindergartenRepo(db *gorm.DB) *KindergartenRepo {
	return &KindergartenRepo{db: db}
}

func (r *KindergartenRepo) Create(ctx context.Context, kg *domain.Kindergarten) error {
	return r.db.WithContext(ctx).Create(kg).Error
}

func (r *KindergartenRepo) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.Kindergarten{}).Count(&count).Error
	return count, err
}

func (r *KindergartenRepo) FindByID(ctx context.Context, id uint64) (*domain.Kindergarten, error) {
	var k domain.Kindergarten
	err := r.db.WithContext(ctx).First(&k, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &k, err
}

func (r *KindergartenRepo) UpdateByID(ctx context.Context, id uint64, cols map[string]any) error {
	if len(cols) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Model(&domain.Kindergarten{}).Where("id = ?", id).Updates(cols).Error
}

// List 平台侧园所列表：显式跨租户，按关键字与状态过滤。
func (r *KindergartenRepo) List(ctx context.Context, page, pageSize int, keyword, status string) ([]domain.Kindergarten, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	q := r.db.WithContext(ctx).Model(&domain.Kindergarten{})
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("name LIKE ? OR principal LIKE ?", like, like)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []domain.Kindergarten
	err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

// FindByIDs 批量取园所，供列表接口回填料名称时避免 N+1。
func (r *KindergartenRepo) FindByIDs(ctx context.Context, ids []uint64) ([]domain.Kindergarten, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var list []domain.Kindergarten
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&list).Error
	return list, err
}

// CountStudents 返回在园幼儿数，用于订单档位核定。
func (r *KindergartenRepo) CountStudents(ctx context.Context, kindergartenID uint64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Table("students").
		Where("kindergarten_id = ? AND deleted_at IS NULL", kindergartenID).
		Count(&n).Error
	return n, err
}
