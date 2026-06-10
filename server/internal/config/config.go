package config

import (
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

type WeChatConfig struct {
	AppID  string `ini:"app_id"`
	Secret string `ini:"secret"`
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

func Load(path string) (*Config, error) {
	f, err := ini.Load(path)
	if err != nil {
		return nil, err
	}

	var c Config
	if err := f.MapTo(&c); err != nil {
		return nil, err
	}

	overrideFromEnv(&c)
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
