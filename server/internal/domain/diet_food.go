package domain

// Food 食物库条目。
//
// 多租户语义（ADR-0005）：
//   - kindergarten_id = 0 是平台预置行（哨兵），>0 是园所副本；
//   - 新园初始化时把 kg=0 行复制一份到本园，此后园所只改自己的副本；
//   - 仓储层默认强制 kindergarten_id 过滤，禁止把 kg=0 行当作可编辑数据回传。
//
// 27 项营养是「每 100g 可食部」的浓度值。全部用指针：NULL = 未测定/不适用，
// 任何求和都必须跳过 NULL，不能当 0（否则 VA/VC/硒会被系统性低估）。
type Food struct {
	SoftDeleteModel
	KindergartenID uint64         `gorm:"not null;index;index:idx_diet_food_preset,priority:1" json:"kindergarten_id"`
	Name           string         `gorm:"size:128;not null;index" json:"name"`
	SpellCode      string         `gorm:"size:64;not null;default:'';index" json:"spell_code"`
	CategoryID     uint64         `gorm:"not null;index" json:"category_id"` // 叶子分类
	EdiblePercent  float64        `gorm:"not null;default:0" json:"edible_percent"`
	Source         DietFoodSource `gorm:"size:20;not null;default:system" json:"source"`

	// Group12 沿用参考产品 37–48 的采购分类 ID；NULL 表示无槽位（薯类/饮料/糖蜜饯）。
	Group12 *int `gorm:"default:null" json:"group12,omitempty"`
	// Group16 是膳食构成表的 16 行分类，由 20 个一级类合并而来。
	Group16 DietFoodGroup16 `gorm:"size:32;not null;default:''" json:"group16"`
	// SourceFlags 逗号分隔的动物性/植物性/乳类/肉类标记，供蛋白质/脂肪/钙铁来源报表做集合运算。
	SourceFlags string `gorm:"size:128;not null;default:''" json:"source_flags"`

	// IsCustomized 标记园所是否改过预置行；平台升级只跳过这些行。内部字段，不对外返回。
	IsCustomized bool `gorm:"not null;default:false" json:"-"`
	// PresetKey 是预置数据的稳定业务键，按 (kindergarten_id, preset_key) 幂等复制/升级。
	// 不建唯一索引：软删除行会占键，判重放在服务层。内部字段，不对外返回。
	PresetKey string `gorm:"size:128;not null;default:'';index:idx_diet_food_preset,priority:2" json:"-"`

	// 27 项营养（每 100g 可食部）。
	Energy        *float64 `gorm:"column:energy" json:"energy"`                 // 千卡
	Heat          *float64 `gorm:"column:heat" json:"heat"`                     // 千焦
	Protein       *float64 `gorm:"column:protein" json:"protein"`               // 克
	Fats          *float64 `gorm:"column:fats" json:"fats"`                     // 克
	Carb          *float64 `gorm:"column:carb" json:"carb"`                     // 克
	DietaryFiber  *float64 `gorm:"column:dietary_fiber" json:"dietary_fiber"`   // 克
	Ash           *float64 `gorm:"column:ash" json:"ash"`                       // 克
	Water         *float64 `gorm:"column:water" json:"water"`                   // 克
	Cholesterol   *float64 `gorm:"column:cholesterol" json:"cholesterol"`       // 毫克
	Retinol       *float64 `gorm:"column:retinol" json:"retinol"`               // 微克（仅食物维护，不进报表）
	Carotin       *float64 `gorm:"column:carotin" json:"carotin"`               // 微克（仅食物维护，不进报表）
	VitaminA      *float64 `gorm:"column:vitamin_a" json:"vitamin_a"`           // 微克 RE
	VitaminB1     *float64 `gorm:"column:vitamin_b1" json:"vitamin_b1"`         // 毫克
	VitaminB2     *float64 `gorm:"column:vitamin_b2" json:"vitamin_b2"`         // 毫克
	NicotinicAcid *float64 `gorm:"column:nicotinic_acid" json:"nicotinic_acid"` // 毫克
	VitaminC      *float64 `gorm:"column:vitamin_c" json:"vitamin_c"`           // 毫克
	VitaminE      *float64 `gorm:"column:vitamin_e" json:"vitamin_e"`           // 毫克 α-TE
	Calcium       *float64 `gorm:"column:calcium" json:"calcium"`               // 毫克
	Phosphorus    *float64 `gorm:"column:phosphorus" json:"phosphorus"`         // 毫克
	Potassium     *float64 `gorm:"column:potassium" json:"potassium"`           // 毫克
	Sodium        *float64 `gorm:"column:sodium" json:"sodium"`                 // 毫克
	Magnesium     *float64 `gorm:"column:magnesium" json:"magnesium"`           // 毫克
	Iron          *float64 `gorm:"column:iron" json:"iron"`                     // 毫克
	Zinc          *float64 `gorm:"column:zinc" json:"zinc"`                     // 毫克
	Selenium      *float64 `gorm:"column:selenium" json:"selenium"`             // 微克
	Copper        *float64 `gorm:"column:copper" json:"copper"`                 // 毫克
	Manganese     *float64 `gorm:"column:manganese" json:"manganese"`           // 毫克
}

func (Food) TableName() string { return "diet_foods" }

// NutritionSnapshot 把食物当前的食部与 27 项浓度复制成快照。
// 只在周食谱保存/复制时调用；菜谱库不存快照（ADR-0006）。
func (f Food) NutritionSnapshot() DietNutritionSnapshot {
	return DietNutritionSnapshot{
		EdiblePercent: f.EdiblePercent,
		Energy:        f.Energy,
		Heat:          f.Heat,
		Protein:       f.Protein,
		Fats:          f.Fats,
		Carb:          f.Carb,
		DietaryFiber:  f.DietaryFiber,
		Ash:           f.Ash,
		Water:         f.Water,
		Cholesterol:   f.Cholesterol,
		Retinol:       f.Retinol,
		Carotin:       f.Carotin,
		VitaminA:      f.VitaminA,
		VitaminB1:     f.VitaminB1,
		VitaminB2:     f.VitaminB2,
		NicotinicAcid: f.NicotinicAcid,
		VitaminC:      f.VitaminC,
		VitaminE:      f.VitaminE,
		Calcium:       f.Calcium,
		Phosphorus:    f.Phosphorus,
		Potassium:     f.Potassium,
		Sodium:        f.Sodium,
		Magnesium:     f.Magnesium,
		Iron:          f.Iron,
		Zinc:          f.Zinc,
		Selenium:      f.Selenium,
		Copper:        f.Copper,
		Manganese:     f.Manganese,
	}
}
