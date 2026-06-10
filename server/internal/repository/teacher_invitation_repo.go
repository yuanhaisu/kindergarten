package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

type TeacherInvitationRepo struct {
	db *gorm.DB
}

func NewTeacherInvitationRepo(db *gorm.DB) *TeacherInvitationRepo {
	return &TeacherInvitationRepo{db: db}
}

// FindByToken 查找待使用且未过期的邀请。
func (r *TeacherInvitationRepo) FindByToken(ctx context.Context, token string) (*domain.TeacherInvitation, error) {
	var inv domain.TeacherInvitation
	err := r.db.WithContext(ctx).
		Where("token = ? AND status = ? AND expires_at > ?", token, domain.TeacherInvitationPending, time.Now()).
		First(&inv).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &inv, err
}

// FindPendingByUserID 查找某教师待使用的邀请。
func (r *TeacherInvitationRepo) FindPendingByUserID(ctx context.Context, userID, kindergartenID uint64) (*domain.TeacherInvitation, error) {
	var inv domain.TeacherInvitation
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND kindergarten_id = ? AND status = ? AND expires_at > ?",
			userID, kindergartenID, domain.TeacherInvitationPending, time.Now()).
		Order("id DESC").
		First(&inv).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &inv, err
}

func (r *TeacherInvitationRepo) Create(ctx context.Context, inv *domain.TeacherInvitation) error {
	return r.db.WithContext(ctx).Create(inv).Error
}

// MarkUsed 标记邀请已使用，同时将 OpenID 回填到对应用户。
func (r *TeacherInvitationRepo) MarkUsed(ctx context.Context, id, userID uint64, openID, unionID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&domain.TeacherInvitation{}).
			Where("id = ?", id).
			Updates(map[string]any{"status": domain.TeacherInvitationUsed}).Error; err != nil {
			return err
		}
		updates := map[string]any{"open_id": openID}
		if unionID != "" {
			updates["union_id"] = unionID
		}
		return tx.Model(&domain.User{}).Where("id = ?", userID).Updates(updates).Error
	})
}

// DeleteByToken 作废指定 token（供重新生成时调用）。
func (r *TeacherInvitationRepo) DeleteByToken(ctx context.Context, token string) error {
	return r.db.WithContext(ctx).
		Where("token = ?", token).
		Delete(&domain.TeacherInvitation{}).Error
}
