package handler

import (
	"github.com/gin-gonic/gin"

	"kindergarten-server/pkg/response"
)

// PlatformDashboard 平台运营概览：6 个指标 + 2 个待办，服务端一次聚合。
func (h *Handler) PlatformDashboard(c *gin.Context) {
	data, err := h.platformSvc.Dashboard(c.Request.Context())
	if err != nil {
		response.ErrInternal(c, "获取运营看板失败")
		return
	}
	response.OK(c, data)
}
