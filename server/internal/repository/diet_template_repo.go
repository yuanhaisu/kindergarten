package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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

// FindByKey 按 (kindergarten_id, template_key) 查模板头，供种子流程判重。
// 不返回条目：判重只需要知道模板是否已存在。
func (r *DietTemplateRepo) FindByKey(
	ctx context.Context,
	kindergartenID uint64,
	templateKey string,
) (*domain.TemplatePlan, error) {
	if templateKey == "" {
		return nil, nil
	}
	var tpl domain.TemplatePlan
	err := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND template_key = ?", kindergartenID, templateKey).
		First(&tpl).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &tpl, nil
}

// CreateWithItems 在一个事务内写入模板头与全部条目。
// 模板条目引用本园 food_id，写入前服务层已按名称解析并跳过缺失项。
func (r *DietTemplateRepo) CreateWithItems(ctx context.Context, tpl *domain.TemplatePlan) error {
	if tpl == nil {
		return errors.New("diet template: nil template")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Omit(clause.Associations)：GORM 默认会顺带写入 tpl.Items，
		// 之后我们还要显式写条目，会撞主键。
		if err := tx.Omit(clause.Associations).Create(tpl).Error; err != nil {
			return err
		}
		for i := range tpl.Items {
			tpl.Items[i].TemplateID = tpl.ID
		}
		if len(tpl.Items) == 0 {
			return nil
		}
		return tx.CreateInBatches(&tpl.Items, 100).Error
	})
}
