package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"kindergarten-server/internal/domain"
)

type AttendanceRepo struct {
	db *gorm.DB
}

func NewAttendanceRepo(db *gorm.DB) *AttendanceRepo {
	return &AttendanceRepo{db: db}
}

// AttendanceClassStatusAgg 单日按班级、状态聚合条数。
type AttendanceClassStatusAgg struct {
	ClassID uint64                  `gorm:"column:class_id"`
	Status  domain.AttendanceStatus `gorm:"column:status"`
	Cnt     int64                   `gorm:"column:cnt"`
}

func (r *AttendanceRepo) AggregateByClassStatus(
	ctx context.Context,
	kindergartenID uint64,
	fromInclusive, toExclusive time.Time,
) ([]AttendanceClassStatusAgg, error) {
	var rows []AttendanceClassStatusAgg
	err := r.db.WithContext(ctx).Model(&domain.Attendance{}).
		Select("class_id, status, COUNT(*) AS cnt").
		Where("kindergarten_id = ? AND deleted_at IS NULL AND date >= ? AND date < ?", kindergartenID, fromInclusive, toExclusive).
		Group("class_id, status").
		Scan(&rows).Error
	return rows, err
}

func (r *AttendanceRepo) CountByKindergarten(ctx context.Context, kindergartenID uint64) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.Attendance{}).
		Where("kindergarten_id = ? AND deleted_at IS NULL", kindergartenID).
		Count(&total).Error
	return total, err
}

func (r *AttendanceRepo) CountByDateRange(
	ctx context.Context,
	kindergartenID uint64,
	fromInclusive, toExclusive time.Time,
) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.Attendance{}).
		Where("kindergarten_id = ? AND deleted_at IS NULL AND date >= ? AND date < ?",
			kindergartenID, fromInclusive, toExclusive).
		Count(&total).Error
	return total, err
}

type AttendanceStudentBrief struct {
	ID      uint64 `gorm:"column:id"`
	ClassID uint64 `gorm:"column:class_id"`
}

func (r *AttendanceRepo) ListActiveStudents(ctx context.Context, kindergartenID uint64) ([]AttendanceStudentBrief, error) {
	var rows []AttendanceStudentBrief
	err := r.db.WithContext(ctx).Model(&domain.Student{}).
		Select("id, class_id").
		Where("kindergarten_id = ? AND status = 1 AND deleted_at IS NULL", kindergartenID).
		Order("class_id ASC, id ASC").
		Scan(&rows).Error
	return rows, err
}

func (r *AttendanceRepo) UpsertBatch(ctx context.Context, rows []domain.Attendance) error {
	if len(rows) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "student_id"}, {Name: "date"}},
			DoNothing: true,
		}).
		CreateInBatches(rows, 200).Error
}

// AttendanceRecordJoinRow 出勤明细列表行。
type AttendanceRecordJoinRow struct {
	ID            uint64                  `gorm:"column:id"`
	StudentID     uint64                  `gorm:"column:student_id"`
	StudentName   string                  `gorm:"column:student_name"`
	StudentAvatar string                  `gorm:"column:student_avatar"`
	ClassName     string                  `gorm:"column:class_name"`
	Date          time.Time               `gorm:"column:date"`
	Status        domain.AttendanceStatus `gorm:"column:status"`
	CheckInTime   *time.Time              `gorm:"column:check_in_time"`
	Remark        string                  `gorm:"column:remark"`
}

func (r *AttendanceRepo) ListRecords(
	ctx context.Context,
	kindergartenID uint64,
	classID *uint64,
	fromInclusive, toExclusive time.Time,
	statusFilter string,
	offset, limit int,
) ([]AttendanceRecordJoinRow, int64, error) {
	base := r.db.WithContext(ctx).Table("attendances AS a").
		Joins("INNER JOIN students AS s ON s.id = a.student_id AND s.deleted_at IS NULL").
		Joins("INNER JOIN classes AS c ON c.id = a.class_id AND c.deleted_at IS NULL").
		Where("a.kindergarten_id = ? AND a.date >= ? AND a.date < ? AND a.deleted_at IS NULL",
			kindergartenID, fromInclusive, toExclusive)
	if classID != nil && *classID > 0 {
		base = base.Where("a.class_id = ?", *classID)
	}
	switch statusFilter {
	case "present":
		base = base.Where("a.status IN ?", []domain.AttendanceStatus{domain.AttendancePresent, domain.AttendanceLate})
	case "leave":
		base = base.Where("a.status = ?", domain.AttendanceLeave)
	case "absent":
		base = base.Where("a.status = ?", domain.AttendanceAbsent)
	}
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	q := base.Select(`a.id, a.student_id, s.name AS student_name, s.avatar AS student_avatar, c.name AS class_name,
			a.date, a.status, a.check_in_time, a.remark`).
		Order("a.date DESC, a.id DESC").Offset(offset).Limit(limit)
	var rows []AttendanceRecordJoinRow
	err := q.Scan(&rows).Error
	return rows, total, err
}

// ExportAttendanceRow 导出用（日期范围内全部记录，不分页）。
type ExportAttendanceRow struct {
	Date        time.Time               `gorm:"column:date"`
	ClassName   string                  `gorm:"column:class_name"`
	StudentName string                  `gorm:"column:student_name"`
	Status      domain.AttendanceStatus `gorm:"column:status"`
	CheckInTime *time.Time              `gorm:"column:check_in_time"`
	Remark      string                  `gorm:"column:remark"`
}

func (r *AttendanceRepo) ListForExport(
	ctx context.Context,
	kindergartenID uint64,
	classID *uint64,
	fromInclusive, toExclusive time.Time,
) ([]ExportAttendanceRow, error) {
	q := r.db.WithContext(ctx).Table("attendances AS a").
		Select(`a.date, c.name AS class_name, s.name AS student_name, a.status, a.check_in_time, a.remark`).
		Joins("INNER JOIN students AS s ON s.id = a.student_id AND s.deleted_at IS NULL").
		Joins("INNER JOIN classes AS c ON c.id = a.class_id AND c.deleted_at IS NULL").
		Where("a.kindergarten_id = ? AND a.date >= ? AND a.date < ? AND a.deleted_at IS NULL",
			kindergartenID, fromInclusive, toExclusive)
	if classID != nil && *classID > 0 {
		q = q.Where("a.class_id = ?", *classID)
	}
	var rows []ExportAttendanceRow
	err := q.Order("a.date ASC, c.name ASC, s.name ASC").Scan(&rows).Error
	return rows, err
}
