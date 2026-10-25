package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"kindergarten-server/internal/domain"
)

// 管理端多班活动（design D12 / tasks 7.1）。
//
// 一个活动 = N 个班级各一条 ParentTask，共享同一个服务端生成的 activity_id。
// 本文件只做编排（校验顺序、事务、锁序、通知）；结构规则一律走 task_rules.go，
// 不在活动入口里复制一份「个体/班级罐/仪式」判定（D12）。

const (
	// taskActivityMaxClasses 活动最多覆盖的班级数（design D12：≤50）。
	taskActivityMaxClasses = 50
	// taskActivityIDPrefix 活动 id 前缀：让「独立任务（空串）」与「活动任务」一眼可分。
	taskActivityIDPrefix = "act_"
)

var (
	// ErrTaskActivityNotFound 该园所下没有任何该 activity_id 的任务。
	ErrTaskActivityNotFound = errors.New("TASK_ACTIVITY_NOT_FOUND")
	// ErrTaskActivityClassesInvalid class_ids 为空、含 0，或含非本园所班级。
	ErrTaskActivityClassesInvalid = errors.New("TASK_ACTIVITY_CLASSES_INVALID")
	// ErrTaskActivityDuplicateClass class_ids 里出现重复班级。
	// 单独一个哨兵：归属校验的计数差也能挡住重复，但那条文案会让调用方
	// 去查「哪个班不属于本园所」，而真正的问题是选重了。
	ErrTaskActivityDuplicateClass = errors.New("TASK_ACTIVITY_DUPLICATE_CLASS")
	// ErrTaskActivityTooManyClasses 超过 50 个班级。
	ErrTaskActivityTooManyClasses = errors.New("TASK_ACTIVITY_TOO_MANY_CLASSES")
	// ErrTaskActivityFieldForbidden 批量编辑出现结构字段（频率/日期/班级/目标/仪式/星星/状态）。
	ErrTaskActivityFieldForbidden = errors.New("TASK_ACTIVITY_FIELD_FORBIDDEN")
	// ErrTaskActivityDeleteForbidden 活动任务禁止单条删除：只能整组删。
	ErrTaskActivityDeleteForbidden = errors.New("TASK_ACTIVITY_DELETE_FORBIDDEN")
	// ErrTaskActivityEditForbidden 活动任务禁止单条改结构字段：只能走活动端点整组改。
	ErrTaskActivityEditForbidden = errors.New("TASK_ACTIVITY_EDIT_FORBIDDEN")
)

// newTaskActivityID 生成不可猜、不连续的活动 id：act_<unixnano>_<24 位 hex>。
// 用 crypto/rand 而不是时间+自增：activity_id 会出现在 URL 里，可猜会让
// 别的园所猜到连续的取值（虽然仍有园所隔离，但没有必要暴露枚举面）。
func newTaskActivityID(now time.Time) (string, error) {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%d_%s", taskActivityIDPrefix, now.UnixNano(), hex.EncodeToString(b[:])), nil
}

// lockOwnedClasses 在事务内锁定并校验班级归属。
//
// 不能用普通 SELECT：InnoDB 的一致性读看不到并发软删，校验通过后仍可能为一个
// 正在被删除的班级建出任务——那类任务在列表/统计的 INNER JOIN classes 里永远不出现，
// 单条编辑时又会 404。FOR SHARE 让并发的软删等待本事务提交。
func lockOwnedClasses(ctx context.Context, tx *gorm.DB, kindergartenID uint64, classIDs []uint64) error {
	var ids []uint64
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "SHARE"}).
		Model(&domain.Class{}).
		Where("id IN ? AND kindergarten_id = ? AND deleted_at IS NULL", classIDs, kindergartenID).
		Pluck("id", &ids).Error; err != nil {
		return err
	}
	if len(ids) != len(classIDs) {
		return ErrTaskActivityClassesInvalid
	}
	return nil
}

