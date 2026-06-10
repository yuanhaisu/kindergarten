package service

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/pkg/pagination"
)

type StudentService struct {
	db       *gorm.DB
	students *repository.StudentRepo
	classes  *repository.ClassRepo
	users    *repository.UserRepo
	roles    *repository.UserRoleRepo
	kg       *repository.KindergartenRepo
	oplog    *repository.OperationLogRepo
}

func NewStudentService(
	db *gorm.DB,
	students *repository.StudentRepo,
	classes *repository.ClassRepo,
	users *repository.UserRepo,
	roles *repository.UserRoleRepo,
	kg *repository.KindergartenRepo,
	oplog *repository.OperationLogRepo,
) *StudentService {
	return &StudentService{
		db: db, students: students, classes: classes, users: users, roles: roles, kg: kg, oplog: oplog,
	}
}

func (s *StudentService) logOp(ctx context.Context, kindergartenID, userID uint64, module, action, targetType string, targetID uint64, detail, ip, ua string) {
	_ = s.oplog.Create(ctx, &domain.OperationLog{
		KindergartenID: kindergartenID,
		UserID:         userID,
		Module:         module,
		Action:         action,
		TargetType:     targetType,
		TargetID:       targetID,
		Detail:         detail,
		IP:             ip,
		UserAgent:      truncateUAStudent(ua),
	})
}

func truncateUAStudent(ua string) string {
	if utf8.RuneCountInString(ua) <= 300 {
		return ua
	}
	return string([]rune(ua)[:300])
}

var phoneReStudent = regexp.MustCompile(`^1[3-9]\d{9}$`)

func statusLabel(st int8) string {
	switch st {
	case 0:
		return "待入学"
	case 1:
		return "在读"
	case 2:
		return "毕业"
	case 3:
		return "退学"
	default:
		return "未知"
	}
}

func relationLabel(rel domain.ParentRelation) string {
	switch rel {
	case domain.RelationFather:
		return "爸爸"
	case domain.RelationMother:
		return "妈妈"
	case domain.RelationGrandfather:
		return "爷爷"
	case domain.RelationGrandmother:
		return "奶奶"
	case domain.RelationOther:
		return "其他"
	default:
		return string(rel)
	}
}

func ageCN(birth, ref time.Time) string {
	if birth.IsZero() {
		return ""
	}
	y1, m1, d1 := birth.Date()
	y2, m2, d2 := ref.Date()
	years := y2 - y1
	if m2 < m1 || (m2 == m1 && d2 < d1) {
		years--
	}
	if years < 0 {
		return "0岁"
	}
	months := (y2-y1)*12 + int(m2-m1)
	if d2 < d1 {
		months--
	}
	half := (months - years*12) >= 6
	if half {
		return fmt.Sprintf("%d岁半", years)
	}
	return fmt.Sprintf("%d岁", years)
}

func parseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty")
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.ParseInLocation("2006-01-02", s, time.Local)
}

func nextStudentNo(ctx context.Context, r *repository.StudentRepo, kgID uint64) (string, error) {
	n, err := r.CountInKindergarten(ctx, kgID)
	if err != nil {
		return "", err
	}
	y := time.Now().Year()
	return fmt.Sprintf("%d%05d", y, n+1), nil
}

// --- list ---

type studentListItem struct {
	ID           uint64 `json:"id"`
	Name         string `json:"name"`
	Avatar       string `json:"avatar"`
	Gender       int8   `json:"gender"`
	Birthday     string `json:"birthday"`
	Age          string `json:"age"`
	StudentNo    string `json:"student_no"`
	ClassName    string `json:"class_name"`
	ClassID      uint64 `json:"class_id"`
	Status       int8   `json:"status"`
	StatusLabel  string `json:"status_label"`
	EnrollDate   string `json:"enroll_date"`
	ParentsCount int64  `json:"parents_count"`
	CreatedAt    string `json:"created_at"`
}

