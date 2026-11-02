package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/pkg/pagination"
)

// 集体对象服务测试打真实 MySQL（与 diet_seed_service_test.go 同源）。
// 清理一律原始 DELETE 硬删：软删除行会占唯一索引，重跑撞键（CLAUDE.md 已知坑）。
const (
	testDietGroupObjectSvcKG      uint64 = 987001
	testDietGroupObjectSvcOtherKG uint64 = 987002
)

func openDietGroupObjectServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := openDietSeedTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&domain.Kindergarten{},
		&domain.AcademicYear{},
		&domain.Class{},
		&domain.DietGroupObject{},
		&domain.DietGroupObjectMeal{},
		&domain.DietGroupObjectClass{},
	), "auto migrate group object tables")
	return db
}

func cleanupDietGroupObjectServiceData(t *testing.T, db *gorm.DB) {
	t.Helper()
	kgIDs := []uint64{testDietGroupObjectSvcKG, testDietGroupObjectSvcOtherKG}
	// 先删子表再删父表：关联表对主表、班级对园所都有真实外键。
	for _, sql := range []string{
		"DELETE FROM diet_group_object_classes WHERE kindergarten_id IN ?",
		"DELETE FROM diet_group_object_meals WHERE kindergarten_id IN ?",
		"DELETE FROM diet_group_objects WHERE kindergarten_id IN ?",
		"DELETE FROM classes WHERE kindergarten_id IN ?",
		"DELETE FROM academic_years WHERE kindergarten_id IN ?",
		"DELETE FROM kindergartens WHERE id IN ?",
	} {
		if err := db.Exec(sql, kgIDs).Error; err != nil {
			t.Logf("cleanup %q: %v", sql, err)
		}
	}
}

type dietGroupObjectServiceFixture struct {
	db         *gorm.DB
	svc        *DietGroupObjectService
	year       *domain.AcademicYear
	emptyYear  *domain.AcademicYear
	otherYear  *domain.AcademicYear
	class1     *domain.Class
	class2     *domain.Class
	class3     *domain.Class
	otherClass *domain.Class
}

func newDietGroupObjectServiceFixture(t *testing.T) *dietGroupObjectServiceFixture {
	t.Helper()
	db := openDietGroupObjectServiceTestDB(t)
	cleanupDietGroupObjectServiceData(t, db)
	t.Cleanup(func() { cleanupDietGroupObjectServiceData(t, db) })

	require.NoError(t, db.Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: testDietGroupObjectSvcKG},
		Name:      "DGO服务测试园",
	}).Error)
	require.NoError(t, db.Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: testDietGroupObjectSvcOtherKG},
		Name:      "DGO服务他园",
	}).Error)

	f := &dietGroupObjectServiceFixture{db: db}
	f.year = &domain.AcademicYear{
		KindergartenID: testDietGroupObjectSvcKG, Year: "2026-2027", IsCurrent: true,
	}
	f.emptyYear = &domain.AcademicYear{
		KindergartenID: testDietGroupObjectSvcKG, Year: "2025-2026",
	}
	f.otherYear = &domain.AcademicYear{
		KindergartenID: testDietGroupObjectSvcOtherKG, Year: "2026-2027", IsCurrent: true,
	}
	for _, year := range []*domain.AcademicYear{f.year, f.emptyYear, f.otherYear} {
		require.NoError(t, db.Create(year).Error)
	}

	f.class1 = &domain.Class{
		KindergartenID: testDietGroupObjectSvcKG, Name: "小一班", Grade: "小班",
		AcademicYear: "2026-2027", Status: 1,
	}
	f.class2 = &domain.Class{
		KindergartenID: testDietGroupObjectSvcKG, Name: "中一班", Grade: "中班",
		AcademicYear: "2026-2027", Status: 1,
	}
	f.class3 = &domain.Class{
		KindergartenID: testDietGroupObjectSvcKG, Name: "大一班", Grade: "大班",
		AcademicYear: "2026-2027", Status: 1,
	}
	f.otherClass = &domain.Class{
		KindergartenID: testDietGroupObjectSvcOtherKG, Name: "他园班", Grade: "小班",
		AcademicYear: "2026-2027", Status: 1,
	}
	for _, class := range []*domain.Class{f.class1, f.class2, f.class3, f.otherClass} {
		require.NoError(t, db.Create(class).Error)
	}

	f.svc = NewDietGroupObjectService(
		repository.NewDietGroupObjectRepo(db),
		repository.NewAcademicYearRepo(db),
		repository.NewClassRepo(db),
	)
	return f
}

