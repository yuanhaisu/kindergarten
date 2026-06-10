package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

type DashboardRepo struct {
	db *gorm.DB
}

func NewDashboardRepo(db *gorm.DB) *DashboardRepo {
	return &DashboardRepo{db: db}
}

func (r *DashboardRepo) CountActiveStudents(ctx context.Context, kindergartenID uint64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.Student{}).
		Where("kindergarten_id = ? AND status = ?", kindergartenID, 1).
		Count(&n).Error
	return n, err
}

func (r *DashboardRepo) CountAttendanceByStatus(
	ctx context.Context,
	kindergartenID uint64,
	from, to time.Time,
	st domain.AttendanceStatus,
) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.Attendance{}).
		Where("kindergarten_id = ?", kindergartenID).
		Where("date >= ? AND date < ?", from, to).
		Where("status = ?", st).
		Count(&n).Error
	return n, err
}

func (r *DashboardRepo) CountPickupByType(
	ctx context.Context,
	kindergartenID uint64,
	from, to time.Time,
	pickupType domain.PickupType,
) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.PickupRecord{}).
		Where("kindergarten_id = ?", kindergartenID).
		Where("date >= ? AND date < ?", from, to).
		Where("type = ?", pickupType).
		Count(&n).Error
	return n, err
}

func (r *DashboardRepo) CountNoticesCreatedBetween(ctx context.Context, kindergartenID uint64, from, to time.Time) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.Notice{}).
		Where("kindergarten_id = ?", kindergartenID).
		Where("created_at >= ? AND created_at < ?", from, to).
		Count(&n).Error
	return n, err
}

func (r *DashboardRepo) CountPendingLeaves(ctx context.Context, kindergartenID uint64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.LeaveRequest{}).
		Where("kindergarten_id = ? AND status = ?", kindergartenID, domain.LeaveStatusPending).
		Count(&n).Error
	return n, err
}

func (r *DashboardRepo) CountGrowthLogsByDate(ctx context.Context, kindergartenID uint64, from, to time.Time) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.GrowthDailyLog{}).
		Where("kindergarten_id = ?", kindergartenID).
		Where("date >= ? AND date < ?", from, to).
		Count(&n).Error
	return n, err
}

func (r *DashboardRepo) CountParentTasksCreatedBetween(ctx context.Context, kindergartenID uint64, from, to time.Time) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.ParentTask{}).
		Where("kindergarten_id = ?", kindergartenID).
		Where("created_at >= ? AND created_at < ?", from, to).
		Count(&n).Error
	return n, err
}

func (r *DashboardRepo) SumPositivePointsInRange(ctx context.Context, kindergartenID uint64, from, to time.Time) (int64, error) {
	var sum int64
	err := r.db.WithContext(ctx).Raw(`
SELECT COALESCE(SUM(pt.points), 0)
FROM point_transactions pt
INNER JOIN students s ON s.id = pt.student_id
WHERE s.kindergarten_id = ?
  AND pt.points > 0
  AND pt.created_at >= ? AND pt.created_at < ?`,
		kindergartenID, from, to).Scan(&sum).Error
	return sum, err
}

func (r *DashboardRepo) CountStudentsCreatedBetween(ctx context.Context, kindergartenID uint64, from, to time.Time) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.Student{}).
		Where("kindergarten_id = ?", kindergartenID).
		Where("created_at >= ? AND created_at < ?", from, to).
		Count(&n).Error
	return n, err
}

func (r *DashboardRepo) CountNewParentRolesInMonth(ctx context.Context, kindergartenID uint64, from, to time.Time) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.UserRole{}).
		Where("role = ? AND kindergarten_id = ?", "parent", kindergartenID).
		Where("created_at >= ? AND created_at < ?", from, to).
		Count(&n).Error
	return n, err
}

// AttendanceDayAgg 单日出勤聚合（按园所）
func (r *DashboardRepo) AttendanceDayAgg(
	ctx context.Context,
	kindergartenID uint64,
	from, to time.Time,
) (present, leave, absent int64, err error) {
	present, err = r.CountAttendanceByStatus(ctx, kindergartenID, from, to, domain.AttendancePresent)
	if err != nil {
		return 0, 0, 0, err
	}
	leave, err = r.CountAttendanceByStatus(ctx, kindergartenID, from, to, domain.AttendanceLeave)
	if err != nil {
		return 0, 0, 0, err
	}
	absent, err = r.CountAttendanceByStatus(ctx, kindergartenID, from, to, domain.AttendanceAbsent)
	if err != nil {
		return 0, 0, 0, err
	}
	return present, leave, absent, nil
}

type ClassRankingRow struct {
	ClassID   uint64
	ClassName string
	Present   int64
	Total     int64
}

