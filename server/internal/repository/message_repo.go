package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

type MessageRepo struct {
	db *gorm.DB
}

func NewMessageRepo(db *gorm.DB) *MessageRepo {
	return &MessageRepo{db: db}
}

func (r *MessageRepo) List(ctx context.Context, userID uint64, msgType string, isRead *bool, page, pageSize int) ([]domain.Message, int64, error) {
	q := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if msgType != "" {
		q = q.Where("type = ?", msgType)
	}
	if isRead != nil {
		q = q.Where("is_read = ?", *isRead)
	}

	var total int64
	if err := q.Model(&domain.Message{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []domain.Message
	err := q.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&list).Error
	return list, total, err
}

func (r *MessageRepo) FindByID(ctx context.Context, id, userID uint64) (*domain.Message, error) {
	var m domain.Message
	err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&m).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &m, err
}

func (r *MessageRepo) MarkRead(ctx context.Context, id, userID uint64) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&domain.Message{}).
		Where("id = ? AND user_id = ? AND is_read = ?", id, userID, false).
		Updates(map[string]any{"is_read": true, "read_at": now}).Error
}

func (r *MessageRepo) MarkAllRead(ctx context.Context, userID uint64, msgType string) (int64, error) {
	now := time.Now()
	q := r.db.WithContext(ctx).
		Model(&domain.Message{}).
		Where("user_id = ? AND is_read = ?", userID, false)
	if msgType != "" {
		q = q.Where("type = ?", msgType)
	}
	result := q.Updates(map[string]any{"is_read": true, "read_at": now})
	return result.RowsAffected, result.Error
}

type UnreadCount struct {
	Type  string `json:"type"`
	Count int64  `json:"count"`
}

func (r *MessageRepo) UnreadCounts(ctx context.Context, userID uint64) ([]UnreadCount, error) {
	var counts []UnreadCount
	err := r.db.WithContext(ctx).
		Model(&domain.Message{}).
		Select("type, count(*) as count").
		Where("user_id = ? AND is_read = ?", userID, false).
		Group("type").
		Scan(&counts).Error
	return counts, err
}