// standardGroupObjectInput 是与默认对象一致的入参：日托、标准比例、五餐启用晚点停用。
func standardGroupObjectInput(yearID uint64, classIDs ...uint64) DietGroupObjectInput {
	return DietGroupObjectInput{
		AcademicYearID: yearID,
		Name:           "全体幼儿",
		CareType:       domain.DietCareTypeDayCare,
		FoodInputType:  domain.DietFoodInputTypeMarketWeight,
		RatioMode:      domain.DietRatioModeStandard,
		ClassIDs:       classIDs,
		Meals:          defaultDietMealConfigMeals(),
	}
}

// customGroupObjectInput 是日托自定比例下合规的一组占比（总系数 0.93）。
func customGroupObjectInput(yearID uint64, classIDs ...uint64) DietGroupObjectInput {
	in := standardGroupObjectInput(yearID, classIDs...)
	in.RatioMode = domain.DietRatioModeCustom
	for i := range in.Meals {
		switch in.Meals[i].MealKey {
		case domain.DietMealBreakfast:
			in.Meals[i].EnergyPercent = 24
		case domain.DietMealLunch:
			in.Meals[i].EnergyPercent = 34
		}
	}
	return in
}

func requireGroupObjectInvalid(t *testing.T, err error, wantSubstring string) {
	t.Helper()
	require.Error(t, err)
	var invalid *DietConfigValidationError
	require.True(t, errors.As(err, &invalid), "必须是可读的业务校验错误，实际：%v", err)
	assert.Contains(t, invalid.Message, wantSubstring)
}

func (f *dietGroupObjectServiceFixture) countObjects(t *testing.T) int64 {
	t.Helper()
	var count int64
	require.NoError(t, f.db.Model(&domain.DietGroupObject{}).
		Where("kindergarten_id = ?", testDietGroupObjectSvcKG).Count(&count).Error)
	return count
}

// TC-NEW-DIE-0065：没有对象的学年返回空列表；有对象时按学年与关键词过滤。
func TestDietGroupObjectService_TC0065_ListEmptyAndFilter(t *testing.T) {
	f := newDietGroupObjectServiceFixture(t)
	ctx := context.Background()

	items, total, page, pageSize, err := f.svc.List(
		ctx, testDietGroupObjectSvcKG, f.emptyYear.ID, "", pagination.Query{})
	require.NoError(t, err)
	assert.Empty(t, items)
	assert.Zero(t, total)
	assert.Equal(t, 1, page)
	assert.Equal(t, 20, pageSize)

	items, total, _, _, err = f.svc.List(
		ctx, testDietGroupObjectSvcKG, 0, "", pagination.Query{Page: 1, PageSize: 10})
	require.NoError(t, err, "当前学年没有对象时也必须返回空列表，不得自动创建")
	assert.Empty(t, items)
	assert.Zero(t, total)

	first, err := f.svc.Create(ctx, testDietGroupObjectSvcKG,
		standardGroupObjectInput(f.year.ID, f.class1.ID, f.class2.ID, f.class3.ID))
	require.NoError(t, err)
	dayCare := standardGroupObjectInput(f.year.ID, f.class1.ID)
	dayCare.Name = "日托班"
	_, err = f.svc.Create(ctx, testDietGroupObjectSvcKG, dayCare)
	require.NoError(t, err)
	emptyYearObject := standardGroupObjectInput(f.emptyYear.ID, f.class1.ID)
	_, err = f.svc.Create(ctx, testDietGroupObjectSvcKG, emptyYearObject)
	require.NoError(t, err)

	items, total, _, _, err = f.svc.List(
		ctx, testDietGroupObjectSvcKG, f.year.ID, "", pagination.Query{})
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	require.Len(t, items, 2)

	items, total, _, _, err = f.svc.List(
		ctx, testDietGroupObjectSvcKG, f.year.ID, " 日托 ", pagination.Query{})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, items, 1)
	assert.Equal(t, "日托班", items[0].Name)

	items, total, _, _, err = f.svc.List(
		ctx, testDietGroupObjectSvcKG, f.emptyYear.ID, "", pagination.Query{})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total, "学年过滤不得串学年")
	require.Len(t, items, 1)

	items, total, _, _, err = f.svc.List(
		ctx, testDietGroupObjectSvcKG, 0, "", pagination.Query{Page: 1, PageSize: 10})
	require.NoError(t, err, "year_id 缺省取当前学年")
	assert.EqualValues(t, 2, total)
	require.Len(t, items, 2)
	assert.NotEqual(t, first.ID, 0)
}

