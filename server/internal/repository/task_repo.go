package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

type TaskRepo struct {
	db *gorm.DB
}

func NewTaskRepo(db *gorm.DB) *TaskRepo {
	return &TaskRepo{db: db}
}

// TaskListRow 管理端任务列表 JOIN 行。
type TaskListRow struct {
	ID          uint64    `gorm:"column:id"`
	ClassID     uint64    `gorm:"column:class_id"`
	ClassName   string    `gorm:"column:class_name"`
	Title       string    `gorm:"column:title"`
	Frequency   string    `gorm:"column:frequency"`
	StartDate   time.Time `gorm:"column:start_date"`
	EndDate     time.Time `gorm:"column:end_date"`
	Status      int8      `gorm:"column:status"`
	CreatorID   uint64    `gorm:"column:creator_id"`
	CreatorNick string    `gorm:"column:creator_nick"`
	CreatedAt   time.Time `gorm:"column:created_at"`
}

func (r *TaskRepo) baseListQuery(kindergartenID uint64, classID *uint64, status *int8) *gorm.DB {
	q := r.db.Table("parent_tasks AS pt").
		Joins("INNER JOIN classes AS c ON c.id = pt.class_id AND c.deleted_at IS NULL").
		Joins("INNER JOIN users AS u ON u.id = pt.creator_id AND u.deleted_at IS NULL").
		Where("pt.kindergarten_id = ? AND pt.deleted_at IS NULL", kindergartenID)
	if classID != nil && *classID > 0 {
		q = q.Where("pt.class_id = ?", *classID)
	}
	if status != nil {
		q = q.Where("pt.status = ?", *status)
	}
	return q
}

func (r *TaskRepo) ListTasks(
	ctx context.Context,
	kindergartenID uint64,
	classID *uint64,
	status *int8,
	offset, limit int,
) ([]TaskListRow, int64, error) {
	base := r.baseListQuery(kindergartenID, classID, status)
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	sel := base.Session(&gorm.Session{}).
		Select(`pt.id, pt.class_id, c.name AS class_name, pt.title, pt.frequency,
			pt.start_date, pt.end_date, pt.status,
			pt.creator_id, u.nickname AS creator_nick, pt.created_at`).
		Order("pt.created_at DESC, pt.id DESC").
		Offset(offset).Limit(limit)
	var rows []TaskListRow
	err := sel.Scan(&rows).Error
	return rows, total, err
}

func (r *TaskRepo) FindTaskByID(ctx context.Context, kindergartenID, taskID uint64) (*domain.ParentTask, error) {
	var t domain.ParentTask
	err := r.db.WithContext(ctx).
		Preload("Class").
		Preload("Creator").
		Where("kindergarten_id = ? AND id = ? AND deleted_at IS NULL", kindergartenID, taskID).
		First(&t).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *TaskRepo) SoftDeleteTask(ctx context.Context, kindergartenID, taskID uint64) (int64, error) {
	res := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND id = ? AND deleted_at IS NULL", kindergartenID, taskID).
		Delete(&domain.ParentTask{})
	return res.RowsAffected, res.Error
}

func (r *TaskRepo) CountCheckinsByTaskID(ctx context.Context, taskID uint64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.TaskCheckIn{}).
		Where("task_id = ? AND deleted_at IS NULL", taskID).
		Count(&n).Error
	return n, err
}

// CountActiveStudentsInClass 在读学生数（status=1）。
func (r *TaskRepo) CountActiveStudentsInClass(ctx context.Context, kindergartenID, classID uint64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.Student{}).
		Where("kindergarten_id = ? AND class_id = ? AND deleted_at IS NULL AND status = ?", kindergartenID, classID, 1).
		Count(&n).Error
	return n, err
}

func (r *TaskRepo) ClassBelongsToKindergarten(ctx context.Context, kindergartenID, classID uint64) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.Class{}).
		Where("id = ? AND kindergarten_id = ? AND deleted_at IS NULL", classID, kindergartenID).
		Count(&n).Error
	return n > 0, err
}

func (r *TaskRepo) CreateTask(ctx context.Context, t *domain.ParentTask) error {
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *TaskRepo) UpdateTask(ctx context.Context, kindergartenID, taskID uint64, updates map[string]interface{}) (int64, error) {
	res := r.db.WithContext(ctx).Model(&domain.ParentTask{}).
		Where("kindergarten_id = ? AND id = ? AND deleted_at IS NULL", kindergartenID, taskID).
		Updates(updates)
	return res.RowsAffected, res.Error
}

// TaskOverlapRow 与日期区间有交集的任务（统计用）。
type TaskOverlapRow struct {
	ID        uint64    `gorm:"column:id"`
	ClassID   uint64    `gorm:"column:class_id"`
	ClassName string    `gorm:"column:class_name"`
	Frequency string    `gorm:"column:frequency"`
	StartDate time.Time `gorm:"column:start_date"`
	EndDate   time.Time `gorm:"column:end_date"`
	Status    int8      `gorm:"column:status"`
}

func (r *TaskRepo) ListTasksOverlappingRange(
	ctx context.Context,
	kindergartenID uint64,
	rangeStart, rangeEnd time.Time,
) ([]TaskOverlapRow, error) {
	var rows []TaskOverlapRow
	err := r.db.WithContext(ctx).Table("parent_tasks AS pt").
		Select("pt.id, pt.class_id, c.name AS class_name, pt.frequency, pt.start_date, pt.end_date, pt.status").
		Joins("INNER JOIN classes AS c ON c.id = pt.class_id AND c.deleted_at IS NULL").
		Where("pt.kindergarten_id = ? AND pt.deleted_at IS NULL", kindergartenID).
		Where("pt.start_date <= ? AND pt.end_date >= ?", rangeEnd, rangeStart).
		Order("pt.id ASC").
		Scan(&rows).Error
	return rows, err
}

func (r *TaskRepo) CountCheckinsByTaskIDs(ctx context.Context, taskIDs []uint64) (map[uint64]int64, error) {
	out := make(map[uint64]int64)
	if len(taskIDs) == 0 {
		return out, nil
	}
	type row struct {
		TaskID uint64 `gorm:"column:task_id"`
		Cnt    int64  `gorm:"column:cnt"`
	}
	var list []row
	err := r.db.WithContext(ctx).Model(&domain.TaskCheckIn{}).
		Select("task_id, COUNT(*) AS cnt").
		Where("task_id IN ? AND deleted_at IS NULL", taskIDs).
		Group("task_id").
		Scan(&list).Error
	if err != nil {
		return nil, err
	}
	for _, x := range list {
		out[x.TaskID] = x.Cnt
	}
	return out, nil
}

func (r *TaskRepo) CountCheckinsInDateRange(ctx context.Context, kindergartenID uint64, rangeStart, rangeEnd time.Time) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Table("task_check_ins AS ci").
		Joins("INNER JOIN parent_tasks AS t ON t.id = ci.task_id AND t.deleted_at IS NULL").
		Where("t.kindergarten_id = ? AND ci.deleted_at IS NULL", kindergartenID).
		Where("ci.date >= ? AND ci.date <= ?", rangeStart, rangeEnd).
		Count(&n).Error
	return n, err
}
