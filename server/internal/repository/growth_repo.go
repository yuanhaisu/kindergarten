package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

type GrowthRepo struct {
	db *gorm.DB
}

func NewGrowthRepo(db *gorm.DB) *GrowthRepo {
	return &GrowthRepo{db: db}
}

// GrowthLogListRow 管理端成长日志列表行。
type GrowthLogListRow struct {
	ID             uint64    `gorm:"column:id"`
	LogDate        time.Time `gorm:"column:log_date"`
	ClassName      string    `gorm:"column:class_name"`
	Content        string    `gorm:"column:content"`
	CreatorID      uint64    `gorm:"column:creator_id"`
	CreatorNick    string    `gorm:"column:creator_nick"`
	MediaCount     int64     `gorm:"column:media_count"`
	LikesCount     int64     `gorm:"column:likes_count"`
	CommentsCount  int64     `gorm:"column:comments_count"`
	AuditStatusRaw *int8     `gorm:"column:audit_status_raw"`
	CreatedAt      time.Time `gorm:"column:created_at"`
}

func (r *GrowthRepo) baseLogListQuery(
	kindergartenID uint64,
	classID *uint64,
	teacherID *uint64,
	dateStr string,
) *gorm.DB {
	q := r.db.Table("growth_daily_logs AS g").
		Joins("INNER JOIN classes AS c ON c.id = g.class_id AND c.deleted_at IS NULL").
		Joins("INNER JOIN users AS u ON u.id = g.creator_id AND u.deleted_at IS NULL").
		Where("g.kindergarten_id = ? AND g.deleted_at IS NULL", kindergartenID)
	if classID != nil && *classID > 0 {
		q = q.Where("g.class_id = ?", *classID)
	}
	if teacherID != nil && *teacherID > 0 {
		q = q.Where("g.creator_id = ?", *teacherID)
	}
	if dateStr != "" {
		q = q.Where("DATE(g.date) = DATE(?)", dateStr)
	}
	return q
}

func (r *GrowthRepo) ListLogs(
	ctx context.Context,
	kindergartenID uint64,
	classID *uint64,
	teacherID *uint64,
	dateStr string,
	offset, limit int,
) ([]GrowthLogListRow, int64, error) {
	base := r.baseLogListQuery(kindergartenID, classID, teacherID, dateStr)
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	sel := base.Session(&gorm.Session{}).
		Select(`g.id,
			g.date AS log_date,
			c.name AS class_name,
			g.content,
			g.creator_id,
			u.nickname AS creator_nick,
			(SELECT COUNT(*) FROM growth_media m WHERE m.log_id = g.id AND m.deleted_at IS NULL) AS media_count,
			(SELECT COUNT(*) FROM growth_likes lk WHERE lk.log_id = g.id AND lk.deleted_at IS NULL) AS likes_count,
			(SELECT COUNT(*) FROM growth_comments cm WHERE cm.log_id = g.id AND cm.deleted_at IS NULL) AS comments_count,
			(SELECT ca.status FROM content_audits ca
			 WHERE ca.kindergarten_id = g.kindergarten_id
			   AND ca.target_type = 'growth' AND ca.target_id = g.id
			   AND ca.deleted_at IS NULL
			 ORDER BY ca.id DESC LIMIT 1) AS audit_status_raw,
			g.created_at`).
		Order("g.created_at DESC, g.id DESC").
		Offset(offset).Limit(limit)
	var rows []GrowthLogListRow
	err := sel.Scan(&rows).Error
	return rows, total, err
}

// FindLogByID 园所内成长日志（未删）。
func (r *GrowthRepo) FindLogByID(ctx context.Context, kindergartenID, logID uint64) (*domain.GrowthDailyLog, error) {
	var log domain.GrowthDailyLog
	err := r.db.WithContext(ctx).
		Preload("Class").
		Preload("Creator").
		Where("kindergarten_id = ? AND id = ? AND deleted_at IS NULL", kindergartenID, logID).
		First(&log).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &log, nil
}

