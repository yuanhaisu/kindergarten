package service

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

var (
	ErrHealthCheckNotFound    = errors.New("HEALTH_CHECK_NOT_FOUND")
	ErrHealthCheckAlreadyDone = errors.New("HEALTH_CHECK_ALREADY_COMPLETED")
)

type HealthCheckService struct {
	db *gorm.DB
	hc *repository.HealthCheckRepo
}

func NewHealthCheckService(db *gorm.DB, hc *repository.HealthCheckRepo) *HealthCheckService {
	return &HealthCheckService{db: db, hc: hc}
}

// ==================== Queue ====================

type HealthCheckQueueItem struct {
	ID             uint64 `json:"id"`
	PickupRecordID uint64 `json:"pickup_record_id"`
	StudentID      uint64 `json:"student_id"`
	StudentName    string `json:"student_name"`
	ClassName      string `json:"class_name"`
	ClassID        uint64 `json:"class_id"`
	CheckStatus    int8   `json:"check_status"`
	OverallStatus  int8   `json:"overall_status"`
	PickupTime     string `json:"pickup_time"`
	Avatar         string `json:"avatar"`
}

func (s *HealthCheckService) GetQueue(ctx context.Context, kgID uint64, classID *uint64, date string) ([]HealthCheckQueueItem, error) {
	rows, err := s.hc.ListQueue(ctx, kgID, classID, date)
	if err != nil {
		return nil, err
	}
	items := make([]HealthCheckQueueItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, HealthCheckQueueItem{
			ID: r.ID, PickupRecordID: r.PickupRecordID,
			StudentID: r.StudentID, StudentName: r.StudentName,
			ClassName: r.ClassName, ClassID: r.ClassID,
			CheckStatus: r.CheckStatus, OverallStatus: r.OverallStatus,
			PickupTime: r.PickupTime, Avatar: r.Avatar,
		})
	}
	return items, nil
}

// ==================== Submit ====================

type SubmitHealthCheckReq struct {
	Temperature   *float64 `json:"temperature"`
	ThroatStatus  *int8    `json:"throat_status"`
	HandStatus    *int8    `json:"hand_status"`
	MouthStatus   *int8    `json:"mouth_status"`
	EyeStatus     *int8    `json:"eye_status"`
	SkinStatus    *int8    `json:"skin_status"`
	OverallStatus int8     `json:"overall_status"`
	IsIsolated    *bool    `json:"is_isolated"`
	SymptomTags   string   `json:"symptom_tags"`
	Remark        string   `json:"remark"`
	MediaIDs      []uint64 `json:"media_ids"`
	Skip          *bool    `json:"skip"` // true = 标记为未到园
}

type HealthCheckDetail struct {
	ID             uint64      `json:"id"`
	PickupRecordID uint64      `json:"pickup_record_id"`
	StudentID      uint64      `json:"student_id"`
	StudentName    string      `json:"student_name"`
	ClassName      string      `json:"class_name"`
	CheckStatus    int8        `json:"check_status"`
	Temperature    *float64    `json:"temperature"`
	ThroatStatus   int8        `json:"throat_status"`
	HandStatus     int8        `json:"hand_status"`
	MouthStatus    int8        `json:"mouth_status"`
	EyeStatus      int8        `json:"eye_status"`
	SkinStatus     int8        `json:"skin_status"`
	OverallStatus  int8        `json:"overall_status"`
	IsIsolated     bool        `json:"is_isolated"`
	SymptomTags    string      `json:"symptom_tags"`
	Remark         string      `json:"remark"`
	Media          []MediaItem `json:"media"`
}

type MediaItem struct {
	ID       uint64 `json:"id"`
	FileURL  string `json:"file_url"`
	ThumbURL string `json:"thumb_url"`
	Category string `json:"category"`
}

func (s *HealthCheckService) SubmitCheck(ctx context.Context, kgID, doctorID, checkID uint64, req SubmitHealthCheckReq) (*HealthCheckDetail, error) {
	check, err := s.hc.FindByID(ctx, kgID, checkID)
	if err != nil {
		return nil, err
	}
	if check == nil {
		return nil, ErrHealthCheckNotFound
	}
	if check.CheckStatus == domain.HealthCheckCompleted {
		return nil, ErrHealthCheckAlreadyDone
	}

	// Handle skip (child not arrived)
	if req.Skip != nil && *req.Skip {
		check.CheckStatus = domain.HealthCheckSkipped
		if err := s.hc.Update(ctx, check); err != nil {
			return nil, err
		}
		return s.buildDetail(ctx, check)
	}

	// Update fields
	if req.Temperature != nil {
		check.Temperature = req.Temperature
	}
	if req.ThroatStatus != nil {
		check.ThroatStatus = domain.HealthCheckItem(*req.ThroatStatus)
	}
	if req.HandStatus != nil {
		check.HandStatus = domain.HealthCheckItem(*req.HandStatus)
	}
	if req.MouthStatus != nil {
		check.MouthStatus = domain.HealthCheckItem(*req.MouthStatus)
	}
	if req.EyeStatus != nil {
		check.EyeStatus = domain.HealthCheckItem(*req.EyeStatus)
	}
	if req.SkinStatus != nil {
		check.SkinStatus = domain.HealthCheckItem(*req.SkinStatus)
	}
	check.OverallStatus = domain.HealthCheckOverall(req.OverallStatus)
	if req.IsIsolated != nil {
		check.IsIsolated = *req.IsIsolated
	}
	check.SymptomTags = req.SymptomTags
	check.Remark = req.Remark
	check.CheckStatus = domain.HealthCheckCompleted
	check.OperatorID = &doctorID

	if err := s.hc.Update(ctx, check); err != nil {
		return nil, err
	}

	// If rejected, mark pickup record as abnormal
	if check.OverallStatus == domain.HealthOverallReject {
		var pr domain.PickupRecord
		if err := s.db.WithContext(ctx).Where("id = ?", check.PickupRecordID).First(&pr).Error; err == nil {
			pr.IsAbnormal = true
			reason := "晨检劝返"
			if check.SymptomTags != "" {
				reason = fmt.Sprintf("晨检劝返: %s", check.SymptomTags)
			}
			pr.AbnormalReason = reason
			s.db.WithContext(ctx).Save(&pr)
		}
	}

	return s.buildDetail(ctx, check)
}

