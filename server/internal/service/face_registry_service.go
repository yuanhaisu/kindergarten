package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"kindergarten-server/internal/ai/face"
	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
	"kindergarten-server/pkg/log"
)

// maxFacesPerStudent 学生底库照片上限（spec student-face-registry：1-5 张）。
const maxFacesPerStudent = 5

// ErrFaceNotEnabled 人脸能力未开启（[face].enabled=false）——注册类操作直接拒绝，
// 合规前置（家长授权，独立变更）完成前不产生真实底库数据（design Risks）。
var ErrFaceNotEnabled = errors.New("FACE_NOT_ENABLED")

// FaceRegistryService 学生人脸底库管理（admin 端）。
// 密钥边界：云端调用只发生在这里；客户端仅经统一上传通道提交照片（file_id）。
type FaceRegistryService struct {
	db       *gorm.DB
	provider face.Provider
	cfg      config.FaceConfig
	// fetch 取图（默认 HTTP GET OSS 直链；测试注入假实现，与识别队列同一形态）
	fetch func(ctx context.Context, url string) ([]byte, error)
}

func NewFaceRegistryService(db *gorm.DB, provider face.Provider, cfg config.FaceConfig) *FaceRegistryService {
	return &FaceRegistryService{db: db, provider: provider, cfg: cfg, fetch: httpFetch}
}

// aggregateFaceStatus 由底库行派生聚合状态：0未录入 1已注册 2注册失败 3注册中。
// 有失败即视为「注册失败」（最需要管理员处理的状态）；全部已注册才是「已注册」。
func aggregateFaceStatus(total, indexed, failed int64) int8 {
	switch {
	case total == 0:
		return 0
	case failed > 0:
		return 2
	case indexed == total:
		return 1
	default:
		return 3
	}
}

func faceStatusLabel(st int8) string {
	switch st {
	case 0:
		return "未录入"
	case 1:
		return "已注册"
	case 2:
		return "注册失败"
	case 3:
		return "注册中"
	default:
		return "未知"
	}
}

// checkStudent 归属校验：学生必须属于该园所（跨园访问一律 NOT_FOUND）。
func (s *FaceRegistryService) checkStudent(ctx context.Context, kgID, studentID uint64) error {
	var n int64
	if err := s.db.WithContext(ctx).Model(&domain.Student{}).
		Where("id = ? AND kindergarten_id = ?", studentID, kgID).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return errors.New("NOT_FOUND")
	}
	return nil
}

func faceItem(f domain.StudentFace) map[string]any {
	return map[string]any{
		"id": f.ID, "face_url": f.FaceURL, "status": f.Status,
		"status_label": faceStatusLabel(f.Status), "last_error": f.LastError,
		"created_at": f.CreatedAt.UTC().Format(time.RFC3339),
	}
}

// ListStudentFaces 学生的底库照片与聚合状态。
func (s *FaceRegistryService) ListStudentFaces(ctx context.Context, kgID, studentID uint64) (map[string]any, error) {
	if err := s.checkStudent(ctx, kgID, studentID); err != nil {
		return nil, err
	}
	var rows []domain.StudentFace
	if err := s.db.WithContext(ctx).Where("student_id = ?", studentID).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	list := make([]map[string]any, 0, len(rows))
	var indexed, failed int64
	for _, f := range rows {
		list = append(list, faceItem(f))
		switch f.Status {
		case domain.FaceStatusIndexed:
			indexed++
		case domain.FaceStatusFailed:
			failed++
		}
	}
	agg := aggregateFaceStatus(int64(len(rows)), indexed, failed)
	return map[string]any{
		"list": list, "total": len(rows),
		"status": agg, "status_label": faceStatusLabel(agg),
		"enabled": s.cfg.Enabled, "provider": s.provider.Name(),
		"max_count": maxFacesPerStudent,
	}, nil
}

// AddStudentFaces 追加底库照片（file_id 来自统一上传通道），立即注册。
// 注册失败不使请求失败——失败原因落在行状态上，供管理端重试（spec FRG-0004）。
func (s *FaceRegistryService) AddStudentFaces(ctx context.Context, kgID, studentID uint64, fileIDs []uint64) (map[string]any, error) {
	if !s.cfg.Enabled {
		return nil, ErrFaceNotEnabled
	}
	if err := s.checkStudent(ctx, kgID, studentID); err != nil {
		return nil, err
	}
	ids := make([]uint64, 0, len(fileIDs))
	seen := make(map[uint64]bool, len(fileIDs))
	for _, fid := range fileIDs {
		if fid == 0 || seen[fid] {
			continue
		}
		seen[fid] = true
		ids = append(ids, fid)
	}
	if len(ids) == 0 {
		return nil, errors.New("INVALID_FILE")
	}
	var existing int64
	if err := s.db.WithContext(ctx).Model(&domain.StudentFace{}).Where("student_id = ?", studentID).Count(&existing).Error; err != nil {
		return nil, err
	}
	if int(existing)+len(ids) > maxFacesPerStudent {
		return nil, fmt.Errorf("FACE_LIMIT_EXCEEDED")
	}
	var files []domain.File
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&files).Error; err != nil {
		return nil, err
	}
	byID := make(map[uint64]domain.File, len(files))
	for _, f := range files {
		byID[f.ID] = f
	}
	// 按提交顺序建行（材料必须在场且为图片）
	for _, fid := range ids {
		f, ok := byID[fid]
		if !ok || f.FileType != "image" {
			return nil, errors.New("INVALID_FILE")
		}
		row := &domain.StudentFace{StudentID: studentID, FaceURL: f.FileURL, Provider: s.provider.Name(), Status: domain.FaceStatusPending}
		if err := s.db.WithContext(ctx).Create(row).Error; err != nil {
			return nil, err
		}
	}
	s.syncStudentFaces(ctx, kgID, studentID, "register")
	return s.ListStudentFaces(ctx, kgID, studentID)
}

