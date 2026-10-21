package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/pkg/pagination"
)

// 授予失败的语义化错误。handler 据此映射 400/403/409，不得吞成 500。
var (
	// ErrAchievementDisabled 徽章不存在或未启用
	ErrAchievementDisabled = errors.New("ACHIEVEMENT_DISABLED")
	// ErrGrantCrossKindergarten 徽章或学生不属于当前园所
	ErrGrantCrossKindergarten = errors.New("GRANT_CROSS_KINDERGARTEN")
	// ErrGrantWitnessInvalid 见证记录不存在、不属于本园，或该学生对它不可见
	ErrGrantWitnessInvalid = errors.New("GRANT_WITNESS_INVALID")
	// ErrGrantDuplicate 该学生已获得过这枚徽章
	ErrGrantDuplicate = errors.New("GRANT_DUPLICATE")
	// ErrGrantStudentNotFound 学生不存在
	ErrGrantStudentNotFound = errors.New("GRANT_STUDENT_NOT_FOUND")
)

type AchievementService struct {
	ach   *repository.AchievementRepo
	oplog *repository.OperationLogRepo
	// db 用于授予链路的跨实体校验（徽章+学生+见证记录+去重）。
	// AchievementRepo 只管 Achievement CRUD，不为一次授予去污染它的职责边界。
	db *gorm.DB
}

func NewAchievementService(ach *repository.AchievementRepo, oplog *repository.OperationLogRepo, db *gorm.DB) *AchievementService {
	return &AchievementService{ach: ach, oplog: oplog, db: db}
}

func IsAchievementNotFound(err error) bool {
	return errors.Is(err, repository.ErrAchievementNotFound)
}

func achievementCategoryLabel(cat string) string {
	switch strings.TrimSpace(strings.ToLower(cat)) {
	case "attendance":
		return "出勤"
	case "growth":
		return "成长"
	case "points", "point":
		return "积分"
	case "task", "tasks":
		return "任务"
	default:
		if strings.TrimSpace(cat) == "" {
			return "其他"
		}
		return cat
	}
}

func (s *AchievementService) ListAchievements(
	ctx context.Context,
	kindergartenID uint64,
	q pagination.Query,
) ([]map[string]interface{}, int64, int, int, error) {
	if kindergartenID == 0 {
		return nil, 0, 0, 0, errors.New("NO_KINDERGARTEN")
	}
	q.Normalize()
	rows, total, err := s.ach.List(ctx, kindergartenID, q.Offset(), q.PageSize)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	out := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		out = append(out, map[string]interface{}{
			"id":             r.ID,
			"name":           r.Name,
			"description":    r.Description,
			"icon_url":       r.Icon,
			"category":       r.Category,
			"category_label": achievementCategoryLabel(r.Category),
			"condition":      r.Condition,
			"earned_count":   r.EarnedCount,
			"status":         int(r.Status),
			"created_at":     r.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return out, total, q.Page, q.PageSize, nil
}

type CreateAchievementReq struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	IconURL     string `json:"icon_url"`
	Category    string `json:"category"`
	Condition   string `json:"condition"`
	TargetValue int    `json:"target_value"`
}

func (s *AchievementService) CreateAchievement(
	ctx context.Context,
	kindergartenID, adminUserID uint64,
	req CreateAchievementReq,
	ip, ua string,
) (map[string]interface{}, error) {
	if kindergartenID == 0 {
		return nil, errors.New("NO_KINDERGARTEN")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New("BAD_NAME")
	}
	row := &domain.Achievement{
		KindergartenID: kindergartenID,
		Name:           name,
		Description:    strings.TrimSpace(req.Description),
		Icon:           strings.TrimSpace(req.IconURL),
		Category:       strings.TrimSpace(req.Category),
		Condition:      strings.TrimSpace(req.Condition),
		TargetValue:    req.TargetValue,
		Status:         1,
	}
	if err := s.ach.Create(ctx, row); err != nil {
		return nil, err
	}
	_ = s.oplog.Create(ctx, &domain.OperationLog{
		KindergartenID: kindergartenID,
		UserID:         adminUserID,
		Module:         "achievements",
		Action:         "create",
		TargetType:     "achievement",
		TargetID:       row.ID,
		Detail:         fmt.Sprintf("name=%s", row.Name),
		IP:             ip,
		UserAgent:      truncateUA(ua),
	})
	return map[string]interface{}{
		"id":         row.ID,
		"created_at": row.CreatedAt.UTC().Format(time.RFC3339),
	}, nil
}

type UpdateAchievementReq struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	TargetValue *int    `json:"target_value"`
}

func (s *AchievementService) UpdateAchievement(
	ctx context.Context,
	kindergartenID, adminUserID, achievementID uint64,
	req UpdateAchievementReq,
	ip, ua string,
) (map[string]interface{}, error) {
	if kindergartenID == 0 {
		return nil, errors.New("NO_KINDERGARTEN")
	}
	patch := map[string]interface{}{}
	if req.Name != nil {
		n := strings.TrimSpace(*req.Name)
		if n == "" {
			return nil, errors.New("BAD_NAME")
		}
		patch["name"] = n
	}
	if req.Description != nil {
		patch["description"] = strings.TrimSpace(*req.Description)
	}
	if req.TargetValue != nil {
		if *req.TargetValue < 0 {
			return nil, errors.New("BAD_TARGET_VALUE")
		}
		patch["target_value"] = *req.TargetValue
	}
	if len(patch) == 0 {
		return nil, errors.New("EMPTY_UPDATE")
	}
	if err := s.ach.Update(ctx, kindergartenID, achievementID, patch); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	_ = s.oplog.Create(ctx, &domain.OperationLog{
		KindergartenID: kindergartenID,
		UserID:         adminUserID,
		Module:         "achievements",
		Action:         "update",
		TargetType:     "achievement",
		TargetID:       achievementID,
		Detail:         fmt.Sprintf("fields=%v", keysOfPatch(patch)),
		IP:             ip,
		UserAgent:      truncateUA(ua),
	})
	return map[string]interface{}{
		"id":         achievementID,
		"updated_at": now.Format(time.RFC3339),
	}, nil
}

