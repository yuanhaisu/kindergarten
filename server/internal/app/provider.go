package app

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"kindergarten-server/internal/ai/face"
	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/service"
	ossx "kindergarten-server/internal/storage/oss"
)

func ProvideGorm(cfg *config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Warn),
		DisableForeignKeyConstraintWhenMigrating: true,
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
		&domain.NoticeTarget{},
		&domain.NoticeTemplate{},
		&domain.LeaveRequest{},
		&domain.Attendance{},
		&domain.PickupRecord{},
		&domain.PickupAuthorization{},
		&domain.PickupQRCode{},
		&domain.PickupDailyDesignation{},

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
		&domain.TimetableTemplate{},
		&domain.TimetableOverride{},
		&domain.ParentTask{},
		&domain.TaskCheckIn{},
		&domain.TaskResponse{},
		&domain.TaskResponseSupplement{},
		&domain.TaskResponseReaction{},
		&domain.TaskCheckInState{},
		&domain.ClassStarJar{},
		&domain.TaskReadMark{},
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

		// 人脸底库与识别队列
		&domain.StudentFace{},
		&domain.RecognitionJob{},

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

	// 幂等补齐园长角色（align-class-calendar 4.3）；失败不阻断启动，与字典播种同策略
	if err := service.SeedPrincipalRolesForAll(context.Background(), db); err != nil {
		println("principal role seeder: " + err.Error())
	}

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
		rdb,
	)
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
	return ossx.NewStorage(ossx.Config{
		Endpoint:        cfg.OSS.Endpoint,
		AccessKeyID:     cfg.OSS.AccessKeyID,
		AccessKeySecret: cfg.OSS.AccessKeySecret,
		BucketName:      cfg.OSS.BucketName,
	})
}
