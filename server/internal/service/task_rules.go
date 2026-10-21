package service

import (
	"context"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

var (
	errTaskStructuralLocked = errors.New("TASK_STRUCTURAL_LOCKED")
	errTaskTargetInvalid    = errors.New("TASK_TARGET_INVALID")
	errTaskStarJarMissing   = errors.New("TASK_STAR_JAR_MISSING")
	// 批量回应没有抢到任何待回应提交（并发/重复点击的落败方）：回滚，不留孤儿回应。
	errTaskBatchNothingPending = errors.New("TASK_BATCH_NOTHING_PENDING")
	// 规格要求个体任务与一次性活动逐条回应，不允许走汇总回应。
	ErrTaskBatchNotAllowed = errors.New("TASK_BATCH_NOT_ALLOWED")
	// 有候选提交但一条都没写成功（DB 写错误）：不能当成「没有待回应」。
	// 并发落败方在读阶段就拿不到行，走的是 errTaskBatchNothingPending，不会到这里。
	ErrTaskBatchAllFailed = errors.New("TASK_BATCH_ALL_FAILED")
	// 提交在写入瞬间被其他回应关闭：让客户端刷新，而不是报「参数无效」。
	ErrTaskRespondConflict = errors.New("TASK_RESPOND_CONFLICT")
	// 行锁等待超时 / 死锁：可重试，映射为 503。
	ErrTaskBusy = errors.New("TASK_BUSY")
)

// isTaskLockConflict 判定 MySQL 锁等待超时（1205）与死锁（1213）。
func isTaskLockConflict(err error) bool {
	var me *mysql.MySQLError
	if errors.As(err, &me) {
		return me.Number == 1205 || me.Number == 1213
	}
	return false
}

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

// batchRespondAllowed 汇总回应只适用于「每日 / 每周的班级任务」：
// 规格要求个体任务与一次性活动逐条回应；未知频率同样收敛为不允许，
// 避免新增频率时悄悄落进 default 分支按任务全程汇总。
func batchRespondAllowed(t *domain.ParentTask) bool {
	if t.TargetStudentID > 0 {
		return false
	}
	return t.Frequency == domain.TaskFrequencyDaily || t.Frequency == domain.TaskFrequencyWeekly
}

// taskPeriodRange 返回周期汇总回应覆盖的日期区间（含首尾）：
// 每日=锚点当天；每周=锚点往前滚动 7 天（不早于任务开始日）。
// 锚点取 min(今天, 任务结束日)：任务结束后仍能用「本周期」关闭最后一个周期，
// 否则已结束任务的待回应提交会永久无法批量关闭。
//
// default 分支只作防御：生产入口由 batchRespondAllowed 拦截，
// 未知频率不会走到这里。
func taskPeriodRange(t *domain.ParentTask, now time.Time) (time.Time, time.Time) {
	loc := shanghaiLoc()
	day := func(v time.Time) time.Time {
		v = v.In(loc)
		return time.Date(v.Year(), v.Month(), v.Day(), 0, 0, 0, 0, loc)
	}
	today := day(now)
	start, end := day(t.StartDate), day(t.EndDate)
	if end.Before(start) {
		end = start
	}
	anchor := today
	if anchor.After(end) {
		anchor = end
	}
	if anchor.Before(start) {
		anchor = start
	}
	switch t.Frequency {
	case domain.TaskFrequencyDaily:
		return anchor, anchor
	case domain.TaskFrequencyWeekly:
		from := anchor.AddDate(0, 0, -6)
		if from.Before(start) {
			from = start
		}
		return from, anchor
	default:
		return start, end
	}
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
