package model

// OperationLog 操作日志（管理后台审计）
type OperationLog struct {
	BaseModel
	KindergartenID uint64 `json:"kindergarten_id" gorm:"not null;index"`
	UserID         uint64 `json:"user_id" gorm:"not null;index"`
	Module         string `json:"module" gorm:"size:50"`
	Action         string `json:"action" gorm:"size:50"`
	TargetType     string `json:"target_type" gorm:"size:50"`
	TargetID       uint64 `json:"target_id"`
	Detail         string `json:"detail" gorm:"type:text"`
	IP             string `json:"ip" gorm:"size:50"`
	UserAgent      string `json:"user_agent" gorm:"size:300"`
}
