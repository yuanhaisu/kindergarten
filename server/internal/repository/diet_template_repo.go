package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

// DietTemplateListRow 模板列表行：契约只返回名称、天数、餐次集合与条目数。
type DietTemplateListRow struct {
	ID        uint64 `gorm:"column:id"`
	Name      string `gorm:"column:name"`
	DayNum    int    `gorm:"column:day_num"`
	MealKeys  string `gorm:"column:meal_keys"`
	ItemCount int64  `gorm:"column:item_count"`
}

// DietTemplateRepo 周食谱模板（按园种子化的只读副本）。
type DietTemplateRepo struct {
	db *gorm.DB
}

func NewDietTemplateRepo(db *gorm.DB) *DietTemplateRepo {
	return &DietTemplateRepo{db: db}
}

func (r *DietTemplateRepo) List(ctx context.Context, kindergartenID uint64) ([]DietTemplateListRow, error) {
	var rows []DietTemplateListRow
	err := r.db.WithContext(ctx).Table("diet_template_plans AS t").
		Where("t.kindergarten_id = ? AND t.deleted_at IS NULL", kindergartenID).
		Select(`t.id, t.name, t.day_num, t.meal_keys,
			(SELECT COUNT(*) FROM diet_template_plan_items i
				WHERE i.template_id = t.id AND i.deleted_at IS NULL) AS item_count`).
		Order("t.sort ASC, t.id ASC").
		Scan(&rows).Error
	return rows, err
}

func (r *DietTemplateRepo) FindByIDWithItems(
	ctx context.Context,
	kindergartenID, id uint64,
) (*domain.TemplatePlan, error) {
	var tpl domain.TemplatePlan
	err := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND id = ?", kindergartenID, id).
		First(&tpl).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var items []domain.TemplatePlanItem
	if err := r.db.WithContext(ctx).
		Where("template_id = ? AND kindergarten_id = ?", tpl.ID, kindergartenID).
		Order("weekday ASC, sort ASC, id ASC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	tpl.Items = items
	return &tpl, nil
}
