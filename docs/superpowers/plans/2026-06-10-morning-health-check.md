# 入园晨检 Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement morning health check recording with doctor role, auto-created pending checks on pickup, photo upload for abnormalities, and admin statistics.

**Architecture:** New doctor role with RBAC-isolated endpoints under `/api/v1/doctor/health/*`. MorningHealthCheck auto-created (status=pending) when PickupRecord(type=1) is created. Doctor completes check via mini-program page in existing teacher app. Admin views stats via web dashboard.

**Tech Stack:** Go (Gin/GORM/Wire), uni-app Vue 3 (teacher mini-program), Nuxt 3 (admin web)

---

### Task 1: Domain Models

**Files:**
- Create: `server/internal/domain/health_check.go`

- [ ] **Step 1: Create domain models**

```go
package domain

// HealthCheckStatus 晨检状态
type HealthCheckStatus int8
const (
	HealthCheckPending   HealthCheckStatus = 0 // 待检
	HealthCheckCompleted HealthCheckStatus = 1 // 已完成
	HealthCheckSkipped   HealthCheckStatus = 2 // 已跳过(未到园)
)

// HealthCheckOverall 综合判断
type HealthCheckOverall int8
const (
	HealthOverallHealthy  HealthCheckOverall = 0 // 健康
	HealthOverallObserve  HealthCheckOverall = 1 // 需观察
	HealthOverallReject   HealthCheckOverall = 2 // 劝返就医
)

// HealthCheckItem 单项检查（0=正常）
type HealthCheckItem int8
const (
	HealthItemNormal      HealthCheckItem = 0
	HealthItemThroatRed   HealthCheckItem = 1 // 喉咙发红
	HealthItemThroatSwollen HealthCheckItem = 2 // 喉咙红肿
	HealthItemHandRash    HealthCheckItem = 1 // 手部红疹
	HealthItemHandBlister HealthCheckItem = 2 // 手部疱疹
	HealthItemMouthUlcer  HealthCheckItem = 1 // 口腔溃疡
	HealthItemMouthBlister HealthCheckItem = 2 // 口腔疱疹
	HealthItemEyeRed      HealthCheckItem = 1 // 眼睛发红
	HealthItemEyeDischarge HealthCheckItem = 2 // 眼睛分泌物
	HealthItemSkinRash    HealthCheckItem = 1 // 皮肤红疹
	HealthItemSkinBlister HealthCheckItem = 2 // 皮肤水疱
)

// MorningHealthCheck 晨检记录
type MorningHealthCheck struct {
	SoftDeleteModel
	KindergartenID uint64             `gorm:"not null;index;column:kindergarten_id" json:"kindergarten_id"`
	ClassID        uint64             `gorm:"not null;index;column:class_id" json:"class_id"`
	StudentID      uint64             `gorm:"not null;index;column:student_id" json:"student_id"`
	PickupRecordID uint64             `gorm:"not null;uniqueIndex;column:pickup_record_id" json:"pickup_record_id"`
	Date           string             `gorm:"type:date;not null;index;column:date" json:"date"`
	CheckStatus    HealthCheckStatus  `gorm:"not null;default:0;column:check_status" json:"check_status"`
	Temperature    *float64           `gorm:"type:decimal(3,1);column:temperature" json:"temperature"`
	ThroatStatus   HealthCheckItem    `gorm:"default:0;column:throat_status" json:"throat_status"`
	HandStatus     HealthCheckItem    `gorm:"default:0;column:hand_status" json:"hand_status"`
	MouthStatus    HealthCheckItem    `gorm:"default:0;column:mouth_status" json:"mouth_status"`
	EyeStatus      HealthCheckItem    `gorm:"default:0;column:eye_status" json:"eye_status"`
	SkinStatus     HealthCheckItem    `gorm:"default:0;column:skin_status" json:"skin_status"`
	OverallStatus  HealthCheckOverall `gorm:"default:0;column:overall_status" json:"overall_status"`
	IsIsolated     bool               `gorm:"default:false;column:is_isolated" json:"is_isolated"`
	SymptomTags    string             `gorm:"size:300;column:symptom_tags" json:"symptom_tags"`
	Remark         string             `gorm:"size:500;column:remark" json:"remark"`
	OperatorID     *uint64            `gorm:"column:operator_id" json:"operator_id"`
}

func (MorningHealthCheck) TableName() string { return "morning_health_checks" }

// HealthCheckMedia 晨检异常照片
type HealthCheckMedia struct {
	SoftDeleteModel
	CheckID  uint64 `gorm:"not null;index;column:check_id" json:"check_id"`
	FileURL  string `gorm:"size:500;column:file_url" json:"file_url"`
	ThumbURL string `gorm:"size:500;column:thumb_url" json:"thumb_url"`
	Category string `gorm:"size:32;column:category" json:"category"`
	FileName string `gorm:"size:200;column:file_name" json:"file_name"`
	FileSize int64  `gorm:"column:file_size" json:"file_size"`
}

func (HealthCheckMedia) TableName() string { return "health_check_media" }
```

