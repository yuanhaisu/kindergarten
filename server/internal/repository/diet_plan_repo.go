package repository

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

// DietPlanRepo 周食谱。列表只返回主表字段（不含明细），详情才加载 items。
type DietPlanRepo struct {
	db *gorm.DB
}

func NewDietPlanRepo(db *gorm.DB) *DietPlanRepo {
	return &DietPlanRepo{db: db}
}

func (r *DietPlanRepo) List(
	ctx context.Context,
	kindergartenID, academicYearID uint64,
	keyword string,
	offset, limit int,
) ([]domain.MealPlan, int64, error) {
	q := r.db.WithContext(ctx).Model(&domain.MealPlan{}).
		Where("kindergarten_id = ?", kindergartenID)
	if academicYearID > 0 {
		q = q.Where("academic_year_id = ?", academicYearID)
	}
	if kw := strings.TrimSpace(keyword); kw != "" {
		q = q.Where("name LIKE ?", "%"+kw+"%")
	}

	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []domain.MealPlan
	err := q.Session(&gorm.Session{}).
		Order("start_date DESC, id DESC").
		Offset(offset).Limit(limit).
		Find(&rows).Error
	return rows, total, err
}

func (r *DietPlanRepo) FindByIDWithItems(
	ctx context.Context,
	kindergartenID, id uint64,
) (*domain.MealPlan, error) {
	var plan domain.MealPlan
	err := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND id = ?", kindergartenID, id).
		First(&plan).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var items []domain.MealPlanItem
	if err := r.db.WithContext(ctx).
		Where("plan_id = ? AND kindergarten_id = ?", plan.ID, kindergartenID).
		Order("date ASC, sort ASC, id ASC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	plan.Items = items
	return &plan, nil
}

// Create 在同一事务内写入周食谱与明细。
func (r *DietPlanRepo) Create(ctx context.Context, plan *domain.MealPlan) error {
	if plan == nil {
		return errors.New("diet plan: nil plan")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		items := plan.Items
		if err := tx.Omit("Items").Create(plan).Error; err != nil {
			return err
		}
		if len(items) == 0 {
			plan.Items = nil
			return nil
		}
		for i := range items {
			items[i].BaseModel = domain.BaseModel{}
			items[i].PlanID = plan.ID
			items[i].KindergartenID = plan.KindergartenID
		}
		if err := tx.Create(&items).Error; err != nil {
			return err
		}
		plan.Items = items
		return nil
	})
}

// Update 只更新可变化的表头字段。config_snapshot 在创建时固化，
// 后续任何修改都不得改写它（design D3、TC-NEW-DIE-0008）。
func (r *DietPlanRepo) Update(ctx context.Context, plan *domain.MealPlan) error {
	if plan == nil {
		return errors.New("diet plan: nil plan")
	}
	return r.db.WithContext(ctx).
		Model(&domain.MealPlan{}).
		Where("id = ? AND kindergarten_id = ?", plan.ID, plan.KindergartenID).
		Updates(map[string]any{
			"name":         plan.Name,
			"start_date":   plan.StartDate,
			"end_date":     plan.EndDate,
			"day_num":      plan.DayNum,
			"week_of_year": plan.WeekOfYear,
		}).Error
}

// Delete 软删除周食谱并硬删其明细。
func (r *DietPlanRepo) Delete(ctx context.Context, kindergartenID, id uint64) (int64, error) {
	var affected int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().
			Where("plan_id = ? AND kindergarten_id = ?", id, kindergartenID).
			Delete(&domain.MealPlanItem{}).Error; err != nil {
			return err
		}
		res := tx.Where("kindergarten_id = ? AND id = ?", kindergartenID, id).
			Delete(&domain.MealPlan{})
		affected = res.RowsAffected
		return res.Error
	})
	return affected, err
}

// ReplaceItems 整单替换周食谱明细。
//
// 硬删旧行：MealPlanItem 带软删除列，软删行虽然不会读出来，但会无限累积，
// 并且与「保存即整单替换」的语义不符。plan_id/kindergarten_id 之外的列由服务层
// 负责填充，这里只兜底归属字段，避免跨单/跨园写入。
func (r *DietPlanRepo) ReplaceItems(
	ctx context.Context,
	planID, kindergartenID uint64,
	items []domain.MealPlanItem,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().
			Where("plan_id = ? AND kindergarten_id = ?", planID, kindergartenID).
			Delete(&domain.MealPlanItem{}).Error; err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}

		prepared := make([]domain.MealPlanItem, len(items))
		copy(prepared, items)
		for i := range prepared {
			prepared[i].BaseModel = domain.BaseModel{}
			prepared[i].PlanID = planID
			prepared[i].KindergartenID = kindergartenID
		}
		return tx.Create(&prepared).Error
	})
}
