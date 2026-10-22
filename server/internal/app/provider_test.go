package app

import "testing"

// 业务日期依赖 DSN 的 loc 与进程时区同源；这里钉住三种配置的判定结果。
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
		{"loc=UTC", "u:p@tcp(127.0.0.1:3306)/db?loc=UTC", true},
		{"DSN 非法", "not a dsn", true},
	}
	for _, c := range cases {
		got := dsnLocIssue(c.dsn) != ""
		if got != c.want {
			t.Errorf("%s: dsnLocIssue(%q) 报问题=%v, 期望 %v", c.name, c.dsn, got, c.want)
		}
	}
}
