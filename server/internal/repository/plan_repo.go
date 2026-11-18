package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

// PlanRepo 套餐目录仓储。目录可改，但成交订单只认快照（ADR-0013）。
type PlanRepo struct {
	db *gorm.DB
}

func NewPlanRepo(db *gorm.DB) *PlanRepo {
	return &PlanRepo{db: db}
}

func (r *PlanRepo) Create(ctx context.Context, p *domain.Plan) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *PlanRepo) FindByID(ctx context.Context, id uint64) (*domain.Plan, error) {
	var p domain.Plan
	err := r.db.WithContext(ctx).First(&p, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

func (r *PlanRepo) FindByCode(ctx context.Context, code string) (*domain.Plan, error) {
	var p domain.Plan
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

// List 返回满足状态筛选的一页套餐与真实总数。
// total 在分页之前统计，与 offset/limit 无关。
func (r *PlanRepo) List(ctx context.Context, status string, offset, limit int) ([]domain.Plan, int64, error) {
	q := r.db.WithContext(ctx).Model(&domain.Plan{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []domain.Plan
	if err := q.Order("tier ASC, id ASC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *PlanRepo) Update(ctx context.Context, id uint64, cols map[string]any) error {
	if len(cols) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Model(&domain.Plan{}).Where("id = ?", id).Updates(cols).Error
}

func (r *PlanRepo) ListModuleCodes(ctx context.Context, planID uint64) ([]string, error) {
	var codes []string
	err := r.db.WithContext(ctx).Model(&domain.PlanModule{}).
		Where("plan_id = ?", planID).
		Order("module_code ASC").
		Pluck("module_code", &codes).Error
	return codes, err
}

func (r *PlanRepo) ListModuleCodesByPlanIDs(ctx context.Context, planIDs []uint64) (map[uint64][]string, error) {
	out := make(map[uint64][]string, len(planIDs))
	if len(planIDs) == 0 {
		return out, nil
	}
	var rows []domain.PlanModule
	if err := r.db.WithContext(ctx).Where("plan_id IN ?", planIDs).Order("module_code ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.PlanID] = append(out[row.PlanID], row.ModuleCode)
	}
	return out, nil
}

// ReplaceModules 整体替换套餐模块。目录修改不影响存量订单
// （订单只读自己的 plan_snapshot）。
func (r *PlanRepo) ReplaceModules(ctx context.Context, planID uint64, codes []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("plan_id = ?", planID).Delete(&domain.PlanModule{}).Error; err != nil {
			return err
		}
		if len(codes) == 0 {
			return nil
		}
		rows := make([]domain.PlanModule, 0, len(codes))
		for _, code := range codes {
			rows = append(rows, domain.PlanModule{PlanID: planID, ModuleCode: code})
		}
		return tx.Create(&rows).Error
	})
}

func (r *PlanRepo) ListTiers(ctx context.Context, planID uint64) ([]domain.PlanTier, error) {
	var list []domain.PlanTier
	err := r.db.WithContext(ctx).Where("plan_id = ?", planID).Order("tier ASC").Find(&list).Error
	return list, err
}

// ReplaceTiers 整体替换档位配置。
func (r *PlanRepo) ReplaceTiers(ctx context.Context, planID uint64, tiers []domain.PlanTier) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("plan_id = ?", planID).Delete(&domain.PlanTier{}).Error; err != nil {
			return err
		}
		if len(tiers) == 0 {
			return nil
		}
		rows := make([]domain.PlanTier, 0, len(tiers))
		for _, t := range tiers {
			t.ID = 0
			t.PlanID = planID
			rows = append(rows, t)
		}
		return tx.Create(&rows).Error
	})
}
