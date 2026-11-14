package repository

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"kindergarten-server/internal/domain"
)

// DietGroupObjectRepo 按学年管理的膳食集体对象（openspec: add-diet-group-objects, design D1）。
//
// 对象、餐次、就餐班级分三张表，读写都在同一事务内完成。关联表带软删除列且
// (group_object_id, meal_key) / (group_object_id, class_id) 是唯一索引：
// 软删除行仍占键位，所以整组保存必须就地 upsert、被移除的关联行用 Unscoped() 硬删。
type DietGroupObjectRepo struct {
	db *gorm.DB
}

func NewDietGroupObjectRepo(db *gorm.DB) *DietGroupObjectRepo {
	return &DietGroupObjectRepo{db: db}
}

// List 按学年分页返回集体对象，预加载餐次与就餐班级。
// 关键词按名称模糊匹配；academic_year_id 由服务层解析后传入，仓储不做默认学年回退。
func (r *DietGroupObjectRepo) List(
	ctx context.Context,
	kindergartenID, academicYearID uint64,
	keyword string,
	offset, limit int,
) ([]domain.DietGroupObject, int64, error) {
	q := r.db.WithContext(ctx).Model(&domain.DietGroupObject{}).
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

	var rows []domain.DietGroupObject
	err := q.Session(&gorm.Session{}).
		Preload("Meals", func(db *gorm.DB) *gorm.DB { return db.Order("sort ASC, id ASC") }).
		Preload("Classes", func(db *gorm.DB) *gorm.DB { return db.Order("class_id ASC, id ASC") }).
		Order("created_at DESC, id DESC").
		Offset(offset).Limit(limit).
		Find(&rows).Error
	return rows, total, err
}

// Get 读取单个对象及其餐次、就餐班级；跨园与不存在一律返回 (nil, nil)。
func (r *DietGroupObjectRepo) Get(
	ctx context.Context,
	kindergartenID, id uint64,
) (*domain.DietGroupObject, error) {
	var obj domain.DietGroupObject
	err := r.db.WithContext(ctx).
		Preload("Meals", func(db *gorm.DB) *gorm.DB { return db.Order("sort ASC, id ASC") }).
		Preload("Classes", func(db *gorm.DB) *gorm.DB { return db.Order("class_id ASC, id ASC") }).
		Where("kindergarten_id = ? AND id = ?", kindergartenID, id).
		First(&obj).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &obj, nil
}

// CountByName 统计同园同学年内的同名对象数；excludeID > 0 时排除自身（修改判重）。
// 名称唯一由服务层校验、不建唯一索引：本变更不提供删除，风险可控（design D1）。
func (r *DietGroupObjectRepo) CountByName(
	ctx context.Context,
	kindergartenID, academicYearID uint64,
	name string,
	excludeID uint64,
) (int64, error) {
	q := r.db.WithContext(ctx).Model(&domain.DietGroupObject{}).
		Where("kindergarten_id = ? AND academic_year_id = ? AND name = ?",
			kindergartenID, academicYearID, name)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	var count int64
	err := q.Count(&count).Error
	return count, err
}

// Create 在同一事务内写入对象与餐次/班级关联。
func (r *DietGroupObjectRepo) Create(ctx context.Context, obj *domain.DietGroupObject) error {
	if obj == nil {
		return errors.New("diet group object: nil object")
	}
	if obj.KindergartenID == 0 {
		return errors.New("diet group object: kindergarten_id required")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		meals := obj.Meals
		classes := obj.Classes
		if err := tx.Omit("Meals", "Classes").Create(obj).Error; err != nil {
			return err
		}
		if err := replaceDietGroupObjectMealsTx(tx, obj, meals); err != nil {
			return err
		}
		return replaceDietGroupObjectClassesTx(tx, obj, classes)
	})
}