- [ ] **Step 2: Verify compilation**

```bash
cd server && go build ./...
```
Expected: compiles (though no code uses these types yet)

---

### Task 2: Repository

**Files:**
- Create: `server/internal/repository/health_check_repo.go`
- Modify: `server/internal/repository/wire.go`

- [ ] **Step 1: Create repository**

```go
package repository

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"kindergarten-server/internal/domain"
)

type HealthCheckRepo struct{ db *gorm.DB }

func NewHealthCheckRepo(db *gorm.DB) *HealthCheckRepo { return &HealthCheckRepo{db: db} }

// --- Queue ---
type HealthCheckQueueRow struct {
	ID             uint64 `gorm:"column:id"`
	PickupRecordID uint64 `gorm:"column:pickup_record_id"`
	StudentID      uint64 `gorm:"column:student_id"`
	StudentName    string `gorm:"column:student_name"`
	ClassName      string `gorm:"column:class_name"`
	ClassID        uint64 `gorm:"column:class_id"`
	CheckStatus    int8   `gorm:"column:check_status"`
	OverallStatus  int8   `gorm:"column:overall_status"`
	PickupTime     string `gorm:"column:pickup_time"`
	Avatar         string `gorm:"column:avatar"`
}

func (r *HealthCheckRepo) ListQueue(ctx context.Context, kgID uint64, classID *uint64, date string) ([]HealthCheckQueueRow, error) {
	base := r.db.WithContext(ctx).Table("morning_health_checks AS hc").
		Joins("INNER JOIN students AS s ON s.id = hc.student_id AND s.deleted_at IS NULL").
		Joins("INNER JOIN classes AS c ON c.id = hc.class_id AND c.deleted_at IS NULL").
		Joins("INNER JOIN pickup_records AS pr ON pr.id = hc.pickup_record_id AND pr.deleted_at IS NULL").
		Where("hc.kindergarten_id = ? AND hc.date = ? AND hc.deleted_at IS NULL", kgID, date)
	if classID != nil && *classID > 0 {
		base = base.Where("hc.class_id = ?", *classID)
	}
	var rows []HealthCheckQueueRow
	err := base.Select(`hc.id, hc.pickup_record_id, hc.student_id, s.name AS student_name,
		c.name AS class_name, c.id AS class_id, hc.check_status, hc.overall_status,
		COALESCE(s.avatar,'') AS avatar,
		TIME_FORMAT(pr.created_at, '%H:%i') AS pickup_time`).
		Order("hc.check_status ASC, pr.created_at ASC").Find(&rows).Error
	return rows, err
}

// --- CRUD ---
func (r *HealthCheckRepo) FindByID(ctx context.Context, kgID, id uint64) (*domain.MorningHealthCheck, error) {
	var row domain.MorningHealthCheck
	err := r.db.WithContext(ctx).Where("id = ? AND kindergarten_id = ?", id, kgID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) { return nil, nil }
	return &row, err
}

func (r *HealthCheckRepo) FindByPickupRecordID(ctx context.Context, prID uint64) (*domain.MorningHealthCheck, error) {
	var row domain.MorningHealthCheck
	err := r.db.WithContext(ctx).Where("pickup_record_id = ?", prID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) { return nil, nil }
	return &row, err
}

func (r *HealthCheckRepo) Create(ctx context.Context, row *domain.MorningHealthCheck) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *HealthCheckRepo) Update(ctx context.Context, row *domain.MorningHealthCheck) error {
	return r.db.WithContext(ctx).Save(row).Error
}

// --- Media ---
func (r *HealthCheckRepo) CreateMedia(ctx context.Context, m *domain.HealthCheckMedia) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *HealthCheckRepo) ListMediaByCheckID(ctx context.Context, checkID uint64) ([]domain.HealthCheckMedia, error) {
	var rows []domain.HealthCheckMedia
	err := r.db.WithContext(ctx).Where("check_id = ?", checkID).Order("id ASC").Find(&rows).Error
	return rows, err
}

// --- Stats ---
type HealthCheckStatsRow struct {
	ClassID    uint64 `gorm:"column:class_id"`
	ClassName  string `gorm:"column:class_name"`
	Total      int64  `gorm:"column:total"`
	Pending    int64  `gorm:"column:pending"`
	Completed  int64  `gorm:"column:completed"`
	Healthy    int64  `gorm:"column:healthy"`
	Observe    int64  `gorm:"column:observe"`
	Reject     int64  `gorm:"column:reject"`
}

func (r *HealthCheckRepo) StatsByDate(ctx context.Context, kgID uint64, date string) ([]HealthCheckStatsRow, error) {
	var rows []HealthCheckStatsRow
	err := r.db.WithContext(ctx).Raw(`
	SELECT hc.class_id, c.name AS class_name,
		COUNT(*) AS total,
		SUM(CASE WHEN hc.check_status = 0 THEN 1 ELSE 0 END) AS pending,
		SUM(CASE WHEN hc.check_status = 1 THEN 1 ELSE 0 END) AS completed,
		SUM(CASE WHEN hc.overall_status = 0 AND hc.check_status = 1 THEN 1 ELSE 0 END) AS healthy,
		SUM(CASE WHEN hc.overall_status = 1 THEN 1 ELSE 0 END) AS observe,
		SUM(CASE WHEN hc.overall_status = 2 THEN 1 ELSE 0 END) AS reject
	FROM morning_health_checks hc
	INNER JOIN classes c ON c.id = hc.class_id AND c.deleted_at IS NULL
	WHERE hc.kindergarten_id = ? AND hc.date = ? AND hc.deleted_at IS NULL
	GROUP BY hc.class_id, c.name
	ORDER BY c.name`, kgID, date).Scan(&rows).Error
	return rows, err
}

type SymptomCount struct {
	Tag   string `gorm:"column:tag"`
	Count int64  `gorm:"column:cnt"`
}

func (r *HealthCheckRepo) SymptomRanking(ctx context.Context, kgID uint64, date string) ([]SymptomCount, error) {
	// Get all symptom_tags for the date, aggregate in Go
	var tags []string
	err := r.db.WithContext(ctx).Model(&domain.MorningHealthCheck{}).
		Where("kindergarten_id = ? AND date = ? AND symptom_tags != '' AND deleted_at IS NULL", kgID, date).
		Pluck("symptom_tags", &tags).Error
	if err != nil { return nil, err }
	// count each tag
	m := make(map[string]int64)
	for _, t := range tags {
		// t is comma-separated, split and count
		for _, tag := range splitTags(t) {
			m[tag]++
		}
	}
	var out []SymptomCount
	for k, v := range m { out = append(out, SymptomCount{Tag: k, Count: v}) }
	return out, nil
}

func splitTags(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			if i > start { parts = append(parts, s[start:i]) }
			start = i + 1
		}
	}
	if start < len(s) { parts = append(parts, s[start:]) }
	return parts
}
```

