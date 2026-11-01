package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

// DietConfigRepo 园所级供餐配置（单例）与餐次。
type DietConfigRepo struct {
	db *gorm.DB
}

func NewDietConfigRepo(db *gorm.DB) *DietConfigRepo {
	return &DietConfigRepo{db: db}
}

// GetByKindergarten 读取园所配置及其餐次；尚未配置时返回 (nil, nil)。
// 餐次按 sort 升序返回，调用方无需再排序。
func (r *DietConfigRepo) GetByKindergarten(ctx context.Context, kindergartenID uint64) (*domain.MealConfig, error) {
	var cfg domain.MealConfig
	err := r.db.WithContext(ctx).
		Where("kindergarten_id = ?", kindergartenID).
		First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var meals []domain.MealConfigMeal
	if err := r.db.WithContext(ctx).
		Where("config_id = ? AND kindergarten_id = ?", cfg.ID, kindergartenID).
		Order("sort ASC, id ASC").
		Find(&meals).Error; err != nil {
		return nil, err
	}
	cfg.Meals = meals
	return &cfg, nil
}

// Save 以 kindergarten_id 为键保存单例配置，并在同一事务内整组替换餐次。
//
// 餐次必须硬删后重插：diet_meal_config_meals 的 (config_id, meal_key) 是唯一索引，
// 而该表带软删除列；软删旧行会永久占住键位，第二次保存必然撞键。
func (r *DietConfigRepo) Save(ctx context.Context, cfg *domain.MealConfig) error {
	if cfg == nil {
		return errors.New("diet config: nil config")
	}
	if cfg.KindergartenID == 0 {
		return errors.New("diet config: kindergarten_id required")
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing domain.MealConfig
		err := tx.Where("kindergarten_id = ?", cfg.KindergartenID).First(&existing).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			if err := tx.Omit("Meals").Create(cfg).Error; err != nil {
				return err
			}
		case err != nil:
			return err
		default:
			if err := tx.Model(&domain.MealConfig{}).
				Where("id = ?", existing.ID).
				Updates(map[string]any{
					"care_type":       cfg.CareType,
					"food_input_type": cfg.FoodInputType,
				}).Error; err != nil {
				return err
			}
			cfg.ID = existing.ID
		}

		if err := tx.Unscoped().
			Where("config_id = ?", cfg.ID).
			Delete(&domain.MealConfigMeal{}).Error; err != nil {
			return err
		}
		if len(cfg.Meals) == 0 {
			return nil
		}

		for i := range cfg.Meals {
			// 复用读回的餐次时 ID/时间戳属于旧行；整组重插必须清空，
			// 否则 GORM 会把旧的 CreatedAt 一并写回。
			cfg.Meals[i].BaseModel = domain.BaseModel{}
			cfg.Meals[i].ConfigID = cfg.ID
			cfg.Meals[i].KindergartenID = cfg.KindergartenID
		}
		return tx.Create(&cfg.Meals).Error
	})
}
