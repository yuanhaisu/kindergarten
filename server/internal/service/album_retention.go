package service

import (
	"context"
	"time"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/pkg/log"
)

// AlbumSoftDeleteRetention 软删相册的保留期（design D8）：
// 满 180 天后才允许硬删，期间保留行供审计核查。
const AlbumSoftDeleteRetention = 180 * 24 * time.Hour

// albumRetentionDefaultBatch 单批硬删的相册数：分批跑，避免一条长事务锁住整张表。
const albumRetentionDefaultBatch = 100

// AlbumRetentionService 软删相册的到期清理。
//
// 它不挂在 HTTP 服务上（没有 handler，也不进 wire）：调度由运维定时执行
// cmd/albumretention（design D8，不新增常驻 worker），读取端另有
// deleted_at + 180d 兜底过滤，两边不能互相替代。
type AlbumRetentionService struct {
	db *gorm.DB
}

func NewAlbumRetentionService(db *gorm.DB) *AlbumRetentionService {
	return &AlbumRetentionService{db: db}
}

// AlbumRetentionStats 到期相册与关联行数（-dry-run 只统计，不删除）。
type AlbumRetentionStats struct {
	Albums        int64
	Photos        int64
	PhotoStudents int64
	AlbumStudents int64
}

// expiredAlbumsWhere 到期判定：只处理软删除且 deleted_at <= now-180d 的相册。
// 未删除的相册永远不在这里出现，因此清理不会碰在线内容。
const expiredAlbumsWhere = `deleted_at IS NOT NULL AND deleted_at <= ?`

// expiredAlbumIDsSQL 到期相册 ID 子查询，供没有 album_id 的关联表回溯。
const expiredAlbumIDsSQL = `SELECT id FROM albums WHERE ` + expiredAlbumsWhere

// CountExpiredAlbums 统计到期待清理的行数，不删除任何数据（命令的 -dry-run）。
func (s *AlbumRetentionService) CountExpiredAlbums(ctx context.Context, now time.Time) (AlbumRetentionStats, error) {
	cutoff := now.Add(-AlbumSoftDeleteRetention)
	var out AlbumRetentionStats
	// 关联行按「属于到期相册」统计：album_photo_students 没有 album_id，
	// 只能经照片回溯；照片与关联都含软删行，与硬删口径保持一致。
	if err := s.db.WithContext(ctx).Unscoped().Model(&domain.Album{}).
		Where(expiredAlbumsWhere, cutoff).
		Count(&out.Albums).Error; err != nil {
		return out, err
	}
	if err := s.db.WithContext(ctx).Unscoped().Model(&domain.AlbumPhoto{}).
		Where("album_id IN ("+expiredAlbumIDsSQL+")", cutoff).
		Count(&out.Photos).Error; err != nil {
		return out, err
	}
	if err := s.db.WithContext(ctx).Unscoped().Model(&domain.AlbumPhotoStudent{}).
		Where("photo_id IN (SELECT id FROM album_photos WHERE album_id IN ("+expiredAlbumIDsSQL+"))", cutoff).
		Count(&out.PhotoStudents).Error; err != nil {
		return out, err
	}
	if err := s.db.WithContext(ctx).Unscoped().Model(&domain.AlbumStudent{}).
		Where("album_id IN ("+expiredAlbumIDsSQL+")", cutoff).
		Count(&out.AlbumStudents).Error; err != nil {
		return out, err
	}
	return out, nil
}

// CleanupExpiredAlbums 分批硬删软删满 180 天的相册及其照片、相册级/照片级
// 幼儿关联，返回删掉的相册数。每批一个事务；任一批失败立即返回错误，
// 不静默跳过（清理漏行比报错更难发现）。
//
// operation_logs 不在清理范围内：审计是事后追责的唯一依据（design D8），
// 相册行删掉后仍要能查到谁在什么时候发布/下架/删除过它。
func (s *AlbumRetentionService) CleanupExpiredAlbums(ctx context.Context, now time.Time, batchSize int) (int, error) {
	if batchSize <= 0 {
		batchSize = albumRetentionDefaultBatch
	}
	cutoff := now.Add(-AlbumSoftDeleteRetention)
	deleted := 0
	for {
		var ids []uint64
		if err := s.db.WithContext(ctx).Unscoped().Model(&domain.Album{}).
			Where(expiredAlbumsWhere, cutoff).
			Order("id ASC").Limit(batchSize).Pluck("id", &ids).Error; err != nil {
			return deleted, err
		}
		if len(ids) == 0 {
			return deleted, nil
		}
		n, err := s.deleteAlbumBatch(ctx, ids, cutoff)
		if err != nil {
			return deleted, err
		}
		deleted += n
		if n == 0 {
			// 选批后这批行已被并发清理或恢复：不再重试，避免空转。
			// 仍然到期的行留给下一次调度。
			return deleted, nil
		}
	}
}

