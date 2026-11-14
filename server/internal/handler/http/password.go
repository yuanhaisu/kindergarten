package handler

import (
	"strings"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/middleware"
	"kindergarten-server/pkg/response"
)

// ChangePassword 自助改密。园长首次登录后用它清掉 must_change_password。
func (h *Handler) ChangePassword(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.ErrUnauthorized(c, "请先登录")
		return
	}
	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "请求参数不合法")
		return
	}
	if err := h.authSvc.ChangePassword(c.Request.Context(), userID, req.OldPassword, req.NewPassword); err != nil {
		switch {
		case err.Error() == "USER_NOT_FOUND":
			response.ErrNotFound(c, "账号不存在")
		case err.Error() == "PASSWORD_INCORRECT":
			response.ErrValidation(c, "原密码不正确")
		case strings.HasPrefix(err.Error(), "WEAK_PASSWORD:"):
			response.ErrValidation(c, "新密码强度不足")
		default:
			response.ErrInternal(c, "修改密码失败")
		}
		return
	}
	response.OK(c, gin.H{"must_change_password": false})
}
