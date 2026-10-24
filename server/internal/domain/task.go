package domain

import "time"

type TaskFrequency string

const (
	TaskFrequencyDaily  TaskFrequency = "daily"
	TaskFrequencyWeekly TaskFrequency = "weekly"
	TaskFrequencyOnce   TaskFrequency = "once"
)

// TaskScope 星星罐范围。
const (
	TaskStarScopeTask  = "task"
	TaskStarScopeClass = "class"
)

// TaskRitual 老师回应仪式。
const (
	TaskRitualSticker   = "sticker"
	TaskRitualVoice     = "voice"
	TaskRitualClassStar = "class_star"
)

// TaskResponseType 回应类型。
const (
	TaskResponseSingle = "single"
	TaskResponsePeriod = "period"
)

// TaskReaction 家长对回应的反馈取值。
// child_happy 是孩子感知质量信号；received / thanks 是家长确认，统计时必须分开。
const (
	TaskReactionChildHappy = "child_happy"
	TaskReactionReceived   = "received"
	TaskReactionThanks     = "thanks"
)

// ParentTask 亲子任务（老师/园所布置）
type ParentTask struct {
	BaseModel
	KindergartenID    uint64        `json:"kindergarten_id" gorm:"not null;index"`
	ClassID           uint64        `json:"class_id" gorm:"not null;index"`
	ActivityID        string        `json:"activity_id" gorm:"size:64;index"`
	CreatorID         uint64        `json:"creator_id" gorm:"not null"`
	TargetStudentID   uint64        `json:"target_student_id" gorm:"not null;default:0;index"`
	Title             string        `json:"title" gorm:"size:200;not null"`
	Description       string        `json:"description" gorm:"type:text"`
	Intro             string        `json:"intro" gorm:"type:text;comment:任务详细介绍"`
	Requirements      string        `json:"requirements" gorm:"type:text;comment:JSON数组-任务要求"`
	Rewards           string        `json:"rewards" gorm:"type:text;comment:JSON数组-奖励"`
	LearningLink      string        `json:"learning_link" gorm:"type:text;comment:和幼儿园学习的关系"`
	Frequency         TaskFrequency `json:"frequency" gorm:"size:20;not null"`
	StartDate         time.Time     `json:"start_date" gorm:"type:date;not null"`
	EndDate           time.Time     `json:"end_date" gorm:"type:date;not null"`
	AllowPublic       bool          `json:"allow_public" gorm:"not null;default:false"`
	RecognitionRitual string        `json:"recognition_ritual" gorm:"size:20"`
	StarScope         string        `json:"star_scope" gorm:"size:20;not null;default:'task'"`
	StarGoal          int           `json:"star_goal" gorm:"not null;default:0"`
	Status            int8          `json:"status" gorm:"default:1;comment:1进行中 0已结束"`

	// NotifyTargetCount 发布通知的「义务集」大小（按家长去重）。
	// NULL = 未能确定（收件人查询失败，或历史任务未写）；统计把它排除而不是当 0——
	// 「未知」与「确实没有家长」必须可区分（handoff §20.10-6）。
	NotifyTargetCount *int `json:"notify_target_count" gorm:"comment:发布通知义务家长数；NULL=未知"`

	Creator *User  `json:"creator,omitempty" gorm:"foreignKey:CreatorID"`
	Class   *Class `json:"class,omitempty" gorm:"foreignKey:ClassID"`
}

func (ParentTask) TableName() string { return "parent_tasks" }

// TaskCheckIn 任务提交（家长提交）
//
// 唯一约束是 (task_id, student_id, date) 三元组——每个孩子每天每项任务提交一次。
// 三个字段必须都带 uniqueIndex 且同名，GORM 才会生成复合索引。
type TaskCheckIn struct {
	BaseModel
	TaskID               uint64     `json:"task_id" gorm:"not null;index;uniqueIndex:idx_task_student_date;index:idx_task_response_date,priority:1"`
	StudentID            uint64     `json:"student_id" gorm:"not null;index;uniqueIndex:idx_task_student_date"`
	UserID               uint64     `json:"user_id" gorm:"not null"`
	Date                 time.Time  `json:"date" gorm:"type:date;not null;uniqueIndex:idx_task_student_date;index:idx_task_response_date,priority:3"`
	Content              string     `json:"content" gorm:"size:500"`
	MediaFileID          uint64     `json:"media_file_id" gorm:"not null;default:0"`
	MediaURL             string     `json:"media_url" gorm:"size:500"`
	IsPublicRequested    bool       `json:"is_public_requested" gorm:"not null;default:false"`
	PublicScopeConfirmed bool       `json:"public_scope_confirmed" gorm:"not null;default:false"`
	PublicApprovedAt     *time.Time `json:"public_approved_at"`
	PublicApprovedBy     uint64     `json:"public_approved_by" gorm:"not null;default:0"`
	PublicRevokedAt      *time.Time `json:"public_revoked_at"`
	ResponseID           uint64     `json:"response_id" gorm:"not null;default:0;index;index:idx_task_response_date,priority:2"`

	Task    *ParentTask `json:"task,omitempty" gorm:"foreignKey:TaskID"`
	Student *Student    `json:"student,omitempty" gorm:"foreignKey:StudentID"`
}