- [ ] **Step 2: Register in Wire set**

In `server/internal/repository/wire.go`, add `NewHealthCheckRepo` to the Set:
```go
NewHealthCheckRepo,
```

- [ ] **Step 3: Verify**

```bash
cd server && go build ./...
```

---

### Task 3: AutoMigrate

**Files:**
- Modify: `server/internal/app/provider.go`

- [ ] **Step 1: Add tables to AutoMigrate**

In `ProvideGorm()`, add after the pickup block:
```go
// 晨检
&domain.MorningHealthCheck{},
&domain.HealthCheckMedia{},
```

- [ ] **Step 2: Verify**

```bash
cd server && go build ./...
```

---

### Task 4: Service Layer

**Files:**
- Create: `server/internal/service/health_check_service.go`
- Modify: `server/internal/service/wire.go`

- [ ] **Step 1: Create service**

```go
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
	ErrHealthCheckNotFound = errors.New("HEALTH_CHECK_NOT_FOUND")
	ErrHealthCheckAlreadyDone = errors.New("HEALTH_CHECK_ALREADY_COMPLETED")
)

type HealthCheckService struct {
	db *gorm.DB
	hc *repository.HealthCheckRepo
	pr *repository.PickupRepo
}

func NewHealthCheckService(db *gorm.DB, hc *repository.HealthCheckRepo, pr *repository.PickupRepo) *HealthCheckService {
	return &HealthCheckService{db: db, hc: hc, pr: pr}
}

// ========== Queue ==========

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
	if err != nil { return nil, err }
	items := make([]HealthCheckQueueItem, len(rows))
	for i, r := range rows {
		items[i] = HealthCheckQueueItem{
			ID: r.ID, PickupRecordID: r.PickupRecordID,
			StudentID: r.StudentID, StudentName: r.StudentName,
			ClassName: r.ClassName, ClassID: r.ClassID,
			CheckStatus: r.CheckStatus, OverallStatus: r.OverallStatus,
			PickupTime: r.PickupTime, Avatar: r.Avatar,
		}
	}
	return items, nil
}

// ========== Check Submission ==========

type SubmitHealthCheckReq struct {
	Temperature   *float64 `json:"temperature"`
	ThroatStatus  *int8    `json:"throat_status"`
	HandStatus    *int8    `json:"hand_status"`
	MouthStatus   *int8    `json:"mouth_status"`
	EyeStatus     *int8    `json:"eye_status"`
	SkinStatus    *int8    `json:"skin_status"`
	OverallStatus int8     `json:"overall_status" binding:"oneof=0 1 2"`
	IsIsolated    *bool    `json:"is_isolated"`
	SymptomTags   string   `json:"symptom_tags"`
	Remark        string   `json:"remark"`
	MediaIDs      []uint64 `json:"media_ids"`
}

type HealthCheckDetail struct {
	ID             uint64  `json:"id"`
	PickupRecordID uint64  `json:"pickup_record_id"`
	StudentID      uint64  `json:"student_id"`
	StudentName    string  `json:"student_name"`
	ClassName      string  `json:"class_name"`
	CheckStatus    int8    `json:"check_status"`
	Temperature    *float64 `json:"temperature"`
	ThroatStatus   int8    `json:"throat_status"`
	HandStatus     int8    `json:"hand_status"`
	MouthStatus    int8    `json:"mouth_status"`
	EyeStatus      int8    `json:"eye_status"`
	SkinStatus     int8    `json:"skin_status"`
	OverallStatus  int8    `json:"overall_status"`
	IsIsolated     bool    `json:"is_isolated"`
	SymptomTags    string  `json:"symptom_tags"`
	Remark         string  `json:"remark"`
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
	if err != nil { return nil, err }
	if check == nil { return nil, ErrHealthCheckNotFound }
	if check.CheckStatus == domain.HealthCheckCompleted { return nil, ErrHealthCheckAlreadyDone }

	// Update fields
	if req.Temperature != nil { check.Temperature = req.Temperature }
	if req.ThroatStatus != nil { check.ThroatStatus = domain.HealthCheckItem(*req.ThroatStatus) }
	if req.HandStatus != nil { check.HandStatus = domain.HealthCheckItem(*req.HandStatus) }
	if req.MouthStatus != nil { check.MouthStatus = domain.HealthCheckItem(*req.MouthStatus) }
	if req.EyeStatus != nil { check.EyeStatus = domain.HealthCheckItem(*req.EyeStatus) }
	if req.SkinStatus != nil { check.SkinStatus = domain.HealthCheckItem(*req.SkinStatus) }
	check.OverallStatus = domain.HealthCheckOverall(req.OverallStatus)
	if req.IsIsolated != nil { check.IsIsolated = *req.IsIsolated }
	check.SymptomTags = req.SymptomTags
	check.Remark = req.Remark
	check.CheckStatus = domain.HealthCheckCompleted
	check.OperatorID = &doctorID

	if err := s.hc.Update(ctx, check); err != nil { return nil, err }

	// Link media
	if len(req.MediaIDs) > 0 {
		s.db.WithContext(ctx).Model(&domain.HealthCheckMedia{}).
			Where("id IN ? AND check_id = ?", req.MediaIDs, checkID).
			Update("check_id", check.ID)
	}

	// Sync pickup record: 劝返 → mark abnormal
	if check.OverallStatus == domain.HealthOverallReject {
		pr, _ := s.pr.FindByID(ctx, check.PickupRecordID)
		if pr != nil {
			pr.IsAbnormal = true
			pr.AbnormalReason = fmt.Sprintf("晨检劝返: %s", strings.ReplaceAll(check.SymptomTags, ",", "、"))
			s.db.WithContext(ctx).Save(pr)
		}
	}

	return s.buildDetail(ctx, check)
}

func (s *HealthCheckService) GetDetail(ctx context.Context, kgID, checkID uint64) (*HealthCheckDetail, error) {
	check, err := s.hc.FindByID(ctx, kgID, checkID)
	if err != nil { return nil, err }
	if check == nil { return nil, ErrHealthCheckNotFound }
	return s.buildDetail(ctx, check)
}

func (s *HealthCheckService) buildDetail(ctx context.Context, check *domain.MorningHealthCheck) (*HealthCheckDetail, error) {
	// Load student + class names
	type nameRow struct { StudentName string `gorm:"column:student_name"`; ClassName string `gorm:"column:class_name"` }
	var nr nameRow
	s.db.WithContext(ctx).Raw(`SELECT s.name AS student_name, c.name AS class_name
		FROM students s INNER JOIN classes c ON c.id = s.class_id
		WHERE s.id = ? AND s.kindergarten_id = ?`, check.StudentID, check.KindergartenID).Scan(&nr)

	media, _ := s.hc.ListMediaByCheckID(ctx, check.ID)
	mitems := make([]MediaItem, len(media))
	for i, m := range media {
		mitems[i] = MediaItem{ID: m.ID, FileURL: m.FileURL, ThumbURL: m.ThumbURL, Category: m.Category}
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

// ========== Media Upload ==========

func (s *HealthCheckService) AddMedia(ctx context.Context, checkID uint64, fileURL, thumbURL, category, fileName string, fileSize int64) (*MediaItem, error) {
	m := &domain.HealthCheckMedia{
		CheckID: checkID, FileURL: fileURL, ThumbURL: thumbURL,
		Category: category, FileName: fileName, FileSize: fileSize,
	}
	if err := s.hc.CreateMedia(ctx, m); err != nil { return nil, err }
	return &MediaItem{ID: m.ID, FileURL: m.FileURL, ThumbURL: m.ThumbURL, Category: m.Category}, nil
}

// ========== Stats ==========

type HealthStatsResp struct {
	Date        string              `json:"date"`
	ByClass     []HealthCheckStatsRow `json:"by_class"`
	Symptoms    []SymptomCount      `json:"symptoms"`
	TotalStudents int64             `json:"total_students"`
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
	if err != nil { return nil, err }
	symptoms, err := s.hc.SymptomRanking(ctx, kgID, date)
	if err != nil { return nil, err }

	rows := make([]HealthCheckStatsRow, len(byClass))
	var total int64
	for i, r := range byClass {
		rows[i] = HealthCheckStatsRow{
			ClassID: r.ClassID, ClassName: r.ClassName,
			Total: r.Total, Pending: r.Pending, Completed: r.Completed,
			Healthy: r.Healthy, Observe: r.Observe, Reject: r.Reject,
		}
		total += r.Total
	}

	sc := make([]SymptomCount, len(symptoms))
	for i, s := range symptoms { sc[i] = SymptomCount{Tag: s.Tag, Count: s.Count} }

	return &HealthStatsResp{Date: date, ByClass: rows, Symptoms: sc, TotalStudents: total}, nil
}
```

