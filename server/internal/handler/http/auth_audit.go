package handler

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/log"
)

// 会话建立审计（#160 L205）。
//
// 登录 / 家长绑定 / 老师扫码绑定 / 切换园所都会改变「谁以什么身份进入了哪个
// 园」，必须在 operation_logs 留下 app_id 与园所。写在这里而不是 service：
// 只有 HTTP 边界同时握有签发结果、请求 IP 与 UA。审计是 best-effort——写失败
// 只告警，绝不因此拒绝一次已经成功的登录或绑定。
//
// Detail 只允许 app_id / role 这类非敏感字段；openid、token、code 一律不落库
// （openid 已有 #172 的脱敏查看路径）。
func (h *Handler) recordAuthAudit(c *gin.Context, action, appID string, result *service.LoginResult) {
	if h == nil || h.oplogSvc == nil || result == nil || result.User == nil {
		return
	}
	kindergartenID, roles := sessionScopeOf(result.User.Roles)
	// 待绑定会话没有园所。operation_logs 按 kindergarten_id 检索，kg=0 的行
	// 在任何园所视角下都不可见；绑定成功时自会写一条有园所的记录。
	if kindergartenID == 0 {
		return
	}
	entry := &domain.OperationLog{
		KindergartenID: kindergartenID,
		UserID:         result.User.ID,
		Module:         "auth",
		Action:         action,
		TargetType:     "user",
		TargetID:       result.User.ID,
		Detail:         fmt.Sprintf(`{"app_id":%q,"role":%q}`, appID, strings.Join(roles, ",")),
		IP:             c.ClientIP(),
		UserAgent:      truncateAuditUA(c.GetHeader("User-Agent")),
	}
	if err := h.oplogSvc.Record(c.Request.Context(), entry); err != nil {
		log.L(c.Request.Context()).Warn("write auth audit failed",
			"action", action, "app_id", appID, "user_id", result.User.ID, "err", err)
	}
}

// recordPasswordChangeAudit 记录自助改密（#182）。
//
// 改密会推进账号级 epoch 并让全部旧会话失效，属于安全敏感动作，必须可追溯。
// Detail 不写任何口令、哈希或 token；写失败只告警，不改变「已改密」的事实。
func (h *Handler) recordPasswordChangeAudit(c *gin.Context, userID uint64) {
	if h == nil || h.oplogSvc == nil || userID == 0 {
		return
	}
	kindergartenID := middleware.GetKindergartenID(c)
	if kindergartenID == 0 {
		return
	}
	entry := &domain.OperationLog{
		KindergartenID: kindergartenID,
		UserID:         userID,
		Module:         "auth",
		Action:         "change_password",
		TargetType:     "user",
		TargetID:       userID,
		Detail:         "{}",
		IP:             c.ClientIP(),
		UserAgent:      truncateAuditUA(c.GetHeader("User-Agent")),
	}
	if err := h.oplogSvc.Record(c.Request.Context(), entry); err != nil {
		log.L(c.Request.Context()).Warn("write password change audit failed",
			"user_id", userID, "err", err)
	}
}

// sessionScopeOf 取签发结果里的会话园与角色名。按 #160 L131，登录/绑定/切换
// 返回的 Roles 已收窄到「当前客户端角色族 ∩ 会话园」，因此这里不再过滤；
// 若未来有人放宽 roles，审计会跟着变宽，属于可接受的可见性。
func sessionScopeOf(roles []service.RoleInfo) (uint64, []string) {
	var (
		kindergartenID uint64
		names          = make([]string, 0, len(roles))
	)
	for _, r := range roles {
		if r.KindergartenID != nil && *r.KindergartenID != 0 {
			kindergartenID = *r.KindergartenID
		}
		if name := strings.TrimSpace(r.Role); name != "" {
			names = append(names, name)
		}
	}
	return kindergartenID, names
}

// truncateAuditUA 与 operation_logs.user_agent 的 varchar(300) 对齐。
// 按 rune 截断，避免把一个多字节字符从中间切开写坏 utf8mb4。
func truncateAuditUA(ua string) string {
	ua = strings.TrimSpace(ua)
	if utf8.RuneCountInString(ua) <= 300 {
		return ua
	}
	return string([]rune(ua)[:300])
}
