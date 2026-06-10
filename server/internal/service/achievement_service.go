package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/pkg/pagination"
)

type AchievementService struct {
	ach   *repository.AchievementRepo
	oplog *repository.OperationLogRepo
}

func NewAchievementService(ach *repository.AchievementRepo, oplog *repository.OperationLogRepo) *AchievementService {
	return &AchievementService{ach: ach, oplog: oplog}
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
