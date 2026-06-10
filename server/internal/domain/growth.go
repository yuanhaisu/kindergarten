package domain

import "time"

type MealRating int8

const (
	MealRatingGood   MealRating = 3
	MealRatingNormal MealRating = 2
	MealRatingPoor   MealRating = 1
)

type SleepQuality int8

const (
	SleepQualityGood   SleepQuality = 3
	SleepQualityNormal SleepQuality = 2
	SleepQualityPoor   SleepQuality = 1
)

type MoodRating int8

const (
	MoodHappy  MoodRating = 3
	MoodNormal MoodRating = 2
	MoodSad    MoodRating = 1
)

// GrowthDailyLog 每日成长日志（老师填写）
type GrowthDailyLog struct {
	SoftDeleteModel
	KindergartenID uint64    `json:"kindergarten_id" gorm:"not null;index"`
	ClassID        uint64    `json:"class_id" gorm:"not null;index"`
	CreatorID      uint64    `json:"creator_id" gorm:"not null"`
	Date           time.Time `json:"date" gorm:"type:date;not null;index"`

	BreakfastRating MealRating `json:"breakfast_rating" gorm:"default:0"`
	LunchRating     MealRating `json:"lunch_rating" gorm:"default:0"`
	DinnerRating    MealRating `json:"dinner_rating" gorm:"default:0"`

	NapDuration  int          `json:"nap_duration"`
	SleepQuality SleepQuality `json:"sleep_quality" gorm:"default:0"`

	MorningMood   MoodRating `json:"morning_mood" gorm:"default:0"`
	AfternoonMood MoodRating `json:"afternoon_mood" gorm:"default:0"`

	PerformanceTags string `json:"performance_tags" gorm:"size:500"`

	Content    string `json:"content" gorm:"type:text"`
	SyncToFeed bool   `json:"sync_to_feed" gorm:"default:true"`
	TargetType int8   `json:"target_type" gorm:"default:1;comment:1全班 2指定学生"`

	Creator *User  `json:"creator,omitempty" gorm:"foreignKey:CreatorID"`
	Class   *Class `json:"class,omitempty" gorm:"foreignKey:ClassID"`
}

func (GrowthDailyLog) TableName() string { return "growth_daily_logs" }

// GrowthLogStudent 成长日志-学生关联（指定发送时使用）
type GrowthLogStudent struct {
	BaseModel
	LogID     uint64 `json:"log_id" gorm:"not null;uniqueIndex:idx_log_student"`
	StudentID uint64 `json:"student_id" gorm:"not null;uniqueIndex:idx_log_student"`
}

func (GrowthLogStudent) TableName() string { return "growth_log_students" }

// GrowthMedia 成长记录媒体文件
type GrowthMedia struct {
	BaseModel
	KindergartenID uint64  `json:"kindergarten_id" gorm:"not null;index"`
	ClassID        uint64  `json:"class_id" gorm:"not null;index"`
	LogID          *uint64 `json:"log_id,omitempty" gorm:"index"`
	UploaderID     uint64  `json:"uploader_id" gorm:"not null"`
	FileURL        string  `json:"file_url" gorm:"size:500;not null"`
	ThumbURL       string  `json:"thumb_url" gorm:"size:500"`
	FileType       string  `json:"file_type" gorm:"size:20;not null;comment:image/video"`
	FileName       string  `json:"file_name" gorm:"size:200"`
	FileSize       int64   `json:"file_size"`
	Duration       int     `json:"duration"`
	SortNum        int     `json:"sort_num" gorm:"default:0"`
}

func (GrowthMedia) TableName() string { return "growth_media" }

// GrowthMediaStudent 媒体-学生关联（AI 识别或手动标记）
type GrowthMediaStudent struct {
	BaseModel
	MediaID   uint64 `json:"media_id" gorm:"not null;uniqueIndex:idx_media_student"`
	StudentID uint64 `json:"student_id" gorm:"not null;uniqueIndex:idx_media_student"`
	IsAuto    bool   `json:"is_auto" gorm:"default:false"`
}

func (GrowthMediaStudent) TableName() string { return "growth_media_students" }

// GrowthComment 成长记录评论（家长留言）
type GrowthComment struct {
	SoftDeleteModel
	LogID    uint64  `json:"log_id" gorm:"not null;index"`
	UserID   uint64  `json:"user_id" gorm:"not null;index"`
	Content  string  `json:"content" gorm:"size:500;not null"`
	ParentID *uint64 `json:"parent_id,omitempty" gorm:"index"`

	User *User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

func (GrowthComment) TableName() string { return "growth_comments" }

// GrowthLike 成长记录点赞
type GrowthLike struct {
	BaseModel
	LogID  uint64 `json:"log_id" gorm:"not null;uniqueIndex:idx_log_user"`
	UserID uint64 `json:"user_id" gorm:"not null;uniqueIndex:idx_log_user"`
}

func (GrowthLike) TableName() string { return "growth_likes" }

// GrowthReport AI 成长报告
type GrowthReport struct {
	SoftDeleteModel
	KindergartenID uint64    `json:"kindergarten_id" gorm:"not null;index"`
	StudentID      uint64    `json:"student_id" gorm:"not null;index"`
	ReportType     string    `json:"report_type" gorm:"size:20;not null;comment:weekly/monthly"`
	PeriodStart    time.Time `json:"period_start" gorm:"type:date;not null"`
	PeriodEnd      time.Time `json:"period_end" gorm:"type:date;not null"`
	Content        string    `json:"content" gorm:"type:text"`
	Highlights     string    `json:"highlights" gorm:"type:text"`
	Concerns       string    `json:"concerns" gorm:"type:text"`
	GeneratedAt    time.Time `json:"generated_at"`
	Status         int8      `json:"status" gorm:"default:1;comment:1已生成 0生成中 -1失败"`

	Student *Student `json:"student,omitempty" gorm:"foreignKey:StudentID"`
}

func (GrowthReport) TableName() string { return "growth_reports" }

// GrowthReportDimension 报告维度评分（对应前端雷达图各维度）
type GrowthReportDimension struct {
	BaseModel
	ReportID uint64  `json:"report_id" gorm:"not null;uniqueIndex:idx_report_dim"`
	DimName  string  `json:"dim_name" gorm:"size:20;not null;uniqueIndex:idx_report_dim;comment:语言/认知/社交/运动/艺术/自理/情绪"`
	Score    float64 `json:"score" gorm:"not null"`
	MaxScore float64 `json:"max_score" gorm:"default:100"`
	Comment  string  `json:"comment" gorm:"size:200"`
}

func (GrowthReportDimension) TableName() string { return "growth_report_dimensions" }
