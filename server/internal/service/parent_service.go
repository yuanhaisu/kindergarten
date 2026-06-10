package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/pkg/pagination"
)

type ParentService struct {
	db       *gorm.DB
	parents  *repository.ParentRepo
	students *repository.StudentRepo
	users    *repository.UserRepo
	roles    *repository.UserRoleRepo
	kg       *repository.KindergartenRepo
	oplog    *repository.OperationLogRepo
}

func NewParentService(
	db *gorm.DB,
	parents *repository.ParentRepo,
	students *repository.StudentRepo,
	users *repository.UserRepo,
	roles *repository.UserRoleRepo,
	kg *repository.KindergartenRepo,
	oplog *repository.OperationLogRepo,
) *ParentService {
	return &ParentService{
		db: db, parents: parents, students: students, users: users, roles: roles, kg: kg, oplog: oplog,
	}
}

func (s *ParentService) logOp(ctx context.Context, kindergartenID, userID uint64, module, action, targetType string, targetID uint64, detail, ip, ua string) {
	_ = s.oplog.Create(ctx, &domain.OperationLog{
		KindergartenID: kindergartenID,
		UserID:         userID,
		Module:         module,
		Action:         action,
		TargetType:     targetType,
		TargetID:       targetID,
		Detail:         detail,
		IP:             ip,
		UserAgent:      truncateUAParent(ua),
	})
}

func truncateUAParent(ua string) string {
	if utf8.RuneCountInString(ua) <= 300 {
		return ua
	}
	return string([]rune(ua)[:300])
}

var phoneReParent = regexp.MustCompile(`^1[3-9]\d{9}$`)

func parentStatusLabel(st int8) string {
	if st == 2 {
		return "禁用"
	}
	return "正常"
}

func parseParentRelation(s string) (domain.ParentRelation, error) {
	pr := domain.ParentRelation(strings.TrimSpace(s))
	switch pr {
	case domain.RelationFather, domain.RelationMother, domain.RelationGrandfather, domain.RelationGrandmother, domain.RelationOther:
		return pr, nil
	default:
		return "", fmt.Errorf("INVALID_RELATION")
	}
}

type parentChildBrief struct {
	StudentID     uint64 `json:"student_id"`
	StudentName   string `json:"student_name"`
	ClassName     string `json:"class_name"`
	Relation      string `json:"relation"`
	RelationLabel string `json:"relation_label"`
}

type parentListItem struct {
	ID          uint64             `json:"id"`
	Nickname    string             `json:"nickname"`
	Avatar      string             `json:"avatar"`
	Phone       string             `json:"phone"`
	Gender      int                `json:"gender"`
	Children    []parentChildBrief `json:"children"`
	Status      int8               `json:"status"`
	StatusLabel string             `json:"status_label"`
	CreatedAt   string             `json:"created_at"`
}

