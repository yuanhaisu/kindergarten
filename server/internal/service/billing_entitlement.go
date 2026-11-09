package service

import "time"

const (
	// graceDays 到期后的宽限期（ADR-0015）。
	graceDays = 15
	// trialDays 新开通园所的全功能试用期（proposal）。
	// 人工延长的上限（60 天）目前只写进 spec，没有延长端点，等需要时再引入常量。
	trialDays = 30
	// commissionEligibleDays 佣金到账满多少天后可结算（ADR-0016）。
	commissionEligibleDays = 30
)

var graceDuration = time.Duration(graceDays) * 24 * time.Hour

// EntitlementState 授权在某一时刻的展示状态。
// active 在期 / grace 宽限期 / expired 已降级。
func EntitlementState(startsAt, expiresAt, now time.Time) string {
	return entitlementState(startsAt, expiresAt, now, graceDuration)
}

func entitlementState(startsAt, expiresAt, now time.Time, grace time.Duration) string {
	if now.Before(startsAt) {
		return "expired"
	}
	if now.Before(expiresAt) {
		return "active"
	}
	if now.Before(expiresAt.Add(grace)) {
		return "grace"
	}
	return "expired"
}
