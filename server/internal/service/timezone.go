package service

import (
	"time"

	// 内嵌时区库：最小镜像（无 tzdata）下 LoadLocation("Asia/Shanghai") 会失败，
	// shanghaiLoc() 只能回退 time.Local，下面 init 的对齐就失去意义。
	// 放在独立文件里，避免挂在某个业务 service 上被重构时一并删掉。
	_ "time/tzdata"
)

// shanghaiLoc 返回园所业务时区。任务起止、周期区间、签到日期等业务日期
// 一律按 Asia/Shanghai 计算。
func shanghaiLoc() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.Local
	}
	return loc
}

func init() {
	// 把 time.Local 对齐到业务时区。这一步不是可选优化：
	// MySQL DSN 用的是 loc=Local，驱动在写 DATE/DATETIME 前会执行 t.In(Local)。
	// 若进程 TZ 是 UTC（容器默认），业务日期的「2026-10-21 00:00 +08」会被
	// 格式化成 2026-10-20 16:00，DATE 列随即少一天。
	// 对齐后 time.Local == shanghaiLoc()，读写两侧是同一个墙钟。
	// 必须在任何 sql.Open 之前执行——包级 init 早于 main 的依赖注入。
	time.Local = shanghaiLoc()
}
