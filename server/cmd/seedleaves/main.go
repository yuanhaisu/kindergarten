package main

// seedleaves 灌入演示用请假记录。
//
// 这段逻辑原先挂在 LeaveService 的读取路径上（ListLeaves / GetLeaveStats），
// 只要有人打开后台的请假列表或统计面板，就会往生产表写入这批假数据。
// 现改为人工触发。
//
// 用法：
//
//	go run ./cmd/seedleaves                  # 默认 kindergarten 4
//	SEED_KG_ID=7 go run ./cmd/seedleaves     # 指定园所
//
// 幂等：该园请假记录已满 demoLeaveCount 条时跳过，不追加、不覆盖。

import (
	"context"
	"fmt"
	"log"
	"time"

	"kindergarten-server/cmd/internal/seedutil"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

const (
	// demoLeaveCount 是补足到的目标条数，同时也是 statuses 的长度。
	demoLeaveCount = 6
	// demoReviewerID 是假数据里扮演审核人的用户 id，仅用于演示。
	demoReviewerID uint64 = 4
)

func main() {
	// 先校验入参、再连库：非法 SEED_KG_ID 应当立刻报错，而不是等连库失败后才发现。
	kgID := seedutil.KindergartenID()
	loc := seedutil.Shanghai()
	db := seedutil.Open()

	if err := seed(context.Background(), repository.NewLeaveRepo(db), kgID, time.Now().In(loc), loc); err != nil {
		log.Fatal("seed:", err)
	}
}

func seed(ctx context.Context, repo *repository.LeaveRepo, kindergartenID uint64, anchor time.Time, loc *time.Location) error {
	total, err := repo.CountByKindergarten(ctx, kindergartenID)
	if err != nil {
		return fmt.Errorf("count existing: %w", err)
	}
	if total >= demoLeaveCount {
		fmt.Printf("kindergarten %d 已有 %d 条请假记录（阈值 %d），跳过\n",
			kindergartenID, total, demoLeaveCount)
		return nil
	}

	students, err := repo.ListSeedStudents(ctx, kindergartenID, 12)
	if err != nil {
		return fmt.Errorf("list students: %w", err)
	}
	if len(students) == 0 {
		return fmt.Errorf("kindergarten %d 没有可用于演示的学生", kindergartenID)
	}

	day := time.Date(anchor.Year(), anchor.Month(), anchor.Day(), 0, 0, 0, 0, loc)
	statuses := []domain.LeaveStatus{
		domain.LeaveStatusPending,
		domain.LeaveStatusPending,
		domain.LeaveStatusApproved,
		domain.LeaveStatusApproved,
		domain.LeaveStatusRejected,
		domain.LeaveStatusCanceled,
	}
	reasons := []string{"感冒发烧", "家中临时有事", "门诊复查", "外地探亲", "身体不适", "私事处理"}

	now := time.Now().UTC()
	leaves := make([]domain.LeaveRequest, 0, len(statuses))
	for i, st := range statuses {
		sr := students[i%len(students)]
		start := day.AddDate(0, 0, -(i % 3))

		lr := domain.LeaveRequest{
			KindergartenID: kindergartenID,
			ClassID:        sr.ClassID,
			StudentID:      sr.StudentID,
			ApplicantID:    sr.ApplicantID,
			Reason:         reasons[i%len(reasons)],
			StartDate:      start,
			EndDate:        start,
			Days:           1,
			Status:         st,
		}
		// 只有已审核状态才带审核信息，避免造出「待审核却有审核人」的矛盾数据。
		if st != domain.LeaveStatusPending && st != domain.LeaveStatusCanceled {
			reviewer := demoReviewerID
			lr.ReviewerID = &reviewer
			lr.ReviewedAt = &now
			lr.ReviewRemark = "系统初始化基线数据"
		}
		leaves = append(leaves, lr)
	}

	if err := repo.CreateBatch(ctx, leaves); err != nil {
		return fmt.Errorf("create batch: %w", err)
	}
	fmt.Printf("已为 kindergarten %d 灌入 %d 条请假记录\n", kindergartenID, len(leaves))
	return nil
}
