package seedutil

import "testing"

func TestDSNTarget(t *testing.T) {
	// 口令刻意用短占位串：测试数据不是真凭据，也不该触发 check-secrets 的 DSN 规则。
	cases := []struct{ dsn, want string }{
		{"kg_app:pass@tcp(127.0.0.1:3306)/kindergarten?charset=utf8mb4&parseTime=True", "127.0.0.1:3306/kindergarten"},
		{"root:p@tcp(rm-xxx.mysql.rds.aliyuncs.com:3306)/kg_dev?loc=Local", "rm-xxx.mysql.rds.aliyuncs.com:3306/kg_dev"},
		{"weird-dsn-without-tcp", "无法解析 DSN（仅支持 tcp 形式）"},
	}
	for _, c := range cases {
		if got := dsnTarget(c.dsn); got != c.want {
			t.Errorf("dsnTarget(%q) = %q, want %q", c.dsn, got, c.want)
		}
	}
}

func TestDemoOptInError(t *testing.T) {
	t.Setenv(envAllowDemo, "")
	if err := demoOptInError("seedleaves", "db:3306/kg"); err == nil {
		t.Fatal("未设置 SEED_ALLOW_DEMO 时应拒绝")
	}

	t.Setenv(envAllowDemo, "1")
	if err := demoOptInError("seedleaves", "db:3306/kg"); err != nil {
		t.Fatalf("设置 SEED_ALLOW_DEMO=1 后应放行，得到 %v", err)
	}
}

func TestDestructiveOptInRequiresBothSwitches(t *testing.T) {
	t.Setenv(envAllowDemo, "1")
	t.Setenv(envAllowDestructive, "")
	if err := destructiveOptInError("refreshtasks", "db:3306/kg"); err == nil {
		t.Fatal("缺少 SEED_ALLOW_DESTRUCTIVE 时应拒绝")
	}

	t.Setenv(envAllowDestructive, "1")
	t.Setenv(envAllowDemo, "")
	if err := destructiveOptInError("refreshtasks", "db:3306/kg"); err == nil {
		t.Fatal("缺少 SEED_ALLOW_DEMO 时应拒绝")
	}

	t.Setenv(envAllowDemo, "1")
	if err := destructiveOptInError("refreshtasks", "db:3306/kg"); err != nil {
		t.Fatalf("双开关齐全应放行，得到 %v", err)
	}
}