- [ ] **Step 2: Register in Wire set**

In `server/internal/service/wire.go`, add:
```go
NewHealthCheckService,
```

- [ ] **Step 3: Verify**

```bash
cd server && go build ./...
```

---

### Task 5: Auto-Create Pending Check on Pickup

**Files:**
- Modify: `server/internal/service/teacher_portal_service.go`

- [ ] **Step 1: Add auto-create logic**

Find `CreatePickupRecord` method. After the record is created and before returning, add:

```go
// When type=DropOff(1), auto-create pending health check
if input.Type == domain.PickupTypeDropOff {
	healthCheck := &domain.MorningHealthCheck{
		KindergartenID: tc.KindergartenID,
		ClassID:        classID,
		StudentID:      input.StudentID,
		PickupRecordID: record.ID,
		Date:           record.Date.Format("2006-01-02"),
		CheckStatus:    domain.HealthCheckPending,
	}
	if err := s.db.Create(healthCheck).Error; err != nil {
		// Log but don't fail the pickup — health check creation is non-critical
		println("auto-create health check failed:", err.Error())
	}
}
```

Note: Need to import `"kindergarten-server/internal/domain"` if not already imported.

The `tc.KindergartenID` and `classID` are available from the teacher context. The `input.Type` field needs to be checked — if the input doesn't have it, default to DropOff in the existing code.