// TC-NEW-DIE-0066 + TC-NEW-DIE-0074：新增默认对象成功，列表可见；详情返回完整配置。
func TestDietGroupObjectService_TC0066_TC0074_CreateDefaultAndDetail(t *testing.T) {
	f := newDietGroupObjectServiceFixture(t)
	ctx := context.Background()

	in := standardGroupObjectInput(f.year.ID, f.class1.ID, f.class2.ID, f.class3.ID)
	created, err := f.svc.Create(ctx, testDietGroupObjectSvcKG, in)
	require.NoError(t, err)
	require.NotZero(t, created.ID)

	items, total, _, _, err := f.svc.List(
		ctx, testDietGroupObjectSvcKG, f.year.ID, "", pagination.Query{})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, items, 1)
	assert.Equal(t, "全体幼儿", items[0].Name)
	assert.Equal(t, domain.DietCareTypeDayCare, items[0].CareType)
	assert.Equal(t, "日托", items[0].CareTypeLabel)
	assert.Equal(t, domain.DietFoodInputTypeMarketWeight, items[0].FoodInputType)
	assert.Equal(t, "市品重量", items[0].FoodInputTypeLabel)
	assert.Equal(t, "DRIs2000", items[0].PolicyName)
	assert.Equal(t, []string{"早餐", "早点", "午餐", "午点", "晚餐"}, items[0].MealNames,
		"列表餐次只含已启用餐次")
	require.Len(t, items[0].Classes, 3)
	assert.ElementsMatch(t,
		[]string{"小一班", "中一班", "大一班"},
		[]string{items[0].Classes[0].Name, items[0].Classes[1].Name, items[0].Classes[2].Name})

	detail, err := f.svc.Get(ctx, testDietGroupObjectSvcKG, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "全体幼儿", detail.Name)
	assert.Equal(t, f.year.ID, detail.AcademicYearID)
	assert.Equal(t, domain.DietCareTypeDayCare, detail.CareType)
	assert.Equal(t, "日托", detail.CareTypeLabel)
	assert.Equal(t, domain.DietRatioModeStandard, detail.RatioMode)
	assert.Equal(t, "标准比例", detail.RatioModeLabel)
	assert.Equal(t, domain.DietFoodInputTypeMarketWeight, detail.FoodInputType)
	assert.Equal(t, "市品重量", detail.FoodInputTypeLabel)
	assert.Equal(t, "DRIs2000", detail.PolicyName)
	assert.Equal(t, 0.95, detail.Coefficient)
	require.Len(t, detail.Classes, 3)
	require.Len(t, detail.Meals, 6)
	assert.Equal(t, domain.DietMealBreakfast, detail.Meals[0].MealKey)
	assert.Equal(t, "07:00", detail.Meals[0].Time)
	assert.Equal(t, 25.0, detail.Meals[0].EnergyPercent)
	assert.True(t, detail.Meals[0].Enabled)
	assert.Equal(t, 1, detail.Meals[0].Sort)
	assert.Equal(t, domain.DietMealEveningSnack, detail.Meals[5].MealKey)
	assert.False(t, detail.Meals[5].Enabled, "晚点默认停用")
	assert.Equal(t, 6, detail.Meals[5].Sort)

	var mealRows, classRows int64
	require.NoError(t, f.db.Model(&domain.DietGroupObjectMeal{}).
		Where("group_object_id = ?", created.ID).Count(&mealRows).Error)
	require.NoError(t, f.db.Model(&domain.DietGroupObjectClass{}).
		Where("group_object_id = ?", created.ID).Count(&classRows).Error)
	assert.EqualValues(t, 6, mealRows)
	assert.EqualValues(t, 3, classRows)

	_, err = f.svc.Get(ctx, testDietGroupObjectSvcOtherKG, created.ID)
	assert.ErrorIs(t, err, ErrDietGroupObjectNotFound, "跨园读取按不存在处理")
	_, err = f.svc.Get(ctx, testDietGroupObjectSvcKG, 1<<40)
	assert.ErrorIs(t, err, ErrDietGroupObjectNotFound)
}