// deleteAlbumBatch 同一事务内按「照片级关联 → 照片 → 相册级关联 → 相册」
// 的顺序硬删。每条删除都重复到期条件：批内相册若在选批后被并发恢复，
// 它的照片与关联不会被连带硬删。
func (s *AlbumRetentionService) deleteAlbumBatch(ctx context.Context, albumIDs []uint64, cutoff time.Time) (int, error) {
	deleted := 0
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		expiredAlbums := `SELECT id FROM albums WHERE id IN ? AND ` + expiredAlbumsWhere
		if err := tx.Unscoped().
			Where("photo_id IN (SELECT id FROM album_photos WHERE album_id IN ("+expiredAlbums+"))", albumIDs, cutoff).
			Delete(&domain.AlbumPhotoStudent{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().
			Where("album_id IN ("+expiredAlbums+")", albumIDs, cutoff).
			Delete(&domain.AlbumPhoto{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().
			Where("album_id IN ("+expiredAlbums+")", albumIDs, cutoff).
			Delete(&domain.AlbumStudent{}).Error; err != nil {
			return err
		}
		res := tx.Unscoped().
			Where("id IN ? AND "+expiredAlbumsWhere, albumIDs, cutoff).
			Delete(&domain.Album{})
		if res.Error != nil {
			return res.Error
		}
		deleted = int(res.RowsAffected)
		return nil
	})
	return deleted, err
}

// ==================== 指标事件（design D10） ====================

// Phase 1 只写结构化 slog，不建分析表；后续接分析平台时从日志聚合。
// 字段只含 ID 与计数：手机号、openid 一律不记录。
const (
	AlbumEventPublish    = "album_publish"
	AlbumEventParentView = "album_parent_view"
	AlbumEventOptOut     = "album_opt_out"

	albumEventRoleTeacher = "teacher"
	albumEventRoleParent  = "parent"
)

// AlbumPublishEvent 老师发布相册。
type AlbumPublishEvent struct {
	KindergartenID uint64
	ClassID        uint64
	AlbumID        uint64
	UserID         uint64
	PhotoCount     int
}

// AlbumParentViewEvent 家长打开相册详情。StudentID 是查看者当前孩子，
// 不是被拍到的其他幼儿；不记录手机号。
type AlbumParentViewEvent struct {
	KindergartenID    uint64
	ClassID           uint64
	AlbumID           uint64
	StudentID         uint64
	VisiblePhotoCount int
}

// AlbumOptOutEvent 家长修改班内共享开关。
type AlbumOptOutEvent struct {
	KindergartenID  uint64
	UserID          uint64
	AllowClassShare bool
}

// LogAlbumPublish 发布成功后的埋点。
func LogAlbumPublish(ctx context.Context, e AlbumPublishEvent) {
	log.L(ctx).Info(AlbumEventPublish,
		"kindergarten_id", e.KindergartenID,
		"class_id", e.ClassID,
		"album_id", e.AlbumID,
		"user_id", e.UserID,
		"photo_count", e.PhotoCount,
		"role", albumEventRoleTeacher,
	)
}

// LogAlbumParentView 家长相册详情返回非错误结果后的埋点。
func LogAlbumParentView(ctx context.Context, e AlbumParentViewEvent) {
	log.L(ctx).Info(AlbumEventParentView,
		"kindergarten_id", e.KindergartenID,
		"class_id", e.ClassID,
		"album_id", e.AlbumID,
		"student_id", e.StudentID,
		"visible_photo_count", e.VisiblePhotoCount,
		"role", albumEventRoleParent,
	)
}

// LogAlbumOptOut 设置里班内共享开关真实变化时的埋点。
func LogAlbumOptOut(ctx context.Context, e AlbumOptOutEvent) {
	log.L(ctx).Info(AlbumEventOptOut,
		"kindergarten_id", e.KindergartenID,
		"user_id", e.UserID,
		"allow_class_share", e.AllowClassShare,
		"role", albumEventRoleParent,
	)
}
