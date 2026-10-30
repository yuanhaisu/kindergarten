package service

import (
	"context"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

const (
	teacherAlbumTitleMaxRunes = 30
	teacherAlbumPhotosMin     = 1
	teacherAlbumPhotosMax     = 50
	teacherAlbumPageSizeMax   = 50
	teacherAlbumDateLayout    = "2006-01-02"
)

// teacherAlbumCategoryLabels 分类四值的中文标签（设计稿第五个胶囊「全部」是筛选值，不入库）。
var teacherAlbumCategoryLabels = map[domain.AlbumCategory]string{
	domain.AlbumCategoryActivity: "活动",
	domain.AlbumCategoryDaily:    "日常",
	domain.AlbumCategoryWork:     "作品",
	domain.AlbumCategoryFestival: "节日",
}

// albumRepo 相册能力直接挂在 TeacherPortalService 上，不新增独立 service：
// handler 聚合结构体与 service 的 Wire Set 都不在本切片改动清单内，新增独立 service
// 会牵动这两个共享文件。AlbumRepo 只是 db 的薄封装，按调用构造没有额外成本。
func (s *TeacherPortalService) albumRepo() *repository.AlbumRepo {
	return repository.NewAlbumRepo(s.db)
}

// TeacherAlbumPhotoInput 创建相册时的一张已上传照片。
type TeacherAlbumPhotoInput struct {
	FileURL  string `json:"file_url"`
	ThumbURL string `json:"thumb_url"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

// TeacherAlbumInput 创建相册请求体。
type TeacherAlbumInput struct {
	Title     string                   `json:"title"`
	Category  string                   `json:"category"`
	AlbumDate string                   `json:"album_date"`
	Photos    []TeacherAlbumPhotoInput `json:"photos"`
}

func normalizeTeacherAlbumCategory(raw string) (domain.AlbumCategory, bool) {
	category := domain.AlbumCategory(strings.TrimSpace(raw))
	_, ok := teacherAlbumCategoryLabels[category]
	return category, ok
}

func teacherAlbumCategoryLabel(category domain.AlbumCategory) string {
	return teacherAlbumCategoryLabels[category]
}

// isHTTPURL 只接受 http(s) 绝对 URL。上传通道返回的就是这种 URL；
// 相对路径/临时文件路径写进相册后，其它端无法加载。
func isHTTPURL(raw string) bool {
	if raw == "" {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	scheme := strings.ToLower(u.Scheme)
	return (scheme == "http" || scheme == "https") && u.Host != ""
}

func teacherAlbumDateString(t time.Time) string {
	return t.In(shanghaiLoc()).Format(teacherAlbumDateLayout)
}

// AlbumStats 相册统计卡：照片总数 / 本月新增 / 相册数。
func (s *TeacherPortalService) AlbumStats(ctx context.Context, tc *TeacherPortalContext) (map[string]any, error) {
	now := teacherPortalDay(time.Now())
	monthStart := now.AddDate(0, 0, -now.Day()+1)
	stats, err := s.albumRepo().Stats(ctx, tc.KindergartenID, tc.ClassID, monthStart, monthStart.AddDate(0, 1, 0))
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"total_photos": stats.TotalPhotos,
		"monthly_new":  stats.MonthlyNew,
		"album_count":  stats.AlbumCount,
	}, nil
}

// ListAlbums 本班相册分页。category 为空返回全部；非法分类按参数错误拒绝，
// 不静默退化成「全部」——否则客户端传错值时会看到与预期不符的列表。
func (s *TeacherPortalService) ListAlbums(
	ctx context.Context,
	tc *TeacherPortalContext,
	category string,
	page, pageSize int,
) (map[string]any, error) {
	var filter domain.AlbumCategory
	if strings.TrimSpace(category) != "" {
		c, ok := normalizeTeacherAlbumCategory(category)
		if !ok {
			return nil, ErrTeacherInvalidParams
		}
		filter = c
	}

	q := parseTeacherPortalPage(page, pageSize)
	if q.PageSize > teacherAlbumPageSizeMax {
		q.PageSize = teacherAlbumPageSizeMax
	}

	albums, total, err := s.albumRepo().List(ctx, tc.KindergartenID, tc.ClassID, filter, q.Offset(), q.PageSize)
	if err != nil {
		return nil, err
	}

	list := make([]map[string]any, 0, len(albums))
	for i := range albums {
		list = append(list, teacherAlbumListItem(&albums[i]))
	}
	return map[string]any{"list": list, "total": total, "page": q.Page, "page_size": q.PageSize}, nil
}

// GetAlbum 相册详情。跨班/跨园与不存在一样返回 ErrTeacherNotFound（404）。
func (s *TeacherPortalService) GetAlbum(ctx context.Context, tc *TeacherPortalContext, id uint64) (map[string]any, error) {
	album, err := s.albumRepo().FindByID(ctx, tc.KindergartenID, tc.ClassID, id)
	if err != nil {
		return nil, err
	}
	if album == nil {
		return nil, ErrTeacherNotFound
	}

	photos, err := s.albumRepo().ListPhotos(ctx, album.ID)
	if err != nil {
		return nil, err
	}
	photoList := make([]map[string]any, 0, len(photos))
	for _, p := range photos {
		photoList = append(photoList, map[string]any{
			"id":        p.ID,
			"file_url":  p.FileURL,
			"thumb_url": p.ThumbURL,
			"width":     p.Width,
			"height":    p.Height,
		})
	}

	out := teacherAlbumListItem(album)
	out["photos"] = photoList
	return out, nil
}

// CreateAlbum 创建相册。整单校验：任一字段非法都不落库。
func (s *TeacherPortalService) CreateAlbum(
	ctx context.Context,
	tc *TeacherPortalContext,
	req TeacherAlbumInput,
) (map[string]any, error) {
	album, photos, err := buildTeacherAlbum(tc, req)
	if err != nil {
		return nil, err
	}
	if err := s.albumRepo().Create(ctx, album, photos); err != nil {
		return nil, err
	}
	return map[string]any{"id": album.ID}, nil
}

func teacherAlbumListItem(a *domain.Album) map[string]any {
	return map[string]any{
		"id":             a.ID,
		"title":          a.Title,
		"category":       string(a.Category),
		"category_label": teacherAlbumCategoryLabel(a.Category),
		"album_date":     teacherAlbumDateString(a.AlbumDate),
		"photo_count":    a.PhotoCount,
		"cover_url":      a.CoverURL,
	}
}

func buildTeacherAlbum(
	tc *TeacherPortalContext,
	req TeacherAlbumInput,
) (*domain.Album, []domain.AlbumPhoto, error) {
	title := strings.TrimSpace(req.Title)
	if n := utf8.RuneCountInString(title); n < 1 || n > teacherAlbumTitleMaxRunes {
		return nil, nil, ErrTeacherInvalidParams
	}

	category, ok := normalizeTeacherAlbumCategory(req.Category)
	if !ok {
		return nil, nil, ErrTeacherInvalidParams
	}

	albumDate, err := time.ParseInLocation(teacherAlbumDateLayout, strings.TrimSpace(req.AlbumDate), shanghaiLoc())
	if err != nil {
		return nil, nil, ErrTeacherInvalidParams
	}

	if len(req.Photos) < teacherAlbumPhotosMin || len(req.Photos) > teacherAlbumPhotosMax {
		return nil, nil, ErrTeacherInvalidParams
	}
	photos := make([]domain.AlbumPhoto, 0, len(req.Photos))
	for _, p := range req.Photos {
		fileURL := strings.TrimSpace(p.FileURL)
		if !isHTTPURL(fileURL) {
			return nil, nil, ErrTeacherInvalidParams
		}
		thumbURL := strings.TrimSpace(p.ThumbURL)
		if thumbURL != "" && !isHTTPURL(thumbURL) {
			return nil, nil, ErrTeacherInvalidParams
		}
		if p.Width < 0 || p.Height < 0 {
			return nil, nil, ErrTeacherInvalidParams
		}
		photos = append(photos, domain.AlbumPhoto{
			FileURL: fileURL, ThumbURL: thumbURL, Width: p.Width, Height: p.Height,
		})
	}

	return &domain.Album{
		KindergartenID: tc.KindergartenID,
		ClassID:        tc.ClassID,
		CreatorID:      tc.UserID,
		Title:          title,
		Category:       category,
		AlbumDate:      albumDate,
		CoverURL:       photos[0].FileURL,
		PhotoCount:     len(photos),
	}, photos, nil
}
