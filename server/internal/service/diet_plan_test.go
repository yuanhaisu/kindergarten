package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/pkg/pagination"
)

// 周食谱 service 测试打真实 MySQL（与 repository/其他 service 测试同源）。
// 清理一律原始 DELETE 硬删：软删除行会占唯一索引，重跑撞键（CLAUDE.md 已知坑）。
const (
	testDietPlanSvcKG      uint64 = 990995
	testDietPlanSvcOtherKG uint64 = 990996
)

var (
	dietPlanWeek1Start = time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC) // 周一
	dietPlanWeek2Start = time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)  // 下周一
)

func openDietPlanServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	cfgPath := filepath.Join("..", "..", "configs", "app.ini")
	if _, err := os.Stat(cfgPath); err != nil {
		t.Skipf("跳过：找不到 %s（本地无 DB 配置）", cfgPath)
	}
	cfg, err := config.Load(cfgPath)
	require.NoError(t, err, "load config")
	if cfg.MySQL.DSN == "" {
		t.Skip("跳过：MySQL DSN 为空")
	}
	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err, "open db")
	require.NoError(t, db.AutoMigrate(
		&domain.MealConfig{}, &domain.MealConfigMeal{},
		&domain.DietGroupObject{},
		&domain.DietGroupObjectMeal{},
		&domain.DietGroupObjectClass{},
		&domain.FoodCategory{}, &domain.Food{},
		&domain.Recipe{}, &domain.RecipeItem{},
		&domain.MealPlan{}, &domain.MealPlanItem{},
		&domain.TemplatePlan{}, &domain.TemplatePlanItem{},
		&domain.DriAgeTarget{},
		&domain.AcademicYear{}, &domain.Class{}, &domain.Student{},
	), "auto migrate diet plan tables")
	return db
}

func cleanupDietPlanServiceData(t *testing.T, db *gorm.DB) {
	t.Helper()
	kgIDs := []uint64{testDietPlanSvcKG, testDietPlanSvcOtherKG}
	// 先删子表再删父表：明细/条目与主表之间有真实外键。
	for _, sql := range []string{
		"DELETE FROM diet_meal_plan_items WHERE kindergarten_id IN ?",
		"DELETE FROM diet_meal_plans WHERE kindergarten_id IN ?",
		"DELETE FROM diet_template_plan_items WHERE kindergarten_id IN ?",
		"DELETE FROM diet_template_plans WHERE kindergarten_id IN ?",
		"DELETE FROM diet_recipe_items WHERE kindergarten_id IN ?",
		"DELETE FROM diet_recipes WHERE kindergarten_id IN ?",
		"DELETE FROM diet_meal_config_meals WHERE kindergarten_id IN ?",
		"DELETE FROM diet_meal_configs WHERE kindergarten_id IN ?",
		"DELETE FROM diet_group_object_classes WHERE kindergarten_id IN ?",
		"DELETE FROM diet_group_object_meals WHERE kindergarten_id IN ?",
		"DELETE FROM diet_group_objects WHERE kindergarten_id IN ?",
		"DELETE FROM diet_foods WHERE kindergarten_id IN ?",
		"DELETE FROM students WHERE kindergarten_id IN ?",
		"DELETE FROM classes WHERE kindergarten_id IN ?",
		"DELETE FROM academic_years WHERE kindergarten_id IN ?",
		"DELETE FROM kindergartens WHERE id IN ?",
	} {
		if err := db.Exec(sql, kgIDs).Error; err != nil {
			t.Logf("cleanup %q: %v", sql, err)
		}
	}
}

type dietPlanFixture struct {
	db             *gorm.DB
	svc            *DietPlanService
	configSvc      *DietConfigService
	objectSvc      *DietGroupObjectService
	foodRepo       *repository.DietFoodRepo
	planRepo       *repository.DietPlanRepo
	tplRepo        *repository.DietTemplateRepo
	driRepo        *repository.DietDriRepo
	classRepo      *repository.ClassRepo
	yearRepo       *repository.AcademicYearRepo
	year           *domain.AcademicYear
	object         *DietGroupObjectDetail
	classesByGrade map[string]*domain.Class
}

func newDietPlanFixture(t *testing.T) *dietPlanFixture {
	t.Helper()
	ctx := context.Background()
	db := openDietPlanServiceTestDB(t)
	cleanupDietPlanServiceData(t, db)
	t.Cleanup(func() { cleanupDietPlanServiceData(t, db) })

	// classes.academic_year / classes.kindergarten_id 有真实外键，加权用例需要园所与班级行。
	require.NoError(t, db.Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: testDietPlanSvcKG}, Name: "S6测试园所",
	}).Error)
	require.NoError(t, db.Create(&domain.Kindergarten{
		BaseModel: domain.BaseModel{ID: testDietPlanSvcOtherKG}, Name: "S6测试他园",
	}).Error)

	configSvc := NewDietConfigService(repository.NewDietConfigRepo(db))
	foodRepo := repository.NewDietFoodRepo(db)
	planRepo := repository.NewDietPlanRepo(db)
	tplRepo := repository.NewDietTemplateRepo(db)
	driRepo := repository.NewDietDriRepo(db)
	classRepo := repository.NewClassRepo(db)
	yearRepo := repository.NewAcademicYearRepo(db)
	objectSvc := NewDietGroupObjectService(
		repository.NewDietGroupObjectRepo(db), yearRepo, classRepo)
	svc := NewDietPlanService(
		planRepo, tplRepo, foodRepo, classRepo, yearRepo, driRepo, configSvc, objectSvc,
	)

	// DRIs 是全局只读数据；测试按正式种子幂等写入，避免依赖开发库是否启动过。
	_, err := driRepo.CreateBatch(ctx, dietDriSeedRows())
	require.NoError(t, err, "seed DRI age targets")

	year := &domain.AcademicYear{
		KindergartenID: testDietPlanSvcKG,
		Year:           "2026-2027",
		IsCurrent:      true,
	}
	require.NoError(t, yearRepo.Create(ctx, year))

	f := &dietPlanFixture{
		db:             db,
		svc:            svc,
		configSvc:      configSvc,
		objectSvc:      objectSvc,
		foodRepo:       foodRepo,
		planRepo:       planRepo,
		tplRepo:        tplRepo,
		driRepo:        driRepo,
		classRepo:      classRepo,
		yearRepo:       yearRepo,
		year:           year,
		classesByGrade: make(map[string]*domain.Class, 3),
	}
	// 三个年级班级先建好（此时无学生）：默认集体对象要覆盖真实班级，
	// 加权用例再按年级补学生，避免对象创建后班级才出现。
	for _, grade := range []string{
		dietPlanClassGradeJunior, dietPlanClassGradeMiddle, dietPlanClassGradeSenior,
	} {
		class := &domain.Class{
			KindergartenID: testDietPlanSvcKG,
			Name:           "S6测试" + grade,
			Grade:          grade,
			AcademicYear:   year.Year,
			Status:         1,
		}
		require.NoError(t, db.Create(class).Error)
		f.classesByGrade[grade] = class
	}
	f.object = f.createGroupObject(t, year.ID, "全体幼儿", f.allClassIDs()...)
	return f
}

func (f *dietPlanFixture) allClassIDs() []uint64 {
	out := make([]uint64, 0, len(f.classesByGrade))
	for _, grade := range []string{
		dietPlanClassGradeJunior, dietPlanClassGradeMiddle, dietPlanClassGradeSenior,
	} {
		out = append(out, f.classesByGrade[grade].ID)
	}
	return out
}

