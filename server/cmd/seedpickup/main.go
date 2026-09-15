package main

// seedpickup 灌入演示用接送记录与授权人。
//
// 这段逻辑原先挂在 PickupService 的读取路径上（GetPickupOverview /
// ListPickupRecords / ListPickupAbnormal），只要有人打开接送面板就会写库。
// 现改为人工触发。
//
// 用法：
//
//	go run ./cmd/seedpickup                  # 默认 kindergarten 4
//	SEED_KG_ID=7 go run ./cmd/seedpickup     # 指定园所
//
// 两段各自幂等：当天已有接送记录则不追加；待审核授权人已达目标数则不追加。

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
	demoStudentLimit  = 16
	demoPendingAuths  = 2
	// demoOperatorID 扮演接送操作人（老师），仅用于演示。
	demoOperatorID uint64 = 4
	// demoCreatorID 扮演授权人创建者，仅用于演示。
	demoCreatorID uint64 = 4
	// authReviewPending 对应 domain.PickupAuthorization.ReviewStatus 的「0 待审核」
	// （domain/pickup.go 中有注释，但未导出常量）。
	authReviewPending int8 = 0
)

func main() {
	// 先校验入参、再连库：非法 SEED_KG_ID 应当立刻报错，而不是等连库失败后才发现。
	kgID := seedutil.KindergartenID()
	loc := seedutil.Shanghai()
	db := seedutil.Open()

	if err := seed(context.Background(), repository.NewPickupRepo(db), kgID, time.Now().In(loc), loc); err != nil {
		log.Fatal("seed:", err)
	}
}

func seed(ctx context.Context, repo *repository.PickupRepo, kindergartenID uint64, day time.Time, loc *time.Location) error {
	existing, err := repo.CountRecordsByDate(ctx, kindergartenID, day)
	if err != nil {
		return fmt.Errorf("count records: %w", err)
	}

	students, err := repo.ListSeedStudents(ctx, kindergartenID, demoStudentLimit)
	if err != nil {
		return fmt.Errorf("list students: %w", err)
	}
	if len(students) == 0 {
		return fmt.Errorf("kindergarten %d 没有可用于演示的学生", kindergartenID)
	}

	if err := seedRecords(ctx, repo, kindergartenID, day, loc, existing, students); err != nil {
		return err
	}
	return seedAuthorizations(ctx, repo, kindergartenID, students)
}

func seedRecords(
	ctx context.Context,
	repo *repository.PickupRepo,
	kindergartenID uint64,
	day time.Time,
	loc *time.Location,
	existing int64,
	students []repository.PickupSeedStudentRow,
) error {
	if existing > 0 {
		fmt.Printf("kindergarten %d 在 %s 已有 %d 条接送记录，跳过该段\n",
			kindergartenID, day.Format("2006-01-02"), existing)
		return nil
	}

	baseDay := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc)
	rows := make([]domain.PickupRecord, 0, len(students)*2)
	for i, st := range students {
		rows = append(rows, domain.PickupRecord{
			KindergartenID: kindergartenID,
			ClassID:        st.ClassID,
			StudentID:      st.StudentID,
			Date:           baseDay,
			Type:           domain.PickupTypeDropOff,
			Method:         domain.PickupMethodManual,
			PickupName:     "家长" + st.Name,
			PickupRelation: "家长",
			OperatorID:     demoOperatorID,
			RecordedAt:     time.Date(baseDay.Year(), baseDay.Month(), baseDay.Day(), 7, 40+(i%20), 0, 0, loc),
		})

		// 留最后两名学生不写接走记录，演示「未接走」状态。
		if i >= len(students)-2 {
			continue
		}
		abnormal := i < 2
		reason := ""
		if abnormal {
			reason = "非授权人员接送，待复核"
		}
		rows = append(rows, domain.PickupRecord{
			KindergartenID: kindergartenID,
			ClassID:        st.ClassID,
			StudentID:      st.StudentID,
			Date:           baseDay,
			Type:           domain.PickupTypePickUp,
			Method:         domain.PickupMethodManual,
			PickupName:     "家长" + st.Name,
			PickupRelation: "家长",
			OperatorID:     demoOperatorID,
			RecordedAt:     time.Date(baseDay.Year(), baseDay.Month(), baseDay.Day(), 16, 20+(i%30), 0, 0, loc),
			IsAbnormal:     abnormal,
			AbnormalReason: reason,
		})
	}

	if err := repo.CreateRecordBatch(ctx, rows); err != nil {
		return fmt.Errorf("create records: %w", err)
	}
	fmt.Printf("已为 kindergarten %d 灌入 %d 条接送记录\n", kindergartenID, len(rows))
	return nil
}

func seedAuthorizations(
	ctx context.Context,
	repo *repository.PickupRepo,
	kindergartenID uint64,
	students []repository.PickupSeedStudentRow,
) error {
	pending, err := repo.CountPendingAuthorizations(ctx, kindergartenID)
	if err != nil {
		return fmt.Errorf("count pending auths: %w", err)
	}
	if pending >= demoPendingAuths {
		fmt.Printf("kindergarten %d 已有 %d 个待审核授权人（目标 %d），跳过该段\n",
			kindergartenID, pending, demoPendingAuths)
		return nil
	}

	need := int(demoPendingAuths - pending)
	now := time.Now().UTC()
	authRows := make([]domain.PickupAuthorization, 0, need)
	for i := 0; i < need && i < len(students); i++ {
		st := students[i]
		authRows = append(authRows, domain.PickupAuthorization{
			StudentID:    st.StudentID,
			Name:         "临时接送人" + st.Name,
			Phone:        fmt.Sprintf("1390000%04d", i+1),
			Relation:     "亲属",
			Status:       1,
			CreatorID:    demoCreatorID,
			ReviewStatus: authReviewPending,
			ExpireAt:     &now,
		})
	}

	if err := repo.CreateAuthorizationBatch(ctx, authRows); err != nil {
		return fmt.Errorf("create authorizations: %w", err)
	}
	fmt.Printf("已为 kindergarten %d 灌入 %d 个待审核授权人\n", kindergartenID, len(authRows))
	return nil
}
