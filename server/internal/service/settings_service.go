package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

var (
	ErrDuplicateAcademicYear = errors.New("DUPLICATE_ACADEMIC_YEAR")
	ErrInvalidFeverThreshold = errors.New("INVALID_FEVER_THRESHOLD")
)

type SettingsService struct {
	db         *gorm.DB
	kg         *repository.KindergartenRepo
	years      *repository.AcademicYearRepo
	app        *repository.KindergartenAppSettingRepo
	classes    *repository.ClassRepo
	students   *repository.StudentRepo
	userRoles  *repository.UserRoleRepo
	oplog      *repository.OperationLogRepo
}

func NewSettingsService(
	db *gorm.DB,
	kg *repository.KindergartenRepo,
	years *repository.AcademicYearRepo,
	app *repository.KindergartenAppSettingRepo,
	classes *repository.ClassRepo,
	students *repository.StudentRepo,
	userRoles *repository.UserRoleRepo,
	oplog *repository.OperationLogRepo,
) *SettingsService {
	return &SettingsService{
		db: db, kg: kg, years: years, app: app,
		classes: classes, students: students, userRoles: userRoles, oplog: oplog,
	}
}

func (s *SettingsService) logOp(ctx context.Context, kindergartenID, userID uint64, module, action, targetType string, targetID uint64, detail, ip, ua string) {
	_ = s.oplog.Create(ctx, &domain.OperationLog{
		KindergartenID: kindergartenID,
		UserID:         userID,
		Module:         module,
		Action:         action,
		TargetType:     targetType,
		TargetID:       targetID,
		Detail:         detail,
		IP:             ip,
		UserAgent:      truncateUASettings(ua),
	})
}

func truncateUASettings(ua string) string {
	if utf8.RuneCountInString(ua) <= 300 {
		return ua
	}
	return string([]rune(ua)[:300])
}

// --- JSON payload (ExtraConfig) ---

type settingsAttendance struct {
	ArriveStartTime      string `json:"arrive_start_time"`
	ArriveEndTime        string `json:"arrive_end_time"`
	PickupStartTime      string `json:"pickup_start_time"`
	LateThresholdMinutes int    `json:"late_threshold_minutes"`
}

type settingsNotice struct {
	NeedAudit        bool `json:"need_audit"`
	AutoRemindHours  int  `json:"auto_remind_hours"`
}

type settingsGrowth struct {
	NeedAudit          bool `json:"need_audit"`
	AllowParentComment bool `json:"allow_parent_comment"`
	AllowParentLike    bool `json:"allow_parent_like"`
}

type settingsPickup struct {
	QRCodeExpireHours        int  `json:"qr_code_expire_hours"`
	NeedAuthorizationAudit   bool `json:"need_authorization_audit"`
}

type settingsPoints struct {
	TaskCheckinPoints       int `json:"task_checkin_points"`
	WeeklyAttendanceBonus   int `json:"weekly_attendance_bonus"`
	MonthlyAttendanceBonus  int `json:"monthly_attendance_bonus"`
}

type settingsHealth struct {
	TemperatureRequired bool    `json:"temperature_required"`
	FeverThreshold      float64 `json:"fever_threshold"`
}

// settingsAlbum 园所级班级相册配置。ClasswideEnabled 存独立列而不是 ExtraConfig：
// 家长相册服务读的是同一列，保证管理端开关与家长端行为同源（openspec design D1）。
// 零值为 false，即新园所默认关闭。
type settingsAlbum struct {
	ClasswideEnabled bool `json:"classwide_enabled"`
}

type settingsPayload struct {
	Attendance settingsAttendance `json:"attendance"`
	Notice     settingsNotice     `json:"notice"`
	Growth     settingsGrowth     `json:"growth"`
	Pickup     settingsPickup     `json:"pickup"`
	Points     settingsPoints     `json:"points"`
	Health     settingsHealth     `json:"health"`
}

func defaultSettingsPayload() settingsPayload {
	return settingsPayload{
		Attendance: settingsAttendance{
			ArriveStartTime: "07:30", ArriveEndTime: "09:00", PickupStartTime: "16:30", LateThresholdMinutes: 15,
		},
		Notice: settingsNotice{NeedAudit: true, AutoRemindHours: 24},
		Growth: settingsGrowth{NeedAudit: false, AllowParentComment: true, AllowParentLike: true},
		Pickup: settingsPickup{QRCodeExpireHours: 24, NeedAuthorizationAudit: true},
		Points: settingsPoints{TaskCheckinPoints: 5, WeeklyAttendanceBonus: 10, MonthlyAttendanceBonus: 50},
		Health: settingsHealth{TemperatureRequired: false, FeverThreshold: DefaultFeverThreshold},
	}
}

