package service

import (
	"context"
	"math"
	"sort"

	"kindergarten-server/internal/domain"
)

// 本文件负责周食谱创建时固化的 DRIs 目标值，以及明细营养快照的换算口径。
// 快照一旦写入就不随供餐配置、学生名册或食物库回溯（design D3/D5、ADR-0006）。

const (
	// dietPlanClassListLimit 取本园全部班级用于人数加权；园所班级数远小于此上限。
	// 仓储的 List 是分页接口，传 0 在部分 GORM 版本会被当作 LIMIT 0。
	dietPlanClassListLimit = 10000

	// dietPlanClassGrade* 是 Class.Grade 的既有取值，来源 class_service.validateGrade；
	// 未识别的年级（含空串）不计入加权，避免把无年龄信息的班级摊到三档之一。
	dietPlanClassGradeJunior = "小班"
	dietPlanClassGradeMiddle = "中班"
	dietPlanClassGradeSenior = "大班"
)

// dietGradeAgeBand 把班级年级映射到 DRIs 年龄档：小→3、中→4、大→5。
// 6 岁档没有对应班级，权重恒为 0；这是 design D3 的既定口径，不是遗漏。
var dietGradeAgeBand = map[string]int{
	dietPlanClassGradeJunior: 3,
	dietPlanClassGradeMiddle: 4,
	dietPlanClassGradeSenior: 5,
}

// dietDriFallbackWeight 是园所没有在读学生时的固定权重（design D3）。
var dietDriFallbackWeight = map[int]float64{
	3: 0.267,
	4: 0.326,
	5: 0.408,
	6: 0,
}

// buildDietDriSnapshot 计算按班级人数加权的 11 项 DRIs 目标值与四档权重。
//
// 权重先按 4 位小数取整、再用取整后的权重计算目标值，保证快照里
// 「权重 × 各档值」能复核出目标值；目标值保留 2 位小数。
// DRIs 表未播种时返回空目标值与权重（不阻断创建），报表侧可据此标注缺失。
func (s *DietPlanService) buildDietDriSnapshot(
	ctx context.Context,
	kindergartenID uint64,
) ([]domain.DietDriTargetSnapshot, []domain.DietDriWeight, error) {
	weights, err := s.dietDriWeights(ctx, kindergartenID)
	if err != nil {
		return nil, nil, err
	}

	values := make(map[int]map[string]domain.DriAgeTarget, len(dietDriAgeBands))
	sortIndex := make(map[string]int)
	for _, ageBand := range dietDriAgeBands {
		rows, err := s.dris.ListByAge(ctx, ageBand)
		if err != nil {
			return nil, nil, err
		}
		for _, row := range rows {
			if values[ageBand] == nil {
				values[ageBand] = make(map[string]domain.DriAgeTarget, len(rows))
			}
			values[ageBand][row.NutrientKey] = row
			// 同一营养素在各档的 sort 一致；取首次出现的值作为输出顺序。
			if _, ok := sortIndex[row.NutrientKey]; !ok {
				sortIndex[row.NutrientKey] = row.Sort
			}
		}
	}

	keys := make([]string, 0, len(sortIndex))
	for key := range sortIndex {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if sortIndex[keys[i]] != sortIndex[keys[j]] {
			return sortIndex[keys[i]] < sortIndex[keys[j]]
		}
		return keys[i] < keys[j]
	})

	targets := make([]domain.DietDriTargetSnapshot, 0, len(keys))
	for _, key := range keys {
		var label, unit string
		sum := 0.0
		for i, ageBand := range dietDriAgeBands {
			row, ok := values[ageBand][key]
			if !ok {
				continue
			}
			if label == "" {
				label = row.Label
				unit = row.Unit
			}
			sum += weights[i].Weight * row.Value
		}
		targets = append(targets, domain.DietDriTargetSnapshot{
			NutrientKey: key,
			Label:       label,
			Unit:        unit,
			Value:       roundDietTargetValue(sum),
		})
	}
	return targets, weights, nil
}

// dietDriWeights 按小/中/大班在读人数计算四档权重。
//
// 只统计能映射到年龄档的班级；没有这类学生时回退固定权重 26.7%/32.6%/40.8%/0。
// 使用 CountStudentsByClassIDs 的 status=1 口径，与「在读学生」一致。
func (s *DietPlanService) dietDriWeights(
	ctx context.Context,
	kindergartenID uint64,
) ([]domain.DietDriWeight, error) {
	classes, _, err := s.classes.List(ctx, kindergartenID, "", "", 0, dietPlanClassListLimit)
	if err != nil {
		return nil, err
	}

	classIDs := make([]uint64, 0, len(classes))
	for _, class := range classes {
		if _, ok := dietGradeAgeBand[class.Grade]; ok {
			classIDs = append(classIDs, class.ID)
		}
	}
	counts, err := s.classes.CountStudentsByClassIDs(ctx, kindergartenID, classIDs)
	if err != nil {
		return nil, err
	}

	studentsByBand := make(map[int]int64, len(dietDriAgeBands))
	var total int64
	for _, class := range classes {
		ageBand, ok := dietGradeAgeBand[class.Grade]
		if !ok {
			continue
		}
		n := counts[class.ID]
		studentsByBand[ageBand] += n
		total += n
	}

	weights := make([]domain.DietDriWeight, 0, len(dietDriAgeBands))
	for _, ageBand := range dietDriAgeBands {
		weight := dietDriFallbackWeight[ageBand]
		if total > 0 {
			weight = roundDietWeight(float64(studentsByBand[ageBand]) / float64(total))
		}
		weights = append(weights, domain.DietDriWeight{AgeBand: ageBand, Weight: weight})
	}
	return weights, nil
}

// dietNutritionAmount 计算一行明细的营养供给量：
// 克数 × 食部% × 每 100g 浓度 ÷ 100。NULL 保持 nil——报表求和必须跳过，不得当 0。
// 周食谱只存浓度快照，本函数是编辑器小计与营养分析共用的换算口径。
func dietNutritionAmount(grams, ediblePercent float64, per100g *float64) *float64 {
	if per100g == nil {
		return nil
	}
	value := grams * (ediblePercent / 100) * (*per100g) / 100
	return &value
}

func roundDietWeight(v float64) float64 {
	return math.Round(v*10000) / 10000
}

func roundDietTargetValue(v float64) float64 {
	return math.Round(v*100) / 100
}
