package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWeChatConfigCredentialPerEnd(t *testing.T) {
	c := WeChatConfig{
		TeacherAppID:  "wx_teacher",
		TeacherSecret: "sec_teacher",
		ParentAppID:   "wx_parent",
		ParentSecret:  "sec_parent",
	}

	gotID, gotSecret, err := c.Credential("teacher")
	if err != nil {
		t.Fatalf("Credential(teacher) unexpected error: %v", err)
	}
	if gotID != "wx_teacher" || gotSecret != "sec_teacher" {
		t.Errorf("Credential(teacher) = (%q, %q), want teacher pair", gotID, gotSecret)
	}

	gotID, gotSecret, err = c.Credential("parent")
	if err != nil {
		t.Fatalf("Credential(parent) unexpected error: %v", err)
	}
	if gotID != "wx_parent" || gotSecret != "sec_parent" {
		t.Errorf("Credential(parent) = (%q, %q), want parent pair", gotID, gotSecret)
	}
}

// 未配置某端时必须为空串并报错，绝不回退到另一端凭据——回退即重现 openid 串号。
func TestWeChatConfigCredentialNoFallback(t *testing.T) {
	c := WeChatConfig{
		TeacherAppID:  "wx_teacher",
		TeacherSecret: "sec_teacher",
	}

	if c.ParentAppID != "" || c.ParentSecret != "" {
		t.Fatalf("parent fields must stay empty when unconfigured, got (%q, %q)", c.ParentAppID, c.ParentSecret)
	}

	if _, _, err := c.Credential("parent"); err == nil {
		t.Fatal("Credential(parent) must fail when parent is unconfigured, not fall back to teacher")
	}

	if gotID, _, _ := c.Credential("teacher"); gotID != "wx_teacher" {
		t.Errorf("Credential(teacher) = %q, want wx_teacher", gotID)
	}
}

func TestWeChatConfigCredentialUnknownRole(t *testing.T) {
	c := WeChatConfig{TeacherAppID: "wx_teacher", TeacherSecret: "sec_teacher"}

	for _, role := range []string{"", "admin", "doctor", "TEACHER"} {
		if _, _, err := c.Credential(role); err == nil {
			t.Errorf("Credential(%q) must fail for unknown role", role)
		}
	}
}

// 缺 secret 也算未配置，不能拿 appid 就去换会话。
func TestWeChatConfigCredentialMissingSecret(t *testing.T) {
	c := WeChatConfig{ParentAppID: "wx_parent"}

	if _, _, err := c.Credential("parent"); err == nil {
		t.Fatal("Credential(parent) must fail when secret is missing")
	}
}

// 旧单对键仅在两端都空时回填到两端，用于回滚到共用 AppID。
func TestWeChatConfigNormalizeLegacyFillsBoth(t *testing.T) {
	c := WeChatConfig{AppID: "wx_legacy", Secret: "sec_legacy"}
	c.normalize()

	if c.TeacherAppID != "wx_legacy" || c.ParentAppID != "wx_legacy" {
		t.Errorf("legacy app_id should fill both ends, got teacher=%q parent=%q", c.TeacherAppID, c.ParentAppID)
	}
	if c.TeacherSecret != "sec_legacy" || c.ParentSecret != "sec_legacy" {
		t.Errorf("legacy secret should fill both ends, got teacher=%q parent=%q", c.TeacherSecret, c.ParentSecret)
	}
}

// 已配置的端不被旧单对键覆盖。
func TestWeChatConfigNormalizeLegacyDoesNotOverride(t *testing.T) {
	c := WeChatConfig{
		TeacherAppID:  "wx_teacher",
		TeacherSecret: "sec_teacher",
		AppID:         "wx_legacy",
		Secret:        "sec_legacy",
	}
	c.normalize()

	if c.TeacherAppID != "wx_teacher" || c.TeacherSecret != "sec_teacher" {
		t.Errorf("teacher pair must not be overridden by legacy, got (%q, %q)", c.TeacherAppID, c.TeacherSecret)
	}
	if c.ParentAppID != "" || c.ParentSecret != "" {
		t.Errorf("parent must stay empty (no fallback to legacy when teacher already set), got (%q, %q)", c.ParentAppID, c.ParentSecret)
	}
}

