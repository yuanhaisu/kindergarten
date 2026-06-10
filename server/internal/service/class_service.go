package service

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/pkg/pagination"
)

type ClassService struct {
	classRepo *repository.ClassRepo
	userRepo  *repository.UserRepo
	oplog     *repository.OperationLogRepo
}

func NewClassService(
	classRepo *repository.ClassRepo,
	userRepo *repository.UserRepo,
	oplog *repository.OperationLogRepo,
) *ClassService {
	return &ClassService{classRepo: classRepo, userRepo: userRepo, oplog: oplog}
}

func (s *ClassService) logOp(
	ctx context.Context,
	kindergartenID, userID uint64,
	module, action, targetType string,
	targetID uint64,
	detail, ip, ua string,
) {
	_ = s.oplog.Create(ctx, &domain.OperationLog{
		KindergartenID: kindergartenID,
		UserID:         userID,
		Module:         module,
		Action:         action,
		TargetType:     targetType,
		TargetID:       targetID,
		Detail:         detail,
		IP:             ip,
		UserAgent:      truncateUA(ua),
	})
}

func truncateUA(ua string) string {
	if utf8.RuneCountInString(ua) <= 300 {
		return ua
	}
	return string([]rune(ua)[:300])
}

func titleLabel(title domain.TeacherTitle) string {
	switch title {
	case domain.TeacherTitleHead:
		return "主班老师"
	case domain.TeacherTitleAssistant:
		return "副班老师"
	case domain.TeacherTitleNursery:
		return "生活老师"
	default:
		return string(title)
	}
}

func defaultAcademicYear(t time.Time) string {
	y := t.Year()
	m := t.Month()
	if m < 9 {
		return fmt.Sprintf("%d-%d", y-1, y)
	}
	return fmt.Sprintf("%d-%d", y, y+1)
}

func parseTeacherTitle(s string) (domain.TeacherTitle, error) {
	switch domain.TeacherTitle(s) {
	case domain.TeacherTitleHead, domain.TeacherTitleAssistant, domain.TeacherTitleNursery:
		return domain.TeacherTitle(s), nil
	default:
		return "", fmt.Errorf("INVALID_TEACHER_TITLE")
	}
}

func validateGrade(grade string) error {
	switch grade {
	case "小班", "中班", "大班":
		return nil
	default:
		return fmt.Errorf("INVALID_GRADE")
	}
}

// --- list / detail DTOs (JSON 与 admin 规格一致) ---

type classTeacherBrief struct {
	ID         uint64 `json:"id"`
	Nickname   string `json:"nickname"`
	Title      string `json:"title"`
	TitleLabel string `json:"title_label"`
}

type classTeacherDetail struct {
	ID         uint64 `json:"id"`
	Nickname   string `json:"nickname"`
	Avatar     string `json:"avatar"`
	Title      string `json:"title"`
	TitleLabel string `json:"title_label"`
	Phone      string `json:"phone"`
}

type classListItem struct {
	ID            uint64             `json:"id"`
	Name          string             `json:"name"`
	Grade         string             `json:"grade"`
	AcademicYear  string             `json:"academic_year"`
	Capacity      int                `json:"capacity"`
	StudentCount  int64              `json:"student_count"`
	Teachers      []classTeacherBrief `json:"teachers"`
	Status        int8               `json:"status"`
	CreatedAt     string             `json:"created_at"`
}

