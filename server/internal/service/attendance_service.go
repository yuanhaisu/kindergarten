package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/pkg/pagination"
)

type AttendanceService struct {
	att     *repository.AttendanceRepo
	classes *repository.ClassRepo
	dash    *repository.DashboardRepo
}

func NewAttendanceService(
	att *repository.AttendanceRepo,
	classes *repository.ClassRepo,
	dash *repository.DashboardRepo,
) *AttendanceService {
	return &AttendanceService{att: att, classes: classes, dash: dash}
}

func recentWorkdays(day time.Time, n int) []time.Time {
	loc := shanghaiLoc()
	out := make([]time.Time, 0, n)
	cur := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc)
	for len(out) < n {
		wd := cur.Weekday()
		if wd != time.Saturday && wd != time.Sunday {
			out = append(out, cur)
		}
		cur = cur.AddDate(0, 0, -1)
	}
	return out
}

func synthAttendanceStatus(dayIndex, studentIndex int) domain.AttendanceStatus {
	// Keep deterministic distribution for stable test data.
	n := (dayIndex + studentIndex) % 10
	switch {
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

func synthCheckIn(day time.Time, studentIndex int, status domain.AttendanceStatus) *time.Time {
	if status != domain.AttendancePresent && status != domain.AttendanceLate {
		return nil
	}
	loc := shanghaiLoc()
	minute := (studentIndex * 7) % 30
	hour := 8
	if status == domain.AttendanceLate {
		hour = 9
	}
	t := time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, loc)
	return &t
}

func (s *AttendanceService) ensureBootstrapData(ctx context.Context, kindergartenID uint64, anchorDay time.Time) error {
	loc := shanghaiLoc()
	from, toExclusive := dayRange(loc, anchorDay)
	total, err := s.att.CountByDateRange(ctx, kindergartenID, from, toExclusive)
	if err != nil {
		return err
	}
	if total > 0 {
		return nil
	}
	students, err := s.att.ListActiveStudents(ctx, kindergartenID)
	if err != nil {
		return err
	}
	if len(students) == 0 {
		return nil
	}
	days := recentWorkdays(anchorDay, 3)
	rows := make([]domain.Attendance, 0, len(days)*len(students))
	for di, d := range days {
		for si, st := range students {
			status := synthAttendanceStatus(di, si)
			rows = append(rows, domain.Attendance{
				KindergartenID: kindergartenID,
				ClassID:        st.ClassID,
				StudentID:      st.ID,
				Date:           d,
				Status:         status,
				CheckInTime:    synthCheckIn(d, si, status),
				Remark:         "",
				RecorderID:     0,
			})
		}
	}
	return s.att.UpsertBatch(ctx, rows)
}

func parseAdminDate(dateStr string) (time.Time, error) {
	s := strings.TrimSpace(dateStr)
	if s == "" {
		return time.Now().In(shanghaiLoc()), nil
	}
	t, err := time.ParseInLocation("2006-01-02", s, shanghaiLoc())
	if err != nil {
		return time.Time{}, fmt.Errorf("BAD_DATE")
	}
	return t, nil
}

func dateRangeInclusive(loc *time.Location, startDay, endDay time.Time) (from, toExclusive time.Time) {
	sy, sm, sd := startDay.In(loc).Date()
	ey, em, ed := endDay.In(loc).Date()
	from = time.Date(sy, sm, sd, 0, 0, 0, 0, loc)
	endDayOnly := time.Date(ey, em, ed, 0, 0, 0, 0, loc)
	toExclusive = endDayOnly.AddDate(0, 0, 1)
	return from, toExclusive
}

type attendanceOverviewClass struct {
	ClassID   uint64 `json:"class_id"`
	ClassName string `json:"class_name"`
	Total     int64  `json:"total"`
	Present   int64  `json:"present"`
	Leave     int64  `json:"leave"`
	Absent    int64  `json:"absent"`
	Rate      string `json:"rate"`
}

type attendanceOverviewData struct {
	Date           string                    `json:"date"`
	TotalStudents  int64                     `json:"total_students"`
	Present        int64                     `json:"present"`
	Leave          int64                     `json:"leave"`
	Absent         int64                     `json:"absent"`
	AttendanceRate string                    `json:"attendance_rate"`
	ByClass        []attendanceOverviewClass `json:"by_class"`
}

func (s *AttendanceService) Overview(ctx context.Context, kindergartenID uint64, dateStr string) (*attendanceOverviewData, error) {
	if kindergartenID == 0 {
		return nil, fmt.Errorf("NO_KINDERGARTEN")
	}
	day, err := parseAdminDate(dateStr)
	if err != nil {
		return nil, err
	}
	if err := s.ensureBootstrapData(ctx, kindergartenID, day); err != nil {
		return nil, err
	}
	loc := shanghaiLoc()
	start, end := dayRange(loc, day)
	dateOut := day.In(loc).Format("2006-01-02")

	totalStudents, err := s.dash.CountActiveStudents(ctx, kindergartenID)
	if err != nil {
		return nil, err
	}

	classList, _, err := s.classes.List(ctx, kindergartenID, "", "", 0, 5000)
	if err != nil {
		return nil, err
	}
	classIDs := make([]uint64, 0, len(classList))
	for _, c := range classList {
		classIDs = append(classIDs, c.ID)
	}
	countsByClass, err := s.classes.CountStudentsByClassIDs(ctx, kindergartenID, classIDs)
	if err != nil {
		return nil, err
	}

	agg, err := s.att.AggregateByClassStatus(ctx, kindergartenID, start, end)
	if err != nil {
		return nil, err
	}
	statusByClass := make(map[uint64]map[domain.AttendanceStatus]int64)
	for _, row := range agg {
		if statusByClass[row.ClassID] == nil {
			statusByClass[row.ClassID] = make(map[domain.AttendanceStatus]int64)
		}
		statusByClass[row.ClassID][row.Status] += row.Cnt
	}

	byClass := make([]attendanceOverviewClass, 0, len(classList))
	var sumPresent, sumLeave int64
	for _, c := range classList {
		total := countsByClass[c.ID]
		st := statusByClass[c.ID]
		var present int64
		if st != nil {
			present = st[domain.AttendancePresent] + st[domain.AttendanceLate]
		}
		var leave int64
		if st != nil {
			leave = st[domain.AttendanceLeave]
		}
		absent := total - present - leave
		if absent < 0 {
			absent = 0
		}
		sumPresent += present
		sumLeave += leave
		byClass = append(byClass, attendanceOverviewClass{
			ClassID:   c.ID,
			ClassName: c.Name,
			Total:     total,
			Present:   present,
			Leave:     leave,
			Absent:    absent,
			Rate:      pct1(present, total),
		})
	}

	topAbsent := totalStudents - sumPresent - sumLeave
	if topAbsent < 0 {
		topAbsent = 0
	}

	return &attendanceOverviewData{
		Date:           dateOut,
		TotalStudents:  totalStudents,
		Present:        sumPresent,
		Leave:          sumLeave,
		Absent:         topAbsent,
		AttendanceRate: pct1(sumPresent, totalStudents),
		ByClass:        byClass,
	}, nil
}

type attendanceRecordStudent struct {
	ID        uint64 `json:"id"`
	Name      string `json:"name"`
	Avatar    string `json:"avatar"`
	ClassName string `json:"class_name"`
}

type attendanceRecordItem struct {
	ID          uint64                  `json:"id"`
	Student     attendanceRecordStudent `json:"student"`
	Date        string                  `json:"date"`
	Status      string                  `json:"status"`
	StatusLabel string                  `json:"status_label"`
	ArriveTime  string                  `json:"arrive_time"`
	PickupTime  string                  `json:"pickup_time"`
	Remark      string                  `json:"remark"`
}

func attendanceStatusAPI(st domain.AttendanceStatus) (key, label string) {
	switch st {
	case domain.AttendancePresent:
		return "present", "出勤"
	case domain.AttendanceAbsent:
		return "absent", "缺勤"
	case domain.AttendanceLeave:
		return "leave", "请假"
	case domain.AttendanceLate:
		return "present", "迟到"
	default:
		return "unknown", "未知"
	}
}

func formatTimeHM(t *time.Time, loc *time.Location) string {
	if t == nil {
		return ""
	}
	return t.In(loc).Format("15:04")
}

func (s *AttendanceService) Records(
	ctx context.Context,
	kindergartenID uint64,
	classID *uint64,
	dateStr, statusFilter string,
	q pagination.Query,
) ([]attendanceRecordItem, int64, int, int, error) {
	if kindergartenID == 0 {
		return nil, 0, 0, 0, fmt.Errorf("NO_KINDERGARTEN")
	}
	day, err := parseAdminDate(dateStr)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	if err := s.ensureBootstrapData(ctx, kindergartenID, day); err != nil {
		return nil, 0, 0, 0, err
	}
	loc := shanghaiLoc()
	start, end := dayRange(loc, day)
	q.Normalize()
	st := strings.TrimSpace(strings.ToLower(statusFilter))
	switch st {
	case "", "present", "leave", "absent":
	default:
		return nil, 0, 0, 0, fmt.Errorf("BAD_STATUS")
	}

	rows, total, err := s.att.ListRecords(ctx, kindergartenID, classID, start, end, st, q.Offset(), q.PageSize)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	out := make([]attendanceRecordItem, 0, len(rows))
	for _, r := range rows {
		apiKey, label := attendanceStatusAPI(r.Status)
		out = append(out, attendanceRecordItem{
			ID: r.ID,
			Student: attendanceRecordStudent{
				ID:        r.StudentID,
				Name:      r.StudentName,
				Avatar:    r.StudentAvatar,
				ClassName: r.ClassName,
			},
			Date:        r.Date.In(loc).Format("2006-01-02"),
			Status:      apiKey,
			StatusLabel: label,
			ArriveTime:  formatTimeHM(r.CheckInTime, loc),
			PickupTime:  "",
			Remark:      r.Remark,
		})
	}
	if out == nil {
		out = []attendanceRecordItem{}
	}
	return out, total, q.Page, q.PageSize, nil
}

func (s *AttendanceService) ExportReport(
	ctx context.Context,
	kindergartenID uint64,
	classID *uint64,
	startStr, endStr, format string,
) ([]byte, string, error) {
	if kindergartenID == 0 {
		return nil, "", fmt.Errorf("NO_KINDERGARTEN")
	}
	if format != "" && format != "xlsx" {
		return nil, "", fmt.Errorf("BAD_FORMAT")
	}
	loc := shanghaiLoc()
	startDay, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(startStr), loc)
	if err != nil {
		return nil, "", fmt.Errorf("BAD_DATE")
	}
	endDay, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(endStr), loc)
	if err != nil {
		return nil, "", fmt.Errorf("BAD_DATE")
	}
	if endDay.Before(startDay) {
		return nil, "", fmt.Errorf("BAD_DATE_RANGE")
	}
	if err := s.ensureBootstrapData(ctx, kindergartenID, endDay); err != nil {
		return nil, "", err
	}
	from, toExc := dateRangeInclusive(loc, startDay, endDay)

	rows, err := s.att.ListForExport(ctx, kindergartenID, classID, from, toExc)
	if err != nil {
		return nil, "", err
	}

	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	headers := []string{"日期", "班级", "学生", "状态", "到园时间", "备注"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}
	for i, row := range rows {
		r := i + 2
		_, lbl := attendanceStatusAPI(row.Status)
		d := row.Date.In(loc).Format("2006-01-02")
		arrive := formatTimeHM(row.CheckInTime, loc)
		for col, v := range []interface{}{d, row.ClassName, row.StudentName, lbl, arrive, row.Remark} {
			cell, _ := excelize.CoordinatesToCellName(col+1, r)
			_ = f.SetCellValue(sheet, cell, v)
		}
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, "", err
	}
	fn := fmt.Sprintf("attendance_report_%s_%s.xlsx",
		startDay.Format("20060102"),
		endDay.Format("20060102"),
	)
	return buf.Bytes(), fn, nil
}

func IsAttendanceClientErr(err error) bool {
	if err == nil {
		return false
	}
	switch err.Error() {
	case "BAD_DATE", "BAD_STATUS", "BAD_FORMAT", "BAD_DATE_RANGE":
		return true
	default:
		return false
	}
}

// ValidateClassInKindergarten 校验班级归属（导出/列表可选）。
func (s *AttendanceService) ValidateClassInKindergarten(ctx context.Context, kindergartenID, classID uint64) error {
	c, err := s.classes.FindByID(ctx, kindergartenID, classID)
	if err != nil {
		return err
	}
	if c == nil {
		return fmt.Errorf("CLASS_NOT_FOUND")
	}
	return nil
}
