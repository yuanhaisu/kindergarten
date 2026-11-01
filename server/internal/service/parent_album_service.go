package service

import (
	"context"
	"strings"
	"time"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// ParentAlbumScope 家长端班级相册的可见范围。
type ParentAlbumScope string

const (
	// ParentAlbumScopeMine 只返回包含当前孩子的照片，是 Phase 1 的默认行为。
	ParentAlbumScopeMine ParentAlbumScope = "mine"
	// ParentAlbumScopeAll 全班视图：不做我的孩子过滤，仅在园所开关开启时可用。
	ParentAlbumScopeAll ParentAlbumScope = "all"
)

// ParseParentAlbumScope 解析 query 中的 scope：缺省（含空白）为 mine，只接受 mine|all。
// 其它值返回 false 由 handler 回 422——把拼错的 scope 静默当 mine 会让端上以为
// 已切到全班视图，实际看到的还是自己的照片。
func ParseParentAlbumScope(raw string) (ParentAlbumScope, bool) {
	switch strings.TrimSpace(raw) {
	case "", string(ParentAlbumScopeMine):
		return ParentAlbumScopeMine, true
	case string(ParentAlbumScopeAll):
		return ParentAlbumScopeAll, true
	default:
		return "", false
	}
}

// parentAlbumScopeArg 解析可选 scope 参数：不传按 Phase 1 的 mine，
// 保留既有调用点（3 参详情 / 4 参列表）不改；非法值返回 ErrParentInvalidParams。
func parentAlbumScopeArg(args []ParentAlbumScope) (ParentAlbumScope, error) {
	if len(args) > 1 {
		return "", ErrParentInvalidParams
	}
	scope := ParentAlbumScopeMine
	if len(args) == 1 {
		scope = args[0]
		if scope == "" {
			scope = ParentAlbumScopeMine
		}
	}
	if scope != ParentAlbumScopeMine && scope != ParentAlbumScopeAll {
		return "", ErrParentInvalidParams
	}
	return scope, nil
}

// parentClasswideEnabled 读园所级「全班视图」开关。没有配置行按关闭处理，
// 且不在读取路径写库（家长打开一次相册不应创建园所设置行）。
func (s *ParentPortalService) parentClasswideEnabled(ctx context.Context, kindergartenID uint64) (bool, error) {
	row, err := repository.NewKindergartenAppSettingRepo(s.db).FindByKindergartenID(ctx, kindergartenID)
	if err != nil {
		return false, err
	}
	if row == nil {
		return false, nil
	}
	return row.ClassAlbumClasswideEnabled, nil
}

// albumRepo 与老师端同一做法：AlbumRepo 只是 db 的薄封装，按调用构造，
// 不改 ParentPortalService 的构造签名（wire_gen 与既有测试调用点都按现签名）。
func (s *ParentPortalService) albumRepo() *repository.AlbumRepo {
	return repository.NewAlbumRepo(s.db)
}

// 学生状态取值与 domain.Student.Status 注释一致：
// 0 待入学 / 1 在读 / 2 毕业 / 3 退学（含转园）。
const (
	parentStudentStatusGraduated int8 = 2
	parentStudentStatusLeft      int8 = 3
)

// parentAlbumRetentionWindow 毕业后保留只读访问的窗口（design D8）。
const parentAlbumRetentionWindow = 90 * 24 * time.Hour

// parentAlbumAccessAllowed 判断当前孩子此刻能否访问班级相册。
//
// 毕业按 status_changed_at 起算 90 天；历史毕业/退学没有时间戳的按已过期处理，
// 不用 updated_at 猜（design D2/D8）。退学/转园立即失去访问。
func parentAlbumAccessAllowed(st *domain.Student, now time.Time) bool {
	if st == nil {
		return false
	}
	switch st.Status {
	case parentStudentStatusGraduated:
		if st.StatusChangedAt == nil {
			return false
		}
		return now.Sub(*st.StatusChangedAt) <= parentAlbumRetentionWindow
	case parentStudentStatusLeft:
		return false
	default:
		return true
	}
}

// ListClassAlbums 家长端班级相册列表：只返回当前孩子所在班级、同园、已发布的相册。
//
// scope 缺省 mine；all 用于全班视图，要求园所开关开启（否则 ErrParentForbidden），
// 且不做 my-child 过滤。两种 scope 都遵守 opt-out 整张隐藏与自己的孩子例外。
//
// 列表只给相册元信息 + visible_photo_count + cover_urls + classwide_enabled，不返回全量照片。
// 查询次数最多 7（Count / 相册 / 照片 / 照片级关联 / 相册级参与者 / opt-out / 园所开关），
// 与相册数量无关；「逐相册查详情再数照片」会退化成 N+1（design D4）。
func (s *ParentPortalService) ListClassAlbums(
	ctx context.Context,
	pc *ParentPortalContext,
	page, pageSize int,
	scopeArgs ...ParentAlbumScope,
) (map[string]any, error) {
	if pc == nil || pc.CurrentStudent == nil {
		return nil, ErrParentForbidden
	}
	scope, err := parentAlbumScopeArg(scopeArgs)
	if err != nil {
		return nil, err
	}
	q := parseTeacherPortalPage(page, pageSize)
	if q.PageSize > teacherAlbumPageSizeMax {
		q.PageSize = teacherAlbumPageSizeMax
	}
	// 留存窗口外与班级缺失都返回空列表：相册是按班的资产，没有班级就没有可看的相册。
	if pc.CurrentStudent.ClassID == 0 || !parentAlbumAccessAllowed(pc.CurrentStudent, time.Now()) {
		return parentAlbumEmptyPage(q.Page, q.PageSize), nil
	}

	// classwide_enabled 决定家长端是否渲染切换入口；即使当前是 mine 也必须返回。
	classwide, err := s.parentClasswideEnabled(ctx, pc.KindergartenID)
	if err != nil {
		return nil, err
	}
	if scope == ParentAlbumScopeAll && !classwide {
		return nil, ErrParentForbidden
	}

	repo := s.albumRepo()
	albums, total, err := repo.List(ctx, pc.KindergartenID, pc.CurrentStudent.ClassID,
		"", domain.AlbumStatusPublished, q.Offset(), q.PageSize)
	if err != nil {
		return nil, err
	}

	albumIDs := make([]uint64, 0, len(albums))
	for i := range albums {
		albumIDs = append(albumIDs, albums[i].ID)
	}
	photosByAlbum, linksByPhoto, participantsByAlbum, optOut, err :=
		s.parentAlbumPhotoSources(ctx, pc.KindergartenID, albumIDs)
	if err != nil {
		return nil, err
	}

	ownChildIDs := parentOwnChildIDs(pc)
	list := make([]map[string]any, 0, len(albums))
	for i := range albums {
		album := &albums[i]
		var visible []domain.AlbumPhoto
		if scope == ParentAlbumScopeAll {
			visible = parentVisibleAlbumClasswidePhotos(
				photosByAlbum[album.ID], linksByPhoto, participantsByAlbum[album.ID],
				ownChildIDs, optOut)
		} else {
			visible = parentVisibleAlbumPhotos(
				photosByAlbum[album.ID], linksByPhoto, participantsByAlbum[album.ID],
				pc.CurrentStudent.ID, ownChildIDs, optOut)
		}
		item := parentAlbumListItem(album)
		item["visible_photo_count"] = len(visible)
		item["cover_urls"] = parentAlbumCoverURLs(visible)
		list = append(list, item)
	}
	return map[string]any{
		"list": list, "total": total, "page": q.Page, "page_size": q.PageSize,
		"classwide_enabled": classwide,
	}, nil
}

// GetClassAlbum 家长端班级相册详情：只返回可见照片，顺序仍按 sort_num、id。
// 草稿/下架/软删与跨班、跨园、留存过期一律 ErrParentNotFound（404），不泄露存在性。
// scope=all 在园所开关关闭时返回 ErrParentForbidden（403），与「相册不存在」区分开。
func (s *ParentPortalService) GetClassAlbum(
	ctx context.Context,
	pc *ParentPortalContext,
	id uint64,
	scopeArgs ...ParentAlbumScope,
) (map[string]any, error) {
	if pc == nil || pc.CurrentStudent == nil {
		return nil, ErrParentForbidden
	}
	scope, err := parentAlbumScopeArg(scopeArgs)
	if err != nil {
		return nil, err
	}
	if pc.CurrentStudent.ClassID == 0 || !parentAlbumAccessAllowed(pc.CurrentStudent, time.Now()) {
		return nil, ErrParentNotFound
	}
	if scope == ParentAlbumScopeAll {
		classwide, err := s.parentClasswideEnabled(ctx, pc.KindergartenID)
		if err != nil {
			return nil, err
		}
		if !classwide {
			return nil, ErrParentForbidden
		}
	}

	repo := s.albumRepo()
	album, err := repo.FindByIDAnyStatus(ctx, pc.KindergartenID, pc.CurrentStudent.ClassID, id)
	if err != nil {
		return nil, err
	}
	if album == nil || album.Status != domain.AlbumStatusPublished {
		return nil, ErrParentNotFound
	}

	photos, err := repo.ListPhotos(ctx, album.ID)
	if err != nil {
		return nil, err
	}
	photoIDs := make([]uint64, 0, len(photos))
	for _, p := range photos {
		photoIDs = append(photoIDs, p.ID)
	}
	linksByPhoto, err := repo.ListPhotoStudentIDsByPhotoIDs(ctx, photoIDs)
	if err != nil {
		return nil, err
	}
	participantsByAlbum, err := repo.ListAlbumStudentIDsByAlbumIDs(ctx, []uint64{album.ID})
	if err != nil {
		return nil, err
	}
	participants := participantsByAlbum[album.ID]
	optOut, err := s.parentOptOutStudents(ctx, pc.KindergartenID, linksByPhoto, participants)
	if err != nil {
		return nil, err
	}

	ownChildIDs := parentOwnChildIDs(pc)
	var visible []domain.AlbumPhoto
	if scope == ParentAlbumScopeAll {
		visible = parentVisibleAlbumClasswidePhotos(photos, linksByPhoto, participants, ownChildIDs, optOut)
	} else {
		visible = parentVisibleAlbumPhotos(
			photos, linksByPhoto, participants, pc.CurrentStudent.ID, ownChildIDs, optOut)
	}
	out := parentAlbumListItem(album)
	out["visible_photo_count"] = len(visible)
	out["photos"] = parentAlbumPhotoList(visible)
	// 成功返回详情才计一次查看：404/无权限路径不产生事件。
	// 字段只有 ID 与可见张数，不记录手机号/openid（design D10）。
	LogAlbumParentView(ctx, AlbumParentViewEvent{
		KindergartenID:    pc.KindergartenID,
		ClassID:           album.ClassID,
		AlbumID:           album.ID,
		StudentID:         pc.CurrentStudent.ID,
		VisiblePhotoCount: len(visible),
	})
	return out, nil
}

// parentAlbumPhotoSources 一次取齐整页相册的可见性判断输入（4 条批量查询）。
func (s *ParentPortalService) parentAlbumPhotoSources(
	ctx context.Context,
	kindergartenID uint64,
	albumIDs []uint64,
) (
	map[uint64][]domain.AlbumPhoto,
	map[uint64][]uint64,
	map[uint64][]uint64,
	map[uint64]bool,
	error,
) {
	emptyPhotos := map[uint64][]domain.AlbumPhoto{}
	emptyLinks := map[uint64][]uint64{}
	emptyParticipants := map[uint64][]uint64{}
	if len(albumIDs) == 0 {
		return emptyPhotos, emptyLinks, emptyParticipants, map[uint64]bool{}, nil
	}

	repo := s.albumRepo()
	photosByAlbum, err := repo.ListPhotosByAlbumIDs(ctx, albumIDs)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	photoIDs := make([]uint64, 0)
	for _, photos := range photosByAlbum {
		for _, p := range photos {
			photoIDs = append(photoIDs, p.ID)
		}
	}
	linksByPhoto, err := repo.ListPhotoStudentIDsByPhotoIDs(ctx, photoIDs)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	participantsByAlbum, err := repo.ListAlbumStudentIDsByAlbumIDs(ctx, albumIDs)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	allParticipants := make([]uint64, 0)
	for _, ids := range participantsByAlbum {
		allParticipants = append(allParticipants, ids...)
	}
	optOut, err := s.parentOptOutStudents(ctx, kindergartenID, linksByPhoto, allParticipants)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return photosByAlbum, linksByPhoto, participantsByAlbum, optOut, nil
}

// parentOptOutStudents 本次判断涉及的全部幼儿里，至少一位监护人关闭班内共享的集合。
// 没有 parent_settings 行按默认开启处理（design D6），SQL 在 repo 里只查显式 false，
// 同一幼儿多监护人时「任一关闭即关闭」也由那条 join 天然表达（design D5）。
func (s *ParentPortalService) parentOptOutStudents(
	ctx context.Context,
	kindergartenID uint64,
	linksByPhoto map[uint64][]uint64,
	participants []uint64,
) (map[uint64]bool, error) {
	seen := make(map[uint64]bool)
	ids := make([]uint64, 0, len(participants))
	add := func(id uint64) {
		if id == 0 || seen[id] {
			return
		}
		seen[id] = true
		ids = append(ids, id)
	}
	for _, rows := range linksByPhoto {
		for _, id := range rows {
			add(id)
		}
	}
	for _, id := range participants {
		add(id)
	}
	if len(ids) == 0 {
		return map[uint64]bool{}, nil
	}
	return s.albumRepo().ListOptOutStudentIDs(ctx, kindergartenID, ids)
}

// parentVisibleAlbumPhotos 按 design D4 过滤照片：
//  1. 照片有逐张幼儿关联时，当前孩子必须在其中；没有逐张关联时按相册级参与者兜底；
//  2. 照片关联到任何关闭班内共享的幼儿且该幼儿不是本家长的孩子时，整张隐藏；
//     本家长的孩子始终可见——撤回的是「别的家长看自己孩子」的权利。
//
// currentChildID 决定「我的孩子」视图；ownChildIDs 是当前家长全部孩子（multi-child
// 家庭里，另一位监护人对某个孩子关闭共享时，不该连带隐藏同一家长眼前的同胞合影）。
// 输入照片已按 sort_num、id 排序，过滤只做删减，不改变相对顺序。
func parentVisibleAlbumPhotos(
	photos []domain.AlbumPhoto,
	linksByPhoto map[uint64][]uint64,
	participants []uint64,
	currentChildID uint64,
	ownChildIDs map[uint64]bool,
	optOut map[uint64]bool,
) []domain.AlbumPhoto {
	return parentFilterAlbumPhotos(
		photos, linksByPhoto, participants, currentChildID, ownChildIDs, optOut, true)
}

// parentVisibleAlbumClasswidePhotos 全班视图的过滤：去掉「必须含当前孩子」这一步，
// opt-out 整张隐藏与自己的孩子例外保持不变（openspec design D2/D4）。
// 照片没有任何逐张关联时仍按相册级参与者兜底；两者都为空时无法判断照片里有谁，
// 只能依赖 Phase 1 已记录的「未逐张标记的合影」残余风险，不做额外猜测。
func parentVisibleAlbumClasswidePhotos(
	photos []domain.AlbumPhoto,
	linksByPhoto map[uint64][]uint64,
	participants []uint64,
	ownChildIDs map[uint64]bool,
	optOut map[uint64]bool,
) []domain.AlbumPhoto {
	return parentFilterAlbumPhotos(
		photos, linksByPhoto, participants, 0, ownChildIDs, optOut, false)
}

// parentFilterAlbumPhotos 是两种 scope 共用的过滤核心，只由 requireCurrentChild 区分：
// true = mine（当前孩子必须在照片的有效关联里），false = all（不按孩子裁剪）。
func parentFilterAlbumPhotos(
	photos []domain.AlbumPhoto,
	linksByPhoto map[uint64][]uint64,
	participants []uint64,
	currentChildID uint64,
	ownChildIDs map[uint64]bool,
	optOut map[uint64]bool,
	requireCurrentChild bool,
) []domain.AlbumPhoto {
	out := make([]domain.AlbumPhoto, 0, len(photos))
	for _, p := range photos {
		linked := linksByPhoto[p.ID]
		effective := linked
		if len(linked) == 0 {
			effective = participants
		}
		if requireCurrentChild && !parentAlbumContainsID(effective, currentChildID) {
			continue
		}
		hidden := false
		for _, studentID := range effective {
			if !ownChildIDs[studentID] && optOut[studentID] {
				hidden = true
				break
			}
		}
		if hidden {
			continue
		}
		out = append(out, p)
	}
	return out
}

// parentOwnChildIDs 当前家长在本园所绑定的全部孩子（含当前孩子）。
func parentOwnChildIDs(pc *ParentPortalContext) map[uint64]bool {
	out := make(map[uint64]bool, len(pc.Students)+1)
	for _, st := range pc.Students {
		out[st.ID] = true
	}
	if pc.CurrentStudent != nil {
		out[pc.CurrentStudent.ID] = true
	}
	return out
}

func parentAlbumContainsID(ids []uint64, id uint64) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

// parentAlbumListItem 家长端相册元信息。复用老师端字段语义（title/category/
// category_label/album_date/photo_count），不暴露 status/published_by 等运营字段。
func parentAlbumListItem(a *domain.Album) map[string]any {
	return map[string]any{
		"id":             a.ID,
		"title":          a.Title,
		"category":       string(a.Category),
		"category_label": teacherAlbumCategoryLabel(a.Category),
		"album_date":     teacherAlbumDateString(a.AlbumDate),
		"photo_count":    a.PhotoCount,
	}
}

// parentAlbumPhotoList 只暴露家长渲染所需字段：不返回照片关联的其他幼儿，
// 否则等于把班里其他孩子的身份信息发给每一位家长。
func parentAlbumPhotoList(photos []domain.AlbumPhoto) []map[string]any {
	out := make([]map[string]any, 0, len(photos))
	for _, p := range photos {
		out = append(out, map[string]any{
			"id":        p.ID,
			"file_url":  p.FileURL,
			"thumb_url": p.ThumbURL,
			"width":     p.Width,
			"height":    p.Height,
		})
	}
	return out
}

// parentAlbumCoverURLs 封面只取可见照片的前 3 张：老师端语义是「相册前 3 张」，
// 但家长端若取未过滤的原始前 3 张，会把别人孩子（或已关闭共享的孩子）的脸
// 当作封面返回，等同绕过照片级过滤。
func parentAlbumCoverURLs(visible []domain.AlbumPhoto) []string {
	n := len(visible)
	if n > teacherAlbumCoverMax {
		n = teacherAlbumCoverMax
	}
	urls := make([]string, 0, n)
	for i := 0; i < n; i++ {
		urls = append(urls, visible[i].FileURL)
	}
	return urls
}

func parentAlbumEmptyPage(page, pageSize int) map[string]any {
	return map[string]any{
		"list":      []map[string]any{},
		"total":     int64(0),
		"page":      page,
		"page_size": pageSize,
	}
}
