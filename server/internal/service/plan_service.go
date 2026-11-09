package service

import (
	"context"
	"fmt"
	"strings"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// PlanService 套餐目录。目录可改，但已成交订单只认快照（ADR-0013）。
type PlanService struct {
	plans *repository.PlanRepo
	logs  *repository.PlatformLogRepo
}

func NewPlanService(plans *repository.PlanRepo, logs *repository.PlatformLogRepo) *PlanService {
	return &PlanService{plans: plans, logs: logs}
}

type PlanRow struct {
	ID          uint64   `json:"id"`
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	Tier        int      `json:"tier"`
	PriceCents  int64    `json:"price_cents"`
	TermMonths  int      `json:"term_months"`
	ModuleCodes []string `json:"module_codes"`
	Status      string   `json:"status"`
	CreatedAt   string   `json:"created_at"`
}

type PlanRequest struct {
	Code        string   `json:"code"`
	Name        string   `json:"name" binding:"required,min=1,max=64"`
	Tier        int      `json:"tier"`
	PriceCents  int64    `json:"price_cents"`
	TermMonths  int      `json:"term_months"`
	ModuleCodes []string `json:"module_codes"`
	Status      string   `json:"status"`
}

func (s *PlanService) ListModules() []ModuleDef { return AllModules() }

func (s *PlanService) ListPlans(ctx context.Context, status string) ([]PlanRow, int64, error) {
	list, total, err := s.plans.List(ctx, status)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]uint64, 0, len(list))
	for _, p := range list {
		ids = append(ids, p.ID)
	}
	mods, err := s.plans.ListModuleCodesByPlanIDs(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	rows := make([]PlanRow, 0, len(list))
	for _, p := range list {
		rows = append(rows, PlanRow{
			ID: p.ID, Code: p.Code, Name: p.Name, Tier: p.Tier,
			PriceCents: p.PriceCents, TermMonths: p.TermMonths,
			ModuleCodes: mods[p.ID], Status: p.Status,
			CreatedAt: p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	return rows, total, nil
}

func (s *PlanService) Create(ctx context.Context, req PlanRequest, operator, ip string) (*domain.Plan, error) {
	if err := validatePlanRequest(req, true); err != nil {
		return nil, err
	}
	if existing, err := s.plans.FindByCode(ctx, req.Code); err != nil {
		return nil, err
	} else if existing != nil {
		return nil, fmt.Errorf("PLAN_CODE_EXISTS")
	}
	if req.TermMonths <= 0 {
		req.TermMonths = 6
	}
	if req.Status == "" {
		req.Status = "active"
	}
	plan := &domain.Plan{
		Code: req.Code, Name: req.Name, Tier: req.Tier,
		PriceCents: req.PriceCents, TermMonths: req.TermMonths, Status: req.Status,
	}
	if err := s.plans.Create(ctx, plan); err != nil {
		return nil, err
	}
	if err := s.plans.ReplaceModules(ctx, plan.ID, normalizeModuleCodes(req.ModuleCodes)); err != nil {
		return nil, err
	}
	_ = s.logs.Create(ctx, &domain.PlatformOperationLog{
		Operator: operator, Module: "plan", Action: "create", TargetType: "plan", TargetID: plan.ID,
		Detail: req.Name, IP: ip,
	})
	return plan, nil
}

func (s *PlanService) Update(ctx context.Context, id uint64, req PlanRequest, operator, ip string) error {
	if err := validatePlanRequest(req, false); err != nil {
		return err
	}
	plan, err := s.plans.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if plan == nil {
		return fmt.Errorf("PLAN_NOT_FOUND")
	}
	cols := map[string]any{"name": req.Name, "price_cents": req.PriceCents}
	if req.Tier > 0 {
		cols["tier"] = req.Tier
	}
	if req.TermMonths > 0 {
		cols["term_months"] = req.TermMonths
	}
	if req.Status != "" {
		cols["status"] = req.Status
	}
	if err := s.plans.Update(ctx, id, cols); err != nil {
		return err
	}
	if req.ModuleCodes != nil {
		if err := s.plans.ReplaceModules(ctx, id, normalizeModuleCodes(req.ModuleCodes)); err != nil {
			return err
		}
	}
	return s.logs.Create(ctx, &domain.PlatformOperationLog{
		Operator: operator, Module: "plan", Action: "update", TargetType: "plan", TargetID: id,
		Detail: req.Name, IP: ip,
	})
}

func (s *PlanService) ListTiers(ctx context.Context, planID uint64) ([]domain.PlanTier, error) {
	return s.plans.ListTiers(ctx, planID)
}

// ReplaceTiers 整体替换档位。档位变更只影响之后创建的订单。
func (s *PlanService) ReplaceTiers(ctx context.Context, planID uint64, tiers []domain.PlanTier, operator, ip string) error {
	plan, err := s.plans.FindByID(ctx, planID)
	if err != nil {
		return err
	}
	if plan == nil {
		return fmt.Errorf("PLAN_NOT_FOUND")
	}
	seen := make(map[int]struct{}, len(tiers))
	for _, t := range tiers {
		if t.Tier <= 0 {
			return fmt.Errorf("INVALID_TIER")
		}
		if _, dup := seen[t.Tier]; dup {
			return fmt.Errorf("DUPLICATE_TIER")
		}
		seen[t.Tier] = struct{}{}
		if t.MaxStudents != 0 && t.MaxStudents < t.MinStudents {
			return fmt.Errorf("INVALID_TIER_RANGE")
		}
		if t.PriceMultiplier <= 0 {
			t.PriceMultiplier = 100
		}
	}
	if err := s.plans.ReplaceTiers(ctx, planID, tiers); err != nil {
		return err
	}
	return s.logs.Create(ctx, &domain.PlatformOperationLog{
		Operator: operator, Module: "plan", Action: "replace_tiers", TargetType: "plan", TargetID: planID,
		Detail: fmt.Sprintf("%d 档", len(tiers)), IP: ip,
	})
}

func validatePlanRequest(req PlanRequest, requireCode bool) error {
	if strings.TrimSpace(req.Name) == "" {
		return fmt.Errorf("PLAN_NAME_REQUIRED")
	}
	if requireCode {
		if strings.TrimSpace(req.Code) == "" {
			return fmt.Errorf("PLAN_CODE_REQUIRED")
		}
	}
	if req.PriceCents < 0 {
		return fmt.Errorf("INVALID_PRICE")
	}
	for _, code := range req.ModuleCodes {
		if !IsRegisteredModule(code) {
			return fmt.Errorf("UNKNOWN_MODULE: %s", code)
		}
	}
	return nil
}
