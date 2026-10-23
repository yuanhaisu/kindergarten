package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"kindergarten-server/internal/domain"
)

var (
	// ErrTaskStructuralLocked 有提交后修改结构字段。导出给 handler 映射成明确文案，
	// 不能落进笼统的「参数错误」——老师根本看不出是结构锁。
	ErrTaskStructuralLocked = errors.New("TASK_STRUCTURAL_LOCKED")
	errTaskTargetInvalid    = errors.New("TASK_TARGET_INVALID")
	errTaskStarJarMissing   = errors.New("TASK_STAR_JAR_MISSING")
	// ErrClassNotFound 班级不存在或不属于当前园所（星星罐端点按 404 返回）。
	ErrClassNotFound = errors.New("CLASS_NOT_FOUND")
	// ErrStarJarGoalInvalid 班级星星罐目标必须 > 0：0 表示「未配置」，
	// 允许写 0 会让配置看起来成功、但 validateClassStarJar 继续拒绝建任务。
	ErrStarJarGoalInvalid = errors.New("STAR_JAR_GOAL_INVALID")
	// ErrTaskRitualInvalid 仪式取值不在 sticker / voice / class_star 内。
	ErrTaskRitualInvalid = errors.New("TASK_RITUAL_INVALID")
	// ErrTaskStarScopeInvalid 星星范围不在 task / class 内。
	ErrTaskStarScopeInvalid = errors.New("TASK_STAR_SCOPE_INVALID")
	// ErrTaskStarGoalInvalid 默认仪式是班级星星且按任务统计时，目标必须 > 0，
	// 否则星星永远无处累加（design D6）。
	ErrTaskStarGoalInvalid = errors.New("TASK_STAR_GOAL_INVALID")
	// ErrTaskStarGoalNegative 星星目标不能为负：0 表示「不启用」，负数只会让
	// 进度条算出 `已集 N/-5` 这种荒唐值，且封顶逻辑全部失效。
	ErrTaskStarGoalNegative = errors.New("TASK_STAR_GOAL_NEGATIVE")
	// ErrTaskActivityIDForbidden 独立任务入口收到 activity_id：只有 /admin/task-activities 能生成分组。
	ErrTaskActivityIDForbidden = errors.New("TASK_ACTIVITY_ID_FORBIDDEN")
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

// 给 handler 用的只读别名：错误判定走 errors.Is，不靠 err.Error() 拼字符串。
var (
	ErrTaskTargetInvalid  = errTaskTargetInvalid
	ErrTaskStarJarMissing = errTaskStarJarMissing
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

// ==================== 班级星星罐学期口径（design D6 / tasks 3.8、7.5） ====================

// classStarJarTerm 是班级星星罐的学期口径。
// Term 是唯一键的一部分（class_star_jars 的 idx_class_star_jar），Start/End 用于按学期统计进度。
type classStarJarTerm struct {
	Term  string
	Start time.Time
	End   time.Time
}

// classStarJarSemester 统一学期取值：与 settings_service.go 同口径，非 fall 一律 spring。
// 直接 TrimSpace+EqualFold 再判定，避免 "FALL" / " fall " 退化成 spring 之外的第三种行为。
func classStarJarSemester(raw string) string {
	if strings.EqualFold(strings.TrimSpace(raw), "fall") {
		return "fall"
	}
	return "spring"
}

// parseTaskDateInLoc 解析 YYYY-MM-DD；空串或格式不符返回 false（调用方走兜底）。
func parseTaskDateInLoc(s string, loc *time.Location) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	t, err := time.ParseInLocation("2006-01-02", s, loc)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// parseAcademicYearSpan 解析 "2031-2032" 这类学年；返回起止年。
// 不强制两年相邻：历史数据里出现过跨两年的写法，强行校验只会把班级打成「无学年」。
func parseAcademicYearSpan(ay string) (int, int, bool) {
	parts := strings.Split(strings.TrimSpace(ay), "-")
	if len(parts) != 2 {
		return 0, 0, false
	}
	start, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	end, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	// 只接受正数且严格递增：`2032-2031` / `2031-2031` 这类脏值会让 fall 兜底
	// 产出 term_start > term_end，宁可整段退化成自然年，也不要给出倒置区间。
	if err1 != nil || err2 != nil || start <= 0 || end <= 0 || end <= start {
		return 0, 0, false
	}
	return start, end, true
}

// classStarJarTermOf 解析班级「当前学期」的 term 与日期范围。
//
// 口径（design D6 第 99 行）：Class.AcademicYear + KindergartenAppSetting.CurrentSemester；
// AcademicYear 为空时用自然年。范围优先取 academic_years 的学期起止，缺失时按自然学期兜底：
// spring = 次年 2/1–7/31，fall = 当年 9/1–次年 1/31。
//
// **只读**：配置行缺失时按默认值计算，不在读取路径写库（§20.4 的 seed 教训）。
func classStarJarTermOf(ctx context.Context, db *gorm.DB, kindergartenID, classID uint64, now time.Time) (classStarJarTerm, error) {
	loc := shanghaiLoc()
	now = now.In(loc)

	var cls struct {
		AcademicYear string `gorm:"column:academic_year"`
	}
	err := db.WithContext(ctx).Table("classes").
		Select("academic_year").
		Where("id = ? AND kindergarten_id = ? AND deleted_at IS NULL", classID, kindergartenID).
		Take(&cls).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return classStarJarTerm{}, ErrClassNotFound
	}
	if err != nil {
		return classStarJarTerm{}, err
	}

	semester := "spring"
	var setting struct {
		CurrentSemester string `gorm:"column:current_semester"`
	}
	err = db.WithContext(ctx).Table("kindergarten_app_settings").
		Select("current_semester").
		Where("kindergarten_id = ? AND deleted_at IS NULL", kindergartenID).
		Take(&setting).Error
	switch {
	case err == nil:
		semester = classStarJarSemester(setting.CurrentSemester)
	case errors.Is(err, gorm.ErrRecordNotFound):
		// 未配置园所设置：按 spring，与 kindergarten_app_settings 的列默认值一致。
	default:
		return classStarJarTerm{}, err
	}

	ay := strings.TrimSpace(cls.AcademicYear)
	startYear, endYear, ok := parseAcademicYearSpan(ay)
	if !ok {
		// 没有学年：用自然年（1/1–12/31），term 用当年年份，避免和真实学年键撞车。
		y := now.Year()
		return classStarJarTerm{
			Term:  fmt.Sprintf("%d-%s", y, semester),
			Start: time.Date(y, 1, 1, 0, 0, 0, 0, loc),
			End:   time.Date(y, 12, 31, 0, 0, 0, 0, loc),
		}, nil
	}

	// 查询与 term 都用解析后的规范形式：库里可能存成 `2031 - 2032`，
	// 直接拿原串去查 academic_years 会永远不命中，term 也会带上空格变成另一个键。
	normAY := fmt.Sprintf("%d-%d", startYear, endYear)
	term := struct {
		SpringStart string `gorm:"column:semester_spring_start"`
		SpringEnd   string `gorm:"column:semester_spring_end"`
		FallStart   string `gorm:"column:semester_fall_start"`
		FallEnd     string `gorm:"column:semester_fall_end"`
	}{}
	err = db.WithContext(ctx).Table("academic_years").
		Select("semester_spring_start, semester_spring_end, semester_fall_start, semester_fall_end").
		Where("kindergarten_id = ? AND year = ? AND deleted_at IS NULL", kindergartenID, normAY).
		Take(&term).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return classStarJarTerm{}, err
	}
	if err == nil {
		var cs, ce string
		if semester == "fall" {
			cs, ce = term.FallStart, term.FallEnd
		} else {
			cs, ce = term.SpringStart, term.SpringEnd
		}
		start, ok1 := parseTaskDateInLoc(cs, loc)
		end, ok2 := parseTaskDateInLoc(ce, loc)
		if ok1 && ok2 && !end.Before(start) {
			return classStarJarTerm{
				Term:  fmt.Sprintf("%s-%s", normAY, semester),
				Start: start,
				End:   end,
			}, nil
		}
	}

	// 兜底自然学期：区间一律从**起始年**推，保证 fall 一定跨到次年 1 月。
	// 用 endYear 推会让非相邻学年（`2031-2035`）算出 4 年多，同年/倒序学年
	// 更会直接倒置——这两类脏值已在 parseAcademicYearSpan 里被拒，这里再兜一层。
	if semester == "fall" {
		return classStarJarTerm{
			Term:  fmt.Sprintf("%s-fall", normAY),
			Start: time.Date(startYear, 9, 1, 0, 0, 0, 0, loc),
			End:   time.Date(startYear+1, 1, 31, 0, 0, 0, 0, loc),
		}, nil
	}
	return classStarJarTerm{
		Term:  fmt.Sprintf("%s-spring", normAY),
		Start: time.Date(startYear+1, 2, 1, 0, 0, 0, 0, loc),
		End:   time.Date(startYear+1, 7, 31, 0, 0, 0, 0, loc),
	}, nil
}

