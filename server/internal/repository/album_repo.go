package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

// AlbumRepo 班级相册的数据访问。所有方法都按 kindergarten_id + class_id 收敛，
// 调用方（teacher service）不允许传入其它班级的上下文。
type AlbumRepo struct {
	db *gorm.DB
}

// ErrAlbumNotFound 相册在事务内已不存在或已软删（读到之后被并发删除）。
// service 映射为 404，绝不能在孤儿相册上继续写照片与关联。
var ErrAlbumNotFound = errors.New("album not found")

func NewAlbumRepo(db *gorm.DB) *AlbumRepo {
	return &AlbumRepo{db: db}
}

// AlbumStats 统计卡三项。monthly_new 按 album_date 所在自然月，不是创建时间。
type AlbumStats struct {
	TotalPhotos int64 `gorm:"column:total_photos"`
	MonthlyNew  int64 `gorm:"column:monthly_new"`
	AlbumCount  int64 `gorm:"column:album_count"`
}

// AlbumStudentRow 相册级/照片级关联幼儿的展示字段。
type AlbumStudentRow struct {
	StudentID uint64 `gorm:"column:student_id"`
	Name      string `gorm:"column:name"`
	Avatar    string `gorm:"column:avatar"`
}

// AlbumPhotoStudentRow 照片级关联幼儿，带照片 ID 供批量分组。
type AlbumPhotoStudentRow struct {
	PhotoID   uint64 `gorm:"column:photo_id"`
	StudentID uint64 `gorm:"column:student_id"`
	Name      string `gorm:"column:name"`
	Avatar    string `gorm:"column:avatar"`
}

// Stats 一次聚合出三项统计，避免三趟查询读到不同快照。
// 只统计已发布相册：草稿/已下架是老师的编辑中状态，不是对家长可见的内容；
// 软删由 GORM 的 deleted_at 条件自动排除。
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
		Where("kindergarten_id = ? AND class_id = ? AND status = ?",
			kindergartenID, classID, domain.AlbumStatusPublished).
		Scan(&out).Error
	return out, err
}

