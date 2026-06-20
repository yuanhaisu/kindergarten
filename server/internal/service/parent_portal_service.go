package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

var (
	ErrParentContext      = errors.New("PARENT_CONTEXT")
	ErrParentNotFound     = errors.New("PARENT_NOT_FOUND")
	ErrParentInvalidParams = errors.New("PARENT_INVALID_PARAMS")
	ErrParentForbidden    = errors.New("PARENT_FORBIDDEN")
)

type ParentPortalContext struct {
	UserID         uint64
	KindergartenID uint64
	CurrentStudent *domain.Student
	Students       []domain.Student
	Class          *domain.Class
	Relation       string
}

type ParentPortalService struct {
	db       *gorm.DB
	parents  *repository.ParentRepo
	roles    *repository.UserRoleRepo
	students *repository.StudentRepo
	classes  *repository.ClassRepo
	pickup   *repository.PickupRepo
	calendar *repository.CalendarRepo
	notice   *repository.NoticeRepo
}

func NewParentPortalService(
	db *gorm.DB,
	parents *repository.ParentRepo,
	roles *repository.UserRoleRepo,
	students *repository.StudentRepo,
	classes *repository.ClassRepo,
	pickup *repository.PickupRepo,
	calendar *repository.CalendarRepo,
	notice *repository.NoticeRepo,
) *ParentPortalService {
	return &ParentPortalService{db: db, parents: parents, roles: roles, students: students, classes: classes, pickup: pickup, calendar: calendar, notice: notice}
}

func (s *ParentPortalService) ResolveContext(ctx context.Context, userID, kindergartenID uint64, studentID *uint64) (*ParentPortalContext, error) {
	ur, err := s.roles.FindParentInKindergarten(ctx, userID, kindergartenID)
	if err != nil || ur == nil {
		return nil, ErrParentContext
	}
	spRows, err := s.parents.ListStudentParentsForUsersInKindergarten(ctx, kindergartenID, []uint64{userID})
	if err != nil || len(spRows) == 0 {
		return nil, ErrParentContext
	}
	var students []domain.Student
	for _, sp := range spRows {
		if sp.Student != nil {
			students = append(students, *sp.Student)
		}
	}
	if len(students) == 0 {
		return nil, ErrParentContext
	}

	current := &students[0]
	relation := string(spRows[0].Relation)
	if studentID != nil && *studentID != 0 {
		for i := range spRows {
			if spRows[i].StudentID == *studentID {
				current = spRows[i].Student
				relation = string(spRows[i].Relation)
				break
			}
		}
	}

	pc := &ParentPortalContext{
		UserID:         userID,
		KindergartenID: kindergartenID,
		CurrentStudent: current,
		Students:       students,
		Relation:       relation,
	}
	if current != nil {
		class, _ := s.classes.FindByID(ctx, kindergartenID, current.ClassID)
		if class != nil {
			pc.Class = class
		}
	}
	return pc, nil
}

// ListGrowthFeed 家长端统一成长记录流:
// - target_type=1 (全班记录): 孩子所在班级的所有班级记录都可见
// - target_type=2 (指定学生): 仅目标学生家长可见
// 支持 year/month 可选过滤
func (s *ParentPortalService) ListGrowthFeed(ctx context.Context, pc *ParentPortalContext, studentID *uint64, year, month int, page, pageSize int) (map[string]any, error) {
	if len(pc.Students) == 0 {
		return map[string]any{"list": []any{}, "total": int64(0), "page": page, "page_size": pageSize}, nil
	}

	studentIDs := make([]uint64, len(pc.Students))
	classIDSet := make(map[uint64]bool)
	for i, st := range pc.Students {
		studentIDs[i] = st.ID
		classIDSet[st.ClassID] = true
	}
	classIDs := make([]uint64, 0, len(classIDSet))
	for cid := range classIDSet {
		classIDs = append(classIDs, cid)
	}

	db := s.db.WithContext(ctx).Model(&domain.GrowthDailyLog{}).
		Where("growth_daily_logs.kindergarten_id = ?", pc.KindergartenID).
		Where("growth_daily_logs.class_id IN ?", classIDs)

	if year > 0 && month > 0 {
		start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
		end := start.AddDate(0, 1, 0)
		db = db.Where("growth_daily_logs.date >= ? AND growth_daily_logs.date < ?", start, end)
	}

	// 统一流: 全班记录 + 个体记录(仅目标学生)
	db = db.Where(
		"growth_daily_logs.target_type = 1 OR (growth_daily_logs.target_type = 2 AND growth_daily_logs.id IN (SELECT log_id FROM growth_log_students WHERE student_id IN ?))",
		studentIDs,
	)

	if studentID != nil && *studentID != 0 {
		db = db.Where(
			"growth_daily_logs.target_type = 1 OR (growth_daily_logs.target_type = 2 AND growth_daily_logs.id IN (SELECT log_id FROM growth_log_students WHERE student_id = ?))",
			*studentID,
		)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (page - 1) * pageSize
	var rows []domain.GrowthDailyLog
	if err := db.Preload("Creator").Preload("Class").Order("growth_daily_logs.date DESC, growth_daily_logs.id DESC").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, err
	}

	list := make([]map[string]any, len(rows))
	for i, r := range rows {
		m := map[string]any{
			"id":               r.ID,
			"date":             r.Date.Format("2006-01-02"),
			"target_type":      r.TargetType,
			"content":          r.Content,
			"performance_tags": r.PerformanceTags,
			"created_at":       r.CreatedAt.UTC().Format(time.RFC3339),
		}
		if r.Creator != nil {
			m["creator"] = map[string]any{
				"id":       r.CreatorID,
				"nickname": r.Creator.Nickname,
				"avatar":   r.Creator.Avatar,
			}
		}
		if r.Class != nil {
			m["class_name"] = r.Class.Name
		}
		// 个体记录时返回关联学生信息
		if r.TargetType == 2 {
			var linkedStudents []domain.Student
			_ = s.db.WithContext(ctx).
				Joins("INNER JOIN growth_log_students gls ON gls.student_id = students.id AND gls.log_id = ?", r.ID).
				Find(&linkedStudents).Error
			studentNames := make([]string, len(linkedStudents))
			for j, st := range linkedStudents {
				studentNames[j] = st.Name
			}
			m["related_students"] = studentNames
			m["related_students_count"] = len(studentNames)
		}
		list[i] = m
	}

	return map[string]any{"list": list, "total": total, "page": page, "page_size": pageSize}, nil
}

// ==================== Pickup ====================

func pickupLabel(t domain.PickupType) string {
	switch t {
	case domain.PickupTypeDropOff:
		return "送入"
	case domain.PickupTypePickUp:
		return "接走"
	default:
		return "未知"
	}
}