// validateClassStarJar 校验班级**当前学期**是否有可用的星星罐目标。
//
// 必须带 term：只按 goal>0 匹配会让上学期的目标继续放行本学期的
// StarScope=class 任务，而学期进度又是按当前学期统计的——任务能建、星星永远从 0 开始。
func validateClassStarJar(ctx context.Context, db *gorm.DB, kindergartenID, classID uint64) error {
	term, err := classStarJarTermOf(ctx, db, kindergartenID, classID, time.Now())
	if err != nil {
		return err
	}
	var n int64
	if err := db.WithContext(ctx).Model(&domain.ClassStarJar{}).
		Where("kindergarten_id = ? AND class_id = ? AND term = ? AND goal > 0",
			kindergartenID, classID, term.Term).
		Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return errTaskStarJarMissing
	}
	return nil
}

// findClassStarJar 读取班级当前学期的星星罐；未配置时返回 nil（不是错误）。
func findClassStarJar(ctx context.Context, db *gorm.DB, kindergartenID, classID uint64, now time.Time) (*domain.ClassStarJar, classStarJarTerm, error) {
	term, err := classStarJarTermOf(ctx, db, kindergartenID, classID, now)
	if err != nil {
		return nil, classStarJarTerm{}, err
	}
	var row domain.ClassStarJar
	err = db.WithContext(ctx).
		Where("kindergarten_id = ? AND class_id = ? AND term = ?", kindergartenID, classID, term.Term).
		Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, term, nil
	}
	if err != nil {
		return nil, term, err
	}
	return &row, term, nil
}

