package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

type AuditRepo struct {
	db *gorm.DB
}

func NewAuditRepo(db *gorm.DB) *AuditRepo {
	return &AuditRepo{db: db}
}

// EnsurePendingAudits 在园所开启对应 need_audit 时，为尚无审核记录的通知 / 成长日志补建待审行（幂等）。
func (r *AuditRepo) EnsurePendingAudits(ctx context.Context, kindergartenID uint64, noticeNeed, growthNeed bool) error {
	if kindergartenID == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if noticeNeed {
			q := `
INSERT INTO content_audits (created_at, updated_at, kindergarten_id, target_type, target_id, submitter_id, status, reject_reason)
SELECT NOW(3), NOW(3), n.kindergarten_id, ?, n.id, n.creator_id, 0, ''
FROM notices n
WHERE n.kindergarten_id = ? AND n.deleted_at IS NULL AND n.status IN (1, 2)
AND NOT EXISTS (
  SELECT 1 FROM content_audits ca
  WHERE ca.kindergarten_id = n.kindergarten_id AND ca.target_type = ? AND ca.target_id = n.id AND ca.deleted_at IS NULL
)`
			if err := tx.Exec(q, string(domain.AuditTargetNotice), kindergartenID, string(domain.AuditTargetNotice)).Error; err != nil {
				return err
			}
		}
		if growthNeed {
			q := `
INSERT INTO content_audits (created_at, updated_at, kindergarten_id, target_type, target_id, submitter_id, status, reject_reason)
SELECT NOW(3), NOW(3), g.kindergarten_id, ?, g.id, g.creator_id, 0, ''
FROM growth_daily_logs g
WHERE g.kindergarten_id = ? AND g.deleted_at IS NULL
AND NOT EXISTS (
  SELECT 1 FROM content_audits ca
  WHERE ca.kindergarten_id = g.kindergarten_id AND ca.target_type = ? AND ca.target_id = g.id AND ca.deleted_at IS NULL
)`
			if err := tx.Exec(q, string(domain.AuditTargetGrowth), kindergartenID, string(domain.AuditTargetGrowth)).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// AuditListScan 列表查询扫描行。
type AuditListScan struct {
	CAID          uint64     `gorm:"column:ca_id"`
	TargetType    string     `gorm:"column:target_type"`
	TargetID      uint64     `gorm:"column:target_id"`
	Status        int8       `gorm:"column:status"`
	SubmitterID   uint64     `gorm:"column:submitter_id"`
	SubmitterNick string     `gorm:"column:submitter_nick"`
	ClassName     string     `gorm:"column:class_name"`
	NoticeTitle   string     `gorm:"column:notice_title"`
	NoticeContent string     `gorm:"column:notice_content"`
	GrowthContent string     `gorm:"column:growth_content"`
	GrowthDate    *time.Time `gorm:"column:growth_date"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
}

func (r *AuditRepo) baseListQuery(kindergartenID uint64, targetType, status string) *gorm.DB {
	q := r.db.Table("content_audits AS ca").
		Joins(`LEFT JOIN notices n ON ca.target_type = 'notice' AND ca.target_id = n.id AND n.deleted_at IS NULL`).
		Joins(`LEFT JOIN growth_daily_logs g ON ca.target_type = 'growth' AND ca.target_id = g.id AND g.deleted_at IS NULL`).
		Joins(`LEFT JOIN users u ON u.id = ca.submitter_id AND u.deleted_at IS NULL`).
		Joins(`LEFT JOIN classes c ON c.id = COALESCE(n.class_id, g.class_id) AND c.deleted_at IS NULL`).
		Where("ca.kindergarten_id = ? AND ca.deleted_at IS NULL", kindergartenID).
		Where(`(ca.target_type = 'comment') OR (ca.target_type = 'notice' AND n.id IS NOT NULL) OR (ca.target_type = 'growth' AND g.id IS NOT NULL)`)

	if tt := strings.TrimSpace(targetType); tt != "" {
		q = q.Where("ca.target_type = ?", tt)
	}
	switch strings.TrimSpace(status) {
	case "pending":
		q = q.Where("ca.status = ?", domain.AuditStatusPending)
	case "passed":
		q = q.Where("ca.status = ?", domain.AuditStatusApproved)
	case "rejected":
		q = q.Where("ca.status = ?", domain.AuditStatusRejected)
	}
	return q
}

func (r *AuditRepo) ListAudits(
	ctx context.Context,
	kindergartenID uint64,
	targetType, status string,
	offset, limit int,
) ([]AuditListScan, int64, error) {
	base := r.baseListQuery(kindergartenID, targetType, status)
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	sel := base.Session(&gorm.Session{}).
		Select(`ca.id AS ca_id,
			ca.target_type,
			ca.target_id,
			ca.status,
			ca.submitter_id,
			COALESCE(u.nickname, '') AS submitter_nick,
			COALESCE(c.name, '') AS class_name,
			COALESCE(n.title, '') AS notice_title,
			COALESCE(n.content, '') AS notice_content,
			COALESCE(g.content, '') AS growth_content,
			g.date AS growth_date,
			ca.created_at`).
		Order("ca.created_at DESC, ca.id DESC").
		Offset(offset).Limit(limit)
	var rows []AuditListScan
	err := sel.WithContext(ctx).Scan(&rows).Error
	return rows, total, err
}

func (r *AuditRepo) FindByID(ctx context.Context, kindergartenID, auditID uint64) (*domain.ContentAudit, error) {
	var ca domain.ContentAudit
	err := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND id = ? AND deleted_at IS NULL", kindergartenID, auditID).
		First(&ca).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &ca, nil
}

func (r *AuditRepo) UpdateAuditDecision(
	ctx context.Context,
	kindergartenID, auditID uint64,
	status domain.AuditStatus,
	reviewerID uint64,
	reviewedAt time.Time,
	rejectReason string,
) error {
	res := r.db.WithContext(ctx).Model(&domain.ContentAudit{}).
		Where("kindergarten_id = ? AND id = ? AND deleted_at IS NULL", kindergartenID, auditID).
		Updates(map[string]interface{}{
			"status":          status,
			"reviewer_id":     reviewerID,
			"reviewed_at":     reviewedAt,
			"reject_reason":   rejectReason,
			"updated_at":      reviewedAt,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// AuditStatusCount 按 target_type + status 聚合（仅 notice / growth / comment）。
type AuditStatusCount struct {
	TargetType string `gorm:"column:target_type"`
	Status     int8   `gorm:"column:status"`
	Cnt        int64  `gorm:"column:cnt"`
}

func (r *AuditRepo) CountByTypeAndStatus(ctx context.Context, kindergartenID uint64) ([]AuditStatusCount, error) {
	var rows []AuditStatusCount
	err := r.db.WithContext(ctx).Model(&domain.ContentAudit{}).
		Select("target_type, status, COUNT(*) AS cnt").
		Where("kindergarten_id = ? AND deleted_at IS NULL", kindergartenID).
		Group("target_type, status").
		Scan(&rows).Error
	return rows, err
}

func (r *AuditRepo) CountPending(ctx context.Context, kindergartenID uint64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.ContentAudit{}).
		Where("kindergarten_id = ? AND deleted_at IS NULL AND status = ?", kindergartenID, domain.AuditStatusPending).
		Count(&n).Error
	return n, err
}

func (r *AuditRepo) CountReviewedBetween(ctx context.Context, kindergartenID uint64, status domain.AuditStatus, start, end time.Time) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.ContentAudit{}).
		Where("kindergarten_id = ? AND deleted_at IS NULL AND status = ?", kindergartenID, status).
		Where("reviewed_at IS NOT NULL AND reviewed_at >= ? AND reviewed_at < ?", start, end).
		Count(&n).Error
	return n, err
}

func (r *AuditRepo) CountReviewedInMonth(ctx context.Context, kindergartenID uint64, start, end time.Time) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.ContentAudit{}).
		Where("kindergarten_id = ? AND deleted_at IS NULL", kindergartenID).
		Where("status IN ?", []domain.AuditStatus{domain.AuditStatusApproved, domain.AuditStatusRejected}).
		Where("reviewed_at IS NOT NULL AND reviewed_at >= ? AND reviewed_at < ?", start, end).
		Count(&n).Error
	return n, err
}