// groupObjectInput 与默认对象一致：日托、标准比例、五餐启用晚点停用，DRIs2000。
func (f *dietPlanFixture) groupObjectInput(
	yearID uint64,
	name string,
	classIDs ...uint64,
) DietGroupObjectInput {
	return DietGroupObjectInput{
		AcademicYearID: yearID,
		Name:           name,
		CareType:       domain.DietCareTypeDayCare,
		FoodInputType:  domain.DietFoodInputTypeMarketWeight,
		RatioMode:      domain.DietRatioModeStandard,
		ClassIDs:       classIDs,
		Meals:          defaultDietMealConfigMeals(),
	}
}

func (f *dietPlanFixture) createGroupObject(
	t *testing.T,
	yearID uint64,
	name string,
	classIDs ...uint64,
) *DietGroupObjectDetail {
	t.Helper()
	object, err := f.objectSvc.Create(
		context.Background(), testDietPlanSvcKG, f.groupObjectInput(yearID, name, classIDs...))
	require.NoError(t, err)
	require.NotZero(t, object.ID)
	return object
}

func dietFloat(v float64) *float64 { return &v }

func (f *dietPlanFixture) createFood(
	t *testing.T,
	kindergartenID uint64,
	name string,
	ediblePercent float64,
	energy, protein *float64,
) *domain.Food {
	t.Helper()
	food := &domain.Food{
		KindergartenID: kindergartenID,
		Name:           name,
		CategoryID:     1,
		EdiblePercent:  ediblePercent,
		Source:         domain.DietFoodSourceCustom,
		Energy:         energy,
		Protein:        protein,
	}
	require.NoError(t, f.foodRepo.Create(context.Background(), food))
	return food
}

func (f *dietPlanFixture) createTemplate(
	t *testing.T,
	key string,
	items []domain.TemplatePlanItem,
) *domain.TemplatePlan {
	t.Helper()
	tpl := &domain.TemplatePlan{
		KindergartenID: testDietPlanSvcKG,
		TemplateKey:    key,
		Name:           "S6测试模板-" + key,
		DayNum:         5,
		MealKeys:       "breakfast,lunch",
		Sort:           1,
	}
	for i := range items {
		items[i].KindergartenID = testDietPlanSvcKG
	}
	tpl.Items = items
	require.NoError(t, f.tplRepo.CreateWithItems(context.Background(), tpl))
	return tpl
}

func (f *dietPlanFixture) createSourcePlan(
	t *testing.T,
	name string,
	start time.Time,
	items []domain.MealPlanItem,
) *domain.MealPlan {
	t.Helper()
	plan := &domain.MealPlan{
		KindergartenID: testDietPlanSvcKG,
		AcademicYearID: f.year.ID,
		Name:           name,
		StartDate:      start,
		EndDate:        start.AddDate(0, 0, 6),
		DayNum:         countDietPlanDays(items),
		CreatedMethod:  domain.DietPlanCreatedByCustom,
		ConfigSnapshot: `{"coefficient":0.95,"meals":[]}`,
		Items:          items,
	}
	require.NoError(t, f.planRepo.Create(context.Background(), plan))
	return plan
}

func (f *dietPlanFixture) seedStudents(t *testing.T, grade string, n int) {
	t.Helper()
	class, ok := f.classesByGrade[grade]
	require.True(t, ok, "fixture 未预建 %s 班级", grade)
	for i := 0; i < n; i++ {
		student := &domain.Student{
			KindergartenID: testDietPlanSvcKG,
			ClassID:        class.ID,
			Name:           fmt.Sprintf("S6测试%s学生%d", grade, i),
			Birthday:       time.Now().AddDate(-5, 0, 0),
			EnrollDate:     time.Now().AddDate(-1, 0, 0),
			Status:         1,
		}
		require.NoError(t, f.db.Create(student).Error)
	}
}

func (f *dietPlanFixture) createCustomPlan(t *testing.T, name string) *DietPlanCreateResult {
	t.Helper()
	return f.createCustomPlanWithObject(t, name, f.object.ID)
}

func (f *dietPlanFixture) createCustomPlanWithObject(
	t *testing.T,
	name string,
	objectID uint64,
) *DietPlanCreateResult {
	t.Helper()
	result, err := f.svc.CreatePlan(context.Background(), testDietPlanSvcKG, DietPlanCreateInput{
		Name:           name,
		AcademicYearID: f.year.ID,
		GroupObjectID:  objectID,
		StartDate:      "2026-10-26",
		EndDate:        "2026-11-01",
		WeekOfYear:     9,
		CreatedMethod:  domain.DietPlanCreatedByCustom,
	})
	require.NoError(t, err)
	return result
}

func dietPlanItemInput(
	date string,
	mealKey domain.DietMealKey,
	recipeName string,
	food *domain.Food,
	grams float64,
) DietPlanItemInput {
	return DietPlanItemInput{
		Date:       date,
		MealKey:    mealKey,
		RecipeName: recipeName,
		FoodID:     food.ID,
		FoodName:   food.Name,
		Grams:      grams,
	}
}

// TC-NEW-DIE-0028 / 0030：自定义新建为空白天数，列表按当前学年返回周次与日期区间。
func TestDietPlanCreateCustomAndList(t *testing.T) {
	f := newDietPlanFixture(t)
	ctx := context.Background()

	result := f.createCustomPlan(t, "第九周")
	assert.NotZero(t, result.ID)
	assert.Equal(t, "2026-10-26", result.StartDate)
	assert.Equal(t, "2026-11-01", result.EndDate)
	assert.Equal(t, 9, result.WeekOfYear)
	assert.Zero(t, result.DayNum, "自定义新建为空白食谱")
	assert.Empty(t, result.Warnings)

	list, total, page, pageSize, err := f.svc.ListPlans(
		ctx, testDietPlanSvcKG, 0, "", pagination.Query{})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, 1, page)
	assert.Equal(t, 20, pageSize)
	require.Len(t, list, 1)
	assert.Equal(t, result.ID, list[0].ID)
	assert.Equal(t, "第九周", list[0].Name)
	assert.Equal(t, f.year.ID, list[0].AcademicYearID)
	assert.Equal(t, "2026-10-26", list[0].StartDate)
	assert.Equal(t, "2026-11-01", list[0].EndDate)
	assert.Equal(t, 9, list[0].WeekOfYear)
	assert.Equal(t, domain.DietPlanCreatedByCustom, list[0].CreatedMethod)
	assert.Equal(t, "自定义", list[0].CreatedMethodLabel)
}

// TC-NEW-DIE-0029：同一学年重名 409；换学年可复用名称。
func TestDietPlanCreateRejectsDuplicateNameInAcademicYear(t *testing.T) {
	f := newDietPlanFixture(t)
	ctx := context.Background()

	f.createCustomPlan(t, "第九周")

	_, err := f.svc.CreatePlan(ctx, testDietPlanSvcKG, DietPlanCreateInput{
		Name:           "第九周",
		AcademicYearID: f.year.ID,
		GroupObjectID:  f.object.ID,
		StartDate:      "2026-11-02",
		EndDate:        "2026-11-08",
		CreatedMethod:  domain.DietPlanCreatedByCustom,
	})
	require.ErrorIs(t, err, ErrDietPlanNameConflict)

	otherYear := &domain.AcademicYear{
		KindergartenID: testDietPlanSvcKG,
		Year:           "2025-2026",
	}
	require.NoError(t, f.yearRepo.Create(ctx, otherYear))
	otherObject := f.createGroupObject(t, otherYear.ID, "上学年全体幼儿", f.allClassIDs()...)
	_, err = f.svc.CreatePlan(ctx, testDietPlanSvcKG, DietPlanCreateInput{
		Name:           "第九周",
		AcademicYearID: otherYear.ID,
		GroupObjectID:  otherObject.ID,
		StartDate:      "2025-10-27",
		EndDate:        "2025-11-02",
		CreatedMethod:  domain.DietPlanCreatedByCustom,
	})
	require.NoError(t, err, "不同学年允许同名")
}

