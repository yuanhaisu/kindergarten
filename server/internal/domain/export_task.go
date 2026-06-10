package domain

import "time"

// ExportTask 异步导出任务（admin 数据导出中心）
type ExportTask struct {
	BaseModel
	KindergartenID uint64 `json:"kindergarten_id" gorm:"not null;index"`
	PublicID         string `json:"public_id" gorm:"uniqueIndex;size:64;not null"` // 对外 task_id，如 export_a1b2...
	Type             string `json:"type" gorm:"size:32;not null"`                  // attendance | students | growth
	Status           string `json:"status" gorm:"size:32;not null"`                 // processing | completed | failed
	Progress         int    `json:"progress" gorm:"default:0"`
	ParamsJSON       string `json:"params_json" gorm:"type:text"`
	ErrorMessage     string `json:"error_message" gorm:"size:500"`
	LocalPath        string `json:"-" gorm:"size:512"`
	FileName         string `json:"file_name" gorm:"size:255"`
	FileSize         int64  `json:"file_size"`
	ExpireAt         *time.Time `json:"expire_at"`
	CompletedAt      *time.Time `json:"completed_at"`
}

func (ExportTask) TableName() string { return "export_tasks" }
