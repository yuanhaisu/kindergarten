package domain

// HealthCheckStatus 晨检状态
type HealthCheckStatus int8

const (
	HealthCheckPending   HealthCheckStatus = 0 // 待检
	HealthCheckCompleted HealthCheckStatus = 1 // 已完成
	HealthCheckSkipped   HealthCheckStatus = 2 // 已跳过（未到园）
)

// HealthCheckOverall 综合判断
type HealthCheckOverall int8

const (
	HealthOverallHealthy HealthCheckOverall = 0 // 健康
	HealthOverallObserve HealthCheckOverall = 1 // 需观察
	HealthOverallReject  HealthCheckOverall = 2 // 劝返就医
)

// 单项检查值（0=正常，1-3=不同程度异常）
// 喉咙: 0正常 1发红 2红肿
// 手部: 0正常 1红疹 2疱疹 3其他
// 口腔: 0正常 1溃疡 2疱疹 3其他
// 眼睛: 0正常 1发红 2分泌物
// 皮肤: 0正常 1红疹 2水疱 3其他
type HealthCheckItem int8

// MorningHealthCheck 晨检记录
type MorningHealthCheck struct {
	SoftDeleteModel
	KindergartenID uint64              `gorm:"not null;index;column:kindergarten_id" json:"kindergarten_id"`
	ClassID        uint64              `gorm:"not null;index;column:class_id" json:"class_id"`
	StudentID      uint64              `gorm:"not null;index;column:student_id" json:"student_id"`
	PickupRecordID uint64              `gorm:"not null;uniqueIndex;column:pickup_record_id" json:"pickup_record_id"`
	Date           string              `gorm:"type:date;not null;index;column:date" json:"date"`
	CheckStatus    HealthCheckStatus   `gorm:"not null;default:0;column:check_status" json:"check_status"`
	Temperature    *float64            `gorm:"type:decimal(4,1);column:temperature" json:"temperature"`
	ThroatStatus   HealthCheckItem     `gorm:"default:0;column:throat_status" json:"throat_status"`
	HandStatus     HealthCheckItem     `gorm:"default:0;column:hand_status" json:"hand_status"`
	MouthStatus    HealthCheckItem     `gorm:"default:0;column:mouth_status" json:"mouth_status"`
	EyeStatus      HealthCheckItem     `gorm:"default:0;column:eye_status" json:"eye_status"`
	SkinStatus     HealthCheckItem     `gorm:"default:0;column:skin_status" json:"skin_status"`
	OverallStatus  HealthCheckOverall  `gorm:"default:0;column:overall_status" json:"overall_status"`
	IsIsolated     bool                `gorm:"default:false;column:is_isolated" json:"is_isolated"`
	SymptomTags    string              `gorm:"size:500;column:symptom_tags" json:"symptom_tags"`
	Remark         string              `gorm:"size:500;column:remark" json:"remark"`
	OperatorID     *uint64             `gorm:"column:operator_id" json:"operator_id"`
}

func (MorningHealthCheck) TableName() string { return "morning_health_checks" }

// HealthCheckMedia 晨检异常照片
type HealthCheckMedia struct {
	SoftDeleteModel
	CheckID  uint64 `gorm:"not null;index;column:check_id" json:"check_id"`
	FileURL  string `gorm:"size:500;column:file_url" json:"file_url"`
	ThumbURL string `gorm:"size:500;column:thumb_url" json:"thumb_url"`
	Category string `gorm:"size:32;column:category" json:"category"`
	FileName string `gorm:"size:200;column:file_name" json:"file_name"`
	FileSize int64  `gorm:"column:file_size" json:"file_size"`
}

func (HealthCheckMedia) TableName() string { return "health_check_media" }
