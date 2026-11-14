package handler

import (
	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/middleware"
	"kindergarten-server/pkg/response"
)

// AdminGetSubscription 园所端「我的套餐」：只读，不暴露订单金额与分销信息。
func (h *Handler) AdminGetSubscription(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	view, err := h.entSvc.Subscription(c.Request.Context(), kgID)
	if err != nil {
		response.ErrInternal(c, "获取套餐信息失败")
		return
	}
	response.OK(c, view)
}

// CommonGetEntitlements 两端小程序启动时裁剪入口。
func (h *Handler) CommonGetEntitlements(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	modules, expiresAt, err := h.entSvc.EntitlementModules(c.Request.Context(), kgID)
	if err != nil {
		response.ErrInternal(c, "获取功能授权失败")
		return
	}
	response.OK(c, gin.H{"modules": modules, "expires_at": expiresAt})
}
