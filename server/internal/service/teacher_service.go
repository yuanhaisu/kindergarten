package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/pkg/pagination"
)

type TeacherService struct {
	db             *gorm.DB
	teachers       *repository.TeacherRepo
	classes        *repository.ClassRepo
	users          *repository.UserRepo
	roles          *repository.UserRoleRepo
	kg             *repository.KindergartenRepo
	oplog          *repository.OperationLogRepo
	invitationRepo *repository.TeacherInvitationRepo
	wx             *auth.WeChatClient
}

func NewTeacherService(
	db *gorm.DB,
	teachers *repository.TeacherRepo,
	classes *repository.ClassRepo,
	users *repository.UserRepo,
	roles *repository.UserRoleRepo,
	kg *repository.KindergartenRepo,
	oplog *repository.OperationLogRepo,
	invitationRepo *repository.TeacherInvitationRepo,
	wx *auth.WeChatClient,
) *TeacherService {
	return &TeacherService{
		db: db, teachers: teachers, classes: classes, users: users, roles: roles, kg: kg, oplog: oplog,
		invitationRepo: invitationRepo,
		wx:             wx,
	}
}

func (s *TeacherService) logOp(ctx context.Context, kindergartenID, userID uint64, module, action, targetType string, targetID uint64, detail, ip, ua string) {
	_ = s.oplog.Create(ctx, &domain.OperationLog{
		KindergartenID: kindergartenID,
		UserID:         userID,
		Module:         module,
		Action:         action,
		TargetType:     targetType,
		TargetID:       targetID,
		Detail:         detail,
		IP:             ip,
		UserAgent:      truncateUATeacher(ua),
	})
}

func truncateUATeacher(ua string) string {
	if utf8.RuneCountInString(ua) <= 300 {
		return ua
	}
	return string([]rune(ua)[:300])
}

var phoneReTeacher = regexp.MustCompile(`^1[3-9]\d{9}$`)

func teacherStatusLabel(st int8) string {
	if st == 2 {
		return "离职"
	}
	return "在职"
}

type teacherClassBrief struct {
	ClassID    uint64 `json:"class_id"`
	ClassName  string `json:"class_name"`
	Title      string `json:"title"`
	TitleLabel string `json:"title_label"`
}

type teacherListItem struct {
	ID          uint64              `json:"id"`
	Nickname    string              `json:"nickname"`
	Avatar      string              `json:"avatar"`
	Phone       string              `json:"phone"`
	Gender      int                 `json:"gender"`
	Classes     []teacherClassBrief `json:"classes"`
	Status      int8                `json:"status"`
	StatusLabel string              `json:"status_label"`
	JoinDate    string              `json:"join_date"`
	CreatedAt   string              `json:"created_at"`
}