func defaultExtraConfigJSON() string {
	b, _ := json.Marshal(defaultSettingsPayload())
	return string(b)
}

func parsePayload(raw string) settingsPayload {
	if strings.TrimSpace(raw) == "" {
		return defaultSettingsPayload()
	}
	var p settingsPayload
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return defaultSettingsPayload()
	}
	def := defaultSettingsPayload()
	if p.Attendance == (settingsAttendance{}) {
		p.Attendance = def.Attendance
	}
	if p.Notice == (settingsNotice{}) {
		p.Notice = def.Notice
	}
	if p.Growth == (settingsGrowth{}) {
		p.Growth = def.Growth
	}
	if p.Pickup == (settingsPickup{}) {
		p.Pickup = def.Pickup
	}
	if p.Points == (settingsPoints{}) {
		p.Points = def.Points
	}
	if p.Health == (settingsHealth{}) {
		p.Health = def.Health
	} else if p.Health.FeverThreshold < MinFeverThreshold || p.Health.FeverThreshold > MaxFeverThreshold {
		// 保留 temperature_required，只把非法阈值归一化，避免整块回退丢掉开关状态。
		p.Health.FeverThreshold = def.Health.FeverThreshold
	}
	return p
}

func (s *SettingsService) ensureAppSetting(ctx context.Context, kgID uint64) (*domain.KindergartenAppSetting, error) {
	row, err := s.app.FindByKindergartenID(ctx, kgID)
	if err != nil {
		return nil, err
	}
	if row != nil {
		return row, nil
	}
	n := &domain.KindergartenAppSetting{
		KindergartenID:  kgID,
		CurrentSemester: "spring",
		ExtraConfig:     defaultExtraConfigJSON(),
	}
	if err := s.app.Create(ctx, n); err != nil {
		return nil, err
	}
	return n, nil
}

func (s *SettingsService) resolveYearRow(ctx context.Context, kgID uint64) (*domain.AcademicYear, error) {
	cur, err := s.years.FindCurrent(ctx, kgID)
	if err != nil {
		return nil, err
	}
	if cur != nil {
		return cur, nil
	}
	return s.years.FindLatest(ctx, kgID)
}

// --- Public DTOs (handler / JSON) ---

type SystemSettingsResp struct {
	Basic struct {
		AcademicYear  string `json:"academic_year"`
		Semester      string `json:"semester"`
		SemesterStart string `json:"semester_start"`
		SemesterEnd   string `json:"semester_end"`
	} `json:"basic"`
	Attendance settingsAttendance `json:"attendance"`
	Notice     settingsNotice     `json:"notice"`
	Growth     settingsGrowth     `json:"growth"`
	Pickup     settingsPickup     `json:"pickup"`
	Points     settingsPoints     `json:"points"`
	Health     settingsHealth     `json:"health"`
	Album      settingsAlbum      `json:"album"`
}

func (s *SettingsService) GetSettings(ctx context.Context, kindergartenID uint64) (*SystemSettingsResp, error) {
	if kindergartenID == 0 {
		return nil, fmt.Errorf("NO_KINDERGARTEN")
	}
	appRow, err := s.ensureAppSetting(ctx, kindergartenID)
	if err != nil {
		return nil, err
	}
	payload := parsePayload(appRow.ExtraConfig)
	sem := strings.ToLower(strings.TrimSpace(appRow.CurrentSemester))
	if sem != "fall" {
		sem = "spring"
	}

	yr, err := s.resolveYearRow(ctx, kindergartenID)
	if err != nil {
		return nil, err
	}
	var out SystemSettingsResp
	if yr != nil {
		out.Basic.AcademicYear = yr.Year
	}
	out.Basic.Semester = sem
	if yr != nil {
		switch sem {
		case "fall":
			out.Basic.SemesterStart = yr.SemesterFallStart
			out.Basic.SemesterEnd = yr.SemesterFallEnd
		default:
			out.Basic.SemesterStart = yr.SemesterSpringStart
			out.Basic.SemesterEnd = yr.SemesterSpringEnd
		}
	}
	out.Attendance = payload.Attendance
	out.Notice = payload.Notice
	out.Growth = payload.Growth
	out.Pickup = payload.Pickup
	out.Points = payload.Points
	out.Health = payload.Health
	out.Album = settingsAlbum{ClasswideEnabled: appRow.ClassAlbumClasswideEnabled}
	return &out, nil
}