// FindLatestAudit 取该日志最新一条审核记录。
func (r *GrowthRepo) FindLatestAudit(ctx context.Context, kindergartenID, logID uint64) (*domain.ContentAudit, error) {
	var ca domain.ContentAudit
	err := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND target_type = ? AND target_id = ? AND deleted_at IS NULL",
			kindergartenID, domain.AuditTargetGrowth, logID).
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

// ListMediaByLogID 日志下的媒体（未删），按 sort_num、id。
func (r *GrowthRepo) ListMediaByLogID(ctx context.Context, logID uint64) ([]domain.GrowthMedia, error) {
	var list []domain.GrowthMedia
	err := r.db.WithContext(ctx).
		Where("log_id = ? AND deleted_at IS NULL", logID).
		Order("sort_num ASC, id ASC").
		Find(&list).Error
	return list, err
}

func (r *GrowthRepo) SoftDeleteLog(ctx context.Context, kindergartenID, logID uint64) (int64, error) {
	res := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND id = ? AND deleted_at IS NULL", kindergartenID, logID).
		Delete(&domain.GrowthDailyLog{})
	return res.RowsAffected, res.Error
}

// SaveContentAudit 新建或更新同目标审核记录（单条覆盖）。
func (r *GrowthRepo) SaveContentAudit(ctx context.Context, ca *domain.ContentAudit) error {
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

// --- admin stats ---

func (r *GrowthRepo) CountLogsInDateRange(ctx context.Context, kindergartenID uint64, start, end time.Time) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.GrowthDailyLog{}).
		Where("kindergarten_id = ? AND deleted_at IS NULL", kindergartenID).
		Where("date >= ? AND date <= ?", start, end).
		Count(&n).Error
	return n, err
}

func (r *GrowthRepo) CountMediaInLogDateRange(ctx context.Context, kindergartenID uint64, start, end time.Time) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Table("growth_media AS m").
		Joins("INNER JOIN growth_daily_logs g ON g.id = m.log_id AND g.deleted_at IS NULL").
		Where("g.kindergarten_id = ? AND m.deleted_at IS NULL", kindergartenID).
		Where("g.date >= ? AND g.date <= ?", start, end).
		Count(&n).Error
	return n, err
}

func (r *GrowthRepo) CountLikesInLogDateRange(ctx context.Context, kindergartenID uint64, start, end time.Time) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Table("growth_likes AS lk").
		Joins("INNER JOIN growth_daily_logs g ON g.id = lk.log_id AND g.deleted_at IS NULL").
		Where("g.kindergarten_id = ? AND lk.deleted_at IS NULL", kindergartenID).
		Where("g.date >= ? AND g.date <= ?", start, end).
		Count(&n).Error
	return n, err
}

func (r *GrowthRepo) CountCommentsInLogDateRange(ctx context.Context, kindergartenID uint64, start, end time.Time) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Table("growth_comments AS cm").
		Joins("INNER JOIN growth_daily_logs g ON g.id = cm.log_id AND g.deleted_at IS NULL").
		Where("g.kindergarten_id = ? AND cm.deleted_at IS NULL", kindergartenID).
		Where("g.date >= ? AND g.date <= ?", start, end).
		Count(&n).Error
	return n, err
}

// GrowthClassStatRow 按班统计。
type GrowthClassStatRow struct {
	ClassName string `gorm:"column:class_name"`
	Teacher   string `gorm:"column:teacher"`
	Logs      int64  `gorm:"column:logs"`
	Media     int64  `gorm:"column:media"`
}

func (r *GrowthRepo) StatsByClass(ctx context.Context, kindergartenID uint64, start, end time.Time) ([]GrowthClassStatRow, error) {
	var rows []GrowthClassStatRow
	err := r.db.WithContext(ctx).Raw(`
SELECT c.name AS class_name,
       COALESCE((
         SELECT u.nickname FROM class_teachers ct
         INNER JOIN users u ON u.id = ct.user_id AND u.deleted_at IS NULL
         WHERE ct.class_id = c.id AND ct.title = 'head' AND ct.deleted_at IS NULL
         ORDER BY ct.id ASC LIMIT 1
       ), '') AS teacher,
       COUNT(DISTINCT g.id) AS logs,
       COUNT(DISTINCT m.id) AS media
FROM classes c
LEFT JOIN growth_daily_logs g ON g.class_id = c.id
  AND g.kindergarten_id = ? AND g.deleted_at IS NULL
  AND g.date >= ? AND g.date <= ?
LEFT JOIN growth_media m ON m.log_id = g.id AND m.deleted_at IS NULL
WHERE c.kindergarten_id = ? AND c.deleted_at IS NULL
GROUP BY c.id, c.name
ORDER BY logs DESC, c.id ASC
`, kindergartenID, start, end, kindergartenID).Scan(&rows).Error
	return rows, err
}

