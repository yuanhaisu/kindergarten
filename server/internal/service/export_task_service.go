package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

type ExportService struct {
	tasks      *repository.ExportTaskRepo
	attendance *AttendanceService
	students   *StudentService
	growth     *GrowthService
}

func NewExportService(
	tasks *repository.ExportTaskRepo,
	attendance *AttendanceService,
	students *StudentService,
	growth *GrowthService,
) *ExportService {
	return &ExportService{
		tasks:      tasks,
		attendance: attendance,
		students:   students,
		growth:     growth,
	}
}

func newExportPublicID() (string, error) {
	var b [10]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "export_" + hex.EncodeToString(b[:]), nil
}

const exportFileTTL = 72 * time.Hour

type attendanceExportParams struct {
	ClassID   *uint64 `json:"class_id"`
	StartDate string  `json:"start_date"`
	EndDate   string  `json:"end_date"`
	Format    string  `json:"format"`
}

type studentsExportParams struct {
	ClassID *uint64 `json:"class_id"`
	Status  *int    `json:"status"`
	Format  string  `json:"format"`
}

type growthExportParams struct {
	ClassID   *uint64 `json:"class_id"`
	StartDate string  `json:"start_date"`
	EndDate   string  `json:"end_date"`
	Format    string  `json:"format"`
}

func exportStorageDir() (string, error) {
	base := filepath.Join(os.TempDir(), "kindergarten-export-tasks")
	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", err
	}
	return base, nil
}

// CreateAttendanceExport POST /admin/export/attendance
func (s *ExportService) CreateAttendanceExport(ctx context.Context, kindergartenID uint64, raw json.RawMessage) (map[string]interface{}, error) {
	if kindergartenID == 0 {
		return nil, errors.New("NO_KINDERGARTEN")
	}
	var p attendanceExportParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, errors.New("BAD_JSON")
	}
	if strings.TrimSpace(p.StartDate) == "" || strings.TrimSpace(p.EndDate) == "" {
		return nil, errors.New("BAD_DATE")
	}
	if p.ClassID != nil && *p.ClassID > 0 {
		if err := s.attendance.ValidateClassInKindergarten(ctx, kindergartenID, *p.ClassID); err != nil {
			return nil, err
		}
	}
	return s.enqueue(ctx, kindergartenID, "attendance", raw)
}

// CreateStudentsExport POST /admin/export/students
func (s *ExportService) CreateStudentsExport(ctx context.Context, kindergartenID uint64, raw json.RawMessage) (map[string]interface{}, error) {
	if kindergartenID == 0 {
		return nil, errors.New("NO_KINDERGARTEN")
	}
	var p studentsExportParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, errors.New("BAD_JSON")
	}
	if p.ClassID != nil && *p.ClassID > 0 {
		if err := s.attendance.ValidateClassInKindergarten(ctx, kindergartenID, *p.ClassID); err != nil {
			return nil, err
		}
	}
	return s.enqueue(ctx, kindergartenID, "students", raw)
}

// CreateGrowthExport POST /admin/export/growth
func (s *ExportService) CreateGrowthExport(ctx context.Context, kindergartenID uint64, raw json.RawMessage) (map[string]interface{}, error) {
	if kindergartenID == 0 {
		return nil, errors.New("NO_KINDERGARTEN")
	}
	var p growthExportParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, errors.New("BAD_JSON")
	}
	if strings.TrimSpace(p.StartDate) == "" || strings.TrimSpace(p.EndDate) == "" {
		return nil, errors.New("BAD_DATE")
	}
	if p.ClassID != nil && *p.ClassID > 0 {
		if err := s.attendance.ValidateClassInKindergarten(ctx, kindergartenID, *p.ClassID); err != nil {
			return nil, err
		}
	}
	return s.enqueue(ctx, kindergartenID, "growth", raw)
}

func (s *ExportService) enqueue(ctx context.Context, kindergartenID uint64, typ string, raw json.RawMessage) (map[string]interface{}, error) {
	pub, err := newExportPublicID()
	if err != nil {
		return nil, err
	}
	row := &domain.ExportTask{
		KindergartenID: kindergartenID,
		PublicID:       pub,
		Type:           typ,
		Status:         "processing",
		Progress:       5,
		ParamsJSON:     string(raw),
	}
	if err := s.tasks.Create(ctx, row); err != nil {
		return nil, err
	}
	id := row.ID
	go s.runExportTask(id)
	return map[string]interface{}{
		"task_id": pub,
		"status":  "processing",
		"message": "报表生成中，完成后可下载",
	}, nil
}