func (s *StudentService) ListStudents(
	ctx context.Context,
	kindergartenID uint64,
	q pagination.Query,
	classID *uint64,
	status *int8,
) ([]studentListItem, int64, int, int, error) {
	if kindergartenID == 0 {
		return nil, 0, 0, 0, fmt.Errorf("NO_KINDERGARTEN")
	}
	q.Normalize()
	list, total, err := s.students.List(ctx, kindergartenID, classID, status, q.Keyword, q.Offset(), q.PageSize)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	ids := make([]uint64, 0, len(list))
	for _, st := range list {
		ids = append(ids, st.ID)
	}
	pc, err := s.students.CountParentsByStudentIDs(ctx, ids)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	ref := time.Now()
	out := make([]studentListItem, 0, len(list))
	for _, st := range list {
		cn := ""
		if st.Class != nil {
			cn = st.Class.Name
		}
		bd := ""
		if !st.Birthday.IsZero() {
			bd = st.Birthday.Format("2006-01-02")
		}
		ed := ""
		if !st.EnrollDate.IsZero() {
			ed = st.EnrollDate.Format("2006-01-02")
		}
		out = append(out, studentListItem{
			ID:           st.ID,
			Name:         st.Name,
			Avatar:       st.Avatar,
			Gender:       st.Gender,
			Birthday:     bd,
			Age:          ageCN(st.Birthday, ref),
			StudentNo:    st.StudentNo,
			ClassName:    cn,
			ClassID:      st.ClassID,
			Status:       st.Status,
			StatusLabel:  statusLabel(st.Status),
			EnrollDate:   ed,
			ParentsCount: pc[st.ID],
			CreatedAt:    st.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return out, total, q.Page, q.PageSize, nil
}

// --- detail ---

type studentParentDetail struct {
	UserID        uint64 `json:"user_id"`
	Name          string `json:"name"`
	Phone         string `json:"phone"`
	Relation      string `json:"relation"`
	RelationLabel string `json:"relation_label"`
	IsPrimary     bool   `json:"is_primary"`
	Avatar        string `json:"avatar,omitempty"`
	IDCardNumber  string `json:"id_card_number,omitempty"`
}

type pickupAuthBrief struct {
	ID       uint64 `json:"id"`
	Name     string `json:"name"`
	Relation string `json:"relation"`
	Phone    string `json:"phone"`
	Status   int8   `json:"status"`
}

type studentDetailResp struct {
	ID                   uint64                `json:"id"`
	Name                 string                `json:"name"`
	Avatar               string                `json:"avatar"`
	Gender               int8                  `json:"gender"`
	Birthday             string                `json:"birthday"`
	Age                  string                `json:"age"`
	StudentNo            string                `json:"student_no"`
	IDCardNumber         string                `json:"id_card_number,omitempty"`
	HealthNote           string                `json:"health_note,omitempty"`
	Address              string                `json:"address,omitempty"`
	ClassID              uint64                `json:"class_id"`
	ClassName            string                `json:"class_name"`
	EnrollDate           string                `json:"enroll_date"`
	Status               int8                  `json:"status"`
	StatusLabel          string                `json:"status_label"`
	Parents              []studentParentDetail `json:"parents"`
	PickupAuthorizations []pickupAuthBrief     `json:"pickup_authorizations"`
	CreatedAt            string                `json:"created_at"`
}

func (s *StudentService) GetStudent(ctx context.Context, kindergartenID, studentID uint64) (*studentDetailResp, error) {
	if kindergartenID == 0 {
		return nil, fmt.Errorf("NO_KINDERGARTEN")
	}
	st, err := s.students.FindByID(ctx, kindergartenID, studentID)
	if err != nil {
		return nil, err
	}
	if st == nil {
		return nil, fmt.Errorf("NOT_FOUND")
	}
	par, err := s.students.ListParentsWithUser(ctx, studentID)
	if err != nil {
		return nil, err
	}
	pu, err := s.students.ListPickupAuthorizations(ctx, studentID)
	if err != nil {
		return nil, err
	}
	ref := time.Now()
	cn := ""
	if st.Class != nil {
		cn = st.Class.Name
	}
	bd := ""
	if !st.Birthday.IsZero() {
		bd = st.Birthday.Format("2006-01-02")
	}
	ed := ""
	if !st.EnrollDate.IsZero() {
		ed = st.EnrollDate.Format("2006-01-02")
	}
	parents := make([]studentParentDetail, 0, len(par))
	for _, p := range par {
		item := studentParentDetail{
			UserID:        p.UserID,
			Relation:      string(p.Relation),
			RelationLabel: relationLabel(p.Relation),
			IsPrimary:     p.IsPrimary,
		}
		if p.User != nil {
			item.Name = p.User.Nickname
			item.Phone = p.User.Phone
			item.Avatar = p.User.Avatar
			item.IDCardNumber = p.User.IDCardNumber
		}
		parents = append(parents, item)
	}
	picks := make([]pickupAuthBrief, 0, len(pu))
	for _, a := range pu {
		picks = append(picks, pickupAuthBrief{
			ID: a.ID, Name: a.Name, Relation: a.Relation, Phone: a.Phone, Status: a.Status,
		})
	}
	return &studentDetailResp{
		ID:                   st.ID,
		Name:                 st.Name,
		Avatar:               st.Avatar,
		Gender:               st.Gender,
		Birthday:             bd,
		Age:                  ageCN(st.Birthday, ref),
		StudentNo:            st.StudentNo,
		IDCardNumber:         st.IDCardNumber,
		HealthNote:           st.HealthNote,
		Address:              st.Address,
		ClassID:              st.ClassID,
		ClassName:            cn,
		EnrollDate:           ed,
		Status:               st.Status,
		StatusLabel:          statusLabel(st.Status),
		Parents:              parents,
		PickupAuthorizations: picks,
		CreatedAt:            st.CreatedAt.UTC().Format(time.RFC3339),
	}, nil
}

// --- create ---

type createParentItem struct {
	Name         string `json:"name" binding:"required"`
	Phone        string `json:"phone" binding:"required"`
	Relation     string `json:"relation" binding:"required"`
	IsPrimary    bool   `json:"is_primary"`
	IDCardNumber string `json:"id_card_number"`
}

type CreateStudentReq struct {
	Name         string             `json:"name" binding:"required,max=50"`
	Gender       int8               `json:"gender"`
	Birthday     string             `json:"birthday" binding:"required"`
	IDCardNumber string             `json:"id_card_number"`
	HealthNote   string             `json:"health_note"`
	Address      string             `json:"address"`
	ClassID      uint64             `json:"class_id" binding:"required"`
	EnrollDate   string             `json:"enroll_date" binding:"required"`
	Parents      []createParentItem `json:"parents"`
}

type createStudentResp struct {
	ID        uint64 `json:"id"`
	StudentNo string `json:"student_no"`
	CreatedAt string `json:"created_at"`
}

func validParentRelation(rel string) bool {
	switch domain.ParentRelation(rel) {
	case domain.RelationFather, domain.RelationMother, domain.RelationGrandfather, domain.RelationGrandmother, domain.RelationOther:
		return true
	default:
		return false
	}
}

func validateParents(parents []createParentItem) error {
	if len(parents) == 0 {
		return fmt.Errorf("PARENTS_REQUIRED")
	}
	seen := make(map[string]struct{}, len(parents))
	for _, p := range parents {
		if !phoneReStudent.MatchString(p.Phone) {
			return fmt.Errorf("INVALID_PHONE")
		}
		if _, dup := seen[p.Phone]; dup {
			return fmt.Errorf("DUPLICATE_PARENT_PHONE")
		}
		seen[p.Phone] = struct{}{}
		if !validParentRelation(p.Relation) {
			return fmt.Errorf("INVALID_RELATION")
		}
	}
	return nil
}

func (s *StudentService) persistStudentWithParentsInTx(
	ctx context.Context,
	tx *gorm.DB,
	kindergartenID uint64,
	kg *domain.Kindergarten,
	st *domain.Student,
	parents []createParentItem,
	hash string,
) error {
	if err := tx.Create(st).Error; err != nil {
		return err
	}
	for _, p := range parents {
		u, err := s.users.FindByPhone(ctx, p.Phone)
		if err != nil {
			return err
		}
		if u == nil {
			u = &domain.User{
				Phone:        p.Phone,
				Nickname:     p.Name,
				PasswordHash: hash,
				IDCardNumber: strings.TrimSpace(p.IDCardNumber),
			}
			if err := tx.Create(u).Error; err != nil {
				return err
			}
		} else if p.IDCardNumber != "" {
			if err := tx.Model(&domain.User{}).Where("id = ?", u.ID).
				Update("id_card_number", strings.TrimSpace(p.IDCardNumber)).Error; err != nil {
				return err
			}
		}
		ur, err := s.roles.FindParentInKindergarten(ctx, u.ID, kindergartenID)
		if err != nil {
			return err
		}
		if ur == nil {
			kid := kindergartenID
			nur := &domain.UserRole{
				UserID:           u.ID,
				Role:             "parent",
				KindergartenID:   &kid,
				KindergartenName: kg.Name,
			}
			if err := tx.Create(nur).Error; err != nil {
				return err
			}
		}
		sp := &domain.StudentParent{
			StudentID: st.ID,
			UserID:    u.ID,
			Relation:  domain.ParentRelation(p.Relation),
			IsPrimary: p.IsPrimary,
		}
		if err := tx.Create(sp).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *StudentService) CreateStudent(
	ctx context.Context,
	kindergartenID, userID uint64,
	req CreateStudentReq,
	ip, ua string,
) (*createStudentResp, error) {
	if kindergartenID == 0 {
		return nil, fmt.Errorf("NO_KINDERGARTEN")
	}
	if err := validateParents(req.Parents); err != nil {
		return nil, err
	}
	cl, err := s.classes.FindByID(ctx, kindergartenID, req.ClassID)
	if err != nil {
		return nil, err
	}
	if cl == nil {
		return nil, fmt.Errorf("CLASS_NOT_FOUND")
	}
	bd, err := parseDate(req.Birthday)
	if err != nil {
		return nil, fmt.Errorf("INVALID_BIRTHDAY")
	}
	ed, err := parseDate(req.EnrollDate)
	if err != nil {
		return nil, fmt.Errorf("INVALID_ENROLL_DATE")
	}
	studentNo, err := nextStudentNo(ctx, s.students, kindergartenID)
	if err != nil {
		return nil, err
	}
	kg, err := s.kg.FindByID(ctx, kindergartenID)
	if err != nil {
		return nil, err
	}
	if kg == nil {
		return nil, fmt.Errorf("KINDERGARTEN_NOT_FOUND")
	}
	st := &domain.Student{
		KindergartenID: kindergartenID,
		ClassID:        req.ClassID,
		Name:           req.Name,
		Gender:         req.Gender,
		Birthday:       bd,
		StudentNo:      studentNo,
		IDCardNumber:   req.IDCardNumber,
		Address:        req.Address,
		HealthNote:     req.HealthNote,
		EnrollDate:     ed,
		Status:         1,
	}
	hash, err := HashPassword("KindyParent#1")
	if err != nil {
		return nil, err
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.persistStudentWithParentsInTx(ctx, tx, kindergartenID, kg, st, req.Parents, hash)
	})
	if err != nil {
		return nil, err
	}
	s.logOp(ctx, kindergartenID, userID, "students", "create", "student", st.ID,
		fmt.Sprintf(`{"name":%q,"student_no":%q}`, st.Name, st.StudentNo), ip, ua)
	return &createStudentResp{
		ID:        st.ID,
		StudentNo: st.StudentNo,
		CreatedAt: st.CreatedAt.UTC().Format(time.RFC3339),
	}, nil
}

// --- update ---

type UpdateStudentReq struct {
	Name       string `json:"name"`
	HealthNote string `json:"health_note"`
	Address    string `json:"address"`
	ClassID    uint64 `json:"class_id"`
}

type updateStudentResp struct {
	ID        uint64 `json:"id"`
	UpdatedAt string `json:"updated_at"`
}

func (s *StudentService) UpdateStudent(
	ctx context.Context,
	kindergartenID, userID, studentID uint64,
	req UpdateStudentReq,
	ip, ua string,
) (*updateStudentResp, error) {
	if kindergartenID == 0 {
		return nil, fmt.Errorf("NO_KINDERGARTEN")
	}
	st, err := s.students.FindByID(ctx, kindergartenID, studentID)
	if err != nil {
		return nil, err
	}
	if st == nil {
		return nil, fmt.Errorf("NOT_FOUND")
	}
	if strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("INVALID_NAME")
	}
	if req.ClassID > 0 && req.ClassID != st.ClassID {
		cl, err := s.classes.FindByID(ctx, kindergartenID, req.ClassID)
		if err != nil {
			return nil, err
		}
		if cl == nil {
			return nil, fmt.Errorf("CLASS_NOT_FOUND")
		}
		st.ClassID = req.ClassID
	}
	st.Name = strings.TrimSpace(req.Name)
	st.HealthNote = req.HealthNote
	st.Address = req.Address
	if err := s.students.Save(ctx, st); err != nil {
		return nil, err
	}
	s.logOp(ctx, kindergartenID, userID, "students", "update", "student", st.ID,
		fmt.Sprintf(`{"name":%q,"class_id":%d}`, st.Name, st.ClassID), ip, ua)
	return &updateStudentResp{
		ID:        st.ID,
		UpdatedAt: st.UpdatedAt.UTC().Format(time.RFC3339),
	}, nil
}

func (s *StudentService) DeleteStudent(ctx context.Context, kindergartenID, userID, studentID uint64, ip, ua string) error {
	if kindergartenID == 0 {
		return fmt.Errorf("NO_KINDERGARTEN")
	}
	st, err := s.students.FindByID(ctx, kindergartenID, studentID)
	if err != nil {
		return err
	}
	if st == nil {
		return fmt.Errorf("NOT_FOUND")
	}
	if err := s.students.UnscopedDeleteParentsByStudent(ctx, studentID); err != nil {
		return err
	}
	if err := s.students.UnscopedDeletePickupAuthsByStudent(ctx, studentID); err != nil {
		return err
	}
	if err := s.students.UnscopedDeletePickupQRCodesByStudent(ctx, studentID); err != nil {
		return err
	}
	if err := s.students.Delete(ctx, kindergartenID, studentID); err != nil {
		return err
	}
	s.logOp(ctx, kindergartenID, userID, "students", "delete", "student", studentID,
		fmt.Sprintf(`{"name":%q,"student_no":%q}`, st.Name, st.StudentNo), ip, ua)
	return nil
}

type UpdateStudentStatusReq struct {
	Status int8   `json:"status"`
	Reason string `json:"reason"`
}

type UpdateStudentStatusResp struct {
	ID          uint64 `json:"id"`
	Status      int8   `json:"status"`
	StatusLabel string `json:"status_label"`
	UpdatedAt   string `json:"updated_at"`
}

func (s *StudentService) UpdateStudentStatus(
	ctx context.Context,
	kindergartenID, userID, studentID uint64,
	req UpdateStudentStatusReq,
	ip, ua string,
) (*UpdateStudentStatusResp, error) {
	if kindergartenID == 0 {
		return nil, fmt.Errorf("NO_KINDERGARTEN")
	}
	st, err := s.students.FindByID(ctx, kindergartenID, studentID)
	if err != nil {
		return nil, err
	}
	if st == nil {
		return nil, fmt.Errorf("NOT_FOUND")
	}
	if req.Status < 0 || req.Status > 3 {
		return nil, fmt.Errorf("INVALID_STATUS")
	}
	st.Status = req.Status
	if err := s.students.Save(ctx, st); err != nil {
		return nil, err
	}
	detail := fmt.Sprintf(`{"status":%d,"reason":%q}`, req.Status, req.Reason)
	s.logOp(ctx, kindergartenID, userID, "students", "update_status", "student", st.ID, detail, ip, ua)
	return &UpdateStudentStatusResp{
		ID:          st.ID,
		Status:      st.Status,
		StatusLabel: statusLabel(st.Status),
		UpdatedAt:   st.UpdatedAt.UTC().Format(time.RFC3339),
	}, nil
}

func genderLabel(g int8) string {
	switch g {
	case 1:
		return "男"
	case 2:
		return "女"
	default:
		return "未知"
	}
}

func parseGenderCell(s string) int8 {
	s = strings.TrimSpace(s)
	switch s {
	case "1", "男", "M", "m":
		return 1
	case "2", "女", "F", "f":
		return 2
	default:
		return 0
	}
}

func mustCell(row, col int) string {
	c, _ := excelize.CoordinatesToCellName(col, row)
	return c
}

// ExportStudentsXLSX 返回 xlsx 二进制与建议文件名（不含路径）。
func (s *StudentService) ExportStudentsXLSX(
	ctx context.Context,
	kindergartenID uint64,
	classID *uint64,
	status *int8,
) ([]byte, string, error) {
	if kindergartenID == 0 {
		return nil, "", fmt.Errorf("NO_KINDERGARTEN")
	}
	list, err := s.students.ListForExport(ctx, kindergartenID, classID, status)
	if err != nil {
		return nil, "", err
	}
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	sheet := "学生名册"
	if err := f.SetSheetName("Sheet1", sheet); err != nil {
		return nil, "", err
	}
	headers := []string{"学号", "姓名", "性别", "生日", "年龄", "班级", "状态", "入学日期"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}
	ref := time.Now()
	for i, stu := range list {
		row := i + 2
		cn := ""
		if stu.Class != nil {
			cn = stu.Class.Name
		}
		bd := ""
		if !stu.Birthday.IsZero() {
			bd = stu.Birthday.Format("2006-01-02")
		}
		ed := ""
		if !stu.EnrollDate.IsZero() {
			ed = stu.EnrollDate.Format("2006-01-02")
		}
		_ = f.SetCellValue(sheet, mustCell(row, 1), stu.StudentNo)
		_ = f.SetCellValue(sheet, mustCell(row, 2), stu.Name)
		_ = f.SetCellValue(sheet, mustCell(row, 3), genderLabel(stu.Gender))
		_ = f.SetCellValue(sheet, mustCell(row, 4), bd)
		_ = f.SetCellValue(sheet, mustCell(row, 5), ageCN(stu.Birthday, ref))
		_ = f.SetCellValue(sheet, mustCell(row, 6), cn)
		_ = f.SetCellValue(sheet, mustCell(row, 7), statusLabel(stu.Status))
		_ = f.SetCellValue(sheet, mustCell(row, 8), ed)
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, "", err
	}
	fn := fmt.Sprintf("students_export_%s.xlsx", time.Now().Format("20060102_150405"))
	return buf.Bytes(), fn, nil
}

// ImportFailDetail 导入失败行说明（与 admin 规格一致）。
type ImportFailDetail struct {
	Row    int    `json:"row"`
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// ImportStudentsResult 批量导入结果。
type ImportStudentsResult struct {
	Total        int                `json:"total"`
	SuccessCount int                `json:"success_count"`
	FailCount    int                `json:"fail_count"`
	FailDetails  []ImportFailDetail `json:"fail_details"`
}

// ImportStudentsExcel 解析 .xlsx：列 姓名、班级、家长手机；可选列 性别、生日。首行可为表头（首格含「姓名」则跳过）。
func (s *StudentService) ImportStudentsExcel(
	ctx context.Context,
	kindergartenID, userID uint64,
	fileBytes []byte,
	ip, ua string,
) (*ImportStudentsResult, error) {
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
	f, err := excelize.OpenReader(bytes.NewReader(fileBytes))
	if err != nil {
		return nil, fmt.Errorf("INVALID_FILE")
	}
	defer func() { _ = f.Close() }()
	sheet := f.GetSheetName(0)
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, err
	}
	start := 0
	if len(rows) > 0 && len(rows[0]) > 0 && strings.Contains(strings.TrimSpace(rows[0][0]), "姓名") {
		start = 1
	}
	hash, err := HashPassword("KindyParent#1")
	if err != nil {
		return nil, err
	}
	out := &ImportStudentsResult{FailDetails: []ImportFailDetail{}}
	defaultBD, _ := time.ParseInLocation("2006-01-02", "2018-01-01", time.Local)
	enrollStr := time.Now().Format("2006-01-02")

	for i := start; i < len(rows); i++ {
		r := rows[i]
		if len(r) == 0 || (len(r) == 1 && strings.TrimSpace(r[0]) == "") {
			continue
		}
		excelRow := i + 1
		name := ""
		if len(r) > 0 {
			name = strings.TrimSpace(r[0])
		}
		className := ""
		if len(r) > 1 {
			className = strings.TrimSpace(r[1])
		}
		phone := ""
		if len(r) > 2 {
			phone = strings.TrimSpace(r[2])
		}
		if name == "" && className == "" && phone == "" {
			continue
		}
		out.Total++
		fail := func(reason string) {
			out.FailCount++
			out.FailDetails = append(out.FailDetails, ImportFailDetail{Row: excelRow, Name: name, Reason: reason})
		}
		if name == "" {
			fail("姓名不能为空")
			continue
		}
		if className == "" {
			fail("班级不能为空")
			continue
		}
		if !phoneReStudent.MatchString(phone) {
			fail("手机号格式错误")
			continue
		}
		cl, err := s.students.FindClassByName(ctx, kindergartenID, className)
		if err != nil {
			return nil, err
		}
		if cl == nil {
			fail("班级不存在")
			continue
		}
		gender := int8(0)
		if len(r) > 3 {
			gender = parseGenderCell(r[3])
		}
		bd := defaultBD
		if len(r) > 4 && strings.TrimSpace(r[4]) != "" {
			if t, e := parseDate(r[4]); e == nil {
				bd = t
			}
		}
		studentNo, err := nextStudentNo(ctx, s.students, kindergartenID)
		if err != nil {
			return nil, err
		}
		parentName := name + "家长"
		st := &domain.Student{
			KindergartenID: kindergartenID,
			ClassID:        cl.ID,
			Name:           name,
			Gender:         gender,
			Birthday:       bd,
			StudentNo:      studentNo,
			Status:         1,
		}
		if ed, e := parseDate(enrollStr); e == nil {
			st.EnrollDate = ed
		}
		parents := []createParentItem{{
			Name:      parentName,
			Phone:     phone,
			Relation:  string(domain.RelationFather),
			IsPrimary: true,
		}}
		err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			return s.persistStudentWithParentsInTx(ctx, tx, kindergartenID, kg, st, parents, hash)
		})
		if err != nil {
			fail("保存失败")
			continue
		}
		out.SuccessCount++
	}
	detail := fmt.Sprintf(`{"total":%d,"success":%d,"fail":%d}`, out.Total, out.SuccessCount, out.FailCount)
	s.logOp(ctx, kindergartenID, userID, "students", "import", "student", 0, detail, ip, ua)
	return out, nil
}

// IsStudentClientErr 业务校验类错误（4xx 提示）。
func IsStudentClientErr(err error) bool {
	if err == nil {
		return false
	}
	switch err.Error() {
	case "NO_KINDERGARTEN", "CLASS_NOT_FOUND", "INVALID_BIRTHDAY", "INVALID_ENROLL_DATE",
		"PARENTS_REQUIRED", "INVALID_PHONE", "INVALID_RELATION", "KINDERGARTEN_NOT_FOUND",
		"INVALID_STATUS", "INVALID_NAME", "DUPLICATE_PARENT_PHONE", "INVALID_FILE":
		return true
	default:
		return false
	}
}
