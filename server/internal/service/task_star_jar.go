package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

// 班级星星罐：目标配置（tasks 3.8 teacher / 7.5 admin）与老师端进度（C-4）。
//
// 目标读写只做「按学期读/写 goal」这一件事：任务创建/更新时的「是否已配置」校验
// 仍然走 task_rules.go 的 validateClassStarJar，两个入口共用，不在这里另写一套。
//
// 进度（stars / progress / over_goal）只给老师端：管理端契约在 C-3 已定且不含进度
// （handoff §23.4-1），所以 GetClassStarJar 的响应保持逐字节不变，
// 老师端走 GetClassStarJarWithProgress。两边的 SQL 口径与 taskStarCounts 一致
// （decision §20.7-5），不另写一套「星星怎么算」。

// classStarJarView 读取与写入共用的响应形状。
func classStarJarView(classID uint64, term classStarJarTerm, row *domain.ClassStarJar) map[string]any {
	goal := 0
	configured := false
	var updatedAt any
	if row != nil {
		// goal=0 的历史行等同「未配置」：validateClassStarJar 只认 goal>0。
		// updated_at 也只在这时给值，与契约「未配置 → updated_at=null」一致。
		if row.Goal > 0 {
			goal = row.Goal
			configured = true
			updatedAt = row.UpdatedAt.UTC().Format(time.RFC3339)
		}
	}
	return map[string]any{
		"class_id":   classID,
		"term":       term.Term,
		"term_start": term.Start.Format("2006-01-02"),
		"term_end":   term.End.Format("2006-01-02"),
		"goal":       goal,
		"configured": configured,
		"updated_at": updatedAt,
	}
}

// loadClassStarJar 是「读目标 / 算进度」共用的入口守卫：园所必须有、班级必须存在。
// 两个读路径共用，避免同一组前置判断写两遍、日后只改一处。
func (s *TaskService) loadClassStarJar(
	ctx context.Context, kindergartenID, classID uint64, now time.Time,
) (*domain.ClassStarJar, classStarJarTerm, error) {
	if kindergartenID == 0 {
		return nil, classStarJarTerm{}, errors.New("NO_KINDERGARTEN")
	}
	if classID == 0 {
		return nil, classStarJarTerm{}, ErrClassNotFound
	}
	return findClassStarJar(ctx, s.db, kindergartenID, classID, now)
}

// GetClassStarJar 读取班级当前学期的星星罐目标；未配置返回 goal=0 / configured=false。
// 老师端与管理端共用：老师端只传自己带的班级，管理端传路径里的班级。
func (s *TaskService) GetClassStarJar(ctx context.Context, kindergartenID, classID uint64) (map[string]any, error) {
	row, term, err := s.loadClassStarJar(ctx, kindergartenID, classID, time.Now())
	if err != nil {
		return nil, err
	}
	return classStarJarView(classID, term, row), nil
}

// GetClassStarJarWithProgress 老师端读取：班级当前学期目标 + 已集星数 + 进度。
//
// 为什么不另开 /progress 端点（C-4 决定 A）：目标与星数必须来自同一次 term 解析。
// 分两次调用会在学期边界（无学年班级的 1/1、学年切换）读到两套 term，
// 表现为「目标 30、进度按上一学期算」。管理端仍用 GetClassStarJar，响应不变。
func (s *TaskService) GetClassStarJarWithProgress(
	ctx context.Context, kindergartenID, classID uint64,
) (map[string]any, error) {
	// 与 SetClassStarJar 同理：读目标与算星数共用一个 now，跨午夜/跨年时
	// 不会出现「旧 term 的目标 + 新 term 的星」。
	now := time.Now()
	row, term, err := s.loadClassStarJar(ctx, kindergartenID, classID, now)
	if err != nil {
		return nil, err
	}
	stars, err := classStarCountInTerm(ctx, s.db, kindergartenID, classID, term)
	if err != nil {
		return nil, err
	}
	view := classStarJarView(classID, term, row)
	view["stars"] = stars
	// 目标直接读 `row`，不从 view 里做类型断言：断言失败会让进度静默变成 null，
	// 与「未配置」无法区分。判据与 classStarJarView 一致：row 非空且 goal>0。
	if row != nil && row.Goal > 0 {
		goal := int64(row.Goal)
		// 向下取整：没到目标时永远不显示 100%。
		view["progress"] = int(min(stars*100/goal, 100))
		view["over_goal"] = stars > goal
	} else {
		view["progress"] = nil
		view["over_goal"] = false
	}
	return view, nil
}