- [ ] **Step 2: Verify**

```bash
cd server && go build ./...
```

---

### Task 6: HTTP Handlers

**Files:**
- Create: `server/internal/handler/http/health_check.go`
- Modify: `server/internal/handler/http/handler.go`

- [ ] **Step 1: Create doctor + admin handlers**

```go
package handler

import (
	"errors"
	"strconv"
	"github.com/gin-gonic/gin"
	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/response"
)

// ============ Doctor Handlers ============

func (h *Handler) DoctorGetQueue(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 { response.ErrForbidden(c, "未绑定园所"); return }
	var classID *uint64
	if cid, err := strconv.ParseUint(c.Query("class_id"), 10, 64); err == nil && cid > 0 { classID = &cid }
	date := c.DefaultQuery("date", "")
	if date == "" { date = c.Query("date") } // frontend sends today's date
	items, err := h.healthCheckSvc.GetQueue(c.Request.Context(), kgID, classID, date)
	if err != nil { response.ErrInternal(c, "获取队列失败"); return }
	response.OK(c, gin.H{"list": items})
}

func (h *Handler) DoctorGetCheckDetail(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	checkID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || checkID == 0 { response.ErrValidation(c, "无效的 ID"); return }
	data, err := h.healthCheckSvc.GetDetail(c.Request.Context(), kgID, checkID)
	if err != nil {
		if errors.Is(err, service.ErrHealthCheckNotFound) { response.ErrNotFound(c, "记录不存在"); return }
		response.ErrInternal(c, "获取详情失败"); return
	}
	response.OK(c, data)
}

func (h *Handler) DoctorSubmitCheck(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	checkID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || checkID == 0 { response.ErrValidation(c, "无效的 ID"); return }
	var req service.SubmitHealthCheckReq
	if err := c.ShouldBindJSON(&req); err != nil { response.ErrValidation(c, "参数不合法"); return }
	userID := middleware.GetUserID(c)
	data, err := h.healthCheckSvc.SubmitCheck(c.Request.Context(), kgID, userID, checkID, req)
	if err != nil {
		if errors.Is(err, service.ErrHealthCheckNotFound) { response.ErrNotFound(c, "记录不存在"); return }
		if errors.Is(err, service.ErrHealthCheckAlreadyDone) { response.ErrValidation(c, "已完成晨检"); return }
		response.ErrInternal(c, "提交失败"); return
	}
	response.OK(c, data)
}

func (h *Handler) DoctorAddMedia(c *gin.Context) {
	checkID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || checkID == 0 { response.ErrValidation(c, "无效的 ID"); return }
	var req struct {
		FileURL  string `json:"file_url" binding:"required"`
		ThumbURL string `json:"thumb_url"`
		Category string `json:"category"`
		FileName string `json:"file_name"`
		FileSize int64  `json:"file_size"`
	}
	if err := c.ShouldBindJSON(&req); err != nil { response.ErrValidation(c, "参数不合法"); return }
	media, err := h.healthCheckSvc.AddMedia(c.Request.Context(), checkID, req.FileURL, req.ThumbURL, req.Category, req.FileName, req.FileSize)
	if err != nil { response.ErrInternal(c, "添加照片失败"); return }
	response.OK(c, media)
}

func (h *Handler) DoctorTodayStats(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	date := c.DefaultQuery("date", "")
	data, err := h.healthCheckSvc.GetStats(c.Request.Context(), kgID, date)
	if err != nil { response.ErrInternal(c, "获取统计失败"); return }
	response.OK(c, data)
}

// ============ Admin Handlers ============

func (h *Handler) AdminHealthStats(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	date := c.DefaultQuery("date", "")
	data, err := h.healthCheckSvc.GetStats(c.Request.Context(), kgID, date)
	if err != nil { response.ErrInternal(c, "获取统计失败"); return }
	response.OK(c, data)
}
```

