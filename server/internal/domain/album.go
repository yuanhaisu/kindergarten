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

// Album 班级相册。封面与照片数在创建时冗余写入，列表无需 join album_photos。
// 本期只有创建与读取，没有编辑/删除接口，因此冗余值不存在漂移来源。
type Album struct {
	SoftDeleteModel
	KindergartenID uint64        `json:"kindergarten_id" gorm:"not null;index"`
	ClassID        uint64        `json:"class_id" gorm:"not null;index"`
	CreatorID      uint64        `json:"creator_id" gorm:"not null;index"`
	Title          string        `json:"title" gorm:"size:30;not null"`
	Category       AlbumCategory `json:"category" gorm:"size:20;not null;index"`
	AlbumDate      time.Time     `json:"album_date" gorm:"type:date;not null;index"`
	CoverURL       string        `json:"cover_url" gorm:"size:500"`
	PhotoCount     int           `json:"photo_count" gorm:"default:0"`
}

func (Album) TableName() string { return "albums" }

// AlbumPhoto 相册照片。只存展示 URL 与尺寸；原文件记录由统一上传通道维护。
// 本期没有删除入口；将来删相册时按 album_id 硬删（随相册一起消失，不做软删）。
type AlbumPhoto struct {
	BaseModel
	AlbumID  uint64 `json:"album_id" gorm:"not null;index"`
	FileURL  string `json:"file_url" gorm:"size:500;not null"`
	ThumbURL string `json:"thumb_url" gorm:"size:500"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	SortNum  int    `json:"sort_num" gorm:"default:0"`
}

func (AlbumPhoto) TableName() string { return "album_photos" }
