package domain

// MealConfig 园所级供餐配置（单例）。
//
// 一个园所只允许一条记录：保存时按 kindergarten_id 更新，不新增第二条。
// 餐次结构变化只影响此后新建的周食谱；已保存食谱读自己的 config_snapshot（design D3）。
type MealConfig struct {
	BaseModel
	KindergartenID uint64            `gorm:"not null;uniqueIndex:idx_diet_meal_config_kg" json:"kindergarten_id"`
	CareType       DietCareType      `gorm:"size:20;not null;default:day_care" json:"care_type"`
	FoodInputType  DietFoodInputType `gorm:"size:20;not null;default:market_weight" json:"food_input_type"`

	Meals []MealConfigMeal `gorm:"foreignKey:ConfigID" json:"meals,omitempty"`
}

func (MealConfig) TableName() string { return "diet_meal_configs" }

// MealConfigMeal 供餐配置下的餐次。
//
// (config_id, meal_key) 唯一。注意该表带软删除列：整组保存必须按 meal_key 就地更新，
// 不能「软删旧行 + 插入新行」——软删除行仍占用唯一索引，第二次保存会撞键。
type MealConfigMeal struct {
	BaseModel
	KindergartenID uint64      `gorm:"not null;index" json:"kindergarten_id"`
	ConfigID       uint64      `gorm:"not null;index;uniqueIndex:idx_diet_meal_config_meal,priority:1" json:"config_id"`
	MealKey        DietMealKey `gorm:"size:20;not null;uniqueIndex:idx_diet_meal_config_meal,priority:2" json:"meal_key"`
	Name           string      `gorm:"size:32;not null" json:"name"`
	Time           string      `gorm:"size:5;not null;default:''" json:"time"` // HH:mm
	EnergyPercent  float64     `gorm:"not null;default:0" json:"energy_percent"`
	Sort           int         `gorm:"not null;default:0" json:"sort"`
	// Enabled 不能带 default:true：GORM 在 Create 路径会把零值 false 替换成默认值，
	// 「停用晚餐」与默认 0.95 系数都会写不进去。服务层始终显式赋值，无需数据库默认值。
	Enabled bool `gorm:"not null" json:"enabled"`

	Config *MealConfig `gorm:"foreignKey:ConfigID" json:"-"`
}

func (MealConfigMeal) TableName() string { return "diet_meal_config_meals" }

// FoodCategory 食物分类（全局只读两级树）。
//
// 不放 sys_dict：分类需要稳定的层级 ID 与食物外键，且全平台共用一套（design D2）。
// 平台级数据，没有 kindergarten_id；启动时只种一次。唯一索引依赖「只种不删」，
// 如确需重建必须硬删（Unscoped），软删除行会占键。
type FoodCategory struct {
	BaseModel
	ParentID uint64 `gorm:"not null;default:0;index;uniqueIndex:idx_diet_food_category_parent_name,priority:1" json:"parent_id"`
	Level    int    `gorm:"not null;default:1" json:"level"` // 1=一级，2=二级
	Name     string `gorm:"size:64;not null;uniqueIndex:idx_diet_food_category_parent_name,priority:2" json:"name"`
	Sort     int    `gorm:"not null;default:0" json:"sort"`
}

func (FoodCategory) TableName() string { return "diet_food_categories" }

// DriAgeTarget 分年龄档的 DRIs 目标值（全局只读）。
//
// 3/4/5/6 岁 × 11 项 = 44 行。11 项是能量、蛋白质、脂肪、碳水化合物、维生素A、B1、B2、C、
// 钙、锌、铁；脂肪/碳水由能量与蛋白派生，千焦由千卡 ×4.186 在报表侧推导，
// 视黄醇/胡萝卜素不进目标表（页面不展示，design D6）。
// 平台级数据，没有 kindergarten_id；启动时只种一次。唯一索引同样依赖「只种不删」。
type DriAgeTarget struct {
	BaseModel
	AgeBand     int     `gorm:"not null;uniqueIndex:idx_diet_dri_age_nutrient,priority:1" json:"age_band"` // 3/4/5/6
	NutrientKey string  `gorm:"size:32;not null;uniqueIndex:idx_diet_dri_age_nutrient,priority:2" json:"nutrient_key"`
	Label       string  `gorm:"size:32;not null" json:"label"`
	Unit        string  `gorm:"size:16;not null" json:"unit"`
	Value       float64 `gorm:"not null;default:0" json:"value"`
	Sort        int     `gorm:"not null;default:0" json:"sort"`
}

func (DriAgeTarget) TableName() string { return "diet_dri_age_targets" }