func (s *ExportService) runExportTask(taskID uint64) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	t, err := s.tasks.FindByID(ctx, taskID)
	if err != nil || t == nil {
		return
	}
	dir, err := exportStorageDir()
	if err != nil {
		_ = s.tasks.MarkFailed(ctx, taskID, "无法创建导出目录")
		return
	}
	localPath := filepath.Join(dir, t.PublicID+".xlsx")
	var data []byte
	var fileName string
	switch t.Type {
	case "attendance":
		var p attendanceExportParams
		if json.Unmarshal([]byte(t.ParamsJSON), &p) != nil {
			_ = s.tasks.MarkFailed(ctx, taskID, "参数无效")
			return
		}
		_ = s.tasks.UpdateStatus(ctx, taskID, "processing", 40)
		data, fileName, err = s.attendance.ExportReport(ctx, t.KindergartenID, p.ClassID, p.StartDate, p.EndDate, p.Format)
	case "students":
		var p studentsExportParams
		if json.Unmarshal([]byte(t.ParamsJSON), &p) != nil {
			_ = s.tasks.MarkFailed(ctx, taskID, "参数无效")
			return
		}
		var st *int8
		if p.Status != nil {
			v := int8(*p.Status)
			st = &v
		}
		_ = s.tasks.UpdateStatus(ctx, taskID, "processing", 40)
		data, fileName, err = s.students.ExportStudentsXLSX(ctx, t.KindergartenID, p.ClassID, st)
	case "growth":
		var p growthExportParams
		if json.Unmarshal([]byte(t.ParamsJSON), &p) != nil {
			_ = s.tasks.MarkFailed(ctx, taskID, "参数无效")
			return
		}
		_ = s.tasks.UpdateStatus(ctx, taskID, "processing", 40)
		data, fileName, err = s.growth.ExportGrowthLogsXLSX(ctx, t.KindergartenID, p.ClassID, p.StartDate, p.EndDate, p.Format)
	default:
		_ = s.tasks.MarkFailed(ctx, taskID, "未知任务类型")
		return
	}
	if err != nil {
		msg := err.Error()
		if len(msg) > 400 {
			msg = msg[:400]
		}
		_ = s.tasks.MarkFailed(ctx, taskID, msg)
		return
	}
	if err := os.WriteFile(localPath, data, 0o644); err != nil {
		_ = s.tasks.MarkFailed(ctx, taskID, "写入导出文件失败")
		return
	}
	fi, _ := os.Stat(localPath)
	sz := int64(0)
	if fi != nil {
		sz = fi.Size()
	}
	now := time.Now().UTC()
	exp := now.Add(exportFileTTL)
	_ = s.tasks.MarkCompleted(ctx, taskID, localPath, fileName, sz, exp, now)
}

// GetExportTaskStatus GET /admin/export/:task_id
func (s *ExportService) GetExportTaskStatus(ctx context.Context, kindergartenID uint64, publicID string) (map[string]interface{}, error) {
	if kindergartenID == 0 {
		return nil, errors.New("NO_KINDERGARTEN")
	}
	t, err := s.tasks.FindByPublicID(ctx, kindergartenID, publicID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, errors.New("NOT_FOUND")
	}
	out := map[string]interface{}{
		"task_id":    t.PublicID,
		"status":     mapExportStatus(t.Status),
		"created_at": t.CreatedAt.UTC().Format(time.RFC3339),
	}
	if t.Status == "processing" {
		out["progress"] = t.Progress
		if t.Progress < 10 {
			out["progress"] = 35
		}
	}
	if t.Status == "completed" {
		out["status"] = "completed"
		out["progress"] = 100
		out["download_url"] = "/admin/export/download/" + t.PublicID
		out["file_name"] = t.FileName
		out["file_size"] = t.FileSize
		if t.ExpireAt != nil {
			out["expire_at"] = t.ExpireAt.UTC().Format(time.RFC3339)
		}
		if t.CompletedAt != nil {
			out["completed_at"] = t.CompletedAt.UTC().Format(time.RFC3339)
		}
	}
	if t.Status == "failed" {
		out["status"] = "failed"
		if t.ErrorMessage != "" {
			out["message"] = t.ErrorMessage
		}
	}
	return out, nil
}

func mapExportStatus(st string) string {
	switch st {
	case "processing":
		return "processing"
	case "completed":
		return "completed"
	case "failed":
		return "failed"
	default:
		return st
	}
}

// GetExportFileForDownload 校验园所后返回本地路径与文件名（流式下载）。
func (s *ExportService) GetExportFileForDownload(ctx context.Context, kindergartenID uint64, publicID string) (localPath, fileName string, err error) {
	if kindergartenID == 0 {
		return "", "", errors.New("NO_KINDERGARTEN")
	}
	t, err := s.tasks.FindByPublicID(ctx, kindergartenID, publicID)
	if err != nil {
		return "", "", err
	}
	if t == nil {
		return "", "", errors.New("NOT_FOUND")
	}
	if t.Status != "completed" || t.LocalPath == "" {
		return "", "", errors.New("NOT_READY")
	}
	if t.ExpireAt != nil && time.Now().After(*t.ExpireAt) {
		return "", "", errors.New("EXPIRED")
	}
	return t.LocalPath, t.FileName, nil
}
