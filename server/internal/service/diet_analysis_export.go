package service

import (
	"context"
	"fmt"

	"github.com/xuri/excelize/v2"
)

// 营养分析 Excel 导出：5 个 sheet，内容与页面 4 页签一致。
// 刻意保留的渲染口径：营养表 14 列（视黄醇当量复用维生素A，胡萝卜素写「-」）；
// 膳食构成只导出 10 列；脂肪来源只导出动物性/植物性 2 列；otherfats 只进
// 「营养分析图」数据 sheet；不出现 retinol / carotin。

const (
	dietAnalysisSheetNutrition   = "带量食谱营养分析表"
	dietAnalysisSheetChart       = "带量食谱营养分析图"
	dietAnalysisSheetMealEnergy  = "带量食谱三餐供能比"
	dietAnalysisSheetComposition = "带量食谱膳食构成表"
	dietAnalysisSheetCalciumIron = "带量食谱钙铁来源"
)

// ExportExcel 同步生成 xlsx；失败时仍返回 error 由 handler 转成 JSON 错误包络。
func (s *DietAnalysisService) ExportExcel(
	ctx context.Context,
	kindergartenID, planID uint64,
) ([]byte, error) {
	result, err := s.Analyze(ctx, kindergartenID, planID)
	if err != nil {
		return nil, err
	}
	return buildDietAnalysisWorkbook(result)
}