// normalizeActivityClassIDs 纯输入校验：非空、无 0、不重复、≤50，并按 id 升序。
//
// 重复班级按校验失败处理而不是静默去重（design D12）：静默去重会让调用方
// 以为「选了 N 个班」，而实际只建了 N-1 条任务。园所归属不在这里查（要进事务）。
func normalizeActivityClassIDs(classIDs []uint64) ([]uint64, error) {
	if len(classIDs) == 0 {
		return nil, ErrTaskActivityClassesInvalid
	}
	seen := make(map[uint64]struct{}, len(classIDs))
	out := make([]uint64, 0, len(classIDs))
	for _, id := range classIDs {
		if id == 0 {
			return nil, ErrTaskActivityClassesInvalid
		}
		if _, ok := seen[id]; ok {
			return nil, ErrTaskActivityDuplicateClass
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) > taskActivityMaxClasses {
		return nil, ErrTaskActivityTooManyClasses
	}
	// 固定按 id 升序：建任务、批量编辑、整组删除的加锁顺序一致，
	// 两个并发操作同一组任务时不会形成 A→B / B→A 死锁。
	slices.Sort(out)
	return out, nil
}

// CreateActivityReq POST /admin/task-activities
//
// 活动没有个体目标（请求形状见 design D12 第 171 行），因此没有 target_student_id。
type CreateActivityReq struct {
	Title             string   `json:"title" binding:"required,max=200"`
	Description       string   `json:"description"`
	LearningLink      string   `json:"learning_link"`
	Frequency         string   `json:"frequency" binding:"required"`
	StartDate         string   `json:"start_date" binding:"required"`
	EndDate           string   `json:"end_date" binding:"required"`
	AllowPublic       bool     `json:"allow_public"`
	RecognitionRitual string   `json:"recognition_ritual"`
	StarScope         string   `json:"star_scope"`
	StarGoal          int      `json:"star_goal"`
	ClassIDs          []uint64 `json:"class_ids"`
	// ActivityID 只为「显式传入就拒绝」而存在：分组由服务端生成。
	ActivityID string `json:"activity_id"`
}

// CreateActivity 事务内逐班建任务，任一班级失败整组回滚。
func (s *TaskService) CreateActivity(
	ctx context.Context,
	kindergartenID, adminUserID uint64,
	req CreateActivityReq,
	ip, ua string,
) (map[string]any, error) {
	if kindergartenID == 0 {
		return nil, errors.New("NO_KINDERGARTEN")
	}
	if strings.TrimSpace(req.ActivityID) != "" {
		// 与 POST /admin/tasks 同一条规则：activity_id 只能由服务端生成。
		return nil, ErrTaskActivityIDForbidden
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, errors.New("BAD_TITLE")
	}
	if err := validateTaskTitleLength(title); err != nil {
		return nil, err
	}
	learningLink := strings.TrimSpace(req.LearningLink)
	if learningLink == "" {
		return nil, errors.New("BAD_LEARNING_LINK")
	}
	freq, err := parseTaskFrequency(req.Frequency)
	if err != nil {
		return nil, err
	}
	sd, ed, err := parseTaskDateRange(req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}
	structure := taskStructure{
		Ritual:    resolveTaskRitual(req.RecognitionRitual, "", domain.TaskRitualSticker),
		StarScope: resolveTaskStarScope(req.StarScope, ""),
		StarGoal:  req.StarGoal,
	}
	classIDs, err := normalizeActivityClassIDs(req.ClassIDs)
	if err != nil {
		return nil, err
	}
	activityID, err := newTaskActivityID(time.Now())
	if err != nil {
		return nil, err
	}

	tasks := make([]*domain.ParentTask, 0, len(classIDs))
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 归属校验与写入必须同事务（并且锁住班级行）：否则并发软删的班级
		// 仍会被建出任务，而它永远不会出现在列表/统计里。
		if err := lockOwnedClasses(ctx, tx, kindergartenID, classIDs); err != nil {
			return err
		}
		for _, classID := range classIDs {
			// 校验与写入交错：任何一班失败（含班级罐未配置）都会把前面已建的
			// 任务一起回滚——这正是「整组」的语义，不能靠「先全验后全建」绕过事务。
			if err := validateTaskStructureForCreate(ctx, tx, kindergartenID, classID, 0, structure); err != nil {
				return err
			}
			t := &domain.ParentTask{
				KindergartenID:    kindergartenID,
				ClassID:           classID,
				ActivityID:        activityID,
				CreatorID:         adminUserID,
				Title:             title,
				Description:       strings.TrimSpace(req.Description),
				LearningLink:      learningLink,
				Frequency:         freq,
				StartDate:         sd,
				EndDate:           ed,
				Status:            1,
				AllowPublic:       req.AllowPublic,
				RecognitionRitual: structure.Ritual,
				StarScope:         structure.StarScope,
				StarGoal:          structure.StarGoal,
			}
			if err := tx.Create(t).Error; err != nil {
				return err
			}
			tasks = append(tasks, t)
		}
		return nil
	})
	if err != nil {
		if isTaskLockConflict(err) {
			return nil, ErrTaskBusy
		}
		return nil, err
	}

	_ = s.oplog.Create(ctx, &domain.OperationLog{
		KindergartenID: kindergartenID,
		UserID:         adminUserID,
		Module:         "tasks",
		Action:         "create_activity",
		TargetType:     "task_activity",
		Detail:         fmt.Sprintf("activity_id=%s class_ids=%v", activityID, classIDs),
		IP:             ip,
		UserAgent:      truncateUA(ua),
	})

	// 通知在落库之后逐任务发（D9）：失败只计数，不回滚活动。
	notified, notifyFailed := 0, 0
	taskIDs := make([]uint64, 0, len(tasks))
	for _, t := range tasks {
		taskIDs = append(taskIDs, t.ID)
		n, f := notifyTaskPublishedFor(ctx, s.db, s.msg, t)
		notified += n
		notifyFailed += f
	}
	return map[string]any{
		"activity_id":         activityID,
		"task_ids":            taskIDs,
		"notified_count":      notified,
		"notify_failed_count": notifyFailed,
	}, nil
}

