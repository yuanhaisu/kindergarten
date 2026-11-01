package service

import (
	"context"
	"fmt"
	"log/slog"

	"kindergarten-server/internal/domain"
)

// albumNotifyTitle 固定标题是幂等键的一部分：老师修改相册标题后重发，
// 不能让家长再收一条「新」消息（design D3）。
const albumNotifyTitle = "班级相册已更新"

// notifyAlbumPublished 在相册发布成功后向参与者家长投递站内消息。
//
// 铁律同 pickup_notify：通知是附带信息，任何失败只记日志，绝不回滚或阻断发布。
// 收件人按「相册参与者 → 监护人 → 本人 allow_class_share」逐层收敛：
//   - 只通知参与者家庭，不向全班广播（全班视图是 Phase 2-A 的读取能力）；
//   - 监护人本人关闭班内共享时不收；无设置行按默认 true；
//   - 同一家长关联多个参与孩子时只发一条，action_path 取第一个孩子。
func (s *TeacherPortalService) notifyAlbumPublished(
	ctx context.Context,
	tc *TeacherPortalContext,
	album *domain.Album,
	photoCount int,
) {
	if s == nil || s.db == nil || s.msg == nil || tc == nil || album == nil {
		return
	}

	var participants []domain.AlbumStudent
	if err := s.db.WithContext(ctx).
		Where("album_id = ? AND kindergarten_id = ?", album.ID, tc.KindergartenID).
		Find(&participants).Error; err != nil {
		slog.Default().Error("album notify: load participants failed",
			"album_id", album.ID, "error", err)
		return
	}
	if len(participants) == 0 {
		return
	}
	studentIDs := make([]uint64, 0, len(participants))
	for _, p := range participants {
		if p.StudentID > 0 {
			studentIDs = append(studentIDs, p.StudentID)
		}
	}
	if len(studentIDs) == 0 {
		return
	}

	var links []domain.StudentParent
	if err := s.db.WithContext(ctx).
		Where("student_id IN ?", studentIDs).
		Order("student_id ASC, user_id ASC").
		Find(&links).Error; err != nil {
		slog.Default().Error("album notify: load guardians failed",
			"album_id", album.ID, "error", err)
		return
	}
	if len(links) == 0 {
		return
	}

	// 一个家长可能关联多个参与孩子：只保留第一个，避免一次发布收多条。
	childOf := make(map[uint64]uint64, len(links))
	userIDs := make([]uint64, 0, len(links))
	for _, l := range links {
		if l.UserID == 0 {
			continue
		}
		if _, ok := childOf[l.UserID]; ok {
			continue
		}
		childOf[l.UserID] = l.StudentID
		userIDs = append(userIDs, l.UserID)
	}
	if len(userIDs) == 0 {
		return
	}

	// 家长设置按 (user_id, kindergarten_id) 唯一；缺失行按 Phase 1 默认 true。
	var settings []domain.ParentSetting
	if err := s.db.WithContext(ctx).
		Where("user_id IN ? AND kindergarten_id = ?", userIDs, tc.KindergartenID).
		Find(&settings).Error; err != nil {
		slog.Default().Error("album notify: load parent settings failed",
			"album_id", album.ID, "error", err)
		return
	}
	allow := make(map[uint64]bool, len(settings))
	for _, st := range settings {
		allow[st.UserID] = st.AllowClassShare
	}

	recipients := make([]messageRecipient, 0, len(userIDs))
	for _, uid := range userIDs {
		if v, ok := allow[uid]; ok && !v {
			continue
		}
		recipients = append(recipients, messageRecipient{
			UserID:     uid,
			ActionPath: fmt.Sprintf("/pages/album/detail?id=%d&student_id=%d", album.ID, childOf[uid]),
		})
	}
	if len(recipients) == 0 {
		return
	}

	content := fmt.Sprintf("老师发布了《%s》，共 %d 张照片", album.Title, photoCount)
	sent, failed := sendMessageToRecipients(
		ctx, s.msg, recipients, domain.MessageTypeAlbum, albumNotifyTitle, content, album.ID, true)
	if failed > 0 {
		slog.Default().Error("album notify: delivery partially failed",
			"album_id", album.ID, "sent", sent, "failed", failed)
	}
}