func buildDietAnalysisWorkbook(result *DietAnalysisResult) ([]byte, error) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	first := f.GetSheetName(0)
	if err := f.SetSheetName(first, dietAnalysisSheetNutrition); err != nil {
		return nil, err
	}
	for _, name := range []string{
		dietAnalysisSheetChart,
		dietAnalysisSheetMealEnergy,
		dietAnalysisSheetComposition,
		dietAnalysisSheetCalciumIron,
	} {
		if _, err := f.NewSheet(name); err != nil {
			return nil, err
		}
	}

	if err := writeDietAnalysisNutritionSheet(f, result); err != nil {
		return nil, err
	}
	if err := writeDietAnalysisChartSheet(f, result); err != nil {
		return nil, err
	}
	if err := writeDietAnalysisMealEnergySheet(f, result); err != nil {
		return nil, err
	}
	if err := writeDietAnalysisCompositionSheet(f, result); err != nil {
		return nil, err
	}
	if err := writeDietAnalysisCalciumIronSheet(f, result); err != nil {
		return nil, err
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeDietAnalysisNutritionSheet(f *excelize.File, result *DietAnalysisResult) error {
	const sheet = dietAnalysisSheetNutrition
	rows := [][]any{
		{"带量食谱营养分析表"},
		{fmt.Sprintf("食谱名称：%s", result.Plan.Name)},
		{
			fmt.Sprintf("周期：%s 至 %s", result.Plan.StartDate, result.Plan.EndDate),
			fmt.Sprintf("餐次系数：%s", dietAnalysisFormatNumber(result.Plan.Coefficient)),
			fmt.Sprintf("有食谱天数：%d", result.Plan.DayNum),
		},
		{"营养素摄入量"},
	}
	for i, row := range rows {
		if err := setDietAnalysisRow(f, sheet, i+1, row); err != nil {
			return err
		}
	}

	// 营养表 14 列：热量（千卡/千焦）+ 蛋白质/脂肪/碳水 + 视黄醇当量/维生素A/胡萝卜素
	// + 维生素B1/B2/C + 钙/锌/铁。视黄醇当量列复用 vitamin_a，胡萝卜素列固定「-」。
	header := []any{"", "热量（千卡）", "热量（千焦）", "蛋白质（克）",
		"脂肪（克）", "碳水化合物（克）", "视黄醇当量（微克）", "维生素A（微克）",
		"胡萝卜素（微克）", "维生素B1（毫克）", "维生素B2（毫克）", "维生素C（毫克）",
		"钙（毫克）", "锌（毫克）", "铁（毫克）"}
	if err := setDietAnalysisRow(f, sheet, 5, header); err != nil {
		return err
	}

	labels := []string{"平均每人每日", "DRIs", "比较%"}
	valueRows := make([][]any, 0, len(labels))
	for _, field := range []string{"intake", "dri", "compare_percent"} {
		valueRows = append(valueRows, dietAnalysisNutritionValues(result, field, true))
	}
	for i, label := range labels {
		row := append([]any{label}, valueRows[i]...)
		if err := setDietAnalysisRow(f, sheet, 6+i, row); err != nil {
			return err
		}
	}

	if err := setDietAnalysisRow(f, sheet, 10, []any{"三大营养素供能"}); err != nil {
		return err
	}
	if err := setDietAnalysisRow(f, sheet, 11, []any{"类别", "蛋白质", "脂肪", "碳水化合物"}); err != nil {
		return err
	}
	macro := dietAnalysisMacroRows(result)
	for i, row := range macro {
		if err := setDietAnalysisRow(f, sheet, 12+i, row); err != nil {
			return err
		}
	}

	if err := setDietAnalysisRow(f, sheet, 22, []any{"蛋白质来源"}); err != nil {
		return err
	}
	if err := setDietAnalysisRow(f, sheet, 23,
		[]any{"类别", "动物蛋白", "豆类蛋白", "要求"}); err != nil {
		return err
	}
	protein := result.ProteinSource
	if err := setDietAnalysisRow(f, sheet, 24, []any{
		"摄入量（克）", protein.Items[0].Value, protein.Items[1].Value, protein.Items[2].Value,
	}); err != nil {
		return err
	}
	if err := setDietAnalysisRow(f, sheet, 25, []any{
		"占蛋白质总量%", protein.Items[0].Percent, protein.Items[1].Percent, protein.Items[2].Percent,
	}); err != nil {
		return err
	}

	if err := setDietAnalysisRow(f, sheet, 27, []any{"脂肪来源"}); err != nil {
		return err
	}
	// 页面/导出只渲染动物性、植物性两列；其他脂只进「营养分析图」sheet。
	displayedFats := displayedDietAnalysisFats(result.FatSource)
	if err := setDietAnalysisRow(f, sheet, 28, append([]any{"类别"},
		dietAnalysisFatLabels(displayedFats)...)); err != nil {
		return err
	}
	if err := setDietAnalysisRow(f, sheet, 29, append([]any{"摄入量（克）"},
		dietAnalysisFatValues(displayedFats, true)...)); err != nil {
		return err
	}
	if err := setDietAnalysisRow(f, sheet, 30, append([]any{"构成比%"},
		dietAnalysisFatValues(displayedFats, false)...)); err != nil {
		return err
	}

	if err := setDietAnalysisRow(f, sheet, 32, []any{"定性分析结论"}); err != nil {
		return err
	}
	for i, line := range dietAnalysisConclusionLines(result.Conclusion) {
		if err := setDietAnalysisRow(f, sheet, 33+i, []any{line}); err != nil {
			return err
		}
	}
	return nil
}

func writeDietAnalysisChartSheet(f *excelize.File, result *DietAnalysisResult) error {
	const sheet = dietAnalysisSheetChart
	if err := setDietAnalysisRow(f, sheet, 1, []any{"带量食谱营养分析图"}); err != nil {
		return err
	}
	// 图数据 sheet 的 13 列不含胡萝卜素（参考导出的口径）。
	header := []any{"热量(千卡)", "热量(千焦)", "蛋白质(克)", "脂肪(克)", "碳水化合物(克)",
		"视黄醇当量(微克)", "维生素A(微克)", "维生素B1(毫克)", "维生素B2(毫克)",
		"维生素C(毫克)", "钙(毫克)", "锌(毫克)", "铁(毫克)"}
	if err := setDietAnalysisRow(f, sheet, 3, header); err != nil {
		return err
	}
	if err := setDietAnalysisRow(f, sheet, 4, dietAnalysisNutritionValues(result, "compare_percent", false)); err != nil {
		return err
	}

	proteinKcal := dietAnalysisNutritionIntake(result, "protein") * dietAnalysisProteinKcal
	fatsKcal := dietAnalysisNutritionIntake(result, "fats") * dietAnalysisFatsKcal
	carbKcal := dietAnalysisNutritionIntake(result, "carb") * dietAnalysisCarbKcal
	if err := setDietAnalysisRow(f, sheet, 6,
		[]any{"三大营养素供能（千卡）", "脂肪", "蛋白质", "碳水化合物"}); err != nil {
		return err
	}
	if err := setDietAnalysisRow(f, sheet, 7, []any{nil,
		dietAnalysisRound2(fatsKcal), dietAnalysisRound2(proteinKcal), dietAnalysisRound2(carbKcal)}); err != nil {
		return err
	}

	// 蛋白质饼图第三块 = 要求 × 2 − 动物 − 豆类（页面 initProteinPieData 的口径）。
	protein := result.ProteinSource
	proteinOther := 0.0
	if len(protein.Items) == 3 {
		proteinOther = dietAnalysisRound2(
			protein.Items[2].Value*2 - protein.Items[0].Value - protein.Items[1].Value)
	}
	if err := setDietAnalysisRow(f, sheet, 9,
		[]any{"蛋白质来源（克）", "动物性食物", "豆类", "其他", "要求"}); err != nil {
		return err
	}
	if err := setDietAnalysisRow(f, sheet, 10, []any{nil,
		protein.Items[0].Value, protein.Items[1].Value, proteinOther, protein.Items[2].Value}); err != nil {
		return err
	}

	// otherfats 只在这个图表数据 sheet 出现。
	fats := result.FatSource
	if err := setDietAnalysisRow(f, sheet, 12,
		[]any{"脂肪来源（克）", "动物性", "植物性", "其他"}); err != nil {
		return err
	}
	if err := setDietAnalysisRow(f, sheet, 13, []any{nil,
		fats.Items[0].Value, fats.Items[1].Value, fats.Items[2].Value}); err != nil {
		return err
	}
	return nil
}

func writeDietAnalysisMealEnergySheet(f *excelize.File, result *DietAnalysisResult) error {
	const sheet = dietAnalysisSheetMealEnergy
	if err := setDietAnalysisRow(f, sheet, 1, []any{"带量食谱三餐供能比"}); err != nil {
		return err
	}
	labels := []any{"类别"}
	energies := []any{"能量（千卡）"}
	percents := []any{"比例"}
	for _, meal := range result.MealEnergy.Meals {
		labels = append(labels, meal.Label)
		energies = append(energies, meal.Energy)
		percents = append(percents, meal.Percent)
	}
	if err := setDietAnalysisRow(f, sheet, 3, labels); err != nil {
		return err
	}
	if err := setDietAnalysisRow(f, sheet, 4, energies); err != nil {
		return err
	}
	if err := setDietAnalysisRow(f, sheet, 5, percents); err != nil {
		return err
	}
	return nil
}

func writeDietAnalysisCompositionSheet(f *excelize.File, result *DietAnalysisResult) error {
	const sheet = dietAnalysisSheetComposition
	if err := setDietAnalysisRow(f, sheet, 1, []any{"带量食谱膳食构成表"}); err != nil {
		return err
	}
	// 页面与导出都只渲染 10 类，坚果/调味品等 6 类只有接口有数据。
	header := []any{"类别"}
	values := []any{"摄入量（克）"}
	for _, group := range dietAnalysisDisplayedComposition(result.Composition.Groups) {
		header = append(header, group.Label)
		values = append(values, group.Grams)
	}
	if err := setDietAnalysisRow(f, sheet, 3, header); err != nil {
		return err
	}
	return setDietAnalysisRow(f, sheet, 4, values)
}

func writeDietAnalysisCalciumIronSheet(f *excelize.File, result *DietAnalysisResult) error {
	const sheet = dietAnalysisSheetCalciumIron
	if err := setDietAnalysisRow(f, sheet, 1, []any{"带量食谱钙铁来源"}); err != nil {
		return err
	}
	calcium := result.CalciumIron.Calcium.Sources
	iron := result.CalciumIron.Iron.Sources
	// 顺序与页面一致：植物性、动物性、奶类及其制品；铁：肉类、非肉类。
	if err := setDietAnalysisRow(f, sheet, 3, []any{
		"来源", "钙-植物性", "钙-动物性", "钙-奶类及其制品", "铁-肉类", "铁-非肉类",
	}); err != nil {
		return err
	}
	if err := setDietAnalysisRow(f, sheet, 4, []any{
		"重量（毫克）", calcium[1].Value, calcium[0].Value, calcium[2].Value,
		iron[0].Value, iron[1].Value,
	}); err != nil {
		return err
	}
	return setDietAnalysisRow(f, sheet, 5, []any{
		"构成比%", calcium[1].Percent, calcium[0].Percent, calcium[2].Percent,
		iron[0].Percent, iron[1].Percent,
	})
}

// dietAnalysisNutritionValues 按列顺序取值：热量/热量/蛋白/脂肪/碳水/
// 视黄醇当量(复用维生素A)/维生素A/胡萝卜素(-)/B1/B2/C/钙/锌/铁。
// withCarotin=false 时输出 13 列（图表数据 sheet 没有胡萝卜素）。
func dietAnalysisNutritionValues(
	result *DietAnalysisResult,
	field string,
	withCarotin bool,
) []any {
	byKey := make(map[string]DietAnalysisNutritionItem, len(result.Nutrition.Items))
	for _, item := range result.Nutrition.Items {
		byKey[item.Key] = item
	}
	value := func(key string) any {
		item := byKey[key]
		switch field {
		case "intake":
			return item.Intake
		case "dri":
			return item.Dri
		default:
			return item.ComparePercent
		}
	}
	vitaminA := value("vitamin_a")
	values := []any{
		value("energy"), value("heat"), value("protein"), value("fats"), value("carb"),
		vitaminA, vitaminA,
	}
	if withCarotin {
		values = append(values, "-")
	}
	return append(values,
		value("vitamin_b1"), value("vitamin_b2"), value("vitamin_c"),
		value("calcium"), value("zinc"), value("iron"))
}

func dietAnalysisNutritionIntake(result *DietAnalysisResult, key string) float64 {
	for _, item := range result.Nutrition.Items {
		if item.Key == key {
			return item.Intake
		}
	}
	return 0
}

// dietAnalysisMacroRows 输出三大营养素供能的 9 行：前两行是接口里的「要求」值，
// 后四行补充页面需要的「现状」派生值与分母，保证导出可直接阅读。
func dietAnalysisMacroRows(result *DietAnalysisResult) [][]any {
	var kcal, kj, percent, ratio DietAnalysisMacroRow
	for _, row := range result.MacroEnergy.Rows {
		switch row.Key {
		case "kcal":
			kcal = row
		case "kj":
			kj = row
		case "percent":
			percent = row
		case "grams_ratio":
			ratio = row
		}
	}
	proteinKcal := dietAnalysisRound2(dietAnalysisNutritionIntake(result, "protein") * dietAnalysisProteinKcal)
	fatsKcal := dietAnalysisRound2(dietAnalysisNutritionIntake(result, "fats") * dietAnalysisFatsKcal)
	carbKcal := dietAnalysisRound2(dietAnalysisNutritionIntake(result, "carb") * dietAnalysisCarbKcal)

	return [][]any{
		{"千卡-要求", kcal.Protein, kcal.Fats, kcal.Carb},
		{"千卡-现状", proteinKcal, fatsKcal, carbKcal},
		{"千焦-要求", kj.Protein, kj.Fats, kj.Carb},
		{"千焦-现状",
			dietAnalysisRound2(proteinKcal * dietAnalysisHeatFactor),
			dietAnalysisRound2(fatsKcal * dietAnalysisHeatFactor),
			dietAnalysisRound2(carbKcal * dietAnalysisHeatFactor)},
		{"占总热量%-要求", percent.Recommended.Protein, percent.Recommended.Fats, percent.Recommended.Carb},
		{"占总热量%-现状", percent.Protein, percent.Fats, percent.Carb},
		{"营养素比例-要求", ratio.Recommended.Protein, ratio.Recommended.Fats, ratio.Recommended.Carb},
		{"营养素比例-现状", ratio.Protein, ratio.Fats, ratio.Carb},
		{"占总热量%分母", result.MacroEnergy.Denominator},
	}
}

func displayedDietAnalysisFats(source DietAnalysisFatSource) []DietAnalysisFatItem {
	displayed := make([]DietAnalysisFatItem, 0, 2)
	for _, item := range source.Items {
		if item.Displayed {
			displayed = append(displayed, item)
		}
	}
	return displayed
}

func dietAnalysisFatLabels(items []DietAnalysisFatItem) []any {
	labels := make([]any, 0, len(items))
	for _, item := range items {
		labels = append(labels, item.Label)
	}
	return labels
}

func dietAnalysisFatValues(items []DietAnalysisFatItem, value bool) []any {
	values := make([]any, 0, len(items))
	for _, item := range items {
		if value {
			values = append(values, item.Value)
			continue
		}
		values = append(values, item.Percent)
	}
	return values
}

func setDietAnalysisRow(f *excelize.File, sheet string, row int, values []any) error {
	for col, value := range values {
		if value == nil {
			continue
		}
		cell, err := excelize.CoordinatesToCellName(col+1, row)
		if err != nil {
			return err
		}
		if err := f.SetCellValue(sheet, cell, value); err != nil {
			return err
		}
	}
	return nil
}
