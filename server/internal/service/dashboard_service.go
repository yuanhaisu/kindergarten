package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

type DashboardService struct {
	dash *repository.DashboardRepo
	kg   *repository.KindergartenRepo
}

func NewDashboardService(dash *repository.DashboardRepo, kg *repository.KindergartenRepo) *DashboardService {
	return &DashboardService{dash: dash, kg: kg}
}

func dayRange(loc *time.Location, t time.Time) (start, end time.Time) {
	y, m, d := t.In(loc).Date()
	start = time.Date(y, m, d, 0, 0, 0, 0, loc)
	end = start.AddDate(0, 0, 1)
	return start, end
}

func monthRange(loc *time.Location, t time.Time) (start, end time.Time) {
	y, m, _ := t.In(loc).Date()
	start = time.Date(y, m, 1, 0, 0, 0, 0, loc)
	end = start.AddDate(0, 1, 0)
	return start, end
}

func pct1(n, d int64) string {
	if d <= 0 {
		return "0%"
	}
	v := float64(n) / float64(d) * 100
	return fmt.Sprintf("%.1f%%", math.Round(v*10)/10)
}

func round1f(numerator, days int64) float64 {
	if days <= 0 {
		return 0
	}
	v := float64(numerator) / float64(days)
	return math.Round(v*10) / 10
}

// --- GET /admin/dashboard/overview ---

type dashboardOverviewKindergarten struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
}

type dashboardOverviewRealtime struct {
	PresentCount      int64  `json:"present_count"`
	LeaveCount        int64  `json:"leave_count"`
	TotalStudentCount int64  `json:"total_student_count"`
	AttendanceRate    string `json:"attendance_rate"`
	PickedUpCount     int64  `json:"picked_up_count"`
	NotPickedUpCount  int64  `json:"not_picked_up_count"`
}

type dashboardOverviewToday struct {
	NewNotices    int64 `json:"new_notices"`
	PendingLeaves int64 `json:"pending_leaves"`
	GrowthLogs    int64 `json:"growth_logs"`
	NewTasks      int64 `json:"new_tasks"`
}

type dashboardOverviewThisMonth struct {
	AvgAttendanceRate  string  `json:"avg_attendance_rate"`
	TotalGrowthLogs    int64   `json:"total_growth_logs"`
	TotalPointsAwarded int64   `json:"total_points_awarded"`
	NewStudents        int64   `json:"new_students"`
	NewParents         int64   `json:"new_parents"`
}

type DashboardOverviewResp struct {
	Kindergarten dashboardOverviewKindergarten `json:"kindergarten"`
	Realtime     dashboardOverviewRealtime     `json:"realtime"`
	Today        dashboardOverviewToday        `json:"today"`
	ThisMonth    dashboardOverviewThisMonth    `json:"this_month"`
}

func (s *DashboardService) Overview(ctx context.Context, kindergartenID uint64) (*DashboardOverviewResp, error) {
	if kindergartenID == 0 {
		return nil, fmt.Errorf("NO_KINDERGARTEN")
	}
	kg, err := s.kg.FindByID(ctx, kindergartenID)
	if err != nil {
		return nil, err
	}
	if kg == nil {
		return nil, fmt.Errorf("KINDERGARTEN_NOT_FOUND")
	}

	loc := shanghaiLoc()
	now := time.Now().In(loc)
	todayStart, todayEnd := dayRange(loc, now)
	monthStart, monthEnd := monthRange(loc, now)

	totalStudents, err := s.dash.CountActiveStudents(ctx, kindergartenID)
	if err != nil {
		return nil, err
	}
	presentToday, err := s.dash.CountAttendanceByStatus(ctx, kindergartenID, todayStart, todayEnd, domain.AttendancePresent)
	if err != nil {
		return nil, err
	}
	leaveToday, err := s.dash.CountAttendanceByStatus(ctx, kindergartenID, todayStart, todayEnd, domain.AttendanceLeave)
	if err != nil {
		return nil, err
	}
	pickedUp, err := s.dash.CountPickupByType(ctx, kindergartenID, todayStart, todayEnd, domain.PickupTypePickUp)
	if err != nil {
		return nil, err
	}

	notPicked := presentToday - pickedUp
	if notPicked < 0 {
		notPicked = 0
	}

	newNotices, err := s.dash.CountNoticesCreatedBetween(ctx, kindergartenID, todayStart, todayEnd)
	if err != nil {
		return nil, err
	}
	pendingLeaves, err := s.dash.CountPendingLeaves(ctx, kindergartenID)
	if err != nil {
		return nil, err
	}
	growthToday, err := s.dash.CountGrowthLogsByDate(ctx, kindergartenID, todayStart, todayEnd)
	if err != nil {
		return nil, err
	}
	newTasks, err := s.dash.CountParentTasksCreatedBetween(ctx, kindergartenID, todayStart, todayEnd)
	if err != nil {
		return nil, err
	}

	mp, ma, ml, err := s.dash.MonthAttendanceRollup(ctx, kindergartenID, monthStart, monthEnd)
	if err != nil {
		return nil, err
	}
	monthDenom := mp + ma + ml
	monthAvg := pct1(mp, monthDenom)

	monthGrowth, err := s.dash.CountGrowthLogsInRange(ctx, kindergartenID, monthStart, monthEnd)
	if err != nil {
		return nil, err
	}
	pointsSum, err := s.dash.SumPositivePointsInRange(ctx, kindergartenID, monthStart, monthEnd)
	if err != nil {
		return nil, err
	}
	newStudents, err := s.dash.CountStudentsCreatedBetween(ctx, kindergartenID, monthStart, monthEnd)
	if err != nil {
		return nil, err
	}
	newParents, err := s.dash.CountNewParentRolesInMonth(ctx, kindergartenID, monthStart, monthEnd)
	if err != nil {
		return nil, err
	}

	return &DashboardOverviewResp{
		Kindergarten: dashboardOverviewKindergarten{ID: kg.ID, Name: kg.Name},
		Realtime: dashboardOverviewRealtime{
			PresentCount:      presentToday,
			LeaveCount:        leaveToday,
			TotalStudentCount: totalStudents,
			AttendanceRate:    pct1(presentToday, totalStudents),
			PickedUpCount:     pickedUp,
			NotPickedUpCount:  notPicked,
		},
		Today: dashboardOverviewToday{
			NewNotices:    newNotices,
			PendingLeaves: pendingLeaves,
			GrowthLogs:    growthToday,
			NewTasks:      newTasks,
		},
		ThisMonth: dashboardOverviewThisMonth{
			AvgAttendanceRate:  monthAvg,
			TotalGrowthLogs:    monthGrowth,
			TotalPointsAwarded: pointsSum,
			NewStudents:        newStudents,
			NewParents:         newParents,
		},
	}, nil
}