// GetHealthPolicy 只读园所晨检体温策略；配置行不存在时返回默认值，不在读取路径写库。
func (s *SettingsService) GetHealthPolicy(ctx context.Context, kindergartenID uint64) (HealthPolicy, error) {
	if kindergartenID == 0 {
		return HealthPolicy{}, fmt.Errorf("NO_KINDERGARTEN")
	}
	row, err := s.app.FindByKindergartenID(ctx, kindergartenID)
	if err != nil {
		return HealthPolicy{}, err
	}
	p := defaultSettingsPayload()
	if row != nil {
		p = parsePayload(row.ExtraConfig)
	}
	return HealthPolicy{
		TemperatureRequired: p.Health.TemperatureRequired,
		FeverThreshold:      p.Health.FeverThreshold,
	}, nil
}

type UpdateSettingsReq struct {
	Attendance *settingsAttendance `json:"attendance"`
	Notice       *settingsNotice     `json:"notice"`
	Growth       *settingsGrowth     `json:"growth"`
	Pickup       *settingsPickup     `json:"pickup"`
	Points       *settingsPoints     `json:"points"`
	Health       *settingsHealth     `json:"health"`
	Album        *settingsAlbum      `json:"album"`
}

type UpdateSettingsResp struct {
	UpdatedAt string `json:"updated_at"`
}

func (s *SettingsService) UpdateSettings(ctx context.Context, kindergartenID, userID uint64, req UpdateSettingsReq, ip, ua string) (*UpdateSettingsResp, error) {
	if kindergartenID == 0 {
		return nil, fmt.Errorf("NO_KINDERGARTEN")
	}
	appRow, err := s.ensureAppSetting(ctx, kindergartenID)
	if err != nil {
		return nil, err
	}
	cur := parsePayload(appRow.ExtraConfig)
	if req.Attendance != nil {
		cur.Attendance = *req.Attendance
	}
	if req.Notice != nil {
		cur.Notice = *req.Notice
	}
	if req.Growth != nil {
		cur.Growth = *req.Growth
	}
	if req.Pickup != nil {
		cur.Pickup = *req.Pickup
	}
	if req.Points != nil {
		cur.Points = *req.Points
	}
	if req.Health != nil {
		if req.Health.FeverThreshold < MinFeverThreshold || req.Health.FeverThreshold > MaxFeverThreshold {
			return nil, ErrInvalidFeverThreshold
		}
		cur.Health = *req.Health
	}
	if req.Album != nil {
		appRow.ClassAlbumClasswideEnabled = req.Album.ClasswideEnabled
	}
	b, err := json.Marshal(cur)
	if err != nil {
		return nil, err
	}
	appRow.ExtraConfig = string(b)
	if err := s.app.Save(ctx, appRow); err != nil {
		return nil, err
	}
	refreshed, err := s.app.FindByKindergartenID(ctx, kindergartenID)
	if err != nil {
		return nil, err
	}
	if refreshed == nil {
		return nil, fmt.Errorf("NOT_FOUND")
	}
	// 审计同时记录 ExtraConfig 与独立列开关：只写 ExtraConfig 会让「仅改开关」
	// 的变更在日志里没有痕迹。
	detail, err := json.Marshal(struct {
		ExtraConfig json.RawMessage `json:"extra_config"`
		Album       settingsAlbum   `json:"album"`
	}{ExtraConfig: b, Album: settingsAlbum{ClasswideEnabled: appRow.ClassAlbumClasswideEnabled}})
	if err != nil {
		return nil, err
	}
	s.logOp(ctx, kindergartenID, userID, "settings", "update_settings", "kindergarten", kindergartenID,
		string(detail), ip, ua)
	return &UpdateSettingsResp{UpdatedAt: refreshed.UpdatedAt.UTC().Format(time.RFC3339)}, nil
}

type AcademicYearListItem struct {
	Year      string `json:"year"`
	IsCurrent bool   `json:"is_current"`
}

type AcademicYearsResp struct {
	List []AcademicYearListItem `json:"list"`
}

