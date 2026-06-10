package domain

// Feedback 家长意见反馈
type Feedback struct {
	SoftDeleteModel
	KindergartenID uint64 `json:"kindergarten_id" gorm:"not null;index"`
	UserID         uint64 `json:"user_id" gorm:"not null;index"`
	Category       string `json:"category" gorm:"size:20;not null;comment:suggestion/complaint/bug"`
	Content        string `json:"content" gorm:"size:1000;not null"`
	Contact        string `json:"contact" gorm:"size:50"`
	Images         string `json:"images" gorm:"size:1000;comment:JSON array of URLs"`
	Status         int8   `json:"status" gorm:"default:0;comment:0待处理 1已回复 2已关闭"`
	Reply          string `json:"reply" gorm:"size:500"`
	RepliedAt      *string `json:"replied_at,omitempty"`

	User *User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

func (Feedback) TableName() string { return "feedbacks" }