// List 本班相册分页，按相册日期倒序；同日按 id 倒序保证顺序稳定。
// category 为空表示不筛选（「全部」）；status 为空表示返回全部状态（老师端默认）。
func (r *AlbumRepo) List(
	ctx context.Context,
	kindergartenID, classID uint64,
	category domain.AlbumCategory,
	status string,
	offset, limit int,
) ([]domain.Album, int64, error) {
	base := r.db.WithContext(ctx).Model(&domain.Album{}).
		Where("kindergarten_id = ? AND class_id = ?", kindergartenID, classID)
	if category != "" {
		base = base.Where("category = ?", category)
	}
	if status != "" {
		base = base.Where("status = ?", status)
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
	return r.findByID(ctx, kindergartenID, classID, id)
}

// FindByIDAnyStatus 老师读详情用：明确表示不按发布状态过滤，
// 草稿与已下架相册在老师端仍要能看到（家长端另有只查 published 的入口）。
func (r *AlbumRepo) FindByIDAnyStatus(ctx context.Context, kindergartenID, classID, id uint64) (*domain.Album, error) {
	return r.findByID(ctx, kindergartenID, classID, id)
}

func (r *AlbumRepo) findByID(ctx context.Context, kindergartenID, classID, id uint64) (*domain.Album, error) {
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

// ListPhotosWithStudents 一次取齐相册照片与照片级幼儿关联：
// 两条查询（照片、关联+幼儿），不做「每张照片再查一次」的 N+1。
// 用 LEFT JOIN 而非 INNER JOIN：学生被软删后关联行仍要参与隐私判断，
// 只是姓名/头像为空。
func (r *AlbumRepo) ListPhotosWithStudents(
	ctx context.Context,
	albumID uint64,
) ([]domain.AlbumPhoto, map[uint64][]AlbumStudentRow, error) {
	photos, err := r.ListPhotos(ctx, albumID)
	if err != nil {
		return nil, nil, err
	}
	out := make(map[uint64][]AlbumStudentRow, len(photos))
	if len(photos) == 0 {
		return photos, out, nil
	}
	photoIDs := make([]uint64, 0, len(photos))
	for _, p := range photos {
		photoIDs = append(photoIDs, p.ID)
	}
	var rows []AlbumPhotoStudentRow
	if err := r.db.WithContext(ctx).Table("album_photo_students AS aps").
		Select("aps.photo_id, aps.student_id, s.name, s.avatar").
		Joins("LEFT JOIN students s ON s.id = aps.student_id AND s.deleted_at IS NULL").
		Where("aps.photo_id IN ? AND aps.deleted_at IS NULL", photoIDs).
		Order("aps.id ASC").
		Scan(&rows).Error; err != nil {
		return nil, nil, err
	}
	for _, row := range rows {
		out[row.PhotoID] = append(out[row.PhotoID], AlbumStudentRow{
			StudentID: row.StudentID, Name: row.Name, Avatar: row.Avatar,
		})
	}
	return photos, out, nil
}

// ListAlbumStudents 相册级参与幼儿（批量参与者的兜底口径）。
func (r *AlbumRepo) ListAlbumStudents(ctx context.Context, albumID uint64) ([]AlbumStudentRow, error) {
	var rows []AlbumStudentRow
	err := r.db.WithContext(ctx).Table("album_students AS ast").
		Select("ast.student_id, s.name, s.avatar").
		Joins("LEFT JOIN students s ON s.id = ast.student_id AND s.deleted_at IS NULL").
		Where("ast.album_id = ? AND ast.deleted_at IS NULL", albumID).
		Order("ast.id ASC").
		Scan(&rows).Error
	return rows, err
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

// Create 事务写入相册、照片、相册级参与者与照片级幼儿：
// 任一环节失败整单回滚，不留半条相册或孤立关联。
// sort_num 一律按切片下标重写，调用方传入的 SortNum 视为无效输入。
func (r *AlbumRepo) Create(
	ctx context.Context,
	album *domain.Album,
	photos []domain.AlbumPhoto,
	participants []uint64,
	photoStudents [][]uint64,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(album).Error; err != nil {
			return err
		}
		photoIDs, err := insertAlbumPhotos(tx, album.ID, photos)
		if err != nil {
			return err
		}
		if err := replaceAlbumStudents(tx, album.ID, album.KindergartenID, participants); err != nil {
			return err
		}
		return replaceAlbumPhotoStudents(tx, album.KindergartenID, photoIDs, photoStudents)
	})
}

// Update 编辑相册：同一事务内更新可编辑字段、同步照片（提交了 id 的保留原 ID，
// 未提交的硬删）、重算 cover_url/photo_count，并整体替换两级幼儿关联。
// 保留照片 ID 是为了让端上正在展示的图片 ID 在编辑后仍然有效；
// 未提交的照片从相册移除属于内容编辑而非审计删除，因此硬删，不占软删行。
func (r *AlbumRepo) Update(
	ctx context.Context,
	album *domain.Album,
	photos []domain.AlbumPhoto,
	participants []uint64,
	photoStudents [][]uint64,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&domain.Album{}).
			Where("id = ? AND kindergarten_id = ? AND class_id = ?", album.ID, album.KindergartenID, album.ClassID).
			Updates(map[string]any{
				"title":       album.Title,
				"category":    album.Category,
				"album_date":  album.AlbumDate,
				"cover_url":   album.CoverURL,
				"photo_count": album.PhotoCount,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			// 读到时还存在、进事务前被并发软删/移走：不能继续把照片与关联
			// 写到孤儿相册上。整单回滚，由 service 映射为 404。
			return ErrAlbumNotFound
		}

		var existing []domain.AlbumPhoto
		if err := tx.Where("album_id = ?", album.ID).Find(&existing).Error; err != nil {
			return err
		}
		existingSet := make(map[uint64]bool, len(existing))
		for _, p := range existing {
			existingSet[p.ID] = true
		}

		photoIDs := make([]uint64, len(photos))
		submitted := make(map[uint64]bool, len(photos))
		for i := range photos {
			if photos[i].ID != 0 {
				if !existingSet[photos[i].ID] || submitted[photos[i].ID] {
					return fmt.Errorf("album photo %d 不属于相册 %d 或重复提交", photos[i].ID, album.ID)
				}
				if err := tx.Model(&domain.AlbumPhoto{}).
					Where("id = ? AND album_id = ?", photos[i].ID, album.ID).
					Updates(map[string]any{
						"file_id":   photos[i].FileID,
						"file_url":  photos[i].FileURL,
						"thumb_url": photos[i].ThumbURL,
						"width":     photos[i].Width,
						"height":    photos[i].Height,
						"sort_num":  i,
					}).Error; err != nil {
					return err
				}
				photoIDs[i] = photos[i].ID
			} else {
				row := photos[i]
				row.AlbumID = album.ID
				row.SortNum = i
				if err := tx.Create(&row).Error; err != nil {
					return err
				}
				photoIDs[i] = row.ID
			}
			submitted[photoIDs[i]] = true
		}

		removed := make([]uint64, 0, len(existing))
		for _, p := range existing {
			if !submitted[p.ID] {
				removed = append(removed, p.ID)
			}
		}
		if len(removed) > 0 {
			if err := tx.Unscoped().Where("photo_id IN ?", removed).Delete(&domain.AlbumPhotoStudent{}).Error; err != nil {
				return err
			}
			if err := tx.Unscoped().Where("id IN ?", removed).Delete(&domain.AlbumPhoto{}).Error; err != nil {
				return err
			}
		}

		if err := replaceAlbumStudents(tx, album.ID, album.KindergartenID, participants); err != nil {
			return err
		}
		return replaceAlbumPhotoStudents(tx, album.KindergartenID, photoIDs, photoStudents)
	})
}