- [ ] **Step 2: Add healthCheckSvc to Handler struct**

In `server/internal/handler/http/handler.go`:
- Add field: `healthCheckSvc *service.HealthCheckService`
- Add constructor param: `healthCheckSvc *service.HealthCheckService,`
- Add init: `healthCheckSvc: healthCheckSvc,`

- [ ] **Step 3: Verify**

```bash
cd server && go build ./...
```

---

### Task 7: Router + RBAC

**Files:**
- Modify: `server/internal/app/router.go`
- Modify: `server/configs/rbac_policy.csv`

- [ ] **Step 1: Register doctor routes**

In `registerTeacherRoutes` (or a new `registerDoctorRoutes` function), add after the teacher routes:

```go
// Doctor routes under teacher app's router group
doctorG := authed.Group("/doctor")
{
	doctor := doctorG.Group("/health")
	{
		doctor.GET("/queue", h.DoctorGetQueue)
		doctor.GET("/check/:id", h.DoctorGetCheckDetail)
		doctor.PUT("/check/:id", h.DoctorSubmitCheck)
		doctor.POST("/check/:id/media", h.DoctorAddMedia)
		doctor.GET("/today/stats", h.DoctorTodayStats)
	}
}
```

Wait — this needs to be in the authed group, not inside registerTeacherRoutes. Let me add it in `NewRouter` after the teacher routes:

