package domain

// StudentFace 学生人脸底库照片。
// 照片经统一上传通道存 OSS，本表记录其云端底库注册状态。
// PersonRef 为云侧人员标识（腾讯云 PersonId，形如 kg{id}_stu{sid}）——我方不存人脸特征值。
type StudentFace struct {
	SoftDeleteModel
	StudentID  uint64 `json:"student_id" gorm:"not null;uniqueIndex:idx_student_face"`
	FaceURL    string `json:"face_url" gorm:"size:500;not null;uniqueIndex:idx_student_face"`
	Provider   string `json:"provider" gorm:"size:20;not null;default:tencent"`
	PersonRef  string `json:"person_ref" gorm:"size:64"`
	Status     int8   `json:"status" gorm:"default:0;index;comment:0待注册 1已注册 2注册失败"`
	LastError  string `json:"last_error" gorm:"size:255"`
}

func (StudentFace) TableName() string { return "student_faces" }

// 底库注册状态
const (
	FaceStatusPending  int8 = 0 // 待注册
	FaceStatusIndexed  int8 = 1 // 已注册（可被识别命中）
	FaceStatusFailed   int8 = 2 // 注册失败（可重试）
)

// RecognitionJob 照片人脸识别任务（异步队列，设计 D2：DB 表驱动轮询）。
// 每张待识别照片一条；worker 以原子 CAS 认领。
type RecognitionJob struct {
	BaseModel
	MediaID        uint64 `json:"media_id" gorm:"not null;uniqueIndex"` // 一张照片一条（唯一，防重复入队）
	KindergartenID uint64 `json:"kindergarten_id" gorm:"not null;index"`
	Status         string `json:"status" gorm:"size:16;not null;default:pending;index:idx_job_claim,priority:1"` // pending/processing/done/failed
	Retries        int    `json:"retries" gorm:"default:0"`
	LastError      string `json:"last_error" gorm:"size:255"`
	WorkerID       string `json:"worker_id" gorm:"size:64"`
	NextRunAt      int64  `json:"next_run_at" gorm:"default:0;index:idx_job_claim,priority:2"` // Unix 秒；退避重试用；与 status 组成认领查询复合索引
	// FaceCount 识别时检测到的人脸数（D11 照片分型）：1=单人照，≥阈值=大合影（不产建议）。
	// 受供应商 MaxFaceNum 上限约束（腾讯云 10，触顶即"≥10"）。
	FaceCount int `json:"face_count" gorm:"default:0"`
}

func (RecognitionJob) TableName() string { return "recognition_jobs" }

// 识别任务状态
const (
	JobStatusPending    = "pending"
	JobStatusProcessing = "processing"
	JobStatusDone       = "done"
	JobStatusFailed     = "failed"
)
