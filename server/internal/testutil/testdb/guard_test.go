package testdb_test

import (
	"strings"
	"testing"

	"kindergarten-server/internal/testutil/testdb"
)

// 测试用 DSN 刻意用短占位口令，既不是真凭据，也不该触发 check-secrets 规则。
const (
	safeTestDSN = "kg_app:s3cret-pw@tcp(127.0.0.1:3316)/kindergarten_test?charset=utf8mb4&parseTime=True"
	devLikeDSN  = "kg_app:s3cret-pw@tcp(127.0.0.1:3316)/kindergarten?charset=utf8mb4&parseTime=True"
	// 开发库也可能带 test 字样；与显式 DSN 同名时必须靠「同名」规则拒绝。
	devNamedTestDSN = "kg_app:s3cret-pw@tcp(127.0.0.1:3316)/kg_test?charset=utf8mb4&parseTime=True"
)

func TestGuardRejectsUnsafeTargets(t *testing.T) {
	cases := []struct {
		name     string
		explicit string
		dev      string
		wantSub  string
	}{
		{"空 DSN", "", devLikeDSN, testdb.TestDSNEnv},
		{"无法解析", "weird-dsn", devLikeDSN, testdb.UnparseableTarget},
		{"缺库名", "user:pw@tcp(127.0.0.1:3306)/", devLikeDSN, "必须包含数据库名"},
		// fail closed：库名没有 test 标记时，即使开发库可读且不同名，也不能确认它是测试库。
		{"无 test 标记", devLikeDSN, devLikeDSN, "无法确认是测试库"},
		{"无 test 标记且开发库未知", "user:pw@tcp(127.0.0.1:3306)/kg_ci", "", "无法确认是测试库"},
		{"与开发库同名", devNamedTestDSN, devNamedTestDSN, "指向开发库"},
		{"开发库 DSN 无法解析", safeTestDSN, "weird-dev-dsn", "无法解析开发库 DSN"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := testdb.Guard(c.explicit, c.dev)
			if err == nil {
				t.Fatalf("Guard(%q, %q) 应拒绝，实际放行", c.explicit, c.dev)
			}
			msg := err.Error()
			if !strings.Contains(msg, c.wantSub) {
				t.Fatalf("错误信息应包含 %q，实际：%s", c.wantSub, msg)
			}
			assertNoCredentials(t, msg)
		})
	}
}

func TestGuardAcceptsIndependentTestDatabase(t *testing.T) {
	cases := []struct {
		name     string
		explicit string
		dev      string
	}{
		{"带 test 标记且与开发库不同名", safeTestDSN, devLikeDSN},
		{"大小写不敏感", "user:pw@tcp(127.0.0.1:3306)/KINDERGARTEN_TEST", devLikeDSN},
		{"开发库未知（靠标记判定）", safeTestDSN, ""},
		{"同主机不同测试库", safeTestDSN, "user:pw@tcp(127.0.0.1:3316)/kindergarten_old"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := testdb.Guard(c.explicit, c.dev); err != nil {
				t.Fatalf("Guard(%q, %q) 应放行，实际：%v", c.explicit, c.dev, err)
			}
		})
	}
}

// 护栏的报错会进 CI 日志与工单；任何账号/口令片段都不能出现。
func TestGuardNeverLeaksCredentials(t *testing.T) {
	dsns := []string{
		safeTestDSN,
		devLikeDSN,
		"user:topsecret@tcp(127.0.0.1:3306)/kindergarten",
		"malformed:topsecret@tcp",
	}
	for _, explicit := range dsns {
		err := testdb.Guard(explicit, devLikeDSN)
		if err == nil {
			continue
		}
		assertNoCredentials(t, err.Error())
	}
}

func assertNoCredentials(t *testing.T, msg string) {
	t.Helper()
	for _, secret := range []string{"s3cret-pw", "topsecret", "kg_app:", "user:", "user:pw"} {
		if strings.Contains(msg, secret) {
			t.Fatalf("错误信息泄露了凭据片段 %q：%s", secret, msg)
		}
	}
}
