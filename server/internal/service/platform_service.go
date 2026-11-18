package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// ErrPhoneExists 平台开通园所时手机号已被占用。
var ErrPhoneExists = errors.New("PHONE_EXISTS")

// PlatformService 平台方运营：登录、园所开通与状态、操作留痕。
// 平台账号与园所账号完全独立（ADR-0014）。
type PlatformService struct {
	db            *gorm.DB
	adminRepo     *repository.PlatformAdminRepo
	kgRepo        *repository.KindergartenRepo
	entRepo       *repository.EntitlementRepo
	orderRepo     *repository.OrderRepo
	userRepo      *repository.UserRepo
	userRoleRepo  *repository.UserRoleRepo
	adminRoleRepo *repository.AdminRepo
	paymentRepo   *repository.PaymentRepo
	commRepo      *repository.CommissionRepo
	distRepo      *repository.DistributorRepo
	logRepo       PlatformAuditStore
	dietSeed      *DietSeedService
	kgStatus      *KindergartenStatusService
	epochs        auth.PlatformEpochStore
	secret        []byte
	atTTL         time.Duration
}

func NewPlatformService(
	db *gorm.DB,
	adminRepo *repository.PlatformAdminRepo,
	kgRepo *repository.KindergartenRepo,
	entRepo *repository.EntitlementRepo,
	orderRepo *repository.OrderRepo,
	userRepo *repository.UserRepo,
	userRoleRepo *repository.UserRoleRepo,
	adminRoleRepo *repository.AdminRepo,
	paymentRepo *repository.PaymentRepo,
	commRepo *repository.CommissionRepo,
	distRepo *repository.DistributorRepo,
	logRepo PlatformAuditStore,
	dietSeed *DietSeedService,
	kgStatus *KindergartenStatusService,
	epochs auth.PlatformEpochStore,
	secret []byte,
	atTTL time.Duration,
) *PlatformService {
	return &PlatformService{
		db: db, adminRepo: adminRepo, kgRepo: kgRepo, entRepo: entRepo, orderRepo: orderRepo,
		userRepo: userRepo, userRoleRepo: userRoleRepo, adminRoleRepo: adminRoleRepo,
		paymentRepo: paymentRepo, commRepo: commRepo, distRepo: distRepo,
		logRepo: logRepo, dietSeed: dietSeed, kgStatus: kgStatus, epochs: epochs,
		secret: secret, atTTL: atTTL,
	}
}

// EpochStore 暴露平台账号的凭证失效存储给路由装配（PlatformAuth）。
func (s *PlatformService) EpochStore() auth.PlatformEpochStore { return s.epochs }

type PlatformAdminInfo struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

type PlatformLoginResult struct {
	Token     string            `json:"token"`
	ExpiresIn int64             `json:"expires_in"`
	Admin     PlatformAdminInfo `json:"admin"`
}

func (s *PlatformService) Login(ctx context.Context, username, password string) (*PlatformLoginResult, error) {
	admin, err := s.adminRepo.FindByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if admin == nil || admin.Status != 1 {
		return nil, fmt.Errorf("INVALID_CREDENTIALS")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("INVALID_CREDENTIALS")
	}
	token, err := auth.SignPlatformAccess(s.secret, fmt.Sprintf("%d", admin.ID), s.atTTL)
	if err != nil {
		return nil, err
	}
	return &PlatformLoginResult{
		Token:     token,
		ExpiresIn: int64(s.atTTL.Seconds()),
		Admin:     PlatformAdminInfo{ID: admin.ID, Username: admin.Username, Name: admin.Name},
	}, nil
}

// PlatformKindergartenRow 园所列表行。
type PlatformKindergartenRow struct {
	ID              uint64     `json:"id"`
	Name            string     `json:"name"`
	Principal       string     `json:"principal"`
	Status          string     `json:"status"`
	StudentCount    int64      `json:"student_count"`
	ActivePlanName  string     `json:"active_plan_name"`
	SoonestExpireAt *time.Time `json:"soonest_expire_at"`
	CreatedAt       time.Time  `json:"created_at"`
	// 超档提示：只展示，不阻断（ENT-0013）。
	TierMaxStudents int  `json:"tier_max_students"`
	OverLimit       bool `json:"over_limit"`
}

