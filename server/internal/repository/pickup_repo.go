package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

type PickupRepo struct {
	db *gorm.DB
}

func NewPickupRepo(db *gorm.DB) *PickupRepo {
	return &PickupRepo{db: db}
}

// CountDistinctStudentsByType 指定日、园所下，某接送类型的去重学生数。
func (r *PickupRepo) CountDistinctStudentsByType(
	ctx context.Context,
	kindergartenID uint64,
	day time.Time,
	pickupType domain.PickupType,
) (int64, error) {
	d := day.Format("2006-01-02")
	var n sql.NullInt64
	err := r.db.WithContext(ctx).Raw(`
SELECT COUNT(DISTINCT student_id) FROM pickup_records
WHERE kindergarten_id = ? AND deleted_at IS NULL AND DATE(date) = ? AND type = ?`,
		kindergartenID, d, pickupType).Scan(&n).Error
	if err != nil || !n.Valid {
		return 0, err
	}
	return n.Int64, nil
}

// CountStudentsDroppedButNotPicked 当日有「送入」且无「接走」的去重学生数（全园）。
func (r *PickupRepo) CountStudentsDroppedButNotPicked(
	ctx context.Context,
	kindergartenID uint64,
	day time.Time,
) (int64, error) {
	d := day.Format("2006-01-02")
	var n sql.NullInt64
	err := r.db.WithContext(ctx).Raw(`
SELECT COUNT(DISTINCT pr.student_id) FROM pickup_records pr
WHERE pr.kindergarten_id = ? AND pr.deleted_at IS NULL AND DATE(pr.date) = ? AND pr.type = ?
AND NOT EXISTS (
  SELECT 1 FROM pickup_records x
  WHERE x.deleted_at IS NULL AND x.kindergarten_id = pr.kindergarten_id
  AND DATE(x.date) = ? AND x.student_id = pr.student_id AND x.type = ?
)`, kindergartenID, d, domain.PickupTypeDropOff, d, domain.PickupTypePickUp).Scan(&n).Error
	if err != nil || !n.Valid {
		return 0, err
	}
	return n.Int64, nil
}

// CountAbnormalRecords 当日异常接送条数。
func (r *PickupRepo) CountAbnormalRecords(ctx context.Context, kindergartenID uint64, day time.Time) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.PickupRecord{}).
		Where("kindergarten_id = ? AND is_abnormal = ? AND deleted_at IS NULL", kindergartenID, true).
		Where("DATE(date) = ?", day.Format("2006-01-02")).
		Count(&n).Error
	return n, err
}

// ClassPickupAgg 按班级的接送统计。
type ClassPickupAgg struct {
	ClassID     uint64 `gorm:"column:class_id"`
	ClassName   string `gorm:"column:class_name"`
	Arrived     int64  `gorm:"column:arrived"`
	PickedUp    int64  `gorm:"column:picked_up"`
	NotPickedUp int64  `gorm:"column:not_picked_up"`
}

