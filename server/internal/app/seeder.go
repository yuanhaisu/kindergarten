package app

import (
	"context"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

// SeedDicts seeds default dictionary data for a kindergarten.
//
// 每个字典类型独立判重（幂等）。注意：不能只查一个类型就对整个函数早退——
// 那样向后追加的新类型会被「已播种过老类型的园所」全部跳过（align-class-calendar 修复）。
func SeedDicts(db *gorm.DB, kgID uint64) error {
	if err := seedPerformanceTag(db, kgID); err != nil {
		return err
	}
	return seedCalendarEventType(db, kgID)
}

// dictTypeAbsent 该园是否尚无某字典类型（判重用）。
func dictTypeAbsent(db *gorm.DB, kgID uint64, typeKey string) (bool, error) {
	var count int64
	if err := db.Model(&domain.SysDictType{}).
		Where("kindergarten_id = ? AND type_key = ?", kgID, typeKey).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count == 0, nil
}

func seedPerformanceTag(db *gorm.DB, kgID uint64) error {
	absent, err := dictTypeAbsent(db, kgID, "performance_tag")
	if err != nil {
		return err
	}
	if !absent {
		return nil
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

// seedCalendarEventType 日历事件类型字典（内置 5 类，不可删除、可改名/停用）。
// 色值取自 design-data/teacher-06 图例；图标键由前端按 item_key 映射，不入字典（避免凭空拟值）。
func seedCalendarEventType(db *gorm.DB, kgID uint64) error {
	absent, err := dictTypeAbsent(db, kgID, "calendar_event_type")
	if err != nil {
		return err
	}
	if !absent {
		return nil
	}

	dictType := &domain.SysDictType{
		KindergartenID: kgID,
		TypeKey:        "calendar_event_type",
		TypeName:       "日历事件类型",
		Description:    "班级日历事件类型（内置 5 类不可删除，园所可扩展）",
		IsSystem:       true,
		SortOrder:      1,
	}
	if err := db.Create(dictType).Error; err != nil {
		return err
	}

	items := []domain.SysDictItem{
		{TypeID: dictType.ID, ItemKey: "notice", ItemLabel: "通知", Extra: `{"color":"#3B82F6"}`, SortOrder: 1, IsEnabled: true},
		{TypeID: dictType.ID, ItemKey: "activity", ItemLabel: "活动", Extra: `{"color":"#22C55E"}`, SortOrder: 2, IsEnabled: true},
		{TypeID: dictType.ID, ItemKey: "holiday", ItemLabel: "假期", Extra: `{"color":"#F97316"}`, SortOrder: 3, IsEnabled: true},
		{TypeID: dictType.ID, ItemKey: "meeting", ItemLabel: "家长会", Extra: `{"color":"#A855F7"}`, SortOrder: 4, IsEnabled: true},
		{TypeID: dictType.ID, ItemKey: "checkup", ItemLabel: "体检", Extra: `{"color":"#EF4444"}`, SortOrder: 5, IsEnabled: true},
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