// TC-NEW-DIE-0067：同学年重名被拒绝且原对象不变；名称长度与空值校验。
func TestDietGroupObjectService_TC0067_RejectsDuplicateName(t *testing.T) {
	f := newDietGroupObjectServiceFixture(t)
	ctx := context.Background()

	original, err := f.svc.Create(ctx, testDietGroupObjectSvcKG,
		standardGroupObjectInput(f.year.ID, f.class1.ID))
	require.NoError(t, err)

	dup := standardGroupObjectInput(f.year.ID, f.class1.ID)
	dup.Name = " 全体幼儿 "
	_, err = f.svc.Create(ctx, testDietGroupObjectSvcKG, dup)
	requireGroupObjectInvalid(t, err, "同名")
	assert.EqualValues(t, 1, f.countObjects(t), "重名保存不得落库")

	unchanged, err := f.svc.Get(ctx, testDietGroupObjectSvcKG, original.ID)
	require.NoError(t, err)
	assert.Equal(t, "全体幼儿", unchanged.Name)
	assert.Equal(t, domain.DietCareTypeDayCare, unchanged.CareType)

	emptyName := standardGroupObjectInput(f.year.ID, f.class1.ID)
	emptyName.Name = "   "
	_, err = f.svc.Create(ctx, testDietGroupObjectSvcKG, emptyName)
	requireGroupObjectInvalid(t, err, "请设置集体对象名称")

	tooLong := standardGroupObjectInput(f.year.ID, f.class1.ID)
	tooLong.Name = strings.Repeat("名", 21)
	_, err = f.svc.Create(ctx, testDietGroupObjectSvcKG, tooLong)
	requireGroupObjectInvalid(t, err, "20 个字符")

	// 名称唯一按学年隔离：别的学年允许同名。
	otherYear := standardGroupObjectInput(f.emptyYear.ID, f.class1.ID)
	_, err = f.svc.Create(ctx, testDietGroupObjectSvcKG, otherYear)
	require.NoError(t, err)
}

// TC-NEW-DIE-0068：未选择就餐班级被拒绝；跨园班级同样被拒绝。
func TestDietGroupObjectService_TC0068_RejectsNoClass(t *testing.T) {
	f := newDietGroupObjectServiceFixture(t)
	ctx := context.Background()

	noClass := standardGroupObjectInput(f.year.ID)
	_, err := f.svc.Create(ctx, testDietGroupObjectSvcKG, noClass)
	requireGroupObjectInvalid(t, err, "至少选择一个就餐班级")
	assert.Zero(t, f.countObjects(t), "校验失败不得落库")

	crossClass := standardGroupObjectInput(f.year.ID, f.otherClass.ID)
	_, err = f.svc.Create(ctx, testDietGroupObjectSvcKG, crossClass)
	requireGroupObjectInvalid(t, err, "不属于本园")
	assert.Zero(t, f.countObjects(t))

	crossYear := standardGroupObjectInput(f.otherYear.ID, f.class1.ID)
	_, err = f.svc.Create(ctx, testDietGroupObjectSvcKG, crossYear)
	requireGroupObjectInvalid(t, err, "学年不存在或不属于本园")
	assert.Zero(t, f.countObjects(t))

	// 重复班级去重后只保存一行。
	deduped, err := f.svc.Create(ctx, testDietGroupObjectSvcKG,
		standardGroupObjectInput(f.year.ID, f.class1.ID, f.class1.ID, f.class2.ID))
	require.NoError(t, err)
	assert.Len(t, deduped.Classes, 2)
}