func (s *SettingsService) ListAcademicYears(ctx context.Context, kindergartenID uint64) (*AcademicYearsResp, error) {
	if kindergartenID == 0 {
		return nil, fmt.Errorf("NO_KINDERGARTEN")
	}
	rows, err := s.years.ListByKindergarten(ctx, kindergartenID)
	if err != nil {
		return nil, err
	}
	out := AcademicYearsResp{List: make([]AcademicYearListItem, 0, len(rows))}
	for _, r := range rows {
		out.List = append(out.List, AcademicYearListItem{Year: r.Year, IsCurrent: r.IsCurrent})
	}
	return &out, nil
}

type CreateAcademicYearReq struct {
	Year                string `json:"year"`
	SemesterSpringStart string `json:"semester_spring_start"`
	SemesterSpringEnd   string `json:"semester_spring_end"`
	SemesterFallStart   string `json:"semester_fall_start"`
	SemesterFallEnd     string `json:"semester_fall_end"`
}

type CreateAcademicYearResp struct {
	Year      string `json:"year"`
	CreatedAt string `json:"created_at"`
}

func parseYMD(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("EMPTY_DATE")
	}
	if _, err := time.Parse("2006-01-02", s); err != nil {
		return fmt.Errorf("INVALID_DATE")
	}
	return nil
}

func (s *SettingsService) CreateAcademicYear(ctx context.Context, kindergartenID, userID uint64, req CreateAcademicYearReq, ip, ua string) (*CreateAcademicYearResp, error) {
	if kindergartenID == 0 {
		return nil, fmt.Errorf("NO_KINDERGARTEN")
	}
	year := strings.TrimSpace(req.Year)
	if year == "" {
		return nil, fmt.Errorf("INVALID_YEAR")
	}
	for _, d := range []string{req.SemesterSpringStart, req.SemesterSpringEnd, req.SemesterFallStart, req.SemesterFallEnd} {
		if err := parseYMD(d); err != nil {
			return nil, err
		}
	}
	dup, err := s.years.FindByYear(ctx, kindergartenID, year)
	if err != nil {
		return nil, err
	}
	if dup != nil {
		return nil, ErrDuplicateAcademicYear
	}
	n, err := s.years.CountByKindergarten(ctx, kindergartenID)
	if err != nil {
		return nil, err
	}
	row := &domain.AcademicYear{
		KindergartenID:      kindergartenID,
		Year:                year,
		SemesterSpringStart: strings.TrimSpace(req.SemesterSpringStart),
		SemesterSpringEnd:   strings.TrimSpace(req.SemesterSpringEnd),
		SemesterFallStart:   strings.TrimSpace(req.SemesterFallStart),
		SemesterFallEnd:     strings.TrimSpace(req.SemesterFallEnd),
		IsCurrent:           n == 0,
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if row.IsCurrent {
			if err := tx.Model(&domain.AcademicYear{}).
				Where("kindergarten_id = ?", kindergartenID).
				Update("is_current", false).Error; err != nil {
				return err
			}
		}
		return tx.Create(row).Error
	}); err != nil {
		return nil, err
	}
	// reload for CreatedAt
	created, err := s.years.FindByYear(ctx, kindergartenID, year)
	if err != nil || created == nil {
		return &CreateAcademicYearResp{Year: year, CreatedAt: time.Now().UTC().Format(time.RFC3339)}, nil
	}
	detail, _ := json.Marshal(req)
	s.logOp(ctx, kindergartenID, userID, "settings", "create_academic_year", "academic_year", created.ID,
		string(detail), ip, ua)
	return &CreateAcademicYearResp{
		Year:      year,
		CreatedAt: created.CreatedAt.UTC().Format(time.RFC3339),
	}, nil
}

type KindergartenInfoResp struct {
	ID             uint64 `json:"id"`
	Name           string `json:"name"`
	Address        string `json:"address"`
	Phone          string `json:"phone"`
	Principal      string `json:"principal"`
	LogoURL        string `json:"logo_url"`
	LicenseNo      string `json:"license_no"`
	Capacity       int    `json:"capacity"`
	CurrentCount   int64  `json:"current_count"`
	ClassCount     int64  `json:"class_count"`
	TeacherCount   int64  `json:"teacher_count"`
	EstablishedAt  string `json:"established_at"`
	Description    string `json:"description"`
}

