package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	mysqldrv "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"kindergarten-server/internal/ai/face"
	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/config"
	handler "kindergarten-server/internal/handler/http"
	"kindergarten-server/internal/service"
	ossx "kindergarten-server/internal/storage/oss"
	"kindergarten-server/pkg/response"
)

// dsnLocIssue 检查 DSN 的 loc 是否与进程时区同源（配置正确时返回空串）。
// 业务日期依赖两者同源：MYSQL_DSN 环境变量会整体替换配置里的 DSN，
// 漏写 loc 时驱动默认 UTC，写 DATE 列会整体差一天且没有任何报错。
func dsnLocIssue(dsn string) string {
	dsnCfg, err := mysqldrv.ParseDSN(dsn)
	if err != nil {
		return "MySQL DSN 解析失败，无法校验 loc: " + err.Error()
	}
	// 比行为、不比名字：tzdata 里 PRC / Asia/Chongqing / Asia/Harbin 都是 +08 的链接名，
	// 按名字比较会把完全正确的配置判成「DATE 可能差一天」——那条 warn 一旦开始说谎，
	// 真正漏写 loc 的那次也会被当成噪音忽略掉。
	// 取冬夏各一个时刻：只看一个时刻，UTC 与 Europe/London 在冬季会相等而漏报。
	for _, probe := range []time.Time{
		time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC),
		time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC),
	} {
		_, procOff := probe.In(time.Local).Zone()
		_, dsnOff := probe.In(dsnCfg.Loc).Zone()
		if procOff != dsnOff {
			return "MySQL DSN 的 loc(" + dsnCfg.Loc.String() + ") 与进程时区(" + time.Local.String() +
				") 偏移不一致（未写 loc 时驱动默认 UTC），DATE 列可能差一天"
		}
	}
	return ""
}

func ProvideGorm(cfg *config.Config) (*gorm.DB, error) {
	if issue := dsnLocIssue(cfg.MySQL.DSN); issue != "" {
		slog.Warn(issue)
	}

	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// schema 由 goose 迁移管理；应用启动不建表、不播种。
	// 新环境初始化顺序：make migrate -> make seedbase。
	return db, nil
}

func ProvideRedis(cfg *config.Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
}

// ProvideWeChatClient 注入两端各自的微信凭据。
// 返回类型保持 *auth.WeChatClient 单值，避免改动 Wire 的 provider 形状。
func ProvideWeChatClient(cfg *config.Config, rdb *redis.Client) *auth.WeChatClient {
	return auth.NewWeChatClient(
		cfg.WeChat.TeacherAppID, cfg.WeChat.TeacherSecret,
		cfg.WeChat.ParentAppID, cfg.WeChat.ParentSecret,
		cfg.WeChat.CodeEnvVersion,
		rdb,
	)
}

// TrustedProxies gin SetTrustedProxies 的来源列表，由 [server] trusted_proxies 提供。
type TrustedProxies []string

func ProvideTrustedProxies(cfg *config.Config) TrustedProxies {
	return TrustedProxies(cfg.Server.TrustedProxyList())
}

type JWTSecret []byte

func ProvideJWTSecret(cfg *config.Config) JWTSecret {
	return JWTSecret(cfg.JWT.Secret)
}

type AccessTTL time.Duration

func ProvideAccessTTL(cfg *config.Config) AccessTTL {
	ttl := cfg.JWT.AccessTTLMin
	if ttl <= 0 {
		ttl = 15
	}
	return AccessTTL(time.Duration(ttl) * time.Minute)
}

type RefreshTTL time.Duration

func ProvideRefreshTTL(cfg *config.Config) RefreshTTL {
	ttl := cfg.JWT.RefreshTTLDay
	if ttl <= 0 {
		ttl = 7
	}
	return RefreshTTL(time.Duration(ttl) * 24 * time.Hour)
}

func ProvideCasbinConfig(cfg *config.Config) *config.CasbinConfig {
	return &cfg.Casbin
}

// ProvideFaceConfig 暴露 [face] 配置（建议分级阈值等读取端用途）。
func ProvideFaceConfig(cfg *config.Config) config.FaceConfig {
	return cfg.Face
}

// ProvideFaceProvider 人脸识别 provider（face.New 按 [face].provider 选 tencent/mock）。
// 作为服务层接口 face.Provider 的构造来源；Wire 需 Bind（见 wire.go）。
func ProvideFaceProvider(cfg *config.Config) (face.Provider, error) {
	return face.New(cfg.Face)
}

func ProvideOSSStorage(cfg *config.Config) (*ossx.Storage, error) {
	storage, err := ossx.NewStorage(ossx.Config{
		Endpoint:        cfg.OSS.Endpoint,
		AccessKeyID:     cfg.OSS.AccessKeyID,
		AccessKeySecret: cfg.OSS.AccessKeySecret,
		BucketName:      cfg.OSS.BucketName,
	})
	if err != nil {
		return nil, err
	}
	// 媒体 URL 在响应序列化前统一签名；OSS 未配置时保持引用原样（本地开发）。
	if storage.Configured() {
		response.SetMediaSigner(func(ref string, thumb bool) string {
			signed, signErr := storage.SignRef(ref, time.Hour, thumb)
			if signErr != nil {
				slog.Warn("sign media url failed", "err", signErr)
				return ref
			}
			return signed
		})
	}
	return storage, nil
}

// ProvideNoticeScheduler 定时通知发布 worker（openspec: notice-scheduled-publish）。
// 只构造不启动：启动挂在 AppHandler 装配处，见 NewAppHandlerWithBackgroundWorkers。
func ProvideNoticeScheduler(db *gorm.DB) *service.NoticeScheduler {
	return service.NewNoticeSchedulerForApp(db)
}

// NewAppHandlerWithBackgroundWorkers 装配 AppHandler 时启动进程内后台 worker。
// InjectEngine 的返回类型固定为 *gin.Engine，后台任务没有别的可挂点；做成显式依赖
// （而不是手工往 wire_gen.go 里加一行）可以保证 make wire 重新生成后启动仍然存在。
func NewAppHandlerWithBackgroundWorkers(h *handler.Handler, scheduler *service.NoticeScheduler, clients *service.ClientService, cfg *config.Config, db *gorm.DB, rdb *redis.Client) (*AppHandler, error) {
	// 客户端注册表先于后台 worker 同步：失败即拒绝启动，避免带病服务登录请求。
	if err := syncClientRegistry(clients, cfg); err != nil {
		return nil, err
	}
	scheduler.Start()
	return NewAppHandler(h).
		WithCORS(cfg.App.AllowedCORSOrigins()).
		WithInfra(db, rdb), nil
}

// syncClientRegistry 把配置里的两端 AppID 幂等同步到 clients 表（#163）。
// 失败阻断启动：配置与启用中的客户端不一致时，登录会出现「有入口无凭据」
// 或「同角色两条 enabled 无法确定用哪条」。
func syncClientRegistry(clients *service.ClientService, cfg *config.Config) error {
	specs := service.SharedClientSpecs(
		cfg.WeChat.TeacherAppID, cfg.WeChat.ParentAppID,
		cfg.Clients.TeacherReleaseVersion, cfg.Clients.ParentReleaseVersion,
	)
	result, err := clients.SyncFromConfig(context.Background(), specs)
	if err != nil {
		return fmt.Errorf("sync client registry: %w", err)
	}
	slog.Info("client registry synced",
		"created", result.Created,
		"disabled", result.Disabled,
		"unchanged", result.Unchanged,
	)
	for _, warning := range result.Warnings {
		slog.Warn("client registry warning", "detail", warning)
	}
	return nil
}