// 修改：替换班级与餐次后旧关联行不残留（硬删），跨园学年/班级与重名、占比错误被拒绝。
func TestDietGroupObjectService_UpdateReplacesLinksAndValidates(t *testing.T) {
	f := newDietGroupObjectServiceFixture(t)
	ctx := context.Background()

	created, err := f.svc.Create(ctx, testDietGroupObjectSvcKG,
		standardGroupObjectInput(f.year.ID, f.class1.ID, f.class2.ID))
	require.NoError(t, err)

	update := customGroupObjectInput(f.year.ID, f.class3.ID)
	update.Name = "日托班"
	updated, err := f.svc.Update(ctx, testDietGroupObjectSvcKG, created.ID, update)
	require.NoError(t, err)
	assert.Equal(t, "日托班", updated.Name)
	assert.Equal(t, domain.DietRatioModeCustom, updated.RatioMode)
	assert.Equal(t, "自定比例", updated.RatioModeLabel)
	assert.Equal(t, 0.93, updated.Coefficient, "24+5+34+5+25 = 93")
	require.Len(t, updated.Classes, 1)
	assert.Equal(t, f.class3.ID, updated.Classes[0].ID)
	assert.Equal(t, 24.0, updated.Meals[0].EnergyPercent)

	var classRows, allClassRows, mealRows, allMealRows int64
	require.NoError(t, f.db.Model(&domain.DietGroupObjectClass{}).
		Where("group_object_id = ?", created.ID).Count(&classRows).Error)
	require.NoError(t, f.db.Unscoped().Model(&domain.DietGroupObjectClass{}).
		Where("group_object_id = ?", created.ID).Count(&allClassRows).Error)
	require.NoError(t, f.db.Model(&domain.DietGroupObjectMeal{}).
		Where("group_object_id = ?", created.ID).Count(&mealRows).Error)
	require.NoError(t, f.db.Unscoped().Model(&domain.DietGroupObjectMeal{}).
		Where("group_object_id = ?", created.ID).Count(&allMealRows).Error)
	assert.EqualValues(t, 1, classRows, "被移除的班级关联不得残留")
	assert.EqualValues(t, 1, allClassRows, "旧班级关联必须硬删，不能软删占键")
	assert.EqualValues(t, 6, mealRows)
	assert.EqualValues(t, 6, allMealRows, "餐次按业务键 upsert，不得重复插入")

	// 跨园学年/班级被拒绝，且原对象不变。
	crossYear := customGroupObjectInput(f.otherYear.ID, f.class3.ID)
	_, err = f.svc.Update(ctx, testDietGroupObjectSvcKG, created.ID, crossYear)
	requireGroupObjectInvalid(t, err, "学年不存在或不属于本园")
	crossClass := customGroupObjectInput(f.year.ID, f.otherClass.ID)
	_, err = f.svc.Update(ctx, testDietGroupObjectSvcKG, created.ID, crossClass)
	requireGroupObjectInvalid(t, err, "不属于本园")
	still, err := f.svc.Get(ctx, testDietGroupObjectSvcKG, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "日托班", still.Name)
	require.Len(t, still.Classes, 1)
	assert.Equal(t, f.class3.ID, still.Classes[0].ID)

	// 同学年重名被拒绝，另一个对象不变。
	other, err := f.svc.Create(ctx, testDietGroupObjectSvcKG,
		standardGroupObjectInput(f.year.ID, f.class1.ID))
	require.NoError(t, err)
	rename := standardGroupObjectInput(f.year.ID, f.class1.ID)
	rename.Name = "日托班"
	_, err = f.svc.Update(ctx, testDietGroupObjectSvcKG, other.ID, rename)
	requireGroupObjectInvalid(t, err, "同名")
	otherDetail, err := f.svc.Get(ctx, testDietGroupObjectSvcKG, other.ID)
	require.NoError(t, err)
	assert.Equal(t, "全体幼儿", otherDetail.Name)

	// 标准比例下非固定占比被拒绝（复用旧校验规则与文案）。
	badPercent := standardGroupObjectInput(f.year.ID, f.class3.ID)
	badPercent.Name = "日托班"
	for i := range badPercent.Meals {
		if badPercent.Meals[i].MealKey == domain.DietMealBreakfast {
			badPercent.Meals[i].EnergyPercent = 24
		}
	}
	_, err = f.svc.Update(ctx, testDietGroupObjectSvcKG, created.ID, badPercent)
	requireGroupObjectInvalid(t, err, "标准比例")
	unchanged, err := f.svc.Get(ctx, testDietGroupObjectSvcKG, created.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.DietRatioModeCustom, unchanged.RatioMode, "校验失败不得改写对象")
	assert.Equal(t, 24.0, unchanged.Meals[0].EnergyPercent)

	missing := customGroupObjectInput(f.year.ID, f.class1.ID)
	_, err = f.svc.Update(ctx, testDietGroupObjectSvcKG, 1<<40, missing)
	assert.ErrorIs(t, err, ErrDietGroupObjectNotFound)
}

