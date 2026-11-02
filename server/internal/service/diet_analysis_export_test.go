package service

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

// TC-NEW-DIE-0064：导出 5 个 sheet，数值与接口一致；膳食构成 10 列、脂肪来源 2 列、
// 营养表 14 列、不出现 retinol / carotin。参考导出里的 sheet 名「三餐功能比」是错字，
// 按页面用词写作「三餐供能比」。
func TestDietAnalysisExportWorkbook(t *testing.T) {
	result := buildGoldenResult(t)
	data, err := buildDietAnalysisWorkbook(result)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	workbook, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer func() { _ = workbook.Close() }()

	assert.Equal(t, []string{
		dietAnalysisSheetNutrition,
		dietAnalysisSheetChart,
		dietAnalysisSheetMealEnergy,
		dietAnalysisSheetComposition,
		dietAnalysisSheetCalciumIron,
	}, workbook.GetSheetList())
	for _, name := range workbook.GetSheetList() {
		assert.NotContains(t, name, "功能", "不要复刻参考导出的「三餐功能比」错字")
	}

	// Sheet 1：营养表 14 列（B..O），视黄醇当量复用维生素A，胡萝卜素固定「-」。
	rows, err := workbook.GetRows(dietAnalysisSheetNutrition)
	require.NoError(t, err)
	require.Greater(t, len(rows), 8)
	header := rows[4]
	require.Len(t, header, 15, "标签列 + 14 个营养列")
	assert.Equal(t, "热量（千卡）", header[1])
	assert.Equal(t, "热量（千焦）", header[2])
	assert.Equal(t, "视黄醇当量（微克）", header[6])
	assert.Equal(t, "维生素A（微克）", header[7])
	assert.Equal(t, "胡萝卜素（微克）", header[8])
	assert.Equal(t, "铁（毫克）", header[14])

	intake := rows[5]
	assert.Equal(t, "平均每人每日", intake[0])
	assert.Equal(t, "1341.75", intake[1])
	assert.Equal(t, "5616.57", intake[2])
	assert.Equal(t, "52.46", intake[3])
	assert.Equal(t, "42.74", intake[4])
	assert.Equal(t, "196.55", intake[5])
	assert.Equal(t, "580.74", intake[6], "视黄醇当量列复用维生素A")
	assert.Equal(t, "580.74", intake[7])
	assert.Equal(t, "-", intake[8], "胡萝卜素列固定显示 -")
	assert.Equal(t, "245.66", intake[12])
	assert.Equal(t, "14.98", intake[14])

	dri := rows[6]
	assert.Equal(t, "DRIs", dri[0])
	assert.Equal(t, "1484.26", dri[1])
	assert.Equal(t, "6213.11", dri[2])
	compare := rows[7]
	assert.Equal(t, "比较%", compare[0])
	assert.Equal(t, "90.4", compare[1])
	assert.Equal(t, "124.82", compare[14])

	// 脂肪来源：页面/导出只有动物性、植物性两列，其他脂不在本 sheet。
	assert.Equal(t, "类别", dietAnalysisExportCell(t, workbook, dietAnalysisSheetNutrition, "A28"))
	assert.Equal(t, "动物性", dietAnalysisExportCell(t, workbook, dietAnalysisSheetNutrition, "B28"))
	assert.Equal(t, "植物性", dietAnalysisExportCell(t, workbook, dietAnalysisSheetNutrition, "C28"))
	assert.Empty(t, dietAnalysisExportCell(t, workbook, dietAnalysisSheetNutrition, "D28"))
	assert.Equal(t, "22.38", dietAnalysisExportCell(t, workbook, dietAnalysisSheetNutrition, "B29"))
	assert.Equal(t, "8.48", dietAnalysisExportCell(t, workbook, dietAnalysisSheetNutrition, "C29"))
	assert.Empty(t, dietAnalysisExportCell(t, workbook, dietAnalysisSheetNutrition, "D29"))

	// Sheet 2：图表数据 13 列（不含胡萝卜素）；otherfats 只在这里出现。
	chartRows, err := workbook.GetRows(dietAnalysisSheetChart)
	require.NoError(t, err)
	chartHeader := chartRows[2]
	require.Len(t, chartHeader, 13)
	assert.Equal(t, "热量(千卡)", chartHeader[0])
	assert.Equal(t, "铁(毫克)", chartHeader[12])
	require.Len(t, chartRows[3], 13, "图表数据行也不含胡萝卜素列")
	assert.NotContains(t, strings.Join(chartHeader, "|"), "胡萝卜素")
	assert.Equal(t, "22.38", dietAnalysisExportCell(t, workbook, dietAnalysisSheetChart, "B13"))
	assert.Equal(t, "8.48", dietAnalysisExportCell(t, workbook, dietAnalysisSheetChart, "C13"))
	assert.Equal(t, "11.88", dietAnalysisExportCell(t, workbook, dietAnalysisSheetChart, "D13"),
		"其他脂只在营养分析图数据 sheet 出现")

	// Sheet 3：6 个餐次，名称按页面用「三餐供能比」。
	assert.Equal(t, "类别", dietAnalysisExportCell(t, workbook, dietAnalysisSheetMealEnergy, "A3"))
	assert.Equal(t, "早餐", dietAnalysisExportCell(t, workbook, dietAnalysisSheetMealEnergy, "B3"))
	assert.Equal(t, "晚点", dietAnalysisExportCell(t, workbook, dietAnalysisSheetMealEnergy, "G3"))
	assert.Equal(t, "174.88", dietAnalysisExportCell(t, workbook, dietAnalysisSheetMealEnergy, "B4"))
	assert.Equal(t, "11.78", dietAnalysisExportCell(t, workbook, dietAnalysisSheetMealEnergy, "B5"))
	assert.Equal(t, "0", dietAnalysisExportCell(t, workbook, dietAnalysisSheetMealEnergy, "G4"))
	assert.Equal(t, "85.88", dietAnalysisFormatNumber(result.MealEnergy.TotalPercent),
		"三餐比例合计 85.88%，不归一化")

	// Sheet 4：膳食构成只导出 10 列（A..K）。
	compositionRows, err := workbook.GetRows(dietAnalysisSheetComposition)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(compositionRows), 4)
	compositionHeader := compositionRows[2]
	require.Len(t, compositionHeader, 11, "类别 + 10 类")
	assert.Equal(t, "谷类", compositionHeader[1])
	assert.Equal(t, "油脂类", compositionHeader[10])
	joined := strings.Join(compositionRows[2], "|") + strings.Join(compositionRows[3], "|")
	for _, hidden := range []string{"坚果", "调味品", "糖果蜜饯", "速食小吃", "饮料", "其他"} {
		assert.NotContains(t, joined, hidden, "%s 不应出现在导出构成表", hidden)
	}

	// Sheet 5：钙铁来源 6 列。
	assert.Equal(t, "钙-植物性", dietAnalysisExportCell(t, workbook, dietAnalysisSheetCalciumIron, "B3"))
	assert.Equal(t, "钙-动物性", dietAnalysisExportCell(t, workbook, dietAnalysisSheetCalciumIron, "C3"))
	assert.Equal(t, "钙-奶类及其制品", dietAnalysisExportCell(t, workbook, dietAnalysisSheetCalciumIron, "D3"))
	assert.Equal(t, "铁-肉类", dietAnalysisExportCell(t, workbook, dietAnalysisSheetCalciumIron, "E3"))
	assert.Equal(t, "铁-非肉类", dietAnalysisExportCell(t, workbook, dietAnalysisSheetCalciumIron, "F3"))
	require.Len(t, dietAnalysisExportRows(t, workbook, dietAnalysisSheetCalciumIron)[3], 6)

	// 全工作簿不得出现 retinol / carotin（英文键）。
	for _, sheet := range workbook.GetSheetList() {
		sheetRows, err := workbook.GetRows(sheet)
		require.NoError(t, err)
		for _, row := range sheetRows {
			for _, cell := range row {
				lower := strings.ToLower(cell)
				assert.NotContains(t, lower, "retinol", "%s 不得导出 retinol", sheet)
				assert.NotContains(t, lower, "carotin", "%s 不得导出 carotin", sheet)
			}
		}
	}
}

func dietAnalysisExportCell(t *testing.T, workbook *excelize.File, sheet, cell string) string {
	t.Helper()
	value, err := workbook.GetCellValue(sheet, cell)
	require.NoError(t, err)
	return value
}

func dietAnalysisExportRows(t *testing.T, workbook *excelize.File, sheet string) [][]string {
	t.Helper()
	rows, err := workbook.GetRows(sheet)
	require.NoError(t, err)
	return rows
}
