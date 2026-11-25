package authharness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// CI 用 KDG_TEST_REQUIRE_MYSQL=1 把「测试库不可用」从跳过改成失败。
// 值写错时按「要求」处理：宁可门禁变红，也不能静默跳过。
func TestRequireMySQLEnv(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "unset", value: "", want: false},
		{name: "one", value: "1", want: true},
		{name: "true", value: "true", want: true},
		{name: "zero", value: "0", want: false},
		{name: "false", value: "false", want: false},
		{name: "invalid-fails-closed", value: "yes-please", want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(RequireDBEnv, tc.value)
			if got := requireMySQL(); got != tc.want {
				t.Fatalf("requireMySQL() = %v, want %v（值 %q）", got, tc.want, tc.value)
			}
		})
	}
}

// 显式 DSN 必须过 testdb 的同一套护栏（含「库名带 test 标记」），
// 不能只比「不等于开发库」；错误信息只允许出现脱敏目标。
func TestInitTestDBRejectsExplicitDSNWithoutTestMarker(t *testing.T) {
	t.Setenv(TestDSNEnv, "tester:secret@tcp(127.0.0.1:3306)/kindergartendev?charset=utf8mb4&parseTime=true")
	t.Setenv(configEnv, "")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.test\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatalf("写入 go.mod 失败：%v", err)
	}
	t.Chdir(dir)

	s := initTestDB()
	if s.err == nil {
		t.Fatalf("库名不含 test 的显式 DSN 必须被拒绝，实际 skip：%q", s.skipReason)
	}
	msg := s.err.Error()
	if !strings.Contains(msg, "test") {
		t.Fatalf("错误必须说明缺少 test 标记：%v", s.err)
	}
	for _, secret := range []string{"tester", "secret"} {
		if strings.Contains(msg, secret) {
			t.Fatalf("错误信息不得回显 DSN 中的账号口令：%v", s.err)
		}
	}
}

// 派生测试库必须按测试包隔离：service 包的 client_service_test 会硬删整张
// clients 表并重新 SyncFromConfig，与 authharness 的登录用例共用同一个派生库
// 时会偶发 403 CLIENT_UNAVAILABLE。同一测试包内则必须稳定复用同一个库名。
func TestDeriveTestTargetSeparatesTestPackages(t *testing.T) {
	const dsn = "tester:secret@tcp(127.0.0.1:3306)/kindergarten?charset=utf8mb4&parseTime=true"

	devDatabase, serviceDB, serviceDSN, err := deriveTestTarget(dsn, `C:\repo`, "service")
	if err != nil {
		t.Fatalf("derive(service): %v", err)
	}
	_, authDB, _, err := deriveTestTarget(dsn, `C:\repo`, "authharness")
	if err != nil {
		t.Fatalf("derive(authharness): %v", err)
	}
	_, serviceDBAgain, _, err := deriveTestTarget(dsn, `C:\repo`, "service")
	if err != nil {
		t.Fatalf("derive(service again): %v", err)
	}

	if devDatabase != "kindergarten" {
		t.Fatalf("devDatabase = %q, want kindergarten", devDatabase)
	}
	if serviceDB != serviceDBAgain {
		t.Fatalf("同一测试包的派生库名必须稳定：%q != %q", serviceDB, serviceDBAgain)
	}
	if serviceDB == authDB {
		t.Fatalf("不同测试包不得共用派生库 %q", serviceDB)
	}
	for _, db := range []string{serviceDB, authDB} {
		if !strings.Contains(db, "_test_") {
			t.Fatalf("派生库名 %q 必须带 _test_ 标记", db)
		}
		if len(db) > 64 {
			t.Fatalf("派生库名 %q 超过 MySQL 标识符上限 64", db)
		}
	}
	if !strings.Contains(serviceDSN, serviceDB) {
		t.Fatalf("派生 DSN 必须指向派生库 %q：%q", serviceDB, serviceDSN)
	}
}