// ListPickupRecords 家长查看孩子的接送记录（分页）。
func (s *ParentPortalService) ListPickupRecords(ctx context.Context, pc *ParentPortalContext, page, pageSize int) (map[string]any, error) {
	if len(pc.Students) == 0 {
		return map[string]any{"list": []any{}, "total": int64(0), "page": page, "page_size": pageSize}, nil
	}

	studentIDs := make([]uint64, len(pc.Students))
	for i, st := range pc.Students {
		studentIDs[i] = st.ID
	}

	if page <= 0 { page = 1 }
	if pageSize <= 0 || pageSize > 50 { pageSize = 20 }
	offset := (page - 1) * pageSize

	rows, total, err := s.pickup.ListRecordsByStudentIDs(ctx, pc.KindergartenID, studentIDs, offset, pageSize)
	if err != nil {
		return nil, err
	}

	type item struct {
		ID             uint64 `json:"id"`
		Type           int8   `json:"type"`
		TypeLabel      string `json:"type_label"`
		Date           string `json:"date"`
		StudentName    string `json:"student_name"`
		ClassName      string `json:"class_name"`
		PickupName     string `json:"pickup_name"`
		PickupRelation string `json:"pickup_relation"`
		RecordedAt     string `json:"recorded_at"`
	}

	list := make([]item, 0, len(rows))
	for _, r := range rows {
		list = append(list, item{
			ID:             r.ID,
			Type:           int8(r.Type),
			TypeLabel:      pickupLabel(r.Type),
			Date:           r.RecordedAt.Format("2006-01-02"),
			StudentName:    r.StudentName,
			ClassName:      r.ClassName,
			PickupName:     r.PickupName,
			PickupRelation: r.PickupRelation,
			RecordedAt:     r.RecordedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	return map[string]any{
		"list":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}, nil
}

func RelationLabel(r domain.ParentRelation) string {
	m := map[domain.ParentRelation]string{
		domain.RelationFather:      "爸爸",
		domain.RelationMother:      "妈妈",
		domain.RelationGrandfather: "爷爷",
		domain.RelationGrandmother: "奶奶",
		domain.RelationOther:       "其他",
	}
	if v, ok := m[r]; ok { return v }
	return string(r)
}

// ==================== Authorized Persons ====================

// CreateAuthorizedPersonInput 家长创建授权接送人入参。
type CreateAuthorizedPersonInput struct {
	StudentID       uint64 `json:"student_id"`
	Name            string `json:"name"`
	Phone           string `json:"phone"`
	SmsCode         string `json:"sms_code"`
	IDCardNumber    string `json:"id_card_number"`
	Avatar          string `json:"avatar"`
	Relation        string `json:"relation"`
	AuthType        string `json:"auth_type"`         // permanent / temporary
	AllowPickup     *bool  `json:"allow_pickup"`
	AllowViewRecord *bool  `json:"allow_view_record"`
	AllowViewStatus *bool  `json:"allow_view_status"`
	AllowGrowthView *bool  `json:"allow_growth_view"`
	NotifyOnPickup  *bool  `json:"notify_on_pickup"`
}

// UpdateAuthorizedPersonInput 家长编辑授权接送人入参。
type UpdateAuthorizedPersonInput struct {
	Name            string `json:"name"`
	Phone           string `json:"phone"`
	Avatar          string `json:"avatar"`
	Relation        string `json:"relation"`
	AuthType        string `json:"auth_type"`
	AllowPickup     *bool  `json:"allow_pickup"`
	AllowViewRecord *bool  `json:"allow_view_record"`
	AllowViewStatus *bool  `json:"allow_view_status"`
	AllowGrowthView *bool  `json:"allow_growth_view"`
	NotifyOnPickup  *bool  `json:"notify_on_pickup"`
	ExpireAt        string `json:"expire_at"`
}

// ListAuthorizedPersons 家长查看授权接送人列表。
func (s *ParentPortalService) ListAuthorizedPersons(ctx context.Context, pc *ParentPortalContext, page, pageSize int) (map[string]any, error) {
	if len(pc.Students) == 0 {
		return map[string]any{"list": []any{}, "total": int64(0), "page": page, "page_size": pageSize}, nil
	}

	studentIDs := make([]uint64, len(pc.Students))
	studentIDSet := make(map[uint64]bool)
	for i, st := range pc.Students {
		studentIDs[i] = st.ID
		studentIDSet[st.ID] = true
	}

	if page <= 0 { page = 1 }
	if pageSize <= 0 || pageSize > 50 { pageSize = 20 }
	offset := (page - 1) * pageSize

	rows, total, err := s.pickup.ListAuthorizationsByStudentIDs(ctx, pc.KindergartenID, studentIDs, offset, pageSize)
	if err != nil {
		return nil, err
	}

	authTypeLabel := func(t string) string {
		if t == "temporary" { return "临时授权" }
		return "长期授权"
	}
	reviewStatusLabel := func(s int8) string {
		switch s {
		case 0: return "待审核"
		case 1: return "已通过"
		case 2: return "已拒绝"
		default: return "未知"
		}
	}

	list := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		item := map[string]any{
			"id":                r.ID,
			"student_id":        r.StudentID,
			"student_name":      r.StudentName,
			"class_name":        r.ClassName,
			"name":              r.Name,
			"phone":             maskPhone(r.Phone),
			"relation":          r.Relation,
			"avatar":            r.Avatar,
			"auth_type":         r.AuthType,
			"auth_type_label":   authTypeLabel(r.AuthType),
			"allow_pickup":      r.AllowPickup,
			"allow_view_record": r.AllowViewRecord,
			"allow_view_status": r.AllowViewStatus,
			"allow_growth_view": r.AllowGrowthView,
			"notify_on_pickup":  r.NotifyOnPickup,
			"status":            r.Status,
			"review_status":     r.ReviewStatus,
			"review_status_label": reviewStatusLabel(r.ReviewStatus),
		}
		if r.ExpireAt != nil {
			item["expire_at"] = r.ExpireAt.UTC().Format(time.RFC3339)
		}
		list = append(list, item)
	}

	return map[string]any{"list": list, "total": total, "page": page, "page_size": pageSize}, nil
}

// CreateAuthorizedPerson 家长添加授权接送人。
func (s *ParentPortalService) CreateAuthorizedPerson(ctx context.Context, pc *ParentPortalContext, req CreateAuthorizedPersonInput) (map[string]any, error) {
	if pc.CurrentStudent == nil {
		return nil, ErrParentInvalidParams
	}
	// 校验必填: name, phone, relation
	if req.Name == "" || req.Phone == "" || req.Relation == "" {
		return nil, ErrParentInvalidParams
	}
	// 校验学生归属
	found := false
	for _, st := range pc.Students {
		if st.ID == pc.CurrentStudent.ID {
			found = true
			break
		}
	}
	if !found {
		return nil, ErrParentForbidden
	}

	now := time.Now()

	allowPickup := true
	if req.AllowPickup != nil { allowPickup = *req.AllowPickup }
	allowViewRecord := false
	if req.AllowViewRecord != nil { allowViewRecord = *req.AllowViewRecord }
	allowViewStatus := false
	if req.AllowViewStatus != nil { allowViewStatus = *req.AllowViewStatus }
	allowGrowthView := false
	if req.AllowGrowthView != nil { allowGrowthView = *req.AllowGrowthView }
	notifyOnPickup := true
	if req.NotifyOnPickup != nil { notifyOnPickup = *req.NotifyOnPickup }

	authType := req.AuthType
	if authType == "" { authType = "permanent" }

	auth := &domain.PickupAuthorization{
		StudentID:       pc.CurrentStudent.ID,
		Name:            req.Name,
		Phone:           req.Phone,
		Relation:        req.Relation,
		IDCard:          req.IDCardNumber,
		Avatar:          req.Avatar,
		Status:          1, // 1=有效
		CreatorID:       pc.UserID,
		AuthType:        authType,
		AllowPickup:     allowPickup,
		AllowViewRecord: allowViewRecord,
		AllowViewStatus: allowViewStatus,
		AllowGrowthView: allowGrowthView,
		NotifyOnPickup:  notifyOnPickup,
		ReviewStatus:    0, // 0=待审核
	}

	if err := s.pickup.CreateAuthorization(ctx, auth); err != nil {
		return nil, err
	}

	statusLabel := "待审核"
	if auth.ReviewStatus == 1 { statusLabel = "已通过" }

	return map[string]any{
		"id":           auth.ID,
		"status":       auth.ReviewStatus,
		"status_label": statusLabel,
		"created_at":   now.UTC().Format(time.RFC3339),
	}, nil
}

// UpdateAuthorizedPerson 家长编辑授权接送人。
func (s *ParentPortalService) UpdateAuthorizedPerson(ctx context.Context, pc *ParentPortalContext, authID uint64, req UpdateAuthorizedPersonInput) (map[string]any, error) {
	if pc.CurrentStudent == nil {
		return nil, ErrParentInvalidParams
	}
	// 归属校验
	existing, err := s.pickup.FindAuthorizationByStudent(ctx, pc.CurrentStudent.ID, authID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrParentNotFound
	}

	updates := make(map[string]interface{})
	if req.Name != "" { updates["name"] = req.Name }
	if req.Phone != "" { updates["phone"] = req.Phone }
	if req.Avatar != "" { updates["avatar"] = req.Avatar }
	if req.Relation != "" { updates["relation"] = req.Relation }
	if req.AuthType != "" { updates["auth_type"] = req.AuthType }
	if req.AllowPickup != nil { updates["allow_pickup"] = *req.AllowPickup }
	if req.AllowViewRecord != nil { updates["allow_view_record"] = *req.AllowViewRecord }
	if req.AllowViewStatus != nil { updates["allow_view_status"] = *req.AllowViewStatus }
	if req.AllowGrowthView != nil { updates["allow_growth_view"] = *req.AllowGrowthView }
	if req.NotifyOnPickup != nil { updates["notify_on_pickup"] = *req.NotifyOnPickup }
	if req.ExpireAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpireAt)
		if err == nil {
			updates["expire_at"] = t
		}
	}

	if len(updates) == 0 {
		return map[string]any{"id": authID, "updated_at": time.Now().UTC().Format(time.RFC3339)}, nil
	}
	updates["updated_at"] = time.Now()

	if err := s.pickup.UpdateAuthorizationFields(ctx, pc.CurrentStudent.ID, authID, updates); err != nil {
		return nil, err
	}

	return map[string]any{"id": authID, "updated_at": time.Now().UTC().Format(time.RFC3339)}, nil
}