// Update 更新对象标量字段，并整组同步餐次与就餐班级。
//
// 餐次/班级按业务键 upsert 而不是「软删重插」：唯一索引包含 group_object_id，
// 软删除行会永久占键。被移除的关联行硬删，避免列表/详情或后续保存读到残留。
func (r *DietGroupObjectRepo) Update(ctx context.Context, obj *domain.DietGroupObject) error {
	if obj == nil {
		return errors.New("diet group object: nil object")
	}
	if obj.KindergartenID == 0 || obj.ID == 0 {
		return errors.New("diet group object: id and kindergarten_id required")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&domain.DietGroupObject{}).
			Where("id = ? AND kindergarten_id = ?", obj.ID, obj.KindergartenID).
			Updates(map[string]any{
				"academic_year_id": obj.AcademicYearID,
				"name":             obj.Name,
				"care_type":        obj.CareType,
				"food_input_type":  obj.FoodInputType,
				"ratio_mode":       obj.RatioMode,
				"policy_name":      obj.PolicyName,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			// MySQL 默认返回实际变更行数：字段值完全相同时也是 0，
			// 只有确认行已不存在（并发删除/跨园）才报 NotFound。
			var existing int64
			if err := tx.Model(&domain.DietGroupObject{}).
				Where("id = ? AND kindergarten_id = ?", obj.ID, obj.KindergartenID).
				Count(&existing).Error; err != nil {
				return err
			}
			if existing == 0 {
				return gorm.ErrRecordNotFound
			}
		}
		if err := replaceDietGroupObjectMealsTx(tx, obj, obj.Meals); err != nil {
			return err
		}
		return replaceDietGroupObjectClassesTx(tx, obj, obj.Classes)
	})
}

// replaceDietGroupObjectMealsTx 删除不在入参中的餐次行，再按 (group_object_id, meal_key) upsert。
// 复用软删除行时把 deleted_at 置空，否则读取侧会继续把它们当成已删除。
func replaceDietGroupObjectMealsTx(
	tx *gorm.DB,
	obj *domain.DietGroupObject,
	meals []domain.DietGroupObjectMeal,
) error {
	keys := make([]string, 0, len(meals))
	for _, meal := range meals {
		keys = append(keys, string(meal.MealKey))
	}
	removed := tx.Unscoped().Where("group_object_id = ?", obj.ID)
	if len(keys) > 0 {
		removed = removed.Where("meal_key NOT IN ?", keys)
	}
	if err := removed.Delete(&domain.DietGroupObjectMeal{}).Error; err != nil {
		return err
	}
	if len(meals) == 0 {
		return nil
	}

	prepared := make([]domain.DietGroupObjectMeal, len(meals))
	copy(prepared, meals)
	for i := range prepared {
		prepared[i].BaseModel = domain.BaseModel{}
		prepared[i].GroupObjectID = obj.ID
		prepared[i].KindergartenID = obj.KindergartenID
	}
	obj.Meals = prepared
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "group_object_id"}, {Name: "meal_key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"name", "time", "energy_percent", "sort", "enabled", "deleted_at", "updated_at",
		}),
	}).Create(&prepared).Error
}

// replaceDietGroupObjectClassesTx 删除不在入参中的班级关联，再按
// (group_object_id, class_id) upsert；语义与餐次一致。
func replaceDietGroupObjectClassesTx(
	tx *gorm.DB,
	obj *domain.DietGroupObject,
	classes []domain.DietGroupObjectClass,
) error {
	classIDs := make([]uint64, 0, len(classes))
	for _, link := range classes {
		classIDs = append(classIDs, link.ClassID)
	}
	removed := tx.Unscoped().Where("group_object_id = ?", obj.ID)
	if len(classIDs) > 0 {
		removed = removed.Where("class_id NOT IN ?", classIDs)
	}
	if err := removed.Delete(&domain.DietGroupObjectClass{}).Error; err != nil {
		return err
	}
	if len(classes) == 0 {
		return nil
	}

	prepared := make([]domain.DietGroupObjectClass, len(classes))
	copy(prepared, classes)
	for i := range prepared {
		prepared[i].BaseModel = domain.BaseModel{}
		prepared[i].GroupObjectID = obj.ID
		prepared[i].KindergartenID = obj.KindergartenID
	}
	obj.Classes = prepared
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "group_object_id"}, {Name: "class_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"deleted_at", "updated_at",
		}),
	}).Create(&prepared).Error
}
