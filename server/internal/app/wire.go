//go:build wireinject
// +build wireinject

package app

import (
	"github.com/gin-gonic/gin"
	"github.com/google/wire"

	"kindergarten-server/internal/config"
	handler "kindergarten-server/internal/handler/http"
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
		rbac.NewEnforcer,
		repository.Set,
		ProvideAuthService,
		service.Set,
		handler.Set,
		NewAppHandler,
		NewRouter,
	)
	return nil, nil
}
