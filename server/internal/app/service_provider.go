package app

import (
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/config"
	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/repository"
	"kindergarten-server/internal/service"
)

func ProvideAuthService(
	userRepo *repository.UserRepo,
	userRoleRepo *repository.UserRoleRepo,
	wx service.WeChatAPI,
	clientSvc *service.ClientService,
	identityRepo *repository.UserIdentityRepo,
	adminAuthz *service.AdminAuthzService,
	kgStatus *service.KindergartenStatusService,
	kgRepo *repository.KindergartenRepo,
	rdb *redis.Client,
	secret JWTSecret,
	atTTL AccessTTL,
	rtTTL RefreshTTL,
	consentRepo *repository.ConsentRepo,
	cfg *config.Config,
	mergeSvc *service.IdentityMergeService,
) *service.AuthService {
	return service.NewAuthService(
		userRepo,
		userRoleRepo,
		wx,
		clientSvc,
		identityRepo,
		adminAuthz,
		kgStatus,
		kgRepo,
		rdb,
		[]byte(secret),
		time.Duration(atTTL),
		time.Duration(rtTTL),
		consentRepo,
		cfg.Consent.Enforce,
		kgRepo, // 成员园所与绑定目标解析共用同一仓储（#169 / #167）
		mergeSvc,
		// 绑定限流的完整维度（手机号）只能在 service 换出手机号后判定，
		// 中间件那层先按 app_id + IP 兜住洪峰（#174 / #167）。
		middleware.NewBindPhoneLimiter(rdb, middleware.BindPhoneMaxPerMinute, time.Minute),
	)
}

// ProvideTenantEpochStore 暴露园所账号级失效存储（#182）。AuthJWT 与
// AdminService（重置/删除管理员）共用同一份存储；Wire 不会把具体类型隐式
// 匹配到接口，必须显式提供。
func ProvideTenantEpochStore(authSvc *service.AuthService) auth.TenantEpochStore {
	if authSvc == nil {
		return nil
	}
	return authSvc.TenantEpochStore()
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
	entrySvc *service.KindergartenEntryService,
	rdb *redis.Client,
	authSvc *service.AuthService,
	secret JWTSecret,
	atTTL AccessTTL,
	rtTTL RefreshTTL,
) *service.PlatformService {
	// 平台访问凭证使用与园所相同的 jwt.access_ttl_min（默认 15 分钟），
	// 续期凭证使用 jwt.refresh_ttl_day（默认 7 天）。此前硬编码 24 小时的
	// 行为已移除（design D5，显式行为变更）。
	var revoker auth.TokenRevoker
	if authSvc != nil {
		revoker = authSvc.TokenRevoker()
	}
	return service.NewPlatformService(
		db, adminRepo, kgRepo, entRepo, orderRepo, userRepo, userRoleRepo,
		adminRoleRepo, paymentRepo, commRepo, distRepo, logRepo, dietSeed, kgStatus, entrySvc,
		auth.NewPlatformEpochStore(rdb, auth.PlatformEpochTTLFor(time.Duration(rtTTL), time.Duration(atTTL))),
		revoker, rdb, []byte(secret),
		time.Duration(atTTL), time.Duration(rtTTL),
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
