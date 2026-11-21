package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"kindergarten-server/internal/domain"
)

type TeacherInvitationRepo struct {
	db *gorm.DB
}

func NewTeacherInvitationRepo(db *gorm.DB) *TeacherInvitationRepo {
	return &TeacherInvitationRepo{db: db}
}

// ErrTeacherInvitationNotPending 邀请已被使用（或已被另一账号消费）。
//
// 绑定流程需要区分「不存在/过期」与「已被使用」：前者客户端提示重新索取
// 邀请码，后者要提示联系园所。仓储层只表达状态事实，映射留给 service。
var ErrTeacherInvitationNotPending = errors.New("TEACHER_INVITATION_NOT_PENDING")

// FindByToken 按 token 读取邀请，**不过滤状态与有效期**。
//
// 绑定流程必须能区分「不存在」「已使用」「已过期」三种失败；旧实现把三者
// 都过滤成 nil，调用方无法给出可行动的提示（#168）。
func (r *TeacherInvitationRepo) FindByToken(ctx context.Context, token string) (*domain.TeacherInvitation, error) {
	var inv domain.TeacherInvitation
	err := r.db.WithContext(ctx).
		Where("token = ?", token).
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

// MarkUsed 标记邀请已使用，并保证「邀请指向的账号 + 当前客户端身份」成立。
//
// 绑定后的身份由 #165 的合并服务从待绑定账号迁移而来；这里再做一次
// OnConflict DoNothing 的补写，覆盖「账号已绑定、邀请仍待使用」的修复场景。
// 不再写 users.open_id / union_id：那是已废弃的单身份字段，多客户端下会互相覆盖。
// 只更新仍处于待使用状态的邀请；并发消费时返回 ErrTeacherInvitationNotPending。
func (r *TeacherInvitationRepo) MarkUsed(ctx context.Context, id, userID uint64, appID, openID string) error {
	if id == 0 || userID == 0 || appID == "" || openID == "" {
		return errors.New("teacherInvitation MarkUsed: id/user/app/openid are required")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&domain.TeacherInvitation{}).
			Where("id = ? AND status = ?", id, domain.TeacherInvitationPending).
			Updates(map[string]any{
				"status":  domain.TeacherInvitationUsed,
				"user_id": userID,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			var inv domain.TeacherInvitation
			if err := tx.First(&inv, id).Error; err != nil {
				return err
			}
			if inv.Status != domain.TeacherInvitationUsed || inv.UserID != userID {
				return ErrTeacherInvitationNotPending
			}
		}

		// 身份已由合并服务写入时是 no-op；DoNothing 保证绝不把身份从别的账号抢走。
		return tx.Clauses(clause.OnConflict{DoNothing: true}).
			Create(&domain.UserIdentity{UserID: userID, AppID: appID, OpenID: openID}).Error
	})
}

// DeleteByToken 作废指定 token（供重新生成时调用）。
func (r *TeacherInvitationRepo) DeleteByToken(ctx context.Context, token string) error {
	return r.db.WithContext(ctx).
		Where("token = ?", token).
		Delete(&domain.TeacherInvitation{}).Error
}
