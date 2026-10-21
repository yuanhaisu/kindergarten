package service

import (
	"testing"
	"time"
)

// 进程时区必须与业务时区一致：MySQL DSN 用 loc=Local，驱动按 time.Local
// 格式化/解析日期列；若两者不同源，DATE 列会整体差一天（容器里 TZ 默认 UTC）。
func TestProcessTimezoneAlignedWithBusiness(t *testing.T) {
	// 不能只比较 time.Local 与 shanghaiLoc()：LoadLocation 失败时后者会回退到
	// time.Local，两边都是 "Local"，断言会假通过。直接钉死名字。
	if got := time.Local.String(); got != "Asia/Shanghai" {
		t.Fatalf("time.Local = %q，期望 timezone.go 的 init 把它对齐到 Asia/Shanghai", got)
	}
	// 业务日期写入后必须仍是同一天：用驱动同样的换算（t.In(Local)）再取日期部分。
	day := time.Date(2026, 10, 21, 0, 0, 0, 0, shanghaiLoc())
	formatted := day.In(time.Local)
	if y, m, d := formatted.Date(); y != 2026 || m != 10 || d != 21 {
		t.Fatalf("业务日期经 time.Local 换算后变成 %v，DATE 列会少一天", formatted.Format("2006-01-02"))
	}
}
