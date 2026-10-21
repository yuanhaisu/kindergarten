package app

import (
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/repository"
	"kindergarten-server/internal/service"
)

func ProvideAuthService(
	db *gorm.DB,
	userRepo *repository.UserRepo,
	userRoleRepo *repository.UserRoleRepo,
	invitationRepo *repository.TeacherInvitationRepo,
	adminAuthz *service.AdminAuthzService,
	rdb *redis.Client,
	wx *auth.WeChatClient,
	secret JWTSecret,
	atTTL AccessTTL,
	rtTTL RefreshTTL,
) *service.AuthService {
	return service.NewAuthService(
		db,
		userRepo,
		userRoleRepo,
		invitationRepo,
		adminAuthz,
		rdb,
		wx,
		[]byte(secret),
		time.Duration(atTTL),
		time.Duration(rtTTL),
	)
}
