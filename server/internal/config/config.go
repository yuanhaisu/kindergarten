package config

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/ini.v1"
)

type Config struct {
	App    AppConfig    `ini:"app"`
	Log    LogConfig    `ini:"log"`
	MySQL  MySQLConfig  `ini:"mysql"`
	Redis  RedisConfig  `ini:"redis"`
	JWT    JWTConfig    `ini:"jwt"`
	WeChat WeChatConfig `ini:"wechat"`
	Casbin CasbinConfig `ini:"casbin"`
	OSS    OSSConfig    `ini:"oss"`
	Face   FaceConfig   `ini:"face"`
}

type AppConfig struct {
	HTTPAddr string `ini:"http_addr"`
	GinMode  string `ini:"gin_mode"`
}

type LogConfig struct {
	Level  string `ini:"level"`
	Format string `ini:"format"`
}

type MySQLConfig struct {
	DSN string `ini:"dsn"`
}

type RedisConfig struct {
	Addr     string `ini:"addr"`
	Password string `ini:"password"`
	DB       int    `ini:"db"`
}

type JWTConfig struct {
	Secret        string `ini:"secret"`
	AccessTTLMin  int    `ini:"access_ttl_min"`
	RefreshTTLDay int    `ini:"refresh_ttl_day"`
}

// WeChatConfig 微信小程序凭据，按端各一对。
//
// 两端 AppID 必须分开：openid 是「用户×小程序」二元的，共用 AppID 会让同一个人
// 在两端拿到同一个 openid，FindByOpenID 只命中一条 User，另一端角色丢失、接口全 403。
type WeChatConfig struct {
	TeacherAppID  string `ini:"teacher_app_id"`
	TeacherSecret string `ini:"teacher_secret"`
	ParentAppID   string `ini:"parent_app_id"`
	ParentSecret  string `ini:"parent_secret"`

	// CodeEnvVersion 小程序码指向的版本：release（正式）/ trial（体验）/ develop（开发）。
	// 家长端小程序码必须与老师当前可扫码的版本一致，否则扫码进不去（默认 release）。
	CodeEnvVersion string `ini:"code_env_version"`

	// 旧的单对键，仅作回滚兼容：两端都未配置时回填到两端，用于退回共用 AppID。
	// 不要再往这里写新配置。
	AppID  string `ini:"app_id"`
	Secret string `ini:"secret"`
}

// Credential 返回指定端的 (AppID, Secret)。
// role 只接受 "teacher" / "parent"；未知 role 或该端未配置均返回错误，
// 绝不回退到另一端凭据——回退即重现 openid 串号。
func (c WeChatConfig) Credential(role string) (appID, secret string, err error) {
	switch role {
	case "teacher":
		appID, secret = c.TeacherAppID, c.TeacherSecret
	case "parent":
		appID, secret = c.ParentAppID, c.ParentSecret
	default:
		return "", "", fmt.Errorf("wechat credential: unknown role %q", role)
	}
	if appID == "" || secret == "" {
		return "", "", fmt.Errorf("wechat credential: role %q not configured (need both app_id and secret)", role)
	}
	return appID, secret, nil
}

// normalize 回填旧单对键。只在两端都为空时生效，因此不会把一端的凭据泄漏到另一端。
func (c *WeChatConfig) normalize() {
	if c.TeacherAppID == "" && c.ParentAppID == "" && c.AppID != "" {
		c.TeacherAppID, c.ParentAppID = c.AppID, c.AppID
	}
	if c.TeacherSecret == "" && c.ParentSecret == "" && c.Secret != "" {
		c.TeacherSecret, c.ParentSecret = c.Secret, c.Secret
	}
	// 缺省正式版：生成小程序码时不因少配一个键就产出体验版/开发版的码。
	if c.CodeEnvVersion == "" {
		c.CodeEnvVersion = "release"
	}
}

type CasbinConfig struct {
	Enable     bool   `ini:"enable"`
	ModelPath  string `ini:"model_path"`
	PolicyPath string `ini:"policy_path"`
}

