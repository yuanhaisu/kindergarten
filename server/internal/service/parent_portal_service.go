package service

import (
	"context"
	"errors"
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
	db      *gorm.DB
	parents *repository.ParentRepo
	roles   *repository.UserRoleRepo
	students *repository.StudentRepo
	classes *repository.ClassRepo
	pickup  *repository.PickupRepo
}

func NewParentPortalService(
	db *gorm.DB,
	parents *repository.ParentRepo,
	roles *repository.UserRoleRepo,
	students *repository.StudentRepo,
	classes *repository.ClassRepo,
	pickup *repository.PickupRepo,
) *ParentPortalService {
	return &ParentPortalService{db: db, parents: parents, roles: roles, students: students, classes: classes, pickup: pickup}
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
