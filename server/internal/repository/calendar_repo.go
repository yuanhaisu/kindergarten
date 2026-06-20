package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

type CalendarRepo struct {
	db *gorm.DB
}

func NewCalendarRepo(db *gorm.DB) *CalendarRepo {
	return &CalendarRepo{db: db}
}

// FindEvents 查询班级日历事件，支持日期范围筛选。
func (r *CalendarRepo) FindEvents(
	ctx context.Context,
	kindergartenID, classID uint64,
	start, end string,
) ([]domain.CalendarEvent, error) {
	db := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND class_id = ?", kindergartenID, classID)
	if start != "" && end != "" {
		db = db.Where("event_date BETWEEN ? AND ?", start, end)
	}
	var rows []domain.CalendarEvent
	if err := db.Order("event_date ASC, start_time ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// FindEventByID 根据ID查询单条日历事件（需匹配园所+班级归属）。
func (r *CalendarRepo) FindEventByID(
	ctx context.Context,
	kindergartenID, classID, id uint64,
) (*domain.CalendarEvent, error) {
	var row domain.CalendarEvent
	err := r.db.WithContext(ctx).
		Where("id = ? AND kindergarten_id = ? AND class_id = ?", id, kindergartenID, classID).
		First(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

// CreateEvent 创建日历事件。
func (r *CalendarRepo) CreateEvent(ctx context.Context, row *domain.CalendarEvent) error {
	return r.db.WithContext(ctx).Create(row).Error
}

// UpdateEvent 更新日历事件（仅限本园所本班级）。
func (r *CalendarRepo) UpdateEvent(
	ctx context.Context,
	kindergartenID, classID, id uint64,
	updates map[string]any,
) (int64, error) {
	res := r.db.WithContext(ctx).
		Model(&domain.CalendarEvent{}).
		Where("id = ? AND kindergarten_id = ? AND class_id = ?", id, kindergartenID, classID).
		Updates(updates)
	return res.RowsAffected, res.Error
}

// DeleteEvent 软删除日历事件（仅限本园所本班级）。
func (r *CalendarRepo) DeleteEvent(
	ctx context.Context,
	kindergartenID, classID, id uint64,
) (int64, error) {
	res := r.db.WithContext(ctx).
		Where("id = ? AND kindergarten_id = ? AND class_id = ?", id, kindergartenID, classID).
		Delete(&domain.CalendarEvent{})
	return res.RowsAffected, res.Error
}

// CalendarDot 日历标记点聚合行。
type CalendarDot struct {
	Date  time.Time `gorm:"column:date"`
	Count int64     `gorm:"column:count"`
}

// AggregateDots 按天聚合事件数量（日历视图标记点数据）。
func (r *CalendarRepo) AggregateDots(
	ctx context.Context,
	kindergartenID, classID uint64,
	start, end string,
) ([]CalendarDot, error) {
	db := r.db.WithContext(ctx).
		Model(&domain.CalendarEvent{}).
		Select("event_date as date, count(*) as count").
		Where("kindergarten_id = ? AND class_id = ?", kindergartenID, classID).
		Group("event_date")
	if start != "" && end != "" {
		db = db.Where("event_date BETWEEN ? AND ?", start, end)
	}
	var rows []CalendarDot
	if err := db.Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}
