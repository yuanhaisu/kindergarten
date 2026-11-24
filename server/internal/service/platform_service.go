package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// ErrPhoneExists 平台开通园所时手机号已被占用。
//
// Deprecated: 开通不再拒绝既有家长/老师账号（#171 改为复用）。保留常量是为了
// 兼容仍然映射它的调用方；新代码使用 ErrPhoneAlreadyAdmin。
var ErrPhoneExists = errors.New("PHONE_EXISTS")

// ErrPhoneAlreadyAdmin 手机号已在任一园所担任管理员。v1 约束「一手机号一 admin」，
// 开通与 setadmin 都不得为同一手机号造出第二个园所管理员。
var ErrPhoneAlreadyAdmin = errors.New("PHONE_ALREADY_ADMIN")

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
	entry         *KindergartenEntryService
	epochs        auth.PlatformEpochStore
	// tenantEpochs 是园所账号（园长）的会话失效存储。开通复用既有账号会覆盖
	// password_hash，等同凭据重置，必须同步推进它（#196）。
	tenantEpochs auth.TenantEpochStore
	revoker      auth.TokenRevoker
	rdb          *redis.Client
	secret       []byte
	atTTL        time.Duration
	rtTTL        time.Duration
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
	entry *KindergartenEntryService,
	epochs auth.PlatformEpochStore,
	revoker auth.TokenRevoker,
	rdb *redis.Client,
	secret []byte,
	atTTL time.Duration,
	rtTTL time.Duration,
) *PlatformService {
	return &PlatformService{
		db: db, adminRepo: adminRepo, kgRepo: kgRepo, entRepo: entRepo, orderRepo: orderRepo,
		userRepo: userRepo, userRoleRepo: userRoleRepo, adminRoleRepo: adminRoleRepo,
		paymentRepo: paymentRepo, commRepo: commRepo, distRepo: distRepo,
		logRepo: logRepo, dietSeed: dietSeed, kgStatus: kgStatus, entry: entry, epochs: epochs,
		tenantEpochs: auth.NewTenantEpochStore(rdb, auth.PlatformEpochTTLFor(rtTTL, atTTL)),
		revoker:      revoker, rdb: rdb, secret: secret, atTTL: atTTL, rtTTL: rtTTL,
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
	Token        string            `json:"token"`
	RefreshToken string            `json:"refresh_token"`
	ExpiresIn    int64             `json:"expires_in"`
	Admin        PlatformAdminInfo `json:"admin"`
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
	return s.issuePlatformSession(ctx, admin)
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
	// PublicCode 园所公开入口码；公开信息，不含内部敏感字段。
	PublicCode string `json:"public_code"`
}

func (s *PlatformService) ListKindergartens(ctx context.Context, page, pageSize int, keyword, status string) ([]PlatformKindergartenRow, int64, error) {
	kgs, total, err := s.kgRepo.List(ctx, page, pageSize, keyword, status)
	if err != nil {
		return nil, 0, err
	}
	now := time.Now()
	kgIDs := make([]uint64, 0, len(kgs))
	for _, kg := range kgs {
		kgIDs = append(kgIDs, kg.ID)
	}
	soonest, err := s.soonestExpiryByKindergarten(ctx, kgIDs, now)
	if err != nil {
		return nil, 0, err
	}
	out := make([]PlatformKindergartenRow, 0, len(kgs))
	for _, kg := range kgs {
		row := PlatformKindergartenRow{
			ID: kg.ID, Name: kg.Name, Principal: kg.Principal,
			Status: kg.Status, CreatedAt: kg.CreatedAt, PublicCode: kg.PublicCode,
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

// soonestExpiryByKindergarten 取本页园所每园最早的未到期授权日期。
// soonest_expire_at 与看板的「N 天内到期」共用 ListActiveExpiring，避免两套口径
// （design D7）。查询失败必须返回错误：静默降级会让到期园所显示成「未设到期」。
func (s *PlatformService) soonestExpiryByKindergarten(ctx context.Context, kgIDs []uint64, now time.Time) (map[uint64]time.Time, error) {
	soonest := make(map[uint64]time.Time, len(kgIDs))
	if len(kgIDs) == 0 {
		return soonest, nil
	}
	ents, err := s.entRepo.ListActiveExpiring(ctx, repository.ActiveExpiryFilter{
		KindergartenIDs: kgIDs, Now: now,
	})
	if err != nil {
		return nil, fmt.Errorf("list active entitlements for kindergartens: %w", err)
	}
	for _, e := range ents {
		if t, ok := soonest[e.KindergartenID]; !ok || e.ExpiresAt.Before(t) {
			soonest[e.KindergartenID] = e.ExpiresAt
		}
	}
	return soonest, nil
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
	PublicCode   string                   `json:"public_code"`
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
		StudentCount: count, PublicCode: kg.PublicCode,
		Entitlements: ents, RecentOrders: briefs,
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
	// PublicCode 园所公开入口码（#177），平台控制台据此下发家长端入口。
	PublicCode string `json:"public_code"`
}

// Onboard 开通园所：创建园所、园长账号、超级管理员角色与试用授权。
//
// 园长账号按手机号解析（FindOrCreateByPhone）：既有家长/老师账号直接复用，
// 只补 admin 角色与管理端口令，昵称/头像/性别与各端档案保持原样。
// 复用会覆盖口令，因此旧 access/refresh 会立即失效（园所账号 epoch，见 #196）。
// 已在任一园所担任 admin 的手机号会被拒绝（一手机号一 admin，v1）。
func (s *PlatformService) Onboard(ctx context.Context, req OnboardRequest, operator, ip string) (*OnboardResult, error) {
	if err := validatePhone(req.Phone); err != nil {
		return nil, fmt.Errorf("INVALID_PHONE: %w", err)
	}
	if err := validatePassword(req.AdminPassword); err != nil {
		return nil, fmt.Errorf("WEAK_PASSWORD: %w", err)
	}

	hash, err := HashPassword(req.AdminPassword)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	trialEnd := now.AddDate(0, 0, trialDays)
	var (
		kgID       uint64
		publicCode string
	)
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		userRepo := s.userRepo.WithDB(tx)
		user, created, err := userRepo.FindOrCreateByPhone(ctx, req.Phone, repository.UserProfileDefaults{
			Nickname:     req.Principal,
			PasswordHash: hash,
		})
		if err != nil {
			return err
		}

		// 同一手机号的开通必须串行：先锁账号行，再用当前读检查 admin 身份。
		// 否则两个并发开通都看不到对方未提交的角色，各自建出一个管理员。
		if err := lockUserForOnboard(ctx, tx, user.ID); err != nil {
			return err
		}
		alreadyAdmin, err := hasAdminRoleForUpdate(ctx, tx, user.ID)
		if err != nil {
			return err
		}
		if alreadyAdmin {
			return ErrPhoneAlreadyAdmin
		}

		// 复用路径只重置管理端口令并打首登改密标记；其余资料字段一个都不碰。
		// 覆盖既有口令等同凭据重置（#196）：必须先推进园所账号 epoch，使改密前
		// 签发的 access/refresh 立即失效。Redis 写入不参与 DB 事务回滚——若后续
		// 步骤失败，该账号会被登出，这是失败方向正确的代价（fail closed）。
		fields := map[string]any{"must_change_password": true}
		if !created {
			if s.tenantEpochs == nil {
				return fmt.Errorf("%w: 园所账号失效存储未配置", ErrAuthStorageUnavailable)
			}
			if err := s.tenantEpochs.BumpTenantEpoch(ctx, fmt.Sprintf("%d", user.ID), time.Now()); err != nil {
				return fmt.Errorf("%w: %v", ErrAuthStorageUnavailable, err)
			}
			fields["password_hash"] = hash
		}
		if err := userRepo.UpdateFields(ctx, user.ID, fields); err != nil {
			return err
		}

		kg := &domain.Kindergarten{Name: req.Name, Principal: req.Principal, Address: req.Address, Status: domain.KindergartenStatusActive}
		if err := tx.Create(kg).Error; err != nil {
			return err
		}
		kgID = kg.ID
		// 公开码与园所同事务：开出来的园所必须立刻可被家长端入口解析（#177）。
		publicCode, err = s.assignPublicCodeTx(ctx, tx, kg.ID)
		if err != nil {
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
		detail := fmt.Sprintf("开通园所：%s", req.Name)
		if !created {
			detail += "（复用既有账号，已重置口令并使其旧会话失效）"
		}
		return s.logRepo.CreateTx(ctx, tx, &domain.PlatformOperationLog{
			Operator: operator, Module: "kindergarten", Action: "onboard", TargetType: "kindergarten",
			TargetID: kg.ID, KindergartenID: kg.ID, Detail: detail, IP: ip,
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
		PublicCode: publicCode,
	}, nil
}

// assignPublicCodeTx 为新园分配公开入口码。
//
// GeneratePublicCode 已规避当前已占用的候选；这里再兜住「检查后、写入前」被并发
// 插入占用的窗口：撞唯一键就换码重试，绝不能把园所留在无公开码的状态。
func (s *PlatformService) assignPublicCodeTx(ctx context.Context, tx *gorm.DB, kindergartenID uint64) (string, error) {
	if s.entry == nil {
		return "", errors.New("园所入口服务未配置，无法生成公开码")
	}
	kgRepo := repository.NewKindergartenRepo(tx)
	for attempt := 1; attempt <= publicCodeMaxAttempts; attempt++ {
		code, err := s.entry.GeneratePublicCode(ctx)
		if err != nil {
			return "", err
		}
		wrote, err := kgRepo.SetPublicCodeIfEmpty(ctx, kindergartenID, code)
		if errors.Is(err, repository.ErrDuplicatePublicCode) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("写入园所公开码失败: %w", err)
		}
		if !wrote {
			return "", errors.New("写入园所公开码失败：园所已有公开码")
		}
		return code, nil
	}
	return "", fmt.Errorf("生成园所公开码失败：连续 %d 次候选码写入冲突", publicCodeMaxAttempts)
}

// lockUserForOnboard 用 SELECT ... FOR UPDATE 锁住账号行，把同一手机号的并发开通串行化。
func lockUserForOnboard(ctx context.Context, tx *gorm.DB, userID uint64) error {
	var u domain.User
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("id").First(&u, userID).Error; err != nil {
		return fmt.Errorf("锁定开通账号失败: %w", err)
	}
	return nil
}

// hasAdminRoleForUpdate 判断账号是否已在任一园所担任管理员。
//
// 必须用锁定读（当前读）：REPEATABLE READ 下普通 SELECT 走事务开始时的快照，
// 会漏掉并发事务刚提交的 admin 角色。UserAdminRole 一并检查，避免只残留
// 角色关联（鉴权按关联给通配权限）却被当成「不是管理员」。
func hasAdminRoleForUpdate(ctx context.Context, tx *gorm.DB, userID uint64) (bool, error) {
	var roles []domain.UserRole
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("id").Where("user_id = ? AND role = ?", userID, "admin").
		Limit(1).Find(&roles).Error; err != nil {
		return false, fmt.Errorf("检查管理员身份失败: %w", err)
	}
	if len(roles) > 0 {
		return true, nil
	}
	var links []domain.UserAdminRole
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("id").Where("user_id = ?", userID).
		Limit(1).Find(&links).Error; err != nil {
		return false, fmt.Errorf("检查管理员角色关联失败: %w", err)
	}
	return len(links) > 0, nil
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