// GrowthDailyTrendRow 按日趋势。
type GrowthDailyTrendRow struct {
	Day   time.Time `gorm:"column:day"`
	Logs  int64     `gorm:"column:logs"`
	Media int64     `gorm:"column:media"`
}

func (r *GrowthRepo) StatsDailyTrend(ctx context.Context, kindergartenID uint64, start, end time.Time) ([]GrowthDailyTrendRow, error) {
	var rows []GrowthDailyTrendRow
	err := r.db.WithContext(ctx).Raw(`
SELECT g.date AS day,
       COUNT(DISTINCT g.id) AS logs,
       COUNT(DISTINCT m.id) AS media
FROM growth_daily_logs g
LEFT JOIN growth_media m ON m.log_id = g.id AND m.deleted_at IS NULL
WHERE g.kindergarten_id = ? AND g.deleted_at IS NULL
  AND g.date >= ? AND g.date <= ?
GROUP BY g.date
ORDER BY g.date ASC
`, kindergartenID, start, end).Scan(&rows).Error
	return rows, err
}

func (r *GrowthRepo) CountLikesByLogID(ctx context.Context, logID uint64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.GrowthLike{}).
		Where("log_id = ? AND deleted_at IS NULL", logID).
		Count(&n).Error
	return n, err
}

func (r *GrowthRepo) CountCommentsByLogID(ctx context.Context, logID uint64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.GrowthComment{}).
		Where("log_id = ? AND deleted_at IS NULL", logID).
		Count(&n).Error
	return n, err
}

// GrowthLogExportRow 成长记录导出用行。
type GrowthLogExportRow struct {
	LogDate   time.Time `gorm:"column:log_date"`
	ClassName string    `gorm:"column:class_name"`
	Content   string    `gorm:"column:content"`
	Creator   string    `gorm:"column:creator"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

// ListLogsForExportDateRange 按日志日期 [fromDay, toDayExclusive) 导出列表。
func (r *GrowthRepo) ListLogsForExportDateRange(
	ctx context.Context,
	kindergartenID uint64,
	classID *uint64,
	fromDay, toDayExclusive time.Time,
) ([]GrowthLogExportRow, error) {
	q := r.db.WithContext(ctx).Table("growth_daily_logs AS g").
		Select(`g.date AS log_date, c.name AS class_name, g.content, u.nickname AS creator, g.created_at`).
		Joins("INNER JOIN classes AS c ON c.id = g.class_id AND c.deleted_at IS NULL").
		Joins("INNER JOIN users AS u ON u.id = g.creator_id AND u.deleted_at IS NULL").
		Where("g.kindergarten_id = ? AND g.deleted_at IS NULL", kindergartenID).
		Where("g.date >= ? AND g.date < ?", fromDay, toDayExclusive)
	if classID != nil && *classID > 0 {
		q = q.Where("g.class_id = ?", *classID)
	}
	var rows []GrowthLogExportRow
	err := q.Order("g.created_at DESC, g.id DESC").Scan(&rows).Error
	return rows, err
}

// FindUserDisplay 取用户昵称与头像（未删用户）。
func (r *GrowthRepo) FindUserDisplay(ctx context.Context, userID uint64) (nickname, avatar string, ok bool) {
	var u domain.User
	err := r.db.WithContext(ctx).Select("id", "nickname", "avatar").
		Where("id = ? AND deleted_at IS NULL", userID).
		First(&u).Error
	if err != nil {
		return "", "", false
	}
	return u.Nickname, u.Avatar, true
}
