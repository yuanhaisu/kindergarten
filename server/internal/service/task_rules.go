package service

import (
	"context"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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
	// 补充回应只能追加到已有主回应的提交上；待回应提交必须先走主回应。
	ErrTaskSupplementForbidden = errors.New("TASK_SUPPLEMENT_FORBIDDEN")
	// 补充回应只支持贴纸/语音，班级星星不适用：一条提交只有一个主回应能计入星星罐。
	ErrTaskSupplementRitual = errors.New("TASK_SUPPLEMENT_RITUAL")
	// 仪式缺少载体：贴纸没选具体贴纸，或语音没有媒体。
	ErrTaskRitualPayload = errors.New("TASK_RITUAL_PAYLOAD")
	// 任务不允许公开：未开启 allow_public、是个体任务，或任务已不在有效期内。
	ErrTaskPublicNotAllowed = errors.New("TASK_PUBLIC_NOT_ALLOWED")
	// 提交当前不在待审核/已公开状态：并发取消、或从未申请就批准。
	ErrTaskPublicStateConflict = errors.New("TASK_PUBLIC_STATE_CONFLICT")
	// 老师批准前没有勾选「内容中只有提交者孩子」。
	ErrTaskPublicScopeUnconfirmed = errors.New("TASK_PUBLIC_SCOPE_UNCONFIRMED")
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

// supplementRitual 归一化补充回应的仪式。
//
// 空值默认贴纸，**不继承**任务默认仪式：任务默认仪式表达的是「主回应的口径」，
// 若继承下来，一个默认 class_star 的任务会让所有补充回应都变成非法请求。
// class_star 与未知取值都拒绝——前者由规格显式要求，后者不能悄悄落到贴纸。
func supplementRitual(ritual string) (string, error) {
	switch ritual {
	case "":
		return domain.TaskRitualSticker, nil
	case domain.TaskRitualSticker, domain.TaskRitualVoice:
		return ritual, nil
	case domain.TaskRitualClassStar:
		return "", ErrTaskSupplementRitual
	default:
		return "", ErrTaskSupplementRitual
	}
}

// validateRitualPayload 校验仪式所需的载体：贴纸必须带 sticker_key（否则家长端渲染不出来），
// 语音必须带媒体（否则家长端播放不了）。class_star 不需要载体。
// 规格原文：贴纸 MUST 能被家长端渲染；语音 MUST 能被家长端播放。
func validateRitualPayload(ritual, stickerKey string, mediaFileID uint64) error {
	switch ritual {
	case domain.TaskRitualSticker:
		if stickerKey == "" {
			return ErrTaskRitualPayload
		}
	case domain.TaskRitualVoice:
		if mediaFileID == 0 {
			return ErrTaskRitualPayload
		}
	}
	return nil
}

// validTaskReaction 家长反馈只接受三种取值。
func validTaskReaction(reaction string) bool {
	switch reaction {
	case domain.TaskReactionChildHappy, domain.TaskReactionReceived, domain.TaskReactionThanks:
		return true
	default:
		return false
	}
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
//
// 回应 id 必须用**当前读**取（FOR UPDATE）。用普通 SELECT 拿到的是事务快照，
// 与并发的 SupplementResponse 交叉时会漏掉它刚提交的补充回应，结果只软删了回应、
// 没删补充关系，留下一条 deleted_at IS NULL 的孤儿关系行。
func deleteTaskCascade(ctx context.Context, tx *gorm.DB, taskID uint64) error {
	var responseIDs []uint64
	if err := tx.WithContext(ctx).Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).
		Model(&domain.TaskResponse{}).Where("task_id = ?", taskID).
		Pluck("id", &responseIDs).Error; err != nil {
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

// ==================== 公开墙状态机（design D7 / tasks 4.1-4.4） ====================

// 公开状态取值，与契约里的 public_status 一致。
const (
	TaskPublicNone     = "none"     // 从未申请
	TaskPublicPending  = "pending"  // 已申请，待老师审核
	TaskPublicApproved = "approved" // 已批准且未撤回（老师确认过内容边界）
	TaskPublicRevoked  = "revoked"  // 已取消或已撤回
)

// taskDayOf 把时间归一到上海墙钟的当天零点。
// 与 taskIsActive 同口径：直接用值自身的年月日，不做时区换算——
// time.Local 已被初始化成 Asia/Shanghai，DSN loc=Local，读回来的 DATE 就是上海零点。
func taskDayOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, shanghaiLoc())
}

// publicCheckinStatus 由字段推导状态，避免客户端各自拼条件拼错。
//
// 「已批准」要求四个条件同时成立：申请中、确认过、有批准时间、未撤回。
// 任何一项缺失都只能算 pending——宁可让家长重走一次审核，也不能把
// 未确认内容边界的记录当已公开放上墙。
func publicCheckinStatus(c *domain.TaskCheckIn) string {
	if !c.IsPublicRequested {
		if c.PublicRevokedAt != nil {
			return TaskPublicRevoked
		}
		return TaskPublicNone
	}
	if c.PublicApprovedAt != nil && c.PublicScopeConfirmed && c.PublicRevokedAt == nil {
		return TaskPublicApproved
	}
	return TaskPublicPending
}

// publicCheckinVisible 公开墙可见条件，逐字对应 design D7 与 tasks 4.3。
//
// classID 传 0 表示「不看读者班级」（老师/管理端回显 public_visible 时用：
// 他们的问题是「这条现在是否生效」，而不是「某个家长能不能看到」）。
func publicCheckinVisible(t *domain.ParentTask, c *domain.TaskCheckIn, classID uint64, today time.Time) bool {
	if t == nil || c == nil {
		return false
	}
	if t.DeletedAt.Valid || c.DeletedAt.Valid {
		return false
	}
	if !t.AllowPublic || t.Status != 1 || t.TargetStudentID > 0 {
		return false
	}
	if classID > 0 && t.ClassID != classID {
		return false
	}
	if taskDayOf(t.EndDate).Before(taskDayOf(today)) {
		return false
	}
	return c.IsPublicRequested && c.PublicScopeConfirmed &&
		c.PublicApprovedAt != nil && c.PublicRevokedAt == nil
}

// publicRequestUpdates 申请公开：清空旧批准、旧确认与旧的撤回时间。
// 重新申请必须重新审核，不得沿用旧批准（design D7 状态转换）。
func publicRequestUpdates() map[string]any {
	return map[string]any{
		"is_public_requested":    true,
		"public_scope_confirmed": false,
		"public_approved_at":     nil,
		"public_approved_by":     0,
		"public_revoked_at":      nil,
	}
}

// publicApproveUpdates 批准公开：老师已确认内容中只有提交者孩子。
func publicApproveUpdates(now time.Time, teacherID uint64) map[string]any {
	return map[string]any{
		"is_public_requested":    true,
		"public_scope_confirmed": true,
		"public_approved_at":     &now,
		"public_approved_by":     teacherID,
		"public_revoked_at":      nil,
	}
}

// publicRevokeUpdates 取消/撤回：清空确认与审核人，写撤回时间。
func publicRevokeUpdates(now time.Time) map[string]any {
	return map[string]any{
		"is_public_requested":    false,
		"public_scope_confirmed": false,
		"public_approved_at":     nil,
		"public_approved_by":     0,
		"public_revoked_at":      &now,
	}
}

// publicCheckinView 四个公开端点共用的响应形状。
// public_visible 由服务端算好：客户端不要自己拼条件，否则条件一改就会出现
// 「界面显示已公开、墙上却没有」的漂移。
func publicCheckinView(t *domain.ParentTask, c *domain.TaskCheckIn, classID uint64, now time.Time) map[string]any {
	var approvedAt, revokedAt any
	if c.PublicApprovedAt != nil {
		approvedAt = c.PublicApprovedAt.UTC().Format(time.RFC3339)
	}
	if c.PublicRevokedAt != nil {
		revokedAt = c.PublicRevokedAt.UTC().Format(time.RFC3339)
	}
	return map[string]any{
		"checkin_id":             c.ID,
		"task_id":                t.ID,
		"is_public_requested":    c.IsPublicRequested,
		"public_scope_confirmed": c.PublicScopeConfirmed,
		"public_approved_at":     approvedAt,
		"public_approved_by":     c.PublicApprovedBy,
		"public_revoked_at":      revokedAt,
		"public_status":          publicCheckinStatus(c),
		"public_visible":         publicCheckinVisible(t, c, classID, now),
	}
}

// checkinLocate 仅用于定位提交所属任务与孩子。
type checkinLocate struct {
	TaskID    uint64 `gorm:"column:task_id"`
	StudentID uint64 `gorm:"column:student_id"`
}

// locateCheckin 用**非锁定读**拿到提交的 task_id / student_id。
//
// 这里刻意不用 FOR UPDATE：公开相关的写路径必须以 parent_tasks 行作为第一把锁
// （见 handoff §12.2），而按 checkin_id 进来的请求事先不知道 task_id。
// InnoDB 的一致性非锁定读不取任何行锁，因此不参与死锁的加锁顺序；
// 真正的校验在拿到任务锁之后用 FOR UPDATE 重读一次完成。
func locateCheckin(ctx context.Context, tx *gorm.DB, checkinID uint64) (*checkinLocate, error) {
	var row checkinLocate
	if err := tx.WithContext(ctx).Table("task_check_ins").
		Select("task_id, student_id").
		Where("id = ? AND deleted_at IS NULL", checkinID).
		Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errTaskNotFound
		}
		return nil, err
	}
	return &row, nil
}