// upsertClassStarJar 设置班级当前学期目标。
//
// 唯一索引包含软删行：命中 ON DUPLICATE 时必须同时清 deleted_at，
// 否则「删过再配」会写成功但读不到（同 growth_likes / task_read_marks 的历史回归）。
func upsertClassStarJar(ctx context.Context, db *gorm.DB, kindergartenID, classID, userID uint64, goal int, now time.Time) (classStarJarTerm, error) {
	// 先解析班级再判目标：跨园所的班级必须拿到 404，而不是被「目标 0」的 422 盖掉。
	term, err := classStarJarTermOf(ctx, db, kindergartenID, classID, now)
	if err != nil {
		return classStarJarTerm{}, err
	}
	if goal <= 0 {
		return classStarJarTerm{}, ErrStarJarGoalInvalid
	}
	row := &domain.ClassStarJar{
		KindergartenID: kindergartenID,
		ClassID:        classID,
		Term:           term.Term,
		Goal:           goal,
		CreatedBy:      userID,
		UpdatedBy:      userID,
	}
	err = db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "kindergarten_id"}, {Name: "class_id"}, {Name: "term"}},
		DoUpdates: clause.Assignments(map[string]any{
			"goal":       goal,
			"updated_by": userID,
			"updated_at": now,
			"deleted_at": nil,
		}),
	}).Create(row).Error
	if err != nil {
		return classStarJarTerm{}, err
	}
	return term, nil
}

