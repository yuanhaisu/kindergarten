package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

type AnnouncementRepo struct {
	db *gorm.DB
}

func NewAnnouncementRepo(db *gorm.DB) *AnnouncementRepo {
	return &AnnouncementRepo{db: db}
}

func (r *AnnouncementRepo) ListAnnouncements(
	ctx context.Context,
	kindergartenID uint64,
	status *domain.AnnouncementStatus,
	offset, limit int,
) ([]domain.Announcement, int64, error) {
	q := r.db.WithContext(ctx).Model(&domain.Announcement{}).
		Where("kindergarten_id = ? AND deleted_at IS NULL", kindergartenID)
	if status != nil {
		q = q.Where("status = ?", *status)
	}
	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []domain.Announcement
	err := q.Session(&gorm.Session{}).
		Preload("Creator").
		Order("is_top DESC, published_at DESC, created_at DESC, id DESC").
		Offset(offset).Limit(limit).
		Find(&list).Error
	return list, total, err
}

func (r *AnnouncementRepo) FindAnnouncementByID(ctx context.Context, kindergartenID, id uint64) (*domain.Announcement, error) {
	var a domain.Announcement
	err := r.db.WithContext(ctx).
		Preload("Creator").
		Where("kindergarten_id = ? AND id = ? AND deleted_at IS NULL", kindergartenID, id).
		First(&a).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func (r *AnnouncementRepo) ListAttachmentsByAnnouncementID(ctx context.Context, announcementID uint64) ([]domain.AnnouncementAttachment, error) {
	var list []domain.AnnouncementAttachment
	err := r.db.WithContext(ctx).
		Where("announcement_id = ?", announcementID).
		Order("sort_num ASC, id ASC").
		Find(&list).Error
	return list, err
}

func (r *AnnouncementRepo) CreateWithAttachments(
	ctx context.Context,
	a *domain.Announcement,
	atts []domain.AnnouncementAttachment,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(a).Error; err != nil {
			return err
		}
		if len(atts) == 0 {
			return nil
		}
		for i := range atts {
			atts[i].AnnouncementID = a.ID
		}
		return tx.Create(&atts).Error
	})
}

func (r *AnnouncementRepo) UpdateColumns(ctx context.Context, kindergartenID, id uint64, patch map[string]interface{}) (int64, error) {
	res := r.db.WithContext(ctx).Model(&domain.Announcement{}).
		Where("kindergarten_id = ? AND id = ? AND deleted_at IS NULL", kindergartenID, id).
		Updates(patch)
	return res.RowsAffected, res.Error
}

func (r *AnnouncementRepo) SoftDeleteAnnouncement(ctx context.Context, kindergartenID, id uint64) (int64, error) {
	res := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND id = ? AND deleted_at IS NULL", kindergartenID, id).
		Delete(&domain.Announcement{})
	return res.RowsAffected, res.Error
}

func (r *AnnouncementRepo) DeleteAttachmentsByAnnouncementID(ctx context.Context, announcementID uint64) error {
	return r.db.WithContext(ctx).Unscoped().
		Where("announcement_id = ?", announcementID).
		Delete(&domain.AnnouncementAttachment{}).Error
}

func (r *AnnouncementRepo) CreateAttachments(ctx context.Context, list []domain.AnnouncementAttachment) error {
	if len(list) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&list).Error
}

func (r *AnnouncementRepo) ReplaceAttachments(
	ctx context.Context,
	announcementID uint64,
	atts []domain.AnnouncementAttachment,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("announcement_id = ?", announcementID).Delete(&domain.AnnouncementAttachment{}).Error; err != nil {
			return err
		}
		if len(atts) == 0 {
			return nil
		}
		for i := range atts {
			atts[i].AnnouncementID = announcementID
		}
		return tx.Create(&atts).Error
	})
}