func (s *TeacherService) ListTeachers(
	ctx context.Context,
	kindergartenID uint64,
	q pagination.Query,
	classID *uint64,
	status *int8,
) ([]teacherListItem, int64, int, int, error) {
	if kindergartenID == 0 {
		return nil, 0, 0, 0, fmt.Errorf("NO_KINDERGARTEN")
	}
	q.Normalize()
	ids, total, err := s.teachers.ListTeacherUserIDs(ctx, kindergartenID, q.Keyword, classID, status, q.Offset(), q.PageSize)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	if len(ids) == 0 {
		return []teacherListItem{}, total, q.Page, q.PageSize, nil
	}
	profiles := make(map[uint64]*domain.TeacherProfile, len(ids))
	for _, uid := range ids {
		p, err := s.teachers.FindProfile(ctx, kindergartenID, uid)
		if err != nil {
			return nil, 0, 0, 0, err
		}
		if p != nil {
			profiles[uid] = p
		}
	}
	ctRows, err := s.teachers.ListClassTeachersForUsersInKindergarten(ctx, kindergartenID, ids)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	byUser := make(map[uint64][]domain.ClassTeacher)
	for _, ct := range ctRows {
		byUser[ct.UserID] = append(byUser[ct.UserID], ct)
	}
	out := make([]teacherListItem, 0, len(ids))
	for _, uid := range ids {
		u, err := s.users.FindByID(ctx, uid)
		if err != nil || u == nil {
			continue
		}
		p := profiles[uid]
		st := int8(1)
		var jd time.Time
		if p != nil {
			st = p.Status
			jd = p.JoinDate
		}
		jds := ""
		if !jd.IsZero() {
			jds = jd.Format("2006-01-02")
		}
		clsBriefs := make([]teacherClassBrief, 0)
		for _, ct := range byUser[uid] {
			cn := ""
			if ct.Class != nil {
				cn = ct.Class.Name
			}
			clsBriefs = append(clsBriefs, teacherClassBrief{
				ClassID: ct.ClassID, ClassName: cn,
				Title: string(ct.Title), TitleLabel: titleLabel(ct.Title),
			})
		}
		out = append(out, teacherListItem{
			ID: uid, Nickname: u.Nickname, Avatar: u.Avatar, Phone: maskPhone(u.Phone),
			Gender: u.Gender, Classes: clsBriefs, Status: st, StatusLabel: teacherStatusLabel(st),
			JoinDate: jds, CreatedAt: u.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return out, total, q.Page, q.PageSize, nil
}

type teacherDetailStats struct {
	TotalGrowthLogs       int64   `json:"total_growth_logs"`
	TotalNotices          int64   `json:"total_notices"`
	AvgParentSatisfaction float64 `json:"avg_parent_satisfaction"`
}

type teacherDetailResp struct {
	ID            uint64              `json:"id"`
	Nickname      string              `json:"nickname"`
	RealName      string              `json:"real_name"`
	Avatar        string              `json:"avatar"`
	Phone         string              `json:"phone"`
	Gender        int                 `json:"gender"`
	IDCardNumber  string              `json:"id_card_number,omitempty"`
	Education     string              `json:"education,omitempty"`
	Major         string              `json:"major,omitempty"`
	CertificateNo string              `json:"certificate_no,omitempty"`
	Classes       []teacherClassBrief `json:"classes"`
	Status        int8                `json:"status"`
	StatusLabel   string              `json:"status_label"`
	JoinDate      string              `json:"join_date"`
	Stats         teacherDetailStats  `json:"stats"`
	CreatedAt     string              `json:"created_at"`
}

func (s *TeacherService) GetTeacher(ctx context.Context, kindergartenID, teacherUserID uint64) (*teacherDetailResp, error) {
	if kindergartenID == 0 {
		return nil, fmt.Errorf("NO_KINDERGARTEN")
	}
	ur, err := s.roles.FindTeacherInKindergarten(ctx, teacherUserID, kindergartenID)
	if err != nil {
		return nil, err
	}
	if ur == nil {
		return nil, fmt.Errorf("NOT_FOUND")
	}
	p, err := s.teachers.FindProfile(ctx, kindergartenID, teacherUserID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("NOT_FOUND")
	}
	u, err := s.users.FindByID(ctx, teacherUserID)
	if err != nil || u == nil {
		return nil, fmt.Errorf("NOT_FOUND")
	}
	ctRows, err := s.teachers.ListClassTeachersForUsersInKindergarten(ctx, kindergartenID, []uint64{teacherUserID})
	if err != nil {
		return nil, err
	}
	clsBriefs := make([]teacherClassBrief, 0, len(ctRows))
	for _, ct := range ctRows {
		cn := ""
		if ct.Class != nil {
			cn = ct.Class.Name
		}
		clsBriefs = append(clsBriefs, teacherClassBrief{
			ClassID: ct.ClassID, ClassName: cn,
			Title: string(ct.Title), TitleLabel: titleLabel(ct.Title),
		})
	}
	gn, _ := s.teachers.CountGrowthLogsByCreator(ctx, kindergartenID, teacherUserID)
	nn, _ := s.teachers.CountNoticesByCreator(ctx, kindergartenID, teacherUserID)
	jds := ""
	if !p.JoinDate.IsZero() {
		jds = p.JoinDate.Format("2006-01-02")
	}
	return &teacherDetailResp{
		ID: teacherUserID, Nickname: u.Nickname, RealName: p.RealName, Avatar: u.Avatar,
		Phone: u.Phone, Gender: u.Gender, IDCardNumber: u.IDCardNumber,
		Education: p.Education, Major: p.Major, CertificateNo: p.CertificateNo,
		Classes: clsBriefs, Status: p.Status, StatusLabel: teacherStatusLabel(p.Status),
		JoinDate:  jds,
		Stats:     teacherDetailStats{TotalGrowthLogs: gn, TotalNotices: nn, AvgParentSatisfaction: 0},
		CreatedAt: u.CreatedAt.UTC().Format(time.RFC3339),
	}, nil
}

type classAssignmentItem struct {
	ClassID uint64 `json:"class_id" binding:"required"`
	Title   string `json:"title" binding:"required"`
}

type CreateTeacherReq struct {
	RealName         string                `json:"real_name" binding:"required"`
	Nickname         string                `json:"nickname" binding:"required"`
	Phone            string                `json:"phone" binding:"required"`
	Gender           int                   `json:"gender"`
	Education        string                `json:"education"`
	Major            string                `json:"major"`
	CertificateNo    string                `json:"certificate_no"`
	JoinDate         string                `json:"join_date" binding:"required"`
	ClassAssignments []classAssignmentItem `json:"class_assignments"`
}

type createTeacherResp struct {
	ID         uint64 `json:"id"`
	InviteCode string `json:"invite_code"`
	ExpiresAt  string `json:"expires_at"`
	CreatedAt  string `json:"created_at"`
}

func (s *TeacherService) validateClassAssignments(
	ctx context.Context,
	kindergartenID uint64,
	items []classAssignmentItem,
) ([]domain.ClassTeacher, error) {
	seen := make(map[uint64]struct{})
	out := make([]domain.ClassTeacher, 0, len(items))
	for _, it := range items {
		if _, dup := seen[it.ClassID]; dup {
			return nil, fmt.Errorf("DUPLICATE_CLASS_IN_ASSIGNMENTS")
		}
		seen[it.ClassID] = struct{}{}
		title, err := parseTeacherTitle(it.Title)
		if err != nil {
			return nil, err
		}
		cl, err := s.classes.FindByID(ctx, kindergartenID, it.ClassID)
		if err != nil {
			return nil, err
		}
		if cl == nil {
			return nil, fmt.Errorf("CLASS_NOT_FOUND")
		}
		out = append(out, domain.ClassTeacher{ClassID: it.ClassID, Title: title, Status: 1})
	}
	return out, nil
}

func (s *TeacherService) CreateTeacher(
	ctx context.Context,
	kindergartenID, adminUserID uint64,
	req CreateTeacherReq,
	ip, ua string,
) (*createTeacherResp, error) {
	if kindergartenID == 0 {
		return nil, fmt.Errorf("NO_KINDERGARTEN")
	}
	if !phoneReTeacher.MatchString(req.Phone) {
		return nil, fmt.Errorf("INVALID_PHONE")
	}
	if len(req.ClassAssignments) == 0 {
		return nil, fmt.Errorf("CLASS_ASSIGNMENTS_REQUIRED")
	}
	jd, err := parseDate(req.JoinDate)
	if err != nil {
		return nil, fmt.Errorf("INVALID_JOIN_DATE")
	}
	kg, err := s.kg.FindByID(ctx, kindergartenID)
	if err != nil {
		return nil, err
	}
	if kg == nil {
		return nil, fmt.Errorf("KINDERGARTEN_NOT_FOUND")
	}
	existing, err := s.users.FindByPhone(ctx, req.Phone)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		ur, err := s.roles.FindTeacherInKindergarten(ctx, existing.ID, kindergartenID)
		if err != nil {
			return nil, err
		}
		if ur != nil {
			return nil, fmt.Errorf("DUPLICATE_TEACHER_PHONE")
		}
	}
	assignments, err := s.validateClassAssignments(ctx, kindergartenID, req.ClassAssignments)
	if err != nil {
		return nil, err
	}
	hash, err := HashPassword("KindyTeacher#1")
	if err != nil {
		return nil, err
	}
	var teacherID uint64
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var u *domain.User
		if existing != nil {
			u = existing
			if err := tx.Model(u).Updates(map[string]any{
				"nickname": strings.TrimSpace(req.Nickname),
				"gender":   req.Gender,
			}).Error; err != nil {
				return err
			}
		} else {
			u = &domain.User{
				Phone: req.Phone, Nickname: strings.TrimSpace(req.Nickname), Gender: req.Gender,
				PasswordHash: hash,
			}
			if err := tx.Create(u).Error; err != nil {
				return err
			}
		}
		teacherID = u.ID
		kid := kindergartenID
		nur := &domain.UserRole{
			UserID:           teacherID,
			Role:             "teacher",
			KindergartenID:   &kid,
			KindergartenName: kg.Name,
		}
		if err := tx.Create(nur).Error; err != nil {
			return err
		}
		prof := &domain.TeacherProfile{
			UserID: teacherID, KindergartenID: kindergartenID,
			RealName: strings.TrimSpace(req.RealName), Education: req.Education, Major: req.Major,
			CertificateNo: req.CertificateNo, JoinDate: jd, Status: 1,
		}
		if err := tx.Create(prof).Error; err != nil {
			return err
		}
		for i := range assignments {
			assignments[i].UserID = teacherID
			assignments[i].ID = 0
			if err := tx.Create(&assignments[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.logOp(ctx, kindergartenID, adminUserID, "teachers", "create", "teacher", teacherID,
		fmt.Sprintf(`{"phone":%q,"nickname":%q}`, req.Phone, req.Nickname), ip, ua)

	inviteCode := generateInviteCode()
	inv := &domain.TeacherInvitation{
		Token:          inviteCode,
		UserID:         teacherID,
		KindergartenID: kindergartenID,
		Status:         domain.TeacherInvitationPending,
		ExpiresAt:      time.Now().Add(7 * 24 * time.Hour),
	}
	if err := s.invitationRepo.Create(ctx, inv); err != nil {
		return nil, err
	}

	u2, err := s.users.FindByID(ctx, teacherID)
	expiry := inv.ExpiresAt.UTC().Format(time.RFC3339)
	if err != nil || u2 == nil {
		return &createTeacherResp{ID: teacherID, InviteCode: inviteCode, ExpiresAt: expiry, CreatedAt: time.Now().UTC().Format(time.RFC3339)}, nil
	}
	return &createTeacherResp{ID: teacherID, InviteCode: inviteCode, ExpiresAt: expiry, CreatedAt: u2.CreatedAt.UTC().Format(time.RFC3339)}, nil
}

// TeacherInvitationInfo 邀请码信息。
type TeacherInvitationInfo struct {
	Token     string `json:"token"`
	Status    int8   `json:"status"`
	ExpiresAt string `json:"expires_at"`
	CreatedAt string `json:"created_at"`
}

// GetTeacherInvitation 获取教师当前有效邀请码。
func (s *TeacherService) GetTeacherInvitation(ctx context.Context, kindergartenID, teacherUserID uint64) (*TeacherInvitationInfo, error) {
	inv, err := s.invitationRepo.FindPendingByUserID(ctx, teacherUserID, kindergartenID)
	if err != nil {
		return nil, err
	}
	if inv == nil {
		return nil, nil
	}
	return &TeacherInvitationInfo{
		Token:     inv.Token,
		Status:    inv.Status,
		ExpiresAt: inv.ExpiresAt.UTC().Format(time.RFC3339),
		CreatedAt: inv.CreatedAt.UTC().Format(time.RFC3339),
	}, nil
}

// RegenerateTeacherInvitation 重新生成邀请码（作废旧邀请，创建新邀请）。
func (s *TeacherService) RegenerateTeacherInvitation(ctx context.Context, kindergartenID, adminUserID, teacherUserID uint64) (*TeacherInvitationInfo, error) {
	ur, err := s.roles.FindTeacherInKindergarten(ctx, teacherUserID, kindergartenID)
	if err != nil {
		return nil, err
	}
	if ur == nil {
		return nil, fmt.Errorf("NOT_FOUND")
	}

	// 作废所有待使用的旧邀请
	old, err := s.invitationRepo.FindPendingByUserID(ctx, teacherUserID, kindergartenID)
	if err != nil {
		return nil, err
	}
	if old != nil {
		_ = s.invitationRepo.DeleteByToken(ctx, old.Token)
	}

	inviteCode := generateInviteCode()
	inv := &domain.TeacherInvitation{
		Token:          inviteCode,
		UserID:         teacherUserID,
		KindergartenID: kindergartenID,
		Status:         domain.TeacherInvitationPending,
		ExpiresAt:      time.Now().Add(7 * 24 * time.Hour),
	}
	if err := s.invitationRepo.Create(ctx, inv); err != nil {
		return nil, err
	}

	return &TeacherInvitationInfo{
		Token:     inv.Token,
		Status:    inv.Status,
		ExpiresAt: inv.ExpiresAt.UTC().Format(time.RFC3339),
		CreatedAt: inv.CreatedAt.UTC().Format(time.RFC3339),
	}, nil
}

// GenerateInviteQRCode 生成老师邀请小程序码（PNG 二进制），没有有效邀请码时自动创建。
func (s *TeacherService) GenerateInviteQRCode(ctx context.Context, kindergartenID, teacherUserID uint64) ([]byte, error) {
	inv, err := s.invitationRepo.FindPendingByUserID(ctx, teacherUserID, kindergartenID)
	if err != nil {
		return nil, err
	}
	if inv == nil {
		ur, err := s.roles.FindTeacherInKindergarten(ctx, teacherUserID, kindergartenID)
		if err != nil {
			return nil, err
		}
		if ur == nil {
			return nil, fmt.Errorf("NOT_FOUND")
		}
		code := generateInviteCode()
		inv = &domain.TeacherInvitation{
			Token:          code,
			UserID:         teacherUserID,
			KindergartenID: kindergartenID,
			Status:         domain.TeacherInvitationPending,
			ExpiresAt:      time.Now().Add(7 * 24 * time.Hour),
		}
		if err := s.invitationRepo.Create(ctx, inv); err != nil {
			return nil, err
		}
	}

	scene := "invite_code=" + inv.Token
	page := "pages/index/index"
	return s.wx.GetWXACodeUnlimit(ctx, scene, page)
}

// GenerateTaskShareCard 生成指向家长端任务详情的小程序码（PNG），供老师转发班级群。
//
// 必须走家长端凭据（GetParentWXACodeUnlimit）：用老师端凭据生成的码会把家长带进老师端小程序。
// scene 只用任务 id（`t=<id>`，≤32 字符）；envVersion 取配置 wechat.code_env_version。
func (s *TeacherService) GenerateTaskShareCard(
	ctx context.Context,
	kindergartenID, classID, taskID uint64,
) (map[string]any, error) {
	// 任务可见范围与老师端一致：本园 + 本班，已删除的任务不能继续转发。
	var task domain.ParentTask
	if err := s.db.WithContext(ctx).
		Where("id = ? AND kindergarten_id = ? AND class_id = ?", taskID, kindergartenID, classID).
		First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("NOT_FOUND")
		}
		return nil, err
	}

	const page = "pages/tasks/detail"
	scene := fmt.Sprintf("t=%d", taskID)
	envVersion := s.wx.CodeEnvVersion()
	png, err := s.wx.GetParentWXACodeUnlimit(ctx, scene, page, envVersion)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"task_id":      taskID,
		"page":         page,
		"scene":        scene,
		"env_version":  envVersion,
		"image_base64": base64.StdEncoding.EncodeToString(png),
	}, nil
}

func generateInviteCode() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("generateInviteCode: rand.Read failed: %v", err))
	}
	return hex.EncodeToString(b)
}