func (s *PlatformService) ListKindergartens(ctx context.Context, page, pageSize int, keyword, status string) ([]PlatformKindergartenRow, int64, error) {
	kgs, total, err := s.kgRepo.List(ctx, page, pageSize, keyword, status)
	if err != nil {
		return nil, 0, err
	}
	now := time.Now()
	// soonest_expire_at 与看板的「30 天内到期」共用 ListActiveExpiring，
	// 一次取回本页园所的在期授权，避免逐园查询与两套口径（design D7）。
	kgIDs := make([]uint64, 0, len(kgs))
	for _, kg := range kgs {
		kgIDs = append(kgIDs, kg.ID)
	}
	soonest := make(map[uint64]time.Time, len(kgIDs))
	if len(kgIDs) > 0 {
		if ents, err := s.entRepo.ListActiveExpiring(ctx, repository.ActiveExpiryFilter{
			KindergartenIDs: kgIDs, Now: now,
		}); err == nil {
			for _, e := range ents {
				if t, ok := soonest[e.KindergartenID]; !ok || e.ExpiresAt.Before(t) {
					soonest[e.KindergartenID] = e.ExpiresAt
				}
			}
		}
	}
	out := make([]PlatformKindergartenRow, 0, len(kgs))
	for _, kg := range kgs {
		row := PlatformKindergartenRow{
			ID: kg.ID, Name: kg.Name, Principal: kg.Principal,
			Status: kg.Status, CreatedAt: kg.CreatedAt,
		}
		if row.Status == "" {
			row.Status = domain.KindergartenStatusActive
		}
		if n, err := s.kgRepo.CountStudents(ctx, kg.ID); err == nil {
			row.StudentCount = n
		}
		if t, ok := soonest[kg.ID]; ok {
			t := t
			row.SoonestExpireAt = &t
		}
		if o, err := s.orderRepo.FindLatestActiveByKindergarten(ctx, kg.ID); err == nil && o != nil {
			row.ActivePlanName = snapshotPlanName(o.PlanSnapshot)
			var snap PlanSnapshot
			if err := json.Unmarshal([]byte(o.PlanSnapshot), &snap); err == nil && snap.TierMaxStudents > 0 {
				row.TierMaxStudents = snap.TierMaxStudents
				row.OverLimit = row.StudentCount > int64(snap.TierMaxStudents)
			}
		}
		out = append(out, row)
	}
	return out, total, nil
}

