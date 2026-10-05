package service

import (
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	ossx "kindergarten-server/internal/storage/oss"
	"kindergarten-server/pkg/log"
)

const (
	maxSingleFileSize = 20 << 20 // 20MB
	maxBatchCount     = 20
)

var allowedTypes = map[string]string{
	".jpg": "image", ".jpeg": "image", ".png": "image", ".gif": "image", ".webp": "image",
	".mp4": "video", ".mov": "video", ".avi": "video",
	".mp3": "audio", ".wav": "audio", ".aac": "audio",
	".pdf": "document", ".doc": "document", ".docx": "document",
}

type FileService struct {
	fileRepo *repository.FileRepo
	oss      *ossx.Storage
}

func NewFileService(fileRepo *repository.FileRepo, oss *ossx.Storage) *FileService {
	return &FileService{fileRepo: fileRepo, oss: oss}
}

type UploadResult struct {
	ID       uint64 `json:"id"`
	FileURL  string `json:"file_url"`
	ThumbURL string `json:"thumb_url,omitempty"`
	FileName string `json:"file_name"`
	FileSize int64  `json:"file_size"`
	FileType string `json:"file_type"`
	Duration int    `json:"duration"`
}

type BatchUploadResult struct {
	Files        []UploadResult `json:"files"`
	SuccessCount int            `json:"success_count"`
	FailCount    int            `json:"fail_count"`
}

func (s *FileService) Upload(ctx context.Context, userID uint64, fh *multipart.FileHeader, bizType string) (*UploadResult, error) {
	if fh.Size > maxSingleFileSize {
		return nil, fmt.Errorf("UPLOAD_TOO_LARGE")
	}

	ext := strings.ToLower(filepath.Ext(fh.Filename))
	fileType, ok := allowedTypes[ext]
	if !ok {
		return nil, fmt.Errorf("UPLOAD_INVALID_TYPE")
	}

	objectKey := generateObjectKey(bizType, ext)
	contentType := fh.Header.Get("Content-Type")

	src, err := fh.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = src.Close() }()

	if err := s.oss.PutObject(objectKey, src, contentType); err != nil {
		log.L(ctx).Error("oss put object failed", "err", err, "key", objectKey)
		return nil, fmt.Errorf("OSS_UPLOAD_FAILED")
	}

	fileURL := s.oss.PublicURL(objectKey)
	thumbURL := ""
	if fileType == "image" {
		thumbURL = fileURL + "?x-oss-process=image/resize,w_200"
	}
	if fileType == "video" {
		thumbURL = fileURL + "?x-oss-process=video/snapshot,t_1000,w_200,m_fast"
	}

	f := &domain.File{
		UserID:   userID,
		BizType:  bizType,
		FileName: fh.Filename,
		FileURL:  fileURL,
		ThumbURL: thumbURL,
		FileSize: fh.Size,
		FileType: fileType,
	}
	if err := s.fileRepo.Create(ctx, f); err != nil {
		return nil, err
	}

	return &UploadResult{
		ID:       f.ID,
		FileURL:  f.FileURL,
		ThumbURL: f.ThumbURL,
		FileName: f.FileName,
		FileSize: f.FileSize,
		FileType: f.FileType,
		Duration: f.Duration,
	}, nil
}

func (s *FileService) BatchUpload(ctx context.Context, userID uint64, files []*multipart.FileHeader, bizType string) (*BatchUploadResult, error) {
	if len(files) > maxBatchCount {
		return nil, fmt.Errorf("UPLOAD_BATCH_LIMIT")
	}

	var results []UploadResult
	failCount := 0

	for _, fh := range files {
		result, err := s.Upload(ctx, userID, fh, bizType)
		if err != nil {
			log.L(ctx).Warn("batch upload single file failed", "file", fh.Filename, "err", err)
			failCount++
			continue
		}
		results = append(results, *result)
	}

	return &BatchUploadResult{
		Files:        results,
		SuccessCount: len(results),
		FailCount:    failCount,
	}, nil
}

func generateObjectKey(bizType, ext string) string {
	now := time.Now()
	return fmt.Sprintf("uploads/%s/%d/%02d/%s%s",
		bizType, now.Year(), now.Month(), uuid.NewString(), ext)
}
