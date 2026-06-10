package repository

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

type HealthCheckRepo struct {
	db *gorm.DB
}

func NewHealthCheckRepo(db *gorm.DB) *HealthCheckRepo {
	return &HealthCheckRepo{db: db}
}

// ==================== Queue ====================

// HealthCheckQueueRow 晨检队列行（待检 + 已完成）
type HealthCheckQueueRow struct {
	ID             uint64 `gorm:"column:id"`
	PickupRecordID uint64 `gorm:"column:pickup_record_id"`
	StudentID      uint64 `gorm:"column:student_id"`
	StudentName    string `gorm:"column:student_name"`
	ClassName      string `gorm:"column:class_name"`
	ClassID        uint64 `gorm:"column:class_id"`
	CheckStatus    int8   `gorm:"column:check_status"`
	OverallStatus  int8   `gorm:"column:overall_status"`
	PickupTime     string `gorm:"column:pickup_time"`
	Avatar         string `gorm:"column:avatar"`
}

func (r *HealthCheckRepo) ListQueue(ctx context.Context, kgID uint64, classID *uint64, date string) ([]HealthCheckQueueRow, error) {
	base := r.db.WithContext(ctx).Table("morning_health_checks AS hc").
		Joins("INNER JOIN students AS s ON s.id = hc.student_id AND s.deleted_at IS NULL").
		Joins("INNER JOIN classes AS c ON c.id = hc.class_id AND c.deleted_at IS NULL").
		Joins("INNER JOIN pickup_records AS pr ON pr.id = hc.pickup_record_id AND pr.deleted_at IS NULL").
		Where("hc.kindergarten_id = ? AND hc.date = ? AND hc.deleted_at IS NULL", kgID, date)
	if classID != nil && *classID > 0 {
		base = base.Where("hc.class_id = ?", *classID)
	}
	var rows []HealthCheckQueueRow
	err := base.Select(`hc.id, hc.pickup_record_id, hc.student_id, s.name AS student_name,
		c.name AS class_name, c.id AS class_id, hc.check_status, hc.overall_status,
		COALESCE(s.avatar,'') AS avatar,
		DATE_FORMAT(pr.created_at, '%H:%i') AS pickup_time`).
		Order("hc.check_status ASC, pr.created_at ASC").
		Find(&rows).Error
	return rows, err
}

// ==================== CRUD ====================

func (r *HealthCheckRepo) FindByID(ctx context.Context, kgID, id uint64) (*domain.MorningHealthCheck, error) {
	var row domain.MorningHealthCheck
	err := r.db.WithContext(ctx).Where("id = ? AND kindergarten_id = ?", id, kgID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

func (r *HealthCheckRepo) Create(ctx context.Context, row *domain.MorningHealthCheck) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *HealthCheckRepo) Update(ctx context.Context, row *domain.MorningHealthCheck) error {
	return r.db.WithContext(ctx).Save(row).Error
}

// ==================== Media ====================

func (r *HealthCheckRepo) CreateMedia(ctx context.Context, m *domain.HealthCheckMedia) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *HealthCheckRepo) ListMediaByCheckID(ctx context.Context, checkID uint64) ([]domain.HealthCheckMedia, error) {
	var rows []domain.HealthCheckMedia
	err := r.db.WithContext(ctx).Where("check_id = ?", checkID).Order("id ASC").Find(&rows).Error
	return rows, err
}

// ==================== Stats ====================

type HealthCheckStatsRow struct {
	ClassID   uint64 `gorm:"column:class_id"`
	ClassName string `gorm:"column:class_name"`
	Total     int64  `gorm:"column:total"`
	Pending   int64  `gorm:"column:pending"`
	Completed int64  `gorm:"column:completed"`
	Healthy   int64  `gorm:"column:healthy"`
	Observe   int64  `gorm:"column:observe"`
	Reject    int64  `gorm:"column:reject"`
}

func (r *HealthCheckRepo) StatsByDate(ctx context.Context, kgID uint64, date string) ([]HealthCheckStatsRow, error) {
	var rows []HealthCheckStatsRow
	err := r.db.WithContext(ctx).Raw(`
	SELECT hc.class_id, c.name AS class_name,
		COUNT(*) AS total,
		SUM(CASE WHEN hc.check_status = 0 THEN 1 ELSE 0 END) AS pending,
		SUM(CASE WHEN hc.check_status = 1 THEN 1 ELSE 0 END) AS completed,
		SUM(CASE WHEN hc.overall_status = 0 AND hc.check_status = 1 THEN 1 ELSE 0 END) AS healthy,
		SUM(CASE WHEN hc.overall_status = 1 THEN 1 ELSE 0 END) AS observe,
		SUM(CASE WHEN hc.overall_status = 2 THEN 1 ELSE 0 END) AS reject
	FROM morning_health_checks hc
	INNER JOIN classes c ON c.id = hc.class_id AND c.deleted_at IS NULL
	WHERE hc.kindergarten_id = ? AND hc.date = ? AND hc.deleted_at IS NULL
	GROUP BY hc.class_id, c.name
	ORDER BY c.name`, kgID, date).Scan(&rows).Error
	return rows, err
}

type SymptomCount struct {
	Tag   string `gorm:"column:tag"`
	Count int64  `gorm:"column:cnt"`
}

func (r *HealthCheckRepo) SymptomRanking(ctx context.Context, kgID uint64, date string) ([]SymptomCount, error) {
	var tags []string
	err := r.db.WithContext(ctx).Model(&domain.MorningHealthCheck{}).
		Where("kindergarten_id = ? AND date = ? AND symptom_tags != '' AND deleted_at IS NULL", kgID, date).
		Pluck("symptom_tags", &tags).Error
	if err != nil {
		return nil, err
	}
	m := make(map[string]int64)
	for _, t := range tags {
		for _, tag := range strings.Split(t, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				m[tag]++
			}
		}
	}
	var out []SymptomCount
	for k, v := range m {
		out = append(out, SymptomCount{Tag: k, Count: v})
	}
	return out, nil
}
