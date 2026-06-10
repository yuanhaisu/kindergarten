package handler

import (
	"strings"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/response"
)

func (h *Handler) SystemStatus(c *gin.Context) {
	initialized, err := h.systemSvc.IsInitialized(c.Request.Context())
	if err != nil {
		response.ErrInternal(c, "查询系统状态失败")
		return
	}
	response.OK(c, gin.H{"initialized": initialized})
}

func (h *Handler) SystemInit(c *gin.Context) {
	var req service.InitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}

	err := h.systemSvc.Initialize(c.Request.Context(), req)
	if err != nil {
		msg := err.Error()
		switch {
		case msg == "ALREADY_INITIALIZED":
			response.ErrBadRequest(c, 40900, "系统已初始化，请直接登录")
		case msg == "PHONE_EXISTS":
			response.ErrBadRequest(c, 40901, "该手机号已被注册")
		case strings.HasPrefix(msg, "INVALID_PHONE"):
			response.ErrValidation(c, "手机号格式不正确")
		case strings.HasPrefix(msg, "WEAK_PASSWORD"):
			response.ErrValidation(c, "密码须包含大写字母、小写字母和数字，且不少于 8 位")
		default:
			response.ErrInternal(c, "系统初始化失败")
		}
		return
	}

	response.Created(c, gin.H{"message": "系统初始化成功，请使用管理员账号登录"})
}
