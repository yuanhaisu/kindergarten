package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

// ==================== PickupAuthorizations ====================

// AuthorizationListRow 授权接送人列表行。
type AuthorizationListRow struct {
	ID              uint64     `gorm:"column:id"`
	StudentID       uint64     `gorm:"column:student_id"`
	StudentName     string     `gorm:"column:student_name"`
	ClassName       string     `gorm:"column:class_name"`
	Name            string     `gorm:"column:name"`
	Phone           string     `gorm:"column:phone"`
	Relation        string     `gorm:"column:relation"`
	IDCard          string     `gorm:"column:id_card"`
	Avatar          string     `gorm:"column:avatar"`
	UserID          *uint64    `gorm:"column:user_id"`
	ExpireAt        *time.Time `gorm:"column:expire_at"`
	Status          int8       `gorm:"column:status"`
	CreatorID       uint64     `gorm:"column:creator_id"`
	AuthType        string     `gorm:"column:auth_type"`
	AllowPickup     bool       `gorm:"column:allow_pickup"`
	AllowViewRecord bool       `gorm:"column:allow_view_record"`
	AllowViewStatus bool       `gorm:"column:allow_view_status"`
	AllowGrowthView bool       `gorm:"column:allow_growth_view"`
	NotifyOnPickup  bool       `gorm:"column:notify_on_pickup"`
	ReviewStatus    int8       `gorm:"column:review_status"`
	AuditedAt       *time.Time `gorm:"column:audited_at"`
	AuditRemark     string     `gorm:"column:audit_remark"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at"`
}

// ListAuthorizationsByStudentIDs 根据学生 ID 列表分页查询授权接送人（家长端用）。
func (r *PickupRepo) ListAuthorizationsByStudentIDs(
	ctx context.Context,
	kindergartenID uint64,
	studentIDs []uint64,
	offset, limit int,
) ([]AuthorizationListRow, int64, error) {
	if len(studentIDs) == 0 {
		return []AuthorizationListRow{}, 0, nil
	}
	base := r.db.WithContext(ctx).Table("pickup_authorizations AS pa").
		Joins("INNER JOIN students AS s ON s.id = pa.student_id AND s.deleted_at IS NULL").
		Joins("INNER JOIN classes AS c ON c.id = s.class_id AND c.deleted_at IS NULL").
		Where("s.kindergarten_id = ? AND pa.deleted_at IS NULL", kindergartenID).
		Where("pa.student_id IN ?", studentIDs)

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []AuthorizationListRow
	err := base.Session(&gorm.Session{}).
		Select(`pa.id, pa.student_id, s.name AS student_name, c.name AS class_name,
			pa.name, pa.phone, pa.relation, pa.id_card, pa.avatar, pa.user_id,
			pa.expire_at, pa.status, pa.creator_id, pa.auth_type,
			pa.allow_pickup, pa.allow_view_record, pa.allow_view_status,
			pa.allow_growth_view, pa.notify_on_pickup, pa.review_status,
			pa.audited_at, pa.audit_remark, pa.created_at, pa.updated_at`).
		Order("pa.created_at DESC, pa.id DESC").
		Offset(offset).Limit(limit).
		Scan(&rows).Error
	return rows, total, err
}

// FindAuthorizationByStudent 根据学生ID和授权ID查询单条授权记录（归属校验）。
func (r *PickupRepo) FindAuthorizationByStudent(
	ctx context.Context,
	studentID, authID uint64,
) (*domain.PickupAuthorization, error) {
	var pa domain.PickupAuthorization
	err := r.db.WithContext(ctx).Model(&domain.PickupAuthorization{}).
		Where("id = ? AND student_id = ? AND deleted_at IS NULL", authID, studentID).
		First(&pa).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &pa, nil
}

