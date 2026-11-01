package domain

// Recipe 菜谱：一组「食物 + 一人均量克数」的命名快照。
//
// 同一园所内名称唯一，但唯一性由服务层保证，不建唯一索引——
// 软删除行仍占用唯一索引键，会让「删除后以同名重建」失败（design D2 风险）。
// 菜谱不存营养快照；营养在写入周食谱时按当时的食物库固化（ADR-0006）。
type Recipe struct {
	SoftDeleteModel
	KindergartenID uint64 `gorm:"not null;index:idx_diet_recipe_kg_name,priority:1" json:"kindergarten_id"`
	Name           string `gorm:"size:128;not null;index:idx_diet_recipe_kg_name,priority:2" json:"name"`
	SpellCode      string `gorm:"size:64;not null;default:'';index" json:"spell_code"`

	Items []RecipeItem `gorm:"foreignKey:RecipeID" json:"items,omitempty"`
}

func (Recipe) TableName() string { return "diet_recipes" }

// RecipeItem 菜谱配料。只存食物 ID、克数与顺序，不存营养快照。
type RecipeItem struct {
	BaseModel
	KindergartenID uint64  `gorm:"not null;index" json:"kindergarten_id"`
	RecipeID       uint64  `gorm:"not null;index" json:"recipe_id"`
	FoodID         uint64  `gorm:"not null;index" json:"food_id"`
	Grams          float64 `gorm:"not null;default:0" json:"grams"`
	Sort           int     `gorm:"not null;default:0" json:"sort"`
}

func (RecipeItem) TableName() string { return "diet_recipe_items" }
