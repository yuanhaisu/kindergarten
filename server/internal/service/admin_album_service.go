package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/pkg/pagination"
)

// OS-ALB-5 管理端班级相册（openspec class-album-parent-sharing 6.1–6.3）：
// 园长只能只读列表/详情并下架已发布相册；admin 不提供创建/编辑/删除端点。
//
// 方法挂在 AdminService 上而不是新增独立 service：handler 聚合与 Wire Set
// 不在本切片改动清单内，AdminService 已有 db 与 logOp；AlbumRepo 只是 db 的
// 薄封装，按调用构造（与 teacher/parent 的 albumRepo() 同一做法）。
//
// 状态机口径见 design D3：published --unpublish--> unpublished。重复下架按
// 幂等成功处理（返回 changed=false，不重复写审计）；草稿下架是非法转换 422。
var (
	ErrAdminAlbumNotFound     = errors.New("ALBUM_NOT_FOUND")
	ErrAdminAlbumInvalidParam = errors.New("ALBUM_INVALID_PARAMS")
	ErrAdminAlbumNotPublished = errors.New("ALBUM_NOT_PUBLISHED")
)

func (s *AdminService) albumRepo() *repository.AlbumRepo {
	return repository.NewAlbumRepo(s.db)
}

type adminAlbumCreator struct {
	ID       uint64 `json:"id"`
	Nickname string `json:"nickname"`
}

// AdminAlbumListItem 契约字段与 design「契约字段溯源」一致：列表不返回照片，
// 只给元信息与冗余的 photo_count，避免管理端列表 N+1。
type AdminAlbumListItem struct {
	ID            uint64            `json:"id"`
	ClassID       uint64            `json:"class_id"`
	ClassName     string            `json:"class_name"`
	Title         string            `json:"title"`
	Category      string            `json:"category"`
	CategoryLabel string            `json:"category_label"`
	AlbumDate     string            `json:"album_date"`
	PhotoCount    int               `json:"photo_count"`
	Status        string            `json:"status"`
	PublishedAt   *string           `json:"published_at"`
	Creator       adminAlbumCreator `json:"creator"`
}