// 创建校验：周期必须周一至周日；三种方式的附加字段互斥；学年必须属于本园。
func TestDietPlanCreateValidatesPeriodMethodAndYear(t *testing.T) {
	f := newDietPlanFixture(t)
	ctx := context.Background()

	otherYear := &domain.AcademicYear{
		KindergartenID: testDietPlanSvcOtherKG,
		Year:           "2026-2027",
	}
	require.NoError(t, f.yearRepo.Create(ctx, otherYear))

	tplID := uint64(1)
	planID := uint64(2)
	cases := []struct {
		name    string
		input   DietPlanCreateInput
		wantMsg string
	}{
		{
			name: "开始日期不是周一",
			input: DietPlanCreateInput{
				Name: "S6校验A", AcademicYearID: f.year.ID,
				StartDate: "2026-10-27", EndDate: "2026-11-02",
				CreatedMethod: domain.DietPlanCreatedByCustom,
			},
			wantMsg: "周一",
		},
		{
			name: "结束日期不是开始 + 6",
			input: DietPlanCreateInput{
				Name: "S6校验B", AcademicYearID: f.year.ID,
				StartDate: "2026-10-26", EndDate: "2026-11-02",
				CreatedMethod: domain.DietPlanCreatedByCustom,
			},
			wantMsg: "结束日期",
		},
		{
			name: "日期格式错误",
			input: DietPlanCreateInput{
				Name: "S6校验C", AcademicYearID: f.year.ID,
				StartDate: "2026/10/26", EndDate: "2026/11/01",
				CreatedMethod: domain.DietPlanCreatedByCustom,
			},
			wantMsg: "YYYY-MM-DD",
		},
		{
			name: "模板方式缺 template_id",
			input: DietPlanCreateInput{
				Name: "S6校验D", AcademicYearID: f.year.ID,
				StartDate: "2026-10-26", EndDate: "2026-11-01",
				CreatedMethod: domain.DietPlanCreatedByTemplate,
			},
			wantMsg: "template_id",
		},
		{
			name: "复制方式缺 source_plan_id",
			input: DietPlanCreateInput{
				Name: "S6校验E", AcademicYearID: f.year.ID,
				StartDate: "2026-10-26", EndDate: "2026-11-01",
				CreatedMethod: domain.DietPlanCreatedByCopy,
			},
			wantMsg: "source_plan_id",
		},
		{
			name: "自定义方式不接受模板字段",
			input: DietPlanCreateInput{
				Name: "S6校验F", AcademicYearID: f.year.ID,
				StartDate: "2026-10-26", EndDate: "2026-11-01",
				CreatedMethod: domain.DietPlanCreatedByCustom, TemplateID: &tplID,
			},
			wantMsg: "自定义",
		},
		{
			name: "模板方式不接受来源食谱",
			input: DietPlanCreateInput{
				Name: "S6校验G", AcademicYearID: f.year.ID,
				StartDate: "2026-10-26", EndDate: "2026-11-01",
				CreatedMethod: domain.DietPlanCreatedByTemplate,
				TemplateID:    &tplID, SourcePlanID: &planID,
			},
			wantMsg: "source_plan_id",
		},
		{
			name: "学年不属于本园",
			input: DietPlanCreateInput{
				Name: "S6校验H", AcademicYearID: otherYear.ID,
				StartDate: "2026-10-26", EndDate: "2026-11-01",
				CreatedMethod: domain.DietPlanCreatedByCustom,
			},
			wantMsg: "学年",
		},
		{
			name: "创建方式无效",
			input: DietPlanCreateInput{
				Name: "S6校验I", AcademicYearID: f.year.ID,
				StartDate: "2026-10-26", EndDate: "2026-11-01",
				CreatedMethod: "unknown",
			},
			wantMsg: "创建方式",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := tc.input
			input.GroupObjectID = f.object.ID
			_, err := f.svc.CreatePlan(ctx, testDietPlanSvcKG, input)
			require.Error(t, err)
			var invalid *DietPlanValidationError
			require.ErrorAs(t, err, &invalid)
			assert.Contains(t, invalid.Message, tc.wantMsg)
		})
	}
}

