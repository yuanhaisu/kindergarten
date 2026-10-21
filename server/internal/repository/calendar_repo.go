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

// FindTeacherEvents 老师视角事件列表：
// 本班事件（含草稿）+ 本园园级事件（仅已发布——园级草稿属发布者工作稿，对老师不可见）。
func (r *CalendarRepo) FindTeacherEvents(
	ctx context.Context, kindergartenID, classID uint64, start, end string, includeTimetable bool,
) ([]domain.CalendarEvent, error) {
	q := r.db.WithContext(ctx).
		Where("kindergarten_id = ?", kindergartenID).
		Where("(scope = ? AND class_id = ?) OR (scope = ? AND status = ?)",
			domain.CalendarScopeClass, classID,
			domain.CalendarScopeKindergarten, domain.CalendarStatusPublished)
	if !includeTimetable {
		// 课表开关关闭：课表来源事件不呈现（数据保留，开关可逆）
		q = q.Where("source <> ?", domain.CalendarSourceTimetable)
	}
	return findCalendarEvents(q, start, end)
}

// FindParentEvents 家长视角事件列表：仅已发布，且受众命中（本班或园级）。
func (r *CalendarRepo) FindParentEvents(
	ctx context.Context, kindergartenID, classID uint64, start, end string, includeTimetable bool,
) ([]domain.CalendarEvent, error) {
	q := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND status = ?", kindergartenID, domain.CalendarStatusPublished).
		Where("(scope = ? AND class_id = ?) OR scope = ?",
			domain.CalendarScopeClass, classID, domain.CalendarScopeKindergarten)
	if !includeTimetable {
		q = q.Where("source <> ?", domain.CalendarSourceTimetable)
	}
	return findCalendarEvents(q, start, end)
}

