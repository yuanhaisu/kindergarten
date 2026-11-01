package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/pkg/log"
)

const (
	teacherAlbumTitleMaxRunes = 30
	teacherAlbumPhotosMin     = 1
	teacherAlbumPhotosMax     = 50
	teacherAlbumPageSizeMax   = 50
	teacherAlbumDateLayout    = "2006-01-02"
	// teacherAlbumCoverMax 设计稿相册卡是 3 格封面，列表只返回前 3 张。
	teacherAlbumCoverMax = 3
	// teacherAlbumReportTimeLayout 发布时间按 RFC3339（UTC）输出，与其它端一致。
	teacherAlbumReportTimeLayout = time.RFC3339
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

// TeacherAlbumPhotoInput 创建/编辑相册时的一张照片。
//
// 文件来源三选一（优先级从高到低）：
//   - SourceMediaID：从成长记录选图，file_url/thumb_url/file_id 以 growth_media 为准；
//   - FileID：本次上传的文件行，URL 以 files 为准，不接受端上改写；
//   - FileURL：历史/兼容路径，仅做 http(s) 校验，file_id 留空。
//
// StudentIDs 的 nil 与空切片的区别是契约的一部分：nil 表示未提交，
// 从成长记录选图时自动带出该记录关联的幼儿；非 nil（含 []）表示显式覆盖。
type TeacherAlbumPhotoInput struct {
	ID            uint64   `json:"id"`
	FileID        uint64   `json:"file_id"`
	FileURL       string   `json:"file_url"`
	ThumbURL      string   `json:"thumb_url"`
	Width         int      `json:"width"`
	Height        int      `json:"height"`
	SourceMediaID uint64   `json:"source_media_id"`
	StudentIDs    []uint64 `json:"student_ids"`
}

// TeacherAlbumInput 创建/编辑相册请求体。Participants 是相册级参与幼儿
// （批量上传的默认参与者，也是照片未逐张标记时的兜底）。
type TeacherAlbumInput struct {
	Title        string                   `json:"title"`
	Category     string                   `json:"category"`
	AlbumDate    string                   `json:"album_date"`
	Participants []uint64                 `json:"participants"`
	Photos       []TeacherAlbumPhotoInput `json:"photos"`
}

func normalizeTeacherAlbumCategory(raw string) (domain.AlbumCategory, bool) {
	category := domain.AlbumCategory(strings.TrimSpace(raw))
	_, ok := teacherAlbumCategoryLabels[category]
	return category, ok
}

func teacherAlbumCategoryLabel(category domain.AlbumCategory) string {
	return teacherAlbumCategoryLabels[category]
}

func validAlbumStatus(raw string) bool {
	switch raw {
	case domain.AlbumStatusDraft, domain.AlbumStatusPublished, domain.AlbumStatusUnpublished:
		return true
	default:
		return false
	}
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

// teacherAlbumPublishedAt 未发布（草稿/下架）返回 nil，端上据 status 判断展示。
func teacherAlbumPublishedAt(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(teacherAlbumReportTimeLayout)
}

// AlbumStats 相册统计卡：照片总数 / 本月新增 / 相册数。只统计已发布相册。
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

// ListAlbums 本班相册分页。category/status 为空返回全部；非法值按参数错误拒绝，
// 不静默退化成「全部」——否则客户端传错值时会看到与预期不符的列表。
func (s *TeacherPortalService) ListAlbums(
	ctx context.Context,
	tc *TeacherPortalContext,
	category, status string,
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

	statusFilter := strings.TrimSpace(status)
	if statusFilter != "" && !validAlbumStatus(statusFilter) {
		return nil, ErrTeacherInvalidParams
	}

	q := parseTeacherPortalPage(page, pageSize)
	if q.PageSize > teacherAlbumPageSizeMax {
		q.PageSize = teacherAlbumPageSizeMax
	}

	albums, total, err := s.albumRepo().List(ctx, tc.KindergartenID, tc.ClassID, filter, statusFilter, q.Offset(), q.PageSize)
	if err != nil {
		return nil, err
	}

	albumIDs := make([]uint64, 0, len(albums))
	for i := range albums {
		albumIDs = append(albumIDs, albums[i].ID)
	}
	covers, err := s.albumRepo().ListFirstPhotos(ctx, albumIDs, teacherAlbumCoverMax)
	if err != nil {
		return nil, err
	}

	list := make([]map[string]any, 0, len(albums))
	for i := range albums {
		item := teacherAlbumListItem(&albums[i])
		item["cover_urls"] = teacherAlbumCoverURLs(covers[albums[i].ID])
		list = append(list, item)
	}
	return map[string]any{"list": list, "total": total, "page": q.Page, "page_size": q.PageSize}, nil
}

// GetAlbum 相册详情。老师端不按状态过滤：草稿/下架相册仍要可编辑与预览。
// 跨班/跨园与不存在一样返回 ErrTeacherNotFound（404）。
func (s *TeacherPortalService) GetAlbum(ctx context.Context, tc *TeacherPortalContext, id uint64) (map[string]any, error) {
	repo := s.albumRepo()
	album, err := repo.FindByIDAnyStatus(ctx, tc.KindergartenID, tc.ClassID, id)
	if err != nil {
		return nil, err
	}
	if album == nil {
		return nil, ErrTeacherNotFound
	}

	photos, photoStudents, err := repo.ListPhotosWithStudents(ctx, album.ID)
	if err != nil {
		return nil, err
	}
	participants, err := repo.ListAlbumStudents(ctx, album.ID)
	if err != nil {
		return nil, err
	}

	unauthorized, err := s.countUnauthorizedAlbumPhotos(ctx, tc.KindergartenID, photos, photoStudents, participants)
	if err != nil {
		return nil, err
	}

	photoList := make([]map[string]any, 0, len(photos))
	for _, p := range photos {
		students := make([]map[string]any, 0, len(photoStudents[p.ID]))
		for _, st := range photoStudents[p.ID] {
			students = append(students, albumStudentView(st))
		}
		var fileID any
		if p.FileID != nil {
			fileID = *p.FileID
		}
		photoList = append(photoList, map[string]any{
			"id":        p.ID,
			"file_id":   fileID,
			"file_url":  p.FileURL,
			"thumb_url": p.ThumbURL,
			"width":     p.Width,
			"height":    p.Height,
			"students":  students,
		})
	}

	participantList := make([]map[string]any, 0, len(participants))
	for _, st := range participants {
		participantList = append(participantList, albumStudentView(st))
	}

	out := teacherAlbumListItem(album)
	out["album_students"] = participantList
	out["photos"] = photoList
	out["unauthorized_photo_count"] = unauthorized
	return out, nil
}

// countUnauthorizedAlbumPhotos 发布预览用：统计「含未开启班内共享幼儿」的照片数。
// 照片有逐张关联时以逐张为准；没有时按相册级参与者兜底（design D4）。
func (s *TeacherPortalService) countUnauthorizedAlbumPhotos(
	ctx context.Context,
	kindergartenID uint64,
	photos []domain.AlbumPhoto,
	photoStudents map[uint64][]repository.AlbumStudentRow,
	participants []repository.AlbumStudentRow,
) (int, error) {
	seen := make(map[uint64]bool)
	studentIDs := make([]uint64, 0, len(participants))
	add := func(id uint64) {
		if id == 0 || seen[id] {
			return
		}
		seen[id] = true
		studentIDs = append(studentIDs, id)
	}
	for _, st := range participants {
		add(st.StudentID)
	}
	for _, rows := range photoStudents {
		for _, st := range rows {
			add(st.StudentID)
		}
	}
	if len(studentIDs) == 0 {
		return 0, nil
	}

	optOut, err := s.albumRepo().ListOptOutStudentIDs(ctx, kindergartenID, studentIDs)
	if err != nil {
		return 0, err
	}
	if len(optOut) == 0 {
		return 0, nil
	}

	count := 0
	for _, p := range photos {
		effective := photoStudents[p.ID]
		if len(effective) == 0 {
			effective = participants
		}
		for _, st := range effective {
			if optOut[st.StudentID] {
				count++
				break
			}
		}
	}
	return count, nil
}

// CreateAlbum 创建相册草稿。整单校验：任一字段非法都不落库。
func (s *TeacherPortalService) CreateAlbum(
	ctx context.Context,
	tc *TeacherPortalContext,
	req TeacherAlbumInput,
) (map[string]any, error) {
	build, err := s.buildTeacherAlbum(ctx, tc, req, nil)
	if err != nil {
		return nil, err
	}
	build.album.Status = domain.AlbumStatusDraft
	if err := s.albumRepo().Create(ctx, build.album, build.photos, build.participants, build.photoStudents); err != nil {
		return nil, err
	}
	return map[string]any{"id": build.album.ID, "status": build.album.Status}, nil
}

// UpdateAlbum 编辑相册。已发布相册的改动保存后立即对家长生效；状态不变。
func (s *TeacherPortalService) UpdateAlbum(
	ctx context.Context,
	tc *TeacherPortalContext,
	id uint64,
	req TeacherAlbumInput,
) (map[string]any, error) {
	repo := s.albumRepo()
	album, err := repo.FindByIDAnyStatus(ctx, tc.KindergartenID, tc.ClassID, id)
	if err != nil {
		return nil, err
	}
	if album == nil {
		return nil, ErrTeacherNotFound
	}
	existing, err := repo.ListPhotos(ctx, album.ID)
	if err != nil {
		return nil, err
	}
	build, err := s.buildTeacherAlbum(ctx, tc, req, existing)
	if err != nil {
		return nil, err
	}
	build.album.ID = album.ID
	build.album.Status = album.Status
	build.album.PublishedAt = album.PublishedAt
	build.album.PublishedBy = album.PublishedBy
	if err := repo.Update(ctx, build.album, build.photos, build.participants, build.photoStudents); err != nil {
		if errors.Is(err, repository.ErrAlbumNotFound) {
			// 读到之后被并发软删：与一开始就不存在一样按 404 处理。
			return nil, ErrTeacherNotFound
		}
		return nil, err
	}
	// 审计要记录编辑后的标题，album 是编辑前的快照，只补齐可编辑字段。
	album.Title = build.album.Title
	album.Category = build.album.Category
	album.AlbumDate = build.album.AlbumDate
	s.writeAlbumAudit(ctx, tc, "edit", album, album.Status, album.Status)
	return map[string]any{"id": album.ID}, nil
}

// PublishAlbum 发布草稿/重新上架已下架相册。任一老师可操作，写审计。
func (s *TeacherPortalService) PublishAlbum(
	ctx context.Context,
	tc *TeacherPortalContext,
	id uint64,
) (map[string]any, error) {
	repo := s.albumRepo()
	album, err := repo.FindByIDAnyStatus(ctx, tc.KindergartenID, tc.ClassID, id)
	if err != nil {
		return nil, err
	}
	if album == nil {
		return nil, ErrTeacherNotFound
	}
	photos, err := repo.ListPhotos(ctx, album.ID)
	if err != nil {
		return nil, err
	}
	if !albumPublishable(album, len(photos)) {
		return nil, ErrTeacherInvalidParams
	}

	from := album.Status
	at := time.Now()
	ok, err := repo.UpdateStatus(ctx, tc.KindergartenID, tc.ClassID, album.ID,
		[]string{domain.AlbumStatusDraft, domain.AlbumStatusUnpublished},
		domain.AlbumStatusPublished, &at, tc.UserID)
	if err != nil {
		return nil, err
	}
	if !ok {
		// 状态校验后被并发改走（或已发布）：按参数错误让端上刷新后重试。
		return nil, ErrTeacherInvalidParams
	}
	album.Status = domain.AlbumStatusPublished
	album.PublishedAt = &at
	album.PublishedBy = tc.UserID
	s.writeAlbumAudit(ctx, tc, "publish", album, from, album.Status)
	// 指标事件只在发布成功、审计写入后发：字段仅 ID 与照片数（design D10）。
	LogAlbumPublish(ctx, AlbumPublishEvent{
		KindergartenID: tc.KindergartenID,
		ClassID:        tc.ClassID,
		AlbumID:        album.ID,
		UserID:         tc.UserID,
		PhotoCount:     len(photos),
	})
	return map[string]any{
		"id":           album.ID,
		"status":       album.Status,
		"published_at": at.UTC().Format(teacherAlbumReportTimeLayout),
	}, nil
}

// UnpublishAlbum 下架已发布相册。下架后家长端立即不可见，发布元数据清空。
func (s *TeacherPortalService) UnpublishAlbum(
	ctx context.Context,
	tc *TeacherPortalContext,
	id uint64,
) (map[string]any, error) {
	repo := s.albumRepo()
	album, err := repo.FindByIDAnyStatus(ctx, tc.KindergartenID, tc.ClassID, id)
	if err != nil {
		return nil, err
	}
	if album == nil {
		return nil, ErrTeacherNotFound
	}
	from := album.Status
	ok, err := repo.UpdateStatus(ctx, tc.KindergartenID, tc.ClassID, album.ID,
		[]string{domain.AlbumStatusPublished}, domain.AlbumStatusUnpublished, nil, 0)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrTeacherInvalidParams
	}
	album.Status = domain.AlbumStatusUnpublished
	album.PublishedAt = nil
	album.PublishedBy = 0
	s.writeAlbumAudit(ctx, tc, "unpublish", album, from, album.Status)
	return map[string]any{"id": album.ID, "status": album.Status}, nil
}

// DeleteAlbum 软删除相册。软删后家长端不可见；相册行保留 180 天供审计。
func (s *TeacherPortalService) DeleteAlbum(
	ctx context.Context,
	tc *TeacherPortalContext,
	id uint64,
) (map[string]any, error) {
	repo := s.albumRepo()
	album, err := repo.FindByIDAnyStatus(ctx, tc.KindergartenID, tc.ClassID, id)
	if err != nil {
		return nil, err
	}
	if album == nil {
		return nil, ErrTeacherNotFound
	}
	ok, err := repo.SoftDelete(ctx, tc.KindergartenID, tc.ClassID, album.ID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrTeacherNotFound
	}
	s.writeAlbumAudit(ctx, tc, "delete", album, album.Status, album.Status)
	return map[string]any{"id": album.ID}, nil
}

// writeAlbumAudit 写操作日志。审计失败不阻断主流程，但必须留下服务端日志，
// 不能静默吞掉（OperationLog 是事后追责的唯一依据）。
func (s *TeacherPortalService) writeAlbumAudit(
	ctx context.Context,
	tc *TeacherPortalContext,
	action string,
	album *domain.Album,
	from, to string,
) {
	detail := fmt.Sprintf("title=%s status=%s", album.Title, to)
	if from != "" && from != to {
		detail = fmt.Sprintf("title=%s status=%s->%s", album.Title, from, to)
	}
	if err := s.db.WithContext(ctx).Create(&domain.OperationLog{
		KindergartenID: tc.KindergartenID,
		UserID:         tc.UserID,
		Module:         "albums",
		Action:         action,
		TargetType:     "album",
		TargetID:       album.ID,
		Detail:         detail,
	}).Error; err != nil {
		log.L(ctx).Error("album audit write failed", "album", album.ID, "action", action, "err", err)
	}
}

func albumPublishable(album *domain.Album, photoCount int) bool {
	if photoCount < teacherAlbumPhotosMin {
		return false
	}
	title := strings.TrimSpace(album.Title)
	if n := utf8.RuneCountInString(title); n < 1 || n > teacherAlbumTitleMaxRunes {
		return false
	}
	if _, ok := teacherAlbumCategoryLabels[album.Category]; !ok {
		return false
	}
	return !album.AlbumDate.IsZero()
}

func albumStudentView(st repository.AlbumStudentRow) map[string]any {
	return map[string]any{
		"id":     st.StudentID,
		"name":   st.Name,
		"avatar": st.Avatar,
	}
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
		"status":         a.Status,
		"published_at":   teacherAlbumPublishedAt(a.PublishedAt),
		"published_by":   a.PublishedBy,
	}
}