// validTaskStarGoal 星星目标下界。0 = 不启用，正数 = 目标；负数没有语义，
// 老师端与管理端必须是同一个下界，否则数据库里的不变量会随入口不同而破裂。
func validTaskStarGoal(goal int) bool {
	return goal >= 0
}

// validTaskRitual 任务默认仪式只接受三种取值；空串表示调用方未指定。
func validTaskRitual(ritual string) bool {
	switch ritual {
	case "", domain.TaskRitualSticker, domain.TaskRitualVoice, domain.TaskRitualClassStar:
		return true
	default:
		return false
	}
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
			return ErrTaskStructuralLocked
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
//
// 只处理**曾经有过公开状态**的提交（申请过 / 确认过 / 批准过）。
// 从未申请过的提交本来就没有要清的东西，给它们盖 public_revoked_at 会把
// 「从未申请」洗成「已撤回」，让 public_status 与撤回时间对家长和老师都失真。
// 这些行上 D7 要求的「清空」本来就是空操作，收窄不改变任何可见性。
func revokePublicForTask(ctx context.Context, tx *gorm.DB, taskID uint64) error {
	now := time.Now()
	return tx.WithContext(ctx).Model(&domain.TaskCheckIn{}).
		Where("task_id = ? AND (is_public_requested = ? OR public_scope_confirmed = ? OR public_approved_at IS NOT NULL)",
			taskID, true, true).
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
// submitterClassID 是**提交者孩子当前所在班级**（不是读者班级）：
//   - 公开墙要求「提交者班级 = 任务班级」，少了这条，转班或任务改派会把内容
//     漏给一个不该看到它的班级；
//   - 0 表示提交者学生记录缺失或已删除。公开墙的 join 拿不到班级时整行被排除，
//     所以这里也必须判不可见——否则老师/管理端会显示「已公开」而墙上没有。
//
// 「读者是否属于任务班级」由公开墙查询自己按读者 `student_id` 判定，不在本函数内。
func publicCheckinVisible(t *domain.ParentTask, c *domain.TaskCheckIn, submitterClassID uint64, today time.Time) bool {
	if t == nil || c == nil {
		return false
	}
	if t.DeletedAt.Valid || c.DeletedAt.Valid {
		return false
	}
	if !t.AllowPublic || t.Status != 1 || t.TargetStudentID > 0 {
		return false
	}
	if submitterClassID == 0 || t.ClassID != submitterClassID {
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

// checkinLocate 仅用于定位提交所属任务、孩子与孩子当前班级。
type checkinLocate struct {
	TaskID         uint64 `gorm:"column:task_id"`
	StudentID      uint64 `gorm:"column:student_id"`
	StudentClassID uint64 `gorm:"column:student_class_id"`
}

// locateCheckin 用**非锁定读**拿到提交的 task_id / student_id / 学生当前班级。
//
// 这里刻意不用 FOR UPDATE：公开相关的写路径必须以 parent_tasks 行作为第一把锁
// （见 handoff §12.2），而按 checkin_id 进来的请求事先不知道 task_id。
// InnoDB 的一致性非锁定读不取任何行锁，因此不参与死锁的加锁顺序；
// 真正的校验在拿到任务锁之后用 FOR UPDATE 重读一次完成。
//
// 顺带取学生班级：`public_visible` 要按公开墙同一套条件算（提交者班级 = 任务班级），
// 否则老师/管理端会看到「已公开」，而墙上其实因为转班已经隐藏。
func locateCheckin(ctx context.Context, tx *gorm.DB, checkinID uint64) (*checkinLocate, error) {
	var row checkinLocate
	if err := tx.WithContext(ctx).Table("task_check_ins AS c").
		Select("c.task_id, c.student_id, COALESCE(s.class_id, 0) AS student_class_id").
		Joins("LEFT JOIN students AS s ON s.id = c.student_id AND s.deleted_at IS NULL").
		Where("c.id = ? AND c.deleted_at IS NULL", checkinID).
		Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errTaskNotFound
		}
		return nil, err
	}
	return &row, nil
}