func findCalendarEvents(q *gorm.DB, start, end string) ([]domain.CalendarEvent, error) {
	if start != "" && end != "" {
		q = q.Where("event_date BETWEEN ? AND ?", start, end)
	}
	var rows []domain.CalendarEvent
	if err := q.Order("event_date ASC, start_time ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// FindTeacherEventByID 单条查询（老师可见性：本班任意状态 / 园级已发布）。未命中返回 (nil, nil)。
func (r *CalendarRepo) FindTeacherEventByID(
	ctx context.Context, kindergartenID, classID, id uint64,
) (*domain.CalendarEvent, error) {
	var row domain.CalendarEvent
	err := r.db.WithContext(ctx).
		Where("id = ? AND kindergarten_id = ?", id, kindergartenID).
		Where("(scope = ? AND class_id = ?) OR (scope = ? AND status = ?)",
			domain.CalendarScopeClass, classID,
			domain.CalendarScopeKindergarten, domain.CalendarStatusPublished).
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
	ctx context.Context, kindergartenID, classID, id uint64, updates map[string]any,
) (int64, error) {
	res := r.db.WithContext(ctx).
		Model(&domain.CalendarEvent{}).
		Where("id = ? AND kindergarten_id = ? AND class_id = ?", id, kindergartenID, classID).
		Updates(updates)
	return res.RowsAffected, res.Error
}

// DeleteEvent 软删除日历事件（仅限本园所本班级）。
func (r *CalendarRepo) DeleteEvent(
	ctx context.Context, kindergartenID, classID, id uint64,
) (int64, error) {
	res := r.db.WithContext(ctx).
		Where("id = ? AND kindergarten_id = ? AND class_id = ?", id, kindergartenID, classID).
		Delete(&domain.CalendarEvent{})
	return res.RowsAffected, res.Error
}

// CalendarDotRow 标记点聚合行：日期 × 事件类型。
type CalendarDotRow struct {
	Date      time.Time `gorm:"column:date"`
	EventType string    `gorm:"column:event_type"`
}

// DotsForTeacher 按天聚合事件类型（与老师列表同口径，含草稿）。
func (r *CalendarRepo) DotsForTeacher(
	ctx context.Context, kindergartenID, classID uint64, start, end string, includeTimetable bool,
) ([]CalendarDotRow, error) {
	q := r.db.WithContext(ctx).
		Model(&domain.CalendarEvent{}).
		Select("event_date AS date, event_type").
		Where("kindergarten_id = ?", kindergartenID).
		Where("(scope = ? AND class_id = ?) OR (scope = ? AND status = ?)",
			domain.CalendarScopeClass, classID,
			domain.CalendarScopeKindergarten, domain.CalendarStatusPublished).
		Group("event_date, event_type").
		Order("event_date ASC, event_type ASC")
	if !includeTimetable {
		q = q.Where("source <> ?", domain.CalendarSourceTimetable)
	}
	if start != "" && end != "" {
		q = q.Where("event_date BETWEEN ? AND ?", start, end)
	}
	var rows []CalendarDotRow
	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// SlotTaken 课表槽位是否已占。必须带 Unscoped：被老师覆盖/取消的实例虽然软删，
// 但仍占据唯一槽位，同步重生成必须据此跳过（软删行占唯一索引是本仓库已知陷阱）。
func (r *CalendarRepo) SlotTaken(ctx context.Context, classID uint64, date time.Time, slot string) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Unscoped().
		Model(&domain.CalendarEvent{}).
		Where("class_id = ? AND event_date = ? AND source_slot = ?", classID, date, slot).
		Count(&n).Error
	return n > 0, err
}

// HardDeleteEvent 硬删事件（Unscoped）。仅用于系统自有行（source=timetable 且未被覆盖）：
// 模板不再需要该槽位时须真正释放唯一键位，否则软删行会占位、该槽位永远无法重建。
func (r *CalendarRepo) HardDeleteEvent(ctx context.Context, kindergartenID, id uint64) error {
	return r.db.WithContext(ctx).Unscoped().
		Where("id = ? AND kindergarten_id = ?", id, kindergartenID).
		Delete(&domain.CalendarEvent{}).Error
}

// ── 园级事件（scope=kindergarten；发布者为管理端园长角色或医生，class_id 为 NULL）──

// FindGardenEvents 园级事件列表；status 非空时按其过滤（draft/published）。
func (r *CalendarRepo) FindGardenEvents(
	ctx context.Context, kindergartenID uint64, start, end, status string,
) ([]domain.CalendarEvent, error) {
	q := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND scope = ?", kindergartenID, domain.CalendarScopeKindergarten)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	return findCalendarEvents(q, start, end)
}

// FindGardenEventByID 园级单条；未命中返回 (nil, nil)。
func (r *CalendarRepo) FindGardenEventByID(ctx context.Context, kindergartenID, id uint64) (*domain.CalendarEvent, error) {
	var row domain.CalendarEvent
	err := r.db.WithContext(ctx).
		Where("id = ? AND kindergarten_id = ? AND scope = ?", id, kindergartenID, domain.CalendarScopeKindergarten).
		First(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

// UpdateGardenEvent 更新园级事件（WHERE scope=kindergarten，不触班级事件）。
func (r *CalendarRepo) UpdateGardenEvent(ctx context.Context, kindergartenID, id uint64, updates map[string]any) (int64, error) {
	res := r.db.WithContext(ctx).
		Model(&domain.CalendarEvent{}).
		Where("id = ? AND kindergarten_id = ? AND scope = ?", id, kindergartenID, domain.CalendarScopeKindergarten).
		Updates(updates)
	return res.RowsAffected, res.Error
}

// DeleteGardenEvent 软删除园级事件。
func (r *CalendarRepo) DeleteGardenEvent(ctx context.Context, kindergartenID, id uint64) (int64, error) {
	res := r.db.WithContext(ctx).
		Where("id = ? AND kindergarten_id = ? AND scope = ?", id, kindergartenID, domain.CalendarScopeKindergarten).
		Delete(&domain.CalendarEvent{})
	return res.RowsAffected, res.Error
}