// teacherAlbumCoverURLs 只取 file_url 与 cover_url 同源；空切片而非 nil，
// 保证 JSON 序列化为 [] 而不是 null，端上可直接 length 判断。
func teacherAlbumCoverURLs(photos []domain.AlbumPhoto) []string {
	urls := make([]string, 0, len(photos))
	for _, p := range photos {
		urls = append(urls, p.FileURL)
	}
	return urls
}

// teacherAlbumBuild 校验后的落库视图：album 携带冗余的 cover_url/photo_count，
// photos 与 photoStudents 按下标一一对应。
type teacherAlbumBuild struct {
	album         *domain.Album
	photos        []domain.AlbumPhoto
	participants  []uint64
	photoStudents [][]uint64
}

// buildTeacherAlbum 校验并组装创建/编辑入参。existing 为编辑前的照片（创建为 nil），
// 用来保证提交的 photo.id 属于本相册；创建时提交 id 一律视为非法。
func (s *TeacherPortalService) buildTeacherAlbum(
	ctx context.Context,
	tc *TeacherPortalContext,
	req TeacherAlbumInput,
	existing []domain.AlbumPhoto,
) (*teacherAlbumBuild, error) {
	title := strings.TrimSpace(req.Title)
	if n := utf8.RuneCountInString(title); n < 1 || n > teacherAlbumTitleMaxRunes {
		return nil, ErrTeacherInvalidParams
	}

	category, ok := normalizeTeacherAlbumCategory(req.Category)
	if !ok {
		return nil, ErrTeacherInvalidParams
	}

	albumDate, err := time.ParseInLocation(teacherAlbumDateLayout, strings.TrimSpace(req.AlbumDate), shanghaiLoc())
	if err != nil {
		return nil, ErrTeacherInvalidParams
	}

	if len(req.Photos) < teacherAlbumPhotosMin || len(req.Photos) > teacherAlbumPhotosMax {
		return nil, ErrTeacherInvalidParams
	}

	repo := s.albumRepo()
	existingSet := make(map[uint64]bool, len(existing))
	for _, p := range existing {
		existingSet[p.ID] = true
	}
	usedIDs := make(map[uint64]bool, len(req.Photos))

	participants := dedupeAlbumIDs(req.Participants)
	explicitStudents := make([]uint64, 0, len(participants))
	explicitStudents = append(explicitStudents, participants...)

	photos := make([]domain.AlbumPhoto, 0, len(req.Photos))
	photoStudents := make([][]uint64, len(req.Photos))
	// 从成长选图的两类批处理：file_id 反查与自动带出幼儿，都在循环后统一查询，避免 N+1。
	pendingFileURL := make(map[int]string)
	pendingLogID := make(map[int]uint64)

	for i, in := range req.Photos {
		if in.ID != 0 {
			if !existingSet[in.ID] || usedIDs[in.ID] {
				return nil, ErrTeacherInvalidParams
			}
			usedIDs[in.ID] = true
		}
		if in.Width < 0 || in.Height < 0 {
			return nil, ErrTeacherInvalidParams
		}

		photo := domain.AlbumPhoto{
			BaseModel: domain.BaseModel{ID: in.ID},
			Width:     in.Width,
			Height:    in.Height,
		}
		switch {
		case in.SourceMediaID > 0:
			media, err := repo.FindGrowthMediaInClass(ctx, tc.KindergartenID, tc.ClassID, in.SourceMediaID)
			if err != nil {
				return nil, err
			}
			if media == nil || media.FileType != "image" {
				return nil, ErrTeacherInvalidParams
			}
			photo.FileURL = media.FileURL
			photo.ThumbURL = media.ThumbURL
			pendingFileURL[i] = media.FileURL
			if media.LogID != nil {
				pendingLogID[i] = *media.LogID
			}
		case in.FileID > 0:
			files, err := repo.FindFilesUnderKindergartenTeachers(ctx, tc.KindergartenID, []uint64{in.FileID})
			if err != nil {
				return nil, err
			}
			f, ok := files[in.FileID]
			if !ok || f.FileType != "image" {
				return nil, ErrTeacherInvalidParams
			}
			fid := in.FileID
			photo.FileID = &fid
			photo.FileURL = f.FileURL
			photo.ThumbURL = f.ThumbURL
		default:
			fileURL := strings.TrimSpace(in.FileURL)
			if !isHTTPURL(fileURL) {
				return nil, ErrTeacherInvalidParams
			}
			thumbURL := strings.TrimSpace(in.ThumbURL)
			if thumbURL != "" && !isHTTPURL(thumbURL) {
				return nil, ErrTeacherInvalidParams
			}
			photo.FileURL = fileURL
			photo.ThumbURL = thumbURL
		}

		if in.StudentIDs != nil {
			ids := dedupeAlbumIDs(in.StudentIDs)
			photoStudents[i] = ids
			explicitStudents = append(explicitStudents, ids...)
			delete(pendingLogID, i) // 显式覆盖优先于成长记录的自动带出
		}
		photos = append(photos, photo)
	}

	// 成长记录来源没有 file_id 字段，只能按 URL 反查（与迁移 SQL 回填口径一致）。
	if len(pendingFileURL) > 0 {
		urls := make([]string, 0, len(pendingFileURL))
		for _, u := range pendingFileURL {
			urls = append(urls, u)
		}
		filesByURL, err := repo.FindFilesByURLUnderKindergartenTeachers(ctx, tc.KindergartenID, urls)
		if err != nil {
			return nil, err
		}
		for i, u := range pendingFileURL {
			if f, ok := filesByURL[u]; ok {
				fid := f.ID
				photos[i].FileID = &fid
			}
		}
	}

	// 自动带出：成长记录关联了哪些幼儿，照片级关联就写哪些（显式覆盖已在上面剔除）。
	if len(pendingLogID) > 0 {
		logIDs := make([]uint64, 0, len(pendingLogID))
		for _, logID := range pendingLogID {
			logIDs = append(logIDs, logID)
		}
		byLog, err := repo.ListGrowthLogStudentsByLogIDs(ctx, logIDs)
		if err != nil {
			return nil, err
		}
		for i, logID := range pendingLogID {
			photoStudents[i] = dedupeAlbumIDs(byLog[logID])
		}
	}

	// 显式（相册级 + 照片级）幼儿必须属于本园本班；自动带出的不做此校验——
	// 孩子转班/毕业后历史记录的关联仍然要参与隐私过滤。
	if len(explicitStudents) > 0 {
		explicitStudents = dedupeAlbumIDs(explicitStudents)
		found, err := repo.FindStudentsInClass(ctx, tc.KindergartenID, tc.ClassID, explicitStudents)
		if err != nil {
			return nil, err
		}
		if len(found) != len(explicitStudents) {
			return nil, ErrTeacherInvalidParams
		}
	}

	return &teacherAlbumBuild{
		album: &domain.Album{
			KindergartenID: tc.KindergartenID,
			ClassID:        tc.ClassID,
			CreatorID:      tc.UserID,
			Title:          title,
			Category:       category,
			AlbumDate:      albumDate,
			CoverURL:       photos[0].FileURL,
			PhotoCount:     len(photos),
		},
		photos:        photos,
		participants:  participants,
		photoStudents: photoStudents,
	}, nil
}

func dedupeAlbumIDs(ids []uint64) []uint64 {
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
