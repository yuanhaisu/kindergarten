package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

var (
	ErrHealthCheckNotFound            = errors.New("HEALTH_CHECK_NOT_FOUND")
	ErrHealthCheckAlreadyDone         = errors.New("HEALTH_CHECK_ALREADY_COMPLETED")
	ErrHealthCheckInvalidItem         = errors.New("HEALTH_CHECK_INVALID_ITEM")
	ErrHealthCheckMediaInvalid        = errors.New("HEALTH_CHECK_MEDIA_INVALID")
	ErrHealthCheckTemperatureRequired = errors.New("HEALTH_CHECK_TEMPERATURE_REQUIRED")
)

type HealthCheckService struct {
	db       *gorm.DB
	hc       *repository.HealthCheckRepo
	msgRepo  *repository.MessageRepo
	fileRepo *repository.FileRepo
	policy   HealthPolicyProvider
}

func NewHealthCheckService(db *gorm.DB, hc *repository.HealthCheckRepo, msgRepo *repository.MessageRepo, fileRepo *repository.FileRepo, policy HealthPolicyProvider) *HealthCheckService {
	return &HealthCheckService{db: db, hc: hc, msgRepo: msgRepo, fileRepo: fileRepo, policy: policy}
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

// HealthCheckQueueResult 医生队列 + 当前园所体温策略。
// 策略与队列同一次下发，避免页面出现「策略已改但仍是旧值」。
type HealthCheckQueueResult struct {
	List   []HealthCheckQueueItem `json:"list"`
	Policy HealthPolicy           `json:"policy"`
}

func (s *HealthCheckService) GetQueueWithPolicy(ctx context.Context, kgID uint64, classID *uint64, date string) (*HealthCheckQueueResult, error) {
	items, err := s.GetQueue(ctx, kgID, classID, date)
	if err != nil {
		return nil, err
	}
	policy, err := s.policy.GetHealthPolicy(ctx, kgID)
	if err != nil {
		return nil, err
	}
	return &HealthCheckQueueResult{List: items, Policy: policy}, nil
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
	Remark        string   `json:"remark"`
	Skip          *bool    `json:"skip"` // true = 标记为未到园
}

// HealthCheckSubmitResult 提交结果：晨检详情 + 家长通知投递情况。
// 嵌入详情保持响应顶层字段不变，只增两个通知计数。
type HealthCheckSubmitResult struct {
	*HealthCheckDetail
	NotifiedCount     int `json:"notified_count"`
	NotifyFailedCount int `json:"notify_failed_count"`
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

func (s *HealthCheckService) SubmitCheck(ctx context.Context, kgID, doctorID, checkID uint64, req SubmitHealthCheckReq) (*HealthCheckSubmitResult, error) {
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
		detail, err := s.buildDetail(ctx, check)
		if err != nil {
			return nil, err
		}
		return &HealthCheckSubmitResult{HealthCheckDetail: detail}, nil
	}

	// 取值校验：越界直接拒绝，不落库
	if err := validateHealthItems(req); err != nil {
		return nil, err
	}

	policy, err := s.policy.GetHealthPolicy(ctx, kgID)
	if err != nil {
		return nil, err
	}

	// 合并到当前记录（未提交的字段保持原值），再按合并后的值派生标签
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
	if policy.TemperatureRequired && check.Temperature == nil {
		return nil, ErrHealthCheckTemperatureRequired
	}
	check.Remark = req.Remark
	check.SymptomTags = deriveSymptomTags(check, policy.FeverThreshold)
	check.OperatorID = &doctorID
	check.CheckStatus = domain.HealthCheckCompleted

	// 条件更新：已完成的记录不接受第二次写入。
	// 先读后写存在并发窗口，让数据库做最终裁决。
	res := s.db.WithContext(ctx).Model(&domain.MorningHealthCheck{}).
		Where("id = ? AND kindergarten_id = ? AND check_status <> ?", checkID, kgID, domain.HealthCheckCompleted).
		Updates(map[string]any{
			"temperature":    check.Temperature,
			"throat_status":  check.ThroatStatus,
			"hand_status":    check.HandStatus,
			"mouth_status":   check.MouthStatus,
			"eye_status":     check.EyeStatus,
			"skin_status":    check.SkinStatus,
			"overall_status": check.OverallStatus,
			"is_isolated":    check.IsIsolated,
			"symptom_tags":   check.SymptomTags,
			"remark":         check.Remark,
			"operator_id":    check.OperatorID,
			"check_status":   domain.HealthCheckCompleted,
		})
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		// 区分「记录不存在」与「已完成」，避免把并发竞争报成成功
		current, findErr := s.hc.FindByID(ctx, kgID, checkID)
		if findErr != nil {
			return nil, findErr
		}
		if current == nil {
			return nil, ErrHealthCheckNotFound
		}
		return nil, ErrHealthCheckAlreadyDone
	}

	updated, err := s.hc.FindByID(ctx, kgID, checkID)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, ErrHealthCheckNotFound
	}
	detail, err := s.buildDetail(ctx, updated)
	if err != nil {
		return nil, err
	}

	// 通知失败不影响晨检结论，但必须在响应中可见
	sent, failed := s.notifyAbnormalParents(ctx, updated, detail)
	return &HealthCheckSubmitResult{
		HealthCheckDetail: detail,
		NotifiedCount:     sent,
		NotifyFailedCount: failed,
	}, nil
}

