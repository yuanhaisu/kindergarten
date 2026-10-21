//go:build wireinject
// +build wireinject

package app

import (
	"github.com/gin-gonic/gin"
	"github.com/google/wire"

	"kindergarten-server/internal/config"
	handler "kindergarten-server/internal/handler/http"
	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/rbac"
	"kindergarten-server/internal/repository"
	"kindergarten-server/internal/service"
)

func InjectEngine(cfg *config.Config) (*gin.Engine, error) {
	wire.Build(
		ProvideGorm,
		ProvideRedis,
		ProvideWeChatClient,
		ProvideJWTSecret,
		ProvideAccessTTL,
		ProvideRefreshTTL,
		ProvideCasbinConfig,
		ProvideOSSStorage,
		ProvideFaceConfig,
		ProvideFaceProvider,
		rbac.NewEnforcer,
		repository.Set,
		// service 层为了可测试性把依赖声明成接口，Wire 不会自动把具体类型
		// 匹配上去 —— 缺了这些 Bind 会报 "no provider found for <iface>"。
		wire.Bind(new(service.ClassRepository), new(*repository.ClassRepo)),
		wire.Bind(new(service.GrowthRepository), new(*repository.GrowthRepo)),
		wire.Bind(new(service.OpLogger), new(*repository.OperationLogRepo)),
		wire.Bind(new(service.TeacherLookup), new(*repository.UserRoleRepo)),
		// face.Provider 由 ProvideFaceProvider 直接返回接口，无需 Bind
		// NewRouter 的细闸中间件按接口收依赖（middleware.PermissionResolver）
		wire.Bind(new(middleware.PermissionResolver), new(*service.AdminAuthzService)),
		ProvideAuthService,
		service.Set,
		handler.Set,
		NewAppHandler,
		NewRouter,
	)
	return nil, nil
}
