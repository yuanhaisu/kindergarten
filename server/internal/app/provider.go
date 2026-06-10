package app

import (
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
	ossx "kindergarten-server/internal/storage/oss"
)

func ProvideGorm(cfg *config.Config) (*gorm.DB, error) {
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

	if err := db.AutoMigrate(
		// 核心实体
		&domain.User{},
		&domain.Kindergarten{},
		&domain.AcademicYear{},
		&domain.KindergartenAppSetting{},
		&domain.Class{},
		&domain.UserRole{},
		&domain.File{},
		&domain.Message{},

		// 师生关系
		&domain.ClassTeacher{},
		&domain.TeacherProfile{},
		&domain.ParentProfile{},
		&domain.Student{},
		&domain.StudentParent{},

		// 通知 / 请假 / 出勤 / 接送
		&domain.Notice{},
		&domain.NoticeAttachment{},
		&domain.NoticeReadLog{},
		&domain.NoticeTemplate{},
		&domain.LeaveRequest{},
		&domain.Attendance{},
		&domain.PickupRecord{},
		&domain.PickupAuthorization{},
		&domain.PickupQRCode{},

		// 成长记录 / 日历 / 任务 / 积分
		&domain.GrowthDailyLog{},
		&domain.GrowthLogStudent{},
		&domain.GrowthMedia{},
		&domain.GrowthMediaStudent{},
		&domain.GrowthComment{},
		&domain.GrowthLike{},
		&domain.GrowthReport{},
		&domain.GrowthReportDimension{},
		&domain.StudentDailySummary{},
		&domain.CalendarEvent{},
		&domain.ParentTask{},
		&domain.TaskCheckIn{},
		&domain.PointAccount{},
		&domain.PointTransaction{},
		&domain.Achievement{},
		&domain.StudentAchievement{},

		// 老师邀请
		&domain.TeacherInvitation{},

		// 公告 / 审核 / RBAC / 推送 / 文件 / 日志 / 反馈
		&domain.Announcement{},
		&domain.Feedback{},
		&domain.AnnouncementAttachment{},
		&domain.ContentAudit{},
		&domain.AdminRole{},
		&domain.AdminPermission{},
		&domain.AdminRolePermission{},
		&domain.UserAdminRole{},
		&domain.WechatPushLog{},
		&domain.FileRecord{},
		&domain.OperationLog{},
		&domain.ExportTask{},

		// 字典
		&domain.SysDictType{},
		&domain.SysDictItem{},

		// 晨检
		&domain.MorningHealthCheck{},
		&domain.HealthCheckMedia{},
	); err != nil {
		return nil, err
	}

	// Seed default dictionary data (idempotent)
	SeedDictsAfterMigration(db)

	return db, nil
}

func ProvideRedis(cfg *config.Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
}

func ProvideWeChatClient(cfg *config.Config, rdb *redis.Client) *auth.WeChatClient {
	return auth.NewWeChatClient(cfg.WeChat.AppID, cfg.WeChat.Secret, rdb)
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

func ProvideOSSStorage(cfg *config.Config) (*ossx.Storage, error) {
	return ossx.NewStorage(ossx.Config{
		Endpoint:        cfg.OSS.Endpoint,
		AccessKeyID:     cfg.OSS.AccessKeyID,
		AccessKeySecret: cfg.OSS.AccessKeySecret,
		BucketName:      cfg.OSS.BucketName,
	})
}