// CreateAuthorization 创建单条授权记录。
// CreateAuthorization 创建授权接送人记录。
//
// 显式 Select 带 default 标签的字段：GORM 对 bool/int 零值（false / 0）会跳过该列
// 并回填 DB 默认值 —— Status:0（已撤销）、AllowPickup:false、NotifyOnPickup:false
// 都会被静默写成默认值。Select 强制写入显式给出的值。
func (r *PickupRepo) CreateAuthorization(ctx context.Context, auth *domain.PickupAuthorization) error {
	return r.db.WithContext(ctx).
		Select(
			"StudentID", "Name", "Phone", "Relation", "IDCard", "Avatar", "UserID",
			"ExpireAt", "Status", "CreatorID", "AuthType",
			"AllowPickup", "AllowViewRecord", "AllowViewStatus", "AllowGrowthView",
			"NotifyOnPickup", "ReviewStatus", "AuditedAt", "AuditRemark",
		).
		Create(auth).Error
}

// UpdateAuthorizationFields 按学生ID和授权ID部分更新字段。
func (r *PickupRepo) UpdateAuthorizationFields(
	ctx context.Context,
	studentID, authID uint64,
	updates map[string]interface{},
) error {
	return r.db.WithContext(ctx).Model(&domain.PickupAuthorization{}).
		Where("id = ? AND student_id = ? AND deleted_at IS NULL", authID, studentID).
		Updates(updates).Error
}

// SoftDeleteAuthorization 软删除一条授权记录。
func (r *PickupRepo) SoftDeleteAuthorization(ctx context.Context, studentID, authID uint64) error {
	return r.db.WithContext(ctx).Model(&domain.PickupAuthorization{}).
		Where("id = ? AND student_id = ? AND deleted_at IS NULL", authID, studentID).
		Update("deleted_at", time.Now()).Error
}

// ==================== Admin Authorization List ====================

// ListAuthorizationsByKindergarten 按园所列授权接送人，支持审核状态筛选、分页。
func (r *PickupRepo) ListAuthorizationsByKindergarten(
	ctx context.Context,
	kindergartenID uint64,
	reviewStatus *int8,
	offset, limit int,
) ([]AuthorizationListRow, int64, error) {
	base := r.db.WithContext(ctx).Table("pickup_authorizations AS pa").
		Joins("INNER JOIN students AS s ON s.id = pa.student_id AND s.deleted_at IS NULL").
		Joins("INNER JOIN classes AS c ON c.id = s.class_id AND c.deleted_at IS NULL").
		Where("s.kindergarten_id = ? AND pa.deleted_at IS NULL", kindergartenID)
	if reviewStatus != nil {
		base = base.Where("pa.review_status = ?", *reviewStatus)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []AuthorizationListRow
	err := base.Session(&gorm.Session{}).
		Select(`pa.id, pa.student_id, s.name AS student_name, c.name AS class_name,
			pa.name, pa.phone, pa.relation, pa.id_card, pa.avatar, pa.user_id,
			pa.expire_at, pa.status, pa.creator_id, pa.auth_type,
			pa.allow_pickup, pa.allow_view_record, pa.allow_view_status,
			pa.allow_growth_view, pa.notify_on_pickup, pa.review_status,
			pa.audited_at, pa.audit_remark, pa.created_at, pa.updated_at`).
		Order("pa.review_status ASC, pa.created_at DESC, pa.id DESC").
		Offset(offset).Limit(limit).
		Scan(&rows).Error
	return rows, total, err
}

// ==================== PickupQRCode ====================

// FindActiveQRCodeByStudent 查询指定学生当前有效的接送二维码。
func (r *PickupRepo) FindActiveQRCodeByStudent(
	ctx context.Context,
	studentID uint64,
) (*domain.PickupQRCode, error) {
	var qr domain.PickupQRCode
	err := r.db.WithContext(ctx).Model(&domain.PickupQRCode{}).
		Where("student_id = ? AND is_active = ?", studentID, true).
		Order("id DESC").
		First(&qr).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &qr, nil
}

// CreateQRCode 创建接送二维码记录。
//
// 显式 Select 全部字段：GORM 对 bool 零值（false）会跳过该列，
// 而 is_active 的 DB 默认值是 true —— 不 Select 会导致 IsActive:false 被写成 true。
func (r *PickupRepo) CreateQRCode(ctx context.Context, qr *domain.PickupQRCode) error {
	return r.db.WithContext(ctx).
		Select("StudentID", "AuthorizationID", "Code", "ExpireAt", "IsActive").
		Create(qr).Error
}

// DeactivateStudentQRCodes 将指定学生所有有效二维码标记为失效。
func (r *PickupRepo) DeactivateStudentQRCodes(ctx context.Context, studentID uint64) error {
	return r.db.WithContext(ctx).Model(&domain.PickupQRCode{}).
		Where("student_id = ? AND is_active = ?", studentID, true).
		Update("is_active", false).Error
}

// FindQRCodeByCode 按凭证串查凭证，并联查其绑定的授权接送人。
//
// 凭证自解释的依据：后端由 code 解出 student 与 authorization，前端不必传 student_id。
func (r *PickupRepo) FindQRCodeByCode(
	ctx context.Context,
	code string,
) (*domain.PickupQRCode, error) {
	var qr domain.PickupQRCode
	err := r.db.WithContext(ctx).
		Preload("Authorization").
		Where("code = ?", code).
		First(&qr).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &qr, nil
}

// ==================== PickupDailyDesignation ====================

// UpsertDailyDesignation 写入/覆盖某生某日的「今日接走人」指定。
// 同一 (student_id, date) 只留一条，重复指定覆盖旧值。
func (r *PickupRepo) UpsertDailyDesignation(
	ctx context.Context,
	kindergartenID, studentID, authorizationID uint64,
	day time.Time,
) error {
	d := day.Format("2006-01-02")
	var existing domain.PickupDailyDesignation
	err := r.db.WithContext(ctx).
		Where("student_id = ? AND date = ?", studentID, d).
		First(&existing).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.db.WithContext(ctx).Create(&domain.PickupDailyDesignation{
			KindergartenID:  kindergartenID,
			StudentID:       studentID,
			AuthorizationID: authorizationID,
			Date:            day,
		}).Error
	}
	return r.db.WithContext(ctx).Model(&existing).
		Updates(map[string]any{
			"authorization_id": authorizationID,
			"kindergarten_id":  kindergartenID,
		}).Error
}