// --- GET /admin/dashboard/attendance-trend ---

type AttendanceTrendPoint struct {
	Date   string `json:"date"`
	Present int64 `json:"present"`
	Leave   int64 `json:"leave"`
	Absent  int64 `json:"absent"`
	Rate    string `json:"rate"`
}

type AttendanceTrendResp struct {
	Period string                 `json:"period"`
	Data   []AttendanceTrendPoint `json:"data"`
}

func (s *DashboardService) AttendanceTrend(ctx context.Context, kindergartenID uint64, period string) (*AttendanceTrendResp, error) {
	if kindergartenID == 0 {
		return nil, fmt.Errorf("NO_KINDERGARTEN")
	}
	if period != "7d" && period != "30d" && period != "12m" {
		period = "30d"
	}
	loc := shanghaiLoc()
	now := time.Now().In(loc)
	out := AttendanceTrendResp{Period: period, Data: make([]AttendanceTrendPoint, 0)}

	switch period {
	case "7d":
		for i := 6; i >= 0; i-- {
			d := now.AddDate(0, 0, -i)
			ds, de := dayRange(loc, d)
			p, l, a, err := s.dash.AttendanceDayAgg(ctx, kindergartenID, ds, de)
			if err != nil {
				return nil, err
			}
			denom := p + l + a
			out.Data = append(out.Data, AttendanceTrendPoint{
				Date:    ds.Format("2006-01-02"),
				Present: p,
				Leave:   l,
				Absent:  a,
				Rate:    pct1(p, denom),
			})
		}
	case "30d":
		for i := 29; i >= 0; i-- {
			d := now.AddDate(0, 0, -i)
			ds, de := dayRange(loc, d)
			p, l, a, err := s.dash.AttendanceDayAgg(ctx, kindergartenID, ds, de)
			if err != nil {
				return nil, err
			}
			denom := p + l + a
			out.Data = append(out.Data, AttendanceTrendPoint{
				Date:    ds.Format("2006-01-02"),
				Present: p,
				Leave:   l,
				Absent:  a,
				Rate:    pct1(p, denom),
			})
		}
	case "12m":
		y, m, _ := now.Date()
		for i := 11; i >= 0; i-- {
			t := time.Date(y, m, 1, 0, 0, 0, 0, loc).AddDate(0, -i, 0)
			ms, me := monthRange(loc, t)
			p, l, a, err := s.dash.AttendanceDayAgg(ctx, kindergartenID, ms, me)
			if err != nil {
				return nil, err
			}
			denom := p + l + a
			out.Data = append(out.Data, AttendanceTrendPoint{
				Date:    ms.Format("2006-01-02"),
				Present: p,
				Leave:   l,
				Absent:  a,
				Rate:    pct1(p, denom),
			})
		}
	}
	return &out, nil
}

