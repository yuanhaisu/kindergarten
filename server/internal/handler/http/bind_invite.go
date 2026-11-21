package handler

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/log"
	"kindergarten-server/pkg/response"
)

type bindInviteReq struct {
	InviteCode string `json:"invite_code" binding:"required"`
}

// BindInvite 老师邀请绑定（#168）。
//
// 调用方必须持有「待绑定 token」：账号与客户端都从 token 取，请求体只带
// 邀请码。绑定成功后返回预创建教师账号的登录结果，客户端直接替换本地凭证。
func (h *Handler) BindInvite(c *gin.Context) {
	var req bindInviteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}

	claims := middleware.GetClaims(c)
	if claims == nil {
		response.ErrUnauthorized(c, "认证凭证无效")
		return
	}
	userID, err := strconv.ParseUint(strings.TrimSpace(claims.UserID), 10, 64)
	if err != nil || userID == 0 {
		response.ErrUnauthorized(c, "认证凭证无效")
		return
	}

	result, err := h.bindInviteSvc.Bind(c.Request.Context(), service.BindInviteRequest{
		UserID:     userID,
		AppID:      claims.AppID,
		InviteCode: req.InviteCode,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrClientUnavailable):
			log.L(c.Request.Context()).Warn("bind invite rejected: client unavailable", "app_id", claims.AppID)
			response.ErrForbidden(c, "小程序客户端未注册或已停用，请更新小程序后重试")
		case errors.Is(err, service.ErrBindInviteTeacherOnly):
			log.L(c.Request.Context()).Warn("bind invite rejected: not teacher client", "app_id", claims.AppID)
			response.ErrForbidden(c, "仅老师端可使用邀请码绑定")
		case errors.Is(err, service.ErrBindInviteChannelMismatch):
			log.L(c.Request.Context()).Warn("bind invite rejected: client scope mismatch",
				"app_id", claims.AppID, "user_id", userID)
			response.ErrForbidden(c, "当前小程序无法绑定该园所，请联系园所管理员")
		case errors.Is(err, service.ErrBindInviteInvalid):
			response.ErrBadRequest(c, 40020, "邀请码无效或已过期，请向园所重新索取")
		case errors.Is(err, service.ErrBindInviteUsed):
			response.ErrBadRequest(c, 40021, "邀请码已被使用，请联系园所管理员")
		case errors.Is(err, service.ErrBindInviteAlreadyBound):
			log.L(c.Request.Context()).Warn("bind invite rejected: teacher account already bound",
				"app_id", claims.AppID, "user_id", userID)
			response.ErrConflict(c, "该教师账号已绑定其他微信，请联系园所管理员")
		default:
			log.L(c.Request.Context()).Error("bind invite failed",
				"app_id", claims.AppID, "user_id", userID, "err", err)
			response.ErrInternal(c, "绑定失败，请稍后重试")
		}
		return
	}

	response.OK(c, result)
}