// FindDailyDesignation 查某生某日的「今日接走人」指定，联查授权人。
func (r *PickupRepo) FindDailyDesignation(
	ctx context.Context,
	studentID uint64,
	day time.Time,
) (*domain.PickupDailyDesignation, error) {
	d := day.Format("2006-01-02")
	var row domain.PickupDailyDesignation
	err := r.db.WithContext(ctx).
		Preload("Authorization").
		Where("student_id = ? AND date = ?", studentID, d).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

// ==================== 托管序列与状态派生 ====================

// FindLastRecordOfDay 取某生某日最后一次托管转移记录（按登记时刻倒序取第一条）。
// 用于「不允许连续同向」判定与状态派生（在园/已接走）。
func (r *PickupRepo) FindLastRecordOfDay(
	ctx context.Context,
	studentID uint64,
	day time.Time,
) (*domain.PickupRecord, error) {
	d := day.Format("2006-01-02")
	var row domain.PickupRecord
	err := r.db.WithContext(ctx).
		Where("student_id = ? AND date = ? AND deleted_at IS NULL", studentID, d).
		Order("recorded_at DESC, id DESC").
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

// CountRecordsOfTypeAfter 某生某日某方向的登记次数（用于「重复接走」判定）。
func (r *PickupRepo) CountRecordsOfTypeAfter(
	ctx context.Context,
	studentID uint64,
	day time.Time,
	pickupType domain.PickupType,
) (int64, error) {
	d := day.Format("2006-01-02")
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.PickupRecord{}).
		Where("student_id = ? AND date = ? AND type = ? AND deleted_at IS NULL",
			studentID, d, pickupType).
		Count(&n).Error
	return n, err
}

// LastRecordRow 某学生当日托管状态摘要。
type LastRecordRow struct {
	StudentID            uint64    `gorm:"column:student_id"`
	Type                 int8      `gorm:"column:type"`
	Remark               string    `gorm:"column:remark"`
	IsAuthorizedOverride bool      `gorm:"column:is_authorized_override"`
	RecordedAt           time.Time `gorm:"column:recorded_at"`
	HasDropOff           bool      `gorm:"column:has_dropoff"`
	// MaxPickupAt 当日最后一次接走时刻；无接走为零值
	MaxPickupAt time.Time `gorm:"column:max_pickup_at"`
}

// LastRecordsOfDay 批量取每个学生当日的托管状态摘要。
//
// 状态派生的唯一依据：「看最后一条方向」，而非「当日是否有某类记录」。
// 另给出 HasDropOff（到过校）与 MaxPickupAt（最后一次接走时刻），
// 供超时未接的累计口径计算：在园区间的末端越过放学时刻即计入。
func (r *PickupRepo) LastRecordsOfDay(
	ctx context.Context,
	studentIDs []uint64,
	day time.Time,
) (map[uint64]LastRecordRow, error) {
	if len(studentIDs) == 0 {
		return map[uint64]LastRecordRow{}, nil
	}
	d := day.Format("2006-01-02")

	var rows []LastRecordRow
	err := r.db.WithContext(ctx).Raw(`
SELECT pr.student_id AS student_id,
       pr.type AS type,
       COALESCE(pr.remark,'') AS remark,
       COALESCE(pr.is_authorized_override, 0) AS is_authorized_override,
       pr.recorded_at AS recorded_at,
       EXISTS(SELECT 1 FROM pickup_records d
              WHERE d.student_id = pr.student_id AND d.deleted_at IS NULL
                AND d.date = ? AND d.type = 1) AS has_dropoff,
       (SELECT MAX(p.recorded_at) FROM pickup_records p
        WHERE p.student_id = pr.student_id AND p.deleted_at IS NULL
          AND p.date = ? AND p.type = 2) AS max_pickup_at
FROM pickup_records pr
WHERE pr.student_id IN ? AND pr.date = ? AND pr.deleted_at IS NULL
  AND pr.id = (
      SELECT pr2.id FROM pickup_records pr2
      WHERE pr2.student_id = pr.student_id AND pr2.deleted_at IS NULL
        AND pr2.date = ?
      ORDER BY pr2.recorded_at DESC, pr2.id DESC
      LIMIT 1
  )
`, d, d, studentIDs, d, d).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	out := make(map[uint64]LastRecordRow, len(rows))
	for _, row := range rows {
		out[row.StudentID] = row
	}
	return out, nil
}

// EverOverdueStudentIDs 「在园区间的末端越过班级放学时刻」的学生集合（累计口径）。
//
// 语义：该生当日曾有在园时刻晚于放学时刻 —— 即「超时未接」发生过。
// 以「在园区间末端 > 放学时刻」判定，而非「放学那一刻在园」：
//   - 07:52 入园、15:00 离园、放学 16:00 → 末端 15:00 未越线，不计
//   - 07:52 入园、17:00 离园、放学 16:00 → 末端 17:00 越线，计一次
//   - 07:52 入园、至今未离、now 18:00、放学 16:00 → 末端 now 越线，计一次
//
// 同一学生当日只计一次；随后被接走也不清零（spec TC-NEW-PKU-111）。
// deadlineHHMM 须由调用方先行解析回退（service 层 ResolvePickupDeadline）。
func (r *PickupRepo) EverOverdueStudentIDs(
	ctx context.Context,
	studentIDs []uint64,
	day time.Time,
	deadlineHHMM string,
	now time.Time,
) (map[uint64]bool, error) {
	out := make(map[uint64]bool)
	if len(studentIDs) == 0 {
		return out, nil
	}
	lastMap, err := r.LastRecordsOfDay(ctx, studentIDs, day)
	if err != nil {
		return nil, err
	}
	limit, err := time.Parse("15:04", deadlineHHMM)
	if err != nil {
		return nil, fmt.Errorf("invalid deadline %q: %w", deadlineHHMM, err)
	}
	deadlineClock := time.Date(2000, 1, 1, limit.Hour(), limit.Minute(), 0, 0, time.UTC)

	for id, lr := range lastMap {
		if !lr.HasDropOff {
			continue // 未到校，不属滞留
		}
		// 在园区间末端：仍在园 → now；已离园 → 最后一次接走时刻
		end := now
		if lr.Type == int8(domain.PickupTypePickUp) && !lr.MaxPickupAt.IsZero() {
			end = lr.MaxPickupAt
		}
		endClock := time.Date(2000, 1, 1, end.Hour(), end.Minute(), 0, 0, time.UTC)
		if endClock.After(deadlineClock) {
			out[id] = true
		}
	}
	return out, nil
}

// ==================== 滞留与超时未接 ====================

// OverdueStudentRow 滞留学生行。
type OverdueStudentRow struct {
	StudentID   uint64 `gorm:"column:student_id"`
	Name        string `gorm:"column:name"`
	Avatar      string `gorm:"column:avatar"`
	ClassID     uint64 `gorm:"column:class_id"`
	ClassName   string `gorm:"column:class_name"`
	ArriveTime  string `gorm:"column:arrive_time"`
	Deadline    string `gorm:"column:pickup_deadline"`
}

// ListOverdueStudents 查「已到园且超过班级放学时刻仍未接走」的学生。
//
// 口径：当日最后一次转移是送入（在园）且当前时刻已过班级放学时刻
// （未配置时回退缺省 16:00）。按学生计，不按事件数。
// classID 为 0 时查全园，否则限该班。
func (r *PickupRepo) ListOverdueStudents(
	ctx context.Context,
	kindergartenID, classID uint64,
	day time.Time,
	now time.Time,
) ([]OverdueStudentRow, error) {
	d := day.Format("2006-01-02")
	nowHM := now.Format("15:04")

	q := r.db.WithContext(ctx).Raw(`
SELECT s.id AS student_id,
       s.name AS name,
       COALESCE(s.avatar,'') AS avatar,
       s.class_id AS class_id,
       COALESCE(c.name,'') AS class_name,
       COALESCE(last_pr.arrive_time,'') AS arrive_time,
       COALESCE(NULLIF(c.pickup_deadline,''), '16:00') AS pickup_deadline
FROM students s
JOIN classes c ON c.id = s.class_id AND c.deleted_at IS NULL
JOIN (
    SELECT pr.student_id AS student_id,
           MAX(CASE WHEN pr.type = 1 THEN TIME_FORMAT(pr.recorded_at, '%H:%i') END) AS arrive_time
    FROM pickup_records pr
    WHERE pr.date = ? AND pr.deleted_at IS NULL
    GROUP BY pr.student_id
) last_pr ON last_pr.student_id = s.id
WHERE s.kindergarten_id = ? AND s.status = 1 AND s.deleted_at IS NULL
  AND (? = 0 OR s.class_id = ?)
  AND COALESCE(NULLIF(c.pickup_deadline,''), '16:00') < ?
  AND EXISTS (
      SELECT 1 FROM pickup_records pr2
      WHERE pr2.student_id = s.id AND pr2.deleted_at IS NULL
        AND pr2.date = ? AND pr2.type = 1
  )
  AND NOT EXISTS (
      SELECT 1 FROM pickup_records pr3
      WHERE pr3.student_id = s.id AND pr3.deleted_at IS NULL
        AND pr3.date = ? AND pr3.type = 2
        AND pr3.recorded_at >= (
            SELECT MAX(pr4.recorded_at) FROM pickup_records pr4
            WHERE pr4.student_id = s.id AND pr4.deleted_at IS NULL
              AND pr4.date = ? AND pr4.type = 1
        )
  )
ORDER BY c.name, s.id
`, d, kindergartenID, classID, classID, nowHM, d, d, d)

	var rows []OverdueStudentRow
	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// CountOverdueNotPicked 统计超时未接学生数（按学生计，非事件数）。
func (r *PickupRepo) CountOverdueNotPicked(
	ctx context.Context,
	kindergartenID, classID uint64,
	day time.Time,
	now time.Time,
) (int64, error) {
	rows, err := r.ListOverdueStudents(ctx, kindergartenID, classID, day, now)
	if err != nil {
		return 0, err
	}
	return int64(len(rows)), nil
}
