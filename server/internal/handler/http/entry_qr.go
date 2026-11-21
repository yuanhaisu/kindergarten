package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/response"
)

// GetParentEntryQRCode 生成当前园所的家长端入口二维码（#176）。
//
// 码属于园所对外物料，按当前登录园所生成；env_version 由管理端选择
// （测试用 trial/develop，线上用 release），空值回落到部署配置。
func (h *Handler) GetParentEntryQRCode(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}

	data, err := h.parentEntryQRSvc.GenerateParentEntryQRCode(c.Request.Context(), kgID, c.Query("env_version"))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEntryKindergartenNotFound):
			response.ErrNotFound(c, "园所不存在")
		case errors.Is(err, auth.ErrInvalidCodeEnvVersion):
			response.ErrValidation(c, "env_version 只接受 release / trial / develop")
		case errors.Is(err, service.ErrEntryChannelMismatch):
			response.ErrConflict(c, "当前园所的通道与家长端小程序不匹配，无法生成二维码")
		case errors.Is(err, service.ErrEntryClientUnavailable):
			response.ErrUnavailable(c, "家长端小程序未就绪，请稍后重试或联系平台")
		default:
			response.ErrInternal(c, "生成园所二维码失败")
		}
		return
	}
	response.OK(c, data)
}
