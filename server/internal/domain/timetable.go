package domain

// TimetableTemplate 年级课表模板条目（周模式）。
//
// 课表两层模型（align-class-calendar D5）：
//   - 班级实际课表 = 年级模板（本表）叠加该班微调（TimetableOverride）；
//   - 同步引擎把结果物化为 CalendarEvent（source=timetable、source_slot="星期-节次"）；
//   - 微调只约束周模式；单日调整由老师直接改日历实例（overridden 覆盖标记），不走本表。
//
// 注意：本表刻意不建唯一索引 —— 全量替换采用「软删旧 + 插新」，
// 软删行会占唯一键位（本仓库已知陷阱：软删除行占唯一索引）。
type TimetableTemplate struct {
	SoftDeleteModel
	KindergartenID uint64 `json:"kindergarten_id" gorm:"not null;index:idx_tt_templates_kg_grade,priority:1"`
	Grade          string `json:"grade" gorm:"size:32;not null;index:idx_tt_templates_kg_grade,priority:2;comment:年级（对应 classes.grade）"`
	Weekday        int    `json:"weekday" gorm:"not null;comment:1-7（周一=1）"`
	Period         int    `json:"period" gorm:"not null;comment:第几节（正整数）"`
	Title          string `json:"title" gorm:"size:200;not null"`
	Location       string `json:"location" gorm:"size:200"`
	StartTime      string `json:"start_time" gorm:"size:10"`
	EndTime        string `json:"end_time" gorm:"size:10"`
}

func (TimetableTemplate) TableName() string { return "timetable_templates" }

// TimetableOverride 班级课表微调条目（周模式），覆盖年级模板的同槽位。
// action：replace 替换模板同槽位 / remove 删除模板同槽位 / add 新增模板没有的槽位。
type TimetableOverride struct {
	SoftDeleteModel
	KindergartenID uint64 `json:"kindergarten_id" gorm:"not null;index:idx_tt_overrides_kg_class,priority:1"`
	ClassID        uint64 `json:"class_id" gorm:"not null;index:idx_tt_overrides_kg_class,priority:2"`
	Weekday        int    `json:"weekday" gorm:"not null;comment:1-7（周一=1）"`
	Period         int    `json:"period" gorm:"not null;comment:第几节"`
	Action         string `json:"action" gorm:"size:20;not null;comment:replace 替换 / remove 删除 / add 新增"`
	Title          string `json:"title" gorm:"size:200"`
	Location       string `json:"location" gorm:"size:200"`
	StartTime      string `json:"start_time" gorm:"size:10"`
	EndTime        string `json:"end_time" gorm:"size:10"`
}

func (TimetableOverride) TableName() string { return "timetable_overrides" }