type studentPreview struct {
	ID     uint64 `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type classDetailResp struct {
	ID               uint64               `json:"id"`
	Name             string               `json:"name"`
	Grade            string               `json:"grade"`
	AcademicYear     string               `json:"academic_year"`
	Capacity         int                  `json:"capacity"`
	StudentCount     int64                `json:"student_count"`
	Status           int8                 `json:"status"`
	Teachers         []classTeacherDetail `json:"teachers"`
	StudentsPreview  []studentPreview     `json:"students_preview"`
	CreatedAt        string               `json:"created_at"`
}

func (s *ClassService) buildTeacherBriefs(rows []domain.ClassTeacher) []classTeacherBrief {
	out := make([]classTeacherBrief, 0, len(rows))
	for _, ct := range rows {
		nick := ""
		if ct.User != nil {
			nick = ct.User.Nickname
		}
		out = append(out, classTeacherBrief{
			ID:         ct.UserID,
			Nickname:   nick,
			Title:      string(ct.Title),
			TitleLabel: titleLabel(ct.Title),
		})
	}
	return out
}

func (s *ClassService) ListClasses(
	ctx context.Context,
	kindergartenID uint64,
	q pagination.Query,
	grade string,
) ([]classListItem, int64, int, int, error) {
	if kindergartenID == 0 {
		return nil, 0, 0, 0, fmt.Errorf("NO_KINDERGARTEN")
	}
	q.Normalize()
	offset := q.Offset()
	classes, total, err := s.classRepo.List(ctx, kindergartenID, grade, q.Keyword, offset, q.PageSize)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	ids := make([]uint64, 0, len(classes))
	for _, c := range classes {
		ids = append(ids, c.ID)
	}
	counts, err := s.classRepo.CountStudentsByClassIDs(ctx, kindergartenID, ids)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	ctRows, err := s.classRepo.ListTeachersByClassIDs(ctx, ids)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	byClass := map[uint64][]domain.ClassTeacher{}
	for _, ct := range ctRows {
		byClass[ct.ClassID] = append(byClass[ct.ClassID], ct)
	}
	list := make([]classListItem, 0, len(classes))
	for _, c := range classes {
		list = append(list, classListItem{
			ID:            c.ID,
			Name:          c.Name,
			Grade:         c.Grade,
			AcademicYear:  c.AcademicYear,
			Capacity:      c.Capacity,
			StudentCount:  counts[c.ID],
			Teachers:      s.buildTeacherBriefs(byClass[c.ID]),
			Status:        c.Status,
			CreatedAt:     c.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return list, total, q.Page, q.PageSize, nil
}

func (s *ClassService) GetClassDetail(ctx context.Context, kindergartenID, classID uint64) (*classDetailResp, error) {
	if kindergartenID == 0 {
		return nil, fmt.Errorf("NO_KINDERGARTEN")
	}
	c, err := s.classRepo.FindByID(ctx, kindergartenID, classID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("NOT_FOUND")
	}
	n, err := s.classRepo.CountStudentsInClass(ctx, kindergartenID, classID)
	if err != nil {
		return nil, err
	}
	ctRows, err := s.classRepo.ListTeachersByClassIDs(ctx, []uint64{classID})
	if err != nil {
		return nil, err
	}
	tch := make([]classTeacherDetail, 0, len(ctRows))
	for _, ct := range ctRows {
		var av, ph string
		if ct.User != nil {
			ph = maskPhone(ct.User.Phone)
			av = ct.User.Avatar
		}
		tch = append(tch, classTeacherDetail{
			ID:         ct.UserID,
			Nickname:   derefNick(ct.User),
			Avatar:     av,
			Title:      string(ct.Title),
			TitleLabel: titleLabel(ct.Title),
			Phone:      ph,
		})
	}
	students, err := s.classRepo.ListStudentsPreview(ctx, kindergartenID, classID, 10)
	if err != nil {
		return nil, err
	}
	prev := make([]studentPreview, 0, len(students))
	for _, st := range students {
		prev = append(prev, studentPreview{ID: st.ID, Name: st.Name, Avatar: st.Avatar})
	}
	return &classDetailResp{
		ID:              c.ID,
		Name:            c.Name,
		Grade:           c.Grade,
		AcademicYear:    c.AcademicYear,
		Capacity:        c.Capacity,
		StudentCount:    n,
		Status:          c.Status,
		Teachers:        tch,
		StudentsPreview: prev,
		CreatedAt:       c.CreatedAt.UTC().Format(time.RFC3339),
	}, nil
}

func derefNick(u *domain.User) string {
	if u == nil {
		return ""
	}
	return u.Nickname
}

// --- create ---

type classTeacherAssign struct {
	UserID uint64 `json:"user_id" binding:"required"`
	Title  string `json:"title" binding:"required"`
}

type CreateClassReq struct {
	Name           string               `json:"name" binding:"required,max=64"`
	Grade          string               `json:"grade" binding:"required"`
	AcademicYear   string               `json:"academic_year"`
	Capacity       int                  `json:"capacity" binding:"required,min=1"`
	TeacherIDs     []classTeacherAssign `json:"teacher_ids"`
}

type createClassResp struct {
	ID        uint64 `json:"id"`
	CreatedAt string `json:"created_at"`
}

func (s *ClassService) CreateClass(
	ctx context.Context,
	kindergartenID, userID uint64,
	req CreateClassReq,
	ip, ua string,
) (*createClassResp, error) {
	if kindergartenID == 0 {
		return nil, fmt.Errorf("NO_KINDERGARTEN")
	}
	if err := validateGrade(req.Grade); err != nil {
		return nil, err
	}
	ay := strings.TrimSpace(req.AcademicYear)
	if ay == "" {
		ay = defaultAcademicYear(time.Now())
	}
	teachers, err := s.buildClassTeachers(ctx, req.TeacherIDs)
	if err != nil {
		return nil, err
	}
	c := &domain.Class{
		KindergartenID: kindergartenID,
		Name:           req.Name,
		Grade:          req.Grade,
		AcademicYear:   ay,
		Capacity:       req.Capacity,
		Status:         1,
	}
	if err := s.classRepo.CreateClassWithTeachers(ctx, c, teachers); err != nil {
		return nil, err
	}
	s.logOp(ctx, kindergartenID, userID, "classes", "create", "class", c.ID,
		fmt.Sprintf(`{"name":%q}`, req.Name), ip, ua)
	return &createClassResp{ID: c.ID, CreatedAt: c.CreatedAt.UTC().Format(time.RFC3339)}, nil
}

func (s *ClassService) buildClassTeachers(ctx context.Context, rows []classTeacherAssign) ([]domain.ClassTeacher, error) {
	seen := map[uint64]struct{}{}
	out := make([]domain.ClassTeacher, 0, len(rows))
	for _, r := range rows {
		if _, ok := seen[r.UserID]; ok {
			return nil, fmt.Errorf("DUPLICATE_TEACHER")
		}
		seen[r.UserID] = struct{}{}
		u, err := s.userRepo.FindByID(ctx, r.UserID)
		if err != nil {
			return nil, err
		}
		if u == nil {
			return nil, fmt.Errorf("USER_NOT_FOUND")
		}
		title, err := parseTeacherTitle(r.Title)
		if err != nil {
			return nil, err
		}
		out = append(out, domain.ClassTeacher{
			UserID: r.UserID,
			Title:  title,
			Status: 1,
		})
	}
	return out, nil
}

// --- update ---

type UpdateClassReq struct {
	Name       string               `json:"name" binding:"required,max=64"`
	Grade      string               `json:"grade" binding:"required"`
	Capacity   int                  `json:"capacity" binding:"required,min=1"`
	TeacherIDs []classTeacherAssign `json:"teacher_ids"`
}

type updateClassResp struct {
	ID        uint64 `json:"id"`
	UpdatedAt string `json:"updated_at"`
}

func (s *ClassService) UpdateClass(
	ctx context.Context,
	kindergartenID, userID, classID uint64,
	req UpdateClassReq,
	ip, ua string,
) (*updateClassResp, error) {
	if kindergartenID == 0 {
		return nil, fmt.Errorf("NO_KINDERGARTEN")
	}
	if err := validateGrade(req.Grade); err != nil {
		return nil, err
	}
	c, err := s.classRepo.FindByID(ctx, kindergartenID, classID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("NOT_FOUND")
	}
	teachers, err := s.buildClassTeachers(ctx, req.TeacherIDs)
	if err != nil {
		return nil, err
	}
	c.Name = req.Name
	c.Grade = req.Grade
	c.Capacity = req.Capacity
	if err := s.classRepo.Save(ctx, c); err != nil {
		return nil, err
	}
	if err := s.classRepo.ReplaceClassTeachers(ctx, classID, teachers); err != nil {
		return nil, err
	}
	s.logOp(ctx, kindergartenID, userID, "classes", "update", "class", classID,
		fmt.Sprintf(`{"name":%q}`, req.Name), ip, ua)
	return &updateClassResp{ID: c.ID, UpdatedAt: c.UpdatedAt.UTC().Format(time.RFC3339)}, nil
}

// --- delete ---

func (s *ClassService) DeleteClass(ctx context.Context, kindergartenID, userID, classID uint64, ip, ua string) error {
	if kindergartenID == 0 {
		return fmt.Errorf("NO_KINDERGARTEN")
	}
	c, err := s.classRepo.FindByID(ctx, kindergartenID, classID)
	if err != nil {
		return err
	}
	if c == nil {
		return fmt.Errorf("NOT_FOUND")
	}
	n, err := s.classRepo.CountStudentsInClass(ctx, kindergartenID, classID)
	if err != nil {
		return err
	}
	if n > 0 {
		return fmt.Errorf("CLASS_NOT_EMPTY")
	}
	if err := s.classRepo.ReplaceClassTeachers(ctx, classID, nil); err != nil {
		return err
	}
	if err := s.classRepo.Delete(ctx, kindergartenID, classID); err != nil {
		return err
	}
	s.logOp(ctx, kindergartenID, userID, "classes", "delete", "class", classID, "{}", ip, ua)
	return nil
}

// --- adjust ---

type AdjustClassReq struct {
	Action      string   `json:"action" binding:"required,oneof=move_in move_out"`
	StudentIDs  []uint64 `json:"student_ids" binding:"required,min=1"`
	FromClassID *uint64  `json:"from_class_id"`
}

type adjustClassResp struct {
	AdjustedCount int64  `json:"adjusted_count"`
	Message       string `json:"message"`
}

func (s *ClassService) AdjustStudents(
	ctx context.Context,
	kindergartenID, userID, targetClassID uint64,
	req AdjustClassReq,
	ip, ua string,
) (*adjustClassResp, error) {
	if kindergartenID == 0 {
		return nil, fmt.Errorf("NO_KINDERGARTEN")
	}
	if req.FromClassID == nil {
		return nil, fmt.Errorf("FROM_CLASS_REQUIRED")
	}
	fromID := *req.FromClassID
	tgt, err := s.classRepo.FindByID(ctx, kindergartenID, targetClassID)
	if err != nil {
		return nil, err
	}
	if tgt == nil {
		return nil, fmt.Errorf("NOT_FOUND")
	}
	src, err := s.classRepo.FindByID(ctx, kindergartenID, fromID)
	if err != nil {
		return nil, err
	}
	if src == nil {
		return nil, fmt.Errorf("FROM_CLASS_NOT_FOUND")
	}

	var fromClassID, toClassID uint64
	switch req.Action {
	case "move_in":
		// 学生从 from_class_id 调入本班 targetClassID
		fromClassID, toClassID = fromID, targetClassID
	case "move_out":
		// 学生从本班 targetClass_id 调出到 from_class_id（复用字段表示目标班）
		fromClassID, toClassID = targetClassID, fromID
	default:
		return nil, fmt.Errorf("INVALID_ACTION")
	}

	okCount, err := s.classRepo.CountStudentsInClassWithIDs(ctx, kindergartenID, fromClassID, req.StudentIDs)
	if err != nil {
		return nil, err
	}
	if okCount != int64(len(req.StudentIDs)) {
		return nil, fmt.Errorf("PARTIAL_MOVE")
	}

	n, err := s.classRepo.UpdateStudentsClass(ctx, kindergartenID, fromClassID, toClassID, req.StudentIDs)
	if err != nil {
		return nil, err
	}
	if n != int64(len(req.StudentIDs)) {
		return nil, fmt.Errorf("PARTIAL_MOVE")
	}
	msg := fmt.Sprintf("成功调整 %d 位学生班级", n)
	s.logOp(ctx, kindergartenID, userID, "classes", "adjust", "class", targetClassID,
		fmt.Sprintf(`{"action":%q,"count":%d}`, req.Action, n), ip, ua)
	return &adjustClassResp{AdjustedCount: n, Message: msg}, nil
}

// ErrNotFound for handler
func IsNotFound(err error) bool {
	return err != nil && err.Error() == "NOT_FOUND"
}

func IsClientClassErr(err error) bool {
	if err == nil {
		return false
	}
	switch err.Error() {
	case "INVALID_GRADE", "INVALID_TEACHER_TITLE", "DUPLICATE_TEACHER", "USER_NOT_FOUND",
		"FROM_CLASS_REQUIRED", "FROM_CLASS_NOT_FOUND", "CLASS_NOT_EMPTY", "PARTIAL_MOVE", "INVALID_ACTION":
		return true
	default:
		return false
	}
}
