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

// Create 写入一条站内消息。
//
// 接送事件首次有了消息写入路径（此前全库无任何 Message 写入点，
// NotifyOnPickup 是死配置）。调用方须保证失败不回滚主业务——
// 通知是附带信息，不是安全关键动作的一部分。
func (r *MessageRepo) Create(ctx context.Context, msg *domain.Message) error {
	return r.db.WithContext(ctx).Create(msg).Error
}

// Exists 判断某用户是否已收到同一业务对象、同一文案、未被删除的某类消息。
// 用于幂等投递：晨检结论保存后若被重试、任务通知重放，不重复打扰家长。
//
// title 必须参与判定：reference_id 只在同一业务实体类型内唯一，而 `type=task`
// 同时承载「任务发布（reference_id=task_id）」与「老师回应（reference_id=response_id）」，
// 两张表的自增 id 会撞数。只按 (user_id, type, reference_id) 判重会把一条正常的
// 发布/回应通知误判成「已送达」。两类消息的标题前缀不同，加上 title 即可区分。
func (r *MessageRepo) Exists(ctx context.Context, userID uint64, msgType string, referenceID uint64, title string) (bool, error) {
	var n int64
	// Model(&Message{}) 自带软删过滤（BaseModel 有 DeletedAt），无需手写 deleted_at。
	err := r.db.WithContext(ctx).Model(&domain.Message{}).
		Where("user_id = ? AND type = ? AND reference_id = ? AND title = ?", userID, msgType, referenceID, title).
		Count(&n).Error
	return n > 0, err
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
