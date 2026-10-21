package service

import (
	"context"
	"log/slog"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// 站内消息写入路径 —— 接送通知与晨检通知共用。
//
// 此前全库无任何 Message 写入点，NotifyOnPickup 是死配置；本文件是唯一的写入入口，
// 各业务必须走这里，不得各自再拼一条 INSERT，否则口径与失败处理会漂移。
//
// 铁律：通知是附带信息，**写失败只记日志，绝不回滚主业务**。

// sendMessage 向指定用户各写一条站内消息。
// dedupe=true 时按 (user_id, type, reference_id) 查重，已存在视为已送达。
// 返回成功与失败条数，便于调用方在响应中暴露投递结果。
func sendMessage(
	ctx context.Context,
	msgRepo *repository.MessageRepo,
	userIDs []uint64,
	msgType domain.MessageType,
	title, content string,
	referenceID uint64,
	dedupe bool,
) (sent, failed int) {
	if msgRepo == nil || len(userIDs) == 0 {
		return 0, 0
	}
	for _, uid := range userIDs {
		if dedupe {
			exists, err := msgRepo.Exists(ctx, uid, string(msgType), referenceID)
			if err != nil {
				slog.Default().Error("notify: dedupe check failed",
					"user_id", uid, "error", err)
				failed++
				continue
			}
			if exists {
				sent++
				continue
			}
		}
		msg := &domain.Message{
			UserID:      uid,
			Type:        string(msgType),
			Title:       title,
			Content:     content,
			ReferenceID: referenceID,
		}
		if err := msgRepo.Create(ctx, msg); err != nil {
			slog.Default().Error("notify: insert message failed",
				"type", msgType, "user_id", uid, "error", err)
			failed++
			continue
		}
		sent++
	}
	return sent, failed
}

// sendPickupMessage 向指定用户各写一条接送类站内消息。
// 返回真正写入成功的条数，便于调用方留痕（如一键通知的 notified 计数）。
func sendPickupMessage(
	ctx context.Context,
	msgRepo *repository.MessageRepo,
	userIDs []uint64,
	title, content string,
	referenceID uint64,
) int {
	sent, _ := sendMessage(ctx, msgRepo, userIDs, domain.MessageTypePickup, title, content, referenceID, false)
	return sent
}

// parentUserIDs 取某学生的全部家长用户 ID（经 StudentParent 关联）。
func parentUserIDs(ctx context.Context, db *gorm.DB, studentID uint64) []uint64 {
	var parents []domain.StudentParent
	if err := db.WithContext(ctx).
		Where("student_id = ?", studentID).Find(&parents).Error; err != nil {
		slog.Default().Error("pickup notify: load parents failed",
			"student_id", studentID, "error", err)
		return nil
	}
	ids := make([]uint64, 0, len(parents))
	for _, p := range parents {
		if p.UserID != 0 {
			ids = append(ids, p.UserID)
		}
	}
	return ids
}

// notifyStudentParents 向某学生的家长推送一条接送类站内消息。
// 返回写入成功的条数。
func notifyStudentParents(
	ctx context.Context,
	db *gorm.DB,
	msgRepo *repository.MessageRepo,
	studentID uint64,
	title, content string,
	referenceID uint64,
) int {
	return sendPickupMessage(ctx, msgRepo, parentUserIDs(ctx, db, studentID), title, content, referenceID)
}
