package main

// seedattendance 灌入演示用出勤数据。
//
// 这段逻辑原先挂在 AttendanceService 的读取路径上（Overview / Records / Export），
// 任何人打开一次后台看板就会往生产表写入编造出勤。现改为人工触发。
//
// 用法：
//
//	go run ./cmd/seedattendance                  # 默认 kindergarten 4
//	SEED_KG_ID=7 go run ./cmd/seedattendance     # 指定园所
//
// 幂等：目标日期范围内已有出勤记录时直接跳过，不覆盖真实数据。

import (
	"context"
	"fmt"
	"log"
	"time"

	"kindergarten-server/cmd/internal/seedutil"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

func main() {
	// 先校验入参、再连库：非法 SEED_KG_ID 应当立刻报错，而不是等连库失败后才发现。
	kgID := seedutil.KindergartenID()
	loc := seedutil.Shanghai()
	db := seedutil.Open()

	if err := seed(context.Background(), repository.NewAttendanceRepo(db), kgID, time.Now().In(loc), loc); err != nil {
		log.Fatal("seed:", err)
	}
}

func seed(ctx context.Context, repo *repository.AttendanceRepo, kindergartenID uint64, anchor time.Time, loc *time.Location) error {
	from := time.Date(anchor.Year(), anchor.Month(), anchor.Day(), 0, 0, 0, 0, loc)
	toExclusive := from.AddDate(0, 0, 1)

	total, err := repo.CountByDateRange(ctx, kindergartenID, from, toExclusive)
	if err != nil {
		return fmt.Errorf("count existing: %w", err)
	}
	if total > 0 {
		fmt.Printf("kindergarten %d 在 %s 已有 %d 条出勤记录，跳过\n",
			kindergartenID, from.Format("2006-01-02"), total)
		return nil
	}

	students, err := repo.ListActiveStudents(ctx, kindergartenID)
	if err != nil {
		return fmt.Errorf("list students: %w", err)
	}
	if len(students) == 0 {
		return fmt.Errorf("kindergarten %d 没有在读学生，无从灌入", kindergartenID)
	}

	days := recentWorkdays(anchor, 3, loc)
	rows := make([]domain.Attendance, 0, len(days)*len(students))
	for di, d := range days {
		for si, st := range students {
			status := synthStatus(di, si)
			rows = append(rows, domain.Attendance{
				KindergartenID: kindergartenID,
				ClassID:        st.ClassID,
				StudentID:      st.ID,
				Date:           d,
				Status:         status,
				CheckInTime:    synthCheckIn(d, si, status, loc),
				RecorderID:     0,
			})
		}
	}

	if err := repo.UpsertBatch(ctx, rows); err != nil {
		return fmt.Errorf("upsert: %w", err)
	}
	fmt.Printf("已为 kindergarten %d 灌入 %d 名学生 × %d 个工作日 = %d 条出勤记录\n",
		kindergartenID, len(students), len(days), len(rows))
	return nil
}

// recentWorkdays 返回 anchor 当天起往前数的 n 个工作日（含当天）。
func recentWorkdays(day time.Time, n int, loc *time.Location) []time.Time {
	out := make([]time.Time, 0, n)
	cur := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc)
	for len(out) < n {
		if wd := cur.Weekday(); wd != time.Saturday && wd != time.Sunday {
			out = append(out, cur)
		}
		cur = cur.AddDate(0, 0, -1)
	}
	return out
}

// synthStatus 按 (天序号, 学生序号) 生成确定性分布，便于复现。
func synthStatus(dayIndex, studentIndex int) domain.AttendanceStatus {
	switch n := (dayIndex + studentIndex) % 10; {
	case n <= 6:
		return domain.AttendancePresent
	case n == 7:
		return domain.AttendanceLate
	case n == 8:
		return domain.AttendanceLeave
	default:
		return domain.AttendanceAbsent
	}
}

func synthCheckIn(day time.Time, studentIndex int, status domain.AttendanceStatus, loc *time.Location) *time.Time {
	if status != domain.AttendancePresent && status != domain.AttendanceLate {
		return nil
	}
	hour := 8
	if status == domain.AttendanceLate {
		hour = 9
	}
	t := time.Date(day.Year(), day.Month(), day.Day(), hour, (studentIndex*7)%30, 0, 0, loc)
	return &t
}
