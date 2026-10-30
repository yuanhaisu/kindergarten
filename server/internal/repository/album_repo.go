package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

// AlbumRepo 班级相册的数据访问。所有方法都按 kindergarten_id + class_id 收敛，
// 调用方（teacher service）不允许传入其它班级的上下文。
type AlbumRepo struct {
	db *gorm.DB
}

func NewAlbumRepo(db *gorm.DB) *AlbumRepo {
	return &AlbumRepo{db: db}
}

// AlbumStats 统计卡三项。monthly_new 按 album_date 所在自然月，不是创建时间。
type AlbumStats struct {
	TotalPhotos int64 `gorm:"column:total_photos"`
	MonthlyNew  int64 `gorm:"column:monthly_new"`
	AlbumCount  int64 `gorm:"column:album_count"`
}

// Stats 一次聚合出三项统计，避免三趟查询读到不同快照。
// 照片数取相册冗余列 photo_count（本期无编辑/删除，冗余值不会漂移）。
func (r *AlbumRepo) Stats(
	ctx context.Context,
	kindergartenID, classID uint64,
	monthStart, monthEnd time.Time,
) (AlbumStats, error) {
	var out AlbumStats
	err := r.db.WithContext(ctx).Model(&domain.Album{}).
		Select(`COALESCE(SUM(photo_count), 0) AS total_photos,
			COALESCE(SUM(CASE WHEN album_date >= ? AND album_date < ? THEN photo_count ELSE 0 END), 0) AS monthly_new,
			COUNT(*) AS album_count`, monthStart, monthEnd).
		Where("kindergarten_id = ? AND class_id = ?", kindergartenID, classID).
		Scan(&out).Error
	return out, err
}

// List 本班相册分页，按相册日期倒序；同日按 id 倒序保证顺序稳定。
// category 为空表示不筛选（「全部」）。
func (r *AlbumRepo) List(
	ctx context.Context,
	kindergartenID, classID uint64,
	category domain.AlbumCategory,
	offset, limit int,
) ([]domain.Album, int64, error) {
	base := r.db.WithContext(ctx).Model(&domain.Album{}).
		Where("kindergarten_id = ? AND class_id = ?", kindergartenID, classID)
	if category != "" {
		base = base.Where("category = ?", category)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []domain.Album
	err := base.Session(&gorm.Session{}).
		Order("album_date DESC, id DESC").
		Offset(offset).Limit(limit).
		Find(&list).Error
	return list, total, err
}

// FindByID 只在本班范围内查找；跨班/跨园与不存在同样返回 (nil, nil)，
// 由 service 统一映射为 404，不泄露其它班级数据是否存在。
func (r *AlbumRepo) FindByID(ctx context.Context, kindergartenID, classID, id uint64) (*domain.Album, error) {
	var a domain.Album
	err := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND class_id = ? AND id = ?", kindergartenID, classID, id).
		First(&a).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

// ListPhotos 相册照片按创建时的提交顺序返回；id 作为同 sort_num 的兜底，顺序稳定。
func (r *AlbumRepo) ListPhotos(ctx context.Context, albumID uint64) ([]domain.AlbumPhoto, error) {
	var list []domain.AlbumPhoto
	err := r.db.WithContext(ctx).
		Where("album_id = ?", albumID).
		Order("sort_num ASC, id ASC").
		Find(&list).Error
	return list, err
}

// ListFirstPhotos 列表页批量取每个相册的前 perAlbum 张照片（sort_num 升序），
// 一次查询覆盖整页相册，避免逐相册查询的 N+1。
// 截断放在 Go 侧：MySQL 5.7 没有窗口函数，而每相册的 LIMIT 无法在单条 SQL 里表达。
func (r *AlbumRepo) ListFirstPhotos(
	ctx context.Context,
	albumIDs []uint64,
	perAlbum int,
) (map[uint64][]domain.AlbumPhoto, error) {
	out := make(map[uint64][]domain.AlbumPhoto, len(albumIDs))
	if len(albumIDs) == 0 || perAlbum <= 0 {
		return out, nil
	}

	var photos []domain.AlbumPhoto
	err := r.db.WithContext(ctx).
		Where("album_id IN ?", albumIDs).
		Order("album_id ASC, sort_num ASC, id ASC").
		Find(&photos).Error
	if err != nil {
		return nil, err
	}
	for _, p := range photos {
		if len(out[p.AlbumID]) >= perAlbum {
			continue
		}
		out[p.AlbumID] = append(out[p.AlbumID], p)
	}
	return out, nil
}

// Create 事务写入相册与照片：任一照片插入失败则整单回滚，不留半条相册。
// sort_num 一律按切片下标重写，调用方传入的 SortNum 视为无效输入。
func (r *AlbumRepo) Create(ctx context.Context, album *domain.Album, photos []domain.AlbumPhoto) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(album).Error; err != nil {
			return err
		}
		if len(photos) == 0 {
			return nil
		}
		rows := make([]domain.AlbumPhoto, len(photos))
		for i := range photos {
			rows[i] = photos[i]
			rows[i].AlbumID = album.ID
			rows[i].SortNum = i
		}
		return tx.Create(&rows).Error
	})
}