// starCountBase 是「什么算一颗星」的唯一实现：提交行的主回应 ritual=class_star。
// taskStarCounts（按任务）与 classStarCountInTerm（按班级 + 学期）都从它出发——
// 同一谓词写两份，日后一边改了另一边忘改，就会变成「任务罐和班级罐算法不同」。
// 手写 Table 不走软删：提交与回应两侧各自补 deleted_at；学期等额外条件由调用方追加。
func starCountBase(ctx context.Context, db *gorm.DB, taskIDs []uint64) *gorm.DB {
	return db.WithContext(ctx).
		Table("task_check_ins AS c").
		// r.task_id = c.task_id：现有写入路径保证一致（dev 库实测 0 行不一致），
		// 但这里是任务罐与班级罐共用的唯一口径，一次错误回填就会静默多计。
		// 走主键比较，零成本。
		Joins("INNER JOIN task_responses AS r ON r.id = c.response_id AND r.task_id = c.task_id AND r.deleted_at IS NULL").
		Where("c.task_id IN ? AND c.deleted_at IS NULL AND r.ritual = ?",
			taskIDs, domain.TaskRitualClassStar)
}

// classStarCountInTerm 统计班级当前学期已集星数。
//
// 口径必须与 taskStarCounts（老师端任务列表的 star_count）一致，只是范围不同：
//   - 只算主回应：计数落在 task_check_ins 行上，response_id 指向的回应
//     ritual=class_star 才计入；period 回应覆盖 N 条提交就计 N 星；
//     补充回应不写 checkin.response_id，天然不计入；
//   - 提交、回应、任务三层都过滤软删；
//   - 只算 StarScope=class 的任务（design D6 / decision §20.7-5）；
//   - 「当前学期」按提交日期 c.date 判定：罐按学期重置，跨学期的长任务
//     按提交日拆到两边；用任务起始日会把整条任务算进某一边。
func classStarCountInTerm(
	ctx context.Context, db *gorm.DB,
	kindergartenID, classID uint64, term classStarJarTerm,
) (int64, error) {
	// 先取本班 class scope 任务（一个班一学期几十条），再按 task_id 计数：
	// 这样能走 (task_id, …) 前缀索引。写成一条三表 JOIN 时，优化器会从
	// task_responses 出发（ritual 无索引）——卡片每次 onShow 都调一次，
	// 数据一多就是「进一次页面扫一遍全表回应」。
	// Model(&ParentTask{}) 自带 deleted_at IS NULL，任务的软删过滤在这里。
	// 日期范围也要收在任务上：提交日期必然落在任务的起止区间内，所以
	// 「end_date < term.Start」或「start_date > term.End」的任务不可能贡献本学期星星。
	// 不加这两条，每次进页面都会把该班**历史上所有** class scope 任务 id 读出来。
	var taskIDs []uint64
	if err := db.WithContext(ctx).Model(&domain.ParentTask{}).
		Where("kindergarten_id = ? AND class_id = ? AND star_scope = ?",
			kindergartenID, classID, domain.TaskStarScopeClass).
		Where("end_date >= ? AND start_date <= ?",
			term.Start.Format("2006-01-02"), term.End.Format("2006-01-02")).
		Pluck("id", &taskIDs).Error; err != nil {
		return 0, err
	}
	if len(taskIDs) == 0 {
		return 0, nil
	}
	var n int64
	err := starCountBase(ctx, db, taskIDs).
		Where("c.date >= ? AND c.date <= ?",
			term.Start.Format("2006-01-02"), term.End.Format("2006-01-02")).
		Count(&n).Error
	return n, err
}

// SetClassStarJar 设置班级当前学期目标（必须 > 0）。
func (s *TaskService) SetClassStarJar(
	ctx context.Context,
	kindergartenID, classID, userID uint64,
	goal int,
	ip, ua string,
) (map[string]any, error) {
	if kindergartenID == 0 {
		return nil, errors.New("NO_KINDERGARTEN")
	}
	if classID == 0 {
		return nil, ErrClassNotFound
	}
	// 写入与回读共用同一个 now：跨 term 边界（无学年班级的 1 月 1 日）时，
	// 两次取时间会写进旧 term、读新 term，返回 200 却显示 goal=0。
	now := time.Now()
	term, err := upsertClassStarJar(ctx, s.db, kindergartenID, classID, userID, goal, now)
	if err != nil {
		return nil, err
	}
	row, _, err := findClassStarJar(ctx, s.db, kindergartenID, classID, now)
	if err != nil {
		return nil, err
	}
	_ = s.oplog.Create(ctx, &domain.OperationLog{
		KindergartenID: kindergartenID,
		UserID:         userID,
		Module:         "tasks",
		Action:         "set_star_jar",
		TargetType:     "class_star_jar",
		TargetID:       classID,
		Detail:         fmt.Sprintf("term=%s goal=%d", term.Term, goal),
		IP:             ip,
		UserAgent:      truncateUA(ua),
	})
	return classStarJarView(classID, term, row), nil
}