func (r *DashboardRepo) ClassRankingToday(ctx context.Context, kindergartenID uint64, dayStart, dayEnd time.Time) ([]ClassRankingRow, error) {
	var classes []domain.Class
	if err := r.db.WithContext(ctx).Where("kindergarten_id = ?", kindergartenID).Find(&classes).Error; err != nil {
		return nil, err
	}
	out := make([]ClassRankingRow, 0, len(classes))
	for _, c := range classes {
		var total int64
		if err := r.db.WithContext(ctx).Model(&domain.Student{}).
			Where("class_id = ? AND kindergarten_id = ? AND status = ?", c.ID, kindergartenID, 1).
			Count(&total).Error; err != nil {
			return nil, err
		}
		var present int64
		if err := r.db.WithContext(ctx).Model(&domain.Attendance{}).
			Where("class_id = ? AND kindergarten_id = ?", c.ID, kindergartenID).
			Where("date >= ? AND date < ?", dayStart, dayEnd).
			Where("status = ?", domain.AttendancePresent).
			Count(&present).Error; err != nil {
			return nil, err
		}
		out = append(out, ClassRankingRow{ClassID: c.ID, ClassName: c.Name, Present: present, Total: total})
	}
	return out, nil
}

type GrowthClassAgg struct {
	ClassID   uint64 `gorm:"column:class_id"`
	ClassName string `gorm:"column:class_name"`
	LogsCount int64  `gorm:"column:logs_count"`
}

func (r *DashboardRepo) GrowthLogsByClass(ctx context.Context, kindergartenID uint64, from, to time.Time) ([]GrowthClassAgg, error) {
	var rows []GrowthClassAgg
	err := r.db.WithContext(ctx).Model(&domain.GrowthDailyLog{}).
		Select("growth_daily_logs.class_id AS class_id, classes.name AS class_name, COUNT(growth_daily_logs.id) AS logs_count").
		Joins("INNER JOIN classes ON classes.id = growth_daily_logs.class_id AND classes.kindergarten_id = ?", kindergartenID).
		Where("growth_daily_logs.kindergarten_id = ?", kindergartenID).
		Where("growth_daily_logs.date >= ? AND growth_daily_logs.date < ?", from, to).
		Group("growth_daily_logs.class_id, classes.name").
		Order("logs_count DESC").
		Limit(10).
		Scan(&rows).Error
	return rows, err
}

func (r *DashboardRepo) CountGrowthLogsInRange(ctx context.Context, kindergartenID uint64, from, to time.Time) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.GrowthDailyLog{}).
		Where("kindergarten_id = ?", kindergartenID).
		Where("date >= ? AND date < ?", from, to).
		Count(&n).Error
	return n, err
}

func (r *DashboardRepo) CountGrowthMediaInRange(ctx context.Context, kindergartenID uint64, from, to time.Time) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.GrowthMedia{}).
		Where("kindergarten_id = ?", kindergartenID).
		Where("created_at >= ? AND created_at < ?", from, to).
		Count(&n).Error
	return n, err
}

func (r *DashboardRepo) CountGrowthLikesForLogsInRange(ctx context.Context, kindergartenID uint64, from, to time.Time) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Table("growth_likes").
		Joins("INNER JOIN growth_daily_logs ON growth_daily_logs.id = growth_likes.log_id").
		Where("growth_daily_logs.kindergarten_id = ?", kindergartenID).
		Where("growth_daily_logs.date >= ? AND growth_daily_logs.date < ?", from, to).
		Count(&n).Error
	return n, err
}

func (r *DashboardRepo) CountGrowthCommentsForLogsInRange(ctx context.Context, kindergartenID uint64, from, to time.Time) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Table("growth_comments").
		Joins("INNER JOIN growth_daily_logs ON growth_daily_logs.id = growth_comments.log_id").
		Where("growth_daily_logs.kindergarten_id = ?", kindergartenID).
		Where("growth_daily_logs.date >= ? AND growth_daily_logs.date < ?", from, to).
		Count(&n).Error
	return n, err
}

func (r *DashboardRepo) MonthAttendanceRollup(
	ctx context.Context,
	kindergartenID uint64,
	from, to time.Time,
) (present, absent, leave int64, err error) {
	var row struct {
		Present int64 `gorm:"column:present_cnt"`
		Absent  int64 `gorm:"column:absent_cnt"`
		Leave   int64 `gorm:"column:leave_cnt"`
	}
	err = r.db.WithContext(ctx).Model(&domain.Attendance{}).
		Select(`COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0) AS present_cnt,
			COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0) AS absent_cnt,
			COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0) AS leave_cnt`,
			domain.AttendancePresent, domain.AttendanceAbsent, domain.AttendanceLeave).
		Where("kindergarten_id = ?", kindergartenID).
		Where("date >= ? AND date < ?", from, to).
		Scan(&row).Error
	return row.Present, row.Absent, row.Leave, err
}

func (r *DashboardRepo) FirstTeacherNickname(ctx context.Context, classID uint64) (string, error) {
	var nick string
	err := r.db.WithContext(ctx).Table("class_teachers AS ct").
		Select("users.nickname").
		Joins("INNER JOIN users ON users.id = ct.user_id").
		Where("ct.class_id = ? AND ct.title = ?", classID, domain.TeacherTitleHead).
		Limit(1).
		Scan(&nick).Error
	if err != nil {
		return "", err
	}
	if nick != "" {
		return nick, nil
	}
	err = r.db.WithContext(ctx).Table("class_teachers AS ct").
		Select("users.nickname").
		Joins("INNER JOIN users ON users.id = ct.user_id").
		Where("ct.class_id = ?", classID).
		Limit(1).
		Scan(&nick).Error
	return nick, err
}
