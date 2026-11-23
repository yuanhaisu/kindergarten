package testdb

import (
	"regexp"
	"strings"
	"testing"
)

var validDBName = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// 派生库名必须在「同一分支同一测试包」内稳定、跨包/跨分支不同，
// 否则并行 `go test ./...` 会互相清表。
func TestDeriveDatabaseNameIsPackageAndBranchScoped(t *testing.T) {
	base := "kindergarten_test"
	a := deriveDatabaseName(base, "ticket/207-test-dsn", "service")
	b := deriveDatabaseName(base, "ticket/207-test-dsn", "service")
	if a != b {
		t.Fatalf("同输入必须稳定：%q != %q", a, b)
	}
	if !strings.HasPrefix(a, "kindergarten_test_") {
		t.Fatalf("不应出现 _test_test_ 双标记：%q", a)
	}
	if !validDBName.MatchString(a) {
		t.Fatalf("派生库名含非法字符（分支名不能漏进库名）：%q", a)
	}
	if len(a) > 64 {
		t.Fatalf("超过 MySQL 标识符上限 64：%q", a)
	}
	if a == base {
		t.Fatalf("派生库名不能等于基础测试库：%q", a)
	}

	if got := deriveDatabaseName(base, "ticket/207-test-dsn", "repository"); got == a {
		t.Fatalf("不同测试包必须落到不同库：%q", got)
	}
	if got := deriveDatabaseName(base, "main", "service"); got == a {
		t.Fatalf("不同分支必须落到不同库：%q", got)
	}
}

func TestDeriveDatabaseNameAlwaysCarriesTestMarker(t *testing.T) {
	cases := []struct {
		name   string
		base   string
		prefix string
	}{
		{"基础库无标记", "kindergarten", "kindergarten_test_"},
		{"大小写标记只保留一个", "Kindergarten_TEST", "Kindergarten_test_"},
		{"空库名回退", "", "kindergarten_test_"},
		{"只有标记", "_test", "kindergarten_test_"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := deriveDatabaseName(c.base, "b", "p")
			if !strings.HasPrefix(got, c.prefix) {
				t.Fatalf("deriveDatabaseName(%q) = %q，应以 %q 开头", c.base, got, c.prefix)
			}
			if strings.Count(got, "_test_") != 1 {
				t.Fatalf("应恰好一个 _test_ 标记：%q", got)
			}
		})
	}
}

func TestDeriveDatabaseNameSanitizesAndTruncates(t *testing.T) {
	got := deriveDatabaseName("my db;drop`x", "b", "p")
	if !validDBName.MatchString(got) {
		t.Fatalf("非法字符必须替换为下划线：%q", got)
	}

	long := deriveDatabaseName(strings.Repeat("a", 80), "b", "p")
	if len(long) > 64 {
		t.Fatalf("长基础库名必须截断到 64 内：len=%d %q", len(long), long)
	}
	if !strings.Contains(long, "_test_") {
		t.Fatalf("截断不能吃掉 _test_ 标记：%q", long)
	}
}

func TestPackageScopeLooksLikeDBName(t *testing.T) {
	got := packageScope()
	if got == "" || !validDBName.MatchString(got) {
		t.Fatalf("packageScope 必须是合法库名片段：%q", got)
	}
}