```go
doctorG := authed.Group("/doctor")
registerDoctorRoutes(doctorG, h.inner)
```

And define:
```go
func registerDoctorRoutes(dg *gin.RouterGroup, h *handler.Handler) {
	health := dg.Group("/health")
	{
		health.GET("/queue", h.DoctorGetQueue)
		health.GET("/check/:id", h.DoctorGetCheckDetail)
		health.PUT("/check/:id", h.DoctorSubmitCheck)
		health.POST("/check/:id/media", h.DoctorAddMedia)
		health.GET("/today/stats", h.DoctorTodayStats)
	}
}
```

- [ ] **Step 2: Register admin health route**

In `registerAdminRoutes`, add:
```go
health := adminG.Group("/health")
{
	health.GET("/stats", h.AdminHealthStats)
}
```

- [ ] **Step 3: Add RBAC policies**

Add to `server/configs/rbac_policy.csv`:
```csv
p, doctor, /api/v1/auth/logout, POST
p, doctor, /api/v1/common/*, *
p, doctor, /api/v1/doctor/*, *
p, doctor, /api/v1/classes/*, GET
p, doctor, /api/v1/children/*, GET
```

---

### Task 8: Wire + Build Verification

- [ ] **Step 1: Regenerate Wire**

```bash
cd server && make wire
```

- [ ] **Step 2: Build all**

```bash
cd server && go build ./...
```

Expected: no errors.

---

### Task 9: Teacher Mini-Program — Doctor Health Check Page

**Files:**
- Create: `teacher/src/api/doctor-health.js`
- Create: `teacher/src/pages/doctor/checkin.vue`
- Modify: `teacher/src/pages.json`

- [ ] **Step 1: Create API wrapper**

