package app

import (
	"testing"

	"kindergarten-server/internal/config"
)

// 业务日期依赖 DSN 的 loc 与进程时区同源；这里钉住几种配置的判定结果。
// 不连库：ParseDSN 只解析字符串。
func TestDSNLocIssue(t *testing.T) {
	cases := []struct {
		name string
		dsn  string
		want bool // 是否应报问题
	}{
		{"未设置 loc", "u:p@tcp(127.0.0.1:3306)/db", true},
		{"loc=Local", "u:p@tcp(127.0.0.1:3306)/db?loc=Local", false},
		{"loc=Asia/Shanghai", "u:p@tcp(127.0.0.1:3306)/db?loc=Asia%2FShanghai", false},
		// 这两个是 +08 的链接名，名字不同但偏移相同：按名字比较会误报，
		// 而一条会说谎的启动告警等于没有告警。
		{"loc=PRC（+08 链接名）", "u:p@tcp(127.0.0.1:3306)/db?loc=PRC", false},
		{"loc=Asia/Chongqing（+08 链接名）", "u:p@tcp(127.0.0.1:3306)/db?loc=Asia%2FChongqing", false},
		{"loc=UTC", "u:p@tcp(127.0.0.1:3306)/db?loc=UTC", true},
		{"loc=Asia/Urumqi（+06，真的不一致）", "u:p@tcp(127.0.0.1:3306)/db?loc=Asia%2FUrumqi", true},
		{"DSN 非法", "not a dsn", true},
	}
	for _, c := range cases {
		got := dsnLocIssue(c.dsn) != ""
		if got != c.want {
			t.Errorf("%s: dsnLocIssue(%q) 报问题=%v, 期望 %v", c.name, c.dsn, got, c.want)
		}
	}
}

// 启动装配必须拒绝缺失或相同的签名密钥：这是两个信任域隔离的唯一兜底，
// 一旦放行，园所与平台凭证就重新共用一个可伪造的密钥。
func TestProvideJWTSecretsRejectUnsafeConfig(t *testing.T) {
	cases := []struct {
		name    string
		jwt     config.JWTConfig
		wantErr bool
	}{
		{name: "缺失平台密钥", jwt: config.JWTConfig{Secret: "tenant-secret"}, wantErr: true},
		{name: "缺失园所密钥", jwt: config.JWTConfig{PlatformSecret: "platform-secret"}, wantErr: true},
		{name: "两者相同", jwt: config.JWTConfig{Secret: "shared", PlatformSecret: "shared"}, wantErr: true},
		{name: "两者独立", jwt: config.JWTConfig{Secret: "tenant-secret", PlatformSecret: "platform-secret"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{JWT: tc.jwt}

			tenant, err := ProvideJWTSecret(cfg)
			if tc.wantErr {
				if err == nil {
					t.Fatal("ProvideJWTSecret() = nil error，不安全的密钥配置必须在启动时失败")
				}
				return
			}
			if err != nil {
				t.Fatalf("ProvideJWTSecret() unexpected error: %v", err)
			}
			if string(tenant) != tc.jwt.Secret {
				t.Errorf("ProvideJWTSecret() = %q, want %q", tenant, tc.jwt.Secret)
			}

			platform, err := ProvidePlatformJWTSecret(cfg)
			if err != nil {
				t.Fatalf("ProvidePlatformJWTSecret() unexpected error: %v", err)
			}
			if string(platform) != tc.jwt.PlatformSecret {
				t.Errorf("ProvidePlatformJWTSecret() = %q, want %q", platform, tc.jwt.PlatformSecret)
			}
		})
	}
}