func keysOfPatch(m map[string]interface{}) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}

func (s *AchievementService) DeleteAchievement(
	ctx context.Context,
	kindergartenID, adminUserID, achievementID uint64,
	ip, ua string,
) error {
	if kindergartenID == 0 {
		return errors.New("NO_KINDERGARTEN")
	}
	if err := s.ach.SoftDelete(ctx, kindergartenID, achievementID); err != nil {
		return err
	}
	_ = s.oplog.Create(ctx, &domain.OperationLog{
		KindergartenID: kindergartenID,
		UserID:         adminUserID,
		Module:         "achievements",
		Action:         "delete",
		TargetType:     "achievement",
		TargetID:       achievementID,
		Detail:         fmt.Sprintf("id=%d", achievementID),
		IP:             ip,
		UserAgent:      truncateUA(ua),
	})
	return nil
}

// GrantToStudent 把徽章授予学生，可选绑定一条成长记录作为获奖见证。
// 校验顺序（design D2）：徽章启用 → 跨园所（徽章/学生）→ 见证记录学生可见 → 重复授予。
// 见证记录必须对该学生可见（全班记录，或包含该学生的个体记录）——否则「见证」是假关联。
func (s *AchievementService) GrantToStudent(
	ctx context.Context,
	kindergartenID, classID, teacherUserID, achievementID uint64,
	studentIDs []uint64,
	logID *uint64,
) ([]uint64, error) {
	if len(studentIDs) == 0 {
		return nil, ErrGrantStudentNotFound
	}

	// 1) 徽章存在且启用
	var ach domain.Achievement
	if err := s.db.WithContext(ctx).
		Where("id = ? AND kindergarten_id = ?", achievementID, kindergartenID).
		First(&ach).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAchievementDisabled
		}
		return nil, err
	}
	if ach.Status != 1 {
		return nil, ErrAchievementDisabled
	}

	// 2) 见证记录：同园 + 该学生可见
	var log *domain.GrowthDailyLog
	if logID != nil && *logID != 0 {
		var row domain.GrowthDailyLog
		if err := s.db.WithContext(ctx).
			Where("id = ? AND kindergarten_id = ?", *logID, kindergartenID).
			First(&row).Error; err != nil {
			return nil, ErrGrantWitnessInvalid
		}
		if row.TargetType != 1 {
			// 个体记录：必须包含该学生；多个学生授予时要求对每个都可见
			for _, sid := range studentIDs {
				var n int64
				s.db.WithContext(ctx).Model(&domain.GrowthLogStudent{}).
					Where("log_id = ? AND student_id = ?", row.ID, sid).Count(&n)
				if n == 0 {
					return nil, ErrGrantWitnessInvalid
				}
			}
		}
		log = &row
	}

	granted := make([]uint64, 0, len(studentIDs))
	for _, sid := range studentIDs {
		// 3) 学生同园 + 与老师同班（spec 场景 TC-NEW-ACH-001：只能授予自己班级的学生）
		var stu domain.Student
		if err := s.db.WithContext(ctx).
			Where("id = ? AND kindergarten_id = ?", sid, kindergartenID).
			First(&stu).Error; err != nil {
			return nil, ErrGrantStudentNotFound
		}
		if classID != 0 && stu.ClassID != classID {
			return nil, ErrGrantCrossKindergarten
		}

		// 4) 重复授予：DB uniqueIndex 是最后防线，这里预检给语义化错误
		var n int64
		s.db.WithContext(ctx).Model(&domain.StudentAchievement{}).
			Where("student_id = ? AND achievement_id = ?", sid, achievementID).Count(&n)
		if n > 0 {
			return nil, ErrGrantDuplicate
		}

		row := domain.StudentAchievement{
			StudentID:     sid,
			AchievementID: achievementID,
			LogID:         logID,
			AchievedAt:    time.Now(),
		}
		if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
			// 并发下唯一索引可能先于预检命中——同样翻译成重复授予，而非 500
			if strings.Contains(err.Error(), "Duplicate") || strings.Contains(err.Error(), "duplicate") {
				return nil, ErrGrantDuplicate
			}
			return nil, err
		}
		granted = append(granted, sid)
	}

	_ = log // 见证记录已校验并写入 LogID；保留变量以便将来扩展（如通知）
	_ = teacherUserID
	return granted, nil
}

// ListEnabled 老师端授予入口用：只列本园**已启用**的徽章（停用的不该出现在选择器里）。
// 不含 Condition/TargetValue 等后台配置——那是 admin 与 P2 规则引擎的事。
func (s *AchievementService) ListEnabled(ctx context.Context, kindergartenID uint64) ([]map[string]any, error) {
	var rows []domain.Achievement
	if err := s.db.WithContext(ctx).
		Where("kindergarten_id = ? AND status = 1", kindergartenID).
		Order("sort_num ASC, id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, map[string]any{
			"id":       r.ID,
			"name":     r.Name,
			"icon":     r.Icon,
			"category": r.Category,
		})
	}
	return out, nil
}