type OSSConfig struct {
	Endpoint        string `ini:"endpoint"`
	AccessKeyID     string `ini:"access_key_id"`
	AccessKeySecret string `ini:"access_key_secret"`
	BucketName      string `ini:"bucket_name"`
}

// FaceConfig 人脸识别（腾讯云 IAI）配置。
// 密钥只在本文件与环境变量中出现，绝不下发客户端（见 photo-tagging-face-recognition design D-密钥边界）。
type FaceConfig struct {
	Provider    string `ini:"provider"`     // mock | tencent
	Enabled     bool   `ini:"enabled"`      // 全局开关（灰度时按园所白名单另控）
	SecretID    string `ini:"secret_id"`
	SecretKey   string `ini:"secret_key"`
	Region      string `ini:"region"`       // 如 ap-beijing
	QPS         int    `ini:"qps"`          // 识别调用速率上限（保护配额）
	DailyBudget int    `ini:"daily_budget"` // 日调用上限（预算护栏）
	SuggestHigh int    `ini:"suggest_high"` // 高置信阈值（0-100，预勾建议）
	SuggestLow  int    `ini:"suggest_low"`  // 低置信阈值（0-100，提示不预勾；也是云端 FaceMatchThreshold）
	// GroupPhotoMinFaces 大合影阈值（D11）：照片检测到的人脸数 ≥ 此值时判为大合影——
	// 不产逐人建议（10 张脸的人脸上限下注定不完整），改由老师一键「全班（当日到场）」。0 = 默认 6。
	GroupPhotoMinFaces int `ini:"group_photo_min_faces"`
}

func Load(path string) (*Config, error) {
	f, err := ini.Load(path)
	if err != nil {
		return nil, err
	}

	var c Config
	if err := f.MapTo(&c); err != nil {
		return nil, err
	}
	c.WeChat.normalize()

	overrideFromEnv(&c)
	c.WeChat.normalize()
	return &c, nil
}

func overrideFromEnv(c *Config) {
	if v := os.Getenv("MYSQL_DSN"); v != "" {
		c.MySQL.DSN = v
	}
	if v := os.Getenv("REDIS_ADDR"); v != "" {
		c.Redis.Addr = v
	}
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		c.Redis.Password = v
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		c.JWT.Secret = v
	}
	if v := os.Getenv("WECHAT_TEACHER_APP_ID"); v != "" {
		c.WeChat.TeacherAppID = v
	}
	if v := os.Getenv("WECHAT_TEACHER_SECRET"); v != "" {
		c.WeChat.TeacherSecret = v
	}
	if v := os.Getenv("WECHAT_PARENT_APP_ID"); v != "" {
		c.WeChat.ParentAppID = v
	}
	if v := os.Getenv("WECHAT_PARENT_SECRET"); v != "" {
		c.WeChat.ParentSecret = v
	}
	// 旧的单对变量，仅作回滚兼容；normalize 会把它回填到两端
	if v := os.Getenv("WECHAT_APP_ID"); v != "" {
		c.WeChat.AppID = v
	}
	if v := os.Getenv("WECHAT_SECRET"); v != "" {
		c.WeChat.Secret = v
	}
	if v := os.Getenv("OSS_ENDPOINT"); v != "" {
		c.OSS.Endpoint = v
	}
	if v := os.Getenv("OSS_ACCESS_KEY_ID"); v != "" {
		c.OSS.AccessKeyID = v
	}
	if v := os.Getenv("OSS_ACCESS_KEY_SECRET"); v != "" {
		c.OSS.AccessKeySecret = v
	}
	if v := os.Getenv("OSS_BUCKET"); v != "" {
		c.OSS.BucketName = v
	}
	if v := os.Getenv("GIN_MODE"); v != "" {
		c.App.GinMode = v
	}
	if v := os.Getenv("HTTP_ADDR"); v != "" {
		c.App.HTTPAddr = v
	}
	if v := os.Getenv("JWT_ACCESS_TTL_MIN"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.JWT.AccessTTLMin = n
		}
	}
}
