package app

import (
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/config"
	"kindergarten-server/internal/repository"
	"kindergarten-server/internal/service"
)

func ProvideAuthService(
	db *gorm.DB,
	userRepo *repository.UserRepo,
	userRoleRepo *repository.UserRoleRepo,
	invitationRepo *repository.TeacherInvitationRepo,
	adminAuthz *service.AdminAuthzService,
	kgStatus *service.KindergartenStatusService,
	rdb *redis.Client,
	wx *auth.WeChatClient,
	secret JWTSecret,
	atTTL AccessTTL,
	rtTTL RefreshTTL,
	consentRepo *repository.ConsentRepo,
	cfg *config.Config,
) *service.AuthService {
	return service.NewAuthService(
		db,
		userRepo,
		userRoleRepo,
		invitationRepo,
		adminAuthz,
		kgStatus,
		rdb,
		wx,
		[]byte(secret),
		time.Duration(atTTL),
		time.Duration(rtTTL),
		consentRepo,
		cfg.Consent.Enforce,
	)
}

// ProvidePlatformService 平台方服务：账号与园所运营独立于园所信任域（ADR-0014）。
func ProvidePlatformService(
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
	logRepo *repository.PlatformLogRepo,
	dietSeed *service.DietSeedService,
	kgStatus *service.KindergartenStatusService,
	rdb *redis.Client,
	secret JWTSecret,
	atTTL AccessTTL,
) *service.PlatformService {
	// 平台方没有 refresh token 流程，凭证直接给 24 小时；园所的 access TTL
	// （15 分钟）只适用于有 refresh 的园所会话。
	_ = atTTL
	return service.NewPlatformService(
		db, adminRepo, kgRepo, entRepo, orderRepo, userRepo, userRoleRepo,
		adminRoleRepo, paymentRepo, commRepo, distRepo, logRepo, dietSeed, kgStatus,
		auth.NewPlatformEpochStore(rdb), []byte(secret), 24*time.Hour,
	)
}

// BillingEnforceModules 授权强制开关。关闭时 ModuleGate 只记录不拦截（design D11）。
type BillingEnforceModules bool

func ProvideBillingEnforceModules(*config.Config) BillingEnforceModules {
	v := strings.ToLower(os.Getenv("BILLING_ENFORCE_MODULES"))
	return BillingEnforceModules(v == "1" || v == "true" || v == "yes")
}

func ProvideEntitlementService(
	ents *repository.EntitlementRepo,
	orders *repository.OrderRepo,
	kgs *repository.KindergartenRepo,
	cache *service.EntitlementCache,
	enforce BillingEnforceModules,
) *service.EntitlementService {
	return service.NewEntitlementService(ents, orders, kgs, cache, bool(enforce))
}
