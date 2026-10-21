package service

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

var (
	errTaskStructuralLocked = errors.New("TASK_STRUCTURAL_LOCKED")
	errTaskTargetInvalid    = errors.New("TASK_TARGET_INVALID")
	errTaskStarJarMissing   = errors.New("TASK_STAR_JAR_MISSING")
)

// validateTargetStudent 校验个体任务目标孩子属于当前班级与园所。
func validateTargetStudent(ctx context.Context, db *gorm.DB, kindergartenID, classID, studentID uint64) error {
	if studentID == 0 {
		return nil
	}
	var n int64
	if err := db.WithContext(ctx).Model(&domain.Student{}).
		Where("id = ? AND kindergarten_id = ? AND class_id = ? AND deleted_at IS NULL AND status = 1", studentID, kindergartenID, classID).
		Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return errTaskTargetInvalid
	}
	return nil
}

// validateClassStarJar 校验班级当前是否有可用的星星罐目标。
func validateClassStarJar(ctx context.Context, db *gorm.DB, kindergartenID, classID uint64) error {
	var n int64
	if err := db.WithContext(ctx).Model(&domain.ClassStarJar{}).
		Where("kindergarten_id = ? AND class_id = ? AND goal > 0", kindergartenID, classID).
		Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return errTaskStarJarMissing
	}
	return nil
}

func sameTaskDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}

func validTaskFrequency(f domain.TaskFrequency) bool {
	switch f {
	case domain.TaskFrequencyDaily, domain.TaskFrequencyWeekly, domain.TaskFrequencyOnce:
		return true
	default:
		return false
	}
}

// assertTaskEditable 有提交后只允许安全字段。
func assertTaskEditable(ctx context.Context, db *gorm.DB, taskID uint64, updates map[string]any) error {
	var n int64
	if err := db.WithContext(ctx).Model(&domain.TaskCheckIn{}).
		Where("task_id = ?", taskID).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return nil
	}
	allowed := map[string]bool{"title": true, "description": true, "learning_link": true, "allow_public": true}
	for k := range updates {
		if !allowed[k] {
			return errTaskStructuralLocked
		}
	}
	return nil
}

// deleteTaskCascade 删除任务及其提交/回应/读标记/频率状态/补充/反馈。
func deleteTaskCascade(ctx context.Context, tx *gorm.DB, taskID uint64) error {
	var responseIDs []uint64
	if err := tx.WithContext(ctx).Unscoped().Model(&domain.TaskResponse{}).Where("task_id = ?", taskID).Pluck("id", &responseIDs).Error; err != nil {
		return err
	}
	if len(responseIDs) > 0 {
		if err := tx.WithContext(ctx).Unscoped().Where("response_id IN ?", responseIDs).Delete(&domain.TaskResponseSupplement{}).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Unscoped().Where("response_id IN ?", responseIDs).Delete(&domain.TaskResponseReaction{}).Error; err != nil {
			return err
		}
	}
	if err := tx.WithContext(ctx).Where("task_id = ?", taskID).Delete(&domain.TaskCheckIn{}).Error; err != nil {
		return err
	}
	if err := tx.WithContext(ctx).Where("task_id = ?", taskID).Delete(&domain.TaskResponse{}).Error; err != nil {
		return err
	}
	if err := tx.WithContext(ctx).Unscoped().Where("task_id = ?", taskID).Delete(&domain.TaskCheckInState{}).Error; err != nil {
		return err
	}
	if err := tx.WithContext(ctx).Unscoped().Where("task_id = ?", taskID).Delete(&domain.TaskReadMark{}).Error; err != nil {
		return err
	}
	res := tx.WithContext(ctx).Where("id = ?", taskID).Delete(&domain.ParentTask{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errTaskNotFound
	}
	return nil
}

// revokePublicForTask 关闭任务公开：清空申请/批准/确认/审核人。
func revokePublicForTask(ctx context.Context, tx *gorm.DB, taskID uint64) error {
	now := time.Now()
	return tx.WithContext(ctx).Model(&domain.TaskCheckIn{}).
		Where("task_id = ?", taskID).
		Updates(map[string]any{
			"is_public_requested":    false,
			"public_scope_confirmed": false,
			"public_approved_at":     nil,
			"public_approved_by":     0,
			"public_revoked_at":      &now,
		}).Error
}
