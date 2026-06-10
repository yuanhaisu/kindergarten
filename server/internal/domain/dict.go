package domain

// SysDictType 字典类型（如 performance_tag、growth_category 等）。
// 每个幼儿园独立配置，type_key 在园所内唯一。
type SysDictType struct {
	SoftDeleteModel
	KindergartenID uint64 `gorm:"not null;uniqueIndex:idx_dict_type_key_kg;column:kindergarten_id" json:"kindergarten_id"`
	TypeKey        string `gorm:"size:64;not null;uniqueIndex:idx_dict_type_key_kg;column:type_key" json:"type_key"`
	TypeName       string `gorm:"size:128;not null;column:type_name" json:"type_name"`
	Description    string `gorm:"size:512;column:description" json:"description"`
	IsSystem       bool   `gorm:"not null;default:false;column:is_system" json:"is_system"`
	SortOrder      int    `gorm:"default:0;column:sort_order" json:"sort_order"`
}

func (SysDictType) TableName() string { return "sys_dict_types" }

// SysDictItem 字典项（字典类型下的具体可选值）。
// Extra 为 JSON 列，预留存储颜色、图标等 UI 元数据。
type SysDictItem struct {
	SoftDeleteModel
	TypeID    uint64 `gorm:"not null;index;column:type_id" json:"type_id"`
	ItemKey   string `gorm:"size:64;not null;column:item_key" json:"item_key"`
	ItemLabel string `gorm:"size:256;not null;column:item_label" json:"item_label"`
	Extra     string `gorm:"type:json;column:extra" json:"extra"`
	SortOrder int    `gorm:"default:0;column:sort_order" json:"sort_order"`
	IsEnabled bool   `gorm:"not null;default:true;column:is_enabled" json:"is_enabled"`
}

func (SysDictItem) TableName() string { return "sys_dict_items" }