func (s *SettingsService) GetKindergartenInfo(ctx context.Context, kindergartenID uint64) (*KindergartenInfoResp, error) {
	if kindergartenID == 0 {
		return nil, fmt.Errorf("NO_KINDERGARTEN")
	}
	k, err := s.kg.FindByID(ctx, kindergartenID)
	if err != nil {
		return nil, err
	}
	if k == nil {
		return nil, fmt.Errorf("NOT_FOUND")
	}
	stu, err := s.students.CountActiveInKindergarten(ctx, kindergartenID)
	if err != nil {
		return nil, err
	}
	cls, err := s.classes.CountInKindergarten(ctx, kindergartenID)
	if err != nil {
		return nil, err
	}
	tch, err := s.userRoles.CountByRoleInKindergarten(ctx, "teacher", kindergartenID)
	if err != nil {
		return nil, err
	}
	est := k.EstablishedAt
	return &KindergartenInfoResp{
		ID:            k.ID,
		Name:          k.Name,
		Address:       k.Address,
		Phone:         k.Phone,
		Principal:     k.Principal,
		LogoURL:       k.LogoURL,
		LicenseNo:     k.LicenseNo,
		Capacity:      k.Capacity,
		CurrentCount:  stu,
		ClassCount:    cls,
		TeacherCount:  tch,
		EstablishedAt: est,
		Description:   k.Description,
	}, nil
}

type UpdateKindergartenInfoReq struct {
	Name          *string `json:"name"`
	Address       *string `json:"address"`
	Phone         *string `json:"phone"`
	Principal     *string `json:"principal"`
	LogoURL       *string `json:"logo_url"`
	LicenseNo     *string `json:"license_no"`
	Capacity      *int    `json:"capacity"`
	EstablishedAt *string `json:"established_at"`
	Description   *string `json:"description"`
}

type UpdateKindergartenInfoResp struct {
	ID        uint64 `json:"id"`
	UpdatedAt string `json:"updated_at"`
}

func (s *SettingsService) UpdateKindergartenInfo(ctx context.Context, kindergartenID, userID uint64, req UpdateKindergartenInfoReq, ip, ua string) (*UpdateKindergartenInfoResp, error) {
	if kindergartenID == 0 {
		return nil, fmt.Errorf("NO_KINDERGARTEN")
	}
	k, err := s.kg.FindByID(ctx, kindergartenID)
	if err != nil {
		return nil, err
	}
	if k == nil {
		return nil, fmt.Errorf("NOT_FOUND")
	}
	cols := map[string]any{}
	if req.Name != nil {
		cols["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Address != nil {
		cols["address"] = strings.TrimSpace(*req.Address)
	}
	if req.Phone != nil {
		cols["phone"] = strings.TrimSpace(*req.Phone)
	}
	if req.Principal != nil {
		cols["principal"] = strings.TrimSpace(*req.Principal)
	}
	if req.LogoURL != nil {
		cols["logo_url"] = strings.TrimSpace(*req.LogoURL)
	}
	if req.LicenseNo != nil {
		cols["license_no"] = strings.TrimSpace(*req.LicenseNo)
	}
	if req.Capacity != nil {
		cols["capacity"] = *req.Capacity
	}
	if req.EstablishedAt != nil {
		v := strings.TrimSpace(*req.EstablishedAt)
		if v != "" {
			if err := parseYMD(v); err != nil {
				return nil, err
			}
		}
		cols["established_at"] = v
	}
	if req.Description != nil {
		cols["description"] = *req.Description
	}
	if len(cols) == 0 {
		return &UpdateKindergartenInfoResp{
			ID:        k.ID,
			UpdatedAt: k.UpdatedAt.UTC().Format(time.RFC3339),
		}, nil
	}
	if err := s.kg.UpdateByID(ctx, kindergartenID, cols); err != nil {
		return nil, err
	}
	k2, err := s.kg.FindByID(ctx, kindergartenID)
	if err != nil {
		return nil, err
	}
	if k2 == nil {
		return nil, fmt.Errorf("NOT_FOUND")
	}
	detail, _ := json.Marshal(cols)
	s.logOp(ctx, kindergartenID, userID, "kindergarten", "update_info", "kindergarten", kindergartenID,
		string(detail), ip, ua)
	return &UpdateKindergartenInfoResp{
		ID:        k2.ID,
		UpdatedAt: k2.UpdatedAt.UTC().Format(time.RFC3339),
	}, nil
}

func IsSettingsClientErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrDuplicateAcademicYear) || errors.Is(err, ErrInvalidFeverThreshold) {
		return true
	}
	switch err.Error() {
	case "INVALID_YEAR", "INVALID_DATE", "EMPTY_DATE":
		return true
	default:
		return false
	}
}
