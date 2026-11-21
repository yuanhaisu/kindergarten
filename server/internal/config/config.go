package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/ini.v1"
)

type Config struct {
	App     AppConfig     `ini:"app"`
	Server  ServerConfig  `ini:"server"`
	Log     LogConfig     `ini:"log"`
	MySQL   MySQLConfig   `ini:"mysql"`
	Redis   RedisConfig   `ini:"redis"`
	JWT     JWTConfig     `ini:"jwt"`
	WeChat  WeChatConfig  `ini:"wechat"`
	Clients ClientConfig  `ini:"clients"`
	Casbin  CasbinConfig  `ini:"casbin"`
	OSS     OSSConfig     `ini:"oss"`
	Face    FaceConfig    `ini:"face"`
	Consent ConsentConfig `ini:"consent"`
}

// ServerConfig 进程级网络配置。
type ServerConfig struct {
	// TrustedProxies 是可信反向代理来源，逗号分隔。只有来自这些地址的
	// X-Forwarded-For 才会被采信为 ClientIP（审计 IP 依赖它）。
	// 未配置时回落到本机回环：生产 Nginx 与后端同机。
	TrustedProxies string `ini:"trusted_proxies"`
}

// DefaultTrustedProxies 是未配置 [server] trusted_proxies 时的回环回退。
var DefaultTrustedProxies = []string{"127.0.0.1", "::1"}

// TrustedProxyList 把逗号分隔的 trusted_proxies 拆成去空格的来源列表；
// 空配置返回本机回环默认值。
func (s ServerConfig) TrustedProxyList() []string {
	parts := strings.Split(s.TrustedProxies, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if proxy := strings.TrimSpace(part); proxy != "" {
			out = append(out, proxy)
		}
	}
	if len(out) == 0 {
		return append([]string(nil), DefaultTrustedProxies...)
	}
	return out
}

type AppConfig struct {
	HTTPAddr string `ini:"http_addr"`
	GinMode  string `ini:"gin_mode"`
	// CORSOrigins 是允许跨域访问的来源，逗号分隔。留空表示不向任何跨域来源
	// 下发 CORS 头（管理端经 Nginx 同源访问，不依赖 CORS）。
	// 生产由部署模板渲染为 https://admin.<domain>；本地开发可加 http://localhost:3000。
	CORSOrigins string `ini:"cors_origins"`
}

// AllowedCORSOrigins 把逗号分隔的 cors_origins 拆成去空格的来源列表。
// 空白项被丢弃；空配置返回空切片，中间件据此默认拒绝所有跨域来源。
func (a AppConfig) AllowedCORSOrigins() []string {
	parts := strings.Split(a.CORSOrigins, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		if origin := strings.TrimSpace(part); origin != "" {
			origins = append(origins, origin)
		}
	}
	return origins
}

// ginReleaseMode 与 gin.ReleaseMode 同值。config 层不引入 gin 依赖，故在此声明。
const ginReleaseMode = "release"

// corsWildcardOrigin 是 CORS 中间件唯一识别的通配来源。
const corsWildcardOrigin = "*"

// CORSConfigWarning 返回 release 模式下 cors_origins 含通配符时的告警文本。
// 返回空串表示无需告警：非 release 模式允许本地调试放开，空配置本身就是默认拒绝。
// 只告警不阻断启动，避免误伤未知部署。
func (a AppConfig) CORSConfigWarning() string {
	if a.GinMode != ginReleaseMode {
		return ""
	}
	for _, origin := range a.AllowedCORSOrigins() {
		if origin == corsWildcardOrigin {
			return `cors_origins contains "*" in release mode: every origin can read cross-origin API responses; configure explicit admin origins (see deploy/config/app.ini.tmpl)`
		}
	}
	return ""
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
// 两端 AppID 必须分开：openid 是「用户×小程序」二元的，身份表以 (app_id, open_id)
// 唯一定位客户端身份；共用 AppID 会让同一自然人在两端的身份无法区分。
// 另外家长端需要非个人主体才能使用 getPhoneNumber。
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

// ClientConfig 小程序客户端注册表（#163）的配置项。
//
// AppID 仍来自 [wechat]（登录凭据与注册表必须同源）；这里只放启动同步
// 首次插入时使用的发版版本，之后 release_version 由数据库维护。
type ClientConfig struct {
	TeacherReleaseVersion string `ini:"teacher_release_version"`
	ParentReleaseVersion  string `ini:"parent_release_version"`
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
	Provider    string `ini:"provider"` // mock | tencent
	Enabled     bool   `ini:"enabled"`  // 全局开关（灰度时按园所白名单另控）
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

// ConsentConfig 隐私政策同意门禁。
// Enforce 默认 false：先发布带同意页的三端版本，确认旧版小程序已退出后再开启，
// 避免线上旧客户端全部收到 403（ADR-0009 向后兼容）。
type ConsentConfig struct {
	Enforce bool `ini:"enforce"`
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
	if v := os.Getenv("CORS_ORIGINS"); v != "" {
		c.App.CORSOrigins = v
	}
	if v := os.Getenv("JWT_ACCESS_TTL_MIN"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.JWT.AccessTTLMin = n
		}
	}
}
