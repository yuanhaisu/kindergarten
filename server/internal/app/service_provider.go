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
		rdb,
		wx,
		[]byte(secret),
		time.Duration(atTTL),
		time.Duration(rtTTL),
	)
}