// UpdateStatus 原子状态迁移：只有当前状态在 from 中才更新，返回是否命中。
// 状态机检查在 service 侧做（区分 404 与 422），这里用 status IN 兜住并发二次点击。
func (r *AlbumRepo) UpdateStatus(
	ctx context.Context,
	kindergartenID, classID, id uint64,
	from []string,
	to string,
	publishedAt *time.Time,
	publishedBy uint64,
) (bool, error) {
	res := r.db.WithContext(ctx).Model(&domain.Album{}).
		Where("id = ? AND kindergarten_id = ? AND class_id = ? AND status IN ?", id, kindergartenID, classID, from).
		Updates(map[string]any{
			"status":       to,
			"published_at": publishedAt,
			"published_by": publishedBy,
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// SoftDelete 软删相册。返回 false 表示已不存在/已删除/跨班跨园，由 service 映射 404。
func (r *AlbumRepo) SoftDelete(ctx context.Context, kindergartenID, classID, id uint64) (bool, error) {
	res := r.db.WithContext(ctx).
		Where("id = ? AND kindergarten_id = ? AND class_id = ?", id, kindergartenID, classID).
		Delete(&domain.Album{})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// FindFilesUnderKindergartenTeachers 校验 file_id 的园所归属：
// files 表本身没有 kindergarten_id，只能经上传者的老师角色判断「属于本园」。
// 只认识本园 teacher 角色的上传者；管理员/家长上传的文件不能挂进班级相册。
func (r *AlbumRepo) FindFilesUnderKindergartenTeachers(
	ctx context.Context,
	kindergartenID uint64,
	ids []uint64,
) (map[uint64]domain.File, error) {
	out := make(map[uint64]domain.File, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var rows []domain.File
	err := r.db.WithContext(ctx).Table("files AS f").
		Select("f.*").
		Where("f.id IN ? AND f.deleted_at IS NULL", ids).
		Where(`EXISTS (SELECT 1 FROM user_roles ur
			WHERE ur.user_id = f.user_id AND ur.role = 'teacher'
			  AND ur.kindergarten_id = ? AND ur.deleted_at IS NULL)`, kindergartenID).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, f := range rows {
		out[f.ID] = f
	}
	return out, nil
}

// FindFilesByURLUnderKindergartenTeachers 从成长记录选图时没有 file_id 可用
// （growth_media 只存 URL 快照），按 file_url 反查文件行——与迁移 SQL 的回填
// 口径一致：同一 URL 命中多行视为歧义，宁可不填 file_id，也不猜。
func (r *AlbumRepo) FindFilesByURLUnderKindergartenTeachers(
	ctx context.Context,
	kindergartenID uint64,
	urls []string,
) (map[string]domain.File, error) {
	out := make(map[string]domain.File, len(urls))
	if len(urls) == 0 {
		return out, nil
	}
	var rows []domain.File
	err := r.db.WithContext(ctx).Table("files AS f").
		Select("f.*").
		Where("f.file_url IN ? AND f.deleted_at IS NULL", urls).
		Where(`EXISTS (SELECT 1 FROM user_roles ur
			WHERE ur.user_id = f.user_id AND ur.role = 'teacher'
			  AND ur.kindergarten_id = ? AND ur.deleted_at IS NULL)`, kindergartenID).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	grouped := make(map[string][]domain.File, len(rows))
	for _, f := range rows {
		grouped[f.FileURL] = append(grouped[f.FileURL], f)
	}
	for url, files := range grouped {
		if len(files) == 1 {
			out[url] = files[0]
		}
	}
	return out, nil
}

// FindStudentsInClass 显式参与幼儿必须属于本园本班；返回命中集合供 service 判缺。
func (r *AlbumRepo) FindStudentsInClass(
	ctx context.Context,
	kindergartenID, classID uint64,
	ids []uint64,
) (map[uint64]domain.Student, error) {
	out := make(map[uint64]domain.Student, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var rows []domain.Student
	err := r.db.WithContext(ctx).
		Where("id IN ? AND kindergarten_id = ? AND class_id = ?", ids, kindergartenID, classID).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, st := range rows {
		out[st.ID] = st
	}
	return out, nil
}

// FindGrowthMediaInClass 从成长记录选图时取来源媒体：必须同园同班且未软删。
func (r *AlbumRepo) FindGrowthMediaInClass(
	ctx context.Context,
	kindergartenID, classID, mediaID uint64,
) (*domain.GrowthMedia, error) {
	var m domain.GrowthMedia
	err := r.db.WithContext(ctx).
		Where("id = ? AND kindergarten_id = ? AND class_id = ?", mediaID, kindergartenID, classID).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

// ListGrowthLogStudentsByLogIDs 批量取多条成长记录关联的幼儿（照片级关联的自动来源），
// 一次查询按 log_id 分组，避免逐张照片查一次。
func (r *AlbumRepo) ListGrowthLogStudentsByLogIDs(
	ctx context.Context,
	logIDs []uint64,
) (map[uint64][]uint64, error) {
	out := make(map[uint64][]uint64, len(logIDs))
	if len(logIDs) == 0 {
		return out, nil
	}
	type row struct {
		LogID     uint64 `gorm:"column:log_id"`
		StudentID uint64 `gorm:"column:student_id"`
	}
	var rows []row
	err := r.db.WithContext(ctx).Table("growth_log_students").
		Select("log_id, student_id").
		Where("log_id IN ? AND deleted_at IS NULL", logIDs).
		Order("id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.LogID] = append(out[row.LogID], row.StudentID)
	}
	return out, nil
}

// ListOptOutStudentIDs 这些幼儿中至少一位监护人明确关闭了班内共享的集合。
// 没有 parent_settings 行按默认开启处理（design D6），因此只查显式 false 的行。
func (r *AlbumRepo) ListOptOutStudentIDs(
	ctx context.Context,
	kindergartenID uint64,
	studentIDs []uint64,
) (map[uint64]bool, error) {
	out := make(map[uint64]bool, len(studentIDs))
	if len(studentIDs) == 0 {
		return out, nil
	}
	var ids []uint64
	err := r.db.WithContext(ctx).Table("student_parents AS sp").
		Select("DISTINCT sp.student_id").
		Joins(`INNER JOIN parent_settings ps ON ps.user_id = sp.user_id
			AND ps.kindergarten_id = ? AND ps.deleted_at IS NULL AND ps.allow_class_share = 0`, kindergartenID).
		Where("sp.student_id IN ? AND sp.deleted_at IS NULL", studentIDs).
		Pluck("sp.student_id", &ids).Error
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		out[id] = true
	}
	return out, nil
}

func insertAlbumPhotos(tx *gorm.DB, albumID uint64, photos []domain.AlbumPhoto) ([]uint64, error) {
	ids := make([]uint64, 0, len(photos))
	if len(photos) == 0 {
		return ids, nil
	}
	rows := make([]domain.AlbumPhoto, len(photos))
	for i := range photos {
		rows[i] = photos[i]
		rows[i].AlbumID = albumID
		rows[i].SortNum = i
	}
	if err := tx.Create(&rows).Error; err != nil {
		return nil, err
	}
	for i := range rows {
		ids = append(ids, rows[i].ID)
	}
	return ids, nil
}

func replaceAlbumStudents(tx *gorm.DB, albumID, kindergartenID uint64, studentIDs []uint64) error {
	if err := tx.Unscoped().Where("album_id = ?", albumID).Delete(&domain.AlbumStudent{}).Error; err != nil {
		return err
	}
	ids := dedupeIDs(studentIDs)
	if len(ids) == 0 {
		return nil
	}
	rows := make([]domain.AlbumStudent, 0, len(ids))
	for _, sid := range ids {
		rows = append(rows, domain.AlbumStudent{
			KindergartenID: kindergartenID, AlbumID: albumID, StudentID: sid,
		})
	}
	return tx.Create(&rows).Error
}

func replaceAlbumPhotoStudents(tx *gorm.DB, kindergartenID uint64, photoIDs []uint64, photoStudents [][]uint64) error {
	if len(photoIDs) == 0 {
		return nil
	}
	// 硬删后重插：唯一键 (photo_id, student_id) 下重复提交必须幂等，
	// 而软删行仍会占住唯一索引，所以这里不能走软删。
	if err := tx.Unscoped().Where("photo_id IN ?", photoIDs).Delete(&domain.AlbumPhotoStudent{}).Error; err != nil {
		return err
	}
	rows := make([]domain.AlbumPhotoStudent, 0)
	for i, pid := range photoIDs {
		if i >= len(photoStudents) {
			break
		}
		for _, sid := range dedupeIDs(photoStudents[i]) {
			rows = append(rows, domain.AlbumPhotoStudent{
				KindergartenID: kindergartenID, PhotoID: pid, StudentID: sid,
			})
		}
	}
	if len(rows) == 0 {
		return nil
	}
	return tx.Create(&rows).Error
}

func dedupeIDs(ids []uint64) []uint64 {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[uint64]bool, len(ids))
	out := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if id == 0 || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}