// BuildConfigSnapshot：对象身份、系数 0.95 与餐次结构正确，DRIs 留空待周食谱服务补。
func TestDietGroupObjectService_BuildConfigSnapshot(t *testing.T) {
	f := newDietGroupObjectServiceFixture(t)
	ctx := context.Background()

	created, err := f.svc.Create(ctx, testDietGroupObjectSvcKG,
		standardGroupObjectInput(f.year.ID, f.class3.ID, f.class1.ID, f.class2.ID))
	require.NoError(t, err)

	snapshot, err := f.svc.BuildConfigSnapshot(ctx, testDietGroupObjectSvcKG, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, snapshot.GroupObjectID)
	assert.Equal(t, "全体幼儿", snapshot.GroupObjectName)
	assert.Equal(t, []uint64{f.class1.ID, f.class2.ID, f.class3.ID},
		snapshot.GroupObjectClassIDs, "班级 ID 升序，供复核加权范围")
	assert.Equal(t, domain.DietCareTypeDayCare, snapshot.CareType)
	assert.Equal(t, domain.DietFoodInputTypeMarketWeight, snapshot.FoodInputType)
	assert.Equal(t, domain.DietRatioModeStandard, snapshot.RatioMode)
	assert.Equal(t, 0.95, snapshot.Coefficient)
	require.Len(t, snapshot.Meals, 6)
	assert.Equal(t, domain.DietMealBreakfast, snapshot.Meals[0].MealKey)
	assert.Equal(t, "07:00", snapshot.Meals[0].Time)
	assert.Equal(t, 25.0, snapshot.Meals[0].EnergyPercent)
	assert.True(t, snapshot.Meals[0].Enabled)
	assert.Equal(t, 1, snapshot.Meals[0].Sort)
	assert.False(t, snapshot.Meals[5].Enabled)
	assert.Empty(t, snapshot.DriTargets, "DRIs 由周食谱服务按就餐班级加权后填充")
	assert.Empty(t, snapshot.DriWeights)

	_, err = f.svc.BuildConfigSnapshot(ctx, testDietGroupObjectSvcKG, 1<<40)
	assert.ErrorIs(t, err, ErrDietGroupObjectNotFound)
	_, err = f.svc.BuildConfigSnapshot(ctx, testDietGroupObjectSvcOtherKG, created.ID)
	assert.ErrorIs(t, err, ErrDietGroupObjectNotFound)
}

// 班级被删除后，列表/详情只返回仍存在的班级（design Risks）。
func TestDietGroupObjectService_DropsDeletedClasses(t *testing.T) {
	f := newDietGroupObjectServiceFixture(t)
	ctx := context.Background()

	created, err := f.svc.Create(ctx, testDietGroupObjectSvcKG,
		standardGroupObjectInput(f.year.ID, f.class1.ID, f.class2.ID, f.class3.ID))
	require.NoError(t, err)
	require.Len(t, created.Classes, 3)

	require.NoError(t, f.db.Delete(&domain.Class{}, f.class2.ID).Error)

	items, _, _, _, err := f.svc.List(
		ctx, testDietGroupObjectSvcKG, f.year.ID, "", pagination.Query{})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Len(t, items[0].Classes, 2, "已删除的班级不得出现在列表")
	for _, class := range items[0].Classes {
		assert.NotEqual(t, f.class2.ID, class.ID)
	}

	detail, err := f.svc.Get(ctx, testDietGroupObjectSvcKG, created.ID)
	require.NoError(t, err)
	require.Len(t, detail.Classes, 2, "已删除的班级不得出现在详情")
	for _, class := range detail.Classes {
		assert.NotEqual(t, f.class2.ID, class.ID)
	}
}

// 没有当前学年时列表返回空，不回退到任意学年。
func TestDietGroupObjectService_ListWithoutCurrentYearReturnsEmpty(t *testing.T) {
	f := newDietGroupObjectServiceFixture(t)
	ctx := context.Background()

	_, err := f.svc.Create(ctx, testDietGroupObjectSvcKG,
		standardGroupObjectInput(f.year.ID, f.class1.ID))
	require.NoError(t, err)
	require.NoError(t, f.db.Model(&domain.AcademicYear{}).
		Where("kindergarten_id = ?", testDietGroupObjectSvcKG).
		Update("is_current", false).Error)

	items, total, page, pageSize, err := f.svc.List(
		ctx, testDietGroupObjectSvcKG, 0, "", pagination.Query{})
	require.NoError(t, err)
	assert.Empty(t, items)
	assert.Zero(t, total)
	assert.Equal(t, 1, page)
	assert.Equal(t, 20, pageSize)
}