func (TaskCheckIn) TableName() string { return "task_check_ins" }

// TaskResponse 老师回应；一次回应可覆盖一条或多条提交。
type TaskResponse struct {
	BaseModel
	KindergartenID uint64     `json:"kindergarten_id" gorm:"not null;index"`
	TaskID         uint64     `json:"task_id" gorm:"not null;index"`
	TeacherID      uint64     `json:"teacher_id" gorm:"not null"`
	Ritual         string     `json:"ritual" gorm:"size:20;not null"`
	StickerKey     string     `json:"sticker_key" gorm:"size:50"`
	Content        string     `json:"content" gorm:"size:500"`
	MediaFileID    uint64     `json:"media_file_id" gorm:"not null;default:0"`
	MediaURL       string     `json:"media_url" gorm:"size:500"`
	ResponseType   string     `json:"response_type" gorm:"size:20;not null"`
	PeriodStart    *time.Time `json:"period_start"`
	PeriodEnd      *time.Time `json:"period_end"`

	// NotifyTargetCount 本次主回应通知的「义务集」大小（按家长去重）。
	// NULL = 未知（收件人查询失败 / 补充回应不投递）；反馈率分母从这里来，
	// 不回看 messages 行数（判重命中不写行、失败无行，都会让分母偏小）。
	NotifyTargetCount *int `json:"notify_target_count" gorm:"comment:回应通知义务家长数；NULL=未知"`

	Task *ParentTask `json:"task,omitempty" gorm:"foreignKey:TaskID"`
}

func (TaskResponse) TableName() string { return "task_responses" }

// TaskResponseSupplement 补充回应关系；主回应由 TaskCheckIn.ResponseID 表达。
type TaskResponseSupplement struct {
	BaseModel
	KindergartenID uint64 `json:"kindergarten_id" gorm:"not null;index"`
	ResponseID     uint64 `json:"response_id" gorm:"not null;uniqueIndex:idx_task_response_supplement"`
	CheckInID      uint64 `json:"check_in_id" gorm:"not null;uniqueIndex:idx_task_response_supplement;index"`
}

func (TaskResponseSupplement) TableName() string { return "task_response_supplements" }

// TaskResponseReaction 家长对回应的反馈，按 (response_id, user_id) 唯一。
type TaskResponseReaction struct {
	BaseModel
	KindergartenID uint64 `json:"kindergarten_id" gorm:"not null;index"`
	ResponseID     uint64 `json:"response_id" gorm:"not null;uniqueIndex:idx_task_response_reaction"`
	UserID         uint64 `json:"user_id" gorm:"not null;uniqueIndex:idx_task_response_reaction"`
	Reaction       string `json:"reaction" gorm:"size:20;not null"`
}

func (TaskResponseReaction) TableName() string { return "task_response_reactions" }

// TaskCheckInState 频率串行化状态行。
type TaskCheckInState struct {
	BaseModel
	TaskID          uint64     `json:"task_id" gorm:"not null;uniqueIndex:idx_task_checkin_state"`
	StudentID       uint64     `json:"student_id" gorm:"not null;uniqueIndex:idx_task_checkin_state"`
	LastCheckInDate *time.Time `json:"last_checkin_date" gorm:"type:date"`
}

func (TaskCheckInState) TableName() string { return "task_checkin_states" }

// ClassStarJar 班级星星罐目标；按班级 + 学期唯一。
type ClassStarJar struct {
	BaseModel
	KindergartenID uint64 `json:"kindergarten_id" gorm:"not null;uniqueIndex:idx_class_star_jar"`
	ClassID        uint64 `json:"class_id" gorm:"not null;uniqueIndex:idx_class_star_jar"`
	Term           string `json:"term" gorm:"size:32;not null;uniqueIndex:idx_class_star_jar"`
	Goal           int    `json:"goal" gorm:"not null;default:0"`
	CreatedBy      uint64 `json:"created_by" gorm:"not null;default:0"`
	UpdatedBy      uint64 `json:"updated_by" gorm:"not null;default:0"`
}

func (ClassStarJar) TableName() string { return "class_star_jars" }

// TaskReadMark 家长对某任务/某孩子的读标记，用于补充回应未读。
type TaskReadMark struct {
	BaseModel
	KindergartenID uint64     `json:"kindergarten_id" gorm:"not null;index"`
	TaskID         uint64     `json:"task_id" gorm:"not null;uniqueIndex:idx_task_read_mark"`
	UserID         uint64     `json:"user_id" gorm:"not null;uniqueIndex:idx_task_read_mark"`
	StudentID      uint64     `json:"student_id" gorm:"not null;uniqueIndex:idx_task_read_mark"`
	LastReadAt     *time.Time `json:"last_read_at"`
}

func (TaskReadMark) TableName() string { return "task_read_marks" }