type PlatformEntitlementRow struct {
	ModuleCode string    `json:"module_code"`
	ModuleName string    `json:"module_name"`
	Source     string    `json:"source"`
	StartsAt   time.Time `json:"starts_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	State      string    `json:"state"`
}

type PlatformOrderBrief struct {
	ID             uint64 `json:"id"`
	OrderNo        string `json:"order_no"`
	AmountDueCents int64  `json:"amount_due_cents"`
	Status         string `json:"status"`
}

type PlatformKindergartenDetail struct {
	ID           uint64                   `json:"id"`
	Name         string                   `json:"name"`
	Principal    string                   `json:"principal"`
	Status       string                   `json:"status"`
	StudentCount int64                    `json:"student_count"`
	Entitlements []PlatformEntitlementRow `json:"entitlements"`
	RecentOrders []PlatformOrderBrief     `json:"recent_orders"`
}

func (s *PlatformService) GetKindergarten(ctx context.Context, id uint64) (*PlatformKindergartenDetail, error) {
	kg, err := s.kgRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if kg == nil {
		return nil, fmt.Errorf("KINDERGARTEN_NOT_FOUND")
	}
	now := time.Now()
	rows, err := s.entRepo.ListByKindergarten(ctx, id)
	if err != nil {
		return nil, err
	}
	ents := make([]PlatformEntitlementRow, 0, len(rows))
	for _, e := range rows {
		ents = append(ents, PlatformEntitlementRow{
			ModuleCode: e.ModuleCode,
			ModuleName: ModuleName(e.ModuleCode),
			Source:     e.Source,
			StartsAt:   e.StartsAt,
			ExpiresAt:  e.ExpiresAt,
			State:      entitlementState(e.StartsAt, e.ExpiresAt, now, graceDuration),
		})
	}
	count, _ := s.kgRepo.CountStudents(ctx, id)
	orders, _, err := s.orderRepo.List(ctx, repository.OrderFilter{KindergartenID: id, Page: 1, PageSize: 5})
	if err != nil {
		return nil, err
	}
	briefs := make([]PlatformOrderBrief, 0, len(orders))
	for _, o := range orders {
		briefs = append(briefs, PlatformOrderBrief{
			ID: o.ID, OrderNo: o.OrderNo, AmountDueCents: o.AmountDueCents,
			// 与订单列表同口径：已收金额来自单次聚合，状态读时派生。
			Status: deriveOrderStatus(&o.Order, o.PaidCents, now),
		})
	}
	status := kg.Status
	if status == "" {
		status = domain.KindergartenStatusActive
	}
	return &PlatformKindergartenDetail{
		ID: kg.ID, Name: kg.Name, Principal: kg.Principal, Status: status,
		StudentCount: count, Entitlements: ents, RecentOrders: briefs,
	}, nil
}

type OnboardRequest struct {
	Name          string `json:"name" binding:"required,min=2,max=64"`
	Principal     string `json:"principal" binding:"required,min=1,max=32"`
	Phone         string `json:"phone" binding:"required"`
	Address       string `json:"address" binding:"max=256"`
	AdminPassword string `json:"admin_password" binding:"required,min=8"`
}

type OnboardResult struct {
	KindergartenID     uint64    `json:"kindergarten_id"`
	AdminPhone         string    `json:"admin_phone"`
	MustChangePassword bool      `json:"must_change_password"`
	TrialExpiresAt     time.Time `json:"trial_expires_at"`
}

// Onboard 开通园所：创建园所、园长账号、超级管理员角色与试用授权。
func (s *PlatformService) Onboard(ctx context.Context, req OnboardRequest, operator, ip string) (*OnboardResult, error) {
	if err := validatePhone(req.Phone); err != nil {
		return nil, fmt.Errorf("INVALID_PHONE: %w", err)
	}
	if err := validatePassword(req.AdminPassword); err != nil {
		return nil, fmt.Errorf("WEAK_PASSWORD: %w", err)
	}
	existing, err := s.userRepo.FindByPhone(ctx, req.Phone)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrPhoneExists
	}

	hash, err := HashPassword(req.AdminPassword)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	trialEnd := now.AddDate(0, 0, trialDays)
	var kgID uint64
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		kg := &domain.Kindergarten{Name: req.Name, Principal: req.Principal, Address: req.Address, Status: domain.KindergartenStatusActive}
		if err := tx.Create(kg).Error; err != nil {
			return err
		}
		kgID = kg.ID

		user := &domain.User{Phone: req.Phone, Nickname: req.Principal, PasswordHash: hash, MustChangePassword: true}
		if err := tx.Create(user).Error; err != nil {
			return err
		}
		if err := tx.Create(&domain.UserRole{
			UserID: user.ID, Role: "admin", KindergartenID: &kg.ID,
			KindergartenName: kg.Name, AdminRole: "super_admin",
		}).Error; err != nil {
			return err
		}
		role := &domain.AdminRole{KindergartenID: kg.ID, Name: "超级管理员", Description: "平台开通园所时创建", IsSystem: true}
		if err := tx.Create(role).Error; err != nil {
			return err
		}
		// 给超级管理员挂通配权限：否则 UserAdminRole 关联会命中「空权限集」分支
		// （admin_authz_service 的 super_admin 兜底只在无关联时生效）。
		perm, err := ensureWildcardPermission(tx)
		if err != nil {
			return err
		}
		if err := tx.Create(&domain.AdminRolePermission{RoleID: role.ID, PermissionID: perm.ID}).Error; err != nil {
			return err
		}
		if err := tx.Create(&domain.UserAdminRole{
			UserID: user.ID, AdminRoleID: role.ID, KindergartenID: kg.ID,
		}).Error; err != nil {
			return err
		}

		ents := []domain.Entitlement{{
			KindergartenID: kg.ID, ModuleCode: "base", Source: domain.EntitlementSourceGift,
			StartsAt:  time.Date(2000, 1, 1, 0, 0, 0, 0, time.Local),
			ExpiresAt: time.Date(2099, 12, 31, 23, 59, 59, 0, time.Local),
		}}
		for _, m := range PremiumModules() {
			ents = append(ents, domain.Entitlement{
				KindergartenID: kg.ID, ModuleCode: m.Code, Source: domain.EntitlementSourceTrial,
				StartsAt: now, ExpiresAt: trialEnd,
			})
		}
		if err := tx.Create(&ents).Error; err != nil {
			return err
		}
		// 开通园所属于强一致清单（design D6）：审计失败时园所、园长与授权一并回滚。
		return s.logRepo.CreateTx(ctx, tx, &domain.PlatformOperationLog{
			Operator: operator, Module: "kindergarten", Action: "onboard", TargetType: "kindergarten",
			TargetID: kg.ID, KindergartenID: kg.ID, Detail: fmt.Sprintf("开通园所：%s", req.Name), IP: ip,
		})
	}); err != nil {
		return nil, err
	}

	// 新园膳食数据在园所事务提交后补齐；失败只记日志，不回滚园所创建。
	if s.dietSeed != nil {
		if _, err := s.dietSeed.EnsureKindergartenSeeded(ctx, kgID); err != nil {
			writeAuditBestEffort(ctx, s.logRepo, &domain.PlatformOperationLog{
				Operator: operator, Module: "kindergarten", Action: "onboard", TargetType: "kindergarten",
				TargetID: kgID, KindergartenID: kgID,
				Detail: fmt.Sprintf("园所已创建，膳食数据初始化失败：%v", err), IP: ip,
			})
		}
	}

	return &OnboardResult{
		KindergartenID: kgID, AdminPhone: req.Phone,
		MustChangePassword: true, TrialExpiresAt: trialEnd,
	}, nil
}

func (s *PlatformService) SetKindergartenStatus(ctx context.Context, id uint64, status, operator, ip string) error {
	if status != domain.KindergartenStatusActive && status != domain.KindergartenStatusSuspended {
		return fmt.Errorf("INVALID_STATUS")
	}
	kg, err := s.kgRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if kg == nil {
		return fmt.Errorf("KINDERGARTEN_NOT_FOUND")
	}
	// 园所启停属于强一致清单：状态与审计同事务，审计失败时状态不变。
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.kgRepo.UpdateByIDTx(ctx, tx, id, map[string]any{"status": status}); err != nil {
			return err
		}
		return s.logRepo.CreateTx(ctx, tx, &domain.PlatformOperationLog{
			Operator: operator, Module: "kindergarten", Action: "update_status", TargetType: "kindergarten",
			TargetID: id, KindergartenID: id, Detail: fmt.Sprintf("%s -> %s", kg.Status, status), IP: ip,
		})
	}); err != nil {
		return err
	}
	// 立即失效状态缓存：停用后第一次请求就要被拦下，而不是等 60 秒 TTL。
	if s.kgStatus != nil {
		if err := s.kgStatus.Invalidate(ctx, id); err != nil {
			return fmt.Errorf("invalidate kindergarten status cache: %w", err)
		}
	}
	return nil
}

func (s *PlatformService) ListLogs(ctx context.Context, f repository.PlatformLogFilter) ([]domain.PlatformOperationLog, int64, error) {
	return s.logRepo.List(ctx, f)
}

func (s *PlatformService) logPlatform(ctx context.Context, operator, module, action, targetType string, targetID uint64, detail, ip string) error {
	return s.logRepo.Create(ctx, &domain.PlatformOperationLog{
		Operator: operator, Module: module, Action: action,
		TargetType: targetType, TargetID: targetID, Detail: detail, IP: ip,
	})
}

// AdminUsername 平台账号用户名，用于操作留痕展示。
func (s *PlatformService) AdminUsername(ctx context.Context, id uint64) string {
	a, err := s.adminRepo.FindByID(ctx, id)
	if err != nil || a == nil {
		return fmt.Sprintf("%d", id)
	}
	return a.Username
}

// Log 供其他平台服务（订单、套餐、分销）写入同一条操作留痕。
func (s *PlatformService) Log(ctx context.Context, operator, module, action, targetType string, targetID uint64, detail, ip string) error {
	return s.logPlatform(ctx, operator, module, action, targetType, targetID, detail, ip)
}

func ensureWildcardPermission(tx *gorm.DB) (*domain.AdminPermission, error) {
	var perm domain.AdminPermission
	err := tx.Where("code = ?", "*").First(&perm).Error
	if err == nil {
		return &perm, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	perm = domain.AdminPermission{Code: "*", Name: "全部权限", Module: "system", Description: "超级管理员"}
	if err := tx.Create(&perm).Error; err != nil {
		return nil, err
	}
	return &perm, nil
}

func snapshotPlanName(snapshot string) string {
	var v struct {
		PlanName string `json:"plan_name"`
	}
	if err := json.Unmarshal([]byte(snapshot), &v); err != nil {
		return ""
	}
	return v.PlanName
}
