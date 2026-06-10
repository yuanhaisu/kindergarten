package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

var ErrAchievementNotFound = errors.New("ACHIEVEMENT_NOT_FOUND")

type AchievementRepo struct {
	db *gorm.DB
}

func NewAchievementRepo(db *gorm.DB) *AchievementRepo {
	return &AchievementRepo{db: db}
}

// AchievementListRow 列表行（含 earned_count 子查询）。
type AchievementListRow struct {
	ID          uint64    `gorm:"column:id"`
	Name        string    `gorm:"column:name"`
	Description string    `gorm:"column:description"`
	Icon        string    `gorm:"column:icon"`
	Category    string    `gorm:"column:category"`
	Condition   string    `gorm:"column:condition"`
	TargetValue int       `gorm:"column:target_value"`
	Status      int8      `gorm:"column:status"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	EarnedCount int64     `gorm:"column:earned_count"`
}

const achievementEarnedCountSQL = `(SELECT COUNT(*) FROM student_achievements sa
INNER JOIN students s ON s.id = sa.student_id AND s.deleted_at IS NULL
WHERE sa.achievement_id = a.id AND s.kindergarten_id = a.kindergarten_id) AS earned_count`

func (r *AchievementRepo) List(
	ctx context.Context,
	kindergartenID uint64,
	offset, limit int,
) ([]AchievementListRow, int64, error) {
	base := r.db.WithContext(ctx).Table("achievements AS a").
		Where("a.kindergarten_id = ? AND a.deleted_at IS NULL", kindergartenID)
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []AchievementListRow
	sel := base.Session(&gorm.Session{}).
		Select("a.id, a.name, a.description, a.icon, a.category, a.condition, a.target_value, a.status, a.created_at, "+achievementEarnedCountSQL).
		Order("a.created_at DESC, a.id DESC").
		Offset(offset).Limit(limit)
	if err := sel.Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *AchievementRepo) FindByID(ctx context.Context, kindergartenID, id uint64) (*domain.Achievement, error) {
	var row domain.Achievement
	err := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND id = ? AND deleted_at IS NULL", kindergartenID, id).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAchievementNotFound
		}
		return nil, err
	}
	return &row, nil
}

func (r *AchievementRepo) Create(ctx context.Context, row *domain.Achievement) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *AchievementRepo) Update(ctx context.Context, kindergartenID, id uint64, patch map[string]interface{}) error {
	res := r.db.WithContext(ctx).Model(&domain.Achievement{}).
		Where("kindergarten_id = ? AND id = ? AND deleted_at IS NULL", kindergartenID, id).
		Updates(patch)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrAchievementNotFound
	}
	return nil
}

func (r *AchievementRepo) SoftDelete(ctx context.Context, kindergartenID, id uint64) error {
	res := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND id = ? AND deleted_at IS NULL", kindergartenID, id).
		Delete(&domain.Achievement{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrAchievementNotFound
	}
	return nil
}