type AdminAlbumStudent struct {
	ID     uint64 `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type AdminAlbumPhoto struct {
	ID       uint64              `json:"id"`
	FileID   *uint64             `json:"file_id"`
	FileURL  string              `json:"file_url"`
	ThumbURL string              `json:"thumb_url"`
	Width    int                 `json:"width"`
	Height   int                 `json:"height"`
	SortNum  int                 `json:"sort_num"`
	Students []AdminAlbumStudent `json:"students"`
}

type AdminAlbumDetail struct {
	AdminAlbumListItem
	Photos       []AdminAlbumPhoto   `json:"photos"`
	Participants []AdminAlbumStudent `json:"participants"`
}

// AdminAlbumUnpublishResult Changed 区分真实状态迁移与重复下架的幂等成功；
// 审计只在前者写入（状态没变就不该留下第二条操作记录）。
type AdminAlbumUnpublishResult struct {
	ID      uint64 `json:"id"`
	Status  string `json:"status"`
	Changed bool   `json:"changed"`
}

func adminAlbumPublishedAt(t *time.Time) *string {
	if t == nil {
		return nil
	}
	v := t.UTC().Format(teacherAlbumReportTimeLayout)
	return &v
}

func adminAlbumListItem(row *repository.AdminAlbumRow) AdminAlbumListItem {
	return AdminAlbumListItem{
		ID:            row.ID,
		ClassID:       row.ClassID,
		ClassName:     row.ClassName,
		Title:         row.Title,
		Category:      row.Category,
		CategoryLabel: teacherAlbumCategoryLabel(domain.AlbumCategory(row.Category)),
		AlbumDate:     teacherAlbumDateString(row.AlbumDate),
		PhotoCount:    row.PhotoCount,
		Status:        row.Status,
		PublishedAt:   adminAlbumPublishedAt(row.PublishedAt),
		Creator:       adminAlbumCreator{ID: row.CreatorID, Nickname: row.CreatorNickname},
	}
}

// ListAdminAlbums 本园全部班级相册分页。非法 category/status 明确拒绝，
// 不静默退化成「全部」——管理端传错筛选值时应看到错误而不是全量列表。
func (s *AdminService) ListAdminAlbums(
	ctx context.Context,
	kindergartenID uint64,
	classID uint64,
	status, category string,
	pq pagination.Query,
) ([]AdminAlbumListItem, int64, int, int, error) {
	if kindergartenID == 0 {
		return nil, 0, 0, 0, fmt.Errorf("NO_KINDERGARTEN")
	}

	var categoryFilter domain.AlbumCategory
	if strings.TrimSpace(category) != "" {
		c, ok := normalizeTeacherAlbumCategory(category)
		if !ok {
			return nil, 0, 0, 0, ErrAdminAlbumInvalidParam
		}
		categoryFilter = c
	}
	statusFilter := strings.TrimSpace(status)
	if statusFilter != "" && !validAlbumStatus(statusFilter) {
		return nil, 0, 0, 0, ErrAdminAlbumInvalidParam
	}

	q := pq
	q.Normalize()
	rows, total, err := s.albumRepo().AdminList(
		ctx, kindergartenID, classID, categoryFilter, statusFilter, q.Offset(), q.PageSize)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	out := make([]AdminAlbumListItem, 0, len(rows))
	for i := range rows {
		out = append(out, adminAlbumListItem(&rows[i]))
	}
	return out, total, q.Page, q.PageSize, nil
}

// GetAdminAlbum 管理端详情：本园内任一班级的相册，草稿/已下架也可查看（园长只读）。
// 跨园、软删与不存在一律 ErrAdminAlbumNotFound（404）。
func (s *AdminService) GetAdminAlbum(ctx context.Context, kindergartenID, id uint64) (*AdminAlbumDetail, error) {
	if kindergartenID == 0 {
		return nil, fmt.Errorf("NO_KINDERGARTEN")
	}
	repo := s.albumRepo()
	row, err := repo.AdminFindByID(ctx, kindergartenID, id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrAdminAlbumNotFound
	}

	photos, photoStudents, err := repo.ListPhotosWithStudents(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	participants, err := repo.ListAlbumStudents(ctx, row.ID)
	if err != nil {
		return nil, err
	}

	photoList := make([]AdminAlbumPhoto, 0, len(photos))
	for _, p := range photos {
		students := make([]AdminAlbumStudent, 0, len(photoStudents[p.ID]))
		for _, st := range photoStudents[p.ID] {
			students = append(students, AdminAlbumStudent{
				ID: st.StudentID, Name: st.Name, Avatar: st.Avatar,
			})
		}
		photoList = append(photoList, AdminAlbumPhoto{
			ID:       p.ID,
			FileID:   p.FileID,
			FileURL:  p.FileURL,
			ThumbURL: p.ThumbURL,
			Width:    p.Width,
			Height:   p.Height,
			SortNum:  p.SortNum,
			Students: students,
		})
	}
	participantList := make([]AdminAlbumStudent, 0, len(participants))
	for _, st := range participants {
		participantList = append(participantList, AdminAlbumStudent{
			ID: st.StudentID, Name: st.Name, Avatar: st.Avatar,
		})
	}

	return &AdminAlbumDetail{
		AdminAlbumListItem: adminAlbumListItem(row),
		Photos:             photoList,
		Participants:       participantList,
	}, nil
}

// UnpublishAdminAlbum 下架已发布相册，返回状态是否真的发生变化。
// 家长端只查 status=published，因此下架后下一次读取立即不可见（design D4）。
//
// 幂等口径：
//   - published   → unpublished，写审计，changed=true；
//   - unpublished → 成功返回 changed=false，不再写审计；
//   - draft       → ErrAdminAlbumNotPublished（422），状态不变；
//   - 跨园/软删/不存在 → ErrAdminAlbumNotFound（404）。
func (s *AdminService) UnpublishAdminAlbum(
	ctx context.Context,
	kindergartenID, operatorID, id uint64,
	ip, ua string,
) (*AdminAlbumUnpublishResult, error) {
	if kindergartenID == 0 {
		return nil, fmt.Errorf("NO_KINDERGARTEN")
	}
	repo := s.albumRepo()
	row, err := repo.AdminFindByID(ctx, kindergartenID, id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrAdminAlbumNotFound
	}

	// 幂等：已下架直接成功，不重复写审计（状态没有变化）。
	if row.Status == domain.AlbumStatusUnpublished {
		return &AdminAlbumUnpublishResult{ID: row.ID, Status: row.Status, Changed: false}, nil
	}
	if row.Status != domain.AlbumStatusPublished {
		return nil, ErrAdminAlbumNotPublished
	}

	ok, err := repo.UpdateStatus(ctx, kindergartenID, row.ClassID, row.ID,
		[]string{domain.AlbumStatusPublished}, domain.AlbumStatusUnpublished, nil, 0)
	if err != nil {
		return nil, err
	}
	if !ok {
		// 读到 published 之后被并发下架/软删：重读后按同一口径收敛，
		// 不能把「别人已经做完了」误报成 500。
		latest, err := repo.AdminFindByID(ctx, kindergartenID, id)
		if err != nil {
			return nil, err
		}
		if latest == nil {
			return nil, ErrAdminAlbumNotFound
		}
		if latest.Status == domain.AlbumStatusUnpublished {
			return &AdminAlbumUnpublishResult{ID: latest.ID, Status: latest.Status, Changed: false}, nil
		}
		return nil, ErrAdminAlbumNotPublished
	}

	// 用 %q 拼 JSON 而不是 json.Marshal：后者会把 ">" 转义成 >，
	// 审计详情在日志/管理端里应保持可读。标题与班级名由 %q 负责转义。
	detail := fmt.Sprintf(`{"title":%q,"class_id":%d,"class_name":%q,"status":"%s->%s"}`,
		row.Title, row.ClassID, row.ClassName,
		domain.AlbumStatusPublished, domain.AlbumStatusUnpublished)
	s.logOp(ctx, kindergartenID, operatorID, "albums", "unpublish", "album", row.ID, detail, ip, ua)

	return &AdminAlbumUnpublishResult{
		ID:      row.ID,
		Status:  domain.AlbumStatusUnpublished,
		Changed: true,
	}, nil
}