func validateHealthItems(req SubmitHealthCheckReq) error {
	items := []struct {
		name  string
		value *int8
		max   int8
	}{
		{"throat_status", req.ThroatStatus, 2},
		{"hand_status", req.HandStatus, 3},
		{"mouth_status", req.MouthStatus, 3},
		{"eye_status", req.EyeStatus, 2},
		{"skin_status", req.SkinStatus, 3},
	}
	for _, item := range items {
		if item.value == nil {
			continue
		}
		if *item.value < 0 || *item.value > item.max {
			return fmt.Errorf("%w: %s 取值必须在 0-%d 之间", ErrHealthCheckInvalidItem, item.name, item.max)
		}
	}
	if req.OverallStatus < 0 || req.OverallStatus > 2 {
		return fmt.Errorf("%w: overall_status 取值必须在 0-2 之间", ErrHealthCheckInvalidItem)
	}
	return nil
}

// deriveSymptomTags 由体温与五项检查值派生症状标签。
// 3=其他 没有对应的权威标签，不臆造；家长视图通过 readableAbnormalItems 仍能看到。
func deriveSymptomTags(check *domain.MorningHealthCheck, feverThreshold float64) string {
	tags := make([]string, 0, 8)
	if check.Temperature != nil && *check.Temperature >= feverThreshold {
		tags = append(tags, "fever")
	}
	switch check.ThroatStatus {
	case 1:
		tags = append(tags, "red_throat")
	case 2:
		tags = append(tags, "swollen_throat")
	}
	switch check.HandStatus {
	case 1:
		tags = append(tags, "hand_rash")
	case 2:
		tags = append(tags, "hand_blister")
	}
	switch check.MouthStatus {
	case 1:
		tags = append(tags, "mouth_ulcer")
	case 2:
		tags = append(tags, "mouth_blister")
	}
	switch check.EyeStatus {
	case 1:
		tags = append(tags, "red_eye")
	case 2:
		tags = append(tags, "eye_discharge")
	}
	switch check.SkinStatus {
	case 1:
		tags = append(tags, "skin_rash")
	case 2:
		tags = append(tags, "skin_blister")
	}
	return strings.Join(tags, ",")
}

func (s *HealthCheckService) notifyAbnormalParents(ctx context.Context, check *domain.MorningHealthCheck, detail *HealthCheckDetail) (sent, failed int) {
	if check.OverallStatus == domain.HealthOverallHealthy {
		return 0, 0
	}
	parents := parentUserIDs(ctx, s.db, check.StudentID)
	if len(parents) == 0 {
		return 0, 0
	}

	conclusion := overallStatusLabel(check.OverallStatus)
	title := fmt.Sprintf("晨检提醒：%s %s", detail.StudentName, conclusion)
	if check.OverallStatus == domain.HealthOverallReject && check.IsIsolated {
		title = "【紧急】" + title
	}
	content := fmt.Sprintf("今日（%s）晨检结论为「%s」。", normalizeDate(check.Date), conclusion)
	if items := readableAbnormalItems(check); len(items) > 0 {
		labels := make([]string, 0, len(items))
		for _, item := range items {
			labels = append(labels, item.PartLabel+item.StatusLabel)
		}
		content += "异常项：" + strings.Join(labels, "、") + "。"
	}
	if check.Remark != "" {
		content += "保健医备注：" + check.Remark + "。"
	}
	content += "请在「我的 → 幼儿档案」查看详情。"

	return sendMessage(ctx, s.msgRepo, parents, domain.MessageTypeSystem, title, content, check.ID, true)
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

var healthMediaCategories = map[string]struct{}{
	"throat": {}, "hand": {}, "mouth": {}, "eye": {}, "skin": {},
}

// AddMedia 把已上传的文件关联到晨检记录。
//
// 文件经 POST /common/upload 上传（biz_type=health），这里只接收 file_id；
// URL/缩略图/大小从 files 行读取，避免客户端伪造。
// 租户边界：晨检记录必须属于当前园所；文件必须是当前操作者上传的晨检文件。
func (s *HealthCheckService) AddMedia(ctx context.Context, kgID, checkID, fileID, operatorID uint64, category, refDesc string) (*MediaItem, error) {
	if _, ok := healthMediaCategories[category]; !ok {
		return nil, fmt.Errorf("%w: 未知拍摄部位 %q", ErrHealthCheckMediaInvalid, category)
	}
	check, err := s.hc.FindByID(ctx, kgID, checkID)
	if err != nil {
		return nil, err
	}
	if check == nil {
		return nil, ErrHealthCheckNotFound
	}

	file, err := s.fileRepo.FindByID(ctx, fileID)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, fmt.Errorf("%w: 文件不存在", ErrHealthCheckMediaInvalid)
	}
	if file.BizType != "health" {
		return nil, fmt.Errorf("%w: 文件不属于晨检业务", ErrHealthCheckMediaInvalid)
	}
	if file.UserID != operatorID {
		return nil, fmt.Errorf("%w: 只能关联本人上传的文件", ErrHealthCheckMediaInvalid)
	}

	m := &domain.HealthCheckMedia{
		CheckID: checkID, FileID: &fileID,
		FileURL: file.FileURL, ThumbURL: file.ThumbURL,
		Category: category, RefDesc: refDesc,
		FileName: file.FileName, FileSize: file.FileSize,
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
