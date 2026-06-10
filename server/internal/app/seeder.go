package app

import (
	"context"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

// SeedDicts seeds default dictionary data for a kindergarten.
// Uses an idempotent check — only inserts if the type doesn't already exist.
func SeedDicts(db *gorm.DB, kgID uint64) error {
	var count int64
	if err := db.Model(&domain.SysDictType{}).
		Where("kindergarten_id = ? AND type_key = ?", kgID, "performance_tag").
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil // already seeded
	}

	dictType := &domain.SysDictType{
		KindergartenID: kgID,
		TypeKey:        "performance_tag",
		TypeName:       "表现标签",
		Description:    "成长记录中的幼儿表现标签",
		IsSystem:       true,
		SortOrder:      0,
	}
	if err := db.Create(dictType).Error; err != nil {
		return err
	}

	items := []domain.SysDictItem{
		{TypeID: dictType.ID, ItemKey: "helpful", ItemLabel: "乐于助人", SortOrder: 1, IsEnabled: true},
		{TypeID: dictType.ID, ItemKey: "active", ItemLabel: "积极发言", SortOrder: 2, IsEnabled: true},
		{TypeID: dictType.ID, ItemKey: "disciplined", ItemLabel: "遵守纪律", SortOrder: 3, IsEnabled: true},
		{TypeID: dictType.ID, ItemKey: "hands_on", ItemLabel: "动手能力强", SortOrder: 4, IsEnabled: true},
		{TypeID: dictType.ID, ItemKey: "focused", ItemLabel: "专注力强", SortOrder: 5, IsEnabled: true},
		{TypeID: dictType.ID, ItemKey: "polite", ItemLabel: "有礼貌", SortOrder: 6, IsEnabled: true},
		{TypeID: dictType.ID, ItemKey: "clean", ItemLabel: "爱干净", SortOrder: 7, IsEnabled: true},
		{TypeID: dictType.ID, ItemKey: "friendly", ItemLabel: "团结友善", SortOrder: 8, IsEnabled: true},
		{TypeID: dictType.ID, ItemKey: "brave", ItemLabel: "勇于尝试", SortOrder: 9, IsEnabled: true},
	}

	for i := range items {
		if err := db.Create(&items[i]).Error; err != nil {
			return err
		}
	}
	return nil
}

// SeedDictsForAll seeds dict data for all existing kindergartens.
func SeedDictsForAll(db *gorm.DB) error {
	var kgs []domain.Kindergarten
	if err := db.Find(&kgs).Error; err != nil {
		return err
	}
	for _, kg := range kgs {
		if err := SeedDicts(db, kg.ID); err != nil {
			return err
		}
	}
	// Also seed for system default (kgID=0) so new kindergartens can copy from it
	return SeedDicts(db, 0)
}

// SeedDictsAfterMigration should be called after AutoMigrate in provider.go.
func SeedDictsAfterMigration(db *gorm.DB) {
	ctx := context.Background()
	_ = db.WithContext(ctx) // satisfy the context pattern
	if err := SeedDictsForAll(db); err != nil {
		// Log but don't fail startup — dict seeding is not critical
		println("dict seeder: " + err.Error())
	}
}