// --- GET /admin/dashboard/class-ranking ---

type ClassRankingItem struct {
	ClassID   uint64 `json:"class_id"`
	ClassName string `json:"class_name"`
	Present   int64  `json:"present"`
	Total     int64  `json:"total"`
	Rate      string `json:"rate"`
}

type ClassRankingResp struct {
	List []ClassRankingItem `json:"list"`
}

func (s *DashboardService) ClassRanking(ctx context.Context, kindergartenID uint64) (*ClassRankingResp, error) {
	if kindergartenID == 0 {
		return nil, fmt.Errorf("NO_KINDERGARTEN")
	}
	loc := shanghaiLoc()
	now := time.Now().In(loc)
	ds, de := dayRange(loc, now)
	rows, err := s.dash.ClassRankingToday(ctx, kindergartenID, ds, de)
	if err != nil {
		return nil, err
	}
	list := make([]ClassRankingItem, 0, len(rows))
	for _, r := range rows {
		list = append(list, ClassRankingItem{
			ClassID:   r.ClassID,
			ClassName: r.ClassName,
			Present:   r.Present,
			Total:     r.Total,
			Rate:      pct1(r.Present, r.Total),
		})
	}
	sort.Slice(list, func(i, j int) bool {
		pi, pj := rankScore(list[i].Present, list[i].Total), rankScore(list[j].Present, list[j].Total)
		if pi != pj {
			return pi > pj
		}
		return list[i].ClassName < list[j].ClassName
	})
	return &ClassRankingResp{List: list}, nil
}

func rankScore(present, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(present) / float64(total)
}

// --- GET /admin/dashboard/growth-stats ---

type GrowthStatsTopClass struct {
	ClassName   string `json:"class_name"`
	LogsCount   int64  `json:"logs_count"`
	TeacherName string `json:"teacher_name"`
}

type GrowthStatsResp struct {
	Period         string               `json:"period"`
	TotalLogs      int64                `json:"total_logs"`
	TotalMedia     int64                `json:"total_media"`
	TotalLikes     int64                `json:"total_likes"`
	TotalComments  int64                `json:"total_comments"`
	DailyAvgLogs   float64              `json:"daily_avg_logs"`
	TopClasses     []GrowthStatsTopClass `json:"top_classes"`
}

func growthPeriodWindow(period string, loc *time.Location, now time.Time) (from, to time.Time) {
	y2, m2, d2 := now.In(loc).Date()
	to = time.Date(y2, m2, d2, 0, 0, 0, 0, loc).AddDate(0, 0, 1)
	switch period {
	case "7d":
		from = to.AddDate(0, 0, -7)
	case "12m":
		from = to.AddDate(-1, 0, 0)
	default:
		from = to.AddDate(0, 0, -30)
	}
	return from, to
}

func (s *DashboardService) GrowthStats(ctx context.Context, kindergartenID uint64, period string) (*GrowthStatsResp, error) {
	if kindergartenID == 0 {
		return nil, fmt.Errorf("NO_KINDERGARTEN")
	}
	if period != "7d" && period != "30d" && period != "12m" {
		period = "30d"
	}
	loc := shanghaiLoc()
	now := time.Now().In(loc)
	from, to := growthPeriodWindow(period, loc, now)
	days := int(to.Sub(from) / (24 * time.Hour))
	if days < 1 {
		days = 1
	}

	totalLogs, err := s.dash.CountGrowthLogsInRange(ctx, kindergartenID, from, to)
	if err != nil {
		return nil, err
	}
	totalMedia, err := s.dash.CountGrowthMediaInRange(ctx, kindergartenID, from, to)
	if err != nil {
		return nil, err
	}
	totalLikes, err := s.dash.CountGrowthLikesForLogsInRange(ctx, kindergartenID, from, to)
	if err != nil {
		return nil, err
	}
	totalComments, err := s.dash.CountGrowthCommentsForLogsInRange(ctx, kindergartenID, from, to)
	if err != nil {
		return nil, err
	}

	aggs, err := s.dash.GrowthLogsByClass(ctx, kindergartenID, from, to)
	if err != nil {
		return nil, err
	}
	top := make([]GrowthStatsTopClass, 0, len(aggs))
	for _, a := range aggs {
		tname, err := s.dash.FirstTeacherNickname(ctx, a.ClassID)
		if err != nil {
			return nil, err
		}
		top = append(top, GrowthStatsTopClass{
			ClassName:   a.ClassName,
			LogsCount:   a.LogsCount,
			TeacherName: tname,
		})
	}

	return &GrowthStatsResp{
		Period:        period,
		TotalLogs:     totalLogs,
		TotalMedia:    totalMedia,
		TotalLikes:    totalLikes,
		TotalComments: totalComments,
		DailyAvgLogs:  round1f(totalLogs, int64(days)),
		TopClasses:    top,
	}, nil
}
