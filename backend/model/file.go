package model

// FileRecord 统一文件记录
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
