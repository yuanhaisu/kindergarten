package service

import (
	"fmt"
	"strconv"
)

// 自动结论：结构化的 DRIs 占比、三大营养素占比、动植物蛋白比与固定建议话术。
// 话术按营养素硬编码，只要评价命中「低」或「高」就整句输出，与缺口大小无关；
// 「合格」不触发（TC-NEW-DIE-0062）。
//
// 证据范围内只观测到钙、锌两条「偏低」话术（参考产品样本的 analysis 中
// 铁评为「高」却没有对应建议），因此话术目录只登记这两条；未登记的营养素
// 即使判「低/高」也不编造文案。

// dietAnalysisSuggestionTexts 是 [nutrient_key][level] → 固定话术。
var dietAnalysisSuggestionTexts = map[string]map[string]string{
	"calcium": {
		"low": "钙摄入量偏低。建议补充一些牛奶与奶制品鱼类、大豆与豆制品、杏仁、酵母。",
	},
	"zinc": {
		"low": "锌摄入量偏低。建议补充一些牛肉、乳酪、蛋鱼、豆类、果仁、麦牙、粗粮。",
	},
}

type dietAnalysisConclusionInput struct {
	foodKindCount      int
	condimentKindCount int
	compareByKey       map[string]float64
	macroEnergy        DietAnalysisMacroEnergy
	proteinSource      DietAnalysisProteinSource
	comments           map[string]string
}

func buildDietAnalysisConclusion(in dietAnalysisConclusionInput) DietAnalysisConclusion {
	nonCondiment := in.foodKindCount - in.condimentKindCount
	if nonCondiment < 0 {
		nonCondiment = 0
	}

	macroPercent := DietAnalysisMacroPercent{}
	ratioFats, ratioCarb := 0.0, 0.0
	for _, row := range in.macroEnergy.Rows {
		switch row.Key {
		case "percent":
			macroPercent = DietAnalysisMacroPercent{
				Protein: row.Protein,
				Fats:    row.Fats,
				Carb:    row.Carb,
			}
		case "grams_ratio":
			ratioFats = row.Fats
			ratioCarb = row.Carb
		}
	}

	suggestions := make([]DietAnalysisSuggestion, 0, 2)
	for _, spec := range dietAnalysisNutrientSpecs {
		if spec.key == "heat" {
			continue // 千焦与热量同源，结论只按 11 项营养素触发
		}
		level, ok := dietAnalysisSuggestionLevel(in.comments[spec.key])
		if !ok {
			continue
		}
		texts, ok := dietAnalysisSuggestionTexts[spec.key]
		if !ok {
			continue
		}
		text, ok := texts[level]
		if !ok || text == "" {
			continue
		}
		suggestions = append(suggestions, DietAnalysisSuggestion{
			NutrientKey: spec.key,
			Level:       level,
			Text:        text,
		})
	}

	return DietAnalysisConclusion{
		FoodKindCount:         in.foodKindCount,
		CondimentKindCount:    in.condimentKindCount,
		NonCondimentKindCount: nonCondiment,
		DriPercent: DietAnalysisDriPercent{
			Energy:    in.compareByKey["energy"],
			Protein:   in.compareByKey["protein"],
			Fats:      in.compareByKey["fats"],
			Carb:      in.compareByKey["carb"],
			VitaminA:  in.compareByKey["vitamin_a"],
			VitaminB1: in.compareByKey["vitamin_b1"],
			VitaminB2: in.compareByKey["vitamin_b2"],
			VitaminC:  in.compareByKey["vitamin_c"],
			Calcium:   in.compareByKey["calcium"],
			Iron:      in.compareByKey["iron"],
			Zinc:      in.compareByKey["zinc"],
		},
		MacroPercent:             macroPercent,
		MacroRatio:               dietAnalysisRatioText(ratioFats, ratioCarb),
		AnimalBeanProteinPercent: in.proteinSource.AnimalBeanPercent,
		Suggestions:              suggestions,
	}
}