// 小程序码版本缺省是正式版：少配一个键不能静默生成体验版/开发版的码。
func TestWeChatConfigCodeEnvVersionDefault(t *testing.T) {
	c := WeChatConfig{}
	c.normalize()
	if c.CodeEnvVersion != "release" {
		t.Errorf("default code_env_version = %q, want release", c.CodeEnvVersion)
	}

	c = WeChatConfig{CodeEnvVersion: "trial"}
	c.normalize()
	if c.CodeEnvVersion != "trial" {
		t.Errorf("explicit code_env_version must be preserved, got %q", c.CodeEnvVersion)
	}
}

// cors_origins 是逗号分隔字符串：空配置得到空列表，空白项被丢弃，各项去空格。
func TestAppConfigAllowedCORSOrigins(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want []string
	}{
		{name: "empty", raw: "", want: nil},
		{name: "blank items only", raw: " , ,, ", want: nil},
		{name: "wildcard", raw: "*", want: []string{"*"}},
		{
			name: "spaces around comma separated origins",
			raw:  " https://admin.example.com ,\t https://ops.example.com ,",
			want: []string{"https://admin.example.com", "https://ops.example.com"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := AppConfig{CORSOrigins: tc.raw}.AllowedCORSOrigins()
			if len(got) != len(tc.want) {
				t.Fatalf("AllowedCORSOrigins(%q) = %#v, want %#v", tc.raw, got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("AllowedCORSOrigins(%q)[%d] = %q, want %q", tc.raw, i, got[i], tc.want[i])
				}
			}
		})
	}
}

// Load 必须把 app.ini 的 cors_origins 原样读入字符串，再由 AllowedCORSOrigins 解析；
// 空值与通配符是两种不同的部署契约，不能在加载阶段被归一化。
func TestLoadCORSOrigins(t *testing.T) {
	cases := []struct {
		name    string
		iniVal  string
		wantLen int
		want    string
	}{
		{name: "empty", iniVal: "", wantLen: 0},
		{name: "wildcard", iniVal: "*", wantLen: 1, want: "*"},
		{name: "staging admin origin", iniVal: "https://test-admin.example.com", wantLen: 1, want: "https://test-admin.example.com"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("CORS_ORIGINS", "") // 避免宿主机环境变量覆盖测试输入
			path := filepath.Join(t.TempDir(), "app.ini")
			content := "[app]\ncors_origins = " + tc.iniVal + "\n"
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatalf("write temp ini: %v", err)
			}

			cfg, err := Load(path)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			got := cfg.App.AllowedCORSOrigins()
			if len(got) != tc.wantLen {
				t.Fatalf("AllowedCORSOrigins() = %#v, want len %d", got, tc.wantLen)
			}
			if tc.want != "" && got[0] != tc.want {
				t.Errorf("AllowedCORSOrigins()[0] = %q, want %q", got[0], tc.want)
			}
		})
	}
}

// release 模式下误配通配符只告警、不阻断启动；其他模式或非通配配置不应产生告警。
func TestAppConfigCORSConfigWarning(t *testing.T) {
	cases := []struct {
		name        string
		ginMode     string
		corsOrigins string
		wantWarn    bool
	}{
		{name: "release wildcard", ginMode: "release", corsOrigins: "*", wantWarn: true},
		{name: "release wildcard among origins", ginMode: "release", corsOrigins: " https://admin.example.com , * ", wantWarn: true},
		{name: "release explicit origins", ginMode: "release", corsOrigins: "https://admin.example.com", wantWarn: false},
		{name: "release empty", ginMode: "release", corsOrigins: "", wantWarn: false},
		{name: "debug wildcard", ginMode: "debug", corsOrigins: "*", wantWarn: false},
		{name: "empty mode wildcard", ginMode: "", corsOrigins: "*", wantWarn: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := AppConfig{GinMode: tc.ginMode, CORSOrigins: tc.corsOrigins}.CORSConfigWarning()
			if tc.wantWarn && got == "" {
				t.Fatal("CORSConfigWarning() = \"\", want a warning")
			}
			if !tc.wantWarn && got != "" {
				t.Errorf("CORSConfigWarning() = %q, want no warning", got)
			}
			if tc.wantWarn && !strings.Contains(got, `"*"`) {
				t.Errorf("warning should name the wildcard, got %q", got)
			}
		})
	}
}
