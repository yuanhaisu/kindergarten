package testdb

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// prepareBareRepo 把一个没有配置文件、没有显式 DSN 的空仓库作为 cwd，
// 用来验证「测试库不可用」时 helper 选择跳过还是失败。
func prepareBareRepo(t *testing.T) {
	t.Helper()
	t.Setenv(TestDSNEnv, "")
	t.Setenv(configEnv, "")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.test\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatalf("写入 go.mod 失败：%v", err)
	}
	t.Chdir(dir)
}

// CI 用 KDG_TEST_REQUIRE_MYSQL=1 把「测试库不可用」从跳过改成失败。
// 未设置该变量时保持本地行为：skip。
func TestInitSetupHonorsRequireMySQL(t *testing.T) {
	prepareBareRepo(t)

	t.Setenv(RequireDBEnv, "")
	optional := initSetup()
	if optional.err != nil {
		t.Fatalf("未要求 MySQL 时不应失败：%v", optional.err)
	}
	if optional.skipReason == "" {
		t.Fatal("未要求 MySQL 且没有测试库时必须跳过")
	}

	t.Setenv(RequireDBEnv, "1")
	required := initSetup()
	if required.err == nil {
		t.Fatalf("已要求 MySQL 且没有测试库时必须失败，实际 skip：%q", required.skipReason)
	}
	if required.skipReason != "" {
		t.Fatalf("失败路径不得同时标记为跳过：%q", required.skipReason)
	}
	if !strings.Contains(required.err.Error(), RequireDBEnv) {
		t.Fatalf("错误必须点名 %s：%v", RequireDBEnv, required.err)
	}
}

// 开关值写错时按「要求」处理：静默跳过会让缺库的 CI 看起来是绿的。
func TestInitSetupTreatsInvalidRequireValueAsRequired(t *testing.T) {
	prepareBareRepo(t)
	t.Setenv(RequireDBEnv, "yes-please")

	if s := initSetup(); s.err == nil {
		t.Fatalf("非法开关值必须失败关闭，实际 skip：%q", s.skipReason)
	}
}