`teacher/src/api/doctor-health.js`:
```javascript
import { requestWithAuth } from "@/api/http"

export function fetchHealthQueue(params = {}) {
  return requestWithAuth({ url: "/doctor/health/queue", method: "GET", data: params })
}

export function fetchCheckDetail(id) {
  return requestWithAuth({ url: `/doctor/health/check/${id}`, method: "GET" })
}

export function submitCheck(id, body) {
  return requestWithAuth({ url: `/doctor/health/check/${id}`, method: "PUT", data: body })
}

export function addCheckMedia(id, body) {
  return requestWithAuth({ url: `/doctor/health/check/${id}/media`, method: "POST", data: body })
}

export function fetchTodayStats(date) {
  return requestWithAuth({ url: "/doctor/health/today/stats", method: "GET", data: { date } })
}
```

- [ ] **Step 2: Create check-in page**

`teacher/src/pages/doctor/checkin.vue` — simplified version with:
- Top stats bar (pending/completed counts)
- Class filter tabs
- Pending students list (sorted by pickup time, showing wait duration)
- Completed students list (collapsed)
- Tapping a pending student opens the check card:
  - Temperature input (numeric keyboard)
  - 5 item rows (throat/hand/mouth/eye/skin) with status toggles
  - Camera button on abnormal items
  - Remark textarea
  - Action buttons: Healthy / Observe / Reject / Isolated
- Submit → PUT /doctor/health/check/:id → move student from pending to completed

(Full template code ~200 lines — content, styling, and script logic following same patterns as existing teacher pages.)

- [ ] **Step 3: Register page route**

In `teacher/src/pages.json`, add:
```json
{ "path": "pages/doctor/checkin", "style": { "navigationBarTitleText": "入园晨检" } }
```

- [ ] **Step 4: Build verify**

```bash
cd teacher && npm run build:mp-weixin
```

---

### Task 10: Admin — Health Stats Page

**Files:**
- Create: `admin/api/health.ts`
- Create: `admin/pages/health/index.vue`
- Modify: `admin/constants/menus.ts`
- Modify: `admin/types/api.d.ts`

- [ ] **Step 1: Add TypeScript types**

In `admin/types/api.d.ts`, add:
```typescript
export interface HealthStatsClassRow {
  class_id: number; class_name: string
  total: number; pending: number; completed: number
  healthy: number; observe: number; reject: number
}
export interface HealthSymptomCount { tag: string; count: number }
export interface HealthStats {
  date: string; by_class: HealthStatsClassRow[]
  symptoms: HealthSymptomCount[]; total_students: number
}
```

- [ ] **Step 2: Create API**

`admin/api/health.ts`:
```typescript
import type { HealthStats } from '~/types/api'
export function getHealthStats(params: { date?: string }) {
  return useHttp().get<HealthStats>('/admin/health/stats', params)
}
```

- [ ] **Step 3: Create stats page**

`admin/pages/health/index.vue` — showing:
- Date picker (defaults today)
- Summary cards: Total / Pending / Completed / Healthy / Observe / Reject
- Per-class table: class name, total, pending, completed, healthy, observe, reject
- Symptom chart bar (if any abnormal symptoms today)
- (minimal viable version: just stat cards + class table)

- [ ] **Step 4: Add menu entry**

In `admin/constants/menus.ts`, add under "数据中心" group:
```typescript
{ title: '晨检统计', path: '/health', icon: DataAnalysis, permission: 'dashboard:view' },
```

- [ ] **Step 5: Build verify**

```bash
cd admin && npm run build
```

---

### Task 11: Login Support for Doctor Role

- [ ] **Step 1: Verify WeChat login supports role=doctor**

The existing login flow `POST /auth/login/wechat` accepts `role` parameter and creates a `UserRole` entry. The `role` value is stored as-is in `user_roles.role`. So logging in with `role=doctor` should already work as long as a doctor user exists in the system.

No code changes needed — just document that doctors login via the teacher mini-program with `role=doctor`.

---

### Verification Checklist

- [ ] `cd server && go build ./...` — backend compiles
- [ ] `cd teacher && npm run build:mp-weixin` — teacher app compiles with doctor page
- [ ] `cd admin && npm run build` — admin web compiles with health page
- [ ] Start server → AutoMigrate creates morning_health_checks + health_check_media tables
- [ ] Teacher scans a child → PickupRecord created → MorningHealthCheck auto-created (pending)
- [ ] Doctor logs in → sees queue → completes check → PickupRecord abnormal if reject
- [ ] Admin opens /health → sees today's stats