// DeleteAuthorizedPerson 家长删除授权接送人（软删除）。
func (s *ParentPortalService) DeleteAuthorizedPerson(ctx context.Context, pc *ParentPortalContext, authID uint64) error {
	if pc.CurrentStudent == nil {
		return ErrParentInvalidParams
	}
	existing, err := s.pickup.FindAuthorizationByStudent(ctx, pc.CurrentStudent.ID, authID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrParentNotFound
	}
	return s.pickup.SoftDeleteAuthorization(ctx, pc.CurrentStudent.ID, authID)
}
// ==================== Pickup QR Code ====================

// GetQRCode 家长获取接送二维码（不存在或已过期则生成新码）。
func (s *ParentPortalService) GetQRCode(ctx context.Context, pc *ParentPortalContext) (map[string]any, error) {
	if pc.CurrentStudent == nil {
		return nil, ErrParentInvalidParams
	}

	now := time.Now()
	loc := now.Location()

	// 当日 23:59:59 过期
	dayEnd := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, loc)

	// 查已有有效码
	qr, err := s.pickup.FindActiveQRCodeByStudent(ctx, pc.CurrentStudent.ID)
	if err != nil {
		return nil, err
	}
	if qr != nil && qr.ExpireAt.After(now) {
		return map[string]any{
			"qr_code":    qr.Code,
			"student":    map[string]any{"id": pc.CurrentStudent.ID, "name": pc.CurrentStudent.Name},
			"expire_at":  qr.ExpireAt.UTC().Format(time.RFC3339),
		}, nil
	}

	// 失效旧码
	if err := s.pickup.DeactivateStudentQRCodes(ctx, pc.CurrentStudent.ID); err != nil {
		return nil, err
	}

	// 生成新码
	code := generateQRCodeToken(pc.CurrentStudent.ID, now)

	newQR := &domain.PickupQRCode{
		StudentID: pc.CurrentStudent.ID,
		Code:      code,
		ExpireAt:  dayEnd,
		IsActive:  true,
	}
	if err := s.pickup.CreateQRCode(ctx, newQR); err != nil {
		return nil, err
	}

	return map[string]any{
		"qr_code":   newQR.Code,
		"student":   map[string]any{"id": pc.CurrentStudent.ID, "name": pc.CurrentStudent.Name},
		"expire_at": newQR.ExpireAt.UTC().Format(time.RFC3339),
	}, nil
}

// generateQRCodeToken 生成接送二维码 token。
func generateQRCodeToken(studentID uint64, now time.Time) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("PKUP_%d_%s_%s", studentID, now.Format("20060102"), hex.EncodeToString(b))
}

// ==================== Leaves ====================

