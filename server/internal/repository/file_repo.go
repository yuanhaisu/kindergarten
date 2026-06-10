package repository

import (
	"context"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

type FileRepo struct {
	db *gorm.DB
}

func NewFileRepo(db *gorm.DB) *FileRepo {
	return &FileRepo{db: db}
}

func (r *FileRepo) Create(ctx context.Context, f *domain.File) error {
	return r.db.WithContext(ctx).Create(f).Error
}

func (r *FileRepo) CreateBatch(ctx context.Context, files []domain.File) error {
	return r.db.WithContext(ctx).Create(&files).Error
}

func (r *FileRepo) FindByID(ctx context.Context, id uint64) (*domain.File, error) {
	var f domain.File
	err := r.db.WithContext(ctx).First(&f, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &f, err
}