// AggregateByClass 按班级聚合（含在读班级名；仅统计有记录的班级，由上层与全量班级合并补零）。
func (r *PickupRepo) AggregateByClass(ctx context.Context, kindergartenID uint64, day time.Time) ([]ClassPickupAgg, error) {
	d := day.Format("2006-01-02")
	// 送入去重
	type dropRow struct {
		ClassID uint64 `gorm:"column:class_id"`
		Cnt     int64  `gorm:"column:cnt"`
	}
	var drops []dropRow
	err := r.db.WithContext(ctx).Raw(`
SELECT pr.class_id, COUNT(DISTINCT pr.student_id) AS cnt
FROM pickup_records pr
INNER JOIN classes c ON c.id = pr.class_id AND c.deleted_at IS NULL AND c.kindergarten_id = ?
WHERE pr.kindergarten_id = ? AND pr.deleted_at IS NULL AND DATE(pr.date) = ? AND pr.type = ?
GROUP BY pr.class_id`, kindergartenID, kindergartenID, d, domain.PickupTypeDropOff).Scan(&drops).Error
	if err != nil {
		return nil, err
	}
	// 接走去重
	var picks []dropRow
	err = r.db.WithContext(ctx).Raw(`
SELECT pr.class_id, COUNT(DISTINCT pr.student_id) AS cnt
FROM pickup_records pr
INNER JOIN classes c ON c.id = pr.class_id AND c.deleted_at IS NULL AND c.kindergarten_id = ?
WHERE pr.kindergarten_id = ? AND pr.deleted_at IS NULL AND DATE(pr.date) = ? AND pr.type = ?
GROUP BY pr.class_id`, kindergartenID, kindergartenID, d, domain.PickupTypePickUp).Scan(&picks).Error
	if err != nil {
		return nil, err
	}
	// 未接走：有送入无接走（同班同日）
	var notPicked []dropRow
	err = r.db.WithContext(ctx).Raw(`
SELECT pr.class_id, COUNT(DISTINCT pr.student_id) AS cnt
FROM pickup_records pr
INNER JOIN classes c ON c.id = pr.class_id AND c.deleted_at IS NULL AND c.kindergarten_id = ?
WHERE pr.kindergarten_id = ? AND pr.deleted_at IS NULL AND DATE(pr.date) = ? AND pr.type = ?
AND NOT EXISTS (
  SELECT 1 FROM pickup_records x
  WHERE x.deleted_at IS NULL AND x.kindergarten_id = pr.kindergarten_id
  AND DATE(x.date) = ? AND x.class_id = pr.class_id AND x.student_id = pr.student_id AND x.type = ?
)
GROUP BY pr.class_id`, kindergartenID, kindergartenID, d, domain.PickupTypeDropOff, d, domain.PickupTypePickUp).Scan(&notPicked).Error
	if err != nil {
		return nil, err
	}

	dropM := make(map[uint64]int64)
	for _, x := range drops {
		dropM[x.ClassID] = x.Cnt
	}
	pickM := make(map[uint64]int64)
	for _, x := range picks {
		pickM[x.ClassID] = x.Cnt
	}
	notM := make(map[uint64]int64)
	for _, x := range notPicked {
		notM[x.ClassID] = x.Cnt
	}

	ids := make(map[uint64]struct{})
	for id := range dropM {
		ids[id] = struct{}{}
	}
	for id := range pickM {
		ids[id] = struct{}{}
	}
	for id := range notM {
		ids[id] = struct{}{}
	}

	var classNames map[uint64]string
	if len(ids) > 0 {
		idList := make([]uint64, 0, len(ids))
		for id := range ids {
			idList = append(idList, id)
		}
		var rows []struct {
			ID   uint64 `gorm:"column:id"`
			Name string `gorm:"column:name"`
		}
		err = r.db.WithContext(ctx).Model(&domain.Class{}).
			Select("id, name").
			Where("kindergarten_id = ? AND id IN ?", kindergartenID, idList).
			Scan(&rows).Error
		if err != nil {
			return nil, err
		}
		classNames = make(map[uint64]string)
		for _, row := range rows {
			classNames[row.ID] = row.Name
		}
	}

	out := make([]ClassPickupAgg, 0, len(ids))
	for id := range ids {
		out = append(out, ClassPickupAgg{
			ClassID:     id,
			ClassName:   classNames[id],
			Arrived:     dropM[id],
			PickedUp:    pickM[id],
			NotPickedUp: notM[id],
		})
	}
	return out, nil
}

