package service

import (
	"fmt"
	"strings"
	"time"

	"kindergarten-server/internal/domain"
)

// 接送登记的判定规则集中在此，handler 只透传结果（design D5）。
//
// 这些是纯函数：输入已知事实（当日已有记录数、最后一条方向、班级放学时刻、当前时刻），
// 输出该次登记是否被允许、理由是否必填。DB 只负责取事实，不做判定。
// 理由：真正会写错的是这些边界，且它们必须可单测（项目测试惯例见 SamePerson）。

// DefaultPickupDeadline 班级未配置放学时刻时的缺省值。
// 产品决策 2026-10-07：班级可配、默认 16:00；design-data 无出处，见 proposal 阻塞清单。
const DefaultPickupDeadline = "16:00"

// ResolvePickupDeadline 解析班级放学时刻；空串或非法值回退缺省。
func ResolvePickupDeadline(classDeadline string) string {
	d := strings.TrimSpace(classDeadline)
	if d == "" {
		return DefaultPickupDeadline
	}
	if _, err := parseHHMM(d); err != nil {
		return DefaultPickupDeadline
	}
	return d
}

// parseHHMM 解析 "HH:mm" 为当日时刻（UTC 仅作容器，比较只看时分）。
func parseHHMM(hhmm string) (time.Time, error) {
	t, err := time.Parse("15:04", hhmm)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid HH:mm %q: %w", hhmm, err)
	}
	return t, nil
}

// timeOfDay 把一个时刻的时分抽成可比较的 time.Time（同为 UTC 容器）。
func timeOfDay(t time.Time) time.Time {
	return time.Date(2000, 1, 1, t.Hour(), t.Minute(), 0, 0, time.UTC)
}

// IsConsecutiveSameDirection 判定本次登记是否与上一次同向（应拒绝）。
//
// 托管序列必须交替：送入 → 接走 → 送入 → 接走。
// last 为 nil 表示当日尚无记录，任何方向都可作为首次登记。
func IsConsecutiveSameDirection(last *domain.PickupRecord, next domain.PickupType) bool {
	if last == nil {
		return false
	}
	return last.Type == next
}

// NeedsRemarkForRepeatPickup 判定本次接走是否为重复（当日第 2+ 次），理由必填。
// priorPickupCount 为本次登记前当日已有的接走记录数。
func NeedsRemarkForRepeatPickup(priorPickupCount int64) bool {
	return priorPickupCount >= 1
}

// IsEarlyPickup 判定本次接走是否早于班级放学时刻。
// deadline 为 "HH:mm"；非法值按 DefaultPickupDeadline 处理。
func IsEarlyPickup(now time.Time, deadline string) bool {
	d := ResolvePickupDeadline(deadline)
	limit, err := parseHHMM(d)
	if err != nil {
		return false // 不可达：ResolvePickupDeadline 已回退
	}
	return timeOfDay(now).Before(timeOfDay(limit))
}

// NeedsRemarkForEarlyPickup 判定提前接走时理由是否必填。
func NeedsRemarkForEarlyPickup(now time.Time, deadline string) bool {
	return IsEarlyPickup(now, deadline)
}

// NeedsRemarkForPickup 汇总「本次接走是否必须填理由」及触发原因。
//
// 两个触发条件任一命中即必填：
//   - 重复接走（当日第 2+ 次）
//   - 提前接走（早于班级放学时刻）
//
// 迟接（超过放学时刻才接走）不触发本规则，归入滞留处置。
// 送入（dropoff）不受本规则约束，恒返回 false。
func NeedsRemarkForPickup(
	pickupType domain.PickupType,
	priorPickupCount int64,
	now time.Time,
	deadline string,
) (required bool, reason string) {
	// 送入不要理由（有意偏离 design-data/admin-05，见 proposal 分歧清单）
	if pickupType != domain.PickupTypePickUp {
		return false, ""
	}
	var reasons []string
	if NeedsRemarkForRepeatPickup(priorPickupCount) {
		reasons = append(reasons, "重复接走")
	}
	if NeedsRemarkForEarlyPickup(now, deadline) {
		reasons = append(reasons, "提前接走")
	}
	if len(reasons) == 0 {
		return false, ""
	}
	return true, strings.Join(reasons, "、")
}

// IsOverdueNotPicked 判定某生当前是否「超时未接」。
//
// 口径：当日最后一次托管转移是送入（仍在园），且当前时刻已过班级放学时刻。
// 按学生计，同一学生当日只计一次，随后被接走也不清零。
func IsOverdueNotPicked(last *domain.PickupRecord, now time.Time, deadline string) bool {
	if last == nil {
		return false // 当日无记录 = 未到园，不属滞留
	}
	if last.Type != domain.PickupTypeDropOff {
		return false // 最后一次是接走 = 已离园
	}
	return !IsEarlyPickup(now, deadline) && isAfterDeadline(now, deadline)
}

// isAfterDeadline 当前时刻是否已过班级放学时刻（含恰好到达）。
func isAfterDeadline(now time.Time, deadline string) bool {
	d := ResolvePickupDeadline(deadline)
	limit, err := parseHHMM(d)
	if err != nil {
		return false
	}
	return !timeOfDay(now).Before(timeOfDay(limit))
}
