package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"kindergarten-server/internal/domain"
)

type LeaveRepo struct {
	db *gorm.DB
}

func NewLeaveRepo(db *gorm.DB) *LeaveRepo {
	return &LeaveRepo{db: db}
}

// LeaveListRow 请假列表 JOIN 结果。
type LeaveListRow struct {
	ID            uint64             `gorm:"column:id"`
	StudentID     uint64             `gorm:"column:student_id"`
	StudentName   string             `gorm:"column:student_name"`
	ClassName     string             `gorm:"column:class_name"`
	ApplicantID   uint64             `gorm:"column:applicant_id"`
	ApplicantNick string             `gorm:"column:applicant_nick"`
	ApplicantRel  string             `gorm:"column:applicant_relation"`
	Reason        string             `gorm:"column:reason"`
	StartDate     time.Time          `gorm:"column:start_date"`
	EndDate       time.Time          `gorm:"column:end_date"`
	Days          int                `gorm:"column:days"`
	Status        domain.LeaveStatus `gorm:"column:status"`
	ReviewerID    *uint64            `gorm:"column:reviewer_id"`
	ReviewerNick  string             `gorm:"column:reviewer_nick"`
	ReviewedAt    *time.Time         `gorm:"column:reviewed_at"`
	CreatedAt     time.Time          `gorm:"column:created_at"`
}

func (r *LeaveRepo) baseListQuery(
	kindergartenID uint64,
	classID *uint64,
	status *domain.LeaveStatus,
	rangeStart, rangeEnd *time.Time,
) *gorm.DB {
	q := r.db.Table("leave_requests AS lr").
		Joins("INNER JOIN students AS s ON s.id = lr.student_id AND s.deleted_at IS NULL").
		Joins("INNER JOIN classes AS c ON c.id = lr.class_id AND c.deleted_at IS NULL").
		Joins("INNER JOIN users AS u ON u.id = lr.applicant_id").
		Joins("LEFT JOIN student_parents AS sp ON sp.student_id = lr.student_id AND sp.user_id = lr.applicant_id").
		Joins("LEFT JOIN users AS rv ON rv.id = lr.reviewer_id").
		Where("lr.kindergarten_id = ? AND lr.deleted_at IS NULL", kindergartenID)
	if classID != nil && *classID > 0 {
		q = q.Where("lr.class_id = ?", *classID)
	}
	if status != nil {
		q = q.Where("lr.status = ?", *status)
	}
	if rangeStart != nil && rangeEnd != nil {
		q = q.Where("lr.start_date <= ? AND lr.end_date >= ?", *rangeEnd, *rangeStart)
	}
	return q
}

func (r *LeaveRepo) List(
	ctx context.Context,
	kindergartenID uint64,
	classID *uint64,
	status *domain.LeaveStatus,
	rangeStart, rangeEnd *time.Time,
	offset, limit int,
) ([]LeaveListRow, int64, error) {
	base := r.baseListQuery(kindergartenID, classID, status, rangeStart, rangeEnd)
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	sel := base.Session(&gorm.Session{}).
		Select(`lr.id, lr.student_id, s.name AS student_name, c.name AS class_name,
			lr.applicant_id, u.nickname AS applicant_nick,
			COALESCE(sp.relation, 'other') AS applicant_relation,
			lr.reason, lr.start_date, lr.end_date, lr.days, lr.status,
			lr.reviewer_id, COALESCE(rv.nickname, '') AS reviewer_nick, lr.reviewed_at, lr.created_at`).
		Order("lr.created_at DESC, lr.id DESC").
		Offset(offset).Limit(limit)
	var rows []LeaveListRow
	err := sel.Scan(&rows).Error
	return rows, total, err
}

// ReasonCount 原始请假原因聚合（用于统计归类）。
type ReasonCount struct {
	Reason string `gorm:"column:reason"`
	Cnt    int64  `gorm:"column:cnt"`
}

func (r *LeaveRepo) CountByStatusInRange(
	ctx context.Context,
	kindergartenID uint64,
	rangeStart, rangeEnd time.Time,
) (pending, approved, rejected, canceled, total int64, err error) {
	type row struct {
		Status domain.LeaveStatus `gorm:"column:status"`
		Cnt    int64              `gorm:"column:cnt"`
	}
	var rows []row
	err = r.db.WithContext(ctx).Model(&domain.LeaveRequest{}).
		Select("status, COUNT(*) AS cnt").
		Where("kindergarten_id = ? AND deleted_at IS NULL AND start_date <= ? AND end_date >= ?",
			kindergartenID, rangeEnd, rangeStart).
		Group("status").
		Scan(&rows).Error
	if err != nil {
		return 0, 0, 0, 0, 0, err
	}
	for _, x := range rows {
		total += x.Cnt
		switch x.Status {
		case domain.LeaveStatusPending:
			pending = x.Cnt
		case domain.LeaveStatusApproved:
			approved = x.Cnt
		case domain.LeaveStatusRejected:
			rejected = x.Cnt
		case domain.LeaveStatusCanceled:
			canceled = x.Cnt
		}
	}
	return pending, approved, rejected, canceled, total, nil
}