// dietAnalysisSuggestionLevel 把评价映射为结论里的 level；「合格」不触发。
func dietAnalysisSuggestionLevel(comment string) (string, bool) {
	switch comment {
	case "低":
		return "low", true
	case "高":
		return "high", true
	default:
		return "", false
	}
}

func dietAnalysisRatioText(fats, carb float64) string {
	return fmt.Sprintf("1 : %s : %s",
		dietAnalysisFormatNumber(fats), dietAnalysisFormatNumber(carb))
}

// dietAnalysisFormatNumber 去掉无意义的尾零（90.4 而不是 90.40），
// 与参考产品 analysis 文本的写法一致。
func dietAnalysisFormatNumber(v float64) string {
	return strconv.FormatFloat(dietAnalysisRound2(v), 'f', -1, 64)
}

// dietAnalysisConclusionLines 生成导出/打印用的定性分析文本，段落结构与
// 参考产品 analysis 字符串一一对应。
func dietAnalysisConclusionLines(conclusion DietAnalysisConclusion) []string {
	lines := []string{
		fmt.Sprintf("1、本周期内食品种类：%d种，其中调味品种：%d种，食物品种：%d种。",
			conclusion.FoodKindCount, conclusion.CondimentKindCount, conclusion.NonCondimentKindCount),
		"2、本周期DRIs占比分析：",
		fmt.Sprintf("1）热量占比：%s%%，蛋白质占比：%s%%，脂肪占比：%s%%，碳水化合物占比：%s%%",
			dietAnalysisFormatNumber(conclusion.DriPercent.Energy),
			dietAnalysisFormatNumber(conclusion.DriPercent.Protein),
			dietAnalysisFormatNumber(conclusion.DriPercent.Fats),
			dietAnalysisFormatNumber(conclusion.DriPercent.Carb)),
		fmt.Sprintf("2）维生素占比分析：维生素A：%s%%，维生素B1：%s%%，维生素B2：%s%%，维生素C：%s%%",
			dietAnalysisFormatNumber(conclusion.DriPercent.VitaminA),
			dietAnalysisFormatNumber(conclusion.DriPercent.VitaminB1),
			dietAnalysisFormatNumber(conclusion.DriPercent.VitaminB2),
			dietAnalysisFormatNumber(conclusion.DriPercent.VitaminC)),
		fmt.Sprintf("3）矿物质占比分析：钙：%s%%，铁：%s%%，锌：%s%%",
			dietAnalysisFormatNumber(conclusion.DriPercent.Calcium),
			dietAnalysisFormatNumber(conclusion.DriPercent.Iron),
			dietAnalysisFormatNumber(conclusion.DriPercent.Zinc)),
		"3、三大营养素占比分析：",
		fmt.Sprintf("1）蛋白质：%s%%，脂肪：%s%%，碳水化合物：%s%%",
			dietAnalysisFormatNumber(conclusion.MacroPercent.Protein),
			dietAnalysisFormatNumber(conclusion.MacroPercent.Fats),
			dietAnalysisFormatNumber(conclusion.MacroPercent.Carb)),
		fmt.Sprintf("2）【蛋白质 ：脂肪 ： 碳水化合物】的比值：【%s】", conclusion.MacroRatio),
		fmt.Sprintf("4、【动物蛋白质 + 豆类蛋白质】占总蛋白质百分比：%s%%",
			dietAnalysisFormatNumber(conclusion.AnimalBeanProteinPercent)),
	}
	if len(conclusion.Suggestions) > 0 {
		lines = append(lines, "5、营养摄入量分析：")
		for i, suggestion := range conclusion.Suggestions {
			lines = append(lines, fmt.Sprintf("%d）%s", i+1, suggestion.Text))
		}
	}
	return lines
}

// dietAnalysisDisplayedComposition 返回页面/导出实际渲染的膳食构成列。
func dietAnalysisDisplayedComposition(
	groups []DietAnalysisCompositionGroup,
) []DietAnalysisCompositionGroup {
	displayed := make([]DietAnalysisCompositionGroup, 0, 10)
	for _, group := range groups {
		if group.Displayed {
			displayed = append(displayed, group)
		}
	}
	return displayed
}