// RetryStudentFaces 重试该学生的底库注册（底库变更后重建云端状态）。
func (s *FaceRegistryService) RetryStudentFaces(ctx context.Context, kgID, studentID uint64) (map[string]any, error) {
	if !s.cfg.Enabled {
		return nil, ErrFaceNotEnabled
	}
	if err := s.checkStudent(ctx, kgID, studentID); err != nil {
		return nil, err
	}
	s.syncStudentFaces(ctx, kgID, studentID, "retry")
	return s.ListStudentFaces(ctx, kgID, studentID)
}

// DeleteStudentFace 移除一张底库照片并重建云端状态。
// 只触及底库（student_faces）——已发布的成长记录与照片点名不受影响（spec FRG-0006）。
func (s *FaceRegistryService) DeleteStudentFace(ctx context.Context, kgID, studentID, faceID uint64) error {
	if err := s.checkStudent(ctx, kgID, studentID); err != nil {
		return err
	}
	res := s.db.WithContext(ctx).Where("id = ? AND student_id = ?", faceID, studentID).Delete(&domain.StudentFace{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("NOT_FOUND")
	}
	s.syncStudentFaces(ctx, kgID, studentID, "delete")
	return nil
}

// syncStudentFaces 底库 → 云端重建：先清空云端条目，再按当前照片逐张注册。
//
// 为什么「清空重建」而不是增量：底库照片被移除时必须同步从云端消失（否则被移除的
// 照片仍可被识别命中）；逐张注册让「一张照片不合格」只影响它自己（FRG-0004 的重试
// 语义落在行上），不拖垮整批。注册动作是管理端低频操作，调用次数的代价可接受。
//
// 注册结果只落到行状态，不返回错误——调用方接口照常 200，失败原因可见可重试。
func (s *FaceRegistryService) syncStudentFaces(ctx context.Context, kgID, studentID uint64, trigger string) {
	var rows []domain.StudentFace
	if err := s.db.WithContext(ctx).Where("student_id = ?", studentID).Order("id ASC").Find(&rows).Error; err != nil {
		log.L(ctx).Error("face registry: load faces failed", "student", studentID, "err", err)
		return
	}
	// 清空云端（幂等：不存在视为成功）
	if err := s.provider.RemovePerson(ctx, kgID, studentID); err != nil {
		msg := truncateFaceErr(err)
		s.markFaces(ctx, rows, domain.FaceStatusFailed, msg, "")
		log.L(ctx).Warn("face registry: clear person failed", "student", studentID, "trigger", trigger, "err", msg)
		return
	}
	personRef := ""
	for _, f := range rows {
		img, err := s.fetch(ctx, f.FaceURL)
		if err != nil {
			s.markFace(ctx, f.ID, domain.FaceStatusFailed, truncateFaceErr(err), "")
			continue
		}
		ref, err := s.provider.EnsurePerson(ctx, kgID, studentID, [][]byte{img})
		if err != nil {
			s.markFace(ctx, f.ID, domain.FaceStatusFailed, truncateFaceErr(err), "")
			continue
		}
		personRef = ref
		s.markFace(ctx, f.ID, domain.FaceStatusIndexed, "", ref)
	}
	log.L(ctx).Info("face registry: sync done", "student", studentID, "trigger", trigger, "faces", len(rows), "person_ref", personRef)
}

func (s *FaceRegistryService) markFace(ctx context.Context, faceID uint64, status int8, lastErr, personRef string) {
	updates := map[string]any{"status": status, "last_error": lastErr}
	if personRef != "" {
		updates["person_ref"] = personRef
	}
	if err := s.db.WithContext(ctx).Model(&domain.StudentFace{}).Where("id = ?", faceID).Updates(updates).Error; err != nil {
		log.L(ctx).Error("face registry: mark face failed", "face", faceID, "err", err)
	}
}

func (s *FaceRegistryService) markFaces(ctx context.Context, rows []domain.StudentFace, status int8, lastErr, personRef string) {
	for _, f := range rows {
		s.markFace(ctx, f.ID, status, lastErr, personRef)
	}
}

// truncateFaceErr 错误信息截断（列宽 255），保留可诊断前缀。
func truncateFaceErr(err error) string {
	msg := err.Error()
	if len(msg) > 250 {
		msg = msg[:250]
	}
	return msg
}