// UpdateActivityReq PUT /admin/task-activities/:activity_id
//
// 结构字段只为「显式传入即拒绝」而存在：静默忽略会让调用方以为批量改了频率/日期，
// 而这正是 D12 要消灭的失败模式。
type UpdateActivityReq struct {
	Title        *string `json:"title"`
	Description  *string `json:"description"`
	LearningLink *string `json:"learning_link"`
	AllowPublic  *bool   `json:"allow_public"`

	Frequency         string   `json:"frequency"`
	StartDate         string   `json:"start_date"`
	EndDate           string   `json:"end_date"`
	ClassIDs          []uint64 `json:"class_ids"`
	TargetStudentID   *uint64  `json:"target_student_id"`
	RecognitionRitual string   `json:"recognition_ritual"`
	StarScope         string   `json:"star_scope"`
	StarGoal          *int     `json:"star_goal"`
	// Status 是唯一允许批量改的「非白名单」字段：活动没有别的结束路径（0=整组结束）。
	Status *int8 `json:"status"`

	ActivityID string `json:"activity_id"`
}

// hasStructuralField 判断请求里是否出现了活动批量编辑不允许的字段。
func (r UpdateActivityReq) hasStructuralField() bool {
	return strings.TrimSpace(r.Frequency) != "" ||
		strings.TrimSpace(r.StartDate) != "" ||
		strings.TrimSpace(r.EndDate) != "" ||
		len(r.ClassIDs) > 0 ||
		r.TargetStudentID != nil ||
		strings.TrimSpace(r.RecognitionRitual) != "" ||
		strings.TrimSpace(r.StarScope) != "" ||
		r.StarGoal != nil ||
		strings.TrimSpace(r.ActivityID) != ""
}

