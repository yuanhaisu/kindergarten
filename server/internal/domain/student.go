package domain

import "time"

// Student 学生
type Student struct {
	SoftDeleteModel
	KindergartenID uint64    `json:"kindergarten_id" gorm:"not null;index"`
	ClassID        uint64    `json:"class_id" gorm:"not null;index"`
	Name           string    `json:"name" gorm:"size:50;not null"`
	Gender         int8      `json:"gender" gorm:"default:0;comment:0未知 1男 2女"`
	Birthday       time.Time `json:"birthday"`
	Avatar         string    `json:"avatar" gorm:"size:500"`
	StudentNo      string    `json:"student_no" gorm:"size:50;index"`
	IDCardNumber   string    `json:"id_card_number" gorm:"size:32;index"`
	Address        string    `json:"address" gorm:"size:256"`
	HealthNote     string    `json:"health_note" gorm:"size:500"`
	EnrollDate     time.Time `json:"enroll_date"`
	Status         int8      `json:"status" gorm:"default:1;comment:0待入学 1在读 2毕业 3退学"`
	// StatusChangedAt 状态变更时间，用于计算毕业后 90 天只读窗口。
	// 历史数据无值（迁移前毕业/退学的学生），读取端按「已过期」处理，不用 updated_at 猜。
	StatusChangedAt *time.Time `json:"status_changed_at"`

	Kindergarten *Kindergarten `json:"kindergarten,omitempty" gorm:"foreignKey:KindergartenID"`
	Class        *Class        `json:"class,omitempty" gorm:"foreignKey:ClassID"`
}

func (Student) TableName() string { return "students" }

type ParentRelation string

const (
	RelationFather      ParentRelation = "father"
	RelationMother      ParentRelation = "mother"
	RelationGrandfather ParentRelation = "grandfather"
	RelationGrandmother ParentRelation = "grandmother"
	// 母系祖辈与叔伯姨舅：2026-10 接送关系枚举统一为 9 值，中文标签由前端映射。
	RelationMaternalGrandfather ParentRelation = "maternal_grandfather"
	RelationMaternalGrandmother ParentRelation = "maternal_grandmother"
	RelationUncle               ParentRelation = "uncle"
	RelationAunt                ParentRelation = "aunt"
	RelationOther               ParentRelation = "other"
)

// Valid 报告关系码是否属于 9 值合法集合。
//
// 关系码是跨端契约的一部分（pickup 授权人、student_parents），
// 合法性判断只在这里维护一份，避免各 service 各自 switch 漂移。
func (r ParentRelation) Valid() bool {
	switch r {
	case RelationFather, RelationMother,
		RelationGrandfather, RelationGrandmother,
		RelationMaternalGrandfather, RelationMaternalGrandmother,
		RelationUncle, RelationAunt, RelationOther:
		return true
	default:
		return false
	}
}

// StudentParent 学生-家长关联
type StudentParent struct {
	BaseModel
	StudentID uint64         `json:"student_id" gorm:"not null;uniqueIndex:idx_student_parent"`
	UserID    uint64         `json:"user_id" gorm:"not null;uniqueIndex:idx_student_parent"`
	Relation  ParentRelation `json:"relation" gorm:"size:20;not null"`
	IsPrimary bool           `json:"is_primary" gorm:"default:false"`

	Student *Student `json:"student,omitempty" gorm:"foreignKey:StudentID"`
	User    *User    `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

func (StudentParent) TableName() string { return "student_parents" }
