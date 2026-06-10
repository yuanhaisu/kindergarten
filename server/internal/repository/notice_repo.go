package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

type NoticeRepo struct {
	db *gorm.DB
}

func NewNoticeRepo(db *gorm.DB) *NoticeRepo {
	return &NoticeRepo{db: db}
}

// NoticeListRow 管理端通知列表行。
type NoticeListRow struct {
	ID            uint64     `gorm:"column:id"`
	Category      string     `gorm:"column:category"`
	Title         string     `gorm:"column:title"`
	ClassName     string     `gorm:"column:class_name"`
	ClassID       uint64     `gorm:"column:class_id"`
	CreatorID     uint64     `gorm:"column:creator_id"`
	CreatorNick   string     `gorm:"column:creator_nick"`
	Status        int8       `gorm:"column:status"`
	PublishedAt   *time.Time `gorm:"column:published_at"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	ReadUsers     int64      `gorm:"column:read_users"`
	ParentTotal   int64      `gorm:"column:parent_total"`
}

func (r *NoticeRepo) baseNoticeListQuery(
	kindergartenID uint64,
	classID *uint64,
	category string,
	status *int8,
	keyword string,
) *gorm.DB {
	q := r.db.Table("notices AS n").
		Joins("INNER JOIN classes AS c ON c.id = n.class_id AND c.deleted_at IS NULL").
		Joins("INNER JOIN users AS u ON u.id = n.creator_id AND u.deleted_at IS NULL").
		Where("n.kindergarten_id = ? AND n.deleted_at IS NULL", kindergartenID)
	if classID != nil && *classID > 0 {
		q = q.Where("n.class_id = ?", *classID)
	}
	if category != "" {
		q = q.Where("n.category = ?", category)
	}
	if status != nil {
		q = q.Where("n.status = ?", *status)
	}
	if kw := strings.TrimSpace(keyword); kw != "" {
		q = q.Where("n.title LIKE ?", "%"+kw+"%")
	}
	return q
}

func (r *NoticeRepo) ListNotices(
	ctx context.Context,
	kindergartenID uint64,
	classID *uint64,
	category string,
	status *int8,
	keyword string,
	offset, limit int,
) ([]NoticeListRow, int64, error) {
	base := r.baseNoticeListQuery(kindergartenID, classID, category, status, keyword)
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	sel := base.Session(&gorm.Session{}).
		Select(`n.id,
			n.category,
			n.title,
			c.name AS class_name,
			n.class_id,
			n.creator_id,
			u.nickname AS creator_nick,
			n.status,
			n.published_at,
			n.created_at,
			(SELECT COUNT(DISTINCT rl.user_id) FROM notice_read_logs rl WHERE rl.notice_id = n.id) AS read_users,
			(SELECT COUNT(DISTINCT sp.user_id) FROM student_parents sp
			 INNER JOIN students s ON s.id = sp.student_id AND s.deleted_at IS NULL
			 WHERE s.class_id = n.class_id AND s.kindergarten_id = n.kindergarten_id) AS parent_total`).
		Order("n.created_at DESC, n.id DESC").
		Offset(offset).Limit(limit)
	var rows []NoticeListRow
	err := sel.Scan(&rows).Error
	return rows, total, err
}

// FindNoticeByID 园所内通知（未删），预加载班级与创建者。
func (r *NoticeRepo) FindNoticeByID(ctx context.Context, kindergartenID, noticeID uint64) (*domain.Notice, error) {
	var n domain.Notice
	err := r.db.WithContext(ctx).
		Preload("Class").
		Preload("Creator").
		Where("kindergarten_id = ? AND id = ? AND deleted_at IS NULL", kindergartenID, noticeID).
		First(&n).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &n, nil
}

func (r *NoticeRepo) ListAttachmentsByNoticeID(ctx context.Context, noticeID uint64) ([]domain.NoticeAttachment, error) {
	var list []domain.NoticeAttachment
	err := r.db.WithContext(ctx).
		Where("notice_id = ?", noticeID).
		Order("sort_num ASC, id ASC").
		Find(&list).Error
	return list, err
}

func (r *NoticeRepo) FindLatestAudit(ctx context.Context, kindergartenID, noticeID uint64) (*domain.ContentAudit, error) {
	var ca domain.ContentAudit
	err := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND target_type = ? AND target_id = ? AND deleted_at IS NULL",
			kindergartenID, domain.AuditTargetNotice, noticeID).
		Order("id DESC").
		First(&ca).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &ca, nil
}

// SaveContentAudit 新建或更新同目标审核记录（单条覆盖）。
func (r *NoticeRepo) SaveContentAudit(ctx context.Context, ca *domain.ContentAudit) error {
	var existing domain.ContentAudit
	err := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND target_type = ? AND target_id = ? AND deleted_at IS NULL",
			ca.KindergartenID, ca.TargetType, ca.TargetID).
		First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.db.WithContext(ctx).Create(ca).Error
		}
		return err
	}
	existing.ReviewerID = ca.ReviewerID
	existing.Status = ca.Status
	existing.ReviewedAt = ca.ReviewedAt
	existing.RejectReason = ca.RejectReason
	return r.db.WithContext(ctx).Save(&existing).Error
}

// FindUserDisplay 取用户昵称与头像（未删用户）。
func (r *NoticeRepo) FindUserDisplay(ctx context.Context, userID uint64) (nickname, avatar string, ok bool) {
	var u domain.User
	err := r.db.WithContext(ctx).Select("id", "nickname", "avatar").
		Where("id = ? AND deleted_at IS NULL", userID).
		First(&u).Error
	if err != nil {
		return "", "", false
	}
	return u.Nickname, u.Avatar, true
}

// NoticeReadStats 某通知下的已读人数、确认人数、班级家长总数（去重 user_id）。
func (r *NoticeRepo) NoticeReadStats(ctx context.Context, kindergartenID, classID, noticeID uint64) (readUsers, confirmed, parentTotal int64, err error) {
	err = r.db.WithContext(ctx).Raw(`
SELECT COALESCE(COUNT(DISTINCT sp.user_id), 0) AS parent_total
FROM student_parents sp
INNER JOIN students s ON s.id = sp.student_id AND s.deleted_at IS NULL
WHERE s.class_id = ? AND s.kindergarten_id = ?
`, classID, kindergartenID).Scan(&parentTotal).Error
	if err != nil {
		return 0, 0, 0, err
	}
	err = r.db.WithContext(ctx).Raw(
		`SELECT COUNT(DISTINCT user_id) FROM notice_read_logs WHERE notice_id = ?`,
		noticeID,
	).Scan(&readUsers).Error
	if err != nil {
		return 0, 0, 0, err
	}
	err = r.db.WithContext(ctx).Model(&domain.NoticeReadLog{}).
		Where("notice_id = ? AND confirmed_at IS NOT NULL", noticeID).
		Count(&confirmed).Error
	if err != nil {
		return 0, 0, 0, err
	}
	return readUsers, confirmed, parentTotal, nil
}

func (r *NoticeRepo) IsClassInKindergarten(ctx context.Context, kindergartenID, classID uint64) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.Class{}).
		Where("kindergarten_id = ? AND id = ? AND deleted_at IS NULL", kindergartenID, classID).
		Count(&n).Error
	return n > 0, err
}

func (r *NoticeRepo) CreateNotice(ctx context.Context, n *domain.Notice) error {
	return r.db.WithContext(ctx).Create(n).Error
}

func (r *NoticeRepo) CreateAttachments(ctx context.Context, list []domain.NoticeAttachment) error {
	if len(list) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&list).Error
}

func (r *NoticeRepo) UpdateNoticeFields(ctx context.Context, kindergartenID, noticeID uint64, patch map[string]interface{}) (int64, error) {
	res := r.db.WithContext(ctx).Model(&domain.Notice{}).
		Where("kindergarten_id = ? AND id = ? AND deleted_at IS NULL", kindergartenID, noticeID).
		Updates(patch)
	return res.RowsAffected, res.Error
}

func (r *NoticeRepo) SoftDeleteNotice(ctx context.Context, kindergartenID, noticeID uint64) (int64, error) {
	res := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND id = ? AND deleted_at IS NULL", kindergartenID, noticeID).
		Delete(&domain.Notice{})
	return res.RowsAffected, res.Error
}

func (r *NoticeRepo) DeleteAttachmentsByNoticeID(ctx context.Context, noticeID uint64) error {
	return r.db.WithContext(ctx).Unscoped().
		Where("notice_id = ?", noticeID).
		Delete(&domain.NoticeAttachment{}).Error
}