// UpdateActivity 批量修改活动任务的安全字段；关闭公开时整组撤销。
func (s *TaskService) UpdateActivity(
	ctx context.Context,
	kindergartenID, adminUserID uint64,
	activityID string,
	req UpdateActivityReq,
	ip, ua string,
) (map[string]any, error) {
	if kindergartenID == 0 {
		return nil, errors.New("NO_KINDERGARTEN")
	}
	activityID = strings.TrimSpace(activityID)
	if activityID == "" {
		return nil, ErrTaskActivityNotFound
	}
	if req.hasStructuralField() {
		return nil, ErrTaskActivityFieldForbidden
	}

	updates := map[string]any{}
	if req.Title != nil {
		t := strings.TrimSpace(*req.Title)
		if t == "" {
			return nil, errors.New("BAD_TITLE")
		}
		if err := validateTaskTitleLength(t); err != nil {
			return nil, err
		}
		updates["title"] = t
	}
	if req.Description != nil {
		updates["description"] = strings.TrimSpace(*req.Description)
	}
	if req.LearningLink != nil {
		// 空串按「未提供」处理，与单任务编辑同口径：学习关联必填，不能被清空。
		if v := strings.TrimSpace(*req.LearningLink); v != "" {
			updates["learning_link"] = v
		}
	}
	if req.AllowPublic != nil {
		updates["allow_public"] = *req.AllowPublic
	}
	if req.Status != nil {
		// status 不是结构字段（C-1 裁定），而且它是活动**唯一**的结束路径：
		// 过期活动若不能置 0，管理端统计会永远把它算成 active。
		if *req.Status != 0 && *req.Status != 1 {
			return nil, errors.New("BAD_STATUS")
		}
		updates["status"] = *req.Status
	}

	updated := 0
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var tasks []domain.ParentTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("kindergarten_id = ? AND activity_id = ? AND deleted_at IS NULL", kindergartenID, activityID).
			Order("id ASC").Find(&tasks).Error; err != nil {
			return err
		}
		if len(tasks) == 0 {
			return ErrTaskActivityNotFound
		}
		for _, t := range tasks {
			if len(updates) > 0 {
				// assertTaskEditable 在这里是一条不变量声明：活动批量编辑只带安全字段，
				// 将来若有人往 updates 里加结构字段，有提交的任务会先被挡下。
				// status 单独摘出：批量结束不能被「有提交后的结构锁」挡住（与管理端单任务同口径）。
				structural := make(map[string]any, len(updates))
				for k, v := range updates {
					if k != "status" {
						structural[k] = v
					}
				}
				if err := assertTaskEditable(ctx, tx, t.ID, structural); err != nil {
					return err
				}
				if err := tx.Model(&domain.ParentTask{}).Where("id = ?", t.ID).
					Updates(updates).Error; err != nil {
					return err
				}
			}
			if req.AllowPublic != nil && !*req.AllowPublic && t.AllowPublic {
				if err := revokePublicForTask(ctx, tx, kindergartenID, adminUserID, t.ID, "activity_allow_public_off"); err != nil {
					return err
				}
			}
			updated++
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrTaskActivityNotFound) {
			return nil, err
		}
		if isTaskLockConflict(err) {
			return nil, ErrTaskBusy
		}
		return nil, err
	}

	_ = s.oplog.Create(ctx, &domain.OperationLog{
		KindergartenID: kindergartenID,
		UserID:         adminUserID,
		Module:         "tasks",
		Action:         "update_activity",
		TargetType:     "task_activity",
		Detail:         fmt.Sprintf("activity_id=%s fields=%d", activityID, len(updates)),
		IP:             ip,
		UserAgent:      truncateUA(ua),
	})
	return map[string]any{"activity_id": activityID, "updated_tasks": updated}, nil
}

// DeleteActivity 整组删除：每个任务走与单条删除相同的级联，全部在一个事务内。
func (s *TaskService) DeleteActivity(
	ctx context.Context,
	kindergartenID, adminUserID uint64,
	activityID, ip, ua string,
) (map[string]any, error) {
	if kindergartenID == 0 {
		return nil, errors.New("NO_KINDERGARTEN")
	}
	activityID = strings.TrimSpace(activityID)
	if activityID == "" {
		return nil, ErrTaskActivityNotFound
	}
	deleted := 0
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var tasks []domain.ParentTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("kindergarten_id = ? AND activity_id = ? AND deleted_at IS NULL", kindergartenID, activityID).
			Order("id ASC").Find(&tasks).Error; err != nil {
			return err
		}
		if len(tasks) == 0 {
			return ErrTaskActivityNotFound
		}
		for _, t := range tasks {
			if err := deleteTaskCascade(ctx, tx, t.ID); err != nil {
				return err
			}
			deleted++
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrTaskActivityNotFound) || errors.Is(err, errTaskNotFound) {
			return nil, ErrTaskActivityNotFound
		}
		if isTaskLockConflict(err) {
			return nil, ErrTaskBusy
		}
		return nil, err
	}

	_ = s.oplog.Create(ctx, &domain.OperationLog{
		KindergartenID: kindergartenID,
		UserID:         adminUserID,
		Module:         "tasks",
		Action:         "delete_activity",
		TargetType:     "task_activity",
		Detail:         fmt.Sprintf("activity_id=%s deleted=%d", activityID, deleted),
		IP:             ip,
		UserAgent:      truncateUA(ua),
	})
	return map[string]any{"activity_id": activityID, "deleted_tasks": deleted}, nil
}
