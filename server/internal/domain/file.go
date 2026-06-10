package domain

// File 上传文件（小程序端使用）
type File struct {
	BaseModel
	UserID   uint64 `gorm:"index;not null" json:"-"`
	BizType  string `gorm:"size:32;index;not null" json:"-"`
	FileName string `gorm:"size:256" json:"file_name"`
	FileURL  string `gorm:"size:512" json:"file_url"`
	ThumbURL string `gorm:"size:512" json:"thumb_url,omitempty"`
	FileSize int64  `json:"file_size"`
	FileType string `gorm:"size:32" json:"file_type"`
	Duration int    `gorm:"default:0" json:"duration"`

	User *User `json:"-" gorm:"foreignKey:UserID"`
}

func (File) TableName() string { return "files" }

// FileRecord 统一文件记录（管理后台使用）
type FileRecord struct {
	BaseModel
	KindergartenID uint64 `json:"kindergarten_id" gorm:"not null;index"`
	UploaderID     uint64 `json:"uploader_id" gorm:"not null"`
	FileName       string `json:"file_name" gorm:"size:200;not null"`
	FileURL        string `json:"file_url" gorm:"size:500;not null"`
	FileType       string `json:"file_type" gorm:"size:50"`
	FileSize       int64  `json:"file_size"`
	BizType        string `json:"biz_type" gorm:"size:50;index"`
	BizID          uint64 `json:"biz_id" gorm:"index"`
}

func (FileRecord) TableName() string { return "file_records" }
