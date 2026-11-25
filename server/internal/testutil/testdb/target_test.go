package testdb_test

import (
	"strings"
	"testing"

	"kindergarten-server/internal/testutil/testdb"
)

// 护栏与报错信息都会打印目标库；一旦带上账号口令，日志、CI 输出与工单里就会
// 出现凭据。Target 的契约只有一条：只回显 host:port/db，其余一律丢弃。
func TestTargetKeepsOnlyHostPortAndDatabase(t *testing.T) {
	cases := []struct {
		name string
		dsn  string
		want string
	}{
		{"常规 tcp", "kg_app:pass@tcp(127.0.0.1:3306)/kindergarten?charset=utf8mb4&parseTime=True", "127.0.0.1:3306/kindergarten"},
		// 口令里带 @：必须按 DSN 语法找最后一个 @，不能按第一个 @ 截断。
		{"口令含 @", "user:p@ss@tcp(10.0.0.5:3306)/db_test", "10.0.0.5:3306/db_test"},
		{"远程实例", "root:pw@tcp(rm-xxx.mysql.rds.aliyuncs.com:3306)/kg_dev?loc=Local", "rm-xxx.mysql.rds.aliyuncs.com:3306/kg_dev"},
		// 非 tcp 的 Addr 是 socket 路径，冒充 host 只会误导排查。
		{"unix socket", "user:secret@unix(/tmp/mysql.sock)/db", testdb.UnparseableTarget},
		{"无法解析", "weird-dsn", testdb.UnparseableTarget},
		{"缺库名", "user:pw@tcp(host:3306)/", testdb.UnparseableTarget},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := testdb.Target(c.dsn)
			if got != c.want {
				t.Fatalf("Target(%q) = %q, want %q", c.dsn, got, c.want)
			}
			for _, secret := range []string{"pass", "secret", "pw", "kg_app", "root"} {
				if strings.Contains(got, secret) {
					t.Fatalf("Target 输出不得包含凭据/账号片段 %q：%q", secret, got)
				}
			}
		})
	}
}