// PickupRecordListRow 接送记录列表行。
type PickupRecordListRow struct {
	ID             uint64              `gorm:"column:id"`
	StudentID      uint64              `gorm:"column:student_id"`
	StudentName    string              `gorm:"column:student_name"`
	ClassName      string              `gorm:"column:class_name"`
	Type           domain.PickupType   `gorm:"column:type"`
	Method         domain.PickupMethod `gorm:"column:method"`
	PickupName     string              `gorm:"column:pickup_name"`
	PickupRelation string              `gorm:"column:pickup_relation"`
	OperatorNick   string              `gorm:"column:operator_nick"`
	IsAbnormal     bool                `gorm:"column:is_abnormal"`
	RecordedAt     time.Time           `gorm:"column:recorded_at"`
	PickupPersonID *uint64             `gorm:"column:pickup_person_id"`
	AbnormalReason string              `gorm:"column:abnormal_reason"`
}

func (r *PickupRepo) ListRecords(
	ctx context.Context,
	kindergartenID uint64,
	classID *uint64,
	day time.Time,
	abnormal *bool,
	offset, limit int,
) ([]PickupRecordListRow, int64, error) {
	base := r.db.WithContext(ctx).Table("pickup_records AS pr").
		Joins("INNER JOIN students AS s ON s.id = pr.student_id AND s.deleted_at IS NULL").
		Joins("INNER JOIN classes AS c ON c.id = pr.class_id AND c.deleted_at IS NULL").
		Joins("LEFT JOIN users AS u ON u.id = pr.operator_id").
		Where("pr.kindergarten_id = ? AND pr.deleted_at IS NULL", kindergartenID).
		Where("DATE(pr.date) = ?", day.Format("2006-01-02"))
	if classID != nil && *classID > 0 {
		base = base.Where("pr.class_id = ?", *classID)
	}
	if abnormal != nil {
		base = base.Where("pr.is_abnormal = ?", *abnormal)
	}
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []PickupRecordListRow
	q := base.Session(&gorm.Session{}).
		Select(`pr.id, pr.student_id, s.name AS student_name, c.name AS class_name,
			pr.type, pr.method, pr.pickup_name, COALESCE(u.nickname,'') AS operator_nick,
			pr.is_abnormal, pr.recorded_at, pr.pickup_person_id, pr.abnormal_reason`).
		Order("pr.recorded_at DESC, pr.id DESC").
		Offset(offset).Limit(limit)
	err := q.Scan(&rows).Error
	return rows, total, err
}

func (r *PickupRepo) ListAbnormalRecords(
	ctx context.Context,
	kindergartenID uint64,
	day *time.Time,
	offset, limit int,
) ([]PickupRecordListRow, int64, error) {
	base := r.db.WithContext(ctx).Table("pickup_records AS pr").
		Joins("INNER JOIN students AS s ON s.id = pr.student_id AND s.deleted_at IS NULL").
		Joins("INNER JOIN classes AS c ON c.id = pr.class_id AND c.deleted_at IS NULL").
		Joins("LEFT JOIN users AS u ON u.id = pr.operator_id").
		Where("pr.kindergarten_id = ? AND pr.is_abnormal = ? AND pr.deleted_at IS NULL", kindergartenID, true)
	if day != nil {
		base = base.Where("DATE(pr.date) = ?", day.Format("2006-01-02"))
	}
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []PickupRecordListRow
	err := base.Session(&gorm.Session{}).
		Select(`pr.id, pr.student_id, s.name AS student_name, c.name AS class_name,
			pr.type, pr.method, pr.pickup_name, COALESCE(u.nickname,'') AS operator_nick,
			pr.is_abnormal, pr.recorded_at, pr.pickup_person_id, pr.abnormal_reason`).
		Order("pr.recorded_at DESC, pr.id DESC").
		Offset(offset).Limit(limit).
		Scan(&rows).Error
	return rows, total, err
}

// FindAuthorizationForKindergarten 园所内授权记录（经学生归属校验）。
func (r *PickupRepo) FindAuthorizationForKindergarten(
	ctx context.Context,
	kindergartenID, authID uint64,
) (*domain.PickupAuthorization, error) {
	var pa domain.PickupAuthorization
	err := r.db.WithContext(ctx).Model(&domain.PickupAuthorization{}).
		Joins("INNER JOIN students AS s ON s.id = pickup_authorizations.student_id AND s.deleted_at IS NULL").
		Where("pickup_authorizations.id = ? AND s.kindergarten_id = ?", authID, kindergartenID).
		First(&pa).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &pa, nil
}

