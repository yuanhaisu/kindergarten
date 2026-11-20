package service

import (
	"context"
	"fmt"
	"time"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// DistributorService 分销商名单与只读佣金台账（ADR-0016）。
type DistributorService struct {
	dists  *repository.DistributorRepo
	comm   *repository.CommissionRepo
	orders *repository.OrderRepo
	kgs    *repository.KindergartenRepo
	logs   PlatformAuditWriter
}

func NewDistributorService(
	dists *repository.DistributorRepo,
	comm *repository.CommissionRepo,
	orders *repository.OrderRepo,
	kgs *repository.KindergartenRepo,
	logs PlatformAuditWriter,
) *DistributorService {
	return &DistributorService{dists: dists, comm: comm, orders: orders, kgs: kgs, logs: logs}
}

type DistributorRequest struct {
	Name           string `json:"name" binding:"required,min=1,max=64"`
	Contact        string `json:"contact"`
	Phone          string `json:"phone"`
	CommissionRate *int   `json:"commission_rate"`
	Status         *int8  `json:"status"`
}

func (s *DistributorService) ListDistributors(ctx context.Context, status int8) ([]domain.Distributor, int64, error) {
	return s.dists.List(ctx, status)
}

func (s *DistributorService) CreateDistributor(ctx context.Context, req DistributorRequest, operator, ip string) (*domain.Distributor, error) {
	rate := 0
	if req.CommissionRate != nil {
		rate = *req.CommissionRate
	}
	if rate < 0 || rate > 100 {
		return nil, fmt.Errorf("INVALID_RATE")
	}
	d := &domain.Distributor{Name: req.Name, Contact: req.Contact, Phone: req.Phone, CommissionRate: rate, Status: 1}
	if err := s.dists.Create(ctx, d); err != nil {
		return nil, err
	}
	// 分销商名册变更属 best-effort：审计故障只记日志，不阻断名册维护。
	writeAuditBestEffort(ctx, s.logs, &domain.PlatformOperationLog{
		Operator: operator, Module: "distributor", Action: "create", TargetType: "distributor", TargetID: d.ID,
		Detail: d.Name, IP: ip,
	})
	return d, nil
}

// UpdateDistributor 修改分销商。佣金比例只影响之后确认的订单，不回算历史。
func (s *DistributorService) UpdateDistributor(ctx context.Context, id uint64, req DistributorRequest, operator, ip string) error {
	d, err := s.dists.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if d == nil {
		return fmt.Errorf("DISTRIBUTOR_NOT_FOUND")
	}
	cols := map[string]any{"name": req.Name, "contact": req.Contact, "phone": req.Phone}
	if req.CommissionRate != nil {
		if *req.CommissionRate < 0 || *req.CommissionRate > 100 {
			return fmt.Errorf("INVALID_RATE")
		}
		cols["commission_rate"] = *req.CommissionRate
	}
	if req.Status != nil {
		cols["status"] = *req.Status
	}
	if err := s.dists.Update(ctx, id, cols); err != nil {
		return err
	}
	writeAuditBestEffort(ctx, s.logs, &domain.PlatformOperationLog{
		Operator: operator, Module: "distributor", Action: "update", TargetType: "distributor", TargetID: id,
		Detail: req.Name, IP: ip,
	})
	return nil
}

type CommissionRow struct {
	ID                  uint64                  `json:"id"`
	OrderNo             string                  `json:"order_no"`
	KindergartenName    string                  `json:"kindergarten_name"`
	DistributorName     string                  `json:"distributor_name"`
	PaymentID           uint64                  `json:"payment_id"`
	BaseAmountCents     int64                   `json:"base_amount_cents"`
	Rate                int                     `json:"rate"`
	AmountCents         int64                   `json:"amount_cents"`
	ReversedAmountCents int64                   `json:"reversed_amount_cents"`
	NetAmountCents      int64                   `json:"net_amount_cents"`
	Status              string                  `json:"status"`
	ReceivedAt          time.Time               `json:"received_at"`
	EligibleAt          time.Time               `json:"eligible_at"`
	ReversalOf          *uint64                 `json:"reversal_of"`
	Reversals           []CommissionReversalRow `json:"reversals,omitempty"`
}

// CommissionReversalRow 冲回行的审计视图，挂在原始佣金明细下返回。
type CommissionReversalRow struct {
	ID              uint64    `json:"id"`
	PaymentID       uint64    `json:"payment_id"`
	BaseAmountCents int64     `json:"base_amount_cents"`
	AmountCents     int64     `json:"amount_cents"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
}

func (s *DistributorService) ListCommissions(ctx context.Context, f repository.CommissionFilter) ([]CommissionRow, int64, map[string]int64, error) {
	// 惰性结算：读取前把已满 30 天的待结算佣金推进为可结算。
	if err := s.comm.PromoteEligible(ctx, time.Now()); err != nil {
		return nil, 0, nil, err
	}
	rows, total, err := s.comm.List(ctx, f)
	if err != nil {
		return nil, 0, nil, err
	}
	summary, err := s.comm.Summary(ctx, f)
	if err != nil {
		return nil, 0, nil, err
	}
	out, err := s.enrichCommissions(ctx, rows)
	if err != nil {
		return nil, 0, nil, err
	}
	return out, total, summary, nil
}

func (s *DistributorService) GetCommission(ctx context.Context, id uint64) (*CommissionRow, error) {
	if err := s.comm.PromoteEligible(ctx, time.Now()); err != nil {
		return nil, err
	}
	row, err := s.comm.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, fmt.Errorf("COMMISSION_NOT_FOUND")
	}
	rows, err := s.enrichCommissions(ctx, []domain.CommissionLedger{*row})
	if err != nil {
		return nil, err
	}
	// 原始佣金明细附带冲回记录，审计可追溯到被退的那笔收款。
	if row.ReversalOf == nil {
		reversals, err := s.comm.ListReversals(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		for _, r := range reversals {
			rows[0].Reversals = append(rows[0].Reversals, CommissionReversalRow{
				ID: r.ID, PaymentID: r.PaymentID, BaseAmountCents: r.BaseAmountCents,
				AmountCents: r.AmountCents, Status: r.Status, CreatedAt: r.CreatedAt,
			})
		}
	}
	return &rows[0], nil
}

func (s *DistributorService) enrichCommissions(ctx context.Context, rows []domain.CommissionLedger) ([]CommissionRow, error) {
	if len(rows) == 0 {
		return []CommissionRow{}, nil
	}
	orderIDs := make([]uint64, 0, len(rows))
	distIDs := make([]uint64, 0, len(rows))
	kgIDs := make([]uint64, 0, len(rows))
	for _, r := range rows {
		orderIDs = append(orderIDs, r.OrderID)
		distIDs = append(distIDs, r.DistributorID)
		kgIDs = append(kgIDs, r.KindergartenID)
	}
	orderMap := make(map[uint64]domain.Order, len(orderIDs))
	for _, id := range uniqueIDs(orderIDs) {
		if o, err := s.orders.FindByID(ctx, id); err == nil && o != nil {
			orderMap[id] = *o
		}
	}
	distMap := make(map[uint64]string, len(distIDs))
	if dists, err := s.dists.FindByIDs(ctx, uniqueIDs(distIDs)); err == nil {
		for _, d := range dists {
			distMap[d.ID] = d.Name
		}
	}
	kgMap := make(map[uint64]string, len(kgIDs))
	if kgs, err := s.kgs.FindByIDs(ctx, uniqueIDs(kgIDs)); err == nil {
		for _, k := range kgs {
			kgMap[k.ID] = k.Name
		}
	}
	out := make([]CommissionRow, 0, len(rows))
	for _, r := range rows {
		row := CommissionRow{
			ID: r.ID, PaymentID: r.PaymentID, BaseAmountCents: r.BaseAmountCents,
			Rate: r.Rate, AmountCents: r.AmountCents,
			ReversedAmountCents: r.ReversedAmountCents,
			NetAmountCents:      r.AmountCents - r.ReversedAmountCents,
			Status:              r.Status,
			EligibleAt:          r.EligibleAt, ReversalOf: r.ReversalOf,
			DistributorName: distMap[r.DistributorID], KindergartenName: kgMap[r.KindergartenID],
		}
		if o, ok := orderMap[r.OrderID]; ok {
			row.OrderNo = o.OrderNo
		}
		row.ReceivedAt = r.CreatedAt
		out = append(out, row)
	}
	return out, nil
}

func uniqueIDs(ids []uint64) []uint64 {
	seen := make(map[uint64]struct{}, len(ids))
	out := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