func (s *ParentService) ListParents(
	ctx context.Context,
	kindergartenID uint64,
	q pagination.Query,
	classID *uint64,
) ([]parentListItem, int64, int, int, error) {
	if kindergartenID == 0 {
		return nil, 0, 0, 0, fmt.Errorf("NO_KINDERGARTEN")
	}
	q.Normalize()
	ids, total, err := s.parents.ListParentUserIDs(ctx, kindergartenID, q.Keyword, classID, q.Offset(), q.PageSize)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	if len(ids) == 0 {
		return []parentListItem{}, total, q.Page, q.PageSize, nil
	}
	profiles := make(map[uint64]*domain.ParentProfile, len(ids))
	for _, uid := range ids {
		p, err := s.parents.FindProfile(ctx, kindergartenID, uid)
		if err != nil {
			return nil, 0, 0, 0, err
		}
		if p != nil {
			profiles[uid] = p
		}
	}
	spRows, err := s.parents.ListStudentParentsForUsersInKindergarten(ctx, kindergartenID, ids)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	byUser := make(map[uint64][]domain.StudentParent)
	for _, sp := range spRows {
		byUser[sp.UserID] = append(byUser[sp.UserID], sp)
	}
	out := make([]parentListItem, 0, len(ids))
	for _, uid := range ids {
		u, err := s.users.FindByID(ctx, uid)
		if err != nil || u == nil {
			continue
		}
		prof := profiles[uid]
		st := int8(1)
		if prof != nil {
			st = prof.Status
		}
		children := make([]parentChildBrief, 0)
		for _, sp := range byUser[uid] {
			sn, cn := "", ""
			if sp.Student != nil {
				sn = sp.Student.Name
				if sp.Student.Class != nil {
					cn = sp.Student.Class.Name
				}
			}
			children = append(children, parentChildBrief{
				StudentID: sp.StudentID, StudentName: sn, ClassName: cn,
				Relation: string(sp.Relation), RelationLabel: relationLabel(sp.Relation),
			})
		}
		out = append(out, parentListItem{
			ID: uid, Nickname: u.Nickname, Avatar: u.Avatar, Phone: maskPhone(u.Phone),
			Gender: u.Gender, Children: children, Status: st, StatusLabel: parentStatusLabel(st),
			CreatedAt: u.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return out, total, q.Page, q.PageSize, nil
}

type parentDetailChild struct {
	StudentID     uint64 `json:"student_id"`
	StudentName   string `json:"student_name"`
	StudentAvatar string `json:"student_avatar"`
	ClassID       uint64 `json:"class_id"`
	ClassName     string `json:"class_name"`
	Relation      string `json:"relation"`
	RelationLabel string `json:"relation_label"`
	IsPrimary     bool   `json:"is_primary"`
}

type parentDetailResp struct {
	ID           uint64              `json:"id"`
	Nickname     string              `json:"nickname"`
	RealName     string              `json:"real_name"`
	Avatar       string              `json:"avatar"`
	Phone        string              `json:"phone"`
	Gender       int                 `json:"gender"`
	Children     []parentDetailChild `json:"children"`
	Status       int8                `json:"status"`
	StatusLabel  string              `json:"status_label"`
	LastLoginAt  string              `json:"last_login_at"`
	CreatedAt    string              `json:"created_at"`
}

func (s *ParentService) GetParent(ctx context.Context, kindergartenID, parentUserID uint64) (*parentDetailResp, error) {
	if kindergartenID == 0 {
		return nil, fmt.Errorf("NO_KINDERGARTEN")
	}
	ur, err := s.roles.FindParentInKindergarten(ctx, parentUserID, kindergartenID)
	if err != nil {
		return nil, err
	}
	if ur == nil {
		return nil, fmt.Errorf("NOT_FOUND")
	}
	prof, err := s.parents.FindProfile(ctx, kindergartenID, parentUserID)
	if err != nil {
		return nil, err
	}
	if prof == nil {
		return nil, fmt.Errorf("NOT_FOUND")
	}
	u, err := s.users.FindByID(ctx, parentUserID)
	if err != nil || u == nil {
		return nil, fmt.Errorf("NOT_FOUND")
	}
	spRows, err := s.parents.ListStudentParentsForUsersInKindergarten(ctx, kindergartenID, []uint64{parentUserID})
	if err != nil {
		return nil, err
	}
	children := make([]parentDetailChild, 0, len(spRows))
	for _, sp := range spRows {
		dc := parentDetailChild{
			StudentID: sp.StudentID, Relation: string(sp.Relation), RelationLabel: relationLabel(sp.Relation),
			IsPrimary: sp.IsPrimary,
		}
		if sp.Student != nil {
			dc.StudentName = sp.Student.Name
			dc.StudentAvatar = sp.Student.Avatar
			dc.ClassID = sp.Student.ClassID
			if sp.Student.Class != nil {
				dc.ClassName = sp.Student.Class.Name
			}
		}
		children = append(children, dc)
	}
	lastLogin := ""
	if prof.LastLoginAt != nil && !prof.LastLoginAt.IsZero() {
		lastLogin = prof.LastLoginAt.UTC().Format(time.RFC3339)
	}
	return &parentDetailResp{
		ID: parentUserID, Nickname: u.Nickname, RealName: prof.RealName, Avatar: u.Avatar,
		Phone: u.Phone, Gender: u.Gender, Children: children,
		Status: prof.Status, StatusLabel: parentStatusLabel(prof.Status),
		LastLoginAt: lastLogin, CreatedAt: u.CreatedAt.UTC().Format(time.RFC3339),
	}, nil
}

type studentBindingItem struct {
	StudentID uint64 `json:"student_id" binding:"required"`
	Relation  string `json:"relation" binding:"required"`
	IsPrimary bool   `json:"is_primary"`
}

type CreateParentReq struct {
	RealName         string               `json:"real_name" binding:"required"`
	Nickname         string               `json:"nickname" binding:"required"`
	Phone            string               `json:"phone" binding:"required"`
	Gender           int                  `json:"gender"`
	StudentBindings  []studentBindingItem `json:"student_bindings" binding:"required,min=1,dive"`
}

type createParentResp struct {
	ID        uint64 `json:"id"`
	CreatedAt string `json:"created_at"`
}

func (s *ParentService) buildStudentParentRows(
	ctx context.Context,
	kindergartenID, userID uint64,
	bindings []studentBindingItem,
) ([]domain.StudentParent, error) {
	if len(bindings) == 0 {
		return nil, fmt.Errorf("STUDENT_BINDINGS_REQUIRED")
	}
	out := make([]domain.StudentParent, 0, len(bindings))
	seen := make(map[uint64]struct{})
	for _, b := range bindings {
		if _, dup := seen[b.StudentID]; dup {
			return nil, fmt.Errorf("DUPLICATE_STUDENT_IN_BINDINGS")
		}
		seen[b.StudentID] = struct{}{}
		if _, err := parseParentRelation(b.Relation); err != nil {
			return nil, err
		}
		st, err := s.students.FindByID(ctx, kindergartenID, b.StudentID)
		if err != nil {
			return nil, err
		}
		if st == nil {
			return nil, fmt.Errorf("STUDENT_NOT_FOUND")
		}
		rel, _ := parseParentRelation(b.Relation)
		out = append(out, domain.StudentParent{
			StudentID: b.StudentID,
			UserID:    userID,
			Relation:  rel,
			IsPrimary: b.IsPrimary,
		})
	}
	return out, nil
}

func (s *ParentService) CreateParent(
	ctx context.Context,
	kindergartenID, adminUserID uint64,
	req CreateParentReq,
	ip, ua string,
) (*createParentResp, error) {
	if kindergartenID == 0 {
		return nil, fmt.Errorf("NO_KINDERGARTEN")
	}
	if !phoneReParent.MatchString(req.Phone) {
		return nil, fmt.Errorf("INVALID_PHONE")
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
		ur, err := s.roles.FindParentInKindergarten(ctx, existing.ID, kindergartenID)
		if err != nil {
			return nil, err
		}
		if ur != nil {
			return nil, fmt.Errorf("DUPLICATE_PARENT_PHONE")
		}
	}
	hash, err := HashPassword("KindyParent#1")
	if err != nil {
		return nil, err
	}
	var parentID uint64
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
		parentID = u.ID
		rows, err := s.buildStudentParentRows(ctx, kindergartenID, parentID, req.StudentBindings)
		if err != nil {
			return err
		}
		kid := kindergartenID
		nur := &domain.UserRole{
			UserID:           parentID,
			Role:             "parent",
			KindergartenID:   &kid,
			KindergartenName: kg.Name,
		}
		if err := tx.Create(nur).Error; err != nil {
			return err
		}
		prof := &domain.ParentProfile{
			UserID: parentID, KindergartenID: kindergartenID,
			RealName: strings.TrimSpace(req.RealName), Status: 1,
		}
		if err := tx.Create(prof).Error; err != nil {
			return err
		}
		for i := range rows {
			rows[i].ID = 0
			if err := tx.Create(&rows[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.logOp(ctx, kindergartenID, adminUserID, "parents", "create", "parent", parentID,
		fmt.Sprintf(`{"phone":%q,"nickname":%q}`, req.Phone, req.Nickname), ip, ua)
	u2, err := s.users.FindByID(ctx, parentID)
	if err != nil || u2 == nil {
		return &createParentResp{ID: parentID, CreatedAt: time.Now().UTC().Format(time.RFC3339)}, nil
	}
	return &createParentResp{ID: parentID, CreatedAt: u2.CreatedAt.UTC().Format(time.RFC3339)}, nil
}

type UpdateParentReq struct {
	Nickname        string               `json:"nickname" binding:"required"`
	StudentBindings []studentBindingItem `json:"student_bindings" binding:"required,min=1,dive"`
}

type updateParentResp struct {
	ID        uint64 `json:"id"`
	UpdatedAt string `json:"updated_at"`
}

func (s *ParentService) UpdateParent(
	ctx context.Context,
	kindergartenID, adminUserID, parentUserID uint64,
	req UpdateParentReq,
	ip, ua string,
) (*updateParentResp, error) {
	if kindergartenID == 0 {
		return nil, fmt.Errorf("NO_KINDERGARTEN")
	}
	ur, err := s.roles.FindParentInKindergarten(ctx, parentUserID, kindergartenID)
	if err != nil {
		return nil, err
	}
	if ur == nil {
		return nil, fmt.Errorf("NOT_FOUND")
	}
	prof, err := s.parents.FindProfile(ctx, kindergartenID, parentUserID)
	if err != nil {
		return nil, err
	}
	if prof == nil {
		return nil, fmt.Errorf("NOT_FOUND")
	}
	u, err := s.users.FindByID(ctx, parentUserID)
	if err != nil || u == nil {
		return nil, fmt.Errorf("NOT_FOUND")
	}
	rows, err := s.buildStudentParentRows(ctx, kindergartenID, parentUserID, req.StudentBindings)
	if err != nil {
		return nil, err
	}
	u.Nickname = strings.TrimSpace(req.Nickname)
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(u).Error; err != nil {
			return err
		}
		sub := tx.Model(&domain.Student{}).Select("id").Where("kindergarten_id = ?", kindergartenID)
		if err := tx.Unscoped().Where("user_id = ? AND student_id IN (?)", parentUserID, sub).
			Delete(&domain.StudentParent{}).Error; err != nil {
			return err
		}
		for i := range rows {
			rows[i].ID = 0
			if err := tx.Create(&rows[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	u3, _ := s.users.FindByID(ctx, parentUserID)
	updated := time.Now().UTC().Format(time.RFC3339)
	if u3 != nil {
		updated = u3.UpdatedAt.UTC().Format(time.RFC3339)
	}
	s.logOp(ctx, kindergartenID, adminUserID, "parents", "update", "parent", parentUserID,
		fmt.Sprintf(`{"nickname":%q}`, u.Nickname), ip, ua)
	return &updateParentResp{ID: parentUserID, UpdatedAt: updated}, nil
}

type UpdateParentStatusReq struct {
	Status int8   `json:"status" binding:"required"`
	Reason string `json:"reason"`
}

type UpdateParentStatusResp struct {
	ID          uint64 `json:"id"`
	Status      int8   `json:"status"`
	StatusLabel string `json:"status_label"`
	UpdatedAt   string `json:"updated_at"`
}

func (s *ParentService) UpdateParentStatus(
	ctx context.Context,
	kindergartenID, adminUserID, parentUserID uint64,
	req UpdateParentStatusReq,
	ip, ua string,
) (*UpdateParentStatusResp, error) {
	if kindergartenID == 0 {
		return nil, fmt.Errorf("NO_KINDERGARTEN")
	}
	if req.Status != 1 && req.Status != 2 {
		return nil, fmt.Errorf("INVALID_STATUS")
	}
	ur, err := s.roles.FindParentInKindergarten(ctx, parentUserID, kindergartenID)
	if err != nil {
		return nil, err
	}
	if ur == nil {
		return nil, fmt.Errorf("NOT_FOUND")
	}
	prof, err := s.parents.FindProfile(ctx, kindergartenID, parentUserID)
	if err != nil {
		return nil, err
	}
	if prof == nil {
		return nil, fmt.Errorf("NOT_FOUND")
	}
	prof.Status = req.Status
	if err := s.parents.SaveProfile(ctx, prof); err != nil {
		return nil, err
	}
	detail := fmt.Sprintf(`{"status":%d,"reason":%q}`, req.Status, req.Reason)
	s.logOp(ctx, kindergartenID, adminUserID, "parents", "update_status", "parent", parentUserID, detail, ip, ua)
	return &UpdateParentStatusResp{
		ID: parentUserID, Status: prof.Status, StatusLabel: parentStatusLabel(prof.Status),
		UpdatedAt: prof.UpdatedAt.UTC().Format(time.RFC3339),
	}, nil
}

func IsParentClientErr(err error) bool {
	if err == nil {
		return false
	}
	switch err.Error() {
	case "NO_KINDERGARTEN", "INVALID_PHONE", "KINDERGARTEN_NOT_FOUND", "DUPLICATE_PARENT_PHONE",
		"STUDENT_BINDINGS_REQUIRED", "DUPLICATE_STUDENT_IN_BINDINGS", "INVALID_RELATION", "STUDENT_NOT_FOUND",
		"INVALID_STATUS":
		return true
	default:
		return false
	}
}
