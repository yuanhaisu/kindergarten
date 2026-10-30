package repository

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

// TeacherMessageReadRepo 老师端聚合消息的读记录（teacher_message_reads）。
//
// 与 MessageRepo 无关：这里不读写 messages.is_read，也不碰 notice_read_logs。
type TeacherMessageReadRepo struct {
	db *gorm.DB
}

func NewTeacherMessageReadRepo(db *gorm.DB) *TeacherMessageReadRepo {
	return &TeacherMessageReadRepo{db: db}
}

// TeacherMessageKey 聚合消息的业务标识，与表唯一键同构。
// Type 取 notice / leave / system；ReferenceID 分别对应 notices.id / leave_requests.id / messages.id。
type TeacherMessageKey struct {
	Type        string
	ReferenceID uint64
}

// ReadSet 一次 IN 查询返回已读键集合，供列表批量判定 is_read（禁止逐条查，避免 N+1）。
// 只在调用方给出的键范围内返回：读记录里已失效的历史行不会抬高已读计数。
func (r *TeacherMessageReadRepo) ReadSet(ctx context.Context, userID uint64, keys []TeacherMessageKey) (map[TeacherMessageKey]struct{}, error) {
	out := make(map[TeacherMessageKey]struct{}, len(keys))
	if userID == 0 || len(keys) == 0 {
		return out, nil
	}
	wanted := make(map[TeacherMessageKey]struct{}, len(keys))
	types := make([]string, 0, 3)
	refs := make([]uint64, 0, len(keys))
	seenType := make(map[string]bool, 3)
	seenRef := make(map[uint64]bool, len(keys))
	for _, k := range keys {
		wanted[k] = struct{}{}
		if !seenType[k.Type] {
			seenType[k.Type] = true
			types = append(types, k.Type)
		}
		if !seenRef[k.ReferenceID] {
			seenRef[k.ReferenceID] = true
			refs = append(refs, k.ReferenceID)
		}
	}
	var rows []domain.TeacherMessageRead
	if err := r.db.WithContext(ctx).
		Select("message_type", "reference_id").
		Where("user_id = ? AND message_type IN ? AND reference_id IN ?", userID, types, refs).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		key := TeacherMessageKey{Type: row.MessageType, ReferenceID: row.ReferenceID}
		if _, ok := wanted[key]; ok {
			out[key] = struct{}{}
		}
	}
	return out, nil
}

// MarkRead 幂等写入一条读记录；已存在时不报错、不改变原记录。返回是否新建。
func (r *TeacherMessageReadRepo) MarkRead(ctx context.Context, userID uint64, key TeacherMessageKey) (bool, error) {
	res := r.db.WithContext(ctx).Exec(
		"INSERT IGNORE INTO teacher_message_reads (user_id, message_type, reference_id, created_at, updated_at) VALUES (?, ?, ?, NOW(3), NOW(3))",
		userID, key.Type, key.ReferenceID)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// MarkAllRead 把一批键幂等记为已读，返回本次新增条数。
// 单条多值 INSERT IGNORE：一次往返；重复键被忽略且不计入 RowsAffected，
// 所以「全部已读」重复点击得到 0，而不是把历史已读重复计数。
func (r *TeacherMessageReadRepo) MarkAllRead(ctx context.Context, userID uint64, keys []TeacherMessageKey) (int64, error) {
	if userID == 0 || len(keys) == 0 {
		return 0, nil
	}
	var b strings.Builder
	b.WriteString("INSERT IGNORE INTO teacher_message_reads (user_id, message_type, reference_id, created_at, updated_at) VALUES ")
	args := make([]any, 0, len(keys)*3)
	for i, k := range keys {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString("(?,?,?,NOW(3),NOW(3))")
		args = append(args, userID, k.Type, k.ReferenceID)
	}
	res := r.db.WithContext(ctx).Exec(b.String(), args...)
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}