// TC-NEW-DIE-0031 / 0033：模板复制出独立食谱；修改生成结果不影响模板本身。
func TestDietPlanCreateFromTemplateCopiesItems(t *testing.T) {
	f := newDietPlanFixture(t)
	ctx := context.Background()

	noodle := f.createFood(t, testDietPlanSvcKG, "S6面条", 100, dietFloat(286), dietFloat(8.3))
	milk := f.createFood(t, testDietPlanSvcKG, "S6牛奶", 100, dietFloat(54), dietFloat(3))
	tpl := f.createTemplate(t, "s6-template-a", []domain.TemplatePlanItem{
		{Weekday: 1, MealKey: domain.DietMealBreakfast, RecipeName: "牛奶", FoodID: milk.ID, FoodName: "牛奶", Grams: 200, Sort: 1},
		{Weekday: 1, MealKey: domain.DietMealLunch, RecipeName: "肉酱意大利面", FoodID: noodle.ID, FoodName: "面条", Grams: 30, Sort: 1},
		{Weekday: 3, MealKey: domain.DietMealLunch, RecipeName: "肉酱意大利面", FoodID: noodle.ID, FoodName: "面条", Grams: 40, Sort: 2},
	})

	result, err := f.svc.CreatePlan(ctx, testDietPlanSvcKG, DietPlanCreateInput{
		Name:           "第十周",
		AcademicYearID: f.year.ID,
		GroupObjectID:  f.object.ID,
		StartDate:      "2026-11-02",
		EndDate:        "2026-11-08",
		WeekOfYear:     10,
		CreatedMethod:  domain.DietPlanCreatedByTemplate,
		TemplateID:     &tpl.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, result.DayNum, "模板有效天数为有明细的不同日期数")
	assert.Empty(t, result.Warnings)

	detail, err := f.svc.GetPlan(ctx, testDietPlanSvcKG, result.ID)
	require.NoError(t, err)
	require.NotNil(t, detail.SourceTemplateID)
	assert.Equal(t, tpl.ID, *detail.SourceTemplateID)
	require.Len(t, detail.Items, 3)
	assert.Equal(t, "2026-11-02", detail.Items[0].Date)
	assert.Equal(t, 1, detail.Items[0].Weekday)
	assert.Equal(t, "牛奶", detail.Items[0].FoodName)
	assert.Equal(t, "2026-11-02", detail.Items[1].Date)
	assert.Equal(t, "2026-11-04", detail.Items[2].Date)
	assert.Equal(t, 3, detail.Items[2].Weekday)

	// TC-0033：改生成结果（菜名 + 克数），模板条目必须原样保留。
	_, err = f.svc.SavePlan(ctx, testDietPlanSvcKG, result.ID, DietPlanSaveInput{
		Items: []DietPlanItemInput{
			dietPlanItemInput("2026-11-02", domain.DietMealBreakfast, "改名牛奶", milk, 150),
		},
	})
	require.NoError(t, err)

	tplDetail, err := f.svc.GetTemplate(ctx, testDietPlanSvcKG, tpl.ID)
	require.NoError(t, err)
	require.Len(t, tplDetail.Items, 3)
	assert.Equal(t, "牛奶", tplDetail.Items[0].RecipeName)
	assert.InDelta(t, 200.0, tplDetail.Items[0].Grams, 1e-9)
	assert.Equal(t, "肉酱意大利面", tplDetail.Items[1].RecipeName)
	assert.InDelta(t, 30.0, tplDetail.Items[1].Grams, 1e-9)
}

// 模板引用的食物已删除时跳过并写入 warnings；其余条目照常复制。
func TestDietPlanCreateFromTemplateSkipsMissingFood(t *testing.T) {
	f := newDietPlanFixture(t)
	ctx := context.Background()

	milk := f.createFood(t, testDietPlanSvcKG, "S6牛奶", 100, dietFloat(54), dietFloat(3))
	tpl := f.createTemplate(t, "s6-template-missing", []domain.TemplatePlanItem{
		{Weekday: 1, MealKey: domain.DietMealBreakfast, RecipeName: "牛奶", FoodID: milk.ID, FoodName: "牛奶", Grams: 200, Sort: 1},
		{Weekday: 2, MealKey: domain.DietMealLunch, RecipeName: "失踪菜", FoodID: 999999999, FoodName: "已删除食物", Grams: 50, Sort: 1},
	})

	result, err := f.svc.CreatePlan(ctx, testDietPlanSvcKG, DietPlanCreateInput{
		Name:           "S6模板缺食物",
		AcademicYearID: f.year.ID,
		GroupObjectID:  f.object.ID,
		StartDate:      "2026-11-02",
		EndDate:        "2026-11-08",
		CreatedMethod:  domain.DietPlanCreatedByTemplate,
		TemplateID:     &tpl.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.DayNum)
	require.Len(t, result.Warnings, 1)
	assert.Contains(t, result.Warnings[0], "已删除食物")
	assert.Contains(t, result.Warnings[0], "已跳过")
}

// TC-NEW-DIE-0032 / 0034 / 0038：复制按星期映射，并用当前食物库重算营养快照。
func TestDietPlanCopyMapsWeekdayAndRecomputesSnapshot(t *testing.T) {
	f := newDietPlanFixture(t)
	ctx := context.Background()

	pork := f.createFood(t, testDietPlanSvcKG, "S6猪瘦肉", 100, dietFloat(143), dietFloat(20.3))
	noodle := f.createFood(t, testDietPlanSvcKG, "S6面条", 100, dietFloat(286), dietFloat(8.3))
	porkID, noodleID := pork.ID, noodle.ID
	source := f.createSourcePlan(t, "S6源第九周", dietPlanWeek1Start, []domain.MealPlanItem{
		{
			Date: dietPlanWeek1Start, Weekday: 1, MealKey: domain.DietMealLunch,
			RecipeName: "肉酱意大利面", FoodID: &porkID, FoodName: "猪瘦肉", Grams: 30, Sort: 1,
			NutritionSnapshot: `{"edible_percent":100,"energy":143,"protein":20.3}`,
		},
		{
			Date: dietPlanWeek1Start.AddDate(0, 0, 2), Weekday: 3, MealKey: domain.DietMealLunch,
			RecipeName: "肉酱意大利面", FoodID: &noodleID, FoodName: "面条", Grams: 40, Sort: 2,
			NutritionSnapshot: `{"edible_percent":100,"energy":286,"protein":8.3}`,
		},
	})

	// 源食谱保存后食物库被修改：复制必须用当前值，而不是照抄源快照。
	pork.Energy = dietFloat(200)
	pork.Protein = dietFloat(25)
	require.NoError(t, f.foodRepo.Update(ctx, pork))

	result, err := f.svc.CreatePlan(ctx, testDietPlanSvcKG, DietPlanCreateInput{
		Name:           "S6复制第十周",
		AcademicYearID: f.year.ID,
		GroupObjectID:  f.object.ID,
		StartDate:      "2026-11-02",
		EndDate:        "2026-11-08",
		CreatedMethod:  domain.DietPlanCreatedByCopy,
		SourcePlanID:   &source.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, result.DayNum)
	assert.Empty(t, result.Warnings)

	detail, err := f.svc.GetPlan(ctx, testDietPlanSvcKG, result.ID)
	require.NoError(t, err)
	require.NotNil(t, detail.SourcePlanID)
	assert.Equal(t, source.ID, *detail.SourcePlanID)
	require.Len(t, detail.Items, 2)
	assert.Equal(t, "2026-11-02", detail.Items[0].Date)
	assert.Equal(t, 1, detail.Items[0].Weekday)
	assert.InDelta(t, 30.0, detail.Items[0].Grams, 1e-9)
	assert.Equal(t, "2026-11-04", detail.Items[1].Date)
	assert.Equal(t, 3, detail.Items[1].Weekday)
	assert.InDelta(t, 40.0, detail.Items[1].Grams, 1e-9)
	require.NotNil(t, detail.Items[0].NutritionSnapshot.Energy)
	assert.InDelta(t, 200.0, *detail.Items[0].NutritionSnapshot.Energy, 1e-9,
		"复制时必须按当前食物库重算营养快照")

	// 源食谱快照保持不变。
	sourceDetail, err := f.svc.GetPlan(ctx, testDietPlanSvcKG, source.ID)
	require.NoError(t, err)
	require.NotNil(t, sourceDetail.Items[0].NutritionSnapshot.Energy)
	assert.InDelta(t, 143.0, *sourceDetail.Items[0].NutritionSnapshot.Energy, 1e-9)
}

// 复制时源食谱已删除的食物跳过并写入 warnings。
func TestDietPlanCopySkipsDeletedFood(t *testing.T) {
	f := newDietPlanFixture(t)
	ctx := context.Background()

	food := f.createFood(t, testDietPlanSvcKG, "S6将删除食物", 100, dietFloat(100), nil)
	foodID := food.ID
	source := f.createSourcePlan(t, "S6源待删食物", dietPlanWeek1Start, []domain.MealPlanItem{
		{
			Date: dietPlanWeek1Start, Weekday: 1, MealKey: domain.DietMealLunch,
			RecipeName: "待删菜", FoodID: &foodID, FoodName: "将删除食物", Grams: 30, Sort: 1,
			NutritionSnapshot: `{"edible_percent":100,"energy":100}`,
		},
	})
	_, err := f.foodRepo.Delete(ctx, testDietPlanSvcKG, food.ID)
	require.NoError(t, err)

	result, err := f.svc.CreatePlan(ctx, testDietPlanSvcKG, DietPlanCreateInput{
		Name:           "S6复制缺食物",
		AcademicYearID: f.year.ID,
		GroupObjectID:  f.object.ID,
		StartDate:      "2026-11-02",
		EndDate:        "2026-11-08",
		CreatedMethod:  domain.DietPlanCreatedByCopy,
		SourcePlanID:   &source.ID,
	})
	require.NoError(t, err)
	assert.Zero(t, result.DayNum)
	require.Len(t, result.Warnings, 1)
	assert.Contains(t, result.Warnings[0], "已删除")
}

// TC-NEW-DIE-0035 / 0036：保存可增删明细；空数组等同清空。
func TestDietPlanSaveAddsRemovesAndClearsItems(t *testing.T) {
	f := newDietPlanFixture(t)
	ctx := context.Background()

	rice := f.createFood(t, testDietPlanSvcKG, "S6米饭", 100, dietFloat(344), dietFloat(2.6))
	pork := f.createFood(t, testDietPlanSvcKG, "S6猪瘦肉", 100, dietFloat(143), dietFloat(20.3))
	plan := f.createCustomPlan(t, "S6保存增删")

	saved, err := f.svc.SavePlan(ctx, testDietPlanSvcKG, plan.ID, DietPlanSaveInput{
		Items: []DietPlanItemInput{
			dietPlanItemInput("2026-10-26", domain.DietMealLunch, "米饭", rice, 80),
			dietPlanItemInput("2026-10-26", domain.DietMealLunch, "肉酱", pork, 30),
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, saved.ItemCount)
	assert.Equal(t, 1, saved.DayNum)

	detail, err := f.svc.GetPlan(ctx, testDietPlanSvcKG, plan.ID)
	require.NoError(t, err)
	require.Len(t, detail.Items, 2)
	assert.Equal(t, 1, detail.DayNum)

	saved, err = f.svc.SavePlan(ctx, testDietPlanSvcKG, plan.ID, DietPlanSaveInput{
		Items: []DietPlanItemInput{
			dietPlanItemInput("2026-10-27", domain.DietMealLunch, "肉酱", pork, 35),
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, saved.ItemCount)
	detail, err = f.svc.GetPlan(ctx, testDietPlanSvcKG, plan.ID)
	require.NoError(t, err)
	require.Len(t, detail.Items, 1)
	assert.Equal(t, "2026-10-27", detail.Items[0].Date)

	saved, err = f.svc.SavePlan(ctx, testDietPlanSvcKG, plan.ID, DietPlanSaveInput{
		Items: []DietPlanItemInput{},
	})
	require.NoError(t, err)
	assert.Zero(t, saved.ItemCount)
	assert.Zero(t, saved.DayNum)
	detail, err = f.svc.GetPlan(ctx, testDietPlanSvcKG, plan.ID)
	require.NoError(t, err)
	assert.Empty(t, detail.Items)
	assert.Zero(t, detail.DayNum)
}

// TC-NEW-DIE-0037：克数为 0 / 菜名为空 / 餐次未启用 / 食物不在本园时 422 且定位菜名与食物。
func TestDietPlanSaveValidatesCells(t *testing.T) {
	f := newDietPlanFixture(t)
	ctx := context.Background()

	rice := f.createFood(t, testDietPlanSvcKG, "S6米饭", 100, dietFloat(344), dietFloat(2.6))
	otherFood := f.createFood(t, testDietPlanSvcOtherKG, "S6他园食物", 100, dietFloat(100), nil)
	plan := f.createCustomPlan(t, "S6保存校验")

	cases := []struct {
		name    string
		item    DietPlanItemInput
		wantMsg []string
	}{
		{
			name:    "克数为 0",
			item:    dietPlanItemInput("2026-10-26", domain.DietMealLunch, "肉酱意大利面", rice, 0),
			wantMsg: []string{"肉酱意大利面", "S6米饭", "大于 0"},
		},
		{
			name:    "克数为负",
			item:    dietPlanItemInput("2026-10-26", domain.DietMealLunch, "肉酱意大利面", rice, -5),
			wantMsg: []string{"肉酱意大利面", "S6米饭"},
		},
		{
			name:    "菜名为空",
			item:    dietPlanItemInput("2026-10-26", domain.DietMealLunch, "   ", rice, 10),
			wantMsg: []string{"菜名"},
		},
		{
			name:    "餐次未启用",
			item:    dietPlanItemInput("2026-10-26", domain.DietMealEveningSnack, "夜宵", rice, 10),
			wantMsg: []string{"夜宵", "晚点", "未在食谱配置中启用"},
		},
		{
			name:    "日期超出周期",
			item:    dietPlanItemInput("2026-11-02", domain.DietMealLunch, "越界菜", rice, 10),
			wantMsg: []string{"2026-11-02", "周期"},
		},
		{
			name:    "食物不在本园",
			item:    dietPlanItemInput("2026-10-26", domain.DietMealLunch, "跨园菜", otherFood, 10),
			wantMsg: []string{"跨园菜", "S6他园食物", "不在本园食物库"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := f.svc.SavePlan(ctx, testDietPlanSvcKG, plan.ID, DietPlanSaveInput{
				Items: []DietPlanItemInput{tc.item},
			})
			require.Error(t, err)
			var invalid *DietPlanValidationError
			require.ErrorAs(t, err, &invalid)
			for _, want := range tc.wantMsg {
				assert.Contains(t, invalid.Message, want)
			}
		})
	}

	// items 缺字段（nil）与空数组语义不同：前者拒绝，后者清空。
	_, err := f.svc.SavePlan(ctx, testDietPlanSvcKG, plan.ID, DietPlanSaveInput{})
	require.Error(t, err)
	var invalid *DietPlanValidationError
	require.ErrorAs(t, err, &invalid)
	assert.Contains(t, invalid.Message, "items")
}

// 同菜同食物的多行在保存时合并克数（契约规则 4）。
func TestDietPlanSaveMergesSameFoodInSameRecipe(t *testing.T) {
	f := newDietPlanFixture(t)
	ctx := context.Background()

	rice := f.createFood(t, testDietPlanSvcKG, "S6米饭", 100, dietFloat(344), dietFloat(2.6))
	plan := f.createCustomPlan(t, "S6保存合并")

	saved, err := f.svc.SavePlan(ctx, testDietPlanSvcKG, plan.ID, DietPlanSaveInput{
		Items: []DietPlanItemInput{
			dietPlanItemInput("2026-10-26", domain.DietMealLunch, "米饭", rice, 10),
			dietPlanItemInput("2026-10-26", domain.DietMealLunch, "米饭", rice, 20),
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, saved.ItemCount)

	detail, err := f.svc.GetPlan(ctx, testDietPlanSvcKG, plan.ID)
	require.NoError(t, err)
	require.Len(t, detail.Items, 1)
	assert.InDelta(t, 30.0, detail.Items[0].Grams, 1e-9)
	assert.Equal(t, 1, detail.Items[0].Sort)
}

// TC-NEW-DIE-0038：保存时用当前食物库重算快照，food_name 也以服务端为准。
func TestDietPlanSaveRecomputesSnapshotAndFoodName(t *testing.T) {
	f := newDietPlanFixture(t)
	ctx := context.Background()

	food := f.createFood(t, testDietPlanSvcKG, "S6鸡肉", 100, dietFloat(133), dietFloat(19.4))
	plan := f.createCustomPlan(t, "S6保存重算")
	_, err := f.svc.SavePlan(ctx, testDietPlanSvcKG, plan.ID, DietPlanSaveInput{
		Items: []DietPlanItemInput{
			dietPlanItemInput("2026-10-26", domain.DietMealLunch, "宫保鸡丁", food, 30),
		},
	})
	require.NoError(t, err)

	food.Name = "S6鸡胸肉"
	food.EdiblePercent = 80
	food.Energy = dietFloat(250)
	require.NoError(t, f.foodRepo.Update(ctx, food))

	_, err = f.svc.SavePlan(ctx, testDietPlanSvcKG, plan.ID, DietPlanSaveInput{
		Items: []DietPlanItemInput{
			dietPlanItemInput("2026-10-26", domain.DietMealLunch, "宫保鸡丁", food, 30),
		},
	})
	require.NoError(t, err)

	detail, err := f.svc.GetPlan(ctx, testDietPlanSvcKG, plan.ID)
	require.NoError(t, err)
	require.Len(t, detail.Items, 1)
	assert.Equal(t, "S6鸡胸肉", detail.Items[0].FoodName, "食物名以服务端为准")
	assert.InDelta(t, 80.0, detail.Items[0].NutritionSnapshot.EdiblePercent, 1e-9)
	require.NotNil(t, detail.Items[0].NutritionSnapshot.Energy)
	assert.InDelta(t, 250.0, *detail.Items[0].NutritionSnapshot.Energy, 1e-9)
}

// TC-NEW-DIE-0039：sort 由服务端按数组顺序在 (date, meal_key) 内重写并持久化。
func TestDietPlanSavePersistsArrayOrder(t *testing.T) {
	f := newDietPlanFixture(t)
	ctx := context.Background()

	food := f.createFood(t, testDietPlanSvcKG, "S6通用食材", 100, dietFloat(100), nil)
	plan := f.createCustomPlan(t, "S6顺序")

	_, err := f.svc.SavePlan(ctx, testDietPlanSvcKG, plan.ID, DietPlanSaveInput{
		Items: []DietPlanItemInput{
			dietPlanItemInput("2026-10-26", domain.DietMealLunch, "菜B", food, 10),
			dietPlanItemInput("2026-10-26", domain.DietMealLunch, "菜A", food, 20),
			dietPlanItemInput("2026-10-26", domain.DietMealLunch, "菜C", food, 30),
		},
	})
	require.NoError(t, err)

	assertOrder := func(want []string) {
		t.Helper()
		detail, err := f.svc.GetPlan(ctx, testDietPlanSvcKG, plan.ID)
		require.NoError(t, err)
		require.Len(t, detail.Items, len(want))
		got := make([]string, 0, len(detail.Items))
		for i, item := range detail.Items {
			got = append(got, item.RecipeName)
			assert.Equal(t, i+1, item.Sort, "sort 必须按数组顺序重写")
		}
		assert.Equal(t, want, got)
	}
	assertOrder([]string{"菜B", "菜A", "菜C"})

	_, err = f.svc.SavePlan(ctx, testDietPlanSvcKG, plan.ID, DietPlanSaveInput{
		Items: []DietPlanItemInput{
			dietPlanItemInput("2026-10-26", domain.DietMealLunch, "菜C", food, 30),
			dietPlanItemInput("2026-10-26", domain.DietMealLunch, "菜B", food, 10),
			dietPlanItemInput("2026-10-26", domain.DietMealLunch, "菜A", food, 20),
		},
	})
	require.NoError(t, err)
	assertOrder([]string{"菜C", "菜B", "菜A"})
}

// TC-NEW-DIE-0040：保存后列表与详情立即可读（无草稿/发布态）。
func TestDietPlanSaveImmediatelyReadable(t *testing.T) {
	f := newDietPlanFixture(t)
	ctx := context.Background()

	food := f.createFood(t, testDietPlanSvcKG, "S6米饭", 100, dietFloat(344), nil)
	plan := f.createCustomPlan(t, "S6即时生效")
	_, err := f.svc.SavePlan(ctx, testDietPlanSvcKG, plan.ID, DietPlanSaveInput{
		Name: strPtr("S6即时生效（修订）"),
		Items: []DietPlanItemInput{
			dietPlanItemInput("2026-10-26", domain.DietMealLunch, "米饭", food, 80),
			dietPlanItemInput("2026-10-27", domain.DietMealLunch, "米饭", food, 90),
		},
	})
	require.NoError(t, err)

	list, total, _, _, err := f.svc.ListPlans(ctx, testDietPlanSvcKG, f.year.ID, "", pagination.Query{})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, list, 1)
	assert.Equal(t, "S6即时生效（修订）", list[0].Name)
	assert.Equal(t, 2, list[0].DayNum)

	detail, err := f.svc.GetPlan(ctx, testDietPlanSvcKG, plan.ID)
	require.NoError(t, err)
	assert.Len(t, detail.Items, 2)
	assert.Equal(t, 2, detail.DayNum)
}

// TC-NEW-DIE-0041：清空删明细但保留食谱本身。
func TestDietPlanClearKeepsPlan(t *testing.T) {
	f := newDietPlanFixture(t)
	ctx := context.Background()

	food := f.createFood(t, testDietPlanSvcKG, "S6米饭", 100, dietFloat(344), nil)
	plan := f.createCustomPlan(t, "S6清空")
	_, err := f.svc.SavePlan(ctx, testDietPlanSvcKG, plan.ID, DietPlanSaveInput{
		Items: []DietPlanItemInput{
			dietPlanItemInput("2026-10-26", domain.DietMealLunch, "米饭", food, 80),
		},
	})
	require.NoError(t, err)

	cleared, err := f.svc.ClearPlan(ctx, testDietPlanSvcKG, plan.ID)
	require.NoError(t, err)
	assert.Equal(t, plan.ID, cleared.ID)
	assert.Zero(t, cleared.DayNum)

	detail, err := f.svc.GetPlan(ctx, testDietPlanSvcKG, plan.ID)
	require.NoError(t, err)
	assert.NotNil(t, detail)
	assert.Empty(t, detail.Items)
	assert.Zero(t, detail.DayNum)

	list, total, _, _, err := f.svc.ListPlans(ctx, testDietPlanSvcKG, 0, "", pagination.Query{})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total, "清空后食谱仍在列表中")
	require.Len(t, list, 1)
}

// TC-NEW-DIE-0042：软删除后列表与详情都不再返回；跨园按不存在处理。
func TestDietPlanDeleteSoftDeletes(t *testing.T) {
	f := newDietPlanFixture(t)
	ctx := context.Background()

	plan := f.createCustomPlan(t, "S6删除")
	require.NoError(t, f.svc.DeletePlan(ctx, testDietPlanSvcKG, plan.ID))

	list, total, _, _, err := f.svc.ListPlans(ctx, testDietPlanSvcKG, 0, "", pagination.Query{})
	require.NoError(t, err)
	assert.Zero(t, total)
	assert.Empty(t, list)

	_, err = f.svc.GetPlan(ctx, testDietPlanSvcKG, plan.ID)
	require.ErrorIs(t, err, ErrDietPlanNotFound)

	_, err = f.svc.GetPlan(ctx, testDietPlanSvcOtherKG, plan.ID)
	require.ErrorIs(t, err, ErrDietPlanNotFound, "跨园详情按不存在处理")

	err = f.svc.DeletePlan(ctx, testDietPlanSvcKG, plan.ID)
	require.ErrorIs(t, err, ErrDietPlanNotFound)
}

// TC-NEW-DIE-0043 + 黄金值：克数变化只改供给量；浓度快照保持录入当时的值；
// NULL 营养保持 null，不得转 0。
func TestDietPlanNutritionSnapshotGoldenAndNull(t *testing.T) {
	f := newDietPlanFixture(t)
	ctx := context.Background()

	// 30g × 食部 100% × 344 千卡/100g ÷ 100 = 103.2 千卡。
	rice := f.createFood(t, testDietPlanSvcKG, "S6米饭", 100, dietFloat(344), nil)
	assert.InDelta(t, 103.2, *dietNutritionAmount(30, 100, rice.Energy), 1e-9)

	plan := f.createCustomPlan(t, "S6营养黄金值")
	saveWithGrams := func(grams float64) {
		t.Helper()
		_, err := f.svc.SavePlan(ctx, testDietPlanSvcKG, plan.ID, DietPlanSaveInput{
			Items: []DietPlanItemInput{
				dietPlanItemInput("2026-10-26", domain.DietMealLunch, "米饭", rice, grams),
			},
		})
		require.NoError(t, err)
	}

	saveWithGrams(30)
	detail, err := f.svc.GetPlan(ctx, testDietPlanSvcKG, plan.ID)
	require.NoError(t, err)
	require.Len(t, detail.Items, 1)
	snapshot := detail.Items[0].NutritionSnapshot
	assert.InDelta(t, 100.0, snapshot.EdiblePercent, 1e-9)
	require.NotNil(t, snapshot.Energy)
	assert.InDelta(t, 344.0, *snapshot.Energy, 1e-9, "快照存每 100g 浓度值")
	assert.InDelta(t, 103.2, *dietNutritionAmount(detail.Items[0].Grams, snapshot.EdiblePercent, snapshot.Energy), 1e-9)
	assert.Nil(t, snapshot.Protein, "缺测营养必须保持 nil")
	raw, err := json.Marshal(snapshot)
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"protein":null`, "NULL 不得序列化成 0")

	saveWithGrams(20)
	detail, err = f.svc.GetPlan(ctx, testDietPlanSvcKG, plan.ID)
	require.NoError(t, err)
	require.Len(t, detail.Items, 1)
	assert.InDelta(t, 68.8, *dietNutritionAmount(detail.Items[0].Grams,
		detail.Items[0].NutritionSnapshot.EdiblePercent, detail.Items[0].NutritionSnapshot.Energy), 1e-9)
	require.NotNil(t, detail.Items[0].NutritionSnapshot.Energy)
	assert.InDelta(t, 344.0, *detail.Items[0].NutritionSnapshot.Energy, 1e-9,
		"改克数不得改写浓度快照")
}

// DRIs 目标值按小/中/大班在读人数加权：3:4:5 → 0.25/0.3333/0.4167/0。
func TestDietPlanSnapshotWeightsByGradeStudents(t *testing.T) {
	f := newDietPlanFixture(t)
	ctx := context.Background()

	f.seedStudents(t, dietPlanClassGradeJunior, 3)
	f.seedStudents(t, dietPlanClassGradeMiddle, 4)
	f.seedStudents(t, dietPlanClassGradeSenior, 5)

	result := f.createCustomPlan(t, "S6加权快照")
	detail, err := f.svc.GetPlan(ctx, testDietPlanSvcKG, result.ID)
	require.NoError(t, err)
	snapshot := detail.ConfigSnapshot

	assert.Equal(t, domain.DietRatioModeStandard, snapshot.RatioMode)
	assert.InDelta(t, 0.95, snapshot.Coefficient, 1e-9)
	require.Len(t, snapshot.Meals, 6)
	require.Len(t, snapshot.DriWeights, 4)
	assert.InDelta(t, 0.25, snapshot.DriWeights[0].Weight, 1e-9)
	assert.InDelta(t, 0.3333, snapshot.DriWeights[1].Weight, 1e-9)
	assert.InDelta(t, 0.4167, snapshot.DriWeights[2].Weight, 1e-9)
	assert.Zero(t, snapshot.DriWeights[3].Weight, "6 岁档默认权重 0")

	require.Len(t, snapshot.DriTargets, 11)
	values := make(map[string]float64, len(snapshot.DriTargets))
	for _, target := range snapshot.DriTargets {
		values[target.NutrientKey] = target.Value
	}
	for _, key := range []string{
		"energy", "protein", "fats", "carb", "vitamin_a",
		"vitamin_b1", "vitamin_b2", "vitamin_c", "calcium", "zinc", "iron",
	} {
		assert.Contains(t, values, key)
	}
	// 0.25×1350 + 0.3333×1450 + 0.4167×1600 = 1487.505
	assert.InDelta(t, 1487.51, values["energy"], 0.02)
	// 0.25×45 + 0.3333×50 + 0.4167×55 = 50.835
	assert.InDelta(t, 50.84, values["protein"], 0.02)
}

// 无在读学生时回退固定权重 26.7%/32.6%/40.8%/0。
func TestDietPlanSnapshotFallbackWeightsWithoutStudents(t *testing.T) {
	f := newDietPlanFixture(t)
	ctx := context.Background()

	result := f.createCustomPlan(t, "S6回退权重")
	detail, err := f.svc.GetPlan(ctx, testDietPlanSvcKG, result.ID)
	require.NoError(t, err)
	require.Len(t, detail.ConfigSnapshot.DriWeights, 4)
	want := []float64{0.267, 0.326, 0.408, 0}
	for i, w := range want {
		assert.InDelta(t, w, detail.ConfigSnapshot.DriWeights[i].Weight, 1e-9)
	}
	// 0.267×1350 + 0.326×1450 + 0.408×1600 = 1485.95
	var energy float64
	for _, target := range detail.ConfigSnapshot.DriTargets {
		if target.NutrientKey == "energy" {
			energy = target.Value
		}
	}
	assert.InDelta(t, 1485.95, energy, 0.02)
}

// 模板列表/详情按园只读；跨园 404。
func TestDietPlanTemplatesListAndCrossTenant(t *testing.T) {
	f := newDietPlanFixture(t)
	ctx := context.Background()

	food := f.createFood(t, testDietPlanSvcKG, "S6模板食物", 100, dietFloat(100), nil)
	tpl := f.createTemplate(t, "s6-template-list", []domain.TemplatePlanItem{
		{Weekday: 2, MealKey: domain.DietMealLunch, RecipeName: "模板菜", FoodID: food.ID, FoodName: "模板食物", Grams: 50, Sort: 1},
	})

	list, err := f.svc.ListTemplates(ctx, testDietPlanSvcKG)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, tpl.ID, list[0].ID)
	assert.Equal(t, []string{"breakfast", "lunch"}, list[0].MealKeys)
	assert.Equal(t, int64(1), list[0].ItemCount)

	otherList, err := f.svc.ListTemplates(ctx, testDietPlanSvcOtherKG)
	require.NoError(t, err)
	assert.Empty(t, otherList, "模板按园隔离")

	_, err = f.svc.GetTemplate(ctx, testDietPlanSvcOtherKG, tpl.ID)
	require.ErrorIs(t, err, ErrDietTemplateNotFound)
}

// TC-NEW-DIE-0071 / 0072：集体对象必填，且必须属于本园与所选学年；校验失败不落库。
func TestDietPlanCreateRequiresGroupObject(t *testing.T) {
	f := newDietPlanFixture(t)
	ctx := context.Background()

	otherYear := &domain.AcademicYear{
		KindergartenID: testDietPlanSvcKG,
		Year:           "2025-2026",
	}
	require.NoError(t, f.yearRepo.Create(ctx, otherYear))
	otherYearObject := f.createGroupObject(t, otherYear.ID, "上学年全体幼儿", f.allClassIDs()...)

	otherKGYear := &domain.AcademicYear{
		KindergartenID: testDietPlanSvcOtherKG,
		Year:           "2026-2027",
		IsCurrent:      true,
	}
	require.NoError(t, f.yearRepo.Create(ctx, otherKGYear))
	otherKGClass := &domain.Class{
		KindergartenID: testDietPlanSvcOtherKG,
		Name:           "S6他园小班",
		Grade:          dietPlanClassGradeJunior,
		Status:         1,
	}
	require.NoError(t, f.db.Create(otherKGClass).Error)
	otherKGObject, err := f.objectSvc.Create(ctx, testDietPlanSvcOtherKG,
		f.groupObjectInput(otherKGYear.ID, "他园全体幼儿", otherKGClass.ID))
	require.NoError(t, err)

	base := DietPlanCreateInput{
		Name:           "S6对象校验",
		AcademicYearID: f.year.ID,
		StartDate:      "2026-10-26",
		EndDate:        "2026-11-01",
		CreatedMethod:  domain.DietPlanCreatedByCustom,
	}
	cases := []struct {
		name    string
		mutate  func(*DietPlanCreateInput)
		wantMsg string
	}{
		{
			name:    "缺少集体对象",
			mutate:  func(*DietPlanCreateInput) {},
			wantMsg: "请选择集体对象",
		},
		{
			name:    "对象不存在",
			mutate:  func(in *DietPlanCreateInput) { in.GroupObjectID = 1 << 40 },
			wantMsg: "集体对象不存在或不属于本园",
		},
		{
			name:    "对象属于他园",
			mutate:  func(in *DietPlanCreateInput) { in.GroupObjectID = otherKGObject.ID },
			wantMsg: "集体对象不存在或不属于本园",
		},
		{
			name:    "对象属于其他学年",
			mutate:  func(in *DietPlanCreateInput) { in.GroupObjectID = otherYearObject.ID },
			wantMsg: "集体对象不属于所选学年",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := base
			tc.mutate(&input)
			_, err := f.svc.CreatePlan(ctx, testDietPlanSvcKG, input)
			require.Error(t, err)
			var invalid *DietPlanValidationError
			require.ErrorAs(t, err, &invalid)
			assert.Contains(t, invalid.Message, tc.wantMsg)
		})
	}

	var count int64
	require.NoError(t, f.db.Model(&domain.MealPlan{}).
		Where("kindergarten_id = ?", testDietPlanSvcKG).Count(&count).Error)
	assert.Zero(t, count, "对象校验失败不得落库")
}

// TC-NEW-DIE-0073：列表与详情返回所属集体对象 ID 与名称。
func TestDietPlanListAndDetailReturnGroupObject(t *testing.T) {
	f := newDietPlanFixture(t)
	ctx := context.Background()

	dayCare := f.createGroupObject(t, f.year.ID, "日托班",
		f.classesByGrade[dietPlanClassGradeJunior].ID)
	first := f.createCustomPlan(t, "S6对象列表A")
	second, err := f.svc.CreatePlan(ctx, testDietPlanSvcKG, DietPlanCreateInput{
		Name:           "S6对象列表B",
		AcademicYearID: f.year.ID,
		GroupObjectID:  dayCare.ID,
		StartDate:      "2026-11-02",
		EndDate:        "2026-11-08",
		CreatedMethod:  domain.DietPlanCreatedByCustom,
	})
	require.NoError(t, err)

	list, total, _, _, err := f.svc.ListPlans(
		ctx, testDietPlanSvcKG, f.year.ID, "", pagination.Query{})
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	byID := make(map[uint64]DietPlanListItem, len(list))
	for _, item := range list {
		byID[item.ID] = item
	}
	assert.Equal(t, f.object.ID, byID[first.ID].GroupObjectID)
	assert.Equal(t, "全体幼儿", byID[first.ID].GroupObjectName)
	assert.Equal(t, dayCare.ID, byID[second.ID].GroupObjectID)
	assert.Equal(t, "日托班", byID[second.ID].GroupObjectName)

	detail, err := f.svc.GetPlan(ctx, testDietPlanSvcKG, second.ID)
	require.NoError(t, err)
	assert.Equal(t, dayCare.ID, detail.GroupObjectID)
	assert.Equal(t, "日托班", detail.GroupObjectName)
}

// TC-NEW-DIE-0075：对象改名后旧食谱详情仍显示快照原名，列表按当前对象名展示。
func TestDietPlanDetailKeepsSnapshotObjectNameAfterRename(t *testing.T) {
	f := newDietPlanFixture(t)
	ctx := context.Background()

	plan := f.createCustomPlan(t, "S6对象改名")
	_, err := f.objectSvc.Update(ctx, testDietPlanSvcKG, f.object.ID,
		f.groupObjectInput(f.year.ID, "日托幼儿", f.allClassIDs()...))
	require.NoError(t, err)

	detail, err := f.svc.GetPlan(ctx, testDietPlanSvcKG, plan.ID)
	require.NoError(t, err)
	assert.Equal(t, f.object.ID, detail.GroupObjectID)
	assert.Equal(t, "全体幼儿", detail.GroupObjectName, "详情必须保留创建时的对象名称")
	assert.Equal(t, "全体幼儿", detail.ConfigSnapshot.GroupObjectName)

	list, _, _, _, err := f.svc.ListPlans(
		ctx, testDietPlanSvcKG, f.year.ID, "", pagination.Query{})
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "日托幼儿", list[0].GroupObjectName, "列表按当前对象名称展示")
}

// TC-NEW-DIE-0069 / 0076：DRIs 只按对象就餐班级加权；对象改班级后旧食谱快照不回溯，
// 新食谱按新班级人数比例计算。
func TestDietPlanDriWeightsFollowGroupObjectClasses(t *testing.T) {
	f := newDietPlanFixture(t)
	ctx := context.Background()

	f.seedStudents(t, dietPlanClassGradeJunior, 3)
	f.seedStudents(t, dietPlanClassGradeMiddle, 4)
	juniorID := f.classesByGrade[dietPlanClassGradeJunior].ID
	middleID := f.classesByGrade[dietPlanClassGradeMiddle].ID

	juniorOnly := f.createGroupObject(t, f.year.ID, "小班对象", juniorID)
	oldPlan := f.createCustomPlanWithObject(t, "S6小班食谱", juniorOnly.ID)
	oldDetail, err := f.svc.GetPlan(ctx, testDietPlanSvcKG, oldPlan.ID)
	require.NoError(t, err)
	assert.Equal(t, juniorOnly.ID, oldDetail.ConfigSnapshot.GroupObjectID)
	assert.Equal(t, "小班对象", oldDetail.ConfigSnapshot.GroupObjectName)
	assert.Equal(t, []uint64{juniorID}, oldDetail.ConfigSnapshot.GroupObjectClassIDs,
		"快照留档加权范围")
	assert.Equal(t, []float64{1, 0, 0, 0}, dietWeightValues(oldDetail.ConfigSnapshot.DriWeights))
	assert.InDelta(t, 1350.0,
		dietTargetValue(oldDetail.ConfigSnapshot.DriTargets, "energy"), 0.02)
	assert.InDelta(t, 600.0,
		dietTargetValue(oldDetail.ConfigSnapshot.DriTargets, "calcium"), 0.02)

	// 对象改为小 + 中班：新食谱按 3:4 人数加权。
	_, err = f.objectSvc.Update(ctx, testDietPlanSvcKG, juniorOnly.ID,
		f.groupObjectInput(f.year.ID, "小班对象", juniorID, middleID))
	require.NoError(t, err)
	newPlan := f.createCustomPlanWithObject(t, "S6小中班食谱", juniorOnly.ID)
	newDetail, err := f.svc.GetPlan(ctx, testDietPlanSvcKG, newPlan.ID)
	require.NoError(t, err)
	weights := dietWeightValues(newDetail.ConfigSnapshot.DriWeights)
	require.Len(t, weights, 4)
	assert.InDelta(t, 0.4286, weights[0], 1e-9)
	assert.InDelta(t, 0.5714, weights[1], 1e-9)
	assert.Zero(t, weights[2])
	assert.Zero(t, weights[3])
	// 0.4286×1350 + 0.5714×1450 = 1407.14
	assert.InDelta(t, 1407.14,
		dietTargetValue(newDetail.ConfigSnapshot.DriTargets, "energy"), 0.02)

	// 旧食谱快照不随对象改动回溯。
	reloaded, err := f.svc.GetPlan(ctx, testDietPlanSvcKG, oldPlan.ID)
	require.NoError(t, err)
	assert.Equal(t, []float64{1, 0, 0, 0},
		dietWeightValues(reloaded.ConfigSnapshot.DriWeights))
	assert.InDelta(t, 1350.0,
		dietTargetValue(reloaded.ConfigSnapshot.DriTargets, "energy"), 0.02)
}

func dietWeightValues(weights []domain.DietDriWeight) []float64 {
	out := make([]float64, 0, len(weights))
	for _, weight := range weights {
		out = append(out, weight.Weight)
	}
	return out
}

func dietTargetValue(targets []domain.DietDriTargetSnapshot, key string) float64 {
	for _, target := range targets {
		if target.NutrientKey == key {
			return target.Value
		}
	}
	return 0
}

func strPtr(v string) *string { return &v }