func (r *PickupRepo) UpdateAuthorizationReview(
	ctx context.Context,
	id uint64,
	reviewStatus int8,
	auditedAt time.Time,
	auditRemark string,
) error {
	return r.db.WithContext(ctx).Model(&domain.PickupAuthorization{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"review_status": reviewStatus,
			"audited_at":    auditedAt,
			"audit_remark":  auditRemark,
		}).Error
}

func (r *PickupRepo) CountRecordsByDate(ctx context.Context, kindergartenID uint64, day time.Time) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.PickupRecord{}).
		Where("kindergarten_id = ? AND deleted_at IS NULL", kindergartenID).
		Where("DATE(date) = ?", day.Format("2006-01-02")).
		Count(&total).Error
	return total, err
}

type PickupSeedStudentRow struct {
	StudentID uint64 `gorm:"column:student_id"`
	ClassID   uint64 `gorm:"column:class_id"`
	Name      string `gorm:"column:name"`
}

func (r *PickupRepo) ListSeedStudents(ctx context.Context, kindergartenID uint64, limit int) ([]PickupSeedStudentRow, error) {
	if limit <= 0 {
		limit = 30
	}
	var rows []PickupSeedStudentRow
	err := r.db.WithContext(ctx).Model(&domain.Student{}).
		Select("id AS student_id, class_id, name").
		Where("kindergarten_id = ? AND status = 1 AND deleted_at IS NULL", kindergartenID).
		Order("class_id ASC, id ASC").
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}

func (r *PickupRepo) CreateRecordBatch(ctx context.Context, rows []domain.PickupRecord) error {
	if len(rows) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(rows, 200).Error
}

func (r *PickupRepo) CountPendingAuthorizations(ctx context.Context, kindergartenID uint64) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Table("pickup_authorizations AS pa").
		Joins("INNER JOIN students AS s ON s.id = pa.student_id AND s.deleted_at IS NULL").
		Where("s.kindergarten_id = ? AND pa.deleted_at IS NULL AND pa.review_status = 0", kindergartenID).
		Count(&total).Error
	return total, err
}

func (r *PickupRepo) CreateAuthorizationBatch(ctx context.Context, rows []domain.PickupAuthorization) error {
	if len(rows) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(rows, 100).Error
}

// ListRecordsByStudentIDs 根据学生 ID 列表查询接送记录（家长端用）。
func (r *PickupRepo) ListRecordsByStudentIDs(
	ctx context.Context,
	kindergartenID uint64,
	studentIDs []uint64,
	offset, limit int,
) ([]PickupRecordListRow, int64, error) {
	base := r.db.WithContext(ctx).Table("pickup_records AS pr").
		Joins("INNER JOIN students AS s ON s.id = pr.student_id AND s.deleted_at IS NULL").
		Joins("INNER JOIN classes AS c ON c.id = pr.class_id AND c.deleted_at IS NULL").
		Joins("LEFT JOIN users AS u ON u.id = pr.operator_id").
		Where("pr.kindergarten_id = ? AND pr.deleted_at IS NULL", kindergartenID).
		Where("pr.student_id IN ?", studentIDs)

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []PickupRecordListRow
	err := base.Session(&gorm.Session{}).
		Select(`pr.id, pr.student_id, s.name AS student_name, c.name AS class_name,
			pr.type, pr.method, pr.pickup_name, COALESCE(pr.pickup_relation,'') AS pickup_relation, COALESCE(u.nickname,'') AS operator_nick,
			pr.is_abnormal, pr.recorded_at, pr.pickup_person_id, pr.abnormal_reason`).
		Order("pr.recorded_at DESC, pr.id DESC").
		Offset(offset).Limit(limit).
		Scan(&rows).Error
	return rows, total, err
}
