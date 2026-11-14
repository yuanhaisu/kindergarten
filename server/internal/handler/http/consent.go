package handler

import (
	"errors"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/response"
)

type consentReq struct {
	PolicyVersion string `json:"policy_version" binding:"required"`
}

// GetConsent 返回当前政策版本与本人同意状态。未同意时也返回 200，
// 客户端据此决定是否进入阻断页。
func (h *Handler) GetConsent(c *gin.Context) {
	uid := middleware.GetUserID(c)
	if uid == 0 {
		response.ErrUnauthorized(c, "缺少认证凭证")
		return
	}
	status, err := h.authSvc.ConsentStatus(c.Request.Context(), uid)
	if err != nil {
		response.ErrInternal(c, "读取同意状态失败")
		return
	}
	response.OK(c, status)
}

// PostConsent 记录同意。source 优先取 terminal_from 请求头，并校验该端在令牌角色中；
// 旧客户端不带该头时回退为按令牌角色推导。
func (h *Handler) PostConsent(c *gin.Context) {
	var req consentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	uid := middleware.GetUserID(c)
	if uid == 0 {
		response.ErrUnauthorized(c, "缺少认证凭证")
		return
	}

	source, ok := consentSourceFromRequest(c)
	if !ok {
		response.ErrBadRequest(c, 40031, "无法识别账号来源端")
		return
	}
	status, err := h.authSvc.RecordConsent(
		c.Request.Context(),
		uid,
		req.PolicyVersion,
		source,
		c.ClientIP(),
		c.Request.UserAgent(),
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrConsentVersionMismatch):
			response.ErrBadRequest(c, 40030, "隐私政策已更新，请刷新后重试")
		case errors.Is(err, service.ErrConsentSourceInvalid):
			response.ErrBadRequest(c, 40031, "无法识别账号来源端")
		default:
			response.ErrInternal(c, "记录同意失败")
		}
		return
	}
	response.OK(c, status)
}

// consentSourceFromRequest 返回 (来源端, 是否合法)。
//   - 带 terminal_from：必须是已知端，且出现在令牌角色里，否则拒绝（防止自称）。
//   - 不带：旧客户端兼容，回退为角色推导；推导为空时交给 service 返回 40031。
func consentSourceFromRequest(c *gin.Context) (string, bool) {
	roles := middleware.GetRoles(c)
	declared := strings.ToLower(strings.TrimSpace(c.GetHeader("terminal_from")))
	if declared == "" {
		return consentSourceFromRoles(roles), true
	}
	switch declared {
	case domain.ConsentSourceAdmin, domain.ConsentSourceTeacher, domain.ConsentSourceParent:
	default:
		return "", false
	}
	if slices.Contains(roles, declared) {
		return declared, true
	}
	return "", false
}

func consentSourceFromRoles(roles []string) string {
	for _, role := range roles {
		switch role {
		case domain.ConsentSourceAdmin, domain.ConsentSourceTeacher, domain.ConsentSourceParent:
			return role
		}
	}
	return ""
}
