package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"kindergarten-server/internal/domain"
)

// 班级星星罐目标配置（tasks 3.8 teacher / 7.5 admin）。
//
// 只做「按学期读写目标」这一件事：任务创建/更新时的「是否已配置」校验仍然
// 走 task_rules.go 的 validateClassStarJar，两个入口共用，不在这里另写一套。
// 进度统计属 C-4（decision §20.7-5），本文件不返回星星数——用未过滤学期的
// 数字冒充班级罐进度会在跨学期时算错。

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

// GetClassStarJar 读取班级当前学期的星星罐目标；未配置返回 goal=0 / configured=false。
// 老师端与管理端共用：老师端只传自己带的班级，管理端传路径里的班级。
func (s *TaskService) GetClassStarJar(ctx context.Context, kindergartenID, classID uint64) (map[string]any, error) {
	if kindergartenID == 0 {
		return nil, errors.New("NO_KINDERGARTEN")
	}
	if classID == 0 {
		return nil, ErrClassNotFound
	}
	row, term, err := findClassStarJar(ctx, s.db, kindergartenID, classID, time.Now())
	if err != nil {
		return nil, err
	}
	return classStarJarView(classID, term, row), nil
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
