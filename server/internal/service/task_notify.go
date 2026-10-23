package service

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/pkg/log"
)

// 亲子任务的站内通知（B3 5.1-5.3）。
//
// 口径（design D9）：
//   - 任务发布：按「家庭」投递，班级任务覆盖该班全部家长，个体任务只覆盖目标孩子；
//   - 老师主回应：按回应覆盖的孩子 → 家长聚合，一次主回应里同一家长只收一条；
//   - 计数一律按家长用户去重，不是按提交条数；
//   - 补充回应不单独发通知（家长靠任务红点发现）；
//   - 通知失败只记日志与失败计数，绝不回滚任务/回应。

// taskNotifyTarget 一条通知的对象：家长 + 消息深链要落到哪个孩子。
type taskNotifyTarget struct {
	UserID    uint64
	StudentID uint64
}

// taskActionPath 任务详情的内部深链。家长端按它跳转，并据此切换到对应孩子。
func taskActionPath(taskID, studentID uint64) string {
	return fmt.Sprintf("/pages/tasks/detail?id=%d&student_id=%d", taskID, studentID)
}

// taskNotifyTargets 取任务发布要通知的家长（已按家长去重）。
//
// 同一家长在该范围内有多个孩子时固定取 student_id 最小的那个：
// 消息深链必须稳定，否则同一个家长每次点同一条消息可能落到不同孩子。
func taskNotifyTargets(ctx context.Context, db *gorm.DB, task *domain.ParentTask) ([]taskNotifyTarget, error) {
	type row struct {
		UserID    uint64 `gorm:"column:user_id"`
		StudentID uint64 `gorm:"column:student_id"`
	}
	// 手写 Table 不走 GORM 软删，两处 deleted_at 必须自己补。
	q := db.WithContext(ctx).Table("student_parents AS sp").
		Select("sp.user_id AS user_id, MIN(sp.student_id) AS student_id").
		Joins("INNER JOIN students AS st ON st.id = sp.student_id AND st.deleted_at IS NULL").
		Where("sp.deleted_at IS NULL AND st.kindergarten_id = ?", task.KindergartenID)
	if task.TargetStudentID > 0 {
		q = q.Where("st.id = ?", task.TargetStudentID)
	} else {
		q = q.Where("st.class_id = ?", task.ClassID)
	}
	var rows []row
	if err := q.Group("sp.user_id").Order("sp.user_id ASC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]taskNotifyTarget, 0, len(rows))
	for _, r := range rows {
		if r.UserID != 0 {
			out = append(out, taskNotifyTarget{UserID: r.UserID, StudentID: r.StudentID})
		}
	}
	return out, nil
}

// taskResponseNotifyTargets 取主回应要通知的家长：被回应覆盖的孩子 → 其全部家长。
// 一次回应覆盖同一家长多个孩子时，深链固定取**第一个被覆盖的提交**所属的孩子
// （按提交 id 升序）——design D9「回应消息用覆盖提交中的第一个孩子」。
func taskResponseNotifyTargets(
	ctx context.Context,
	db *gorm.DB,
	kindergartenID uint64,
	checkins []domain.TaskCheckIn,
) ([]taskNotifyTarget, error) {
	if len(checkins) == 0 {
		return nil, nil
	}
	// 每个孩子最早被覆盖的提交 id：用来决定同一位家长的消息落到哪个孩子。
	earliest := make(map[uint64]uint64, len(checkins))
	studentIDs := make([]uint64, 0, len(checkins))
	for _, c := range checkins {
		if prev, ok := earliest[c.StudentID]; !ok {
			earliest[c.StudentID] = c.ID
			studentIDs = append(studentIDs, c.StudentID)
		} else if c.ID < prev {
			earliest[c.StudentID] = c.ID
		}
	}

	type row struct {
		UserID    uint64 `gorm:"column:user_id"`
		StudentID uint64 `gorm:"column:student_id"`
	}
	var rows []row
	err := db.WithContext(ctx).Table("student_parents AS sp").
		Select("sp.user_id AS user_id, sp.student_id AS student_id").
		Joins("INNER JOIN students AS st ON st.id = sp.student_id AND st.deleted_at IS NULL").
		Where("sp.deleted_at IS NULL AND st.kindergarten_id = ? AND st.id IN ?", kindergartenID, studentIDs).
		Order("sp.user_id ASC, sp.student_id ASC").Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	// 同一家长（可能同一孩子的父母双方、或多个孩子）只保留一个深链目标：
	// 取「最早被覆盖的提交」最小的那个孩子。
	picked := make(map[uint64]uint64, len(rows))
	order := make([]uint64, 0, len(rows))
	for _, r := range rows {
		if r.UserID == 0 {
			continue
		}
		cur, ok := picked[r.UserID]
		if !ok {
			picked[r.UserID] = r.StudentID
			order = append(order, r.UserID)
			continue
		}
		if earliest[r.StudentID] < earliest[cur] {
			picked[r.UserID] = r.StudentID
		}
	}
	out := make([]taskNotifyTarget, 0, len(order))
	for _, uid := range order {
		out = append(out, taskNotifyTarget{UserID: uid, StudentID: picked[uid]})
	}
	return out, nil
}

// notifyTaskPublished 任务发布后给目标家庭写站内消息，返回按家长去重的投递计数。
// 失败只记日志并计数：通知是附带信息，不能把已创建的任务拖回事务。
func (s *TeacherPortalService) notifyTaskPublished(ctx context.Context, task *domain.ParentTask) (sent, failed int) {
	targets, err := taskNotifyTargets(ctx, s.db, task)
	if err != nil {
		log.L(ctx).Error("task publish notify: load targets failed", "task", task.ID, "err", err)
		return 0, 0
	}
	recipients := make([]messageRecipient, 0, len(targets))
	for _, t := range targets {
		recipients = append(recipients, messageRecipient{
			UserID:     t.UserID,
			ActionPath: taskActionPath(task.ID, t.StudentID),
		})
	}
	sent, failed = sendMessageToRecipients(ctx, s.msg, recipients, domain.MessageTypeTask,
		"新亲子任务："+task.Title,
		fmt.Sprintf("老师发布了亲子任务「%s」，点击查看并陪孩子一起完成。", task.Title),
		task.ID, true)
	if failed > 0 {
		log.L(ctx).Warn("task publish notify: partial failure", "task", task.ID, "sent", sent, "failed", failed)
	}
	return sent, failed
}

// notifyTaskResponse 老师主回应后按家长聚合投递一条消息。
// checkins 只能是**本次回应真正覆盖成功**的提交（批量回应里失败的提交不通知）。
// reference_id 用 response_id：同一回应重试不会重复打扰（dedupe）。
func (s *TeacherPortalService) notifyTaskResponse(
	ctx context.Context,
	task *domain.ParentTask,
	responseID uint64,
	checkins []domain.TaskCheckIn,
) (sent, failed int) {
	targets, err := taskResponseNotifyTargets(ctx, s.db, task.KindergartenID, checkins)
	if err != nil {
		log.L(ctx).Error("task response notify: load targets failed",
			"task", task.ID, "response", responseID, "err", err)
		return 0, 0
	}
	recipients := make([]messageRecipient, 0, len(targets))
	for _, t := range targets {
		recipients = append(recipients, messageRecipient{
			UserID:     t.UserID,
			ActionPath: taskActionPath(task.ID, t.StudentID),
		})
	}
	sent, failed = sendMessageToRecipients(ctx, s.msg, recipients, domain.MessageTypeTask,
		"老师回应了："+task.Title,
		fmt.Sprintf("老师在亲子任务「%s」中回应了孩子的打卡，点击查看。", task.Title),
		responseID, true)
	if failed > 0 {
		log.L(ctx).Warn("task response notify: partial failure",
			"task", task.ID, "response", responseID, "sent", sent, "failed", failed)
	}
	return sent, failed
}