// ParentLeaveInput 家长请假输入
type ParentLeaveInput struct {
	StudentID uint64 `json:"student_id"`
	Reason    string `json:"reason"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Days      int    `json:"days"`
}

func parentLeaveMap(lr domain.LeaveRequest) map[string]any {
	student := map[string]any{"id": lr.StudentID, "name": "学生", "avatar": ""}
	if lr.Student != nil {
		student["name"] = lr.Student.Name
		student["avatar"] = lr.Student.Avatar
	}
	applicant := map[string]any{"id": lr.ApplicantID, "nickname": "家长"}
	if lr.Applicant != nil && lr.Applicant.Nickname != "" {
		applicant["nickname"] = lr.Applicant.Nickname
	}
	statusLabel := "待审批"
	switch lr.Status {
	case domain.LeaveStatusApproved: statusLabel = "已同意"
	case domain.LeaveStatusRejected: statusLabel = "已拒绝"
	case domain.LeaveStatusCanceled: statusLabel = "已取消"
	}
	return map[string]any{
		"id": lr.ID, "student": student, "applicant": applicant,
		"reason": lr.Reason, "start_date": lr.StartDate.Format("2006-01-02"),
		"end_date": lr.EndDate.Format("2006-01-02"), "days": lr.Days,
		"status": lr.Status, "status_label": statusLabel,
		"review_remark": lr.ReviewRemark,
		"created_at": lr.CreatedAt.UTC().Format(time.RFC3339),
		"reviewed_at": formatOptionalTime(lr.ReviewedAt),
	}
}

func (s *ParentPortalService) ListLeaves(ctx context.Context, pc *ParentPortalContext, status string, page, pageSize int) (map[string]any, error) {
	offset := (page - 1) * pageSize
	db := s.db.WithContext(ctx).Model(&domain.LeaveRequest{}).
		Where("kindergarten_id = ? AND applicant_id = ?", pc.KindergartenID, pc.UserID)
	switch status {
	case "pending":  db = db.Where("status = ?", domain.LeaveStatusPending)
	case "approved": db = db.Where("status = ?", domain.LeaveStatusApproved)
	case "rejected": db = db.Where("status = ?", domain.LeaveStatusRejected)
	case "canceled": db = db.Where("status = ?", domain.LeaveStatusCanceled)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil { return nil, err }
	var rows []domain.LeaveRequest
	if err := db.Preload("Student").Preload("Applicant").Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	list := make([]map[string]any, len(rows))
	for i, r := range rows { list[i] = parentLeaveMap(r) }
	return map[string]any{"list": list, "total": total, "page": page, "page_size": pageSize}, nil
}

func (s *ParentPortalService) GetLeave(ctx context.Context, pc *ParentPortalContext, id uint64) (map[string]any, error) {
	var lr domain.LeaveRequest
	if err := s.db.WithContext(ctx).Preload("Student").Preload("Applicant").
		Where("id = ? AND kindergarten_id = ? AND applicant_id = ?", id, pc.KindergartenID, pc.UserID).
		First(&lr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil, ErrParentNotFound }
		return nil, err
	}
	return parentLeaveMap(lr), nil
}

func (s *ParentPortalService) CreateLeave(ctx context.Context, pc *ParentPortalContext, req ParentLeaveInput) (map[string]any, error) {
	if req.StudentID == 0 || strings.TrimSpace(req.Reason) == "" || req.StartDate == "" || req.EndDate == "" {
		return nil, ErrParentInvalidParams
	}
	startDate, err := time.ParseInLocation("2006-01-02", req.StartDate, time.Local)
	if err != nil { return nil, ErrParentInvalidParams }
	endDate, err := time.ParseInLocation("2006-01-02", req.EndDate, time.Local)
	if err != nil { return nil, ErrParentInvalidParams }
	if endDate.Before(startDate) { return nil, ErrParentInvalidParams }

	// Verify student belongs to this parent
	studentFound := false
	var studentClassID uint64
	for _, s := range pc.Students {
		if s.ID == req.StudentID { studentFound = true; studentClassID = s.ClassID; break }
	}
	if !studentFound { return nil, ErrParentForbidden }

	days := req.Days
	if days <= 0 { days = int(endDate.Sub(startDate).Hours()/24) + 1 }

	lr := &domain.LeaveRequest{
		KindergartenID: pc.KindergartenID, ClassID: studentClassID,
		StudentID: req.StudentID, ApplicantID: pc.UserID,
		Reason: strings.TrimSpace(req.Reason), StartDate: startDate, EndDate: endDate,
		Days: days, Status: domain.LeaveStatusPending,
	}
	if err := s.db.WithContext(ctx).Create(lr).Error; err != nil { return nil, err }
	return map[string]any{"id": lr.ID, "status": int8(lr.Status), "status_label": "待审批", "created_at": lr.CreatedAt.UTC().Format(time.RFC3339)}, nil
}

func (s *ParentPortalService) UpdateLeave(ctx context.Context, pc *ParentPortalContext, id uint64, req ParentLeaveInput) error {
	var lr domain.LeaveRequest
	if err := s.db.WithContext(ctx).Where("id = ? AND kindergarten_id = ? AND applicant_id = ?", id, pc.KindergartenID, pc.UserID).First(&lr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return ErrParentNotFound }
		return err
	}
	if lr.Status != domain.LeaveStatusPending { return ErrParentInvalidParams }
	updates := map[string]any{}
	if req.Reason != "" { updates["reason"] = strings.TrimSpace(req.Reason) }
	if req.StartDate != "" {
		if t, err := time.ParseInLocation("2006-01-02", req.StartDate, time.Local); err == nil { updates["start_date"] = t }
	}
	if req.EndDate != "" {
		if t, err := time.ParseInLocation("2006-01-02", req.EndDate, time.Local); err == nil { updates["end_date"] = t }
	}
	if req.Days > 0 { updates["days"] = req.Days }
	if len(updates) == 0 { return nil }
	s.db.WithContext(ctx).Model(&lr).Updates(updates)
	return nil
}

func (s *ParentPortalService) CancelLeave(ctx context.Context, pc *ParentPortalContext, id uint64) error {
	res := s.db.WithContext(ctx).Model(&domain.LeaveRequest{}).
		Where("id = ? AND kindergarten_id = ? AND applicant_id = ?", id, pc.KindergartenID, pc.UserID).
		Update("status", domain.LeaveStatusCanceled)
	if res.RowsAffected == 0 { return ErrParentNotFound }
	return nil
}

// ==================== Notices ====================

// MarkNoticeRead 标记通知已读。
func (s *ParentPortalService) MarkNoticeRead(ctx context.Context, pc *ParentPortalContext, noticeID uint64) error {
	return s.notice.RecordReadLog(ctx, noticeID, pc.UserID)
}

func (s *ParentPortalService) ListNotices(ctx context.Context, pc *ParentPortalContext, category string, page, pageSize int) (map[string]any, error) {
	classIDs := make([]uint64, len(pc.Students))
	for i, st := range pc.Students { classIDs[i] = st.ClassID }
	if len(classIDs) == 0 { return map[string]any{"list": []any{}, "total": int64(0), "page": page, "page_size": pageSize}, nil }

	db := s.db.WithContext(ctx).Model(&domain.Notice{}).
		Where("kindergarten_id = ? AND class_id IN ? AND status = ?", pc.KindergartenID, classIDs, domain.NoticeStatusPublished)
	if category != "" { db = db.Where("category = ?", category) }

	var total int64
	if err := db.Count(&total).Error; err != nil { return nil, err }

	offset := (page - 1) * pageSize
	var rows []domain.Notice
	if err := db.Order("published_at DESC").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil { return nil, err }

	list := make([]map[string]any, len(rows))
	for i, n := range rows {
		var readLog domain.NoticeReadLog
		_ = s.db.WithContext(ctx).Where("notice_id = ? AND user_id = ?", n.ID, pc.UserID).First(&readLog).Error
		isRead := !readLog.ReadAt.IsZero()
		isConfirmed := readLog.ConfirmedAt != nil
		catLabel := string(n.Category)
		list[i] = map[string]any{
			"id": n.ID, "category": n.Category, "category_label": catLabel,
			"title": n.Title, "content": n.Content, "need_confirm": n.NeedConfirm,
			"status": n.Status, "published_at": formatOptionalTime(n.PublishedAt),
			"created_at": n.CreatedAt.UTC().Format(time.RFC3339),
			"is_read": isRead, "is_confirmed": isConfirmed,
		}
	}
	return map[string]any{"list": list, "total": total, "page": page, "page_size": pageSize}, nil
}

func (s *ParentPortalService) GetNotice(ctx context.Context, pc *ParentPortalContext, id uint64) (map[string]any, error) {
	var n domain.Notice
	if err := s.db.WithContext(ctx).Where("id = ? AND kindergarten_id = ? AND status = ?", id, pc.KindergartenID, domain.NoticeStatusPublished).First(&n).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil, ErrParentNotFound }
		return nil, err
	}
	var readLog domain.NoticeReadLog
	_ = s.db.WithContext(ctx).Where("notice_id = ? AND user_id = ?", n.ID, pc.UserID).First(&readLog).Error
	isRead := !readLog.ReadAt.IsZero()
	isConfirmed := readLog.ConfirmedAt != nil
	return map[string]any{
		"id": n.ID, "category": n.Category, "title": n.Title, "content": n.Content,
		"need_confirm": n.NeedConfirm, "status": n.Status,
		"published_at": formatOptionalTime(n.PublishedAt),
		"created_at": n.CreatedAt.UTC().Format(time.RFC3339),
		"is_read": isRead, "is_confirmed": isConfirmed,
	}, nil
}

func (s *ParentPortalService) ConfirmNotice(ctx context.Context, pc *ParentPortalContext, id uint64) (map[string]any, error) {
	var n domain.Notice
	if err := s.db.WithContext(ctx).Where("id = ? AND kindergarten_id = ?", id, pc.KindergartenID).First(&n).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil, ErrParentNotFound }
		return nil, err
	}
	now := time.Now()
	var log domain.NoticeReadLog
	result := s.db.WithContext(ctx).Where("notice_id = ? AND user_id = ?", id, pc.UserID).First(&log)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		s.db.WithContext(ctx).Create(&domain.NoticeReadLog{NoticeID: id, UserID: pc.UserID, ReadAt: now, ConfirmedAt: &now})
	} else {
		s.db.WithContext(ctx).Model(&log).Updates(map[string]any{"read_at": now, "confirmed_at": now})
	}
	return map[string]any{"id": id, "is_confirmed": true, "confirmed_at": now.UTC().Format(time.RFC3339)}, nil
}

// ==================== Growth Detail / Comments / Likes ====================

func (s *ParentPortalService) verifyGrowthAccess(ctx context.Context, pc *ParentPortalContext, logID uint64) (*domain.GrowthDailyLog, error) {
	var log domain.GrowthDailyLog
	if err := s.db.WithContext(ctx).Where("id = ? AND kindergarten_id = ?", logID, pc.KindergartenID).First(&log).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil, ErrParentNotFound }
		return nil, err
	}
	if log.TargetType == 1 {
		// 全班记录：验证孩子所在班级
		for _, st := range pc.Students {
			if st.ClassID == log.ClassID { return &log, nil }
		}
		return nil, ErrParentForbidden
	}
	// 指定学生记录：验证目标学生在家长孩子中
	var count int64
	s.db.WithContext(ctx).Model(&domain.GrowthLogStudent{}).
		Where("log_id = ? AND student_id IN ?", logID, studentIDs(pc.Students)).Count(&count)
	if count == 0 { return nil, ErrParentForbidden }
	return &log, nil
}

func studentIDs(students []domain.Student) []uint64 {
	ids := make([]uint64, len(students))
	for i, s := range students { ids[i] = s.ID }
	return ids
}

func (s *ParentPortalService) GrowthFeedDetail(ctx context.Context, pc *ParentPortalContext, logID uint64) (map[string]any, error) {
	log, err := s.verifyGrowthAccess(ctx, pc, logID)
	if err != nil { return nil, err }

	m := map[string]any{
		"id": log.ID, "date": log.Date.Format("2006-01-02"), "target_type": log.TargetType,
		"content": log.Content, "performance_tags": log.PerformanceTags,
		"breakfast_rating": log.BreakfastRating, "lunch_rating": log.LunchRating,
		"dinner_rating": log.DinnerRating, "nap_duration": log.NapDuration,
		"sleep_quality": log.SleepQuality, "morning_mood": log.MorningMood,
		"afternoon_mood": log.AfternoonMood, "sync_to_feed": log.SyncToFeed,
		"created_at": log.CreatedAt.UTC().Format(time.RFC3339),
	}

	var creator domain.User
	if err := s.db.WithContext(ctx).First(&creator, log.CreatorID).Error; err == nil {
		m["creator"] = map[string]any{"id": creator.ID, "nickname": creator.Nickname, "avatar": creator.Avatar}
	}

	var media []domain.GrowthMedia
	s.db.WithContext(ctx).Where("log_id = ?", logID).Order("sort_num ASC, id ASC").Find(&media)
	m["media"] = media

	var likesCount int64
	s.db.WithContext(ctx).Model(&domain.GrowthLike{}).Where("log_id = ?", logID).Count(&likesCount)
	var myLike int64
	s.db.WithContext(ctx).Model(&domain.GrowthLike{}).Where("log_id = ? AND user_id = ?", logID, pc.UserID).Count(&myLike)
	m["likes_count"] = likesCount
	m["is_liked"] = myLike > 0

	var commentsCount int64
	s.db.WithContext(ctx).Model(&domain.GrowthComment{}).Where("log_id = ?", logID).Count(&commentsCount)
	m["comments_count"] = commentsCount

	return m, nil
}

func (s *ParentPortalService) LikeGrowth(ctx context.Context, pc *ParentPortalContext, logID uint64) (map[string]any, error) {
	if _, err := s.verifyGrowthAccess(ctx, pc, logID); err != nil { return nil, err }
	var existing domain.GrowthLike
	err := s.db.WithContext(ctx).Where("log_id = ? AND user_id = ?", logID, pc.UserID).First(&existing).Error
	if err == nil { return map[string]any{"is_liked": true, "likes_count": int64(0)}, nil } // already liked, idempotent
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) { return nil, err }
	s.db.WithContext(ctx).Create(&domain.GrowthLike{LogID: logID, UserID: pc.UserID})
	var count int64
	s.db.WithContext(ctx).Model(&domain.GrowthLike{}).Where("log_id = ?", logID).Count(&count)
	return map[string]any{"is_liked": true, "likes_count": count}, nil
}

func (s *ParentPortalService) UnlikeGrowth(ctx context.Context, pc *ParentPortalContext, logID uint64) (map[string]any, error) {
	if _, err := s.verifyGrowthAccess(ctx, pc, logID); err != nil { return nil, err }
	s.db.WithContext(ctx).Where("log_id = ? AND user_id = ?", logID, pc.UserID).Delete(&domain.GrowthLike{})
	var count int64
	s.db.WithContext(ctx).Model(&domain.GrowthLike{}).Where("log_id = ?", logID).Count(&count)
	return map[string]any{"is_liked": false, "likes_count": count}, nil
}

func (s *ParentPortalService) ListGrowthComments(ctx context.Context, pc *ParentPortalContext, logID uint64) (map[string]any, error) {
	if _, err := s.verifyGrowthAccess(ctx, pc, logID); err != nil { return nil, err }
	var rows []domain.GrowthComment
	if err := s.db.WithContext(ctx).Preload("User").Where("log_id = ?", logID).Order("created_at ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	list := make([]map[string]any, len(rows))
	for i, r := range rows {
		u := map[string]any{}
		if r.User != nil { u = map[string]any{"id": r.User.ID, "nickname": r.User.Nickname, "avatar": r.User.Avatar} }
		list[i] = map[string]any{"id": r.ID, "content": r.Content, "parent_id": r.ParentID, "user": u, "created_at": r.CreatedAt.UTC().Format(time.RFC3339)}
	}
	return map[string]any{"list": list, "total": len(rows)}, nil
}

func (s *ParentPortalService) CreateGrowthComment(ctx context.Context, pc *ParentPortalContext, logID uint64, content string, parentID *uint64) (map[string]any, error) {
	if strings.TrimSpace(content) == "" { return nil, ErrParentInvalidParams }
	if _, err := s.verifyGrowthAccess(ctx, pc, logID); err != nil { return nil, err }
	row := &domain.GrowthComment{LogID: logID, UserID: pc.UserID, Content: strings.TrimSpace(content)}
	if parentID != nil { row.ParentID = parentID }
	if err := s.db.WithContext(ctx).Create(row).Error; err != nil { return nil, err }
	return map[string]any{"id": row.ID, "created_at": row.CreatedAt.UTC().Format(time.RFC3339)}, nil
}

func (s *ParentPortalService) ListGrowthMedia(ctx context.Context, pc *ParentPortalContext, year, month int) (map[string]any, error) {
	studentIDs := studentIDs(pc.Students)
	db := s.db.WithContext(ctx).Model(&domain.GrowthMedia{}).
		Where("kindergarten_id = ?", pc.KindergartenID)
	if year > 0 && month > 0 {
		start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
		end := start.AddDate(0, 1, 0)
		db = db.Where("created_at >= ? AND created_at < ?", start, end)
	}
	// Filter: media belongs to class-level logs OR individual logs for these students
	db = db.Where(
		"log_id IN (SELECT id FROM growth_daily_logs WHERE kindergarten_id = ? AND (target_type = 1 OR id IN (SELECT log_id FROM growth_log_students WHERE student_id IN ?)))",
		pc.KindergartenID, studentIDs,
	)
	var rows []domain.GrowthMedia
	if err := db.Order("created_at DESC").Find(&rows).Error; err != nil { return nil, err }
	return map[string]any{"list": rows, "total": len(rows), "year": year, "month": month}, nil
}

func (s *ParentPortalService) GetGrowthReport(ctx context.Context, pc *ParentPortalContext, studentID uint64, reportType string) (map[string]any, error) {
	if studentID == 0 { studentID = pc.CurrentStudent.ID }
	db := s.db.WithContext(ctx).Model(&domain.GrowthReport{}).
		Where("student_id = ?", studentID)
	if reportType != "" { db = db.Where("report_type = ?", reportType) }
	var report domain.GrowthReport
	if err := db.Order("created_at DESC").First(&report).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return map[string]any{}, nil }
		return nil, err
	}
	var dims []domain.GrowthReportDimension
	s.db.WithContext(ctx).Where("report_id = ?", report.ID).Find(&dims)
	return map[string]any{"report": report, "dimensions": dims}, nil
}

// ==================== Tasks ====================

func (s *ParentPortalService) ListTasks(ctx context.Context, pc *ParentPortalContext, page, pageSize int) (map[string]any, error) {
	classIDs := make([]uint64, len(pc.Students))
	for i, st := range pc.Students { classIDs[i] = st.ClassID }
	if len(classIDs) == 0 { return map[string]any{"list": []any{}, "total": int64(0), "page": page, "page_size": pageSize}, nil }

	db := s.db.WithContext(ctx).Model(&domain.ParentTask{}).
		Where("kindergarten_id = ? AND class_id IN ? AND status = ?", pc.KindergartenID, classIDs, 1)

	var total int64
	if err := db.Count(&total).Error; err != nil { return nil, err }

	offset := (page - 1) * pageSize
	var rows []domain.ParentTask
	if err := db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil { return nil, err }

	list := make([]map[string]any, len(rows))
	for i, r := range rows {
		// Check if current student has checked in today
		var checkedIn bool
		s.db.WithContext(ctx).Model(&domain.TaskCheckIn{}).
			Where("task_id = ? AND student_id = ? AND date = ?", r.ID, pc.CurrentStudent.ID, time.Now().Format("2006-01-02")).
			Select("count(*) > 0").Scan(&checkedIn)

		var checkinCount int64
		s.db.WithContext(ctx).Model(&domain.TaskCheckIn{}).Where("task_id = ?", r.ID).Count(&checkinCount)

		list[i] = map[string]any{
			"id": r.ID, "title": r.Title, "description": r.Description,
			"intro": r.Intro, "frequency": r.Frequency, "points": r.Points,
			"start_date": r.StartDate.Format("2006-01-02"), "end_date": r.EndDate.Format("2006-01-02"),
			"status": r.Status, "checked_in_today": checkedIn,
			"checkin_count": checkinCount,
			"created_at": r.CreatedAt.UTC().Format(time.RFC3339),
		}
	}
	return map[string]any{"list": list, "total": total, "page": page, "page_size": pageSize}, nil
}

func (s *ParentPortalService) GetTask(ctx context.Context, pc *ParentPortalContext, id uint64) (map[string]any, error) {
	var t domain.ParentTask
	if err := s.db.WithContext(ctx).Where("id = ? AND kindergarten_id = ?", id, pc.KindergartenID).First(&t).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil, ErrParentNotFound }
		return nil, err
	}
	var checkedIn bool
	s.db.WithContext(ctx).Model(&domain.TaskCheckIn{}).
		Where("task_id = ? AND student_id = ? AND date = ?", id, pc.CurrentStudent.ID, time.Now().Format("2006-01-02")).
		Select("count(*) > 0").Scan(&checkedIn)
	return map[string]any{
		"id": t.ID, "title": t.Title, "description": t.Description, "intro": t.Intro,
		"requirements": t.Requirements, "rewards": t.Rewards,
		"frequency": t.Frequency, "points": t.Points,
		"start_date": t.StartDate.Format("2006-01-02"), "end_date": t.EndDate.Format("2006-01-02"),
		"status": t.Status, "checked_in_today": checkedIn,
		"created_at": t.CreatedAt.UTC().Format(time.RFC3339),
	}, nil
}

func (s *ParentPortalService) CheckinTask(ctx context.Context, pc *ParentPortalContext, taskID uint64, content, imageURL string) (map[string]any, error) {
	var t domain.ParentTask
	if err := s.db.WithContext(ctx).Where("id = ? AND kindergarten_id = ? AND status = ?", taskID, pc.KindergartenID, 1).First(&t).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil, ErrParentNotFound }
		return nil, err
	}
	today := time.Now().Format("2006-01-02")
	var existing int64
	s.db.WithContext(ctx).Model(&domain.TaskCheckIn{}).Where("task_id = ? AND student_id = ? AND date = ?", taskID, pc.CurrentStudent.ID, today).Count(&existing)
	if existing > 0 { return nil, ErrParentInvalidParams } // duplicate checkin

	checkin := &domain.TaskCheckIn{
		TaskID: taskID, StudentID: pc.CurrentStudent.ID, UserID: pc.UserID,
		Date: time.Now().Truncate(24 * time.Hour), Content: content, ImageURL: imageURL,
		Points: t.Points,
	}
	if err := s.db.WithContext(ctx).Create(checkin).Error; err != nil { return nil, err }
	return map[string]any{
		"id": checkin.ID, "points_earned": t.Points,
		"message": fmt.Sprintf("+%d积分!", t.Points),
		"created_at": checkin.CreatedAt.UTC().Format(time.RFC3339),
	}, nil
}

func (s *ParentPortalService) CheckinHistory(ctx context.Context, pc *ParentPortalContext, taskID uint64) (map[string]any, error) {
	var rows []domain.TaskCheckIn
	if err := s.db.WithContext(ctx).Preload("Student").
		Where("task_id = ? AND student_id = ?", taskID, pc.CurrentStudent.ID).
		Order("date DESC").Find(&rows).Error; err != nil { return nil, err }
	list := make([]map[string]any, len(rows))
	for i, r := range rows {
		studentName := ""
		if r.Student != nil { studentName = r.Student.Name }
		list[i] = map[string]any{
			"id": r.ID, "date": r.Date.Format("2006-01-02"), "content": r.Content,
			"image_url": r.ImageURL, "points": r.Points, "student_name": studentName,
			"created_at": r.CreatedAt.UTC().Format(time.RFC3339),
		}
	}
	return map[string]any{"list": list, "total": len(rows)}, nil
}

// ==================== Points & Achievements ====================

func (s *ParentPortalService) PointsOverview(ctx context.Context, pc *ParentPortalContext) (map[string]any, error) {
	if pc.CurrentStudent == nil { return nil, ErrParentInvalidParams }
	var account domain.PointAccount
	s.db.WithContext(ctx).Where("student_id = ?", pc.CurrentStudent.ID).First(&account)
	totalPoints := account.TotalPoints - account.UsedPoints
	level := 1
	levelName := "新手上路"
	if totalPoints >= 500 { level = 3; levelName = "积分达人" } else if totalPoints >= 200 { level = 2; levelName = "活跃家长" }
	thisWeek := time.Now().AddDate(0, 0, -int(time.Now().Weekday())+1)
	var weekPoints int64
	s.db.WithContext(ctx).Model(&domain.PointTransaction{}).
		Where("student_id = ? AND created_at >= ? AND points > 0", pc.CurrentStudent.ID, thisWeek).Select("COALESCE(SUM(points), 0)").Scan(&weekPoints)
	return map[string]any{
		"total_points": totalPoints, "level": level, "level_name": levelName,
		"this_week": weekPoints, "this_month": int64(0),
	}, nil
}

func (s *ParentPortalService) PointsHistory(ctx context.Context, pc *ParentPortalContext, page, pageSize int) (map[string]any, error) {
	if pc.CurrentStudent == nil { return nil, ErrParentInvalidParams }
	offset := (page - 1) * pageSize
	var total int64
	s.db.WithContext(ctx).Model(&domain.PointTransaction{}).Where("student_id = ?", pc.CurrentStudent.ID).Count(&total)
	var rows []domain.PointTransaction
	s.db.WithContext(ctx).Where("student_id = ?", pc.CurrentStudent.ID).Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&rows)
	return map[string]any{"list": rows, "total": total, "page": page, "page_size": pageSize}, nil
}

func (s *ParentPortalService) Leaderboard(ctx context.Context, pc *ParentPortalContext) (map[string]any, error) {
	if pc.Class == nil { return map[string]any{"list": []any{}, "my_rank": 0, "my_points": 0}, nil }
	var rows []struct {
		StudentID   uint64 `gorm:"column:student_id"`
		TotalPoints int64  `gorm:"column:total_points"`
		StudentName string `gorm:"column:student_name"`
	}
	s.db.WithContext(ctx).Raw(`
		SELECT pa.student_id, COALESCE(pa.total_points - pa.used_points, 0) AS total_points, s.name AS student_name
		FROM point_accounts pa INNER JOIN students s ON s.id = pa.student_id
		WHERE s.class_id = ? AND s.status = 1
		ORDER BY total_points DESC LIMIT 20`, pc.Class.ID).Scan(&rows)
	list := make([]map[string]any, len(rows))
	myRank, myPoints := 0, int64(0)
	for i, r := range rows {
		list[i] = map[string]any{"rank": i + 1, "student_id": r.StudentID, "student_name": r.StudentName, "total_points": r.TotalPoints}
		if r.StudentID == pc.CurrentStudent.ID { myRank = i + 1; myPoints = r.TotalPoints }
	}
	return map[string]any{"period": "weekly", "list": list, "my_rank": myRank, "my_points": myPoints}, nil
}

func (s *ParentPortalService) ListAchievements(ctx context.Context, pc *ParentPortalContext) (map[string]any, error) {
	if pc.CurrentStudent == nil { return nil, ErrParentInvalidParams }
	var earnedIDs []uint64
	s.db.WithContext(ctx).Model(&domain.StudentAchievement{}).Where("student_id = ?", pc.CurrentStudent.ID).Pluck("achievement_id", &earnedIDs)
	earnedSet := make(map[uint64]bool, len(earnedIDs))
	for _, id := range earnedIDs { earnedSet[id] = true }

	var all []domain.Achievement
	s.db.WithContext(ctx).Find(&all)
	earned := make([]map[string]any, 0)
	notEarned := make([]map[string]any, 0)
	for _, a := range all {
		m := map[string]any{"id": a.ID, "name": a.Name, "icon": a.Icon, "category": a.Category, "points": a.Points}
		if earnedSet[a.ID] { earned = append(earned, m) } else { notEarned = append(notEarned, m) }
	}
	return map[string]any{"earned": earned, "not_earned": notEarned, "earned_count": len(earned), "total_count": len(all)}, nil
}

// ==================== Feedback ====================

type ParentFeedbackInput struct {
	Category string `json:"category"`
	Content  string `json:"content"`
	Contact  string `json:"contact"`
	Images   string `json:"images"`
}

func (s *ParentPortalService) ListFeedback(ctx context.Context, pc *ParentPortalContext, page, pageSize int) (map[string]any, error) {
	offset := (page - 1) * pageSize
	var total int64
	s.db.WithContext(ctx).Model(&domain.Feedback{}).Where("kindergarten_id = ? AND user_id = ?", pc.KindergartenID, pc.UserID).Count(&total)
	var rows []domain.Feedback
	if err := s.db.WithContext(ctx).Where("kindergarten_id = ? AND user_id = ?", pc.KindergartenID, pc.UserID).Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	list := make([]map[string]any, len(rows))
	for i, r := range rows {
		statusLabel := "已提交"
		if r.Status == 1 { statusLabel = "已回复" } else if r.Status == 2 { statusLabel = "已关闭" }
		list[i] = map[string]any{
			"id": r.ID, "category": r.Category, "content": r.Content,
			"status": r.Status, "status_label": statusLabel,
			"reply": r.Reply, "created_at": r.CreatedAt.UTC().Format(time.RFC3339),
		}
	}
	return map[string]any{"list": list, "total": total, "page": page, "page_size": pageSize}, nil
}

func (s *ParentPortalService) CreateFeedback(ctx context.Context, pc *ParentPortalContext, req ParentFeedbackInput) (map[string]any, error) {
	if strings.TrimSpace(req.Content) == "" { return nil, ErrParentInvalidParams }
	fb := &domain.Feedback{
		KindergartenID: pc.KindergartenID, UserID: pc.UserID,
		Category: req.Category, Content: strings.TrimSpace(req.Content),
		Contact: req.Contact, Images: req.Images, Status: 0,
	}
	if err := s.db.WithContext(ctx).Create(fb).Error; err != nil { return nil, err }
	return map[string]any{"id": fb.ID, "status": 0, "status_label": "已提交", "created_at": fb.CreatedAt.UTC().Format(time.RFC3339)}, nil
}

// ==================== Student Bind/Unbind ====================

func (s *ParentPortalService) BindStudent(ctx context.Context, pc *ParentPortalContext, inviteCode string) (map[string]any, error) {
	if strings.TrimSpace(inviteCode) == "" { return nil, ErrParentInvalidParams }
	// Find student by invite code (stored as student_no or dedicated field)
	var student domain.Student
	if err := s.db.WithContext(ctx).Where("student_no = ? AND kindergarten_id = ?", inviteCode, pc.KindergartenID).First(&student).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil, ErrParentNotFound }
		return nil, err
	}
	// Check not already bound
	var existing int64
	s.db.WithContext(ctx).Model(&domain.StudentParent{}).Where("student_id = ? AND user_id = ?", student.ID, pc.UserID).Count(&existing)
	if existing > 0 { return nil, ErrParentInvalidParams }
	sp := &domain.StudentParent{StudentID: student.ID, UserID: pc.UserID, Relation: "other", IsPrimary: false}
	if err := s.db.WithContext(ctx).Create(sp).Error; err != nil { return nil, err }
	return map[string]any{"id": sp.ID, "student_id": student.ID, "student_name": student.Name, "created_at": sp.CreatedAt.UTC().Format(time.RFC3339)}, nil
}

func (s *ParentPortalService) UnbindStudent(ctx context.Context, pc *ParentPortalContext, id uint64) error {
	res := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, pc.UserID).Delete(&domain.StudentParent{})
	if res.RowsAffected == 0 { return ErrParentNotFound }
	return nil
}

// ==================== Settings ====================

func (s *ParentPortalService) GetSettings(ctx context.Context, pc *ParentPortalContext) (map[string]any, error) {
	return map[string]any{
		"notification": map[string]any{"notice_push": true, "leave_push": true, "pickup_push": true, "growth_push": true, "task_push": true, "announcement_push": true},
		"privacy":      map[string]any{"show_phone_to_teacher": true, "allow_growth_share": true},
	}, nil
}

func (s *ParentPortalService) UpdateSettings(ctx context.Context, pc *ParentPortalContext, settings map[string]any) error {
	// Settings are stored in-memory/session for now; persistent storage can be added later
	return nil
}

// ==================== Home Overview ====================

func (s *ParentPortalService) HomeOverview(ctx context.Context, pc *ParentPortalContext) (map[string]any, error) {
	if pc.CurrentStudent == nil { return nil, ErrParentInvalidParams }
	now := time.Now()
	today := now.Format("2006-01-02")
	cn := ""; if pc.Class != nil { cn = pc.Class.Name }

	studentsList := make([]map[string]any, len(pc.Students))
	for i, st := range pc.Students {
		cname := ""; if st.Class != nil { cname = st.Class.Name }
		studentsList[i] = map[string]any{"id": st.ID, "name": st.Name, "avatar": st.Avatar, "class_name": cname, "class_id": st.ClassID}
	}

	// Today summary
	var attendance domain.Attendance
	err := s.db.WithContext(ctx).Where("student_id = ? AND date = ?", pc.CurrentStudent.ID, today).First(&attendance).Error
	attendanceStatus := "not_arrived"
	attendanceLabel := "未入园"
	arriveTime := ""
	if err == nil {
		if attendance.Status == domain.AttendancePresent { attendanceStatus = "present"; attendanceLabel = "已入园" }
		if !attendance.CheckInTime.IsZero() { arriveTime = attendance.CheckInTime.Format("15:04") }
	}

	var summary domain.StudentDailySummary
	s.db.WithContext(ctx).Where("student_id = ? AND date = ?", pc.CurrentStudent.ID, today).First(&summary)

	var newGrowthCount int64
	if pc.Class != nil {
		s.db.WithContext(ctx).Model(&domain.GrowthDailyLog{}).Where("kindergarten_id = ? AND class_id = ? AND date = ?", pc.KindergartenID, pc.Class.ID, today).Count(&newGrowthCount)
	}
	var newNoticeCount int64
	if pc.Class != nil {
		s.db.WithContext(ctx).Model(&domain.Notice{}).Where("kindergarten_id = ? AND class_id = ? AND status = ? AND published_at >= ?", pc.KindergartenID, pc.Class.ID, domain.NoticeStatusPublished, today).Count(&newNoticeCount)
	}
	var pendingTasks int64
	if pc.Class != nil {
		s.db.WithContext(ctx).Model(&domain.ParentTask{}).Where("kindergarten_id = ? AND class_id = ? AND status = ?", pc.KindergartenID, pc.Class.ID, 1).Count(&pendingTasks)
	}

	var leave domain.LeaveRequest
	leaveStatus := map[string]any(nil)
	if err := s.db.WithContext(ctx).Where("student_id = ? AND start_date <= ? AND end_date >= ? AND status = ?", pc.CurrentStudent.ID, now, now, domain.LeaveStatusApproved).First(&leave).Error; err == nil {
		leaveStatus = map[string]any{"id": leave.ID, "reason": leave.Reason, "status": leave.Status, "status_label": "已同意"}
	}

	return map[string]any{
		"date": now.Format("2006-01-02"),
		"weekday": []string{"星期日","星期一","星期二","星期三","星期四","星期五","星期六"}[now.Weekday()],
		"current_student": map[string]any{"id": pc.CurrentStudent.ID, "name": pc.CurrentStudent.Name, "avatar": pc.CurrentStudent.Avatar, "class_name": cn, "class_id": pc.CurrentStudent.ClassID},
		"students": studentsList,
		"today_summary": map[string]any{
			"attendance_status": attendanceStatus, "attendance_label": attendanceLabel, "arrive_time": arriveTime,
			"meal_rating": summary.MealRating, "meal_label": mealRatingLabel(int(summary.MealRating)),
			"nap_duration": summary.NapDuration, "sleep_quality": summary.SleepQuality,
			"sleep_label": sleepQualityLabel(int(summary.SleepQuality)),
			"mood_rating": summary.MoodRating, "mood_label": moodRatingLabel(int(summary.MoodRating)),
			"performance_tag": summary.PerformanceTag,
			"new_growth_count": newGrowthCount, "new_notice_count": newNoticeCount,
			"pending_tasks": pendingTasks, "leave_status": leaveStatus,
		},
	}, nil
}

func mealRatingLabel(r int) string { switch r { case 3: return "很好"; case 2: return "一般"; case 1: return "较差"; default: return "" } }
func sleepQualityLabel(q int) string { switch q { case 3: return "很好"; case 2: return "一般"; case 1: return "较差"; default: return "" } }
func moodRatingLabel(m int) string { switch m { case 3: return "愉快"; case 2: return "平稳"; case 1: return "低落"; default: return "" } }

// ==================== Student Profile ====================

func (s *ParentPortalService) GetStudentProfile(ctx context.Context, pc *ParentPortalContext) (map[string]any, error) {
	if pc.CurrentStudent == nil { return nil, ErrParentInvalidParams }
	cn := ""; if pc.Class != nil { cn = pc.Class.Name }

	// Calculate age
	age := ""
	if !pc.CurrentStudent.Birthday.IsZero() {
		now := time.Now()
		years := now.Year() - pc.CurrentStudent.Birthday.Year()
		if now.Month() < pc.CurrentStudent.Birthday.Month() || (now.Month() == pc.CurrentStudent.Birthday.Month() && now.Day() < pc.CurrentStudent.Birthday.Day()) {
			years--
		}
		if years > 0 { age = fmt.Sprintf("%d岁", years) }
	}

	// Get parents list
	var parents []map[string]any
	var sps []domain.StudentParent
	s.db.WithContext(ctx).Preload("User").Where("student_id = ?", pc.CurrentStudent.ID).Find(&sps)
	for _, sp := range sps {
		p := map[string]any{"id": sp.ID, "relation": string(sp.Relation), "relation_label": RelationLabel(sp.Relation), "is_primary": sp.IsPrimary}
		if sp.User != nil { p["nickname"] = sp.User.Nickname; p["avatar"] = sp.User.Avatar; p["phone"] = maskPhone(sp.User.Phone) }
		parents = append(parents, p)
	}

	return map[string]any{
		"id": pc.CurrentStudent.ID, "name": pc.CurrentStudent.Name, "avatar": pc.CurrentStudent.Avatar,
		"gender": pc.CurrentStudent.Gender, "birthday": pc.CurrentStudent.Birthday, "age": age,
		"student_no": pc.CurrentStudent.StudentNo, "health_note": pc.CurrentStudent.HealthNote,
		"class_name": cn, "class_id": pc.CurrentStudent.ClassID,
		"address": pc.CurrentStudent.Address, "enroll_date": pc.CurrentStudent.EnrollDate,
		"parents": parents,
	}, nil
}

// ==================== Home / Announcements / Feeds ====================

func (s *ParentPortalService) ListAnnouncements(ctx context.Context, pc *ParentPortalContext, page, pageSize int) (map[string]any, error) {
	classIDs := make([]uint64, len(pc.Students))
	for i, st := range pc.Students { classIDs[i] = st.ClassID }
	if len(classIDs) == 0 { return map[string]any{"list": []any{}, "total": int64(0)}, nil }
	offset := (page - 1) * pageSize
	db := s.db.WithContext(ctx).Model(&domain.Announcement{}).
		Where("kindergarten_id = ? AND (class_id IN ? OR class_id = 0) AND status = ?", pc.KindergartenID, classIDs, 1)
	var total int64
	if err := db.Count(&total).Error; err != nil { return nil, err }
	var rows []domain.Announcement
	if err := db.Order("is_top DESC, published_at DESC").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil { return nil, err }
	list := make([]map[string]any, len(rows))
	for i, a := range rows {
		list[i] = map[string]any{"id": a.ID, "title": a.Title, "content": a.Content, "is_top": a.IsTop, "published_at": a.PublishedAt.Format("2006-01-02"), "created_at": a.CreatedAt.UTC().Format(time.RFC3339)}
	}
	return map[string]any{"list": list, "total": total}, nil
}

func (s *ParentPortalService) ListHomeFeeds(ctx context.Context, pc *ParentPortalContext, page, pageSize int) (map[string]any, error) {
	if len(pc.Students) == 0 {
		return map[string]any{"list": []any{}, "total": int64(0), "page": page, "page_size": pageSize}, nil
	}

	// 收集孩子所在的所有班级
	classIDs := make([]uint64, 0, len(pc.Students))
	seen := map[uint64]bool{}
	for _, st := range pc.Students {
		if !seen[st.ClassID] {
			seen[st.ClassID] = true
			classIDs = append(classIDs, st.ClassID)
		}
	}

	type feedItem struct {
		Type      string    `json:"type"`
		CreatedAt time.Time `json:"created_at"`
		Data      map[string]any
	}
	var items []feedItem

	// 1) 通知（最近 3 条）
	var notices []domain.Notice
	s.db.WithContext(ctx).Model(&domain.Notice{}).
		Where("kindergarten_id = ? AND class_id IN ? AND status = ?", pc.KindergartenID, classIDs, domain.NoticeStatusPublished).
		Order("published_at DESC").Limit(3).Find(&notices)
	noticeIDs := make([]uint64, len(notices))
	for i, n := range notices { noticeIDs[i] = n.ID }
	readStatus, _ := s.notice.CheckReadStatus(ctx, pc.UserID, noticeIDs)

	for _, n := range notices {
		ts := n.CreatedAt
		if n.PublishedAt != nil { ts = *n.PublishedAt }
		items = append(items, feedItem{
			Type: "notice", CreatedAt: ts,
			Data: map[string]any{
				"id":    n.ID,
				"title": n.Title,
				"notice": map[string]any{
					"id": n.ID, "title": n.Title,
					"content_preview": growthContentPreview(n.Content, 80),
					"creator":          map[string]any{"nickname": ""},
					"is_read":          readStatus[n.ID],
				},
			},
		})
	}

	// 2) 成长记录（最近 3 条）
	var logs []domain.GrowthDailyLog
	s.db.WithContext(ctx).Preload("Creator").Model(&domain.GrowthDailyLog{}).
		Where("kindergarten_id = ? AND class_id IN ?", pc.KindergartenID, classIDs).
		Order("date DESC, id DESC").Limit(3).Find(&logs)
	for _, l := range logs {
		creatorNick := ""
		if l.Creator != nil { creatorNick = l.Creator.Nickname }
		items = append(items, feedItem{
			Type: "growth", CreatedAt: l.CreatedAt,
			Data: map[string]any{
				"id": l.ID,
				"growth": map[string]any{
					"id":      l.ID,
					"content": l.Content,
					"creator": map[string]any{"nickname": creatorNick},
				},
			},
		})
	}

	// 3) 按时间倒序，取前 pageSize 条
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	total := int64(len(items))
	offset := (page - 1) * pageSize
	if offset >= len(items) {
		return map[string]any{"list": []any{}, "total": total, "page": page, "page_size": pageSize}, nil
	}
	end := offset + pageSize
	if end > len(items) { end = len(items) }
	pageItems := items[offset:end]

	list := make([]map[string]any, len(pageItems))
	for i, it := range pageItems {
		list[i] = map[string]any{
			"id":         it.Data["id"],
			"type":       it.Type,
			it.Type:      it.Data[it.Type],
			"created_at": it.CreatedAt.UTC().Format(time.RFC3339),
		}
	}
	return map[string]any{"list": list, "total": total, "page": page, "page_size": pageSize}, nil
}

// ==================== Class Info ====================

func (s *ParentPortalService) ClassInfo(ctx context.Context, pc *ParentPortalContext) (map[string]any, error) {
	if pc.Class == nil { return nil, ErrParentInvalidParams }
	var studentCount int64
	s.db.WithContext(ctx).Model(&domain.Student{}).Where("kindergarten_id = ? AND class_id = ? AND status = ?", pc.KindergartenID, pc.Class.ID, 1).Count(&studentCount)
	var teachers []map[string]any
	var cts []domain.ClassTeacher
	s.db.WithContext(ctx).Preload("User").Where("class_id = ? AND status = ?", pc.Class.ID, 1).Find(&cts)
	for _, ct := range cts {
		t := map[string]any{"id": ct.UserID, "title": string(ct.Title)}
		if ct.User != nil { t["nickname"] = ct.User.Nickname; t["avatar"] = ct.User.Avatar }
		teachers = append(teachers, t)
	}
	return map[string]any{"id": pc.Class.ID, "name": pc.Class.Name, "grade": pc.Class.Grade, "academic_year": pc.Class.AcademicYear, "student_count": studentCount, "teachers": teachers}, nil
}

// ==================== Student Profile ====================

func (s *ParentPortalService) UpdateStudentProfile(ctx context.Context, pc *ParentPortalContext, req map[string]any) error {
	if pc.CurrentStudent == nil { return ErrParentInvalidParams }
	updates := map[string]any{}
	if v, ok := req["avatar"]; ok { updates["avatar"] = v }
	if v, ok := req["health_note"]; ok { updates["health_note"] = v }
	if v, ok := req["address"]; ok { updates["address"] = v }
	if len(updates) == 0 { return nil }
	return s.db.WithContext(ctx).Model(pc.CurrentStudent).Updates(updates).Error
}

// ListClassCalendar 查询当前学生所在班级的日历事件。
func (s *ParentPortalService) ListClassCalendar(ctx context.Context, pc *ParentPortalContext, start, end string) (map[string]any, error) {
	if pc.CurrentStudent == nil {
		return map[string]any{"list": []any{}, "total": 0}, nil
	}
	rows, err := s.calendar.FindEvents(ctx, pc.KindergartenID, pc.CurrentStudent.ClassID, start, end)
	if err != nil {
		return nil, err
	}
	return map[string]any{"list": rows, "total": len(rows)}, nil
}
