package config

import "testing"

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
