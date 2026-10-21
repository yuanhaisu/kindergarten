package service

import "kindergarten-server/internal/domain"

// AbnormalItem 晨检异常项的可读展示。
//
// 与 symptom_tags 的区别：tags 是统计口径（11 个受控标签），
// AbnormalItem 是给人看的（含 3=其他 这类没有对应标签的取值）。
type AbnormalItem struct {
	Part        string `json:"part"`
	PartLabel   string `json:"part_label"`
	StatusLabel string `json:"status_label"`
}

// normalizeDate MySQL DATE 列在 parseTime=true 下可能被驱动转成
// "2026-10-18T00:00:00+08:00"；对外只保留日期部分。
func normalizeDate(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}

func overallStatusLabel(v domain.HealthCheckOverall) string {
	switch v {
	case domain.HealthOverallObserve:
		return "需观察"
	case domain.HealthOverallReject:
		return "劝返就医"
	default:
		return "健康"
	}
}

// readableAbnormalItems 把五项检查值翻译成家长可读的异常项。
// 0=正常不返回；未知/越界取值兜底为「其他异常」，避免历史脏数据静默丢失。
func readableAbnormalItems(check *domain.MorningHealthCheck) []AbnormalItem {
	items := make([]AbnormalItem, 0, 5)
	appendItem := func(part, partLabel string, v domain.HealthCheckItem, labels map[domain.HealthCheckItem]string) {
		if v == 0 {
			return
		}
		label, ok := labels[v]
		if !ok {
			label = "其他异常"
		}
		items = append(items, AbnormalItem{Part: part, PartLabel: partLabel, StatusLabel: label})
	}

	appendItem("throat", "喉咙", check.ThroatStatus, map[domain.HealthCheckItem]string{
		1: "发红", 2: "红肿",
	})
	appendItem("hand", "手部", check.HandStatus, map[domain.HealthCheckItem]string{
		1: "红疹", 2: "疱疹", 3: "其他异常",
	})
	appendItem("mouth", "口腔", check.MouthStatus, map[domain.HealthCheckItem]string{
		1: "溃疡", 2: "疱疹", 3: "其他异常",
	})
	appendItem("eye", "眼睛", check.EyeStatus, map[domain.HealthCheckItem]string{
		1: "发红", 2: "分泌物",
	})
	appendItem("skin", "皮肤", check.SkinStatus, map[domain.HealthCheckItem]string{
		1: "红疹", 2: "水疱", 3: "其他异常",
	})
	return items
}
