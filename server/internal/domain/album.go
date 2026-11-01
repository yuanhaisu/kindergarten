package domain

import "time"

// AlbumCategory 班级相册分类。设计稿的第五个胶囊「全部」是筛选值，不是分类。
type AlbumCategory string

const (
	AlbumCategoryActivity AlbumCategory = "activity"
	AlbumCategoryDaily    AlbumCategory = "daily"
	AlbumCategoryWork     AlbumCategory = "work"
	AlbumCategoryFestival AlbumCategory = "festival"
)

// 相册发布状态。新建即草稿，家长端只读 published；下架与软删都立即收回可见性。
const (
	AlbumStatusDraft       = "draft"
	AlbumStatusPublished   = "published"
	AlbumStatusUnpublished = "unpublished"
)

// Album 班级相册。封面与照片数在创建时冗余写入，列表无需 join album_photos。
// 编辑/删除照片时需在同一事务里重算这两个冗余值（Phase 1 起有编辑入口）。
//
// 复合索引 (kindergarten_id, class_id, status, album_date) 覆盖家长端
// 「同园同班 + 已发布 + 按日期倒序」这一条唯一的高频查询；四个字段的单列索引
// 仍保留，供老师端按班/按日期等其它筛选使用。
type Album struct {
	SoftDeleteModel
	KindergartenID uint64        `json:"kindergarten_id" gorm:"not null;index;index:idx_albums_kg_class_status_date,priority:1"`
	ClassID        uint64        `json:"class_id" gorm:"not null;index;index:idx_albums_kg_class_status_date,priority:2"`
	CreatorID      uint64        `json:"creator_id" gorm:"not null;index"`
	Title          string        `json:"title" gorm:"size:30;not null"`
	Category       AlbumCategory `json:"category" gorm:"size:20;not null;index"`
	AlbumDate      time.Time     `json:"album_date" gorm:"type:date;not null;index;index:idx_albums_kg_class_status_date,priority:4"`
	CoverURL       string        `json:"cover_url" gorm:"size:500"`
	PhotoCount     int           `json:"photo_count" gorm:"default:0"`
	// Status 的 default 标签只影响写入：零值空串被 GORM 省略，由库默认值落 draft。
	// 显式写入 published/unpublished 不受影响，因此这里不存在 ParentSetting 那种
	// 「显式 false 被默认值翻回」的问题。
	Status      string     `json:"status" gorm:"size:20;not null;default:draft;index:idx_albums_kg_class_status_date,priority:3"`
	PublishedAt *time.Time `json:"published_at"`
	PublishedBy uint64     `json:"published_by" gorm:"not null;default:0"`
}

func (Album) TableName() string { return "albums" }

// AlbumPhoto 相册照片。file_url/thumb_url 是读取缓存，文件真相源是 files 表；
// file_id 遵循 media-assets 的引用统一字段，历史行（迁移前写入）可能为空。
// 同一文件可同时被成长记录与班级相册引用，可见性按各自引用判定，互不影响。
type AlbumPhoto struct {
	BaseModel
	AlbumID  uint64  `json:"album_id" gorm:"not null;index"`
	FileID   *uint64 `json:"file_id" gorm:"index"`
	FileURL  string  `json:"file_url" gorm:"size:500;not null"`
	ThumbURL string  `json:"thumb_url" gorm:"size:500"`
	Width    int     `json:"width"`
	Height   int     `json:"height"`
	SortNum  int     `json:"sort_num" gorm:"default:0"`
}

func (AlbumPhoto) TableName() string { return "album_photos" }

// AlbumStudent 相册级参与幼儿：批量上传时选的默认参与者，
// 也是照片未逐张标记时「这孩子是否可能出现在本相册」的兜底依据。
type AlbumStudent struct {
	BaseModel
	KindergartenID uint64 `json:"kindergarten_id" gorm:"not null;index"`
	AlbumID        uint64 `json:"album_id" gorm:"not null;uniqueIndex:uk_album_student"`
	StudentID      uint64 `json:"student_id" gorm:"not null;uniqueIndex:uk_album_student;index"`
}

func (AlbumStudent) TableName() string { return "album_students" }

// AlbumPhotoStudent 照片级幼儿关联。存在时以它为准（「我的孩子」照片级过滤）；
// 从成长记录选图时由 growth_log_students 自动带出。
type AlbumPhotoStudent struct {
	BaseModel
	KindergartenID uint64 `json:"kindergarten_id" gorm:"not null;index"`
	PhotoID        uint64 `json:"photo_id" gorm:"not null;uniqueIndex:uk_album_photo_student"`
	StudentID      uint64 `json:"student_id" gorm:"not null;uniqueIndex:uk_album_photo_student;index"`
}

func (AlbumPhotoStudent) TableName() string { return "album_photo_students" }
