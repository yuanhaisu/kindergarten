package service

import (
	"context"
	"time"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

type MessageService struct {
	msgRepo *repository.MessageRepo
}

func NewMessageService(msgRepo *repository.MessageRepo) *MessageService {
	return &MessageService{msgRepo: msgRepo}
}

type MessageItem struct {
	ID          uint64     `json:"id"`
	Type        string     `json:"type"`
	Title       string     `json:"title"`
	Content     string     `json:"content"`
	ReferenceID uint64     `json:"reference_id"`
	IsRead      bool       `json:"is_read"`
	CreatedAt   time.Time  `json:"created_at"`
	ReadAt      *time.Time `json:"read_at"`
}

func toMessageItem(m *domain.Message) MessageItem {
	return MessageItem{
		ID:          m.ID,
		Type:        m.Type,
		Title:       m.Title,
		Content:     m.Content,
		ReferenceID: m.ReferenceID,
		IsRead:      m.IsRead,
		CreatedAt:   m.CreatedAt,
		ReadAt:      m.ReadAt,
	}
}

func (s *MessageService) List(ctx context.Context, userID uint64, msgType string, isRead *bool, page, pageSize int) ([]MessageItem, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	msgs, total, err := s.msgRepo.List(ctx, userID, msgType, isRead, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	items := make([]MessageItem, 0, len(msgs))
	for _, m := range msgs {
		items = append(items, toMessageItem(&m))
	}
	return items, total, nil
}

type UnreadCountResult struct {
	Total  int64          `json:"total"`
	ByType map[string]int64 `json:"by_type"`
}

func (s *MessageService) UnreadCount(ctx context.Context, userID uint64) (*UnreadCountResult, error) {
	counts, err := s.msgRepo.UnreadCounts(ctx, userID)
	if err != nil {
		return nil, err
	}

	allTypes := []string{"notice", "leave", "pickup", "growth", "task", "system"}
	byType := make(map[string]int64, len(allTypes))
	for _, t := range allTypes {
		byType[t] = 0
	}

	var total int64
	for _, c := range counts {
		byType[c.Type] = c.Count
		total += c.Count
	}

	return &UnreadCountResult{Total: total, ByType: byType}, nil
}

type MarkReadResult struct {
	ID     uint64     `json:"id"`
	IsRead bool       `json:"is_read"`
	ReadAt *time.Time `json:"read_at"`
}

func (s *MessageService) MarkRead(ctx context.Context, msgID, userID uint64) (*MarkReadResult, error) {
	msg, err := s.msgRepo.FindByID(ctx, msgID, userID)
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, nil
	}

	if err := s.msgRepo.MarkRead(ctx, msgID, userID); err != nil {
		return nil, err
	}

	now := time.Now()
	return &MarkReadResult{ID: msgID, IsRead: true, ReadAt: &now}, nil
}

type MarkAllReadResult struct {
	MarkedCount int64 `json:"marked_count"`
}

func (s *MessageService) MarkAllRead(ctx context.Context, userID uint64, msgType string) (*MarkAllReadResult, error) {
	count, err := s.msgRepo.MarkAllRead(ctx, userID, msgType)
	if err != nil {
		return nil, err
	}
	return &MarkAllReadResult{MarkedCount: count}, nil
}