func (s *HealthCheckService) GetDetail(ctx context.Context, kgID, checkID uint64) (*HealthCheckDetail, error) {
	check, err := s.hc.FindByID(ctx, kgID, checkID)
	if err != nil {
		return nil, err
	}
	if check == nil {
		return nil, ErrHealthCheckNotFound
	}
	return s.buildDetail(ctx, check)
}

func (s *HealthCheckService) buildDetail(ctx context.Context, check *domain.MorningHealthCheck) (*HealthCheckDetail, error) {
	type nameRow struct {
		StudentName string `gorm:"column:student_name"`
		ClassName   string `gorm:"column:class_name"`
	}
	var nr nameRow
	s.db.WithContext(ctx).Raw(`
		SELECT s.name AS student_name, c.name AS class_name
		FROM students s INNER JOIN classes c ON c.id = s.class_id
		WHERE s.id = ? AND s.kindergarten_id = ? AND s.deleted_at IS NULL`,
		check.StudentID, check.KindergartenID).Scan(&nr)

	media, _ := s.hc.ListMediaByCheckID(ctx, check.ID)
	mitems := make([]MediaItem, 0, len(media))
	for _, m := range media {
		mitems = append(mitems, MediaItem{ID: m.ID, FileURL: m.FileURL, ThumbURL: m.ThumbURL, Category: m.Category})
	}

	return &HealthCheckDetail{
		ID: check.ID, PickupRecordID: check.PickupRecordID,
		StudentID: check.StudentID, StudentName: nr.StudentName, ClassName: nr.ClassName,
		CheckStatus: int8(check.CheckStatus),
		Temperature: check.Temperature,
		ThroatStatus: int8(check.ThroatStatus), HandStatus: int8(check.HandStatus),
		MouthStatus: int8(check.MouthStatus), EyeStatus: int8(check.EyeStatus),
		SkinStatus: int8(check.SkinStatus), OverallStatus: int8(check.OverallStatus),
		IsIsolated: check.IsIsolated, SymptomTags: check.SymptomTags, Remark: check.Remark,
		Media: mitems,
	}, nil
}

// ==================== Media ====================

func (s *HealthCheckService) AddMedia(ctx context.Context, checkID uint64, fileURL, thumbURL, category, fileName string, fileSize int64) (*MediaItem, error) {
	m := &domain.HealthCheckMedia{
		CheckID: checkID, FileURL: fileURL, ThumbURL: thumbURL,
		Category: category, FileName: fileName, FileSize: fileSize,
	}
	if err := s.hc.CreateMedia(ctx, m); err != nil {
		return nil, err
	}
	return &MediaItem{ID: m.ID, FileURL: m.FileURL, ThumbURL: m.ThumbURL, Category: m.Category}, nil
}

// ==================== Stats ====================

type HealthStatsResp struct {
	Date          string                `json:"date"`
	ByClass       []HealthCheckStatsRow `json:"by_class"`
	Symptoms      []SymptomCount        `json:"symptoms"`
	TotalStudents int64                 `json:"total_students"`
}

type HealthCheckStatsRow struct {
	ClassID   uint64 `json:"class_id"`
	ClassName string `json:"class_name"`
	Total     int64  `json:"total"`
	Pending   int64  `json:"pending"`
	Completed int64  `json:"completed"`
	Healthy   int64  `json:"healthy"`
	Observe   int64  `json:"observe"`
	Reject    int64  `json:"reject"`
}

type SymptomCount struct {
	Tag   string `json:"tag"`
	Count int64  `json:"count"`
}

func (s *HealthCheckService) GetStats(ctx context.Context, kgID uint64, date string) (*HealthStatsResp, error) {
	byClass, err := s.hc.StatsByDate(ctx, kgID, date)
	if err != nil {
		return nil, err
	}
	symptoms, err := s.hc.SymptomRanking(ctx, kgID, date)
	if err != nil {
		return nil, err
	}

	rows := make([]HealthCheckStatsRow, 0, len(byClass))
	var total int64
	for _, r := range byClass {
		rows = append(rows, HealthCheckStatsRow{
			ClassID: r.ClassID, ClassName: r.ClassName,
			Total: r.Total, Pending: r.Pending, Completed: r.Completed,
			Healthy: r.Healthy, Observe: r.Observe, Reject: r.Reject,
		})
		total += r.Total
	}

	sc := make([]SymptomCount, 0, len(symptoms))
	for _, s := range symptoms {
		sc = append(sc, SymptomCount{Tag: s.Tag, Count: s.Count})
	}

	return &HealthStatsResp{Date: date, ByClass: rows, Symptoms: sc, TotalStudents: total}, nil
}
