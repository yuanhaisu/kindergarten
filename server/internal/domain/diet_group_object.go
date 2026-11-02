package domain

// 按学年管理的膳食集体对象（openspec: add-diet-group-objects, design D1/D2）。
//
// 旧单例表 diet_meal_configs 在 kindergarten_id 上有唯一索引，无法靠 AutoMigrate 删除，
// 复用会让「一园一学年多条对象」永远撞键；因此对象、餐次、就餐班级各开新表。
// 旧单例表保留在库中但不再读写。

// DietGroupObject 是供餐配置的唯一边界：一个园所的一个学年可以有多条对象。
//
// 名称唯一由服务层校验（同园同学年）；不建唯一索引——软删除行仍占键，且本变更不提供删除
// （design D1）。就餐班级与餐次是整组保存的子表，见下面两个关联模型。
type DietGroupObject struct {
	SoftDeleteModel
	KindergartenID uint64            `gorm:"not null;index:idx_diet_group_object_kg_year,priority:1" json:"kindergarten_id"`
	AcademicYearID uint64            `gorm:"not null;index:idx_diet_group_object_kg_year,priority:2" json:"academic_year_id"`
	Name           string            `gorm:"size:64;not null" json:"name"`
	CareType       DietCareType      `gorm:"size:20;not null;default:day_care" json:"care_type"`
	FoodInputType  DietFoodInputType `gorm:"size:20;not null;default:market_weight" json:"food_input_type"`
	RatioMode      DietRatioMode     `gorm:"size:20;not null;default:standard" json:"ratio_mode"`
	// PolicyName 是膳食营养素参考摄入量标准；本变更固定 DRIs2000（design D5）。
	PolicyName string `gorm:"size:32;not null;default:DRIs2000" json:"policy_name"`

	Meals   []DietGroupObjectMeal  `gorm:"foreignKey:GroupObjectID" json:"meals,omitempty"`
	Classes []DietGroupObjectClass `gorm:"foreignKey:GroupObjectID" json:"classes,omitempty"`
}

func (DietGroupObject) TableName() string { return "diet_group_objects" }

// DietGroupObjectMeal 集体对象下的餐次。
//
// (group_object_id, meal_key) 唯一。与 MealConfigMeal 一样带软删除列：整组保存必须按
// meal_key 就地 upsert，删除的餐次用 Unscoped() 硬删——软删除行仍占用唯一索引。
type DietGroupObjectMeal struct {
	BaseModel
	KindergartenID uint64      `gorm:"not null;index" json:"kindergarten_id"`
	GroupObjectID  uint64      `gorm:"not null;index;uniqueIndex:idx_diet_group_object_meal,priority:1" json:"group_object_id"`
	MealKey        DietMealKey `gorm:"size:20;not null;uniqueIndex:idx_diet_group_object_meal,priority:2" json:"meal_key"`
	Name           string      `gorm:"size:32;not null" json:"name"`
	Time           string      `gorm:"size:5;not null;default:''" json:"time"` // HH:mm
	EnergyPercent  float64     `gorm:"not null;default:0" json:"energy_percent"`
	Sort           int         `gorm:"not null;default:0" json:"sort"`
	// Enabled 不能带 default:true：GORM 在 Create 路径会把零值 false 替换成默认值，
	// 「停用晚餐」会写不进去。服务层始终显式赋值（同 MealConfigMeal）。
	Enabled bool `gorm:"not null" json:"enabled"`

	GroupObject *DietGroupObject `gorm:"foreignKey:GroupObjectID" json:"-"`
}

func (DietGroupObjectMeal) TableName() string { return "diet_group_object_meals" }

// DietGroupObjectClass 集体对象的就餐班级。
//
// (group_object_id, class_id) 唯一。班级是动态数据：只存 ID，名称与年级由读取侧 join 班级表。
type DietGroupObjectClass struct {
	BaseModel
	KindergartenID uint64 `gorm:"not null;index" json:"kindergarten_id"`
	GroupObjectID  uint64 `gorm:"not null;index;uniqueIndex:idx_diet_group_object_class,priority:1" json:"group_object_id"`
	ClassID        uint64 `gorm:"not null;uniqueIndex:idx_diet_group_object_class,priority:2" json:"class_id"`

	GroupObject *DietGroupObject `gorm:"foreignKey:GroupObjectID" json:"-"`
}

func (DietGroupObjectClass) TableName() string { return "diet_group_object_classes" }
