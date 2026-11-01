package domain

import "time"

// MealPlan 周食谱（计划食谱）。
//
// 名称在同一学年内唯一，由服务层保证（不建唯一索引，避免软删除行占键）。
// ConfigSnapshot 在创建时固化：餐次/占比/口径 + 按班级人数加权后的 11 项 DRIs 目标值。
// 后续修改供餐配置或学生名册都不得改写它（design D3、TC-NEW-DIE-0008）。
type MealPlan struct {
	SoftDeleteModel
	KindergartenID uint64                `gorm:"not null;index:idx_diet_plan_kg_year,priority:1" json:"kindergarten_id"`
	AcademicYearID uint64                `gorm:"not null;index:idx_diet_plan_kg_year,priority:2" json:"academic_year_id"`
	Name           string                `gorm:"size:128;not null;index" json:"name"`
	StartDate      time.Time             `gorm:"type:date;not null" json:"start_date"`
	EndDate        time.Time             `gorm:"type:date;not null" json:"end_date"`
	DayNum         int                   `gorm:"not null;default:0" json:"day_num"` // 有明细的不同日期数
	WeekOfYear     int                   `gorm:"not null;default:0" json:"week_of_year"`
	CreatedMethod  DietPlanCreatedMethod `gorm:"size:20;not null;default:custom" json:"created_method"`
	ConfigSnapshot string                `gorm:"type:json;not null" json:"config_snapshot"`

	// 来源模板/来源食谱只做追溯，不建立级联关系；模板或源食谱删除不影响本食谱。
	SourceTemplateID *uint64 `gorm:"default:null" json:"source_template_id,omitempty"`
	SourcePlanID     *uint64 `gorm:"default:null" json:"source_plan_id,omitempty"`

	Items []MealPlanItem `gorm:"foreignKey:PlanID" json:"items,omitempty"`
}

func (MealPlan) TableName() string { return "diet_meal_plans" }

// MealPlanItem 周食谱明细：一行 = 某天 × 某餐次 × 某道菜里的某种食物。
//
// FoodName 是菜名/食物名的历史快照：食物被删除后历史食谱仍要可读。
// NutritionSnapshot 是录入当时的每 100g 可食部浓度快照，报表只用它计算，不回连食物库（ADR-0006）。
type MealPlanItem struct {
	BaseModel
	KindergartenID    uint64      `gorm:"not null;index" json:"kindergarten_id"`
	PlanID            uint64      `gorm:"not null;index:idx_diet_plan_item,priority:1" json:"plan_id"`
	Date              time.Time   `gorm:"type:date;not null;index:idx_diet_plan_item,priority:2" json:"date"`
	Weekday           int         `gorm:"not null;default:0" json:"weekday"` // 1=周一 … 7=周日
	MealKey           DietMealKey `gorm:"size:20;not null;index:idx_diet_plan_item,priority:3" json:"meal_key"`
	RecipeName        string      `gorm:"size:128;not null" json:"recipe_name"`
	RecipeID          *uint64     `gorm:"default:null" json:"recipe_id,omitempty"`
	FoodID            *uint64     `gorm:"default:null;index" json:"food_id,omitempty"`
	FoodName          string      `gorm:"size:128;not null" json:"food_name"`
	Grams             float64     `gorm:"not null;default:0" json:"grams"`
	NutritionSnapshot string      `gorm:"type:json;not null" json:"nutrition_snapshot"`
	Sort              int         `gorm:"not null;default:0" json:"sort"`
}

func (MealPlanItem) TableName() string { return "diet_meal_plan_items" }

// TemplatePlan 周食谱模板（按园种子化的只读副本）。
//
// 模板由平台初稿在食物库就绪后复制到各园，条目引用本园 food_id，因此不跨租户引用。
// TemplateKey 是幂等种子键：同一园所同一模板重复播种时按它判重。
type TemplatePlan struct {
	BaseModel
	KindergartenID uint64 `gorm:"not null;index:idx_diet_template_kg_key,priority:1" json:"kindergarten_id"`
	TemplateKey    string `gorm:"size:64;not null;default:'';index:idx_diet_template_kg_key,priority:2" json:"template_key"`
	Name           string `gorm:"size:128;not null" json:"name"`
	DayNum         int    `gorm:"not null;default:0" json:"day_num"`
	MealKeys       string `gorm:"size:255;not null;default:''" json:"meal_keys"` // 逗号分隔的 meal_key 集合
	Sort           int    `gorm:"not null;default:0" json:"sort"`

	Items []TemplatePlanItem `gorm:"foreignKey:TemplateID" json:"items,omitempty"`
}

func (TemplatePlan) TableName() string { return "diet_template_plans" }

// TemplatePlanItem 模板条目。只引用本园食物；食物缺失时复制流程跳过并在报告中列出（design D10）。
// FoodName 是食物名的快照：食物被删除后模板详情仍要能渲染名称（与 MealPlanItem 同理）。
type TemplatePlanItem struct {
	BaseModel
	KindergartenID uint64      `gorm:"not null;index" json:"kindergarten_id"`
	TemplateID     uint64      `gorm:"not null;index" json:"template_id"`
	Weekday        int         `gorm:"not null;default:0" json:"weekday"` // 1=周一 … 7=周日
	MealKey        DietMealKey `gorm:"size:20;not null" json:"meal_key"`
	RecipeName     string      `gorm:"size:128;not null" json:"recipe_name"`
	FoodID         uint64      `gorm:"not null;index" json:"food_id"`
	FoodName       string      `gorm:"size:128;not null" json:"food_name"`
	Grams          float64     `gorm:"not null;default:0" json:"grams"`
	Sort           int         `gorm:"not null;default:0" json:"sort"`
}

func (TemplatePlanItem) TableName() string { return "diet_template_plan_items" }