type UpdateTeacherReq struct {
	Nickname         string                `json:"nickname"`
	Phone            string                `json:"phone"`
	Gender           int                   `json:"gender"`
	Education        string                `json:"education"`
	ClassAssignments []classAssignmentItem `json:"class_assignments"`
}

type updateTeacherResp struct {
	ID        uint64 `json:"id"`
	UpdatedAt string `json:"updated_at"`
}

func (s *TeacherService) UpdateTeacher(
	ctx context.Context,
	kindergartenID, adminUserID, teacherUserID uint64,
	req UpdateTeacherReq,
	ip, ua string,
) (*updateTeacherResp, error) {
	if kindergartenID == 0 {
		return nil, fmt.Errorf("NO_KINDERGARTEN")
	}
	ur, err := s.roles.FindTeacherInKindergarten(ctx, teacherUserID, kindergartenID)
	if err != nil {
		return nil, err
	}
	if ur == nil {
		return nil, fmt.Errorf("NOT_FOUND")
	}
	p, err := s.teachers.FindProfile(ctx, kindergartenID, teacherUserID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("NOT_FOUND")
	}
	u, err := s.users.FindByID(ctx, teacherUserID)
	if err != nil || u == nil {
		return nil, fmt.Errorf("NOT_FOUND")
	}
	if nn := strings.TrimSpace(req.Nickname); nn != "" {
		u.Nickname = nn
	}
	if ph := strings.TrimSpace(req.Phone); ph != "" {
		if !phoneReTeacher.MatchString(ph) {
			return nil, fmt.Errorf("INVALID_PHONE")
		}
		u.Phone = ph
	}
	if req.Gender > 0 {
		u.Gender = req.Gender
	}
	if req.Education != "" {
		p.Education = req.Education
	}
	if err := s.users.Update(ctx, u); err != nil {
		return nil, err
	}
	if len(req.ClassAssignments) > 0 {
		assignments, err := s.validateClassAssignments(ctx, kindergartenID, req.ClassAssignments)
		if err != nil {
			return nil, err
		}
		err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			sub := tx.Model(&domain.Class{}).Select("id").Where("kindergarten_id = ?", kindergartenID)
			if err := tx.Unscoped().Where("user_id = ? AND class_id IN (?)", teacherUserID, sub).
				Delete(&domain.ClassTeacher{}).Error; err != nil {
				return err
			}
			for i := range assignments {
				assignments[i].UserID = teacherUserID
				assignments[i].ID = 0
				if err := tx.Create(&assignments[i]).Error; err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	if err := s.teachers.SaveProfile(ctx, p); err != nil {
		return nil, err
	}
	u3, _ := s.users.FindByID(ctx, teacherUserID)
	updated := time.Now().UTC().Format(time.RFC3339)
	if u3 != nil {
		updated = u3.UpdatedAt.UTC().Format(time.RFC3339)
	}
	s.logOp(ctx, kindergartenID, adminUserID, "teachers", "update", "teacher", teacherUserID,
		fmt.Sprintf(`{"nickname":%q}`, u.Nickname), ip, ua)
	return &updateTeacherResp{ID: teacherUserID, UpdatedAt: updated}, nil
}

func (s *TeacherService) DeleteTeacher(ctx context.Context, kindergartenID, adminUserID, teacherUserID uint64, ip, ua string) error {
	if kindergartenID == 0 {
		return fmt.Errorf("NO_KINDERGARTEN")
	}
	ur, err := s.roles.FindTeacherInKindergarten(ctx, teacherUserID, kindergartenID)
	if err != nil {
		return err
	}
	if ur == nil {
		return fmt.Errorf("NOT_FOUND")
	}
	if err := s.teachers.DeleteClassTeachersForUserInKindergarten(ctx, teacherUserID, kindergartenID); err != nil {
		return err
	}
	if err := s.roles.DeleteTeacherRole(ctx, teacherUserID, kindergartenID); err != nil {
		return err
	}
	if err := s.teachers.UnscopedDeleteProfile(ctx, kindergartenID, teacherUserID); err != nil {
		return err
	}
	s.logOp(ctx, kindergartenID, adminUserID, "teachers", "delete", "teacher", teacherUserID, "{}", ip, ua)
	return nil
}

type UpdateTeacherStatusReq struct {
	Status int8   `json:"status"`
	Reason string `json:"reason"`
}

type UpdateTeacherStatusResp struct {
	ID          uint64 `json:"id"`
	Status      int8   `json:"status"`
	StatusLabel string `json:"status_label"`
	UpdatedAt   string `json:"updated_at"`
}

func (s *TeacherService) UpdateTeacherStatus(
	ctx context.Context,
	kindergartenID, adminUserID, teacherUserID uint64,
	req UpdateTeacherStatusReq,
	ip, ua string,
) (*UpdateTeacherStatusResp, error) {
	if kindergartenID == 0 {
		return nil, fmt.Errorf("NO_KINDERGARTEN")
	}
	if req.Status != 1 && req.Status != 2 {
		return nil, fmt.Errorf("INVALID_STATUS")
	}
	ur, err := s.roles.FindTeacherInKindergarten(ctx, teacherUserID, kindergartenID)
	if err != nil {
		return nil, err
	}
	if ur == nil {
		return nil, fmt.Errorf("NOT_FOUND")
	}
	p, err := s.teachers.FindProfile(ctx, kindergartenID, teacherUserID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("NOT_FOUND")
	}
	p.Status = req.Status
	if err := s.teachers.SaveProfile(ctx, p); err != nil {
		return nil, err
	}
	detail := fmt.Sprintf(`{"status":%d,"reason":%q}`, req.Status, req.Reason)
	s.logOp(ctx, kindergartenID, adminUserID, "teachers", "update_status", "teacher", teacherUserID, detail, ip, ua)
	return &UpdateTeacherStatusResp{
		ID: teacherUserID, Status: p.Status, StatusLabel: teacherStatusLabel(p.Status),
		UpdatedAt: p.UpdatedAt.UTC().Format(time.RFC3339),
	}, nil
}

func IsTeacherClientErr(err error) bool {
	if err == nil {
		return false
	}
	switch err.Error() {
	case "NO_KINDERGARTEN", "INVALID_PHONE", "CLASS_ASSIGNMENTS_REQUIRED", "INVALID_JOIN_DATE",
		"KINDERGARTEN_NOT_FOUND", "DUPLICATE_TEACHER_PHONE", "DUPLICATE_CLASS_IN_ASSIGNMENTS",
		"CLASS_NOT_FOUND", "INVALID_STATUS", "INVALID_TEACHER_TITLE":
		return true
	default:
		return false
	}
}
