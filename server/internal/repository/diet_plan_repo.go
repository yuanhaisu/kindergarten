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

// FindByName 在同一学年内按名称查未删除的周食谱；excludeID > 0 时排除自身（改名判重）。
// 名称唯一由服务层保证、不建唯一索引：软删除行仍会占唯一键，删除后应能复用名称（design D2）。
func (r *DietPlanRepo) FindByName(
	ctx context.Context,
	kindergartenID, academicYearID uint64,
	name string,
	excludeID uint64,
) (*domain.MealPlan, error) {
	q := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND academic_year_id = ? AND name = ?",
			kindergartenID, academicYearID, name)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	var plan domain.MealPlan
	err := q.First(&plan).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &plan, nil
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

// UpdateWithItems 在一个事务内更新表头（名称/天数）并整单替换明细，返回受影响的行数。
//
// 整单保存必须单事务：先更新表头再替换明细，任何一步失败都不得留下「新名称 + 旧明细」
// 或「旧名称 + 新明细」的中间态。起始/结束日期在创建后不可改，这里不写。
// 表头更新 0 行（并发被删/跨园）时直接返回，不写入任何明细。
func (r *DietPlanRepo) UpdateWithItems(
	ctx context.Context,
	plan *domain.MealPlan,
	items []domain.MealPlanItem,
) (int64, error) {
	if plan == nil {
		return 0, errors.New("diet plan: nil plan")
	}
	var affected int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&domain.MealPlan{}).
			Where("id = ? AND kindergarten_id = ?", plan.ID, plan.KindergartenID).
			Updates(map[string]any{
				"name":    plan.Name,
				"day_num": plan.DayNum,
			})
		if res.Error != nil {
			return res.Error
		}
		affected = res.RowsAffected
		if affected == 0 {
			// MySQL 默认返回「实际变更行数」：表头值完全相同时也会是 0。
			// 只有确认行已不存在（并发删除/跨园）才提前返回，避免误报 404。
			var existing int64
			if err := tx.Model(&domain.MealPlan{}).
				Where("id = ? AND kindergarten_id = ?", plan.ID, plan.KindergartenID).
				Count(&existing).Error; err != nil {
				return err
			}
			if existing == 0 {
				return nil
			}
			affected = 1
		}
		if err := tx.Unscoped().
			Where("plan_id = ? AND kindergarten_id = ?", plan.ID, plan.KindergartenID).
			Delete(&domain.MealPlanItem{}).Error; err != nil {
			return err
		}
		if len(items) == 0 {
			plan.Items = nil
			return nil
		}

		prepared := make([]domain.MealPlanItem, len(items))
		copy(prepared, items)
		for i := range prepared {
			prepared[i].BaseModel = domain.BaseModel{}
			prepared[i].PlanID = plan.ID
			prepared[i].KindergartenID = plan.KindergartenID
		}
		if err := tx.Create(&prepared).Error; err != nil {
			return err
		}
		plan.Items = prepared
		return nil
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