type ClassLeaveCount struct {
	ClassName string `gorm:"column:class_name"`
	Cnt       int64  `gorm:"column:cnt"`
}

func (r *LeaveRepo) CountByClassInRange(
	ctx context.Context,
	kindergartenID uint64,
	rangeStart, rangeEnd time.Time,
) ([]ClassLeaveCount, error) {
	var rows []ClassLeaveCount
	err := r.db.WithContext(ctx).Table("leave_requests AS lr").
		Joins("INNER JOIN classes AS c ON c.id = lr.class_id AND c.deleted_at IS NULL").
		Where("lr.kindergarten_id = ? AND lr.deleted_at IS NULL AND lr.start_date <= ? AND lr.end_date >= ?",
			kindergartenID, rangeEnd, rangeStart).
		Select("c.name AS class_name, COUNT(*) AS cnt").
		Group("c.id, c.name").
		Order("cnt DESC").
		Scan(&rows).Error
	return rows, err
}

func (r *LeaveRepo) CountReasonsInRange(
	ctx context.Context,
	kindergartenID uint64,
	rangeStart, rangeEnd time.Time,
) ([]ReasonCount, error) {
	var rows []ReasonCount
	err := r.db.WithContext(ctx).Model(&domain.LeaveRequest{}).
		Select("reason, COUNT(*) AS cnt").
		Where("kindergarten_id = ? AND deleted_at IS NULL AND start_date <= ? AND end_date >= ?",
			kindergartenID, rangeEnd, rangeStart).
		Group("reason").
		Scan(&rows).Error
	return rows, err
}

func (r *LeaveRepo) FindByID(ctx context.Context, kindergartenID, id uint64) (*domain.LeaveRequest, error) {
	var lr domain.LeaveRequest
	err := r.db.WithContext(ctx).
		Where("id = ? AND kindergarten_id = ? AND deleted_at IS NULL", id, kindergartenID).
		First(&lr).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &lr, nil
}

func (r *LeaveRepo) UpdateAudit(
	ctx context.Context,
	kindergartenID, leaveID uint64,
	status domain.LeaveStatus,
	reviewerID uint64,
	reviewRemark string,
	reviewedAt time.Time,
) error {
	return r.db.WithContext(ctx).Model(&domain.LeaveRequest{}).
		Where("id = ? AND kindergarten_id = ? AND deleted_at IS NULL", leaveID, kindergartenID).
		Updates(map[string]interface{}{
			"status":        status,
			"reviewer_id":   reviewerID,
			"reviewed_at":   reviewedAt,
			"review_remark": reviewRemark,
		}).Error
}

func (r *LeaveRepo) CountByKindergarten(ctx context.Context, kindergartenID uint64) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.LeaveRequest{}).
		Where("kindergarten_id = ? AND deleted_at IS NULL", kindergartenID).
		Count(&total).Error
	return total, err
}

type LeaveSeedStudentRow struct {
	StudentID   uint64 `gorm:"column:student_id"`
	ClassID     uint64 `gorm:"column:class_id"`
	ApplicantID uint64 `gorm:"column:applicant_id"`
}

func (r *LeaveRepo) ListSeedStudents(ctx context.Context, kindergartenID uint64, limit int) ([]LeaveSeedStudentRow, error) {
	if limit <= 0 {
		limit = 20
	}
	var rows []LeaveSeedStudentRow
	err := r.db.WithContext(ctx).Table("students AS s").
		Joins("LEFT JOIN student_parents AS sp ON sp.student_id = s.id").
		Select("s.id AS student_id, s.class_id, MIN(sp.user_id) AS applicant_id").
		Where("s.kindergarten_id = ? AND s.status = 1 AND s.deleted_at IS NULL", kindergartenID).
		Group("s.id, s.class_id").
		Having("MIN(sp.user_id) > 0").
		Order("s.class_id ASC, s.id ASC").
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}

func (r *LeaveRepo) CreateBatch(ctx context.Context, leaves []domain.LeaveRequest) error {
	if len(leaves) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		CreateInBatches(leaves, 100).Error
}
